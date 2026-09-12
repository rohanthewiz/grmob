package main

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
)

// The prerequisite checks, one table per target.
//
// doctor prints all three; `grmob android` and `grmob ios` run their own table
// first and stop with its remedies before starting anything slow. One set of
// checks serves both so that the advice doctor prints is exactly the condition
// the build will hold you to — two lists would disagree the first time one of
// them was edited.
//
// gomobile is not on either list. The native commands build gomobile and
// gobind from the app's own module graph (see ensureGomobile), so the version
// that binds the app is the one grmob's go.mod pins, and nothing needs to be
// installed globally.

// check is one prerequisite: what was looked for, whether it is there, what
// was found, and what to do when it is not.
type check struct {
	item   string
	ok     bool
	detail string // what was found, or why it failed
	fix    string // the remedy; printed only when !ok
	// optional checks are reported but do not block a build: adb is needed to
	// install an APK, not to make one.
	optional bool
}

func cmdDoctor(args []string) error {
	if len(args) > 0 {
		return fmt.Errorf("doctor takes no arguments")
	}
	report("Browser (WASM)", wasmChecks())
	report("Android", androidChecks())
	report("iOS", iosChecks())
	return nil
}

// report prints one target's table and whether the target is buildable.
func report(target string, checks []check) {
	ready := blocking(checks) == nil
	state := "ready"
	if !ready {
		state = "not ready"
	}
	fmt.Printf("\n%s — %s\n", target, state)
	for _, c := range checks {
		mark := "✓"
		if !c.ok {
			mark = "✗"
			if c.optional {
				mark = "-"
			}
		}
		fmt.Printf("  %s %-22s %s\n", mark, c.item, c.detail)
		if !c.ok && c.fix != "" {
			for _, line := range strings.Split(c.fix, "\n") {
				fmt.Printf("      %s\n", line)
			}
		}
	}
}

// blocking returns an error naming every failed required check with its
// remedy, or nil when the target can build.
func blocking(checks []check) error {
	var msgs []string
	for _, c := range checks {
		if !c.ok && !c.optional {
			msgs = append(msgs, fmt.Sprintf("%s: %s\n    %s", c.item, c.detail, strings.ReplaceAll(c.fix, "\n", "\n    ")))
		}
	}
	if len(msgs) == 0 {
		return nil
	}
	return errors.New(strings.Join(msgs, "\n"))
}

// --- Browser ---------------------------------------------------------------

func wasmChecks() []check {
	var out []check

	goVer, err := output("", "go", "env", "GOVERSION")
	c := check{item: "Go 1.26+", detail: goVer,
		fix: "Install Go 1.26 or newer: https://go.dev/dl/"}
	if err != nil {
		c.detail = "go not found on PATH"
	} else {
		c.ok = goAtLeast(goVer, 1, 26)
	}
	out = append(out, c)

	// wasm_exec.js is the JS half of Go's js/wasm port. build.sh copies it
	// from the toolchain so the shim always matches the compiler that made
	// main.wasm; a toolchain packaged without it (some distro packages split
	// it out) cannot run a module at all.
	c = check{item: "wasm_exec.js", fix: "Reinstall Go from https://go.dev/dl/ (this toolchain ships without its js/wasm support files)"}
	if goroot, err := output("", "go", "env", "GOROOT"); err == nil {
		for _, p := range []string{"lib/wasm/wasm_exec.js", "misc/wasm/wasm_exec.js"} {
			if fileExists(filepath.Join(goroot, p)) {
				c.ok, c.detail = true, filepath.Join(goroot, p)
				break
			}
		}
		if !c.ok {
			c.detail = "not in " + goroot
		}
	}
	return append(out, c)
}

// goAtLeast compares a GOVERSION string ("go1.26.1", "go1.27rc1") against a
// minimum major.minor.
func goAtLeast(v string, major, minor int) bool {
	m := regexp.MustCompile(`^go(\d+)\.(\d+)`).FindStringSubmatch(v)
	if m == nil {
		return false
	}
	maj, _ := strconv.Atoi(m[1])
	mnr, _ := strconv.Atoi(m[2])
	return maj > major || (maj == major && mnr >= minor)
}

// --- Android ---------------------------------------------------------------

// androidEnv is where the Android toolchain was found. Filled by
// androidChecks and handed to the build so it runs against the same SDK,
// NDK and JDK that were checked.
type androidEnv struct {
	sdk, ndk, javaHome string
}

