package highlight

import (
	"fmt"
	"testing"
)

// A repeat highlighting is answered from the cache with the very same rows,
// which is both the saving (no re-tokenising on a pass that did not touch the
// buffer) and what lets the diff compare the rows by identity. A change of
// source or of scheme is a different entry, and the oldest entries age out.
func TestRowsAreCachedBySourceAndScheme(t *testing.T) {
	src := "package main\n\nfunc main() { println(\"hi\") }\n"
	a := Go().Rows(src, Darcula)
	b := Go().Rows(src, Darcula)
	if len(a) == 0 || &a[0] != &b[0] {
		t.Fatal("the same source and scheme were tokenised twice")
	}
	if c := Go().Rows(src+"// more\n", Darcula); &c[0] == &a[0] {
		t.Fatal("an edited source was answered from the old entry")
	}
	if d := Go().Rows(src, Scheme{}); &d[0] == &a[0] {
		t.Fatal("another scheme was answered from the first scheme's entry")
	}
	if e := JSON().Rows(src, Darcula); &e[0] == &a[0] {
		t.Fatal("another lexer was answered from Go's entry")
	}

	for i := range rowCacheSize + 1 {
		Go().Rows(fmt.Sprintf("var x = %d\n", i), Darcula)
	}
	if f := Go().Rows(src, Darcula); &f[0] == &a[0] {
		t.Fatal("the cache did not evict an entry older than its size")
	}
	rowCache.mu.Lock()
	n := rowCache.order.Len()
	rowCache.mu.Unlock()
	if n > rowCacheSize {
		t.Fatalf("the cache holds %d entries, more than its size %d", n, rowCacheSize)
	}
}

// What a pass that does not touch a 500-line buffer pays for its highlighting:
// a cache hit. The "cold" case is what every pass paid before the cache (plan
// P8a measured 0.19 ms and 465 KB there).
//
//	go test -run '^$' -bench Rows500 -benchmem ./highlight
func BenchmarkRows500Lines(b *testing.B) {
	var src string
	for i := range 500 {
		src += fmt.Sprintf("\tx%d := fmt.Sprintf(\"%%d\", %d) // line %d\n", i, i, i)
	}
	src = "package main\n\nfunc f() {\n" + src + "}\n"
	b.Run("cold", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			Go().Rows(src, Darcula) // the cache is emptied below, so every call computes
			rowCache.mu.Lock()
			rowCache.order.Init()
			clear(rowCache.index)
			rowCache.mu.Unlock()
		}
	})
	b.Run("warm", func(b *testing.B) {
		b.ReportAllocs()
		Go().Rows(src, Darcula)
		for b.Loop() {
			Go().Rows(src, Darcula)
		}
	})
}
