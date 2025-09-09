package internal

import (
	"PoolManagerVM/backend/internal/worker"
	"PoolManagerVM/backend/utils"
	"context"
	"log"
	"strconv"
	"time"

	"github.com/gophercloud/gophercloud/openstack/compute/v2/servers"
)

func Backwork(ctx context.Context) {
	for {
		allServers, err := GetAllServers()
		if err != nil {
			log.Printf("Error : %v", err)
			return
		}
		var myPool []servers.Server
		for _, s := range allServers {
			if s.Metadata["userID"] == "admin" {
				myPool = append(myPool, s)
			}
		}
		if len(myPool) == 0 {
			cfg, err := utils.LoadConfig("config.toml")
			if err != nil {
				log.Printf("Error")
				return
			}
			numVM, err := strconv.Atoi(cfg.Metadata["minVM"])
			if err != nil {
				log.Printf("Error : %v", err)
			}
			for range numVM {
				worker.AddJob(*worker.CreateJob("base", worker.CreateVMAdmin, nil), false)
			}
		} else {
			numVM, err := strconv.Atoi(myPool[0].Metadata["minVM"])
			if err != nil {
				log.Printf("Error : %v", err)
			}

			if len(myPool) < numVM {

				for i := 0; i < numVM-len(myPool); i++ {
					worker.AddJob(*worker.CreateJob("base", worker.CreateVMAdmin, nil), false)
				}
			}
		}
		select {
		case <-ctx.Done():
			log.Println("Backwork stopped")
			return
		case <-time.After(20 * time.Second):
			// next cycle
		}
	}
}
