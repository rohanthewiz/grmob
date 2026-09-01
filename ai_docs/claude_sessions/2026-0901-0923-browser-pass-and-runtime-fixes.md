# Tutorial browser pass: master merge + six WASM runtime fixes

Session: https://claude.ai/code/session_01RqyVsuqXEL719QsPhLf8ht

## Goal

The Phase 8 follow-up: walk all 40 tutorial lessons in a real browser.
The pass doubled as a shakedown of the wasm runtime — it found six real
bugs, fixed them, and absorbed master's 13-commit runtime-hardening
series mid-way.

## How the pass was run

`GOOS=js GOARCH=wasm go build -o wasm/main.wasm ./wasm`, then
`python3 -m http.server 8080` from wasm/ (left running), driven through
claude-in-chrome. Small controls are unreliable to hit by screenshot
coordinates — the working pattern is `javascript_tool` for clicks/queries
(`.click()`, `getComputedStyle`, dataset inspection) plus `computer type`
for real keystrokes into focused inputs, with screenshots for eyeballing.
Measure the element that owns the style: two probes (7.2's radius, 7.5's
transition) read the demoPanel/Text wrapper instead of the styled node
and looked like false failures.

### The two-machines gotcha

Mid-session every localhost navigation started returning
`chrome-error://chromewebdata` while curl saw 200. Cause: a second
claude-in-chrome extension connected from ANOTHER machine ("WorkLaptop")
and the session silently switched to it — localhost there has no server.
`list_connected_browsers` / `select_browser` (Browser 2 = the MacBook Air
M3) fixed it. If localhost breaks while curl works, check which browser
is selected first.

## Merge of master (commit e4f45e4)

Master's 13 commits overlapped the first round of runtime fixes
(inputTypeFor, checkbox `checked`, payload-type normalization, the
wasm/verify node harness, alignment tables). Resolution: kept BOTH sides
in createElement (this branch's Modal chassis + master's inputTypeFor);
took master's object-literal tagForType wholesale — wasm/verify parses
those literals textually, so their exact shape is load-bearing.
Re-applied on top what master lacked; then the pass found more.

## The six runtime fixes (wasm/grmob-runtime.js)

1. **remove-child / add patches dropped** (merge commit): the Go diff
   emits "remove-child" for child-list shrink (reconcile/patch.go:131)
   and "add" for nil slots; patch() had no such cases. Navigating
   index → lesson left chapters 6–8's rows alive (clickable!) below the
   lesson body. "add" needs handling BEFORE the querySelector guard —
   its TargetID doesn't exist yet.
2. **Stack containers now always flex** (merge commit): a bare Column of
   Texts ran inline ("128Posts") because flex was gated on
   Gap/Justify/Align. STACK_CONTAINERS (Row/Column/Card/Box/Scroll/
   SafeArea/List/Fragment/Theme) get display:flex + native axis in
   createElement (for Style:null nodes) AND restated in styleFromGrMob.
3. **`label` prop ignored on update-props** (4e620b6): Buttons carry
   text as `label`; createElement mapped it, the patch path didn't — a
   navigation diff pairing two Buttons positionally kept the old caption
   (2.1's "+1" arrived reading "−", inherited from 1.5's stepper).
4. **Buttons stretched full-width** (4e620b6): flex columns default to
   cross-axis stretch; components.Button's contract is hug-by-default
   (FullWidth = block Display + Width). Fix: themed `Display:
   inline`/`inline-block` translates to `width: fit-content` — NOT
   `align-self: flex-start`, which would override an
   AlignItemsCenter Row and top-pin the button. Verified: 4.1's Save
   143px ↔ 1458px on the Full width toggle.
5. **Push channel armed** (4e620b6): the page now defines
   `GrMobApplyPatches`, so wasm/main.go's renderInitial attaches the
   manager's pump listener. Before: UseInterval's clock froze in hidden
   tabs — the IsDirty poll rides requestAnimationFrame, which Chrome
   suspends entirely when `document.visibilityState === "hidden"` (the
   Go ticker kept running; 36 ticks applied at once on a manual
   RenderAgain). The rAF poll loop stays as fallback.
6. **styleFromGrMob made total + FontWeight** (c4948ac): 7.2's own demo
   caught it — core.BorderRadius(0) left corners at 12px. update-style
   carries the WHOLE new Style, so absent/zero means "unset now", but
   every property sat behind `if (style.X)` and the reused element kept
   stale declarations. Every managed property is now assigned or ""
   (removal) each call. FontWeight added while there (200/400/700 are
   literal CSS values; bold was silently dropped on the web).
   Gotcha: wasm/verify pins the exact string
   `out.textAlign = textAlignFor(style.Align)` — kept total by relying
   on textAlignFor's `""` fallback for unset.

## What the browser pass verified (all interactions live)

Ch1–3 (earlier in session): checkbox recompose, axis flip, counter,
tap log, controlled input round-trip + UPPERCASE + Clear, Match/If,
keyed list add/remove (ids don't renumber), clock tick + pause,
UseTimeout fired-once vs pokes, UseEffect dep gating (2 runs, re-click
no refetch), UseMemo, UseReducer atomic pairs.

Ch4: variant/emphasis axes, Full width toggle real again, chips
(selected = muted Surface+Primary "pressed in" — by design, not
inverted), ListRow select/clear + FlexGrow spine, accordion, tabs
page-swap (the "page N of 3" badge is Info-page-only — probe
accordingly).

Ch5: failed submit → reveal → clean RSVP; OnBlur policy (blur event
reaches Go — the void-payload path works); cross-field mismatch →
matching submit → server error installed on Email WITH core.Focus ring;
"12x" → (0,false); Reset restores initials. Chrome autofilled 5.4's
password fields (type=password now real) — typed over, didn't read.

Ch6: push/pop discards frame state, lesson counter survives; Replace
holds depth 3 through checkout; PopToRoot unwinds 5→1 with "13 of 40
lessons opened" intact (session scope above the Navigator); modal
backdrop-dismiss via target-guard, checkbox alive across reopen; toast
layer draws above everything.

Ch7: Material Secondary #03DAC6; scoped WithTheme flips only the card
(Follow blue → rgb(98,0,238), Contents stays blue); 7.2 radius
14→0→14 + weight 700 after the totality fix; 800ms transition observed
mid-glide (blend color) then settled.

Ch8: boundary fallback shows the REAL panic ("index out of range [0]
with length 0"), heals on untick; handler panic → A:2/B:1 "Skewed by
1", repair works; debug demo records `[duplicate-key] ×2` live and
NOTHING while off (browser runs debug-off, so 8.1/8.2's earlier panics
recorded nothing — the zero-cost story demonstrated); Cached frozen
while Live advances 11s; "Take a bow 🎉" fires the completion toast.

## State / follow-ups

- Branch: worktree/grmob-interact-tutorial, 5 commits ahead of master
  (merge + 2 fix commits + this doc's). Full `go test ./...` and
  `wasm/verify/run.sh` green.
- Follow-up: htmlout also drops FontWeight — same one-line fix plus its
  snapshot tests; left out to keep the pass scoped.
- Follow-up: Spacer height and Image objectFit live outside
  styleFromGrMob (props, not Style) — an update-style on those nodes
  can't disturb them today, but any future "reset cssText" refactor
  would; the totality approach was chosen precisely to avoid that.
- The :8080 http.server may still be running for manual poking.
- Next: merge this branch to master.
