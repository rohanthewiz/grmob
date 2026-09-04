package components

import (
	"errors"
	"testing"

	"github.com/rohanthewiz/grmob/core"
)

// buttons returns the Button nodes under n in document order.
func buttons(n *core.Node) []*core.Node {
	var out []*core.Node
	var walk func(*core.Node)
	walk = func(n *core.Node) {
		if n.Type == "Button" {
			out = append(out, n)
		}
		for _, c := range n.Children {
			walk(c)
		}
	}
	walk(n)
	return out
}

func TestPaginationLabelsAndEdges(t *testing.T) {
	ctx := core.NewContext()
	ctx.BeginRenderPass()
	got := -1
	n := Pagination{Page: 0, PageCount: 3, OnChange: func(p int) { got = p }}.Render(ctx)

	if findText(n, "Page 1 of 3") == nil {
		t.Error("label should be 1-based with the count")
	}
	bs := buttons(n)
	if len(bs) != 2 {
		t.Fatalf("want prev and next buttons, got %d", len(bs))
	}
	if !bs[0].Style.Disabled {
		t.Error("Prev should be disabled on the first page")
	}
	if bs[1].Style.Disabled {
		t.Error("Next should be enabled before the last page")
	}
	ctx.TriggerCallback(bs[1].Props["onClick"].(string))
	if got != 1 {
		t.Errorf("Next should request page 1, got %d", got)
	}

	ctx.BeginRenderPass()
	n = Pagination{Page: 2, PageCount: 3, OnChange: func(p int) { got = p }}.Render(ctx)
	bs = buttons(n)
	if !bs[1].Style.Disabled || bs[0].Style.Disabled {
		t.Error("on the last page only Next is disabled")
	}
	ctx.TriggerCallback(bs[0].Props["onClick"].(string))
	if got != 1 {
		t.Errorf("Prev should request page 1, got %d", got)
	}
}

func TestPaginationOpenEndedNeverDisablesNext(t *testing.T) {
	ctx := core.NewContext()
	ctx.BeginRenderPass()
	n := Pagination{Page: 4, OnChange: func(int) {}}.Render(ctx)
	if findText(n, "Page 5") == nil || findText(n, "Page 5 of 0") != nil {
		t.Error("unknown PageCount should show the page alone")
	}
	if buttons(n)[1].Style.Disabled {
		t.Error("Next stays enabled when the page count is unknown")
	}
}

func TestLoadMoreStates(t *testing.T) {
	ctx := core.NewContext()

	ctx.BeginRenderPass()
	n := LoadMore{}.Render(ctx)
	if n.Type != "Fragment" || len(n.Children) != 0 {
		t.Errorf("a complete list renders an empty Fragment, got %q with %d children", n.Type, len(n.Children))
	}

	ctx.BeginRenderPass()
	loaded := false
	n = LoadMore{HasMore: true, OnLoadMore: func() { loaded = true }}.Render(ctx)
	bs := buttons(n)
	if len(bs) != 1 || bs[0].Props["label"] != "Load more" {
		t.Fatalf("HasMore should render one Load more button, got %d", len(bs))
	}
	ctx.TriggerCallback(bs[0].Props["onClick"].(string))
	if !loaded {
		t.Error("tapping Load more should call OnLoadMore")
	}

	ctx.BeginRenderPass()
	n = LoadMore{HasMore: true, Loading: true, Err: errors.New("x"), OnLoadMore: func() {}}.Render(ctx)
	if findText(n, "Loading…") == nil || len(buttons(n)) != 0 {
		t.Error("Loading wins over both HasMore and Err")
	}

	ctx.BeginRenderPass()
	retried := false
	n = LoadMore{HasMore: true, Err: errors.New("timed out"), OnLoadMore: func() { retried = true }}.Render(ctx)
	if findText(n, "timed out") == nil {
		t.Error("Err should show its message")
	}
	bs = buttons(n)
	if len(bs) != 1 || bs[0].Props["label"] != "Retry" {
		t.Fatalf("Err should render one Retry button, got %d", len(bs))
	}
	ctx.TriggerCallback(bs[0].Props["onClick"].(string))
	if !retried {
		t.Error("Retry falls back to OnLoadMore when OnRetry is nil")
	}

	ctx.BeginRenderPass()
	n = LoadMore{Err: errors.New("raw"), ErrorText: "Could not load", OnRetry: func() {}}.Render(ctx)
	if findText(n, "Could not load") == nil || findText(n, "raw") != nil {
		t.Error("ErrorText should replace err.Error()")
	}
}
