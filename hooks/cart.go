package hooks

import (
	"sync"

	"github.com/rohanthewiz/grmob/cart"
	"github.com/rohanthewiz/grmob/core"
)

// cartRecord is the per-slot memory of one UseCart: which store this slot is
// subscribed to and how to let go of it. It lives in the hook-slot array (via
// core.NewState) for the reasons locationRecord does. The mutex covers the
// close path, which runs on whichever goroutine closes the context.
//
// The cart itself is not kept here. The Store is the source of truth and is
// read fresh on every pass, so there is no copy in the slot to fall out of
// step with it: a change made between the notification and the render that
// it triggers is simply in that render.
type cartRecord struct {
	mu     sync.Mutex
	store  *cart.Store
	cancel func()
	// closing says the release is registered with the context: once for the
	// life of the slot, because ctx.OnClose has no unregister.
	closing bool
}

// follow subscribes the slot to s, dropping any earlier store's
// subscription. A nil s subscribes to nothing.
func (r *cartRecord) follow(ctx *core.Context, s *cart.Store) {
	r.mu.Lock()
	if r.store == s && (r.cancel != nil || s == nil) {
		r.mu.Unlock()
		return
	}
	old := r.cancel
	r.store, r.cancel = s, nil
	r.mu.Unlock()

	if old != nil {
		old()
	}
	if s == nil {
		return
	}
	// RequestRender, not a State.Set: there is no value to store, only the
	// news that the store changed, and the next pass reads it from there.
	cancel := s.Subscribe(ctx.RequestRender)
	r.mu.Lock()
	r.cancel = cancel
	r.mu.Unlock()
}

// release drops the subscription. Idempotent, and it forgets the store, so a
// re-mount over the same context (the drain semantics of ctx.Close)
// subscribes again.
func (r *cartRecord) release() {
	r.mu.Lock()
	cancel := r.cancel
	r.store, r.cancel = nil, nil
	r.mu.Unlock()
	if cancel != nil {
		cancel()
	}
}

// UseCart returns the store's cart, priced, and re-renders the calling
// component whenever the cart changes, from this screen or any other:
//
//	q := hooks.UseCart(ctx, store)
//	return core.Column(
//	    comps.CartList{Items: q.Cart.Items, Currency: q.Currency,
//	        OnQuantity: store.SetQuantity, OnRemove: store.Remove},
//	    comps.CartSummary{Quote: q},
//	)
//
// It is the bridge cart.Store's doc describes: the cart outlives every
// screen, so it lives in the store, and each screen that shows it calls this
// once per pass. A product page and an app bar badge each calling UseCart on
// the same store stay in step without passing anything between them.
//
// # One slot, every pass
//
// The hook takes one slot, so the hook rules apply: call it unconditionally.
// The store may change between passes (a sign-in swaps the account's cart);
// the slot moves its subscription to the new one. A nil store returns an
// empty Quote and subscribes to nothing.
//
// # When the subscription ends
//
// As with every hook here, on the context's Close, not when the component
// leaves the tree: a screen pushed on the navigation stack lets go when its
// frame is popped. A subscription costs a map entry and a RequestRender per
// change, so one outliving its screen is harmless.
func UseCart(ctx *core.Context, s *cart.Store) cart.Quote {
	slot := core.NewState(ctx, &cartRecord{})
	rec := slot.Get()

	// Registered before subscribing, and re-armed by the close itself: a
	// Close drains the context's registry, so after one the next pass must
	// register again, which clearing closing lets it do (UseLocationWhen's
	// shape).
	rec.mu.Lock()
	register := !rec.closing
	rec.closing = true
	rec.mu.Unlock()
	if register {
		ctx.OnClose(func() {
			rec.release()
			rec.mu.Lock()
			rec.closing = false
			rec.mu.Unlock()
		})
	}
	rec.follow(ctx, s)

	if s == nil {
		return cart.Config{}.Price(cart.Cart{})
	}
	return s.Quote()
}
