import SwiftUI

/// core.RichTextEditor: an editable formatted document.
///
/// # Why a UITextView, and what the mapping is
///
/// SwiftUI has no editable styled-text control, so this is the same
/// UIViewRepresentable-over-UITextView that GrMobCodeEditor is — and for a
/// richer reason. `NSTextStorage` is an attributed string: the characters and
/// their formatting are one value, which is exactly the shape richtext.Doc has.
/// The whole of this file is that correspondence, in both directions:
///
///	  richtext.Doc                    NSAttributedString
///	  ──────────────────────────────────────────────────────────────
///	  Run.Bold / .Italic          symbolic font traits
///	  Run.Underline / .Strike     .underlineStyle / .strikethroughStyle
///	  Run.Code                    a monospaced font, plus .grMobCode so the
///	                              reverse mapping does not have to guess
///	  Run.Link                    .link
///	  Block.Kind                  .grMobBlockKind on the paragraph, plus a
///	                              paragraph style, plus a drawn prefix for the
///	                              two list kinds
///
/// The two custom attribute keys are what make the reverse mapping a *reading*
/// rather than an inference. Without `.grMobBlockKind` a heading would have to
/// be recognized by its font size, which is a guess that breaks the moment a
/// theme changes one; without `.grMobPrefix` the bullet drawn in front of a list
/// item would come back as part of the user's text.
///
/// # Commands split two ways, and the split is the design
///
/// A *mark* is an attribute over a range, so it is applied to the textStorage in
/// place — which preserves the caret, costs no rebuild, and gives the empty
/// selection case for free through `typingAttributes`: press bold, type, and the
/// characters arrive bold because UIKit carries the typing attributes forward.
///
/// A *block kind* changes a paragraph's prefix and its indentation, so there is
/// no in-place edit that expresses it. Those go the long way round: read the
/// document out, transform it, rebuild the attributed string, restore the
/// selection by offset. That is the same path the web runtime takes for every
/// command, and it is used here only where an in-place edit would not do.
///
/// # The type-check gap, stated
///
/// Behind `canImport(UIKit)`, which is false for the macOS target `ios/verify`
/// type-checks against — so the fallback below is what that pass compiles, and
/// the UIKit half is covered by the iOS-SDK type-check the same script runs when
/// Xcode is present, plus `mobile/verify/richtext_test.go` and a device pass.
/// GrMobMapView and GrMobCodeEditor carry the identical caveat.
#if canImport(UIKit)

import UIKit

extension NSAttributedString.Key {
    /// The block kind a paragraph is, carried on every character of it so that
    /// the reverse mapping reads rather than infers. See the file doc.
    static let grMobBlockKind = NSAttributedString.Key("grMobBlockKind")
    /// Marks the bullet or number drawn in front of a list item, which is text
    /// as far as the text engine is concerned and is not the user's.
    static let grMobPrefix = NSAttributedString.Key("grMobPrefix")
    /// Inline code. Carried explicitly because the alternative is to recognize
    /// it by its font, and a font is a look rather than a fact.
    static let grMobCode = NSAttributedString.Key("grMobCode")
}

struct GrMobRichTextEditor: View {
    let node: GrMobNode
    let grow: GrMobGrow

    @Environment(\.grMobRuntime) private var runtime

    var body: some View {
        GrMobRichTextRepresentable(node: node, runtime: runtime)
            .grMobBox(node.style, grow: grow)
    }
}

private struct GrMobRichTextRepresentable: UIViewRepresentable {
    let node: GrMobNode
    let runtime: GrMobRuntime?

    func makeCoordinator() -> GrMobRichTextCoordinator { GrMobRichTextCoordinator() }

    func makeUIView(context: Context) -> UITextView {
        let view = UITextView()
        view.delegate = context.coordinator
        view.backgroundColor = .clear
        view.textContainerInset = .zero
        view.textContainer.lineFragmentPadding = 0
        // A document is prose: it wraps, unlike a code buffer.
        view.isScrollEnabled = true
        view.alwaysBounceHorizontal = false
        // Prose wants the corrections a code buffer refuses — a note is words —
        // except the two that change punctuation the writer chose.
        view.autocorrectionType = .yes
        view.spellCheckingType = .yes
        view.smartDashesType = .no
        view.dataDetectorTypes = []
        context.coordinator.view = view
        return view
    }

