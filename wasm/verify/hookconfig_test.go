package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"
)

// .claude/settings.json is a shape Claude Code will actually load.
//
// # The bug this is here for
//
// The session-doc hook was written to run before `git commit` and configured
// like this:
//
//	{"matcher": "Bash",
//	 "hooks": [{"type": "command",
//	            "if": "Bash(git commit:*)",
//	            "command": "…/session-doc-check.sh"}]}
//
// Both halves of that are wrong, and neither is visible from the file. A
// PreToolUse `matcher` is a regex over the TOOL NAME — `Bash`, `Write|Edit` —
// so `"Bash"` there selects EVERY Bash call; `Bash(git commit:*)` is
// permission-rule spelling, which belongs in `permissions.allow` and which no
// matcher has ever read. And `if` is not a field the hook schema has, so the
// narrowing everybody believed was happening was a key sitting in a JSON
// object being ignored.
//
// The file parsed. `jq -e` said so, which is why the previous session's
// verification passed: JSON validity is not schema validity, and the check
// that was run could only ever have found a syntax error.
//
// # Why a shape check and not a live one
//
// The thing worth proving is that Claude Code loads this and runs the script
// on the commits it was written for, and that is an end-to-end fact about
// another program: it needs a fresh process, a staged file and a commit to be
// about. That was done by hand once (see the session doc) and cannot be an
// arm here — this package has no business launching an agent.
//
// What a test CAN hold is the half that was actually wrong, which is the file:
// every key in it is a key the schema has, and every matcher is a tool name
// rather than a permission rule. A typo in a hook config fails no build, logs
// nothing where anybody looks, and produces exactly what a correct config
// produces on the runs where the hook had nothing to say — silence. That is
// the shape this repository keeps writing arms for.
func TestTheHookConfigUsesOnlyKeysTheSchemaHas(t *testing.T) {
	root := filepath.Join("..", "..")
	path := filepath.Join(root, ".claude", "settings.json")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading .claude/settings.json: %v\n\n"+
			"The file is tracked and holds the session-doc hook. If it was "+
			"removed on purpose, remove this check with it and say in the "+
			"commit message that the repository no longer warns about a "+
			"commit with no session doc — that is a decision, not a cleanup.",
			err)
	}
	// Decoded into `any` rather than a struct: a struct with the schema's
	// fields on it would SILENTLY DROP an unknown key, which is the exact
	// failure being checked for. DisallowUnknownFields on a struct would work
	// too and would stop at the first one; the map says all of them, and the
	// message can name the key.
	var settings map[string]any
	if err := json.Unmarshal(raw, &settings); err != nil {
		t.Fatalf(".claude/settings.json is not JSON: %v", err)
	}

	hooks, ok := settings["hooks"].(map[string]any)
	if !ok {
		t.Fatalf(".claude/settings.json has no `hooks` object (top-level keys: "+
			"%s). The session-doc hook lives there and nothing else in this "+
			"repository configures Claude Code.", strings.Join(keysOf(settings), ", "))
	}

	groups := 0
	entries := 0
	for event, v := range hooks {
		if !hookEvents[event] {
			t.Errorf("hooks.%s is not an event Claude Code fires. The events "+
				"are: %s.\n\nAn unknown event name is not an error anywhere — "+
				"the object is simply never matched — so the hook under it "+
				"never runs and nothing says so.\n\n%s",
				event, strings.Join(keysOf(hookEvents), ", "), hookSchemaNote())
			continue
		}
		list, ok := v.([]any)
		if !ok {
			t.Errorf("hooks.%s is %T and the schema wants an array of "+
				"{matcher, hooks} objects.", event, v)
			continue
		}
		for i, g := range list {
			group, ok := g.(map[string]any)
			if !ok {
				t.Errorf("hooks.%s[%d] is %T, not an object.", event, i, g)
				continue
			}
			groups++
			for k := range group {
				if !hookGroupKeys[k] {
					t.Errorf("hooks.%s[%d] has key %q, which the hook schema "+
						"does not have. Its keys are: %s.\n\n"+
						"A key the schema does not have is IGNORED — the file "+
						"still parses, Claude Code still loads it, and the "+
						"behaviour it was meant to produce simply does not "+
						"happen. `if` sat here for a session doing nothing at "+
						"all.", event, i, k, strings.Join(keysOf(hookGroupKeys), ", "))
				}
			}
			if m, present := group["matcher"]; present {
				matcher, ok := m.(string)
				if !ok {
					t.Errorf("hooks.%s[%d].matcher is %T, not a string.",
						event, i, m)
				} else {
					checkMatcher(t, event, i, matcher)
				}
			}
			inner, ok := group["hooks"].([]any)
			if !ok {
				t.Errorf("hooks.%s[%d] has no `hooks` array; nothing in it can "+
					"run.", event, i)
				continue
			}
			for j, h := range inner {
				hook, ok := h.(map[string]any)
				if !ok {
					t.Errorf("hooks.%s[%d].hooks[%d] is %T, not an object.",
						event, i, j, h)
					continue
				}
				entries++
				for k := range hook {
					if !hookEntryKeys[k] {
						t.Errorf("hooks.%s[%d].hooks[%d] has key %q, which the "+
							"hook schema does not have. Its keys are: %s.\n\n"+
							"This is where `if: \"Bash(git commit:*)\"` was, "+
							"and it is why session-doc-check.sh reads the "+
							"command out of its stdin payload now: the "+
							"matcher can only say \"a Bash call\", so a "+
							"narrower question has to be asked inside the "+
							"script.\n\n%s", event, i, j, k,
							strings.Join(keysOf(hookEntryKeys), ", "),
							hookSchemaNote())
					}
				}
				checkHookEntry(t, root, event, i, j, hook)
			}
		}
	}

	// The walk reaching anything at all. Every arm in this repository that
	// walks a structure says this, and for the same reason: a walk over an
	// empty list passes silently and reads as a clean result.
	if groups == 0 || entries == 0 {
		t.Fatalf("no hook entry was found in .claude/settings.json (%d group(s), "+
			"%d entr(ies)). This repository configures at least the session-doc "+
			"hook, so the traversal is not reaching it and this check is over "+
			"nothing.", groups, entries)
	}
	t.Logf("%d hook group(s), %d entr(ies) in .claude/settings.json; every key "+
		"is one the schema has and every matcher is a tool name.", groups, entries)
}

