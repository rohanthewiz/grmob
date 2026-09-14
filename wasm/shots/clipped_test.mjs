// CLIPPED, held to pages built to fail, in a real Chrome.
//
// Every committed shot passes the check, so the shots themselves cannot show
// that it fires. These pages can: each puts a claimed string where one rule
// should catch it — past the bottom of the frame, under a scroller's bottom
// edge, past a field's right edge — beside a control that is whole, so a check
// that reported everything would fail here too.
//
// Needs a Chrome (or GRMOB_CHROME) and a Node with a global WebSocket, like
// shot.mjs; without one the file skips rather than fails, as browser.mjs does.
//
//	node --test wasm/shots/clipped_test.mjs
//
// # Why a launcher of its own
//
// shot.mjs runs main() when it is imported, so its Chrome helpers cannot be
// borrowed without taking a picture. The launch below is the same flags with
// the waiting reduced to what a blank page needs.

import test from "node:test";
import assert from "node:assert/strict";
import { spawn } from "node:child_process";
import { existsSync, mkdtempSync, readFileSync, rmSync } from "node:fs";
import { tmpdir } from "node:os";
import { join } from "node:path";

import { CLIPPED } from "./clipped.mjs";

// The same candidates shot.mjs and wasm/verify/browser.mjs look in.
const CHROME = [
    process.env.GRMOB_CHROME,
    "/Applications/Google Chrome.app/Contents/MacOS/Google Chrome",
    "/Applications/Chromium.app/Contents/MacOS/Chromium",
    "/usr/bin/google-chrome",
    "/usr/bin/chromium",
    "/usr/bin/chromium-browser",
].filter(Boolean).find((p) => existsSync(p));

const sleep = (ms) => new Promise((r) => setTimeout(r, ms));

// launch starts one headless Chrome for the whole file and returns a
// `check(html, shows)` that loads the page and runs CLIPPED against ".phone".
async function launch(t) {
    const profile = mkdtempSync(join(tmpdir(), "grmob-clipped-test-"));
    const chrome = spawn(CHROME, [
        "--headless=new",
        "--remote-debugging-port=0",
        `--user-data-dir=${profile}`,
        "--no-first-run",
        "--no-default-browser-check",
        "--disable-extensions",
        "--disable-gpu",
        "--hide-scrollbars",
        "--window-size=800,700",
        "about:blank",
    ], { stdio: "ignore" });
    t.after(async () => {
        chrome.kill();
        await sleep(200);
        rmSync(profile, { recursive: true, force: true });
    });

    const portFile = join(profile, "DevToolsActivePort");
    for (let i = 0; i < 300 && !existsSync(portFile); i++) await sleep(100);
    const port = readFileSync(portFile, "utf8").split("\n")[0].trim();
    let page;
    for (let i = 0; i < 100 && !page; i++) {
        const list = await (await fetch(`http://127.0.0.1:${port}/json/list`)).json();
        page = list.find((x) => x.type === "page");
        if (!page) await sleep(100);
    }

    const ws = new WebSocket(page.webSocketDebuggerUrl);
    await new Promise((res, rej) => {
        ws.addEventListener("open", res, { once: true });
        ws.addEventListener("error", rej, { once: true });
    });
    t.after(() => ws.close());
    let nextID = 0;
    const pending = new Map();
    ws.addEventListener("message", (e) => {
        const msg = JSON.parse(e.data);
        const done = pending.get(msg.id);
        if (!done) return;
        pending.delete(msg.id);
        done(msg);
    });
    const send = (method, params = {}) => new Promise((res) => {
        const id = ++nextID;
        pending.set(id, res);
        ws.send(JSON.stringify({ id, method, params }));
    });
    const evaluate = async (expression) => {
        const { result } = await send("Runtime.evaluate", {
            expression, awaitPromise: true, returnByValue: true,
        });
        if (result.exceptionDetails) throw new Error(result.exceptionDetails.text);
        return result.result.value;
    };

    return async (html, shows) => {
        // document.write rather than a navigation: no load event to wait for,
        // and every case starts from the same blank document.
        await evaluate(`document.open(); document.write(${JSON.stringify(html)}); document.close();
            new Promise((d) => requestAnimationFrame(() => requestAnimationFrame(d)))`);
        return evaluate(`(${CLIPPED})(${JSON.stringify(shows)}, ".phone")`);
    };
}

// The bezel: 300×200 at the page's origin, overflow visible — the frame is a
// rect the screenshot is cut to, not an element that clips, so the first rule
// ("outside the frame") is the only one that can catch text below it.
const page = (body) => `<!doctype html><html><head><style>
  body { margin: 0; font: 16px/20px sans-serif; }
  .phone { position: absolute; left: 0; top: 0; width: 300px; height: 200px; }
  .at { position: absolute; left: 10px; margin: 0; }
  input, textarea { font: 16px sans-serif; padding: 2px 4px; border: 1px solid #888; box-sizing: border-box; }
</style></head><body><div class="phone">${body}</div></body></html>`;

