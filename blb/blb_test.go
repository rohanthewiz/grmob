package blb

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/rohanthewiz/grmob/comps"
)

// fixture reads a verbatim reply saved from the live endpoint. The file name
// is the passage id the reply answered.
func fixture(t *testing.T, id string) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join("testdata", id+".reply"))
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func TestPassageID(t *testing.T) {
	cases := []struct{ ref, tr, want string }{
		{"John 3:16", "", "KJV.John.3.16"},
		{"John 3:16-18", "esv", "ESV.John.3.16-18"},
		{"1 John 4:7-8", "NKJV", "NKJV.1John.4.7-8"},
		{"1John 4:8", "", "KJV.1John.4.8"},
		{"II Kings 2:11", "", "KJV.2Kings.2.11"},
		{"I John 1:9", "", "KJV.1John.1.9"},
		{"III John 1:4", "", "KJV.3John.1.4"},
		{"Isaiah 53:5", "", "KJV.Isaiah.53.5"}, // not "1saiah": no space after the I
		{"Song of Songs 2:1", "", "KJV.SongofSongs.2.1"},
		{"Psalm 23", "", "KJV.Psalm.23"},
		{"Rom 8:28,31", "", "KJV.Rom.8.28,31"},
		{"Gen. 1.1", "", "KJV.Gen.1.1"},
		{"  Jude 3  ", "", "KJV.Jude.3"},
	}
	for _, c := range cases {
		got, err := PassageID(c.ref, c.tr)
		if err != nil || got != c.want {
			t.Errorf("PassageID(%q, %q) = %q, %v; want %q", c.ref, c.tr, got, err, c.want)
		}
	}
	for _, bad := range []string{"", "John", "3:16", "John three"} {
		if _, err := PassageID(bad, ""); !errors.Is(err, ErrInvalidReference) {
			t.Errorf("PassageID(%q) err = %v, want ErrInvalidReference", bad, err)
		}
	}
}

func TestParseARange(t *testing.T) {
	p, err := parseReply(fixture(t, "KJV.John.3.16-18"), false)
	if err != nil {
		t.Fatal(err)
	}
	if p.Reference != "John 3:16-18" || p.Translation != "KJV" {
		t.Errorf("reference %q translation %q", p.Reference, p.Translation)
	}
	if p.URL != "https://www.blueletterbible.org/kjv/jhn/3/16/" {
		t.Errorf("URL = %q; the s_ deep-link segment should be dropped", p.URL)
	}
	if p.ChapterID != 1000 {
		t.Errorf("ChapterID = %d, want 1000 (from s_1000016)", p.ChapterID)
	}
	nums := []int{}
	for _, v := range p.Verses {
		nums = append(nums, v.Number)
	}
	if !slices.Equal(nums, []int{16, 17, 18}) {
		t.Fatalf("verse numbers = %v", nums)
	}
	if !strings.HasPrefix(p.Verses[0].Text, "For God so loved the world") ||
		!strings.HasSuffix(p.Verses[0].Text, "have everlasting life.") {
		t.Errorf("verse 16 = %q", p.Verses[0].Text)
	}
	// The last verse stops at the body's close: the "Powered by" link and
	// the logo after it are not scripture.
	if last := p.Verses[2].Text; strings.Contains(last, "Powered") || !strings.HasSuffix(last, "Son of God.") {
		t.Errorf("verse 18 = %q", last)
	}
	if !strings.HasPrefix(p.Text(), "For God so loved") || strings.Contains(p.Text(), "16") {
		t.Errorf("Text() should join the verses without their numbers: %q", p.Text())
	}
}

// Quotes inside the text arrive as JavaScript escapes (\') and HTML
// entities; both must come out as the characters.
func TestParseUnescapesQuotes(t *testing.T) {
	p, err := parseReply(fixture(t, "ESV.Gen.3.1"), false)
	if err != nil {
		t.Fatal(err)
	}
	got := p.Verses[0].Text
	if !strings.Contains(got, `"Did God actually say, 'You shall not eat`) {
		t.Errorf("verse = %q", got)
	}
}

// Off by default: footnote asterisks and the KJV's supplied-word brackets
// are removed. KeepMarkers keeps both.
func TestMarkers(t *testing.T) {
	p, _ := parseReply(fixture(t, "KJV.Psa.23.1-2"), false)
	if got := p.Verses[0].Text; got != "A Psalm of David. The LORD is my shepherd; I shall not want." {
		t.Errorf("clean = %q", got)
	}
	p, _ = parseReply(fixture(t, "KJV.Psa.23.1-2"), true)
	if got := p.Verses[0].Text; !strings.Contains(got, "[is]") {
		t.Errorf("kept = %q", got)
	}

	p, _ = parseReply(fixture(t, "NKJV.Rom.1.16-17"), false)
	if strings.Contains(p.Text(), "*") {
		t.Errorf("footnote markers left in: %q", p.Text())
	}
	p, _ = parseReply(fixture(t, "NKJV.Rom.1.16-17"), true)
	if !strings.Contains(p.Text(), "Christ,*") {
		t.Errorf("KeepMarkers dropped a footnote marker: %q", p.Text())
	}
}

