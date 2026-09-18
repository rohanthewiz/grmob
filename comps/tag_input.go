package comps

import (
	"slices"
	"strings"
	"unicode/utf8"

	"github.com/rohanthewiz/grmob/core"
)

// ConcernTagInputInert is raised, in debug builds only, when a TagInput has
// no OnChange. Typing a tag and pressing return clears the draft and adds
// nothing, and a ✕ removes nothing — a field that swallows what it is given
// and looks exactly like one that is working until the form is submitted
// without the tags. The bar PasswordField's inert case is reported against.
const ConcernTagInputInert = "tag-input-inert"

// TagInput is a set of short strings the reader types one at a time: email
// recipients, labels on a note, interests on a profile. The tags wrap above
// the input, and each has its own ✕.
//
//	comps.TagInput{
//	    Tags:        tags.Get(),
//	    OnChange:    tags.Set,
//	    Label:       "Labels",
//	    Placeholder: "Add a label",
//	}
//
//	┌ Column ─────────────────────────────────────────────┐
//	│ ┌ Row  role=list  wrap ───────────────────────────┐ │
//	│ │ ( design  ✕ ) ( urgent  ✕ ) ( q3 roadmap  ✕ )   │ │  one listitem
//	│ │ ( later  ✕ )                                    │ │  per tag
//	│ └─────────────────────────────────────────────────┘ │
//	│ ┌ InputWithSubmit ────────────────────────────────┐ │
//	│ │ Add a label                                     │ │  return or ","
//	│ └─────────────────────────────────────────────────┘ │  commits the draft
//	└─────────────────────────────────────────────────────┘
//
// # Committing a tag
//
// A tag is committed by the keyboard's return / done action, or by typing a
// separator (Separators, "," by default). The two paths are one rule over the
// input's text:
//
//	"design"          nothing yet; the draft is "design"
//	"design,"         commit "design"; the draft is ""
//	"a, b, c"         (a paste) commit "a" and "b"; the draft is " c"
//	"a, b, c,"        commit all three; the draft is ""
//	return on " c"    commit "c"; the draft is ""
//
// Every committed piece is trimmed, and empty pieces and exact duplicates of
// a tag already held are dropped — the tag is on screen already, so the
// reader sees the result they wanted. Once Max tags are held the rest of a
// paste is discarded and the input is disabled; it stays visible so its
// label and placeholder still say what the field is for.
//
// # The ✕ is its own button; the tag is not one
//
// Chip has exactly one tap target. A ✕ inside it would be a button inside a
// button, which no accessibility tree can represent, and a chip that removed
// itself when tapped would bind the tag's largest surface to a destructive
// action. So a tag is a pill Row holding the text and a ghost ✕ Button named
// "Remove design"; the text is inert.
//
// # The draft is the widget's
//
// The half-typed text is held here, in a hook, and not by the caller. It is
// DateRangePicker's test: no application wants a half-made tag — a form
// submits the set, and "desi" is not a member of it. Owning it means the hook
// rules apply: render a TagInput unconditionally, in a stable position.
//
// There is no "backspace in an empty input removes the last tag". The input
// reports its text, not its keys, and the text of an empty input does not
// change when backspace is pressed — PINInput met the same wall.
//
// # Accessibility
//
// The strip is a RoleList and each tag a listitem, so a reader hears "list, 3
// items" and then each tag followed by its "Remove …" button. Neither the
// list nor the items are given a name: an item's name would stand in for its
// children on the targets that merge a labelled container, and the ✕ button
// inside it would stop being reachable. The input is named by Label.
//
// # Theme roles read
//
//	Pill       Colors.Surface fill, ColorPalette.BorderColor hairline
//	Tag text   Typography.Body
//	✕          a ghost Button (Primary's on-light tone)
//	Gaps       Spacing.XS between tags and between the strip and the input
type TagInput struct {
	// Tags are the committed tags, owned by the caller.
	Tags []string

	// OnChange receives the whole new set after every commit or removal.
	// Nil reports ConcernTagInputInert.
	OnChange func([]string)

	// Placeholder is drawn in the empty input.
	Placeholder string

	// Label names the input for assistive technology ("Labels"). It is not
	// drawn; a FormField around the widget is the visible label.
	Label string

	// Max caps the number of tags; 0 means no cap.
	Max int

	// Separators are the characters that commit the draft when typed; empty
	// gives ",".
	Separators string

	// RemoveLabel prefixes each ✕'s accessible name; empty gives "Remove".
	RemoveLabel string

	// Disabled disables the input and every ✕.
	Disabled bool

	// Style is applied to the outer column after its defaults.
	Style []core.StyleProp
}

