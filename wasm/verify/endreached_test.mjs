// The runtime's core.OnEndReached path: an IntersectionObserver over a
// List's last child, re-pointed whenever a patch changes what that child is.
//
// The interesting behaviour here is not the first observation — it is what
// happens when the list *grows*, because the observation target moves with
// every appended page and nothing in the patch stream says so. A runtime that
// observed once at creation would report the edge exactly once per screen and
// then go quiet forever, which reads as "the feed stops after page two" and
// looks like a Go bug.
//
// The harness's IntersectionObserver is a declaration, not a measurement:
// there is no layout here, so a test says "this row came into view" and the
// runtime's own bookkeeping decides whether anything was listening. See
// load.mjs.

import test from "node:test";
import assert from "node:assert/strict";

import { loadRuntime, nodeAt } from "./load.mjs";

// Named textRow, not row: every test below destructures a `row` accessor
// out of mountList, and a module-level `row` would sit in its temporal dead
// zone for the whole test body.
const textRow = (label) => ({ Type: "Text", Props: { content: label } });

// mountList renders a List with the given props and rows, and returns the
// runtime, the list element, and a row accessor.
function mountList(props, rows) {
    const rt = loadRuntime();
    rt.GrMob.mount(JSON.stringify({
        Type: "Column",
        Children: [{ Type: "List", Props: props, Children: rows }],
    }));
    rt.drainFrames();
    return {
        rt,
        list: nodeAt(rt.document, "root/0"),
        row: (i) => nodeAt(rt.document, `root/0/${i}`),
    };
}

test("a List with onEndReached observes its last row", () => {
    const { rt, list, row } = mountList({ onEndReached: "cb_0" }, [textRow("a"), textRow("b"), textRow("c")]);

    assert.deepEqual(rt.observedBy(list), [row(2)],
        "the observation must be on the last child, not the first or the list itself");
});

test("the last row coming into view dispatches the callback", () => {
    const { rt, list, row } = mountList({ onEndReached: "cb_0" }, [textRow("a"), textRow("b")]);

    assert.equal(rt.intersect(row(1)), true, "nothing was observing the last row");
    // Void payload: Go registered this through the plain callback channel, so
    // a {value} envelope would route it to the text map where the ID does not
    // exist and the handler would silently never run.
    assert.deepEqual(rt.dispatched, [{ id: "cb_0", payload: {} }]);
    assert.ok(list);
});

test("a row that is not the last one reports nothing", () => {
    const { rt, row } = mountList({ onEndReached: "cb_0" }, [textRow("a"), textRow("b"), textRow("c")]);

    assert.equal(rt.intersect(row(0)), false);
    assert.deepEqual(rt.dispatched, []);
});

test("a List without the prop observes nothing", () => {
    const { rt, list } = mountList({}, [textRow("a")]);

    assert.deepEqual(rt.observedBy(list), []);
});

test("an appended page moves the observation to the new last row", () => {
    const { rt, list } = mountList({ onEndReached: "cb_0" }, [textRow("a"), textRow("b")]);

    rt.GrMob.patch(JSON.stringify([
        { Type: "add-child", TargetID: "root/0", Changes: textRow("c") },
    ]));

    const appended = nodeAt(rt.document, "root/0/2");
    assert.deepEqual(rt.observedBy(list), [appended],
        "the observer still watches the old last row, so the next page is never asked for");
    assert.equal(rt.intersect(appended), true);
    assert.deepEqual(rt.dispatched, [{ id: "cb_0", payload: {} }]);
});

test("a patch aimed at a row re-points the list's observer", () => {
    // The patch target is a *row*, not the list, so the post-batch pass has
    // to walk up the ancestor chain to find the container whose last child
    // may have changed — the same walk syncTouchedTabViews makes.
    const { rt, list } = mountList({ onEndReached: "cb_0" }, [textRow("a"), textRow("b")]);

    rt.GrMob.patch(JSON.stringify([
        { Type: "replace", TargetID: "root/0/1", Changes: textRow("b (edited)") },
    ]));

    const replaced = nodeAt(rt.document, "root/0/1");
    assert.deepEqual(rt.observedBy(list), [replaced],
        "the observer still watches the element the replace detached");
});

test("a List that loses onEndReached is torn down", () => {
    const { rt, list } = mountList({ onEndReached: "cb_0" }, [textRow("a"), textRow("b")]);

    // An update-props patch carries the WHOLE new props map, so an absent key
    // means the prop is gone rather than unchanged: pruneStaleListeners drops
    // the ID and the post-batch pass has to notice the observer it left.
    rt.GrMob.patch(JSON.stringify([
        { Type: "update-props", TargetID: "root/0", Changes: {} },
    ]));

    assert.deepEqual(rt.observedBy(list), []);
});

test("a refreshed callback ID is the one that fires", () => {
    // IDs are positional and a pass can hand this list a different one; the
    // observer re-reads the ID at fire time rather than closing over it.
    const { rt, row } = mountList({ onEndReached: "cb_0" }, [textRow("a"), textRow("b")]);

    rt.GrMob.patch(JSON.stringify([
        { Type: "update-props", TargetID: "root/0", Changes: { onEndReached: "cb_7" } },
    ]));

    rt.intersect(row(1));
    assert.deepEqual(rt.dispatched, [{ id: "cb_7", payload: {} }]);
});

test("an empty List never reports the edge", () => {
    // There is no last row to have reached. The first page is the app's to
    // ask for, and this matches what the other three renderers do for free.
    const { rt, list } = mountList({ onEndReached: "cb_0" }, []);

    assert.deepEqual(rt.observedBy(list), []);
    assert.deepEqual(rt.dispatched, []);
});
