//go:build grmob

package main

import (
	"github.com/rohanthewiz/grmob/core"
	"github.com/rohanthewiz/grmob/render"
	"myapp/app"
)

var manager *render.Manager

// Exported to native (called once)
func InitApp() {
	ctx := core.NewContext().With(
		core.WithThemeOpt(app.AppTheme),
		core.WithConfigOpt(app.Config),
	)
	manager = render.New(ctx, app.App)
}

// Exported to native (to get first render)
func RenderInitial() string {
	return manager.RenderAndGetPatches()
}

// Exported to native (to simulate external event). Dispatch goes through the
// manager so the handler and its render run under one render mutex.
func TriggerCallback(id string) string {
	return manager.DispatchCallback(id)
}

func TriggerTextCallback(id, val string) string {
	return manager.DispatchTextCallback(id, val)
}
