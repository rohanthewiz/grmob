package core

import (
	"strconv"
	"strings"
	"sync"
)

// The editing surfaces' shared machinery: commands, selection, and the two
// props every host reads the same way.
//
// core.CodeEditor and core.RichTextEditor are both a *controlled text surface
// whose decoration is decided in Go and whose buffer is the host's while
// focused*. Three rules hold for both, and this file is where the two that are
// not per-node live.
//
// # 1. Commands are epoch-stamped props
//
// A toolbar's "indent this" or "make this bold" is imperative, and Go has
// exactly one channel to a host: the render tree and the patches that update
// it. So a command travels the way core.Focus's does — as props the reconciler
// diffs and the renderers react to:
//
//	RunEditorCommand(ref, EditIndent)
//	        │  epoch++, command = "indent"
//	        ▼
//	next render pass: the editor carrying EditorTarget(ref) stamps
//	        "editorEpoch"   : N       ── the command generation
//	        "editorCommand" : "indent"
//	        ▼
//	reconciler emits update-props for that editor
//	        ▼
//	host keys on editorEpoch *changing* and runs the command once, on
//	whatever the host's own selection currently is
//
// The epoch is a counter rather than a flag for the reason focus.go gives at
// length: a command has to be able to repeat. Indenting twice is two commands
// with the same string, and two identical prop maps produce no patch and
// therefore no effect. Bumping the counter is what makes the second one
// observable.
//
// # The one way this differs from focus, and why
//
// Focus state is per-app (focusState hangs off Context) because a dismiss has
// to reach every field on screen — Go was never told which one the user tapped
// into. An editor command names its editor: there is no "indent whichever
// editor has the cursor", because the toolbar that issues it is *next to* the
// editor it belongs to. So the epoch lives on the ref itself, and exactly one
// node in the tree stamps anything. Two editors on one screen therefore cost
// two refs and no cross-talk, where two fields share one focus epoch by
// construction.
//
// # 2. Selection rides the text channel
//
// The bridge has four callback channels — void, bool, int, text — and a
// selection is two ints. Rather than grow the bridge, a code editor reports
// "start:end" on the text channel and this file parses it before the app's
// func(start, end int) is called, exactly as NumericInput parses its string.
// The offsets are byte offsets into the UTF-8 value *on the wire*; each host
// converts from its own unit (UTF-16 on Android and the DOM, String.Index on
// iOS), so app code never sees a platform's idea of a character.

// EditorRef names one editing surface so that a toolbar can send it commands.
//
// It is the editor half of FocusRef and is used the same way: a hook makes one
// that is stable across render passes, EditorTarget puts it on the editor, and
// RunEditorCommand sends to it.
//
//	ref := core.UseEditorRef(ctx)
//
//	core.CodeEditor(src, onChange, rows, core.EditorTarget(ref))
//	core.Button("Indent", func() { core.RunEditorCommand(ref, core.EditIndent) })
//
// Unlike FocusRef it carries its own command state rather than pointing at the
// app's — see the file doc for why one editor's commands are nobody else's.
type EditorRef struct {
	ctx *Context

	// Guarded because a command may be issued from a timer or a network
	// callback while a render pass is reading the stamp.
	mu      sync.Mutex
	epoch   int
	command string
}

// UseEditorRef returns an EditorRef that is stable for the lifetime of this
// hook slot, which is what makes the ref usable as an identity.
//
// A hook rather than a bare constructor for exactly FocusRef's reason: a ref
// built inline in a render function is a new pointer every pass, so
// EditorTarget would stamp one identity and the toolbar's handler would bump
// another — and the command would silently never reach a node. NewState both
// pins the pointer and reserves the cursor slot properly.
//
// A widget that calls this consumes a positional hook slot on the caller's
// context and must therefore be rendered unconditionally, like any other hook
// user; comps.CodeEditor says so in its own doc.
func UseEditorRef(ctx *Context) *EditorRef {
	// NewState keeps only the first value handed to it, so the ref allocated
	// on later passes is discarded and Get returns the original pointer. The
	// slot goes through a variable because State's accessors have pointer
	// receivers and NewState's return value is not addressable.
	slot := NewState(ctx, &EditorRef{ctx: ctx})
	return slot.Get()
}

// EditorTarget marks the editing surface it is applied to as ref's.
//
// A nil ref returns a nil prop rather than panicking — leafNode skips a nil
// item — so `core.EditorTarget(maybeRef)` degrades to an editor no toolbar can
// command instead of crashing a render pass.
//
// Applying it to anything but a CodeEditor or a RichTextEditor is harmless and
// pointless: the stamp lands and no renderer reads it.
func EditorTarget(ref *EditorRef) BehaviorProp {
	if ref == nil {
		return nil
	}
	return behaviorFunc(func(ctx *Context, n *Node) {
		if n.Props == nil {
			n.Props = map[string]any{}
		}
		stampEditor(n.Props, ref)
	})
}

// stampEditor writes the ref's current command onto a props map, or writes
// nothing when no command has ever been issued.
//
// Both keys are always written together, never one alone — the same rule
// stampFocus states, for the same reason: an update-props patch carries the
// *whole new props map* and the renderers iterate the keys it contains, so a
// key that disappears between passes is invisible on the far side.
//
// Epoch 0 is the sentinel for "nothing has ever been issued", so an editor
// whose toolbar has never been touched renders byte-identical trees to one
// with no ref at all. Each host checks the zero as well, because both props
// always travel together and a 0 must never be read as an instruction.
func stampEditor(props map[string]any, ref *EditorRef) {
	ref.mu.Lock()
	epoch, command := ref.epoch, ref.command
	ref.mu.Unlock()
	if epoch == 0 {
		return
	}
	props["editorEpoch"] = epoch
	props["editorCommand"] = command
}

