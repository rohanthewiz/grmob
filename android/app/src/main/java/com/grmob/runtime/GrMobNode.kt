package com.grmob.runtime

import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateListOf
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.setValue
import org.json.JSONArray
import org.json.JSONObject

/**
 * One node of the live UI tree, mirroring Go's core.Node.
 *
 * This is a *data* tree, not a view tree: composables read from it and
 * Compose's snapshot system does the rest. Each mutable aspect is backed by
 * snapshot state so a patch write invalidates exactly the composable scopes
 * that read it —
 *
 *   patch            mutates              recomposes
 *   ─────            ───────              ──────────
 *   update-props  →  node.props        →  only composables reading this
 *                                         node's props
 *   update-style  →  node.style        →  only this node's box/text styling
 *   add/remove/   →  parent.children   →  only the parent's children loop
 *   replace          (SnapshotStateList)  (siblings keep their composition
 *                                         state thanks to key())
 *
 * `type` and `key` are immutable by design: the Go reconciler never mutates
 * a node across those axes — it emits `replace`, which swaps in a fresh
 * GrMobNode instance here.
 */
class GrMobNode(
    val type: String,
    val key: String,
    props: Map<String, Any?>,
    style: GrMobStyle?,
    children: List<GrMobNode>,
) {
    var props by mutableStateOf(props)
    var style by mutableStateOf(style)
    val children = mutableStateListOf<GrMobNode>().apply { addAll(children) }

    /** Typed prop accessors; Go serializes props with lowercase keys. */
    fun stringProp(name: String): String = props[name] as? String ?: ""
    fun boolProp(name: String): Boolean = props[name] as? Boolean ?: false
    fun intProp(name: String): Int = (props[name] as? Number)?.toInt() ?: 0
    fun doubleProp(name: String): Double = (props[name] as? Number)?.toDouble() ?: 0.0

    companion object {
        /** Decodes a Go core.Node JSON object (keys are the Go field names). */
        fun parse(obj: JSONObject): GrMobNode {
            val childArray = obj.optJSONArray("Children")
            val children = ArrayList<GrMobNode>(childArray?.length() ?: 0)
            if (childArray != null) {
                for (i in 0 until childArray.length()) {
                    // Go child slots can hold JSON null (nil *Node); skip them —
                    // the reconciler's Diff treats nil slots as absent too.
                    val child = childArray.optJSONObject(i) ?: continue
                    children.add(parse(child))
                }
            }
            return GrMobNode(
                type = obj.optString("Type"),
                key = obj.optString("Key"),
                props = parseProps(obj.optJSONObject("Props")),
                style = GrMobStyle.parse(obj.optJSONObject("Style")),
                children = children,
            )
        }

        /** Converts a JSON props object into plain Kotlin maps/lists/scalars. */
        fun parseProps(obj: JSONObject?): Map<String, Any?> {
            if (obj == null) return emptyMap()
            val out = LinkedHashMap<String, Any?>(obj.length())
            for (k in obj.keys()) out[k] = fromJson(obj.get(k))
            return out
        }

        private fun fromJson(v: Any?): Any? = when (v) {
            null, JSONObject.NULL -> null
            is JSONObject -> {
                val m = LinkedHashMap<String, Any?>(v.length())
                for (k in v.keys()) m[k] = fromJson(v.get(k))
                m
            }
            is JSONArray -> (0 until v.length()).map { fromJson(v.get(it)) }
            else -> v // String, Boolean, Int, Long, Double pass through
        }
    }
}
