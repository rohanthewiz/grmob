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
// files is that commit's tree, and that a tip-clean push with a dirty commit
// under it passes by default and fails when grmob.prepush.everyCommit is on.
// None of that is visible in the text. A temporary repository with two
// commits is the smallest thing that has the shape.
//
//	commit 1   a.go unformatted ("func  F(){}")
//	commit 2   a.go gofmt-clean                 ◄── the pushed tip
//
//	default           checks commit 2          exit 0
//	everyCommit=true  checks commit 1, then 2  exit 1, names commit 1
//
// The repository has no remote-tracking refs, so the hook's
// `--not --remotes` branch is the one taken, which is the new-branch case.
//
// Global and system git config are shut out (GIT_CONFIG_GLOBAL, NOSYSTEM), so
// a machine's own hooksPath, signing or template settings cannot change what
// is measured.
func TestThePrePushHookChecksEveryCommitOnlyWhenAsked(t *testing.T) {
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
	write := func(src string) {
		t.Helper()
		if err := os.WriteFile(filepath.Join(repo, "a.go"), []byte(src), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	git("init", "-q", "-b", "main")
	write("package a\n\nfunc  F(){}\n")
	git("add", "a.go")
	git("-c", "user.name=t", "-c", "user.email=t@example.com", "commit", "-qm", "unformatted")
	dirty := git("rev-parse", "HEAD")
	write("package a\n\nfunc F() {}\n")
	git("add", "a.go")
	git("-c", "user.name=t", "-c", "user.email=t@example.com", "commit", "-qm", "formatted")
	tip := git("rev-parse", "HEAD")

	zero := strings.Repeat("0", len(tip))
	stdin := "refs/heads/main " + tip + " refs/heads/main " + zero + "\n"
	run := func() (int, string) {
		t.Helper()
		cmd := exec.Command("sh", hook, "origin", "https://example.invalid/r.git")
		cmd.Dir = repo
		cmd.Env = isolatedGitEnv(repo)
		cmd.Stdin = strings.NewReader(stdin)
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

	if code, out := run(); code != 0 {
		t.Errorf("with grmob.prepush.everyCommit unset the hook checks the tip "+
			"only, and the tip is clean, so the push should pass. It exited "+
			"%d:\n%s", code, out)
	}

	git("config", "grmob.prepush.everyCommit", "true")
	code, out := run()
	if code != 1 {
		t.Errorf("with grmob.prepush.everyCommit=true the unformatted first "+
			"commit should block the push; the hook exited %d:\n%s", code, out)
	}
	if !strings.Contains(out, dirty) || !strings.Contains(out, "a.go") {
		t.Errorf("the refusal should name the commit (%s) and the file "+
			"(a.go) it is about; it said:\n%s", dirty, out)
	}
	if strings.Contains(out, "at "+tip+" (formatted) is not gofmt-clean") {
		t.Errorf("the clean tip was reported as unformatted:\n%s", out)
	}

	// Turned back off, the hook is back to the tip. The refusal message's
	// `git -c grmob.prepush.everyCommit=false push` reaches the same `git
	// config --get` through GIT_CONFIG_PARAMETERS, which git exports to its
	// hooks; the repository setting stands in for it here.
	git("config", "grmob.prepush.everyCommit", "false")
	if code, out := run(); code != 0 {
		t.Errorf("with grmob.prepush.everyCommit=false the hook is back to the "+
			"tip; it exited %d:\n%s", code, out)
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
