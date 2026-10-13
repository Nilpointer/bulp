#!/usr/bin/env bash
# Builds build/Elgato_Lights-<version>-macOS.zip: a universal (arm64 + x86_64)
# "Elgato Lights.app". Runs on macOS with Go and the Xcode command line tools.
# The app is only ad-hoc signed (no Apple Developer ID, not notarized).
set -euo pipefail
cd "$(dirname "$0")/../.."

version=$(grep -om1 'release version="[^"]*"' packaging/de.codecauldron.Elg.metainfo.xml | cut -d'"' -f2)
app="build/Elgato Lights.app"
rm -rf "$app" build/AppIcon.iconset
mkdir -p "$app/Contents/MacOS" "$app/Contents/Resources" build/AppIcon.iconset

export CGO_ENABLED=1 MACOSX_DEPLOYMENT_TARGET=12.0
GOARCH=arm64 CC="clang -arch arm64" go build -trimpath -o build/elg-menubar-arm64 ./cmd/elg-menubar
GOARCH=amd64 CC="clang -arch x86_64" go build -trimpath -o build/elg-menubar-amd64 ./cmd/elg-menubar
lipo -create -output "$app/Contents/MacOS/elg-menubar" build/elg-menubar-arm64 build/elg-menubar-amd64

sed "s/@VERSION@/$version/g" packaging/macos/Info.plist >"$app/Contents/Info.plist"
for s in 16 32 128 256 512; do
	sips -z $s $s packaging/macos/AppIcon.png --out build/AppIcon.iconset/icon_${s}x${s}.png >/dev/null
	sips -z $((s * 2)) $((s * 2)) packaging/macos/AppIcon.png --out build/AppIcon.iconset/icon_${s}x${s}@2x.png >/dev/null
done
iconutil -c icns build/AppIcon.iconset -o "$app/Contents/Resources/AppIcon.icns"

codesign --force --sign - "$app"
ditto -c -k --keepParent "$app" "build/Elgato_Lights-$version-macOS.zip"
echo "build/Elgato_Lights-$version-macOS.zip"
