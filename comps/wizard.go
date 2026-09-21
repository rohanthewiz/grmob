package comps

import (
	"fmt"

	"github.com/rohanthewiz/grmob/core"
)

// ConcernWizardInert is raised, in debug builds only, when a Wizard with more
// than one step has no OnChange. Next and Back would report to nobody, so the
// flow could never leave its first step.
const ConcernWizardInert = "wizard-inert"

// ConcernWizardNoSteps is raised, in debug builds only, when a Wizard has no
// Steps. It renders an empty column, which is never what a caller meant.
const ConcernWizardNoSteps = "wizard-no-steps"

// WizardStep is one step of a Wizard.
type WizardStep struct {
	// Title names the step in the indicator and is drawn as the heading over
	// its Body.
	Title string

	// Body is the step's content. It is rendered only while the step is the
	// current one; see "Bodies must not own hooks" on Wizard.
	Body core.View

	// Blocked disables Next (or Finish) while the step's input is not yet
	// acceptable. The caller computes it on every pass, usually from a form:
	// Blocked: !form.Valid(). The zero value lets the step advance, so a step
	// that is only read needs no field set.
	Blocked bool

	// Optional lets a Blocked step be passed all the same. Next stays enabled
	// and reads SkipLabel while the step is Blocked, so the button says what
	// pressing it will do: the step's input is not taken.
	Optional bool
}

