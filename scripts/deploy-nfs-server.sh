#!/bin/bash
# deploy-nfs-server.sh
# Deploys an NFS server VM on OpenStack for nbgrader exchange

set -euo pipefail

export OS_CLIENT_CONFIG_FILE="${OS_CLIENT_CONFIG_FILE:-$HOME/.config/openstack/clouds.yaml}"
export OS_CLOUD="${INFRA_OS_CLOUD:-ipp-idcs-vmpoolmanager}"

VM_NAME="infra-nfs-nbgrader"
IMAGE="ubuntu-2404.amd64-genericcloud.20260108"
FLAVOR="vd.1"
NETWORK="public-2"
KEY_NAME="macbook" # You may need to change this if your SSH key name is different in OpenStack

if [ -z "${API_KEYNAME:-}" ]; then
  # Try to guess the keyname from the env or use a fallback
  if pkgx openstack keypair list | grep -qi "macbook"; then
      KEY_NAME="macbook"
  else
      KEY_NAME=$(pkgx openstack keypair list -c Name -f value | head -n 1)
  fi
else
  KEY_NAME="$API_KEYNAME"
fi

echo "Deploying NFS server on $OS_CLOUD..."
echo "Image: $IMAGE"
echo "Flavor: $FLAVOR"
echo "Network: $NETWORK"
echo "Keypair: $KEY_NAME"

# Create the server
SERVER_ID=$(pkgx openstack server create \
    --image "$IMAGE" \
    --flavor "$FLAVOR" \
    --network "$NETWORK" \
    --key-name "$KEY_NAME" \
    --user-data scripts/nfs-cloud-init.yaml \
    --format value -c id \
    "$VM_NAME")

echo "Server created with ID: $SERVER_ID"
echo "Waiting for server to become ACTIVE..."

while true; do
    STATUS=$(pkgx openstack server show "$SERVER_ID" -c status -f value)
    if [ "$STATUS" == "ACTIVE" ]; then
        break
    elif [ "$STATUS" == "ERROR" ]; then
        echo "Error: Server failed to build!"
        exit 1
    fi
    sleep 5
done

# Get the IP address
NFS_IP=$(pkgx openstack server show "$SERVER_ID" -c addresses -f value | grep -oE '[0-9]+\.[0-9]+\.[0-9]+\.[0-9]+' | head -n 1)

echo ""
echo "✅ NFS Server successfully deployed!"
echo "NFS_SERVER_IP=$NFS_IP"
echo ""
echo "Please add the following to your .env file:"
echo "NFS_SERVER_IP=$NFS_IP"
