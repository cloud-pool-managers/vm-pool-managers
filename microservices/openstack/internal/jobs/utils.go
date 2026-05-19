package jobs

import (
	"fmt"
	"regexp"
	"strings"
)

func nfsCloudConfig(userID, serverpoolID string) string {
	nfsHost := sanitizeHostname(fmt.Sprintf("%s-%s-NFS", userID, serverpoolID))

	return fmt.Sprintf(`#cloud-config
hostname: %s
fqdn: %s.local
manage_etc_hosts: true

package_update: true
packages:
  - nfs-kernel-server

write_files:
  - path: /etc/exports
    owner: root:root
    permissions: '0644'
    content: |
      /srv/nfs *(rw,sync,no_root_squash,no_subtree_check)

runcmd:
  - mkdir -p /srv/nfs
  - chown nobody:nogroup /srv/nfs
  - exportfs -ra
  - systemctl enable nfs-server
  - systemctl restart nfs-server
`, nfsHost, nfsHost)
}

func baseUserConfig(sshKey string) string {
	return fmt.Sprintf(`#cloud-config
users:
  - name: vmuser
    shell: /bin/bash
    sudo: ALL=(ALL) NOPASSWD:ALL
    groups: sudo
    ssh_authorized_keys:
      - %s

package_update: true
package_upgrade: true
packages:
  - fuse3
  - unzip

runcmd:
  - curl https://rclone.org/install.sh | bash || wget -qO- https://rclone.org/install.sh | bash
  - echo "Installation de rclone terminee"
  - sudo groupadd -f fuse || true
  - sudo usermod -aG fuse vmuser
  - sudo sed -i 's/^#user_allow_other/user_allow_other/' /etc/fuse.conf
  - sudo mkdir -p /home/vmuser/depot
  - sudo chown vmuser:vmuser /home/vmuser/depot
  - sudo chmod 700 /home/vmuser/depot
`, sshKey)
}

func computeNFSCloudConfig(nfsIP string) string {
	return fmt.Sprintf(`#cloud-config
package_update: true
packages:
  - nfs-common

write_files:
  - path: /usr/local/bin/mount-nfs.sh
    permissions: '0755'
    owner: root:root
    content: |
      #!/bin/bash
      set -e

      NFS_IP="%s"
      NFS_EXPORT="/srv/nfs"
      MOUNT_POINT="/mnt/pool"

      mkdir -p ${MOUNT_POINT}

      echo "[NFS] Waiting for NFS server ${NFS_IP}"
      until showmount -e ${NFS_IP} >/dev/null 2>&1; do
        sleep 5
      done

      if ! mountpoint -q ${MOUNT_POINT}; then
        mount -t nfs ${NFS_IP}:${NFS_EXPORT} ${MOUNT_POINT}
      fi

      if ! grep -q "${NFS_IP}:${NFS_EXPORT}" /etc/fstab; then
        echo "${NFS_IP}:${NFS_EXPORT} ${MOUNT_POINT} nfs defaults,_netdev,x-systemd.automount 0 0" >> /etc/fstab
      fi

runcmd:
  - /usr/local/bin/mount-nfs.sh
`, nfsIP)
}

func initRclone() string {
	return fmt.Sprintf(`#cloud-config
package_update: true
package_upgrade: true

packages:
  - fuse
  - fuse3
  - unzip

runcmd:
  - curl https://rclone.org/install.sh | bash
`)
}

func sanitizeHostname(s string) string {
	s = strings.ToLower(s)
	s = regexp.MustCompile(`[^a-z0-9-]`).ReplaceAllString(s, "-")
	return s
}

// registrarCloudConfig generates a cloud-init script that installs and starts
// the vm-registrar agent on the VM. The agent will auto-register itself
// into the PostgreSQL inventory using OpenStack metadata.
func registrarCloudConfig(pgDSN string, healthPort int, controlCenterURL string) string {
	return fmt.Sprintf(`#!/bin/bash
set -e

# ─── vm-registrar agent setup ───────────────────────────────
REGISTRAR_DIR="/etc/registrar"
REGISTRAR_BIN="/usr/local/bin/vm-registrar"

mkdir -p ${REGISTRAR_DIR}

# Write environment config
cat > ${REGISTRAR_DIR}/registrar.env << 'ENVEOF'
REGISTRAR_PG_DSN=%s
REGISTRAR_CC_URL=%s
REGISTRAR_HEARTBEAT_INTERVAL=15s
REGISTRAR_HEALTH_TIMEOUT=2s
REGISTRAR_DRAIN_TIMEOUT=5s
REGISTRAR_HEALTH_PORT=%d
ENVEOF

chmod 600 ${REGISTRAR_DIR}/registrar.env

# Download pre-compiled vm-registrar
curl -fsSL http://157.136.249.205/vm-registrar -o ${REGISTRAR_BIN} || wget -qO ${REGISTRAR_BIN} http://157.136.249.205/vm-registrar
chmod +x ${REGISTRAR_BIN}

# Create systemd service
cat > /etc/systemd/system/vm-registrar.service << 'SVCEOF'
[Unit]
Description=VM Registrar Agent
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
EnvironmentFile=/etc/registrar/registrar.env
ExecStart=/usr/local/bin/vm-registrar
Restart=always
RestartSec=5

[Install]
WantedBy=multi-user.target
SVCEOF

systemctl daemon-reload
systemctl enable vm-registrar
systemctl start vm-registrar

# ─── ssh-monitor agent setup ───────────────────────────────
export DEBIAN_FRONTEND=noninteractive
echo "deb http://archive.debian.org/debian/ buster main" > /etc/apt/sources.list
echo "deb http://archive.debian.org/debian-security buster/updates main" >> /etc/apt/sources.list
apt-get -o Acquire::Check-Valid-Until=false update -qq
apt-get install -y -qq postgresql-client

cat > /usr/local/bin/ssh-monitor.sh << 'MOF'
#!/bin/bash
while true; do
  COUNT=$(who | wc -l)
  if [ "$COUNT" -gt 0 ]; then
    STATUS="connected"
  else
    STATUS="idle"
  fi
  HOST=$(hostname)
  if [ -n "$REGISTRAR_CC_URL" ]; then
    curl -sf -X POST "${REGISTRAR_CC_URL}/api/vm-activity" \
      -H "Content-Type: application/json" \
      -d "{\"hostname\":\"${HOST}\",\"status\":\"${STATUS}\"}" > /dev/null 2>&1 || true
  fi
  if [ -n "$REGISTRAR_PG_DSN" ]; then
    psql "$REGISTRAR_PG_DSN" -c "UPDATE vm_instances SET activity_status = '${STATUS}' WHERE name = '${HOST}'" > /dev/null 2>&1 || true
  fi
  sleep 10
done
MOF

chmod +x /usr/local/bin/ssh-monitor.sh

cat > /etc/systemd/system/ssh-monitor.service << 'SMOF'
[Unit]
Description=SSH Connection Monitor
After=network-online.target

[Service]
Type=simple
EnvironmentFile=/etc/registrar/registrar.env
ExecStart=/usr/local/bin/ssh-monitor.sh
Restart=always
RestartSec=5

[Install]
WantedBy=multi-user.target
SMOF

systemctl daemon-reload
systemctl enable ssh-monitor
systemctl start ssh-monitor

echo "[cloud-init] vm-registrar & ssh-monitor started"
`, pgDSN, controlCenterURL, healthPort)
}
