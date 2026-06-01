#!/bin/bash
# Creates an OpenStack snapshot for each Jupyter environment.
# Each snapshot has docker image pre-pulled → VMs start Jupyter in ~30s instead of 5min.
#
# Usage: ./scripts/make-jupyter-snapshots.sh [env_name]
#   env_name: optional, run only one env (e.g. "scipy")
#
# Requirements: openstack CLI, ssh access with SSH_PRIVATE_KEY_PATH

set -uo pipefail

OS_CLOUD="${OS_CLOUD:-ipp-idcs-vmpool}"
BASE_IMAGE="ubuntu-2204-docker"
FLAVOR="vd.24"
NETWORK="public-2"
KEYPAIR="maelan-mac"
SSH_KEY="${SSH_PRIVATE_KEY_PATH:-$HOME/.ssh/id_ed25519}"
SNAPSHOT_PREFIX="jupyter-snapshot"
POSTGRES_DSN="${POSTGRES_DSN:-postgres://admin:P00lManager_Secure_2026@localhost:5432/control_center?sslmode=disable}"

# Each entry: "snapshot-suffix|docker-image|display-label"
ENVS=(
  "scipy|registry.virtualdata.cloud.idcs.polytechnique.fr/docker-hub-proxy/jupyter/scipy-notebook:latest|Python scientifique (scipy-notebook)"
  "scipy-plus|registry.virtualdata.cloud.idcs.polytechnique.fr/plmlab-hub-proxy/docker-images/scipy-notebook-plus:2023.01.24|Python scientifique+"
  "datascience|registry.virtualdata.cloud.idcs.polytechnique.fr/docker-hub-proxy/jupyter/datascience-notebook:2343e33dec46|Python R Julia (datascience)"
  "julia|registry.virtualdata.cloud.idcs.polytechnique.fr/plmlab-hub-proxy/docker-images/julia:0.0.4|Julia"
  "bio583|registry.virtualdata.cloud.idcs.polytechnique.fr/plmlab-hub-proxy/ip-paris/idcs/docker/bio583:0.0.1|BIO583"
  "eco589|registry.virtualdata.cloud.idcs.polytechnique.fr/gitlab-in2p3-proxy/energy4climate/public/education/eco-589-tutorials:0.2|ECO589"
  "compeco|albop/computational_economics:latest|Computational Economics"
  "mec431|registry.virtualdata.cloud.idcs.polytechnique.fr/gitlab-hub-proxy/bleyerj/x_mec431:040520231145|MEC431"
  "mec558|registry.virtualdata.cloud.idcs.polytechnique.fr/gitlab-in2p3-proxy/ipsl/lmd/intro/jupyterlabimages:07-11-2023|MEC558"
  "map579|registry.virtualdata.cloud.idcs.polytechnique.fr/plmlab-hub-proxy/docker-images/xeus-cling:0.0.5|MAP579"
  "mec552a|registry.virtualdata.cloud.idcs.polytechnique.fr/gitlab-inria-proxy/mgenet/mec552a-repo2docker:latest|MEC552A"
  "mec552b|registry.virtualdata.cloud.idcs.polytechnique.fr/jupyter/mec552b-repo2docker:1d894fa3|MEC552B"
  "mec568|registry.virtualdata.cloud.idcs.polytechnique.fr/gitlab-inria-proxy/mgenet/mec568-repo2docker:latest|MEC568"
  "mec581|registry.virtualdata.cloud.idcs.polytechnique.fr/gitlab-inria-proxy/mgenet/mec-581-repo-2-docker:9b9d98b7|MEC581"
  "mec666|registry.virtualdata.cloud.idcs.polytechnique.fr/gitlab-in2p3-proxy/energy4climate/public/education/climate_change_and_energy_transition:0.2|MEC666"
)

FILTER="${1:-}"

log() { echo "[$(date +%H:%M:%S)] $*"; }
die() { echo "ERROR: $*" >&2; exit 1; }

wait_ssh() {
  local ip=$1
  log "Waiting for SSH on $ip..."
  for i in $(seq 1 60); do
    if ssh -o StrictHostKeyChecking=no -o ConnectTimeout=5 -o BatchMode=yes \
        -i "$SSH_KEY" "vmuser@$ip" "true" 2>/dev/null; then
      return 0
    fi
    sleep 5
  done
  die "SSH never became available on $ip"
}

