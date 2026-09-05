package tutorial

import (
	"fmt"
	"maps"
	"strconv"
	"strings"
	"time"

	"github.com/rohanthewiz/grmob/components"
	"github.com/rohanthewiz/grmob/core"
	"github.com/rohanthewiz/grmob/hooks"
)

// chapter4 — The Widget Library: a tour of the components package and the
// contract every widget in it shares. Widgets are structs with named fields
// implementing core.View, so they compose exactly like the primitives and can
// grow an optional knob without breaking a call site; looks come from
// ctx.Theme() with Style as the per-use override; and state stays with the
// caller — a widget renders what it is passed and reports intent through
// callbacks. Accordion, the package's one widget that owns state, gets a
// lesson of its own precisely because owning state means inheriting hook
// obligations.
func chapter4() Chapter {
	return Chapter{
		Title:   "The Widget Library",
		Icon:    "🧩",
		Summary: "Buttons, pills, rows, accordions, tabs, tables and the screen furniture around them — the components package and its controlled-widget contract.",
		Lessons: []Lesson{
			lessonButtons(),
			lessonPills(),
			lessonListRow(),
			lessonAccordion(),
			lessonTabs(),
			lessonCollections(),
			lessonScreenFurniture(),
			lessonEndlessFeeds(),
			lessonCalendars(),
		},
	}
}

// --- 4.1 -----------------------------------------------------------------

// variantNames double as the 4.1 segment captions and the constant-name
// suffixes ("Success" → components.VariantSuccess): the demo's printed struct
// literal is built by pasting them onto the type prefix, so captions and code
// cannot disagree. variantValues is the parallel value table; 4.2 reuses it
// with its own captions, which is the point of Variant being shared — one
// vocabulary across the whole package.
var (
	variantNames  = []string{"Default", "Success", "Warning", "Error"}
	variantValues = []components.Variant{
		components.VariantDefault,
		components.VariantSuccess,
		components.VariantWarning,
		components.VariantError,
	}
	emphasisNames  = []string{"Filled", "Outlined", "Ghost"}
	emphasisValues = []components.Emphasis{
		components.EmphasisFilled,
		components.EmphasisOutlined,
		components.EmphasisGhost,
	}
)

// buttonSnippet prints the Button literal the demo's current knobs would
// build, omitting every zero-value field. The omission is the lesson: what is
// not set contributes nothing, so the printed struct is the whole truth about
// the button on screen.
func buttonSnippet(variant, emphasis int, disabled, fullWidth bool) string {
	lines := []string{
		"components.Button{",
		`    Label: "Save changes",`,
		"    OnTap: save,",
	}
	if variant != 0 {
		lines = append(lines, "    Variant: components.Variant"+variantNames[variant]+",")
	}
	if emphasis != 0 {
		lines = append(lines, "    Emphasis: components.Emphasis"+emphasisNames[emphasis]+",")
	}
	if disabled {
		lines = append(lines, "    Disabled: true,")
	}
	if fullWidth {
		lines = append(lines, "    FullWidth: true,")
	}
	lines = append(lines, "}")
	return strings.Join(lines, "\n")
}

func lessonButtons() Lesson {
	return Lesson{
		Title:   "Buttons: two axes",
		Summary: "Variant says what an action means; Emphasis says how loudly — and zero values change nothing.",
		Body: func(ctx *core.Context) core.View {
			variant := core.NewState(ctx, 0)
			emphasis := core.NewState(ctx, 0)
			disabled := core.NewState(ctx, false)
			fullWidth := core.NewState(ctx, false)
			taps := core.NewState(ctx, 0)

			return core.Column(
				core.Gap(14),
				prose("Every button you have tapped in this tutorial was a components.Button. "+
					"The widget library is structs with named fields, each implementing core.View, "+
					"so widgets compose exactly like the primitives do — and a field can be added "+
					"without breaking a single call site:"),
				codeBlock(`components.Button{Label: "Save", OnTap: save}   // the theme's Button, untouched

components.Button{Label: "Delete", OnTap: rm,
    Variant: components.VariantError}            // meaning: a palette status role

components.Button{Label: "Cancel", OnTap: back,
    Emphasis: components.EmphasisOutlined}       // weight: how much color it spends`),
				prose("Color is two orthogonal axes, not one enum. Variant picks which role the "+
					"button spends — Success, Warning, Error, or Default for the theme's Primary. "+
					"Emphasis picks how much of it: Filled, Outlined, or Ghost. A flat enum would "+
					"need a value per combination the moment a design wants an outlined destructive "+
					"button. And both zero values contribute no style props at all, so the plain "+
					"Button{Label, OnTap} renders exactly core.Button — whatever the theme chose "+
					"for its Button base shows through untouched."),
				demoPanel("Every knob re-renders one struct — and prints the literal that would build it.",
					caption("Variant — what the action means:"),
					components.SegmentedControl{
						Style:     segWrap,
						Labels:    variantNames,
						Selected:  variant.Get(),
						OnSelect:  func(i int) { variant.Set(i) },
						KeyPrefix: "btn-variant-",
					},
					caption("Emphasis — how loudly it says it:"),
					components.SegmentedControl{
						Style:     segWrap,
						Labels:    emphasisNames,
						Selected:  emphasis.Get(),
						OnSelect:  func(i int) { emphasis.Set(i) },
						KeyPrefix: "btn-emphasis-",
					},
					checkRow("Disabled", disabled),
					checkRow("Full width", fullWidth),
					components.Button{
						Label:     "Save changes",
						Variant:   variantValues[variant.Get()],
						Emphasis:  emphasisValues[emphasis.Get()],
						Disabled:  disabled.Get(),
						FullWidth: fullWidth.Get(),
						OnTap:     func() { taps.Set(taps.Get() + 1) },
					},
					caption(fmt.Sprintf("taps landed: %d", taps.Get())),
					core.If(disabled.Get(),
						caption("Disabled is one muted look for every variant — a dimmed red would "+
							"still read as danger, so an inert control stops signaling entirely."),
					),
					codeBlock(buttonSnippet(variant.Get(), emphasis.Get(), disabled.Get(), fullWidth.Get())),
				),
				keyPoints(
					"Variant × Emphasis are orthogonal: which role a button spends, and how much of it — an outlined error button needs no fifth enum value.",
					"Zero values contribute nothing: Button{Label, OnTap} is exactly core.Button, the theme's own Button base untouched.",
					"Disabled is three contracts at once: the muted look, the platform refusing to dispatch (and announcing the state), and a no-op handler kept registered so a racing tap can't crash.",
					"Filled picks its label ink by WCAG contrast against the fill; Outlined and Ghost wear the raw role color — prefer Filled for status actions.",
				),
			)
		},
	}
}

// --- 4.2 -----------------------------------------------------------------

// pillStatusLabels map index-for-index onto variantValues: Draft is the
// default Primary, Live is success, Expiring warns, Failed errs. One shared
// index is the whole wiring — the same index-is-the-value contract the
// chapter-2 conditionals demo used.
var pillStatusLabels = []string{"Draft", "Live", "Expiring", "Failed"}

// pillTopics feed the multi-select chip group. Kept as a slice, not a map, so
// the chips and the picked-summary caption render in one stable order.
var pillTopics = []string{"goroutines", "generics", "reflection", "testing"}

