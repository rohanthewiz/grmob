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
# # The rule
#
#	nothing staged            silent. `git commit -a` and `git commit <path>`
#	                          stage inside the commit itself, so there is
#	                          nothing here to read and a guess would be noise
#	a doc is staged           silent. The commit carries its own doc
#	only session docs staged  silent. A doc-only commit IS the doc
#	otherwise                 say so
#
# stdin is the hook payload and is deliberately not read: the question is about
# the INDEX, which is a better answer than the command line — `git commit -m
# "..."` says nothing about what is in it.
set -u

docs="ai_docs/claude_sessions/"

# -z for the reason internal/themehistory asks for it: git C-quotes any path it
# cannot write literally, and a quoted path matches no prefix test. See
# wasm/verify/gitquoting_test.go, which holds every git listing in the Go
# sources to this rule and cannot see a shell script.
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
