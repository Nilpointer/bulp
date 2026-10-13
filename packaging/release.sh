#!/usr/bin/env bash
# Builds every release file into build/release/: the Flatpak bundle, the
# AppImage, the bulp CLI for Linux and macOS (amd64/arm64) and SHA256SUMS.
# Needs flatpak + org.flatpak.Builder and podman (or docker); see README.
set -euo pipefail
cd "$(dirname "$0")/.."

out=build/release
rm -rf "$out"
mkdir -p "$out"

mise run flatpak-bundle
mise run appimage

for target in linux/amd64 linux/arm64 darwin/amd64 darwin/arm64; do
	os=${target%/*} arch=${target#*/}
	CGO_ENABLED=0 GOOS=$os GOARCH=$arch go build -trimpath -ldflags=-s -o "$out/bulp-$os-$arch" ./cmd/bulp
done

version=$(grep -om1 'release version="[^"]*"' packaging/de.codecauldron.Bulp.metainfo.xml | cut -d'"' -f2)
cp "build/Bulp-$version-x86_64.AppImage" build/de.codecauldron.Bulp.flatpak "$out/"
(cd "$out" && sha256sum -- * >SHA256SUMS)
ls -lh "$out"
