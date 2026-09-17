package comps

import (
	"strings"
	"testing"

	"github.com/rohanthewiz/grmob/core"
	"github.com/rohanthewiz/grmob/internal/qr"
)

// qrRects reads the rectangles back out of a drawn module path. The path is a
// sequence of move/line/line/line/close subpaths, one per run of dark modules,
// so a test can recover exactly what would be painted rather than asserting on
// a coordinate list it would have to keep in step by hand.
func qrRects(t *testing.T, shape *core.Node) [][4]float64 {
	t.Helper()
	d, ok := shape.Props["d"].([]float64)
	if !ok {
		t.Fatalf("shape has no path: %#v", shape.Props)
	}
	var out [][4]float64
	for i := 0; i < len(d); {
		if d[i] != core.PathMove {
			t.Fatalf("subpath at %d starts with opcode %v, want a move", i, d[i])
		}
		x, y := d[i+1], d[i+2]
		i += 3
		var maxX, maxY = x, y
		for i < len(d) && d[i] == core.PathLine {
			maxX, maxY = max(maxX, d[i+1]), max(maxY, d[i+2])
			i += 3
		}
		if i >= len(d) || d[i] != core.PathClose {
			t.Fatalf("subpath starting at (%v,%v) is not closed", x, y)
		}
		i++
		out = append(out, [4]float64{x, y, maxX - x, maxY - y})
	}
	return out
}

// qrDrawnGrid turns those rectangles back into a module grid, so the drawing
// can be compared with the encoder's own output module for module.
func qrDrawnGrid(t *testing.T, n *core.Node, side, quiet int) map[[2]int]bool {
	t.Helper()
	grid := map[[2]int]bool{}
	for _, r := range qrRects(t, n.Children[1]) {
		if r[3] != 1 {
			t.Fatalf("a run is %v modules tall, want 1", r[3])
		}
		for i := 0; i < int(r[2]); i++ {
			x, y := int(r[0])-quiet+i, int(r[1])-quiet
			if x < 0 || y < 0 || x >= side || y >= side {
				t.Fatalf("a module at (%d,%d) falls outside the %d-module symbol", x, y, side)
			}
			grid[[2]int{x, y}] = true
		}
	}
	return grid
}

func TestQRCodeIsACanvasOfABackgroundAndOnePathOfModules(t *testing.T) {
	const data = "cats://pair?t=9f2c1a"
	ctx, n := renderDebug(t, QRCode{Data: data, Label: "Pairing code"})
	defer ctx.Close()

	if n.Type != "Canvas" {
		t.Fatalf("root = %q, want a Canvas", n.Type)
	}
	if len(n.Children) != 2 {
		t.Fatalf("%d shapes, want two: the light field and one path of every dark module", len(n.Children))
	}

	code, err := qr.Encode(data, qr.Medium)
	if err != nil {
		t.Fatal(err)
	}
	wantView := float64(code.Size + 2*qrDefaultQuiet)
	if n.Props["vw"] != wantView || n.Props["vh"] != wantView {
		t.Errorf("viewBox = %v×%v, want %v square: one unit per module plus the quiet zone",
			n.Props["vw"], n.Props["vh"], wantView)
	}
	if n.Props["scale"] != string(core.CanvasFit) {
		t.Errorf("scale = %v, want fit: a QR Code that is stretched on one axis is unreadable", n.Props["scale"])
	}
	if n.Style.Width != "160px" || n.Style.Height != "160px" {
		t.Errorf("box = %s×%s, want the 160px default square", n.Style.Width, n.Style.Height)
	}

	// The background must cover the whole viewBox, quiet zone included: the
	// margin is only a margin if it is light.
	bg := qrRects(t, n.Children[0])
	if len(bg) != 1 || bg[0] != [4]float64{0, 0, wantView, wantView} {
		t.Errorf("background = %v, want one rectangle over the whole %v-unit box", bg, wantView)
	}

	// Every module the encoder set is drawn, and nothing else is.
	drawn := qrDrawnGrid(t, n, code.Size, qrDefaultQuiet)
	for y := 0; y < code.Size; y++ {
		for x := 0; x < code.Size; x++ {
			if code.Dark(x, y) != drawn[[2]int{x, y}] {
				t.Fatalf("module (%d,%d): encoder says dark=%v, the drawing says %v",
					x, y, code.Dark(x, y), drawn[[2]int{x, y}])
			}
		}
	}
}

