package cartdb

import (
	"path/filepath"
	"reflect"
	"testing"

	"github.com/rohanthewiz/bytdb"
	"github.com/rohanthewiz/grmob/cart"
)

// A cart saved through a Store comes back from disk after a "relaunch":
// close the file, open it again, and make a fresh Store.
func TestCartSurvivesARelaunch(t *testing.T) {
	path := filepath.Join(t.TempDir(), "cart.bytdb")

	db, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	var saveErr error
	s := cart.NewStore(cart.Config{Persister: db.Cart("default"), OnError: func(err error) { saveErr = err }})
	s.Add(cart.Item{ID: "kettle", Label: "Kettle", UnitAmount: 4500, Meta: map[string]string{"sku": "K-1"}})
	s.Add(cart.Item{ID: "filters", Label: "Filters", UnitAmount: 650, Quantity: 2})
	s.SetQuantity("kettle", 3)
	if err := s.ApplyCode("ANY"); err == nil {
		t.Fatal("no rules: the default policy rejects every code")
	}
	s.Dispatch(cart.ApplyCode{Code: "welcome"})
	if saveErr != nil {
		t.Fatal(saveErr)
	}
	want := s.Cart()
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}

	db, err = Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	got := cart.NewStore(cart.Config{Persister: db.Cart("default")}).Cart()
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("after relaunch:\n got %+v\nwant %+v", got, want)
	}
}

func TestNamedCartsAreSeparateAndMissingIsEmpty(t *testing.T) {
	eng, err := bytdb.Open(filepath.Join(t.TempDir(), "app.bytdb"))
	if err != nil {
		t.Fatal(err)
	}
	defer eng.Close()
	db, err := New(eng)
	if err != nil {
		t.Fatal(err)
	}

	a, b := db.Cart("alice"), db.Cart("bob")
	if err := a.Save(cart.Cart{Items: []cart.Item{{ID: "x", Quantity: 1}}}); err != nil {
		t.Fatal(err)
	}
	got, err := b.Load()
	if err != nil || !got.IsEmpty() {
		t.Fatalf("bob's cart: %+v, %v", got, err)
	}
	// Close on a borrowed engine leaves it open for the app.
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}
	if got, err := a.Load(); err != nil || got.Count() != 1 {
		t.Fatalf("engine still open after DB.Close: %+v, %v", got, err)
	}
}