// Wizard is a multi-step flow on one screen: a StepIndicator, the current
// step's title and body, and a Back / Next footer.
//
//	comps.Wizard{
//	    Label:   "Checkout",
//	    Steps:   []comps.WizardStep{
//	        {Title: "Address", Body: addressForm, Blocked: !addr.Valid()},
//	        {Title: "Gift note", Body: noteField, Optional: true, Blocked: note == ""},
//	        {Title: "Review", Body: summary},
//	    },
//	    Current:  step.Get(),
//	    OnChange: step.Set,
//	    OnFinish: placeOrder,
//	}
//
//	┌ Column  role=group  "Checkout" ───────────────────────────────┐
//	│ StepIndicator   (✓) Address ── (2) Gift note ── (3) Review    │
//	│ Text  role=status   "Step 2 of 3, optional"                   │
//	│ Text  role=heading  "Gift note"                               │
//	│ <Steps[Current].Body>                                         │
//	│ Footer():  [ Back ]                              [ Skip ]     │
//	└───────────────────────────────────────────────────────────────┘
//
// # The caller holds Current
//
// As with Tabs and StepIndicator. The step a flow is on is the state an app
// most wants to own: to resume a half-finished checkout, to jump to the step
// a server-side error belongs to, to leave the flow from outside. OnChange
// receives the index to move to and the caller Sets it.
//
// # Which moves are offered
//
//	Back             to Current-1. Absent on the first step and not merely
//	                 disabled: there is nothing it could ever do there, and a
//	                 dead button is a question ("why can't I?") with no answer.
//	Next             to Current+1. Disabled while the step is Blocked, unless
//	                 the step is Optional, when it reads SkipLabel instead.
//	Finish           the last step's Next. It calls OnFinish and not OnChange.
//	A done step      in the indicator, tappable (StepIndicator.OnTap). Later
//	                 steps are not: jumping ahead would pass a Blocked step
//	                 without the wizard having been asked.
//
// # Bodies must not own hooks
//
// ONLY THE CURRENT STEP'S BODY IS RENDERED. A Body that calls a hook
// (core.NewState, forms.UseForm, a comps widget that holds one, such as
// Accordion or DatePicker) is therefore rendered on some passes and not on
// others, which is a conditional hook: every hook after it shifts onto a
// neighbour's slot on the pass the step changes. Debug mode reports it as
// cursor drift.
//
// Hold every step's state above the Wizard, and hand each Body the values:
//
//	addr := forms.UseForm(ctx, addrSpec)      // all steps' hooks, every pass
//	note := core.NewState(ctx, "")
//	comps.Wizard{Steps: []comps.WizardStep{
//	    {Title: "Address", Body: addressFields(addr)},   // views only
//	    {Title: "Gift note", Body: noteField(note)},
//	}}
//
// This is also what a wizard wants: a step's input must survive going Back
// and forward again, and state owned by a Body would be discarded with it.
// A Body that cannot avoid hooks can take a scope of its own, ctx.Scope(title),
// whose cursor does not disturb its siblings.
//
// The Wizard itself holds no hook.
//
// # Where the footer goes
//
// By default at the end of the wizard's own column. A form long enough to
// scroll wants the buttons pinned instead, and the pin is comps.Screen's
// Footer, so the footer is an exported view and DetachFooter stops the wizard
// drawing it twice:
//
//	w := comps.Wizard{…, DetachFooter: true}
//	comps.Screen{Scroll: true, KeyboardAware: true,
//	    Children: []core.View{w}, Footer: w.Footer()}
//
// Inline is the default because it depends on nothing but a Column.
//
// # Accessibility
//
// The column is a RoleGroup named by Label. The indicator states the position
// for a reader who goes looking ("Step 2 of 3: Gift note"). The line under it
// says the same in fewer words and is a RoleStatus, so that a step change is
// announced where live regions are: a Next that replaced the screen's
// content and said nothing would leave a reader on a button that is now
// about something else. The title is a heading, so the new content is one
// heading-jump away.
//
// What this does not do is move focus to the heading, which is what a
// page-style navigation does. core.Focus reaches fields and Buttons; no
// target focuses a Text. VoiceOver announces no live region (core/role.go),
// so on iOS a step change is silent until the reader moves.
//
// # Theme roles read
//
//	Position line   Typography.Caption, Colors.TextSecondary
//	Title           Typography.Title
//	Buttons         comps.Button: Back outlined, Next filled
//	Gaps            Spacing.MD between parts, Spacing.SM in the footer
type Wizard struct {
	// Steps are the flow's steps, in order.
	Steps []WizardStep

	// Current is the zero-based index of the step shown. It is clamped into
	// the Steps range.
	Current int

	// OnChange receives the index to move to: Current-1 from Back, Current+1
	// from Next, a done step's index from the indicator. Nil with more than
	// one step reports ConcernWizardInert.
	OnChange func(step int)

	// OnFinish is called by the last step's button. Nil leaves that button
	// disabled: a Finish that does nothing reads as a hung app.
	OnFinish func()

	// NextLabel, BackLabel, FinishLabel and SkipLabel are the buttons' words.
	// Empty gives "Next", "Back", "Finish" and "Skip".
	NextLabel, BackLabel, FinishLabel, SkipLabel string

	// PositionLabel builds the status line from the zero-based current index,
	// the number of steps and whether the step is Optional. Nil gives "Step 2
	// of 3", with ", optional" after an Optional step's. The seam
	// StepIndicator.PositionLabel is: the widget knows the facts, the app
	// knows the language.
	PositionLabel func(current, total int, optional bool) string

	// Label names the group, and prefixes the indicator's name ("Checkout,
	// step 2 of 3: Gift note").
	Label string

	// DetachFooter leaves the footer out of the wizard's column, for a caller
	// that places Footer() elsewhere. See "Where the footer goes".
	DetachFooter bool

	// Style is applied to the outer column after the widget's own props.
	Style []core.StyleProp
}

// current is Current clamped into range; zero when there are no steps. Render
// and Footer both go through it so the body drawn and the buttons offered
// agree on one index even when the caller's is out of range.
func (w Wizard) current() int {
	if len(w.Steps) == 0 {
		return 0
	}
	return min(max(w.Current, 0), len(w.Steps)-1)
}

// or returns s, or fallback when s is empty.
func or(s, fallback string) string {
	if s == "" {
		return fallback
	}
	return s
}

