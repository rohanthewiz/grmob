# Package comps — Overlays & feedback

```go
import "github.com/rohanthewiz/grmob/comps"
```

Dialogs, action sheets, menus, snackbars, banners, progress, spinners, skeletons and empty states.

One of 6 topic pages of [package comps](comps.md), which has the package overview and an index of every topic. This page documents the declarations in `comps/dialog.go`, `comps/action_sheet.go`, `comps/menu.go`, `comps/snackbar.go`, `comps/banner.go`, `comps/progress_bar.go`, `comps/spinner.go`, `comps/skeleton.go`, `comps/empty_state.go`.

## Index

- [Constants](#constants) — `SnackbarDuration`
- [`type ActionSheet`](#type-actionsheet)
    - [`func (ActionSheet) Render`](#func-actionsheet-render)
- [`type Banner`](#type-banner)
    - [`func (Banner) Render`](#func-banner-render)
- [`type Dialog`](#type-dialog)
    - [`func (Dialog) Render`](#func-dialog-render)
- [`type DialogAction`](#type-dialogaction)
- [`type EmptyState`](#type-emptystate)
    - [`func (EmptyState) Render`](#func-emptystate-render)
- [`type Menu`](#type-menu)
    - [`func (Menu) Render`](#func-menu-render)
- [`type ProgressBar`](#type-progressbar)
    - [`func (ProgressBar) Render`](#func-progressbar-render)
- [`type SheetAction`](#type-sheetaction)
- [`type Skeleton`](#type-skeleton)
    - [`func (Skeleton) Render`](#func-skeleton-render)
- [`type Snackbar`](#type-snackbar)
    - [`func (Snackbar) Render`](#func-snackbar-render)
- [`type Spinner`](#type-spinner)
    - [`func (Spinner) Render`](#func-spinner-render)
- [`type SpinnerSize`](#type-spinnersize)

## Constants

SnackbarDuration is the default time a Snackbar stays up: long enough to read a short sentence and reach an Undo, and within Material's four-to-ten second range for a snackbar with an action.

```go
const SnackbarDuration = 4 * time.Second
```

<small>[comps/snackbar.go:110](https://github.com/rohanthewiz/grmob/blob/master/comps/snackbar.go#L110)</small>

## Types

### type ActionSheet

```go
type ActionSheet struct {
	// Visible is the caller's open/closed state.
	Visible bool

	// Title names the sheet. It is drawn as the card's heading and is the
	// sheet's accessible name. Empty draws no heading.
	Title string

	// Actions are drawn top to bottom, each a full-width ghost button.
	Actions []SheetAction

	// Cancel is the label of the separate dismiss button under the actions.
	// Empty omits it; the scrim and the platform gestures still dismiss.
	Cancel string

	// OnDismiss is called for a scrim tap, for Cancel, and after any action.
	// Nil makes the scrim inert and the sheet close only when the caller
	// flips Visible.
	OnDismiss func()

	// Backdrop overrides the scrim colour. Empty keeps core.Modal's default.
	Backdrop string

	// Style is applied to the card after the widget's own props, so it can
	// cap the width (core.MaxWidth) or replace the accessible name.
	Style []core.StyleProp
}
```

ActionSheet is the "Share / Copy link / Delete" list that rises from the bottom edge: a short column of full-width actions and a separate Cancel, drawn over the screen in a core.Modal.

	comps.ActionSheet{
	    Visible: open.Get(),
	    Title:   "Note",
	    Actions: []comps.SheetAction{
	        {Label: "Share", OnTap: share},
	        {Label: "Delete", Variant: comps.VariantError, OnTap: del},
	    },
	    Cancel:    "Cancel",
	    OnDismiss: func() { open.Set(false) },
	}

#### The placement question, and what answered it

core.Modal has no placement prop, and the four hosts do not agree on where a Modal's content goes:

	target    Modal chassis                           content lands
	───────   ─────────────────────────────────────   ─────────────────────
	web ×2    fixed inset-0 flex column, align and    centred
	          justify center (htmlout modalChassis,
	          styleFromGrMob)
	Compose   Dialog window > Column(fillMaxWidth)    centred window;
	          > ColumnChildren (FlexGrow → weight)    children weighted
	SwiftUI   .sheet, detents medium/large >          already the bottom
	          VStack > PlainChildren (grow ignored)   sheet, top of it

So the widget puts two children in the Modal instead of one:

	┌ Modal ──────────────────────────────────────────┐
	│ ┌ Box FlexGrow(1)  filler, tap = OnDismiss ───┐ │
	│ │                                             │ │  ← pushes the card
	│ └─────────────────────────────────────────────┘ │    down on web and
	│ ┌ Card  Width 100%  AccessibilityLabel(Title) ┐ │    Compose; zero
	│ │ Title                                       │ │    height on iOS
	│ │ [ Share                                   ] │ │
	│ │ [ Delete                                  ] │ │
	│ │ ─────────────────────────────────────────── │ │
	│ │ [ Cancel                                  ] │ │
	│ └─────────────────────────────────────────────┘ │
	└─────────────────────────────────────────────────┘

Each host then does the right thing with no host change:

  - Web: the filler grows along the overlay's column and the card sits on the bottom edge. The overlay's align-items:center would shrink both children to their content, so the filler takes AlignSelf(stretch) (a web-only prop, which is exactly where it is needed) and the card takes Width("100%").
  - Compose: the filler is weighted, so the Column fills the dialog window's height and the card is last in it. The card paints a background, which is what tells GrMobModal not to add its own white surface around the whole column.
  - SwiftUI: the filler has no content and grow is not read inside a sheet, so it is zero-height and the card is the sheet's content. The card's background becomes the sheet's surface (GrMobModal reads the first child that paints one; the filler paints none).

The filler carries the scrim's job. On the web a backdrop tap is only reported when it lands on the overlay element itself, and on Compose the filler is inside the dialog window, so a tap above the card would do nothing unless the filler reports it. It is hidden from assistive technology: the Cancel action is the accessible way out, as the scrim is.

Not device-verified: the Compose and SwiftUI rows above come from reading Renderer.kt and Renderer.swift, not from a run on hardware.

#### Actions are buttons, not a listbox

The plan that proposed this widget suggested RoleListBox with RoleOption items. That pair is for choosing a value: an option states selected or not selected, and a reader would announce "Delete, not selected". An action sheet is a set of commands. ARIA's word for that is a menu, which core.Role does not have, so each action is a real comps.Button (a button on every target) inside the labelled card, and the Modal chassis supplies the dialog semantics as it does for Dialog.

#### Picking an action closes the sheet

Unlike Dialog's Confirm, an action tap calls the action's OnTap and then OnDismiss. A sheet is a menu: UIKit's action sheet and Material's bottom sheet menu both close on selection, and a caller would otherwise write open.Set(false) into every handler. An action whose follow-up needs another question ("Delete?") opens a Dialog from its OnTap; both are Modals, and the same render pass closes one and opens the other.

Cancel calls OnDismiss, like a scrim tap. There is no Cancel.OnTap to fall back from, because a cancel that does something other than dismiss is an action and belongs in Actions.

#### Theme roles read

	Panel          Components.Card, through comps.Card
	Title          Typography.Subtitle, bold (Card's title treatment)
	Action ink     Variant.OnLight — Colors.PrimaryOnLight, or ErrorOnLight
	               for a destructive action (ghost Button)
	Cancel         Colors.PrimaryOnLight, outlined
	Rule           Colors.Border, through Separator

<small>[comps/action_sheet.go:106](https://github.com/rohanthewiz/grmob/blob/master/comps/action_sheet.go#L106)</small>

#### func (ActionSheet) Render

```go
func (s ActionSheet) Render(ctx *core.Context) *core.Node
```

Render builds Modal > (filler, Card) as drawn in the type doc.

<small>[comps/action_sheet.go:165](https://github.com/rohanthewiz/grmob/blob/master/comps/action_sheet.go#L165)</small>

### type Banner

```go
type Banner struct {
	// Text is the message. Content is the escape hatch for the growing middle
	// — an arbitrary view in its place, taking precedence when set.
	Text    string
	Content core.View

	// Variant selects the semantic role: Success, Warning, Error, or the zero
	// value for the theme's Primary — the neutral "here is some information"
	// strip.
	Variant Variant

	// Glyph is the leading mark. Empty takes the variant's default (ⓘ ✓ ⚠ ⊗);
	// NoGlyph drops it entirely, for a strip that should read as quietly as
	// possible.
	Glyph   string
	NoGlyph bool

	// ActionLabel and OnAction render a single trailing action — "Retry",
	// "Reconnect", "Undo". Action is the slot form and takes precedence.
	//
	// The built button is a *default* ghost, not one tinted with the banner's
	// variant. The strip already says what kind of thing it is twice, in the
	// border and the glyph; a third telling would put the least legible
	// combination this package has (see Button's contrast table — outlined
	// and ghost own neither their fill nor their backdrop) on the one control
	// the user is meant to hit.
	ActionLabel string
	OnAction    func()
	Action      core.View

	// OnDismiss adds a trailing ✕ that closes the banner. It is the caller's
	// job to stop rendering the widget; nothing here holds state.
	OnDismiss func()

	// Style is applied after the widget's own treatment, so any of it can be
	// overridden. A caller who wants an edge-to-edge strip with no frame —
	// the shape a banner pinned directly under an AppBar usually wants —
	// spells that out:
	//
	//	Style: []core.StyleProp{core.BorderWidth(0), core.BorderRadius(0)}
	Style []core.StyleProp
}
```

Banner is the inline strip that tells the user something about the screen they are on: a failed refresh over content that is still good, a "Reconnecting…", an offline notice, a "Your changes were saved".

	comps.Banner{Text: "Could not refresh. Showing saved copy.",
	    Variant: comps.VariantWarning, ActionLabel: "Retry", OnAction: reload}

	┌ Row ─────────────────────────────────────────────────────────┐
	│ ⚠  ┌ Box FlexGrow(1) ───────────────┐  [Retry]  [✕]          │
	│    │ Text                           │                        │
	│    └────────────────────────────────┘                        │
	└──────────────────────────────────────────────────────────────┘

#### It is not a toast

core.ShowToast reaches the platform's own transient overlay and disappears on a timer. A Banner is part of the tree: it stays until the state that produced it changes, which is what a condition the user may need to act on requires. Use the toast for "Copied", the banner for "You are offline".

#### The variant is a tint, not a fill

Badge and a filled Button spend the whole variant color as a background. A strip that runs the width of the screen cannot: a saturated Error red across a screen reads as a failure of the app rather than of one fetch, and the palette carries no muted \*container\* tone to fill with instead. (It carries an on-light tone now, which is the opposite end of the range — ink for a light surface, not a wash to sit behind one — so it does not answer this. A container tone would still be a palette decision, not a Banner one.)

So the variant is spent on the edges: a hairline border and the leading glyph take the role's on-light tone, the fill stays the theme's Surface, and the text keeps the primary ink so it is legible whatever the role. That also means a Banner's contrast does not depend on which variant it is, which the alternatives could not promise.

#### Color is not the message, again

Nothing here announces "error" to a screen reader: the glyph is marked decorative (a reader saying "circled times" is worse than silence) and a border has no voice. Text must therefore carry the meaning on its own — "Could not refresh", not "Something went wrong" next to a red edge. Same rule Badge documents, and the same WCAG 1.4.1 behind it.

#### It announces itself when it appears

A banner is a live region: it turns up because something changed, usually while the reader is somewhere else on the screen, and a message nobody is looking at is a message nobody gets. So the strip carries core.RoleAlert when its variant is Error and core.RoleStatus otherwise — the same split the variant already draws visually, in the one vocabulary that has a word for "interrupt" and a word for "mention at the next pause".

Error interrupts because a failed action is the case where continuing is the wrong thing to do; everything else waits, because "Saved" arriving mid-sentence is how a live region becomes a thing users switch off.

The role is on the strip rather than on the message so that an appearing banner is announced whole — the text, and the label of any action beside it, which is the part the reader needs in order to know what to do about it.

It is a default, not a fixture. Style is applied after the widget's own props, so a caller can name a different role, or core.RoleNone for a strip that is really static content and should not interrupt anything.

<small>[comps/banner.go:70](https://github.com/rohanthewiz/grmob/blob/master/comps/banner.go#L70)</small>

#### func (Banner) Render

```go
func (b Banner) Render(ctx *core.Context) *core.Node
```

<small>[comps/banner.go:113](https://github.com/rohanthewiz/grmob/blob/master/comps/banner.go#L113)</small>

### type Dialog

```go
type Dialog struct {
	// Visible is the caller's open/closed state. The Modal renders its content
	// on every pass regardless and the host maps this to visibility.
	Visible bool

	// Title names the dialog. It is drawn as the card's heading and doubles as
	// the dialog's accessible name. Leave it empty only when Body names itself.
	Title string

	// Message is the one or two sentences under the title. Ignored when Body
	// is set.
	Message string

	// Body replaces Message with arbitrary content: a checkbox, a text field,
	// a list. It is the escape hatch in the same simple-path-plus-slot idiom
	// as ListRow.Content and Card.Header.
	Body core.View

	// Confirm is the affirmative action, drawn filled on the trailing side. A
	// zero Label omits it.
	Confirm DialogAction

	// Cancel is the dismissive action, drawn ghost on the leading side. A zero
	// Label omits it. A nil OnTap falls back to OnDismiss.
	Cancel DialogAction

	// OnDismiss is called for a backdrop tap and as Cancel's fallback. Nil
	// makes the scrim inert.
	OnDismiss func()

	// Backdrop overrides the scrim colour. Empty keeps core.Modal's default.
	Backdrop string

	// Style is applied to the card, after the widget's own props, so it can
	// override padding, width or background.
	Style []core.StyleProp
}
```

Dialog is the "Delete this note?" moment: a title, a sentence, and a row of at most two buttons, drawn over the screen in a core.Modal.

	comps.Dialog{
	    Visible:   confirming.Get(),
	    Title:     "Delete note?",
	    Message:   "This cannot be undone.",
	    Confirm:   comps.DialogAction{Label: "Delete", Variant: comps.VariantError, OnTap: del},
	    Cancel:    comps.DialogAction{Label: "Keep"},
	    OnDismiss: func() { confirming.Set(false) },
	}

#### Why the widget exists

core.Modal draws the scrim and the sheet and nothing inside them, so every caller hand-rolled the card, the title and the button row — and the hand-rolls disagreed on the two things a dialog must be consistent about: which side the cancel button sits on, and which button looks dangerous. Dialog settles both and adds nothing else.

#### One struct, three shapes

The shape follows from which actions carry a Label, not from a mode field:

	Confirm  Cancel   shape
	───────  ──────   ─────────────────────────────────────────────
	  set     set     confirm — the two-button "are you sure?"
	  set      —      alert   — one acknowledgement button
	   —       —      sheet   — title and body only; the Body slot
	                            carries its own controls, and a backdrop
	                            tap (OnDismiss) is the way out

Three widgets for these would share every line but the footer, and a caller moving from an alert to a confirm would have to change type rather than add a field. A Cancel with no Confirm is drawn as a lone button too; it is an alert whose one button happens to be the dismissive one.

#### Button order is fixed: cancel leading, confirm trailing

Material 3 and Apple's HIG both put the dismissive action on the leading side and the affirmative one on the trailing side of a horizontal pair, and the web has no convention strong enough to argue with them. There is no order knob: an order field is exactly the per-call-site disagreement this widget exists to remove. The pair is packed against the trailing edge (JustifyEnd), which is where both platforms put it and where a thumb is.

Cancel is always drawn EmphasisGhost so it reads as the way out. Confirm is filled and takes its own Variant, so a destructive action says VariantError and gets the theme's Error fill with an ink chosen for contrast by Button.

#### Controlled, like core.Modal

Dialog holds no state and never closes itself. Visible renders the caller's state; every way out reports intent and leaves the decision with the caller:

  - A backdrop tap calls OnDismiss, and so do the platform gestures the hosts route through the same prop: Compose's Dialog reports the back gesture and an outside tap through onDismissRequest, and SwiftUI (which presents a Modal as a sheet, not a centred card) reports a swipe-down. With OnDismiss nil the web scrim is inert, which is how a dialog that must be answered is written; the natives may still hide the sheet, but Visible stays true in Go and the next render shows it again.
  - Cancel.OnTap, when nil, falls back to OnDismiss. "Keep" and a tap on the scrim almost always mean the same thing, and writing the same closure twice is how the two drift apart.
  - Confirm.OnTap does not close the dialog. Confirming usually starts work whose outcome decides what shows next (close, or show an error), so the caller's handler sets Visible false when it is ready to.

Because core.Modal hides rather than unmounts, a Body slot's hook state survives a close; reset it in OnDismiss if the dialog should forget.

#### Accessibility

Both web targets write role="dialog" and aria-modal on the Modal chassis and both natives present a platform dialog, so the widget does not add a role (core.Role deliberately has no RoleDialog; see core/role.go). What the chassis cannot know is the dialog's name, so the card carries AccessibilityLabel(Title). The title text is also a section heading, via Card, so a reader can jump to it.

#### Theme roles read

	Card base       Components.Card, through core.Card
	Title           Typography.Subtitle, bold (Card's title treatment)
	Message         Typography.Body
	Confirm fill    Variant.Color — Colors.Primary, or Error/Success/Warning
	Cancel ink      Colors.Primary's on-light tone (ghost Button)
	Button gap      Spacing.SM

<small>[comps/dialog.go:94](https://github.com/rohanthewiz/grmob/blob/master/comps/dialog.go#L94)</small>

#### func (Dialog) Render

```go
func (d Dialog) Render(ctx *core.Context) *core.Node
```

Render builds Modal > Card(title, body, footer).

	┌ Modal (scrim; role=dialog on the web) ────────────────┐
	│  ┌ Card  AccessibilityLabel(Title) ────────────────┐  │
	│  │ Title                                (heading)  │  │
	│  │ Message  — or —  Body                           │  │
	│  │                        ┌ Row JustifyEnd ──────┐ │  │
	│  │                        │ [Cancel]  [Confirm]  │ │  │
	│  │                        └──────────────────────┘ │  │
	│  └─────────────────────────────────────────────────┘  │
	└───────────────────────────────────────────────────────┘

<small>[comps/dialog.go:161](https://github.com/rohanthewiz/grmob/blob/master/comps/dialog.go#L161)</small>

### type DialogAction

```go
type DialogAction struct {
	// Label is the button text. Empty means the action is absent.
	Label string

	// OnTap is called when the button is pressed.
	OnTap func()

	// Variant colours a Confirm button's fill (VariantError for a destructive
	// action). A Cancel button is always ghost and reads only the variant's
	// on-light ink, so Cancel's Variant is rarely worth setting.
	Variant Variant

	// Disabled greys the button and drops its taps, for a Confirm that waits
	// on a Body field ("type DELETE to confirm") or on work in flight.
	Disabled bool
}
```

DialogAction is one button in a Dialog's footer.

<small>[comps/dialog.go:133](https://github.com/rohanthewiz/grmob/blob/master/comps/dialog.go#L133)</small>

### type EmptyState

```go
type EmptyState struct {
	// Glyph is the large mark above the text. Empty draws none, which is the
	// right call for the busy case where a mark would look like a state the
	// user is meant to read.
	//
	// It is decoration and is hidden from assistive technology: a reader
	// announcing "open mailbox with lowered flag" ahead of "No messages yet"
	// is noise. Title has to carry the meaning.
	Glyph string

	// Title is the primary line — what is going on. Hint is the quieter line
	// under it — what to do about it.
	Title string
	Hint  string

	// ActionLabel and OnAction render the way out: "Retry", "Clear filters",
	// "Invite someone". Action is the slot form and takes precedence, for a
	// pair of buttons or anything else.
	//
	// The built button is outlined, not filled. An empty state is a dead end,
	// not a call to action — a solid Primary button in the middle of an empty
	// screen is the loudest thing on it, and the screen has nothing to say.
	ActionLabel string
	OnAction    func()
	Action      core.View

	// Style is applied after the widget's own defaults, so the padding, the
	// centering and the full width are all overridable.
	Style []core.StyleProp
}
```

EmptyState is the centered placeholder that stands in for content a screen does not have: a list with nothing in it, a fetch still in flight, a fetch that failed.

	comps.EmptyState{Glyph: "🔍", Title: "No sermons match “grace”",
	    Hint: "Try a different word, or clear the filters."}

#### One widget for empty, busy and failed

Those three look like three states but they are one shape — a mark, a line saying what is going on, a quieter line saying what to do about it, and sometimes a way out — and screens that build them separately end up wording and spacing them differently:

	empty   EmptyState{Glyph: "📭", Title: "No messages yet"}
	busy    EmptyState{Title: "Loading sermons…"}
	failed  EmptyState{Glyph: "☁", Title: err.Error(),
	            ActionLabel: "Retry", OnAction: reload}

The busy case is a line of text rather than a spinner on purpose: core has no indeterminate progress node, and ProgressBar is determinate (it takes a 0..1 value), so there is nothing here to animate a wait with. Naming what is loading is more useful than an animation in any case — "Loading sermons…" tells the user which of the screen's three sections is slow.

#### The busy line moves when the wait is not the whole screen

The three-state example above puts the busy words in Title, which is right for a wait that owns the screen and wrong for the other place a wait renders: under the last row of a paged list, as the footer that says the next page is coming. There, body-sized primary ink reads as one more row — the reader tries to parse "Loading sermons…" as content — so the tail case puts the words in Hint and leaves Title empty:

	screen  EmptyState{Title: "Loading sermons…"}
	tail    EmptyState{Hint: "Loading more…"}

The rule the split comes out of is \*errors and empties speak in the primary line; a wait speaks there only when it is the whole screen\*. The widget cannot apply it itself, because it is handed a slot and never learns whether that slot is a screen's middle or a list's end — so this is the caller's call, written down here so the next screen does not re-derive it and land somewhere else.

A tail usually wants less air than a screen-sized placeholder too. Padding is a default like any other and Style is applied after it:

	EmptyState{Hint: "Loading more…", Style: []core.StyleProp{core.Padding(t.Spacing.SM)}}

#### Width is load-bearing

The column sets Width 100%, which looks redundant and is not. On both natives a column hugs its widest child (Compose wrap-content; grmob's SwiftUI layout does the same), so without it the whole block sits at the leading edge with its children centered inside a box only as wide as the longest line — centered text that is not centered on the screen. The two DOM targets fill the line already, as any block box does, so the bug is invisible on the target you are most likely to be looking at.

<small>[comps/empty_state.go:63](https://github.com/rohanthewiz/grmob/blob/master/comps/empty_state.go#L63)</small>

#### func (EmptyState) Render

```go
func (e EmptyState) Render(ctx *core.Context) *core.Node
```

<small>[comps/empty_state.go:94](https://github.com/rohanthewiz/grmob/blob/master/comps/empty_state.go#L94)</small>

### type Menu

```go
type Menu struct {
	// Trigger is the button that opens the menu. Its OnTap is ignored and
	// replaced by OnOpen; see the type doc.
	Trigger Button

	// Open is the caller's open/closed state.
	Open bool

	// OnOpen is called when the trigger is tapped. Nil leaves a trigger that
	// does nothing, which is Trigger.Disabled's job done badly; set that
	// instead.
	OnOpen func()

	// OnDismiss is called for a scrim tap, for Cancel, and after any item,
	// exactly as ActionSheet.OnDismiss.
	OnDismiss func()

	// Title names the sheet: its heading and its accessible name. Empty
	// draws no heading, which leaves the sheet unnamed; set it.
	Title string

	// Items are the menu's entries, top to bottom.
	Items []SheetAction

	// Cancel labels the dismiss button under the items. Empty omits it.
	Cancel string

	// Backdrop overrides the scrim colour, as ActionSheet.Backdrop.
	Backdrop string

	// Style is applied to the sheet's card, as ActionSheet.Style. The
	// trigger is styled through Trigger.Style.
	Style []core.StyleProp
}
```

Menu is the overflow "⋯" and the "Sort by" picker: a button that opens a short list, where the tap that picks is the tap that closes it.

	comps.Menu{
	    Trigger:   comps.Button{Label: "⋯", AccessibilityLabel: "Note actions",
	                            Emphasis: comps.EmphasisGhost},
	    Open:      open.Get(),
	    OnOpen:    func() { open.Set(true) },
	    OnDismiss: func() { open.Set(false) },
	    Title:     "Groceries",
	    Items: []comps.SheetAction{
	        {Label: "Rename", OnTap: rename},
	        {Label: "Delete", Variant: comps.VariantError, OnTap: del},
	    },
	    Cancel: "Cancel",
	}

	┌ Row (no padding, no gap) ───────────────────────────┐
	│ [ ⋯ ]  Trigger, OnTap = OnOpen                      │
	│ ActionSheet{Visible: Open, Actions: Items, …}       │  ← a Modal: drawn
	└─────────────────────────────────────────────────────┘    over the screen,
	                                                          no room taken here

#### It is an ActionSheet with a trigger, and nothing more

A dropdown anchored under its button needs a popover positioned against the trigger's frame, and no host sends a frame to Go; that is a node type, and the plan that proposed this widget listed it as out of reach. Every target can already present a Modal, and ActionSheet has settled where a Modal's list goes on each of them, so the menu is that sheet. On a phone that is the platform's own shape for a menu of actions; in a browser window it is the same panel on the bottom edge, which is the one place it reads as a compromise.

So the items are SheetActions, picking one runs its OnTap and then OnDismiss, and Cancel, the scrim and a tap above the panel all dismiss. The ActionSheet type doc is where each of those is argued.

#### The trigger is a Button template, not a View slot

A core.View handed in by the caller cannot be given a tap handler: a widget cannot add props to a View it did not build. So the trigger is a comps.Button whose Label, Variant, Emphasis, Disabled, Style and names all apply and whose OnTap is overwritten with OnOpen, the same template move DatePicker makes with its Calendar. A trigger that must be something other than a button (a whole row) is an ActionSheet next to that row.

#### Open is the caller's, which is what makes a menu per row cost one state

DatePicker owns its open flag in a hook, and a Menu could too. It does not, because the commonest menu is the "⋯" on every row of a list, and a hook-holding widget in a loop takes a positional slot per row: the slots drift as soon as a filter or a delete changes the row count. Controlled, one state holds which row's menu is open, and every row compares against it:

	open := core.NewState(ctx, "")
	for _, n := range notes {
	    comps.Menu{
	        Open:      open.Get() == n.ID,
	        OnOpen:    func() { open.Set(n.ID) },
	        OnDismiss: func() { open.Set("") },
	        …
	    }
	}

It also leaves Menu conditional-safe, like every widget that takes no hook.

#### A picker is a menu whose items are checked

"Sort by: Newest" is the same widget: each item sets one value, and the current one is SheetAction.Checked. There is no Value/OnChange pair, since the items already carry their handlers and a second path to the same setter is a second thing to keep in step. RadioGroup is the form for a choice that should be compared on screen rather than behind a tap.

#### What the trigger announces

Its own label and hint, and that it opens a dialog: the trigger carries core.AccessibilityHasPopup(core.PopupDialog), which both web targets write as aria-haspopup="dialog". It states no expanded state, which is core.Style's disclosure near miss: a control that opens a dialog is not expanded. The value is dialog and not menu because the sheet is a dialog of buttons to a reader, not an ARIA menu with a menu's keyboard; see core.PopupKind. Once the sheet is open the Modal's dialog role is what a reader is inside. An icon trigger such as "⋯" needs Trigger.AccessibilityLabel, and a picker's trigger reads best with the current value in its label.

#### Theme roles read

	Trigger    as comps.Button, from the template's Variant and Emphasis
	Sheet      as ActionSheet

<small>[comps/menu.go:98](https://github.com/rohanthewiz/grmob/blob/master/comps/menu.go#L98)</small>

#### func (Menu) Render

```go
func (m Menu) Render(ctx *core.Context) *core.Node
```

Render builds Row(trigger, ActionSheet) as drawn in the type doc.

<small>[comps/menu.go:134](https://github.com/rohanthewiz/grmob/blob/master/comps/menu.go#L134)</small>

### type ProgressBar

```go
type ProgressBar struct {
	// Value is the completed fraction, 0 to 1. Values outside that range are
	// clamped rather than rejected: a bar fed a ratio from live counters
	// should pin at full and keep rendering, not draw outside its track.
	// NaN is treated as 0.
	Value float64

	// Thickness is the track height in px; 0 means 6.
	Thickness float64

	// Color is the fill; empty uses the theme's Primary. TrackColor is the
	// groove behind it; empty uses the theme's Surface.
	Color      string
	TrackColor string

	// Style is applied to the track after the defaults, so the bar's width,
	// margins and corner are all overridable.
	Style []core.StyleProp

	// AccessibilityLabel names what is progressing ("Upload"), and nothing
	// else — the value is announced separately, through the role and the
	// range below. When empty the bar is hidden from assistive tech: an
	// unlabeled bar announces a bare number with nothing to attach it to, and
	// a bar beside its own "Uploading, 45%" caption should stay silent.
	//
	// It used to carry the percentage as well ("Upload, 45 percent"), because
	// no renderer had a progress semantic to put the number in. Three of the
	// four do now, and a name was the wrong channel for it in the way
	// comps.Chip's old ", selected" suffix was: a name is meant to be
	// stable, so a bar ticking from 44 to 45 re-announced the whole string
	// rather than the part that changed, and nothing could act on a number
	// buried in it.
	AccessibilityLabel string

	// ValueText is the spoken form of the value, for a caller who wants
	// particular words ("almost done", "3 of 5 uploaded"). Empty is the
	// normal case and means "let each platform say the number in its own
	// words", which is the better answer on three of the four targets:
	//
	//	web ×2    aria-valuenow over an implicit 0..100, which a browser
	//	          announces as a localized percentage
	//	Compose   ProgressBarRangeInfo, which TalkBack localizes the same way
	//	SwiftUI   nothing. There is no numeric accessibility value on this
	//	          platform — accessibilityValue takes a string — so an
	//	          unaccompanied bar announces its name alone.
	//
	// That last row is why the field exists rather than being left out: iOS is
	// the one target where the value genuinely has nowhere to go, and an app
	// that would rather have English than silence there can say so. It costs
	// the localization on the other three, which is why it is not the default
	// — ARIA and Compose both announce this text *instead of* the number.
	//
	// The renderers deliberately do not supply it themselves. A framework
	// emitting "45 percent" would be inventing English for every app in every
	// locale, which is the same move GrMobStyle.swift turns down for
	// AccessibilityExpanded; these are the caller's own words.
	ValueText string
}
```

ProgressBar is the determinate track-and-fill bar: an upload, a download, a quota, a multi-step form's position.

	comps.ProgressBar{Value: 0.45, AccessibilityLabel: "Upload"}

#### Why the fill is a percentage width and not a pair of flex weights

The obvious construction is two boxes weighted FlexGrow(v) and FlexGrow(1-v), letting the flex algorithm split the track. That was exact on Android, where FlexGrow maps onto Compose's Modifier.weight — and silently wrong on iOS, where it mapped onto frame(maxWidth: .infinity). SwiftUI stacks have no weight, so two growers split the free space \*equally regardless of their values\*: every bar would have rendered at 50% on iOS, at every value, with nothing in the tree to suggest a bug.

A percentage width is proportional on all three targets instead:

	Android   fillMaxWidth(fraction)  — exact
	HTML      width:<pct>%            — exact
	iOS       containerRelativeFrame  — proportional, see the caveat below

The iOS caveat is that containerRelativeFrame measures against the nearest \*container\* (the scroll view or root), not the immediate parent. A bar that spans its container — the common case, a full-width bar in a screen column — is therefore exact; one inset inside a narrow card reads wider than it should. That is an over-long fill in an uncommon layout, against a permanently-half-full bar everywhere.

That custom SwiftUI Layout has since landed: GrMobFlexStack, with the arithmetic in GrMobFlexSolver, resolves FlexGrow by value on all three targets. This widget has not been migrated, so the caveat above is still what it does today — but the blocker is gone, and moving to flex would remove the caveat, since a flex child is measured against its immediate parent rather than the nearest container.

#### The fill is always rendered

Even at Value 0, where it is zero pixels wide. Keeping the child count fixed means advancing progress is a style patch on one node rather than an insert or a remove, so the reconciler emits an update-style op per frame instead of restructuring the tree — which is also what lets a Transition on the fill animate the bar smoothly.

<small>[comps/progress_bar.go:52](https://github.com/rohanthewiz/grmob/blob/master/comps/progress_bar.go#L52)</small>

#### func (ProgressBar) Render

```go
func (p ProgressBar) Render(ctx *core.Context) *core.Node
```

<small>[comps/progress_bar.go:111](https://github.com/rohanthewiz/grmob/blob/master/comps/progress_bar.go#L111)</small>

### type SheetAction

```go
type SheetAction struct {
	// Label is the button text.
	Label string

	// OnTap is called before the sheet's OnDismiss.
	OnTap func()

	// Variant tints the label. VariantError marks a destructive action.
	Variant Variant

	// Disabled greys the action and drops its taps. A disabled action does
	// not dismiss the sheet either, because it does not dispatch at all.
	Disabled bool

	// Checked marks the action as the current choice, for the "Sort by"
	// shape where each action sets one value (see Menu). The label gains a
	// leading ✓ and the accessible name a ", selected" suffix.
	//
	// The suffix rather than core.AccessibilitySelected, because the action
	// is a button and ARIA defines aria-pressed there, which would announce a
	// toggle ("Newest, pressed") that a second tap does not turn off. The
	// radio pair was the other candidate and is rejected for the reason the
	// type doc gives for the listbox one: inside a radiogroup the WASM
	// runtime checks the radio an arrow lands on, which here would run the
	// action and close the sheet on the first ArrowDown. It is the same
	// English fallback BottomBar uses for its current item.
	Checked bool
}
```

SheetAction is one row of an ActionSheet.

<small>[comps/action_sheet.go:135](https://github.com/rohanthewiz/grmob/blob/master/comps/action_sheet.go#L135)</small>

### type Skeleton

```go
type Skeleton struct {
	// Lines is how many bars to stack. Zero means one.
	Lines int

	// Height is a bar's height in points. Zero takes the theme's body font
	// size, so a text placeholder is about as tall as the text it stands in
	// for and scales with the theme.
	Height float64

	// Width is every bar's width, as a CSS-ish length ("100%", "180px").
	// Empty is "100%".
	Width string

	// LastLineWidth shortens the final bar, which is what makes a stack read
	// as a paragraph rather than as a table. Empty is "60%". It applies only
	// when Lines is 2 or more: on a single bar the last line is the only
	// line, and silently rendering it at 60% would make the simplest call
	// surprising.
	LastLineWidth string

	// Gap is the space between bars. Zero takes the theme's SM step.
	Gap float64

	// Radius is the bar's corner radius. Zero is 4 — enough to read as a
	// placeholder rather than as a rule. Set it to half the height (or just
	// to 999, which clamps) for a pill, or with a square Width/Height for the
	// circle an avatar placeholder wants.
	Radius float64

	// Color overrides the bar fill. Empty takes the theme's Border role; see
	// the type comment for why that and not Surface.
	Color string

	// AccessibilityLabel names the whole block for assistive technology.
	// Empty is "Loading".
	//
	// The individual bars are always hidden — a reader walking six unlabeled
	// boxes is worse than silence — and the label goes on the container
	// instead, which takes core.RoleStatus.
	//
	// # Why `status` and not the `group` it used to get
	//
	// Both web exporters supply RoleGroup to a named container that says
	// nothing about what it is, which made the name legal and stopped there:
	// `group` says "these things belong together and this is what they are
	// called", so a reader announced "Loading" only if the user happened to
	// walk onto the block. A skeleton is not a group of things — the bars
	// stand in for content that is not here yet — and what it is is ARIA's
	// definition of `status`: one advisory that is *replaced*. The wait
	// announces itself when it starts, and the content replaces it when it
	// arrives, which is exactly the region's contract.
	//
	// It is a live region, so it is announced without the reader looking at
	// it, which is what the old note here said a screen could not rely on and
	// had to put in text instead. That caveat is closed on the two web targets
	// and on Android (Compose's polite live region); SwiftUI has no live
	// region property, so on iOS this is still a labelled container, which
	// VoiceOver announces on arrival but does not interrupt for.
	//
	// For no announcement at all, pass core.AccessibilityHidden() in Style.
	AccessibilityLabel string

	// Style is applied to the container after the widget's own defaults.
	// Per-bar styling is not exposed: a skeleton whose bars differ is a
	// composition of Skeletons, not one Skeleton with more knobs.
	Style []core.StyleProp
}
```

Skeleton is the grey placeholder that holds a screen's shape while its content loads: one bar, or a stack of them standing in for a paragraph.

	comps.Skeleton{}                            // one line
	comps.Skeleton{Lines: 3}                    // a paragraph, last line short
	comps.Skeleton{Width: "44px", Height: 44, Radius: 999}   // an avatar

#### Skeleton or EmptyState

They answer different questions. A skeleton says "content is coming and it will look roughly like this", which is worth saying when the layout is known and stable — a feed of rows, a profile header. EmptyState says "there is nothing here yet, and here is why", which is what a screen with no predictable shape, or a wait long enough to need explaining, wants instead. A list of three placeholder rows reads better than "Loading…"; a whole screen of grey bars reads worse.

#### No shimmer, and why that is not a shortcut

The moving highlight every design system puts on a skeleton is a repeating keyframe animation. core.Transition is not one — it animates a property from one declared value to another, driven natively, and there is no state change here to drive. The alternative is looping it from Go with hooks.UseInterval, which would push a render pass and a patch across the bridge for every frame of a decoration, on every placeholder on screen. That is the one thing the framework's "declare in Go, animate natively" model exists to avoid, so the bars are static until a repeating animation is a core primitive.

#### The color is the Border role, not Surface

Surface is the palette's obvious "muted fill", and it is the wrong answer for the same reason Separator gives: it is the fill a \*panel\* uses, so a Surface bar inside a card disappears. Border is the neutral that is visible against both Background and Surface, which is where placeholders sit. It is nominally a stroke role and this is a fill — the palette carries no third neutral, and being visible beats being nominally correct.

<small>[comps/skeleton.go:46](https://github.com/rohanthewiz/grmob/blob/master/comps/skeleton.go#L46)</small>

#### func (Skeleton) Render

```go
func (s Skeleton) Render(ctx *core.Context) *core.Node
```

<small>[comps/skeleton.go:114](https://github.com/rohanthewiz/grmob/blob/master/comps/skeleton.go#L114)</small>

### type Snackbar

```go
type Snackbar struct {
	// Visible is the caller's shown/hidden state. Rising to true starts the
	// timeout; falling to false cancels it.
	Visible bool

	// Message is the one line of text. A new Message while visible restarts
	// the timeout.
	Message string

	// Action is the button label. Empty, or a nil OnAction, draws no button.
	Action string

	// OnAction is called when the action is tapped. It does not hide the
	// snackbar; set Visible false in it.
	OnAction func()

	// OnTimeout is called once, Duration after Visible turns true. Nil, like
	// a negative Duration, means the snackbar stays until the caller hides it.
	OnTimeout func()

	// Duration is how long the snackbar stays up. Zero uses
	// SnackbarDuration; a negative value never times out.
	Duration time.Duration

	// Variant tints the strip. VariantError also makes it an alert.
	Variant Variant

	// Style is applied to the strip after the widget's own props.
	Style []core.StyleProp
}
```

Snackbar is the "Note deleted · Undo" strip: one line of text and at most one action, shown for a few seconds and then gone.

	comps.Snackbar{
	    Visible:   undo.Get() != nil,
	    Message:   "Note deleted",
	    Action:    "Undo",
	    OnAction:  restore,
	    OnTimeout: func() { undo.Set(nil) },
	}

#### Why a widget and not core.ShowToast

core.ShowToast is fire-and-forget: Go hands the host a string and the host draws and removes it. What it cannot carry is a button, and an Undo is the whole reason a snackbar exists. Giving the toast an action callback would be a change on all four renderers. A widget the caller renders needs none, so this is the widget and the toast extension stays the eventual shape.

#### Controlled, and it holds one hook

The caller owns visibility, as with Dialog. Snackbar never hides itself; it reports two things and leaves the decision with the caller:

  - OnTimeout, Duration after Visible turns true. The timer is hooks.UseTimeoutWhile keyed on Message, so a replaced message gets its full Duration, and hiding the snackbar cancels a pending timeout.
  - OnAction, when the action is tapped. The caller usually undoes the work and hides the snackbar in the same handler.

Because of the hook, the rules Accordion documents apply: render a Snackbar in a stable position on every pass and drive Visible, rather than leaving it out of the tree. A hidden snackbar is Display none and its timer is cancelled, so it costs nothing.

#### Where it goes

It is a strip, not an overlay, so the caller places it. Two placements work on every target:

	Screen.Footer   core.Column(snackbar, bottomBar): pinned above the bar,
	                and a hidden snackbar takes no space
	core.ZStack     a layer with core.StackAlign(core.StackAlignBottom) over
	                content that fills the stack

The Footer form never covers content: the scroll region shrinks while the strip is up. That is usually what a list with an Undo wants, since the row that was just removed is the one a user looks for.

#### Accessibility

The strip is RoleStatus, a polite live region, so the message is read at the next pause and does not interrupt. VariantError makes it RoleAlert, which does interrupt; a failure the user must hear before continuing is what that role is for. No accessible name is set, because a live region announces its content and a label would replace the message.

#### Theme roles read

	Strip       Colors.TextPrimary as the fill, Colors.Background as the ink:
	            the inverse of the page, which is what makes it read as a
	            transient layer rather than as content
	Variant     Variant.Color as the fill and Variant.Ink for contrast, when
	            a Variant is set
	Message     Typography.Body
	Spacing     Spacing.SM gap and vertical padding, Spacing.MD horizontal

<small>[comps/snackbar.go:76](https://github.com/rohanthewiz/grmob/blob/master/comps/snackbar.go#L76)</small>

#### func (Snackbar) Render

```go
func (s Snackbar) Render(ctx *core.Context) *core.Node
```

Render builds Row(message, action) and holds the timeout hook.

<small>[comps/snackbar.go:113](https://github.com/rohanthewiz/grmob/blob/master/comps/snackbar.go#L113)</small>

### type Spinner

```go
type Spinner struct {
	// Hidden removes the spinner from display, which also stops its frames.
	// Prefer it to leaving the widget out where the spinner has a fixed place
	// in the layout; see the type doc.
	Hidden bool

	// Size picks the diameter. The zero value is SpinnerMedium.
	Size SpinnerSize

	// Label is the status announcement. Empty uses "Loading".
	Label string

	// Style is applied to the outer Box after the widget's own props.
	Style []core.StyleProp
}
```

Spinner is the "something is happening, shape unknown" indicator: a ring with a dot orbiting it.

	comps.Spinner{Hidden: !loading.Get()}
	comps.Spinner{Size: comps.SpinnerLarge, Label: "Uploading"}

It completes the loading trio. Skeleton is for content whose layout is known and whose data is not; ProgressBar is for work that knows how far it has got; Spinner is for everything else.

#### The platform turns it

The ring carries core.Spin(spinPeriodMs), so each renderer's own frame clock drives the rotation: CSS keyframes on the web, a graphics layer on Compose, a TimelineView on SwiftUI. Go declares the spin once, in the style of the first pass, and sends nothing afterwards.

It used to be stepped from Go instead — a state slot holding the angle and hooks.UseIntervalWhile adding 30 degrees every 80ms — because no renderer could draw a loop on its own. That cost a render pass of the whole app per step (twelve a second) while visible, drew twelve discrete positions rather than a turn, and gave the widget two hook slots with the ordering rule that comes with them. core.Spin removed all three.

#### No hooks, so it is conditional-safe

Spinner takes no hook slot, so core.If(loading, comps.Spinner{}) is as correct as Hidden. Hidden remains the switch to prefer where the spinner has a fixed place in a layout: a hidden spinner is Display none, which keeps the tree the same shape both ways, so showing it is a style patch rather than an inserted subtree. Either way a hidden or absent spinner draws no frames on any target (see core.Spin, "What it costs").

	┌ Box  role=status  label="Loading" ┐
	│  ┌ Column  ring, Spin(1000) ────┐ │
	│  │            ●                 │ │
	│  │                              │ │
	│  └──────────────────────────────┘ │
	└───────────────────────────────────┘

The dot is what makes the rotation visible. A ring of uniform colour turned about its centre draws the same pixels at every angle, and core has no per-side border colour to draw a gap in the ring with.

#### Accessibility

The outer Box is RoleStatus with Label (default "Loading"), a polite live region announced when it appears. The ring beneath it is decoration and is hidden from assistive technology. A hidden spinner is display:none and so is not announced at all.

No target slows or stops the spin for a reduce-motion setting, because core has no signal for it yet; see core.Spin, "Reduced motion".

#### Theme roles read

	Ring       Colors.BorderColor()
	Dot        Colors.Primary
	Diameter   Spacing.MD / LG / XL for Small / Medium / Large

<small>[comps/spinner.go:68](https://github.com/rohanthewiz/grmob/blob/master/comps/spinner.go#L68)</small>

#### func (Spinner) Render

```go
func (s Spinner) Render(ctx *core.Context) *core.Node
```

Render builds the status box and the spinning ring.

<small>[comps/spinner.go:117](https://github.com/rohanthewiz/grmob/blob/master/comps/spinner.go#L117)</small>

### type SpinnerSize

```go
type SpinnerSize string
```

SpinnerSize selects a Spinner's diameter from the theme's spacing scale.

<small>[comps/spinner.go:85](https://github.com/rohanthewiz/grmob/blob/master/comps/spinner.go#L85)</small>

```go
const (
	// SpinnerMedium is Spacing.LG across; the zero value.
	SpinnerMedium SpinnerSize = ""
	// SpinnerSmall is Spacing.MD across, for inline use beside text.
	SpinnerSmall SpinnerSize = "small"
	// SpinnerLarge is Spacing.XL across, for a spinner that is the screen's
	// whole content.
	SpinnerLarge SpinnerSize = "large"
)
```

