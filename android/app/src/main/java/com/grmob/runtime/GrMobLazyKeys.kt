package com.grmob.runtime

// GrMobLazyKeys.kt imports nothing, so android/verify's JVM harness can run
// it off a device (see android/verify/run.sh). GrMobList calls it.

/**
 * The LazyColumn item key for each of a core.List's rows, in order, given the
 * rows' Go keys ("" for an unkeyed row). Every key returned is distinct.
 *
 * # Why the keys need making unique
 *
 * GrMobList flattens every Fragment among a List's children into one row
 * list, so each row can be recycled on its own, and each For group in Go is
 * one such Fragment. Go keys are unique within one sibling list (debug
 * mode's duplicate-key check reads one list at a time), not across groups.
 * Two groups keyed by the same IDs, a pinned and a recent copy of one
 * message, say, gave LazyColumn the same key twice after flattening, and
 * Compose throws on that ("Key … was already used"), which ends the app
 * (plan R9d).
 *
 * # The rule
 *
 *	""          the row's index (an Int): positional, as before
 *	first "k"   "k", unchanged, so a list with no repeat keeps every key it had
 *	n-th "k"    Pair("k", n-1): the occurrence, as Go's own callback-ID
 *	            scopes tell a repeated key apart (idScope.seen, "0~1")
 *
 * A repeat is a Pair rather than a spelled string like "k~1" so it cannot
 * meet a real key, whatever that key's characters. Int, String and Pair keys
 * never compare equal to one another. Each is something Android's Bundle can
 * hold (Kotlin's Pair is Serializable), which LazyColumn needs to save item
 * state.
 *
 * The occurrence count is positional. If a row with a repeated key appears
 * earlier in the list, every later copy's key moves by one, and those rows
 * lose their composed state. That is the price of a list whose keys do not
 * identify its rows, and Go's IDs pay it the same way.
 */
fun grMobLazyKeys(keys: List<String>): List<Any> {
    val seen = HashMap<String, Int>()
    return keys.mapIndexed { i, key ->
        if (key.isEmpty()) {
            i
        } else {
            val before = seen[key] ?: 0
            seen[key] = before + 1
            if (before == 0) key else Pair(key, before)
        }
    }
}
