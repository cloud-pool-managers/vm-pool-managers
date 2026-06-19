package models

import "time"

// VMUsage accumule le temps d'activité d'une VM, par mois, pour la comptabilisation
// de consommation et de coût (heures-VM, vCPU·h, Go·h). Une ligne par (mois, VM).
// Alimenté par un échantillonnage périodique des VMs ACTIVE (internal/monitoring).
type VMUsage struct {
	ID          uint      `json:"id" gorm:"primaryKey"`
	YearMonth   string    `json:"year_month" gorm:"index:idx_usage_month_vm,unique;index"` // "2026-06"
	VMID        string    `json:"vm_id" gorm:"index:idx_usage_month_vm,unique"`
	PoolID      string    `json:"pool_id" gorm:"index"`
	UserID      string    `json:"user_id" gorm:"index"`
	Flavor      string    `json:"flavor"`
	VCPUs       int       `json:"vcpus"`
	RAMMB       int       `json:"ram_mb"`
	Seconds     int64     `json:"seconds"` // secondes d'activité cumulées dans le mois
	LastSampled time.Time `json:"last_sampled"`
}
