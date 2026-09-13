// Checks for GrMobMinContent — the CSS min-content widths that floor the
// shrink arm (GrMobMinContent.swift).
//
// # Why this can run here at all
//
// The floor is a measurement of text, and a measurement of text normally needs
// the platform that draws it. This one does not: GrMobMinContent measures with
// CoreText, which macOS has, precisely so the walk and the numbers it produces
// can be checked without a simulator — the same bargain GrMobFlexSolver makes
// by being arithmetic.
//
// # What is asserted, and what cannot be
//
// Exact point widths are not. The system font differs between macOS and iOS
// and between releases, so pinning "4.12 is 21.0 points" would pin the harness
// to a font version. What is pinned is every RELATION the floor's correctness
// rests on — a string with no break opportunity measures as wide as itself, a
// phrase measures as its longest word, bold is wider than regular, a container
// sums or maxes its children — each of which is false in exactly the ways this
// could be got wrong.
import CoreGraphics
import Foundation

private func node(_ type: String, style: GrMobStyle? = nil,
                  props: [String: Any] = [:], children: [GrMobNode] = []) -> GrMobNode {
    GrMobNode(type: type, key: "", props: props, style: style, children: children)
}

private func text(_ s: String, style: GrMobStyle? = nil) -> GrMobNode {
    node("Text", style: style, props: ["content": s])
}

