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
COMMIT="unknown"
if command -v git >/dev/null 2>&1; then
  COMMIT="$(git -C "$ROOT" rev-parse --short HEAD 2>/dev/null || printf unknown)"
fi
BUILD_DATE="$(date -u +%Y-%m-%dT%H:%M:%SZ)"
LDFLAGS="-s -w -X deskachain/internal/version.Version=$VERSION -X deskachain/internal/version.Commit=$COMMIT -X deskachain/internal/version.BuildDate=$BUILD_DATE"

if [ "${SKIP_TESTS:-0}" != "1" ] && [ "$SKIP" != "1" ]; then
  (cd "$ROOT" && go test ./node/...)
fi

rm -rf "$DIST"
mkdir -p "$DIST"

build_target() {
  goos="$1"
  goarch="$2"
  dir="$3"
  ext="$4"
  outdir="$DIST/$dir"
  mkdir -p "$outdir"
  (cd "$ROOT" && GOOS="$goos" GOARCH="$goarch" go build -trimpath -ldflags "$LDFLAGS" -o "$outdir/deskachain$ext" ./node/cmd/deskachain)
  (cd "$ROOT" && GOOS="$goos" GOARCH="$goarch" go build -trimpath -ldflags "$LDFLAGS" -o "$outdir/idrminer$ext" ./node/cmd/idrminer)
  (cd "$ROOT" && GOOS="$goos" GOARCH="$goarch" go build -trimpath -ldflags "$LDFLAGS" -o "$outdir/idrservice$ext" ./node/cmd/idrservice)
  printf 'built %s\n' "$outdir"
}

build_target windows amd64 windows-amd64 .exe
build_target linux amd64 linux-amd64 ""
build_target linux arm64 linux-arm64 ""

printf 'release build complete: %s\n' "$DIST"
