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

        if (node.Type === "Modal") {
            // The overlay chassis. Core's ModalNode carries no Style — its
            // look is these fixed rules plus the backdrop prop — so this is
            // assigned once here and only display (visible) and background
            // (backdrop) ever change, both through the prop paths below.
            // display starts "none" because Visible defaults to false in Go;
            // the visible prop in the loop below sets the truth either way.
            Object.assign(el.style, {
                position: "fixed",
                top: 0, left: 0, right: 0, bottom: 0,
                display: "none",
                flexDirection: "column",
                alignItems: "center",
                justifyContent: "center",
                zIndex: 1000,
            });
        }

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

        // Containers are stacks by definition: on the native renderers a
        // Row/Column is inherently an HStack/VStack, so the web has to opt
        // into the same default or diverge — a block-flow div lets inline
        // children (Text renders as <span>) run together on one line, which
        // is exactly what a bare Column of texts looked like before this.
        // Assigned before applyStyle so a style-driven flex block (which
        // carries the axis and alignment logic in styleFromGrMob) still wins.
        // Modal is excluded: its fixed-overlay chassis above already sets
        // flex and toggles display through the visible prop. Spacer is
        // excluded because it is a sized void, not a stack.
        if (STACK_CONTAINERS.has(node.Type)) {
            el.style.display = "flex";
            el.style.flexDirection = node.Type === "Row" ? "row" : "column";
        }

        if (node.Style) {
            applyStyle(el, node.Style, node.Type);
        }

        if (node.Props) {
            for (const [key, value] of Object.entries(node.Props)) {
                if (key === "visible" && node.Type === "Modal") {
                    // flex, not block: the overlay centers its content.
                    el.style.display = value ? "flex" : "none";
                } else if (key === "backdrop" && node.Type === "Modal") {
                    el.style.background = value;
                } else if (key === "onDismiss" && node.Type === "Modal") {
                    // Checked before the generic on* branch: "dismiss" is not
                    // a DOM event. The real trigger is a click on the backdrop
                    // itself — attachModalDismiss guards on the target so a
                    // click inside the content never dismisses.
                    attachModalDismiss(el, value);
                } else if (key === "onLongPress") {
                    // Before the generic on* branch for the same reason
                    // onDismiss is: there is no "longpress" DOM event, so the
                    // generic path would attach a listener that never fires
                    // and mark the slot taken so the real wiring could never
                    // be installed.
                    attachLongPress(el, value);
                } else if (key.startsWith("on")) {
                    const event = mapEventName(key);
                    el.dataset[`listener_${key}`] = value;
                    if (!el.dataset[`has_listener_${key}`]) {
                        el.dataset[`has_listener_${key}`] = "true";
                        el.addEventListener(event, (e) => {
                            const latestCbId = el.dataset[`listener_${key}`];
                            if (latestCbId && eventQualifies(key, e, el)) {
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

    // The container types that stack their children (Row horizontally,
    // everything else vertically) — the same axis rule styleFromGrMob and
    // htmlout's styleValue apply when a node opts into flex explicitly.
    // Fragment and Theme are grouping nodes, but this runtime draws them as
    // real divs (see tagForType), so they need the stacking default too.
    const STACK_CONTAINERS = new Set([
        "Row", "Column", "Card", "Box", "Scroll", "SafeArea", "List",
        "Fragment", "Theme",
    ]);

    // Wires a Modal's backdrop tap to Go's OnDismiss. Same latest-ID dataset
    // discipline as the generic listener path: callback IDs are per-pass, so
    // the listener is attached once and reads the current ID at click time.
    // The target guard is the semantic line between "tapped outside" and
    // "tapped the dialog": clicks inside the content bubble up through this
    // element too, but their target is the content, not the overlay.
    function attachModalDismiss(el, cbId) {
        el.dataset.listener_onDismiss = cbId;
        if (!el.dataset.has_listener_onDismiss) {
            el.dataset.has_listener_onDismiss = "true";
            el.addEventListener("click", (e) => {
                if (e.target !== el) return;
                const latestCbId = el.dataset.listener_onDismiss;
                if (latestCbId) {
                    // Void callback: the envelope must carry no value, like
                    // focus/blur — see extractEventPayload.
                    window.GoInvokeCallback(latestCbId, {});
                }
            });
        }
    }

    // How long a press has to be held before it counts as a long press.
    // 500ms is what Android's ViewConfiguration and UIKit's
    // UILongPressGestureRecognizer both use by default, so a gesture written
    // once in Go feels the same on all three targets.
    const LONG_PRESS_MS = 500;

    // Wires core.OnLongPress, which the DOM has no event for.
    //
    // The generic `on*` path cannot express this: it maps one prop to one DOM
    // event, and mapEventName's fallback derived the nonexistent "longpress",
    // so an onLongPress prop attached a listener for an event the browser
    // never fires. This synthesizes the gesture instead — a timer armed on
    // pointerdown and disarmed by anything that ends the press.
    //
    // pointer* rather than touch* or mouse*: one set of events covers finger,
    // pen and mouse, which is what the native long-press recognizers do.
    //
    // The follow-up click is suppressed. A press held past the threshold and
    // then released still produces a click, and firing both handlers for one
    // gesture is wrong on every platform — Android's combinedClickable and
    // SwiftUI's gesture arbitration both pick one. The flag rides the dataset
    // so the click listener (which is a separate closure, possibly attached
    // on a different pass) can see it.
    function attachLongPress(el, cbId) {
        el.dataset.listener_onLongPress = cbId;
        if (el.dataset.has_listener_onLongPress) return;
        el.dataset.has_listener_onLongPress = "true";

        let timer = null;
        const disarm = () => {
            if (timer !== null) {
                clearTimeout(timer);
                timer = null;
            }
        };

        el.addEventListener("pointerdown", () => {
            disarm();
            timer = setTimeout(() => {
                timer = null;
                // Re-read at fire time, exactly as every other listener here
                // does: the ID is positional and may have been refreshed —
                // or pruned — by a pass that landed during the press.
                const latestCbId = el.dataset.listener_onLongPress;
                if (!latestCbId) return;
                el.dataset.longPressFired = "true";
                window.GoInvokeCallback(latestCbId, {});
            }, LONG_PRESS_MS);
        });
        // Every way a press can stop being a press. pointerleave covers the
        // finger sliding off the element, which on native cancels the
        // gesture rather than completing it.
        for (const ev of ["pointerup", "pointercancel", "pointerleave"]) {
            el.addEventListener(ev, disarm);
        }
    }

    // Drops the callback IDs of handler props this node no longer carries.
    //
    // Callback IDs are *positional*: core/event.go re-derives them from a
    // per-pass counter, so "cb_3" belongs to whichever node happens to be the
    // fourth registration this pass. A node that stops carrying, say, onClick
    // must therefore forget the ID it last saw — keep it, and the next pass
    // hands that same ID to some other node, and clicking this element fires
    // that node's handler. The failure is silent and looks like a wiring bug
    // in the app.
    //
    // An update-props patch carries the *whole* new props map (reconcile
    // emits new.Props, never a delta — see reconcile/patch.go), so a
    // listener_* entry whose prop key is absent from Changes is definitively
    // gone rather than merely unchanged.
    //
    // The DOM listener itself is left attached and simply goes inert: every
    // listener this runtime installs re-reads its ID from the dataset at
    // dispatch time and does nothing when it is missing. Keeping the
    // has_listener_* marker alongside means a prop that comes back on a later
    // pass re-uses that one listener instead of stacking a second copy.
    function pruneStaleListeners(el, props) {
        // Object.keys snapshots, so deleting inside the loop is safe.
        for (const key of Object.keys(el.dataset)) {
            if (!key.startsWith("listener_")) continue;
            const prop = key.slice("listener_".length);
            if (!(prop in props)) {
                delete el.dataset[key];
            }
        }
    }

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
    // other role (the fallback a container reads when AlignItems is unset —
    // crossAxisAlignFor below is that role's table) and are meant to fall
    // through to "". See core.TextAlignments.
    //
    // "" clears the property rather than leaving it alone, which is what makes
    // an Align changed from "center" to "stretch" on the patch path stop being
    // centered.
    //
    // The four rows map each value to itself: start/end are CSS's
    // direction-aware keywords (originally the physical left/right, which
    // disagreed with both natives in RTL locales — see the authority's doc).
    // The table still earns its keep as a filter, because the identity must
    // NOT extend to the two cross-axis values above.
    function textAlignFor(align) {
        return {
            start: "start",
            center: "center",
            end: "end",
            justify: "justify",
        }[align] || "";
    }

    // core.Alignment -> CSS align-items: Style.Align's *second* role, the
    // cross-axis value a vertical-stacking container falls back to when
    // AlignItems is unset. Go states this table once, in htmlout/crossaxis.go,
    // and this is its restatement; the two are compared by
    // TestRuntimeCrossAxisAlignsMatchGo in wasm/verify. Same shape rule as
    // textAlignFor above: a flat literal in a function named crossAxisAlignFor,
    // subscripted by that function's own argument, falling back to "".
    //
    // Like the text-align table, this one was added to *close* a gap: both
    // natives have read the fallback since they existed, so Align: "center" on
    // a Column centered the children on device and only the text on the web.
    //
    // The values are the AlignItems spellings ("flex-start", not CSS's newer
    // "start") because AlignItems itself is emitted verbatim below, and the
    // fallback means "behave as if that AlignItems had been set" — one
    // semantic, one CSS spelling, whichever prop stated it. justify and
    // baseline have no row: no native dispatch answers for them on the cross
    // axis (baseline falls through to start-packing there), so a row here
    // would move two targets out of four. See the authority's doc.
    function crossAxisAlignFor(align) {
        return {
            start: "flex-start",
            center: "center",
            end: "flex-end",
            stretch: "stretch",
        }[align] || "";
    }

    // Which node types read the fallback at all, by the flex axis each stacks
    // along — exactly the containers the natives read it for (Card is a Column
    // whose Go theme style carries the card look, on every renderer). Row is
    // absent on purpose, everywhere: the fallback applies to a horizontal
    // cross axis only, and a gate is needed at all because styleFromGrMob
    // serializes every node — without it, a Text carrying Align in its
    // ordinary text role would become a flex container. Go's copy is
    // alignFallbackAxes in htmlout/crossaxis.go; the two are compared by
    // TestRuntimeAlignFallbackAxesMatchGo, so keep the flat-literal shape.
    function alignFallbackAxisFor(nodeType) {
        return {
            Column: "column",
            Card: "column",
            List: "column",
        }[nodeType] || "";
    }

    // The Style -> CSS mapping. nodeType decides the default flex axis, the
    // same rule htmlout's styleValue uses: a Row stacks horizontally, every
    // other container vertically.
    function styleFromGrMob(style, nodeType) {
        // Every property this function manages is assigned on every call —
        // a real value, or "" (which Object.assign turns into removal of the
        // inline declaration). The wire contract forces totality: an
        // update-style patch carries the WHOLE new Style (reconcile/patch.go),
        // so a zero field means "unset now", not "unmentioned" — and because
        // the patch path reuses the live element, a guarded `if (style.X)`
        // left the old declaration standing whenever a field returned to
        // zero. Lesson 7.2's core.BorderRadius(0) is the canonical victim:
        // the corners stayed rounded because nothing ever cleared them.
        const out = {};
        out.fontSize = style.FontSize ? `${style.FontSize}px` : "";
        // core.Weight's values (200/400/700) are literal CSS font-weight
        // numbers, so the int crosses as-is.
        out.fontWeight = style.FontWeight ? `${style.FontWeight}` : "";
        out.color = style.TextColor || "";
        // Unconditional on purpose, twice over: textAlignFor answers "" for
        // an unset or placement-only Align (keeping the totality rule), and
        // wasm/verify's TestRuntimeStyleAppliesTextAlign pins this exact
        // call so the text-align table cannot go unread on this target.
        out.textAlign = textAlignFor(style.Align);
        out.background = style.Background || "";
        out.padding = style.Padding ? edgeToCSS(style.Padding) : "";
        out.margin = style.Margin ? edgeToCSS(style.Margin) : "";
        out.borderRadius = style.BorderRadius ? `${style.BorderRadius}px` : "";
        out.width = style.Width || "";
        out.height = style.Height || "";
        // Flex layout. A plain <div> is block flow and ignores gap,
        // justify-content and align-items entirely, so a node that sets any
        // of them has to be made a flex container first — without this,
        // AlignItems ("stretch" included) was declared in Go and silently
        // dropped on the web, while both natives honored it.
        //
        // The effective cross-axis value is AlignItems, else the Align
        // fallback — the same read Renderer.swift's crossAxisValue does, gated
        // the same two ways htmlout's styleValue gates it: only the
        // vertical-stacking container types, and not when an explicit
        // FlexDirection flipped the node to a row, because the fallback
        // applies to a horizontal cross axis only on every target. Safe on
        // the patch path: an update-style patch carries the whole new Style
        // (reconcile/patch.go), so an absent AlignItems here means unset, not
        // unmentioned. The prefix test admits "column-reverse", whose cross
        // axis is horizontal all the same.
        const dir = style.FlexDirection || (nodeType === "Row" ? "row" : "column");
        let alignItems = style.AlignItems || "";
        if (!alignItems && dir.startsWith("column") && alignFallbackAxisFor(nodeType)) {
            alignItems = crossAxisAlignFor(style.Align || "");
        }
        // Stack containers are flex whether or not this Style asks for it —
        // createElement plants the same default for nodes with no Style at
        // all, and the totality rule above means this function must restate
        // it here or an update-style patch would clear it.
        if (style.Gap || style.JustifyContent || alignItems || style.FlexDirection || STACK_CONTAINERS.has(nodeType)) {
            out.display = "flex";
            out.flexDirection = dir;
        } else {
            out.display = "";
            out.flexDirection = "";
        }
        out.gap = style.Gap ? `${style.Gap}px` : "";
        out.justifyContent = style.JustifyContent || "";
        out.alignItems = alignItems || "";
        // Display is deliberately NOT emitted. Go's DisplayMode carries values
        // that are not CSS display keywords ("visible", "hidden"), and
        // assigning one through el.style would overwrite the flex display in
        // this object first and then be rejected by the browser, leaving the
        // container in block flow. htmlout hit the sibling problem from the
        // other side — a valid "block" emitted after the flex declarations
        // beat them, so a themed Card's own Display: block killed its
        // align-items — and now resolves Display against its flex container
        // in styleValue; emitting nothing keeps this runtime out of both
        // traps.
        // One reading of Display survives, translated rather than emitted:
        // an inline display is the themes' way of saying "hug your content"
        // (components.Button documents FullWidth as block display + width
        // precisely because the bundled themes give Button an inline one).
        // Inside this runtime's always-flex stacks the inline keyword itself
        // is inert — flex items are blockified — and the cross-axis default
        // (stretch) would spread every button across its Column. width:
        // fit-content carries the same intent in a way that is safe on both
        // axes: in a Column it stops the stretch, in a Row it restates what
        // the main axis already does, and unlike align-self it cannot
        // override the container's cross-axis alignment (which would top-pin
        // a button inside an AlignItemsCenter Row). An explicit Width — the
        // other half of the FullWidth contract — wins over it above, hence
        // the guard.
        if (!style.Width && (style.Display === "inline" || style.Display === "inline-block")) {
            out.width = "fit-content";
        }
        // A flex *item* property: how this node behaves inside its parent's
        // layout, so it needs no display:flex of its own.
        out.flexGrow = style.FlexGrow ? `${style.FlexGrow}` : "";
        out.border = (style.BorderWidth && style.BorderColor)
            ? `${style.BorderWidth}px solid ${style.BorderColor}`
            : "";
        // core.Transition's canonical "<ms>ms <easing>" is valid CSS as-is;
        // the browser drives the frames, same declare-in-Go model as the
        // native renderers.
        out.transition = style.Transition ? `all ${style.Transition}` : "";
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
            onSubmit: "keydown",
            // core.OnTouch means "the finger went down", which is
            // pointerdown. The fallback below would have derived "touch",
            // which is not a DOM event at all — so the prop attached a
            // listener nothing ever fired.
            onTouch: "pointerdown"
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
    function eventQualifies(propKey, e, el) {
        if (propKey === "onClick" && el && el.dataset.longPressFired) {
            // One gesture, one handler: the press already fired onLongPress
            // (see attachLongPress), and the browser's synthetic click on
            // release must not also fire onClick. Cleared here rather than on
            // pointerup so the flag is still standing when the click arrives.
            delete el.dataset.longPressFired;
            return false;
        }
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
            // "add" fills a slot that was nil in the old tree, so its
            // TargetID does not exist in the DOM yet — resolve the parent
            // from the path and insert at the slot's index instead of
            // falling through to the lookup below, which would drop it.
            if (p.Type === "add") {
                const slash = p.TargetID.lastIndexOf("/");
                const parent = document.querySelector(`[data-node-path="${p.TargetID.slice(0, slash)}"]`);
                if (!parent) return;
                const index = Number(p.TargetID.slice(slash + 1));
                const added = renderNode(p.Changes, p.TargetID);
                parent.insertBefore(added, parent.children[index] || null);
                return;
            }

            const el = document.querySelector(`[data-node-path="${p.TargetID}"]`);
            if (!el) {
                return;
            }

            switch (p.Type) {
                case "update-props":
                    // Before the per-key loop, for the same reason
                    // createElement applies it after one: the hint reads two
                    // props at once, and the patch carries the whole new map.
                    //
                    // Unconditional, exactly as createElement does it. Gating
                    // this on "imeAction" or "onSubmit" being present meant a
                    // field that *lost* its onSubmit kept the enterkeyhint it
                    // was given when it had one, so the keyboard went on
                    // advertising a submit affordance the field no longer had.
                    applyEnterKeyHint(el, p.Changes);
                    for (const [k, v] of Object.entries(p.Changes)) {
                        if (k === "value") {
                            if (el.value === v) continue;
                            el.value = v;
                        } else if (k === "content") {
                            if (el.textContent === v) continue;
                            el.textContent = v;
                        } else if (k === "label") {
                            // Buttons carry their text as `label`, not
                            // `content`, and createElement maps both onto
                            // textContent — this branch is the update half of
                            // that mapping. Without it a reused Button kept
                            // its old caption whenever a navigation diff
                            // paired it positionally with a different one.
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
                        } else if (k === "visible" && el.dataset.nodeType === "Modal") {
                            // This IS the modal open/close path: toggling
                            // core.Visible reaches the page as a prop patch,
                            // never as a subtree add/remove — the content
                            // stays mounted, which is why its state survives
                            // a close.
                            el.style.display = v ? "flex" : "none";
                        } else if (k === "backdrop" && el.dataset.nodeType === "Modal") {
                            el.style.background = v;
                        } else if (k === "onDismiss" && el.dataset.nodeType === "Modal") {
                            // Before the generic on* branch, which would
                            // attach a listener for a "dismiss" DOM event
                            // that never fires — and, worse, mark the
                            // listener slot taken so the real one could
                            // never be attached.
                            attachModalDismiss(el, v);
                        } else if (k === "onLongPress") {
                            attachLongPress(el, v);
                        } else if (k.startsWith("on")) {
                            const event = mapEventName(k);
                            el.dataset[`listener_${k}`] = v;
                            if (!el.dataset[`has_listener_${k}`]) {
                                el.dataset[`has_listener_${k}`] = "true";
                                el.addEventListener(event, (e) => {
                                    const latestCbId = el.dataset[`listener_${k}`];
                                    if (latestCbId && eventQualifies(k, e, el)) {
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
                    pruneStaleListeners(el, p.Changes);
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

                // "remove-child" is the Go diff's shrink patch: the new tree
                // has fewer children, and each surplus child arrives as its
                // own path, highest index first (see reconcile.Patch's
                // ordering contract), so removal is identical to "remove".
                // Dropping these — the switch's original behavior — left the
                // old screen's tail siblings alive in the DOM after any
                // navigation to a screen with fewer children.
                case "remove-child":
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

    // --- Toast overlay -------------------------------------------------------
    //
    // Toasts are host chrome, not app tree: core.ShowToast crosses as a
    // system event (see GrMobSystemEvent below), so nothing here is ever
    // reconciled or patched — each toast is a throwaway element with its own
    // timer. The container is lazily created and permanent: pointer-events
    // none, so a toast never steals a tap from the app underneath it, and a
    // z-index above the Modal overlay (1000), because a toast confirming a
    // modal's action must not be drawn behind the modal it confirms.

    let toastLayer = null;

    function ensureToastLayer() {
        if (toastLayer) return toastLayer;
        toastLayer = document.createElement("div");
        Object.assign(toastLayer.style, {
            position: "fixed",
            bottom: "24px",
            left: 0, right: 0,
            display: "flex",
            flexDirection: "column",
            alignItems: "center",
            gap: "8px",
            zIndex: 2000,
            pointerEvents: "none",
        });
        document.body.appendChild(toastLayer);
        return toastLayer;
    }

    function showToast(payload) {
        const el = document.createElement("div");
        el.textContent = payload.message || "";
        // The default look; a styled toast's overrides land on top of it.
        Object.assign(el.style, {
            background: "#2F3437",
            color: "#FFFFFF",
            padding: "10px 18px",
            borderRadius: "8px",
            fontSize: "14px",
            maxWidth: "80vw",
            boxShadow: "0 4px 12px rgba(0,0,0,0.25)",
            opacity: "0",
            transition: "opacity 150ms ease",
        });
        if (payload.style) {
            // The style crosses as a Go core.Style (capitalized fields), so
            // it goes through the same mapping every node style does.
            Object.assign(el.style, styleFromGrMob(payload.style, "Toast"));
        }
        ensureToastLayer().appendChild(el);
        // Two frames, not one: the element must be painted at opacity 0
        // before the transition target is set, or it appears without fading.
        requestAnimationFrame(() => requestAnimationFrame(() => {
            el.style.opacity = "1";
        }));
        const duration = payload.duration || 2000;
        setTimeout(() => {
            el.style.opacity = "0";
            setTimeout(() => el.remove(), 200); // after the fade-out
        }, duration);
    }

    return {
        mount,
        patch,
        showToast,
    };
})();

// The system-event sink the WASM host looks for at startup (wasm/main.go's
// registerSystemEvents): defined at page level, before the wasm module is
// instantiated, so the host's feature check finds it. Unknown event names are
// dropped on purpose — a newer app on an older page degrades to silence, the
// same contract the host applies when this function is missing entirely.
window.GrMobSystemEvent = function (name, payloadJSON) {
    if (name === "toast") {
        GrMob.showToast(JSON.parse(payloadJSON));
    }
};

// The patch push channel the WASM host feature-checks at startup (wasm/
// main.go's renderInitial): when this exists, async state changes — timers,
// goroutines — arrive here directly from the manager's pump, on the state
// write's own schedule. Without it the host falls back to the IsDirty poll
// below, which rides requestAnimationFrame — and rAF is fully suspended in a
// hidden tab, so an UseInterval clock froze the moment the tab lost
// visibility even though the Go ticker kept running. Defined at page level,
// before the wasm module is instantiated, so the host's check finds it; the
// poll loop stays as a harmless fallback (with a listener attached the pump
// consumes the diff, so the poll sees a clean tree).
window.GrMobApplyPatches = function (patchJSON) {
    GrMob.patch(patchJSON);
};

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
