// Store: bytdb-backed persistence for the todo list.
//
// The shape of the integration matters more than the SQL. Loading could ride
// hooks.UseEffect, but effects run on their own goroutine — the first frame
// would mount empty and the rows would pop in a patch later. Instead the
// store opens lazily on the first render pass (openStore is cheap to call
// repeatedly), the snapshot it read at open seeds the NewState initial values
// — so persisted rows are already in the initial tree — and every later
// write goes through the mutation helpers in app.go — the
// single choke point — as a synchronous write-through. bytdb's single-row
// writes are microseconds (WAL, fsync-before-ack), so riding the event path
// costs nothing perceptible and means there is no dirty-flag or debounce
// machinery to get wrong.
//
//	first render ──▶ openStore ──▶ SELECT * ──▶ NewState initial
//	tap/submit  ──▶ mutation helper ──▶ State.Set (UI)
//	                              └───▶ store write-through (disk)
//
// Persistence is strictly optional: with no data directory registered
// (mobile.SetDataDir unset — web preview, bare unit tests) openStore returns
// nil, and every method below is nil-receiver-safe, so app.go calls them
// unconditionally instead of branching at each site.
//
// (Blank line below keeps this a file comment — app.go owns the package doc.)

package todoapp

import (
	"log"
	"path/filepath"
	"sync"

	"github.com/rohanthewiz/grmob/mobile"
	"github.com/rohanthewiz/bytdb"
	bsql "github.com/rohanthewiz/bytdb/sql"
)

// todoStore wraps one open bytdb engine plus the snapshot read at open.
// The snapshot is deliberately a one-time read, not a live mirror: after the
// first render the Context's state is the in-process source of truth and the
// table only needs to be correct for the *next* process. (Tests simulate a
// relaunch with closeStore + re-Register, which forces a fresh read.)
type todoStore struct {
	eng  *bytdb.Engine
	db   *bsql.DB
	path string

	initial []Todo // rows as of open, id-ascending
	nextID  int    // max(id)+1 as of open — keeps IDs unique across launches
}

// The open store is a package-level singleton because the bridge itself is
// one process-wide app: bytdb holds an exclusive file lock, so a second Open
// of the same path would fail. The mutex covers open/close races between the
// render goroutine and tests.
var (
	storeMu sync.Mutex
	store   *todoStore
)

// openStore returns the store for the current data directory, opening it on
// first use. It is called on every render pass; after the first call it is a
// mutex acquire and a string compare. A changed data directory (tests moving
// to a fresh t.TempDir) closes the old engine and opens the new path.
func openStore() *todoStore {
	storeMu.Lock()
	defer storeMu.Unlock()

	dir := mobile.DataDir()
	if dir == "" {
		return nil // no shell-provided directory: run in-memory
	}
	path := filepath.Join(dir, "todos.bytdb")
	if store != nil && store.path == path {
		return store
	}
	if store != nil {
		if err := store.eng.Close(); err != nil {
			log.Printf("todoapp: closing previous store: %v", err)
		}
		store = nil
	}

	eng, err := bytdb.Open(path)
	if err != nil {
		// A failed open degrades to in-memory rather than crashing the app:
		// losing persistence beats losing the UI. The error is logged once
		// here, not retried per render — openStore would loop on Open
		// otherwise, so a nil store with the path recorded would be nicer,
		// but a plain nil keeps the retry cheap and harmless (Open on a bad
		// path fails fast).
		log.Printf("todoapp: opening store at %s: %v", path, err)
		return nil
	}
	db := bsql.New(eng)

	if _, err := db.Exec(
		`CREATE TABLE IF NOT EXISTS todos (id int PRIMARY KEY, title text, done boolean)`,
	); err != nil {
		log.Printf("todoapp: creating todos table: %v", err)
		_ = eng.Close()
		return nil
	}

	s := &todoStore{eng: eng, db: db, path: path, nextID: 1}

	// ORDER BY id restores insertion order — the order the list renders in.
	res, err := db.Exec(`SELECT id, title, done FROM todos ORDER BY id`)
	if err != nil {
		log.Printf("todoapp: loading todos: %v", err)
		_ = eng.Close()
		return nil
	}
	for _, row := range res.Rows {
		t := Todo{ID: asInt(row[0])}
		t.Title, _ = row[1].(string)
		t.Done, _ = row[2].(bool)
		s.initial = append(s.initial, t)
		if t.ID >= s.nextID {
			s.nextID = t.ID + 1
		}
	}

	store = s
	return s
}

// closeStore releases the engine and forgets the singleton. The app never
// calls this — the engine lives for the process, and bytdb's WAL makes a
// hard kill safe — but tests use it to simulate a relaunch: close, re-open,
// and the next snapshot comes from disk instead of memory.
func closeStore() {
	storeMu.Lock()
	defer storeMu.Unlock()
	if store == nil {
		return
	}
	if err := store.eng.Close(); err != nil {
		log.Printf("todoapp: closing store: %v", err)
	}
	store = nil
}

// snapshot returns the rows and next ID read at open. The slice is copied
// because NewState keeps a reference to its initial value: two Register
// cycles over one open store must not share a backing array, or the
// copy-on-write discipline in app.go would be subverted from below.
func (s *todoStore) snapshot() ([]Todo, int) {
	if s == nil {
		return nil, 1
	}
	return append([]Todo(nil), s.initial...), s.nextID
}

// --- Write-throughs ------------------------------------------------------
// One per mutation helper in app.go, same granularity: the SQL mirrors the
// slice operation instead of rewriting the whole table, so cost tracks the
// user's action, not the list size. Failures are logged and swallowed — the
// in-memory list already updated and remains authoritative for this process;
// the worst case is this one action missing after the next launch.

func (s *todoStore) add(t Todo) {
	if s == nil {
		return
	}
	s.exec(`INSERT INTO todos (id, title, done) VALUES ($1, $2, $3)`, t.ID, t.Title, t.Done)
}

func (s *todoStore) setDone(id int, done bool) {
	if s == nil {
		return
	}
	s.exec(`UPDATE todos SET done = $1 WHERE id = $2`, done, id)
}

func (s *todoStore) remove(id int) {
	if s == nil {
		return
	}
	s.exec(`DELETE FROM todos WHERE id = $1`, id)
}

func (s *todoStore) clearDone() {
	if s == nil {
		return
	}
	s.exec(`DELETE FROM todos WHERE done`)
}

func (s *todoStore) exec(query string, args ...any) {
	if _, err := s.db.Exec(query, args...); err != nil {
		log.Printf("todoapp: %s: %v", query, err)
	}
}

// asInt widens whatever integer type the engine hands back. bytdb returns
// int64 for int columns today; the switch keeps the load path working if a
// future version narrows that.
func asInt(v any) int {
	switch n := v.(type) {
	case int64:
		return int(n)
	case int:
		return n
	case int32:
		return int(n)
	default:
		return 0
	}
}
