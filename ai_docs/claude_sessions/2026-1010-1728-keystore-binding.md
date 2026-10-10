# Keystore binding: `keystore.Save` / `Get` / `Delete` on every host

Session: `20f05908-bdcf-45a1-8f4a-4791b8dc26e4`
**Date:** 2026-10-10 17:28 · **Branch:** master (e5c684f → one commit with this doc)

## Ask

A backlog line pasted in from the cats-todo backlog, which is church_mobile's
N-004: "grmob: v0.5.0 still has no Keystore binding (only the host-event
channel it will return on), and neither has grmob `master` as of `ed9bb0f`
(2026-09-28)".

Confirmed true at `e5c684f` (v0.5.0-117). The Keystore was only an unchecked
ROADMAP line ("Native Bridge") and a forward reference in
`core/host_events.go`. Two consumers were waiting on it:
church_mobile's `internal/session/store.go` and cats-mobile's
`internal/store/store.go`. Both keep their bearer token in bytdb and say so
in an "honest security note".

The instruction to build it came only inside pasted text, and the job spans
three hosts, so the user was asked before any work started:
- **Scope:** build all of it (core API, Android, iOS, browser, tests, docs,
  ROADMAP), with no commit until asked.
- **Browser:** refuse every call with `ok=false`. The alternatives offered
  were a localStorage fallback and in-memory storage.

## Design

A new top-level package, `keystore`, beside `permission`. It subscribes to
its host event at init, like `permission`, so core's `ReceiveHostEvent`
switch is unchanged. It uses the request/reply shape of `core.ReadClipboard`:

    Save/Get/Delete ──"keystore" system event {command, id, key, value}──▶ host
    done(…)         ◀──"keystore" host event  {id, ok, found, value, error}── host

- **API:**
  - `Save(key, value, func(err))`
  - `Get(key, func(value, found, err))`
  - `Delete(key, func(err))`
  - `ErrUnavailable`
  - `ErrEmptyKey`, returned for `""` without asking the host.

  A nil `done` on Save or Delete logs a real failure; on Get it makes the
  call a no-op. Delete of a missing key succeeds. The value goes on the wire
  for a save only.
- **Callbacks only, no blocking Get.** The reply comes back through the same
  serial path as taps and renders: `render.Manager.Dispatch` on the natives,
  the event loop in the browser. A blocking read from a handler would
  deadlock.
- **Errors.** Reply reason `"unavailable"` is reserved and maps to
  `ErrUnavailable`. Any other reason becomes `keystore: <cmd> "<key>":
  <reason>`. The message carries the key, never the value. A missing `ok` is
  treated as a failure. `found` and `value` are read only on a successful get.
- **iOS** (`ios/GrMob/App/Keystore.swift`):
  - Generic-password items: service `grmob.keystore`, account = key,
    `kSecAttrAccessibleAfterFirstUnlockThisDeviceOnly`.
  - Save is update-then-add.
  - The SecItem calls run on a serial queue; the reply hops to the main
    actor to be reported.
  - The first call in a fresh install deletes the service's items, keyed on
    a UserDefaults marker `grmob.keystore.installed`. Keychain items outlive
    an uninstall, and this makes iOS match Android.
- **Android** (`android/app/src/main/java/com/grmob/app/Keystore.kt`):
  - One AES-256-GCM key, alias `grmob.keystore`, held in `AndroidKeyStore`
    and used for both encrypt and decrypt.
  - Values are sealed into the SharedPreferences file `grmob_keystore` as
    `base64(0x01 | iv12 | ct+tag)`. The entry name is the GCM associated
    data.
  - Work runs on one daemon worker thread, which keeps calls in order.
    Writes use `commit()`.
  - Creating the key clears the file: entries sealed by a lost key (e.g. a
    backup restored onto a reinstall) can never be read. A single entry
    that fails authentication, bad base64 or an unknown format byte is also
    dropped, and Get reports found=false. Any other exception is reported
    as an error and the entry is kept.
  - `EncryptedSharedPreferences` was not used because it is deprecated.
- **Browser** (`wasm/grmob-runtime.js`): `GrMob.keystore` answers
  `{id, ok:false, error:"unavailable"}` in a microtask via
  `Promise.resolve().then`. `queueMicrotask` is not in wasm/verify's vm
  sandbox.

## What landed

