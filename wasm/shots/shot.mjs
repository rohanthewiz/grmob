// The camera: a CDP client that mounts one app, drives it to a state, and
// writes a clipped PNG.
//
//   node shot.mjs --dir www --out ../../docs/images/todo.png scripts/todo.js
//   node shot.mjs --dir www --out - scripts/todo.js        (a probe: print,
//                                                           do not write)
//
// An action script is the body of an async function, so it may `await` and
// it may `return` — and a probe is simply a script whose last statement is
// a return. That is how the strings in internal/shotclaims were read off a
// real render before they were written down.
//
// # Why this and not the browser-automation tools
//
// Two reasons, and the second is the one that decided it.
//
// Framing. `Page.captureScreenshot` takes a `clip`, so the capture is the
// bezel's bounding rect and nothing else — the window's own size never
// reaches the PNG and every shot is framed identically however Chrome
// decided to size itself. A viewport capture would make the picture a
// function of the browser's mood.
//
// Resolution. This asks for `scale: 2` and gets it. The MCP browser tools
// downscale their captures on the way out (1745 CSS px arriving as a
// 1311px JPEG), which is a picture of a phone screen rendered smaller than
// a phone screen.
//
// # Why it is beside wasm/verify rather than inside it
//
// wasm/verify/browser.mjs already finds a Chrome and speaks CDP, and this
// duplicates about forty lines of it. The alternative was to export those
// from a file that is one long test script with a `main()` at the bottom,
// which would have made a suite that runs on every verify pass into a
// library for a harness that runs by hand. Two small copies of a launch
// sequence, in two programs with different lifetimes, is the cheaper of
// the two mistakes — and the copy is named here so that a Chrome path
// which stops working is fixed in both.

import { spawn } from "node:child_process";
import { existsSync, mkdtempSync, readFileSync, writeFileSync } from "node:fs";
import http from "node:http";
import { tmpdir } from "node:os";
import { dirname, extname, join, resolve } from "node:path";
import { fileURLToPath } from "node:url";

const HERE = dirname(fileURLToPath(import.meta.url));

// Where a Chrome might be. The same list wasm/verify/browser.mjs uses, env
// var first so an unusual install can be pointed at without an edit.
const CHROME_CANDIDATES = [
    process.env.GRMOB_CHROME,
    "/Applications/Google Chrome.app/Contents/MacOS/Google Chrome",
    "/Applications/Chromium.app/Contents/MacOS/Chromium",
    "/usr/bin/google-chrome",
    "/usr/bin/chromium",
    "/usr/bin/chromium-browser",
].filter(Boolean);

const sleep = (ms) => new Promise((r) => setTimeout(r, ms));

function usage(msg) {
    console.error(`${msg}

usage: node shot.mjs --dir <served dir> --out <file|-> <script.js>

  --dir   the directory to serve; must hold index.html, main.wasm,
          wasm_exec.js and grmob-runtime.js (shoot.sh builds one)
  --out   where the PNG goes, or "-" to print the action script's return
          value instead of writing anything. The probe form is how the
          rendered DOM is read at all — see the header of any script.`);
    process.exit(2);
}

function parseArgs(argv) {
    const out = { dir: "", out: "", script: "" };
    for (let i = 0; i < argv.length; i++) {
        const a = argv[i];
        if (a === "--dir") out.dir = argv[++i];
        else if (a === "--out") out.out = argv[++i];
        else if (a.startsWith("--")) usage(`unknown flag ${a}`);
        else out.script = a;
    }
    if (!out.dir) usage("no --dir");
    if (!out.out) usage("no --out");
    if (!out.script) usage("no action script");
    return out;
}

// --- The action script's vocabulary --------------------------------------

