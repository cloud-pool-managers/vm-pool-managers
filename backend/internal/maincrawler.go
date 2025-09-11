package internal

import (
	"PoolManagerVM/backend/utils"
	"context"
	"fmt"
	"log"
	"time"

	"github.com/gophercloud/gophercloud/openstack/compute/v2/servers"
)

func Monitor(c context.Context) {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-c.Done():
			log.Println("Monitoring stopped")
			return

		case <-ticker.C:
			log.Println("Checking serverpools...")
			allServers, err := utils.GetAllServers()
			if err != nil {
				log.Fatalf("Error : %v", err)
				return
			}
			CheckAndCreate(allServers)
		}
	}
}

// works only if one type of server in pool
func CheckAndCreate(allServers []servers.Server) {
	serverPools := map[string][]servers.Server{}
	minVM := map[string]int{}
	maxVM := map[string]int{}

	for _, s := range allServers {
		poolID := s.Metadata["serverpool"]
		serverPools[poolID] = append(serverPools[poolID], s)
		minVM[poolID] = utils.ParseInt(s.Metadata["minVM"])
		maxVM[poolID] = utils.ParseInt(s.Metadata["maxVM"])
	}

	for poolID, serversInPool := range serverPools {
		active := len(serversInPool)
		missing := minVM[poolID] - active
		if missing > 0 {
			fmt.Printf("Serverpool %s: missing %d VM(s)\n", poolID, missing)
			for range missing {
				// go createVM(poolID, active, missing)
			}
		}
	}
}
