#!/bin/sh
# Build the GrMob AAR with gomobile.
#
# Binds the `mobile` bridge package (the Kotlin runtime's call surface) plus
# an app package, whose init() registers the root view with the bridge.
# The app package is the first argument, defaulting to the demo app:
#   android/build.sh ./examples/todoapp
# Any Go package whose init calls mobile.Register works.
set -e

# gomobile/gobind live in GOPATH/bin, which isn't always on PATH.
PATH="$PATH:$(go env GOPATH)/bin"

if ! command -v gomobile >/dev/null; then
  echo "gomobile is required: go install golang.org/x/mobile/cmd/gomobile@latest golang.org/x/mobile/cmd/gobind@latest" >&2
  exit 1
fi

# gomobile locates the SDK/NDK through these; default to the standard macOS
# install location when the caller hasn't exported them.
[ -n "$ANDROID_HOME" ] || export ANDROID_HOME="$HOME/Library/Android/sdk"
if [ -z "$ANDROID_NDK_HOME" ] && [ -d "$ANDROID_HOME/ndk" ]; then
  export ANDROID_NDK_HOME="$ANDROID_HOME/ndk/$(ls "$ANDROID_HOME/ndk" | sort -V | tail -1)"
fi

cd "$(dirname "$0")/.."
mkdir -p android/app/libs

# Which app package to bind alongside the bridge. Defaults to the demo app;
# pass another package path to ship a different app in the same shell.
APP_PKG="${1:-./examples/mobileapp}"

# LDFLAGS is handed to the Go linker unchanged, which is how an app bakes in
# build-time configuration (a server URL, a build tag) via -X without needing
# a config file in the APK. Empty means no -ldflags argument at all: gomobile
# rejects an empty string there.
#   LDFLAGS="-X example.com/app/internal/api.DefaultBaseURL=http://localhost:8000" android/build.sh ./app
gomobile bind -target=android -androidapi 24 \
  -o android/app/libs/grmob.aar \
  ${LDFLAGS:+-ldflags "$LDFLAGS"} \
  ./mobile "$APP_PKG"
