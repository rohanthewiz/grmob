package verify

import (
	"strings"
	"testing"
)

// core.CodeEditor's host contract, on both native shells.
//
// # Why this file exists at all
//
// The editor is the node whose correctness is least in Go. Go decides the
// colours and nothing else: the buffer belongs to the host, the caret belongs
// to the host, and the three rules that make a controlled editor usable — the
// echo guard, the per-line stale rule, the command epoch — are decisions each
// host makes in its own language against its own text control.
//
// The web half has a real test (wasm/verify/codeeditor_test.mjs, which drives
// the runtime against the harness DOM). The iOS half type-checks under
// ios/verify against the real iOS SDK, which says the calls exist and not that
// they are made in the right order. The Kotlin half compiles under
//
//	android/build.sh && (cd android && ./gradlew :app:assembleDebug)
//
// which needs the Android SDK and an NDK and so is not part of `go test ./...`.
//
// So what is held here is the shape of the contract, read out of the source. It
// is a weaker instrument than a test that drives the code, and it is the
// strongest one available for the claim "all four hosts implement the same
// rule". mapview_test.go carries the same argument at length; see
// switchlabels_test.go for why reading source is the technique and what it
// costs.

var (
	swiftCodeEditor  = nativeFile("ios", "GrMob", "Runtime", "GrMobCodeEditor.swift")
	kotlinCodeEditor = nativeFile("android", "app", "src", "main", "java", "com",
		"grmob", "runtime", "GrMobCodeEditor.kt")
)

// pinExprs asserts that each expression appears in the file, naming why.
//
// valuesIn rather than codeIn: most of what is pinned below is a call with a
// *prop name* in it — `node.boolProp("readOnly")`, `if text == "\n"` — and the
// literal is as much the subject as the call is. Blanking literals would leave
// several of these matching a call to the same function about some other prop.
// Comments are still blanked, which is the part that matters: a note saying a
// host guards its echoes must not satisfy a check that it does.
func pinExprs(t *testing.T, file string, pins []struct{ expr, why string }) {
	t.Helper()
	src := valuesIn(t, file)
	for _, pin := range pins {
		if !strings.Contains(src, pin.expr) {
			t.Errorf("%s: %q not found — %s", file, pin.expr, pin.why)
		}
	}
}

// Rule 1: the echo guard, which is what makes a controlled buffer typeable.
//
// Every host keeps a queue of the values it has sent upstream. An upstream
// value matching a queued entry is this editor's own edit coming back and must
// be dropped — applying it would move the caret to the end mid-typing. One
// matching nothing we sent can only be Go speaking for itself and must land
// even mid-typing. And the queue is dropped *through* the match rather than at
// it, because Go coalesces renders and skips intermediate values.
//
// This is the same contract core.TextArea has lived under since it existed; the
// point of pinning it here is that a new node type had to restate it, and a
// restatement that dropped one of the three arms would be a caret that jumps
// on a slow network and nowhere else.
func TestBothNativeCodeEditorsGuardTheirEchoes(t *testing.T) {
	pinExprs(t, swiftCodeEditor, []struct{ expr, why string }{
		{"private var pendingEchoes: [String] = []",
			"the queue of values sent upstream and not yet seen come back"},
		{"if let echo = pendingEchoes.firstIndex(of: value)",
			"an upstream value we sent is an echo, not an instruction"},
		{"pendingEchoes.removeSubrange(...echo)",
			"dropped through the match, because Go may coalesce renders"},
		{"if textView.isFirstResponder",
			"the buffer is the host's while focused and Go's otherwise"},
	})
	pinExprs(t, kotlinCodeEditor, []struct{ expr, why string }{
		{"val pendingEchoes = remember { mutableListOf<String>() }",
			"the queue of values sent upstream and not yet seen come back"},
		{"val echo = pendingEchoes.indexOf(upstream)",
			"an upstream value we sent is an echo, not an instruction"},
		{"repeat(echo + 1) { pendingEchoes.removeAt(0) }",
			"dropped through the match, because Go may coalesce renders"},
		{"if (!focused) {",
			"the buffer is the host's while focused and Go's otherwise"},
	})
}