// checkMatcher holds one matcher to being a name and not a permission rule.
//
// A matcher is a regex, so `Write|Edit` and `Notebook.*` are both legitimate
// and a strict identifier test would reject them. What it is a regex OVER is
// the tool name, and that is the fact worth testing: the characters below are
// the ones that only ever appear in the OTHER spelling — `Bash(git commit:*)`,
// which is a permission rule and matches no tool that has ever existed.
//
// `*` is in the list even though it is a legal regex operator, because a
// matcher wanting "everything" is written as `.*` or left out entirely, and a
// bare `*` in a matcher is a glob somebody meant.
var matcherNotAName = regexp.MustCompile(`[()*:\[\]]`)

func checkMatcher(t *testing.T, event string, i int, matcher string) {
	t.Helper()
	if !matcherNotAName.MatchString(matcher) {
		return
	}
	t.Errorf("hooks.%s[%d].matcher is %q, which is permission-rule spelling "+
		"rather than a tool name.\n\n"+
		"A matcher is a regex over the TOOL NAME — `Bash`, `Write|Edit`, "+
		"`Notebook.*`. `Bash(git commit:*)` is the shape that goes in "+
		"`permissions.allow`, and as a matcher it names no tool, so the hook "+
		"under it never runs and nothing reports that.\n\n"+
		"To narrow a Bash hook to one command, match `Bash` and read "+
		"`.tool_input.command` out of the payload on stdin — see "+
		".claude/hooks/session-doc-check.sh.", event, i, matcher)
}