// The run merging is the one piece of arithmetic in the drawing, so it is
// checked directly: no two rectangles in a row may touch, or a run was cut
// short, and every rectangle is exactly one module tall.
func TestQRCodeMergesRunsWithinARow(t *testing.T) {
	ctx, n := renderDebug(t, QRCode{Data: strings.Repeat("merge runs ", 12), Level: ECLow})
	defer ctx.Close()

	rects := qrRects(t, n.Children[1])
	if len(rects) == 0 {
		t.Fatal("no modules drawn")
	}
	ends := map[float64]float64{} // row y -> the x the previous run ended at
	merged := 0
	for _, r := range rects {
		if r[3] != 1 {
			t.Fatalf("rectangle %v is not one module tall", r)
		}
		if r[2] > 1 {
			merged++
		}
		if end, seen := ends[r[1]]; seen && r[0] <= end {
			t.Fatalf("row %v: a run starts at %v but the previous one ran to %v — they should have been one rectangle",
				r[1], r[0], end)
		}
		ends[r[1]] = r[0] + r[2]
	}
	if merged == 0 {
		t.Error("no run was merged; a symbol always has adjacent dark modules (the finders alone have seven in a row)")
	}
}

func TestQRCodeQuietZone(t *testing.T) {
	code, err := qr.Encode("quiet", qr.Medium)
	if err != nil {
		t.Fatal(err)
	}

	// The default is the specification's four modules on every side, and the
	// first dark module — the top-left finder's corner — sits at exactly that
	// offset.
	ctx, n := renderDebug(t, QRCode{Data: "quiet"})
	defer ctx.Close()
	if got := n.Props["vw"]; got != float64(code.Size+8) {
		t.Errorf("viewBox = %v, want %d: %d modules plus four each side", got, code.Size+8, code.Size)
	}
	if first := qrRects(t, n.Children[1])[0]; first[0] != 4 || first[1] != 4 {
		t.Errorf("first module at (%v,%v), want (4,4)", first[0], first[1])
	}

	// A negative Quiet opts out, for a caller supplying the margin from the
	// layout around the code.
	ctx2, bare := renderDebug(t, QRCode{Data: "quiet", Quiet: -1})
	defer ctx2.Close()
	if got := bare.Props["vw"]; got != float64(code.Size) {
		t.Errorf("Quiet -1: viewBox = %v, want the bare %d modules", got, code.Size)
	}
	if first := qrRects(t, bare.Children[1])[0]; first[0] != 0 || first[1] != 0 {
		t.Errorf("Quiet -1: first module at (%v,%v), want the origin", first[0], first[1])
	}

	ctx3, wide := renderDebug(t, QRCode{Data: "quiet", Quiet: 10})
	defer ctx3.Close()
	if got := wide.Props["vw"]; got != float64(code.Size+20) {
		t.Errorf("Quiet 10: viewBox = %v, want %d", got, code.Size+20)
	}
}

// The property that matters more than which colours are picked: whatever the
// theme, the modules are darker than the field they sit on, by a wide margin.
// A themed QR Code that came out light-on-dark would look fine and scan for
// nobody.
func TestQRCodeIsDarkOnLightInEveryBundledTheme(t *testing.T) {
	for name, theme := range core.BundledThemes() {
		ctx := core.NewContext()
		themed := ctx.WithTheme(theme)
		themed.BeginRenderPass()
		n := QRCode{Data: "contrast"}.Render(themed)
		themed.EndRenderPass()
		ctx.Close()

		dark, _ := n.Children[1].Props["fill"].(string)
		light, _ := n.Children[0].Props["fill"].(string)
		dl, okD := relativeLuminance(dark)
		ll, okL := relativeLuminance(light)
		if !okD || !okL {
			t.Errorf("%s: unreadable colours %q on %q", name, dark, light)
			continue
		}
		if ll <= dl {
			t.Errorf("%s: modules %q (luminance %.3f) on %q (%.3f) — a QR Code must be dark on light",
				name, dark, dl, light, ll)
		}
		// WCAG's contrast formula, which is also a fair proxy for what a
		// camera's binarizer needs. 7:1 is the level the widget's own choice
		// of colours is meant to keep clear of.
		if ratio := (ll + 0.05) / (dl + 0.05); ratio < 7 {
			t.Errorf("%s: contrast %.1f:1 between %q and %q, want 7:1 or better", name, ratio, dark, light)
		}
	}
}

func TestQRCodeLabelNamesTheCodeAndNeverReadsTheData(t *testing.T) {
	const data = "https://example.com/a-very-long-link-nobody-wants-spelled-out"
	ctx, n := renderDebug(t, QRCode{Data: data})
	defer ctx.Close()
	if n.Style.AccessibilityLabel != "QR code" {
		t.Errorf("label = %q, want the default %q", n.Style.AccessibilityLabel, "QR code")
	}
	if n.Style.AccessibilityRole != core.RoleImg {
		t.Errorf("role = %q, want %q", n.Style.AccessibilityRole, core.RoleImg)
	}
	if n.Style.AccessibilityHidden {
		t.Error("a named code is an image, not decoration")
	}

	ctx2, named := renderDebug(t, QRCode{Data: data, Label: "Scan to pair this device"})
	defer ctx2.Close()
	if named.Style.AccessibilityLabel != "Scan to pair this device" {
		t.Errorf("label = %q, want the caller's", named.Style.AccessibilityLabel)
	}
	if strings.Contains(named.Style.AccessibilityLabel, "example.com") {
		t.Error("the data must never reach the label")
	}
}

