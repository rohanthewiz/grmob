package comps

import (
	"slices"
	"strings"
	"testing"

	"github.com/rohanthewiz/grmob/core"
)

func sampleDiscussion() []DiscussionComment {
	return []DiscussionComment{
		{Key: "a", Author: "Ana", Time: "2h", Body: "Has anyone tried the new build?", Likes: 3, Replies: []DiscussionComment{
			{Key: "b", Author: "Ben", Time: "1h", Body: "Works on my Pixel.", Likes: 1, Liked: true, Replies: []DiscussionComment{
				{Key: "c", Author: "Cy", Body: "Same on iOS."},
			}},
			{Key: "d", Author: "Dee", Body: "Crashes on launch for me."},
		}},
		{Key: "e", Author: "Eli", Body: "Release notes?", Deleted: true},
	}
}

// itemsOf returns every list item, depth-first, with its nesting level.
func itemsOf(n *core.Node) (keys []string, levels []int) {
	var walk func(*core.Node)
	walk = func(x *core.Node) {
		if x.Style.AccessibilityRole == core.RoleListItem {
			keys = append(keys, x.Key)
			levels = append(levels, x.Style.AccessibilityNestingLevel)
		}
		for _, c := range x.Children {
			walk(c)
		}
	}
	walk(n)
	return
}

func labelled(n *core.Node, label string) *core.Node {
	return findFirst(n, func(x *core.Node) bool { return x.Style.AccessibilityLabel == label })
}

// The tree is lists of items, each item stating its depth; the header rows
// carry the reply-to phrase for the natives.
func TestDiscussionIsANestedList(t *testing.T) {
	_, n := renderDebug(t, Discussion{Title: "5 comments", Comments: sampleDiscussion()})

	if h := findText(n, "5 comments"); h == nil || h.Style.AccessibilityRole != core.RoleHeading {
		t.Error("Title should be a heading")
	}
	keys, levels := itemsOf(n)
	if !slices.Equal(keys, []string{"c:a", "c:b", "c:c", "c:d", "c:e"}) {
		t.Errorf("items = %v", keys)
	}
	if !slices.Equal(levels, []int{1, 2, 3, 2, 1}) {
		t.Errorf("levels = %v", levels)
	}
	for _, phrase := range []string{"Ana, 2h", "Ben, reply to Ana, 1h", "Cy, reply to Ben"} {
		if labelled(n, phrase) == nil {
			t.Errorf("no header named %q", phrase)
		}
	}
	// Every list holds items only.
	var check func(*core.Node)
	check = func(x *core.Node) {
		if x.Style.AccessibilityRole == core.RoleList {
			for _, c := range x.Children {
				if c.Style.AccessibilityRole != core.RoleListItem {
					t.Errorf("a list holds a %s that is not an item", c.Type)
				}
			}
		}
		for _, c := range x.Children {
			check(c)
		}
	}
	check(n)
}

// Read-only: no Reply buttons, no Like toggles, counts as text, and a
// deleted comment shows the placeholder.
func TestDiscussionReadOnly(t *testing.T) {
	_, n := renderDebug(t, Discussion{Comments: sampleDiscussion()})
	if len(buttonsOf(n)) != 0 {
		t.Error("no OnReply: no Reply buttons")
	}
	if findText(n, "♥ 3") == nil {
		t.Error("the like count should be drawn as text")
	}
	if findFirst(n, func(x *core.Node) bool {
		return x.Type == "Paragraph" && strings.Contains(paragraphText(x), "This comment was deleted.")
	}) == nil {
		t.Error("a deleted comment shows the placeholder")
	}
	if findText(n, "Release notes?") != nil {
		t.Error("a deleted comment's body is not drawn")
	}
}

