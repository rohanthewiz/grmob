# A pre-push gofmt guard, and a lexer blind to `if`

**Session:** https://claude.ai/code/session_013PPKnYoYbsVv3vW7oAXA6p
**Date:** 2026-09-13 19:11
**Branch:** master (b67bc63 → 779a74f)

## 1. The asks

1. `/sl`: load the last session doc.
2. "add a pre-push gofmt guard" (the last doc's Next item 27).
3. `/sw`: this doc, commit, push.

## 2. Commits

| Commit | What |
|---|---|
| 779a74f | Pre-push gofmt guard, and the script lexer sees `if git` |

## 3. The hook: `.githooks/pre-push`

- **What it runs.** CI's own `gofmt -l .`, on each pushed ref's tip. It fails
  on any output. It also fails on anything gofmt writes to stderr, meaning a
  file it cannot parse. CI fails on that too, because `run:` blocks use
  `bash -e` and `x=$(gofmt -l .)` inherits gofmt's exit 2.
- **On what.** The tip's tracked Go files, not the working tree:
  `git archive -o <tmp>.tar <sha> -- '*.go'`, untar into a temp dir, and run
  gofmt there. The working tree would be wrong both ways:
  - uncommitted fixes would pass the check and still fail CI;
  - half-edited files not being pushed would block unrelated pushes;
  - `.claude/worktrees/` (whole checkouts, ignored via `.git/info/exclude`)
    and scratch files would be scanned, because gofmt walks directories
    without reading ignore rules.
  About 0.2s for 505 Go files.
- **Tips only**, not every commit in the range. CI checks only the tip too.
- **It blocks.** `.claude/hooks/session-doc-check.sh` only warns, because it
  is often wrong, but this is the same check CI runs. `git push --no-verify`
  skips it.
- **It lets the push through, with a message, when:**
  - gofmt is not on PATH;
  - `git archive` exits 128 (no Go files matched, or not a tree-ish);
  - `mktemp` fails;
  - the ref is being deleted (all-zero sha).
- **Commits named twice** (a branch and a tag on one commit) are checked once.
- **The failure message** lists the files and then the fix: `gofmt -w`,
  `go run ./internal/apidoc/gen` (docs/api cites source lines), commit, push.
- **Enabling it.** `git config core.hooksPath .githooks`. It is set in this
  checkout and documented in README's "Developing with GrMob". A clone does
  not get it automatically.

## 4. The tests

- **Cases fed on stdin:**
  - clean HEAD, and the annotated tag v0.4.0: pass;
  - a `commit-tree` commit adding a misformatted `core/zz_bad.go`: blocked,
    and reported once although two refs named it;
  - a syntax-error file: blocked, showing gofmt's parse error;
  - a delete, a commit on the empty tree, a nonexistent sha, empty stdin:
    pass.
- **Through git:** pushes to a scratch bare repo. The clean HEAD went
  through, the bad commit was refused, and `--no-verify` pushed it. The real
  push of 779a74f to origin also ran through the hook.
- `go test ./...`, `go vet ./wasm/verify/` and `gofmt -l .` were clean.

## 5. What went wrong

- **First version piped `git archive … | tar -xf -`.** The pipeline's status
  is tar's, and tar exits 0 on empty input. A commit with no Go files
  therefore printed nothing and passed, and a failed export would have too.
  Writing the archive to a file with `-o` and checking git's own status fixed
  it. The skip message now prints git's reason.
- **A wrong comment, corrected before commit.** It said a syntax error only
  fails CI at `go vet`. It fails at the gofmt step, through `bash -e`.
- **Python heredoc edits turned `\n` in Go source into real newlines**
  ("string literal not terminated"). Use `r'''…'''` for Go text that contains
  escapes.
- **gofmt flagged my own edit.** The new keywords were longer than the old
  keys, so gofmt re-aligned the whole `commandPrefixes` map. The hook would
  have refused the push.

## 6. The lexer gap: `if git` was not a git call

- `TestEveryGitListingInAScriptAsksForNulSeparatedPaths` found 1 git call
  across 78 scripts both before and after the hook was added.
- **Why:** `repositoryFiles` already lists untracked files
  (`ls-files --others --exclude-standard`), so the file was read. The hook's
  only call is `if ! git archive`. `endCommand` strips only words from
  `commandPrefixes`, which had `then`, `do`, `else` and `!` but not `if`,
  `elif`, `while` or `until`. The call was read as an argument to `if`.
- **The fix.** A new case, "a call as a condition" (four shapes), failed
  first with 0 of 4 found. After the four keywords were added, it passes, and
  the scan reports 2 calls in 78 scripts.
- `git archive` does not list paths, so the -z rule never applied to it. The
  gap mattered for a future `if git ls-files …`.

## 7. Process notes

- **Pipelines hide failures.** A check built on `a | b` answers with `b`'s
  status. When `a` decides pass or fail, write `a` to a file and test its
  status.
- **When a walk-the-repo test's count does not move after a file is added,
  find out why.** Here the file was read but its call was missed.

## Next

**Legend:** *age* is how many session docs ago the item was first raised,
counted from this doc (0 = raised here; `≥ k` = older than the last rebuild's
window). *value*: **high** = worked around today or a second consumer has
arrived; **medium** = blocks one named thing or is a visible defect; **low** =
nobody has hit the gap yet. Sorted by age, oldest first; new items last.

The previous doc's item 27 (a pre-push gofmt guard) is done: 779a74f.

1. **(age ≥12 · value high) Lessons on hardware, and checks still open.**
   - Real devices for everything (deferred by the user).
   - **Screen readers:**
     - radio and StepIndicator done-step semantics
     - combobox active option
     - "pop-up" on Menu/DatePicker triggers
     - aria-current/selected on current items
     - Calendar's grid on the web
     - CodeEditor toolbar role on Compose
     - SearchableSelect natives reading field + list + status
     - AccessibilityHidden confining TalkBack/VoiceOver behind a Drawer
   - **iOS:**
     - sheet `Dialog` and ActionSheet filler
     - Drawer slide by eye and under Reduce Motion
     - RTL for Drawer and CodeEditor (needs a way to force layout direction)
     - editing in the sideways code editor (caret reveal, selection handles)
     - `banded` and a drag on 4.6's list
   - **Pinning and Drawer:**
     - `Screen.Footer` pinning
     - 100% layers in a pinned-height ZStack
     - the panel's `Height 100%` in an HStack
     - `core.Focus` on a Button
     - a hardware keyboard reaching a shut panel
   - **Android:**
     - predictive back from contents
     - MaxWidth where it binds (tablet)
     - a capped stretched child in RTL
2. **(age ≥12 · value low) ", done" and native ", today".** Natives still
   append English ", today". StepIndicator keeps ", done" (documented: no ARIA
   state fits). PageUp/PageDown do not change month, and composite arrow keys
   are not flipped for RTL.
3. **(age ≥12 · non-goal) Tracked-Go-file count sentences** in `wasm/verify` are
   hand-edited when Go files are added. Still 505.
4. **(age ≥12 · non-goal) Windowing.** Declined in `non_goals.md`.
5. **(age ≥12 · non-goal) Rename `docs/components.md` to `comps.md`.**
6. **(age ≥12 · non-goal) Rewrite `components` in the older plans.**
7. **(age ≥12 · non-goal) Trim the copied Android shell's permissions.**
8. **(age ≥12 · non-goal) Replace the iOS usage strings further.**
9. **(age ≥12 · non-goal) C4 `Carousel`** until the host reports a scroll
   offset.
10. **(age 10 · non-goal) A parent that gains `onBack` after its descendants
    outranks them** on Android; the web ranks by document order.
11. **(age 10 · non-goal) An AppBar outside the Navigator** with a custom
    `OnBack` is outranked by the Navigator's pop on Android.
12. **(age 9 · non-goal) Forward does not re-open a screen left by browser
    back.**
13. **(age 8 · non-goal) Sticky headers, placement animation and
    `OnEndReached` in a List with no viewport** on Compose. Give the List a
    Height.
14. **(age 6 · value low) The cost of a paused TimelineView per node** in
    SwiftUI's box chain (`GrMobMotion`). Not profiled.
15. **(age 6 · non-goal) MaxWidth with a growing sibling on the natives.**
16. **(age 6 · non-goal) The typed-hash fold's `history.length` fallback.**
17. **(age 6 · non-goal) A page's own `pushState` while a claim is on screen.**
18. **(age 6 · non-goal) A Drawer's shut panel is composed on the natives.**
19. **(age 4 · value medium) MinWidth/MinHeight on the natives.** Documented as
    web-only, but DatePicker (MinWidth 300), the rich-text link prompt (280)
    and RichTextEditor (MinHeight) rely on them. It has to fit into
    `widthModifier`'s layout lambda on Android; iOS needs the same.
20. **(age 4 · value low) A Row in a horizontal scroll with FlexGrow children**
    has the same zero-weight collapse on Compose. `LocalGrMobUnboundedHeight`
    covers Columns only. Rows inside a bounded LazyColumn with growing children
    too.
21. **(age 4 · value low) Shot-claim field-text check** makes no RTL or
    letter-spacing adjustment (RTL fields use the box check).
22. **(age 4 · non-goal) A List with no Height in a scrolled page is not lazy**
    on iOS or Android. Give the List a Height to virtualise.
23. **(age 3 · value low) The DataTable tree walk covered initial state only.**
    Open overlays and toggled demos (4.6 Compact, 6.x sheets with growing
    content) were not walked. If a layout bug turns up there, turn the walk
    into a kept test over a few scripted states.
24. **(age 2 · value medium) The iOS root `.id(root.viewID)` is unbuilt.** Build
    with the simulator, repeat the 1.3 → bottom → "Next ›" check, and confirm
    an in-lesson tap keeps its scroll. It shipped in v0.4.0 without a device
    run, but the iOS conformance CI job passed.
25. **(age 2 · value low) Next on the web and hot reload after a root
    replace.** Check in a browser that 1.4 opens at its top, and that `serve
    -dev` still puts the reader back at the same scroll offset.
26. **(age 2 · value low) "Scroll position" in PopToRoot's claims.** The
    `core.PopToRoot` doc and lesson 6's "Unwinding" prose say the root frame
    keeps its scroll position. Hook state survives; the native scroll offset
    does not. Reword both, or add a scroll-offset hook the host reports into.
27. **(age 0 · value low) The hook is off in every other checkout.**
    `core.hooksPath` is per clone. Cloud sessions (claude.ai/code) and fresh
    clones push unguarded until someone runs the README line. A SessionStart
    hook in `.claude/settings.json` could set it, but that is a write to git
    config nobody asked for. `hookconfig_test.go` would hold the entry to the
    schema.
28. **(age 0 · value low) The local gofmt is not CI's toolchain.** The hook
    uses whatever `gofmt` is on PATH. CI uses `go-version-file: go.mod`
    (1.26.1). Formatting rarely changes between releases. If it does, the hook
    and CI can disagree. Running `"$(go env GOROOT)"/bin/gofmt` from the repo
    root would use the toolchain go.mod selects.
29. **(age 0 · value low) Other shell shapes the script lexer has no case for.**
    Brace groups (`{ git …; }`) and `case` arms (`pat) git …`) were not
    tested. The condition keywords were missing until today,
    and nothing noticed. Add cases; fix what fails.
30. **(age 0 · non-goal) Checking every commit in a push, not just the tip.**
    CI checks only the tip, so blocking an intermediate commit that a later
    commit fixes prevents no failure.
