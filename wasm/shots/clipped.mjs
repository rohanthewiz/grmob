// The shutter's clipping check, apart from the camera that runs it.
//
// shot.mjs evaluates CLIPPED in the page before it writes a PNG. It lives in
// its own module for the reason wasm/verify's inkface.mjs and fold.mjs do:
// clipped_test.mjs can hold it to pages built to fail. A run over the real
// shots cannot — every committed shot passes, and a check that never fires
// reads exactly like a check that works.

// CLIPPED is evaluated in the page with the claimed strings and the clip
// selector, and returns one line per string that no occurrence shows whole.
//
// # Why this exists
//
// internal/shotclaims holds each picture to the strings its app renders, by
// reading a node tree in a Go test. A tree has no layout, so a string can be
// in the tree whole and in the picture cut off: a code line wider than the
// phone scrolls sideways inside its CodeEditor, and tutorial-lesson.png
// claimed "func Profile(ctx *core.Context) core.View {" while the frame
// showed "…core.Vie". The camera is the one place the layout exists.
//
// # How "whole" is decided
//
// The text of every text node is concatenated in document order, each
// occurrence of the string becomes a DOM Range over the nodes it spans, and
// the Range's line boxes (getClientRects — one per wrapped line, so "TRY IT"
// broken over two lines still counts) must each lie inside:
//
//	the clip rect (the bezel)
//	  └─ every ancestor whose overflow on that axis is not visible
//	       (the app's scroll view vertically, a CodeEditor's <pre>
//	        sideways) — the boxes that actually cut the glyphs off
//
// One whole occurrence is enough: "Header()" is both a code token and a
// checkbox label, and the claim is that the reader can see it, not where.
// A Range with no boxes at all is text that is not laid out (the CodeEditor's
// transparent textarea, a display:none subtree) and never counts as shown.
//
// Some claimed text is not a text node at all: a field's placeholder or typed
// value, and a <select>'s chosen option, are painted by the control. Those are
// checked as the control's whole box against the same frame and clipping
// ancestors — coarser, since a long value can still scroll inside its own
// field, but a field cut by the frame is caught. The CodeEditor's buffer is a
// textarea too and is skipped: its ink is transparent.
//
// # Field text, by where it is painted
//
// A single-line <input> is narrower than that coarse rule needs to be. Its
// text is one line with no wrapping, so where each character lands can be
// computed without a layout the DOM exposes: the field's content box, its
// scrollLeft, its text-align, and the width of the text before and through the
// claimed string in the field's own font (a canvas measureText). That rect is
// then held to the field's content box — the box an input clips its text to —
// before the frame and the ancestors, so a value that runs past the field's
// right edge is reported for its hidden tail and passes for its visible head.
//
// Only the string actually painted is searched: the value when there is one,
// the placeholder only when the value is empty. The coarse rule accepted a
// placeholder hidden behind a typed value.
//
// Three cases keep the box rule, each because the arithmetic above would be a
// guess: a <textarea> (it wraps), a <select> (the option's paint is the
// platform's), and a right-to-left field (its overflow scrolls from the other
// edge and scrollLeft's sign differs between engines).
//
// Concatenating across elements can match a string that straddles two
// unrelated nodes. That can only make the check more lenient, never report a
// visible string as clipped, which is the direction a shutter check should
// err in.
export const CLIPPED = `(shows, clipSel) => {
    const frame = document.querySelector(clipSel).getBoundingClientRect();
    const nodes = [];
    let text = "";
    const walk = document.createTreeWalker(document.body, NodeFilter.SHOW_TEXT);
    for (let n = walk.nextNode(); n; n = walk.nextNode()) {
        nodes.push({ n, start: text.length });
        text += n.data;
    }
    // The text node holding character offset off, and the offset within it.
    const at = (off) => {
        let lo = 0, hi = nodes.length - 1;
        while (lo < hi) {
            const mid = (lo + hi + 1) >> 1;
            if (nodes[mid].start <= off) lo = mid; else hi = mid - 1;
        }
        return { node: nodes[lo].n, offset: off - nodes[lo].start };
    };
    // Sub-pixel glyph overhang is not a clipped letter.
    const SLACK = 0.5;
    const inside = (r, box) =>
        r.left >= box.left - SLACK && r.right <= box.right + SLACK &&
        r.top >= box.top - SLACK && r.bottom <= box.bottom + SLACK;
    // Why a set of boxes is not whole: the first box that cuts one, or "".
    // el is the innermost element whose overflow could clip them.
    const cutRects = (rects, el) => {
        if (rects.length === 0) return "not laid out";
        for (const r of rects) {
            if (!inside(r, frame)) return "outside the frame";
            for (let a = el; a && a !== document.body; a = a.parentElement) {
                const cs = getComputedStyle(a);
                const box = a.getBoundingClientRect();
                const cutX = cs.overflowX !== "visible" && (r.left < box.left - SLACK || r.right > box.right + SLACK);
                const cutY = cs.overflowY !== "visible" && (r.top < box.top - SLACK || r.bottom > box.bottom + SLACK);
                if (cutX || cutY) {
                    const name = a.getAttribute("data-node-type") || a.tagName.toLowerCase();
                    return (cutX ? "cut sideways" : "cut vertically") + " by its " + name;
                }
            }
        }
        return "";
    };
    const laidOut = (rs) => Array.from(rs).filter((r) => r.width > 0 && r.height > 0);
    const cutBy = (range) => {
        let el = range.commonAncestorContainer;
        if (el.nodeType !== 1) el = el.parentElement;
        return cutRects(laidOut(range.getClientRects()), el);
    };
    // The texts a form control paints itself, which no text node holds. An
    // input paints its placeholder only while its value is empty.
    const painted = (el) => {
        if (el.getAttribute("data-grmob-chrome") === "codebuffer") return [];
        if (el.tagName === "SELECT") return el.selectedOptions[0] ? [el.selectedOptions[0].text] : [];
        if (el.tagName === "INPUT") return [el.value || el.placeholder || ""];
        return [el.value || "", el.placeholder || ""];
    };
    // One canvas for every measurement, made on first use: most shots have
    // no field text claimed, and they should not pay for a canvas.
    let ctx2d = null;
    const measure = (font, s) => {
        ctx2d = ctx2d || document.createElement("canvas").getContext("2d");
        ctx2d.font = font;
        return ctx2d.measureText(s).width;
    };
    // Where s is painted inside a single-line input, as a rect in viewport
    // coordinates, or null when the arithmetic would be a guess (see "Field
    // text, by where it is painted").
    const inputTextRect = (el, s) => {
        if (el.tagName !== "INPUT") return null;
        const cs = getComputedStyle(el);
        if (cs.direction === "rtl") return null;
        const shown = painted(el)[0];
        const i = shown.indexOf(s);
        if (i < 0) return null;
        // cs.font is the shorthand Chrome composes from the longhands; the
        // explicit form is the fallback for an engine that leaves it empty.
        const font = cs.font || [cs.fontStyle, cs.fontWeight, cs.fontSize, cs.fontFamily].join(" ");
        const px = (v) => parseFloat(v) || 0;
        const box = el.getBoundingClientRect();
        const content = {
            left: box.left + px(cs.borderLeftWidth) + px(cs.paddingLeft),
            right: box.right - px(cs.borderRightWidth) - px(cs.paddingRight),
            top: box.top + px(cs.borderTopWidth) + px(cs.paddingTop),
            bottom: box.bottom - px(cs.borderBottomWidth) - px(cs.paddingBottom),
        };
        const whole = measure(font, shown);
        const room = content.right - content.left;
        // Text that fits is placed by text-align; text that does not starts
        // at the leading edge and moves by the field's own scroll.
        let x0 = content.left - el.scrollLeft;
        if (whole < room) {
            if (cs.textAlign === "center") x0 = content.left + (room - whole) / 2;
            else if (cs.textAlign === "right" || cs.textAlign === "end") x0 = content.right - whole;
        }
        // Measured through the end of s rather than as s alone, so kerning
        // across the boundary with the text before it is counted.
        const rect = {
            left: x0 + measure(font, shown.slice(0, i)),
            right: x0 + measure(font, shown.slice(0, i + s.length)),
            top: content.top,
            bottom: content.bottom,
        };
        return { rect, content };
    };
    const controls = Array.from(document.querySelectorAll("input, textarea, select"));
    const out = [];
    for (const s of shows) {
        const why = [];
        let whole = false;
        for (let i = text.indexOf(s); i >= 0 && !whole; i = text.indexOf(s, i + 1)) {
            const a = at(i), b = at(i + s.length - 1);
            const range = document.createRange();
            range.setStart(a.node, a.offset);
            range.setEnd(b.node, b.offset + 1);
            const reason = cutBy(range);
            if (reason === "") whole = true; else why.push(reason);
        }
        for (const c of controls) {
            if (whole) break;
            if (!painted(c).some((t) => t.includes(s))) continue;
            const placed = inputTextRect(c, s);
            let reason;
            if (placed) {
                // The field clips its own text first; then the text, not the
                // field, is what the frame and the ancestors must hold.
                const { rect, content } = placed;
                const cutX = rect.left < content.left - SLACK || rect.right > content.right + SLACK;
                reason = cutX ? "cut sideways by its input" : cutRects([rect], c.parentElement);
            } else {
                // The box's own overflow clips what is inside it, not the box,
                // so the clipping ancestors start at its parent.
                reason = cutRects(laidOut(c.getClientRects()), c.parentElement);
            }
            if (reason === "") whole = true; else why.push(reason);
        }
        if (!whole) {
            out.push(JSON.stringify(s) + ": " + (why.length ? [...new Set(why)].join("; ") : "not in the page"));
        }
    }
    return out;
}`;