// Reply and Like report the comment's key; Like is a selected-state toggle
// with a stable name.
func TestDiscussionReplyAndLike(t *testing.T) {
	var replied, liked []string
	d := Discussion{
		Comments: sampleDiscussion(),
		OnReply:  func(k string) { replied = append(replied, k) },
		OnLike:   func(k string) { liked = append(liked, k) },
	}
	ctx, n := renderDebug(t, d)

	reply := labelled(n, "Reply to Ben")
	if reply == nil {
		t.Fatal("no \"Reply to Ben\"")
	}
	ctx.TriggerCallback(reply.Props["onClick"].(string))
	if !slices.Equal(replied, []string{"b"}) {
		t.Errorf("replied = %v", replied)
	}
	if labelled(n, "Reply to Eli") != nil {
		t.Error("a deleted comment offers no Reply")
	}

	like := labelled(n, "Like, 1") // Ben's
	if like == nil {
		t.Fatal("no \"Like, 1\"")
	}
	if like.Style.AccessibilitySelected != core.SelectedWhen(true) {
		t.Error("Ben's comment is liked: the toggle should be selected")
	}
	ctx.TriggerCallback(like.Props["onClick"].(string))
	if !slices.Equal(liked, []string{"b"}) {
		t.Errorf("liked = %v", liked)
	}
}

// Tapping the fold hides the replies, and tapping again brings them back.
func TestDiscussionFoldsAThread(t *testing.T) {
	d := Discussion{Comments: sampleDiscussion()}
	ctx, n := renderDebug(t, d)
	fold := findFirst(n, func(x *core.Node) bool {
		return x.Style.AccessibilityRole == core.RoleButton && x.Style.AccessibilityLabel == "2 replies"
	})
	if fold == nil || fold.Style.AccessibilityExpanded != core.ExpandedOpen {
		t.Fatal("Ana's thread should have an open \"2 replies\" fold")
	}
	ctx.TriggerCallback(fold.Props["onClick"].(string))

	n = renderPass(ctx, d)
	if findText(n, "Works on my Pixel.") != nil {
		t.Error("folded: Ben's reply should be hidden")
	}
	fold = findFirst(n, func(x *core.Node) bool {
		return x.Style.AccessibilityLabel == "2 replies" && x.Style.AccessibilityRole == core.RoleButton
	})
	if fold.Style.AccessibilityExpanded != core.ExpandedClosed {
		t.Error("the fold should now state collapsed")
	}
	ctx.TriggerCallback(fold.Props["onClick"].(string))
	if n = renderPass(ctx, d); findText(n, "Works on my Pixel.") == nil {
		t.Error("unfolded: Ben's reply should be back")
	}
}

// InitiallyCollapsed starts every thread folded, except one that holds the
// comment being answered: the composer is never folded away.
func TestDiscussionInitiallyCollapsedKeepsTheReplyOpen(t *testing.T) {
	_, n := renderDebug(t, Discussion{Comments: sampleDiscussion(), InitiallyCollapsed: true})
	if findText(n, "Works on my Pixel.") != nil {
		t.Error("InitiallyCollapsed should fold Ana's thread")
	}

	_, n = renderDebug(t, Discussion{
		Comments: sampleDiscussion(), InitiallyCollapsed: true,
		ReplyingTo: "c", Composer: core.Text("composer-here"),
	})
	if findText(n, "composer-here") == nil || findText(n, "Same on iOS.") == nil {
		t.Error("the thread holding the reply target should be open, composer and all")
	}
}

// The composer sits under the comment named by ReplyingTo, at the top when
// that is empty, and at the top (with a concern) when it names nobody.
func TestDiscussionComposerPlacement(t *testing.T) {
	composer := core.Text("composer-here")

	_, n := renderDebug(t, Discussion{Comments: sampleDiscussion(), Composer: composer})
	if n.Children[0].Props["content"] != "composer-here" {
		t.Error("ReplyingTo empty: the composer should be first")
	}

	_, n = renderDebug(t, Discussion{Comments: sampleDiscussion(), Composer: composer, ReplyingTo: "d"})
	dee := findFirst(n, func(x *core.Node) bool { return x.Key == "c:d" })
	if findText(dee, "composer-here") == nil {
		t.Error("ReplyingTo d: the composer should be inside Dee's comment")
	}

	core.SetDebugMode(true)
	core.ClearConcerns()
	t.Cleanup(func() { core.SetDebugMode(false); core.ClearConcerns() })
	ctx := core.NewContext()
	ctx.BeginRenderPass()
	n = Discussion{Comments: sampleDiscussion(), Composer: composer, ReplyingTo: "zz"}.Render(ctx)
	ctx.EndRenderPass()
	if !strings.Contains(core.DumpConcerns(), ConcernDiscussionReplyTargetMissing) {
		t.Error("an unknown ReplyingTo should be reported")
	}
	if n.Children[0].Props["content"] != "composer-here" {
		t.Error("an unknown ReplyingTo should still draw the composer at the top")
	}
}

