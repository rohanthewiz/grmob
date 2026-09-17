package shotclaims

import (
	"context"
	"encoding/json"
	"fmt"
	"image"
	_ "image/png"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"
)

// The three questions the manifest can answer about itself.
//
// The fourth — whether the strings are really on the screen — is not here
// and cannot be: it needs the app, so it lives in the app's own package
// and this file only checks that something claims to ask it. See the
// package header.
//
// # Why one test and three subtests
//
// The same reason wasm/verify/sharedparse_test.go gives: each of these
// ends in an arm that stops on a reading that came back empty, and a stop
// ends the goroutine. Three questions in one function is three questions
// any one of which can silence the other two — and the silence here is
// the failure mode the whole package exists to end.
func TestEveryScreenshotIsClaimedAndEveryClaimIsShown(t *testing.T) {
	root := filepath.Join("..", "..")

	t.Run("the manifest against the image directory", func(t *testing.T) {
		checkFilesAndClaimsAgree(t, root)
	})
	t.Run("each row's own shape", func(t *testing.T) {
		checkRowsAreWellFormed(t, root)
	})
	t.Run("the README's captions against the claims", func(t *testing.T) {
		checkCaptionsQuoteTheScreen(t, root)
	})
	t.Run("the harness that takes them", func(t *testing.T) {
		checkEveryClaimIsTakeable(t, root)
	})
	t.Run("each composite against its parts", func(t *testing.T) {
		checkCompositesAreTheirParts(t, root)
	})
}

// --------------------------------------------------------------------------
// Composites
// --------------------------------------------------------------------------

// compositeBlock is the side, in CSS pixels, of the squares a composite and
// its parts are compared over; compositeInset is how far inside each part's
// slot the first square starts.
//
// # Why squares, and not pixels
//
// The composite is the browser's resampling of each part — an 828×1200 shot
// drawn 420 CSS px tall at the capture's dpr — and nothing here reproduces
// Chrome's filter. An area average is what every reasonable filter preserves:
// the mean colour over a square of drawn pixels is the mean of the source area
// under it. So the comparison survives the resampling and still sees content:
// a line of different text, a different row ticked, or a part shifted by a
// crop changes the mean of the squares it crosses by far more than resampling
// does.
//
// The inset keeps squares off a slot's edge. Slot widths are fractional
// (289.8 CSS px for todo.png) and the two sides snap them to device pixels
// differently, so the outermost column of a slot is partly ground on one side
// and partly shot on the other.
const (
	compositeBlock = 12.0
	compositeInset = 2.0
)

// compositeTolerance is the most any square's mean may differ between the
// composite and the part drawn there, per channel, out of 255.
//
// # Where 13 comes from
//
// Measured on the committed images, and on copies of the parts with a white
// box painted over their densest text (the composite left as it was, which is
// a part re-taken without re-shooting the composite):
//
//	the composite as committed      5.2 todo · 9.3 signup · 7.9 tabs-list
//	one glyph (8×12 CSS px) gone    17.2 in todo · 24.2 in tabs-list
//	a text row (240×24 CSS px) gone 35.7 in todo
//
// 13 is between the worst resampling reading and the smallest content one,
// with about a third of a glyph's reading as room on each side. That is the
// limit of the claim: a change smaller than a glyph (a single anti-aliased
// pixel, a colour shift of a few levels) is below it, and a re-shoot on a
// machine whose Chrome resamples noticeably differently could need the
// table re-taken. A crop or a re-proportioned part never reaches this reading
// at all — the size check stops it first.
//
// # Which Chrome
//
// The table was taken from images shot by Chrome 152 on this project's macOS
// machine, and the resampling row is a property of that Chrome's image filter,
// not of the images. So the version is recorded beside the number as
// compositeMeasuredOnChrome, and the check reads the local Chrome's major
// version (see localChromeMajor): on a different one it logs the difference,
// and a failure names both versions.
//
// Re-measured on Chrome 153 (153.0.8010.37, same machine) by re-taking every
// shot with wasm/shots/shoot.sh into a scratch directory: all seven PNGs,
// hero.png included, came out byte-identical to the committed ones, so the
// readings are the Chrome 152 readings exactly — worst channel difference 5.2
// (todo), 9.3 (signup) and 7.9 (tabs-list) against 13. The number moved to 153
// on that evidence; the tolerance did not need to.
//
// A log and not a failure, for two reasons. The test reads committed PNGs and
// launches nothing, so the local Chrome is only a proxy for the one that shot
// them — right after a re-shoot here, stale on a machine that never re-shot.
// And a newer Chrome that resamples the same way is the common case; failing
// on every Chrome release would be a check people learn to ignore. What the
// log buys is the first question a reader of a resampling-sized failure should
// ask, answered in the output.
const (
	compositeTolerance        = 13.0
	compositeMeasuredOnChrome = 153
)

