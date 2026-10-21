# Package cartdb

```go
import "github.com/rohanthewiz/grmob/cart/cartdb"
```

Package cartdb keeps carts in a bytdb database: a cart.Persister that survives the app being killed, on any platform with a data directory.

	db, err := cartdb.Open(filepath.Join(mobile.DataDir(), "cart.bytdb"))
	if err != nil { … }                      // run in memory instead
	store := cart.NewStore(cart.Config{Persister: db.Cart("default"), …})

An app that already keeps its records in bytdb hands over its own engine instead (New), since bytdb holds an exclusive lock on its file and a second Open of the same path would fail.

## Why its own package

cart is imported by comps and hooks, and so by every GrMob app. Keeping the bytdb import here means an app that persists its cart elsewhere (its server, its own database) never links a storage engine it does not use.

## The table

	carts (name text PRIMARY KEY, body text)

One row per named cart, holding the cart as JSON. A cart is read and written whole: it is tens of lines, it changes a line at a time on a tap, and storing it as rows would buy partial updates nothing needs at the price of a schema that must change whenever cart.Item gains a field. JSON absorbs new fields; an old row decodes with them zero, which every cart.Item field is designed to mean something sensible as.

## Index

- [`type DB`](#type-db)
    - [`func New`](#func-new)
    - [`func Open`](#func-open)
    - [`func (*DB) Cart`](#func-db-cart)
    - [`func (*DB) Close`](#func-db-close)

## Types

### type DB

```go
type DB struct {
	// contains filtered or unexported fields
}
```

DB is a bytdb database holding carts.

<small>[cart/cartdb/cartdb.go:40](https://github.com/rohanthewiz/grmob/blob/master/cart/cartdb/cartdb.go#L40)</small>

#### func New

```go
func New(eng *bytdb.Engine) (*DB, error)
```

New keeps carts in an engine the app already has open, creating the carts table if it is not there. Close does not close eng.

<small>[cart/cartdb/cartdb.go:65](https://github.com/rohanthewiz/grmob/blob/master/cart/cartdb/cartdb.go#L65)</small>

#### func Open

```go
func Open(path string) (*DB, error)
```

Open opens (creating if need be) the bytdb file at path for carts.

<small>[cart/cartdb/cartdb.go:49](https://github.com/rohanthewiz/grmob/blob/master/cart/cartdb/cartdb.go#L49)</small>

#### func (*DB) Cart

```go
func (d *DB) Cart(name string) cart.Persister
```

Cart is the Persister for the cart called name. An app with one cart uses one name throughout; one with a cart per signed-in account names each by the account.

<small>[cart/cartdb/cartdb.go:84](https://github.com/rohanthewiz/grmob/blob/master/cart/cartdb/cartdb.go#L84)</small>

#### func (*DB) Close

```go
func (d *DB) Close() error
```

Close closes the engine if Open made it.

<small>[cart/cartdb/cartdb.go:74](https://github.com/rohanthewiz/grmob/blob/master/cart/cartdb/cartdb.go#L74)</small>

