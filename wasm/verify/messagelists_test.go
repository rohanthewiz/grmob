package main

import (
	"sort"
	"strings"
)

// How a failure message in this package names a set of things.
//
// # The convention
//
// Every check here reports what it found by listing it, and every one of those
// lists is SORTED before it is joined. Not for tidiness: a finding is read by
// diffing it against the last run's, and a list that arrives in map order or
// directory order is different text on every run for the same fact. Two runs
// of a broken check should produce the same paragraph, and the only thing that
// makes them is an order that does not depend on how the walk happened to
// reach things.
//
// # Where the sort goes, which is the part worth stating
//
// After the format, never before. Sorting the elements and then formatting
// them orders a list by whatever field the underlying struct compares on —
// which for a walk row is the order the scan found the files in, not anything
// a reader can predict. Sorting the rendered STRINGS orders the list by the
// line a person actually sees, which is the only order a paragraph can be
// diffed in.
//
// # Why these two are here and the named renderers are not
//
// These are the shape with nothing domain-specific in them: a map's keys, and
// a slice put through a caller's Sprintf. The named renderers — recordList,
// walkList, repositoryWalkList, enumerationList — are one call to listOf each
// and live beside the type they render, because what they encode is which
// fields of that type a reader needs in order to go and find the thing. That
// is a fact about the type, not about messages.
//
// What this file is, then, is the place the convention is written down. It had
// been four copies of three lines, then one helper with the rule in its doc
// and a second helper in another file doing the same thing to maps, which is a
// convention that exists and is stated nowhere a reader would look for it.

// keysOf is a map's keys, sorted, so a failure message reads the same on every
// run and can be diffed.
func keysOf[V any](m map[string]V) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

// listOf is a slice rendered for a message: each element formatted, the
// results sorted, joined with ", ".
//
// The sort is of the formatted strings — see the convention above, which is
// the reason this takes a format function rather than returning the strings
// for a caller to join.
func listOf[T any](in []T, format func(T) string) string {
	out := make([]string, 0, len(in))
	for _, item := range in {
		out = append(out, format(item))
	}
	sort.Strings(out)
	return strings.Join(out, ", ")
}
