package main

import (
	"bytes"
	"embed"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"runtime/debug"
	"strings"
	"text/template"
	"unicode"
)

// templates is the scaffold, one file per file the app gets.
//
// Names map to output paths by dropping a trailing ".tmpl" and turning a
// leading "dot-" into "."; see outputPath. Go sources carry the suffix so this
// module never compiles them (and its checks that count and parse every Go
// file in the tree never see them), and so a go.mod among them could not turn
// the directory into a nested module, which would drop it from embed.
//
//go:embed templates
var templates embed.FS

// appConfig is grmob.json: the identities an app has outside Go. Written once
// by new and read by the native commands; the app may edit it.
type appConfig struct {
	Name string `json:"name"` // what a launcher shows: "My App"
	ID   string `json:"id"`   // Android applicationId and iOS bundle identifier
}

// scaffoldData is what the templates see.
type scaffoldData struct {
	Module string // the app's module path
	Name   string // display name
	ID     string // application ID
}

func cmdNew(args []string) error {
	fset := flag.NewFlagSet("new", flag.ContinueOnError)
	module := fset.String("module", "", "Go module path (default: the directory's name)")
	name := fset.String("name", "", "display name shown under the app icon (default: from the directory's name)")
	id := fset.String("id", "", "application ID for Android and iOS (default: com.example.<name>)")
	version := fset.String("grmob", "", "grmob version to require, e.g. v0.3.0 or master (default: this command's own version)")
	replace := fset.String("replace", "", "use the grmob checkout at this path instead of a published version")
	noBuild := fset.Bool("no-build", false, "skip the first build")
	fset.Usage = func() {
		fmt.Fprintln(fset.Output(), "usage: grmob new <dir> [flags]")
		fset.PrintDefaults()
	}

	// Flags are accepted on either side of <dir>. The flag package stops at
	// the first positional argument, and `grmob new myapp -name "My App"` is
	// the order people type, so parse once, take the directory, and parse the
	// remainder again.
	if err := fset.Parse(args); err != nil {
		return err
	}
	if fset.NArg() < 1 {
		fset.Usage()
		return errors.New("new: missing <dir>")
	}
	dir := fset.Arg(0)
	if err := fset.Parse(fset.Args()[1:]); err != nil {
		return err
	}
	if fset.NArg() > 0 {
		return fmt.Errorf("new: unexpected arguments %v", fset.Args())
	}

	abs, err := filepath.Abs(dir)
	if err != nil {
		return err
	}
	if err := requireEmptyDir(abs); err != nil {
		return err
	}

	base := filepath.Base(abs)
	data := scaffoldData{Module: *module, Name: *name, ID: *id}
	if data.Module == "" {
		data.Module = base
	}
	if data.Name == "" {
		data.Name = displayName(base)
	}
	if data.ID == "" {
		data.ID = "com.example." + idSegment(base)
	}
	if err := validateID(data.ID); err != nil {
		return err
	}

	require, replaceDir, err := resolveGrmob(*version, *replace)
	if err != nil {
		return err
	}

	fmt.Printf("Creating %s in %s\n", data.Name, abs)
	if err := renderTemplates(abs, data); err != nil {
		return err
	}
	cfg, _ := json.MarshalIndent(appConfig{Name: data.Name, ID: data.ID}, "", "  ")
	if err := os.WriteFile(filepath.Join(abs, configFile), append(cfg, '\n'), 0o644); err != nil {
		return err
	}

	// The module is made with the go command rather than a go.mod template so
	// that the go directive is the toolchain actually installed, and so the
	// requirement graph and go.sum come from resolution rather than from a
	// guess written into this binary.
	if err := run(abs, nil, "go", "mod", "init", data.Module); err != nil {
		return err
	}
	if replaceDir != "" {
		fmt.Printf("Using the grmob checkout at %s (replace directive)\n", replaceDir)
		if err := run(abs, nil, "go", "mod", "edit", "-replace", grmobModule+"="+replaceDir); err != nil {
			return err
		}
	} else {
		if err := run(abs, nil, "go", "get", grmobModule+"@"+require); err != nil {
			return fmt.Errorf("%w\n\nIf that version predates `grmob new` (it needs the webhost package), pass -grmob master", err)
		}
	}
	if err := run(abs, nil, "go", "mod", "tidy"); err != nil {
		return err
	}

	if !*noBuild {
		fmt.Println("First build (browser target):")
		if err := run(abs, nil, "sh", "build.sh"); err != nil {
			return fmt.Errorf("the app was created but its first build failed: %w", err)
		}
	}

	// `grmob web` is listed beside dev.sh rather than among the native
	// targets: it is the browser build, and it is the only command that
	// notices a host page left behind by a grmob upgrade. dev.sh builds
	// whatever wasm/index.html holds, so without this line a new app's
	// author would not learn the check exists until the page misbehaves.
	rel := dir
	fmt.Printf(`
Done. Next:

  cd %s
  ./dev.sh                 # http://localhost:8080 — rebuilds and hot-swaps on every save

A one-off browser build, after checking wasm/index.html against go.mod's grmob
(it warns when an upgrade changed the page; -refresh re-renders it):

  go run %s/cmd/grmob web

Native targets, once their SDKs are installed:

  go run %s/cmd/grmob doctor     # what this machine can build, and what to install
  go run %s/cmd/grmob android    # APK via the Android SDK + NDK
  go run %s/cmd/grmob ios        # simulator build via Xcode

`, rel, grmobModule, grmobModule, grmobModule, grmobModule)
	return nil
}

