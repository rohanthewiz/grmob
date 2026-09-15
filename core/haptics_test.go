package core

import "testing"

// Every kind goes out as one "haptic" event carrying its own spelling, and
// the Haptic switch and HapticKinds agree: a kind added to one and not the
// other fails here.
func TestHapticSendsEveryKind(t *testing.T) {
	var kinds []string
	SetSystemEventHandler(func(name string, data map[string]any) {
		if name != systemEventHaptic {
			t.Errorf("unexpected event %q", name)
			return
		}
		k, _ := data["kind"].(string)
		kinds = append(kinds, k)
	})
	t.Cleanup(func() { SetSystemEventHandler(nil) })

	all := HapticKinds()
	for _, k := range all {
		Haptic(k)
	}
	if len(kinds) != len(all) {
		t.Fatalf("sent %d events for %d kinds", len(kinds), len(all))
	}
	for i, k := range all {
		if kinds[i] != string(k) {
			t.Errorf("event %d kind = %q, want %q", i, kinds[i], k)
		}
	}
}

// An unknown kind is dropped, and so is everything when no host listens.
func TestHapticDropsUnknownKinds(t *testing.T) {
	calls := 0
	SetSystemEventHandler(func(string, map[string]any) { calls++ })
	t.Cleanup(func() { SetSystemEventHandler(nil) })

	Haptic("sucess")
	Haptic("")
	if calls != 0 {
		t.Errorf("unknown kinds sent %d events", calls)
	}

	SetSystemEventHandler(nil)
	Haptic(HapticSuccess) // must not panic with no handler
}