// The prelude every action script runs inside.
//
// These helpers speak the runtime's own attribute names, which is what
// makes a driven state honest: `tap` does not call a Go function, it finds
// the element carrying the click handler and invokes the callback ID the
// renderer stamped on it — exactly what grmob-runtime.js does when a
// finger lands there.
//
//   tap(text)              click the innermost thing showing this text
//   typeInto(hint, value)  change the input with this placeholder or value
//   toggle(text, on)       tick or untick the checkbox in that row
//   within(attr, text)     the holder of an attribute inside that row
//   byAttr(attr)           every element carrying one, in tree order
//   invoke(el, attr, p)    dispatch one element's callback by hand
//   selectTab(index)       move the tab strip
//   scrollTo(text)         bring the node showing this text into the frame
//   settle()               wait two frames for the patch to land
//
// # The correction `tap` needed
//
// A container's textContent equals its only child's, so `byText("Add")`
// matched the Row wrapping the button as readily as the button itself —
// and dispatching the Row's handler is dispatching whatever the row does,
// which on a list is "select this article" rather than "press this
// button". The fix is to prefer the candidate that actually carries the
// handler and, among those, the innermost. There is always exactly one
// such element and it is always the deepest match.
const PRELUDE = `
const byAttr = (attr) => Array.from(document.querySelectorAll("[" + attr + "]"));
const showing = (el, text) => (el.textContent || "").includes(text);

// The deepest element carrying this attribute whose text includes the
// string. Deepest, because every ancestor of a match is also a match.
const target = (attr, text) => {
    const hits = byAttr(attr).filter((el) => showing(el, text));
    if (hits.length === 0) {
        throw new Error("nothing with " + attr + " is showing " + JSON.stringify(text));
    }
    return hits.reduce((best, el) => (best.contains(el) ? el : best));
};

const settle = () => new Promise((done) =>
    requestAnimationFrame(() => requestAnimationFrame(() => done())));

const invoke = async (el, attr, payload) => {
    window.GoInvokeCallback(el.getAttribute(attr), payload);
    await settle();
};

const tap = async (text) =>
    invoke(target("data-listener_on-click", text), "data-listener_on-click", {});

const typeInto = async (hint, value) => {
    const fields = byAttr("data-listener_on-change");
    const el = fields.find((f) => f.placeholder === hint || f.value === hint)
        || fields.find((f) => showing(f, hint));
    if (!el) throw new Error("no field matching " + JSON.stringify(hint));
    await invoke(el, "data-listener_on-change", { value });
};

// The holder of this attribute inside the deepest row showing the text.
//
// \`target\` cannot answer this one and the difference is a checkbox: a
// checkbox carries the toggle listener and has no text of its own, so
// "the element with the listener whose text includes X" matches nothing.
// What a person means by "tick Call Ada" is the box in the row that says
// Call Ada, which is this: the innermost element showing the text that
// contains such a holder, and then the holder.
const within = (attr, text) => {
    const rows = Array.from(document.querySelectorAll("*"))
        .filter((el) => showing(el, text) && el.querySelector("[" + attr + "]"));
    if (rows.length === 0) {
        throw new Error("no element showing " + JSON.stringify(text) +
            " contains anything with " + attr);
    }
    return rows.reduce((best, el) => (best.contains(el) ? el : best))
        .querySelector("[" + attr + "]");
};

const toggle = async (text, on) =>
    invoke(within("data-listener_on-toggle", text), "data-listener_on-toggle",
        { value: on });

const selectTab = async (index) => {
    const strip = byAttr("data-listener_on-tab-change")[0];
    if (!strip) throw new Error("no tab strip on this screen");
    await invoke(strip, "data-listener_on-tab-change", { value: index });
};

// Scroll the nearest scrolling ancestor so the node showing this text sits
// at the top of the frame. Instant rather than smooth: a smooth scroll is
// an animation, and the camera would catch whatever fraction of it had
// happened.
const scrollTo = async (text) => {
    const el = Array.from(document.querySelectorAll("*"))
        .filter((e) => showing(e, text))
        .reduce((best, e) => (best && best.contains(e) ? e : best || e), null);
    if (!el) throw new Error("nothing is showing " + JSON.stringify(text));
    el.scrollIntoView({ block: "start", behavior: "instant" });
    await settle();
};
`;

