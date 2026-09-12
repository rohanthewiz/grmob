// An element's inline style for dom.mjs, with CSS shorthands expanded into
// their longhands the way a real CSSStyleDeclaration does.
//
// # Why a plain object was not enough
//
// dom.mjs used to give every element `style = {}`. The runtime only ever
// assigns onto it and never reads the cascade back, so that looked like a
// faithful model of the bookkeeping — and for independent properties it is.
// It is wrong for exactly one relationship: a shorthand and its longhands are
// two views of the same declarations. In a browser
//
//	el.style.gap = "12px";       // row-gap: 12px; column-gap: 12px
//	el.style.rowGap = "";        // removes row-gap
//	el.style.columnGap = "";     // removes column-gap — the gap is gone
//
// while on a plain object `gap` is still "12px" after all three. styleFromGrMob
// is total — it restates every property it manages and writes "" for the unset
// ones — so it is precisely the code that assigns a shorthand and then clears
// its longhands in the same object. It did that twice: `gap` then
// rowGap/columnGap erased every core.Gap on the web, and `overflow` then a ""
// overflowX erased half of a TextGrid's Overflow. The replay harness agreed
// with Go both times, because this class of bug is invisible by construction
// to a model with no expansion.
//
// # The model
//
// Longhand values are the only thing stored. Assigning a shorthand expands it
// into its longhands (recursively: border → borderTop → borderTopColor);
// reading one composes it back from them. Three rules make the reads honest:
//
//	all longhands never written  → undefined  (see "Deliberate differences")
//	any longhand missing or ""   → ""         (a browser cannot serialize it)
//	otherwise                    → the shorthand text as the author wrote it,
//	                               if no longhand has changed since; else a
//	                               serialization of the longhands
//
// The expansion rules come from CSS itself, and the ones the tests lean on are
// pinned against a real Chrome by CSSOM_READS below: shorthand_test.mjs holds
// this model to that table, and browser.mjs check 14 holds the table to Chrome.
//
// # Deliberate differences from a browser, each because a test depends on it
//
//   - A property nothing ever wrote reads `undefined`, not "". The suites
//     distinguish "the runtime never touched this" from "the runtime cleared
//     it", which is the same reason dom.mjs starts `value` and `checked` as
//     undefined rather than at a browser default.
//   - A shorthand whose longhands still hold what it set reads back as the
//     text that set it ("0px 16px 0px 16px"), where Chrome returns its own
//     shortest form ("0px 16px"). The runtime never reads a shorthand back for
//     its value, and the suites assert on what the runtime wrote.
//   - Values are not validated or canonicalized: "#FF0000" stays "#FF0000"
//     rather than becoming rgb(255, 0, 0), and an invalid value is stored
//     rather than ignored.
//
// # What it refuses to guess
//
// A shorthand value outside the forms modeled here (a two-radius
// border-radius, a font other than a CSS-wide keyword) throws instead of
// expanding into something plausible. So does an element given both an
// UNMODELED shorthand and one of its longhands — the one situation where a
// missing expansion would change an answer. Either is a signal to extend this
// file, and a loud one, which is the same stance dom.mjs takes throughout:
// a shim that quietly lies is worse than no shim.

/**
 * CSSOM_READS is what a real CSSStyleDeclaration answers, as rows of
 * assignment sequences and the reads that follow them.
 *
 * It lives beside the model rather than in the model's test because it has two
 * readers: shorthand_test.mjs replays every row against makeStyle, and
 * browser.mjs check 14 replays every row against a fresh element's style in
 * headless Chrome. The rows were first taken by hand in a Chrome console; the
 * second reader is what keeps them true.
 *
 * The reads are chosen where Chrome and the model are meant to agree exactly:
 * longhand values, the empty string a browser returns for a shorthand it can no
 * longer serialize, and serializations after a longhand changed. A shorthand
 * read back unchanged is left out on purpose — Chrome returns its own shortest
 * spelling and the model returns the author's text (see "Deliberate
 * differences" above).
 */
