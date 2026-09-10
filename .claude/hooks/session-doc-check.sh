#!/bin/sh
# Notice a commit that owes a session doc, and say so without stopping it.
#
# # What this is for
#
# A session that does real work and commits without a session doc loses its
# whole `## Next` list at once, and it does not read as a gap afterwards: the
# previous doc's `## Next` section is still there, still looks live, and every
# item in it is done. The follow-ups the session generated exist nowhere but
# the commit message. That happened here — see 8d14384, five items of real
# work, a long commit message and no doc — and the next session had to
# re-derive the list by reading the code.
#
# .claude/commands/commit.md asks the model to check for this. That is a habit,
# and a note in a file is exactly what did not stop 8d14384. This is the same
# check as a mechanism.
#
# # Why it warns and never blocks
#
# A hook that refuses a commit is a hook people disable, and then the check is
# gone for the cases it was written for as well as the ones it got wrong. There
# are plenty it would get wrong: a typo fix, a rebase, a merge, a file the
# person wrote themselves. None of those owes anything.
#
# So it emits `additionalContext` — a sentence into the model's context, before
# the commit runs — and exits 0 on every path, including the ones where git
# itself failed. The decision stays where it was.
#
# # Which tool calls reach here, and why the payload IS read
#
# It was written believing `.claude/settings.json` could name the commit:
# `"matcher": "Bash"` with an `"if": "Bash(git commit:*)"` beside it, and the
# comment here said stdin was "deliberately not read". Both halves were wrong
# in the same way. A PreToolUse `matcher` is a regex over the TOOL NAME —
# `Bash`, `Write|Edit` — and `Bash(git commit:*)` is permission-rule spelling,
# which no matcher has ever accepted; `if` is not a field the hook schema has
# at all. So the entry either selected every Bash call or none, and this
# script ran on neither the occasions it was written for nor a knowable subset
# of them. See wasm/verify/hookconfig_test.go, which now holds the settings
# file to the schema so that a second guess about it fails a test rather than
# going unnoticed for a session.
#
# What is left is that the matcher can only say "a Bash call", so the command
# filter has to be here. stdin is the PreToolUse payload and `.tool_input.
# command` is the command about to run.
#
# The filter is deliberately loose in one direction: `git` as a word, anything
# that is not a command separator, then `commit` as a word. That catches `git
# commit -m`, `git -C dir commit` and `cd x && git commit`, and it also catches
# `git log --grep commit`, which is not a commit at all. The cost of that false
# positive is one `git diff --cached` and, if non-doc files happen to be
# staged, a sentence nobody needed — the same bounded looseness the NUL
# handling below already accepts. The cost of a false NEGATIVE is the whole
# check, silently, which is what this file exists to stop.
#
# # What decides what to say, which is not the command
#
# The INDEX. `git commit -m "..."` says nothing about what is in it, and `-a`
# and `<path>` stage inside the commit itself. So the command line answers
# "is this a commit", and `git diff --cached` answers "does it owe a doc".
#
# # The rule
#
#	not a git commit          silent. Some other Bash call
#	nothing staged            silent. `git commit -a` and `git commit <path>`
#	                          stage inside the commit itself, so there is
#	                          nothing here to read and a guess would be noise
#	a doc is staged           silent. The commit carries its own doc
#	only session docs staged  silent. A doc-only commit IS the doc
#	otherwise                 say so
set -u

docs="ai_docs/claude_sessions/"

# The payload, read whole. Small — one tool call's input — and read before
# anything else so that a git that hangs cannot leave the caller writing into a
# pipe nobody is draining.
payload=$(cat 2>/dev/null) || payload=""

# jq is what the hook documentation's own examples use and it is present on
# this machine, but a hook that vanishes when a tool is missing is a check that
# vanishes silently, which is the failure mode this whole file is about. So a
# jq that is absent or that cannot parse falls back to scanning the RAW
# payload: strictly looser — a `git commit` quoted inside some other command's
# arguments would match — and loose here only ever costs a sentence.
cmd=$(printf '%s' "$payload" | jq -r '.tool_input.command // empty' 2>/dev/null) || cmd=""
[ -n "$cmd" ] || cmd=$payload

printf '%s\n' "$cmd" | grep -qE '(^|[^A-Za-z0-9_./-])git[^;&|]*[[:space:]]commit([^A-Za-z0-9_-]|$)' || exit 0

# -z for the reason internal/themehistory asks for it: git C-quotes any path it
# cannot write literally, and a quoted path matches no prefix test. See
# wasm/verify/gitquoting_test.go, which holds every git listing in the Go
# sources to this rule — and, since TestEveryGitListingInAScriptAsksForZ was
# added beside it, this file too.
#
# The NUL records are then turned into lines, which gives back exactly one of
# the two problems -z solves: a path containing a real newline splits into two
# records. POSIX sh has no NUL-safe read, and the consequence here is bounded —
# each half is still classified, so the worst case is a WARNING nobody needed
# on a commit staging a file with a newline in its name. A hook that only ever
# prints a sentence can afford that; the walker in internal/themehistory, whose
# same mistake dropped a file from a table in silence, could not.
staged=$(git diff --cached --name-only -z 2>/dev/null | tr '\0' '\n') || exit 0
[ -n "$staged" ] || exit 0

# Split on newlines only, so a path holding a space stays one record.
doc=0
other=0
oldIFS=$IFS
IFS='
'
for path in $staged; do
	case "$path" in
	"$docs"*) doc=1 ;;
	*) other=1 ;;
	esac
done
IFS=$oldIFS

[ "$other" -eq 1 ] || exit 0
[ "$doc" -eq 0 ] || exit 0

cat <<'EOF'
{"hookSpecificOutput":{"hookEventName":"PreToolUse","additionalContext":"This commit stages files outside ai_docs/claude_sessions/ and includes no session doc. If this session did real work, its `## Next` list exists only in your context right now and is gone after the commit — 8d14384 is what that looks like afterwards. Mention /sess-wrap (doc, commit, push) or /sess-save (doc only) in one line and ask whether to run one first. If the commit owes nothing — a typo fix, a rebase, a file the user wrote themselves — proceed without comment."}}
EOF
exit 0
