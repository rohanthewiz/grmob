package main

import (
	"encoding/base64"
	"fmt"
	"net/url"
	"strings"
)

// The data: URI table: core.Image srcs the Android image loader cannot fetch
// (Coil 2 has no data: fetcher), and the bytes they carry.
//
// GrMobDataUri.kt decodes them by hand, because java.util.Base64 is API 26
// against a minSdk of 24 and the file imports nothing so this harness can run
// it. A hand-written decoder is exactly what needs an independent statement of
// the answers, so Want comes from Go's encoding/base64 and net/url rather than
// from a second transliteration. The browser and iOS (URLSession) decode data
// URIs natively and need no table.
//
// The inputs are built by encoding known bytes, so every valid case is a real
// encoding of something rather than a hand-typed base64 string, and the
// invalid cases are named for the rule each one breaks.

// dataURICase is one src and what Go decodes it to; Want is nil when the src
// must not decode.
type dataURICase struct {
	Name string
	Src  string
	Want []byte
}

// onePixelPNG is a complete 1x1 PNG: the shape of the payloads apps actually
// carry (gonotes embeds pasted screenshots), including bytes above 0x7F.
var onePixelPNG = []byte{
	0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a, 0x00, 0x00, 0x00, 0x0d,
	0x49, 0x48, 0x44, 0x52, 0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x01,
	0x08, 0x06, 0x00, 0x00, 0x00, 0x1f, 0x15, 0xc4, 0x89, 0x00, 0x00, 0x00,
	0x0d, 0x49, 0x44, 0x41, 0x54, 0x78, 0xda, 0x63, 0xfc, 0xcf, 0xc0, 0xf0,
	0x1f, 0x00, 0x05, 0x05, 0x02, 0x00, 0x5f, 0xc8, 0xf1, 0xd2, 0x00, 0x00,
	0x00, 0x00, 0x49, 0x45, 0x4e, 0x44, 0xae, 0x42, 0x60, 0x82,
}

// goDataURIBytes is the reference decoder: Go's own answer for a src.
//
// Whitespace is dropped before a base64 payload is decoded (line-wrapped
// exports are common, and both browsers and iOS accept them). A payload with
// padding goes through StdEncoding, which demands the padding be exact; one
// without goes through RawStdEncoding. A payload without ";base64" is
// percent-decoded with url.PathUnescape, which leaves '+' alone as a data URL
// must.
func goDataURIBytes(src string) []byte {
	if len(src) < 5 || !strings.EqualFold(src[:5], "data:") {
		return nil
	}
	comma := strings.IndexByte(src, ',')
	if comma < 0 {
		return nil
	}
	meta, payload := src[5:comma], src[comma+1:]
	if strings.HasSuffix(strings.ToLower(meta), ";base64") {
		p := strings.Map(func(r rune) rune {
			if r == ' ' || r == '\n' || r == '\r' || r == '\t' {
				return -1
			}
			return r
		}, payload)
		enc := base64.RawStdEncoding
		if strings.Contains(p, "=") {
			enc = base64.StdEncoding
		}
		b, err := enc.DecodeString(p)
		if err != nil {
			return nil
		}
		return nonNil(b)
	}
	s, err := url.PathUnescape(payload)
	if err != nil {
		return nil
	}
	return nonNil([]byte(s))
}

// nonNil keeps "decoded to nothing" (data:,) apart from "did not decode".
func nonNil(b []byte) []byte {
	if b == nil {
		return []byte{}
	}
	return b
}

// wrap breaks s into lines of n characters joined by sep, as a mail or PEM
// style exporter does.
func wrap(s string, n int, sep string) string {
	var parts []string
	for len(s) > n {
		parts = append(parts, s[:n])
		s = s[n:]
	}
	return strings.Join(append(parts, s), sep)
}

func dataURICases() []dataURICase {
	png := base64.StdEncoding.EncodeToString(onePixelPNG)
	srcs := []struct{ name, src string }{
		{"a padded PNG", "data:image/png;base64," + png},
		{"the same PNG unpadded", "data:image/png;base64," + strings.TrimRight(png, "=")},
		{"wrapped at 16 with LF", "data:image/png;base64," + wrap(png, 16, "\n")},
		{"wrapped at 20 with CRLF and spaces", "data:image/png;base64," + wrap(png, 20, "\r\n  ")},
		{"one byte, padded", "data:;base64," + base64.StdEncoding.EncodeToString([]byte("a"))},
		{"one byte, unpadded", "data:;base64," + base64.RawStdEncoding.EncodeToString([]byte("a"))},
		{"two bytes, padded", "data:;base64," + base64.StdEncoding.EncodeToString([]byte("ab"))},
		{"two bytes, unpadded", "data:;base64," + base64.RawStdEncoding.EncodeToString([]byte("ab"))},
		{"three bytes", "data:;base64," + base64.StdEncoding.EncodeToString([]byte("abc"))},
		{"every sextet", "data:;base64," + base64.StdEncoding.EncodeToString([]byte{0xfb, 0xff, 0xbf, 0x00, 0x10, 0x83})},
		{"upper-case scheme and BASE64", "DATA:image/gif;BASE64," + png},
		{"media type with a parameter", "data:image/png;name=shot.png;base64," + png},
		{"empty base64 payload", "data:image/png;base64,"},
		{"empty plain payload", "data:,"},
		{"percent-encoded SVG", "data:image/svg+xml," + url.PathEscape(`<svg xmlns="http://www.w3.org/2000/svg"/>`)},
		{"plain text with UTF-8 and a plus", "data:text/plain,caf%C3%A9+é"},
		{"a character outside the alphabet", "data:;base64,QUJD*"},
		{"a lone trailing sextet", "data:;base64,QUJDR"},
		{"data after the padding", "data:;base64,QQ==QUJD"},
		{"padding one short", "data:;base64,QQ="},
		{"padding alone", "data:;base64,="},
		{"no comma", "data:image/png;base64"},
		{"an https src", "https://example.com/a.png"},
	}
	out := make([]dataURICase, 0, len(srcs))
	for _, s := range srcs {
		out = append(out, dataURICase{Name: s.name, Src: s.src, Want: goDataURIBytes(s.src)})
	}
	return out
}

// writeDataURICases emits the table. A byte list crosses as Ints (Kotlin has
// no unsigned byte literal that reads naturally), and "does not decode" as
// null.
func writeDataURICases(b *strings.Builder) {
	b.WriteString(`
// GENERATED by android/verify/gen.go (datauri.go) from Go's encoding/base64
// and net/url.
//
// Each case is a core.Image src and the bytes Go decodes it to, or null when
// it must not decode.

val dataUriCases: List<DataUriCase> = listOf(
`)
	for _, c := range dataURICases() {
		want := "null"
		if c.Want != nil {
			ints := make([]string, len(c.Want))
			for i, x := range c.Want {
				ints[i] = fmt.Sprint(x)
			}
			want = "listOf(" + strings.Join(ints, ", ") + ")"
		}
		fmt.Fprintf(b, "    DataUriCase(%s, %s, %s),\n", kstr(c.Name), kstr(c.Src), want)
	}
	b.WriteString(")\n")
}