    func updateUIView(_ view: UITextView, context: Context) {
        let coordinator = context.coordinator
        coordinator.runtime = runtime
        coordinator.onChange = node.stringProp("onChange")
        coordinator.onSelectionChange = node.stringProp("onSelectionChange")
        coordinator.base = GrMobRichStyle(style: node.style)
        // isEditable, not isUserInteractionEnabled: a read-only document is
        // still content the reader is meant to select and copy.
        view.isEditable = !node.boolProp("readOnly")
        coordinator.applyDoc(node.stringProp("doc"))
        coordinator.applyPlaceholder(node.stringProp("placeholder"))
        coordinator.runCommand(epoch: node.intProp("editorEpoch"),
                               command: node.stringProp("editorCommand"))
    }
}

/// The fonts and the ink a document is drawn in, resolved once from the Go
/// style so that every run and every block reads the same base.
struct GrMobRichStyle {
    let size: CGFloat
    let ink: UIColor

    init(style: GrMobStyle?) {
        size = (style?.fontSize ?? 0) > 0 ? style!.fontSize : 17
        ink = UIColor(style?.textColor ?? Color.primary)
    }

    /// The font for one block kind, before any run's own marks are folded in.
    /// Headings step down from the base rather than naming three sizes, so a
    /// theme that changes the body size moves the whole scale with it.
    func font(for kind: String) -> UIFont {
        switch kind {
        case "h1": return .boldSystemFont(ofSize: size * 1.6)
        case "h2": return .boldSystemFont(ofSize: size * 1.35)
        case "h3": return .boldSystemFont(ofSize: size * 1.15)
        case "code": return .monospacedSystemFont(ofSize: size * 0.9, weight: .regular)
        default: return .systemFont(ofSize: size)
        }
    }

    /// The paragraph style for one block kind: the indentation that makes a
    /// list read as a list and a quote read as a quote.
    ///
    /// `headIndent` as well as `firstLineHeadIndent` on the list kinds is what
    /// makes a wrapped list item line up under its own text rather than back
    /// under its bullet — the one thing a drawn prefix cannot do for itself.
    func paragraph(for kind: String) -> NSParagraphStyle {
        let style = NSMutableParagraphStyle()
        style.paragraphSpacing = size * 0.4
        switch kind {
        case "bullet", "numbered":
            style.firstLineHeadIndent = 0
            style.headIndent = size * 1.4
        case "quote":
            style.firstLineHeadIndent = size
            style.headIndent = size
        case "code":
            style.firstLineHeadIndent = size * 0.5
            style.headIndent = size * 0.5
        default:
            break
        }
        return style
    }
}

final class GrMobRichTextCoordinator: NSObject, UITextViewDelegate {
    weak var view: UITextView?
    var runtime: GrMobRuntime?
    var onChange = ""
    var onSelectionChange = ""
    var base = GrMobRichStyle(style: nil)

    /// The document this editor currently holds, in the wire's own shape — a
    /// JSON object as Foundation collections. Kept as the wire shape rather
    /// than as Swift structs so that there is one representation on this host
    /// and it is the one that crosses, exactly as the web runtime does it.
    private var doc: [String: Any] = [:]
    /// The JSON of every document sent upstream and not yet seen come back.
    private var pendingEchoes: [String] = []
    private var lastEpoch: Int?
    private var lastSelection = ""
    private var placeholderLabel: UILabel?

    // MARK: Go -> the document