// --- The shot's own header line ------------------------------------------

// Every action script opens with one line of JSON saying which app it is
// of and how big the frame is:
//
//   // grmob-shot: {"app": "todoapp", "w": 414, "h": 600}
//
// A composite names a page of its own instead of an app, and says what to
// clip to, because it has no bezel:
//
//   // grmob-shot: {"page": "hero.html", "clip": "#hero"}
//
// In the script rather than in a table in shoot.sh, because the size is a
// property of the shot in the same way the taps are: a counter wants a
// short frame and the tutorial wants a tall one, and a table somewhere
// else is one more thing to keep in step with the file it describes.
const HEADER = /^\s*\/\/\s*grmob-shot:\s*(\{.*\})\s*$/m;

function readScript(path) {
    const body = readFileSync(path, "utf8");
    const m = HEADER.exec(body);
    if (!m) {
        throw new Error(`${path} has no "// grmob-shot: {…}" header line, so ` +
            `nothing says which app it drives or how big the frame is.`);
    }
    let head;
    try {
        head = JSON.parse(m[1]);
    } catch (e) {
        throw new Error(`${path}: the grmob-shot header is not JSON: ${e.message}`);
    }
    if (!head.app && !head.page) {
        throw new Error(`${path}: the header names neither an app to mount ` +
            `nor a page of its own`);
    }
    return { head, body };
}

// --- Chrome --------------------------------------------------------------

const DEVTOOLS_PORT_WAIT_MS = 60000;
const CDP_TIMEOUT_MS = 30000;

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

async function devtoolsPort(profile, exit, errLines) {
    const portFile = join(profile, "DevToolsActivePort");
    for (let waited = 0; waited < DEVTOOLS_PORT_WAIT_MS; waited += 100) {
        if (existsSync(portFile)) {
            const first = readFileSync(portFile, "utf8").split("\n")[0].trim();
            if (first) return Number(first);
        }
        // After the file, not before: a Chrome that wrote its port and then
        // exited still left a usable answer behind.
        if (exit.done) break;
        await sleep(100);
    }
    const how = exit.done
        ? `it exited (${exit.signal || exit.code}) without writing one`
        : `it is still running and has written none after ` +
          `${DEVTOOLS_PORT_WAIT_MS / 1000}s`;
    const tail = errLines.length
        ? `\n\nIts last ${errLines.length} line(s) of stderr:\n  ` +
          errLines.join("\n  ")
        : `\n\nIt wrote nothing to stderr, which for a rejected flag or a ` +
          `missing library it would have — so the problem is the launch ` +
          `itself: the binary, the profile directory, or a sandbox.`;
    throw new Error(`Chrome never reported a DevTools port: ${how}.${tail}`);
}

async function connect(wsURL) {
    const ws = new WebSocket(wsURL);
    await new Promise((res, rej) => {
        ws.addEventListener("open", res, { once: true });
        ws.addEventListener("error", rej, { once: true });
    });
    let nextID = 0;
    const pending = new Map();
    ws.addEventListener("message", (e) => {
        const msg = JSON.parse(e.data);
        if (msg.id === undefined) return;
        const entry = pending.get(msg.id);
        if (!entry) return;
        pending.delete(msg.id);
        if (msg.error) entry.reject(new Error(`${entry.method}: ${msg.error.message}`));
        else entry.resolve(msg.result);
    });
    return {
        // Bounded, because a CDP call that never answers is
        // indistinguishable from one still in flight — and without a bound
        // this program's failure mode is a hang, which is what a person
        // waits through instead of reading.
        send(method, params = {}) {
            const id = ++nextID;
            return new Promise((res, rej) => {
                const timer = setTimeout(() => {
                    pending.delete(id);
                    rej(new Error(`${method}: no answer in ${CDP_TIMEOUT_MS}ms`));
                }, CDP_TIMEOUT_MS);
                pending.set(id, {
                    method,
                    resolve: (v) => { clearTimeout(timer); res(v); },
                    reject: (e) => { clearTimeout(timer); rej(e); },
                });
                ws.send(JSON.stringify({ id, method, params }));
            });
        },
        close: () => ws.close(),
    };
}

