#!/usr/bin/env bash

set -euo pipefail

: "${PROXMOX_HOST:=root@172.16.0.204}"
: "${VMID:=128}"
: "${NAME:=sally-dev-vm}"
: "${IMAGE_URL:=https://cloud-images.ubuntu.com/noble/current/noble-server-cloudimg-amd64.img}"
: "${IMAGE_FILE:=noble-server-cloudimg-amd64.img}"
: "${IMAGE_STORAGE_PATH:=/var/lib/vz/template/iso}"
: "${STORAGE:=tank2}"
: "${DISK_SIZE:=32G}"
: "${CORES:=2}"
: "${MEMORY_MB:=4096}"
: "${BRIDGE:=vmbr0}"
: "${IP_CONFIG:=dhcp}"
: "${CI_USER:=ubuntu}"
: "${CI_UPGRADE:=0}"
: "${START:=1}"
: "${ONBOOT:=1}"
: "${SSH_PUBLIC_KEY_FILE:=${HOME}/.ssh/id_rsa.pub}"

remote_image_path="${IMAGE_STORAGE_PATH}/${IMAGE_FILE}"
remote_ssh_key_path="/tmp/sally-${VMID}-authorized.pub"

if [[ -f "${SSH_PUBLIC_KEY_FILE}" ]]; then
  scp -q "${SSH_PUBLIC_KEY_FILE}" "${PROXMOX_HOST}:${remote_ssh_key_path}"
else
  echo "SSH public key not found at ${SSH_PUBLIC_KEY_FILE}" >&2
  exit 1
fi

ssh "${PROXMOX_HOST}" "mkdir -p '${IMAGE_STORAGE_PATH}' && test -f '${remote_image_path}' || wget -O '${remote_image_path}' '${IMAGE_URL}'"

if ssh "${PROXMOX_HOST}" "qm config '${VMID}' >/dev/null 2>&1"; then
  echo "VM ${VMID} already exists on ${PROXMOX_HOST}; applying desired config"
else
  ssh "${PROXMOX_HOST}" qm create "${VMID}" \
    --name "${NAME}" \
    --ostype l26 \
    --machine q35 \
    --bios seabios \
    --agent enabled=1 \
    --cores "${CORES}" \
    --memory "${MEMORY_MB}" \
    --net0 "virtio,bridge=${BRIDGE},firewall=1" \
    --serial0 socket \
    --vga serial0 \
    --onboot "${ONBOOT}"

  ssh "${PROXMOX_HOST}" qm importdisk "${VMID}" "${remote_image_path}" "${STORAGE}"
  imported_disk="$(
    ssh "${PROXMOX_HOST}" "qm config '${VMID}' | sed -n 's/^unused0: //p'"
  )"
  if [[ -z "${imported_disk}" ]]; then
    echo "Could not find imported disk for VM ${VMID}" >&2
    exit 1
  fi

  ssh "${PROXMOX_HOST}" qm set "${VMID}" \
    --scsihw virtio-scsi-pci \
    --scsi0 "${imported_disk},discard=on" \
    --boot order=scsi0 \
    --ide2 "${STORAGE}:cloudinit"

  ssh "${PROXMOX_HOST}" qm resize "${VMID}" scsi0 "${DISK_SIZE}"
fi

ssh "${PROXMOX_HOST}" qm set "${VMID}" \
  --name "${NAME}" \
  --agent enabled=1 \
  --cores "${CORES}" \
  --memory "${MEMORY_MB}" \
  --net0 "virtio,bridge=${BRIDGE},firewall=1" \
  --ipconfig0 "ip=${IP_CONFIG}" \
  --ciuser "${CI_USER}" \
  --ciupgrade "${CI_UPGRADE}" \
  --sshkeys "${remote_ssh_key_path}" \
  --onboot "${ONBOOT}"

if [[ "${START}" == "1" ]]; then
  ssh "${PROXMOX_HOST}" "qm status '${VMID}' | grep -q 'status: running' || qm start '${VMID}'"
fi

echo "VM ${VMID} (${NAME}) is converged on ${PROXMOX_HOST}"
echo "Check its address with:"
echo "  ssh ${PROXMOX_HOST} \"qm guest cmd ${VMID} network-get-interfaces\""