// Render builds the column. It takes no hook slot.
func (w Wizard) Render(ctx *core.Context) *core.Node {
	t := ctx.Theme()
	n := len(w.Steps)

	if core.IsDebugMode() {
		switch {
		case n == 0:
			core.ReportConcern(ConcernWizardNoSteps, "Wizard has no Steps, so it draws nothing")
		case n > 1 && w.OnChange == nil:
			core.ReportConcern(ConcernWizardInert,
				"Wizard has more than one step and no OnChange, so Next and Back move nothing; set OnChange and store the index it reports in Current")
		}
	}

	items := make([]core.PropsAndChildren, 0, len(w.Style)+9)
	items = append(items,
		// The theme's screen inset belongs to the screen the wizard is on.
		core.Padding(0),
		core.Gap(float64(t.Spacing.MD)),
		core.AccessibilityRole(core.RoleGroup),
	)
	if w.Label != "" {
		items = append(items, core.AccessibilityLabel(w.Label))
	}
	items = append(items, asProps(w.Style)...)
	if n == 0 {
		return core.Column(items...).Render(ctx)
	}

	cur := w.current()
	step := w.Steps[cur]

	titles := make([]string, n)
	for i, s := range w.Steps {
		titles[i] = s.Title
	}
	indicator := StepIndicator{Steps: titles, Current: cur, Label: w.Label}
	if w.OnChange != nil {
		// Passed through as it is: StepIndicator already offers done steps
		// only, which is the wizard's rule too.
		indicator.OnTap = w.OnChange
	}

	var position string
	if w.PositionLabel != nil {
		position = w.PositionLabel(cur, n, step.Optional)
	} else {
		position = fmt.Sprintf("Step %d of %d", cur+1, n)
		if step.Optional {
			position += ", optional"
		}
	}

	items = append(items,
		indicator,
		// Position and title share a column so the MD gap does not part two
		// lines that are read as one.
		core.Column(
			core.Padding(0),
			core.Gap(float64(t.Spacing.XS)),
			core.Text(position,
				core.UseStyle(t.Typography.Caption),
				core.TextColor(t.Colors.TextSecondary),
				core.AccessibilityRole(core.RoleStatus),
			),
			core.Text(step.Title, append(
				[]core.StyleProp{core.UseStyle(t.Typography.Title)},
				headingProps(0, headingLevelSection)...,
			)...),
		),
	)
	if step.Body != nil {
		// Keyed by index: two steps' bodies are different content in the same
		// child slot, and without a key the diff would morph one form into
		// the next, carrying the focus and the text of a field at the same
		// position across. A key makes it a replacement.
		items = append(items, core.Keyed(fmt.Sprintf("wizard-step-%d", cur), step.Body))
	}
	if !w.DetachFooter {
		items = append(items, w.Footer())
	}
	return core.Column(items...).Render(ctx)
}

// Footer is the Back / Next row, for a caller that places it outside the
// wizard (with DetachFooter set). It reads the same fields Render does, so it
// must be taken from the same Wizard value on the same pass.
//
//	first step     [            ]            [ Next ]
//	middle step    [ Back ]                  [ Next ]   or [ Skip ]
//	last step      [ Back ]                  [ Finish ]
func (w Wizard) Footer() core.View {
	return core.ComponentFunc(func(ctx *core.Context) *core.Node {
		t := ctx.Theme()
		n := len(w.Steps)
		row := []core.PropsAndChildren{
			core.Padding(0),
			core.Gap(float64(t.Spacing.SM)),
			core.AlignItemsProp(core.AlignItemsCenter),
		}
		if n == 0 {
			return core.Row(row...).Render(ctx)
		}
		cur := w.current()
		step := w.Steps[cur]
		last := cur == n-1

		// move guards OnChange once for both directions. A nil OnChange is a
		// reported concern and must not be a panic in a release build.
		move := func(to int) func() {
			return func() {
				if w.OnChange != nil {
					w.OnChange(to)
				}
			}
		}

		if cur > 0 {
			row = append(row, Button{
				Label:    or(w.BackLabel, "Back"),
				Emphasis: EmphasisOutlined,
				OnTap:    move(cur - 1),
			})
		}
		// A growing filler, and not core.Spacer, which is a fixed size. It is
		// there on the first step too, so Next keeps its place at the
		// trailing edge and does not jump when Back appears.
		row = append(row, core.Box(core.FlexGrow(1), core.AccessibilityHidden()))

		forward := Button{Label: or(w.NextLabel, "Next"), OnTap: move(cur + 1)}
		// An Optional step is never a wall, only a different word.
		forward.Disabled = step.Blocked && !step.Optional
		if step.Blocked && step.Optional {
			forward.Label = or(w.SkipLabel, "Skip")
		}
		if last {
			// The last step's forward move is OnFinish. "Skip" is not offered
			// here: there is no later step to skip to, and "Skip" on a button
			// that submits would mislabel what it does.
			forward.Label = or(w.FinishLabel, "Finish")
			forward.OnTap = w.OnFinish
			forward.Disabled = forward.Disabled || w.OnFinish == nil
		}
		row = append(row, forward)
		return core.Row(row...).Render(ctx)
	})
}
