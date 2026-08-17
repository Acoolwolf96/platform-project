#!/usr/bin/env bash
set -euo pipefail

TEMPLATE_ID=9000
TEMPLATE_NAME="ubuntu-2404-cloudinit"
STORAGE="local-lvm"              # confirm this matches `pvesm status` before running
IMAGE_URL="https://cloud-images.ubuntu.com/noble/current/noble-server-cloudimg-amd64.img"
IMAGE_FILE="noble-server-cloudimg-amd64.img"

wget -q "$IMAGE_URL" -O "$IMAGE_FILE"

qm create "$TEMPLATE_ID" \
  --name "$TEMPLATE_NAME" \
  --memory 1024 \
  --cores 1 \
  --net0 virtio,bridge=vmbr0

qm importdisk "$TEMPLATE_ID" "$IMAGE_FILE" "$STORAGE"
qm set "$TEMPLATE_ID" --scsihw virtio-scsi-pci --scsi0 "$STORAGE:vm-${TEMPLATE_ID}-disk-0"
qm set "$TEMPLATE_ID" --ide2 "$STORAGE:cloudinit"
qm set "$TEMPLATE_ID" --boot order=scsi0
qm set "$TEMPLATE_ID" --agent enabled=1
qm template "$TEMPLATE_ID"

echo "Template $TEMPLATE_ID ($TEMPLATE_NAME) is ready on storage '$STORAGE'."
