// The three keyboard facts a shimmed DOM cannot check, checked in a browser.
//
// wasm/verify's other suites run the real grmob-runtime.js against dom.mjs — a
// few hundred lines that model element trees, attributes, listeners and which
// element holds focus. That is enough for almost everything, and its limits
// are stated in its own header: there is no layout, no bubbling, and `focus()`
// is an assignment. Three claims the keyboard pattern rests on sit exactly in
// that blind spot, and no amount of widening the shim would settle them,
// because each one is a claim about what a *browser* does:
//
//   1. tabindex="-1" really takes a <button> out of the tab order. The roving
//      tabindex is the whole reason a tab strip is one stop rather than five;
//      in dom.mjs the attribute is a string nobody reads.
//   2. A disabled control refuses focus. The runtime relies on this to keep a
//      disabled member from being landed on; dom.mjs's focus() assigns.
//   3. preventDefault on ArrowDown stops the page scrolling. A listbox that
//      moved its selection *and* scrolled the page under it would be unusable,
//      and defaultPrevented in a shim is a flag the shim set itself.
//
// # How
//
// A headless Chrome, driven over the DevTools protocol with Node's built-in
// WebSocket. No npm, no lockfile, no node_modules — the promise wasm/verify's
// run.sh makes — and no network: the page is served from a localhost HTTP
// server over the repository's own files, and the browser is one that is
// already installed.
//
// Keys are dispatched through Input.dispatchKeyEvent, which is the browser's
// real input path: the tab order is walked by the browser's own focus
// algorithm and a scroll is a real scroll. Nothing here synthesizes an event
// object, which is the point — a synthesized event is what dom.mjs already
// does.
//
// # When it does not run
//
// If there is no Chrome to launch, or the Node in use has no WebSocket global
// (before v21), the pass prints SKIP and exits 0. That is the same stance
// ios/verify takes toward a missing iPhoneOS SDK: the script's promise is that
// it catches what this machine can catch, and a pass that fails on a machine
// missing an optional tool is a pass people learn to ignore.

import { spawn } from "node:child_process";
import { mkdtempSync, writeFileSync, readFileSync, existsSync } from "node:fs";
import { tmpdir } from "node:os";
import { join, dirname } from "node:path";
import { fileURLToPath } from "node:url";
import http from "node:http";

const HERE = dirname(fileURLToPath(import.meta.url));
const RUNTIME = join(HERE, "..", "grmob-runtime.js");

// Where a Chrome might be. The env var wins so a machine with an unusual
// install (or a Chromium) can point at it without editing this list.
const CHROME_CANDIDATES = [
    process.env.GRMOB_CHROME,
    "/Applications/Google Chrome.app/Contents/MacOS/Google Chrome",
    "/Applications/Chromium.app/Contents/MacOS/Chromium",
    "/usr/bin/google-chrome",
    "/usr/bin/chromium",
    "/usr/bin/chromium-browser",
].filter(Boolean);

function findChrome() {
    return CHROME_CANDIDATES.find((p) => existsSync(p)) || null;
}

function skip(why) {
    console.log(`SKIP: browser keyboard pass (${why})`);
    process.exit(0);
}

// --------------------------------------------------------------------------
// The page
// --------------------------------------------------------------------------

// A sentinel button either side of the mount point, which is what makes the
// tab order measurable: "the strip is one stop" is a statement about where
// focus goes *next*, so there has to be a next.
//
// The tall filler is what makes the scroll measurable. A document that cannot
// scroll would pass check 3 whether or not the key was consumed.
const PAGE = `<!DOCTYPE html>
<html><head><meta charset="utf-8"><title>grmob keyboard</title></head>
<body>
<button id="before">before</button>
<div id="app"></div>
<button id="after">after</button>
<div style="height: 4000px"></div>
<script>
  // The bridge the runtime dispatches into. It is never asserted on here —
  // the dispatch path has its own suites — but it has to exist, or a click
  // the browser generates while walking the tab order throws.
  window.__dispatched = [];
  window.GoInvokeCallback = (id, payload) => window.__dispatched.push({ id, payload });
</script>
<script src="/grmob-runtime.js"></script>
</body></html>
`;

// --------------------------------------------------------------------------
// CDP
// --------------------------------------------------------------------------

