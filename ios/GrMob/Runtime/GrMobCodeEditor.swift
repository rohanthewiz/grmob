import SwiftUI

/// core.CodeEditor: an editable monospace buffer with syntax colour, a
/// line-number gutter and the keyboard behaviour a programmer's editor has.
///
/// # Why this is not a SwiftUI view
///
/// SwiftUI has no editable styled-text control. `TextEditor` binds a plain
/// `String` and draws it in one font; there is no way to give run N of line M a
/// colour, and no way to lay a second view over it at exactly its pitch,
/// because the pitch is not something SwiftUI will tell you. `UITextView` is
/// the control that has always had this — one `NSTextStorage` carrying both the
/// characters and their attributes — so the editor is a UIViewRepresentable
/// over one, the same shape GrMobMapView takes over `MKMapView`.
///
/// # The three rules, as they land here
///
///  1. **Echo guard.** Identical bookkeeping to GrMobTextField's
///     `pendingEchoes`: the buffer is the host's while it is first responder
///     and Go's otherwise, and an upstream value matching one we sent is our
///     own edit coming back rather than an instruction.
///
///  2. **Decoration is advisory and per line.** Go's rows are applied as
///     *attributes over the existing characters* — never by assigning
///     `attributedText`, which rebuilds the storage and puts the caret at the
///     end. A row whose text no longer equals the line under it is skipped and
///     that line is drawn in the base ink until the next patch, so the line
///     being typed loses its colours for a frame and no other line does.
///
///  3. **Commands are epoch-stamped props.** See `runCommand` for the one way
///     this differs from a focus command: an editor that mounts under a
///     standing epoch adopts it without running it.
///
/// # What the IME needs, and gets
///
/// Every path that would touch the buffer checks `markedTextRange` first. While
/// the user is composing — a Pinyin candidate, a Japanese conversion — the
/// marked text is uncommitted and belongs entirely to the input method; both
/// reporting it upstream and re-attributing over it would fight the
/// composition. So composition is left alone and the buffer catches up when it
/// commits.
///
/// # The type-check gap, stated
///
/// This file's real implementation is behind `canImport(UIKit)`, which is false
/// for the macOS target `ios/verify` type-checks against. So what runs in that
/// pass is the fallback below, and the UIKit half is held to the contract
/// textually by `mobile/verify/codeeditor_test.go` and by a simulator run.
/// GrMobMapView carries the identical caveat for the identical reason; this is
/// not a new gap, but it is the second file in it.
#if canImport(UIKit)

import UIKit

struct GrMobCodeEditor: View {
    let node: GrMobNode
    let grow: GrMobGrow

    @Environment(\.grMobRuntime) private var runtime

    var body: some View {
        GrMobCodeEditorRepresentable(node: node, runtime: runtime)
            .grMobBox(node.style, grow: grow)
    }
}

// MARK: - The representable

private struct GrMobCodeEditorRepresentable: UIViewRepresentable {
    let node: GrMobNode
    let runtime: GrMobRuntime?

    func makeCoordinator() -> GrMobCodeCoordinator { GrMobCodeCoordinator() }

    func makeUIView(context: Context) -> GrMobCodeEditorView {
        let view = GrMobCodeEditorView()
        view.textView.delegate = context.coordinator
        context.coordinator.view = view
        return view
    }

    /// Order matters here and is worth reading top to bottom.
    ///
    /// The coordinator's settings go first because everything below may call
    /// back into it. The value goes before the rows, because the rows are
    /// compared against the buffer's lines and a stale buffer would make every
    /// row look stale. The gutter goes after both, because its width is a
    /// function of how many lines there now are. The command goes last, because
    /// it acts on the buffer this pass just settled.
    func updateUIView(_ view: GrMobCodeEditorView, context: Context) {
        let coordinator = context.coordinator
        coordinator.runtime = runtime
        coordinator.onChange = node.stringProp("onChange")
        coordinator.onSelectionChange = node.stringProp("onSelectionChange")
        coordinator.tabSize = node.intProp("tabSize")
        coordinator.commentPrefix = node.stringProp("commentPrefix")

        view.applyStyle(node.style)
        // isEditable, not isUserInteractionEnabled: a read-only buffer still
        // focuses, still shows a caret and still selects, which is the whole
        // difference between read-only and disabled. core.ReadOnly says so.
        view.textView.isEditable = !node.boolProp("readOnly")
        view.showsGutter = node.boolProp("lineNumbers")

        coordinator.applyValue(node.stringProp("value"))
        coordinator.applyRows(node.children, base: node.style)
        view.refreshGutter()
        coordinator.runCommand(epoch: node.intProp("editorEpoch"),
                               command: node.stringProp("editorCommand"))
        // core.Focus / core.DismissKeyboard, last: the responder should arrive
        // at a buffer this pass has already settled, so that a command issued
        // alongside a value change does not put the caret in a stale one.
        //
        // The command lands on textView and never on the view itself: the
        // editor's own UIView is the box holding the buffer and the gutter, and
        // the gutter is chrome the caret must never reach. See
        // GrMobEditorFocus.swift.
        coordinator.applyFocus(epoch: node.intProp("focusEpoch"),
                               action: node.stringProp("focusAction"))
    }
}

