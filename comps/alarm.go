package comps

import (
	"github.com/rohanthewiz/grmob/alarm"
	"github.com/rohanthewiz/grmob/core"
)

// AlarmRow is one alarm in a list: its time as the title, its label and repeat
// days under it, and a switch that turns it on and off.
//
//	core.For(alarms.Get(), func(a alarm.Alarm, i int) core.View {
//	    return core.Keyed(a.ID, comps.AlarmRow{Alarm: a, OnToggle: func(on bool) { setEnabled(a.ID, on) }})
//	})
//
// It is a SwitchRow, so everything in that type's doc applies — the whole row
// toggles, and OnToggle is a setter that receives the new value. The subtitle
// is "Wake up · Weekdays", or just the days for an unlabelled alarm, which is
// what a reader hears as the switch's hint after hearing the time as its name.
type AlarmRow struct {
	Alarm alarm.Alarm

	// Hour24 writes the time as 06:30 rather than 6:30 AM.
	Hour24 bool

	// OnToggle receives the new Enabled value.
	OnToggle func(on bool)

	// Style is passed through to the row.
	Style []core.StyleProp
}

func (r AlarmRow) Render(ctx *core.Context) *core.Node {
	subtitle := r.Alarm.DaysLabel()
	if r.Alarm.Label != "" {
		subtitle = r.Alarm.Label + " · " + subtitle
	}
	return SwitchRow{
		Title:    r.Alarm.TimeLabel(r.Hour24),
		Subtitle: subtitle,
		On:       r.Alarm.Enabled,
		OnToggle: r.OnToggle,
		Style:    r.Style,
	}.Render(ctx)
}

// AlarmRinging is the screen an alarm puts up while it rings: the time it was
// set for, its label, and two buttons.
//
//	ringer := hooks.UseAlarms(ctx, alarms.Get(), opts)
//	if a, ok := ringer.Ringing(); ok {
//	    return comps.AlarmRinging{Alarm: a, OnSnooze: ringer.Snooze, OnDismiss: ringer.Dismiss}
//	}
//
//	          ┌───────────────────────┐
//	          │        6:30 AM        │   the alarm's time, display size
//	          │        Wake up        │   label, if any
//	          │                       │
//	          │ [       Snooze      ] │   filled: the easy target half-awake
//	          │ [      Dismiss      ] │   outlined
//	          └───────────────────────┘
//
// # Why Snooze is the big button
//
// Every alarm clock makes snooze the easy target and dismiss the deliberate
// one, because the costly mistake is dismissing by accident — a snooze pressed
// by mistake costs nine minutes, a dismiss pressed by mistake costs the
// morning. Filled against outlined is that distinction in this library's
// vocabulary. OnSnooze nil drops the button, for an alarm with no snooze.
//
// # Accessibility
//
// The panel is an alert (core.RoleAlert), so a screen reader announces it when
// it appears rather than waiting for the user to find it, and its label is a
// sentence: "Alarm, 6:30 AM, Wake up". The time and label inside are hidden to
// avoid reading that twice; the buttons are ordinary buttons.
type AlarmRinging struct {
	Alarm alarm.Alarm

	// Hour24 writes the time as 06:30.
	Hour24 bool

	OnSnooze  func()
	OnDismiss func()

	// Style is applied last, to the panel.
	Style []core.StyleProp
}

func (r AlarmRinging) Render(ctx *core.Context) *core.Node {
	t := ctx.Theme()
	when := r.Alarm.TimeLabel(r.Hour24)

	label := "Alarm, " + when
	if r.Alarm.Label != "" {
		label += ", " + r.Alarm.Label
	}

	items := []core.PropsAndChildren{
		core.Padding(t.Spacing.LG),
		core.Gap(float64(t.Spacing.MD)),
		core.AlignItemsProp(core.AlignItemsCenter),
		core.Width("100%"),
		core.AccessibilityRole(core.RoleAlert),
		core.AccessibilityLabel(label),
	}
	for _, sp := range r.Style {
		items = append(items, sp)
	}
	items = append(items,
		core.Text(when,
			core.FontSize(56),
			core.FontWeight(core.Light),
			core.TextColor(t.Colors.TextPrimary),
			core.AccessibilityHidden(),
		),
	)
	if r.Alarm.Label != "" {
		items = append(items, core.Text(r.Alarm.Label,
			core.UseStyle(t.Typography.Subtitle),
			core.TextColor(t.Colors.TextSecondary),
			core.AccessibilityHidden(),
		))
	}
	// A spacer's worth of gap before the buttons, so the two groups read as
	// "what" and "what to do about it".
	items = append(items, core.Box(core.Padding(0), core.Height(px(float64(t.Spacing.LG)))))
	if r.OnSnooze != nil {
		items = append(items, Button{Label: "Snooze", OnTap: r.OnSnooze, FullWidth: true})
	}
	if r.OnDismiss != nil {
		items = append(items, Button{Label: "Dismiss", OnTap: r.OnDismiss, FullWidth: true, Emphasis: EmphasisOutlined})
	}
	return core.Column(items...).Render(ctx)
}
