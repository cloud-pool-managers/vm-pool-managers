package config

import (
	"fmt"
	"log"
	"strings"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"PoolManagerVM/backend/models"
	"PoolManagerVM/backend/utils"
)

var Database *gorm.DB

func Sync_DB() {
	var err error
	Database, err = gorm.Open(sqlite.Open("PoolManagerVM.db"), &gorm.Config{})
	if err != nil {
		panic("failed to connect database")
	}

	Database.AutoMigrate(&models.User{})
	Database.AutoMigrate(&models.Server{})
	Database.AutoMigrate(&models.ServerPool{})

	allserv, err := utils.GetAllServers()
	if err != nil {
		log.Fatalf("error connexion to openstack")
	}

	// debug
	// for _, s := range allserv {
	// 	data, _ := json.MarshalIndent(s, "", " ")
	// 	fmt.Println(string(data))
	// 	fmt.Println("-------------------------------------------------------")
	// }

	for _, s := range allserv {
		pool := models.ServerPool{}

		poolID, hasPool := s.Metadata["serverpool-id"]
		userID, hasUser := s.Metadata["userID"]

		if hasPool && hasUser {
			pool = models.ServerPool{
				ServerpoolID: poolID,
				UserID:       userID,
				MinVM:        utils.ParseInt(s.Metadata["minVM"]),
				MaxVM:        utils.ParseInt(s.Metadata["maxVM"]),
				PendingJobs:  0,
			}

			Database.Clauses(clause.OnConflict{
				Columns:   []clause.Column{{Name: "serverpool_id"}, {Name: "user_id"}},
				DoUpdates: clause.AssignmentColumns([]string{"min_vm", "max_vm", "pending_jobs"}),
			}).FirstOrCreate(&pool)
		}

		server := models.Server{
			ID:       s.ID,
			Name:     s.Name,
			Status:   s.Status,
			FlavorID: fmt.Sprintf("%v", s.Flavor["id"]),
			ImageID:  fmt.Sprintf("%v", s.Image["id"]),
		}

		if pool.ID != 0 {
			server.PoolID = &pool.ID
		}
		Database.Save(&server)
	}
}

func Resync_DB() {
	allServ, err := utils.GetAllServers()
	if err != nil {
		log.Println("Error fetching data from OpenStack:", err)
		return
	}

	existingServerIDs := make(map[string]struct{})

	for _, s := range allServ {
		poolID := strings.TrimSpace(s.Metadata["serverpool-id"])
		userID := strings.TrimSpace(s.Metadata["userID"])
		if poolID == "" || userID == "" {
			log.Println("Skipping server ", s.ID, ": missing poolID or userID")
			continue
		}
		if err := Database.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "serverpool-id"}, {Name: "user_id"}},
			DoUpdates: clause.AssignmentColumns([]string{"min_vm", "max_vm"}),
		}).Create(&models.ServerPool{
			ServerpoolID: poolID,
			UserID:       userID,
			MinVM:        utils.ParseInt(s.Metadata["minVM"]),
			MaxVM:        utils.ParseInt(s.Metadata["maxVM"]),
			PendingJobs:  0,
		}).Error; err != nil {
			log.Println("Error create/update pool:", err)
			continue
		}

		server := models.Server{
			ID:       s.ID,
			Name:     s.Name,
			Status:   s.Status,
			FlavorID: s.Flavor["id"].(string),
			ImageID:  s.Image["id"].(string),
		}

		var linkedPool models.ServerPool
		if err := Database.Where("serverpool_id = ? AND user_id = ?", poolID, userID).First(&linkedPool).Error; err != nil {
			server.PoolID = &linkedPool.ID
		}

		if err := Database.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "id"}},
			DoUpdates: clause.AssignmentColumns([]string{"name", "status", "flavor_id", "image_id", "pool_id"}),
		}).Create(&server).Error; err != nil {
			log.Println("Error create/update server:", err)
			continue
		}

		existingServerIDs[s.ID] = struct{}{}
	}
	var dbServers []models.Server
	if err := Database.Find(&dbServers).Error; err != nil {
		log.Println("Error fetching server DB:", err)
		return
	}

	for _, s := range dbServers {
		if _, ok := existingServerIDs[s.ID]; !ok {
			log.Println("Server ", s.ID, " not in Openstack, delete")
			Database.Delete(&s)
		}
	}

}
