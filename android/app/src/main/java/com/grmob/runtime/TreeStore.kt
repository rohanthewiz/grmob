package com.grmob.runtime

import android.util.Log
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.setValue
import org.json.JSONArray
import org.json.JSONObject

/**
 * Holds the live node tree and applies Go's reconciler patches to it.
 *
 * This replaces the old PatchRenderer's path→View map. Patches are resolved
 * against the *current* tree at apply time by walking the positional path, so
 * there is no stale-path cache to drift out of sync after structural changes.
 * The ordering guarantees the Go side documents (patches applied in emitted
 * order; sibling removals arrive highest-index-first) are exactly what makes
 * this walk safe within a batch.
 *
 * Threading: every method must be called on the main thread. GrMobRuntime
 * funnels both delivery paths (synchronous event returns and async pushes)
 * through a single main-thread handler, which also preserves the bridge's
 * arrival-order contract.
 */
/**
 * What one [TreeStore.mount] cost, split at its one internal seam.
 *
 * Both stages scale with the payload, which is the whole reason they are
 * measured: the open question on Android is how much of a cold launch a
 * smaller initial tree would buy back. See [Startup] for the diagram and for
 * how to get these printed.
 *
 * @param chars the payload's length in UTF-16 code units, which is what a
 *   Kotlin String can report without re-encoding 400KB on the launch path.
 *   Equal to Go's byte count for ASCII and smaller wherever the payload is
 *   not — a "—" is one unit here and three bytes there — so read it as the
 *   same number to within the prose's punctuation, not as the byte count.
 * @param parseNanos org.json's time turning the string into JSONObjects.
 * @param buildNanos GrMobNode.parse's time turning those into the snapshot
 *   state tree Compose reads.
 */
data class MountStats(val chars: Int, val parseNanos: Long, val buildNanos: Long)

class TreeStore {
    var root by mutableStateOf<GrMobNode?>(null)
        private set

    /**
     * Called when this tree is found to have drifted from Go's: a patch whose
     * path resolves to nothing, a payload that is not a patch list, or a patch
     * that threw. GrMobRuntime answers it with mobile.Resync, whose replace of
     * "root" puts Go's tree back (N-133). Patches are positional and
     * incremental, so a tree that is wrong once stays wrong; the log line that
     * used to say "the next full replace re-synchronizes" was true, but
     * nothing ever asked for one.
     */
    var onDrift: (() -> Unit)? = null

    /**
     * Once per drift, not once per symptom: the stale patches already queued
     * behind the one that dangled will dangle too, and each asking again
     * would only fetch the same tree again. Cleared by a replace of "root",
     * which is the answer (or a navigation, which resets it too).
     */
    private var resyncPending = false

    private fun drifted() {
        if (resyncPending) return
        resyncPending = true
        onDrift?.invoke()
    }

    /**
     * Mounts the initial full tree (the RenderInitial payload).
     *
     * Returns what the mount cost, split at the one seam inside it, or null if
     * the payload was not a tree. See [MountStats] and [Startup] for why the
     * split is worth the two extra clock reads on a once-per-process path.
     */
    fun mount(json: String): MountStats? {
        if (!json.trimStart().startsWith("{")) {
            // Not a tree — most likely a Go-side error report. Surface it in
            // full (logcat truncates single lines) instead of crashing on the
            // JSON parse and burying the real failure.
            json.chunked(3000).forEach { Log.e("GrMob", it) }
            return null
        }
        // The two stages are timed separately because they are separate levers:
        // the parse is org.json's cost for the bytes, the build is ours for the
        // nodes, and a payload that halved in bytes but not in nodes (or the
        // reverse) would move only one of them.
        val t0 = System.nanoTime()
        val obj = JSONObject(json)
        val t1 = System.nanoTime()
        root = GrMobNode.parse(obj)
        val t2 = System.nanoTime()
        return MountStats(chars = json.length, parseNanos = t1 - t0, buildNanos = t2 - t1)
    }

