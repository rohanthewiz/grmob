package cart

import (
	"errors"
	"reflect"
	"sync"
	"testing"
)

var kettle = Item{ID: "kettle", Label: "Pour-over kettle", UnitAmount: 4500}
var filters = Item{ID: "filters", Label: "Filters (100)", UnitAmount: 650}

func TestAddMergesByIDAndClampsToTheCap(t *testing.T) {
	cfg := Config{MaxQuantity: 3}
	c := cfg.Reduce(Cart{}, Add{Item: kettle})
	c = cfg.Reduce(c, Add{Item: filters})
	c = cfg.Reduce(c, Add{Item: Item{ID: "kettle", Label: "Kettle v2", UnitAmount: 4000, Quantity: 5}})

	if len(c.Items) != 2 || c.Items[0].ID != "kettle" {
		t.Fatalf("merged line keeps its place: %+v", c.Items)
	}
	k := c.Items[0]
	if k.Quantity != 3 || k.MaxQuantity != 3 {
		t.Errorf("1+5 clamps to the config cap 3, carried on the line: got qty %d max %d", k.Quantity, k.MaxQuantity)
	}
	if k.Label != "Kettle v2" || k.UnitAmount != 4000 {
		t.Errorf("a re-added item's newest fields win: %+v", k)
	}
	if c.Count() != 4 {
		t.Errorf("Count = %d, want 4", c.Count())
	}
}

func TestItemCapOutranksConfigCap(t *testing.T) {
	cfg := Config{MaxQuantity: 10}
	c := cfg.Reduce(Cart{}, Add{Item: Item{ID: "x", MaxQuantity: 2, Quantity: 9}})
	if c.Items[0].Quantity != 2 {
		t.Fatalf("qty %d, want the item's own cap 2", c.Items[0].Quantity)
	}
}

func TestSetQuantityBelowOneRemoves(t *testing.T) {
	cfg := Config{}
	c := cfg.Reduce(Cart{}, Add{Item: kettle})
	c = cfg.Reduce(c, SetQuantity{ID: "kettle", Quantity: 4})
	if c.Items[0].Quantity != 4 {
		t.Fatalf("qty %d, want 4", c.Items[0].Quantity)
	}
	c = cfg.Reduce(c, SetQuantity{ID: "kettle", Quantity: 0})
	if !c.IsEmpty() {
		t.Fatalf("quantity 0 removes the line: %+v", c.Items)
	}
	if got := cfg.Reduce(c, SetQuantity{ID: "nope", Quantity: 2}); !reflect.DeepEqual(got, c) {
		t.Errorf("unknown ID is ignored")
	}
}

// Every earlier snapshot must survive later actions untouched: render code
// holds them without a lock.
func TestReduceNeverModifiesItsInput(t *testing.T) {
	cfg := Config{}
	base := cfg.Reduce(cfg.Reduce(Cart{}, Add{Item: kettle}), Add{Item: filters})
	base = cfg.Reduce(base, ApplyCode{Code: "a"})
	before := Cart{
		Items: append([]Item(nil), base.Items...),
		Codes: append([]string(nil), base.Codes...),
	}
	for _, a := range []Action{
		Add{Item: kettle}, SetQuantity{ID: "filters", Quantity: 7}, Remove{ID: "kettle"},
		ApplyCode{Code: "b"}, RemoveCode{Code: "A"}, Clear{},
	} {
		_ = cfg.Reduce(base, a)
		if !reflect.DeepEqual(base, before) {
			t.Fatalf("%T modified its input: %+v", a, base)
		}
	}
}

func TestCodesAreNormalizedAndDeduped(t *testing.T) {
	cfg := Config{}
	c := cfg.Reduce(Cart{}, ApplyCode{Code: " spring10 "})
	c = cfg.Reduce(c, ApplyCode{Code: "SPRING10"})
	if !reflect.DeepEqual(c.Codes, []string{"SPRING10"}) {
		t.Fatalf("codes %v", c.Codes)
	}
	if !c.HasCode("Spring10") {
		t.Error("HasCode compares normalized")
	}
	c = cfg.Reduce(c, RemoveCode{Code: "spring10"})
	if len(c.Codes) != 0 {
		t.Errorf("RemoveCode normalizes too: %v", c.Codes)
	}
}

