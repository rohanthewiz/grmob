package main

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

// The -z rule again, over the files a go/parser walk cannot see.
//
// # Why this exists beside TestEveryGitListingAsksForNulSeparatedPaths
//
// That check parses every Go file in the repository and holds each git call
// that lists paths to asking for -z. It is exact, because it is looking at one
// call's argument list in a syntax tree.
//
// It is also blind to every git invocation that is not Go. When it was
// written, that set was empty. It stopped being empty in the same session:
// .claude/hooks/session-doc-check.sh runs `git diff --cached --name-only -z`,
// follows the rule by hand, and carries a comment saying so — and the comment
// was the only thing holding it. The file the rule was written FOR became one
// of the files nothing checked.
//
// The rest of the set is the three platform `verify/run.sh`, `build.sh`, the
// gate scripts, and any .mjs that ever shells out. None of them touches git
// today. The point is that none of them is stopped from doing so tomorrow, and
// the failure mode is a file silently missing from a listing rather than an
// error — which is the shape this repository writes arms for rather than
// grepping for once.
//
// # Why this is a lexer and not a grep, given that a grep was the alternative
//
// The item that asked for this said a grep "would be loose in a way the parse
// is not — it cannot tell a comment from a call". session-doc-check.sh is the
// proof: it holds eighteen occurrences of the word `git` and exactly ONE of
// them is an invocation. The other seventeen are prose in the comment block
// explaining why the one asks for -z. A grep reports eighteen findings, or it
// reports none and the file is exempted, and either way the check is gone.
//
// So the text is lexed:
//
//	sh    quotes, backslash escapes, and `#` as a comment only where a comment
//	      can start — not inside a string, not mid-word ($#, ${#x}, a URL
//	      fragment). Commands are split on ; & | ( ) newline and backtick
//	js    // and /* */ stripped, then every string and template literal is
//	      lexed AS a shell command, because that is what a git call looks like
//	      from JavaScript: it is an argument to execSync, not syntax
//
// Quoted regions in sh are lexed twice on purpose — once as part of the word
// they sit in, and once on their own — so `sh -c 'git ls-tree --name-only'` is
// seen as the git call it is rather than as a single opaque argument to sh.
//
// # What it is loose about, said plainly
//
// Three things, and all three are in the direction of missing a call rather
// than inventing one, except the last:
//
//	heredoc bodies are skipped   a heredoc is data being fed to a command —
//	                             the one in this repository is a JSON literal
//	                             — so `ssh host <<EOF … git ls-tree … EOF`
//	                             would not be found
//	line numbers inside a        reported as the line the quote OPENED on,
//	quoted region                which is the line somebody looks at anyway
//	a git in command position    if a string that is not a command happens to
//	inside a string              begin with the word `git`, it is reported.
//	                             The cost is a message naming a line
//
// The first is why this is a companion to the Go check and not a replacement
// for it: a parse tree has no equivalent hole.
func TestEveryGitListingInAScriptAsksForNulSeparatedPaths(t *testing.T) {
	root := filepath.Join("..", "..")
	_, considered, from, err := citingFiles(root)
	if err != nil {
		t.Fatalf("enumerating the repository (%s): %v", from, err)
	}
	paths := make([]string, 0, len(considered))
	for p := range considered {
		paths = append(paths, p)
	}
	// Sorted for the reason the Go check sorts: a check whose findings arrive
	// in map order cannot be diffed against its last run.
	sort.Strings(paths)

	files := 0
	seen := 0
	for _, rel := range paths {
		// This file's own prose names the subcommands and the flags, and it is
		// not a script, but the enumeration does not know that — skipped by
		// name for the same reason gitquoting_test.go skips itself.
		if rel == "wasm/verify/gitscript_test.go" {
			continue
		}
		src, readErr := os.ReadFile(filepath.Join(root, filepath.FromSlash(rel)))
		if readErr != nil {
			continue
		}
		kind := scriptKind(rel, src)
		if kind == notAScript {
			continue
		}
		files++
		for _, cmd := range scriptGitCommands(kind, string(src)) {
			seen++
			if !gitListsPaths(cmd.args) || hasArg(cmd.args, "-z") {
				continue
			}
			t.Errorf("%s:%d asks git for a listing of paths and does not ask "+
				"for -z:\n\n  git %s\n\n"+
				"git C-quotes any path it cannot write literally, so `core/ä.go` "+
				"arrives as `\"core/\\303\\244.go\"`, quotes included, and the "+
				"quoting rule is not even the same between subcommands — "+
				"ls-tree writes a space literally and status --porcelain quotes "+
				"it. -z makes every record a NUL-terminated path with no "+
				"escaping.\n\n"+
				"In a POSIX shell there is no NUL-safe read, so the records "+
				"have to be turned back into lines with `tr '\\0' '\\n'` — "+
				"which gives back the newline case alone. That is a real "+
				"concession and .claude/hooks/session-doc-check.sh writes down "+
				"why it can afford it. Make the same argument here, or read "+
				"the listing somewhere that can hold a NUL.",
				rel, cmd.line, strings.Join(cmd.args, " "))
		}
	}

	// The enumeration reaching the tree at all, and reaching the ONE call this
	// was written for. Every arm here that walks the repository says this: a
	// walk that found nothing passes silently and reads as a clean result —
	// and a lexer is a great deal easier to break into finding nothing than a
	// parser is.
	if files == 0 || seen == 0 {
		t.Fatalf("the script walk found %d git invocation(s) in %d script(s) "+
			"out of %d file(s) enumerated by %s. This repository has at least "+
			"one — .claude/hooks/session-doc-check.sh runs `git diff --cached "+
			"--name-only -z` — so the lexer is not reaching it and this check "+
			"is over nothing.", seen, files, len(paths), from)
	}
	t.Logf("%d git invocation(s) across %d script(s); every one that lists "+
		"paths asks for -z. Enumerated by %s.", seen, files, from)
}

