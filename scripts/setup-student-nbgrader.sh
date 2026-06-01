#!/bin/bash
# setup-student-nbgrader.sh
# Configures nbgrader (student mode) on a student VM.
# Called automatically during VM setup if the pool has nbgrader enabled.
# Usage: ./scripts/setup-student-nbgrader.sh <student-vm-ip> <student-name> <instructor-vm-ip>

set -euo pipefail

STUDENT_IP="${1:?Usage: $0 <student-vm-ip> <student-name> <instructor-vm-ip>}"
STUDENT_NAME="${2:?missing student-name}"
INSTRUCTOR_IP="${3:?missing instructor-vm-ip}"
SSH_KEY="${SSH_PRIVATE_KEY_PATH:-$HOME/.ssh/id_ed25519}"

echo "=== Configuring nbgrader (student mode) on $STUDENT_IP for $STUDENT_NAME ==="

ssh -i "$SSH_KEY" -o StrictHostKeyChecking=no "vmuser@$STUDENT_IP" bash << ENDSSH
set -euo pipefail

# Install nbgrader if not present
if ! command -v nbgrader &>/dev/null; then
    pip install --quiet nbgrader 2>/dev/null || sudo pip3 install --quiet nbgrader
fi

# Create student assignment directories and mount NFS exchange
mkdir -p /home/vmuser/nbgrader/{assignments,submitted,exchange}

# Configure nbgrader in student mode
mkdir -p /home/vmuser/.jupyter
cat > /home/vmuser/nbgrader/nbgrader_config.py << 'NBCFG'
c = get_config()
c.CourseDirectory.root = '/home/vmuser/nbgrader'
c.CourseDirectory.course_id = 'course'
c.Exchange.root = '/home/vmuser/nbgrader/exchange'
NBCFG

# Enable only the assignment list extension (student side)
jupyter nbextension install --user --py nbgrader --quiet 2>/dev/null || true
jupyter nbextension enable --user --py nbgrader --quiet 2>/dev/null || true
jupyter serverextension enable --user --py nbgrader --quiet 2>/dev/null || true

# Install NFS client and mount the exchange directory
sudo apt-get update -qq
sudo apt-get install -y nfs-common

# Retrieve the NFS Server IP. If not passed as an env var, we fallback to a placeholder.
# In a real environment, this should be dynamically injected or loaded.
NFS_SERVER_IP="${NFS_SERVER_IP:-157.136.249.205}"
sudo mount -t nfs \${NFS_SERVER_IP}:/srv/nbgrader/exchange /home/vmuser/nbgrader/exchange
echo "\${NFS_SERVER_IP}:/srv/nbgrader/exchange /home/vmuser/nbgrader/exchange nfs defaults 0 0" | sudo tee -a /etc/fstab

echo "✅ Student nbgrader configured for $STUDENT_NAME"
ENDSSH

echo "✅ Setup complete for student $STUDENT_NAME on $STUDENT_IP"
