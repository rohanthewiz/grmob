# Dropping the CHANGELOG, tagging v0.1.0, and the first CI workflow

Session: https://claude.ai/code/session_011WASqH3Z74UCj6VWGcQKAV

Loaded `2026-0901-2001-phase6.1-wrong-docs.md` as context. Three commits:

| | |
|---|---|
| `d1f7269` | Drop CHANGELOG.md; commits and session docs are the history of record |
| `v0.1.0` | Annotated tag on `d1f7269`, pushed |
| (this) | Add CI workflow covering Go, WASM, iOS and Android |

## 1. The CHANGELOG is gone

The user's call: *"I don't want to do a CHANGELOG. The commits and saved
claude sessions should suffice."* That reads two ways — delete the file, or
just stop doing 6.3's additions — and the global instruction is not to remove
existing content unless sure, so it was put back as a question. Answer:
delete entirely.

The file was safe to remove. Nothing outside `ai_docs/` referenced it: no
README link, no docs page, no `mkdocs.yml` nav entry. Verified by grep before
touching anything, not assumed.

Five places in `ai_docs/plans/fable-pre-0.1.0-analysis.md` carried
obligations it had created, all retired:

- **6.3's omissions audit** became the decision that retired it, keeping a
  one-line record of what the audit had demanded (accessibility props,
  `core.MaybeProp`, `CameraView`/`TabView`, …) so the reasoning survives the
  file.
- **6.3's dated prose** item — `styling-and-theming.md`'s "Before 2026-08-31"
  changelog prose — had "Move to CHANGELOG" as its fix. With no destination
  the remedy changed to "rewrite as timeless prose". **Still open**, just with
  a different fix.
- **6.1 row 17** struck through as moot.
- **6.2** no longer lists the CHANGELOG among docs claiming toast APIs.
- **Phase 7 and the suggested order** lost the tag's changelog dependency.

Edits applied with a `python3` heredoc asserting `count(old) == 1` per
replacement, the method this repo has used since Phase 1.

### The one thing that lost its home

The `SubscribeRender`/`RegisterRender` removal from Phase 1.12 is an
**exported-API removal** whose only record is now commit `c441809` and its
session doc. Worth hand-carrying into a GitHub release body if one is written.

### A process note

The first commit caught only the deletion — the plan edits were never staged,
and `git commit` without `-a` silently committed one file of two. Caught by
reading `--stat` after the fact and fixed with `--amend`. Read the stat line.

## 2. v0.1.0 tagged

Gated on a full baseline, run rather than assumed:

| Check | Result |
|---|---|
| `gofmt -l .` | clean |
| `go vet ./...` | clean |
| `go test ./...` | all ok |
| `go test -race ./...` | all ok |
| `GOOS=js GOARCH=wasm go build -o … ./wasm` | ok, 5.4 MB |
| `wasm/verify/run.sh` | ok |
| `ios/verify/run.sh` | ok, 3 checks |
| `:app:compileDebugKotlin --rerun-tasks` | ok, no warnings |

**The Kotlin now provably compiles.** Carried-forward item 3 from the last
session said `GrMobLongPressButton` (Phase 1.10) had never been compiled. A
plain `assembleDebug` reported everything `UP-TO-DATE`, which proves nothing,
so the Kotlin task was forced with `--rerun-tasks`. Clean, no warnings.
`GrMobLongPressButton` lives in `android/app/src/main/java/com/grmob/runtime/
Renderer.kt`, inside the `:app` main source set, so it is genuinely covered.

The SDK needed pointing at from a non-interactive shell — `ANDROID_HOME` and
`JAVA_HOME` are wired into `~/.zshrc` by the 1934 session but that rc is not
sourced here. Passed inline per command; **no `local.properties` was
written**, since it is machine-specific.

The tag is annotated, and its body is the release notes — largely the deleted
CHANGELOG's 0.1.0 section, which was accurate about what exists (its problems
were omissions and a dead link, not falsehood). It ends with a **known limits**
section rather than implying none: no device run of either native long-press
path, Compose still missing `Gap`/`Justify`/Modal backdrop, no CI at the time
of tagging.

No `Co-Authored-By`/session footer on the tag: that convention covers commit
messages and PR descriptions, and a tag annotation is public release notes.

Pushed. `9b95cd2` (tag object) → `d1f7269` (commit). The Go module proxy
resolves `github.com/rohanthewiz/grmob@v0.1.0` already.