// env is the environment gomobile and Gradle need. gomobile reads
// ANDROID_HOME, and ANDROID_NDK_HOME only when the user set one; Gradle reads
// ANDROID_HOME (in place of a local.properties) and JAVA_HOME.
func (e androidEnv) env() []string {
	env := []string{"ANDROID_HOME=" + e.sdk}
	if e.ndk != "" {
		env = append(env, "ANDROID_NDK_HOME="+e.ndk)
	}
	if e.javaHome != "" {
		env = append(env, "JAVA_HOME="+e.javaHome)
	}
	return env
}

func androidChecks() []check {
	checks, _ := androidProbe()
	return checks
}

func androidProbe() ([]check, androidEnv) {
	var env androidEnv
	var out []check

	sdkFix := "Install Android Studio (https://developer.android.com/studio) and open it once,\n" +
		"or install the command-line tools and set ANDROID_HOME to the SDK directory."
	c := check{item: "Android SDK", fix: sdkFix}
	env.sdk = androidSDKDir()
	switch {
	case env.sdk == "":
		c.detail = "ANDROID_HOME is not set and no SDK is at the default location"
	case !fileExists(filepath.Join(env.sdk, "platforms")):
		c.detail = env.sdk + " has no platforms/ — the SDK is incomplete"
	default:
		c.ok, c.detail = true, env.sdk
	}
	out = append(out, c)

	// gomobile compiles Go's runtime for Android with the NDK's clang.
	//
	// Which NDK is left to gomobile unless ANDROID_NDK_HOME says otherwise:
	// given only ANDROID_HOME it takes the newest side-by-side NDK under
	// $ANDROID_HOME/ndk that supports the requested targets, reading each
	// one's meta/platforms.json and meta/abis.json. That is a better answer
	// than a newest-by-name pick here (a newest NDK can have dropped an ABI
	// the shell still builds), so this check asks only whether there is
	// anything for gomobile to choose from.
	c = check{item: "Android NDK", fix: "Android Studio → Settings → Languages & Frameworks → Android SDK → SDK Tools →\n" +
		"check \"NDK (Side by side)\" and apply. Or: sdkmanager --install \"ndk;28.2.13676358\""}
	if ndk := os.Getenv("ANDROID_NDK_HOME"); ndk != "" {
		env.ndk = ndk
		c.ok, c.detail = fileExists(ndk), ndk
		if !c.ok {
			c.detail = "ANDROID_NDK_HOME is " + ndk + ", which does not exist"
		}
	} else if names := subdirs(filepath.Join(env.sdk, "ndk")); env.sdk != "" && len(names) > 0 {
		c.ok, c.detail = true, strings.Join(names, ", ")+" in "+filepath.Join(env.sdk, "ndk")
	} else {
		c.detail = "no NDK installed"
	}
	out = append(out, c)

	// The shell's Gradle plugin (AGP 9) runs on JDK 17 or newer. macOS ships
	// a /usr/bin/java stub that exists and fails, so presence on PATH says
	// nothing; the version is what is checked. Android Studio's bundled JBR
	// is accepted too, which is the JDK most people who installed Studio
	// already have without knowing it.
	c = check{item: "JDK 17+", fix: "Install a JDK 17 or newer (e.g. `brew install openjdk@21`), or install Android Studio,\n" +
		"whose bundled JBR is used automatically."}
	if home, ver := findJDK(17); home != "" || ver > 0 {
		env.javaHome = home
		c.ok = ver >= 17
		c.detail = fmt.Sprintf("Java %d", ver)
		if home != "" {
			c.detail += " at " + home
		}
		if !c.ok {
			c.detail += " (too old)"
		}
	} else {
		c.detail = "no working java found"
	}
	out = append(out, c)

	c = check{item: "adb", optional: true, detail: "not found (needed only for -install)",
		fix: "Install \"Android SDK Platform-Tools\" from the SDK Manager."}
	if env.sdk != "" && fileExists(filepath.Join(env.sdk, "platform-tools", "adb")) {
		c.ok, c.detail = true, filepath.Join(env.sdk, "platform-tools", "adb")
	}
	out = append(out, c)

	return out, env
}

// androidSDKDir finds the SDK: ANDROID_HOME, then its older name
// ANDROID_SDK_ROOT, then the location Android Studio installs to.
func androidSDKDir() string {
	for _, v := range []string{"ANDROID_HOME", "ANDROID_SDK_ROOT"} {
		if d := os.Getenv(v); d != "" {
			return d
		}
	}
	home, _ := os.UserHomeDir()
	var d string
	switch runtime.GOOS {
	case "darwin":
		d = filepath.Join(home, "Library", "Android", "sdk")
	case "windows":
		d = filepath.Join(os.Getenv("LOCALAPPDATA"), "Android", "Sdk")
	default:
		d = filepath.Join(home, "Android", "Sdk")
	}
	if fileExists(d) {
		return d
	}
	return ""
}

