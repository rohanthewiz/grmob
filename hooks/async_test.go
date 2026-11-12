package hooks_test

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/rohanthewiz/grmob/core"
	"github.com/rohanthewiz/grmob/grmobtest"
	"github.com/rohanthewiz/grmob/hooks"
)

// asyncProbe renders one UseAsync slot and records what each pass returned.
// fetch sleeps for the delay its key asks for and answers "<key>": the
// network stand-in, on the bubble's fake clock.
type asyncProbe struct {
	ctx     *core.Context
	renders chan struct{}
	last    hooks.Async[string]
	runs    atomic.Int32
	delays  map[string]time.Duration
	fail    map[string]error
	key     string
	// seen records the context each run got, for the cancellation checks.
	seen chan context.Context
}

func newAsyncProbe(key string) *asyncProbe {
	p := &asyncProbe{
		ctx:     core.NewContext(),
		renders: make(chan struct{}, 64),
		delays:  map[string]time.Duration{},
		fail:    map[string]error{},
		key:     key,
		seen:    make(chan context.Context, 16),
	}
	p.ctx.OnStateChange(func() { p.renders <- struct{}{} })
	return p
}

func (p *asyncProbe) render() {
	key := p.key // captured at render time, as the hook's doc asks
	renderPass(p.ctx, func(ctx *core.Context) {
		p.last = hooks.UseAsync(ctx, func(c context.Context) (string, error) {
			p.runs.Add(1)
			p.seen <- c
			select {
			case <-time.After(p.delays[key]):
			case <-c.Done():
				return "", c.Err()
			}
			if err := p.fail[key]; err != nil {
				return "", err
			}
			return "answer " + key, nil
		}, key)
	})
}

// The first render already says Loading, and the answer arrives with a render
// request and is what the next pass returns.
func TestUseAsyncLoadsOnMountAndRendersTheAnswer(t *testing.T) {
	grmobtest.WithFakeClock(t, func(t *testing.T) {
		p := newAsyncProbe("a")
		p.delays["a"] = 200 * time.Millisecond
		defer p.ctx.Close()

		p.render()
		if !p.last.Loading || p.last.Data != "" || p.last.Err != nil {
			t.Fatalf("first pass = %+v, want Loading and nothing else", p.last)
		}
		awaitSignal(t, p.renders, "the render request when the answer lands")
		p.render()
		if p.last.Loading || p.last.Data != "answer a" || p.last.Err != nil {
			t.Fatalf("after the answer = %+v", p.last)
		}
		// A pass with the same deps starts nothing.
		p.render()
		if got := p.runs.Load(); got != 1 {
			t.Fatalf("fn ran %d times for one set of deps, want 1", got)
		}
	})
}

// A deps change cancels the run in flight and drops its answer, even when the
// old answer comes back last: the slow first fetch must not overwrite the
// fast second one.
func TestUseAsyncKeepsOnlyTheLatestDepsAnswer(t *testing.T) {
	grmobtest.WithFakeClock(t, func(t *testing.T) {
		p := newAsyncProbe("slow")
		p.delays["slow"] = time.Second
		p.delays["fast"] = 10 * time.Millisecond
		defer p.ctx.Close()

		p.render()
		first := <-p.seen
		p.key = "fast"
		p.render()
		if !p.last.Loading || p.last.Data != "" {
			t.Fatalf("a deps change must drop the old deps' outcome and load: %+v", p.last)
		}
		if first.Err() == nil {
			t.Fatal("the superseded run's context was not cancelled")
		}
		awaitSignal(t, p.renders, "the fast answer")
		p.render()
		if p.last.Data != "answer fast" {
			t.Fatalf("Data = %q, want the latest deps' answer", p.last.Data)
		}
		// Let any straggler finish; nothing may overwrite the answer.
		time.Sleep(2 * time.Second)
		p.render()
		if p.last.Data != "answer fast" || p.last.Loading {
			t.Fatalf("a superseded run changed the outcome: %+v", p.last)
		}
	})
}

// The stale-answer half without the context's help: a run that ignores
// cancellation and returns success late is still dropped.
func TestUseAsyncDropsAnAnswerThatIgnoredCancellation(t *testing.T) {
	grmobtest.WithFakeClock(t, func(t *testing.T) {
		ctx := core.NewContext()
		defer ctx.Close()
		var last hooks.Async[int]
		dep := 1
		pass := func() {
			d := dep
			renderPass(ctx, func(ctx *core.Context) {
				last = hooks.UseAsync(ctx, func(context.Context) (int, error) {
					time.Sleep(time.Duration(10-d) * 100 * time.Millisecond) // the first is slowest
					return d, nil
				}, d)
			})
		}
		pass()
		dep = 2
		pass()
		time.Sleep(2 * time.Second)
		pass()
		if last.Data != 2 {
			t.Fatalf("Data = %d, want 2: the first run's late answer overwrote the second's", last.Data)
		}
	})
}

