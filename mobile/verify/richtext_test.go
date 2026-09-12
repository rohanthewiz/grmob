package verify

import "testing"

// core.RichTextEditor's host contract, on both native shells.
//
// The same argument codeeditor_test.go makes, one node over and with more at
// stake: a rich-text editor's *value* is a document, so each host owns a
// serializer in both directions, and a serializer that drifts does not draw
// something ugly — it changes what is stored. The web half has a real test
// (wasm/verify/richtext_test.mjs, which drives both directions and the whole
// command vocabulary); the iOS half type-checks against the real iOS SDK; the
// Kotlin half type-checks under android/verify/sources.sh, against the classpath
// gradle resolves. So what is held here is the shape of the contract, read out
// of the source — which is still the only instrument for the ORDER of the calls,
// since a type-check says they exist and nothing more.
//
// See mapview_test.go for the long version of why that is the strongest
// instrument available for "all four hosts implement the same rule", and
// switchlabels_test.go for what reading source costs.

var (
	swiftRichText  = nativeFile("ios", "GrMob", "Runtime", "GrMobRichText.swift")
	kotlinRichText = nativeFile("android", "app", "src", "main", "java", "com",
		"grmob", "runtime", "GrMobRichText.kt")
)

// The echo guard, over the document's JSON rather than a string of text.
//
// The comparison being on the *JSON string* is the part worth pinning: Go
// marshals with a fixed key order, so the same document is always the same
// bytes, and a host that compared parsed values instead would be doing a deep
// equality on every keystroke for an answer a string compare already has.
func TestBothNativeRichTextEditorsGuardTheirEchoes(t *testing.T) {
	pinExprs(t, swiftRichText, []struct{ expr, why string }{
		{"private var pendingEchoes: [String] = []",
			"the queue of documents sent upstream and not yet seen come back"},
		{"if view.isFirstResponder, let echo = pendingEchoes.firstIndex(of: json)",
			"an upstream document we sent is an echo, not an instruction"},
		{"pendingEchoes.removeSubrange(...echo)",
			"dropped through the match, because Go may coalesce renders"},
	})
	pinExprs(t, kotlinRichText, []struct{ expr, why string }{
		{"private val pendingEchoes = ArrayList<String>()",
			"the queue of documents sent upstream and not yet seen come back"},
		{"val echo = pendingEchoes.indexOf(json)",
			"an upstream document we sent is an echo, not an instruction"},
		{"repeat(echo + 1) { pendingEchoes.removeAt(0) }",
			"dropped through the match, because Go may coalesce renders"},
	})
}

// Commands are epoch-stamped props, and an editor that was not there when one
// was issued missed it — the same rule the code editor is held to, and the same
// failure if it is dropped: returning to a screen would re-apply whatever
// command last named its editor.
func TestBothNativeRichTextEditorsAdoptAStandingEpoch(t *testing.T) {
	pinExprs(t, swiftRichText, []struct{ expr, why string }{
		{"guard let last = lastEpoch else {", "the first pass records the epoch and runs nothing"},
		{"guard epoch != 0, epoch != last else { return }",
			"epoch 0 is Go's 'never issued' sentinel, and an unchanged epoch is not news"},
	})
	pinExprs(t, kotlinRichText, []struct{ expr, why string }{
		{"if (previous == null || epoch == 0 || epoch == previous) return",
			"the first pass records the epoch and runs nothing, epoch 0 is the " +
				"'never issued' sentinel, and an unchanged epoch is not news"},
	})
}

// The block kind is *recorded* rather than inferred.
//
// Each host puts a marker on the paragraph and reads it back, because the
// alternative is to recognize a heading by its font size — a guess that breaks
// the moment a theme changes one, and one that cannot tell a paragraph the user
// made big from a heading at all. The same argument covers the drawn list
// prefix, which without a marker of its own comes back as part of the user's
// text.
func TestBothNativeRichTextEditorsRecordTheBlockKindRatherThanInferringIt(t *testing.T) {
	pinExprs(t, swiftRichText, []struct{ expr, why string }{
		{`static let grMobBlockKind = NSAttributedString.Key("grMobBlockKind")`,
			"the paragraph's kind, carried rather than guessed from its font"},
		{`static let grMobPrefix = NSAttributedString.Key("grMobPrefix")`,
			"the drawn bullet is not the user's text"},
		{"if attrs[.grMobPrefix] != nil { return }",
			"and the reverse mapping drops it"},
	})
	pinExprs(t, kotlinRichText, []struct{ expr, why string }{
		{"internal class GrMobBlockSpan(val kind: String)",
			"the paragraph's kind, carried rather than guessed from its size"},
		{"internal class GrMobPrefixSpan", "the drawn bullet is not the user's text"},
		{"GrMobPrefixSpan::class.java).isEmpty()", "and the reverse mapping drops it"},
	})
}

