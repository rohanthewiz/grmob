package richtext

import (
	"strconv"
	"strings"
)

// Markdown import and export, over exactly the subset this model holds.
//
// # What this is for, and what it is not
//
// It is the door: a note written here can leave as Markdown and come back, and
// a test can spell a document as something a person can read. It is *not* the
// wire — see the package doc — and it is not a CommonMark implementation. A
// document this package did not write is read on a best effort and whatever it
// does not recognize becomes paragraph text, which is the same degradation
// normalizeKind makes one level up: the words are never at risk, only their
// presentation.
//
// # The one place the subset leaves CommonMark
//
// Underline. CommonMark has no syntax for it — `_x_` is emphasis, not underline
// — so it is written as an inline `<u>` tag, which CommonMark permits as raw
// HTML and which every renderer draws. The alternative was to drop the mark on
// export, which would make the round trip lossy for a mark the toolbar offers.
//
// # The two things Markdown cannot hold, stated rather than hidden
//
// A blank line is Markdown's block separator and nothing else, so an *empty
// paragraph* — which is a real block here, and how a writer asks for space —
// has no representation. Markdown() drops empty blocks and FromMarkdown()
// treats every run of blank lines as one separator. The JSON keeps them, and
// the JSON is the wire and the storage; this is a loss at the export door only.
//
// A code span's content is literal by definition, so the other five marks are
// not expressible on one. Markdown() drops them; again the JSON keeps them.
//
// Both are pinned by tests, so neither can quietly become something else.
//
// # Escaping
//
// The emitter escapes the characters that would otherwise re-enter the document
// as syntax, and the parser un-escapes them. That is what makes the round trip a
// property rather than a hope: a paragraph that happens to contain `**` has to
// come back as a paragraph containing `**`, not as a bold run with nothing in
// it. TestMarkdownRoundTripsTheSubset is the check.

// Markdown renders the document as CommonMark (plus one `<u>`; see above).
//
// Blocks are separated by a blank line, which is what makes two consecutive
// paragraphs two paragraphs on the way back in. List items are the exception:
// consecutive bullets are written on consecutive lines, because a blank line
// between them makes a "loose" list in CommonMark and, more to the point, makes
// them read as separate lists.
func (d Doc) Markdown() string {
	var out strings.Builder
	previous := BlockKind("")
	for i, block := range d.Blocks {
		// An empty block has no Markdown: a blank line is the separator, so an
		// empty paragraph would be written as a second separator and read back
		// as nothing. Dropped deliberately rather than emitted and lost — see
		// the file doc. An empty *code* block is kept, because an empty fence is
		// a fence and reads back as one.
		if len(block.Runs) == 0 && block.Kind != BlockCode {
			continue
		}
		if out.Len() > 0 {
			out.WriteByte('\n')
			// A run of list items is one list, so no blank line inside it.
			if !(block.Kind.IsList() && previous == block.Kind) {
				out.WriteByte('\n')
			}
		}
		out.WriteString(block.markdown(d.numberOf(i)))
		previous = block.Kind
	}
	return out.String()
}

// numberOf is the ordinal a Numbered block is written with: its position in the
// unbroken run of Numbered blocks it belongs to.
//
// Counted rather than always written as "1." — which CommonMark would also
// accept and renumber — because the Markdown is something a person reads. A
// list that reads 1. 2. 3. in the file is worth the backward walk, and the walk
// is over consecutive siblings, so a document of N blocks costs O(N) in total
// rather than O(N²): each block walks back only over its own run.
func (d Doc) numberOf(i int) int {
	n := 1
	for j := i - 1; j >= 0 && d.Blocks[j].Kind == Numbered; j-- {
		n++
	}
	return n
}

func (b Block) markdown(number int) string {
	if b.Kind == BlockCode {
		// Verbatim, and fenced rather than indented: an indented code block
		// cannot hold a line that starts with less indentation than the marker,
		// and a fence can hold anything that is not the fence.
		return "```\n" + b.Text() + "\n```"
	}
	body := inlineMarkdown(b.Runs)
	switch b.Kind {
	case Heading1:
		return "# " + body
	case Heading2:
		return "## " + body
	case Heading3:
		return "### " + body
	case Bullet:
		return "- " + body
	case Numbered:
		return strconv.Itoa(number) + ". " + body
	case Quote:
		return "> " + body
	}
	// A paragraph, and the one place a *leading* character can be mistaken for
	// a block marker. Escaping it here rather than in escapeInline keeps a `#`
	// in the middle of a sentence unescaped, which is what a reader wants.
	return escapeLeadingMarker(body)
}

// inlineMarkdown writes one block's runs.
//
// The marks nest in a fixed order — link outermost, then bold, italic, strike,
// underline, code innermost — so that the emitter and the parser agree about
// what encloses what without either having to think about it. Any order would
// do; having *an* order is what matters, and this one puts the mark that
// suppresses the others (code) where it belongs.
func inlineMarkdown(runs []Run) string {
	var out strings.Builder
	for _, run := range runs {
		out.WriteString(run.markdown())
	}
	return out.String()
}

