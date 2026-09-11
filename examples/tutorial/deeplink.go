package tutorial

import (
	"strings"
	"sync"

	"github.com/rohanthewiz/grmob/core"
)

// Deep links: a URL that opens a particular lesson, and an address bar that
// follows the reader from lesson to lesson so any moment of the tutorial can
// be copied and shared. On the web host that is the page's hash
// (https://rohanthewiz.github.io/grmob/#2.3); the natives have no address
// bar, and nothing here assumes one.
//
// On the web the app does not know what a URL is. It speaks the two generic
// channels core already has, and the host page translates:
//
//	page ──HostEvent("route", {lesson: "2.3"})──▶ app   (boot, hashchange)
//	page ◀──SendSystemEvent("route", {lesson})──── app   (every navigation)
//
// Inbound, the page reports the lesson its URL names and the app navigates
// there. Outbound, every door into or out of a lesson — a contents row,
// Prev/Next, ‹ Contents, Finish — reports the lesson now on screen, and the
// page rewrites its hash. A native shell drops the unknown system event
// (SystemEvents.swift / SystemEvents.kt fall through on names they do not
// know) and never sends the host event, so the natives ignore the outbound
// half by construction rather than by a platform check.
//
// # The natives, where there is no page to translate
//
// A phone has no address bar, so nothing can send "route" — which for a long
// time meant the natives had no deep links at all and reaching lesson 4.12 on
// a device was a scroll through 49 rows. core.OnDeepLink is the other inbound
// path, and on the natives it is the only one:
//
//	OS ──HostEvent("deeplink", {url: "grmob://lesson/4.12"})──▶ app
//
// So the app does have to know what a URL is here, and that is the right place
// for it: the shells forward the string verbatim precisely so that what
// "grmob://lesson/4.12" means is the tutorial's business and not theirs. See
// core/deeplink.go.
//
// The two inbound paths converge on goTo and neither echoes back, so a link
// resolves in one hop on every host.
//
// The two directions do not echo each other: a "route" host event makes the
// app navigate *without* sending "route" back, because the page's hash is
// already what it is; and the page rewrites the hash with replaceState,
// which fires no hashchange. So a shared link resolves in one hop.

// routeEvent is the name on both channels. The payload is one string field,
// "lesson", carrying a lesson ID ("2.3"), a chapter number ("2", which opens
// the chapter's first lesson) or "" for the contents screen. Outbound it is
// always a lesson ID: the app reports what is on screen, never a shorthand.
const routeEvent = "route"

// routeRecord is the hook-slot memory of useDeepLinks: whether this app's
// subscription to the host event is live. Same shape as hooks.UseLifecycle's
// record, for the same reason — App runs on every render pass, so the
// subscription has to be taken exactly once per context tree, and the slot is
// the only per-tree memory a render function has.
type routeRecord struct {
	mu         sync.Mutex
	subscribed bool
}

// useDeepLinks subscribes the app to inbound route events, once per context
// tree, and releases the subscription when the tree closes (render.Manager
// .Close), which is what lets the browser host re-mount and lets tests build
// one app after another without stale subscribers steering the wrong stack.
//
// It is called from App on the session scope — the context *above* the
// Navigator — because a route change has to survive frame swaps: the handler
// pushes and replaces frames, so it cannot live in one.
func (t *tutorial) useDeepLinks(ctx *core.Context) {
	slot := core.NewState(ctx, &routeRecord{})
	rec := slot.Get()

	rec.mu.Lock()
	already := rec.subscribed
	rec.subscribed = true
	rec.mu.Unlock()
	if already {
		return
	}

	cancel := core.OnHostEvent(routeEvent, func(data map[string]any) {
		id, _ := data["lesson"].(string)
		t.goTo(ctx, id)
	})
	// And the natives' path, which carries a URL rather than a lesson ID. Both
	// subscriptions are taken together and released together: they are two
	// spellings of the same request and no host sends both.
	cancelLink := core.OnDeepLink(func(url string) {
		id, ok := lessonFromURL(url)
		if !ok {
			return
		}
		t.goTo(ctx, id)
	})
	ctx.OnClose(func() {
		cancel()
		cancelLink()
		rec.mu.Lock()
		rec.subscribed = false
		rec.mu.Unlock()
	})
}

// lessonLinkPrefix is the one URL shape this app answers to. A scheme the
// shells declare (AndroidManifest.xml, ios/project.yml) and a single path
// segment carrying what the "route" event calls `lesson`.
const lessonLinkPrefix = "grmob://lesson/"

