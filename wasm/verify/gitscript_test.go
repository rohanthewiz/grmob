package main

import (
	"os"
	"path"
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
//	a heredoc body is lexed     only when the command it feeds is one that RUNS
//	only for some commands      its stdin — see shellFromStdin. `cat <<EOF` and
//	                            `jq <<EOF` are data and are skipped, so a git
//	                            call reached through a wrapper this file has
//	                            not heard of is missed
//	line numbers inside a       reported as the line the quote OPENED on,
//	quoted region               which is the line somebody looks at anyway
//	a git in command position   if a string that is not a command happens to
//	inside a string             begin with the word `git`, it is reported.
//	                            The cost is a message naming a line
//
// # Why a heredoc is read at all, given that it started out as data
//
// It was skipped outright, on the grounds that a heredoc is input being fed to
// a command and the one in this repository is a JSON literal. That is true of
// that heredoc and not of the shape:
//
//	ssh host <<EOF        the body is a script, run by the remote shell
//	git ls-tree --name-only HEAD
//	EOF
//
// is a deploy script's ordinary spelling, and every path it lists comes back
// C-quoted. The body is already delimited, so lexing it costs nothing but the
// decision of WHEN — and skipping it always is one answer to that decision,
// not the absence of one.
//
// The discriminator is the command, because that is what the question actually
// is: does anything run this text. `cat <<EOF` and `jq <<EOF` are data, and
// lexing a JSON body as shell would report every `"git …"` string in it. So a
// body is lexed only for a command that runs its stdin, and the words are
// scanned rather than just the first one — `docker exec -i c bash <<EOF` and
// `sudo -u x ssh host <<EOF` are both the shape, and neither has the shell in
// command position.
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
			// A heredoc. The redirection itself ends nothing: `git ls-tree -z
			// <<EOF` is still a git call, so the word assembly carries on
			// across the body either way.
			//
			// Whether the BODY is commands is a question about the command it
			// is being fed to — see the header. `strings.Contains` is the same
			// short-circuit the quoted case uses: a body with no `git` in it
			// anywhere cannot produce a finding, and most bodies are that.
			if i+1 < len(src) && src[i+1] == '<' {
				body, bodyLine, next := heredocBody(src, i, &line)
				if depth < 3 && strings.Contains(body, "git") &&
					feedsAShell(words, word.String(), started) {
					out = append(out, lexShell(body, bodyLine, depth+1)...)
				}
				i = next
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

// The commands that RUN their standard input rather than reading it as data.
//
// A heredoc fed to one of these is a script; a heredoc fed to anything else is
// input. `ssh` is here because the remote end is a shell — that is what ssh
// with no command IS — and it is the member of this list a repository is most
// likely to grow a git call under.
//
// A word is matched on its BASE, so `/bin/sh` and `/usr/bin/ssh` are the same
// answer as `sh` and `ssh`. Nothing here is matched loosely: `shell.py` and
// `mysh` are not shells, and a list that guessed at them would lex a Python
// heredoc as commands.
var shellFromStdin = map[string]bool{
	"ssh":  true,
	"sh":   true,
	"bash": true,
	"zsh":  true,
	"dash": true,
	"ash":  true,
	"ksh":  true,
	"mksh": true,
}

// feedsAShell is whether the command being assembled runs its stdin.
//
// Every word is looked at rather than the first, because the shell is usually
// not the command: `docker exec -i c bash`, `kubectl exec -i p -- sh`, `sudo
// -u deploy ssh host`. That is loose in the direction of lexing a body that is
// data — the cost of which is a finding naming a line somebody can look at —
// and tight in the direction that matters, which is not missing a script.
//
// `partial` is the word still being assembled when the `<<` was reached, since
// `ssh host<<EOF` is legal and leaves `host` unflushed.
func feedsAShell(words []string, partial string, started bool) bool {
	for _, w := range words {
		if shellFromStdin[path.Base(w)] {
			return true
		}
	}
	return started && shellFromStdin[path.Base(partial)]
}

// heredocBody is a heredoc's body, the line that body starts on, and the index
// of the last byte the redirection consumed. The line counter is advanced over
// the whole thing.
//
// This was skipHeredoc, which returned only the third of those. The body is
// returned now because the caller decides whether it is a script — see
// feedsAShell — and the line is returned because a finding inside one has to
// report its position in the enclosing FILE rather than in the body.
//
// The delimiter may be quoted (`<<'EOF'`) and `<<-` strips leading tabs from
// the terminator, both of which are spelled here because both appear in
// ordinary scripts and getting either wrong would swallow the rest of a file.
// The tab stripping is applied only when LOOKING FOR the terminator: the body
// is handed back as it was written, and the shell lexer does not care about
// leading whitespace.
func heredocBody(src string, at int, line *int) (body string, bodyLine, end int) {
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
	term := delim.String()
	if term == "" {
		return "", 0, at + 1
	}
	// The body starts on the line after the one the redirection is on, and the
	// rest of THAT line is still shell — but a heredoc is nearly always last on
	// its line, and treating the remainder as body costs nothing this
	// repository has. Advance to the newline first.
	for i < len(src) && src[i] != '\n' {
		i++
	}
	bodyLine = *line + 1
	var out strings.Builder
	for i < len(src) {
		i++ // past the newline
		*line++
		start := i
		for i < len(src) && src[i] != '\n' {
			i++
		}
		got := src[start:i]
		looking := got
		if dash {
			looking = strings.TrimLeft(got, "\t")
		}
		if looking == term {
			return out.String(), bodyLine, i - 1
		}
		out.WriteString(got)
		out.WriteByte('\n')
		if i >= len(src) {
			break
		}
	}
	// An unterminated heredoc. The body is whatever was collected, which is
	// the rest of the file — a script in that state does not run, and the
	// alternative is discarding a call this check exists to find.
	return out.String(), bodyLine, len(src) - 1
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

// The lexer finds a git call in the shapes that hide one, and does not invent
// one where there is none.
//
// # Why this exists beside the walk above
//
// The walk is over this repository, and this repository has exactly ONE git
// call in a script. So the walk's whole reading is one invocation, and every
// rule the lexer carries — command position, quoted recursion, redirection
// targets, comments that are not comments, heredoc bodies — is exercised by a
// file that happens not to contain the shape it is about. Those rules were
// checked by hand, once, by editing scripts and reverting them; a rule checked
// that way is a rule that was true on an afternoon.
//
// The cases below are the shapes themselves, so the lexer answers for them on
// every run. That matters most for the two directions this file is loose in:
// a shape it must NOT report (the seventeen mentions of `git` in a comment
// block) and a shape it must (a body a remote shell runs).
//
// Each case names the invocations it expects, `git` dropped, as they would
// appear in a finding.
func TestTheScriptLexerFindsAGitCallInTheShapesThatHideIt(t *testing.T) {
	cases := []struct {
		name string
		kind scriptLang
		src  string
		want []string
	}{{
		name: "a plain call",
		kind: shellScript,
		src:  "git ls-tree --name-only -z HEAD\n",
		want: []string{"ls-tree --name-only -z HEAD"},
	}, {
		// The reason this is a lexer. session-doc-check.sh holds the word
		// eighteen times and one of them is a call.
		name: "the word in prose, above a call",
		kind: shellScript,
		src: "# git writes a path back C-quoted, so this git call asks git\n" +
			"# for -z. Not every git listing does; this one does.\n" +
			"git diff --cached --name-only -z\n",
		want: []string{"diff --cached --name-only -z"},
	}, {
		name: "a mention that is an argument, not a command",
		kind: shellScript,
		src:  "grep git README.md\necho git ls-files\n",
		want: nil,
	}, {
		name: "leading assignments and a prefix word",
		kind: shellScript,
		src:  "GIT_DIR=x env LC_ALL=C git ls-files -z\n",
		want: []string{"ls-files -z"},
	}, {
		// The quoted recursion: the region is folded into the word AND lexed
		// on its own, so the command inside `sh -c` is a command.
		name: "a call inside sh -c",
		kind: shellScript,
		src:  "sh -c 'git ls-tree --name-only HEAD'\n",
		want: []string{"ls-tree --name-only HEAD"},
	}, {
		// The finding that came out of a break-test: without the redirection
		// rule the arguments read `diff --name-only 2 /dev/null`.
		name: "a redirection is not an argument",
		kind: shellScript,
		src:  "git diff --name-only -z 2>/dev/null\n",
		want: []string{"diff --name-only -z"},
	}, {
		name: "a call after a separator and inside a subshell",
		kind: shellScript,
		src:  "cd x && git status -z\n(git ls-files -z)\n",
		want: []string{"status -z", "ls-files -z"},
	}, {
		// `#` is only a comment where a word can begin.
		name: "a hash mid-word is not a comment",
		kind: shellScript,
		src:  "n=$#\ngit ls-files -z\n",
		want: []string{"ls-files -z"},
	}, {
		// The item this case is here for. A heredoc fed to a shell is a
		// script, and every path the call below lists comes back C-quoted.
		name: "a heredoc a remote shell runs",
		kind: shellScript,
		src:  "ssh host <<EOF\ngit ls-tree --name-only HEAD\nEOF\necho done\n",
		want: []string{"ls-tree --name-only HEAD"},
	}, {
		name: "a heredoc a shell runs, reached past other words",
		kind: shellScript,
		src:  "docker exec -i c bash <<'SH'\ngit ls-files\nSH\n",
		want: []string{"ls-files"},
	}, {
		name: "a dash heredoc, terminator indented",
		kind: shellScript,
		src:  "\tssh host <<-EOF\n\tgit status --porcelain\n\tEOF\n",
		want: []string{"status --porcelain"},
	}, {
		// The other half of the discriminator: a body nothing runs stays data,
		// which is what keeps a JSON literal from being lexed as commands.
		name: "a heredoc that is data",
		kind: shellScript,
		src:  "cat <<EOF > out.json\n{\"cmd\": \"git ls-tree --name-only\"}\nEOF\n",
		want: nil,
	}, {
		// The line counting has to survive the body, or every finding after a
		// heredoc names the wrong line. Checked by position below.
		name: "a call after a data heredoc",
		kind: shellScript,
		src:  "cat <<EOF\nnot\na\nscript\nEOF\ngit ls-files -z\n",
		want: []string{"ls-files -z"},
	}, {
		name: "a call from javascript, in a string",
		kind: jsScript,
		src:  "const out = execSync(\"git ls-tree --name-only HEAD\")\n",
		want: []string{"ls-tree --name-only HEAD"},
	}, {
		name: "a mention from javascript, in a comment",
		kind: jsScript,
		src:  "// run git ls-tree --name-only here one day\nconst x = 1\n",
		want: nil,
	}}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			var got []string
			for _, cmd := range scriptGitCommands(c.kind, c.src) {
				got = append(got, strings.Join(cmd.args, " "))
			}
			if strings.Join(got, " | ") != strings.Join(c.want, " | ") {
				t.Errorf("the lexer read %d invocation(s) and this shape has "+
					"%d.\n\ngot:  %v\nwant: %v\n\nsource:\n%s",
					len(got), len(c.want), got, c.want, c.src)
			}
		})
	}
}

