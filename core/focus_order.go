package core

// Focus traversal: the order the input focus walks through a form, and the
// keyboard's "next" action that walks it.
//
// core.Focus made focus settable by *name*. This file makes it settable by
// *position*: a form declares the sequence its fields should be visited in,
// and the return key on every field but the last advances along it.
//
//	email    := core.UseFocusRef(ctx)
//	password := core.UseFocusRef(ctx)
//	confirm  := core.UseFocusRef(ctx)
//
//	core.UseFocusOrder(ctx, email, password, confirm)
//
//	core.Input(v, "you@example.com", onChange, core.FocusTarget(email))
//	core.InputPassword(p, "", onChange, core.FocusTarget(password))
//	core.InputPassword(c, "", onChange, core.FocusTarget(confirm))
//
// That is the whole app-facing surface for the common case: one line naming
// the order, and the fields already carry the FocusTarget that core.Focus
// needed anyway. The first two fields' keyboards now read "Next" and move the
// cursor down the form; the last keeps whatever submit action it was given.
//
// # Why the order is declared and not inferred
//
// The obvious alternative is the order the fields *render* in — what a
// browser's tab order and Compose's moveFocus(FocusDirection.Next) both use,
// and it needs no declaration at all. It was rejected for one concrete
// reason: a field cannot know, while it is being built, whether anything
// comes after it. Render order is only complete once the pass ends, and the
// props are stamped during the pass, so the IME action would have to be
// declared per field ("this one shows Next") — trading one line per form for
// one line per field, and a form that half-traverses when a field is missed.
//
// A declared order also survives the two things render order does not: a
// core.Cached subtree (which never re-renders, so it never re-registers) and
// a visual arrangement that does not match the tree (a two-column Row of
// fields, where reading order is not child order).
//
// # Where traversal is decided
//
// In Go, not on the platform. Compose could walk its own focus graph with
// moveFocus and SwiftUI could chain @FocusState itself, but then the order
// would be the platform's idea of it — derived from layout, differing between
// the two, and invisible to the Go code that declared it. Instead the IME
// action rides the existing onSubmit channel: the platform reports "the user
// pressed Next" exactly as it reports "the user pressed Done", and Go decides
// what that means. One round trip per press of the action key, on a channel
// that already carries every button tap.

// focusOrder is one traversal sequence: the refs of a form's fields in the
// order the return key should walk them.
//
// Rebuilt on every pass rather than kept in a hook slot, because it holds no
// state worth preserving — the sequence is recomputed from the arguments
// each time, which is what lets a conditionally rendered field join and leave
// the order as it appears and disappears. Identity is never compared across
// passes, so the per-pass allocation buys correctness for free.
type focusOrder struct {
	refs []*FocusRef
}

// UseFocusOrder declares the order the input focus walks through a set of
// fields. Call it in the component that renders those fields, above them:
//
//	core.UseFocusOrder(ctx, email, password, confirm)
//
// Two things follow from it. core.FocusNext and core.FocusPrevious can move
// relative to any ref in the list; and every field but the last advertises
// the keyboard's "next" action, which advances to the field after it. The
// last field advertises nothing new, so a form's final field keeps its own
// submit action — see FocusTarget for the rule when a field has both.
//
// # Why "above them" is not merely style
//
// Membership is read while a field's props are stamped, so a field rendered
// before this call runs sees the *previous* pass's membership. Hooks belong
// at the top of a render function anyway, and the fields of a form are
// rendered by the component that owns their refs, so the natural shape is
// already the correct one — but a call moved into a child component that
// renders after the fields would stamp a form that never advances.
//
// # It reserves no hook slot
//
// Despite the name, this is not a hook: everything it records lives on the
// refs, which are slot-stable already (see UseFocusRef), so there is nothing
// for a slot to hold. The Use prefix says where it belongs — inside a render
// function, on every pass — which is the part a caller has to get right. The
// consequence of not being a hook is only ever permissive: calling it
// conditionally is safe, where a real hook would drift the cursor.
//
// A nil ref in the list is skipped rather than panicking, so
// `core.UseFocusOrder(ctx, email, maybeRef, confirm)` degrades to the order
// without it — the same tolerance MaybeProp and FocusTarget have. A ref
// belonging to a different app's context is skipped for the same reason
// focusState is per-app: two apps in one process must not share an order.
//
// Listing a ref twice is allowed and the last position wins; there is no
// meaningful "field visited twice" and no reason to panic over a typo.
func UseFocusOrder(ctx *Context, refs ...*FocusRef) {
	if ctx == nil {
		return
	}
	// Copied rather than retained: `refs` is the caller's slice when the call
	// site spreads one (`UseFocusOrder(ctx, mine...)`), and the order would
	// then change under us whenever they wrote to it.
	kept := make([]*FocusRef, 0, len(refs))
	for _, ref := range refs {
		if ref == nil || ref.ctx == nil || ref.ctx.focus != ctx.focus {
			continue
		}
		kept = append(kept, ref)
	}
	ctx.focus.setOrder(kept)
}

