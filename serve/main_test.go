package main

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/rohanthewiz/rweb"
)

// plainTestServer is the non-dev route table from main over a temp
// directory, driven through RWeb's synthetic requests.
func plainTestServer(t *testing.T, files map[string]string) (*rweb.Server, string) {
	t.Helper()
	dir := t.TempDir()
	for name, body := range files {
		if err := os.MkdirAll(filepath.Dir(filepath.Join(dir, name)), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	static, err := newStaticFiles(dir)
	if err != nil {
		t.Fatal(err)
	}
	s := rweb.NewServer()
	getAndHead(s, "/", static.serveIndex)
	getAndHead(s, "/*path", static.serve)
	return s, dir
}

// The one header a static host must get right for a WASM page: without
// application/wasm, WebAssembly.instantiateStreaming refuses the module. The
// rest are the types the page's other files need to load as what they are.
func TestStaticContentTypes(t *testing.T) {
	s, _ := plainTestServer(t, map[string]string{
		"index.html":       "<html></html>",
		"main.wasm":        "\x00asm",
		"wasm_exec.js":     "//",
		"grmob-runtime.js": "//",
	})
	for path, want := range map[string]string{
		"/main.wasm":        "application/wasm",
		"/wasm_exec.js":     "javascript",
		"/grmob-runtime.js": "javascript",
		"/index.html":       "text/html",
		"/":                 "text/html",
	} {
		res := s.Request("GET", path, nil, nil)
		if got := res.Header("Content-Type"); !strings.Contains(got, want) {
			t.Errorf("%s: Content-Type = %q, want %s", path, got, want)
		}
		if st := res.Status(); st != http.StatusOK {
			t.Errorf("%s: status %d", path, st)
		}
	}
}

// "/" is the page, a directory is its index.html, and a directory without
// one is not found (no listing, which http.FileServer would have given).
func TestStaticIndexes(t *testing.T) {
	s, _ := plainTestServer(t, map[string]string{
		"index.html":       "root page",
		"shots/index.html": "shots page",
		"empty/x.txt":      "x",
	})
	for path, want := range map[string]string{
		"/":       "root page",
		"/shots/": "shots page",
		"/shots":  "shots page",
	} {
		if got := string(s.Request("GET", path, nil, nil).Body()); got != want {
			t.Errorf("%s: body %q, want %q", path, got, want)
		}
	}
	if st := s.Request("GET", "/empty/", nil, nil).Status(); st != http.StatusNotFound {
		t.Errorf("/empty/: status %d, want 404", st)
	}
}

// Nothing outside the served directory is reachable, however the path is
// spelled: literal dot-dots, encoded ones, and a symlink pointing out.
func TestStaticStaysInsideTheDirectory(t *testing.T) {
	s, dir := plainTestServer(t, map[string]string{"index.html": "x"})
	secret := filepath.Join(filepath.Dir(dir), "secret.txt")
	if err := os.WriteFile(secret, []byte("secret"), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Remove(secret) })
	if err := os.Symlink(secret, filepath.Join(dir, "link.txt")); err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{
		"/../secret.txt",
		"/%2e%2e/secret.txt",
		"/..%2fsecret.txt",
		"/link.txt",
		"/nope.txt",
	} {
		res := s.Request("GET", path, nil, nil)
		if strings.Contains(string(res.Body()), "secret") {
			t.Errorf("%s: served a file outside the directory", path)
		}
	}
}

// A browser revalidating an unchanged file gets 304 and no body; one whose
// copy predates the file gets the file.
func TestStaticConditionalGet(t *testing.T) {
	s, dir := plainTestServer(t, map[string]string{"main.wasm": "\x00asm"})
	mod := time.Date(2026, 9, 1, 12, 0, 0, 0, time.UTC)
	if err := os.Chtimes(filepath.Join(dir, "main.wasm"), mod, mod); err != nil {
		t.Fatal(err)
	}
	ims := func(t time.Time) []rweb.Header {
		return []rweb.Header{{Key: "If-Modified-Since", Value: t.Format(http.TimeFormat)}}
	}
	if st := s.Request("GET", "/main.wasm", ims(mod), nil).Status(); st != http.StatusNotModified {
		t.Errorf("unchanged file: status %d, want 304", st)
	}
	res := s.Request("GET", "/main.wasm", ims(mod.Add(-time.Hour)), nil)
	if st := res.Status(); st != http.StatusOK || string(res.Body()) != "\x00asm" {
		t.Errorf("stale copy: status %d body %q, want 200 and the file", st, res.Body())
	}
}
