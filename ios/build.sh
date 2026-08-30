#!/bin/sh
# Build the GrMob xcframework with gomobile.
#
# Binds the `mobile` bridge package (the Swift runtime's call surface) plus
# an app package, whose init() registers the root view with the bridge.
# The app package is the first argument, defaulting to the demo app:
#   ios/build.sh ./examples/todoapp
# Any Go package whose init calls mobile.Register works (and which exports at
# least one bindable symbol; see mobileapp.AppName for why).
#
# Requires full Xcode (not just Command Line Tools): gomobile drives xcodebuild
# to assemble the xcframework. After installing Xcode, point the tools at it:
#   sudo xcode-select -s /Applications/Xcode.app
set -e

# gomobile/gobind live in GOPATH/bin, which isn't always on PATH.
PATH="$PATH:$(go env GOPATH)/bin"

if ! command -v gomobile >/dev/null; then
  echo "gomobile is required: go install golang.org/x/mobile/cmd/gomobile@latest golang.org/x/mobile/cmd/gobind@latest" >&2
  exit 1
fi
if ! xcodebuild -version >/dev/null 2>&1; then
  echo "full Xcode is required (xcodebuild not available); install it and run: sudo xcode-select -s /Applications/Xcode.app" >&2
  exit 1
fi

cd "$(dirname "$0")/.."
mkdir -p ios/Frameworks

# Which app package to bind alongside the bridge. Defaults to the demo app;
# pass another package path to ship a different app in the same shell.
APP_PKG="${1:-./examples/mobileapp}"

# Both slices so the same framework serves the simulator and real devices.
# The framework module is named from -o (GrMob); symbols inside carry
# per-package prefixes (Mobile*, plus one per bound app package).
gomobile bind -target=ios,iossimulator \
  -o ios/Frameworks/GrMob.xcframework \
  ./mobile "$APP_PKG"
