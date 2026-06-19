package grpc

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"control_center/config"
	"control_center/models"
)

// /api/pool/presets — presets de création de pool (staff, préfixe /api/pool/).
//
//	GET           → liste les presets de l'utilisateur
//	POST {…}      → enregistre un preset
//	DELETE ?id=N  → supprime un preset (le sien)
func handlePoolPresets(w http.ResponseWriter, r *http.Request) {
	id, _ := identityFrom(r.Context())
	owner := id.Email

	switch r.Method {
	case http.MethodGet:
		var presets []models.PoolPreset
		config.Database.Where("owner_email = ?", owner).Order("name").Find(&presets)
		writeJSONMoodle(w, http.StatusOK, map[string]any{"presets": presets})

	case http.MethodPost:
		var p models.PoolPreset
		if err := json.NewDecoder(r.Body).Decode(&p); err != nil || strings.TrimSpace(p.Name) == "" {
			writeJSONMoodle(w, http.StatusBadRequest, map[string]string{"error": "nom du preset requis"})
			return
		}
		preset := models.PoolPreset{
			OwnerEmail:  owner,
			Name:        strings.TrimSpace(p.Name),
			Image:       p.Image,
			Flavor:      p.Flavor,
			Network:     p.Network,
			Config:      p.Config,
			MinVM:       p.MinVM,
			MaxVM:       p.MaxVM,
			AppPort:     p.AppPort,
			OffDays:     p.OffDays,
			ComputeMode: p.ComputeMode,
		}
		// Upsert par (owner, name) : ré-enregistrer un même nom met à jour.
		var existing models.PoolPreset
		if config.Database.Where("owner_email = ? AND name = ?", owner, preset.Name).First(&existing).Error == nil {
			preset.ID = existing.ID
			preset.CreatedAt = existing.CreatedAt
			config.Database.Save(&preset)
		} else {
			config.Database.Create(&preset)
		}
		writeJSONMoodle(w, http.StatusOK, map[string]any{"ok": true, "preset": preset})

	case http.MethodDelete:
		pid, _ := strconv.Atoi(r.URL.Query().Get("id"))
		if pid <= 0 {
			writeJSONMoodle(w, http.StatusBadRequest, map[string]string{"error": "id requis"})
			return
		}
		config.Database.Where("id = ? AND owner_email = ?", pid, owner).Delete(&models.PoolPreset{})
		writeJSONMoodle(w, http.StatusOK, map[string]any{"ok": true})

	default:
		writeJSONMoodle(w, http.StatusMethodNotAllowed, map[string]string{"error": "méthode non autorisée"})
	}
}
