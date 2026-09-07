package com.grmob.runtime

// The picker menu, checked as behaviour rather than as source text.
//
// A `DropdownMenu`'s content is a composable lambda. Nothing outside an
// Android device can open one and read back which rows are headings, which
// refuse a tap, or which value each one dispatches — so three facts about that
// menu rested entirely on mobile/verify reading Renderer.kt as text and
// finding the right substrings in it. That is the same position the iOS side
// was in before ios/verify, and the same answer applies: GrMobSelectMenu.kt
// imports nothing at all, so the decision the menu is a direct drawing of is a
// function from a list of maps to a list of data classes, and a plain JVM can
// run it.
//
// This is that runner. It was the last of the four renderers whose
// transliteration nothing executed.
//
// What it does *not* reach is the drawing: the lines that hand `item.label` to
// a Text, `item.isDisabled` to `enabled`, and `item.value` to `textChanged`.
// mobile/verify still reads those as text, and that is now the whole of what
// it reads — exactly the split ios/verify left behind.
//
// The expectations come from Go. gen.go computes them with
// core.SelectMenuSections over the shared case table (internal/menufixture),
// so what is compared here is the *transliteration*, which is the thing that
// can drift.

/** One case from gen.go: an option list and the menu Go says it becomes. */
data class MenuCase(
    val name: String,
    val options: List<Map<String, Any?>>,
    val want: List<WantSection>,
)

/**
 * core.SelectMenuSection, generated.
 *
 * [first] is carried explicitly so the Kotlin section's own computed property
 * is compared rather than assumed — it is the key that tells two runs sharing
 * a heading apart.
 */
data class WantSection(
    val heading: String,
    val first: Int,
    val disabled: Boolean,
    val items: List<WantItem>,
)

/** core.SelectMenuItem, generated. */
data class WantItem(
    val index: Int,
    val value: String,
    val label: String,
    val disabled: Boolean,
)

/**
 * Runs every case through [grMobMenuSections] and reports each difference as
 * its own line, naming the case and the position.
 *
 * One line per property rather than a dump of two structures: a failure here
 * should say which property broke, because the fix is a different one for each
 * of them.
 */
fun checkSelectMenu(cases: List<MenuCase>): List<String> {
    val problems = mutableListOf<String>()

    for (c in cases) {
        val got = grMobMenuSections(c.options)

        if (got.size != c.want.size) {
            problems.add("${c.name}: ${got.size} sections, Go says ${c.want.size}")
            continue
        }
        got.zip(c.want).forEachIndexed { i, (g, w) ->
            if (g.heading != w.heading) {
                problems.add("${c.name} section $i: heading \"${g.heading}\", Go says \"${w.heading}\"")
            }
            // The key a run is identified by. Two runs may share a heading, so
            // this is the only thing that can tell them apart.
            if (g.first != w.first) {
                problems.add("${c.name} section $i: first ${g.first}, Go says ${w.first}")
            }
            // core.SelectOption.GroupDisabled, resolved. The dangerous wrong
            // answer is reading the declaration when the run is *opened*,
            // which passes every case whose first option carries it — hence
            // the fixture case that puts it on the last one.
            if (g.isDisabled != w.disabled) {
                problems.add("${c.name} section $i: disabled ${g.isDisabled}, Go says ${w.disabled}")
            }
            if (g.items.size != w.items.size) {
                problems.add("${c.name} section $i: ${g.items.size} options, Go says ${w.items.size}")
                return@forEachIndexed
            }
            g.items.zip(w.items).forEachIndexed { j, (gi, wi) ->
                val where = "${c.name} section $i option $j"
                if (gi.index != wi.index) {
                    problems.add("$where: index ${gi.index}, Go says ${wi.index}")
                }
                // The value is what goes up to Go when the row is tapped. The
                // dangerous wrong answer is the label, which would work in
                // every test whose labels happen to equal its values.
                if (gi.value != wi.value) {
                    problems.add("$where: value \"${gi.value}\", Go says \"${wi.value}\"")
                }
                if (gi.label != wi.label) {
                    problems.add("$where: label \"${gi.label}\", Go says \"${wi.label}\"")
                }
                if (gi.isDisabled != wi.disabled) {
                    problems.add("$where: disabled ${gi.isDisabled}, Go says ${wi.disabled}")
                }
            }
        }
    }
    return problems
}

