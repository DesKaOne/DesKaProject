#!/usr/bin/env bash
set -euo pipefail

root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$root/node"

printf '%s\n' 'Phase 10 readiness snapshot'
printf 'go=%s\n' "$(go version)"
printf 'module=%s\n' "$(awk '\/\\^module \/{print $2; exit}' go.mod)"\n
echo 'tests=go test ./...'
go test ./...

echo 'required_phase9_scripts'
for f in \
  "$root/scripts/testnet-smoke.sh" \
  "$root/scripts/testnet-health.sh" \
  "$root/scripts/testnet-observe.sh" \
  "$root/scripts/testnet-backup.sh" \
  "$root/scripts/build.sh" \
  "$root/scripts/package.sh" \
  "$root/scripts/smoke-test.sh"; do
  test -f "$f"
  printf 'present=%s\n' "$f"
done

echo 'required_phase10_docs'
for f in \
  "$root/docs/Phase10-PreMainnet-Mainnet-Readiness.md" \
  "$root/docs/Phase10-Evidence-Ledger.md" \
  "$root/docs/Phase10.1-Network-Identity-Genesis-Decision.md" \
  "$root/docs/Phase10.2-Production-Topology-Roles.md" \
  "$root/docs/Phase10.3-Release-Artifact-Control.md" \
  "$root/docs/Phase10.4-Security-Threat-Model-Gate.md" \
  "$root/docs/Phase10.5-Economic-Protocol-Assumptions.md" \
  "$root/docs/Phase10.6-Operational-Readiness.md" \
  "$root/docs/Phase10.7-External-Dependencies.md" \
  "$root/docs/Phase10.8-Mainnet-Release-Gate.md"; do
  test -f "$f"
  printf 'present=%s\n' "$f"
done

echo 'mainnet_placeholder_guard'
grep -Eq 'Mainnet network ID.*(TBD|pending approval)' "$root/docs/Phase10.1-Network-Identity-Genesis-Decision.md"
grep -Eq 'Mainnet chain ID.*(TBD|pending approval)' "$root/docs/Phase10.1-Network-Identity-Genesis-Decision.md"
grep -Eq 'Genesis artifact.*(TBD|pending)' "$root/docs/Phase10.1-Network-Identity-Genesis-Decision.md"

echo 'snapshot=PASS'
