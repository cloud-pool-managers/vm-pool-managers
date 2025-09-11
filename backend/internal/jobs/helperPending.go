package jobs

import (
	"PoolManagerVM/backend/config"
	"PoolManagerVM/backend/models"

	"gorm.io/gorm"
)

func IncrementPending(poolID, userID string, paramID int) error {
	return config.Database.Transaction(func(tx *gorm.DB) error {
		var pool models.Serverpool
		if err := tx.Where(&models.Serverpool{
			ServerpoolID: poolID,
			UserID:       userID,
		}).FirstOrCreate(&pool).Error; err != nil {
			return err
		}
		if paramID >= len(pool.Params) {
			// Créer un param par défaut si nécessaire
			newParam := models.Param{
				ServerpoolID: poolID,
				UserID:       userID,
				MinVM:        0,
				MaxVM:        0,
				PendingJobs:  1, // on l'incrémente directement
				ImageRef:     "",
				FlavorRef:    "",
				Networks:     models.JSONStringSlice{},
			}
			if err := tx.Create(&newParam).Error; err != nil {
				return err
			}
			return nil
		}
		param := pool.Params[paramID]
		param.PendingJobs++
		return tx.Model(&pool).Update("pending_jobs", param.PendingJobs).Error
	})
}

func DecrementPending(poolID, userID string, paramID int) error {
	return config.Database.Transaction(func(tx *gorm.DB) error {
		var pool models.Serverpool
		// Charger les Params associés
		if err := tx.Preload("Params").Where(&models.Serverpool{
			ServerpoolID: poolID,
			UserID:       userID,
		}).FirstOrCreate(&pool).Error; err != nil {
			return err
		}

		// Vérifier que le paramID est valide
		if paramID >= len(pool.Params) {
			// Rien à décrémenter
			return nil
		}

		param := pool.Params[paramID]
		if param.PendingJobs > 0 {
			param.PendingJobs--
			return tx.Model(&param).Update("pending_jobs", param.PendingJobs).Error
		}

		return nil
	})
}
