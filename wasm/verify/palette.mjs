// The bundled palettes, as browser.mjs paints them.
//
// # Why this table exists
//
// core.ColorPalette.ControlBorder has WCAG 1.4.11's 3:1 floor under it, and
// components/variant_test.go measures the tone against every fill a control
// can be drawn on. That census is arithmetic over hex strings — it says
// #89898E is 3.12:1 on #F2F2F7 — and until this file nothing in the repository
// had ever *looked* at the result. Two retints deep, with a whole third
// palette added, every pass that touched those hexes was a contrast
// calculation, a type-check or a DOM shim.
//
// browser.mjs runs a real Chrome. So the pairs below are painted, screenshotted
// and read back pixel by pixel, and what the check asserts is that the colour
// Chrome put on the screen is the colour the census did its arithmetic about.
// That is the half no amount of Go can reach: an alpha channel, a blend mode,
// a hairline antialiased down to a tint, a colour profile — each of them makes
// the number true and the screen wrong.
//
// # It is pinned, not transcribed
//
// wasm/verify/palette_test.go rebuilds this table from core.BundledThemes()
// and internal/palette and fails if it has drifted, in either direction. A
// theme added to core without a row here is a palette the browser pass has
// never seen; a row here for a theme core does not have is a check painting
// colours nothing ships.
//
// The ratio is Go's own number, carried across so a failure can say what the
// census believed. Nothing in the browser recomputes it — a second WCAG
// implementation is the last thing a contrast floor needs.
//
// Each row is one (theme, subject) pair:
//
//	role       the theme's ControlBorder tone; ratio is 0, since a tone on its
//	           own has nothing to contrast with
//	backdrop   a fill the tone can be drawn on, with the census's ratio
export const PALETTES = [
    { theme: "AmberTheme", kind: "role", what: "ControlBorder", hex: "#8D6E63", ratio: 0.00 },
    { theme: "AmberTheme", kind: "backdrop", what: "Background", hex: "#FFFFFF", ratio: 4.62 },
    { theme: "AmberTheme", kind: "backdrop", what: "Surface", hex: "#FFF8E1", ratio: 4.35 },
    { theme: "AmberTheme", kind: "backdrop", what: "Card fill", hex: "#FFFFFF", ratio: 4.62 },
    { theme: "AmberTheme", kind: "backdrop", what: "Input fill", hex: "#FFF8E1", ratio: 4.35 },
    { theme: "AmberTheme", kind: "backdrop", what: "CheckBox fill", hex: "#FFFFFF", ratio: 4.62 },
    { theme: "AmberTheme", kind: "backdrop", what: "TextArea fill", hex: "#FFF8E1", ratio: 4.35 },
    { theme: "AmberTheme", kind: "backdrop", what: "Text fill", hex: "#FFFFFF", ratio: 4.62 },
    { theme: "DefaultTheme", kind: "role", what: "ControlBorder", hex: "#89898E", ratio: 0.00 },
    { theme: "DefaultTheme", kind: "backdrop", what: "Background", hex: "#FFFFFF", ratio: 3.48 },
    { theme: "DefaultTheme", kind: "backdrop", what: "Surface", hex: "#F2F2F7", ratio: 3.12 },
    { theme: "DefaultTheme", kind: "backdrop", what: "Card fill", hex: "#FFFFFF", ratio: 3.48 },
    { theme: "DefaultTheme", kind: "backdrop", what: "Input fill", hex: "#FFFFFF", ratio: 3.48 },
    { theme: "DefaultTheme", kind: "backdrop", what: "CheckBox fill", hex: "#FFFFFF", ratio: 3.48 },
    { theme: "DefaultTheme", kind: "backdrop", what: "TextArea fill", hex: "#FFFFFF", ratio: 3.48 },
    { theme: "DefaultTheme", kind: "backdrop", what: "Text fill", hex: "#FFFFFF", ratio: 3.48 },
    { theme: "MaterialTheme", kind: "role", what: "ControlBorder", hex: "#757575", ratio: 0.00 },
    { theme: "MaterialTheme", kind: "backdrop", what: "Background", hex: "#FFFFFF", ratio: 4.61 },
    { theme: "MaterialTheme", kind: "backdrop", what: "Surface", hex: "#F5F5F5", ratio: 4.23 },
    { theme: "MaterialTheme", kind: "backdrop", what: "Card fill", hex: "#FFFFFF", ratio: 4.61 },
    { theme: "MaterialTheme", kind: "backdrop", what: "Input fill", hex: "#FAFAFA", ratio: 4.41 },
    { theme: "MaterialTheme", kind: "backdrop", what: "TextArea fill", hex: "#FAFAFA", ratio: 4.41 },
];