export const CSSOM_READS = [
    // The two bugs, as the CSSOM sees them.
    { sets: [["gap", "12px"], ["rowGap", ""]], reads: { gap: "", rowGap: "", columnGap: "12px" } },
    { sets: [["gap", "12px"], ["rowGap", ""], ["columnGap", ""]], reads: { gap: "" } },
    { sets: [["overflow", "hidden"], ["overflowX", ""]], reads: { overflow: "", overflowX: "", overflowY: "hidden" } },

    { sets: [["overflow", "hidden"], ["overflowX", "auto"]], reads: { overflow: "auto hidden", overflowY: "hidden" } },
    { sets: [["overflow", "hidden"], ["overflowX", "hidden"]], reads: { overflow: "hidden" } },
    { sets: [["padding", "1px 2px"], ["paddingLeft", "5px"]], reads: { padding: "1px 2px 1px 5px", paddingTop: "1px", paddingRight: "2px", paddingBottom: "1px" } },
    { sets: [["padding", "1px"], ["paddingTop", ""]], reads: { padding: "", paddingRight: "1px" } },
    { sets: [["padding", "1px 2px"], ["paddingLeft", "2px"]], reads: { padding: "1px 2px" } },
    { sets: [["margin", "1px 2px 3px"]], reads: { marginLeft: "2px", marginBottom: "3px", marginRight: "2px" } },
    { sets: [["inset", "1px"], ["left", ""]], reads: { inset: "", top: "1px" } },
    { sets: [["border", "1px solid red"], ["borderBottomColor", "blue"]], reads: { border: "", borderBottom: "1px solid blue", borderTopWidth: "1px", borderTopStyle: "solid", borderBottomColor: "blue" } },
    // The tab strip's own sequence (TAB_STYLE in grmob-runtime.js).
    { sets: [["border", "none"], ["borderBottom", "2px solid transparent"]], reads: { borderTopStyle: "none", borderTopWidth: "medium", borderTopColor: "currentcolor", borderBottomWidth: "2px", border: "" } },
    { sets: [["border", "1px solid red"], ["border", ""]], reads: { borderTop: "", borderLeftColor: "" } },
    { sets: [["borderTop", "2px solid"]], reads: { borderTopColor: "currentcolor", borderTopWidth: "2px" } },
    { sets: [["borderRadius", "8px"], ["borderTopLeftRadius", ""]], reads: { borderRadius: "", borderBottomRightRadius: "8px" } },
    { sets: [["flex", "1 1 0px"], ["flexGrow", "2"]], reads: { flex: "2 1 0px", flexShrink: "1", flexBasis: "0px" } },
    { sets: [["flex", "none"]], reads: { flexGrow: "0", flexShrink: "0", flexBasis: "auto" } },
    // The overlay's stamp (OVERLAY_CHILD_AREA).
    { sets: [["gridArea", "1/1"]], reads: { gridRowStart: "1", gridColumnStart: "1", gridRowEnd: "auto", gridColumnEnd: "auto" } },
    // The code buffer's and the tab's `font: inherit` followed by a longhand.
    { sets: [["font", "inherit"], ["fontWeight", "600"]], reads: { font: "", fontSize: "inherit", fontWeight: "600" } },
    // The code editor's gutter handing padding-left back (syncCodeGutter):
    // the author's padding, the gutter's inset over its left side, then the
    // author's left value restored.
    { sets: [["padding", "1px 2px 3px 7px"], ["paddingLeft", "3ch"], ["paddingLeft", "7px"]], reads: { padding: "1px 2px 3px 7px", paddingLeft: "7px", paddingTop: "1px" } },
];

const CSS_WIDE = new Set(["inherit", "initial", "unset", "revert", "revert-layer"]);

// tokens splits a value on whitespace outside parentheses, so a color such as
// "rgba(128, 128, 128, 0.35)" is one token.
function tokens(value) {
    const out = [];
    let depth = 0;
    let cur = "";
    for (const ch of value.trim()) {
        if (ch === "(") depth++;
        if (ch === ")") depth--;
        if (depth === 0 && /\s/.test(ch)) {
            if (cur) out.push(cur);
            cur = "";
            continue;
        }
        cur += ch;
    }
    if (cur) out.push(cur);
    return out;
}

function refuse(name, value, why) {
    return new Error(
        `cssstyle.mjs cannot expand ${name}: ${JSON.stringify(value)} (${why}). ` +
        `Model it in SHORTHANDS rather than trusting a readback this file would have to guess.`,
    );
}

