package render_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/rohanthewiz/grmob/core"
	"github.com/rohanthewiz/grmob/render"
)

// A pass that will not encode (N-133, the plan's R3).
//
// encoding/json refuses NaN and ±Inf, and a float is the one ordinary value
// an app hands a node that JSON cannot spell: a progress value computed as
// done/total with total == 0, a FlexGrow derived from the same. The manager
// used to advance currentTree first and then send {"error":...} in place of
// the patches, so every other change in that pass was lost for good: Go
// believed the host had it. These hold the fix: the pass is dropped like a
// panicking one, and the next encodable pass carries what it carried.

// encodeApp shows a label and one float, used either as a style float
// (FlexGrow) or as a prop (a Slider's value), driven from outside.
type encodeApp struct {
	once   sync.Once
	label  core.State[string]
	value  core.State[float64]
	asProp bool
}

func (a *encodeApp) view(ctx *core.Context) core.View {
	label := core.NewState(ctx, "before")
	value := core.NewState(ctx, 1.0)
	a.once.Do(func() { a.label, a.value = label, value })
	var number core.View = core.Box(core.FlexGrow(value.Get()))
	if a.asProp {
		number = core.Slider(value.Get(), 0, 1, func(float64) {})
	}
	return core.Column(core.Text(label.Get()), number)
}

// syncBuffer is a log sink the pump goroutine can write while the test reads.
type syncBuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (b *syncBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.Write(p)
}

func (b *syncBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.String()
}

// quietLog captures the manager's log for the test's length.
func quietLog(t *testing.T) *syncBuffer {
	t.Helper()
	buf := &syncBuffer{}
	prev := log.Writer()
	log.SetOutput(buf)
	t.Cleanup(func() { log.SetOutput(prev) })
	return buf
}

func TestAnUnencodablePassIsDroppedAndItsChangesArriveWithTheNextGoodOne(t *testing.T) {
	for _, tc := range []struct {
		name   string
		asProp bool
		field  string
	}{
		{"a NaN in a style float", false, "root/1 (Box) Style.FlexGrow = NaN"},
		{"a NaN in a prop", true, "root/1 (Slider) Props.value = NaN"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			logged := quietLog(t)
			app := &encodeApp{asProp: tc.asProp}
			m := render.New(core.NewContext(), app.view)
			defer m.Close()
			m.RenderInitial()

			// The unrelated text change rides in the same pass as the NaN.
			app.label.Set("after")
			app.value.Set(math.NaN())
			if out := m.RenderAgain(); out != "[]" {
				t.Fatalf("the unencodable pass returned %s, want [] (the pass dropped, nothing for the host to apply)", out)
			}
			if !strings.Contains(logged.String(), tc.field) {
				t.Errorf("the log does not name %q:\n%s", tc.field, logged.String())
			}

			// While the NaN stands, every pass is dropped the same way: the
			// screen keeps the last tree the host has.
			app.label.Set("still")
			if out := m.RenderAgain(); out != "[]" {
				t.Fatalf("a second pass with the NaN still standing returned %s, want []", out)
			}

			// The value is finite again: the pass is diffed against the tree
			// the host really holds, so the text change arrives with it.
			app.value.Set(0.5)
			out := m.RenderAgain()
			if !strings.Contains(out, `"still"`) {
				t.Fatalf("the first good pass after the NaN does not carry the text change: %s", out)
			}
			if strings.Contains(out, "error") {
				t.Fatalf("the good pass carries an error payload: %s", out)
			}
		})
	}
}

// The mount has no last good tree to keep. It used to send the error object as
// the tree and leave the screen blank; it now mounts the placeholder a
// panicking mount gets, and the first pass that encodes replaces it.
func TestAnUnencodableMountStandsInThePlaceholderUntilAPassEncodes(t *testing.T) {
	quietLog(t)
	var (
		once  sync.Once
		value core.State[float64]
	)
	m := render.New(core.NewContext(), func(ctx *core.Context) core.View {
		v := core.NewState(ctx, math.Inf(1))
		once.Do(func() { value = v })
		return core.Column(core.Text(fmt.Sprintf("value %v", v.Get())), core.Box(core.Gap(v.Get())))
	})
	defer m.Close()

	tree := decodeTree(t, m.RenderInitial())
	if tree.Type != "Text" || tree.Props["content"] != "Something went wrong." {
		t.Fatalf("an unencodable mount sent %+v, want the placeholder Text", tree)
	}

	value.Set(3)
	out := m.RenderAgain()
	var patches []jsonPatch
	decodeInto(t, out, &patches)
	if len(patches) != 1 || patches[0].Type != "replace" || patches[0].TargetID != "root" {
		t.Fatalf("the first encodable pass after the placeholder = %s, want one replace of root", out)
	}
	if !strings.Contains(out, "value 3") {
		t.Errorf("the replace does not carry the app's tree: %s", out)
	}
}

// Under debug mode the drop is a concern too, naming where the number is, so
// a test can fail on it rather than on a screen that quietly stopped.
func TestAnUnencodablePassIsAConcernUnderDebugMode(t *testing.T) {
	quietLog(t)
	core.SetDebugMode(true)
	core.ClearConcerns()
	defer func() {
		core.SetDebugMode(false)
		core.ClearConcerns()
	}()

	// The style float, and an infinity this time: a Slider clamps its value
	// into [min, max], which turns -Inf into 0 before it reaches the node
	// (NaN slips through the clamp, which is why the prop case above uses it).
	app := &encodeApp{}
	m := render.New(core.NewContext(), app.view)
	defer m.Close()
	m.RenderInitial()
	app.value.Set(math.Inf(-1))
	m.RenderAgain()

	var found []string
	for _, c := range core.Concerns() {
		if c.Kind == core.ConcernUnencodablePass {
			found = append(found, c.Detail)
		}
	}
	if len(found) != 1 || found[0] != "root/1 (Box) Style.FlexGrow = -Inf" {
		t.Fatalf("unencodable-pass concerns = %q, want the one naming root/1's value", found)
	}
}

// The push path drops the same way: nothing reaches the listener while the
// NaN stands, and the change arrives once it is gone.
func TestAnUnencodablePushedPassIsNotPushed(t *testing.T) {
	logged := quietLog(t)
	app := &encodeApp{}
	m := render.New(core.NewContext(), app.view)
	defer m.Close()
	m.RenderInitial()
	l := newChanListener()
	m.SetListener(l)

	// The pump renders the NaN pass and drops it; the log is the only sign.
	app.label.Set("after")
	app.value.Set(math.NaN())
	for waited := time.Duration(0); !strings.Contains(logged.String(), "dropped the render pass"); waited += 5 * time.Millisecond {
		if waited > 2*time.Second {
			t.Fatal("the pump never rendered the NaN pass")
		}
		time.Sleep(5 * time.Millisecond)
	}
	select {
	case p := <-l.ch:
		t.Fatalf("the dropped pass pushed %s", p)
	default:
	}

	// Finite again: the push carries both text changes' end state.
	app.value.Set(2)
	app.label.Set("later")
	awaitPush(t, l, 2*time.Second, func(p string) bool {
		if strings.Contains(p, "error") {
			t.Errorf("an error payload was pushed: %s", p)
		}
		return strings.Contains(p, `"later"`)
	})
}

func decodeInto(t *testing.T, s string, v any) {
	t.Helper()
	if err := json.Unmarshal([]byte(s), v); err != nil {
		t.Fatalf("decode %s: %v", s, err)
	}
}
