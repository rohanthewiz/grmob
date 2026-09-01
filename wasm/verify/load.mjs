// Loads the real wasm/grmob-runtime.js into a controlled context.
//
// The runtime is a classic browser script, not a module: it declares
// `const GrMob = (() => {...})()` at the top level and, at the bottom, calls
// waitForWasm() and assigns window.GrMobRequestPermission. So it cannot be
// imported — it has to be *evaluated* with the globals a page would have
// already provided, which is exactly what node:vm is for.
//
// Nothing here modifies the file under test. The one addition is a single
// appended statement that publishes GrMob onto the context, because a
// top-level `const` in a vm script stays in that script's lexical scope and
// would otherwise be unreachable. (In a browser it is reachable: a classic
// script's top-level const is a global lexical binding, which is how
// index.html's inline script calls GrMob.mount.)

import { readFileSync } from "node:fs";
import vm from "node:vm";
import { newDOM } from "./dom.mjs";

const RUNTIME = new URL("../grmob-runtime.js", import.meta.url);

// The statement appended to the source. Named so a stack trace or a diff
// makes it obvious this line is the harness's and not the runtime's.
const EXPORT_SHIM = "\n;globalThis.__grmobExport = GrMob;\n";

/**
 * loadRuntime evaluates grmob-runtime.js against a fresh DOM and returns
 * everything a test needs to drive it.
 *
 * @returns {{GrMob: {mount: Function, patch: Function},
 *            document: object, window: object,
 *            mountPoint: object,
 *            drainFrames: () => number, pendingFrames: () => number,
 *            dispatched: Array<{id: string, payload: any}>}}
 */
export function loadRuntime({ mountId = "app" } = {}) {
    const dom = newDOM();

    // The mount point the page would have in its markup (index.html's
    // <div id="app">). Registered by hand because this DOM has no parser.
    const mountPoint = dom.document.createElement("div");
    mountPoint.setAttribute("id", mountId);
    dom.document.body.appendChild(mountPoint);
    dom.document.byId.set(mountId, mountPoint);

    // Every dispatch the runtime makes toward "Go", in order. Tests assert on
    // this instead of stubbing per case, so an unexpected extra dispatch —
    // the failure mode a keydown listener is most likely to have — shows up
    // as a length mismatch rather than passing unnoticed.
    const dispatched = [];
    dom.window.GoInvokeCallback = (id, payload) => {
        // The payload is round-tripped through JSON, which is what
        // index.html's real GoInvokeCallback does before handing it to Go
        // (ReceiveEvent takes a string). Doing the same here is the faithful
        // model *and* it settles a vm detail: an object built inside the
        // context has that context's Object.prototype, so a strict deep-equal
        // against a host-realm literal fails on prototype identity alone.
        // Serializing at the boundary — as the bridge does — keeps realm
        // bookkeeping out of every assertion.
        dispatched.push({ id, payload: JSON.parse(JSON.stringify(payload)) });
    };

    // Pending timers, keyed by the id setTimeout handed out, in insertion
    // order — which is the order drainTimers runs them in.
    const timers = new Map();
    let timerID = 0;

    const sandbox = {
        document: dom.document,
        window: dom.window,
        requestAnimationFrame: dom.requestAnimationFrame,
        // A queue, not a timer, for the same reason requestAnimationFrame is
        // one (see dom.mjs): a test has to be able to say "now the delay
        // elapsed" and observe the difference, and a real timer would make
        // that a race. It also keeps waitForWasm()'s 100ms poll from spinning
        // the process forever — nothing fires unless a test drains.
        //
        // Long press is what needs this: the runtime synthesizes the gesture
        // from a pointerdown plus a 500ms setTimeout, and both the firing and
        // the cancelling are behavior worth pinning.
        setTimeout: (fn, delay = 0) => {
            const id = ++timerID;
            timers.set(id, { fn, delay });
            return id;
        },
        clearTimeout: (id) => timers.delete(id),
        console,
        performance,
    };
    // The page's own `window.window === window` identity, which the runtime
    // does not rely on but any future code reading window.document would.
    sandbox.globalThis = sandbox;

    const context = vm.createContext(sandbox);
    const source = readFileSync(RUNTIME, "utf8") + EXPORT_SHIM;
    vm.runInContext(source, context, { filename: RUNTIME.pathname });

    const GrMob = sandbox.__grmobExport;
    if (!GrMob || typeof GrMob.mount !== "function" || typeof GrMob.patch !== "function") {
        throw new Error("grmob-runtime.js did not expose mount/patch");
    }

    return {
        GrMob,
        document: dom.document,
        window: dom.window,
        mountPoint,
        drainFrames: dom.drainFrames,
        pendingFrames: dom.pendingFrames,
        dispatched,

        // drainTimers fires every timer queued *at the moment it is called*
        // whose delay is at least minDelay, oldest first, and returns how
        // many ran. The filter exists so a test can fire the long-press timer
        // (500ms) without also firing waitForWasm's 100ms poll, which would
        // just re-queue itself; timers a callback queues while draining are
        // left for the next call rather than run in this one.
        drainTimers(minDelay = 0) {
            const batch = [...timers.entries()].filter(([, t]) => t.delay >= minDelay);
            let ran = 0;
            for (const [id, t] of batch) {
                timers.delete(id);
                t.fn();
                ran++;
            }
            return ran;
        },
        pendingTimers: () => timers.size,
    };
}

/**
 * nodeAt returns the element the runtime gave a given node path, which is the
 * same lookup its own patch handler does.
 */
export function nodeAt(document, path) {
    return document.querySelector(`[data-node-path="${path}"]`);
}
