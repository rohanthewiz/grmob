# A labelled group stops repeating itself on Compose

**Session:** 8f64c69c-386f-4c67-a2c4-72f8aafce0ff
**Date:** 2026-09-19 21:46
**Branch:** master (4eac758 → this commit)

## The ask

"Let's work on N-055 and N-056", then `/sw`.

## Finding: they were one bug

N-055 (a `comps.Stepper` reading "2, Guests, 2") and N-056 (a `comps.Drawer`
panel reading "Notebook, Notebook") were filed as two low-value curiosities
with a note that the second had "the same shape" as the first. They are not
two shapes; they are one mechanism seen through two widgets.

`boxModifier` gives a node with an `AccessibilityLabel`
`semantics(mergeDescendants = true)`. The comment there says it means the name
to *replace* the content, the way `aria-label` does on the web and a label
after `.combine` does in SwiftUI. Compose does not read it that way. A merging
node with children emits its own `contentDescription` as a **fake child**
beside the Texts rather than in place of them (the note on
`LocalGrMobNamedControl` had already established this for controls), and a
stated `AccessibilityValue` becomes a `stateDescription` TalkBack reads as
well. So any child Text that repeats the label or the value is spoken twice:

	  comps.Stepper  label "Guests", value "2", a Text "2" between the buttons
	    "2, Guests, 2"
	  comps.Drawer   label "Notebook", a Text "Notebook" in its header
	    "Notebook, Notebook"

## Why the repair is not in `comps`

Neither widget can drop its Text:

- The web scopes `aria-value*` to `progressbar` and drops a group's value
  entirely, so the number between the Stepper's buttons is the **only** place
  the web hears it. (This is the same finding that moved
  `progressBarRangeInfo` behind `core.RoleProgressBar` in a previous session —
  see the note on `grMobValue`.)
- A Drawer's title is a visible heading, and the landmark is named by what the
  eye reads.

iOS reads a labelled container's name in place of its contents, so it never
had the problem. Compose is the one target that reads a merged label this way,
which is where the fix belongs.

## Why it is not `LocalGrMobNamedControl`

That local already exists and already solves the *control* half: a named
control's label stands for its whole content, so everything under it goes
quiet. Its own doc explains why a label alone does not open it — "a labelled
group or region keeps its content readable, which is why a `comps.Stepper`
still says its number". That reasoning is right and is left intact. A group's
content is introduced by its name, not replaced by it; only the part that
*repeats* the name is redundant.

## What was built

### `LocalGrMobGroupSaid` (`android/…/runtime/Renderer.kt`)

A `CompositionLocal<Set<String>>` carrying what the nearest labelled ancestor
already puts into the announcement.

- **`spokenByLabel(node)`** returns that node's own label plus, when stated,
  its `AccessibilityValue.text`, each folded with `trim().lowercase()`. Empty
  when the node states no label — which is also exactly when `boxModifier`
  does not merge, so nothing of the node's is read with its children's.
  Folding is by ear, not by byte: a reader speaks "Notebook" and "notebook "
  identically.
- **`RenderNode`** computes `said = spokenByLabel(node)` and provides it when
  it is non-empty and differs from the current scope. It joins the existing
  `buildList` of subtree locals, so an unchanged tree still provides nothing.
- **`GrMobText`** gains `echo`: true when the Text states no label of its own
  and its folded content is in the scope. It shares the one
  `clearAndSetSemantics { }` with the existing `quiet`.

**The scope is replaced, never latched.** This is the one design decision
worth spelling out, and it is the opposite of how the four boolean locals
beside it behave. Compose's own `mergeConfig` skips a child that merges its
descendants — the source comment reads "they're independently
screen-reader-focusable" — so a merge stops at the nearest merging node. A
nested labelled node's Texts are spoken under *its* name and have nothing to
do with an outer group's. A latched union would silence a Text that echoes a
group it was never merged into. `spokenByLabel` returning `emptySet()` for an
unlabelled node is what makes inheritance fall out of the same expression.

The guard `node.style?.accessibilityLabel.isNullOrEmpty()` in `GrMobText` is
there because a labelled node is inside the scope it just opened; without it a
labelled Text would silence itself. It is the same trap
`!namesItsContent(node)` avoids one line above.

## Verification

### Measured on the emulator, built both ways

