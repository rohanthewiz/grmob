import CoreGraphics
import CoreText
import Foundation
import SwiftUI

/// CSS min-content width, computed over the node tree.
///
/// # Why this is computed and not measured
///
/// Every CSS flex item carries `min-width: auto`, which floors it at its own
/// min-content size, and that floor is why an overflowing row on the web
/// overflows rather than grinding its children down to nothing. GrMobFlexSolver
/// needs the floor as a number, one per child.
///
/// The obvious way to get it on this host is to ask the view: SwiftUI
/// documents `ProposedViewSize.zero` as the way to request a subview's
/// minimum. That was tried first and it does not work, and the reason is not a
/// detail — a SwiftUI `Text` ACCEPTS any width it is proposed and wraps or
/// truncates to fit, so `sizeThatFits(.zero)` on the tutorial's lesson number
/// reports a width of exactly 0.0. Measured on a simulator:
///
///     [GRMOBMIN] axis=horizontal i=0 zero=0.0 base=21.0
///
/// A view that answers "zero" to "how small can you be" has no minimum to
/// report, so there is nothing to read out of the view layer at all. What
/// there is instead is the node: a `core.Text` carries its string and its
/// style, and CSS's min-content of a run of text is a function of those two.
/// So this walks the tree and measures the text itself.
///
/// # What min-content is
///
/// The width of the widest UNBREAKABLE run — the widest piece the line breaker
/// cannot split. For "Rows, Columns & spacing" that is "Columns"; for "4.12",
/// which has no break opportunity in it at all, it is the whole string, which
/// is why a row number is its own minimum and was the shape that exposed the
/// missing floor.
///
/// Containers compose the way CSS says: a row's min-content is the sum of its
/// children's plus its gaps (they all sit on one line), a column's is the
/// widest of its children's (they sit on separate ones).
///
/// # Every place this deliberately under-estimates
///
/// A floor that is too LOW can only leave a child as crushable as it was
/// before this file existed. A floor that is too HIGH makes a line overflow
/// that a browser would have fitted, which is a visible defect this host would
/// own alone. So every unknown here resolves downwards:
///
///   a declared main size    floors at 0. CSS's automatic minimum size is
///                           min(specified size suggestion, content size
///                           suggestion) — a declared width is a CEILING on
///                           the minimum, not a floor — and an empty
///                           `Box(Width("60px"))` therefore floors at 0 in a
///                           browser, which is the column internal/pinfixture
///                           records and wasm/verify has watched Chrome
///                           produce.
///
///   a border                not added. It is drawn as an overlay here
///                           (grMobBorder strokes the shape rather than
///                           insetting it), so unlike padding and margin it is
///                           not in the base size the floor is compared
///                           against, and adding it would be measuring
///                           something the rest of the arithmetic does not
///                           carry.
///
///   every node type that    floors at 0 — buttons, inputs, images, maps. Each
///   is not text or a        has a min-content size in CSS and none of them is
///   plain container         a function of a string, so each would need its
///                           own measurement. Text is where the divergence was
///                           found and where it bites, being the only leaf
///                           whose whole business is to be narrower than it
///                           wants to be.
///
///   the column axis         not measured here. A text's min-content HEIGHT
///                           is a function of the width it is laid out at,
///                           which a tree walk does not know — but the flex
///                           layout measures exactly that as the child's base
///                           size. So for a Column this file only decides
///                           WHETHER a child floors at its base
///                           (floorsHeightAtContent) and the layout supplies
///                           the number. The verdict keeps the same bias:
///                           every child it is unsure of floors at zero.
enum GrMobMinContent {