func lessonPills() Lesson {
	return Lesson{
		Title:   "Badges, chips & segments",
		Summary: "One pill family: Badge states, Chip selects, SegmentedControl is chips-plus-loop, extracted.",
		Body: func(ctx *core.Context) core.View {
			status := core.NewState(ctx, 0)
			picked := core.NewState(ctx, map[string]bool{})

			// Copy, flip, Set: the immutable-update rule from chapter 2,
			// applied to a map. Earlier renders still hold the old map, so the
			// handler must never mutate it in place.
			togglePick := func(topic string) func() {
				return func() {
					next := make(map[string]bool, len(picked.Get())+1)
					maps.Copy(next, picked.Get())
					if next[topic] {
						delete(next, topic)
					} else {
						next[topic] = true
					}
					picked.Set(next)
				}
			}

			// Derived, not stored: the summary re-reads the map through the
			// topics slice each pass, so it can never drift from the chips.
			var chosen []string
			for _, topic := range pillTopics {
				if picked.Get()[topic] {
					chosen = append(chosen, topic)
				}
			}

			// FlexWrap for the same reason segWrap exists on this chapter's
			// segmented controls: four topic captions are wider than a phone,
			// and without wrapping the last chip ("testing") sat off the right
			// edge of the screen — invisible in a multi-select group whose
			// point is that you can see what you picked.
			chips := []core.PropsAndChildren{core.Gap(8), core.FlexWrap(true)}
			for _, topic := range pillTopics {
				chips = append(chips, core.Keyed("topic-"+topic, components.Chip{
					Label:             topic,
					Selected:          picked.Get()[topic],
					OnTap:             togglePick(topic),
					AccessibilityHint: "Toggles this topic",
				}))
			}

			return core.Column(
				core.Gap(14),
				prose("Three widgets, one family tree. Badge is the pill that cannot be tapped — "+
					"it states. Chip is the pill that can — controlled, like every input: it renders "+
					"your Selected and reports taps. SegmentedControl is what falls out of writing "+
					"\"a row of chips plus a loop\" enough times: single-select, where Selected is an "+
					"index into Labels."),
				codeBlock(`components.Badge{Text: "Live", Variant: components.VariantSuccess}

components.Chip{
    Label:    topic,
    Selected: picked[topic], // caller state — the chip holds none
    OnTap:    toggle(topic),
}

components.SegmentedControl{
	Style:     segWrap,
    Labels:   []string{"Draft", "Live", "Expiring", "Failed"},
    Selected: status.Get(),
    OnSelect: func(i int) { status.Set(i) },
}`),
				prose("Two contracts to notice. Color reinforces but text carries: a Variant tints "+
					"the pill, and the label still has to say \"Failed\" on its own, because nothing "+
					"announces a tint to a screen reader and nothing should — that is WCAG 1.4.1. "+
					"And selection lives with you: tapping a chip below flips nothing inside the "+
					"chip. The handler copies a map, flips one key, Sets it, and the chips re-render "+
					"from it like any other state."),
				demoPanel("A badge fed by a segmented control, and a chip group over one map.",
					caption("Pick a status — the Badge takes its variant from the same index:"),
					components.SegmentedControl{
						Style:     segWrap,
						Labels:    pillStatusLabels,
						Selected:  status.Get(),
						OnSelect:  func(i int) { status.Set(i) },
						KeyPrefix: "pill-status-",
					},
					core.Row(
						core.Gap(8),
						core.AlignItemsProp(core.AlignItemsCenter),
						components.Badge{
							Text:    pillStatusLabels[status.Get()],
							Variant: variantValues[status.Get()],
						},
						caption(fmt.Sprintf("← Badge{Text: %q, Variant: components.Variant%s}",
							pillStatusLabels[status.Get()], variantNames[status.Get()])),
					),
					components.Separator{},
					caption("Chips — a multi-select over one map in this lesson's state:"),
					core.Row(chips...),
					core.IfElse(len(chosen) == 0,
						caption("nothing picked — every chip renders Selected straight from the map"),
						caption("picked: "+strings.Join(chosen, " · ")),
					),
				),
				keyPoints(
					"Badge is a statement, Chip a controlled toggle, SegmentedControl a controlled index — none of them holds state.",
					"Color reinforces, text carries: the label must say \"Failed\" by itself; no renderer announces a tint (WCAG 1.4.1).",
					"Update the chips' map immutably — copy, flip, Set — the chapter-2 slice rule, applied to a map.",
					"Key looped chips, and give twin segmented controls a KeyPrefix, so the reconciler matches pills by identity, not position.",
				),
			)
		},
	}
}

// --- 4.3 -----------------------------------------------------------------

// teamMember is the roster demo's row data. badge is the optional trailing
// status — "" renders a chevron instead — so one loop exercises both trailing
// shapes the ListRow docs describe.
type teamMember struct {
	name, role, badge string
}

var teamMembers = []teamMember{
	{"June Gopher", "Runtime wrangler", "on call"},
	{"Rex Burrows", "Reconciler referee", ""},
	{"Sal Tunnels", "Theme therapist", ""},
}

func lessonListRow() Lesson {
	return Lesson{
		Title:   "ListRow & Avatar",
		Summary: "Slots over hand-rolled rows: leading control, growing middle, pinned trailing — selection controlled.",
		Body: func(ctx *core.Context) core.View {
			// The selected member's name, "" for none. A name rather than an
			// index because the row's identity is the member, not the position
			// — the same reasoning as the chapter-2 keyed list.
			selected := core.NewState(ctx, "")

			rows := []core.PropsAndChildren{core.Gap(6)}
			for _, m := range teamMembers {
				var trailing core.View = caption("›")
				if m.badge != "" {
					trailing = components.Badge{Text: m.badge, Variant: components.VariantWarning}
				}
				rows = append(rows, core.Keyed(m.name, components.ListRow{
					Leading:  components.Avatar{Name: m.name, Size: 36},
					Title:    m.name,
					Subtitle: m.role,
					Trailing: trailing,
					Selected: selected.Get() == m.name,
					OnTap: func() {
						// Tap toggles: re-tapping the selected row clears it.
						if selected.Get() == m.name {
							selected.Set("")
						} else {
							selected.Set(m.name)
						}
					},
					AccessibilityLabel: m.name + ", " + m.role,
				}))
			}

			return core.Column(
				core.Gap(14),
				prose("ListRow is the leading / title / trailing shape every list ends up "+
					"hand-rolling: a checkbox and a task, an avatar and a name, a label and an "+
					"amount. Its fields are slots — Leading, Trailing, and the Content override "+
					"take any core.View — which is the library's composition idiom: where a struct "+
					"field is a core.View, anything can sit in it."),
				codeBlock(`components.ListRow{
    Leading:  components.Avatar{Name: m.Name, Size: 36},
    Title:    m.Name,
    Subtitle: m.Role,
    Trailing: components.Badge{Text: "on call",
        Variant: components.VariantWarning},
    Selected: selected.Get() == m.Name,
    OnTap:    func() { selected.Set(m.Name) },
}`),
				prose("The middle column is the row's spine: it renders even when empty, always "+
					"with FlexGrow(1), so it soaks up the slack and pins Trailing hard against the "+
					"edge in every configuration — a promise JustifyBetween cannot make once a slot "+
					"goes missing. The Avatar in the Leading slot earns its cameo: give it a Name "+
					"and it derives initials (first word, last word) and announces the name; give "+
					"it neither and it hides from assistive tech rather than announce \"image\"."),
				demoPanel("Tap a row to select it; tap it again to clear. Rows are Keyed by name.",
					core.Column(rows...),
					core.IfElse(selected.Get() == "",
						caption("No row selected — tap one."),
						caption("Selected: "+selected.Get()),
					),
				),
				keyPoints(
					"Slots are core.View fields: Leading and Trailing take any view; Content replaces Title/Subtitle when set.",
					"The middle column always renders, always FlexGrow(1) — that spine is what pins Trailing to the edge.",
					"OnTap nil registers nothing: a presentational row carries no callback and no gesture recognizer on any platform.",
					"ListRow synthesizes no accessibility name — its slots carry meaning it can't see, so you name the row; Avatar does synthesize one, and hides itself when nameless.",
				),
			)
		},
	}
}