// --- shorthand specs ---------------------------------------------------------
//
// Each spec names its immediate longhands (which may be shorthands
// themselves), splits a non-empty, non-keyword value into one value per
// longhand, and joins longhand values back into the shorthand's text.

const SIDES = ["Top", "Right", "Bottom", "Left"];

// box is the 1–4 value pattern: top, right, bottom, left, where a missing
// right copies top, a missing bottom copies top, and a missing left copies
// right. Serialized back to the shortest form that says the same thing, as
// the CSSOM serializes it.
function box(name, longhands) {
    return {
        longhands,
        expand(value) {
            const t = tokens(value);
            if (t.length > 4 || value.includes("/")) {
                throw refuse(name, value, "only the one-to-four value form is modeled");
            }
            const [top, right = top, bottom = top, left = right] = t;
            return [top, right, bottom, left];
        },
        serialize([top, right, bottom, left]) {
            if (left !== right) return `${top} ${right} ${bottom} ${left}`;
            if (bottom !== top) return `${top} ${right} ${bottom}`;
            if (right !== top) return `${top} ${right}`;
            return top;
        },
    };
}

// pair is the 1–2 value pattern of gap (row, column) and overflow (x, y):
// one value applies to both.
function pair(name, longhands) {
    return {
        longhands,
        expand(value) {
            const t = tokens(value);
            if (t.length > 2) throw refuse(name, value, "expected one or two values");
            const [a, b = a] = t;
            return [a, b];
        },
        serialize([a, b]) {
            return a === b ? a : `${a} ${b}`;
        },
    };
}

const BORDER_STYLES = new Set([
    "none", "hidden", "dotted", "dashed", "solid", "double", "groove", "ridge", "inset", "outset",
]);
const isBorderWidth = (t) => /^(thin|medium|thick)$/.test(t) || /^-?[\d.]+[a-z%]*$/i.test(t);
// The initial values a border side shorthand resets the parts it does not
// mention to.
const BORDER_INITIAL = ["medium", "none", "currentcolor"];

// borderParts parses "<width> <style> <color>" in any order, each optional.
function borderParts(name, value) {
    const parts = [...BORDER_INITIAL];
    const seen = [false, false, false];
    for (const t of tokens(value)) {
        const i = BORDER_STYLES.has(t) ? 1 : isBorderWidth(t) ? 0 : 2;
        if (seen[i]) throw refuse(name, value, "two values for the same part of a border");
        seen[i] = true;
        parts[i] = t;
    }
    return parts;
}

function borderSide(side) {
    const name = `border${side}`;
    return {
        longhands: [`border${side}Width`, `border${side}Style`, `border${side}Color`],
        expand: (value) => borderParts(name, value),
        // Parts at their initial value are left out. How a browser spells a
        // side whose parts are ALL initial varies; nothing reads that case.
        serialize(parts) {
            const said = parts.filter((p, i) => p !== BORDER_INITIAL[i]);
            return said.length ? said.join(" ") : "none";
        },
    };
}

