package main

import (
	"errors"
	"flag"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// The native targets.
//
// Both follow the same five steps, and the order is chosen so that the fast,
// certain failures come before the slow ones:
//
//	1. prerequisites   doctor's table for the target; stop with its remedies
//	2. shell           copy grmob's android/ or ios/ into the app, once
//	3. gomobile        build gomobile + gobind from the app's module graph
//	4. bind            gomobile bind grmob/mobile + ./app → .aar / .xcframework
//	5. platform build  Gradle assembleDebug / xcodegen + xcodebuild
//
// Step 2 is the only one that writes files the app owns. The shells are
// copied rather than built in place because an app has to be able to change
// them — its name, its icon, the permissions it declares — and the module
// cache is read-only. The copy records which grmob it came from; a build
// against a different grmob warns, because the Kotlin and Swift renderers
// apply the Go side's patch format and a mismatch shows up as layout that is
// subtly wrong rather than as an error.

// shellRecord is the file, inside a vendored shell, naming the grmob it was
// copied from.
const shellRecord = ".grmob-shell"

// shellSpec describes one platform shell in the grmob module.
type shellSpec struct {
	dir string // "android" or "ios", both in grmob and in the app

	// skip lists slash-separated paths, relative to dir, that are not copied:
	// build state that exists in a working checkout (a -replace points at
	// one), the repository's own verification harnesses, and the shell's
	// build.sh, which binds grmob's demo app from grmob's root and would be
	// wrong in an app.
	skip []string

	// executable lists files that must be runnable. The module cache stores
	// every file read-only and without an execute bit, so modes are restated
	// here rather than copied.
	executable []string
}

var androidShell = shellSpec{
	dir:        "android",
	skip:       []string{".gradle", ".kotlin", ".idea", "build", "app/build", "app/libs", "local.properties", "verify", "device", "build.sh"},
	executable: []string{"gradlew"},
}

var iosShell = shellSpec{
	dir:  "ios",
	skip: []string{"build", "Frameworks", "GrMobApp.xcodeproj", "DerivedData", "xcuserdata", "verify", "build.sh", "GrMob/Info.plist"},
}

// patch rewrites one file of a freshly copied shell. Each is anchored on a
// literal that must occur exactly once, and fails rather than guessing when it
// does not: a shell whose anchor moved is a grmob release this command was not
// written against, and TestShellPatchesStillApply holds the anchors to grmob's
// own shells so that is caught in this repository, not on someone's build.
type patch struct {
	file  string // relative to the shell dir
	apply func(string) (string, error)
}

// replaceOnce returns a patch function replacing the single occurrence of old.
func replaceOnce(old, new string) func(string) (string, error) {
	return func(s string) (string, error) {
		if n := strings.Count(s, old); n != 1 {
			return "", fmt.Errorf("expected exactly one %q, found %d", old, n)
		}
		return strings.Replace(s, old, new, 1), nil
	}
}

// insertLineBefore returns a patch function inserting line above the single
// line containing anchor, at that line's indentation — which is how a key is
// added to a YAML mapping without parsing YAML.
func insertLineBefore(anchor, line string) func(string) (string, error) {
	return func(s string) (string, error) {
		lines := strings.Split(s, "\n")
		at := -1
		for i, l := range lines {
			if strings.Contains(l, anchor) {
				if at >= 0 {
					return "", fmt.Errorf("expected one line containing %q, found several", anchor)
				}
				at = i
			}
		}
		if at < 0 {
			return "", fmt.Errorf("no line contains %q", anchor)
		}
		indent := lines[at][:len(lines[at])-len(strings.TrimLeft(lines[at], " \t"))]
		lines = append(lines[:at], append([]string{indent + line}, lines[at:]...)...)
		return strings.Join(lines, "\n"), nil
	}
}

// androidPatches set the app's identity in the Android shell. Only the
// applicationId changes: the Gradle namespace (and the Kotlin package) stays
// com.grmob.app, because it names the shell's own classes, not the app, and
// Android allows the two to differ.
func androidPatches(cfg appConfig) []patch {
	return []patch{
		{"app/build.gradle", replaceOnce(`applicationId = "com.grmob.app"`, `applicationId = "`+cfg.ID+`"`)},
		{"app/src/main/AndroidManifest.xml", replaceOnce(`android:label="GrMob"`, `android:label="`+xmlEscape(cfg.Name)+`"`)},
	}
}

// iosPatches set the app's identity in project.yml. The target stays
// GrMobApp (it names the Xcode target and scheme this command builds); the
// launcher label is CFBundleDisplayName, which is what iOS shows under the
// icon.
func iosPatches(cfg appConfig) []patch {
	return []patch{
		{"project.yml", replaceOnce("PRODUCT_BUNDLE_IDENTIFIER: com.grmob.demo", "PRODUCT_BUNDLE_IDENTIFIER: "+cfg.ID)},
		// strconv.Quote output is a valid YAML double-quoted scalar.
		{"project.yml", insertLineBefore("UILaunchScreen: {}", "CFBundleDisplayName: "+strconv.Quote(cfg.Name))},
	}
}

func xmlEscape(s string) string {
	return strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;", `"`, "&quot;", "'", "&apos;").Replace(s)
}

// --- android ---------------------------------------------------------------

func cmdAndroid(args []string) error {
	fset := flag.NewFlagSet("android", flag.ContinueOnError)
	install := fset.Bool("install", false, "install the APK on the connected device or emulator and launch it")
	refresh := fset.Bool("refresh", false, "copy grmob's Android shell over android/ again (overwrites edits to copied files)")
	if err := fset.Parse(args); err != nil {
		return err
	}

	root, cfg, err := appContext()
	if err != nil {
		return err
	}

	fmt.Println("Checking Android prerequisites…")
	checks, env := androidProbe()
	if err := blocking(checks); err != nil {
		return fmt.Errorf("the Android target is not ready on this machine:\n%v\n\n(`grmob doctor` shows every target)", err)
	}
	adb := filepath.Join(env.sdk, "platform-tools", "adb")
	if *install && !fileExists(adb) {
		return errors.New("-install needs adb: install \"Android SDK Platform-Tools\" from the SDK Manager")
	}

	shell := filepath.Join(root, androidShell.dir)
	if err := vendorShell(root, androidShell, androidPatches(cfg), *refresh); err != nil {
		return err
	}

	path, err := ensureGomobile(root)
	if err != nil {
		return err
	}

	fmt.Println("Binding the app (gomobile, Android)…")
	aar := filepath.Join(shell, "app", "libs", "grmob.aar")
	if err := os.MkdirAll(filepath.Dir(aar), 0o755); err != nil {
		return err
	}
	bind := []string{"bind", "-target=android", "-androidapi", "24", "-o", aar}
	bind = append(bind, ldflags()...)
	bind = append(bind, grmobModule+"/mobile", "./app")
	if err := run(root, append(env.env(), path), "gomobile", bind...); err != nil {
		return err
	}

	fmt.Println("Assembling the APK (Gradle; the first run downloads Gradle and the Android dependencies)…")
	// `sh gradlew` rather than ./gradlew: it runs even where the execute bit
	// was lost, e.g. a shell committed from a filesystem that dropped it.
	if err := run(shell, env.env(), "sh", "gradlew", "assembleDebug"); err != nil {
		return err
	}
	apk := filepath.Join(shell, "app", "build", "outputs", "apk", "debug", "app-debug.apk")
	fmt.Printf("\nAPK: %s\n", apk)

	if !*install {
		fmt.Println("Install it with -install, or open android/ in Android Studio and press Run.")
		return nil
	}
	if err := run(root, nil, adb, "install", "-r", apk); err != nil {
		return fmt.Errorf("%w\n(is a device connected or an emulator running? `adb devices` lists them)", err)
	}
	// The activity class keeps the shell's namespace; only the application
	// ID is the app's. See androidPatches.
	return run(root, nil, adb, "shell", "am", "start", "-n", cfg.ID+"/com.grmob.app.MainActivity")
}

// --- ios -------------------------------------------------------------------

func cmdIOS(args []string) error {
	fset := flag.NewFlagSet("ios", flag.ContinueOnError)
	open := fset.Bool("open", false, "open the Xcode project after building")
	refresh := fset.Bool("refresh", false, "copy grmob's iOS shell over ios/ again (overwrites edits to copied files)")
	if err := fset.Parse(args); err != nil {
		return err
	}

	root, cfg, err := appContext()
	if err != nil {
		return err
	}

	fmt.Println("Checking iOS prerequisites…")
	if err := blocking(iosChecks()); err != nil {
		return fmt.Errorf("the iOS target is not ready on this machine:\n%v\n\n(`grmob doctor` shows every target)", err)
	}

	shell := filepath.Join(root, iosShell.dir)
	if err := vendorShell(root, iosShell, iosPatches(cfg), *refresh); err != nil {
		return err
	}

	path, err := ensureGomobile(root)
	if err != nil {
		return err
	}

	fmt.Println("Binding the app (gomobile, iOS device + simulator slices)…")
	bind := []string{"bind", "-target=ios,iossimulator", "-o", filepath.Join(shell, "Frameworks", "GrMob.xcframework")}
	bind = append(bind, ldflags()...)
	bind = append(bind, grmobModule+"/mobile", "./app")
	if err := run(root, []string{path}, "gomobile", bind...); err != nil {
		return err
	}

	fmt.Println("Generating the Xcode project…")
	if err := run(shell, nil, "xcodegen", "generate"); err != nil {
		return err
	}

	// A simulator build proves the whole chain links — Swift shell, Go
	// framework, generated project — without needing a signing identity,
	// which a device build does and a first run should not.
	fmt.Println("Building for the iOS Simulator…")
	if err := run(shell, nil, "xcodebuild",
		"-project", "GrMobApp.xcodeproj", "-scheme", "GrMobApp", "-configuration", "Debug",
		"-destination", "generic/platform=iOS Simulator", "-derivedDataPath", "build",
		"-quiet", "build"); err != nil {
		return err
	}
	app := filepath.Join(shell, "build", "Build", "Products", "Debug-iphonesimulator", "GrMobApp.app")
	fmt.Printf("\nApp: %s\n", app)
	fmt.Printf("Run it on a booted simulator:\n  xcrun simctl install booted %q && xcrun simctl launch booted %s\n", app, cfg.ID)

	if *open {
		return run(root, nil, "open", filepath.Join(shell, "GrMobApp.xcodeproj"))
	}
	fmt.Println("Or open ios/GrMobApp.xcodeproj in Xcode (-open) to run on a device.")
	return nil
}

// --- shared steps ----------------------------------------------------------

// appContext finds the app root and reads its grmob.json.
func appContext() (string, appConfig, error) {
	root, err := findAppRoot()
	if err != nil {
		return "", appConfig{}, err
	}
	cfg, err := readConfig(root)
	return root, cfg, err
}

// ldflags passes LDFLAGS through to gomobile's Go linker, as grmob's own
// build scripts do, so an app can bake in build-time settings with -X.
// Nothing is passed when it is empty: gomobile rejects an empty -ldflags.
func ldflags() []string {
	if v := os.Getenv("LDFLAGS"); v != "" {
		return []string{"-ldflags", v}
	}
	return nil
}

// vendorShell makes sure the app has the platform shell, copying it from the
// resolved grmob module when it is absent or when refresh asks.
//
//	android/ missing              → copy, patch, record the version
//	android/ present, recorded    → leave it; warn if the record ≠ go.mod's grmob
//	android/ present, no record   → leave it; the app made or adopted it by hand
//	-refresh                      → copy over, patch, re-record
func vendorShell(root string, spec shellSpec, patches []patch, refresh bool) error {
	dst := filepath.Join(root, spec.dir)
	src, err := grmobDir(root)
	if err != nil {
		return err
	}
	version, err := grmobVersion(root)
	if err != nil {
		return err
	}

	if fileExists(dst) && !refresh {
		recorded, err := os.ReadFile(filepath.Join(dst, shellRecord))
		if err == nil && strings.TrimSpace(string(recorded)) != version {
			fmt.Printf("\nwarning: %s/ was copied from grmob %s, but go.mod now resolves %s.\n"+
				"The native renderer should match the Go side; re-copy it with -refresh\n"+
				"(files you edited in %s/ will be overwritten — commit them first).\n\n",
				spec.dir, strings.TrimSpace(string(recorded)), version, spec.dir)
		}
		return nil
	}

	fmt.Printf("Copying grmob's %s shell into %s/ (from %s)…\n", spec.dir, spec.dir, version)
	if err := copyTree(filepath.Join(src, spec.dir), dst, spec); err != nil {
		return err
	}
	for _, p := range patches {
		path := filepath.Join(dst, filepath.FromSlash(p.file))
		b, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		out, err := p.apply(string(b))
		if err != nil {
			return fmt.Errorf("setting the app's identity in %s/%s: %w (this grmob's shell does not match this command)", spec.dir, p.file, err)
		}
		if err := os.WriteFile(path, []byte(out), 0o644); err != nil {
			return err
		}
	}
	return os.WriteFile(filepath.Join(dst, shellRecord), []byte(version+"\n"), 0o644)
}

// copyTree copies src to dst, skipping spec.skip and restating file modes.
func copyTree(src, dst string, spec shellSpec) error {
	skipped := func(rel string) bool {
		for _, s := range spec.skip {
			if rel == s || strings.HasPrefix(rel, s+"/") {
				return true
			}
		}
		return false
	}
	return filepath.WalkDir(src, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, p)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		if rel != "." && skipped(rel) {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		out := filepath.Join(dst, filepath.FromSlash(rel))
		if d.IsDir() {
			return os.MkdirAll(out, 0o755)
		}
		if !d.Type().IsRegular() {
			return nil // no symlinks or devices in a shell; nothing to follow
		}
		b, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		mode := os.FileMode(0o644)
		for _, x := range spec.executable {
			if rel == x {
				mode = 0o755
			}
		}
		// Remove first: a -refresh over a file copied from the read-only
		// module cache by an older version of this command may lack write
		// permission.
		_ = os.Remove(out)
		return os.WriteFile(out, b, mode)
	})
}

// ensureGomobile builds gomobile and gobind into <app>/.grmob/bin from the
// app's module graph and returns a PATH entry putting them first.
//
// Built rather than required on PATH for two reasons. The version: grmob's
// go.mod pins golang.org/x/mobile (its `tool` block), and binding with a
// different gomobile than the one grmob's bridge is tested with is an
// avoidable variable. And the dependency: gomobile bind compiles a generated
// program that imports golang.org/x/mobile/bind, which must be resolvable
// from the app's module — `go get` of the three packages below is what adds it,
// the step gomobile's own error message would otherwise ask for.
//
// gobind has to be on PATH, not merely built: gomobile finds it by name.
func ensureGomobile(root string) (string, error) {
	fmt.Println("Preparing gomobile (built from this module's graph)…")
	version, err := output(root, "go", "list", "-m", "-f", "{{.Version}}", "golang.org/x/mobile")
	if err != nil || version == "" {
		version = "latest"
	}
	pkgs := []string{"golang.org/x/mobile/bind", "golang.org/x/mobile/cmd/gomobile", "golang.org/x/mobile/cmd/gobind"}
	get := []string{"get"}
	for _, p := range pkgs {
		get = append(get, p+"@"+version)
	}
	if err := run(root, nil, "go", get...); err != nil {
		return "", err
	}
	bin := filepath.Join(root, ".grmob", "bin")
	if err := run(root, nil, "go", "build", "-o", bin+string(filepath.Separator), pkgs[1], pkgs[2]); err != nil {
		return "", err
	}
	return "PATH=" + bin + string(filepath.ListSeparator) + os.Getenv("PATH"), nil
}
