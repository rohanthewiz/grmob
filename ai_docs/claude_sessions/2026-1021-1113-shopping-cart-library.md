# A shopping cart in the library: package cart, UseCart and five comps widgets

Session: `251771fb-1569-4b39-955c-6e222bd83f19`
**Date:** 2026-10-21 11:13 · **Branch:** master (b8e9982, plus this commit)

## Goal

The user asked what a fully configurable shopping cart component would take,
then said to make it a library component. The answer was that most of the
visual pieces already existed (`StripeCheckout`, `FormatMoney`, `Stepper`,
`Badge`, `EmptyState`, `core.OnSwipe`, bytdb), and that the real work was a
cart model and state shared across screens, since GrMob has no provider
mechanism.

## Design, in three layers

```
  tap ──▶ Store.SetQuantity(id, n) ──▶ Dispatch(SetQuantity{…})
                                          │
                     Config.Reduce(cart, action) ──▶ new Cart value
                                          │
               ┌──────────────────────────┼─────────────────────┐
               ▼                          ▼                     ▼
        Persister.Save            subscribers notified    (old Cart value
        (write-through)           UseCart → RequestRender  left untouched)
  render ──▶ Store.Quote() ──▶ Config.Price(cart) ──▶ Quote{Subtotal, Adjustments, Total}
```

### `cart/` (new package, imports nothing from core)

- `cart.go`: package doc; `Item` (ID merges lines, `MaxQuantity`, string-only
  `Meta` so a cart survives JSON); `Cart{Items, Codes}` with `Count`,
  `Subtotal`, `Find`, `HasCode`.
- `reduce.go`: `Config` (Currency, MaxQuantity, Rules, AcceptCode, Persister,
  OnError); a closed `Action` set (`Add`, `SetQuantity`, `Remove`,
  `ApplyCode`, `RemoveCode`, `Clear`, `Replace`); `cfg.Reduce`, which never
  mutates its input (every action clones). `Add` fills a zero
  `Item.MaxQuantity` from `Config.MaxQuantity`, so a stored line carries its
  own cap and a widget can disable "+" without the Config.
- `price.go`: `Quote`, `Adjustment{Label, Amount, Code}`, `Rule{Code, Apply}`.
  Rules run in order on the running total (the order is the policy). A coded
  rule applies only while its code is on the cart. An empty cart is priced
  with no rules. `Rate` is basis points (`cart.Percent(8.25)`), `Rate.Of`
  rounds half away from zero in integers. Built-ins: `PercentOff`,
  `AmountOff` (both clamp at zero), `FlatShipping(label, amount, freeOver)`
  (a zero line when free, drawn "Free"), `Tax`.
- `store.go`: `Persister{Load, Save}`; `Store` with one mutex, saves under the
  lock (ordered), subscribers and `OnError` called after unlock (so they can
  read the store back). `Dispatch` skips save and notify when
  `reflect.DeepEqual` says nothing changed. Method values match the widget
  callbacks: `SetQuantity(id, n)`, `Remove(id)`, `ApplyCode(code) error`,
  `RemoveCode(code)`. A zero `Store` is usable.
- `cartdb/`: bytdb `Persister`. `Open(path)` or `New(engine)` (an app's own
  engine; `Close` leaves it open). One `carts(name text PRIMARY KEY, body
  text)` table, a cart per row as JSON, saved with `INSERT … ON CONFLICT (name)
  DO UPDATE`. Its own package so apps that don't persist this way never link
  bytdb.

### `hooks/cart.go`: `UseCart(ctx, store) cart.Quote`

One slot. Subscribes with `ctx.RequestRender`, moves the subscription if the
store pointer changes, releases on `ctx.OnClose` and re-arms after a Close
(UseLocationWhen's `closing` shape). The cart is read fresh from the store each
pass, never copied into the slot.

### comps widgets (new apidoc topic "Cart & checkout", `comps-commerce.md`)

- `CartLine`: thumbnail (`ImageWithMode` fill, hidden) or `Leading` slot; the
  item's text as one labelled phrase; Stepper bounded 1..cap (99 when
  uncapped, none when capped at 1) named "Quantity, <item>"; ghost error
  "Remove" named "Remove <item>". Controls appear only with their callbacks,
  so the zero line is read-only. `SwipeToRemove` is opt-in (leading swipe).
- `CartList`: keyed `line:<id>` / `sep:<id>`, EmptyState or `Empty` slot,
  `Line` override, `ReadOnly`, pass-through fields;
  `ConcernCartListDuplicateID`.
- `CartSummary`: a Card from the Quote alone, reusing StripeCheckout's
  `checkoutLine`; a discount's detail is its code; a zero adjustment reads
  `FreeLabel`. Plus `StripeCheckoutFromCart(q)`.
- `CartButton`: a Row with RoleButton, glyph and Badge beside it (not
  overlaid), named "Cart, 3 items"; "99+" cap; `ConcernCartButtonInert`.
  Padding is explicit sides, not shorthands, so a caller's single side
  clears (added to `TestACallerStylePropOutranksAWidgetsOwnInsets`).
- `PromoCodeField`: two hooks (draft, refusal); `OnApply func(string) error`;
  the refusal is a RoleAlert and clears on typing; applied codes are pills.
  TagInput's pill was extracted to `removablePill` (TagInput delegates; a nil
  remove draws no ✕).

`StripeCheckout` moved from the "actions" topic to "commerce". A package-level
test var `cart` in `comps/stripe_checkout_test.go` collided with the new import
and was renamed `checkoutCart`.

## Registrations and docs

- `internal/apidoc/packages.go`: `cart`, `cart/cartdb`, the comps topic;
  `mkdocs.yml` nav; `go run ./internal/apidoc/gen`.
- `docs/components.md`: CartList (with the package overview), CartLine,
  CartSummary, CartButton, PromoCodeField sections after StripeCheckout.
- `ai_docs/SKILL.md` and the installed `~/.claude/skills/grmob-native-mobile-go/SKILL.md`:
  a "Cart & checkout" row in the widget table.
- Lesson 4.43 "A shopping cart" (`examples/tutorial/chapter4.go`): a store via
  `hooks.UseMemo`, catalog Add buttons, CartButton, CartList with swipe and
  review toggles, PromoCodeField (SPRING10, plus a FREESHIP rule written in
  place), CartSummary with a Checkout footer. `TestShoppingCartLesson`.
- Counts: 86 lessons (README, docs/tutorial-interactive.md, wasm/index.html,
  shotclaims), chapter 4 = 43; tracked Go files 713 → 732 in
  `wasm/verify`; `docs/images/tutorial-contents.png` re-taken.

## Verification

- `go test ./...` green (44 packages ok), `gofmt -l .` empty, `go vet` clean on
  the touched packages; `go test -race ./cart/ ./hooks/` green.
- Throwaway shots of lesson 4.43 (scripts deleted afterwards): totals right
  ($70 − $7 SPRING10 = $63 ≥ $50, free shipping, tax $5.20, total $68.20), the
  refusal alert, pills and steppers all drawn as intended.
- Not run on Android, iOS or with a screen reader (N-123).

## Next

Closed: None. Declined: None. Raised: N-123, N-124, N-125.
Deferred: None. Promoted: None. Moved: None.
Updated: None. Full list: `ai_docs/todo/next-list.md`.