// compositeLayout is what a composite page says about where its parts go.
// Read from the page itself, so a change to the page and a change to this
// test cannot drift apart: the page is the only copy of the numbers.
type compositeLayout struct {
	parts   []string // image files, in page order
	height  float64  // every part's drawn height, CSS px
	gap     float64  // between neighbouring parts, CSS px
	padding float64  // around the row, all four sides, CSS px
}

var (
	compositeImg = regexp.MustCompile(`<img\s+[^>]*src="images/([^"]+)"`)
	cssLength    = func(prop string) *regexp.Regexp {
		return regexp.MustCompile(`(?:^|[;{\s])` + prop + `:\s*(\d+(?:\.\d+)?)px\s*;`)
	}
)

// checkCompositesAreTheirParts holds each composite to being its parts, whole
// and current.
//
// # What it catches, and why the manifest needs it
//
// A composite carries no strings on the argument that its parts' claims
// already hold them. That is true of a composite that shows every part
// entire, drawn from the parts as they are now. Two ordinary edits break it
// silently: a crop in the page (a max-height, an object-fit) removes text the
// parts still claim, and re-taking a part without re-shooting the composite
// leaves the README's first picture showing a screen that no longer exists.
// Neither changes a claim, so without this both pass.
//
// Two readings, for those two failures:
//
//	size     the composite's pixel size is the one the page's layout gives the
//	         parts' own aspect ratios — a crop or a re-proportioned part moves it
//	content  every square inside each slot has the mean colour of the part's
//	         area under it — a stale or swapped part moves it
func checkCompositesAreTheirParts(t *testing.T, root string) {
	t.Helper()

	// Once per run, before any reading: a Chrome whose major version differs
	// from the one the tolerance was measured on is logged whether or not a
	// composite then fails, so a borderline reading has the context beside it.
	if local, _, err := localChromeMajor(); err == nil && local != compositeMeasuredOnChrome {
		t.Log(chromeProvenance())
	}

	checked := 0
	for _, c := range Claims {
		if !c.Composite() {
			continue
		}
		checked++
		checkComposite(t, root, c)
	}
	// The reaching arm: the manifest has a composite today, and a check that
	// found none would pass by having looked at nothing.
	if checked == 0 {
		t.Fatalf("no claim in the manifest is a composite, and %s holds "+
			"hero.png. Either the row changed shape or the reading is wrong.",
			imageDir)
	}
}