func TestParseAWholeChapter(t *testing.T) {
	p, err := parseReply(fixture(t, "KJV.Psa.117"), false)
	if err != nil {
		t.Fatal(err)
	}
	if p.Reference != "Psalms 117" || len(p.Verses) != 2 {
		t.Errorf("reference %q, %d verses", p.Reference, len(p.Verses))
	}
}

func TestParseErrors(t *testing.T) {
	if _, err := parseReply(fixture(t, "KJV.Foo.1.1"), false); !errors.Is(err, ErrInvalidReference) {
		t.Errorf("BLB's error bubble: err = %v, want ErrInvalidReference", err)
	}
	for _, junk := range []string{"", "<html>Bad gateway</html>", "o.responseText = '<h6>John 3:16 (KJV)</h6>';"} {
		if _, err := parseReply(junk, false); !errors.Is(err, ErrUnexpectedResponse) {
			t.Errorf("parseReply(%q) err = %v, want ErrUnexpectedResponse", junk, err)
		}
	}
}

// The request Fetch sends, through a fake BLB: the path, the id, par style,
// and the client's default translation when the call names none.
func TestFetchAgainstAFakeServer(t *testing.T) {
	var gotPath, gotID, gotStyle string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath, gotID, gotStyle = r.URL.Path, r.URL.Query().Get("id"), r.URL.Query().Get("style")
		if gotID == "KJV.Foo.1.1" {
			w.Write([]byte(fixture(t, "KJV.Foo.1.1")))
			return
		}
		if gotID == "KJV.Down.1.1" {
			http.Error(w, "nope", http.StatusBadGateway)
			return
		}
		w.Write([]byte(fixture(t, "KJV.John.3.16-18")))
	}))
	defer srv.Close()

	c := &Client{BaseURL: srv.URL + "/", HTTPClient: srv.Client(), Translation: "kjv"}
	p, err := c.Fetch(context.Background(), "John 3:16-18", "")
	if err != nil {
		t.Fatal(err)
	}
	if gotPath != "/remoteExtensions/toolTip/toolTipRemote.cfm" || gotID != "KJV.John.3.16-18" || gotStyle != "par" {
		t.Errorf("request path %q id %q style %q", gotPath, gotID, gotStyle)
	}
	if len(p.Verses) != 3 {
		t.Errorf("%d verses", len(p.Verses))
	}

	if _, err := c.Fetch(context.Background(), "Foo 1:1", ""); !errors.Is(err, ErrInvalidReference) {
		t.Errorf("unknown book: err = %v", err)
	}
	if _, err := c.Fetch(context.Background(), "Down 1:1", ""); !errors.Is(err, ErrUnexpectedResponse) {
		t.Errorf("HTTP 502: err = %v", err)
	}
}

func TestSearchURL(t *testing.T) {
	got := SearchURL(" John 3:16 ", "esv")
	want := "https://www.blueletterbible.org/search/preSearch.cfm?Criteria=John+3%3A16&t=ESV"
	if got != want {
		t.Errorf("SearchURL = %q, want %q", got, want)
	}
}

// Every saved reply's s_ id is chapter × 1000 + its first verse, so the
// chapter comes out as the id's thousands, and StudyURL rebuilds the id the
// reply carried for that verse.
func TestStudyURLFromEveryFixture(t *testing.T) {
	for _, c := range []struct {
		id      string
		chapter int
		verse   int
		want    string
	}{
		{"ESV.Gen.3.1", 3, 1, "https://www.blueletterbible.org/esv/gen/3/1/s_3001"},
		{"KJV.John.3.16-18", 1000, 16, "https://www.blueletterbible.org/kjv/jhn/3/16/s_1000016"},
		{"KJV.Psa.117", 595, 1, "https://www.blueletterbible.org/kjv/psa/117/1/s_595001"},
		{"KJV.Psa.23.1-2", 501, 1, "https://www.blueletterbible.org/kjv/psa/23/1/s_501001"},
		{"NKJV.Rom.1.16-17", 1047, 16, "https://www.blueletterbible.org/nkjv/rom/1/16/s_1047016"},
	} {
		p, err := parseReply(fixture(t, c.id), false)
		if err != nil {
			t.Fatalf("%s: %v", c.id, err)
		}
		if p.ChapterID != c.chapter {
			t.Errorf("%s: ChapterID = %d, want %d", c.id, p.ChapterID, c.chapter)
		}
		// The bare s_ form is exactly the link in the reply.
		if got := p.StudyURL(c.verse, ""); got != c.want || !strings.Contains(fixture(t, c.id), got) {
			t.Errorf("%s: StudyURL(%d, \"\") = %q, want %q (the reply's own link)", c.id, c.verse, got, c.want)
		}
	}
}

