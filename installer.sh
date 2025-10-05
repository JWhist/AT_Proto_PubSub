#!/usr/bin/env bash
set -euo pipefail

# Usage: ./installer.sh <service_name> <working_dir> <compose_file>
# Example: ./installer.sh myapp /srv/myapp /srv/myapp/compose.yaml

SERVICE_NAME=$1
WORKING_DIR=$2
COMPOSE_FILE=$3
SYSTEMD_FILE="/etc/systemd/system/${SERVICE_NAME}.service"

# Ensure docker compose exists
if ! command -v docker compose >/dev/null 2>&1; then
  echo "Error: docker compose is not installed or not in PATH."
  exit 1
fi

# Ensure working dir exists
mkdir -p "$WORKING_DIR"

# Copy compose file if it exists locally and isn't already in place
if [[ -f "$COMPOSE_FILE" ]]; then
  echo "Compose file found: $COMPOSE_FILE"
else
  echo "Error: Compose file $COMPOSE_FILE does not exist."
  exit 1
fi

# Create systemd service file
cat > "$SYSTEMD_FILE" <<EOF
[Unit]
Description=${SERVICE_NAME} Docker Compose Service
Requires=docker.service
After=docker.service

[Service]
WorkingDirectory=${WORKING_DIR}
ExecStart=/usr/bin/docker compose -f ${COMPOSE_FILE} up -d
ExecStop=/usr/bin/docker compose -f ${COMPOSE_FILE} down
Restart=always
TimeoutStartSec=0

[Install]
WantedBy=multi-user.target
EOF

# Reload systemd and enable service
systemctl daemon-reload
systemctl enable "${SERVICE_NAME}.service"
systemctl start "${SERVICE_NAME}.service"

echo "✅ Service ${SERVICE_NAME} installed and started."
echo "Check status with: systemctl status ${SERVICE_NAME}.service"
