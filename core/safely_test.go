package core

import (
	"bytes"
	"log"
	"strings"
	"sync"
	"testing"
)

// lockedBuffer is a log sink safe to write from one goroutine and read from
// another, which is how Safely is used.
type lockedBuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (b *lockedBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.Write(p)
}

func (b *lockedBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.String()
}

// captureLog sends the standard logger to a buffer for the test's length.
func captureLog(t *testing.T) *lockedBuffer {
	t.Helper()
	buf := &lockedBuffer{}
	prev := log.Writer()
	log.SetOutput(buf)
	t.Cleanup(func() { log.SetOutput(prev) })
	return buf
}

// elementAt indexes s at i. A function, so the out-of-range read in the test
// below is a runtime panic and not something the compiler or vet can see.
func elementAt(s []int, i int) int { return s[i] }

func TestSafelyRunsFnAndReportsANormalReturn(t *testing.T) {
	logged := captureLog(t)
	ran := false
	if !Safely("probe", func() { ran = true }) {
		t.Fatal("Safely reported a panic for a function that returned")
	}
	if !ran {
		t.Fatal("Safely did not run fn")
	}
	if logged.String() != "" {
		t.Fatalf("a normal return logged %q", logged.String())
	}
}

// The panic is recovered on the calling goroutine, logged with where it
// happened and a stack, and reported as false. A goroutine running this test
// body would have ended the test binary without the recover.
func TestSafelyRecoversAPanicAndLogsWhereAndTheStack(t *testing.T) {
	logged := captureLog(t)
	done := make(chan bool)
	go func() {
		done <- Safely("UseInterval tick", func() {
			// An out-of-range read: a runtime panic, as a real bug would be.
			_ = elementAt(nil, 3)
		})
	}()
	if ok := <-done; ok {
		t.Fatal("Safely reported a normal return for a panicking function")
	}
	out := logged.String()
	for _, want := range []string{
		"grmob: recovered panic during UseInterval tick",
		"index out of range [3] with length 0",
		"safely_test.go", // the stack names the panicking frame
	} {
		if !strings.Contains(out, want) {
			t.Errorf("log lacks %q:\n%s", want, out)
		}
	}
}

// Under debug mode the panic is also a handler-panic concern naming where; with
// debug mode off nothing is collected.
func TestSafelyReportsAConcernOnlyUnderDebugMode(t *testing.T) {
	captureLog(t)
	ClearConcerns()
	t.Cleanup(ClearConcerns)

	Safely("Debouncer call", func() { panic("boom") })
	if got := Concerns(); len(got) != 0 {
		t.Fatalf("debug mode off collected %v", got)
	}

	SetDebugMode(true)
	t.Cleanup(func() { SetDebugMode(false) })
	Safely("Debouncer call", func() { panic("boom") })
	got := Concerns()
	if len(got) != 1 || got[0].Kind != ConcernHandlerPanic ||
		got[0].Detail != "Debouncer call panicked: boom" {
		t.Fatalf("concerns = %+v, want one handler-panic for the Debouncer call", got)
	}
}

func TestSafelyTreatsANilFnAsAPanic(t *testing.T) {
	logged := captureLog(t)
	if Safely("nil fn", nil) {
		t.Fatal("a nil fn reported a normal return")
	}
	if !strings.Contains(logged.String(), "recovered panic during nil fn") {
		t.Fatalf("log = %q", logged.String())
	}
}
