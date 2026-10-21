package core

// SwipeDirection is the way a finger travelled in a swipe, as OnSwipe and
// OnVerticalSwipe report it.
//
// The horizontal pair is logical, not physical: SwipeLeading means the finger
// moved toward the leading edge, which is left in a left-to-right layout and
// right in a right-to-left one. That is the direction a reading-order widget
// cares about. A carousel steps forward on SwipeLeading in either direction
// with no layout check of its own, and Go could not make that check anyway,
// because only the host knows the direction it laid the node out in. Each
// host resolves it there, the way a CanvasMirrorsRTL canvas's taps are
// reflected on the host before Go sees them.
//
// The values are the wire payload, so a handler can switch on them directly.
type SwipeDirection string

const (
	// SwipeLeading: the finger moved toward the leading edge (left in LTR).
	// Content that follows the finger reveals what comes after.
	SwipeLeading SwipeDirection = "leading"
	// SwipeTrailing: the finger moved toward the trailing edge (right in
	// LTR), revealing what comes before.
	SwipeTrailing SwipeDirection = "trailing"
	// SwipeUp: the finger moved up the screen.
	SwipeUp SwipeDirection = "up"
	// SwipeDown: the finger moved down the screen.
	SwipeDown SwipeDirection = "down"
)

// SwipeMinDistance is how far, in layout units (CSS px, dp, pt), a finger has
// to travel along the swipe's axis between press and release for the gesture
// to count. Every host reads the same number, so a swipe written once in Go
// takes the same hand on all three.
//
// Distance alone, not velocity. A fling detector differs per platform
// (Compose's VelocityTracker, UIKit's predicted end, nothing at all in the
// DOM), and porting one into each host would be three subtly different
// gestures. A deliberate 40-unit drag is a swipe everywhere. A tap's jitter
// (under 10 units on every platform's own slop) never is.
const SwipeMinDistance = 40

// OnSwipe fires once when a horizontal swipe on the node ends: a press that
// travelled at least SwipeMinDistance sideways, and further sideways than up
// or down, before it lifted. The handler gets SwipeLeading or SwipeTrailing.
//
//	core.Box(
//	    core.OnSwipe(func(d core.SwipeDirection) {
//	        if d == core.SwipeLeading {
//	            next()
//	        } else {
//	            prev()
//	        }
//	    }),
//	    card,
//	)
//
// # A discrete gesture, not a drag
//
// The report is one event at release, carrying a direction, with no stream of
// deltas while the finger moves. A per-frame delta would be a callback and a
// render pass per pointer move over the bridge, and the reason to want one
// (content that tracks the finger) is a host animation core does not have.
// What a swipe answers is "which way", and comps.Gallery, the widget that
// asked for it, needs only that.
//
// # One axis claimed
//
// The node claims only the horizontal axis, so a swipeable card inside a
// vertical Scroll still scrolls the page when the finger moves up or down.
// Each host gives the vertical motion back in its own way:
//
//	Web      touch-action: pan-y pinch-zoom on the node, so the browser keeps
//	         vertical panning and hands horizontal motion to the page
//	Android  detectHorizontalDragGestures' slop, which only claims a drag
//	         that crosses the touch slop sideways
//	iOS      a simultaneous DragGesture, which never blocks the ScrollView's
//	         own pan; the axis is decided at release
//
// OnVerticalSwipe claims the other axis. A node carrying both claims both,
// and a release reports on whichever axis the finger travelled further along.
//
// # Taps under a swipe
//
// A swipe that fires is not also a tap. A child's OnClick or OnLongPress
// under the finger does not run for the same press. Compose consumes the
// drag's pointer changes, which cancels a child's clickable. The DOM runtime
// swallows the click the browser sends after the release, and a long press
// is disarmed once the finger moves past the slop, as both native
// recognizers disarm theirs. SwiftUI's tap recognizer already fails once
// the finger travels.
//
// A press that moves less than SwipeMinDistance fires nothing on the swipe.
// It is still a tap if it stayed within the platform's tap slop.
//
// # Disabled
//
// A node that is Disabled drops the swipe in Go, so the handler does not run
// on any host. The check is made when the event arrives, against the node as
// this pass built it, so a Disabled prop written after OnSwipe still counts.
//
// # Wire
//
// "onSwipe", a text callback whose payload is the direction's value. A
// payload that is not "leading" or "trailing" is dropped, never guessed at.
// htmlout records the ID as data-onswipe, like onShapeTap's: a static page
// has no pointer to read, but a loader that has one knows what to call.
func OnSwipe(handler func(SwipeDirection)) BehaviorProp {
	return swipeProp("onSwipe", handler, SwipeLeading, SwipeTrailing)
}

// OnVerticalSwipe fires once when a vertical swipe on the node ends, with
// SwipeUp or SwipeDown. It is OnSwipe turned a quarter: the same distance,
// the same "further along this axis than the other" rule, and it claims only
// the vertical axis (touch-action: pan-x on the web), so a horizontal Scroll
// around it still pans.
//
// A node inside a vertical Scroll that carries it takes the vertical drags
// that would have scrolled, on Android and the web. That is what claiming an
// axis means. Use it on something that does not sit in a vertical scroll, or
// that should stop the scroll: a sheet's grab handle, a full-screen viewer.
//
// Wire: "onVerticalSwipe", a text callback; payload "up" or "down".
func OnVerticalSwipe(handler func(SwipeDirection)) BehaviorProp {
	return swipeProp("onVerticalSwipe", handler, SwipeUp, SwipeDown)
}

// swipeProp registers a swipe handler under key, accepting only the two
// directions of its axis.
//
// The closure holds the node rather than reading Disabled now. Props apply in
// argument order, so a core.Disabled written after OnSwipe has not landed yet
// when this runs. By the time a host dispatches, the pass has finished and n
// is the node it sent. A later pass builds a new node and registers a new
// closure under the same ID (see beginPass), so a stale n is never read.
func swipeProp(key string, handler func(SwipeDirection), a, b SwipeDirection) BehaviorProp {
	return behaviorFunc(func(ctx *Context, n *Node) {
		if handler == nil {
			// A nil handler claims no axis. Registering a no-op would make
			// the node take the gesture away from a Scroll around it for
			// nothing.
			return
		}
		if n.Props == nil {
			n.Props = map[string]any{}
		}
		n.Props[key] = ctx.registerTextCallback(func(payload string) {
			d := SwipeDirection(payload)
			if d != a && d != b {
				return
			}
			if n.Style != nil && n.Style.Disabled {
				return
			}
			handler(d)
		})
	})
}
