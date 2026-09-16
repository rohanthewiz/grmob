package mobile

import (
	"testing"
	"time"
)

// A known IANA id loads with its DST rules; an unknown one falls back to the
// fixed offset the shell passed, keeping the name so logs still say which.
func TestResolveTimeZone(t *testing.T) {
	chicago := resolveTimeZone("America/Chicago", -5*3600)
	winter := time.Date(2026, 1, 15, 12, 0, 0, 0, chicago)
	summer := time.Date(2026, 7, 15, 12, 0, 0, 0, chicago)
	if _, off := winter.Zone(); off != -6*3600 {
		t.Errorf("January offset = %d, want CST", off)
	}
	if _, off := summer.Zone(); off != -5*3600 {
		t.Errorf("July offset = %d, want CDT", off)
	}

	fixed := resolveTimeZone("Not/AZone", 19800)
	if name, off := time.Date(2026, 1, 1, 0, 0, 0, 0, fixed).Zone(); name != "Not/AZone" || off != 19800 {
		t.Errorf("fallback = %s %d", name, off)
	}
	if name, _ := time.Now().In(resolveTimeZone("", 0)).Zone(); name != "Local" {
		t.Errorf("empty name gave %q", name)
	}
}
