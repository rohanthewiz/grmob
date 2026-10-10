package com.grmob.app

import android.content.Context
import android.content.SharedPreferences
import android.security.keystore.KeyGenParameterSpec
import android.security.keystore.KeyProperties
import android.util.Base64
import android.util.Log
import org.json.JSONObject
import java.security.KeyStore
import java.util.concurrent.Executors
import javax.crypto.AEADBadTagException
import javax.crypto.Cipher
import javax.crypto.KeyGenerator
import javax.crypto.SecretKey
import javax.crypto.spec.GCMParameterSpec

/**
 * The Android half of Go's keystore package (keystore/keystore.go).
 *
 *   keystore.Save   ──▶ "keystore" {command: "save",   id, key, value}
 *   keystore.Get    ──▶ "keystore" {command: "get",    id, key}
 *   keystore.Delete ──▶ "keystore" {command: "delete", id, key}
 *                   ◀── host event "keystore" {id, ok, found, value, error}
 *
 * Every request is answered, failures included, because Go holds the
 * caller's callback until the id comes back and has no timeout to fall back
 * on (the same contract as Clipboard).
 *
 * # The construction
 *
 *   AndroidKeyStore                     SharedPreferences "grmob_keystore"
 *   ┌───────────────────────┐           ┌──────────────────────────────────┐
 *   │ "grmob.keystore"      │  seals    │ <key> → base64(                  │
 *   │ AES-256, GCM only,    │ ────────▶ │   0x01 | iv(12) | ciphertext+tag │
 *   │ never exported        │           │ )                                │
 *   └───────────────────────┘           └──────────────────────────────────┘
 *
 * One AES key, generated inside the Keystore (in the TEE where the device
 * has one) and never readable by this process: the app can only ask the
 * Keystore to encrypt or decrypt with it. Each value is sealed with AES-GCM
 * and a fresh IV the Keystore picks, and the entry's own name is the
 * cipher's associated data, so a ciphertext copied from one entry onto
 * another fails authentication rather than reading back as the wrong secret.
 * The leading 0x01 is a format version: a later change of construction can
 * recognise and migrate old entries instead of failing them.
 *
 * Not EncryptedSharedPreferences: androidx.security-crypto is deprecated, and
 * it is this same construction plus a Tink dependency. Not a key per entry:
 * Keystore operations are IPC to a system daemon, and one key keeps a read to
 * one IPC rather than a lookup and a decryption.
 *
 * No user authentication is required to use the key. A credential is what
 * the app needs at launch and in the background, before anyone has touched
 * the screen; a biometric gate is a separate feature ("FaceID / Biometric"
 * on the roadmap), not a property of where the token lives.
 *
 * # When the key is gone
 *
 * Auto Backup (allowBackup in the manifest) restores the preferences file
 * onto a reinstall but cannot restore the key, which never leaves the
 * device's Keystore. Every entry in the file is then sealed by a key that no
 * longer exists, and can never be read. So the moment this object has to
 * *create* the key, it first clears the file: those entries are noise, and
 * the honest answer to Get is found=false — the same as on iOS, where the
 * items are ThisDeviceOnly and are never restored to new hardware at all.
 * An individual entry that fails authentication under the current key
 * (corrupted, or tampered with by root) is dropped the same way.
 *
 * # Threading
 *
 * [SystemEvents] delivers on the main thread, but Keystore calls are IPC
 * and the first key generation can take a noticeable fraction of a second,
 * so every request hops to one worker thread. One thread, not a pool: it
 * keeps requests in the order Go sent them, so a Save followed by a Get
 * reads what was saved. The reply goes straight from the worker to
 * GrMobRuntime.hostEvent, which is already a thread-safe hop onto the event
 * executor.
 */
object Keystore {
    private const val TAG = "GrMobKeystore"
    private const val PROVIDER = "AndroidKeyStore"
    private const val ALIAS = "grmob.keystore"
    private const val PREFS = "grmob_keystore"
    private const val TRANSFORMATION = "AES/GCM/NoPadding"
    private const val FORMAT: Byte = 1
    private const val IV_BYTES = 12
    private const val TAG_BITS = 128

    private val worker = Executors.newSingleThreadExecutor { r ->
        Thread(r, "grmob-keystore").apply { isDaemon = true }
    }

    // Written once on the main thread by attach, read on the worker.
    @Volatile private var prefs: SharedPreferences? = null
    @Volatile private var report: ((String, String) -> Unit)? = null

    // Touched only on the worker thread: the loaded key, so each request is
    // one Keystore operation rather than a KeyStore.load and a lookup first.
    private var secret: SecretKey? = null

    /** Retains the application context's preferences only, and the reporter. */
    fun attach(context: Context, out: (String, String) -> Unit) {
        prefs = context.applicationContext.getSharedPreferences(PREFS, Context.MODE_PRIVATE)
        report = out
    }

    fun handle(data: JSONObject) {
        val id = data.optString("id")
        if (id.isEmpty()) return
        val command = data.optString("command")
        val key = data.optString("key")
        val value = data.optString("value")
        worker.execute { perform(id, command, key, value) }
    }

