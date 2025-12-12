package utils

import (
	"PoolManagerVM/backend/models"
	"context"
	"fmt"

	"github.com/gophercloud/gophercloud/v2/openstack/compute/v2/servers"
)

func GetAllServers() ([]servers.Server, error) {

	pages, err := servers.List(models.ComputeClient, servers.ListOpts{}).
		AllPages(context.Background())
	if err != nil {
		return nil, fmt.Errorf("failed to list servers: %w", err)
	}

	allServers, err := servers.ExtractServers(pages)
	if err != nil {
		return nil, fmt.Errorf("failed to extract servers: %w", err)
	}

	return allServers, nil
}
