package core

import "github.com/rohanthewiz/grmob/richtext"

// The text-edit ledger: how a native host tells Go's echo of its own
// keystroke from Go's rewrite of the text, and what happens to keystrokes
// that were typed against text Go has since replaced.
//
// # The race this exists for
//
// A native text field is the host's while it is focused: every keystroke is
// drawn at once and sent to Go, and Go's answer arrives later, on another
// thread's schedule. Go usually answers with the same text (an echo), which
// the host must ignore so a late echo cannot snap the caret back. Sometimes
// Go answers with different text (a rewrite): TagInput clearing its draft
// once a separator commits a tag, a validator normalizing. The host used to
// tell the two apart by value, keeping a queue of every value it had sent and
// treating any upstream value not in the queue as a rewrite.
//
// That breaks when the keyboard outruns the round trip. Typed at machine
// speed into a TagInput holding "beta":
//
//	host sends   "beta,"  "beta,g"  "beta,ga"  "beta,gam"
//	Go takes     "beta,"  → commits the tag, rewrites the draft to ""
//	host gets    ""       (not in its queue, so a rewrite: text = "", queue cleared)
//	Go takes     "beta,g" "beta,ga" …  full values typed on the old text
//
// The rewrite lands after keystrokes the host had already sent. Go then runs
// those stale values through the handler as if they were new, and the echoes
// of them no longer match a queue the rewrite emptied. On the Android
// emulator ",gamma," committed "mma".
//
// # The protocol
//
// Two numbers, one in each direction:
//
//	host → Go   TriggerTextEdit(id, value, seq, epoch)
//	              seq    a number the host gives each edit, rising
//	              epoch  the last rewrite count the host has adopted
//	Go → host   props on the field node
//	              editSeq    the last edit Go has taken for this field
//	              editEpoch  how many times Go has rewritten this field
//
// Go keeps one ledger per text callback ID. It holds the value the host is
// believed to be showing. A render whose value differs from that value is a
// rewrite, so Go bumps the epoch and records the new value as the host's. An
// edit stamped with an older epoch was typed on text that has since been
// replaced, so it never reaches the handler.
//
// # editSeq names the last edit applied, not the last one received
//
// The host uses editSeq as the base of a rewrite: its own record of that
// edit is the text Go read before it rewrote, and the typing since is what
// it replays. So a dropped edit must not advance it. The first version did,
// and the emulator caught it within a minute. The rewrite of "eta," reached
// the host in a render that had already taken the dropped "eta,g", so the
// host rebased from "eta,g" and replayed only "a" of "ga". Nothing is lost
// by leaving dropped edits unacknowledged: the host clears every pending
// record when it adopts the rewrite that made them stale.
//
//	            host                                   Go
//	"beta,"   seq 5 epoch 0  ───────────────▶  applied; draft := ""
//	"beta,g"  seq 6 epoch 0  ───────────────▶  (queued behind seq 5)
//	          ◀──────── value "" editSeq 5 editEpoch 1   render ≠ "beta,", a rewrite
//	          epoch 1 > 0: rewrite based on seq 5 ("beta,")
//	          rebase "beta,g" onto "": "g"
//	"g"       seq 7 epoch 1  ───────────────▶
//	                                            seq 6: epoch 0 < 1, dropped
//	                                            seq 7: applied; draft := "g"
//	          ◀──────── value "g" editSeq 7 editEpoch 1   an echo
//
// The host decides echo versus rewrite by the epoch, not by the value.
// Rebasing is the host's job: it has the text as of the edit Go took (its
// own record of seq 5) and the text it shows now, and it replays the
// difference onto the rewrite. See GrMobTextField in each native renderer.
//
// # What does not carry the stamps
//
// Nothing does until the host has sent a TriggerTextEdit. That covers every
// web build (the wasm host calls Go synchronously, so there is no in-flight
// keystroke to race), static exports, and a native session before its first
// keystroke. Tests and exports that pinned a field's props see no change. A
// host that finds no stamps falls back to the value queue.
//
// # Every field, once the host is sequenced
//
// The first TriggerTextEdit says the host speaks the protocol. From then on
// every text field gets a ledger at its first render, holding the rendered
// value as the host's, and carries the stamps (editSeq 0 until an edit of its
// own is applied).
//
// It used to be each field's own first edit that made its ledger, and that
// left a hole the Android emulator found in comps.PINInput, which is six
// fields and one value. Typed at about 130ms a key:
//
//	cell 0  "3" sent, then "31" before focus moved   Go: code "31", focus → 2
//	cell 1  focused by the first render, still ""    Go's render: cell 1 = "1"
//	        "4" typed before that render arrives     Go: code "34"   the 1 is lost
//
// Go's change of cell 1 from "" to "1" was a rewrite, but cell 1 had no
// ledger to count it in, so the "4" typed on the old text was applied as if
// it were new. With a ledger from the first render, the render bumps cell 1's
// epoch, the "4" (epoch 0) is dropped, and the host replays it onto "1" as
// "14", which PINInput reads as a paste at cell 1: code "314".
//
// PINInput has since become one field (comps/pin_input.go), for the losses
// no ledger could fix. The rule stands for any field Go rewrites before its
// first edit; render/text_edit_test.go replays the sequence on a field of
// three boxes built for the purpose.
//
// A host tells stamps from no stamps by the presence of editEpoch, not by a
// nonzero editSeq: a field stamped from its render has editSeq 0 until its
// first edit, and its rewrites must still be read as rewrites.
//
// The ledger is keyed by callback ID, which is positional within a pass (see
// callbackRegistry.beginPass). A field that moves inherits whatever ledger
// its new position had. That is harmless: the next render sees a value that
// differs from the inherited one, bumps the epoch, and the host, which
// adopted the node's epoch when it mounted, takes Go's value.

