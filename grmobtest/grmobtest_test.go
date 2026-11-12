package grmobtest

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/rohanthewiz/grmob/core"
	"github.com/rohanthewiz/grmob/hooks"
)

func TestMain(m *testing.M) {
	core.SetDebugMode(true)
	m.Run()
}

// runawayEnv makes the test binary, run again as a child, host a bubble that
// never settles (see TestARunawayBubbleFailsFastAndNamesTheTest).
const runawayEnv = "GRMOBTEST_RUNAWAY_CHILD"

// The child half: the shape a broken Close gives a hook's test, a drain that
// waits for quiet while a ticker nothing stopped keeps it from coming. It
// must not return; the watchdog's panic is what ends it.
func TestRunawayBubbleChild(t *testing.T) {
	if os.Getenv(runawayEnv) != "1" {
		t.Skip("run only as the child of TestARunawayBubbleFailsFastAndNamesTheTest")
	}
	SettleLimit = 300 * time.Millisecond
	WithFakeClock(t, func(t *testing.T) {
		tick := time.NewTicker(10 * time.Millisecond)
		for {
			select {
			case <-tick.C:
				continue
			case <-time.After(50 * time.Millisecond):
			}
			break
		}
		tick.Stop()
	})
	t.Fatal("the drain went quiet with the ticker still running")
}

// A bubble that never settles is a fast failure that names its test, not a
// hang until go test's timeout. Run in a child process because the failure
// is a panic that ends the binary.
func TestARunawayBubbleFailsFastAndNamesTheTest(t *testing.T) {
	cmd := exec.Command(os.Args[0], "-test.run=^TestRunawayBubbleChild$", "-test.count=1")
	cmd.Env = append(os.Environ(), runawayEnv+"=1")
	start := time.Now()
	out, err := cmd.CombinedOutput()
	took := time.Since(start)
	if err == nil {
		t.Fatalf("the child passed; a runaway bubble must fail:\n%s", out)
	}
	if !strings.Contains(string(out), "TestRunawayBubbleChild's bubble did not settle") {
		t.Fatalf("the child failed without the watchdog's message:\n%s", out)
	}
	if took > 20*time.Second {
		t.Fatalf("the child took %v to fail; the watchdog is set to 300ms", took)
	}
}

// demo is one screen with every control the Driver has a verb for, plus a
// timer, so each test below drives a real render.Manager rather than a stub.
func demo(ctx *core.Context) core.View {
	count := core.NewState(ctx, 0)
	name := core.NewState(ctx, "")
	alerts := core.NewState(ctx, false)
	fired := core.NewState(ctx, false)
	hooks.UseTimeout(ctx, func() { fired.Set(true) }, 300*time.Millisecond)

	status := "waiting"
	if fired.Get() {
		status = "fired"
	}
	return core.Column(
		core.Text(fmt.Sprintf("Count: %d", count.Get())),
		core.Button("+", func() { count.Set(count.Get() + 1) }),
		core.Input(name.Get(), "Your name", name.Set),
		core.Text("Hello, "+name.Get()),
		core.Switch(alerts.Get(), alerts.Set, core.AccessibilityLabel("Alerts")),
		core.Text(fmt.Sprintf("Alerts: %v", alerts.Get())),
		core.Text("Timer "+status),
	)
}

func TestTheDriverTapsTypesTogglesAndReads(t *testing.T) {
	app := New(t, demo)

	if !app.HasText("Count: 0") {
		t.Fatal("the counter does not open at 0")
	}
	app.Tap("+")
	app.Tap("+")
	if !app.HasText("Count: 2") {
		t.Fatal("two taps did not reach 2")
	}

	app.Type("Your name", "Ada")
	if !app.HasText("Hello, Ada") {
		t.Fatal("typing did not reach the greeting")
	}
	// The field now holds the value, which is on screen too.
	if !app.Shows("Ada") {
		t.Fatal("Shows does not find the typed value")
	}

	app.Toggle("Alerts", true)
	if !app.HasText("Alerts: true") {
		t.Fatal("Toggle did not switch the named Switch on")
	}

	app.NoConcerns()
}

