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

// --- UsePermissionLive ------------------------------------------------------

// resumeApp drives one full foreground transition. Two calls because
// core.OnLifecycle notifies on changes and the record starts at active, so an
// app that never left it has nothing to come back from.
func resumeApp() {
	core.ReceiveLifecycle(core.LifecycleBackground)
	core.ReceiveLifecycle(core.LifecycleActive)
}

// The bug this hook exists for: the user is refused, opens Settings, grants
// it, and comes back to a screen still saying the camera is off.
func TestUsePermissionLiveRechecksOnForeground(t *testing.T) {
	cmds := recordPermissionCommands(t)
	ctx := core.NewContext()
	t.Cleanup(ctx.Close)

	body := func(ctx *core.Context) { hooks.UsePermissionLive(ctx, permission.Camera) }
	renderPass(ctx, body)
	renderPass(ctx, body) // a second render must not register a second watch

	if len(*cmds) != 1 {
		t.Fatalf("commands on mount = %v, want one check", *cmds)
	}

	resumeApp()

	if len(*cmds) != 2 || (*cmds)[1] != "check" {
		t.Fatalf("commands after a resume = %v, want a second %q — a screen that "+
			"sent the user to Settings has to notice what they did there",
			*cmds, "check")
	}
}

// A resume must never prompt. The user did nothing to ask for a dialog, and a
// dialog they did not expect is the one they refuse.
func TestUsePermissionLiveNeverRequests(t *testing.T) {
	cmds := recordPermissionCommands(t)
	ctx := core.NewContext()
	t.Cleanup(ctx.Close)

	renderPass(ctx, func(ctx *core.Context) {
		hooks.UsePermissionLive(ctx, permission.Location)
	})
	resumeApp()
	resumeApp()

	for _, c := range *cmds {
		if c != "check" {
			t.Fatalf("commands = %v, want checks only", *cmds)
		}
	}
}

// The watch is released with the tree, and a re-mount takes it again — the
// same drain semantics every other hook here carries.
func TestUsePermissionLiveReleasesItsWatchOnClose(t *testing.T) {
	recordPermissionCommands(t)
	ctx := core.NewContext()
	body := func(ctx *core.Context) { hooks.UsePermissionLive(ctx, permission.Storage) }

	renderPass(ctx, body)
	if n := permission.WatchedForeground(permission.Storage); n != 1 {
		t.Fatalf("watchers after mount = %d, want 1", n)
	}

	ctx.Close()
	if n := permission.WatchedForeground(permission.Storage); n != 0 {
		t.Fatalf("watchers after close = %d, want 0", n)
	}

	renderPass(ctx, body)
	if n := permission.WatchedForeground(permission.Storage); n != 1 {
		t.Errorf("watchers after re-mount = %d, want 1 — a slot that remembers it "+
			"watched can never be re-mounted", n)
	}
	ctx.Close()
}

// Two screens watching the same kind is one check per resume, which is the
// objection that used to keep this hook from existing.
func TestTwoLiveScreensAreOneCheckPerResume(t *testing.T) {
	cmds := recordPermissionCommands(t)
	ctx := core.NewContext()
	t.Cleanup(ctx.Close)

	renderPass(ctx, func(ctx *core.Context) {
		hooks.UsePermissionLive(ctx, permission.Camera)
		hooks.UsePermissionLive(ctx, permission.Camera)
	})
	before := len(*cmds)

	resumeApp()

	if got := len(*cmds) - before; got != 1 {
		t.Errorf("checks on resume = %d, want 1 — the permission owns the "+
			"re-check, not the screen", got)
	}
}

// The two hooks sit at different cursor positions and must not read each
// other's slot. A component using both would otherwise have one of them
// believe it had already started.
func TestUsePermissionAndUsePermissionLiveDoNotShareASlot(t *testing.T) {
	cmds := recordPermissionCommands(t)
	ctx := core.NewContext()
	t.Cleanup(ctx.Close)

	renderPass(ctx, func(ctx *core.Context) {
		hooks.UsePermission(ctx, permission.Camera)
		hooks.UsePermissionLive(ctx, permission.Microphone)
	})

	if len(*cmds) != 2 {
		t.Fatalf("commands = %v, want one check per hook", *cmds)
	}
	if n := permission.WatchedForeground(permission.Microphone); n != 1 {
		t.Errorf("microphone watchers = %d, want 1", n)
	}
	if n := permission.WatchedForeground(permission.Camera); n != 0 {
		t.Errorf("camera watchers = %d, want 0 — plain UsePermission takes no watch", n)
	}
}
