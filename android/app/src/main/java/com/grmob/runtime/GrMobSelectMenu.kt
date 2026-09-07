package com.grmob.runtime

/**
 * A picker's option list, decomposed into the menu a person sees.
 *
 * Split out of GrMobSelect (Renderer.kt) for the same reason its Swift twin —
 * GrMobSelectMenu.swift — is split out of Renderer.swift: a `DropdownMenu`'s
 * content is a composable lambda, and no test that does not launch an Android
 * device can open the menu and read back which rows are headings, which
 * refuse a tap, or which value each one dispatches. What is easy to get wrong,
 * though, is a function from a list of maps to a list of data classes.
 *
 * This file imports nothing — no Compose, no Android — so the decomposition is
 * plain Kotlin that a JVM harness could run. There is no such harness on this
 * side yet (the Android build's only check is `compileDebugKotlin`, and
 * mobile/verify reads this source as text), which is the honest state of it:
 * the *shape* here is what makes one possible, and the rule itself is pinned
 * by core.SelectMenuSections (core/select_menu.go) and by ios/verify running
 * the Swift transliteration against cases generated from it.
 *
 * The rule is core.SelectOption.Group's. The comments here say what the code
 * does; that field says why.
 */

/**
 * One choosable row of the menu: an option resolved out of the flat wire map.
 *
 * [index] is the option's position in the original list. It is carried because
 * two options are allowed to share a label — core.Select's Value is the
 * identity, the Label is written to be read — so a row needs something else to
 * be known by.
 */
data class GrMobMenuItem(
    val index: Int,
    val value: String,
    val label: String,
    val isDisabled: Boolean,
)

/**
 * One run of consecutive options sharing a heading.
 *
 * An empty [heading] is the run of options that stand on their own at the top
 * level of the list. That is a real section rather than an absence of one, so
 * the drawing code is a single loop over sections with the heading item as its
 * only branch.
 *
 * [isDisabled] is core.SelectOption.GroupDisabled, resolved: the whole run is
 * unavailable. Every item of such a run is itself disabled — the propagation
 * happens when the run is closed, below — so on this target the flag adds
 * nothing the renderer has to read: Material's dropdown has no section
 * construct, and the heading item this renderer writes ahead of a run is
 * already `enabled = false`, which is what makes it a label rather than a
 * choice. It is carried because it is part of the shape core states, and a
 * transliteration that drops a field cannot be compared with the authority.
 */
data class GrMobMenuSection(
    val heading: String,
    val isDisabled: Boolean,
    val items: List<GrMobMenuItem>,
) {
    /** The index the run starts at — what identifies it when a heading cannot,
     *  since core.SelectOption.Group allows the same heading either side of a
     *  different one and that is two sections. */
    val first: Int get() = items.firstOrNull()?.index ?: 0
}

/**
 * Splits a picker's flattened options into runs by their "group".
 *
 * Runs, not a gather: consecutive options sharing a heading are one section,
 * in the order they were written. A run is closed by the *next* option naming
 * a different heading, so the last run of a list has nothing following it and
 * is flushed after the loop — the one piece of bookkeeping this rule needs,
 * and the piece each renderer that reimplemented it had to remember.
 */
fun grMobMenuSections(options: List<Map<String, Any?>>): List<GrMobMenuSection> {
    val sections = mutableListOf<GrMobMenuSection>()
    // The run being filled, and the heading it was opened with. `building` is
    // separate from the heading because the empty heading is a legitimate one
    // and would otherwise be indistinguishable from "nothing open yet".
    var heading = ""
    var items = mutableListOf<GrMobMenuItem>()
    var runDisabled = false
    var building = false

    // Closing a run is two steps, not one. A run's disabled state is stated by
    // *any* of its options (core.SelectOption.GroupDisabled), so it is not
    // known until the run ends — and the items collected before it was known
    // have to be marked on the way out. This is the half of the rule that is
    // new since the flush; the flush itself is the older half.
    fun closeRun() {
        val resolved = if (runDisabled) items.map { it.copy(isDisabled = true) } else items
        sections.add(GrMobMenuSection(heading, runDisabled, resolved))
    }

    options.forEachIndexed { i, option ->
        val group = option["group"] as? String ?: ""
        if (!building || group != heading) {
            if (building) closeRun()
            heading = group
            items = mutableListOf()
            runDisabled = false
            building = true
        }
        if ((option["groupDisabled"] as? String ?: "") == "true") runDisabled = true
        val value = option["value"] as? String ?: ""
        // The label defaulted to the value at core.Select's flattening seam,
        // so this fallback is for a hand-assembled node that never went
        // through it — a menu row with no text is worse than one showing its
        // value.
        val label = (option["label"] as? String ?: "").ifEmpty { value }
        items.add(
            GrMobMenuItem(
                index = i,
                value = value,
                label = label,
                // The wire carries the string "true", core.SelectedState's
                // spelling one property over. Anything else is not disabled.
                isDisabled = (option["disabled"] as? String ?: "") == "true",
            )
        )
    }
    if (building) closeRun()
    return sections
}