const SHORTHANDS = {
    gap: pair("gap", ["rowGap", "columnGap"]),
    overflow: pair("overflow", ["overflowX", "overflowY"]),
    padding: box("padding", SIDES.map((s) => `padding${s}`)),
    margin: box("margin", SIDES.map((s) => `margin${s}`)),
    inset: box("inset", ["top", "right", "bottom", "left"]),
    borderWidth: box("borderWidth", SIDES.map((s) => `border${s}Width`)),
    borderStyle: box("borderStyle", SIDES.map((s) => `border${s}Style`)),
    borderColor: box("borderColor", SIDES.map((s) => `border${s}Color`)),
    borderRadius: box("borderRadius", [
        "borderTopLeftRadius", "borderTopRightRadius", "borderBottomRightRadius", "borderBottomLeftRadius",
    ]),
    borderTop: borderSide("Top"),
    borderRight: borderSide("Right"),
    borderBottom: borderSide("Bottom"),
    borderLeft: borderSide("Left"),
    border: {
        longhands: SIDES.map((s) => `border${s}`),
        expand(value) {
            borderParts("border", value); // validates; every side gets the same text
            return SIDES.map(() => value);
        },
        // All four sides alike or no single `border` can say it.
        serialize: (sides) => (sides.every((s) => s === sides[0]) ? sides[0] : ""),
    },
    flex: {
        longhands: ["flexGrow", "flexShrink", "flexBasis"],
        expand(value) {
            const t = tokens(value);
            const num = (x) => /^[\d.]+$/.test(x);
            if (t.length === 1 && t[0] === "none") return ["0", "0", "auto"];
            if (t.length === 1 && t[0] === "auto") return ["1", "1", "auto"];
            // A bare number is flex-grow with shrink 1 and a zero basis.
            if (t.length === 1) return num(t[0]) ? [t[0], "1", "0%"] : ["1", "1", t[0]];
            if (t.length === 2) return num(t[1]) ? [t[0], t[1], "0%"] : [t[0], "1", t[1]];
            if (t.length === 3) return t;
            throw refuse("flex", value, "expected one to three values");
        },
        serialize: (parts) => parts.join(" "),
    },
    gridArea: {
        longhands: ["gridRowStart", "gridColumnStart", "gridRowEnd", "gridColumnEnd"],
        expand(value) {
            const t = value.split("/").map((s) => s.trim());
            if (t.length > 4 || t.some((s) => s === "")) throw refuse("gridArea", value, "expected one to four lines");
            // An omitted line copies a named line from its opposite side and is
            // `auto` otherwise (numbers are not names).
            const named = (x) => x !== undefined && /^[a-z_-][\w-]*$/i.test(x) && x !== "auto" && x !== "span";
            const [rs, cs = named(rs) ? rs : "auto", re = named(rs) ? rs : "auto", ce = named(cs) ? cs : "auto"] = t;
            return [rs, cs, re, ce];
        },
        serialize: (lines) => lines.join(" / "),
    },
    // Only the CSS-wide keywords ("inherit" is the one the runtime writes).
    // The full grammar — style, weight and size positions, a size/line-height
    // pair, a family list — is a parser this harness has no use for.
    font: {
        longhands: ["fontStyle", "fontVariant", "fontWeight", "fontStretch", "fontSize", "lineHeight", "fontFamily"],
        expand(value) {
            throw refuse("font", value, "only CSS-wide keywords are modeled");
        },
        serialize: () => "",
    },
};

// Shorthands this file does not expand, with their longhands. Kept only to
// notice when an element is given one of these AND one of its longhands —
// then the missing expansion would change an answer, and the write throws.
const UNMODELED = {
    background: ["backgroundColor", "backgroundImage", "backgroundPosition", "backgroundSize",
        "backgroundRepeat", "backgroundAttachment", "backgroundOrigin", "backgroundClip"],
    outline: ["outlineColor", "outlineStyle", "outlineWidth"],
    textDecoration: ["textDecorationLine", "textDecorationColor", "textDecorationStyle", "textDecorationThickness"],
    transition: ["transitionProperty", "transitionDuration", "transitionTimingFunction", "transitionDelay", "transitionBehavior"],
    animation: ["animationName", "animationDuration", "animationTimingFunction", "animationDelay",
        "animationIterationCount", "animationDirection", "animationFillMode", "animationPlayState"],
    flexFlow: ["flexDirection", "flexWrap"],
    placeItems: ["alignItems", "justifyItems"],
    placeContent: ["alignContent", "justifyContent"],
    placeSelf: ["alignSelf", "justifySelf"],
    gridRow: ["gridRowStart", "gridRowEnd"],
    gridColumn: ["gridColumnStart", "gridColumnEnd"],
    overscrollBehavior: ["overscrollBehaviorX", "overscrollBehaviorY"],
    listStyle: ["listStyleType", "listStylePosition", "listStyleImage"],
    // The logical box properties map onto physical sides by writing mode,
    // which is layout, which this harness does not have.
    paddingInline: ["paddingLeft", "paddingRight"],
    paddingBlock: ["paddingTop", "paddingBottom"],
    marginInline: ["marginLeft", "marginRight"],
    marginBlock: ["marginTop", "marginBottom"],
};

// leafNames flattens a property to the stored longhands it covers.
function leafNames(name) {
    const spec = SHORTHANDS[name];
    return spec ? spec.longhands.flatMap(leafNames) : [name];
}