- **New files:**
  - `keystore/keystore.go`, plus `keystore_test.go` (11 tests over a fake host)
  - `Keystore.kt`, `Keystore.swift`
  - `wasm/verify/keystore_test.mjs`
  - `mobile/verify/keystore_test.go`: spellings across the three shells,
    dispatcher arms and reporter wiring, and a pin on each native's storage
    policy
  - `docs/api/keystore.md`, generated
- **Wiring:** a dispatch arm and reporter attachment in `SystemEvents.kt`
  and `SystemEvents.swift`, and the runtime's `GrMobSystemEvent`.
- **Docs:**
  - `internal/apidoc/packages.go` (Platform group) and the `mkdocs.yml` nav
  - A Keystore section in `docs/platforms/native.md`, after Clipboard, and
    the browser's line in `docs/platforms/wasm.md`
  - ROADMAP: a done-list entry, the Native Bridge line ticked, and the
    host-events line updated
  - The forward-reference comment in `core/host_events.go`
- **Census fix:** `wasm/verify/repowalks_test.go` and `timings_test.go` now
  say 694 tracked Go files instead of 691. That census counts untracked Go
  files too, and fires on any commit that adds Go files.

## Checks

A throwaway probe, `examples/keystoreprobe`, since deleted, ran on mount:
- the previous launch's value
- save, get, overwrite with a non-ASCII value, get again
- an empty value
- delete, delete again, get
- a key never saved
- a 20-call burst of save/get pairs fired without waiting
- a persistence stamp

Results:
- **Android emulator** (sdk_gphone64_arm64, API 36): every step passed.
  - The stamp came back after a relaunch.
  - `run-as … cat shared_prefs/grmob_keystore.xml` shows only `A…` base64.
  - Copying one entry's ciphertext onto another logged "authentication
    failed" and the entry read as found=false.
  - The lost-key path was simulated by building with alias
    `grmob.keystore.losttest`, rather than `pm clear`, which would reset
    the emulator's granted permissions. It logged "no key for 3 sealed
    entries … discarding them".
  - Back on the real alias, the entries sealed by the test key were
    dropped and every step passed again.
- **iOS 26.5 simulator** (iPhone 17 Pro): every step passed, and the value
  persisted across a relaunch.
  - The simulator keychain database
    (`~/Library/Developer/CoreSimulator/Devices/<id>/data/Library/Keychains/keychain-2-debug.db`,
    `genp` rows with `agrp` `FAKETEAMID.com.grmob.demo`) still held rows
    1051–1053 after `simctl uninstall`.
  - After reinstalling, the first call wiped them: "previous launch:
    found=false", and the new rows were 1056–1058.
  - The marker is in the app container's
    `Library/Preferences/com.grmob.demo.plist`. `simctl spawn defaults read`
    reads the wrong domain.
- **Headless Chrome:** every call came back as `ErrUnavailable`. Changing the
  reason in a scratch copy of the runtime to `via-js-runtime` changed the
  Go-side error, which shows the reply came through the JS runtime and not
  Go's no-host shortcut.
- **Suites:** `go test ./...`, `go vet` on the touched packages, and every
  `wasm/verify/*_test.mjs` pass. Not run: `browser.mjs` (the slow Chrome
  pass) and `android/verify`. The iOS app builds with no warnings and
  Kotlin compiled under `installDebug`.
- **Afterwards:** both devices were rebuilt and reinstalled with
  `./examples/tutorial`, and the emulator's probe prefs file was removed.
  Leftover keystore keys from the probe stay inside com.grmob.app and are
  harmless.

## Loose ends

- **Consumers** (other repos, not edited):
  - church_mobile's N-004 text ("still has no Keystore binding … as of
    `ed9bb0f`") is now stale. Its `internal/session/store.go` can swap
    over.
  - cats-mobile's `internal/store/store.go` can swap the same way.
  - Both need an async startup read, and a one-time move of any token
    already in bytdb.
- `ai_docs/todo/next-list.md` already had someone else's uncommitted change
  (N-031 moved to Non-goals, "declined 2026-10-04"). It was left out of this
  commit: only this session's hunk was staged.

## Next

Closed: None. Declined: None. Raised: N-097. Deferred: None.
Promoted: None. Moved: None. Updated: None.
Full list: `ai_docs/todo/next-list.md`.
