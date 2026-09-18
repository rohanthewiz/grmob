package comps

import (
	"testing"

	"github.com/rohanthewiz/grmob/core"
)

var attendees = []Avatar{
	{Name: "Ada Lovelace"},
	{Name: "Grace Hopper"},
	{Name: "Katherine Johnson"},
	{Name: "Dorothy Vaughan"},
	{Name: "Mary Jackson"},
	{Name: "Annie Easley"},
}

// Six faces under Max 4: three faces and a "+3" disc, each on a ring layer,
// every layer placed at the start and pushed right one step further.
func TestAvatarStackOverlapsLayersAtIncreasingOffsets(t *testing.T) {
	_, n := renderDebug(t, AvatarStack{Avatars: attendees, Max: 4})

	if n.Type != "ZStack" {
		t.Fatalf("root = %q, want a ZStack", n.Type)
	}
	// Size 32, ring 2, overlap 0.2: each disc is 36 across and the step is
	// round(32·0.8) = 26, so four discs span 36 + 3·26.
	if n.Style.Width != "114px" || n.Style.Height != "36px" {
		t.Errorf("stack = %s × %s, want 114px × 36px", n.Style.Width, n.Style.Height)
	}
	if len(n.Children) != 8 {
		t.Fatalf("layers = %d, want a ring and a disc for each of four", len(n.Children))
	}
	for i, c := range n.Children {
		if c.Style.StackAlign != core.StackAlignStart {
			t.Errorf("layer %d placed %q, want start", i, c.Style.StackAlign)
		}
		if !c.Style.AccessibilityHidden {
			t.Errorf("layer %d should be hidden behind the stack's one name", i)
		}
		want := 26 * (i / 2)
		if i%2 == 1 {
			want += 2 // the face sits inside its ring
		}
		if got := c.Style.Margin.Left; got != want {
			t.Errorf("layer %d margin-left = %d, want %d", i, got, want)
		}
	}

	// The surplus is drawn last, so nothing covers its count.
	if findText(n.Children[7], "+3") == nil {
		t.Error("the last layer should be the +3 disc")
	}
	if got := n.Children[1].Style.Width; got != "32px" {
		t.Errorf("face width = %s, want the stack's Size", got)
	}
}

func TestAvatarStackIsOneNamedPicture(t *testing.T) {
	cases := []struct {
		name  string
		stack AvatarStack
		want  string
	}{
		{"one", AvatarStack{Avatars: attendees[:1]}, "Ada Lovelace"},
		{"two", AvatarStack{Avatars: attendees[:2]}, "Ada Lovelace and Grace Hopper"},
		{"surplus", AvatarStack{Avatars: attendees, Max: 3}, "Ada Lovelace, Grace Hopper and 4 others"},
		{"unnamed", AvatarStack{Avatars: []Avatar{{Initials: "?"}, {Initials: "?"}}}, "2 people"},
		{"mixed", AvatarStack{Avatars: []Avatar{{Name: "Ada Lovelace"}, {Initials: "?"}}}, "Ada Lovelace and 1 other"},
		{"override", AvatarStack{Avatars: attendees, Label: "6 attendees"}, "6 attendees"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			_, n := renderDebug(t, c.stack)
			if n.Style.AccessibilityRole != core.RoleImg {
				t.Errorf("role = %q, want img", n.Style.AccessibilityRole)
			}
			if got := n.Style.AccessibilityLabel; got != c.want {
				t.Errorf("name = %q, want %q", got, c.want)
			}
		})
	}
}

// A list that fits exactly draws every face and no disc; a negative ring
// draws no ring layers.
func TestAvatarStackFitsAndRinglessStacks(t *testing.T) {
	_, n := renderDebug(t, AvatarStack{Avatars: attendees[:4], Max: 4, RingWidth: -1})
	if len(n.Children) != 4 {
		t.Fatalf("layers = %d, want four faces and no rings", len(n.Children))
	}
	for _, c := range n.Children {
		if findFirst(c, func(x *core.Node) bool {
			return x.Type == "Text" && len(x.Props["content"].(string)) > 0 && x.Props["content"].(string)[0] == '+'
		}) != nil {
			t.Error("a list that fits should draw no surplus disc")
		}
	}

	_, empty := renderDebug(t, AvatarStack{})
	if empty.Type != "Box" || !empty.Style.AccessibilityHidden {
		t.Error("an empty stack should be a hidden box, not a picture of nobody")
	}
}
