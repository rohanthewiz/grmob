// The facts a shimmed DOM cannot check, checked in a browser: four about the
// keyboard, two about paint, four about layout, and one about what a browser
// does with an accessibility value nobody here resolves.
//
// wasm/verify's other suites run the real grmob-runtime.js against dom.mjs — a
// few hundred lines that model element trees, attributes, listeners and which
// element holds focus. That is enough for almost everything, and its limits
// are stated in its own header: there is no layout, no bubbling, and `focus()`
// is an assignment, and nothing is ever painted. Eleven claims sit exactly in
// that blind spot, and no amount of widening the shim would settle them,
// because each one is a claim about what a *browser* does:
//
//   1. tabindex="-1" takes a <button> out of the tab order. The roving
//      tabindex is the whole reason a tab strip is one stop rather than five;
//      in dom.mjs the attribute is a string nobody reads.
//   2. a disabled control refuses focus. The runtime relies on this to keep a
//      disabled member from being landed on; dom.mjs's focus() assigns.
//   3. preventDefault on ArrowDown stops the scroll. A listbox that moved its
//      selection *and* scrolled the page under it would be unusable, and
//      defaultPrevented in a shim is a flag the shim set itself.
//   4. a toolbar of plain <button>s is one tab stop, and the arrows reach the
//      rest. The other half of the roving tabindex: check 1 says the members
//      are out of the tab order, and this says they are still reachable — by
//      the browser's own focus algorithm rather than by an assignment.
//   5. the palette reaches the screen. core.ColorPalette.ControlBorder has
//      WCAG 1.4.11's 3:1 floor under it and components/variant_test.go
//      measures every pair — as arithmetic over hex strings, which is all Go
//      can do. Two retints and a whole third palette later, no pass had ever
//      *looked* at the result. This one paints the pairs and reads the pixels
//      back out of a screenshot, which is the only place an alpha channel, a
//      colour profile or a hairline antialiased into a tint can be caught.
//   6. a sticky band stays put while the rows scroll under it.
//      core.StickyHeader() writes position:sticky, top:0 and z-index:1, and
//      dom.mjs can say those three landed on the element and nothing more: it
//      has no layout at all. "The property is written" is exactly what stays
//      true when the box around it defeats the pin — an ancestor with overflow
//      other than visible, a flex item shrunk to its container, a containing
//      block that is not the scroller.
//   7. a real widget draws the palette. Check 5 paints the census's pairs as
//      boxes this file builds — a model of a control boundary, and a good one.
//      Everything between the palette role and a chip's actual ring goes
//      through `components`, which is Go, so a widget that had stopped
//      declaring the boundary tone would leave every swatch painting perfectly.
//      gen.go renders a real components.Chip and a real core.Input through each
//      bundled theme and reads the colours off the rendered nodes; this mounts
//      those trees and reads the pixels back. The two are the tone's two
//      spenders and they read it from two different places in Go, which is why
//      the field is worth painting rather than assumed from the chip.
//   8. a browser applies ARIA's own rules to a value range.
//      This is the one claim here that is not about the runtime at all. Both
//      DOM exporters deliberately do not implement the implicit 0..100, the
//      indeterminate spelling or the clamping — they write aria-valuenow and
//      its two bounds verbatim, on the argument that a browser applies those
//      rules itself. core.ValueRange.Progress states them for the platforms
//      that do not (Compose, through android/verify's JVM pass), so the
//      repository held the rule to one target and asserted the web's half by
//      reasoning. This asks Chrome, through its own accessibility tree.
//   9. two arrangements of the same band lay out the same way, overflow
//      included. components.GroupHeader moved its padding from the Row onto
//      the growing control inside it so that a press lands on the whole band,
//      and the warrant for the move is that it costs nothing. ios/verify checks
//      that through GrMobFlexSolver and records one place it is not free: under
//      an offer narrower than the band, that solver shrinks each child in
//      proportion to a base that *includes* the child's own padding, so the
//      same 16 points is inside the proportion in one arrangement and outside
//      it in the other. CSS distributes shrink over the inner flex base size
//      instead, which is a different rule — and the comment recording that had
//      never been asked of a browser. This asks one, and the answer is a
//      genuine cross-target divergence rather than an artefact.
//  10. a real band's tap target spans it, and its control is its tallest child.
//      Check 9 is the band as arithmetic over synthetic sizes, which is the
//      right shape for a distribution and cannot reach two things. One is
//      cross-axis: the disclosure branch puts the insets on a button one level
//      inside the Row's growing heading wrapper, so whether a press lands on
//      the whole band depends on whether a non-growing child fills a growing
//      parent, and GrMobFlexSolver is a main-axis distributor. The other is
//      about text: every SameHeight case rests on the padded control being the
//      band's tallest child, half of which is "a bold caption is no shorter
//      than a plain one" — a measurement no Go test can take. gen.go renders
//      real components.GroupHeaders through every bundled theme and this mounts
//      them with real glyphs in them.
//  11. a fixed-size container squeezes its child along its main axis and lets
//      it spill across. core.Spacer became "a Box with a fixed size", and the
//      note closing that work recorded that Compose constrains a child to the
//      declared size where the DOM was believed to let it spill — for every
//      fixed-size container, not just a Spacer — with nothing anywhere having
//      asked. This asks, and the DOM's answer is per-axis rather than the
//      blanket one assumed: squeezed along the main axis (a flex item's shrink
//      factor defaults to 1 and an empty box has no automatic minimum to stop
//      at), and spilling across the cross one.
//
// The numbering is one sequence, and it is the order the checks run in rather
// than the order they were written. It is also load-bearing: a dozen comments
// across four languages cite these by number, so the list above and the markers
// in main() are held to each other by
// TestTheBrowserChecksAreOneNumberedSequence in checknumbering_test.go — which
// is what a numbering nobody could renumber safely was missing.
//
// # How
//
// A headless Chrome, driven over the DevTools protocol with Node's built-in
// WebSocket. No npm, no lockfile, no node_modules — the promise wasm/verify's
// run.sh makes — and no network: the page is served from a localhost HTTP
// server over the repository's own files, and the browser is one that is
// already installed.
//
// Keys are dispatched through Input.dispatchKeyEvent, which is the browser's
// real input path: the tab order is walked by the browser's own focus
// algorithm and a scroll is a real scroll. Nothing here synthesizes an event
// object, which is the point — a synthesized event is what dom.mjs already
// does.
//
// # When it does not run
//
// If there is no Chrome to launch, or the Node in use has no WebSocket global
// (before v21), the pass prints SKIP and exits 0. That is the same stance
// ios/verify takes toward a missing iPhoneOS SDK: the script's promise is that
// it catches what this machine can catch, and a pass that fails on a machine
// missing an optional tool is a pass people learn to ignore.

import { spawn } from "node:child_process";
import { mkdtempSync, writeFileSync, readFileSync, existsSync } from "node:fs";
import { tmpdir } from "node:os";
import { join, dirname } from "node:path";
import { fileURLToPath } from "node:url";
import http from "node:http";
import zlib from "node:zlib";

import { PALETTES } from "./palette.mjs";
import { VALUE_RANGES, axRange, valueRangeProblem } from "./valuerange.mjs";
import { startupVerdict } from "./startup.mjs";

// The widget swatches come from the transcript rather than from a .mjs table,
// because they are real components rendered by Go: gen.go builds the trees and
// reads their colours off the rendered nodes, which is the whole point (see
// widgetCase there). run.sh generates that file and points every consumer at
// it, this one included.
//
// A hard requirement rather than an optional extra, and the reason startup.mjs
// exists: the other checks here skip when the machine has no Chrome, which is a
// fact about the machine, and a missing transcript is a fact about how this
// script was invoked. The two stances and their order are stated there, where a
// test can reach every answer without arranging a machine that has the fault.
const TRANSCRIPT = process.env.GRMOB_TRANSCRIPT;
const TRANSCRIPT_EXISTS = Boolean(TRANSCRIPT) && existsSync(TRANSCRIPT);
const TRANSCRIPT_JSON = TRANSCRIPT_EXISTS
    ? JSON.parse(readFileSync(TRANSCRIPT, "utf8"))
    : {};
const WIDGETS = TRANSCRIPT_JSON.widgets || [];
// internal/bandfixture, for check 9. Same file, same reason: a real
// components.GroupHeader's geometry, read off the rendered band by Go, which is
// not something a table of numbers in a .mjs file could be.
const BANDS = TRANSCRIPT_JSON.bands || [];
// Real components.GroupHeaders, for check 10. Same file, same reason as the
// three tables above it — and a different subject from BANDS, which is the same
// band as arithmetic over synthetic sizes. See bandRender in gen.go: these are
// the two band claims that are measurements of a rendered widget with glyphs in
// it rather than of a distribution, and neither has a target but this one.
const BAND_RENDERS = TRANSCRIPT_JSON.bandRenders || [];

const HERE = dirname(fileURLToPath(import.meta.url));
const RUNTIME = join(HERE, "..", "grmob-runtime.js");

// Where a Chrome might be. The env var wins so a machine with an unusual
// install (or a Chromium) can point at it without editing this list.
const CHROME_CANDIDATES = [
    process.env.GRMOB_CHROME,
    "/Applications/Google Chrome.app/Contents/MacOS/Google Chrome",
    "/Applications/Chromium.app/Contents/MacOS/Chromium",
    "/usr/bin/google-chrome",
    "/usr/bin/chromium",
    "/usr/bin/chromium-browser",
].filter(Boolean);

function findChrome() {
    return CHROME_CANDIDATES.find((p) => existsSync(p)) || null;
}

// Carry out startup.mjs's decision. The stances live there; what is here is
// the exit code and the stream each one is written to.
function actOn(verdict) {
    if (verdict.action === "fail") {
        console.error(`FAIL: ${verdict.why}`);
        process.exit(1);
    }
    if (verdict.action === "skip") {
        console.log(`SKIP: browser keyboard pass (${verdict.why})`);
        process.exit(0);
    }
}

// --------------------------------------------------------------------------
// The page
// --------------------------------------------------------------------------

// A sentinel button either side of the mount point, which is what makes the
// tab order measurable: "the strip is one stop" is a statement about where
// focus goes *next*, so there has to be a next.
//
// The tall filler is what makes the scroll measurable. A document that cannot
// scroll would pass check 3 whether or not the key was consumed.
const PAGE = `<!DOCTYPE html>
<html><head><meta charset="utf-8"><title>grmob keyboard</title></head>
<body>
<button id="before">before</button>
<div id="app"></div>
<button id="after">after</button>
<div style="height: 4000px"></div>
<script>
  // The bridge the runtime dispatches into. It is never asserted on here —
  // the dispatch path has its own suites — but it has to exist, or a click
  // the browser generates while walking the tab order throws.
  window.__dispatched = [];
  window.GoInvokeCallback = (id, payload) => window.__dispatched.push({ id, payload });
</script>
<script src="/grmob-runtime.js"></script>
</body></html>
`;

// --------------------------------------------------------------------------
// PNG
// --------------------------------------------------------------------------

// A minimal PNG decoder, because a screenshot arrives as one and the only
// question worth asking of it is the colour of a pixel.
//
// # Why hand-rolled
//
// wasm/verify's run.sh promises Go and Node and nothing else — no npm, no
// lockfile, no node_modules. A decoder is the price of that promise, and the
// price is small: Chrome's screenshots are 8-bit, non-interlaced, truecolour
// with or without alpha, `node:zlib` is built in, and the five scanline
// filters are the whole of the format that matters here.
//
// # What it refuses
//
// Everything else, loudly. A palette-indexed, 16-bit or interlaced image would
// decode to plausible garbage under a decoder that guessed, and this check's
// entire output is "the pixel is #89898E" — a wrong answer is worse here than
// no answer, because it is a contrast claim about a colour nothing painted.
function decodePNG(buf) {
    const SIG = [0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a];
    for (let i = 0; i < SIG.length; i++) {
        if (buf[i] !== SIG[i]) throw new Error("not a PNG");
    }

    let width = 0, height = 0, bitDepth = 0, colorType = 0;
    const idat = [];
    for (let off = 8; off + 8 <= buf.length;) {
        const len = buf.readUInt32BE(off);
        const type = buf.toString("ascii", off + 4, off + 8);
        const data = buf.subarray(off + 8, off + 8 + len);
        if (type === "IHDR") {
            width = data.readUInt32BE(0);
            height = data.readUInt32BE(4);
            bitDepth = data[8];
            colorType = data[9];
            if (data[12] !== 0) throw new Error("interlaced PNG");
        } else if (type === "IDAT") {
            idat.push(data);
        } else if (type === "IEND") {
            break;
        }
        off += 12 + len; // length + type + data + CRC
    }
    if (bitDepth !== 8) throw new Error(`PNG bit depth ${bitDepth}, want 8`);
    const channels = { 0: 1, 2: 3, 4: 2, 6: 4 }[colorType];
    if (!channels) throw new Error(`PNG colour type ${colorType} is not truecolour or grey`);

    const raw = zlib.inflateSync(Buffer.concat(idat));
    const stride = width * channels;
    const out = Buffer.alloc(height * stride);
    let prev = Buffer.alloc(stride);

    // Unfiltering. Each scanline is preceded by a filter byte and is decoded
    // against the pixel to its left (a), the one above (b) and the one above
    // left (c) — all of them already-decoded bytes, which is why this runs in
    // place and in order.
    for (let y = 0; y < height; y++) {
        const rowStart = y * (stride + 1);
        const filter = raw[rowStart];
        const line = raw.subarray(rowStart + 1, rowStart + 1 + stride);
        const cur = out.subarray(y * stride, (y + 1) * stride);
        for (let i = 0; i < stride; i++) {
            const a = i >= channels ? cur[i - channels] : 0;
            const b = prev[i];
            const c = i >= channels ? prev[i - channels] : 0;
            let v = line[i];
            switch (filter) {
                case 0: break;
                case 1: v += a; break;
                case 2: v += b; break;
                case 3: v += (a + b) >> 1; break;
                case 4: {
                    // Paeth: whichever of a, b, c the linear predictor a+b-c
                    // is closest to, ties going to a then b.
                    const p = a + b - c;
                    const pa = Math.abs(p - a), pb = Math.abs(p - b), pc = Math.abs(p - c);
                    v += (pa <= pb && pa <= pc) ? a : (pb <= pc ? b : c);
                    break;
                }
                default: throw new Error(`PNG scanline filter ${filter}`);
            }
            cur[i] = v & 0xff;
        }
        prev = cur;
    }
    return { width, height, channels, data: out };
}

