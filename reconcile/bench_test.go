package reconcile

import (
	"fmt"
	"testing"

	"github.com/rohanthewiz/grmob/core"
)

// The diff benchmarks (plan P0, N-138): the number every later change to the
// diff (P2's generated Style equality and lazy child paths) has to beat.
//
// The tree is the plan's measuring shape, rendered through core's real
// builders so every node carries the Style a theme gives it: one Column of
// keyed Row(Text, Text, Button) children. That is four nodes a row plus the
// Column, so 250 rows is 1,001 nodes and 2,500 rows is 10,001.
//
//	go test -run '^$' -bench Diff -benchmem ./reconcile

// benchTree renders rows rows; the row at changed (if any) says something
// different, and nothing else does.
func benchTree(rows, changed int) *core.Node {
	ctx := core.NewContext()
	ctx.BeginRenderPass()
	kids := make([]core.PropsAndChildren, 0, rows)
	for i := range rows {
		title := fmt.Sprintf("Row %d", i)
		if i == changed {
			title += " (edited)"
		}
		kids = append(kids, core.Keyed(fmt.Sprintf("r%d", i), core.Row(
			core.Text(title),
			core.Text("A line of detail under the title"),
			core.Button("Open", func() {}),
		)))
	}
	return core.Column(kids...).Render(ctx)
}

func nodeCount(n *core.Node) int {
	c := 1
	for _, k := range n.Children {
		c += nodeCount(k)
	}
	return c
}

var benchRows = []int{250, 2500}

// Two renders of the same state: every node is compared and nothing is
// emitted, which is the cost of a pass whose handler changed nothing.
func BenchmarkDiffUnchanged(b *testing.B) {
	for _, rows := range benchRows {
		old, cur := benchTree(rows, -1), benchTree(rows, -1)
		b.Run(fmt.Sprintf("nodes=%d", nodeCount(old)), func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				if p := Diff(old, cur, "root"); len(p) != 0 {
					b.Fatalf("an unchanged tree produced %d patches", len(p))
				}
			}
		})
	}
}

// One row's title changed in the middle: the common pass, a tap that edits
// one thing on a long screen.
func BenchmarkDiffOneChange(b *testing.B) {
	for _, rows := range benchRows {
		old, cur := benchTree(rows, -1), benchTree(rows, rows/2)
		b.Run(fmt.Sprintf("nodes=%d", nodeCount(old)), func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				if p := Diff(old, cur, "root"); len(p) != 1 {
					b.Fatalf("one changed title produced %d patches, want 1", len(p))
				}
			}
		})
	}
}
