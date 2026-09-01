// A minimal DOM for the WASM runtime harness.
//
// The runtime touches 24 distinct DOM members in total (createElement,
// querySelector, activeElement, dataset, style, textContent, value,
// placeholder, src, disabled, checked, rows, children, setAttribute,
// removeAttribute, appendChild, replaceWith, remove, addEventListener, focus,
// blur, tagName, innerHTML, getElementById). That is small enough to
// implement rather than approximate, which is why this file exists instead of
// a jsdom dependency — the same trade ios/verify makes with its hand-written
// Swift harness.
//
// # What this is and is not
//
// It is a faithful model of the *bookkeeping* the runtime depends on:
// attribute storage, the dataset name mapping, parent/child links, listener
// registration and dispatch, and which element holds focus. Those are the
// things a patch can get wrong.
//
// It is not a browser. There is no layout, no CSS cascade, no real keyboard
// and no event bubbling — the runtime attaches every listener directly to the
// element that owns the prop (see mapEventName's note on focus and blur), so
// bubbling is not part of the contract under test. Anything that depends on
// real rendering — whether enterkeyhint actually relabels a soft keyboard,
// whether focus() opens it — is out of reach here and stays out of reach.
//
// Where a behavior is deliberately simplified, the comment says so, because a
// shim that quietly lies is worse than no shim at all.

// dataset property name <-> attribute name, the standard's mapping:
// `nodeType` <-> `data-node-type`. Implemented rather than faked because the
// runtime both writes dataset properties (el.dataset.focusEpoch) and reads
// attributes elsewhere, and a shim that kept them in separate stores would
// hide a real collision.
function datasetToAttr(name) {
    return "data-" + name.replace(/[A-Z]/g, (c) => "-" + c.toLowerCase());
}

function attrToDataset(attr) {
    return attr
        .slice("data-".length)
        .replace(/-([a-z])/g, (_, c) => c.toUpperCase());
}

// makeDataset returns an attribute-backed dataset object. A Proxy rather than
// a plain object so that every read and write goes through the element's
// attribute map — which is what makes `el.dataset.nodeType = "Row"` and
// `el.getAttribute("data-node-type")` two views of one fact, as in a browser.
//
// Values are stringified on write, as the DOM does: the runtime stores an
// integer epoch and compares it back with String() on both sides precisely
// because of this, and a shim that preserved the number would make that
// comparison pass for the wrong reason.
function makeDataset(el) {
    return new Proxy(
        {},
        {
            get(_, name) {
                if (typeof name !== "string") return undefined;
                const v = el.attributes.get(datasetToAttr(name));
                return v === undefined ? undefined : v;
            },
            set(_, name, value) {
                el.attributes.set(datasetToAttr(name), String(value));
                return true;
            },
            has(_, name) {
                return el.attributes.has(datasetToAttr(name));
            },
            deleteProperty(_, name) {
                el.attributes.delete(datasetToAttr(name));
                return true;
            },
            ownKeys() {
                return [...el.attributes.keys()]
                    .filter((a) => a.startsWith("data-"))
                    .map(attrToDataset);
            },
            getOwnPropertyDescriptor() {
                return { enumerable: true, configurable: true };
            },
        }
    );
}

class Element {
    constructor(doc, tag) {
        this.ownerDocument = doc;
        this.tagName = tag.toUpperCase();
        this.attributes = new Map();
        this.children = [];
        this.parentNode = null;
        this.listeners = new Map();
        // style is a plain object: the runtime only ever assigns properties
        // onto it (Object.assign in applyStyle, el.style.height for Spacer),
        // and never reads the cascade back.
        this.style = {};
        this.textContent = "";
        // The form-control properties the runtime sets directly rather than
        // as attributes, exactly as a browser distinguishes them. `value` in
        // particular is a *property* — the runtime's echo bookkeeping depends
        // on reading back what it wrote, not on an attribute.
        this.value = undefined;
        this.placeholder = undefined;
        this.src = undefined;
        this.disabled = false;
        // A checkbox's live state and a textarea's height. Both start
        // undefined rather than at a browser default (false, 2) for the same
        // reason value does: an element the runtime never wrote to must be
        // distinguishable from one it wrote a default into, which is exactly
        // what the conformance replay compares.
        this.checked = undefined;
        this.rows = undefined;
        this.dataset = makeDataset(this);
    }

    setAttribute(name, value) {
        this.attributes.set(name, String(value));
    }

    getAttribute(name) {
        const v = this.attributes.get(name);
        return v === undefined ? null : v;
    }

    hasAttribute(name) {
        return this.attributes.has(name);
    }

    removeAttribute(name) {
        this.attributes.delete(name);
    }

    appendChild(child) {
        // textContent here is a stored string, not the live concatenation of
        // descendant text a browser computes. The runtime only ever sets it
        // on leaf types (Text's content, Button's label), so the two agree —
        // and this guard is what keeps that true: the moment the runtime puts
        // children under a node it has given text to, the simplification has
        // become a lie and says so, rather than quietly diverging.
        if (this.textContent !== "") {
            throw new Error(
                `harness DOM: appendChild onto <${this.tagName.toLowerCase()}> which already has textContent`
            );
        }
        if (child.parentNode) child.parentNode.removeChild(child);
        child.parentNode = this;
        this.children.push(child);
        return child;
    }

