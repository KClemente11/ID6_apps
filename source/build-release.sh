#!/bin/bash
set -euo pipefail
cd "$(dirname "$0")"
python3 tools/stage_payload.py --check
export CGO_ENABLED=0
build="$(mktemp -d)"
trap 'rm -rf "$build"' EXIT
dest="${1:-$PWD/dist}"
mkdir -p "$dest"
dest="$(cd "$dest" && pwd)"
app="$build/NBT Evo Gauge Installer.app"
mkdir -p "$app/Contents/MacOS" "$app/Contents/Resources"
GOOS=darwin GOARCH=arm64 go build -trimpath -ldflags='-s -w' -o "$build/darwin-arm64" .
GOOS=darwin GOARCH=amd64 go build -trimpath -ldflags='-s -w' -o "$build/darwin-amd64" .
lipo -create "$build/darwin-arm64" "$build/darwin-amd64" -output "$app/Contents/MacOS/NBTEvoGauges"
cp Info.plist "$app/Contents/Info.plist"
cp -R third-party "$app/Contents/Resources/"
cp README.md "$app/Contents/Resources/Quick Start.md"
codesign --force --sign - "$app"
codesign --verify --strict "$app"
ditto -c -k --keepParent "$app" "$dest/NBT-Evo-Gauges-macOS-universal.zip"
for arch in amd64; do
  stage="$build/windows-$arch"
  mkdir -p "$stage"
  GOOS=windows GOARCH="$arch" go build -trimpath -ldflags='-s -w -H windowsgui' -o "$stage/NBT Evo Gauge Installer.exe" .
  cp README.md "$stage/Quick Start.md"
  cp -R third-party "$stage/"
  ditto -c -k "$stage" "$dest/NBT-Evo-Gauges-Windows-x64.zip"
done
(cd "$dest" && shasum -a 256 NBT-Evo-Gauges-*.zip > SHA256SUMS.txt)
