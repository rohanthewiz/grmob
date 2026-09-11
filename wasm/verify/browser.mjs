// The facts a shimmed DOM cannot check, checked in a browser: four about the
// keyboard, two about paint, five about layout, and one about what a browser
// does with an accessibility value nobody here resolves.
//
// wasm/verify's other suites run the real grmob-runtime.js against dom.mjs — a
// few hundred lines that model element trees, attributes, listeners and which
// element holds focus. That is enough for almost everything, and its limits
// are stated in its own header: there is no layout, no bubbling, and `focus()`
// is an assignment, and nothing is ever painted. Twelve claims sit exactly in
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
//      them with real glyphs in them. It reads the paint as well as the rects:
//      the band's own fill, the ink its words are set in, the count pill, and
//      the digits inside it. A band is those things, and for a while only the
//      first was sampled — so a band painting its fill over an invisible label
//      passed every rect below it, and later a pill that rendered no number
//      did too. Where the rows it scans fall is measured against the face the
//      browser resolved rather than assumed, and the grayscale antialiasing
//      every one of those readings rests on is probed before any of them.
//  11. a fixed-size container squeezes its child along its main axis and lets
//      it spill across. core.Spacer became "a Box with a fixed size", and the
//      note closing that work recorded that Compose constrains a child to the
//      declared size where the DOM was believed to let it spill — for every
//      fixed-size container, not just a Spacer — with nothing anywhere having
//      asked. This asks, and the DOM's answer is per-axis rather than the
//      blanket one assumed: squeezed along the main axis (a flex item's shrink
//      factor defaults to 1 and an empty box has no automatic minimum to stop
//      at), and spilling across the cross one.
//  12. a pinned child keeps its base in a browser, and the siblings do not
//      depend on the order. internal/pinfixture pairs a Compose column with a
//      CSS one over one overflowing Row, and the CSS column is
//      GrMobFlexSolver's — this repository's own flex arithmetic, which check 9
//      just caught disagreeing with a real Chrome about how an overflow deficit
//      is divided when a child has padding. The pin fixture's children have
//      none, so the two rules coincide and the solver's answer is the web's: a
//      piece of reasoning about whether a known divergence applies, made by the
//      person who wrote the fixture and asked of nobody. This asks, and
//      recomputes nothing — every claim is one the fixture states, held against
//      measured pixels. The fixture states the CSS column outright now, so the
//      control row's 24/80/16 is held here too: three of its rows were pinned
//      by the claims below and that one was determined by nothing. It also
//      states the SPACING each Row inserts, which is where the two targets part
//      company a second time — a flex line charges its gap between every
//      adjacent pair and a Compose Row clamps each one to what is left, and
//      every case carried gap 0 until one of them did not.
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
import { foldVerdict } from "./fold.mjs";
import { bandTargetCensus, bandTargetRead } from "./bandtarget.mjs";

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
// The antialiasing probes the band grid's ink assertions rest on, one per
// distinct text declaration those assertions read. See gen.go's inkProbe for
// what a probe reproduces, and INK_PROBE_GRID for how the verdict is read.
const INK_PROBES = TRANSCRIPT_JSON.inkProbes || [];
// The character pairs gen.go's inkGlyphPerCharacter refuses a fixture string
// for, mounted so the face this browser resolved can be asked which of them it
// actually draws as one glyph. See gen.go's inkLigature and inkLigatureCensus.
const INK_LIGATURES = TRANSCRIPT_JSON.inkLigatures || [];
// internal/pinfixture, for check 12. One overflowing Row in four arrangements,
// with the answer a Compose Row gives already computed — the same table
// ios/verify solves through GrMobFlexSolver. Here it is laid out by a browser.
const PINS = TRANSCRIPT_JSON.pins || [];

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

// How far apart two "#RRGGBB" strings are, as the largest single-channel
// difference. A maximum rather than a sum, so a tolerance means the same thing
// whatever the colour: three channels each two off is as close as one channel
// two off, and both are the same rounding.
function channelDistance(a, b) {
    if (!a || !b) return 255;
    let worst = 0;
    for (let i = 1; i < 7; i += 2) {
        const d = Math.abs(parseInt(a.slice(i, i + 2), 16) - parseInt(b.slice(i, i + 2), 16));
        if (d > worst) worst = d;
    }
    return worst;
}

// A colour composited over a backdrop, for the themes whose ink carries an
// alpha channel.
//
// DefaultTheme's TextSecondary is #3C3C4399 — eight digits — so what a
// screenshot holds where the words are is not the declared colour but the
// declared colour at 60% over the band's fill. A six-digit colour is returned
// unchanged, which is every other bundled theme.
function over(color, backdrop) {
    if (!color || color.length !== 9) return color;
    const a = parseInt(color.slice(7, 9), 16) / 255;
    let out = "#";
    for (let i = 1; i < 7; i += 2) {
        const fg = parseInt(color.slice(i, i + 2), 16);
        const bg = parseInt(backdrop.slice(i, i + 2), 16);
        out += hex2(Math.round(fg * a + bg * (1 - a)));
    }
    return out;
}

// How far a sampled pixel may sit from the ink it is supposed to be. Text is
// antialiased and the composite above is arithmetic this file does rather than
// the browser, so an exact match is asking two roundings to agree.
const INK_EPSILON = 3;

// And what that tolerance has to stay far below.
//
// "Three is far below the distance to any other colour in a band" was the whole
// argument for the number, and it was a sentence: nothing measured the distance
// it is a claim about. A band paints three colours the label's composite could
// be confused with — the fill behind the words, the count pill, and the page
// under the band — and if the composite drifted within a few channels of one of
// them, the ink scan would go on passing while reading a pixel that agrees with
// two answers.
//
// So the distance is asserted per case, at INK_MARGIN times the tolerance. A
// factor rather than an absolute, because the thing being claimed is a RATIO:
// the tolerance exists to absorb two roundings, and it is safe exactly while
// the colours it discriminates between are an order away from it. Four is the
// smallest factor that is unambiguously an order — a composite 12 channels from
// the fill is still a colour a person can see against it — and the bundled
// themes clear it by a wide margin: the closest pair anything here paints is
// DefaultTheme's translucent secondary ink over its own band fill, at 109. So
// this fires on a palette that has stopped being readable rather than on
// ordinary theming.
const INK_MARGIN = 4;

// Where the ink is looked for, as fractions of the band it has ink in.
//
// One row was not enough, and the argument for it said so out loud: the label's
// vertical middle is the row "most likely" to cross a stem. Likely is not a
// property a check can rest on — a face whose x-height band happened to fall
// between two stems at that exact y reports a label with no ink in it, which
// fails in the safe direction and is still a failure, on whichever machine's
// Chrome picked that fallback face.
//
// # These used to be fractions of the line BOX
//
// The rows have to land inside the band where the run has ink — for a word,
// between its baseline and its x-height; an ascender or a descender reads
// backdrop where a lowercase word has none. That band's position inside a line
// box is a relationship between a theme's line-height and a font's vertical
// metrics, and this file chooses neither.
//
// So the fractions were of the box and a separate check held them to the
// measured band. They were 0.4/0.5/0.6 and the topmost sat 0.014px OUTSIDE it;
// they became 0.5/0.6/0.7, which for the three bundled faces put them inside a
// band running from 0.39 to 0.80 of the box. Then the grid grew a theme with an
// explicit line-height, and leading does not move the band — it moves the BOX
// around it:
//
//	face                     box      x-height band, as fractions of the box
//	AmberTheme, Default      15px     0.401 to 0.800
//	MaterialTheme            14px     0.391 to 0.786
//	Default + tall caption   29px     0.388 to 0.690   ← 0.7 is outside it
//
// The last row passed, by rounding: 0.7 of 29px is 20.3, the band ends at
// 20.01, and the device row is 20. A hundredth of a pixel, on the same check
// and for the same reason as the last one — which is the second time a fraction
// of the box has been a hundredth of a pixel from being a fraction of nothing.
//
// # So they are fractions of the band
//
// A fraction of the measured band is inside it by construction, at every
// line-height and in every face, and it needs no check to say so. That is the
// same measurement doing the work directly instead of being spent afterwards on
// checking a guess — and it is what "hold the number to the thing on the other
// side of the comparison" means here: the other side is the run's own ink
// extent, and this file now reads the rows off it.
//
// A quarter, a half and three quarters: evenly spaced, with a quarter of the
// band clear at each end so the outermost rows are not sitting on the baseline
// or on the x-height line itself, where a rounding either way lands on a
// horizontal edge of every glyph at once.
//
// What still has to be checked is the other half of the old pair — that three
// fractions are three ROWS. See inkRows.
const INK_ROWS = [0.25, 0.5, 0.75];

// Whether INK_ROWS is a set of fractions OF the band, which is the whole of
// what makes "the rows are inside the ink" true without a check.
//
// While the fractions were of the line box, a fraction outside the band was a
// measurable fault and inkBandFault reported it. Now the band is the coordinate
// system, so the guarantee is structural — and a structural guarantee is only
// as good as the numbers that define it. An entry at 1.3 puts a row below the
// baseline, into the descender space, and nothing downstream would say so: the
// three verdicts are ORs across the rows, so a row that reads backdrop costs
// the redundancy the three exist for and fails nothing.
//
// That is exactly the silence the three rows were added to end, one level up.
// So the constant is held to its own definition: inside the band, distinct, and
// in order, so that the outermost two really are the outermost two.
//
// Returns null when the fractions are fractions.
function inkRowsFault() {
    if (INK_ROWS.length < 2) {
        return `INK_ROWS is ${JSON.stringify(INK_ROWS)}. The redundancy the ink scan ` +
            `claims is that several rows cross a stem where one might not, and one ` +
            `fraction is the "most likely" row this replaced`;
    }
    for (let i = 0; i < INK_ROWS.length; i++) {
        const f = INK_ROWS[i];
        if (!(f > 0 && f < 1)) {
            return `INK_ROWS is ${JSON.stringify(INK_ROWS)} and ${f} is not strictly ` +
                `between 0 and 1. These are fractions of the band the run has INK in ` +
                `— from its baseline up by an x-height — and that is what makes "every ` +
                `scanned row crosses every glyph" true by construction rather than by ` +
                `a measurement. A fraction outside [0, 1] puts a row into the ` +
                `ascenders or the descenders, where a lowercase word has nothing, and ` +
                `nothing downstream reports it: the scan's three verdicts are ORs ` +
                `across the rows, so a row reading backdrop costs the redundancy the ` +
                `three exist for and fails no assertion`;
        }
        if (i > 0 && !(f > INK_ROWS[i - 1])) {
            return `INK_ROWS is ${JSON.stringify(INK_ROWS)}, which is not strictly ` +
                `increasing. Two equal fractions are one row read twice — the ` +
                `redundancy reported and not had — and an unsorted list makes the ` +
                `margin at each end of the band a different pair of numbers from the ` +
                `first and last entries`;
        }
    }
    return null;
}

// How many device pixels of the band the outermost scanned rows must keep
// clear of its two edges.
//
// # The argument this replaces
//
// INK_ROWS is a quarter, a half and three quarters, and the case for the two
// outer ones was a sentence: a quarter of the band is enough clearance that
// they are not sitting on the baseline or on the x-height line, where a
// rounding either way lands on a horizontal edge of every glyph in the run at
// once. That is the "most likely" argument the three rows replaced, in the
// coordinate system that replaced it — an assertion about a distance nothing
// measured, in a file that now measures the band it is a fraction of.
//
// A quarter of a band is a fraction; what a rounding has to survive is DEVICE
// PIXELS, and the two are joined by a font size and a display. The tall-caption
// theme in this grid is here precisely because that join moves.
//
// # Measured
//
// Across the twenty bands and their eight count pills, at dpr 1:
//
//	band              height   clear of the two edges
//	Material label    5.53px   1 and 2, or 2 and 1, by branch
//	Default label     5.99px   1 and 2, or 2 and 1, by branch
//	Default count     8.73px   2 and 2, or 2 and 3
//	tall label        8.75px   2 and 2
//	tall count       12.77px   3 and 4
//
// The tightest is one device pixel, on the two bundled palettes' labels — a
// 5.5px band with three rows a quarter apart has nowhere else to put them. So
// the floor is 1: a row must be OFF both edges, and a band too short for that
// is one where a quarter of it buys no clearance at all.
//
// Deliberately not 2. That would fail two of the three bundled palettes this
// grid exists to check, which would make it a number chosen against nothing
// rather than a claim about rounding — the shape of thing the last several of
// these have been about. What 1 rules out is the case the argument was really
// against: a band short enough, or a display coarse enough, that a quarter
// rounds onto the edge itself and all three rows share a glyph's horizontal
// boundary.
const INK_EDGE_CLEARANCE = 1;

// How much of a scanned row's own window has to still be glyph one device row
// either side of it, as a fraction of the columns scanned.
//
// # The rounding nobody had measured the size of
//
// INK_EDGE_CLEARANCE is 1, and the argument above it is that a row ON the
// baseline or ON the x-height line sits on a horizontal edge of every glyph in
// the run at once, so a single rounding takes all three rows off the ink
// together. The number that came out of that argument was a floor picked by
// what this grid happens to contain — 1 clears every band here and 2 fails
// fifteen of the twenty-eight boxes — and the rounding it is about was never
// measured at all. That is the same shape as the fraction it replaced: a
// distance asserted in a file that can read the pixels and see.
//
// So the pixels are read. The coverage of a device row is the fraction of the
// scanned window whose pixel is not the backdrop, and the question a clearance
// is really about is what one row of movement costs: for each row the scan
// reads, the coverage of the rows either side of it.
//
// # Measured, over the twenty bands and eight count pills
//
//	the row in question             coverage one device row out, at worst
//	the three the scan reads        0.239   (DefaultTheme, the long title)
//	a row on the BASELINE           0.100   — and 0.000 in every count pill
//	a row on the X-HEIGHT line      0.348   — and 0.000 in two count pills
//
// The floor is fixed from both sides by that table, which is the whole of why
// it is a number rather than a preference: above 0.100, so a row on the
// baseline fails it in every one of the twenty-eight boxes, and below 0.239, so
// every row the scan actually reads clears it. 0.15 sits in that gap with the
// wider margin on the side that has the measurements from more than one face.
//
// # What it does not settle, said plainly
//
// The x-height end. A row placed there has ASCENDERS above it — a cap, a `J`, a
// digit — so its outward neighbour comes back between 0.000 and 0.348, and this
// floor separates nothing at that end. What rules that row out is the other
// argument: a row above the x-height misses every round letter beside the
// ascender, which is why the band is the x-height band and why INK_ROWS is a
// set of fractions strictly inside it.
//
// That argument used to end here, as a sentence with no number under it — which
// is the same shape as the floor it stands in for, one paragraph up, before
// anybody read a pixel for it. It has one now: INK_ASCENDER_SEPARATION
// partitions the run's columns and measures what a row up there would actually
// be reading, and holds it in both directions.
const INK_ROW_ROUNDING = 0.15;

// Where those fractions actually land, in device pixels, for one measured band.
//
// The fractions are of the band and the redundancy they buy is three rows of a
// glyph's bitmap, and those are two different things joined by the band's
// height and a device pixel ratio. The three are a quarter of the band apart,
// so they become three distinct rows only while a quarter of the band is at
// least one device pixel: an x-height of 6 CSS px at dpr 2 gives 3 device
// pixels of separation and the argument holds comfortably, and a caption small
// enough that a quarter of its x-height rounds under a pixel makes all three
// the same y — a scan that is one row run three times, reporting a redundancy
// it does not have.
//
// A function of the band and the ratio rather than a claim about either, for
// the reason the fold guard is: what a font metric or a display does to a
// layout is not something this file gets to assume.
function inkRows(metrics, dpr) {
    const height = metrics.bandBottom - metrics.bandTop;
    return INK_ROWS.map((f) => Math.round((metrics.bandTop + height * f) * dpr));
}

// The band this text has ink in, and why the probe glyph decides it.
//
// # What the band is
//
// It runs from the baseline UP by the ink height of the SHORTEST glyph the
// content can contain, because a row is only worth scanning if every glyph in
// the run reaches it:
//
//	a word    "x" — a lowercase letter with no ascender and no descender is the
//	          shortest thing a word can be made of, and its ink height is the
//	          font's x-height. A row above it crosses an ascender and misses
//	          every round letter beside it.
//	a count   "0" — digits in every face this could resolve are lining figures,
//	          all of one height, so the band is that height and there is no
//	          shorter member to spoil it.
//
// Measured off the browser rather than derived: the page reports the inline
// text box (which is the font's content area, so its top is the baseline less
// the ascent) and the canvas metrics for the probe glyph in the element's own
// resolved font. Both come back in page coordinates, so INK_ROWS is a fraction
// of this and a scanned device row needs no leading arithmetic in between.
//
// # What can go wrong, and why it is one function
//
// The rows are a fraction of the band, so they cannot fall outside it — that
// check existed while they were fractions of the line BOX and is gone with the
// coordinate system that made it possible. What is left is the band itself
// being unusable, in four ways, and all four are one question asked of two
// boxes: the metrics never arrived, the canvas would not take the element's
// resolved font, the run wrapped, or the band is too short for three fractions
// of it to be three rows of pixels.
//
// Returns { rows, top, bottom } or { problem }. Never both, and never neither: a
// caller that got neither would scan an empty list of rows and report a box
// with no ink in it, which is the failure this whole apparatus exists to keep
// from being reported for the wrong cause.
//
// top and bottom are the band's own two edges in device rows — the same pair
// the clearance above is measured against — so a caller asking what a rounding
// would cost (see inkRoundingVerdict) is asking about the rows this function
// held the scan clear of, rather than rounding the band a second time.
function inkBandRows(where, subject, m, dpr, why) {
    if (!m) {
        return { problem: `${where}: no font metrics came back for ${subject}, so ` +
            `where the scanned rows would fall relative to the glyphs is unknown. ` +
            `INK_ROWS is three fractions of the band between a baseline and an ` +
            `x-height, and with no measurement of that band there is nothing to take ` +
            `a fraction of` };
    }
    if (m.wanted) {
        return { problem: `${where}: the canvas would not take ${subject}' resolved ` +
            `font (${m.wanted}) and stayed on ${m.font}, so every metric below would ` +
            `be another face's. The band the rows are fractions of is this face's ` +
            `x-height over this face's baseline, and a fallback's numbers are not ` +
            `that` };
    }
    if (m.lines !== 1) {
        return { problem: `${where}: ${subject} occupy ${m.lines} line boxes` +
            (m.text ? ` (${JSON.stringify(m.text)})` : "") + `. The band is measured ` +
            `off ONE inline text box: over two, the measurement is the first line's ` +
            `and the words the scan is looking for are spread over both. Either the ` +
            `run has grown past the band's width or the band has narrowed under it` };
    }
    const rows = inkRows(m, dpr);
    if (new Set(rows).size !== rows.length) {
        return { problem: `${where}: ${subject} have ink between ` +
            `y=${m.bandTop.toFixed(2)} and ${m.bandBottom.toFixed(2)} — ` +
            `${(m.bandBottom - m.bandTop).toFixed(2)}px, measured from the ` +
            `"${m.probe}" of ${m.font} — and INK_ROWS (${INK_ROWS.join(", ")}) of ` +
            `that at dpr ${dpr} resolves to rows ${rows.join(", ")}: two of them are ` +
            `the same row of pixels.\n\n` +
            `${why}. The three exist because the middle of a run is only the row ` +
            `"most likely" to cross a stem, and a scan that reads one row twice has ` +
            `one chance and reports three. A quarter of this band is under a device ` +
            `pixel, so either the type is smaller than anything this grid was built ` +
            `for or the display is coarser` };
    }
    // And how far the outermost rows actually sit from the two edges of the
    // band, which was a quarter by assertion until this line. See
    // INK_EDGE_CLEARANCE.
    const top = Math.round(m.bandTop * dpr), bottom = Math.round(m.bandBottom * dpr);
    const clear = [
        ["the x-height line", rows[0] - top],
        ["the baseline", bottom - rows[rows.length - 1]],
    ].filter(([, d]) => d < INK_EDGE_CLEARANCE);
    if (clear.length > 0) {
        return { problem: `${where}: ${subject} have ink between ` +
            `y=${m.bandTop.toFixed(2)} and ${m.bandBottom.toFixed(2)}, and INK_ROWS ` +
            `(${INK_ROWS.join(", ")}) of that at dpr ${dpr} puts the outermost rows ` +
            `at ${rows[0]} and ${rows[rows.length - 1]} — ` +
            clear.map(([edge, d]) => `${d} device pixel${d === 1 ? "" : "s"} clear of ` +
                edge).join(" and ") + `, against ${INK_EDGE_CLEARANCE}.\n\n` +
            `${why}. The fractions are inside (0, 1), so the rows are inside the band ` +
            `by construction — and "inside" is not the claim. A row on the baseline or ` +
            `on the x-height line sits on a horizontal edge of every glyph in the run ` +
            `at once, where one rounding takes all three rows off the ink together and ` +
            `the redundancy they exist for goes with them. A quarter of the band was ` +
            `argued to be enough room for that; this is the same band measured` };
    }
    return { rows, top, bottom };
}

// One box, scanned across the given rows between two x positions, for one ink
// over one fill.
//
// The three questions are asked in one pass because they are three readings of
// the same pixel, and separating them would be three passes over one rect:
//
//	notFill   some pixel is not the backdrop. A run of words rendered in the
//	          fill colour, or not rendered at all, has none.
//	ink       some pixel IS the declared ink. A stem's interior is unblended at
//	          any size a caption is set at, so the declared colour is present
//	          when it is the colour being used.
//	offBy     the worst pixel that is not a blend of the two. See offSegment.
//
// x0 and x1 are CSS pixels and half-open, so a caller can hand it a window
// narrower than the element — which the count pill's does, because a pill's own
// leading and trailing edges are the apexes of a 999-radius curve and every
// pixel there is a blend with the band behind it.
function scanInk(img, dpr, rows, x0, x1, fill, want) {
    const from = Math.round(x0 * dpr), to = Math.round(x1 * dpr);
    let notFill = false, ink = false, darkest = null, best = -1;
    let offBy = 0, stranger = null;
    for (const y of rows) {
        for (let x = from; x < to; x++) {
            const got = pixelAt(img, x, y);
            if (got === null) continue;
            if (got !== fill) notFill = true;
            // A stem's interior is unblended, so the pixel furthest from the
            // backdrop is the ink itself. Compared with a tolerance because the
            // composite is done in eight bits here and a browser's rounding is
            // its own.
            const away = channelDistance(got, fill);
            if (away > best) { best = away; darkest = got; }
            if (channelDistance(got, want) <= INK_EPSILON) ink = true;
            const off = offSegment(got, fill, want);
            if (off > offBy) { offBy = off; stranger = got; }
        }
    }
    return { notFill, ink, darkest, offBy, stranger, columns: to - from };
}

// How much of one device row of a window is glyph rather than backdrop.
//
// The predicate is the one inkExtent and surplusInk use — a pixel within
// INK_EPSILON of the fill is the backdrop and anything else is a glyph — so
// "coverage" means the same thing everywhere in this file, and a row's reading
// can be compared with another row's.
//
// Returns null when no pixel of the row is in the capture, which is the
// screenshot falling short of a rect rather than a row with nothing in it. The
// two would otherwise both come back as zero, and only one of them is a
// statement about the paint.
function inkRowCoverage(img, dpr, y, x0, x1, fill) {
    const from = Math.round(x0 * dpr), to = Math.round(x1 * dpr);
    let ink = 0, read = 0;
    for (let x = from; x < to; x++) {
        const got = pixelAt(img, x, y);
        if (got === null) continue;
        read++;
        if (channelDistance(got, fill) > INK_EPSILON) ink++;
    }
    return read === 0 ? null : ink / read;
}

// What separates a row that reads the words from a row that reads only the
// ascenders.
//
// INK_ROW_ROUNDING is a floor on how much ink a device row of movement costs,
// and its own comment records that it separates the BASELINE end of the band
// and separates nothing at the x-height end: an ascender puts the outward
// neighbour of a scanned row anywhere between 0.000 and 0.348, so no floor
// drawn up there is a bound at all. What actually rules out a scan row at or
// above the x-height line is a structural argument — a row up there misses
// every round letter beside the ascender — and that argument had no number
// attached to it.
//
// Here it is a number. Partition the run's columns into the ones whose ink
// reaches above the x-height (the ascenders and the capitals) and the rest, and
// ask what fraction of THE REST has ink on a given row:
//
//	at the x-height line       0.708 to 0.947 across the twenty bands
//	one device row above it    0.011 to 0.154
//
// A row above the line reads a tenth of the round letters. The line itself
// reads three quarters of them and more. INK_ASCENDER_SEPARATION is the
// midpoint of those two brackets, so the margin is the same on each side
// (0.276 below it, 0.278 above), and it is held in BOTH directions: the row
// above must be under it and the line itself must be over it.
//
// The second direction is not decoration. On its own the first is satisfied by
// a title with no round letters in it — "Illinois" set in ascenders would score
// nothing anywhere and pass — and the whole claim is that these two rows are
// looking at different things.
const INK_ASCENDER_SEPARATION = 0.43;

// How far above the x-height line a column's ink has to reach for that column
// to be an ascender.
//
// Not one device row. Round letters are drawn with an optical overshoot — an
// `o` is cut a hair taller than an `x` so the two look the same height — so a
// partition taken one row up puts the tops of o, e, a and c on the ascender
// side of it. Measured: that row still finds ink in up to 15% of the columns
// the partition below calls non-ascender, which is precisely those overshoots.
// Two rows is clear of them.
const INK_ASCENDER_PROBE = 2;

// The measurement, one box at a time.
//
// Returns { problem } or null. Asked of the label's words and not of the
// count's digits: a partition into ascenders and round letters is a fact about
// lower-case text, and a run of digits is cut to one height with no round
// letters below it to miss.
function inkAscenderVerdict(where, subject, img, dpr, m, x0, x1, fill) {
    const top = Math.round(m.bandTop * dpr);
    const from = Math.round(x0 * dpr), to = Math.round(x1 * dpr);
    const inked = (y, x) => {
        const got = pixelAt(img, x, y);
        return got !== null && channelDistance(got, fill) > INK_EPSILON;
    };
    // The partition, and then the two readings over one half of it.
    const plain = [];
    for (let x = from; x < to; x++) {
        if (!inked(top - INK_ASCENDER_PROBE, x)) plain.push(x);
    }
    const coverage = (y) => plain.filter((x) => inked(y, x)).length / plain.length;
    if (plain.length === 0) {
        return { problem: `${where}: every one of the ${to - from} columns of ` +
            `${subject} has ink ${INK_ASCENDER_PROBE} device rows above the x-height ` +
            `line, so there are no non-ascender columns to measure and the argument ` +
            `that keeps INK_ROWS off that line cannot be made about this run.\n\n` +
            `That argument is that a row at or above the x-height reads the ascenders ` +
            `and misses the round letters; a title with nothing but ascenders in it ` +
            `has no round letters to miss, and the fractions INK_ROWS are taken at ` +
            `would be resting on a sentence this grid no longer demonstrates` };
    }
    const above = coverage(top - 1), at = coverage(top);
    if (above >= INK_ASCENDER_SEPARATION) {
        return { problem: `${where}: one device row above the x-height line, ` +
            `${(above * 100).toFixed(1)}% of ${subject}' ${plain.length} ` +
            `non-ascender columns still have ink — against ` +
            `${(INK_ASCENDER_SEPARATION * 100).toFixed(0)}%.\n\n` +
            `INK_ROWS are three fractions strictly inside the band, and what keeps ` +
            `them off the x-height line is not INK_ROW_ROUNDING — that floor separates ` +
            `the baseline end and separates nothing here, because an ascender puts the ` +
            `outward neighbour anywhere up to 0.348. It is this: a row up there reads ` +
            `the ascenders and misses the round letters, so it is not a reading of the ` +
            `words. With the two rows scoring alike, that has stopped being true of ` +
            `this face and the fractions are a preference again` };
    }
    if (at <= INK_ASCENDER_SEPARATION) {
        return { problem: `${where}: at the x-height line itself only ` +
            `${(at * 100).toFixed(1)}% of ${subject}' ${plain.length} non-ascender ` +
            `columns have ink — against ` +
            `${(INK_ASCENDER_SEPARATION * 100).toFixed(0)}%, and ` +
            `${(above * 100).toFixed(1)}% one row above it.\n\n` +
            `This is the other half of the same bracket, and it is what stops the ` +
            `first half being satisfied for the wrong reason: "a row above the line ` +
            `reads few of the round letters" is trivially true of a run that has few ` +
            `round letters. The line itself is supposed to read most of them, and a ` +
            `run where it does not is one this comparison says nothing about` };
    }
    return null;
}

// What one device row of rounding would cost the scan, read off the screenshot.
//
// See INK_ROW_ROUNDING. inkBandRows holds the outermost scanned rows a device
// pixel clear of the band's two edges, on the argument that a row on an edge
// loses its ink to a rounding — and until this function that argument was a
// sentence about a magnitude nothing had measured. Here it is the measurement:
// each scanned row's two neighbouring rows are read, and the thinner of them is
// what the scan would be looking at if the band, the font metric or the display
// moved it by one.
//
// Also returns the reading just BELOW the band, which is the same question
// asked of the edge the clearance is really protecting: it is what a row on the
// baseline would score, and the grid holds the floor to being above it (see
// where this is called). A floor no reading in the grid falls under is a floor
// that separates nothing.
//
// Returns { problem } or { outside }.
function inkRoundingVerdict(where, subject, img, dpr, band, x0, x1, fill) {
    for (const y of band.rows) {
        const above = inkRowCoverage(img, dpr, y - 1, x0, x1, fill);
        const below = inkRowCoverage(img, dpr, y + 1, x0, x1, fill);
        if (above === null || below === null) {
            return { problem: `${where}: the rows either side of ${subject}' scanned ` +
                `row ${y} are not in the screenshot, which is ${img.width}×` +
                `${img.height}. What a rounding of one device row would cost this scan ` +
                `cannot be read from a capture that does not contain the row it would ` +
                `land on` };
        }
        const worst = Math.min(above, below);
        if (worst < INK_ROW_ROUNDING) {
            return { problem: `${where}: ${subject} are scanned on row ${y}, and one ` +
                `device row out from it the window is ${(worst * 100).toFixed(1)}% ` +
                `glyph — ${(above * 100).toFixed(1)}% above and ` +
                `${(below * 100).toFixed(1)}% below — against a floor of ` +
                `${(INK_ROW_ROUNDING * 100).toFixed(0)}%.

` +
                `The band runs from y=${band.top} to ${band.bottom} in device rows, and ` +
                `INK_EDGE_CLEARANCE is ${INK_EDGE_CLEARANCE} — the device rows the ` +
                `outermost scanned rows are held clear of both ends of it, because a row ` +
                `sitting on the baseline or on the x-height line sits on a horizontal ` +
                `edge of every glyph in the run at once, and one rounding either way ` +
                `takes all three rows off the ink together. This is that argument measured, and this row is on such an ` +
                `edge: over the twenty bands and eight pills the rows this scan reads ` +
                `come back at 0.239 at worst by the same reading, and a row on the ` +
                `baseline comes back at 0.100 and under` };
        }
    }
    const outside = inkRowCoverage(img, dpr, band.bottom + 1, x0, x1, fill);
    if (outside === null) {
        return { problem: `${where}: the row below ${subject}' baseline (${band.bottom + 1}) ` +
            `is not in the screenshot, which is ${img.width}×${img.height}. That row is ` +
            `what a scan with no edge clearance would be reading, and the floor the rows ` +
            `above were just held to is only worth anything while something in this grid ` +
            `falls under it` };
    }
    return { outside };
}

// How far the paint and the run's own client rect may part company at either
// end of the words, in device-independent pixels.
//
// # One quantity with two signs
//
// The run's client rect is where the browser says the glyphs are, and the paint
// is where they turn out to be. The two disagree at each end by a little, in
// either direction, and for the same reasons: the glyph's own bearings (an
// italic's, the outer half of an antialiased stem, a face whose advance width
// is wider or narrower than its outline) and a ROUNDING — a client rect ends at
// a fraction of a CSS pixel and the device pixel containing that fraction is
// part glyph and part backdrop, belonging to neither side.
//
// Both signs are read, and this is the bound on both:
//
//	outward   surplusInk holds everything in the element's box beyond the run
//	          ±this to the band's own fill. What it asserts is therefore "the
//	          far side of the box is backdrop" rather than "the rect is exact
//	          to the pixel", which is not a claim a client rect can support.
//	inward    inkExtent reads the paint's own leading and trailing edge and
//	          holds the rect to reaching within this of each. That is the half
//	          that says the rect is where the words ARE, rather than that the
//	          columns around it are empty.
//
// # Measured, in both directions
//
// With no gutter at all, five of the twenty bands report the pixel at the run's
// own trailing edge — AmberTheme's plain band ends its words at x=99.48 and the
// device pixel spanning 99 to 100 is half a stem and half backdrop. Across the
// whole grid the paint and the rect part company by at most 0.52px outward
// (AmberTheme's plain band) and 0.71px inward (the tall-caption theme's long
// title, whose last glyph's advance runs past its outline), so one device pixel
// of gutter clears every band in either direction and two is that measurement
// with a pixel of margin — against a surplus 180px wide on the narrowest shape,
// and against the tens of pixels a rect in the wrong place would be out by.
const INK_RUN_GUTTER = 2;

// The layout's own quantum, in CSS pixels.
//
// Chrome stores lengths as LayoutUnits of a 64th of a CSS pixel, and a run's
// client rect is its advance width rounded up to one of them. That is not a
// tolerance chosen against a measurement: it is the size of the grid the two
// numbers live on, and it is what says how far apart they may legitimately be.
const LAYOUT_UNIT = 1 / 64;

// --------------------------------------------------------------------------
// The windows the ink scan reads, held to the layout that produced them
// --------------------------------------------------------------------------
//
// # The failure that does not look like one
//
// Removing a guard turns a pass into a failure, and a break-test for that is
// easy: take the guard out and watch the check fire. The surplus/extent pair
// fails the other way round. inkExtent reads the columns of the label's rect
// that are not the band's fill and calls the first and last of them the ends of
// the words. Widen the run rect by six pixels AND paint something in those six
// pixels, and the extent moves with the paint, the comparison is about a
// different quantity, and the check that exists for exactly that defect PASSES.
// Nineteen failures either way — the count is identical on both sides — and
// only the surplused band's own message tells the two runs apart.
//
// The general shape is worth naming, because nothing here had looked for it: a
// check can be made to pass BY a defect when the quantity it measures is
// derived from a value that nothing holds. Ordering it after a check that
// happens to constrain that value — which is what the `else` in the label scan
// does — makes the PAIR sound and leaves the value itself unheld. The pair is
// still worth having; it is not a substitute for holding the value.
//
// # So the values are held
//
// Every window this scan reads is a rect or an offset that came from somewhere,
// and each of them is now a claim of its own, made against the layout and
// before a single pixel is read:
//
//	the label's run rect   Range.getClientRects on the text node. Held to
//	                       starting at the leading edge of the label's own
//	                       content box and to ending inside it —
//	                       inkRunRectFault.
//	the digit window       r.badge.x plus gen.go's badgePadLeft, which is
//	                       Go's reading of the pill's Style. Held to the
//	                       padding the BROWSER resolved on that same element —
//	                       inkBadgePadFault.
//	the three rows         fractions of a band measured off the face this
//	                       browser resolved. Already held, by inkRowsFault,
//	                       inkBandRows and INK_EDGE_CLEARANCE.
//
// None of the three reads a pixel, so no amount of paint can move any of them,
// and none of them depends on another check having run first.

