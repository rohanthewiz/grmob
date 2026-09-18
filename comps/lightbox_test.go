package comps

import (
	"strings"
	"testing"

	"github.com/rohanthewiz/grmob/core"
)

// A Modal around a black panel: the close button, the fitted image, the
// caption, and every way out reported once through OnDismiss.
func TestLightboxIsAModalAroundAFittedImage(t *testing.T) {
	dismissed := 0
	ctx, n := renderDebug(t, Lightbox{
		Src:       "https://example.com/p.jpg",
		Alt:       "A heron on a post",
		Caption:   "Morning, the harbour",
		Open:      true,
		OnDismiss: func() { dismissed++ },
	})

	if n.Type != "Modal" || n.Props["visible"] != true {
		t.Fatalf("root = %q visible %v, want an open Modal", n.Type, n.Props["visible"])
	}
	if n.Props["backdrop"] != lightboxScrim {
		t.Errorf("scrim = %v, want the lightbox's own near-black", n.Props["backdrop"])
	}
	panel := n.Children[0]
	if panel.Style.Background != lightboxPanel {
		t.Errorf("panel fill = %q; it must be filled, or white ink vanishes on an iOS sheet", panel.Style.Background)
	}

	img := findFirst(n, func(c *core.Node) bool { return c.Type == "Image" })
	if img == nil {
		t.Fatal("no image drawn")
	}
	if img.Props["contentMode"] != string(core.ContentModeFit) {
		t.Errorf("mode = %v, want fit: a lightbox shows what the thumbnail cropped", img.Props["contentMode"])
	}
	if img.Style.AccessibilityLabel != "A heron on a post" || img.Style.Height != "400px" {
		t.Errorf("image name/height = %q/%s", img.Style.AccessibilityLabel, img.Style.Height)
	}
	if cap := findText(n, "Morning, the harbour"); cap == nil || cap.Style.TextColor != lightboxInk {
		t.Error("the caption should be drawn in the lightbox's white")
	}

	btns := buttonsOf(n)
	if len(btns) != 1 || btns[0].Style.AccessibilityLabel != "Close" {
		t.Fatalf("want one close button named Close, got %d", len(btns))
	}
	ctx.TriggerCallback(btns[0].Props["onClick"].(string))
	ctx.TriggerCallback(n.Props["onDismiss"].(string))
	if dismissed != 2 {
		t.Errorf("dismissals = %d, want one each from the button and the scrim", dismissed)
	}
}

// With no Alt the image is silent rather than announced as "image".
func TestLightboxWithoutAltHidesTheImage(t *testing.T) {
	_, n := renderDebug(t, Lightbox{Src: "x.jpg", OnDismiss: func() {}})
	img := findFirst(n, func(c *core.Node) bool { return c.Type == "Image" })
	if !img.Style.AccessibilityHidden {
		t.Error("an image with no Alt should be hidden")
	}
}

func TestLightboxWithoutOnDismissIsAConcern(t *testing.T) {
	newQuietRowHarness(t, func() core.View { return Lightbox{Src: "x.jpg", Open: true} })
	if !strings.Contains(core.DumpConcerns(), ConcernLightboxInescapable) {
		t.Errorf("want %s, got:\n%s", ConcernLightboxInescapable, core.DumpConcerns())
	}
}
