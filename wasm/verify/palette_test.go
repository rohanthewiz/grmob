package main

import (
	"fmt"
	"math"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"testing"

	"github.com/rohanthewiz/grmob/core"
	"github.com/rohanthewiz/grmob/internal/palette"
)

// palette.mjs is the bundled palettes as browser.mjs paints them, and this is
// what keeps it from being a transcription.
//
// # Why the browser needs a copy at all
//
// The same reason inputtype_test.go's table does: the runtime draws in a
// browser and cannot call into Go while it does. browser.mjs mounts a tree of
// swatches, screenshots them and reads the pixels back, and the hexes it
// paints have to come from somewhere. A hand-copied palette would be a set of
// colours that were the theme's on the day somebody typed them.
//
// # What the browser adds that Go cannot
//
// components/variant_test.go's census is arithmetic over strings. It proves
// #89898E is 3.12:1 on #F2F2F7 and it cannot prove that either of those
// colours ever reaches a screen. Everything between the palette and the pixel
// — the runtime's style mapping, CSS shorthand parsing, alpha, a hairline
// border antialiased into a tint, the browser's colour management — is
// invisible to it, and every one of those makes the number true and the
// control invisible.
//
// So the ratio travels with the row: the browser never recomputes it (a
// second WCAG implementation in one module is exactly what a contrast floor
// cannot survive), and a pixel that is not the stated hex is reported
// alongside the number the census believed about it.
var mjsPaletteRow = regexp.MustCompile(
	`\{\s*theme:\s*"([^"]*)",\s*kind:\s*"([^"]*)",\s*what:\s*"([^"]*)",\s*hex:\s*"([^"]*)",\s*ratio:\s*([0-9.]+)\s*\}`)

// paletteRow is one row of the table, in both languages.
type paletteRow struct {
	theme, kind, what, hex string
	ratio                  float64
}

func (r paletteRow) String() string {
	return fmt.Sprintf(`{ theme: %q, kind: %q, what: %q, hex: %q, ratio: %.2f }`,
		r.theme, r.kind, r.what, r.hex, r.ratio)
}

// wantPaletteRows is the table as core and internal/palette state it: one
// `role` row per theme carrying the ControlBorder tone, then one `backdrop`
// row per fill that tone can land on, carrying the census's ratio.
//
// Sorted by theme then by the order palette.Backdrops returns, which is the
// order a reader of the .mjs file sees and the order the swatches are painted
// in. Deriving the order rather than sorting it alphabetically keeps the two
// files diffable by eye when a theme is added.
func wantPaletteRows(t *testing.T) []paletteRow {
	t.Helper()

	byName := core.BundledThemes()
	names := make([]string, 0, len(byName))
	for name := range byName {
		names = append(names, name)
	}
	sort.Strings(names)

	var out []paletteRow
	for _, name := range names {
		theme := byName[name]
		tone := theme.Colors.ControlBorderColor()
		edge, ok := palette.Luminance(tone)
		if !ok {
			t.Fatalf("%s: ControlBorder %q does not parse", name, tone)
		}
		out = append(out, paletteRow{theme: name, kind: "role",
			what: "ControlBorder", hex: tone})
		for _, b := range palette.Backdrops(theme) {
			lum, ok := palette.Luminance(b.Hex)
			if !ok {
				t.Fatalf("%s: backdrop %s = %q does not parse", name, b.What, b.Hex)
			}
			out = append(out, paletteRow{theme: name, kind: "backdrop",
				what: b.What, hex: b.Hex,
				ratio: math.Round(palette.Ratio(edge, lum)*100) / 100})
		}
	}
	return out
}

func paletteMJS(t *testing.T) string {
	t.Helper()
	src, err := os.ReadFile(filepath.Join(".", "palette.mjs"))
	if err != nil {
		t.Fatalf("reading palette.mjs: %v", err)
	}
	return string(src)
}

// The browser's copy of the palettes is core's.
//
// Both directions fail, and they fail differently. A missing row is a theme
// (or a backdrop) the browser pass has never painted, which is the silence
// this whole file exists to break — the palette would go on being measured by
// arithmetic alone and nothing would say so. A surplus row is a check painting
// a colour nothing ships, which passes forever and means nothing.
func TestTheBrowsersPaletteTableIsCores(t *testing.T) {
	src := paletteMJS(t)

	var got []paletteRow
	for _, m := range mjsPaletteRow.FindAllStringSubmatch(src, -1) {
		ratio, err := strconv.ParseFloat(m[5], 64)
		if err != nil {
			t.Errorf("palette.mjs: row %s has an unreadable ratio: %v", m[0], err)
			continue
		}
		got = append(got, paletteRow{theme: m[1], kind: m[2], what: m[3],
			hex: m[4], ratio: ratio})
	}
	want := wantPaletteRows(t)

	if len(got) != len(want) {
		t.Errorf("palette.mjs has %d rows, core has %d", len(got), len(want))
	}
	for i := 0; i < len(got) && i < len(want); i++ {
		if got[i] != want[i] {
			t.Errorf("palette.mjs row %d is\n\t%s\ncore says\n\t%s", i, got[i], want[i])
		}
	}
	if len(got) < len(want) {
		for _, row := range want[len(got):] {
			t.Errorf("palette.mjs has no row for %s/%s — the browser pass paints every "+
				"row in that file and nothing else, so this pair has never been "+
				"looked at:\n\t%s", row.theme, row.what, row)
		}
	}
	for _, row := range got[min(len(got), len(want)):] {
		t.Errorf("palette.mjs carries a row core does not have:\n\t%s\nThe browser is "+
			"painting a colour nothing ships, which passes whatever it measures", row)
	}

	// The whole table, ready to paste, when it has drifted. Regenerating by
	// hand from three themes and seven backdrops each is how a transcription
	// gets made in the first place.
	if t.Failed() {
		var b []byte
		for _, row := range want {
			b = append(b, "    "+row.String()+",\n"...)
		}
		t.Logf("palette.mjs's PALETTES should be:\n%s", b)
	}
}
