package permission

import (
	"go/ast"
	"go/parser"
	"go/token"
	"sort"
	"strconv"
	"sync"
	"testing"

	"github.com/rohanthewiz/grmob/core"
)

// withHost installs a recording system-event handler for the duration of a
// test and returns the events it saw.
//
// The handler has to be installed even by tests that do not read the events,
// because its *absence* is a behaviour: send short-circuits to Unavailable
// when nothing is listening (see TestNoHostMeansUnavailableRatherThanWaiting),
// so a test that forgot this would be exercising the headless path while
// believing it was exercising the wire.
func withHost(t *testing.T) *recorder {
	t.Helper()
	resetForTest()

	r := &recorder{}
	core.SetSystemEventHandler(func(name string, data map[string]any) {
		r.add(name, data)
	})
	t.Cleanup(func() {
		core.SetSystemEventHandler(nil)
		resetForTest()
	})
	return r
}

type recorder struct {
	mu     sync.Mutex
	events []event
}

type event struct {
	name string
	data map[string]any
}

func (r *recorder) add(name string, data map[string]any) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.events = append(r.events, event{name, data})
}

func (r *recorder) all() []event {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]event(nil), r.events...)
}

func (r *recorder) only(t *testing.T) event {
	t.Helper()
	all := r.all()
	if len(all) != 1 {
		t.Fatalf("want exactly one system event, got %d: %+v", len(all), all)
	}
	return all[0]
}

// --- The wire ---------------------------------------------------------------

// Both commands go out under one event name with the permission as a field.
//
// One name and a kind, rather than two event names or one per permission, for
// the reason core's sensor channel is shaped that way: a host's dispatch stays
// a single switch, and the next permission is a constant here rather than an
// arm in three shells.
func TestRequestAndCheckAreOneEventWithACommand(t *testing.T) {
	for _, tc := range []struct {
		call    func(Permission)
		command string
	}{
		{Request, "request"},
		{Check, "check"},
	} {
		r := withHost(t)
		tc.call(Camera)

		got := r.only(t)
		if got.name != "permission" {
			t.Errorf("event name = %q, want %q", got.name, "permission")
		}
		if got.data["command"] != tc.command {
			t.Errorf("command = %v, want %q", got.data["command"], tc.command)
		}
		if got.data["kind"] != "camera" {
			t.Errorf("kind = %v, want %q — the permission travels as a field, not as "+
				"a second event name", got.data["kind"], "camera")
		}
	}
}

// The kind is the constant's own string, unwrapped. Both natives and both web
// runtimes compare it against string literals, so a Permission that crossed as
// anything else would match no arm anywhere and be silently dropped.
func TestEveryPermissionCrossesAsItsOwnSpelling(t *testing.T) {
	for _, p := range Permissions() {
		r := withHost(t)
		Check(p)
		if got := r.only(t).data["kind"]; got != string(p) {
			t.Errorf("kind for %v = %v, want the constant verbatim", p, got)
		}
	}
}

// --- The record -------------------------------------------------------------

func TestAHostAnswerBecomesTheCurrentStatus(t *testing.T) {
	withHost(t)

	if got := Current(Camera); got != Unknown {
		t.Fatalf("a permission nobody asked about = %q, want the zero value", got)
	}
	core.ReceiveHostEvent("permission", map[string]any{
		"kind": "camera", "status": "granted",
	})
	if got := Current(Camera); got != Granted {
		t.Errorf("after the host answered: %q, want %q", got, Granted)
	}
	if !IsGranted(Camera) {
		t.Error("IsGranted disagrees with Current")
	}
}

// Each permission has its own slot: an answer about one must not move another.
// The channel is shared, so this is the property that makes it usable.
func TestOnePermissionsAnswerLeavesTheOthersAlone(t *testing.T) {
	withHost(t)

	core.ReceiveHostEvent("permission", map[string]any{
		"kind": "camera", "status": "granted",
	})
	for _, p := range Permissions() {
		want := Unknown
		if p == Camera {
			want = Granted
		}
		if got := Current(p); got != want {
			t.Errorf("%v = %q, want %q", p, got, want)
		}
	}
}

// IsGranted is true for Granted alone. The three other stated statuses and the
// zero value all mean "do not open the camera yet", and a helper that guessed
// otherwise would be a security-shaped bug wearing a convenience's name.
func TestIsGrantedIsTrueForGrantedAlone(t *testing.T) {
	// Unknown first, from a fresh record: nothing has been asked, so the
	// helper must not read that as permission.
	withHost(t)
	if IsGranted(Location) {
		t.Error("an unasked permission reads as granted — the unknown cases must fall to " +
			"not-yet-usable, or a screen opens a camera the OS has not")
	}

	for _, tc := range []struct {
		status Status
		want   bool
	}{
		{Granted, true},
		{Denied, false},
		{Prompt, false},
		{Unavailable, false},
	} {
		withHost(t)
		Receive(Location, tc.status)
		if got := IsGranted(Location); got != tc.want {
			t.Errorf("status %q: IsGranted = %v, want %v", tc.status, got, tc.want)
		}
	}
}

