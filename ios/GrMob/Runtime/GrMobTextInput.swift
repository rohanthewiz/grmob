#if canImport(UIKit)

import SwiftUI
import UIKit

/// core.Input, InputPassword, NumericInput and TextArea: a UITextField, or a
/// UITextView for the multiline one, hosted in SwiftUI.
///
/// # The controlled-input compromise
///
/// Go owns the value, but the keyboard needs its keystrokes shown at once, and
/// the Go round trip is asynchronous. So the field is the user's *while
/// focused* (every edit is sent upstream, and a late echo never snaps the caret
/// back) and Go's when not (an upstream change such as a validation rewrite or
/// a restored state lands the moment the user is not typing).
///
/// The one upstream change that must land mid-focus is a deliberate rewrite:
/// Go clearing the draft after a submit, or a transform normalising each key.
/// Echoes and rewrites are told apart by bookkeeping, not heuristics: every
/// edit goes to Go with a sequence number and the rewrite epoch this field has
/// adopted, and Go stamps the node with the last edit it applied and its own
/// rewrite count. A higher epoch is a rewrite and wins even while focused, and
/// the keys typed since the edit Go read are replayed onto it. See
/// TextEditLedger (GrMobTextEdits.swift) for the rule and core/text_edit.go
/// for the protocol.
///
/// # Why UIKit and not SwiftUI's TextField
///
/// This was a SwiftUI TextField over a `Binding<String>`, and the binding is a
/// second copy of the text. SwiftUI reconciles it with the UITextField it draws
/// on its own schedule, and a key that reaches UIKit between a rewrite being
/// assigned to the binding and SwiftUI pushing it down is overwritten without
/// the binding ever hearing of it. The ledger cannot replay a key it never saw.
///
/// Measured on the iOS 26.5 simulator, lesson 2.3 with UPPERCASE on, one
/// XCUITest `typeText("abcdefghijklmnopqrst")`, logging the binding's setter
/// and the stamps:
///
///	  stamp ABCDEFG (a rewrite)   local "ABCDEFg" → text = "ABCDEFG"
///	  set   ABCDEFGl                        ← h, i, j, k never reported
///	  ...
///	  final "ABCDEFGLMNOPQRST"
///
/// and "hello world" typed the same way came out "HELLO WOD". Every key under
/// a transform is answered by a rewrite, so the window is open on every key.
///
/// It is the iOS form of what Android's value-and-callback BasicTextField did
/// (Renderer.kt keeps the table), and the fix is the same one: one buffer.
/// Here that buffer is the UITextField's own text. A rewrite reads it and
/// writes it in one turn of the main thread, which is also the thread UIKit
/// delivers keys on. The keyboard can still deliver queued keys *inside* the
/// write, and GrMobTextInputCoordinator.write says how those are kept.
struct GrMobTextField: View {
    let node: GrMobNode
    let grow: GrMobGrow
    var password = false
    var numeric = false
    var multiline = false

    @Environment(\.grMobRuntime) private var runtime

    var body: some View {
        Group {
            if multiline {
                GrMobTextAreaRepresentable(node: node, runtime: runtime)
            } else {
                GrMobLineFieldRepresentable(node: node, runtime: runtime,
                                            password: password, numeric: numeric)
            }
        }
        .grMobBox(Self.boxStyle(node.style), grow: grow)
    }

    /// The style grMobBox is given: the node's, without its accessible name
    /// and hint, which go on the UIKit field itself (see applyAccessibility).
    ///
    /// grMobBox states a label with `.accessibilityElement(children:)` and
    /// `.accessibilityLabel`, which on a SwiftUI control names the control.
    /// On a hosted UIKit view it makes a new SwiftUI element around it:
    /// XCUITest on the iOS 26.5 simulator read lesson 5.7's field as an
    /// "Other" named "One-time code, 0 of 6 entered" holding an unnamed
    /// TextField. VoiceOver would meet a name with no field in it, then a
    /// field with no name. On the field, the name is the text field's own.
    static func boxStyle(_ s: GrMobStyle?) -> GrMobStyle? {
        guard var t = s else { return nil }
        t.accessibilityLabel = ""
        t.accessibilityHint = ""
        return t
    }
}