/** isShorthand reports whether reads and writes of name are expanded here. */
export function isShorthand(name) {
    return Object.hasOwn(SHORTHANDS, name);
}

// WRITES is the key the proxy answers with its record of every assignment, for
// erasedShorthands below; a Symbol so no CSS property name can collide with it.
const WRITES = Symbol("writes");

/**
 * erasedShorthands lists the shorthands last assigned a real value that no
 * longer read back as one because a later assignment set one of their
 * longhands to "" — the gap and TextGrid overflow bugs, stated as a query.
 * A longhand given a different real value (border then borderBottom) is an
 * override, not an erasure, and is not reported.
 */
export function erasedShorthands(style) {
    const { written, leaves } = style[WRITES];
    const out = [];
    for (const [name, value] of written) {
        if (!isShorthand(name) || value.trim() === "") continue;
        if (leafNames(name).some((l) => leaves.get(l) === "")) out.push(name);
    }
    return out;
}

/** makeStyle returns a new, empty inline style. */
export function makeStyle() {
    const leaves = new Map(); // stored longhand (or unexpanded property) → value
    const written = new Map(); // every property assigned, in first-write order → last value
    const texts = new Map(); // shorthand → { text, snap }: what set it, and the leaves it left

    const snapshot = (name) => leafNames(name).map((l) => String(leaves.get(l))).join(" ");

    function assign(name, value) {
        const spec = SHORTHANDS[name];
        if (!spec) {
            leaves.set(name, value);
            return;
        }
        const v = value.trim();
        let parts;
        if (v === "") parts = spec.longhands.map(() => "");
        else if (CSS_WIDE.has(v)) parts = spec.longhands.map(() => v);
        else parts = spec.expand(value);
        spec.longhands.forEach((l, i) => assign(l, parts[i]));
        texts.set(name, { text: value, snap: snapshot(name) });
    }

    function read(name) {
        const spec = SHORTHANDS[name];
        if (!spec) return leaves.get(name);
        const ls = leafNames(name).map((l) => leaves.get(l));
        if (ls.every((x) => x === undefined)) return undefined;
        if (ls.some((x) => x === undefined || x === "")) return "";
        const rec = texts.get(name);
        if (rec && rec.snap === snapshot(name)) return rec.text;
        const parts = spec.longhands.map(read);
        if (parts.some((p) => p === "")) return "";
        // A CSS-wide keyword serializes only when every part says it.
        if (parts.some((p) => CSS_WIDE.has(p))) {
            return parts.every((p) => p === parts[0]) ? parts[0] : "";
        }
        return spec.serialize(parts);
    }

    function refuseUnmodeledMix(name) {
        for (const [sh, longs] of Object.entries(UNMODELED)) {
            const other = name === sh ? longs.find((l) => written.has(l)) : longs.includes(name) && written.has(sh) ? sh : null;
            if (other) {
                throw new Error(
                    `cssstyle.mjs: this element was given both ${sh} and its longhand ` +
                    `${name === sh ? other : name}, and ${sh} is not expanded here, so what either ` +
                    `reads back would be a guess. Model ${sh} in SHORTHANDS.`,
                );
            }
        }
    }

    return new Proxy({}, {
        get(_, name) {
            if (name === WRITES) return { written, leaves };
            return typeof name === "string" ? read(name) : undefined;
        },
        set(_, name, value) {
            if (typeof name !== "string") return false;
            // A CSSStyleDeclaration stringifies what it is given; null clears.
            const v = value === null ? "" : String(value);
            refuseUnmodeledMix(name);
            assign(name, v);
            written.delete(name);
            written.set(name, v);
            return true;
        },
        has(_, name) {
            return typeof name === "string" && read(name) !== undefined;
        },
        // Enumeration is the properties the runtime assigned, as on the plain
        // object this replaced, so Object.keys(el.style) still answers "what
        // did the runtime write" (totality_test.mjs sweeps by it).
        ownKeys() {
            return [...written.keys()];
        },
        getOwnPropertyDescriptor(_, name) {
            if (typeof name !== "string" || !written.has(name)) return undefined;
            return { enumerable: true, configurable: true, writable: true, value: read(name) };
        },
    });
}
