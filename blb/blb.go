// Package blb fetches Bible passages from Blue Letter Bible
// (blueletterbible.org), for comps.BibleVerse and anything else that wants a
// verse's text, its canonical reference and a link back to the passage.
//
//	p, err := blb.Fetch(ctx, "John 3:16-18", "ESV")
//	// p.Reference "John 3:16-18", p.Translation "ESV",
//	// p.Verses [{16 "For God so loved…"} {17 …} {18 …}],
//	// p.URL "https://www.blueletterbible.org/esv/jhn/3/16/"
//
// # Where the text comes from
//
// Blue Letter Bible has no open JSON API. What it does publish, for anyone to
// embed, is the BLB ScriptTagger (blueletterbible.org/webtools): a script a
// site includes so that every reference on its pages shows the verse in a
// hover bubble. The bubble's contents arrive from one endpoint,
//
//	GET /remoteExtensions/toolTip/toolTipRemote.cfm?id=KJV.John.3.16&style=par
//
// which answers with a few lines of JavaScript that hand an HTML string to the
// tagger. This package makes the same request the tagger makes and reads the
// same HTML out of the reply, so it serves exactly what BLB already serves to
// every page that embeds its tool, under the same terms: the text links back
// to Blue Letter Bible (Passage.URL), and comps.BibleVerse draws that link by
// default.
//
// The endpoint is not a documented API, so its shape can change without
// notice. The parser is deliberately narrow (the title, the verse markers,
// the first passage link) and fails with ErrUnexpectedResponse rather than
// returning half a verse when the shape moves. The fixtures in blb_test.go
// are verbatim replies, and the live test (GRMOB_BLB_LIVE=1) checks the
// endpoint itself.
//
// # Where it works
//
// On Android and iOS (Go's net/http under gomobile) and on a server. A
// browser build cannot call it directly: the reply carries no
// Access-Control-Allow-Origin header, so the browser refuses to hand it to
// WebAssembly. A browser app points Client.BaseURL at a same-origin proxy
// that forwards EndpointPath to www.blueletterbible.org. Proxy is that
// handler, for the app's own server; `go run ./serve` and a scaffolded
// app's ./dev.sh mount one, so verses load in a browser during development
// once BaseURL is the page's origin. A static host (GitHub Pages) has no
// server to put it on.
//
// # Translations
//
// The translation is BLB's code, upper-case: KJV, NKJV, NLT, NIV, ESV, CSB,
// NASB20, NASB95, LSB, AMP, NET, RSV, ASV, YLT and WEB all answer. All but
// the KJV, ASV, YLT and WEB are under copyright. An app that stores or
// republishes their text, rather than showing it with the link, should check
// the publisher's terms first.
package blb

