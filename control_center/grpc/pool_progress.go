package grpc

import (
	"net/http"
	"time"

	"control_center/config"
	"control_center/models"
)

// ProgressRow : état d'un étudiant inscrit dans un pool (vue progression live).
type ProgressRow struct {
	Name       string     `json:"name"`
	Email      string     `json:"email,omitempty"`
	HasVM      bool       `json:"has_vm"`
	IP         string     `json:"ip,omitempty"`
	PowerState string     `json:"power_state,omitempty"` // ACTIVE | SHUTOFF | SUSPENDED…
	Activity   string     `json:"activity,omitempty"`    // active | connected | idle | suspended
	Healthy    bool       `json:"healthy"`
	LastActive *time.Time `json:"last_active,omitempty"`
}

// GET /api/pool/progress?pool_id=&user_id= — tableau de bord de progression par étudiant
// inscrit (A1) : qui a lancé sa VM, qui est actif (notebook/kernel ou SSH), dernière activité.
// Staff uniquement (préfixe /api/pool/). Inclut les inscrits SANS VM (« pas encore lancé »).
func handlePoolProgress(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSONMoodle(w, http.StatusMethodNotAllowed, map[string]string{"error": "GET requis"})
		return
	}
	poolID := r.URL.Query().Get("pool_id")
	if poolID == "" {
		writeJSONMoodle(w, http.StatusBadRequest, map[string]string{"error": "pool_id requis"})
		return
	}
	// Un non-admin ne voit que SES pools.
	userID := effectiveEmail(r, r.URL.Query().Get("user_id"))

	// Roster : étudiants inscrits dans le pool.
	var pool models.Serverpool
	if err := config.Database.
		Preload("ListStudents.Students").
		Where("serverpool_id = ? AND user_id = ?", poolID, userID).
		First(&pool).Error; err != nil {
		writeJSONMoodle(w, http.StatusNotFound, map[string]string{"error": "pool introuvable"})
		return
	}
	roster := pool.ListStudents.Students

	// État live des VMs du pool (réutilise l'inventaire : power state + activité Jupyter/SSH).
	vmByStudent := map[string]InventoryVM{}
	if inv, err := buildInventory(); err == nil {
		for _, p := range inv {
			if p.PoolID != poolID || p.UserID != userID {
				continue
			}
			for _, vm := range p.VMs {
				if vm.Student != "" {
					vmByStudent[vm.Student] = vm
				}
			}
		}
	}

	rows := make([]ProgressRow, 0, len(roster))
	launched, activeNow := 0, 0
	for _, s := range roster {
		row := ProgressRow{Name: s.Name, Email: s.MoodleEmail}
		if vm, ok := vmByStudent[s.Name]; ok {
			row.HasVM = true
			row.IP = vm.IP
			row.PowerState = vm.PowerState
			row.Activity = vm.ActivityStatus
			row.Healthy = vm.Healthy
			if !vm.LastActive.IsZero() {
				la := vm.LastActive
				row.LastActive = &la
			}
			launched++
			if vm.ActivityStatus == "active" || vm.ActivityStatus == "connected" {
				activeNow++
			}
		}
		rows = append(rows, row)
	}

	writeJSONMoodle(w, http.StatusOK, map[string]any{
		"pool_id":  poolID,
		"enrolled": len(roster),
		"launched": launched,
		"active":   activeNow,
		"rows":     rows,
	})
}