// checkHookEntry holds one hook entry: it says what type it is, and a command
// hook naming a script in this repository names one that is there and can run.
//
// The path check is worth having on its own. A `command` pointing at a file
// that does not exist fails at hook time, inside another process, on somebody
// else's machine — and a PreToolUse hook that fails is a hook that is not
// there, which for a warn-only check is indistinguishable from a green run.
func checkHookEntry(t *testing.T, root, event string, i, j int, hook map[string]any) {
	t.Helper()
	typ, _ := hook["type"].(string)
	if !hookTypes[typ] {
		t.Errorf("hooks.%s[%d].hooks[%d] has type %q; the types are: %s.\n\n%s",
			event, i, j, typ, strings.Join(keysOf(hookTypes), ", "),
			hookSchemaNote())
		return
	}
	if typ != "command" {
		return
	}
	cmd, _ := hook["command"].(string)
	if cmd == "" {
		t.Errorf("hooks.%s[%d].hooks[%d] is a command hook with no command.",
			event, i, j)
		return
	}
	// Only the repository's own scripts are resolved. A hook running `jq` or
	// `prettier` is naming something on PATH and this check has no opinion
	// about the machine it will run on; `$CLAUDE_PROJECT_DIR/...` is a claim
	// about THIS tree, which is a claim a test in it can settle.
	const marker = "$CLAUDE_PROJECT_DIR"
	idx := strings.Index(cmd, marker)
	if idx < 0 {
		return
	}
	rest := cmd[idx+len(marker):]
	// The command is a shell line, so the path is quoted and followed by
	// whatever arguments it takes. Cut at the first whitespace and drop the
	// quote that closed the substitution.
	rest = strings.TrimPrefix(rest, `"`)
	if k := strings.IndexAny(rest, " \t"); k >= 0 {
		rest = rest[:k]
	}
	rest = strings.Trim(rest, `"'`)
	rel := strings.TrimPrefix(rest, "/")
	if rel == "" {
		return
	}
	full := filepath.Join(root, filepath.FromSlash(rel))
	info, err := os.Stat(full)
	if err != nil {
		t.Errorf("hooks.%s[%d].hooks[%d] runs %s/%s, which is not in this "+
			"repository: %v.\n\nA hook whose command is missing fails inside "+
			"Claude Code, and a warn-only hook that fails looks exactly like "+
			"one with nothing to say.", event, i, j, marker, rel, err)
		return
	}
	if info.Mode()&0o111 == 0 {
		t.Errorf("hooks.%s[%d].hooks[%d] runs %s/%s and its mode is %v — not "+
			"executable. git tracks the execute bit, so this is a property of "+
			"the commit rather than of one checkout.",
			event, i, j, marker, rel, info.Mode())
	}
}

// The events Claude Code fires, from its own hooks reference.
//
// Listed rather than pattern-matched because the failure mode of a misspelled
// event is silence: the object under it is never matched and the hook simply
// never runs.
//
// This list and the three below it are a reading of ONE build of Claude Code,
// named in hookSchemaReadFrom. Anything added here is added with that record:
// a table updated against a newer release and left claiming the old version is
// a snapshot that has quietly stopped being one.
var hookEvents = map[string]bool{
	"PermissionRequest":  true,
	"PreToolUse":         true,
	"PostToolUse":        true,
	"PostToolUseFailure": true,
	"Notification":       true,
	"Stop":               true,
	"SubagentStop":       true,
	"PreCompact":         true,
	"PostCompact":        true,
	"UserPromptSubmit":   true,
	"SessionStart":       true,
	"SessionEnd":         true,
}

// The keys of a matcher group, and of one hook entry inside it.
//
// This is the whole point of the check, so the lists are the schema's and
// nothing else. `if` is conspicuously not among them.
var hookGroupKeys = map[string]bool{
	"matcher": true,
	"hooks":   true,
}

