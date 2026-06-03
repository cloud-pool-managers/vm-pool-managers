package grpc

import (
	"control_center/config"
	"control_center/models"
	"encoding/json"
	"net/http"
	"strings"
)

// handleImageProposals stores and lists teacher image proposals.
//
//	POST /api/image-proposals   {github_url, name, description, submitted_by}
//	GET  /api/image-proposals?user=<email>   → that teacher's proposals (newest first)
func handleImageProposals(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		var body struct {
			GithubURL   string `json:"github_url"`
			Name        string `json:"name"`
			Description string `json:"description"`
			SubmittedBy string `json:"submitted_by"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			http.Error(w, "invalid JSON", http.StatusBadRequest)
			return
		}
		body.GithubURL = strings.TrimSpace(body.GithubURL)
		body.Name = strings.TrimSpace(body.Name)
		if body.GithubURL == "" || body.Name == "" {
			http.Error(w, "github_url et name sont requis", http.StatusBadRequest)
			return
		}
		if !strings.HasPrefix(body.GithubURL, "https://github.com/") &&
			!strings.HasPrefix(body.GithubURL, "http://github.com/") {
			http.Error(w, "github_url doit être une URL github.com", http.StatusBadRequest)
			return
		}

		p := models.ImageProposal{
			GithubURL:   body.GithubURL,
			Name:        body.Name,
			Description: strings.TrimSpace(body.Description),
			SubmittedBy: strings.TrimSpace(body.SubmittedBy),
			Status:      "pending",
		}
		if err := config.Database.Create(&p).Error; err != nil {
			http.Error(w, "db error: "+err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(p)

	case http.MethodGet:
		var list []models.ImageProposal
		q := config.Database.Order("created_at DESC")
		if u := r.URL.Query().Get("user"); u != "" {
			q = q.Where("submitted_by = ?", u)
		}
		q.Find(&list)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(list)

	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}
