package apidoc

import (
	"fmt"
	"go/ast"
	"go/build"
	"go/doc"
	"go/parser"
	"go/token"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
)

// ModulePath is grmob's module path. Import paths on the generated pages are
// built from it, and repoRoot uses it to tell the module root apart from the
// nested module under cmd/docs.
const ModulePath = "github.com/rohanthewiz/grmob"

// SourceBaseURL is where a "declared in" line points. It is pinned to a branch
// rather than a tag or a commit on purpose: the pages are regenerated from the
// working tree and committed alongside it, so a line number is only ever
// correct for the same revision the pages were built from, and master is the
// revision a reader arriving from the docs site is looking at.
const SourceBaseURL = "https://github.com/rohanthewiz/grmob/blob/master/"

// Pkg is one documented package: where its source lives, what the nav calls
// it, and which group it sits in on the overview page.
type Pkg struct {
	// Dir is the package directory relative to the module root ("core", or a
	// nested one such as "a/b"). It is also the key everything else is derived from: the
	// import path, the page filename, the nav entry.
	Dir string

	// Group is the overview page's section heading. Packages with the same
	// Group are listed together, in the order they appear in Packages.
	Group string

	// Blurb is one line of editorial orientation shown on the overview page
	// next to the package name. The package's own doc comment is the authority
	// on what the package *is*; this says when you reach for it, which is the
	// question an index answers and a doc comment does not.
	Blurb string

	// Topics, when set, splits the package's reference across several pages:
	// the package page keeps the package comment and becomes an index of the
	// topic pages, and each topic gets a sibling page of its own. Left empty,
	// the package is one page, which is right for every package whose page a
	// reader can still scroll. See Topic.
	Topics []Topic
}

// Topic is one page of a split package: the declarations of a named set of
// source files.
//
// # Why by source file
//
// A page has to be decided per declaration, and the only grouping a
// declaration carries without anyone maintaining it is the file it lives in.
// grmob's files are already cut along topic lines (layout.go, theme.go,
// audio.go), so a table of files is short, reads as a table of contents, and
// changes only when a file is added — which is exactly when someone should
// decide where its declarations are documented. The alternatives go stale in
// the way that matters for a generated reference: a table of symbol names
// needs an edit for every new function, and a prefix rule ("Audio*") files
// OnAudioStatus under O.
//
// Every type, function and const or var block is placed by its own file, with
// one exception: go/doc attaches methods and typed constants to their type, and
// they follow the type onto its page even when declared elsewhere —
// (*Context).OnClose, written in cleanup.go, is documented beside Context. A
// constructor is not an exception: go/doc also attaches those to the type they
// return, and splitTopics moves each back to its own file's page, because in
// core nearly everything returns View.
//
// # What keeps the table honest
//
// splitTopics fails the generator, and therefore `go test ./internal/apidoc`,
// when a file declares something documented and no topic lists it (its
// declarations would be on no page), when a listed file is not a source file of
// the package (a rename left the table behind), and when two topics list the
// same file. A file with nothing exported may be left out.
type Topic struct {
	// Slug names the page: Pkg.Page() without ".md", a dash, then Slug
	// ("core-layout.md"). The page stays a flat sibling of every other page,
	// so cross-references remain bare relative links.
	Slug string

	// Title is the topic page's heading suffix and its nav entry's text.
	Title string

	// Blurb is one line on what the topic covers, shown in the package page's
	// topic table and under the topic page's title.
	Blurb string

	// Files are base names within the package directory.
	Files []string
}

// TopicPage is a topic's generated file name under docs/api/.
func (p Pkg) TopicPage(t Topic) string {
	return strings.TrimSuffix(p.Page(), ".md") + "-" + t.Slug + ".md"
}

// ImportPath is the path a caller writes in an import statement.
func (p Pkg) ImportPath() string { return ModulePath + "/" + p.Dir }

// Name is the package's clause name, which for every package here is the last
// element of its directory. Nothing in grmob's public surface renames a package
// away from its directory, and the loader verifies that rather than trusting it
// (see load).
func (p Pkg) Name() string { return path.Base(p.Dir) }

// Page is the generated file's name under docs/api/. Nested directories are
// flattened with a dash ("a/b" -> "a-b.md") so every page is a
// sibling of every other one, which is what lets an in-page cross-reference be
// a bare relative link with no "../" to get wrong.
func (p Pkg) Page() string { return strings.ReplaceAll(p.Dir, "/", "-") + ".md" }