function getJSON(url) {
    return new Promise((resolve, reject) => {
        http.get(url, (r) => {
            let body = "";
            r.on("data", (c) => (body += c));
            r.on("end", () => {
                try { resolve(JSON.parse(body)); } catch (e) { reject(e); }
            });
        }).on("error", reject);
    });
}

const sleep = (ms) => new Promise((r) => setTimeout(r, ms));

// Chrome writes its chosen port into the profile directory when started with
// --remote-debugging-port=0, which is how this avoids picking a fixed port
// that another process (or another copy of this script) might hold.
async function devtoolsPort(profile) {
    const portFile = join(profile, "DevToolsActivePort");
    for (let i = 0; i < 100; i++) {
        if (existsSync(portFile)) {
            const first = readFileSync(portFile, "utf8").split("\n")[0].trim();
            if (first) return Number(first);
        }
        await sleep(100);
    }
    throw new Error("Chrome never reported a DevTools port");
}

/** A single page target's CDP session, as a send(method, params) function. */
// How long a DevTools call may take before the harness gives up on it. See
// send.
const CDP_TIMEOUT_MS = 15000;

async function connect(wsURL) {
    const ws = new WebSocket(wsURL);
    await new Promise((resolve, reject) => {
        ws.addEventListener("open", resolve, { once: true });
        ws.addEventListener("error", reject, { once: true });
    });

    let nextID = 0;
    const pending = new Map();
    const events = new Map();
    ws.addEventListener("message", (e) => {
        const msg = JSON.parse(e.data);
        if (msg.id !== undefined) {
            const entry = pending.get(msg.id);
            if (!entry) return;
            pending.delete(msg.id);
            if (msg.error) entry.reject(new Error(`${entry.method}: ${msg.error.message}`));
            else entry.resolve(msg.result);
            return;
        }
        const waiter = events.get(msg.method);
        if (waiter) { events.delete(msg.method); waiter(msg.params); }
    });

    return {
        // Every round trip is bounded.
        //
        // A CDP call that never gets an answer is indistinguishable from one
        // still in flight, so without this the harness's failure mode is a hang
        // rather than a failure — and a hang is strictly worse: it is what a
        // CI run does for its whole timeout instead of reporting in a minute.
        // The number is generous next to a round trip that normally takes
        // single-digit milliseconds; what it is sized against is a page that
        // will never answer at all.
        send(method, params = {}) {
            const id = ++nextID;
            return new Promise((resolve, reject) => {
                const timer = setTimeout(() => {
                    pending.delete(id);
                    reject(new Error(`${method}: no answer from the page in ${CDP_TIMEOUT_MS}ms`));
                }, CDP_TIMEOUT_MS);
                pending.set(id, {
                    method,
                    resolve: (v) => { clearTimeout(timer); resolve(v); },
                    reject: (e) => { clearTimeout(timer); reject(e); },
                });
                ws.send(JSON.stringify({ id, method, params }));
            });
        },
        once(method) {
            return new Promise((resolve) => events.set(method, resolve));
        },
        close: () => ws.close(),
    };
}

// --------------------------------------------------------------------------
// The trees under test
// --------------------------------------------------------------------------

// A tab strip of real <button> elements: three members, the middle one
// selected, so the roving tabindex is "-1", "0", "-1". Buttons rather than
// boxes on purpose — a <button> is focusable *by default*, so this is the
// case where tabindex="-1" has something to take away.
const TABLIST = {
    Type: "Row",
    Style: { AccessibilityRole: "tablist" },
    Children: [0, 1, 2].map((i) => ({
        Type: "Button",
        Style: {
            AccessibilityRole: "tab",
            AccessibilitySelected: i === 1 ? "true" : "false",
        },
        Props: { label: `Tab ${i}`, onClick: `cb_${i}` },
    })),
};

// A listbox of options, the first selected. Divs, which is what a row of
// content is on every screen this pattern was written for.
const LISTBOX = {
    Type: "Column",
    Style: { AccessibilityRole: "listbox" },
    Children: [0, 1, 2].map((i) => ({
        Type: "Box",
        Style: {
            AccessibilityRole: "option",
            AccessibilitySelected: i === 0 ? "true" : "false",
        },
        Props: {},
    })),
};

