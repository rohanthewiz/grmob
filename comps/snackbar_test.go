package comps

import (
	"testing"
	"time"

	"github.com/rohanthewiz/grmob/core"
)

func TestSnackbarIsAnInverseStatusStripWithAnAction(t *testing.T) {
	ctx, n := renderDebug(t, Snackbar{
		Visible: true, Message: "Note deleted", Action: "Undo",
		OnAction: func() {}, Duration: -1,
	})
	defer ctx.Close()

	th := core.DefaultTheme
	if n.Type != "Row" || n.Style.AccessibilityRole != core.RoleStatus {
		t.Fatalf("root = %q role %q, want a status Row", n.Type, n.Style.AccessibilityRole)
	}
	if n.Style.AccessibilityLabel != "" {
		t.Error("a live region announces its content; a label would replace the message")
	}
	if n.Style.Background != th.Colors.TextPrimary || n.Style.Display == core.DisplayNone {
		t.Errorf("visible strip: background %q display %q", n.Style.Background, n.Style.Display)
	}
	msg := findText(n, "Note deleted")
	if msg == nil || msg.Style.TextColor != th.Colors.Background || msg.Style.FlexGrow != 1 {
		t.Fatal("the message takes the inverse ink and the row's slack")
	}
	btns := buttonsOf(n)
	if len(btns) != 1 || btns[0].Props["label"] != "Undo" {
		t.Fatalf("want one Undo button, got %d", len(btns))
	}
	if btns[0].Style.TextColor != th.Colors.Background {
		t.Errorf("action ink = %q, want the strip's ink: on-light ink vanishes on the inverse fill",
			btns[0].Style.TextColor)
	}
}

func TestSnackbarHiddenIsDisplayNoneAndErrorIsAnAlert(t *testing.T) {
	ctx, n := renderDebug(t, Snackbar{Message: "Upload failed", Variant: VariantError})
	defer ctx.Close()
	if n.Style.Display != core.DisplayNone {
		t.Errorf("hidden snackbar display = %q, want none", n.Style.Display)
	}
	if n.Style.AccessibilityRole != core.RoleAlert {
		t.Errorf("error role = %q, want alert", n.Style.AccessibilityRole)
	}
	if n.Style.Background != core.DefaultTheme.Colors.Error {
		t.Errorf("error fill = %q", n.Style.Background)
	}
	if len(buttonsOf(n)) != 0 {
		t.Error("no Action: no button")
	}
}

func TestSnackbarActionCallsOnlyOnAction(t *testing.T) {
	var acted, timedOut int
	ctx, n := renderDebug(t, Snackbar{
		Visible: true, Message: "Deleted", Action: "Undo",
		OnAction: func() { acted++ }, OnTimeout: func() { timedOut++ },
		Duration: time.Hour,
	})
	defer ctx.Close()
	ctx.TriggerCallback(buttonsOf(n)[0].Props["onClick"].(string))
	if acted != 1 || timedOut != 0 {
		t.Errorf("acted=%d timedOut=%d, want 1 and 0", acted, timedOut)
	}
}

func TestSnackbarOmitsTheButtonWithoutAHandler(t *testing.T) {
	ctx, n := renderDebug(t, Snackbar{Visible: true, Message: "Saved", Action: "View"})
	defer ctx.Close()
	if len(buttonsOf(n)) != 0 {
		t.Error("an Action with no OnAction would be a button that does nothing")
	}
}

// renderSnackbar renders on a fresh context and leaves it open so the timeout
// hook's timer can fire; the caller closes it.
func renderSnackbar(s Snackbar) *core.Context {
	ctx := core.NewContext()
	ctx.BeginRenderPass()
	s.Render(ctx)
	ctx.EndRenderPass()
	return ctx
}

func TestSnackbarTimesOutOnlyWhenVisibleWithAPositiveDuration(t *testing.T) {
	cases := []struct {
		name  string
		s     Snackbar
		fires bool
	}{
		{"visible", Snackbar{Visible: true, Duration: 15 * time.Millisecond}, true},
		{"hidden", Snackbar{Duration: 15 * time.Millisecond}, false},
		{"negative duration", Snackbar{Visible: true, Duration: -1}, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			fired := make(chan struct{}, 4)
			c.s.Message = "x"
			c.s.OnTimeout = func() { fired <- struct{}{} }
			ctx := renderSnackbar(c.s)
			defer ctx.Close()

			select {
			case <-fired:
				if !c.fires {
					t.Error("fired, want no timeout")
				}
			case <-time.After(150 * time.Millisecond):
				if c.fires {
					t.Error("no timeout within 150ms of a 15ms Duration")
				}
			}
		})
	}
}
