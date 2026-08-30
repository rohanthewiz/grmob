package mobile_test

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/rohanthewiz/grmob/core"
	"github.com/rohanthewiz/grmob/mobile"
)

func testApp(ctx *core.Context) core.View {
	return core.ComponentFunc(func(ctx *core.Context) *core.Node {
		count := core.NewState(ctx, 0)
		return core.Column(
			core.Text(fmt.Sprintf("count: %d", count.Get())),
			core.Button("increment", func() {
				count.Set(count.Get() + 1)
			}),
		).Render(ctx)
	})
}

// TestBridgeEventRoundTrip exercises the exact call sequence a native shell
// makes over the bound API: register, initial render, then dispatch an event
// by callback ID and apply the returned patches. No listener is attached, so
// the synchronous event path must carry the full result.
func TestBridgeEventRoundTrip(t *testing.T) {
	mobile.Register(core.NewContext(), testApp)

	var tree struct {
		Children []struct {
			Props map[string]any
		}
	}
	if err := json.Unmarshal([]byte(mobile.RenderInitial()), &tree); err != nil {
		t.Fatalf("initial render is not valid JSON: %v", err)
	}
	onClick, ok := tree.Children[1].Props["onClick"].(string)
	if !ok || onClick == "" {
		t.Fatalf("button is missing its onClick ID: %+v", tree.Children[1].Props)
	}

	out := mobile.TriggerCallback(onClick)
	if !strings.Contains(out, "update-props") || !strings.Contains(out, "count: 1") {
		t.Errorf("TriggerCallback should return the state-change patches, got: %s", out)
	}

	// Re-registering must cleanly replace the app (old pump shut down, fresh
	// state), since a native process can re-init after e.g. an Activity restart.
	mobile.Register(core.NewContext(), testApp)
	if !strings.Contains(mobile.RenderInitial(), "count: 0") {
		t.Errorf("re-registered app should start from fresh state")
	}
}
