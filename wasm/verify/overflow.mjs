// Whether anything in a rendered lesson escapes its box, spills off the
// screen, or is drawn smaller than the size it states: browser check 24.
//
// # Where this came from
//
// A sweep of all 80 tutorial lessons in headless Chrome, at two phone widths
// and in the split view, found every overflow session 2026-0924-1211 fixed:
// long words past their column, an <input>'s intrinsic floor pushing a Send
// button off a row, a slider's user-agent margin, a nested Scroll collapsed
// to its border, a 36px avatar drawn 34 wide. It lived in a scratchpad, so
// nothing held those fixes (N-081). This is that sweep's judgement, kept.
//
// # The split
//
// The browser measures and this file judges. OVERFLOW_READ_JS runs in the
// page and returns one flat record per element that draws a box: its rect,
// its layout size, what it clips, what it states, and which recorded element
// is its parent. overflowFindings is a pure function over those records, so
// every rule below is reachable from overflow_test.mjs with hand-built boxes
// — the same move fold.mjs and startup.mjs made — and a reading that turns
// out wrong is fixed where a test can see it.
//
// # The rules
//
//	escape    an in-flow box outside its parent on an axis where the parent
//	          does not clip. Nothing cuts it off, so it draws over whatever
//	          is beside the parent. Absolutely positioned boxes are exempt:
//	          a badge hung off an avatar's corner is outside its parent on
//	          purpose.
//	clipped   a box outside its nearest clipping ancestor on an axis that
//	          ancestor hides (overflow: hidden or clip) rather than scrolls.
//	          This is lesson 1.1's "Following" cut off by the card's rounded
//	          corners: drawn, and invisible.
//	offscreen a box past the viewport's left or right edge with no clipping
//	          ancestor on that axis. Vertical is not asked: a page is allowed
//	          to be taller than the window.
//	text      a box holding its own text whose scrollWidth passes its
//	          clientWidth while it does not clip: a word that cannot break.
//	squeezed  a box with an inline px width or height (a stated core.Width or
//	          Height) laid out smaller than that, with no inline max-width or
//	          max-height stating that it may be. offsetWidth rather than the
//	          rect, so a scale transition mid-flight is not a squeeze.
//
// # What is skipped, and why
//
// Four kinds of box, with everything inside them:
//
//   - a CodeEditor, which scrolls sideways by design and whose lines are
//     longer than a phone on purpose;
//   - a rotated box (a computed transform with a skew term: the compass
//     rose, a clock's hands), whose bounding rect grows as it turns;
//   - an SVG element below the <svg> itself, whose geometry is the canvas's
//     viewBox and not CSS layout;
//   - an inert subtree (the `inert` attribute core.Inert writes). The one
//     the tutorial has is lesson 4.18's shut Drawer panel, parked beside the
//     screen in a clipping Row until it slides in, and inert and hidden from
//     readers for exactly that reason (comps/drawer.go). It was the sweep's
//     only finding on the tree 2026-0924-1211 left, at all three widths. The
//     rule is the attribute and not the widget, because inert is the app
//     saying "not on the page right now"; a subtree that is inert while
//     visible (a drawer's content layer under an open panel) is only ever
//     reached after a toggle, which this check does not make.
//
// A box that draws nothing (display: none, or no client rects) is not
// recorded at all.
//
// # Reporting once
//
// A row that runs off the screen takes every descendant with it. A box is
// reported for offscreen or clipped only when its parent was not reported for
// the same thing, so a finding names the outermost box that went wrong rather
// than its forty children. escape and squeezed are about one box and its
// parent, and need no such rule.

// OVERFLOW_EPSILON is the slack on every comparison, in CSS pixels. Rects are
// fractional and a flex line distributes remainders, so an edge that lands on
// its parent's edge can arrive a fraction past it. A pixel is below anything
// a reader would see as a spill (the smallest real one fixed was 2px, the
// avatar) and above any rounding.
export const OVERFLOW_EPSILON = 1;