// --- Malformed answers ------------------------------------------------------

// A payload this package cannot read leaves the record where it was.
//
// Every alternative to dropping it is a lie with a cost: Granted opens a
// device the OS did not authorise, Denied hides a feature that works, and
// Unavailable sends the user to a settings screen with no switch on it. The
// screen keeps drawing what it was drawing, which is the one outcome that
// misleads nobody.
func TestAMalformedAnswerChangesNothing(t *testing.T) {
	withHost(t)
	Receive(Camera, Granted)

	for _, payload := range []map[string]any{
		{"status": "denied"},                        // no kind
		{"kind": "camera"},                          // no status
		{"kind": "camera", "status": "maybe"},       // no such status
		{"kind": "bluetooth", "status": "denied"},   // no such permission
		{"kind": "camera", "status": ""},            // Unknown is not an answer
		{"kind": 7, "status": "denied"},             // not even a string
		{"kind": "camera", "status": []string{"x"}}, // nor this
	} {
		core.ReceiveHostEvent("permission", payload)
		if got := Current(Camera); got != Granted {
			t.Errorf("payload %+v moved the record to %q", payload, got)
		}
	}
}

// Unknown is refused as an *incoming* status even through the typed door,
// because a host reporting it would be answering the question with the absence
// of an answer — and a screen reading Unknown draws the spinner it draws
// before anyone has asked.
func TestAHostCannotReportUnknown(t *testing.T) {
	withHost(t)
	Receive(Camera, Granted)
	Receive(Camera, Unknown)

	if got := Current(Camera); got != Granted {
		t.Errorf("Unknown was accepted as an answer: %q", got)
	}
}

// --- Subscriptions ----------------------------------------------------------

func TestSubscribersHearChangesAndCanCancel(t *testing.T) {
	withHost(t)

	type call struct {
		p Permission
		s Status
	}
	var got []call
	cancel := On(func(p Permission, s Status) { got = append(got, call{p, s}) })

	Receive(Camera, Prompt)
	Receive(Camera, Granted)
	cancel()
	Receive(Camera, Denied)

	want := []call{{Camera, Prompt}, {Camera, Granted}}
	if len(got) != len(want) {
		t.Fatalf("heard %+v, want %+v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("call %d = %+v, want %+v", i, got[i], want[i])
		}
	}
	// Cancelled, but the record still moves — On observes, it does not gate.
	if Current(Camera) != Denied {
		t.Errorf("the record stopped moving when the subscriber left: %q", Current(Camera))
	}
}

// A repeat of the status already on record notifies nobody.
//
// Not a micro-optimisation: the intended way to notice a permission changed in
// the system settings is to re-Check on every foreground, and the overwhelming
// majority of those answer with the same status. Notifying on each would put a
// render pass on every app resume, for news that is not news.
func TestARepeatedAnswerDoesNotNotify(t *testing.T) {
	withHost(t)

	calls := 0
	defer On(func(Permission, Status) { calls++ })()

	Receive(Camera, Granted)
	Receive(Camera, Granted)
	Receive(Camera, Granted)

	if calls != 1 {
		t.Errorf("%d notifications for one distinct answer, want 1", calls)
	}
}

// A subscriber may read the record and manage subscriptions from inside its
// own handler: the notify loop runs outside the lock, so this deadlocks if
// that ever stops being true.
func TestASubscriberCanReadAndSubscribeFromItsOwnHandler(t *testing.T) {
	withHost(t)

	done := make(chan Status, 1)
	defer On(func(p Permission, _ Status) {
		inner := On(func(Permission, Status) {})
		inner()
		done <- Current(p)
	})()

	Receive(Camera, Granted)
	select {
	case got := <-done:
		if got != Granted {
			t.Errorf("the handler read %q", got)
		}
	default:
		t.Fatal("the handler never ran")
	}
}

// --- No host ----------------------------------------------------------------

// With no system-event sink attached there is no platform to ask, and that is
// what Unavailable means — so it is reported immediately rather than left as
// Unknown, which a screen draws a spinner for.
//
// This is the state of every Go test, every htmlout export and any embedder
// that has not wired a shell, so the alternative is not hypothetical: it is a
// permission-gated screen that spins forever under test.
func TestNoHostMeansUnavailableRatherThanWaiting(t *testing.T) {
	resetForTest()
	core.SetSystemEventHandler(nil)
	t.Cleanup(resetForTest)

	Check(Camera)
	if got := Current(Camera); got != Unavailable {
		t.Errorf("headless Check left %q, want %q", got, Unavailable)
	}
	Request(Microphone)
	if got := Current(Microphone); got != Unavailable {
		t.Errorf("headless Request left %q, want %q", got, Unavailable)
	}
}