// AwaitText waits on the clock the app's timers run on, so inside a
// WithFakeClock bubble a 300 ms timeout costs no real time.
func TestAwaitTextWaitsForATimerOnTheBubblesClock(t *testing.T) {
	WithFakeClock(t, func(t *testing.T) {
		app := New(t, demo)
		if !app.HasText("Timer waiting") {
			t.Fatal("the timer fired before it was due")
		}
		start := time.Now()
		app.AwaitText("Timer fired")
		if waited := time.Since(start); waited < 300*time.Millisecond {
			t.Fatalf("the text arrived after %v of the bubble's clock, before the 300ms timeout was due", waited)
		}
		app.NoConcerns()
	})
}

// HasText is exact and Text-only; Shows is a substring over every word on
// screen but never a callback ID.
func TestHasTextIsExactAndShowsSkipsCallbackIDs(t *testing.T) {
	app := New(t, demo)
	root := app.Tree()

	if HasText(root, "Count") {
		t.Error("HasText matched a substring")
	}
	if HasText(root, "+") {
		t.Error("HasText matched a Button's label")
	}
	if !Shows(root, "Count") || !Shows(root, "+") || !Shows(root, "Your name") {
		t.Error("Shows missed a word on screen (a Text, a label, a placeholder)")
	}
	if Shows(root, "cb_") {
		t.Error("Shows matched a callback ID")
	}
}

// Every string under an on… prop in a tree using every callback kind is
// recognised as a callback ID. This is what holds callbackPrefixes to core's
// spellings: a kind core added without this list knowing would leave its IDs
// searchable by Shows, and fail here.
func TestEveryCallbackPropIsRecognisedAsAnID(t *testing.T) {
	kinds := func(ctx *core.Context) core.View {
		on := core.NewState(ctx, false)
		v := core.NewState(ctx, 0.0)
		return core.Column(
			core.OnBack(func() {}),
			core.Button("void", func() {}),
			core.Input("", "text", func(string) {}),
			core.Checkbox(on.Get(), on.Set),
			core.Slider(v.Get(), 0, 10, v.Set),
			core.Keyed("row-1", core.Button("keyed", func() {})),
			core.TabView(
				core.Tabs(core.Tab("One", ""), core.Tab("Two", "")),
				core.OnTabChange(func(int) {}),
				core.Content(core.Text("one"), core.Text("two")),
			),
		)
	}
	app := New(t, kinds)
	seen := map[string]bool{}
	for _, n := range app.FindAll(func(*core.Node) bool { return true }) {
		for k, v := range n.Props {
			id, ok := v.(string)
			if !ok || !strings.HasPrefix(k, "on") {
				continue
			}
			if !isCallbackID(id) {
				t.Errorf("%s.%s = %q is not recognised as a callback ID", n.Type, k, id)
			}
			seen[id[:strings.Index(id, "cb_")+3]] = true
		}
	}
	// Each kind present, so the loop above was not vacuous for any of them.
	for _, p := range callbackPrefixes {
		if !seen[p] {
			t.Errorf("no %s… ID in the tree; the fixture no longer exercises that kind", p)
		}
	}
}

// A record one test leaves behind does not reach the next test's app: New
// resets when it opens the only live Driver.
func TestNewStartsFromAFreshProcessesState(t *testing.T) {
	core.ReceiveHostEvent("location", map[string]any{"lat": 51.5, "lng": -0.12, "accuracy": 5.0})
	if !core.CurrentLocation().Received {
		t.Fatal("the fixture's fix was not recorded, so this test would pass vacuously")
	}

	var got core.Location
	New(t, func(ctx *core.Context) core.View {
		got = hooks.UseLocation(ctx)
		return core.Text("map")
	}).Tree()
	if got.Received {
		t.Fatalf("the app's first render read a fix from before it existed: %+v", got)
	}
}