const hex2 = (n) => n.toString(16).padStart(2, "0").toUpperCase();

// The colour at one pixel, as "#RRGGBB", or null outside the image.
//
// Greyscale images are expanded to three equal channels rather than refused:
// a screenshot of black-and-white swatches is a legitimate thing for Chrome to
// hand back, and every colour this check compares against is spelled in six
// digits.
function pixelAt(img, x, y) {
    x = Math.round(x);
    y = Math.round(y);
    if (x < 0 || y < 0 || x >= img.width || y >= img.height) return null;
    const i = (y * img.width + x) * img.channels;
    const d = img.data;
    if (img.channels <= 2) return `#${hex2(d[i])}${hex2(d[i])}${hex2(d[i])}`;
    return `#${hex2(d[i])}${hex2(d[i + 1])}${hex2(d[i + 2])}`;
}

// --------------------------------------------------------------------------
// CDP
// --------------------------------------------------------------------------

function getJSON(url) {
    return new Promise((resolve, reject) => {
        http.get(url, (r) => {
            let body = "";
            r.on("data", (c) => (body += c));
            r.on("end", () => {
                try { resolve(JSON.parse(body)); } catch (e) { reject(e); }
            });
        }).on("error", reject);
    });
}

const sleep = (ms) => new Promise((r) => setTimeout(r, ms));

// Chrome writes its chosen port into the profile directory when started with
// --remote-debugging-port=0, which is how this avoids picking a fixed port
// that another process (or another copy of this script) might hold.
async function devtoolsPort(profile) {
    const portFile = join(profile, "DevToolsActivePort");
    for (let i = 0; i < 100; i++) {
        if (existsSync(portFile)) {
            const first = readFileSync(portFile, "utf8").split("\n")[0].trim();
            if (first) return Number(first);
        }
        await sleep(100);
    }
    throw new Error("Chrome never reported a DevTools port");
}

/** A single page target's CDP session, as a send(method, params) function. */
// How long a DevTools call may take before the harness gives up on it. See
// send.
const CDP_TIMEOUT_MS = 15000;

async function connect(wsURL) {
    const ws = new WebSocket(wsURL);
    await new Promise((resolve, reject) => {
        ws.addEventListener("open", resolve, { once: true });
        ws.addEventListener("error", reject, { once: true });
    });

    let nextID = 0;
    const pending = new Map();
    const events = new Map();
    ws.addEventListener("message", (e) => {
        const msg = JSON.parse(e.data);
        if (msg.id !== undefined) {
            const entry = pending.get(msg.id);
            if (!entry) return;
            pending.delete(msg.id);
            if (msg.error) entry.reject(new Error(`${entry.method}: ${msg.error.message}`));
            else entry.resolve(msg.result);
            return;
        }
        const waiter = events.get(msg.method);
        if (waiter) { events.delete(msg.method); waiter(msg.params); }
    });

    return {
        // Every round trip is bounded.
        //
        // A CDP call that never gets an answer is indistinguishable from one
        // still in flight, so without this the harness's failure mode is a hang
        // rather than a failure — and a hang is strictly worse: it is what a
        // CI run does for its whole timeout instead of reporting in a minute.
        // The number is generous next to a round trip that normally takes
        // single-digit milliseconds; what it is sized against is a page that
        // will never answer at all.
        send(method, params = {}) {
            const id = ++nextID;
            return new Promise((resolve, reject) => {
                const timer = setTimeout(() => {
                    pending.delete(id);
                    reject(new Error(`${method}: no answer from the page in ${CDP_TIMEOUT_MS}ms`));
                }, CDP_TIMEOUT_MS);
                pending.set(id, {
                    method,
                    resolve: (v) => { clearTimeout(timer); resolve(v); },
                    reject: (e) => { clearTimeout(timer); reject(e); },
                });
                ws.send(JSON.stringify({ id, method, params }));
            });
        },
        once(method) {
            return new Promise((resolve) => events.set(method, resolve));
        },
        close: () => ws.close(),
    };
}

// --------------------------------------------------------------------------
// The trees under test
// --------------------------------------------------------------------------

// A tab strip of real <button> elements: three members, the middle one
// selected, so the roving tabindex is "-1", "0", "-1". Buttons rather than
// boxes on purpose — a <button> is focusable *by default*, so this is the
// case where tabindex="-1" has something to take away.
const TABLIST = {
    Type: "Row",
    Style: { AccessibilityRole: "tablist" },
    Children: [0, 1, 2].map((i) => ({
        Type: "Button",
        Style: {
            AccessibilityRole: "tab",
            AccessibilitySelected: i === 1 ? "true" : "false",
        },
        Props: { label: `Tab ${i}`, onClick: `cb_${i}` },
    })),
};

// A listbox of options, the first selected. Divs, which is what a row of
// content is on every screen this pattern was written for.
const LISTBOX = {
    Type: "Column",
    Style: { AccessibilityRole: "listbox" },
    Children: [0, 1, 2].map((i) => ({
        Type: "Box",
        Style: {
            AccessibilityRole: "option",
            AccessibilitySelected: i === 0 ? "true" : "false",
        },
        Props: {},
    })),
};

// A filter bar: a Row carrying role="toolbar" over three real <button> chips,
// which is what components.ChipStrip renders and what a caller puts the role on.
//
// This is the fixture the tab-order claim needs a browser for, and it is a
// stronger case than the tablist above. A tablist's members are <button
// role="tab"> and their tab stops are governed by the same tabindex attribute;
// what is new here is that the toolbar's members are not named by its role at
// all — the runtime found them by walking for focusable controls — so this
// check is the one that says the *walk* found the right three elements and not
// merely that tabindex works.
//
// Deliberately no roles on the chips. A chip is a plain core.Button, so if
// focusableMembers only recognised elements carrying a role it would find none
// here and the strip would keep all three of its stops.
const TOOLBAR = {
    Type: "Row",
    Style: { AccessibilityRole: "toolbar" },
    Children: ["All", "Sermons", "Articles"].map((label, i) => ({
        Type: "Button",
        Style: {},
        Props: { label, onClick: `cb_${i}` },
    })),
};

// One enabled button and one disabled one, side by side with no composite
// around them — so what is being measured is the control's own refusal and
// not a roving tabindex.
const BUTTONS = {
    Type: "Row",
    Children: [
        { Type: "Button", Style: {}, Props: { label: "live" } },
        { Type: "Button", Style: { Disabled: true }, Props: { label: "dead" } },
    ],
};

// A grid of palette swatches, one per row of palette.mjs.
//
// # The shape of a swatch, and why it has two boxes
//
//	┌─ outer: 120x52, filled with the backdrop ─┐
//	│                                           │   the fill sample is taken
//	│    ┌─ inner: 80x24, same fill, 1px ─┐     │   here, clear of the inner
//	│    │  border in the ControlBorder   │     │   box
//	│    └────────────────────────────────┘     │
//	└───────────────────────────────────────────┘
//
// The inner box is filled with the *same* colour as the outer one, so the
// border has the backdrop on both sides — which is what the census's number is
// about. It is 1px because that is what core.Theme's Input and TextArea
// frames are.
//
// What the border sample catches is a *blended* edge rather than a thin one.
// A border drawn in a translucent colour, or composited under an opacity, or
// dropped entirely by a guard, leaves no pixel that is the tone — and the
// contrast a reader gets is then not the contrast the census computed. A
// border declared narrower than a device pixel is a different story: Chrome
// snaps a solid sub-pixel border up to one full-strength pixel at dpr 1, so
// that mutation changes no pixel and this check correctly reports nothing.
//
// A `role` row has no backdrop, so its swatch is the tone alone and only the
// centre is sampled. It is what makes the tone's own hex a measured fact
// rather than something inferred from a border that happened to look right.
// A pinned band over rows that scroll under it — the first thing this pass
// asks that is about *layout* rather than about the keyboard or a colour.
//
// core.StickyHeader() writes three declarations (position:sticky, top:0,
// z-index:1) and the framework's claim for them is that a List child stays put
// while the rows move. Nothing in wasm/verify could ever have looked at that:
// dom.mjs has no layout at all, so a suite there can assert the three
// properties were written and nothing more — which is a check on the style
// mapping, not on the pin. position:sticky is also the property most likely to
// be silently defeated by the box around it: an ancestor with overflow other
// than visible, a flex item shrunk to its container, a containing block that
// is not the scroller. Every one of those leaves the declarations exactly as
// written and the band scrolling away.
//
// Two colours because the pin is checked twice: once through the rects the
// browser reports, and once through the pixels, at a point where an unpinned
// band would have left a row behind. A rect can be right while nothing is on
// screen at it.
const BAND_FILL = "#123456";
const ROW_FILL = "#ABCDEF";
const STICKY_ROWS = 6;

// core.StickyHeader()'s declarations, as this file has to state them.
//
// It cannot import them. browser.mjs mounts JSON trees through the real
// runtime, which is the whole point — what is being checked is what these
// three do in a browser, not what Go writes — so the values live here as
// literals the way palette.mjs's hexes do.
//
// And like palette.mjs's hexes, they are pinned rather than transcribed:
// wasm/verify/sticky_test.go applies core.StickyHeader() to an empty Style and
// requires this object to be exactly the fields it set, with exactly those
// values. A fourth declaration added to the Go prop, or one of these three
// changed, fails there — which is the failure that would otherwise leave this
// pass green while checking a pin the framework no longer writes.
const STICKY_DECLARATIONS = { Position: "sticky", Top: "0", ZIndex: 1 };

// core.ShrinkNone: how a core.Style spells a flex-shrink factor of ZERO.
//
// A mounted JSON tree carries the field as written, and a literal 0 there is
// what the runtime reads as "nothing was set" — every other number in a
// core.Style means unset by being zero, and flex-shrink is the one whose CSS
// initial value is not. So `FlexShrink: 0` in a fixture is a declaration the
// runtime discards, and the sticky fixture below carried exactly that for as
// long as it has existed: its comment said the List must not be compressed and
// the number saying so did nothing.
//
// Pinned rather than transcribed, by sticky_test.go, for the same reason the
// three declarations above are: a fixture spelling a contract the framework has
// changed is a pass that has stopped being about the framework.
const SHRINK_NONE = -1;

const STICKY = {
    Type: "Scroll",
    Style: {
        Width: "300px", Height: "160px", Overflow: "auto",
        Padding: { Top: 0, Right: 0, Bottom: 0, Left: 0 }, Gap: 0,
    },
    Children: [{
        Type: "List",
        // A shrink factor of zero because the Scroll is a flex column and its
        // child would otherwise be compressed to fit rather than overflowing
        // it — at which point there is nothing to scroll and the check would
        // pass by having no subject. The control assertions below say so out
        // loud.
        //
        // This said `FlexShrink: 0` for as long as it existed, and that number
        // did nothing: the runtime read a literal zero as "unset" and wrote no
        // declaration. It works now (see SHRINK_NONE above and core.ShrinkNone)
        // and it is still not what produces the overflow — the List measures
        // 400px in a 160px port with or without it, because a flex item's
        // automatic minimum size is content-based and these rows carry text. It
        // is kept as a statement of intent and the check no longer rests on it;
        // the assertion in check 6 pins the arrangement directly.
        Style: {
            Padding: { Top: 0, Right: 0, Bottom: 0, Left: 0 }, Gap: 0,
            FlexShrink: SHRINK_NONE,
        },
        Children: [
            {
                Type: "Box",
                Style: {
                    Height: "40px", Background: BAND_FILL,
                    // Spread rather than written out again: there is one
                    // statement of the pin in this file and sticky_test.go
                    // holds that statement to core.StickyHeader().
                    ...STICKY_DECLARATIONS,
                },
            },
            ...Array.from({ length: STICKY_ROWS }, () => ({
                Type: "Box",
                Style: { Height: "60px", Background: ROW_FILL },
            })),
        ],
    }],
};

// One progressbar per row of valuerange.mjs, named by the case so the
// accessibility tree can be read back by name.
//
// Boxes rather than any widget: what is under test is what a *browser* does
// with the four aria-value* attributes, so the tree has to be the attributes
// and nothing else. components.ProgressBar would bring a fill, a track and a
// theme, none of which the accessibility tree can see, and would only ever
// produce the one range it builds by construction.
//
// The whole table mounts at once and the accessibility tree is read once, which
// is what keeps twenty-two cases to a single round trip.
const VALUE_BARS = {
    Type: "Column",
    Style: { Padding: { Top: 0, Right: 0, Bottom: 0, Left: 0 }, Gap: 0 },
    Children: VALUE_RANGES.map((row) => ({
        Type: "Box",
        Style: {
            Height: "6px",
            AccessibilityRole: "progressbar",
            // The case name is the accessible name, which is how a row is
            // found again in the tree Chrome computes. internal/valuefixture
            // guarantees they are unique (TestEveryCaseIsNamedOnce).
            AccessibilityLabel: row.name,
            AccessibilityValue: row.wire,
        },
        Props: {},
    })),
};

const SWATCHES_PER_ROW = 6;