// --- 4.4 -----------------------------------------------------------------

// The accordion demo's FAQ is about Accordion itself, so the demo teaches
// while being poked. The answers carry the liveness test's sentinel phrases —
// visible only while their section is expanded, which is the behavior under
// test.
type faqEntry struct {
	question string
	answer   string
	expanded bool
}

var accordionFAQ = []faqEntry{
	{"Where does the open state live?",
		"In a NewState slot on the caller's context — this lesson's own state. That makes " +
			"Accordion the package's one and only hook user; every other widget holds nothing.",
		true},
	{"Why not wrap one in core.If?",
		"Slots are claimed in call order. Hide one accordion and the next NewState reads its " +
			"neighbor's bool — debug mode reports exactly that as cursor drift.",
		false},
	{"What may Content hold?",
		"Any hook-free view: text, buttons, inputs bound to the lesson's state, whole columns. " +
			"Content renders only while expanded, so it must bring no NewState of its own.",
		false},
}

func lessonAccordion() Lesson {
	return Lesson{
		Title:   "Accordion: the stateful widget",
		Summary: "The package's one hook user: it claims a slot on your context, so the rules of hooks follow it.",
		Body: func(ctx *core.Context) core.View {
			faq := make([]core.View, 0, len(accordionFAQ)+1)
			for _, f := range accordionFAQ {
				faq = append(faq, components.Accordion{
					Title:             f.question,
					InitiallyExpanded: f.expanded,
					Content:           prose(f.answer),
				})
			}
			faq = append(faq, caption("Collapse and expand freely — every pass renders all "+
				"three accordions, so the slots never move."))

			return core.Column(
				core.Gap(14),
				prose("Every widget so far held no state. Accordion is the exception: its "+
					"expanded/collapsed bool lives in a NewState inside the widget's own Render. "+
					"There is no private slot for it to use — a widget's Render receives your "+
					"Context, so that NewState claims a slot on this lesson's state exactly as if "+
					"the lesson had called it directly."),
				codeBlock(`components.Accordion{
    Title:             "Shipping & returns",
    InitiallyExpanded: true,
    Content:           core.Text("Orders ship within two days."),
}`),
				prose("Owning a slot means owning the rules of hooks. Render an Accordion "+
					"unconditionally, in a stable position, every pass — hide one behind a "+
					"conditional and its slot shifts onto whatever renders next. The inverse rule "+
					"guards Content: it renders only while expanded, so a hook inside it would "+
					"appear and disappear with the toggle — the conditional-hook bug with a "+
					"tap-target attached. Interactive hook-free content is fine; its callbacks "+
					"re-register on every pass it is visible."),
				demoPanel("Three accordions, three bool slots on this lesson's context, claimed in render order.",
					faq...,
				),
				keyPoints(
					"Accordion calls NewState on your context — render it unconditionally, in a stable position, every pass.",
					"Content renders only while expanded, so it must be hook-free; interactive hook-free content is fine.",
					"InitiallyExpanded seeds the slot on the first pass only — after that the user's taps own it.",
					"Debug mode reports a conditionally rendered accordion as cursor drift — this tutorial's tests would fail before a device saw it.",
				),
			)
		},
	}
}

// --- 4.5 -----------------------------------------------------------------

// tabPageLabels name both the demo's segmented strip and the core.Tab items
// in the snippet, so the hand-rolled pattern and the native widget read as the
// same three pages.
var tabPageLabels = []string{"Info", "Stats", "Settings"}

func lessonTabs() Lesson {
	return Lesson{
		Title:   "Tabs & the wire contract",
		Summary: "Tabs is a facade over core.TabView, a node the native renderers draw — controlled like everything else.",
		Body: func(ctx *core.Context) core.View {
			page := core.NewState(ctx, 0)
			// The Settings page's checkbox state. Declared here, above the
			// page switch, because hooks must run on every pass — the page
			// that *reads* it comes and goes; the slot must not.
			notify := core.NewState(ctx, false)

			return core.Column(
				core.Gap(14),
				prose("components.Tabs is a different kind of widget: a facade. core.TabView "+
					"defines a wire contract — a \"TabView\" node whose tabs, selectedIndex, and "+
					"onTabChange props the native renderers consume — and node contracts live in "+
					"core, next to the registry of types the renderers know. What core's four "+
					"positional props lack is ergonomics, so the struct adds field names and "+
					"delegates everything else. One tab implementation; one facade over it."),
				codeBlock(`components.Tabs{
    Items: []core.TabItem{
        core.Tab("Info", "ℹ"),
        core.Tab("Stats", "📊"),
        core.Tab("Settings", "⚙"),
    },
    Selected: page.Get(),
    OnChange: func(i int) { page.Set(i) },
    Content:  []core.View{infoPage, statsPage, settingsPage},
}`),
				prose("Everything about it is the controlled contract again: Selected in, "+
					"OnChange out, and all pages ride along as children — the renderer draws the "+
					"strip and shows the selected one (Android as a Material tab row; iOS "+
					"hand-rolls a matching top bar, because SwiftUI's own TabView owns its "+
					"selection locally, the wrong shape for controlled Go state). One honest "+
					"caveat: the web host does not draw TabView yet, so the demo below composes "+
					"the same contract from parts it does draw — a SegmentedControl as the strip, "+
					"core.Match as the page switch. Same state, same shape; on native, swap the "+
					"pair for the Tabs above and change nothing else."),
				demoPanel("The strip writes an int; Match reads it — the tab contract, hand-assembled.",
					components.SegmentedControl{
						Style:     segWrap,
						Labels:    tabPageLabels,
						Selected:  page.Get(),
						OnSelect:  func(i int) { page.Set(i) },
						KeyPrefix: "tabdemo-",
					},
					core.Match(page.Get(),
						core.Case(0, core.Column(
							core.Gap(6),
							prose("Pages are plain views — this one is an ordinary Column riding "+
								"along as a child."),
							core.Row(
								core.Gap(8),
								core.AlignItemsProp(core.AlignItemsCenter),
								components.Badge{Text: "page 1 of 3"},
								caption("nothing here knows it lives in a tab"),
							),
						)),
						core.Case(1, core.Column(
							core.Gap(6),
							caption("64% of the gopher quota used:"),
							components.ProgressBar{Value: 0.64, AccessibilityLabel: "Gopher quota"},
						)),
						core.Default[int](core.Column(
							core.Gap(6),
							checkRow("Email me on new releases", notify),
							core.IfElse(notify.Get(),
								caption("email notifications: ON"),
								caption("email notifications: off"),
							),
							caption("Switch away and back — the box holds, because its slot lives "+
								"on this lesson's frame, above the page switch."),
						)),
					),
				),
				keyPoints(
					"Tabs adds names to core.TabView's positional props and delegates — the node type and its wire contract stay in core, next to the renderers.",
					"All pages are children of the node; the native side draws the strip and shows Selected. Selection is controlled: Selected in, OnChange out.",
					"The web host doesn't render TabView yet — compose the contract there: SegmentedControl for the strip, Match for the pages.",
					"Keep page-independent state above the switch: pages come and go with the selection; hook slots must not.",
				),
			)
		},
	}
}

