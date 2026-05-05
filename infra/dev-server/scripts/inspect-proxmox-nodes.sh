#!/usr/bin/env bash

set -euo pipefail

NODES=("${@:-root@172.16.0.202 root@172.16.0.203 root@172.16.0.204}")

if [[ $# -eq 0 ]]; then
  NODES=(root@172.16.0.202 root@172.16.0.203 root@172.16.0.204)
fi

for node in "${NODES[@]}"; do
  echo "== ${node} =="
  ssh -o BatchMode=yes -o ConnectTimeout=5 "${node}" '
    echo "hostname=$(hostname)"
    echo "cpu_count=$(nproc)"
    awk "/MemTotal|MemAvailable/ {printf \"%s=%0.1fGiB\n\", $1, $2/1024/1024}" /proc/meminfo
    echo "-- filesystems --"
    df -h --output=source,size,used,avail,pcent,target | grep -E "Filesystem| /$|/var/lib/vz|/mnt|/rpool|/tank|/data" || true
    echo "-- proxmox storage --"
    pvesm status 2>/dev/null || true
    echo "-- lxc --"
    pct list 2>/dev/null || true
    echo "-- qemu --"
    qm list 2>/dev/null || true
  '
  echo
done
