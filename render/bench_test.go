package render_test

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/rohanthewiz/grmob/core"
	"github.com/rohanthewiz/grmob/render"
)

// The pass benchmarks (plan P0, N-138): a whole pass through render.Manager,
// build, diff and encode, on the tree shape reconcile's diff benchmarks use
// (keyed Row(Text, Text, Button) rows in a Column) under a counter and its
// button. P1's generation counter, P2's equality and P3's registry purge are
// each measured against these.
//
// No listener is attached, so the pump stays parked and each iteration is
// exactly one pass on the benchmark's goroutine. Debug mode is off, as in a
// release build: under it every pass is also audited, which is not the cost
// an app ships with.
//
//	go test -run '^$' -bench Manager -benchmem ./render

func benchApp(rows int) func(*core.Context) core.View {
	return func(ctx *core.Context) core.View {
		taps := core.NewState(ctx, 0)
		kids := make([]core.PropsAndChildren, 0, rows+2)
		kids = append(kids,
			core.Button("+", func() { taps.Set(taps.Get() + 1) }),
			core.Text(fmt.Sprintf("%d taps", taps.Get())),
		)
		for i := range rows {
			kids = append(kids, core.Keyed(fmt.Sprintf("r%d", i), core.Row(
				core.Text(fmt.Sprintf("Row %d", i)),
				core.Text("A line of detail under the title"),
				core.Button("Open", func() {}),
			)))
		}
		return core.Column(kids...)
	}
}

func releaseMode(b *testing.B) {
	was := core.IsDebugMode()
	core.SetDebugMode(false)
	b.Cleanup(func() { core.SetDebugMode(was) })
}

// A pass with nothing changed: the whole tree is built and compared, and
// "[]" goes out. It is the floor every pass pays, and the pass P1 removes
// from every tap.
func BenchmarkManagerNoopPass(b *testing.B) {
	releaseMode(b)
	for _, rows := range []int{250, 2500} {
		b.Run(fmt.Sprintf("rows=%d", rows), func(b *testing.B) {
			mgr := render.New(core.NewContext(), benchApp(rows))
			defer mgr.Close()
			mgr.RenderInitial()
			b.ReportAllocs()
			for b.Loop() {
				if out := mgr.RenderAgain(); out != "[]" {
					b.Fatalf("an unchanged pass sent %s", out)
				}
			}
		})
	}
}

// A tap that changes one Text: the handler, the pass and its one patch.
func BenchmarkManagerTap(b *testing.B) {
	releaseMode(b)
	for _, rows := range []int{250, 2500} {
		b.Run(fmt.Sprintf("rows=%d", rows), func(b *testing.B) {
			mgr := render.New(core.NewContext(), benchApp(rows))
			defer mgr.Close()
			var root core.Node
			if err := json.Unmarshal([]byte(mgr.RenderInitial()), &root); err != nil {
				b.Fatal(err)
			}
			// The counter's button is the first callback registered in the
			// root scope on every pass, so its ID is the same each time.
			id, ok := root.Children[0].Props["onClick"].(string)
			if !ok {
				b.Fatalf("the first child is not the + button: %+v", root.Children[0])
			}
			b.ReportAllocs()
			for b.Loop() {
				if out := mgr.DispatchCallback(id); out == "[]" {
					b.Fatal("a tap changed nothing")
				}
			}
		})
	}
}