const SWATCHES = {
    Type: "Column",
    Style: { Padding: { Top: 0, Right: 0, Bottom: 0, Left: 0 }, Gap: 0 },
    Children: Array.from(
        { length: Math.ceil(PALETTES.length / SWATCHES_PER_ROW) },
        (_, r) => ({
            Type: "Row",
            Style: { Padding: { Top: 0, Right: 0, Bottom: 0, Left: 0 }, Gap: 0 },
            Children: PALETTES
                .slice(r * SWATCHES_PER_ROW, (r + 1) * SWATCHES_PER_ROW)
                .map((p) => ({
                    Type: "Box",
                    Style: { Width: "120px", Height: "52px", Background: p.hex },
                    Children: p.kind === "role" ? [] : [{
                        Type: "Box",
                        Style: {
                            Width: "80px", Height: "24px",
                            Margin: { Top: 14, Right: 20, Bottom: 14, Left: 20 },
                            Background: p.hex,
                            BorderColor: PALETTES.find(
                                (q) => q.theme === p.theme && q.kind === "role").hex,
                            BorderWidth: 1,
                        },
                    }],
                })),
        })),
};

// Where swatch i sits in the mounted tree. The runtime stamps every element
// with its path, so the coordinates come from the browser's own layout rather
// than from this file guessing at one.
const swatchPath = (i) =>
    `root/${Math.floor(i / SWATCHES_PER_ROW)}/${i % SWATCHES_PER_ROW}`;

// The widget swatches, all of them, on one page.
//
// Each of gen.go's trees is already a page: a Box carrying its theme's own
// Colors.Background, 24px of padding and a 240px width, with the widget alone
// inside it. They were mounted one at a time and screenshot one at a time,
// which is three CDP round trips and a PNG decode per case — the slowest thing
// in this file by a wide margin, and it doubled the day the Input swatch was
// added. Laid out as a grid they are one mount, one screenshot and one decode
// for the whole census.
//
// The trees themselves are untouched, which is the part that matters: what is
// asserted is that each widget's *own* declarations reach the screen, and a
// tree this file had edited would be a different widget. What changes is only
// where they sit. The theme's page fill is now a sibling's fill rather than the
// document's, and every sample here was already taken inside a rect the browser
// reported — the page inside its own 24px padding, the widget's fill in its
// leading padding, the ring on its own horizontal edges — so none of them was
// ever reading the document behind it.
//
// Three per row keeps the grid inside the 800px viewport (3 x 240 = 720, and
// --hide-scrollbars means nothing takes a column back), which is what
// captureBeyondViewport: false requires: a swatch below the fold has a rect and
// no pixels.
const WIDGETS_PER_ROW = 3;

const WIDGET_GRID = {
    Type: "Column",
    Style: { Padding: { Top: 0, Right: 0, Bottom: 0, Left: 0 }, Gap: 0 },
    Children: Array.from(
        { length: Math.ceil(WIDGETS.length / WIDGETS_PER_ROW) },
        (_, r) => ({
            Type: "Row",
            // AlignItems flex-start so a short swatch beside a tall one keeps
            // its own height. Under the default stretch the two page boxes in a
            // row would be the same height, which paints more of a theme's
            // Background than the theme asked for — harmless to every sample
            // taken here, and still a layout no widgetCase describes.
            Style: {
                Padding: { Top: 0, Right: 0, Bottom: 0, Left: 0 }, Gap: 0,
                AlignItems: "flex-start",
            },
            Children: WIDGETS
                .slice(r * WIDGETS_PER_ROW, (r + 1) * WIDGETS_PER_ROW)
                .map((w) => JSON.parse(w.tree)),
        })),
};

// Where widget i's page box sits in that grid, and its widget one level in.
const widgetPath = (i) =>
    `root/${Math.floor(i / WIDGETS_PER_ROW)}/${i % WIDGETS_PER_ROW}`;

// --------------------------------------------------------------------------
// The bands
// --------------------------------------------------------------------------
//
// components.GroupHeader's two inset arrangements, laid out by a real browser
// at every offer internal/bandfixture states.
//
// # The claim, and the one target that had never been asked
//
// The band's padding used to be on the Row and is now on the growing control
// inside it, so that a press lands on the whole band rather than on a strip in
// the middle of it. The warrant for the move is that it costs nothing: padding
// on a stretched child fills exactly the space the same padding on its parent
// held.
//
// That was verified on the web by the pixels it did not move — by hand, once,
// by a person looking at a screen. ios/verify turned it into a check, solving
// both arrangements through GrMobFlexSolver, and found the one place it is not
// free: under an offer narrower than the band's own content, that solver shrinks
// each child in proportion to a *base that includes the child's own padding*, so
// the control's insets are inside the proportion in one arrangement and outside
// it in the other and the label ends up with different room.
//
//	120pt offered, a 100pt label and a 24pt badge
//	  the SwiftUI solver   insets on the control  63.14   insets on the Row  64.52
//
// The note beside that arithmetic says CSS distributes shrink over the inner
// flex base size rather than the outer one, so a browser may well agree where
// this does not — and that nothing in the repository had asked one. This is the
// asking. It is the cheapest kind of check to have been missing: wasm/verify
// already mounts trees through a real runtime and already reads rects back.
//
// # The two modelling decisions, and why each is the faithful one
//
// min-width: 0 on both flex items. CSS gives a flex item an automatic minimum
// size — it will not shrink below its own min-content width — and the fixture's
// label is a synthetic box with a *declared* width, so its min-content is that
// whole width and nothing would shrink at all. Both arrangements would then
// agree by never reaching the arithmetic, which is agreement with no subject.
// A real band's label is text, whose min-content is its longest word, so
// clearing the automatic minimum is what restores the behaviour the fixture is
// standing in for. It is also what the SwiftUI solver does: it has no notion of
// a content-based floor.
//
// An indefinite offer becomes width: max-content. bandfixture spells "no
// definite offer" as a negative number, which is SwiftUI probing for an ideal
// size; max-content is the same question in CSS, and the two arrangements have
// to agree about what they hug to as well.
//
// That second one was a judgement and is now a measurement. min-content is the
// other candidate and asks a different question, so check 9 mounts every
// intrinsic keyword and holds each to the band's natural width: they agree on
// this fixture, which is what made the choice safe, and it is a property of the
// fixture rather than of CSS — nothing here has text to wrap.
const bandEdges = (i) => ({ Top: i.top, Right: i.right, Bottom: i.bottom, Left: i.left });

const bandPx = (n) => `${n}px`;

// One band, one arrangement, one offer.
//
// The shape mirrors solveBand in ios/verify/band.swift exactly: a Row carrying
// the arrangement's own padding and gap, a growing control carrying the
// arrangement's control padding with the label inside it, and a badge beside it
// when the case has one.
function bandTree(a, c, offer) {
    const children = [{
        Type: "Box",
        Style: {
            Padding: bandEdges(a.control),
            FlexGrow: a.grow,
            FlexShrink: 1,
            MinWidth: "0",
        },
        Children: [{
            Type: "Box",
            Style: { Width: bandPx(c.label.w), Height: bandPx(c.label.h) },
        }],
    }];
    if (c.badge.w > 0) {
        children.push({
            Type: "Box",
            Style: {
                Width: bandPx(c.badge.w), Height: bandPx(c.badge.h),
                // Shrinkable, because the solver this is compared with shrinks
                // it too: its growth weight is 0 and its shrink share is its
                // base like everything else's. The cleared minimum is a
                // statement of that intent rather than a load-bearing
                // declaration — this box is empty, so its own automatic
                // minimum is already 0, where the control's is the whole of
                // the label it wraps.
                FlexShrink: 1, MinWidth: "0",
            },
        });
    }
    return {
        Type: "Row",
        Style: {
            // The offer is what the band is *given*, and a Row here is a
            // content box — this runtime writes no box-sizing — so an offer
            // has to have the Row's own padding taken off it before it becomes
            // a declared width. That subtraction is band.swift's own line:
            // `inner = offer - chrome`, the proposal a SwiftUI Layout receives
            // after its container's padding has been removed. Without it the
            // two arrangements are handed different outer widths (the one with
            // padding on the Row is 32px wider) and every comparison below is
            // between two bands of different sizes.
            //
            // max-content needs no such adjustment: it is a content-box
            // keyword already, and the padding lands outside it in both
            // arrangements alike.
            //
            // WHICH intrinsic keyword this is was a judgement with nothing
            // behind it until check 9's intrinsic block, which mounts all
            // three candidates and holds each to the band's natural width.
            // They agree here — every child is a box with a declared size, so
            // there is nothing to wrap — and the day one of them stops
            // agreeing is the day the choice starts mattering.
            Width: offer < 0
                ? "max-content"
                : bandPx(offer - a.row.left - a.row.right),
            Padding: bandEdges(a.row),
            Gap: a.gap,
            AlignItems: a.align,
        },
        Children: children,
    };
}

// Every band the check mounts, flattened, so the whole table is one mount and
// one round trip of rects — the same economy the widget grid above is built on.
// Each entry carries the case, the arrangement and the offer it was built for,
// so a failure can name all three.
function bandMounts() {
    const out = [];
    for (const c of BANDS) {
        for (const offer of c.offers) {
            for (const a of [c.now, c.before]) {
                out.push({ c, a, offer, tree: bandTree(a, c, offer) });
            }
        }
    }
    return out;
}

// The band's natural width in one arrangement: everything laid out at its own
// size, with nothing shrunk. Both arrangements hold the same chrome, so this is
// the same number for both — which is what makes "one overflows and the other
// does not" a failure rather than a case.
function bandNatural(a, c) {
    const hasBadge = c.badge.w > 0;
    return a.row.left + a.row.right +
        a.control.left + a.control.right + c.label.w +
        (hasBadge ? a.gap + c.badge.w : 0);
}

// Rects are LayoutUnits — sixty-fourths of a pixel — and two arrangements that
// arrive at the same number by different routes can land on adjacent ones. The
// divergence this check is about is over a point wide, so the tolerance is far
// below anything it could hide.
const BAND_EPSILON = 0.05;
const bandSame = (a, b) => Math.abs(a - b) <= BAND_EPSILON;

// --------------------------------------------------------------------------
// The checks
// --------------------------------------------------------------------------
// The rendered bands
// --------------------------------------------------------------------------
//
// Real components.GroupHeaders, laid out with real glyphs in them, for the two
// band claims that are not arithmetic.
//
// # The tap target (the cross-axis question)
//
// The band's chrome is padding on the growing control rather than on the Row,
// so that a press lands on the whole band rather than on a strip in the middle
// of it. On the plain branch the growing child *is* the control and the claim
// is a main-axis one, which check 9 settles. On the disclosure branch it is
// not: the Row's growing child is a heading wrapper with no chrome at all, and
// the button carrying the insets sits inside it with no weight of its own.
//
//	 Row ─────────────────────────────────────────────────
//	│┌ Box grow:1 ─────────────────────────┐  ┌───┐       │
//	││┌ Row role=button ─────────────────┐ │  │ 3 │ 16px  │
//	│││  16px  ▾ January 2026       8px  │ │  └───┘       │
//	││└──────────────────────────────────┘ │              │
//	│└─────────────────────────────────────┘              │
//	 ─────────────────────────────────────────────────────
//
// Whether the inner box reaches the outer one's edges is a question about the
// *cross* axis of a vertical container, and GrMobFlexSolver — the arithmetic
// ios/verify checks the band with — is a main-axis distributor. It has no
// answer, and internal/bandfixture's arrangementOf renders the plain band
// deliberately to stay out of its way. So the picture in components.bandInsets
// has been assuming it. A browser can be asked, and this asks one.
//
// # The taller child (the question about text)
//
// internal/bandfixture's third case — a badge taller than the control — is the
// one place the two inset arrangements disagree, and the reason it cannot
// happen to a real band is given as two facts: the control's vertical insets
// are larger than the badge's, and both wrap the same caption type. The first
// is arithmetic and bandfixture_test.go checks it. The second is a claim about
// glyphs: the label is a *bold* caption and the badge's text is a plain one at
// the same tier, and "no shorter" is a text measurement no Go test can take.
//
// It is not a formality. The insets differ by 4 points, so a bold caption
// shorter than a plain one by more than that would make the badge the tallest
// child of a real band — and every SameHeight case in the fixture would then be
// describing a layout the framework does not build.
//
// # Why the whole table is one mount
//
// The same economy the widget grid and the band table are built on: nine trees
// in one Column, one round trip of rects and one screenshot.
//
// The screenshot is newer than the rest of this and it changes a constraint the
// comment here used to record. While nothing read a pixel there was no fold to
// stay above — a rect is reported for a node the screenshot would never have
// covered — and the paint check ends that: a band pushed past the bottom of the
// viewport has a rect and no pixels, so the grid is now subject to the same
// bound WIDGETS_PER_ROW keeps the swatches inside, and says so by name when it
// stops fitting.
const BAND_RENDER_GRID = {
    Type: "Column",
    Style: {
        Padding: { Top: 0, Right: 0, Bottom: 0, Left: 0 }, Gap: 0,
        // Each page declares its own width; flex-start keeps it rather than
        // stretching every band to the widest one in the column.
        AlignItems: "flex-start",
    },
    Children: BAND_RENDERS.map((b) => JSON.parse(b.tree)),
};

// gen.go's paths are relative to a mount whose root is the case's own page box.
// Nested in the grid, that box is root/i, so every path gains one level.
const bandRenderPath = (i, path) => path ? path.replace(/^root/, `root/${i}`) : null;

// The tolerance check 9 uses, for the same reason: rects are LayoutUnits, and
// two edges that arrive at the same place by different routes can land on
// adjacent ones.
const bandRenderSame = (a, b) => Math.abs(a - b) <= BAND_EPSILON;

