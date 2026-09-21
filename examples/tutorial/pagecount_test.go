package tutorial

import (
	"os"
	"regexp"
	"strconv"
	"testing"
)

// The browser page's header states the tutorial's size ("77 lessons · 8
// chapters") in hand-written HTML, which no render reaches. Every other place
// that states the count is either a render (the contents' progress caption)
// or pinned by a screenshot claim, so this was the one that drifted: it said
// 60 from the foldables round until 74, fourteen lessons later. Counted from
// Chapters here, so a lesson added without the page fails a test instead.
func TestPageHeaderCountsTheLessons(t *testing.T) {
	page, err := os.ReadFile("../../wasm/index.html")
	if err != nil {
		t.Fatalf("reading the page: %v", err)
	}
	m := regexp.MustCompile(`(\d+) lessons · (\d+) chapters`).FindSubmatch(page)
	if m == nil {
		t.Fatal(`wasm/index.html no longer states "N lessons · M chapters"; update this test if the header changed on purpose`)
	}
	lessons := 0
	for _, c := range Chapters {
		lessons += len(c.Lessons)
	}
	if got, _ := strconv.Atoi(string(m[1])); got != lessons {
		t.Errorf("wasm/index.html says %d lessons; the tutorial has %d", got, lessons)
	}
	if got, _ := strconv.Atoi(string(m[2])); got != len(Chapters) {
		t.Errorf("wasm/index.html says %d chapters; the tutorial has %d", got, len(Chapters))
	}
}
