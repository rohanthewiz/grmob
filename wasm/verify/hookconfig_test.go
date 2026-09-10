package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"
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
				"never runs and nothing says so.",
				event, strings.Join(keysOf(hookEvents), ", "))
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
							"script.", event, i, j, k,
							strings.Join(keysOf(hookEntryKeys), ", "))
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
		t.Errorf("hooks.%s[%d].hooks[%d] has type %q; the types are: %s.",
			event, i, j, typ, strings.Join(keysOf(hookTypes), ", "))
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
