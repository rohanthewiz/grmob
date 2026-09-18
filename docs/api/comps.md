# Package comps

```go
import "github.com/rohanthewiz/grmob/comps"
```

Package components is grmob's widget library: higher-level UI pieces built entirely on the public core API, in the idiom of element's components package (Workstream 3 of the element-lessons plan).

## The struct-widget idiom

Every widget here is a struct implementing core.View, configured through named fields:

	comps.Card{
	    Title: "Account",
	    Body:  balanceSummary,
	    Footer: comps.Badge{Text: "verified"},
	}

Structs, not more constructor funcs in core, for two reasons. Named fields scale to many optional knobs where positional arguments do not — a widget can grow a field without breaking a single call site. And a core.View-typed field is a natural composition slot: Card's Header/Body/Footer accept any view, the way element's Card distinguishes Body (a string) from BodyComponent (a component). Where a widget offers both a simple path and a slot (Card.Title vs Card.Header), the slot wins when both are set.

## Discipline

The package deliberately lives outside core and touches nothing internal: if a widget can't be built out here, that is a gap in core's primitives, not a reason to reach inside. Widgets take their look from ctx.Theme() — colors come from the palette, sizes from the spacing/typography scales, never hard-coded — and accept Style overrides for per-use adjustment. (core keeps the widgets it already had — modal, toast, tabview; new widgets land here.)

## Hooks inside widgets

A widget's Render receives the caller's Context, so the hook rules apply exactly as they do to any component: a widget that calls NewState consumes a positional slot on the caller's context and must therefore be rendered unconditionally, every pass, like any other hook user. core.SetDebugMode flags violations as cursor-drift concerns.

Two widgets do: Accordion (expanded or collapsed) and DatePicker (is the sheet open, which month is being browsed). Both own state that is purely about the widget's own presentation, which is the bar — anything an application might want to read, drive or persist stays with the caller. Calendar is the counter-example worth keeping in view: the month on screen looks like private view state and is not, because a screen opening on the month of its next event has to be able to say so, so Calendar takes no hooks and DatePicker is where that state gets packaged for the form case.

## Topics

Package comps's reference is split into 7 topic pages by source file. The index below lists every top-level declaration with the page it is on.

| Topic | What it covers | Declares |
| --- | --- | --- |
| [Screens & structure](comps-structure.md) | Screen, app and bottom bars, the FAB, tabs, drawers, step indicators, two-pane and foldable layouts, cards, accordions, headings, breadcrumbs and separators, labelled or not. | 17 types, 13 functions and methods |
| [Lists & tables](comps-lists.md) | List rows, the settings-row family (switch, checkbox, select and slider), input rows, key-value lists, bullet lists, grouped and paged lists, data tables and timelines. | 21 types, 16 functions and methods |
| [Inputs & pickers](comps-inputs.md) | Form fields, password fields, one-time code fields, tag inputs, search, searchable selects, radio groups, dates, date ranges, times and calendars, and the two editors. | 16 types, 15 functions and methods |
| [Buttons & choices](comps-actions.md) | Buttons and their variants, copy buttons, links, chips, segmented controls, steppers, ratings and badges. | 12 types, 12 functions and methods |
| [Overlays & feedback](comps-overlays.md) | Dialogs, lightboxes, action sheets, menus, snackbars, banners, progress, spinners, skeletons and empty states. | 13 types, 10 functions and methods |
| [Data display & maps](comps-display.md) | Avatars and avatar stacks, stat tiles, the compass, clocks, countdowns and alarms, an audio player, message bubbles, expandable text, QR codes, map panels and static maps. | 21 types, 23 functions and methods |
| [Charts](comps-charts.md) | Sparklines, line, area, bar and scatter charts, histograms, heatmaps and calendar heatmaps, donuts and pies, and gauges, drawn on core.Canvas. | 16 types, 12 functions and methods |

## Index

