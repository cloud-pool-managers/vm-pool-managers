package grpc

import (
	"context"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"control_center/config"
	"control_center/models"

	"github.com/danielgtaylor/huma/v2"
)

// Tarification (configurable par .env, valeurs par défaut indicatives).
func envFloat(key string, def float64) float64 {
	if v, err := strconv.ParseFloat(strings.TrimSpace(os.Getenv(key)), 64); err == nil && v >= 0 {
		return v
	}
	return def
}

func priceVCPUHour() float64 { return envFloat("PRICE_VCPU_HOUR", 0.012) }
func priceGBHour() float64   { return envFloat("PRICE_GB_HOUR", 0.006) }
func priceCurrency() string {
	if c := strings.TrimSpace(os.Getenv("PRICE_CURRENCY")); c != "" {
		return c
	}
	return "€"
}

// vmCost calcule le coût d'une consommation à partir des secondes et du flavor.
func vmCost(seconds int64, vcpus, ramMB int) (vmHours, vcpuHours, gbHours, cost float64) {
	vmHours = float64(seconds) / 3600
	vcpuHours = vmHours * float64(vcpus)
	gbHours = vmHours * (float64(ramMB) / 1024)
	cost = vcpuHours*priceVCPUHour() + gbHours*priceGBHour()
	return
}

// GET /api/pricing est servi par HUMA (registerHumaRoutes dans huma.go).

// registerUsageHuma enregistre GET /api/usage (consommation & coûts, F1/F4).
func registerUsageHuma(api huma.API) {
	huma.Register(api, huma.Operation{
		OperationID: "get-usage", Method: http.MethodGet, Path: "/api/usage",
		Summary: "Consommation et coût du mois", Tags: []string{"usage"},
	}, func(ctx context.Context, in *struct {
		Month string `query:"month"`
		By    string `query:"by"`
	}) (*AnyOutput, error) {
		month := in.Month
		if month == "" {
			month = time.Now().UTC().Format("2006-01")
		}
		by := in.By
		if by != "pool" {
			by = "user"
		}

		q := config.Database.Where("year_month = ?", month)
		id, _ := identityFrom(ctx)
		if id.Role != RoleAdmin {
			q = q.Where("user_id = ?", id.Email) // un prof ne voit que ses pools
		}

		var rows []models.VMUsage
		if err := q.Find(&rows).Error; err != nil {
			return nil, huma.Error500InternalServerError("lecture de la consommation échouée")
		}

		return &AnyOutput{Body: buildUsageReport(rows, month, by)}, nil
	})
}

// buildUsageReport agrège les lignes de consommation par utilisateur/pool (forme JSON inchangée).
func buildUsageReport(rows []models.VMUsage, month, by string) map[string]any {
	type group struct {
		Key       string  `json:"key"`
		VMHours   float64 `json:"vm_hours"`
		VCPUHours float64 `json:"vcpu_hours"`
		GBHours   float64 `json:"gb_hours"`
		Cost      float64 `json:"cost"`
	}
	groups := map[string]*group{}
	var totVM, totVCPU, totGB, totCost float64
	for _, u := range rows {
		key := u.UserID
		if by == "pool" {
			key = u.PoolID
		}
		if key == "" {
			key = "—"
		}
		g := groups[key]
		if g == nil {
			g = &group{Key: key}
			groups[key] = g
		}
		vmH, vcpuH, gbH, cost := vmCost(u.Seconds, u.VCPUs, u.RAMMB)
		g.VMHours += vmH
		g.VCPUHours += vcpuH
		g.GBHours += gbH
		g.Cost += cost
		totVM += vmH
		totVCPU += vcpuH
		totGB += gbH
		totCost += cost
	}

	out := make([]*group, 0, len(groups))
	for _, g := range groups {
		out = append(out, g)
	}

	return map[string]any{
		"month":    month,
		"by":       by,
		"currency": priceCurrency(),
		"groups":   out,
		"totals": map[string]float64{
			"vm_hours": totVM, "vcpu_hours": totVCPU, "gb_hours": totGB, "cost": totCost,
		},
	}
}
