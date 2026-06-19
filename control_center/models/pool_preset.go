package models

import "time"

// PoolPreset : configuration de création de pool sauvegardée par un enseignant,
// réapplicable plus tard (image, flavor, réseau, config, port, jours off, mode calcul).
// Ne stocke pas le nom/min/max du pool (propres à chaque instanciation).
type PoolPreset struct {
	ID          uint      `json:"id" gorm:"primaryKey"`
	OwnerEmail  string    `json:"owner_email" gorm:"index"`
	Name        string    `json:"name"`
	Image       string    `json:"image"`
	Flavor      string    `json:"flavor"`
	Network     string    `json:"network"`
	Config      string    `json:"config"`
	AppPort     int       `json:"app_port"`
	OffDays     string    `json:"off_days"`
	ComputeMode bool      `json:"compute_mode"`
	CreatedAt   time.Time `json:"created_at"`
}
