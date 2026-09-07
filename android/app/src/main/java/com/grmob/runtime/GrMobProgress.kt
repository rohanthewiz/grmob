package com.grmob.runtime

/**
 * What a core.ValueRange's three numbers amount to, as a decision separate
 * from the Compose property it lands in.
 *
 * # Why this is its own file
 *
 * It was a three-way branch inside `grMobValue`, which is an extension on
 * `SemanticsPropertyReceiver` — so exercising it meant constructing a Compose
 * semantics scope, which means an instrumented test on a device. Nothing
 * outside a device could run it, and what checked it instead was
 * mobile/verify reading GrMobStyle.kt as text and finding
 * `ProgressBarRangeInfo.Indeterminate` somewhere in the file. That is a check
 * that the *word* is present, and it stayed green for a branch that reached
 * the word on the wrong condition.
 *
 * This file imports nothing at all, exactly as GrMobSelectMenu.kt does and
 * for the same reason: a plain JVM can run it, so `android/verify` compares
 * the decision against Go's — core.ValueRange.Progress, over the shared table
 * in internal/valuefixture. What is left in GrMobStyle.kt is the assignment,
 * which is the part that genuinely needs Compose.
 *
 * # The readings are Go's strings
 *
 * Spelled as the same four literals core.ProgressReading uses, so the harness
 * compares values rather than a mapping table that would itself need
 * checking. An enum would be the more Kotlin-ish choice and would put a second
 * vocabulary between the two sides.
 */
data class GrMobProgress(
    val reading: String,
    /**
     * The position and the range. Meaningful when [reading] is
     * [GRMOB_PROGRESS_DETERMINATE]; for an empty range they are the numbers as
     * stated so a caller can report them, and otherwise they are 0f, which is
     * not a position.
     */
    val now: Float,
    val min: Float,
    val max: Float,
)

/** Nothing numeric was stated: the range property is left alone. */
const val GRMOB_PROGRESS_UNSTATED = "unstated"

/** Bounds and no position — a bar running with no idea how far. */
const val GRMOB_PROGRESS_INDETERMINATE = "indeterminate"

/** A position inside a real range. */
const val GRMOB_PROGRESS_DETERMINATE = "determinate"

/**
 * A position inside a range that is not one. Compose's ProgressBarRangeInfo
 * requires a non-empty range and throws otherwise, so this reading exists to
 * be caught before the property is assigned: crashing a render over an
 * accessibility annotation is the wrong trade.
 */
const val GRMOB_PROGRESS_EMPTY_RANGE = "empty-range"

/**
 * The wire-string-to-number rule, in one place.
 *
 * Null for an empty string, for anything that does not parse, and for the
 * non-finite spellings Kotlin's own parser accepts — `"NaN".toFloatOrNull()`
 * returns NaN rather than null, and NaN is not a position any range property
 * on any platform can hold. Go's core.ValueRange.Progress refuses the same
 * three, which is what makes the two comparable.
 *
 * Null rather than a 0f default is the whole reason ValueRange's numbers cross
 * as strings: 0 is a bar at the start of an upload.
 */
fun grMobProgressNumber(s: String): Float? = s.toFloatOrNull()?.takeIf { it.isFinite() }

/**
 * Resolves the three parsed numbers into the one claim they make.
 *
 * Mirrors core.ValueRange.Progress step for step. The order matters and is the
 * order the rules are stated in on the Go type:
 *
 * 1. no position at all is either ARIA's indeterminate bar (if a bound was
 *    stated) or no numeric claim;
 * 2. an unstated bound takes ARIA's own default, 0 and 100, which is what
 *    makes a bare position announce as a percentage;
 * 3. a range that is not one is reported rather than assigned;
 * 4. the position is clamped into the range — a live counter that overshot
 *    must not announce a number the range does not contain.
 *
 * The text plays no part: it is a separate claim on a separate property, and
 * it is honoured on nodes that carry no range at all.
 */
fun grMobProgressOf(now: Float?, min: Float?, max: Float?): GrMobProgress {
    if (now == null) {
        return if (min != null || max != null) {
            GrMobProgress(GRMOB_PROGRESS_INDETERMINATE, 0f, 0f, 0f)
        } else {
            GrMobProgress(GRMOB_PROGRESS_UNSTATED, 0f, 0f, 0f)
        }
    }
    val lo = min ?: 0f
    val hi = max ?: 100f
    if (hi <= lo) return GrMobProgress(GRMOB_PROGRESS_EMPTY_RANGE, now, lo, hi)
    return GrMobProgress(GRMOB_PROGRESS_DETERMINATE, now.coerceIn(lo, hi), lo, hi)
}