// findJDK returns a JAVA_HOME to hand Gradle (empty when the java on PATH
// should be used as is) and the Java feature version found. Candidates are
// tried in the order Gradle itself would honour them, and the first that is
// new enough wins; if none is, the newest seen is reported so doctor can say
// "too old" rather than "missing".
func findJDK(min int) (home string, version int) {
	type cand struct{ home, java string }
	var cands []cand
	if h := os.Getenv("JAVA_HOME"); h != "" {
		cands = append(cands, cand{h, filepath.Join(h, "bin", "java")})
	}
	cands = append(cands, cand{"", "java"})
	if runtime.GOOS == "darwin" {
		if h, err := output("", "/usr/libexec/java_home", "-v", strconv.Itoa(min)+"+"); err == nil && h != "" {
			cands = append(cands, cand{h, filepath.Join(h, "bin", "java")})
		}
		for _, h := range []string{
			"/Applications/Android Studio.app/Contents/jbr/Contents/Home",
			"/Applications/Android Studio Preview.app/Contents/jbr/Contents/Home",
		} {
			cands = append(cands, cand{h, filepath.Join(h, "bin", "java")})
		}
	}
	bestHome, best := "", 0
	for _, c := range cands {
		v := javaVersion(c.java)
		if v >= min {
			return c.home, v
		}
		if v > best {
			bestHome, best = c.home, v
		}
	}
	return bestHome, best
}

// javaVersion runs `java -version` and returns the feature version: 17 for
// "17.0.9", 8 for the old "1.8.0_392" scheme, 0 when it does not run.
func javaVersion(java string) int {
	cmd := exec.Command(java, "-version")
	b, err := cmd.CombinedOutput() // the version banner goes to stderr
	if err != nil {
		return 0
	}
	m := regexp.MustCompile(`version "(\d+)(?:\.(\d+))?`).FindSubmatch(b)
	if m == nil {
		return 0
	}
	v, _ := strconv.Atoi(string(m[1]))
	if v == 1 && len(m[2]) > 0 {
		v, _ = strconv.Atoi(string(m[2]))
	}
	return v
}

// --- iOS -------------------------------------------------------------------

func iosChecks() []check {
	if runtime.GOOS != "darwin" {
		return []check{{item: "macOS", detail: "iOS apps can only be built on macOS",
			fix: "Build the iOS target on a Mac."}}
	}
	var out []check

	// gomobile drives xcodebuild to assemble the xcframework, and the
	// Command Line Tools alone do not have it — the most common state of a
	// Mac that has only ever built Go. `xcodebuild -version` is the test
	// because it fails in exactly that state.
	c := check{item: "Xcode", fix: "Install Xcode from the App Store, then:\n" +
		"  sudo xcode-select -s /Applications/Xcode.app\n" +
		"  sudo xcodebuild -license accept\n" +
		"  xcodebuild -runFirstLaunch"}
	if v, err := output("", "xcodebuild", "-version"); err == nil {
		c.ok = true
		c.detail, _, _ = strings.Cut(v, "\n")
	} else {
		c.detail = "xcodebuild is not usable (Command Line Tools only, or the license is not accepted)"
	}
	out = append(out, c)

	// Recent Xcode releases download platform SDKs separately. Without the
	// simulator SDK the bind's iossimulator slice and the simulator build
	// both fail with a message about a missing destination.
	c = check{item: "iOS Simulator SDK", fix: "xcodebuild -downloadPlatform iOS"}
	if sdks, err := output("", "xcodebuild", "-showsdks"); err == nil && strings.Contains(sdks, "iphonesimulator") {
		c.ok, c.detail = true, "installed"
	} else {
		c.detail = "not installed"
	}
	out = append(out, c)

	// The Xcode project is generated from project.yml rather than tracked,
	// the way this repository's own ios/ is.
	c = check{item: "xcodegen", fix: "brew install xcodegen"}
	if p, err := exec.LookPath("xcodegen"); err == nil {
		c.ok, c.detail = true, p
	} else {
		c.detail = "not found on PATH"
	}
	return append(out, c)
}

// --- helpers ---------------------------------------------------------------

func fileExists(p string) bool {
	_, err := os.Stat(p)
	return err == nil
}

// subdirs lists the names of dir's subdirectories, or nil when it has none or
// does not exist. Deliberately unordered by version: see the NDK check for why
// choosing among them is gomobile's job, not this command's.
func subdirs(dir string) []string {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	var names []string
	for _, e := range entries {
		if e.IsDir() {
			names = append(names, e.Name())
		}
	}
	return names
}
