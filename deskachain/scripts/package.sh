#!/usr/bin/env sh
set -eu

VERSION="v0.4.10-testnet-rc1"
SKIP=0
for arg in "$@"; do
  case "$arg" in
    --skip-tests|-SkipTests)
      SKIP=1
      ;;
    *)
      VERSION="$arg"
      ;;
  esac
done
ROOT="$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)"
DIST="$ROOT/dist"
RELEASE_DIR="$DIST/releases"

if [ "${SKIP_TESTS:-0}" = "1" ] || [ "$SKIP" = "1" ]; then
  sh "$ROOT/scripts/build.sh" "$VERSION" --skip-tests
else
  sh "$ROOT/scripts/build.sh" "$VERSION"
fi

rm -rf "$RELEASE_DIR"
mkdir -p "$RELEASE_DIR"

for target in windows-amd64 linux-amd64 linux-arm64; do
  src="$DIST/$target"
  stage="$RELEASE_DIR/stage-$target"
  [ -d "$src" ] || { echo "missing build output: $src" >&2; exit 1; }
  rm -rf "$stage"
  mkdir -p "$stage/examples" "$stage/docs" "$stage/config" "$stage/scripts"
  cp -R "$src"/. "$stage/"
  cp "$ROOT/docs/Release.md" "$stage/QUICKSTART.md"
  cp "$ROOT/docs/Release.md" "$stage/docs/"
  cp "$ROOT/docs/Announcement-v0.4.6-testnet-rc1.md" "$stage/docs/"
  cp "$ROOT/docs/DeployTestnet.md" "$stage/docs/"
  cp "$ROOT/docs/DifficultyObservation.md" "$stage/docs/"
  cp "$ROOT/docs/Explorer.md" "$stage/docs/"
  cp "$ROOT/docs/Faucet.md" "$stage/docs/"
  cp "$ROOT/docs/FeedbackSummaryTemplate.md" "$stage/docs/"
  cp "$ROOT/docs/GitHubLabels.md" "$stage/docs/"
  cp "$ROOT/docs/GitHubReleaseChecklist.md" "$stage/docs/"
  cp "$ROOT/docs/GitHubRelease-v0.4.6-testnet-rc1.md" "$stage/docs/"
  cp "$ROOT/docs/IssueTriage.md" "$stage/docs/"
  cp "$ROOT/docs/KnownIssues.md" "$stage/docs/"
  cp "$ROOT/docs/LongRunTestnet.md" "$stage/docs/"
  cp "$ROOT/docs/Mining.md" "$stage/docs/"
  cp "$ROOT/docs/MiningStability.md" "$stage/docs/"
  cp "$ROOT/docs/MultiHostTestnet.md" "$stage/docs/"
  cp "$ROOT/docs/OperatorChecklist.md" "$stage/docs/"
  cp "$ROOT/docs/PostReleaseMonitoring.md" "$stage/docs/"
  cp "$ROOT/docs/Preflight.md" "$stage/docs/"
  cp "$ROOT/docs/PublicTestnetQuickstart.md" "$stage/docs/"
  cp "$ROOT/docs/RC2Planning.md" "$stage/docs/"
  cp "$ROOT/docs/ReleaseNotes-v0.4.6-testnet-rc1.md" "$stage/docs/"
  cp "$ROOT/docs/SeedOperatorPublish.md" "$stage/docs/"
  cp "$ROOT/docs/SeedMonitoringChecklist.md" "$stage/docs/"
  cp "$ROOT/docs/ServiceNode.md" "$stage/docs/"
  cp "$ROOT/docs/Staking.md" "$stage/docs/"
  cp "$ROOT/docs/Systemd.md" "$stage/docs/"
  cp "$ROOT/docs/TesterOnboarding.md" "$stage/docs/"
  cp "$ROOT/docs/TesterResponseSnippets.md" "$stage/docs/"
  cp "$ROOT/docs/TestnetFeedbackChecklist.md" "$stage/docs/"
  cp "$ROOT/docs/TestnetGenesis.md" "$stage/docs/"
  cp "$ROOT/docs/TestnetTopology.md" "$stage/docs/"
  cp "$ROOT/scripts/testnet-health.ps1" "$stage/scripts/"
  cp "$ROOT/scripts/testnet-health.sh" "$stage/scripts/"
  cp "$ROOT/config/testnet-seeds.example.txt" "$stage/config/"
  cp "$ROOT/README.md" "$stage/"
  [ -f "$ROOT/README-ID.md" ] && cp "$ROOT/README-ID.md" "$stage/"
  [ -f "$ROOT/LICENSE" ] && cp "$ROOT/LICENSE" "$stage/"
  cp -R "$ROOT/examples/systemd" "$stage/examples/"
  cp -R "$ROOT/examples/testnet" "$stage/examples/"

  if find "$stage" \( -name wallets.json -o -name chain.db -o -name mempool.json -o -name peers.json -o -name node.lock -o -name node_id -o -name faucet_state.json -o -name dkcservice-state.json -o -name dkcservice-states.json -o -name '*.key' -o -name '*.pem' \) | grep . >/dev/null; then
    echo "refusing to package runtime/private state" >&2
    exit 1
  fi
  if find "$stage" -type d \( -name testdata -o -name data -o -name wallets -o -name .git -o -name dist \) | grep . >/dev/null; then
    echo "refusing to package runtime/private directory" >&2
    exit 1
  fi

  if [ "$target" = "windows-amd64" ]; then
    archive="$RELEASE_DIR/deskachain-$VERSION-$target.zip"
    if command -v zip >/dev/null 2>&1; then
      (cd "$stage" && zip -qr "$archive" .)
    elif command -v python3 >/dev/null 2>&1; then
      (cd "$stage" && python3 - "$archive" <<'PY'
import os, sys, zipfile
archive = sys.argv[1]
with zipfile.ZipFile(archive, "w", zipfile.ZIP_DEFLATED) as zf:
    for root, _, files in os.walk("."):
        for name in files:
            path = os.path.join(root, name)
            zf.write(path, path[2:] if path.startswith("./") else path)
PY
)
    else
      echo "zip or python3 is required for windows archive" >&2
      exit 1
    fi
  else
    archive="$RELEASE_DIR/deskachain-$VERSION-$target.tar.gz"
    (cd "$stage" && tar -czf "$archive" .)
  fi
  if [ "$target" = "windows-amd64" ]; then
    contents="$(mktemp)"
    if command -v unzip >/dev/null 2>&1; then
      unzip -Z1 "$archive" > "$contents"
    else
      python3 - "$archive" > "$contents" <<'PY'
