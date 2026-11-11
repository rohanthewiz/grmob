// GrMob.watchExit: the "stopped, reload" banner (N-131, the plan's R9a).
//
// A Go program that dies on its own (a deadlock, a panic nothing recovered)
// used to leave the page on screen looking alive, with every tap throwing
// "Go program has already exited" into a console nobody had open. The page
// hands the runtime go.run's promise and the runtime draws a banner when it
// settles. What is worth pinning, since no Go test can see the page:
//
//   - a program that exits on its own gets the banner, once, as an alert
//   - a Shutdown somebody asked for (the hot-reload client) draws nothing,
//     and Shutdown still reaches Go
//   - nothing is drawn before the program has actually exited

import test from "node:test";
import assert from "node:assert/strict";

import { loadRuntime } from "./load.mjs";

const settle = () => new Promise((resolve) => setImmediate(resolve));

// The banner is the one child of <body> that is not the mount point.
const banners = (rt) => rt.document.body.children.filter((c) => c.id === "grmob-stopped");

// exitable stands in for go.run's promise and for the GrMobWASM object Go
// publishes, whose Shutdown makes main return and so settles the promise.
function exitable(rt) {
    let exit;
    const exited = new Promise((resolve) => { exit = resolve; });
    const shutdowns = [];
    rt.window.GrMobWASM = {
        Shutdown(...args) {
            shutdowns.push(args);
            exit();
        },
    };
    return { exited, exit, shutdowns };
}

test("a program that exits on its own gets one banner, as an alert", async () => {
    const rt = loadRuntime();
    const go = exitable(rt);
    rt.GrMob.watchExit(go.exited);

    await settle();
    assert.equal(banners(rt).length, 0, "nothing is drawn while the program runs");

    go.exit();
    await settle();
    const [bar] = banners(rt);
    assert.ok(bar, "no banner after the program exited");
    assert.equal(bar.getAttribute("role"), "alert", "a screen reader hears it without hunting for it");
    assert.equal(bar.style.position, "fixed", "over the page, not inside the dead tree");
    assert.match(bar.children[0].textContent, /stopped/);
    assert.equal(bar.children[1].tagName, "BUTTON");
    assert.equal(bar.children[1].textContent, "Reload");

    // A second module that dies too (a hot reload into a broken build) does
    // not stack a second banner.
    const again = exitable(rt);
    rt.GrMob.watchExit(again.exited);
    again.exit();
    await settle();
    assert.equal(banners(rt).length, 1);
});

test("a Shutdown somebody asked for draws nothing, and still reaches Go", async () => {
    const rt = loadRuntime();
    const go = exitable(rt);
    rt.GrMob.watchExit(go.exited);

    // What serve/devclient.js does: call Shutdown, then await the exit.
    rt.window.GrMobWASM.Shutdown("from the reload client");
    await go.exited;
    await settle();

    assert.deepEqual(go.shutdowns, [["from the reload client"]], "the wrapper passed the call through");
    assert.equal(banners(rt).length, 0, "a hot reload is not a death");
});

test("a page whose Go publishes no Shutdown still gets the banner", async () => {
    const rt = loadRuntime();
    rt.window.GrMobWASM = {};
    rt.GrMob.watchExit(Promise.resolve());
    await settle();
    assert.equal(banners(rt).length, 1);
});
