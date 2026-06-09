package models

import "time"

// MoodleSession : session légère créée après un login Moodle réussi (analogue à GitHubSession).
// Sert au portail étudiant pour identifier l'utilisateur par email (pas d'auth gRPC/JWT).
type MoodleSession struct {
	ID           string `gorm:"primaryKey"`
	Email        string `gorm:"index"`
	FullName     string
	MoodleUserID int
	Role         string    // "admin" (site admin) ou "student"
	CreatedAt    time.Time `gorm:"autoCreateTime"`
}
