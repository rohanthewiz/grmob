package verify

import (
	"strings"
	"testing"
)

// core.MaxLines on both native renderers.
//
// Like core.Rotate, the prop is easy to half-implement: a parser that reads
// the key and a Text that never passes it on compile clean in both languages,
// and neither native runs under `go test ./...`. The iOS floor is a third
// place, and the one that decides whether a capped chart label can shrink
// inside its slot at all; ios/verify's mincontent check holds its behaviour,
// this holds that the renderer's Text reads the field.

func TestBothNativeParsersReadMaxLines(t *testing.T) {
	for _, pin := range []struct{ file, key string }{
		{swiftStyle, `int("MaxLines")`},
		{kotlinStyle, `optInt("MaxLines"`},
	} {
		if src := valuesIn(t, pin.file); !strings.Contains(src, pin.key) {
			t.Errorf("%s: does not parse %s — core.MaxLines crosses the bridge and "+
				"is dropped on this target", pin.file, pin.key)
		}
	}
}

func TestBothNativeTextsApplyMaxLines(t *testing.T) {
	swift := codeIn(t, swiftRenderer)
	for _, want := range []string{".lineLimit(cap > 0 ? cap : nil)", ".truncationMode(.tail)"} {
		if !strings.Contains(swift, want) {
			t.Errorf("%s: GrMobText has no %s — the cap parses and never truncates", swiftRenderer, want)
		}
	}
	kotlin := codeIn(t, kotlinRenderer)
	for _, want := range []string{"maxLines = if (cap > 0) cap else Int.MAX_VALUE", "TextOverflow.Ellipsis"} {
		if !strings.Contains(kotlin, want) {
			t.Errorf("%s: GrMobText has no %s — the cap parses and never truncates", kotlinRenderer, want)
		}
	}
	floor := codeIn(t, nativeFile("ios", "GrMob", "Runtime", "GrMobMinContent.swift"))
	if !strings.Contains(floor, "(node.style?.maxLines ?? 0) > 0 { return 0 }") {
		t.Error("GrMobMinContent no longer floors a capped Text at 0; a capped label widens its flex slot on iOS")
	}
}
