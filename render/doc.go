// Package render is the loop that turns an app function into patches: it owns
// the retained tree, serialises every pass, and decides when a pass happens.
//
// A [Manager] is constructed once per app with a root view function, and from
// then on it is the only thing that calls that function.
//
//	m := render.New(core.NewContext(), App)
//	m.SetListener(hostListener)     // optional: the Go -> host push channel
//	first := m.RenderInitial()      // the whole tree, as JSON
//	patches := m.Dispatch("tick", fn)
//
// # Two ways in, one at a time
//
// A pass starts from one of two directions, and the Manager's mutex is what
// keeps them from interleaving:
//
//	host event ──▶ Dispatch* ──┐
//	                           ├──▶ render ──▶ Diff ──▶ patches
//	State.Set  ──▶ push pump ──┘      (mutex held for the whole pass)
//
// Both paths mutate state that cannot tolerate interleaving — the context's
// hook cursor and its callback registry are both reset at the start of a pass
// — so an event handler can never run in the middle of one. The practical
// consequence for an app is a guarantee worth relying on: a handler always
// observes a settled tree, and the writes it makes are rendered together by the
// pass that follows.
//
// # Pushed updates are coalesced
//
// State written outside a host event (a timer, a network response, any
// goroutine) reaches the screen through [PatchListener]. Nudges are coalesced
// through a one-slot buffer, so a burst of writes produces one pass over the
// settled state rather than one pass per write, and the last write is never
// lost. An empty patch set is never pushed.
//
// [PatchListener.ApplyPatches] is called from a background goroutine. A native
// implementation must hop to its own UI thread before touching views.
//
// # Closing
//
// [Manager.Close] stops the pump and closes the context tree, which stops the
// background resources hooks registered on it. It is the one shutdown entry
// point, which is what lets a host that replaces a running app — mobile.Register,
// the WASM runtime re-mounting, a hot reload — do so without leaking a ticker
// that renders into a dead tree.
package render
