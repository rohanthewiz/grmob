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
///  1. **Echo guard.** GrMobTextField's TextEditLedger: the buffer is the
///     host's while it is first responder and Go's otherwise, Go's edit stamps
///     tell our own edit coming back from a rewrite, and typing in flight when
///     a rewrite lands is replayed onto it with the caret where it was
///     (`rebaseCaret`).
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

    /// The editor's size when its parent does not bound the height: the text's
    /// own height, so the viewport equals the content and nothing scrolls.
    ///
    /// A scroll-enabled UITextView has no intrinsic height — scrolling is the
    /// reason it has none — so without this SwiftUI's default for a
    /// representable asked for an ideal size gave the editor zero height. A
    /// comps.CodeEditor with no Height inside a scrolling screen (every code
    /// block in every tutorial lesson) drew as its padding and background
    /// alone: a dark bar with no text.
    ///
    ///     proposed height      reported
    ///     ---------------      --------
    ///     nil or infinite      content height at the width on offer
    ///     finite               nil — the default, which fills the proposal
    ///                          (a real viewport, e.g. core.Height("240px"))
    ///
    /// The same contract as Compose's verticalScrollWhenBounded in
    /// Renderer.kt, which fixed the Android form of this (a crash rather than
    /// an empty box), and as the DOM, where an overflow:auto box of auto
    /// height is as tall as its content.
    ///
    /// Width follows the proposal when there is one: text never wraps here
    /// (see GrMobCodeEditorView.init), so a line wider than the box scrolls
    /// sideways rather than widening the editor. Only an ideal-size query with
    /// no width gets the widest line plus the gutter.
    func sizeThatFits(_ proposal: ProposedViewSize, uiView: GrMobCodeEditorView,
                      context: Context) -> CGSize? {
        if let height = proposal.height, height.isFinite { return nil }
        let offeredWidth = proposal.width.flatMap { $0.isFinite ? $0 : nil }
        let content = uiView.contentSize()
        return CGSize(width: offeredWidth ?? content.width, height: content.height)
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

        coordinator.applyValue(node.stringProp("value"),
                               editSeq: node.intProp("editSeq"),
                               editEpoch: node.intProp("editEpoch"),
                               stamped: node.props["editEpoch"] != nil)
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
///
///     ┌ GrMobCodeEditorView ─────────────────────────────┐
///     │┌ gutter ┐┌ sideways (UIScrollView, x only) ──────┐│
///     ││ 1      ││┌ textView (UITextView, y only) ───────┼┼──┐
///     ││ 2      │││ as wide as the longest line          ││  │
///     ││ 3      │││                                      ││  │
///     │└────────┘│└──────────────────────────────────────┼┼──┘
///     │          └───────────────────────────────────────┘│
///     └──────────────────────────────────────────────────┘
///
/// Two scroll views, one per axis, because UITextView scrolls vertically only.
/// Given a contentSize wider than itself it does move sideways, but it draws
/// text only inside its own width — measured on the iOS 26 simulator, a swipe
/// revealed an empty band where the rest of each line should be. So the text
/// view is made as wide as the longest line, which it draws whole, and the
/// sideways scroll view clips it to the box. The gutter sits outside that
/// scroll view, so the numbers stay put while the code moves left.
final class GrMobCodeEditorView: UIView {
    /// TextKit 1, asked for by name. A plain `UITextView()` has been TextKit 2
    /// since iOS 16, and TextKit 2 wraps to the view's width whatever the
    /// text container is told: the unbounded container size and
    /// `widthTracksTextView = false` below were ignored, so long lines wrapped
    /// to column zero on device, and the height sizeThatFits measured (the
    /// unwrapped lines) cut the wrapped buffer's last lines off. TextKit 1
    /// honours the container, which is the whole no-wrap design.
    let textView = GrMobUnwrappedTextView(usingTextLayoutManager: false)
    private let sideways = UIScrollView()
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

    /// Every line's paragraph style: left aligned, with a left-to-right base
    /// direction. `.natural` would take the base from the first strong
    /// character, and a line with none ("}", an indent, a blank) would fall
    /// back to the app language's direction.
    static let leftToRight: NSParagraphStyle = {
        let style = NSMutableParagraphStyle()
        style.alignment = .left
        style.baseWritingDirection = .leftToRight
        return style
    }()
    private var ink = UIColor.label
    private var lineCount = 1

    /// The per-line paragraph style with the tab stops a code buffer wants: a
    /// literal tab advances to the next multiple of `tabStops` columns.
    ///
    /// UIKit's default is a fixed 28pt interval, which at 13pt monospace is
    /// about 3.6 columns: a tab-indented Go file drew its indents a fraction of
    /// a column short of four and its tab-aligned comments out of line. The
    /// width is measured off the font rather than assumed, so a Go-set font
    /// size moves it with the text. `tabStops = []` clears UIKit's twelve
    /// preset stops, which would otherwise win over `defaultTabInterval` for
    /// the first twelve tabs on a line.
    ///
    /// Compose has no tab stops at all and expands each tab into spaces
    /// instead (GrMobTabStops in GrMobCodeEditor.kt); the web sets CSS
    /// tab-size. All three draw a tab to the same column.
    func paragraph(tabStops: Int) -> NSParagraphStyle {
        let style = GrMobCodeEditorView.leftToRight.mutableCopy() as! NSMutableParagraphStyle
        let column = ("0" as NSString).size(withAttributes: [.font: font]).width
        style.tabStops = []
        style.defaultTabInterval = column * CGFloat(max(tabStops, 1))
        return style
    }

    override init(frame: CGRect) {
        super.init(frame: frame)

        // Code is left to right in every locale. Under an RTL app language
        // UIKit mirrors a view's subview layout and a text view aligns and
        // bidi-orders each line to the language, so the gutter moved to the
        // right and a line opening with a neutral such as "}" put it at the
        // far end. An editor's columns are what it means, so every view here
        // is forced left to right, and each line's paragraph style (see
        // GrMobCodeEditorView.leftToRight) fixes the base direction the bidi
        // algorithm would otherwise take from the language. Compose pins its
        // layout direction and both DOM targets write dir="ltr" for the same
        // reason.
        for view: UIView in [self, sideways, textView, gutter] {
            view.semanticContentAttribute = .forceLeftToRight
        }
        textView.textAlignment = .left

        textView.backgroundColor = .clear
        textView.textContainerInset = .zero
        textView.textContainer.lineFragmentPadding = 0
        // No wrapping: a code line is one line, and a wrapped one restarts at
        // column zero, which reads as a new statement at the outermost indent.
        // So the container is given unbounded width, the text view is sized to
        // the longest line, and `sideways` scrolls it (see the type doc).
        textView.textContainer.lineBreakMode = .byClipping
        textView.textContainer.widthTracksTextView = false
        textView.textContainer.size = CGSize(width: CGFloat.greatestFiniteMagnitude,
                                             height: CGFloat.greatestFiniteMagnitude)
        textView.isScrollEnabled = true
        textView.alwaysBounceHorizontal = false
        textView.isDirectionalLockEnabled = true

        sideways.showsVerticalScrollIndicator = false
        sideways.alwaysBounceVertical = false
        sideways.alwaysBounceHorizontal = false
        sideways.isDirectionalLockEnabled = true
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

        sideways.addSubview(textView)
        addSubview(sideways)
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

    /// The whole buffer's extent, gutter included: the widest line and every
    /// line's height. Measured through the text view's own layout rather than
    /// as lineCount × line height, so the syntax rows' attributes (which may
    /// change a run's font) are what is measured.
    func contentSize() -> CGSize {
        let used = textExtent()
        // An empty buffer has no line fragment, so usedRect is zero tall; one
        // line of the editor's font keeps an empty editor from vanishing.
        let height = max(used.height, font.lineHeight)
        return CGSize(width: ceil(used.width + gutterWidth), height: ceil(height))
    }

    /// The laid-out text's extent with the container unbounded: the longest
    /// line's width and the sum of the lines' heights.
    private func textExtent() -> CGSize {
        textView.holdContainerUnbounded()
        let layout = textView.layoutManager
        layout.ensureLayout(for: textView.textContainer)
        return layout.usedRect(for: textView.textContainer).size
    }

    /// Scrolls `sideways` so the caret is on screen, a column of room either
    /// side. UITextView keeps its caret visible vertically by itself; it
    /// cannot horizontally, because as far as it knows it is already showing
    /// its whole width.
    ///
    /// Laid out first: a keystroke that lengthens the line reaches here before
    /// the layout pass that widens the text view, and scrolling towards a
    /// caret beyond the old content width would stop short of it.
    func revealCaret() {
        guard let range = textView.selectedTextRange else { return }
        layoutIfNeeded()
        let caret = textView.caretRect(for: range.end)
        guard caret.minX.isFinite else { return }
        let room = ("0" as NSString).size(withAttributes: [.font: font]).width
        sideways.scrollRectToVisible(
            CGRect(x: caret.minX - room, y: 0, width: caret.width + 2 * room, height: 1),
            animated: false)
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
        let viewport = CGSize(width: max(0, bounds.width - inset), height: bounds.height)
        sideways.frame = CGRect(x: inset, y: 0, width: viewport.width, height: viewport.height)
        // The text view is at least the viewport wide, so a short buffer still
        // takes taps across the whole box, and otherwise as wide as its longest
        // line, which is what lets it draw that line whole.
        let width = max(viewport.width, ceil(textExtent().width))
        textView.frame = CGRect(x: 0, y: 0, width: width, height: viewport.height)
        // After the frame, not before: assigning it is one of the UIKit paths
        // that narrows the container. See GrMobUnwrappedTextView.
        textView.holdContainerUnbounded()
        // Exactly the viewport tall, so `sideways` never scrolls vertically;
        // that axis belongs to the text view.
        sideways.contentSize = CGSize(width: width, height: viewport.height)
        // The gutter is pinned to the left of the box and offset upward by
        // however far the buffer has scrolled, so number N stays beside line N.
        // Its height is the whole text, not the visible box, which is what lets
        // the offset carry it off the top.
        //
        // Exactly its own text's height, and never the box's. A UILabel centres
        // its lines vertically in a frame taller than they are, and this one
        // used to be at least the box tall: lesson 4.13's four lines in a 170pt
        // editor drew "1" beside the fourth line, with 2 to 4 hanging below the
        // buffer. Fitted, the label's first line is at its top, which is where
        // the text view's is (its insets and line padding are zero).
        let numbers = gutter.sizeThatFits(CGSize(width: inset, height: .greatestFiniteMagnitude))
        gutter.frame = CGRect(x: 0, y: -textView.contentOffset.y,
                              width: inset,
                              height: ceil(numbers.height))
        // Room for the digits, so the gutter's own trailing column is a gap
        // rather than a number touching the code. The width has two columns
        // beyond the digits (gutterWidth); the label gives up the last one,
        // and its right-aligned numbers end a column short of the buffer. It
        // was `insetBy(dx: 0, dy: 0)`, which gives up nothing, and the numbers
        // sat against the first character of every line.
        let column = ("0" as NSString).size(withAttributes: [.font: font]).width
        gutter.frame.size.width = max(0, inset - column)
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

    /// What this editor has sent Go and not yet heard back about. See
    /// GrMobTextField for the full argument; the bookkeeping is identical, and
    /// so is the class (GrMobTextEdits.swift).
    private var ledger = TextEditLedger()
    /// The (value, editSeq, editEpoch) last read. updateUIView runs for every
    /// reason SwiftUI has, and only a change to one of the three is news: an
    /// echo moves only the ack, and a refused edit moves the ack and the epoch
    /// and leaves the value where it was.
    private var lastStamp: (value: String, seq: Int, epoch: Int)?
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

    func applyValue(_ value: String, editSeq: Int, editEpoch: Int, stamped: Bool) {
        // Nothing lands mid-composition. The stamp is recorded only past this
        // guard, so the next pass after the composition ends reads it.
        guard let textView = view?.textView, textView.markedTextRange == nil else { return }
        if let last = lastStamp, last == (value, editSeq, editEpoch) { return }
        lastStamp = (value, editSeq, editEpoch)

        guard textView.isFirstResponder else {
            // Go-owned while blurred; anything in flight died with the session.
            ledger.reset(value, epoch: editEpoch)
            if textView.text != value { textView.text = value }
            return
        }
        // Go's edit stamps say whether this is our own typing coming back or
        // Go speaking for itself (core/text_edit.go). It used to be a queue of
        // sent values, which let keystrokes already in flight when a rewrite
        // landed be applied by Go as if they were new; see GrMobTextField.
        let local = textView.text ?? ""
        guard let next = ledger.upstream(value, ack: editSeq, goEpoch: editEpoch, local: local,
                                         stamped: stamped) else {
            return
        }
        // A rewrite, which wins even mid-typing. The caret follows the typing
        // that was replayed onto it; with nothing replayed it lands at the end
        // of Go's text, which is where assigning `text` always put it.
        let caret = !stamped ? (next as NSString).length
            : rebaseCaret(basis: ledger.lastBasis, local: local, rewrite: value,
                          caret: textView.selectedRange.location + textView.selectedRange.length)
        if textView.text != next { textView.text = next }
        textView.selectedRange = NSRange(location: caret, length: 0)
        // Typing Go has not seen yet, replayed onto its rewrite.
        if next != value { send(next) }
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
            // tabSize columns per tab, or 4 for a literal-tab indent
            // (tabSize 0) — the width the web and Compose give the same buffer.
            .paragraphStyle: editor.paragraph(tabStops: tabSize > 0 ? tabSize : 4),
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
        view?.refreshGutter()
        send(textView.text ?? "")
    }

    /// Every edit leaves by this one path, so the ledger records exactly what
    /// Go was sent and under which epoch.
    private func send(_ value: String) {
        guard !onChange.isEmpty, let runtime else { return }
        ledger.sent(runtime.textEdited(onChange, value, epoch: ledger.epoch), value)
    }

    /// The selection, as byte offsets into the UTF-8 value.
    ///
    /// UIKit counts UTF-16 code units and core.OnSelectionChange promises
    /// bytes, so the conversion is this method's whole job beyond the dispatch.
    /// It is the same conversion the other three hosts make from their own
    /// unit, which is what lets app code see one number.
    func textViewDidChangeSelection(_ textView: UITextView) {
        // Before the guard: keeping the caret on screen is the host's job
        // whether or not Go listens for the selection.
        view?.revealCaret()
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

/// The editor's UITextView, holding its text container at unbounded width.
///
/// The no-wrap design (see GrMobCodeEditorView) is a text container wider
/// than any line, so each code line lays out as one line fragment. Configuring
/// the container once is not enough. Measured on the iOS 26 simulator, with
/// `widthTracksTextView = false`:
///
///     pass                              container width
///     ----                              ---------------
///     init                              unbounded
///     after the editor assigns frame    310  (the view's width)
///     inside this view's layout         310  → lines wrapped to column zero
///
/// So the container is put back on both sides of UIKit's own layout pass, and
/// after every frame assignment (GrMobCodeEditorView.layoutSubviews). Each is
/// a size comparison when nothing has changed.
final class GrMobUnwrappedTextView: UITextView {
    override func layoutSubviews() {
        holdContainerUnbounded()
        super.layoutSubviews()
        holdContainerUnbounded()
    }

    func holdContainerUnbounded() {
        let unbounded = CGSize(width: CGFloat.greatestFiniteMagnitude,
                               height: CGFloat.greatestFiniteMagnitude)
        if textContainer.size != unbounded {
            textContainer.size = unbounded
        }
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
