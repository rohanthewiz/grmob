package core

import (
	"fmt"
	"testing"
)

// What the three debug guards buy, measured.
//
// # The gap this closes
//
// Each of the three debug checks is written the same way — an IsDebugMode test
// at the top, before any work:
//
//	AuditTree            core/a11y_audit.go   the accessibility + placement walk
//	Context.EndRenderPass core/debug.go       the hook-cursor audit
//	renderAll            core/layout.go       the duplicate-key check
//
// And each of those three guards is *unobservable*. upsertConcern carries its
// own IsDebugMode test as a backstop, so deleting any of the three leaves the
// behaviour identical: nothing is recorded, Concerns() stays empty, and every
// behavioural test in this package goes on passing. TestTheAuditIsSilentWithDebugModeOff
// says so in as many words, and says that it is therefore asserting less than
// it looks like it is.
//
// What the guards buy is cost. A production app renders many times a second,
// and without them every frame would walk its whole tree twice and build
// detail strings for findings that are then thrown away by the backstop. That
// is the entire argument for the three lines, and nothing measured it.
//
// # Why allocations rather than a benchmark
//
// A benchmark reports a number and passes whatever the number is. The item
// this closes asked for one, and a benchmark alone would have re-created the
// problem one level up: a measurement nobody is held to.
//
// testing.AllocsPerRun is the assertable form of the same measurement. It is
// deterministic — GOMAXPROCS is pinned to 1 for the duration, and there is a
// warm-up run before counting — so an exact count is a legitimate assertion
// rather than a threshold somebody will have to keep raising. And allocation
// is the right proxy for what these checks actually cost: all three build maps
// and format strings, none of them does arithmetic.
//
// The benchmarks below are still here, for the case the numbers are wanted
// rather than the verdict. They share these fixtures, so neither is a second
// description of the subject.
//
// # What each assertion can catch
//
// Deleting the guard in AuditTree or EndRenderPass takes their counts off zero,
// which is caught outright. The third is different in kind and needs the
// differential below: renderAll allocates on any path, because rendering is
// what it does.

// costTree is a list-shaped tree: enough rows that a walk is measurable, with
// every property the audit looks at present on each row — a role, an id, a key,
// and children of two node types.
func costTree(rows int) *Node {
	kids := make([]*Node, 0, rows)
	for i := 0; i < rows; i++ {
		kids = append(kids, &Node{
			Type: "Row",
			Key:  fmt.Sprintf("r%d", i),
			Style: &Style{
				AccessibilityRole: RoleListItem,
				AccessibilityID:   fmt.Sprintf("row-%d", i),
			},
			Children: []*Node{
				{Type: "Text", Props: map[string]any{"content": "Sermons"}},
				{Type: "Button", Props: map[string]any{"onClick": "cb_0"}},
			},
		})
	}
	return &Node{
		Type:     "Column",
		Style:    &Style{AccessibilityRole: RoleList},
		Children: kids,
	}
}

// costViews are the views renderAll is measured over. A plain node per view, so
// what the measurement moves with is renderAll's own work rather than the
// components'.
type costView struct{ key string }

func (v costView) Render(ctx *Context) *Node { return &Node{Type: "Row", Key: v.key} }

func costViewList(n int) []View {
	out := make([]View, 0, n)
	for i := 0; i < n; i++ {
		out = append(out, costView{key: fmt.Sprintf("r%d", i)})
	}
	return out
}

// costContext is a context that has been through a pass: eight hook slots
// allocated and a cursor that stopped short of them, which is the shape
// auditCursor has the most work to do on — it copies the slot slice and formats
// a finding.
func costContext() *Context {
	ctx := NewContext()
	NewState(ctx, 0)
	NewState(ctx, "")
	NewState(ctx, 0.0)
	NewState(ctx, false)
	ctx.Cursor = 2
	return ctx
}

// debugOff puts the package back in its default state after a case that turned
// debug mode on. Every test here measures both settings, so none of them can
// leave the flag where it found it without saying so.
func debugOff(t *testing.T) {
	t.Helper()
	t.Cleanup(func() {
		SetDebugMode(false)
		ClearConcerns()
	})
	SetDebugMode(false)
	ClearConcerns()
}

// The accessibility audit is free when debug mode is off.
//
// The `on` half is not decoration: an assertion that a guarded call allocates
// nothing is vacuous if the thing behind the guard allocates nothing either,
// and would go on passing after the work it guards was deleted. So the pair is
// the claim — off is zero, on is not.
func TestTheAccessibilityAuditCostsNothingWithDebugModeOff(t *testing.T) {
	debugOff(t)
	tree := costTree(40)

	off := testing.AllocsPerRun(50, func() { AuditTree(tree) })
	if off != 0 {
		t.Errorf("AuditTree allocates %.0f times per call with debug mode off, want 0 — "+
			"the guard in AuditTree is gone or no longer covers the walk, and every "+
			"frame of every production app is paying for a tree walk whose findings "+
			"upsertConcern then throws away", off)
	}

	SetDebugMode(true)
	on := testing.AllocsPerRun(50, func() { AuditTree(tree) })
	SetDebugMode(false)
	ClearConcerns()
	if on == 0 {
		t.Error("AuditTree allocates nothing with debug mode ON either — the guard " +
			"above is guarding nothing, so this test proves nothing about it")
	}
	t.Logf("AuditTree over %d rows: %.0f allocs on, %.0f off", 40, on, off)
}