import (
	"context"
	"errors"
	"fmt"
	"html"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// DefaultBaseURL is Blue Letter Bible's origin.
const DefaultBaseURL = "https://www.blueletterbible.org"

// userAgent names this package to BLB, on Fetch's requests and on the ones
// Proxy makes for a browser, so both read the same in BLB's logs.
const userAgent = "grmob-blb/1 (+https://github.com/rohanthewiz/grmob)"

// DefaultTranslation is used when neither the call nor the Client names one.
// The King James is the one translation BLB serves that is in the public
// domain everywhere it is read (outside the UK's Crown patent), which makes
// it the safe default for an app that has not thought about licensing yet.
const DefaultTranslation = "KJV"

// ErrInvalidReference is returned when BLB does not recognise the reference
// ("Hezekiah 1:1", "John 99:1"). Test for it with errors.Is.
var ErrInvalidReference = errors.New("blb: invalid scripture reference")

// ErrUnexpectedResponse is returned when the reply parses as neither a
// passage nor BLB's error bubble: the endpoint changed shape, or a proxy in
// front of it answered with something else.
var ErrUnexpectedResponse = errors.New("blb: unexpected response")

// Verse is one numbered verse of a passage.
type Verse struct {
	Number int
	Text   string
}

// Passage is a fetched reference.
type Passage struct {
	// Reference is BLB's own spelling of what was asked for, with the book
	// name expanded: "jn 3:16" comes back as "John 3:16", "Song 2:1" as
	// "Song of Songs 2:1". Show this rather than the caller's input.
	Reference string

	// Translation is the code the text is from, as BLB titled it ("ESV").
	Translation string

	// Verses are in the order BLB returned them, which is reading order.
	Verses []Verse

	// URL is the passage's page on blueletterbible.org, the link the text
	// owes its source. Empty only if BLB's reply carried no passage link.
	URL string
}

// Text is the passage as one paragraph, verses joined by a space and their
// numbers left out: the form for a share sheet or a notification.
func (p Passage) Text() string {
	parts := make([]string, 0, len(p.Verses))
	for _, v := range p.Verses {
		parts = append(parts, v.Text)
	}
	return strings.Join(parts, " ")
}

// Client fetches passages. The zero value is ready to use.
type Client struct {
	// HTTPClient makes the requests. Nil uses a client with a 15-second
	// timeout. The timeout is a backstop; a call's context still governs.
	HTTPClient *http.Client

	// BaseURL replaces DefaultBaseURL, for a proxy (see "Where it works"
	// and Proxy) or a test server. No trailing slash is needed.
	BaseURL string

	// Translation is the default for calls that pass "". Empty means
	// DefaultTranslation.
	Translation string

	// KeepMarkers keeps the translators' marks BLB leaves in the text: the
	// asterisk after a word that has a footnote (ESV, NKJV), and the square
	// brackets the KJV puts around words the translators supplied
	// ("The LORD [is] my shepherd"). Off, they are removed, which is how a
	// verse is quoted outside a study edition.
	KeepMarkers bool
}

// DefaultClient is used by the package-level Fetch.
var DefaultClient = &Client{}

// defaultHTTPClient is shared by every Client with a nil HTTPClient, so their
// connections pool.
var defaultHTTPClient = &http.Client{Timeout: 15 * time.Second}

// maxReply bounds how much of a reply is read. The longest passage the
// tagger will serve (a whole psalm 119) is about 60 KB of HTML; a reply ten
// times that is not a passage.
const maxReply = 1 << 20

// Fetch fetches ref in translation with DefaultClient.
func Fetch(ctx context.Context, ref, translation string) (Passage, error) {
	return DefaultClient.Fetch(ctx, ref, translation)
}

// Fetch fetches ref ("John 3:16", "1 John 4:7-8", "Psalm 23", "Rom 8:28,31")
// in translation ("" for the Client's default).
func (c *Client) Fetch(ctx context.Context, ref, translation string) (Passage, error) {
	if translation == "" {
		translation = c.Translation
	}
	id, err := PassageID(ref, translation)
	if err != nil {
		return Passage{}, err
	}

	q := url.Values{}
	q.Set("id", id)
	// "par" puts each verse's number in a <span class="vRef"> inside one run
	// of text; "line" gives each verse its own link instead, and the number
	// as that link's text. par is the shape the parser reads.
	q.Set("style", "par")
	q.Set("target", "true")
	endpoint := strings.TrimRight(c.baseURL(), "/") + EndpointPath + "?" + q.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return Passage{}, fmt.Errorf("blb: building request for %q: %w", ref, err)
	}
	req.Header.Set("User-Agent", userAgent)

	hc := c.HTTPClient
	if hc == nil {
		hc = defaultHTTPClient
	}
	resp, err := hc.Do(req)
	if err != nil {
		return Passage{}, fmt.Errorf("blb: fetching %q: %w", ref, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return Passage{}, fmt.Errorf("blb: fetching %q: HTTP %d: %w", ref, resp.StatusCode, ErrUnexpectedResponse)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxReply))
	if err != nil {
		return Passage{}, fmt.Errorf("blb: reading %q: %w", ref, err)
	}

	p, err := parseReply(string(body), c.KeepMarkers)
	if err != nil {
		return Passage{}, fmt.Errorf("blb: %q: %w", ref, err)
	}
	return p, nil
}

func (c *Client) baseURL() string {
	if c.BaseURL != "" {
		return c.BaseURL
	}
	return DefaultBaseURL
}

