package jobs

import (
	"PoolManagerVM/backend/models"
	"context"
	"fmt"

	"github.com/gophercloud/gophercloud/v2/openstack/compute/v2/servers"
)

// StopVM powers off a VM (off-days) without deleting it — the disk/state is kept.
func StopVM(instanceID string) error {
	if err := servers.Stop(context.Background(), models.ComputeClient, instanceID).ExtractErr(); err != nil {
		return fmt.Errorf("failed to stop VM %s: %w", instanceID, err)
	}
	return nil
}

// StartVM powers a previously stopped VM back on.
func StartVM(instanceID string) error {
	if err := servers.Start(context.Background(), models.ComputeClient, instanceID).ExtractErr(); err != nil {
		return fmt.Errorf("failed to start VM %s: %w", instanceID, err)
	}
	return nil
}

// SuspendVM suspend une VM (hibernation) : l'état mémoire est préservé sur disque,
// les vCPU/RAM sont libérés. Reprise rapide via ResumeVM.
func SuspendVM(instanceID string) error {
	if err := servers.Suspend(context.Background(), models.ComputeClient, instanceID).ExtractErr(); err != nil {
		return fmt.Errorf("failed to suspend VM %s: %w", instanceID, err)
	}
	return nil
}

// ResumeVM reprend une VM suspendue.
func ResumeVM(instanceID string) error {
	if err := servers.Resume(context.Background(), models.ComputeClient, instanceID).ExtractErr(); err != nil {
		return fmt.Errorf("failed to resume VM %s: %w", instanceID, err)
	}
	return nil
}
