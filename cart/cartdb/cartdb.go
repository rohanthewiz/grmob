// Package cartdb keeps carts in a bytdb database: a cart.Persister that
// survives the app being killed, on any platform with a data directory.
//
//	db, err := cartdb.Open(filepath.Join(mobile.DataDir(), "cart.bytdb"))
//	if err != nil { … }                      // run in memory instead
//	store := cart.NewStore(cart.Config{Persister: db.Cart("default"), …})
//
// An app that already keeps its records in bytdb hands over its own engine
// instead (New), since bytdb holds an exclusive lock on its file and a
// second Open of the same path would fail.
//
// # Why its own package
//
// cart is imported by comps and hooks, and so by every GrMob app. Keeping
// the bytdb import here means an app that persists its cart elsewhere (its
// server, its own database) never links a storage engine it does not use.
//
// # The table
//
//	carts (name text PRIMARY KEY, body text)
//
// One row per named cart, holding the cart as JSON. A cart is read and
// written whole: it is tens of lines, it changes a line at a time on a tap,
// and storing it as rows would buy partial updates nothing needs at the
// price of a schema that must change whenever cart.Item gains a field. JSON
// absorbs new fields; an old row decodes with them zero, which every
// cart.Item field is designed to mean something sensible as.
package cartdb

import (
	"encoding/json"
	"fmt"

	"github.com/rohanthewiz/bytdb"
	bsql "github.com/rohanthewiz/bytdb/sql"
	"github.com/rohanthewiz/grmob/cart"
)

// DB is a bytdb database holding carts.
type DB struct {
	eng *bytdb.Engine
	sql *bsql.DB
	// owned says Close should close the engine: Open made it. An engine
	// handed to New belongs to the app, which closes it itself.
	owned bool
}

// Open opens (creating if need be) the bytdb file at path for carts.
func Open(path string) (*DB, error) {
	eng, err := bytdb.Open(path)
	if err != nil {
		return nil, fmt.Errorf("cartdb: opening %s: %w", path, err)
	}
	d, err := New(eng)
	if err != nil {
		_ = eng.Close()
		return nil, err
	}
	d.owned = true
	return d, nil
}

// New keeps carts in an engine the app already has open, creating the carts
// table if it is not there. Close does not close eng.
func New(eng *bytdb.Engine) (*DB, error) {
	d := &DB{eng: eng, sql: bsql.New(eng)}
	if _, err := d.sql.Exec(`CREATE TABLE IF NOT EXISTS carts (name text PRIMARY KEY, body text)`); err != nil {
		return nil, fmt.Errorf("cartdb: creating carts table: %w", err)
	}
	return d, nil
}

// Close closes the engine if Open made it.
func (d *DB) Close() error {
	if !d.owned {
		return nil
	}
	return d.eng.Close()
}

// Cart is the Persister for the cart called name. An app with one cart uses
// one name throughout; one with a cart per signed-in account names each by
// the account.
func (d *DB) Cart(name string) cart.Persister { return persister{db: d, name: name} }

// persister is one named row.
type persister struct {
	db   *DB
	name string
}

// Load reads the row, or returns an empty cart when there is none.
func (p persister) Load() (cart.Cart, error) {
	res, err := p.db.sql.Exec(`SELECT body FROM carts WHERE name = $1`, p.name)
	if err != nil {
		return cart.Cart{}, fmt.Errorf("cartdb: loading %q: %w", p.name, err)
	}
	if len(res.Rows) == 0 {
		return cart.Cart{}, nil
	}
	body, _ := res.Rows[0][0].(string)
	var c cart.Cart
	if err := json.Unmarshal([]byte(body), &c); err != nil {
		return cart.Cart{}, fmt.Errorf("cartdb: decoding %q: %w", p.name, err)
	}
	return c, nil
}

// Save upserts the row in one statement, so a crash leaves either the old
// cart or the new one, never neither.
func (p persister) Save(c cart.Cart) error {
	body, err := json.Marshal(c)
	if err != nil {
		return fmt.Errorf("cartdb: encoding %q: %w", p.name, err)
	}
	if _, err := p.db.sql.Exec(
		`INSERT INTO carts VALUES ($1, $2) ON CONFLICT (name) DO UPDATE SET body = excluded.body`,
		p.name, string(body),
	); err != nil {
		return fmt.Errorf("cartdb: saving %q: %w", p.name, err)
	}
	return nil
}
