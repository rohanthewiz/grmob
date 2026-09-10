Please commit the changes.

Before committing, check whether this session owes a session doc.

A session that does real work and commits without one loses its whole `## Next`
list at once, and it does not read as a gap afterwards: the previous doc's
`## Next` section is still there, still looks live, and every item in it is
done. The follow-ups the session actually generated exist nowhere but the
commit message, and the next session has to re-derive them by reading the code.
That has happened here — see commit `8d14384`, which did five items of real
work, wrote a long commit message and saved no doc.

`/sess-wrap` makes the doc and the commit one action; this is the check for
when `/commit` is reached on its own.

So: if the changes being committed touch anything outside
`ai_docs/claude_sessions/`, and no session doc for this session is going in
alongside them, say so in one line before committing — name `/sess-wrap` (doc,
commit, push) or `/sess-save` (doc only) and ask whether to run one first.

Do not save a doc unasked, and do not block the commit if the answer is no: a
commit with no session behind it — a typo fix, a rebase, a file the user wrote
themselves — owes nothing.
