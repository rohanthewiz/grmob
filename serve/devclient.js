// The hot-reload client. Injected into index.html by `go run ./serve -dev`
// only (see serve/dev.go); the shipped page never carries it.
//
// It listens to the dev server's event stream and, when a new main.wasm is
// built, swaps it into the running page: stop the old module through
// GrMobWASM.Shutdown, wait for it to actually exit, boot the new one through
// the page's own GrMobHost.boot, and put back what the page can put back —
// the lesson comes back on its own (the route replay in boot), the scroll
// offsets are read off the DOM here and written back after the mount.
(() => {
    const host = window.GrMobHost;
    if (!host) {
        console.warn("grmob dev: the page defines no GrMobHost; falling back to page reloads");
    }
    // The build this document was served against, stamped on the script
    // tag by the server. Compared with what the stream says is current, so a
    // build that finished between the page load and the stream connecting —
    // or while the dev server was down and restarting — is not missed.
    let build = (document.currentScript && document.currentScript.dataset.build) || "";

    // --- The little bit of UI: a status pill and a compile-error overlay ---
    const style = document.createElement("style");
    style.textContent = `
      #grmob-dev-pill { position: fixed; left: 12px; bottom: 12px; z-index: 100000;
        font: 12px/1 ui-monospace, SFMono-Regular, Menlo, Consolas, monospace;
        padding: 6px 10px; border-radius: 999px; color: #fff; background: #2b6cb0;
        opacity: 0; transition: opacity .25s; pointer-events: none; }
      #grmob-dev-pill.show { opacity: .92; }
      #grmob-dev-pill.err { background: #a33c4b; }
      #grmob-dev-overlay { position: fixed; inset: 0; z-index: 99999; overflow: auto;
        background: rgba(20, 22, 26, .94); color: #ffb4bd; padding: 2rem;
        font: 13px/1.5 ui-monospace, SFMono-Regular, Menlo, Consolas, monospace;
        white-space: pre-wrap; }
      #grmob-dev-overlay h2 { margin: 0 0 1rem; font: 600 15px system-ui, sans-serif; color: #fff; }
      #grmob-dev-overlay p { color: #8b93a3; font: 12px system-ui, sans-serif; }
    `;
    document.head.appendChild(style);
    const pill = document.createElement("div");
    pill.id = "grmob-dev-pill";
    document.body.appendChild(pill);
    let pillTimer = null;
    function status(text, { err = false, hold = false } = {}) {
        clearTimeout(pillTimer);
        pill.textContent = text;
        pill.classList.toggle("err", err);
        pill.classList.add("show");
        if (!hold) pillTimer = setTimeout(() => pill.classList.remove("show"), 1600);
    }
    let overlay = null;
    function showError(title, text) {
        if (!overlay) {
            overlay = document.createElement("div");
            overlay.id = "grmob-dev-overlay";
            document.body.appendChild(overlay);
        }
        overlay.replaceChildren();
        const h = document.createElement("h2");
        h.textContent = title;
        const p = document.createElement("p");
        p.textContent = "The previous build is still running behind this. Fix the error and save; this goes away on the next successful build.";
        overlay.append(h, document.createTextNode(text), p);
    }
    function clearError() {
        if (overlay) { overlay.remove(); overlay = null; }
    }

    // --- Scroll replay ----------------------------------------------------
    // Scroll nodes are addressed by their node path, which is stable across a
    // swap as long as the tree's shape at that point did not change — and
    // when it did, the offset is meaningless anyway and is simply not applied.
    function saveScroll() {
        const saved = {};
        document.querySelectorAll('#app [data-node-type="Scroll"]').forEach(el => {
            if (el.scrollTop) saved[el.dataset.nodePath] = el.scrollTop;
        });
        return saved;
    }
    function restoreScroll(saved) {
        // Two frames out, not one: the route replay lands the lesson through
        // the push channel, and its patches may apply a frame after boot()
        // resolves.
        requestAnimationFrame(() => requestAnimationFrame(() => {
            for (const [path, top] of Object.entries(saved)) {
                const el = document.querySelector(`#app [data-node-path="${path}"]`);
                if (el) el.scrollTop = top;
            }
        }));
    }

    // --- The swap ---------------------------------------------------------
    // Swaps are serialized on a promise chain: a second build finishing while
    // the first swap is mid-flight waits its turn, so two boots never race
    // for the mount point.
    let chain = Promise.resolve();
    function swap(next) {
        chain = chain.then(() => doSwap(next)).catch(err => {
            console.error("grmob dev: hot reload failed, reloading the page", err);
            location.reload();
        });
        return chain;
    }
    async function doSwap(next) {
        if (!host) { location.reload(); return; }
        build = next;
        status("reloading…", { hold: true });
        const scroll = saveScroll();
        // A rejected `running` means the page never booted (main.wasm was
        // missing, say); there is nothing to stop, and the throw takes the
        // whole swap to the page-reload fallback above, which is right.
        const prev = await host.running;
        if (!window.GrMobWASM || typeof window.GrMobWASM.Shutdown !== "function") {
            throw new Error("the running module has no GrMobWASM.Shutdown");
        }
        window.GrMobWASM.Shutdown();
        // go.run's promise settles when main returns. Bounded, because a
        // module that will not stop should not wedge the loop; the fallback
        // is the page reload, which stops it for certain.
        await Promise.race([
            prev.exited,
            new Promise((_, reject) => setTimeout(() => reject(new Error("the old module did not exit")), 2000)),
        ]);
        // wasm_exec.js leaves the runtime's pending scheduler wake-up armed
        // after exit; when it fires it calls _resume on an exited program,
        // which throws into the console. Disarm it. Private field, so
        // guarded: if a future wasm_exec renames it, the cost is one console
        // error per reload, not a broken reload.
        const timeouts = prev.go && prev.go._scheduledTimeouts;
        if (timeouts && typeof timeouts.values === "function") {
            for (const t of timeouts.values()) clearTimeout(t);
            timeouts.clear();
        }
        host.running = host.boot();
        try {
            await host.running;
        } catch (err) {
            // The new module compiled but died on boot (a panic in init, a
            // RenderInitial that threw): show it in place of the app, the way
            // the page's own boot error does, and let the next build retry.
            showError("The new module failed to start", err && err.stack ? err.stack : String(err));
            status("boot failed", { err: true, hold: true });
            return;
        }
        clearError();
        restoreScroll(scroll);
        status("reloaded");
    }

    // --- The stream -------------------------------------------------------
    const events = new EventSource("/__dev/events");
    events.addEventListener("hello", e => {
        const d = JSON.parse(e.data);
        if (d.error) showError("Build failed", d.error);
        if (d.build && d.build !== build) swap(d.build);
        else status("hot reload on");
    });
    events.addEventListener("building", () => status("building…", { hold: true }));
    events.addEventListener("buildfail", e => {
        const d = JSON.parse(e.data);
        showError("Build failed", d.output);
        status("build failed", { err: true, hold: true });
    });
    events.addEventListener("reload", e => {
        const d = JSON.parse(e.data);
        if (d.kind === "page") { location.reload(); return; }
        clearError();
        swap(d.build);
    });
    // EventSource reconnects on its own; the pill just says so meanwhile.
    events.onerror = () => status("dev server offline", { err: true, hold: true });
})();
