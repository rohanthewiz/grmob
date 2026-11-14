# N-139, N-143, N-141: watcher graph cache, highlight cache, form memo, release workflow, `go tool grmob`

Session: `94ba3d17-9a13-41bc-96af-d1aadbbafc95`

This is the second batch of an unattended `/work-next-list` run. The first
batch is `2026-1114-1650-n151-n138-n144-group-echo-pins-one-pass-per-event-generated-style-equal`.
All three items here moved forward but stay Open.

## 1. N-139: host and library hot spots

- **P7: the dev watcher (`543c829`).**
  - `serve/dev.go` caches the package directories from `go list -deps` in
    `pkgDirs`, keyed by the go.mod and go.sum stamps (`graphKey`). Each poll
    only lists and stats those directories' files.
  - `watch` sets `pkgDirs = nil` after a build, so a new import is picked
    up.
  - A `listGraph` hook lets the test count reads:
    `TestTheWatcherReadsTheBuildGraphOnlyWhenItMayHaveChanged`.
- **P8a and P8d: highlighting (`ac5b1f4`).**
  - `highlight/cache.go` is a 16-entry LRU keyed by `{lexer any, scheme,
    src}`. The lexer is `"go"`, `"json"`, or the table lexer's `*langSpec`.
    A caller's own Highlighter is not cached.
  - A hit returns the same slice. reconcile's `reflect.DeepEqual` then
    compares the props by identity, so the diff of an unchanged editor is
    cheap too.
  - `BenchmarkRows500Lines`: cold 365 µs and 1.3 MB, warm 0.37 µs and 0
    allocs.
- **P8c: form error reads (`d0e3d77`).**
  - `Form.Error` and `Errors` share a memo keyed by
    `ctx.DirtyGeneration()`, the generation P1 introduced. Every form-record
    write and every `core.State` write bumps it.
  - `Error` reads its own field rather than building `Errors()`'s map.
  - `Valid` and `Submit` stay fresh on purpose, because a rule may read a
    plain variable the handler just set. The test covers that case.
- **P8b was not done.** DataTable's doc promises "holds no state and calls
  no hook". Caching the sort needs a hook slot or a Rows-identity key, and
  the second goes stale when a caller reorders Rows in place. Written into
  the item as a decision.
- **Remaining:** P6 (web runtime hot spots) and P9 (iOS).

## 2. N-143: housekeeping (`0967d2e`)

- **D3:** a new `.github/workflows/release.yml`, run on a `v*` tag.
  - It runs apidiff, pinned to `x/exp@v0.0.0-20261014143641-f0251eb4aa3e`,
    against `git describe --tags --abbrev=0 "$tag^"`, using a worktree of
    the previous tag. The report goes to the job summary.
  - A patch release with incompatible changes fails the job. A minor
    release is reported, not failed, since breaking changes are allowed
    below 1.0.
  - Then `gh release create --verify-tag --notes-from-tag`. It skips a tag
    that already has a release.
  - The apidiff commands were run locally from v0.8.1 to HEAD: compatible
    additions only. The workflow is unrun until the next tag.
  - No releases were backfilled for past tags.
- **D5:**
  - Removed `Context.idGen`, `children` and `childrenCursor`. Nothing wrote
    them, so the loops in `Reset` and `auditCursor` ran over nil.
  - Removed the runtime's `const DEBUG`.
  - `AppConfig.Locale` is exported, so it stays, documented as unread.
- **D6:**
  - README and `wasm/index.html` now say 87 lessons.
  - `docs/index.md` names the twenty hooks.
  - `build.sh`'s `-s -w` saving was measured at 3.6% (13.78 MB to 13.28 MB).
- **D7:** ROADMAP marks deep links and the CLI as done.
- **Left:** README's "Upgrading to v0.3.0?" callout. It still helps someone
  upgrading from before 0.3.
- The item now waits only on the user's `ai_docs/` decision (D2) and is
  re-rated `(user's decision)`.

## 3. N-141: CLI (`9d1d7ae`)

- **C3:** the scaffold runs `go mod edit -tool=…/cmd/grmob`. The Done
  message, the README template, README, getting-started and SKILL.md all
  say `go tool grmob <cmd>`, with a `go get -tool` line for apps made
  before this change.
- **`grmob version`** (`cmd/grmob/version.go`):
  - Under `go tool`, `info.Main` is the grmob module itself, not the app.
  - With a replace directive, `Main.Version` is the zero placeholder, so the
    replace directory is printed instead.
  - A checkout build prints its VCS revision.
- **Checked:** the scaffold test runs `go tool grmob version` in the new app.
  A real scaffold in the scratchpad ran `go tool grmob doctor`.
- **Remaining:** `grmob dev`, the extra doctor checks, C4 (`-fast` and
  skipping an unchanged bind) and C5 (the shell sync).

## 4. Collateral

Each new tracked Go file moves `wasm/verify`'s quoted file count, which is
now 777. `docs/api` was regenerated after each exported doc change.

## Next

Closed: None. Declined: None. Raised: None. Deferred: None. Promoted: None.
Moved: None. Updated: N-139, N-141, N-143. Full list:
`ai_docs/todo/next-list.md`.