// checkComposite is one composite's two readings.
func checkComposite(t *testing.T, root string, c Claim) {
	t.Helper()

	layout, ok := readCompositeLayout(t, root, c)
	if !ok {
		return
	}
	if strings.Join(layout.parts, ",") != strings.Join(c.MadeOf, ",") {
		t.Errorf("%s is made of %v by its claim, and its page draws %v.\n\n"+
			"The claim's order is the page's order, so a reader comparing the "+
			"picture with the row finds the parts where the row says.",
			c.File, c.MadeOf, layout.parts)
		return
	}

	composite, ok := decodePNG(t, root, c.File)
	if !ok {
		return
	}
	parts := make([]image.Image, len(c.MadeOf))
	for i, name := range c.MadeOf {
		if parts[i], ok = decodePNG(t, root, name); !ok {
			return
		}
	}

	// Size. The dpr is not recorded anywhere, so it is read off the height,
	// which is the one dimension with no fractional term in it: every part is
	// drawn exactly layout.height tall, so the capture is (height + 2·padding)
	// CSS px tall and its pixel height must be a whole multiple of that.
	cb := composite.Bounds()
	cssHeight := layout.height + 2*layout.padding
	dpr := float64(cb.Dy()) / cssHeight
	if dpr < 1 || dpr != math.Trunc(dpr) {
		t.Errorf("%s is %d px tall, and its page draws its parts %.0f CSS px "+
			"tall inside %.0f px of padding: %.0f CSS px, which that height is "+
			"not a whole multiple of (%.3f).\n\n"+
			"A composite shot from its page is the page's height at a whole "+
			"dpr. This one was cropped, or taken from a different page.",
			c.File, cb.Dy(), layout.height, layout.padding, cssHeight, dpr)
		return
	}
	widths := make([]float64, len(parts))
	cssWidth := 2*layout.padding + layout.gap*float64(len(parts)-1)
	for i, p := range parts {
		pb := p.Bounds()
		widths[i] = layout.height * float64(pb.Dx()) / float64(pb.Dy())
		cssWidth += widths[i]
	}
	// One CSS pixel of slack: the row's width is a sum of fractional widths,
	// and the capture's clip rounds it once.
	if got := float64(cb.Dx()) / dpr; math.Abs(got-cssWidth) > 1 {
		t.Errorf("%s is %d px wide at dpr %.0f — %.1f CSS px — and its parts, "+
			"drawn whole at %.0f px tall with the page's gap and padding, come "+
			"to %.1f.\n\n"+
			"A composite narrower than its parts has lost some of one; wider, "+
			"it was taken from parts of different proportions than the ones "+
			"in %s now. Either way the picture is not the parts the manifest "+
			"says it is — re-shoot it after its parts (wasm/shots/shoot.sh).",
			c.File, cb.Dx(), dpr, got, layout.height, cssWidth, imageDir)
		return
	}

	// Content, one slot at a time.
	x := layout.padding
	for i, p := range parts {
		pb := p.Bounds()
		scale := float64(pb.Dy()) / layout.height // part pixels per CSS px
		worst, worstAt, squares := 0.0, "", 0
		for by := compositeInset; by+compositeBlock <= layout.height-compositeInset; by += compositeBlock {
			for bx := compositeInset; bx+compositeBlock <= widths[i]-compositeInset; bx += compositeBlock {
				// Inward rounding on both sides, so neither mean includes a
				// pixel from outside the square it stands for.
				got := meanRGB(composite,
					int(math.Ceil((x+bx)*dpr)), int(math.Ceil((layout.padding+by)*dpr)),
					int(math.Floor((x+bx+compositeBlock)*dpr)), int(math.Floor((layout.padding+by+compositeBlock)*dpr)))
				want := meanRGB(p,
					pb.Min.X+int(math.Ceil(bx*scale)), pb.Min.Y+int(math.Ceil(by*scale)),
					pb.Min.X+int(math.Floor((bx+compositeBlock)*scale)), pb.Min.Y+int(math.Floor((by+compositeBlock)*scale)))
				squares++
				for ch := range 3 {
					if d := math.Abs(got[ch] - want[ch]); d > worst {
						worst = d
						worstAt = fmt.Sprintf("the square %.0f,%.0f CSS px into the slot "+
							"(composite %s, part %s)", bx, by, rgbString(got), rgbString(want))
					}
				}
			}
		}
		if squares == 0 {
			t.Errorf("%s: no %.0f px square fits inside %s's slot, so nothing "+
				"about its content was compared.", c.File, compositeBlock, c.MadeOf[i])
		}
		t.Logf("%s: %s over %d squares, worst channel difference %.1f/255 at %s",
			c.File, c.MadeOf[i], squares, worst, worstAt)
		if worst > compositeTolerance {
			t.Errorf("%s does not show %s as it is now: %s differs by %.1f/255 "+
				"in one channel, against a tolerance of %.0f.\n\n"+
				"The composite carries no strings of its own because its parts' "+
				"claims hold them, which is only true while it is a picture of "+
				"those parts. A difference this size is content, not resampling: "+
				"a part re-taken without re-shooting the composite, or parts in a "+
				"different order. Re-shoot it after its parts (wasm/shots/shoot.sh).\n\n"+
				"%s",
				c.File, c.MadeOf[i], worstAt, worst, compositeTolerance, chromeProvenance())
		}
		x += widths[i] + layout.gap
	}
}