var hookEntryKeys = map[string]bool{
	"type":          true,
	"command":       true,
	"prompt":        true,
	"timeout":       true,
	"statusMessage": true,
}

var hookTypes = map[string]bool{
	"command": true,
	"prompt":  true,
	"agent":   true,
}

// Which Claude Code the four key tables above are a reading of.
//
// # Why a version and not just a list
//
// hookEvents, hookGroupKeys, hookEntryKeys and hookTypes were copied out of
// the hooks reference embedded in one build of Claude Code. That makes them a
// fact about THAT BUILD, sitting in a repository which will outlive it, and
// the direction they fail in is the bad one: a key a later release ADDS is a
// key this file has never heard of, so a CORRECT config fails a check whose
// whole purpose is to catch an incorrect one.
//
// Nothing here said which build. That is the same omission verifyTimingsTakenOn
// and themehistoryTimingsTakenOn exist to end for a wall clock, and it is the
// same omission for the same reason: a number with no machine attached and a
// list with no version attached are both readings presented as facts, and a
// reader holding a disagreement cannot tell a regression from a different
// computer, or a typo from a release.
//
// # What is done about it, since the direction cannot be fixed
//
// An unknown key still fails, and should: this repository's settings.json is
// written by hand, and `if` sat in it for a whole session doing nothing. What
// changes is that the failure now arrives with the two facts that tell the
// reader WHICH of the two things it is — the version these tables came from,
// and the version of the Claude Code standing on this machine. See
// hookSchemaNote, which every message in this file whose authority is a table
// ends with.
//
// The version difference itself is reported and never asserted, for the reason
// the timings records give about a machine: asserting it would fail on every
// computer whose Claude Code has moved on, which is every computer eventually.
//
// # Re-reading it
//
// The reference is inside the binary rather than on disk. The reliable way to
// re-read it is to ask a Claude Code session for its hooks reference and take
// four things out of the answer: the event names, the keys of a matcher group,
// the keys of one hook entry, and the hook types. Nothing else in that
// document is used here.
//
//	claude --version        the build the answer is coming from
var hookSchemaReadFrom = struct {
	// The build the tables were read out of.
	claudeVersion string
	// When, so a reader can weigh the drift without knowing this repository's
	// commit dates. Parsed by the arm below, so a placeholder is a failure.
	readOn string
	// Where in that build, for whoever re-reads it.
	source string
}{
	claudeVersion: "2.1.267",
	readOn:        "2026-09-10",
	source:        "the hooks reference embedded in the Claude Code binary",
}