// OVERFLOW_READ_JS is an expression that evaluates, in the page, to
// {viewport, boxes}. Each box is:
//
//	parent    the index of the nearest recorded ancestor, or -1
//	path      data-node-path, or null for a host page's own element
//	type      data-node-type, or the tag name
//	l t r b   the border box's rect, rounded to 1/100 px
//	ox oy     "v" visible, "c" hidden or clip, "s" auto or scroll
//	abs       position is absolute or fixed
//	skip      "editor", "rotated", "svg", "inert", or null — this box's own reason;
//	          overflowFindings extends it to descendants
//	w h       the inline width/height when stated in px, else null
//	ow oh     offsetWidth/offsetHeight, or null for an SVG element
//	capW capH an inline max-width/max-height is stated
//	text      the box's own text (its direct text nodes), trimmed, up to 40
//	          characters, or ""
//	sw cw     scrollWidth and clientWidth, read only for a box with its own
//	          text; clientWidth is 0 for an inline box, which is not asked
//
// root is a CSS selector for the subtree to walk; the tutorial's is #app.
export const OVERFLOW_READ_JS = (root) => `(() => {
    const top = document.querySelector(${JSON.stringify(root)});
    if (!top) return null;
    const round = (n) => Math.round(n * 100) / 100;
    const clipOf = (v) => v === "visible" ? "v" : (v === "hidden" || v === "clip") ? "c" : "s";
    const px = (v) => /^-?[0-9.]+px$/.test(v) ? parseFloat(v) : null;
    const index = new Map();
    const boxes = [];
    const walk = [top, ...top.querySelectorAll("*")];
    for (const el of walk) {
        if (el.getClientRects().length === 0) continue;
        const cs = getComputedStyle(el);
        if (cs.display === "none") continue;
        let parent = -1;
        for (let p = el.parentElement; p && p !== top.parentElement; p = p.parentElement) {
            if (index.has(p)) { parent = index.get(p); break; }
        }
        const r = el.getBoundingClientRect();
        let skip = null;
        if (el.dataset && el.dataset.nodeType === "CodeEditor") skip = "editor";
        else if (el.inert) skip = "inert";
        else if (el instanceof SVGElement && !(el instanceof SVGSVGElement)) skip = "svg";
        else if (cs.transform && cs.transform !== "none") {
            const m = /matrix(?:3d)?\\(([^)]*)\\)/.exec(cs.transform);
            if (m) {
                const v = m[1].split(",").map(Number);
                // matrix(a, b, c, d, e, f) and matrix3d's m12, m21 sit at
                // [1] and [2] of matrix but [1] and [4] of matrix3d.
                const [b, c] = cs.transform.startsWith("matrix3d") ? [v[1], v[4]] : [v[1], v[2]];
                if (Math.abs(b) > 1e-6 || Math.abs(c) > 1e-6) skip = "rotated";
            }
        }
        let text = "";
        for (const n of el.childNodes) if (n.nodeType === 3) text += n.textContent;
        text = text.trim().replace(/\\s+/g, " ").slice(0, 40);
        const html = el instanceof HTMLElement;
        index.set(el, boxes.length);
        boxes.push({
            parent,
            path: (el.dataset && el.dataset.nodePath) || null,
            type: (el.dataset && el.dataset.nodeType) || el.tagName.toLowerCase(),
            l: round(r.left), t: round(r.top), r: round(r.right), b: round(r.bottom),
            ox: clipOf(cs.overflowX), oy: clipOf(cs.overflowY),
            abs: cs.position === "absolute" || cs.position === "fixed",
            skip,
            w: html ? px(el.style.width) : null,
            h: html ? px(el.style.height) : null,
            ow: html ? el.offsetWidth : null,
            oh: html ? el.offsetHeight : null,
            capW: html && el.style.maxWidth !== "",
            capH: html && el.style.maxHeight !== "",
            text,
            sw: text && html ? el.scrollWidth : 0,
            cw: text && html ? el.clientWidth : 0,
        });
    }
    return { viewport: document.documentElement.clientWidth, boxes };
})()`;