// MARK: - The view: a text view and the gutter beside it

/// The editor's box: the buffer, and the line numbers to its left.
///
/// The gutter is a sibling rather than an inset inside the text container
/// because the numbers must not be part of the text — text is selectable,
/// copyable and editable, and a buffer whose first columns are not the user's
/// is not the buffer. It scrolls with the text view instead, which is what
/// `scrollViewDidScroll` re-lays it out for.
final class GrMobCodeEditorView: UIView {
    let textView = UITextView()
    private let gutter = UILabel()

    var showsGutter = false {
        didSet {
            gutter.isHidden = !showsGutter
            setNeedsLayout()
        }
    }

    /// The monospace face everything here is measured in. One font for the
    /// buffer and the gutter, which is what keeps number N beside line N
    /// without measuring anything: every line is exactly one line fragment
    /// tall, because nothing wraps.
    private(set) var font = UIFont.monospacedSystemFont(ofSize: 13, weight: .regular)
    private var ink = UIColor.label
    private var lineCount = 1

    override init(frame: CGRect) {
        super.init(frame: frame)

        textView.backgroundColor = .clear
        textView.textContainerInset = .zero
        textView.textContainer.lineFragmentPadding = 0
        // No wrapping: a code line is one line, and a wrapped one restarts at
        // column zero, which reads as a new statement at the outermost indent.
        // So the container is given unbounded width and the text view scrolls
        // sideways instead.
        textView.textContainer.lineBreakMode = .byClipping
        textView.textContainer.widthTracksTextView = false
        textView.textContainer.size = CGSize(width: CGFloat.greatestFiniteMagnitude,
                                             height: CGFloat.greatestFiniteMagnitude)
        textView.isScrollEnabled = true
        textView.alwaysBounceHorizontal = false
        // Every one of these corrupts source. Smart quotes turn " into “,
        // smart dashes turn -- into an em dash, autocorrect rewrites
        // identifiers, and autocapitalization capitalises the first keyword of
        // every line.
        textView.autocorrectionType = .no
        textView.autocapitalizationType = .none
        textView.smartQuotesType = .no
        textView.smartDashesType = .no
        textView.smartInsertDeleteType = .no
        textView.spellCheckingType = .no

        gutter.numberOfLines = 0
        gutter.textAlignment = .right
        gutter.isUserInteractionEnabled = false
        gutter.isHidden = true

        addSubview(textView)
        addSubview(gutter)
    }

    @available(*, unavailable)
    required init?(coder: NSCoder) { fatalError("GrMobCodeEditorView is not loaded from a nib") }

    /// The Go style's font size and ink, resolved into the one font and the one
    /// colour the whole editor is drawn in. The *syntax* colours come from the
    /// rows and override this per run; this is the default ink underneath them.
    func applyStyle(_ style: GrMobStyle?) {
        let size = (style?.fontSize ?? 0) > 0 ? style!.fontSize : 13
        let next = UIFont.monospacedSystemFont(ofSize: size, weight: .regular)
        let nextInk = UIColor(style?.textColor ?? Color.primary)
        guard next != font || nextInk != ink else { return }
        font = next
        ink = nextInk
        gutter.font = next
        gutter.textColor = nextInk.withAlphaComponent(0.45)
        textView.font = next
        textView.textColor = nextInk
        setNeedsLayout()
    }

