#!/bin/bash
# Session de collaboration code-server sur la VM infra : monte les fichiers de l'hôte (sshfs)
# et lance un code-server RW (+ un RO :ro). Idempotent. Usage: collab-up.sh <safe_id> <host_ip>
# Imprime sur stdout : "RW_PORT RO_PORT"
set -euo pipefail
SAFE="$1"; HOST_IP="$2"
KEY=/home/ubuntu/.ssh/student_key
REG=registry.virtualdata.cloud.idcs.polytechnique.fr/docker-hub-proxy/codercom/code-server:latest
MNT="/srv/collab/$SAFE"; PF="/srv/collab/.ports/$SAFE"
mkdir -p /srv/collab/.ports
if [ -f "$PF" ] && docker inspect -f '{{.State.Running}}' "collab-$SAFE-rw" 2>/dev/null | grep -q true; then
  cat "$PF"; exit 0
fi
mkdir -p "$MNT"
mountpoint -q "$MNT" || sshfs -o allow_other,reconnect,IdentityFile=$KEY,StrictHostKeyChecking=no,UserKnownHostsFile=/dev/null "vmuser@$HOST_IP:/home/vmuser" "$MNT"
fp(){ local p=$1; while ss -ltnH "sport = :$p" 2>/dev/null | grep -q .; do p=$((p+1)); done; echo $p; }
RW=$(fp 9000); RO=$(fp $((RW+1)))
docker rm -f "collab-$SAFE-rw" "collab-$SAFE-ro" 2>/dev/null || true
docker run -d --restart=always --name "collab-$SAFE-rw" -p $RW:$RW -v "$MNT":/home/coder/project \
  $REG --auth none --cert --bind-addr 0.0.0.0:$RW /home/coder/project >/dev/null
docker run -d --restart=always --name "collab-$SAFE-ro" -p $RO:$RO -v "$MNT":/home/coder/project:ro \
  --entrypoint /bin/bash $REG -lc "mkdir -p ~/.local/share/code-server/User; printf '{\"files.readonlyInclude\":{\"**/*\":true}}' > ~/.local/share/code-server/User/settings.json; exec code-server --auth none --cert --bind-addr 0.0.0.0:$RO /home/coder/project" >/dev/null
printf "%s %s\n" "$RW" "$RO" | tee "$PF"