// --- 4.6 -----------------------------------------------------------------

// archiveEntry is the 4.6 fixture: the shape of an archive row — a sermon,
// a transaction, a ticket — with a date to group by and two text columns to
// sort by. Newest first, as an archive feed arrives.
type archiveEntry struct {
	id      int
	title   string
	speaker string
	date    time.Time
}

var archive = []archiveEntry{
	{1, "The Narrow Gate", "M. Adeyemi", time.Date(2026, 3, 22, 0, 0, 0, 0, time.UTC)},
	{2, "Salt and Light", "R. Okafor", time.Date(2026, 3, 15, 0, 0, 0, 0, time.UTC)},
	{3, "Ask, Seek, Knock", "M. Adeyemi", time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)},
	{4, "The Two Houses", "L. Mensah", time.Date(2026, 2, 22, 0, 0, 0, 0, time.UTC)},
	{5, "Treasures in Heaven", "R. Okafor", time.Date(2026, 2, 8, 0, 0, 0, 0, time.UTC)},
	{6, "Blessed Are the Meek", "M. Adeyemi", time.Date(2026, 1, 25, 0, 0, 0, 0, time.UTC)},
	{7, "A Lamp on a Stand", "L. Mensah", time.Date(2026, 1, 18, 0, 0, 0, 0, time.UTC)},
	{8, "Wise and Foolish Builders", "R. Okafor", time.Date(2026, 1, 4, 0, 0, 0, 0, time.UTC)},
	{9, "The Golden Rule", "M. Adeyemi", time.Date(2025, 12, 28, 0, 0, 0, 0, time.UTC)},
}

// archiveKey is the reconciler key for an entry. Prefixed by kind, so a
// screen mixing this list with another keyed by integer id cannot collide.
func archiveKey(e archiveEntry) string { return fmt.Sprintf("entry:%d", e.id) }

// archiveMonth groups an entry by calendar month. Key is the sortable form,
// Label the readable one.
func archiveMonth(e archiveEntry) components.Group {
	return components.Group{Key: e.date.Format("2006-01"), Label: e.date.Format("January 2006")}
}

// loadMorePageSize is how many rows each "Load more" tap reveals in the
// grouped-list demo, standing in for one page from a server.
const loadMorePageSize = 3

func lessonCollections() Lesson {
	return Lesson{
		Title:   "Collections: GroupedList & DataTable",
		Summary: "Keyed, grouped, paged rows over core.List — the sort, the page and the group all controlled by you.",
		Body: func(ctx *core.Context) core.View {
			// The table's three controls, each a state the widget reads and
			// reports back to. A nil sort means "the caller's order".
			sortBy := core.NewState[*components.Sort](ctx, nil)
			page := core.NewState(ctx, 0)
			compact := core.NewState(ctx, false)

			// The grouped list's pager: how many rows have "arrived". A real
			// screen keeps the accumulated rows themselves; here the fixture
			// is already in memory and the count stands in for the offset.
			shown := core.NewState(ctx, loadMorePageSize)
			loaded := archive
			if shown.Get() < len(loaded) {
				loaded = loaded[:shown.Get()]
			}

			table := components.DataTable[archiveEntry]{
				Columns: []components.Column[archiveEntry]{
					{Title: "Title", Weight: 2,
						Text: func(e archiveEntry) string { return e.title },
						Less: func(a, b archiveEntry) bool { return a.title < b.title }},
					{Title: "Speaker", Weight: 1, Narrow: true,
						Text: func(e archiveEntry) string { return e.speaker },
						Less: func(a, b archiveEntry) bool { return a.speaker < b.speaker }},
					{Title: "Date", Align: core.JustifyEnd,
						Text: func(e archiveEntry) string { return e.date.Format("Jan 2") },
						Less: func(a, b archiveEntry) bool { return a.date.Before(b.date) }},
				},
				Rows:     archive,
				Key:      archiveKey,
				Sort:     sortBy.Get(),
				OnSort:   func(s components.Sort) { sortBy.Set(&s) },
				Compact:  compact.Get(),
				Dividers: true,
				Pagination: &components.Pagination{
					Page:     page.Get(),
					PageSize: 4,
					OnChange: page.Set,
					// Archive-flavored steppers. Also keeps the demo's Next
					// distinct from the lesson screen's own "Next ›" below it.
					PrevLabel: "‹ Newer",
					NextLabel: "Older ›",
				},
			}

			grouped := components.GroupedList[archiveEntry]{
				Items:   loaded,
				Key:     archiveKey,
				GroupBy: archiveMonth,
				Row: func(e archiveEntry) core.View {
					return components.ListRow{
						Title:    e.title,
						Subtitle: e.speaker,
						Trailing: caption(e.date.Format("Jan 2")),
					}
				},
				// The last month on screen is only as long as the pages
				// loaded so far, so its count stays hidden until the archive
				// is complete. Tap Load more and watch the number appear on
				// the band that was open.
				HideTrailingCount: shown.Get() < len(archive),
				Footer: components.LoadMore{
					HasMore:    shown.Get() < len(archive),
					OnLoadMore: func() { shown.Set(shown.Get() + loadMorePageSize) },
				},
			}

			return core.Column(
				core.Gap(14),
				prose("An archive screen is always the same assembly: a keyed row per item over "+
					"core.List so a thousand rows compose lazily, a header wherever the month "+
					"changes, and a pager at the tail. GroupedList and DataTable own that assembly. "+
					"They hold no state and call no hook — the sort, the page and the compact "+
					"switch are yours, read from your state and reported back through callbacks, "+
					"exactly as Chip and ListRow report a tap."),
				codeBlock(`components.DataTable[Entry]{
    Columns: []components.Column[Entry]{
        {Title: "Title", Weight: 2, Text: title, Less: byTitle},
        {Title: "Speaker", Narrow: true, Text: speaker},
        {Title: "Date", Align: core.JustifyEnd, Text: day},
    },
    Rows:    entries,
    Key:     entryKey,
    Sort:    sortBy.Get(),
    OnSort:  func(s components.Sort) { sortBy.Set(&s) },
    Compact: compact.Get(),
    Pagination: &components.Pagination{
        Page: page.Get(), PageSize: 4, OnChange: page.Set,
        PrevLabel: "‹ Newer", NextLabel: "Older ›"},
}`),
				prose("Rows are sorted, then paged, then grouped. A column with Less is sorted here, "+
					"on a copy — your slice is never reordered under you; a column marked Sortable "+
					"without Less only reports the tap, for a table whose server does the sorting. "+
					"Which one you want follows from where the rows came from: Less sorts all of "+
					"Rows and only Rows, so it is right when that is the whole set and wrong when "+
					"it is a page someone else chose — sorting a window gives you the first rows "+
					"of the window under a header claiming the first rows of the table. Debug mode "+
					"reports the detectable half of that as a partial-sort concern. "+
					"A Pagination with a PageSize and no PageCount is sliced here too, and the "+
					"footer's \"of N\" is derived from the rows. Compact drops the Narrow columns "+
					"while the sort keeps addressing your column list, so toggling it never "+
					"re-points the active sort."),
				demoPanel("Tap a header to sort, again to flip. Page with the footer. Compact hides the Speaker column.",
					core.Row(core.Gap(8), core.Padding(0),
						components.Chip{Label: "Compact", Selected: compact.Get(),
							OnTap: func() { compact.Set(!compact.Get()) }},
						core.If(sortBy.Get() != nil, components.Chip{Label: "Clear sort",
							OnTap: func() { sortBy.Set(nil) }}),
					),
					table,
				),
				prose("GroupedList is the same body without columns: your Row draws each item, "+
					"GroupBy names its group, and the Footer slot takes the pager. Grouping is "+
					"run-length over the order you hand it — a header is emitted wherever the key "+
					"changes — so a feed that arrives sorted by date gets its month headers for "+
					"free, and an offset pager that appends a page can only ever grow the last "+
					"group: nothing above the fold moves on Load more."),
				codeBlock(`components.GroupedList[Entry]{
    Items:   pager.Items,
    Key:     entryKey,
    GroupBy: func(e Entry) components.Group {
        return components.Group{
            Key:   e.Date.Format("2006-01"),
            Label: e.Date.Format("January 2006")}
    },
    Row:    func(e Entry) core.View { return entryRow(e) },
    HideTrailingCount: pager.HasMore,
    Footer: components.LoadMore{
        HasMore: pager.HasMore, Loading: pager.Loading,
        Err: pager.Err, OnLoadMore: pager.LoadMore},
}`),
				prose("The header's count counts the rows the widget was handed. Under a pager "+
					"that is \"the rows loaded so far\", and every group but the last is closed — "+
					"the next group's first row ended it — so only the last one's number is "+
					"provisional. HideTrailingCount suppresses that one badge while more pages "+
					"exist, so no count on screen is ever wrong; the header keeps its key across "+
					"the change, so the badge is patched in rather than the band replaced."),
				demoPanel("Load more reveals three rows at a time; the tail disappears when the archive is complete.",
					grouped,
				),
				keyPoints(
					"Both widgets are hook-free and fully controlled: Sort, Page and Compact live in your state; OnSort and OnChange report intent.",
					"Give a column Less only when Rows is the whole set; when the server pages, use Sortable and put the sort in the query — a client-side sort of one page is a partial sort wearing a total one's header.",
					"Sort, then page, then group — a client-side page's headers agree with its rows, and group runs follow the sort.",
					"Grouping is by run, not by bucket: sorted input yields one header per group; an append-only pager never moves an earlier header.",
					"HideTrailingCount hides the last group's badge while a pager has more to fetch: a closed group's count is final, an open one's is a number about to change.",
					"LoadMore is the four-state tail every paged screen hand-rolls: nothing, Load more, Loading…, or the error with Retry — Loading wins over Err, Err over HasMore.",
					"Key must be unique across the list and stable across renders; core.List keeps row state attached to it through reorders.",
				),
			)
		},
	}
}