// refPattern splits a human reference into book and location. The book is
// everything up to the last space before a digit; a leading ordinal ("1",
// "2", "3", "I", "II") is part of the book.
//
//	"1 John 4:7-8"     → book "1 John", loc "4:7-8"
//	"Song of Songs 2"  → book "Song of Songs", loc "2"
//	"Rom 8:28,31"      → book "Rom", loc "8:28,31"
var refPattern = regexp.MustCompile(`^\s*((?:[1-3]\s*)?[A-Za-z][A-Za-z .]*?)\s*(\d+(?:[:.]\d+(?:-\d+)?(?:,\d+(?:-\d+)?)*)?)\s*$`)

// PassageID builds the tagger's id for ref: translation, book with its spaces
// removed, then chapter and verses with "." for ":".
//
//	PassageID("1 John 4:7-8", "esv") == "ESV.1John.4.7-8"
//	PassageID("Psalm 23", "")        == "KJV.Psalm.23"
//
// BLB resolves the book name itself, abbreviations included, so this does
// no book lookup of its own and an unknown book is reported by BLB
// (ErrInvalidReference from Fetch), not here. An error here means the
// reference has no chapter number to send.
func PassageID(ref, translation string) (string, error) {
	m := refPattern.FindStringSubmatch(ref)
	if m == nil {
		return "", fmt.Errorf("blb: %q has no book and chapter: %w", ref, ErrInvalidReference)
	}
	book := strings.NewReplacer(" ", "", ".", "").Replace(m[1])
	// A roman ordinal is how some readers write "II Kings"; BLB's id wants
	// the digit. It counts only when a space follows it in the input, which
	// is what keeps "Isaiah" from becoming "1saiah". Longest first, so "III "
	// is not read as "I" plus "II".
	trimmed := strings.TrimSpace(ref)
	for _, r := range []struct{ roman, digit string }{{"III", "3"}, {"II", "2"}, {"I", "1"}} {
		if strings.HasPrefix(trimmed, r.roman+" ") {
			book = r.digit + strings.TrimPrefix(book, r.roman)
			break
		}
	}
	loc := strings.ReplaceAll(m[2], ":", ".")

	if translation == "" {
		translation = DefaultTranslation
	}
	return strings.ToUpper(translation) + "." + book + "." + loc, nil
}

// SearchURL is a link to ref on Blue Letter Bible that needs no fetch first:
// BLB's own search, which opens the passage when the query is a reference.
// It is what the ScriptTagger links a reference to, and the fallback for a
// comps.BibleVerse whose fetch failed.
func SearchURL(ref, translation string) string {
	if translation == "" {
		translation = DefaultTranslation
	}
	q := url.Values{}
	q.Set("Criteria", strings.TrimSpace(ref))
	q.Set("t", strings.ToUpper(translation))
	return DefaultBaseURL + "/search/preSearch.cfm?" + q.Encode()
}

// The reply, verbatim apart from the SVG logo and whitespace:
//
//	var o = {};
//	o.TTipID = 'NKJV.Rom.1.16-17';
//	o.responseText = ''+
//	    '<div id="blbTagger" class="bubble">'+
//	    '  <div class="bubHead"><div><h6>Romans 1:16-17 (NKJV)</h6></div></div>'+
//	    '  <div id="bubBody" class="bubBody"><div>'+
//	    '    <a href="https://www.blueletterbible.org/nkjv/rom/1/16/s_1047016" …>Rom 1:16-17 NKJV</a>'+
//	    '    - <span class="vRef">16</span> For I am not ashamed …'+
//	    '    <span class="vRef">17</span> For in it …'+
//	    '  </div></div>'+
//	    '  <a href="…/BLB_ScriptTagger.cfm" class="blb-cta">Powered by <svg>…</svg></a>'+
//	    '</div>';
//	BLB.Tagger.AjaxObject.handleSuccess(o);
//
// An unknown reference has the same frame, an <h6> of "Blue Letter Bible -
// Error" and the message in the body.
var (
	// jsString matches one single-quoted JavaScript literal, escapes and all.
	jsString = regexp.MustCompile(`'((?:[^'\\]|\\.)*)'`)
	title    = regexp.MustCompile(`<h6>(.*?)</h6>`)
	// The title is "Book C:V (TRANS)"; the translation is the last
	// parenthesised word, so a reference can never be mistaken for it.
	titleParts = regexp.MustCompile(`^(.*\S)\s*\(([^()]+)\)$`)
	verseMark  = regexp.MustCompile(`<span class="vRef">(\d+)</span>`)
	passageRef = regexp.MustCompile(`href="(https?://[^"]*blueletterbible\.org/[a-z0-9]+/[a-z0-9]+/\d+/\d+/)(?:s_\d+)?"`)
	tag        = regexp.MustCompile(`<[^>]*>`)
	space      = regexp.MustCompile(`\s+`)
	footnote   = regexp.MustCompile(`\*`)
	bracket    = regexp.MustCompile(`[\[\]]`)
)