    /// The gutter's content and width, recomputed from the buffer.
    ///
    /// The width is the widest number plus a column of room, measured in the
    /// advance of a "0" — which in a fixed-pitch font is the advance of every
    /// glyph. Both DOM targets say the same thing as `Nch`.
    func refreshGutter() {
        let text = textView.text ?? ""
        lineCount = max(1, text.reduce(1) { $1 == "\n" ? $0 + 1 : $0 })
        guard showsGutter else {
            setNeedsLayout()
            return
        }
        gutter.text = (1...lineCount).map(String.init).joined(separator: "\n")
        setNeedsLayout()
    }

    private var gutterWidth: CGFloat {
        guard showsGutter else { return 0 }
        let digits = String(lineCount).count
        let advance = ("0" as NSString).size(withAttributes: [.font: font]).width
        return CGFloat(digits + 2) * advance
    }

    override func layoutSubviews() {
        super.layoutSubviews()
        let inset = gutterWidth
        textView.frame = CGRect(x: inset, y: 0,
                                width: max(0, bounds.width - inset), height: bounds.height)
        // The gutter is pinned to the left of the box and offset upward by
        // however far the buffer has scrolled, so number N stays beside line N.
        // Its height is the whole text, not the visible box, which is what lets
        // the offset carry it off the top.
        gutter.frame = CGRect(x: 0, y: -textView.contentOffset.y,
                              width: inset,
                              height: max(bounds.height, textView.contentSize.height))
        // Room for the digits, so the gutter's own trailing column is a gap
        // rather than a number touching the code.
        gutter.frame = gutter.frame.insetBy(dx: 0, dy: 0)
    }
}

// MARK: - The coordinator: everything that is a decision rather than a frame

final class GrMobCodeCoordinator: NSObject, UITextViewDelegate {
    weak var view: GrMobCodeEditorView?
    var runtime: GrMobRuntime?
    var onChange = ""
    var onSelectionChange = ""
    var tabSize = 4
    var commentPrefix = "//"

    /// Every value this editor has sent upstream and not yet seen come back.
    /// See GrMobTextField for the full argument; the bookkeeping is identical.
    private var pendingEchoes: [String] = []
    /// The last selection reported, so the several delegate calls one gesture
    /// produces cost one Go render pass rather than several.
    private var lastSelection = ""
    /// nil until the first pass has been seen. See runCommand.
    private var lastEpoch: Int?
    /// The focus-command memory. See GrMobEditorFocus.swift for why it is
    /// separate from lastEpoch above, which tracks a different kind of command
    /// with a deliberately different first-sight rule.
    private var focus = GrMobEditorFocus()

    /// One indent: tabSize spaces, or a literal tab when tabSize is 0 — which
    /// is what Go source wants.
    private var indentUnit: String {
        tabSize > 0 ? String(repeating: " ", count: tabSize) : "\t"
    }

    // MARK: Go -> the buffer

    func applyValue(_ value: String) {
        guard let textView = view?.textView, textView.markedTextRange == nil else { return }

        if textView.isFirstResponder {
            if let echo = pendingEchoes.firstIndex(of: value) {
                // Dropped *through* the match rather than at it: Go may
                // coalesce renders and skip intermediate values.
                pendingEchoes.removeSubrange(...echo)
                return
            }
            // Not an echo, so this can only be Go speaking for itself — a
            // validator normalizing the text, a draft cleared after a submit.
            // It wins even mid-typing, and moving the caret is correct: the
            // text under it was replaced.
            pendingEchoes.removeAll()
        } else {
            // Go-owned while blurred; any queued echoes died with the session.
            pendingEchoes.removeAll()
        }
        if textView.text != value {
            textView.text = value
        }
    }

