#!/bin/bash
# nbgrader-instructor cloud-init script
# This script is stored as a ConfigPool and applied to instructor VMs at boot.
# It installs JupyterLab + nbgrader and starts JupyterLab as a systemd service.
set -euo pipefail
export DEBIAN_FRONTEND=noninteractive

apt-get update -q
apt-get install -y python3-pip python3-venv git sqlite3

# Install in a virtualenv to avoid system package conflicts
python3 -m venv /home/vmuser/jupyter-env
/home/vmuser/jupyter-env/bin/pip install --quiet --upgrade pip
/home/vmuser/jupyter-env/bin/pip install --quiet jupyterlab nbgrader

# Enable nbgrader extensions
/home/vmuser/jupyter-env/bin/jupyter nbextension install --sys-prefix --py nbgrader --quiet || true
/home/vmuser/jupyter-env/bin/jupyter nbextension enable --sys-prefix --py nbgrader --quiet || true
/home/vmuser/jupyter-env/bin/jupyter serverextension enable --sys-prefix --py nbgrader --quiet || true

# Create nbgrader directory structure
mkdir -p /home/vmuser/nbgrader/{source,release,submitted,autograded,feedback}
chown -R vmuser:vmuser /home/vmuser/nbgrader /home/vmuser/jupyter-env

# nbgrader config
cat > /home/vmuser/nbgrader/nbgrader_config.py << 'NBCFG'
c = get_config()
c.CourseDirectory.root = '/home/vmuser/nbgrader'
c.CourseDirectory.course_id = 'course'
NBCFG

# JupyterLab systemd service
cat > /etc/systemd/system/jupyterlab.service << 'SVC'
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
  --ServerApp.allow_remote_access=True
Restart=always
RestartSec=5

[Install]
WantedBy=multi-user.target
SVC

systemctl daemon-reload
systemctl enable jupyterlab
systemctl start jupyterlab