// --- The accessibility value range -----------------------------------------
//
// The second transliteration this harness runs, and it arrived the same way
// the first did. core.ValueRange.Progress decides what three wire strings
// amount to — a determinate bar, ARIA's indeterminate one, an empty range, or
// no numeric claim — and GrMobProgress.kt is that decision in Kotlin.
//
// It used to be a three-way branch inside `grMobValue`, an extension on
// SemanticsPropertyReceiver, so nothing off-device could reach it: what
// checked it was mobile/verify looking for the word `Indeterminate` somewhere
// in GrMobStyle.kt, which is a check that the word is present and not that the
// branch reaches it on the right condition.
//
// What this does not reach is the assignment — the two lines that hand the
// reading to ProgressBarRangeInfo. mobile/verify still reads those as text,
// which is the same split GrMobSelectMenu.kt's drawing is left in.

/** One case from gen.go: a wire range and the reading Go says it has. */
data class ProgressCase(
    val name: String,
    val now: String,
    val min: String,
    val max: String,
    val text: String,
    val want: WantProgress,
)

/** core.Progress, generated. */
data class WantProgress(
    val reading: String,
    val now: Float,
    val min: Float,
    val max: Float,
)

/**
 * Runs every case through [grMobProgressNumber] and [grMobProgressOf] — the
 * parse and the decision, in that order, because that is the order a renderer
 * applies them and the parse is half of what can go wrong.
 *
 * The numbers are compared exactly rather than within a tolerance. Both sides
 * are parsing the same decimal string and neither does arithmetic on it, so a
 * difference here is a different *rule*, not a different rounding — and Go
 * emits the literals with shortest round-tripping precision so a Float can
 * hold them.
 */
fun checkProgress(cases: List<ProgressCase>): List<String> {
    val problems = mutableListOf<String>()

    for (c in cases) {
        val got = grMobProgressOf(
            grMobProgressNumber(c.now),
            grMobProgressNumber(c.min),
            grMobProgressNumber(c.max),
        )
        val where = "${c.name} (now=\"${c.now}\" min=\"${c.min}\" max=\"${c.max}\")"

        if (got.reading != c.want.reading) {
            problems.add("$where: reading \"${got.reading}\", Go says \"${c.want.reading}\"")
            // The numbers only mean anything under an agreed reading, so a
            // disagreement here would report three more differences that are
            // all the same one.
            continue
        }
        if (got.now != c.want.now) {
            problems.add("$where: now ${got.now}, Go says ${c.want.now}")
        }
        if (got.min != c.want.min) {
            problems.add("$where: min ${got.min}, Go says ${c.want.min}")
        }
        if (got.max != c.want.max) {
            problems.add("$where: max ${got.max}, Go says ${c.want.max}")
        }
    }
    return problems
}

fun main() {
    // A guard on the fixture itself. The table is generated by another
    // program, and an empty one would make every assertion above vacuous while
    // reporting success.
    if (menuCases.size < 6) {
        System.err.println("FAIL: only ${menuCases.size} menu cases were generated")
        kotlin.system.exitProcess(1)
    }

    val problems = checkSelectMenu(menuCases)
    if (problems.isNotEmpty()) {
        System.err.println("FAIL: the Kotlin picker menu disagrees with Go:")
        problems.forEach { System.err.println("  $it") }
        kotlin.system.exitProcess(1)
    }
    println("OK: ${menuCases.size} picker menus match Go's decomposition")

    // The same guard, for the same reason: a table generated by another
    // program can arrive empty, and an empty one passes every assertion.
    if (progressCases.size < 12) {
        System.err.println("FAIL: only ${progressCases.size} value ranges were generated")
        kotlin.system.exitProcess(1)
    }

    val valueProblems = checkProgress(progressCases)
    if (valueProblems.isNotEmpty()) {
        System.err.println("FAIL: the Kotlin value range disagrees with Go:")
        valueProblems.forEach { System.err.println("  $it") }
        kotlin.system.exitProcess(1)
    }
    println("OK: ${progressCases.size} value ranges match Go's reading")
}
