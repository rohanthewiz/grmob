package apidoc

import "strings"

// Anchor ids on a generated page are not ours to choose: gkdocs renders
// markdown with goldmark and enables parser.WithAutoHeadingID(), which computes
// an id from the heading text, and it does not enable goldmark's attribute
// extension, so the `## Heading {#custom-id}` escape hatch is unavailable. Every
// link into a page — the per-page index, the cross-package doc links — therefore
// has to predict the id the renderer will compute.
//
// This is goldmark's parser.ids.Generate, reproduced:
//
//	trim leading and trailing space
//	then, byte by byte:
//	    ASCII letter or digit  ->  kept, lowercased
//	    space, '-' or '_'      ->  '-'
//	    any other ASCII byte   ->  dropped   (backtick, '*', '(', ')', '.', ',')
//	    any multi-byte rune    ->  dropped
//	an empty result           ->  "heading"
//
// The one part deliberately *not* reproduced is goldmark's duplicate handling:
// the second heading on a page that slugs to an id already taken silently
// becomes "<id>-1". Silently is the problem — the page renders, the heading
// looks right, and every link aimed at it now lands on the wrong section. So
// instead of predicting the counter, the generator asserts that every heading a
// link aims at kept the id its own text produces (see
// TestLinkedAnchorsAreNotStolen), which turns that class of breakage into a
// failing test.
func slug(s string) string {
	s = strings.TrimSpace(s)

	var b strings.Builder
	b.Grow(len(s))
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch {
		case c >= 'a' && c <= 'z', c >= '0' && c <= '9':
			b.WriteByte(c)
		case c >= 'A' && c <= 'Z':
			b.WriteByte(c + ('a' - 'A'))
		case c == ' ' || c == '\t' || c == '\n' || c == '\v' || c == '\f' || c == '\r',
			c == '-', c == '_':
			b.WriteByte('-')
		}
		// Everything else, including every byte of a multi-byte rune (all of
		// which have the high bit set and so fail all the cases above), is
		// dropped — the same as goldmark, which skips non-ASCII runes whole.
	}

	if b.Len() == 0 {
		return "heading"
	}
	return b.String()
}

// symAnchor is the id of the section a symbol gets its own heading in, built
// from the parts rather than from the rendered heading text so that a doc link
// — which arrives as (receiver, name) and never as a heading string — can be
// resolved without reconstructing the heading.
//
// It must agree with the headings symbolHeading writes, and the two agree
// because the heading is the only place the prefix words "type" and "func"
// appear and both are slug-stable:
//
//	kind   heading                    anchor
//	type   ### type Node              type-node
//	func   ### func Text              func-text
//	method #### func (*Node) Clone    func-node-clone
//
// A pointer receiver and a value receiver land on the same anchor because '*'
// is dropped, which is correct: a method set has one method of a given name, so
// the two forms can never both exist to collide.
func symAnchor(kind, recv, name string) string {
	if recv != "" {
		return "func-" + slug(recv) + "-" + slug(name)
	}
	return kind + "-" + slug(name)
}