// textEditLedger is Go's side of one text field's edit traffic.
type textEditLedger struct {
	// seq is the last edit Go has applied for this field. The host drops its
	// own record of every edit up to it, and takes that edit's text as the
	// base of a rewrite; see "editSeq names the last edit applied".
	seq int

	// epoch counts Go's rewrites of this field. An edit carrying a lower one
	// was typed before the host had seen the latest rewrite.
	epoch int

	// hostValue is the text the host is believed to be showing: the value of
	// the last edit applied, or the value of the last rewrite, whichever came
	// later. A dropped edit does not change it, because the host is about to
	// replace that text with the rewrite anyway.
	hostValue string
}

// acceptEdit records one edit and reports whether its handler should run.
// The registry's mutex guards the ledger map, as it guards the handler maps.
func (r *callbackRegistry) acceptEdit(id, value string, seq, epoch int) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.edits == nil {
		r.edits = make(map[string]*textEditLedger)
	}
	r.sequenced = true
	l := r.edits[id]
	if l == nil {
		// The first edit this field has sent, and it was never stamped: the
		// host was not sequenced when it was last rendered. Its epoch is
		// whatever the host adopted, which for such a field is 0, since the
		// host reads a missing editEpoch as 0.
		l = &textEditLedger{epoch: epoch}
		r.edits[id] = l
	}
	if epoch < l.epoch {
		return false
	}
	if seq > l.seq {
		l.seq = seq
	}
	l.hostValue = value
	return true
}

// stampEdit writes editSeq and editEpoch onto a text field's props, bumping
// the epoch first when the value being rendered is a rewrite. Before the host
// is sequenced it does nothing for a field with no ledger; after, it makes
// one. See the file comment for the rule.
//
// canon, when not nil, maps a value to the form Go itself would render it in,
// and is consulted only when the bytes differ. It exists for RichTextEditor,
// whose host sends its own JSON of the document: the same document spelled by
// org.json or JSONSerialization is not byte-for-byte what encoding/json writes
// (key order, escaped slashes, an empty block list), and without it every
// keystroke's echo would read as a rewrite. See textEditFields.
//
// It is idempotent within a pass: a second render of the same value finds the
// ledger's hostValue already equal to it.
func (r *callbackRegistry) stampEdit(id, value string, canon func(string) string, props map[string]any) {
	r.mu.Lock()
	defer r.mu.Unlock()
	l := r.edits[id]
	if l == nil {
		if !r.sequenced {
			return
		}
		// A field the host will mount (or already shows) with this value:
		// that is the text it believes it is showing, so this render is not a
		// rewrite. See "Every field, once the host is sequenced".
		l = &textEditLedger{hostValue: value}
		r.edits[id] = l
	}
	if value != l.hostValue {
		// The host's spelling, rendered Go's way. When it matches, the render is
		// an echo, and hostValue takes Go's bytes so the next pass compares
		// equal without parsing again.
		if canon == nil || canon(l.hostValue) != value {
			l.epoch++
		}
		l.hostValue = value
	}
	props["editSeq"] = l.seq
	props["editEpoch"] = l.epoch
}

