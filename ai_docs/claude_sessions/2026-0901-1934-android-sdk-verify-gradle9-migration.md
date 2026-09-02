# Android SDK verification, first emulator run, and the Gradle 9 migration

Session: https://claude.ai/code/session_011WASqH3Z74UCj6VWGcQKAV

## Goal

Started as "verify an Android SDK is available now." It grew, by the user's
own follow-ups, into: wire the SDK into the shell, boot the emulator, install
the demo app on it, and then migrate the Android build off Gradle 8 — which
turned out to be the first time the `android/` module had ever been built and
run in this repo's history.

No Go code was touched in this session. Everything is toolchain, build files
and docs.

## 1. What the SDK survey found

A complete standalone SDK already existed at `~/Library/Android/sdk` (no
Android Studio anywhere). Present and verified working, not merely present:

| Component | Version |
| --- | --- |
| platforms | android-33, 34, 35, 36, 36.1 |
| build-tools | 33.0.1 … 37.0.0 |
| platform-tools | adb 37.0.1 |
| cmdline-tools | `latest` (sdkmanager 20.0) |
| NDK | 28.2.13676358 |
| emulator + image | android-36.1 `google_apis_playstore/arm64-v8a` |
| AVD | `Medium_Phone_API_36.1` |
| JDK | OpenJDK 17.0.17 (Homebrew) |

Two gaps, both environmental rather than missing software:

1. **Nothing pointed at it.** `ANDROID_HOME`, `ANDROID_SDK_ROOT`,
   `ANDROID_NDK_HOME`, `JAVA_HOME` all unset; `sdkmanager`/`avdmanager`/
   `emulator` not on `PATH`. `~/.zshrc:11` had a *commented-out* `ANDROID_HOME`
   pointing at a long-dead path.
2. **`~/bin/adb` (36.0.0, a stray copy) shadowed the SDK's adb 37.0.1**,
   because `shell_config` prepends `$HOME/bin` to `PATH`.

The NDK ships only `darwin-x86_64` host binaries and runs under Rosetta on this
arm64 Mac. That is normal; `aarch64-linux-android21-clang --version` was run to
confirm it actually executes rather than assuming.

## 2. Shell wiring (`~/.zshrc`)

Appended a guarded block (backup at `~/.zshrc.bak-<timestamp>`). Design points,
all commented in the file itself:

- The SDK bin dirs are **prepended**, not appended. `shell_config` is sourced at
  the top of `.zshrc` and puts `$HOME/bin` first, so appending would have left
  the stale adb winning. Prepending fixed precedence without deleting anything.
- `ANDROID_NDK_HOME` is **pinned** to an explicit version rather than globbed
  for "newest": toolchain/ABI behaviour changes between NDK majors, and a glob
  would bump it silently.
- `JAVA_HOME` is resolved via `/usr/libexec/java_home -v 17` so Homebrew patch
  upgrades of `openjdk@17` don't strand a hardcoded Cellar path.
- The whole block is `if [ -d … ]`-guarded, matching the existing
  `shell_config` style, so the rc stays portable to machines without the SDK.

`~/bin/adb` was later removed at the user's request (a lone Mach-O binary, not
a symlink, with no other platform-tools strays beside it). The `PATH` prepend
is now belt-and-suspenders rather than load-bearing, and was kept for that.

## 3. A correction worth recording: `gomobile version` lies

`gomobile version` prints `binary is out of date, re-install it` **even at the
exactly-pinned version**. The initial read of this as "stale, needs rebuild"
was wrong.

Root cause, chased down rather than guessed: `runVersion` shells out to
`go list -f '{{.Stale}}' golang.org/x/mobile/cmd/gomobile`. A binary installed
via `go install pkg@version` resolves *that module's own* dependency versions,
while `go list` computes the build ID in the **project's** module context —
which pins `x/mod v0.29.0`, `x/tools v0.38.0` via the `tool` block in `go.mod`.
Two contexts, two build IDs, a permanent false "stale":

```
$ GOBIN=~/go/bin go list -f '{{.Stale}} {{.StaleReason}}' golang.org/x/mobile/cmd/gomobile
true build ID mismatch
```

Expect this message forever. **The bind is the real signal.** `gomobile` and
`gobind` were reinstalled at the version `go.mod`'s `tool` block pins
(`v0.0.0-20251021151156-188f512ec823`), not `@latest`, because the repo
deliberately pins that toolchain.

## 4. Emulator + first install

`Medium_Phone_API_36.1` booted in **18s**. Waiting on `adb devices` alone would
have reported ready too early — the transport shows up `offline` well before
the framework is up — so the wait polls `sys.boot_completed`.

