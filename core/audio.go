package core

import (
	"strings"
	"sync"
)

// Audio is the app's one player: a single stream the platform plays on the
// app's behalf, with the platform's own media session behind it — the
// lock-screen controls, the notification, the headphone buttons, the
// "keeps going when the app is in the background". It is a service, not a
// node: nothing about playback belongs in the view tree (a screen showing
// the player can be popped while the audio keeps going), and the thing it
// ultimately drives is an OS-level facility the app does not own.
//
// The API is therefore two one-way channels plus a record in between:
//
//	AudioLoad/Play/Pause/... ──"audio" system event──▶ host player
//	CurrentAudioStatus ◀── audioStatus ◀──"audio_status" host event── host
//
// Commands go out as one system event named "audio" whose payload carries a
// "command" (see the audioCommand constants), the same channel ShowToast
// and OpenURL use. Status comes back as the "audio_status" host event
// (host_events.go), which core folds into the record below and announces
// to subscribers — hooks.UseAudio is the way a screen normally listens.
//
// Each host maps the commands onto its own player:
//
//	Android   Media3 ExoPlayer inside a MediaSessionService
//	iOS       AVPlayer + MPNowPlayingInfoCenter + MPRemoteCommandCenter
//	Browser   HTMLAudioElement + the Media Session API
//	Headless  nothing (no system-event handler registered)
//
// # Why one player and not a node per stream
//
// A phone has one audio focus. Two streams cannot both hold the lock screen,
// and every platform's media session is process-wide, so a second player
// would either fight the first for the notification or silently not have
// one. Modeling the singleton honestly — package-level functions, one
// status — is what makes background playback and remote controls fall out
// of the platform for free instead of being re-invented per node.
//
// # Optimistic status
//
// AudioLoad and AudioStop update the record before the host answers: Load
// records the track as loading, Stop records idle. That is what lets a
// screen show "this sermon is the one playing" on the very next render
// rather than a round trip later, and what makes a headless run (tests, a
// shell with no audio support) report a stable state instead of an empty
// one. Everything else — playing, paused, position, duration, errors — is
// the host's word and is never guessed.

// AudioState is the player's phase, as the host last reported it (or as
// core set optimistically; see the package comment).
type AudioState string

const (
	AudioIdle    AudioState = "idle"    // nothing loaded
	AudioLoading AudioState = "loading" // buffering, or waiting for the host to answer a Load
	AudioPlaying AudioState = "playing"
	AudioPaused  AudioState = "paused"
	AudioEnded   AudioState = "ended" // played to the end; Play or Seek starts it again
	AudioError   AudioState = "error" // see AudioStatus.Error
)

// AudioTrack names what to play and how the platform should describe it on
// the lock screen. Only URL is required; the rest is metadata the media
// session shows (and, on a phone, is the difference between a notification
// reading "Unknown" and one reading the sermon's title).
type AudioTrack struct {
	URL        string
	Title      string
	Artist     string // the speaker, the band, the podcast host
	Album      string // the series, the show
	ArtworkURL string
}

// AudioStatus is everything the app can know about playback. Position and
// Duration are seconds; Duration is 0 until the host has learned it (a
// streamed file reports it once the headers arrive). Rate is the playback
// speed, 1 being normal.
type AudioStatus struct {
	Track    AudioTrack
	State    AudioState
	Position float64
	Duration float64
	Rate     float64
	Error    string // set when State is AudioError
}

// Loaded reports whether a track is loaded, in any state but idle.
func (s AudioStatus) Loaded() bool { return s.State != "" && s.State != AudioIdle }

// Progress is Position as a fraction of Duration, 0 while Duration is
// unknown — what a seek slider wants.
func (s AudioStatus) Progress() float64 {
	if s.Duration <= 0 {
		return 0
	}
	p := s.Position / s.Duration
	if p < 0 {
		return 0
	}
	if p > 1 {
		return 1
	}
	return p
}

// audioCommand values are the "command" field of the "audio" system event.
// Spelled out as constants so the three hosts and this file agree by name.
const (
	audioCmdLoad  = "load"
	audioCmdPlay  = "play"
	audioCmdPause = "pause"
	audioCmdSeek  = "seek"
	audioCmdSkip  = "skip"
	audioCmdRate  = "rate"
	audioCmdStop  = "stop"
)

