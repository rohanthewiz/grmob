package verify

import (
	"regexp"
	"strings"
	"testing"
)

// core.Select on the two natives.
//
// The dispatch arm first, for the reason textgrid_test.go gives: both natives
// end their type dispatch in a catch-all that renders the children in a plain
// column, and a picker has no children — so a missing arm draws an empty box
// with no error anywhere.

func TestSwiftTypeDispatchDrawsSelect(t *testing.T) {
	syntax := swiftSwitch.with(swiftRenderer, "struct RenderNode:", "switch node.type {")
	requireArms(t, "Renderer.swift", "RenderNode", syntax.labels(t), "Select")
}

func TestKotlinTypeDispatchDrawsSelect(t *testing.T) {
	syntax := kotlinWhen.with(kotlinRenderer, "fun RenderNodeContent(", "when (node.type) {")
	requireArms(t, "Renderer.kt", "RenderNodeContent", syntax.labels(t), "Select")
}

// Neither renderer may build the picker from a platform picker control.
//
// This is the load-bearing half of a decision made on the *web*: htmlout's
// borderResetTypes writes `border: none` for a <select> whose Go style
// declares no frame, and that reset is only correct because the other three
// targets draw nothing but what the style asks for. SwiftUI's
// .pickerStyle(.menu) and Material's ExposedDropdownMenuBox each draw a frame,
// a container colour and an indicator of their own that no Go style can
// remove — so reaching for either here would silently make the picker the one
// control whose edge comes from the platform on two targets and from the theme
// on the other two.
//
// Checked as an absence on the composite's code rather than on the file,
// because the forbidden constructs are perfectly good elsewhere: nothing else
// in either renderer uses them today, but a future date or time picker very
// well might, and this check has no business reaching into it.
//
// The positive half — that the style's own box is what draws the control — is
// what the grMobBox / boxModifier substring says. Without it the negative half
// alone would pass on a renderer that drew no box at all.
func TestNativeSelectDrawsTheStylesBoxAndNotAPlatformPicker(t *testing.T) {
	for _, pin := range []struct {
		file, marker string
		next         *regexp.Regexp
		// box is the styling primitive the arm must draw through; platform
		// names the control it must not be built from; menu is the behavior
		// construct that supplies the list.
		box, platform, menu string
	}{
		{swiftRenderer, "private struct GrMobSelect", swiftCompositeStart,
			".grMobBox(", "pickerStyle", "Menu {"},
		{kotlinRenderer, "private fun GrMobSelect", kotlinCompositeStart,
			"boxModifier(", "ExposedDropdownMenuBox", "DropdownMenu("},
	} {
		body := dispatchArm(t, pin.file, pin.marker, pin.next)
		if !strings.Contains(body, pin.box) {
			t.Errorf("%s: %s does not draw through %s — the picker's frame, fill and radius "+
				"stop coming from the Go style, and htmlout's border reset for <select> is "+
				"resting on nothing", pin.file, pin.marker, pin.box)
		}
		if strings.Contains(body, pin.platform) {
			t.Errorf("%s: %s is built from %s, which draws a frame of its own that no Go style "+
				"can remove", pin.file, pin.marker, pin.platform)
		}
		if !strings.Contains(body, pin.menu) {
			t.Errorf("%s: %s builds no %s — the option list has nowhere to appear",
				pin.file, pin.marker, pin.menu)
		}
	}
}

// The two files the decomposition now lives in, and which no other check in
// this package reads. Both are UI-free by construction — that is the whole
// point of them existing — and TestNativeMenuDecompositionIsUIFree is what
// keeps them that way.
var (
	swiftSelectMenu  = nativeFile("ios", "GrMob", "Runtime", "GrMobSelectMenu.swift")
	kotlinSelectMenu = nativeFile("android", "app", "src", "main", "java", "com", "grmob",
		"runtime", "GrMobSelectMenu.kt")
)

// The picker's menu, and what is left for a text check to hold.
//
// A SwiftUI Menu's content is a closure of views and a Compose DropdownMenu's
// is a composable lambda; neither can be read back, so three facts about the
// open menu — the runs, the headings, the refused rows — used to rest entirely
// on finding the right substrings in a 900-line renderer.
//
// They rest on a UI-free function now. grMobMenuSections (GrMobSelectMenu.swift
// / .kt) turns the flat option list into sections and rows, and each renderer's
// menu is a loop over the result. ios/verify compiles the Swift one into its
// harness and runs it against cases generated from core.SelectMenuSections, so
// the *decision* is checked by behaviour on that side. The Android half has no
// runner — the Android build's only check is compileDebugKotlin — so its
// decomposition is still held by the shape checks below plus the Go authority
// both transliterations follow.
//
// What no harness can reach on either platform is the last step: the line that
// hands a row's value to textChanged and its disabled flag to the construct
// that refuses the tap. That is what the checks in this file are now for, and
// it is a much smaller surface than the one they used to cover.

