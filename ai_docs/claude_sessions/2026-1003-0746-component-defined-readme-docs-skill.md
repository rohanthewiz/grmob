# What a component is: defined in core.View, the README, the site docs and a skill

Session: `182bd6ae-f308-436a-ad48-d5cc85d50d42`

## Ask

From the cats-todo backlog: "Ratify what a component is and give instruction
in the README and site docs, and add a skill on how to make a component."
Then commit what remains, including the overlap from concurrent sessions,
with the session doc and the next-list saved and pushed.

## State found

- `ai_docs/SKILL-component.md` (`grmob-component`) was already drafted,
  untracked, and installed at `~/.claude/skills/grmob-component/` at 02:56.
  `ai_docs/SKILL.md` had uncommitted edits from 02:48: "Starting an app"
  (`grmob new` and its flags, the scaffold's files, the working loop), the
  widget table by `comps-*` topic, a Persistence section (bytdb, lazy open),
  the `core.Padding(0)` pitfall, and a description that points at
  grmob-component. All of it is committed here.
- A peer session (N-085, shell surface) had uncommitted work in the tree. It
  committed and pushed `6fda8ec` before this commit, so none of its files
  are in this one.

## The definition

It is stated in the doc comment on `core.View` (`core/view.go`), so
`docs/api/core-views.md` carries it too. Four words, used in these senses:

| Word | Means |
|---|---|
| View | the interface `Render(ctx) *Node`; each of the other three is one |
| Primitive | a core constructor whose node type a host draws itself; adding one touches every host |
| Component | a View written in Go out of other Views, with no host code; an app's screens are components |
| Widget | a component built for reuse: a struct with named fields keeping all six rules; every `comps` type is one |

A component takes one of three shapes: a function returning a View, a
`ComponentFunc`, or a struct with a `Render` method.

The contract:

```
correctness: every component, screens included
  1. Render builds a fresh tree; never mutate a Node after returning it
  2. hooks: none, or all before any branch, with the component rendered every pass;
     a widget owns state only when it is purely presentational
  3. never register a nil callback; call the caller's callbacks nil-safely
reuse: every widget, too
  4. look from ctx.Theme(); the caller's Style applied last
  5. role / name / state on the nodes; misuse reported with ReportConcern
  6. every field's zero value means something sensible
```

Anything that can't be written under these rules is a gap in core's
primitives (a node type, Style field or role), not a component.

Why the split into two groups: breaking rules 1–3 makes an app misbehave
(a reused pointer reads as "unchanged" to `Diff`, cursor drift, a nil-handler
panic on a native tap). Rules 4–6 are about reuse. An app's own screen may
hard-code a colour, but a widget must not.

## Changes

- `core/view.go`: doc comments on `View` (vocabulary, shapes, contract) and
  `ComponentFunc` (the `http.HandlerFunc` analogy).
- `docs/concepts/components.md` (new, in the nav after "Views &
  Composition"). It covers the four words, the three shapes with a decision
  diagram, the contract as a table of rule / what breaking it looks like, a
  worked leaf widget (`Tally`), field conventions, callbacks, a hook-owning
  widget (`Spoiler`), accessibility, testing (`renderDebug` plus
  `core.AuditTree`), when it is not a component, contributing to `comps` in
  short, and the skill for agents.
- Links to it from `docs/concepts/views.md` (the intro now uses the four
  words), `docs/components.md`'s idiom section, and `docs/index.md`'s "Where
  to go next".
- `README.md`: a new "Your own components" section after Events, with the
  vocabulary table, a short `Tally` that takes its pill colour from
  `comps.Variant` (a first draft had a literal `#FFFFFF`, which broke rule 4),
  the six rules, and links. Also a "What's in the box" bullet.
- `comps/doc.go`: "Package components" became "Package comps" (stale since the
  rename). "Two widgets do" take hooks was stale too: 16 files call
  `core.NewState` or `hooks.Use*` in code. It now names Accordion,
  DatePicker, PasswordField, ExpandableText and Snackbar, and says each one
  notes it in its doc comment. Menu and the clocks only mention hooks in
  usage examples, so they are not named.
- `ai_docs/SKILL-component.md`: the vocabulary table and the six-rule
  contract at the top, each rule pointing to its section. The installed
  copy was synced, keeping its fetch-latest header, which now names
  `core.View` as the canonical source.
- `ai_docs/SKILL.md` (and the installed copy): the Composition section uses
  the four words and points at grmob-component. It used to say a
  `ComponentFunc` gets "its own hook slots". It doesn't: `renderAll`
  renders children on the same context, so a ComponentFunc's hooks are
  claimed when the tree renders it, but still in the caller's positional
  sequence.
- `docs/api/core-views.md` and `docs/api/comps.md` were regenerated.

## Verification

- The skill's three ```go blocks, the README's `Tally` and the concern
  snippet were extracted into a throwaway `internal/zzcomp` package. `go vet`
  passed and the tests passed: Spoiler reveals on tap; both Tallys render
  with no concerns, including with a caller `PaddingLeft`. The package was
  deleted afterwards, so the tracked Go file count is unchanged.
- `go run ./internal/apidoc/gen`, `gofmt -l .` (empty), `go test ./...`
  (all pass).

## Next

Closed: None. Declined: None. Raised: N-087. Deferred: None. Promoted: None.
Updated: None. Full list: `ai_docs/todo/next-list.md`.