var (
	audioMu     sync.RWMutex
	audioStatus = AudioStatus{State: AudioIdle, Rate: 1}
	audioSubs   = map[int]func(AudioStatus){}
	audioNext   int
)

// AudioOpt configures AudioLoad.
type AudioOpt interface {
	Apply(*audioLoadConfig)
}

type audioLoadConfig struct {
	autoplay bool
	start    float64
	rate     float64
}

type audioOptFunc func(*audioLoadConfig)

func (f audioOptFunc) Apply(c *audioLoadConfig) { f(c) }

// AudioAutoplay controls whether AudioLoad starts playing as soon as the
// host can. The default is true: a tap on "play" that only buffered would
// need a second tap, and the browser only allows autoplay from inside a
// user gesture anyway — which a tap handler is.
func AudioAutoplay(on bool) AudioOpt {
	return audioOptFunc(func(c *audioLoadConfig) { c.autoplay = on })
}

// AudioStartAt begins playback at the given second instead of at 0 — the
// "resume where you left off" option. The host clamps it to the track.
func AudioStartAt(seconds float64) AudioOpt {
	return audioOptFunc(func(c *audioLoadConfig) {
		if seconds > 0 {
			c.start = seconds
		}
	})
}

// AudioWithRate sets the initial playback speed; 1 is normal. Non-positive
// values are ignored.
func AudioWithRate(rate float64) AudioOpt {
	return audioOptFunc(func(c *audioLoadConfig) {
		if rate > 0 {
			c.rate = rate
		}
	})
}

// AudioLoad replaces whatever is loaded with track and, by default, starts
// playing it. An empty URL is dropped here rather than sent, since every
// host would have to reject it separately and none could report that it
// had (the same rule OpenURL applies).
//
// The status record is updated optimistically — see the package comment —
// and subscribers are notified before the command leaves, so a screen that
// re-renders on the notification already sees the new track.
func AudioLoad(track AudioTrack, opts ...AudioOpt) {
	if strings.TrimSpace(track.URL) == "" {
		return
	}
	conf := audioLoadConfig{autoplay: true, rate: 1}
	for _, o := range opts {
		o.Apply(&conf)
	}

	setAudioStatus(AudioStatus{
		Track:    track,
		State:    AudioLoading,
		Position: conf.start,
		Rate:     conf.rate,
	})

	SendSystemEvent("audio", map[string]any{
		"command":  audioCmdLoad,
		"url":      track.URL,
		"title":    track.Title,
		"artist":   track.Artist,
		"album":    track.Album,
		"artwork":  track.ArtworkURL,
		"autoplay": conf.autoplay,
		"start":    conf.start,
		"rate":     conf.rate,
	})
}

// AudioPlay resumes the loaded track. After AudioEnded it starts over from
// the beginning. With nothing loaded it does nothing.
func AudioPlay() {
	if !CurrentAudioStatus().Loaded() {
		return
	}
	SendSystemEvent("audio", map[string]any{"command": audioCmdPlay})
}

// AudioPause pauses the loaded track, keeping its position.
func AudioPause() {
	if !CurrentAudioStatus().Loaded() {
		return
	}
	SendSystemEvent("audio", map[string]any{"command": audioCmdPause})
}

// AudioToggle plays when paused and pauses when playing — the one-button
// transport control. Anything else (loading, ended, error) is treated as
// "please play", which is what a user tapping the button means.
func AudioToggle() {
	if CurrentAudioStatus().State == AudioPlaying {
		AudioPause()
		return
	}
	AudioPlay()
}

// AudioSeek moves playback to the given second. The host clamps it to
// [0, duration].
func AudioSeek(seconds float64) {
	if !CurrentAudioStatus().Loaded() {
		return
	}
	if seconds < 0 {
		seconds = 0
	}
	SendSystemEvent("audio", map[string]any{"command": audioCmdSeek, "position": seconds})
}

// AudioSkip moves playback by delta seconds relative to where it actually is
// — negative to go back. The host does the arithmetic, not core: the
// position core knows is up to one status tick old, and a "+30s" computed
// from a stale number lands somewhere subtly wrong.
func AudioSkip(delta float64) {
	if !CurrentAudioStatus().Loaded() {
		return
	}
	SendSystemEvent("audio", map[string]any{"command": audioCmdSkip, "delta": delta})
}

