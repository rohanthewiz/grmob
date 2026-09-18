package comps

import (
	"strings"
	"testing"
	"time"

	"github.com/rohanthewiz/grmob/core"
)

// TimePicker takes no hooks, so these tests render it with the rowHarness
// only where they need a callback dispatched through a Context; the harness
// also fails any pass that raises a concern, which is the "empty concern
// list" half of the definition of done.

// pickerMorning is 9:30 AM on a day with a date and a location that are not
// the zero ones, so a test can see both carried through OnChange.
var pickerMorning = time.Date(2026, time.March, 14, 9, 30, 0, 0, time.FixedZone("UTC+2", 2*3600))

// selectsOf returns the picker's Selects in order: hour, minute, and the
// period when there is one.
func selectsOf(n *core.Node) []*core.Node {
	var out []*core.Node
	var walk func(*core.Node)
	walk = func(n *core.Node) {
		if n == nil {
			return
		}
		if n.Type == "Select" {
			out = append(out, n)
		}
		for _, c := range n.Children {
			walk(c)
		}
	}
	walk(n)
	return out
}

// optionLabels reads a Select's option labels off the wire shape core.Select
// writes, so a test asserts what every renderer is handed.
func optionLabels(t *testing.T, sel *core.Node) []string {
	t.Helper()
	opts, ok := sel.Props["options"].([]map[string]string)
	if !ok {
		t.Fatalf("options prop is %T", sel.Props["options"])
	}
	out := make([]string, len(opts))
	for i, o := range opts {
		out[i] = o["label"]
	}
	return out
}

// timeHarness holds the value OnChange writes back, and every value it was
// handed, so a test can assert exactly one change per pick.
type timeHarness struct {
	*rowHarness
	value   time.Time
	changes []time.Time
}

func newTimeHarness(t *testing.T, start time.Time, build func(h *timeHarness) TimePicker) *timeHarness {
	t.Helper()
	h := &timeHarness{value: start}
	h.rowHarness = newRowHarness(t, func() core.View {
		p := build(h)
		p.Value = h.value
		p.OnChange = func(v time.Time) {
			h.changes = append(h.changes, v)
			h.value = v
		}
		return p
	})
	return h
}

// pick dispatches Select i's change with an option value and re-renders.
func (h *timeHarness) pick(i int, value string) {
	h.t.Helper()
	sels := selectsOf(h.node)
	if i >= len(sels) {
		h.t.Fatalf("select %d of %d", i, len(sels))
	}
	id, ok := sels[i].Props["onChange"].(string)
	if !ok {
		h.t.Fatalf("select %d carries no onChange", i)
	}
	h.ctx.TriggerTextCallback(id, value)
	h.render()
}

func TestTimePickerIsANamedGroupOfThreePickers(t *testing.T) {
	h := newTimeHarness(t, pickerMorning, func(*timeHarness) TimePicker {
		return TimePicker{Label: "Start time"}
	})
	n := h.node
	if n.Style.AccessibilityRole != core.RoleGroup {
		t.Errorf("role = %q, want group", n.Style.AccessibilityRole)
	}
	if n.Style.AccessibilityLabel != "Start time" {
		t.Errorf("label = %q, want Label", n.Style.AccessibilityLabel)
	}
	if got := n.Style.AccessibilityValue.Text; got != "9:30 AM" {
		t.Errorf("group value = %q, want the whole time as DigitalClock writes it", got)
	}

	sels := selectsOf(n)
	if len(sels) != 3 {
		t.Fatalf("selects = %d, want hour, minute and period on a 12-hour picker", len(sels))
	}
	for i, want := range []string{"Hour", "Minute", "AM/PM"} {
		if got := sels[i].Style.AccessibilityLabel; got != want {
			t.Errorf("select %d is named %q, want %q", i, got, want)
		}
	}
	for i, want := range []string{"9", "30", timePeriodAM} {
		if got := sels[i].Props["value"]; got != want {
			t.Errorf("select %d value = %v, want %q", i, got, want)
		}
	}

	colon := findText(n, ":")
	if colon == nil || !colon.Style.AccessibilityHidden {
		t.Error("the colon should be drawn and hidden from assistive technology")
	}
}

