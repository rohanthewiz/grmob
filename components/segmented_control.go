package components

import "github.com/rohanthewiz/grmob/core"

// SegmentedControl is a controlled single-select rendered as a row of chips —
// a filter bar, a mode switcher, a scope picker.
//
//	Row (Gap)
//	  ├─ Chip "All"     ← Selected == 0
//	  ├─ Chip "Active"
//	  └─ Chip "Done"
//
// It is the extraction of todoapp's filter bar, which was the loop the Chip
// widget itself came out of. Chip solved one segment; what stayed hand-written
// was everything around it — the row, the gap, the keying, the index
// comparison, and the per-segment accessibility label. Those are the parts
// with the quiet failure modes: forget the key and the reconciler matches
// segments by position, forget the accessibility label and a screen reader
// reads three unnamed buttons.
//
//	components.SegmentedControl{
//	    Labels:   []string{"All", "Active", "Done"},
//	    Selected: filter.Get(),
//	    OnSelect: func(i int) { filter.Set(i) },
//	}
//
// # Selection is an index, and the caller owns it
//
// The control holds no state: it renders Selected and reports taps. That is
// the same contract Chip has, one level up, and it is what lets the selected
// index be the app's own filter enum — todoapp's filterAll/filterActive/
// filterDone are literally indices into Labels.
//
// A Selected outside the range of Labels selects nothing. That is a legal
// state, not a defensive check: a scope picker that starts with no scope
// chosen says so with -1 rather than by adding a fourth "none" segment.
//
// # Segment is a template, not a set of pass-through fields
//
// Everything a Chip can do — Style, SelectedStyle, the accessibility hint —
// is set once on Segment and applies to every segment; Label, Selected and
// OnTap are filled in per segment and any value set for them on the template
// is ignored, since those three are exactly what the control is computing.
// The alternative was re-exporting Chip's surface as SegmentStyle,
// SelectedSegmentStyle, SegmentHint and so on, which grows a field every time
// Chip does. This is the move InputRow already makes with Button.
//
// # Why the accessibility label is a function
//
// It is the one thing that genuinely varies per segment and is not derivable
// from the caption: todoapp announces "Show active tasks" for a chip captioned
// "Active". A parallel []string would have to be kept in step with Labels by
// hand, so it is a function of the caption instead, and nil means "let Chip
// use the caption itself".
type SegmentedControl struct {
	// Labels are the segment captions, left to right. Selected indexes this
	// slice.
	Labels []string

	// Selected is the index of the active segment. Out of range selects
	// nothing.
	Selected int

	// OnSelect fires with the tapped segment's index.
	//
	// A nil OnSelect renders an inert control rather than one that panics:
	// core.Button registers whatever handler it is given, and a nil func in
	// the registry crashes when a native tap dispatches to it.
	OnSelect func(int)

	// Segment is the template every segment is rendered from. Its Label,
	// Selected and OnTap are overwritten per segment; everything else —
	// Style, SelectedStyle, AccessibilityHint — applies to all of them.
	Segment Chip

	// SegmentLabel derives a segment's accessibility label from its caption
	// and index. Nil leaves Chip to announce the caption itself. Chip appends
	// ", selected" to whichever name it ends up with, so this returns the
	// name only.
	SegmentLabel func(label string, index int) string

	// KeyPrefix is prepended to each segment's reconciler key, which is
	// otherwise the caption. Set it when two controls on one screen could
	// otherwise draw from the same captions — keys only have to be unique
	// among siblings, but a prefix also makes a debug-mode duplicate-key
	// concern name the control it came from.
	//
	// Captions are assumed distinct. Two segments with the same caption
	// collide, which debug mode reports rather than silently mismatching
	// rows — a segmented control with two identical captions is a bug in the
	// caller either way.
	KeyPrefix string

	// Gap is the horizontal spacing between segments, in points. Zero means
	// the theme's SM step, not zero spacing — the segments are the control's
	// own internal layout, the same reasoning InputRow's Gap carries. Ask for
	// no gap through Style: []core.StyleProp{core.Gap(0)}.
	Gap float64

	// Style is applied to the row after Gap, so it overrides it.
	Style []core.StyleProp
}

func (s SegmentedControl) Render(ctx *core.Context) *core.Node {
	gap := s.Gap
	if gap == 0 {
		gap = float64(ctx.Theme().Spacing.SM)
	}

	items := make([]core.PropsAndChildren, 0, len(s.Style)+len(s.Labels)+1)
	items = append(items, core.Gap(gap))
	// Caller Style last among the props, so it wins over the gap above.
	for _, sp := range s.Style {
		items = append(items, sp)
	}

	// Segments are appended straight into the row rather than through
	// core.For. For wraps its output in a Fragment, and while both natives
	// and (since the grouping-node fix) the HTML exporter inline that away,
	// it is still a node the reconciler walks on every pass for no gain —
	// the control already has the slice and is already building the row's
	// argument list.
	for i, label := range s.Labels {
		seg := s.Segment
		seg.Label = label
		seg.Selected = i == s.Selected

		// Capture the index, not the loop variable's identity: the closure
		// outlives this pass and must still report the segment it was built
		// for. A nil OnSelect becomes a no-op here rather than a nil in the
		// callback registry.
		idx := i
		if s.OnSelect != nil {
			seg.OnTap = func() { s.OnSelect(idx) }
		} else {
			seg.OnTap = func() {}
		}

		if s.SegmentLabel != nil {
			seg.AccessibilityLabel = s.SegmentLabel(label, i)
		}

		items = append(items, core.Keyed(s.KeyPrefix+label, seg))
	}

	return core.Row(items...).Render(ctx)
}
