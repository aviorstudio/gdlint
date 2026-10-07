#!/usr/bin/env bash
export GOTOOLCHAIN=local
set -euo pipefail
VERSION="${VERSION:-dev}"
COMMIT="${COMMIT:-$(git rev-parse HEAD)}"
DATE="${DATE:-$(date -u +%Y-%m-%dT%H:%M:%SZ)}"
mkdir -p dist

build() {
  goos="$1"
  goarch="$2"
  os_label="$3"
  arch_label="$4"
  archive_ext="$5"
  exe=""
  if [ "$goos" = "windows" ]; then
    exe=".exe"
  fi

  name="gdlint_${os_label}_${arch_label}"
  out_dir="dist/$name"
  mkdir -p "$out_dir"
  out_path="$PWD/$out_dir/gdlint$exe"
  GOOS="$goos" GOARCH="$goarch" CGO_ENABLED=0 go build -trimpath \
    -ldflags "-s -w -X main.version=$VERSION -X main.commit=$COMMIT -X main.date=$DATE" \
    -o "$out_path" .
  cp README.md "$out_dir/README.md"

  if [ "$archive_ext" = "zip" ]; then
    (cd "$out_dir" && zip -q "../$name.zip" "gdlint$exe" README.md)
  else
    tar -C "$out_dir" -czf "dist/$name.tar.gz" "gdlint$exe" README.md
  fi
}

build linux amd64 Linux x86_64 tar.gz
build linux arm64 Linux arm64 tar.gz
build darwin amd64 Darwin x86_64 tar.gz
build darwin arm64 Darwin arm64 tar.gz
build windows amd64 Windows x86_64 zip
build windows arm64 Windows arm64 zip

python3 scripts/check_archives.py --record
