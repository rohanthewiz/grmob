import AVFoundation
import MediaPlayer
import UIKit

/// The iOS half of grmob's audio service (core/audio.go): commands in as the
/// "audio" system event, status out as the "audio_status" host event.
///
///     core.AudioLoad/Play/... ──"audio" {command}──▶ handle ──▶ AVPlayer
///     core.CurrentAudioStatus ◀──"audio_status"──── emit ◀── KVO + periodic observer
///
/// One AVPlayer for the process, kept for the app's lifetime; each load swaps
/// its item. Around it, the three things that make audio "native" on iOS
/// rather than merely audible:
///
///   - the audio session is `.playback`, which — together with the `audio`
///     background mode in Info.plist — is what keeps the stream going when
///     the app leaves the foreground;
///   - `MPNowPlayingInfoCenter` gets the track's title, speaker, artwork and
///     live position, which is the lock-screen card and the Control Center
///     tile;
///   - `MPRemoteCommandCenter` maps the card's buttons, the headset clicker
///     and CarPlay back onto the same commands the app itself sends.
///
/// Status is reported on every observed change (time-control status, item
/// status, duration, end of playback, errors) and twice a second while a time
/// observer is installed, which is what moves the app's seek bar. Every
/// report is a full status, never a delta.
///
/// Main-actor only: AVPlayer is safe from any thread but the now-playing
/// center and the remote command center are not, and every entry point here
/// arrives from `SystemEvents.dispatch`, which already hops to the main
/// actor.
@MainActor
final class AudioPlayer {
    static let shared = AudioPlayer()

    /// Where status goes: (eventName, payloadJSON). `SystemEvents.attach`
    /// points it at `GrMobRuntime.hostEvent`, which serializes the delivery
    /// with render passes and applies the patches the tick produced.
    var report: ((String, String) -> Void)?

    private var player: AVPlayer?
    private var item: AVPlayerItem?
    /// The loaded track's URL as Go named it; "" when nothing is loaded.
    private var url = ""
    private var track: [String: Any] = [:]
    private var errorText = ""
    private var rate: Float = 1
    private var ended = false
    private var artwork: MPMediaItemArtwork?

    private var timeObserver: Any?
    private var playerObservations: [NSKeyValueObservation] = []
    private var itemObservations: [NSKeyValueObservation] = []
    private var itemNotifications: [NSObjectProtocol] = []
    private var commandsInstalled = false
    private var interruptionToken: NSObjectProtocol?

    private init() {}

    // MARK: - Commands

    /// Dispatches one "audio" system event. Unknown commands are dropped,
    /// matching every host's contract for unknown events.
    func handle(_ data: [String: Any]) {
        switch data["command"] as? String {
        case "load": load(data)
        case "play": play()
        case "pause": player?.pause()
        case "seek": seek(to: number(data, "position") ?? 0)
        case "skip": skip(by: number(data, "delta") ?? 0)
        case "rate": setRate(Float(number(data, "rate") ?? 0))
        case "stop": stop()
        default: break
        }
    }

    private func load(_ data: [String: Any]) {
        guard let string = data["url"] as? String, !string.isEmpty,
              let target = URL(string: string)
        else { return }

        activateSession()
        teardownItem()
        url = string
        track = data
        errorText = ""
        ended = false
        artwork = nil
        rate = Float(number(data, "rate") ?? 1)
        if rate <= 0 { rate = 1 }

        let player = self.player ?? makePlayer()
        let newItem = AVPlayerItem(url: target)
        item = newItem
        observe(newItem)
        player.replaceCurrentItem(with: newItem)
        // defaultRate (iOS 16+) is the speed play() resumes at, so a rate
        // chosen while paused survives the next play without setting
        // player.rate directly — which would also *start* playback.
        player.defaultRate = rate
        if let start = number(data, "start"), start > 0 {
            player.seek(to: CMTime(seconds: start, preferredTimescale: 600))
        }
        installRemoteCommands()
        loadArtwork(from: data["artwork"] as? String)
        updateNowPlaying()

        if (data["autoplay"] as? Bool) ?? true {
            play()
        } else {
            emit()
        }
    }

    private func play() {
        guard let player, !url.isEmpty else { return }
        if ended {
            // After the end, play means "from the top": AVPlayer stays
            // parked at the end otherwise.
            ended = false
            player.seek(to: .zero)
        }
        player.defaultRate = rate
        player.play()
    }

    private func seek(to seconds: Double) {
        guard let player else { return }
        let duration = currentDuration
        let clamped = max(0, duration > 0 ? min(seconds, duration) : seconds)
        if ended && clamped < duration { ended = false }
        player.seek(
            to: CMTime(seconds: clamped, preferredTimescale: 600),
            toleranceBefore: .zero, toleranceAfter: .zero
        ) { [weak self] _ in
            MainActor.assumeIsolated { self?.emit() }
        }
    }

