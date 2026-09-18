package comps

import (
	"fmt"
	"math"

	"github.com/rohanthewiz/grmob/core"
	"github.com/rohanthewiz/grmob/hooks"
)

// ConcernAudioPlayerNoTrack is raised, in debug builds only, when an
// AudioPlayer has no Track.URL. Play would load nothing, so the widget
// disables it — and a player whose only button is dimmed looks exactly like
// one waiting for its stream to buffer.
const ConcernAudioPlayerNoTrack = "audio-player-no-track"

// AudioPlayer is the transport for one track on the app's one player: the
// title, a seek bar with the elapsed and total time under it, and back /
// play-pause / forward, with an optional speed button and Stop.
//
//	comps.AudioPlayer{
//	    Track: core.AudioTrack{URL: sermon.URL, Title: sermon.Title, Artist: sermon.Speaker},
//	    Rates: []float64{1, 1.25, 1.5, 2},
//	}
//
//	┌ Column  role=group  name=Title ──────────────────────┐
//	│ Sunday, 14 March                                     │  Typography.Body, bold
//	│ Pastor Ade                                           │  Artist, or the state
//	│ ●━━━━━━━━━━━━━━━━○──────────────────────────────     │  Slider, seeks on release
//	│ 12:04                                        41:30   │  elapsed · total
//	│            [ −15s ]  [ Pause ]  [ +15s ]             │
//	│               [ Speed 1.25× ]  [ Stop ]              │  when Rates / ShowStop
//	└──────────────────────────────────────────────────────┘
//
// # One player, many widgets
//
// core's audio is a singleton (see core/audio.go): one stream, one media
// session, one lock screen. So an AudioPlayer does not own a player, it is a
// view of the one there is, and it asks one question of the status: is the
// loaded track mine (same URL)? If it is, the controls drive it. If it is not
// — nothing loaded, or another screen's track — this widget shows its own
// track idle, Play loads it (replacing whatever was playing, as a phone
// does), and the other controls are disabled because they would act on
// somebody else's stream. A list of sermons can therefore put an AudioPlayer
// on every detail screen without any of them fighting.
//
// # The scrub reading is the widget's
//
// While the thumb is down, the elapsed time follows the finger, and the seek
// happens once, on release (core.OnSliderChangeEnd). The thumb itself needs
// nothing — the native renderers show the finger's value during a drag — but
// the time label does, so the widget holds "dragging to t" in a hook.
//
// That is the opposite of SliderRow, which gives its draft to the caller,
// and the difference is what the reading is for. SliderRow's live reading is
// decoration on a value the app owns, and holding it would charge every
// SliderRow a whole-tree render per drag tick. Here the value being drafted
// is the host's playback position, which no app holds, and the reading is
// the point of scrubbing: you are looking for 12:04. The cost is also
// already paid — a playing track re-renders the tree on every status tick.
//
// # Accessibility
//
// The column is a RoleGroup named by the track's title, so a reader entering
// it hears what is playing. The seek bar is named "Position" with its value
// spoken as "12:04 of 41:30" rather than as seconds. The skip buttons show
// "−15s" and are named "Back 15 seconds" / "Forward 15 seconds".
//
// # Theme roles read
//
//	Title        Typography.Body, bold
//	Second line  Typography.Caption over TextSecondary
//	Times        Typography.Caption over TextSecondary
//	Controls     comps.Button: Play filled, the rest outlined
//	Gaps         Spacing.SM
type AudioPlayer struct {
	// Track is what this player plays. URL is required, and is how the
	// widget recognizes its own track in the shared status.
	Track core.AudioTrack

	// SkipSeconds is the back / forward step; 0 means 15, and a negative
	// value leaves the skip buttons out.
	SkipSeconds float64

	// Rates, when set, adds a speed button that cycles through them in order
	// ("Speed 1.25×"). A rate the player is at that is not in the list steps
	// to the first.
	Rates []float64

	// ShowStop adds a Stop button, which unloads the track.
	ShowStop bool

	// Style is applied to the outer column after its defaults.
	Style []core.StyleProp
}