// A filter bar: a Row carrying role="toolbar" over three real <button> chips,
// which is what components.ChipStrip renders and what a caller puts the role on.
//
// This is the fixture the tab-order claim needs a browser for, and it is a
// stronger case than the tablist above. A tablist's members are <button
// role="tab"> and their tab stops are governed by the same tabindex attribute;
// what is new here is that the toolbar's members are not named by its role at
// all — the runtime found them by walking for focusable controls — so this
// check is the one that says the *walk* found the right three elements and not
// merely that tabindex works.
//
// Deliberately no roles on the chips. A chip is a plain core.Button, so if
// focusableMembers only recognised elements carrying a role it would find none
// here and the strip would keep all three of its stops.
const TOOLBAR = {
    Type: "Row",
    Style: { AccessibilityRole: "toolbar" },
    Children: ["All", "Sermons", "Articles"].map((label, i) => ({
        Type: "Button",
        Style: {},
        Props: { label, onClick: `cb_${i}` },
    })),
};

// One enabled button and one disabled one, side by side with no composite
// around them — so what is being measured is the control's own refusal and
// not a roving tabindex.
const BUTTONS = {
    Type: "Row",
    Children: [
        { Type: "Button", Style: {}, Props: { label: "live" } },
        { Type: "Button", Style: { Disabled: true }, Props: { label: "dead" } },
    ],
};

// --------------------------------------------------------------------------
// The checks
// --------------------------------------------------------------------------

