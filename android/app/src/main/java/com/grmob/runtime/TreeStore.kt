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
class TreeStore {
    var root by mutableStateOf<GrMobNode?>(null)
        private set

    /** Mounts the initial full tree (the RenderInitial payload). */
    fun mount(json: String) {
        if (!json.trimStart().startsWith("{")) {
            // Not a tree — most likely a Go-side error report. Surface it in
            // full (logcat truncates single lines) instead of crashing on the
            // JSON parse and burying the real failure.
            json.chunked(3000).forEach { Log.e("GrMob", it) }
            return
        }
        root = GrMobNode.parse(JSONObject(json))
    }

    /** Applies one patch batch (the RenderAgain / push payload). */
    fun applyPatches(json: String) {
        if (!json.trimStart().startsWith("[")) {
            // Same guard, and the same reason, as mount above: render.renderJSON
            // returns {"error":"failed to encode JSON"} when a payload will not
            // marshal (a NaN in Props is enough), and JSONArray() on that throws
            // out of the main-thread handler and takes the app down — burying the
            // encode failure that actually caused it. Swift's TreeStore has always
            // logged and returned here; this matches it.
            json.chunked(3000).forEach { Log.e("GrMob", it) }
            return
        }
        val patches = JSONArray(json)
        for (i in 0 until patches.length()) {
            val p = patches.getJSONObject(i)
            apply(
                type = p.getString("Type"),
                path = p.getString("TargetID"),
                changes = p.opt("Changes"),
            )
        }
    }

    private fun apply(type: String, path: String, changes: Any?) {
        when (type) {
            "update-props" -> resolve(path)?.props =
                GrMobNode.parseProps(changes as? JSONObject)

            "update-style" -> resolve(path)?.style =
                GrMobStyle.parse(changes as? JSONObject)

            "replace" -> {
                val node = GrMobNode.parse(changes as? JSONObject ?: return)
                val (parent, index) = parentOf(path) ?: run {
                    // Path is "root" itself: swap the whole tree.
                    if (path == ROOT) root = node else warn(type, path)
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
        // rather than crash; the next full replace re-synchronizes.
        Log.w("GrMob", "patch $type could not resolve $path")
    }

    private companion object {
        const val ROOT = "root"
    }
}