// parseReply reads a Passage out of the tagger's JavaScript reply.
func parseReply(reply string, keepMarkers bool) (Passage, error) {
	// Only the literals after "responseText =" are the bubble; the one
	// before is the TTipID, which would otherwise be read as HTML.
	_, after, ok := strings.Cut(reply, "responseText")
	if !ok {
		return Passage{}, ErrUnexpectedResponse
	}
	var b strings.Builder
	for _, m := range jsString.FindAllStringSubmatch(after, -1) {
		b.WriteString(unescapeJS(m[1]))
	}
	doc := b.String()

	tm := title.FindStringSubmatch(doc)
	if tm == nil {
		return Passage{}, ErrUnexpectedResponse
	}
	heading := cleanText(tm[1], true)
	marks := verseMark.FindAllStringSubmatchIndex(doc, -1)
	if len(marks) == 0 {
		if strings.Contains(heading, "Error") {
			return Passage{}, ErrInvalidReference
		}
		return Passage{}, ErrUnexpectedResponse
	}

	p := Passage{Reference: heading}
	if parts := titleParts.FindStringSubmatch(heading); parts != nil {
		p.Reference, p.Translation = parts[1], parts[2]
	}
	if m := passageRef.FindStringSubmatch(doc); m != nil {
		// The s_NNN segment is BLB's internal verse id, a deep link into the
		// study page's interlinear. Without it the address is the plain
		// passage page, which is the one to share.
		p.URL = m[1]
	}

	// Each verse runs from the end of its marker to the start of the next,
	// and the last one to the close of the body's <div>.
	for i, m := range marks {
		num, _ := strconv.Atoi(doc[m[2]:m[3]])
		end := len(doc)
		if i+1 < len(marks) {
			end = marks[i+1][0]
		} else if j := strings.Index(doc[m[1]:], "</div>"); j >= 0 {
			end = m[1] + j
		}
		text := cleanText(doc[m[1]:end], keepMarkers)
		if text == "" {
			continue
		}
		p.Verses = append(p.Verses, Verse{Number: num, Text: text})
	}
	if len(p.Verses) == 0 {
		return Passage{}, ErrUnexpectedResponse
	}
	return p, nil
}

// cleanText turns a fragment of the bubble's HTML into plain text.
func cleanText(s string, keepMarkers bool) string {
	s = tag.ReplaceAllString(s, " ")
	s = html.UnescapeString(s)
	if !keepMarkers {
		s = footnote.ReplaceAllString(s, "")
		s = bracket.ReplaceAllString(s, "")
	}
	return strings.TrimSpace(space.ReplaceAllString(s, " "))
}

// unescapeJS undoes the escapes BLB's literals use: \' and \" for quotes in
// the text, \\ for a backslash, and \n / \t, which it has not been seen to
// emit but which are the same rule. Anything else after a backslash is
// kept as written.
func unescapeJS(s string) string {
	if !strings.Contains(s, `\`) {
		return s
	}
	var b strings.Builder
	b.Grow(len(s))
	for i := 0; i < len(s); i++ {
		if s[i] != '\\' || i+1 == len(s) {
			b.WriteByte(s[i])
			continue
		}
		i++
		switch s[i] {
		case 'n':
			b.WriteByte('\n')
		case 't':
			b.WriteByte('\t')
		default:
			b.WriteByte(s[i])
		}
	}
	return b.String()
}