// The hook-cursor audit is free when debug mode is off.
//
// Its cost is smaller than the audit's — a slice copy and, when there is drift,
// one Sprintf — and it is paid at the end of every render pass of every context
// in the tree, which is where the multiplier is.
func TestTheCursorAuditCostsNothingWithDebugModeOff(t *testing.T) {
	debugOff(t)
	ctx := costContext()

	off := testing.AllocsPerRun(50, func() { ctx.EndRenderPass() })
	if off != 0 {
		t.Errorf("EndRenderPass allocates %.0f times per call with debug mode off, "+
			"want 0 — the guard is gone, and every context in the tree is copying "+
			"its slot slice once per pass to feed a check that records nothing", off)
	}

	SetDebugMode(true)
	on := testing.AllocsPerRun(50, func() { ctx.EndRenderPass() })
	SetDebugMode(false)
	ClearConcerns()
	if on == 0 {
		t.Error("EndRenderPass allocates nothing with debug mode ON either — the " +
			"guard above is guarding nothing")
	}
	t.Logf("EndRenderPass on a 4-slot context: %.0f allocs on, %.0f off", on, off)
}

// The duplicate-key check is free when debug mode is off.
//
// This one cannot be asserted as a zero. The guard is at the *call site* in
// renderAll rather than inside the check, and renderAll allocates on every path
// because building the child slice is its job — so "off costs nothing" has to be
// measured as a difference rather than as an absolute.
//
// The difference is taken against the check's own cost, measured in the same
// test on the same nodes:
//
//	renderAll(on) - renderAll(off)  ==  checkDuplicateKeys alone
//
// which says exactly the thing the guard is for, and is self-calibrating: a
// change to how checkDuplicateKeys sizes its map moves both sides. Deleting the
// guard makes the left side zero while the right side stays where it is.
func TestTheDuplicateKeyCheckCostsNothingWithDebugModeOff(t *testing.T) {
	debugOff(t)
	views := costViewList(40)
	ctx := NewContext()

	// The nodes the check would see, rendered once outside the measurement.
	nodes := make([]*Node, 0, len(views))
	for _, v := range views {
		nodes = append(nodes, v.Render(ctx))
	}

	off := testing.AllocsPerRun(50, func() { renderAll(ctx, "Column", views) })
	SetDebugMode(true)
	on := testing.AllocsPerRun(50, func() { renderAll(ctx, "Column", views) })
	SetDebugMode(false)
	ClearConcerns()

	// Unguarded, so this is the check's cost whatever the flag says.
	alone := testing.AllocsPerRun(50, func() { checkDuplicateKeys("Column", nodes) })
	if alone == 0 {
		t.Fatal("checkDuplicateKeys allocates nothing at all — there is no cost for " +
			"the guard in renderAll to save, and this test cannot say anything")
	}

	if got := on - off; got != alone {
		t.Errorf("renderAll costs %.0f extra allocations with debug mode on and the "+
			"check itself costs %.0f — the guard in renderAll is not what stands "+
			"between a production render pass and a map allocation per container "+
			"per frame (off=%.0f, on=%.0f)", got, alone, off, on)
	}
	t.Logf("renderAll over %d views: %.0f allocs on, %.0f off; the check alone: %.0f",
		len(views), on, off, alone)
}

// The numbers, for when the verdict above is not what is wanted.
//
// Not on any verification path — `go test ./...` compiles benchmarks and runs
// none of them — so these are a tool rather than a check:
//
//	go test ./core/ -run=NONE -bench=DebugMode -benchmem
//
// The three tests above are what fails when a guard goes missing. These are
// what says how much it was worth.
func BenchmarkAuditTreeDebugModeOff(b *testing.B) { benchAudit(b, false) }
func BenchmarkAuditTreeDebugModeOn(b *testing.B)  { benchAudit(b, true) }

func benchAudit(b *testing.B, on bool) {
	tree := costTree(40)
	SetDebugMode(on)
	defer func() {
		SetDebugMode(false)
		ClearConcerns()
	}()
	b.ReportAllocs()
	for b.Loop() {
		AuditTree(tree)
	}
}

func BenchmarkEndRenderPassDebugModeOff(b *testing.B) { benchCursor(b, false) }
func BenchmarkEndRenderPassDebugModeOn(b *testing.B)  { benchCursor(b, true) }

func benchCursor(b *testing.B, on bool) {
	ctx := costContext()
	SetDebugMode(on)
	defer func() {
		SetDebugMode(false)
		ClearConcerns()
	}()
	b.ReportAllocs()
	for b.Loop() {
		ctx.EndRenderPass()
	}
}

func BenchmarkRenderAllDebugModeOff(b *testing.B) { benchRenderAll(b, false) }
func BenchmarkRenderAllDebugModeOn(b *testing.B)  { benchRenderAll(b, true) }

func benchRenderAll(b *testing.B, on bool) {
	views := costViewList(40)
	ctx := NewContext()
	SetDebugMode(on)
	defer func() {
		SetDebugMode(false)
		ClearConcerns()
	}()
	b.ReportAllocs()
	for b.Loop() {
		renderAll(ctx, "Column", views)
	}
}
