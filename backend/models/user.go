package models

// Modèle utilisateur
type User struct {
	ID       uint   `gorm:"primaryKey"`
	Name     string `json:"name"`
	Email    string `json:"email"`
	Password string `gorm:"not null" json:"-"`
}