func checkMinContent() -> [String] {
    var problems: [String] = []

    // --- the runs a line breaker may not split -----------------------------
    let runCases: [(String, [String])] = [
        // The string that started this. No break opportunity anywhere in it,
        // so the whole thing is one run and a row number is its own minimum —
        // which is why it was the child that exposed the missing floor.
        ("4.12", ["4.12"]),
        ("Rows, Columns & spacing", ["Rows,", "Columns", "&", "spacing"]),
        // CSS breaks AFTER a hyphen and keeps it on the first line.
        ("one-off", ["one-", "off"]),
        ("and/or", ["and/", "or"]),
        // Runs of whitespace collapse to one opportunity; leading and
        // trailing whitespace produce no empty run to measure.
        ("  a   b  ", ["a", "b"]),
        ("", []),
        ("   ", []),
    ]
    for (input, want) in runCases {
        let got = GrMobMinContent.runs(of: input)
        if got != want {
            problems.append("runs(of: \(input.debugDescription)): got \(got), want \(want)")
        }
    }

    // --- what the measurement says -----------------------------------------
    //
    // A string with no break opportunity is its own minimum: min-content and
    // the ideal width are the same number for it.
    let whole = GrMobMinContent.textWidth("4.12", style: nil)
    let oneGlyph = GrMobMinContent.textWidth("4", style: nil)
    if !(whole > oneGlyph * 2) {
        problems.append("an unbreakable run should measure as its whole self: "
            + "\"4.12\" is \(whole), \"4\" is \(oneGlyph)")
    }

    // A phrase measures as its longest word and NOT as the whole phrase —
    // both halves, because a walk that returned the whole string would pass
    // the first test and floor every paragraph at its unwrapped width.
    let phrase = "Rows, Columns & spacing"
    let longest = GrMobMinContent.textWidth("Columns", style: nil)
    let got = GrMobMinContent.textWidth(phrase, style: nil)
    if abs(got - longest) > 0.01 {
        problems.append("a phrase should measure as its longest word: got \(got), want \(longest)")
    }
    if got >= GrMobMinContent.measure(phrase, style: nil) {
        problems.append("a phrase's min-content should be narrower than the phrase")
    }

    // The weight reaches the font. This is the assertion that would fail if
    // grMobFontWeightPair's second column were wrong or unread — a floor
    // measured at the regular face for text drawn bold is a floor that is
    // quietly too small, which is the failure mode nothing visible catches.
    var bold = GrMobStyle()
    bold.fontWeight = 700
    var regular = GrMobStyle()
    regular.fontWeight = 400
    let boldWidth = GrMobMinContent.textWidth("4.12", style: bold)
    let regularWidth = GrMobMinContent.textWidth("4.12", style: regular)
    if !(boldWidth > regularWidth) {
        problems.append("bold should measure wider than regular: \(boldWidth) vs \(regularWidth)")
    }
    // And the size, which is the other half of the font.
    var big = GrMobStyle()
    big.fontSize = 34
    if !(GrMobMinContent.textWidth("4.12", style: big) > whole) {
        problems.append("a larger font size should measure wider")
    }

    // --- how containers compose --------------------------------------------
    //
    // A row puts its children on one line, so its minimum is all of theirs
    // plus the gaps that do not shrink. A column stacks them, so it is the
    // widest one. Asserted against the parts rather than against numbers,
    // which is what makes this a check of the rule and not of a font.
    let a = text("Columns")
    let b = text("4.12")
    let wA = GrMobMinContent.width(of: a)
    let wB = GrMobMinContent.width(of: b)

    var gapped = GrMobStyle()
    gapped.gap = 8
    let row = node("Row", style: gapped, children: [a, b])
    if abs(GrMobMinContent.width(of: row) - (wA + wB + 8)) > 0.01 {
        problems.append("a row's minimum should be its children plus its gaps: "
            + "got \(GrMobMinContent.width(of: row)), want \(wA + wB + 8)")
    }

    let column = node("Column", children: [a, b])
    if abs(GrMobMinContent.width(of: column) - max(wA, wB)) > 0.01 {
        problems.append("a column's minimum should be its widest child")
    }

    // Nesting, which is the property that keeps the floor from silently
    // stopping one level down: the row's minimum has to see through the
    // column to the text inside it.
    let nested = node("Row", children: [node("Column", children: [a]), b])
    if abs(GrMobMinContent.width(of: nested) - (wA + wB)) > 0.01 {
        problems.append("a nested column should carry its child's minimum outwards")
    }

    // --- the box around the content ----------------------------------------
    //
    // The floor is compared against a flex base size, and that base is the
    // whole painted box: grMobBox applies the padding and then the margin
    // around the content. A badge is the case that showed it — a Text with 8
    // points of padding on each side, crushed by a row to TR / Y / IT while a
    // floor that counted only the glyphs said it had room.
    var padded = GrMobStyle()
    padded.padding = GrMobStyle.Edges(top: 2, right: 8, bottom: 2, left: 8)
    let bare = GrMobMinContent.width(of: text("TRY IT"))
    if abs(GrMobMinContent.width(of: text("TRY IT", style: padded)) - (bare + 16)) > 0.01 {
        problems.append("horizontal padding should be added to the floor")
    }
    var margined = GrMobStyle()
    margined.margin = GrMobStyle.Edges(top: 0, right: 4, bottom: 0, left: 6)
    if abs(GrMobMinContent.width(of: text("TRY IT", style: margined)) - (bare + 10)) > 0.01 {
        problems.append("horizontal margin should be added to the floor")
    }
    // A drawn border insets the content as well (GrMobStyle.contentInsets,
    // CSS's border-box), so it is added on both sides; a width with no colour
    // draws nothing and adds nothing.
    var bordered = GrMobStyle()
    bordered.borderWidth = 2
    bordered.borderColor = .gray
    if abs(GrMobMinContent.width(of: text("TRY IT", style: bordered)) - (bare + 4)) > 0.01 {
        problems.append("a drawn border's width should be added to the floor on both sides")
    }
    var uncoloured = GrMobStyle()
    uncoloured.borderWidth = 2
    if abs(GrMobMinContent.width(of: text("TRY IT", style: uncoloured)) - bare) > 0.01 {
        problems.append("a border width with no colour draws nothing and should add nothing")
    }
    // The vertical insets are not: this is one axis.
    var tall = GrMobStyle()
    tall.padding = GrMobStyle.Edges(top: 40, right: 0, bottom: 40, left: 0)
    if abs(GrMobMinContent.width(of: text("TRY IT", style: tall)) - bare) > 0.01 {
        problems.append("vertical padding should not reach a width")
    }
    // A container with nothing measurable inside it has no floor, and its own
    // padding does not become one — an empty Box is not floored at its insets.
    if GrMobMinContent.width(of: node("Box", style: padded)) != 0 {
        problems.append("an empty box should have no floor at all")
    }

    // --- a cap clamps the floor --------------------------------------------
    //
    // CSS clamps the content size suggestion by a definite max-width, so a
    // capped box never holds a row open wider than it can ever be drawn. The
    // cap limits the padded box (border-box, as on the WASM page); the margin
    // stays outside it.
    let long = text("Supercalifragilistic")
    let longWidth = GrMobMinContent.width(of: long)
    var capped = GrMobStyle()
    capped.maxWidth = "40px"
    if abs(GrMobMinContent.width(of: text("Supercalifragilistic", style: capped)) - 40) > 0.01 {
        problems.append("a points MaxWidth should clamp the floor: got "
            + "\(GrMobMinContent.width(of: text("Supercalifragilistic", style: capped))), want 40")
    }
    var cappedBox = capped
    cappedBox.padding = GrMobStyle.Edges(top: 0, right: 8, bottom: 0, left: 8)
    cappedBox.margin = GrMobStyle.Edges(top: 0, right: 5, bottom: 0, left: 5)
    if abs(GrMobMinContent.width(of: text("Supercalifragilistic", style: cappedBox)) - 50) > 0.01 {
        problems.append("a MaxWidth caps the padded box and leaves the margin outside: got "
            + "\(GrMobMinContent.width(of: text("Supercalifragilistic", style: cappedBox))), want 50")
    }
    var roomy = GrMobStyle()
    roomy.maxWidth = "4000px"
    if abs(GrMobMinContent.width(of: text("Supercalifragilistic", style: roomy)) - longWidth) > 0.01 {
        problems.append("a MaxWidth wider than the content should not change the floor")
    }
    var pctCap = GrMobStyle()
    pctCap.maxWidth = "10%"
    if abs(GrMobMinContent.width(of: text("Supercalifragilistic", style: pctCap)) - longWidth) > 0.01 {
        problems.append("a percentage MaxWidth has no container here and should not clamp")
    }

    // --- everywhere it floors at zero on purpose ---------------------------
    //
    // Each of these is an UNDER-estimate rather than an omission, and the
    // direction matters: a floor that is too low leaves a child exactly as
    // crushable as it was before this existed, while one that is too high
    // overflows a line a browser would have fitted.
    var sized = GrMobStyle()
    sized.width = "60px"
    // A declared width CAPS the automatic minimum (CSS takes the smaller of
    // the specified and content suggestions), and this host cannot see the
    // content behind the frame that declaration becomes. Zero is what a
    // browser gives the empty sized boxes internal/pinfixture mounts.
    if GrMobMinContent.width(of: text("4.12", style: sized)) != 0 {
        problems.append("a declared width should cap the minimum at zero")
    }
    var pct = GrMobStyle()
    pct.width = "50%"
    if GrMobMinContent.width(of: text("4.12", style: pct)) != 0 {
        problems.append("a percentage width is a declaration too")
    }
    // The node types with a min-content size that is not a function of a
    // string. Each would need its own measurement; none has one.
    for type in ["Button", "Input", "Image", "MapView", "Spacer"] {
        if GrMobMinContent.width(of: node(type, props: ["content": "4.12"])) != 0 {
            problems.append("\(type) should floor at zero until something measures it")
        }
    }

    // --- the column axis: which children floor at their base height -------
    //
    // No number is pinned, because the layout measures the height. The
    // verdict is, and each case below is one where the wrong answer is
    // visible: a wrong "no" leaves lines overlapping in a squeezed column, a
    // wrong "yes" holds a scroller open at its whole content so it never
    // scrolls.
    let floors = GrMobMinContent.floorsHeightAtContent
    if !floors(text("4.12")) {
        problems.append("a Text should floor at its lines")
    }
    if !floors(node("Button")) {
        problems.append("a Button should floor at its label")
    }
    if !floors(node("Column", children: [text("a"), node("Row", children: [text("b"), node("Spacer")])])) {
        problems.append("a column of text rows should floor at its content")
    }
    for type in ["Scroll", "List", "TextArea", "CodeEditor", "TextGrid", "RichTextEditor"] {
        if floors(node(type)) {
            problems.append("\(type) is a scroll container and should floor at zero")
        }
    }
    // One scroller anywhere inside puts the container in doubt, which is what
    // keeps a Screen's nested List absorbing the squeeze.
    if floors(node("Column", children: [text("title"), node("Box", children: [node("List")])])) {
        problems.append("a column holding a List, however deep, should floor at zero")
    }
    var clipped = GrMobStyle()
    clipped.overflow = "hidden"
    if floors(node("Column", style: clipped, children: [text("a")])) {
        problems.append("Overflow(hidden) makes a scroll container, which floors at zero")
    }
    var visible = GrMobStyle()
    visible.overflow = "visible"
    if !floors(node("Column", style: visible, children: [text("a")])) {
        problems.append("Overflow(visible) is the initial value and should keep the floor")
    }
    var rigid = GrMobStyle()
    rigid.height = "48px"
    if !floors(node("Image", style: rigid)) {
        problems.append("a points Height is a rigid frame and should floor at it")
    }
    var bareNumber = GrMobStyle()
    bareNumber.height = "48"
    if !floors(node("Image", style: bareNumber)) {
        problems.append("a bare-number Height is points too")
    }
    var autoHeight = GrMobStyle()
    autoHeight.height = "auto"
    if !floors(text("a", style: autoHeight)) {
        problems.append("Height(auto) is the initial value and should keep the floor")
    }
    var relative = GrMobStyle()
    relative.height = "50%"
    if floors(text("a", style: relative)) {
        problems.append("a percentage Height resolves against the column and should floor at zero")
    }
    var rigidScroll = GrMobStyle()
    rigidScroll.height = "200px"
    if floors(node("Scroll", style: rigidScroll)) {
        problems.append("a scroller should floor at zero even with a points Height")
    }
    for type in ["Image", "Input", "MapView", "Checkbox"] {
        if floors(node(type)) {
            problems.append("\(type) with no Height should floor at zero")
        }
    }

    return problems
}