- [Screens & structure](comps-structure.md)
    - [`type Accordion`](comps-structure.md#type-accordion)
    - [`type AppBar`](comps-structure.md#type-appbar)
    - [`type BarItem`](comps-structure.md#type-baritem)
    - [`type BottomBar`](comps-structure.md#type-bottombar)
    - [`type Breadcrumb`](comps-structure.md#type-breadcrumb)
    - [`type Card`](comps-structure.md#type-card)
    - [`type Drawer`](comps-structure.md#type-drawer)
    - [`type DrawerItem`](comps-structure.md#type-draweritem)
    - [`type FAB`](comps-structure.md#type-fab)
    - [`type FABSize`](comps-structure.md#type-fabsize)
    - [`type LabeledSeparator`](comps-structure.md#type-labeledseparator)
    - [`type Screen`](comps-structure.md#type-screen)
    - [`type Separator`](comps-structure.md#type-separator)
    - [`type StepIndicator`](comps-structure.md#type-stepindicator)
    - [`type Tabs`](comps-structure.md#type-tabs)
    - [`type TwoPane`](comps-structure.md#type-twopane)
    - [`type TwoPaneCompact`](comps-structure.md#type-twopanecompact)
- [Lists & tables](comps-lists.md)
    - [Constants](comps-lists.md#constants) — `ConcernPartialSort`, `ConcernSelectRowValueNotAnOption`
    - [`type BulletList`](comps-lists.md#type-bulletlist)
    - [`type CheckboxRow`](comps-lists.md#type-checkboxrow)
    - [`type Collapse`](comps-lists.md#type-collapse)
    - [`type CollapseBand`](comps-lists.md#type-collapseband)
    - [`type Column`](comps-lists.md#type-column)
    - [`type DataTable`](comps-lists.md#type-datatable)
    - [`type Group`](comps-lists.md#type-group)
    - [`type GroupHeader`](comps-lists.md#type-groupheader)
    - [`type GroupedList`](comps-lists.md#type-groupedlist)
    - [`type InputRow`](comps-lists.md#type-inputrow)
    - [`type KeyValue`](comps-lists.md#type-keyvalue)
    - [`type KeyValueList`](comps-lists.md#type-keyvaluelist)
    - [`type ListRow`](comps-lists.md#type-listrow)
    - [`type LoadMore`](comps-lists.md#type-loadmore)
    - [`type Pagination`](comps-lists.md#type-pagination)
    - [`type SelectRow`](comps-lists.md#type-selectrow)
    - [`type SliderRow`](comps-lists.md#type-sliderrow)
    - [`type Sort`](comps-lists.md#type-sort)
    - [`type SwitchRow`](comps-lists.md#type-switchrow)
    - [`type Timeline`](comps-lists.md#type-timeline)
    - [`type TimelineEvent`](comps-lists.md#type-timelineevent)
- [Inputs & pickers](comps-inputs.md)
    - [Constants](comps-inputs.md#constants) — `ConcernCalendarRangeReversed`, `ConcernDateRangePickerInert`, `ConcernPINInputInert`, `ConcernPINValueTooLong`, `ConcernPasswordFieldInert`, `ConcernTagInputInert`, `ConcernTimePickerInert`, `RichToolLink`
    - [Variables](comps-inputs.md#variables) — `RichToolbarDefault`
    - [`type Calendar`](comps-inputs.md#type-calendar)
    - [`type CodeEditor`](comps-inputs.md#type-codeeditor)
    - [`type DatePicker`](comps-inputs.md#type-datepicker)
    - [`type DateRangePicker`](comps-inputs.md#type-daterangepicker)
    - [`type FormField`](comps-inputs.md#type-formfield)
    - [`type PINInput`](comps-inputs.md#type-pininput)
    - [`type PasswordField`](comps-inputs.md#type-passwordfield)
    - [`type RadioGroup`](comps-inputs.md#type-radiogroup)
    - [`type RadioOption`](comps-inputs.md#type-radiooption)
    - [`type RichTextEditor`](comps-inputs.md#type-richtexteditor)
    - [`type RichToolItem`](comps-inputs.md#type-richtoolitem)
    - [`type RichToolbar`](comps-inputs.md#type-richtoolbar)
        - [`func UseRichToolbar`](comps-inputs.md#func-userichtoolbar)
    - [`type SearchField`](comps-inputs.md#type-searchfield)
    - [`type SearchableSelect`](comps-inputs.md#type-searchableselect)
    - [`type TagInput`](comps-inputs.md#type-taginput)
    - [`type TimePicker`](comps-inputs.md#type-timepicker)
- [Buttons & choices](comps-actions.md)
    - [Constants](comps-actions.md#constants) — `ColorTransparent`, `ConcernLinkInert`
    - [`type Badge`](comps-actions.md#type-badge)
    - [`type Button`](comps-actions.md#type-button)
    - [`type Chip`](comps-actions.md#type-chip)
    - [`type ChipStrip`](comps-actions.md#type-chipstrip)
    - [`type CopyButton`](comps-actions.md#type-copybutton)
    - [`type Emphasis`](comps-actions.md#type-emphasis)
    - [`type Link`](comps-actions.md#type-link)
    - [`type Prominence`](comps-actions.md#type-prominence)
    - [`type Rating`](comps-actions.md#type-rating)
    - [`type SegmentedControl`](comps-actions.md#type-segmentedcontrol)
    - [`type Stepper`](comps-actions.md#type-stepper)
    - [`type Variant`](comps-actions.md#type-variant)
- [Overlays & feedback](comps-overlays.md)
    - [Constants](comps-overlays.md#constants) — `ConcernLightboxInescapable`, `SnackbarDuration`
    - [`type ActionSheet`](comps-overlays.md#type-actionsheet)
    - [`type Banner`](comps-overlays.md#type-banner)
    - [`type Dialog`](comps-overlays.md#type-dialog)
    - [`type DialogAction`](comps-overlays.md#type-dialogaction)
    - [`type EmptyState`](comps-overlays.md#type-emptystate)
    - [`type Lightbox`](comps-overlays.md#type-lightbox)
    - [`type Menu`](comps-overlays.md#type-menu)
    - [`type ProgressBar`](comps-overlays.md#type-progressbar)
    - [`type SheetAction`](comps-overlays.md#type-sheetaction)
    - [`type Skeleton`](comps-overlays.md#type-skeleton)
    - [`type Snackbar`](comps-overlays.md#type-snackbar)
    - [`type Spinner`](comps-overlays.md#type-spinner)
    - [`type SpinnerSize`](comps-overlays.md#type-spinnersize)
- [Data display & maps](comps-display.md)
    - [Constants](comps-display.md#constants) — `ConcernAudioPlayerNoTrack`, `ConcernCountdownUntilUnset`, `ConcernNoMapProvider`, `ConcernQRDataTooLong`, `ConcernStopwatchSinceUnset`, `DefaultMapHeight`, `DefaultMapPanelHeight`, `DefaultMapScale`, `DefaultMapWidth`, `DefaultMapZoom`, `FitPadding`, `MaxFitZoom`, and 6 more
    - [`func FitRegion`](comps-display.md#func-fitregion)
    - [`func GoogleMapsHandoff`](comps-display.md#func-googlemapshandoff)
    - [`func OSMStaticMap`](comps-display.md#func-osmstaticmap)
    - [`func OpenStreetMapHandoff`](comps-display.md#func-openstreetmaphandoff)
    - [`func PlaceCount`](comps-display.md#func-placecount)
    - [`type AlarmRinging`](comps-display.md#type-alarmringing)
    - [`type AlarmRow`](comps-display.md#type-alarmrow)
    - [`type AnalogClock`](comps-display.md#type-analogclock)
    - [`type AudioPlayer`](comps-display.md#type-audioplayer)
    - [`type Avatar`](comps-display.md#type-avatar)
    - [`type AvatarStack`](comps-display.md#type-avatarstack)
    - [`type Compass`](comps-display.md#type-compass)
    - [`type Countdown`](comps-display.md#type-countdown)
    - [`type DigitalClock`](comps-display.md#type-digitalclock)
    - [`type ECLevel`](comps-display.md#type-eclevel)
    - [`type ExpandableText`](comps-display.md#type-expandabletext)
    - [`type MapHandoff`](comps-display.md#type-maphandoff)
    - [`type MapPanel`](comps-display.md#type-mappanel)
    - [`type MapPin`](comps-display.md#type-mappin)
    - [`type MessageBubble`](comps-display.md#type-messagebubble)
    - [`type QRCode`](comps-display.md#type-qrcode)
    - [`type StatTile`](comps-display.md#type-stattile)
    - [`type StaticMap`](comps-display.md#type-staticmap)
    - [`type StaticMapArea`](comps-display.md#type-staticmaparea)
    - [`type StaticMapProvider`](comps-display.md#type-staticmapprovider)
        - [`func GoogleStaticMap`](comps-display.md#func-googlestaticmap)
    - [`type Stopwatch`](comps-display.md#type-stopwatch)
- [Charts](comps-charts.md)
    - [`type AreaChart`](comps-charts.md#type-areachart)
    - [`type BarChart`](comps-charts.md#type-barchart)
    - [`type CalendarHeatmap`](comps-charts.md#type-calendarheatmap)
    - [`type ChartPoint`](comps-charts.md#type-chartpoint)
    - [`type ChartSeries`](comps-charts.md#type-chartseries)
    - [`type ChartSlice`](comps-charts.md#type-chartslice)
    - [`type DayValue`](comps-charts.md#type-dayvalue)
    - [`type DonutChart`](comps-charts.md#type-donutchart)
    - [`type Gauge`](comps-charts.md#type-gauge)
    - [`type Heatmap`](comps-charts.md#type-heatmap)
    - [`type Histogram`](comps-charts.md#type-histogram)
    - [`type LineChart`](comps-charts.md#type-linechart)
    - [`type PieChart`](comps-charts.md#type-piechart)
    - [`type ScatterChart`](comps-charts.md#type-scatterchart)
    - [`type ScatterSeries`](comps-charts.md#type-scatterseries)
    - [`type Sparkline`](comps-charts.md#type-sparkline)