    func applyDoc(_ json: String) {
        guard let view, view.markedTextRange == nil else { return }

        // The echo guard, over the document's JSON rather than a string of
        // text. Same three arms GrMobTextField's has: an echo is dropped, a
        // rewrite lands even mid-typing, and a blurred editor is Go's outright.
        // The comparison is on the JSON string because Go marshals with a fixed
        // key order, so the same document is always the same bytes.
        if view.isFirstResponder, let echo = pendingEchoes.firstIndex(of: json) {
            pendingEchoes.removeSubrange(...echo)
            return
        }
        pendingEchoes.removeAll()
        guard let parsed = GrMobRichJSON.parse(json) else { return }
        guard GrMobRichJSON.stringify(doc) != json else { return }
        doc = parsed
        rebuild(selecting: nil)
    }

    func applyPlaceholder(_ prompt: String) {
        guard let view else { return }
        let empty = GrMobRichJSON.text(of: doc).isEmpty
        guard !prompt.isEmpty, empty else {
            placeholderLabel?.removeFromSuperview()
            placeholderLabel = nil
            return
        }
        let label = placeholderLabel ?? {
            let made = UILabel()
            made.numberOfLines = 0
            made.isUserInteractionEnabled = false
            view.addSubview(made)
            placeholderLabel = made
            return made
        }()
        label.text = prompt
        label.font = base.font(for: "p")
        label.textColor = base.ink.withAlphaComponent(0.45)
        label.frame = CGRect(x: 0, y: 0, width: view.bounds.width, height: label.font.lineHeight)
    }

    /// Rebuilds the attributed string from the document, optionally restoring a
    /// selection afterwards.
    ///
    /// `attributedText =` rather than an edit, and that is correct *here* and
    /// nowhere else in this file: the document's structure changed, so there is
    /// no range to edit. Every path that can avoid it does — see the file doc on
    /// why marks go through textStorage instead.
    private func rebuild(selecting range: NSRange?) {
        guard let view else { return }
        view.attributedText = GrMobRichMapper.attributed(from: doc, base: base)
        if let range {
            let length = (view.text as NSString).length
            view.selectedRange = NSRange(location: min(range.location, length),
                                         length: min(range.length, max(0, length - range.location)))
        }
        applyPlaceholder(placeholderLabel?.text ?? "")
    }

    // MARK: The document -> Go

    func textViewDidChange(_ textView: UITextView) {
        guard textView.markedTextRange == nil else { return }
        doc = GrMobRichMapper.document(from: textView.attributedText)
        send()
    }

    private func send() {
        let json = GrMobRichJSON.stringify(doc)
        pendingEchoes.append(json)
        if !onChange.isEmpty { runtime?.textChanged(onChange, json) }
        applyPlaceholder(placeholderLabel?.text ?? "")
    }

    /// The selection report: where the caret is, and what formatting is active
    /// there — which is what a toolbar draws its pressed state from.
    ///
    /// The offsets are UTF-8 byte offsets into the document's plain text, which
    /// is core.RichSelection's promise and the one coordinate system all four
    /// hosts can produce.
    func textViewDidChangeSelection(_ textView: UITextView) {
        guard !onSelectionChange.isEmpty else { return }
        let attributed = textView.attributedText ?? NSAttributedString()
        let range = textView.selectedRange
        // A bare caret reports the marks to its left, which is what the next
        // character typed there would inherit — the same rule the web runtime's
        // richMarksAt applies.
        var probe = range
        if probe.length == 0 { probe = NSRange(location: max(0, range.location - 1), length: min(1, range.location)) }

        var marks: [String] = []
        var link = ""
        var block = "p"
        if attributed.length > 0, probe.location < attributed.length {
            let attrs = attributed.attributes(at: min(probe.location, attributed.length - 1), effectiveRange: nil)
            if let traits = (attrs[.font] as? UIFont)?.fontDescriptor.symbolicTraits {
                if traits.contains(.traitBold) { marks.append("bold") }
                if traits.contains(.traitItalic) { marks.append("italic") }
            }
            if attrs[.underlineStyle] != nil { marks.append("underline") }
            if attrs[.strikethroughStyle] != nil { marks.append("strike") }
            if attrs[.grMobCode] != nil { marks.append("code") }
            if let url = attrs[.link] { link = "\(url)" }
            block = (attrs[.grMobBlockKind] as? String) ?? "p"
        }
        let text = attributed.string as NSString
        let payload = GrMobRichJSON.stringify([
            "s": GrMobRichMapper.utf8Offset(in: text, utf16Offset: range.location),
            "e": GrMobRichMapper.utf8Offset(in: text, utf16Offset: range.location + range.length),
            "marks": marks, "link": link, "block": block,
        ])
        guard payload != lastSelection else { return }
        lastSelection = payload
        runtime?.textChanged(onSelectionChange, payload)
    }