// Packages is the documented public surface, in nav order.
//
// It is a hand-kept list rather than a walk of the tree, because "every package
// that compiles" is the wrong set and the difference is not mechanical. Left out
// deliberately:
//
//	internal/...        not importable, by construction
//	examples/...        programs, and the tutorial's lessons are prose already
//	*/verify, aria/gen  test harnesses and generators — developer tooling whose
//	                    audience reads the source, not a reference page
//	aria/...            importable, but spec-parsing machinery that feeds the
//	                    accessibility fixture rather than framework API; no app
//	                    imports it, so a page for it read as something to learn
//	serve, wasm         package main
//
// TestPackagesCoversEveryPublicPackage holds the list to that rule: it walks
// the tree and fails if an importable non-main package appears that is neither
// listed here nor matched by one of the exclusions above. A new public package
// therefore cannot be added without either documenting it or saying, in that
// test's terms, why not.
var Packages = []Pkg{
	{
		Dir:   "core",
		Group: "Core",
		Blurb: "Views, nodes, state, styling, events — everything an app builds its UI out of.",
		// core was the first package split into topics: as a single page it ran
		// to ~7,800 lines, past the point where its sidebar TOC (every type
		// and function in the package, in alphabetical order) helps anyone
		// find anything. The order below follows the narrative docs' order of
		// concepts, not the alphabet.
		Topics: []Topic{{
			Slug:  "views",
			Title: "Views & state",
			Blurb: "View, Node, Context and state slots; text and inline runs; conditionals, caching, error boundaries and debug-mode concerns.",
			Files: []string{"view.go", "node.go", "text.go", "paragraph.go", "context.go", "cleanup.go", "cached.go",
				"conditionals.go", "error_boundary.go", "render_manager.go", "debug.go"},
		}, {
			Slug:  "layout",
			Title: "Layout",
			Blurb: "Rows, columns, stacks, scrolls and lists, and the alignment vocabulary they are placed with.",
			Files: []string{"layout.go", "list.go", "list_start.go", "stack_align.go", "alignment.go", "keyboard.go", "placement_audit.go"},
		}, {
			// Styling is two pages, split along the line style.go and
			// style_props.go already draw: the Style struct and the enums its
			// fields take, then the StyleProp constructors a view is written
			// with. As one page it ran to ~1,900 lines, half of it the struct's
			// field docs, so a reader after Padding scrolled past all of them.
			// The first keeps the "style" slug so existing links to
			// core-style.md still land on the Style type.
			Slug:  "style",
			Title: "Styling: the Style struct",
			Blurb: "Style and the value types its fields take: alignment, flex, position, weights and edge insets.",
			Files: []string{"style.go"},
		}, {
			Slug:  "style-props",
			Title: "Styling: style props",
			Blurb: "The StyleProp constructors: spacing and per-side insets, flex, typography, colour, borders, per-corner radii and animation.",
			Files: []string{"style_props.go", "margin_sides.go", "padding_sides.go", "corners.go", "animation.go"},
		}, {
			Slug:  "theme",
			Title: "Theming",
			Blurb: "Themes, palettes, typography and spacing scales, and per-component defaults.",
			Files: []string{"theme.go"},
		}, {
			Slug:  "controls",
			Title: "Controls",
			Blurb: "Buttons, text inputs, switches, sliders, selects, images, tab views, text grids and vector canvases.",
			Files: []string{"button.go", "input.go", "keyboard_kind.go", "switch.go", "slider.go", "select_menu.go", "image.go",
				"tabview.go", "textgrid.go", "canvas.go"},
		}, {
			Slug:  "editors",
			Title: "Editors",
			Blurb: "The code editor and the rich text editor, and the refs and commands that drive them.",
			Files: []string{"codeeditor.go", "editor.go", "richtext.go"},
		}, {
			Slug:  "events",
			Title: "Events & focus",
			Blurb: "Event props, host and system events, focus refs and focus order, and scrolling a node into view.",
			Files: []string{"event.go", "behavioral_props.go", "host_events.go", "sys_events.go", "focus.go",
				"focus_order.go", "scroll_to.go"},
		}, {
			Slug:  "navigation",
			Title: "Navigation & overlays",
			Blurb: "The navigator stack, modals, toasts, deep links and opening URLs.",
			Files: []string{"navigation.go", "modal.go", "toast.go", "deeplink.go", "openurl.go"},
		}, {
			Slug:  "accessibility",
			Title: "Accessibility",
			Blurb: "Roles, selected, expanded and current states, value ranges and the accessibility audit.",
			Files: []string{"role.go", "popup.go", "selected.go", "expanded.go", "current.go", "value.go", "a11y_audit.go"},
		}, {
			Slug:  "device",
			Title: "Device services",
			Blurb: "Audio, camera, clipboard, haptics, local notifications, compass heading, location, maps, the app lifecycle, and the window's size and fold.",
			Files: []string{"audio.go", "camera.go", "clipboard.go", "haptics.go", "notifications.go", "heading.go", "location.go", "mapview.go", "lifecycle.go", "window.go"},
		}},
	},
	{
		Dir:   "hooks",
		Group: "Core",
		Blurb: "Effects, timers, memos, reducers and the other hooks layered on core's state slots.",
	},
	{
		Dir:   "render",
		Group: "Rendering",
		Blurb: "The render loop: mount a root view, dispatch host events, drain pushed updates.",
	},
	{
		Dir:   "reconcile",
		Group: "Rendering",
		Blurb: "The tree diff and the patch vocabulary every host applies.",
	},
	{
		Dir:   "comps",
		Group: "Widgets",
		Blurb: "The widget library — cards, tabs, accordions and friends, built on the public core API.",
		// comps is split for core's reason: as one page it ran to ~6,000
		// lines. The topics are the questions a screen's author asks in
		// order — what frames the screen, what goes in its lists, how a value
		// is typed or picked, what a tap does, what floats over it, what
		// shows data — rather than the alphabet, which puts AppBar beside
		// Avatar. doc.go is left out: it declares nothing.
		Topics: []Topic{{
			Slug:  "structure",
			Title: "Screens & structure",
			Blurb: "Screen, app and bottom bars, the FAB, tabs, drawers, step indicators, two-pane and foldable layouts, cards, accordions, headings, breadcrumbs and separators, labelled or not.",
			Files: []string{"screen.go", "app_bar.go", "bottom_bar.go", "fab.go", "tabs.go", "drawer.go", "step_indicator.go",
				"two_pane.go", "card.go", "accordion.go", "disclosure.go", "heading.go", "breadcrumb.go", "separator.go",
				"labeled_separator.go"},
		}, {
			Slug:  "lists",
			Title: "Lists & tables",
			Blurb: "List rows, the settings-row family (switch, checkbox, select and slider), input rows, key-value lists, bullet lists, grouped and paged lists, data tables and timelines.",
			Files: []string{"list_row.go", "settings_row.go", "select_row.go", "slider_row.go", "input_row.go",
				"key_value_list.go", "bullet_list.go", "grouped_list.go", "grouping.go", "paging.go", "data_table.go", "timeline.go"},
		}, {
			Slug:  "inputs",
			Title: "Inputs & pickers",
			Blurb: "Form fields, password fields, one-time code fields, tag inputs, search, searchable selects, radio groups, dates, date ranges, times and calendars, and the two editors.",
			Files: []string{"form_field.go", "password_field.go", "pin_input.go", "tag_input.go", "search_field.go", "searchable_select.go", "radio_group.go",
				"date_picker.go", "date_range_picker.go", "time_picker.go", "calendar.go", "code_editor.go", "rich_text_editor.go", "rich_text_view.go"},
		}, {
			Slug:  "actions",
			Title: "Buttons & choices",
			Blurb: "Buttons and their variants, copy buttons, links, chips, segmented controls, steppers, ratings and badges.",
			Files: []string{"button.go", "variant.go", "copy_button.go", "link.go", "chip.go", "chip_strip.go", "segmented_control.go",
				"stepper.go", "rating.go", "badge.go"},
		}, {
			Slug:  "overlays",
			Title: "Overlays & feedback",
			Blurb: "Dialogs, lightboxes, action sheets, menus, snackbars, banners, progress, spinners, skeletons and empty states.",
			Files: []string{"dialog.go", "lightbox.go", "action_sheet.go", "menu.go", "snackbar.go", "banner.go",
				"progress_bar.go", "spinner.go", "skeleton.go", "empty_state.go"},
		}, {
			Slug:  "display",
			Title: "Data display & maps",
			Blurb: "Avatars and avatar stacks, stat tiles, the compass, clocks, countdowns and alarms, an audio player, message bubbles and threads, typing indicators, reaction bars, polls, expandable text, QR codes, map panels and static maps.",
			Files: []string{"avatar.go", "avatar_stack.go", "stat_tile.go", "compass.go", "clock.go", "timers.go", "alarm.go", "audio_player.go", "message_bubble.go", "message_thread.go", "typing_indicator.go", "reaction_bar.go", "poll.go", "expandable_text.go", "qr_code.go",
				"map_panel.go", "static_map.go"},
		}, {
			Slug:  "charts",
			Title: "Charts",
			Blurb: "Sparklines, line, area, bar and scatter charts, histograms, heatmaps and calendar heatmaps, donuts and pies, and gauges, drawn on core.Canvas.",
			Files: []string{"chart.go", "sparkline.go", "line_chart.go", "bar_chart.go", "histogram.go", "scatter_chart.go",
				"heatmap.go", "donut_chart.go", "gauge.go"},
		}},
	},
	{
		Dir:   "alarm",
		Group: "Widgets",
		Blurb: "Alarm clock arithmetic: when an alarm next rings, and whether it fell due between two checks.",
	},
	{
		Dir:   "forms",
		Group: "Widgets",
		Blurb: "Validation rules, the form hook that owns values and error visibility, and bound inputs.",
	},
	{
		Dir:   "richtext",
		Group: "Widgets",
		Blurb: "The document model behind core.RichTextEditor: a formatted document as Go values.",
	},
	{
		Dir:   "highlight",
		Group: "Widgets",
		Blurb: "Go syntax highlighting, for the code editor and the tutorial's listings.",
	},
	{
		Dir:   "htmlout",
		Group: "Exporters",
		Blurb: "Render a node tree to a standalone HTML document.",
	},
	{
		Dir:   "jsonout",
		Group: "Exporters",
		Blurb: "Render a node tree to JSON, for host renderers and for inspecting a tree in a test.",
	},
	{
		Dir:   "mobile",
		Group: "Platform",
		Blurb: "The gomobile-bindable bridge the Android and iOS shells call into.",
	},
	{
		Dir:   "permission",
		Group: "Platform",
		Blurb: "Asking the platform for the camera, the microphone, location, the media store.",
	},
}