/// The node's accessible name and hint, on the UIKit view that is the control
/// (GrMobTextField.boxStyle says why not through grMobBox). nil rather than ""
/// for an unstated one, so UIKit's own reading (the placeholder, for a field
/// with no name) is left alone.
func applyAccessibility(_ view: UIView, _ s: GrMobStyle?) {
    let label = s?.accessibilityLabel ?? ""
    let hint = s?.accessibilityHint ?? ""
    view.accessibilityLabel = label.isEmpty ? nil : label
    view.accessibilityHint = hint.isEmpty ? nil : hint
}

// MARK: - The two representables

/// One line: a UITextField.
private struct GrMobLineFieldRepresentable: UIViewRepresentable {
    let node: GrMobNode
    let runtime: GrMobRuntime?
    let password: Bool
    let numeric: Bool

    func makeCoordinator() -> GrMobTextInputCoordinator { GrMobTextInputCoordinator() }

    func makeUIView(context: Context) -> UITextField {
        let field = UITextField()
        field.borderStyle = .none
        field.delegate = context.coordinator
        field.addTarget(context.coordinator, action: #selector(GrMobTextInputCoordinator.editingChanged),
                        for: .editingChanged)
        // A field is as wide as it is laid out, not as wide as its text:
        // SwiftUI proposes the width and sizeThatFits accepts it.
        field.setContentHuggingPriority(.defaultLow, for: .horizontal)
        field.setContentCompressionResistancePriority(.defaultLow, for: .horizontal)
        context.coordinator.input = field
        return field
    }

    /// The size SwiftUI's TextField gave: the width on offer, or, asked for an
    /// ideal size (a flex Row measuring its children's bases), the width of the
    /// text or the placeholder, whichever is wider. The height is one line of
    /// the field's font.
    func sizeThatFits(_ proposal: ProposedViewSize, uiView: UITextField, context: Context) -> CGSize? {
        let intrinsic = uiView.intrinsicContentSize
        let placeholder = (uiView.attributedPlaceholder?.size().width).map { ceil($0) } ?? 0
        let ideal = max(intrinsic.width, placeholder)
        let width = proposal.width.flatMap { $0.isFinite ? $0 : nil } ?? ideal
        return CGSize(width: width, height: ceil(uiView.font?.lineHeight ?? intrinsic.height))
    }

    /// The coordinator's settings first, because the value and the focus
    /// command below call back into it; the value before the focus command,
    /// so the responder arrives at text this pass has already settled.
    func updateUIView(_ field: UITextField, context: Context) {
        let coordinator = context.coordinator
        coordinator.configure(node: node, runtime: runtime)

        let look = GrMobTextLook(node.style)
        field.font = look.font
        field.textColor = look.color
        field.textAlignment = look.alignment(in: field)
        field.attributedPlaceholder = NSAttributedString(
            string: node.stringProp("placeholder"),
            attributes: [.font: look.font, .foregroundColor: UIColor.placeholderText])

        field.isSecureTextEntry = password
        field.keyboardType = numeric ? .decimalPad : grMobKeyboardType(node.stringProp("keyboard"))
        // A code the system can fill from a text message or a password
        // manager: iOS offers the SMS code above the keyboard for a field
        // that says it holds one. A digits keyboard on a text field is what
        // comps.PINInput asks for, and it is the one-time code's field.
        field.textContentType = node.stringProp("keyboard") == "digits" && !password ? .oneTimeCode : nil
        field.returnKeyType = coordinator.returnKey
        // `.disabled(true)` in grMobBox reaches a representable as the
        // environment's isEnabled, and a UITextField has to be told.
        field.isEnabled = context.environment.isEnabled
        applyAccessibility(field, node.style)

        coordinator.applyValue(node.stringProp("value"), editSeq: node.intProp("editSeq"),
                               editEpoch: node.intProp("editEpoch"),
                               stamped: node.props["editEpoch"] != nil)
        coordinator.applyFocus(epoch: node.intProp("focusEpoch"), action: node.stringProp("focusAction"))
    }
}

