package core

// Haptics: a short, named buzz from the device's vibration motor.
//
// A system event like OpenURL — one way, fire-and-forget, no Context, callable
// from any goroutine — because nothing about it is part of the view tree and
// there is nothing to hear back: a device with no motor, a user who turned
// system haptics off, and a browser with no Vibration API all degrade to the
// same silence, and none of them is something a screen should draw.
//
// # Why named kinds rather than a duration
//
// iOS does not let an app drive the motor by time at all; it offers three
// generator families (selection, impact at a weight, notification with an
// outcome), and each is tuned by Apple to feel like the rest of the system.
// Android can vibrate by milliseconds but has its own predefined effects that
// match its system feel. A kind is the common denominator that lets both
// natives use their native effect, and the browser, which only has
// milliseconds, gets a small pattern table instead.
//
//	kind       iOS                                  Android (API 29+)       browser
//	selection  UISelectionFeedbackGenerator         EFFECT_TICK             5ms
//	light      UIImpactFeedbackGenerator(.light)    EFFECT_TICK             10ms
//	medium     UIImpactFeedbackGenerator(.medium)   EFFECT_CLICK            20ms
//	heavy      UIImpactFeedbackGenerator(.heavy)    EFFECT_HEAVY_CLICK      30ms
//	success    UINotificationFeedbackGenerator      EFFECT_DOUBLE_CLICK     10,40,10
//	warning    UINotificationFeedbackGenerator      waveform, two pulses    20,60,20
//	error      UINotificationFeedbackGenerator      waveform, three pulses  30,40,30,40,30
//
// Android below API 29 has no predefined effects and falls back to one-shot
// or waveform vibrations of similar length (see Haptics.kt).
//
// # Choosing a kind
//
// selection for a value ticking past (a picker, a slider detent); light /
// medium / heavy for a physical-feeling collision or press of that weight;
// success / warning / error for the outcome of something the user was
// waiting on. "An agent needs you" is a warning: something outside the app
// wants attention, nothing has failed.

// HapticKind names one haptic effect. See the table above for what each one
// maps to on every host.
type HapticKind string

const (
	HapticSelection HapticKind = "selection"
	HapticLight     HapticKind = "light"
	HapticMedium    HapticKind = "medium"
	HapticHeavy     HapticKind = "heavy"
	HapticSuccess   HapticKind = "success"
	HapticWarning   HapticKind = "warning"
	HapticError     HapticKind = "error"
)

// systemEventHaptic is the event every shell dispatches on, with the kind
// under "kind". Held to the shells by mobile/verify's
// TestHapticEventSpellingsAgree, which walks HapticKinds.
const systemEventHaptic = "haptic"

// HapticKinds lists every kind, in the order the table above documents them.
//
// It exists for the shell coverage test in mobile/verify rather than for
// apps: iterating core's own list is what makes adding an eighth kind fail
// that test until all three shells spell it, instead of the new kind silently
// doing nothing on whichever shell was forgotten.
func HapticKinds() []HapticKind {
	return []HapticKind{
		HapticSelection, HapticLight, HapticMedium, HapticHeavy,
		HapticSuccess, HapticWarning, HapticError,
	}
}

// Haptic asks the host to play one haptic effect.
//
// A kind outside HapticKinds is dropped here rather than sent: every shell
// would have to ignore it separately, and an older shell receiving a newer
// kind already ignores it, so the check in Go only catches a typo'd
// HapticKind("sucess") at the one place that can log nothing useful either
// way — the call simply does nothing, same as on a device with no motor.
func Haptic(kind HapticKind) {
	switch kind {
	case HapticSelection, HapticLight, HapticMedium, HapticHeavy,
		HapticSuccess, HapticWarning, HapticError:
	default:
		return
	}
	SendSystemEvent(systemEventHaptic, map[string]any{"kind": string(kind)})
}
