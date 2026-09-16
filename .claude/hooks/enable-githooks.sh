#!/bin/sh
# Point this clone's git at .githooks, once a session, because git will not.
#
# # What this is for
#
# .githooks/pre-push refuses a push whose commits are not gofmt-clean — the
# check CI opens with, and the one that silently skipped vet and every test
# after it on seven master pushes in a row. It runs only when git is told to
# look in .githooks:
#
#	git config core.hooksPath .githooks
#
# That setting lives in .git/config, which is NOT tracked. So it is per clone,
# and git means it to be: a repository that could enable its own hooks would
# be running the author's shell script on everybody who cloned it, which is
# why .githooks is a convention and .git/hooks is the only directory git reads
# without being asked. The guard was therefore on in exactly one checkout —
# the one where somebody had typed the line — and off in every other, this one
# included when this file was written.
#
# The gap cannot be closed by git and cannot be closed by a note in a README,
# which is the same kind of thing that did not stop the pushes. What can close
# it is the one program that starts in this directory on this machine
# regularly: a SessionStart hook runs before the session does anything, costs
# one `git config --get` on the runs where there is nothing to do, and needs
# nobody to remember.
#
#	.claude/settings.json  ──SessionStart──▶  this script
#	                                             │
#	                            git config --get core.hooksPath
#	                                             │
#	          ┌──────────────────────┬───────────┴───────────┐
#	       unset                  .githooks              something else
#	          │                      │                       │
#	    set it, say so            silent             LEFT ALONE, say so
#
# # Why it never overwrites a value somebody else put there
#
# A hooksPath that is already set and is not .githooks is a deliberate act:
# a shared hooks directory for several repositories, or a tool like pre-commit
# that installs its own. Replacing it would disable that person's hooks to
# enable this repository's, in a process they did not run on purpose, and they
# would find out when the other check stopped firing. So a foreign value is
# reported into the session's context and otherwise untouched — the decision
# stays with the person, which is the same rule session-doc-check.sh follows
# for the same reason.
#
# # Why every path exits 0
#
# A SessionStart hook that fails is noise at the top of every session, and
# this one is a convenience: the push guard being off is a state the
# repository survived for its whole life so far. Nothing here is worth
# refusing to start a session over. Errors become a sentence or silence.
#
# # What holds it
#
# wasm/verify/hookconfig_test.go: the settings entry is in the file and names
# this script, and the script itself is run against throwaway repositories for
# all four branches of the diagram above. The push guard it turns on is held
# by wasm/verify/prepush_test.go, which is about what the hook DOES and has
# never had anything to say about whether it was reachable.
set -u

# The tracked hooks directory, as it is written into core.hooksPath. Relative
# on purpose: git resolves a relative hooksPath against the top of the working
# tree, so the value stays right if the checkout moves, and `git config --get`
# from a subdirectory still reports this exact string.
hooks_dir=".githooks"

# $CLAUDE_PROJECT_DIR is the project root Claude Code exports for hooks. The
# fallback is the working directory, which is where hooks are run from anyway;
# it matters only for a direct invocation, which is what the tests do.
repo=${CLAUDE_PROJECT_DIR:-.}

# A sentence into the session's context, and nothing at all when the argument
# is empty. SessionStart's additionalContext is the same channel
# session-doc-check.sh uses on PreToolUse.
#
# The text is passed through unquoted into JSON, so every message below is
# plain prose: no double quote, no backslash. That is a real constraint and it
# is kept rather than solved, because the alternative is a JSON encoder in
# POSIX sh for four fixed strings.
say() {
	[ -n "${1:-}" ] || return 0
	printf '{"hookSpecificOutput":{"hookEventName":"SessionStart","additionalContext":"%s"}}\n' "$1"
}

# A git that is not there, or a directory that is not a working tree. Both are
# possible — a hook fired in a scratch directory, a checkout exported without
# .git — and neither is this script's business to complain about.
command -v git >/dev/null 2>&1 || exit 0
git -C "$repo" rev-parse --is-inside-work-tree >/dev/null 2>&1 || exit 0

# The hooks themselves. If .githooks is missing there is nothing to point at,
# and pointing at it anyway would shadow .git/hooks with an empty directory —
# strictly worse than doing nothing.
[ -d "$repo/$hooks_dir" ] || exit 0

# The effective value, which is deliberately NOT --local: a global or system
# core.hooksPath is exactly the deliberate setting this must not tread on, and
# reading only the local scope would report it as unset and overwrite it.
current=$(git -C "$repo" config --get core.hooksPath 2>/dev/null) || current=""

if [ -n "$current" ]; then
	# Already ours. The common path, and silent: a session does not need to
	# be told that a setting is still what it was.
	[ "$current" = "$hooks_dir" ] && exit 0

	# An absolute path naming the same directory is also ours — somebody typed
	# the full path once. Compared after resolution so that a symlinked or
	# relative spelling of the same directory does not read as foreign.
	if [ -d "$current" ] && [ "$(cd "$current" 2>/dev/null && pwd -P)" = "$(cd "$repo/$hooks_dir" && pwd -P)" ]; then
		exit 0
	fi

	say "This clone has core.hooksPath set to $current, which is not the repository's .githooks, so .githooks/pre-push (the gofmt guard CI opens with) does not run on a push from here. It was left alone on purpose: a hooksPath somebody set is usually another tool's. Mention this in one line if the user pushes, and do not change the setting without asking."
	exit 0
fi

git -C "$repo" config --local core.hooksPath "$hooks_dir" 2>/dev/null || {
	say "Could not set core.hooksPath in this clone, so .githooks/pre-push (the gofmt guard CI opens with) will not run on a push from here. Run gofmt -l . before pushing."
	exit 0
}

# The execute bit, checked only on the path that just turned the directory on.
# git tracks the mode, so this is normally a property of the commit rather than
# of a checkout — but a working tree on a filesystem that carries no execute
# bit gets a hooksPath that is set and a hook that never runs, which looks
# exactly like a clean push.
if [ ! -x "$repo/$hooks_dir/pre-push" ]; then
	say "core.hooksPath now points at .githooks in this clone, but .githooks/pre-push is not executable here, so the gofmt guard still will not run. chmod +x it, or run gofmt -l . before pushing."
	exit 0
fi

say "The pre-push gofmt guard was off in this clone (core.hooksPath is per clone and is not tracked) and has been switched on: git config --local core.hooksPath .githooks. Pushes from here now run .githooks/pre-push, the same gofmt check CI opens with. Nothing else changed."
exit 0