// --- 4.7 -----------------------------------------------------------------

// furnitureDebounce is how long 4.7's search waits after the last keystroke
// before the query it feeds actually moves. Short enough that the demo feels
// immediate, long enough that the gap between the field and the results is
// visible — which is the whole point of showing it.
const furnitureDebounce = 250 * time.Millisecond

// archiveSpeakers is the chip strip's filter set: "" is the "All" chip, so a
// single string of state covers "no filter" and "this one" without a second
// flag.
var archiveSpeakers = []string{"", "M. Adeyemi", "R. Okafor", "L. Mensah"}

// matchesArchive is the demo's filter: a speaker chip, and a case-insensitive
// substring over the two text fields. Both empty means everything.
func matchesArchive(e archiveEntry, query, speaker string) bool {
	if speaker != "" && e.speaker != speaker {
		return false
	}
	if query == "" {
		return true
	}
	q := strings.ToLower(query)
	return strings.Contains(strings.ToLower(e.title), q) ||
		strings.Contains(strings.ToLower(e.speaker), q)
}

func lessonScreenFurniture() Lesson {
	return Lesson{
		Title:   "Screen furniture: bars, banners & placeholders",
		Summary: "AppBar, Banner, SearchField, ChipStrip, StatTile, Skeleton and EmptyState — the seven pieces every screen assembles around its content.",
		Body: func(ctx *core.Context) core.View {
			// Hooks first and unconditionally, as in every lesson: the demo
			// below branches on state, and a hook inside a branch would drift
			// the cursor.
			query := core.NewState(ctx, "")   // what the field shows, per keystroke
			applied := core.NewState(ctx, "") // what the list filters by, debounced
			speaker := core.NewState(ctx, "")
			loading := core.NewState(ctx, false)
			offline := core.NewState(ctx, false)
			backNote := core.NewState(ctx, false)
			search := hooks.UseDebounce(ctx, furnitureDebounce)

			matches := make([]archiveEntry, 0, len(archive))
			for _, e := range archive {
				if matchesArchive(e, applied.Get(), speaker.Get()) {
					matches = append(matches, e)
				}
			}

			// The three-way body switch every fetching screen has. Skeleton
			// while the shape is known and the data is not, EmptyState when
			// the answer is legitimately nothing, the rows otherwise.
			var body core.View
			switch {
			case loading.Get():
				body = components.Skeleton{Lines: 3, AccessibilityLabel: "Loading the archive"}
			case len(matches) == 0:
				body = components.EmptyState{
					Glyph: "🔎",
					Title: "Nothing matches that",
					Hint:  "Try a shorter word, or tap All.",
					// Outlined, and it clears both filters at once — an empty
					// state's action is the way out of the state that produced
					// it.
					ActionLabel: "Clear filters",
					OnAction: func() {
						query.Set("")
						applied.Set("")
						speaker.Set("")
						search.Cancel()
					},
				}
			default:
				rows := make([]core.PropsAndChildren, 0, len(matches))
				for _, e := range matches {
					rows = append(rows, components.ListRow{
						Title:    e.title,
						Subtitle: e.speaker,
						Trailing: caption(e.date.Format("Jan 2")),
					})
				}
				body = core.Column(append([]core.PropsAndChildren{
					core.Gap(0), core.Padding(0),
				}, rows...)...)
			}

			chips := make([]components.Chip, 0, len(archiveSpeakers))
			for _, s := range archiveSpeakers {
				label, value := s, s
				if s == "" {
					label = "All"
				}
				chips = append(chips, components.Chip{
					Label:    label,
					Selected: speaker.Get() == value,
					OnTap:    func() { speaker.Set(value) },
				})
			}

			return core.Column(
				core.Gap(14),
				prose("Every screen in an app is the same furniture around different content: a bar "+
					"naming where you are, a strip when something needs saying, a field to narrow "+
					"the list, a placeholder for the three moments the list has nothing to show. "+
					"Seven widgets cover it, and none of them holds state — each renders what you "+
					"pass and reports intent, exactly like Chip and DataTable."),
				codeBlock(`components.AppBar{Title: "Archive", Subtitle: "9 sermons",
    Actions: []core.View{refreshButton}}

components.Banner{Text: "Reconnecting…",
    Variant: components.VariantWarning,
    ActionLabel: "Retry now", OnAction: retry}

components.EmptyState{Glyph: "🔎", Title: "Nothing matches that",
    Hint: "Try a shorter word.", ActionLabel: "Clear filters",
    OnAction: clear}`),
				prose("AppBar draws its back arrow exactly when core.CanPop says there is a screen "+
					"underneath — so a tab root gets none without asking and a pushed screen gets "+
					"one without wiring. The arrow in the demo below is real, and it is there "+
					"because this lesson *is* a pushed screen; it is harmless only because the "+
					"demo sets OnBack, which replaces core.Pop rather than running before it. The "+
					"title takes the theme's Subtitle size with the primary ink and a bold weight: "+
					"Typography.Title is the screen's large heading — 28 points here — and does "+
					"not fit in a bar."),
				prose("Banner spends its variant on the edges, not on the fill: a hairline border "+
					"and the leading glyph take the role color while the strip keeps the theme's "+
					"Surface and the primary ink. A saturated red running the width of the screen "+
					"reads as a failure of the app rather than of one fetch, and the palette "+
					"carries no muted container tone to fill with instead. It also means the "+
					"banner's contrast does not depend on which variant it is."),
				demoPanel("Type to search — the list moves a moment after you stop. Toggle the two simulations; tap ‹ to see OnBack replace Pop.",
					components.AppBar{
						Title:    "Archive",
						Subtitle: fmt.Sprintf("%d of %d sermons", len(matches), len(archive)),
						OnBack:   func() { backNote.Set(true) },
						Actions: []core.View{components.Button{
							Label:              "↻",
							Emphasis:           components.EmphasisGhost,
							AccessibilityLabel: "Refresh",
							OnTap:              func() { loading.Set(!loading.Get()) },
						}},
					},
					core.If(backNote.Get(), caption("Back tapped — OnBack ran instead of core.Pop, so you are still here.")),
					core.If(offline.Get(), components.Banner{
						Text:        "Offline. Showing a saved copy.",
						Variant:     components.VariantWarning,
						ActionLabel: "Reconnect",
						OnAction:    func() { offline.Set(false) },
						OnDismiss:   func() { offline.Set(false) },
					}),
					components.SearchField{
						Value:       query.Get(),
						Placeholder: "Search the archive",
						OnChange: func(s string) {
							// The field is controlled, so the keystroke has to
							// land now or the characters do not appear; the
							// *reaction* is what waits.
							query.Set(s)
							search.Call(func() { applied.Set(s) })
						},
						// Enter means "now": cancel the pending call, then act.
						OnSubmit: func() {
							search.Cancel()
							applied.Set(query.Get())
						},
						OnClear: func() {
							search.Cancel()
							query.Set("")
							applied.Set("")
						},
					},
					components.ChipStrip{Chips: chips},
					core.Row(
						core.Gap(16), core.Padding(0),
						components.StatTile{
							Label: "Showing", Value: itoaLen(len(matches)), Fill: true,
							Delta:        fmt.Sprintf("of %d", len(archive)),
							DeltaVariant: components.VariantDefault,
						},
						components.StatTile{
							Label: "Filter", Value: chipValueLabel(speaker.Get()), Fill: true,
						},
					),
					body,
					core.Row(
						core.Gap(8), core.Padding(0),
						components.Chip{Label: "Simulate offline", Selected: offline.Get(),
							OnTap: func() { offline.Set(!offline.Get()) }},
						components.Chip{Label: "Simulate loading", Selected: loading.Get(),
							OnTap: func() { loading.Set(!loading.Get()) }},
					),
				),
				prose("SearchField holds no text of its own, which is what lets it live in a header "+
					"that appears and disappears — a hook-slot consumer could not. That also means "+
					"it cannot debounce its own OnChange: the value has to reach state on the "+
					"keystroke or typing looks broken. hooks.UseDebounce is the other half — Call "+
					"re-arms the delay, Cancel drops what is pending, and the demo above uses both "+
					"(Enter and the ✕ cancel, then act)."),
				codeBlock(`d := hooks.UseDebounce(ctx, 250*time.Millisecond)

components.SearchField{
    Value: query.Get(),
    OnChange: func(s string) {
        query.Set(s)                      // now: the field is controlled
        d.Call(func() { applied.Set(s) }) // in 250ms, if typing stopped
    },
    OnSubmit: func() { d.Cancel(); applied.Set(query.Get()) },
}`),
				prose("Skeleton and EmptyState answer different questions. A skeleton says content "+
					"is coming and will look roughly like this, which is worth saying when the "+
					"layout is known; an empty state says there is nothing here and why. Its bars "+
					"take the palette's Border role rather than Surface — Surface is a panel's "+
					"fill, so a Surface bar inside a card disappears, the same trap Separator "+
					"documents. They do not shimmer: a moving highlight is a repeating keyframe "+
					"animation, and core.Transition animates a property between two declared "+
					"values. Looping it from Go would push a render pass and a bridge patch per "+
					"frame of a decoration."),
				prose("StatTile is the one place in the package where the zero Variant is not the "+
					"theme's Primary: its delta defaults to the secondary ink, because whether a "+
					"number going up is good is your domain, not a widget's. Attendance up is a "+
					"success, spend up is not, latency up is an incident — so the default says "+
					"nothing and you say the rest. Fill sets FlexGrow *and* a zero FlexBasis, "+
					"which is what makes the four targets agree: Compose and SwiftUI divide the "+
					"whole axis by weight, CSS divides only the leftover space."),
				keyPoints(
					"All seven are stateless and controlled — no hooks, so any of them may be rendered conditionally.",
					"AppBar's back arrow follows core.CanPop; Leading replaces it, HideBack suppresses it, OnBack replaces core.Pop rather than running before it.",
					"Banner tints its border and glyph, never its fill: contrast stays the same whatever the variant, and the text still has to carry the meaning.",
					"EmptyState covers empty, busy and failed — one shape, so three states cannot drift apart in wording or spacing. Its Width 100% is load-bearing: a column hugs its widest child on both natives.",
					"A controlled field cannot debounce its own OnChange; debounce the reaction with hooks.UseDebounce, and Cancel before acting on Enter.",
					"Every glyph in these widgets is decoration and is hidden from assistive tech — the text beside it is what gets announced.",
					"ChipStrip wraps by default and pans with Scrollable — wrapping for a set the reader should see all of, panning for a filter bar that would otherwise push the content off screen (4.8).",
				),
			)
		},
	}
}

