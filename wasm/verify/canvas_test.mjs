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
        // The node children are the shapes; a leading <defs> is chrome.
        const gradients = c.gradients ?? [];
        const [defs, ...paths] = gradients.length ? [...svg.children] : [null, ...svg.children];
        assert.equal(paths.length, c.shapes.length);
        if (gradients.length) {
            assert.equal(defs.tagName, "defs");
            assert.equal(defs.getAttribute("data-grmob-chrome"), "gradients");
            assert.equal(defs.getAttribute("data-node-path"), null, "the <defs> is not a node");
            assert.equal(defs.children.length, gradients.length);
            gradients.forEach((g, i) => {
                const server = defs.children[i];
                assert.equal(server.namespaceURI, SVG_NS);
                assert.equal(server.tagName, g.tag);
                assert.deepEqual(pairs(drawn(server)), pairs(g.attrs), `gradient ${i} of ${c.what}`);
                assert.equal(server.children.length, g.stops.length);
                g.stops.forEach((stop, j) => {
                    assert.equal(server.children[j].tagName, "stop");
                    assert.deepEqual(pairs(drawn(server.children[j])), pairs(stop));
                });
            });
        }
        paths.forEach((path, i) => {
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

// Gradient ids follow the slot: a shape added ahead of a gradient shape moves
// its id and its fill reference together, a patch that drops the gradient
// drops the <defs>, and an add-child lands after the chrome.
test("gradient <defs> follows add, update and loss of the gradient", () => {
    const grad = { gradient: "linear", gradientAt: [0, 0, 0, 10], gradientStops: [0, 1], gradientColors: ["#000000", "#ffffff"] };
    const tree = JSON.stringify({
        Type: "Canvas",
        Props: { vw: 10, vh: 10, scale: "fit" },
        Children: [
            { Type: "CanvasShape", Props: { d: [0, 0, 0, 1, 10, 10], ...grad } },
        ],
    });
    const { rt, svg } = mount(tree);
    const defs = () => [...svg.children].filter((el) => el.getAttribute("data-grmob-chrome") === "gradients");
    const shapes = () => [...svg.children].filter((el) => el.getAttribute("data-node-path") !== null);

    assert.equal(defs().length, 1);
    assert.equal(svg.children[0], defs()[0], "the <defs> leads");
    assert.equal(shapes()[0].getAttribute("fill"), "url(#grmob-root-0-fill-0)");
    assert.equal(defs()[0].children[0].getAttribute("id"), "grmob-root-0-fill-0");

    // A second shape added after it: the add lands after the chrome, and
    // the first shape keeps slot 0.
    rt.GrMob.patch(JSON.stringify([{
        Type: "add",
        TargetID: "root/0/1",
        Changes: { Type: "CanvasShape", Props: { d: [0, 0, 0], ...grad, gradientColors: ["#ff0000", "#0000ff"] } },
    }]));
    rt.drainFrames();
    assert.equal(svg.children[0], defs()[0], "the <defs> still leads after an add");
    assert.equal(shapes().length, 2);
    assert.equal(shapes()[1].getAttribute("data-node-path"), "root/0/1");
    assert.equal(shapes()[1].getAttribute("fill"), "url(#grmob-root-0-fill-1)");
    assert.deepEqual([...defs()[0].children].map((g) => g.getAttribute("id")),
        ["grmob-root-0-fill-0", "grmob-root-0-fill-1"]);

    // Losing both gradients to flat fills removes the <defs>.
    rt.GrMob.patch(JSON.stringify([
        { Type: "update-props", TargetID: "root/0/0", Changes: { d: [0, 0, 0], fill: "#123456" } },
        { Type: "update-props", TargetID: "root/0/1", Changes: { d: [0, 0, 0] } },
    ]));
    rt.drainFrames();
    assert.equal(defs().length, 0, "a canvas with no gradient keeps no <defs>");
    assert.equal(shapes()[0].getAttribute("fill"), "#123456");
    assert.equal(shapes()[1].getAttribute("fill"), "none");
});