// Marks are edited in place and block kinds are not, and the split is the
// design: an attribute over a range preserves the caret and costs no rebuild,
// while a block kind changes a paragraph's prefix and indentation and has no
// in-place spelling at all.
//
// A host that rebuilt for a mark would move the caret on every bold press; one
// that tried to edit a block kind in place would leave the old bullet behind.
func TestBothNativeRichTextEditorsEditMarksInPlace(t *testing.T) {
	pinExprs(t, swiftRichText, []struct{ expr, why string }{
		{"storage.setAttributes(GrMobRichMapper.setMark(command, on: on, in: attrs, base: base),",
			"a mark is an attribute over a range, applied without a rebuild"},
		{`if command.hasPrefix("block:") {`, "a block kind takes the long way round"},
		{"rebuild(selecting: range)", "...and restores the selection by offset"},
	})
	pinExprs(t, kotlinRichText, []struct{ expr, why string }{
		{"GrMobRichMapper.applyMark(text, editText.selectionStart,",
			"a mark is a span over a range, applied without a rebuild"},
		{`command.startsWith("block:")`, "a block kind takes the long way round"},
		{"rebuild(start, end)", "...and restores the selection by offset"},
	})
}

// An empty selection sets the typing attributes: press bold, type, and the
// characters arrive bold. Both platforms have a native spelling for it and the
// two are different, which is exactly why this is pinned per host rather than
// assumed — UIKit carries typingAttributes forward, and Android grows a
// zero-length INCLUSIVE_INCLUSIVE span over whatever is typed at the position.
func TestBothNativeRichTextEditorsSetTypingAttributes(t *testing.T) {
	pinExprs(t, swiftRichText, []struct{ expr, why string }{
		{"view.typingAttributes = GrMobRichMapper.setMark(command, on: on,",
			"UIKit's own spelling of a mark with nothing selected"},
	})
	pinExprs(t, kotlinRichText, []struct{ expr, why string }{
		{"else Spanned.SPAN_INCLUSIVE_INCLUSIVE",
			"Android's: a zero-length span that grows over what is typed at it"},
	})
}

// The toggle's direction is decided by *all*, not *any*: selecting a sentence
// with one bold word in it and pressing bold makes the sentence bold rather
// than unbolding the word. Every editor does this, and a host that got it
// backwards would be subtly maddening rather than obviously broken.
func TestBothNativeRichTextEditorsToggleOnAll(t *testing.T) {
	pinExprs(t, swiftRichText, []struct{ expr, why string }{
		{"if !GrMobRichMapper.hasMark(command, in: attrs) { everywhere = false }",
			"every run in the range must carry the mark for the toggle to remove it"},
	})
	pinExprs(t, kotlinRichText, []struct{ expr, why string }{
		{"val on = !marksIn(text, from, to).names.contains(command)",
			"every run in the range must carry the mark for the toggle to remove it"},
	})
}

// The maximal-run rule, restated per host because each builds its own runs.
//
// Without it a document gains a run boundary on every keystroke while looking
// identical on screen — and the runs are the wire, so it grows without bound in
// both the payload and the database.
func TestBothNativeRichTextEditorsMergeAdjacentRuns(t *testing.T) {
	pinExprs(t, swiftRichText, []struct{ expr, why string }{
		{"if var previous = out.last, sameMarks(previous, run) {",
			"adjacent runs with identical formatting are one run"},
	})
	pinExprs(t, kotlinRichText, []struct{ expr, why string }{
		{"if (previous != null && sameMarks(previous, run)) {",
			"adjacent runs with identical formatting are one run"},
	})
}

// Paragraph is the wire's absent kind, so a host that wrote it explicitly would
// produce a document with the same meaning and different bytes — which the echo
// guard compares, so every keystroke in a paragraph would read as a rewrite.
func TestBothNativeRichTextEditorsOmitTheParagraphKind(t *testing.T) {
	pinExprs(t, swiftRichText, []struct{ expr, why string }{
		{`if kind != "p" { block["k"] = kind }`, "an absent kind means paragraph"},
		{`blocks[i].removeValue(forKey: "k")`, "and setting a block back to one removes it"},
	})
	pinExprs(t, kotlinRichText, []struct{ expr, why string }{
		{`if (kind != "p") block.put("k", kind)`, "an absent kind means paragraph"},
		{`if (kind == "p") block.remove("k")`, "and setting a block back to one removes it"},
	})
}