// One computed length, as a number.
//
// getComputedStyle serialises a resolved length as "8px". Anything else — a
// keyword, an empty string from an element nothing was read from — is a value
// this arithmetic cannot use, and null says so rather than letting parseFloat
// hand back a NaN that compares false against everything and reports nothing.
function inkLength(own, prop) {
    const raw = own ? own[prop] : undefined;
    if (typeof raw !== "string" || !raw.endsWith("px")) return null;
    const n = parseFloat(raw);
    return Number.isFinite(n) ? n : null;
}

// The label's run rect, held to the box it is supposed to be a rect inside.
//
// The run rect is the browser's answer to "where are the glyphs" and it is the
// origin of every reading the label scan makes. This is the layout's own answer
// to the same question, from the other side: the element's border box less its
// border and padding is where its inline content may be, and a single-line run
// under `text-align: start` begins exactly at the leading edge of it.
//
// Both directions matter and they catch different defects.
//
//	the leading edge, exactly    a run displaced into the empty half of a
//	                             stretched box. Nothing else asks: the rows,
//	                             the scan and the surplus are all measured
//	                             FROM the rect, and a displaced rect satisfies
//	                             all three.
//	the trailing edge, inside    a rect that has grown past its own box.
//	the width, independently      a rect wider than the words. The containment
//	                             bound above cannot see this one: a stretched
//	                             label leaves up to 218px of slack, so six
//	                             extra pixels of run are still comfortably
//	                             inside the box. The second answer is the
//	                             advance width the same face gives for the same
//	                             string, measured through the canvas — the
//	                             layout's arithmetic checked against the font's,
//	                             which is two answers rather than one.
//
// Measured: the run starts exactly at the label's content edge in all twenty
// bands (a difference of 0.000 in every one) and ends between 0 and 218px short
// of the trailing edge, depending on whether the branch stretches its label or
// hugs it. So the leading claim is an equality and the trailing one is a bound.
//
// And the rect is wider than the advance by 0.0044 to 0.0137px across the
// twenty — every one of them positive, and every one of them exactly
// `ceil(advance * 64) / 64 - advance`, which is the layout rounding the width up
// to a LayoutUnit. So the bound on that difference is LAYOUT_UNIT and comes from
// the derivation rather than from the measurements: any string may land anywhere
// in that quantum. It is held two-sided, because which way a disagreement in a
// double's last bits would fall is not worth betting on and the second side
// costs a 64th of a pixel — against a defect that has to move the rect by a
// whole one to change any verdict downstream.
//
// Written for one writing direction, and it says so rather than quietly
// measuring the wrong edge: under `direction: rtl` a start-aligned run begins at
// the box's RIGHT content edge, and this arithmetic would report every band as
// displaced by its own trailing slack.
function inkRunRectFault(where, box, own, m) {
    const runX = m.runX, runW = m.runW;
    const dir = own ? own["direction"] : undefined;
    if (dir !== "ltr") {
        return `${where}: the label resolves direction as ${dir === undefined
            ? "nothing that was read" : dir}, and the run-rect arithmetic below is ` +
            `written for ltr — it asks whether the words begin at the LEFT content ` +
            `edge of their box. Under any other writing direction a correct band ` +
            `would be reported as displaced by however much trailing slack its ` +
            `branch leaves, so the check refuses rather than measuring the wrong edge`;
    }
    const parts = [
        ["border-left-width", inkLength(own, "border-left-width")],
        ["padding-left", inkLength(own, "padding-left")],
        ["border-right-width", inkLength(own, "border-right-width")],
        ["padding-right", inkLength(own, "padding-right")],
    ];
    const unread = parts.filter(([, v]) => v === null);
    if (unread.length > 0) {
        return `${where}: the label resolves ${unread.map(([n]) => n).join(", ")} as ` +
            `${unread.map(([n]) => JSON.stringify(own ? own[n] : undefined)).join(", ")}, ` +
            `and the content box those numbers describe is where the run's rect is ` +
            `held to be. Every reading of this box is taken from that rect, so a ` +
            `content edge nothing could compute is a rect nothing is behind`;
    }
    const [, bl] = parts[0], [, pl] = parts[1], [, br] = parts[2], [, pr] = parts[3];
    const lead = box.x + bl + pl;
    const trail = box.x + box.w - br - pr;
    const why = `\n\nThe three ink rows are fractions of a band measured off that ` +
        `rect's top, the ink scan sweeps it, and surplusInk holds everything outside ` +
        `it to the backdrop — all three are taken FROM the rect, so none of them is a ` +
        `statement about where the rect is. This one is, and it is made against the ` +
        `layout rather than against the paint: it needs no pixel, so nothing painted ` +
        `in this box can move it.`;
    if (!bandRenderSame(runX, lead)) {
        return `${where}: the label's box starts at x=${box.x.toFixed(2)} with ` +
            `${bl}px of border and ${pl}px of padding, so its content begins at ` +
            `${lead.toFixed(2)} — and the browser puts the run of words at ` +
            `${runX.toFixed(2)}, ${Math.abs(runX - lead).toFixed(2)}px ` +
            `${runX > lead ? "further in" : "before it"}.` + why;
    }
    if (runX + runW > trail + BAND_EPSILON) {
        return `${where}: the label's content box ends at x=${trail.toFixed(2)} ` +
            `(${box.x.toFixed(2)} + ${box.w.toFixed(2)} less ${br}px of border and ` +
            `${pr}px of padding) and the browser puts the end of the run at ` +
            `${(runX + runW).toFixed(2)}, ${(runX + runW - trail).toFixed(2)}px past ` +
            `it. A run that has grown out of its own box is a rect nothing below can ` +
            `be a statement about.` + why;
    }
    // And the width, against the only other thing that knows it.
    //
    // The containment bound above is slack by construction on the stretched
    // branches — the widest of them leaves 218px between the words and the end
    // of the box — so a run rect a few pixels too wide is entirely inside it.
    // This is the answer that does not come from the layout at all.
    if (typeof m.runAdvance !== "number" || !Number.isFinite(m.runAdvance)) {
        return `${where}: the run's rect is ${runW.toFixed(4)}px wide and the face ` +
            `this browser resolved reports no advance for the same string, so there ` +
            `is one answer again. Every reading of this box is taken from that rect ` +
            `and nothing else knows how wide the words are.` + why;
    }
    const off = runW - m.runAdvance;
    if (Math.abs(off) >= LAYOUT_UNIT) {
        return `${where}: the browser's layout makes the run of words ` +
            `${runW.toFixed(4)}px wide and the same face's own advance for the same ` +
            `string is ${m.runAdvance.toFixed(4)}px — ${Math.abs(off).toFixed(4)}px ` +
            `${off > 0 ? "wider" : "narrower"}, against a LayoutUnit of ` +
            `${LAYOUT_UNIT}.\n\nThe rect is the advance rounded up to one of those, ` +
            `so the two are the same number on two grids and may differ by less than ` +
            `one quantum. A larger difference is a rect that is not the width of ` +
            `these words — which is the defect the surplus/extent pair can be made to ` +
            `MISS rather than to report: widen the rect and paint something in the ` +
            `extra columns, and inkExtent finds ink out there, agrees with the rect, ` +
            `and passes. The count of failures is identical on both sides of that ` +
            `injection. This asks the question the other way round, without a pixel.`;
    }
    return null;
}

// And the digit window, held the same way.
//
// The window the count is scanned in is `r.badge.x + badgePadLeft` to
// `r.badge.x + r.badge.w - badgePadRight`: one number from the browser's layout
// and two from gen.go, which read them off the pill's core.Style. The two
// halves have never been compared. A padding that reached the element through a
// stylesheet, a UA default or a runtime mapping — the same list inkOwnFault
// exists for — moves the window without moving either number gen.go sent, and
// the scan goes on reporting about a rect that is not the one it names.
//
// Which way it fails is worth stating. A window too NARROW reads digits and
// passes on fewer of them; a window too WIDE reaches the pill's 999-radius
// corner, where every pixel is a blend with the band rather than with the pill,
// and reports a third colour for painting that is correct. Both are the scan
// measuring a different quantity than the one its message names.
function inkBadgePadFault(where, own, padLeft, padRight) {
    const sides = [
        ["padding-left", padLeft, "badgePadLeft"],
        ["padding-right", padRight, "badgePadRight"],
    ];
    for (const [prop, sent, field] of sides) {
        const got = inkLength(own, prop);
        if (got === null) {
            return `${where}: the count pill resolves ${prop} as ` +
                `${JSON.stringify(own ? own[prop] : undefined)}, and the digit window ` +
                `is that padding in from the pill's own rect. gen.go sends ` +
                `${sent}px for it, read off the node's core.Style; with the browser's ` +
                `answer unreadable the two cannot be compared and the window is a ` +
                `number nothing is behind`;
        }
        if (!bandRenderSame(got, sent)) {
            return `${where}: gen.go read ${sent}px of ${field} off the count pill's ` +
                `core.Style and the browser resolves ${prop} as ${got}px.\n\n` +
                `The digits are scanned between those two paddings, so the window is ` +
                `Go's arithmetic applied to the browser's rect and the two have come ` +
                `apart. A padding that arrived through a stylesheet, a UA default or a ` +
                `runtime mapping is on the element and in no Style — the same gap ` +
                `inkOwnFault exists for — and it moves this window without moving ` +
                `anything gen.go sent. Too narrow and the scan reports about fewer ` +
                `digits than it names; too wide and it reaches the pill's 999-radius ` +
                `corner, where every pixel blends with the BAND rather than the pill ` +
                `and correct painting reads as a third colour`;
        }
    }
    return null;
}

// Where the ink in a box actually starts and ends, read off the screenshot.
//
// # The rect nothing was behind
//
// The run's client rect is the browser's own answer to "where are the glyphs",
// and every reading the label scan makes is taken relative to it: the three
// rows are fractions of a band measured from that rect's top, the ink scan
// sweeps it, and surplusInk holds everything outside it to the backdrop. Those
// are two halves of one number, and a rect that reported the wrong PLACE
// satisfies both of them — put it over half the words and the half it covers
// has ink in it (scanInk is happy) while the half it does not is outside the
// window and holds ink (surplusInk fires, but naming a colour rather than a
// rect); put it beside them and the ink is all "surplus". Neither half is a
// statement that the rect is where the words are, because both are taken FROM
// the rect.
//
// This is the other source. A screenshot knows where the glyphs are without
// asking the layout: the columns of the element's box that are not the band's
// own fill are the ones a glyph reached. Holding the client rect to that is the
// browser's arithmetic checked against the browser's paint, which is two
// answers rather than one — and it is what makes the run rect a measurement
// instead of a premise.
//
// # What it rests on, which is not free
//
// "The columns that are not the fill are the ones a glyph reached" is true of
// the label's box because surplusInk has just held everything in that box and
// outside the run to the backdrop. Anything else painted there on these rows —
// a focus ring, a chevron's tint, a second run — would be ink to this function
// and would move the extent, and the comparison would then be between a client
// rect and some other object's paint.
//
// So the caller asks this only where that has been established, and says so at
// the call site: the surplus check runs first and this one is in its else. The
// two were sound in that order before anybody wrote the dependency down, which
// is a different thing from being sound.
//
// Half-open on the right in CSS pixels: `to` is the far edge of the last device
// column with ink in it, so a rect and an extent that agree exactly come back
// as the same pair of numbers.
//
// Returns null when the box holds no ink at all, which scanInk's own notFill
// verdict reports in its own words.
function inkExtent(img, dpr, rows, x0, x1, fill) {
    const from = Math.round(x0 * dpr), to = Math.round(x1 * dpr);
    let first = null, last = null;
    for (const y of rows) {
        for (let x = from; x < to; x++) {
            const got = pixelAt(img, x, y);
            if (got === null) continue;
            // The same predicate surplusInk uses, so "ink" means one thing on
            // both sides of the comparison: a pixel within the tolerance of the
            // fill is the backdrop, and anything else is a glyph.
            if (channelDistance(got, fill) <= INK_EPSILON) continue;
            if (first === null || x < first) first = x;
            if (last === null || x > last) last = x;
        }
    }
    if (first === null) return null;
    return { from: first / dpr, to: (last + 1) / dpr };
}

// Whether anything in a box but outside a window is something other than the
// backdrop, and if so where.
//
// The other half of scanInk. That one reads the run of glyphs and asks three
// questions about it; this reads everything the element's rect holds BESIDE the
// run and asks one — is it the fill? — because on every branch where the text
// box is stretched, that is what the rest of the rect is by construction.
//
// Compared with a tolerance rather than exactly, for scanInk's reason: the
// screenshot's eight bits and this file's own compositing arithmetic are two
// roundings, and an exact match asks them to agree.
//
// Returns null when the surplus is all backdrop, which is every bundled theme.
function surplusInk(img, dpr, rows, box, keepFrom, keepTo, fill) {
    const from = Math.round(box.x * dpr), to = Math.round((box.x + box.w) * dpr);
    const skipFrom = Math.round(keepFrom * dpr), skipTo = Math.round(keepTo * dpr);
    for (const y of rows) {
        for (let x = from; x < to; x++) {
            if (x >= skipFrom && x < skipTo) continue;
            const got = pixelAt(img, x, y);
            if (got === null) continue;
            if (channelDistance(got, fill) > INK_EPSILON) {
                return { at: x / dpr, got };
            }
        }
    }
    return null;
}

// How far a scanned pixel may sit off the line between the band's fill and the
// label's composited ink.
//
// See offSegment for what the line is. The number is INK_EPSILON itself rather
// than one of its own, and that is the argument for it: a pixel within the ink
// tolerance of the composite is accepted AS the ink two lines below, so the
// distance at which "off the line" starts mattering is exactly the distance at
// which the scan stops being able to tell. A looser tolerance here would admit
// pixels the ink test is already willing to call ink; a tighter one would be a
// second number nothing decides.
//
// It is a bound the bundled themes clear by a wide margin, which is the other
// half of the claim: the fifteen bands and their count pills come in at 0.9
// channels off the line at the worst — DefaultTheme's translucent secondary
// ink, whose blend has the most rounding to do — against a floor of 3.
// Antialiasing is exact arithmetic on this line (see offSegment, and
// INK_PROBE_GRID for the rendering mode that makes it so), and a blend of two
// eight-bit colours rounded to eight bits is at most half a channel off it, so
// what is being absorbed is the browser's rounding and nothing else.
const OFF_SEGMENT_EPSILON = INK_EPSILON;

// How far a pixel is from being a blend of the two colours the label's box is
// declared to hold.
//
// # Why a segment and not a set of colours
//
// confusableInk asks whether the composite is far from the three colours the
// band DECLARES, and a screenshot holds more than three: the chevron's tint, a
// focus ring, a pressed state, anything a future band draws inside the label's
// rect. None of them is in the fixture today and each is a colour a pixel in
// that box could be, so the guarantee "this scan can tell the ink from
// everything else here" was a claim about the declaration rather than about the
// capture.
//
// This is the other direction, and it is the one the capture can answer.
// Antialiasing is linear: a glyph pixel at coverage c is drawn as
// `fill + c * (want - fill)`, exactly — the alpha in a translucent ink
// multiplies into c and comes out on the same line — so every legitimate pixel
// in the label's box lies ON the segment between the fill and the composited
// ink. A pixel off that segment is a colour neither of them blends into, which
// is a third thing drawn inside the rect, which is a colour confusableInk was
// never given the chance to rule out.
//
// # One metric, and it used to be two
//
// Every tolerance in this file is a MAX-CHANNEL distance — channelDistance's
// metric, chosen there so that a number means the same thing whatever the
// colour. This function used to project the pixel onto the segment with the
// EUCLIDEAN dot product and then report the max-channel distance from that
// point, which is two metrics in four lines: the t it picked was the one
// minimising a quantity nobody compares anything against, and the number handed
// to OFF_SEGMENT_EPSILON was therefore not the number the projection had
// minimised. They agree about zero and disagree about everything else, so a
// pixel could be reported as further off the line than the line's own closest
// max-channel point is.
//
// So the minimisation is done in the metric the answer is expressed in. Each
// channel's signed error is affine in t, the max of their absolute values is
// convex and piecewise linear, and the minimum of such a function over [0, 1]
// is at an endpoint or where two of its pieces cross — which is a short list to
// enumerate exactly rather than a search to converge.
//
// Returned as a distance rather than a boolean so the caller can name the worst
// one, which is the only thing a reader can act on.
function offSegment(got, fill, want) {
    const rgb = (hex) => [1, 3, 5].map((i) => parseInt(hex.slice(i, i + 2), 16));
    const p = rgb(got), a = rgb(fill), b = rgb(want);
    // The blend at t is a + t*(b - a), so channel i is off by
    // `base[i] + slope[i]*t` — affine in t, and the answer is the largest of
    // the three in absolute value.
    const base = [0, 1, 2].map((i) => p[i] - a[i]);
    const slope = [0, 1, 2].map((i) => -(b[i] - a[i]));
    const at = (t) => Math.max(
        Math.abs(base[0] + slope[0] * t),
        Math.abs(base[1] + slope[1] * t),
        Math.abs(base[2] + slope[2] * t));

    // The six affine pieces: each channel's error and its negation, since the
    // absolute value of an affine function is the max of the two.
    const pieces = [];
    for (let i = 0; i < 3; i++) {
        pieces.push([base[i], slope[i]]);
        pieces.push([-base[i], -slope[i]]);
    }
    // The endpoints, and every crossing of two pieces. A pixel "past" either
    // end of the segment is not a blend of the two colours at all, so t is
    // confined to [0, 1] and such a pixel is measured against the end it is
    // past rather than against a point outside the segment.
    const candidates = [0, 1];
    for (let j = 0; j < pieces.length; j++) {
        for (let k = j + 1; k < pieces.length; k++) {
            const dm = pieces[j][1] - pieces[k][1];
            if (dm === 0) continue;
            const t = (pieces[k][0] - pieces[j][0]) / dm;
            if (t > 0 && t < 1) candidates.push(t);
        }
    }
    let worst = Infinity;
    for (const t of candidates) worst = Math.min(worst, at(t));
    return worst;
}