// purgeEditsLocked drops the ledger of every text callback that did not survive
// the pass. Called from purge with the lock already held.
func (r *callbackRegistry) purgeEditsLocked(live map[string]func(string)) {
	for id := range r.edits {
		if _, ok := live[id]; !ok {
			delete(r.edits, id)
		}
	}
}

// textEditField says where a stamped node keeps the value the host edits,
// and how to compare the host's spelling of it with Go's.
type textEditField struct {
	// prop is the key holding the value: "value" for the text fields and the
	// code editor, "doc" for the rich-text editor.
	prop string
	// canon renders a host's value the way Go would. nil means the value is
	// plain text and bytes are the comparison.
	canon func(string) string
}

// textEditFields are the leaf nodes whose value a native host edits locally
// and sends as text, keyed by node type. Select also uses a text callback,
// but its value is picked from a list and not typed, so no keystroke can
// outrun it.
//
// The two editors joined the four fields once the natives had the ledger:
// both used to guard their echoes with the value queue this protocol
// replaced, and both have the same race. A CodeEditor rebases like a field,
// because its value is plain text. A RichTextEditor's value is a JSON
// document, which has no "insertion at either end" to replay, so its hosts
// adopt a rewrite as it stands; what it gains is Go dropping the stale edits
// rather than applying them after the rewrite, and an echo recognized by its
// epoch rather than by bytes the two sides spell differently.
var textEditFields = map[string]textEditField{
	"Input":          {prop: "value"},
	"InputPassword":  {prop: "value"},
	"NumericInput":   {prop: "value"},
	"TextArea":       {prop: "value"},
	"CodeEditor":     {prop: "value"},
	"RichTextEditor": {prop: "doc", canon: canonicalDocJSON},
}

// canonicalDocJSON is a host's document JSON as Go would write it. A payload
// that does not parse is returned unchanged: it never equals Go's render, so
// the render reads as a rewrite and the host takes Go's document, which is
// the right answer to a host that sent something unreadable (its onChange
// dropped the edit; see RichTextEditor).
func canonicalDocJSON(payload string) string {
	doc, err := richtext.ParseJSON(payload)
	if err != nil {
		return payload
	}
	return doc.JSON()
}

// stampTextEdit is leafNode's hook: for a typed text field it adds the edit
// stamps, when the field has a ledger.
func stampTextEdit(ctx *Context, typ string, props map[string]any) {
	field, ok := textEditFields[typ]
	if !ok {
		return
	}
	id, _ := props["onChange"].(string)
	value, _ := props[field.prop].(string)
	if id == "" {
		return
	}
	ctx.registry.stampEdit(id, value, field.canon, props)
}

// TriggerTextEdit dispatches one keystroke's worth of text from a native
// field: TriggerTextCallback plus the sequence and epoch the host stamped on
// it. An edit typed before the host had seen Go's latest rewrite of the field
// is acknowledged and dropped rather than handed to the app. See the file
// comment.
//
// Unknown IDs are silent no-ops, as for every Trigger* method.
func (ctx *Context) TriggerTextEdit(id, val string, seq, epoch int) {
	fn, ok := ctx.registry.lookupText(id)
	if !ok {
		return
	}
	if ctx.registry.acceptEdit(id, val, seq, epoch) {
		fn(val)
	}
}