func (r Run) markdown() string {
	// Inside inline code nothing else is expressible in Markdown — the whole
	// point of a code span is that its content is literal — so a code run
	// carries its text unescaped and drops the other marks on export. That is
	// lossy in exactly one direction and only for Markdown; the JSON keeps all
	// six.
	if r.Code {
		text := "`" + r.Text + "`"
		if r.Link != "" {
			return "[" + text + "](" + r.Link + ")"
		}
		return text
	}
	text := escapeInline(r.Text)
	if r.Underline {
		text = "<u>" + text + "</u>"
	}
	if r.Strike {
		text = "~~" + text + "~~"
	}
	if r.Italic {
		text = "*" + text + "*"
	}
	if r.Bold {
		text = "**" + text + "**"
	}
	if r.Link != "" {
		text = "[" + text + "](" + r.Link + ")"
	}
	return text
}

// inlineSpecials are the characters that would otherwise re-enter the document
// as syntax. The backslash is first because escapeInline replaces in order and
// escaping it afterwards would double every escape it had just written.
var inlineSpecials = []string{`\`, "*", "_", "`", "[", "]", "~", "<", ">"}

func escapeInline(s string) string {
	for _, special := range inlineSpecials {
		s = strings.ReplaceAll(s, special, `\`+special)
	}
	return s
}

// escapeLeadingMarker guards the one position where a character that is
// harmless inline becomes a block marker: the start of a paragraph.
//
// `#`, `-`, `+` and `N.` at column zero are a heading and two kinds of list
// item. Escaped with a backslash, which CommonMark defines as making the next
// character literal, and which unescapeInline takes back off.
func escapeLeadingMarker(body string) string {
	if body == "" {
		return body
	}
	switch body[0] {
	case '#', '-', '+':
		return `\` + body
	}
	// An ordered-list marker: digits, then a dot.
	digits := 0
	for digits < len(body) && body[digits] >= '0' && body[digits] <= '9' {
		digits++
	}
	if digits > 0 && digits < len(body) && body[digits] == '.' {
		return body[:digits] + `\` + body[digits:]
	}
	return body
}

// FromMarkdown reads the subset back.
//
// The error is always nil today and the signature keeps it anyway, because the
// alternative is worse than an unused return: this is the *import* door, the
// thing most likely to grow a real refusal (a size limit, a fence that never
// closes), and a signature change later would break every caller. A parse that
// cannot make sense of a line makes it a paragraph, which is what every
// forgiving Markdown reader does.
func FromMarkdown(src string) (Doc, error) {
	lines := strings.Split(strings.ReplaceAll(src, "\r\n", "\n"), "\n")
	var doc Doc
	for i := 0; i < len(lines); i++ {
		line := lines[i]

		// A fenced code block, taken whole: everything up to the closing fence
		// is one block's text, unparsed. An unclosed fence runs to the end of
		// the document, which is what every reader does with one.
		if strings.HasPrefix(strings.TrimSpace(line), "```") {
			var body []string
			i++
			for i < len(lines) && !strings.HasPrefix(strings.TrimSpace(lines[i]), "```") {
				body = append(body, lines[i])
				i++
			}
			doc.Blocks = append(doc.Blocks, Block{
				Kind: BlockCode,
				Runs: []Run{{Text: strings.Join(body, "\n")}},
			})
			continue
		}

		if strings.TrimSpace(line) == "" {
			// A blank line is a separator and nothing else. Markdown has no
			// empty paragraph — see the file doc — so a run of them is one
			// separator rather than a block apiece, which is what every reader
			// does with a run of them.
			continue
		}

		kind, body := splitBlockMarker(line)
		doc.Blocks = append(doc.Blocks, Block{Kind: kind, Runs: parseInline(body, Run{})})
	}
	return doc, nil
}

// splitBlockMarker reads a line's leading block marker, returning the kind and
// the text after it.
//
// Leading white space is tolerated and dropped: a Markdown file written by hand
// often indents its list items, and this model has no nesting for the indent to
// mean anything else.
func splitBlockMarker(line string) (BlockKind, string) {
	trimmed := strings.TrimLeft(line, " \t")
	for prefix, kind := range map[string]BlockKind{
		"### ": Heading3,
		"## ":  Heading2,
		"# ":   Heading1,
		"- ":   Bullet,
		"+ ":   Bullet,
		"* ":   Bullet,
		"> ":   Quote,
	} {
		if strings.HasPrefix(trimmed, prefix) {
			return kind, trimmed[len(prefix):]
		}
	}
	// An ordered item: digits, a dot, a space.
	digits := 0
	for digits < len(trimmed) && trimmed[digits] >= '0' && trimmed[digits] <= '9' {
		digits++
	}
	if digits > 0 && digits+1 < len(trimmed) && trimmed[digits] == '.' && trimmed[digits+1] == ' ' {
		return Numbered, trimmed[digits+2:]
	}
	// Not a marker, so the line is a paragraph — and its own leading white space
	// is kept, because nothing has claimed it.
	return Paragraph, line
}

// parseInline turns one block's text into runs, carrying the marks of whatever
// it is nested inside.
//
// Recursive rather than a stack machine: each mark's opener is matched against
// its closer, the text between them is parsed again with that mark added, and
// the scan continues after the closer. That handles nesting for free and makes
// the emitter's fixed order irrelevant to the parser, which is why an
// `**a *b* c**` written by hand reads correctly even though this package would
// never write it that way.
//
// A marker with no closer is literal text, which is the forgiving reading and
// the only one that keeps a document of prose from turning into formatting the
// author did not ask for.
func parseInline(src string, inherited Run) []Run {
	var out []Run
	var literal strings.Builder

	flush := func() {
		if literal.Len() == 0 {
			return
		}
		run := inherited.marks()
		run.Text = literal.String()
		out = append(out, run)
		literal.Reset()
	}

	for i := 0; i < len(src); {
		rest := src[i:]

		// A backslash escape: the next byte is literal, whatever it is. First,
		// so that an escaped marker is never seen as a marker.
		if rest[0] == '\\' && len(rest) > 1 {
			literal.WriteByte(rest[1])
			i += 2
			continue
		}

		// A link, which is the only construct whose closer is two characters
		// apart from its opener. Checked before the emphasis markers because its
		// label may contain them.
		if rest[0] == '[' {
			if label, url, width, ok := splitLink(rest); ok {
				flush()
				nested := inherited.marks()
				nested.Link = url
				out = append(out, parseInline(label, nested)...)
				i += width
				continue
			}
		}

		// Inline code, whose content is literal — so it is not parsed again.
		if rest[0] == '`' {
			if end := strings.IndexByte(rest[1:], '`'); end >= 0 {
				flush()
				run := inherited.marks()
				run.Code = true
				run.Text = rest[1 : 1+end]
				out = append(out, run)
				i += end + 2
				continue
			}
		}

		// The paired markers, longest first: `**` has to be tested before `*` or
		// every bold run would read as an empty italic one.
		matched := false
		for _, marker := range []struct {
			open, close string
			apply       func(*Run)
		}{
			{"**", "**", func(r *Run) { r.Bold = true }},
			{"~~", "~~", func(r *Run) { r.Strike = true }},
			{"<u>", "</u>", func(r *Run) { r.Underline = true }},
			{"*", "*", func(r *Run) { r.Italic = true }},
			{"_", "_", func(r *Run) { r.Italic = true }},
		} {
			if !strings.HasPrefix(rest, marker.open) {
				continue
			}
			body := rest[len(marker.open):]
			end := indexUnescaped(body, marker.close)
			if end < 0 {
				continue
			}
			flush()
			nested := inherited.marks()
			marker.apply(&nested)
			out = append(out, parseInline(body[:end], nested)...)
			i += len(marker.open) + end + len(marker.close)
			matched = true
			break
		}
		if matched {
			continue
		}

		literal.WriteByte(rest[0])
		i++
	}
	flush()
	return out
}

// splitLink reads `[label](url)` starting at src[0] == '['.
//
// The label is scanned with nesting counted, so a link whose label contains
// brackets — `[[a] b](u)` — closes at the right one. The URL is taken to the
// first `)`, which is the limitation CommonMark solves with angle brackets and
// this package does not: a URL containing an unescaped `)` is truncated. Rare
// enough, and stated rather than hidden.
func splitLink(src string) (label, url string, width int, ok bool) {
	depth := 0
	for i := 0; i < len(src); i++ {
		switch src[i] {
		case '\\':
			i++
		case '[':
			depth++
		case ']':
			depth--
			if depth > 0 {
				continue
			}
			if i+1 >= len(src) || src[i+1] != '(' {
				return "", "", 0, false
			}
			close := strings.IndexByte(src[i+2:], ')')
			if close < 0 {
				return "", "", 0, false
			}
			return src[1:i], src[i+2 : i+2+close], i + 3 + close, true
		}
	}
	return "", "", 0, false
}

// indexUnescaped is strings.Index that skips a match preceded by a backslash,
// so `a\*b*c` closes its italic at the second star and not the first.
func indexUnescaped(s, sub string) int {
	for at := 0; at < len(s); {
		found := strings.Index(s[at:], sub)
		if found < 0 {
			return -1
		}
		abs := at + found
		if abs == 0 || s[abs-1] != '\\' {
			return abs
		}
		at = abs + 1
	}
	return -1
}