// The choice goes up as the option's *value*, through the text channel.
//
// core.Select registers a func(string), so Go put the handler in the text
// callback map. A renderer dispatching the index or the label would reach Go
// with an ID that map has an entry for and a payload that means something
// else — the label case is the dangerous one, since a picker whose labels
// happen to equal its values would work in every test anyone wrote.
//
// The Swift half is anchored on grMobMenuItems rather than on GrMobSelect: the
// buttons live in a helper because SwiftUI's Section is a container and a run
// has to be handed to it whole. The anchor follows the code rather than the
// code being kept in one place to suit the anchor.
func TestNativeSelectDispatchesTheOptionValue(t *testing.T) {
	for _, pin := range []struct {
		file, marker string
		next         *regexp.Regexp
		dispatch     string
		// value is the menu row's value as the renderer spells it. Both
		// languages read it off the same GrMobMenuItem field now, which is
		// what makes one substring able to say "not the label and not the
		// index".
		value string
	}{
		{swiftRenderer, "private func grMobMenuItems(", swiftCompositeStart,
			"textChanged(", "item.value"},
		{kotlinRenderer, "private fun GrMobSelect", kotlinCompositeStart,
			"textChanged(", "item.value"},
	} {
		body := dispatchArm(t, pin.file, pin.marker, pin.next)
		at := strings.Index(body, pin.dispatch)
		if at < 0 {
			t.Errorf("%s: %s does not call %s — a choice never reaches Go",
				pin.file, pin.marker, pin.dispatch)
			continue
		}
		// The argument that follows has to be the row's value. Read as text
		// rather than parsed: the two languages spell the call differently and
		// there is nothing shared to compare against.
		rest := body[at:]
		if end := strings.Index(rest, "\n"); end > 0 {
			rest = rest[:end]
		}
		if !strings.Contains(rest, pin.value) {
			t.Errorf("%s: %s dispatches %q, which is not the menu row's value",
				pin.file, pin.marker, strings.TrimSpace(rest))
		}
	}
}

// Each renderer draws the menu the decomposition describes, and does not
// decompose the list a second time of its own.
//
// Neither obligation is a compile-time one. A renderer that ignored the
// sections and looped over the raw options would parse cleanly, draw a flat
// list of choosable rows, and disagree with both web targets — a grouped
// picker whose headings are gone and a disabled option that can be chosen, on
// device only. That is the same shape the gap-longhand gap had, and the same
// reason it can only be caught here.
//
// Read off the composite's own code rather than the file, so a construct used
// somewhere else in a 900-line renderer cannot stand in for this one.
func TestNativeSelectDrawsTheDecomposedMenu(t *testing.T) {
	for _, pin := range []struct {
		file, marker string
		next         *regexp.Regexp
		// sections is the call into the UI-free decomposition, heading is the
		// construct a run's label is drawn with, and disabled is the read that
		// refuses a tap. All three are required: a renderer that computed the
		// sections and drew a flat list, or drew a heading it never filled,
		// would satisfy any one of them alone.
		sections, heading, disabled string
	}{
		// SwiftUI's Section is a container, so the run is handed to it whole
		// and the rows are drawn by a helper (checked separately below).
		{swiftRenderer, "private struct GrMobSelect", swiftCompositeStart,
			"grMobMenuSections(options)", "Section(section.heading)", "grMobMenuItems("},
		// Material's dropdown has no Section, so a run is announced by an
		// unclickable item written ahead of it.
		{kotlinRenderer, "private fun GrMobSelect", kotlinCompositeStart,
			"grMobMenuSections(options)", "enabled = false", "!item.isDisabled"},
	} {
		body := dispatchArm(t, pin.file, pin.marker, pin.next)
		for what, want := range map[string]string{
			"ask the decomposition for its sections": pin.sections,
			"draw a section's heading":               pin.heading,
			"honour a disabled option":               pin.disabled,
		} {
			if !strings.Contains(body, want) {
				t.Errorf("%s: %s does not %s (no %q) — the picker is grouped on the web and "+
					"flat here", pin.file, pin.marker, what, want)
			}
		}
	}
}

// The Swift rows live in a helper the check above cannot see into, and the
// disabling there is the one fact about the iOS menu that no harness reaches.
//
// It goes on the Button and never on the Section: on the Section it would take
// the whole run with it, which is a much larger wrong answer than a row that
// can be tapped.
func TestSwiftMenuItemsDisableTheButtonAndNotTheSection(t *testing.T) {
	body := dispatchArm(t, swiftRenderer, "private func grMobMenuItems(", swiftCompositeStart)
	if !strings.Contains(body, ".disabled(item.isDisabled)") {
		t.Error("Renderer.swift: grMobMenuItems does not disable a row from its own " +
			"isDisabled — a disabled option can be chosen on iOS")
	}
	if strings.Contains(body, "Section(") {
		t.Error("Renderer.swift: grMobMenuItems builds a Section — the disabling in here " +
			"would then take a whole run with it")
	}
}

