# Shared by android/verify and ios/verify, which source it from their own
# directories (see their gate.sh). It began in android/verify's gate.sh; the
# iOS pass had the same skip-means-exit-0 stance and no switch out of it.
#
# skipped prints a SKIP line and, when GRMOB_VERIFY_STRICT is set, ends the pass
# as a failure instead of letting it continue to a zero exit.
#
# A skip is this directory's stance on purpose — the pass catches what the
# machine can catch — but the stance has one cost: from outside, a run that
# skipped everything and a run that checked everything both exit 0, and the
# difference is a word in the scroll-back. That is fine at a desk and wrong
# anywhere the exit code is the only reader (CI, a pre-push hook, a session
# that needs to KNOW the Kotlin compiled). Strict mode is that reader's switch:
#
#	sh android/verify/run.sh                          skip → exit 0
#	GRMOB_VERIFY_STRICT=1 sh android/verify/run.sh    skip → exit 3
#	GRMOB_VERIFY_STRICT=1 sh ios/verify/run.sh        skip → exit 3
#
# 3 rather than 1 so a caller can tell "could not check" from "checked and
# found a fault". The message goes out first either way, so the remedy the skip
# names is never swallowed by the exit.
skipped() {
  echo "SKIP: $1"
  if [ -n "${GRMOB_VERIFY_STRICT:-}" ]; then
    echo "FAIL: GRMOB_VERIFY_STRICT is set, and a skipped check is not a checked one" >&2
    exit 3
  fi
}