// AudioSetRate changes the playback speed; 1 is normal, 1.5 is the podcast
// listener's favorite. Non-positive rates are ignored — 0 would be a pause
// spelled confusingly, and the hosts reject it anyway.
func AudioSetRate(rate float64) {
	if rate <= 0 || !CurrentAudioStatus().Loaded() {
		return
	}
	SendSystemEvent("audio", map[string]any{"command": audioCmdRate, "rate": rate})
}

// AudioStop unloads the track and releases the media session: the
// lock-screen controls disappear and the status returns to idle. Pause is
// what a user usually wants; Stop is for "sign out", "this content is no
// longer available", and the like.
func AudioStop() {
	setAudioStatus(AudioStatus{State: AudioIdle, Rate: 1})
	SendSystemEvent("audio", map[string]any{"command": audioCmdStop})
}

// CurrentAudioStatus returns the last known status. Safe from any goroutine.
func CurrentAudioStatus() AudioStatus {
	audioMu.RLock()
	defer audioMu.RUnlock()
	return audioStatus
}

// OnAudioStatus subscribes fn to every status change. The returned function
// cancels the subscription. fn runs on whichever goroutine delivered the
// change — a host bridge call, or the app's own AudioLoad — and must not
// block; the usual body is a State write or a RequestRender.
//
// Most screens want hooks.UseAudio instead, which subscribes once per
// component and re-renders on each change.
func OnAudioStatus(fn func(AudioStatus)) (cancel func()) {
	audioMu.Lock()
	defer audioMu.Unlock()
	id := audioNext
	audioNext++
	audioSubs[id] = fn
	return func() {
		audioMu.Lock()
		defer audioMu.Unlock()
		delete(audioSubs, id)
	}
}

// ReceiveAudioStatus is the typed entry point for a host that builds the
// status in Go (a test, an embedder). The JSON hosts arrive through
// ReceiveHostEvent("audio_status", ...) instead, which decodes into this.
//
// Track metadata is not something a host reports — it only ever echoes the
// URL — so the incoming Track's URL is matched against the loaded track: the
// same URL keeps the title, artist and artwork the app supplied; a different
// one (a host playing something the app did not load, which should not
// happen but must not corrupt the record) keeps only the URL.
func ReceiveAudioStatus(s AudioStatus) {
	audioMu.RLock()
	prev := audioStatus
	audioMu.RUnlock()

	if s.Track.URL == prev.Track.URL {
		s.Track = prev.Track
	}
	if s.Rate <= 0 {
		s.Rate = prev.Rate
	}
	if s.State == "" {
		s.State = AudioIdle
	}
	if s.State != AudioError {
		s.Error = ""
	}
	setAudioStatus(s)
}

// receiveAudioStatus decodes the "audio_status" host event. Missing fields
// take their zero value except rate, which ReceiveAudioStatus fills from the
// previous status — a host that never reports rate should not reset it.
//
// The payload keys are the contract every host writes:
//
//	url       string   the loaded track's URL, "" when idle
//	state     string   an AudioState value
//	position  number   seconds
//	duration  number   seconds, 0 while unknown
//	rate      number   playback speed
//	error     string   message when state is "error"
func receiveAudioStatus(data map[string]any) {
	s := AudioStatus{}
	s.Track.URL, _ = data["url"].(string)
	if st, ok := data["state"].(string); ok {
		s.State = AudioState(st)
	}
	s.Position, _ = numberProp(data, "position")
	s.Duration, _ = numberProp(data, "duration")
	s.Rate, _ = numberProp(data, "rate")
	s.Error, _ = data["error"].(string)
	ReceiveAudioStatus(s)
}

// setAudioStatus stores s and notifies subscribers outside the lock, so a
// subscriber may read CurrentAudioStatus, subscribe, or cancel from inside
// its handler.
func setAudioStatus(s AudioStatus) {
	audioMu.Lock()
	audioStatus = s
	fns := make([]func(AudioStatus), 0, len(audioSubs))
	for _, fn := range audioSubs {
		fns = append(fns, fn)
	}
	audioMu.Unlock()

	for _, fn := range fns {
		fn(s)
	}
}

// resetAudioForTest returns the record and subscriptions to their initial
// state; tests call it so one test's playback cannot leak into the next.
func resetAudioForTest() {
	audioMu.Lock()
	defer audioMu.Unlock()
	audioStatus = AudioStatus{State: AudioIdle, Rate: 1}
	audioSubs = map[int]func(AudioStatus){}
}