// requireEmptyDir refuses to scaffold over existing work. A directory that
// exists and is empty is fine (a freshly cloned empty repository is exactly
// that, minus .git, which is allowed too).
func requireEmptyDir(dir string) error {
	entries, err := os.ReadDir(dir)
	if errors.Is(err, fs.ErrNotExist) {
		return os.MkdirAll(dir, 0o755)
	}
	if err != nil {
		return err
	}
	for _, e := range entries {
		if e.Name() != ".git" {
			return fmt.Errorf("new: %s is not empty", dir)
		}
	}
	return nil
}

// resolveGrmob decides which grmob the new app requires. In order:
//
//  1. -replace: a local checkout, wired with a replace directive.
//  2. -grmob: an explicit version, branch or commit for `go get`.
//  3. This command's own module version, when it was run as
//     `go run github.com/rohanthewiz/grmob/cmd/grmob@vX`. The scaffold's
//     templates are written against the framework they shipped with, so the
//     same version is the one known to compile.
//  4. A development build (`go run ./cmd/grmob` inside a checkout) has no
//     version, only a source tree; the checkout it was built from is found
//     through this file's own path and used as a replace. That is what makes
//     the scaffold testable before a release carries it.
//  5. Otherwise "latest".
func resolveGrmob(version, replace string) (require, replaceDir string, err error) {
	if replace != "" {
		dir, err := filepath.Abs(replace)
		if err != nil {
			return "", "", err
		}
		if !isGrmobCheckout(dir) {
			return "", "", fmt.Errorf("new: -replace %s is not a grmob checkout (no go.mod declaring %s)", replace, grmobModule)
		}
		return "", dir, nil
	}
	if version != "" {
		return version, "", nil
	}
	if bi, ok := debug.ReadBuildInfo(); ok && bi.Main.Path == grmobModule &&
		bi.Main.Version != "" && bi.Main.Version != "(devel)" {
		return bi.Main.Version, "", nil
	}
	// runtime.Caller reports the path this file was compiled from. Under
	// `go run ./cmd/grmob` that is the checkout; under -trimpath it is a
	// module-relative path that will not stat, and the check below rejects it.
	if _, file, _, ok := runtime.Caller(0); ok {
		dir := filepath.Dir(filepath.Dir(filepath.Dir(file)))
		if isGrmobCheckout(dir) {
			return "", dir, nil
		}
	}
	return "latest", "", nil
}

func isGrmobCheckout(dir string) bool {
	b, err := os.ReadFile(filepath.Join(dir, "go.mod"))
	if err != nil {
		return false
	}
	first, _, _ := strings.Cut(string(b), "\n")
	return strings.TrimSpace(first) == "module "+grmobModule
}

// renderTemplates writes every embedded template under dir.
//
// The delimiters are [[ ]] rather than {{ }} because two of the outputs speak
// Go template syntax themselves: build.sh passes `-f '{{.Dir}}'` to go list,
// and a {{ }} scaffold would have to escape every one of those.
func renderTemplates(dir string, data scaffoldData) error {
	return fs.WalkDir(templates, "templates", func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		src, err := templates.ReadFile(p)
		if err != nil {
			return err
		}
		rendered, err := renderTemplate(p, src, data)
		if err != nil {
			return err
		}
		out := filepath.Join(dir, outputPath(strings.TrimPrefix(p, "templates/")))
		if err := os.MkdirAll(filepath.Dir(out), 0o755); err != nil {
			return err
		}
		// Shell scripts are the only executables in the scaffold; embed keeps
		// no file modes, so the extension is the record of which ones they are.
		mode := os.FileMode(0o644)
		if strings.HasSuffix(out, ".sh") {
			mode = 0o755
		}
		return os.WriteFile(out, rendered, mode)
	})
}