    private func skip(by delta: Double) {
        seek(to: currentPosition + delta)
    }

    private func setRate(_ newRate: Float) {
        guard newRate > 0, let player else { return }
        rate = newRate
        player.defaultRate = newRate
        if player.timeControlStatus == .playing {
            player.rate = newRate
        }
        emit()
    }

    private func stop() {
        teardownItem()
        player?.replaceCurrentItem(with: nil)
        url = ""
        track = [:]
        errorText = ""
        ended = false
        artwork = nil
        rate = 1 // Go's record resets to 1 on stop; the report must agree
        MPNowPlayingInfoCenter.default().nowPlayingInfo = nil
        // Deactivating with the notify option lets whatever this player
        // interrupted (music, a podcast) resume.
        try? AVAudioSession.sharedInstance().setActive(false, options: .notifyOthersOnDeactivation)
        emit()
    }

    // MARK: - Player and observation

    private func makePlayer() -> AVPlayer {
        let player = AVPlayer()
        // Stall handling stays with AVFoundation: it waits for enough buffer
        // before starting (reported as "loading" through timeControlStatus)
        // rather than stuttering.
        player.automaticallyWaitsToMinimizeStalling = true
        self.player = player

        // Twice a second while playing; the observer is not called while
        // paused, so this is also the ticker's own throttle.
        timeObserver = player.addPeriodicTimeObserver(
            forInterval: CMTime(seconds: 0.5, preferredTimescale: 600), queue: .main
        ) { [weak self] _ in
            MainActor.assumeIsolated { self?.tick() }
        }
        playerObservations = [
            player.observe(\.timeControlStatus) { [weak self] _, _ in
                MainActor.assumeIsolated { self?.emit() }
            },
            player.observe(\.rate) { [weak self] _, _ in
                MainActor.assumeIsolated { self?.updateNowPlaying() }
            },
        ]
        observeInterruptions()
        return player
    }

    private func observe(_ item: AVPlayerItem) {
        itemObservations = [
            item.observe(\.status) { [weak self] item, _ in
                MainActor.assumeIsolated {
                    guard let self else { return }
                    if item.status == .failed {
                        self.errorText = item.error?.localizedDescription ?? "playback failed"
                    }
                    self.emit()
                }
            },
            item.observe(\.duration) { [weak self] _, _ in
                MainActor.assumeIsolated {
                    self?.updateNowPlaying()
                    self?.emit()
                }
            },
        ]
        let center = NotificationCenter.default
        itemNotifications = [
            center.addObserver(forName: .AVPlayerItemDidPlayToEndTime, object: item, queue: .main) { [weak self] _ in
                MainActor.assumeIsolated {
                    self?.ended = true
                    self?.emit()
                }
            },
            center.addObserver(forName: .AVPlayerItemFailedToPlayToEndTime, object: item, queue: .main) { [weak self] note in
                MainActor.assumeIsolated {
                    let error = note.userInfo?[AVPlayerItemFailedToPlayToEndTimeErrorKey] as? Error
                    self?.errorText = error?.localizedDescription ?? "playback failed"
                    self?.emit()
                }
            },
        ]
    }

    private func teardownItem() {
        itemObservations = []
        for token in itemNotifications { NotificationCenter.default.removeObserver(token) }
        itemNotifications = []
        item = nil
    }

    /// A phone call or an alarm pauses the session; when it ends, the
    /// system says whether resuming is appropriate (it is not if the user
    /// answered the call and hung up minutes later, say).
    private func observeInterruptions() {
        guard interruptionToken == nil else { return }
        interruptionToken = NotificationCenter.default.addObserver(
            forName: AVAudioSession.interruptionNotification, object: nil, queue: .main
        ) { [weak self] note in
            MainActor.assumeIsolated {
                guard let info = note.userInfo,
                      let raw = info[AVAudioSessionInterruptionTypeKey] as? UInt,
                      let type = AVAudioSession.InterruptionType(rawValue: raw)
                else { return }
                switch type {
                case .began:
                    self?.emit() // the system paused; timeControlStatus KVO reports it too
                case .ended:
                    let optionsRaw = info[AVAudioSessionInterruptionOptionKey] as? UInt ?? 0
                    if AVAudioSession.InterruptionOptions(rawValue: optionsRaw).contains(.shouldResume) {
                        self?.play()
                    }
                @unknown default:
                    break
                }
            }
        }
    }

    private func activateSession() {
        let session = AVAudioSession.sharedInstance()
        do {
            // .spokenAudio: the system pauses (rather than ducks) this
            // session for a spoken interruption such as navigation prompts —
            // right for a sermon, wrong for background music.
            try session.setCategory(.playback, mode: .spokenAudio)
            try session.setActive(true)
        } catch {
            NSLog("GrMob: audio session activation failed: \(error)")
        }
    }

