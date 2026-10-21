package cart

import (
	"log"
	"reflect"
	"sync"
)

// Persister keeps a cart between launches. Package cartdb has one backed by
// bytdb; anything that can hold a blob (a file, the app's own database, a
// server) can be one.
type Persister interface {
	// Load returns the saved cart, or an empty Cart and a nil error when
	// nothing has been saved yet.
	Load() (Cart, error)

	// Save replaces the saved cart with c.
	Save(c Cart) error
}

// Store is one cart shared by every screen of an app: the product page that
// adds to it, the app bar badge that counts it, and the cart screen that
// edits it all read the same Store, and a change made through any of them
// re-renders the others (through hooks.UseCart).
//
//	var store = cart.NewStore(cfg)               // one per app, made once
//
//	q := hooks.UseCart(ctx, store)               // in each screen
//	comps.CartList{
//	    Items:      q.Cart.Items,
//	    Currency:   q.Currency,
//	    OnQuantity: store.SetQuantity,           // plain setters: the
//	    OnRemove:   store.Remove,                // widgets' callback shapes
//	}
//
// # Why a store and not state in a screen
//
// core state lives in a context's hook slots, which belong to one tree
// position. A cart outlives every screen that shows it, and GrMob has no
// provider mechanism to hand one value down a navigation stack, so the cart
// lives here and screens subscribe to it. This is the shape the host-reported
// values already take (UseLocation, UsePermissionLive): a process-wide source
// and a hook that re-renders its caller when the source changes.
//
// # Concurrency
//
// Every method is safe from any goroutine. Changes are serialized by one
// mutex; the save runs under it, so saves land in the order the changes were
// made (bytdb's single-row writes take microseconds, the cost the todoapp
// example already accepts on its tap path). Subscribers are called after the
// lock is released, so one may read the Store back without deadlocking.
type Store struct {
	cfg Config

	mu   sync.Mutex
	cart Cart
	subs map[uint64]func()
	next uint64
}

// NewStore makes a Store with cfg, holding the cart cfg.Persister has saved
// (or an empty one). A load error goes to cfg.OnError and the store starts
// empty, for OnError's reason: an app with a broken file still works.
//
// A zero Store is also usable: an in-memory cart with Config{}.
func NewStore(cfg Config) *Store {
	s := &Store{cfg: cfg, subs: map[uint64]func(){}}
	if cfg.Persister != nil {
		c, err := cfg.Persister.Load()
		if err != nil {
			s.report(err)
		} else {
			s.cart = c
		}
	}
	return s
}

// Config is the Store's configuration, as NewStore was given it.
func (s *Store) Config() Config { return s.cfg }

// Cart is the current cart. See the package doc on snapshots: it is safe to
// keep and read, and must not be modified.
func (s *Store) Cart() Cart {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.cart
}

// Quote is the current cart, priced by the Store's rules.
func (s *Store) Quote() Quote { return s.cfg.Price(s.Cart()) }

// Dispatch applies action. When the cart actually changes it is saved and
// every subscriber is told; an action that changes nothing (a "+" at the
// cap, removing a line already gone) does neither, so a widget's no-op tap
// costs no render pass and no write.
func (s *Store) Dispatch(action Action) {
	s.mu.Lock()
	next := s.cfg.Reduce(s.cart, action)
	// DeepEqual rather than a changed flag threaded through every Action:
	// a cart is tens of lines, and the comparison is the one definition of
	// "changed" that cannot drift from what the actions do.
	if reflect.DeepEqual(next, s.cart) {
		s.mu.Unlock()
		return
	}
	s.cart = next
	var saveErr error
	if s.cfg.Persister != nil {
		saveErr = s.cfg.Persister.Save(next)
	}
	subs := make([]func(), 0, len(s.subs))
	for _, fn := range s.subs {
		subs = append(subs, fn)
	}
	s.mu.Unlock()

	// Reported outside the lock, like the subscribers are called: an
	// OnError that reads the Store (to show the cart it failed to save)
	// would otherwise deadlock.
	if saveErr != nil {
		s.report(saveErr)
	}
	for _, fn := range subs {
		fn()
	}
}

// Add puts it in the cart; see Add.
func (s *Store) Add(it Item) { s.Dispatch(Add{Item: it}) }

// SetQuantity sets a line's quantity; below one removes it. Its shape is
// CartList.OnQuantity's.
func (s *Store) SetQuantity(id string, quantity int) {
	s.Dispatch(SetQuantity{ID: id, Quantity: quantity})
}

// Remove takes a line out. Its shape is CartList.OnRemove's.
func (s *Store) Remove(id string) { s.Dispatch(Remove{ID: id}) }

// Clear empties the cart, codes included.
func (s *Store) Clear() { s.Dispatch(Clear{}) }

// ApplyCode applies a promo code when Config.AcceptCode takes it, and returns
// AcceptCode's error otherwise. A code already applied returns nil and
// changes nothing. Its shape is PromoCodeField.OnApply's, so the field shows
// the error under the input.
func (s *Store) ApplyCode(code string) error {
	code = NormalizeCode(code)
	if code == "" {
		return nil
	}
	if err := s.cfg.accept(code, s.Cart()); err != nil {
		return err
	}
	s.Dispatch(ApplyCode{Code: code})
	return nil
}

// RemoveCode takes a promo code off. Its shape is PromoCodeField.OnRemove's.
func (s *Store) RemoveCode(code string) { s.Dispatch(RemoveCode{Code: code}) }

// Subscribe registers fn to run after every change, and returns the function
// that unregisters it. fn runs on the goroutine that made the change, after
// the Store's lock is released. Cancelling twice is harmless.
func (s *Store) Subscribe(fn func()) (cancel func()) {
	if fn == nil {
		return func() {}
	}
	s.mu.Lock()
	// Made here as well as in NewStore so a zero Store is usable.
	if s.subs == nil {
		s.subs = map[uint64]func(){}
	}
	id := s.next
	s.next++
	s.subs[id] = fn
	s.mu.Unlock()
	return func() {
		s.mu.Lock()
		delete(s.subs, id)
		s.mu.Unlock()
	}
}

// report hands a persistence error to OnError, or logs it.
func (s *Store) report(err error) {
	if s.cfg.OnError != nil {
		s.cfg.OnError(err)
		return
	}
	log.Printf("cart: %v", err)
}