/// Several lines: a UITextView, `rows` lines tall, scrolling inside beyond.
///
/// `rows` lines reserved and no more, which is what the SwiftUI field it
/// replaced drew with `.lineLimit(rows, reservesSpace: true)`; 3 when the node
/// says nothing, as before.
private struct GrMobTextAreaRepresentable: UIViewRepresentable {
    let node: GrMobNode
    let runtime: GrMobRuntime?

    func makeCoordinator() -> GrMobTextInputCoordinator { GrMobTextInputCoordinator() }

    func makeUIView(context: Context) -> GrMobTextAreaView {
        let view = GrMobTextAreaView()
        view.delegate = context.coordinator
        context.coordinator.input = view
        return view
    }

    func sizeThatFits(_ proposal: ProposedViewSize, uiView: GrMobTextAreaView,
                      context: Context) -> CGSize? {
        let rows = node.intProp("rows") > 0 ? node.intProp("rows") : 3
        let line = uiView.font?.lineHeight ?? 20
        let spacing = GrMobTextLook(node.style).lineSpacing
        let height = ceil(CGFloat(rows) * line + CGFloat(rows - 1) * spacing)
        let width = proposal.width.flatMap { $0.isFinite ? $0 : nil }
            ?? ceil(max(uiView.placeholder.intrinsicContentSize.width, 80))
        return CGSize(width: width, height: height)
    }

    func updateUIView(_ view: GrMobTextAreaView, context: Context) {
        let coordinator = context.coordinator
        coordinator.configure(node: node, runtime: runtime)

        let look = GrMobTextLook(node.style)
        view.apply(look: look)
        view.placeholder.text = node.stringProp("placeholder")
        view.keyboardType = grMobKeyboardType(node.stringProp("keyboard"))
        // A disabled text view neither edits nor selects; a read-only one
        // would still select, which is not what core.Style.Disabled means.
        view.isEditable = context.environment.isEnabled
        view.isSelectable = context.environment.isEnabled
        applyAccessibility(view, node.style)

        coordinator.applyValue(node.stringProp("value"), editSeq: node.intProp("editSeq"),
                               editEpoch: node.intProp("editEpoch"),
                               stamped: node.props["editEpoch"] != nil)
        coordinator.applyFocus(epoch: node.intProp("focusEpoch"), action: node.stringProp("focusAction"))
    }
}

/// The TextArea's view: a UITextView with the insets SwiftUI's plain field
/// had (none), and a placeholder, which UITextView does not draw.
final class GrMobTextAreaView: UITextView {
    let placeholder = UILabel()
    /// The attributes every run of text is drawn with. A UITextView keeps
    /// them on the text itself, so text assigned from Go gets them here and
    /// typed text gets them through typingAttributes.
    private(set) var attributes: [NSAttributedString.Key: Any] = [:]

    init() {
        super.init(frame: .zero, textContainer: nil)
        backgroundColor = .clear
        textContainerInset = .zero
        textContainer.lineFragmentPadding = 0
        isScrollEnabled = true
        placeholder.textColor = .placeholderText
        placeholder.numberOfLines = 0
        addSubview(placeholder)
    }

    required init?(coder: NSCoder) { fatalError("init(coder:) is not used") }

    func apply(look: GrMobTextLook) {
        let paragraph = NSMutableParagraphStyle()
        paragraph.lineSpacing = look.lineSpacing
        paragraph.alignment = look.alignment(in: self)
        let next: [NSAttributedString.Key: Any] = [
            .font: look.font, .foregroundColor: look.color, .paragraphStyle: paragraph,
        ]
        font = look.font
        placeholder.font = look.font
        placeholder.textAlignment = paragraph.alignment
        guard !NSDictionary(dictionary: next).isEqual(to: attributes) else { return }
        attributes = next
        typingAttributes = next
        // Restyle what is already there, keeping the selection.
        let selection = selectedRange
        attributedText = NSAttributedString(string: text ?? "", attributes: next)
        selectedRange = selection
    }

