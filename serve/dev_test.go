package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/rohanthewiz/rweb"
)

// devTestServer is the dev route table over a temp directory holding the
// given files, driven through RWeb's synthetic requests (Server.Request),
// which run the real router and middleware without a socket.
func devTestServer(t *testing.T, files map[string]string, buildID string) (*rweb.Server, *devServer) {
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
	d := &devServer{dir: dir, hub: newDevHub(), buildID: buildID}
	t.Cleanup(d.hub.Close)
	s := rweb.NewServer()
	d.routes(s, static)
	return s, d
}

// The shipped page must reach the browser unchanged except for the one
// appended tag, and that tag must carry the build identity the client
// compares against the stream's "hello".
func TestDevServerInjectsTheClientBeforeBody(t *testing.T) {
	page := "<html><body><div id=\"app\"></div>\n<script>x</script>\n</body>\n</html>\n"
	s, _ := devTestServer(t, map[string]string{"index.html": page}, "42-7")

	// "/" and "/index.html" are the same page; a fetch of either must not
	// bypass the injection by falling through to the file server.
	for _, route := range []string{"/", "/index.html"} {
		res := s.Request("GET", route, nil, nil)
		body := string(res.Body())

		want := `<script src="/__dev/client.js" data-build="42-7"></script>` + "\n</body>"
		if !strings.Contains(body, want) {
			t.Fatalf("%s: client tag not injected before </body>:\n%s", route, body)
		}
		if !strings.HasPrefix(body, "<html><body><div id=\"app\"></div>\n<script>x</script>\n") {
			t.Fatalf("%s: page content altered:\n%s", route, body)
		}
		if got := res.Header("Cache-Control"); got != "no-store" {
			t.Fatalf("%s: Cache-Control = %q, want no-store", route, got)
		}
	}
}

// Everything else under the directory is the plain file server, still
// uncached in dev, and the client script is served from memory.
func TestDevServerServesFilesAndClient(t *testing.T) {
	s, _ := devTestServer(t, map[string]string{"index.html": "<body></body>", "main.wasm": "\x00asm"}, "1-1")

	res := s.Request("GET", "/main.wasm", nil, nil)
	if got := res.Header("Content-Type"); got != "application/wasm" {
		t.Fatalf("main.wasm Content-Type = %q, want application/wasm", got)
	}
	if got := res.Header("Cache-Control"); got != "no-store" {
		t.Fatalf("main.wasm Cache-Control = %q, want no-store", got)
	}

	res = s.Request("GET", "/__dev/client.js", nil, nil)
	if !bytes.Equal(res.Body(), devClient) {
		t.Fatalf("/__dev/client.js is not the embedded client")
	}
	if got := res.Header("Content-Type"); got != "application/javascript" {
		t.Fatalf("client Content-Type = %q", got)
	}
}

// A page's stream opens with the state a late page needs: which build is
// current and whether the last build failed — and that hello is the first
// thing on its channel, ahead of anything broadcast afterwards.
func TestDevServerHelloCarriesBuildAndError(t *testing.T) {
	d := &devServer{hub: newDevHub(), buildID: "1-2", lastFail: "boom"}
	t.Cleanup(d.hub.Close)

	ch := d.subscribe()
	d.broadcast(sseEvent{"building", map[string]any{}})

	first := (<-ch).(rweb.SSEvent)
	if first.Type != "hello" || first.Data != `{"build":"1-2","error":"boom"}` {
		t.Fatalf("unexpected stream opening: %+v", first)
	}
	if second := (<-ch).(rweb.SSEvent); second.Type != "building" {
		t.Fatalf("broadcast not delivered after hello: %+v", second)
	}
	if n := d.hub.ClientCount(); n != 1 {
		t.Fatalf("hub has %d clients, want 1", n)
	}
}

// A build's outcome is recorded and announced together, so a page that
// subscribes afterwards is greeted with the new state.
func TestBuildStateReachesLaterHello(t *testing.T) {
	d := &devServer{hub: newDevHub(), buildID: "old"}
	t.Cleanup(d.hub.Close)
	d.setStateAndBroadcast("new", "", sseEvent{"reload", map[string]any{"kind": "wasm", "build": "new"}})

	hello := (<-d.subscribe()).(rweb.SSEvent)
	if hello.Data != `{"build":"new","error":""}` {
		t.Fatalf("hello after a build = %s", hello.Data)
	}
}

// diffStamps must report edits, additions and removals, and nothing when
// the two snapshots agree — a spurious difference is a spurious rebuild.
func TestDiffStamps(t *testing.T) {
	before := map[string]string{"/m/a.go": "1", "/m/b.go": "1", "/m/gone.go": "1"}
	after := map[string]string{"/m/a.go": "1", "/m/b.go": "2", "/m/new.go": "1"}
	got := diffStamps(before, after)
	want := []string{"/m/b.go", "/m/gone.go", "/m/new.go"}
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("diffStamps = %v, want %v", got, want)
	}
	if got := diffStamps(after, after); len(got) != 0 {
		t.Fatalf("identical snapshots differ: %v", got)
	}
}
