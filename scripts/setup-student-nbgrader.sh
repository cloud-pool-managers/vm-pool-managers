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

# Create student assignment directories
mkdir -p /home/vmuser/nbgrader/{assignments,submitted}

# Configure nbgrader in student mode
mkdir -p /home/vmuser/.jupyter
cat > /home/vmuser/nbgrader/nbgrader_config.py << 'NBCFG'
c = get_config()
c.CourseDirectory.root = '/home/vmuser/nbgrader'
c.CourseDirectory.course_id = 'course'
NBCFG

# Enable only the assignment list extension (student side)
jupyter nbextension install --user --py nbgrader --quiet 2>/dev/null || true
jupyter nbextension enable --user --py nbgrader --quiet 2>/dev/null || true
jupyter serverextension enable --user --py nbgrader --quiet 2>/dev/null || true

# Create a systemd mount for submitted assignments (SFTP to instructor VM)
# This mounts instructor:/nbgrader/submitted/$STUDENT_NAME/ → ~/nbgrader/submitted/
# Requires rclone to be configured (done by the main rclone setup)
mkdir -p /home/vmuser/nbgrader/submitted/$STUDENT_NAME

echo "✅ Student nbgrader configured for $STUDENT_NAME"
ENDSSH

echo "✅ Setup complete for student $STUDENT_NAME on $STUDENT_IP"
