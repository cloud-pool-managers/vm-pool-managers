#!/bin/bash
set -euo pipefail

POOL_ID=$(curl -s http://169.254.169.254/openstack/latest/meta_data.json | jq -r .meta.serverpool_id)
USER_ID=$(curl -s http://169.254.169.254/openstack/latest/meta_data.json | jq -r .meta.user_id)
JUPYTER_BASE_URL="/api/jupyter-proxy/${POOL_ID}/${USER_ID}/"

mkdir -p /home/vmuser/nbgrader/{source,release,submitted,autograded,feedback,exchange}

chown -R vmuser:vmuser /home/vmuser/nbgrader /home/vmuser/jupyter-env || true

cat > /home/vmuser/nbgrader/nbgrader_config.py << 'NBCFG'
c = get_config()
c.CourseDirectory.root = '/home/vmuser/nbgrader'
c.CourseDirectory.course_id = 'course'
c.Exchange.root = '/home/vmuser/nbgrader/exchange'
NBCFG
chown vmuser:vmuser /home/vmuser/nbgrader/nbgrader_config.py

cat > /etc/systemd/system/jupyterlab.service << SVC
[Unit]
Description=JupyterLab
After=network.target

[Service]
Type=simple
User=vmuser
WorkingDirectory=/home/vmuser/nbgrader
ExecStart=/home/vmuser/jupyter-env/bin/jupyter lab \
  --no-browser --ip=0.0.0.0 --port=8888 \
  --ServerApp.token='' \
  --ServerApp.password='' \
  --ServerApp.allow_origin='*' \
  --ServerApp.allow_remote_access=True \
  --ServerApp.base_url=${JUPYTER_BASE_URL} \
  --ServerApp.disable_check_xsrf=True
Restart=always
RestartSec=5

[Install]
WantedBy=multi-user.target
SVC

systemctl daemon-reload
systemctl unmask jupyterlab || true
systemctl enable jupyterlab

# Enable nbgrader extensions
sudo -u vmuser /home/vmuser/jupyter-env/bin/jupyter nbextension install --sys-prefix --py nbgrader --overwrite
sudo -u vmuser /home/vmuser/jupyter-env/bin/jupyter nbextension enable --sys-prefix --py nbgrader
sudo -u vmuser /home/vmuser/jupyter-env/bin/jupyter serverextension enable --sys-prefix --py nbgrader

systemctl restart jupyterlab

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
  systemctl restart codeserver
fi
