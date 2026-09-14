package main

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// .githooks/pre-push, run against a throwaway repository.
//
// # Why a live run and not a lexer check
//
// The hook's claims are about what git and gofmt do together: which commits
// `git rev-list` names for a push, that `git archive` of one commit's Go
// files is that commit's tree, which files `git diff` says a commit changed,
// and which of those findings block. None of that is visible in the text. A
// temporary repository with three commits is the smallest thing that has the
// shape:
//
//	commit 1  "unformatted"  a.go unformatted ("func  F(){}")
//	commit 2  "adds b"       b.go added, clean; a.go carried unchanged
//	commit 3  "formatted"    a.go gofmt-clean                 ◄── the tip
//
// # Pushes, and what each must answer
//
//	push                      setting   checks      exit  report
//	────                      ───────   ──────      ────  ──────
//	new branch (1..3)         unset     1, 2, 3     1     a.go under 1; 2 says
//	                                                      "listed above", not a.go
//	new branch (1..3)         true      1, 2, 3     1     the same
//	new branch (1..3)         false     3           0
//	remote has 1, push 2..3   unset     2, 3        0     a.go noted under 2
//	                                                      as from before the push
//	remote has 1, push 2      unset     2 (tip)     1     the same file blocks at
//	                                                      the tip: CI checks it
//
// A new branch has no remote-tracking refs here, so the hook's
// `--not --remotes` branch is the one taken. "Remote has 1" sends commit 1 as
// the remote's old tip, which this clone holds, so `<remote>..<local>` is.
//
// Global and system git config are shut out (GIT_CONFIG_GLOBAL, NOSYSTEM), so
// a machine's own hooksPath, signing or template settings cannot change what
// is measured.
func TestThePrePushHookChecksEveryCommitAndBlamesTheOneThatChangedAFile(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("no git on PATH")
	}
	if _, err := exec.LookPath("go"); err != nil {
		t.Skip("no go on PATH; the hook finds gofmt through `go env GOROOT`")
	}
	hook, err := filepath.Abs(filepath.Join("..", "..", ".githooks", "pre-push"))
	if err != nil {
		t.Fatal(err)
	}

	repo := t.TempDir()
	git := func(args ...string) string {
		t.Helper()
		cmd := exec.Command("git", args...)
		cmd.Dir = repo
		cmd.Env = isolatedGitEnv(repo)
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("git %s: %v\n%s", strings.Join(args, " "), err, out)
		}
		return strings.TrimSpace(string(out))
	}
	commit := func(name, src, message string) string {
		t.Helper()
		if err := os.WriteFile(filepath.Join(repo, name), []byte(src), 0o644); err != nil {
			t.Fatal(err)
		}
		git("add", name)
		git("-c", "user.name=t", "-c", "user.email=t@example.com", "commit", "-qm", message)
		return git("rev-parse", "HEAD")
	}

	git("init", "-q", "-b", "main")
	dirty := commit("a.go", "package a\n\nfunc  F(){}\n", "unformatted")
	carrier := commit("b.go", "package a\n\nfunc G() {}\n", "adds b")
	tip := commit("a.go", "package a\n\nfunc F() {}\n", "formatted")
	zero := strings.Repeat("0", len(tip))

	// push runs the hook as git would for one ref moving from `from` (zero for
	// a new branch) to `to`.
	push := func(from, to string) (int, string) {
		t.Helper()
		cmd := exec.Command("sh", hook, "origin", "https://example.invalid/r.git")
		cmd.Dir = repo
		cmd.Env = isolatedGitEnv(repo)
		cmd.Stdin = strings.NewReader("refs/heads/main " + to + " refs/heads/main " + from + "\n")
		var stderr bytes.Buffer
		cmd.Stderr = &stderr
		err := cmd.Run()
		if exit, ok := err.(*exec.ExitError); ok {
			return exit.ExitCode(), stderr.String()
		}
		if err != nil {
			t.Fatalf("running the hook: %v\n%s", err, stderr.String())
		}
		return 0, stderr.String()
	}
	// section is the hook's report for one commit: from the start of its
	// header line ("pre-push: ... at <sha> (subject) ...") to the next line
	// that starts another report, or the end.
	section := func(out, sha string) string {
		at := strings.Index(out, " at "+sha+" ")
		if at < 0 {
			return ""
		}
		rest := out[strings.LastIndex(out[:at], "\n")+1:]
		if next := strings.Index(rest[1:], "\npre-push: "); next >= 0 {
			rest = rest[:next+1]
		}
		return rest
	}

	newBranch := func(setting string) {
		t.Helper()
		code, out := push(zero, tip)
		if code != 1 {
			t.Errorf("%s: the unformatted first commit should block the push; "+
				"the hook exited %d:\n%s", setting, code, out)
		}
		first := section(out, dirty)
		if !strings.Contains(first, "is not gofmt-clean") || !strings.Contains(first, "    a.go") {
			t.Errorf("%s: the refusal should list a.go under the commit that "+
				"introduced it (%s); it said:\n%s", setting, dirty, out)
		}
		second := section(out, carrier)
		if strings.Contains(second, "    a.go") {
			t.Errorf("%s: %s did not change a.go, so a.go should not be listed "+
				"under it again; it said:\n%s", setting, carrier, out)
		}
		if !strings.Contains(second, "+ 1 unchanged since") || !strings.Contains(second, "listed above") {
			t.Errorf("%s: %s carries a.go from a commit of this push, which "+
				"should be a count pointing at the listing above; it said:\n%s",
				setting, carrier, out)
		}
		if section(out, tip) != "" {
			t.Errorf("%s: the clean tip was reported:\n%s", setting, out)
		}
	}

	// The default is every commit.
	newBranch("with grmob.prepush.everyCommit unset")
	git("config", "grmob.prepush.everyCommit", "true")
	newBranch("with grmob.prepush.everyCommit=true")

	// Turned off, the hook is back to the tip. The refusal message's
	// `git -c grmob.prepush.everyCommit=false push` reaches the same `git
	// config --get` through GIT_CONFIG_PARAMETERS, which git exports to its
	// hooks; the repository setting stands in for it here.
	git("config", "grmob.prepush.everyCommit", "false")
	if code, out := push(zero, tip); code != 0 {
		t.Errorf("with grmob.prepush.everyCommit=false the hook checks the tip "+
			"only, and the tip is clean; it exited %d:\n%s", code, out)
	}
	git("config", "--unset", "grmob.prepush.everyCommit")

	// Commit 1 already on the remote: a.go's fault is not this push's.
	code, out := push(dirty, tip)
	if code != 0 {
		t.Errorf("with the unformatted commit already pushed, a later commit "+
			"that only carries a.go should not block while the tip is clean; "+
			"the hook exited %d:\n%s", code, out)
	}
	if got := section(out, carrier); !strings.Contains(got, "pre-push: note:") ||
		!strings.Contains(got, "from before this push") || !strings.Contains(got, "a.go") {
		t.Errorf("the carried a.go should still be noted under %s as from "+
			"before this push; the hook said:\n%s", carrier, out)
	}

	// The same carried file at the tip blocks: CI's gofmt runs on the tip.
	code, out = push(dirty, carrier)
	if code != 1 {
		t.Errorf("a tip that carries an unformatted file is what CI fails on, "+
			"wherever the file came from; the hook exited %d:\n%s", code, out)
	}
	if got := section(out, carrier); !strings.Contains(got, "is not gofmt-clean") || !strings.Contains(got, "a.go") {
		t.Errorf("the refusal should name a.go under the tip (%s); it said:\n%s", carrier, out)
	}
}

// isolatedGitEnv is the process environment with every git config source but
// the repository's own shut off, and HOME pointed inside the test's directory
// so nothing under the real one is read.
func isolatedGitEnv(home string) []string {
	env := []string{}
	for _, kv := range os.Environ() {
		if strings.HasPrefix(kv, "GIT_") || strings.HasPrefix(kv, "HOME=") {
			continue
		}
		env = append(env, kv)
	}
	return append(env,
		"HOME="+home,
		"GIT_CONFIG_GLOBAL="+os.DevNull,
		"GIT_CONFIG_NOSYSTEM=1",
	)
}