func TestPriceRunsRulesInOrderOnTheRunningTotal(t *testing.T) {
	cfg := Config{
		Currency: "usd",
		Rules: []Rule{
			PercentOff("SPRING10", "Spring sale", Percent(10)),
			FlatShipping("Shipping", 599, 5000),
			Tax("Tax", Percent(8.25)),
		},
	}
	c := cfg.Reduce(Cart{}, Add{Item: Item{ID: "k", UnitAmount: 4500, Quantity: 1}})

	q := cfg.Price(c)
	want := []Adjustment{{Label: "Shipping", Amount: 599}, {Label: "Tax", Amount: 421}}
	if !reflect.DeepEqual(q.Adjustments, want) || q.Total != 4500+599+421 {
		t.Fatalf("no code: %+v total %d", q.Adjustments, q.Total)
	}
	if q.Currency != "USD" {
		t.Errorf("currency %q, want USD", q.Currency)
	}

	// $60 subtotal: 10% off is $6, leaving $54 ≥ $50, so shipping is free;
	// tax is 8.25% of $54 = $4.455 → $4.46 (half away from zero).
	c = cfg.Reduce(Cart{}, Add{Item: Item{ID: "k", UnitAmount: 6000}})
	c = cfg.Reduce(c, ApplyCode{Code: "spring10"})
	q = cfg.Price(c)
	want = []Adjustment{
		{Label: "Spring sale", Amount: -600, Code: "SPRING10"},
		{Label: "Shipping", Amount: 0},
		{Label: "Tax", Amount: 446},
	}
	if !reflect.DeepEqual(q.Adjustments, want) || q.Total != 6000-600+446 {
		t.Fatalf("with code: %+v total %d", q.Adjustments, q.Total)
	}
}

func TestEmptyCartIsPricedWithoutRules(t *testing.T) {
	cfg := Config{Rules: []Rule{FlatShipping("Shipping", 599, 0)}}
	q := cfg.Price(Cart{Codes: []string{"X"}})
	if len(q.Adjustments) != 0 || q.Total != 0 {
		t.Fatalf("empty cart: %+v", q)
	}
}

func TestDiscountsNeverGoNegative(t *testing.T) {
	cfg := Config{Rules: []Rule{AmountOff("", "Gift card", 10000), PercentOff("", "Half", Percent(50))}}
	q := cfg.Price(Cart{Items: []Item{{ID: "a", UnitAmount: 3000, Quantity: 1}}})
	if q.Total != 0 || len(q.Adjustments) != 1 || q.Adjustments[0].Amount != -3000 {
		t.Fatalf("got %+v", q)
	}
}

func TestRateOfRoundsHalfAwayFromZero(t *testing.T) {
	cases := []struct {
		r    Rate
		in   int64
		want int64
	}{
		{Percent(8.25), 5400, 446}, // 445.5
		{Percent(8.25), -5400, -446},
		{Percent(10), 5, 1}, // 0.5
		{Percent(10), 4, 0}, // 0.4
		{Percent(0), 9999, 0},
	}
	for _, c := range cases {
		if got := c.r.Of(c.in); got != c.want {
			t.Errorf("%d bp of %d = %d, want %d", c.r, c.in, got, c.want)
		}
	}
}

func TestZeroRuleIsHarmless(t *testing.T) {
	q := Config{Rules: []Rule{{}}}.Price(Cart{Items: []Item{{ID: "a", UnitAmount: 1, Quantity: 1}}})
	if q.Total != 1 {
		t.Fatalf("total %d", q.Total)
	}
}

