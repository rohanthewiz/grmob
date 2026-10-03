# Safe-area insets on the web: N-061 declined, N-088 raised and fixed

Session: `86be6cd1-e285-4b27-b11e-44df367a3edf`

## Ask

Two items from the cats-todo backlog, one after the other:

1. N-061, "the browser reports no safe-area insets", with the 2026-09-29
   recommendation to make it a non-goal.
2. N-088, raised by (1): the tutorial page sets `viewport-fit=cover` and pads
   nothing.

## N-061: declined (committed as `908839a`)

`Window.Insets` stays zero in the browser. Zero is the true answer for a page
in a browser window. The one wrong case is a page drawn behind a notch with
`viewport-fit=cover` (usually an installed PWA). Reading the values back costs
a probe element and a `getComputedStyle`, a forced layout, on every report.
The page that opts into cover owns its HTML anyway: a CSS padding of
`env(safe-area-inset-*)` on the element hosting the app keeps the content
clear, and no number has to reach Go.

- `core/window.go`: `SafeInsets`' first line said a browser reports "whatever
  the platform puts over the viewport", which contradicted the per-host table
  under it (Browser: zero). Fixed, and a paragraph added giving the non-goal
  and its reason. `docs/api/core-device.md` regenerated
  (`go run ./internal/apidoc/gen`).
- `wasm/grmob-runtime.js`: the windowMetrics comment names N-061.

Re-checking the recommendation's premise ("no bundled page is a PWA") turned
up `wasm/index.html`'s `viewport-fit=cover` with no `env()` anywhere on the
web side. That became N-088.

## N-088: fixed

### Measured first

Measured in Safari on the booted iPhone 17 Pro simulator (iOS 26.5, not
standalone), with two scratch probe pages that read `env()` back through a
hidden padded element. Insets are top/right/bottom/left, in px:

| | with cover | without cover |
|---|---|---|
| portrait | 0/0/0/0 | 0/0/0/0 |
| landscape | 0/62/20/62 | 0/0/20/0 (sides letterboxed) |

- Portrait: the item's worry about content under the home indicator was
  unfounded. On iOS 26 the body ends above Safari's floating toolbar either
  way.
- Landscape with cover: the tutorial's header ran under the Dynamic Island,
  and the island hid the "Docs" link.
- Without cover, Safari still reports the 20px bottom inset and does not pad
  for it, so dropping cover would not have removed the need for padding.

### The fix

`wasm/index.html`, on `body`:

    padding: env(safe-area-inset-top) env(safe-area-inset-right)
             env(safe-area-inset-bottom) env(safe-area-inset-left);

One rule on body rather than one per region. The header, the bezel, the
bezel-less layout under 520px and the split all sit inside body, and each has
its own padding story (main's padding is 0 in two of the three layouts).
`* { box-sizing: border-box }` keeps the padded body at 100% height. Cover is
kept, so body's background paints behind the insets.

- Landscape after: the header clears the island on both sides, "Docs" is
  visible, and the page sits clear of the home indicator.
- Portrait after: the same as before.
- Cost: in landscape the header's `--panel` band no longer reaches the side
  edges; the 62px strips are `--bg`, as Safari's own letterboxing would be.
- The runtime comment from N-061 that said the tutorial lacked this padding
  now says it has it.

### Driving rotation

osascript has no assistive access on this machine, so clicking Simulator's
Device menu fails with -1719. Rotation was done instead by a throwaway
xcodegen project in the scratchpad: a Host app plus a `bundle.ui-testing`
target with `CODE_SIGNING_ALLOWED: NO`. Its test only sets
`XCUIDevice.shared.orientation`, which persists after the test, so
`simctl openurl` and `simctl io screenshot` then drive Safari in landscape.
`XCUIApplication(bundleIdentifier: "com.apple.mobilesafari").open(url)` failed
with "has not loaded accessibility". Landscape screenshots come out in
device-portrait coordinates and need `sips -r`. This recipe is in the iOS
simulator memory.

The tutorial was served by a separate `go run ./serve -addr :8093`, because
something else was already listening on :8080. It and the probe server were
stopped afterwards.

## Verification

- N-061: `go test ./core/ ./internal/... ./cmd/grmob/` passed, including the
  API-doc freshness test. `node --check wasm/grmob-runtime.js` passed.
- N-088: simulator screenshots in portrait and landscape, before and after.
  `./wasm/verify/run.sh` exit 0, and `node --check` on the runtime passed.
- Not seen on a physical iPhone.

## Next

Closed: N-088. Declined: N-061. Raised: N-088. Deferred: None.
Promoted: None. Updated: None. Full list: `ai_docs/todo/next-list.md`.
