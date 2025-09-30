package jobs

import (
	"PoolManagerVM/backend/models"
	"PoolManagerVM/backend/utils"
	"context"
	"log"
	"os"

	"github.com/gophercloud/gophercloud/v2"
	"github.com/gophercloud/gophercloud/v2/openstack"
	"github.com/gophercloud/gophercloud/v2/openstack/blockstorage/v3/volumes"
	"github.com/gophercloud/utils/v2/openstack/clientconfig"
)

func CreateVolumeAndAttach(workerID int, job models.Job) error {

	volumeOpts := volumes.CreateOpts{
		Size:        utils.ParseInt(job.Data["size"]),
		Description: job.Data["description"],
		Name:        job.Data["name"],
		VolumeType:  job.Data["volume_type"],
	}

	provider, err := clientconfig.AuthenticatedClient(context.Background(), &clientconfig.ClientOpts{
		Cloud: os.Getenv("OPTS_CLOUD"),
	})
	if err != nil {
		log.Println("Failed to authenticate:", err)
		return err
	}

	BlockStorageClient, err := openstack.NewBlockStorageV3(provider, gophercloud.EndpointOpts{
		Availability: gophercloud.AvailabilityPublic,
	})
	if err != nil {
		log.Println("Failed to create BlockStorageV3 client:", err)
		return err
	}

	volumeSchedulerHintOpts := volumes.SchedulerHintOpts{}

	newVolume, err := volumes.Create(context.Background(), BlockStorageClient, volumeOpts, volumeSchedulerHintOpts).Extract()
	if err != nil {
		log.Println("Failed to create volume:", err)
		log.Println(volumeOpts)
		return err
	}
	log.Println("Volume created with ID:", newVolume.ID)
	log.Println(newVolume)

	allServs, err := utils.GetAllServers()
	if err != nil {
		log.Println("Failed to get all servers:", err)
		return err
	}

	// if serv.id is in job.data
	// log.Println("Attaching volunme to server ", job.Data["server_id"])
	// attachOpts := volumes.AttachOpts{
	// 	InstanceUUID: job.Data["server_id"],
	// 	MountPoint:   "/dev/vdb",
	// 	Mode:         "rw",
	// }
	// err = volumes.Attach(context.Background(), client, newVolume.ID, attachOpts).ExtractErr()
	// if err != nil {
	// 	log.Println("Failed to attach volume:", err)
	// 	return err
	// }

	for _, serv := range allServs {
		if utils.NoVolAttached(serv) && serv.Status == "ACTIVE" {
			log.Printf("Attaching volume to server %s\n", serv.ID)
			attachOpts := volumes.AttachOpts{
				InstanceUUID: serv.ID,
				MountPoint:   "/dev/vdb",
				Mode:         "rw",
			}
			err = volumes.Attach(context.Background(), BlockStorageClient, newVolume.ID, attachOpts).ExtractErr()
			if err != nil {
				log.Println("Failed to attach volume:", err)
				return err
			}
			break
		}
	}

	log.Printf("Worker %d completed the job of creating and attaching a volume\n", workerID)

	return nil

}