// A finding inside a heredoc names the line it is on in the FILE.
//
// Separate from the table above because that one compares argument lists and
// this is about the other half of a finding. A check that reports the right
// call at the wrong line sends a reader to a line that is fine, and a heredoc
// is where the counting is easiest to get wrong: the body is consumed by a
// function of its own, and every line of it has to reach the counter whether
// the body was lexed or skipped.
func TestALineNumberSurvivesAHeredoc(t *testing.T) {
	// Line 1 is the `ssh`, 2 the call inside the body, 3 the terminator, 4 the
	// echo, 5 the call after it.
	const src = "ssh host <<EOF\ngit ls-tree --name-only HEAD\nEOF\necho done\ngit status\n"
	got := scriptGitCommands(shellScript, src)
	if len(got) != 2 {
		t.Fatalf("expected the call inside the body and the one after it, got "+
			"%d: %v", len(got), got)
	}
	if got[0].line != 2 {
		t.Errorf("the call inside the heredoc body is reported at line %d and "+
			"it is on line 2 of the file. A body is lexed as a fragment, so "+
			"the line it STARTS on has to be carried in — see heredocBody's "+
			"second return value.", got[0].line)
	}
	if got[1].line != 5 {
		t.Errorf("the call after the heredoc is reported at line %d and it is "+
			"on line 5. Every line of a body has to reach the counter, "+
			"terminator included, whether or not the body was lexed.",
			got[1].line)
	}
}