// --------------------------------------------------------------------------
// The fixed-size boxes
// --------------------------------------------------------------------------
//
// What a container of a declared size does with a child bigger than it.
//
// # Why this is a question at all
//
// core.Spacer became "a Box with a fixed size" so that all four targets would
// lay a Spacer's children out the same way, and the note that closed that work
// recorded a difference nobody had measured: Compose's Modifier.width/height set
// a child's minimum AND maximum, so a child bigger than the void is squeezed
// where the DOM was believed to let it spill. It is not a Spacer property — it
// is what every fixed-size container on that target does — and nothing anywhere
// had asked whether the four targets agree about it for ANY fixed-size box.
//
// This is the DOM half of that question, and the answer it gives is not the one
// the note assumed. A browser does not simply let the child spill:
//
//	the container's MAIN axis    the child is a flex item, its flex-shrink
//	                             defaults to 1, and an empty box's automatic
//	                             minimum is 0 — so it is SQUEEZED to the
//	                             container's extent
//	the container's CROSS axis   a declared size beats align-items: stretch, and
//	                             nothing shrinks across the line — so it SPILLS
//
// Which axis is which follows from the container: a core.Box stacks vertically,
// so it squeezes height and spills width, and a core.Row does exactly the
// opposite. Both are mounted, because "the main axis is the one that shrinks" is
// the claim, and a check with one container cannot tell it from "height is the
// one that shrinks".
//
// So the four targets do not agree, and they do not disagree the way the note
// said either: the DOM and Compose agree on the main axis and differ on the
// cross one. See docs/platforms/native.md for the whole census, and
// mobile/verify's TestTheNativesFixedSizeArmsAreTheOnesTheCensusDescribes for
// the two native call sites this rests on.
const VOID_W = 120, VOID_H = 40;
const OVERSIZE_W = 200, OVERSIZE_H = 80;

// One fixed-size container with one child too big for it, in both directions.
//
// The child declares a size on both axes and exceeds the container on both, so
// a single mount shows the squeeze and the spill at once and neither answer has
// to be inferred from the other's absence.
const fixedSizeCase = (type, axis) => ({
    what: `a fixed-size core.${type}`,
    // The container's main axis: what a child is laid out along, and therefore
    // the axis a flex item's shrink factor applies to.
    axis,
    tree: {
        Type: type,
        Style: {
            Width: `${VOID_W}px`, Height: `${VOID_H}px`,
            // No padding and no gap: every number below is the container's own
            // extent against the child's, and chrome would make each comparison
            // a subtraction the reader has to do.
            Padding: { Top: 0, Right: 0, Bottom: 0, Left: 0 }, Gap: 0,
        },
        Children: [{
            Type: "Box",
            Style: { Width: `${OVERSIZE_W}px`, Height: `${OVERSIZE_H}px` },
        }],
    },
});

// And the same container with the child pinned, which is the whole subject of
// core.ShrinkNone.
//
// The main-axis squeeze above is a flex item shrinking, so "do not shrink" is
// exactly the declaration that should stop it — and until core.ShrinkNone there
// was no way to write one: core.FlexShrink(0) stored a zero that Style.Merge,
// htmlout and this runtime each read as "nothing was set". A break-test that
// mutated a fixture's factor from 1 to 0 moved no pixel on any target, which is
// how a declaration nobody can write announces itself.
//
// So this is the case that could not exist. The container is the same, the child
// is the same, and the only difference is the factor — which makes the pair a
// statement about the declaration rather than about the layout.
const pinnedCase = (type, axis) => {
    const c = fixedSizeCase(type, axis);
    c.what = `a fixed-size core.${type} with the child pinned`;
    c.pinned = true;
    c.tree = JSON.parse(JSON.stringify(c.tree));
    c.tree.Children[0].Style.FlexShrink = SHRINK_NONE;
    return c;
};

const FIXED_SIZE_CASES = [
    fixedSizeCase("Box", "vertical"),
    fixedSizeCase("Row", "horizontal"),
    pinnedCase("Box", "vertical"),
    pinnedCase("Row", "horizontal"),
];

// --------------------------------------------------------------------------