    // MARK: Commands

    /// One core.RunEditorCommand. See GrMobCodeCoordinator.runCommand for why an
    /// editor adopts a standing epoch without running it; the rule is the same
    /// on all four hosts.
    func runCommand(epoch: Int, command: String) {
        guard let last = lastEpoch else {
            lastEpoch = epoch
            return
        }
        guard epoch != 0, epoch != last else { return }
        lastEpoch = epoch
        guard let view, view.isEditable else { return }

        switch command {
        case "undo":
            view.undoManager?.undo()
            textViewDidChange(view)
            return
        case "redo":
            view.undoManager?.redo()
            textViewDidChange(view)
            return
        default:
            break
        }

        if command.hasPrefix("block:") {
            // A block kind changes a paragraph's prefix and its indentation, so
            // there is no in-place edit that expresses it: read out, transform,
            // rebuild, restore.
            let kind = String(command.dropFirst("block:".count))
            let range = view.selectedRange
            doc = GrMobRichMapper.document(from: view.attributedText)
            doc = GrMobRichMapper.setBlockKind(doc, over: paragraphSpan(in: view), to: kind)
            rebuild(selecting: range)
            send()
            return
        }

        applyMark(command, in: view)
    }

    /// The block indices the selection touches, which is what a block command
    /// acts on.
    private func paragraphSpan(in view: UITextView) -> ClosedRange<Int> {
        let text = view.attributedText.string as NSString
        let selection = view.selectedRange
        let before = text.substring(to: min(selection.location, text.length))
        let through = text.substring(to: min(selection.location + selection.length, text.length))
        let first = before.components(separatedBy: "\n").count - 1
        let last = through.components(separatedBy: "\n").count - 1
        return first...max(first, last)
    }

    /// A mark, applied to the textStorage in place — which preserves the caret
    /// and costs no rebuild — or to the typing attributes when nothing is
    /// selected, which is what makes "press bold, then type" work here for free.
    private func applyMark(_ command: String, in view: UITextView) {
        let range = view.selectedRange
        let storage = view.textStorage

        if command.hasPrefix("link:") {
            let url = String(command.dropFirst("link:".count))
            guard range.length > 0, let link = URL(string: url) else { return }
            storage.addAttribute(.link, value: link, range: range)
            textViewDidChange(view)
            return
        }
        if command == "unlink" {
            guard range.length > 0 else { return }
            storage.removeAttribute(.link, range: range)
            textViewDidChange(view)
            return
        }

        // All, not any: selecting a sentence with one bold word in it and
        // pressing bold makes the sentence bold rather than unbolding the word.
        let on = !markIsEverywhere(command, in: storage, range: range, view: view)
        guard range.length > 0 else {
            view.typingAttributes = GrMobRichMapper.setMark(command, on: on,
                                                            in: view.typingAttributes, base: base)
            return
        }
        storage.beginEditing()
        storage.enumerateAttributes(in: range, options: []) { attrs, subrange, _ in
            storage.setAttributes(GrMobRichMapper.setMark(command, on: on, in: attrs, base: base),
                                  range: subrange)
        }
        storage.endEditing()
        textViewDidChange(view)
    }

    private func markIsEverywhere(_ command: String, in storage: NSTextStorage,
                                  range: NSRange, view: UITextView) -> Bool {
        guard range.length > 0 else {
            return GrMobRichMapper.hasMark(command, in: view.typingAttributes)
        }
        var everywhere = true
        storage.enumerateAttributes(in: range, options: []) { attrs, _, _ in
            if !GrMobRichMapper.hasMark(command, in: attrs) { everywhere = false }
        }
        return everywhere
    }
}

