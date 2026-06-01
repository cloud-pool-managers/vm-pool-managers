package models

import "time"

type GitHubSession struct {
	ID        string    `gorm:"primaryKey"`
	Login     string    `gorm:"not null"`
	SSHKeys   string    `gorm:"type:text"` // JSON array of key strings
	CreatedAt time.Time `gorm:"autoCreateTime"`
}

// GitHubOAuthState stores the CSRF state server-side (avoids cookie SameSite issues).
type GitHubOAuthState struct {
	State     string    `gorm:"primaryKey"`
	CreatedAt time.Time `gorm:"autoCreateTime"`
}