test("CLIPPED against pages built to fail", { skip: !CHROME && "no Chrome to launch" }, async (t) => {
    if (typeof WebSocket !== "function") {
        t.skip("this Node has no global WebSocket (Node 22 or later)");
        return;
    }
    const check = await launch(t);

    await t.test("a string whole inside the frame is not reported", async () => {
        const got = await check(page(`<p class="at" style="top:10px">Whole line</p>`), ["Whole line"]);
        assert.deepEqual(got, []);
    });

    await t.test("a string cut by the bottom of the frame is reported", async () => {
        // 20px line box from 190 to 210: the top half is in the picture.
        const got = await check(page(
            `<p class="at" style="top:10px">Whole line</p>` +
            `<p class="at" style="top:190px">Tail line</p>`), ["Whole line", "Tail line"]);
        assert.deepEqual(got, [`"Tail line": outside the frame`]);
    });

    await t.test("a string wholly below the frame is reported", async () => {
        const got = await check(page(`<p class="at" style="top:400px">Below</p>`), ["Below"]);
        assert.deepEqual(got, [`"Below": outside the frame`]);
    });

    await t.test("a string under a scroller's bottom edge is reported by the scroller", async () => {
        const got = await check(page(
            `<div data-node-type="scroll" style="position:absolute;top:0;left:0;width:300px;height:100px;overflow:auto">` +
            `<p style="margin:0;height:90px">Top</p><p style="margin:0">Under the edge</p></div>`),
            ["Top", "Under the edge"]);
        assert.deepEqual(got, [`"Under the edge": cut vertically by its scroll`]);
    });

    await t.test("a field's value is checked by its text: the head passes, the hidden tail does not", async () => {
        const html = page(
            `<input style="position:absolute;top:10px;left:10px;width:120px" ` +
            `value="Alpha beta gamma delta epsilon">`);
        assert.deepEqual(await check(html, ["Alpha"]), []);
        assert.deepEqual(await check(html, ["epsilon"]), [`"epsilon": cut sideways by its input`]);
    });

    await t.test("a field scrolled to its end shows the tail and hides the head", async () => {
        // The script is a block, not top-level declarations: document.open
        // keeps the same window, so a second write of this page would
        // redeclare a top-level `const f`, throw, and leave the field
        // unscrolled — which made "Alpha" read as whole on the second check.
        const html = page(
            `<input id="f" style="position:absolute;top:10px;left:10px;width:120px" ` +
            `value="Alpha beta gamma delta epsilon">` +
            `<script>{ const f = document.getElementById("f"); f.scrollLeft = f.scrollWidth; }</script>`);
        assert.deepEqual(await check(html, ["epsilon"]), []);
        assert.deepEqual(await check(html, ["Alpha"]), [`"Alpha": cut sideways by its input`]);
    });

    await t.test("a placeholder behind a typed value is not shown", async () => {
        const html = page(
            `<input style="position:absolute;top:10px;left:10px;width:200px" placeholder="Email" value="ada@example.com">`);
        assert.deepEqual(await check(html, ["ada@example.com"]), []);
        assert.deepEqual(await check(html, ["Email"]), [`"Email": not in the page`]);
    });

    await t.test("an empty field's placeholder is checked like a value", async () => {
        const html = page(
            `<input style="position:absolute;top:10px;left:10px;width:200px" placeholder="Email">`);
        assert.deepEqual(await check(html, ["Email"]), []);
    });

    await t.test("a field cut by the bottom of the frame is reported", async () => {
        const html = page(
            `<input style="position:absolute;top:185px;left:10px;width:200px" value="Low field">`);
        assert.deepEqual(await check(html, ["Low field"]), [`"Low field": outside the frame`]);
    });

    await t.test("a letter-spaced value is measured with its spacing", async () => {
        // Unspaced, "Alpha beta gamma" fits this field with room to spare;
        // 6px after each of its 16 characters pushes "gamma" past the edge.
        const html = page(
            `<input style="position:absolute;top:10px;left:10px;width:160px;letter-spacing:6px" ` +
            `value="Alpha beta gamma">`);
        assert.deepEqual(await check(html, ["Alpha"]), []);
        assert.deepEqual(await check(html, ["gamma"]), [`"gamma": cut sideways by its input`]);
    });

    await t.test("a right-to-left field is checked from its right edge", async () => {
        // The value starts at the right: its first word is whole and its last
        // runs off the left edge.
        const html = page(
            `<input dir="rtl" style="position:absolute;top:10px;left:10px;width:120px" ` +
            `value="אלפא בטא גמא דלתא אפסילון זטא">`);
        assert.deepEqual(await check(html, ["אלפא"]), []);
        assert.deepEqual(await check(html, ["זטא"]), [`"זטא": cut sideways by its input`]);
    });

    await t.test("a right-to-left field scrolled to its end shows the tail", async () => {
        // scrollLeft runs negative in a right-to-left scroller.
        const html = page(
            `<input id="r" dir="rtl" style="position:absolute;top:10px;left:10px;width:120px" ` +
            `value="אלפא בטא גמא דלתא אפסילון זטא">` +
            `<script>{ const f = document.getElementById("r"); f.scrollLeft = -f.scrollWidth; }</script>`);
        assert.deepEqual(await check(html, ["זטא"]), []);
        assert.deepEqual(await check(html, ["אלפא"]), [`"אלפא": cut sideways by its input`]);
    });

    await t.test("a right-to-left field with mixed-direction text keeps the box rule", async () => {
        // Digits run left to right inside the value, so where "זטא" lands is
        // not arithmetic; the whole field is inside the frame, and that is
        // all the box rule asks.
        const html = page(
            `<input dir="rtl" style="position:absolute;top:10px;left:10px;width:120px" ` +
            `value="אלפא 2026 גמא דלתא אפסילון זטא">`);
        assert.deepEqual(await check(html, ["זטא"]), []);
    });

    await t.test("a textarea keeps the box rule", async () => {
        const inside = page(
            `<textarea style="position:absolute;top:10px;left:10px;width:200px;height:60px">Some notes</textarea>`);
        assert.deepEqual(await check(inside, ["Some notes"]), []);
        const cut = page(
            `<textarea style="position:absolute;top:170px;left:10px;width:200px;height:60px">Some notes</textarea>`);
        // Two reasons, both true: the textarea's own text node has no line
        // boxes (the control paints its value), and the control's box runs
        // past the frame.
        assert.deepEqual(await check(cut, ["Some notes"]), [`"Some notes": not laid out; outside the frame`]);
    });
});