    /** Applies one patch batch (the RenderAgain / push payload). */
    fun applyPatches(json: String) {
        if (!json.trimStart().startsWith("[")) {
            // Same guard, and the same reason, as mount above: render.renderJSON
            // returns {"error":"failed to encode JSON"} when a payload will not
            // marshal (a NaN in Props is enough), and JSONArray() on that throws
            // out of the main-thread handler and takes the app down — burying the
            // encode failure that actually caused it. Swift's TreeStore has always
            // logged and returned here; this matches it. (Go stopped sending
            // that string with N-133, which drops such a pass in Go; the guard
            // stays for any payload this cannot parse.) Either way the pass
            // never arrived, so the tree has drifted.
            json.chunked(3000).forEach { Log.e("GrMob", it) }
            drifted()
            return
        }
        // A patch that throws (a malformed payload, a Changes of the wrong
        // shape) used to take the main-thread handler, and the app, with it.
        // The patches before it have landed, so the tree is now part one pass
        // and part another: logged, and Go is asked for its own.
        try {
            val patches = JSONArray(json)
            for (i in 0 until patches.length()) {
                val p = patches.getJSONObject(i)
                apply(
                    type = p.getString("Type"),
                    path = p.getString("TargetID"),
                    changes = p.opt("Changes"),
                )
            }
        } catch (e: Exception) {
            Log.e("GrMob", "a patch threw; asking Go for its tree", e)
            drifted()
        }
    }

    private fun apply(type: String, path: String, changes: Any?) {
        when (type) {
            // A miss used to be silent here (a safe call), and an update is
            // the commonest patch, so most drift went unseen.
            "update-props" -> (resolve(path) ?: return warn(type, path)).props =
                GrMobNode.parseProps(changes as? JSONObject)

            "update-style" -> (resolve(path) ?: return warn(type, path)).style =
                GrMobStyle.parse(changes as? JSONObject)

            "replace" -> {
                val node = GrMobNode.parse(changes as? JSONObject ?: return)
                val (parent, index) = parentOf(path) ?: run {
                    // Path is "root" itself: swap the whole tree. That is a
                    // resync's answer, so whatever had drifted is gone now.
                    if (path == ROOT) {
                        root = node
                        resyncPending = false
                    } else {
                        warn(type, path)
                    }
                    return
                }
                if (index in parent.children.indices) parent.children[index] = node
                else warn(type, path)
            }

            // "add" targets the slot the node should occupy; "add-child"
            // targets the parent and always appends. Both reduce to an insert
            // clamped to the current child count.
            "add" -> {
                val node = GrMobNode.parse(changes as? JSONObject ?: return)
                val (parent, index) = parentOf(path) ?: return warn(type, path)
                parent.children.add(index.coerceIn(0, parent.children.size), node)
            }
            "add-child" -> {
                val node = GrMobNode.parse(changes as? JSONObject ?: return)
                val parent = resolve(path) ?: return warn(type, path)
                parent.children.add(node)
            }

            "remove", "remove-child" -> {
                val (parent, index) = parentOf(path) ?: return warn(type, path)
                if (index in parent.children.indices) parent.children.removeAt(index)
                else warn(type, path)
            }

            else -> warn(type, path)
        }
    }

    /** Walks a positional path ("root/0/2") to its node, or null if it dangles. */
    private fun resolve(path: String): GrMobNode? {
        var node = root ?: return null
        if (path == ROOT) return node
        for (seg in path.removePrefix("$ROOT/").split('/')) {
            val idx = seg.toIntOrNull() ?: return null
            node = node.children.getOrNull(idx) ?: return null
        }
        return node
    }

    /** Resolves a path to (parent node, child index); null for "root" or a dangling path. */
    private fun parentOf(path: String): Pair<GrMobNode, Int>? {
        val cut = path.lastIndexOf('/')
        if (cut < 0) return null
        val index = path.substring(cut + 1).toIntOrNull() ?: return null
        val parent = resolve(path.substring(0, cut)) ?: return null
        return parent to index
    }

    private fun warn(type: String, path: String) {
        // A dangling patch means the Go and Kotlin trees disagree — log loudly
        // rather than crash, and ask Go for its tree (drifted).
        Log.w("GrMob", "patch $type could not resolve $path")
        drifted()
    }

    private companion object {
        const val ROOT = "root"
    }
}
