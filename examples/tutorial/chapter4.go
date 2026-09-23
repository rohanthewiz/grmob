package tutorial

import (
	"cmp"
	"fmt"
	"maps"
	"math"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/rohanthewiz/grmob/alarm"
	"github.com/rohanthewiz/grmob/comps"
	"github.com/rohanthewiz/grmob/core"
	"github.com/rohanthewiz/grmob/hooks"
	"github.com/rohanthewiz/grmob/permission"
	"github.com/rohanthewiz/grmob/richtext"
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
			lessonCompass(),
			lessonStaticMap(),
			lessonLiveMap(),
			lessonCodeEditor(),
			lessonRichText(),
			lessonSmallControls(),
			lessonChoicesAndProgress(),
			lessonMenus(),
			lessonDrawers(),
			lessonClocksAndDrawing(),
			lessonCharts(),
			lessonFoldables(),
			lessonFAB(),
			lessonQRCode(),
			lessonTimers(),
			lessonDateRange(),
			lessonTimePicker(),
			lessonSmallPieces(),
			lessonHeatAndSpread(),
			lessonCopyLinkAndList(),
			lessonAudioPlayer(),
			lessonMessageBubbles(),
			lessonReadMoreAndHalfStars(),
			lessonMessageThread(),
			lessonChatFamily(),
			lessonTreeAndWizard(),
			lessonFourMoreCharts(),
			lessonEditableGrid(),
		},
	}
}

// --- 4.1 -----------------------------------------------------------------

// variantNames double as the 4.1 segment captions and the constant-name
// suffixes ("Success" → comps.VariantSuccess): the demo's printed struct
// literal is built by pasting them onto the type prefix, so captions and code
// cannot disagree. variantValues is the parallel value table; 4.2 reuses it
// with its own captions, which is the point of Variant being shared — one
// vocabulary across the whole package.
var (
	variantNames  = []string{"Default", "Success", "Warning", "Error"}
	variantValues = []comps.Variant{
		comps.VariantDefault,
		comps.VariantSuccess,
		comps.VariantWarning,
		comps.VariantError,
	}
	emphasisNames  = []string{"Filled", "Outlined", "Ghost"}
	emphasisValues = []comps.Emphasis{
		comps.EmphasisFilled,
		comps.EmphasisOutlined,
		comps.EmphasisGhost,
	}
)

