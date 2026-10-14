package com.grmob.runtime

// core.Image's `src`, as the model the Android image loader is handed.
//
// # Why this file exists
//
// A `data:` URI is a whole image inside the src string (RFC 2397), and the
// other three targets load one with no help: a browser's <img> decodes it, and
// iOS's URLSession has a data: protocol handler of its own. Coil 2 does not —
// its fetchers cover http(s), file, content, android.resource and assets, and a
// String model with any other scheme fails to load and draws nothing. Apps
// meet data URIs whenever they show Markdown pasted from somewhere that embeds
// images (gonotes stores pasted screenshots that way).
//
// So the src is decoded here, and the bytes go to Coil as a ByteArray, which
// Coil 2 maps to its ByteBufferFetcher and decodes like any downloaded body.
// Every other src passes through untouched.
//
// # Why a hand-written base64 decoder
//
// java.util.Base64 is API 26 and the app's minSdk is 24; android.util.Base64
// is an Android class, and this file imports nothing so that the JVM harness
// in android/verify can run it off a device (as it runs GrMobTextEdits.kt).
// The decoder is small, and the harness holds it to Go's encoding/base64 over
// a generated case table (android/verify/gen.go, dataUriCases).
//
// # What it accepts
//
//	data:[<mediatype>][;base64],<data>
//
//   - The scheme is case-insensitive, as RFC 3986 makes every scheme.
//   - ";base64" must end the metadata; anything else before the comma is the
//     media type and its parameters, which Coil does not need (it sniffs the
//     image format from the bytes).
//   - Base64 payloads may carry whitespace (line-wrapped exports) and may omit
//     the trailing padding. Any other character outside the standard alphabet
//     makes the src undecodable.
//   - A payload without ";base64" is percent-encoded text: %XX escapes are
//     decoded and everything else is taken as UTF-8. A '%' that does not
//     start an escape is kept as itself, as WHATWG's percent-decode keeps it.
//
// An undecodable data URI yields null. AsyncImage treats a null model as "no
// image" and draws its empty state, which is what a broken <img> amounts to on
// the web.

/**
 * The model to hand AsyncImage for a core.Image src: the decoded bytes of a
 * `data:` URI, null for one that cannot be decoded, and [src] itself otherwise.
 */
fun grMobImageModel(src: String): Any? =
    if (isDataUri(src)) grMobDataUriBytes(src) else src

/** Whether [src] uses the data: scheme (case-insensitive). */
internal fun isDataUri(src: String): Boolean = src.regionMatches(0, "data:", 0, 5, ignoreCase = true)

/**
 * The bytes a `data:` URI carries, or null when [src] is not one or its
 * payload does not decode.
 */
fun grMobDataUriBytes(src: String): ByteArray? {
    if (!isDataUri(src)) return null
    val comma = src.indexOf(',')
    if (comma < 0) return null
    val meta = src.substring(5, comma)
    val payload = src.substring(comma + 1)
    return if (meta.endsWith(";base64", ignoreCase = true)) {
        decodeBase64(payload)
    } else {
        percentDecode(payload)
    }
}

/**
 * Standard-alphabet base64 (RFC 4648 §4), tolerant of whitespace and of
 * missing padding, strict about everything else.
 *
 * Works in quanta of four sextets: each full quantum is three bytes, and a
 * final partial quantum of two or three sextets is one or two bytes. A lone
 * sextet cannot encode a byte and is an error, as it is in Go's decoder. '='
 * may only appear as the tail of the last quantum.
 */
internal fun decodeBase64(s: String): ByteArray? {
    val out = java.io.ByteArrayOutputStream(s.length * 3 / 4)
    var acc = 0      // up to four sextets, packed high to low
    var n = 0        // sextets in acc
    var padding = 0  // '=' seen; once > 0 only more '=' or whitespace may follow
    for (c in s) {
        if (c == ' ' || c == '\n' || c == '\r' || c == '\t') continue
        if (c == '=') {
            padding++
            continue
        }
        if (padding > 0) return null
        val v = sextet(c)
        if (v < 0) return null
        acc = (acc shl 6) or v
        n++
        if (n == 4) {
            out.write(acc shr 16 and 0xFF)
            out.write(acc shr 8 and 0xFF)
            out.write(acc and 0xFF)
            acc = 0
            n = 0
        }
    }
    // Padding must fill out the last quantum exactly (or be absent).
    if (padding > 0 && (n == 0 || n + padding != 4)) return null
    when (n) {
        0 -> {}
        1 -> return null
        2 -> out.write(acc shr 4 and 0xFF)
        3 -> {
            out.write(acc shr 10 and 0xFF)
            out.write(acc shr 2 and 0xFF)
        }
    }
    return out.toByteArray()
}

/** The value of one base64 character, or -1 for a character outside the alphabet. */
private fun sextet(c: Char): Int = when (c) {
    in 'A'..'Z' -> c - 'A'
    in 'a'..'z' -> c - 'a' + 26
    in '0'..'9' -> c - '0' + 52
    '+' -> 62
    '/' -> 63
    else -> -1
}

/**
 * A percent-encoded payload's bytes: each %XX is one byte, every other
 * character is its UTF-8 encoding. A '%' not followed by two hex digits is
 * kept as a literal '%', which is WHATWG's percent-decode and so what a
 * browser does with the same data URL. It never fails; the nullable return
 * matches [decodeBase64] so the two arms read alike.
 */
internal fun percentDecode(s: String): ByteArray? {
    val out = java.io.ByteArrayOutputStream(s.length)
    var i = 0
    while (i < s.length) {
        val c = s[i]
        if (c == '%' && i + 2 < s.length) {
            val hi = Character.digit(s[i + 1], 16)
            val lo = Character.digit(s[i + 2], 16)
            if (hi >= 0 && lo >= 0) {
                out.write(hi shl 4 or lo)
                i += 3
                continue
            }
        }
        // One code point at a time, so a surrogate pair is encoded as the
        // character it is rather than as two invalid halves.
        val cp = s.codePointAt(i)
        val chars = Character.charCount(cp)
        out.write(s.substring(i, i + chars).toByteArray(Charsets.UTF_8))
        i += chars
    }
    return out.toByteArray()
}
