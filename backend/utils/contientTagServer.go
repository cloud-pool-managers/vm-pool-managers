package utils

import (
	"context"

	"github.com/gophercloud/gophercloud/v2"
	openstack_networking_tags_v2 "github.com/gophercloud/gophercloud/v2/openstack/networking/v2/extensions/attributestags"
	openstack_networking_ports_v2 "github.com/gophercloud/gophercloud/v2/openstack/networking/v2/ports"
)

// vérifie que le port contient le tag privatetopublic
func ContientTagServer(
	context context.Context,
	networkClient *gophercloud.ServiceClient,
	port openstack_networking_ports_v2.Port,
) bool {

	exists, _ := openstack_networking_tags_v2.Confirm(context, networkClient, "ports", port.ID, "privatetopublic").Extract()
	return exists
}