    /// Rule 2: Go's rows, applied per line, only where they still describe what
    /// is on screen.
    ///
    /// `setAttributes` over the existing characters, never `attributedText =`.
    /// The second rebuilds the storage, which moves the caret to the end and
    /// throws away the selection — on every keystroke, since every keystroke
    /// brings a new set of rows a few milliseconds later.
    func applyRows(_ rows: [GrMobNode], base: GrMobStyle?) {
        guard let editor = view, let textView = view?.textView,
              textView.markedTextRange == nil else { return }

        let ns = (textView.text ?? "") as NSString
        let plain: [NSAttributedString.Key: Any] = [
            .font: editor.font,
            .foregroundColor: textView.textColor ?? UIColor.label,
        ]
        // Newly typed characters inherit the *base*, not whatever run the caret
        // happens to sit at the end of — otherwise typing after a string
        // literal continues in the string colour until Go catches up.
        textView.typingAttributes = plain

        let storage = textView.textStorage
        storage.beginEditing()
        var lineStart = 0
        var index = 0
        while true {
            let rest = NSRange(location: lineStart, length: ns.length - lineStart)
            let newline = ns.range(of: "\n", options: [], range: rest)
            let lineEnd = newline.location == NSNotFound ? ns.length : newline.location
            let lineRange = NSRange(location: lineStart, length: lineEnd - lineStart)

            // Reset first, always: a line that has *lost* its agreement must
            // lose the colours it had, and painting only the agreeing lines
            // would leave the old decoration standing on the others.
            if lineRange.length > 0 {
                storage.setAttributes(plain, range: lineRange)
                applyRuns(index < rows.count ? rows[index] : nil,
                          lineRange: lineRange, lineText: ns.substring(with: lineRange),
                          storage: storage, plain: plain)
            }

            if newline.location == NSNotFound { break }
            lineStart = lineEnd + 1
            index += 1
        }
        storage.endEditing()
    }

    /// One line's runs, applied only if their concatenation is still exactly
    /// the line. A row that disagrees leaves the line in the base ink — see
    /// core.CodeEditor's rule 2 for why this never runs the other way.
    private func applyRuns(_ row: GrMobNode?, lineRange: NSRange, lineText: String,
                           storage: NSTextStorage, plain: [NSAttributedString.Key: Any]) {
        guard let runs = row?.props["runs"] as? [[String: Any]] else { return }
        var decorated = ""
        for run in runs { decorated += (run["t"] as? String) ?? "" }
        guard decorated == lineText else { return }

        var at = lineRange.location
        for run in runs {
            let text = (run["t"] as? String) ?? ""
            let length = (text as NSString).length
            if length == 0 { continue }
            var attrs = plain
            let bits = (run["a"] as? NSNumber)?.intValue ?? 0
            if let fg = GrMobStyle.parseColor(run["fg"] as? String) {
                attrs[.foregroundColor] = UIColor(fg)
            }
            if let bg = GrMobStyle.parseColor(run["bg"] as? String) {
                attrs[.backgroundColor] = UIColor(bg)
            }
            // The Grid* attribute bits, in the spellings a monospace face has.
            // Bold and italic go through the symbolic traits so the fixed pitch
            // survives them; a synthesized oblique would not.
            if let base = attrs[.font] as? UIFont, bits & 0x5 != 0 {
                var traits = base.fontDescriptor.symbolicTraits
                if bits & 1 != 0 { traits.insert(.traitBold) }
                if bits & 4 != 0 { traits.insert(.traitItalic) }
                if let descriptor = base.fontDescriptor.withSymbolicTraits(traits) {
                    attrs[.font] = UIFont(descriptor: descriptor, size: base.pointSize)
                }
            }
            // Dim has no attribute of its own, so it fades whatever colour the
            // run ended up with — the same thing GrMobGridRow does.
            if bits & 2 != 0, let fg = attrs[.foregroundColor] as? UIColor {
                attrs[.foregroundColor] = fg.withAlphaComponent(0.6)
            }
            if bits & 8 != 0 { attrs[.underlineStyle] = NSUnderlineStyle.single.rawValue }
            if bits & 16 != 0 { attrs[.strikethroughStyle] = NSUnderlineStyle.single.rawValue }
            storage.setAttributes(attrs, range: NSRange(location: at, length: length))
            at += length
        }
    }

    // MARK: The buffer -> Go

    func textViewDidChange(_ textView: UITextView) {
        guard textView.markedTextRange == nil else { return }
        let value = textView.text ?? ""
        pendingEchoes.append(value)
        view?.refreshGutter()
        if !onChange.isEmpty { runtime?.textChanged(onChange, value) }
    }