// Loaded is one package's parsed documentation, plus the fileset its positions
// are relative to.
type Loaded struct {
	Pkg  Pkg
	Doc  *doc.Package
	FSet *token.FileSet

	// page is the generated page this documentation renders onto: Pkg.Page()
	// for a whole package, a topic page for one of splitTopics' parts. A doc
	// link compares it with its target's page to decide whether a bare
	// "#anchor" is enough.
	page string

	// files are the base names of the source files Doc was built from, which
	// splitTopics checks a Topic's file list against.
	files []string
}

// load parses one package and builds its documentation.
//
// The file list comes from go/build rather than from a directory listing so
// that build constraints and the _test.go suffix are honoured by the same rules
// the compiler uses. It resolves against build.Default, i.e. the host GOOS and
// GOARCH, which is exact for grmob because no package in Packages has a single
// constrained file — the only //go:build lines in the tree are in wasm's
// package main and in tests. Should that ever change, the omission would be
// silent here, so TestNoBuildConstraintsInDocumentedPackages fails on the first
// constrained file to appear in a documented package rather than letting a page
// quietly lose a declaration.
func load(root string, p Pkg) (*Loaded, error) {
	dir := filepath.Join(root, filepath.FromSlash(p.Dir))

	bp, err := build.ImportDir(dir, 0)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", p.Dir, err)
	}
	if bp.Name != p.Name() {
		// The generator derives the page title and every "package x" mention
		// from the directory; a package whose clause disagrees would be
		// documented under a name no import statement produces.
		return nil, fmt.Errorf("%s: package clause is %q, want %q",
			p.Dir, bp.Name, p.Name())
	}

	fset := token.NewFileSet()
	files := make([]*ast.File, 0, len(bp.GoFiles))
	for _, name := range bp.GoFiles {
		f, err := parser.ParseFile(fset, filepath.Join(dir, name), nil, parser.ParseComments)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", p.Dir, err)
		}
		files = append(files, f)
	}

	// Default mode: unexported declarations are dropped, which is the whole
	// point — this is a reference for callers. doc.NewFromFiles also does the
	// grouping a reader expects, attaching each constructor and method to the
	// type it belongs to instead of listing it among the package functions.
	dp, err := doc.NewFromFiles(fset, files, p.ImportPath())
	if err != nil {
		return nil, fmt.Errorf("%s: %w", p.Dir, err)
	}

	return &Loaded{Pkg: p, Doc: dp, FSet: fset, page: p.Page(), files: bp.GoFiles}, nil
}

