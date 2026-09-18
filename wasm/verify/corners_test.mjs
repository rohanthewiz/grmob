// core.CornerRadii on the web runtime: four values in CSS's order when the
// node names its corners, the one radius otherwise, and nothing when both go
// back to zero (the totality rule: an update-style patch carries the whole
// Style, so a field returning to zero must clear what it drew).

import test from "node:test";
import assert from "node:assert/strict";

import { loadRuntime, nodeAt } from "./load.mjs";

function mountBox(style) {
    const rt = loadRuntime();
    rt.GrMob.mount(JSON.stringify({ Type: "Column", Children: [{ Type: "Box", Style: style }] }));
    return { rt, el: () => nodeAt(rt.document, "root/0") };
}

test("named corners draw four values and replace the one radius", () => {
    const { el } = mountBox({ BorderRadius: 8, Corners: { TopLeft: 12, BottomLeft: 12 } });
    assert.equal(el().style.borderRadius, "12px 0px 0px 12px");
});

test("without corners the one radius draws", () => {
    const { el } = mountBox({ BorderRadius: 8 });
    assert.equal(el().style.borderRadius, "8px");
});

test("corners returning to zero fall back, then clear", () => {
    const { rt, el } = mountBox({ BorderRadius: 8, Corners: { TopRight: 4 } });
    rt.GrMob.patch(JSON.stringify([{ Type: "update-style", TargetID: "root/0", Changes: { BorderRadius: 8 } }]));
    assert.equal(el().style.borderRadius, "8px");
    rt.GrMob.patch(JSON.stringify([{ Type: "update-style", TargetID: "root/0", Changes: {} }]));
    assert.equal(el().style.borderRadius, "");
});
