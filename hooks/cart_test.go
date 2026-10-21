package hooks_test

import (
	"testing"
	"time"

	"github.com/rohanthewiz/grmob/cart"
	"github.com/rohanthewiz/grmob/core"
	"github.com/rohanthewiz/grmob/hooks"
)

// UseCart must hand back the store's priced cart, re-render on every change
// made anywhere, follow a swapped store, and let go on close.
func TestUseCartRendersOnChangesFollowsTheStoreAndReleasesOnClose(t *testing.T) {
	ctx := core.NewContext()
	renders := make(chan struct{}, 16)
	ctx.OnStateChange(func() { renders <- struct{}{} })

	store := cart.NewStore(cart.Config{Currency: "eur"})
	var store2 *cart.Store
	current := store
	var q cart.Quote
	body := func(ctx *core.Context) { q = hooks.UseCart(ctx, current) }

	renderPass(ctx, body)
	renderPass(ctx, body) // a second pass must not stack a subscription
	if q.Currency != "EUR" || q.Count() != 0 {
		t.Fatalf("initial quote %+v", q)
	}

	// A change made somewhere else (another screen) reaches this one.
	store.Add(cart.Item{ID: "a", UnitAmount: 250, Quantity: 2})
	awaitSignal(t, renders, "render request after Add")
	assertQuiet(t, renders, 30*time.Millisecond, "a stacked subscription rendered twice")
	renderPass(ctx, body)
	if q.Total != 500 || q.Count() != 2 {
		t.Fatalf("after Add: %+v", q)
	}

	// A no-op change requests nothing.
	store.SetQuantity("missing", 3)
	assertQuiet(t, renders, 30*time.Millisecond, "render for a change that changed nothing")

	// Swapping stores moves the subscription.
	store2 = cart.NewStore(cart.Config{})
	current = store2
	renderPass(ctx, body)
	store.Add(cart.Item{ID: "b", UnitAmount: 1})
	assertQuiet(t, renders, 30*time.Millisecond, "the old store still re-renders this slot")
	store2.Add(cart.Item{ID: "c", UnitAmount: 1})
	awaitSignal(t, renders, "render request from the new store")

	ctx.Close()
	store2.Add(cart.Item{ID: "d", UnitAmount: 1})
	assertQuiet(t, renders, 30*time.Millisecond, "subscription survived Close")

	// A re-mount over the closed context subscribes again.
	renderPass(ctx, body)
	store2.Add(cart.Item{ID: "e", UnitAmount: 1})
	awaitSignal(t, renders, "render request after re-mount")
}

func TestUseCartWithANilStoreIsEmpty(t *testing.T) {
	ctx := core.NewContext()
	var q cart.Quote
	renderPass(ctx, func(ctx *core.Context) { q = hooks.UseCart(ctx, nil) })
	if q.Count() != 0 || q.Currency != "USD" {
		t.Fatalf("nil store quote %+v", q)
	}
}