// Which lexer a file gets, or that it gets none.
type scriptLang int

const (
	notAScript scriptLang = iota
	shellScript
	jsScript
)

// scriptKind is the language of a file, by extension and then by shebang.
//
// The shebang pass is what catches a script with no extension — `aria/fetch`
// rather than `aria/fetch.sh` — which is a shape this repository does not use
// today and which costs one line to keep covered. Only the first line is
// looked at, and only its first bytes, so a Go file whose first line happens
// to mention sh is not caught by accident.
func scriptKind(rel string, src []byte) scriptLang {
	switch filepath.Ext(rel) {
	case ".sh", ".bash", ".zsh":
		return shellScript
	case ".js", ".mjs", ".cjs":
		return jsScript
	}
	if !strings.HasPrefix(string(src), "#!") {
		return notAScript
	}
	line := string(src)
	if i := strings.IndexByte(line, '\n'); i >= 0 {
		line = line[:i]
	}
	switch {
	case strings.Contains(line, "sh"):
		return shellScript
	case strings.Contains(line, "node"):
		return jsScript
	}
	return notAScript
}

// One git invocation found in a script: its arguments after the word `git`,
// and the line it starts on.
type scriptCommand struct {
	args []string
	line int
}

// scriptGitCommands is every git invocation in one script.
func scriptGitCommands(kind scriptLang, src string) []scriptCommand {
	var out []scriptCommand
	switch kind {
	case shellScript:
		out = lexShell(src, 1, 0)
	case jsScript:
		// The top level of a .js file is not a command line, so nothing is
		// lexed as one. What is: the contents of every string and template
		// literal, because a git call reaches a shell from JavaScript as an
		// ARGUMENT — execSync("git ls-tree …") — and never as syntax.
		for _, s := range jsStrings(src) {
			out = append(out, lexShell(s.text, s.line, 1)...)
		}
	}
	return out
}

// Words that may stand in front of a command without changing what the command
// is, plus the shape of a leading environment assignment.
//
// `git` has to be in COMMAND POSITION for this to report it, which is what
// keeps the eighteen mentions of the word in session-doc-check.sh from being
// eighteen findings. These are the things that can legitimately precede it.
var commandPrefixes = map[string]bool{
	"env":     true,
	"command": true,
	"exec":    true,
	"sudo":    true,
	"time":    true,
	"nohup":   true,
	"builtin": true,
	"then":    true,
	"do":      true,
	"else":    true,
	"!":       true,
}