// --- The enumerations -------------------------------------------------------

// Permissions() and Statuses() restate const blocks a few lines away, which is
// the duplication that quietly goes stale — nothing about adding a fifth
// permission forces anyone to scroll down. Both lists are what the host
// coverage checks in mobile/verify and wasm/verify rest on, so they are pinned
// to their declarations rather than trusted.
//
// The syntax tree rather than a text match, for the reason core's enum pins
// use one: the question is syntactic, and the doc comments in this file spell
// out "granted", "denied" and "prompt" in prose several times over.
func TestPermissionsMatchesTheDeclaredConstants(t *testing.T) {
	requireExactEnum(t, "Permission", "Permissions()", Permissions())
}

// Statuses() is deliberately narrower than the type: Unknown is declared and
// must stay out, for the reason core.RoleNone stays out of Roles().
func TestStatusesAreTheAnswersAHostCanGive(t *testing.T) {
	requireExactEnum(t, "Status", "Statuses() plus Unknown", append(Statuses(), Unknown))

	if Unknown != "" {
		t.Errorf("Unknown = %q, want the empty string so an unrecorded permission is it", Unknown)
	}
	for _, s := range Statuses() {
		if s == Unknown {
			t.Error("Statuses() lists Unknown; the census is the answers a host can give")
		}
	}
}

// The spellings are the W3C Permissions API's own, which is what lets the
// browser host answer with the string it was handed. Pinned as literals rather
// than derived, because a test that read them from the constants would agree
// with any rename.
func TestTheStatusSpellingsAreTheBrowsersOwn(t *testing.T) {
	for _, pin := range []struct {
		got  Status
		want string
	}{
		{Granted, "granted"},
		{Denied, "denied"},
		{Prompt, "prompt"},
	} {
		if string(pin.got) != pin.want {
			t.Errorf("%q, want %q — navigator.permissions answers in these words and the "+
				"browser host forwards them unchanged", pin.got, pin.want)
		}
	}
	// Unavailable is *not* the browser's: the Permissions API has three states
	// and this is the fourth, for a descriptor no browser knows and for a
	// device that has no such hardware. It is spelled plainly rather than
	// borrowed.
	if Unavailable != "unavailable" {
		t.Errorf("Unavailable = %q", Unavailable)
	}
}

// requireExactEnum holds a list function against the const block it restates.
func requireExactEnum[T ~string](t *testing.T, typeName, listName string, values []T) {
	t.Helper()

	declared := declaredConstants(t, typeName)

	listed := map[string]bool{}
	for _, v := range values {
		s := string(v)
		if listed[s] {
			t.Errorf("%s lists %q twice", listName, s)
		}
		listed[s] = true
	}

	names := make([]string, 0, len(declared))
	for name := range declared {
		names = append(names, name)
	}
	sort.Strings(names)

	for _, name := range names {
		if !listed[declared[name]] {
			t.Errorf("%s = %q is declared in permission.go but missing from %s",
				name, declared[name], listName)
		}
		delete(listed, declared[name])
	}
	left := make([]string, 0, len(listed))
	for v := range listed {
		left = append(left, v)
	}
	sort.Strings(left)
	for _, v := range left {
		t.Errorf("%s yields %q, which no %s constant declares", listName, v, typeName)
	}
}

// declaredConstants returns the constants of the named type declared in
// permission.go, as name -> value. It fails rather than returning an empty map
// at every step: a check that reads nothing must not read as a pass.
func declaredConstants(t *testing.T, typeName string) map[string]string {
	t.Helper()

	file, err := parser.ParseFile(token.NewFileSet(), "permission.go", nil, 0)
	if err != nil {
		t.Fatalf("parsing permission.go: %v", err)
	}

	out := map[string]string{}
	for _, decl := range file.Decls {
		gen, ok := decl.(*ast.GenDecl)
		if !ok || gen.Tok != token.CONST {
			continue
		}
		for _, spec := range gen.Specs {
			vs, ok := spec.(*ast.ValueSpec)
			if !ok || vs.Type == nil {
				continue
			}
			if id, ok := vs.Type.(*ast.Ident); !ok || id.Name != typeName {
				continue
			}
			for i, name := range vs.Names {
				if i >= len(vs.Values) {
					continue
				}
				lit, ok := vs.Values[i].(*ast.BasicLit)
				if !ok || lit.Kind != token.STRING {
					continue
				}
				value, err := strconv.Unquote(lit.Value)
				if err != nil {
					t.Fatalf("unquoting %s: %v", name.Name, err)
				}
				out[name.Name] = value
			}
		}
	}
	if len(out) == 0 {
		t.Fatalf("permission.go declares no %s constants — has the type been renamed?", typeName)
	}
	return out
}
