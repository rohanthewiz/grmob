# Session: a generated reference, and a skill that had five wrong names

**Session ID:** session_01MWC1yLaPNZ7utRWwj2p8rp
**Date:** 2026-09-12, ~09:37
**Branch:** todo/create-mkdoc-style-api-docs-host-fa50

## Goal

Two asks:

1. An mkdocs-style **API reference** — the narrative docs from 2026-08-30
   already exist; what was missing was a per-package statement of the exact
   surface. Hosted with `rohanthewiz/gkdocs`.
2. A **skill** for grmob, in the shape the other libraries here have one.

## What landed

### The generator: `internal/apidoc`

Generated from the packages' own doc comments rather than written, on the
argument in its package comment: `docs/` says things the code cannot and is
written by hand, `docs/api` says only what the code already says, so it is
derived and committed in the shape `aria/gen`'s fixture already established.

```
go run ./internal/apidoc/gen        write docs/api/
go test ./internal/apidoc           fail if docs/api/ is stale
```

Stdlib only — `go/build` for the file list, `go/parser`, `go/doc`, and
`go/doc/comment` for the comments. Nothing new in `go.mod`. Output: 14 pages,
828K, 13 packages, 136 exported types, 508 exported functions and methods.

Files:

- `packages.go` — the hand-kept `Packages` catalogue (dir, nav group, one-line
  blurb), the loader, `RepoRoot`, and the symbol index.
- `render.go` — the markdown. Page shape is godoc's, bent to fit gkdocs' table
  of contents, which is built from **h2 and h3 only**:

  ```
  #      package name              the page title
  ##     Index / Constants / ...   the fixed sections, and package-doc headings
  ###    type Node, func Text      one per top-level symbol — the useful TOC
  ####   func (*Node) Clone        methods, reachable from the page index
  ```

  For a package the size of core that is the difference between a usable
  sidebar and a wall. The per-page Index is plain nested lists under one h2 for
  the same reason — a heading per kind would put the same names in the sidebar
  twice, interleaved with the first copy.
- `slug.go` — goldmark's `parser.ids.Generate`, reproduced.
- `gen/main.go` — the command. Writes before it prunes, so a failure part way
  through leaves too many pages rather than too few; prunes only `.md`, since
  gkdocs serves anything else under `docs_dir` as a static file.

### Three things that were not obvious, each now a test

**Anchors could not be chosen.** gkdocs does not enable goldmark's attribute
extension, so `{#custom-id}` is unavailable and every link into a page has to
*predict* the id goldmark will compute. `slug()` reproduces that algorithm and
deliberately omits one part of it: goldmark's duplicate counter. A renamed id
is not an error to goldmark — it appends `-1` and renders both headings — so
the entire cost falls on the links, which keep pointing at the un-suffixed id
and silently arrive at the wrong section. `TestLinkedAnchorsAreNotStolen`
asserts the property that actually matters instead: every *structural* heading
kept the id its own text produces. Prose headings lifted out of doc comments
are allowed to repeat (two types both document an "Accessibility" section, and
nothing links to either).

**`go/doc/comment` writes its own anchors.** The markdown printer's default is
`## Text {#hdr-Text}`, which is goldmark's attribute syntax — so with gkdocs
the braces would have reached the page as part of the heading's text. Found by
the collision test, which fired on the five core headings whose `{#hdr-…}`
suffixes made them identical. Suppressed with `Printer.HeadingID` returning
`""`.

**Generic receivers broke every cross-reference to a method.**
`doc.Func.Recv` arrives as `*State[T]`; `slug` has no case for `[` or `]` so it
deletes them, giving `statet`, while a doc link arrives as `("State", "Get")`
and can only ever produce `state`. Five links were dead before the fix
(`State.Get`, `State.Set`, `DataTable.Render`, `GroupedList.Render`,
`GroupedList.AutoLoadWithheld`). Type-parameter lists are now dropped from
method headings; the code fence under each carries the full signature,
receiver variable and parameters included.

The other tests: staleness in both directions (missing/stale page, and an
orphaned one the generator no longer writes), every in-directory link resolving
to a real page and a real anchor, `mkdocs.yml` nav mentioning every page, the
`Packages` catalogue covering every importable non-main package or naming its
exclusion in `excluded()`, no build-constrained file in a documented package
(the one assumption `load` makes that could cost a page a declaration with no
visible sign), and every documented package having a package comment.

### Cross-package doc links resolve for real

`[core.Node]` in reconcile's comment becomes `core#type-node` on the served
page. Two pieces make that work: `LookupPackage` is widened past the parser's
default (which only recognises packages the documenting *file* imports — so
`[components.Tabs]` would have been literal brackets anywhere that does not
import components), and a symbol index built over every loaded package before
rendering, because a doc link does not say what *kind* of thing it points at
and `type-node` and `func-node` are different anchors.

