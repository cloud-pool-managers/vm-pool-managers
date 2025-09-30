package utils

import "github.com/gophercloud/gophercloud/v2/openstack/compute/v2/servers"

func NoVolAttached(server servers.Server) bool {
	return len(server.AttachedVolumes) == 0
}