// readCompositeLayout reads a composite's page, found through its action
// script's header, into the numbers checkComposite lays the parts out with.
func readCompositeLayout(t *testing.T, root string, c Claim) (compositeLayout, bool) {
	t.Helper()

	script := filepath.Join(scriptDir, strings.TrimSuffix(c.File, filepath.Ext(c.File))+".js")
	raw, err := os.ReadFile(filepath.Join(root, script))
	if err != nil {
		t.Errorf("reading %s: %v", script, err)
		return compositeLayout{}, false
	}
	m := shotHeader.FindSubmatch(raw)
	var head struct {
		Page string `json:"page"`
		Clip string `json:"clip"`
	}
	if m == nil || json.Unmarshal(m[1], &head) != nil || head.Page == "" || head.Clip == "" {
		t.Errorf("%s: the grmob-shot header names no page and clip, so there "+
			"is no layout to hold %s to.", script, c.File)
		return compositeLayout{}, false
	}
	page := filepath.Join(filepath.Dir(scriptDir), head.Page)
	html, err := os.ReadFile(filepath.Join(root, page))
	if err != nil {
		t.Errorf("reading %s: %v", page, err)
		return compositeLayout{}, false
	}

	// The two rules the layout lives in: the clipped container's and its
	// images'. Selected by the clip the header names, so a page with other
	// rules in it is read for the ones that shape the capture.
	sel := regexp.QuoteMeta(head.Clip)
	row := regexp.MustCompile(sel + `\s*\{([^}]*)\}`).FindSubmatch(html)
	img := regexp.MustCompile(sel + `\s+img\s*\{([^}]*)\}`).FindSubmatch(html)
	if row == nil || img == nil {
		t.Errorf("%s has no `%s { … }` and `%s img { … }` rules, which is where "+
			"the composite's gap, padding and part height are read from.", page, head.Clip, head.Clip)
		return compositeLayout{}, false
	}
	length := func(block []byte, prop string) (float64, bool) {
		v := cssLength(prop).FindSubmatch(block)
		if v == nil {
			t.Errorf("%s: no single `%s: <n>px;` in the %s rule. The check lays "+
				"the parts out from it and will not guess.", page, prop, head.Clip)
			return 0, false
		}
		f, err := strconv.ParseFloat(string(v[1]), 64)
		return f, err == nil
	}
	var l compositeLayout
	var okH, okG, okP bool
	l.height, okH = length(img[1], "height")
	l.gap, okG = length(row[1], "gap")
	l.padding, okP = length(row[1], "padding")
	for _, src := range compositeImg.FindAllSubmatch(html, -1) {
		l.parts = append(l.parts, string(src[1]))
	}
	return l, okH && okG && okP
}

// decodePNG reads one image from the image directory.
func decodePNG(t *testing.T, root, name string) (image.Image, bool) {
	t.Helper()
	f, err := os.Open(filepath.Join(root, imageDir, name))
	if err != nil {
		t.Errorf("opening %s/%s: %v", imageDir, name, err)
		return nil, false
	}
	defer f.Close()
	img, _, err := image.Decode(f)
	if err != nil {
		t.Errorf("decoding %s/%s: %v", imageDir, name, err)
		return nil, false
	}
	return img, true
}

// meanRGB is the mean 8-bit colour over [x0,x1)×[y0,y1). An empty rectangle
// reads as black, which checkComposite never asks for: its squares are 12 CSS
// px at a dpr of at least one, and inward rounding takes at most two pixels.
func meanRGB(img image.Image, x0, y0, x1, y1 int) [3]float64 {
	var sum [3]float64
	n := 0
	for y := y0; y < y1; y++ {
		for x := x0; x < x1; x++ {
			r, g, b, _ := img.At(x, y).RGBA()
			sum[0] += float64(r >> 8)
			sum[1] += float64(g >> 8)
			sum[2] += float64(b >> 8)
			n++
		}
	}
	if n == 0 {
		return sum
	}
	return [3]float64{sum[0] / float64(n), sum[1] / float64(n), sum[2] / float64(n)}
}

func rgbString(c [3]float64) string {
	return fmt.Sprintf("rgb(%.0f, %.0f, %.0f)", c[0], c[1], c[2])
}

// imageDir is where the shots live, relative to the repository root. One
// constant because four checks name it and a reader chasing a failure
// wants one answer to "where".
const imageDir = "docs/images"

// scriptDir is where the actions that produce them live. One action
// script per image, named after it: docs/images/todo.png is taken by
// wasm/shots/scripts/todo.js.
const scriptDir = "wasm/shots/scripts"

