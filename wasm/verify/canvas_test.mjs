// core.Canvas through the real runtime: an <svg> in the SVG namespace holding
// one <path> per shape, with attributes that must be exactly the ones htmlout
// exports. Go computes htmlout's answers for each case (gen.go canvasCases);
// this file computes the runtime's from the same tree and compares, which is
// what holds canvasPathData and canvasShapeAttrs to PathData and
// CanvasShapeAttrs across the language boundary.

import test from "node:test";
import assert from "node:assert/strict";

import { loadRuntime, loadTranscript, nodeAt } from "./load.mjs";

const SVG_NS = "http://www.w3.org/2000/svg";

function mount(tree) {
    const rt = loadRuntime();
    rt.GrMob.mount(JSON.stringify({ Type: "Column", Children: [JSON.parse(tree)] }));
    rt.drainFrames();
    return { rt, svg: nodeAt(rt.document, "root/0") };
}

// The attributes a path carries, minus the runtime's own bookkeeping
// (data-node-path, data-node-type), as ordered name/value pairs.
const drawn = (el) => [...el.attributes.entries()]
    .filter(([name]) => !name.startsWith("data-"))
    .flat();

// Order is compared as a set of pairs: attribute order is not observable in
// a browser, and the runtime removes stale ones before writing new ones.
const pairs = (flat) => {
    const out = new Map();
    for (let i = 0; i + 1 < flat.length; i += 2) out.set(flat[i], flat[i + 1]);
    return out;
};

const { canvases } = loadTranscript();

test("the transcript carries canvas cases", () => {
    assert.ok(canvases && canvases.length > 0, "gen.go produced no canvas cases");
});

for (const c of canvases ?? []) {
    test(`canvas matches htmlout: ${c.what}`, () => {
        const { svg } = mount(c.tree);
        assert.equal(svg.namespaceURI, SVG_NS, "the canvas is an SVG element");
        assert.equal(svg.tagName, "svg");
        assert.equal(svg.getAttribute("viewBox"), c.viewBox);
        assert.equal(svg.getAttribute("preserveAspectRatio"), c.preserve);
        assert.equal(svg.style.display, "block");
        assert.equal(svg.style.width, "100%");
        assert.equal(svg.children.length, c.shapes.length);
        svg.children.forEach((path, i) => {
            assert.equal(path.namespaceURI, SVG_NS, `shape ${i} is an SVG element`);
            assert.equal(path.tagName, "path");
            const attrs = new Map([...pairs(drawn(path))]
                .filter(([name]) => !name.startsWith("aria-") && name !== "role"));
            assert.deepEqual(attrs, pairs(c.shapes[i]), `shape ${i} of ${c.what}`);
        });
    });
}

test("an update-props on a shape rewrites its paint and drops what it lost", () => {
    const tree = JSON.stringify({
        Type: "Canvas",
        Props: { vw: 10, vh: 10, scale: "fit" },
        Children: [
            { Type: "CanvasShape", Props: { d: [0, 0, 0, 1, 10, 10], stroke: "#000", strokeWidth: 2, cap: "round", dash: [2, 1] } },
            { Type: "CanvasShape", Props: { d: [0, 5, 5, 3], fill: "#f00" } },
        ],
    });
    const { rt, svg } = mount(tree);
    const untouched = svg.children[1];

    rt.GrMob.patch(JSON.stringify([{
        Type: "update-props",
        TargetID: "root/0/0",
        Changes: { d: [0, 0, 10, 1, 10, 0], fill: "#0f0" },
    }]));
    rt.drainFrames();

    const line = svg.children[0];
    assert.equal(line.getAttribute("d"), "M0 10L10 0");
    assert.equal(line.getAttribute("fill"), "#0f0");
    for (const gone of ["stroke", "stroke-width", "vector-effect", "stroke-linecap", "stroke-dasharray"]) {
        assert.equal(line.getAttribute(gone), null, `${gone} survived losing the stroke`);
    }
    assert.equal(svg.children[1], untouched, "the other shape is the same element");
});

test("an update-props on the canvas moves its viewBox, scale and aspect ratio together", () => {
    const { rt, svg } = mount(JSON.stringify({ Type: "Canvas", Props: { vw: 100, vh: 50, scale: "fit" } }));
    assert.equal(svg.style.aspectRatio, "100 / 50");

    rt.GrMob.patch(JSON.stringify([{
        Type: "update-props",
        TargetID: "root/0",
        Changes: { vw: 200, vh: 40, scale: "stretch" },
    }]));
    rt.drainFrames();

    assert.equal(svg.getAttribute("viewBox"), "0 0 200 40");
    assert.equal(svg.getAttribute("preserveAspectRatio"), "none");
    assert.equal(svg.style.aspectRatio, "200 / 40");
});
