package core_test

import (
	"testing"

	"github.com/rohanthewiz/grmob/core"
)

// recordEvents installs a recording system-event handler and restores the
// previous (absent) one when the test ends. The handler is package-level in
// core, so a test that forgot to detach would leak into every later test in
// the binary.
func recordEvents(t *testing.T) *[]struct {
	Name string
	Data map[string]any
} {
	t.Helper()
	var seen []struct {
		Name string
		Data map[string]any
	}
	core.SetSystemEventHandler(func(name string, data map[string]any) {
		seen = append(seen, struct {
			Name string
			Data map[string]any
		}{name, data})
	})
	t.Cleanup(func() { core.SetSystemEventHandler(nil) })
	return &seen
}

func TestOpenURLSendsOneSystemEvent(t *testing.T) {
	seen := recordEvents(t)

	core.OpenURL("https://www.blueletterbible.org/kjv/jhn/3/16")

	if len(*seen) != 1 {
		t.Fatalf("expected exactly one system event, got %d", len(*seen))
	}
	got := (*seen)[0]
	if got.Name != "open_url" {
		t.Errorf("event name = %q, want %q", got.Name, "open_url")
	}
	if url, _ := got.Data["url"].(string); url != "https://www.blueletterbible.org/kjv/jhn/3/16" {
		t.Errorf("payload url = %q, want the URL passed in", url)
	}
	// The payload carries the URL and nothing else: hosts read one key, and a
	// stray second one would be a silent contract change for all four of them.
	if len(got.Data) != 1 {
		t.Errorf("payload has %d keys, want 1: %v", len(got.Data), got.Data)
	}
}

// An empty URL is dropped in Go rather than handed to a host that could only
// reject it — and could not report having done so, since a system event has
// no return channel.
func TestOpenURLDropsAnEmptyURL(t *testing.T) {
	seen := recordEvents(t)

	core.OpenURL("")

	if len(*seen) != 0 {
		t.Fatalf("empty URL sent %d events, want none: %v", len(*seen), *seen)
	}
}

// Non-http schemes pass through untouched: `tel:` and `mailto:` links are the
// second and third reasons this function exists, and validating them here
// would mean re-implementing each platform's scheme table.
func TestOpenURLPassesNonHTTPSchemesThrough(t *testing.T) {
	seen := recordEvents(t)

	for _, want := range []string{"tel:+15551234567", "mailto:office@example.org"} {
		core.OpenURL(want)
	}

	if len(*seen) != 2 {
		t.Fatalf("expected two events, got %d", len(*seen))
	}
	for i, want := range []string{"tel:+15551234567", "mailto:office@example.org"} {
		if got, _ := (*seen)[i].Data["url"].(string); got != want {
			t.Errorf("event %d url = %q, want %q", i, got, want)
		}
	}
}

// With no handler installed the call is a no-op rather than a panic — the
// headless case (unit tests, the HTML exporter) must not need a stub host.
func TestOpenURLWithNoHandlerIsHarmless(t *testing.T) {
	core.SetSystemEventHandler(nil)
	core.OpenURL("https://example.org")
}
