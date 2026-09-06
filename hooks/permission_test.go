package hooks_test

import (
	"testing"
	"time"

	"github.com/rohanthewiz/grmob/core"
	"github.com/rohanthewiz/grmob/hooks"
	"github.com/rohanthewiz/grmob/permission"
)

// recordPermissionCommands captures the traffic UsePermission generates and
// detaches the handler afterwards so one test cannot see the next one's.
//
// A handler must be installed even where the commands are not read, because
// its absence is a behaviour: permission.Check short-circuits to Unavailable
// when nothing is listening, so a test that forgot this would be exercising
// the headless path.
func recordPermissionCommands(t *testing.T) *[]string {
	t.Helper()
	var cmds []string
	core.SetSystemEventHandler(func(name string, data map[string]any) {
		if name != "permission" {
			return
		}
		if c, ok := data["command"].(string); ok {
			cmds = append(cmds, c)
		}
	})
	t.Cleanup(func() { core.SetSystemEventHandler(nil) })
	return &cmds
}

// The hook checks once, re-renders when the answer lands, and never prompts.
//
// "Never prompts" is the assertion that matters and the reason Check and
// Request are separate functions at all: a Request run from a render pass
// would put the OS dialog on screen as a side effect of drawing, at a moment
// the user did nothing to cause, which is the surest route to a permanent
// refusal.
func TestUsePermissionChecksOnceAndNeverRequests(t *testing.T) {
	cmds := recordPermissionCommands(t)
	ctx := core.NewContext()
	renders := make(chan struct{}, 16)
	ctx.OnStateChange(func() { renders <- struct{}{} })

	var last permission.Status
	body := func(ctx *core.Context) { last = hooks.UsePermission(ctx, permission.Camera) }

	renderPass(ctx, body)
	renderPass(ctx, body) // a second render must not issue a second check

	if len(*cmds) != 1 || (*cmds)[0] != "check" {
		t.Fatalf("commands = %v, want exactly one %q", *cmds, "check")
	}
	if last != permission.Unknown {
		t.Errorf("first pass returned %q — the answer is asynchronous, so a screen has "+
			"to be able to draw the not-yet state", last)
	}

	// Mounting itself asks for no render: unlike UseHeading, which hears its
	// own sensor coming on, nothing has changed until the platform answers.
	assertQuiet(t, renders, 30*time.Millisecond, "render request at mount")

	permission.Receive(permission.Camera, permission.Granted)
	awaitSignal(t, renders, "render request after the platform answered")
	assertQuiet(t, renders, 30*time.Millisecond,
		"second render request (the subscription stacked over two passes)")

	renderPass(ctx, body)
	if last != permission.Granted {
		t.Errorf("after the answer: %q", last)
	}
}

// One process-wide channel carries every permission, so a hook watching one
// must ignore the others. Without the filter a screen asking about two
// permissions renders twice per answer, and a screen asking about one renders
// for answers it has no interest in at all.
func TestUsePermissionIgnoresOtherPermissionsAnswers(t *testing.T) {
	recordPermissionCommands(t)
	ctx := core.NewContext()
	renders := make(chan struct{}, 16)
	ctx.OnStateChange(func() { renders <- struct{}{} })

	body := func(ctx *core.Context) { hooks.UsePermission(ctx, permission.Camera) }
	renderPass(ctx, body)

	permission.Receive(permission.Microphone, permission.Denied)
	assertQuiet(t, renders, 30*time.Millisecond,
		"render request from a microphone answer, in a hook watching the camera")
}

// After a close the tree stays renderable and a re-mount must be able to
// subscribe again — the same drain semantics UseInterval and UseHeading carry.
func TestUsePermissionSubscribesAgainAfterAClose(t *testing.T) {
	cmds := recordPermissionCommands(t)
	ctx := core.NewContext()
	body := func(ctx *core.Context) { hooks.UsePermission(ctx, permission.Location) }

	renderPass(ctx, body)
	ctx.Close()
	renderPass(ctx, body)

	if len(*cmds) != 2 {
		t.Errorf("commands = %v, want a second check after the close — a slot that "+
			"remembers it subscribed can never be re-mounted", *cmds)
	}
}

// With no host attached the hook resolves to Unavailable rather than sitting
// on Unknown, so a permission-gated screen under test takes the "cannot do
// this" branch instead of drawing its spinner forever.
func TestUsePermissionResolvesImmediatelyWithNoHost(t *testing.T) {
	core.SetSystemEventHandler(nil)
	ctx := core.NewContext()

	var last permission.Status
	renderPass(ctx, func(ctx *core.Context) {
		last = hooks.UsePermission(ctx, permission.Storage)
	})

	if last != permission.Unavailable {
		t.Errorf("headless hook returned %q, want %q on the very first pass",
			last, permission.Unavailable)
	}
}
