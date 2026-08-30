package com.grmob.app

import com.grmob.runtime.GrMobBridge
import mobile.Mobile
import mobile.PatchListener

/**
 * GrMobBridge implementation over the gomobile-generated classes.
 *
 * `mobile.Mobile` / `mobile.PatchListener` come from app/libs/grmob.aar,
 * produced by ../build.sh from the Go `mobile` package (see mobile/bridge.go
 * for the delivery contract). This replaces the old hand-rolled JNI
 * `external fun` bridge — gomobile owns the FFI now.
 *
 * The app itself is Go code: the bound app package registers its root view in
 * an init step (mobile.Register), which runs when the AAR's native library
 * loads, so by the time these calls happen the app is installed.
 */
/**
 * @param dataDir the app's writable directory (Context.filesDir) — passed in
 * because only an Android Context can name it, and this class deliberately
 * holds no Context. Registered with Go before anything renders: Go-side
 * persistence (mobile.SetDataDir / DataDir; see examples/todoapp's bytdb
 * store) hydrates on the first render pass, so this must beat
 * GrMobRuntime.start().
 */
class GomobileBridge(dataDir: String) : GrMobBridge {
    init {
        Mobile.setDataDir(dataDir)
    }

    override fun renderInitial(): String = Mobile.renderInitial()

    override fun triggerCallback(id: String): String = Mobile.triggerCallback(id)

    override fun triggerTextCallback(id: String, value: String): String =
        Mobile.triggerTextCallback(id, value)

    override fun triggerBoolCallback(id: String, value: Boolean): String =
        Mobile.triggerBoolCallback(id, value)

    override fun triggerIntCallback(id: String, value: Long): String =
        Mobile.triggerIntCallback(id, value)

    override fun setListener(listener: (String) -> Unit) {
        Mobile.setListener(object : PatchListener {
            override fun applyPatches(patches: String) = listener(patches)
        })
    }
}