    /// Text from Go, in the field's attributes. Assigning `text` alone would
    /// draw it in the view's defaults until the next keystroke.
    func setText(_ value: String) {
        attributedText = NSAttributedString(string: value, attributes: attributes)
        refreshPlaceholder()
    }

    func refreshPlaceholder() {
        placeholder.isHidden = !(text ?? "").isEmpty
    }

    override func layoutSubviews() {
        super.layoutSubviews()
        let width = bounds.width
        let size = placeholder.sizeThatFits(CGSize(width: width, height: .greatestFiniteMagnitude))
        placeholder.frame = CGRect(x: 0, y: 0, width: width, height: size.height)
    }
}

/// core.Keyboard's kinds as UIKit keyboards. numberPad is digits only and has
/// no return key, which is right for a code and for nothing that submits on
/// return; a caller asking for digits on a submitting field gets the pad
/// anyway, as it asked.
func grMobKeyboardType(_ kind: String) -> UIKeyboardType {
    switch kind {
    case "digits": return .numberPad
    case "decimal": return .decimalPad
    case "phone": return .phonePad
    case "email": return .emailAddress
    case "url": return .URL
    default: return .default
    }
}

// MARK: - The look

/// The text style a field draws in, from the node's GrMobStyle: what
/// grMobTextStyle gives a SwiftUI Text, spelled for UIKit, which reads none of
/// SwiftUI's environment.
struct GrMobTextLook {
    let font: UIFont
    let color: UIColor
    /// SwiftUI has line spacing, not line height, and so does UIKit: the
    /// difference between the requested height and the font size, as
    /// grMobTextStyle approximates it.
    let lineSpacing: CGFloat
    private let align: String

    init(_ s: GrMobStyle?) {
        let size = (s?.fontSize ?? 0) > 0 ? s!.fontSize : 17
        // The same weight ladder the drawn Text and the min-content floor use,
        // in its CoreText half; UIFont.Weight's raw values are that scale.
        let weight = UIFont.Weight(rawValue: grMobFontWeightPair(s?.fontWeight ?? 0).1)
        font = UIFont.systemFont(ofSize: size, weight: weight)
        color = s?.textColor.map { UIColor($0) } ?? .label
        lineSpacing = (s?.lineHeight ?? 0) > 0 ? max(CGFloat(s!.lineHeight) - size, 0) : 0
        align = s?.align ?? ""
    }

    /// core.TextAlignments as NSTextAlignment. "end" follows the view's own
    /// layout direction, as SwiftUI's `.trailing` did; "justify" is drawn
    /// leading, as grMobTextAlignment draws it.
    func alignment(in view: UIView) -> NSTextAlignment {
        switch align {
        case "center": return .center
        case "end": return view.effectiveUserInterfaceLayoutDirection == .rightToLeft ? .left : .right
        default: return .natural
        }
    }
}

// MARK: - The coordinator: one field's half of the text-edit protocol

/// Everything about one field that is a decision rather than a view, shared
/// by both representables: the ledger, the focus edges, submit, and focus
/// commands.
final class GrMobTextInputCoordinator: NSObject, UITextFieldDelegate, UITextViewDelegate {
    /// The UITextField or UITextView this coordinator drives. UITextInput is
    /// the protocol both share for the caret and the composition.
    weak var input: (UIView & UITextInput)?

    private var runtime: GrMobRuntime?
    private var onChange = ""
    private var onSubmit = ""
    private var onFocus = ""
    private var onBlur = ""
    private var imeAction = ""

    /// What this field has sent Go and not yet heard back about.
    private var ledger = TextEditLedger()
    /// The (value, editSeq, editEpoch) last read. updateUIView runs for every
    /// reason SwiftUI has, and only a change to one of the three is news: an
    /// echo moves only the ack, and a refused edit moves the ack and the epoch
    /// and leaves the value where it was.
    private var lastStamp: (value: String, seq: Int, epoch: Int)?
    /// Go's value and epoch as of the last pass, which is what a field
    /// taking focus starts from.
    private var upstream = ""
    private var upstreamEpoch = 0
    /// The focus-command memory; see GrMobEditorFocus.swift.
    private var focus = GrMobEditorFocus()
    /// Up while `write` puts Go's text into the field.
    private var applying = false

