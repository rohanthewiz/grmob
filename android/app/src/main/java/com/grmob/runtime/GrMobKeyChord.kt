package com.grmob.runtime

/**
 * One chord of a core.AccessibilityKeyShortcuts value: "Control+Shift+P" is
 * key "P" with control and shift held.
 *
 * # Why a file of its own, with no imports
 *
 * The parse and the match are pure string logic, and a Kotlin file that
 * imports nothing can be compiled and run off a device (see android/verify's
 * run.sh). The part that needs Android, turning a KeyEvent into a key name
 * and walking the tree, is GrMobRuntime.handleKeyEvent.
 *
 * # Which chords are page-global
 *
 * A chord holding Control, Alt or Meta, or a bare function key F1 to F24. Those
 * type nothing, so answering them from anywhere on the screen steals nothing.
 * A bare key (PageDown, a letter) stays with the widget that owns it; see Go's
 * Style.AccessibilityKeyShortcuts. The same rule as the WASM runtime's
 * isPageGlobalChord.
 */
data class GrMobKeyChord(
    val key: String,
    val control: Boolean,
    val alt: Boolean,
    val meta: Boolean,
    val shift: Boolean,
) {
    val pageGlobal: Boolean
        get() = control || alt || meta || FUNCTION_KEY.matches(key)

    /**
     * Whether a key press is this chord. Modifiers match exactly, so
     * Control+S is not answered by Control+Shift+S. A one-character key
     * compares case-insensitively, because the name handleKeyEvent builds for
     * a letter is lower case whether or not Shift is held.
     */
    fun matches(key: String, control: Boolean, alt: Boolean, meta: Boolean, shift: Boolean): Boolean {
        if (control != this.control || alt != this.alt || meta != this.meta || shift != this.shift) return false
        return if (this.key.length == 1) this.key.equals(key, ignoreCase = true) else this.key == key
    }

    companion object {
        private val FUNCTION_KEY = Regex("F([1-9]|1[0-9]|2[0-4])")

        /**
         * Every chord in a space-separated value. A chord with an empty key or
         * an unknown modifier name is dropped, so a misspelling matches
         * nothing rather than something else.
         */
        fun parseAll(spec: String): List<GrMobKeyChord> =
            spec.split(' ').filter { it.isNotEmpty() }.mapNotNull { parse(it) }

        fun parse(chord: String): GrMobKeyChord? {
            val parts = chord.split('+')
            val key = parts.last()
            if (key.isEmpty()) return null
            var control = false
            var alt = false
            var meta = false
            var shift = false
            for (part in parts.dropLast(1)) {
                when (part) {
                    "Control" -> control = true
                    "Alt" -> alt = true
                    "Meta" -> meta = true
                    "Shift" -> shift = true
                    else -> return null
                }
            }
            return GrMobKeyChord(key, control, alt, meta, shift)
        }
    }
}
