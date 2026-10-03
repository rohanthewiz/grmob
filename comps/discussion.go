package comps

import (
	"strconv"

	"github.com/rohanthewiz/grmob/core"
)

// ConcernDiscussionReplyTargetMissing is raised, in debug builds only, when
// Discussion.ReplyingTo names a key no comment has. The composer is drawn at
// the top instead, so the reader's draft does not vanish, but it is no longer
// under the comment the app thinks it is answering.
const ConcernDiscussionReplyTargetMissing = "discussion-reply-target-missing"

// DiscussionComment is one comment and the replies under it.
type DiscussionComment struct {
	// Key identifies the comment for as long as it exists: a server ID, not
	// an index. It names the comment to OnReply and OnLike, keys its row,
	// and is what ReplyingTo is compared with.
	Key string

	// Author is drawn bold over the body; AvatarSrc, when set, is their
	// picture, and their initials are drawn otherwise.
	Author    string
	AvatarSrc string

	// Time is drawn as given, after the author ("2h", "Mar 4").
	Time string

	// Body is the comment's text.
	Body string

	// Likes is the count, Liked whether the reader is one of them.
	Likes int
	Liked bool

	// Deleted draws DeletedText in place of Body and offers no Reply or Like,
	// but keeps the comment's place and its replies: a deleted comment in
	// the middle of a thread is the context its replies were written in.
	Deleted bool

	// Replies are the answers to this comment, oldest first, each of which
	// may have replies of its own.
	Replies []DiscussionComment
}

// Discussion is a threaded comment section: comments, their replies indented
// under them with a thread line, a Reply and a Like on each, and threads that
// fold away. The comments under an article, a forum topic, the Q&A on a
// course lesson.
//
//	comps.Discussion{
//	    Title:      "12 comments",
//	    Comments:   comments,                 // []comps.DiscussionComment, a tree
//	    OnReply:    func(key string) { replyingTo.Set(key) },
//	    OnLike:     toggleLike,
//	    ReplyingTo: replyingTo.Get(),
//	    Composer:   comps.InputRow{Value: draft.Get(), OnChange: draft.Set, OnSubmit: post, …},
//	}
//
//	┌ Column ────────────────────────────────────────────────┐
//	│ 12 comments                                 heading 2  │
//	│ ┌ Column role=list ──────────────────────────────────┐ │
//	│ │ (A) Ana · 2h                    listitem, level 1  │ │
//	│ │     Has anyone tried the new build?                │ │
//	│ │     Reply   ♡ 3   ▾ 2 replies                      │ │
//	│ │  │ ┌ role=list ───────────────────────────────────┐│ │
//	│ │  │ │ (B) Ben · 1h               listitem, level 2 ││ │
//	│ │  │ │     Works on my Pixel.                       ││ │
//	│ │  │ │     Reply   ♥ 1                              ││ │
//	│ │  │ │     [ Composer, when ReplyingTo is Ben's ]   ││ │
//	│ │  │ └──────────────────────────────────────────────┘│ │
//	│ └────────────────────────────────────────────────────┘ │
//	└────────────────────────────────────────────────────────┘
//
// # Who holds what
//
// The comments are the caller's: they are server state other people change,
// as Poll's counts are. OnLike reports the key and the caller flips Liked and
// the count when its data comes back; OnReply reports the key and the caller
// sets ReplyingTo, which places Composer under that comment. With ReplyingTo
// empty the Composer, if any, is at the top, for a new thread. Posting is the
// Composer's business; the widget never sees the draft.
//
// Which threads are folded is the widget's own, Accordion's open/closed by
// another name: no application wants to read or persist it. A thread that
// holds the ReplyingTo comment is drawn open whatever its state, so the
// composer can never be folded away under the reader.
//
// # Hooks
//
// Discussion takes one hook (the folded set), so it must be rendered
// unconditionally, every pass. Composer moves from comment to comment as
// ReplyingTo changes, so it must be hook-free: an InputRow over the caller's
// own draft state, as above, is the shape.
//
// # Depth
//
// Each level of replies is indented by a thread line, up to MaxDepth levels;
// deeper replies keep that indent rather than walking off a phone's screen.
// A reply drawn at a capped depth is still a level deeper in the outline
// (below), so nothing is lost to a reader who cannot see the indent.
//
// # Accessibility
//
// The structure is a list of lists: each level of replies is a core.RoleList
// whose items carry core.AccessibilityNestingLevel, which the web states as
// aria-level ("level 2") and Android and iOS cannot express at all. For
// those two the header row is named with what the indent says, "Ben, reply
// to Ana, 1h". The name is dropped on the web, where the level already says
// it, and announced on both natives.
//
// Like is a Chip, a toggle: one stable name, "Like, 3", and its state as
// selected. Reply is named "Reply to Ana", since a column of identical "Reply"
// buttons is no help to a reader moving from control to control. The fold is
// the package's disclosure: a button that states expanded or collapsed and is
// named by its count, "2 replies".
//
// # Theme roles read
//
//	Author       Typography.Body, bold, Colors.TextPrimary
//	Time         Typography.Caption, Colors.TextSecondary
//	Body         Typography.Body, Colors.TextPrimary
//	Deleted      Typography.Body, italic, Colors.TextSecondary
//	Thread line  Colors.Border, centred under the avatar
//	Fold         Typography.Caption, Colors.Primary's ink tone
//	Gaps         Spacing.XS inside a comment, Spacing.MD between comments
type Discussion struct {
	// Title heads the section ("12 comments"). Empty draws no heading.
	Title string

	// HeadingLevel is the Title's tier; zero is 2.
	HeadingLevel int

	// Comments are the top-level comments, in the order to draw them.
	Comments []DiscussionComment

	// OnReply reports the key of the comment whose Reply was tapped. Nil
	// draws no Reply buttons.
	OnReply func(key string)

	// OnLike reports the key of the comment whose Like was tapped. Nil draws
	// each count as text rather than a toggle.
	OnLike func(key string)

	// ReplyingTo is the key Composer is drawn under. Empty draws it at the
	// top.
	ReplyingTo string

	// Composer is the reply or new-comment box. Nil draws none. It must be
	// hook-free; see Hooks.
	Composer core.View

	// MaxDepth caps the indent at this many levels of replies. Zero is 4.
	MaxDepth int

	// InitiallyCollapsed starts every thread with its replies folded. It
	// seeds the widget's state on the first pass only.
	InitiallyCollapsed bool

	// EmptyText replaces "No comments yet." when Comments is empty.
	EmptyText string

	// ReplyLabel replaces "Reply", LikeLabel "Like" and DeletedText
	// "This comment was deleted.".
	ReplyLabel  string
	LikeLabel   string
	DeletedText string

	// Style is applied to the outer column after the widget's own props.
	Style []core.StyleProp
}

