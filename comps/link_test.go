package comps

import (
	"strings"
	"testing"

	"github.com/rohanthewiz/grmob/core"
)

// A link is a labelled RoleLink container that hugs its text, drawn in the
// primary on-light ink, whose text is hidden so the name is read once.
func TestLinkIsANamedLinkInThePrimaryInk(t *testing.T) {
	_, n := renderDebug(t, Link{Text: "Privacy policy", URL: "https://example.com/privacy"})

	if n.Style.AccessibilityRole != core.RoleLink {
		t.Fatalf("role = %q, want link", n.Style.AccessibilityRole)
	}
	if n.Style.AccessibilityLabel != "Privacy policy" {
		t.Errorf("name = %q", n.Style.AccessibilityLabel)
	}
	if n.Style.AlignSelf != core.AlignItemsStart {
		t.Errorf("align-self = %q, want start so the empty width beside it is not a target", n.Style.AlignSelf)
	}
	text := findText(n, "Privacy policy")
	if text == nil {
		t.Fatal("the link text is drawn")
	}
	if want := VariantDefault.OnLight(core.DefaultTheme); text.Style.TextColor != want {
		t.Errorf("ink = %q, want %q", text.Style.TextColor, want)
	}
	if !text.Style.AccessibilityHidden {
		t.Error("the text should be hidden under the container's name")
	}
}

// URL goes to OpenURL; OnTap, when set, wins and nothing is opened.
func TestLinkTapOpensTheURLUnlessOnTapIsSet(t *testing.T) {
	var opened []string
	core.SetSystemEventHandler(func(name string, data map[string]any) {
		if name == "open_url" {
			opened = append(opened, data["url"].(string))
		}
	})
	t.Cleanup(func() { core.SetSystemEventHandler(nil) })

	ctx, n := renderDebug(t, Link{Text: "Help", URL: "https://example.com/help"})
	ctx.TriggerCallback(n.Props["onClick"].(string))
	if len(opened) != 1 || opened[0] != "https://example.com/help" {
		t.Errorf("opened = %v, want the URL once", opened)
	}

	tapped := 0
	ctx, n = renderDebug(t, Link{Text: "Help", URL: "https://example.com/help", OnTap: func() { tapped++ }})
	ctx.TriggerCallback(n.Props["onClick"].(string))
	if tapped != 1 || len(opened) != 1 {
		t.Errorf("OnTap ran %d times and %d URLs were opened; want 1 and still 1", tapped, len(opened))
	}
}

// A link that goes nowhere is reported.
func TestLinkWithNoDestinationReportsInert(t *testing.T) {
	core.SetDebugMode(true)
	core.ClearConcerns()
	t.Cleanup(func() { core.SetDebugMode(false); core.ClearConcerns() })
	ctx := core.NewContext()
	ctx.BeginRenderPass()
	Link{Text: "Terms"}.Render(ctx)
	ctx.EndRenderPass()
	if dump := core.DumpConcerns(); !strings.Contains(dump, ConcernLinkInert) {
		t.Errorf("concerns = %q, want %s", dump, ConcernLinkInert)
	}
}
