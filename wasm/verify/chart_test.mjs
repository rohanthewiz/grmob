// A chart's hidden data table (core.AccessibilityChart) through the real
// runtime. Go renders real comps charts and computes the table htmlout writes
// for each (gen.go chartCases); this file mounts the same trees and reads the
// runtime's table back, which is what holds chartTableRows and
// CHART_TABLE_STYLE to htmlout.ChartTableRows and htmlout.ChartTableStyle
// across the language boundary.

import test from "node:test";
import assert from "node:assert/strict";

import { loadRuntime, loadTranscript, nodeAt } from "./load.mjs";

function mount(tree) {
    const rt = loadRuntime();
    rt.GrMob.mount(JSON.stringify({ Type: "Column", Children: [JSON.parse(tree)] }));
    rt.drainFrames();
    return { rt, chart: nodeAt(rt.document, "root/0") };
}

// The table the runtime drew, read back into htmlout's three parts.
function readTable(table) {
    const [first, ...rest] = [...table.children];
    const hasCaption = first && first.tagName === "CAPTION";
    const [thead, tbody] = hasCaption ? rest : [first, ...rest];
    const cells = (tr) => [...tr.children].map((c) => ({
        tag: c.tagName, scope: c.getAttribute("scope"), text: c.textContent,
    }));
    return {
        caption: hasCaption ? first.textContent : "",
        head: cells(thead.children[0]),
        body: [...tbody.children].map(cells),
    };
}

const { charts } = loadTranscript();

test("the transcript carries chart cases", () => {
    assert.ok(charts && charts.length > 0, "gen.go produced no chart cases");
});

for (const c of charts ?? []) {
    test(`chart table matches htmlout: ${c.what}`, () => {
        const { chart } = mount(c.tree);
        // A figure, not an img: an img's children are presentational, and
        // the table would be unreachable inside one.
        assert.equal(chart.getAttribute("role"), "figure");
        assert.ok(chart.getAttribute("aria-label"), "the figure keeps the summary as its name");

        const table = chart.children[0];
        assert.equal(table.tagName, "TABLE", "the table leads the chart's children");
        assert.equal(table.getAttribute("data-grmob-chrome"), "charttable");
        assert.equal(table.getAttribute("data-node-path"), null, "the table is not a node");
        assert.equal(table.getAttribute("style"), c.style, "htmlout.ChartTableStyle, restated");

        const got = readTable(table);
        assert.equal(got.caption, c.caption);
        assert.deepEqual(got.head.map((h) => h.text), c.head);
        assert.ok(got.head.every((h) => h.tag === "TH" && h.scope === "col"));
        assert.deepEqual(got.body.map((r) => r.map((cell) => cell.text)), c.body);
        for (const row of got.body) {
            assert.equal(row[0].tag, "TH");
            assert.equal(row[0].scope, "row");
            assert.ok(row.slice(1).every((cell) => cell.tag === "TD"));
        }

        // The chart's own children keep their paths, one slot past the table.
        const kids = JSON.parse(c.tree).Children ?? [];
        kids.forEach((_, i) => {
            assert.equal(chart.children[i + 1].getAttribute("data-node-path"), `root/0/${i}`);
        });
    });
}

test("a chart that loses its data loses the table and the figure role", () => {
    const c = charts[0];
    const { rt, chart } = mount(c.tree);
    const node = JSON.parse(c.tree);
    const props = { ...node.Props };
    delete props.chartData;

    rt.GrMob.patch(JSON.stringify([{ Type: "update-props", TargetID: "root/0", Changes: props }]));
    assert.equal(chart.getAttribute("role"), "img");
    assert.notEqual(chart.children[0].getAttribute("data-grmob-chrome"), "charttable");

    // And gains them back, once, with an update-style in between that must
    // not reset the role to the Style's img.
    rt.GrMob.patch(JSON.stringify([
        { Type: "update-props", TargetID: "root/0", Changes: node.Props },
        { Type: "update-style", TargetID: "root/0", Changes: node.Style },
    ]));
    assert.equal(chart.getAttribute("role"), "figure");
    const tables = [...chart.children].filter((el) => el.getAttribute("data-grmob-chrome") === "charttable");
    assert.equal(tables.length, 1, "re-applying the data replaces the table rather than adding one");
});

test("an add patch lands past the table", () => {
    const c = charts[0];
    const { rt, chart } = mount(c.tree);
    const n = (JSON.parse(c.tree).Children ?? []).length;
    rt.GrMob.patch(JSON.stringify([{
        Type: "add", TargetID: `root/0/${n}`,
        Changes: { Type: "Text", Props: { content: "added" } },
    }]));
    const last = chart.children[chart.children.length - 1];
    assert.equal(last.getAttribute("data-node-path"), `root/0/${n}`);
    assert.equal(chart.children[0].getAttribute("data-grmob-chrome"), "charttable");
});
