package comps

import (
	"testing"
	"time"

	"github.com/rohanthewiz/grmob/alarm"
	"github.com/rohanthewiz/grmob/core"
)

func TestAlarmRowTitleAndSubtitle(t *testing.T) {
	n := renderView(t, AlarmRow{Alarm: alarm.Alarm{Hour: 6, Minute: 30, Label: "Wake up", Days: alarm.Weekdays, Enabled: true}})
	if findText(n, "6:30 AM") == nil || findText(n, "Wake up · Weekdays") == nil {
		t.Errorf("row text missing:\n%s", dumpText(n))
	}
	sw := findFirst(n, func(n *core.Node) bool { return n.Type == "Switch" })
	if sw == nil || sw.Props["checked"] != true {
		t.Errorf("switch missing or off: %v", sw)
	}
}

func TestAlarmRingingAnnouncesAndOffersBothActions(t *testing.T) {
	n := renderView(t, AlarmRinging{
		Alarm:     alarm.Alarm{Hour: 18, Minute: 5, Label: "Pick up", Days: []time.Weekday{time.Monday}},
		OnSnooze:  func() {},
		OnDismiss: func() {},
	})
	if n.Style.AccessibilityRole != core.RoleAlert || n.Style.AccessibilityLabel != "Alarm, 6:05 PM, Pick up" {
		t.Errorf("role %q label %q", n.Style.AccessibilityRole, n.Style.AccessibilityLabel)
	}
	for _, label := range []string{"Snooze", "Dismiss"} {
		if findFirst(n, func(n *core.Node) bool { return n.Props["label"] == label }) == nil {
			t.Errorf("no %s button", label)
		}
	}
	if findFirst(renderView(t, AlarmRinging{OnDismiss: func() {}}), func(n *core.Node) bool { return n.Props["label"] == "Snooze" }) != nil {
		t.Error("Snooze drawn with no OnSnooze")
	}
}