// Rule 2: decoration is advisory and PER LINE.
//
// Each host compares a row's concatenated text with the line under it and
// paints only where the two agree. The comparison is the whole rule, and the
// failure it prevents is specific: Go is a keystroke behind for a few
// milliseconds after every keypress, so its rows describe the text as it was
// *before* the key. A host that painted them anyway would show the user the
// previous line while they type the current one.
//
// The other half is that decoration never runs the other way. Nothing here may
// write a row's text into the buffer, which is why each host's paint path is an
// attribute or span operation rather than an assignment of text.
func TestBothNativeCodeEditorsSkipAStaleRow(t *testing.T) {
	pinExprs(t, swiftCodeEditor, []struct{ expr, why string }{
		{"guard decorated == lineText else { return }",
			"a row that no longer describes its line is skipped, not painted"},
		{"storage.setAttributes(",
			"the colours are attributes over the existing characters; assigning " +
				"attributedText would rebuild the storage and move the caret"},
	})
	pinExprs(t, kotlinCodeEditor, []struct{ expr, why string }{
		{"if (runs == null || runsText(runs) != line) {",
			"a row that no longer describes its line is skipped, not painted"},
		{"OffsetMapping.Identity",
			"the transformation is a colouring: it adds spans and never a " +
				"character, so visual offset N is buffer offset N"},
		{"visualTransformation = transformation",
			"the colours go through the transformation, never through the " +
				"TextFieldValue — which is what keeps an IME composition alive"},
	})
}

// Rule 3: commands are epoch-stamped props, and an editor that was not there
// when one was issued missed it.
//
// The epoch guard has two arms and they are different failures. Re-running on
// an unchanged epoch is the ordinary one — an update-props patch carries the
// whole props map, so an editor re-rendered for its value would re-run whatever
// command was last issued. Running on *first sight* is the subtle one, and it
// is where an editor deliberately differs from a focus command: a focus command
// re-fires on a field that mounts while it is the target (that is what makes
// "push a screen and put the cursor in its search box" work), and an editor
// command must not, or returning to a screen would re-indent its buffer.
func TestBothNativeCodeEditorsAdoptAStandingEpochWithoutRunningIt(t *testing.T) {
	pinExprs(t, swiftCodeEditor, []struct{ expr, why string }{
		{"guard let last = lastEpoch else {",
			"the first pass records the epoch and runs nothing"},
		{"guard epoch != 0, epoch != last else { return }",
			"epoch 0 is Go's 'never issued' sentinel, and an unchanged epoch is not news"},
	})
	pinExprs(t, kotlinCodeEditor, []struct{ expr, why string }{
		{"if (previous == null || editorEpoch == 0 || editorEpoch == previous) {",
			"the first pass records the epoch and runs nothing, epoch 0 is the " +
				"'never issued' sentinel, and an unchanged epoch is not news"},
	})
}

// The four settings that corrupt source, refused on both hosts.
//
// Each one of these rewrites what the user typed: autocorrect changes
// identifiers, capitalization capitalises the first keyword of every line,
// smart quotes turn " into a curly quote that no compiler accepts, and smart
// dashes turn -- into an em dash. They are on by default in both platforms'
// text controls, so every one of them has to be turned off by name.
//
// Compose has two of the four rather than four: KeyboardOptions carries
// autoCorrect and capitalization, and Android's IME has no smart-quote or
// smart-dash substitution to refuse.
func TestBothNativeCodeEditorsRefuseTheSubstitutions(t *testing.T) {
	pinExprs(t, swiftCodeEditor, []struct{ expr, why string }{
		{"textView.autocorrectionType = .no", "autocorrect rewrites identifiers"},
		{"textView.autocapitalizationType = .none", "every line would start with a capital"},
		{"textView.smartQuotesType = .no", `" would become a curly quote no compiler accepts`},
		{"textView.smartDashesType = .no", "-- would become an em dash"},
	})
	pinExprs(t, kotlinCodeEditor, []struct{ expr, why string }{
		{"autoCorrect = false", "autocorrect rewrites identifiers"},
		{"capitalization = KeyboardCapitalization.None", "every line would start with a capital"},
	})
}

// Tab and Return, the two keys a programmer's editor takes away from the
// platform, and the one place the two hosts legitimately differ about *where*.
//
// Tab would otherwise move focus out of the editor, which makes indenting
// impossible. Return would insert a bare newline at column zero, which
// un-indents every block as it is written.
//
// iOS catches both in shouldChangeTextIn, because UIKit routes a hardware Tab
// and every Return through it. Compose does not: an IME inserts a newline
// through the text input session and never as a key event, so a key handler
// would auto-indent on a tablet with a keyboard and nowhere else. The Compose
// host therefore catches Tab as a key and Return on the value.
func TestBothNativeCodeEditorsOwnTabAndReturn(t *testing.T) {
	pinExprs(t, swiftCodeEditor, []struct{ expr, why string }{
		{`if text == "\n" {`, "Return copies the previous line's indentation"},
		{`if text == "\t" {`, "Tab inserts an indent rather than moving focus"},
		{"private func leadingWhitespace(before location: Int",
			"what Return copies: the indentation of the line the caret is in"},
	})
	pinExprs(t, kotlinCodeEditor, []struct{ expr, why string }{
		{"commit(autoIndent(buffer, next, indentUnit(tabSize)))",
			"Return is caught on the value, so a soft keyboard's newline is " +
				"auto-indented too"},
		{"event.key != Key.Tab", "Tab is caught as a key: no soft keyboard has one"},
		{"if (caret <= 0 || next.text[caret - 1] != '\\n') return next",
			"the narrow test for 'a newline was just inserted'"},
	})
}

