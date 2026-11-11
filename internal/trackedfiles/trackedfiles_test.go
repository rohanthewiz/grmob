package trackedfiles

import (
	"bytes"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// executableMagics are the first bytes of a compiled native program, the
// formats `go build` writes on the platforms this repository is built on.
//
// Mach-O is listed in both byte orders and both word sizes. The fat
// (universal) header, CA FE BA BE, is left out on purpose: a Java .class file
// begins with the same four bytes, and the Android shell is a Gradle project.
// A universal binary is not what `go build` makes, so leaving it out costs
// nothing this check exists for.
var executableMagics = []struct {
	name  string
	magic []byte
}{
	{"ELF", []byte{0x7f, 'E', 'L', 'F'}},
	{"Mach-O 32-bit", []byte{0xfe, 0xed, 0xfa, 0xce}},
	{"Mach-O 64-bit", []byte{0xfe, 0xed, 0xfa, 0xcf}},
	{"Mach-O 32-bit, little-endian", []byte{0xce, 0xfa, 0xed, 0xfe}},
	{"Mach-O 64-bit, little-endian", []byte{0xcf, 0xfa, 0xed, 0xfe}},
	{"PE (Windows)", []byte{'M', 'Z'}},
}

// executableKind names the format whose magic head begins with, or "".
func executableKind(head []byte) string {
	for _, m := range executableMagics {
		if bytes.HasPrefix(head, m.magic) {
			return m.name
		}
	}
	return ""
}

// No compiled program is tracked (plan Part D, item 1).
//
// Two were: `chat` at the root, a 3.9 MB Mach-O that `go build
// ./examples/chat` leaves in the working directory, and cmd/docs/docs, the
// 9.6 MB docs server `go build` leaves beside its source. Between them they
// were a fifth of every module download and in every clone, and nothing
// read them (the docs image builds its server from source). A build product
// has no business in the history, and a binary is the one kind of file no
// review reads, so this holds the line by content rather than by name: a
// binary under any name, anywhere in the tree, fails here.
func TestNoTrackedFileIsACompiledProgram(t *testing.T) {
	root, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("no git on PATH: this check is a question about what git tracks")
	}
	// -z: paths as git stores them, NUL-separated, never quoted.
	out, err := exec.Command("git", "-C", root, "ls-files", "--cached", "-z").Output()
	if err != nil {
		t.Skipf("git could not list this tree (%v): not a checkout, nothing tracked to check", err)
	}

	var found []string
	checked := 0
	for _, rel := range strings.Split(string(out), "\x00") {
		if rel == "" {
			continue
		}
		f, err := os.Open(filepath.Join(root, filepath.FromSlash(rel)))
		if err != nil {
			// An index entry with no file on disk (deleted, unstaged; a sparse
			// checkout): there is nothing here to read.
			continue
		}
		head := make([]byte, 4)
		n, _ := io.ReadFull(f, head)
		f.Close()
		checked++
		if kind := executableKind(head[:n]); kind != "" {
			found = append(found, rel+" ("+kind+")")
		}
	}
	if checked == 0 {
		t.Fatal("git listed no readable files, so this check checked nothing")
	}
	if len(found) > 0 {
		t.Fatalf("%d tracked file(s) are compiled programs:\n  %s\n\n"+
			"A build product does not belong in the history. Untrack it with "+
			"`git rm --cached <path>`, and add the path to .gitignore so the "+
			"next build does not bring it back.",
			len(found), strings.Join(found, "\n  "))
	}
}

// The magics match the formats they claim to, and a file that only starts
// like one does not slip through on its extension. These are the heads of
// the two files the check was written for, and of files that are allowed.
func TestTheMagicsTellAProgramFromAFile(t *testing.T) {
	for _, tc := range []struct {
		head []byte
		want string
	}{
		{[]byte{0xcf, 0xfa, 0xed, 0xfe}, "Mach-O 64-bit, little-endian"}, // chat, cmd/docs/docs (arm64)
		{[]byte{0x7f, 'E', 'L', 'F'}, "ELF"},                             // the same build on Linux CI
		{[]byte{'P', 'K', 0x03, 0x04}, ""},                               // gradle-wrapper.jar, a zip
		{[]byte{0xca, 0xfe, 0xba, 0xbe}, ""},                             // a Java class file
		{[]byte{0x00, 'a', 's', 'm'}, ""},                                // a wasm module (wasm/main.wasm is ignored, but allowed)
		{[]byte{0x89, 'P', 'N', 'G'}, ""},                                // the screenshots under docs/
		{[]byte("pack"), ""},
		{nil, ""}, // an empty file
	} {
		if got := executableKind(tc.head); got != tc.want {
			t.Errorf("executableKind(% x) = %q, want %q", tc.head, got, tc.want)
		}
	}
}