// Render draws the transport. It takes two hook slots — the audio status
// subscription and the scrub reading — so render it unconditionally.
func (p AudioPlayer) Render(ctx *core.Context) *core.Node {
	t := ctx.Theme()

	// Both hooks before any branch, in a fixed order.
	status := hooks.UseAudio(ctx)
	scrub := core.NewState(ctx, -1.0)

	hasTrack := p.Track.URL != ""
	if core.IsDebugMode() && !hasTrack {
		core.ReportConcern(ConcernAudioPlayerNoTrack,
			"AudioPlayer has no Track.URL, so Play is disabled and the player can never start")
	}

	mine := hasTrack && status.Loaded() && status.Track.URL == p.Track.URL
	position, duration := 0.0, 0.0
	if mine {
		position, duration = status.Position, status.Duration
		if s := scrub.Get(); s >= 0 {
			position = s
		}
	}

	track := p.Track
	onPlay := func() {
		if mine {
			core.AudioToggle()
			return
		}
		core.AudioLoad(track)
	}
	playLabel := "Play"
	if mine && status.State == core.AudioPlaying {
		playLabel = "Pause"
	}

	items := make([]core.PropsAndChildren, 0, len(p.Style)+10)
	items = append(items,
		core.Padding(0),
		core.Gap(float64(t.Spacing.SM)),
		core.AccessibilityRole(core.RoleGroup),
		core.AccessibilityLabel(orDefault(p.Track.Title, "Audio player")),
	)
	items = append(items, asProps(p.Style)...)

	if p.Track.Title != "" {
		// Body in bold, not Typography.Subtitle: the bundled Subtitle is a
		// grey secondary heading, and the title is the one thing on the
		// player a reader looks for first.
		items = append(items, core.Text(p.Track.Title,
			core.UseStyle(t.Typography.Body),
			core.FontWeight(core.Bold)))
	}
	if line := p.secondLine(status, mine); line != "" {
		items = append(items, core.Text(line,
			core.UseStyle(t.Typography.Caption),
			core.TextColor(t.Colors.TextSecondary)))
	}

	seek := []core.PropsAndChildren{
		core.OnSliderChangeEnd(func(v float64) {
			scrub.Set(-1)
			core.AudioSeek(v)
		}),
		core.Width("100%"),
		core.Disabled(!mine || duration <= 0),
		core.AccessibilityLabel("Position"),
	}
	// The spoken value only once there is a range to speak it in. Until the
	// host has learned the duration (nothing loaded, or a stream still
	// reading its headers) the range is 0 to 0, which Compose cannot
	// express and the a11y audit reports; the disabled bar says enough.
	if duration > 0 {
		seek = append(seek, core.AccessibilityValue(core.ValueRange{
			Now:  fmt.Sprintf("%.0f", position),
			Min:  "0",
			Max:  fmt.Sprintf("%.0f", duration),
			Text: audioClock(position) + " of " + audioClock(duration),
		}))
	}

	items = append(items,
		core.Slider(position, 0, duration, func(v float64) { scrub.Set(v) }, seek...),
		core.Row(
			// Full width, or the Row hugs its two labels and space-between
			// has nothing to distribute.
			core.Padding(0),
			core.Width("100%"),
			core.Justify(core.JustifyBetween),
			// Hidden: the seek bar's value already says both, and two bare
			// times read in passing are noise.
			core.AccessibilityHidden(),
			core.Text(audioClock(position), core.UseStyle(t.Typography.Caption), core.TextColor(t.Colors.TextSecondary)),
			core.Text(audioClock(duration), core.UseStyle(t.Typography.Caption), core.TextColor(t.Colors.TextSecondary)),
		),
	)

	transport := []core.PropsAndChildren{
		core.Padding(0),
		core.Gap(float64(t.Spacing.SM)),
		core.Justify(core.JustifyCenter),
	}
	skip := p.SkipSeconds
	if skip == 0 {
		skip = 15
	}
	if skip > 0 {
		transport = append(transport, Button{
			Label:              fmt.Sprintf("−%gs", skip),
			OnTap:              func() { core.AudioSkip(-skip) },
			Emphasis:           EmphasisOutlined,
			Disabled:           !mine,
			AccessibilityLabel: fmt.Sprintf("Back %g seconds", skip),
		})
	}
	transport = append(transport, Button{
		Label:    playLabel,
		OnTap:    onPlay,
		Disabled: !hasTrack,
	})
	if skip > 0 {
		transport = append(transport, Button{
			Label:              fmt.Sprintf("+%gs", skip),
			OnTap:              func() { core.AudioSkip(skip) },
			Emphasis:           EmphasisOutlined,
			Disabled:           !mine,
			AccessibilityLabel: fmt.Sprintf("Forward %g seconds", skip),
		})
	}
	items = append(items, core.Row(transport...))

	if len(p.Rates) > 0 || p.ShowStop {
		extra := []core.PropsAndChildren{
			core.Padding(0),
			core.Gap(float64(t.Spacing.SM)),
			core.Justify(core.JustifyCenter),
		}
		if len(p.Rates) > 0 {
			rate := status.Rate
			if !mine || rate == 0 {
				rate = 1
			}
			rates := p.Rates
			extra = append(extra, Button{
				Label:    fmt.Sprintf("Speed %g×", rate),
				OnTap:    func() { core.AudioSetRate(nextAudioRate(rates, rate)) },
				Emphasis: EmphasisOutlined,
				Disabled: !mine,
			})
		}
		if p.ShowStop {
			extra = append(extra, Button{
				Label:    "Stop",
				OnTap:    core.AudioStop,
				Emphasis: EmphasisOutlined,
				Disabled: !mine,
			})
		}
		items = append(items, core.Row(extra...))
	}

	return core.Column(items...).Render(ctx)
}

// secondLine is the Artist, except when this track is loading or failed,
// when it is the state instead — the two moments the reader needs telling
// why the controls are not doing anything.
func (p AudioPlayer) secondLine(s core.AudioStatus, mine bool) string {
	if mine {
		switch s.State {
		case core.AudioLoading:
			return "Loading…"
		case core.AudioError:
			if s.Error != "" {
				return "Couldn't play: " + s.Error
			}
			return "Couldn't play"
		}
	}
	return p.Track.Artist
}

// nextAudioRate is the rate after current in rates, wrapping; a current rate
// not in the list steps to the first. Rates are compared with a tolerance
// because a host may report 1.2500001 for the 1.25 it was sent.
func nextAudioRate(rates []float64, current float64) float64 {
	for i, r := range rates {
		if math.Abs(r-current) < 0.001 {
			return rates[(i+1)%len(rates)]
		}
	}
	return rates[0]
}

// audioClock formats seconds as m:ss, or h:mm:ss past an hour. Rounded to the
// nearest second, as a phone's player shows it.
func audioClock(seconds float64) string {
	if seconds < 0 || math.IsNaN(seconds) || math.IsInf(seconds, 0) {
		seconds = 0
	}
	s := int(seconds + 0.5)
	h, m, sec := s/3600, (s%3600)/60, s%60
	if h > 0 {
		return fmt.Sprintf("%d:%02d:%02d", h, m, sec)
	}
	return fmt.Sprintf("%d:%02d", m, sec)
}