// MARK: - The mapping, both ways

enum GrMobRichMapper {

    /// richtext.Doc -> NSAttributedString. See the file doc for the table.
    static func attributed(from doc: [String: Any], base: GrMobRichStyle) -> NSAttributedString {
        let out = NSMutableAttributedString()
        let blocks = doc["b"] as? [[String: Any]] ?? []
        var ordinal = 0

        for (index, block) in blocks.enumerated() {
            let kind = block["k"] as? String ?? "p"
            ordinal = kind == "numbered" ? ordinal + 1 : 0
            let blockAttrs: [NSAttributedString.Key: Any] = [
                .font: base.font(for: kind),
                .foregroundColor: base.ink,
                .paragraphStyle: base.paragraph(for: kind),
                .grMobBlockKind: kind,
            ]

            // The list prefix, drawn as text because a UITextView has no list
            // rendering of its own — and marked so the reverse mapping drops it
            // rather than handing the user back their own bullets.
            if kind == "bullet" || kind == "numbered" {
                var prefixAttrs = blockAttrs
                prefixAttrs[.grMobPrefix] = true
                let prefix = kind == "bullet" ? "•\t" : "\(ordinal).\t"
                out.append(NSAttributedString(string: prefix, attributes: prefixAttrs))
            }

            for run in block["r"] as? [[String: Any]] ?? [] {
                guard let text = run["t"] as? String, !text.isEmpty else { continue }
                var attrs = blockAttrs
                if truthy(run["b"]) { attrs[.font] = withTrait(attrs[.font] as? UIFont, .traitBold) }
                if truthy(run["i"]) { attrs[.font] = withTrait(attrs[.font] as? UIFont, .traitItalic) }
                if truthy(run["u"]) { attrs[.underlineStyle] = NSUnderlineStyle.single.rawValue }
                if truthy(run["s"]) { attrs[.strikethroughStyle] = NSUnderlineStyle.single.rawValue }
                if truthy(run["c"]) {
                    attrs[.grMobCode] = true
                    attrs[.font] = UIFont.monospacedSystemFont(ofSize: base.size * 0.9, weight: .regular)
                }
                if let url = run["l"] as? String, !url.isEmpty, let link = URL(string: url) {
                    attrs[.link] = link
                }
                out.append(NSAttributedString(string: text, attributes: attrs))
            }

            if index < blocks.count - 1 {
                out.append(NSAttributedString(string: "\n", attributes: blockAttrs))
            }
        }
        return out
    }

    /// NSAttributedString -> richtext.Doc.
    ///
    /// A reading rather than an inference: the block kind comes off
    /// `.grMobBlockKind` and the drawn prefixes come off `.grMobPrefix`, both of
    /// which this file put there. What is inferred is only what the text engine
    /// owns — the font's traits, the underline, the link — because those are the
    /// attributes the *user* can change through the keyboard and the system's
    /// own controls.
    static func document(from attributed: NSAttributedString?) -> [String: Any] {
        guard let attributed, attributed.length > 0 else { return ["b": [["r": []]]] }
        let text = attributed.string as NSString

        var blocks: [[String: Any]] = []
        var paragraphStart = 0
        while paragraphStart <= text.length {
            let rest = NSRange(location: paragraphStart, length: text.length - paragraphStart)
            let newline = text.range(of: "\n", options: [], range: rest)
            let end = newline.location == NSNotFound ? text.length : newline.location
            let span = NSRange(location: paragraphStart, length: end - paragraphStart)

            var kind = "p"
            var runs: [[String: Any]] = []
            if span.length > 0 {
                kind = attributed.attribute(.grMobBlockKind, at: span.location,
                                            effectiveRange: nil) as? String ?? "p"
                attributed.enumerateAttributes(in: span, options: []) { attrs, subrange, _ in
                    if attrs[.grMobPrefix] != nil { return }
                    let piece = text.substring(with: subrange)
                    guard !piece.isEmpty else { return }
                    var run: [String: Any] = ["t": piece]
                    if let traits = (attrs[.font] as? UIFont)?.fontDescriptor.symbolicTraits {
                        if traits.contains(.traitBold) { run["b"] = 1 }
                        if traits.contains(.traitItalic) { run["i"] = 1 }
                    }
                    if attrs[.underlineStyle] != nil { run["u"] = 1 }
                    if attrs[.strikethroughStyle] != nil { run["s"] = 1 }
                    if attrs[.grMobCode] != nil { run["c"] = 1 }
                    if let url = attrs[.link] { run["l"] = "\(url)" }
                    runs.append(run)
                }
            }
            var block: [String: Any] = ["r": merge(runs)]
            if kind != "p" { block["k"] = kind }
            blocks.append(block)

            if newline.location == NSNotFound { break }
            paragraphStart = end + 1
        }
        return ["b": blocks]
    }