    /// The selection, as byte offsets into the UTF-8 value.
    ///
    /// UIKit counts UTF-16 code units and core.OnSelectionChange promises
    /// bytes, so the conversion is this method's whole job beyond the dispatch.
    /// It is the same conversion the other three hosts make from their own
    /// unit, which is what lets app code see one number.
    func textViewDidChangeSelection(_ textView: UITextView) {
        guard !onSelectionChange.isEmpty else { return }
        let text = textView.text ?? ""
        let range = textView.selectedRange
        let payload = "\(utf8Offset(in: text, utf16Offset: range.location))"
            + ":\(utf8Offset(in: text, utf16Offset: range.location + range.length))"
        guard payload != lastSelection else { return }
        lastSelection = payload
        runtime?.textChanged(onSelectionChange, payload)
    }

    func scrollViewDidScroll(_ scrollView: UIScrollView) {
        // The gutter is a sibling, so it does not scroll on its own.
        view?.setNeedsLayout()
    }

    /// The two keys a programmer's editor takes away from the platform.
    ///
    /// Return would otherwise insert a bare newline at column zero, which
    /// un-indents every block as it is written; Tab (from a hardware keyboard —
    /// the soft keyboard has no Tab) would insert a literal tab whatever the
    /// buffer's indent convention is. Both are re-implemented and every other
    /// key is left alone, including the ones that make a UITextView worth using
    /// — undo, word motion, dictation, and the input method.
    ///
    /// Single characters only, so a *paste* containing newlines or tabs is not
    /// caught: pasted text is the user's own and is inserted verbatim.
    func textView(_ textView: UITextView, shouldChangeTextIn range: NSRange,
                  replacementText text: String) -> Bool {
        if text == "\n" {
            let indent = leadingWhitespace(before: range.location, in: textView)
            insert("\n" + indent, at: range, in: textView)
            return false
        }
        if text == "\t" {
            insert(indentUnit, at: range, in: textView)
            return false
        }
        return true
    }

    // MARK: Commands

    /// One core.RunEditorCommand.
    ///
    /// # Why an editor adopts a standing epoch instead of running it
    ///
    /// A focus command deliberately re-fires on a field that mounts while it is
    /// the target — that is what makes "push a screen and put the cursor in its
    /// search box" work. An editor command is the opposite: it names a *moment*
    /// and an edit, and an editor that was not on screen when it was issued
    /// missed it. Running it at mount would indent a buffer every time its
    /// screen came back. So the first pass records the epoch and runs nothing;
    /// only a change after that is an instruction. All four hosts agree on this.
    /// One core.Focus or core.DismissKeyboard, applied to the buffer.
    func applyFocus(epoch: Int, action: String) {
        focus.apply(epoch: epoch, action: action, to: view?.textView)
    }

    func runCommand(epoch: Int, command: String) {
        guard let last = lastEpoch else {
            lastEpoch = epoch
            return
        }
        guard epoch != 0, epoch != last else { return }
        lastEpoch = epoch
        guard let textView = view?.textView else { return }

        if command == "selectAll" {
            // Allowed on a read-only buffer: selecting is reading, which is
            // exactly what read-only permits.
            textView.selectedRange = NSRange(location: 0, length: (textView.text as NSString).length)
            return
        }
        guard textView.isEditable else { return }
        switch command {
        case "indent", "outdent", "commentLine":
            transformLines(command, in: textView)
        default:
            // A toolbar that outgrew its editor is a no-op, not a crash.
            break
        }
    }

    /// The three line commands. The selection is first widened to whole lines,
    /// because all three are line commands — indenting "the middle of line 4"
    /// means indenting line 4 — and the rewritten block then takes the
    /// selection, so a second indent indents the same lines rather than a range
    /// that has drifted under the first one's inserted characters.
    private func transformLines(_ command: String, in textView: UITextView) {
        let ns = (textView.text ?? "") as NSString
        var block = ns.lineRange(for: textView.selectedRange)
        // lineRange(for:) includes the paragraph terminator; dropping it keeps
        // the split from producing a phantom empty line after the block.
        while block.length > 0, ns.character(at: block.location + block.length - 1) == 10 {
            block.length -= 1
        }

        let lines = ns.substring(with: block).components(separatedBy: "\n")
        let replacement: String
        switch command {
        case "indent":
            replacement = lines.map { indentUnit + $0 }.joined(separator: "\n")
        case "outdent":
            replacement = lines.map { outdent($0) }.joined(separator: "\n")
        default:
            guard !commentPrefix.isEmpty else { return }
            // The toggle is decided for the whole run, not per line: a
            // partly-commented block becomes fully commented rather than
            // inverting line by line, which is what makes the command its own
            // undo. Blank lines do not vote and are left blank.
            let allCommented = lines.allSatisfy {
                $0.trimmingCharacters(in: .whitespaces).isEmpty
                    || $0.drop(while: { $0 == " " || $0 == "\t" }).hasPrefix(commentPrefix)
            }
            replacement = lines
                .map { allCommented ? uncomment($0) : comment($0) }
                .joined(separator: "\n")
        }

        textView.textStorage.replaceCharacters(in: block, with: replacement)
        textView.selectedRange = NSRange(location: block.location,
                                         length: (replacement as NSString).length)
        // Programmatic storage edits do not call the delegate.
        textViewDidChange(textView)
    }