// The version of the Claude Code on this machine, or "" if there is none to
// ask.
//
// # Why this runs another program, in a package that says it will not
//
// The header of this file draws a line at launching an agent: proving the hook
// actually fires needs a fresh session, a staged file and a commit to be
// about, and that is not a test's business. `claude --version` is on the other
// side of that line — it prints one line and exits, starts no session, reads
// no file of this repository's and changes nothing. It is worth the fork for
// the one sentence it produces, which is the difference between a failure a
// reader can act on and one they have to go and investigate.
//
// Computed once, because every failure message in this file wants it and a
// process per finding would be a fork per line. A missing binary is an ANSWER
// here rather than a failure — CI machines have no Claude Code and the config
// check is still worth running there — and the deadline is because this is
// another program and nothing here knows what it will do.
var installedClaude = sync.OnceValue(func() string {
	bin, err := exec.LookPath("claude")
	if err != nil {
		return ""
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	out, err := exec.CommandContext(ctx, bin, "--version").Output()
	if err != nil {
		return ""
	}
	// `2.1.267 (Claude Code)`. Only the first field is taken, so a change to
	// what follows it is not read as a change of version.
	fields := strings.Fields(string(out))
	if len(fields) == 0 {
		return ""
	}
	return fields[0]
})

// hookSchemaNote is the sentence that goes under any finding whose authority
// is one of the four tables in this file.
//
// Four shapes, and which one a reader gets is decided by ORDERING the two
// versions rather than by comparing them for equality.
//
//	no claude here     the comparison cannot be made at all
//	the same build     an unknown key cannot be a release that moved, so the
//	                   finding is about the file
//	a NEWER claude     this check's bad direction is live: the key may be one
//	                   a release after the tables added
//	an OLDER claude    the key cannot be a later addition — nothing later than
//	                   the tables is installed here
//
// The last two used to be one message saying "a different build", which is the
// check declining to do the comparison it had just made: `2.1.300` and `2.0.9`
// against a table read from `2.1.267` are opposite findings, and the reader was
// told to go and work out which by hand.
func hookSchemaNote() string {
	rec := hookSchemaReadFrom
	got := installedClaude()
	if got == "" {
		return fmt.Sprintf("The key tables in this check are a reading of %s, "+
			"taken from Claude Code %s on %s. There is no `claude` on this "+
			"machine to compare against, so this finding is either a mistake "+
			"in .claude/settings.json or a key a release after %s added. Ask a "+
			"Claude Code session for its hooks reference before changing the "+
			"file — and if the schema has grown, update hookSchemaReadFrom in "+
			"the same commit as the table.",
			rec.source, rec.claudeVersion, rec.readOn, rec.claudeVersion)
	}
	provenance := fmt.Sprintf("The key tables in this check were read from "+
		"Claude Code %s (%s, %s) and the `claude` on this machine is %s",
		rec.claudeVersion, rec.source, rec.readOn, got)

	order, ordered := compareVersions(got, rec.claudeVersion)
	switch {
	case !ordered:
		// One of the two is not a version this can order — a nightly spelling
		// the regexp below accepts and the comparator does not, or a `claude`
		// that has started printing something else first. Reported as the
		// unknown it is, rather than by guessing a direction.
		return provenance + ", and these two cannot be ordered against each " +
			"other, so nothing here can say whether this install is ahead of " +
			"the tables or behind them. Ask a Claude Code session for its " +
			"hooks reference; if the key is in it, add it to the table and " +
			"move hookSchemaReadFrom in the same commit."
	case order == 0:
		return fmt.Sprintf("This is not a version difference: %s. The schema "+
			"has not moved under this file, so the finding is about the file.",
			provenance)
	case order > 0:
		return provenance + fmt.Sprintf(" — NEWER than the tables. So a key "+
			"they have never heard of may be one a release after %s added "+
			"rather than a mistake, which is this check's bad direction: it "+
			"fails a CORRECT config. Ask a session for its hooks reference; if "+
			"the key is in it, add it to the table and move "+
			"hookSchemaReadFrom to %s in the same commit.",
			rec.claudeVersion, got)
	default:
		return provenance + fmt.Sprintf(" — OLDER than the tables. So the key "+
			"cannot be one a release added after they were read: nothing later "+
			"than %s is installed here, and the tables already describe %s. "+
			"Either it is a mistake in .claude/settings.json, or it is a key "+
			"THIS build has and %s dropped — which is worth knowing, because "+
			"the file has to load on the Claude Code people are running. Ask "+
			"this install's session for its hooks reference before changing "+
			"either.", got, rec.claudeVersion, rec.claudeVersion)
	}
}

// compareVersions orders two dotted versions the way a reader would: -1 if a
// is behind b, +1 if it is ahead, 0 if they are the same release. `ok` is
// false when either side is not something this can order at all.
//
// # Why this is here rather than golang.org/x/mod/semver
//
// The whole question is which of two opposite findings a reader is holding —
// a key a later release added, or a key an older install has — and that turns
// on ORDER.
//
// This paragraph used to say "twenty lines and no dependency, against a
// repository whose go.mod has one line in it", and both halves of that were
// wrong. go.mod has several requirements and one of them is golang.org/x/mod,
// held indirect by the `tool` block that pins gomobile — so the import moves a
// line in go.mod rather than adding a download. And semver answers all four
// rules below, which was checked and not assumed; see
// TestTheDottedVersionParsersAreTheOnesTheReasonCovers, where the four
// comparisons are written out.
//
// What is actually left of the argument is smaller and is about ONE caller:
// semver wants a leading `v` and `claude --version` prints `2.1.267`, so
// either way something here wraps it — and for a single caller a wrapper costs
// about what the parser does. The second orderer is where that stops being
// true, and the arm named above is what says so at the moment it arrives
// rather than several sessions later.
//
// # The rules, and what each one is for
//
//	numeric, field by field   `2.1.267` against `2.1.30` is 267 > 30, which
//	                          string comparison gets backwards
//	a missing field is 0      `2.1` and `2.1.0` are the same release
//	a pre-release tail is     semver's rule, and the one that matters here:
//	  BEHIND its release      `2.2.0-nightly` is not yet `2.2.0`
//	anything else is not      a field that is not digits, or an empty version.
//	  ordered                 The caller says so instead of picking a side
//
// A field that overflows an int is not a version anybody ships, and ParseInt
// saying so is the same answer as a field made of letters: not ordered.
func compareVersions(a, b string) (int, bool) {
	af, aPre, aOK := versionFields(a)
	bf, bPre, bOK := versionFields(b)
	if !aOK || !bOK {
		return 0, false
	}
	for i := 0; i < len(af) || i < len(bf); i++ {
		x, y := 0, 0
		if i < len(af) {
			x = af[i]
		}
		if i < len(bf) {
			y = bf[i]
		}
		switch {
		case x < y:
			return -1, true
		case x > y:
			return 1, true
		}
	}
	// The numbers are equal, so only the tail can separate them. A release is
	// ahead of any pre-release of itself; two different pre-release tails of
	// one release are left equal rather than ordered by their text, which is
	// where semver's own rules get intricate and where nothing in this
	// repository has a question.
	switch {
	case aPre && !bPre:
		return -1, true
	case !aPre && bPre:
		return 1, true
	}
	return 0, true
}

// versionFields is a version's numeric fields, whether it carries a
// pre-release tail, and whether it is orderable at all.
func versionFields(v string) (fields []int, pre bool, ok bool) {
	// The tail is cut before anything is parsed: `-` and `+` are the two
	// characters versionish allows after the numbers, and neither belongs to
	// the field it follows.
	if i := strings.IndexAny(v, "-+"); i >= 0 {
		pre = true
		v = v[:i]
	}
	if v == "" {
		return nil, false, false
	}
	for _, f := range strings.Split(v, ".") {
		n, err := strconv.Atoi(f)
		if err != nil || n < 0 {
			return nil, false, false
		}
		fields = append(fields, n)
	}
	return fields, pre, true
}

// The key tables say which build of Claude Code they are a reading of, and
// this run says whether it is standing on that build.
//
// # What is asserted and what is only reported
//
// Asserted: the record is filled in and its two machine-readable fields are
// the shapes they claim to be — a version made of digits and dots, and a date
// that parses. A placeholder in either is a record that attributes the tables
// to nothing, which is the state this record exists to end.
//
// Reported: everything about the machine. A newer Claude Code here is the
// ordinary case and not a finding, and an arm over it would fail on every
// computer whose install has moved on. What is worth having is the SENTENCE,
// and the place it is worth having it is under a failure — which is
// hookSchemaNote's job, not this test's. This one exists so the fact is on the
// record on a green run too, where somebody reading -v output can see how old
// the snapshot has become.
func TestTheHookSchemaTablesSayWhichClaudeCodeTheyCameFrom(t *testing.T) {
	rec := hookSchemaReadFrom
	if rec.claudeVersion == "" || rec.readOn == "" || rec.source == "" {
		t.Fatalf("hookSchemaReadFrom has an empty field (%+v).\n\n"+
			"The four key tables in this file are a reading of one build of "+
			"Claude Code, and a record with a hole in it attributes them to "+
			"nothing — which is the state where an unknown key cannot be told "+
			"apart from a release that added one.", rec)
	}
	// A version, and not a word somebody left behind. `latest`, `unknown` and
	// an empty-looking string all pass a non-empty test and all attribute the
	// tables to nothing.
	if !versionish.MatchString(rec.claudeVersion) {
		t.Errorf("hookSchemaReadFrom.claudeVersion is %q, which is not a "+
			"version. It is the build the key tables were read out of, and it "+
			"is compared against `claude --version` — a word here makes both "+
			"the comparison and the failure messages meaningless.",
			rec.claudeVersion)
	}
	if _, err := time.Parse("2006-01-02", rec.readOn); err != nil {
		t.Errorf("hookSchemaReadFrom.readOn is %q, which is not a date: %v.\n\n"+
			"It is how far a reader can tell the snapshot has drifted without "+
			"going through this repository's commit dates.", rec.readOn, err)
	}
	// The tables reaching anything. An emptied table makes every key in the
	// settings file unknown, so the check would fail loudly — but it would
	// fail naming the file, and the cause would be here.
	if len(hookEvents) == 0 || len(hookGroupKeys) == 0 ||
		len(hookEntryKeys) == 0 || len(hookTypes) == 0 {
		t.Fatalf("a key table is empty (%d event(s), %d group key(s), %d entry "+
			"key(s), %d type(s)). Every key in .claude/settings.json would be "+
			"reported as one the schema does not have, naming the file for a "+
			"fault that is in this one.", len(hookEvents), len(hookGroupKeys),
			len(hookEntryKeys), len(hookTypes))
	}

	got := installedClaude()
	if got == "" {
		t.Logf("the key tables in this file are a reading of %s, taken from "+
			"Claude Code %s on %s. There is no `claude` on this machine, so "+
			"nothing here can say how far that snapshot has drifted.",
			rec.source, rec.claudeVersion, rec.readOn)
		return
	}
	order, ordered := compareVersions(got, rec.claudeVersion)
	switch {
	case !ordered:
		// Not an assertion for the same reason the drift is not one, and
		// reported because it is the one state where hookSchemaNote cannot
		// name a direction under a failure. A `claude` printing something the
		// comparator cannot read is worth seeing on a green run rather than
		// discovering inside a finding.
		t.Logf("the key tables in this file were read from Claude Code %s (%s, "+
			"%s) and this machine's `claude` answers %q, which cannot be "+
			"ordered against it. A finding from this file will say so instead "+
			"of naming a direction; see compareVersions for what it can read.",
			rec.claudeVersion, rec.source, rec.readOn, got)
	case order == 0:
		t.Logf("this run is on the build the key tables came from: Claude Code "+
			"%s, read from %s on %s. An unknown key found by this file today "+
			"is a fault in .claude/settings.json and cannot be a release that "+
			"moved.", got, rec.source, rec.readOn)
	case order > 0:
		t.Logf("the key tables in this file were read from Claude Code %s (%s, "+
			"%s) and this machine runs %s, which is NEWER.\n\n"+
			"That is not a failure and this check does not make it one — a "+
			"newer install is the ordinary case, and an arm over it would fail "+
			"on every computer whose Claude Code has moved on. It is recorded "+
			"because this check's bad direction is live in exactly this "+
			"state: a key a release after %s added would be reported as a "+
			"config that is wrong.",
			rec.claudeVersion, rec.source, rec.readOn, got, rec.claudeVersion)
	default:
		// The other direction, and it is the interesting one to see on a
		// green run: the tables were read from a build nobody here is
		// running, so this run is not exercising the schema it checks
		// against.
		t.Logf("the key tables in this file were read from Claude Code %s (%s, "+
			"%s) and this machine runs %s, which is OLDER.\n\n"+
			"Also not a failure. It does mean this checkout's settings.json "+
			"is being held to a schema newer than the Claude Code that will "+
			"load it, so a key %s dropped and %s still has would be reported "+
			"here — which is a finding about the file, in the opposite "+
			"direction from the one above.",
			rec.claudeVersion, rec.source, rec.readOn, got, rec.claudeVersion,
			got)
	}
}

// The comparator behind those four branches, over the shapes a version takes
// and the ones it does not.
//
// # Why this is a table and not two calls
//
// The thing being fixed was a comparison that could not say WHICH WAY, and the
// failure it produced was a correct-looking sentence: "a different build" is
// true of `2.1.300` and of `2.0.9` and tells a reader nothing they can act on.
// A comparator with the same property — right about equality, wrong about
// order — passes any test that only asks whether two versions differ, so the
// cases here are chosen to be the ones string comparison gets backwards and
// the ones a naive split panics on.
func TestOrderingTwoClaudeVersions(t *testing.T) {
	cases := []struct {
		a, b string
		// -1, 0, +1, or 2 for "cannot be ordered", which is a value no
		// comparison returns.
		want int
	}{
		// The pair the whole item is about: string comparison puts `2.1.30`
		// ahead of `2.1.267` because `3` > `2`, and that is a reader sent to
		// look for a key a release added when the release is behind them.
		{"2.1.267", "2.1.30", 1},
		{"2.1.30", "2.1.267", -1},
		{"2.1.267", "2.1.267", 0},
		{"2.1.300", "2.1.267", 1},
		{"2.0.9", "2.1.267", -1},
		// Field-by-field, not lexical, at every position.
		{"10.0.0", "9.9.9", 1},
		{"2.2", "2.10", -1},
		// A missing field is a zero, so these are the same release.
		{"2.1", "2.1.0", 0},
		{"2.1.0.0", "2.1", 0},
		{"3", "3.0.0", 0},
		// A pre-release is behind its release and ahead of the one before it.
		{"2.2.0-nightly", "2.2.0", -1},
		{"2.2.0", "2.2.0-nightly", 1},
		{"2.2.0-nightly", "2.1.999", 1},
		{"2.2.0+build.7", "2.2.0", -1},
		// Two tails of one release are left equal rather than ordered by
		// their text; see compareVersions.
		{"2.2.0-a", "2.2.0-b", 0},
		// And the shapes that are not versions. Each one would otherwise be
		// answered with a confident direction: "" splits to one empty field,
		// `latest` is the placeholder the record's own arm refuses, and a
		// leading dot is what a bad split produces.
		{"", "2.1.267", 2},
		{"2.1.267", "", 2},
		{"latest", "2.1.267", 2},
		{"2.1.x", "2.1.267", 2},
		{".2.1", "2.1", 2},
		{"2..1", "2.0.1", 2},
		{"-1.0", "1.0", 2},
	}
	for _, c := range cases {
		got, ordered := compareVersions(c.a, c.b)
		if !ordered {
			got = 2
		}
		if got != c.want {
			t.Errorf("compareVersions(%q, %q) = %d, want %d.\n\n"+
				"This decides which of two opposite sentences a failing "+
				"config check prints: a key a release ADDED (this install is "+
				"ahead of the tables) or a key an older build has that the "+
				"tables' build dropped. A wrong direction here is a reader "+
				"sent to ask a session about a key that cannot be new.",
				c.a, c.b, got, c.want)
		}
	}
	// The comparison this repository actually makes, on whatever machine is
	// running. Reported, never asserted — see the record's own arm.
	if got := installedClaude(); got != "" {
		order, ordered := compareVersions(got, hookSchemaReadFrom.claudeVersion)
		t.Logf("this machine's claude is %s against the tables' %s: order=%d "+
			"ordered=%v.", got, hookSchemaReadFrom.claudeVersion, order, ordered)
	}
}

// A version and not a word: digits and dots, with an optional pre-release tail
// that a nightly or a release candidate would carry.
var versionish = regexp.MustCompile(`^[0-9]+(\.[0-9]+)*([-+][0-9A-Za-z.-]+)?$`)

// keysOf is a map's keys, sorted, so a failure message reads the same on every
// run and can be diffed.
func keysOf[V any](m map[string]V) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}