// Whether a composited ink is too close to something else the band paints for
// the scan to tell them apart, and if so which.
//
// The three candidates are everything a pixel inside a band can legitimately
// be: the fill the words sit on, the pill beside them, and the page under the
// band. gen.go already refuses the two cases that are widget faults — a label
// declared in the band's own fill, a pill declared in it — but it compares
// DECLARATIONS, and the thing this scan reads is a composite. An eight-digit
// ink at a low enough alpha lands arbitrarily close to its own backdrop while
// the two declarations stay different strings.
//
// "Everything a pixel inside a band can legitimately be" is a claim about what
// the band DECLARES, and a screenshot holds whatever was painted: a chevron's
// tint, a focus ring, a pressed state. None is in the fixture today and each is
// a colour a label pixel could be, so this guard would clear a case whose box
// contains something it was never shown. offSegment is the other side of that
// — it asks the capture whether the box holds anything but blends of these two
// colours — and the pair is what makes the tolerance mean what it says.
//
// Returns null when the case discriminates, which is every bundled theme.
function confusableInk(want, b) {
    for (const [what, color] of [
        ["the band's own fill, which is what the words are drawn on", b.fill],
        ["the count pill beside them", b.badgeFill],
        ["the page the band sits on", b.page],
    ]) {
        if (!color) continue;
        const distance = channelDistance(want, color);
        if (distance <= INK_EPSILON * INK_MARGIN) return { what, color, distance };
    }
    return null;
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

// The antialiasing probes, mounted with the bands and read off the same
// screenshot.
//
// # The rendering mode the whole ink scan rests on
//
// offSegment's argument is that antialiasing is linear: a glyph pixel at
// coverage c is `fill + c * (want - fill)`, exactly, so every legitimate pixel
// in a text box is ON the segment between the two colours. That is GRAYSCALE
// antialiasing's arithmetic. LCD subpixel antialiasing gives each channel its
// own coverage — that is the entire point of it — so an ordinary glyph edge
// gets a coloured fringe and lands well off the line, and INK_EPSILON's exact
// match on a stem's interior is resting on the same thing.
//
// Headless Chrome turns subpixel antialiasing off, which is why every band
// comes in at a fraction of a channel off a line it has no business being on.
// No line of this file said so. If a Chrome ever arrived with it on, or a flag
// here changed, the failure would be every band reporting a third colour inside
// its label's box — a message about a palette, for a cause that is not one.
//
// # One probe stood in for thirty, and now each box names its own
//
// The mode used to be asked once, of black words on a white page at one size in
// one box. What that licensed was a claim about every label and every count in
// the grid, in colours it did not paint, at weights it did not set, at whatever
// alpha a theme's ink carries — and Chrome picks a text rendering path PER
// ELEMENT. So the measurement that answered was one, and the boxes it answered
// for are the ones the tolerance is spent on.
//
// gen.go now builds one probe per distinct text DECLARATION the scan reads —
// the scanned node's own Style over a white box, with the ink replaced by black
// at the same alpha — and each band names the probe that answers for its label
// and for its count. Black at any alpha over white is still a grey, so the
// verdict is the one a screenshot can state without begging the question: under
// grayscale antialiasing every blend of two greys is a grey, so a pixel whose
// channels differ is a subpixel-rendered one. See gen.go's inkProbe for what a
// probe can and cannot reproduce.
//
// # Why they are one Row
//
// The grid is a fold budget — a band pushed past the bottom of the viewport has
// a rect and no pixels — and the probes are eleven boxes that would otherwise
// cost eleven bands' worth of height. Side by side they cost one. Each declares
// its own width and refuses to shrink, and the scan holds the rendered rect to
// that width: a Row that ran out of room would squeeze them into slivers, and a
// probe read across two device pixels reports a grey page because it looked at
// almost none of one.
const INK_PROBE_GRID = {
    Type: "Row",
    Style: {
        Padding: { Top: 0, Right: 0, Bottom: 0, Left: 0 }, Gap: 0,
        AlignItems: "flex-start",
    },
    Children: INK_PROBES.map((p) => JSON.parse(p.tree)),
};

// Each probe's own path in the grid. The probe Row is the last child, so the
// bands keep theirs, and a probe is a box inside it.
const inkProbePath = (i) => `root/${BAND_RENDERS.length}/${i}`;

// And the element inside it that actually draws the glyphs.
//
// A probe is a white Box with one Text child carrying the scanned node's Style
// (gen.go's inkProbeFor). The Box is what the screenshot scan reads — it is the
// rect with the ground and the fringe in it — and it is NOT the element whose
// rendering path is in question: the Box declares a background, a width and a
// flex-shrink and no typography at all, so it inherits the page's 16px/400 and
// resolves nothing the label resolves.
//
// Comparing the label's text node against that Box was comparing two different
// kinds of thing, and it showed: twenty-five properties differed for no reason
// but the mix-up — display, flex-direction, font-size, font-weight,
// line-height, height, every padding and every border-radius. Read here
// instead, the two agree on all of them, which is the comparison the probe's
// whole argument wanted to make.
//
// It also puts the Box itself INTO the ancestry sweep, where it belongs. As a
// strict ancestor of the element being scanned it is now held to
// INK_PATH_PROPS like every other box above it, rather than being neutral by
// construction and unmeasured.
const inkProbeTextPath = (i) => `${inkProbePath(i)}/0`;

// The refused pairs, mounted.
//
// One Row of bare Text nodes carrying the first probe's declaration — see
// gen.go's inkLigatureRow for why that declaration and not the page's. They
// are read for their GLYPH COUNTS and never for a pixel, so unlike the probes
// they declare no width, no ground and no shrink rule: what a glyph count
// needs is a laid-out element, and what the probes' rules are for is a rect
// that can be scanned across.
//
// Their own Row rather than the end of the probes', because the probes'
// widths are declared and held to, and nine more items sharing that Row is a
// way to make eleven probes report as slivers over a question about faces.
const INK_LIGATURE_GRID = {
    Type: "Row",
    Style: {
        Padding: { Top: 0, Right: 0, Bottom: 0, Left: 0 }, Gap: 4,
        AlignItems: "flex-start",
    },
    Children: INK_LIGATURES.map((l) => JSON.parse(l.tree)),
};

// Each pair's own path. The ligature Row is the last child of the grid, after
// the probe Row, so both of the paths above keep theirs.
const inkLigaturePath = (j) => `root/${BAND_RENDERS.length + 1}/${j}`;

// How far apart a pixel's channels may be before it is a coloured fringe rather
// than a grey.
//
// Zero would be the arithmetic answer — a blend of two greys is a grey in exact
// arithmetic — and one absorbs the browser's eight-bit rounding, which can round
// three identical values to two different integers only if they were not
// identical to begin with. A real subpixel fringe is not near this: the whole
// mechanism exists to put one channel at full coverage while another is at
// none, and it lands tens of channels apart.
const SUBPIXEL_EPSILON = 1;

// And what the probe is supposed to measure, which is not the same number.
//
// The paragraph above says the arithmetic answer is zero and that the one is
// there to absorb a rounding — and it never said what the probe actually
// reports. It reports zero: a blend of two greys rounded to eight bits is a
// grey exactly, because the three channels enter the rounding as one value.
// So the whole of SUBPIXEL_EPSILON is margin, and nothing recorded that.
//
// That is the same shape as a tolerance whose distance to the thing it
// discriminates against nothing measured (see INK_MARGIN). A margin that is
// silently spent is a margin nobody notices leaving: the day a channel comes
// back one apart, this file's account of why the number is one has stopped
// being true, and the check would go on passing until it drifted to two.
//
// So the floor is asserted separately from the ceiling. Above SUBPIXEL_EPSILON
// is subpixel antialiasing and the whole ink scan is unsound; between the two
// is a page that is still grey and an explanation that is not.
const SUBPIXEL_FLOOR = 0;

// What an ANCESTOR can do to the way an element's glyphs are drawn, and what
// each property reads as when it is doing nothing.
//
// # The half of the question a probe cannot copy
//
// gen.go's probes carry the scanned node's own Style, copied whole, so every
// declaration the element makes about itself travels with it. What no copy of a
// declaration can reproduce is what sits ABOVE the element: a transform, an
// opacity below one, a filter, a `will-change` anywhere up the chain puts the
// subtree on a compositing layer, and a browser draws text on a composited
// layer by a different route than text painted straight into the page. The
// probe would stay on the old route and go on reporting a grey box while the
// band it answers for had moved.
//
// That is the same shape as the fault the per-declaration probes replaced —
// one measurement standing in for boxes it was not taken of — one level up, and
// it was recorded as a known gap and checked by nothing.
//
// # Why neutrality rather than a comparison
//
// The two chains are not the same shape and cannot be: a band's label sits
// inside a control inside a Row inside a page box, and a probe sits in the
// probe Row. Holding one chain to the other would be comparing two structures
// that have no reason to match. What CAN be said of both, and is the thing the
// probe's argument actually needs, is that neither chain contributes anything:
// with every ancestor neutral on all of these, the declaration is the whole of
// what decides the rendering path, and copying the declaration copies the
// question.
//
// So both sides are held to it — the scanned boxes and the probes. A band whose
// ancestry stopped being neutral suppresses its own scan (the probe no longer
// answers for it); a probe whose ancestry stopped being neutral suppresses
// every box it answers for (the probe is no longer measuring the path the bands
// are on).
//
// The values are Chrome's computed-style defaults for a box that declares none
// of this. `opacity` is the string "1" because a computed opacity is serialised
// as a number-valued string; the rest are keywords.
const INK_PATH_PROPS = {
    // Each of these puts the subtree on its own compositing layer, or asks the
    // browser to prepare to.
    transform: "none",
    perspective: "none",
    filter: "none",
    backdropFilter: "none",
    willChange: "auto",
    // Below 1 the subtree is composited and then blended, which is the same
    // move by another name.
    opacity: "1",
    // Blending and isolation both force a stacking context and a separate
    // buffer for the subtree to be composited out of.
    mixBlendMode: "normal",
    isolation: "auto",
    transformStyle: "flat",
    // `contain: paint` (and the strict/content shorthands) is another way to
    // ask for one.
    contain: "none",
    // And the two that ask for a rendering mode outright. Both inherit, so an
    // ancestor setting either reaches the glyphs without appearing on the
    // element's own declaration at all — which is precisely the shape of thing
    // a copied Style cannot carry.
    webkitFontSmoothing: "auto",
    textRendering: "auto",
};

// What the probe and the box it answers for are allowed to disagree about.
//
// # Why this is a list of exceptions and not a list of coverage
//
// INK_PATH_PROPS is twelve properties chosen by argument, and an argument is
// the wrong instrument for the second half of the question. That list was
// assembled from what is known to composite a subtree or to change a glyph's
// rendering mode. A property added to CSS after it was written, or one already
// there that nobody thought of — `font-synthesis`, `font-variation-settings`,
// `text-size-adjust` — lands on one of these two elements and not the other,
// and a twelve-property comparison says nothing at all about it. The list can
// only be as wide as somebody remembered to make it, and a browser widens the
// space it is drawn from without asking.
//
// So the comparison stopped being a list. Both elements' ENTIRE computed style
// is read — every longhand the browser enumerates, 476 of them in the Chrome
// this was measured on, plus any custom property in effect — and THIS is the
// set that may differ, with the reason each entry is on it. A property nobody
// has heard of is now a failure by default rather than a silence by default,
// which is the direction that a browser release can only widen.
//
// # What was measured
//
// Read against the probe's own text element (see inkProbeTextPath — the white
// Box around it is not the thing drawing glyphs, and reading that instead was
// putting twenty-five extra properties in this table), the twenty-eight scanned
// boxes in this grid differ from their probes on exactly these twenty-two
// properties and on nothing else.
//
// Everything else agrees: display, flex-direction, flex-shrink, font-size,
// font-weight, line-height, height, block-size, all eight paddings and all
// eight border radii. That agreement is the statement worth having — it is what
// says the probe really is a copy of the declaration it answers for, measured
// rather than asserted, and it is a statement the twelve-property comparison
// could not make at all.
//
// # Held in both directions
//
// Nothing outside this set may differ on any box, and every property in it must
// differ on SOME box in the run (see the permission census below). The second
// direction is its own fault: a permission nothing spends is slack in the list,
// and "differs only where permitted" is satisfied by differing nowhere.
const INK_OWN_INK =
    "the probe repaints the declared ink as black at the same alpha, which is " +
    "the whole of what makes its screenshot readable as greys (gen.go's " +
    "inkProbeColor)";
const INK_OWN_FOLLOWS_INK =
    INK_OWN_INK + ", and this property takes its value from the element's own " +
    "`color` while nothing sets it, which nothing in either tree does";
const INK_OWN_BACKDROP =
    "inkProbeFor clears the run's own Background, because a probe reads its " +
    "whole rect and a fill on the text would be a second colour in it — the " +
    "count pill declares one and the label does not";
const INK_OWN_USED_SIZE =
    "the probe is a fixed-width box painting inkProbeText and the scanned box " +
    "is as wide as its own words, so this is the used inline size, or a " +
    "percentage resolved against it";
const INK_OWN_MAY_DIFFER = {
    // The ink itself, and everything that follows it.
    "color": INK_OWN_INK,
    "-webkit-text-fill-color": INK_OWN_FOLLOWS_INK,
    "-webkit-text-stroke-color": INK_OWN_FOLLOWS_INK,
    "caret-color": INK_OWN_FOLLOWS_INK,
    "text-decoration-color": INK_OWN_FOLLOWS_INK,
    "text-emphasis-color": INK_OWN_FOLLOWS_INK,
    "outline-color": INK_OWN_FOLLOWS_INK,
    "column-rule-color": INK_OWN_FOLLOWS_INK,
    "row-rule-color": INK_OWN_FOLLOWS_INK,
    "border-top-color": INK_OWN_FOLLOWS_INK,
    "border-right-color": INK_OWN_FOLLOWS_INK,
    "border-bottom-color": INK_OWN_FOLLOWS_INK,
    "border-left-color": INK_OWN_FOLLOWS_INK,
    "border-inline-start-color": INK_OWN_FOLLOWS_INK,
    "border-inline-end-color": INK_OWN_FOLLOWS_INK,
    "border-block-start-color": INK_OWN_FOLLOWS_INK,
    "border-block-end-color": INK_OWN_FOLLOWS_INK,
    // The ground the probe stands on.
    "background-color": INK_OWN_BACKDROP,
    // And the size, which is the one thing a probe deliberately does not copy:
    // inkProbeFor clears Width and Height so the probe is the declared box.
    // transform-origin and perspective-origin are percentages of the border
    // box, so they are that same size restated rather than a second fact.
    "width": INK_OWN_USED_SIZE,
    "inline-size": INK_OWN_USED_SIZE,
    "transform-origin": INK_OWN_USED_SIZE,
    "perspective-origin": INK_OWN_USED_SIZE,
};

// Which of its two jobs a path is doing.
//
// # The two questions one path was answering
//
// A path here names an element, and the checks ask two quite different things
// of one. WHERE the pixels are — a rect to scan, a point to sample a fill at,
// a window to look for digits in. And WHOSE declaration is in question — the
// computed style the probe's copy is compared against, and the chain above the
// element whose glyphs are being drawn.
//
// For a box that draws its own glyphs those are the same element and the
// distinction never surfaces. For a box with the glyphs somewhere INSIDE it
// they are not, and reading the second question at the first question's path
// answers it about the wrong element. That is not hypothetical: the
// antialiasing probes did exactly this for as long as they existed.
// `inkProbePath` names the white Box because the Box is the rect the
// screenshot scan reads, and `probeOwn` and `probeAncestry` followed it there
// — so the comparison that licenses the whole ink scan was being made against
// a box that declares a background, a width and no typography at all, and
// twenty-five properties differed for no reason but the mix-up.
//
// # What is held, and where the badge sits
//
// A path used as a DECLARATION SUBJECT has to name the element that draws the
// glyphs: no element children, and text of its own. Measured on this grid, all
// three subjects are (20 labels, 8 counts, 11 probe text nodes), every one a
// span with no element child and direct text.
//
// The count is the interesting one. It reads as a pill with digits inside it,
// and it is not: components.Badge is `core.Text` with a fill, a radius and
// paddings on it, so the pill and the digits are ONE element and reading
// `badgeOwn` at the pill is reading it at the digits. That was true by
// accident — nothing said it, and a badge that grew an icon beside its number
// would become a Box with a Text in it, the pill path would go on naming the
// Box, and `badgeOwn` would quietly start comparing a container against a
// probe built from the digits' Style. Held here, that refactor fails by name.
//
// The complement is held too, at the one place the file has a path for each
// job: the probe's sample rect must NOT be a glyph-drawing leaf. It is the
// white ground the fringe is measured against, and it has to be a box with
// exactly one child — which is what makes `inkProbeTextPath`'s `/0` a
// derivation rather than an index somebody typed.
function inkSubjectShape(read) {
    return `a <${read.tag}> with ` +
        (read.elements === 0 ? "no element children" :
            `${read.elements} element ${read.elements === 1 ? "child" : "children"}`) +
        ` and ${read.directText ? "text of its own" : "no text of its own"}`;
}

// Returns null when the element at a subject path is the one drawing the
// glyphs, which is every path this grid reads a declaration at today.
function inkSubjectFault(where, subject, read) {
    if (!read) {
        return `${where}: nothing was found at the path ${subject} are read at, so ` +
            `the declaration this scan is about was read off no element at all`;
    }
    if (read.elements === 0 && read.directText) return null;
    return `${where}: ${subject} are read at ${inkSubjectShape(read)}, which is a box ` +
        `with the glyphs somewhere inside it rather than the element drawing them.\n\n` +
        `Every declaration this scan compares — the computed style the antialiasing ` +
        `probe is held to, and the chain swept above it — is a question about the ` +
        `element that paints the run. Asked of a container, the answer is about the ` +
        `container: it inherits the page's typography, resolves none of what the run ` +
        `resolves, and the comparison passes or fails for reasons that have nothing ` +
        `to do with how these glyphs are drawn. See the note above inkSubjectShape: ` +
        `this is the ` +
        `fault the probes shipped with, and the path that names the rect to scan is ` +
        `not in general the path that names the declaration to ask about`;
}

// And that every declaration this read came back with is one the guard above
// was actually asked about.
//
// # The guard covered the paths that ask, not the shape of the mistake
//
// inkSubjectFault is asked at the three paths a declaration is read at today —
// the label, the count, and the probe's own text. That is the whole of the
// grid's declaration reads and it is a fact about the current file rather than
// a property of it: `r.band`, `r.control`, `r.wrapper`, `r.leading` and
// `r.chevron` are read as rects and none of them is read as a declaration, so
// the confusion the guard is about is simply not available at any of them
// today. What makes that true is that nobody has added a declaration read to
// one — the same sentence that was true of the badge until components.Badge
// was asked to grow.
//
// The read is now driven by one list (see `declarations` in the evaluate), so
// the three go together by construction. This is the other half: it holds what
// came BACK to the same pairing, so a `somethingOwn` written straight into the
// evaluate — around the table, the way the previous version's lines were — is
// a failure here rather than a silent fourth declaration nothing guards.
//
// # Paired by the table, not by the spelling
//
// The previous version took every key ending in "Own" or "Ancestry" and looked
// for the same stem plus "Subject". That is the convention the `declarations`
// table produces and it is not a property of it, so the guard had two holes
// with one shape: a read named for what it measures rather than for how it is
// measured — `ownStyle`, `labelBox` — is outside the pairing by spelling, and
// a MISSING `Own` beside a Subject that is there is not a mismatch at all,
// because there is no key to notice.
//
// So the read carries a manifest of what it asked (see `out.reads` in the
// evaluate) and this is held to that: every declared stem must have all three
// answers, and every answer in the read that has the SHAPE of one of the three
// must sit at a stem the manifest names. Shape rather than spelling is what
// closes the second hole — `own()` returns a computed style whatever it is
// stored under, and a hand-written read cannot avoid looking like what it is.
//
// Returns null when the read asked what it says it asked, which is every read
// this grid takes.

// What a subject read is, named once.
//
// The narrowest object inkReadShape is handed that must NOT come back "a
// computed style", and therefore the lower edge of INK_OWN_SHAPE_FLOOR. Named
// rather than written inline because the floor is now stated as sitting
// between two measured populations and this is one of them — see where the
// floor's two edges are reported together.
const INK_SUBJECT_KEYS = ["tag", "elements", "directText", "text"];

// What one value in the read looks like, or "" for anything that is not one of
// the three declaration answers.
//
// An empty array is called a chain. It is genuinely ambiguous — an ancestry
// with nothing in it is the common case here and is indistinguishable from an
// empty list of anything else — and the ambiguity is resolved towards
// reporting, because being wrong that way costs a line in a table nobody had
// to write, and being wrong the other way is the hole this exists to close.
function inkReadShape(v) {
    if (Array.isArray(v)) {
        return v.every((e) => e && typeof e === "object" &&
            "prop" in e && "neutral" in e) ? "an ancestry sweep" : "";
    }
    if (!v || typeof v !== "object") return "";
    const keys = Object.keys(v);
    if (keys.length === INK_SUBJECT_KEYS.length &&
        INK_SUBJECT_KEYS.every((k) => k in v)) {
        return "a subject read";
    }
    // A computed style is every longhand the build exposes — hundreds — and
    // every value in it is a string. Nothing else this read returns is both.
    // The floor is derived from the one enumeration this file has counted
    // rather than chosen to look sensible: see INK_OWN_SHAPE_FLOOR.
    if (keys.length > INK_OWN_SHAPE_FLOOR &&
        Object.values(v).every((x) => typeof x === "string")) {
        return "a computed style";
    }
    return "";
}

function inkDeclarationGuard(where, read) {
    if (!read) return null;
    if (!read.reads) {
        return `${where}: the read came back with no manifest of what it asked, so ` +
            `nothing says which of its keys are declaration reads.\n\n` +
            `The evaluate sets \`out.reads\` from the same \`declarations\` table it ` +
            `derives the reads from — that pairing is the whole guard, and without ` +
            `the manifest this check is back to recognising a declaration read by ` +
            `the way somebody spelled its key`;
    }
    const declared = read.reads.declarations || [];
    const subjectsOnly = read.reads.subjectsOnly || [];
    const problems = [];

    // Every declared stem, all three answers. A missing Own is as much a hole
    // as a missing Subject: the probe comparison and the ancestry sweep and
    // the guard saying the path names a glyph-drawing leaf are one question
    // asked three ways, and any two of them without the third is that question
    // half asked.
    const accounted = new Set();
    for (const as of declared) {
        for (const suffix of ["Own", "Ancestry", "Subject"]) {
            accounted.add(as + suffix);
            if (read[as + suffix] === undefined) {
                problems.push(`${as} is in the manifest and ${as}${suffix} is not in ` +
                    `the read`);
            }
        }
    }
    for (const as of subjectsOnly) {
        accounted.add(as + "Subject");
        if (read[as + "Subject"] === undefined) {
            problems.push(`${as} is in the manifest as a subject read and ` +
                `${as}Subject is not in the read`);
        }
        for (const suffix of ["Own", "Ancestry"]) {
            if (read[as + suffix] !== undefined) {
                problems.push(`${as} is in the manifest as a subject read and ` +
                    `${as}${suffix} came back beside it`);
            }
        }
    }
    // And nothing shaped like one of the three outside the manifest.
    for (const key of Object.keys(read)) {
        if (key === "reads" || accounted.has(key)) continue;
        const shape = inkReadShape(read[key]);
        if (shape) {
            problems.push(`${key} is ${shape} and the manifest does not name it`);
        }
    }
    if (problems.length === 0) return null;
    return `${where}: ${problems.join("; ")}.\n\n` +
        `A declaration read — the computed style the antialiasing probe is held to, ` +
        `or the chain swept above the glyphs — is a question about the element that ` +
        `PAINTS the run, and the subject read is the only thing that says the path ` +
        `names one. Asked of a container the answers are about the container and ` +
        `both come back perfectly plausible: it inherits the page's typography and ` +
        `resolves none of what the run resolves. See inkSubjectFault, and the ` +
        `\`declarations\` table the three reads are supposed to be derived from — ` +
        `a read written around that table is a fourth declaration with no guard ` +
        `beside it, whatever its key is called`;
}

// And the other side of the pair, for the probe's white ground.
//
// Returns null when the sample rect is a container with exactly one child and
// no text of its own — which is what gen.go's inkProbeFor builds and what
// makes inkProbeTextPath's `/0` the element inside it.
function inkProbeBoxFault(where, read) {
    if (!read) {
        return `${where}: nothing was found at the probe's own path, so the rect the ` +
            `fringe is measured in was read off no element at all`;
    }
    if (read.elements === 1 && !read.directText) return null;
    return `${where}: the probe's sample rect is ${inkSubjectShape(read)}, and ` +
        `inkProbeFor builds it as a white Box holding exactly one Text.\n\n` +
        `Those two facts are what make inkProbeTextPath — the probe path plus "/0" — ` +
        `the element that draws the probe's glyphs rather than an index somebody ` +
        `typed. A probe with text of its own would be painting glyphs into the ground ` +
        `the fringe is measured against; a probe with a second child would leave "/0" ` +
        `naming one of two, and every declaration compared below would be one of them ` +
        `chosen by position`;
}

// Which browser's enumeration the table above was written against.
//
// # A list of exceptions has a build behind it
//
// inkOwnDiff compares every computed property the browser enumerates, and
// INK_OWN_MAY_DIFFER is the list of the ones that may differ. That list was
// assembled by running the grid, reading what differed, and giving each entry a
// reason — so it is a list of exceptions to ONE build's enumeration, and
// nothing recorded which.
//
// The widening is the point of the design: a property that ships in a later
// Chrome joins the comparison without anybody editing this file, and if the
// probe and the box it answers for resolve it differently that is a real
// finding. It is also, on the first run on a newer browser, a failure that
// arrives with no context at all — a property name nobody here has heard of,
// differing for a reason nobody has thought about, in the middle of a check
// about antialiasing. The message below says which build the table was measured
// against and how many more properties this one exposes, so a reader can tell
// "the two declarations came apart" from "this browser grew a property".
//
// `props` is the count, not the list. The list would be four hundred names in a
// source file and would have to be regenerated on every Chrome; the count is
// the one number that separates the two causes, and the delta is what the
// message reports.
//
// The dpr is not recorded here, and the machine is not either. A computed style
// is in CSS pixels, so nothing in this comparison moves with the device ratio —
// the ink scan's SAMPLE POINTS do, and those are multiplied by a dpr this file
// reads back rather than assumes. What a different machine could change is the
// font the face resolves to, and that is INK_OWN_MAY_DIFFER's business only
// insofar as both sides resolve it alike, which they do by being one page.
const INK_OWN_MEASURED_ON = {
    browser: "Chrome/152.0.7977.83",
    props: 476,
};

// How many string values make a computed style, taken from the enumeration
// above.
//
// # A bare number with its argument in a comment
//
// inkReadShape's test used to be `keys.length > 50`, with "a computed style is
// every longhand the build exposes — hundreds" written beside it. The sentence
// is true and 50 is not derived from it: it is a number that sits somewhere
// between "hundreds" and the four keys of a subject read, and nothing said how
// far it was from either edge or what would move it. Meanwhile the file
// already records the one enumeration anybody here has counted — 476, on the
// build INK_OWN_MAY_DIFFER was assembled against — and already counts what the
// running build exposes.
//
// A quarter of it. The gap this has to straddle is between four keys and
// several hundred, so any number in the middle separates them and the property
// worth having is that it MOVES with the only measurement available: a file
// that re-measures INK_OWN_MEASURED_ON against a newer Chrome re-derives this
// with it. What a quarter buys over a half is room for a build that exposes
// far fewer longhands than Chrome does before the shape test stops recognising
// one — and that a run whose own enumeration falls under it says so, rather
// than the guard silently going quiet.
//
// # And a fraction is still a fraction of one population
//
// The quarter is derived and it is derived from ONE number. What makes the
// floor a measurement rather than a taste is the pair of populations it has to
// separate, and both are in hand on every run: the widest computed style this
// grid reads (inkOwnRead, counted where the pairs are compared) is what must
// stay ABOVE it, and INK_SUBJECT_KEYS — the narrowest object the shape test has
// to reject — is what must stay under it. Neither edge is asserted by the
// arithmetic: `Math.floor(props / 4)` is under four the moment `props` is, and
// a build that enumerates fewer properties than this floor is one where the
// guard has gone quiet.
//
// So both edges are checked and reported as one bracket. See where inkOwnRead
// is compared with this.
const INK_OWN_SHAPE_FLOOR = Math.floor(INK_OWN_MEASURED_ON.props / 4);

// The sentence the message adds when this browser is not that one. Returns ""
// when the enumeration matches, because then the delta explains nothing.
function inkOwnBuildNote(read, browser) {
    const same = browser === INK_OWN_MEASURED_ON.browser;
    const delta = read - INK_OWN_MEASURED_ON.props;
    if (same && delta === 0) return "";
    return `\n\nINK_OWN_MAY_DIFFER was measured against ` +
        `${INK_OWN_MEASURED_ON.browser}, which enumerated ` +
        `${INK_OWN_MEASURED_ON.props} computed properties; this is ${browser} and it ` +
        `enumerates ${read}` +
        (delta === 0 ? `, the same number` :
            `, ${Math.abs(delta)} ${delta > 0 ? "more" : "fewer"}`) +
        `. ` +
        (delta > 0
            ? `Properties this browser has and that one did not were never looked at ` +
              `when the table was written, so a name nobody here recognises is more ` +
              `likely to be one of those than a declaration coming apart — check it ` +
              `against the table's argument and add it with a reason, or find out why ` +
              `these two elements resolve it differently.`
            : `The table was written against a wider enumeration than this one, so ` +
              `every entry in it may no longer be reachable — the permission census ` +
              `below is the check that notices that.`);
}

// Which face drew the glyphs, asked of the one party that knows.
//
// # The gap the two width answers leave
//
// inkRunRectFault holds the run rect's width to the advance the same face
// gives for the same string, and calls that two answers. It is two answers to
// the arithmetic and one answer to the shaping: Chrome's canvas text
// measurement and Chrome's layout go through the same shaper, so a face that
// resolved to something other than what was asked for moves BOTH numbers
// together and they agree about the wrong face. What that pair catches is
// arithmetic done to the rect after shaping, which is the class the injected
// defect belongs to; what it cannot see is the resolution itself.
//
// That argument holds while both sides are ASKED the same thing, and the
// canvas is not: it is handed a shorthand assembled out of four computed
// longhands, which is less than an element's font request. See
// inkCanvasFaceFault, which is what holds the two requests together — without
// it, a canvas on another face makes the pair above fire with a message about
// the layout.
//
// Nothing on the page can see it either. A computed `font-family` is the
// REQUEST — a list, with the UA default at the end of it — and neither the
// canvas nor `getComputedStyle` reports which entry the browser reached. So
// this is read over the DevTools protocol instead:
// `CSS.getPlatformFontsForNode` is the compositor's own record of the faces it
// laid a node's glyphs out with, one entry per face with the number of glyphs
// that face drew. It is a third party to a question the other two agree on by
// construction.
//
// # What is held
//
// One face per run. A run drawn by two is a run with fallback in it, and every
// number the ink scan takes is then an average over faces: the ascent the band
// rows are fractions of came from `measureText` under one font shorthand, and
// the advance the rect is held to is a sum the layout took over several.
//
// One glyph per character. Measured on this grid, every run's glyph count is
// exactly its string length — twelve for "January 2026", twenty-six for the
// long title, one for a count's digit, six for the probe's own text. That is
// the third answer to "was the string on the page the string that was
// measured": `measureText(e.textContent)` reads the DOM's characters and this
// counts the compositor's glyphs. A face that substituted a ligature for two
// characters would break it, which for ASCII digits and Latin words in a UA
// default is a finding rather than a false alarm — and if a fixture ever wants
// a string where it is not, the fixture is the thing to change.
//
// The same family as the probe. The probe's whole claim is that its grey box
// is this box's antialiasing; a probe drawn by a different face is measuring a
// different face's edges, and neither the ancestry sweep nor the computed-style
// comparison can see it — `font-family` agrees on both sides precisely because
// it is the request they share.
function inkFaceList(faces) {
    return faces.map((f) => `${f.family} (${f.glyphs} glyph` +
        `${f.glyphs === 1 ? "" : "s"})`).join(", ");
}

// The fold the question below is asked through.
//
// # Lowercasing was the whole of the normalisation
//
// inkLigatureSeeds' pairs are ASCII and this grid's fixtures are Latin, so
// `text.toLowerCase().includes(pair)` was right about every string that has
// ever reached this note. It is right BECAUSE of gen.go's refusal rather than
// independently of it: inkGlyphPerCharacter drops any fixture outside printable
// ASCII, so a composed form or a zero-width joiner never arrives.
//
// And this note is the message that fires when that refusal did not do its job.
// It is inkFaceFault's third arm's companion and is reached only on a run where
// a string the check meant to keep out is in the grid — so deciding what such a
// string contains by an ASCII-shaped test is deciding it in the one world where
// the ASCII assumption is known to have failed. A label carrying U+FB01
// contains "fi" in every sense this note is about and in none that
// `includes("fi")` can see, and the note would then say the seed list is short
// of this face while the fixture is holding the pair in a single code point —
// a confident sentence about the wrong one of two suspects.
//
// # Three steps, in this order
//
//	NFKD          the COMPATIBILITY decomposition, which is the one that turns
//	              a precomposed ligature into its letters: U+FB01 becomes "fi".
//	              Canonical NFD leaves it whole, because an f-ligature is a
//	              compatibility equivalence and not a canonical one — so NFD,
//	              the form that would be right for accents, is the wrong form
//	              for exactly the characters this note is about.
//	format chars  the soft hyphen, the zero-width and bidi controls, the word
//	              joiner and the BOM. None carries an advance or a glyph, so a
//	              pair split by one is a pair the shaper still joins and a pair
//	              this note must still find.
//	long s        U+017F to "s". U+FB05's compatibility decomposition runs
//	              through it — UnicodeData gives <compat> 017F 0074 — and
//	              `toLowerCase` will not finish the job, because mapping the
//	              long s to "s" is a case FOLDING and that is a case mapping.
//	              This build's normaliser happens to answer "st" for U+FB05
//	              outright; one that answered "ſt" would leave the pair one
//	              character short of the seed. Taken explicitly so the reading
//	              does not depend on which.
//	lowercase     last, because the steps above produce letters that need it:
//	              NFKD of a ligature yields base letters in whatever case the
//	              composed form carried.
//
// gen.go folds too now, and through the same three answers: inkGlyphFold
// expands the same presentation forms, drops the same format characters and
// takes the same long-s step, and asks the pair question before the
// printable-ASCII one. What it does not have is a normaliser — that module
// carries no Unicode tables — so its expansion is written out for the seven
// characters the seed list is about rather than derived. The two are the same
// answer about a seed written some other way and this one is wider everywhere
// else, which is the direction that costs a note a clause rather than a
// fixture a refusal.
//
// # And it is applied to both sides of the question
//
// The pairs this note looks for are inkLigatureSeeds', and they are ASCII by
// construction today — so folding the string alone finds them. That is a
// property of the seed list rather than of the fold, and the list is written
// in gen.go while the fold is here: a pair added there in a composed form, or
// with a joiner in it, would be looked for UNFOLDED inside a folded string and
// could never be found. The note would then say this face joins nothing the
// list names while holding the pair it joined, which is the same wrong suspect
// this fold exists to stop naming, reached from the other end.
//
// So the seeds go through the same fold, and the two sides are joined in the
// fold rather than in an assumption about one of them. See inkLigatureNote,
// where a seed the fold changed also earns a clause: a reader told that "fi"
// is not in a string is owed the form it was actually looked for in.
const INK_FOLD_IGNORABLE =
    /[\u00AD\u180E\u200B-\u200F\u202A-\u202E\u2060-\u2064\u206A-\u206F\uFEFF]/gu;

function inkFold(text) {
    return typeof text === "string"
        ? text.normalize("NFKD").replace(INK_FOLD_IGNORABLE, "")
            .replace(/\u017F/gu, "s").toLowerCase()
        : "";
}

// What the measured ligature row adds to a glyph-count mismatch.
//
// The refusal in gen.go is what keeps this arm quiet, and a run that reaches
// it anyway is a face joining a pair inkLigatureSeeds does not name — or that
// list working and something else entirely doing the joining. Until the row
// was mounted this message could not tell those apart and did not try. Now it
// can say which of the nine this face actually joins, and whether the string
// in front of the reader holds one of them.
//
// Empty when there is no measurement, rather than hedged: an absent census is
// reported by inkLigatureCensus in its own message and does not belong in the
// middle of somebody else's.
function inkLigatureNote(text, ligated) {
    if (!ligated || ligated.length === 0) return ``;
    const folded = inkFold(text);
    // The seeds through the same fold as the string. See the note above
    // inkFold: folding one side is enough only while the other side is ASCII,
    // which is gen.go's business and not this file's.
    const seeds = ligated.map((pair) => ({ pair, as: inkFold(pair) }));
    // Whether the answer below is one a plain lowercase would have given. A
    // fold that changed something is a string carrying a composed form or a
    // format character — a fixture gen.go's refusal was supposed to have
    // dropped — and it earns a clause, because the pair the reader is being
    // told about may not be two characters in the string in front of them.
    const flat = typeof text === "string" ? text.toLowerCase() : "";
    const how = folded === flat ? `` :
        ` (asked of ${JSON.stringify(folded)}, which is this string under NFKD with ` +
        `the zero-width and bidi controls dropped — a pair written as one code point ` +
        `or split by a joiner is still the pair this face joins)`;
    // And the same clause for the other side. A seed the fold changed is a
    // pair inkLigatureSeeds spells in a form the string could not hold
    // literally, and the reader is owed the form it was actually looked for
    // in — otherwise the sentence names a pair and the search names another.
    const bent = seeds.filter((s) => s.as !== s.pair);
    const asWritten = bent.length === 0 ? `` :
        ` (${bent.map((s) => `${JSON.stringify(s.pair)} looked for as ` +
            `${JSON.stringify(s.as)}`).join(", ")}, since the seeds go through the ` +
        `same fold as the string)`;
    const here = seeds.filter((s) => folded.includes(s.as));
    if (here.length > 0) {
        return `\n\nThis face was measured to draw ` +
            `${here.map((s) => s.pair).join(", ")} as one glyph, ` +
            `and this string contains ${here.length === 1 ? "it" : "them"}${how}` +
            `${asWritten} — so ` +
            `the refusal that was supposed to keep this string out of the grid did not ` +
            `fire. inkGlyphPerCharacter and this face disagree about the same pair, ` +
            `which is a fixture that got through rather than a substitution`;
    }
    return `\n\nThe pairs this face was measured to join are ${ligated.join(", ")}, ` +
        `and none of them is in this string${how}${asWritten}. So whatever it joined ` +
        `is not one ` +
        `inkLigatureSeeds names, and the list is short of this face rather than ` +
        `the fixture being short of the list`;
}

// Returns null when one face drew the whole run, drew a glyph per character,
// and is the face the probe was drawn with.
function inkFaceFault(where, subject, what, faces, text, probeFaces, ligated) {
    if (!faces || faces.length === 0) {
        return `${where}: the browser reports no platform font for ${subject}, so ` +
            `which face drew those glyphs is unknown. The run's width is checked ` +
            `against an advance from the same shaper that laid it out, and that pair ` +
            `agrees about whatever face was resolved — see the note above ` +
            `inkFaceList. This read is the only thing here that looks at the ` +
            `resolution rather than at the request`;
    }
    if (faces.length > 1) {
        return `${where}: ${subject} are drawn by ${faces.length} faces — ` +
            `${inkFaceList(faces)} — so part of this run fell back.\n\n` +
            `Every measurement the ink scan takes over this box is a single face's: ` +
            `the ascent the three sampled rows are fractions of comes from one font ` +
            `shorthand handed to a canvas, and the advance the run rect is held to is ` +
            `that same face's for the whole string. With two faces in the run those ` +
            `are averages, and the antialiasing probe — one declaration, one face — ` +
            `answers for neither`;
    }
    const glyphs = faces[0].glyphs;
    const chars = text === null || text === undefined ? null : [...text].length;
    if (chars !== null && glyphs !== chars) {
        return `${where}: ${subject} are ${chars} characters ("${text}") and ` +
            `${faces[0].family} drew ${glyphs} glyphs for them.\n\n` +
            `The run rect's width is held to \`measureText(textContent)\`, which is an ` +
            `advance for the DOM's CHARACTERS; this is a count of the compositor's ` +
            `GLYPHS. They match on every run in this grid, and a mismatch means the ` +
            `face is substituting — a ligature, a composed form — so the string that ` +
            `was measured and the string that was drawn are not the same sequence, ` +
            `and both width answers are about the former.\n\n` +
            `One glyph per character is a property of the strings this grid uses and ` +
            `not of any face, and gen.go's inkGlyphPerCharacter is where that is ` +
            `decided: it refuses a scanned run carrying an f-ligature pair or anything ` +
            `outside printable ASCII, so a fixture whose words a face is entitled to ` +
            `draw as one glyph fails at the fixture rather than arriving here as a ` +
            `font substitution. What is left for this message is the substitution ` +
            `itself — a face doing it to a string that check passed` +
            inkLigatureNote(text, ligated);
    }
    if (probeFaces && probeFaces.length === 1 &&
        probeFaces[0].family !== faces[0].family) {
        return `${where}: ${subject} are drawn by ${faces[0].family} and the ` +
            `antialiasing probe for ${what} is drawn by ${probeFaces[0].family}.\n\n` +
            `The probe's grey box is offered as this box's antialiasing, and an edge ` +
            `is a property of the face that drew it. Neither of the other two guards ` +
            `can see this: the ancestry sweep is about what sits above the elements, ` +
            `and the computed-style comparison finds \`font-family\` identical on both ` +
            `sides — that list is the request the two share, and this is the answer ` +
            `they got`;
    }
    return null;
}

// The shorthand the canvas is handed, spelled once.
//
// Carried as source rather than as a function, for the same reason FNV1A_JS is:
// it is evaluated in the page, in two different expressions — the rects' own
// evaluate, where the ink band and the run advance are measured off a canvas,
// and the after-faces evaluate, where that canvas is held to the compositor's
// face. Two spellings would let the join drift off the thing it joins, and the
// drift would be silent: the second read would be holding a shorthand nothing
// ever measured with.
//
// `head` puts one family at the front of the element's own list and is left out
// by the measuring side. `instead` replaces that list outright: a substitution
// is the wrong shape for the join itself — see inkCanvasFaceFault, where
// prepending is the whole of what keeps an unreachable platform family from
// reporting every run — and the right shape for the canary that asks whether
// such a name reaches a face at all, because there the fallback IS the answer.
const INK_FONT_SHORTHAND_JS = `((cs, head, instead) =>
    cs.fontStyle + " " + cs.fontWeight + " " + cs.fontSize + " " +
    (instead || (head ? head + ", " : "") + cs.fontFamily))`;

// The part of that shorthand that selects a face before the family does.
//
// Style, weight and size — everything a font request carries except the list.
// Two readers need it: the canary probes, which mount one span per distinct
// request so the generic is resolved at a size some canary is actually asked
// at, and the canvas measurement, which reports the request each run was
// measured under so the two can be paired.
//
// One expression rather than two, because the pairing is a JOIN and a join on
// two spellings of a key is a join that silently matches nothing: a probe keyed
// "normal 400 12px" and a run keyed "normal 400 12.0px" would leave every
// comparison below unasked, and an unasked comparison here reports as agreement.
const INK_CANARY_REQ_JS = `((cs) =>
    cs.fontStyle + " " + cs.fontWeight + " " + cs.fontSize)`;

// The family the canary falls back to.
//
// A generic, because a generic always resolves — that is what makes "the same
// advance as this alone" a statement about the name in front of it rather than
// about a second name that also missed. `monospace` rather than `serif` or
// `sans-serif` because the separation wanted is metric: the faces this grid
// draws with are proportional, and a fixed-advance face is the one whose
// widths for these strings are nothing like theirs, so the two readings come
// apart by pixels rather than by a rounding.
//
// Being wrong the only way this can be wrong — a platform face whose advance
// for the string happens to equal this one's — costs a run the confirmation
// and reports the join as silent when it bit. That is the understating
// direction: it says a reading is missing, never that a face was named when it
// was not.
const INK_CANARY_FALLBACK = "monospace";

// And how far the canary's two readings have to come apart before that
// separation is a face's rather than a rounding's.
//
// inkCanvasNameReached compares them against LAYOUT_UNIT, which is the bound
// every other comparison in this file is asked under and is the wrong bound to
// leave unbracketed HERE. Everywhere else a LayoutUnit is the width of the
// answer: two numbers about one face, and the tolerance keeps the check off a
// double's last bits. This one is a claim about the note above — that a
// fixed-advance face and this grid's proportional ones are nothing like each
// other over these strings — and a run that clears a 64th of a pixel has not
// shown that at all.
//
// The direction it fails in is the silent one. A face sitting within a
// LayoutUnit of monospace for a badge's single digit makes canary and base one
// number, the join reads as a name that reached nothing, and the message sends
// a reader looking for an unreachable family that is perfectly reachable —
// while the run PASSES, because a silent join is reported and a face that was
// never held is not.
//
// So the margin is stated and the population is measured against it, which is
// the shape asked.canvasWidest already gives the join's own bound. A quarter
// of a CSS pixel — sixteen LayoutUnits, so a reading that clears it is not a
// rounding under any device ratio this grid is captured at, and it sits about
// five times under what the population actually achieves: the narrowest of the
// grid's own canaries separates its two advances by 1.22px, on a single-digit
// badge, which is the shortest string here and the least room a canary gets.
//
// Set from that measurement rather than from a fraction of it, for the reason
// INK_ROW_ROUNDING is: a margin derived from what a run scores moves whenever
// the run does, and a margin stated as a number is a bracket the run can be
// found to have left.
const INK_CANARY_MARGIN = 0.25;

// And the string the fallback's own face is read off.
//
// Three characters covering the three shapes this grid is measured over — a
// lowercase letter with a descender, a capital, a digit — because a family
// list can resolve per character and the question is which face this generic
// reaches for the kind of text the canary is asked about. No pair
// inkLigatureSeeds names is in it, so nothing here can be joined into one
// glyph and reported as a face resolving differently.
const INK_CANARY_PROBE_TEXT = "Ag0";

// And whether the canvas resolved that shorthand to the face the compositor
// drew the run with.
//
// # The fourth reader
//
// Three readers of this grid are held to one another now. The rects, the
// computed styles and the run metrics come back from one evaluate and are of
// one layout by construction; the capture is bracketed by
// LAYOUT_FINGERPRINT_JS; the platform-font reads are bracketed by that and by
// TREE_FINGERPRINT_JS together — see facesHeld.
//
// The fourth is a canvas. `band()` opens one per scanned box and takes three
// numbers off it: the ascent that puts the baseline under the inline text box's
// top, the ink ascent the three sampled rows are fractions of, and
// `measureText(textContent)`, which is one of inkRunRectFault's two width
// answers. Those numbers ride back with the rects, so no relayout can get
// between them and the boxes they are about — that window is closed. What
// nothing held is WHICH FACE they are about.
//
// The canvas is handed a shorthand this file assembles out of four computed
// longhands, and an element's font request is not four longhands. `font-stretch`
// selects a width face. `font-size-adjust` changes the size the face is asked
// for. `font-variant-caps` can reach a small-caps face or have one synthesised.
// `font-variation-settings` and `font-optical-sizing` pick an instance of a
// variable font. None of them travels through the shorthand, and any of them
// leaves the canvas measuring a face that is not the one on the screen.
//
// # Why the two width answers cannot see it
//
// inkRunRectFault compares the run rect's width with the canvas's advance and
// calls that two answers. The note above inkFaceList says why that pair is
// blind to the RESOLUTION: canvas measurement and layout go through one shaper,
// so a family list that resolved to something unexpected moves both numbers
// together and they agree about the wrong face.
//
// That argument holds while both sides are given the same request, which is
// exactly what an assembled shorthand is not. When the requests differ the pair
// does fire — and it fires with the wrong story. Its message is about arithmetic
// done to a rect after shaping, and it would send a reader to the layout for a
// face the canvas picked.
//
// # The join
//
// The same measurement, with CSS.getPlatformFontsForNode's own family at the
// head of the element's own list. If the canvas had already resolved that list
// to that face, naming it first changes nothing and the two advances are the
// same double. A difference is the canvas on one face while the compositor drew
// another.
//
// Bounded by a LayoutUnit rather than asked as an equality. Two measurements of
// one face, on one canvas, in one evaluate, of one string are the same number;
// the bound is there so the check is about a face and not about a double's last
// bits, and a whole run drawn by two different faces is nowhere near that close.
//
// Prepended rather than substituted, and that is the whole of what makes this a
// one-sided test. A platform font name is a FACE's name and need not be a family
// CSS can reach — ".AppleSystemUIFont" is not — so a substitution would be
// measuring the canvas default and reporting every run in the grid. At the head
// of the list, a name that resolves to nothing is skipped and the list resolves
// as it did: equal advances, and the check goes quiet rather than wrong. What is
// left to report is the one refusal the browser states out loud — the assignment
// itself not taking, which is the `unspellable` arm.
//
// Taken in the after-faces evaluate, because the family it names comes back over
// the protocol and there is nowhere earlier to know it. It costs no round trip
// — the two fingerprints were being taken there anyway — and it is covered by
// both of them, which is the same suppression every other face consultation
// gets.
//
// # What the one-sidedness costs, and the canary that prices it
//
// Prepending buys the check its safety and spends its evidence. A head family
// that resolves to no face is SKIPPED, the list resolves exactly as it did, and
// the two advances are equal — which is the same number this check returns when
// the canvas and the compositor genuinely agree. "The face was named and
// matched" and "the name reached nothing and the join was a no-op" arrive as
// one reading, and a count of them recited in the tail says the first while
// meaning either.
//
// Nothing already in hand separates them. The element's own list is the
// fallback in both cases, so asIs is the same number either way; a substitution
// would tell them apart and is the thing this deliberately does not do.
//
// So a second pair of measurements, on the same canvas, in the same evaluate,
// of the same string: the advance under INK_CANARY_FALLBACK alone, and the
// advance under this family in front of INK_CANARY_FALLBACK. The generic always
// resolves, so the second reading falls back to the first EXACTLY when the name
// in front of it reaches nothing. A difference is the name reaching a face; an
// equality is the join having been asked of a name CSS cannot follow.
//
// That reading is what the arm below reports and what asked.canvasNamed counts,
// and the two numbers the tail recites are now "asked" and "bit".
//
// # The missing-box arm, and why it is unreachable
//
// `!m.asked` without `unspellable` is the page's querySelector finding no
// element at a path the rects were read at. It cannot be reached through a tree
// that moved, and the reason is that TREE_FINGERPRINT_JS hashes EXACTLY this
// population — every `[data-node-path]` in the document, the path itself among
// the fields fed — and the second fingerprint is taken in this same evaluate.
// A path that stopped resolving in between changes that hash, treeMovedFault
// reports it, and facesHeld drops every face consultation before this one is
// composed.
//
// What is left is one evaluate's `querySelectorAll("[data-node-path]")` and
// `querySelector('[data-node-path="..."]')` disagreeing about the same
// attribute, which is not a page fault at all. The arm is kept and says so: it
// is the reader that would fire if the fingerprint's population ever stopped
// being the population these lookups draw from, and its silence on every run is
// the evidence that it has not.
// Whether the family the join names reaches a face on this canvas at all.
//
// See inkCanvasFaceFault's canary. Split out because two readers need the same
// answer — the arm that reports a silent join, and the counter the tail recites
// — and a second spelling of the comparison would let them disagree about which
// runs were confirmed.
//
// The bound is LAYOUT_UNIT for the same reason it is everywhere else here: two
// advances off one canvas are being compared, and the question is whether they
// are the same face's, not whether they are the same double.
function inkCanvasNameReached(m) {
    return typeof m.canary === "number" && typeof m.base === "number" &&
        Math.abs(m.canary - m.base) >= LAYOUT_UNIT;
}

// How far a joined run's own face is from the generic the canary falls back to.
//
// `named` is the advance this canvas gives the box with the compositor's family
// at the head of its own list; `base` is the advance it gives the SAME string,
// at the same size, under INK_CANARY_FALLBACK alone. Both are measureText on
// one canvas in one evaluate, so their difference is two faces' widths for one
// string and nothing else — which is the whole of what the canary below rests
// on.
//
// Split out for the reason inkCanvasNameReached is: the arm that reports the
// premise failing and the number the tail recites are the same comparison, and
// a second spelling of it would let them disagree.
//
// Null when either number is missing. Those are runs inkCanvasFaceFault has
// already reported on their own line, and a gap guessed at from one advance is
// not a reading.
function inkCanvasGenericGap(m) {
    return m && typeof m.named === "number" && typeof m.base === "number"
        ? Math.abs(m.named - m.base) : null;
}

// What this browser calls the face behind that generic, for a sentence to name.
//
// One entry per distinct font request the joined runs make — see where the
// probes are mounted. A family list is allowed to answer per size (an optical
// cut, a display face at the large end), so a single read at whatever the page
// default happens to be would be naming a face no canary is asked at.
//
// Collapsed to one clause when every request came back with the same answer,
// which is what a stack without such a cut does and what makes the long form
// worth printing when it happens.
//
// Null when nothing was read. That costs a sentence its noun and costs the
// premise nothing — see inkCanvasFallbackFault, which decides on advances.
//
// # And the per-request reads are a decision's input, not only a noun's
//
// One read per distinct request is a DOM.querySelector and a
// CSS.getPlatformFontsForNode each, inside the window two fingerprints are
// holding open, for a sentence the premise no longer consults. What makes the
// per-request answers worth their round trips is that something now reads them
// one at a time: inkCanaryAgreement joins each probe to the runs measured
// at ITS request and holds the name against the advances. A single read at the
// page default could not be joined to anything, and a difference between the
// six would have been a curiosity in a string.
function inkCanaryFaceList(probes) {
    if (!probes || probes.length === 0) return null;
    const read = probes.filter((p) => p.faces && p.faces.length > 0);
    if (read.length === 0) return null;
    const answers = new Set(read.map((p) => inkFaceList(p.faces)));
    if (answers.size === 1) return [...answers][0];
    return read.map((p) => `${inkFaceList(p.faces)} at ${p.req}`).join("; ");
}

// How many requests the generic was asked at, how many came back, and how many
// different answers they were.
//
// The spread is the reading that says what the per-request probes bought. One
// answer across every request is a stack with no optical cut at these sizes —
// the single probe this replaced would have said the same thing, and the six
// reads are what establishes that rather than assuming it. More than one is the
// case the single probe got wrong: a family list answering per size, naming a
// face at 19px that no canary asked at 12px is put to.
//
// `read` apart from `mounted` because a probe that came back with no faces is a
// request whose answer is missing while the others are present, and
// inkCanaryFaceList's sentence would quietly be about the rest.
function inkCanaryFaceSpread(probes) {
    if (!probes || probes.length === 0) return { mounted: 0, read: 0, answers: 0 };
    const read = probes.filter((p) => p.faces && p.faces.length > 0);
    return {
        mounted: probes.length, read: read.length,
        answers: new Set(read.map((p) => inkFaceList(p.faces))).size,
    };
}

// What the six probe reads bought, and what they cannot bound.
//
// # A round trip each, and five of them agreeing
//
// One probe per distinct request is a DOM.querySelector and a
// CSS.getPlatformFontsForNode apiece, inside the window two fingerprints are
// holding open. inkCanaryFaceSpread says how many answers came back and how
// many of them differed; on this grid that is six reads and one answer, which
// is the reading asking per request was written to get — and is also the
// evidence that five of those round trips bought a sentence rather than a fact.
//
// The honest next move is a bound, and a bound needs the CONDITION under which
// a second read could differ from the first. That condition is not a guess: the
// request key is style, weight and size (see INK_CANARY_REQ_JS), a family list
// answers per REQUEST, and a read can only differ from another where the two
// requests differ. So the axes are what to measure.
//
//\tvaried   an axis these requests take more than one value on. An agreement
//\t         across it is a real reading: the stack has no cut at those values
//\t         and one read would have carried the finding.
//\tfixed    an axis every request agrees on. Nothing here says anything about
//\t         it — six reads at one weight are six reads at one weight — and
//\t         "no optical cut" is a claim about the varied axes only.
//
// That is the whole of what separates a redundant read from a necessary one,
// and it turns "read one, and read a second only where the answers could
// differ" from an instinct into a rule with this run's own numbers under it: on
// a grid whose requests differ in size alone, the bound is one read per distinct
// SIZE, and the style and weight columns are blind however many probes are
// mounted.
//
// # Parsed from the end
//
// The key is `fontStyle + " " + fontWeight + " " + fontSize`, and fontStyle is
// the one of the three that can be two words ("oblique 10deg"). Size is the
// last token and weight the one before it, so taking them off the end leaves
// whatever remains as the style — which is right for both spellings, where
// splitting on the first space is right for only one.
//
// Null when nothing was read, for the same reason inkCanaryFaceList is: a
// sentence with no population under it should have no noun rather than a zero.
function inkCanaryReqAxes(probes) {
    const rows = (probes || []).filter((p) => p && p.req);
    if (rows.length === 0) return null;
    const axes = [
        { name: "style", values: new Map() },
        { name: "weight", values: new Map() },
        { name: "size", values: new Map() },
    ];
    for (const p of rows) {
        const parts = p.req.split(" ");
        const size = parts.pop();
        const weight = parts.pop();
        const at = { style: parts.join(" "), weight, size };
        // The answer this request came back with, so an axis can say not only
        // that it was varied but whether varying it changed anything. A probe
        // that came back empty still counts as a value of the axis — the
        // request was asked at it — and contributes no answer.
        const answer = p.faces && p.faces.length > 0 ? inkFaceList(p.faces) : null;
        for (const axis of axes) {
            const value = at[axis.name];
            if (!axis.values.has(value)) axis.values.set(value, new Set());
            if (answer !== null) axis.values.get(value).add(answer);
        }
    }
    const varied = axes.filter((a) => a.values.size > 1);
    const fixed = axes.filter((a) => a.values.size === 1);
    // What a read at one value of each varied axis would come to, which is the
    // bound: the product of the varied axes' value counts, against the number
    // of probes actually mounted. On a grid that varies one axis those are the
    // same number and nothing is being spent twice; the gap between them is
    // what the deduplication in the mount already saves.
    const grid = varied.reduce((n, a) => n * a.values.size, 1);
    return {
        varied: varied.map((a) => ({ name: a.name, values: [...a.values.keys()] })),
        fixed: fixed.map((a) => ({ name: a.name, value: [...a.values.keys()][0] })),
        reads: rows.length, grid,
    };
}

// The axis census as a clause, and the whole of what the reads bound.
//
// Two halves, because the reading has two: what varying an axis showed, and
// what the axes nobody varied leave unsaid. The second half is the one a tail
// without it gets wrong — "no optical cut at these sizes" is true and is read
// as "one probe would have done", and neither of those is a statement about
// weight or style when every request carried the same ones.
//
// The bound comes out of the same numbers: one read per point of the varied
// grid is what could have produced this finding, and the probes are mounted per
// distinct request, so on a grid varying one axis those two are equal and the
// six reads are six values of it rather than six asks at one.
function inkCanaryAxisPhrase(axes) {
    if (!axes) return "no request this run could read";
    const varied = axes.varied.map(
        (a) => `${a.name} (${a.values.join(", ")})`).join(" and ");
    const fixed = axes.fixed.map((a) => `${a.name} ${a.value}`).join(" and ");
    if (axes.varied.length === 0) {
        return `no axis at all — ` +
            (axes.reads === 1 ? `the single read is at ` :
                `every one of the ${axes.reads} reads is at `) +
            `${fixed}, so the answers agreeing says nothing beyond that one request`;
    }
    return `${varied}, which is ${axes.grid} point${axes.grid === 1 ? "" : "s"} of ` +
        `the request grid and the bound on what reading more of them could show` +
        (axes.fixed.length === 0 ? `` :
            ` — and at one ${fixed} throughout, so nothing here is a reading about ` +
            `${axes.fixed.map((a) => a.name).join(" or ")}`);
}

// And what the name-against-advance comparison declined to ask.
//
// Said only when there was something, because a clause that reports zeroes on
// every healthy run is a clause a reader stops seeing — and the state this is
// for is the comparison going quiet, which is exactly when the numbers stop
// being zero. See inkCanaryAgreement: a probe that named two faces is that
// reading declining on a page state, and without this the runs it cost arrive
// in the recital as a population that was simply smaller.
function inkCanaryPassedPhrase(asked) {
    const parts = [];
    if (asked.canvasGenericMulti > 0) {
        parts.push(`${asked.canvasGenericMulti} of the probes naming more than one ` +
            `face, which is a generic reaching two for one string and leaves no ` +
            `single name to hold an advance against, costing the join ` +
            `${asked.canvasGenericUnjoined} run` +
            `${asked.canvasGenericUnjoined === 1 ? "" : "s"}`);
    }
    if (asked.canvasGenericUnread > 0) {
        parts.push(`${asked.canvasGenericUnread} of them coming back with no face ` +
            `at all`);
    }
    if (parts.length === 0) return ``;
    return `, with ${parts.join(" and ")} — passed over rather than counted as
    agreement`;
}

// And what the per-request reads came to, as the tail says it.
//
// # A two-way branch on a three-way question
//
// This clause was spelled inline in the recital, and it chose between "one
// face at every one of them" and "N different faces across them" on
// `canvasGenericAnswers === 1`. Both branches are sentences about a stack, and
// there is a third state neither of them is about: a run where every probe
// came back with no faces answers zero, takes the second branch, and recites
// "0 different faces across them, which is a family list answering per request
// and the reason one probe is not enough" over a run that read nothing at all.
//
// That is not a rare shape of bug — it is the same shape inkCanaryPassedPhrase
// exists for, one clause down. A reading going quiet arrives in the recital as
// a number, and a number in a sentence written for the other case is a
// confident claim about the wrong thing. Nothing on this machine produces it
// (every probe here answers), which is why it stood: the browser pass runs the
// live population and the live population has never been empty.
//
// So the clause is a function, the third branch is written, and the whole of
// it is asked in inkcanary_test.go over populations this browser does not
// give. Split out for the reason inkCanvasGenericGap is: the sentence and the
// reading it is about are now one spelling.
function inkCanarySpreadPhrase(asked) {
    if (asked.canvasGenericRead === 0) {
        return `no face at any of them, so nothing below is a reading about this
    stack: ${asked.canvasGenericReqs} probe${asked.canvasGenericReqs === 1 ? "" : "s"}
    were mounted and none answered, and the axes an answer would have been read across
    are ${inkCanaryAxisPhrase(asked.canvasGenericAxes)}`;
    }
    if (asked.canvasGenericAnswers === 1) {
        return `one face at every one of them, which is this stack having no optical cut
    across ${inkCanaryAxisPhrase(asked.canvasGenericAxes)}`;
    }
    return `${asked.canvasGenericAnswers} different faces across them, which is a family
    list answering per request and the reason one probe is not enough, over
    ${inkCanaryAxisPhrase(asked.canvasGenericAxes)}`;
}

// The probe's answer and the advances' answer to one question, held against
// each other.
//
// # Two readings nothing compared
//
// The mounted probe says WHICH face the generic reached at a request: a span
// set in INK_CANARY_FALLBACK at that request, and CSS.getPlatformFontsForNode
// on it. inkCanvasGenericGap says HOW FAR that face is from the run's, over the
// run's own string, off one canvas. They are the same question — is the generic
// the face this run is drawn by — asked of a third party and asked of a
// measurement, and nothing put them side by side.
//
// That is the shape inkFaceList's own note is about, and it has a direction
// each:
//
//	one name, two advances    the probe says the generic resolved to the very
//	                         family this run is drawn by, and the same string
//	                         measures differently under the two requests. Held
//	                         here, because nothing else looks at it.
//	two names, one advance    the generic resolved to some other face and that
//	                         face measures like this run's. That is the canary
//	                         resting on nothing, and inkCanvasFallbackFault
//	                         reports it on the advances — which is the reading
//	                         that decides, since one face under two spellings
//	                         passes any name comparison.
//
// The two are complements and each is held by the reader that decides it. Said
// once between them: two messages about one page state would send a reader
// looking for two faults.
//
// # What a disagreement actually means
//
// Same face, same request, same canvas, same string: one advance. So a name
// that says the two requests reached one face while the advances say they
// reached two is one of the readings being about something else — the platform
// reporting a face that did not draw, the family list resolving past its head
// for this string, or the probe's box not being at the request it is filed
// under. Every one of those makes a sentence elsewhere confidently wrong, and
// none of them is visible from either reading alone.
//
// # And what it was NOT asked of
//
// A probe whose `faces` has two entries is passed over: the generic reached
// more than one face for "Ag0" and there is no single name to hold against an
// advance. That is true, and it is also the population where a
// name-versus-metric disagreement is most likely — a family list resolving past
// its head is exactly the state both readings are trying to see — and it was
// dropped without a count. `asked` said how many runs were joined and nothing
// said how many were passed over, so the comparison going quiet on some machine
// would arrive in the tail as a smaller number with no reason attached, which
// reads as this grid having had fewer runs rather than as this reading having
// stopped.
//
// So the skips are counted and named apart:
//
//	multi     probes that named more than one face — the comparison declining
//	          on a page state, which is the interesting one
//	unread    probes that came back with no faces at all — a request whose
//	          answer is missing rather than plural
//	unjoined  runs whose own request had a probe in the first bucket, which is
//	          what those skips actually cost this reading in rows
//
// Returns the population as well as the fault, because a comparison asked of no
// runs is not agreement and the tail has to be able to say which it had.
function inkCanaryAgreement(probes, rows) {
    // The probes that named exactly one face, by request. Two faces on a probe
    // is a generic that reached more than one for "Ag0" — there is no single
    // name to compare and inkFaceList's own arm is the reader for it.
    const named = new Map();
    // And the requests that were passed over for that reason, kept apart from
    // the ones that were never read at all: a row skipped because its probe
    // answered with two faces is this comparison declining, and a row skipped
    // because nothing came back is a read that failed.
    const plural = new Set();
    let multi = 0, unread = 0;
    for (const p of probes || []) {
        if (!p || !p.req) continue;
        if (!p.faces || p.faces.length === 0) { unread++; continue; }
        if (p.faces.length === 1) { named.set(p.req, p.faces[0].family); continue; }
        multi++;
        plural.add(p.req);
    }
    let asked = 0, unjoined = 0;
    const split = [];
    for (const m of rows) {
        if (!m || !m.asked || !m.family || !m.req) continue;
        if (plural.has(m.req)) { unjoined++; continue; }
        if (!named.has(m.req)) continue;
        const gap = inkCanvasGenericGap(m);
        if (gap === null) continue;
        asked++;
        // The other direction is inkCanvasFallbackFault's, and it is the one
        // that decides the premise. This arm is the half nothing was holding.
        if (named.get(m.req) !== m.family || gap < LAYOUT_UNIT) continue;
        split.push({ m, gap, face: named.get(m.req) });
    }
    if (split.length === 0) {
        return { asked, agreed: asked, multi, unread, unjoined, fault: null };
    }
    const worst = split.reduce((w, s) => (s.gap > w.gap ? s : w), split[0]);
    return {
        asked, agreed: asked - split.length, multi, unread, unjoined,
        fault: `the probe and the advances disagree about what ` +
            `${INK_CANARY_FALLBACK} reaches at ${worst.m.req}: the probe says ` +
            `${worst.face}, which is the family this run is drawn by, and the same ` +
            `run's own string ("${worst.m.text}") measures ` +
            `${worst.gap.toFixed(6)}px differently under the two requests — against ` +
            `a LayoutUnit of ${LAYOUT_UNIT}` +
            (split.length === 1 ? `` : `, over ${split.length} such runs`) + `.

` +
            `Those are two readings of one question. The probe is a span set in ` +
            `${INK_CANARY_FALLBACK} at that very request with ` +
            `CSS.getPlatformFontsForNode asked what drew it; the advances are ` +
            `measureText on one canvas for one string, once under the compositor's ` +
            `family and once under ${INK_CANARY_FALLBACK} alone. One face measured ` +
            `twice returns one width, so a name saying the two requests reached the ` +
            `same face while the widths say otherwise means one of the readings is ` +
            `about something else: a platform naming a face that did not draw this ` +
            `string, a family list resolving past its head for these glyphs, or a ` +
            `probe filed under a request it was not mounted at.

` +
            `What it costs is a sentence that reads as one finding and is two. The ` +
            `canary's premise is decided on the advances — see ` +
            `inkCanvasFallbackFault — and the face it NAMES comes from the probe, ` +
            `so a reader is handed a number from one reading and a noun from the ` +
            `other while the two are about different faces. The opposite direction, ` +
            `two names over one advance, is that fault's own blind arm and is ` +
            `reported there rather than twice here`,
    };
}

// Whether the generic the canary falls back to measures like the faces this
// grid is drawn with.
//
// # The premise the canary itself rests on
//
// inkCanvasNameReached reads "the two advances came apart" as "the family in
// front of the generic reached a face". That inference has a second leg the
// reading cannot see: the generic must resolve to a face with DIFFERENT widths.
// Handed a browser where `monospace` maps onto the very family the text is set
// in, the two requests resolve to one face, the advances are equal for every
// run, and the canary reports the join as having bitten nothing — everywhere,
// silently, while the join was working perfectly.
//
// INK_CANARY_FALLBACK's own note argues that a fixed-advance face is nothing
// like this grid's proportional ones. That is an argument about a font stack on
// some machine, not a reading of this one.
//
// # Why the browser is not asked what the face is CALLED
//
// It was, and a name is the one thing this cannot be decided by. The first
// version of this compared the family strings CSS.getPlatformFontsForNode
// returned for the generic's probe against the strings it returned for the
// joined runs. Those come from one API on one page, so they agree today — and
// a platform that reports one face under two spellings ("Times" for the run,
// "Times New Roman" for the generic's node) makes the comparison miss and the
// trap go unreported. That is the silent direction again, reached from inside
// the reader that exists to close it.
//
// The metric answer is already in hand at the counter and is the answer the
// canary actually needs. What breaks the canary is not two nodes sharing a face
// NAME, it is two requests producing the same ADVANCE — and that is precisely
// inkCanvasGenericGap, measured over each run's own string at its own size, on
// the same canvas, in the same evaluate. A gap inside a LayoutUnit is a canary
// with nothing to read.
//
// The bound is LAYOUT_UNIT rather than INK_CANARY_MARGIN because this is the
// canary's own decision bound: inkCanvasNameReached calls anything under it "the
// name reached nothing", so a run whose face lands that close to the generic is
// one the canary CANNOT answer for. INK_CANARY_MARGIN is the wider bracket, and
// it is asked in the census over the runs that did separate — the two populations
// are complements and each is held to the bound that decides it.
//
// This subsumes the name test rather than dropping it: one face under two names
// gives one advance, so every match the string comparison could find is a match
// this makes, and the ones it misses are exactly the ones it was missing.
//
// # What the arms mean
//
// No gap readable at all is the premise unmeasured, and it is reported for the
// reason the canary exists: a silence must not be counted as a confirmation.
// Every joined run carries both advances by construction, so this arm is the
// reader that fires if that ever stops being true.
//
// A run whose face measures like the generic is the trap itself. Asked of the
// JOINED runs only, not of every box on the page: a probe or a ligature node
// that happens to be drawn in the generic's face costs the canary nothing,
// because no canary is put to those boxes.
function inkCanvasFallbackFault(faces, rows, count) {
    if (count === 0) return null;
    const gaps = rows.map(inkCanvasGenericGap).filter((g) => g !== null);
    const named = inkCanaryFaceList(faces);
    if (gaps.length === 0) {
        return `nothing measured ${INK_CANARY_FALLBACK} against the faces this grid ` +
            `is drawn by, so the canary behind ${count} canvas joins is resting on an ` +
            `unread premise.\n\n` +
            `That canary measures one string under ${INK_CANARY_FALLBACK} alone and ` +
            `under the compositor's family in front of it, and reads a difference as ` +
            `the family having reached a face. The reading is only that if the two ` +
            `requests resolve to faces of different widths, which is a fact about this ` +
            `machine's font stack and is answered by the pair of advances every joined ` +
            `run already produced. Both of them come back from one evaluate, so a run ` +
            `carrying neither is that evaluate having returned something this file does ` +
            `not recognise`;
    }
    const blind = rows.filter((m) => {
        const gap = inkCanvasGenericGap(m);
        return gap !== null && gap < LAYOUT_UNIT;
    });
    if (blind.length === 0) return null;
    // The worst of them, because the message is about a bound and the reader
    // wants the number furthest from it that still fell inside.
    const widest = blind.reduce(
        (w, m) => Math.max(w, inkCanvasGenericGap(m)), 0);
    const example = blind[0];
    return `${INK_CANARY_FALLBACK} measures like the face ` +
        `${blind.length === 1 ? "one" : blind.length} of this grid's ${count} joined ` +
        `runs ${blind.length === 1 ? "is" : "are"} drawn by — no more than ` +
        `${widest.toFixed(6)}px apart over ${blind.length === 1 ? "that run's" :
        "those runs'"} own string${blind.length === 1 ? "" : "s"} (e.g. ` +
        `"${example.text}", ${inkCanvasGenericGap(example).toFixed(6)}px), against a ` +
        `LayoutUnit of ${LAYOUT_UNIT}` +
        (named === null ? `` : ` — the generic resolves to ${named} here`) + `.\n\n` +
        `So the canary behind those joins is comparing a face with itself. It ` +
        `measures one string under ${INK_CANARY_FALLBACK} alone and under the ` +
        `compositor's family in front of ${INK_CANARY_FALLBACK}. Both requests now ` +
        `come back with the same advance, and equal is exactly what this file reads as ` +
        `"the name reached nothing". Every such join reports as silent, the tail ` +
        `recites them as unconfirmed, and the reader is sent after an unreachable ` +
        `family name that the page can reach.\n\n` +
        `This is asked of advances rather than of face names on purpose: one face ` +
        `reported under two spellings would pass a name comparison and break the ` +
        `canary just the same. INK_CANARY_FALLBACK is a generic chosen on a metric ` +
        `argument — a fixed-advance face's widths for these strings are nothing like ` +
        `a proportional one's — and this is that argument failing on this machine ` +
        `rather than in this file. A different generic, or a family this grid does ` +
        `not draw with, restores the separation`;
}

function inkCanvasFaceFault(where, subject, faces, m) {
    // One face is the arm above this one's business: with a run drawn by two,
    // there is no single family to name and inkFaceFault has already said so.
    const family = faces && faces.length === 1 ? faces[0].family : null;
    if (!family) return null;
    if (!m) {
        return `${where}: nothing measured ${subject} against ${family}, so the ink ` +
            `band and the run advance over this box are a canvas's numbers with no ` +
            `face on them. The read rides in the same evaluate as the two ` +
            `fingerprints after the last CSS.getPlatformFontsForNode; an entry ` +
            `missing from it is that measurement not having been taken for this box`;
    }
    if (m.unspellable) {
        return `${where}: the compositor drew ${subject} with ${m.unspellable}, and a ` +
            `canvas will not take that as a font family — so the measurement that ` +
            `would hold the ink band and the run advance to this face could not be ` +
            `made.\n\n` +
            `A platform font name is a FACE's name and a CSS font-family is a ` +
            `request for one; they coincide often enough to be worth asking, and this ` +
            `is the browser saying they do not coincide here. Every number the ink ` +
            `scan takes off its canvas is still whichever face that canvas resolved, ` +
            `and nothing in this run says which face that is`;
    }
    if (!m.asked) {
        return `${where}: ${subject} were not re-measured against ${family} — the ` +
            `after-faces evaluate looked the box up by the same data-node-path the ` +
            `rects were read at and found no element there.\n\n` +
            `A tree that moved does not produce this. TREE_FINGERPRINT_JS hashes ` +
            `every [data-node-path] in the document, path included, and its second ` +
            `reading is taken in this very evaluate — so a path that stopped ` +
            `resolving between the rects and here moves that hash, treeMovedFault ` +
            `reports it, and facesHeld drops this consultation before the message is ` +
            `composed. What reaches here instead is one evaluate's querySelectorAll ` +
            `and querySelector disagreeing about the same attribute. This arm is the ` +
            `reader that would fire if the fingerprint's population ever stopped ` +
            `being the population these lookups draw from`;
    }
    if (typeof m.canary !== "number") {
        return `${where}: the canvas would not take ${family} in front of ` +
            `${INK_CANARY_FALLBACK}, so nothing here says whether that name reaches ` +
            `a face.\n\n` +
            `The family alone was spellable — the arm above it would have fired ` +
            `otherwise — and the same name followed by a generic was not, which is ` +
            `the canvas refusing a list it accepted the head of. Without that ` +
            `reading, the join below it is one-sided with nothing pricing the one ` +
            `side: an equal pair of advances would mean either that the face was ` +
            `named and matched or that the name resolved to nothing and was skipped`;
    }
    if (!inkCanvasNameReached(m)) {
        return `${where}: ${subject} are drawn by ${family}, and no canvas on this ` +
            `page can reach that name — ${INK_CANARY_FALLBACK} alone and ` +
            `${family} in front of ${INK_CANARY_FALLBACK} both measure ` +
            `${m.base.toFixed(4)}px for the same string ("${m.text}"), against a ` +
            `LayoutUnit of ${LAYOUT_UNIT}.\n\n` +
            `So the join was made and bit nothing. It puts the compositor's family at ` +
            `the head of this element's own list, and a head that resolves to no face ` +
            `is skipped: the list resolves as it did, the two advances are equal, and ` +
            `the check goes quiet for the one reason that is not evidence. Prepending ` +
            `rather than substituting is deliberate — a platform font name is a ` +
            `FACE's name and need not be a family CSS can reach, and substituting ` +
            `would measure the canvas default and report every run in this grid — so ` +
            `the silence is the design working and this is the reading that keeps it ` +
            `from being counted as a success.\n\n` +
            `Every canvas number over this box is still unheld: the ascent the three ` +
            `sampled rows are fractions of, and the advance inkRunRectFault holds the ` +
            `run rect's width to`;
    }
    const off = m.asIs - m.named;
    if (Math.abs(off) < LAYOUT_UNIT) return null;
    return `${where}: ${subject} are drawn by ${family}, and the canvas the ink band ` +
        `and the run advance are measured on makes them ${m.asIs.toFixed(4)}px wide ` +
        `against ${m.named.toFixed(4)}px for ${family} itself — ` +
        `${Math.abs(off).toFixed(4)}px ${off > 0 ? "wider" : "narrower"} for the same ` +
        `string ("${m.text}"), against a LayoutUnit of ${LAYOUT_UNIT}.\n\n` +
        `Both numbers are measureText on one canvas in one evaluate, and the only ` +
        `difference between the two requests is that the second names the face ` +
        `CSS.getPlatformFontsForNode reports at the head of this element's own family ` +
        `list. Naming a face the canvas has already resolved to changes nothing, so a ` +
        `difference is the canvas on some other face — and a font shorthand ` +
        `assembled out of four computed longhands is how it gets there: ` +
        `font-stretch, font-size-adjust, the font-variant longhands and the ` +
        `variation settings each select a face, and none of them is in it.\n\n` +
        `What that costs is every canvas number over this box: the ascent the three ` +
        `sampled rows are fractions of, and the advance inkRunRectFault holds the run ` +
        `rect's width to. The rect is the compositor's and that advance is not, so ` +
        `the pair is two faces being compared and its message names the layout`;
}

// What this build's face does with the pairs the fixture refuses.
//
// # A refusal with nothing behind it but a claim about faces in general
//
// gen.go's inkGlyphPerCharacter refuses any fixture string containing one of
// inkLigatureSeeds' nine pairs, so that a word a face is entitled to draw as
// one glyph fails at the fixture instead of arriving in inkFaceFault's third
// arm as a font substitution. The list is deliberately a superset: it is nine
// pairs chosen from what serif faces commonly carry, and being wrong towards
// refusing an innocent pair costs a fixture author one word.
//
// That argument is about faces in general, and it was the whole of what stood
// behind the list — while the one party who can answer for THIS face was
// already on the line. Every glyph count in this check comes from
// CSS.getPlatformFontsForNode, and a glyph count for a two-character node is
// precisely the answer to "does this face draw 'st' as one glyph".
//
// So the pairs are mounted (see INK_LIGATURE_GRID) and asked. Neither answer
// is a failure. A pair drawn as one glyph is a refusal this build earns; a
// pair drawn as two is a refusal carried for a build that is not this one,
// which is a fair thing to carry and a different thing to say — and the
// difference is what the tail now reports instead of the list's size.
//
// # What IS a failure
//
// That the row answered about the face the grid draws with. The measurement is
// taken in one declaration — the first probe's — and it is worth nothing to
// the runs unless the family it resolves is a family those runs resolve too.
// A row drawn by some other face is a perfectly confident census about
// somebody else's ligatures, which is the same shape of mistake the tree
// fingerprint exists to stop one level up.
//
// And that the grid has ONE face for it to speak for. "The row's family is one
// of the grid's" is not "the row's family is the grid's": eleven probes are
// eleven declarations, and two differing in weight are two faces that may
// ligate differently. What holds the grid to one face today is inkFaceFault's
// last arm, and that is a per-pair claim — each scanned box against its own
// probe — which is exactly the claim a census over the whole grid cannot be
// derived from. So the grid-wide one is asked here, where it is being rested
// on.
function inkLigatureCensus(seeds, runFamilies) {
    const problems = [];
    const ligated = [], carried = [];
    let family = null;
    for (const s of seeds) {
        const chars = [...s.pair].length;
        if (!s.faces || s.faces.length === 0) {
            problems.push(`the ligature row's ${JSON.stringify(s.pair)} reports no ` +
                `platform font, so whether this face draws it as one glyph could not ` +
                `be asked. That pair is refused to every fixture string in the grid ` +
                `and this row is the only thing that measures the refusal`);
            continue;
        }
        if (s.faces.length > 1) {
            problems.push(`the ligature row's ${JSON.stringify(s.pair)} is drawn by ` +
                `${s.faces.length} faces — ${inkFaceList(s.faces)} — so its glyph ` +
                `count is two faces' and says nothing about either. Two characters ` +
                `that fell back are not a pair this row can report on`);
            continue;
        }
        if (family === null) {
            family = s.faces[0].family;
        } else if (family !== s.faces[0].family) {
            problems.push(`the ligature row is drawn by ${family} and its ` +
                `${JSON.stringify(s.pair)} by ${s.faces[0].family}. Every node in the ` +
                `row carries one declaration, so two families in it means the face ` +
                `was chosen per string — and a census over pairs that were not all ` +
                `put to the same face is not a census`);
            continue;
        }
        // Fewer glyphs than characters is the ligature. More would be a
        // decomposition, which cannot happen here — these are ASCII pairs —
        // and is counted as "not ligated" rather than guessed at.
        if (s.faces[0].glyphs < chars) ligated.push(s.pair);
        else carried.push(s.pair);
    }
    if (runFamilies.size > 1) {
        problems.push(`the grid's runs are drawn by ${runFamilies.size} faces — ` +
            `${[...runFamilies].sort().join(", ")} — and the ligature row is one ` +
            `declaration` +
            (family === null ? `` : ` (${family})`) + `.\n\n` +
            `The row measures what inkGlyphPerCharacter's refusal is worth against the ` +
            `face it was mounted in, and gen.go mounts it in the first probe's ` +
            `declaration on the argument that the grid has one face. What holds that ` +
            `argument today is inkFaceFault's last arm, which is a per-pair claim: ` +
            `each scanned box agrees with ITS OWN probe, and the grid mounts one probe ` +
            `per text declaration it reads — two of those differing in weight are two ` +
            `faces that may ligate differently. With more than one family resolved ` +
            `across the grid ` +
            `the census covers the strings drawn in one of them and carries nothing ` +
            `for the rest — and which of the refused pairs the other face joins is ` +
            `unmeasured rather than measured and passing`);
    }
    if (family !== null && runFamilies.size > 0 && !runFamilies.has(family)) {
        problems.push(`the ligature row is drawn by ${family} and no run this grid ` +
            `scans is: those are drawn by ${[...runFamilies].join(", ")}.\n\n` +
            `The row exists to measure what inkGlyphPerCharacter's refusal is worth ` +
            `against the face the fixture strings are actually drawn in, and it was ` +
            `put to a different one. gen.go draws the row in the first probe's ` +
            `declaration precisely because every scanned box and the probe answering ` +
            `for it are held to one resolved family — see inkFaceFault's last arm, ` +
            `which is the check that makes "the grid has one face" a fact rather ` +
            `than an assumption`);
    }
    return { problems, ligated, carried, family };
}

// The page both readers have to be describing.
//
// # Two readers, one document, and nothing saying so
//
// Every other measurement this check makes comes back from ONE
// Runtime.evaluate: the rects, the run metrics, the ancestry sweeps and the
// computed styles are all taken inside a single expression, so they are of one
// layout by construction and a mount landing between two of them is not a
// thing that can happen.
//
// The platform-font read is not that shape. CSS.getPlatformFontsForNode takes
// a node id, a node id comes from DOM.querySelector, and DOM.querySelector is
// asked against a root id taken once by DOM.getDocument before any of them —
// so the faces are dozens of round trips over a document that could in
// principle change under them. A mount that replaced the tree between the
// rects and the faces would leave the second reader answering about boxes the
// first one never saw, and NEITHER READ COULD SAY SO: a face is a family name
// and a glyph count, and both are entirely plausible for the wrong element.
// The failure would arrive as inkFaceFault's third arm — a run whose glyph
// count is not its character count — pointing at a font substitution that did
// not happen.
//
// So the page is fingerprinted on both sides of the protocol reads, with the
// same expression, and the FIRST one is taken inside the evaluate that returns
// the rects. That is what makes it the rects' own page rather than a third
// reading of a third moment.
//
// # What is in the fingerprint, and what deliberately is not
//
// How many elements the runtime has mounted, and a hash over each one's path,
// tag, child count and text length in document order.
//
// Not the geometry. A scroll, a resize or a font that finished loading moves
// every rect in the page and changes no fact about which element is which, and
// the claim being made here is about IDENTITY — that the node id the protocol
// resolved for a path names the element the page read that path at. Folding
// layout into it would turn a benign relayout into a failure about fonts.
//
// That exclusion is right for the question this one answers and it leaves the
// other half of the window unasked. See LAYOUT_FINGERPRINT_JS, which asks it.

// The hash both fingerprints are built with.
//
// Carried as source rather than as a function, because these are two
// expressions evaluated in the page and not two calls into this file. One
// definition so that a change to the mixing cannot leave the two answering in
// different arithmetic — they are compared with each other and never with a
// recorded number, so a drift would not show up as a wrong hash, it would show
// up as two readers that never agree.
const FNV1A_JS = `
    // FNV-1a. Math.imul keeps the multiply in 32 bits, which is what makes
    // this the same number on every run rather than a double that has started
    // rounding away the low ones.
    let h = 2166136261;
    const feed = (s) => {
        for (let i = 0; i < s.length; i++) {
            h = Math.imul(h ^ s.charCodeAt(i), 16777619);
        }
    };`;

const TREE_FINGERPRINT_JS = `(() => {
    const all = document.querySelectorAll("[data-node-path]");
    ${FNV1A_JS}
    for (const e of all) {
        // "|" separates the fields and is not a character a data-node-path can
        // carry — they are "root" and slash-joined indices — so two different
        // trees cannot feed one byte sequence by running fields together.
        feed(e.getAttribute("data-node-path") + "|" + e.tagName + "|" +
            e.childElementCount + "|" + e.textContent.length + "||");
    }
    return { n: all.length, hash: (h >>> 0).toString(16) };
})()`;

// Returns null when the tree the protocol read is the tree the page read.
function treeMovedFault(before, after, paths) {
    if (!before || !after) {
        return `the page was not fingerprinted on both sides of the ${paths} ` +
            `platform-font reads, so nothing holds them to describing the same ` +
            `document as the rects.\n\n` +
            `The first fingerprint rides back with the rects, inside the same ` +
            `Runtime.evaluate; the second is taken after the last ` +
            `CSS.getPlatformFontsForNode. One of them did not arrive, which means the ` +
            `read that carries it did not happen — see the note above ` +
            `TREE_FINGERPRINT_JS for why the two readers need joining at all`;
    }
    if (before.n === after.n && before.hash === after.hash) return null;
    return `the page changed under the ${paths} platform-font reads: it held ` +
        `${before.n} mounted elements when the rects were taken and ${after.n} ` +
        `afterwards` +
        (before.n === after.n
            ? ` — the same count, and a different tree (${before.hash} then ` +
              `${after.hash})`
            : ``) + `.\n\n` +
        `Every other measurement in this check comes back from one evaluate and is ` +
        `of one layout by construction. The faces are not: each is a ` +
        `DOM.querySelector against a root node id taken before any of them, and a ` +
        `tree that moved in between hands back ids for boxes the rects are not ` +
        `about. What that produces downstream is a run whose glyph count is not its ` +
        `character count, reported as a font substitution — a true statement about ` +
        `some other page, delivered as a finding about this one. Nothing below has ` +
        `consulted a face`;
}

// The layout both the rects and the screenshot have to be of.
//
// # The other side of the same window
//
// TREE_FINGERPRINT_JS joins the two readers on IDENTITY and says why it leaves
// geometry out: a scroll or a resize moves every rect in the page and changes
// no fact about which element is which, so folding layout into that hash would
// turn a benign relayout into a failure about fonts.
//
// The gap that leaves is on the other side of the same window. Between the
// evaluate that returns the rects and the capture that returns the pixels sit
// two round trips — Page.captureScreenshot itself, and the read that comes
// back with it — and EVERY ink assertion in this grid samples the capture at
// `rect.x * dpr`. The rects say where a box is; the screenshot says what
// colour is at a coordinate. A relayout in between moves the pixels under
// coordinates the rects already reported, and what comes back is a colour read
// out of the wrong box: a band fill that is the band above's, an ink extent
// measured across a gap, a fringe found in a margin. The tree fingerprint
// reports the page unchanged throughout, because it IS unchanged — nothing
// about which element is which has moved.
//
// So a second fingerprint, over the same elements' geometry, bracketing the
// capture rather than the font reads. The two windows nest and neither
// subsumes the other: identity has to hold as far as the last font read, and
// layout has to hold as far as the shutter — and then again from the shutter
// to that same last font read, which is a third reading of this expression and
// a window of its own. See faceLayoutFault for what fits through it.
//
// # Over every mounted element, and not over the boxes that get sampled
//
// The honest population would be the rects this check actually samples, and
// they are not available as one list: the paths are per band and per probe,
// and asking the second reader for them would be thirty expressions where the
// first took one. Every mounted element is a superset of them, so a layout
// this hash calls unchanged is one in which no sampled box moved — which is
// the direction that matters. Being wrong the other way costs a run its ink
// coverage and says so; being wrong the direction a narrower hash would risk
// costs a reader a confident finding about the wrong box.
//
// # The ratio rides back with it
//
// devicePixelRatio was a round trip of its own, taken after the capture and
// before the faces, and it multiplies every sample point this check computes.
// It comes back inside this expression instead, so the number the coordinates
// are scaled by and the fingerprint that holds those coordinates to the
// capture are of one read rather than of two.
const LAYOUT_FINGERPRINT_JS = `(() => {
    const all = document.querySelectorAll("[data-node-path]");
    ${FNV1A_JS}
    for (const e of all) {
        // The rect as the page reports it, at full precision. An unchanged
        // layout gives an identical double on both reads, so nothing is
        // rounded here: rounding would be a tolerance, and the tolerance that
        // matters is a device pixel, which is what the sample points are
        // rounded to and not what a fingerprint should be deciding.
        const r = e.getBoundingClientRect();
        feed(e.getAttribute("data-node-path") + "|" + r.x + "|" + r.y + "|" +
            r.width + "|" + r.height + "||");
    }
    return {
        n: all.length, hash: (h >>> 0).toString(16),
        // The three page-wide numbers a rect cannot carry. A scroll moves
        // every viewport-relative rect and is caught by the hash anyway; these
        // are here so the message can say WHICH of the things that move a
        // capture moved, rather than reporting a changed hash and leaving the
        // reader to guess.
        dpr: window.devicePixelRatio,
        scrollX: window.scrollX, scrollY: window.scrollY,
        viewW: window.innerWidth, viewH: window.innerHeight,
    };
})()`;

// What moved between two readings, named one at a time.
//
// Split out because this fingerprint brackets two windows now — the rects to
// the shutter, and the shutter to the last platform-font read — and the four
// ways a layout can differ are the same list for both while what they COST is
// not. One list and two consequences, rather than one message hedging about
// which of the two readers it is warning.
function layoutMovedList(before, after) {
    const moved = [];
    if (before.dpr !== after.dpr) {
        moved.push(`the device pixel ratio went from ${before.dpr} to ${after.dpr}`);
    }
    if (before.scrollX !== after.scrollX || before.scrollY !== after.scrollY) {
        moved.push(`the page scrolled from (${before.scrollX}, ${before.scrollY}) to ` +
            `(${after.scrollX}, ${after.scrollY})`);
    }
    if (before.viewW !== after.viewW || before.viewH !== after.viewH) {
        moved.push(`the viewport went from ${before.viewW}×${before.viewH} to ` +
            `${after.viewW}×${after.viewH}`);
    }
    if (before.n !== after.n) {
        moved.push(`the page held ${before.n} mounted elements and now holds ` +
            `${after.n}`);
    } else if (before.hash !== after.hash) {
        moved.push(`the same ${before.n} elements are at different rects ` +
            `(${before.hash} then ${after.hash})`);
    }
    return moved;
}

// Returns null when the pixels the capture carries are of the layout the rects
// describe.
function layoutMovedFault(before, after) {
    if (!before || !after) {
        return `the page was not fingerprinted on both sides of the screenshot, so ` +
            `nothing holds the capture to the layout the rects were read from.\n\n` +
            `The first fingerprint rides back with the rects, inside the same ` +
            `Runtime.evaluate; the second is taken immediately after ` +
            `Page.captureScreenshot. One of them did not arrive, which means the read ` +
            `that carries it did not happen — see the note above ` +
            `LAYOUT_FINGERPRINT_JS for why a capture needs joining to the rects at ` +
            `all`;
    }
    const moved = layoutMovedList(before, after);
    if (moved.length === 0) return null;
    return `the page relaid out between the rects and the screenshot: ` +
        `${moved.join(", and ")}.\n\n` +
        `Every ink assertion in this grid samples the capture at \`rect.x * dpr\`: ` +
        `the rects say where a box is and the capture says what colour is at a ` +
        `coordinate, and the two are only one statement while the layout holding ` +
        `them together has not moved. It has, so a sample taken at a rect this ` +
        `check holds reads a pixel out of some other box — a band's fill answered ` +
        `by its neighbour, an extent measured across a gap — and every one of those ` +
        `would be reported with the confidence of a screenshot.\n\n` +
        `The tree fingerprint says nothing about this and is right not to: it is ` +
        `about which element is which, and that has not changed. See ` +
        `LAYOUT_FINGERPRINT_JS. No pixel below this line was read`;
}

// The third window: the shutter to the last face read.
//
// # What the first two hold, and the gap between them
//
// The tree fingerprint brackets the rects to the last platform-font read and
// holds IDENTITY across it. The layout fingerprint brackets the rects to the
// shutter and holds GEOMETRY across that. Between them they leave one window
// unasked, and it is the one the faces are actually read in: from the capture
// to the last CSS.getPlatformFontsForNode.
//
// A font that finished loading in there is the thing that fits through it. It
// moves every run's rect and changes no fact about which element is which, so
// the tree fingerprint is silent and correct; the layout question had already
// been asked and answered at the shutter, so that one is silent too. What
// comes back afterwards is the NEW face's glyph counts and family names, laid
// beside a capture painted with the old one — inkFaceFault comparing a
// substitution against pixels that cannot show it, and a ligature census about
// a face the grid was not drawn in.
//
// So the layout reading taken at the shutter is compared with one taken after
// the last face read, and the two rides come back in one evaluate: the tree
// fingerprint had to be taken there anyway, and two round trips would put a
// window between the two readings that close the windows.
//
// # What this cannot see
//
// A face swap that moved no mounted box. The hash is over rects, so a font
// whose metrics happen to match the one it replaced, or one under text in a
// box whose size is declared rather than measured, changes glyph counts
// without changing a single number in here. That is a miss rather than a false
// finding, which is the direction every fingerprint in this file is wrong in —
// and the run rects the ink scan holds are content-sized, so the swap this
// window exists for is one that does move them.
//
// Returns null when the faces were read in the layout the capture was taken
// in.
function faceLayoutFault(before, after, paths) {
    if (!before || !after) {
        return `the layout was not fingerprinted on both sides of the ${paths} ` +
            `platform-font reads, so nothing holds the faces to the layout the ` +
            `screenshot was taken in.\n\n` +
            `The first of those two readings is the one taken at the shutter; the ` +
            `second rides back with the tree fingerprint after the last ` +
            `CSS.getPlatformFontsForNode. One of them did not arrive, which means the ` +
            `read that carries it did not happen — see the note above ` +
            `faceLayoutFault for what fits through the window they close`;
    }
    const moved = layoutMovedList(before, after);
    if (moved.length === 0) return null;
    return `the page relaid out under the ${paths} platform-font reads: ` +
        `${moved.join(", and ")}.\n\n` +
        `The rects, the capture and the faces are three readings of one rendering, ` +
        `and this is the window between the last two. A font that finished loading ` +
        `in it moves every run's rect and changes no fact about which element is ` +
        `which — so the tree fingerprint holds, and the layout question had already ` +
        `been asked at the shutter — while the glyph counts read afterwards are the ` +
        `new face's and every pixel sampled below is the old one's.\n\n` +
        `What that produces is inkFaceFault's third arm reporting a substitution the ` +
        `capture cannot show, or a ligature census about a face this grid was not ` +
        `drawn in. Both would be true statements about a rendering nothing here ` +
        `measured. No face was consulted below this line`;
}

// One chain, reported.
//
// Returns null when every ancestor is neutral, which is every mount this grid
// makes today: the band grid and the probe Row are plain boxes and rows with
// fills and paddings, and none of INK_PATH_PROPS' properties is anything
// core.Style can even spell on them.
//
// A null chain — the element was not found — is itself a fault, because the
// alternative is reporting "the ancestry is neutral" about an element nothing
// looked at.
function inkAncestryFault(where, subject, chain) {
    if (chain === null || chain === undefined) {
        return `${where}: nothing could be read about what sits above ${subject}, so ` +
            `whether an ancestor is deciding how those glyphs are drawn is unknown. ` +
            `The antialiasing probe copies the element's own declaration and can ` +
            `reproduce nothing above it — see INK_PATH_PROPS`;
    }
    if (chain.length === 0) return null;
    const first = chain[0];
    return `${where}: ${subject} sit under ${first.at}, whose ${first.prop} is ` +
        `${first.got} rather than ${first.neutral}` +
        (chain.length > 1 ? ` (and ${chain.length - 1} more)` : "") + `.\n\n` +
        `The antialiasing probe that licenses every ink reading over that box is a ` +
        `copy of the element's own Style on a white ground. It reproduces the ` +
        `declaration and it cannot reproduce an ancestor: a transform, an opacity, a ` +
        `filter or a will-change up the chain composites the subtree, and a browser ` +
        `draws text into a composited layer by a different route than text painted ` +
        `straight into the page. The probe would stay on the old route and go on ` +
        `reporting a grey box for a band that had moved off it.\n\n` +
        `Either that ancestor comes back out of this grid, or the probes stop being ` +
        `a statement about the boxes they answer for`;
}

// The other half of the same question: whether the two ELEMENTS resolve the
// same rendering path.
//
// # What the ancestry check leaves open
//
// inkAncestryFault holds every strict ancestor of both the probe and the box it
// answers for to INK_PATH_PROPS' neutral values, on the argument that the
// element's OWN declaration travels with the Style copy and so needs no
// comparison. It travels as a core.Style. What a browser resolves is CSS.
//
// A `Rotate` on the label is spelled in the Style and is copied. A
// `-webkit-font-smoothing`, a `will-change` or a `contain` that arrived through
// a stylesheet, a UA default for an element type, or a mapping the runtime
// grows later, sits on the element itself and in no Style at all — so it can be
// on the scanned box and not on the probe, or the other way round, and the
// chain above both of them stays perfectly neutral while they are drawn by
// different routes. That was the assumption the ancestry sweep rested on:
// measured above, and taken on trust at the one element the probe is a copy OF.
//
// # Why a comparison here and neutrality there
//
// The two chains have no reason to be the same shape and cannot be compared;
// the two ELEMENTS are supposed to be the same declaration and can be. So the
// pair is: every ancestor neutral, and the two elements equal. Together those
// say the resolved path is the same on both sides, which is the whole of what
// makes the probe's grey box a statement about the band.
//
// # Equal on what
//
// On everything. The comparison used to be INK_PATH_PROPS' twelve, which made
// this check exactly as wide as a list somebody assembled by argument; it is
// now every computed property the browser enumerates, minus the twenty-two
// INK_OWN_MAY_DIFFER gives a reason for. See that table for the reasons and for
// why the exception list is the sound direction to write it in.
//
// Returns { read, off, used } rather than deciding: `read` is how many
// properties were compared, `off` is the ones that differ with no permission,
// and `used` is the permissions this pair spends — which the census below needs
// from every pair, including the ones some other guard has already suppressed.
function inkOwnDiff(box, probe) {
    // The union, so a property present on one side and absent on the other is a
    // difference rather than a key nobody iterated.
    const props = [...new Set([...Object.keys(box), ...Object.keys(probe)])].sort();
    const differ = props.filter((prop) => box[prop] !== probe[prop]);
    return {
        read: props.length,
        off: differ.filter((prop) => !(prop in INK_OWN_MAY_DIFFER)),
        used: differ.filter((prop) => prop in INK_OWN_MAY_DIFFER),
    };
}

// Returns null when the two agree everywhere they are required to, which is
// every mount this grid makes today.
function inkOwnFault(where, subject, what, box, probe, browser) {
    if (!box || !probe) {
        return `${where}: ${!box ? subject + " resolve" : "the antialiasing probe for " +
            what + " resolves"} no computed properties at all — nothing was read from ` +
            `that element. The probe answers for this box on the argument that the two ` +
            `resolve the same rendering path, and with one side unread that is an ` +
            `assumption again` +
            (box ? `. The probe's own text node is the element inside its white Box ` +
                `(see inkProbeTextPath); a Box with no child there is a probe that ` +
                `paints no glyphs` : "");
    }
    const { read, off } = inkOwnDiff(box, probe);
    if (off.length === 0) return null;
    const first = off[0];
    return `${where}: ${subject} resolve ${first} as ${box[first]}, and the ` +
        `antialiasing probe for ${what} resolves it as ${probe[first]}` +
        (off.length > 1 ? ` (and ${off.length - 1} more of the ${read} computed ` +
            `properties differ without a permission: ${off.slice(1).join(", ")})` : "") +
        `.

` +
        `The probe is gen.go's copy of this element's core.Style, and the reason its ` +
        `grey box says anything about this band is that the two are supposed to be one ` +
        `declaration. A Style is not what a browser resolves: a property that reaches ` +
        `an element through a stylesheet, a UA default or a runtime mapping is on the ` +
        `element and in no Style, so it lands on one of these two and not the other ` +
        `while every ancestor of both stays neutral.

` +
        `All ${read} of this browser's computed properties are compared, and ` +
        `${Object.keys(INK_OWN_MAY_DIFFER).length} of them have a reason to differ — ` +
        `see INK_OWN_MAY_DIFFER. ${first} is not one of them, so either the two ` +
        `declarations have come apart, or that property belongs in the table with an ` +
        `argument beside it saying why a probe may resolve it differently and still be ` +
        `measuring this box's rendering path` +
        inkOwnBuildNote(read, browser);
}

// One band's tap-target ledger, spent.
//
// The reading is bandtarget.mjs's — see there for why the ledger's own arms
// live in a module and why the reading no longer moves the census — and this
// is the half that does move it: each part that held at its own counter, and
// `targets` for the conjunction the tail recites. One reading feeds both,
// which is the property that made this one function to begin with, and the
// conjunction is derived from the parts rather than carried beside them.
function bandTargetTally(where, target, asked, hasWrapper) {
    const read = bandTargetRead(where, target, hasWrapper);
    for (const counter of read.counters) asked[counter]++;
    if (read.whole) asked.targets++;
    return read.problems;
}

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
    Children: BAND_RENDERS.map((b) => JSON.parse(b.tree))
        .concat([INK_PROBE_GRID, INK_LIGATURE_GRID]),
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
// mobile/verify's TestTheComposeFixedDimensionSetsAMaximum and
// TestTheSwiftUIFixedDimensionProposesAndDoesNotClip for the two native call
// sites this rests on.
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
// The pinned Row
// --------------------------------------------------------------------------
//
// core.FlexShrink(0) on an overflowing Row, laid out by a browser, over the
// fixture the pin census is built on.
//
// # The sentence this replaces
//
// internal/pinfixture states one Row four ways and pairs two columns: a Compose
// one, which is a transcription of foundation-layout's measure loop, and a CSS
// one, which ios/verify produces by solving the same Row through
// GrMobFlexSolver. That solver is this repository's own flex arithmetic, split
// out so it can run without a simulator — it is not a browser.
//
// Check 9 above is where that distinction stopped being academic: under an
// offer narrower than the band, GrMobFlexSolver shrinks each child in
// proportion to a base that INCLUDES the child's own padding, and CSS
// distributes shrink over the inner flex base size, which excludes it. A real
// divergence, recorded in ios/verify/band.swift and confirmed by a browser one
// check up.
//
// The pin fixture's children have no padding, so the two rules coincide and the
// solver's column is CSS's — which is a piece of reasoning, made by whoever
// wrote the fixture, about whether a known divergence applies. Nothing had
// asked. This asks: the same six Rows, mounted, measured.
//
// # What is compared, and why none of it is a second transcription
//
// The one thing this must not do is compute the CSS answer in JavaScript. That
// would be a third spelling of a flex line — after the solver and the browser —
// and the check would then be about whether two transcriptions agree.
//
// So every claim here is one the FIXTURE states, held against measured pixels:
//
//	the census's column    the fixture STATES the CSS extents for every row, and
//	                       a browser lays them out. That is the claim the census
//	                       prints, and it is why the field exists: the three
//	                       claims below determine the pinned rows and determine
//	                       nothing at all about the control row, whose 24/80/16
//	                       was three numbers in a comment beside two mechanisms
//	                       that produce them.
//	the declaration        a pinned child lays out at its own base, in every
//	                       arrangement. That is what core.FlexShrink(0) says and
//	                       what the Compose column shows; a browser is the third
//	                       target to be asked.
//	the spacing            a flex line charges its gap between every adjacent
//	                       pair, whatever the children ended up at; a Compose
//	                       Row clamps each one to what is left. The fixture's
//	                       last row is the only case with a gap in it, and until
//	                       it existed the sentence three documents repeat was
//	                       measured by nothing. Both arms, like the agreement.
//	agreement, both ways   `mainsAgreeWithCSS` says whether these three extents
//	                       are the Compose ones. Both arms are asserted, for the
//	                       reason the fixture gives about deriving the flag: a
//	                       check that only ever confirmed a divergence would pass
//	                       just as well if the divergence quietly went away.
//	order does not matter  the three pinned rows are permutations of one set of
//	                       children, and a CSS flex line's sizes do not depend on
//	                       position. This is the CSS half of "the siblings
//	                       diverge" — the half ios/verify asserts of the solver —
//	                       and it is what makes Compose's order-dependence a
//	                       divergence rather than a coincidence.
//	the control shrinks    the no-pin row's 200px child comes back smaller, and
//	                       the line adds up to exactly the offer. One declaration
//	                       apart from the row above it.
//
// The last four pin the pinned rows to the exact numbers the census records —
// "the pin keeps its base" plus agreement with Compose on the first plus
// order-independence across all three. The control row is determined by none of
// them, which is what the stated column is for: 24/80/16 is the scaled-base
// rule's answer and nothing else here produces it. Its own two assertions —
// every child shrank, and the line fits exactly — are kept beside it, because
// they say WHY the three numbers are what they are and the column only says
// that they are.
//
// # The modelling, and the one declaration that is NOT load-bearing
//
// The Row declares the offer as a width and has no padding and no gap, which is
// the fixture's own shape: every number below is a child's extent against the
// container's, with nothing for the reader to subtract.
//
// Every child carries `min-width: 0`, and unlike bandTree's control it does
// nothing here. That is measured rather than assumed — a break-test removed it
// and every row laid out identically — and the reason is the one bandTree's
// badge already states about itself: CSS's automatic minimum for a flex item is
// min(its specified size, its content size), and these children are empty
// boxes, so their content size is 0 and their automatic minimum is 0 already.
// It is bandTree's *control* the declaration matters to, because that child
// wraps a label with a declared width.
//
// So it is kept as a statement of intent — the fixture is standing in for a
// child that can be squeezed, and GrMobFlexSolver has no content-based floor
// either — and named as inert, because an inert declaration a reader believes
// is load-bearing is worse than no declaration at all. Nothing in
// controls_test.go pins it, for the same reason: a control has to have a
// subject.
//
// # What the number is held to
//
// There is a second tolerance over this same fixture: ios/verify/pin.swift's
// pinEpsilon is 0.0001, four hundred times finer, because what IT compares is
// GrMobFlexSolver's own doubles rather than a browser's LayoutUnits. Two
// numbers for one table, each argued for on its own page, and until
// Case.resolution neither was held to anything.
//
// They are not supposed to agree. A tolerance bounds the machinery on one side
// of a comparison and the two sides here are Chrome and a Swift solver; what
// they share is the other side, which is internal/pinfixture, and what that
// side requires is the same of both. Case.resolution is the smallest distance
// apart any two of a case's numbers are — a tolerance at or above it accepts
// one of the fixture's numbers where another was meant — so both harnesses read
// it and each keeps its own value an order below.
const PIN_EPSILON = 0.05;
const pinSame = (a, b) => Math.abs(a - b) <= PIN_EPSILON;

// How far below a case's resolution the tolerance has to stay.
//
// INK_MARGIN's argument, applied to a different tolerance: what is claimed is a
// RATIO, so this is a factor rather than an absolute, and four is the smallest
// factor that is unambiguously an order. The bundled fixture clears it by a wide
// margin — the tightest case resolves 4 pixels against a tolerance of 0.05 — so
// this fires on numbers that have crowded together or on a tolerance somebody
// widened to make a failure go away.
const PIN_MARGIN = 4;

// One pinned-Row case as a tree. The pin is core.FlexShrink(0), which reaches
// this runtime as the same declaration the fixed-size cases above use.
function pinTree(c) {
    return {
        Type: "Row",
        Style: {
            Width: `${c.offer}px`,
            Padding: { Top: 0, Right: 0, Bottom: 0, Left: 0 },
            Gap: c.gap,
            AlignItems: "flex-start",
        },
        Children: c.children.map((child) => ({
            Type: "Box",
            Style: {
                Width: `${child.base}px`,
                Height: "20px",
                FlexGrow: 0,
                FlexShrink: child.pinned ? SHRINK_NONE : 1,
                // Intent rather than mechanism: these boxes are empty, so
                // their automatic minimum is already 0. See PIN_GRID.
                MinWidth: "0",
            },
        })),
    };
}

// Every pinned Row, in one Column, so the whole table is one mount and one
// round trip of rects — the same economy the widget grid and the band table are
// built on.
const PIN_GRID = {
    Type: "Column",
    Style: {
        Padding: { Top: 0, Right: 0, Bottom: 0, Left: 0 }, Gap: 0,
        AlignItems: "flex-start",
    },
    Children: PINS.map(pinTree),
};

// --------------------------------------------------------------------------

async function main() {
    const chromePath = findChrome();
    actOn(startupVerdict({
        transcriptPath: TRANSCRIPT,
        transcriptExists: TRANSCRIPT_EXISTS,
        widgets: WIDGETS.length,
        bands: BANDS.length,
        bandRenders: BAND_RENDERS.length,
        pins: PINS.length,
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
        //
        // The height is also the band grid's fold budget. That grid reads
        // pixels, so a band below the bottom of the viewport has a rect and no
        // paint, and foldVerdict names this number as the knob when it stops
        // fitting — which it did the moment the grid grew a fourth theme and a
        // row of antialiasing probes. It is raised rather than the grid
        // trimmed, because what the grid holds is the set of shapes and
        // palettes the ink scan has to be right about, and the viewport is the
        // arbitrary half of that pair. The scroll control is unaffected: its
        // subject is a 4000px filler, which is taller than this by an order.
        "--window-size=800,900",
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
    // The two counts the tail recites, filled in by the band grid. Declared out
    // here because the OK line is printed after the try block and a number
    // recited from inside it would be BAND_RENDERS.length again — see `asked`,
    // where the argument is.
    const asked = {
        // The tap target is three assertions and the tail recites their
        // conjunction, so both are counted: `targets` is the number the
        // sentence at the bottom is allowed to say, and the three beside it
        // are what the census reports. See the note above `declared`.
        targets: 0, targetLead: 0, targetTrail: 0, targetStretch: 0,
        insets: 0, fills: 0, words: 0, counts: 0,
        // And what the refusal behind the glyph-per-character claim comes to
        // on the face this browser resolved. Not a count of anything asserted
        // — see inkLigatureCensus, where neither answer is a failure — but the
        // tail recites the claim, and reciting it without this was reciting a
        // superset as though it had been measured.
        ligatureSeeds: 0, ligated: [], ligatureFamily: null,
        // And how wide this build's computed-style enumeration turned out to
        // be. INK_OWN_SHAPE_FLOOR is derived from the build the exception table
        // was written against; this is the population on the other side of it,
        // and the tail recites the bracket rather than the fraction.
        ownRead: 0,
        // And how many of this grid's runs had the canvas their ink band and
        // run advance are measured on held to the face the compositor named.
        // See inkCanvasFaceFault: the join is one-sided, so this is a count of
        // the runs where it could be asked and not of the runs that have a
        // face.
        canvasFaces: 0,
        // And how many of those named a family a canvas can actually reach.
        // The two are equal on a run that passes — a name that reaches nothing
        // is reported — and they are separate numbers because the first alone
        // was reciting a skipped join as a confirmed one. See the canary in
        // inkCanvasFaceFault.
        canvasNamed: 0,
        // And the widest those two advances ever came apart while staying
        // inside LAYOUT_UNIT. The bound is derived from two measurements of one
        // face being the same double; this is what the population does under
        // it, which is the half nothing recorded.
        canvasWidest: 0,
        // And the narrowest the CANARY's two advances came apart on a run
        // where the name reached a face — the same question asked of the
        // reading that decides canvasNamed, whose own bound has an argument
        // behind it that nothing was holding to the population. Null rather
        // than 0, because a minimum over no runs at all is not a thin margin.
        // See INK_CANARY_MARGIN.
        canvasNarrowest: null,
        // And which face INK_CANARY_FALLBACK reached on this browser, which
        // is the other half of that canary's premise: a generic that resolved
        // to the family the grid is drawn with would have the canary comparing
        // a face with itself. See inkCanvasFallbackFault.
        //
        // A NAME, and a name is what that premise cannot be decided by — see
        // inkCanvasFallbackFault, which decides it on the pair of advances
        // below. This is what the tail calls the generic and what a failure
        // prints beside its numbers; an empty read costs a sentence its noun
        // and costs the premise nothing.
        canvasFallback: null,
        // And how far the generic actually is from the nearest face this grid's
        // joined runs are drawn by, in pixels, over those runs' own strings at
        // their own sizes.
        //
        // This is the premise itself as a number: the canary reads "the two
        // advances came apart" as "the family reached a face", and that reading
        // is worth exactly this much. Null when no run was joined, because a
        // minimum over nothing is not a small separation.
        canvasGenericNearest: null,
        // And what the per-request probes came back with: how many requests the
        // generic was asked at, how many answered, and how many distinct faces
        // those answers were.
        //
        // The spread is what says whether asking per request bought anything —
        // see inkCanaryFaceSpread. One answer across every request is a stack
        // with no optical cut at these sizes, which is a reading and not an
        // assumption; more than one is the case a single probe at the page
        // default got wrong.
        canvasGenericReqs: 0, canvasGenericRead: 0, canvasGenericAnswers: 0,
        // And how many joined runs had BOTH readings of "is the generic this
        // run's face" available — the probe's name and the pair of advances —
        // and how many of those agreed. See inkCanaryAgreement: a comparison
        // asked of no runs is not agreement, and the tail has to be able to say
        // which of the two it had.
        canvasGenericPaired: 0, canvasGenericAgreed: 0,
        // And what that comparison declined to ask, which nothing counted:
        // probes that named more than one face, probes that came back with
        // none, and the runs those first skips cost the join. See
        // inkCanaryAgreement — a reading going quiet arrives as a smaller
        // population, and a smaller population with no reason beside it reads
        // as a smaller grid.
        canvasGenericMulti: 0, canvasGenericUnread: 0, canvasGenericUnjoined: 0,
        // And the axes the requests actually differ in, which is what bounds
        // the reads. See inkCanaryReqAxes.
        canvasGenericAxes: null,
    };
    try {
        const port = await devtoolsPort(profile);
        // Which browser this is, read before anything is measured in it.
        //
        // See INK_OWN_MEASURED_ON: the exception table is a list of properties
        // that differ, taken against ONE build's enumeration, and a build that
        // enumerates more brings properties into the comparison that nobody has
        // had an opinion about. That is the right failure and a confusing one
        // to meet, so the failure message gets to say which build the table was
        // written against and how far this one is from it.
        const versionInfo = await getJSON(`http://127.0.0.1:${port}/json/version`);
        const browserBuild = versionInfo.Browser || "an unidentified browser";
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
            const fold = foldVerdict({
                what: where, bottom: rects.page.y + rects.page.h,
                screen: wImg.height / dpr, knob: "WIDGETS_PER_ROW or the window size",
            });
            if (fold) {
                problems.push(fold);
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

        // The rects, and for the two boxes that get scanned for ink, the
        // metrics that say where in them a glyph actually is.
        //
        // The metrics come back with the rects rather than in a second round
        // trip because they are read off the same layout: a measurement taken
        // after another mount would be describing a page this one no longer is.
        // See inkBandRows for what the numbers are and why the probe glyph
        // decides the band.
        const renderRects = await evaluate(`${JSON.stringify(
            BAND_RENDERS.map((b, i) => ({
                band: bandRenderPath(i, b.band),
                wrapper: bandRenderPath(i, b.wrapper),
                control: bandRenderPath(i, b.control),
                badge: bandRenderPath(i, b.badge),
                label: bandRenderPath(i, b.label),
                chevron: bandRenderPath(i, b.chevron),
                leading: bandRenderPath(i, b.leading),
            })).concat(INK_PROBES.map((p, i) => ({
                  probe: inkProbePath(i), probeText: inkProbeTextPath(i),
              })))
              .concat([{ grid: "root" }]))}.map((paths) => {
            const el = (path) => path
                ? document.querySelector('[data-node-path="' + path + '"]') : null;
            const at = (path) => {
                const e = el(path);
                if (!e) return null;
                const r = e.getBoundingClientRect();
                return { x: r.left, y: r.top, w: r.width, h: r.height };
            };
            // Where this element's glyphs have ink, in page coordinates.
            //
            // The inline text box is the font's content area, so its top is the
            // baseline less the ascent — which is the one number no computed
            // style reports and no leading arithmetic here has to guess at. A
            // run that wrapped reports more than one box, and that is returned
            // rather than averaged: three fractions of a two-line box are three
            // rows of nothing in particular.
            const band = (path, probe) => {
                const e = el(path);
                if (!e) return null;
                const cs = getComputedStyle(e);
                const cx = document.createElement("canvas").getContext("2d");
                // One spelling of the shorthand, shared with the read that
                // holds this canvas to the face the compositor drew the run
                // with. See INK_FONT_SHORTHAND_JS and inkCanvasFaceFault.
                cx.font = (${INK_FONT_SHORTHAND_JS})(cs);
                // An unparsed font shorthand leaves the canvas on its own
                // default, and every number below would then be another face's.
                if (!cx.font.includes(cs.fontSize)) {
                    return { font: cx.font, wanted: cs.fontSize, lines: 0 };
                }
                const range = document.createRange();
                range.selectNodeContents(e);
                const rects = [...range.getClientRects()];
                if (rects.length !== 1) {
                    return { font: cx.font, lines: rects.length, text: e.textContent };
                }
                const m = cx.measureText(probe);
                const baseline = rects[0].top + m.fontBoundingBoxAscent;
                return {
                    font: cx.font, lines: 1, probe,
                    bandTop: baseline - m.actualBoundingBoxAscent,
                    bandBottom: baseline,
                    // The run's own horizontal extent, which is not the
                    // element's. A stretched text box is several times the
                    // width of the words in it, and the vertical band above was
                    // measured for the RUN — see the label scan for why the two
                    // halves have to be looking at the same rect.
                    runX: rects[0].left, runW: rects[0].width,
                    // And the same width from the other side. The rect is the
                    // LAYOUT's answer to "how wide is this run"; this is the
                    // same face's own advance for the same string, measured
                    // through the canvas that is already open for the ascent
                    // above and already held to having parsed the shorthand.
                    // See inkRunRectFault: two answers, so a rect that reported
                    // the wrong width is a disagreement rather than a fact.
                    runAdvance: cx.measureText(e.textContent).width,
                };
            };
            // What every element ABOVE this one contributes to the way its
            // glyphs are drawn. See INK_PATH_PROPS for the list and for why a
            // probe cannot reproduce any of it.
            //
            // Strict ancestors: the element's own declaration is the thing the
            // probe copies, so a transform on the label itself travels to the
            // probe with the rest of the Style and is not this question. Every
            // ancestor up to the document element, because a compositing layer
            // three boxes up decides a rendering path just as well as the
            // parent does.
            const pathProps = ${JSON.stringify(INK_PATH_PROPS)};
            const ancestry = (path) => {
                const e = el(path);
                if (!e) return null;
                const found = [];
                for (let p = e.parentElement; p; p = p.parentElement) {
                    const cs = getComputedStyle(p);
                    for (const prop of Object.keys(pathProps)) {
                        const neutral = pathProps[prop];
                        const got = cs[prop];
                        if (got === undefined || got === neutral) continue;
                        found.push({
                            at: p.getAttribute("data-node-path") || p.tagName.toLowerCase(),
                            prop, got, neutral,
                        });
                    }
                }
                return found;
            };
            // And what the element ITSELF resolves, which is the other half
            // of the same question. gen.go's probe copies the scanned node's
            // core.Style, and what a browser resolves is CSS: a property that
            // arrived through a stylesheet, a UA default or a runtime mapping
            // this file has never heard of sits on the element and not in the
            // Style, so it can be on one of the two and not the other. Read on
            // both sides and compared — see inkOwnFault.
            //
            // EVERY property, not pathProps' twelve. The declaration is
            // supposed to be the same on both sides, so the honest question is
            // "do these two elements resolve the same CSS" and the list of
            // exceptions is INK_OWN_MAY_DIFFER's, up here where it can carry an
            // argument. Enumerating the object rather than keying it off a list
            // is the point: cs.length is every longhand this build of the
            // browser exposes plus every custom property in effect, so a
            // property that ships in a later Chrome joins the comparison
            // without anybody editing this file.
            const own = (path) => {
                const e = el(path);
                if (!e) return null;
                const cs = getComputedStyle(e);
                const got = {};
                // getPropertyValue takes the dashed name the index yields, and
                // is the only accessor that reaches a custom property at all.
                for (let i = 0; i < cs.length; i++) {
                    got[cs[i]] = cs.getPropertyValue(cs[i]);
                }
                return got;
            };
            // Whether the element at a path is the one DRAWING the glyphs, or
            // a box with the glyphs somewhere inside it.
            //
            // See inkSubjectFault. Reported rather than decided, so the fault
            // message can say what it found: how many element children the
            // node has, whether any of its direct child nodes is text, and the
            // tag, which is what a reader needs to recognise the box.
            const subject = (path) => {
                const e = el(path);
                if (!e) return null;
                return {
                    tag: e.tagName.toLowerCase(),
                    elements: e.childElementCount,
                    directText: [...e.childNodes].some((n) =>
                        n.nodeType === 3 && n.textContent.trim() !== ""),
                    text: e.textContent,
                };
            };
            // Every path a DECLARATION is read at, and the only ones.
            //
            // Three reads are about one element for one reason: the computed
            // style the probe is held to, the chain swept above it, and the
            // subject guard saying the path names the box PAINTING the run
            // rather than a box with the run somewhere inside it. See
            // inkSubjectFault — the two jobs a path does here are "where the
            // pixels are" and "whose declaration is in question", and those
            // are the same element only for a glyph-drawing leaf.
            //
            // Written as three sets of hand-matched lines, as they were, a
            // fourth declaration read could be added at a path with no guard
            // beside it. Derived from one list it cannot be, and
            // inkDeclarationGuard holds the returned object to the same
            // pairing for anything written around this table.
            const declarations = [
                { at: "label", as: "label" },
                { at: "badge", as: "badge" },
                // Both of the probe's reads are taken at its text node rather
                // than at the white Box around it — see inkProbeTextPath. That
                // is the element drawing the glyphs, and it puts the Box into
                // its own ancestry sweep instead of leaving it neutral by
                // construction.
                { at: "probeText", as: "probe" },
            ];
            const out = {};
            for (const k of Object.keys(paths)) out[k] = at(paths[k]);
            if (paths.label) out.labelBand = band(paths.label, "x");
            if (paths.badge) out.badgeBand = band(paths.badge, "0");
            for (const d of declarations) {
                if (!paths[d.at]) continue;
                out[d.as + "Ancestry"] = ancestry(paths[d.at]);
                out[d.as + "Own"] = own(paths[d.at]);
                out[d.as + "Subject"] = subject(paths[d.at]);
            }
            // The complement, at the one place this file has a path for each
            // job: the probe's SAMPLE rect must not be a glyph-drawing leaf.
            // See inkProbeBoxFault.
            if (paths.probe) out.probeBoxSubject = subject(paths.probe);
            // What this read asked, said by the read.
            //
            // inkDeclarationGuard used to find the declaration reads by
            // looking for a key ending in "Own" or "Ancestry", which is the
            // convention the table above produces and not a property of it: a
            // read named for what it measures rather than for how — ownStyle
            // — is outside the guard by spelling alone, and so is a missing
            // Own beside a Subject that IS there. So the manifest travels
            // with the answers, and the guard is held to it rather than to a
            // suffix.
            out.reads = {
                declarations: declarations.filter((d) => paths[d.at])
                    .map((d) => d.as),
                // The one stem this file reads a subject at and no
                // declaration: it is the other half of the same question and
                // must not acquire the other two, because a probe's sample
                // rect is deliberately NOT the element that paints the run.
                subjectsOnly: paths.probe ? ["probeBox"] : [],
            };
            // And the tree these reads were taken from, so the platform-font
            // reads that follow them over the protocol can be held to the same
            // page. It rides with the grid's entry because it is one fact about
            // the document rather than one per path. See TREE_FINGERPRINT_JS.
            if (paths.grid) out.tree = ${TREE_FINGERPRINT_JS};
            // And the layout they were taken in, for the capture that follows
            // them to be held to. Rides with the grid's entry for the same
            // reason the tree does — it is one fact about the document — and
            // is taken here, in the rects' own evaluate, because "the layout
            // the rects describe" is not a thing a later read can go back and
            // ask about. See LAYOUT_FINGERPRINT_JS.
            if (paths.grid) out.layout = ${LAYOUT_FINGERPRINT_JS};
            return out;
        })`);
        // The grid's own box rides last and the probes before it, so the band
        // loop below still indexes by case.
        const gridRead = renderRects.pop();
        const probeReads = renderRects.splice(BAND_RENDERS.length);
        const probeRects = probeReads.map((r) => r.probe);

        // Every declaration that came back has its subject guard beside it.
        // See inkDeclarationGuard: asked of the whole read rather than of the
        // three paths that ask today, so the guard is about the shape of the
        // mistake and not about the current contents of the file.
        for (const [what, read] of [
            ...BAND_RENDERS.map((b, i) => [`${b.theme}/${b.what}`, renderRects[i]]),
            ...INK_PROBES.map((p, i) =>
                [`the antialiasing probe for ${p.what}`, probeReads[i]]),
            ["the band render grid", gridRead],
        ]) {
            const unguarded = inkDeclarationGuard(what, read);
            if (unguarded) problems.push(unguarded);
        }

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

        // And whether those pixels are of the layout the rects describe.
        //
        // See LAYOUT_FINGERPRINT_JS. The first reading rode back inside the
        // rects' own evaluate; this is the same expression, taken as soon as
        // the capture is in hand so the window it brackets is the shutter and
        // nothing else. The dpr every sample point below is multiplied by
        // comes back with it rather than as a read of its own.
        const layoutAfter = await evaluate(LAYOUT_FINGERPRINT_JS);
        const layoutFault = layoutMovedFault(
            gridRead ? gridRead.layout : null, layoutAfter);
        if (layoutFault) problems.push(layoutFault);
        // The fallback is for the one case layoutMovedFault reports and cannot
        // repair: with no second reading there is no ratio in it either, and
        // the fold guards below still have a screen height to state even
        // though nothing will sample a pixel.
        const bandDpr = layoutAfter && layoutAfter.dpr !== undefined
            ? layoutAfter.dpr : await evaluate(`window.devicePixelRatio`);

        // Which platform face actually drew each run.
        //
        // See inkFaceFault. This is the one answer the page cannot give: a
        // computed `font-family` is the request, not the resolution, and both
        // of the run's width answers come from the same shaper working on
        // whatever it resolved. CSS.getPlatformFontsForNode is the compositor's
        // own record of what it laid the glyphs out with — family name and
        // glyph count per face — so it is a THIRD party to a question the other
        // two agree on by construction.
        //
        // One DOM.getDocument for the tree and one querySelector per path. The
        // reads are batched here, with the rects and the screenshot, because
        // they are about this layout: a face read after another mount would be
        // describing a page this one no longer is.
        await session.send("DOM.enable");
        await session.send("CSS.enable");
        const domRoot = (await session.send("DOM.getDocument", { depth: 0 })).root.nodeId;
        const facesAt = async (path) => {
            if (!path) return null;
            const { nodeId } = await session.send("DOM.querySelector",
                { nodeId: domRoot, selector: `[data-node-path="${path}"]` });
            if (!nodeId) return null;
            const { fonts } = await session.send("CSS.getPlatformFontsForNode", { nodeId });
            return fonts.map((f) => ({ family: f.familyName, glyphs: f.glyphCount }));
        };
        const bandFaces = [];
        for (let i = 0; i < BAND_RENDERS.length; i++) {
            const b = BAND_RENDERS[i];
            bandFaces.push({
                label: await facesAt(bandRenderPath(i, b.label)),
                badge: await facesAt(bandRenderPath(i, b.badge)),
            });
        }
        const probeFaces = [];
        for (let i = 0; i < INK_PROBES.length; i++) {
            probeFaces.push(await facesAt(inkProbeTextPath(i)));
        }
        // And the refused pairs, in the same window as the rest: a glyph count
        // read after another mount is a count of some other page's glyphs, and
        // the fingerprint below covers these reads with the others.
        const ligatureFaces = [];
        for (let j = 0; j < INK_LIGATURES.length; j++) {
            ligatureFaces.push({
                pair: INK_LIGATURES[j].pair,
                faces: await facesAt(inkLigaturePath(j)),
            });
        }

        // And the runs whose canvas is about to be held to the face the reads
        // above just named. The table is built here rather than in the page
        // because the family is the protocol's answer and the path is this
        // file's; the page is handed the pairing and measures it. Keyed by
        // band and stem, not by position — the loops below ask for a box, and
        // an index into a flat array is a coupling nothing would notice
        // breaking. See inkCanvasFaceFault.
        const canvasRuns = [];
        for (let i = 0; i < BAND_RENDERS.length; i++) {
            const b = BAND_RENDERS[i];
            for (const [as, path, faces] of [
                ["label", bandRenderPath(i, b.label), bandFaces[i].label],
                ["badge", bandRenderPath(i, b.badge), bandFaces[i].badge],
            ]) {
                canvasRuns.push({
                    at: `${i}|${as}`, path: path || null,
                    // One face or nothing: a run that fell back has no single
                    // family to name and inkFaceFault's second arm is already
                    // reporting it.
                    family: faces && faces.length === 1 ? faces[0].family : null,
                });
            }
        }
        // And which face this browser gives the canary's own generic — at each
        // size the canary is actually asked at.
        //
        // See inkCanvasFallbackFault. Nothing already mounted is set in
        // INK_CANARY_FALLBACK — the whole grid is drawn in the theme's stack —
        // so the question needs a box, and the boxes are made here and removed
        // before the loop below reads a pixel.
        //
        // # One probe per request, not one probe
        //
        // The first version mounted a single span carrying `font-family` and
        // nothing else, so the generic resolved at whatever the page default
        // was. The runs are measured at their OWN computed size, weight and
        // style — that is what INK_FONT_SHORTHAND_JS assembles — and a family
        // list is allowed to answer per size: an optical cut, a display face at
        // the large end, a stack whose monospace entry only exists in one
        // width. Any of those leaves the probe naming a face no canary is put
        // to.
        //
        // So the requests are taken off the joined runs themselves and
        // deduplicated, and each gets a probe carrying that whole shorthand
        // with the generic as its family. The shorthand is built the same way
        // the canvas builds it — style, weight, size, family — because the
        // point is for the two to be asking one question.
        //
        // Mounted inside the bracketed window on purpose, and invisible to
        // both fingerprints by construction: TREE_FINGERPRINT_JS and
        // LAYOUT_FINGERPRINT_JS walk `[data-node-path]` and these spans carry
        // no such attribute, and `position:fixed` takes them out of flow so no
        // rect they do hash can move. That is an argument, and the two
        // fingerprints are also the thing that would report it wrong: a mount
        // that moved a rect fires faceLayoutFault, which drops every face arm
        // rather than believing this one.
        //
        // Off the left edge rather than `display:none` or `visibility:hidden`,
        // because a box that is not rendered has no glyphs and
        // CSS.getPlatformFontsForNode reports the faces that DREW something.
        const canaryReqs = await evaluate(`(() => {
            const paths = ${JSON.stringify(canvasRuns
                .filter((r) => r.family && r.path).map((r) => r.path))};
            const seen = [];
            for (const path of paths) {
                const e = document.querySelector('[data-node-path="' + path + '"]');
                if (!e) continue;
                const cs = getComputedStyle(e);
                // The head of INK_FONT_SHORTHAND_JS, which is the part that
                // selects a face before the family does. Through the shared
                // expression, because the measurement below reports the same
                // key and the two are joined on it. See INK_CANARY_REQ_JS.
                const req = (${INK_CANARY_REQ_JS})(cs);
                if (seen.includes(req)) continue;
                seen.push(req);
                const probe = document.createElement("span");
                probe.setAttribute("data-ink-canary", String(seen.length - 1));
                probe.style.cssText = "position:fixed;top:0;left:-9999px;font:" +
                    req + " ${INK_CANARY_FALLBACK}";
                probe.textContent = ${JSON.stringify(INK_CANARY_PROBE_TEXT)};
                document.body.appendChild(probe);
            }
            return seen;
        })()`);
        // Keyed by the request rather than by position, so a message can say
        // which size an answer is about — and so two requests that came back
        // with the same face collapse into one clause instead of reading as
        // two findings. See inkCanaryFaceList.
        const canaryFaces = [];
        for (let i = 0; i < (canaryReqs || []).length; i++) {
            const { nodeId } = await session.send("DOM.querySelector",
                { nodeId: domRoot, selector: `[data-ink-canary="${i}"]` });
            let faces = null;
            if (nodeId) {
                const { fonts } = await session.send(
                    "CSS.getPlatformFontsForNode", { nodeId });
                faces = fonts.map(
                    (f) => ({ family: f.familyName, glyphs: f.glyphCount }));
            }
            canaryFaces.push({ req: canaryReqs[i], faces });
        }
        if (canaryReqs && canaryReqs.length > 0) {
            // Removed here rather than left for the page teardown: every
            // reading after this one is about the grid, and spans nothing else
            // knows about are exactly the kind of thing a later check would
            // find and have no account of.
            await evaluate(`(() => {
                for (const e of document.querySelectorAll("[data-ink-canary]")) {
                    e.remove();
                }
                return true;
            })()`);
        }

        // And whether all of that was about the page the rects came from, and
        // about the rendering the capture holds.
        //
        // See TREE_FINGERPRINT_JS. The first fingerprint rode back inside the
        // rects' own evaluate; this is the same expression after the last
        // font read. A tree that moved in between makes every face above a
        // statement about some other document, so the face arm is dropped
        // rather than reported — an unheld claim is not evidence, and the one
        // thing it would produce is a confident diagnosis of a font
        // substitution that did not happen.
        //
        // The layout fingerprint rides back in the SAME evaluate, and closes
        // the window the other two leave between them: identity holds as far
        // as here, geometry was held as far as the shutter, and a font that
        // finished loading in between moves the rects without moving the tree.
        // See faceLayoutFault. One expression rather than two, because two
        // round trips would put a window between the two readings that close
        // the windows.
        const facePaths = BAND_RENDERS.length * 2 + INK_PROBES.length +
            INK_LIGATURES.length + (canaryReqs ? canaryReqs.length : 0);
        const afterFaces = await evaluate(`({
            tree: ${TREE_FINGERPRINT_JS}, layout: ${LAYOUT_FINGERPRINT_JS},
            canvas: ${JSON.stringify(canvasRuns)}.map((r) => {
                const e = r.path
                    ? document.querySelector('[data-node-path="' + r.path + '"]') : null;
                // Two ways of not asking, kept apart because they are
                // different findings. No element at a path the fingerprint in
                // this same expression says is still mounted is
                // inkCanvasFaceFault's first arm; no single family is a run
                // inkFaceFault has already reported, and that fault returns
                // before the arm can see it.
                if (!e) return { at: r.at, asked: false, missing: true };
                if (!r.family) return { at: r.at, asked: false };
                const cs = getComputedStyle(e);
                const cx = document.createElement("canvas").getContext("2d");
                const text = e.textContent;
                const family = JSON.stringify(r.family);
                // The shorthand band() measured with, and then the same
                // request with the compositor's own family in front of the
                // element's list.
                cx.font = (${INK_FONT_SHORTHAND_JS})(cs);
                const list = cx.font, asIs = cx.measureText(text).width;
                cx.font = (${INK_FONT_SHORTHAND_JS})(cs, family);
                // An assignment the canvas refuses leaves cx.font exactly as
                // it was, so an unchanged serialisation is the browser saying
                // it will not take this family — which is a different answer
                // from "the same face" and must not be read as one.
                if (cx.font === list) {
                    return { at: r.at, asked: false, unspellable: r.family };
                }
                const named = cx.measureText(text).width;
                // And the canary, which prices the join's one-sidedness. A head
                // family that resolves to no face is skipped and the two
                // advances above are equal for a reason that is not agreement
                // — see inkCanvasFaceFault. These two readings tell that apart:
                // a generic that always resolves, and the same generic with
                // this family in front of it. Equal means the family reached
                // nothing.
                //
                // Substituting is right HERE and wrong above: this asks what
                // the name resolves to, so falling back to the generic is the
                // answer rather than a wrong measurement.
                const fallback = ${JSON.stringify(INK_CANARY_FALLBACK)};
                cx.font = (${INK_FONT_SHORTHAND_JS})(cs, null, fallback);
                const bare = cx.font, base = cx.measureText(text).width;
                cx.font = (${INK_FONT_SHORTHAND_JS})(cs, null,
                    family + ", " + fallback);
                // The same refusal test as above: an unchanged serialisation
                // is a list the canvas would not take, and null rather than a
                // width so the arm that reads this cannot mistake a refusal
                // for an equality.
                const canary = cx.font === bare
                    ? null : cx.measureText(text).width;
                // And the request this run was measured under, plus the
                // family the protocol named it — the two halves of the pairing
                // inkCanaryAgreement makes. The family rides back rather than
                // being looked up from the table by its key, so the name in the
                // comparison is the name this very row was measured with.
                return { at: r.at, asked: true, text, family: r.family,
                    req: (${INK_CANARY_REQ_JS})(cs),
                    asIs, named, base, canary };
            }),
        })`);
        // What came back, by the same key the table was built with.
        const canvasRead = new Map(
            (afterFaces && Array.isArray(afterFaces.canvas) ? afterFaces.canvas : [])
                .map((r) => [r.at, r]));
        const canvasAt = (i, as) => canvasRead.get(`${i}|${as}`) || null;
        const treeFault = treeMovedFault(
            gridRead ? gridRead.tree : null, afterFaces ? afterFaces.tree : null,
            facePaths);
        if (treeFault) problems.push(treeFault);
        const faceLayout = faceLayoutFault(
            layoutAfter, afterFaces ? afterFaces.layout : null, facePaths);
        if (faceLayout) problems.push(faceLayout);
        // Every consultation of a face rests on both of those, so the pair is
        // one name — the way pixelsHeld is one name for the two questions the
        // sampling sites ask. A face read is held when the tree it was taken
        // over is the rects' tree AND the layout it was taken in is the
        // capture's layout; either one gone and what the read describes is a
        // rendering nothing else here measured.
        const facesHeld = !treeFault && !faceLayout;

        // What the refused pairs come to on the face this browser resolved.
        //
        // See inkLigatureCensus. Suppressed by the same two fingerprints and
        // for the same reason as every other face arm: a glyph count read off a
        // tree that moved is a count of some other page's glyphs, one read in a
        // layout the capture is not of is a count for a face the pixels do not
        // show, and a census is the one thing worse than a missing measurement
        // to state confidently.
        //
        // The families the grid's own runs were drawn by are collected here
        // rather than inside the census, because they are what the loops above
        // already produced — the census's job is to say whether the row was put
        // to one of them.
        const runFamilies = new Set();
        for (const f of [...bandFaces.flatMap((b) => [b.label, b.badge]),
                         ...probeFaces]) {
            if (f && f.length === 1) runFamilies.add(f[0].family);
        }
        let ligatures = { ligated: [], carried: [], family: null };
        if (facesHeld) {
            if (INK_LIGATURES.length === 0 && INK_PROBES.length > 0) {
                problems.push(`no ligature row came with the transcript, and ` +
                    `inkGlyphPerCharacter's refusal is a claim about faces in general ` +
                    `with nothing measuring it on this one. gen.go mounts one node per ` +
                    `pair in inkLigatureSeeds (see inkLigatureRow); an empty table is a ` +
                    `superset being carried without a reading`);
            }
            ligatures = inkLigatureCensus(ligatureFaces, runFamilies);
            for (const p of ligatures.problems) problems.push(p);
        }
        asked.ligatureSeeds = INK_LIGATURES.length;
        asked.ligated = ligatures.ligated;
        asked.ligatureFamily = ligatures.family;

        // Every consultation of a face goes through this, so the suppression is
        // one decision rather than a condition repeated at each call site.
        //
        // The census rides along because inkFaceFault's third arm is the one
        // the refusal exists to keep quiet, and a run that reaches it anyway is
        // a face joining a pair inkLigatureSeeds does not name. What that arm
        // could not say before is which — so it is handed the pairs this face
        // was measured to join, and says whether the string in front of it
        // holds one.
        const faceFault = (...args) =>
            facesHeld ? inkFaceFault(...args, ligatures.ligated) : null;

        // And the same suppression for the canvas join, which rests on exactly
        // the same two fingerprints: the family it names is one of the reads
        // they bracket, and the re-measure itself is taken in the evaluate that
        // closes them. See inkCanvasFaceFault.
        const canvasFault = (...args) =>
            facesHeld ? inkCanvasFaceFault(...args) : null;
        // How many runs were actually held to their compositor's face, for the
        // tail to recite instead of a constant. Counted off what came back
        // rather than incremented at the call site — a ledger that moves when
        // it is read is the shape bandTargetTally was split out of.
        const canvasAsked = facesHeld
            ? [...canvasRead.values()].filter((r) => r.asked) : [];
        asked.canvasFaces = canvasAsked.length;
        // And how many of those named a face the canvas could reach, which is
        // the number that means what the tail was saying. See the canary in
        // inkCanvasFaceFault: a head family that resolves to nothing leaves the
        // two advances equal without either of them having been about it.
        // And whether the generic that canary falls back to measures like the
        // faces this grid is drawn with, which is the other leg of the same
        // premise.
        //
        // Asked of the JOINED runs' own advances rather than of every face on
        // the page: the probe and ligature boxes have faces too and no canary
        // is put to them, so a generic that reached one of those costs this
        // reading nothing. Taken off the very rows the counts below are, so
        // the count and the premise cannot be about different populations —
        // and off their advances rather than their family names, because one
        // face reported under two spellings breaks the canary and passes a
        // name comparison. See inkCanvasFallbackFault.
        asked.canvasFallback = inkCanaryFaceList(canaryFaces);
        // And what those reads came to, which is the reading that says what
        // asking per request bought. See inkCanaryFaceSpread.
        const canarySpread = inkCanaryFaceSpread(canaryFaces);
        asked.canvasGenericReqs = canarySpread.mounted;
        asked.canvasGenericRead = canarySpread.read;
        asked.canvasGenericAnswers = canarySpread.answers;
        // And which axes of the request key those reads actually differ in,
        // which is the only thing that says whether a second read could have
        // answered differently from the first. See inkCanaryReqAxes: the
        // spread says the answers agreed and this says what they agreed
        // ACROSS, and an axis every request shares is one the agreement is
        // silent about however many probes were mounted.
        asked.canvasGenericAxes = inkCanaryReqAxes(canaryFaces);
        if (facesHeld) {
            const fallbackFault = inkCanvasFallbackFault(
                canaryFaces, canvasAsked, canvasAsked.length);
            if (fallbackFault) problems.push(fallbackFault);
            // And the two readings of that same premise against each other,
            // joined on the request each was taken at. See inkCanaryAgreement:
            // the direction fallbackFault decides is two names over one
            // advance, and this is the other one — one name over two advances,
            // which nothing was looking at.
            const agreement = inkCanaryAgreement(canaryFaces, canvasAsked);
            asked.canvasGenericPaired = agreement.asked;
            asked.canvasGenericAgreed = agreement.agreed;
            asked.canvasGenericMulti = agreement.multi;
            asked.canvasGenericUnread = agreement.unread;
            asked.canvasGenericUnjoined = agreement.unjoined;
            if (agreement.fault) problems.push(agreement.fault);
        }
        // And the premise as a number, for the tail to recite in place of the
        // name it used to assert something about. Through the same helper the
        // arm above decides with, so the sentence and the check cannot come
        // apart. Null when no run produced the pair, which is the state that
        // arm reports rather than a separation of zero.
        const canvasGaps = canvasAsked
            .map(inkCanvasGenericGap).filter((g) => g !== null);
        asked.canvasGenericNearest = canvasGaps.length === 0 ? null
            : canvasGaps.reduce((w, g) => Math.min(w, g), Infinity);
        const canvasReached = canvasAsked.filter(inkCanvasNameReached);
        asked.canvasNamed = canvasReached.length;
        // And how much room the reading behind that count had. Taken off the
        // same rows and through the same predicate, so the census below cannot
        // be about a population the counter is not — see INK_CANARY_MARGIN,
        // where the bound being bracketed is inkCanvasNameReached's own.
        asked.canvasNarrowest = canvasReached.length === 0 ? null
            : canvasReached.reduce(
                (w, r) => Math.min(w, Math.abs(r.canary - r.base)), Infinity);
        // And what the bound the join is asked under actually costs on this
        // run, kept for the check below rather than derived there — the widths
        // are here and the census is six hundred lines down.
        asked.canvasWidest = canvasAsked.reduce(
            (w, r) => Math.max(w, Math.abs(r.asIs - r.named)), 0);

        // Whether the grid as a whole is on the screen, asked once.
        //
        // Both of the guards below ask about the box they are about to sample,
        // which is right and is not the whole question. Every band and every
        // probe in this check is a child of one mounted Column, and a window
        // too short for it clips a run of them at once: the last band's fold
        // guard fires, and so does every probe's, because the probe Row mounts
        // after all twenty of them. Twenty-one messages about twenty-one boxes,
        // naming the same knob twenty-one times, and the one line that says
        // what actually happened — the grid does not fit — is not among them.
        //
        // So the grid's own box is measured first. When it is past the bottom,
        // this is the only thing said about the viewport and nothing below
        // reads a pixel: the geometry the rest of the check makes its claims
        // about is a layout, and a layout is the same whether or not the
        // screenshot reached it.
        //
        // The container's own rect rather than the lowest child's, so the two
        // questions stay independent: a child that overflowed its container
        // would be past the fold with the grid still fitting, and the per-box
        // guards are what would catch it.
        let gridClipped = false;
        if (!gridRead || !gridRead.grid) {
            problems.push(`the band render grid mounts as one Column at "root" and ` +
                `nothing is at that path, so whether the grid as a whole is on the ` +
                `screen could not be asked. Every band and probe below is a child of ` +
                `it, and a short window clips them together`);
        } else {
            const gridFold = foldVerdict({
                what: "the band render grid",
                bottom: gridRead.grid.y + gridRead.grid.h,
                screen: bandImg.height / bandDpr,
                knob: "the window size, or the number of themes and shapes the " +
                    "grid mounts",
            });
            if (gridFold) {
                gridClipped = true;
                problems.push(`${gridFold}

` +
                    `Asked of the grid rather than of the boxes in it, so this is one ` +
                    `message and not ${BAND_RENDERS.length + INK_PROBES.length}: every ` +
                    `band and every antialiasing probe here is a child of that Column, ` +
                    `and a window this short clips a run of them at once — the probe Row ` +
                    `last of all, because it mounts after all ${BAND_RENDERS.length} ` +
                    `bands. Nothing below this line read a pixel; the layout assertions ` +
                    `still ran, because a rect is the same whether or not the capture ` +
                    `reached it`);
            }
        }

        // Whether a pixel read below is a statement about the box the rect
        // names, which is two independent questions with one answer.
        //
        // The grid can be past the bottom of the window, in which case the
        // capture has no pixels for it; and the page can have relaid out
        // between the rects and the capture, in which case it has pixels for
        // boxes that are no longer where the rects say. Both leave every
        // sample below reading something that is not what it is reported as,
        // and neither touches the layout assertions — a rect is the same
        // whether or not the capture reached it, and the rects are the FIRST
        // side of the pair that came apart.
        //
        // One name for the pair so the suppression is one decision rather than
        // a condition repeated at each sampling site, the way faceFault is one
        // decision for the reads the tree fingerprint holds.
        const pixelsHeld = !gridClipped && !layoutFault;

        // The rendering mode every ink assertion below rests on. See
        // INK_PROBE_GRID for the argument; here it is one pass per probe.
        //
        // Read before any band, and keyed so each band asks about ITS OWN
        // declaration rather than about the page: subpixel antialiasing puts
        // ordinary text off offSegment's line, so with this unasked the failure
        // would be twenty bands each naming a third colour in a box, and the
        // cause is not a colour at all.
        //
        // A probe that came back coloured suppresses the scans that rest on it
        // and nothing else. That is the whole reason the probes are per
        // declaration: a theme whose ink moved onto another rendering path used
        // to invalidate the verdict for every band in the grid, and now
        // invalidates the boxes that share its declaration.
        // Before any of it: the fractions the rows are taken at have to be
        // fractions of the band. See inkRowsFault — with the coordinate system
        // moved onto the measured band, this is what inkBandFault used to be.
        const rowsFault = inkRowsFault();
        if (rowsFault) problems.push(rowsFault);

        const probeVerdict = new Map();
        // And what each probe's own text element resolves, kept so the boxes it
        // answers for can be compared with it. See inkOwnFault.
        //
        // Populated here rather than inside the scan below, because it is a
        // fact about the element and not about the capture: a computed style is
        // the same whether or not the screenshot reached the box. That matters
        // for the permission census, which has to see every pair in the grid
        // even on a run where a short window suppressed all the pixel reads.
        const probeSelf = new Map();
        for (let i = 0; i < INK_PROBES.length; i++) {
            probeSelf.set(INK_PROBES[i].key, {
                what: INK_PROBES[i].what,
                own: probeReads[i] ? probeReads[i].probeOwn : null,
                // The face the compositor actually drew this probe's glyphs
                // with, and the string it drew — see inkFaceFault. A probe
                // whose greys are another face's greys is a measurement of
                // another face's antialiasing.
                faces: probeFaces[i],
                text: probeReads[i] && probeReads[i].probeSubject
                    ? probeReads[i].probeSubject.text : null,
            });
        }
        if (INK_PROBES.length === 0) {
            problems.push(`no antialiasing probes came with the transcript, and every ` +
                `ink assertion below rests on one. gen.go builds a probe per text ` +
                `declaration the scan reads (see inkProbe); an empty table is a scan ` +
                `about to trust a rendering mode nothing measured`);
        }

        // Which of INK_OWN_MAY_DIFFER's permissions this grid actually spends.
        //
        // inkOwnFault holds every scanned box to differing from its probe
        // NOWHERE outside that table, and a claim of that shape is satisfied by
        // a pair that differs nowhere at all. So a permission no box in the
        // grid needs is slack in the list: it would go on licensing a
        // difference nobody has seen, and the day something started resolving
        // it differently for a reason that is not the one written beside it,
        // this check would stay silent. That is item 7's argument, applied to a
        // struct a browser owns rather than one this repository does.
        //
        // Asked once per run and over every (box, probe) pair the transcript
        // names, independently of the fold and of every other guard: a computed
        // style is the same whether or not the screenshot reached the box, and
        // a pair whose scan some other failure suppressed still resolved the
        // properties this is a census of.
        const inkOwnSpent = new Set();
        let inkOwnPairs = 0;
        // How wide this browser's enumeration actually is, kept so the census
        // below can say it. See INK_OWN_MEASURED_ON: a permission that has
        // stopped being reachable and a browser that has stopped exposing the
        // property are two different findings with one message.
        let inkOwnRead = 0;
        for (let i = 0; i < BAND_RENDERS.length; i++) {
            const b = BAND_RENDERS[i], r = renderRects[i];
            if (!r) continue;
            for (const [own, key] of [[r.labelOwn, b.labelProbe],
                                      [r.badgeOwn, b.badgeProbe]]) {
                const self = probeSelf.get(key);
                if (!own || !self || !self.own) continue;
                inkOwnPairs++;
                const diff = inkOwnDiff(own, self.own);
                inkOwnRead = Math.max(inkOwnRead, diff.read);
                for (const prop of diff.used) inkOwnSpent.add(prop);
            }
        }
        if (INK_PROBES.length > 0 && inkOwnPairs === 0) {
            problems.push(`not one of the ${BAND_RENDERS.length} bands in this grid ` +
                `could be compared with the probe that answers for it — either the ` +
                `scanned boxes or the probes' own text nodes resolved nothing. The ` +
                `whole of what makes a probe's grey box a statement about a band is ` +
                `that the two elements resolve the same CSS, and with no pair read that ` +
                `is unasked rather than true`);
        } else {
            const unused = Object.keys(INK_OWN_MAY_DIFFER)
                .filter((prop) => !inkOwnSpent.has(prop));
            if (unused.length > 0) {
                problems.push(`INK_OWN_MAY_DIFFER permits ` +
                    `${Object.keys(INK_OWN_MAY_DIFFER).length} properties to differ ` +
                    `between a scanned box and its antialiasing probe, and ` +
                    `${unused.length} of them differ nowhere in the ${inkOwnPairs} ` +
                    `pairs this grid measured: ${unused.join(", ")}.

` +
                    `The reason on file for ${unused[0]} is that ` +
                    `${INK_OWN_MAY_DIFFER[unused[0]]} — and no pair in this grid needs ` +
                    `it. An unspent permission is slack: "the two differ only where ` +
                    `permitted" is satisfied by a pair that differs nowhere, so a ` +
                    `permission nothing exercises licenses a difference nobody has ` +
                    `seen. Either the grid stopped mounting the case that earned it ` +
                    `(the count pill is what earns background-color, and it is the only ` +
                    `thing that does), or the reason beside it has stopped being true ` +
                    `and the entry belongs out of the table` +
                    inkOwnBuildNote(inkOwnRead, browserBuild));
            }
        }
        asked.ownRead = inkOwnRead;

        // And whether a computed style read on THIS build still looks like one
        // to the guard that has to recognise it.
        //
        // inkReadShape counts string values against INK_OWN_SHAPE_FLOOR, which
        // is derived from the build INK_OWN_MAY_DIFFER was measured on and not
        // from the build in front of it. What this run enumerates is
        // inkOwnRead, and it is only in hand here — the guard runs with the
        // rects, long before any pair has been compared — so the derivation is
        // what the guard uses and this is where the derivation is checked
        // against the measurement.
        //
        // Under the floor, inkDeclarationGuard's third question goes quiet
        // rather than wrong: a hand-written `out.ownStyle` written around the
        // `declarations` table would be shaped like nothing at all, the
        // manifest check would find nothing missing, and the hole that guard
        // was rewritten to close would be open again with every message
        // silent.
        // The lower edge of the same bracket, which is not a measurement of
        // the browser and is a property of the derivation: a floor at or under
        // a subject read's key count would call one a computed style the moment
        // inkReadShape's own subject arm stopped matching — a fifth key added
        // to the subject read is all that takes — and the manifest check would
        // then find a declaration answer where a subject sits. Reported here
        // rather than beside the constant so a reader meets the floor's two
        // edges in one place: what it must stay under, and what it must stay
        // above.
        if (INK_OWN_SHAPE_FLOOR <= INK_SUBJECT_KEYS.length) {
            problems.push(`inkReadShape calls anything with more than ` +
                `${INK_OWN_SHAPE_FLOOR} string values a computed style, and a subject ` +
                `read has ${INK_SUBJECT_KEYS.length} keys.

` +
                `The floor is a quarter of the ${INK_OWN_MEASURED_ON.props} ` +
                `properties ${INK_OWN_MEASURED_ON.browser} enumerates, and it has to ` +
                `sit between two populations: above the narrowest object the shape ` +
                `test must reject, which is a subject read, and below the widest one ` +
                `it must recognise, which is a computed style. It has come down to ` +
                `the reject population. inkReadShape's own subject arm is what keeps ` +
                `that from mattering today — it matches on the four key NAMES first — ` +
                `so this is a bracket that has closed while every message stayed ` +
                `silent, and the next key added to a subject read opens it`);
        }
        if (inkOwnRead > 0 && inkOwnRead <= INK_OWN_SHAPE_FLOOR) {
            problems.push(`this browser enumerates ${inkOwnRead} computed properties ` +
                `and inkReadShape calls anything with more than ` +
                `${INK_OWN_SHAPE_FLOOR} string values a computed style.

` +
                `That floor is a quarter of the ${INK_OWN_MEASURED_ON.props} ` +
                `${INK_OWN_MEASURED_ON.browser} enumerated — it has to sit above the ` +
                `${INK_SUBJECT_KEYS.length} keys of a subject read and below what a ` +
                `computed style comes to — and this build is under ` +
                `it, so a declaration read taken here is not shaped like one to ` +
                `inkDeclarationGuard. That guard would go on passing while a read ` +
                `written around the \`declarations\` table went unseen, which is the ` +
                `hole it exists to close: the manifest says what was asked, and the ` +
                `shape test is the half that notices what was asked without being ` +
                `declared` +
                inkOwnBuildNote(inkOwnRead, browserBuild));
        }
        for (let i = 0; pixelsHeld && i < INK_PROBES.length; i++) {
            const p = INK_PROBES[i], rect = probeRects[i];
            if (!rect || rect.w === 0) {
                problems.push(`the antialiasing probe for ${p.what} was not laid out. ` +
                    `The probes mount as one Row at the end of the band grid, and every ` +
                    `ink assertion over those boxes is resting on the rendering mode it ` +
                    `exists to read`);
                probeVerdict.set(p.key, "not laid out");
                continue;
            }
            // The declared width, held to. The probes share one Row and each
            // refuses to shrink, so a rect narrower than its declaration is a
            // Row that ran out of width — and a probe squeezed to a sliver
            // reports a grey page because almost none of it was read.
            if (!bandRenderSame(rect.w, p.width)) {
                problems.push(`the antialiasing probe for ${p.what} is ` +
                    `${rect.w.toFixed(2)}px wide and declares ${p.width}px. The probes ` +
                    `are laid out side by side in one Row to keep the grid above the ` +
                    `fold, each with core.ShrinkNone; a narrower rect means the Row ran ` +
                    `out of room and the probe is being read across a sliver of itself, ` +
                    `which reports "no fringe" for want of pixels`);
                probeVerdict.set(p.key, "squeezed");
                continue;
            }
            // And the other way a probe can have a rect and no pixels, which
            // until this line was the one that reported nothing at all.
            //
            // The bands each run foldVerdict before they are sampled; the probe
            // Row mounts after all twenty of them and was checked only for its
            // width. A viewport that fit every band and clipped the probes was
            // not the "not laid out" case above — a box past the bottom of the
            // window has a perfectly good rect — it was the pixel loop below
            // reading null for every sample, leaving `worst` at 0 and `fringe`
            // at null, and passing. Eleven probes silently measuring nothing,
            // under twenty bands whose ink readings all rest on them.
            //
            // So the same guard the bands get, with the same knob named: the
            // grid's height is what pushed the probes off, and the window size
            // is what a reader turns.
            const probeFold = foldVerdict({
                what: `the antialiasing probe for ${p.what}`,
                bottom: rect.y + rect.h,
                screen: bandImg.height / bandDpr,
                knob: "the window size, or the number of themes and shapes the " +
                    "grid mounts",
            });
            if (probeFold) {
                problems.push(probeFold);
                probeVerdict.set(p.key, "past the bottom of the viewport");
                continue;
            }
            // What is above the probe, which is the half of the rendering
            // question its own Style cannot carry. See INK_PATH_PROPS: a probe
            // under a composited ancestor is measuring a path the bands are not
            // on, and its verdict stops being about them.
            //
            // Swept from the probe's text node (inkProbeTextPath), so the white
            // Box that gen.go wraps it in is one of the ancestors measured. It
            // used to be the starting point instead, which left it neutral by
            // construction: a compositing property arriving on that Box — from
            // a runtime mapping of Background, Width or FlexShrink that nobody
            // here has thought about — sat between the probe's glyphs and every
            // box this sweep looked at, and nothing asked.
            //
            // And before that, that the two paths this probe uses are the two
            // elements they are supposed to be: a white Box holding exactly
            // one Text, sampled at the Box and asked about at the Text. See
            // inkSubjectFault for the confusion this pair is the fix for — it
            // is the probes' own, and holding it here is what keeps the "/0"
            // in inkProbeTextPath a derivation from the shape gen.go builds
            // rather than an index that happens to land on the right child.
            const probeShape = inkProbeBoxFault(
                `the antialiasing probe for ${p.what}`,
                probeReads[i].probeBoxSubject) ||
                inkSubjectFault(`the antialiasing probe for ${p.what}`,
                    "the probe's glyphs", probeReads[i].probeSubject);
            if (probeShape) {
                problems.push(probeShape);
                probeVerdict.set(p.key, "a shape inkProbeFor does not build");
                continue;
            }
            const probeAncestry = inkAncestryFault(
                `the antialiasing probe for ${p.what}`, "the probe's glyphs",
                probeReads[i].probeAncestry);
            if (probeAncestry) {
                problems.push(probeAncestry);
                probeVerdict.set(p.key, "an ancestor that decides how glyphs are drawn");
                continue;
            }
            // And the face this probe's own glyphs were drawn with, held to
            // being one face drawing a glyph per character. The boxes it
            // answers for are compared against it (inkFaceFault's last arm);
            // this is the arm that says the probe's own edges are a single
            // face's to begin with, which is what makes that comparison mean
            // anything.
            const probeFace = faceFault(
                `the antialiasing probe for ${p.what}`, "the probe's glyphs", p.what,
                probeFaces[i], probeReads[i].probeSubject
                    ? probeReads[i].probeSubject.text : null, null);
            if (probeFace) {
                problems.push(probeFace);
                probeVerdict.set(p.key, "a face the ink scan cannot rest on");
                continue;
            }
            let worst = 0, fringe = null;

            for (let y = 0; y < Math.round(rect.h * bandDpr); y++) {
                for (let x = 0; x < Math.round(rect.w * bandDpr); x++) {
                    const got = pixelAt(bandImg,
                        Math.round(rect.x * bandDpr) + x,
                        Math.round(rect.y * bandDpr) + y);
                    if (got === null) continue;
                    const ch = [1, 3, 5].map((i) => parseInt(got.slice(i, i + 2), 16));
                    const spread = Math.max(...ch) - Math.min(...ch);
                    // The first pixel too, so a probe that came back perfectly
                    // grey still names one: the floor message below is about a
                    // spread of zero, and "the pixel null" is not a reading.
                    if (fringe === null || spread > worst) { worst = spread; fringe = got; }
                }
            }
            if (fringe === null) {
                // Every sample came back null with the rect on screen and at
                // its declared width, which pixelAt only answers when the
                // capture itself is short of the rect. Reported rather than
                // read as "no fringe": a probe nobody could look at is not a
                // probe that came back grey.
                problems.push(`the antialiasing probe for ${p.what} lies at ` +
                    `${rect.x.toFixed(0)},${rect.y.toFixed(0)} and not one of its ` +
                    `${Math.round(rect.w * bandDpr)}×${Math.round(rect.h * bandDpr)} ` +
                    `device pixels is in the screenshot, which is ` +
                    `${bandImg.width}×${bandImg.height}. The rect is on screen and at ` +
                    `its declared width, so this is the capture and not the layout — ` +
                    `and with no pixel read, "no coloured fringe" is a statement about ` +
                    `nothing`);
                probeVerdict.set(p.key, "no pixel of it was in the screenshot");
            } else if (worst > SUBPIXEL_EPSILON) {
                probeVerdict.set(p.key, `the pixel ${fringe}, ${worst} channels apart`);
                problems.push(`the antialiasing probe for ${p.what} contains the pixel ` +
                    `${fringe}, whose channels are ${worst} apart. That probe paints ` +
                    `those declarations' own face and weight in black at their own ` +
                    `alpha on white, and every blend of two greys is a grey — so a ` +
                    `coloured pixel in it is LCD subpixel antialiasing: each channel ` +
                    `gets its own coverage.\n\n` +
                    `That is the arithmetic the band ink scan is built on. offSegment ` +
                    `holds every pixel in a label's box to the line between the fill ` +
                    `and the ink on the argument that coverage is one scalar, and ` +
                    `INK_EPSILON's exact match on a stem's interior rests on the same ` +
                    `thing. With subpixel rendering on, ordinary correct text is off ` +
                    `that line and every box this probe answers for would report a ` +
                    `third colour. ` +
                    `Headless Chrome disables it; something about this browser, its ` +
                    `flags, or that declaration has changed.`);
            } else if (worst > SUBPIXEL_FLOOR) {
                // Below the ceiling and above the floor: still grey, and no
                // longer grey for the reason written down. See SUBPIXEL_FLOOR.
                problems.push(`the antialiasing probe for ${p.what} has a worst pixel ` +
                    `of ${fringe}, whose channels are ${worst} apart. That is under ` +
                    `SUBPIXEL_EPSILON (${SUBPIXEL_EPSILON}) so this is not subpixel ` +
                    `rendering — and it is over the ${SUBPIXEL_FLOOR} the arithmetic ` +
                    `says a grayscale blend of two greys produces, which is what these ` +
                    `probes have measured every time they have run.\n\n` +
                    `The tolerance is documented as absorbing "the browser's eight-bit ` +
                    `rounding", and there is nothing there for it to absorb: the three ` +
                    `channels of a grey enter that rounding as one value and leave it ` +
                    `as one. So the whole of SUBPIXEL_EPSILON is margin, and this is ` +
                    `the margin being spent — something is now colouring a grey ` +
                    `slightly, and the next channel of drift would fail as an ` +
                    `antialiasing verdict for a cause that is not antialiasing.`);
            }
        }

        // Whether the scan of one box may be believed, and if not, why.
        //
        // Returns null when the probe that answers for this declaration came
        // back grey. Anything else is a reason to say nothing about the box:
        // reporting "a third colour in the label" under a rendering mode that
        // puts one there would be the failure this whole apparatus exists to
        // keep from being reported as a palette fault.
        const inkUnreadable = (where, subject, key, ancestry, own, read, faces,
                               canvas) => {
            // First, whether the two questions below are even being asked of
            // the right element. Both of them — the chain above the glyphs and
            // the declaration the probe copies — are about the box that PAINTS
            // the run, and a path that names a container answers them about
            // the container. See inkSubjectFault; this is asked before either
            // because a wrong element makes both answers wrong quietly.
            const wrongElement = inkSubjectFault(where, subject, read);
            if (wrongElement) return wrongElement;
            // The scanned box's own ancestry, asked before the probe's verdict
            // is consulted: a band under a composited ancestor is a box the
            // probe never answered for, whatever the probe measured. See
            // INK_PATH_PROPS.
            const above = inkAncestryFault(where, subject, ancestry);
            if (above) return above;
            if (!key) {
                return `${where}: the transcript names no antialiasing probe for ` +
                    `${subject}, and the ink assertions over that box rest on one. ` +
                    `gen.go builds a probe per text declaration the scan reads; a case ` +
                    `with none is a case whose rendering mode nothing measured`;
            }
            const verdict = probeVerdict.get(key);
            if (verdict) {
                return `${where}: the antialiasing probe for ${subject} reported ` +
                    `${verdict}, so nothing was read from that box. offSegment and ` +
                    `INK_EPSILON are both grayscale antialiasing's arithmetic, and under ` +
                    `any other rendering mode a correct box fails them`;
            }
            // And last, the assumption the ancestry sweep rests on: that the
            // probe and this box resolve the same rendering path on their own
            // two elements, because one was built from the other's Style. A
            // Style is not what a browser resolves. See inkOwnFault.
            const self = probeSelf.get(key);
            const declaration = inkOwnFault(where, subject, self ? self.what : key, own,
                self ? self.own : null, browserBuild);
            if (declaration) return declaration;
            // And last of all, which face the compositor reached for. Last
            // because it is the only one of these read outside the page: the
            // three above are about what the document says, and this is about
            // what the font stack resolved the document's request to. See
            // inkFaceFault.
            const resolved = faceFault(where, subject, self ? self.what : key, faces,
                read ? read.text : null, self ? self.faces : null);
            if (resolved) return resolved;
            // And after that, the fourth reader: the canvas the ink band and
            // the run advance came off, held to the face the compositor just
            // named. Asked last of all because it is a statement about ONE
            // face, and the arm above is what makes there be one. See
            // inkCanvasFaceFault — a canvas on another face makes the three
            // rows fractions of another face's band, so this suppresses the
            // scan the way the other three do rather than reporting beside it.
            return canvasFault(where, subject, faces, canvas);
        };

        // What a scan with no edge clearance would be reading, one box at a
        // time, so the floor those scans were just held to can be held to
        // separating something. See INK_ROW_ROUNDING and inkRoundingVerdict.
        const inkOutsideBand = [];

        // What the tail of this check is allowed to recite.
        //
        // # A pass for a question nobody asked
        //
        // The OK line at the bottom of this file says twenty real bands paint
        // their words in their own ink and their counts in digits inside their
        // pills. Both numbers came from BAND_RENDERS.length — the size of the
        // table gen.go sent — and neither came from the work. Every path that
        // suppresses a scan does push a problem, so the line is never printed
        // over a suppression today; what it means is that the counts are true
        // by a property of the code around them rather than by having been
        // counted, and a skip added later that pushed nothing would leave the
        // recitation intact.
        //
        // There is already one such skip. `if (r.label && ...)` drops the whole
        // ink scan for a band whose label node the mount does not have, and
        // unlike the badge and the heading wrapper a step above, nothing
        // compares that against what gen.go declared. A label that vanished
        // took twenty bands' worth of recitation with it and said nothing.
        //
        // So the two are counted where they are actually asked, held against
        // what the transcript declared, and the tail recites the count rather
        // than the constant. The gap this closes is item 4's — a run that
        // reports "one message and a PASS for everything rect-shaped in the
        // same breath" now says how much of the pass was asked.
        //
        // # The three the census left behind
        //
        // Closing that gap for the words and the counts left the other three
        // claims in the same sentence still reciting BAND_RENDERS.length: the
        // tap target, the declared inset and the band's own fill. Two of them
        // are rect assertions that genuinely do run for every band, and the
        // argument for leaving them was that their skips are loud.
        //
        // The fill's is not: it is sampled inside `if (pixelsHeld)` and
        // skipped with every other pixel read, so a run on a short window
        // recited twenty painted fills having read none. That is exactly the
        // shape of the label's skip, which had the same argument until it
        // turned out not to hold — so all five are counted now, and the
        // argument is retired rather than reapplied.
        //
        // A rect claim's count is over the bands that got as far as being
        // measured; the inset's is over the bands whose leading child gen.go
        // names, because a band with none is a band the claim is not about and
        // says so with its own message.
        //
        // # The tap target is three claims, and the census reports three
        //
        // `targets` counted the CONJUNCTION of the leading edge, the trailing
        // edge and the disclosure branch's wrapper stretch, on the argument
        // that any one of them failing means the target did not span the band.
        // That argument is right about the tail's sentence and wrong about the
        // census: a band that failed the stretch and a band whose leading edge
        // moved decremented one number, so the line said how many bands lost
        // the claim without saying which of the three cost them — and it could
        // not tell a run where one band lost all three from a run where three
        // bands lost one each.
        //
        // So the three are counted separately and the census reports those;
        // `targets` stays because it is the number the tail is allowed to
        // recite, and it is not censused itself, because a shortfall in it is
        // a shortfall in one of the three and the three say which.
        //
        // The stretch's population is the bands that HAVE a wrapper. It is the
        // disclosure branch's mechanism and the plain branch has no equivalent
        // — a claim counted over twenty bands when eight can make it is a
        // census line that fails on every run.
        // The three the tap-target claim is made of, counted by the module that
        // owns the parts. See bandTargetCensus: the ternary that used to pick a
        // population here was a second spelling of the predicate bandTargetRead
        // decides `wanted` with, and the census line's phrase was a third — so
        // a row with the wrong `everyBand` produced a shortfall on every run
        // and no message saying which row it was. The predicate that says a
        // band is on the disclosure branch stays here, because a band record's
        // shape is this file's.
        const targetCensus = bandTargetCensus(BAND_RENDERS, (b) => Boolean(b.wrapper));
        const declared = {
            ...Object.fromEntries(targetCensus.map((r) => [r.counter, r.declared])),
            insets: BAND_RENDERS.filter((b) => b.leading).length,
            fills: BAND_RENDERS.length,
            words: BAND_RENDERS.filter((b) => b.label).length,
            counts: BAND_RENDERS.filter((b) => b.badge).length,
        };

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
            // The same question about the label, which nothing had asked. The
            // ink scan below is guarded on `r.label` alone, so a band whose
            // label node the mount does not have skipped every reading of its
            // words in silence — see `asked` for what that cost the tail.
            if (Boolean(r.label) !== Boolean(b.label)) {
                problems.push(`${where}: gen.go ${b.label ? "found" : "found no"} label ` +
                    `and the mount ${r.label ? "has" : "has no"} node at that path. ` +
                    `Everything this check says about a band's words is guarded on that ` +
                    `node being there, so the two disagreeing is a run of assertions ` +
                    `that would not have been made and would not have been missed`);
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
            //
            // Both of these are pixels, so both are asked only while the grid
            // as a whole is on the screen: the fold guard above says that once,
            // for every band and probe together, rather than once per box.
            if (pixelsHeld) {
                const bandFold = foldVerdict({
                    what: where, bottom: r.band.y + r.band.h,
                    screen: bandImg.height / bandDpr,
                    knob: "the window size, or the number of themes and shapes the " +
                        "grid mounts",
                });
                if (bandFold) {
                    problems.push(bandFold);
                    continue;
                }
                const bandFill = pixelAt(bandImg,
                    (r.band.x + 6) * bandDpr, (r.band.y + r.band.h / 2) * bandDpr);
                if (bandFill === b.fill) {
                    // Counted here, on the one path that actually read the
                    // pixel. Everything above this line is a rect.
                    asked.fills++;
                }
                if (bandFill !== b.fill) {
                    problems.push(`${where}: the band painted as ${bandFill}, and its ` +
                        `own Style declares ${b.fill} (the page behind it is ` +
                        `${b.page}). A band's fill is why it may span its container edge ` +
                        `to edge rather than being inset like a row, and it is the one ` +
                        `thing in the recipe that every rect below is blind to — this ` +
                        `grid measured nine real bands and read no pixels at all until ` +
                        `this line`);
                    continue;
                }
            }

            // The words, which the fill alone says nothing about.
            //
            // A band is a fill, a run of words and a count, and until this line
            // two of the three were unread: a band painting its own fill over
            // an invisible label passed every assertion here. That is the gap
            // the widget grid does not have — it reads a fill AND scans both
            // boundary edges, on the argument that either single edge is
            // consistent with a correct frame.
            //
            // Text cannot be sampled at a point the way a fill can: a glyph is
            // a few stems in a field of backdrop, and where they land depends
            // on the font. So the label's rect is scanned and the two things
            // that could be wrong are asked separately:
            //
            //	there is ink     some pixel in the label's box is not the band's
            //	                 fill. A label rendered in the fill colour, or
            //	                 not rendered at all, has none.
            //	it is THIS ink   some pixel is exactly the colour the widget
            //	                 declares. A stem's interior is unblended at any
            //	                 size a caption is set at, so the declared colour
            //	                 is present when it is the colour being used —
            //	                 and a label drawn in the theme's other ink role
            //	                 would satisfy the first claim and fail this one.
            //
            // Scanned across three rows taken as fractions of the band this run
            // has ink in — from its baseline up by an x-height, measured off
            // this element's own inline text box. See INK_ROWS for why one row
            // was not enough and why the fractions are of the band rather than
            // of the line box, and inkRows for what has to be true of that band
            // before three fractions of it are three rows of pixels.
            //
            // And every pixel the scan touches is held to being a blend of the
            // two colours the band declares for that box — see offSegment,
            // which is confusableInk's claim asked of the capture rather than
            // of the declaration.
            if (r.label && pixelsHeld) {
                // What the ink is supposed to LOOK like, which is not always
                // what it is declared as. DefaultTheme's TextSecondary is
                // #3C3C4399 — eight digits, so the words are drawn at 60%
                // opacity over the band and a screenshot holds the composite.
                // Ignoring the alpha would compare against a colour nothing
                // paints; reading it and compositing is the same arithmetic a
                // browser does, and it is worth doing rather than falling back
                // to "some pixel differs from the fill" because the theme with
                // the translucent ink is the one where that fallback would be
                // the whole check.
                const want = over(b.labelInk, b.fill);

                // Before any pixel is read: the tolerance this scan compares
                // with is only meaningful while the composite is far from
                // every other colour the band paints. That was a sentence
                // beside INK_EPSILON and is a check now — a case where the ink
                // has drifted to within a few channels of the fill, the pill
                // or the page is one where "it is THIS ink" agrees with two
                // answers, and a check that cannot fail must not report a
                // pass.
                // And the rendering mode this box's own declaration resolves
                // to, which is what makes every reading below arithmetic
                // rather than a guess. See inkUnreadable.
                const unreadable = inkUnreadable(where, "the label's words", b.labelProbe,
                    r.labelAncestry, r.labelOwn, r.labelSubject, bandFaces[i].label,
                    canvasAt(i, "label"));
                const confusable = unreadable ? null : confusableInk(want, b);
                if (unreadable) {
                    problems.push(unreadable);
                } else if (confusable) {
                    problems.push(`${where}: the label's ink ${b.labelInk} composites ` +
                        `over this band to ${want}, which is ${confusable.distance} ` +
                        `channels from ${confusable.color} — ${confusable.what}. The ink ` +
                        `scan below accepts a pixel within ${INK_EPSILON} of the ` +
                        `composite, a tolerance that exists to absorb this file's ` +
                        `eight-bit blend and the browser's own rounding, and it is only ` +
                        `worth anything while the colours it tells apart are an order ` +
                        `further apart than that (${INK_EPSILON * INK_MARGIN}). A theme ` +
                        `this close is one where the words are not readable either`);
                } else {
                    // The rows, which are a fraction of this element's own
                    // measured ink band rather than of its line box — so
                    // "inside the band" is true by construction and what is
                    // left to check is that the band was measurable at all and
                    // is tall enough for three fractions of it to be three rows
                    // of pixels. See INK_ROWS for the coordinate system and for
                    // the theme that moved it, and inkBandRows for the four
                    // ways a band can be unusable.
                    const band = inkBandRows(where, "the label's words",
                        r.labelBand, bandDpr,
                        `A word set in lower case has ink only between its baseline ` +
                        `and its x-height, so that band is all there is to spread ` +
                        `three rows across`);
                    if (band.problem) {
                        problems.push(band.problem);
                        continue;
                    }
                    const rows = band.rows;

                    // The run's own rect, and then everything else in the
                    // box.
                    //
                    // # Two halves that were looking at different rects
                    //
                    // The rows above are measured for the RUN — inkBandRows
                    // takes the inline text box's own top and the resolved
                    // face's x-height — and the scan swept `label.x` to
                    // `label.x + label.w`, which on the stretched branches is
                    // several times the width of the words. Every column past
                    // the run is backdrop: it answers notFill false, it sits at
                    // t=0 on the segment, and it says nothing either way. So
                    // the measured half and the read half were two rects, and
                    // only one of them had been measured.
                    //
                    // Narrowing to the run alone would have been the other
                    // fault — measuring where the glyphs are with a number
                    // taken from where the glyphs are — so the box is not
                    // dropped. It is SAID instead: the run is scanned for the
                    // three verdicts, and the surplus on either side of it is
                    // held to being nothing but the band's own fill. That is a
                    // claim neither half made before, and it is the one that
                    // catches a second thing painted in the label's rect but
                    // outside its words.
                    const runFrom = r.labelBand.runX;
                    const runTo = r.labelBand.runX + r.labelBand.runW;
                    // And whether that rect is inside the box it was measured
                    // in — asked first, because everything below is taken from
                    // it and none of the three readings can tell. See
                    // inkRunRectFault for the pair this closes.
                    const runRect = inkRunRectFault(where, r.label, r.labelOwn,
                        r.labelBand);
                    if (runRect) {
                        problems.push(runRect);
                        continue;
                    }
                    const scan = scanInk(bandImg, bandDpr, rows,
                        runFrom, runTo, b.fill, want);
                    if (scan.columns < 1) {
                        problems.push(`${where}: the label's run of words reports a rect ` +
                            `${r.labelBand.runW}px wide, which leaves no column to scan. ` +
                            `Every verdict below would be a statement about no pixels`);
                        continue;
                    }
                    // What a rounding of one device row would cost this scan,
                    // which is the magnitude INK_EDGE_CLEARANCE's argument is
                    // about and nothing had read. See inkRoundingVerdict.
                    // And what keeps those fractions off the x-height line,
                    // which is a structural argument rather than that floor.
                    // See INK_ASCENDER_SEPARATION.
                    const ascender = inkAscenderVerdict(where, "the label's words",
                        bandImg, bandDpr, r.labelBand, runFrom, runTo, b.fill);
                    if (ascender) problems.push(ascender.problem);
                    const rounding = inkRoundingVerdict(where, "the label's words",
                        bandImg, bandDpr, band, runFrom, runTo, b.fill);
                    if (rounding.problem) {
                        problems.push(rounding.problem);
                    } else {
                        inkOutsideBand.push({ where, subject: "the label's words",
                            coverage: rounding.outside });
                    }
                    // The rest of the box. A gutter of one device-independent
                    // pixel at each end of the run absorbs what a glyph puts
                    // outside its own client rect — an italic's side bearing,
                    // the outer half-pixel of an antialiased stem — so what is
                    // being asserted is "the far side of the box is backdrop"
                    // rather than "the run's rect is exact to the pixel".
                    const surplus = surplusInk(bandImg, bandDpr, rows, r.label,
                        runFrom - INK_RUN_GUTTER, runTo + INK_RUN_GUTTER, b.fill);
                    if (surplus) {
                        problems.push(`${where}: the label's box runs from ` +
                            `x=${r.label.x.toFixed(2)} to ` +
                            `${(r.label.x + r.label.w).toFixed(2)} and its run of words ` +
                            `occupies ${runFrom.toFixed(2)} to ${runTo.toFixed(2)}. At ` +
                            `x=${surplus.at.toFixed(2)}, outside the words, the pixel is ` +
                            `${surplus.got} and the band's fill is ${b.fill}.\n\n` +
                            `The three ink rows are measured for the RUN — the inline ` +
                            `text box's baseline and the resolved face's x-height — and ` +
                            `the rest of the element's rect is backdrop by construction ` +
                            `on every branch where the box is stretched. Something is ` +
                            `painted in the label's rect and outside its words, which is ` +
                            `a colour the run's own scan sweeps past and never sees.\n\n` +
                            `The run-rect check below is not asked of this box. inkExtent ` +
                            `reads the columns of this rect that are not the fill and ` +
                            `calls the first and last of them the ends of the words, ` +
                            `which is only the words while nothing else is painted here ` +
                            `on these rows — and that is exactly what has just fired. ` +
                            `Whatever this pixel belongs to would move the extent, and ` +
                            `the comparison would be about a different quantity`);
                    } else {
                        // The other half is asked only here, and that is the
                        // dependency stated rather than assumed. Until this
                        // line the two checks were sound by an accident of
                        // order: inkExtent calls the columns of this rect that
                        // are not the fill "the words", which is true only
                        // because the check above has just held everything in
                        // the rect and outside the run to the backdrop. A
                        // surplus downgraded to a warning, or moved below this,
                        // would leave the extent measuring whatever else was
                        // painted here and saying nothing about the run.

                        // And whether the run's rect is where the words are.
                        //
                        // Everything above is taken from that rect: the rows are
                        // fractions of a band measured off its top, the scan sweeps
                        // it, and the surplus is what it leaves over. None of the
                        // three says the rect is in the right PLACE, because each
                        // is measured from it.
                        //
                        // The surplus check bounds one direction of that on its
                        // own: paint outside the rect is a colour where the
                        // backdrop should be, so a rect that had shrunk onto half
                        // the words fails there. What nothing saw is the other
                        // direction — a rect wider than the words, or displaced
                        // into the empty half of a stretched box, whose extra
                        // columns are backdrop and answer nothing either way. This
                        // is that half, and it is the paint's own answer rather
                        // than another reading of the layout's: see inkExtent. The
                        // two together hold the rect and the ink to within a gutter
                        // of each other at both ends.
                        const extent = inkExtent(bandImg, bandDpr, rows,
                            r.label.x, r.label.x + r.label.w, b.fill);
                        if (extent) {
                            const short = [
                                ["starts", extent.from - runFrom],
                                ["ends", runTo - extent.to],
                            ].filter(([, d]) => d > INK_RUN_GUTTER);
                            if (short.length > 0) {
                                problems.push(`${where}: the browser puts the label's run of ` +
                                    `words at x=${runFrom.toFixed(2)} to ${runTo.toFixed(2)}, ` +
                                    `and the ink in that box runs from ` +
                                    `${extent.from.toFixed(2)} to ${extent.to.toFixed(2)} — ` +
                                    short.map(([end, d]) =>
                                        `the paint ${end} ${d.toFixed(2)}px inside the rect`)
                                        .join(", ") + `, against a gutter of ` +
                                    `${INK_RUN_GUTTER}px.\n\n` +
                                    `The rows scanned here are fractions of a band measured ` +
                                    `off that rect's own top, the ink scan sweeps it, and ` +
                                    `surplusInk holds everything outside it to the ` +
                                    `backdrop — all three are taken FROM the rect, so none ` +
                                    `of them says the words are in it. surplusInk catches a ` +
                                    `rect the paint spills OUT of; this is the other ` +
                                    `direction, read off the screenshot: a rect the paint ` +
                                    `does not reach the end of is one wider than the words ` +
                                    `or sitting beside them, in columns that are backdrop ` +
                                    `and answer nothing either way`);
                            }
                        }
                    }
                    if (scan.offBy > OFF_SEGMENT_EPSILON) {
                        problems.push(`${where}: a pixel in the label's box is ` +
                            `${scan.stranger}, which is ${scan.offBy.toFixed(1)} ` +
                            `channels off the line ` +
                            `between the band's fill ${b.fill} and the label's ` +
                            `composited ink ${want}. Every pixel a glyph antialiases ` +
                            `is on that line — coverage blends the two and nothing ` +
                            `else — so this is a THIRD colour drawn inside the rect ` +
                            `the scan reads. confusableInk cleared this case against ` +
                            `the three colours the band declares, and a colour it was ` +
                            `never shown is a colour it could not rule the ink out ` +
                            `against: either something is being painted here that ` +
                            `should not be, or the band fixture has to declare it so ` +
                            `the tolerance can be checked against it too`);
                    }
                    if (!scan.notFill) {
                        problems.push(`${where}: every pixel on all ${INK_ROWS.length} ` +
                            `scanned rows of the label's own run of words is ${b.fill}, ` +
                            `the band's own fill. The words are not there — and until this scan a band ` +
                            `that painted its fill over them passed every assertion in ` +
                            `this check, because a rect is the same either way`);
                    } else if (!scan.ink) {
                        problems.push(`${where}: the label's box has ink in it and the ` +
                            `pixel furthest from the band's ${b.fill} is ${scan.darkest}. ` +
                            `components.GroupHeader declares ${b.labelInk} for the words ` +
                            `(core.TextColor(TextSecondary)), which over this band ` +
                            `composites to ${want}. Something is drawn there in another ` +
                            `colour, which is what a label that lost its declaration and ` +
                            `inherited one looks like`);
                    }
                    // Reached only with every reading above taken: the rows
                    // measured, the run rect held, the surplus swept and the
                    // three verdicts decided. See `asked`.
                    asked.words++;
                }
            }

            // And the count pill, which is a fill AND a number.
            //
            // The fill is sampled at a point, half the pill's OWN leading
            // padding in and vertically centred: the pill's radius is 999, so
            // its leftmost point is the apex of a curve and every pixel there
            // is an antialiased blend — the same trap the widget grid's ring
            // scan records. At mid-height the box is at its widest, and half a
            // padding in is clear of that curve and short of the first digit.
            //
            // It was a literal 5, which is inside the 8px padding the pill
            // carries today and is a number chosen against a widget's current
            // insets — the same shape as a byte window measured off somebody
            // else's file. gen.go reads the padding off the rendered node and
            // refuses a pill with too little of it to sample inside, so this
            // point moves when the widget does.
            //
            // The DIGITS are scanned, for the reason the label's words are.
            // "A band is a fill, a run of words and a count" is the argument
            // this whole scan was added for, and the count was two thirds
            // unread: a pill that painted itself and rendered no number passed
            // every assertion here, which is exactly the state the label was in
            // before any of this existed. A number cannot be sampled at a point
            // any more than a word can.
            if (r.badge && pixelsHeld) {
                // Computed once: it is the same sentence either way, and it was
                // being built twice to be tested and then reported.
                const digitsUnreadable = inkUnreadable(
                    where, "the count's digits", b.badgeProbe, r.badgeAncestry,
                    r.badgeOwn, r.badgeSubject, bandFaces[i].badge,
                    canvasAt(i, "badge"));
                const badgeFill = pixelAt(bandImg,
                    (r.badge.x + b.badgePadLeft / 2) * bandDpr,
                    (r.badge.y + r.badge.h / 2) * bandDpr);
                if (badgeFill !== b.badgeFill) {
                    problems.push(`${where}: the count pill painted as ${badgeFill}, and ` +
                        `its own Style declares ${b.badgeFill} (the band behind it is ` +
                        `${b.fill}). A badge with no pill is a number sitting on the band, ` +
                        `which lays out identically and is a different widget`);
                } else if (digitsUnreadable) {
                    problems.push(digitsUnreadable);
                } else {
                    // The digits' own box: between the pill's two paddings, and
                    // nothing else. Outside them are the pill's leading and
                    // trailing edges, which are the apexes of a 999-radius
                    // curve and therefore blends with the BAND rather than with
                    // the pill — a third colour to the segment test, and the
                    // one place in this scan where that would be correct
                    // painting rather than a fault.
                    const ink = over(b.badgeInk, b.badgeFill);
                    const confusable = channelDistance(ink, b.badgeFill);
                    if (confusable <= INK_EPSILON * INK_MARGIN) {
                        problems.push(`${where}: the count's ink ${b.badgeInk} composites ` +
                            `over its pill to ${ink}, which is ${confusable} channels ` +
                            `from the pill's own ${b.badgeFill}. The scan below accepts a ` +
                            `pixel within ${INK_EPSILON} of the composite and is only ` +
                            `worth anything while the two colours are an order further ` +
                            `apart than that (${INK_EPSILON * INK_MARGIN}) — and a count ` +
                            `this close to its pill is one nobody can read either. ` +
                            `components.Badge picks the ink against the fill ` +
                            `(Variant.Ink) precisely so this does not happen`);
                    } else {
                        // The two numbers the window is built from, held to
                        // what the browser resolved on that same element. See
                        // inkBadgePadFault: until this line the window was Go's
                        // arithmetic applied to the browser's rect, and nothing
                        // had asked whether the two agreed about the padding.
                        const badgePad = inkBadgePadFault(where, r.badgeOwn,
                            b.badgePadLeft, b.badgePadRight);
                        if (badgePad) {
                            problems.push(badgePad);
                            continue;
                        }
                        const from = r.badge.x + b.badgePadLeft;
                        const to = r.badge.x + r.badge.w - b.badgePadRight;
                        const band = inkBandRows(where, "the count's digits",
                            r.badgeBand, bandDpr,
                            `Digits are lining figures — every one of them runs from ` +
                            `the baseline to the same cap height — so that band is all ` +
                            `there is to spread three rows across`);
                        if (band.problem) {
                            problems.push(band.problem);
                        } else {
                            const rows = band.rows;
                            // The same question the label's rows are held to:
                            // what one device row of rounding would cost this
                            // scan. Digits are lining figures, so their band is
                            // taller than a word's x-height band and the rows
                            // are further apart — and the pill's window is six
                            // columns wide, where one column is a sixth of the
                            // reading. See INK_ROW_ROUNDING for both numbers.
                            const rounding = inkRoundingVerdict(where,
                                "the count's digits", bandImg, bandDpr, band, from, to,
                                b.badgeFill);
                            if (rounding.problem) {
                                problems.push(rounding.problem);
                            } else {
                                inkOutsideBand.push({ where,
                                    subject: "the count's digits",
                                    coverage: rounding.outside });
                            }
                            const scan = scanInk(bandImg, bandDpr, rows, from, to,
                                b.badgeFill, ink);
                            if (scan.columns < 1) {
                                // The window is what is left of the pill after
                                // its own two paddings, and an empty one makes
                                // every verdict below a statement about no
                                // pixels: "the number is not there" would fire
                                // whatever the pill painted.
                                problems.push(`${where}: the count pill is ` +
                                    `${r.badge.w}px wide and its own paddings are ` +
                                    `${b.badgePadLeft} and ${b.badgePadRight}, which ` +
                                    `leaves no column between them for the digits to be ` +
                                    `scanned in. Every reading below would be about an ` +
                                    `empty window`);
                            } else if (scan.offBy > OFF_SEGMENT_EPSILON) {
                                problems.push(`${where}: a pixel in the count's own box ` +
                                    `is ${scan.stranger}, ${scan.offBy.toFixed(1)} ` +
                                    `channels off the line between the pill's ` +
                                    `${b.badgeFill} and its composited ink ${ink}. ` +
                                    `Between the pill's two paddings there is nothing ` +
                                    `but fill and digit, so a third colour there is ` +
                                    `either something else being drawn inside the pill ` +
                                    `or the scan reaching past a padding and into the ` +
                                    `curve of the pill's own edge`);
                            }
                            if (!scan.notFill) {
                                problems.push(`${where}: every pixel on all ` +
                                    `${INK_ROWS.length} scanned rows between the count ` +
                                    `pill's paddings is ${b.badgeFill}, the pill's own ` +
                                    `fill. The number is not there. A band is a fill, a ` +
                                    `run of words and a count, and until this scan the ` +
                                    `count was the one of the three read as a colour ` +
                                    `and never as digits — a pill that painted itself ` +
                                    `over an empty box passed every assertion here`);
                            } else if (!scan.ink) {
                                problems.push(`${where}: the count pill has ink in it ` +
                                    `and the pixel furthest from its ${b.badgeFill} is ` +
                                    `${scan.darkest}. components.Badge declares ` +
                                    `${b.badgeInk} for the digits, which over the pill ` +
                                    `composites to ${ink}. Something is drawn there in ` +
                                    `another colour, which is what a count that lost ` +
                                    `its declaration and inherited one looks like`);
                            }
                            // Not credited when the window came back empty: the
                            // three verdicts above all ran, and every one of
                            // them is a statement about no pixels. The label's
                            // equivalent leaves the loop at that point; this one
                            // goes on to say what it found in nothing, so the
                            // ledger is what has to decline it. See `asked`.
                            if (scan.columns >= 1) asked.counts++;
                        }
                    }
                }
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

            // This band's tap-target ledger: one verdict per assertion the
            // claim is made of, written here and reduced at the bottom of the
            // loop. See BAND_TARGET_PARTS — the tail recites the conjunction
            // and the census counts the parts, and this is the one reading
            // both of them come out of.
            const target = {};

            // The band's leading edge is the control's, which is the whole move:
            // the insets came off the Row so that a press lands on the leading
            // edge rather than 16px into it.
            const lead = r.control.x - r.band.x;
            target.lead = bandRenderSame(lead, b.rowLeft);
            if (!target.lead) {
                problems.push(`${where}: the control starts ${lead.toFixed(2)}px into ` +
                    `the band and the Row's own leading inset is ${b.rowLeft}px. A press ` +
                    `on the first ${lead.toFixed(2)}px of this band lands on nothing`);
            }

            // And where the control's own inset puts its content, which is the
            // other half of the same declaration.
            //
            // The line above says the tap target reaches the band's edge; this
            // says the words do not. Both matter, and they are what makes
            // components.GroupHeader.ControlStyle a safe thing to hand a
            // caller: an indent through it moves the content and MUST NOT move
            // the target, which is the whole reason the chrome sits on the
            // control rather than on the Row.
            //
            // The fifth shape declares PaddingLeft(40) through that field, and
            // until this line nothing said it had moved anything. Every
            // assertion over that band was one the other four also make, so a
            // ControlStyle that stopped reaching the control would have left
            // the shape passing while exercising nothing — and it is in the
            // grid to put the ink scan in front of a label rect the other four
            // never produce. gen.go holds the rendered padding to the number
            // the fixture declared (which is the half a rect cannot see) and
            // refuses an indent that lands on the theme's own inset; this is
            // the same number in pixels, on the screen.
            //
            // The control's FIRST child, not the label: the disclosure branch
            // puts the chevron in front of the words, so `label.x - control.x`
            // there is the padding plus a glyph and a gap, and only the padding
            // is declared anywhere.
            if (r.leading) {
                const inset = r.leading.x - r.control.x;
                if (bandRenderSame(inset, b.controlPadLeft)) asked.insets++;
                if (!bandRenderSame(inset, b.controlPadLeft)) {
                    problems.push(`${where}: the control's content starts ` +
                        `${inset.toFixed(2)}px into it and the control's own leading ` +
                        `padding is ${b.controlPadLeft}px. ` +
                        (b.controlIndent > 0
                            ? `That inset is the caller's own indent through ` +
                              `components.GroupHeader.ControlStyle — the declaration ` +
                              `this shape is in the grid to exercise, and the one that ` +
                              `is supposed to move the label without moving the tap ` +
                              `target the line above just measured`
                            : `That is the band's own chrome, which sits on the control ` +
                              `rather than on the Row so that a press lands on the whole ` +
                              `band and the words still sit in from its edge`));
                }
            } else {
                problems.push(`${where}: gen.go names no leading child of the control, ` +
                    `so nothing here says where the control's own inset put its content. ` +
                    `The band's chrome is on the control precisely so the tap target and ` +
                    `the words can have different leading edges, and only one of the two ` +
                    `was measured`);
            }

            // And its trailing edge stops where the band's content does: at the
            // badge when there is one, at the Row's own trailing inset when
            // there is not. This is the half check 9 cannot see on the
            // disclosure branch, because it is the wrapper that is distributed
            // to and the button that has to fill it.
            const wantRight = r.band.x + r.band.w - b.rowRight -
                (r.badge ? r.badge.w + b.gap : 0);
            const gotRight = r.control.x + r.control.w;
            target.trail = bandRenderSame(gotRight, wantRight);
            if (!target.trail) {
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
                target.stretch = bandRenderSame(r.control.w, r.wrapper.w) &&
                    bandRenderSame(r.control.x, r.wrapper.x);
                if (!target.stretch) {
                    problems.push(`${where}: the button is ${r.control.w.toFixed(2)}px ` +
                        `wide at x=${r.control.x.toFixed(2)} inside a heading wrapper ` +
                        `${r.wrapper.w.toFixed(2)}px wide at x=${r.wrapper.x.toFixed(2)}. ` +
                        `A non-growing child of a vertical container is supposed to be ` +
                        `stretched to it — that is the cross-axis fallback core.Box and ` +
                        `core.SafeArea both document — and it is the only thing making ` +
                        `the disclosure band's tap target span the band`);
                }
            }
            // The ledger above is the whole of the tap-target claim, so this is
            // where it is counted: each part at its own counter, and the
            // conjunction the tail recites DERIVED from the parts rather than
            // carried alongside them. See bandTargetTally — a number that says
            // how many bands lost the claim without saying which assertion
            // cost them is what the census is for, and a conjunction nothing
            // ties to those counts is what this is.
            for (const p of bandTargetTally(where, target, asked, Boolean(r.wrapper))) {
                problems.push(p);
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

            // Keyed by theme and by the parity group gen.go names, because the
            // comparison below is between two bands that file says are one
            // band with and without a handler.
            //
            // The grouping used to be worked out here from two booleans — the
            // branch, and whether a count was showing — which was the same
            // claim while there were exactly three shapes and one of them had
            // no partner. It is a fact about the fixture rather than about the
            // rects, so the fixture states it: a shape with no pair (the
            // indented long-titled band) is measured on its own and compared
            // with nothing, and adding a fourth shape does not silently start
            // or stop a comparison.
            if (b.pair) {
                const key = `${b.theme}\u0000${b.pair}`;
                const seen = byTheme.get(key) || {};
                seen[b.collapsible ? "disclosure" : "plain"] = { b, r };
                byTheme.set(key, seen);
            }
        }

        // How much of what the tail recites was actually asked.
        //
        // See `asked`. Every number below is counted at the end of its own
        // check's success path, so it says how many boxes were read rather
        // than how many the transcript declared. A shortfall means some band's
        // words or count went unscanned, and the reason is one of the messages
        // above — which is the point: on a run where the grid was clipped, or a
        // probe came back coloured, or a label's node was missing, the reader
        // gets one line saying how much of the recitation below is not a
        // statement about anything.
        //
        // It is a census and not a second failure. Every path that suppresses a
        // scan has already said why; what none of them said is how many.
        for (const [what, population, subject] of [
            // The tap target's three rows come from the table that owns them,
            // population and phrase together. See bandTargetCensus.
            ...targetCensus.map((r) => [r.counter, r.population, r.subject]),
            ["insets", "bands whose leading child gen.go names",
                "their content held to their own declared inset"],
            ["fills", "bands in this grid", "their own fill read off the screenshot"],
            ["words", "bands that declare a label", "their words in their own ink"],
            ["counts", "bands that declare a count", "their counts as digits"],
        ]) {
            if (asked[what] >= declared[what]) continue;
            problems.push(`${declared[what] - asked[what]} of the ${declared[what]} ` +
                `${population} did not get ${subject} — ${asked[what]} did.

` +
                `The reasons are among the messages above; this is the count, which ` +
                `none of them carries. The line this check prints when it passes used ` +
                `to recite the size of gen.go's table, so without this a run could ` +
                `suppress every pixel read in the grid, report the one message that ` +
                `says so, and pass every rect-shaped assertion in the same breath — ` +
                `and a reader who skimmed the tail would find twenty bands recited ` +
                `and no way to tell how many had been looked at`);
        }

        // And what the floor those scans were held to actually separates.
        //
        // INK_ROW_ROUNDING is a bound with a measurement on each side of it:
        // above what a row on the band's own baseline scores, below what the
        // rows the scan reads score. The first half is the one nothing would
        // report on its own — a floor that no reading in this grid falls under
        // is a floor every row passes for free, and the clearance it exists to
        // justify would be protecting nothing. So the reading just outside each
        // band is kept as it is taken and the thinnest of them is asked here.
        //
        // The same shape as SUBPIXEL_FLOOR, and for the same reason: a margin
        // that is silently spent is a margin nobody notices leaving.
        if (inkOutsideBand.length > 0) {
            const thinnest = inkOutsideBand.reduce((a, b) =>
                b.coverage < a.coverage ? b : a);
            if (thinnest.coverage >= INK_ROW_ROUNDING) {
                problems.push(`the row immediately below the band is at least ` +
                    `${(thinnest.coverage * 100).toFixed(1)}% glyph in all ` +
                    `${inkOutsideBand.length} boxes this grid scans — the thinnest is ` +
                    `${thinnest.where}, ${thinnest.subject} — and INK_ROW_ROUNDING is ` +
                    `${(INK_ROW_ROUNDING * 100).toFixed(0)}%.\n\n` +
                    `That floor is what makes INK_EDGE_CLEARANCE a measurement: every ` +
                    `scanned row is held to keeping its ink under one device row of ` +
                    `movement, and the number is set above what a row ON the baseline ` +
                    `scores (0.100 at worst when it was measured, and 0.000 in every ` +
                    `count pill) and below what the rows a clearance of ` +
                    `${INK_EDGE_CLEARANCE} lands on score (0.239 at worst). With ` +
                    `nothing in the grid under the floor, the first of those two ` +
                    `brackets has gone: the rows would pass wherever they were put, and ` +
                    `the clearance would be a preference again`);
            }
        }

        // And the same question of the bound the canvas join is asked under.
        //
        // LAYOUT_UNIT is the tolerance there because two measureText calls, on
        // one canvas, in one evaluate, for one string, under two requests that
        // resolve to one face, return THE SAME DOUBLE. The bound exists so the
        // check is about a face and not about a double's last bits — and that
        // is a statement about the derivation with nothing recording what the
        // population actually does under it. A margin that is silently spent is
        // a margin nobody notices leaving, which is the argument INK_ROW_ROUNDING
        // has a census on both sides of it for.
        //
        // Zero is what the premise predicts, so anything at all inside the
        // bound is reported: the run is not failing the check, it is spending a
        // tolerance the check was not supposed to need. The other bracket is
        // not hypothetical — a canvas put on a different face costs about 18px
        // on this grid's strings, three orders above this bound — so the
        // reporting threshold is the premise itself rather than a fraction of
        // the margin.
        if (asked.canvasFaces > 0 && asked.canvasWidest > 0) {
            problems.push(`the canvas join is bounded by a LayoutUnit ` +
                `(${LAYOUT_UNIT}) and the widest of the ${asked.canvasFaces} runs it ` +
                `was asked of came apart by ${asked.canvasWidest.toFixed(6)}px.

` +
                `That is under the bound, so no run reported — and the bound is not ` +
                `supposed to be doing any work here. Both numbers are measureText on ` +
                `ONE canvas, in one evaluate, for one string; the only difference ` +
                `between the requests is a family at the head of the list that the ` +
                `canvas has already resolved to. Two such calls return the same ` +
                `double, and the tolerance is there so the check is about a face ` +
                `rather than about a rounding. A nonzero widest means that premise ` +
                `has stopped holding: either the head family is reaching a second ` +
                `face with very nearly the same advances, or something between the ` +
                `two calls is moving the measurement — and the next face that ` +
                `differs by less than ${LAYOUT_UNIT} goes unreported`);
        }

        // And the same question of the bound the CANARY is asked under, which
        // is the reading that says the join bit a face at all.
        //
        // inkCanvasNameReached takes a LayoutUnit of difference between the
        // generic alone and the family in front of it as the name having
        // reached something. The argument for that bound is metric — a
        // fixed-advance face is nothing like this grid's proportional ones over
        // these strings — and a run that clears it by a 64th of a pixel has
        // demonstrated nothing of the kind. Below INK_CANARY_MARGIN the reading
        // is still true and has stopped being evidence, and the way it fails
        // next is silent: canary and base become one number, every run reads as
        // a name that reached nothing, and the tail says so about a family the
        // page can reach perfectly well.
        //
        // Asked only where a run reached, because that is the population the
        // bound decides. A minimum over no such runs is not a thin margin, and
        // the arm above this one is what fires when there are none.
        if (asked.canvasNamed > 0 && asked.canvasNarrowest < INK_CANARY_MARGIN) {
            problems.push(`the canary that prices the canvas join is separated by ` +
                `${asked.canvasNarrowest.toFixed(6)}px at its narrowest over the ` +
                `${asked.canvasNamed} runs it confirmed, against a margin of ` +
                `${INK_CANARY_MARGIN}px.

` +
                `That reading is what makes ${asked.canvasNamed} of ` +
                `${asked.canvasFaces} joins evidence rather than a skipped lookup, and ` +
                `it works by measuring one string twice: under ` +
                `${INK_CANARY_FALLBACK} alone, and under the compositor's family in ` +
                `front of ${INK_CANARY_FALLBACK}. A difference is the family reaching ` +
                `a face. The bound on that difference is a LayoutUnit ` +
                `(${LAYOUT_UNIT}) and the reason a LayoutUnit is enough is metric — ` +
                `the faces this grid draws with are proportional and a fixed-advance ` +
                `face's widths for these strings are nothing like theirs — so a ` +
                `separation this thin means the generic has resolved to something ` +
                `very near the face the text is set in.

` +
                `What that costs on the next run is not this message. It is silence: ` +
                `once the two coincide within a LayoutUnit, every join in the grid ` +
                `reports as a name that reached nothing, and the reader is sent after ` +
                `an unreachable family name that is reachable`);
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
        let pairsRun = 0;
        for (const [key, seen] of byTheme) {
            const [theme, pair] = key.split("\u0000");
            const where = `${theme}/${pair}`;
            if (!seen.plain || !seen.disclosure) {
                problems.push(`${where}: only the ` +
                    `${seen.plain ? "plain" : "disclosure"} half of this parity pair was ` +
                    `measured, so the comparison did not run. gen.go refuses a pair with ` +
                    `one branch, so this is the mount losing a band rather than the ` +
                    `fixture declaring one`);
                continue;
            }
            pairsRun++;
            const p = seen.plain.r, d = seen.disclosure.r;

            if (!d.chevron || !d.label || !p.label) {
                problems.push(`${where}: the branch pair is missing a text rect ` +
                    `(chevron ${Boolean(d.chevron)}, disclosure label ${Boolean(d.label)}, ` +
                    `plain label ${Boolean(p.label)}) — gen.go locates all three by the ` +
                    `words they carry, so the band's content has changed shape`);
                continue;
            }
            const overhang = Math.max(0, d.chevron.h - d.label.h);
            const taller = d.band.h - p.band.h;
            if (!bandRenderSame(taller, overhang)) {
                problems.push(`${where}: the disclosure band is ` +
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
                problems.push(`${where}: the plain band's control starts ` +
                    `${(p.control.x - p.band.x).toFixed(2)}px into it and the ` +
                    `disclosure's starts ${(d.control.x - d.band.x).toFixed(2)}px in. ` +
                    `Both branches take the same bandInsets, so the label moves on the ` +
                    `day a caller adds a handler`);
            }
            const pRight = p.band.x + p.band.w - (p.control.x + p.control.w);
            const dRight = d.band.x + d.band.w - (d.control.x + d.control.w);
            if (!bandRenderSame(pRight, dRight)) {
                problems.push(`${where}: the plain band's control stops ` +
                    `${pRight.toFixed(2)}px short of the trailing edge and the ` +
                    `disclosure's stops ${dRight.toFixed(2)}px short. The two branches ` +
                    `are the same band, so the tap target's trailing edge is not ` +
                    `supposed to move when a handler arrives`);
            }
        }
        // And every pair the fixture declares was actually compared. A
        // comparison that runs over nothing reports nothing, which is the one
        // result that reads the same as a pass.
        const wantPairs = new Set(BAND_RENDERS.filter((b) => b.pair)
            .map((b) => `${b.theme}\u0000${b.pair}`)).size;
        if (wantPairs !== pairsRun) {
            problems.push(`gen.go declares ${wantPairs} theme-and-shape parity pairs and ` +
                `${pairsRun} were compared. The pairing is what holds "a caller who adds ` +
                `OnToggle gets a control and not a relayout" to pixels, and a pair that ` +
                `does not run says nothing at all`);
        }
        if (BAND_RENDERS.length && wantPairs === 0) {
            problems.push(`no band shape declares a parity pair, so the branch ` +
                `comparison ran over nothing — bandRenderBuilders in gen.go is supposed ` +
                `to carry a plain band and a disclosure band for each one`);
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

        // ------------------------------------------------------------------
        // 12. a pinned child keeps its base in a browser, and the siblings do
        //     not depend on the order
        // ------------------------------------------------------------------
        //
        // See PIN_GRID for the fixture, for what each claim rests on, and for
        // why nothing here recomputes a flex line.
        await mount(PIN_GRID);

        // Widths AND leading edges. The edges are what the spacing claim is
        // read out of: a gap is not a node, so the only way to ask a browser
        // what it inserted between two children is to subtract.
        const pinRects = await evaluate(`${JSON.stringify(
            PINS.map((c, i) => c.children.map((_, j) => `root/${i}/${j}`)))}.map((paths) =>
            paths.map((p) => {
                const el = document.querySelector('[data-node-path="' + p + '"]');
                if (!el) return null;
                const r = el.getBoundingClientRect();
                return { w: r.width, x: r.left };
            }))`);

        // Each child's extent, by name, across the three arrangements that have
        // a pin. The order claim spans cases, so it is collected as they are
        // read and compared after all of them — the same shape pin.swift's
        // pinnedExtent/unpinnedExtent pair has.
        const pinnedRowExtent = {};
        let sawAgreement = false, sawDivergence = false;
        let sawGapAgreement = false, sawGapDivergence = false;
        // Whether any row charges a gap strictly between zero and the Row's own
        // spacing. The collapse is a min over two quantities, and a fixture
        // whose charged gaps only ever come out at one end or the other is
        // described just as well by "charge the gap unless the Row has
        // overflowed" — a simpler rule, and not the one MeasureCompose
        // transcribes. internal/pinfixture carries a row for the middle arm;
        // this refuses to read a transcript that has lost it, for the reason
        // the overflow guard refuses one that cannot overflow.
        let sawPartialGap = false;

        for (let i = 0; i < PINS.length; i++) {
            const c = PINS[i], boxes = pinRects[i];
            const where = `the pinned Row, ${c.what}`;
            if (boxes.some((b) => b === null)) {
                problems.push(`${where}: a child was not laid out`);
                continue;
            }
            // Before anything is compared: this pass's tolerance is finer than
            // the finest distinction the fixture asks it to make. See
            // PIN_EPSILON — internal/pinfixture derives the floor and
            // ios/verify/pin.swift holds its own, different tolerance to it.
            if (!(c.resolution > PIN_EPSILON * PIN_MARGIN)) {
                problems.push(`${where}: the fixture's numbers come as close together as ` +
                    `${c.resolution} and every comparison below uses a tolerance of ` +
                    `${PIN_EPSILON}. A tolerance within a factor of ${PIN_MARGIN} of a ` +
                    `case's resolution can accept one of the fixture's own numbers where ` +
                    `another was meant — the partial-spacing row charges 4 against a Row ` +
                    `declaring 16 and against the 0 a fully collapsed gap would give, and ` +
                    `at that tolerance the three are one answer`);
                continue;
            }
            const mains = boxes.map((b) => b.w);
            if (c.css.length !== mains.length) {
                problems.push(`${where}: internal/pinfixture states ${c.css.length} CSS ` +
                    `extents for ${mains.length} children, so the column the census ` +
                    `prints is about a different Row`);
                continue;
            }

            // The census's CSS column, measured.
            //
            // This is the one claim here that is about the table rather than
            // about the pin, and it is why the column became a field. Three of
            // the rows are pinned to their numbers by the assertions below —
            // the pin keeps its base, the extents match Compose where the
            // fixture says they do, a child's width does not depend on where it
            // sits. The control row's 24/80/16 is determined by none of them:
            // it is the scaled-base rule producing three particular numbers,
            // and a reader who trusted them was trusting a comment.
            //
            // Nothing is recomputed. The fixture STATES the column and this
            // reads pixels; a flex line written in JavaScript would make the
            // check about whether two transcriptions agree, which is the thing
            // this whole mount exists not to be.
            c.css.forEach((want, j) => {
                if (!pinSame(mains[j], want)) {
                    problems.push(`${where}: ${c.children[j].name} laid out at ` +
                        `${mains[j].toFixed(2)}px and internal/pinfixture states the CSS ` +
                        `column as [${c.css.join(", ")}]. That column is what the census ` +
                        `prints; nothing computes it, so a browser and GrMobFlexSolver ` +
                        `are the two things holding it — and the control row is the one ` +
                        `no other claim here determines`);
                }
            });

            // The spacing, which is a gap between two edges rather than a node.
            //
            // A CSS flex line charges its gap between every adjacent pair
            // whatever happened to the children — it is used space, taken out
            // before anything is distributed. A Compose Row clamps each one to
            // what is left, so an overflowing Row inserts none after the child
            // that spent the axis. Three documents repeat that sentence and
            // every case in the fixture carried gap 0 until one of them did
            // not, so nothing had ever measured it.
            //
            // Both arms, for the reason the extents' agreement has both: four
            // of the six rows have no gap and agree vacuously, which is what
            // bounds the two that do not.
            let gapsSame = true;
            for (let j = 0; j + 1 < mains.length; j++) {
                const measured = boxes[j + 1].x - (boxes[j].x + mains[j]);
                if (!pinSame(measured, c.gap)) {
                    problems.push(`${where}: a browser leaves ${measured.toFixed(2)}px ` +
                        `between ${c.children[j].name} and ${c.children[j + 1].name}, and ` +
                        `the Row declares a ${c.gap}px gap. A flex line's gap is used ` +
                        `space: it comes off the free space before anything is shrunk and ` +
                        `is charged between every adjacent pair, whatever the children ` +
                        `ended up at`);
                }
                if (!pinSame(c.compose.gaps[j], c.gap)) gapsSame = false;
                // And whether this row charges a gap that is NEITHER the Row's
                // spacing nor zero. See sawPartialGap.
                if (c.compose.gaps[j] > 0 && c.compose.gaps[j] < c.gap) {
                    sawPartialGap = true;
                }
            }
            if (gapsSame) sawGapAgreement = true; else sawGapDivergence = true;
            if (gapsSame !== c.gapsAgreeWithCSS) {
                problems.push(`${where}: internal/pinfixture's Compose column inserts ` +
                    `[${c.compose.gaps.join(", ")}] of spacing in a Row whose gap is ` +
                    `${c.gap}px, which ${gapsSame ? "is" : "is not"} what the browser ` +
                    `above was just measured doing — and the fixture says ` +
                    `gapsAgreeWithCSS=${c.gapsAgreeWithCSS}. spaceAfterLastNoWeight is ` +
                    `min(spacing, what is left); a fixture that had stopped carrying a ` +
                    `row with a gap in it would assert one arm of that twice`);
            }

            // The declaration, on the target that has always had flex-shrink.
            // Compose needed Modifier.pinMainAxis to express this at all; a
            // browser needs one number, and this is the first time anything has
            // watched it arrive.
            c.children.forEach((child, j) => {
                if (child.pinned && !pinSame(mains[j], child.base)) {
                    problems.push(`${where}: the pinned child ${child.name} declared ` +
                        `${child.base}px with core.FlexShrink(0) and laid out at ` +
                        `${mains[j].toFixed(2)}px. That declaration is the whole subject ` +
                        `of the pin census — a browser that shrinks it anyway means the ` +
                        `CSS column of that table is describing something else`);
                }
            });

            // Agreement with the Compose column, both ways. The fixture derives
            // which it is (see agreesWithCSS), so this is the browser being
            // held to a stated fact rather than to whatever it produced.
            const agrees = mains.every((w, j) => pinSame(w, c.compose.mains[j]));
            if (c.mainsAgreeWithCSS) sawAgreement = true; else sawDivergence = true;
            if (agrees !== c.mainsAgreeWithCSS) {
                problems.push(`${where}: a browser lays the children out at ` +
                    `[${mains.map((w) => w.toFixed(2)).join(", ")}] and ` +
                    `internal/pinfixture's Compose column is ` +
                    `[${c.compose.mains.join(", ")}], which it says the two ` +
                    `${c.mainsAgreeWithCSS ? "agree" : "differ"} about. ` +
                    (c.mainsAgreeWithCSS
                        ? `They agree only where the pinned child comes first, and by ` +
                          `two different routes — CSS clamps the shrinkable children to ` +
                          `zero because the deficit exceeds their bases, Compose offers ` +
                          `them nothing because the pin already took more than the Row ` +
                          `had. A browser that has stopped agreeing means one of those ` +
                          `two rules is not what the fixture says it is`
                        : `Compose gives each child what the ones before it left and CSS ` +
                          `shares the deficit over every child that can shrink; a ` +
                          `browser that now AGREES means the divergence this fixture ` +
                          `exists to record has gone, and ios/verify is asserting it ` +
                          `against a solver rather than against the web`));
            }

            // The order. Collected here, compared below.
            if (c.children.some((child) => child.pinned)) {
                c.children.forEach((child, j) => {
                    const seen = pinnedRowExtent[child.name];
                    if (seen === undefined) {
                        pinnedRowExtent[child.name] = { w: mains[j], what: c.what };
                        return;
                    }
                    if (!pinSame(seen.w, mains[j])) {
                        problems.push(`the pinned Row: ${child.name} is ` +
                            `${seen.w.toFixed(2)}px wide with ${seen.what} and ` +
                            `${mains[j].toFixed(2)}px with ${c.what}. A CSS flex line's ` +
                            `sizes do not depend on where a child sits, which is the ` +
                            `whole of what makes Compose's answer a DIVERGENCE rather ` +
                            `than a second way of arriving at the same table — and it ` +
                            `is the property that pins the census's CSS column for the ` +
                            `two rows nothing else determines`);
                    }
                });
                continue;
            }

            // The control row: nothing pinned, so every child shrinks and the
            // line fits exactly. One declaration apart from the row above it,
            // which is what makes the pin load-bearing rather than decorative.
            const total = mains.reduce((a, w) => a + w, 0);
            if (!pinSame(total, c.offer)) {
                problems.push(`${where}: its children lay out at ` +
                    `[${mains.map((w) => w.toFixed(2)).join(", ")}], which comes to ` +
                    `${total.toFixed(2)}px in a ${c.offer}px Row. With nothing pinned ` +
                    `and a deficit smaller than what the children have to give, a flex ` +
                    `line shrinks to fit exactly — a total that is not the offer means ` +
                    `either a child refused to shrink or the fixture stopped ` +
                    `overflowing`);
            }
            c.children.forEach((child, j) => {
                if (!(mains[j] < child.base - PIN_EPSILON)) {
                    problems.push(`${where}: ${child.name} declared ${child.base}px ` +
                        `with no pin and laid out at ${mains[j].toFixed(2)}px. This is ` +
                        `the control for every pinned row: the same three children one ` +
                        `declaration apart, and a child that keeps its base here means ` +
                        `the pin is not what is keeping it in the rows above`);
                }
            });
        }

        // Both arms have to have run, for the reason band.swift gives about its
        // own table: a fixture where every row diverged would assert the
        // divergence and never reach the agreement, and the check would pass
        // just as well against a browser that had stopped agreeing anywhere.
        if (!sawAgreement || !sawDivergence) {
            problems.push(`the pinned Row: internal/pinfixture carried ` +
                (sawAgreement ? "" : "no case where CSS and Compose agree ") +
                (sawDivergence ? "" : "no case where they diverge ") +
                `— it is supposed to carry both, and half of what this check asserts ` +
                `is asserted over nothing`);
        }
        if (!sawGapAgreement || !sawGapDivergence) {
            problems.push(`the pinned Row: internal/pinfixture carried ` +
                (sawGapAgreement ? "" : "no case whose spacing survives ") +
                (sawGapDivergence ? "" : "no case whose spacing collapses ") +
                `— four of its six rows have no gap at all, and the other two are ` +
                `the only things that have ever measured min(spacing, what is left)`);
        }
        if (!sawPartialGap) {
            problems.push(`the pinned Row: every gap internal/pinfixture's Compose ` +
                `column charges is either the Row's own spacing or zero, so nothing ` +
                `here tells min(spacing, what is left) apart from \`charge the gap ` +
                `unless the Row has overflowed\`. Those two rules agree at both ends ` +
                `of the min and part company only in the middle; the fixture's ` +
                `partialGap row is what reaches it, and this transcript no longer ` +
                `carries it`);
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
    ${asked.targets} real bands span their own tap targets, ${asked.insets} indent their
    own content by their own declared inset and ${asked.fills} paint their own
    fill, ${asked.words} of them their words in their own ink and ${asked.counts}
    their counts as digits inside their own pills — every one of those five counted
    where its own check ran rather than off the size of gen.go's table, and held to
    it — on three rows
    taken as fractions of the ink band of the
    face this browser resolved for that very element, far enough off both its edges
    that a device row of rounding leaves them on the ink, and inside a band whose
    own top reads three quarters of the round letters where the row above it reads
    a tenth,
    over a run rect the paint reaches the ends of with the rest of the box held to
    the backdrop, behind ${INK_PROBES.length} antialiasing probes, one per text
    declaration any of it reads, every one of them and every box they answer for
    read at the element that draws the glyphs rather than at the box around it,
    drawn by one platform face this browser names, one across the whole grid, and
    named again to the canvas the ink band and the run advance are measured on in
    ${asked.canvasFaces} runs — a name a canary finds reaches a real face in
    ${asked.canvasNamed} of them — the thinnest of those canaries telling its two
    advances apart by ${asked.canvasNarrowest === null
        ? "no measured amount" : asked.canvasNarrowest.toFixed(4) + "px"}, where a
    LayoutUnit would have been read as a reaching name, and the generic it falls back
    to resolving on this browser to ${asked.canvasFallback === null
        ? "a face nothing here read" : asked.canvasFallback} — read at
    ${asked.canvasGenericRead} of the ${asked.canvasGenericReqs} distinct requests the
    joined runs make rather than once at whatever the page default is, and answering
    with ${inkCanarySpreadPhrase(asked)} — and measuring
    ${asked.canvasGenericNearest === null ? "no measured amount"
        : asked.canvasGenericNearest.toFixed(4) + "px"} from the nearest of the faces
    those runs are drawn by, over their own strings at their own sizes rather than by
    a name either of them could be reported under twice, so the agreement is a face's
    and not a skipped lookup's, with those two readings of one question — the name a
    probe reports and the distance a canvas measures — held against each other on
    ${asked.canvasGenericPaired} runs where both were in hand and agreeing on
    ${asked.canvasGenericAgreed}${inkCanaryPassedPhrase(asked)} — which gave the same
    width both ways, the widest of those
    ${asked.canvasFaces} differences being ${asked.canvasWidest.toFixed(6)}px against
    a bound of ${LAYOUT_UNIT},
    over the same tree the rects were read from and in the same layout the
    capture holds — a glyph per character of strings gen.go refuses a ligature
    pair in, ${asked.ligated.length} of those ${asked.ligatureSeeds} pairs
    drawn as one glyph by ${asked.ligatureFamily || "no face this run could name"}${
        asked.ligated.length > 0 ? ` (${asked.ligated.join(", ")})` : ``}, and the
    same face the probe was drawn with — under an ancestry that decides nothing
    about how a glyph is drawn, and
    resolving every computed property its probe's own text node resolves except
    the ${Object.keys(INK_OWN_MAY_DIFFER).length} INK_OWN_MAY_DIFFER gives a reason
    for — ${asked.ownRead} properties on this build against the
    ${INK_OWN_MEASURED_ON.props} ${INK_OWN_MEASURED_ON.browser} enumerated, both above
    the ${INK_OWN_SHAPE_FLOOR} it takes to be shaped like a computed style and that
    above the ${INK_SUBJECT_KEYS.length} keys of a subject read — and each permission
    spent by some box
    in the grid — and are taller
    than their badges with real glyphs in them, a fixed-size container squeezes its
    child along the main axis and lets it spill across — unless the child is
    pinned with core.FlexShrink(0), which until core.ShrinkNone was a declaration
    nobody could write — and ${PINS.length} arrangements of one overflowing Row
    honour that same pin wherever the child sits and lay their siblings out at
    the extents the pin census prints, agreeing with Compose in the
    ${PINS.filter((c) => c.mainsAgreeWithCSS).length} case internal/pinfixture says
    they agree and differing in the ${PINS.filter((c) => !c.mainsAgreeWithCSS).length}
    it says they differ, and charging the spacing a flex line charges where a
    Compose Row charges none`);
}

await main();