// The decomposition files import no UI, on either platform.
//
// This is what makes both halves runnable at all: ios/verify compiles
// GrMobSelectMenu.swift into a plain macOS executable and android/verify
// compiles GrMobSelectMenu.kt for a JVM, and a single `import SwiftUI` or
// `import androidx.compose...` would end either — the harness would stop
// building, and the fallback would be the source-text checks this file used to
// be made of. mobile/verify/value_test.go states the same rule for
// GrMobProgress.kt, which is the second file to earn it.
//
// Stated as an absence of imports rather than of any particular symbol,
// because it is the whole dependency that matters and a new UI framework
// would be spelled a way this file could not guess.
func TestNativeMenuDecompositionIsUIFree(t *testing.T) {
	for _, pin := range []struct {
		file string
		// allowed is the one import each file may carry: Foundation supplies
		// Swift's dictionary and string types, and the Kotlin file needs
		// nothing at all.
		allowed  string
		prefixes []string
	}{
		{swiftSelectMenu, "import Foundation", []string{"import "}},
		{kotlinSelectMenu, "", []string{"import "}},
	} {
		for _, line := range strings.Split(codeIn(t, pin.file), "\n") {
			line = strings.TrimSpace(line)
			for _, prefix := range pin.prefixes {
				if !strings.HasPrefix(line, prefix) || line == pin.allowed {
					continue
				}
				t.Errorf("%s: %q — the decomposition has to stay runnable off a device, "+
					"which means importing no UI", pin.file, line)
			}
		}
	}
}

// Both transliterations exist and say the same thing.
//
// core.SelectMenuSections is the authority; each native carries a copy because
// neither can call into Go while drawing. Both copies are now *executed*
// against a table generated from that authority — ios/verify runs the Swift
// one, android/verify the Kotlin one, both over internal/menufixture — so this
// test is no longer the only thing standing between a drifting copy and a
// silent wrong menu.
//
// It is kept anyway, and deliberately: those two harnesses are shell scripts
// that need a Swift toolchain and a Kotlin compiler, and this runs under a
// bare `go test ./...`. What it pins is the handful of lines whose deletion
// changes the answer, so a copy that has been gutted fails here as well as
// there.
//
// Five facts, each one a property of core.SelectOption.Group, .Disabled or
// .GroupDisabled that a rewrite is most likely to lose.
func TestNativeMenuDecompositionsAgree(t *testing.T) {
	for _, pin := range []struct {
		file string
		// runs is the comparison that closes a run when the heading changes —
		// a gather would silently reorder the caller's list, which the field
		// says this widget must not do. flush is the append after the loop,
		// without which a run that ends the list is dropped. disabled is the
		// wire spelling, and index is the identity a row is known by.
		//
		// flush carries the `return` with it on purpose. Closing a run is one
		// named step now, called from two places — the loop's own close and
		// the end-of-loop flush — so the call alone appears twice and matches
		// a file whose flush has been deleted outright. The `return` is what
		// makes this the flush and not the other one.
		//
		// The break-test that caught this pin being too weak is the same shape
		// as the one that caught htmlout's trailing <optgroup>, which is not a
		// coincidence: an end-of-loop flush is hard to check *because* it
		// looks like the thing inside the loop.
		//
		// runDisable is the second half of closing a run, and the newer one: a
		// run disabled by any of its options has to mark the items collected
		// before that was known. See core.SelectOption.GroupDisabled.
		runs, flush, runDisable, disabled, index string
	}{
		{swiftSelectMenu, "group != heading",
			"    if building {\n        closeRun()\n    }\n    return sections",
			"runDisabled\n            ? items.map", `== "true"`, "index: i"},
		{kotlinSelectMenu, "group != heading",
			"    if (building) closeRun()\n    return sections",
			"if (runDisabled) items.map { it.copy(isDisabled = true) }",
			`== "true"`, "index = i"},
	} {
		src := valuesIn(t, pin.file)
		for what, want := range map[string]string{
			"close a run when the heading changes": pin.runs,
			"flush a run that ends the list":       pin.flush,
			"disable the items of a disabled run":  pin.runDisable,
			"read the wire's disabled spelling":    pin.disabled,
			"carry the option's index":             pin.index,
		} {
			if !strings.Contains(src, want) {
				t.Errorf("%s: does not %s (no %q)", pin.file, what, want)
			}
		}
	}
}