func TestQRCodeStyleIsAppliedLast(t *testing.T) {
	ctx, n := renderDebug(t, QRCode{
		Data:  "style",
		Size:  200,
		Style: []core.StyleProp{core.Width("100%"), core.Margin(8)},
	})
	defer ctx.Close()
	if n.Style.Width != "100%" {
		t.Errorf("width = %q, want the caller's override of the 200px default", n.Style.Width)
	}
	if n.Style.Height != "200px" {
		t.Errorf("height = %q, want the Size the caller asked for", n.Style.Height)
	}
	if n.Style.Margin.Top != 8 {
		t.Errorf("margin = %+v, want the caller's 8", n.Style.Margin)
	}
}

func TestQRCodeLevelSelectsTheSymbol(t *testing.T) {
	// A higher level spends more of the symbol on redundancy, so the same data
	// needs a bigger symbol. That is the visible consequence of the field, and
	// it is what says the level reaches the encoder at all.
	const data = "the same forty-odd bytes at four levels!!"
	var sizes []float64
	for _, level := range []ECLevel{ECLow, ECMedium, ECQuartile, ECHigh} {
		ctx, n := renderDebug(t, QRCode{Data: data, Level: level})
		sizes = append(sizes, n.Props["vw"].(float64))
		ctx.Close()
	}
	for i := 1; i < len(sizes); i++ {
		if sizes[i] < sizes[i-1] {
			t.Errorf("symbol sizes %v are not non-decreasing with the level", sizes)
			break
		}
	}
	if sizes[0] >= sizes[3] {
		t.Errorf("ECLow and ECHigh both give a %v-unit symbol; the level is not reaching the encoder", sizes[0])
	}

	// The zero value is Medium, not Low: the default must be the middle one.
	ctxA, def := renderDebug(t, QRCode{Data: data})
	defer ctxA.Close()
	ctxB, med := renderDebug(t, QRCode{Data: data, Level: ECMedium})
	defer ctxB.Close()
	if def.Props["vw"] != med.Props["vw"] {
		t.Errorf("the zero level gives a %v-unit symbol and ECMedium %v", def.Props["vw"], med.Props["vw"])
	}
}

// Data that cannot fit draws nothing rather than a truncated symbol, because a
// truncated symbol scans — it just scans to the wrong thing.
func TestQRCodeTooLongDrawsAnEmptyBoxAndReportsAConcern(t *testing.T) {
	core.SetDebugMode(true)
	core.ClearConcerns()
	defer func() { core.SetDebugMode(false); core.ClearConcerns() }()

	ctx := core.NewContext()
	ctx.BeginRenderPass()
	n := QRCode{Data: strings.Repeat("x", 3000), Size: 120}.Render(ctx)
	ctx.EndRenderPass()
	defer ctx.Close()

	if n.Type == "Canvas" {
		t.Fatal("a symbol was drawn for data no QR Code can hold")
	}
	if n.Style.Width != "120px" || n.Style.Height != "120px" {
		t.Errorf("box = %s×%s, want the 120px the caller asked for kept, so the screen does not reflow when the data is fixed",
			n.Style.Width, n.Style.Height)
	}
	found := false
	for _, c := range core.Concerns() {
		if c.Kind == ConcernQRDataTooLong {
			found = true
			if !strings.Contains(c.Detail, "3000") {
				t.Errorf("concern detail %q does not say how long the data was", c.Detail)
			}
		}
	}
	if !found {
		t.Errorf("no %s concern; the failure would be silent", ConcernQRDataTooLong)
	}
}

// An empty Data is a legal symbol — all padding — rather than a special case,
// which keeps the zero value from being a hole a caller falls into.
func TestQRCodeZeroValueRendersTheSmallestSymbol(t *testing.T) {
	ctx, n := renderDebug(t, QRCode{})
	defer ctx.Close()
	if n.Type != "Canvas" {
		t.Fatalf("root = %q, want a Canvas even with no data", n.Type)
	}
	if got := n.Props["vw"]; got != float64(21+2*qrDefaultQuiet) {
		t.Errorf("viewBox = %v, want a version-1 symbol (21 modules) plus its quiet zone", got)
	}
}

func BenchmarkQRCodeRender(b *testing.B) {
	ctx := core.NewContext()
	defer ctx.Close()
	v := QRCode{Data: "cats://pair?t=9f2c1a&host=studio.local&exp=1789123456"}
	for b.Loop() {
		ctx.BeginRenderPass()
		v.Render(ctx)
		ctx.EndRenderPass()
	}
}