// A 12-hour hour list reads as a clock does, with 12 first and valued 0, so
// the sum that makes a 24-hour hour needs no case for noon or midnight.
func TestTimePickerTwelveHourListStartsAtTwelve(t *testing.T) {
	n := TimePicker{Value: pickerMorning, OnChange: func(time.Time) {}}.Render(core.NewContext())
	hours := optionLabels(t, selectsOf(n)[0])
	want := []string{"12", "1", "2", "3", "4", "5", "6", "7", "8", "9", "10", "11"}
	if strings.Join(hours, ",") != strings.Join(want, ",") {
		t.Errorf("hours = %v, want %v", hours, want)
	}
	opts := selectsOf(n)[0].Props["options"].([]map[string]string)
	if opts[0]["value"] != "0" {
		t.Errorf("12's value = %q, want 0", opts[0]["value"])
	}
}

func TestTimePickerHour24HasNoPeriodAndPadsTheHour(t *testing.T) {
	n := TimePicker{Value: pickerMorning, Hour24: true, OnChange: func(time.Time) {}}.Render(core.NewContext())
	sels := selectsOf(n)
	if len(sels) != 2 {
		t.Fatalf("selects = %d, want hour and minute only", len(sels))
	}
	hours := optionLabels(t, sels[0])
	if len(hours) != 24 || hours[0] != "00" || hours[9] != "09" || hours[23] != "23" {
		t.Errorf("hours = %v, want 00…23", hours)
	}
	if sels[0].Props["value"] != "9" {
		t.Errorf("hour value = %v, want 9", sels[0].Props["value"])
	}
	if got := n.Style.AccessibilityValue.Text; got != "09:30" {
		t.Errorf("group value = %q, want 09:30", got)
	}
}

func TestTimePickerMinuteStepThinsTheList(t *testing.T) {
	n := TimePicker{Value: pickerMorning, MinuteStep: 15, OnChange: func(time.Time) {}}.Render(core.NewContext())
	got := optionLabels(t, selectsOf(n)[1])
	if strings.Join(got, ",") != "00,15,30,45" {
		t.Errorf("minutes = %v, want 00,15,30,45", got)
	}

	all := TimePicker{Value: pickerMorning, OnChange: func(time.Time) {}}.Render(core.NewContext())
	if n := len(optionLabels(t, selectsOf(all)[1])); n != 60 {
		t.Errorf("an unset step offers %d minutes, want every one of 60", n)
	}
}

// An off-step minute is shown as itself, in order, and nothing is written.
func TestTimePickerKeepsAMinuteOffTheStep(t *testing.T) {
	odd := time.Date(2026, time.March, 14, 9, 7, 0, 0, time.UTC)
	h := newTimeHarness(t, odd, func(*timeHarness) TimePicker {
		return TimePicker{MinuteStep: 15}
	})
	sel := selectsOf(h.node)[1]
	if got := strings.Join(optionLabels(t, sel), ","); got != "00,07,15,30,45" {
		t.Errorf("minutes = %v, want the odd one inserted in order", got)
	}
	if sel.Props["value"] != "7" {
		t.Errorf("minute value = %v, want 7: the field shows the time the caller holds", sel.Props["value"])
	}
	if len(h.changes) != 0 {
		t.Errorf("rendering an off-step time wrote %v; rounding is a write nobody made", h.changes)
	}

	h.pick(1, "30")
	if got := strings.Join(optionLabels(t, selectsOf(h.node)[1]), ","); got != "00,15,30,45" {
		t.Errorf("after a pick the minutes are %v, want the odd one gone", got)
	}
}

// Every pick is one complete time, reported at once, with the date and the
// location carried through and the seconds dropped.
func TestTimePickerAHourPickIsOneChangeOnTheSameDay(t *testing.T) {
	withSeconds := pickerMorning.Add(45*time.Second + 5*time.Millisecond)
	h := newTimeHarness(t, withSeconds, func(*timeHarness) TimePicker { return TimePicker{} })

	h.pick(0, "10")
	if len(h.changes) != 1 {
		t.Fatalf("changes = %d, want exactly one", len(h.changes))
	}
	got := h.changes[0]
	want := time.Date(2026, time.March, 14, 10, 30, 0, 0, pickerMorning.Location())
	if !got.Equal(want) || got.Location() != pickerMorning.Location() {
		t.Errorf("reported %v, want %v in the Value's own location", got, want)
	}
	if got.Second() != 0 || got.Nanosecond() != 0 {
		t.Errorf("reported %v, want the seconds the field does not show dropped", got)
	}
}

