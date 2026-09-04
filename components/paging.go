package components

import "github.com/rohanthewiz/grmob/core"

// Pagination is the numbered-page footer: "‹ Prev  Page 2 of 7  Next ›".
//
// It serves two paging models with one struct, told apart by PageCount:
//
//   - PageCount > 0: the caller owns the pages. Whatever rows it hands the
//     collection are the current page, and OnChange asks it to fetch another.
//     This is the server-side shape.
//   - PageCount == 0 and PageSize > 0: the collection owns the pages. It
//     slices the full row set itself and derives PageCount from len(rows).
//     This is the client-side shape; DataTable implements it, GroupedList
//     does not (a grouped feed pages by "Load more", not by number).
//   - PageCount == 0 and PageSize == 0: open-ended. The label shows only the
//     current page and Next is never disabled, for a caller that learns
//     there is no next page only by asking.
//
// Selection is controlled, as everywhere in this package: Page is read from
// the caller's state and OnChange writes it back.
type Pagination struct {
	// Page is the 0-based current page.
	Page int
	// PageSize is rows per page; only read by a collection doing its own
	// slicing (see the type comment).
	PageSize int
	// PageCount is the total number of pages, 0 when unknown.
	PageCount int
	OnChange  func(page int)

	// PrevLabel and NextLabel override the button text. Empty takes the
	// defaults below.
	PrevLabel string
	NextLabel string

	// Style is applied to the footer row after its defaults.
	Style []core.StyleProp
}

func (p Pagination) Render(ctx *core.Context) *core.Node {
	t := ctx.Theme()

	prev := p.PrevLabel
	if prev == "" {
		prev = "‹ Prev"
	}
	next := p.NextLabel
	if next == "" {
		next = "Next ›"
	}

	// A page is 0-based in code and 1-based to people.
	label := "Page " + itoa(p.Page+1)
	if p.PageCount > 0 {
		label += " of " + itoa(p.PageCount)
	}

	atStart := p.Page <= 0
	atEnd := p.PageCount > 0 && p.Page >= p.PageCount-1

	items := make([]core.PropsAndChildren, 0, len(p.Style)+7)
	items = append(items,
		core.Justify(core.JustifyBetween),
		core.AlignItemsProp(core.AlignItemsCenter),
		core.Gap(float64(t.Spacing.SM)),
	)
	for _, sp := range p.Style {
		items = append(items, sp)
	}
	items = append(items,
		p.button(prev, p.Page-1, atStart, "Previous page"),
		core.Text(label, core.UseStyle(t.Typography.Caption)),
		p.button(next, p.Page+1, atEnd, "Next page"),
	)
	return core.Row(items...).Render(ctx)
}

// button is one of the two steppers. Disabled at the edge rather than hidden,
// so the footer keeps its shape from the first page to the last.
func (p Pagination) button(label string, target int, disabled bool, a11y string) core.View {
	b := Button{
		Label:              label,
		Emphasis:           EmphasisGhost,
		Disabled:           disabled,
		AccessibilityLabel: a11y,
	}
	if p.OnChange != nil {
		b.OnTap = func() { p.OnChange(target) }
	}
	return b
}

// LoadMore is the append-style tail of an offset-paged list: the "Load more"
// button, the "Loading…" note while a page is in flight, and the retry strip
// when a page failed. It exists because every paged screen in an app
// hand-rolls exactly this state machine at the bottom of its list.
//
//	HasMore  Loading  Err   renders
//	false    false    nil   nothing (the list is complete)
//	true     false    nil   [Load more]
//	*        true     *     Loading…
//	*        false    set   message  [Retry]
//
// Err wins over HasMore because a failed fetch says nothing about whether
// more rows exist; Loading wins over Err because a retry in flight has
// superseded the failure it is retrying.
type LoadMore struct {
	HasMore bool
	Loading bool
	Err     error

	// OnLoadMore fetches the next page. OnRetry re-runs a failed fetch; when
	// nil, Retry falls back to OnLoadMore, which is the right answer for a
	// pager whose load call is idempotent about the offset.
	OnLoadMore func()
	OnRetry    func()

	// Label, LoadingLabel and RetryLabel override the copy. ErrorText
	// replaces err.Error() — for an app that maps transport errors to
	// something a person should read.
	Label        string
	LoadingLabel string
	RetryLabel   string
	ErrorText    string

	// Style is applied to the tail's container after its defaults.
	Style []core.StyleProp
}

func (l LoadMore) Render(ctx *core.Context) *core.Node {
	t := ctx.Theme()

	items := make([]core.PropsAndChildren, 0, len(l.Style)+6)
	items = append(items,
		core.Justify(core.JustifyCenter),
		core.AlignItemsProp(core.AlignItemsCenter),
		core.Gap(float64(t.Spacing.SM)),
	)
	for _, sp := range l.Style {
		items = append(items, sp)
	}

	switch {
	case l.Loading:
		label := l.LoadingLabel
		if label == "" {
			label = "Loading…"
		}
		items = append(items, core.Text(label, core.UseStyle(t.Typography.Caption)))

	case l.Err != nil:
		msg := l.ErrorText
		if msg == "" {
			msg = l.Err.Error()
		}
		retry := l.OnRetry
		if retry == nil {
			retry = l.OnLoadMore
		}
		label := l.RetryLabel
		if label == "" {
			label = "Retry"
		}
		items = append(items,
			core.Box(core.FlexGrow(1), core.Text(msg,
				core.UseStyle(t.Typography.Caption),
				core.TextColor(t.Colors.Error))),
			Button{Label: label, Emphasis: EmphasisGhost, OnTap: retry},
		)

	case l.HasMore:
		label := l.Label
		if label == "" {
			label = "Load more"
		}
		items = append(items, Button{Label: label, Emphasis: EmphasisGhost, OnTap: l.OnLoadMore})

	default:
		// Complete: no tail at all. A Fragment costs the list nothing and
		// keeps the footer's position in the child list stable for the
		// reconciler, which is why this is not a nil return.
		return core.Fragment().Render(ctx)
	}

	return core.Row(items...).Render(ctx)
}
