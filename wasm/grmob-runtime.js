// grmob-runtime.js

const GrMob = (() => {
    let rootElement = null;
    const DEBUG = true;

    function renderNode(node, path = "") {
        const el = createElement(node);
        el.setAttribute("data-node-path", path);


        if (node.Type === "Spacer" && node.Props) {
            applySpacerSize(el, node.Props.size);
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

        // A grid or row built without a Style still needs its chassis, which
        // lives in styleFromGrMob (keyed on the node type) so that an
        // update-style patch re-applies it rather than clearing it. Nodes
        // core.TextGrid builds always carry a Style and take the applyStyle
        // path below; this covers a hand-assembled one.
        if ((node.Type === "TextGrid" || node.Type === "GridRow") && !node.Style) {
            applyStyle(el, {}, node.Type);
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
                } else if (key === "min" || key === "max" || key === "step") {
                    applySliderBound(el, key, value, node.Props.value);
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
                else if (key === "runs") {
                    applyGridRuns(el, value);
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

    // The size of a Spacer, on both axes.
    //
    // core.Spacer(n) is n x n on both natives — Compose
    // Spacer(Modifier.size(n.dp)), SwiftUI Color.clear.frame(width:height:) —
    // and this used to set the height alone, so a Spacer between two items of
    // a Row held them 0px apart in the browser and n points apart on device.
    //
    // flex-shrink:0 is the other half. Every container this runtime draws is a
    // flex container (see STACK_CONTAINERS), a flex item's default is to
    // shrink under pressure, and a gap whose whole job is to hold a fixed
    // distance must not be the thing that gives way. The natives have fixed
    // frames and no equivalent to shrink, so this reproduces their behavior
    // rather than adding to it.
    //
    // A missing or zero size clears all three, which is what makes this safe
    // to call from the update path: a Spacer whose size prop goes away falls
    // back to nothing rather than keeping the last size it was handed.
    function applySpacerSize(el, size) {
        const px = Number(size) > 0 ? `${Number(size)}px` : "";
        el.style.width = px;
        el.style.height = px;
        el.style.flexShrink = px ? "0" : "";
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
        applyAccessibility(el, style);

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

    // core.Style's three accessibility fields -> the ARIA attributes that mean
    // the same thing. Attributes rather than style properties, which is why
    // this is here and not in styleFromGrMob — the same split Disabled makes.
    //
    // Both natives have read these since they existed (Compose
    // contentDescription / clearAndSetSemantics, SwiftUI accessibilityLabel /
    // accessibilityHint / accessibilityHidden); the two web targets read none
    // of them, so a decorative node marked AccessibilityHidden was correctly
    // skipped by TalkBack and VoiceOver and announced by every screen reader
    // on the web.
    //
    // aria-hidden wins alone: it prunes the element and its subtree from the
    // accessibility tree, which makes a name or description on the same node
    // contradictory rather than additive. Compose's clearAndSetSemantics
    // branch and SwiftUI's accessibilityHidden branch make the same exclusive
    // choice.
    //
    // The hint becomes aria-description, not aria-describedby: the latter
    // takes an ID reference and there is no second element here to point at.
    // Support for aria-description is thinner than the rest of ARIA — it is
    // the newest of the three — and the alternative is dropping the author's
    // hint entirely.
    //
    // Every attribute is set or removed on every call, the same totality rule
    // styleFromGrMob follows and for the same reason: an update-style patch
    // carries the whole new Style, so a field back at its zero value means
    // "unset now", and a guarded write would leave the old attribute standing.
    function applyAccessibility(el, style) {
        const hidden = !!style.AccessibilityHidden;
        setOrRemove(el, "aria-hidden", hidden ? "true" : "");
        setOrRemove(el, "aria-label", hidden ? "" : (style.AccessibilityLabel || ""));
        setOrRemove(el, "aria-description", hidden ? "" : (style.AccessibilityHint || ""));
    }

    // Sets an attribute to a non-empty value, or removes it. There is no empty
    // string that means "absent" for an attribute the way there is for a style
    // property, so the removal has to be spelled.
    function setOrRemove(el, name, value) {
        if (value) {
            el.setAttribute(name, value);
        } else {
            el.removeAttribute(name);
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
    // A range input's bounds. The attribute is the API here (there is no
    // .min property on a generic element), and the value is re-applied
    // afterwards because the browser clamps it to the bounds *at assignment
    // time*: Go's props arrive in key order (max, min, ..., value) on the
    // create path, but an update-props patch may carry a widened max after
    // the value was already clamped to the old one.
    function applySliderBound(el, key, bound, value) {
        if (el.type !== "range") return;
        if (bound === undefined || bound === null || bound === "") {
            el.removeAttribute(key);
        } else {
            el.setAttribute(key, String(bound));
        }
        if (value !== undefined && el.value != value) el.value = value;
    }

    // One row of a core.TextGrid: its runs, each a <span> carrying only the
    // declarations its run set. The row is rebuilt whole on every runs prop
    // — the reconciler already decided this row changed, and a row is a few
    // dozen spans at most, so diffing spans against spans would cost more
    // than it saved. Rebuilding also keeps the row's spans out of the node
    // tree: they carry no data-node-path and no patch is ever addressed to
    // one, which is what lets a row be replaced without disturbing the
    // positional addressing of everything around it.
    //
    // The attribute bits are core's Grid* constants. Dim has no CSS
    // spelling, so it is opacity, as in htmlout's gridRunStyle.
    function applyGridRuns(el, runs) {
        el.innerHTML = "";
        if (!Array.isArray(runs)) return;
        for (const run of runs) {
            const span = document.createElement("span");
            span.textContent = run.t ?? "";
            if (run.fg) span.style.color = run.fg;
            if (run.bg) span.style.background = run.bg;
            const a = Number(run.a) || 0;
            if (a & 1) span.style.fontWeight = "700";
            if (a & 2) span.style.opacity = "0.6";
            if (a & 4) span.style.fontStyle = "italic";
            const lines = [];
            if (a & 8) lines.push("underline");
            if (a & 16) lines.push("line-through");
            if (lines.length) span.style.textDecoration = lines.join(" ");
            el.appendChild(span);
        }
    }

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
        // A single elevation number on every target (Compose's
        // Modifier.shadow(elevation), SwiftUI's .shadow(radius:y:)) against a
        // CSS property that wants offsets, a blur and a color. The arithmetic
        // is the SwiftUI mapping restated — grMobShadow in GrMobStyle.swift
        // uses blur = elevation/2 and a y offset of elevation/3 — so one
        // core.Shadow(4) draws a comparable shadow on all three targets that
        // draw one at all. The color is SwiftUI's default black at a third
        // alpha, spelled out because CSS has no default.
        //
        // Rounded to two decimals rather than emitted at full float precision:
        // an elevation of 4 divides into 1.3333333333333333, which is noise in
        // a declaration measured in device pixels. htmlout rounds the same way,
        // so the two web targets emit the same string for the same elevation.
        out.boxShadow = style.Shadow
            ? `0 ${round2(style.Shadow / 3)}px ${round2(style.Shadow / 2)}px rgba(0,0,0,0.33)`
            : "";
        // An absolute line box height in px, not CSS's unitless multiplier:
        // that is what the field means on the natives (Compose takes
        // `lineHeight = n.sp`, SwiftUI derives a lineSpacing from n minus the
        // font size), so the unit has to be written or the same number would
        // mean two different things.
        out.lineHeight = style.LineHeight ? `${style.LineHeight}px` : "";
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
        // Style.Display, resolved against the flex block above rather than
        // emitted verbatim. Go's DisplayMode carries five values and only
        // three of them are CSS display keywords, so a blanket assignment
        // would overwrite the flex display in this object with a string the
        // browser then rejects, leaving the container in block flow. That is
        // why this used to emit nothing at all — and the cost of emitting
        // nothing was that core.Display(core.DisplayNone) hid a node on both
        // natives (Renderer.swift and Renderer.kt bail out before any layout)
        // and on htmlout, and did nothing here. Each value now gets the
        // treatment it actually needs:
        //
        //   - "none" is assigned last and wins over the flex display, on the
        //     same principle htmlout's styleValue applies: hiding beats
        //     layout on every target that reads Display at all.
        //   - "hidden" is not a display at all; it becomes visibility below.
        //   - "visible" likewise.
        //   - "block" stays unemitted: a block-level flex container is
        //     exactly display:flex, and outside a flex container the div is
        //     block already, so the mode is a no-op either way here.
        //   - "inline" is translated rather than emitted, just below.
        if (style.Display === "none") {
            out.display = "none";
        }
        // DisplayHidden / DisplayVisible in the property that means what they
        // say: keep the node's space, drop its pixels. Both natives read the
        // mode exactly this way (SwiftUI .opacity(0), Compose alpha 0), and
        // visibility is its CSS spelling — display:none above is the other
        // one, no pixels AND no space, which is why the two cannot share a
        // property. Total like everything else here, so a node that stops
        // being hidden becomes visible again on a patch.
        //
        // "visible" is assigned rather than treated as a no-op default: a node
        // nested inside a hidden ancestor inherits hidden, and an explicit
        // DisplayVisible is the only way an author can override that. The
        // natives get this for free, since opacity does not inherit.
        out.visibility = (style.Display === "hidden" || style.Display === "visible")
            ? style.Display
            : "";
        // One more reading of Display, translated rather than emitted:
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
        // Style.Animation is a CSS animation shorthand ("bounce 2s infinite").
        // Emitted verbatim, and it is the one property here that depends on
        // something the runtime does not supply: a matching @keyframes rule,
        // which has to come from the hosting page's stylesheet. The
        // declaration is inert until it does. Neither native reads the field.
        out.animation = style.Animation || "";
        // The remaining CSS-shaped fields of core.Style. Every one of them had
        // a StyleProp constructor in Go and no reader on any of the four
        // targets — declared and dropped. They are one property each here and
        // have no direct Compose/SwiftUI equivalent, so the web pair honors
        // them and the native gap is documented rather than faked. htmlout's
        // styleValue emits the same set.
        //
        // Verbatim for the same reason width and height are: core's dimension
        // strings ("40px", "45%", "auto") are already CSS lengths, and the
        // enums (Position, AlignItems, FlexWrap, Overflow, WhiteSpace) hold
        // the CSS keywords themselves.
        out.minWidth = style.MinWidth || "";
        out.minHeight = style.MinHeight || "";
        out.maxWidth = style.MaxWidth || "";
        out.maxHeight = style.MaxHeight || "";
        out.overflow = style.Overflow || "";
        out.whiteSpace = style.WhiteSpace || "";

        // The grid chassis (core.TextGrid): the fixed rules of a grid and its
        // rows, applied here rather than once at creation because every
        // property above is reassigned on every update-style patch, and a
        // chassis set only at creation would be wiped by the first one. The
        // author's own values win where they set one. A <pre> is already
        // fixed-pitch and unwrapped; this pins the margin a <pre> carries by
        // default, a line height the rows are sized against, and sideways
        // scrolling for a grid wider than the screen. Each row keeps one line
        // even when it has no runs, so the rows below it stay on the cell
        // grid. Same rules as htmlout's textGridChassis / gridRowChassis.
        if (nodeType === "TextGrid") {
            out.margin = out.margin || "0";
            out.lineHeight = out.lineHeight || "1.2";
            out.whiteSpace = out.whiteSpace || "pre";
            out.overflowX = out.overflow ? "" : "auto";
        }
        if (nodeType === "GridRow") {
            out.minHeight = out.minHeight || "1.2em";
        }
        // Out-of-flow placement. The offsets are assigned whether or not
        // Position is set, matching CSS itself: they are inert on a static box
        // rather than an error, and a node can sit in a positioned ancestor's
        // containing block without restating its own Position.
        out.position = style.Position || "";
        out.top = style.Top || "";
        out.right = style.Right || "";
        out.bottom = style.Bottom || "";
        out.left = style.Left || "";
        out.zIndex = style.ZIndex ? `${style.ZIndex}` : "";
        // Flex container properties that are deliberately NOT part of the
        // display:flex decision above. Unlike Gap/JustifyContent/AlignItems,
        // none of these does anything on its own — flex-wrap and the axis gaps
        // only have meaning once the box is already a flex container — so
        // promoting a box for them alone would change its layout to no
        // purpose.
        out.flexWrap = style.FlexWrap || "";
        out.rowGap = style.RowGap ? `${style.RowGap}px` : "";
        out.columnGap = style.ColumnGap ? `${style.ColumnGap}px` : "";
        // Flex *item* properties, joining flexGrow above.
        out.alignSelf = style.AlignSelf || "";
        out.flexBasis = style.FlexBasis || "";
        out.flexShrink = style.FlexShrink ? `${style.FlexShrink}` : "";
        return out;
    }

    // A Go core.EdgeInsets -> the four-value CSS shorthand, resolving the
    // Horizontal/Vertical fields into the sides that were not set explicitly.
    //
    // EdgeInsets carries six fields, not four: the per-side values plus a
    // shorthand pair that core.PaddingHorizontal / core.PaddingVertical write.
    // Reading only the four sides — which is what this function used to do —
    // meant PaddingHorizontal(16) applied cleanly in Go, rendered as 16px on
    // both natives, and as nothing at all here.
    //
    // "Set explicitly" means non-zero, so PaddingHorizontal(16) plus
    // PaddingLeft(0) cannot ask for a zero left inset. Both natives are lossy
    // in exactly the same way for the same reason (a Go zero value carries no
    // "was it set?" bit), and matching them is the point. Go states the rule
    // in htmlout/edges.go; the two are independent statements of one contract,
    // each with its own tests, the same arrangement the conformance replay
    // makes for the prop table.
    // Two decimal places, the precision a CSS length measured in device pixels
    // is meaningful at. Go's copy is round2 in htmlout/export.go.
    function round2(v) {
        return Math.round(v * 100) / 100;
    }

    function edgeToCSS(edge) {
        const side = (explicit, shorthand) => (explicit || 0) || (shorthand || 0);
        const h = edge.Horizontal, v = edge.Vertical;
        return `${side(edge.Top, v)}px ${side(edge.Right, h)}px `
            + `${side(edge.Bottom, v)}px ${side(edge.Left, h)}px`;
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
            Slider: "input",

            // A monospace grid and its rows (core.TextGrid); see
            // applyGridRuns for the spans inside a row.
            TextGrid: "pre",
            GridRow: "div",

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
            Slider: "range",
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
            // A Slider's end-of-drag event: "change" fires once when the
            // thumb is released, where "input" (onChange) fires on every
            // pixel of the drag. Same DOM event as onToggle, different prop,
            // so a slider can carry both.
            onChangeEnd: "change",
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
        if (["input", "textarea", "numericinput", "inputpassword", "slider"].includes(goType)) {
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
                            // Loose equality on purpose: a range input's
                            // value reads back as a string ("12.5") while Go
                            // sends a number, and a strict compare would
                            // re-assign on every status tick.
                            if (el.value == v) continue;
                            el.value = v;
                        } else if (k === "min" || k === "max" || k === "step") {
                            applySliderBound(el, k, v, p.Changes.value);
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
                        } else if (k === "runs") {
                            applyGridRuns(el, v);
                        } else if (k === "size" && el.dataset.nodeType === "Spacer") {
                            // The update half of the Spacer sizing in
                            // renderNode. Without it a Spacer whose size
                            // changed kept its original gap: the size lives in
                            // Props, so the change arrives as an update-props
                            // patch and nothing here read the key. Gated on
                            // the node type because "size" is a plausible prop
                            // name for a future node that means something else
                            // by it.
                            applySpacerSize(el, v);
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

    // Drops the "" entries from a styleFromGrMob result, turning a total
    // declaration map into an overlay that only states what the Style set.
    // Used where a style is layered onto defaults rather than applied to an
    // element the runtime owns outright — see showToast.
    function definedDecls(decls) {
        const out = {};
        for (const [k, v] of Object.entries(decls)) {
            if (v !== "") out[k] = v;
        }
        return out;
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
            // The style crosses as a Go core.Style (capitalized fields), so it
            // goes through the same mapping every node style does — but only
            // the declarations it actually sets are applied.
            //
            // styleFromGrMob is *total*: it returns every property it manages
            // on every call, with "" for the unset ones, so that an
            // update-style patch reusing a live element clears what the new
            // Style no longer carries. A toast has no patch path — each one is
            // a throwaway element rendered once — and the "" entries would
            // instead erase the defaults assigned just above, which is what a
            // core.UseToastStyle setting only a background used to do to the
            // padding, the radius, and the drop shadow.
            Object.assign(el.style, definedDecls(styleFromGrMob(payload.style, "Toast")));
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

    // ---- Audio ----------------------------------------------------------
    //
    // The browser half of core's audio service (core/audio.go): one
    // HTMLAudioElement for the page, driven by the "audio" system event's
    // commands, reporting back through the "audio_status" host event. The
    // Media Session API is wired too, so the OS media keys, the lock screen
    // on a phone browser, and Chrome's media hub all show the track and can
    // drive it — the same affordances the native shells get from their
    // media sessions, which is the point of the feature.
    //
    // Status leaves through window.GrMobWASM.HostEvent, which the Go host
    // installs (wasm/main.go). It is looked up per call rather than captured,
    // because the runtime loads before the wasm module does. Reports are
    // driven by the element's own events (playing, pause, timeupdate, ...),
    // never synchronously from inside a command: a command arrives from
    // inside a Go handler (the tap that called AudioLoad), and re-entering
    // Go from there is legal on wasm but pointless — the element will fire
    // "loadstart" a tick later anyway.
    const audio = (() => {
        let el = null;
        let track = {};      // the last load's metadata, for the media session
        let phase = "idle";  // an AudioState value
        let errorText = "";
        let lastTick = 0;    // throttles timeupdate, which fires ~4Hz

        function element() {
            if (el) return el;
            el = new Audio();
            el.preload = "auto";
            el.addEventListener("loadstart", () => setPhase("loading"));
            el.addEventListener("waiting", () => setPhase("loading"));
            el.addEventListener("playing", () => setPhase("playing"));
            el.addEventListener("pause", () => { if (!el.ended) setPhase("paused"); });
            el.addEventListener("ended", () => setPhase("ended"));
            el.addEventListener("durationchange", report);
            el.addEventListener("ratechange", report);
            el.addEventListener("seeked", report);
            el.addEventListener("error", () => {
                const e = el.error;
                errorText = e ? (e.message || `media error ${e.code}`) : "unknown error";
                setPhase("error");
            });
            el.addEventListener("timeupdate", () => {
                const now = Date.now();
                if (now - lastTick < 500) return;
                lastTick = now;
                report();
                positionState();
            });
            return el;
        }

        function setPhase(next) {
            phase = next;
            report();
            if ("mediaSession" in navigator) {
                navigator.mediaSession.playbackState =
                    next === "playing" ? "playing" : next === "paused" ? "paused" : "none";
            }
        }

        function status() {
            const a = el;
            return {
                url: track.url || "",
                state: track.url ? phase : "idle",
                position: a ? (a.currentTime || 0) : 0,
                duration: a && Number.isFinite(a.duration) ? a.duration : 0,
                rate: a ? a.playbackRate : 1,
                error: errorText,
            };
        }

        function report() {
            const host = window.GrMobWASM;
            if (!host || typeof host.HostEvent !== "function") return;
            host.HostEvent("audio_status", JSON.stringify(status()));
        }

        // The lock-screen scrubber's notion of where playback is; a
        // best-effort call because setPositionState throws on a NaN
        // duration and on some older browsers.
        function positionState() {
            if (!("mediaSession" in navigator) || !el) return;
            try {
                if (Number.isFinite(el.duration)) {
                    navigator.mediaSession.setPositionState({
                        duration: el.duration,
                        playbackRate: el.playbackRate,
                        position: Math.min(el.currentTime, el.duration),
                    });
                }
            } catch (_) { /* unsupported here; the controls still work */ }
        }

        function installMediaSession() {
            if (!("mediaSession" in navigator)) return;
            const ms = navigator.mediaSession;
            ms.metadata = new MediaMetadata({
                title: track.title || "",
                artist: track.artist || "",
                album: track.album || "",
                artwork: track.artwork ? [{ src: track.artwork }] : [],
            });
            const set = (action, fn) => { try { ms.setActionHandler(action, fn); } catch (_) { } };
            set("play", () => play());
            set("pause", () => pause());
            set("stop", () => stop());
            set("seekbackward", (d) => skip(-(d.seekOffset || 15)));
            set("seekforward", (d) => skip(d.seekOffset || 15));
            set("seekto", (d) => { if (d.seekTime !== undefined) seek(d.seekTime); });
        }

        function clearMediaSession() {
            if (!("mediaSession" in navigator)) return;
            navigator.mediaSession.metadata = null;
            navigator.mediaSession.playbackState = "none";
        }

        function load(cmd) {
            if (!cmd.url) return;
            const a = element();
            track = cmd;
            errorText = "";
            phase = "loading";
            a.src = cmd.url;
            a.playbackRate = cmd.rate > 0 ? cmd.rate : 1;
            a.defaultPlaybackRate = a.playbackRate;
            if (cmd.start > 0) a.currentTime = cmd.start;
            a.load();
            installMediaSession();
            if (cmd.autoplay !== false) play();
        }

        function play() {
            if (!el || !track.url) return;
            // play() returns a promise that rejects when the browser's
            // autoplay policy refuses — which it will if this did not come
            // from a user gesture. Surfaced as an error state rather than a
            // silent stall so the app can show a play button.
            const p = el.play();
            if (p && p.catch) {
                p.catch((err) => {
                    errorText = err && err.message ? err.message : "playback was blocked";
                    setPhase("error");
                });
            }
        }

        function pause() { if (el) el.pause(); }

        function seek(seconds) {
            if (!el) return;
            const max = Number.isFinite(el.duration) ? el.duration : Infinity;
            el.currentTime = Math.max(0, Math.min(seconds, max));
            if (phase === "ended" && el.currentTime < max) phase = "paused";
        }

        function skip(delta) { if (el) seek(el.currentTime + delta); }

        function rate(r) {
            if (!el || !(r > 0)) return;
            el.playbackRate = r;
            el.defaultPlaybackRate = r;
        }

        function stop() {
            if (el) {
                el.pause();
                el.removeAttribute("src");
                el.load();
                el.playbackRate = 1; // Go's record resets to 1 on stop
                el.defaultPlaybackRate = 1;
            }
            track = {};
            errorText = "";
            phase = "idle";
            clearMediaSession();
            report();
        }

        // The "audio" system event's dispatcher. Unknown commands are
        // dropped, matching every host's contract for unknown events.
        function handle(cmd) {
            switch (cmd.command) {
                case "load": load(cmd); break;
                case "play": play(); break;
                case "pause": pause(); break;
                case "seek": seek(Number(cmd.position) || 0); break;
                case "skip": skip(Number(cmd.delta) || 0); break;
                case "rate": rate(Number(cmd.rate)); break;
                case "stop": stop(); break;
            }
        }

        return { handle, status };
    })();

    // The browser half of core's lifecycle event (core/lifecycle.go): is
    // the app on screen. The Page Visibility API is the one signal a page
    // gets that means what a phone's foreground/background means — a
    // switched tab, a minimized window, a phone browser sent to the home
    // screen all report hidden — so it is the source, and it can only tell
    // two states apart: visible is "active", hidden is "background". The
    // natives report "inactive" between the two; a page never does.
    //
    // Reported through the same window.GrMobWASM.HostEvent the audio
    // status uses, looked up per call for the same reason: the runtime
    // loads before the wasm module does, and a visibility change that
    // arrives before Go is up has nothing to tell and nobody to tell it.
    // Go dedupes a repeat of the current state (browsers fire the event
    // twice on some tab switches), so this reports every change verbatim.
    // Guarded so the runtime still loads where there is no document with
    // events — the verify harness's minimal DOM, say.
    (() => {
        if (typeof document === "undefined" || typeof document.addEventListener !== "function") return;
        document.addEventListener("visibilitychange", () => {
            const host = window.GrMobWASM;
            if (!host || typeof host.HostEvent !== "function") return;
            host.HostEvent("lifecycle", JSON.stringify({
                state: document.hidden ? "background" : "active",
            }));
        });
    })();

    return {
        mount,
        patch,
        showToast,
        audio,
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
        return;
    }
    if (name === "audio") {
        // core's audio service (core/audio.go): the page owns the one
        // player, and reports back over GrMobWASM.HostEvent.
        GrMob.audio.handle(JSON.parse(payloadJSON));
        return;
    }
    if (name === "open_url") {
        // A new browsing context is the web's nearest equivalent of handing a
        // URL to the OS: the app's own page keeps its state and its wasm
        // instance, which navigating in place would destroy. "noopener"
        // severs window.opener so the opened page cannot reach back into this
        // one, and "noreferrer" keeps the app's URL out of the destination's
        // logs — the links apps hand to core.OpenURL are third-party by
        // definition.
        //
        // A popup blocker can refuse this (window.open returns null) when the
        // call is not attributable to a user gesture. Nothing is reported:
        // core.OpenURL is fire-and-forget by contract, and a system event has
        // no return channel to report it over.
        const { url } = JSON.parse(payloadJSON);
        if (url) window.open(url, "_blank", "noopener,noreferrer");
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
