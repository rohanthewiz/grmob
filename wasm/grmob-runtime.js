// grmob-runtime.js

const GrMob = (() => {
    let rootElement = null;
    const DEBUG = true;

    function renderNode(node, path = "") {
        const el = createElement(node);
        el.setAttribute("data-node-path", path);


        if (node.Type === "Spacer" && node.Props && node.Props.size) {
            el.style.height = `${node.Props.size}px`;
        }

        if (node.Children) {
            node.Children.forEach((child, i) => {
                const childEl = renderNode(child, `${path}/${i}`);
                el.appendChild(childEl);
            });
        }

        return el;
    }

    function createElement(node) {
        const el = document.createElement(tagForType(node.Type));
        // The Go node type, kept on the element because the tag alone cannot
        // recover it (Row, Column, Card and Box are all divs) and update-style
        // patches carry only the changed Style — see the patch handler.
        el.dataset.nodeType = node.Type;

        // The <input> variant, which the tag alone cannot express: tagForType
        // sends four Go node types to <input>, and an <input> with no type
        // attribute is a text box. Without this a Checkbox drew as a text
        // field and its state had nowhere to appear at all.
        //
        // Set once here and never on the update path, because a node type
        // cannot change under a patch — the reconciler emits a replace for a
        // changed type (reconcile/patch.go), so the element that carries a
        // given type is always one this function built.
        const inputType = inputTypeFor(node.Type);
        if (inputType) {
            el.setAttribute("type", inputType);
        }

        if (node.Style) {
            applyStyle(el, node.Style, node.Type);
        }

        if (node.Props) {
            for (const [key, value] of Object.entries(node.Props)) {
                if (key.startsWith("on")) {
                    const event = mapEventName(key);
                    el.dataset[`listener_${key}`] = value;
                    if (!el.dataset[`has_listener_${key}`]) {
                        el.dataset[`has_listener_${key}`] = "true";
                        el.addEventListener(event, (e) => {
                            const latestCbId = el.dataset[`listener_${key}`];
                            if (latestCbId && eventQualifies(key, e)) {
                                const payload = extractEventPayload(e, node.Type);
                                window.GoInvokeCallback(latestCbId, payload);
                            }
                        });
                    }
                } else if (key === "value") {
                    el.value = value;
                } else if (key === "placeholder") {
                    el.placeholder = value;
                } else if (key === "content") {
                    el.textContent = value;
                }
                else if (key === "label") {
                    el.textContent = value;
                }
                else if (key === "checked") {
                    // The property, not the attribute. A `checked` attribute
                    // is only the control's *default* state, which the
                    // browser stops consulting the moment the user touches
                    // the box; the property is the live state, and the live
                    // state is what Go is describing.
                    el.checked = !!value;
                }
                else if (key === "rows") {
                    applyRows(el, value);
                }
                else if (key === "src" && node.Type === "Image") {
                    el.src = value;
                }
                else if (key === "contentMode" && node.Type === "Image") {
                    el.style.objectFit = objectFitFor(value);
                }
                else if (key === "focusEpoch") {
                    // Recorded so the update path below can tell a genuinely
                    // new command from a props patch that merely happened to
                    // carry the same stamp along (an update-props patch sends
                    // the whole new map, not just the keys that changed).
                    el.dataset.focusEpoch = value;
                    applyFocusCommand(el, value, node.Props.focusAction);
                }

            }
            // After the loop, not inside it: the hint is a function of two
            // props (imeAction and onSubmit) and Object.entries fixes no order
            // between them, so deciding it per key would depend on which one
            // the map happened to yield first.
            applyEnterKeyHint(el, node.Props);
        }

        return el;
    }

    // Applies a Go Style to a live element. Split from styleFromGrMob because
    // one Go field does not map to CSS at all: Disabled is an element
    // *property* on form controls (which is what makes the browser refuse to
    // dispatch the events whose callback IDs are wired above), and an ARIA
    // state plus pointer-events elsewhere. Both the initial render and the
    // update-style patch go through here so a control that becomes disabled
    // mid-session actually stops responding.
    function applyStyle(el, style, nodeType) {
        Object.assign(el.style, styleFromGrMob(style, nodeType));

        const disabled = !!style.Disabled;
        if (FORM_CONTROLS.has(el.tagName.toLowerCase())) {
            el.disabled = disabled;
        } else {
            if (disabled) {
                el.setAttribute("aria-disabled", "true");
            } else {
                el.removeAttribute("aria-disabled");
            }
            el.style.pointerEvents = disabled ? "none" : "";
        }
    }

    const FORM_CONTROLS = new Set(["button", "input", "textarea", "select"]);

    // Go's focus commands (core.Focus / core.DismissKeyboard) — see
    // core/focus.go. The epoch says when, the action says what.
    //
    // Epoch 0 means no command has ever been issued and Go stamped nothing,
    // so this is unreachable for an app that never touches focus; it is
    // checked anyway because the two props always travel together and a 0
    // must never be read as an instruction.
    //
    // Deferred a frame because focus() on an element outside the document is
    // a silent no-op, and on the initial render createElement builds the tree
    // detached — mount appends it only once the whole thing is assembled.
    // The same deferral is harmless on the patch path, where the element is
    // already live.
    //
    // "blur" is guarded on this element actually holding focus: a dismiss
    // reaches every field on the page and exactly one of them is the one to
    // release. Only the target is told "focus"; every other field is told ""
    // and does nothing, because focusing over there already blurs this one.
    function applyFocusCommand(el, epoch, action) {
        if (!epoch) return;
        if (action !== "focus" && action !== "blur") return;
        requestAnimationFrame(() => {
            if (action === "focus") {
                el.focus();
            } else if (document.activeElement === el) {
                el.blur();
            }
        });
    }

    // core.ContentMode -> CSS object-fit. Go states this table once, in
    // htmlout/objectfit.go, and this is its restatement here. The two are
    // compared by TestRuntimeObjectFitsMatchGo in wasm/verify, which parses
    // *this literal* out of this file and runs under a plain `go test ./...`,
    // so a change on either side fails until it is made on both. Keep it a
    // flat literal in a function named objectFitFor, subscripted by that
    // function's own argument and falling back to "" — the parse reads that
    // shape, the same one tagForType and inputTypeFor are written in. (It
    // takes the subscript name off the signature, so `mode` here and `type`
    // there are both fine; what it will not accept is a subscript that is
    // neither.)
    //
    // Go's table holds the bare value ("contain") rather than the whole
    // declaration, because that is the half the two sides share: htmlout
    // joins "object-fit:" onto it for its style attribute, and this assigns
    // it to a property.
    //
    // An unknown or absent mode yields "", which *clears* the property. That
    // is not the same as doing nothing, and the patch path is why: an Image
    // whose contentMode prop is removed has to fall back to the browser's
    // default rather than keep the last mode it was handed.
    function objectFitFor(mode) {
        return {
            fit: "contain",
            fill: "cover",
            stretch: "fill",
            center: "none",
        }[mode] || "";
    }

    // core.Alignment -> CSS text-align. Go states this table once, in
    // htmlout/textalign.go, and this is its restatement here; the two are
    // compared by TestRuntimeTextAlignsMatchGo in wasm/verify. Same shape rule
    // as objectFitFor above: a flat literal in a function named textAlignFor,
    // subscripted by that function's own argument, falling back to "".
    //
    // This is the newest of the four tables and the only one that was added to
    // *close* a gap rather than to pin an existing copy. Until it existed this
    // runtime did not read style.Align at all, in any form — so every
    // core.Align on the web target was silently dropped while htmlout emitted
    // a text-align and both natives set one. Four renderers, three behaviors,
    // and one of them was "nothing".
    //
    // Only four of the six Alignments are here. AlignStretch and AlignBaseline
    // name a cross-axis placement, not a text alignment, and CSS text-align
    // has no such keyword; they reach this function through Style.Align's
    // other role (the fallback a container reads when AlignItems is unset) and
    // are meant to fall through to "". See core.TextAlignments.
    //
    // "" clears the property rather than leaving it alone, which is what makes
    // an Align changed from "center" to "stretch" on the patch path stop being
    // centered.
    function textAlignFor(align) {
        return {
            start: "left",
            center: "center",
            end: "right",
            justify: "justify",
        }[align] || "";
    }

    // The Style -> CSS mapping. nodeType decides the default flex axis, the
    // same rule htmlout's styleValue uses: a Row stacks horizontally, every
    // other container vertically.
    function styleFromGrMob(style, nodeType) {
        const out = {};
        if (style.FontSize) out.fontSize = `${style.FontSize}px`;
        if (style.TextColor) out.color = style.TextColor;
        // Guarded on style.Align rather than assigned unconditionally, so a
        // patch whose Changes do not mention Align leaves the property alone —
        // the same convention every other property in this function follows,
        // because applyStyle Object.assigns the result over the live style.
        if (style.Align) out.textAlign = textAlignFor(style.Align);
        if (style.Background) out.background = style.Background;
        if (style.Padding) out.padding = edgeToCSS(style.Padding);
        if (style.Margin) out.margin = edgeToCSS(style.Margin);
        if (style.BorderRadius) out.borderRadius = `${style.BorderRadius}px`;
        if (style.Width) out.width = style.Width;
        if (style.Height) out.height = style.Height;
        // Flex layout. A plain <div> is block flow and ignores gap,
        // justify-content and align-items entirely, so a node that sets any
        // of them has to be made a flex container first — without this,
        // AlignItems ("stretch" included) was declared in Go and silently
        // dropped on the web, while both natives honored it.
        if (style.Gap || style.JustifyContent || style.AlignItems || style.FlexDirection) {
            out.display = "flex";
            out.flexDirection = style.FlexDirection || (nodeType === "Row" ? "row" : "column");
            if (style.Gap) out.gap = `${style.Gap}px`;
            if (style.JustifyContent) out.justifyContent = style.JustifyContent;
            if (style.AlignItems) out.alignItems = style.AlignItems;
        }
        // Display is deliberately NOT emitted. Go's DisplayMode carries values
        // that are not CSS display keywords ("visible", "hidden"), and unlike
        // htmlout — where an invalid declaration is dropped and the earlier
        // display:flex survives — assigning one through el.style would
        // overwrite the flex display in this object first and then be
        // rejected by the browser, leaving the container in block flow.
        // A flex *item* property: how this node behaves inside its parent's
        // layout, so it needs no display:flex of its own.
        if (style.FlexGrow) out.flexGrow = `${style.FlexGrow}`;
        if (style.BorderWidth && style.BorderColor) {
            out.border = `${style.BorderWidth}px solid ${style.BorderColor}`;
        }
        // core.Transition's canonical "<ms>ms <easing>" is valid CSS as-is;
        // the browser drives the frames, same declare-in-Go model as the
        // native renderers.
        if (style.Transition) out.transition = `all ${style.Transition}`;
        return out;
    }

    function edgeToCSS(edge) {
        return `${edge.Top}px ${edge.Right}px ${edge.Bottom}px ${edge.Left}px`;
    }

    // The Go node type -> the HTML tag. Go states this table once, in
    // htmlout/tag.go, and this is its restatement in the language that
    // actually calls createElement — the runtime cannot call into Go to ask.
    // The two are compared by TestRuntimeTagsMatchGo in wasm/verify, which
    // parses *this literal* out of this file and runs under a plain
    // `go test ./...`, so a change made on either side fails until it is made
    // on both. That test reads the object literal textually and checks the
    // `[type] || "div"` that follows it, so keep this a flat literal in a
    // function named tagForType. Same arrangement as inputTypeFor below.
    //
    // A census, not a list of exceptions: the plain <div> types are spelled
    // out even though the fallback would produce the same element, so that a
    // node type added to core and not taught to this runtime shows up as a
    // missing row here rather than as silence.
    function tagForType(type) {
        return {
            Text: "span",
            Button: "button",
            Image: "img",
            TextArea: "textarea",

            // Told apart from each other by inputTypeFor, below.
            Input: "input",
            InputPassword: "input",
            NumericInput: "input",
            Checkbox: "input",

            Box: "div",
            Card: "div",
            Column: "div",
            Row: "div",
            Scroll: "div",
            SafeArea: "div",
            List: "div",
            Modal: "div",
            TabView: "div",
            Spacer: "div",
            CameraView: "div",

            // Grouping nodes, and the one place this runtime deliberately
            // disagrees with htmlout, which emits their children with no box
            // at all. It can: it is a static snapshot. This runtime cannot,
            // because patches are addressed positionally — a TargetID of
            // "root/1/0" is resolved against the data-node-path attributes
            // renderNode writes by walking node.Children — so the DOM has to
            // stay isomorphic to the node tree. Dropping the element for a
            // Fragment would send every patch beneath it to the wrong node.
            //
            // The cost is real and known: inside a flex parent this div
            // becomes the single flex item and swallows the gap and alignment
            // meant for the children. Fixing it means teaching the addressing
            // scheme about nodes that have no element, not deleting these two
            // rows. See transparentTypes in htmlout/tag.go.
            Fragment: "div",
            Theme: "div",
        }[type] || "div";
    }

    // The Go node type -> the <input> type attribute. Only the four types
    // tagForType collapses onto <input> appear here; every other node has a
    // tag that already says what it is, and gets no type attribute (which is
    // why the fallback below is "" and not an error).
    //
    // Go states this table once, in htmlout/inputtype.go, and this is its
    // restatement in the language that actually sets the attribute — the
    // runtime cannot call into Go to ask. The two are not kept in step by
    // hand: htmlout.InputTypes() is compared against *this literal*, parsed
    // out of this file, by TestRuntimeInputTypesMatchGo in wasm/verify, which
    // runs under a plain `go test ./...`. So a change made on either side
    // fails until it is made on both. That test parses the object literal
    // textually and checks the `[type]` that follows it, so keep this a flat
    // literal in a function named inputTypeFor.
    //
    // This is the discriminator the DOM needs and dataset.nodeType cannot
    // supply: nodeType tells *this code* what a node is, the type attribute
    // tells the *browser*, and only the latter decides what gets drawn.
    function inputTypeFor(type) {
        return {
            Input: "text",
            InputPassword: "password",
            NumericInput: "number",
            Checkbox: "checkbox",
        }[type] || "";
    }

    // The visible height of a TextArea, in lines. A property rather than an
    // attribute, matching value and placeholder: this is live control state
    // the runtime keeps in step with Go, not markup written once.
    //
    // Guarded on a positive integer because rows is "limited to only positive
    // numbers" in the DOM — assigning 0 is an error, not a request for a
    // zero-line box. core.TextArea always supplies a positive count, so the
    // guard only covers a hand-built core.Node, which then keeps the
    // browser's own default height. htmlout differs there, defaulting an
    // absent rows to 3, because it has to emit *some* attribute value where
    // this path can simply say nothing.
    function applyRows(el, rows) {
        const n = Number(rows);
        if (Number.isInteger(n) && n > 0) {
            el.rows = n;
        }
    }

    function mapEventName(propKey) {
        return {
            onClick: "click",
            onChange: "input",
            onToggle: "change",
            // Listed explicitly even though the fallback below would derive
            // the same names: these are the two DOM events that do not
            // bubble, and naming them here is where a reader looks to find
            // out that the listener has to sit on the element itself. It
            // does — createElement and the update-props patch both attach on
            // the node that owns the prop — so nothing more is needed, but a
            // future move to delegated listeners would break exactly here.
            onFocus: "focus",
            onBlur: "blur",
            // An <input> outside a <form> has no submit event of its own, so
            // the return key is observed where it actually happens. The
            // Enter filter lives in eventQualifies rather than here, because
            // this map only names events — a keydown listener that fired the
            // handler on every keystroke would be a very loud bug.
            onSubmit: "keydown"
        }[propKey] || propKey.toLowerCase().replace(/^on/, "");
    }

    // Filters the raw DOM event before it becomes a Go dispatch.
    //
    // Only onSubmit needs this: it listens on keydown (see mapEventName) and
    // means one particular key. Shift+Enter is excluded so a textarea keeps
    // its "newline without submitting" convention, which is the same split
    // the native renderers get for free — Compose gives a multiline field a
    // newline key rather than an action key.
    //
    // Every other event maps one-to-one onto a DOM event and qualifies
    // unconditionally.
    function eventQualifies(propKey, e) {
        if (propKey !== "onSubmit") return true;
        if (e.key !== "Enter" || e.shiftKey) return false;
        // Nothing else should also act on this keypress — inside a <form> the
        // browser would otherwise submit the page out from under the app.
        e.preventDefault();
        return true;
    }

    // The keyboard action hint, the web's thin equivalent of Android's
    // ImeAction and SwiftUI's submitLabel. A soft keyboard (mobile browsers,
    // and desktop only in dev tools' device mode) relabels its return key
    // from it; a hardware keyboard ignores it entirely, which is why the
    // *behavior* rides onSubmit and never this attribute.
    //
    // Removed rather than set to "" when neither prop asks for one: an empty
    // enterkeyhint is not a valid value, and an update-props patch carries
    // the whole new map, so a field that loses its submit action has to lose
    // the attribute with it.
    function applyEnterKeyHint(el, props) {
        const hint = props.imeAction === "next" ? "next" : (props.onSubmit ? "done" : "");
        if (hint) {
            el.setAttribute("enterkeyhint", hint);
        } else {
            el.removeAttribute("enterkeyhint");
        }
    }

    // Builds the envelope Go receives for one DOM event.
    //
    // `type` is the *Go node type* ("Input", "Checkbox", ...), not the HTML
    // tag. The two are not interchangeable: tagForType collapses Input,
    // InputPassword, NumericInput and Checkbox all onto <input>, so a tag
    // cannot tell a text field from a checkbox and a checkbox would be asked
    // for its .value instead of its .checked. Both call sites therefore pass
    // the Go type — createElement has it in hand, and the update path reads
    // it back off the element, which is one of the reasons it is recorded
    // there at all.
    //
    // Matched case-insensitively so the two call sites cannot drift apart
    // again: they did once, silently, and the cost was total — a text field
    // present at the initial render sent {} for every keystroke, Go routed
    // the void envelope to the void callback map where a txt_ ID does not
    // exist, and typing did nothing at all.
    function extractEventPayload(e, type) {
        // Focus and blur are void events: Go registered them through the
        // plain callback channel, so the envelope must carry no value at all.
        // Checked before the type test below on purpose — a focus event on an
        // <input> would otherwise be sent as {value: "..."}, and Go, seeing a
        // string, would dispatch it to the *text* callback map, where a void
        // ID does not exist. The handler would silently never run.
        // keydown is here for the same reason: it carries onSubmit, which Go
        // registered as a void callback. Sending {value} would route it into
        // the *text* callback map, where a void ID does not exist, and the
        // submit handler would silently never run.
        if (e.type === "focus" || e.type === "blur" || e.type === "keydown") {
            return {};
        }
        const goType = String(type || "").toLowerCase();
        if (["input", "textarea", "numericinput", "inputpassword"].includes(goType)) {
            return { value: e.target.value };
        }
        if (goType === "checkbox") {
            return { value: e.target.checked };
        }
        return {};
    }


    function mount(jsonTree, mountPointId = "app") {
        const tree = typeof jsonTree === "string" ? JSON.parse(jsonTree) : jsonTree;
        const root = renderNode(tree, "root");
        rootElement = document.getElementById(mountPointId);
        rootElement.innerHTML = "";
        rootElement.appendChild(root);
    }

    function patch(patchList) {
        const patches = typeof patchList === "string" ? JSON.parse(patchList) : patchList;

        patches.forEach(p => {
            const el = document.querySelector(`[data-node-path="${p.TargetID}"]`);
            if (!el) {
                return;
            }

            switch (p.Type) {
                case "update-props":
                    // Before the per-key loop, for the same reason
                    // createElement applies it after one: the hint reads two
                    // props at once, and the patch carries the whole new map.
                    if ("imeAction" in p.Changes || "onSubmit" in p.Changes) {
                        applyEnterKeyHint(el, p.Changes);
                    }
                    for (const [k, v] of Object.entries(p.Changes)) {
                        if (k === "value") {
                            if (el.value === v) continue;
                            el.value = v;
                        } else if (k === "content") {
                            if (el.textContent === v) continue;
                            el.textContent = v;
                        } else if (k === "placeholder") {
                            if (el.placeholder === v) continue;
                            el.placeholder = v;
                        } else if (k === "checked") {
                            // No echo guard, unlike value above: assigning a
                            // boolean back onto a checkbox costs nothing,
                            // where re-assigning a text field's value would
                            // move the caret to the end mid-typing.
                            el.checked = !!v;
                        } else if (k === "rows") {
                            applyRows(el, v);
                        } else if (k === "src") {
                            if (el.src === v) continue;
                            el.src = v;
                        } else if (k === "contentMode") {
                            el.style.objectFit = objectFitFor(v);
                        } else if (k === "focusEpoch") {
                            // The epoch is the whole trigger; focusAction is
                            // only read once it has moved. An update-props
                            // patch carries the entire new props map, so a
                            // field re-rendered for its value would otherwise
                            // re-run whatever focus command was last issued.
                            if (String(el.dataset.focusEpoch) === String(v)) continue;
                            el.dataset.focusEpoch = v;
                            applyFocusCommand(el, v, p.Changes.focusAction);
                        } else if (k === "focusAction") {
                            // Handled with focusEpoch above; on its own it
                            // says when nothing, only what.
                            continue;
                        } else if (k.startsWith("on")) {
                            const event = mapEventName(k);
                            el.dataset[`listener_${k}`] = v;
                            if (!el.dataset[`has_listener_${k}`]) {
                                el.dataset[`has_listener_${k}`] = "true";
                                el.addEventListener(event, (e) => {
                                    const latestCbId = el.dataset[`listener_${k}`];
                                    if (latestCbId && eventQualifies(k, e)) {
                                        // The Go node type, not the tag: see
                                        // extractEventPayload. It was
                                        // recorded on the element at creation
                                        // because the tag cannot recover it.
                                        const payload = extractEventPayload(e, el.dataset.nodeType);
                                        window.GoInvokeCallback(latestCbId, payload);
                                    }
                                });
                            }
                        }
                    }
                    break;


                case "update-style":
                    // The patch carries only the changed Style, not the node
                    // type — and styleFromGrMob needs the type to pick a
                    // flex axis. It was recorded on the element at creation.
                    applyStyle(el, p.Changes, el.dataset.nodeType || "");
                    break;

                case "replace":
                    const newEl = renderNode(p.Changes, p.TargetID);
                    el.replaceWith(newEl);
                    break;

                case "remove":
                    el.remove();
                    break;

                case "add-child":
                    const index = el.children.length;
                    const newChild = renderNode(p.Changes, `${p.TargetID}/${index}`);
                    el.appendChild(newChild);
                    break;
            }
        });
    }

    return {
        mount,
        patch,
    };
})();

function checkLoop() {

    if (window.GrMobWASM.IsDirty()) {
        const patch = window.GrMobWASM.RenderAgain();
        GrMob.patch(patch);
    }
    requestAnimationFrame(checkLoop);
}

function waitForWasm() {
    if (window.GrMobWASM) {
        checkLoop();
    } else {
        setTimeout(waitForWasm, 100);
    }
}
waitForWasm();


window.GrMobRequestPermission = function (permission, callback) {
    if (permission === "camera") {
        navigator.mediaDevices.getUserMedia({ video: true })
            .then(stream => {
                // Permissão concedida
                stream.getTracks().forEach(track => track.stop()); // parar stream após teste
                callback(true);
            })
            .catch(err => {
                console.warn("Camera permission denied:", err);
                callback(false);
            });
    }
    // poderás adicionar outros casos como 'microphone', 'geolocation' etc.
}