// lexShell is every git invocation in a fragment of shell.
//
// `line` is the line the fragment starts on, so a fragment lifted out of a
// quoted region or a JavaScript string reports positions in the enclosing
// FILE. `depth` bounds the recursion into quoted regions: a quoted region is
// lexed on its own as well as being folded into the word it sits in, and
// without a bound a file of nested quotes would be lexed exponentially.
func lexShell(src string, line, depth int) []scriptCommand {
	var out []scriptCommand
	// The words of the command being assembled, and whether the current word
	// has any characters in it yet — which is how `#` tells a comment from a
	// fragment identifier in the middle of a word.
	var words []string
	var word strings.Builder
	started := false
	cmdLine := line
	// Set by a redirection operator: the word after `>` or `<` is a FILE and
	// not an argument to the command. Without this, `git diff --name-only
	// 2>/dev/null` reports its arguments as `diff --name-only 2 /dev/null` —
	// which changes no verdict, since neither piece is a listing flag, but
	// puts a command in the failure message that nobody wrote.
	discardNext := false

	flushWord := func() {
		if !started {
			return
		}
		if discardNext {
			discardNext = false
		} else {
			words = append(words, word.String())
		}
		word.Reset()
		started = false
	}
	// endCommand closes the command being assembled and keeps it if it is a
	// git call. The prefix words are dropped first, so `env FOO=1 git status`
	// is a git call and `grep git` is not.
	endCommand := func() {
		flushWord()
		defer func() { words = nil; cmdLine = line; discardNext = false }()
		for len(words) > 0 {
			w := words[0]
			if commandPrefixes[w] || (strings.Contains(w, "=") &&
				!strings.HasPrefix(w, "=")) {
				words = words[1:]
				continue
			}
			break
		}
		if len(words) == 0 || words[0] != "git" {
			return
		}
		out = append(out, scriptCommand{args: append([]string(nil), words[1:]...),
			line: cmdLine})
	}

	for i := 0; i < len(src); i++ {
		c := src[i]
		switch c {
		case '\n':
			endCommand()
			line++
			cmdLine = line
		case '\\':
			// The next byte is literal, whatever it is. A backslash-newline is
			// a line continuation and joins the command, which falls out of
			// this: the newline is consumed here rather than reaching the case
			// above, so only the counter has to be told.
			if i+1 < len(src) {
				if src[i+1] == '\n' {
					line++
				} else {
					word.WriteByte(src[i+1])
					started = true
				}
				i++
			}
		case '\'', '"':
			quote := c
			j := i + 1
			var inner strings.Builder
			for ; j < len(src); j++ {
				if src[j] == '\\' && quote == '"' && j+1 < len(src) {
					inner.WriteByte(src[j+1])
					j++
					continue
				}
				if src[j] == quote {
					break
				}
				inner.WriteByte(src[j])
			}
			text := inner.String()
			word.WriteString(text)
			started = true
			// Lexed again on its own, so a command passed to `sh -c` is a
			// command here too. Bounded, and the bound is generous: this
			// repository's deepest is one level.
			if depth < 3 && strings.Contains(text, "git") {
				out = append(out, lexShell(text, line, depth+1)...)
			}
			line += strings.Count(text, "\n")
			i = j
		case '#':
			// A comment only where one can begin: at the start of a word. In
			// the middle of one this is $#, ${#x}, a URL fragment, or a colour.
			if started {
				word.WriteByte('#')
				break
			}
			for i < len(src) && src[i] != '\n' {
				i++
			}
			endCommand()
			line++
			cmdLine = line
		case ';', '&', '|', '(', ')', '`', '{', '}':
			endCommand()
		case ' ', '\t', '\r':
			flushWord()
		case '<':
			// A heredoc, whose body is data rather than commands — see the
			// header for what that misses. The redirection itself ends nothing:
			// `git ls-tree -z <<EOF` is still a git call, so only the BODY is
			// skipped and the word assembly carries on.
			if i+1 < len(src) && src[i+1] == '<' {
				i = skipHeredoc(src, i, &line)
				break
			}
			flushWord()
			discardNext = true
		case '>':
			flushWord()
			// The file descriptor a redirection names is written up against
			// the operator (`2>`), so it has already been flushed as a word.
			if n := len(words); n > 0 && isAllDigits(words[n-1]) {
				words = words[:n-1]
			}
			discardNext = true
		default:
			word.WriteByte(c)
			started = true
		}
	}
	endCommand()
	return out
}