    /// The min-content width of a node, in points.
    ///
    /// Recursive, and the recursion is the point: a floor that stopped at the
    /// first container would be a floor that silently does nothing one level
    /// down, which is the same class of quiet divergence from CSS this exists
    /// to close.
    static func width(of node: GrMobNode) -> CGFloat {
        // A declared width caps the minimum rather than raising it; see the
        // under-estimate table above. A percentage is a declaration too — it
        // resolves against the container, which is the thing being sized.
        if !(node.style?.width ?? "").isEmpty { return 0 }

        let inner: CGFloat
        switch node.type {
        case "Text":
            inner = textWidth(node.stringProp("content"), style: node.style)
        // A row lays its children out on one line, so its minimum is all of
        // theirs plus the gaps, which do not shrink. `horizontalGap` is the
        // same read GrMobFlexStack makes for the same axis.
        case "Row":
            let gaps = CGFloat(max(node.children.count - 1, 0)) * (node.style?.horizontalGap ?? 0)
            inner = node.children.reduce(gaps) { $0 + width(of: $1) }
        // A column stacks them, so its minimum is the widest one. Card and Box
        // render through GrMobColumn and are the same shape; Fragment and
        // Theme place nothing of their own and pass their children through, so
        // the widest child is theirs too.
        case "Column", "Card", "Box", "Fragment", "Theme":
            inner = node.children.reduce(0) { max($0, width(of: $1)) }
        default:
            return 0
        }
        // A node with nothing to measure inside it has no minimum to speak of,
        // and adding its padding to zero would floor an empty Box at its own
        // insets — which is not what a browser does with one.
        guard inner > 0 else { return 0 }
        return capped(inner, node.style) + outerInsets(node.style)
    }

    /// Whether a Column child's automatic minimum height is its content
    /// height (CSS `min-height: auto` on a column flex item) rather than zero.
    ///
    /// With no floor, a Column offered less height than its children need
    /// shrinks every one of them below its content, and SwiftUI draws a
    /// squeezed Text at its full height anyway, so its lines overlap the next
    /// sibling. A browser keeps each child whole and lets the column overflow,
    /// and Compose's Column measures children in order and never squeezes a
    /// Text below its lines either. A DatePicker's day-marker dots drawn over
    /// their numbers in a squeezed sheet was this.
    ///
    /// The answer is yes only where the base size the layout measures IS the
    /// CSS floor, and no wherever that is in doubt — the width walk's bias, for
    /// the width walk's reason: a floor too high overflows a column a browser
    /// would have fitted.
    ///
    /// ```
    ///   rule (first match wins)        floor  why
    ///   -----------------------        -----  ---
    ///   Overflow other than visible    no     CSS: a scroll container's
    ///                                         automatic minimum is zero
    ///   a scrolling node type          no     the same rule, even with a
    ///                                         points Height
    ///   Height in points               yes    a rigid frame: the base is the
    ///                                         declared size, and so is CSS's
    ///                                         min(specified, content) whenever
    ///                                         the content is taller
    ///   any other Height ("50%")       no     resolves against the column
    ///                                         being sized
    ///   Text, Button, Spacer           yes    the base is their lines (a
    ///                                         Spacer's is empty)
    ///   Row/Column/Card/Box/           yes    only when every child is: one
    ///   Fragment/Theme                        child in doubt is squeezable
    ///                                         height inside the base
    ///   anything else                  no     images, inputs, maps, toggles
    /// ```
    ///
    /// The scroll-container rule is also what keeps a nested scroll scrolling:
    /// a Screen's FlexGrow Scroll or List — and any container holding one —
    /// floors at zero, so it still absorbs the squeeze instead of being held
    /// open at its whole content height. That is the `min-height: 0` trap CSS
    /// authors hit in nested flex columns, avoided here by the verdict.
    static func floorsHeightAtContent(_ node: GrMobNode) -> Bool {
        let overflow = node.style?.overflow ?? ""
        if !overflow.isEmpty && overflow != "visible" { return false }
        switch node.type {
        case "Scroll", "List", "TextArea", "CodeEditor", "TextGrid", "RichTextEditor":
            return false
        default:
            break
        }
        // "auto" is CSS's initial height, the same as saying nothing.
        let height = node.style?.height ?? ""
        if !height.isEmpty && height != "auto" {
            return GrMobMaxWidth.fixedLimit(height) != nil
        }
        switch node.type {
        case "Text", "Button", "Spacer":
            return true
        case "Row", "Column", "Card", "Box", "Fragment", "Theme":
            return node.children.allSatisfy(floorsHeightAtContent)
        default:
            return false
        }
    }