// Render draws the discussion. It takes one hook slot; see Hooks.
func (d Discussion) Render(ctx *core.Context) *core.Node {
	// Before any branch. The set holds the keys whose fold the reader has
	// flipped from InitiallyCollapsed, so it starts empty either way and a
	// comment that arrives later starts the way every other did.
	flipped := core.NewState(ctx, map[string]bool{})
	t := ctx.Theme()

	r := discussionRender{
		d:        d,
		t:        t,
		flipped:  flipped.Get(),
		maxDepth: d.MaxDepth,
		toggle: func(key string) {
			// A fresh map: State.Set compares nothing, but a map shared with
			// the last pass's closures would change under them.
			next := make(map[string]bool, len(flipped.Get())+1)
			for k, v := range flipped.Get() {
				next[k] = v
			}
			if next[key] {
				delete(next, key)
			} else {
				next[key] = true
			}
			flipped.Set(next)
		},
	}
	if r.maxDepth <= 0 {
		r.maxDepth = 4
	}

	items := make([]core.PropsAndChildren, 0, len(d.Style)+5)
	items = append(items, core.Padding(0), core.Gap(float64(t.Spacing.MD)))
	items = append(items, asProps(d.Style)...)

	if d.Title != "" {
		title := []core.StyleProp{core.UseStyle(t.Typography.Subtitle), core.FontWeight(core.Bold)}
		title = append(title, headingProps(d.HeadingLevel, headingLevelSection)...)
		items = append(items, core.Text(d.Title, title...))
	}

	target := d.ReplyingTo != "" && containsComment(d.Comments, d.ReplyingTo)
	if d.ReplyingTo != "" && !target && core.IsDebugMode() {
		core.ReportConcern(ConcernDiscussionReplyTargetMissing,
			"Discussion.ReplyingTo is \""+d.ReplyingTo+"\", which no comment has; the composer is drawn at the top")
	}
	if d.Composer != nil && !target {
		items = append(items, core.Keyed("composer", d.Composer))
	}

	if len(d.Comments) == 0 {
		items = append(items, core.Text(orDefault(d.EmptyText, "No comments yet."),
			core.UseStyle(t.Typography.Body),
			core.TextColor(t.Colors.TextSecondary),
			core.Align(core.AlignCenter),
		))
	} else {
		items = append(items, r.list(d.Comments, 0, ""))
	}
	return core.Column(items...).Render(ctx)
}

