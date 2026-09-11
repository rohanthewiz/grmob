package main

import "strings"

// A token in backquotes is quoted, not claimed.
//
// # What this is about
//
// Two rules in this package hold a token in prose to something outside the
// prose: a Test-shaped name is held to being a test this repository has, and
// — in the form that was declined — a wall clock is held to living in a
// timings record. Both broke on the same sentence shape, which is a sentence
// ABOUT a token rather than one pointing at it:
//
//	was called `TestTheShapesThisRepositoryKeepsTwoCopiesOfAreInStep`
//	this table said `30.14–30.42s` while the record said something else
//
// Both are written above the way this file says to write them, which is also
// why neither is a finding against the rule stated here — a file that had to
// be exempted from its own convention would be the argument against it.
//
// Neither is a stale reference. Both name a thing that is deliberately gone,
// because the sentence's whole subject is that it is gone — a rename with no
// dead name in it explains nothing, and "this figure was wrong" with the
// wrong figure removed is not a sentence. A checker that cannot tell these
// from a live reference either fails on them or carries a table of them, and
// prosenames_test.go carried exactly that table for four sessions.
//
// # Why backquotes, rather than a marker invented for this
//
// Because this repository already means it. coresAttribution states the rule
// for the terms it names — those go in backquotes, and a term named in prose
// is neither claimed nor checked — and trackedGoFileFigure restates it for a
// figure: a count written in the fixed form is a reading of the tree, and a
// count written any other way is English. What was missing was not a
// convention. It was that the convention had never been stated for the two
// shapes a rule reads, and nothing read it.
//
// It was also already being applied by hand, which is the strongest evidence
// that it is the right spelling. Measured over every Go comment and string
// constant outside ai_docs:
//
//	91 duration figures written as a range, 4 of them backquoted
//	3 of those 4  quotations of a past reading — the two main.go carried
//	              against this record, and main_test.go's wholeRun
//
// Three of the four figures anybody thought to backquote are exactly the
// class this names. Nobody was told to do that.
//
// # What the escape costs, which is the number that decided it
//
// An escape is only worth having if it is narrow, and "backquoted" is a
// typographic habit as much as a statement — the risk is somebody quoting a
// live name out of ordinary emphasis and silently opting it out. So it was
// counted before it was written:
//
//	356 Test-shaped mentions in Go prose
//	13 of them backquoted
//	12 of the 13  in prosenames_test.go, whose mentions are already exempt
//	              by name — nine of them the dead names that are the rule's
//	              own evidence
//	1 of the 13   `func TestMain(` inside a tutorial code block, which is a
//	              sample somebody is meant to write rather than a citation
//
// Outside the file that was already exempt, this escape lets through exactly
// one mention, and that one is not a citation of anything. It costs no
// coverage this repository currently has.
//
// The bar for using it is the bar the table it replaces already stated: the
// sentence has to be ABOUT the token. A sentence that merely uses a dead name
// to point at a live test is the finding these rules exist for, and
// backquoting it is how you hide the finding rather than how you record the
// history.
//
// # What it does not buy
//
// It does not make the wall-clock rule writable, and that is worth saying
// here because the item asking for this convention predicted it would.
//
// That rule's residue was five sentences that were RIGHT. All five are now
// resolved, but only two of them by this: the pair in
// internal/themehistory/main.go, whose sentence is unchanged in substance and
// simply backquoted. The other three were fixed by naming a record field
// instead of restating it — the fix this repository already had — and what
// the convention contributed there was that the history those lines carried
// survived the fix instead of being deleted with the copy.
//
// The rule is still declined, on a reason this cleared the way to see:
// "lives in a record" is not a span anything can define, because readings
// legitimately live in a `…TimingsTakenOn` declaration, in a measurement
// table in the prose beside one, and in the `costs:` field of a structure
// that is a record in everything but name. See ai_docs/plans/non_goals.md,
// where that measurement is re-run against this convention rather than
// against the tree that had none.

// quotedRuns is the byte spans of text this prose writes in backquotes.
//
// Pairs, taken left to right: the first backquote opens, the second closes,
// the third opens. A trailing unpaired backquote therefore opens nothing and
// the text after it is not quoted, which is the safe direction for a rule
// that uses this as an ESCAPE — a comment with a stray backquote in it gets
// checked rather than silently skipped.
//
// Spans are half-open over the text BETWEEN the backquotes, so the delimiters
// themselves are not inside any span.
func quotedRuns(text string) [][2]int {
	var runs [][2]int
	for i := 0; ; {
		open := strings.IndexByte(text[i:], '`')
		if open < 0 {
			return runs
		}
		open += i
		closes := strings.IndexByte(text[open+1:], '`')
		if closes < 0 {
			// Unpaired, so it opens nothing. See the doc.
			return runs
		}
		closes += open + 1
		runs = append(runs, [2]int{open + 1, closes})
		i = closes + 1
	}
}

// quotedAt says whether the token starting at this byte offset stands inside
// one of those runs.
//
// Linear over the runs rather than a binary search, because a comment group
// holds a handful of them and the caller is already running a regexp over the
// same text.
func quotedAt(runs [][2]int, start int) bool {
	for _, r := range runs {
		if start >= r[0] && start < r[1] {
			return true
		}
	}
	return false
}

// unwrapRawLiteral strips the delimiters off a Go raw string literal.
//
// This matters only because of quotedRuns. A raw literal is spelled with the
// same byte the convention uses, so scanning `lit.Value` straight would make
// the literal's own delimiters a pair and read EVERY token inside it as
// quoted — a silent exemption for every mention in a raw string, which is
// where this repository keeps its code samples. An interpreted literal is
// delimited with `"` and passes through untouched.
//
// The offsets of everything inside shift by one, which is why the caller
// unwraps before it records line numbers rather than after.
//
// # It is load-bearing for nothing today, and that was measured
//
// Removing it changes no count: 387 mentions either way. The one Test-shaped
// name this repository keeps in a raw literal is the `func TestMain(` in
// examples/tutorial/chapter8.go, and the sample skip excludes that one on
// its own.
//
// It stays because of what its absence would be, which is not a missed
// finding but an INVISIBLE one: a raw literal that named a test would be
// skipped entirely, with nothing in any message saying a literal had been
// read as one long quotation. That is the same reason
// checkProseNamesResolve keeps a Fatalf for a declaration list that comes
// back empty — a guard that has never fired, kept because the failure it
// guards against is silent rather than loud. A rule this repository declines
// is one that produces no findings; a mechanism that stops a kept rule
// lying is a different thing and is not held to the same count.
func unwrapRawLiteral(s string) string {
	if len(s) >= 2 && s[0] == '`' && s[len(s)-1] == '`' {
		return s[1 : len(s)-1]
	}
	return s
}
