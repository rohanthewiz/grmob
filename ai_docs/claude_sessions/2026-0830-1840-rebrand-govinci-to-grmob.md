# Session: Rebrand govinci → grmob — new repo, module path, brand identifiers, licensing

Session ID: `bbf191e5-9fbd-4a57-a799-ceb8a2faa81a`
(Claude session link: https://claude.ai/code/session_01VE2bWpQ3DjYkNt5p4287se)
Date: 2026-08-30 (~18:07–18:40)
Branch: `master` — first session in the new `grmob` repo; continues from
`2026-0830-0920-todo-persistence-bytdb.md`, which was the last commit made
under the old name.

## What happened, in order

Asked: rebrand `~/projs/go/govinci` to `grmob` in a new working copy at
`~/projs/go/grmob`, pushing to the empty repo
`https://github.com/rohanthewiz/grmob.git`.

1. **Scope survey first.** 111 tracked files, 75 containing the brand, 18
   carrying it in the path. Critically, `find` also turned up ~300MB of
   *untracked* build output (`ios/build/` 248M, `ios/Frameworks/` 43M,
   `android/app/libs/` 16M) — the repo had no top-level `.gitignore`, so
   those were untracked only by luck. Every subsequent step worked from
   `git ls-files` / `git grep`, never the filesystem, so build artifacts
   were never touched.

2. **Two decisions put to the user** rather than guessed:
   - Casing → **`GrMob`** (over `Grmob` / `GRMob`) for type names and prose.
   - History → **preserve**. Done with `git clone ./govinci ./grmob`, which
     brings full history and leaves untracked artifacts behind, then
     `git remote set-url origin` to the new URL. Verified the source tree
     was clean and fully pushed first (no unpushed commits, no stashes).

3. **Renames via `git mv`** so history follows: `ios/Govinci` → `ios/GrMob`,
   `ios/GovinciUITests` → `ios/GrMobUITests`,
   `android/.../com/govinci` → `com/grmob`, plus 18 individual files.

4. **Content rewrite, four ordered perl rules** — the ordering is the whole
   trick, since a single case-insensitive replace would corrupt identifiers:
   ```
   s{github\.com/GraHms/govinci}{github.com/rohanthewiz/grmob}g;  # module path first
   s{Govinci}{GrMob}g;                                            # type/brand names
   s{govinci(?=[A-Z])}{grMob}g;                                   # camelCase prefix: govinciBox -> grMobBox
   s{govinci}{grmob}g;                                            # bare token: com.grmob, grmob.aar, build tag
   ```

5. **Demo data vs. attribution, separated deliberately.** `GraHms` was both
   the old module owner *and* the original author's handle. Rewrote only the
   module path on the first pass, then in a second commit neutralized the
   placeholder *demo* strings (`grahms_dev` → `gopher_dev`; profile names →
   `Fulano de Tal` / `Jane Doe` / `Your Name`) while leaving genuine
   attribution intact.

6. **Top-level `.gitignore`**, deliberately narrow: `android/.gitignore` and
   `ios/.gitignore` already cover their platforms, so this one adds only Go
   build/test output, `wasm/main.wasm`, the three static pages the htmlout
   examples write, `.cats-todo/`, and editor/OS cruft.

7. **Licensing.** User removed `CODE_OF_CONDUCT.md` (it routed incident
   reports to a stale placeholder address). Added `Copyright (c) 2026 Rohan
   Allison` **above** the retained 2025 notice in `LICENSE` and README —
   added, never replaced. Then unified the upstream author's name to the
   legal form `Ismael Matsinhe` in both notices (README had been using the
   GitHub handle `GraHms` for the same person).

## Commits this session (oldest first)

```
8af25e8  Rebrand govinci to grmob
33d932c  Neutralize demo-data names; add top-level .gitignore
1966395  Share copyright with Rohan Allison; drop CODE_OF_CONDUCT
18c6280  Use the legal name consistently in the copyright notice
```
77 files changed, 578 insertions(+), 586 deletions(-) vs. `75870be`.
All pushed; `master` tracks `origin/master` at the new repo.

## Name mapping (for reference)

| Layer | Before | After |
|---|---|---|
| Module | `github.com/GraHms/govinci` | `github.com/rohanthewiz/grmob` |
| Build tag | `//go:build govinci` | `//go:build grmob` |
| Swift/Kotlin types | `GovinciRuntime`, `GovinciNode` | `GrMobRuntime`, `GrMobNode` |
| Swift modifiers | `.govinciBox(...)` | `.grMobBox(...)` |
| Android pkg / appId | `com.govinci.app` | `com.grmob.app` |
| Artifacts | `govinci.aar`, `Govinci.xcframework` | `grmob.aar`, `GrMob.xcframework` |
| WASM | `govinci-runtime.js`, `window.GovinciWASM` | `grmob-runtime.js`, `window.GrMobWASM` |
| Xcode | `ios/Govinci/`, `GovinciApp` | `ios/GrMob/`, `GrMobApp` |
| Gradle root | `GovinciAndroid` | `GrMobAndroid` |

## Verification performed

- `go build ./...` and `go test ./...` — 8 test packages pass.
- `GOOS=js GOARCH=wasm go build ./wasm` — ok.
- `ios/verify/run.sh` — compiles the renamed Swift runtime files and replays
  a Go-generated transcript through them:
  *"OK: 9 patch batches applied; tree matches Go's final render."* This is
  the fast loop that proves the Swift rename didn't break the data layer,
  with no simulator needed.
- `git ls-files -i -c --exclude-standard` — empty, confirming the new
  `.gitignore` shadows no tracked file (notably `wasm/wasm_exec.js`, which
  is copied from the Go toolchain and must stay tracked).

Not re-verified: a real simulator/device run on either platform, and the
gomobile binds themselves (`ios/build.sh`, `android/build.sh`) — see next
steps.

## Key knowledge worth carrying forward

- **The shell here is zsh, which does not word-split unquoted variables.**
  `for f in $files; do perl -pi -e '...' "$f"; done` silently passed the
  entire newline-joined list to perl as one argument (`Can't open
  README.md\nROADMAP.md\n...`) and edited nothing. Use
  `git grep -lIz ... | xargs -0` instead. The failure is quiet — it looked
  like a perl error on one file, not a total no-op.
- **perl and non-ASCII: match `©` as raw bytes `\xc2\xa9`.** In default byte
  mode `\x{00A9}` means the single byte `\xA9` and will not match UTF-8
  input; and with `-CSD` the input decodes but the `-e` script itself does
  not (no `use utf8`), so a literal `©` in the pattern stays two bytes and
  also fails. Byte-level `\xc2\xa9` on both sides is the reliable form.
- **`git grep -lI`** (capital I = skip binary) is the right file selector for
  a mass rewrite: it is limited to tracked files and won't hand a `.aar` or
  `.xcframework` binary to a text tool.
- **Case-sensitive replaces leak.** The exact-case module rule missed four
  lowercase `github.com/grahms/govinci` import paths in README, which rule 4
  then turned into the wrong-owner `github.com/grahms/grmob`. Always grep for
  near-miss variants after a mechanical pass, not just the original token.
- **MIT forks: add, never replace.** The original copyright notice must be
  retained; a new holder's line goes alongside it. Author *bylines* and demo
  *placeholder data* are different categories from the copyright notice and
  warrant separate decisions — which is why they landed in different commits
  here.
- `ios/GrMob/Info.plist` is gitignored and generated; the Xcode project is
  generated from the tracked `ios/project.yml` via `xcodegen generate`. Both
  were already so before the rebrand — not a regression.

## Possible next steps

- Regenerate the native artifacts under the new names and do a simulator
  pass on both platforms: `ios/build.sh` → `GrMob.xcframework`,
  `android/build.sh` → `grmob.aar`, and `cd ios && xcodegen generate` for
  `GrMobApp.xcodeproj`. Nothing in this session exercised gomobile itself.
- `docs/ui-architecture.md:3` still carries the upstream byline
  **Ismael GraHms** — the last remaining `GraHms` in the repo. Left as the
  original author wrote it; open question whether to switch it to the legal
  name for consistency, or add a co-author line if that doc gets reworked.
- Old repo `rohanthewiz/govinci` is untouched and still has its own history;
  decide whether to archive it or add a pointer to `grmob` in its README.
- Consider a `go.mod` `retract` or a note for anyone who had imported the old
  `github.com/GraHms/govinci` path.
