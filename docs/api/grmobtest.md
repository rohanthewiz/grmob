# Package grmobtest

```go
import "github.com/rohanthewiz/grmob/grmobtest"
```

Package grmobtest drives a GrMob app in a plain go test: render it, find a control on screen, tap or type into it the way a host does, and read what is on screen afterwards.

	func TestCounter(t *testing.T) {
		app := grmobtest.New(t, App)
		app.Tap("+")
		app.Tap("+")
		if !app.HasText("Count: 2") {
			t.Fatal("two taps did not reach 2")
		}
		app.NoConcerns()
	}

## What it drives

A Driver wraps a render.Manager, the same engine every target runs, so a test needs no browser, simulator or device. Every read renders the current tree and decodes the JSON a host would receive, and every tap or keystroke dispatches the callback ID found in that tree. That is the discipline every host follows: callback IDs are assigned per pass, so an ID is always read from the tree it is about to act on, never kept from an earlier one.

The decoded tree is a \*core.Node, the type the wire format is marshalled from, so Style's fields read with their Go names and types. Props hold JSON's types after the round trip: numbers are float64, a callback is its ID string, and a nested value is a map\[string]any or a \[]any.

## Process-wide state

Some of grmob's state outlives any one app: the device records (window, lifecycle, location, heading, audio), the pending host requests (clipboard, share, keystore, notification sweeps), the permission record and the debug concern collector. A test that leaves a location fix behind would hand it to the next test's first render, which is how a suite passes in file order and fails under go test -shuffle.

New therefore calls Reset when it creates the only live Driver in the process, and the Driver's cleanup calls it again when the last one closes, so each test starts from a fresh process's state and leaves one behind. Two Drivers alive in one test share the state, as two screens in one app would. Reset is exported for tests that do not use a Driver.

One consequence: a report a host makes at boot (a window size, a fold, a colour scheme) is sent after New and before the first read, not before New, which would reset it away. That is also when a host sends it, because New renders nothing; the first read is the first render.

	app := grmobtest.New(t, App)
	core.ReceiveWindow(core.Window{Width: 841, Height: 673})
	app.Tree() // the first render reads the window just reported

Tests that use a Driver must not call t.Parallel: the state above is one per process, so two tests running at once would read each other's.

## Debug mode

NoConcerns reads the debug audit (core.SetDebugMode), which records only while debug mode is on. Turn it on once for the package, in TestMain, before any app renders:

	func TestMain(m *testing.M) {
		core.SetDebugMode(true)
		m.Run()
	}

NoConcerns fails the test when debug mode is off, rather than passing with nothing audited.

## Index

