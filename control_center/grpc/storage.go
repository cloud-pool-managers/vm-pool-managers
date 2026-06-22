package grpc

import (
	"context"
	"net/http"
	"os"
	"strconv"
	"strings"

	"control_center/config"
	"control_center/models"

	"github.com/danielgtaylor/huma/v2"
)

// storageQuotaGB : quota de stockage alloué par groupe (utilisateur/pool), en Go.
// Configurable via STORAGE_QUOTA_GB ; 0 = pas de quota (alerte jamais déclenchée).
func storageQuotaGB() int {
	if v, err := strconv.Atoi(strings.TrimSpace(os.Getenv("STORAGE_QUOTA_GB"))); err == nil && v >= 0 {
		return v
	}
	return 200
}

// registerStorageHuma enregistre GET /api/storage — stockage ALLOUÉ (disque des
// flavors × VMs) par utilisateur ou pool, avec quota et alerte de dépassement.
// Staff uniquement ; un non-admin ne voit que ses propres pools.
func registerStorageHuma(api huma.API) {
	huma.Register(api, huma.Operation{
		OperationID: "get-storage", Method: http.MethodGet, Path: "/api/storage",
		Summary: "Stockage alloué et quotas", Tags: []string{"usage"},
	}, func(ctx context.Context, in *struct {
		By string `query:"by"`
	}) (*AnyOutput, error) {
		by := in.By
		if by != "pool" {
			by = "user"
		}

		// Flavors : id -> disque (Go).
		var flavors []models.Flavor
		config.Database.Find(&flavors)
		diskByFlavor := map[string]int{}
		for _, f := range flavors {
			diskByFlavor[f.ID] = f.Disk
		}

		id, _ := identityFrom(ctx)
		q := config.Database.Model(&models.Server{})
		if id.Role != RoleAdmin {
			q = q.Where("user_id = ?", id.Email)
		}
		var servers []models.Server
		if err := q.Find(&servers).Error; err != nil {
			return nil, huma.Error500InternalServerError("lecture des serveurs échouée")
		}

		return &AnyOutput{Body: buildStorageReport(servers, diskByFlavor, by)}, nil
	})
}

// buildStorageReport agrège le stockage alloué par groupe (forme JSON inchangée).
func buildStorageReport(servers []models.Server, diskByFlavor map[string]int, by string) map[string]any {
	type group struct {
		Key    string `json:"key"`
		VMs    int    `json:"vms"`
		DiskGB int    `json:"disk_gb"`
		Quota  int    `json:"quota_gb"`
		Over   bool   `json:"over_quota"`
	}
	quota := storageQuotaGB()
	groups := map[string]*group{}
	totalDisk, totalVMs := 0, 0
	for _, s := range servers {
		key := s.UserID
		if by == "pool" {
			key = s.ServerpoolID
		}
		if key == "" {
			key = "—"
		}
		g := groups[key]
		if g == nil {
			g = &group{Key: key, Quota: quota}
			groups[key] = g
		}
		g.VMs++
		g.DiskGB += diskByFlavor[s.FlavorRef]
		totalVMs++
		totalDisk += diskByFlavor[s.FlavorRef]
	}
	for _, g := range groups {
		g.Over = quota > 0 && g.DiskGB > quota
	}

	out := make([]*group, 0, len(groups))
	for _, g := range groups {
		out = append(out, g)
	}

	return map[string]any{
		"by":       by,
		"quota_gb": quota,
		"groups":   out,
		"totals":   map[string]int{"vms": totalVMs, "disk_gb": totalDisk},
	}
}
