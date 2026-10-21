# PromoCodeField: an async apply, and refusals as sentences (N-124, N-125)

Session: `251771fb-1569-4b39-955c-6e222bd83f19`
**Date:** 2026-10-21 11:22 · **Branch:** master (089e560 → a9ad530 → 1878cd2, plus this commit)

## Goal

`/work-next-list N-124 N-125`: the two Open items raised earlier in this
session by the shopping-cart work (`2026-1021-1113-shopping-cart-library`).
Both touch `comps/promo_code_field.go`, so they ran in ID order, one commit
each. Neither needed the user.

## N-124: `OnApplyAsync` (a9ad530)

Premise held: `OnApply` was synchronous, with no pending state, so a
server-checked code would freeze the event path.

- New field `OnApplyAsync func(code string, done func(error))`, which wins
  over `OnApply` (StripeCheckout's OnPay-over-CheckoutURL precedent), and
  `PendingLabel` (default "Checking…").
- A third hook, `pending`. While it is set, the input and Apply are disabled
  and Apply reads `PendingLabel`. A second submit is ignored.
- `done` is wrapped in a `sync.Once`, so only its first call counts (a
  caller answering from both a timeout and a response). It is safe from any
  goroutine because `State.Set` is.
- I first wrote an attempt counter to drop superseded answers, then removed
  it: the input is locked while an attempt is out, so there is never a second
  attempt to supersede the first. That saved a hook.
- Documented recipe: on server success, `store.Dispatch(cart.ApplyCode{…})`
  rather than `Store.ApplyCode`, which would ask `AcceptCode` again. The
  pricing rule must still be in `Config.Rules`.
- Test `TestPromoCodeFieldAsyncApplyIsPendingUntilDone` (run with -race): the
  async path wins over sync, the trimmed code is passed, pending locks both
  controls, a refusal arrives from another goroutine, an acceptance clears
  the draft, and a late second `done` is ignored.
  `TestPromoCodeFieldAsyncAloneIsNotInert` checks the concern only fires
  when both callbacks are nil.

## N-125: refusal text (1878cd2)

Premise held: the field drew `err.Error()` verbatim, so the lesson showed
"that code isn't valid".

- Default `ErrorText` is `sentenceCase`: upper-case the first rune when it is
  lowercase, change nothing else. `ErrorText func(error) string` replaces it.
- `cart.ErrUnknownCode` keeps its lowercase text (Go convention, ST1005), so
  it composes into longer messages and `errors.Is` works. Its doc now says
  where it gets sentence-cased.
- Decision recorded in the item and in `docs/components.md`: the cart's other
  English defaults stay per-field overrides, the way every comps widget
  localises, with no cart-wide string bundle.
- Tests: a table (lowercase, already capitalised, `é` → `É`, a leading digit)
  plus the override. The existing refusal assertions in comps and
  `TestShoppingCartLesson` now expect "That code isn't valid".

## Verification

`go test ./...` green after each item, `gofmt -l .` empty, `go vet ./comps/`
clean, `docs/api` regenerated (`comps-commerce.md`, `cart.md`). There was no
visual change beyond the button label and the capitalised alert, which the
tests assert, so I took no screenshots.

## Next

Closed: N-124, N-125. Declined: None. Raised: None.
Deferred: None. Promoted: None. Moved: None.
Updated: None. Full list: `ai_docs/todo/next-list.md`.
