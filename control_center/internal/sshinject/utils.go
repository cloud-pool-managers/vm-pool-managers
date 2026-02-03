package sshinject

import (
	"bytes"
	cc "control_center/config"
	"control_center/models"
	"fmt"
	"log"
	"os"
	"regexp"
	"strings"
	"time"

	"golang.org/x/crypto/ssh"
)

func LoadPrivateKey(path string) (ssh.Signer, error) {
	// log.Printf("path: %s\n", path)
	key, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return ssh.ParsePrivateKey(key)
}

func SshConfig(user string, signer ssh.Signer) *ssh.ClientConfig {
	return &ssh.ClientConfig{
		User: user,
		Auth: []ssh.AuthMethod{
			ssh.PublicKeys(signer),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         10 * time.Second,
	}
}

func RunSSHcmd(client *ssh.Client, cmd string) error {
	session, err := client.NewSession()
	if err != nil {
		return fmt.Errorf("new session: %w", err)
	}
	defer session.Close()

	var stderr bytes.Buffer
	var stdout bytes.Buffer

	session.Stdout = &stdout
	session.Stderr = &stderr

	if err := session.Run(cmd); err != nil {
		log.Printf("SSH stdout: %s", stdout.String())
		log.Printf("SSH stderr: %s", stderr.String())
		if stderr.Len() > 0 {
			return fmt.Errorf("ssh command error: %s", stderr.String())
		}
		return fmt.Errorf("ssh command failed: %w", err)
	}

	return nil
}

func UsernameFromEmail(email string) string {
	local := strings.Split(email, "@")[0]
	local = strings.ToLower(local)

	// remplacer caractères interdits
	re := regexp.MustCompile(`[^a-z0-9_.-]`)
	local = re.ReplaceAllString(local, "")

	if len(local) > 32 {
		local = local[:32]
	}

	return local
}

func RetryConfigureSSHUserNFS(server *models.Server, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	delay := 5 * time.Second

	for {
		err := configureSSHUserNFS(server)
		if err == nil {
			log.Printf("[SSH][OK] %s configured or already configured\n", server.IP_Address)
			return nil
		}

		if time.Now().After(deadline) {
			return fmt.Errorf("timeout after %s: %w", timeout, err)
		}

		log.Printf("[SSH][WAIT] %s not ready yet: %v\n", server.IP_Address, err)
		time.Sleep(delay)

		if delay < 30*time.Second {
			delay *= 2
		}
	}
}

func configureSSHUserNFS(server *models.Server) error {
	signer, err := LoadPrivateKey(os.Getenv("SSH_PRIVATE_KEY_PATH"))
	if err != nil {
		return err
	}

	config := SshConfig("vmuser", signer)

	addr := fmt.Sprintf("%s:22", server.IP_Address)
	client, err := ssh.Dial("tcp", addr, config)
	if err != nil {
		return err
	}
	defer client.Close()

	var user models.User
	if err := cc.Database.
		Where("email = ?", server.UserID).
		First(&user).Error; err != nil {
		return fmt.Errorf("fetch user failed: %w", err)
	}

	cmd := cmdInitNFS(user)
	if err := RunSSHcmd(client, cmd); err != nil {
		return fmt.Errorf("run ssh cmd failed: %w", err)
	}
	return nil
}

func cmdInitNFS(user models.User) string {
	userUsername := UsernameFromEmail(user.Email)

	return fmt.Sprintf(`
set -e

MARKER="/var/lib/poolmanager/nfs_config_done"
POOL_MOUNT="/mnt/pool"
PROF_GROUP="pool_prof"
STUDENT_GROUP="pool_student"

if [ -f "$MARKER" ]; then
  echo "[NFS] already configured"
  exit 0
fi

ensure_group() {
  GROUP="$1"
  if ! getent group "$GROUP" >/dev/null; then
    sudo groupadd "$GROUP"
  fi
}

create_user() {
  USERNAME="$1"
  PUBKEY="$2"
  ROLE="$3" # prof | student

  if ! id "$USERNAME" >/dev/null 2>&1; then
    if [ -d "/home/$USERNAME" ]; then
      sudo useradd -M -s /bin/bash "$USERNAME"
    else
      sudo useradd -m -s /bin/bash "$USERNAME"
    fi
  fi

  HOME="/home/$USERNAME"
  SSH="$HOME/.ssh"
  AUTH="$SSH/authorized_keys"

  sudo mkdir -p "$SSH"
  sudo chmod 700 "$SSH"
  sudo touch "$AUTH"
  sudo chmod 600 "$AUTH"

  if ! sudo grep -qxF "$PUBKEY" "$AUTH"; then
    echo "$PUBKEY" | sudo tee -a "$AUTH" > /dev/null
  fi

  if [ "$ROLE" = "prof" ]; then
    sudo usermod -aG sudo "$USERNAME"
    sudo usermod -aG "$PROF_GROUP" "$USERNAME"
  else
    sudo usermod -aG "$STUDENT_GROUP" "$USERNAME"
  fi

  sudo chown -R "$USERNAME:$USERNAME" "$SSH"

  # Lien vers le pool NFS (idempotent)
  sudo ln -sfn "$POOL_MOUNT" "$HOME/pool"
  sudo chown -h "$USERNAME:$USERNAME" "$HOME/pool"
}

sudo chmod 755 /home

ensure_group "$PROF_GROUP"
ensure_group "$STUDENT_GROUP"

create_user "%s" "%s" "prof"

sudo mkdir -p /var/lib/poolmanager
sudo touch "$MARKER"
sudo chmod 644 "$MARKER"

echo "[NFS] user configuration done"
`,
		userUsername,
		user.Keypubuser,
	)
}

// func cmdInitNFS(user models.User) string {
// 	userUsername := UsernameFromEmail(user.Email)

// 	return fmt.Sprintf(`
// set -e

// MARKER="/var/lib/poolmanager/nfs_config_done"
// POOL_MOUNT="/mnt/pool"
// PROF_GROUP="pool_prof"

// if [ -f "$MARKER" ]; then
//   echo "NFS users already configured"
//   exit 0
// fi

// ensure_group() {
//   GROUP="$1"
//   if ! getent group "$GROUP" >/dev/null; then
//     sudo groupadd "$GROUP"
//   fi
// }

// create_user() {
//   USERNAME="$1"
//   PUBKEY="$2"
//   ROLE="$3" # prof | student

//   if ! id "$USERNAME" >/dev/null 2>&1; then
//     if [ -d "/home/$USERNAME" ]; then
//       sudo useradd -M -s /bin/bash "$USERNAME"
//     else
//       sudo useradd -m -s /bin/bash "$USERNAME"
//     fi
//   fi

//   HOME="/home/$USERNAME"
//   SSH="$HOME/.ssh"
//   AUTH="$SSH/authorized_keys"

//   sudo mkdir -p "$SSH"
//   sudo chmod 700 "$SSH"
//   sudo touch "$AUTH"
//   sudo chmod 600 "$AUTH"

//   if ! sudo grep -qxF "$PUBKEY" "$AUTH"; then
//     echo "$PUBKEY" | sudo tee -a "$AUTH" > /dev/null
//   fi

//   if [ "$ROLE" = "prof" ]; then
//     sudo usermod -aG sudo "$USERNAME"
//     sudo usermod -aG "$PROF_GROUP" "$USERNAME"
//   else
//     sudo usermod -aG "$STUDENT_GROUP" "$USERNAME"
//   fi

//   sudo chown -R "$USERNAME:$USERNAME" "$SSH"

//   # Lien NFS dans le home (idempotent)
//   sudo ln -sfn "$POOL_MOUNT" "$HOME/pool"
//   sudo chown -h "$USERNAME:$USERNAME" "$HOME/pool"
// }

// sudo chmod 755 /home

// ensure_group "$PROF_GROUP"
// create_user "%s" "%s" "prof"

// sudo mkdir -p /var/lib/poolmanager
// sudo touch "$MARKER"
// sudo chmod 644 "$MARKER"

// echo "NFS user configuration done"
// `,
// 		userUsername,
// 		user.Keypubuser,
// 	)
// }
