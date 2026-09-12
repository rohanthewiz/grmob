// Command grmob starts and builds GrMob apps that live in their own module.
//
//	grmob new <dir>          scaffold an app: Go app package, browser host, build script
//	grmob doctor             which targets this machine can build, and what is missing
//	grmob android [-install] bind the app into the Android shell and assemble an APK
//	grmob ios [-open]        bind the app into the iOS shell and build it for the simulator
//
// Install or run it at the same version as the framework the app uses:
//
//	go run github.com/rohanthewiz/grmob/cmd/grmob@latest new myapp
//
// # Why a command and not a copy of this repository's scripts
//
// Everything in this repository that builds an app assumes it runs from the
// grmob module root: ./build.sh builds ./wasm, whose main.go dot-imports an
// example; android/build.sh and ios/build.sh bind ./examples/mobileapp; the
// shells sit beside them. An app in its own module could only copy those
// files, and a copy is pinned to whatever version it was read from. The
// failures that follow are quiet — a grmob-runtime.js from an older release
// that erases every core.Gap, a hand-patched runtime that the next version
// bump overwrites, a host page with no Shutdown so the dev server can only
// page-reload.
//
// So the scaffold keeps the rule "anything grmob owns comes from the grmob
// module in go.mod, at build time":
//
//	app module (yours)                     grmob module (go.mod pins it)
//	──────────────────                     ─────────────────────────────
//	app/app.go         ─── imports ──────▶ core, mobile, ...
//	wasm/main.go       ─── webhost.Run ──▶ webhost
//	wasm/index.html                        wasm/grmob-runtime.js ┐
//	build.sh           ─── copies ◀──────── wasm/camera.js        ┘ each build, if changed
//	dev server         ─── go run ───────▶ serve -dev
//	android/, ios/     ◀── vendored once ── android/, ios/  (re-vendor with -refresh)
//
// The one thing that is copied into the app and kept there is the native
// shells, because an app is expected to edit them (icons, permissions, its own
// name), and a record of the version they came from lets the build warn when
// go.mod has moved past it.
package main

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// grmobModule is this framework's module path: the dependency the scaffold
// adds, and the module whose directory the native commands copy shells from.
const grmobModule = "github.com/rohanthewiz/grmob"

// configFile marks an app's root and records the two identities the native
// shells need and Go code does not: the name a launcher shows and the
// platform application ID. JSON rather than a Go file because it is read by
// this command, not compiled into the app.
const configFile = "grmob.json"

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}
	var err error
	switch cmd, args := os.Args[1], os.Args[2:]; cmd {
	case "new":
		err = cmdNew(args)
	case "doctor":
		err = cmdDoctor(args)
	case "android":
		err = cmdAndroid(args)
	case "ios":
		err = cmdIOS(args)
	case "help", "-h", "-help", "--help":
		usage()
	default:
		fmt.Fprintf(os.Stderr, "grmob: unknown command %q\n\n", cmd)
		usage()
		os.Exit(2)
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "grmob: %v\n", err)
		os.Exit(1)
	}
}

func usage() {
	fmt.Fprint(os.Stderr, `usage: grmob <command> [flags]

  new <dir>     scaffold a GrMob app targeting the browser (WASM)
  doctor        report which targets this machine can build
  android       build the app for Android (needs Android SDK + NDK, JDK 17+)
  ios           build the app for iOS (needs Xcode + xcodegen, macOS only)

Run "grmob <command> -h" for a command's flags.
`)
}

// run executes a command with its output streamed to the terminal, so a long
// gomobile or Gradle step shows progress instead of going silent for minutes.
// env entries are appended to the current environment.
func run(dir string, env []string, name string, args ...string) error {
	fmt.Printf("  $ %s %s\n", name, strings.Join(args, " "))
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Env = append(os.Environ(), env...)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("%s %s: %w", name, strings.Join(args, " "), err)
	}
	return nil
}

// output executes a command and returns its trimmed stdout. On failure the
// error carries stderr, which is where go and xcodebuild put the explanation.
func output(dir string, name string, args ...string) (string, error) {
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	out, err := cmd.Output()
	if err != nil {
		var ee *exec.ExitError
		if errors.As(err, &ee) && len(ee.Stderr) > 0 {
			return "", fmt.Errorf("%s %s: %w\n%s", name, strings.Join(args, " "), err, strings.TrimSpace(string(ee.Stderr)))
		}
		return "", fmt.Errorf("%s %s: %w", name, strings.Join(args, " "), err)
	}
	return strings.TrimSpace(string(out)), nil
}

// findAppRoot walks up from the working directory to the directory holding
// grmob.json, so the native commands work from anywhere inside an app the way
// go commands work from anywhere inside a module.
func findAppRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, configFile)); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("no %s in this directory or any parent; run this inside an app made by `grmob new`", configFile)
		}
		dir = parent
	}
}

// grmobDir is the directory of the grmob module the app at root resolves —
// the module cache for a normal require, the checkout for a replace. It is
// where the runtime JS and the native shells are read from, which is what
// keeps them at the same version as the Go code the app compiles against.
func grmobDir(root string) (string, error) {
	return output(root, "go", "list", "-m", "-f", "{{.Dir}}", grmobModule)
}

// grmobVersion identifies the resolved grmob module for the vendored-shell
// record: the version for a require, "replace <dir>" for a local checkout,
// where there is no version and the directory is the identity.
func grmobVersion(root string) (string, error) {
	return output(root, "go", "list", "-m", "-f", "{{if .Replace}}replace {{.Replace.Path}}{{else}}{{.Version}}{{end}}", grmobModule)
}