upsert_config() {
  local suffix=$1 docker_image=$2
  local config_name="jupyter-snapshot-${suffix}"
  local script
  script=$(cat <<SCRIPT
#!/bin/bash
# Start Jupyter (image pre-pulled in snapshot)
until sudo docker info >/dev/null 2>&1; do sleep 2; done
sudo docker run -d --restart=always --name jupyter \
  -p 8888:8888 \
  -e JUPYTER_ENABLE_LAB=yes \
  -v /home/vmuser:/home/jovyan/work \
  --user root \
  -e NB_USER=jovyan \
  -e CHOWN_HOME=yes \
  ${docker_image} \
  start-notebook.sh --NotebookApp.token='' --NotebookApp.password='' --ip=0.0.0.0 \
  || sudo docker start jupyter 2>/dev/null || true
SCRIPT
)
  if [ -n "$POSTGRES_DSN" ] && command -v psql &>/dev/null; then
    psql "$POSTGRES_DSN" -c "
      INSERT INTO config_pools (user_id, name, data)
      VALUES ('system', '${config_name}', \$\$${script}\$\$)
      ON CONFLICT (user_id, name) DO UPDATE SET data = EXCLUDED.data;
    " &>/dev/null && log "[$suffix] Config '$config_name' upserted in PostgreSQL." || true
  fi
}

process_env() {
  local suffix=$1 docker_image=$2 label=$3
  local snapshot_name="${SNAPSHOT_PREFIX}-${suffix}"
  local vm_name="snapshot-builder-${suffix}-$$"

  upsert_config "$suffix" "$docker_image"

  # Skip if snapshot already exists
  if openstack --os-cloud "$OS_CLOUD" image show "$snapshot_name" &>/dev/null; then
    log "[$suffix] Snapshot '$snapshot_name' already exists, skipping."
    return 0
  fi

  log "[$suffix] Starting VM '$vm_name'..."
  local vm_id
  vm_id=$(openstack --os-cloud "$OS_CLOUD" server create \
    --image "$BASE_IMAGE" \
    --flavor "$FLAVOR" \
    --network "$NETWORK" \
    --key-name "$KEYPAIR" \
    --wait \
    --format value -c id \
    "$vm_name")

  log "[$suffix] VM $vm_id created. Getting IP..."
  local ip=""
  for i in $(seq 1 30); do
    ip=$(openstack --os-cloud "$OS_CLOUD" server show "$vm_id" \
      --format value -c addresses 2>/dev/null | grep -oE '[0-9]+\.[0-9]+\.[0-9]+\.[0-9]+' | head -1 || true)
    [ -n "$ip" ] && break
    sleep 3
  done
  [ -z "$ip" ] && { openstack --os-cloud "$OS_CLOUD" server delete "$vm_id" --wait; die "[$suffix] No IP found"; }

  wait_ssh "$ip"

  log "[$suffix] Pulling Docker image: $docker_image"
  ssh -o StrictHostKeyChecking=no -o ConnectTimeout=10 -i "$SSH_KEY" "vmuser@$ip" \
    "sudo docker pull ${docker_image}" || {
      log "[$suffix] WARNING: docker pull failed, snapshot may not work correctly"
    }

  log "[$suffix] Stopping VM for clean snapshot..."
  openstack --os-cloud "$OS_CLOUD" server stop "$vm_id"
  # Wait for SHUTOFF
  for i in $(seq 1 30); do
    status=$(openstack --os-cloud "$OS_CLOUD" server show "$vm_id" --format value -c status)
    [ "$status" = "SHUTOFF" ] && break
    sleep 3
  done

  log "[$suffix] Creating snapshot '$snapshot_name'..."
  snap_id=$(openstack --os-cloud "$OS_CLOUD" server image create \
    --name "$snapshot_name" \
    --format value -c id \
    "$vm_id" || true)
  if [ -z "$snap_id" ]; then
    log "[$suffix] WARNING: snapshot creation returned no ID, checking by name..."
    snap_id=$(openstack --os-cloud "$OS_CLOUD" image list --format value -c ID -c Name | grep "$snapshot_name" | awk '{print $1}' || true)
  fi
  if [ -n "$snap_id" ]; then
    log "[$suffix] Waiting for snapshot $snap_id to become active..."
    for i in $(seq 1 60); do
      snap_status=$(openstack --os-cloud "$OS_CLOUD" image show "$snap_id" --format value -c status 2>/dev/null || echo "error")
      if [ "$snap_status" = "active" ]; then
        log "[$suffix] Snapshot is active."
        break
      fi
      log "[$suffix] Snapshot status: $snap_status (attempt $i/60)..."
      sleep 15
    done
  fi

  log "[$suffix] Deleting build VM..."
  openstack --os-cloud "$OS_CLOUD" server delete "$vm_id" || true
  sleep 10

  log "[$suffix] Done. Snapshot '$snapshot_name' is ready."
}

log "Starting Jupyter snapshot builder (OS_CLOUD=$OS_CLOUD)"
log "Base image: $BASE_IMAGE | Flavor: $FLAVOR | Network: $NETWORK"
echo ""

for entry in "${ENVS[@]}"; do
  IFS='|' read -r suffix docker_image label <<< "$entry"
  if [ -n "$FILTER" ] && [ "$suffix" != "$FILTER" ]; then
    continue
  fi
  process_env "$suffix" "$docker_image" "$label"
done

log "All done."
echo ""
log "Snapshots created (prefix: $SNAPSHOT_PREFIX-):"
openstack --os-cloud "$OS_CLOUD" image list --format value -c Name | grep "^${SNAPSHOT_PREFIX}-" | sort
