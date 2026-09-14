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
// The measurement uses the field's own letter-spacing as well as its font:
// canvas has a letterSpacing of its own, and without it a spaced-out value is
// measured narrower than it paints, which lets a clipped tail pass.
//
// A right-to-left field is placed from the other edge. Its leading edge is
// the content box's right, text that fits is aligned by text-align read in
// that direction (start is right, end is left), and overflow scrolls with a
// scrollLeft that runs from 0 to negative, which is the CSSOM rule Chrome
// follows (the only engine this check runs in). Offsets are measured leftward
// from the leading edge. That arithmetic is only right when the whole value
// runs one way. A right-to-left value holding any left-to-right letter or any
// digit (both run left to right inside it, reordering where the claimed
// string lands) is laid out by the browser instead: a hidden copy of the
// value over the field's content box, with the field's font, spacing,
// direction, alignment and bidi mode, answers with one rect per visual run
// (bidiRects), and the field's scroll moves those rects as it moves the
// arithmetic's. A field's scroll offset is a whole pixel while advances are
// not, so field text is allowed a pixel past the content edge (FIELD_SLACK).
//
// Two cases keep the box rule, each because the arithmetic above would be a
// guess: a <textarea> (it wraps) and a <select> (the option's paint is the
// platform's).
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
    const measure = (font, spacing, s) => {
        ctx2d = ctx2d || document.createElement("canvas").getContext("2d");
        ctx2d.font = font;
        // "normal" is canvas's own spelling of none, the same as CSS's.
        ctx2d.letterSpacing = spacing;
        return ctx2d.measureText(s).width;
    };
    // A value that does not run one way in a right-to-left field: any letter
    // outside the right-to-left scripts, or any digit. The backslashes are
    // doubled because this source is a template literal evaluated in the page.
    const mixedInRtl = /\\p{Nd}|(?=\\p{L})(?![\\p{Script=Arabic}\\p{Script=Hebrew}\\p{Script=Syriac}\\p{Script=Thaana}\\p{Script=Nko}])/u;
    // Where the characters [i, i + n) of a right-to-left field's value are
    // painted, when the value mixes directions. Bidi reordering is not
    // arithmetic over advances, so the browser lays the value out itself: a
    // hidden, fixed-position copy over the field's content box with the
    // field's font, spacing, direction, alignment and bidi mode, whose Range
    // rects are then the visual runs of the substring, one per run. The
    // field's own scroll moves them all by -scrollLeft, as it moves the
    // arithmetic's rect below. white-space: pre keeps the copy one line and
    // lets a long value overflow its start edge the way the field's does.
    const bidiRects = (el, cs, font, spacing, shown, i, n, content) => {
        const probe = document.createElement("div");
        probe.textContent = shown;
        Object.assign(probe.style, {
            position: "fixed", left: content.left + "px", top: content.top + "px",
            width: (content.right - content.left) + "px", margin: "0", padding: "0", border: "0",
            whiteSpace: "pre", overflow: "visible", visibility: "hidden",
            font, letterSpacing: spacing, direction: cs.direction,
            textAlign: cs.textAlign, unicodeBidi: cs.unicodeBidi,
        });
        document.body.appendChild(probe);
        const range = document.createRange();
        range.setStart(probe.firstChild, i);
        range.setEnd(probe.firstChild, i + n);
        const shift = -el.scrollLeft;
        const rects = Array.from(range.getClientRects())
            .filter((r) => r.width > 0)
            .map((r) => ({ left: r.left + shift, right: r.right + shift, top: content.top, bottom: content.bottom }));
        probe.remove();
        return rects;
    };
    // Where s is painted inside a single-line input, as rects in viewport
    // coordinates (one, or one per bidi run), or null when the field is not
    // a single-line input (see "Field text, by where it is painted").
    const inputTextRect = (el, s) => {
        if (el.tagName !== "INPUT") return null;
        const cs = getComputedStyle(el);
        const rtl = cs.direction === "rtl";
        const shown = painted(el)[0];
        const i = shown.indexOf(s);
        if (i < 0) return null;
        const spacing = cs.letterSpacing || "normal";
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
        if (rtl && mixedInRtl.test(shown)) {
            const rects = bidiRects(el, cs, font, spacing, shown, i, s.length, content);
            return rects.length ? { rects, content } : null;
        }
        const whole = measure(font, spacing, shown);
        const room = content.right - content.left;
        // Measured through the end of s rather than as s alone, so kerning
        // across the boundary with the text before it is counted.
        const before = measure(font, spacing, shown.slice(0, i));
        const through = measure(font, spacing, shown.slice(0, i + s.length));
        let rect;
        if (!rtl) {
            // Text that fits is placed by text-align; text that does not
            // starts at the leading edge and moves by the field's own scroll.
            let x0 = content.left - el.scrollLeft;
            if (whole < room) {
                if (cs.textAlign === "center") x0 = content.left + (room - whole) / 2;
                else if (cs.textAlign === "right" || cs.textAlign === "end") x0 = content.right - whole;
            }
            rect = { left: x0 + before, right: x0 + through, top: content.top, bottom: content.bottom };
        } else {
            // The leading edge is the right one. scrollLeft is 0 or negative,
            // and a negative one has moved the text rightward.
            let xr = content.right - el.scrollLeft;
            if (whole < room) {
                if (cs.textAlign === "center") xr = content.left + (room + whole) / 2;
                else if (cs.textAlign === "left" || cs.textAlign === "end") xr = content.left + whole;
            }
            rect = { left: xr - through, right: xr - before, top: content.top, bottom: content.bottom };
        }
        return { rects: [rect], content };
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
                const { rects, content } = placed;
                // FIELD_SLACK, not SLACK: a field's scroll offset is a whole
                // pixel and its text's advances are not, so a value scrolled
                // flush to one end can sit up to a pixel past that edge in
                // the arithmetic while the browser draws it whole (measured
                // on a right-to-left field scrolled to its end: 0.7px).
                const FIELD_SLACK = 1;
                const cutX = rects.some((r) => r.left < content.left - FIELD_SLACK || r.right > content.right + FIELD_SLACK);
                reason = cutX ? "cut sideways by its input" : cutRects(rects, c.parentElement);
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
