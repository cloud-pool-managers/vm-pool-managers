package grpc

import (
	"encoding/json"
	"net/http"
	"strings"

	"control_center/config"
	"control_center/models"
)

// GET /api/announcement — annonce courante (public : visible par tous, même non connecté).
func handleAnnouncement(w http.ResponseWriter, r *http.Request) {
	var a models.Announcement
	config.Database.Order("id ASC").First(&a)
	writeJSONMoodle(w, http.StatusOK, map[string]any{
		"message": a.Message, "active": a.Active, "updated_at": a.UpdatedAt,
	})
}

// POST /api/admin/announcement {message, active} — définit l'annonce (admin uniquement).
func handleAdminAnnouncement(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSONMoodle(w, http.StatusMethodNotAllowed, map[string]string{"error": "POST requis"})
		return
	}
	var req struct {
		Message string `json:"message"`
		Active  bool   `json:"active"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONMoodle(w, http.StatusBadRequest, map[string]string{"error": "JSON invalide"})
		return
	}
	var a models.Announcement
	if err := config.Database.Order("id ASC").First(&a).Error; err != nil {
		a = models.Announcement{}
	}
	a.Message = strings.TrimSpace(req.Message)
	a.Active = req.Active && a.Message != ""
	if err := config.Database.Save(&a).Error; err != nil {
		writeJSONMoodle(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSONMoodle(w, http.StatusOK, map[string]any{"message": a.Message, "active": a.Active})
}