// The hour picked on a 12-hour clock lands in the period the field shows, and
// flipping the period keeps the hour within it — including at 12.
func TestTimePickerTwelveHourArithmetic(t *testing.T) {
	evening := time.Date(2026, time.March, 14, 21, 30, 0, 0, time.UTC)
	h := newTimeHarness(t, evening, func(*timeHarness) TimePicker { return TimePicker{} })

	if v := selectsOf(h.node)[0].Props["value"]; v != "9" {
		t.Fatalf("9 PM's hour value = %v, want 9", v)
	}
	h.pick(0, "0") // "12" in the PM period is noon
	if got := h.value.Hour(); got != 12 {
		t.Errorf("12 in the afternoon = hour %d, want 12", got)
	}
	h.pick(2, timePeriodAM) // noon to midnight
	if got := h.value.Hour(); got != 0 {
		t.Errorf("12 PM flipped to AM = hour %d, want 0", got)
	}
	h.pick(0, "7")
	h.pick(2, timePeriodPM)
	if got := h.value.Hour(); got != 19 {
		t.Errorf("7 AM flipped to PM = hour %d, want 19", got)
	}
	if len(h.changes) != 4 {
		t.Errorf("changes = %d, want one per pick", len(h.changes))
	}
}

// A re-pick of the option already shown is not a change.
func TestTimePickerSamePickReportsNothing(t *testing.T) {
	h := newTimeHarness(t, pickerMorning, func(*timeHarness) TimePicker { return TimePicker{} })
	h.pick(1, "30")
	h.pick(2, timePeriodAM)
	if len(h.changes) != 0 {
		t.Errorf("re-picking the shown options reported %v", h.changes)
	}
}

func TestTimePickerDisabledDisablesEveryPicker(t *testing.T) {
	h := newTimeHarness(t, pickerMorning, func(*timeHarness) TimePicker {
		return TimePicker{Disabled: true}
	})
	for i, s := range selectsOf(h.node) {
		if !s.Style.Disabled {
			t.Errorf("select %d is not disabled", i)
		}
	}
	// A pick racing the disabling patch lands on a handler that is still
	// registered, and does nothing.
	h.pick(0, "10")
	if len(h.changes) != 0 {
		t.Errorf("a disabled picker reported %v", h.changes)
	}
}

func TestTimePickerLocalisesThePeriod(t *testing.T) {
	n := TimePicker{
		Value: pickerMorning, OnChange: func(time.Time) {},
		AMLabel: "a.m.", PMLabel: "p.m.",
	}.Render(core.NewContext())
	if got := strings.Join(optionLabels(t, selectsOf(n)[2]), ","); got != "a.m.,p.m." {
		t.Errorf("periods = %v", got)
	}
	if got := n.Style.AccessibilityValue.Text; got != "9:30 a.m." {
		t.Errorf("group value = %q, want the localised period spoken too", got)
	}
}

// A zero Value is midnight, not a blank: there is no half-made time.
func TestTimePickerZeroValueIsMidnight(t *testing.T) {
	n := TimePicker{OnChange: func(time.Time) {}}.Render(core.NewContext())
	sels := selectsOf(n)
	for i, want := range []string{"0", "0", timePeriodAM} {
		if got := sels[i].Props["value"]; got != want {
			t.Errorf("select %d value = %v, want %q", i, got, want)
		}
	}
}

func TestTimePickerInertIsAConcern(t *testing.T) {
	core.SetDebugMode(true)
	core.ClearConcerns()
	t.Cleanup(func() { core.SetDebugMode(false); core.ClearConcerns() })

	TimePicker{Value: pickerMorning}.Render(core.NewContext())
	if !strings.Contains(core.DumpConcerns(), ConcernTimePickerInert) {
		t.Errorf("want %s, got:\n%s", ConcernTimePickerInert, core.DumpConcerns())
	}
}