async function main() {
    const chromePath = findChrome();
    actOn(startupVerdict({
        transcriptPath: TRANSCRIPT,
        transcriptExists: TRANSCRIPT_EXISTS,
        widgets: WIDGETS.length,
        bands: BANDS.length,
        bandRenders: BAND_RENDERS.length,
        hasWebSocket: typeof WebSocket === "function",
        chromePath,
    }));

    // The server. It answers exactly two paths, because a directory server
    // would be a second thing to get right and this needs no more than the
    // runtime file and the page that loads it.
    const runtimeSource = readFileSync(RUNTIME, "utf8");
    const server = http.createServer((req, res) => {
        if (req.url === "/grmob-runtime.js") {
            res.writeHead(200, { "content-type": "text/javascript" });
            res.end(runtimeSource);
            return;
        }
        res.writeHead(200, { "content-type": "text/html" });
        res.end(PAGE);
    });
    await new Promise((r) => server.listen(0, "127.0.0.1", r));
    const origin = `http://127.0.0.1:${server.address().port}`;

    const profile = mkdtempSync(join(tmpdir(), "grmob-chrome-"));
    const chrome = spawn(chromePath, [
        "--headless=new",
        "--remote-debugging-port=0",
        `--user-data-dir=${profile}`,
        "--no-first-run",
        "--no-default-browser-check",
        "--disable-extensions",
        "--disable-gpu",
        // The scroll check reads window.scrollY one frame after the key. A
        // smooth scroll is an animation, so the number it would read is
        // whatever fraction of the distance had been covered — which makes
        // "did not scroll" and "has not finished scrolling yet" the same
        // measurement.
        "--disable-smooth-scrolling",
        // A known window size: the scroll check needs a viewport shorter than
        // the document, and a headless default that changed would change what
        // the check means.
        "--window-size=800,600",
        // The palette check reads hexes back out of a screenshot, so the
        // browser must not colour-manage them into something else, and a
        // scrollbar must not shift the swatches under the coordinates the
        // page reported for them.
        "--force-color-profile=srgb",
        "--hide-scrollbars",
        "about:blank",
    ], { stdio: ["ignore", "ignore", "ignore"] });

    let session;
    const problems = [];
    try {
        const port = await devtoolsPort(profile);
        const targets = await getJSON(`http://127.0.0.1:${port}/json/list`);
        const page = targets.find((t) => t.type === "page");
        if (!page) throw new Error("Chrome opened no page target");

        session = await connect(page.webSocketDebuggerUrl);
        await session.send("Page.enable");
        await session.send("Runtime.enable");
        const loaded = session.once("Page.loadEventFired");
        await session.send("Page.navigate", { url: origin });
        await loaded;

        const evaluate = async (expression) => {
            const r = await session.send("Runtime.evaluate", {
                expression, returnByValue: true, awaitPromise: true,
            });
            if (r.exceptionDetails) {
                throw new Error(`page threw: ${r.exceptionDetails.text} ` +
                    `${r.exceptionDetails.exception?.description || ""}`);
            }
            return r.result.value;
        };

        // Mount a tree and wait a frame, which is when the runtime's deferred
        // focus/roving work has run (it defers by one rAF because the initial
        // tree is built detached).
        const mount = (tree) => evaluate(
            `new Promise((done) => {
                document.getElementById("app").innerHTML = "";
                GrMob.mount(${JSON.stringify(JSON.stringify(tree))});
                requestAnimationFrame(() => requestAnimationFrame(() => done(true)));
            })`);

        const key = async (name, code) => {
            const common = { key: name, code: name, windowsVirtualKeyCode: code,
                nativeVirtualKeyCode: code };
            await session.send("Input.dispatchKeyEvent", { type: "rawKeyDown", ...common });
            await session.send("Input.dispatchKeyEvent", { type: "keyUp", ...common });
            // Two frames and a beat. The default action of a key is the
            // browser's own work, on its own schedule — a scroll in
            // particular lands after the event has been delivered — so a
            // single rAF reads the state before the thing being measured has
            // happened.
            //
            // The wall-clock fallback beside it is not belt and braces. A key
            // that changes NOTHING — no focus move, no scroll, no DOM edit —
            // gives the browser no reason to paint, so requestAnimationFrame is
            // never called back and a wait on it alone never returns. Which is
            // precisely the case every check here is trying to detect: this
            // harness asks "did the key do something", and waiting for a frame
            // means waiting for the answer to be yes.
            //
            // That was not hypothetical. The toolbar check below hung for its
            // whole timeout the first time its subject was broken, having
            // passed every time its subject worked — a check that can only
            // report success, which is the one result a check must not be able
            // to guarantee itself. Resolving twice is harmless; the first
            // settle wins.
            await evaluate(`new Promise((d) => {
                const done = () => d(true);
                setTimeout(done, 400);
                setTimeout(() => requestAnimationFrame(
                    () => requestAnimationFrame(done)), 50);
            })`);
        };

        // The element focus is on, described well enough to name in a failure.
        const focused = () => evaluate(`(() => {
            const a = document.activeElement;
            if (!a || a === document.body) return "body";
            if (a.id) return "#" + a.id;
            return (a.getAttribute("data-node-path") || a.tagName.toLowerCase())
                + "[" + (a.getAttribute("role") || "") + "]"
                + "(" + (a.getAttribute("tabindex") ?? "none") + ")";
        })()`);

        // ------------------------------------------------------------------
        // 1. tabindex="-1" takes a <button> out of the tab order
        // ------------------------------------------------------------------
        await mount(TABLIST);

        const stamped = await evaluate(
            `[...document.getElementById("app").firstChild.children]
                .map((c) => c.getAttribute("tabindex"))`);
        if (String(stamped) !== "-1,0,-1") {
            problems.push(`the roving tabindex is ${JSON.stringify(stamped)}, want ["-1","0","-1"] ` +
                `— the rest of this check measures something else if this is wrong`);
        }

        await evaluate(`document.getElementById("before").focus()`);
        await key("Tab", 9);
        const entered = await focused();
        if (!entered.includes("[tab]") || !entered.endsWith("(0)")) {
            problems.push(`Tab entered the strip at ${entered}, want the member holding ` +
                `tabindex="0" — a browser that ignored the attribute would land on the first tab`);
        }

        await key("Tab", 9);
        const left = await focused();
        if (left !== "#after") {
            problems.push(`a second Tab landed on ${left}, want #after — the two members ` +
                `carrying tabindex="-1" are still in the tab order, so the strip is three ` +
                `stops rather than one`);
        }

        // ------------------------------------------------------------------
        // 2. a disabled control refuses focus
        // ------------------------------------------------------------------
        await mount(BUTTONS);

        const refused = await evaluate(`(() => {
            const dead = document.querySelector('[data-node-path="root/1"]');
            dead.focus();
            return document.activeElement !== dead;
        })()`);
        if (!refused) {
            problems.push(`a disabled <button> took focus from an explicit focus() call`);
        }

        await evaluate(`document.getElementById("before").focus()`);
        await key("Tab", 9);
        const firstStop = await focused();
        await key("Tab", 9);
        const secondStop = await focused();
        if (secondStop !== "#after") {
            problems.push(`Tab reached ${firstStop} then ${secondStop}; want the live button ` +
                `then #after — the disabled one is still a tab stop`);
        }

        // ------------------------------------------------------------------
        // 3. preventDefault on ArrowDown stops the scroll
        // ------------------------------------------------------------------
        // The control first: the same key, on a page with no composite
        // focused, must scroll. Without this half a browser that never
        // scrolled on ArrowDown would make the real check vacuous.
        await mount(LISTBOX);
        // #after, not the body: a scroll needs a focused element for the key
        // to be delivered to, and the body is not focusable.
        await evaluate(`window.scrollTo(0, 0); document.getElementById("after").focus();`);
        await key("ArrowDown", 40);
        const controlScroll = await evaluate(`window.scrollY`);
        if (!(controlScroll > 0)) {
            problems.push(`ArrowDown with nothing focused did not scroll the page ` +
                `(scrollY ${controlScroll}), so the check below proves nothing`);
        }

        await evaluate(`
            window.scrollTo(0, 0);
            document.querySelector('[data-node-path="root/0"]').focus();`);
        await key("ArrowDown", 40);
        const state = await evaluate(`({
            scrollY: window.scrollY,
            focused: document.activeElement.getAttribute("data-node-path"),
        })`);
        if (state.scrollY !== 0) {
            problems.push(`ArrowDown inside a listbox scrolled the page to ${state.scrollY} ` +
                `— the handler moved the active option and the page moved under it`);
        }
        if (state.focused !== "root/1") {
            problems.push(`ArrowDown moved focus to ${state.focused}, want root/1 — the ` +
                `listbox did not take the key at all, so nothing was prevented`);
        }
        // ------------------------------------------------------------------
        // 4. a toolbar of plain <button>s is one tab stop, and the arrows
        //    reach the rest
        // ------------------------------------------------------------------
        // The claim the second member rule exists for, and the one the shim
        // cannot make: there `tabindex` is a string nobody reads, so "the strip
        // is one stop" restates the attribute the runtime just wrote. Here the
        // tab order is walked by the browser's own focus algorithm, and a chip
        // that was still in it lands between the strip and #after.
        await mount(TOOLBAR);

        const chipStops = await evaluate(
            `[...document.getElementById("app").firstChild.children]
                .map((c) => c.getAttribute("tabindex"))`);
        if (String(chipStops) !== "0,-1,-1") {
            problems.push(`the toolbar's roving tabindex is ${JSON.stringify(chipStops)}, ` +
                `want ["0","-1","-1"] — focusableMembers did not find three plain ` +
                `<button> chips, and the rest of this check measures something else`);
        }

        await evaluate(`document.getElementById("before").focus()`);
        await key("Tab", 9);
        const enteredBar = await focused();
        if (!enteredBar.endsWith("(0)")) {
            problems.push(`Tab entered the toolbar at ${enteredBar}, want the chip ` +
                `holding tabindex="0"`);
        }

        await key("Tab", 9);
        const leftBar = await focused();
        if (leftBar !== "#after") {
            problems.push(`a second Tab landed on ${leftBar}, want #after — a ` +
                `twelve-chip filter bar is still twelve stops in the page's tab order, ` +
                `which is the whole thing the toolbar keyboard was added to fix`);
        }

        // And the two chips Tab now skips are reachable by the arrows, which is
        // the half that makes taking them out of the tab order legitimate.
        await evaluate(`document.getElementById("before").focus()`);
        await key("Tab", 9);
        await key("ArrowRight", 39);
        const arrowed = await focused();
        await key("ArrowRight", 39);
        const arrowedTwice = await focused();
        if (arrowed === enteredBar || arrowedTwice === arrowed) {
            problems.push(`ArrowRight left focus at ${arrowed} then ${arrowedTwice} — ` +
                `the chips Tab no longer reaches are not reachable at all, which is ` +
                `strictly worse than the three tab stops this replaced`);
        }

        // ------------------------------------------------------------------
        // 5. the palette reaches the screen
        // ------------------------------------------------------------------
        // components/variant_test.go crosses every theme's ControlBorder with
        // every fill a control can be drawn on and checks the pair against
        // WCAG 1.4.11's 3:1 floor. All of that is arithmetic over hex strings:
        // it proves #89898E is 3.12:1 on #F2F2F7 and it cannot prove either
        // colour ever reaches a screen. Everything in between — the runtime's
        // style mapping, CSS shorthand parsing, an alpha channel, a colour
        // profile, a hairline antialiased into a tint — is invisible to Go,
        // and every one of them leaves the number true and the control
        // unreadable.
        //
        // So the pairs are painted and the pixels are read back. What is
        // asserted is equality with the hex, not a ratio: the ratio came from
        // Go with the table (see palette.mjs) and recomputing it here would be
        // a second WCAG implementation, which is the one thing a contrast
        // floor cannot survive. A pixel that is the stated colour makes the
        // census's number true of something; a pixel that is not makes it a
        // fact about nothing, and the failure says which.
        await mount(SWATCHES);

        const dpr = await evaluate(`window.devicePixelRatio`);
        // The coordinates come from the browser's own layout rather than
        // from arithmetic here: a swatch is 120x52 by declaration, and where
        // it lands depends on the body margin, the sentinel button above it
        // and whatever the runtime's own flex defaults do.
        const swatchPaths = PALETTES.map((_, i) => swatchPath(i));
        const boxes = await evaluate(`${JSON.stringify(swatchPaths)}.map((p) => {
            const at = (path) => {
                const el = document.querySelector('[data-node-path="' + path + '"]');
                if (!el) return null;
                const r = el.getBoundingClientRect();
                return { x: r.left, y: r.top, w: r.width, h: r.height };
            };
            return { outer: at(p), inner: at(p + "/0") };
        })`);

        const shot = await session.send("Page.captureScreenshot",
            { format: "png", captureBeyondViewport: false });
        const img = decodePNG(Buffer.from(shot.data, "base64"));

        for (let i = 0; i < PALETTES.length; i++) {
            const p = PALETTES[i];
            const box = boxes[i];
            const where = `${p.theme}/${p.what}`;
            if (!box || !box.outer || box.outer.w === 0) {
                problems.push(`${where}: the swatch was not laid out — the palette ` +
                    `check measured nothing for this pair`);
                continue;
            }

            // The fill, sampled inside the outer box and clear of the inner
            // one, which starts 20px in.
            const o = box.outer;
            const fill = pixelAt(img, (o.x + 6) * dpr, (o.y + 6) * dpr);
            if (fill !== p.hex) {
                problems.push(`${where}: the fill painted as ${fill}, not ${p.hex} — ` +
                    (p.kind === "role"
                        ? `every ratio the census records for ${p.theme} is about a ` +
                          `tone the browser does not produce`
                        : `the ${p.ratio.toFixed(2)}:1 the census records against this ` +
                          `backdrop is a fact about a colour nothing painted`));
                continue;
            }
            if (p.kind === "role") continue;

            // The hairline, scanned across the inner box's left edge at its
            // vertical middle. A window of three pixels rather than one
            // because the rect's left edge is a float and a border straddles
            // the rounding; what is being asserted is that a *fully
            // saturated* border pixel exists at all, which is exactly what a
            // sub-pixel or blended edge would not have.
            const b = box.inner;
            const y = (b.y + b.h / 2) * dpr;
            const tone = PALETTES.find(
                (q) => q.theme === p.theme && q.kind === "role").hex;
            let found = null;
            const seen = [];
            for (let dx = -1; dx <= 1; dx++) {
                const got = pixelAt(img, b.x * dpr + dx, y);
                seen.push(got);
                if (got === tone) found = got;
            }
            if (!found) {
                problems.push(`${where}: the 1px frame in ${tone} painted as ` +
                    `${seen.join("/")} — a boundary blended into its backdrop is not ` +
                    `the ${p.ratio.toFixed(2)}:1 the census measured, and no Go test ` +
                    `can see the difference`);
            }
        }
        // ------------------------------------------------------------------
        // 6. a sticky band stays put while the rows scroll under it
        // ------------------------------------------------------------------
        // The first question here that is about layout. Checks 1-4 are about
        // the keyboard and check 5 is about a colour; all five could be asked
        // of a page with no geometry at all. This one cannot be asked anywhere
        // else in wasm/verify: dom.mjs has no layout, so its suites can assert
        // that position:sticky was written and stop there — and "the property
        // is on the element" is exactly what stays true when the box around it
        // defeats the pin.
        await mount(STICKY);

        // The control. A scroller with nothing to scroll would pass every
        // assertion below by never moving anything, which is the same trap
        // check 3 opens with.
        const scroller = `document.querySelector('[data-node-path="root"]')`;
        const scrollable = await evaluate(`(() => {
            const el = ${scroller};
            return { over: el.scrollHeight - el.clientHeight, client: el.clientHeight };
        })()`);
        if (!(scrollable.over > 80)) {
            problems.push(`the sticky fixture has ${scrollable.over}px of overflow in a ` +
                `${scrollable.client}px port — there is nothing for the band to stay put ` +
                `against, so the rest of this check proves nothing`);
        }

        // And the overflow is the one the fixture says it is.
        //
        // The control above passes on ANY overflow, and this fixture has two
        // arrangements available to it. The recorded one is the List standing at
        // its own content height inside a shorter Scroll. The other is the List
        // compressed to the port with its rows spilling out of *it* — also a
        // scroll, also green, and a different arrangement from the one the
        // sticky band is a child of.
        //
        // # What this found out about the shrink factor beside it
        //
        // The List's `FlexShrink` said 0 for as long as this fixture existed and
        // did nothing at all: a literal zero in a mounted tree is what every
        // renderer reads as "nothing was set" (see core.ShrinkNone). It means
        // what it says now — and the List is 400px tall in a 160px port either
        // way, measured, so it never was the reason and it cannot be. A flex
        // item's automatic minimum size is content-based, and these rows carry
        // text, so the List could not be compressed below them whatever its
        // shrink factor said.
        //
        // The declaration stays because it states the intent and is now true
        // rather than inert — a fixture whose rows lost their text would need it
        // — but the claim it used to carry, that it is what keeps this check
        // from having no subject, was never true. This assertion is: it pins the
        // arrangement itself rather than one of the mechanisms that could
        // produce it.
        const listHeight = await evaluate(
            `document.querySelector('[data-node-path="root/0"]').getBoundingClientRect().height`);
        if (listHeight <= scrollable.client) {
            problems.push(`the sticky fixture's List is ${listHeight}px tall in a ` +
                `${scrollable.client}px port, so it was compressed to fit and whatever ` +
                `overflow the control above found is the rows spilling out of the List ` +
                `rather than the List overflowing the Scroll. The band is pinned inside ` +
                `an arrangement the fixture does not describe — see SHRINK_NONE`);
        }

        const rects = async () => evaluate(`(() => {
            const at = (p) => {
                const el = document.querySelector('[data-node-path="' + p + '"]');
                const r = el.getBoundingClientRect();
                return { x: r.left, y: r.top, w: r.width, h: r.height };
            };
            return { port: at("root"), band: at("root/0/0"), row: at("root/0/1"),
                     scrollTop: ${scroller}.scrollTop };
        })()`);

        const before = await rects();
        await evaluate(`${scroller}.scrollTop = 120;`);
        const after = await rects();

        if (after.scrollTop !== 120) {
            problems.push(`the port did not scroll (scrollTop ${after.scrollTop}) — ` +
                `nothing below measures a pin`);
        }
        // The rows moved by the full scroll distance, which is what says the
        // band's staying put is a pin rather than a page that did not move.
        const rowMoved = before.row.y - after.row.y;
        if (Math.abs(rowMoved - 120) > 1) {
            problems.push(`the first row moved ${rowMoved}px for a 120px scroll — the ` +
                `fixture is not behaving like a scroller`);
        }
        if (Math.abs(after.band.y - after.port.y) > 1) {
            problems.push(`the band sits ${(after.band.y - after.port.y).toFixed(1)}px ` +
                `from the top of the port after a 120px scroll, want 0 — position:sticky ` +
                `is written on the element and the box around it is defeating it, which ` +
                `is what no shimmed DOM can tell you`);
        }

        // And the pixels, at a point the band now covers and would not have if
        // it had scrolled away. A rect is what the browser says it laid out;
        // this is what it painted there. The two differ whenever something is
        // drawn over the band — which is the failure z-index:1 is the third of
        // core.StickyHeader()'s declarations for.
        const stickyShot = await session.send("Page.captureScreenshot",
            { format: "png", captureBeyondViewport: false });
        const stickyImg = decodePNG(Buffer.from(stickyShot.data, "base64"));
        const sample = pixelAt(stickyImg,
            (after.band.x + after.band.w / 2) * dpr,
            (after.band.y + after.band.h / 2) * dpr);
        if (sample !== BAND_FILL) {
            const why = sample === null
                ? "the band's own rect is off the screenshot, so it is not pinned at all"
                : sample === ROW_FILL
                    ? "a row is on screen where the band's rect says the band is"
                    : "something else is drawn over it";
            problems.push(`the middle of the pinned band painted as ${sample}, not ` +
                `${BAND_FILL} — ${why}. A rect is what the browser laid out; this is ` +
                `what it put on the screen there`);
        }

        // ------------------------------------------------------------------
        // 7. a real widget draws the palette
        // ------------------------------------------------------------------
        //
        // The swatch grid above proves Chrome puts the census's hexes on the
        // screen. It cannot prove that anything in the framework asks it to:
        // the boxes are built here, and every widget's route from
        // core.ColorPalette.ControlBorder to a border declaration runs through
        // `components`, which this file cannot call.
        //
        // So gen.go renders one quiet components.Chip and one core.Input per
        // bundled theme, on a page painted in that theme's own Background, and
        // reads the three colours off the rendered nodes. What is mounted below
        // is those trees, unmodified; what is asserted is that each widget's own
        // declarations survive to the pixel.
        //
        // # What the field adds that the chip did not
        //
        // Not "a widget on a tag the user agent draws a border on" — the chip
        // was already that. components.Chip is a tappable control and exports
        // as a <button>, which is the first member borderResetTypes ever had.
        // Both widgets are therefore drawing over a user-agent rule, and if the
        // runtime had ever left one in force the chip would have shown it.
        //
        // What the field adds is two things the chip cannot say. It is a second
        // *tag* with a different user-agent rule (<button> is given `outset`,
        // <input> `inset`, and an <input> also arrives with a fill and padding
        // of its own), and it reads the tone from a second *authority* in Go:
        // components.chipRing takes Colors.ControlBorderColor, the role, while
        // core.Input takes Components.Input.BorderColor, a literal the theme
        // states and core/theme_test.go pins to the role separately. Those two
        // hold the same hex in every bundled theme, which is exactly why each
        // case names its own source (widgetCase.RingFrom) rather than both
        // being compared against whichever is handy.
        //
        // # Why both edges are scanned
        //
        // A frame drawn on some sides and not others. One edge in the declared
        // tone says the tone survived; it says nothing about whether the box was
        // closed, and a border emitted per-side — or a user-agent rule surviving
        // on one side under a partial override — paints exactly that. The
        // two-tone styles (`inset`, `outset`) are caught by either edge on its
        // own; the one-sided case is caught by neither unless both are read.
        //
        // # One page, not six
        //
        // Every swatch mounts together and is measured out of a single
        // screenshot. See WIDGET_GRID: the trees are gen.go's, unmodified, and
        // all this changes is that a theme's page fill is a sibling's fill
        // rather than the document's — which no sample here was ever reading,
        // since each is taken inside a rect the browser reported for the node
        // that declares the colour.
        await mount(WIDGET_GRID);

        const widgetRects = await evaluate(`${JSON.stringify(
            WIDGETS.map((_, i) => widgetPath(i)))}.map((p) => {
            const at = (path) => {
                const el = document.querySelector('[data-node-path="' + path + '"]');
                if (!el) return null;
                const r = el.getBoundingClientRect();
                return { x: r.left, y: r.top, w: r.width, h: r.height };
            };
            return { page: at(p), widget: at(p + "/0") };
        })`);

        const gridShot = await session.send("Page.captureScreenshot",
            { format: "png", captureBeyondViewport: false });
        const wImg = decodePNG(Buffer.from(gridShot.data, "base64"));

        for (let i = 0; i < WIDGETS.length; i++) {
            const w = WIDGETS[i];
            const where = `${w.theme}/${w.what}`;
            const rects = widgetRects[i];
            if (!rects.page || !rects.widget || rects.widget.w === 0) {
                problems.push(`${where}: the widget was not laid out — nothing below ` +
                    `measured anything`);
                continue;
            }
            // A swatch the grid pushed off the bottom of the viewport has a
            // rect and no pixels, and pixelAt would answer null for every
            // sample below — three failures naming colours, none of them
            // saying the swatch was never on screen. Said once, here.
            if (rects.page.y + rects.page.h > wImg.height / dpr + 0.5) {
                problems.push(`${where}: the swatch grid runs past the bottom of the ` +
                    `viewport (this one ends at ${Math.round(rects.page.y + rects.page.h)}px ` +
                    `of a ${Math.round(wImg.height / dpr)}px screenshot), so nothing was ` +
                    `painted where its rect says it is — WIDGETS_PER_ROW or the window ` +
                    `size needs to grow with the census`);
                continue;
            }

            // The page, sampled inside its own 24px padding and so clear of
            // the widget. This is the ring's outer backdrop, and it is the
            // theme's Colors.Background rather than the document's white.
            const pageAt = pixelAt(wImg, (rects.page.x + 8) * dpr, (rects.page.y + 8) * dpr);
            if (pageAt !== w.page) {
                problems.push(`${where}: the page behind the widget painted as ` +
                    `${pageAt}, not ${w.page} — the ${w.ratioOnPage.toFixed(2)}:1 the ` +
                    `census records for this ring is against a fill nothing painted`);
                continue;
            }

            // The widget's own fill, sampled in its leading padding rather
            // than at its centre — the centre is where the label is, and a
            // chip's ink blends with the fill across every antialiased stem.
            // (The first attempt sampled the middle and read #8D8D90 out of an
            // #F2F2F7 chip, which is the label and not the fill.) Vertically
            // centred, so a pill's corner radius is nowhere near it.
            const c = rects.widget;
            const fillAt = pixelAt(wImg, (c.x + 6) * dpr, (c.y + c.h / 2) * dpr);
            if (fillAt !== w.fill) {
                problems.push(`${where}: the widget's fill painted as ${fillAt}, not ` +
                    `${w.fill} — its own Style says one colour and the screen has ` +
                    `another, and the ${w.ratioOnFill.toFixed(2)}:1 inside the ring is ` +
                    `about the declared one`);
                continue;
            }

            // The ring, scanned across the *horizontal* edges at the widget's
            // middle. The swatch grid scans a left edge because its inner box
            // is square; a chip is a pill, and a pill's leftmost point is the
            // apex of a curve, where every pixel is an antialiased blend — the
            // first version of this check read #907267 out of an #8D6E63 ring
            // and was measuring the corner radius. A horizontal edge at
            // mid-width is the one run of either shape that is a straight line
            // at any radius.
            //
            // Both of them, not just the top. A user agent's `inset` border —
            // which is what an <input> is given, and what a renderer that
            // emitted colour and width without style would leave in force —
            // paints the top and left in a darkened tone and the bottom and
            // right in a lightened one. Either single edge is consistent with a
            // correct frame; the pair is not.
            //
            // Three device pixels per edge, for the reason the swatch scan uses
            // three: the rect's edge is a float and a 1px border straddles the
            // rounding at dpr 2. The bottom is scanned *inward* from the last
            // row of the rect for the same reason — c.y + c.h is the first row
            // past the box. What is asserted is that a fully saturated boundary
            // pixel exists on each edge, which is exactly what a border blended
            // into its backdrop, or tinted by the user agent's own style, would
            // not have.
            const x = (c.x + c.w / 2) * dpr;
            for (const edge of [
                { name: "top", rows: [0, 1, 2].map((dy) => c.y * dpr + dy) },
                { name: "bottom", rows: [0, 1, 2].map((dy) => (c.y + c.h) * dpr - 1 - dy) },
            ]) {
                let ring = null;
                const seen = [];
                for (const y of edge.rows) {
                    const got = pixelAt(wImg, x, y);
                    seen.push(got);
                    if (got === w.ring) ring = got;
                }
                if (!ring) {
                    problems.push(`${where}: the widget's ${edge.name} boundary in ` +
                        `${w.ring} painted as ${seen.join("/")}. The census measures ` +
                        `that tone at ${w.ratioOnPage.toFixed(2)}:1 against the page ` +
                        `and ${w.ratioOnFill.toFixed(2)}:1 against the fill, and no Go ` +
                        `test and no painted swatch can tell you the widget stopped ` +
                        `drawing it — or that a user agent is still drawing one of ` +
                        `its own underneath`);
                }
            }
        }

        // ------------------------------------------------------------------
        // 8. a browser applies ARIA's own rules to a value range
        // ------------------------------------------------------------------
        //
        // The two DOM exporters write aria-valuenow, -valuemin and -valuemax
        // verbatim and resolve nothing, on the argument that a browser applies
        // ARIA's rules itself. core.ValueRange.Progress states those rules for
        // the platforms that do not — and until this check the only thing that
        // had ever compared against it was a JVM running Compose's
        // transliteration. So the web half of a rule the whole vocabulary
        // rests on was an assumption.
        await mount(VALUE_BARS);
        await session.send("Accessibility.enable");
        const { nodes: axNodes } = await session.send("Accessibility.getFullAXTree");

        // The verdict is valuerange.mjs's, not this file's. Its two halves
        // point in opposite directions and only one of them can fail on a
        // machine with a shipping browser, so the decision lives beside the
        // table where valuerange_test.mjs can hand it all four answers
        // directly — see the note there. What is left here is the round trip.
        for (const row of VALUE_RANGES) {
            const problem = valueRangeProblem(row, axRange(axNodes, row.name));
            if (problem) problems.push(problem);
        }


        // ------------------------------------------------------------------
        // 9. two arrangements of the same band lay out the same way
        // ------------------------------------------------------------------
        //
        // See bandMounts above for the claim and for the two modelling
        // decisions. What is new here is the target: ios/verify solves this
        // through GrMobFlexSolver and records a divergence under overflow that
        // it attributes to shrinking in proportion to a base that includes the
        // child's own padding. CSS shrinks in proportion to the *inner* flex
        // base size, which excludes it — so the prediction is that a browser
        // agrees at every offer, including the ones where the SwiftUI solver
        // does not, and that the recorded difference is a real cross-target
        // divergence rather than an artefact of either implementation.
        //
        // Asserted in both directions, like the pinned divergence in check 8: a
        // browser that started disagreeing is a failure somebody reads, and so
        // is one that stopped reaching the overflow arm at all.
        const mounts = bandMounts();
        await mount({
            Type: "Column",
            Style: {
                Padding: { Top: 0, Right: 0, Bottom: 0, Left: 0 }, Gap: 0,
                // flex-start so a 120px band beside a 360px one keeps its own
                // declared width rather than being stretched to the column's.
                // A declared Width already wins over stretch; this says so.
                AlignItems: "flex-start",
            },
            Children: mounts.map((m) => m.tree),
        });

        // One round trip for the whole table. The control is the Row's first
        // child and the badge, where there is one, its second — the shape
        // bandTree builds and the shape band.swift solves.
        const bandRects = await evaluate(`${JSON.stringify(
            mounts.map((_, i) => `root/${i}`))}.map((p) => {
            const at = (path) => {
                const el = document.querySelector('[data-node-path="' + path + '"]');
                if (!el) return null;
                const r = el.getBoundingClientRect();
                return { x: r.left, y: r.top, w: r.width, h: r.height };
            };
            return { row: at(p), control: at(p + "/0"), badge: at(p + "/1") };
        })`);

        // The measures band.swift takes, in the Row's own coordinates. A rect
        // is a border box and these boxes carry no border, so a control's width
        // is its content plus its own padding — which is exactly the solver's
        // "used outer size", and why the label's room is that width less the
        // arrangement's insets.
        const bandLayout = (m, r) => ({
            width: r.row.w,
            height: r.row.h,
            controlX: r.control.x - r.row.x,
            controlWidth: r.control.w,
            controlHeight: r.control.h,
            labelX: r.control.x - r.row.x + m.a.control.left,
            labelWidth: r.control.w - m.a.control.left - m.a.control.right,
            badgeX: r.badge ? r.badge.x - r.row.x : null,
            badgeWidth: r.badge ? r.badge.w : null,
        });

        // Paired back up by (case, offer): bandMounts emits `now` then `before`
        // for each, which is the order this walks in twos.
        let sawBandSlack = false, sawBandOverflow = false;
        for (let i = 0; i < mounts.length; i += 2) {
            const nowM = mounts[i], beforeM = mounts[i + 1];
            const c = nowM.c;
            const at = nowM.offer < 0
                ? "an indefinite proposal" : `${nowM.offer}px`;
            const where = `${c.what} at ${at}`;

            const nowR = bandRects[i], beforeR = bandRects[i + 1];
            if (!nowR.row || !beforeR.row || nowR.row.w === 0) {
                problems.push(`${where}: the band was not laid out — nothing below ` +
                    `measured anything`);
                continue;
            }
            if (Boolean(nowR.badge) !== (c.badge.w > 0)) {
                problems.push(`${where}: the band mounted with ` +
                    `${nowR.badge ? "a badge" : "no badge"} and the case says ` +
                    `${c.badge.w > 0 ? "it has one" : "it has none"} — bandTree and the ` +
                    `fixture have drifted apart`);
                continue;
            }

            const now = bandLayout(nowM, nowR);
            const before = bandLayout(beforeM, beforeR);

            // The control assertion first, because everything else is a
            // comparison between two bands and this is what says they are two.
            // An arrangement that had not actually moved its insets would make
            // every measure below a comparison between two copies of one band.
            // And that the Row is centring its children rather than stretching
            // them, which is what the height comparison at the bottom rests on:
            // a Row's height is its tallest child plus its own vertical padding
            // only while a child is free to be shorter than the line. The
            // fixture reads the alignment off the real widget (Arrangement.Align)
            // and band.swift refuses a case that is not centred; here the
            // browser does the laying out, so the same fact is available as a
            // measurement — a stretched control is as tall as the line, and a
            // centred one is as tall as its own content plus its own insets.
            //
            // The oversized-badge case is the one that can tell the difference.
            // In every real band the padded control is already the tallest
            // child, so stretching it changes nothing; the fixture carries a
            // taller badge precisely so that something is taller than it.
            for (const [m, l] of [[nowM, now], [beforeM, before]]) {
                const want = c.label.h + m.a.control.top + m.a.control.bottom;
                if (!bandSame(l.controlHeight, want)) {
                    problems.push(`${where}: with the insets ${m.a.what} the control is ` +
                        `${l.controlHeight.toFixed(2)}px tall and its content plus its ` +
                        `own insets is ${want}px. The band states align-items ` +
                        `${m.a.align}, and a control stretched to the line makes the ` +
                        `height comparison below a claim about a layout the widget does ` +
                        `not build`);
                }
            }

            const grew = now.controlWidth - before.controlWidth;
            const wantGrew = c.now.control.left + c.now.control.right;
            if (!bandSame(grew, wantGrew)) {
                problems.push(`${where}: the control is ${grew.toFixed(2)}px wider with ` +
                    `the insets on it and the insets it carries total ${wantGrew}px. The ` +
                    `target is supposed to have absorbed exactly the chrome that left ` +
                    `the Row, and if it did not, every comparison below is between two ` +
                    `copies of one band`);
            }
            if (!bandSame(now.controlX, c.now.row.left)) {
                problems.push(`${where}: the control starts ${now.controlX.toFixed(2)}px ` +
                    `into the band with the insets on it, and the Row's own leading ` +
                    `inset is ${c.now.row.left}px. The whole point of the move is that a ` +
                    `press lands on the band's leading edge rather than 16px into it`);
            }

            // Overflow is decided once, from the `now` arrangement, which is
            // only legitimate while the two hold the same chrome — the claim
            // internal/bandfixture's Rewind is supposed to conserve. Checked
            // rather than assumed: a Rewind that had stopped conserving it
            // would put one arrangement in the shrink arm and the other in the
            // grow arm, and every comparison below would be between two
            // different questions.
            const natural = bandNatural(nowM.a, c);
            const naturalBefore = bandNatural(beforeM.a, c);
            if (!bandSame(natural, naturalBefore)) {
                problems.push(`${where}: the band's natural width is ${natural}px with ` +
                    `the insets ${c.now.what} and ${naturalBefore}px with them ` +
                    `${c.before.what}. The two are supposed to hold the same chrome, so ` +
                    `internal/bandfixture's Rewind has stopped conserving it and the ` +
                    `two arrangements are no longer the same band`);
                continue;
            }
            const overflowed = nowM.offer >= 0 && nowM.offer < natural - BAND_EPSILON;
            if (overflowed) sawBandOverflow = true; else sawBandSlack = true;

            // The control the overflow arm needs, and it is not a formality.
            // CSS gives a flex item an automatic minimum size, so an item that
            // refused to shrink would leave both arrangements at their natural
            // width, overflowing the container — and they would agree, because
            // neither had done any arithmetic. That is agreement with no
            // subject, and it is exactly what dropping the min-width: 0 in
            // bandTree produces: the fixture's label is a box with a declared
            // width, whose min-content is the whole of it.
            //
            // So under overflow the label must have ended up with *less* room
            // than it asked for, in both arrangements, before their agreement
            // means anything.
            if (overflowed) {
                for (const [what, got] of [
                    [c.now.what, now.labelWidth], [c.before.what, before.labelWidth],
                ]) {
                    if (got >= c.label.w - BAND_EPSILON) {
                        problems.push(`${where}: with the insets ${what} the label still ` +
                            `has ${got.toFixed(2)}px for ${c.label.w}px of content, in a ` +
                            `band offered less than its natural ${natural}px. Nothing ` +
                            `shrank — the flex item's automatic minimum size is still in ` +
                            `force — so the two arrangements agree by never reaching the ` +
                            `arithmetic this check is about`);
                    }
                }
                // And the badge shares the deficit, which is what makes this the
                // same problem the SwiftUI solver is given: there the badge's
                // growth weight is 0 and its shrink share is its base like
                // everything else's. A badge pinned at its own width would move
                // the whole deficit onto the control — the two arrangements
                // would still agree, for the reason they always do, but the
                // numbers this check measures would no longer be the numbers
                // ios/verify's are being compared with.
                for (const [what, got] of [
                    [c.now.what, now.badgeWidth], [c.before.what, before.badgeWidth],
                ]) {
                    if (got !== null && got >= c.badge.w - BAND_EPSILON) {
                        problems.push(`${where}: with the insets ${what} the badge is ` +
                            `still its whole ${c.badge.w}px wide in a band that overflows, ` +
                            `so the control is absorbing the entire deficit. ` +
                            `ios/verify/band.swift divides it between both children, and ` +
                            `the two passes are no longer solving the same problem`);
                    }
                }
            }

            // Every measure, at every offer, in both arrangements. There is no
            // separate overflow arm here and that IS the finding: on this
            // target the deficit is divided the same way in both arrangements,
            // so the identity that holds with slack goes on holding without it.
            const measures = [
                ["the band's width", now.width, before.width],
                ["the label's leading edge", now.labelX, before.labelX],
                ["the label's width", now.labelWidth, before.labelWidth],
            ];
            if (now.badgeX !== null && before.badgeX !== null) {
                measures.push(["the badge's leading edge", now.badgeX, before.badgeX]);
            }
            for (const [what, a, b] of measures) {
                if (bandSame(a, b)) continue;
                problems.push(`${where}: ${what} is ${a.toFixed(2)}px with the insets ` +
                    `${c.now.what} and ${b.toFixed(2)}px with them ${c.before.what}. ` +
                    (overflowed
                        ? `This is the overflow arm, and a browser agreeing here is the ` +
                          `recorded cross-target divergence: CSS distributes shrink over ` +
                          `the inner flex base size, which excludes a child's own ` +
                          `padding, where GrMobFlexSolver's base includes it (see ` +
                          `ios/verify/band.swift, which asserts the disagreement). A ` +
                          `browser that has stopped agreeing means that rule has ` +
                          `changed, and the note in band.swift is now wrong about the web`
                        : `Moving the band's padding onto its control is supposed to be ` +
                          `the same pixels one node in, and in a browser it is not`));
            }

            // The cross axis, which is where the two arrangements genuinely
            // differ and the fixture says which way. With align-items centre a
            // Row's height is its tallest child plus its own vertical padding,
            // so moving that padding onto one child stops it being added to the
            // other — invisible while the padded control is the taller child,
            // which every real band is, and visible in the fixture's one
            // deliberately oversized badge. Asked of a browser for the first
            // time here: the model is CSS's own, and ios/verify was checking a
            // transliteration of it.
            const sameHeight = bandSame(now.height, before.height);
            if (sameHeight !== c.sameHeight) {
                problems.push(`${where}: the band is ${now.height.toFixed(2)}px tall with ` +
                    `the insets ${c.now.what} and ${before.height.toFixed(2)}px with them ` +
                    `${c.before.what}, and internal/bandfixture says the two ` +
                    `${c.sameHeight ? "agree" : "differ"}. ` +
                    (c.sameHeight
                        ? `A band whose padded control is its tallest child keeps its ` +
                          `height across the move, so either the Row has stopped ` +
                          `centring its children or a child is taller than the fixture ` +
                          `thinks`
                        : `A badge taller than the padded control loses the Row's ` +
                          `vertical padding when that padding moves; heights that now ` +
                          `agree mean a real band's agreement holds for a different ` +
                          `reason than the one recorded`));
            }
        }
        // Both arms have to have run, for the reason band.swift gives about its
        // own: a table whose offers all had slack would assert the identity and
        // never reach the arithmetic this check was added for.
        if (!sawBandSlack || !sawBandOverflow) {
            problems.push(`the band offers reached ` +
                (sawBandSlack ? "" : "no case with slack ") +
                (sawBandOverflow ? "" : "no case that overflows ") +
                `— internal/bandfixture is supposed to carry both, and the half of ` +
                `this check that asks whether the insets survive overflow is asserted ` +
                `over nothing`);
        }


        // The intrinsic offer, which is the one modelling decision in bandTree
        // that is a judgement rather than a reading.
        //
        // internal/bandfixture spells "no definite offer" as a negative number —
        // SwiftUI probing for an ideal size — and bandTree spells that as
        // `width: max-content`. Nothing held that. `min-content` is the other
        // candidate and it asks a different question, so the comment saying
        // max-content was the faithful one was a decision with no consequence
        // attached to it.
        //
        // Two things are asked here, and the second is the interesting one:
        //
        //	the band hugs its content   at an indefinite offer each arrangement
        //	                            must lay out at its natural width, which
        //	                            is what "ideal size" means and is the
        //	                            whole of what the comparisons above need
        //	                            from the keyword
        //	the candidates agree        max-content, min-content and fit-content
        //	                            all land on that number for THIS fixture,
        //	                            so the choice between them is not
        //	                            load-bearing
        //
        // The second is why the judgement was safe, and it is a fact about the
        // fixture rather than about CSS: every child here is a box with a
        // declared size, so there is nothing to wrap and the three intrinsic
        // sizings coincide. A fixture that grew a child with real text would
        // separate them — min-content is its longest word — and this fails on
        // the day that happens, which is exactly the day somebody should choose
        // deliberately instead of inheriting a comment.
        const intrinsic = [];
        for (const c of BANDS) {
            for (const offer of c.offers.filter((o) => o < 0)) {
                for (const a of [c.now, c.before]) {
                    for (const keyword of ["max-content", "min-content", "fit-content"]) {
                        const tree = bandTree(a, c, offer);
                        tree.Style.Width = keyword;
                        intrinsic.push({ c, a, keyword, tree });
                    }
                }
            }
        }
        if (intrinsic.length === 0) {
            problems.push(`internal/bandfixture states no indefinite offer, so bandTree's ` +
                `max-content arm is never built and the judgement behind it is unasked`);
        } else {
            await mount({
                Type: "Column",
                Style: {
                    Padding: { Top: 0, Right: 0, Bottom: 0, Left: 0 }, Gap: 0,
                    AlignItems: "flex-start",
                },
                Children: intrinsic.map((m) => m.tree),
            });
            const intrinsicWidths = await evaluate(`${JSON.stringify(
                intrinsic.map((_, i) => `root/${i}`))}.map((p) => {
                const el = document.querySelector('[data-node-path="' + p + '"]');
                return el ? el.getBoundingClientRect().width : null;
            })`);
            intrinsic.forEach((m, i) => {
                const got = intrinsicWidths[i];
                const want = bandNatural(m.a, m.c);
                if (got === null) {
                    problems.push(`${m.c.what} at width:${m.keyword}: the band was not ` +
                        `laid out`);
                    return;
                }
                if (bandSame(got, want)) return;
                problems.push(`${m.c.what} with the insets ${m.a.what} is ` +
                    `${got.toFixed(2)}px wide at width:${m.keyword} and its natural ` +
                    `width is ${want}px. ` +
                    (m.keyword === "max-content"
                        ? `bandTree renders bandfixture's indefinite offer as max-content ` +
                          `on the argument that it is the CSS spelling of "hug your ` +
                          `content", and every comparison at that offer is between two ` +
                          `bands that are supposed to have hugged`
                        : `That keyword is not the one bandTree uses — it is mounted here ` +
                          `as a control. While all three intrinsic sizings land on the ` +
                          `natural width the choice between them costs nothing, which is ` +
                          `why the comment could state it without holding it to anything. ` +
                          `They have now come apart, so the judgement has a consequence ` +
                          `and somebody has to make it on purpose`));
            });
        }

        // ------------------------------------------------------------------
        // 10. a real band's tap target spans it, and its control is its tallest
        //     child
        // ------------------------------------------------------------------
        //
        // See BAND_RENDER_GRID for both claims and for why neither is reachable
        // from the arithmetic fixture check 9 runs on. In short: one is a
        // cross-axis question a main-axis solver cannot answer, and the other is
        // a measurement of text.
        await mount(BAND_RENDER_GRID);

        const renderRects = await evaluate(`${JSON.stringify(
            BAND_RENDERS.map((b, i) => ({
                band: bandRenderPath(i, b.band),
                wrapper: bandRenderPath(i, b.wrapper),
                control: bandRenderPath(i, b.control),
                badge: bandRenderPath(i, b.badge),
                label: bandRenderPath(i, b.label),
                chevron: bandRenderPath(i, b.chevron),
            })))}.map((paths) => {
            const at = (path) => {
                if (!path) return null;
                const el = document.querySelector('[data-node-path="' + path + '"]');
                if (!el) return null;
                const r = el.getBoundingClientRect();
                return { x: r.left, y: r.top, w: r.width, h: r.height };
            };
            const out = {};
            for (const k of Object.keys(paths)) out[k] = at(paths[k]);
            return out;
        })`);

        // And the paint, which this grid did not read at all until now.
        //
        // Every other browser check that mounts something real samples a colour;
        // this one measured rects only, so a band that laid out perfectly and
        // painted nothing would have passed every assertion below it. That is
        // not a hypothetical gap: a band's own fill is the whole reason it may
        // span its container edge to edge instead of being inset like a row, and
        // it is the one declaration in GroupHeader's recipe that geometry cannot
        // see.
        //
        // One screenshot for the whole grid, like the widget grid's, and the
        // same fold rule applies to it now that pixels are being read — a band
        // pushed past the bottom of the viewport has a rect and no pixels.
        const bandShot = await session.send("Page.captureScreenshot",
            { format: "png", captureBeyondViewport: false });
        const bandImg = decodePNG(Buffer.from(bandShot.data, "base64"));
        const bandDpr = await evaluate(`window.devicePixelRatio`);

        // Keyed by theme so the two branches can be held against each other
        // below: adding a handler to a band is supposed to hand the caller a
        // control and not a relayout.
        const byTheme = new Map();
        for (let i = 0; i < BAND_RENDERS.length; i++) {
            const b = BAND_RENDERS[i], r = renderRects[i];
            const where = `${b.theme}/${b.what}`;

            if (!r.band || !r.control || r.band.w === 0) {
                problems.push(`${where}: the band was not laid out — gen.go's paths ` +
                    `name nodes this mount does not have, so nothing below measured ` +
                    `anything`);
                continue;
            }
            if (Boolean(r.wrapper) !== Boolean(b.wrapper)) {
                problems.push(`${where}: gen.go ${b.wrapper ? "found" : "found no"} ` +
                    `heading wrapper and the mount ${r.wrapper ? "has" : "has no"} node ` +
                    `at that path — the two have drifted apart`);
                continue;
            }
            if (Boolean(r.badge) !== Boolean(b.badge)) {
                problems.push(`${where}: gen.go ${b.badge ? "found" : "found no"} badge ` +
                    `and the mount ${r.badge ? "has" : "has no"} node at that path`);
                continue;
            }

            // The band painted its own fill.
            //
            // Sampled six device-independent pixels into the band and vertically
            // centred. The band Row's own leading inset is 0 — that is the move
            // this whole check is about — so that point is inside the control,
            // and inside the control's own leading padding, which is a run of
            // the band's fill with no ink in it. The control declares no
            // background of its own, so what is there is the Row's, showing
            // through: which is the arrangement being asserted as much as the
            // colour is.
            //
            // Vertically centred for the widget grid's reason: a horizontal edge
            // at mid-height is clear of any glyph and of any corner the band
            // might grow.
            if (r.band.y + r.band.h > bandImg.height / bandDpr + 0.5) {
                problems.push(`${where}: the band grid runs past the bottom of the ` +
                    `viewport (this one ends at ${Math.round(r.band.y + r.band.h)}px of a ` +
                    `${Math.round(bandImg.height / bandDpr)}px screenshot), so nothing was ` +
                    `painted where its rect says it is — the grid or the window size ` +
                    `needs to grow with the number of themes`);
                continue;
            }
            const bandFill = pixelAt(bandImg,
                (r.band.x + 6) * bandDpr, (r.band.y + r.band.h / 2) * bandDpr);
            if (bandFill !== b.fill) {
                problems.push(`${where}: the band painted as ${bandFill}, and its own ` +
                    `Style declares ${b.fill} (the page behind it is ${b.page}). A band's ` +
                    `fill is why it may span its container edge to edge rather than being ` +
                    `inset like a row, and it is the one thing in the recipe that every ` +
                    `rect below is blind to — this grid measured nine real bands and read ` +
                    `no pixels at all until this line`);
                continue;
            }

            // The control assertion, and it is the one that makes the rest mean
            // anything. Both branches reach the band's trailing edge below, and
            // a control with a grow weight of its own would reach it for a
            // reason that has nothing to do with the question — on the
            // disclosure branch it would be filling the wrapper by growing into
            // it rather than by being stretched to it.
            if (b.controlGrows) {
                problems.push(`${where}: the control declares a grow weight of its own. ` +
                    `The plain branch's control grows because it IS the band Row's ` +
                    `growing child; the disclosure's button is supposed to have none, ` +
                    `and everything measured here would otherwise be true for the wrong ` +
                    `reason`);
            }

            // The band's leading edge is the control's, which is the whole move:
            // the insets came off the Row so that a press lands on the leading
            // edge rather than 16px into it.
            const lead = r.control.x - r.band.x;
            if (!bandRenderSame(lead, b.rowLeft)) {
                problems.push(`${where}: the control starts ${lead.toFixed(2)}px into ` +
                    `the band and the Row's own leading inset is ${b.rowLeft}px. A press ` +
                    `on the first ${lead.toFixed(2)}px of this band lands on nothing`);
            }

            // And its trailing edge stops where the band's content does: at the
            // badge when there is one, at the Row's own trailing inset when
            // there is not. This is the half check 9 cannot see on the
            // disclosure branch, because it is the wrapper that is distributed
            // to and the button that has to fill it.
            const wantRight = r.band.x + r.band.w - b.rowRight -
                (r.badge ? r.badge.w + b.gap : 0);
            const gotRight = r.control.x + r.control.w;
            if (!bandRenderSame(gotRight, wantRight)) {
                problems.push(`${where}: the control's trailing edge is at ` +
                    `${gotRight.toFixed(2)}px and the band's content ends at ` +
                    `${wantRight.toFixed(2)}px — ${(wantRight - gotRight).toFixed(2)}px ` +
                    `of the band is not part of the tap target. ` +
                    (b.wrapper
                        ? `The button carries no weight of its own, so it reaches the ` +
                          `growing wrapper's edge only by being stretched to it; a ` +
                          `wrapper that stopped stretching its child makes the picture ` +
                          `in components.bandInsets wrong about this branch`
                        : `The control is the band Row's growing child, so this is the ` +
                          `distribution check 9 measures, in a band with real text`));
            }

            // The mechanism, on the branch that has one. Stated separately from
            // the edge above because the two fail together and mean different
            // things: an edge that stops short with a wrapper that the button
            // fills is a band whose wrapper stopped growing, and one the button
            // does not fill is the cross-axis stretch going away.
            if (r.wrapper) {
                if (!bandRenderSame(r.control.w, r.wrapper.w) ||
                    !bandRenderSame(r.control.x, r.wrapper.x)) {
                    problems.push(`${where}: the button is ${r.control.w.toFixed(2)}px ` +
                        `wide at x=${r.control.x.toFixed(2)} inside a heading wrapper ` +
                        `${r.wrapper.w.toFixed(2)}px wide at x=${r.wrapper.x.toFixed(2)}. ` +
                        `A non-growing child of a vertical container is supposed to be ` +
                        `stretched to it — that is the cross-axis fallback core.Box and ` +
                        `core.SafeArea both document — and it is the only thing making ` +
                        `the disclosure band's tap target span the band`);
                }
            }

            // The taller child, with glyphs in it. Both halves of the reason are
            // named, because the failure is a fact about a font and the reader
            // has to know which of the two claims it broke.
            if (r.badge && !(r.control.h > r.badge.h)) {
                problems.push(`${where}: the control is ${r.control.h.toFixed(2)}px tall ` +
                    `and the badge is ${r.badge.h.toFixed(2)}px. ` +
                    `internal/bandfixture's SameHeight cases rest on the padded control ` +
                    `being the band's tallest child, for two reasons: its vertical ` +
                    `insets are larger than the badge's (checked in Go) and a bold ` +
                    `caption is no shorter than a plain one at the same tier (checked ` +
                    `here, and nowhere else). A badge that is the taller child makes ` +
                    `every SameHeight case a claim about a layout this framework does ` +
                    `not build`);
            }

            // Keyed by branch AND by whether the count is showing, because the
            // parity comparison below is between two bands of the same shape: a
            // count-hidden band's trailing inset is its control's rather than
            // its badge's, so pairing one with a badged band would compare two
            // trailing edges that are supposed to sit in different places.
            const seen = byTheme.get(b.theme) || {};
            seen[(b.collapsible ? "disclosure" : "plain") + (b.badge ? "" : "NoCount")] =
                { b, r };
            byTheme.set(b.theme, seen);
        }

        // The two branches are the same band — and the measurement is what says
        // in what sense.
        //
        // components.GroupHeader spells its insets once for both branches, on
        // the argument that "a caller who adds OnToggle to a GroupHeader gets a
        // control and not a relayout". Held to pixels, that is true of the
        // chrome and NOT true of the height: the disclosure band is a point
        // taller in every bundled theme, because its button holds a chevron the
        // plain band does not and a control's height is its tallest child plus
        // its own insets. The glyph's line box exceeds the caption's, which is a
        // fact about a font stack rather than about this framework.
        //
        // So the height is checked as an *equation* rather than as an equality,
        // and it is the equation that says the difference is content:
        //
        //	disclosure band - plain band  =  max(0, chevron line box - label's)
        //
        // Both directions fail. A difference larger than that is chrome that has
        // drifted between the branches, which is the thing the shared bandInsets
        // exists to prevent; a difference smaller is a chevron whose line box
        // stopped exceeding the words', which would make this comment wrong
        // about why the two differ.
        //
        // Only the two badged shapes are paired. The count-hidden band is a
        // different band — its trailing inset is the control's rather than the
        // badge's — and comparing it with either would be comparing two widths
        // that are supposed to differ.
        let sawBranchPair = false;
        for (const [theme, seen] of byTheme) {
            if (!seen.plain || !seen.disclosure) continue;
            sawBranchPair = true;
            const p = seen.plain.r, d = seen.disclosure.r;

            if (!d.chevron || !d.label || !p.label) {
                problems.push(`${theme}: the branch pair is missing a text rect ` +
                    `(chevron ${Boolean(d.chevron)}, disclosure label ${Boolean(d.label)}, ` +
                    `plain label ${Boolean(p.label)}) — gen.go locates all three by the ` +
                    `words they carry, so the band's content has changed shape`);
                continue;
            }
            const overhang = Math.max(0, d.chevron.h - d.label.h);
            const taller = d.band.h - p.band.h;
            if (!bandRenderSame(taller, overhang)) {
                problems.push(`${theme}: the disclosure band is ` +
                    `${taller.toFixed(2)}px taller than the plain one and its chevron's ` +
                    `line box exceeds the label's by ${overhang.toFixed(2)}px ` +
                    `(chevron ${d.chevron.h.toFixed(2)}px, words ${d.label.h.toFixed(2)}px). ` +
                    `The two branches carry the same insets around the same caption tier, ` +
                    `so the whole of the difference is supposed to be the one child the ` +
                    `plain band does not have. ` +
                    (taller > overhang
                        ? `More than that is chrome that has drifted between the two ` +
                          `branches, which is exactly what spelling bandInsets once for ` +
                          `both was meant to prevent`
                        : `Less than that means a control is no longer as tall as its ` +
                          `tallest child plus its own insets, so the height half of ` +
                          `internal/bandfixture's SameHeight cases is modelling ` +
                          `something else`));
            }

            // The chrome, which is the half the promise is actually about and
            // the half that holds exactly.
            if (!bandRenderSame(p.control.x - p.band.x, d.control.x - d.band.x)) {
                problems.push(`${theme}: the plain band's control starts ` +
                    `${(p.control.x - p.band.x).toFixed(2)}px into it and the ` +
                    `disclosure's starts ${(d.control.x - d.band.x).toFixed(2)}px in. ` +
                    `Both branches take the same bandInsets, so the label moves on the ` +
                    `day a caller adds a handler`);
            }
            const pRight = p.band.x + p.band.w - (p.control.x + p.control.w);
            const dRight = d.band.x + d.band.w - (d.control.x + d.control.w);
            if (!bandRenderSame(pRight, dRight)) {
                problems.push(`${theme}: the plain band's control stops ` +
                    `${pRight.toFixed(2)}px short of the trailing edge and the ` +
                    `disclosure's stops ${dRight.toFixed(2)}px short. The two branches ` +
                    `are the same band, so the tap target's trailing edge is not ` +
                    `supposed to move when a handler arrives`);
            }
        }
        if (BAND_RENDERS.length && !sawBranchPair) {
            problems.push(`no theme rendered both a plain band and a disclosure band, so ` +
                `the branch-parity comparison above ran over nothing — ` +
                `bandRenderBuilders in gen.go is supposed to carry both`);
        }




        // ------------------------------------------------------------------
        // 11. a fixed-size container squeezes its child along its main axis
        //     and lets it spill across
        // ------------------------------------------------------------------
        //
        // See FIXED_SIZE_CASES for the question and for why it has two
        // containers rather than one.
        await mount({
            Type: "Column",
            Style: {
                Padding: { Top: 0, Right: 0, Bottom: 0, Left: 0 }, Gap: 0,
                AlignItems: "flex-start",
            },
            Children: FIXED_SIZE_CASES.map((c) => c.tree),
        });
        const voidRects = await evaluate(`${JSON.stringify(
            FIXED_SIZE_CASES.map((_, i) => i))}.map((i) => {
            const at = (path) => {
                const el = document.querySelector('[data-node-path="' + path + '"]');
                if (!el) return null;
                const r = el.getBoundingClientRect();
                return { x: r.left, y: r.top, w: r.width, h: r.height };
            };
            return { box: at("root/" + i), child: at("root/" + i + "/0") };
        })`);

        for (let i = 0; i < FIXED_SIZE_CASES.length; i++) {
            const c = FIXED_SIZE_CASES[i], r = voidRects[i];
            if (!r.box || !r.child) {
                problems.push(`${c.what}: the container or its child was not laid out`);
                continue;
            }

            // The container keeps the size it declared, whatever its child
            // does. This is the half that is the same on all four targets and
            // it is asserted first, because every claim below is stated
            // relative to it — a container that had itself grown to fit its
            // child would make "the child spills" and "the child fits"
            // indistinguishable.
            if (!bandRenderSame(r.box.w, VOID_W) || !bandRenderSame(r.box.h, VOID_H)) {
                problems.push(`${c.what}: the container is ` +
                    `${r.box.w.toFixed(2)}x${r.box.h.toFixed(2)} and it declared ` +
                    `${VOID_W}x${VOID_H}. A fixed size that a child can enlarge is not ` +
                    `a fixed size, and nothing below is measuring what it says it is`);
                continue;
            }

            const main = c.axis === "vertical"
                ? { name: "height", box: r.box.h, child: r.child.h, want: VOID_H,
                    declared: OVERSIZE_H }
                : { name: "width", box: r.box.w, child: r.child.w, want: VOID_W,
                    declared: OVERSIZE_W };
            const cross = c.axis === "vertical"
                ? { name: "width", child: r.child.w, declared: OVERSIZE_W, box: r.box.w }
                : { name: "height", child: r.child.h, declared: OVERSIZE_H, box: r.box.h };

            // The control: the child has to be asking for more than it can
            // have on both axes, or one of the two answers below is being
            // read off a child that fit.
            if (main.declared <= main.want || cross.declared <= cross.box) {
                problems.push(`${c.what}: the child asks for ` +
                    `${OVERSIZE_W}x${OVERSIZE_H} in a ${VOID_W}x${VOID_H} container, ` +
                    `which does not exceed it on both axes — one of the two answers ` +
                    `below is about a child that fits`);
                continue;
            }

            // The pinned cases are the same measurement with the opposite
            // expected answer on the main axis: a factor of 0 is an instruction
            // not to shrink, so the child keeps its declared size and the
            // container overflows on both axes.
            if (c.pinned) {
                if (!bandRenderSame(main.child, main.declared)) {
                    problems.push(`${c.what}: its child declared ` +
                        `${main.declared}px of ${main.name} with a shrink factor of ` +
                        `zero and laid out at ${main.child.toFixed(2)}px. ` +
                        `core.FlexShrink(0) is supposed to keep it at its own size and ` +
                        `let the ${main.want}px container overflow — that instruction ` +
                        `was unexpressible until core.ShrinkNone, because a zero in ` +
                        `that field is what every guard in the framework reads as ` +
                        `"nothing was set". A child that shrank anyway means this ` +
                        `target has gone back to discarding it.`);
                }
                if (!bandRenderSame(cross.child, cross.declared)) {
                    problems.push(`${c.what}: its child laid out at ` +
                        `${cross.child.toFixed(2)}px of ${cross.name} rather than the ` +
                        `${cross.declared}px it declared. Nothing shrinks across the ` +
                        `line whatever the factor says, so the cross axis is supposed ` +
                        `to be untouched by pinning the child`);
                }
                continue;
            }

            // The main axis: squeezed. A flex item's shrink factor defaults to
            // 1 and this child is empty, so its automatic minimum size is 0 and
            // there is nothing to stop it being taken down to the container's
            // own extent.
            if (!bandRenderSame(main.child, main.want)) {
                problems.push(`${c.what}: its child declared ${main.declared}px of ` +
                    `${main.name} — the container's main axis — and laid out at ` +
                    `${main.child.toFixed(2)}px in a ${main.want}px container. A ` +
                    `browser is supposed to shrink it to the line: the child is a flex ` +
                    `item with the default shrink factor and, being empty, no automatic ` +
                    `minimum to stop at. ` +
                    (main.child > main.want
                        ? `A child that keeps its main-axis size means the DOM has ` +
                          `stopped agreeing with Compose about the one axis they agreed ` +
                          `on, and the census in docs/platforms/native.md is wrong`
                        : `A child smaller than the container is not overflow at all`));
            }

            // The cross axis: spills. A declared size beats align-items:
            // stretch, and nothing shrinks a flex item across the line — so the
            // child keeps its own size and hangs out of the box.
            if (!bandRenderSame(cross.child, cross.declared)) {
                problems.push(`${c.what}: its child declared ${cross.declared}px of ` +
                    `${cross.name} — the container's cross axis — and laid out at ` +
                    `${cross.child.toFixed(2)}px. Nothing shrinks a flex item across ` +
                    `the line, so it is supposed to keep the size it asked for and ` +
                    `spill out of a ${cross.box}px container. A browser that clipped or ` +
                    `shrank it here would agree with Compose on both axes, and the ` +
                    `recorded divergence would be gone`);
            }
        }

    } finally {
        if (session) session.close();
        chrome.kill();
        server.close();
    }

    if (problems.length) {
        console.error("FAIL: the browser disagrees with the keyboard pattern:");
        for (const p of problems) console.error(`  ${p}`);
        process.exit(1);
    }
    console.log(`OK: roving tabindex, disabled focus, ArrowDown and the toolbar walk
    hold in a real browser, ${PALETTES.length} palette swatches paint the
    hexes the contrast census measures, ${WIDGETS.length} real widgets draw
    their own boundary tones on both edges, a sticky band pins, ${VALUE_RANGES.length}
    value ranges resolve the way core.Progress says a browser resolves them,
    ${BANDS.length} bands lay out identically in both inset arrangements at every
    offer — overflow included, which is where the SwiftUI solver does not — and
    hug their own natural width under every intrinsic keyword,
    ${BAND_RENDERS.length} real bands span their own tap targets, paint their own
    fill and are taller than
    their badges with real glyphs in them, and a fixed-size container squeezes its
    child along the main axis and lets it spill across — unless the child is
    pinned with core.FlexShrink(0), which until core.ShrinkNone was a declaration
    nobody could write`);
}

await main();
