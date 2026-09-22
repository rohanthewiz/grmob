package tutorial

import (
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
	"testing"
)

// The README's chapter table and both "N lessons across M chapters"
// sentences restate the curriculum's shape by hand. The finale (8.5) already
// computes its figures from Chapters, but nothing held the prose to them, and
// the table went stale by 23 lessons in chapter 4 alone before anyone saw it
// (N-063). These tests read the documents and compare them with the live
// curriculum, so adding a lesson without touching the table fails here and
// names the row.
//
// The table is found by its row shape, not by a heading: `| 4 — Title | 37 |`.
// The title column has to match the chapter's Title exactly, so renaming a
// chapter also fails here. That is on purpose, because a row that no longer
// names its chapter is stale too.

// The docs sit two levels up from this package (examples/tutorial).
const repoRoot = "../../"

var chapterRow = regexp.MustCompile(`(?m)^\| (\d+) — (.+?) \| (\d+) \|`)

func TestReadmeChapterTableMatchesTheCurriculum(t *testing.T) {
	src, err := os.ReadFile(repoRoot + "README.md")
	if err != nil {
		t.Fatal(err)
	}
	rows := chapterRow.FindAllStringSubmatch(string(src), -1)
	if len(rows) != len(Chapters) {
		t.Fatalf("README's chapter table has %d rows; the curriculum has %d chapters",
			len(rows), len(Chapters))
	}
	for i, r := range rows {
		num, _ := strconv.Atoi(r[1])
		count, _ := strconv.Atoi(r[3])
		ch := Chapters[i]
		if num != i+1 || r[2] != ch.Title || count != len(ch.Lessons) {
			t.Errorf("README row %q: want | %d — %s | %d |", r[0], i+1, ch.Title, len(ch.Lessons))
		}
	}
}

func TestDocsLessonTotalsMatchTheCurriculum(t *testing.T) {
	want := fmt.Sprintf("%d lessons across %d chapters", len(flatLessons), len(Chapters))
	// The screenshot's alt text ("0 of 80 lessons opened") describes a
	// picture taken on a day, so it is left out: it is right as long as the
	// image is, and re-taking the image is a separate job.
	for _, doc := range []string{"README.md", "docs/tutorial-interactive.md"} {
		src, err := os.ReadFile(repoRoot + doc)
		if err != nil {
			t.Fatal(err)
		}
		// A sentence can wrap across lines in Markdown, so newlines are
		// folded to spaces before the search.
		flat := strings.Join(strings.Fields(string(src)), " ")
		if !strings.Contains(flat, want) {
			t.Errorf("%s should say %q", doc, want)
		}
	}
}
