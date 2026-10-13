#!/usr/bin/env bash
# Builds build/Bulp-<version>-x86_64.AppImage. Needs podman (or docker)
# and network access for the first run (container image, appimagetool).
set -euo pipefail
cd "$(dirname "$0")/../.."

id=de.codecauldron.Bulp
version=$(grep -om1 'release version="[^"]*"' packaging/$id.metainfo.xml | cut -d'"' -f2)
out=build/appimage
appdir=$out/AppDir
tools=$out/tools
engine=$(command -v podman || command -v docker)
image=localhost/bulp-appimage-builder

rm -rf "$appdir"
mkdir -p "$appdir/usr/bin" "$tools"

# Static fallback: runs anywhere, opens the panel in a browser.
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -o "$appdir/usr/bin/bulp-menubar-static" ./cmd/bulp-menubar

# GTK build, linked against Ubuntu 24.04's libraries with the host's Go toolchain.
"$engine" build -q -t "$image" packaging/appimage >/dev/null
"$engine" run --rm --security-opt label=disable \
	-v "$(go env GOROOT)":/usr/local/go:ro \
	-v "$(go env GOMODCACHE)":/go/pkg/mod:ro \
	-v "$PWD":/src -w /src \
	-e GOMODCACHE=/go/pkg/mod -e GOCACHE=/tmp/gocache -e GOPROXY=off -e GOTOOLCHAIN=local -e GOFLAGS=-mod=mod \
	"$image" /usr/local/go/bin/go build -trimpath -o "$appdir/usr/bin/bulp-menubar" ./cmd/bulp-menubar

install -m755 packaging/appimage/AppRun "$appdir/AppRun"
install -m644 packaging/$id.desktop packaging/$id.svg "$appdir/"
ln -sf $id.svg "$appdir/.DirIcon"
install -Dm644 packaging/$id.metainfo.xml "$appdir/usr/share/metainfo/$id.metainfo.xml"

# appimagetool is itself an AppImage; fetched once into its own directory.
tool=$tools/appimagetool-x86_64.AppImage
if [[ ! -x $tool ]]; then
	curl -fsSL -o "$tool" https://github.com/AppImage/appimagetool/releases/download/continuous/appimagetool-x86_64.AppImage
	chmod +x "$tool"
fi
ARCH=x86_64 VERSION=$version "$tool" --no-appstream "$appdir" "build/Bulp-$version-x86_64.AppImage"
