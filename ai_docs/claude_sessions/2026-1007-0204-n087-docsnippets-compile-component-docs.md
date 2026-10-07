# N-087: the component docs' code is compiled and run by a test

Session: `ca6ca295-dc54-4e57-95ba-f0c109e7604b`
**Date:** 2026-10-07 02:04 · **Branch:** master (fa3a4ba → one commit with this doc)

This session also did N-083 (`2026-1007-0157-n083-hbox-paddingless-row`). That
work is in its own doc and commit.

## Ask

N-087 from the next-list, pasted in from the cats-todo backlog: the
component examples are compiled by no test. These are `Tally`, `Spoiler`,
the `renderDebug`/`renderPass` harness and the concern snippet, which appear
in the README, `docs/concepts/components.md` and
`ai_docs/SKILL-component.md`. The SKILL file says its widget "compiles as
written". That was last checked by hand on 2026-10-03. The suggested fix was
a test that extracts the ```go blocks under named headings into a temp
module and builds them.

## What was there

- **Two excerpt tests already exist.** `examples/counter/app_test.go` and
  `examples/todoapp/tutorial_excerpt_test.go` *trace* a fence back to the
  source file it quotes. That check cannot apply here: the docs are the only
  copy of Tally and Spoiler.
- **The go-command approach has a precedent.** `cmd/grmob/new_test.go`
  (`TestNewScaffoldsAnAppThatBuildsAndPasses`) builds a scratch module with a
  `replace` back to this checkout and runs `go test`/`go vet` in it. The new
  test does the same.
- **`-short` is one policed lever.** `wasm/verify/shortlever_test.go` allows
  only internal/themehistory's skip, so the new test runs on every
  `go test`. It takes about a second.
- **Fence inventory:**
  - README "## Your own components": a short Tally.
  - components.md:
    - Leaf widget: a Tally.
    - "## A widget that owns state": the Spoiler.
    - "## Accessibility": the concern snippet, a `const` and an `if`, indented
      inside a list item.
    - "## Testing a component": `renderDebug`, `renderPass` and
      `TestSpoilerRevealsOnTap`. It uses `findFirst`/`findText`, which the
      prose describes but does not define.
  - SKILL-component.md: §2 Tally (`package ui` + imports), §6 Spoiler (no
    package clause), §8 the harness with `findFirst`/`findText`.
  - Deliberately left out: the `View` interface quoted from core, the
    `todoRow` fragment (it needs todoapp's `Todo`), and the Callbacks
    fragment (`w.OnTap`).

## What landed

### `internal/docsnippets`

`docsnippets.go` is the package comment only. `docsnippets_test.go` holds
`TestComponentDocSnippetsBuildAndPass`:

    README.md / components.md / SKILL-component.md
        │  go fences under the headings in the `docs` table
        ▼
    t.TempDir()/go.mod   replace grmob => this checkout (go line copied, go.sum copied)
    readme/  concepts/  skill/      one package per doc, one file per fence
        ▼
    go vet ./...   then   go test -count=1 -v ./...   (GOWORK=off, GOFLAGS=-mod=mod)

How it builds and checks the packages:

- **One package per doc.** All three docs declare a Tally, so they cannot
  share a package. Within a doc the fences are read together: the test
  exercises the Spoiler above it, and the concern snippet reads Tally's
  Label.
- **Fence extraction (`goFences`).** It tracks the nearest heading. Non-Go
  fences are also tracked, so a `#` inside one is not taken for a heading.
  An indented fence's body is dedented by the opening fence's indentation.
- **Assembly (`assemble`):**
  - A fence with a package clause is used as written.
  - One without gets `package ui` plus imports for the qualifiers it uses
    (`qualifiers`, via go/scanner, so comments do not count) from
    `knownImports`.
  - A section's `wrap` pastes a fragment into a format. The concern snippet
    goes into `func (w Tally) reportsMisuse() { … }`; a local const is legal.
  - A fence that uses `testing.` becomes `_test.go`.
  - The doc's `given` source (findFirst/findText for the concept page) is
    written as `given_test.go`.
- **`//line <abs doc path>:<line>`** precedes every fence body, so compile,
  vet and test failures name the markdown line.
- **Guards against checking nothing:**
  - Each section states its fence count, so a renamed heading or a moved
    fence fails the test.
  - Every `func Test…(t *testing.T)` the fences declare (read off the fences
    with `testFunc`, not listed by hand) must show `--- PASS` once per doc
    that declares it.
  - If no test is found at all, the run stops with a Fatal.

### Docs

- `ai_docs/SKILL-component.md` §2 and `docs/concepts/components.md`'s "compiles
  as written" sentences now say `internal/docsnippets` builds them on every
  `go test`.

### wasm/verify prose rules this ran into

- **The prose's tracked-Go-file figure was stale.** `TestTheQuestionsOnTheSharedRepositoryParseAreTheOnesDecidedOn`
  holds `wasm/verify/repowalks_test.go` (3 places) and `timings_test.go`
  (2 places) to the `git ls-files` count of Go files. N-083's `hbox_test.go`
  had already made the figure stale (688 → 689), and wasm/verify was not run
  before that commit. It is now 691, including the two new files.
- **Sample test names in prose must resolve.** The same test requires every
  Test-shaped name in Go prose to be a test this repository declares, and
  `TestSpoilerRevealsOnTap` exists only in markdown. Backquotes are the
  documented escape (`quotedprose_test.go`), but they pair left to right
  across a whole comment group. A ```` ```go ```` in the header diagram
  threw the pairing off, and a raw-string map key is unwrapped before
  checking, so it gets no exemption. Fixed by naming no sample test in prose
  at all: the expected tests are read off the fences.

## Checks

Mutation runs, each a one-token sed edit reversed exactly, with `git diff`
empty afterwards:

| mutation | failure |
|---|---|
| `core.ReportConcern` → `ReportConcernX` in components.md | `vet: …/docs/concepts/components.md:310: undefined: core.ReportConcernX` |
| `core.If(revealed.Get(), …)` → `!revealed.Get()` in SKILL-component.md | `SKILL-component.md:436: content starts hidden`, `--- FAIL: TestSpoilerRevealsOnTap` |
| README `## Your own components` renamed | `0 go fence(s) under "## Your own components", the table expects 1` |

Then `go vet ./...` and the full `go test ./...` passed with nothing failing.

## Loose ends

- The other session's uncommitted N-031 move in `ai_docs/todo/next-list.md`
  is still left out of this commit, as it was for N-083.

## Next

Closed: N-087. Declined: None. Raised: None.
Deferred: None. Promoted: None.
Updated: None. Full list: `ai_docs/todo/next-list.md`.