async function main() {
    if (typeof WebSocket !== "function") {
        skip("this Node has no WebSocket global; v21 or newer has one");
    }
    const chromePath = findChrome();
    if (!chromePath) {
        skip("no Chrome or Chromium found; set GRMOB_CHROME to point at one");
    }

    // The server. It answers exactly two paths, because a directory server
    // would be a second thing to get right and this needs no more than the
    // runtime file and the page that loads it.
    const runtimeSource = readFileSync(RUNTIME, "utf8");
    const server = http.createServer((req, res) => {
        if (req.url === "/grmob-runtime.js") {
            res.writeHead(200, { "content-type": "text/javascript" });
            res.end(runtimeSource);
            return;
        }
        res.writeHead(200, { "content-type": "text/html" });
        res.end(PAGE);
    });
    await new Promise((r) => server.listen(0, "127.0.0.1", r));
    const origin = `http://127.0.0.1:${server.address().port}`;

    const profile = mkdtempSync(join(tmpdir(), "grmob-chrome-"));
    const chrome = spawn(chromePath, [
        "--headless=new",
        "--remote-debugging-port=0",
        `--user-data-dir=${profile}`,
        "--no-first-run",
        "--no-default-browser-check",
        "--disable-extensions",
        "--disable-gpu",
        // The scroll check reads window.scrollY one frame after the key. A
        // smooth scroll is an animation, so the number it would read is
        // whatever fraction of the distance had been covered — which makes
        // "did not scroll" and "has not finished scrolling yet" the same
        // measurement.
        "--disable-smooth-scrolling",
        // A known window size: the scroll check needs a viewport shorter than
        // the document, and a headless default that changed would change what
        // the check means.
        "--window-size=800,600",
        "about:blank",
    ], { stdio: ["ignore", "ignore", "ignore"] });

    let session;
    const problems = [];
    try {
        const port = await devtoolsPort(profile);
        const targets = await getJSON(`http://127.0.0.1:${port}/json/list`);
        const page = targets.find((t) => t.type === "page");
        if (!page) throw new Error("Chrome opened no page target");

        session = await connect(page.webSocketDebuggerUrl);
        await session.send("Page.enable");
        await session.send("Runtime.enable");
        const loaded = session.once("Page.loadEventFired");
        await session.send("Page.navigate", { url: origin });
        await loaded;

        const evaluate = async (expression) => {
            const r = await session.send("Runtime.evaluate", {
                expression, returnByValue: true, awaitPromise: true,
            });
            if (r.exceptionDetails) {
                throw new Error(`page threw: ${r.exceptionDetails.text} ` +
                    `${r.exceptionDetails.exception?.description || ""}`);
            }
            return r.result.value;
        };

        // Mount a tree and wait a frame, which is when the runtime's deferred
        // focus/roving work has run (it defers by one rAF because the initial
        // tree is built detached).
        const mount = (tree) => evaluate(
            `new Promise((done) => {
                document.getElementById("app").innerHTML = "";
                GrMob.mount(${JSON.stringify(JSON.stringify(tree))});
                requestAnimationFrame(() => requestAnimationFrame(() => done(true)));
            })`);

        const key = async (name, code) => {
            const common = { key: name, code: name, windowsVirtualKeyCode: code,
                nativeVirtualKeyCode: code };
            await session.send("Input.dispatchKeyEvent", { type: "rawKeyDown", ...common });
            await session.send("Input.dispatchKeyEvent", { type: "keyUp", ...common });
            // Two frames and a beat. The default action of a key is the
            // browser's own work, on its own schedule — a scroll in
            // particular lands after the event has been delivered — so a
            // single rAF reads the state before the thing being measured has
            // happened.
            //
            // The wall-clock fallback beside it is not belt and braces. A key
            // that changes NOTHING — no focus move, no scroll, no DOM edit —
            // gives the browser no reason to paint, so requestAnimationFrame is
            // never called back and a wait on it alone never returns. Which is
            // precisely the case every check here is trying to detect: this
            // harness asks "did the key do something", and waiting for a frame
            // means waiting for the answer to be yes.
            //
            // That was not hypothetical. The toolbar check below hung for its
            // whole timeout the first time its subject was broken, having
            // passed every time its subject worked — a check that can only
            // report success, which is the one result a check must not be able
            // to guarantee itself. Resolving twice is harmless; the first
            // settle wins.
            await evaluate(`new Promise((d) => {
                const done = () => d(true);
                setTimeout(done, 400);
                setTimeout(() => requestAnimationFrame(
                    () => requestAnimationFrame(done)), 50);
            })`);
        };

        // The element focus is on, described well enough to name in a failure.
        const focused = () => evaluate(`(() => {
            const a = document.activeElement;
            if (!a || a === document.body) return "body";
            if (a.id) return "#" + a.id;
            return (a.getAttribute("data-node-path") || a.tagName.toLowerCase())
                + "[" + (a.getAttribute("role") || "") + "]"
                + "(" + (a.getAttribute("tabindex") ?? "none") + ")";
        })()`);

        // ------------------------------------------------------------------
        // 1. tabindex="-1" takes a <button> out of the tab order
        // ------------------------------------------------------------------
        await mount(TABLIST);

        const stamped = await evaluate(
            `[...document.getElementById("app").firstChild.children]
                .map((c) => c.getAttribute("tabindex"))`);
        if (String(stamped) !== "-1,0,-1") {
            problems.push(`the roving tabindex is ${JSON.stringify(stamped)}, want ["-1","0","-1"] ` +
                `— the rest of this check measures something else if this is wrong`);
        }

        await evaluate(`document.getElementById("before").focus()`);
        await key("Tab", 9);
        const entered = await focused();
        if (!entered.includes("[tab]") || !entered.endsWith("(0)")) {
            problems.push(`Tab entered the strip at ${entered}, want the member holding ` +
                `tabindex="0" — a browser that ignored the attribute would land on the first tab`);
        }

        await key("Tab", 9);
        const left = await focused();
        if (left !== "#after") {
            problems.push(`a second Tab landed on ${left}, want #after — the two members ` +
                `carrying tabindex="-1" are still in the tab order, so the strip is three ` +
                `stops rather than one`);
        }

        // ------------------------------------------------------------------
        // 2. a disabled control refuses focus
        // ------------------------------------------------------------------
        await mount(BUTTONS);

        const refused = await evaluate(`(() => {
            const dead = document.querySelector('[data-node-path="root/1"]');
            dead.focus();
            return document.activeElement !== dead;
        })()`);
        if (!refused) {
            problems.push(`a disabled <button> took focus from an explicit focus() call`);
        }

        await evaluate(`document.getElementById("before").focus()`);
        await key("Tab", 9);
        const firstStop = await focused();
        await key("Tab", 9);
        const secondStop = await focused();
        if (secondStop !== "#after") {
            problems.push(`Tab reached ${firstStop} then ${secondStop}; want the live button ` +
                `then #after — the disabled one is still a tab stop`);
        }

        // ------------------------------------------------------------------
        // 3. preventDefault on ArrowDown stops the scroll
        // ------------------------------------------------------------------
        // The control first: the same key, on a page with no composite
        // focused, must scroll. Without this half a browser that never
        // scrolled on ArrowDown would make the real check vacuous.
        await mount(LISTBOX);
        // #after, not the body: a scroll needs a focused element for the key
        // to be delivered to, and the body is not focusable.
        await evaluate(`window.scrollTo(0, 0); document.getElementById("after").focus();`);
        await key("ArrowDown", 40);
        const controlScroll = await evaluate(`window.scrollY`);
        if (!(controlScroll > 0)) {
            problems.push(`ArrowDown with nothing focused did not scroll the page ` +
                `(scrollY ${controlScroll}), so the check below proves nothing`);
        }

        await evaluate(`
            window.scrollTo(0, 0);
            document.querySelector('[data-node-path="root/0"]').focus();`);
        await key("ArrowDown", 40);
        const state = await evaluate(`({
            scrollY: window.scrollY,
            focused: document.activeElement.getAttribute("data-node-path"),
        })`);
        if (state.scrollY !== 0) {
            problems.push(`ArrowDown inside a listbox scrolled the page to ${state.scrollY} ` +
                `— the handler moved the active option and the page moved under it`);
        }
        if (state.focused !== "root/1") {
            problems.push(`ArrowDown moved focus to ${state.focused}, want root/1 — the ` +
                `listbox did not take the key at all, so nothing was prevented`);
        }
        // ------------------------------------------------------------------
        // 4. a toolbar of plain <button>s is one tab stop, and the arrows
        //    reach the rest
        // ------------------------------------------------------------------
        // The claim the second member rule exists for, and the one the shim
        // cannot make: there `tabindex` is a string nobody reads, so "the strip
        // is one stop" restates the attribute the runtime just wrote. Here the
        // tab order is walked by the browser's own focus algorithm, and a chip
        // that was still in it lands between the strip and #after.
        await mount(TOOLBAR);

        const chipStops = await evaluate(
            `[...document.getElementById("app").firstChild.children]
                .map((c) => c.getAttribute("tabindex"))`);
        if (String(chipStops) !== "0,-1,-1") {
            problems.push(`the toolbar's roving tabindex is ${JSON.stringify(chipStops)}, ` +
                `want ["0","-1","-1"] — focusableMembers did not find three plain ` +
                `<button> chips, and the rest of this check measures something else`);
        }

        await evaluate(`document.getElementById("before").focus()`);
        await key("Tab", 9);
        const enteredBar = await focused();
        if (!enteredBar.endsWith("(0)")) {
            problems.push(`Tab entered the toolbar at ${enteredBar}, want the chip ` +
                `holding tabindex="0"`);
        }

        await key("Tab", 9);
        const leftBar = await focused();
        if (leftBar !== "#after") {
            problems.push(`a second Tab landed on ${leftBar}, want #after — a ` +
                `twelve-chip filter bar is still twelve stops in the page's tab order, ` +
                `which is the whole thing the toolbar keyboard was added to fix`);
        }

        // And the two chips Tab now skips are reachable by the arrows, which is
        // the half that makes taking them out of the tab order legitimate.
        await evaluate(`document.getElementById("before").focus()`);
        await key("Tab", 9);
        await key("ArrowRight", 39);
        const arrowed = await focused();
        await key("ArrowRight", 39);
        const arrowedTwice = await focused();
        if (arrowed === enteredBar || arrowedTwice === arrowed) {
            problems.push(`ArrowRight left focus at ${arrowed} then ${arrowedTwice} — ` +
                `the chips Tab no longer reaches are not reachable at all, which is ` +
                `strictly worse than the three tab stops this replaced`);
        }
    } finally {
        if (session) session.close();
        chrome.kill();
        server.close();
    }

    if (problems.length) {
        console.error("FAIL: the browser disagrees with the keyboard pattern:");
        for (const p of problems) console.error(`  ${p}`);
        process.exit(1);
    }
    console.log("OK: roving tabindex, disabled focus, ArrowDown and the toolbar walk\n    hold in a real browser");
}

await main();