Declaration printing filters unexported struct and interface members the way
godoc does. The `// contains filtered or unexported fields` marker reaches the
output through a deliberate abuse of `go/printer`: it is injected as a field
whose *type is an identifier whose name is the comment text*. A real
`*ast.CommentGroup` does not work — `go/printer` only emits comments it can
find in a whole `*ast.File`'s comment map, and what is printed here is a single
synthesised declaration.

### Five packages had no package comment at all

`core`, `hooks`, `render`, `reconcile`, `jsonout` — their pages opened on an
import line and an index, a reference that says what is there and never what it
is for. New `doc.go` in each, written from the source read this session
(`core/node.go`'s immutability contract, `core/context.go`, `NewState`'s lock
note, `render/manager.go`'s mutex and one-slot coalescing buffer,
`reconcile/patch.go`'s ordering rule and the missing move patches).
`TestEveryDocumentedPackageHasAPackageComment` is what stops the next one
shipping blank.

### Hosting: `cmd/docs`, a nested module

gkdocs v0.1.4 brings rweb, element, logger, serr, logrus, goldmark and yaml.
None of that belongs in grmob's `go.mod`, which every consumer of the framework
inherits — so the server is a module of its own, `github.com/rohanthewiz/grmob/cmd/docs`,
and nothing importing grmob ever reads its manifest.

- `main.go` — `gkdocs.New` + rweb. `KeepTrailingSlashes` and `StrictSlashes`
  are set together and commented as a pair: they are the one misconfiguration
  that produces a site which looks correct and links wrong. `findConfig` walks
  up from the working directory, which is what lets it start from the repo root,
  from `cmd/docs`, or from the container's `/app` with no argument.
  `AssetMaxAge` caches the embedded theme only — markdown stays uncached so an
  edit shows on the next reload.
- `Dockerfile` — build context is the **repository root** (the image needs
  `mkdocs.yml` and `docs/`). Distroless static. Nothing is generated at image
  build time, since `docs/api/` is committed.
- `docs.sh` at the root — `cd cmd/docs && go run .`, because `go run ./cmd/docs`
  from the root would have the root module try to build a package it does not
  contain.
- `.github/workflows/ci.yml` — a step to vet and build it, since neither
  `go vet ./...` nor `go test -race ./...` reaches a nested module. `gofmt -l .`
  already covered the files; it walks directories, not modules.

**Not GitHub Pages.** gkdocs renders on request, so there is no static tree to
upload, and Pages is already taken by the tutorial's WASM build.

`mkdocs.yml` gained an `API Reference` nav section, hand-ordered (editorial
rather than alphabetical) with the test holding it to completeness. The
`!!python/name:` risk the 2026-08-30 session flagged turned out to be a
non-issue: gkdocs strips python tags with a regex before handing the YAML to
`yaml.v3`.

### The skill

`ai_docs/SKILL.md`, 588 lines, in the convention the other libraries here use —
canonical copy in the repo, installed copy at
`~/.claude/skills/grmob-native-mobile-go/SKILL.md` carrying the
raw-GitHub pointer plus the full text as fallback, the way the rweb skill does.

Sections: the mental model and the four consequences, a complete app, views and
containers (the interleaved `PropsAndChildren` contract), state and the rules of
hooks with the drift shown as code, state-lives-high, scoping, hooks with the
dependency-equality trap and the latest-closure note, styling with `UseStyle`'s
cannot-clear edge, accessibility, events and the three prop families,
conditionals and keys, navigation with the two `Reset`s, forms, the widget
library, caching's four constraints, debug mode, the test loop, the four
targets and the bridge, and ten pitfalls in the order they bite.

## Verification

- `gofmt -l .` clean, `go vet ./...` clean, `go test ./...` green.
- `cmd/docs`: `go vet ./...` and `go build` clean.
- The site served on a real port: `/`, `/api/`, `/api/core`, `/api/reconcile`,
  `/health` all 200; zero `{#hdr` occurrences in the rendered HTML;
  `id="type-node"`, `id="func-newstate"`, `id="func-state-set"`, `id="index"`,
  `id="constants"` all present and matching what the pages link to;
  `href="core#type-node"` on the reconcile page, and that page 200s.
  Screenshotted — nav, on-this-page TOC and hyperlinked doc links all render.
- **Every code sample in the skill was compiled** against the real source, in a
  scratch module with a `replace` onto this worktree. This is the check that
  earned its keep: it caught five wrong claims written from the concepts docs
  rather than from the declarations.

## What the compile check caught

Worth recording, because all five read as plausible and none would have been
found by review:

| Written | Actual |
|---|---|
| `ctx.Theme().Palette.Primary` | `Theme.Colors`, not `Palette` |
| `core.Justify(core.SpaceBetween)` | `core.JustifyBetween` |
| `SegmentedControl{Options: …}` | the field is `Labels` |
| `Screen{AppBar: …, Children: …}` | `Screen` has no `AppBar` field; it goes in `Children` |
| `core.OnTap(open)` | `core.OnClick` — there is no `OnTap` |

Reading `core/role.go` corrected a sixth thing, which was advice rather than a
name: the skill had `AccessibilityRole(RoleButton)` on a `core.Button`, and
that file says plainly that `Button` exports as a `<button>` and builds a real
control on both natives — `RoleButton` exists for the *other* case, a `Box` or
`Row` with an `OnClick` that every renderer otherwise draws as inert scenery.

## The repo's own checks caught two things

Both are `wasm/verify`'s prose suite working as designed:

1. Eleven new Go files moved the tracked count 450 → 461, and five sentences in
   `repowalks_test.go` and `timings_test.go` price a walk against that number.
   Updated, as the failure message instructs.
2. `slug.go` referenced `TestGeneratedAnchorsAreUnique` after that test had been
   renamed to `TestLinkedAnchorsAreNotStolen`. Pointed at the live name.

## Notes / risks

- **`docs/api/core.md` is 7,700 lines / 478K.** That is one page per package,
  the way godoc does it, and the sidebar TOC makes it navigable — but core is
  large enough that splitting it (by file? by concern?) is a real future call.
  Nothing depends on it being one page except the flat-sibling link scheme,
  which would need a thought if pages moved into subdirectories.
- **`cmd/docs` placement deviates from what was asked.** The option chosen said
  `docs/server/`. `docs/` is the `docs_dir`, and gkdocs serves anything under it
  as a static file, so the server's own `main.go`, `go.mod` and `Dockerfile`
  would have been web-reachable and copied by any `mkdocs build`. Moved to
  `cmd/docs` and `excluded()` names `cmd/` accordingly.
- **`build.ImportDir` resolves against the host GOOS/GOARCH.** No documented
  package has a constrained file today and a test says so, but the failure mode
  if one appeared would be a page quietly missing a declaration on whichever
  machine ran the generator.
- **The installed skill's pointer URL 404s until this branch reaches master.**
  The fallback framing handles it, same as the rweb skill.
- **`aria/spec` is documented under a "Tools" group.** It is a real exported
  package, but it is spec-parsing machinery rather than framework API, and
  whether it belongs in a user-facing reference at all is a judgement that could
  go the other way.

## Next

1. **(new · value med) `docs/api/core.md` is one 7,700-line page.** Navigable
   via the sidebar, and godoc does the same, but core is large enough that the
   question is open. Splitting it would need the flat-sibling link scheme
   reconsidered — every cross-reference is currently a bare relative filename
   with no `../` to get wrong, which is exactly what subdirectories would
   break.
2. **(new · value med) Nothing checks links from `docs/api/` *out* into the
   narrative pages.** `TestEveryGeneratedLinkResolves` skips `../` destinations
   on the grounds that the docs link walk covers them — the 2026-08-30 session
   did that walk with an ad-hoc script, and there is no committed check. The
   overview page's `../concepts/architecture.md` is unverified by anything.
3. **(new · value low) `aria/spec` in a user-facing reference.** Documented
   under "Tools" and arguably should not be there at all. One line in
   `Packages` either way.
4. **(new · value low) The skill's pointer URL is dead until merge.** Expected,
   and the fallback framing handles it, but worth a glance after the first push
   to master that the raw URL actually serves.
5. **(carried · value low) The Android remedy drill has never run.** Syntax-
   checked, takes its own skip correctly on a machine with no SDK, every
   executable line unexecuted — the fault it arranges needs an SDK, a network
   and a cold gradle cache. The first CI run on a branch that touches it is its
   first real test. Worth reverting rather than debugging in place if noisy.
6. **(carried · value low) `hero.png` is a picture of pictures and its parts
   are held twice.** The composite carries no strings of its own, on the
   argument that the three shots under it are already asserted. Right today;
   stops being right the moment the hero is cropped, since a crop can remove
   text the parts still claim.
7. **(carried · value low) A third face is still a skip.** Any machine
   resolving something other than Times or Liberation Serif gets the skip plus
   the calibration table. What nobody has is a face where the readings do NOT
   clear — the report's "WHICH DOES NOT SEPARATE" arm has fired exactly once,
   on Liberation Serif under the old fractions, and is unreachable by any face
   this project has met.
8. **(carried · value low) Windowing as a proposal.** Declined on the payload
   in `non_goals.md`; `TestWhatWindowingWouldSave` asserts its own share and is
   the profile to re-take if a forty-lesson chapter ever ships. Not a Next item
   any more except as this line.
9. **(moved) The four hardware items.** `ai_docs/plans/need_hardware.md`.
10. **(declined, non-goal)** Twenty-seven entries, unchanged. See
    `ai_docs/plans/non_goals.md`.
