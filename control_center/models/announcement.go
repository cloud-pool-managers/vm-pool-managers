package models

import "time"

// Announcement : message d'annonce affiché à tous les utilisateurs (bandeau).
// Une seule ligne (la plus récente fait foi).
type Announcement struct {
	ID        uint      `json:"id" gorm:"primaryKey"`
	Message   string    `json:"message"`
	Active    bool      `json:"active"`
	UpdatedAt time.Time `json:"updated_at"`
}