// splitTopics partitions a package's documentation into one part per Topic,
// in Topics order. A package with no Topics yields no parts.
//
// Each part is a shallow copy of the whole package's doc.Package with the four
// declaration lists filtered and Doc (the package comment) cleared — the
// comment belongs on the package page, once. Copying rather than building a
// fresh doc.Package matters: its Parser and Printer read unexported state (the
// package's full symbol set and import names), so a part still recognises
// "[Node]" written in any file of the package, whichever page Node landed on.
//
//	core (whole)                        parts
//	────────────                        ─────
//	Consts  Vars  Funcs  Types   ──▶    core-views.md   decls in view.go, node.go, …
//	  │       │     │      │            core-layout.md  decls in layout.go, list.go, …
//	  └───────┴─────┴──────┴── placed by the file of each declaration's position
func splitTopics(l *Loaded) ([]*Loaded, error) {
	p := l.Pkg
	if len(p.Topics) == 0 {
		return nil, nil
	}

	present := map[string]bool{}
	for _, f := range l.files {
		present[f] = true
	}
	owner := map[string]int{} // base file name -> index into p.Topics
	for i, t := range p.Topics {
		for _, f := range t.Files {
			if j, dup := owner[f]; dup {
				return nil, fmt.Errorf("%s: %s is listed in both topic %q and topic %q",
					p.Dir, f, p.Topics[j].Slug, t.Slug)
			}
			if !present[f] {
				return nil, fmt.Errorf("%s: topic %q lists %s, which is not a source file of the package "+
					"(renamed or deleted? update apidoc.Packages)", p.Dir, t.Slug, f)
			}
			owner[f] = i
		}
	}

	parts := make([]*Loaded, len(p.Topics))
	for i, t := range p.Topics {
		d := *l.Doc
		d.Doc = ""
		d.Consts, d.Vars, d.Funcs, d.Types = nil, nil, nil, nil
		parts[i] = &Loaded{Pkg: p, Doc: &d, FSet: l.FSet, page: p.TopicPage(t), files: t.Files}
	}

	// topicOf maps a declaration to its part. It is also where an unassigned
	// file is caught: at the first declaration that would otherwise be dropped,
	// which is what the error can then name.
	topicOf := func(pos token.Pos, what string) (*doc.Package, error) {
		file := filepath.Base(l.FSet.Position(pos).Filename)
		i, ok := owner[file]
		if !ok {
			return nil, fmt.Errorf("%s: %s declares %s but no topic lists the file, "+
				"so it would be on no page — add it to a Topic in apidoc.Packages", p.Dir, file, what)
		}
		return parts[i].Doc, nil
	}

	// go/doc has already sorted each list by name, and appending in that order
	// keeps every part sorted too.
	for _, v := range l.Doc.Consts {
		d, err := topicOf(v.Decl.Pos(), "constants "+strings.Join(v.Names, ", "))
		if err != nil {
			return nil, err
		}
		d.Consts = append(d.Consts, v)
	}
	for _, v := range l.Doc.Vars {
		d, err := topicOf(v.Decl.Pos(), "variables "+strings.Join(v.Names, ", "))
		if err != nil {
			return nil, err
		}
		d.Vars = append(d.Vars, v)
	}
	for _, fn := range l.Doc.Funcs {
		d, err := topicOf(fn.Decl.Pos(), "func "+fn.Name)
		if err != nil {
			return nil, err
		}
		d.Funcs = append(d.Funcs, fn)
	}
	for _, t := range l.Doc.Types {
		d, err := topicOf(t.Decl.Pos(), "type "+t.Name)
		if err != nil {
			return nil, err
		}
		// Constructors are placed by their own file, not their type's. go/doc
		// calls any function returning T a constructor of T, and in core most
		// of the package returns View or *Node: left under the type, Row,
		// Button and Image would all be documented on the views page, and the
		// layout page would be missing the declarations layout.go is for. A
		// copy of the type carries the constructors that stay, so the whole
		// package's doc.Type is not edited under the rest of the run.
		c := *t
		c.Funcs = nil
		for _, fn := range t.Funcs {
			fd, err := topicOf(fn.Decl.Pos(), "func "+fn.Name)
			if err != nil {
				return nil, err
			}
			if fd == d {
				c.Funcs = append(c.Funcs, fn)
			} else {
				fd.Funcs = append(fd.Funcs, fn)
			}
		}
		d.Types = append(d.Types, &c)
	}

	// A moved constructor was appended after its part's package-level
	// functions; restore go/doc's by-name order.
	for _, part := range parts {
		sort.Slice(part.Doc.Funcs, func(i, j int) bool { return part.Doc.Funcs[i].Name < part.Doc.Funcs[j].Name })
	}
	return parts, nil
}