// checkFilesAndClaimsAgree holds the manifest and the directory to being
// the same set, in both directions.
//
// Both, because the two failures are different and neither implies the
// other. An unclaimed file is a picture nothing holds — which is the
// state this package was written to end, arriving again by somebody
// adding a shot. A claim with no file is a row describing nothing, and it
// is the more confusing of the two: its per-app test goes on passing, so
// the repository reports that a screenshot which does not exist is
// accurate.
func checkFilesAndClaimsAgree(t *testing.T, root string) {
	t.Helper()

	entries, err := os.ReadDir(filepath.Join(root, imageDir))
	if err != nil {
		t.Fatalf("reading %s: %v", imageDir, err)
	}
	onDisk := map[string]bool{}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		onDisk[e.Name()] = true
	}
	// The reaching arm. An empty directory would report every claim as a
	// row describing nothing — a thousand findings in the shape of one
	// wrong reading — and the repository has seven shots in it.
	if len(onDisk) == 0 {
		t.Fatalf("%s holds no files, and this manifest has %d claim(s). "+
			"The reading is wrong rather than the repository being bare.",
			imageDir, len(Claims))
	}

	claimed := map[string]bool{}
	for _, c := range Claims {
		if claimed[c.File] {
			t.Errorf("%s is claimed twice. A file has one state and one set "+
				"of strings; two rows for it means one of them is describing "+
				"a picture that was replaced.", c.File)
		}
		claimed[c.File] = true
		if !onDisk[c.File] {
			t.Errorf("the manifest claims %s/%s and no such file is there.\n\n"+
				"Its per-app test goes on passing while this is true, so the "+
				"repository is reporting that a screenshot it does not have "+
				"is accurate. Either the shot was renamed — say so here — or "+
				"it is gone and this row should go with it.", imageDir, c.File)
		}
	}
	for _, name := range sortedKeys(onDisk) {
		if claimed[name] {
			continue
		}
		t.Errorf("%s/%s is in the tree and nothing claims it.\n\n"+
			"A screenshot with no claim is a fact about a screen written "+
			"down on the day it was true, with nothing that reads it — which "+
			"is what this package exists to stop. Add a row saying which app "+
			"it is of, how it was driven there, and what is legible in it.",
			imageDir, name)
	}
}

// checkRowsAreWellFormed holds each row to being one of the two shapes
// this manifest has: a shot of an app, or a composite of other shots.
//
// The split is not decoration. A composite shows nothing its parts do not,
// so giving it Shows of its own would be asserting the same strings twice
// with one copy free to drift — the exact failure the package is about,
// reintroduced by the manifest itself.
func checkRowsAreWellFormed(t *testing.T, root string) {
	t.Helper()

	byFile := map[string]bool{}
	for _, c := range Claims {
		byFile[c.File] = true
	}

	for _, c := range Claims {
		if c.Composite() {
			if c.Package != "" || len(c.Shows) != 0 || c.Test != "" {
				t.Errorf("%s is a composite and also carries an app, strings "+
					"or a test. A composite is a picture of pictures: what it "+
					"shows is held by the claims it is made of, and a second "+
					"copy here is a claim nothing re-derives.", c.File)
			}
			for _, part := range c.MadeOf {
				if !byFile[part] {
					t.Errorf("%s is made of %s and no claim names that file.",
						c.File, part)
				}
			}
			continue
		}

		// A shot of an app. Every field is load-bearing: the directory is
		// where a person goes, the state is how they get back to the
		// picture, the strings are what is checked, and the test is what
		// checks them.
		if c.Package == "" || c.State == "" || len(c.Shows) == 0 || c.Test == "" {
			t.Errorf("%s is missing one of the four things a claim needs: "+
				"package %q, state %q, %d string(s), test %q.", c.File,
				c.Package, c.State, len(c.Shows), c.Test)
			continue
		}
		if _, err := os.Stat(filepath.Join(root, c.Package)); err != nil {
			t.Errorf("%s says it is a shot of %s and that directory is not "+
				"there: %v", c.File, c.Package, err)
		}
		shown := map[string]bool{}
		for _, s := range c.Shows {
			if shown[s] {
				t.Errorf("%s lists %q twice.", c.File, s)
			}
			shown[s] = true
		}
		for _, q := range c.Quoted {
			if !shown[q] {
				t.Errorf("%s says its caption quotes %q and that is not one "+
					"of the strings the shot is held to.\n\n"+
					"Quoted is the join between the README's prose and a "+
					"rendered tree, and it only joins them where the same "+
					"string is on both sides. A quotation that is not in "+
					"Shows is held against the caption and against nothing "+
					"else, which is the state the caption was already in.",
					c.File, q)
			}
		}
	}
}

