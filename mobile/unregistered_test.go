package mobile

import (
	"bytes"
	"encoding/json"
	"log"
	"strings"
	"sync"
	"testing"
)

// A shell calling in before Register (plan R9b) used to panic on the nil
// manager inside gomobile's frame, which ends the process. Every entry point
// now answers instead: RenderInitial with a tree that says what is wrong, the
// rest with "[]". Internal so it can clear mgr; it restores it, since the
// other tests in this package register an app.
func TestEveryEntryPointAnswersBeforeRegister(t *testing.T) {
	saved := mgr
	mgr = nil
	warnUnregistered = sync.Once{}
	var logged bytes.Buffer
	prevOut := log.Writer()
	log.SetOutput(&logged)
	t.Cleanup(func() {
		mgr = saved
		warnUnregistered = sync.Once{}
		log.SetOutput(prevOut)
	})

	var tree struct {
		Type  string
		Props map[string]any
	}
	if err := json.Unmarshal([]byte(RenderInitial()), &tree); err != nil {
		t.Fatalf("RenderInitial before Register is not a tree: %v", err)
	}
	if tree.Type != "Text" || tree.Props["content"] != notRegistered {
		t.Fatalf("RenderInitial before Register = %+v, want a Text saying %q", tree, notRegistered)
	}

	SetListener(nil) // nothing to attach to; must not panic
	for name, call := range map[string]func() string{
		"RenderAgain":         RenderAgain,
		"TriggerCallback":     func() string { return TriggerCallback("cb_0") },
		"TriggerTextCallback": func() string { return TriggerTextCallback("txt_cb_0", "x") },
		"TriggerTextEdit":     func() string { return TriggerTextEdit("txt_cb_0", "x", 1, 0) },
		"TriggerBoolCallback": func() string { return TriggerBoolCallback("bool_cb_0", true) },
		"TriggerIntCallback":  func() string { return TriggerIntCallback("int_cb_0", 1) },
		"Resync":              Resync,
		"ReportHostEvent":     func() string { return ReportHostEvent("window", `{"width":400}`) },
	} {
		if got := call(); got != "[]" {
			t.Errorf("%s before Register = %q, want []", name, got)
		}
	}

	// Said once, not once per call.
	if n := strings.Count(logged.String(), "mobile.Register was never called"); n != 1 {
		t.Errorf("the missing Register was logged %d times, want once:\n%s", n, logged.String())
	}
}