// renderTemplate executes one scaffold template with the scaffold's
// delimiters and strictness.
//
// One function for both callers — new, rendering the embedded copies, and web
// -refresh, rendering the host page from the grmob module go.mod resolves — so
// that a page refreshed later is byte-for-byte what new would write today.
// web's "differs" warning is only meaningful while that holds.
func renderTemplate(name string, src []byte, data scaffoldData) ([]byte, error) {
	tmpl, err := template.New(name).Delims("[[", "]]").Option("missingkey=error").Parse(string(src))
	if err != nil {
		return nil, fmt.Errorf("template %s: %w", name, err)
	}
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return nil, fmt.Errorf("template %s: %w", name, err)
	}
	return buf.Bytes(), nil
}

// --- web -------------------------------------------------------------------

// The host page's place in an app, and the template it is rendered from,
// relative to the grmob module's directory. The template path is the same one
// the embed above reads, which is what lets a published grmob serve it: the
// module zip carries cmd/grmob/templates like any other directory of the
// root module.
const (
	hostPagePath     = "wasm/index.html"
	hostPageTemplate = "cmd/grmob/templates/wasm/index.html.tmpl"
)

// cmdWeb builds the browser target, first comparing the app's host page with
// the one the grmob module in go.mod scaffolds.
//
// # Why the host page needs a command at all
//
// Everything else grmob owns on the browser side follows go.mod by itself:
// webhost is imported, and build.sh re-copies grmob-runtime.js and camera.js
// on every build. The host page is the exception. new renders it once, because
// an app is expected to edit it, and nothing afterwards compares it with a
// newer grmob's — so v0.3.0's page, whose Scroll rule collapses a horizontal
// Scroll to 0px, stays in every app scaffolded then through every upgrade,
// unless someone reads the Upgrading notes and pastes CSS by hand.
//
// # Why a diff and not a version record, as the native shells have
//
// The shells carry a record file inside android/ and ios/, directories grmob
// creates and fills. The host page is one file in wasm/, beside the app's own
// main.go, and a record would be a second file the app has to keep next to it.
// Comparing against a fresh render needs no record and is exact: the same
// bytes mean current. Different bytes mean an older grmob's page OR the app's
// own edits, which this cannot tell apart — so it says "differs", names both,
// and only -refresh writes.
//
//	wasm/index.html missing   → render and write it
//	same bytes as the render  → nothing to say; build
//	differs, no -refresh      → warn (an older page, or local edits); build
//	differs, -refresh         → overwrite with the render; build
func cmdWeb(args []string) error {
	fset := flag.NewFlagSet("web", flag.ContinueOnError)
	refresh := fset.Bool("refresh", false, "re-render wasm/index.html from the grmob module in go.mod (overwrites edits to it)")
	noBuild := fset.Bool("no-build", false, "check (or with -refresh, re-render) the host page only; skip build.sh")
	if err := fset.Parse(args); err != nil {
		return err
	}

	root, cfg, err := appContext()
	if err != nil {
		return err
	}
	if _, err := syncHostPage(root, cfg, *refresh); err != nil {
		return err
	}
	if *noBuild {
		return nil
	}
	fmt.Println("Building the browser target (build.sh)…")
	return run(root, nil, "sh", "build.sh")
}

// hostPageState is what syncHostPage found and did, one value per arm of the
// table on cmdWeb. Returned rather than only printed so a test can walk the
// arms without reading stdout.
type hostPageState int

const (
	hostPageCurrent   hostPageState = iota // same bytes as the render
	hostPageWritten                        // was missing; rendered
	hostPageDiffers                        // differs; left alone, warned
	hostPageRefreshed                      // differed; overwritten
)