// lessonFromURL reads a lesson reference out of a deep link, or reports that
// there is not one.
//
// The accepted forms are exactly the ones routeEvent's payload accepts, which
// is the point — a link is a second spelling of the same request and must not
// be able to express anything the web hash cannot:
//
//	grmob://lesson/4.12   a lesson
//	grmob://lesson/4      a chapter, which opens its first lesson
//	grmob://lesson/       the contents screen
//
// Anything else returns false and is dropped. That includes a URL on this
// app's scheme with some other path (grmob://settings), because a shell
// forwards every URL it is handed and an app that treated an unrecognised one
// as "" would navigate the reader to the contents screen for a link that was
// not about lessons at all.
//
// Unresolvable IDs are not this function's problem: goTo puts them through
// resolveRoute, which is the same gate the web hash goes through, so
// "grmob://lesson/99.99" is dropped there rather than being validated twice.
func lessonFromURL(url string) (string, bool) {
	rest, ok := strings.CutPrefix(url, lessonLinkPrefix)
	if !ok {
		return "", false
	}
	// A trailing slash is the contents screen spelled the way a URL spells an
	// empty last segment, and anything deeper is not a shape this app claims.
	rest = strings.TrimSuffix(rest, "/")
	if strings.Contains(rest, "/") {
		return "", false
	}
	return rest, true
}

// goTo navigates to the lesson a host route names, or to the contents for
// "". It is the inbound half: it moves the stack but does not report the
// move back, for the reason given in the package comment above.
//
// Which mutation it uses depends on where the reader is, so that the stack
// keeps the shape the buttons maintain — [contents] or [contents, lesson],
// never deeper. From the contents a lesson is pushed, as a row tap would; from
// another lesson it is replaced, as Next would; "" pops to the root. An ID
// that names no lesson (a stale or mistyped link) is ignored, and so is the
// lesson already on screen — a hashchange that changes nothing must not
// discard the demo state of the lesson being read.
func (t *tutorial) goTo(ctx *core.Context, id string) {
	if id == "" {
		if t.current.Get() != "" {
			t.current.Set("")
			core.PopToRoot(ctx)
		}
		return
	}
	e, ok := resolveRoute(id)
	if !ok {
		return
	}
	// Compared after resolving, so that "3" while 3.1 is showing is the
	// no-op it should be, not a rebuild of the frame being read.
	if e.ID == t.current.Get() {
		return
	}
	t.markVisited(e.ID)
	if t.current.Get() == "" {
		t.current.Set(e.ID)
		core.Push(ctx, t.lessonRoute(e.Index))
		return
	}
	t.current.Set(e.ID)
	core.Replace(ctx, t.lessonRoute(e.Index))
}

// open is the one door into a lesson for the app's own controls. It records
// progress, moves the stack (Push from the contents, Replace between
// lessons — see lessonRoute for why), and reports the new lesson to the
// host. Every button that used to call markVisited + Push/Replace calls this
// instead, so the address bar cannot drift from the screen because one path
// forgot to report.
func (t *tutorial) open(ctx *core.Context, index int, replace bool) {
	e := flatLessons[index]
	t.markVisited(e.ID)
	t.current.Set(e.ID)
	if replace {
		core.Replace(ctx, t.lessonRoute(index))
	} else {
		core.Push(ctx, t.lessonRoute(index))
	}
	reportRoute(e.ID)
}

// toContents is the matching door out: ‹ Contents and Finish both land on the
// table of contents, and both report that the URL should name no lesson.
func (t *tutorial) toContents(ctx *core.Context) {
	t.current.Set("")
	core.Pop(ctx)
	reportRoute("")
}

// reportRoute tells the host which lesson is on screen. Fire-and-forget,
// like a toast: a host with no address bar has nothing to do with it.
func reportRoute(id string) {
	core.SendSystemEvent(routeEvent, map[string]any{"lesson": id})
}

// resolveRoute turns what a link names into a lesson. Two spellings are
// accepted: a lesson ID ("2.3"), and a bare chapter number ("2"), which is
// the chapter's first lesson — a link to a chapter is a link to where
// reading it starts, and the tutorial has no chapter screen to land on
// instead. Anything else (a chapter or lesson past the end, a stray string)
// resolves to nothing and the caller ignores it.
//
// Linear scans: there are forty lessons and this runs once per link.
func resolveRoute(id string) (lessonEntry, bool) {
	for _, e := range flatLessons {
		if e.ID == id {
			return e, true
		}
	}
	for _, e := range flatLessons {
		if e.ChapterNum == chapterNumber(id) {
			return e, true // the flat index is in reading order, so this is lesson N.1
		}
	}
	return lessonEntry{}, false
}

// chapterNumber parses a bare chapter route ("3") as a 1-based chapter
// number, or 0 for anything that is not one. Hand-rolled rather than
// strconv.Atoi because Atoi accepts "+3", " 3" and "03", none of which is
// an address the app ever reports, and a link should round-trip exactly.
func chapterNumber(id string) int {
	if id == "" || id[0] == '0' {
		return 0
	}
	n := 0
	for _, r := range id {
		if r < '0' || r > '9' {
			return 0
		}
		n = n*10 + int(r-'0')
	}
	return n
}