// discussionAvatar is the avatar's size, in points. The thread line's
// indent is derived from it, so the two stay aligned if it changes.
const discussionAvatar = 28

// discussionRender carries what every level of the recursion needs.
type discussionRender struct {
	d        Discussion
	t        *core.Theme
	flipped  map[string]bool
	maxDepth int
	toggle   func(key string)
}

// list is one level: the top-level comments, or one comment's replies. It
// holds items and nothing else, which is what lets it be a RoleList.
func (r discussionRender) list(comments []DiscussionComment, depth int, parent string) core.View {
	items := make([]core.PropsAndChildren, 0, len(comments)+3)
	items = append(items,
		core.Padding(0),
		core.Gap(float64(r.t.Spacing.MD)),
		core.AccessibilityRole(core.RoleList),
	)
	for _, c := range comments {
		// "c:" so a comment key can never collide with the composer's.
		items = append(items, core.Keyed("c:"+c.Key, r.comment(c, depth, parent)))
	}
	return core.Column(items...)
}

// comment draws one comment and, unless folded, its replies.
func (r discussionRender) comment(c DiscussionComment, depth int, parent string) core.View {
	t, d := r.t, r.d

	// The header: avatar, author, time. Named for the natives; see the
	// type's Accessibility section.
	spoken := c.Author
	if parent != "" {
		spoken += ", reply to " + parent
	}
	if c.Time != "" {
		spoken += ", " + c.Time
	}
	header := []core.PropsAndChildren{
		core.Padding(0),
		core.Gap(float64(t.Spacing.SM)),
		core.AlignItemsProp(core.AlignItemsCenter),
		core.AccessibilityLabel(spoken),
		Avatar{
			Src:  c.AvatarSrc,
			Name: c.Author,
			Size: discussionAvatar,
			// The author's name is the next thing read; the picture
			// would only say it first.
			Style: []core.StyleProp{core.AccessibilityHidden()},
		},
		core.Text(c.Author,
			core.UseStyle(t.Typography.Body),
			core.FontWeight(core.Bold),
			core.TextColor(t.Colors.TextPrimary),
		),
	}
	if c.Time != "" {
		header = append(header, core.Text(c.Time,
			core.UseStyle(t.Typography.Caption),
			core.TextColor(t.Colors.TextSecondary),
		))
	}

	var body core.View
	if c.Deleted {
		body = core.Paragraph([]core.Span{{
			Text:   orDefault(d.DeletedText, "This comment was deleted."),
			Italic: true,
			Color:  t.Colors.TextSecondary,
		}}, core.UseStyle(t.Typography.Body))
	} else {
		body = core.Text(c.Body,
			core.UseStyle(t.Typography.Body),
			core.TextColor(t.Colors.TextPrimary),
		)
	}

	// The fold is open when the reader has not flipped it from the
	// default, or when the comment being answered is somewhere inside.
	open := r.flipped[c.Key] == d.InitiallyCollapsed ||
		(d.ReplyingTo != "" && containsComment(c.Replies, d.ReplyingTo))

	items := []core.PropsAndChildren{
		core.Padding(0),
		core.Gap(float64(t.Spacing.XS)),
		core.AccessibilityRole(core.RoleListItem),
		core.AccessibilityNestingLevel(depth + 1),
		core.Row(header...),
		body,
	}
	if actions := r.actions(c, open); actions != nil {
		items = append(items, actions)
	}
	if d.Composer != nil && d.ReplyingTo != "" && d.ReplyingTo == c.Key {
		items = append(items, core.Keyed("composer", d.Composer))
	}
	if len(c.Replies) > 0 && open {
		replies := r.list(c.Replies, depth+1, c.Author)
		if depth+1 <= r.maxDepth {
			// The thread line: a rule down the side of the replies, the
			// indent itself. A leading spacer rather than padding, so it
			// sits on the start side under RTL on the web too.
			//
			// The line is centred under the avatar above it, and the same
			// margin after it makes the whole indent one avatar wide, so a
			// reply's avatar starts where its parent's ended:
			//
			//	(A) Ana          ← avatar, discussionAvatar wide
			//	 │ (B) Ben       ← margin, 2px rule, margin = one avatar
			replies = core.Row(
				core.Padding(0),
				core.Gap(0),
				core.Box(
					core.Padding(0),
					core.Width("2px"),
					core.MarginHorizontal(discussionAvatar/2-1),
					core.AlignSelf(core.AlignItemsStretch),
					core.BackgroundColor(t.Colors.Border),
					core.AccessibilityHidden(),
				),
				core.Column(core.Padding(0), core.FlexGrow(1), core.FlexBasis("0"), replies),
			)
		}
		items = append(items, replies)
	}
	return core.Column(items...)
}

