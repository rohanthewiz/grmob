# Session: the role prop, seen from its first consumer

Session: https://claude.ai/code/session_01AM7rDTyaXrFQ4X6rPDdUWp
Date: 2026-09-05 (follows "chip-inversion-and-role-prop", same session)

## Ask

"Now adopt both downstream in church_mobile" — the Chip inversion and
`core.AccessibilityRole`, immediately after they landed here.

Most of the diff is in `../church/church_mobile` (branch `roh/use-grmob`,
commits `ee21dcc` + `b5e864e`, 22 files, +347/−164), whose own doc
`ai_docs/claude_sessions/2026-0905-1303-adopt-chip-look-and-roles.md` carries
the detail. What belongs here is the two things the adoption sent *back*, and
what the widget looked like from the other side.

## What came back upstream (commit `d5745de`)

- **`RoleLink`, the sixteenth role.** It was designed out for want of a
  consumer — "no consumer here, drop it" was the note. The first screen adopted
  had three: a scripture reference inside a summary, a phone number, an event's
  website, all `core.OpenURL`. Every hand-written `core.OnClick` in that app
  turned out to be a link.

  The lesson is narrower than "include everything". `RoleImage` was dropped for
  the same reason and is still right to drop, because `core.Image` is already
  an `<img>` on the web and an Image on both natives — the node type carries
  it. Nothing carries link-ness: `core.OpenURL` is a callback like any other,
  so a row that dials a number and a row that files a form are the same
  tappable `Box`. **The test for whether a role earns its place is whether any
  node type already implies it**, not whether a consumer has asked yet.

- **`GroupHeader`'s label is a heading.** The band titles a run of rows, and on
  a banded feed it is exactly what a reader navigating by heading wants once the
  screen's title has scrolled off. On the *label* and not the band, so the
  heading is named "March" rather than "March, 12" — the count badge stays the
  separate thing it is.

## The scorecard

    AppBar banner + heading  -> free, twelve screens, zero downstream lines
    Calendar day cells       -> free, 42 buttons per month
    GroupHeader heading      -> free once sent back
    RoleButton               -> the tab bar, and ~40 content rows via one helper
    RoleLink                 -> three sites, all core.OpenURL
    RoleHeading              -> section titles, month bands, detail + form titles
    RoleNavigation           -> the tab bar
    RoleTable/RowGroup/...   -> no consumer; that app has no DataTable
    RoleList/ListItem        -> refused, honestly (see below)
    RoleToolbar/Status/Alert -> no consumer downstream this pass

## What the adoption proved about the design

- **"Roles are never inferred" is the rule that paid.** The single
  highest-value line downstream is one conditional in the app's `contentRow`
  helper, turning some forty tappable rows into announced controls. It could
  not have been `components.ListRow`'s own decision: a row that pushes a
  screen, one that opens a browser and one that toggles a setting are the same
  widget with the same handler. The widget knows it has a tap; only the caller
  knows what kind of tap it is. That is the doc's argument, arrived at from the
  consumer's side.

- **The ownership rule bites in a second place.** `pagedList` downstream cannot
  claim `RoleList`, because its "Load more" footer sits *inside* the
  `core.List` and a list role there would own a non-`listitem` child. That is
  the same problem `DataTable`'s rowgroup documents, met from the other
  direction and by an app that never read that comment. The pattern is general:
  **a container that mixes items with chrome cannot take a collection role**,
  and it is worth saying once in `core.Role`'s own doc rather than twice in
  widgets.

- **Sixteen values, four used downstream.** The tabular five and the collection
  pair went unused because that app has no table and its lists carry footers.
  That is not waste — the table set is exercised by `DataTable`'s own tests and
  by the tutorial — but it does mean the *landmarks* are the half a screen
  actually reaches for, and the half whose natives do nothing.

## Friction found (downstream, recorded here because it is grmob's)

- **No `tab` / `tablist`.** The app's nav bar is a hand-rolled tab set and got
  `navigation` + five `button`s. A real tab role needs a tablist that reports
  *which member is selected*, which is a state question rather than a role one —
  `aria-selected` has no home on `Style`, and both natives express it
  differently again. The app still appends ", selected" to the name, as it did
  before roles existed.

- **No `dialog`.** `confirmDialog` announces as a container with a heading
  inside it. This one probably belongs to `core.Modal` — it is a node type, so
  the renderers could give it the role without the author asking, the way a
  Button is already a `<button>`.

- **No heading level.** A screen with a bar title over month bands has two
  tiers and produces a flat list of headings. Not worth a field until something
  has three, but the shape of the fix (a `Level int` beside the role, or
  `RoleHeading2`) is worth deciding before a consumer invents one.

- **No live region for a chat log.** ARIA's `log` is the case; `RoleStatus` is
  the nearest thing here and is not the same promise.

## Verification

Upstream: `go build`, `go vet`, `gofmt`, `go test ./...`, `-race`,
`wasm/verify/run.sh`, `ios/verify/run.sh`, `GOOS=js` build — all clean. The
native role pins forced both `RoleLink` arms, as designed.

Downstream: the same Go set plus `-race`, and all 15 screen goldens
re-recorded and read hunk by hunk — every change is a `role=` attribute or the
chip inversion, nothing else moved.

## Next

1. `core.Role` doc: state the container-mixes-items-with-chrome rule once,
   where both `DataTable` and a hand-written list can find it.
2. Decide the heading-level question before a consumer invents one.
3. `core.Modal` could carry `dialog` itself, the way Button carries `button`.
4. The `<button>` user-agent border divergence on the two DOM renderers
   (`EmphasisGhost` draws a rule on the web and not on the natives) — visible
   downstream, since every header action is a ghost button.
5. `Calendar.Deselectable` and a counted `Marked` (carried over).
6. Small: "emit `OnEndReached` before the children" in its doc (carried over).
7. Small: the mid-list busy case in `EmptyState`'s doc (carried over).
8. Then Tier C: heading plumbing + `Rotate` + Compass.