    // MARK: - Status

    private var currentPosition: Double {
        guard let player else { return 0 }
        let t = player.currentTime().seconds
        return t.isFinite ? max(0, t) : 0
    }

    private var currentDuration: Double {
        guard let item else { return 0 }
        let d = item.duration.seconds
        return d.isFinite ? d : 0
    }

    /// Folds AVPlayer's several flags into core's one AudioState.
    private var state: String {
        if !errorText.isEmpty { return "error" }
        if url.isEmpty { return "idle" }
        if ended { return "ended" }
        guard let player else { return "loading" }
        switch player.timeControlStatus {
        case .playing: return "playing"
        case .waitingToPlayAtSpecifiedRate: return "loading"
        default: return item?.status == .readyToPlay ? "paused" : "loading"
        }
    }

    private func tick() {
        emit()
        updateNowPlaying()
    }

    private func emit() {
        let payload: [String: Any] = [
            "url": url,
            "state": state,
            "position": currentPosition,
            "duration": currentDuration,
            "rate": Double(rate),
            "error": errorText,
        ]
        guard let data = try? JSONSerialization.data(withJSONObject: payload),
              let json = String(data: data, encoding: .utf8)
        else { return }
        report?("audio_status", json)
    }

    // MARK: - Lock screen

    private func installRemoteCommands() {
        guard !commandsInstalled else { return }
        commandsInstalled = true
        let center = MPRemoteCommandCenter.shared()
        // Remote command handlers are delivered on the main thread.
        center.playCommand.addTarget { [weak self] _ in
            MainActor.assumeIsolated { self?.play() }
            return .success
        }
        center.pauseCommand.addTarget { [weak self] _ in
            MainActor.assumeIsolated { self?.player?.pause() }
            return .success
        }
        center.togglePlayPauseCommand.addTarget { [weak self] _ in
            MainActor.assumeIsolated {
                guard let self else { return }
                if self.player?.timeControlStatus == .playing { self.player?.pause() } else { self.play() }
            }
            return .success
        }
        center.skipForwardCommand.preferredIntervals = [15]
        center.skipForwardCommand.addTarget { [weak self] event in
            let interval = (event as? MPSkipIntervalCommandEvent)?.interval ?? 15
            MainActor.assumeIsolated { self?.skip(by: interval) }
            return .success
        }
        center.skipBackwardCommand.preferredIntervals = [15]
        center.skipBackwardCommand.addTarget { [weak self] event in
            let interval = (event as? MPSkipIntervalCommandEvent)?.interval ?? 15
            MainActor.assumeIsolated { self?.skip(by: -interval) }
            return .success
        }
        center.changePlaybackPositionCommand.addTarget { [weak self] event in
            guard let event = event as? MPChangePlaybackPositionCommandEvent else { return .commandFailed }
            MainActor.assumeIsolated { self?.seek(to: event.positionTime) }
            return .success
        }
    }

    private func updateNowPlaying() {
        guard !url.isEmpty else { return }
        var info: [String: Any] = [
            MPMediaItemPropertyTitle: track["title"] as? String ?? "",
            MPMediaItemPropertyArtist: track["artist"] as? String ?? "",
            MPMediaItemPropertyAlbumTitle: track["album"] as? String ?? "",
            MPNowPlayingInfoPropertyElapsedPlaybackTime: currentPosition,
            MPNowPlayingInfoPropertyPlaybackRate: Double(player?.rate ?? 0),
            MPNowPlayingInfoPropertyDefaultPlaybackRate: Double(rate),
        ]
        let duration = currentDuration
        if duration > 0 { info[MPMediaItemPropertyPlaybackDuration] = duration }
        if let artwork { info[MPMediaItemPropertyArtwork] = artwork }
        MPNowPlayingInfoCenter.default().nowPlayingInfo = info
    }

    /// Artwork is fetched off the main actor and applied only if the same
    /// track is still loaded when it arrives.
    private func loadArtwork(from string: String?) {
        guard let string, !string.isEmpty, let artworkURL = URL(string: string) else { return }
        let forURL = url
        Task { [weak self] in
            guard let (data, _) = try? await URLSession.shared.data(from: artworkURL),
                  let image = UIImage(data: data)
            else { return }
            await MainActor.run {
                guard let self, self.url == forURL else { return }
                self.artwork = MPMediaItemArtwork(boundsSize: image.size) { _ in image }
                self.updateNowPlaying()
            }
        }
    }

    // MARK: - Helpers

    /// JSON numbers arrive as NSNumber whatever their Go type was.
    private func number(_ data: [String: Any], _ key: String) -> Double? {
        (data[key] as? NSNumber)?.doubleValue
    }
}