    func configure(node: GrMobNode, runtime: GrMobRuntime?) {
        self.runtime = runtime
        onChange = node.stringProp("onChange")
        onSubmit = node.stringProp("onSubmit")
        onFocus = node.stringProp("onFocus")
        onBlur = node.stringProp("onBlur")
        // Decided in Go from core.UseFocusOrder: "next" on every field of a
        // declared order but the last. Go also wired onSubmit to advance the
        // focus, so this only chooses the key's label.
        imeAction = node.stringProp("imeAction")
    }

    /// The keyboard's action key: "next" for a field with somewhere to go
    /// (core.UseFocusOrder), "done" for one that acts on return, the plain
    /// return key for a field that does neither. Next is tested first because
    /// it is the more specific claim: Go only stamps it on a field whose
    /// onSubmit it wired itself.
    var returnKey: UIReturnKeyType {
        imeAction == "next" ? .next : (onSubmit.isEmpty ? .default : .done)
    }

    // MARK: The text, whichever view holds it

    private var text: String {
        (input as? UITextField)?.text ?? (input as? UITextView)?.text ?? ""
    }

    /// Assigning text sends no editingChanged and no textViewDidChange, so Go's
    /// own value is never reported back to it as typing.
    private func setText(_ value: String) {
        if let field = input as? UITextField {
            field.text = value
        } else if let area = input as? GrMobTextAreaView {
            area.setText(value)
        }
    }

    // MARK: Go -> the field

    func applyValue(_ value: String, editSeq: Int, editEpoch: Int, stamped: Bool) {
        upstream = value
        upstreamEpoch = editEpoch
        // Nothing lands mid-composition (a CJK input method's marked text).
        // The stamp is recorded only past this guard, so the pass after the
        // composition commits reads it.
        guard let input, input.markedTextRange == nil else { return }
        if let last = lastStamp, last == (value, editSeq, editEpoch) { return }
        lastStamp = (value, editSeq, editEpoch)

        guard input.isFirstResponder else {
            // Go's while blurred; anything in flight died with the focus.
            ledger.reset(value, epoch: editEpoch)
            if text != value { setText(value) }
            return
        }
        // Read and written in this one call, on the main thread, which is
        // where UIKit delivers keys: every key typed so far is in `local`,
        // and none is lost in a second copy of the text, which is the reason
        // this field is UIKit (see GrMobTextField). Keys the keyboard delivers
        // *during* the write are the one arrival left, and `write` and the
        // send after it are arranged for them.
        let local = text
        guard let next = ledger.upstream(value, ack: editSeq, goEpoch: editEpoch, local: local,
                                         stamped: stamped) else {
            return
        }
        // A rewrite, which wins even mid-typing: written as the smallest
        // replacement that turns the field's text into `next`, so UIKit moves
        // the caret as it does for any edit (see `write`).
        write(next)
        // What the field holds now is Go's text, plus any typing replayed onto
        // it, plus any key the keyboard delivered during the write. Whatever
        // of that Go has not seen goes up as an edit at the new epoch.
        let now = text
        if now != value { send(now) }
    }