// overflowFindings judges one reading. It answers a list of findings, each
// {kind, box, detail}, where box is the offending record; an empty list is a
// lesson with nothing wrong.
export function overflowFindings({ viewport, boxes }) {
    const E = OVERFLOW_EPSILON;
    const skipped = boxes.map(() => false);
    boxes.forEach((b, i) => {
        skipped[i] = Boolean(b.skip) || (b.parent >= 0 && skipped[b.parent]);
    });

    // Per box, per axis: the nearest ancestor that does not show overflow on
    // that axis, or -1 for none. Parents come before children in document
    // order, so one pass fills it.
    const clipX = boxes.map(() => -1);
    const clipY = boxes.map(() => -1);
    boxes.forEach((b, i) => {
        const p = b.parent;
        if (p < 0) return;
        clipX[i] = boxes[p].ox !== "v" ? p : clipX[p];
        clipY[i] = boxes[p].oy !== "v" ? p : clipY[p];
    });

    const out = [];
    const flagged = { offscreen: new Set(), clipped: new Set() };
    const outside = (b, a, axis) => axis === "x"
        ? b.l < a.l - E || b.r > a.r + E
        : b.t < a.t - E || b.b > a.b + E;
    const span = (b, axis) => axis === "x"
        ? `x ${b.l}..${b.r}`
        : `y ${b.t}..${b.b}`;

    boxes.forEach((b, i) => {
        if (skipped[i]) return;
        const p = b.parent >= 0 ? boxes[b.parent] : null;

        if (p && !b.abs) {
            for (const axis of ["x", "y"]) {
                const shows = axis === "x" ? p.ox === "v" : p.oy === "v";
                if (shows && outside(b, p, axis)) {
                    out.push({ kind: "escape", box: b,
                        detail: `${span(b, axis)} outside its parent ${p.type}'s ${span(p, axis)}` });
                }
            }
        }

        for (const axis of ["x", "y"]) {
            const a = axis === "x" ? clipX[i] : clipY[i];
            if (a >= 0) {
                const anc = boxes[a];
                const hides = (axis === "x" ? anc.ox : anc.oy) === "c";
                if (hides && outside(b, anc, axis) && !flagged.clipped.has(b.parent)) {
                    flagged.clipped.add(i);
                    out.push({ kind: "clipped", box: b,
                        detail: `${span(b, axis)} cut off by ${anc.type}'s ${span(anc, axis)}` });
                }
            } else if (axis === "x" && (b.l < -E || b.r > viewport + E) && !flagged.offscreen.has(b.parent)) {
                flagged.offscreen.add(i);
                out.push({ kind: "offscreen", box: b,
                    detail: `${span(b, "x")} past a ${viewport}px viewport` });
            }
        }
        // A box inherits its parent's flag so a finding is reported once, at
        // the outermost box, even when an unflagged child sits between.
        if (b.parent >= 0 && flagged.clipped.has(b.parent)) flagged.clipped.add(i);
        if (b.parent >= 0 && flagged.offscreen.has(b.parent)) flagged.offscreen.add(i);

        if (b.text && b.ox === "v" && b.cw > 0 && b.sw > b.cw + E) {
            out.push({ kind: "text", box: b,
                detail: `its text is ${b.sw}px wide in a ${b.cw}px box` });
        }

        if (b.w !== null && b.ow !== null && !b.capW && b.ow + E / 2 < b.w) {
            out.push({ kind: "squeezed", box: b, detail: `drawn ${b.ow}px wide, stated ${b.w}px` });
        }
        if (b.h !== null && b.oh !== null && !b.capH && b.oh + E / 2 < b.h) {
            out.push({ kind: "squeezed", box: b, detail: `drawn ${b.oh}px tall, stated ${b.h}px` });
        }
    });
    return out;
}

// overflowLine is one finding as a failure reads: where, what, and enough of
// the box to find it in the lesson.
export function overflowLine(where, f) {
    const b = f.box;
    const name = b.path ? `${b.type} ${b.path}` : b.type;
    const text = b.text ? ` "${b.text}"` : "";
    return `${where}: ${f.kind}: ${name}${text}: ${f.detail}`;
}