    private fun perform(id: String, command: String, key: String, value: String) {
        val store = prefs
        if (store == null) {
            fail(id, "keystore host not attached")
            return
        }
        // Go refuses an empty key before sending; this guards a host-side
        // caller, not Go.
        if (key.isEmpty()) {
            fail(id, "empty key")
            return
        }
        try {
            when (command) {
                "save" -> {
                    val sealed = seal(sealingKey(store), key, value)
                    // commit, not apply: this is the worker thread, and the
                    // answer to Save has to be whether the bytes reached disk.
                    if (store.edit().putString(key, sealed).commit()) {
                        succeed(id)
                    } else {
                        fail(id, "SharedPreferences commit failed")
                    }
                }
                "get" -> {
                    val k = sealingKey(store) // may clear entries a lost key sealed
                    val sealed = store.getString(key, null)
                    if (sealed == null) {
                        found(id, null)
                    } else {
                        found(id, openOrDrop(store, k, key, sealed))
                    }
                }
                "delete" -> {
                    // No key needed to forget an entry, and removing a name
                    // that holds nothing is a success: sign-out clears the
                    // token whether or not one was saved.
                    if (store.edit().remove(key).commit()) {
                        succeed(id)
                    } else {
                        fail(id, "SharedPreferences commit failed")
                    }
                }
                // A newer Go with a command this shell predates: answered, so
                // the caller's callback is not stranded.
                else -> fail(id, "unknown command '$command'")
            }
        } catch (e: Exception) {
            // The value is never logged; the key name is not a secret.
            Log.w(TAG, "$command '$key' failed", e)
            fail(id, "${e.javaClass.simpleName}: ${e.message ?: "no message"}")
        }
    }

    /**
     * The one AES key, loaded from the Keystore or created there on first use.
     *
     * Creating it is the moment to clear the preferences file — see "When the
     * key is gone" above. The clear happens before generation so that a crash
     * between the two leaves an empty file and no key, which the next call
     * handles the same way, rather than a new key beside unreadable entries.
     */
    private fun sealingKey(store: SharedPreferences): SecretKey {
        secret?.let { return it }
        val keyStore = KeyStore.getInstance(PROVIDER).apply { load(null) }
        (keyStore.getKey(ALIAS, null) as? SecretKey)?.let {
            secret = it
            return it
        }
        if (store.all.isNotEmpty()) {
            Log.w(TAG, "no key for ${store.all.size} sealed entries (restored from a backup?); discarding them")
            store.edit().clear().commit()
        }
        val generator = KeyGenerator.getInstance(KeyProperties.KEY_ALGORITHM_AES, PROVIDER)
        generator.init(
            KeyGenParameterSpec.Builder(ALIAS, KeyProperties.PURPOSE_ENCRYPT or KeyProperties.PURPOSE_DECRYPT)
                .setBlockModes(KeyProperties.BLOCK_MODE_GCM)
                .setEncryptionPaddings(KeyProperties.ENCRYPTION_PADDING_NONE)
                .setKeySize(256)
                .build(),
        )
        return generator.generateKey().also { secret = it }
    }

    /** Seals one value: version byte, the IV the Keystore chose, then ciphertext+tag. */
    private fun seal(k: SecretKey, name: String, plain: String): String {
        val cipher = Cipher.getInstance(TRANSFORMATION)
        // No IV is passed: an AndroidKeyStore key requires randomized
        // encryption by default and refuses a caller-chosen IV, which is the
        // right default — a repeated GCM IV under one key is catastrophic.
        cipher.init(Cipher.ENCRYPT_MODE, k)
        cipher.updateAAD(name.toByteArray(Charsets.UTF_8))
        val body = cipher.doFinal(plain.toByteArray(Charsets.UTF_8))
        val iv = cipher.iv
        check(iv.size == IV_BYTES) { "unexpected GCM IV length ${iv.size}" }
        return Base64.encodeToString(byteArrayOf(FORMAT) + iv + body, Base64.NO_WRAP)
    }

    /**
     * Opens one entry, or drops it and answers "not found" when it can never
     * be opened: bad base64, an unknown format byte, or a GCM tag that does
     * not verify under the current key. Anything else — a Keystore IPC
     * failure, say — propagates as an error, because it may succeed next
     * time and the entry must not be thrown away for it.
     */
    private fun openOrDrop(store: SharedPreferences, k: SecretKey, name: String, sealed: String): String? {
        val raw = try {
            Base64.decode(sealed, Base64.NO_WRAP)
        } catch (e: IllegalArgumentException) {
            return drop(store, name, "not base64")
        }
        if (raw.size <= 1 + IV_BYTES || raw[0] != FORMAT) {
            return drop(store, name, "unrecognised format")
        }
        val cipher = Cipher.getInstance(TRANSFORMATION)
        cipher.init(Cipher.DECRYPT_MODE, k, GCMParameterSpec(TAG_BITS, raw, 1, IV_BYTES))
        cipher.updateAAD(name.toByteArray(Charsets.UTF_8))
        val plain = try {
            cipher.doFinal(raw, 1 + IV_BYTES, raw.size - 1 - IV_BYTES)
        } catch (e: AEADBadTagException) {
            return drop(store, name, "authentication failed")
        }
        return String(plain, Charsets.UTF_8)
    }

    private fun drop(store: SharedPreferences, name: String, why: String): String? {
        Log.w(TAG, "discarding unreadable entry '$name': $why")
        store.edit().remove(name).commit()
        return null
    }

    private fun succeed(id: String) = send(JSONObject().put("id", id).put("ok", true))

    /** A get's answer: the value when there is one, found=false when null. */
    private fun found(id: String, value: String?) {
        val payload = JSONObject().put("id", id).put("ok", true).put("found", value != null)
        if (value != null) payload.put("value", value)
        send(payload)
    }

    private fun fail(id: String, reason: String) =
        send(JSONObject().put("id", id).put("ok", false).put("error", reason))

    private fun send(payload: JSONObject) {
        report?.invoke("keystore", payload.toString())
    }
}