    /// Writes `next` over the field's text as ONE replacement of the span
    /// that differs (extended to the caret when the caret follows it, see
    /// below), through UITextInput, which is how the keyboard itself edits
    /// the document.
    ///
    /// # Why not assign `text`
    ///
    /// Measured on the iOS 26.5 simulator, typing into 2.3's
    /// UPPERCASE field with one XCUITest `typeText`, logging this method:
    ///
    ///	  local "HEllo"  next "HELLo"  caret 5
    ///	  text = "HELLo"; caret = 5
    ///	  read back:     "HELLo w"     caret 5     ← " w" arrived inside the write
    ///
    /// Assigning `text` makes UIKit deliver the keys its keyboard session had
    /// queued, there and then, inside the assignment, and without an
    /// editingChanged for them. They went in at the end, and the caret this
    /// method then set from its own arithmetic put the rest of the typing in
    /// front of them: "hello world" came out "HELLO ORLDW".
    ///
    /// A replacement of the differing span is a small edit the keyboard
    /// session can follow, and it tells this method where the caret belongs:
    /// an edit before the caret shifts it, an edit after it leaves it.
    /// "HEllo" to "HELLo" replaces "ll" with "LL" and the caret stays at 5. A
    /// committed TagInput draft replaces "beta," with "" and the caret comes
    /// back with the "ga" typed after it. A key delivered inside the write is
    /// found by reading the text back, and the caret goes after it.
    ///
    /// `applying` is up for the duration: the replacement is Go's text, not
    /// the user's, and must not leave as an edit through editingChanged.
    /// applyValue sends what the write leaves behind instead.
    private func write(_ next: String) {
        let current = text
        guard current != next else { return }
        guard let input else { return }
        let a = Array(current.utf16), b = Array(next.utf16)
        let n = min(a.count, b.count)
        var prefix = 0
        while prefix < n && a[prefix] == b[prefix] { prefix += 1 }
        if prefix > 0 && UTF16.isLeadSurrogate(a[prefix - 1]) { prefix -= 1 }
        var suffix = 0
        while suffix < n - prefix && a[a.count - 1 - suffix] == b[b.count - 1 - suffix] { suffix += 1 }
        if suffix > 0 && UTF16.isTrailSurrogate(a[a.count - suffix]) { suffix -= 1 }
        // The span replaced, chosen so the caret lands where it belongs with
        // no move after the edit. UITextInput's `replace` leaves the caret at
        // the end of the replacement, and a correction made afterwards is too
        // late: the keyboard inserts a key it was holding at the caret the
        // replacement left. Logged on the simulator, with 2.3's UPPERCASE:
        //
        //	"HELlo", caret 5 → replace "l" with "L" → caret 4 → set to 5
        //	" " arrived between the two, at 4          → "HELL o"
        //
        // So when the caret is at or after the change, which is where it is
        // when Go transforms the key just typed, the span is extended to end
        // at the caret, and the replacement leaves the caret there by itself.
        // A change after the caret or around it is rarer and keeps a
        // correction, by the rule rebaseMapOffset states for the replay: the
        // caret stays put before the change, and inside a change that kept
        // its length (capitalizing words: "hellox| world" → "Hellox| World",
        // one span from the H to the W), and goes to the end of Go's new text
        // only when it sat inside text Go replaced with text of another
        // length, where there is no telling. Sending it to the span's end
        // there too put the next keys after the W: "Hellox Wyzorld".
        let delta = b.count - a.count
        let start = prefix
        var end = a.count - suffix
        let was = caretOffset(input) ?? a.count
        let caret: Int
        if was >= end {
            end = was
            caret = was + delta
        } else if was <= start || delta == 0 {
            caret = was
        } else {
            caret = b.count - suffix
        }
        // [start, end) in the field's text is [start, end + delta) in next:
        // both ends sit in text the two share, the end counted from the back.
        let replacement = String(decoding: b[start..<(end + delta)], as: UTF16.self)

        applying = true
        defer {
            applying = false
            (input as? GrMobTextAreaView)?.refreshPlaceholder()
        }
        if let from = input.position(from: input.beginningOfDocument, offset: start),
           let to = input.position(from: input.beginningOfDocument, offset: end),
           let range = input.textRange(from: from, to: to) {
            input.replace(range, withText: replacement)
        } else {
            setText(next)
        }

        // Keys delivered inside the write are the user's typing, and the
        // caret goes after them, wherever UIKit put them: the span where the
        // text read back differs from `next` is theirs.
        let after = Array(text.utf16)
        if after == b {
            if caretOffset(input) != caret { setCaret(input, caret) }
        } else {
            let m = min(after.count, b.count)
            var suffixBack = 0
            while suffixBack < m && after[after.count - 1 - suffixBack] == b[b.count - 1 - suffixBack] {
                suffixBack += 1
            }
            setCaret(input, after.count - suffixBack)
        }
    }