// RunEditorCommand sends one command to ref's editor, to be applied to
// whatever the host's own selection is at the time it lands.
//
// Called from an event handler, as a toolbar button's whole body:
//
//	core.Button("Bold", func() { core.RunEditorCommand(ref, core.EditBold) })
//
// The command strings each editor understands are its own; see the Edit*
// constants below for the census. A command an editor does not know is
// deliberately a no-op on every host rather than an error — the alternative is
// a screen that crashes because a toolbar outgrew its editor.
//
// Issuing a command for a ref whose editor is not currently in the tree does
// nothing visible: no node stamps it, so no host acts. The command still
// consumes an epoch, which is correct — it happened, it simply had no target
// on screen.
//
// A nil ref is a no-op rather than a panic, matching EditorTarget.
func RunEditorCommand(ref *EditorRef, command string) {
	if ref == nil || ref.ctx == nil {
		return
	}
	ref.mu.Lock()
	ref.epoch++
	ref.command = command
	ref.mu.Unlock()

	// RequestRender rather than MarkDirty, for core.Focus's reason: a command
	// issued from a timer or a network callback has no bridge call pending to
	// carry the patches back, so it needs the push channel nudged. From an
	// event handler the nudge is redundant but harmless.
	ref.ctx.RequestRender()
}

// The commands a core.CodeEditor understands. Spelled as constants so a
// toolbar and a renderer cannot disagree about a string literal, and so the
// census of what v1 supports is one list rather than four.
//
// Each acts on the host's current selection, or on the line the caret is in
// when the selection is empty — which is what makes "indent" useful without a
// selection, and is the behavior every code editor has.
const (
	// EditIndent inserts one indent at the start of every line the selection
	// touches. One indent is tabSize spaces, or a tab when tabSize is 0.
	EditIndent = "indent"
	// EditOutdent removes one indent's worth of leading white space from every
	// line the selection touches, and leaves a line that has none alone.
	EditOutdent = "outdent"
	// EditCommentLine toggles the commentPrefix on every line the selection
	// touches: it comments them all when any is uncommented, and uncomments
	// them when every one is already commented. The toggle is decided for the
	// whole run rather than per line, so a partially-commented block becomes
	// fully commented rather than inverting line by line.
	EditCommentLine = "commentLine"
	// EditSelectAll selects the whole buffer.
	EditSelectAll = "selectAll"
)

// parseSelection splits the "start:end" the hosts report into two byte
// offsets.
//
// ok is false for anything that is not two parseable integers, which is what
// keeps a malformed report (a host mid-refactor, a fuzzed bridge message) from
// calling app code with a zero range that looks like a real deselection. A
// negative or reversed pair is normalized rather than refused: a drag upward
// is reported end-first by some hosts and is a perfectly ordinary selection.
func parseSelection(payload string) (start, end int, ok bool) {
	colon := strings.IndexByte(payload, ':')
	if colon < 0 {
		return 0, 0, false
	}
	start, err := strconv.Atoi(payload[:colon])
	if err != nil {
		return 0, 0, false
	}
	end, err = strconv.Atoi(payload[colon+1:])
	if err != nil {
		return 0, 0, false
	}
	if start < 0 || end < 0 {
		return 0, 0, false
	}
	if start > end {
		start, end = end, start
	}
	return start, end, true
}

// OnSelectionChange reports the editing surface's selection as it moves.
//
// The handler receives byte offsets into the UTF-8 value, half-open as every
// range in Go is: start == end is a caret with nothing selected.
//
//	core.CodeEditor(src, onChange, rows,
//	    core.OnSelectionChange(func(start, end int) { status.Set(start, end) }),
//	)
//
// It rides the text channel carrying "start:end" rather than needing a fifth
// bridge channel; see this file's doc. A payload that is not two integers is
// dropped rather than delivered as a zero range.
//
// Applied to a node that is not an editing surface it is inert, like every
// other behavior prop on a node type that does not read it.
func OnSelectionChange(handler func(start, end int)) BehaviorProp {
	if handler == nil {
		return nil
	}
	return behaviorFunc(func(ctx *Context, n *Node) {
		if n.Props == nil {
			n.Props = map[string]any{}
		}
		n.Props["onSelectionChange"] = ctx.registerTextCallback(func(payload string) {
			if start, end, ok := parseSelection(payload); ok {
				handler(start, end)
			}
		})
	})
}

// ReadOnly makes an editing surface show a caret and allow selection while
// refusing every edit.
//
// It is deliberately not core.Disabled. A disabled control is inert and greyed
// and is skipped by assistive technology's traversal; a read-only code block is
// *content* — the user is meant to read it, select it and copy out of it — and
// on every platform that is a different state with a different look. The two
// are not interchangeable and a widget that wants the greyed-out reading can
// still ask for it.
func ReadOnly() BehaviorProp {
	return behaviorFunc(func(ctx *Context, n *Node) {
		if n.Props == nil {
			n.Props = map[string]any{}
		}
		n.Props["readOnly"] = true
	})
}

// Placeholder is the prompt an empty editing surface shows.
//
// core.Input and core.TextArea take theirs positionally, because they had one
// before the mixed argument list existed; the editors take it as a prop, which
// is the shape every option added since uses.
func Placeholder(text string) BehaviorProp {
	return behaviorFunc(func(ctx *Context, n *Node) {
		if n.Props == nil {
			n.Props = map[string]any{}
		}
		n.Props["placeholder"] = text
	})
}