    /// setBlockKind is the one document transformation this host performs on the
    /// model rather than on the text, because a block kind has no in-place
    /// spelling — see the coordinator.
    static func setBlockKind(_ doc: [String: Any], over span: ClosedRange<Int>,
                             to kind: String) -> [String: Any] {
        var blocks = doc["b"] as? [[String: Any]] ?? []
        for i in span where i >= 0 && i < blocks.count {
            if kind == "p" {
                // Paragraph is the wire's absent kind, so writing it explicitly
                // would make a document that round-trips through here differ
                // from one Go marshalled — same bytes, different keys.
                blocks[i].removeValue(forKey: "k")
            } else {
                blocks[i]["k"] = kind
            }
        }
        return ["b": blocks]
    }

    /// merge collapses adjacent runs with identical formatting and drops the
    /// empty ones — the maximal-run rule, which keeps a document from growing a
    /// run boundary on every keystroke while looking identical on screen.
    private static func merge(_ runs: [[String: Any]]) -> [[String: Any]] {
        var out: [[String: Any]] = []
        for run in runs {
            guard let text = run["t"] as? String, !text.isEmpty else { continue }
            if var previous = out.last, sameMarks(previous, run) {
                previous["t"] = (previous["t"] as? String ?? "") + text
                out[out.count - 1] = previous
                continue
            }
            out.append(run)
        }
        return out
    }

    private static func sameMarks(_ a: [String: Any], _ b: [String: Any]) -> Bool {
        for key in ["b", "i", "u", "s", "c"] where truthy(a[key]) != truthy(b[key]) {
            return false
        }
        return (a["l"] as? String ?? "") == (b["l"] as? String ?? "")
    }

    // MARK: Marks as attributes

    static func hasMark(_ command: String, in attrs: [NSAttributedString.Key: Any]) -> Bool {
        switch command {
        case "bold": return trait(attrs, .traitBold)
        case "italic": return trait(attrs, .traitItalic)
        case "underline": return attrs[.underlineStyle] != nil
        case "strike": return attrs[.strikethroughStyle] != nil
        case "code": return attrs[.grMobCode] != nil
        default: return false
        }
    }

    static func setMark(_ command: String, on: Bool, in attrs: [NSAttributedString.Key: Any],
                        base: GrMobRichStyle) -> [NSAttributedString.Key: Any] {
        var next = attrs
        let font = attrs[.font] as? UIFont ?? base.font(for: "p")
        switch command {
        case "bold":
            next[.font] = on ? withTrait(font, .traitBold) : withoutTrait(font, .traitBold)
        case "italic":
            next[.font] = on ? withTrait(font, .traitItalic) : withoutTrait(font, .traitItalic)
        case "underline":
            if on { next[.underlineStyle] = NSUnderlineStyle.single.rawValue }
            else { next.removeValue(forKey: .underlineStyle) }
        case "strike":
            if on { next[.strikethroughStyle] = NSUnderlineStyle.single.rawValue }
            else { next.removeValue(forKey: .strikethroughStyle) }
        case "code":
            if on {
                next[.grMobCode] = true
                next[.font] = UIFont.monospacedSystemFont(ofSize: base.size * 0.9, weight: .regular)
            } else {
                next.removeValue(forKey: .grMobCode)
                next[.font] = UIFont.systemFont(ofSize: base.size)
            }
        default:
            break
        }
        return next
    }

