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
///   the cross axis          there is no min-height half of this. A text's
///                           min-content HEIGHT is a function of the width it
///                           is laid out at, which a tree walk does not know,
///                           and a Column that overflows its height rather
///                           than compressing is a bigger behavioural change
///                           than the defect being fixed. Columns keep the
///                           floorless behaviour they have always had.
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
        return inner + outerInsets(node.style)
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
        return CGFloat(s.padding.left + s.padding.right + s.margin.left + s.margin.right)
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