// LoadAll parses every documented package.
func LoadAll(root string) ([]*Loaded, error) {
	out := make([]*Loaded, 0, len(Packages))
	for _, p := range Packages {
		l, err := load(root, p)
		if err != nil {
			return nil, err
		}
		out = append(out, l)
	}
	return out, nil
}

// RepoRoot walks up from the working directory to grmob's module root.
//
// It cannot stop at the first go.mod it finds, the way a single-module
// repository's generator can: cmd/docs is a module of its own (so that the docs
// server's dependencies stay out of the framework's go.mod), and a run started
// from inside it would otherwise take that directory for the root. So the
// module line has to match, and the walk continues past a go.mod that belongs to
// somebody else.
func RepoRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	start := dir
	for {
		if isModuleRoot(filepath.Join(dir, "go.mod")) {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("no go.mod declaring %s above %s — "+
				"run this from inside the repository", ModulePath, start)
		}
		dir = parent
	}
}

func isModuleRoot(goMod string) bool {
	data, err := os.ReadFile(goMod)
	if err != nil {
		return false
	}
	for line := range strings.SplitSeq(string(data), "\n") {
		if strings.TrimSpace(line) == "module "+ModulePath {
			return true
		}
	}
	return false
}

// symbolIndex maps an import path to the anchors of that package's exported
// symbols, keyed by the name a doc link would use ("Node", "Node.Clone").
//
// It exists because a doc link does not say what kind of thing it points at.
// `[Node]` arrives as a bare name, and the anchor for a type is "type-node"
// while the anchor for a function of the same name is "func-node" — so
// resolving a link needs to know which one the target package actually declares.
// Building the index over every loaded package first, then rendering, is what
// makes a link from core's doc comment into components resolvable at all.
//
// It is built from the units that render onto pages — a whole package, or each
// topic part of a split one — so the page travels with the anchor: a link to
// core's Row has to say core-layout.md, not core.md. Anchors are unique within
// a package, so a split package's parts never contend for a key.
type symbolIndex map[string]map[string]symLoc