// A tab names itself before the id; another verse of the chapter swaps both
// the path's verse and the id's.
func TestStudyURLTabsAndOtherVerses(t *testing.T) {
	p := Passage{URL: "https://www.blueletterbible.org/kjv/jhn/3/16/", ChapterID: 1000}
	for _, c := range []struct {
		verse int
		tool  string
		want  string
	}{
		{16, ToolInterlinear, "https://www.blueletterbible.org/kjv/jhn/3/16/t_conc_1000016"},
		{17, ToolCrossRefs, "https://www.blueletterbible.org/kjv/jhn/3/17/t_corr_1000017"},
		{18, ToolCommentaries, "https://www.blueletterbible.org/kjv/jhn/3/18/t_comms_1000018"},
		{21, ToolDictionaries, "https://www.blueletterbible.org/kjv/jhn/3/21/t_refs_1000021"},
		{1, ToolBibles, "https://www.blueletterbible.org/kjv/jhn/3/1/t_bibles_1000001"},
	} {
		if got := p.StudyURL(c.verse, c.tool); got != c.want {
			t.Errorf("StudyURL(%d, %q) = %q\nwant %q", c.verse, c.tool, got, c.want)
		}
	}
}

// A Passage built by hand has no ChapterID, and a URL that is not a passage
// page has no verse segment to swap: no address rather than a wrong one.
func TestStudyURLNeedsAChapterAndAPassagePage(t *testing.T) {
	for _, p := range []Passage{
		{URL: "https://www.blueletterbible.org/kjv/jhn/3/16/"},
		{ChapterID: 1000},
		{ChapterID: 1000, URL: SearchURL("John 3:16", "")},
	} {
		if got := p.StudyURL(16, ToolCrossRefs); got != "" {
			t.Errorf("%+v: StudyURL = %q, want empty", p, got)
		}
	}
	p := Passage{URL: "https://www.blueletterbible.org/kjv/jhn/3/16/", ChapterID: 1000}
	if got := p.StudyURL(0, ""); got != "" {
		t.Errorf("verse 0: StudyURL = %q, want empty", got)
	}
}

// The real endpoint, opt-in: it is the only check that BLB still answers in
// the shape the fixtures recorded.
//
//	GRMOB_BLB_LIVE=1 go test ./blb/ -run Live -v
func TestLiveEndpoint(t *testing.T) {
	if os.Getenv("GRMOB_BLB_LIVE") == "" {
		t.Skip("set GRMOB_BLB_LIVE=1 to call blueletterbible.org")
	}
	for _, c := range []struct{ ref, tr, ref2 string }{
		{"jn 3:16", "", "John 3:16"},
		{"1 John 4:7-8", "ESV", "1 John 4:7-8"},
		{"Psalm 23", "NKJV", "Psalms 23"},
		{"II Kings 2:11", "", "2 Kings 2:11"},
	} {
		p, err := Fetch(context.Background(), c.ref, c.tr)
		if err != nil {
			t.Errorf("%s: %v", c.ref, err)
			continue
		}
		if p.Reference != c.ref2 || len(p.Verses) == 0 || p.URL == "" {
			t.Errorf("%s: %+v", c.ref, p)
		}
	}
	// StudyURL for a verse no reply named: BLB titles the study page with
	// the verse's opening words, so the page is the verse's and not merely
	// the chapter's.
	p, err := Fetch(context.Background(), "John 3:16", "KJV")
	if err != nil {
		t.Fatal(err)
	}
	u := p.StudyURL(17, ToolCrossRefs)
	resp, err := http.Get(u)
	if err != nil {
		t.Fatalf("%s: %v", u, err)
	}
	page, _ := io.ReadAll(io.LimitReader(resp.Body, 64<<10))
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK || !strings.Contains(string(page), "For God sent not his") {
		t.Errorf("%s: HTTP %d, and the title should quote John 3:17", u, resp.StatusCode)
	}
	if _, err := Fetch(context.Background(), "Hezekiah 1:1", ""); !errors.Is(err, ErrInvalidReference) {
		t.Errorf("Hezekiah: err = %v", err)
	}
}

// comps cannot import blb (it would put net/http in every browser build), so
// comps.DefaultBibleStudyTools spells BLB's tab keys out itself. This holds
// the two lists together: every default is one of the Tool constants.
func TestBibleStudyDefaultsAreBLBTools(t *testing.T) {
	known := []string{ToolInterlinear, ToolBibles, ToolCrossRefs, ToolCommentaries, ToolDictionaries, ToolMisc}
	for _, tool := range comps.DefaultBibleStudyTools() {
		if !slices.Contains(known, tool.Tool) {
			t.Errorf("comps.DefaultBibleStudyTools has %q (%s), which is not a blb Tool constant", tool.Tool, tool.Label)
		}
	}
	// A method value of StudyURL is what BibleStudy.StudyURL takes.
	_ = comps.BibleStudy{StudyURL: Passage{}.StudyURL}
}
