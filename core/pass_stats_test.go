package core

import (
	"bytes"
	"log"
	"testing"
	"time"
)

func withPassRecord(t *testing.T) {
	t.Helper()
	was := IsDebugMode()
	SetDebugMode(true)
	ClearPassStats()
	t.Cleanup(func() {
		SetDebugMode(was)
		SetPassLogging(false)
		ClearPassStats()
	})
}

// The record keeps the most recent passStatsCap passes, oldest first, so a
// session left in debug mode does not grow it without end.
func TestPassStatsKeepTheMostRecentPassesOldestFirst(t *testing.T) {
	withPassRecord(t)
	total := passStatsCap + 88
	for i := range total {
		RecordPassStat(PassStat{Kind: "render", Patches: i})
	}
	got := PassStats()
	if len(got) != passStatsCap {
		t.Fatalf("kept %d passes, want the cap %d", len(got), passStatsCap)
	}
	if got[0].Patches != 88 || got[len(got)-1].Patches != total-1 {
		t.Fatalf("kept passes %d..%d, want %d..%d", got[0].Patches, got[len(got)-1].Patches, 88, total-1)
	}
	for i := 1; i < len(got); i++ {
		if got[i].Patches != got[i-1].Patches+1 {
			t.Fatalf("out of order at %d: %d after %d", i, got[i].Patches, got[i-1].Patches)
		}
	}

	ClearPassStats()
	RecordPassStat(PassStat{Kind: "mount"})
	if got := PassStats(); len(got) != 1 || got[0].Kind != "mount" {
		t.Fatalf("after a clear, the record is %+v", got)
	}
}

// Logging is its own switch, and still needs debug mode to record anything.
func TestPassLoggingWritesOneLinePerPass(t *testing.T) {
	withPassRecord(t)
	var buf bytes.Buffer
	prev, flags := log.Writer(), log.Flags()
	log.SetOutput(&buf)
	log.SetFlags(0)
	t.Cleanup(func() { log.SetOutput(prev); log.SetFlags(flags) })

	stat := PassStat{Kind: "dispatch", Trigger: "cb_3", Handler: 12 * time.Microsecond,
		Build: time.Millisecond, Diff: 97 * time.Microsecond, Encode: 20 * time.Microsecond,
		Patches: 3, Bytes: 412}
	RecordPassStat(stat)
	if buf.Len() != 0 {
		t.Fatalf("logged with pass logging off: %q", buf.String())
	}

	SetPassLogging(true)
	RecordPassStat(stat)
	want := "grmob: pass dispatch cb_3: 1.129ms (handler 12µs, build 1ms, diff 97µs, encode 20µs), 3 patches, 412 B\n"
	if buf.String() != want {
		t.Fatalf("logged %q, want %q", buf.String(), want)
	}

	buf.Reset()
	SetDebugMode(false)
	RecordPassStat(stat)
	if buf.Len() != 0 || len(PassStats()) != 2 {
		t.Fatalf("with debug mode off a pass was recorded or logged: %q, %d kept", buf.String(), len(PassStats()))
	}
}