// imgTag reads one HTML <img> out of the markdown. The README frames its
// shots in raw HTML rather than in markdown image syntax because it
// centres and sizes them, so this is the only form that has to be read.
var imgTag = regexp.MustCompile(`<img\s+[^>]*>`)

// attr pulls one double-quoted attribute out of a tag.
func attr(tag, name string) string {
	m := regexp.MustCompile(name + `="([^"]*)"`).FindStringSubmatch(tag)
	if m == nil {
		return ""
	}
	return m[1]
}

// checkCaptionsQuoteTheScreen holds the README's alt text to the claims.
//
// # What this is actually for
//
// Alt text is the caption a reader who cannot see the picture gets, and
// this repository's alt text states screen contents: "showing Count: 3",
// "the error 'The two passwords differ'", "0 of 62 lessons opened". Those
// are the same kind of sentence as the six lesson counts that were wrong
// in three directions at once — a fact about a screen, transcribed.
//
// So the quotations are held from both ends. The app's own test says the
// string is really rendered; this says the caption really says it. A
// wording change then fails on the side that changed rather than
// silently making the other side's copy stale.
//
// # Why both directions of the reference are checked
//
// An image the README does not show is one nobody is served by keeping
// current, and a README image with no claim is the original problem
// walking back in through the document rather than through the directory.
func checkCaptionsQuoteTheScreen(t *testing.T, root string) {
	t.Helper()

	raw, err := os.ReadFile(filepath.Join(root, "README.md"))
	if err != nil {
		t.Fatalf("reading README.md: %v", err)
	}
	byFile := map[string]Claim{}
	for _, c := range Claims {
		byFile[c.File] = c
	}

	shown := map[string]bool{}
	for _, tag := range imgTag.FindAllString(string(raw), -1) {
		src := attr(tag, "src")
		if !strings.HasPrefix(src, imageDir+"/") {
			// An image from somewhere else entirely is not this
			// manifest's business; nothing in the README has one today,
			// and a badge or an external asset arriving should not fail
			// a check about screenshots.
			continue
		}
		name := strings.TrimPrefix(src, imageDir+"/")
		claim, ok := byFile[name]
		if !ok {
			t.Errorf("README.md shows %s and nothing claims it.\n\n"+
				"Every screenshot in the README is a statement about a "+
				"screen, and one with no claim is a statement nothing "+
				"re-derives. Add a row to Claims.", src)
			continue
		}
		shown[name] = true

		alt := attr(tag, "alt")
		if alt == "" {
			t.Errorf("README.md shows %s with no alt text.", src)
			continue
		}
		for _, q := range claim.Quoted {
			if !strings.Contains(alt, q) {
				t.Errorf("README.md's caption for %s does not quote %q.\n\n"+
					"caption: %s\n\n"+
					"That string is what the claim and the caption have in "+
					"common — it is asserted against a rendered tree by %s "+
					"and stated here in prose. If the screen's wording "+
					"changed, both ends move together; if the caption was "+
					"simply reworded, drop the entry from Quoted rather than "+
					"leaving a join that joins nothing.",
					src, q, alt, claim.Test)
			}
		}
	}
	for _, c := range Claims {
		if !shown[c.File] {
			t.Errorf("%s/%s is claimed and the README does not show it.\n\n"+
				"The cost of a shot is keeping it true, and the reason to pay "+
				"it is that somebody reads it. A picture nothing displays is "+
				"neither.", imageDir, c.File)
		}
	}
}

// sortedKeys is the set in a stable order, so that findings from two runs
// can be diffed against each other. Map order cannot be.
func sortedKeys(m map[string]bool) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

// shotHeader is the one line of JSON every action script opens with,
// saying which app it drives and how big the frame is. Parsed here rather
// than merely looked for, because the half worth checking is the app name:
// a script that drives todoapp under a claim that says the picture is of
// signup is a harness and a manifest that disagree about what the file
// is a picture of, and neither of them would notice.
var shotHeader = regexp.MustCompile(`(?m)^\s*//\s*grmob-shot:\s*(\{.*\})\s*$`)