// The selection crosses as UTF-8 byte offsets, which is the unit
// core.OnSelectionChange promises and the one thing all four hosts can agree
// on. Each host counts something else natively — UTF-16 on Android, UTF-16 on
// the DOM, String.Index on iOS — so each has to convert, and a host that
// skipped the conversion would report offsets that are right for ASCII and
// silently wrong for every accented character on screen.
func TestBothNativeCodeEditorsReportSelectionInBytes(t *testing.T) {
	pinExprs(t, swiftCodeEditor, []struct{ expr, why string }{
		{"private func utf8Offset(in text: String, utf16Offset: Int) -> Int",
			"UIKit counts UTF-16 code units; the wire carries bytes"},
		{"text.utf8.distance(from: text.utf8.startIndex, to: index)", "the conversion itself"},
	})
	pinExprs(t, kotlinCodeEditor, []struct{ expr, why string }{
		{"internal fun utf8Offset(text: String, offset: Int): Int",
			"Compose counts UTF-16 chars; the wire carries bytes"},
		{"toByteArray(Charsets.UTF_8).size", "the conversion itself"},
	})
}

// readOnly is not disabled, on either host.
//
// A disabled control is inert and greyed and is skipped by assistive
// technology's traversal; a read-only code block is *content* — the user is
// meant to read it, select it and copy out of it. core.ReadOnly's doc says so
// and this is the pair of lines that make it true on the phones. A host that
// implemented readOnly as disabled would ship a code block nobody can copy from,
// which is a screenshot.
func TestBothNativeCodeEditorsTreatReadOnlyAsNotDisabled(t *testing.T) {
	pinExprs(t, swiftCodeEditor, []struct{ expr, why string }{
		{`view.textView.isEditable = !node.boolProp("readOnly")`,
			"isEditable, not isUserInteractionEnabled: a read-only buffer still selects"},
	})
	pinExprs(t, kotlinCodeEditor, []struct{ expr, why string }{
		{"readOnly = readOnly", "BasicTextField's own read-only state"},
		{"enabled = !node.isDisabled()",
			"enabled tracks core.Disabled alone, so a read-only editor is still focusable"},
	})
}

// Neither host may wrap. A wrapped code line restarts at column zero, which
// reads as a new statement at the outermost indent — so wrapping destroys
// exactly the structure indentation exists to show. Both hosts therefore give
// the buffer unbounded width and pan it sideways instead.
func TestNeitherNativeCodeEditorWraps(t *testing.T) {
	pinExprs(t, swiftCodeEditor, []struct{ expr, why string }{
		{"textView.textContainer.widthTracksTextView = false",
			"the container must not be sized to the view, or the text wraps to it"},
		{"textView.textContainer.lineBreakMode = .byClipping", "and must not break lines"},
	})
	pinExprs(t, kotlinCodeEditor, []struct{ expr, why string }{
		{"Box(Modifier.horizontalScroll(horizontal))",
			"the field sits in an unbounded-width scroll, so a long line pans " +
				"rather than wrapping"},
	})
}

// The gutter is chrome, not content, on both hosts: a sibling view rather than
// characters in the buffer. Numbers inside the text would be selectable,
// copyable and editable, and a buffer whose first columns are not the user's is
// not the buffer.
func TestBothNativeCodeEditorsDrawTheGutterBesideTheBuffer(t *testing.T) {
	pinExprs(t, swiftCodeEditor, []struct{ expr, why string }{
		{"private let gutter = UILabel()", "a sibling view, not text in the buffer"},
		{"gutter.isUserInteractionEnabled = false",
			"a drag over the numbers must reach the buffer behind them"},
	})
	pinExprs(t, kotlinCodeEditor, []struct{ expr, why string }{
		{"if (lineNumbers) {", "the gutter is a sibling composable, not text in the field"},
		{"Row(s.boxModifier(extra).verticalScroll(vertical))",
			"the gutter and the field share one vertical scroll, so number N " +
				"stays beside line N"},
	})
}