// itoaLen is the row count as a string. It exists so the demo's StatTile takes
// a pre-formatted Value, which is the contract: the widget does no number
// formatting, because grouping and locale are the caller's.
func itoaLen(n int) string { return strconv.Itoa(n) }

// chipValueLabel renders the speaker filter for the StatTile: the empty
// sentinel is the "All" chip, and it has to read the same in both places.
func chipValueLabel(speaker string) string {
	if speaker == "" {
		return "All"
	}
	return speaker
}

// --- 4.8 -----------------------------------------------------------------

// endlessPageSize is how many rows each end-reached report reveals in 4.8,
// standing in for one page from a server. Smaller than loadMorePageSize so
// the demo's short fixture takes several scrolls to exhaust.
const endlessPageSize = 2

// endlessViewport is the height the demo's list is boxed into. A sticky band
// pins to its nearest scrolling ancestor, and the tutorial page itself is
// that ancestor unless something closer says otherwise — which would pin the
// band to the top of the browser window, halfway up an unrelated lesson. On a
// phone the List owns the screen and needs no such box.
const endlessViewport = "260px"

func lessonEndlessFeeds() Lesson {
	return Lesson{
		Title:   "Endless feeds: sticky bands & sideways strips",
		Summary: "core.Horizontal, core.OnEndReached and core.StickyHeader — the three props that turn a paged list into a feed, on all four targets.",
		Body: func(ctx *core.Context) core.View {
			// How many rows have "arrived". A real screen keeps the
			// accumulated rows; the fixture is fixed, so a count is enough.
			loaded := core.NewState(ctx, endlessPageSize)
			// How many times the edge has been reported. Shown in the demo
			// because the debounce is the interesting part and it is
			// invisible otherwise: without it this number would run away.
			fetches := core.NewState(ctx, 0)
			speaker := core.NewState(ctx, "")

			rows := make([]archiveEntry, 0, loaded.Get())
			for _, e := range archive {
				if speaker.Get() != "" && e.speaker != speaker.Get() {
					continue
				}
				rows = append(rows, e)
			}
			// total is the filtered set's size — what "of N" should say once a
			// speaker chip has narrowed the feed. len(archive) would be the
			// unfiltered count and would read as rows that never arrive.
			total := len(rows)
			hasMore := loaded.Get() < total
			if hasMore {
				rows = rows[:loaded.Get()]
			}

			loadNext := func() {
				fetches.Set(fetches.Get() + 1)
				loaded.Set(loaded.Get() + endlessPageSize)
			}

			chips := make([]components.Chip, 0, len(archiveSpeakers))
			for _, s := range archiveSpeakers {
				label, value := s, s
				if s == "" {
					label = "Everyone"
				}
				chips = append(chips, components.Chip{
					Label:    label,
					Selected: speaker.Get() == value,
					OnTap: func() {
						speaker.Set(value)
						loaded.Set(endlessPageSize)
					},
				})
			}

			return core.Column(
				core.Gap(14),
				prose("A paged list becomes a feed with three props, and none of them is a new "+
					"widget: a strip that pans sideways instead of wrapping, a band that stays put "+
					"while its rows scroll under it, and an edge that fires when the bottom is "+
					"near. Each is one field on something you already have, and each works on all "+
					"four targets."),
				codeBlock(`core.Scroll(core.Horizontal(), chips...)   // the sideways strip

core.List(
    core.OnEndReached(pager.LoadNext),                 // near the bottom
    core.Keyed("group:2026-03",
        core.Row(core.StickyHeader(), monthLabel)),    // the pinned band
    rows...,
)`),
				prose("core.Horizontal spells itself entirely in style — flex-direction row plus an "+
					"overflow — which is why both web targets already drew it the day it was "+
					"written and only the two natives needed code. Compose reads the axis and "+
					"builds a Row on a horizontal scroll state; SwiftUI builds a "+
					"ScrollView(.horizontal). core.StickyHeader is the same trick: it is "+
					"Position: sticky, a value core.Style has always carried and the browser has "+
					"always honoured, and the natives now answer it with the one thing that means "+
					"the same — a pinned header inside their lazy list."),
				prose("core.OnEndReached is the one with real work behind it. Every platform "+
					"reports the bottom more than once for the same bottom — an observer re-fires "+
					"on resize, .onAppear re-fires when a row is recycled, a snapshot flow emits "+
					"per visible-index change — so the prop remembers how many rows the list held "+
					"when it last ran and refuses to run again until that number changes. One "+
					"debounce in Go, rather than four different notions of \"again\" in four "+
					"renderers."),
				demoPanel("Scroll the box: the month band pins, and the next two rows arrive before you reach the end. The strip pans sideways.",
					components.ChipStrip{Scrollable: true, Chips: chips},
					core.Box(
						core.Height(endlessViewport),
						core.Overflow("auto"),
						core.BorderRadius(8),
						components.GroupedList[archiveEntry]{
							Items:             rows,
							Key:               archiveKey,
							GroupBy:           archiveMonth,
							StickyHeaders:     true,
							HideTrailingCount: hasMore,
							Row: func(e archiveEntry) core.View {
								return components.ListRow{
									Title:    e.title,
									Subtitle: e.speaker,
									Trailing: caption(e.date.Format("Jan 2")),
								}
							},
							OnEndReached: loadNext,
							// The footer stays. Auto-loading replaces the tap,
							// not the tail: this is still where "Loading…" and
							// a failed page's Retry live, and it is the manual
							// fallback wherever the edge cannot be reported.
							Footer: components.LoadMore{
								HasMore:    hasMore,
								OnLoadMore: loadNext,
							},
						},
					),
					caption(fmt.Sprintf("%d of %d rows, %d fetches", len(rows), total, fetches.Get())),
					core.Row(core.Gap(8), core.Padding(0),
						components.Chip{Label: "Start over", OnTap: func() {
							loaded.Set(endlessPageSize)
							fetches.Set(0)
						}},
					),
				),
				prose("Keep the footer. A screen that drops its LoadMore for OnEndReached gains a "+
					"feed that stops silently at whatever page failed, and loses the one control "+
					"that still works on a static export or in a browser with no "+
					"IntersectionObserver. Handing the same load function to both is the intended "+
					"shape: the debounce means a tap and a scroll cannot double-load, because "+
					"neither can fire while the row count is unchanged."),
				codeBlock(`components.GroupedList[Entry]{
    Items:         pager.Items,
    StickyHeaders: true,
    OnEndReached:  pager.LoadNext,   // the scroll
    Footer: components.LoadMore{     // and the tap, and the states
        HasMore: pager.HasMore, Loading: pager.Loading,
        Err: pager.Err, OnLoadMore: pager.LoadNext},
}`),
				prose("The demo boxes its list in a fixed height because a sticky band pins to its "+
					"nearest scrolling ancestor, and on this page that would otherwise be the "+
					"browser window — the band would stick to the top of the screen, above "+
					"lessons it has nothing to do with. A real screen hands the List the screen "+
					"and the question does not arise."),
				keyPoints(
					"core.Horizontal() is a StyleProp: flex-direction plus overflow, so the web targets needed no code and the natives read the axis in their Scroll composite alone.",
					"core.StickyHeader() reuses Position: sticky rather than inventing a marker — and supplies Top and ZIndex, without which a sticky box silently never sticks or is painted over.",
					"core.OnEndReached fires at most once per row count, so a slow fetch cannot double-load and an exhausted feed goes quiet instead of re-asking forever.",
					"An empty list never reports the edge: the first page is yours to ask for, the next one is the scroll's.",
					"Keep the LoadMore footer — it carries the loading and error states, and it is the fallback where no edge can be reported.",
					"ChipStrip.Scrollable is one line that pans; the default still wraps, which is right for a set the reader should see all of.",
					"Sticky bands only pin inside a List: both natives implement pinning in their lazy container, so the same marker on a Column child is web-only.",
				),
			)
		},
	}
}