// Two Drivers alive at once share the state (the second New does not reset
// under the first), and the last one to close leaves a fresh process's state
// behind.
func TestResetRunsOnlyOnTheEdges(t *testing.T) {
	t.Run("two live drivers", func(t *testing.T) {
		first := New(t, demo)
		core.ReceiveHostEvent("location", map[string]any{"lat": 1.0, "lng": 2.0, "accuracy": 3.0})
		New(t, demo)
		if !core.CurrentLocation().Received {
			t.Fatal("opening a second Driver reset the state under the first")
		}
		first.Tap("+")
		if !first.HasText("Count: 1") {
			t.Fatal("the first Driver stopped working when the second opened")
		}
	})
	if core.CurrentLocation().Received {
		t.Fatal("the fix outlived the last Driver's close")
	}
}

// fakeTB records a fatal failure instead of ending the goroutine, so a test
// can read the message a Driver gives a person whose test has gone wrong.
type fakeTB struct {
	testing.TB
	msg string
}

type fatal struct{}

func (f *fakeTB) Helper() {}
func (f *fakeTB) Fatal(args ...any) {
	f.msg = fmt.Sprint(args...)
	panic(fatal{})
}
func (f *fakeTB) Fatalf(format string, args ...any) {
	f.msg = fmt.Sprintf(format, args...)
	panic(fatal{})
}

// failure runs fn against a fakeTB and returns the fatal message, or "" when
// fn returned without one.
func failure(t *testing.T, fn func(tb testing.TB)) (msg string) {
	t.Helper()
	f := &fakeTB{TB: t}
	defer func() {
		if r := recover(); r != nil {
			if _, ok := r.(fatal); !ok {
				panic(r)
			}
			msg = f.msg
		}
	}()
	fn(f)
	return ""
}

// Each verb names what it looked for when it is not there, rather than
// panicking on a nil node or dispatching nothing.
func TestMissingTargetsFailWithWhatWasMissing(t *testing.T) {
	app := New(t, demo)
	cases := []struct {
		name string
		do   func(d *Driver)
		want string
	}{
		{"tap", func(d *Driver) { d.Tap("Delete") }, `no tappable node on screen is a Button labelled "Delete"`},
		{"type", func(d *Driver) { d.Type("Email", "x") }, `no field with placeholder "Email"`},
		{"toggle", func(d *Driver) { d.Toggle("Wi-Fi", true) }, `no Checkbox or Switch named "Wi-Fi"`},
	}
	for _, c := range cases {
		got := failure(t, func(tb testing.TB) {
			c.do(&Driver{t: tb, mgr: app.mgr, ctx: app.ctx})
		})
		if !strings.Contains(got, c.want) {
			t.Errorf("%s: failure %q, want it to contain %q", c.name, got, c.want)
		}
	}
}

// NoConcerns refuses to pass with the audit off, and reports what the audit
// found with it on.
func TestNoConcernsNeedsDebugModeAndReportsFindings(t *testing.T) {
	core.SetDebugMode(false)
	got := failure(t, func(tb testing.TB) { NoConcerns(tb) })
	core.SetDebugMode(true)
	if !strings.Contains(got, "needs debug mode") {
		t.Fatalf("with debug mode off, NoConcerns said %q", got)
	}

	// Two siblings with one key is a concern the audit records.
	app := New(t, func(ctx *core.Context) core.View {
		return core.Column(
			core.Keyed("same", core.Text("a")),
			core.Keyed("same", core.Text("b")),
		)
	})
	app.Tree()
	got = failure(t, func(tb testing.TB) { NoConcerns(tb) })
	if !strings.Contains(got, "debug concerns raised") {
		t.Fatalf("a duplicate key passed NoConcerns: %q", got)
	}
	// Leave the collector empty for whatever runs next in this process; the
	// Driver's cleanup would too, but this test should not depend on that.
	core.ClearConcerns()
}

// The decoded tree is core.Node, so Style reads with its Go names.
func TestTheTreeDecodesIntoCoreNode(t *testing.T) {
	app := New(t, func(ctx *core.Context) core.View {
		return core.Column(core.Text("hi", core.FontSize(22), core.AccessibilityLabel("greeting")))
	})
	n := app.Find(Named("greeting"))
	if n == nil || n.Type != "Text" {
		t.Fatalf("Named did not find the labelled Text: %+v", n)
	}
	if n.Style.FontSize != 22 {
		t.Fatalf("Style.FontSize = %v after decoding, want 22", n.Style.FontSize)
	}
}