    /// The end of the selection, in UTF-16 units from the start.
    private func caretOffset(_ input: UIView & UITextInput) -> Int? {
        guard let range = input.selectedTextRange else { return nil }
        return input.offset(from: input.beginningOfDocument, to: range.end)
    }

    private func setCaret(_ input: UIView & UITextInput, _ offset: Int) {
        guard let at = input.position(from: input.beginningOfDocument, offset: offset) else { return }
        input.selectedTextRange = input.textRange(from: at, to: at)
    }

    /// One core.Focus or core.DismissKeyboard.
    ///
    /// Epoch 0 means no command has ever been issued. A field that mounts
    /// while it is already the target takes the caret, because "push a screen
    /// and put the cursor in its search box" issues the command one pass
    /// before the field exists. "blur" is guarded on this field holding the
    /// caret, and "" (another field is the target) does nothing: focusing
    /// that one already takes the caret from here.
    func applyFocus(epoch: Int, action: String) {
        focus.apply(epoch: epoch, action: action, to: input)
    }

    // MARK: The field -> Go

    @objc func editingChanged() {
        edited()
    }

    func textViewDidChange(_ textView: UITextView) {
        (textView as? GrMobTextAreaView)?.refreshPlaceholder()
        edited()
    }

    private func edited() {
        // Go's own text being written (see `write`) is not an edit.
        guard !applying else { return }
        // Marked text is the input method's, not yet the user's: it goes up
        // when it is committed, which is another change.
        guard input?.markedTextRange == nil else { return }
        send(text)
    }

    /// Every edit leaves by this one path, so the ledger records exactly what
    /// Go was sent and under which epoch.
    private func send(_ value: String) {
        guard !onChange.isEmpty, let runtime else { return }
        ledger.sent(runtime.textEdited(onChange, value, epoch: ledger.epoch), value)
    }

    // MARK: Focus edges and submit

    func textFieldDidBeginEditing(_ textField: UITextField) { began() }
    func textFieldDidEndEditing(_ textField: UITextField) { ended() }
    func textViewDidBeginEditing(_ textView: UITextView) { began() }
    func textViewDidEndEditing(_ textView: UITextView) { ended() }

    /// The field becomes the user's: editing starts from Go's value, and the
    /// ledger from Go's epoch. The seeding goes first, because the onFocus
    /// dispatch can land a render before this returns.
    private func began() {
        if text != upstream { setText(upstream) }
        ledger.reset(upstream, epoch: upstreamEpoch)
        if !onFocus.isEmpty { runtime?.click(onFocus) }
    }

    /// Both edges ride the void channel, like onSubmit. None is sent at mount:
    /// a field that never had the caret never loses it.
    private func ended() {
        if !onBlur.isEmpty { runtime?.click(onBlur) }
    }

    /// Return dispatches onSubmit as a plain void event, the same channel as a
    /// Button tap, and gives up the caret, as the SwiftUI field did on submit.
    /// A field that should keep it (TagInput, a focus order's next field) is
    /// focused by Go's own core.Focus, which arrives as a command.
    func textFieldShouldReturn(_ textField: UITextField) -> Bool {
        if !onSubmit.isEmpty { runtime?.click(onSubmit) }
        textField.resignFirstResponder()
        return false
    }
}

#else

import SwiftUI

/// The non-iOS build. Renderer.swift's dispatch names this type on every
/// platform, and the macOS typecheck in ios/verify compiles these files
/// without UIKit. The value, drawn and not editable: an NSTextField is a
/// different control with a different responder chain, and inventing one to
/// satisfy a typecheck would be shipping an unexercised field. The code
/// editor's stub takes the same shape for the same reason.
struct GrMobTextField: View {
    let node: GrMobNode
    let grow: GrMobGrow
    var password = false
    var numeric = false
    var multiline = false

    var body: some View {
        Text(password ? "" : node.stringProp("value"))
            .grMobBox(node.style, grow: grow)
    }
}

#endif