// isAllDigits is whether a word is a file descriptor rather than an argument.
func isAllDigits(s string) bool {
	if s == "" {
		return false
	}
	for i := 0; i < len(s); i++ {
		if s[i] < '0' || s[i] > '9' {
			return false
		}
	}
	return true
}

// skipHeredoc advances past a heredoc body, returning the index of its last
// consumed byte and advancing the line counter over it.
//
// The delimiter may be quoted (`<<'EOF'`) and `<<-` strips leading tabs from
// the terminator, both of which are spelled here because both appear in
// ordinary scripts and getting either wrong would swallow the rest of a file.
func skipHeredoc(src string, at int, line *int) int {
	i := at + 2
	dash := false
	if i < len(src) && src[i] == '-' {
		dash = true
		i++
	}
	for i < len(src) && (src[i] == ' ' || src[i] == '\t') {
		i++
	}
	var delim strings.Builder
	for i < len(src) && src[i] != '\n' && src[i] != ' ' && src[i] != '\t' &&
		src[i] != ';' && src[i] != '&' {
		if src[i] != '\'' && src[i] != '"' {
			delim.WriteByte(src[i])
		}
		i++
	}
	end := delim.String()
	if end == "" {
		return at + 1
	}
	// The body starts on the line after the one the redirection is on, and the
	// rest of THAT line is still shell — but a heredoc is nearly always last on
	// its line, and treating the remainder as body costs nothing this
	// repository has. Advance to the newline first.
	for i < len(src) && src[i] != '\n' {
		i++
	}
	for i < len(src) {
		i++ // past the newline
		*line++
		start := i
		for i < len(src) && src[i] != '\n' {
			i++
		}
		got := src[start:i]
		if dash {
			got = strings.TrimLeft(got, "\t")
		}
		if got == end {
			return i - 1
		}
		if i >= len(src) {
			break
		}
	}
	return len(src) - 1
}

// One string literal lifted out of JavaScript, with the line it opened on.
type jsString struct {
	text string
	line int
}

// jsStrings is every string and template literal in a JavaScript file, with
// comments removed first.
//
// Comment removal is the whole reason this is not a regexp: `// git ls-tree`
// in a comment and `execSync("git ls-tree")` are the same characters to a
// pattern and opposite answers to the question.
//
// A regexp literal is not distinguished from division, and it does not need to
// be: the worst case is a fragment of source lexed as a shell command, which
// produces a finding only if it begins with the word `git`.
func jsStrings(src string) []jsString {
	var out []jsString
	line := 1
	for i := 0; i < len(src); i++ {
		switch src[i] {
		case '\n':
			line++
		case '/':
			if i+1 >= len(src) {
				continue
			}
			if src[i+1] == '/' {
				for i < len(src) && src[i] != '\n' {
					i++
				}
				line++
			} else if src[i+1] == '*' {
				j := strings.Index(src[i+2:], "*/")
				if j < 0 {
					return out
				}
				line += strings.Count(src[i:i+2+j+2], "\n")
				i = i + 2 + j + 1
			}
		case '\'', '"', '`':
			quote := src[i]
			j := i + 1
			var inner strings.Builder
			for ; j < len(src); j++ {
				if src[j] == '\\' && j+1 < len(src) {
					inner.WriteByte(src[j+1])
					j++
					continue
				}
				if src[j] == quote {
					break
				}
				// An unterminated ordinary string cannot span a line, so a
				// newline inside one means this was not a string at all —
				// most likely an apostrophe in a comment that the pass above
				// did not remove. Give up on it rather than swallowing the
				// rest of the file.
				if src[j] == '\n' && quote != '`' {
					break
				}
				inner.WriteByte(src[j])
			}
			text := inner.String()
			if strings.Contains(text, "git") {
				out = append(out, jsString{text: text, line: line})
			}
			line += strings.Count(src[i:min(j+1, len(src))], "\n")
			i = j
		}
	}
	return out
}
