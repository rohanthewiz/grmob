package tutorial

import (
	"testing"
	"time"

	"github.com/rohanthewiz/grmob/render"
)

// Chapter 3 demo-liveness tests. The hooks lessons are the first whose demos
// change without a dispatch — ticks, timeouts, and effect goroutines write
// state on their own schedule — so alongside the tap/type discipline these
// tests add one waiting primitive: poll the rendered tree until an expected
// text appears, with a deadline that only bounds a hang (the hooks package's
// own tests own the precise timing semantics; here the question is just
// "does the demo come alive on screen").

// awaitText re-renders and re-reads the tree until sub appears somewhere in
// it. Polling the tree rather than sleeping a fixed interval keeps the pass
// count honest with debug mode — every poll is an audited render — and keeps
// the test as fast as the demo, not as slow as a worst-case sleep.
func awaitText(t *testing.T, mgr *render.Manager, sub string) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for !hasTextContaining(tree(t, mgr), sub) {
		if time.Now().After(deadline) {
			t.Fatalf("timed out waiting for %q to render", sub)
		}
		time.Sleep(25 * time.Millisecond)
	}
}

// --- 3.1 The clock --------------------------------------------------------

func TestClockDemoTicksAndPauses(t *testing.T) {
	mgr := newApp(t)
	openLesson(t, mgr, "The clock: UseInterval")

	if !hasText(tree(t, mgr), "0 s") {
		t.Fatal("the clock should mount at 0 s")
	}
	awaitText(t, mgr, "1 s") // one real tick proves the interval → Set → render loop

	// Pausing flips state the next tick's closure will read — the structural
	// evidence (the paused caption) is immediate, no second tick needed.
	toggleCheckbox(t, mgr, 0, true)
	if !hasTextContaining(tree(t, mgr), "Paused") {
		t.Fatal("pausing should reveal the paused caption")
	}
	assertNoConcerns(t)
}

// --- 3.2 The timeout ------------------------------------------------------

func TestTimeoutDemoFiresOnceAndStaysFired(t *testing.T) {
	mgr := newApp(t)
	openLesson(t, mgr, "Once, later: UseTimeout")

	if !hasTextContaining(tree(t, mgr), "Nothing yet") {
		t.Fatal("the timeout demo should open on its pending caption")
	}
	awaitText(t, mgr, "Right on time")

	// Renders after the fire must not re-arm the slot: poke twice and the
	// fired card must still be there (a re-armed timer would flip the demo
	// back to "Nothing yet" until it fired again).
	tap(t, mgr, "Poke a render")
	tap(t, mgr, "Poke a render")
	cur := tree(t, mgr)
	if !hasTextContaining(cur, "renders poked: 2") {
		t.Fatal("both pokes should be counted")
	}
	if !hasTextContaining(cur, "Right on time") || hasTextContaining(cur, "Nothing yet") {
		t.Fatal("poking renders must not re-arm the fired timeout")
	}
	assertNoConcerns(t)
}

// --- 3.3 Effects ----------------------------------------------------------

func TestEffectDemoFetchesOnMountAndOnDepsChange(t *testing.T) {
	mgr := newApp(t)
	openLesson(t, mgr, "Effects: UseEffect")

	// Mount: both effects run — the no-deps one flips the mounted line, the
	// dep'd one delivers Ziggy's profile after the simulated latency.
	awaitText(t, mgr, "digs tunnels")
	cur := tree(t, mgr)
	if !hasTextContaining(cur, "fetch effect runs: 1") {
		t.Fatal("the mount fetch should be counted exactly once")
	}
	if !hasTextContaining(cur, "ran once, at mount") {
		t.Fatal("the no-deps effect should have marked the mount")
	}

	// Changing the dep re-runs the effect: the loading caption derives from
	// got.id != sel, so it shows immediately, then the fetch lands.
	tap(t, mgr, "Pip")
	if !hasTextContaining(tree(t, mgr), "fetching Pip's profile") {
		t.Fatal("switching gophers should derive a loading caption")
	}
	awaitText(t, mgr, "ships the largest diffs")
	if !hasTextContaining(tree(t, mgr), "fetch effect runs: 2") {
		t.Fatal("the dep change should have re-run the fetch once")
	}

	// Re-selecting the same gopher re-renders with an unchanged dep — the
	// effect must not run again. Give a wrongly re-run effect enough time to
	// have landed (its own delay plus margin) before reading the count.
	tap(t, mgr, "Pip")
	time.Sleep(effectFetchDelay + 250*time.Millisecond)
	if !hasTextContaining(tree(t, mgr), "fetch effect runs: 2") {
		t.Fatal("an unchanged dep must not re-run the effect")
	}
	assertNoConcerns(t)
}

// --- 3.4 Memoization ------------------------------------------------------

func TestMemoDemoRecomputesOnlyWhenDepsChange(t *testing.T) {
	mgr := newApp(t)
	openLesson(t, mgr, "Caching: UseMemo")

	if !hasTextContaining(tree(t, mgr), "compute() calls: 1") {
		t.Fatal("the memo should compute once on mount")
	}

	typeInto(t, mgr, "go")
	cur := tree(t, mgr)
	if !hasTextContaining(cur, "compute() calls: 2") {
		t.Fatal("a changed query dep should recompute exactly once")
	}
	if !hasText(cur, "gopher · goroutine") {
		t.Fatal("filtering on 'go' should match gopher and goroutine")
	}

	// Forced re-renders change no dep — and note every tree() read in this
	// test is itself a full render pass, all of them cache hits.
	tap(t, mgr, "Re-render (changes no dep)")
	tap(t, mgr, "Re-render (changes no dep)")
	cur = tree(t, mgr)
	if !hasTextContaining(cur, "forced re-renders: 2") {
		t.Fatal("both forced re-renders should be counted")
	}
	if !hasTextContaining(cur, "compute() calls: 2") {
		t.Fatal("re-renders with unchanged deps must not recompute")
	}
	assertNoConcerns(t)
}

// --- 3.5 Reducer ----------------------------------------------------------

func TestReducerDemoAppliesActions(t *testing.T) {
	mgr := newApp(t)
	openLesson(t, mgr, "Actions: UseReducer")

	cur := tree(t, mgr)
	if !hasText(cur, "0") || !hasText(cur, "0 moves") {
		t.Fatal("the score should mount at 0 with 0 moves")
	}

	tap(t, mgr, "+1")
	tap(t, mgr, "+1")
	tap(t, mgr, "+5")
	cur = tree(t, mgr)
	if !hasText(cur, "7") {
		t.Fatal("+1 +1 +5 should score 7")
	}
	if !hasText(cur, "3 moves") {
		t.Fatal("three dispatches should count 3 moves — score and moves step together")
	}

	tap(t, mgr, "Reset")
	cur = tree(t, mgr)
	if !hasText(cur, "0") || !hasText(cur, "0 moves") {
		t.Fatal("Reset should zero both fields in one action")
	}
	assertNoConcerns(t)
}