- [Variables](#variables) — `AwaitTimeout`
- [`func Button`](#func-button)
- [`func Find`](#func-find)
- [`func FindAll`](#func-findall)
- [`func HasText`](#func-hastext)
- [`func Named`](#func-named)
- [`func NoConcerns`](#func-noconcerns)
- [`func OfType`](#func-oftype)
- [`func Parse`](#func-parse)
- [`func Reset`](#func-reset)
- [`func Shows`](#func-shows)
- [`type Driver`](#type-driver)
    - [`func New`](#func-new)
    - [`func NewWithContext`](#func-newwithcontext)
    - [`func (*Driver) AwaitText`](#func-driver-awaittext)
    - [`func (*Driver) Context`](#func-driver-context)
    - [`func (*Driver) Find`](#func-driver-find)
    - [`func (*Driver) FindAll`](#func-driver-findall)
    - [`func (*Driver) HasText`](#func-driver-hastext)
    - [`func (*Driver) Manager`](#func-driver-manager)
    - [`func (*Driver) NoConcerns`](#func-driver-noconcerns)
    - [`func (*Driver) Shows`](#func-driver-shows)
    - [`func (*Driver) Tap`](#func-driver-tap)
    - [`func (*Driver) TapWhere`](#func-driver-tapwhere)
    - [`func (*Driver) Toggle`](#func-driver-toggle)
    - [`func (*Driver) Tree`](#func-driver-tree)
    - [`func (*Driver) Type`](#func-driver-type)

## Variables

AwaitTimeout is how long AwaitText waits before failing the test.

Generous because it is a ceiling, not a cost: AwaitText returns as soon as the text is on screen, so a passing test waits only as long as the app takes. Under testing/synctest the wait is on the bubble's fake clock and costs no real time at all.

```go
var AwaitTimeout = 5 * time.Second
```

<small>[grmobtest/grmobtest.go:88](https://github.com/rohanthewiz/grmob/blob/master/grmobtest/grmobtest.go#L88)</small>

## Functions

### func Button

```go
func Button(label string) func(*core.Node) bool
```

Button accepts the Button labelled label.

<small>[grmobtest/tree.go:153](https://github.com/rohanthewiz/grmob/blob/master/grmobtest/tree.go#L153)</small>

### func Find

```go
func Find(root *core.Node, pred func(*core.Node) bool) *core.Node
```

Find returns the first node under root, root included, that pred accepts, in depth-first order (a parent before its children, children in order), or nil.

Depth-first and parent-first means a predicate an ancestor also satisfies ("shows the word Inbox") finds the outermost match. A test that wants the innermost writes the predicate to say so, or uses FindAll and takes the last.

<small>[grmobtest/tree.go:18](https://github.com/rohanthewiz/grmob/blob/master/grmobtest/tree.go#L18)</small>

### func FindAll

```go
func FindAll(root *core.Node, pred func(*core.Node) bool) []*core.Node
```

FindAll returns every node under root, root included, that pred accepts, in the order Find visits them.

<small>[grmobtest/tree.go:35](https://github.com/rohanthewiz/grmob/blob/master/grmobtest/tree.go#L35)</small>

### func HasText

```go
func HasText(root *core.Node, s string) bool
```

HasText reports whether a Text node under root says exactly s.

Exact, and Text only, because it is the check that cannot pass by accident: HasText(tree, "3") is not satisfied by "13 items" or by a button labelled "3". Shows is the looser question.

<small>[grmobtest/tree.go:58](https://github.com/rohanthewiz/grmob/blob/master/grmobtest/tree.go#L58)</small>

### func Named

```go
func Named(name string) func(*core.Node) bool
```

Named accepts every node whose accessible name (core.AccessibilityLabel) is name: the words a screen reader says for it, and the most stable address a control has.

<small>[grmobtest/tree.go:167](https://github.com/rohanthewiz/grmob/blob/master/grmobtest/tree.go#L167)</small>

### func NoConcerns

```go
func NoConcerns(t testing.TB)
```

NoConcerns is Driver.NoConcerns for a test without a Driver.

<small>[grmobtest/grmobtest.go:258](https://github.com/rohanthewiz/grmob/blob/master/grmobtest/grmobtest.go#L258)</small>

### func OfType

```go
func OfType(nodeType string) func(*core.Node) bool
```

OfType accepts every node of the type ("Input", "Checkbox", "List").

<small>[grmobtest/tree.go:160](https://github.com/rohanthewiz/grmob/blob/master/grmobtest/tree.go#L160)</small>

### func Parse

```go
func Parse(t testing.TB, wire string) *core.Node
```

Parse decodes a tree as a host receives it (render.Manager.RenderInitial, mobile.RenderInitial), failing the test if it is not one.

<small>[grmobtest/grmobtest.go:271](https://github.com/rohanthewiz/grmob/blob/master/grmobtest/grmobtest.go#L271)</small>

### func Reset

```go
func Reset()
```

Reset returns grmob's process-wide state to a fresh process's: the device records and their subscriptions, every pending host request, the permission record, and the debug concern collector. The debug-mode switch itself is left alone, since TestMain sets it once for the package.

New and the last Driver's cleanup call it, so a test using a Driver need not. Call it directly between tests that drive an app some other way (the mobile bridge, a bare render.Manager), and never while an app is running: it drops the running app's subscriptions to the records it clears.

<small>[grmobtest/grmobtest.go:289](https://github.com/rohanthewiz/grmob/blob/master/grmobtest/grmobtest.go#L289)</small>

### func Shows

```go
func Shows(root *core.Node, s string) bool
```

Shows reports whether s appears, as a substring, in any word root puts on screen: a Text's content, a Button's label, a row's title, a field's placeholder or value, one line of a code block.

Every string prop is searched, because "on screen" is not one node type: a test asking whether a screen shows "Paid" should not need to know whether the word is a Text or a chip's label. Two things are left out:

  - callback IDs ("cb\_3", "txt\_cb\_r2/e0/1"), so Shows("3") cannot be satisfied by the third button's handler;
  - Style, which is a separate field on the wire, so a colour or a length is never mistaken for a word.

A code block (core.TextGrid) draws a line as several coloured runs, so a GridRow is matched on its runs joined; a match spanning two lines is not a match, as it would not be for a Text.

<small>[grmobtest/tree.go:80](https://github.com/rohanthewiz/grmob/blob/master/grmobtest/tree.go#L80)</small>

## Types

### type Driver

```go
type Driver struct {
	// contains filtered or unexported fields
}
```

Driver is one running app under test. Make one with New; it is closed by the test's cleanup.

<small>[grmobtest/grmobtest.go:97](https://github.com/rohanthewiz/grmob/blob/master/grmobtest/grmobtest.go#L97)</small>

#### func New

```go
func New(t testing.TB, root func(*core.Context) core.View) *Driver
```

New renders root in a fresh core.Context and returns a Driver for it. The app is closed when the test ends.

<small>[grmobtest/grmobtest.go:105](https://github.com/rohanthewiz/grmob/blob/master/grmobtest/grmobtest.go#L105)</small>

#### func NewWithContext

```go
func NewWithContext(t testing.TB, ctx *core.Context, root func(*core.Context) core.View) *Driver
```

NewWithContext is New with a Context the caller prepared, for an app that needs one: a theme (core.NewContext().WithTheme(theme)), or a test that keeps the Context to drive navigation or deep links itself.

<small>[grmobtest/grmobtest.go:113](https://github.com/rohanthewiz/grmob/blob/master/grmobtest/grmobtest.go#L113)</small>

#### func (*Driver) AwaitText

```go
func (d *Driver) AwaitText(s string)
```

AwaitText re-renders until s is on screen (Shows), failing the test after AwaitTimeout. It is for text a timer, an effect or a goroutine puts there.

It polls the tree rather than sleeping a fixed interval, so a test is as fast as the app and every poll is an audited render.

<small>[grmobtest/grmobtest.go:237](https://github.com/rohanthewiz/grmob/blob/master/grmobtest/grmobtest.go#L237)</small>

#### func (*Driver) Context

```go
func (d *Driver) Context() *core.Context
```

Context returns the app's Context.

<small>[grmobtest/grmobtest.go:132](https://github.com/rohanthewiz/grmob/blob/master/grmobtest/grmobtest.go#L132)</small>

#### func (*Driver) Find

```go
func (d *Driver) Find(pred func(*core.Node) bool) *core.Node
```

Find returns the first node in the current tree, in depth-first order, that pred accepts, or nil.

<small>[grmobtest/grmobtest.go:142](https://github.com/rohanthewiz/grmob/blob/master/grmobtest/grmobtest.go#L142)</small>

#### func (*Driver) FindAll

```go
func (d *Driver) FindAll(pred func(*core.Node) bool) []*core.Node
```

FindAll returns every node in the current tree that pred accepts, in depth-first order.

<small>[grmobtest/grmobtest.go:149](https://github.com/rohanthewiz/grmob/blob/master/grmobtest/grmobtest.go#L149)</small>

#### func (*Driver) HasText

```go
func (d *Driver) HasText(s string) bool
```

HasText reports whether a Text node in the current tree says exactly s.

<small>[grmobtest/grmobtest.go:155](https://github.com/rohanthewiz/grmob/blob/master/grmobtest/grmobtest.go#L155)</small>

#### func (*Driver) Manager

```go
func (d *Driver) Manager() *render.Manager
```

Manager returns the render.Manager underneath, for a dispatch the Driver has no method for (DispatchIntCallback, DispatchTextEdit, Dispatch).

<small>[grmobtest/grmobtest.go:129](https://github.com/rohanthewiz/grmob/blob/master/grmobtest/grmobtest.go#L129)</small>

#### func (*Driver) NoConcerns

```go
func (d *Driver) NoConcerns()
```

NoConcerns fails the test if the debug audit recorded anything (a hook called in a different order, a duplicate key, a handler that panicked), with the full dump. It fails too when debug mode is off, since the audit then records nothing and the check would pass vacuously.

<small>[grmobtest/grmobtest.go:252](https://github.com/rohanthewiz/grmob/blob/master/grmobtest/grmobtest.go#L252)</small>

#### func (*Driver) Shows

```go
func (d *Driver) Shows(s string) bool
```

Shows reports whether s appears anywhere in the words the current tree puts on screen. See the package-level Shows.

<small>[grmobtest/grmobtest.go:162](https://github.com/rohanthewiz/grmob/blob/master/grmobtest/grmobtest.go#L162)</small>

#### func (*Driver) Tap

```go
func (d *Driver) Tap(label string) string
```

Tap presses the Button labelled label and returns the patches the pass produced. The test fails if no such Button is on screen.

<small>[grmobtest/grmobtest.go:169](https://github.com/rohanthewiz/grmob/blob/master/grmobtest/grmobtest.go#L169)</small>

#### func (*Driver) TapWhere

```go
func (d *Driver) TapWhere(pred func(*core.Node) bool, what string) string
```

TapWhere presses the first node pred accepts that has an onClick, and returns the patches. what describes the node for the failure message ("the row titled Inbox").

Only nodes with an onClick are considered, so a predicate that also matches the node's ancestors (a row is inside a list that shows the same words) lands on the one a finger would.

<small>[grmobtest/grmobtest.go:181](https://github.com/rohanthewiz/grmob/blob/master/grmobtest/grmobtest.go#L181)</small>

#### func (*Driver) Toggle

```go
func (d *Driver) Toggle(name string, on bool) string
```

Toggle sets the Checkbox or Switch named name to on, and returns the patches. A control's name is its accessible name (core.AccessibilityLabel), which is also what a screen reader says and what comps' rows give their control; a bare checkbox with no name cannot be told apart from another one, so Toggle refuses to guess and fails the test.

<small>[grmobtest/grmobtest.go:217](https://github.com/rohanthewiz/grmob/blob/master/grmobtest/grmobtest.go#L217)</small>

#### func (*Driver) Tree

```go
func (d *Driver) Tree() *core.Node
```

Tree renders the app and returns the tree a host would receive.

<small>[grmobtest/grmobtest.go:135](https://github.com/rohanthewiz/grmob/blob/master/grmobtest/grmobtest.go#L135)</small>

#### func (*Driver) Type

```go
func (d *Driver) Type(placeholder, text string) string
```

Type delivers text to the field showing placeholder, as one change from the platform's text watcher (a paste, or the whole value after a keystroke), and returns the patches. The test fails if no field with that placeholder and an onChange is on screen.

It sets the field's whole value; it does not append to what is there, because that is what a host sends.

<small>[grmobtest/grmobtest.go:200](https://github.com/rohanthewiz/grmob/blob/master/grmobtest/grmobtest.go#L200)</small>