// --- The static server ---------------------------------------------------

const TYPES = {
    ".html": "text/html", ".js": "text/javascript", ".mjs": "text/javascript",
    ".wasm": "application/wasm", ".png": "image/png", ".css": "text/css",
};

// A directory server, unlike wasm/verify's two-path one, because the page
// needs four files and a composite page needs the finished PNGs beside
// them. Confined to the served directory by resolving and then checking
// the prefix: this serves a build directory to a browser on the same
// machine, and the cost of the check is three lines.
function serve(dir) {
    const rootDir = resolve(dir);
    return http.createServer((req, res) => {
        const path = decodeURIComponent(new URL(req.url, "http://x").pathname);
        const file = resolve(join(rootDir, path === "/" ? "/index.html" : path));
        if (!file.startsWith(rootDir) || !existsSync(file)) {
            res.writeHead(404).end("not found");
            return;
        }
        res.writeHead(200, {
            "content-type": TYPES[extname(file)] || "application/octet-stream",
        });
        res.end(readFileSync(file));
    });
}

// --- main ----------------------------------------------------------------

async function main() {
    const args = parseArgs(process.argv.slice(2));
    const chromePath = CHROME_CANDIDATES.find((p) => existsSync(p));
    if (!chromePath) {
        throw new Error(`no Chrome found. Looked at:\n  ` +
            CHROME_CANDIDATES.join("\n  ") +
            `\n\nSet GRMOB_CHROME to one.`);
    }
    if (typeof WebSocket !== "function") {
        throw new Error(`this Node has no global WebSocket (Node 22+ does), ` +
            `and the DevTools protocol is a WebSocket.`);
    }

    const { head, body } = readScript(resolve(args.script));
    const server = serve(args.dir);
    await new Promise((r) => server.listen(0, "127.0.0.1", r));
    const origin = `http://127.0.0.1:${server.address().port}`;

    const profile = mkdtempSync(join(tmpdir(), "grmob-shots-"));
    const chrome = spawn(chromePath, [
        "--headless=new",
        "--remote-debugging-port=0",
        `--user-data-dir=${profile}`,
        "--no-first-run",
        "--no-default-browser-check",
        "--disable-extensions",
        "--disable-gpu",
        // A scrollbar inside the bezel reads as part of the app rather
        // than as part of the browser, which is a lie the picture tells
        // about every platform this framework renders to.
        "--hide-scrollbars",
        "--force-color-profile=srgb",
        // Larger than any frame a shot asks for, so the bezel is never
        // the thing being clipped by the window.
        "--window-size=1000,1700",
        "about:blank",
    ], { stdio: ["ignore", "ignore", "pipe"] });

    const chromeErr = [];
    chrome.stderr.on("data", (c) => {
        for (const line of String(c).split("\n")) {
            if (line.trim()) chromeErr.push(line.trimEnd());
        }
        if (chromeErr.length > 40) chromeErr.splice(0, chromeErr.length - 40);
    });
    const exit = { code: null, signal: null, done: false };
    chrome.on("exit", (code, signal) => {
        Object.assign(exit, { code, signal, done: true });
    });

    let session = null;
    try {
        const port = await devtoolsPort(profile, exit, chromeErr);
        const targets = await getJSON(`http://127.0.0.1:${port}/json/list`);
        const page = targets.find((t) => t.type === "page");
        if (!page) throw new Error("Chrome opened no page target");
        session = await connect(page.webSocketDebuggerUrl);
        await session.send("Page.enable");
        await session.send("Runtime.enable");

        const evaluate = async (expression) => {
            const r = await session.send("Runtime.evaluate", {
                expression, returnByValue: true, awaitPromise: true,
            });
            if (r.exceptionDetails) {
                throw new Error(`the page threw: ${r.exceptionDetails.text} ` +
                    (r.exceptionDetails.exception?.description || ""));
            }
            return r.result.value;
        };

        // Two shapes of page. Almost every shot is an app in the bezel,
        // named by `app`; a composite is a page of its own that arranges
        // finished PNGs, named by `page`, and has no app in it at all.
        const size = head.w && head.h ? `w=${head.w}&h=${head.h}` : "";
        const url = head.page
            ? `${origin}/${head.page}?${size}`
            : `${origin}/?app=${head.app}&${size}`;
        await session.send("Page.navigate", { url });

        // Poll for the app, not for the load event. See the ready flag in
        // index.html: `load` fires several megabytes before there is
        // anything to photograph.
        //
        // A composite page has no module to wait for and sets the flag
        // itself once its images have decoded, so the same poll serves
        // both and neither needs a branch.
        const deadline = Date.now() + 60000;
        for (;;) {
            const state = await evaluate(
                `({ ready: !!window.__grmobReady, failed: window.__grmobFailed || "" })`);
            if (state.failed) throw new Error(`the page failed to start: ${state.failed}`);
            if (state.ready) break;
            if (Date.now() > deadline) {
                throw new Error(`the app never mounted. main.wasm is several ` +
                    `megabytes; if this is a cold run, try again, and if it ` +
                    `is not, run with --out - and read what the page says.`);
            }
            await sleep(100);
        }

        // The action script, inside the prelude, as the body of an async
        // function so it may await and may return a value.
        const value = await evaluate(
            `(async () => { ${PRELUDE}\n${body}\n })()`);

        // One more settle before the shutter: the last action's patch has
        // been applied, but a transition declared on a node (this
        // framework has core.Transition) is still running.
        await evaluate(`new Promise((d) => setTimeout(d, 250))`);

        if (args.out === "-") {
            // The probe form. This is how the rendered DOM is read at all
            // — the browser-automation tools refuse this page's content —
            // and it is how every claim in internal/shotclaims was checked
            // against a real render before it was written down.
            console.log(JSON.stringify(value ?? null, null, 2));
            return;
        }

        // Clip to the bezel, so the window's size never reaches the file.
        const clip = head.clip || ".phone";
        const rect = await evaluate(
            `(() => { const el = document.querySelector(${JSON.stringify(clip)});
                      if (!el) throw new Error("nothing matches ${clip}");
                      const r = el.getBoundingClientRect();
                      return { x: r.x, y: r.y, width: r.width, height: r.height }; })()`);
        // Rounded before the shutter, and to nearest rather than outward.
        //
        // Two fractions arrive here and only one is about the subject. The
        // bezel's POSITION is fractional because it is centred in a window
        // whose leftover space is odd, which says nothing about the shot;
        // its SIZE is fractional only where the content makes it so, as
        // the composite's three images and their gaps do. Rounding outward
        // would fold the first into the second — a frame asked for at 260
        // came back 261 because it happened to be centred half a pixel
        // down, which is a dimension nobody can reproduce and nobody meant.
        const clipRect = {
            x: Math.round(rect.x),
            y: Math.round(rect.y),
            width: Math.round(rect.width),
            height: Math.round(rect.height),
        };
        const shot = await session.send("Page.captureScreenshot", {
            format: "png",
            captureBeyondViewport: true,
            clip: { ...clipRect, scale: 2 },
        });
        writeFileSync(resolve(args.out), Buffer.from(shot.data, "base64"));
        console.log(`${args.out}  ${clipRect.width}×${clipRect.height} css, ` +
            `${clipRect.width * 2}×${clipRect.height * 2} px`);
    } finally {
        if (session) session.close();
        chrome.kill();
        server.close();
    }
}

main().catch((err) => {
    console.error(String(err && err.stack ? err.stack : err));
    process.exit(1);
});