// checkEveryClaimIsTakeable holds the manifest against the harness that
// produces the images.
//
// # Why this is worth a check of its own
//
// The first version of this harness was deleted rather than committed, and
// re-taking a screenshot then meant rebuilding an import-swap script, a
// host page and a CDP driver out of a session document. What replaced it
// is wasm/shots, and what stops it decaying the same way is that a claim
// with no way to re-take it is exactly as unpinned as the picture was: the
// test in the app's package would still say the strings are on the screen,
// and nobody could produce the image that shows them.
//
// Both directions again. A claim with no script cannot be re-taken; a
// script with no claim takes a picture nothing holds, which is where this
// whole mechanism came in.
func checkEveryClaimIsTakeable(t *testing.T, root string) {
	t.Helper()

	entries, err := os.ReadDir(filepath.Join(root, scriptDir))
	if err != nil {
		t.Fatalf("reading %s: %v", scriptDir, err)
	}
	scripts := map[string]bool{}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".js") {
			continue
		}
		scripts[strings.TrimSuffix(e.Name(), ".js")] = true
	}
	if len(scripts) == 0 {
		t.Fatalf("%s holds no action scripts, and this manifest has %d "+
			"claim(s). The reading is wrong rather than the harness being "+
			"empty.", scriptDir, len(Claims))
	}

	claimed := map[string]bool{}
	for _, c := range Claims {
		base := strings.TrimSuffix(c.File, filepath.Ext(c.File))
		claimed[base] = true
		if !scripts[base] {
			t.Errorf("nothing takes %s/%s: %s/%s.js is not there.\n\n"+
				"A picture that cannot be re-taken is a picture that will "+
				"be wrong and stay wrong, because the cost of correcting it "+
				"is reconstructing a harness. Add the script, or drop the "+
				"claim and the image with it.", imageDir, c.File, scriptDir, base)
			continue
		}
		checkScriptDrivesTheClaimedApp(t, root, base, c)
	}
	for _, name := range sortedKeys(scripts) {
		if !claimed[name] {
			t.Errorf("%s/%s.js takes a picture nothing claims.\n\n"+
				"Either it writes %s/%s.png and that image needs a row here, "+
				"or it is left over from a shot that has gone.",
				scriptDir, name, imageDir, name)
		}
	}
}

// checkScriptDrivesTheClaimedApp holds one script's header to its claim.
func checkScriptDrivesTheClaimedApp(t *testing.T, root, base string, c Claim) {
	t.Helper()

	path := filepath.Join(scriptDir, base+".js")
	raw, err := os.ReadFile(filepath.Join(root, path))
	if err != nil {
		t.Errorf("reading %s: %v", path, err)
		return
	}
	m := shotHeader.FindSubmatch(raw)
	if m == nil {
		t.Errorf("%s has no `// grmob-shot: {…}` header, so nothing in it "+
			"says which app it drives or how big the frame is. The driver "+
			"refuses such a script; this says so at test time instead.", path)
		return
	}
	var head struct {
		App  string `json:"app"`
		Page string `json:"page"`
	}
	if err := json.Unmarshal(m[1], &head); err != nil {
		t.Errorf("%s: the grmob-shot header is not JSON: %v", path, err)
		return
	}
	if c.Composite() {
		if head.Page == "" {
			t.Errorf("%s claims to be a composite and %s mounts an app "+
				"(%q) rather than naming a page of its own.", c.File, path, head.App)
		}
		return
	}
	// The claim says "examples/todoapp" and the header says "todoapp":
	// the same word with the directory in front of it, which is the whole
	// point of choosing the app by its directory name in the host.
	if want := strings.TrimPrefix(c.Package, "examples/"); head.App != want {
		t.Errorf("%s says docs/images/%s is a shot of %s, and %s drives "+
			"%q.\n\n"+
			"One of the two is a picture of a different app from the one it "+
			"is held to, and the test that asserts its strings would go on "+
			"passing either way — it renders the app the CLAIM names.",
			"the manifest", c.File, c.Package, path, head.App)
	}
}

// --------------------------------------------------------------------------
// Which Chrome compositeTolerance was measured on
// --------------------------------------------------------------------------

// chromeCandidates is where a Chrome might be, in the order wasm/verify's
// browser.mjs and wasm/shots/shot.mjs look: GRMOB_CHROME first, so a machine
// that shoots with an unusual install is read the same way here.
func chromeCandidates() []string {
	var c []string
	if env := os.Getenv("GRMOB_CHROME"); env != "" {
		c = append(c, env)
	}
	return append(c,
		"/Applications/Google Chrome.app/Contents/MacOS/Google Chrome",
		"/Applications/Chromium.app/Contents/MacOS/Chromium",
		"/usr/bin/google-chrome",
		"/usr/bin/chromium",
		"/usr/bin/chromium-browser",
	)
}