    /// The content minimum clamped by core.MaxWidth, in points only.
    ///
    /// CSS: an item's content size suggestion "is further clamped by the max
    /// main size if that is definite". A capped box can be narrower than its
    /// longest word — the word overflows, as it does in a browser — so a
    /// floor above the cap would hold a row open for a box that can never
    /// grow to fill it.
    ///
    /// The cap limits the box grMobBox paints, padding included (the host's
    /// Width and MaxWidth are both border-box, as the WASM page's
    /// `box-sizing: border-box` makes them there), so the padding is folded
    /// in before the clamp and only the margin is added outside it. A
    /// percentage cap has no container here and does not clamp; a floor that
    /// is too high overflows a line, but a floor guessed too low is the
    /// crushed-badge bug this walk exists to prevent.
    private static func capped(_ inner: CGFloat, _ s: GrMobStyle?) -> CGFloat {
        guard let s, let cap = GrMobMaxWidth.fixedLimit(s.maxWidth) else { return inner }
        // The border inset counts with the padding: both are inside the
        // border box the cap limits (GrMobStyle.contentInsets).
        let padding = CGFloat(s.padding.left + s.padding.right) + 2 * s.borderInset
        return min(inner + padding, cap) - padding
    }

    /// What a node's own box adds around its content along the main axis.
    ///
    /// The floor is compared against a flex base size, and on this host that
    /// base is the whole painted box: grMobBox applies the padding and then
    /// the margin outside it, so a badge whose text measures 26 points is 42
    /// wide with 8 points of padding on each side, and a floor of 26 lets the
    /// row crush it to the shape that started this — "TRY IT" drawn as
    /// TR / Y / IT. This is CSS's rule too: the automatic minimum floors the
    /// content box, and the item's outer size carries its padding as well.
    private static func outerInsets(_ s: GrMobStyle?) -> CGFloat {
        guard let s else { return 0 }
        // A drawn border insets the content too (GrMobStyle.contentInsets), so
        // it is part of what the box adds around its content.
        return CGFloat(s.padding.left + s.padding.right + s.margin.left + s.margin.right)
            + 2 * s.borderInset
    }

    /// The widest unbreakable run of a string, at a node's own text style.
    static func textWidth(_ text: String, style: GrMobStyle?) -> CGFloat {
        runs(of: text).reduce(0) { max($0, measure($1, style: style)) }
    }

    /// The pieces a line breaker may not split, in order.
    ///
    /// Whitespace is the break opportunity everybody agrees on. The hyphen and
    /// the slash are here because CSS breaks AFTER both, and leaving them out
    /// would over-estimate "one-off" and "and/or" — and over-estimating is the
    /// one direction with a visible cost (see the table above). The separators
    /// are kept on the run they follow, which is what a browser draws: a line
    /// broken after "one-" leaves the hyphen on the first line.
    ///
    /// What this does not model is the rest of UAX #14 — CJK, where nearly
    /// every character is a break opportunity, and the several punctuation
    /// classes with their own rules. Both would only make the runs SHORTER,
    /// so the floor this produces for them is already on the safe side of the
    /// browser's.
    static func runs(of text: String) -> [String] {
        var out: [String] = []
        var current = ""
        for ch in text {
            if ch.isWhitespace {
                if !current.isEmpty { out.append(current); current = "" }
                continue
            }
            current.append(ch)
            if ch == "-" || ch == "/" {
                out.append(current)
                current = ""
            }
        }
        if !current.isEmpty { out.append(current) }
        return out
    }