```
emulator-5554  device  product:sdk_gphone64_arm64  device:emu64a
Android 16 / API 36 / arm64-v8a
```

The AVD's `arm64-v8a` matches the AAR's `jni/arm64-v8a/libgojni.so`, so the
native lib loads with no translation layer.

**Gradle did not exist on the machine and there was no wrapper in the repo.**
Homebrew was deliberately *not* used: it ships 9.7.1, and AGP 8.1.0 needs
Gradle 8.x. Instead Gradle 8.7 was downloaded to a scratchpad (SHA-256 verified
against `services.gradle.org`) purely to get a first build.

Result: `BUILD SUCCESSFUL in 35s`, APK installed, and the demo verified
*interactively* — tapping Increment took **Count 0 → 3**, proving the Go core
drives Compose *and* receives events back, not just a static first frame.

## 5. The Gradle 9 migration — empirical, not from memory

Every step below was an observed failure in a throwaway copy of `android/`,
not a guess:

| # | Config | Result |
| --- | --- | --- |
| 1 | AGP 8.1.0 + Gradle 9.7.1 | `DependencyHandler.module(Object)` — client-module API removed |
| 2 | AGP **8.13.2** (latest 8.x) + Gradle 9.7.1 | AGP 8.x uses `InternalProblems`, **removed in Gradle 9.6.0** |
| 3 | AGP 9.4.0 + `kotlin-android` | AGP 9 has built-in Kotlin; the plugin conflicts |
| 4 | AGP 9.4.0 + Kotlin 2.4.10, no `kotlin-android` | **BUILD SUCCESSFUL** |

**Step 2 is the load-bearing finding.** No AGP 8.x can reach current Gradle, so
bumping within the 8 line buys only Gradle 9.0–9.5 and a repeat of the exercise
later. The major bump is the only way forward.

Not verified, and deliberately not claimed: AGP 8.13.2 + Gradle **9.5**. That
is Gradle's own suggested fallback in the error text, but it was never run.

## 6. What actually changed in the repo

`android/build.gradle`
- AGP `8.1.0 → 9.4.0`, Kotlin `1.9.24 → 2.4.10`
- added `compose-compiler-gradle-plugin:2.4.10`
- comments record *why* the AGP major was unavoidable, so nobody retries 8.x

`android/app/build.gradle`
- dropped `id 'kotlin-android'` (AGP 9 supplies Kotlin itself)
- added `id 'org.jetbrains.kotlin.plugin.compose'`
- `kotlinOptions {}` → typed `kotlin { compilerOptions { … } }`
- deleted `composeOptions`/`kotlinCompilerExtensionVersion` — Kotlin 2.x moved
  the Compose compiler into a plugin locked to the Kotlin version, so the old
  "keep 1.5.14 ↔ 1.9.24 in sync" comment describes a coupling that no longer
  exists
- converted Groovy space-assignments (`namespace =`, `compose = true`, …) that
  Gradle 10 removes

**Committed Gradle wrapper** — `gradlew`, `gradlew.bat`, `gradle/wrapper/`,
pinned to 9.7.1 **with `distributionSha256Sum`** so the distribution is
checksum-verified on download. This retires the scratchpad-Gradle problem: a
clean clone now builds with no system Gradle.

Docs
- `docs/tutorial-todo.md` said `cd android && gradle assembleDebug`, which would
  fail on any fresh machine → `./gradlew`
- `docs/platforms/native.md` documented only Android Studio; added a CLI section

`CHANGELOG.md` — entry under `[Unreleased]`.

Notably **not** required: no Compose BOM bump (2024.06.00 worked as-is), no
`compileSdk`/`minSdk` change, and no source edits — the seven Kotlin runtime
files compiled clean under Kotlin 2.4.

## 7. Verification

Full pipeline from a clean state (`build/`, `app/build/`, `.gradle/`,
`app/libs/` all deleted):

```
android/build.sh          → grmob.aar (7.7 MB)
./gradlew assembleDebug   → BUILD SUCCESSFUL in 13s
```

- **Zero** deprecation warnings under `--warning-mode all` (was 1 error + 2)
- APK installed, launched, tapped: **Count 0 → 2**, `FATAL` count **0**
- APK **19.3 MB → 16.1 MB** — AGP 9 strips `libgojni.so` (arm64 3.47 → 2.39 MB),
  which AGP 8.1 explicitly reported it could not do

## 8. Left alone, on purpose

- `targetSdk 34`. Play wants 35+, but that is a release-policy decision, not a
  build fix.
- Gradle's configuration-cache suggestion.
- The iOS side — untouched this session.
