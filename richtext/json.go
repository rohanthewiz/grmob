package richtext

import (
	"encoding/json"
	"strings"
)

// The wire encoding: short keys, GridRun style.
//
//	{"b":[{"k":"h1","r":[{"t":"Title"}]},
//	      {"k":"p", "r":[{"t":"Hello "},{"t":"world","b":1,"l":"https://x"}]}]}
//
// # Why short keys
//
// The same argument core.GridRun's json tags make, one level up: this JSON is
// the *value of a controlled input*, so it crosses the bridge on every
// keystroke, in both directions. The key names are the part of a document that
// is not content, and a note of a few paragraphs would otherwise spend more
// bytes on the word "underline" than on its own text.
//
// # Why the marks are 1 and not true
//
// `"b":1` rather than `"b":true` for the same reason, and because the flags are
// read by three hand-written host serializers as well as by Go: a number is one
// truthiness test in JavaScript, Swift and Kotlin alike, where a JSON bool
// decodes to three different types with three different optional dances. Both
// are accepted on the way in — see markFlag — so a host that sends `true` is
// understood rather than silently unformatted.
//
// # Why the shadow types
//
// Doc, Block and Run are the types app code holds, and their field names are
// the ones that read well at a call site. These are the same values with the
// wire's names on them. Encoding through a shadow rather than tagging the real
// types keeps the two decisions apart: the exported struct is an API and the
// wire is a format, and they are allowed to change for different reasons.

type wireDoc struct {
	Blocks []wireBlock `json:"b,omitempty"`
}

type wireBlock struct {
	// Kind is omitted for a paragraph, which is the commonest block by a wide
	// margin and the value an absent key decodes to anyway (normalizeKind maps
	// "" onto Paragraph). So a document of plain prose carries no kind keys at
	// all.
	Kind BlockKind `json:"k,omitempty"`
	Runs []wireRun `json:"r,omitempty"`
}

type wireRun struct {
	Text string `json:"t"`
	// The five marks, each written only when set. markFlag's underlying type is
	// bool, so `,omitempty` drops the false ones — which is exactly "this mark
	// is off" — while its MarshalJSON writes the 1 the hosts read.
	Bold      markFlag `json:"b,omitempty"`
	Italic    markFlag `json:"i,omitempty"`
	Underline markFlag `json:"u,omitempty"`
	Strike    markFlag `json:"s,omitempty"`
	Code      markFlag `json:"c,omitempty"`
	Link      string   `json:"l,omitempty"`
}

// markFlag is one mark on the wire: written as 1, read from anything a host
// might plausibly have sent.
//
// The asymmetry is deliberate and is the whole reason this type exists rather
// than a plain bool. Writing 1 keeps the truthiness test one operation in
// JavaScript, Swift and Kotlin alike. Reading *anything truthy* means a host
// that sends `true` — because its JSON library writes bools for bools, which
// two of the three do by default — is understood rather than silently
// unformatted, which is the failure mode a strict int would have: encoding/json
// refuses `true` into an int, and the refusal would fail the whole document
// rather than one mark.
//
// It never returns an error, for that same reason. A value this does not
// recognize is a mark that is off, not a document that cannot be read.
type markFlag bool

func (m markFlag) MarshalJSON() ([]byte, error) {
	if m {
		return []byte("1"), nil
	}
	return []byte("0"), nil
}

func (m *markFlag) UnmarshalJSON(data []byte) error {
	switch strings.TrimSpace(string(data)) {
	case "", "null", "0", "false", `""`, `"0"`, `"false"`:
		*m = false
	default:
		*m = true
	}
	return nil
}

// MarshalJSON writes the wire form. See the file doc for the shape.
func (d Doc) MarshalJSON() ([]byte, error) {
	out := wireDoc{Blocks: make([]wireBlock, len(d.Blocks))}
	for i, block := range d.Blocks {
		out.Blocks[i] = wireBlock{Kind: block.Kind, Runs: make([]wireRun, len(block.Runs))}
		// Paragraph is the default on the way back, so writing it would put the
		// commonest key on the wire for nothing.
		if block.Kind == Paragraph {
			out.Blocks[i].Kind = ""
		}
		for j, run := range block.Runs {
			out.Blocks[i].Runs[j] = wireRun{
				Text:      run.Text,
				Bold:      markFlag(run.Bold),
				Italic:    markFlag(run.Italic),
				Underline: markFlag(run.Underline),
				Strike:    markFlag(run.Strike),
				Code:      markFlag(run.Code),
				Link:      run.Link,
			}
		}
		if len(block.Runs) == 0 {
			// A blank line. The empty slice would marshal as `[]` and the nil as
			// nothing at all; nothing at all is right, and it is what `r` being
			// absent decodes back to.
			out.Blocks[i].Runs = nil
		}
	}
	if len(d.Blocks) == 0 {
		out.Blocks = nil
	}
	return json.Marshal(out)
}

// UnmarshalJSON reads the wire form, normalizing as it goes.
//
// Everything that arrives here came from outside Go — a host's serializer, a
// database row written by an older version of this package — so nothing is
// trusted to be in the vocabulary. An unknown block kind becomes a paragraph
// (see normalizeKind); a mark is read as truthy rather than as a particular
// type (markFlag); a run with no text is dropped, because an empty run is
// invisible on every target and costs a span on all four.
//
// What is deliberately *not* dropped is an empty block: that is a blank line,
// and a blank line is something a writer typed.
func (d *Doc) UnmarshalJSON(data []byte) error {
	var in wireDoc
	if err := json.Unmarshal(data, &in); err != nil {
		return err
	}
	out := Doc{}
	if len(in.Blocks) > 0 {
		out.Blocks = make([]Block, 0, len(in.Blocks))
	}
	for _, block := range in.Blocks {
		next := Block{Kind: normalizeKind(block.Kind)}
		for _, run := range block.Runs {
			if run.Text == "" {
				continue
			}
			next.Runs = append(next.Runs, Run{
				Text:      run.Text,
				Bold:      bool(run.Bold),
				Italic:    bool(run.Italic),
				Underline: bool(run.Underline),
				Strike:    bool(run.Strike),
				Code:      bool(run.Code),
				Link:      run.Link,
			})
		}
		out.Blocks = append(out.Blocks, next)
	}
	*d = out
	return nil
}

// JSON is Marshal without the error, for the call sites that cannot fail and
// should not have to say so.
//
// MarshalJSON here can only fail if encoding/json cannot encode a struct of
// strings and ints, which it can. The node builder needs a string and a render
// pass has nowhere to put an error, so this is where that is stated once
// instead of at every call site with a dropped `_`.
func (d Doc) JSON() string {
	data, err := json.Marshal(d)
	if err != nil {
		// Unreachable: the wire types are strings and ints. An empty document is
		// the safest thing to send if it ever becomes reachable — it is what an
		// editor with nothing in it holds, rather than a truncated document the
		// host would then echo back as the user's own.
		return `{}`
	}
	return string(data)
}

// ParseJSON reads a document from the wire form.
//
// A forwarder rather than a call to json.Unmarshal at each site, because the
// three places that do it (the node's onChange, a database read, a test) all
// want the same thing: a Doc and an error, with no pointer dance.
func ParseJSON(data string) (Doc, error) {
	var doc Doc
	err := json.Unmarshal([]byte(data), &doc)
	return doc, err
}