// --- 4.9 -----------------------------------------------------------------

// tutorialToday is the demo's "today". Pinned rather than read from the clock
// for the reason components.Calendar makes Today a field in the first place:
// a page whose picture changes at midnight cannot be snapshot-tested, and a
// lesson about not consulting the clock should not consult the clock.
var tutorialToday = time.Date(2026, time.March, 11, 12, 0, 0, 0, time.UTC)

// The bounds of the demo's calendar: the span the 4.6 archive fixture covers.
// They make the arrows disable at the ends, which is the visible half of
// Min/Max.
var (
	tutorialCalMin = time.Date(2025, time.December, 1, 12, 0, 0, 0, time.UTC)
	tutorialCalMax = time.Date(2026, time.March, 31, 12, 0, 0, 0, time.UTC)
)

// archiveOn finds the entry falling on a calendar day, which is what the
// calendar's Marked and the caption under it both ask. Compared by the Y/M/D
// triple rather than by instant: the widget hands out midday and the fixture
// is stamped at midnight, and they are the same day.
func archiveOn(d time.Time) (archiveEntry, bool) {
	for _, e := range archive {
		ey, em, ed := e.date.Date()
		dy, dm, dd := d.Date()
		if ey == dy && em == dm && ed == dd {
			return e, true
		}
	}
	return archiveEntry{}, false
}