**Locked into the tag:** `go.mod` requires `bytdb` (+ `btypedb`, `serr`,
`btype`) though only `examples/todoapp` imports it, so every consumer of
v0.1.0 inherits them. Phase 7's dependency-footprint item; a nested module for
the examples would drop it from the public surface in 0.2.0.

## 3. CI

`.github/workflows/ci.yml`, three parallel jobs so one red renderer cannot
hide another:

| Job | Runner | Checks |
|---|---|---|
| `go` | ubuntu | gofmt, vet, `test -race`, WASM build, `wasm/verify` |
| `ios` | macos-latest | `ios/verify/run.sh` |
| `android` | ubuntu | `gomobile bind` → `assembleDebug` |

Only `-race` runs, not a plain `go test ./...` before it: same tests, plus
instrumentation, and the engine dispatches callbacks and timer pushes from
separate goroutines.

### Two things the plan's checklist had wrong

Both found by running the steps, not by reading them.

1. **`GOOS=js GOARCH=wasm go build ./wasm` does not work.** It fails with
   `build output "wasm" already exists and is a directory` — the package path
   is also a directory name. `-o` is mandatory. This check has been in the
   plan's verification list all along and cannot ever have run as written.
2. **The Android job cannot just call Gradle.** `app/libs/grmob.aar` is not
   tracked (the repo keeps no binaries), so the AAR must be bound first — and
   `go tool gomobile bind` fails with `gobind was not found. Please run
   gomobile init`. gomobile shells out to a **gobind executable**, which a
   `tool` block does not put on `PATH`. Building the pinned gobind into a temp
   dir and prepending it works: verified locally, 7.3 MB AAR, exit 0.

That second finding also answers **6.3's open "the gomobile pin is inert for
every documented workflow" item**. The CI job is now the only thing in the
repo that honors the pin. `android/build.sh`, `README.md:260` and
`docs/platforms/native.md:62` still say `go install …gomobile@latest`; left
alone as out of scope, but the fix is now known:

```sh
go build -o "$SOME_BIN_DIR/gobind" golang.org/x/mobile/cmd/gobind
PATH="$SOME_BIN_DIR:$PATH" go tool gomobile bind …
```

`android/build.sh` is deliberately not reused by CI: it resolves gomobile from
`GOPATH/bin` and defaults `ANDROID_HOME` to a macOS path.

### Android went in beyond the plan's six checks

It is the only guard on `Renderer.kt`, `TreeStore.kt` and `GrMobStyle.kt`,
which have no unit tests — the same gap that let the Phase 1.10 Kotlin sit
uncompiled. It is also the slowest and most fragile job (NDK, three ABIs). If
it proves flaky, moving it to `workflow_dispatch` is the fallback.

### Validated vs. not

Validated locally: the YAML parses (three jobs, expected step counts), the
gomobile bind through the pinned toolchain, `./gradlew assembleDebug
--no-daemon` clean with no deprecation warnings, and the
`cmdline-tools/latest/bin/sdkmanager` path shape.

**Not verifiable off-runner:** `ANDROID_NDK_LATEST_HOME`, whether the hosted
image's `cmdline-tools` path matches, and the macOS image's Swift setup. The
first run confirms them.

## Carried forward

1. CI has never run. The first push to `master` is its own smoke test; expect
   the Android job to need a nudge.
2. `android/build.sh` + two docs still document `go install …gomobile@latest`
   against the `tool` pin (6.3). The fix is above and is a few lines.
3. Neither native long-press path has run on a device — compile-verified only.
4. `wasm/main.go`'s `renderInitial` still re-mounts over the same global `ctx`
   (plan item 2.2); 1.6 fixed the hooks half, the stale-slot half is open.
5. 6.3's dated-prose item now wants a rewrite, not a move.
6. **6.2 untouched** and still the larger documentation job: absent package
   doc comments, 182 of `core`'s 279 exports undocumented.
7. v0.1.0's dependency surface includes `bytdb` and friends; revisit for
   0.2.0.

## Where the plan stands

- **Phase 1 — done.** **Phase 6.1 — done.** **Phase 7 CI — done**; the tag is
  cut and pushed.
- Phase 2 (stability), 3 (renderer parity), 4 (performance), 5 (quick wins),
  6.2 / 6.3 / 6.4 (documentation), and Phase 7's remaining hygiene items
  (dependency footprint, dead-code deletion, CONTRIBUTING/SECURITY) — open.

Next per the plan: Phase 3 renderer parity, DOM/htmlout first.
