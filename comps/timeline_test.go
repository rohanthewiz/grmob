package comps

import (
	"fmt"
	"testing"

	"github.com/rohanthewiz/grmob/core"
)

func orderTimeline() Timeline {
	return Timeline{
		Label: "Order history",
		Events: []TimelineEvent{
			{Time: "09:12", Title: "Order placed"},
			{Time: "11:40", Title: "Packed", Subtitle: "Warehouse 3"},
			{Title: "Delivered", Variant: VariantSuccess, Content: core.Text("Signed by R.")},
		},
	}
}

// railParts returns a row's top segment, dot and bottom segment.
func railParts(row *core.Node) (top, dot, bottom *core.Node) {
	rail := row.Children[0]
	return rail.Children[0], rail.Children[1], rail.Children[2]
}

func TestTimelineIsANamedListOfStretchedRows(t *testing.T) {
	_, n := renderDebug(t, orderTimeline())

	if n.Type != "Column" || n.Style.AccessibilityRole != core.RoleList || n.Style.AccessibilityLabel != "Order history" {
		t.Fatalf("root = %q role %q label %q", n.Type, n.Style.AccessibilityRole, n.Style.AccessibilityLabel)
	}
	if len(n.Children) != 3 {
		t.Fatalf("want 3 rows, got %d", len(n.Children))
	}
	for i, row := range n.Children {
		if row.Style.AccessibilityRole != core.RoleListItem {
			t.Errorf("row %d role = %q, want listitem", i, row.Style.AccessibilityRole)
		}
		if row.Style.AlignItems != core.AlignItemsStretch {
			t.Errorf("row %d alignItems = %q: the rail is as tall as the body only when the row stretches",
				i, row.Style.AlignItems)
		}
		if row.Style.Padding.Top != 0 || row.Style.Padding.Bottom != 0 {
			t.Errorf("row %d padding = %+v: row padding breaks the line between rows", i, row.Style.Padding)
		}
		if !row.Children[0].Style.AccessibilityHidden {
			t.Errorf("row %d rail must be hidden: it is drawing", i)
		}
	}
}

// The line is continuous: every segment between two dots paints, the first
// top and the last bottom keep their size and paint nothing.
func TestTimelineLineRunsBetweenDotsOnly(t *testing.T) {
	_, n := renderDebug(t, orderTimeline())
	line := core.DefaultTheme.Colors.BorderColor()

	for i, row := range n.Children {
		top, _, bottom := railParts(row)
		wantTop, wantBottom := line, line
		if i == 0 {
			wantTop = ""
		}
		if i == len(n.Children)-1 {
			wantBottom = ""
		}
		if top.Style.Background != wantTop || bottom.Style.Background != wantBottom {
			t.Errorf("row %d segments = %q / %q, want %q / %q",
				i, top.Style.Background, bottom.Style.Background, wantTop, wantBottom)
		}
		if bottom.Style.FlexGrow != 1 {
			t.Errorf("row %d bottom segment must grow to the row's height", i)
		}
		if top.Style.Height == "" {
			t.Errorf("row %d top segment has no height, so its dot would not align", i)
		}
	}

	bodies := []*core.Node{n.Children[0].Children[1], n.Children[2].Children[1]}
	if bodies[0].Style.Padding.Bottom != core.DefaultTheme.Spacing.MD {
		t.Errorf("spacing below an event = %d, want Spacing.MD inside the body", bodies[0].Style.Padding.Bottom)
	}
	if bodies[1].Style.Padding.Bottom != 0 {
		t.Error("the last event needs no space below it")
	}
}

func TestTimelineDotAlignsWithTheFirstLine(t *testing.T) {
	_, n := renderDebug(t, orderTimeline())
	th := core.DefaultTheme

	withTime, _, _ := railParts(n.Children[0])
	withoutTime, _, _ := railParts(n.Children[2])
	if withTime.Style.Height == withoutTime.Style.Height && th.Typography.Caption.FontSize != th.Typography.Body.FontSize {
		t.Errorf("a Time caption and a title are different first lines, but both tops are %s", withTime.Style.Height)
	}
	if got, want := withoutTime.Style.Height, fmt.Sprintf("%dpx", Timeline{}.railTop(th, TimelineEvent{})); got != want {
		t.Errorf("title-first top = %s, want %s", got, want)
	}
}

func TestTimelineBodyOrderAndDotVariant(t *testing.T) {
	_, n := renderDebug(t, orderTimeline())
	th := core.DefaultTheme

	body := n.Children[1].Children[1]
	var texts []string
	for _, c := range body.Children {
		if c.Type == "Text" {
			texts = append(texts, c.Props["content"].(string))
		}
	}
	if len(texts) != 3 || texts[0] != "11:40" || texts[1] != "Packed" || texts[2] != "Warehouse 3" {
		t.Errorf("body = %v, want time, title, subtitle", texts)
	}
	if title := findText(body, "Packed"); title.Style.FontWeight != core.Bold {
		t.Error("the title is bold")
	}

	_, dot0, _ := railParts(n.Children[0])
	_, dot2, _ := railParts(n.Children[2])
	if dot0.Style.Background != th.Colors.Primary || dot2.Style.Background != th.Colors.SuccessColor() {
		t.Errorf("dots = %q / %q, want Primary then Success", dot0.Style.Background, dot2.Style.Background)
	}
	if findText(n.Children[2], "Signed by R.") == nil {
		t.Error("Content renders under the text")
	}
}
