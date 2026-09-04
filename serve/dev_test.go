package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// The shipped page must reach the browser unchanged except for the one
// appended tag, and that tag must carry the build identity the client
// compares against the stream's "hello".
func TestDevServerInjectsTheClientBeforeBody(t *testing.T) {
	dir := t.TempDir()
	page := "<html><body><div id=\"app\"></div>\n<script>x</script>\n</body>\n</html>\n"
	if err := os.WriteFile(filepath.Join(dir, "index.html"), []byte(page), 0o644); err != nil {
		t.Fatal(err)
	}
	d := &devServer{dir: dir, next: http.NotFoundHandler(), subs: map[chan sseEvent]struct{}{}, buildID: "42-7"}

	rec := httptest.NewRecorder()
	d.ServeHTTP(rec, httptest.NewRequest("GET", "/", nil))
	body := rec.Body.String()

	want := `<script src="/__dev/client.js" data-build="42-7"></script>` + "\n</body>"
	if !strings.Contains(body, want) {
		t.Fatalf("client tag not injected before </body>:\n%s", body)
	}
	if !strings.HasPrefix(body, "<html><body><div id=\"app\"></div>\n<script>x</script>\n") {
		t.Fatalf("page content altered:\n%s", body)
	}
	if got := rec.Header().Get("Cache-Control"); got != "no-store" {
		t.Fatalf("Cache-Control = %q, want no-store", got)
	}
}

// The stream opens with the state a late page needs: which build is current
// and whether the last build failed.
func TestDevServerHelloCarriesBuildAndError(t *testing.T) {
	d := &devServer{dir: t.TempDir(), next: http.NotFoundHandler(), subs: map[chan sseEvent]struct{}{}, buildID: "1-2", lastFail: "boom"}
	req := httptest.NewRequest("GET", "/__dev/events", nil)
	ctx, cancel := context.WithCancel(req.Context())
	rec := httptest.NewRecorder()
	done := make(chan struct{})
	go func() {
		d.ServeHTTP(rec, req.WithContext(ctx))
		close(done)
	}()
	cancel()
	<-done
	got := rec.Body.String()
	if !strings.HasPrefix(got, "event: hello\ndata: {\"build\":\"1-2\",\"error\":\"boom\"}\n\n") {
		t.Fatalf("unexpected stream opening:\n%s", got)
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