    private static func trait(_ attrs: [NSAttributedString.Key: Any],
                              _ trait: UIFontDescriptor.SymbolicTraits) -> Bool {
        (attrs[.font] as? UIFont)?.fontDescriptor.symbolicTraits.contains(trait) ?? false
    }

    /// Bold and italic go through the symbolic traits rather than through a
    /// named face, so a theme's own font keeps its family when it is emboldened.
    private static func withTrait(_ font: UIFont?, _ trait: UIFontDescriptor.SymbolicTraits) -> UIFont {
        guard let font else { return .systemFont(ofSize: 17) }
        var traits = font.fontDescriptor.symbolicTraits
        traits.insert(trait)
        guard let descriptor = font.fontDescriptor.withSymbolicTraits(traits) else { return font }
        return UIFont(descriptor: descriptor, size: font.pointSize)
    }

    private static func withoutTrait(_ font: UIFont, _ trait: UIFontDescriptor.SymbolicTraits) -> UIFont {
        var traits = font.fontDescriptor.symbolicTraits
        traits.remove(trait)
        guard let descriptor = font.fontDescriptor.withSymbolicTraits(traits) else { return font }
        return UIFont(descriptor: descriptor, size: font.pointSize)
    }

    /// A UTF-16 offset as a byte offset into the UTF-8 text, which is the unit
    /// core.RichSelection promises.
    static func utf8Offset(in text: NSString, utf16Offset: Int) -> Int {
        let clamped = max(0, min(utf16Offset, text.length))
        return (text.substring(to: clamped) as String).utf8.count
    }
}

/// truthy reads a wire mark, which core writes as 1 and a hand-written document
/// may spell as `true`. Both are accepted for richtext's own markFlag reason.
private func truthy(_ value: Any?) -> Bool {
    if let number = value as? NSNumber { return number.intValue != 0 }
    if let flag = value as? Bool { return flag }
    if let text = value as? String { return !text.isEmpty && text != "0" && text != "false" }
    return false
}

/// The document's JSON, in and out. JSONSerialization rather than Codable
/// because the wire shape *is* what this host holds — there are no Swift structs
/// in the middle to encode, which is the same decision the web runtime makes.
enum GrMobRichJSON {
    static func parse(_ json: String) -> [String: Any]? {
        guard let data = json.data(using: .utf8),
              let object = try? JSONSerialization.jsonObject(with: data) as? [String: Any] else {
            return nil
        }
        return object
    }

    static func stringify(_ value: [String: Any]) -> String {
        guard let data = try? JSONSerialization.data(withJSONObject: value,
                                                     options: [.sortedKeys]),
              let text = String(data: data, encoding: .utf8) else {
            return "{}"
        }
        return text
    }

    /// The document's plain text, for the empty test the placeholder needs.
    static func text(of doc: [String: Any]) -> String {
        var out = ""
        for block in doc["b"] as? [[String: Any]] ?? [] {
            for run in block["r"] as? [[String: Any]] ?? [] {
                out += run["t"] as? String ?? ""
            }
        }
        return out.trimmingCharacters(in: .whitespacesAndNewlines)
    }
}

#else

/// The non-iOS build. Renderer.swift's dispatch has one arm for this node type
/// on every platform, so the type has to exist on every platform — and the macOS
/// typecheck that runs in ios/verify compiles these files without UIKit. See
/// GrMobMapView.swift, which takes the same shape for the same reason.
struct GrMobRichTextEditor: View {
    let node: GrMobNode
    let grow: GrMobGrow

    var body: some View {
        Color.clear.grMobBox(node.style, grow: grow)
    }
}

#endif