// actions is the row under a comment: Reply, Like, and the fold. Nil when
// there is nothing to put in it, so a read-only leaf has no empty row.
func (r discussionRender) actions(c DiscussionComment, open bool) core.View {
	t, d := r.t, r.d
	row := []core.PropsAndChildren{
		core.Padding(0),
		core.Gap(float64(t.Spacing.SM)),
		core.AlignItemsProp(core.AlignItemsCenter),
		core.FlexWrap(true),
	}
	n := len(row)

	if d.OnReply != nil && !c.Deleted {
		key, reply := c.Key, d.OnReply
		row = append(row, Button{
			Label:              orDefault(d.ReplyLabel, "Reply"),
			OnTap:              func() { reply(key) },
			Emphasis:           EmphasisGhost,
			AccessibilityLabel: orDefault(d.ReplyLabel, "Reply") + " to " + c.Author,
		})
	}

	like := orDefault(d.LikeLabel, "Like")
	switch {
	case d.OnLike != nil && !c.Deleted:
		glyph := "♡"
		if c.Liked {
			glyph = "♥"
		}
		label := glyph + " " + like
		if c.Likes > 0 {
			label = glyph + " " + strconv.Itoa(c.Likes)
		}
		key, onLike := c.Key, d.OnLike
		row = append(row, Chip{
			Label:              label,
			Selected:           c.Liked,
			OnTap:              func() { onLike(key) },
			AccessibilityLabel: like + ", " + strconv.Itoa(c.Likes),
		})
	case c.Likes > 0:
		// Read-only: the count as text, not a toggle that does nothing.
		row = append(row, core.Text("♥ "+strconv.Itoa(c.Likes),
			core.UseStyle(t.Typography.Caption),
			core.TextColor(t.Colors.TextSecondary),
		))
	}

	if len(c.Replies) > 0 {
		count := "1 reply"
		if len(c.Replies) != 1 {
			count = strconv.Itoa(len(c.Replies)) + " replies"
		}
		key, toggle := c.Key, r.toggle
		ink := VariantDefault.AsInk(t)
		row = append(row, disclosure{
			Label:        count,
			Hint:         "Shows or hides the replies",
			Expanded:     open,
			OnToggle:     func() { toggle(key) },
			ChevronStyle: []core.StyleProp{core.UseStyle(t.Typography.Caption), core.TextColor(ink)},
			// Padding(0): the disclosure's row is a core.Row, which arrives
			// with the theme's screen inset, and this one sits inline.
			ControlStyle: []core.StyleProp{core.Padding(0), core.Gap(float64(t.Spacing.XS))},
			Control: []core.View{core.Text(count,
				core.UseStyle(t.Typography.Caption),
				core.TextColor(ink),
			)},
		}.view())
	}

	if len(row) == n {
		return nil
	}
	return core.Row(row...)
}

// containsComment reports whether key is any comment in the tree.
func containsComment(comments []DiscussionComment, key string) bool {
	for _, c := range comments {
		if c.Key == key || containsComment(c.Replies, key) {
			return true
		}
	}
	return false
}