// Render draws the tags and the input. It takes two hooks: the draft, and
// the focus ref the input keeps across a return.
func (in TagInput) Render(ctx *core.Context) *core.Node {
	t := ctx.Theme()

	// Before any branch: the hook rule PasswordField's doc spells out.
	draft := core.NewState(ctx, "")
	// Return commits a tag, and the reader's next move is the next tag, so
	// the input has to keep the keyboard. The web and Compose never let go
	// of it on a submit; SwiftUI does, by default, and a phone reader had to
	// tap the field again between every two tags (the simulator, lesson 5.8).
	// Asking for focus back on each submit is a no-op where focus never left,
	// and on iOS it returns the keyboard one pass after SwiftUI dropped it.
	field := core.UseFocusRef(ctx)

	if core.IsDebugMode() && in.OnChange == nil {
		core.ReportConcern(ConcernTagInputInert,
			"TagInput has no OnChange, so a committed tag is dropped and a ✕ removes nothing")
	}
	report := in.OnChange
	if report == nil {
		report = func([]string) {}
	}

	// Captured by value for the closures below: the handlers must act on the
	// set this pass drew, not on a later pass's TagInput.
	tags := in.Tags
	full := in.Max > 0 && len(tags) >= in.Max

	onChange := func(s string) {
		pieces, rest := splitDraft(s, orDefault(in.Separators, ","))
		if next, changed := commitTags(tags, pieces, in.Max); changed {
			report(next)
		}
		draft.Set(rest)
	}
	onSubmit := func() {
		if next, changed := commitTags(tags, []string{draft.Get()}, in.Max); changed {
			report(next)
		}
		draft.Set("")
		core.Focus(field)
	}

	items := make([]core.PropsAndChildren, 0, len(in.Style)+4)
	items = append(items,
		core.Padding(0),
		core.Gap(float64(t.Spacing.XS)),
	)
	items = append(items, asProps(in.Style)...)

	// A Row with no tags would still take a Gap's worth of height above the
	// input, so the strip is left out entirely when there is nothing in it.
	if len(tags) > 0 {
		strip := make([]core.PropsAndChildren, 0, len(tags)+4)
		strip = append(strip,
			core.Padding(0),
			core.FlexWrap(true),
			core.Gap(float64(t.Spacing.XS)),
			core.AccessibilityRole(core.RoleList),
		)
		removeLabel := orDefault(in.RemoveLabel, "Remove")
		for i, tag := range tags {
			strip = append(strip, in.pill(t, tag, removeLabel+" "+tag, func() {
				report(removeTag(tags, i))
			}))
		}
		items = append(items, core.Row(strip...))
	}

	input := []core.PropsAndChildren{core.FocusTarget(field)}
	if in.Label != "" {
		input = append(input, core.AccessibilityLabel(in.Label))
	}
	if in.Disabled || full {
		input = append(input, core.Disabled(true))
	}
	items = append(items, core.InputWithSubmit(draft.Get(), in.Placeholder, onChange, onSubmit, input...))

	return core.Column(items...).Render(ctx)
}

// pill is one tag: the inert text and its ✕, in a rounded Surface box.
func (in TagInput) pill(t *core.Theme, tag, removeName string, remove func()) core.View {
	return core.Row(
		core.AccessibilityRole(core.RoleListItem),
		core.AccessibilityNestingLevel(1),
		core.AlignItemsProp(core.AlignItemsCenter),
		core.Gap(2),
		// Asymmetric: the text needs room on its leading side, while the ✕
		// carries its own padding on the trailing one.
		core.PaddingVertical(2),
		core.PaddingLeft(10),
		core.PaddingRight(2),
		core.BorderRadius(16),
		core.BackgroundColor(t.Colors.Surface),
		core.BorderWidth(1),
		core.BorderColor(t.Colors.BorderColor()),
		core.Text(tag, core.UseStyle(t.Typography.Body)),
		Button{
			Label:              "✕",
			OnTap:              remove,
			Emphasis:           EmphasisGhost,
			Disabled:           in.Disabled,
			AccessibilityLabel: removeName,
			Style: []core.StyleProp{
				core.PaddingVertical(2),
				core.PaddingHorizontal(6),
				core.FlexShrink(0),
			},
		},
	)
}

// splitDraft cuts the input's text at every separator. Every piece before the
// last separator is complete; what follows it is the new draft. Text with no
// separator is all draft.
func splitDraft(s, seps string) (complete []string, rest string) {
	last := strings.LastIndexAny(s, seps)
	if last < 0 {
		return nil, s
	}
	// The separator may be more than one byte ("、"), so the rest starts
	// after the whole rune found, not after one byte.
	_, size := utf8.DecodeRuneInString(s[last:])
	complete = strings.FieldsFunc(s[:last], func(r rune) bool { return strings.ContainsRune(seps, r) })
	return complete, s[last+size:]
}

// commitTags appends each trimmed, non-empty, not-yet-held piece to a copy of
// tags, stopping at max (0 is no cap). It never modifies tags: that slice is
// the caller's state.
func commitTags(tags, pieces []string, max int) (next []string, changed bool) {
	next = append([]string(nil), tags...)
	for _, p := range pieces {
		p = strings.TrimSpace(p)
		if p == "" || slices.Contains(next, p) {
			continue
		}
		if max > 0 && len(next) >= max {
			break
		}
		next = append(next, p)
		changed = true
	}
	return next, changed
}

// removeTag returns a copy of tags without the i-th.
func removeTag(tags []string, i int) []string {
	next := make([]string, 0, len(tags))
	next = append(next, tags[:i]...)
	return append(next, tags[i+1:]...)
}
