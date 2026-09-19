// The runtime's core.OnStartReached path and its companions: an
// IntersectionObserver over a List's *first* child, re-pointed when a batch
// replaces that child, and the data-key every keyed node carries so a thread
// can find the reader's row again after a prepend.
//
// What a prepend does to scrollTop needs layout, which dom.mjs does not have;
// browser.mjs's check 19 holds that half in Chrome. Here is the bookkeeping.

import test from "node:test";
import assert from "node:assert/strict";

import { loadRuntime, nodeAt } from "./load.mjs";

const keyedRow = (key) => ({ Type: "Text", Key: key, Props: { content: key } });

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

// The start observer, read back the way load.mjs's observedBy reads the end
// one: its current targets.
const startTargets = (list) => (list.__grmobStartObserver ? [...list.__grmobStartObserver.targets] : []);

test("a List with onStartReached observes its first row", () => {
    const { list, row } = mountList({ onStartReached: "cb_1" }, [keyedRow("a"), keyedRow("b")]);
    assert.deepEqual(startTargets(list), [row(0)]);
});

test("the first row coming into view dispatches the callback, void", () => {
    const { rt, row } = mountList({ onStartReached: "cb_1" }, [keyedRow("a"), keyedRow("b")]);
    assert.equal(rt.intersect(row(0)), true);
    assert.deepEqual(rt.dispatched, [{ id: "cb_1", payload: {} }]);
});

test("both edges on one List watch their own rows", () => {
    const { rt, list, row } = mountList({ onStartReached: "cb_1", onEndReached: "cb_2" },
        [keyedRow("a"), keyedRow("b"), keyedRow("c")]);
    assert.deepEqual(startTargets(list), [row(0)]);
    assert.deepEqual(rt.observedBy(list), [row(2)]);
});

// A prepend reaches the runtime as a replace in every slot (the reconciler
// pairs by position): the first row is a new element, and the observer must
// move to it.
test("a replaced first row moves the observation to the new one", () => {
    const { rt, list, row } = mountList({ onStartReached: "cb_1" }, [keyedRow("b"), keyedRow("c")]);
    const before = row(0);
    rt.GrMob.patch(JSON.stringify([
        { Type: "replace", TargetID: "root/0/0", Changes: keyedRow("a") },
    ]));
    assert.notEqual(row(0), before);
    assert.deepEqual(startTargets(list), [row(0)]);
});

test("a List that loses onStartReached is torn down", () => {
    const { rt, list } = mountList({ onStartReached: "cb_1" }, [keyedRow("a")]);
    rt.GrMob.patch(JSON.stringify([{ Type: "update-props", TargetID: "root/0", Changes: {} }]));
    assert.deepEqual(startTargets(list), []);
});

test("a keyed node carries its key as data-key, an unkeyed one none", () => {
    const { rt, row } = mountList({}, [keyedRow("msg:1"), { Type: "Text", Props: { content: "x" } }]);
    assert.equal(row(0).dataset.key, "msg:1");
    assert.equal(row(1).dataset.key, undefined);
    rt.GrMob.patch(JSON.stringify([{ Type: "replace", TargetID: "root/0/1", Changes: keyedRow("msg:2") }]));
    assert.equal(row(1).dataset.key, "msg:2");
});

test("startAtEnd is recorded on the list, and cleared when a patch drops it", () => {
    const { rt, list } = mountList({ startAtEnd: true }, [keyedRow("a")]);
    assert.equal(list.dataset.startAtEnd, "true");
    rt.GrMob.patch(JSON.stringify([{ Type: "update-props", TargetID: "root/0", Changes: { startAtEnd: false } }]));
    assert.equal(list.dataset.startAtEnd, undefined);
});