// The selection crosses as UTF-8 byte offsets, which is core.RichSelection's
// promise and the one unit all four hosts can produce. Both natives count
// UTF-16 natively, so both have to convert.
func TestBothNativeRichTextEditorsReportSelectionInBytes(t *testing.T) {
	pinExprs(t, swiftRichText, []struct{ expr, why string }{
		{"static func utf8Offset(in text: NSString, utf16Offset: Int) -> Int",
			"UIKit counts UTF-16 code units; the wire carries bytes"},
		{"(text.substring(to: clamped) as String).utf8.count", "the conversion itself"},
	})
	pinExprs(t, kotlinRichText, []struct{ expr, why string }{
		{"fun utf8Offset(text: CharSequence, offset: Int): Int",
			"Android counts UTF-16 chars; the wire carries bytes"},
		{"toByteArray(Charsets.UTF_8).size", "the conversion itself"},
	})
}

// read-only is not disabled here either: a read-only document is content the
// reader is meant to select and copy.
func TestBothNativeRichTextEditorsTreatReadOnlyAsNotDisabled(t *testing.T) {
	pinExprs(t, swiftRichText, []struct{ expr, why string }{
		{`view.isEditable = !node.boolProp("readOnly")`,
			"isEditable, not isUserInteractionEnabled"},
	})
	pinExprs(t, kotlinRichText, []struct{ expr, why string }{
		{"view.setTextIsSelectable(readOnly)",
			"a read-only document is still selectable and copyable"},
	})
}

// The Android editor reaches past Compose on purpose, and the reason is
// Spannable. This is the one arm of the node census where the two natives'
// *approach* differs rather than their spelling, so it is pinned rather than
// left to be discovered by someone wondering why one file imports AndroidView.
func TestTheAndroidRichTextEditorUsesAClassicEditText(t *testing.T) {
	pinExprs(t, kotlinRichText, []struct{ expr, why string }{
		{"import androidx.compose.ui.viewinterop.AndroidView",
			"the escape hatch osmdroid is hosted through"},
		{"EditText(context).apply {",
			"an Editable is a string plus spans, which is richtext.Doc's own shape"},
	})
}

// Focus commands, on the two hosts that host a classic text view.
//
// This node is the one place in the runtime where neither phone can hand a
// focus command to its own framework. Both editors are a UIKit/Android View
// inside the declarative tree — a UITextView in a UIViewRepresentable, an
// EditText in an AndroidView — so SwiftUI's @FocusState and Compose's
// FocusRequester both address the *wrapper* and the text control never hears
// them.
//
// Android carries one extra obligation that no other node in this runtime has,
// and it is the one most likely to be dropped by someone copying the Compose
// field's implementation: a classic View does not raise or lower the soft
// keyboard as a consequence of focus. requestFocus() leaves the keyboard down
// and clearFocus() leaves it up, so the InputMethodManager has to be asked on
// both edges — which is the whole point of core.DismissKeyboard reaching here.
func TestBothNativeRichTextEditorsTakeFocusCommands(t *testing.T) {
	pinExprs(t, swiftRichText, []struct{ expr, why string }{
		{`node.intProp("focusEpoch")`, "the command's generation"},
		{`node.stringProp("focusAction")`, "focus / blur / \"\""},
		{"focus.apply(epoch: epoch, action: action, to: view)",
			"applied to the hosted UITextView's responder, which is the only" +
				" thing SwiftUI's focus system cannot reach"},
	})
	pinExprs(t, kotlinRichText, []struct{ expr, why string }{
		{`node.intProp("focusEpoch")`, "the command's generation"},
		{`node.stringProp("focusAction")`, "focus / blur / \"\""},
		{"if (epoch == 0 || epoch == lastFocusEpoch) return",
			"a remembered epoch: update runs every pass, and the stamp stays on" +
				" the node forever, so without this one command re-takes focus" +
				" on every later pass"},
		{"imm?.showSoftInput(editText, InputMethodManager.SHOW_IMPLICIT)",
			"a classic View leaves the keyboard down after requestFocus()"},
		{"imm?.hideSoftInputFromWindow(editText.windowToken, 0)",
			"and leaves it up after clearFocus(), which is the half" +
				" core.DismissKeyboard exists for"},
	})
}
