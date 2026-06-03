package models

import "time"

// ImageProposal is a teacher's request to add a new VM image, pointing to the
// GitHub repo of the image (repo2docker-style) to build it from.
type ImageProposal struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	GithubURL   string    `json:"github_url"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	SubmittedBy string    `json:"submitted_by"`
	Status      string    `gorm:"default:'pending'" json:"status"` // pending | approved | rejected
	CreatedAt   time.Time `json:"created_at"`
}