// symLoc is where a symbol is documented: its page under docs/api/ and the
// anchor on that page.
type symLoc struct {
	Page   string
	Anchor string
}

func newSymbolIndex(units []*Loaded) symbolIndex {
	idx := symbolIndex{}
	for _, l := range units {
		syms := idx[l.Pkg.ImportPath()]
		if syms == nil {
			syms = map[string]symLoc{}
			idx[l.Pkg.ImportPath()] = syms
		}
		at := func(anchor string) symLoc { return symLoc{Page: l.page, Anchor: anchor} }
		for _, f := range l.Doc.Funcs {
			syms[f.Name] = at(symAnchor("func", "", f.Name))
		}
		for _, t := range l.Doc.Types {
			syms[t.Name] = at(symAnchor("type", "", t.Name))
			for _, f := range t.Funcs { // constructors: documented under the type
				syms[f.Name] = at(symAnchor("func", "", f.Name))
			}
			for _, m := range t.Methods {
				syms[t.Name+"."+m.Name] = at(symAnchor("func", t.Name, m.Name))
			}
		}
		// Constants and variables have no heading of their own; they link to
		// the section on whichever page their block landed on.
		for _, v := range l.Doc.Consts {
			for _, name := range v.Names {
				if ast.IsExported(name) {
					syms[name] = at("constants")
				}
			}
		}
		for _, v := range l.Doc.Vars {
			for _, name := range v.Names {
				if ast.IsExported(name) {
					syms[name] = at("variables")
				}
			}
		}
	}
	return idx
}

// sortedGroups returns the overview page's group headings in the order their
// first package appears in Packages, so the nav and the overview agree.
func sortedGroups() []string {
	var groups []string
	seen := map[string]bool{}
	for _, p := range Packages {
		if !seen[p.Group] {
			seen[p.Group] = true
			groups = append(groups, p.Group)
		}
	}
	return groups
}

// stableNames returns names sorted, for the one place go/doc does not sort for
// us and a stable page still matters.
func stableNames(names []string) []string {
	out := append([]string(nil), names...)
	sort.Strings(out)
	return out
}
