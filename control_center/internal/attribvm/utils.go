package attribvm

import (
	"bytes"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"control_center/internal/sshinject"
	"control_center/models"

	"golang.org/x/crypto/ssh"
)

//
// ========== ENTRY POINT ==========
//

func (s *Service) installRclone(server *models.Server, student *models.Student) error {
	username := sshinject.UsernameFromEmail(student.Name)

	signer, err := sshinject.LoadPrivateKey(os.Getenv("SSH_PRIVATE_KEY_PATH"))
	if err != nil {
		return err
	}

	config := sshinject.SshConfig("vmuser", signer)
	addr := fmt.Sprintf("%s:22", server.IP_Address)

	client, err := ssh.Dial("tcp", addr, config)
	if err != nil {
		return err
	}
	defer client.Close()

	// --- LOCAL DEPOT VM ---
	log.Println("ensureLocalDepotFolder")
	if err := ensureLocalDepotFolder(username); err != nil {
		return err
	}

	// --- REMOTE VM SSH ---
	log.Println("ensureRemoteSSHKey")
	if err := sshinject.RunSSHcmd(client, ensureRemoteSSHKeyCmd(username)); err != nil {
		return err
	}

	// --- REMOTE VM authorized_keys ---
	log.Println("authorizeDepotKey: reading remote pubkey")
	pubKey, err := sshinject.RunSSHcmdWithOutput(client, readRemotePubKeyCmd(username))
	if err != nil {
		log.Printf("ERROR reading remote pubkey: %v", err)
		return err
	}

	pubKey = strings.TrimSpace(pubKey)
	pubKey = strings.ReplaceAll(pubKey, `"`, `\"`)
	if err := authorizeDepotKey(pubKey); err != nil {
		return err
	}

	// --- REMOTE VM depot folder ---
	log.Println("ensureRemoteMountPoint")
	if err := sshinject.RunSSHcmd(client, ensureRemoteMountPointCmd(username)); err != nil {
		return err
	}

	// --- REMOTE VM rclone config ---
	log.Println("rCloneConfig")
	if err := sshinject.RunSSHcmd(client, rcloneConfigCmd(username)); err != nil {
		return err
	}

	// --- REMOTE VM systemd rclone mount ---
	log.Println("systemd setup")
	if err := sshinject.RunSSHcmd(client, rcloneSystemdCmd(username)); err != nil {
		return err
	}

	return nil
}

//
// ========== LOCAL (DEPOT VM) ==========
//

func ensureLocalDepotFolder(username string) error {
	path := filepath.Join("/home/ubuntu/depot", username)
	log.Printf("ensureLocalDepotFolder: %s", path)
	return os.MkdirAll(path, 0700)
}

func authorizeDepotKey(pubKey string) error {
	cmd := fmt.Sprintf(`
set -eux

KEY="%s"
FILE=/home/ubuntu/.ssh/authorized_keys

echo ">>> Installing key:"
echo "$KEY"

install -d -m 700 -o ubuntu -g ubuntu /home/ubuntu/.ssh
touch "$FILE"
chmod 600 "$FILE"
chown ubuntu:ubuntu "$FILE"

grep -qxF "$KEY" "$FILE" || echo "$KEY" >> "$FILE"
`, pubKey)

	log.Println("authorizeDepotKey: running local cmd")
	log.Println("----- BEGIN CMD -----")
	log.Println(cmd)
	log.Println("----- END CMD -----")

	return runLocalCmd(cmd)
}

//
// ========== REMOTE VM ==========
//

func ensureRemoteSSHKeyCmd(username string) string {
	return fmt.Sprintf(`
set -e
HOME=/home/%[1]s
SSH=$HOME/.ssh

sudo -u %[1]s mkdir -p "$SSH"
sudo -u %[1]s chmod 700 "$SSH"

if [ ! -f "$SSH/id_ed25519" ]; then
  sudo -u %[1]s ssh-keygen -t ed25519 -f "$SSH/id_ed25519" -N ""
fi

sudo -u %[1]s chmod 600 "$SSH/id_ed25519"
sudo -u %[1]s chmod 644 "$SSH/id_ed25519.pub"
`, username)
}

func readRemotePubKeyCmd(username string) string {
	// lecture en tant que l’utilisateur final pour bypasser les permissions
	return fmt.Sprintf(`sudo -u %s cat /home/%s/.ssh/id_ed25519.pub`, username, username)
}

func ensureRemoteMountPointCmd(username string) string {
	return fmt.Sprintf(`
sudo mkdir -p /home/%[1]s/depot
sudo chown %[1]s:%[1]s /home/%[1]s/depot
sudo chmod 700 /home/%[1]s/depot
`, username)
}

func rcloneConfigCmd(username string) string {
	return fmt.Sprintf(`
sudo -u %[1]s mkdir -p /home/%[1]s/.config/rclone

sudo -u %[1]s tee /home/%[1]s/.config/rclone/rclone.conf > /dev/null << EOF
[depot_%[1]s]
type = sftp
host = 157.136.252.74
user = ubuntu
key_file = /home/%[1]s/.ssh/id_ed25519
shell_type = unix
EOF

sudo chown %[1]s:%[1]s /home/%[1]s/.config/rclone/rclone.conf
sudo chmod 600 /home/%[1]s/.config/rclone/rclone.conf
`, username)
}

func rcloneSystemdCmd(username string) string {
	return fmt.Sprintf(`
set -e

SERVICE=/etc/systemd/system/rclone-depot-%[1]s.service

sudo tee "$SERVICE" > /dev/null << EOF
[Unit]
Description=Rclone depot mount for %[1]s
After=network-online.target
Wants=network-online.target

[Service]
User=%[1]s
ExecStart=/usr/bin/rclone mount depot_%[1]s:/home/ubuntu/depot/%[1]s /home/%[1]s/depot \
  --vfs-cache-mode writes \
  --log-file /home/%[1]s/.rclone_mount.log \
  --log-level INFO
ExecStop=/bin/fusermount -u /home/%[1]s/depot
Restart=on-failure

[Install]
WantedBy=multi-user.target
EOF

sudo chmod 644 "$SERVICE"
sudo systemctl daemon-reload
sudo systemctl enable rclone-depot-%[1]s.service
sudo systemctl start rclone-depot-%[1]s.service
`, username)
}

//
// ========== UTIL ==========
//

func runLocalCmd(cmd string) error {
	log.Println("runLocalCmd: executing")

	c := exec.Command("bash", "-c", cmd)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	c.Stdout = &stdout
	c.Stderr = &stderr

	err := c.Run()

	log.Printf("runLocalCmd STDOUT:\n%s", stdout.String())
	log.Printf("runLocalCmd STDERR:\n%s", stderr.String())

	if err != nil {
		return fmt.Errorf("runLocalCmd failed: %w", err)
	}
	return nil
}