    /// Removes one indent's worth of leading white space, and leaves a line
    /// that has none alone rather than eating a glyph. A leading tab goes
    /// whatever the unit is, because a buffer mixes them: a tab-indented file
    /// outdented by a four-space unit would otherwise lose nothing at all.
    private func outdent(_ line: String) -> String {
        if line.hasPrefix(indentUnit) { return String(line.dropFirst(indentUnit.count)) }
        if line.hasPrefix("\t") { return String(line.dropFirst()) }
        var out = Substring(line)
        var removed = 0
        while removed < indentUnit.count, out.first == " " {
            out = out.dropFirst()
            removed += 1
        }
        return String(out)
    }

    /// Inserts the prefix at the start of the line's *indentation*, not at
    /// column zero, so a commented block keeps the shape of the code it came
    /// from. A blank line stays blank: a file of "// " on its empty lines is
    /// trailing white space a formatter strips on the next save.
    private func comment(_ line: String) -> String {
        guard !line.trimmingCharacters(in: .whitespaces).isEmpty else { return line }
        let indent = line.prefix { $0 == " " || $0 == "\t" }
        return indent + commentPrefix + " " + line.dropFirst(indent.count)
    }

    /// Removes the first prefix and the single space that usually follows it.
    /// One space, not all of them: "//     aligned" is a comment whose own
    /// indentation is part of what it says.
    private func uncomment(_ line: String) -> String {
        guard let at = line.range(of: commentPrefix) else { return line }
        var after = at.upperBound
        if after < line.endIndex, line[after] == " " { after = line.index(after: after) }
        return String(line[line.startIndex..<at.lowerBound]) + String(line[after...])
    }

    // MARK: Small shared pieces

    /// The indentation of the line the caret is in, which Return copies onto
    /// the new line. Read from the text *before* the caret so that splitting a
    /// line mid-way still continues at that line's indent.
    private func leadingWhitespace(before location: Int, in textView: UITextView) -> String {
        let ns = (textView.text ?? "") as NSString
        let head = ns.substring(to: min(location, ns.length))
        let line = head.components(separatedBy: "\n").last ?? ""
        return String(line.prefix { $0 == " " || $0 == "\t" })
    }

    /// Replaces a range with text and leaves the caret after it — the
    /// platform's own typing behaviour, re-implemented for the two keys whose
    /// default had to be refused.
    private func insert(_ replacement: String, at range: NSRange, in textView: UITextView) {
        textView.textStorage.replaceCharacters(in: range, with: replacement)
        let caret = range.location + (replacement as NSString).length
        textView.selectedRange = NSRange(location: caret, length: 0)
        textViewDidChange(textView)
    }

    private func utf8Offset(in text: String, utf16Offset: Int) -> Int {
        let clamped = max(0, min(utf16Offset, (text as NSString).length))
        let index = String.Index(utf16Offset: clamped, in: text)
        return text.utf8.distance(from: text.utf8.startIndex, to: index)
    }
}

#else

/// The non-iOS build. Renderer.swift's dispatch has one arm for this node type
/// on every platform, so the type has to exist on every platform — and the
/// macOS typecheck that runs in ios/verify compiles these files without UIKit.
///
/// A box and no editor, which is also the honest rendering: an NSTextView is a
/// different control with a different responder chain and a different input
/// method, and inventing one to satisfy a typecheck would be shipping an
/// unexercised editor. GrMobMapView takes the same shape for the same reason.
struct GrMobCodeEditor: View {
    let node: GrMobNode
    let grow: GrMobGrow

    var body: some View {
        Color.clear.grMobBox(node.style, grow: grow)
    }
}

#endif