// Past MaxDepth the indent stops growing, and the outline keeps counting.
func TestDiscussionMaxDepth(t *testing.T) {
	_, n := renderDebug(t, Discussion{Comments: sampleDiscussion(), MaxDepth: 1})
	// One thread line: under Ana. Ben's replies are not indented again.
	lines := 0
	var walk func(*core.Node)
	walk = func(x *core.Node) {
		if x.Style.Width == "2px" {
			lines++
		}
		for _, c := range x.Children {
			walk(c)
		}
	}
	walk(n)
	if lines != 1 {
		t.Errorf("thread lines = %d, want 1 at MaxDepth 1", lines)
	}
	if _, levels := itemsOf(n); !slices.Equal(levels, []int{1, 2, 3, 2, 1}) {
		t.Errorf("levels = %v; the outline must not be capped with the indent", levels)
	}
}

// The column beside each thread line may shrink below its content's
// min-content width. Without MinWidth 0 a Composer drawn one level down (an
// input's intrinsic width plus its button) pushed the whole level of replies
// past a phone's right edge on the web, as lesson 4.40 showed.
func TestDiscussionIndentedRepliesMayShrink(t *testing.T) {
	_, n := renderDebug(t, Discussion{
		Comments:   sampleDiscussion(),
		ReplyingTo: "b",
		Composer:   InputRow{Value: "", OnChange: func(string) {}, Button: Button{Label: "Post"}},
	})
	columns := 0
	var walk func(*core.Node)
	walk = func(x *core.Node) {
		// The row holding a thread line: its second child is the replies.
		if len(x.Children) == 2 && x.Children[0].Style.Width == "2px" {
			columns++
			if got := x.Children[1].Style.MinWidth; got != "0" {
				t.Errorf("the replies column beside a thread line has MinWidth %q, want \"0\"", got)
			}
		}
		for _, c := range x.Children {
			walk(c)
		}
	}
	walk(n)
	if columns != 2 {
		t.Errorf("found %d indented reply columns, want 2 (under Ana and under Ben)", columns)
	}
}

func TestDiscussionEmpty(t *testing.T) {
	_, n := renderDebug(t, Discussion{})
	if findText(n, "No comments yet.") == nil {
		t.Error("the empty text is missing")
	}
	_, n = renderDebug(t, Discussion{EmptyText: "Be the first."})
	if findText(n, "Be the first.") == nil {
		t.Error("EmptyText should replace the default")
	}
}

func TestDiscussionCallerStyleWins(t *testing.T) {
	_, n := renderDebug(t, Discussion{Comments: sampleDiscussion(), Style: []core.StyleProp{core.Gap(37)}})
	if n.Style.Gap != 37 {
		t.Errorf("gap = %v, want the caller's 37", n.Style.Gap)
	}
}

func TestDiscussionEveryTheme(t *testing.T) {
	for name, th := range core.BundledThemes() {
		core.SetDebugMode(true)
		core.ClearConcerns()
		ctx := core.NewContext().WithTheme(th)
		ctx.BeginRenderPass()
		n := Discussion{Comments: sampleDiscussion(), OnReply: func(string) {}, OnLike: func(string) {}}.Render(ctx)
		ctx.EndRenderPass()
		core.AuditTree(n)
		if dump := core.DumpConcerns(); dump != "" {
			t.Errorf("%s: %s", name, dump)
		}
		core.SetDebugMode(false)
		core.ClearConcerns()
	}
}