// setOrder records refs as one traversal sequence, replacing whatever order
// each of them was in before.
//
// The ordered flag is sticky for the life of the app, and deliberately so: it
// is what FocusTarget consults to decide whether to stamp the imeAction key
// at all. An app that never declares an order renders exactly the trees it
// rendered before this file existed — the same guarantee the epoch-0 sentinel
// gives the focus commands — while an app that declares one anywhere gets the
// key on every named field forever after, so the key can never *vanish* from
// a node between passes. That matters because an update-props patch carries
// the whole new props map and the renderers iterate the keys it contains: a
// key that disappears is invisible on the far side, and a field dropped from
// an order would otherwise keep advertising a "next" that no longer exists.
func (f *focusState) setOrder(refs []*FocusRef) {
	f.mu.Lock()
	defer f.mu.Unlock()
	o := &focusOrder{refs: refs}
	for i, ref := range refs {
		ref.order = o
		ref.index = i
	}
	f.ordered = true
}

// neighbor returns the ref delta places from ref along its order, or nil at
// either end. Walking off the end is a no-op rather than a wrap: the last
// field of a form submits, it does not send the user back to the top, and a
// wrapping order would make "next" on the final field indistinguishable from
// a mis-declared one.
//
// A ref in no order has no neighbors, which is what makes traversal opt-in.
func (f *focusState) neighbor(ref *FocusRef, delta int) *FocusRef {
	if ref == nil {
		return nil
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	if ref.order == nil {
		return nil
	}
	i := ref.index + delta
	if i < 0 || i >= len(ref.order.refs) {
		return nil
	}
	return ref.order.refs[i]
}

// successor answers both questions FocusTarget has to ask, under one lock
// hold: what follows ref (nil if nothing), and whether this app has ever
// declared an order at all.
//
// The two are separate because they mean different things. No successor is
// the ordinary state of a form's last field, which still needs the key
// written — with an empty value — so the renderers see it change if the field
// later gains one. Never having declared an order is the state of every app
// written before this feature, which needs nothing written ever.
func (f *focusState) successor(ref *FocusRef) (next *FocusRef, ordered bool) {
	f.mu.Lock()
	ordered = f.ordered
	f.mu.Unlock()
	if !ordered {
		return nil, false
	}
	return f.neighbor(ref, 1), true
}

// stampTraversal writes the keyboard action for a field named by ref, and
// wires the action to the field that follows it.
//
// The submit ride is the whole trick: "the user pressed Next" reaches Go as
// an ordinary void callback on the onSubmit prop, which every renderer
// already dispatches, so advertising a Next key costs one string prop and no
// new bridge surface at all.
//
//	imeAction "next"  keyboard shows Next; onSubmit focuses the next field
//	          ""      keyboard shows whatever onSubmit alone implies
//
// An explicit onSubmit wins and suppresses the Next action. A field that was
// built with InputWithSubmit has been told, in so many words, what its return
// key does; silently replacing that with "advance" would break the one call
// shape whose entire purpose is to say otherwise. The cost is that a middle
// field cannot both submit and advance — which is not a shape a keyboard can
// express either, since it has one action key.
func stampTraversal(ctx *Context, n *Node, ref *FocusRef) {
	next, ordered := ctx.focus.successor(ref)
	if !ordered {
		return
	}
	if next == nil {
		n.Props["imeAction"] = ""
		return
	}
	if _, explicit := n.Props["onSubmit"]; explicit {
		n.Props["imeAction"] = ""
		return
	}
	n.Props["imeAction"] = "next"
	n.Props["onSubmit"] = ctx.registerCallback(func() { Focus(next) })
}

// FocusNext moves the input focus to the field after ref in its order.
//
// This is what the keyboard's Next action runs, and it is callable directly
// for the cases a keyboard cannot reach — a "Next" button drawn above the
// keyboard, a barcode scan that fills one field and should land in the next:
//
//	core.Button("Next", func() { core.FocusNext(current) })
//
// It takes the field to move *from* rather than reading "the focused field",
// because Go does not reliably know which field that is: the framework wires
// OnFocus only where an app asked for it, so the field the user tapped into
// is one Go was never told about (see focus.go). Naming the source is honest
// about that, and every caller has it — the keyboard action is stamped on a
// known field, and an app-drawn toolbar tracks the current field with OnFocus
// if it wants one.
//
// At the end of the order, or for a ref in no order at all, this does
// nothing. A nil ref is a no-op, matching Focus and FocusTarget.
func FocusNext(ref *FocusRef) {
	focusNeighbor(ref, 1)
}

// FocusPrevious moves the input focus to the field before ref in its order.
// See FocusNext for the shape and for why the source field is named.
//
// It has no keyboard action behind it on any platform here: neither the
// Android IME nor the iOS keyboard offers a "previous" key, and SwiftUI gives
// no input-accessory toolbar for free. This exists for the toolbar an app
// draws itself, above a KeyboardAware region, which is where a back-and-forth
// pair of arrows actually belongs.
func FocusPrevious(ref *FocusRef) {
	focusNeighbor(ref, -1)
}

// focusNeighbor is the shared body: find the neighbor and focus it.
//
// The end of the order needs no branch of its own. neighbor returns nil
// there, and Focus documents nil as a no-op — so walking off the end costs
// nothing and, importantly, consumes no epoch, which would otherwise re-stamp
// every field on screen for a command with no target. A guard here would be
// provably dead code saying the same thing twice.
//
// Reaching the end is silent. A form whose last field's Next key did nothing
// would be a declaration bug, not a runtime one, and there is no one to
// report it to on the far side of a keyboard.
func focusNeighbor(ref *FocusRef, delta int) {
	if ref == nil || ref.ctx == nil {
		return
	}
	Focus(ref.ctx.focus.neighbor(ref, delta))
}