// Reload starts again with the same deps, keeps Data and clears Err at once,
// so a Retry shows Loading rather than the error it is retrying.
func TestUseAsyncReloadKeepsDataAndClearsTheError(t *testing.T) {
	grmobtest.WithFakeClock(t, func(t *testing.T) {
		p := newAsyncProbe("k")
		p.delays["k"] = 50 * time.Millisecond
		defer p.ctx.Close()

		p.render()
		awaitSignal(t, p.renders, "the first answer")
		p.render()
		if p.last.Data != "answer k" {
			t.Fatalf("setup: %+v", p.last)
		}

		boom := errors.New("offline")
		p.fail["k"] = boom
		p.last.Reload()
		awaitSignal(t, p.renders, "Reload's own render request")
		p.render()
		if !p.last.Loading || p.last.Data != "answer k" || p.last.Err != nil {
			t.Fatalf("during a reload = %+v, want Loading with the old Data and no Err", p.last)
		}
		awaitSignal(t, p.renders, "the failed reload")
		p.render()
		if !errors.Is(p.last.Err, boom) || p.last.Loading || p.last.Data != "answer k" {
			t.Fatalf("after a failed reload = %+v, want the error with the old Data kept", p.last)
		}

		delete(p.fail, "k")
		p.last.Reload()
		p.render()
		if p.last.Err != nil || !p.last.Loading {
			t.Fatalf("Reload must clear Err at once: %+v", p.last)
		}
		awaitSignal(t, p.renders, "Reload's render request")
		awaitSignal(t, p.renders, "the retried answer")
		p.render()
		if p.last.Err != nil || p.last.Data != "answer k" {
			t.Fatalf("after a good retry = %+v", p.last)
		}
		if got := p.runs.Load(); got != 3 {
			t.Fatalf("fn ran %d times, want 3 (mount and two reloads)", got)
		}
	})
}

// Close cancels the run, nothing lands afterwards, Reload is a no-op on a
// closed tree, and a render after a remount starts again.
func TestUseAsyncStopsOnCloseAndStartsAgainOnARemount(t *testing.T) {
	grmobtest.WithFakeClock(t, func(t *testing.T) {
		p := newAsyncProbe("x")
		p.delays["x"] = time.Second

		p.render()
		run := <-p.seen
		reload := p.last.Reload
		p.ctx.Close()
		if run.Err() == nil {
			t.Fatal("Close did not cancel the run's context")
		}
		assertQuiet(t, p.renders, 2*time.Second, "render request after Close")
		reload()
		assertQuiet(t, p.renders, 2*time.Second, "render request from a Reload after Close")
		if got := p.runs.Load(); got != 1 {
			t.Fatalf("fn ran %d times; a Reload after Close must start nothing", got)
		}

		p.render() // the remount
		if !p.last.Loading {
			t.Fatalf("a render after Close must start the work again: %+v", p.last)
		}
		awaitSignal(t, p.renders, "the remount's answer")
		p.render()
		if p.last.Data != "answer x" {
			t.Fatalf("after the remount = %+v", p.last)
		}
		p.ctx.Close()
	})
}

// A panic in fn is recovered, logged, and becomes ErrAsyncPanicked.
func TestUseAsyncTurnsAPanicIntoAnError(t *testing.T) {
	grmobtest.WithFakeClock(t, func(t *testing.T) {
		ctx := core.NewContext()
		defer ctx.Close()
		renders := make(chan struct{}, 4)
		ctx.OnStateChange(func() { renders <- struct{}{} })
		var last hooks.Async[string]
		pass := func() {
			renderPass(ctx, func(ctx *core.Context) {
				last = hooks.UseAsync(ctx, func(context.Context) (string, error) {
					panic("the parser fell over")
				})
			})
		}
		pass()
		awaitSignal(t, renders, "the render request after the panic")
		pass()
		if !errors.Is(last.Err, hooks.ErrAsyncPanicked) || last.Loading {
			t.Fatalf("after a panic = %+v, want ErrAsyncPanicked", last)
		}
		core.ClearConcerns() // the handler-panic concern this test raised on purpose
	})
}
