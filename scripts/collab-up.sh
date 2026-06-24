#!/bin/bash
# Session de co-édition TEMPS RÉEL sur la VM infra : JupyterLab collaboratif (RTC/Yjs, image
# collab-jupyter:latest baked) montant les fichiers de l'hôte (sshfs). Hôte + binôme ouvrent la
# MÊME session → frappes/curseurs en direct, sans code à échanger. Usage: collab-up.sh <safe_id> <host_ip> → "PORT"
set -euo pipefail
SAFE="$1"; HOST_IP="$2"
KEY=/home/ubuntu/.ssh/student_key
NAME="collab-$SAFE-jl"; MNT="/srv/collab/$SAFE"; PF="/srv/collab/.ports/$SAFE"
mkdir -p /srv/collab/.ports
if [ -f "$PF" ] && docker inspect -f '{{.State.Running}}' "$NAME" 2>/dev/null | grep -q true; then cat "$PF"; exit 0; fi
mkdir -p "$MNT"
mountpoint -q "$MNT" || sshfs -o allow_other,reconnect,IdentityFile=$KEY,StrictHostKeyChecking=no,UserKnownHostsFile=/dev/null "vmuser@$HOST_IP:/home/vmuser" "$MNT"
PORT=9100; while ss -ltnH "sport = :$PORT" 2>/dev/null | grep -q .; do PORT=$((PORT+1)); done
BASE="/api/jupyter-proxy/$NAME/"
docker rm -f "$NAME" 2>/dev/null || true
docker run -d --restart=always --name "$NAME" -p $PORT:$PORT -v "$MNT":/home/jovyan/work collab-jupyter:latest \
  bash -lc "exec jupyter lab --ip=0.0.0.0 --port=$PORT --no-browser --IdentityProvider.token='' --ServerApp.password='' --ServerApp.allow_origin='*' --ServerApp.allow_remote_access=True --ServerApp.disable_check_xsrf=True --ServerApp.base_url='$BASE'" >/dev/null
echo "$PORT" | tee "$PF"