// buttonSnippet prints the Button literal the demo's current knobs would
// build, omitting every zero-value field. The omission is the lesson: what is
// not set contributes nothing, so the printed struct is the whole truth about
// the button on screen.
func buttonSnippet(variant, emphasis int, disabled, fullWidth bool) string {
	lines := []string{
		"comps.Button{",
		`    Label: "Save changes",`,
		"    OnTap: save,",
	}
	if variant != 0 {
		lines = append(lines, "    Variant: comps.Variant"+variantNames[variant]+",")
	}
	if emphasis != 0 {
		lines = append(lines, "    Emphasis: comps.Emphasis"+emphasisNames[emphasis]+",")
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
				prose("Every button you have tapped in this tutorial was a comps.Button. "+
					"The widget library is structs with named fields, each implementing core.View, "+
					"so widgets compose exactly like the primitives do — and a field can be added "+
					"without breaking a single call site:"),
				codeBlock(`comps.Button{Label: "Save", OnTap: save}   // the theme's Button, untouched

comps.Button{Label: "Delete", OnTap: rm,
    Variant: comps.VariantError}            // meaning: a palette status role

comps.Button{Label: "Cancel", OnTap: back,
    Emphasis: comps.EmphasisOutlined}       // weight: how much color it spends`),
				prose("Color is two orthogonal axes, not one enum. Variant picks which role the "+
					"button spends — Success, Warning, Error, or Default for the theme's Primary. "+
					"Emphasis picks how much of it: Filled, Outlined, or Ghost. A flat enum would "+
					"need a value per combination the moment a design wants an outlined destructive "+
					"button. And both zero values contribute no style props at all, so the plain "+
					"Button{Label, OnTap} renders exactly core.Button — whatever the theme chose "+
					"for its Button base shows through untouched."),
				demoPanel("Every knob re-renders one struct — and prints the literal that would build it.",
					caption("Variant — what the action means:"),
					comps.SegmentedControl{
						Style:     segWrap,
						Labels:    variantNames,
						Selected:  variant.Get(),
						OnSelect:  func(i int) { variant.Set(i) },
						KeyPrefix: "btn-variant-",
					},
					caption("Emphasis — how loudly it says it:"),
					comps.SegmentedControl{
						Style:     segWrap,
						Labels:    emphasisNames,
						Selected:  emphasis.Get(),
						OnSelect:  func(i int) { emphasis.Set(i) },
						KeyPrefix: "btn-emphasis-",
					},
					checkRow("Disabled", disabled),
					checkRow("Full width", fullWidth),
					comps.Button{
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

// paneLabels caption the tab-strip arrangement of the same widget. Three, and
// short, because the point of that demo is the announcement rather than the
// content: the strip is the one thing on the panel that is a tablist, and a
// reader stepping through it should hear "tab 2 of 3" without the row wrapping
// onto a second line first.
var paneLabels = []string{"Sermons", "Articles", "Notes"}

func lessonPills() Lesson {
	return Lesson{
		Title:   "Badges, chips & segments",
		Summary: "One pill family: Badge states, Chip selects, SegmentedControl is chips-plus-loop, extracted.",
		Body: func(ctx *core.Context) core.View {
			status := core.NewState(ctx, 0)
			picked := core.NewState(ctx, map[string]bool{})
			// The tab-strip arrangement's own selection. A third state rather
			// than reusing `status`, because the two demos below answer
			// different questions and sharing one index would make the tab
			// strip move when you picked a badge variant.
			pane := core.NewState(ctx, 0)

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
				chips = append(chips, core.Keyed("topic-"+topic, comps.Chip{
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
				codeBlock(`comps.Badge{Text: "Live", Variant: comps.VariantSuccess}

comps.Chip{
    Label:    topic,
    Selected: picked[topic], // caller state — the chip holds none
    OnTap:    toggle(topic),
}

comps.SegmentedControl{
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
				prose("What a reader hears is a third contract, and it is the one you get for free. "+
					"Every Chip states core.AccessibilitySelected — on the unselected ones too, "+
					"which is the half that matters: a row where only the chosen pill answers is "+
					"announced as one toggle among three pieces of furniture. That is why the type "+
					"has three values and not a bool."),
				prose("The state does not say which ARIA attribute it becomes; the role does. A "+
					"chip with no role is a <button>, so it is aria-pressed — a toggle answering "+
					"only for itself. Say the row is a tablist and the pills are tabs, and the same "+
					"field comes out as aria-selected — one of a set, where choosing one unchooses "+
					"the rest. Two props, no new field, and neither widget knows which arrangement "+
					"it is in:"),
				codeBlock(`comps.SegmentedControl{
    Labels:   []string{"Sermons", "Articles"},
    Selected: tab.Get(),
    OnSelect: func(i int) { tab.Set(i) },
    // Without these two: a group of toggle buttons (aria-pressed).
    // With them: a tab strip (aria-selected).
    Style:   []core.StyleProp{core.AccessibilityRole(core.RoleTabList)},
    Segment: comps.Chip{Style: []core.StyleProp{core.AccessibilityRole(core.RoleTab)}},
}`),
				prose("Two things that does not buy, and both are ARIA's rules rather than the "+
					"widget's. A tablist claims its children are tabs, so a row that also holds a "+
					"count or an add button is not one. And the tabs cannot point at the panel they "+
					"control: aria-controls is an ID reference and a Style carries values, not "+
					"references. A strip that is really wired to panels is core.TabView (4.5), "+
					"which owns both ends of that relationship and writes the whole wiring itself."),
				demoPanel("A badge fed by a segmented control, and a chip group over one map.",
					caption("Pick a status — the Badge takes its variant from the same index:"),
					comps.SegmentedControl{
						Style:     segWrap,
						Labels:    pillStatusLabels,
						Selected:  status.Get(),
						OnSelect:  func(i int) { status.Set(i) },
						KeyPrefix: "pill-status-",
					},
					core.Row(
						core.Gap(8),
						core.AlignItemsProp(core.AlignItemsCenter),
						comps.Badge{
							Text:    pillStatusLabels[status.Get()],
							Variant: variantValues[status.Get()],
						},
						caption(fmt.Sprintf("← Badge{Text: %q, Variant: comps.Variant%s}",
							pillStatusLabels[status.Get()], variantNames[status.Get()])),
					),
					comps.Separator{},
					caption("Chips — a multi-select over one map in this lesson's state:"),
					core.Row(chips...),
					core.IfElse(len(chosen) == 0,
						caption("nothing picked — every chip renders Selected straight from the map"),
						caption("picked: "+strings.Join(chosen, " · ")),
					),
					comps.Separator{},
					caption("The same widget as a tab strip — two roles, and the state each chip "+
						"already sets goes out as aria-selected instead of aria-pressed:"),
					comps.SegmentedControl{
						Style: []core.StyleProp{
							core.Gap(8),
							core.FlexWrap(true),
							core.AccessibilityRole(core.RoleTabList),
						},
						Labels:    paneLabels,
						Selected:  pane.Get(),
						OnSelect:  func(i int) { pane.Set(i) },
						KeyPrefix: "pill-pane-",
						Segment: comps.Chip{
							Style: []core.StyleProp{core.AccessibilityRole(core.RoleTab)},
						},
					},
					// A panel of sorts, and deliberately an unwired one: the
					// tabs above cannot point at it, because aria-controls is
					// an ID reference. What makes this honest is that it is
					// captioned as a demonstration rather than dressed up as a
					// real tab view — 4.5 has the wired one.
					caption("showing: "+paneLabels[pane.Get()]),
				),
				keyPoints(
					"Badge is a statement, Chip a controlled toggle, SegmentedControl a controlled index — none of them holds state.",
					"Color reinforces, text carries: the label must say \"Failed\" by itself; no renderer announces a tint (WCAG 1.4.1).",
					"Update the chips' map immutably — copy, flip, Set — the chapter-2 slice rule, applied to a map.",
					"Key looped chips, and give twin segmented controls a KeyPrefix, so the reconciler matches pills by identity, not position.",
					"Every chip states core.AccessibilitySelected, unselected ones included — SelectedOff is a value with a job, not a way of saying nothing.",
					"The role picks the attribute: a plain chip is aria-pressed, a chip carrying RoleTab inside a RoleTabList row is aria-selected.",
					"And the pair buys the keyboard on the web: a tablist is one tab stop, and the arrow keys move within it.",
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

// outlineNode is one line of 4.3's flattened-tree demo: a title and how deep
// it sits. A flat slice, not a tree of children, because that is the shape the
// demo is about — a list is a flat run of siblings, so the depth has to travel
// as data rather than as structure.
type outlineNode struct {
	title string
	depth int
}

var canonOutline = []outlineNode{
	{"New Testament", 1},
	{"Gospels", 2},
	{"Matthew", 3},
	{"Mark", 3},
	{"Letters", 2},
	{"Romans", 3},
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
					trailing = comps.Badge{Text: m.badge, Variant: comps.VariantWarning}
				}
				rows = append(rows, core.Keyed(m.name, comps.ListRow{
					Leading:  comps.Avatar{Name: m.name, Size: 36},
					Title:    m.name,
					Subtitle: m.role,
					Trailing: trailing,
					Selected: selected.Get() == m.name,
					// The row is one choice in a listbox — see the RoleListBox
					// on the Column below, which is the half a row cannot
					// supply for itself.
					Selectable: true,
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
				codeBlock(`comps.ListRow{
    Leading:  comps.Avatar{Name: m.Name, Size: 36},
    Title:    m.Name,
    Subtitle: m.Role,
    Trailing: comps.Badge{Text: "on call",
        Variant: comps.VariantWarning},
    Selected: selected.Get() == m.Name,
    OnTap:    func() { selected.Set(m.Name) },
}`),
				prose("The middle column is the row's spine: it renders even when empty, always "+
					"with FlexGrow(1), so it soaks up the slack and pins Trailing hard against the "+
					"edge in every configuration — a promise JustifyBetween cannot make once a slot "+
					"goes missing. The Avatar in the Leading slot earns its cameo: give it a Name "+
					"and it derives initials (first word, last word) and announces the name; give "+
					"it neither and it hides from assistive tech rather than announce \"image\"."),
				prose("A list where tapping a row changes which one is chosen is a listbox, "+
					"and saying so is what lets a row announce its selection at all. Put "+
					"core.RoleListBox on the container and Selectable on each row: the row "+
					"becomes an option carrying a real selected state, both values of it, so a "+
					"reader says \"selected\" on the chosen row and \"not selected\" on the rest. "+
					"Without the pair the state has nowhere to live — ARIA scopes it to a "+
					"handful of roles and a plain row is none of them — so the widget falls "+
					"back to appending \", selected\" to the row's name, which announces once "+
					"and makes a name that is supposed to be stable move. The container's "+
					"role also buys the keyboard on the web: the list becomes one tab stop, "+
					"the arrow keys move between rows, and Enter or Space runs the focused "+
					"row's OnTap."),
				codeBlock(`core.Column(
    core.AccessibilityRole(core.RoleListBox),   // the container's half
    comps.ListRow{Title: m.Name, Selectable: true,
        Selected: selected.Get() == m.Name},    // the row's half
)`),
				demoPanel("Tap a row to select it; tap it again to clear. Rows are Keyed by name.",
					// The listbox half of the pair. Sayable here because this
					// Column holds nothing but rows: a heading or a footer
					// inside it would be a foreign child and cost the role.
					core.Column(append([]core.PropsAndChildren{
						core.AccessibilityRole(core.RoleListBox),
					}, rows...)...),
					core.IfElse(selected.Get() == "",
						caption("No row selected — tap one."),
						caption("Selected: "+selected.Get()),
					),
				),
				prose("A row cannot be both, and the reason is ARIA's rather than the "+
					"widget's: an option carries the selected state and no depth, a listitem "+
					"carries the depth and no selected state. Set both and Selectable wins — "+
					"the state is what the tap changes, and a depth inside a container that "+
					"never claimed to be a list is decoration."),
				prose("A row can also say how deep it sits. NestingLevel makes it a listitem at "+
					"that depth, which is the one thing an indented outline cannot say any other "+
					"way: a list is a flat run of siblings — that is what makes it virtualizable "+
					"— so the nesting lives in the data and the indent is pixels a screen reader "+
					"never sees. The field is opt-in because it needs a partner: a listitem is "+
					"owned by a list, and a row cannot see its own container, so you put RoleList "+
					"on the list and the depth on each row. Ask for neither and a row is the "+
					"unroled box it has always been."),
				codeBlock(`core.List(
    core.AccessibilityRole(core.RoleList),
    comps.ListRow{Title: "Gospels", NestingLevel: 2},
    comps.ListRow{Title: "Matthew", NestingLevel: 3,
        Style: []core.StyleProp{indentBy(3)}},   // pixels; the level is the announcement
)`),
				demoPanel("Six rows, one flat list, three depths. The indent is decoration; NestingLevel is what a reader hears.",
					outlineDemo(),
				),
				keyPoints(
					"Slots are core.View fields: Leading and Trailing take any view; Content replaces Title/Subtitle when set.",
					"The middle column always renders, always FlexGrow(1) — that spine is what pins Trailing to the edge.",
					"OnTap nil registers nothing: a presentational row carries no callback and no gesture recognizer on any platform.",
					"ListRow synthesizes no accessibility name — its slots carry meaning it can't see, so you name the row; Avatar does synthesize one, and hides itself when nameless.",
					"NestingLevel makes a row a listitem at a depth — the only way a flattened outline can be more than an indent; put RoleList on the list yourself.",
				),
			)
		},
	}
}

// outlineDemo is the flattened tree: one core.List carrying the list role, six
// ListRows carrying their depths, indented by the same number that is
// announced.
//
// The indent is derived from the depth rather than stored beside it, which is
// the demo's whole argument in one line: the pixels and the announcement are
// two renderings of one fact, and a reader who gets only the pixels gets
// nothing.
func outlineDemo() core.View {
	return core.ComponentFunc(func(ctx *core.Context) *core.Node {
		items := []core.PropsAndChildren{
			core.Gap(2),
			core.Padding(0),
			// The other half of the pair. Without it every row below is an
			// orphan listitem — a role naming a structure that is not there,
			// which core/role.go calls worse than no role at all.
			core.AccessibilityRole(core.RoleList),
		}
		for _, n := range canonOutline {
			items = append(items, core.Keyed(n.title, comps.ListRow{
				Title:        n.title,
				NestingLevel: n.depth,
				Style:        []core.StyleProp{indentBy(n.depth)},
			}))
		}
		return core.List(items...).Render(ctx)
	})
}

// indentBy is the pixels half of a depth: one step of left inset per level.
//
// One prop for the one side that varies. This used to be a whole EdgeInsets
// through UseStyle, because core had no left-side prop and UseStyle replaces
// Padding outright rather than merging edge by edge — so the helper had to
// restate the three sides it did not care about, in numbers copied out of the
// theme that a theme edit would never reach. core.PaddingLeft is what that
// workaround was waiting for.
//
// A StyleProp and not a Style, so it composes with whatever the row already
// carries: the theme Row's own vertical padding stays, and a caller can put
// core.PaddingRight after this one without either undoing the other.
func indentBy(depth int) core.StyleProp {
	return core.PaddingLeft(16 * depth)
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
				faq = append(faq, comps.Accordion{
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
				codeBlock(`comps.Accordion{
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
				prose("An accordion's Title is also a heading, at level 3, and the tier is not "+
					"decoration: a reader navigating by heading needs to know that a question "+
					"sits inside a section rather than beside it. The package fills the outline "+
					"in — an AppBar title is 1, a Card title or a GroupedList band is 2, an "+
					"accordion question is 3 — and each of the lower three takes a HeadingLevel "+
					"field for when you put it somewhere else, because only the bar's position "+
					"is fixed by construction. This very screen is the three tiers at once: the "+
					"lesson's name above, this accordion's questions below, and Key points "+
					"between them at 2."),
				codeBlock(`comps.Accordion{Title: "Shipping", HeadingLevel: 4} // inside a card in a section
comps.GroupedList[Sermon]{GroupBy: byMonth, HeadingLevel: 1} // a feed with no bar`),
				prose("The heading is the only one in the package that does not ride the words, "+
					"and the reason is the other half of what a disclosure has to announce. A "+
					"header that says what it is called and not whether it is open is a control "+
					"a reader has no reason to press — so the row states core.RoleButton and "+
					"core.AccessibilityExpanded, and ARIA defines that state for a button and "+
					"not for the group role a named row would otherwise be given. A button's "+
					"children are presentational, though, so the tier cannot stay on the title "+
					"inside it. It moves to a Box wrapped around the row, named explicitly with "+
					"the Title — which is what stops the heading from being called \"▸ Shipping\", "+
					"since a heading with no name of its own takes one from its content. That "+
					"nesting is ARIA's own accordion pattern, and a reader hears the question "+
					"twice: once as an outline entry to jump to, once as a control that says "+
					"collapsed or expanded. On the web that is a heading div carrying an "+
					"aria-level and an aria-label, wrapped around a button div carrying an "+
					"aria-expanded, and neither element holds the other's attributes."),
				prose("core.ExpandedWhen is what states both halves. Setting the open case and "+
					"leaving the shut one alone is the mistake the three-valued type exists to "+
					"prevent — a collapsed section that answers nothing is announced as an "+
					"ordinary button, and \"collapsed\" is the whole of what invites the press. "+
					"On a phone the two targets diverge further than usual: Compose has "+
					"expand/collapse actions and offers TalkBack the one the state calls for, "+
					"while SwiftUI has no expanded trait at all and says nothing."),
				demoPanel("Three accordions, three bool slots on this lesson's context, claimed in render order.",
					faq...,
				),
				keyPoints(
					"Accordion calls NewState on your context — render it unconditionally, in a stable position, every pass.",
					"Title is a level-3 heading; HeadingLevel moves it, and Header opts out — a view you built is yours to describe.",
					"The header row is a button carrying core.AccessibilityExpanded, with the heading wrapped around it — ARIA's accordion shape, because aria-expanded is not defined for a group.",
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

// tabDemoPanelID is the id of the region 4.5's hand-assembled strip switches,
// and the string every segment's core.AccessibilityControls points at. One
// constant referenced twice, because a typo in an IDREF is not an error
// anywhere — it is a tab announcing a region that does not exist.
const tabDemoPanelID = "tutorial-tabdemo-panel"

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
				prose("comps.Tabs is a different kind of widget: a facade. core.TabView "+
					"defines a wire contract — a \"TabView\" node whose tabs, selectedIndex, and "+
					"onTabChange props the native renderers consume — and node contracts live in "+
					"core, next to the registry of types the renderers know. What core's four "+
					"positional props lack is ergonomics, so the struct adds field names and "+
					"delegates everything else. One tab implementation; one facade over it."),
				codeBlock(`comps.Tabs{
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
				prose("A hand-assembled strip has one thing the node type gets for free, and it "+
					"is worth wiring by hand once. core.TabView mints element ids and writes the "+
					"whole relationship — which region each tab shows — from the node type. Built "+
					"out of parts, that relationship has to be stated: core.AccessibilityID names "+
					"the region, core.AccessibilityControls points every segment at it, and the "+
					"row and the segments take RoleTabList and RoleTab. Without it a screen "+
					"reader announces three tabs governing nothing."),
				codeBlock(`comps.SegmentedControl{
    Labels: tabPageLabels, Selected: page.Get(), OnSelect: page.Set,
    Style:   []core.StyleProp{core.AccessibilityRole(core.RoleTabList)},
    Segment: comps.Chip{Style: []core.StyleProp{
        core.AccessibilityRole(core.RoleTab),
        core.AccessibilityControls("tabdemo-panel"),
    }},
}
core.Box(
    core.AccessibilityID("tabdemo-panel"),
    core.AccessibilityLabel(tabPageLabels[page.Get()]),
    core.Match(page.Get(), cases...),
)`),
				prose("Every segment points at the same id, and that is right rather than lazy: "+
					"there is one region and its contents change, so three ids would name two "+
					"regions that do not exist. The Box also picks up role=\"group\" from both "+
					"web exporters — ARIA forbids an accessible name on a plain div, so without "+
					"a role the label you just gave the panel would be dropped by every browser "+
					"and read out by both phones. You never ask for that one; it is supplied."),
				demoPanel("The strip writes an int; Match reads it — the tab contract, hand-assembled.",
					comps.SegmentedControl{
						Style:     append(append([]core.StyleProp{}, segWrap...), core.AccessibilityRole(core.RoleTabList)),
						Labels:    tabPageLabels,
						Selected:  page.Get(),
						OnSelect:  func(i int) { page.Set(i) },
						KeyPrefix: "tabdemo-",
						// The segment template: every chip is a tab, and every
						// one of them points at the single region below.
						Segment: comps.Chip{Style: []core.StyleProp{
							core.AccessibilityRole(core.RoleTab),
							core.AccessibilityControls(tabDemoPanelID),
						}},
					},
					core.Box(
						core.AccessibilityID(tabDemoPanelID),
						// Named for the page that is showing, so a reader
						// following a tab's aria-controls arrives somewhere
						// that says where it landed.
						core.AccessibilityLabel(tabPageLabels[page.Get()]),
						core.Match(page.Get(),
							core.Case(0, core.Column(
								core.Gap(6),
								prose("Pages are plain views — this one is an ordinary Column riding "+
									"along as a child."),
								core.Row(
									core.Gap(8),
									core.AlignItemsProp(core.AlignItemsCenter),
									comps.Badge{Text: "page 1 of 3"},
									caption("nothing here knows it lives in a tab"),
								),
							)),
							core.Case(1, core.Column(
								core.Gap(6),
								caption("64% of the gopher quota used:"),
								comps.ProgressBar{Value: 0.64, AccessibilityLabel: "Gopher quota"},
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
				),
				keyPoints(
					"Tabs adds names to core.TabView's positional props and delegates — the node type and its wire contract stay in core, next to the renderers.",
					"All pages are children of the node; the native side draws the strip and shows Selected. Selection is controlled: Selected in, OnChange out.",
					"The web host doesn't render TabView yet — compose the contract there: SegmentedControl for the strip, Match for the pages.",
					"A hand-built strip states its own semantics: RoleTabList on the row, RoleTab on the segments, and AccessibilityID / AccessibilityControls for the region each tab shows — the one relationship core.Style carries, because an element cannot be said as a value.",
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
func archiveMonth(e archiveEntry) comps.Group {
	return comps.Group{Key: e.date.Format("2006-01"), Label: e.date.Format("January 2006")}
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
			sortBy := core.NewState[*comps.Sort](ctx, nil)
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

			table := comps.DataTable[archiveEntry]{
				Columns: []comps.Column[archiveEntry]{
					{Title: "Title", Weight: 2,
						Text: func(e archiveEntry) string { return e.title },
						Less: func(a, b archiveEntry) bool { return a.title < b.title }},
					{Title: "Speaker", Weight: 1, Narrow: true,
						Text: func(e archiveEntry) string { return e.speaker },
						Less: func(a, b archiveEntry) bool { return a.speaker < b.speaker }},
					// A fixed width, so "Mar 1" and "Mar 22" line up down the
					// column instead of each row's date hugging its own text.
					{Title: "Date", Align: core.JustifyEnd, Width: 56,
						Text: func(e archiveEntry) string { return e.date.Format("Jan 2") },
						Less: func(a, b archiveEntry) bool { return a.date.Before(b.date) }},
				},
				Rows:     archive,
				Key:      archiveKey,
				Sort:     sortBy.Get(),
				OnSort:   func(s comps.Sort) { sortBy.Set(&s) },
				Compact:  compact.Get(),
				Dividers: true,
				Pagination: &comps.Pagination{
					Page:     page.Get(),
					PageSize: 4,
					OnChange: page.Set,
					// Archive-flavored steppers. Also keeps the demo's Next
					// distinct from the lesson screen's own "Next ›" below it.
					PrevLabel: "‹ Newer",
					NextLabel: "Older ›",
				},
			}

			grouped := comps.GroupedList[archiveEntry]{
				Items:   loaded,
				Key:     archiveKey,
				GroupBy: archiveMonth,
				Row: func(e archiveEntry) core.View {
					return comps.ListRow{
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
				Footer: comps.LoadMore{
					HasMore:    shown.Get() < len(archive),
					OnLoadMore: func() { shown.Set(shown.Get() + loadMorePageSize) },
				},
			}

			// The banded demo's own state: which months are shut.
			//
			// A set held by the *screen*, which is the whole of comps.
			// Collapse's argument for being caller-owned — the widget calls no
			// hook, so it could not keep this without becoming un-renderable
			// inside a core.IfElse, and "which months are shut" is screen state
			// that should survive a pager reload anyway.
			// Every month starts shut, which is the half of Collapse a Header
			// override does *not* own: the run is withheld by the widget, so
			// this demo opens as four bands and nothing else, and a tap on one
			// brings its rows back. Derived from the fixture rather than
			// written out, so a month added to the archive is shut too instead
			// of being the one run that opens for no stated reason.
			allShut := make(map[string]bool, 4)
			for _, e := range archive {
				allShut[archiveMonth(e).Key] = true
			}
			shutMonths := core.NewState(ctx, allShut)
			collapse := comps.Collapse{
				IsCollapsed: func(g comps.Group) bool { return shutMonths.Get()[g.Key] },
				OnToggle: func(g comps.Group) {
					// Cloned rather than mutated: core.State compares by
					// reference to decide whether anything changed, so writing
					// into the live map would toggle nothing on screen.
					next := maps.Clone(shutMonths.Get())
					next[g.Key] = !next[g.Key]
					shutMonths.Set(next)
				},
			}

			banded := comps.GroupedList[archiveEntry]{
				Items:    archive,
				Key:      archiveKey,
				GroupBy:  archiveMonth,
				Collapse: collapse,
				Row: func(e archiveEntry) core.View {
					return comps.ListRow{Title: e.title, Subtitle: e.speaker}
				},
				// The override, and the reason CollapseBand exists. Collapse
				// reaches past a Header for the row hiding and stops at it for
				// the *control*, because a band the widget also built would be
				// a second control for the same run — so an override that
				// wanted a collapsible band used to owe a button, an
				// aria-expanded stated on every pass, and a heading wrapper.
				Header: func(g comps.Group) core.View {
					return core.Row(
						core.AlignItemsProp(core.AlignItemsCenter),
						// No padding on the row. The insets are on the
						// control instead, which is the point of
						// ControlStyle: a band's padding on the row holding
						// the button is dead space, since the button fills
						// the box it was given and 16px of it would be a
						// place a press does nothing.
						core.Padding(0),
						core.PaddingRight(16),
						comps.CollapseBand{
							Collapse: collapse,
							Group:    g,
							// FlexGrow on the wrapper, so the tally sits hard
							// against the trailing edge — the same thing the
							// default band does with its own count badge.
							Style: []core.StyleProp{core.FlexGrow(1)},
							ControlStyle: []core.StyleProp{
								core.PaddingLeft(16),
								core.PaddingRight(8),
								core.PaddingVertical(10),
							},
						},
						comps.Badge{Text: strconv.Itoa(g.Count)},
					)
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
				codeBlock(`comps.DataTable[Entry]{
    Columns: []comps.Column[Entry]{
        {Title: "Title", Weight: 2, Text: title, Less: byTitle},
        {Title: "Speaker", Narrow: true, Text: speaker},
        {Title: "Date", Align: core.JustifyEnd, Text: day},
    },
    Rows:    entries,
    Key:     entryKey,
    Sort:    sortBy.Get(),
    OnSort:  func(s comps.Sort) { sortBy.Set(&s) },
    Compact: compact.Get(),
    Pagination: &comps.Pagination{
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
						comps.Chip{Label: "Compact", Selected: compact.Get(),
							OnTap: func() { compact.Set(!compact.Get()) }},
						core.If(sortBy.Get() != nil, comps.Chip{Label: "Clear sort",
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
				codeBlock(`comps.GroupedList[Entry]{
    Items:   pager.Items,
    Key:     entryKey,
    GroupBy: func(e Entry) comps.Group {
        return comps.Group{
            Key:   e.Date.Format("2006-01"),
            Label: e.Date.Format("January 2006")}
    },
    Row:    func(e Entry) core.View { return entryRow(e) },
    HideTrailingCount: pager.HasMore,
    Footer: comps.LoadMore{
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
				prose("Bands can collapse, and the state is yours: Collapse is a predicate and a "+
					"handler, and the screen keeps the set of shut keys. Two fields rather than "+
					"one because they are useless apart — a predicate with no handler hides rows "+
					"behind a band nobody can operate, and a handler with no predicate announces "+
					"a state it does not have. The widget owning the set instead would cost it "+
					"the thing that makes it composable: GroupedList calls no hook, so it can be "+
					"rendered inside a core.IfElse without moving anybody's hook cursor."),
				prose("A Header override changes what that costs. Collapse reaches past your "+
					"header for the row hiding — your run still collapses — and stops at it for "+
					"the control, because a band the widget also built would be a second control "+
					"for the same run. comps.CollapseBand is that control on its own: the "+
					"heading, the button, and the aria-expanded that has to be restated on every "+
					"pass, with none of the default band's chrome. Its ControlStyle is where "+
					"your insets go, and that is not a preference — padding put on the row "+
					"around it is dead space, because the button fills the box it was handed and "+
					"a press 16px into the margin does nothing."),
				codeBlock(`shut := core.NewState(ctx, map[string]bool{})
collapse := comps.Collapse{
    IsCollapsed: func(g comps.Group) bool { return shut.Get()[g.Key] },
    OnToggle: func(g comps.Group) {
        next := maps.Clone(shut.Get())
        next[g.Key] = !next[g.Key]
        shut.Set(next)
    },
}

comps.GroupedList[Entry]{
    Items: entries, Key: entryKey, GroupBy: byMonth,
    Collapse: collapse,
    Header: func(g comps.Group) core.View {
        return core.Row(core.Padding(0), core.PaddingRight(16),
            comps.CollapseBand{
                Collapse: collapse, Group: g,
                Style:        []core.StyleProp{core.FlexGrow(1)},
                ControlStyle: []core.StyleProp{core.PaddingLeft(16), core.PaddingVertical(10)},
            },
            comps.Badge{Text: strconv.Itoa(g.Count)},
        )
    },
}`),
				demoPanel("Every month starts shut. Tap a band to open its run — the chevron and the aria-expanded are the control's, the badge beside it is the caller's own, kept outside the button so a reader is not told the count as part of its name.",
					banded,
				),
				prose("Whatever goes inside the button is presentational — a reader does not "+
					"descend into a control, and its name comes from Group.Label — which is why "+
					"the tally sits outside it here and in the default band both. Hand the same "+
					"CollapseBand a zero Collapse and it renders the label in a plain heading "+
					"rather than a button with a dead handler: an expansion stated with nothing "+
					"to toggle it is announced on both web targets, is silently nothing on "+
					"Android, and is exactly what core.AuditTree reports as an inert "+
					"disclosure."),
				keyPoints(
					"Both widgets are hook-free and fully controlled: Sort, Page and Compact live in your state; OnSort and OnChange report intent.",
					"Give a column Less only when Rows is the whole set; when the server pages, use Sortable and put the sort in the query — a client-side sort of one page is a partial sort wearing a total one's header.",
					"Sort, then page, then group — a client-side page's headers agree with its rows, and group runs follow the sort.",
					"Grouping is by run, not by bucket: sorted input yields one header per group; an append-only pager never moves an earlier header.",
					"HideTrailingCount hides the last group's badge while a pager has more to fetch: a closed group's count is final, an open one's is a number about to change.",
					"LoadMore is the four-state tail every paged screen hand-rolls: nothing, Load more, Loading…, or the error with Retry — Loading wins over Err, Err over HasMore.",
					"Key must be unique across the list and stable across renders; core.List keeps row state attached to it through reorders.",
					"Collapse is one type with two functions because either alone is broken; the set of shut keys is screen state, so the widget stays hook-free.",
					"A Header override keeps the row hiding and owns the control: place a comps.CollapseBand in your own row rather than rebuilding a button, an aria-expanded and a heading wrapper.",
					"Put your insets on CollapseBand.ControlStyle, not on the row around it — padding outside the button is a place a press does nothing.",
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
				body = comps.Skeleton{Lines: 3, AccessibilityLabel: "Loading the archive"}
			case len(matches) == 0:
				body = comps.EmptyState{
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
					rows = append(rows, comps.ListRow{
						Title:    e.title,
						Subtitle: e.speaker,
						Trailing: caption(e.date.Format("Jan 2")),
					})
				}
				body = core.Column(append([]core.PropsAndChildren{
					core.Gap(0), core.Padding(0),
				}, rows...)...)
			}

			chips := make([]comps.Chip, 0, len(archiveSpeakers))
			for _, s := range archiveSpeakers {
				label, value := s, s
				if s == "" {
					label = "All"
				}
				chips = append(chips, comps.Chip{
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
				codeBlock(`comps.AppBar{Title: "Archive", Subtitle: "9 sermons",
    Actions: []core.View{refreshButton}}

comps.Banner{Text: "Reconnecting…",
    Variant: comps.VariantWarning,
    ActionLabel: "Retry now", OnAction: retry}

comps.EmptyState{Glyph: "🔎", Title: "Nothing matches that",
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
				demoPanel("Type to search — the list moves a moment after you stop. Toggle the two simulations; tap ‹ once to see OnBack replace Pop, twice to leave.",
					comps.AppBar{
						Title:    "Archive",
						Subtitle: fmt.Sprintf("%d of %d sermons", len(matches), len(archive)),
						// The first back shows the note; a second one pops. OnBack
						// is also what Android's system back runs while this bar is
						// on screen (see AppBar.OnBack), so a handler that only ever
						// set the note left the lesson with no way out but
						// "‹ Contents". Popping on the second press keeps the point
						// of the demo — the first press visibly did not pop — and
						// gives system back its exit.
						OnBack: func() {
							if backNote.Get() {
								core.Pop(ctx)
								return
							}
							backNote.Set(true)
						},
						Actions: []core.View{comps.Button{
							Label:              "↻",
							Emphasis:           comps.EmphasisGhost,
							AccessibilityLabel: "Refresh",
							OnTap:              func() { loading.Set(!loading.Get()) },
						}},
					},
					core.If(backNote.Get(), caption("Back tapped — OnBack ran instead of core.Pop, so you are still here. Back again leaves the lesson.")),
					core.If(offline.Get(), comps.Banner{
						Text:        "Offline. Showing a saved copy.",
						Variant:     comps.VariantWarning,
						ActionLabel: "Reconnect",
						OnAction:    func() { offline.Set(false) },
						OnDismiss:   func() { offline.Set(false) },
					}),
					comps.SearchField{
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
					comps.ChipStrip{Chips: chips},
					core.Row(
						core.Gap(16), core.Padding(0),
						comps.StatTile{
							Label: "Showing", Value: itoaLen(len(matches)), Fill: true,
							Delta:        fmt.Sprintf("of %d", len(archive)),
							DeltaVariant: comps.VariantDefault,
						},
						comps.StatTile{
							Label: "Filter", Value: chipValueLabel(speaker.Get()), Fill: true,
						},
					),
					body,
					core.Row(
						core.Gap(8), core.Padding(0),
						comps.Chip{Label: "Simulate offline", Selected: offline.Get(),
							OnTap: func() { offline.Set(!offline.Get()) }},
						comps.Chip{Label: "Simulate loading", Selected: loading.Get(),
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

comps.SearchField{
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

			chips := make([]comps.Chip, 0, len(archiveSpeakers))
			for _, s := range archiveSpeakers {
				label, value := s, s
				if s == "" {
					label = "Everyone"
				}
				chips = append(chips, comps.Chip{
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
					comps.ChipStrip{Scrollable: true, Chips: chips},
					core.Box(
						core.Height(endlessViewport),
						core.Overflow("auto"),
						core.BorderRadius(8),
						comps.GroupedList[archiveEntry]{
							Items:             rows,
							Key:               archiveKey,
							GroupBy:           archiveMonth,
							StickyHeaders:     true,
							HideTrailingCount: hasMore,
							Row: func(e archiveEntry) core.View {
								return comps.ListRow{
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
							Footer: comps.LoadMore{
								HasMore:    hasMore,
								OnLoadMore: loadNext,
							},
						},
					),
					// The footer is a strip too, and the other shape of one:
					// shorter than the screen, so it has free space, and a
					// FlexGrow spacer takes it and pushes the count to the far
					// edge. The chip strip above overflows a phone, where a
					// grower gets nothing; this one is the case where CSS hands
					// the grower the leftover width, which Compose's
					// GrMobGrowStrip reproduces inside a horizontal scroll.
					//
					// Children rather than Chips: a strip given Children draws
					// them instead of its Chips, and this one holds more than
					// chips.
					comps.ChipStrip{
						Scrollable: true,
						Children: []core.View{
							comps.Chip{Label: "Start over", OnTap: func() {
								loaded.Set(endlessPageSize)
								fetches.Set(0)
							}},
							core.Box(core.FlexGrow(1)),
							caption(fmt.Sprintf("%d of %d rows, %d fetches", len(rows), total, fetches.Get())),
						},
						Style: []core.StyleProp{core.AlignItemsProp(core.AlignItemsCenter)},
					},
				),
				prose("Keep the footer. A screen that drops its LoadMore for OnEndReached gains a "+
					"feed that stops silently at whatever page failed, and loses the one control "+
					"that still works on a static export or in a browser with no "+
					"IntersectionObserver. Handing the same load function to both is the intended "+
					"shape: the debounce means a tap and a scroll cannot double-load, because "+
					"neither can fire while the row count is unchanged."),
				codeBlock(`comps.GroupedList[Entry]{
    Items:         pager.Items,
    StickyHeaders: true,
    OnEndReached:  pager.LoadNext,   // the scroll
    Footer: comps.LoadMore{     // and the tap, and the states
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
// for the reason comps.Calendar makes Today a field in the first place:
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

// tutorialAlsoOn is the calendar lesson's second source of dots: whatever
// else a day carries besides its sermon — a baptism, a members' meeting.
// Keyed by the day rather than by instant for the reason archiveOn compares
// that way.
//
// It exists because the 4.6 archive has at most one entry on any day, so a
// count taken over it alone could never draw more than one dot — and a Marked
// that cannot show two is the exact signature this lesson is here to explain.
// March 1 ends up with three marks and March 15 with two, which is what makes
// the cluster legible as a count in the demo rather than in prose.
var tutorialAlsoOn = map[string]int{
	"2026-03-01": 2,
	"2026-03-15": 1,
}

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
		Summary: "comps.Calendar and comps.DatePicker — a controlled month, a today you supply, and cells built at midday for a reason.",
		Body: func(ctx *core.Context) core.View {
			// Which month is on screen, and which day is chosen. Both belong
			// to the caller: this demo drives one calendar and one date field
			// from the same selection, which it could not do if the widget
			// owned either.
			month := core.NewState(ctx, tutorialToday)
			picked := core.NewState(ctx, time.Time{})

			// The count the grid draws: the day's sermon, if it has one, plus
			// anything else on it. A count, not a bool — one dot and two dots
			// are different facts about a Sunday.
			marked := func(d time.Time) int {
				n := tutorialAlsoOn[d.Format("2006-01-02")]
				if _, ok := archiveOn(d); ok {
					n++
				}
				return n
			}

			// What the chosen day has on it — the reason a calendar is worth
			// more than a text field.
			note := "Tap a dotted day. One dot is one thing on it — March 1 has three, and tapping a chosen day again clears it."
			if p := picked.Get(); !p.IsZero() {
				if e, ok := archiveOn(p); ok {
					note = fmt.Sprintf("%s — %s, %s", e.title, e.speaker, p.Format("Monday 2 January 2006"))
				} else {
					note = fmt.Sprintf("Nothing on %s.", p.Format("Monday 2 January 2006"))
				}
				// What the dots said, spelled out: the grid can draw the count
				// and cannot name what it is counting.
				if extra := tutorialAlsoOn[p.Format("2006-01-02")]; extra > 0 {
					note += fmt.Sprintf(" (and %d more that day)", extra)
				}
			}

			return core.Column(
				core.Gap(14),
				prose("A calendar is a grid of buttons over a little date arithmetic, and both are the "+
					"kind of thing every app rewrites slightly differently. comps.Calendar draws "+
					"the month; you own what it shows. The month on screen, the selected day and the "+
					"day that counts as today are three separate fields, because a screen that opens "+
					"on the month of its next event needs to say so."),
				codeBlock(`month  := core.NewState(ctx, someDate)
picked := core.NewState(ctx, time.Time{})

comps.Calendar{
    Month:         month.Get(),   OnMonthChange: month.Set,
    Selected:      picked.Get(),  OnSelect:      picked.Set,
    Today:         today,                       // a field, not a clock read
    Min:           season.Start, Max: season.End,
    Marked:        func(d time.Time) int { return len(eventsOn(d)) },  // a count, so two dots mean two
    Deselectable:  true,                                               // a second tap on the chosen day reports the zero time
}`),
				demoPanel("The dots are the 4.6 archive. The ring is \"today\"; the fill is your selection; the arrows die at the ends of the range.",
					comps.Calendar{
						Month:         month.Get(),
						OnMonthChange: month.Set,
						Selected:      picked.Get(),
						OnSelect:      picked.Set,
						Today:         tutorialToday,
						Min:           tutorialCalMin,
						Max:           tutorialCalMax,
						Marked:        marked,
						// This grid is a filter over the caption below it, so
						// "no day" is a state it can be in. The DatePicker
						// under it is a field and is not — it clears through
						// its own Clear button, and forces this off.
						Deselectable: true,
					},
					caption(note),
					comps.FormField{
						Label: "Same selection, as a field",
						Hint:  "DatePicker is the grid behind a summary, with the two view states it needs of its own.",
						Input: comps.DatePicker{
							Selected:    picked.Get(),
							OnSelect:    picked.Set,
							OnClear:     func() { picked.Set(time.Time{}) },
							Placeholder: "Choose a date",
							Title:       "Service date",
							Calendar: comps.Calendar{
								Today:  tutorialToday,
								Min:    tutorialCalMin,
								Max:    tutorialCalMax,
								Marked: marked,
							},
						},
					},
				),
				prose("Marked counts; it does not answer yes or no. Two services on one Sunday and one "+
					"service on one Sunday are different facts about the day, and a reader scanning a "+
					"month for its busy weeks is asking exactly that — so the cell draws one dot per "+
					"thing, up to three. A caller holding only a yes/no writes it as a count and loses "+
					"nothing. Past three the answer a reader takes away is \"several\" rather than a "+
					"number, which is what a capped cluster says; an exact count that matters goes into "+
					"DayLabel, where a screen reader can read it out. The dots themselves are hidden "+
					"from assistive technology, because the widget knows how many things a day holds "+
					"and nothing about what any of them is."),
				prose("Deselectable is how a grid says \"nothing\". Selected already spells that as the "+
					"zero time, so a second tap on the chosen day reports the same zero back through "+
					"OnSelect — the value makes a round trip through your state and there is no second "+
					"callback to wire. It is off by default, and the default is the interesting half: a "+
					"picker asking which day the appointment is has no \"no day\" to offer, and a stray "+
					"second tap that quietly emptied the field would lose an answer nobody asked to "+
					"lose. The grid above opts in because it is a filter; the DatePicker under it does "+
					"not, and clears through its own Clear button."),
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
				codeBlock(`comps.FormField{
    Label: "Event date",
    Input: comps.DatePicker{
        Selected: date.Get(),
        OnSelect: date.Set,
        OnClear:  func() { date.Set(time.Time{}) },
        Calendar: comps.Calendar{Today: today, Min: today},  // the template
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
					"Marked returns a count, not a bool: the cell draws one dot per thing on the day, capped at three, and a yes/no caller returns 0 or 1.",
					"Marked is called 42 times a render, adjacent months included — make it a lookup, not a query.",
					"Deselectable lets a second tap on the chosen day report the zero time, which is what Selected already means by it. Off by default, because a picker has no \"no day\" and DatePicker forces it off.",
					"MonthLabel, WeekdayLabel and DayLabel are the localization seams: Go's time package speaks English only.",
					"DatePicker owns two view states and takes a Calendar as a template; wrap it in FormField for the label, hint and error.",
				),
			)
		},
	}
}

// --- 4.10 ----------------------------------------------------------------

// tutorialBearings are the demo's hand-picked headings: one per quadrant plus
// the seam. 359 is there deliberately — a compass that looks right at 0, 90,
// 180 and 270 and wrong at 359 has a normalisation bug, and this is where a
// reader can see that it does not.
var tutorialBearings = []float64{0, 45, 135, 217, 300, 359}

func lessonCompass() Lesson {
	return Lesson{
		Title:   "Sensors: the compass",
		Summary: "core.Rotate, hooks.UseHeading and comps.Compass — a paint transform, a refcounted sensor, and the difference between \"no compass\" and \"no reading yet\".",
		Body: func(ctx *core.Context) core.View {
			// The hand-driven bearing: what the widget draws when a caller
			// supplies the number itself. Compass takes a float and not a
			// core.Heading precisely so this works — a route leg or a wind
			// direction is a bearing too.
			bearing := core.NewState(ctx, 45.0)

			// The live one. Mounting this hook turns the device's
			// magnetometer on and leaving the lesson turns it off, because a
			// lesson is a navigation route and a route's cleanup registry is
			// what releases the reference. That is the arrangement UseHeading
			// asks for in as many words.
			live := hooks.UseHeading(ctx)

			// True north is where the compass runs into the permission
			// package, and it is the reason that package has functions in it
			// at all. A magnetic bearing is free on every platform; a
			// *geographic* one needs the local declination, which needs
			// knowing where on the planet you are — so iOS reports
			// Heading.HasTrue only once location authorization has been
			// granted, and neither Android's rotation vector nor the browser's
			// orientation events carry it at all.
			//
			// UsePermission checks and never prompts, which is what makes it
			// safe here: a Request from inside a render pass would put the OS
			// dialog on screen as a side effect of drawing.
			locationStatus := hooks.UsePermission(ctx, permission.Location)

			chips := make([]comps.Chip, 0, len(tutorialBearings))
			for _, deg := range tutorialBearings {
				value := deg
				chips = append(chips, comps.Chip{
					Label:    fmt.Sprintf("%.0f°", value),
					Selected: bearing.Get() == value,
					OnTap:    func() { bearing.Set(value) },
				})
			}

			// What the live half has to say, which is three different
			// sentences and not one. This is the whole reason Heading carries
			// Received beside Available.
			var liveNote string
			switch {
			case !live.Received:
				liveNote = "Waiting for the first reading. On a desktop browser this becomes " +
					"\"no compass\" after a couple of seconds; on a phone it should not last that long."
			case !live.Available:
				liveNote = "No compass here: " + live.Error
			default:
				liveNote = fmt.Sprintf("%.0f° %s, give or take %.0f°",
					live.Magnetic, core.Cardinal(live.Magnetic), live.Accuracy)
			}

			// The four states a permission-gated feature has to draw, which is
			// the whole argument for Status having four values. Prompt is the
			// only one with a button on it.
			var permissionNote string
			var permissionAction core.View = core.Fragment()
			switch locationStatus {
			case permission.Granted:
				if live.HasTrue {
					permissionNote = fmt.Sprintf("True north: %.0f°, %.0f° off magnetic.",
						live.True, core.AngleDelta(live.Magnetic, live.True))
				} else {
					permissionNote = "Location is granted. This platform still reports no true " +
						"heading — only iOS carries one, and only once it has a fix."
				}
			case permission.Prompt:
				permissionNote = "Undecided. Asking will show the platform's dialog."
				permissionAction = comps.Button{
					Label: "Use my location",
					// From a tap, never from the render pass. Every platform
					// here either requires that or punishes the alternative.
					OnTap: func() { permission.Request(permission.Location) },
				}
			case permission.Denied:
				permissionNote = "Refused. Asking again shows nothing on most platforms — the " +
					"fix is the system settings, which is why Denied and Prompt are two words."
			case permission.Unavailable:
				permissionNote = "This platform cannot grant it at all, so there is nothing to " +
					"ask for and no settings screen to send anyone to. A browser preview and " +
					"a Go test are both in this state."
			default:
				permissionNote = "Checking…"
			}

			return core.Column(
				core.Gap(14),
				prose("A compass is two things the framework did not have: a style prop that turns a "+
					"node, and a sensor. core.Rotate is the first — one float, clockwise degrees, about "+
					"the node's own centre. It is a paint transform on all four targets, so a turned "+
					"node keeps the space it laid out with and never shoves its siblings around."),
				codeBlock(`core.Box(
    core.Width("48px"), core.Height("48px"),
    core.Rotate(-heading),        // clockwise degrees, about the centre
)`),
				demoPanel("Pick a bearing. The rose turns the other way, which is what keeps N pointing at north.",
					comps.ChipStrip{Chips: chips},
					comps.Compass{Heading: bearing.Get(), ShowDegrees: true},
					caption(fmt.Sprintf("Compass{Heading: %.0f} — the rose is drawn with Rotate(%.0f).",
						bearing.Get(), -bearing.Get())),
				),
				prose("Which half turns is the design decision, not the arithmetic. A magnetic compass "+
					"has a fixed card and a needle that swings to north; a navigation compass — every "+
					"phone — turns the whole card under a fixed mark at twelve o'clock. This is the "+
					"second, because the question a phone user is asking is \"which way am I facing\", "+
					"and that is read off the top. So the rose rotates by minus the heading: turn the "+
					"device clockwise and the rose must turn counter-clockwise by the same amount to "+
					"keep pointing at the same piece of the world."),
				prose("The mark sits on the rose's rim, drawn over it, and for a long time it could "+
					"not: Box stacks vertically on all four targets and absolute positioning is "+
					"web-only, so a mark drawn over the rose would have been a web-only widget "+
					"wearing a portable name. core.ZStack is the container that fixed it — every "+
					"child in the same box, in tree order, so the last one written is on top."),
				codeBlock(`core.ZStack(
    core.Width("160px"), core.Height("160px"),
    rose,                             // painted first, underneath
    core.Column(                      // the mark, in a box as tall as the stack
        core.Height("160px"),
        core.Justify(core.JustifyStart),
        core.AlignItemsProp(core.AlignItemsCenter),
        core.Text("▼"),
    ),
)`),
				prose("A ZStack centres every layer, on all four targets — a SwiftUI ZStack, a Compose "+
					"Box and a single-cell CSS grid, all told to centre rather than left to their own "+
					"defaults, because Compose's is the top-left corner and the others' is the middle. "+
					"There is no per-child alignment prop. A layer that wants to be somewhere else "+
					"says so with its own box, which is what the Column above is: as tall as the "+
					"stack, justifying its one glyph to the start, so the mark lands on the rim while "+
					"the Column itself is centred like everything else."),
				codeBlock(`h := hooks.UseHeading(ctx)          // starts the sensor, releases it on unmount

switch {
case !h.Received:  // the first reading has not landed yet  -> a spinner
case !h.Available: // this device has no compass, h.Error says why
default:
    comps.Compass{Heading: h.Magnetic, ShowDegrees: true}
}`),
				demoPanel("The live sensor, if this device has one.",
					comps.Compass{Heading: live.Magnetic, Size: 120, ShowDegrees: live.Available},
					caption(liveNote),
					caption(fmt.Sprintf("Received=%v  Available=%v  Active=%v",
						live.Received, live.Available, live.Active)),
				),
				prose("Received and Available are two facts and a screen draws different things for "+
					"them. \"No reading yet\" is a spinner; \"this device has no compass\" is a "+
					"different screen entirely, and a spinner there spins forever. One boolean could "+
					"not say both, so Heading carries both."),
				prose("Starting and stopping are refcounted, not toggled. Two screens can each hold the "+
					"sensor and each let go, and the magnetometer stops when the second one does — not "+
					"the first. A plain on/off flag makes the opposite bug easy and silent: a badge in "+
					"a tab bar and a compass screen both start it, the screen is popped, and the badge "+
					"quietly stops updating with nothing in any log."),
				prose("On iOS Safari the browser will not hand over orientation events unless the "+
					"request came from a tap. core.StartHeading is where that request is made, so a "+
					"hook that mounts on navigation may be refused — and when it is, the reason comes "+
					"back as Available=false with a message, which is exactly what a \"tap to enable "+
					"the compass\" button is for. The recovery is a second StartHeading from inside "+
					"the tap."),
				prose("True north is where a sensor runs into an authorization. A magnetic bearing "+
					"is free everywhere; a geographic one needs the local declination, which needs "+
					"knowing where you are — so iOS fills in Heading.True only once location has "+
					"been granted, and the compass host deliberately prompts for nothing, because "+
					"a permission dialog nobody expected is worse than a bearing a few degrees off "+
					"a map. Asking is the app's job, and permission is where it lives."),
				codeBlock(`switch hooks.UsePermission(ctx, permission.Location) {  // checks, never prompts
case permission.Granted:     return mapView(ctx)
case permission.Prompt:      return askButton()   // Request from a tap
case permission.Denied:      return openSettingsHint()
case permission.Unavailable: return nil           // nothing to ask for here
default:                     return comps.Skeleton{}   // the check is in flight
}`),
				demoPanel("The live status. Unavailable in a browser preview; on a phone this is a real dialog.",
					caption("permission.Location — "+string(locationStatus)),
					caption(permissionNote),
					permissionAction,
				),
				prose("Check and Request are two functions because they are two operations, and "+
					"collapsing them is wrong in either direction. A check that prompts puts the OS "+
					"dialog up as a side effect of a screen mounting, which is the surest route to a "+
					"permanent refusal; a request that only checks leaves a button that does nothing. "+
					"So the hook checks on mount, and asking stays yours, from a gesture."),
				prose("Four statuses, not a bool, and the fourth is the one people leave out. Denied "+
					"is fixable in the system settings and Unavailable is not — a device with no "+
					"camera, a permission the manifest never declared, an app with no host attached "+
					"at all — so a screen that offers \"Open Settings\" for both sends someone to a "+
					"page with no switch on it. Unknown is the fifth and is the zero value: the "+
					"check is asynchronous, so the first pass has no answer and draws a placeholder."),
				prose("Nothing tells an app that a permission changed while it was in the background. "+
					"A user is refused, taps your \"Open Settings\" button, grants it there and comes "+
					"back — and UsePermission still says denied, because its one check happened on "+
					"mount. hooks.UsePermissionLive is the same hook plus a re-check on every return "+
					"to the foreground, and it is what a screen drawing a denied state wants. The "+
					"re-check is owned by the permission rather than by a screen: there is one device "+
					"with one camera, so five screens watching it are one check per resume between "+
					"them, and an app with no live watcher takes no lifecycle subscription at all."),
				prose("The angle is never folded onto the circle on its way to a renderer. 350 to 370 "+
					"and 350 to 10 point the same way and are not the same animation — the first "+
					"sweeps twenty degrees forwards and the second unwinds three hundred and forty the "+
					"other way. Nothing turning today, so nothing to see; the moment a Transition "+
					"reaches a rotated node it is the difference between a needle that nudges and one "+
					"that spins the long way round every time you pass north."),
			)
		},
	}
}

// --- 4.11 ----------------------------------------------------------------

// tutorialPlaces are the demo's hand-picked points: one ordinary coordinate,
// one at the antimeridian and one past the pole. The last two are there for the
// reason 4.10's 359° bearing is — a widget that looks right over Lisbon and
// wrong at 179°E has a wrapping bug, and this is where a reader can watch the
// two rules (clamp the latitude, wrap the longitude) do different things.
var tutorialPlaces = []struct {
	name     string
	lat, lng float64
}{
	{"Lisbon", 38.7223, -9.1393},
	{"Suva", -18.1416, 178.4419},
	{"Past the pole", 89, 20},
}

func lessonStaticMap() Lesson {
	return Lesson{
		Title:   "Maps: a picture and a hand-off",
		Summary: "comps.StaticMap — a map image from a provider you choose, and a tap that leaves for the platform's own maps app.",
		Body: func(ctx *core.Context) core.View {
			place := core.NewState(ctx, 0)
			marker := core.NewState(ctx, true)
			zoom := core.NewState(ctx, comps.DefaultMapZoom)

			current := tutorialPlaces[place.Get()%len(tutorialPlaces)]

			chips := make([]comps.Chip, 0, len(tutorialPlaces))
			for i, p := range tutorialPlaces {
				idx := i
				chips = append(chips, comps.Chip{
					Label:    p.name,
					Selected: place.Get() == idx,
					OnTap:    func() { place.Set(idx) },
				})
			}

			shown := comps.StaticMap{
				Lat: current.lat, Lng: current.lng,
				Zoom:   zoom.Get(),
				Marker: marker.Get(),
				Label:  current.name,
				// Stated, because there is no default to inherit any more —
				// and stated as the dead one on purpose. This lesson is about
				// the URL the widget builds, which OSMStaticMap still builds
				// correctly; the picture under the caption stays empty, and
				// the prose below says why. A lesson that hid that behind a
				// key nobody here has would teach the seam and not the
				// decision.
				Provider: comps.OSMStaticMap,
			}
			// The URL the widget will request, asked for rather than rebuilt:
			// Area() is the widget's own resolution of the defaults and the
			// clamps, so the caption below and the image beside it cannot
			// disagree about what was fetched. Doing this by hand — defaulting
			// the zoom here, clamping the size here — is exactly the second
			// answer Area() exists to prevent.
			requested := comps.OSMStaticMap(shown.Area())

			return core.Column(
				core.Gap(14),
				prose("A map is the first widget in this package that draws something nobody here "+
					"drew. comps.StaticMap builds a URL, hands it to core.Image, and makes the "+
					"whole thing tappable — so what you see is a picture a tile service rendered, and "+
					"what a tap does is leave for the platform's own maps app."),
				codeBlock(`comps.StaticMap{
    Lat: 38.7223, Lng: -9.1393,
    Label:  "Lisbon Baptist Church",
    Marker: true,
}`),
				prose("That is the whole widget on all four targets, with no renderer work behind it: "+
					"an Image node the reconciler already patches and a core.OpenURL the three hosts "+
					"already hand to the system. A live panning map is a different feature — a node "+
					"type with MapKit, osmdroid and Leaflet behind it — and the question it answers "+
					"is \"interact with a map\", not \"where is this\"."),
				demoPanel("Pick a place. The switch is a setting, so the marker changes on the tap.",
					comps.ChipStrip{Chips: chips},
					// SwitchRow rather than a ListRow with a trailing core.Switch:
					// the whole row is the target, and the switch is named after it.
					comps.SwitchRow{
						Title:    "Marker",
						Subtitle: "A pin at the centre",
						On:       marker.Get(),
						OnToggle: marker.Set,
					},
					comps.ChipStrip{Chips: zoomChips(zoom)},
					shown,
					caption(fmt.Sprintf("%s — %g, %g at zoom %d",
						current.name, current.lat, current.lng, zoom.Get())),
					caption("Requested: "+requested),
				),
				prose("The frame above is empty, and that is this lesson's real subject. The URL in "+
					"the caption is correct and the host it names is gone: staticmap.openstreetmap.de "+
					"was the OpenStreetMap community's keyless static-image service, it has been "+
					"discontinued, and it no longer resolves. It used to be this widget's default, "+
					"which is exactly why the widget no longer has one — a default is a decision "+
					"taken on a caller's behalf, and this is what it looks like when one expires "+
					"under them."),
				prose("So the provider is the seam, and it is a policy an app owns. GoogleStaticMap(key) "+
					"is in the box and resolves; anything else is one function. A StaticMap with no "+
					"Provider at all renders this same empty frame and reports "+
					"comps.ConcernNoMapProvider in debug mode, so a build that has not chosen "+
					"says so during development rather than shipping a grey rectangle."),
				codeBlock(`// A provider is one function, and it sees values already
// defaulted and already clamped — no zero Zoom, no 4000px width.
Provider: func(a comps.StaticMapArea) string {
    return "https://tiles.example.com/" + ...
}`),
				prose("The hand-off is one URL for three platforms, because nothing in this framework "+
					"knows which platform it is on — core.OpenURL promises only the portable part. So "+
					"the default is an https maps URL that all three resolve and that both phones "+
					"open in the installed app. An app that does know its platform can hand back a "+
					"geo: or maps:// URL instead, and that is the one thing the Label argument on a "+
					"MapHandoff is for: the cross-platform URL deliberately carries the coordinates "+
					"and not the name, because a name is a search and a search can land on a "+
					"different St Mary's in a different country."),
				codeBlock(`Handoff: func(lat, lng float64, label string) string {
    return fmt.Sprintf("geo:%g,%g?q=%g,%g(%s)", lat, lng, lat, lng, label)
}

// And returning "" is how you say there is no hand-off at all:
// the widget renders a picture with no link role and no callback.`),
				prose("Which is the accessibility decision too. A tappable map is a core.RoleLink, "+
					"not a RoleButton: a button does something here and a link goes somewhere else, "+
					"and this one leaves the app entirely, which a reader deserves to know before "+
					"they follow it. A map with no hand-off is a core.RoleImg — a picture standing "+
					"in for one fact, exactly as the compass rose does — and either way the image "+
					"inside is hidden, so the widget announces once instead of reading out a "+
					"provider URL."),
				prose("Two of the chips above are not places anyone will ask for, and they are the "+
					"ones worth watching. Latitude is clamped to where Web Mercator stops (±85.0511°), "+
					"because every tile service here projects with Mercator and the poles are at "+
					"infinity — a request past it comes back as an error image. Longitude wraps "+
					"instead: 190°E is 170°W, the same meridian, and clamping it to 180 would move "+
					"the point rather than name it. Two rules because they are two different "+
					"geographic facts."),
				keyPoints(
					"A picture plus a hand-off, not a map engine: Image + OpenURL, so it works on all four targets today.",
					"The provider is a policy. The keyless default is a volunteer service; a load-bearing map belongs on a paid one.",
					"One hand-off URL for three platforms, because nothing here knows its platform. A platform-specific one is a one-line func.",
					"Tappable is a link (it leaves the app); untappable is an img. The image inside is hidden either way.",
					"Lat 0, Lng 0 is the Gulf of Guinea. There is no unset coordinate, so a screen with no location yet renders a Skeleton instead.",
				),
			)
		},
	}
}

// zoomChips is the zoom picker for 4.11: three scales far enough apart that the
// image visibly changes, named for what each one shows rather than by number.
func zoomChips(zoom core.State[int]) []comps.Chip {
	levels := []struct {
		label string
		value int
	}{
		{"City", 11},
		{"Street", comps.DefaultMapZoom},
		{"Building", 18},
	}
	chips := make([]comps.Chip, 0, len(levels))
	for _, l := range levels {
		value := l.value
		chips = append(chips, comps.Chip{
			Label:    l.label,
			Selected: zoom.Get() == value,
			OnTap:    func() { zoom.Set(value) },
		})
	}
	return chips
}

// --- 4.12 ----------------------------------------------------------------

// tutorialPin is one marker in the live-map demo: an id that outlives the
// slice index, and a position. The id is the whole point of the type — a marker
// is identified by it in OnMarkerTap and keyed by it in the tree, and an index
// changes every time something is inserted above it.
type tutorialPin struct {
	id       string
	lat, lng float64
}

// The seed pins, one per walkable landmark near the map's opening region, so the
// demo opens with something to tap rather than with an empty map.
var tutorialPins = []tutorialPin{
	{"rossio", 38.7139, -9.1394},
	{"castelo", 38.7139, -9.1334},
	{"belem", 38.6970, -9.2065},
}

func lessonLiveMap() Lesson {
	return Lesson{
		Title:   "Live maps: markers and the echo guard",
		Summary: "core.MapView — the platform's own map, markers as keyed children, and the one rule that makes a controlled map usable.",
		Body: func(ctx *core.Context) core.View {
			pins := core.NewState(ctx, tutorialPins)
			selected := core.NewState(ctx, "")
			nextID := core.NewState(ctx, 1)
			// The region the app is asking for. It starts as the opening view and
			// is *not* written on every pan: the readout below comes from the
			// same state, so a pan that did not reach it is visible as a readout
			// that has not moved.
			region := core.NewState(ctx, core.Region{Lat: 38.7139, Lng: -9.1394, Zoom: 13})
			// What the map last reported, kept separately from what the app is
			// asking for. Two slots, because the whole lesson is the difference
			// between them.
			reported := core.NewState(ctx, core.Region{})

			// --- The live position, which is the closing prose made runnable ---
			//
			// UsePermissionLive rather than UsePermission, because the Denied
			// branch below sends the reader to the system settings and the
			// non-live hook would still say "off" when they came back: nothing
			// on any platform announces a permission change. That is the hook's
			// own documented reason to exist and this is its first consumer.
			locationStatus := hooks.UsePermissionLive(ctx, permission.Location)
			// And UNCONDITIONALLY, which is the part worth reading twice.
			//
			// core.NewState hands out hook slots by call position, so a hook
			// inside a `case permission.Granted:` arm shifts every slot after it
			// the moment the permission changes — core/debug.go's cursor audit
			// names that failure, and this package's TestMain turns the audit on.
			// So the shape this lesson cannot use is the one core.StartLocation
			// and hooks.UseLocation used to recommend; both docs now say what
			// this line does instead.
			//
			// What gates the sensor is the *route*, not a branch: a reader is
			// here because they navigated to this lesson, and leaving it pops
			// the route, whose cleanup registry releases the GPS. The permission
			// gates what gets DRAWN, below.
			//
			// And the second gate, which is what UseLocationWhen is: the flag
			// below turns the sensor off without unmounting anything. It
			// defaults to true, so mounting this lesson behaves exactly as it
			// did when this line read `hooks.UseLocation(ctx)` — the switch is
			// the demonstration, not the behaviour.
			gpsOn := core.NewState(ctx, true)
			fix := hooks.UseLocationWhen(ctx, gpsOn.Get())

			markers := make([]core.PropsAndChildren, 0, len(pins.Get()))
			for _, p := range pins.Get() {
				markers = append(markers, core.Marker(p.id, p.lat, p.lng, p.id))
			}

			mapItems := []core.PropsAndChildren{
				core.Width("100%"),
				core.Height("260px"),
				core.OnRegionChange(func(r core.Region) { reported.Set(r) }),
				core.OnMarkerTap(func(id string) { selected.Set(id) }),
				core.OnMapTap(func(lat, lng float64) {
					id := fmt.Sprintf("pin-%d", nextID.Get())
					nextID.Set(nextID.Get() + 1)
					pins.Set(append(append([]tutorialPin{}, pins.Get()...),
						tutorialPin{id, lat, lng}))
					selected.Set(id)
				}),
			}
			mapItems = append(mapItems, markers...)

			reportedNote := "Nothing reported yet — pan or zoom the map."
			if r := reported.Get(); r.Zoom != 0 {
				reportedNote = fmt.Sprintf("Reported: %.4f, %.4f at zoom %.2f",
					r.Lat, r.Lng, r.Zoom)
			}

			// The four sentences a position has, which is most of the reason
			// Location carries Received beside Available. "Not yet" and "never"
			// are different screens, and cold GPS can take tens of seconds.
			//
			// The order of the arms is the part worth keeping, and both ends of
			// it were got wrong once:
			//
			//   1. A fix in hand wins outright. A position that arrived is
			//      ground truth and the permission record is bookkeeping, so a
			//      status that has not caught up must not be able to hide a
			//      coordinate the sensor actually delivered.
			//   2. Then the permission, BEFORE Received. With Received tested
			//      first, a browser preview and a Go test — neither of which can
			//      ever be granted anything — would sit on "waiting for the
			//      first fix" forever, which is a spinner telling a lie.
			//      Received is a question about time only once there is a
			//      permission for a fix to arrive under.
			var fixNote string
			switch {
			case !gpsOn.Get():
				// First, because it is the app's own instruction and outranks
				// every reading: core keeps the last fix, so without this arm
				// the panel would print a coordinate beside a switch that says
				// the GPS is off.
				fixNote = "The GPS is off. The hook is still called on every pass — it is " +
					"the same slot — and it is holding no reference to the sensor."
			case fix.Received && fix.Available:
				// Accuracy printed beside the coordinates and not tucked away,
				// because a 2000m radius is a fix of the city and looks exactly
				// like a 5m one to any code that reads only Lat and Lng.
				fixNote = fmt.Sprintf("%.5f, %.5f — accurate to about %.0f m",
					fix.Lat, fix.Lng, fix.Accuracy)
			case locationStatus == permission.Denied || locationStatus == permission.Unavailable:
				fixNote = "No position, and none on the way — see the line below for why."
			case !fix.Received:
				fixNote = "Waiting for the first fix. Cold GPS can take half a minute outdoors " +
					"and forever indoors, which is what Received is for."
			default:
				fixNote = "No fix: " + fix.Error
			}

			// The four states a permission-gated feature has to draw, which is
			// the argument for Status having four values. Prompt is the only one
			// with a button on it, and the button is a tap and never a render.
			var fixPermissionNote string
			var fixPermissionAction core.View = core.Fragment()
			switch locationStatus {
			case permission.Granted:
				fixPermissionNote = "Granted. The sensor runs for as long as this lesson is " +
					"and the switch below is on."
			case permission.Prompt:
				fixPermissionNote = "Undecided, so the sensor has nothing to report yet. " +
					"Asking shows the platform's dialog."
				fixPermissionAction = comps.Button{
					Label:    "Use my location",
					Emphasis: comps.EmphasisOutlined,
					// From a tap, never from the render pass — see the
					// permission package. The readout above recovers on its own
					// once this is granted: the hosts hold a refused start open
					// and re-arm it when the answer changes, which they did not
					// always do.
					OnTap: func() { permission.Request(permission.Location) },
				}
			case permission.Denied:
				fixPermissionNote = "Refused. Asking again shows nothing on most platforms — " +
					"the fix is the system settings, and this readout updates when you come " +
					"back, because the hook above is the Live one."
			case permission.Unavailable:
				fixPermissionNote = "This platform cannot grant it at all. A Go test and a " +
					"static export are both here, which is why the readout says so rather " +
					"than spinning."
			default:
				fixPermissionNote = "Checking…"
			}

			return core.Column(
				core.Gap(14),
				prose("core.MapView is the platform's own map: MapKit on iOS, osmdroid over "+
					"OpenStreetMap on Android, Leaflet in the browser, and a placeholder box in "+
					"a static export. It is the node to reach for when the map is part of the "+
					"screen — a set of markers to compare, a region to explore, a place to pick "+
					"by tapping. For \"where is this\", comps.StaticMap is an image and a "+
					"hand-off, and it is almost always the right answer."),
				codeBlock(`core.MapView(core.Region{Lat: 38.7139, Lng: -9.1394, Zoom: 13},
    core.Width("100%"), core.Height("260px"),
    core.OnMarkerTap(func(id string) { open(id) }),
    core.Marker("rossio",  38.7139, -9.1394, "Rossio"),
    core.Marker("castelo", 38.7139, -9.1334, "Castelo"),
)`),
				prose("Markers are child nodes, not a prop — the same decision core.TextGrid "+
					"makes about its rows. A marker set sent as one value means every marker is "+
					"re-read whenever any of them moves, and the host rebuilds its whole "+
					"annotation layer: a visible flicker on every platform, and a lost callout on "+
					"two of them. As children they are ordinary keyed nodes, so the reconciler "+
					"emits one update for the one marker that moved. core.Marker keys itself from "+
					"its id, which is the one field it could not do without."),
				demoPanel("Tap the map to drop a pin; tap a pin to select it. The readout is what the map reported back.",
					core.MapView(region.Get(), mapItems...),
					caption(reportedNote),
					caption(fmt.Sprintf("Asking for: %.4f, %.4f at zoom %.2f — %d pins%s",
						region.Get().Lat, region.Get().Lng, region.Get().Zoom,
						len(pins.Get()), selectedNote(selected.Get()))),
					core.Row(
						core.Gap(8),
						comps.Button{Label: "Back to the centre", OnTap: func() {
							region.Set(core.Region{Lat: 38.7139, Lng: -9.1394, Zoom: 13})
						}},
						comps.Button{Label: "Show Belém", Emphasis: comps.EmphasisOutlined,
							OnTap: func() {
								region.Set(core.Region{Lat: 38.6970, Lng: -9.2065, Zoom: 15})
							}},
						comps.Button{Label: "Reset pins", Emphasis: comps.EmphasisGhost,
							OnTap: func() {
								pins.Set(tutorialPins)
								selected.Set("")
							}},
					),
				),
				prose("Now pan the map and watch the two lines. \"Reported\" moves and \"Asking "+
					"for\" does not — this demo deliberately does not echo the region into the "+
					"state it renders from. The map stays where you left it anyway, and that is "+
					"the one rule that makes a controlled map usable: Go's region is applied only "+
					"when it *changes*."),
				prose("Which is also why \"Back to the centre\" does nothing after a pan, if the "+
					"centre is what this demo is already asking for. Re-rendering the same region "+
					"is never a re-centre — from a host's side that is indistinguishable from the "+
					"unrelated re-render the guard exists to ignore. \"Show Belém\" moves the map "+
					"because it is a different region, and after it \"Back to the centre\" is a "+
					"different region too. An app that wants the button to work from a pan echoes "+
					"OnRegionChange into the state it renders from, which is one line and is what "+
					"core.MapView recommends."),
				prose("Without that rule a map is unusable. A map is the one widget whose value "+
					"the user changes continuously by touching it — so if every render re-centred "+
					"on Go's region, any unrelated re-render would snap the map back under the "+
					"finger. And an app that did echo the pan into state would fight its own "+
					"round trip, because the echo arrives a frame late and moves the map again. "+
					"So each host remembers the region it last applied and compares: Go moving "+
					"the map is an instruction, Go merely re-rendering is not. The two buttons "+
					"above are instructions, and they work."),
				codeBlock(`// The same comparison, read the other way, is what stops a host's own
// recentring from arriving back in Go as a user gesture — no timing
// flag required, which is what the web half's test proved necessary.
core.OnRegionChange(func(r core.Region) { region.Set(r) })  // echo, if you want one`),
				prose("A pan is reported once, after it stops. Each host throttles on its own "+
					"side of the bridge — a 120ms quiet window on iOS and the web, osmdroid's own "+
					"DelayedMapListener on Android — because a drag generates a region per frame "+
					"and every one that crossed would be a full Go render pass. The window also "+
					"coalesces a pinch, which ends as a pan and a zoom a few milliseconds apart."),
				prose("ShowUserLocation is the host map's own blue dot, and it is not "+
					"hooks.UseLocation. The dot comes from the map SDK's own plumbing and tells "+
					"Go nothing about where anybody is; the hook is a position fix in Go that "+
					"needs no map. They share one OS permission and nothing else — which is what "+
					"lets \"centre the map on me\" be a decision an app makes rather than a prop."),
				codeBlock(`// The dot, drawn by the platform:
core.MapView(region, core.ShowUserLocation())

// The coordinates, in Go. Call the hook unconditionally — it is a hook —
// and gate what you DRAW on the permission:
status := hooks.UsePermissionLive(ctx, permission.Location)
loc := hooks.UseLocation(ctx)
if loc.Received && loc.Available {
    region = core.Region{Lat: loc.Lat, Lng: loc.Lng, Zoom: 16}
}`),
				demoPanel("This panel runs the code above. On a phone it is a real fix; in a browser preview or a Go test it is the \"cannot\" branch.",
					caption(fixNote),
					caption(fixPermissionNote),
					core.Row(
						core.Gap(8),
						fixPermissionAction,
						comps.Button{
							Label:    "Centre the map on me",
							Emphasis: comps.EmphasisGhost,
							// Disabled rather than hidden, so the button is a
							// visible statement about the state rather than a
							// control that comes and goes. A tap with no fix
							// would set the region to 0,0 — which is a real
							// place in the Gulf of Guinea and a confusing map.
							Disabled: !fix.Received || !fix.Available,
							OnTap: func() {
								region.Set(core.Region{Lat: fix.Lat, Lng: fix.Lng, Zoom: 16})
							},
						},
					),
					// The flag, as a control the reader can flick. A switch
					// rather than a button because it shows its own state:
					// "the GPS is on" is a fact about right now, and a button
					// would only say what tapping it does. A SwitchRow, so the whole
					// row flicks it rather than only the control at its edge.
					comps.SwitchRow{
						Title:    "Keep the GPS on",
						Subtitle: "The flag hooks.UseLocationWhen takes",
						On:       gpsOn.Get(),
						OnToggle: gpsOn.Set,
					},
				),
				prose("That panel is the one place in this tutorial that asks the OS for "+
					"anything, and the order of the two hooks is the lesson. Both are called "+
					"on every pass, unconditionally: hook slots are handed out by call "+
					"position, so a UseLocation tucked inside a `case permission.Granted:` arm "+
					"moves every slot after it the instant the permission changes. What the "+
					"permission gates is the readout and the buttons, not the hook."),
				prose("Which leaves the question of when the dialog appears, and the answer is "+
					"the route. Mounting this lesson starts the sensor and leaving it stops "+
					"the sensor, because a lesson is a navigation route and a route's cleanup "+
					"registry is what releases the reference — so the reader asked for this by "+
					"navigating here. A screen that must not ask until a tap belongs behind a "+
					"button that navigates to it, which is the same mechanism spelled with one "+
					"more screen."),
				prose("The switch in that panel is the other gate, for the screen that cannot "+
					"be its own route — a settings pane with a map preview beside a permission "+
					"toggle. hooks.UseLocationWhen takes the flag as an argument, so the call "+
					"site never moves and the sensor follows the answer: flick it off and the "+
					"GPS is released, flick it on and the hook rejoins. That is the same intent "+
					"as wrapping the hook in an `if`, with the one difference that matters — "+
					"one slot, every pass, whatever the flag says."),
				codeBlock(`// The gate that is not a route, and not a conditional hook:
gpsOn := core.NewState(ctx, true)
fix := hooks.UseLocationWhen(ctx, gpsOn.Get())   // one slot either way

// NOT this — the slot moves the moment the flag changes:
// if gpsOn.Get() { fix = hooks.UseLocation(ctx) }`),
				prose("Android reports \"not granted\" and waits; iOS shows the dialog from the "+
					"sensor itself, because CLLocationManager can and startUpdatingLocation on "+
					"an undecided authorization reports nothing at all, forever. Either way the "+
					"refusal is an event and not a silence — Available: false with a reason — "+
					"and either way granting it afterwards works: both hosts hold a refused "+
					"start open and re-arm it when the answer changes, rather than needing the "+
					"screen to be remounted."),
				keyPoints(
					"MapView is for maps that are part of the screen; StaticMap is for \"where is this\", and is smaller in every way.",
					"Markers are keyed child nodes, so one that moves is one patch rather than a rebuilt layer.",
					"The region is applied only when it changes — otherwise a re-render snaps the map out from under the user.",
					"A pan is reported once, after it stops, and every host throttles on its own side.",
					"ShowUserLocation is the platform's dot; hooks.UseLocation is a fix in Go. Same permission, different features.",
					"Call both hooks unconditionally and gate what you draw — a hook inside a permission branch shifts every slot after it.",
					"UseLocationWhen is the gate for a screen that cannot be its own route: one slot on every pass, with the sensor following a flag.",
					"Accuracy is part of a position. 2000m is the cell tower's guess and looks exactly like a GPS fix to code that reads only Lat and Lng.",
					"In the browser the map needs Leaflet on the host page. Without it the node draws a placeholder that still carries its region.",
				),
			)
		},
	}
}

// selectedNote is the trailing clause of the live map's status line, kept out of
// the body for the reason the boolean lesson's sentence-builders are: one
// conditional, read once.
func selectedNote(id string) string {
	if id == "" {
		return ""
	}
	return ", " + id + " selected"
}

// --- 4.13 ----------------------------------------------------------------

// codeEditorSeed is the snippet the editing demo opens with. Short enough to
// read on a phone, and deliberately made of four different token classes — a
// comment, a keyword, a call and a string — so the colours are visible before
// the reader has typed anything.
//
// A raw literal rather than a codeBlock argument, because this one is *data*:
// it is the demo's initial state, not a snippet the lesson is showing. That
// difference is why it is not picked up by highlight_test.go's corpus walk,
// which reads codeBlock call sites.
const codeEditorSeed = `// Try me: edit, and the colours follow.
func greet(name string) string {
	return "hello, " + name
}`

func lessonCodeEditor() Lesson {
	return Lesson{
		Title:   "Editing code: the decorated buffer",
		Summary: "comps.CodeEditor — a real buffer with Go's own lexer behind it, and the three rules that make a host-owned buffer controllable from Go.",
		Body: func(ctx *core.Context) core.View {
			src := core.NewState(ctx, codeEditorSeed)
			// The toolbar's ref. A hook, and therefore called unconditionally
			// and before anything that might return early — the same rule every
			// other hook in this tutorial is under. The widget deliberately does
			// not call it for you; see its doc for why a code *block* must be
			// free of hook obligations.
			ref := core.UseEditorRef(ctx)
			// The caret, as the editor reports it. Two ints, because core parsed
			// the "start:end" the hosts send before this handler ever ran.
			selStart := core.NewState(ctx, 0)
			selEnd := core.NewState(ctx, 0)

			return core.Column(
				core.Gap(16),
				prose("Everything up to here has been a picture of code. core.TextGrid draws "+
					"rows of coloured runs and nothing can be typed into it, which is right "+
					"for a snippet in a document and useless for a config screen, a snippet "+
					"runner, or a rule the user is meant to write. comps.CodeEditor is "+
					"the other half: the same rows, over a buffer the platform owns."),
				demoPanel("Edit it. The lexer re-runs on every keystroke.",
					core.Column(
						core.Gap(10),
						comps.CodeEditor{
							Value:       src.Get(),
							OnChange:    src.Set,
							Language:    "go",
							LineNumbers: true,
							Toolbar:     ref,
							Height:      "170px",
							OnSelectionChange: func(start, end int) {
								selStart.Set(start)
								selEnd.Set(end)
							},
						},
						core.Text(selectionNote(selStart.Get(), selEnd.Get(), src.Get()),
							core.FontSize(13),
						),
					),
				),
				prose("The widget did three things. It ran highlight.Go() over the value and "+
					"handed the result to the node as one styled row per line. It painted the "+
					"surface from highlight.Darcula, because the token colours and the "+
					"background have to come from one scheme or the code is legible by "+
					"accident. And it built that toolbar out of comps.Buttons, each of "+
					"which sends one command to the ref you gave it."),
				codeBlock(`ref := core.UseEditorRef(ctx)      // the toolbar's address

comps.CodeEditor{
    Value:       src.Get(),
    OnChange:    src.Set,
    Language:    "go",        // or Highlighter: a highlight.Highlighter
    LineNumbers: true,
    Toolbar:     ref,         // no ref, no toolbar, and no hook either
    Height:      "170px",
}`),
				prose("The ref is yours and not the widget's, and that is a deliberate "+
					"asymmetry. A ref has to be stable across passes, which means a hook, which "+
					"means anything holding one must be rendered unconditionally on every "+
					"pass. That is a fine obligation for an editor with a toolbar and a bad one "+
					"for a read-only code block — which is what every snippet in this tutorial "+
					"now is, rendered inside conditionals and loops and lesson bodies. So the "+
					"toolbar names the ref, and an editor without one touches no hook."),
				comps.Separator{},
				prose("Underneath, the hard part is not the colours. It is that the buffer "+
					"belongs to the platform — a UITextView, a BasicTextField, a <textarea> — "+
					"while the value belongs to Go, and the round trip between them takes a "+
					"few milliseconds that the typist can out-run. Three rules make that "+
					"work, and all four renderers implement the same three."),
				prose("One: the echo guard, which core.TextArea has lived under since it "+
					"existed. Every value the host sends upstream is queued, and Go's echo of "+
					"it is dropped rather than written back — assigning it would throw the "+
					"caret to the end of the buffer, mid-word. A value Go sends that the host "+
					"never sent is something else entirely: a validator normalizing the text, "+
					"a draft cleared after a submit. That one lands even mid-typing, and "+
					"moving the caret is then correct, because the text under it was replaced."),
				prose("Two: decoration is advisory, and per line. Go's rows are a description "+
					"of the text as it was when Go last saw it, so the line being typed is "+
					"described wrongly for a frame. Each host compares a row's text with the "+
					"line under it and paints only where the two agree; the line you are "+
					"typing goes plain for one frame and every other line keeps its colours. "+
					"The rule never runs the other way — a row's text is never written into "+
					"the buffer — so a lexer that is wrong can make the screen ugly and can "+
					"never make it lose a character."),
				prose("Three: commands are epoch-stamped props, the same mechanism core.Focus "+
					"uses. RunEditorCommand bumps a counter on the ref, the next render pass "+
					"stamps the counter and the command string onto the editor, and the host "+
					"acts once when the counter changes. A counter rather than a flag because "+
					"indenting twice is two commands with the same string, and two identical "+
					"prop maps produce no patch at all."),
				codeBlock(`// The toolbar's whole body. The command acts on the host's
// own selection, which Go never has to know.
core.Button("Indent", func() {
    core.RunEditorCommand(ref, core.EditIndent)
})`),
				prose("There is one place an editor command deliberately differs from a focus "+
					"command. A focus command re-fires on a field that mounts while it is the "+
					"target — that is what makes \"push a screen and put the cursor in its "+
					"search box\" work. An editor command names a moment and an edit, so an "+
					"editor that was not on screen when it was issued missed it: every host "+
					"adopts a standing epoch without running it. Otherwise coming back to this "+
					"lesson would re-indent the snippet."),
				prose("The selection travels the same way a NumericInput's number does: as "+
					"text. The bridge has four channels — void, bool, int, text — and a "+
					"selection is two numbers, so the hosts send \"start:end\" and core parses "+
					"it before your func(start, end int) is called. The offsets are bytes into "+
					"the UTF-8 value, which is the one unit all four hosts can agree on: "+
					"Android and the browser count UTF-16, and iOS counts String.Index."),
				keyPoints(
					"comps.CodeEditor is the widget; core.CodeEditor is the node and highlight is the lexer.",
					"A read-only editor with no toolbar is the display half — a code block you can select and copy.",
					"No toolbar, no hook: the ref is the caller's, so an editor can be rendered inside a conditional.",
					"Language picks the lexer by name; Highlighter takes one of your own, including one that caches.",
					"The zero Scheme is picked from the theme's background, so a light app does not get a dark rectangle.",
					"The buffer is the host's while focused and Go's otherwise; Go's echo of your own keystroke never moves the caret.",
					"A row is applied only where it still describes its line, so a stale lexer costs one line's colour for one frame.",
					"Commands ride an epoch, not a flag, so the same command twice is two commands.",
					"Selection crosses as bytes into the UTF-8 value; each host converts from its own unit.",
					"Not in v1: autocomplete, folding, find-and-replace, a host-side grammar. Each is a driver away.",
				),
			)
		},
	}
}

// selectionNote is the editor demo's status line: where the caret is, in the
// unit the callback reports it in, plus the line it lands on.
//
// The line number is derived here rather than reported by the host, and that is
// the honest split: a host knows about characters and the wire carries byte
// offsets, while "which line is that" is a question about the value — which Go
// has. Counting newlines in the prefix is the whole calculation.
func selectionNote(start, end int, src string) string {
	if start > len(src) {
		// The report can arrive one pass ahead of the value it describes, which
		// is the same skew rule 2 exists for one level down. Saying nothing is
		// better than printing an offset past the end of the buffer.
		return ""
	}
	line := strings.Count(src[:start], "\n") + 1
	if start == end {
		return fmt.Sprintf("caret at byte %d — line %d", start, line)
	}
	return fmt.Sprintf("bytes %d–%d selected (%d) — from line %d", start, end, end-start, line)
}

// --- 4.14 ----------------------------------------------------------------

// richNoteSeed is the note the rich-text demo opens with: one of every block
// kind the toolbar offers and several of the marks, so the reader can see what
// the buttons do before pressing one.
//
// Built as values rather than parsed from Markdown, because the lesson's own
// point is that the document is the value — a parse here would quietly make the
// demo depend on the import door it is only supposed to illustrate.
var richNoteSeed = richtext.Doc{Blocks: []richtext.Block{
	{Kind: richtext.Heading2, Runs: []richtext.Run{{Text: "A note"}}},
	{Kind: richtext.Paragraph, Runs: []richtext.Run{
		{Text: "Some "},
		{Text: "bold", Bold: true},
		{Text: " and some "},
		{Text: "italic", Italic: true},
		{Text: ", and a "},
		{Text: "link", Link: "https://example.com"},
		{Text: "."},
	}},
	{Kind: richtext.Bullet, Runs: []richtext.Run{{Text: "a bullet"}}},
	{Kind: richtext.Bullet, Runs: []richtext.Run{{Text: "another"}}},
	{Kind: richtext.Quote, Runs: []richtext.Run{{Text: "Someone said this."}}},
}}

func lessonRichText() Lesson {
	return Lesson{
		Title:   "Rich text: a document as the value",
		Summary: "comps.RichTextEditor — formatted text whose value is a richtext.Doc, owned by Go and mapped by each host.",
		Body: func(ctx *core.Context) core.View {
			note := core.NewState(ctx, richNoteSeed)
			// The toolbar's state: the ref its buttons command, the last
			// selection its buttons are drawn from, and the link prompt. Four
			// hook slots, called unconditionally and before anything that could
			// return early — the same rule every other hook in this tutorial is
			// under.
			bar := comps.UseRichToolbar(ctx)
			showMarkdown := core.NewState(ctx, false)

			return core.Column(
				core.Gap(16),
				prose("A code editor's value is a string and its colours are computed from "+
					"it. A rich-text editor has no such split: the formatting is not derived "+
					"from the text, it *is* part of the value, and the user edits it "+
					"directly. So the value is a document — richtext.Doc — and every host "+
					"maps that document to and from its own text engine."),
				demoPanel("Write in it. The toolbar shows what is active under the caret.",
					core.Column(
						core.Gap(10),
						comps.RichTextEditor{
							Doc:         note.Get(),
							OnChange:    note.Set,
							Placeholder: "Write something…",
							Toolbar:     bar,
							MinHeight:   "150px",
						},
						comps.Button{
							Label:    markdownToggleLabel(showMarkdown.Get()),
							Emphasis: comps.EmphasisGhost,
							OnTap:    func() { showMarkdown.Set(!showMarkdown.Get()) },
						},
						core.If(showMarkdown.Get(),
							comps.CodeEditor{
								Value:    note.Get().Markdown(),
								ReadOnly: true,
								Height:   "150px",
							},
						),
						caption("The same document, read-only, as comps.RichTextView:"),
						comps.RichTextView{Doc: note.Get()},
					),
				),
				prose("That second panel is the same document through Doc.Markdown(), "+
					"re-rendered on every keystroke. It is import and export, not the wire: "+
					"Markdown cannot represent a selection-preserving edit, and making it the "+
					"value would put a Markdown parser in four hosts. What actually crosses "+
					"is the document's JSON, and that is also what you persist — bytdb takes "+
					"it as it is."),
				codeBlock(`bar := comps.UseRichToolbar(ctx)    // ref + selection + link prompt

comps.RichTextEditor{
    Doc:         note.Get(),
    OnChange:    note.Set,
    Placeholder: "Write something…",
    Toolbar:     bar,          // no bar, no toolbar, and no hook either
    MinHeight:   "150px",
}`),
				prose("The toolbar is the caller's for the same reason the code editor's is. "+
					"A ref has to be stable across passes, which means a hook, which means "+
					"anything holding one must be rendered unconditionally. A note being "+
					"*displayed* — a comment, a description, the body of a card in a list — "+
					"is the thing rendered inside an `if`, so the read-only editor has to be "+
					"free of hook obligations. ReadOnly with no toolbar is one display half. "+
					"comps.RichTextView, under the editor above, is the lighter one: each "+
					"block is a core.Paragraph, plain text runs with their marks and "+
					"tappable links, with no editor hosted per note, which is what a list "+
					"of fifty notes wants."),
				comps.Separator{},
				prose("Which buttons look pressed is the one thing Go cannot work out. Go "+
					"owns the document and the host owns the caret, so \"is the text under "+
					"the cursor bold\" is a question only the host can answer — it comes back "+
					"through OnRichSelectionChange as a core.RichSelection, and the toolbar "+
					"draws itself from the last one. That is widget-private state, the same "+
					"kind DatePicker's open sheet is: presentation only, nothing an app would "+
					"want to read."),
				prose("The commands ride the same epoch-stamped prop pair the code editor's "+
					"do. Two of them carry an argument in the string — core.EditBlock(kind) "+
					"and core.EditLink(url) — because the command channel is one prop, and "+
					"widening it to a map would change the shape all four hosts read for the "+
					"sake of two commands. Everything after the first colon is the argument, "+
					"so a URL keeps its own colons."),
				prose("The four hosts do genuinely different work here, and it is worth "+
					"knowing which. iOS maps the document onto an NSAttributedString and "+
					"edits marks straight into the text storage, which preserves the caret. "+
					"Android reaches past Compose to a classic EditText, because Spannable "+
					"has had a span type for every mark and every paragraph treatment for a "+
					"decade. The browser gets a contenteditable whose every command is a pure "+
					"transformation of the document — the selection is only ever read and "+
					"restored, never operated on, because a Range under contenteditable is "+
					"the least predictable surface on the web. And htmlout writes "+
					"Doc.HTML() and stops, because a snapshot has no caret."),
				prose("Paste is worth a sentence of its own. On the web it is intercepted, "+
					"and what reaches the document is the clipboard's text/plain — so nothing "+
					"a word processor put on the clipboard becomes part of the value. That is "+
					"the only way a document stays a richtext.Doc rather than whatever HTML "+
					"happened to be copied."),
				keyPoints(
					"The value is a richtext.Doc, not a string and not the platform's markup.",
					"The JSON is the wire and the storage; Markdown is the import/export door.",
					"Seven block kinds and six marks. Tables, images and nesting are non-goals until something drives them.",
					"A ReadOnly editor with no toolbar is the display half — and takes no hook, so it can be rendered anywhere.",
					"UseRichToolbar is the hook: the ref, the last selection, and the link prompt.",
					"Which buttons are pressed comes from the host, because Go owns the document and the host owns the caret.",
					"Commands are epoch-stamped props; EditBlock and EditLink carry their argument in the string.",
					"Paste is serialized through the document, so foreign markup dies at the edge.",
					"Undo is the platform's on the natives and the runtime's own stack of Docs on the web.",
				),
			)
		},
	}
}

// markdownToggleLabel is the caption of 4.14's Markdown button, kept out of the
// body for the reason the other sentence-builders in this chapter are: one
// conditional, read once.
func markdownToggleLabel(showing bool) string {
	if showing {
		return "Hide the Markdown"
	}
	return "Show the Markdown"
}

// lessonSmallControls is the Tier A bundle of comps that every app reaches for
// and used to hand-roll: a Stepper for small counts, a Rating, a Spinner for
// unshaped waiting, and a BottomBar with Screen.Footer to pin it. It is
// appended at the end of the chapter rather than beside ListRow so the numbers
// deep links already use (grmob://lesson/4.12 is the live map) do not move.
//
// The Spinner shows the other half of the declare-in-Go model: its ring
// carries core.Spin, so the platform turns it and Go sends nothing per frame.
// It holds no hooks, so core.If would be correct too; the demo still flips
// Hidden because the spinner has a fixed place in the row.
func lessonSmallControls() Lesson {
	return Lesson{
		Title:   "Small controls: Stepper, Rating, Spinner & BottomBar",
		Summary: "Four widgets every app hand-rolls — a clamped counter, a star row, a spinner that costs nothing when hidden, and a bar pinned by Screen.Footer.",
		Body: func(ctx *core.Context) core.View {
			// Hooks first and unconditionally, as in every lesson.
			guests := core.NewState(ctx, 2)
			stars := core.NewState(ctx, 0)
			loading := core.NewState(ctx, false)
			tab := core.NewState(ctx, 0)

			tabs := []string{"Home", "Search", "Me"}
			icons := []string{"🏠", "🔍", "👤"}
			items := make([]comps.BarItem, len(tabs))
			for i := range tabs {
				i := i // captured per item: OnTap outlives this loop pass
				items[i] = comps.BarItem{Icon: icons[i], Label: tabs[i], OnTap: func() { tab.Set(i) }}
			}

			rated := "Not rated yet"
			if stars.Get() > 0 {
				rated = fmt.Sprintf("You rated it %d of 5", stars.Get())
			}
			loadLabel := "Simulate loading"
			if loading.Get() {
				loadLabel = "Stop loading"
			}

			return core.Column(
				core.Gap(14),
				prose("A Stepper is the tapping form of a number. It clamps into Min..Max and "+
					"calls OnChange only when the value actually moves, so the handler is a "+
					"setter and the button that would leave the range is disabled for you. "+
					"Bounds are opt-in: they apply when Max > Min."),
				codeBlock(`comps.ListRow{
    Title:    "Guests",
    Trailing: comps.Stepper{Value: guests.Get(), Min: 1, Max: 8,
        OnChange: guests.Set, Label: "Guests"},
}`),
				prose("A Rating is a row of glyphs. Interactive, each star is a named button "+
					"(\"3 of 5\"); ReadOnly, the stars are decoration and the group's value is "+
					"the one announcement. Value is a float, so half-stars can arrive later "+
					"without changing your type."),
				prose("A Spinner's ring carries core.Spin, so each platform turns it on its own "+
					"frame clock and Go sends nothing while it spins: no patches, no render "+
					"passes. It holds no hooks, so leaving it out is safe; flipping Hidden keeps "+
					"its place in a layout, and a hidden spinner draws no frames."),
				codeBlock(`comps.Spinner{Hidden: !loading.Get()}`),
				prose("A BottomBar is navigation when Selected >= 0 and a toolbar when it is "+
					"negative. Put it in Screen.Footer, which pins it outside the scroll "+
					"region; among Children it would scroll away."),
				codeBlock(`comps.Screen{
    Scroll:   true,
    Children: []core.View{feed},
    Footer:   comps.BottomBar{Items: items, Selected: tab.Get()},
}`),
				demoPanel("Step the guests to a bound, rate it, start the spinner, and switch tabs.",
					comps.ListRow{
						Title:    "Guests",
						Subtitle: fmt.Sprintf("Booking for %d", guests.Get()),
						Trailing: comps.Stepper{
							Value: guests.Get(), Min: 1, Max: 8,
							OnChange: guests.Set, Label: "Guests",
						},
					},
					comps.Rating{
						Value:    float64(stars.Get()),
						OnChange: stars.Set,
						Label:    "Your rating",
					},
					caption(rated),
					core.Row(
						core.Gap(12),
						core.AlignItemsProp(core.AlignItemsCenter),
						comps.Button{
							Label:    loadLabel,
							Emphasis: comps.EmphasisOutlined,
							OnTap:    func() { loading.Set(!loading.Get()) },
						},
						comps.Spinner{Hidden: !loading.Get(), Label: "Loading the demo"},
					),
					comps.BottomBar{Items: items, Selected: tab.Get()},
					caption("Showing: "+tabs[tab.Get()]),
				),
				keyPoints(
					"Stepper clamps and reports only real changes; the button at a bound is disabled, and Label names the group.",
					"Rating's interactive stars are named buttons; read-only stars are hidden and the group states the score.",
					"Spinner is turned by the platform through core.Spin; it holds no hooks, and Hidden keeps its place while stopping its frames.",
					"BottomBar is navigation with Selected >= 0 and a toolbar below zero; each cell takes an equal share of the width.",
					"Screen.Footer pins a bar outside the scroll region and grows the content to push it to the bottom edge.",
				),
			)
		},
	}
}

// lessonChoicesAndProgress is Tier B's in-page widgets: RadioGroup, the
// vertical form of a choice, StepIndicator, the header of a multi-screen flow,
// and Timeline, the record of what happened after it. The demo is one small
// checkout whose shipping step is the radio group, followed by the order's
// history, so the three widgets read as parts of one purchase. It is appended at the end of
// the chapter for the reason 4.15 was: lesson numbers already in deep links do
// not move.
func lessonChoicesAndProgress() Lesson {
	return Lesson{
		Title:   "Radio groups, steps & timelines",
		Summary: "RadioGroup shows every option at once, StepIndicator shows where a flow is, and Timeline shows what happened since.",
		Body: func(ctx *core.Context) core.View {
			ship := core.NewState(ctx, "std")
			step := core.NewState(ctx, 0)
			steps := []string{"Account", "Shipping", "Payment", "Review"}
			stage := core.NewState(ctx, 1)

			// The order's history so far: the first stage+1 of these, the
			// delivered event in Success once it is reached.
			history := []comps.TimelineEvent{
				{Time: "09:12", Title: "Order placed", Subtitle: "4 items"},
				{Time: "11:40", Title: "Packed", Subtitle: "Warehouse 3"},
				{Time: "14:05", Title: "Out for delivery", Subtitle: "Driver: Sam"},
				{Time: "16:30", Title: "Delivered", Subtitle: "Left at the front door", Variant: comps.VariantSuccess},
			}
			shown := history[:stage.Get()+1]

			options := []comps.RadioOption{
				{Value: "std", Label: "Standard", Subtitle: "3–5 days · free"},
				{Value: "exp", Label: "Express", Subtitle: "Next day · $9"},
				{Value: "pick", Label: "Pick up in store", Subtitle: "Unavailable at this address", Disabled: true},
			}
			chosen := ""
			for _, o := range options {
				if o.Value == ship.Get() {
					chosen = o.Label
				}
			}

			return core.Column(
				core.Gap(14),
				prose("Three widgets pick one value. core.Select hides the options until it is "+
					"opened. SegmentedControl lays two to four short labels side by side. "+
					"RadioGroup lists every option vertically, with a subtitle each, for the "+
					"choice a user should compare before making."),
				codeBlock(`comps.RadioGroup{
    Label: "Shipping",
    Options: []comps.RadioOption{
        {Value: "std", Label: "Standard", Subtitle: "3–5 days"},
        {Value: "exp", Label: "Express", Subtitle: "Next day"},
    },
    Value:    ship.Get(),
    OnChange: ship.Set,
}`),
				prose("The whole row is the target, and the ring is drawn rather than a platform "+
					"control, so one tap reaches Go once on every target. OnChange fires only "+
					"for a different option, so it can be a plain setter."),
				prose("The group is a radiogroup and each row a radio, so a reader hears \"radio "+
					"button, checked\". In the browser the group is one tab stop, sitting on the "+
					"checked radio, and the arrow keys move the check itself — OnChange fires as "+
					"you arrow, which is why it is a setter."),
				prose("A StepIndicator is the header of a flow: done steps ticked, the current "+
					"step filled, the rest outlined. Only done steps are tappable, so going "+
					"back is free and skipping ahead is not something you have to guard. A "+
					"long flow scrolls sideways rather than guessing a width to collapse at."),
				codeBlock(`comps.StepIndicator{
    Steps:   []string{"Account", "Shipping", "Payment", "Review"},
    Current: step.Get(),
    OnTap:   step.Set,   // done steps only
}`),
				demoPanel("Step through the checkout with Next, then tap a ticked step to go back. The store pickup option is disabled.",
					comps.StepIndicator{
						Steps:   steps,
						Current: step.Get(),
						OnTap:   step.Set,
						Label:   "Checkout",
					},
					core.If(steps[step.Get()] == "Shipping", comps.RadioGroup{
						Label:    "Shipping",
						Options:  options,
						Value:    ship.Get(),
						OnChange: ship.Set,
					}),
					caption(fmt.Sprintf("On %s · shipping: %s", steps[step.Get()], chosen)),
					core.Row(
						core.Gap(8),
						comps.Button{
							Label:    "Back",
							Emphasis: comps.EmphasisOutlined,
							Disabled: step.Get() == 0,
							OnTap:    func() { step.Set(step.Get() - 1) },
						},
						comps.Button{
							Label:    "Next",
							Disabled: step.Get() == len(steps)-1,
							OnTap:    func() { step.Set(step.Get() + 1) },
						},
					),
				),
				prose("A Timeline is the record after the flow: events down a line, a dot per "+
					"event. No renderer draws a line across siblings, so each row draws its own "+
					"piece — a stretched rail whose bottom segment grows through the space "+
					"below the event and meets the next row's top segment."),
				codeBlock(`comps.Timeline{
    Label: "Order history",
    Events: []comps.TimelineEvent{
        {Time: "09:12", Title: "Order placed"},
        {Time: "11:40", Title: "Packed", Subtitle: "Warehouse 3"},
    },
}`),
				demoPanel("Advance the order and watch the line extend to each new event.",
					comps.Timeline{Label: "Order history", Events: shown},
					comps.Button{
						Label:    "Advance the order",
						Emphasis: comps.EmphasisOutlined,
						Disabled: stage.Get() == len(history)-1,
						OnTap:    func() { stage.Set(stage.Get() + 1) },
					},
				),
				keyPoints(
					"RadioGroup is the vertical, every-option-visible choice; Select is compact and SegmentedControl is horizontal.",
					"Each row is the tap target and the ring is drawn, so there is no second control to double-dispatch.",
					"OnChange fires only for a different, enabled option.",
					"It is a labelled radiogroup of radios; in the browser the arrows move the check, so OnChange fires as a user arrows through.",
					"StepIndicator ticks done steps, fills the current one and makes only done steps tappable.",
					"Its strip is named \"Step 2 of 4: Shipping\": navigation when OnTap is set, a group when it is not, and it scrolls sideways when the flow is long.",
					"Timeline draws the line per row: a stretched rail whose bottom segment grows through the event's spacing to meet the next row's.",
					"Timeline is a list of list items; the rail is hidden, and Variant colours an event's dot.",
				),
			)
		},
	}
}

// --- 4.17 -----------------------------------------------------------------

// menuNote is one row of 4.17's list. Age is days since the note was edited,
// which is what the Newest and Oldest orders sort on.
type menuNote struct {
	ID, Title string
	Age       int
}

// menuSorts are 4.17's picker entries in menu order: the value the order
// state holds, and the label the item and the trigger show.
var menuSorts = []struct{ Value, Label string }{
	{"newest", "Newest"},
	{"oldest", "Oldest"},
	{"title", "Title"},
}

// searchCountries is 4.17's SearchableSelect list: long enough that typing
// beats scrolling, grouped by continent so the Group subtitle shows, and one
// disabled entry at the end.
var searchCountries = []core.SelectOption{
	{Value: "ar", Label: "Argentina", Group: "South America"},
	{Value: "au", Label: "Australia", Group: "Oceania"},
	{Value: "br", Label: "Brazil", Group: "South America"},
	{Value: "ca", Label: "Canada", Group: "North America"},
	{Value: "eg", Label: "Egypt", Group: "Africa"},
	{Value: "fr", Label: "France", Group: "Europe"},
	{Value: "de", Label: "Germany", Group: "Europe"},
	{Value: "gh", Label: "Ghana", Group: "Africa"},
	{Value: "in", Label: "India", Group: "Asia"},
	{Value: "jp", Label: "Japan", Group: "Asia"},
	{Value: "ke", Label: "Kenya", Group: "Africa"},
	{Value: "mx", Label: "Mexico", Group: "North America"},
	{Value: "no", Label: "Norway", Group: "Europe"},
	{Value: "pt", Label: "Portugal", Group: "Europe"},
	{Value: "es", Label: "Spain", Group: "Europe"},
	{Value: "aq", Label: "Antarctica", Group: "No deliveries", Disabled: true},
}

// lessonMenus teaches the two ways to pick from a list that is not all on
// screen. The first demo puts the two menus an app reaches for first on one
// screen: a "⋯" on every row of a list, and a Sort picker above it. One state
// names which menu is open, so deleting a row takes nothing with it but its
// own entry. That is the argument for Menu being controlled, made where a
// reader can watch it hold. The second demo is a SearchableSelect followed by
// a City field in one focus order, so the return key's Next and the list's
// place outside that order can both be tried.
func lessonMenus() Lesson {
	return Lesson{
		Title:   "Menus & searchable selects",
		Summary: "comps.Menu opens an action sheet from a button; comps.SearchableSelect filters a long list under a search field.",
		Body: func(ctx *core.Context) core.View {
			// Which menu is open: a note's ID, "sort", or "" for none. One
			// state serves every menu on the screen.
			open := core.NewState(ctx, "")
			order := core.NewState(ctx, "newest")
			pinned := core.NewState(ctx, "")
			deleted := core.NewState(ctx, []string{})
			last := core.NewState(ctx, "")

			// The SearchableSelect demo: the chosen value, the field's text,
			// and a City field after it in one return-key order.
			country := core.NewState(ctx, "")
			countryQuery := core.NewState(ctx, "")
			city := core.NewState(ctx, "")
			countryRef := core.UseFocusRef(ctx)
			cityRef := core.UseFocusRef(ctx)
			core.UseFocusOrder(ctx, countryRef, cityRef)
			shipsTo := "nothing chosen yet"
			for _, o := range searchCountries {
				if o.Value == country.Get() && o.Value != "" {
					shipsTo = o.Label
				}
			}

			all := []menuNote{
				{ID: "groceries", Title: "Groceries", Age: 3},
				{ID: "trip", Title: "Trip ideas", Age: 1},
				{ID: "books", Title: "Books to read", Age: 7},
			}
			notes := make([]menuNote, 0, len(all))
			for _, n := range all {
				if !slices.Contains(deleted.Get(), n.ID) {
					notes = append(notes, n)
				}
			}
			slices.SortFunc(notes, func(a, b menuNote) int {
				// The pinned note leads whatever the order; the chosen
				// order sorts the rest.
				if ap, bp := a.ID == pinned.Get(), b.ID == pinned.Get(); ap != bp {
					if ap {
						return -1
					}
					return 1
				}
				switch order.Get() {
				case "oldest":
					return b.Age - a.Age
				case "title":
					return strings.Compare(a.Title, b.Title)
				}
				return a.Age - b.Age
			})

			closeMenu := func() { open.Set("") }

			sortLabel := ""
			sortItems := make([]comps.SheetAction, 0, len(menuSorts))
			for _, s := range menuSorts {
				if s.Value == order.Get() {
					sortLabel = s.Label
				}
				sortItems = append(sortItems, comps.SheetAction{
					Label:   s.Label,
					Checked: s.Value == order.Get(),
					OnTap: func() {
						order.Set(s.Value)
						last.Set("sorted by " + strings.ToLower(s.Label))
					},
				})
			}

			rows := make([]core.PropsAndChildren, 0, len(notes)+2)
			// No inset and no gap: each ListRow carries the theme's row
			// padding already.
			rows = append(rows, core.Padding(0), core.Gap(0))
			for _, n := range notes {
				pinLabel, pinTo, subtitle := "Pin to top", n.ID, menuAge(n.Age)
				if n.ID == pinned.Get() {
					pinLabel, pinTo, subtitle = "Unpin", "", "Pinned · "+subtitle
				}
				rows = append(rows, comps.ListRow{
					Title:    n.Title,
					Subtitle: subtitle,
					Trailing: comps.Menu{
						Trigger: comps.Button{
							Label:              "⋯",
							AccessibilityLabel: "Actions for " + n.Title,
							Emphasis:           comps.EmphasisGhost,
						},
						Open:      open.Get() == n.ID,
						OnOpen:    func() { open.Set(n.ID) },
						OnDismiss: closeMenu,
						Title:     n.Title,
						Items: []comps.SheetAction{
							{Label: pinLabel, OnTap: func() {
								pinned.Set(pinTo)
								last.Set(strings.ToLower(pinLabel) + " " + n.Title)
							}},
							{Label: "Delete", Variant: comps.VariantError, OnTap: func() {
								// A fresh slice: the state's value is shared
								// with the render that read it.
								deleted.Set(append(slices.Clone(deleted.Get()), n.ID))
								if pinned.Get() == n.ID {
									pinned.Set("")
								}
								last.Set("deleted " + n.Title)
							}},
						},
						Cancel: "Cancel",
						Style:  []core.StyleProp{core.MaxWidth("520px")},
					},
				})
			}

			return core.Column(
				core.Gap(14),
				prose("A menu is a button that opens a short list, and the tap that picks is the "+
					"tap that closes it. No host can place a popover under its button from Go, "+
					"because none sends Go the button's position, so comps.Menu opens the action "+
					"sheet from lesson 6.7: the same bottom-edge panel, with a trigger attached."),
				codeBlock(`comps.Menu{
    Trigger:   comps.Button{Label: "⋯", AccessibilityLabel: "Note actions"},
    Open:      open.Get() == note.ID,
    OnOpen:    func() { open.Set(note.ID) },
    OnDismiss: func() { open.Set("") },
    Title:     note.Title,
    Items: []comps.SheetAction{
        {Label: "Pin to top", OnTap: pin},
        {Label: "Delete", Variant: comps.VariantError, OnTap: del},
    },
    Cancel: "Cancel",
}`),
				prose("Trigger is a comps.Button template: its label, emphasis and names are used, "+
					"and its OnTap is replaced by OnOpen. A widget cannot attach a tap to a View "+
					"you built, which is why the slot is a Button and not any View."),
				prose("Open is your state, not the widget's. Every ⋯ below reads one state holding "+
					"which note's menu is open, so deleting a row moves nothing. A menu that kept "+
					"its own flag in a hook would take a slot per row, and those slots drift as "+
					"soon as the row count changes."),
				prose("A picker is the same widget with a checked item. Checked leads the label "+
					"with ✓ and marks the item as the current one (core.CurrentTrue). Each item sets the value in "+
					"its own OnTap, and the trigger's label shows the current choice."),
				codeBlock(`comps.SheetAction{
    Label:   "Newest",
    Checked: order.Get() == "newest",
    OnTap:   func() { order.Set("newest") },
}`),
				demoPanel("Open a note's ⋯ to pin or delete it, and change the order with Sort.",
					comps.Menu{
						Trigger: comps.Button{
							Label:              "Sort: " + sortLabel + " ▾",
							AccessibilityLabel: "Sort: " + sortLabel,
							Emphasis:           comps.EmphasisOutlined,
						},
						Open:      open.Get() == "sort",
						OnOpen:    func() { open.Set("sort") },
						OnDismiss: closeMenu,
						Title:     "Sort by",
						Items:     sortItems,
						Cancel:    "Cancel",
						Style:     []core.StyleProp{core.MaxWidth("520px")},
					},
					core.Column(rows...),
					core.If(len(notes) == 0, caption("Every note is deleted.")),
					core.If(len(deleted.Get()) > 0, comps.Button{
						Label:    "Restore deleted notes",
						Emphasis: comps.EmphasisOutlined,
						OnTap: func() {
							deleted.Set([]string{})
							last.Set("restored the notes")
						},
					}),
					core.If(last.Get() != "", caption("✓ "+last.Get())),
				),
				prose("A menu suits a handful of choices. For a list too long to scroll, such as "+
					"countries, comps.SearchableSelect filters as you type: a SearchField with "+
					"the matching options listed under it. Query is the field's text and Value "+
					"is the choice, and both are your state."),
				codeBlock(`comps.SearchableSelect{
    Label:         "Country",
    Options:       countries,   // []core.SelectOption
    Value:         country.Get(),
    OnChange:      country.Set,
    Query:         query.Get(),
    OnQueryChange: query.Set,
    FocusRef:      countryRef,
}`),
				prose("Picking an option reports its Value, writes its label into the field and "+
					"puts the keyboard away. The list shows while the text is not the chosen "+
					"label, so the pick closes it and editing the text opens it again. Clear "+
					"empties the text and the choice together."),
				prose("The list never takes focus, so typing carries on while it changes. The "+
					"return key belongs to the form: with FocusRef in core.UseFocusOrder the "+
					"keyboard shows Next and moves to the City field, past the list. In a "+
					"browser the field is an ARIA combobox: the arrows move through the "+
					"matches while the caret stays in the field, Enter picks, and focus is "+
					"still in the field afterwards."),
				demoPanel("Type \"an\" and pick a country, then try Clear. Antarctica is disabled.",
					comps.SearchableSelect{
						Label:         "Country",
						Placeholder:   "Search countries",
						Options:       searchCountries,
						Value:         country.Get(),
						OnChange:      country.Set,
						Query:         countryQuery.Get(),
						OnQueryChange: countryQuery.Set,
						MaxResults:    5,
						FocusRef:      countryRef,
					},
					core.Input(city.Get(), "City", city.Set,
						core.FocusTarget(cityRef),
						core.AccessibilityLabel("City"),
					),
					caption("Ships to: "+shipsTo),
				),
				keyPoints(
					"Menu is a Button that opens an ActionSheet; picking an item runs its OnTap, then OnDismiss.",
					"No popover is anchored to the trigger: no host sends Go its position, so the list is the bottom-edge sheet on every target.",
					"Trigger is a Button template: its label, emphasis and names apply, and OnOpen replaces its OnTap.",
					"Open is controlled, so one state can say which row's menu is open, and a list of menus holds no hook slots.",
					"A picker is a menu whose current item is Checked: a leading ✓ and core.CurrentTrue.",
					"The trigger says it opens a dialog (aria-haspopup) and states no expanded state: a control that opens a dialog is not a disclosure.",
					"SearchableSelect lists matches while the field's text is not the chosen label; a pick reports Value, writes the label and closes the list.",
					"The list never takes focus and has no field of its own, so the return key's Next skips it and moves on through the form.",
					"The field is a combobox controlling a labelled listbox, and a status line says how many matched, since the list appears silently under the field.",
				),
			)
		},
	}
}

// menuAge is 4.17's row subtitle, kept out of the body so the one plural
// branch is read once.
func menuAge(days int) string {
	if days == 1 {
		return "Edited 1 day ago"
	}
	return fmt.Sprintf("Edited %d days ago", days)
}

// --- 4.18 Drawers ------------------------------------------------------------

// drawerSections are the 4.18 demo's destinations: the drawer's rows, the
// app bar's title and the body caption all read from one table, so the three
// cannot disagree about which section is showing.
var drawerSections = []struct {
	Icon, Label, Subtitle string
}{
	{"📥", "Inbox", "12 notes"},
	{"⭐", "Starred", "3 notes"},
	{"🗄", "Archive", "40 notes"},
	{"🗑", "Trash", "empty"},
}

// lessonDrawers teaches comps.Drawer on a small pretend screen: an app bar with
// a ☰, a body naming the current section, and the drawer over both. The demo
// pins the drawer's height because a lesson scrolls, which is the one sizing
// rule a caller has to know, and it wires the focus handoff both ways so the
// keyboard path can be tried in a browser: ☰ focuses the ✕, dismissing
// focuses the ☰. It is appended at the end of the chapter for the reason 4.15
// was: lesson numbers already in deep links do not move.
func lessonDrawers() Lesson {
	return Lesson{
		Title:   "Drawers",
		Summary: "comps.Drawer pins a panel of destinations to the leading edge over the screen, opened by a ☰ and closed by a pick.",
		Body: func(ctx *core.Context) core.View {
			// Hooks first and unconditionally, as in every lesson.
			open := core.NewState(ctx, false)
			section := core.NewState(ctx, 0)
			closeRef := core.UseFocusRef(ctx)
			menuRef := core.UseFocusRef(ctx)

			t := ctx.Theme()
			current := drawerSections[section.Get()]

			items := make([]comps.DrawerItem, len(drawerSections))
			for i, s := range drawerSections {
				items[i] = comps.DrawerItem{
					Icon:     s.Icon,
					Label:    s.Label,
					Subtitle: s.Subtitle,
					OnTap:    func() { section.Set(i) },
				}
			}

			// The pretend screen under the drawer. Its own fill so the scrim
			// visibly dims something, and no inset so the app bar meets the
			// edges the way it would at a real screen's top.
			screen := core.Column(
				core.Padding(0),
				core.Gap(0),
				core.Width("100%"),
				core.Height("100%"),
				core.BackgroundColor(t.Colors.Surface),
				comps.AppBar{
					Title: current.Label,
					Leading: comps.Button{
						Label:              "☰",
						AccessibilityLabel: "Open navigation",
						Emphasis:           comps.EmphasisGhost,
						FocusRef:           menuRef,
						OnTap: func() {
							open.Set(true)
							// The ☰ is inside the layer the drawer hides, so
							// focus moves to the panel's ✕ with it.
							core.Focus(closeRef)
						},
					},
				},
				core.Column(
					core.Gap(8),
					caption("Showing "+current.Label),
					prose(current.Subtitle+" in this section."),
				),
			)

			return core.Column(
				core.Gap(14),
				prose("A Drawer is side navigation: a panel of destinations pinned to the "+
					"leading edge over the screen. It is a ZStack of two layers, the screen "+
					"and the panel, not a Modal. A Modal is a centred Dialog window on Android "+
					"and a bottom sheet on iOS, so only the web could put it at an edge; a "+
					"ZStack layer fills its box on all four targets."),
				codeBlock(`comps.Drawer{
    Open:      open.Get(),
    OnDismiss: func() { open.Set(false); core.Focus(menuRef) },
    Title:     "Notebook",
    Width:     "75%",            // default 280px
    Items:     items,            // []comps.DrawerItem{Icon, Label, Subtitle, OnTap}
    Selected:  section.Get(),
    CloseRef:  closeRef,
    Content:   screen,           // its AppBar's ☰ opens and focuses closeRef
    Style:     []core.StyleProp{core.Height("340px")},
}`),
				prose("Picking a destination runs its OnTap and then OnDismiss, so no handler "+
					"closes the drawer itself. While it is open the screen layer is hidden "+
					"from assistive technology, which keeps a screen reader inside the panel."),
				prose("A layer has no dialog to move focus into, so the handoff is yours: the "+
					"☰ focuses the ✕ through CloseRef, and OnDismiss focuses the ☰ through "+
					"Button.FocusRef. The drawer covers its own box, so inside a scrolling "+
					"page like this one it needs a pinned height."),
				demoPanel("Open the drawer, pick a section, then open it again and close it with ✕ or the dimmed area.",
					comps.Drawer{
						Open: open.Get(),
						OnDismiss: func() {
							open.Set(false)
							core.Focus(menuRef)
						},
						Title: "Notebook",
						// A share rather than the 280px default: the demo
						// panel is narrower than a phone, and the scrim has to
						// stay wide enough to see and to tap.
						Width:    "75%",
						Items:    items,
						Selected: section.Get(),
						CloseRef: closeRef,
						Content:  screen,
						Style:    []core.StyleProp{core.Height("340px")},
					},
				),
				keyPoints(
					"Drawer is a ZStack layer over Content, so it sits at the leading edge on every target.",
					"Picking a row calls its OnTap, then OnDismiss; the ✕ and the scrim call OnDismiss.",
					"While open, Content is hidden from assistive technology; CloseRef and Button.FocusRef move the keyboard focus.",
					"The drawer covers its ZStack: pin a height when it sits in a scrolling column.",
					"Android back and browser back close an open drawer through core.OnBack; on the web the screen behind an open drawer is inert, so Tab stays in the panel.",
				),
			)
		},
	}
}

// --- 4.19 Clocks, drawing and alarms ----------------------------------------

// tutorialSeries is the 4.19 chart's data: twelve points, rotated by the
// Shift button so every press changes every shape's path without the demo
// needing a random source a test would have to pin.
var tutorialSeries = []float64{12, 18, 15, 26, 22, 31, 28, 36, 33, 41, 38, 45}

// tutorialShares are the donut's slices, with the palette roles that colour
// them read from the theme at render time.
var tutorialShares = []float64{45, 30, 25}

func lessonClocksAndDrawing() Lesson {
	return Lesson{
		Title:   "Clocks, drawing and alarms",
		Summary: "hooks.UseNow, comps.AnalogClock and DigitalClock, core.Canvas, and an in-app alarm with hooks.UseAlarms.",
		Body: func(ctx *core.Context) core.View {
			t := ctx.Theme()

			// Hooks first and unconditionally, in a fixed order.
			now := hooks.UseNow(ctx, time.Second)
			hour24 := core.NewState(ctx, false)
			shift := core.NewState(ctx, 0)
			alarms := core.NewState(ctx, []alarm.Alarm{
				{ID: "wake", Hour: 6, Minute: 30, Label: "Wake up", Days: alarm.Weekdays},
				{ID: "run", Hour: 7, Minute: 15, Label: "Run", Days: alarm.Weekend},
			})
			added := core.NewState(ctx, 0)

			setEnabled := func(id string, on bool) {
				next := slices.Clone(alarms.Get())
				for i := range next {
					if next[i].ID == id {
						next[i].Enabled = on
					}
				}
				alarms.Set(next)
			}

			ringer := hooks.UseAlarms(ctx, alarms.Get(), hooks.AlarmOptions{
				Haptics: true,
				// Off screen, the OS rings instead: each alarm becomes a
				// scheduled notification when the app leaves the foreground,
				// and is taken back when it returns.
				Notify: true,
				// A short snooze, so a reader trying it waits a minute rather
				// than nine.
				Snooze: time.Minute,
				// A one-time alarm switches itself off once it has rung; the
				// hook reports the ring and leaves the list to the app.
				OnRing: func(a alarm.Alarm) {
					if a.Once() {
						setEnabled(a.ID, false)
					}
				},
			})
			// Checks, never prompts; the prompt is the button's, from a tap.
			// Live, like the exact-alarm check below: a user who allows
			// notifications from the Settings app rather than the dialog comes
			// back to a screen that has already dropped the button.
			notifyStatus := hooks.UsePermissionLive(ctx, permission.Notifications)
			// Android's second switch: without "Alarms & reminders" a banner is
			// scheduled inexactly and can come a minute late. Only Denied shows
			// the button — iOS and the browser answer Granted, and the moment
			// before any answer (Unknown) should not flash a button that then
			// vanishes. The Live variant because the grant happens on a
			// Settings page: it re-checks when the app comes back to the
			// foreground, which a mount-only check would never see.
			exactStatus := hooks.UsePermissionLive(ctx, permission.ExactAlarms)

			// The chart: a line over an area, both built from the same points.
			series := slices.Clone(tutorialSeries)
			k := shift.Get() % len(series)
			series = append(series[k:], series[:k]...)
			const w, h = 110.0, 50.0
			line := core.NewPath()
			for i, v := range series {
				line.LineTo(float64(i)*w/float64(len(series)-1), h-v)
			}
			area := core.NewPath()
			for i, v := range series {
				area.LineTo(float64(i)*w/float64(len(series)-1), h-v)
			}
			area.LineTo(w, h).LineTo(0, h).Close()

			// The donut: sectors from twelve o'clock (-90°), a sliver of gap
			// between each, in the theme's first three series colours. Those
			// are the categorical Chart role, not Success and Warning, which
			// would say "fine" and "careful" about slices that mean neither.
			// Indexed round the list, because a theme may state fewer
			// colours than the demos below reach for.
			chartColors := t.Colors.ChartColors()
			hue := func(i int) string { return chartColors[i%len(chartColors)] }
			slices3 := []string{hue(0), hue(1), hue(2)}
			var donut []core.Shape
			start := -90.0
			for i, share := range tutorialShares {
				sweep := share / 100 * 360
				donut = append(donut, core.Shape{
					Path: core.Sector(50, 50, 30, 48, start+1, sweep-2),
					Fill: slices3[i],
				})
				start += sweep
			}

			var ringPanel core.View = caption("No alarm is ringing.")
			if a, ok := ringer.Ringing(); ok {
				ringPanel = comps.AlarmRinging{Alarm: a, Hour24: hour24.Get(), OnSnooze: ringer.Snooze, OnDismiss: ringer.Dismiss}
			} else if a, at, ok := ringer.Snoozed(); ok {
				ringPanel = caption(fmt.Sprintf("%s is snoozed until %s.", a.TimeLabel(hour24.Get()), at.Format("15:04:05")))
			}

			rows := make([]core.PropsAndChildren, 0, len(alarms.Get()))
			for _, a := range alarms.Get() {
				id := a.ID
				rows = append(rows, core.Keyed(id, comps.AlarmRow{
					Alarm:    a,
					Hour24:   hour24.Get(),
					OnToggle: func(on bool) { setEnabled(id, on) },
				}))
			}

			return core.Column(
				core.Gap(14),
				prose("A clock widget draws a time it is handed and never ticks by itself. hooks.UseNow is "+
					"the ticker, and it wakes on the wall clock's second boundary rather than a second "+
					"after the screen mounted, so the seconds change when the status bar's do."),
				codeBlock(`now := hooks.UseNow(ctx, time.Second)
comps.AnalogClock{Time: now, ShowSeconds: true, Numerals: true, Smooth: true}
comps.DigitalClock{Time: now, ShowSeconds: true, ShowDate: true}`),
				demoPanel("The live time, twice.",
					comps.SwitchRow{Title: "24-hour clock", On: hour24.Get(), OnToggle: hour24.Set},
					core.Row(
						core.Gap(20),
						core.AlignItemsProp(core.AlignItemsCenter),
						core.Justify(core.JustifyCenter),
						comps.AnalogClock{Time: now, Size: 150, ShowSeconds: true, Numerals: true, Smooth: true},
						comps.DigitalClock{Time: now, Hour24: hour24.Get(), ShowSeconds: true, ShowDate: true, Size: 30},
					),
				),
				prose("The analog face has no drawing in it: every hand and tick is a layer the size of the "+
					"dial, turned with core.Rotate. A layer turns about its own centre, which is the dial's "+
					"centre, so a stick in its top half sweeps round like a hand. Smooth adds a short "+
					"Transition, and the angles never wrap, so the second hand does not spin back at the "+
					"top of each minute."),
				prose("Charts need what a box cannot do: a line between two arbitrary points. core.Canvas "+
					"is that primitive. Shapes are written in a coordinate space of your choosing and "+
					"scaled into the box; stroke widths stay in pixels however the drawing is stretched."),
				codeBlock(`core.Canvas(110, 50, []core.Shape{
    {Path: area, Fill: primary + "33"},
    {Path: line, Stroke: primary, StrokeWidth: 2, Cap: core.CapRound, Join: core.JoinRound},
}, core.CanvasStretch, core.Height("120px"))`),
				demoPanel("Shift the data. Each shape is one node, so a change patches its path in place.",
					comps.Button{Label: "Shift", OnTap: func() { shift.Set(shift.Get() + 1) }},
					core.Canvas(w, h, []core.Shape{
						{Path: area, Fill: t.Colors.Primary + "33"},
						{Path: line, Stroke: t.Colors.Primary, StrokeWidth: 2, Cap: core.CapRound, Join: core.JoinRound},
						{Path: core.Line(0, h, w, h), Stroke: t.Colors.BorderColor()},
					}, core.CanvasStretch, core.Height("120px"),
						core.AccessibilityLabel(fmt.Sprintf("Line chart rising from %.0f to %.0f", series[0], series[len(series)-1]))),
					core.Row(
						core.Gap(16),
						core.AlignItemsProp(core.AlignItemsCenter),
						core.Canvas(100, 100, donut, core.Width("110px"),
							core.AccessibilityLabel("Donut chart: 45%, 30% and 25%")),
						caption("core.Sector(cx, cy, inner, outer, start, sweep) — a pie wedge when inner is 0."),
					),
					core.Row(
						core.Gap(16),
						core.AlignItemsProp(core.AlignItemsCenter),
						// Two circles drawn the same way round. Nonzero counts
						// the inner one as more inside, so the ring is solid;
						// even-odd counts it as a second crossing, so it is a
						// hole.
						core.Canvas(100, 50, []core.Shape{
							{Path: core.NewPath().Arc(25, 25, 22, 0, 360).Close().Arc(25, 25, 12, 0, 360).Close(),
								Fill: t.Colors.Primary, FillRule: core.FillEvenOdd},
							{Path: core.NewPath().Arc(75, 25, 22, 0, 360).Close().Arc(75, 25, 12, 0, 360).Close(),
								Fill: t.Colors.Primary},
						}, core.Width("110px"),
							core.AccessibilityLabel("A ring with a hole beside a solid disc")),
						caption("Two circles, one path: FillEvenOdd cuts the inner one out; the default rule does not."),
					),
					core.Row(
						core.Gap(16),
						core.AlignItemsProp(core.AlignItemsCenter),
						// A radial highlight off-centre on a disc, and a
						// diagonal linear fade across the theme's first three
						// series colours. Gradient geometry is in viewBox
						// units, the same space as the paths it fills.
						core.Canvas(100, 50, []core.Shape{
							{Path: core.Circle(25, 25, 22), FillGradient: core.RadialGradientFill(18, 18, 30,
								core.Stop(0, "#FFFFFF"), core.Stop(0.35, hue(0)), core.Stop(1, hue(6)))},
							{Path: core.Rect(52, 3, 46, 44), FillGradient: core.LinearGradientFill(52, 3, 98, 47,
								core.Stop(0, hue(0)), core.Stop(0.5, hue(1)), core.Stop(1, hue(2)))},
						}, core.Width("110px"),
							core.AccessibilityLabel("A shaded sphere beside a square fading blue to orange to green")),
						caption("FillGradient: RadialGradientFill and LinearGradientFill, in the drawing's own units."),
					),
					core.Row(
						core.Gap(16),
						core.AlignItemsProp(core.AlignItemsCenter),
						// The same gradients on strokes. A dashed zigzag
						// fades left to right, so the colour follows x in
						// viewBox units while the dashes and width stay in
						// layout units; a ring's outline takes a radial
						// gradient whose centre is off the ring, so its
						// colour turns from one side to the other.
						core.Canvas(100, 50, []core.Shape{
							{Path: core.NewPath().MoveTo(4, 40).LineTo(16, 10).LineTo(28, 40).LineTo(40, 10).LineTo(50, 36),
								StrokeWidth: 3, Cap: core.CapRound, Join: core.JoinRound, Dash: []float64{7, 4},
								StrokeGradient: core.LinearGradientFill(4, 0, 50, 0, core.Stop(0, hue(0)), core.Stop(1, hue(2)))},
							{Path: core.Circle(76, 25, 19), StrokeWidth: 4,
								StrokeGradient: core.RadialGradientFill(58, 8, 44, core.Stop(0, hue(1)), core.Stop(1, hue(7)))},
						}, core.Width("110px"),
							core.AccessibilityLabel("A dashed zigzag fading blue to aqua beside a ring shading orange to red")),
						caption("StrokeGradient: the same constructors on an outline; the width stays in layout units."),
					),
				),
				prose("An alarm is the time arithmetic in package alarm, a hook that checks it every second, "+
					"and two widgets. The hook asks whether each alarm fell due since the last check, not "+
					"whether it is due now, so a late tick cannot miss one. On screen it rings in the app. "+
					"With Notify, leaving the screen hands every upcoming alarm to the operating system as "+
					"a scheduled notification, so it still goes off with the app closed, and coming back "+
					"takes them back."),
				codeBlock(`ringer := hooks.UseAlarms(ctx, alarms.Get(), hooks.AlarmOptions{Haptics: true, Notify: true})
if a, ok := ringer.Ringing(); ok {
    return comps.AlarmRinging{Alarm: a, OnSnooze: ringer.Snooze, OnDismiss: ringer.Dismiss}
}`),
				demoPanel("Set one, then wait on this screen.",
					comps.Button{Label: "Ring at the next minute", OnTap: func() {
						added.Set(added.Get() + 1)
						at := time.Now().Truncate(time.Minute).Add(time.Minute)
						alarms.Set(append(slices.Clone(alarms.Get()), alarm.Alarm{
							ID:      fmt.Sprintf("soon-%d", added.Get()),
							Hour:    at.Hour(),
							Minute:  at.Minute(),
							Label:   "Try it",
							Enabled: true,
						}))
					}},
					core.Column(append([]core.PropsAndChildren{core.Gap(0)}, rows...)...),
					ringPanel,
					core.If(notifyStatus != permission.Granted,
						core.Row(
							core.Gap(12),
							core.AlignItemsProp(core.AlignItemsCenter),
							comps.Button{
								Label:    "Allow notifications",
								Emphasis: comps.EmphasisOutlined,
								OnTap:    func() { permission.Request(permission.Notifications) },
							},
							caption("Needed to ring with the app closed."),
						)),
					core.If(exactStatus == permission.Denied,
						core.Row(
							core.Gap(12),
							core.AlignItemsProp(core.AlignItemsCenter),
							comps.Button{
								Label:    "Allow exact alarms",
								Emphasis: comps.EmphasisOutlined,
								OnTap:    func() { permission.Request(permission.ExactAlarms) },
							},
							caption("Without it Android may ring a minute late."),
						)),
				),
				keyPoints(
					"Clock widgets take a time.Time; hooks.UseNow ticks on the wall clock's boundary.",
					"AnalogClock is built from rotated full-dial layers, so it needs no drawing primitive.",
					"core.Canvas maps a viewBox onto its box; CanvasFit keeps the shape, CanvasStretch fills the box.",
					"Paths flatten to move, line, cubic and close in Go, so every target draws the same curve.",
					"hooks.UseAlarms rings alarms that fell due since its last check; with Notify the OS rings them while the app is away.",
					"Switch one-time alarms off in OnRing, which also hears about alarms the OS rang while the app was away.",
				),
			)
		},
	}
}

// --- 4.20 Charts ------------------------------------------------------------

// chartMonths and the three series below are 4.20's data. Fixed rather than
// random so the lesson's tests can assert on what the charts say; the Shift
// button rotates them, which changes every path and every summary.
var (
	chartMonths  = []string{"Jan", "Feb", "Mar", "Apr", "May", "Jun", "Jul", "Aug", "Sep", "Oct", "Nov", "Dec"}
	chartVisits  = []float64{120, 180, 150, 260, 220, 310, 280, 360, 330, 410, 380, 450}
	chartSignups = []float64{40, 55, 60, 52, 80, 95, math.NaN(), 110, 105, 130, 140, 150}
	chartSteps   = []float64{8200, 10400, 6100, 12000, 9000, 4300, 7600}
)

// rotated returns xs turned left by k, leaving the input alone.
func rotated[T any](xs []T, k int) []T {
	if len(xs) == 0 {
		return xs
	}
	k %= len(xs)
	return append(slices.Clone(xs[k:]), xs[:k]...)
}

func lessonCharts() Lesson {
	return Lesson{
		Title:   "Charts",
		Summary: "comps.Sparkline, LineChart, AreaChart, BarChart, ScatterChart, DonutChart, PieChart and Gauge: charts drawn on core.Canvas with axes as Text.",
		Body: func(ctx *core.Context) core.View {
			shift := core.NewState(ctx, 0)
			battery := core.NewState(ctx, 72.0)

			k := shift.Get()
			visits := rotated(chartVisits, k)
			signups := rotated(chartSignups, k)
			steps := rotated(chartSteps, k)

			return core.Column(
				core.Gap(14),
				prose("The chart widgets are pure Go on top of core.Canvas. Scales and axis ticks are "+
					"computed in Go, rounded outwards to 1, 2 or 5 times a power of ten so the labels "+
					"read 0, 100, 200. Labels, legends and the value in a gauge are ordinary Text laid "+
					"out around the drawing, and colours come from the theme's roles."),
				codeBlock(`comps.LineChart{
    Subject: "Traffic",
    Labels:  months,
    Series: []comps.ChartSeries{
        {Name: "Visits", Values: visits},
        {Name: "Sign-ups", Values: signups}, // NaN leaves a gap
    },
    Points: true,
}`),
				demoPanel("Shift the data.",
					comps.Button{Label: "Shift", OnTap: func() { shift.Set(shift.Get() + 1) }},
					comps.LineChart{
						Subject: "Traffic",
						Labels:  rotated(chartMonths, k),
						Series: []comps.ChartSeries{
							{Name: "Visits", Values: visits},
							{Name: "Sign-ups", Values: signups},
						},
						Points: true,
					},
					comps.AreaChart{
						Subject: "Visits",
						Labels:  rotated(chartMonths, k),
						Series:  []comps.ChartSeries{{Name: "Visits", Values: visits}},
						Height:  110,
					},
				),
				prose("A bar chart gives each category an equal slot, so its labels sit under their bars "+
					"exactly. Its axis always includes zero: a bar's length only means something when "+
					"it starts there. A sparkline has no axes at all and spends its whole height on the "+
					"shape."),
				demoPanel("Bars and a sparkline.",
					comps.BarChart{
						Subject: "Steps this week",
						Labels:  rotated([]string{"Mon", "Tue", "Wed", "Thu", "Fri", "Sat", "Sun"}, k),
						Series:  []comps.ChartSeries{{Name: "Steps", Values: steps}},
						Height:  120,
					},
					core.Row(
						core.Gap(12),
						core.AlignItemsProp(core.AlignItemsCenter),
						core.Text(fmt.Sprintf("%.0f visits", visits[len(visits)-1]), core.FontWeight(core.Bold)),
						core.Column(core.FlexGrow(1), core.FlexBasis("0"),
							comps.Sparkline{Values: visits, Subject: "Visits by month", ShowLast: true, Area: true}),
					),
				),
				prose("The same widgets bend to other questions. Smooth draws a monotone curve, which "+
					"never overshoots a point, Stacked puts series on top of each other so the top line "+
					"is the whole, and Horizontal turns a bar chart on its side for names too long to "+
					"sit under a bar. A label longer than its slot is cut with an ellipsis rather than "+
					"pushing its neighbours along. A scatter plots two measures against each other."),
				codeBlock(`comps.AreaChart{Stacked: true, Smooth: true, Series: parts}
comps.BarChart{Horizontal: true, ShowValues: true, Labels: names, Series: spend}
comps.ScatterChart{Series: []comps.ScatterSeries{{Points: points}}}`),
				demoPanel("Stacks, sideways bars and a scatter.",
					comps.AreaChart{
						Subject: "Visits by source",
						Labels:  rotated(chartMonths, k),
						Stacked: true,
						Smooth:  true,
						Series: []comps.ChartSeries{
							{Name: "Search", Values: rotated([]float64{12, 18, 16, 22, 28, 26, 30, 34, 31, 36, 40, 44}, k)},
							{Name: "Direct", Values: rotated([]float64{8, 9, 12, 10, 11, 14, 13, 15, 18, 16, 17, 19}, k)},
							{Name: "Social", Values: rotated([]float64{3, 4, 6, 9, 7, 8, 12, 10, 9, 13, 15, 14}, k)},
						},
						Height: 120,
					},
					comps.BarChart{
						Subject:    "Monthly spend",
						Horizontal: true,
						ShowValues: true,
						Labels:     []string{"Rent", "Groceries and household", "Transport", "Subscriptions and memberships"},
						Series:     []comps.ChartSeries{{Name: "Spend", Values: rotated([]float64{1200, 450, 200, 85}, k)}},
						// Compact, because each tick label has two thirds of
						// an interval and a wider one is cut ("$15…").
						Format: func(v float64) string {
							if v >= 1000 {
								return fmt.Sprintf("$%.1fk", v/1000)
							}
							return fmt.Sprintf("$%.0f", v)
						},
					},
					comps.BarChart{
						Subject:    "Tickets by quarter",
						Labels:     []string{"First quarter", "Second quarter", "Third quarter", "Fourth quarter"},
						Stacked:    true,
						ShowValues: true,
						Series: []comps.ChartSeries{
							{Name: "Opened", Values: rotated([]float64{40, 52, 47, 60}, k)},
							{Name: "Reopened", Values: rotated([]float64{6, 9, 4, 8}, k)},
						},
						Height: 110,
					},
					comps.ScatterChart{
						Subject: "Distance and pace",
						Series: []comps.ScatterSeries{{Name: "Runs", Points: []comps.ChartPoint{
							{X: 3, Y: 5.2}, {X: 5, Y: 5.5}, {X: 8, Y: 5.9}, {X: 10, Y: 6.1}, {X: 4, Y: 5.0},
							{X: 12, Y: 6.4}, {X: 6, Y: 5.6}, {X: 15, Y: 6.8}, {X: 21, Y: 7.2}, {X: 7, Y: 5.4},
						}}},
						XFormat: func(v float64) string { return fmt.Sprintf("%.0f km", v) },
						Height:  110,
					},
				),
				prose("Donuts, pies and gauges keep their shape in any box. The donut's legend rounds its "+
					"percentages so they add up to 100, and a gauge's arc is a stroke, so its thickness "+
					"is in pixels at every size."),
				demoPanel("Drain the battery.",
					core.Row(
						core.Gap(16),
						core.FlexWrap(true),
						core.Justify(core.JustifyCenter),
						comps.DonutChart{
							Subject: "Budget",
							Size:    130,
							Slices: []comps.ChartSlice{
								{Label: "Rent", Value: 1200},
								{Label: "Food", Value: 450},
								{Label: "Transport", Value: 200},
								{Label: "Fun", Value: 150},
							},
							CenterValue: "$2,000",
							CenterLabel: "a month",
						},
						comps.Gauge{
							Value:     battery.Get(),
							Label:     "Battery",
							ValueText: fmt.Sprintf("%.0f%%", battery.Get()),
							Size:      130,
						},
					),
					comps.Button{Label: "Use 12%", OnTap: func() {
						next := battery.Get() - 12
						if next < 0 {
							next = 100
						}
						battery.Set(next)
					}},
				),
				keyPoints(
					"Every chart is one accessible element that speaks a summary; its labels are hidden.",
					"Axis ticks are nice numbers computed in Go, and labels are Text placed by flex weights.",
					"A NaN value is a gap in a line and a missing bar.",
					"Series colours come from theme roles; set ChartSeries.Color or Colors to choose your own.",
				),
			)
		},
	}
}

// --- 4.21 Foldables and window size -----------------------------------------

// foldNotes are the 4.21 list–detail demo's rows: a title for the list pane
// and a body for the detail pane.
var foldNotes = []struct{ Title, Body string }{
	{"Groceries", "Oat milk, lemons, the good bread, and something for Sunday."},
	{"Trip ideas", "A lake with a short trail, somewhere with a bookshop, no more than three hours away."},
	{"Hinge notes", "A half-opened device wants content above the crease and controls below it."},
}

// describeFold is the readout's fold line, in words a reader can check
// against the device in their hand.
func describeFold(win core.Window) string {
	if !win.HasFold {
		return "none"
	}
	f := win.Fold
	at := fmt.Sprintf("x %.0f", f.Bounds.X)
	if f.Orientation == core.FoldHorizontal {
		at = fmt.Sprintf("y %.0f", f.Bounds.Y)
	}
	sep := "continuous"
	if f.Separating {
		sep = "separating"
	}
	return fmt.Sprintf("%s, %s at %s, %s", f.State, f.Orientation, at, sep)
}

// lessonFoldables teaches the window record and the one layout built on it.
// The readout is the part worth running on a foldable or its emulator: fold
// the device, bend it, rotate it, and every line changes with no code in the
// lesson asking for it. The TwoPane demo sits in a scrolling page, so it sets
// IgnoreHorizontalFold — the one arrangement a scrolling pane cannot line up.
func lessonFoldables() Lesson {
	return Lesson{
		Title:   "Foldables and window size",
		Summary: "hooks.UseWindow and comps.TwoPane: size classes, tabletop and book postures, and panes that sit on either side of a hinge instead of across it.",
		Body: func(ctx *core.Context) core.View {
			// Hooks first and unconditionally, as in every lesson.
			win := hooks.UseWindow(ctx)
			picked := core.NewState(ctx, 0)

			t := ctx.Theme()
			readout := func(label, value string) core.View {
				return core.Row(
					core.Gap(8),
					core.Text(label, core.UseStyle(t.Typography.Caption), core.Width("110px"), core.FlexShrink(0)),
					core.Text(value, core.UseStyle(t.Typography.Body), core.FlexGrow(1)),
				)
			}
			size := "not reported yet"
			if win.Received {
				size = fmt.Sprintf("%.0f × %.0f", win.Width, win.Height)
			}

			list := make([]core.PropsAndChildren, 0, len(foldNotes)+1)
			list = append(list, core.Gap(6))
			for i, n := range foldNotes {
				emphasis := comps.EmphasisGhost
				if i == picked.Get() {
					emphasis = comps.EmphasisOutlined
				}
				list = append(list, comps.Button{
					Label:    n.Title,
					Emphasis: emphasis,
					OnTap:    func() { picked.Set(i) },
				})
			}
			note := foldNotes[picked.Get()]

			return core.Column(
				core.Gap(14),
				prose("A foldable changes shape under a running app: it unfolds from a phone "+
					"into a small tablet, and it bends into postures a slab never has. "+
					"hooks.UseWindow returns the window the host reported — its size, and the "+
					"fold crossing it if there is one — and re-renders when either changes."),
				demoPanel("Fold, unfold, bend or rotate the device and watch every line change.",
					readout("Window", size),
					readout("Width class", string(win.WidthClass())),
					readout("Height class", string(win.HeightClass())),
					readout("Posture", string(win.Posture())),
					readout("Fold", describeFold(win)),
				),
				codeBlock(`win := hooks.UseWindow(ctx)
switch {
case win.Posture() == core.PostureTabletop:  // hinge across, device standing
    return videoAboveControls
case win.WidthClass() == core.SizeCompact:   // a phone, or a folded foldable
    return listOnly
}
return listBesideDetail`),
				prose("Branch on the size class and the posture, not the raw width. The "+
					"report changes with every pixel of a window drag; the class changes at "+
					"600 and 840, which is where a layout actually wants to."),
				prose("comps.TwoPane is the layout most screens want from all of this. On a "+
					"separating fold it sizes the first pane to end at the hinge, so nothing "+
					"straddles the crease; with room and no hinge it splits by Ratio; on a "+
					"phone it stacks both panes or shows the one Compact names."),
				codeBlock(`comps.TwoPane{
    First:   noteList,
    Second:  noteDetail,
    Ratio:   0.4,
    Compact: comps.TwoPaneFirst,   // on a phone, tap through instead
}`),
				demoPanel("Pick a note. Unfold the device to put the list and the note side by side.",
					comps.TwoPane{
						First:  core.Column(list...),
						Second: core.Column(core.Gap(6), core.Text(note.Title, core.UseStyle(t.Typography.Subtitle)), prose(note.Body)),
						Ratio:  0.4,
						// A lesson scrolls, so this pane moves past a
						// horizontal hinge and no offset could line it up.
						IgnoreHorizontalFold: true,
					},
				),
				keyPoints(
					"hooks.UseWindow returns core.Window: Width and Height in layout units, WidthClass and HeightClass, Posture, and the fold.",
					"Posture is tabletop for a half-opened horizontal hinge and book for a vertical one; everything else is normal.",
					"Only a separating fold is laid out around. A flat, continuous panel can be crossed.",
					"Fold bounds are in window coordinates; a TwoPane that does not start at the window's corner passes its Origin.",
					"Android reports through Jetpack WindowManager, the browser through viewport segments and device posture, and iOS reports size alone.",
				),
			)
		},
	}
}

// lessonFAB is the floating action button and the Screen slot that floats it.
// A lesson is itself a scrolling screen, so the demo cannot hand a Screen a
// Floating view without nesting one safe area inside another; instead it
// floats the FAB over a fixed-height ZStack, which is exactly what
// Screen.Floating builds around the content, and prints the Screen form as
// code. It is appended at the end of the chapter for the reason 4.15 was:
// lesson numbers already in deep links do not move.
func lessonFAB() Lesson {
	return Lesson{
		Title:   "The floating action button",
		Summary: "comps.FAB, a raised disc for a screen's one primary action, and Screen.Floating, the layer it floats on.",
		Body: func(ctx *core.Context) core.View {
			// Hooks first and unconditionally, as in every lesson.
			notes := core.NewState(ctx, 3)
			t := ctx.Theme()

			// A mixed prop-and-child list, as core.Column takes: the gap first,
			// then one row per note.
			rows := make([]core.PropsAndChildren, 0, notes.Get()+1)
			rows = append(rows, core.Gap(4))
			for i := 1; i <= notes.Get(); i++ {
				rows = append(rows, comps.ListRow{Title: fmt.Sprintf("Note %d", i)})
			}

			return core.Column(
				core.Gap(14),
				prose("A FAB is comps.Button in a circle: the fill, the ink, the disabled "+
					"dimming and the Style order are all Button's, so a FAB and a Button on "+
					"one screen cannot disagree about what primary looks like. Icon-only it is "+
					"a 56-point disc; with a Label it is the extended pill, at the same height, "+
					"so naming the action does not move it."),
				codeBlock(`comps.FAB{Icon: "+", AccessibilityLabel: "New note", OnTap: create}
comps.FAB{Icon: "✎", Label: "Compose", OnTap: compose}`),
				prose("It does not place itself. The only container that draws one child over "+
					"another is core.ZStack, which sizes to its largest layer, so the screen "+
					"that already fills the safe area is what hosts it: Screen.Floating makes "+
					"the content the base layer and places the FAB bottom-end, above the Footer."),
				codeBlock(`comps.Screen{
    Scroll:   true,
    Children: []core.View{notes},
    Floating: comps.FAB{Icon: "+", AccessibilityLabel: "New note", OnTap: create},
    Footer:   comps.BottomBar{Items: tabs, Selected: tab.Get()},
}`),
				prose("A plus is a glyph, not a name, so an icon-only FAB needs an "+
					"AccessibilityLabel; the widget invents nothing to say instead."),
				demoPanel("Tap the disc to add a note. The stack below is the shape Screen.Floating builds around a screen's content.",
					core.ZStack(
						core.Width("100%"),
						core.Height("220px"),
						core.BorderRadius(12),
						core.BackgroundColor(t.Colors.Surface),
						// The content layer fills the stack, as the screen's
						// column does under Floating.
						core.Scroll(
							core.Width("100%"),
							core.Height("100%"),
							core.Column(rows...),
						),
						core.Box(
							core.StackAlign(core.StackAlignBottomEnd),
							core.Margin(t.Spacing.MD),
							comps.FAB{
								Icon:               "+",
								AccessibilityLabel: "New note",
								OnTap:              func() { notes.Set(notes.Get() + 1) },
							},
						),
					),
					core.Row(
						core.Gap(12),
						core.AlignItemsProp(core.AlignItemsCenter),
						comps.FAB{Icon: "✎", Label: "Compose", OnTap: func() {}},
						comps.FAB{Icon: "↑", Size: comps.FABSmall, AccessibilityLabel: "Back to top", OnTap: func() {}},
					),
					caption(fmt.Sprintf("%d notes", notes.Get())),
				),
				keyPoints(
					"FAB is a Button in a circle; Variant, Emphasis, Disabled and Style all mean what they mean on Button.",
					"Icon-only is a 56-point disc (40 for FABSmall); Label makes it an extended pill at the same height.",
					"Screen.Floating stacks the content under the view and places it bottom-end with a Spacing.LG margin, above the Footer.",
					"The base layer states Width and Height 100%: a ZStack centres and hugs any layer that says nothing.",
					"An icon-only FAB needs AccessibilityLabel; a glyph is not a name.",
				),
			)
		},
	}
}

// --- 4.23 ----------------------------------------------------------------

// qrLessonLevels pairs the four error-correction levels with their captions,
// so the segmented control's labels and the value it selects cannot drift
// apart — the same trick 4.1 plays with variantNames and variantValues.
var qrLessonLevels = []struct {
	caption  string
	level    comps.ECLevel
	recovers string
}{
	{"L", comps.ECLow, "about 7%"},
	{"M", comps.ECMedium, "about 15%"},
	{"Q", comps.ECQuartile, "about 25%"},
	{"H", comps.ECHigh, "about 30%"},
}

// lessonQRCode is the QR code widget: a Canvas of modules encoded in Go.
//
// The demo is the error-correction level, because that is the one field whose
// effect is visible rather than argued: the same payload at H needs a bigger
// symbol than at L, so at a fixed drawn width the modules shrink, and that —
// not damage — is what a code on a screen actually fails on.
//
// Appended at the end of the chapter for the reason 4.15 and 4.22 were: lesson
// numbers already in deep links do not move.
func lessonQRCode() Lesson {
	return Lesson{
		Title:   "Codes a camera can read",
		Summary: "comps.QRCode, a QR symbol encoded in Go and drawn as one Canvas path — no image file, no network round trip.",
		Body: func(ctx *core.Context) core.View {
			// Hooks first and unconditionally, as in every lesson.
			level := core.NewState(ctx, 1) // M, the widget's own default
			chosen := qrLessonLevels[level.Get()]

			return core.Column(
				core.Gap(14),
				prose("comps.QRCode encodes its Data in Go and draws the result on a "+
					"core.Canvas. There is no image file to ship, no service to call and no "+
					"dependency outside the module: the encoder lives in internal/qr, because "+
					"a QR code is a closed algorithm with published test vectors and that is "+
					"the kind of thing cheaper to own than to track."),
				codeBlock(`comps.QRCode{
    Data:  "cats://pair?t=9f2c1a&host=studio.local",
    Label: "Scan to pair this device",
}`),
				prose("Size is the whole box, quiet zone included — the light margin a reader "+
					"needs is drawn inside the square rather than around it, so the widget's "+
					"footprint does not depend on how long the data turned out to be. Every "+
					"dark module goes into one filled path: three thousand separate shapes "+
					"would be the reconciler's worst case, and they would be antialiased "+
					"against each other into hairlines a decoder can read as light."),
				prose("The level is the trade the demo shows. More redundancy means a larger "+
					"symbol for the same payload, and at a fixed drawn width a larger symbol "+
					"means smaller modules — which is what a phone camera struggles with. The "+
					"default is ECMedium for that reason, not ECHigh."),
				demoPanel("Pick an error-correction level and watch the modules shrink.",
					comps.SegmentedControl{
						Labels:       qrLessonLevelCaptions(),
						Selected:     level.Get(),
						OnSelect:     func(i int) { level.Set(i) },
						KeyPrefix:    "qr-level-",
						SegmentLabel: func(label string, _ int) string { return "Level " + label },
					},
					core.Row(
						core.Justify(core.JustifyCenter),
						comps.QRCode{
							Data:  "https://grmob.example/pair?t=9f2c1a&host=studio.local",
							Size:  200,
							Level: chosen.level,
							Label: "Scan to pair this device",
						},
					),
					caption(fmt.Sprintf("Level %s recovers %s of the symbol.", chosen.caption, chosen.recovers)),
				),
				prose("Colour is not themed, and that is the point. A camera's binarizer assumes "+
					"dark modules on a light field, so the widget uses the theme's own ink and "+
					"surface only when those are already dark-on-light with room to spare, and "+
					"otherwise falls back to black on white. In a dark theme that means a white "+
					"square — which is what every payment app shows, for this reason. There is "+
					"no Foreground field: an unscannable code looks exactly like a working one."),
				prose("The data is never spoken. A reader announcing a 300-character URL one "+
					"character at a time helps nobody, so Label should say what scanning it "+
					"will do, and the same action should be on screen somewhere a person can "+
					"reach without a second device."),
				keyPoints(
					"comps.QRCode encodes in Go and draws one Canvas: no image file, no network, no third-party dependency.",
					"Size is the whole box including the quiet zone, which defaults to the standard's four modules.",
					"Every dark module is in a single filled path, with runs merged along each row — no child per module, and no seams between them.",
					"The zero Level is ECMedium: at a fixed width more redundancy buys damage tolerance a screen does not need, and costs module size a camera does.",
					"Dark-on-light always, whatever the theme; there is no colour override, because an unscannable code fails silently.",
					"Label names what scanning does; the data itself is never announced.",
				),
			)
		},
	}
}

// qrLessonLevelCaptions is the segmented control's label slice, taken from the
// same table the values come from.
func qrLessonLevelCaptions() []string {
	out := make([]string, len(qrLessonLevels))
	for i, l := range qrLessonLevels {
		out[i] = l.caption
	}
	return out
}

// --- 4.24 ----------------------------------------------------------------

// lessonTimers is the pair of widgets that own a tick — the only two in the
// package that watch a duration rather than draw one they were handed.
//
// The demo runs both at once because the interesting thing is the difference:
// the countdown holds a deadline and reports when it passes, the stopwatch
// holds nothing and reports nothing. The "ran out N times" caption is what
// makes OnDone's once-per-crossing visible without the reader having to take
// it on trust.
//
// Appended at the end of the chapter for the reason 4.15, 4.22 and 4.23 were:
// lesson numbers already in deep links do not move.
func lessonTimers() Lesson {
	return Lesson{
		Title:   "Counting down, counting up",
		Summary: "comps.Countdown and comps.Stopwatch: the two widgets that own a tick, and why OnDone comes from an effect.",
		Body: func(ctx *core.Context) core.View {
			// Hooks first and unconditionally, as in every lesson. The
			// deadline's initial value is read on the mount pass only, so
			// entering the lesson starts a fresh ten seconds.
			deadline := core.NewState(ctx, time.Now().Add(10*time.Second))
			ranOut := core.NewState(ctx, 0)

			// The stopwatch's two numbers, held here because the widget holds
			// neither: time banked from earlier runs, and the start of the
			// current one.
			banked := core.NewState(ctx, time.Duration(0))
			since := core.NewState(ctx, time.Time{})
			running := core.NewState(ctx, false)

			// The four moves from the type doc, as the handlers the buttons
			// get. Each one is an assignment or two and no arithmetic the
			// widget could have done instead.
			start := func() { since.Set(time.Now()); running.Set(true) }
			pause := func() {
				banked.Set(banked.Get() + time.Since(since.Get()))
				running.Set(false)
			}
			reset := func() { banked.Set(0); running.Set(false) }

			return core.Column(
				core.Gap(14),
				prose("Every widget so far has been handed what it draws. These two are not: "+
					"a countdown has to know what time it is now, so it owns a tick. That makes "+
					"both of them hook callers, with the rule 4.4 gave for Accordion — render "+
					"them in a stable position on every pass and drive Hidden, rather than "+
					"wrapping them in a core.If."),
				codeBlock(`comps.Countdown{Until: expiresAt, OnDone: func() { code.Set("") }}
comps.Stopwatch{Since: startedAt.Get(), Elapsed: banked.Get(), Running: running.Get()}`),
				prose("The tick is hooks.UseIntervalWhile with an empty callback: the widget "+
					"reads the clock in its own Render, so all a tick has to do is bring the "+
					"render back. It runs only while there is a reason for it — a finished "+
					"countdown stops, and a hidden one stops too unless it still owes an "+
					"OnDone, because hiding it removes the reason to draw but not the reason "+
					"to count."),
				prose("OnDone comes from a hooks.UseEffect keyed on whether the deadline has "+
					"passed, never from the render pass. A render may run more than once for "+
					"one state and runs while the tree is being built, so a handler called "+
					"from inside it would fire twice or re-enter the renderer. Keyed on the "+
					"crossing rather than on the widget, it fires once each time the deadline "+
					"goes by — which means moving Until forward is the whole of a restart."),
				demoPanel("Restart the countdown and watch the caption. Then start, pause and resume the stopwatch.",
					core.Row(
						core.Gap(24),
						core.Justify(core.JustifyCenter),
						core.AlignItemsProp(core.AlignItemsCenter),
						core.Column(
							core.Gap(4),
							core.AlignItemsProp(core.AlignItemsCenter),
							comps.Countdown{
								Until: deadline.Get(),
								Size:  34,
								// Set from the effect's goroutine, which is
								// where every OnDone runs.
								OnDone: func() { ranOut.Set(ranOut.Get() + 1) },
							},
							caption("Countdown"),
						),
						core.Column(
							core.Gap(4),
							core.AlignItemsProp(core.AlignItemsCenter),
							comps.Stopwatch{
								Since:   since.Get(),
								Elapsed: banked.Get(),
								Running: running.Get(),
								Size:    34,
							},
							caption("Stopwatch"),
						),
					),
					core.Row(
						core.Gap(8),
						core.Justify(core.JustifyCenter),
						comps.Button{
							// "Restart", not "Restart 10s": the three
							// buttons came to 278pt against the 276 a
							// 402pt iPhone leaves this row, and the
							// flex shrink (CSS's, on iOS and the web)
							// takes the shortfall from the one label
							// that can wrap. The countdown itself shows
							// the 10s the moment it restarts.
							Label:    "Restart",
							Emphasis: comps.EmphasisOutlined,
							OnTap:    func() { deadline.Set(time.Now().Add(10 * time.Second)) },
						},
						comps.Button{
							Label: map[bool]string{true: "Pause", false: "Start"}[running.Get()],
							OnTap: func() {
								if running.Get() {
									pause()
									return
								}
								start()
							},
						},
						comps.Button{
							Label:    "Reset",
							Emphasis: comps.EmphasisGhost,
							OnTap:    reset,
						},
					),
					caption(fmt.Sprintf("Ran out %d time%s", ranOut.Get(), plural(ranOut.Get()))),
				),
				prose("The two roundings go opposite ways, and both are conservative. The "+
					"countdown rounds up, so it never says you have less time than you do; the "+
					"stopwatch truncates, so it never claims more elapsed time than has "+
					"passed and its first second reads 0:00. Neither shows hundredths: a "+
					"core.State change requests a render of the whole tree, and two animated "+
					"digits are not worth a hundred passes a second."),
				prose("The stopwatch's two fields are the pair every stopwatch keeps, and they "+
					"live with the caller for the reason SliderRow's draft does — state held "+
					"inside the widget is state the app cannot save, restore, or show anywhere "+
					"else, and a running stopwatch is exactly the thing an app wants to keep "+
					"across a screen change."),
				keyPoints(
					"Both widgets own a tick, so both are hook callers: stable position, every pass, Hidden rather than core.If.",
					"The tick is UseIntervalWhile with an empty callback — the widget reads the clock itself, so a tick only has to bring the render back.",
					"A finished countdown stops ticking; a hidden one stops too, unless it still owes an OnDone.",
					"OnDone is an effect keyed on the crossing, so it fires once per deadline and re-arms when Until moves forward.",
					"Countdown rounds up and Stopwatch truncates: neither ever flatters the number it is reporting.",
					"The stopwatch's banked time and start instant belong to the caller, which is what makes pause, resume and reset plain assignments.",
				),
			)
		},
	}
}

// --- 4.25 ----------------------------------------------------------------

// tutorialStayMarked is the demo hotel's fully-booked nights: the two days in
// the middle of March 2026 the reader will find dotted, so the grid in the
// sheet is carrying information while they pick around it rather than being a
// bare month.
var tutorialStayMarked = map[string]int{"2026-03-18": 1, "2026-03-19": 2}

// 4.25 — a range of days, which is two dates and one question: who holds the
// half-made one. The lesson's spine is that the answer is not the same as
// every other widget's in this chapter, and why: the pending start is the one
// piece of state an application genuinely does not want, because holding it in
// the caller would make backing out of the sheet destroy the range the reader
// opened it to look at.
//
// The demo shows both halves of D6 on one screen — the picker that runs the
// protocol, and a static grid underneath wearing the same band with no
// OnSelect at all — because the two fields on Calendar are display and the
// picker is only their packaging.
//
// Appended at the end of the chapter for the reason 4.15, 4.22, 4.23 and 4.24
// were: lesson numbers already in deep links do not move.
func lessonDateRange() Lesson {
	return Lesson{
		Title:   "Picking a span of days",
		Summary: "comps.DateRangePicker and Calendar's band: one rule over a pending start, and why the widget holds it.",
		Body: func(ctx *core.Context) core.View {
			// The span, as the two dates it is. Held here — a completed range
			// is exactly the state an application wants — while the half-made
			// one stays inside the widget.
			from := core.NewState(ctx, time.Time{})
			to := core.NewState(ctx, time.Time{})

			// What the sheet's grid dots: nights the demo hotel has already
			// let. A count, as 4.9's was.
			marked := func(d time.Time) int { return tutorialStayMarked[d.Format("2006-01-02")] }

			// The reading under the field. Both ends are midday in the same
			// location, so the subtraction is a whole number of days and the
			// nights are one fewer than the days the band covers.
			reading := "Nothing booked. Tap a day, then tap another — either way round."
			if f, tt := from.Get(), to.Get(); !f.IsZero() && !tt.IsZero() {
				nights := int(tt.Sub(f).Hours() / 24)
				reading = fmt.Sprintf("%s → %s · %d night%s",
					f.Format("Mon 2 Jan"), tt.Format("Mon 2 Jan"), nights, plural(nights))
			}

			return core.Column(
				core.Gap(14),
				prose("A date range is two dates, and picking one is two taps. Everything "+
					"awkward about it lives in the gap between them: after the first tap there "+
					"is a range that is half made, and something has to hold it. "+
					"comps.DateRangePicker holds it itself — the third piece of state it owns, "+
					"after the open sheet and the browsed month 4.9's DatePicker already had."),
				codeBlock(`comps.DateRangePicker{
    Start:    from.Get(),
    End:      to.Get(),
    OnChange: func(a, b time.Time) { from.Set(a); to.Set(b) },  // once, on the second tap
    Calendar: comps.Calendar{Today: today, Min: today},         // the template
}`),
				prose("That is the opposite of the call SliderRow made in 6.8, and for the same "+
					"test applied to a different answer: hold state in the widget only when no "+
					"application wants it. A form's field is a span of days or it is nothing — "+
					"\"from the 14th, no end yet\" is a value it would have to invent a way to "+
					"hold and a way to draw. Worse, a caller holding it would find the first tap "+
					"had already overwritten the range they opened the sheet to check, so the "+
					"backdrop, the ✕ and the back gesture would all be traps."),
				prose("What the widget does with that state is one rule, and the rule is the whole "+
					"protocol: no pending start, and this tap becomes it; a pending start, and this "+
					"tap is the other end. Everything a range picker is usually specified with "+
					"falls out of those two lines. The third tap starts a new range, because "+
					"completing one clears the pending start. Tapping one day twice is a one-day "+
					"range. And tapping the earlier day second orders the pair rather than "+
					"throwing the tap away — \"the other end\" is not a claim about which end."),
				demoPanel("Tap a day, then another. Then reopen and tap a third time: the old span leaves the grid and a new one begins.",
					comps.FormField{
						Label: "Stay dates",
						Hint:  "Two taps. The sheet closes on the one that completes the range.",
						Input: comps.DateRangePicker{
							Start:       from.Get(),
							End:         to.Get(),
							Placeholder: "Choose your nights",
							Title:       "Stay dates",
							OnChange: func(a, b time.Time) {
								from.Set(a)
								to.Set(b)
							},
							OnClear: func() {
								from.Set(time.Time{})
								to.Set(time.Time{})
							},
							Calendar: comps.Calendar{
								Today:  tutorialToday,
								Min:    tutorialCalMin,
								Max:    tutorialCalMax,
								Marked: marked,
							},
						},
					},
					caption(reading),
					// The same two fields on a grid that takes no taps at all:
					// RangeStart and RangeEnd are display, and the picker is
					// their packaging. No OnMonthChange either, so this one
					// draws no arrows — the static case from 4.9.
					comps.Calendar{
						RangeStart: from.Get(),
						RangeEnd:   to.Get(),
						Today:      tutorialToday,
						DayLabel: func(d time.Time) string {
							return d.Format("2 January 2006") + ", summary"
						},
					},
					caption("The same span on a grid with no OnSelect — the fields are display, and this one is a picture."),
				),
				prose("Calendar draws the band itself. The two endpoints wear the selected day's "+
					"fill — there is one \"this day is chosen\" look in the grid and it stays one "+
					"look — and the days between wear the same colour thinned to 20% and give up "+
					"their corner radius, which is the whole of what makes a run of them read as "+
					"one shape rather than as a row of pills. Each endpoint is rounded on its "+
					"outside and square toward the band (core.CornerRadii), so the fill runs "+
					"into the band with no notch at the join."),
				codeBlock(`comps.Calendar{RangeStart: from, RangeEnd: to, Today: today}

// │ 15  16 [17]▓18▓▓19▓▓20▓[21] 22 │   [n] endpoint, ▓ interior`),
				prose("Three edges the band has opinions about. It runs through the leading and "+
					"trailing adjacent days rather than stopping at the 1st, because cutting it "+
					"where the month happens to end would stop it somewhere the reader can see no "+
					"reason for. It breaks at the end of each week row, which is where a calendar "+
					"breaks. And today's ring survives inside it, going square with the cells it "+
					"sits in: a range covering today is the common case, not the odd one, and "+
					"losing the ring there would be the one place the grid stopped saying what day "+
					"it is."),
				prose("For a screen reader every day of the span is announced as selected, not "+
					"only its two ends — ARIA's own date-range grid marks the whole band, and the "+
					"nights between the taps are as chosen as the days that named them. The ends "+
					"are then named in the cell's label, \", start of range\" and \", end of "+
					"range\". That is a suffix rather than a state for the one reason the "+
					"calendar accepts a suffix at all: no target has a property for it, so the "+
					"alternative is fourteen identically named selected days with no findable "+
					"edge."),
				keyPoints(
					"A range is two dates and one question: who holds the half-made one. The widget does.",
					"Holding the pending start is what makes backing out of the sheet harmless — every way out discards it.",
					"One rule — no pending start begins one, a pending start ends one — gives the third tap, the one-day range and the out-of-order pair for free.",
					"OnChange fires once per completed range, always with start ≤ end.",
					"Calendar.RangeStart and RangeEnd are display: a grid can wear the band with no OnSelect at all.",
					"The band is the selected fill at both ends and that colour thinned, square, between them; today keeps its ring inside it.",
					"Every day of the span is announced as selected; the two ends are named, because no platform has a property that could say it.",
				),
			)
		},
	}
}

// plural is the "1 night" / "2 nights" branch, spelled out rather than
// printed as "night(s)": the caption is prose the reader is meant to read.
func plural(n int) string {
	if n == 1 {
		return ""
	}
	return "s"
}

// --- 4.26 ----------------------------------------------------------------

// tutorialAppointment is where 4.26's field starts: half past nine on the
// demo's "today", so the date half and the time half both have something to
// show before the reader touches either.
var tutorialAppointment = time.Date(2026, time.March, 11, 9, 30, 0, 0, time.UTC)

// 4.26 — a time of day, and the sheet it turned out not to need. The lesson's
// spine is the question 4.25 asked, answered the other way: a range has a
// half-made state that somebody must hold, and a time has none, so the widget
// holds nothing and reports every pick at once.
//
// The demo composes a DatePicker and a TimePicker into one time.Time, because
// "the date and location are carried through" is the claim that makes the
// field useful and it is only visible with a date beside it. A DigitalClock
// reads the same value under the same Hour24 switch, so the field and the
// clock are seen to be configured with one word.
//
// Appended at the end of the chapter for the reason 4.25 was.
func lessonTimePicker() Lesson {
	return Lesson{
		Title:   "Picking a time of day",
		Summary: "comps.TimePicker: two or three native pickers in a row, and why a time needs no sheet and no Done.",
		Body: func(ctx *core.Context) core.View {
			// One value for both fields. The DatePicker changes its day and
			// the TimePicker its clock, and neither disturbs the other's half.
			when := core.NewState(ctx, tutorialAppointment)
			hour24 := core.NewState(ctx, false)

			// DatePicker reports midday of the tapped day; the clock the
			// TimePicker set is put back on it here. That is the only line of
			// glue the pair needs, and it is the DatePicker's half: the
			// TimePicker already keeps whatever date it is handed.
			setDay := func(d time.Time) {
				w := when.Get()
				when.Set(time.Date(d.Year(), d.Month(), d.Day(), w.Hour(), w.Minute(), 0, 0, w.Location()))
			}

			return core.Column(
				core.Gap(14),
				prose("4.25's range picker held a half-made value, because \"from the 14th, no "+
					"end yet\" is a state no form wants and a sheet the reader backs out of must "+
					"not have written it. A time of day asks the same question and gets the other "+
					"answer. Change only the hour of 9:30 and you have 10:30 — a real time, and "+
					"exactly the one asked for. There is no half-made time, so there is nothing to "+
					"hold, nothing to confirm and nothing to discard."),
				codeBlock(`comps.TimePicker{
    Value:      when.Get(),
    OnChange:   when.Set,   // on every pick, with the date kept
    MinuteStep: 15,
    Label:      "Start time",
}`),
				prose("So there is no sheet. DatePicker needs one because a month grid does not fit "+
					"in a field; a time's two or three choices do. Each is a core.Select — the "+
					"platform's own picker — which also rules out a sheet for a second reason: "+
					"a sheet of pickers is a popup opening popups. And they are Selects rather "+
					"than Steppers because 9:00 to 17:30 is two picks, where a Stepper wants eight "+
					"taps on the hour alone and clamps at 23 where a clock wraps."),
				demoPanel("Pick an hour, a minute, a period — each is reported at once. Change the date: the time stays.",
					comps.FormField{
						Label: "Date",
						Input: comps.DatePicker{
							Selected: when.Get(),
							OnSelect: setDay,
							Title:    "Appointment date",
							Calendar: comps.Calendar{
								Today: tutorialToday,
								Min:   tutorialCalMin,
								Max:   tutorialCalMax,
							},
						},
					},
					comps.FormField{
						Label: "Start time",
						Hint:  "Every quarter hour.",
						Input: comps.TimePicker{
							Value:      when.Get(),
							OnChange:   when.Set,
							Hour24:     hour24.Get(),
							MinuteStep: 15,
							Label:      "Start time",
						},
					},
					comps.SwitchRow{
						Title:    "24-hour clock",
						Subtitle: "One word for the field and the clock",
						On:       hour24.Get(),
						OnToggle: hour24.Set,
					},
					comps.DigitalClock{Time: when.Get(), Hour24: hour24.Get(), ShowDate: true, Size: 32},
					caption("Holding "+when.Get().Format("Mon 2 Jan 2006, 15:04")+"."),
				),
				prose("OnChange hands back the Value it was given with the hour and minute "+
					"replaced, on the same date and in the same location, so the two fields above "+
					"share one time.Time and each edits its own half. The seconds are dropped: the "+
					"field does not show them, and a value it reports should be one it would draw. "+
					"A zero Value is midnight rather than a blank, for the same reason there is no "+
					"sheet — a blank option would bring back the half-made time."),
				prose("MinuteStep thins the minute list. A time loaded from elsewhere that is off "+
					"the step — 9:07 on this quarter-hour field — is shown as itself, with :07 "+
					"slotted into the list in order, rather than rounded: rounding for display "+
					"would be a lie about the value, and rounding through OnChange would be a "+
					"write the reader never made."),
				prose("For a screen reader the row is a group named by Label whose value is the "+
					"whole time, the shape 4.15's Stepper has, and each picker is named — Hour, "+
					"Minute, AM/PM — because a picker's own text is only its current option, and "+
					"\"9, pop-up button\" does not say which part of which time it is."),
				keyPoints(
					"A time has no half-made state, so TimePicker holds none: no hooks, no sheet, no Done.",
					"Every pick is reported at once as a complete time.",
					"The date and location of Value are kept, so a DatePicker and a TimePicker compose into one time.Time.",
					"Hour24 is DigitalClock's word; the field and the clock are configured alike.",
					"An off-step minute is shown as itself, never rounded.",
					"Built from core.Select, so each part is the platform's own picker.",
				),
			)
		},
	}
}

// --- 4.27 ----------------------------------------------------------------

// tutorialOrderCrumbs is 4.27's trail: the order sits two levels under the
// orders list, so the breadcrumb has two ancestors to go back to.
var tutorialOrderCrumbs = []string{"Orders", "March", "#40121"}

// tutorialSharedWith are the six people 4.27's order is shared with. Initials
// only, no Src, so the stack draws with no network at all.
var tutorialSharedWith = []comps.Avatar{
	{Name: "Ada Lovelace"},
	{Name: "Grace Hopper"},
	{Name: "Katherine Johnson"},
	{Name: "Dorothy Vaughan"},
	{Name: "Mary Jackson"},
	{Name: "Annie Easley"},
}

// tutorialReceipt is the image 4.27's Lightbox opens: a placeholder service's
// URL, the same one examples/fintechapp uses, at a photo's proportions.
const tutorialReceipt = "https://dummyimage.com/800x600/1e3a5f/ffffff&text=Receipt+%2340121"

// 4.27 — Tier E of the second low-hanging-fruit round, seven small widgets
// taught together because none of them has a lesson's worth of idea alone.
// The demo is one screen that uses all seven the way an app would — an order's
// detail page — so each piece is seen in the place it was made for rather than
// on a bare panel, and the prose says the one thing about each that is not
// obvious from its picture.
//
// The bar's Inbox badge clears when Inbox is tapped, so the badge is seen to
// be caller state that a render reads, not something the bar counts.
//
// Appended at the end of the chapter for the reason 4.25 was.
func lessonSmallPieces() Lesson {
	return Lesson{
		Title:   "Seven small pieces",
		Summary: "KeyValueList, Breadcrumb, AvatarStack, LabeledSeparator, PasswordField, a badge on BottomBar and Lightbox — one order screen built from all of them.",
		Body: func(ctx *core.Context) core.View {
			crumbNote := core.NewState(ctx, "")
			viewing := core.NewState(ctx, false)
			password := core.NewState(ctx, "")
			tab := core.NewState(ctx, 0)
			unread := core.NewState(ctx, 3)

			// The badge is the caller's count drawn as text; empty draws none.
			inboxBadge := ""
			if n := unread.Get(); n > 0 {
				inboxBadge = strconv.Itoa(n)
			}
			choose := func(i int) func() {
				return func() {
					tab.Set(i)
					if i == 1 {
						unread.Set(0) // visiting the inbox reads it
					}
				}
			}

			return core.Column(
				core.Gap(14),
				prose("None of these seven has a lesson's worth of idea on its own, and every one "+
					"of them is a thing screens kept hand-rolling. So here they are together, "+
					"doing their jobs on one screen: the detail page of an order."),
				demoPanel("Tap a crumb, the receipt, Show, and the Inbox.",
					comps.Breadcrumb{
						Items: tutorialOrderCrumbs,
						OnTap: func(i int) { crumbNote.Set("Back to " + tutorialOrderCrumbs[i] + ".") },
					},
					caption(orEmpty(crumbNote.Get(), "The last crumb is where you are, so it is not a button.")),
					comps.KeyValueList{
						Label:    "Order details",
						Dividers: true,
						Rows: []comps.KeyValue{
							{Key: "Placed", Value: "14 Mar 2026"},
							{Key: "Items", Value: "3"},
							{Key: "Total", Value: "$42.10"},
						},
					},
					comps.ListRow{
						Title:    "Shared with",
						Trailing: comps.AvatarStack{Avatars: tutorialSharedWith, Max: 4},
					},
					comps.Button{
						Label:    "View receipt",
						Emphasis: comps.EmphasisOutlined,
						OnTap:    func() { viewing.Set(true) },
						Style:    []core.StyleProp{core.AccessibilityHasPopup(core.PopupDialog)},
					},
					comps.Lightbox{
						Src:       tutorialReceipt,
						Alt:       "Receipt for order 40121, total $42.10",
						Caption:   "Receipt #40121",
						Open:      viewing.Get(),
						OnDismiss: func() { viewing.Set(false) },
					},
					comps.LabeledSeparator{Label: "or"},
					comps.FormField{
						Label: "Confirm with your password",
						Input: comps.PasswordField{
							Value:    password.Get(),
							OnChange: password.Set,
							Label:    "Password",
						},
					},
					comps.BottomBar{
						Selected: tab.Get(),
						Items: []comps.BarItem{
							{Icon: "🧾", Label: "Orders", OnTap: choose(0)},
							{Icon: "📬", Label: "Inbox", OnTap: choose(1),
								Badge: inboxBadge, BadgeLabel: inboxBadge + " unread"},
							{Icon: "👤", Label: "Me", OnTap: choose(2)},
						},
					},
				),
				prose("KeyValueList is ListRows with the key as the title and the value trailing, "+
					"so the value is pinned by the row's growing middle and nothing new solves "+
					"layout. It is a list of list items, each named \"Total, $42.10\", so a reader "+
					"hears a fact once rather than a word and a number as two strangers."),
				prose("Breadcrumb is a navigation landmark of ghost buttons, and its last item is "+
					"text marked as the current page — a button that went where you already are "+
					"would be a dead tab stop at the end of every trail. With no OnTap the whole "+
					"trail is text: a location label rather than a way back."),
				prose("AvatarStack overlaps with a ZStack, not a Row: a negative margin is not "+
					"portable, a margin on a stack layer is. The ring around each face is a disc "+
					"layer of its own rather than a border, because a border is drawn inside the "+
					"box on a phone and outside it in a static export. Max counts the +N disc, so "+
					"Max 4 is four discs wide however long the list, and the whole stack is one "+
					"picture with one name."),
				codeBlock(`comps.FormField{
    Label: "Password",
    Input: comps.PasswordField{Value: pw.Get(), OnChange: pw.Set, Label: "Password"},
}`),
				prose("PasswordField is the input, not the field — FormField already owns the "+
					"label — and it keeps one piece of state, whether the text is showing, because "+
					"no application wants that. The toggle's caption flips between Show and Hide "+
					"but its accessible name stays \"Show password\", with pressed or not pressed "+
					"as its state: a name that flipped would read as two different buttons."),
				prose("A BarItem's Badge turns its icon into a two-layer ZStack, the Badge placed "+
					"at the top-end corner, and joins the item's name — \"Inbox, 3 unread\". An "+
					"item with no Badge renders exactly the tree it always did."),
				prose("Lightbox is Dialog's Modal plumbing with an image where the card would be, "+
					"fitted rather than cropped because seeing what the thumbnail cut off is the "+
					"reason to open it. It is black in every theme, and the panel is filled rather "+
					"than transparent because an iOS sheet is not drawn over the scrim."),
				keyPoints(
					"KeyValueList: ListRows, a list of named items, the value in the quieter ink.",
					"Breadcrumb: a navigation landmark; the current page is text, not a button.",
					"AvatarStack: ZStack layers at growing margins; the ring is a layer; one name for the picture.",
					"LabeledSeparator: two growing Separators and a word that is read.",
					"PasswordField: goes in a FormField; owns only the reveal; a stable name with a pressed state.",
					"BarItem.Badge: a top-end ZStack layer over the icon, and part of the item's name.",
					"Lightbox: a Modal, a fitted image, dark everywhere, and a close button that is always wired.",
				),
			)
		},
	}
}

// orEmpty returns s, or fallback when s is empty — the lesson's caption shows
// a standing hint until a crumb has been tapped.
func orEmpty(s, fallback string) string {
	if s == "" {
		return fallback
	}
	return s
}

// --- 4.28 ----------------------------------------------------------------

// tutorialWorkouts is 4.28's calendar: a deterministic scatter of workouts
// over the seventeen weeks up to the demo's today, so the grid has every step
// of the scale in it without a random source. The pattern — rest on most
// Sundays, longer sessions at weekends — is only there so the picture looks
// like a real habit rather than noise.
func tutorialWorkouts() []comps.DayValue {
	var out []comps.DayValue
	for i := range 17 * 7 {
		day := tutorialToday.AddDate(0, 0, -i)
		n := (i*7 + i/5) % 6 // 0–5, spread unevenly
		switch day.Weekday() {
		case time.Sunday:
			if i%3 != 0 {
				n = 0
			}
		case time.Saturday:
			n = min(5, n+2)
		}
		if n > 0 {
			out = append(out, comps.DayValue{Day: day, Value: float64(n)})
		}
	}
	return out
}

// tutorialOrdersByHour is 4.28's matrix: orders per weekday and two-hour
// slot, with a lunchtime and an evening peak and one slot with no data (the
// till was down), which is what the no-data colour is for.
var tutorialOrdersByHour = [][]float64{
	{2, 8, 5, 3, 6, 2},
	{3, 9, 7, 2, 7, 3},
	{1, 6, math.NaN(), 3, 8, 4},
	{2, 7, 6, 4, 9, 5},
	{4, 9, 8, 5, 11, 8},
}

// tutorialResponseTimes is 4.28's histogram sample: 160 response times in
// milliseconds, built from a fixed sum of three sines so the shape is a skewed
// hump with a long tail — the shape real latencies have — and identical on
// every run.
func tutorialResponseTimes() []float64 {
	out := make([]float64, 160)
	for i := range out {
		x := float64(i)
		v := 180 + 55*math.Sin(x*0.37) + 30*math.Sin(x*1.13) + 25*math.Sin(x*2.71)
		if i%9 == 0 {
			v += 140 // the slow tail
		}
		out[i] = math.Round(math.Max(40, v))
	}
	return out
}

// 4.28 — Tier F of the second low-hanging-fruit round: the two charts whose
// catch the plan named. The lesson is organised around the two catches
// rather than the widgets, because each was settled by a decision a reader
// building their own chart would meet: a quantity needs its own palette role,
// and a range needs its edges on the axis.
//
// Appended at the end of the chapter for the reason 4.25 was.
func lessonHeatAndSpread() Lesson {
	return Lesson{
		Title:   "Heat and spread",
		Summary: "comps.Heatmap, CalendarHeatmap and Histogram, the theme's new Sequential role, and why a histogram's axis names edges.",
		Body: func(ctx *core.Context) core.View {
			extra := core.NewState(ctx, 0.0)
			binChoice := core.NewState(ctx, 0)

			days := tutorialWorkouts()
			if e := extra.Get(); e > 0 {
				days = append(days, comps.DayValue{Day: tutorialToday, Value: e})
			}
			binCounts := []int{0, 4, 8, 16}

			return core.Column(
				core.Gap(14),
				prose("4.20's charts colour their series from the theme's Chart role, and that role "+
					"is categorical: its eight hues are ordered to stay apart, not to mean more. A "+
					"heatmap paints a quantity, which needs the opposite — every step visibly more "+
					"than the one before. So the palette grew a role: Sequential, five steps of one "+
					"blue, evenly spaced in lightness, lightest first."),
				codeBlock(`comps.CalendarHeatmap{
    Subject: "Workouts",
    Days:    workouts,   // []comps.DayValue; one date's entries are summed
    End:     today,      // the last day drawn
}`),
				demoPanel("Log a workout today and watch its cell step up.",
					comps.CalendarHeatmap{
						Subject: "Workouts",
						Days:    days,
						End:     tutorialToday,
					},
					comps.Button{
						Label:    "Log a workout",
						Emphasis: comps.EmphasisOutlined,
						OnTap:    func() { extra.Set(extra.Get() + 1) },
					},
					caption(fmt.Sprintf("Today: %g logged by you.", extra.Get())),
				),
				prose("A grey cell is \"nothing\", not the first step. In a contribution calendar an "+
					"empty day is the ground and every colour is some activity, so CalendarHeatmap "+
					"passes NaN for zero; the plain Heatmap draws zero as a value and keeps grey for "+
					"cells with no data at all, like Wednesday's missing slot below. Days after End "+
					"are not drawn — they have not happened, which is different again."),
				demoPanel("Orders by weekday and hour.",
					comps.Heatmap{
						Subject:      "Orders by hour",
						RowLabels:    []string{"Mon", "Tue", "Wed", "Thu", "Fri"},
						ColumnLabels: []string{"9", "11", "13", "15", "17", "19"},
						Values:       tutorialOrdersByHour,
					},
				),
				prose("Why a list and not a ramp from Primary: a ramp from the brand colour hands the "+
					"scale's range to the brand. From a white page, AmberTheme's amber spans less "+
					"than half the lightness DefaultTheme's blue does, so the same data would read "+
					"less than half as steep. A list is chosen and checked once; a brand that wants its own hue "+
					"states its own list. A dark theme states the same list reversed, so \"more\" "+
					"still runs away from the page."),
				prose("A histogram looks like a bar chart, and the catch was the axis. BarChart "+
					"centres a string under each bar; a bin is a range, and what a reader measures "+
					"against is its edges. n bins have n+1 edges spread evenly edge to edge, which is "+
					"exactly the spacing 4.20's line chart gives its points, so the edges are drawn "+
					"on that axis and the bars touch."),
				demoPanel("Change the bin target. The edges stay round.",
					comps.SegmentedControl{
						Labels:   []string{"Auto", "4", "8", "16"},
						Selected: binChoice.Get(),
						OnSelect: binChoice.Set,
					},
					comps.Histogram{
						Subject: "Response time (ms)",
						Values:  tutorialResponseTimes(),
						Bins:    binCounts[binChoice.Get()],
					},
				),
				prose("Bins is a target, not a promise. The edges come from the same nice-number "+
					"scale the chart axes use, so a bin is 10 or 25 or 50 wide and starts on a "+
					"multiple of it; ask for 16 and you may get 12. Auto is Sturges' rule, "+
					"⌈log₂ n⌉ + 1. The count axis never ticks at a half, because half a sample is a "+
					"gridline pointing at nothing."),
				keyPoints(
					"ColorPalette.Sequential: a quantity's scale, least to most, one hue, even steps.",
					"Heatmap: steps, not a gradient; NaN is no data in Surface; one shape per colour.",
					"CalendarHeatmap: a Heatmap of weeks × weekdays; zero is empty, the future is not drawn.",
					"Histogram: nice bin edges, drawn on LineChart's point axis; the bars touch.",
					"Every one is a single image with one spoken sentence, like every chart here.",
				),
			)
		},
	}
}

// 4.29 — the first batch of the third low-hanging-fruit round: CopyButton
// (G1) and Tier H's three small pieces. The lesson is organised around what
// each piece hands to the platform or refuses to measure, because that is
// the part a reader building their own would get wrong: the copy goes to the
// clipboard and the confirmation to the toast, the link to OpenURL, and the
// calendar's width is the caller's number, not a measurement.
//
// Appended at the end of the chapter for the reason 4.25 was.
func lessonCopyLinkAndList() Lesson {
	return Lesson{
		Title:   "Copy, link and list",
		Summary: "comps.CopyButton, Link and BulletList, and sizing a CalendarHeatmap to the window with WeeksFor.",
		Body: func(ctx *core.Context) core.View {
			followed := core.NewState(ctx, 0)
			// The in-sentence demo's second link, counted apart from the
			// first so a tap shows which run the host dispatched.
			reported := core.NewState(ctx, 0)
			win := hooks.UseWindow(ctx)

			// The lesson column's own inset: the screen's padding and the
			// demo panel's, both sides. Only an estimate is needed — WeeksFor
			// rounds down, so a few px either way moves at most one week.
			// Zero before the host has reported a window (a headless test,
			// the static export) leaves the widget's own default of 17.
			weeks, shown := 0, 17
			if win.Width > 0 {
				weeks = comps.CalendarHeatmap{}.WeeksFor(win.Width - 64)
				shown = weeks
			}

			return core.Column(
				core.Gap(14),
				prose("CopyButton puts a fixed string on the clipboard, ticks the haptic motor and "+
					"shows the platform's toast. It holds no state: the confirmation is the toast and "+
					"not a caption that flips back after a second, because flipping back needs a timer, "+
					"a timer is a hook, and a hook would forbid rendering the button inside an if. "+
					"Every code block in this tutorial now carries one, which is only possible because it is stateless."),
				codeBlock(`comps.CopyButton{
    Text:               inviteCode,
    Label:              "Copy code",
    AccessibilityLabel: "Copy invite code",
}`),
				demoPanel("An invite code with its copy button.",
					core.Row(
						core.Gap(12),
						core.AlignItemsProp(core.AlignItemsCenter),
						core.Text("CATS-4721-QX", core.FontSize(20), core.FontWeight(core.Bold), core.FlexGrow(1)),
						comps.CopyButton{
							Text:               "CATS-4721-QX",
							Label:              "Copy code",
							AccessibilityLabel: "Copy invite code",
							CopiedMessage:      "Invite code copied",
							Emphasis:           comps.EmphasisOutlined,
						},
					),
					comps.CopyButton{Label: "Copy link", Emphasis: comps.EmphasisOutlined},
					caption("The second button has no Text yet — a link still loading — so it is disabled rather than able to clear the clipboard."),
				),
				prose("Link is a line of text that goes somewhere. It is announced as a link, not a "+
					"button, because a reader deciding whether to follow a control needs to know it "+
					"leaves the screen. With a URL it calls core.OpenURL; with OnTap it goes wherever "+
					"the handler takes it. On a line of its own it is not underlined: being a line in "+
					"the link colour is what sets it apart."),
				demoPanel("One link out of the app, one within it.",
					comps.Link{Text: "GrMob on GitHub", URL: "https://github.com/rohanthewiz/grmob",
						AccessibilityHint: "Opens in your browser"},
					comps.Link{Text: "Follow an in-app link", OnTap: func() { followed.Set(followed.Get() + 1) }},
					caption(fmt.Sprintf("In-app link followed %d time%s.", followed.Get(), plural(followed.Get()))),
				),
				prose("Inside a sentence a link is a run of a core.Paragraph, and Link.Span makes one. "+
					"It is underlined there, because in running text the colour is the only other "+
					"difference, and a reader who cannot see colour would miss it. A run can also be "+
					"a link in a colour of its own: each one draws in its own colour on every target."),
				codeBlock(`core.Paragraph([]core.Span{
    {Text: "Read the "},
    comps.Link{Text: "guide", OnTap: openGuide}.Span(ctx),
    {Text: ", or "},
    {Text: "report a problem", Underline: true,
        Color: theme.Colors.Error, OnTap: report},
    {Text: "."},
})`),
				demoPanel("Two links in one sentence, each in its own colour.",
					core.Paragraph([]core.Span{
						{Text: "Read the "},
						comps.Link{Text: "guide", OnTap: func() { followed.Set(followed.Get() + 1) }}.Span(ctx),
						{Text: ", or "},
						{Text: "report a problem", Underline: true, Color: ctx.Theme().Colors.Error,
							OnTap: func() { reported.Set(reported.Get() + 1) }},
						{Text: "."},
					}),
					caption(fmt.Sprintf("Problem reported %d time%s.", reported.Get(), plural(reported.Get()))),
				),
				prose("BulletList is the list every lesson's key points already were, as a widget: a "+
					"list of listitems, the marker hidden and pinned so a long item wraps under its "+
					"own first word. Ordered numbers share one right-aligned column."),
				demoPanel("Ordered, from 9, so the column has to fit \"10.\".",
					comps.BulletList{
						Ordered: true,
						Start:   9,
						Label:   "Release steps",
						Items: []string{
							"Tag the release.",
							"Build the four targets, and run each one's verify suite before uploading anything.",
							"Publish the notes.",
						},
					},
				),
				prose("A CalendarHeatmap always fits its width — its columns stretch — so the question "+
					"is the cells' shape: a year across a phone is a grid of slivers. WeeksFor turns a "+
					"width into the number of weeks at which the cells come out square. The width is "+
					"yours, from hooks.UseWindow less your padding; nothing measures it."),
				codeBlock(`win := hooks.UseWindow(ctx)
comps.CalendarHeatmap{
    Days:  workouts,
    Weeks: comps.CalendarHeatmap{}.WeeksFor(win.Width - 64),
}`),
				demoPanel("Rotate the device or resize the window: the weeks follow.",
					comps.CalendarHeatmap{
						Subject: "Workouts",
						Days:    tutorialWorkouts(),
						End:     tutorialToday,
						Weeks:   weeks,
					},
					caption(fmt.Sprintf("Window %.0fpx wide: %d weeks.", win.Width, shown)),
				),
				keyPoints(
					"CopyButton: WriteClipboard, a light haptic and a toast; stateless, so safe in an if.",
					"CopyButton with no Text is disabled: an empty write would clear the clipboard.",
					"Link: RoleLink, OnTap or OpenURL(URL); a line of its own, not underlined.",
					"BulletList: a list of listitems, hidden pinned markers, ordered numbers in one column.",
					"CalendarHeatmap.WeeksFor: square cells for a width you supply; no measurement, no hook.",
				),
			)
		},
	}
}

// tutorialTrack is a freely licensed sample stream — SoundHelix publishes its
// test songs for exactly this — the same one examples/mobileapp plays.
var tutorialTrack = core.AudioTrack{
	URL:    "https://www.soundhelix.com/examples/mp3/SoundHelix-Song-1.mp3",
	Title:  "SoundHelix Song 1",
	Artist: "SoundHelix",
	Album:  "GrMob Tutorial",
}

// 4.30 — G3 of the third low-hanging-fruit round. The lesson is organised
// around the singleton: an AudioPlayer is a view of the app's one player,
// not a player of its own, and every control's enabled state follows from
// asking whether the loaded track is this one.
//
// Appended at the end of the chapter for the reason 4.25 was.
func lessonAudioPlayer() Lesson {
	return Lesson{
		Title:   "An audio player",
		Summary: "comps.AudioPlayer: the transport for one track on the app's one player, and the scrub reading it holds.",
		Body: func(ctx *core.Context) core.View {
			return core.Column(
				core.Gap(14),
				prose("core's audio is one player for the whole app: one stream, one lock screen, one "+
					"set of headphone buttons. comps.AudioPlayer is the transport for a single track "+
					"on it — title, seek bar, elapsed and total time, back, play-pause and forward, "+
					"and optionally a speed button and Stop."),
				codeBlock(`comps.AudioPlayer{
    Track:    core.AudioTrack{URL: url, Title: "Song 1", Artist: "SoundHelix"},
    Rates:    []float64{1, 1.25, 1.5, 2},
    ShowStop: true,
}`),
				demoPanel("Press Play. On a device, lock the screen: the platform's controls are the same player.",
					comps.AudioPlayer{
						Track:    tutorialTrack,
						Rates:    []float64{1, 1.25, 1.5, 2},
						ShowStop: true,
					},
				),
				prose("A player widget does not own a player. It asks the shared status one question — "+
					"is the loaded track mine? — and everything follows. If it is, the controls drive "+
					"it. If it is not, the widget shows its own track idle, Play loads it (replacing "+
					"whatever was playing, as a phone does) and the rest is disabled, because it "+
					"would act on somebody else's stream. So every episode screen in a podcast app "+
					"can carry one without any of them fighting."),
				prose("While you drag the seek bar the elapsed time follows your finger, and the seek "+
					"is sent once, on release. The widget holds that reading itself — the opposite of "+
					"SliderRow, which leaves a drag's draft to you — because the value being dragged "+
					"is the host's playback position, which no app holds, and because seeing 12:04 "+
					"is the point of scrubbing."),
				keyPoints(
					"AudioPlayer: a view of the one player, keyed by Track.URL; not a player of its own.",
					"Not this track loaded? Play loads it; the controls that would drive another stream are disabled.",
					"The scrub reading is the widget's; the seek is sent once, on release.",
					"It holds hooks (UseAudio and the scrub), so render it unconditionally.",
				),
			)
		},
	}
}

// tutorialChatLine is one message of lesson 4.31's transcript.
type tutorialChatLine struct {
	from, text, time string // from "" is the reader
}

// 4.31 — G4 of the third low-hanging-fruit round, extracted from
// examples/chat. The lesson is organised around the two things the widget
// refuses to be — a thread, and a bubble with a tail — because each refusal
// names a wall a reader building a chat screen will hit.
//
// Appended at the end of the chapter for the reason 4.25 was.
func lessonMessageBubbles() Lesson {
	return Lesson{
		Title:   "Message bubbles",
		Summary: "comps.MessageBubble: whose side, which colours, one spoken stop per message.",
		Body: func(ctx *core.Context) core.View {
			thread := core.NewState(ctx, []tutorialChatLine{
				{"Ana", "Did you see the new release?", "10:41"},
				{"Ana", "Bubbles are a widget now.", "10:41"},
				{"", "Not yet — what changed?", "10:42"},
			})
			draft := core.NewState(ctx, "")

			send := func() {
				text := strings.TrimSpace(draft.Get())
				if text == "" {
					return
				}
				thread.Set(append(append([]tutorialChatLine(nil), thread.Get()...),
					tutorialChatLine{"", text, "10:43"}))
				draft.Set("")
			}

			lines := thread.Get()
			bubbles := []core.PropsAndChildren{
				core.Gap(6),
				// A transcript is ARIA's log: new messages are announced and
				// the rest stays readable in order. See examples/chat.
				core.AccessibilityRole(core.RoleLog),
			}
			for i, l := range lines {
				// The sender line only where the speaker changes: the second
				// of Ana's two messages is Continued, which hides the line
				// and still speaks her name.
				bubbles = append(bubbles, core.Keyed(fmt.Sprintf("line-%d", i), comps.MessageBubble{
					Text:      l.text,
					Sender:    l.from,
					Mine:      l.from == "",
					Time:      l.time,
					Continued: i > 0 && l.from != "" && lines[i-1].from == l.from,
				}))
			}

			return core.Column(
				core.Gap(14),
				prose("A message bubble is a rounded box held to one side of the row: the trailing side "+
					"for your own words in the theme's Primary, the leading side for everyone else's. "+
					"Theirs has no palette role to take — there is no muted container tone — so it is "+
					"Surface with a hairline, Banner's answer to the same gap."),
				codeBlock(`comps.MessageBubble{Text: "Did you see?", Sender: "Ana", Time: "10:41"}
comps.MessageBubble{Text: "Not yet", Mine: true, Time: "10:42"}`),
				demoPanel("Send a message. Ana's second line draws no sender: it is Continued, the second of her run.",
					core.Column(bubbles...),
					comps.InputRow{
						Value:       draft.Get(),
						Placeholder: "Message…",
						OnChange:    draft.Set,
						OnSubmit:    send,
						Button:      comps.Button{Label: "Send"},
					},
				),
				prose("Each bubble is one stop for a screen reader, named who-what-when: \"Ana, Did you "+
					"see the new release?, 10:41\". Your own are named \"You, …\" — MineLabel "+
					"localizes it. Put the bubbles under a RoleLog container so new ones are announced."),
				prose("The tail is a corner, not a point: the bottom corner on the sender's side is "+
					"nearly square, drawn with core.CornerRadii (a radius per corner). A bubble is "+
					"one message; a conversation that opens at its newest and loads older ones is "+
					"comps.MessageThread, at the end of this chapter."),
				keyPoints(
					"MessageBubble: Mine on the trailing side in Primary; theirs on the leading side in Surface with a hairline.",
					"Sender is drawn on theirs only; Continued hides it on the rest of a run and still speaks it.",
					"One spoken stop per message, who-what-when; use a RoleLog container for the transcript.",
					"The tail is one nearly square corner (core.CornerRadii); a whole conversation is comps.MessageThread.",
				),
			)
		},
	}
}

// 4.32 — Tier I of the third low-hanging-fruit round: the two items whose
// catch was settled without a renderer. Both catches are the same shape —
// something a text node cannot tell you or do — so the lesson is organised
// around what was used instead: a length threshold in place of a measurement,
// and a drawing in place of a clipped glyph.
//
// Appended at the end of the chapter for the reason 4.25 was.
func lessonReadMoreAndHalfStars() Lesson {
	return Lesson{
		Title:   "Read more, and half a star",
		Summary: "comps.ExpandableText's length threshold, and Rating.Halves drawing its stars instead of clipping a glyph.",
		Body: func(ctx *core.Context) core.View {
			score := core.NewState(ctx, 3.5)
			return core.Column(
				core.Gap(14),
				prose("ExpandableText caps a paragraph at a few lines with core.MaxLines and adds a "+
					"Read more that opens it in place. The catch is knowing when to offer it: the "+
					"honest rule is \"when the cap cut something\", and no host reports that — it "+
					"is a rendered height. So the toggle shows past a length, Lines × 40 characters "+
					"by default, and ToggleAfter tunes it."),
				codeBlock(`comps.ExpandableText{Text: summary, Lines: 3}`),
				demoPanel("The long review is capped and can be opened; the short one needs neither.",
					comps.ExpandableText{
						Lines: 3,
						Text: "Arrived a day early and set up in ten minutes. The instructions are clear, " +
							"the parts are labelled, and the one screw that was missing turned out to be " +
							"taped inside the lid. After a month of daily use the hinge is as stiff as on " +
							"day one and the finish has not marked. The only thing I would change is the " +
							"cable, which is a little short for a desk against a wall.",
					},
					comps.ExpandableText{Text: "Does what it says. Would buy again."},
				),
				prose("Rating rounds to whole stars unless Halves is set. Half a text glyph would be a "+
					"\"★\" clipped by a half-width box, and a text node cannot be trusted with that: "+
					"SwiftUI truncates a Text squeezed below its width to \"…\" instead of letting it "+
					"be cut. So a Halves rating draws every star on a small Canvas, and the half is "+
					"exact geometry — the left half of a star is the polygon of its left-side points."),
				codeBlock(`comps.Rating{Value: product.Average, Halves: true, ReadOnly: true}`),
				demoPanel("Step the average by halves.",
					comps.Rating{Value: score.Get(), Halves: true, ReadOnly: true, Label: "Average rating"},
					core.Row(
						core.Gap(8),
						comps.Button{Label: "− 0.5", Emphasis: comps.EmphasisOutlined,
							OnTap: func() { score.Set(math.Max(0, score.Get()-0.5)) }},
						comps.Button{Label: "+ 0.5", Emphasis: comps.EmphasisOutlined,
							OnTap: func() { score.Set(math.Min(5, score.Get()+0.5)) }},
					),
					caption(fmt.Sprintf("Value = %g, announced \"%g of 5\".", score.Get(), score.Get())),
				),
				keyPoints(
					"ExpandableText: MaxLines while closed; the toggle past ToggleAfter runes (Lines × 40).",
					"A cap is only applied when there is a toggle to lift it.",
					"The toggle's name stays \"Read more\"; aria-expanded says which way it is.",
					"Rating.Halves: rounds to halves and draws canvas stars; without it, nothing changed.",
				),
			)
		},
	}
}

// --- 4.33 ----------------------------------------------------------------

// threadPageSize is how many messages 4.33's thread opens with and how many
// each older page brings.
const threadPageSize = 12

// threadHistory is 4.33's pretend server: 48 messages, oldest first, between
// Ana, Bruno and the reader. Generated rather than written out, because what
// the lesson is about is how many there are and that they arrive in pages.
func threadHistory() []comps.ThreadMessage {
	who := []string{"Ana", "Ana", "", "Bruno", ""}
	out := make([]comps.ThreadMessage, 48)
	for i := range out {
		from := who[i%len(who)]
		out[i] = comps.ThreadMessage{
			Key:    fmt.Sprintf("m%d", i+1),
			Sender: from,
			Mine:   from == "",
			Text:   fmt.Sprintf("Message %d", i+1),
			Time:   fmt.Sprintf("09:%02d", 12+i),
		}
	}
	return out
}

// lessonMessageThread is comps.MessageThread over a pretend server that
// answers each "load older" after a short delay, so the loading row and the
// once-per-page guard are both visible.
func lessonMessageThread() Lesson {
	return Lesson{
		Title:   "Message threads",
		Summary: "comps.MessageThread opens at the newest message, loads older ones at the top, and keeps your place.",
		Body: func(ctx *core.Context) core.View {
			history := threadHistory()
			// How many of the history's newest messages the thread holds.
			shown := core.NewState(ctx, threadPageSize)
			loading := core.NewState(ctx, false)
			pages := core.NewState(ctx, 0)
			sent := core.NewState(ctx, []comps.ThreadMessage(nil))
			draft := core.NewState(ctx, "")

			// The fetch: 700ms after loading starts, one page lands above.
			hooks.UseTimeoutWhile(ctx, loading.Get(), func() {
				shown.Set(min(shown.Get()+threadPageSize, len(history)))
				pages.Set(pages.Get() + 1)
				loading.Set(false)
			}, 700*time.Millisecond)

			msgs := append(append([]comps.ThreadMessage(nil), history[len(history)-shown.Get():]...), sent.Get()...)
			var older func()
			if shown.Get() < len(history) {
				older = func() { loading.Set(true) }
			}
			send := func() {
				text := strings.TrimSpace(draft.Get())
				if text == "" {
					return
				}
				next := append([]comps.ThreadMessage(nil), sent.Get()...)
				next = append(next, comps.ThreadMessage{
					Key: fmt.Sprintf("sent%d", len(next)+1), Mine: true, Text: text, Time: "10:00",
				})
				sent.Set(next)
				draft.Set("")
			}

			return core.Column(
				core.Gap(14),
				prose("A conversation is the one list that grows upward. It opens on the newest "+
					"message, and scrolling back to the top asks for the page before. When that page "+
					"lands above you, you should still be looking at the message you were reading."),
				codeBlock(`comps.MessageThread{
    Messages:    msgs,        // oldest first, each with a stable Key
    OnLoadOlder: loadOlder,   // nil once there is nothing older
    Loading:     fetching,
}`),
				demoPanel("Scroll to the top of the thread: an older page loads, and your place is kept.",
					comps.MessageThread{
						Messages:    msgs,
						OnLoadOlder: older,
						Loading:     loading.Get(),
						Height:      "320px",
					},
					caption(fmt.Sprintf("%d of %d messages loaded · %d older page%s fetched.",
						shown.Get(), len(history), pages.Get(), plural(pages.Get()))),
					comps.InputRow{
						Value:       draft.Get(),
						Placeholder: "Message…",
						OnChange:    draft.Set,
						OnSubmit:    send,
						Button:      comps.Button{Label: "Send"},
					},
				),
				prose("Three things here are the host's, and two List props declare them. "+
					"core.StartAtEnd opens the list on its last row and keeps it there as rows arrive "+
					"while you are at the end. core.OnStartReached fires when you reach the top, once "+
					"per row count, like OnEndReached at the bottom. The place-keeping needs no prop: "+
					"every bubble is Keyed by its message, and each host keeps the row you were "+
					"looking at where it was."),
				prose("Go never learns a scroll offset. It would only compare it with zero, and by the "+
					"time Go could answer an offset with a correction, the host would already have "+
					"drawn a frame in the wrong place."),
				keyPoints(
					"MessageThread: oldest first, every message with a stable Key, OnLoadOlder nil at the beginning.",
					"core.StartAtEnd opens a List at its end; core.OnStartReached is the top edge, guarded per row count.",
					"The loading caption is a line above the list, so the list's first row, which every host keeps your place by, is always a message.",
				),
			)
		},
	}
}

// tutorialPollOption is one answer of lesson 4.34's poll: the label and the
// votes other people cast. The reader's own vote is lesson state, added on
// top when the comps.PollOption values are built.
type tutorialPollOption struct {
	label string
	votes int
}

// 4.34 — Phase 1 of the fourth low-hanging-fruit round: the three widgets the
// chat family was missing. The lesson is organised around the one question
// all three answer differently — who holds the state — because that is what a
// reader wiring them to a server has to get right: the indicator holds its
// own animation and so must always render, and the bar and the poll hold
// nothing and so are fed by the caller's data.
//
// Appended at the end of the chapter for the reason 4.25 was.
func lessonChatFamily() Lesson {
	return Lesson{
		Title:   "Typing, reactions and polls",
		Summary: "comps.TypingIndicator, ReactionBar and Poll: the chat family's presence widgets, and who holds their state.",
		Body: func(ctx *core.Context) core.View {
			// All three hooks first, unconditionally. The TypingIndicator
			// below adds two of its own, which is the lesson's first point.
			typing := core.NewState(ctx, true)
			reactions := core.NewState(ctx, []comps.Reaction{
				{Emoji: "👍", Count: 3, Mine: true, Label: "thumbs up"},
				{Emoji: "🎉", Count: 1, Label: "party popper"},
				{Emoji: "🤔", Count: 0, Label: "thinking face"},
			})
			// -1 is "not voted". The lesson may hold an index with a
			// sentinel, because it writes the initial value out; the widget
			// may not, because its zero value has to be safe. See Poll.
			voted := core.NewState(ctx, -1)

			// The reader's reaction toggles: Mine flips and Count follows.
			// Copy, change the copy, Set it.
			toggle := func(emoji string) {
				next := append([]comps.Reaction(nil), reactions.Get()...)
				for i := range next {
					if next[i].Emoji != emoji {
						continue
					}
					if next[i].Mine {
						next[i].Count--
					} else {
						next[i].Count++
					}
					next[i].Mine = !next[i].Mine
				}
				reactions.Set(next)
			}

			// Three options at one vote each, so that the reader's vote makes
			// 2 + 1 + 1: 50 / 25 / 25. Reset and look at the closed poll
			// below it for the 34 / 33 / 33 case.
			others := []tutorialPollOption{{"Tabs", 1}, {"Spaces", 1}, {"Whatever gofmt says", 1}}
			options := func(mine int) []comps.PollOption {
				out := make([]comps.PollOption, len(others))
				for i, o := range others {
					out[i] = comps.PollOption{Label: o.label, Votes: o.votes}
					if i == mine {
						out[i].Votes++
						out[i].Mine = true
					}
				}
				return out
			}

			return core.Column(
				core.Gap(14),
				prose("A chat screen needs three small things around its bubbles: a sign that somebody is "+
					"writing, the reactions under a message, and now and then a poll. They are a lesson "+
					"together because they answer one question in two ways: who holds the state."),

				codeBlock(`comps.TypingIndicator{Visible: anaTyping.Get(), Who: "Ana"}`),
				demoPanel("Three dots in a theirs-coloured bubble. The switch hides it; the widget is still in the tree.",
					comps.MessageBubble{Text: "Did you see the new release?", Sender: "Ana", Time: "10:41",
						Style: []core.StyleProp{core.MarginBottom(6)}},
					comps.TypingIndicator{Visible: typing.Get(), Who: "Ana", Caption: true},
					core.Row(
						core.Padding(0),
						core.Gap(8),
						core.AlignItemsProp(core.AlignItemsCenter),
						core.Checkbox(typing.Get(), typing.Set, core.AccessibilityLabel("Ana is typing")),
						caption("Ana is typing"),
					),
				),
				prose("The dots are driven by two hooks inside the widget: which dot is dark, and the "+
					"interval that moves it. A widget that owns hooks must render on every pass, so the "+
					"switch is the Visible field and never a core.If around the widget. Hidden is "+
					"Display none, and the interval is hooks.UseIntervalWhile, so a hidden indicator "+
					"costs no render passes."),
				prose("Go only picks which dot is dark. core.Transition declares the fade, and the "+
					"platform draws every frame of it. What fades is each dot's core.Opacity, between 0.4 "+
					"and 1. With reduced motion on, the hosts drop the Transition and the dots step "+
					"instead, which is still readable as a blink."),

				codeBlock(`comps.ReactionBar{
    Reactions: []comps.Reaction{
        {Emoji: "👍", Count: 3, Mine: true, Label: "thumbs up"},
        {Emoji: "🎉", Count: 1, Label: "party popper"},
    },
    OnToggle: func(emoji string) { toggle(emoji) },
}`),
				demoPanel("Tap a chip. The 🤔 has no count, so it is not drawn until somebody holds it.",
					comps.MessageBubble{Text: "Bubbles are a widget now.", Sender: "Ana", Time: "10:41",
						Style: []core.StyleProp{core.MarginBottom(6)}},
					comps.ReactionBar{Reactions: reactions.Get(), OnToggle: toggle},
				),
				prose("A reaction is server state: other people move the same count. So the bar draws "+
					"what it is given and reports the tap, and the toggle above is the caller's. It is a "+
					"comps.ChipStrip of comps.Chips underneath, and adds the spoken name. No platform "+
					"names an emoji reliably, so Reaction.Label does: \"thumbs up, 3 reactions\". The "+
					"reader's own is the chip's selected state, not words in its name."),

				codeBlock(`comps.Poll{
    Question: "Tabs or spaces?",
    Options: []comps.PollOption{
        {Label: "Tabs", Votes: 2, Mine: true},
        {Label: "Spaces", Votes: 1},
    },
    OnVote: func(i int) { castVote(i) },
}`),
				demoPanel("Vote, and the buttons become result bars. Reset asks again.",
					comps.Poll{
						Question: "Tabs or spaces?",
						Options:  options(voted.Get()),
						OnVote:   voted.Set,
					},
					comps.Button{
						Label:    "Reset the poll",
						Emphasis: comps.EmphasisGhost,
						Disabled: voted.Get() < 0,
						OnTap:    func() { voted.Set(-1) },
					},
				),
				demoPanel("A closed poll: ShowResults draws the bars with no vote of yours. One vote each, and the shares still total 100.",
					comps.Poll{
						Question:    "Tabs or spaces? (closed)",
						Options:     options(-1),
						ShowResults: true,
					},
				),
				prose("Three options with a vote each round to 33 + 33 + 33, and a poll that totals 99% "+
					"looks broken. Poll apportions the shares by the largest-remainder method: every "+
					"share is rounded down, and the points left over go to the largest remainders, "+
					"earliest first. The bars are drawn from the true fractions."),
				prose("The reader's vote is PollOption.Mine, not an index on the Poll. An int field "+
					"would default to 0, and a poll written without it would open already voted for "+
					"its first option. After the vote each option is one spoken stop: \"Tabs, 50 "+
					"percent, 2 votes, your choice\"."),
				keyPoints(
					"TypingIndicator owns hooks: always render it and switch it with Visible, never with core.If.",
					"Its dots fade by core.Opacity under core.Transition; Go steps the phase, the platform draws the frames.",
					"ReactionBar and Poll hold no state: counts and votes are server data, so the caller toggles and records.",
					"Reaction.Label names the emoji; a count of zero is not drawn; Trailing is the slot for an add chip.",
					"Poll's shares always total 100 (largest remainder), and the reader's choice is PollOption.Mine.",
				),
			)
		},
	}
}

// tutorialFiles is lesson 4.35's tree: a small project, three levels deep,
// with one empty folder so that TreeNode.Branch has something to show.
//
//	docs/
//	  guide.md
//	  api/
//	    core.md
//	    comps.md
//	src/
//	  main.go
//	  app.go
//	assets/          empty: a branch by its flag
//	README.md
//
// IDs are paths, which are unique in the whole tree by construction. That is
// the rule TreeView asks for, since its Expanded map has one key space.
func tutorialFiles() []comps.TreeNode {
	folder, file := core.Text("📁"), core.Text("📄")
	return []comps.TreeNode{
		{ID: "docs", Label: "docs", Leading: folder, Children: []comps.TreeNode{
			{ID: "docs/guide.md", Label: "guide.md", Leading: file},
			{ID: "docs/api", Label: "api", Leading: folder, Children: []comps.TreeNode{
				{ID: "docs/api/core.md", Label: "core.md", Leading: file},
				{ID: "docs/api/comps.md", Label: "comps.md", Leading: file},
			}},
		}},
		{ID: "src", Label: "src", Leading: folder, Children: []comps.TreeNode{
			{ID: "src/main.go", Label: "main.go", Leading: file},
			{ID: "src/app.go", Label: "app.go", Leading: file},
		}},
		{ID: "assets", Label: "assets", Leading: folder, Branch: true},
		{ID: "README.md", Label: "README.md", Leading: file},
	}
}

// 4.35 — Phase 3 of the fourth low-hanging-fruit round: the two structure
// widgets. They share a lesson because they share a rule, and it is the
// converse of 4.34's: there the indicator owned hooks and so had to always
// render; here both widgets render only part of what they were given (the
// open branches, the current step), and so they and their content must own
// none.
//
// Appended at the end of the chapter for the reason 4.25 was.
func lessonTreeAndWizard() Lesson {
	return Lesson{
		Title:   "Trees and wizards",
		Summary: "comps.TreeView and Wizard: structures that draw only part of their data, so the caller holds all the state.",
		Body: func(ctx *core.Context) core.View {
			// Every hook of the lesson, here, on every pass. The wizard's
			// step bodies below read these and declare nothing: that is the
			// lesson's second point, acted out.
			open := core.NewState(ctx, map[string]bool{"docs": true})
			chosen := core.NewState(ctx, "docs/guide.md")
			step := core.NewState(ctx, 0)
			name := core.NewState(ctx, "")
			note := core.NewState(ctx, "")
			placed := core.NewState(ctx, false)

			// Copy, flip, Set: the map in the slot is never written to.
			toggle := func(id string) {
				next := maps.Clone(open.Get())
				next[id] = !next[id]
				open.Set(next)
			}

			status := "Nothing chosen"
			if chosen.Get() != "" {
				status = "Chosen: " + chosen.Get()
			}

			steps := []comps.WizardStep{
				{
					Title: "Your name",
					Body: comps.FormField{
						Label: "Name",
						Hint:  "Next stays disabled until this has something in it.",
						Input: core.Input(name.Get(), "Ada Lovelace", name.Set),
					},
					Blocked: strings.TrimSpace(name.Get()) == "",
				},
				{
					Title: "Gift note",
					Body: comps.FormField{
						Label: "Note",
						Hint:  "Optional: while it is empty the button reads Skip.",
						Input: core.Input(note.Get(), "Happy birthday!", note.Set),
					},
					Optional: true,
					Blocked:  strings.TrimSpace(note.Get()) == "",
				},
				{
					Title: "Review",
					Body: comps.KeyValueList{Rows: []comps.KeyValue{
						{Key: "Name", Value: name.Get()},
						{Key: "Note", Value: cmp.Or(strings.TrimSpace(note.Get()), "(none)")},
					}},
				},
			}

			return core.Column(
				core.Gap(14),
				prose("Two widgets for structure: a hierarchy that opens and shuts, and a flow that moves "+
					"through steps. Both draw only a part of what they are given, the open branches "+
					"or the current step, and that one fact decides where all of their state lives."),

				codeBlock(`open := core.NewState(ctx, map[string]bool{"docs": true})

comps.TreeView{
    Label:    "Project files",
    Nodes:    files,                 // []comps.TreeNode{ID, Label, Leading, Children}
    Expanded: open.Get(),
    OnToggle: func(id string) {      // copy, flip, Set
        next := maps.Clone(open.Get())
        next[id] = !next[id]
        open.Set(next)
    },
    Selected: chosen.Get(),
    OnSelect: chosen.Set,
}`),
				demoPanel("Tap a folder to open it and a file to choose it. assets is empty, and still a folder.",
					comps.TreeView{
						Label:    "Project files",
						Nodes:    tutorialFiles(),
						Expanded: open.Get(),
						OnToggle: toggle,
						Selected: chosen.Get(),
						OnSelect: chosen.Set,
					},
					core.Text(status,
						core.UseStyle(ctx.Theme().Typography.Caption),
						core.TextColor(ctx.Theme().Colors.TextSecondary),
						core.AccessibilityRole(core.RoleStatus),
					),
					core.Row(
						core.Padding(0),
						core.Gap(8),
						comps.Button{Label: "Collapse all", Emphasis: comps.EmphasisGhost,
							OnTap: func() { open.Set(map[string]bool{}) }},
						comps.Button{Label: "Reveal core.md", Emphasis: comps.EmphasisGhost,
							OnTap: func() {
								open.Set(map[string]bool{"docs": true, "docs/api": true})
								chosen.Set("docs/api/core.md")
							}},
					),
				),
				prose("The tree keeps nothing. Which folders are open is the map you pass, and a tap only "+
					"reports an ID. That is why the two buttons under it are one line each: collapsing "+
					"everything is an empty map, and revealing a file is a map of its ancestors. A widget "+
					"that kept the map in a hook could offer neither."),
				prose("A row has one meaning: a branch toggles and a leaf selects. IDs must be unique in "+
					"the whole tree, since the map has one key space, so the lesson uses paths. A shut "+
					"branch's children are not rendered at all, and a tree costs what its open part "+
					"costs. A node with no Children is a leaf unless it sets Branch, which is how an "+
					"empty or not-yet-loaded folder keeps its chevron."),
				prose("To a screen reader it is nested lists: each item carries its depth, and each row "+
					"inside it is a button. A branch says \"docs, expanded\" and the chosen file says it "+
					"is current. ARIA's tree role promises arrow keys that no target supplies yet, so "+
					"the widget does not claim it."),

				codeBlock(`name := core.NewState(ctx, "")      // every step's state, above the wizard
step := core.NewState(ctx, 0)

comps.Wizard{
    Label: "Order",
    Steps: []comps.WizardStep{
        {Title: "Your name", Body: nameField(name), Blocked: name.Get() == ""},
        {Title: "Gift note", Body: noteField(note), Optional: true, Blocked: note.Get() == ""},
        {Title: "Review",    Body: summary(name, note)},
    },
    Current:  step.Get(),
    OnChange: step.Set,
    OnFinish: placeOrder,
}`),
				demoPanel("Next is disabled until there is a name. The note is optional, so its Next reads Skip while it is empty.",
					core.IfElse(placed.Get(),
						core.Column(
							core.Padding(0),
							core.Gap(8),
							comps.Banner{Text: "Order placed for " + name.Get() + ".", Variant: comps.VariantSuccess},
							comps.Button{Label: "Start again", Emphasis: comps.EmphasisGhost, OnTap: func() {
								placed.Set(false)
								step.Set(0)
								name.Set("")
								note.Set("")
							}},
						),
						comps.Wizard{
							Label:    "Order",
							Steps:    steps,
							Current:  step.Get(),
							OnChange: step.Set,
							OnFinish: func() { placed.Set(true) },
						},
					),
				),
				prose("Only the current step's Body is rendered. So a Body must not own a hook: it would be "+
					"called on some passes and not on others, and every hook after it would shift onto a "+
					"neighbour's slot the moment the step changed. The three fields here are declared "+
					"once, at the top of the lesson, and each Body is handed values. It is also what a "+
					"wizard needs. Go Back from the note and the name is still there, because the name "+
					"never belonged to the step that left."),
				prose("The Wizard can sit inside a core.IfElse, as it does here, for the same reason: it "+
					"holds no hook itself. Back is absent on the first step and not disabled, since "+
					"there is nothing it could ever do. Done steps in the strip are tappable and later "+
					"ones are not, so a Blocked step cannot be jumped past. The last step's button calls "+
					"OnFinish and never OnChange."),
				prose("The footer is drawn inline. For a long form, set DetachFooter and hand "+
					"wizard.Footer() to comps.Screen's Footer, which pins it above the keyboard."),
				keyPoints(
					"TreeView and Wizard render part of their data, so neither they nor their content may own hooks.",
					"TreeView's open state is your map[string]bool: copy, flip, Set. IDs are unique tree-wide.",
					"A branch toggles, a leaf selects; TreeNode.Branch keeps an empty folder a folder.",
					"Hold every Wizard step's state above the Wizard; Blocked disables Next, Optional turns it into Skip.",
					"Wizard.Footer() with DetachFooter lifts the buttons into Screen.Footer.",
				),
			)
		},
	}
}

// tutorialCandles is a fortnight of prices for 4.36: mostly rising, with two
// falling days and one doji (day 6, open = close), so every kind of candle
// the lesson talks about is on screen.
var tutorialCandles = []comps.Candle{
	{Open: 102, High: 106, Low: 101, Close: 105},
	{Open: 105, High: 109, Low: 104, Close: 108},
	{Open: 108, High: 110, Low: 103, Close: 104},
	{Open: 104, High: 107, Low: 102, Close: 106},
	{Open: 106, High: 112, Low: 105, Close: 111},
	{Open: 111, High: 114, Low: 108, Close: 111},
	{Open: 111, High: 113, Low: 106, Close: 107},
	{Open: 107, High: 111, Low: 106, Close: 110},
	{Open: 110, High: 116, Low: 109, Close: 115},
	{Open: 115, High: 118, Low: 113, Close: 117},
}

var tutorialCandleDays = []string{"1", "2", "3", "4", "5", "6", "7", "8", "9", "10"}

// tutorialPeaks stands in for what a server would compute from a recording:
// 120 loudness samples shaped like speech, phrases swelling and dying away
// with pauses between them. It is a fixed function of the index and not
// random, so the lesson draws the same strip on every run and every target,
// which the screenshots depend on. One sample is a transient at full scale,
// for the lesson's point about downsampling by maximum.
func tutorialPeaks() []float64 {
	peaks := make([]float64, 120)
	for i := range peaks {
		x := float64(i)
		// A slow envelope (the phrases) times a fast ripple (the syllables).
		phrase := math.Abs(math.Sin(x / 19))
		ripple := 0.55 + 0.45*math.Sin(x*1.7)*math.Cos(x/3.1)
		peaks[i] = math.Min(1, 0.08+0.8*phrase*ripple)
	}
	peaks[67] = 1
	return peaks
}

// 4.36 — Phase 4 of the fourth low-hanging-fruit round: four charts over the
// scaffolding 4.20 and 4.28 built. The lesson is organised around the one
// question each chart had to answer that the earlier ones did not, since the
// drawing itself is by now routine: what colour means up, what to do with
// data that breaks the shape's promise, how to label a rim with no text in
// the canvas, and how to keep a bar round in a stretched drawing.
//
// Appended at the end of the chapter for the reason 4.25 was.
func lessonFourMoreCharts() Lesson {
	return Lesson{
		Title:   "Four more charts",
		Summary: "comps.CandlestickChart, FunnelChart, RadarChart and Waveform, and the one decision each had to make.",
		Body: func(ctx *core.Context) core.View {
			convention := core.NewState(ctx, 0)
			rates := core.NewState(ctx, true)
			rival := core.NewState(ctx, false)
			played := core.NewState(ctx, 0.3)
			win := hooks.UseWindow(ctx)

			t := ctx.Theme()
			up, down := t.Colors.Success, t.Colors.Error
			if convention.Get() == 1 {
				up, down = down, up
			}

			profiles := []comps.ChartSeries{{Name: "Ade", Values: []float64{8, 6, 7, 9, 5}}}
			if rival.Get() {
				profiles = append(profiles, comps.ChartSeries{Name: "Bo", Values: []float64{5, 9, 6, 4, 8}})
			}

			// The strip's width is the window less everything between the
			// window's edge and the strip: the screen's padding, the demo
			// panel's border and padding, both sides, measured at 118 px in
			// a 414 px window and rounded up, since a bar too few only
			// loosens the strip while a bar too many starts closing the gaps.
			//
			// The window is capped at a phone's first. In the browser's
			// two-pane layout (split.go) the demos sit on a phone-sized
			// screen inside a desktop-sized window, and the window's real
			// width would ask for three times the bars the strip can hold.
			//
			// Zero before a window is reported (a headless test, the static
			// export) leaves Bars unset, and the widget's own cap applies.
			strip := comps.Waveform{Peaks: tutorialPeaks(), Progress: played.Get()}
			if win.Width > 0 {
				strip.Bars = strip.BarsFor(math.Min(win.Width, 430) - 120)
			}

			return core.Column(
				core.Gap(14),
				prose("4.20 and 4.28 built the chart scaffolding: a stretched core.Canvas, a nice-number "+
					"scale, labels placed by flex arithmetic because the canvas holds no text, and one "+
					"spoken sentence per chart. Four more charts sit on it, and with the drawing "+
					"routine, what is left of each is one decision."),
				prose("A candlestick is a wick from the period's low to its high and a body from open to "+
					"close. Its axis does not start at zero: a candle states a range, not a length, and "+
					"prices near 100 on an axis from 0 are a row of slivers. The decision is the colour. "+
					"Green for a rise is the Western convention; markets in China, Japan and Korea print "+
					"red for a rise. So both are fields, defaulting to the theme's Success and Error."),
				codeBlock(`comps.CandlestickChart{
    Subject: "ACME",
    Candles: candles,   // []comps.Candle{Open, High, Low, Close}
    Labels:  days,
    UpColor: up, DownColor: down,   // "" means Success and Error
    ShowLegend: true,
}`),
				demoPanel("Swap the convention. Day 6 closed where it opened: its body is one px, not nothing.",
					comps.SegmentedControl{
						Labels:   []string{"Green rises", "Red rises"},
						Selected: convention.Get(),
						OnSelect: convention.Set,
					},
					comps.CandlestickChart{
						Subject:    "ACME",
						Candles:    tutorialCandles,
						Labels:     tutorialCandleDays,
						UpColor:    up,
						DownColor:  down,
						ShowLegend: true,
					},
				),
				prose("Colour is the only drawn difference between a rise and a fall, so the spoken "+
					"sentence ends by counting them. The one-px body is exact without measuring: the "+
					"plot is Height px tall and the stretched canvas maps y linearly, so a px is "+
					"100/Height viewBox units."),
				prose("A funnel's band is as wide at the top as its own stage and at the bottom as the "+
					"next, so the slope is the drop. The names, values and rates are columns of Text "+
					"cut into the same px bands as the drawing; the rates column is shifted half a "+
					"band, which puts each rate on the boundary it describes."),
				demoPanel("The rate is each stage as a share of the one before.",
					comps.SwitchRow{Title: "Show rates", On: rates.Get(), OnToggle: rates.Set},
					comps.FunnelChart{
						Subject:   "Checkout",
						ShowRates: rates.Get(),
						Stages: []comps.FunnelStage{
							{Label: "Visited", Value: 1200},
							{Label: "Signed up", Value: 744},
							{Label: "Added to cart", Value: 310},
							{Label: "Paid", Value: 93},
						},
					},
				),
				prose("The decision was a stage larger than the one before it. It is drawn as given and "+
					"its rate reads over 100%, because real funnels have them (people re-entering at a "+
					"later step) and clamping would misreport the data. But the usual cause is stages "+
					"passed out of order, so debug builds report it until AllowIncrease says it is "+
					"meant. The stages are one hue fading, not the categorical palette: they are one "+
					"population at successive moments, not different kinds of thing."),
				prose("A radar's labels sit round a rim, and a ZStack offers nine named places: enough "+
					"for Compass's four letters, not for five axes. So each label is a layer the stack "+
					"centres, moved by core.Translate in px from the same trigonometry as the drawing. "+
					"The px are exact because Size is: the canvas is a Size px square, so Go knows where "+
					"every spoke ends. Labels on the right are start-aligned and on the left "+
					"end-aligned, so text always runs away from the drawing."),
				demoPanel("Add a second profile. Outlines stay readable where fills would pile up.",
					comps.SwitchRow{Title: "Compare with Bo", On: rival.Get(), OnToggle: rival.Set},
					comps.RadarChart{
						Subject: "Player",
						Axes:    []string{"Speed", "Power", "Stamina", "Skill", "Vision"},
						Series:  profiles,
						Max:     10,
						Filled:  !rival.Get(),
					},
				),
				prose("The grid is polygons, not circles, so a gridline between two spokes is straight, "+
					"as the data's edge is. Every axis shares Max: measures on different scales are the "+
					"caller's to normalise. Fewer than three axes enclose nothing, and debug builds say so."),
				prose("A waveform's peaks come from the caller. No host decodes audio for Go, so a "+
					"server or a build step computes them and ships them beside the file. The decision "+
					"here was how to keep a bar round in a drawing stretched to an unknown width, where "+
					"a rounded rectangle's corners would stretch into ellipses. A Canvas stroke is "+
					"never scaled, so each bar is a line: its thickness is the stroke's width and its "+
					"round ends are the stroke's caps, exact px on every target."),
				codeBlock(`w := comps.Waveform{Peaks: peaks, Progress: position / duration}
w.Bars = w.BarsFor(stripWidth)   // how many 3px bars fit; you know your insets, Go cannot measure`),
				demoPanel("120 peaks into however many bars fit. The spike two thirds along survives: a bucket keeps its maximum, never its mean.",
					strip,
					core.Row(
						core.Padding(0),
						core.Gap(8),
						comps.Button{Label: "Back", Emphasis: comps.EmphasisOutlined, OnTap: func() {
							played.Set(math.Max(0, played.Get()-0.1))
						}},
						comps.Button{Label: "Forward", Emphasis: comps.EmphasisOutlined, OnTap: func() {
							played.Set(math.Min(1, played.Get()+0.1))
						}},
					),
					caption(fmt.Sprintf("%.0f%% played.", played.Get()*100)),
				),
				prose("It is a picture, not a control. Seeking by tapping the strip needs the tap's x "+
					"position, which no event carries to Go. So AudioPlayer takes the peaks as its "+
					"Waveform field, draws the strip above its seek bar, hides it from screen readers "+
					"(the slider already speaks the position) and keeps the slider for seeking."),
				keyPoints(
					"CandlestickChart: the axis brackets the prices; UpColor and DownColor because the convention is regional; a doji keeps a 1px body.",
					"FunnelChart: a growing stage is drawn, never clamped, and reported in debug until AllowIncrease.",
					"RadarChart: rim labels are centred ZStack layers moved by core.Translate in px, exact because Size is.",
					"Waveform: bars are round-capped strokes, which never stretch; BarsFor sizes it; buckets keep their maximum.",
					"AudioPlayer.Waveform draws the strip above the seek bar. It shows progress and does not seek.",
				),
			)
		},
	}
}

// budgetSheet is lesson 4.37's state: the rows, and an id per row. The two
// slices move together, which is why they are one value in one state slot: an
// insert that set them in two Sets could be seen, for one pass, with a row the
// ids do not cover.
type budgetSheet struct {
	Rows [][]string
	IDs  []string
	Next int // the next id to mint; never reused, so a deleted row's id stays dead
}

// with returns the sheet with one cell replaced. Copied, not mutated: the
// slot would otherwise hold the same slices and the undo stack would hold
// them too, all of them showing the newest value.
func (b budgetSheet) with(r, c int, v string) budgetSheet {
	rows := make([][]string, len(b.Rows))
	for i := range b.Rows {
		rows[i] = slices.Clone(b.Rows[i])
	}
	rows[r][c] = v
	b.Rows = rows
	return b
}

// insert adds an empty row below row after; remove takes row r out.
func (b budgetSheet) insert(after int) budgetSheet {
	b.Rows = slices.Insert(slices.Clone(b.Rows), after+1, []string{"", "", "", "false"})
	b.IDs = slices.Insert(slices.Clone(b.IDs), after+1, "row"+strconv.Itoa(b.Next))
	b.Next++
	return b
}

func (b budgetSheet) remove(r int) budgetSheet {
	b.Rows = slices.Delete(slices.Clone(b.Rows), r, r+1)
	b.IDs = slices.Delete(slices.Clone(b.IDs), r, r+1)
	return b
}

// totals sums the Amount column, all of it and the unpaid part. A cell the
// grid let through is "" or a number, so the parse cannot fail on anything
// but an empty cell, which counts as nothing.
func (b budgetSheet) totals() (all, unpaid float64) {
	for _, row := range b.Rows {
		amount, _ := strconv.ParseFloat(row[2], 64)
		all += amount
		if paid, _ := strconv.ParseBool(row[3]); !paid {
			unpaid += amount
		}
	}
	return all, unpaid
}

// tutorialBudget is the sheet 4.37 opens on.
func tutorialBudget() budgetSheet {
	return budgetSheet{
		Rows: [][]string{
			{"Rent", "Home", "1200", "true"},
			{"Groceries", "Food", "310.5", "false"},
			{"Bus pass", "Travel", "64", "true"},
			{"Dinner out", "Food", "48", "false"},
		},
		IDs:  []string{"row0", "row1", "row2", "row3"},
		Next: 4,
	}
}

// dollars is the Amount column's Format: display only, so the editor opens on
// "310.5" and the cell reads "$310.50".
func dollars(v string) string {
	f, err := strconv.ParseFloat(v, 64)
	if err != nil {
		return v
	}
	return "$" + strconv.FormatFloat(f, 'f', 2, 64)
}

// 4.37 — Phase 5 of the fourth low-hanging-fruit round: EditableGrid. The
// lesson is a budget, because a budget uses every kind of cell once and has a
// derived value (the total) that shows where formulas went: into the caller.
//
// Appended at the end of the chapter for the reason 4.25 was.
func lessonEditableGrid() Lesson {
	return Lesson{
		Title:   "A grid you can type into",
		Summary: "comps.EditableGrid: one editor at a time, a commit per cell, and the rows stay yours.",
		Body: func(ctx *core.Context) core.View {
			sheet := core.NewState(ctx, tutorialBudget())
			// The undo stack the widget does not have: every commit arrives
			// here, so the history is one slice of the values it replaced.
			undo := core.NewState(ctx, []budgetSheet(nil))

			apply := func(next budgetSheet) {
				undo.Set(append(slices.Clone(undo.Get()), sheet.Get()))
				sheet.Set(next)
			}
			b := sheet.Get()
			all, unpaid := b.totals()

			return core.Column(
				core.Gap(14),
				prose("DataTable shows rows. This edits cells. It looks like a table of text fields and "+
					"is not one: every cell is a box of text, and only the cell you tap becomes a "+
					"field. A sheet of real fields would be hundreds of native inputs, each a tab "+
					"stop, each with the arrow keys its caret wants and the grid wants too."),

				codeBlock(`comps.EditableGrid{
    Label: "Budget",
    Columns: []comps.GridColumn{
        {Title: "Item", Weight: 2},
        {Title: "Category", Kind: comps.GridChoice, Options: []string{"Home", "Food", "Travel"}},
        {Title: "Amount", Kind: comps.GridNumber, Format: dollars, Validate: notNegative},
        {Title: "Paid", Kind: comps.GridBool, Width: 56},
    },
    Rows:     sheet.Get().Rows,                       // [][]string: the caller parses
    Key:      func(r int) string { return ids[r] },   // identity across insert and delete
    OnChange: func(r, c int, v string) { apply(sheet.Get().with(r, c, v)) },
    OnInsertRow: func(after int) { apply(sheet.Get().insert(after)) },
    OnDeleteRow: func(r int)     { apply(sheet.Get().remove(r)) },
    MinWidth: 460,                                    // narrower than this, scroll sideways
    Style: []core.StyleProp{core.Height("232px")},    // a List with no height is not lazy
}`),
				demoPanel("Tap a cell, type, press return. Try -5 in Amount. A row number opens that row's menu.",
					comps.EditableGrid{
						Label: "Budget",
						Columns: []comps.GridColumn{
							{Title: "Item", Weight: 2},
							{Title: "Category", Kind: comps.GridChoice, Weight: 1.6,
								Options: []string{"Home", "Food", "Travel"}},
							{Title: "Amount", Kind: comps.GridNumber, Weight: 1.4, Format: dollars,
								Validate: func(v string) string {
									if strings.HasPrefix(v, "-") {
										return "An amount cannot be negative."
									}
									return ""
								}},
							{Title: "Paid", Kind: comps.GridBool, Width: 44},
						},
						Rows:        b.Rows,
						Key:         func(r int) string { return b.IDs[r] },
						OnChange:    func(r, c int, v string) { apply(sheet.Get().with(r, c, v)) },
						OnInsertRow: func(after int) { apply(sheet.Get().insert(after)) },
						OnDeleteRow: func(r int) { apply(sheet.Get().remove(r)) },
						// Four columns and the row numbers want more than a
						// phone's demo panel has, so the sheet scrolls sideways
						// rather than drawing "$12…".
						MinWidth: 460,
						Compact:  true,
						Style:    []core.StyleProp{core.Height("232px")},
					},
					core.Text(fmt.Sprintf("Total %s, of which %s is unpaid.",
						dollars(strconv.FormatFloat(all, 'f', -1, 64)),
						dollars(strconv.FormatFloat(unpaid, 'f', -1, 64))),
						core.FontWeight(core.Bold),
						core.AccessibilityRole(core.RoleStatus),
					),
					core.Row(
						core.Padding(0),
						core.Gap(8),
						comps.Button{Label: fmt.Sprintf("Undo (%d)", len(undo.Get())),
							Emphasis: comps.EmphasisOutlined, Disabled: len(undo.Get()) == 0,
							OnTap: func() {
								stack := undo.Get()
								sheet.Set(stack[len(stack)-1])
								undo.Set(slices.Clone(stack[:len(stack)-1]))
							}},
						comps.Button{Label: "Reset", Emphasis: comps.EmphasisGhost, OnTap: func() {
							sheet.Set(tutorialBudget())
							undo.Set(nil)
						}},
					),
				),
				prose("The draft is the grid's and the data is yours. While you type, nothing reaches "+
					"OnChange: no application wants a half-typed cell. The return key commits and "+
					"moves down a row, as a spreadsheet does; tapping elsewhere commits and stays; "+
					"the ✕ at the end of the cell throws the draft away. There is no Escape, because "+
					"key events do not reach Go, and the ✕ is what that costs."),
				prose("A commit that Validate refuses never reaches you. The cell stays open with a red "+
					"border and the message appears under the grid, in an alert a screen reader "+
					"announces. A Number column refuses what is not a number before your Validate is "+
					"asked. Format is display only: the Amount cell reads $310.50 and its editor opens "+
					"on 310.5, which is also what OnChange reports."),
				prose("Cells are strings and the grid is not generic. A typed grid needs a getter and a "+
					"setter per column, which is a heavy API for what a text field produces anyway. "+
					"Kind picks the editor: Text and Number open the field, Bool toggles on one tap, "+
					"and Choice is the platform's own picker, always there, so choosing is one tap "+
					"and not two."),
				prose("The total and the Undo button are the lesson's, not the widget's. There are no "+
					"formulas: a formula engine is a parser and a dependency graph, which is an "+
					"application. But every commit passes through your OnChange, so a derived value is "+
					"a loop over your rows, and undo is a slice of the sheets you replaced. Key is what "+
					"makes the row menu safe. Without it rows are keyed by index, and deleting row 1 "+
					"hands row 2's node to row 3's data."),
				prose("It costs what it draws. core.List windows the rows on a phone, but Go still "+
					"builds every cell each pass, and each keystroke in the editor is a pass. About "+
					"five thousand cells is comfortable; past that, page the rows. Columns are never "+
					"windowed and the row numbers scroll away with the rest, so on a phone the honest "+
					"advice is few columns. MinWidth lets a wide sheet scroll sideways instead of "+
					"squeezing."),
				keyPoints(
					"One editor at a time: cells are text until tapped. Return commits and moves down, a tap elsewhere commits, ✕ discards.",
					"OnChange fires once per commit, only for a changed value, and never with a value Validate refused.",
					"Rows is [][]string. Kind chooses the editor; Format is display only; the caller parses.",
					"Key gives rows an identity, so an insert or delete cannot re-pair rows or move an open editor.",
					"No formulas and no undo inside: every commit passes through the caller, so both are a few lines there.",
					"EditableGrid holds hooks: render it in a stable position, never inside core.If.",
				),
			)
		},
	}
}
