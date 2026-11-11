// GrMob.instantiateWasm: how every host page loads main.wasm (N-137 h, the
// plan's R9h).
//
// The pages used to try instantiateStreaming and, on any rejection, retry a
// buffered instantiate from the same response. That was meant for a server
// sending the wrong MIME type, but it also caught every real failure, after
// streaming had already consumed the body, so the page reported "body
// already used" instead of the CompileError or LinkError behind it. What is
// pinned here, with Node's real WebAssembly and real Responses:
//
//   - an application/wasm response streams, and instantiates
//   - any other type is buffered, and instantiates (the fallback still works)
//   - a module that will not compile, or will not link, rejects with that
//     error, on both paths, and not with a complaint about the body

import test from "node:test";
import assert from "node:assert/strict";

import { loadRuntime } from "./load.mjs";

// The smallest valid module: the magic number and version, no sections.
const EMPTY = new Uint8Array([0x00, 0x61, 0x73, 0x6d, 0x01, 0x00, 0x00, 0x00]);

// A module that imports env.f, a function taking nothing and returning
// nothing, so it links only when the import object supplies one. Sections:
// type (one func type ()→()), import ("env" "f" as func of type 0).
const NEEDS_ENV_F = new Uint8Array([
    0x00, 0x61, 0x73, 0x6d, 0x01, 0x00, 0x00, 0x00,
    0x01, 0x04, 0x01, 0x60, 0x00, 0x00,
    0x02, 0x09, 0x01, 0x03, 0x65, 0x6e, 0x76, 0x01, 0x66, 0x00, 0x00,
]);

// A truncated build: the header and half a section.
const TRUNCATED = new Uint8Array([0x00, 0x61, 0x73, 0x6d, 0x01, 0x00, 0x00, 0x00, 0x01, 0x04, 0x01]);

const served = (bytes, type) => new Response(bytes, { headers: { "Content-Type": type } });

// The host realm's WebAssembly: a vm context has V8's, without Node's
// instantiateStreaming. The page passes nothing and gets the browser's.
const load = (response, imports = {}) =>
    loadRuntime().GrMob.instantiateWasm(response, imports, WebAssembly);

test("an application/wasm response streams and instantiates", async () => {
    let streamed = false;
    const spy = {
        instantiateStreaming(r, i) { streamed = true; return WebAssembly.instantiateStreaming(r, i); },
        instantiate(b, i) { return WebAssembly.instantiate(b, i); },
    };
    const result = await loadRuntime().GrMob.instantiateWasm(served(EMPTY, "application/wasm"), {}, spy);
    assert.ok(result.instance instanceof WebAssembly.Instance);
    assert.equal(streamed, true, "an application/wasm response was not streamed");
});

test("any other type is buffered and instantiates", async () => {
    for (const type of ["application/octet-stream", "text/plain; charset=utf-8", ""]) {
        const result = await load(served(EMPTY, type));
        assert.ok(result.instance instanceof WebAssembly.Instance, `type ${JSON.stringify(type)}`);
    }
});

// Only exactly "application/wasm" streams. The Web API spec matches the
// whole value ignoring case and Node matches it case-sensitively, and both
// refuse parameters, so anything else takes the buffered path, which every
// engine accepts. Found by an earlier version of this test, which stripped
// the parameters and lowered the case, and was refused by Node both times
// ("WebAssembly response has unsupported MIME type").
test("any type but exactly application/wasm is buffered, and loads", async () => {
    for (const type of ["Application/Wasm; charset=binary", "application/wasm;x=1", "Application/WASM"]) {
        let streamed = false;
        const spy = {
            instantiateStreaming(r, i) { streamed = true; return WebAssembly.instantiateStreaming(r, i); },
            instantiate(b, i) { return WebAssembly.instantiate(b, i); },
        };
        const result = await loadRuntime().GrMob.instantiateWasm(served(EMPTY, type), {}, spy);
        assert.ok(result.instance instanceof WebAssembly.Instance, type);
        assert.equal(streamed, false, `${type} was streamed`);
    }
    // Surrounding whitespace is not part of the type.
    const result = await load(served(EMPTY, " application/wasm "));
    assert.ok(result.instance instanceof WebAssembly.Instance);
});

test("a module that will not compile rejects with the CompileError, on both paths", async () => {
    for (const type of ["application/wasm", "application/octet-stream"]) {
        await assert.rejects(load(served(TRUNCATED, type)), (err) => {
            assert.equal(err.name, "CompileError", `${type}: got ${err.name}: ${err.message}`);
            return true;
        });
    }
});

test("a module that will not link rejects with the LinkError, on both paths", async () => {
    for (const type of ["application/wasm", "application/octet-stream"]) {
        // The import is present but not a function: a LinkError, the error a
        // wasm_exec.js from another Go release gives.
        await assert.rejects(load(served(NEEDS_ENV_F, type), { env: { f: 1 } }), (err) => {
            assert.equal(err.name, "LinkError", `${type}: got ${err.name}: ${err.message}`);
            return true;
        });
        // And it links when the import is right, so the module is sound.
        const ok = await load(served(NEEDS_ENV_F, type), { env: { f() {} } });
        assert.ok(ok.instance instanceof WebAssembly.Instance);
    }
});

// The shape that hid the errors, kept here to show the test can tell: the
// old pages' expression, given the same truncated module, reports the body
// and not the compile error.
test("the old try-then-retry shape reported the body, not the error", async () => {
    const source = Promise.resolve(served(TRUNCATED, "application/wasm"));
    const old = WebAssembly.instantiateStreaming(source, {})
        .catch(() => source.then((r) => r.arrayBuffer()).then((b) => WebAssembly.instantiate(b, {})));
    await assert.rejects(old, (err) => {
        assert.notEqual(err.name, "CompileError");
        assert.match(String(err.message), /body|used|read|unusable/i);
        return true;
    });
});
