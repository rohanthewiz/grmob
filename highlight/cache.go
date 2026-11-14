package highlight

import (
	"container/list"
	"sync"

	"github.com/rohanthewiz/grmob/core"
)

// A small LRU of highlighted buffers, shared by the built-in highlighters.
//
// # Why a highlighter needs a cache
//
// comps.CodeEditor highlights its Value in Render, and Render runs on every
// pass of the screen, not only when the buffer changes: a tap anywhere else
// re-tokenised the whole buffer. That was 0.19 ms and 465 KB at 500 lines,
// and 0.92 ms and 2.7 MB at 1,700 (plan P8a, N-139). The tutorial's code
// blocks re-highlighted static source the same way, about a fifth of a
// lesson's build CPU (P8d). The input is a pure function of (lexer, scheme,
// source), so the answer can be kept.
//
// # The second payoff
//
// A hit returns the *same* []core.GridRow as the last pass. reconcile compares
// props with reflect.DeepEqual, which answers two slices sharing a backing
// array as equal without walking them, so an unchanged editor's rows cost the
// diff one comparison instead of every run.
//
// That makes the returned rows shared: a caller must treat them as read-only,
// which is already the contract for anything placed on a core.Node (nodes are
// frozen after render).
//
// # Size
//
// 16 entries: a screen rarely shows more than a handful of code surfaces, and
// a typing user turns each keystroke into one new entry, so the old versions
// of a buffer age out within a few keys. Entries hold the source string and
// its rows, so 16 large buffers is a few MB at worst.
const rowCacheSize = 16

// rowKey identifies one highlighting. lexer is a comparable stand-in for the
// highlighter: a name for the hand-written lexers, the *langSpec pointer for
// a table-driven one. A caller's own Highlighter is never cached; its Rows is
// not known to be pure.
type rowKey struct {
	lexer  any
	scheme Scheme
	src    string
}

type rowEntry struct {
	key  rowKey
	rows []core.GridRow
}

var rowCache = struct {
	mu    sync.Mutex
	order *list.List // front is most recently used; values are *rowEntry
	index map[rowKey]*list.Element
}{order: list.New(), index: map[rowKey]*list.Element{}}

// cachedRows returns compute's rows for key, computing them at most once while
// the entry stays in the cache. compute runs outside the lock, so two passes
// highlighting different buffers do not wait on each other; two racing on the
// same key both compute it and the second store wins, which is harmless.
func cachedRows(key rowKey, compute func() []core.GridRow) []core.GridRow {
	rowCache.mu.Lock()
	if el, ok := rowCache.index[key]; ok {
		rowCache.order.MoveToFront(el)
		rows := el.Value.(*rowEntry).rows
		rowCache.mu.Unlock()
		return rows
	}
	rowCache.mu.Unlock()

	rows := compute()

	rowCache.mu.Lock()
	defer rowCache.mu.Unlock()
	if el, ok := rowCache.index[key]; ok {
		rowCache.order.MoveToFront(el)
		return el.Value.(*rowEntry).rows
	}
	rowCache.index[key] = rowCache.order.PushFront(&rowEntry{key: key, rows: rows})
	for rowCache.order.Len() > rowCacheSize {
		last := rowCache.order.Back()
		rowCache.order.Remove(last)
		delete(rowCache.index, last.Value.(*rowEntry).key)
	}
	return rows
}