// memPersister is a Persister that records every save.
type memPersister struct {
	saved   Cart
	saves   int
	loadErr error
	saveErr error
}

func (m *memPersister) Load() (Cart, error) { return m.saved, m.loadErr }
func (m *memPersister) Save(c Cart) error {
	m.saves++
	m.saved = c
	return m.saveErr
}

func TestStoreNotifiesAndPersistsOnlyRealChanges(t *testing.T) {
	p := &memPersister{}
	s := NewStore(Config{MaxQuantity: 2, Persister: p})
	calls := 0
	cancel := s.Subscribe(func() {
		calls++
		_ = s.Cart() // reading back from a subscriber must not deadlock
	})

	s.Add(kettle)
	s.SetQuantity("kettle", 2)
	s.SetQuantity("kettle", 5) // clamps to 2: no change
	s.Remove("missing")        // no change
	if calls != 2 || p.saves != 2 {
		t.Fatalf("calls %d saves %d, want 2 and 2", calls, p.saves)
	}
	if p.saved.Items[0].Quantity != 2 {
		t.Errorf("saved %+v", p.saved)
	}

	cancel()
	cancel()
	s.Clear()
	if calls != 2 {
		t.Errorf("cancelled subscriber still called")
	}
}

func TestStoreLoadsFromItsPersister(t *testing.T) {
	p := &memPersister{saved: Cart{Items: []Item{{ID: "a", Quantity: 2, UnitAmount: 100}}}}
	s := NewStore(Config{Persister: p})
	if s.Quote().Total != 200 {
		t.Fatalf("loaded total %d", s.Quote().Total)
	}
}

func TestStorePersistenceErrorsGoToOnError(t *testing.T) {
	boom := errors.New("disk full")
	p := &memPersister{loadErr: boom, saveErr: boom}
	var got []error
	var s *Store
	s = NewStore(Config{Persister: p, OnError: func(err error) {
		got = append(got, err)
		if s != nil {
			_ = s.Cart() // OnError may read the store
		}
	}})
	s.Add(kettle)
	if len(got) != 2 {
		t.Fatalf("errors %v, want load and save", got)
	}
	if s.Cart().IsEmpty() {
		t.Error("a failed save keeps the change in memory")
	}
}

func TestStoreApplyCodeAsksAcceptCode(t *testing.T) {
	s := NewStore(Config{Rules: []Rule{PercentOff("SPRING10", "Sale", Percent(10))}})
	if err := s.ApplyCode("nope"); !errors.Is(err, ErrUnknownCode) {
		t.Fatalf("unknown code: %v", err)
	}
	if err := s.ApplyCode("spring10"); err != nil {
		t.Fatalf("known code: %v", err)
	}
	if !reflect.DeepEqual(s.Cart().Codes, []string{"SPRING10"}) {
		t.Fatalf("codes %v", s.Cart().Codes)
	}

	minOrder := errors.New("needs $50")
	s = NewStore(Config{AcceptCode: func(code string, c Cart) error {
		if c.Subtotal() < 5000 {
			return minOrder
		}
		return nil
	}})
	if err := s.ApplyCode("ANY"); err != minOrder {
		t.Fatalf("custom policy: %v", err)
	}
}

func TestZeroStoreIsUsable(t *testing.T) {
	var s Store
	called := false
	s.Subscribe(func() { called = true })
	s.Add(kettle)
	if !called || s.Cart().Count() != 1 {
		t.Fatal("zero Store works in memory")
	}
}

func TestStoreIsSafeForConcurrentUse(t *testing.T) {
	s := NewStore(Config{})
	s.Subscribe(func() { _ = s.Quote() })
	var wg sync.WaitGroup
	for i := range 8 {
		wg.Go(func() {
			for j := range 50 {
				s.Add(Item{ID: string(rune('a' + i)), UnitAmount: 1})
				_ = s.Quote()
				if j%10 == 0 {
					s.Remove(string(rune('a' + i)))
				}
			}
		})
	}
	wg.Wait()
}
