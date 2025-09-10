package internal

import (
	"PoolManagerVM/backend/config"
	"PoolManagerVM/backend/internal/worker"
	"PoolManagerVM/backend/models"
	"PoolManagerVM/backend/utils"
	"context"
	"log"
	"strconv"
	"time"

	"github.com/gophercloud/gophercloud/openstack/compute/v2/servers"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// Backwork is a background loop that monitors admin servers and ensures a minimum number of VMs are running.
// It fetches all servers, filters those owned by the admin, and compares the current count to a configured minimum.
// If there are too few, it adds jobs to create additional VMs. The loop repeats every 20 seconds.

func Backwork(ctx context.Context) {

	for {
		allServers, err := utils.GetAllServers()
		if err != nil {
			log.Printf("Error : %v", err)
			return
		}

		// get all admin VMs
		var myPool []servers.Server
		for _, s := range allServers {
			if s.Metadata["userID"] == "admin" {
				myPool = append(myPool, s)
			}
		}

		var minVM int
		if len(myPool) == 0 {
			// if no VMs, create some with config file
			cfg, err := utils.LoadConfig("config.toml")
			if err != nil {
				log.Printf("Error")
				return
			}
			minVM, err = strconv.Atoi(cfg.Metadata["minVM"])
			if err != nil {
				log.Printf("Error : %v", err)
			}
		} else {
			minVM, err = strconv.Atoi(myPool[0].Metadata["minVM"])
			if err != nil {
				log.Printf("Error : %v", err)
			}
		}

		cfg, err := utils.LoadConfig("config.toml")
		if err != nil {
			log.Printf("Error")
			return
		}

		// adding PendingJobs on current serverpool to not create duplicate
		err = config.Database.Transaction(func(tx *gorm.DB) error {
			// var pool models.ServerPool
			// if err := tx.Where("serverpool_id = ? AND user_id = ?", "PoolVMs", "admin").FirstOrCreate(&pool, models.ServerPool{
			// 	ServerpoolID: "PoolVms",
			// 	UserID:       "admin",
			// 	PendingJobs:  0,
			// 	MinVM:        utils.ParseInt(cfg.Metadata["minVM"]),
			// 	MaxVM:        utils.ParseInt(cfg.Metadata["maxVM"]),
			// }).Error; err != nil {
			// 	return err
			// }

			pool := models.ServerPool{
				ServerpoolID: "PoolVms",
				UserID:       "admin",
				PendingJobs:  0,
				MinVM:        utils.ParseInt(cfg.Metadata["minVM"]),
				MaxVM:        utils.ParseInt(cfg.Metadata["maxVM"]),
			}

			// Insert ou update si la combinaison serverpool_id + user_id existe déjà
			if err := tx.Clauses(clause.OnConflict{
				Columns:   []clause.Column{{Name: "serverpool_id"}, {Name: "user_id"}},
				DoUpdates: clause.AssignmentColumns([]string{"pending_jobs", "min_vm", "max_vm"}),
			}).Create(&pool).Error; err != nil {
				return err
			}

			current := len(myPool) + pool.PendingJobs
			if current < minVM {
				numToCreate := minVM - current
				for range numToCreate {
					worker.AddJob(*worker.CreateJob("base", worker.CreateVMAdmin, nil), false)
					pool.PendingJobs++
				}
				if err := tx.Model(&pool).Update("pending_jobs", pool.PendingJobs).Error; err != nil {
					return err
				}
			}
			return nil
		})

		if err != nil {
			log.Println("DB error: ", err)
		}

		select {
		case <-ctx.Done():
			log.Println("Backwork stopped")
			return
		case <-time.After(10 * time.Second):
			// next cycle
		}
	}
}
