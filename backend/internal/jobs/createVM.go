package jobs

import (
	"PoolManagerVM/backend/models"
	"PoolManagerVM/backend/utils"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/google/uuid"
	"github.com/gophercloud/gophercloud/openstack/compute/v2/servers"
	"github.com/gophercloud/gophercloud/v2/openstack/compute/v2/keypairs"
	"github.com/gophercloud/utils/openstack/clientconfig"
)

func CreateVM(workerID int, job models.Job) error {

	metadata := map[string]string{}
	if metaStr, ok := job.Data["Metadata"]; ok && metaStr != "" {
		if err := json.Unmarshal([]byte(metaStr), &metadata); err != nil {
			log.Println("Error unmarshall metadata: ", err)
		}
	}
	metadata["user_id"] = job.Data["user_id"]
	metadata["serverpool_id"] = job.Data["serverpool_id"]
	metadata["min_vm"] = job.Data["min_vm"]
	metadata["max_vm"] = job.Data["max_vm"]

	var networks models.JSONStringSlice
	if err := networks.Scan(job.Data["networks"]); err != nil {
		log.Println("Failed to parse networks:", err)
		networks = models.JSONStringSlice{} // fallback
	}

	paramID := utils.ParseInt(job.Data["ID"])
	fmt.Println("Worker ", workerID, " takes the job of creating a VM")

	serv := models.Server{
		Name:         job.Data["name"],
		FlavorRef:    job.Data["flavor_ref"],
		ImageRef:     job.Data["image_ref"],
		UserID:       job.Data["user_id"],
		ServerpoolID: job.Data["serverpool_id"],
		Metadata:     metadata,
		Networks:     networks,
	}

	opts := &clientconfig.ClientOpts{
		Cloud: os.Getenv("OPTS_CLOUD"),
	}

	client, err := clientconfig.NewServiceClient("compute", opts)
	if err != nil {
		return fmt.Errorf("failed to create compute client: %w", err)
	}

	createOpts := servers.CreateOpts{
		Name:      fmt.Sprintf(`%s-%s`, serv.ServerpoolID, uuid.New().String()),
		FlavorRef: serv.FlavorRef,
		ImageRef:  serv.ImageRef,
		Metadata:  serv.Metadata,
		Networks:  []servers.Network{{UUID: os.Getenv("NETWORK_ID")}},
	}

	createOptsExt := keypairs.CreateOptsExt{
		CreateOptsBuilder: createOpts,
		KeyName:           os.Getenv("API_KEYNAME"),
	}

	server, err := servers.Create(client, createOptsExt).Extract()
	if err != nil {
		log.Println("failed to create VM:", err)
		return fmt.Errorf("failed to create VM: %w", err)
	}

	DecrementPending(uint(paramID))
	log.Println("[VM] Creating server ID=", server.ID, " , Name=", server.Name)

	for {
		current, err := servers.Get(client, server.ID).Extract()
		if err != nil {
			return fmt.Errorf("failed to get server status: %w", err)
		}

		if current.Status == "ACTIVE" {
			log.Printf("[VM] Server %s is ACTIVE\n", current.ID)
			break
		}

		if current.Status == "ERROR" {
			return fmt.Errorf("server %s failed to boot (ERROR state)", current.ID)
		}

		log.Printf("[VM] Waiting for server %s (status=%s)\n", current.ID, current.Status)
		time.Sleep(3 * time.Second)
	}

	fmt.Println("Worker ", workerID, " finished its job")

	return nil
}