    /// One run, measured with CoreText.
    ///
    /// CoreText rather than UIKit's `NSAttributedString.size(withAttributes:)`
    /// for one reason and it is a structural one: `ios/verify` type-checks
    /// every runtime file against a macOS target, where UIKit does not exist,
    /// and it compiles this file into its harness so the walk above can be
    /// checked without a simulator. CoreText is on both platforms, so the
    /// measurement runs in the harness rather than being taken on trust.
    ///
    /// The font is built to match `grMobTextStyle`'s
    /// `.font(.system(size:weight:))`: the system font at the same size, with
    /// the same weight expressed as CoreText's -1...1 trait — the scale
    /// UIFont.Weight's raw values are on. Both answers come out of the one
    /// grMobFontWeightPair below, so the face this measures cannot drift from
    /// the face grMobTextStyle draws.
    static func measure(_ run: String, style: GrMobStyle?) -> CGFloat {
        guard !run.isEmpty else { return 0 }
        let size = (style?.fontSize ?? 0) > 0 ? style!.fontSize : 17
        let font = systemFont(size: size, trait: grMobFontWeightPair(style?.fontWeight ?? 0).1)
        let attributed = NSAttributedString(string: run, attributes: [
            NSAttributedString.Key(kCTFontAttributeName as String): font,
        ])
        let line = CTLineCreateWithAttributedString(attributed)
        return CGFloat(CTLineGetTypographicBounds(line, nil, nil, nil))
    }

    /// The system font at a size and a weight trait.
    ///
    /// `CTFontCreateUIFontForLanguage(.system, size, nil)` gives the regular
    /// face; the weight is applied by re-describing it, which is what
    /// UIFont.systemFont(ofSize:weight:) does underneath. A descriptor that
    /// cannot be satisfied comes back nil, and the regular face is then the
    /// answer — a weight that could not be found is a lighter, narrower face,
    /// which is an under-estimate and therefore safe.
    private static func systemFont(size: CGFloat, trait: CGFloat) -> CTFont {
        let base = CTFontCreateUIFontForLanguage(.system, size, nil)
            ?? CTFontCreateWithName("Helvetica" as CFString, size, nil)
        guard trait != 0 else { return base }
        let descriptor = CTFontDescriptorCreateCopyWithAttributes(
            CTFontCopyFontDescriptor(base),
            [kCTFontTraitsAttribute: [kCTFontWeightTrait: trait]] as CFDictionary)
        return CTFontCreateWithFontDescriptor(descriptor, size, nil)
    }
}

/// Go's Weight constants are the CSS numeric scale (200/400/700...), and this
/// is the one ladder that maps them onto a face — in two vocabularies, because
/// two things here need a font and they ask for it differently.
///
/// `.0` is SwiftUI's named weight, which grMobTextStyle hands to
/// `.font(.system(size:weight:))`. `.1` is the same weight as a number on
/// CoreText's -1...1 scale, which GrMobMinContent needs to build a CTFont to
/// measure with; the numbers are UIFont.Weight's documented raw values, which
/// is the same scale under another name.
///
/// One arm per line, both answers on it, because the whole reason this is one
/// function is that a ladder split in two is a ladder that can be edited in
/// one place: a measurement taken at a different weight from the one that gets
/// drawn is a floor that is quietly wrong, and wrong in the direction that
/// overflows a row.
///
/// It lives here rather than beside its SwiftUI caller because `ios/verify`
/// compiles this file into its harness and does not compile Renderer.swift —
/// the measurement can be checked off-device only if everything it reaches is
/// on this side of that line.
func grMobFontWeightPair(_ w: Int) -> (Font.Weight, CGFloat) {
    switch w {
    case ..<1: (.regular, 0)
    case ..<200: (.ultraLight, -0.8)
    case ..<300: (.thin, -0.6)
    case ..<400: (.light, -0.4)
    case ..<500: (.regular, 0)
    case ..<600: (.medium, 0.23)
    case ..<700: (.semibold, 0.3)
    case ..<800: (.bold, 0.4)
    case ..<900: (.heavy, 0.56)
    default: (.black, 0.62)
    }
}
