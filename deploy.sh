#!/usr/bin/env bash
set -euo pipefail

# Usage: ./deploy.sh <app_name> <vps_user>@<vps_host> <remote_dir>
# Example: ./deploy.sh myapp root@1.2.3.4 /srv/myapp

APP_NAME=$1
VPS=$2
REMOTE_DIR=$3
IMAGE_NAME="${APP_NAME}:latest"
TAR_FILE="${APP_NAME}.tar.gz"
COMPOSE_FILE="compose.yaml"

# Step 1: Build Docker image locally
echo "Building Docker image ${IMAGE_NAME}..."
docker build -t "${IMAGE_NAME}" .

# Step 2: Save Docker image to tarball
echo "Saving Docker image to ${TAR_FILE}..."
docker save "${IMAGE_NAME}" | gzip > "${TAR_FILE}"

# Step 3: Copy image tarball and compose.yaml to VPS
echo "Copying image and compose.yaml to ${VPS}:${REMOTE_DIR}..."
scp "${TAR_FILE}" "${COMPOSE_FILE}" "${VPS}:${REMOTE_DIR}/"

# Step 4: Load image on VPS and clean up tarball
echo "Loading image on VPS..."
ssh "${VPS}" bash -c "'
cd ${REMOTE_DIR}
docker load < ${TAR_FILE}
rm -f ${TAR_FILE}
'"

# Step 5: Restart systemd service
echo "Restarting systemd service ${APP_NAME}.service..."
ssh "${VPS}" "sudo systemctl restart ${APP_NAME}.service"

echo "✅ Deployment complete! Service ${APP_NAME} should now be running."