    removeChild(child) {
        const i = this.children.indexOf(child);
        if (i >= 0) this.children.splice(i, 1);
        child.parentNode = null;
        return child;
    }

    replaceWith(next) {
        const parent = this.parentNode;
        if (!parent) return;
        const i = parent.children.indexOf(this);
        // Detach the incoming node from wherever it was first, so a replace
        // can never leave one element in two child lists.
        if (next.parentNode) next.parentNode.removeChild(next);
        parent.children[i] = next;
        next.parentNode = parent;
        this.parentNode = null;
        // Focus does not survive the element that held it, which is what the
        // browser does and what makes a "blur" command after a replace a
        // no-op rather than a crash.
        if (this.ownerDocument.activeElement === this) {
            this.ownerDocument.activeElement = null;
        }
    }

    remove() {
        if (this.parentNode) this.parentNode.removeChild(this);
        if (this.ownerDocument.activeElement === this) {
            this.ownerDocument.activeElement = null;
        }
    }

    set innerHTML(v) {
        // The runtime only ever assigns "" (mount clearing its root), so this
        // is the clear, not a parser. Anything else is a harness bug and says
        // so rather than silently doing nothing.
        if (v !== "") throw new Error("harness DOM: innerHTML only supports \"\"");
        for (const c of this.children) c.parentNode = null;
        this.children = [];
    }

    addEventListener(type, fn) {
        if (!this.listeners.has(type)) this.listeners.set(type, []);
        this.listeners.get(type).push(fn);
    }

    // dispatch delivers an event to this element's own listeners only.
    //
    // No bubbling and no capture: the runtime attaches every listener to the
    // element that owns the prop, and says so where it wires focus and blur
    // (the two DOM events that do not bubble). A future move to delegated
    // listeners would break here, loudly, which is the right outcome.
    dispatch(type, init = {}) {
        const event = {
            type,
            target: this,
            currentTarget: this,
            defaultPrevented: false,
            preventDefault() {
                this.defaultPrevented = true;
            },
            ...init,
        };
        for (const fn of this.listeners.get(type) || []) fn(event);
        return event;
    }

    focus() {
        this.ownerDocument.activeElement = this;
    }

    blur() {
        if (this.ownerDocument.activeElement === this) {
            this.ownerDocument.activeElement = null;
        }
    }

    // The two helpers the tests use, not part of the DOM the runtime sees.

    // find walks the subtree for the first element satisfying pred.
    find(pred) {
        if (pred(this)) return this;
        for (const c of this.children) {
            const hit = c.find(pred);
            if (hit) return hit;
        }
        return null;
    }

    // findAll collects every element in the subtree satisfying pred, in
    // document order.
    findAll(pred, out = []) {
        if (pred(this)) out.push(this);
        for (const c of this.children) c.findAll(pred, out);
        return out;
    }
}

class Document {
    constructor() {
        this.activeElement = null;
        this.body = new Element(this, "body");
        this.byId = new Map();
    }

    createElement(tag) {
        return new Element(this, tag);
    }

    // A mount point the harness registers by hand; the runtime only ever
    // looks up the one id it was given.
    getElementById(id) {
        return this.byId.get(id) || null;
    }

    // The one selector shape the runtime uses: [data-node-path="root/1/0"].
    // Parsed rather than pattern-matched loosely, so a runtime that started
    // emitting a different selector fails here instead of matching nothing
    // and silently skipping every patch.
    querySelector(selector) {
        const m = /^\[([a-zA-Z-]+)="(.*)"\]$/.exec(selector);
        if (!m) {
            throw new Error(`harness DOM: unsupported selector ${selector}`);
        }
        const [, attr, value] = m;
        return this.body.find((el) => el.getAttribute(attr) === value);
    }
}

// newDOM returns a document, a window and a drainable animation-frame queue.
//
// requestAnimationFrame is a queue rather than a timer on purpose: the
// runtime defers focus() and blur() by one frame (the initial tree is built
// detached, and focus() on a node outside the document is a silent no-op), so
// a test has to be able to say "now the frame happened" and observe the
// difference. An automatic timer would make that a race.
export function newDOM() {
    const document = new Document();
    const frames = [];
    const window = {
        // Filled in by the harness or a test; the runtime calls it to reach
        // "Go". Left undefined here so an unexpected dispatch is a loud
        // TypeError rather than a silently swallowed event.
        GoInvokeCallback: undefined,
    };
    window.document = document;

    return {
        document,
        window,
        requestAnimationFrame(fn) {
            frames.push(fn);
            return frames.length;
        },
        // drainFrames runs everything queued, including anything queued by a
        // callback while draining, and returns how many ran.
        drainFrames() {
            let ran = 0;
            while (frames.length) {
                const batch = frames.splice(0, frames.length);
                for (const fn of batch) {
                    fn(performance.now());
                    ran++;
                }
            }
            return ran;
        },
        pendingFrames: () => frames.length,
    };
}

export { Element, Document };
