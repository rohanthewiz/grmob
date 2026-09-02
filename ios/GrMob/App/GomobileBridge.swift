import Foundation
import GrMob

/// GrMobBridge implementation over the gomobile-generated framework.
///
/// The `Mobile*` functions and `MobilePatchListenerProtocol` come from
/// Frameworks/GrMob.xcframework, produced by ../build.sh from the Go
/// `mobile` package (see mobile/bridge.go for the delivery contract). gobind
/// prefixes each bound package's symbols with its capitalized package name,
/// so the bridge package's functions surface as MobileRenderInitial,
/// MobileTriggerCallback, ... and Go interfaces as <Name>Protocol. Go's int
/// binds as C long, which Swift imports as Int.
///
/// The app itself is Go code: the bound app package registers its root view
/// in an init step (mobile.Register), which runs when the Go runtime
/// initializes on first use of the framework — so by the time these calls
/// happen the app is installed. (The Go side requires the app package to
/// export at least one bindable symbol or gobind never links it; see
/// mobileapp.AppName.)
final class GomobileBridge: GrMobBridge, @unchecked Sendable {
    // Go holds a proxy for the listener it was handed, but keep our own
    // strong reference too so the Swift object's lifetime never depends on
    // the bridge's internal ref-counting.
    private let retained = Locked<Listener?>(nil)
    // Same rationale as `retained`, for the system-event sink.
    private let retainedSystem = Locked<SystemListener?>(nil)

    init() {
        // Register the writable directory before anything renders — Go-side
        // persistence (mobile.SetDataDir / DataDir; see examples/todoapp's
        // bytdb store) hydrates on the first render pass, so this must beat
        // GrMobRuntime.start(). Application Support rather than Documents:
        // a database is app-managed data, not a user-visible document, and
        // unlike Documents the directory doesn't exist until created.
        let fm = FileManager.default
        let dir = fm.urls(for: .applicationSupportDirectory, in: .userDomainMask)[0]
        try? fm.createDirectory(at: dir, withIntermediateDirectories: true)
        MobileSetDataDir(dir.path)
    }

    func renderInitial() -> String { MobileRenderInitial() }

    func triggerCallback(_ id: String) -> String {
        MobileTriggerCallback(id)
    }

    func triggerTextCallback(_ id: String, _ value: String) -> String {
        MobileTriggerTextCallback(id, value)
    }

    func triggerBoolCallback(_ id: String, _ value: Bool) -> String {
        MobileTriggerBoolCallback(id, value)
    }

    func triggerIntCallback(_ id: String, _ value: Int) -> String {
        MobileTriggerIntCallback(id, value)
    }

    func setListener(_ listener: @escaping (String) -> Void) {
        let l = Listener(listener)
        retained.set(l)
        MobileSetListener(l)
    }

    // System events (toasts, external URLs) ride their own single-method
    // interface for the same reason patches do: gobind cannot bind a Go func
    // parameter, so a callback crosses the FFI as a protocol or not at all.
    // See mobile/sysevents.go for the payload contract and SystemEvents.swift
    // for what this shell does with each event.
    func setSystemEventListener(_ listener: @escaping (String, String) -> Void) {
        let l = SystemListener(listener)
        retainedSystem.set(l)
        MobileSetSystemEventListener(l)
    }

    private final class Listener: NSObject, MobilePatchListenerProtocol {
        private let deliver: (String) -> Void

        init(_ deliver: @escaping (String) -> Void) {
            self.deliver = deliver
        }

        // Called from a Go goroutine; GrMobRuntime hops to the main thread.
        func applyPatches(_ patches: String?) {
            deliver(patches ?? "")
        }
    }
}

extension GomobileBridge {
    /// Bridges the bound Go interface onto a plain Swift closure. Mirrors
    /// `Listener` above; gobind hands both parameters across as optionals.
    fileprivate final class SystemListener: NSObject, MobileSystemEventListenerProtocol {
        private let deliver: (String, String) -> Void

        init(_ deliver: @escaping (String, String) -> Void) {
            self.deliver = deliver
        }

        // Called from a Go goroutine; SystemEvents hops to the main actor.
        func onSystemEvent(_ name: String?, payload: String?) {
            deliver(name ?? "", payload ?? "")
        }
    }
}

/// Minimal lock-guarded box (the bridge is Sendable by contract; this keeps
/// the mutable listener reference honest about it).
private final class Locked<T>: @unchecked Sendable {
    private var value: T
    private let lock = NSLock()

    init(_ value: T) { self.value = value }

    func set(_ newValue: T) {
        lock.lock()
        defer { lock.unlock() }
        value = newValue
    }
}
