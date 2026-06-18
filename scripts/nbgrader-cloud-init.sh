#!/bin/bash
# nbgrader-instructor cloud-init script
# This script is stored as a ConfigPool and applied to instructor VMs at boot.
# It installs JupyterLab + nbgrader and starts JupyterLab as a systemd service.
set -euo pipefail
export DEBIAN_FRONTEND=noninteractive

apt-get update -q
# Install NFS client and dependencies
apt-get install -y python3-pip python3-venv git sqlite3 nfs-common jq curl

# Get metadata for pool_id and user_id to correctly set the Jupyter base_url
POOL_ID=$(curl -s http://169.254.169.254/openstack/latest/meta_data.json | jq -r .meta.serverpool_id)
USER_ID=$(curl -s http://169.254.169.254/openstack/latest/meta_data.json | jq -r .meta.user_id)
JUPYTER_BASE_URL="/api/jupyter-proxy/${POOL_ID}/${USER_ID}/"

# Install in a virtualenv to avoid system package conflicts
python3 -m venv /home/vmuser/jupyter-env
/home/vmuser/jupyter-env/bin/pip install --quiet --upgrade pip
/home/vmuser/jupyter-env/bin/pip install --quiet jupyterlab nbgrader

# Enable nbgrader extensions
/home/vmuser/jupyter-env/bin/jupyter nbextension install --sys-prefix --py nbgrader --quiet || true
/home/vmuser/jupyter-env/bin/jupyter nbextension enable --sys-prefix --py nbgrader --quiet || true
/home/vmuser/jupyter-env/bin/jupyter serverextension enable --sys-prefix --py nbgrader --quiet || true

# Create nbgrader directory structure
mkdir -p /home/vmuser/nbgrader/{source,release,submitted,autograded,feedback,exchange}

# Mount the NFS exchange directory
# Note: Ensure NFS_SERVER_IP is replaced with your actual NFS server IP if not injected dynamically
NFS_SERVER_IP="157.136.249.205" # Default placeholder, will be updated by deploy script or user
mount -t nfs ${NFS_SERVER_IP}:/srv/nbgrader/exchange /home/vmuser/nbgrader/exchange
echo "${NFS_SERVER_IP}:/srv/nbgrader/exchange /home/vmuser/nbgrader/exchange nfs defaults 0 0" >> /etc/fstab

chown -R vmuser:vmuser /home/vmuser/nbgrader /home/vmuser/jupyter-env

# nbgrader config
cat > /home/vmuser/nbgrader/nbgrader_config.py << 'NBCFG'
c = get_config()
c.CourseDirectory.root = '/home/vmuser/nbgrader'
c.CourseDirectory.course_id = 'course'
c.Exchange.root = '/home/vmuser/nbgrader/exchange'
NBCFG

# JupyterLab systemd service
cat > /etc/systemd/system/jupyterlab.service << SVC
[Unit]
Description=JupyterLab Instructor
After=network.target

[Service]
Type=simple
User=vmuser
WorkingDirectory=/home/vmuser/nbgrader
ExecStart=/home/vmuser/jupyter-env/bin/jupyter lab \
  --no-browser --ip=0.0.0.0 --port=8888 \
  --ServerApp.token='' \
  --ServerApp.password='' \
  --ServerApp.allow_origin=* \
  --ServerApp.allow_remote_access=True \
  --ServerApp.base_url=${JUPYTER_BASE_URL}
Restart=always
RestartSec=5

[Install]
WantedBy=multi-user.target
SVC

systemctl daemon-reload
systemctl enable jupyterlab
systemctl start jupyterlab

# --- VS Code (code-server) à côté de Jupyter, port 8080 ---
# Installé au runtime (aucune image modifiée). Non-fatal : si l'install échoue
# (réseau), Jupyter reste fonctionnel. --auth none : même modèle d'accès que Jupyter.
if ! command -v code-server >/dev/null 2>&1; then
  curl -fsSL https://code-server.dev/install.sh | sh || true
fi
if command -v code-server >/dev/null 2>&1; then
  # Extensions Python + Jupyter (depuis Open VSX) pour exécuter les notebooks
  # via le serveur Jupyter local (localhost:8888) = même environnement.
  sudo -u vmuser /usr/bin/code-server --install-extension ms-python.python --install-extension ms-toolsai.jupyter || true
  cat > /etc/systemd/system/codeserver.service << 'SVC'
[Unit]
Description=code-server (VS Code Web)
After=network.target

[Service]
Type=simple
User=vmuser
ExecStart=/usr/bin/code-server --auth none --cert --bind-addr 0.0.0.0:8443 /home/vmuser
Restart=always
RestartSec=5

[Install]
WantedBy=multi-user.target
SVC
  systemctl daemon-reload
  systemctl enable codeserver
  systemctl start codeserver
fi