The group node is not reachable by Tab, so the evidence is the accessibility
tree — which is precisely the thing TalkBack reads. Lesson 4.15 (Stepper) and
4.18 (Drawer), `uiautomator dump`, with the fixed renderer and then with
`git checkout` of `Renderer.kt` alone and a reinstall:

	  Stepper   before  'Guests'(desc) · Decrease · '2' TextView · Increase
	            after   'Guests'(desc) · Decrease · (empty View) · Increase
	  Drawer    before  'Notebook'(desc) · 'Notebook' TextView · Close · rows
	            after   'Notebook'(desc) · (empty View)        · Close · rows

`clearAndSetSemantics { }` leaves the node in the layout tree with nothing to
say, which is the empty View in the "after" column.

### What is *not* verified

The utterance itself. Tab reaches the Stepper's buttons but never the group
node, and TalkBack's own reading-order shortcuts ignore `adb shell input` —
confirmed again here for `input keycombination META_LEFT DPAD_RIGHT`, and for
an injected tap into the touch explorer. Hearing a group needs the Fold6's
virtual HID keyboard (Meta+Right), and the Fold6 was not attached. The tree is
what TalkBack speaks, so the change is sound, but the spoken line is unheard.
Worth a listen on the next device pass; folded into N-002.

The `stateDescription` carrying the Stepper's value is likewise invisible to
`uiautomator dump`. It is untouched by this change — it is written by
`grMobValue` on the group's own `boxModifier`, and only the Text child's
semantics were altered.

### Tests

- **`mobile/verify/group_echo_test.go`** (new) pins the rule in the Kotlin
  source, the way its sibling `named_control_test.go` does. Beyond the
  expression pins it asserts the *negative*: `LocalGrMobGroupSaid provides
  true` and `LocalGrMobGroupSaid.current + said` must not appear, because a
  latched or unioned scope is the failure mode the design rules out and a
  reader would not otherwise see that it had been considered.
- **`named_control_test.go`** updates the one pin the two halves now share
  (`if (quiet || echo)`), with a note pointing at the other test.
- **`comps/stepper_test.go`** and **`comps/drawer_test.go`** pin the Go-side
  premise: the drawn value equals the announced value, and the drawer's title
  is both the landmark's name and a drawn Text. The Compose rule fires on that
  *equality*, so a `Format` or a header that diverged would switch the repair
  off silently. Each assertion carries a pointer to `LocalGrMobGroupSaid`.
- `mobile/verify` was left reading native source as text; `doc.go` calls that
  a deliberate limit, so the premise checks went to `comps` rather than
  importing `comps` there.
- `:app:compileDebugKotlin`, `go build ./...`, `go vet ./...` and
  `go test ./...` all pass.

### An unrelated failure fixed on the way

`wasm/verify`'s `TestTheQuestionsOnTheSharedRepositoryParseAreTheOnesDecidedOn`
failed on a clean tree before this session's changes: five sentences quoted
"628 tracked Go files" against 629. The new test file makes it 630. The test's
own message says what to do and the figures were updated in
`repowalks_test.go` and `timings_test.go`. This is that arm working as
designed, not a bug.

## Housekeeping

The emulator was restored per the harness note: `accessibility_enabled 0`,
`enabled_accessibility_services null`, `com.google.android.tts` re-enabled,
and the scratch `uiautomator` dumps removed from `/sdcard`. The tutorial app
installed from this build remains on the emulator.

## Files

- `android/app/src/main/java/com/grmob/runtime/Renderer.kt`:
  `LocalGrMobGroupSaid`, `spokenByLabel`, the `said`/`says` pair in
  `RenderNode`, `echo` in `GrMobText`.
- `mobile/verify/group_echo_test.go` (new).
- `mobile/verify/named_control_test.go`: the shared pin.
- `comps/stepper_test.go`, `comps/drawer_test.go`: the premise ties.
- `wasm/verify/repowalks_test.go`, `wasm/verify/timings_test.go`: the count.
- `ai_docs/todo/next-list.md`: N-055 and N-056 closed.

## Next

Closed: N-055, N-056. Declined: None. Raised: None. Updated: N-002 (the
Stepper and Drawer utterances are now a listen-and-confirm on the Fold6, not a
fix). Full list: `ai_docs/todo/next-list.md`.