// chromeVersion matches the first dotted version in `--version` output:
// "Google Chrome 152.0.7843.41" and "Chromium 152.0.7843.41 built on …" alike.
// At least one dot is required so a digit in a product name is not a version.
var chromeVersion = regexp.MustCompile(`\b(\d+)\.\d+(?:\.\d+)*\b`)

// localChromeMajor is the major version of the first Chrome found, with its
// path. Memoised: the composite check asks once up front and again in any
// failure message, and `--version` on macOS loads the browser's framework,
// which is not free.
//
// Errors (no Chrome, a binary that will not run, unparseable output) are
// returned rather than reported: the version is context for a reading, and a
// machine with no Chrome can still check committed PNGs.
//
// sync.OnceValue over a struct because sync.OnceValues carries only two
// results, and callers want all three.
func localChromeMajor() (major int, path string, err error) {
	r := localChromeRead()
	return r.major, r.path, r.err
}

type chromeRead struct {
	major int
	path  string
	err   error
}

var localChromeRead = sync.OnceValue(func() chromeRead {
	major, path, err := chromeMajorAt(chromeCandidates(), 10*time.Second)
	return chromeRead{major, path, err}
})

// chromeMajorAt runs `--version` on the first existing candidate and parses
// its major version. Only the first existing one is asked, as the harnesses
// launch only that one: a second install is not the Chrome that shot.
func chromeMajorAt(candidates []string, timeout time.Duration) (major int, path string, err error) {
	for _, path := range candidates {
		if _, err := os.Stat(path); err != nil {
			continue
		}
		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		out, err := exec.CommandContext(ctx, path, "--version").Output()
		cancel()
		if err != nil {
			return 0, path, fmt.Errorf("%s --version: %w", path, err)
		}
		major, err := parseChromeMajor(string(out))
		return major, path, err
	}
	return 0, "", fmt.Errorf("no Chrome at any of %v", candidates)
}

func parseChromeMajor(out string) (int, error) {
	m := chromeVersion.FindStringSubmatch(out)
	if m == nil {
		return 0, fmt.Errorf("no version in %q", strings.TrimSpace(out))
	}
	return strconv.Atoi(m[1])
}

// chromeProvenance is the sentence a log or a failure carries about which
// Chrome compositeTolerance was measured on and which one this machine has.
func chromeProvenance() string {
	local, path, err := localChromeMajor()
	switch {
	case err != nil:
		return fmt.Sprintf("compositeTolerance was measured on Chrome %d; this "+
			"machine's Chrome could not be read (%v).", compositeMeasuredOnChrome, err)
	case local == compositeMeasuredOnChrome:
		return fmt.Sprintf("compositeTolerance was measured on Chrome %d, the "+
			"major version at %s.", compositeMeasuredOnChrome, path)
	default:
		return fmt.Sprintf("compositeTolerance was measured on Chrome %d, and "+
			"%s is Chrome %d. If the composite was re-shot with this Chrome and "+
			"the reading is near the tolerance rather than far past it, its "+
			"resampling may differ: re-take the table above compositeTolerance "+
			"before moving the number.",
			compositeMeasuredOnChrome, path, local)
	}
}

// The parse is the one piece of the version read that can be wrong without a
// Chrome to disagree with it, so it is held to the shapes the harnesses meet.
func TestParseChromeMajor(t *testing.T) {
	for out, want := range map[string]int{
		"Google Chrome 152.0.7843.41\n":                 152,
		"Chromium 131.0.6778.85 built on Debian 12.8\n": 131,
		"Google Chrome for Testing 99.0.4844.51 \n":     99,
	} {
		if got, err := parseChromeMajor(out); err != nil || got != want {
			t.Errorf("parseChromeMajor(%q) = %d, %v; want %d", out, got, err, want)
		}
	}
	if _, err := parseChromeMajor("Google Chrome\n"); err == nil {
		t.Error("parseChromeMajor accepted output with no version in it")
	}
	if _, _, err := chromeMajorAt([]string{filepath.Join(t.TempDir(), "absent")}, time.Second); err == nil {
		t.Error("chromeMajorAt found a Chrome in an empty directory")
	}
}
