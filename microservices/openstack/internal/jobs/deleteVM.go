package jobs

import (
	"PoolManagerVM/backend/models"
	"context"
	"fmt"

	"github.com/gophercloud/gophercloud/v2/openstack/compute/v2/servers"
)

func DeleteVM(instanceID string) error {

	// Supprime la VM
	err := servers.Delete(context.Background(),
		models.ComputeClient, instanceID).ExtractErr()
	if err != nil {
		return fmt.Errorf("failed to delete VM %s: %w", instanceID, err)
	}

	return nil
}