import sys, zipfile
with zipfile.ZipFile(sys.argv[1]) as zf:
    for name in zf.namelist():
        print(name)
PY
    fi
  else
    contents="$(mktemp)"
    tar -tzf "$archive" > "$contents"
  fi
  if grep -E '(^|/)(testdata|data|wallets|\.git|dist)(/|$)|(^|/)(wallets\.json|chain\.db|mempool\.json|peers\.json|node\.lock|node_id|faucet_state\.json|dkcservice-state\.json|dkcservice-states\.json)|\.(key|pem)$' "$contents" >/dev/null; then
    echo "refusing archive with runtime/private contents: $archive" >&2
    cat "$contents" >&2
    rm -f "$contents"
    exit 1
  fi
  if grep -E '\.env$' "$contents" | grep -Ev '^(./)?examples/(systemd|testnet)/' >/dev/null; then
    echo "refusing archive with non-example env file: $archive" >&2
    cat "$contents" >&2
    rm -f "$contents"
    exit 1
  fi
  printf 'contents %s\n' "$archive"
  sed -n '1,40p' "$contents"
  rm -f "$contents"
  rm -rf "$stage"
  printf 'packaged %s\n' "$archive"
done

checksums="$RELEASE_DIR/SHA256SUMS.txt"
rm -f "$checksums"
for file in "$RELEASE_DIR"/*; do
  [ -f "$file" ] || continue
  [ "$(basename "$file")" = "SHA256SUMS.txt" ] && continue
  if command -v sha256sum >/dev/null 2>&1; then
    (cd "$RELEASE_DIR" && sha256sum "$(basename "$file")") >> "$checksums"
  else
    hash="$(shasum -a 256 "$file" | awk '{print $1}')"
    printf '%s  %s\n' "$hash" "$(basename "$file")" >> "$checksums"
  fi
done
printf 'checksums %s\n' "$checksums"
