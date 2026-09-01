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
                            if (latestCbId) {
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

    // core.ContentMode -> CSS object-fit. Kept in sync with htmlout's
    // objectFit(); an unknown or absent mode clears the property so the
    // browser's own default applies.
    function objectFitFor(mode) {
        return {
            fit: "contain",
            fill: "cover",
            stretch: "fill",
            center: "none",
        }[mode] || "";
    }

    // The Style -> CSS mapping. nodeType decides the default flex axis, the
    // same rule htmlout's styleValue uses: a Row stacks horizontally, every
    // other container vertically.
    function styleFromGrMob(style, nodeType) {
        const out = {};
        if (style.FontSize) out.fontSize = `${style.FontSize}px`;
        if (style.TextColor) out.color = style.TextColor;
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

    function tagForType(type) {
        switch (type) {
            case "Text": return "span";
            case "Input":
            case "InputPassword":
            case "NumericInput":
            case "Checkbox": return "input";
            case "TextArea": return "textarea";
            case "Button": return "button";
            case "Image": return "img";
            case "Card":
            case "Row":
            case "Column":
            case "Scroll":
            case "SafeArea":
            case "Fragment":
            case "Spacer": return "div";
            default: return "div";
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
            onBlur: "blur"
        }[propKey] || propKey.toLowerCase().replace(/^on/, "");
    }

    function extractEventPayload(e, type) {
        // Focus and blur are void events: Go registered them through the
        // plain callback channel, so the envelope must carry no value at all.
        // Checked before the type test below on purpose — a focus event on an
        // <input> would otherwise be sent as {value: "..."}, and Go, seeing a
        // string, would dispatch it to the *text* callback map, where a void
        // ID does not exist. The handler would silently never run.
        if (e.type === "focus" || e.type === "blur") {
            return {};
        }
        if (["input", "textarea", "numericinput", "inputpassword"].includes(type)) {
            return { value: e.target.value };
        }
        if (type === "checkbox") {
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
                                    if (latestCbId) {
                                        const payload = extractEventPayload(e, el.tagName.toLowerCase());
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