func lessonCalendars() Lesson {
	return Lesson{
		Title:   "Calendars: a month grid and a date field",
		Summary: "components.Calendar and components.DatePicker — a controlled month, a today you supply, and cells built at midday for a reason.",
		Body: func(ctx *core.Context) core.View {
			// Which month is on screen, and which day is chosen. Both belong
			// to the caller: this demo drives one calendar and one date field
			// from the same selection, which it could not do if the widget
			// owned either.
			month := core.NewState(ctx, tutorialToday)
			picked := core.NewState(ctx, time.Time{})

			marked := func(d time.Time) bool {
				_, ok := archiveOn(d)
				return ok
			}

			// What the chosen day has on it — the reason a calendar is worth
			// more than a text field.
			note := "Tap a dotted day: those are the ones with a sermon."
			if p := picked.Get(); !p.IsZero() {
				if e, ok := archiveOn(p); ok {
					note = fmt.Sprintf("%s — %s, %s", e.title, e.speaker, p.Format("Monday 2 January 2006"))
				} else {
					note = fmt.Sprintf("Nothing on %s.", p.Format("Monday 2 January 2006"))
				}
			}

			return core.Column(
				core.Gap(14),
				prose("A calendar is a grid of buttons over a little date arithmetic, and both are the "+
					"kind of thing every app rewrites slightly differently. components.Calendar draws "+
					"the month; you own what it shows. The month on screen, the selected day and the "+
					"day that counts as today are three separate fields, because a screen that opens "+
					"on the month of its next event needs to say so."),
				codeBlock(`month  := core.NewState(ctx, someDate)
picked := core.NewState(ctx, time.Time{})

components.Calendar{
    Month:         month.Get(),   OnMonthChange: month.Set,
    Selected:      picked.Get(),  OnSelect:      picked.Set,
    Today:         today,                       // a field, not a clock read
    Min:           season.Start, Max: season.End,
    Marked:        func(d time.Time) bool { return hasEvent(d) },
}`),
				demoPanel("The dots are the 4.6 archive. The ring is \"today\"; the fill is your selection; the arrows die at the ends of the range.",
					components.Calendar{
						Month:         month.Get(),
						OnMonthChange: month.Set,
						Selected:      picked.Get(),
						OnSelect:      picked.Set,
						Today:         tutorialToday,
						Min:           tutorialCalMin,
						Max:           tutorialCalMax,
						Marked:        marked,
					},
					caption(note),
					components.FormField{
						Label: "Same selection, as a field",
						Hint:  "DatePicker is the grid behind a summary, with the two view states it needs of its own.",
						Input: components.DatePicker{
							Selected:    picked.Get(),
							OnSelect:    picked.Set,
							OnClear:     func() { picked.Set(time.Time{}) },
							Placeholder: "Choose a date",
							Title:       "Service date",
							Calendar: components.Calendar{
								Today:  tutorialToday,
								Min:    tutorialCalMin,
								Max:    tutorialCalMax,
								Marked: marked,
							},
						},
					},
				),
				prose("Today is a field and not a time.Now(). A render that reads the clock is not a "+
					"function of its inputs, so the same grid would snapshot differently after "+
					"midnight; and \"today\" is a question about a time zone that the widget cannot "+
					"answer and the caller can. A zero Today draws no ring, which is the honest "+
					"picture of a calendar nobody has told what day it is."),
				prose("The grid is always six rows, padded at both ends with the neighbouring months' "+
					"days. A grid that sized itself to its month would change height between "+
					"February and August and shove everything below it up and down on every arrow "+
					"tap. It also means changing month patches 42 numbers and no structure. Those "+
					"padding days are dimmed and inert — a controlled calendar cannot move its own "+
					"month, so a tap on them would either select a day the grid no longer highlights "+
					"or fire two callbacks in an order the caller has to guess."),
				prose("Every cell is built at midday, and the value OnSelect hands back is midday too. "+
					"That is not fussiness: midnight does not exist on every calendar day. Chile "+
					"springs forward at 24:00, so 2026-09-06 in Santiago starts at 01:00 and Go "+
					"resolves a request for its midnight to 2026-09-05 23:00 — the day before. A "+
					"grid built at midnight emits two cells that both read as the 5th, and the 6th "+
					"can never be picked, in exactly the zones nobody testing in UTC will ever look "+
					"at. Midday is skipped by no transition in the tz database."),
				codeBlock(`components.FormField{
    Label: "Event date",
    Input: components.DatePicker{
        Selected: date.Get(),
        OnSelect: date.Set,
        OnClear:  func() { date.Set(time.Time{}) },
        Calendar: components.Calendar{Today: today, Min: today},  // the template
    },
}`),
				prose("DatePicker is the packaging, not a second calendar. It owns the two states no "+
					"application ever wants — is the sheet open, which month is being browsed — and "+
					"is therefore the second widget in the package with hook obligations, after "+
					"Accordion: render it unconditionally, every pass. Everything else it hands "+
					"through to a Calendar template, the way SegmentedControl hands its Segment "+
					"through to a Chip. There is no label or error line on it, because FormField "+
					"already has both and any input drops into its slot."),
				keyPoints(
					"Month, Selected and Today are three separate fields: the month on screen is view state a screen often wants to drive itself.",
					"The widget never calls time.Now(); Today is a field, so the grid is a pure function of its inputs and a zero Today simply draws no ring.",
					"Six rows always — a grid that resized itself would move everything under it on every arrow tap, and a fixed shape makes a month change a pure prop patch.",
					"Cells are built at midday, because midnight is a local time that does not exist on every day in every zone.",
					"Min and Max are compared by calendar day, so a Max stamped at 15:04 still includes its own day; an arrow whose whole target month is out of range is disabled.",
					"Marked is called 42 times a render, adjacent months included — make it a lookup, not a query.",
					"MonthLabel, WeekdayLabel and DayLabel are the localization seams: Go's time package speaks English only.",
					"DatePicker owns two view states and takes a Calendar as a template; wrap it in FormField for the label, hint and error.",
				),
			)
		},
	}
}