// syncHostPage renders the host page from the grmob module the app at root
// resolves and applies cmdWeb's table to wasm/index.html.
//
// The template is read from that module's directory rather than from this
// command's embed. The two differ whenever the grmob command is run at a
// version other than go.mod's (`@latest` against an app pinned older, say),
// and the page has to match the runtime build.sh copies — which is go.mod's.
func syncHostPage(root string, cfg appConfig, refresh bool) (hostPageState, error) {
	src, err := grmobDir(root)
	if err != nil {
		return 0, err
	}
	version, err := grmobVersion(root)
	if err != nil {
		return 0, err
	}
	tmpl, err := os.ReadFile(filepath.Join(src, filepath.FromSlash(hostPageTemplate)))
	if errors.Is(err, fs.ErrNotExist) {
		return 0, fmt.Errorf("grmob %s has no host-page template (%s): it predates `grmob new`, so there is nothing to refresh from — upgrade grmob in go.mod first", version, hostPageTemplate)
	}
	if err != nil {
		return 0, err
	}
	// The app's module path, for the template data new had. The host page does
	// not use it today, but renderTemplate is strict (missingkey=error), and a
	// later template that did would otherwise render differently here than in
	// new — which is the one thing the "differs" comparison cannot afford.
	module, err := output(root, "go", "list", "-m")
	if err != nil {
		return 0, err
	}
	want, err := renderTemplate(hostPageTemplate, tmpl, scaffoldData{Module: module, Name: cfg.Name, ID: cfg.ID})
	if err != nil {
		return 0, err
	}

	dst := filepath.Join(root, filepath.FromSlash(hostPagePath))
	got, err := os.ReadFile(dst)
	switch {
	case errors.Is(err, fs.ErrNotExist):
		fmt.Printf("Rendering %s from grmob %s…\n", hostPagePath, version)
		if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
			return 0, err
		}
		return hostPageWritten, os.WriteFile(dst, want, 0o644)
	case err != nil:
		return 0, err
	case bytes.Equal(got, want):
		return hostPageCurrent, nil
	case !refresh:
		fmt.Printf("\nwarning: %s differs from the host page grmob %s scaffolds.\n"+
			"Either it came from an older grmob or it was edited here. Re-render it with -refresh\n"+
			"(edits to it will be overwritten — commit them first).\n\n",
			hostPagePath, version)
		return hostPageDiffers, nil
	}
	fmt.Printf("Re-rendering %s from grmob %s…\n", hostPagePath, version)
	return hostPageRefreshed, os.WriteFile(dst, want, 0o644)
}

// outputPath maps a template's name to the file it produces:
// "wasm/main.go.tmpl" → "wasm/main.go", "dot-gitignore" → ".gitignore".
// embed skips dotfiles unless asked, and naming them plainly keeps them
// visible in this repository too.
func outputPath(name string) string {
	name = strings.TrimSuffix(name, ".tmpl")
	dir, file := filepath.Split(name)
	if rest, ok := strings.CutPrefix(file, "dot-"); ok {
		file = "." + rest
	}
	return filepath.Join(dir, file)
}

// displayName turns a directory name into a launcher label:
// "my-todo_app" → "My Todo App".
func displayName(base string) string {
	words := strings.FieldsFunc(base, func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsDigit(r)
	})
	for i, w := range words {
		r := []rune(w)
		r[0] = unicode.ToUpper(r[0])
		words[i] = string(r)
	}
	if len(words) == 0 {
		return "GrMob App"
	}
	return strings.Join(words, " ")
}

// idSegment reduces a directory name to one application-ID segment: lower
// case ASCII letters and digits, starting with a letter. That is the
// intersection of what Android (letters, digits, underscore) and iOS
// (letters, digits, hyphen, period) accept, so one ID serves both.
func idSegment(base string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(base) {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
		}
	}
	s := strings.TrimLeft(b.String(), "0123456789")
	if s == "" {
		return "app"
	}
	return s
}

// validateID holds an explicit -id to the same intersection idSegment
// produces, and to Android's rule of at least two segments. Caught here
// rather than by Gradle or Xcode minutes into a native build.
func validateID(id string) error {
	segs := strings.Split(id, ".")
	if len(segs) < 2 {
		return fmt.Errorf("new: application ID %q needs at least two segments, like com.example.app", id)
	}
	for _, s := range segs {
		if s == "" || !(s[0] >= 'a' && s[0] <= 'z' || s[0] >= 'A' && s[0] <= 'Z') {
			return fmt.Errorf("new: application ID %q: each segment must start with a letter", id)
		}
		for _, r := range s {
			if !(r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9') {
				return fmt.Errorf("new: application ID %q: use only letters and digits in each segment", id)
			}
		}
	}
	return nil
}

// readConfig loads grmob.json from an app root.
func readConfig(root string) (appConfig, error) {
	var cfg appConfig
	b, err := os.ReadFile(filepath.Join(root, configFile))
	if err != nil {
		return cfg, err
	}
	if err := json.Unmarshal(b, &cfg); err != nil {
		return cfg, fmt.Errorf("%s: %w", configFile, err)
	}
	if cfg.Name == "" || cfg.ID == "" {
		return cfg, fmt.Errorf("%s: both \"name\" and \"id\" must be set", configFile)
	}
	return cfg, validateID(cfg.ID)
}
