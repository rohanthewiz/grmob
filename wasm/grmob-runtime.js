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

        // After the children, not in createElement: the selection decides
        // which page is visible, so it has nothing to act on until the pages
        // exist. createElement built the bar (a function of the props alone);
        // this is the half that needs the subtree.
        if (node.Type === "TabView") {
            syncTabView(el);
        }

        // Same slot, same reason: an editor's rows are its children, and the
        // stale-line rule is a comparison between those rows and the buffer —
        // there was nothing to compare a moment ago. This is also what draws
        // the gutter, whose width is a function of how many rows there are.
        if (node.Type === "CodeEditor") {
            syncCodeEditor(el);
        }

        // Same slot, same reason: a map's markers are its children, so there is
        // nothing to add to the Leaflet layer until they exist. The map itself
        // may also have to wait for the tree to be appended before Leaflet can
        // measure it, which syncMap handles by deferring a frame.
        if (node.Type === "MapView") {
            syncMap(el);
        }

        // Likewise: the cell an overlay puts its layers in is a property of
        // the children, and there were none a moment ago.
        if (OVERLAY_TYPES.has(node.Type)) {
            syncOverlay(el);
        }

        // Same slot, same reason: the observer watches the last child, and
        // createElement ran before there were any. Gated on the prop so the
        // other several hundred nodes of a tree do not each pay an attribute
        // read to be told they are not lists.
        if (node.Props && node.Props.onEndReached) {
            syncEndReached(el);
        }

        return el;
    }

    function createElement(node) {
        const el = document.createElement(tagForType(node.Type));
        // The Go node type, kept on the element because the tag alone cannot
        // recover it (Row, Column, Card and Box are all divs) and update-style
        // patches carry only the changed Style — see the patch handler.
        el.dataset.nodeType = node.Type;

        // The style pass, run for every node — including one that carries no
        // Style at all, which is what the `|| {}` is for.
        //
        // styleFromGrMob is *total*, and several of the properties it assigns
        // are answers to the node TYPE rather than to the Style: the flex axis
        // a Row or a Column stacks along, the single-cell grid a ZStack draws
        // its layers in, the fixed rules of a TextGrid, and the three-valued
        // border that turns the user agent's own 2px outset rule off for a
        // <button>, an <input> and a <select>. A node with no Style needs
        // every one of those exactly as much as a node with one — so the empty
        // Style goes through the same function rather than each default being
        // restated here, which is what this used to do for three node types
        // and never did for the border.
        //
        // The update-style patch has always called applyStyle unconditionally
        // (reconcile emits the whole new Style, so the patch handler has no
        // "no style" case to guard). A conditional call *here* therefore meant
        // a styleless <button> kept the browser's border until some later
        // patch gave it a Style and took the border away — one node drawn two
        // ways depending on whether anything had touched it since it was
        // built. Nothing core constructs is ever styleless, since every widget
        // reads a theme base, so only a hand-assembled tree could reach it;
        // that made it cheaper to close than to keep documenting.
        applyStyle(el, node.Style || {}, node.Type);

        if (node.Type === "Modal") {
            // The one thing about a Modal that is neither a Style nor a prop
            // this element has been given yet: it starts closed, because
            // core.Visible defaults to false in Go. The `visible` prop in the
            // loop below sets the truth either way, and a hand-assembled node
            // that carries no such prop stays shut rather than splicing a
            // dialog's body into the middle of the page.
            //
            // The rest of the chassis — the fixed inset-0 box, the centred
            // flex column, the z-index — is in styleFromGrMob beside the grid
            // chassis, which is where a set of node-type defaults has to live
            // to survive an update-style patch. It used to be assigned here,
            // and a hand-assembled Modal carrying any Style at all had it
            // cleared out from under it by the total pass, with nothing to put
            // it back.
            //
            // The accessibility half — role="dialog" plus aria-modal — used to
            // be restated here too, because applyStyle did not run for a node
            // with no Style. It rides the unconditional call above now.
            el.style.display = "none";
        }

        // The map nodes' dataset, written here and on the update path for the
        // reason applyEnterKeyHint is called in both: the sync pass reads the
        // whole set at once, and Object.entries fixes no order between lat, lng
        // and zoom. Before the props loop is fine too — this writes dataset
        // entries the loop never touches — but after keeps the two call sites
        // reading alike.
        //
        // Unconditional, and it answers for its own node types: a node that is
        // neither a MapView nor a Marker leaves with nothing written.
        if (node.Props) {
            applyMapProps(el, node.Props, node.Type);
        }

        // The <input> variant, which the tag alone cannot express: tagForType
        // sends several Go node types to <input>, and an <input> with no type
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

        // The one thing the type attribute cannot say: that this checkbox is a
        // switch. HTML has no switch element and `switch` is its own answer (a
        // boolean attribute on a checkbox, WHATWG); Safari draws a track and a
        // thumb from it and most engines still draw the box, which is the same
        // bool in the same state rather than a broken control. What makes it
        // announce correctly everywhere is role="switch", and that arrives
        // through applyAccessibility, because a role has one slot per element
        // and an author's own Style may have filled it.
        //
        // The empty string is a *present* boolean attribute in the DOM, which
        // is all the attribute means. htmlout writes switch="switch" for the
        // same presence, because element emits key="value" pairs and repeating
        // the name is the spec-blessed spelling of a bare one.
        //
        // Here rather than on the update path, for the reason above it: a node
        // type cannot change under a patch.
        if (node.Type === "Switch") {
            el.setAttribute("switch", "");
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
                } else if (key === "onTabChange" && node.Type === "TabView") {
                    // Before the generic on* branch, for the reason onDismiss
                    // and onLongPress are: there is no "tabchange" DOM event,
                    // so that branch attached a listener nothing could ever
                    // fire and marked the slot taken so the real wiring could
                    // never be installed. The real trigger is a click on one
                    // of the bar's buttons, which reads this ID back off the
                    // dataset at fire time (buildTabBar).
                    el.dataset.listener_onTabChange = value;
                } else if (key === "selectedIndex" && node.Type === "TabView") {
                    // Recorded rather than acted on here: the selection needs
                    // the pages, which do not exist yet — renderNode syncs it
                    // once they do, and every later read (syncTabView) takes
                    // it from the element.
                    el.dataset.tabSelected = value;
                } else if (node.Type === "RichTextEditor" && RICH_EDITOR_PROPS.has(key)) {
                    // Handled together by applyRichTextProps after the loop, for
                    // the same reason the CodeEditor branch below exists: two of
                    // these keys would otherwise attach listeners for events the
                    // div never fires and mark the listener slot taken.
                } else if (node.Type === "CodeEditor" && CODE_EDITOR_PROPS.has(key)) {
                    // Handled together by applyCodeEditorProps after the loop.
                    // Ahead of every branch below for the reason onLongPress
                    // and onEndReached are ahead of the generic on* arm: two
                    // of these keys would otherwise attach listeners for
                    // events the <pre> never fires *and* mark the listener
                    // slot taken, so the real wiring could never be installed.
                } else if (key === "onLongPress") {
                    // Before the generic on* branch for the same reason
                    // onDismiss is: there is no "longpress" DOM event, so the
                    // generic path would attach a listener that never fires
                    // and mark the slot taken so the real wiring could never
                    // be installed.
                    attachLongPress(el, value);
                } else if (key === "onEndReached") {
                    // Before the generic on* branch, for the third time and
                    // the same reason: "endreached" is not a DOM event. The
                    // trigger is an IntersectionObserver over the list's last
                    // child, pointed at it by syncEndReached once renderNode
                    // has built the children.
                    attachEndReached(el, value);
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
                    // Remembered on the element as well as drawn, because a
                    // row inside a CodeEditor has to re-decide on every
                    // keystroke whether Go's colours still describe the
                    // buffer's line — and Go sends no new patch for a row
                    // whose runs did not change. See syncCodeRow. Harmless
                    // for a TextGrid row, which nothing ever asks again.
                    el.__grmobRuns = value;
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
            // After the loop for the same reason: the bar reads tabs,
            // selectedIndex and onTabChange together, and Object.entries fixes
            // no order between them.
            if (node.Type === "TabView") {
                buildTabBar(el, node.Props);
            }
            // And after it for a third reason of the same shape: a picker's
            // options and its value are one fact, and the value has to be
            // assigned once the options it names exist. The `value` branch in
            // the loop above already ran and did nothing, which is exactly
            // what a <select> does with a value it has no option for.
            if (node.Type === "Select") {
                applySelectOptions(el, node.Props.options, node.Props.value);
            }
        }

        // After the loop, and outside the `if (node.Props)` above, because an
        // editor with no props at all still needs its chrome: a <pre> with no
        // textarea in it is a picture of a code editor, not one.
        if (node.Type === "CodeEditor") {
            buildCodeEditor(el);
            applyCodeEditorProps(el, node.Props, true);
        }
        if (node.Type === "RichTextEditor") {
            buildRichTextEditor(el);
            applyRichTextProps(el, node.Props, true);
        }

        return el;
    }

    // --- TabView chrome ------------------------------------------------------
    //
    // core.TabView's wire contract is a tabs prop, a controlled selectedIndex,
    // an optional onTabChange callback ID and one child per page. Both natives
    // consume all four — Renderer.kt draws a Material TabRow above the selected
    // page, Renderer.swift a hand-rolled bar above the same — and this runtime
    // read none of them: a TabView was a bare box holding every page at once,
    // with no bar and no way to switch, so an app whose navigation *is* a
    // TabView had no navigation at all here.
    //
    // htmlout/tabview.go is the other half of this pass and carries the shared
    // reasoning: what the chrome is, why the pages are hidden rather than
    // dropped, and why the icon is not drawn. The two are authored twice rather
    // than shared, exactly as the Modal chassis is — a declaration list is a
    // string there and a property object here — with
    // TestRuntimeDrawsTheSameTabChrome pinning the part that is a contract
    // rather than a look: the roles, the ARIA state and the data attributes.
    //
    // # The bar is not a node
    //
    // It carries no data-node-path, no patch is ever addressed to it, and it is
    // marked data-grmob-chrome so the two places that convert a *node* child
    // index into a *DOM* child index can skip it (chromeOffset). Chrome always
    // precedes the node children, which is what keeps that conversion a fixed
    // offset instead of a search — and is the order a screen reader wants
    // anyway. It is the same trick a TextGrid row's runs use, one step harder:
    // those spans sit under a node with no children of its own, so nothing ever
    // had to count past them.
    const TAB_BAR_STYLE = {
        display: "flex",
        flexDirection: "row",
        alignItems: "stretch",
        flexShrink: "0",
        borderBottom: "1px solid rgba(128,128,128,0.35)",
    };

    // One tab: equal width (Android's TabRow, SwiftUI's .frame(maxWidth:
    // .infinity)) and a <button> talked out of looking like one. color/font
    // inherit is what makes the bar theme-neutral — the tabs take the app's own
    // ink and face, so the bar reads correctly on a surface this code knows
    // nothing about.
    const TAB_STYLE = {
        flex: "1 1 0",
        padding: "12px 8px 10px",
        border: "none",
        borderBottom: "2px solid transparent",
        background: "none",
        color: "inherit",
        font: "inherit",
        textAlign: "center",
        cursor: "pointer",
        opacity: "0.6",
    };

    // The two selection states, each stating every property the other does.
    // Object.assign only ever sets, so a tab that stops being selected has to
    // be given the unselected value of all three — the same totality rule
    // styleFromGrMob lives by, for the same reason: this pair is re-applied on
    // every sync, and a property left out is a property left standing.
    const TAB_SELECTED_STYLE = {
        opacity: "1",
        fontWeight: "600",
        borderBottomColor: "currentColor",
    };
    const TAB_UNSELECTED_STYLE = {
        opacity: "0.6",
        fontWeight: "",
        borderBottomColor: "transparent",
    };

    // The tags whose implicit ARIA role is `generic` — the ones that name
    // nothing to a screen reader on their own, so a role= attribute written
    // onto them adds meaning instead of replacing it.
    //
    // The one caller is the tab panel wiring below. A tab's aria-controls has
    // to name an element carrying role="tabpanel", and the page it points at is
    // whatever node type the app put there; stamping the role onto a <button>,
    // an <img> or an <input> page would take away the role the browser already
    // gives it, which is a worse outcome than leaving that page unwired. The
    // whole point of the wiring is accessibility, so it must not cost any.
    //
    // Go states this set once, in genericTags (htmlout/tag.go), and
    // TestRuntimeGenericTagsMatchGo in wasm/verify compares the two under a
    // plain `go test ./...`. That test reads this literal out of the source
    // textually, so keep it a flat array of string literals on one line.
    const GENERIC_TAGS = new Set(["div", "pre", "span"]);

    // The node types that state their own ARIA role, and the role each states.
    // Neither is a value in core.Role, which is the property this table
    // carries: these are roles the framework emits and does not name.
    //
    //   Modal    a dialog by virtue of being an overlay; core.ModalNode has no
    //            Style field, so the role has nothing else to ride on
    //   Switch   a switch by virtue of being one; the DOM has no switch
    //            element, so an <input type="checkbox"> plus this role is what
    //            a reader needs to announce the control correctly
    //
    // Go states this table once, in ownRoles (htmlout/tag.go), and
    // TestRuntimeOwnRolesMatchGo in wasm/verify compares the two under a plain
    // `go test ./...` — the same treatment tagForType, inputTypeFor,
    // GENERIC_TAGS and BORDER_RESET_TYPES get. That test reads this literal out
    // of the source textually, so keep it a flat object literal in a function
    // named ownRole.
    //
    // The cost of the two copies disagreeing is a role attribute on one target
    // and not the other: a control that announces itself on the web build and
    // not in the static export, or the reverse, which is a silence rather than
    // an error.
    function ownRole(nodeType) {
        return {
            Modal: "dialog",
            Switch: "switch",
        }[nodeType] || "";
    }

    // The node types whose user-agent stylesheet draws a border of its own, and
    // which therefore need one written back to nothing when the Go style asks
    // for no border.
    //
    // Compose, SwiftUI and htmlout all draw a border only when
    // BorderWidth > 0 && BorderColor != "". The guard alone gets the negative
    // case wrong on the web: a <button> with no border in its style keeps the
    // browser's own 2px outset rule, which no core.BorderWidth(0) can remove
    // because emitting nothing is exactly what leaves the user agent in charge.
    // components.Button's EmphasisGhost — "outlined without the rule" — is what
    // made that visible, drawing a rule on both web targets and none on either
    // phone.
    //
    // Keyed by node type rather than by tag because five node types share
    // <input> and only three of them want this: a checkbox's border *is* the
    // control and a range track has none. Both bundled themes now give
    // Components.Input and Components.TextArea a frame of their own, which is
    // what let the text fields join at all — resetting a border nothing
    // replaces is levelling down. borderResetTypes in htmlout/tag.go carries
    // the long version.
    //
    // Go states this set once, in borderResetTypes (htmlout/tag.go), and
    // TestRuntimeBorderResetTypesMatchGo in wasm/verify compares the two under
    // a plain `go test ./...`. That test reads this literal out of the source
    // textually, so keep it a flat array of string literals on one line.
    const BORDER_RESET_TYPES = new Set(["Button", "Input", "InputPassword", "NumericInput", "TextArea", "Select"]);

    // The node types whose children are drawn on top of one another rather
    // than along an axis — core.ZStack, the framework's one z-axis container.
    //
    // A single-cell CSS grid rather than absolute positioning, because an
    // absolutely positioned child is out of flow and contributes nothing to
    // its parent's size: an unsized overlay would collapse here while a
    // SwiftUI ZStack and a Compose Box both size to their largest child. Every
    // child is put in row 1, column 1 (OVERLAY_CHILD_AREA below), so the track
    // sizes to the biggest of them and the rest are drawn in the same cell.
    //
    // Go states this set once, in overlayTypes (htmlout/stack.go), and
    // TestRuntimeOverlayTypesMatchGo in wasm/verify compares the two under a
    // plain `go test ./...`. That test reads this literal out of the source
    // textually, so keep it a flat array of string literals on one line.
    const OVERLAY_TYPES = new Set(["ZStack"]);

    // The cell every layer of an overlay occupies. Assigned to the *children*
    // of an overlay, which is why it is a lone constant rather than part of
    // styleFromGrMob's output: a layer has no idea it is a layer, so the
    // container stamps it (syncOverlay). htmlout imposes the same declaration
    // through its `imposed` channel; the two are pinned together by
    // TestRuntimeOverlayChildAreaMatchesGo.
    const OVERLAY_CHILD_AREA = "1/1";

    // Where each layer sits inside that cell: core.StackAlignment -> the
    // [justify-self, align-self] pair. Go states this once, as stackPlacements
    // in htmlout/stack.go, and TestRuntimeStackPlacementsMatchGo in wasm/verify
    // compares the two under a plain `go test ./...`.
    //
    // Total, centre included, and both halves for the same reason the rest of
    // styleFromGrMob is total: syncOverlay restates every property it manages
    // on every pass, so a layer that *loses* its placement goes back to the
    // middle instead of keeping the last one it was given. The centre row is
    // also what stops a layer's own Style.AlignSelf — a flexbox property that
    // a grid item honours too — from moving it in contradiction of
    // core.ZStack's centre-on-both-axes contract.
    //
    // That test reads this literal out of the source textually, so keep it a
    // flat object literal with one `"key": ["a", "b"],` line per entry.
    const STACK_PLACEMENTS = {
        "": ["center", "center"],
        "top-start": ["start", "start"],
        "top": ["center", "start"],
        "top-end": ["end", "start"],
        "start": ["start", "center"],
        "end": ["end", "center"],
        "bottom-start": ["start", "end"],
        "bottom": ["center", "end"],
        "bottom-end": ["end", "end"],
    };

    // The id prefix every element id inside one TabView is built from.
    //
    // aria-controls and aria-labelledby are IDREFs, so the wiring cannot be
    // expressed without ids, and ids are document-global: two TabViews on one
    // page must not both call their first tab "tab-0". The node path is the
    // identity that is already unique per element here — it is what every patch
    // is resolved against — so the uniqueness of these ids is exactly the
    // uniqueness this runtime's addressing already rests on.
    //
    // It is also what makes the ids the *same* strings htmlout writes rather
    // than merely the same shape: that exporter walks the identical path (see
    // tabScope in htmlout/tabview.go), so a TabView at "root/1" is
    // "grmob-root-1-tab-0" on both web targets. The slashes become dashes so an
    // id is usable as a CSS selector fragment without escaping; a path is
    // "root" followed by digits, so no two paths can collapse onto one scope.
    function tabScope(el) {
        return "grmob-" + (el.getAttribute("data-node-path") || "").replace(/\//g, "-");
    }

    // The two ids one tab/page pair uses to point at each other.
    const tabId = (scope, i) => `${scope}-tab-${i}`;
    const panelId = (scope, i) => `${scope}-panel-${i}`;

    // Whether a page element can carry the panel half of the wiring.
    //
    // Two of the three rules htmlout states in tabPanelBoxes; the third — "the
    // page renders as exactly one element" — is a question only that exporter
    // has to ask, because it drops the box for a Fragment or a Theme while this
    // runtime boxes both (see transparentTypes in htmlout/tag.go). Here page i
    // is always exactly the element in child slot i, so there is always exactly
    // one element to name.
    //
    // aria-hidden is read back off the element rather than out of the Style,
    // because that is where applyAccessibility put it and this function runs
    // long after: an author who took a page out of the accessibility tree on
    // purpose should not have it named as a panel, which would assert a
    // relationship they severed.
    //
    // A role is read back the same way and for the same reason, with one extra
    // step: the attribute has two writers — the author's
    // core.AccessibilityRole and this wiring's own "tabpanel" — so a bare
    // "has a role" test would read the panel this function wired last time as
    // an author's role and unwire it on the next sync, then rewire it on the
    // one after.
    //
    // What tells them apart is data-grmob-panel, a marker this wiring stamps
    // on every panel it writes, in the same channel data-grmob-chrome uses for
    // the same kind of fact: this is something the framework put here.
    //
    // It used to be the *value*. "tabpanel" was a string no core.Role spelled,
    // so an element carrying it could only have got it from here — which
    // worked, and which made the absence of a core.RoleTabPanel constant
    // load-bearing, so a hand-built tab strip could never name its own panels
    // however precisely it wired them. The marker says the thing the value was
    // standing in for, and it says it about *this element* rather than about
    // the vocabulary.
    //
    // "group" is the one core.Role that is *also* accepted, and it is the one
    // value it is not theft to replace. A group says these things belong
    // together and this is what they are called; a tabpanel says all of that
    // and which tab shows it, so writing one over the other adds a fact rather
    // than destroying one — which is the test every rule here applies. It has
    // to be accepted, besides: applyAccessibility supplies `group` to any named
    // page whether the author asked for it or not (see ariaRole), so rejecting
    // it would silently stop every page carrying an AccessibilityLabel from
    // being wired at all.
    //
    // The id is the third slot with two writers, and unlike the role it cannot
    // be shared: a page given a core.Style.AccessibilityID has an author's
    // string in it, and something else on the page is pointing at that string.
    // So a page keeps its own id and goes unwired. Comparing against this
    // wiring's own value is what tells an author's id from one this function
    // wrote a sync ago — the same discrimination the role does one line up.
    function canBeTabPanel(page, scope, i) {
        const role = page.getAttribute("role");
        const id = page.getAttribute("id");
        const mine = page.dataset.grmobPanel !== undefined;
        return (
            GENERIC_TAGS.has(page.tagName.toLowerCase()) &&
            page.getAttribute("aria-hidden") !== "true" &&
            (role === null || role === "group" || (role === "tabpanel" && mine)) &&
            (id === null || id === panelId(scope, i))
        );
    }

    // The panel half of the wiring, applied to one page element.
    //
    // Total, like everything else syncTabView does: every attribute is written
    // or removed on every call. A page can stop being eligible without being
    // replaced — an update-style can set AccessibilityHidden on it, and a
    // shrinking tabs prop can leave it with no tab — and a guarded write would
    // leave a role and a dangling reference standing after the reason for them
    // was gone.
    //
    // aria-labelledby is dropped when the page names itself. A panel is
    // normally named by its tab, which is the whole point of the reference, but
    // core.Style.AccessibilityLabel is an explicit act by the app author and
    // aria-labelledby wins over aria-label in the accessible-name calculation —
    // so writing it unconditionally would silently discard the name they chose.
    // The tab still points *at* the panel either way; only the naming is left
    // to the author.
    function wireTabPanel(page, scope, i, wired) {
        const named = !!page.getAttribute("aria-label");
        const mine = page.dataset.grmobPanel !== undefined;
        if (wired) {
            page.setAttribute("id", panelId(scope, i));
            page.setAttribute("role", "tabpanel");
            page.dataset.grmobPanel = "";
            setOrRemove(page, "aria-labelledby", named ? "" : tabId(scope, i));
            return;
        }
        // Not wired. aria-labelledby comes off whatever the reason, because
        // this wiring is its only writer — no core.Style field maps onto it —
        // so an unwired page carrying one is claiming to be named by a tab
        // that is not pointing back.
        setOrRemove(page, "aria-labelledby", "");
        // The other two slots are shared with core.Style, and only what this
        // function wrote comes off them. The marker is what makes that
        // distinction possible, and the id half of it is a bug the old
        // unconditional `setOrRemove(page, "id", "")` had: a page carrying its
        // own core.Style.AccessibilityID is exactly the page this wiring stands
        // down for, and it was being stood down for by having that id deleted.
        // The static export never had the bug, because it decides once and
        // writes nothing it did not decide.
        if (!mine) {
            return;
        }
        delete page.dataset.grmobPanel;
        // By prefix rather than by equality, because a page can move: a tab
        // reorder leaves page i holding the id this wiring minted for slot j,
        // which is still this wiring's to clear. The "grmob-" prefix is
        // reserved for exactly this (see core.Style.AccessibilityID), so a
        // string that matches cannot be an author's.
        const id = page.getAttribute("id");
        if (id !== null && id.startsWith(scope + "-panel-")) {
            page.removeAttribute("id");
        }
        // The role is the one attribute here this function does not own
        // outright: applyAccessibility writes the author's core.Role — or the
        // group it supplies to a named container — into the same slot. So the
        // unwired case clears only the wiring's own value, and puts back what
        // applyAccessibility would have left there, which for a page eligible
        // to be wired at all can only be `group` (ariaRole's fallback) or
        // nothing. Restoring rather than clearing matters because nothing
        // guarantees another style patch: a page that stops being wired — its
        // tab was dropped, the tabs prop shrank — would otherwise sit with an
        // aria-label no browser announces, which is the exact silence the
        // fallback exists to close.
        //
        // The corner it does not restore is an authored core.RoleGroup on a
        // page with no name, which comes back as no role. Nothing is lost that
        // a reader could hear: an unnamed group adds no node to the
        // accessibility tree.
        if (page.getAttribute("role") === "tabpanel") {
            setOrRemove(page, "role", named ? "group" : "");
        }
    }

    // How many leading children of this element are chrome rather than nodes.
    //
    // Counted rather than derived from the node type, because a TabView with no
    // tabs prop has no bar: "TabView" alone would answer 1 for an element whose
    // first child is really its first page, and every add patch aimed at that
    // TabView would then land one slot late.
    function chromeOffset(el) {
        let n = 0;
        while (n < el.children.length && el.children[n].dataset.grmobChrome !== undefined) {
            n++;
        }
        return n;
    }

    // How many of this element's children are *nodes* — the ones carrying a
    // data-node-path — rather than chrome the runtime drew itself.
    //
    // Deliberately not `children.length - chromeOffset(el)`, which is the same
    // number only while every piece of chrome is leading. A CodeEditor's filler
    // rows are trailing (they stand in for buffer lines Go has not sent a row
    // for yet), so counting the node children directly is the only formulation
    // that survives them. It is also the more honest question: "how many
    // children answer to a path" is what an add-child index means.
    function nodeChildCount(el) {
        let n = 0;
        for (const child of el.children) {
            if (child.getAttribute("data-node-path") !== null) n++;
        }
        return n;
    }

    // Builds (or refreshes) a TabView's bar. Called after the props loop on
    // both the create and the update path, because the bar is a function of the
    // whole props map and Object.entries fixes no order between tabs,
    // selectedIndex and onTabChange — the same reason applyEnterKeyHint is
    // called after its loop rather than inside it.
    //
    // Rebuilt only when the tab strip itself changed, which is what the
    // signature is for. selectedIndex changes on every switch, and a switch is
    // exactly when a keyboard user has one of these buttons focused; rebuilding
    // unconditionally would throw that focus away on every tab press. The
    // selection is applied by syncTabView instead, which mutates the buttons in
    // place.
    //
    // props is the whole new props map on both paths — reconcile emits
    // new.Props and never a delta — so an absent tabs key means "no tabs now",
    // and removing the bar is the right reading of it rather than a lost
    // update.
    function buildTabBar(el, props) {
        const tabs = Array.isArray(props.tabs) ? props.tabs : [];
        const signature = JSON.stringify(tabs.map(t => (t && t.label) || ""));

        const first = el.children[0];
        const existing = first && first.dataset.grmobChrome === "tabbar" ? first : null;
        if (existing && existing.dataset.tabSignature === signature) return;
        if (existing) existing.remove();
        if (!tabs.length) return;

        const bar = document.createElement("div");
        bar.dataset.grmobChrome = "tabbar";
        bar.dataset.tabSignature = signature;
        bar.setAttribute("role", "tablist");
        Object.assign(bar.style, TAB_BAR_STYLE);

        tabs.forEach((tab, i) => {
            const button = document.createElement("button");
            // type="button" so a bar inside a <form> cannot submit it.
            button.setAttribute("type", "button");
            button.setAttribute("role", "tab");
            button.dataset.tabIndex = i;
            Object.assign(button.style, TAB_STYLE);
            button.textContent = (tab && tab.label) || "";
            button.addEventListener("click", () => {
                // Re-read at fire time, exactly as every other listener in this
                // file does: callback IDs are positional and a later pass may
                // have refreshed or pruned this one. The ID lives on the
                // TabView, not on the button — one handler serves every tab and
                // the index is what distinguishes them, which is the shape
                // core.OnTabChange has.
                const latestCbId = el.dataset.listener_onTabChange;
                if (!latestCbId) return;
                // A number, so ReceiveEventPayload's float64 case routes it to
                // the *int* callback map. Sending the index as a string would
                // land it in the text map, where an int ID does not exist, and
                // the tap would silently do nothing.
                window.GoInvokeCallback(latestCbId, { value: i });
            });
            bar.appendChild(button);
        });

        el.insertBefore(bar, el.children[0] || null);
    }

    // Applies a TabView's selection: which tab reads as selected, and which
    // page is visible.
    //
    // Separate from buildTabBar because this is the part that runs constantly —
    // see the end of patch(), which recomputes it for every TabView a batch
    // could have disturbed. It is idempotent and reads only the element, so it
    // is safe to call more often than strictly needed, and that is the whole
    // design: no individual patch case has to know about tabs.
    //
    // An out-of-range index selects no tab and shows no page, deliberately
    // unclamped — Renderer.swift compares `i == selected` for the indicator and
    // guards the page with `children.indices.contains`, and Renderer.kt guards
    // the page with getOrNull. See tabSelectedIndex in htmlout/tabview.go.
    function syncTabView(el) {
        const selected = Number(el.dataset.tabSelected || 0);
        const offset = chromeOffset(el);
        const bar = offset ? el.children[0] : null;
        // How many tabs there are, which is what decides whether page i is part
        // of a tab set at all. A TabView with no tabs prop draws no bar and so
        // wires no panels; one with fewer tabs than pages wires only the pages
        // a tab can point at.
        const tabCount = bar ? bar.children.length : 0;
        const scope = tabScope(el);

        if (bar) {
            // The callback ID mirrored onto the bar, so the live DOM carries
            // the same data-ontabchange the export writes. The listener above
            // is what actually dispatches; this is the record of it.
            const cbId = el.dataset.listener_onTabChange;
            if (cbId) {
                bar.setAttribute("data-ontabchange", cbId);
            } else {
                bar.removeAttribute("data-ontabchange");
            }
            for (let i = 0; i < tabCount; i++) {
                const tab = bar.children[i];
                const on = i === selected;
                tab.setAttribute("aria-selected", on ? "true" : "false");
                Object.assign(tab.style, on ? TAB_SELECTED_STYLE : TAB_UNSELECTED_STYLE);
                // The id goes on every tab whether or not it controls a panel:
                // it costs nothing, an id names nothing on its own, and it lets
                // the panel point back here without the two halves of the
                // wiring having to agree twice about which tabs are eligible.
                tab.setAttribute("id", tabId(scope, i));
                // aria-controls only where there is a panel to control. A
                // dangling IDREF is worse than an absent one: a screen reader
                // announcing a tab that controls a region, and finding no such
                // region, is a lie about the document — whereas a tab with no
                // aria-controls is merely one that has not said what it governs.
                const page = el.children[offset + i];
                setOrRemove(tab, "aria-controls",
                    page && canBeTabPanel(page, scope, i) ? panelId(scope, i) : "");
            }
        }

        for (let i = offset; i < el.children.length; i++) {
            const page = el.children[i];
            const index = i - offset;
            // baseDisplay is the display this page would have with nothing
            // hiding it, recorded by applyStyle — which is now the single
            // place this runtime decides a display at all. Restoring it rather
            // than clearing the declaration is the point: a Column page cleared
            // to "" would lose the display:flex every stack container gets, and
            // come back as block flow.
            page.style.display = index === selected ? (page.dataset.baseDisplay || "") : "none";
            // The panel half. It rides on the same pass as the hiding for the
            // reason the hiding is here at all: both are functions of the
            // TabView's props *and* of which children it currently has, and
            // recomputing them together at the end of a batch is what keeps
            // every individual patch case from having to know about tabs.
            wireTabPanel(page, scope, index, index < tabCount && canBeTabPanel(page, scope, index));
        }
    }

    // Recomputes the selection of every TabView at or above the elements a
    // patch batch touched.
    //
    // The selection is derived state — a function of the TabView's props and of
    // its children — and a batch can invalidate it three different ways: an
    // update-props carrying a new selectedIndex, an update-style on a page
    // (styleFromGrMob is total, so it assigns a display on every pass and
    // overwrites the hiding), or an add/remove/replace that changes which
    // children there are. Recomputing once at the end is what keeps those three
    // cases from each having to know about tabs. Nothing paints in between: a
    // batch is one synchronous run.
    // Puts every child of an overlay in the stack's one grid cell.
    //
    // The declaration belongs on the children and the knowledge belongs to the
    // parent, which is the whole reason this is a pass rather than a line in
    // styleFromGrMob: a layer has no idea it is a layer. htmlout says the same
    // thing through its `imposed` channel, where the parent likewise writes a
    // declaration into markup the child assembles.
    //
    // Idempotent, and re-run rather than tracked: assigning the same string to
    // the same property is free, and the alternative — remembering which
    // children have been stamped — is state that can be wrong.
    //
    // Nothing clears the property on a child that leaves an overlay, because
    // nothing can: a node type change is a replace (reconcile/patch.go), which
    // discards the element and everything on it. A child that merely moves
    // between overlays is stamped identically by both.
    function syncOverlay(el) {
        for (const child of el.children) {
            child.style.gridArea = OVERLAY_CHILD_AREA;
            // core.Style.StackAlign, which reached the element as a data
            // attribute in applyStyle. It is read back here rather than
            // written there because the placement is the *stack's* to impose:
            // a layer has no idea it is a layer, and justify-self/align-self
            // written by any node would move a flex item too. htmlout makes
            // the same split through its `imposed` channel.
            const place = STACK_PLACEMENTS[child.dataset.stackAlign || ""]
                || STACK_PLACEMENTS[""];
            child.style.justifySelf = place[0];
            child.style.alignSelf = place[1];
        }
    }

    // The overlay half of syncTouchedTabViews, and it walks upward for the
    // same reason: a patch names the element it changed, and what needs
    // re-stamping is the overlay somewhere above it. An "add" lands a brand
    // new child under a ZStack that was itself untouched, so a pass over the
    // touched elements alone would leave exactly the new layer unplaced.
    function syncTouchedOverlays(touched) {
        const done = new Set();
        for (const start of touched) {
            for (let el = start; el; el = el.parentNode) {
                if (el.dataset && OVERLAY_TYPES.has(el.dataset.nodeType) && !done.has(el)) {
                    done.add(el);
                    syncOverlay(el);
                }
            }
        }
    }

    function syncTouchedTabViews(touched) {
        const done = new Set();
        for (const start of touched) {
            for (let el = start; el; el = el.parentNode) {
                if (el.dataset && el.dataset.nodeType === "TabView" && !done.has(el)) {
                    done.add(el);
                    syncTabView(el);
                }
            }
        }
    }


    // --- The keyboard half of a composite widget -----------------------------
    //
    // core.Role is a vocabulary: it says what a node *is* and stops there, and
    // its own doc says so where it declares the two pairs this section is
    // about —
    //
    //     A listbox is a real control in ARIA's model, and the pattern that
    //     goes with it is larger than two attributes: the container takes
    //     keyboard focus, the arrow keys move an active option, and the reader
    //     is told which option is active through a roving tabindex or
    //     aria-activedescendant.
    //
    // None of that was anywhere. Three consumers shipped a role that claimed
    // more than the widget did: examples/mobileapp's article list is a
    // listbox of divs that no keyboard could reach at all, and
    // examples/social's bottom bar and tutorial 4.5 are tablists of buttons
    // that a keyboard could reach only by tabbing through every one of them.
    // The first is a control with no keyboard operation; the second announces
    // "tab, 1 of 3" and then behaves like three unrelated buttons.
    //
    // # Why this is here and not in core
    //
    // The entry that asked for this said it needed a focus concept core does
    // not have. It does not, and the reason is worth stating because it is
    // what made the work small: everything the pattern needs is already on
    // the wire.
    //
    //     what is a member of what   role="listbox" / "option", role="tablist"
    //                                / "tab" — a structural role owns what is
    //                                inside it, which core/role.go states as a
    //                                rule an author has to keep
    //     which one is chosen        aria-selected, which both pairs already
    //                                carry and which a strip sets on every
    //                                member rather than only the live one
    //     which way the arrows go    the container's own flex-direction, which
    //                                this runtime planted from stackAxisFor
    //     what activation means      the onClick the author already wired
    //
    // So there is no new prop, no new core type, and no source change in any
    // of the three consumers — they were already saying all of it. What was
    // missing was a target that reads it.
    //
    // core/focus.go is not the missing piece either. That file is about
    // putting the cursor in a *named* field, from Go, as a command that rides
    // the render tree; this is about which of a widget's own members holds the
    // one tab stop, which changes on a keystroke with no render in between.
    // Routing it through Go would be a render pass per arrow key.
    //
    // # Both phones lose nothing
    //
    // VoiceOver and TalkBack navigate a collection by swipe, not by arrow key,
    // and neither native has a listbox in its semantics vocabulary at all —
    // both spell a chosen item as the `selected` state and honour it on any
    // node without being told what contains it. So this is a web-target
    // concern in the same way the Modal chassis is, and there is no native arm
    // missing.
    //
    // # htmlout writes none of it, deliberately
    //
    // The static exporter is not a runtime (docs/platforms/exporters.md says
    // so, and its TabView bar is already inert chrome). A roving tabindex
    // *without* the key handler that moves it is strictly worse than nothing:
    // it takes every member but one out of the tab order and supplies no way
    // to reach them, so a static export would go from "three tab stops" to
    // "one tab stop and two unreachable items". tabindex here is behaviour,
    // not semantics, and it is only correct in the presence of the code below.

    // # Two ways to be a composite, and why the second one had to exist
    //
    // A composite is a widget that holds ONE tab stop and moves it among its
    // members with the arrow keys. Everything in this section is driven by the
    // question "what are this container's members", and ARIA answers it two
    // different ways depending on the role.
    //
    //	listbox, tablist   the role names its members. A listbox's members are
    //	                   its options, a tablist's are its tabs, and the pair
    //	                   is the whole rule.
    //	toolbar            the role does not. ARIA says a toolbar is "a
    //	                   collection of commonly used function buttons or
    //	                   controls" and stops there: a toolbar may hold
    //	                   buttons, links, groups, separators and inputs, and
    //	                   there is no `toolbaritem`.
    //
    // The second rule is the one this whole section was restructured for. For
    // two releases `toolbar` was in the orientation table and out of the
    // keyboard: the axis was announced and no tab stop moved, which left
    // components.ChipStrip a run of controls a keyboard crosses one Tab at a
    // time — twelve stops on a twelve-chip filter bar, where ARIA promises one.
    //
    // The rule for a toolbar's members is the one the absence forces: every
    // focusable control inside it that is not inside a nested composite. See
    // focusableMembers.

    // The two structural pairs, container role to member role.
    //
    // Kept to the two roles that name their members *and* own their children.
    // `list`/`listitem` is content rather than a control and has no pattern.
    //
    // `menu`, `tree` and `grid` are the three patterns still absent, and they
    // are absent for two different reasons that aria/verify/refusals_test.go
    // now holds apart: `menu` and `tree` need member roles core does not carry
    // (`menuitem`, `treeitem`), where `grid` needs no new member role at all —
    // core already has `row` and `cell` — and needs a two-dimensional walk this
    // one-dimensional machinery has no shape for.
    const COMPOSITE_MEMBERS = { listbox: "option", tablist: "tab" };

    // Composites whose members ARIA does not name. See focusableMembers.
    //
    // A set rather than a second column on the table above, because the two
    // are different *kinds* of answer rather than two values of one: a row in
    // COMPOSITE_MEMBERS is a fact about ARIA's vocabulary that
    // wasm/verify/keynav_test.go pins against core.Role, and membership here is
    // a decision about a walk. Collapsing them into one table with a sentinel
    // value would make the pin read a sentinel as a role name.
    const COMPOSITE_FOCUSABLE = new Set(["toolbar"]);

    // The member roles, derived rather than restated. compositeOf needs to know
    // whether an element's own role makes it a named member of something —
    // which decides which of the two walks out it takes — and deriving that
    // from the table is what keeps a third structural pair from having to be
    // remembered in two places.
    const MEMBER_ROLES = new Set(Object.values(COMPOSITE_MEMBERS));

    // What a toolbar counts as one of its controls.
    //
    // Two ways to be one, and they are the two ways this framework builds a
    // control at all:
    //
    //	a natively focusable tag       core.Button, core.Input, core.Select,
    //	                               core.TextArea, an anchor. The browser
    //	                               gives these a tab stop without being
    //	                               asked, which is exactly the tab stop the
    //	                               toolbar is taking over.
    //	a container that says it is    a Box or a Row carrying core.RoleButton
    //	a control and has a handler    or core.RoleLink with an OnTap — the
    //	                               shape core.RoleButton's own doc exists
    //	                               for. It has no tab stop of its own until
    //	                               this section gives it one.
    //
    // Deliberately NOT "anything carrying tabindex". This section writes
    // tabindex onto every member it finds, so a membership test that read the
    // attribute would answer differently on the second sync than on the first
    // — every member would stay a member forever, including one whose role or
    // tag had since changed.
    //
    // FOCUSABLE_TAGS is the same five tags as NATIVELY_ACTIVATED below and is
    // deliberately a second set: that one is about which elements the browser
    // fires a click on for Enter and Space, this one is about which elements it
    // will focus. The two coincide today and are not one fact — an <a> with no
    // href is activated and not focusable — so a single set would be a place
    // for the two claims to drift into each other unnoticed.
    //
    // The role half is core.TappableContainerRoles(), which is where the fact
    // lives: core's own role_control_test.go holds every role in the
    // vocabulary to one side of the question or the other, so a role added
    // there cannot quietly turn out to be a control a toolbar steps over.
    // wasm/verify/keynav_test.go pins these two strings to that list.
    const FOCUSABLE_TAGS = new Set(["BUTTON", "A", "INPUT", "SELECT", "TEXTAREA"]);
    const CONTROL_ROLES = new Set(["button", "link"]);

    // Elements the browser already activates from the keyboard. A <button>
    // fires a real click on both Enter and Space and an <a href> on Enter, so
    // synthesizing one here would run the author's handler twice.
    const NATIVELY_ACTIVATED = new Set(["BUTTON", "A", "INPUT", "SELECT", "TEXTAREA"]);

    // The composites that answer a printable key by jumping to a member whose
    // name starts with it.
    //
    // A listbox and not a tablist, which is ARIA's own division rather than a
    // shortcut: type-to-jump is part of the listbox pattern because a listbox
    // can be a hundred options long and the arrows are useless at that size,
    // and it is not part of the tab pattern because a tab strip has three
    // members and they are all on screen. examples/mobileapp's article list is
    // the shape that asked.
    const COMPOSITE_TYPEAHEAD = new Set(["listbox"]);

    // How long a typed string stays live. ARIA's authoring practices suggest
    // "roughly 500ms" and every implementation picks a number in that
    // neighbourhood; the exact value matters less than there being one, since
    // it is what separates "se" meaning Sermons from an s and an e typed a
    // minute apart.
    const TYPEAHEAD_RESET_MS = 500;

    // The one piece of *state* this whole section owns.
    //
    // Everything else here is derived from the DOM on demand — which member
    // holds the stop, which container a member belongs to, which way the arrows
    // go — and that is what makes the rest survive every patch for free: there
    // is nothing to invalidate. A search string cannot be derived from
    // anything, so it is held, and the holding is kept as small as it can be:
    //
    //	one buffer, not one per widget   only one thing has focus at a time, so
    //	                                 a second widget's buffer could never be
    //	                                 the live one. The container is recorded
    //	                                 beside the text so a keystroke in a
    //	                                 different widget starts over rather
    //	                                 than continuing someone else's search.
    //	a timestamp, not a timer         setTimeout would need cancelling on
    //	                                 every unmount, and a widget removed by
    //	                                 a patch has no unmount hook to cancel
    //	                                 it from. Expiry is checked on the next
    //	                                 keystroke instead, which is the only
    //	                                 moment it can matter.
    //
    // The container reference is dropped the next time anyone types anywhere.
    // A widget removed by a patch while its buffer is live is therefore held
    // by this object until then — one element, replaced by the next keystroke
    // — which is the cost of not owning a timer.
    const typeahead = { container: null, text: "", at: 0 };

    function compositeMemberRole(el) {
        if (!el || !el.getAttribute) return "";
        return COMPOSITE_MEMBERS[el.getAttribute("role")] || "";
    }

    // Whether this element is a composite of either kind.
    //
    // The one question the walks ask about an *ancestor or a stranger*, as
    // opposed to compositeMemberRole, which is the question a role-based walk
    // asks about its own container. Both walks stop at this: a nested composite
    // owns its members and its own tab stop, and pooling them would give one
    // widget two.
    function isComposite(el) {
        if (!el || !el.getAttribute) return false;
        const role = el.getAttribute("role");
        return !!COMPOSITE_MEMBERS[role] || COMPOSITE_FOCUSABLE.has(role);
    }

    // A container's members, by whichever rule its role uses. The single entry
    // point: nothing outside this function picks between the two walks.
    function compositeMembersOf(container) {
        const memberRole = compositeMemberRole(container);
        if (memberRole) return compositeMembers(container, memberRole);
        if (COMPOSITE_FOCUSABLE.has(container.getAttribute("role"))) {
            return focusableMembers(container);
        }
        return [];
    }

    // Whether an element is one of the controls a toolbar takes over the tab
    // stop of. See FOCUSABLE_TAGS and CONTROL_ROLES for the two ways.
    function isFocusableControl(el) {
        if (el.disabled) return false;
        if (FOCUSABLE_TAGS.has(el.tagName)) return true;
        return CONTROL_ROLES.has(el.getAttribute("role")) &&
            !!el.dataset.listener_onClick;
    }

    // The members of one composite, in document order.
    //
    // A subtree walk rather than a children scan, because nothing says a
    // member is a direct child: components.ListRow renders a row inside
    // whatever core.For and core.Keyed wrap it in, and a tab strip may have
    // its buttons inside a scroller. The walk stops at three things:
    //
    //   a nested composite of the same kind   its members are its own, and a
    //                                         listbox inside a listbox would
    //                                         otherwise pool both sets
    //   an aria-hidden subtree                pruned from the accessibility
    //                                         tree, so it has no members
    //   a disabled form control               the browser refuses it focus
    //                                         outright, so arrowing onto it
    //                                         would move the tab stop to a
    //                                         place no focus can follow. An
    //                                         aria-disabled div is *not*
    //                                         excluded: it is still focusable,
    //                                         and ARIA keeps a disabled option
    //                                         reachable so a user can tell it
    //                                         is there.
    function compositeMembers(container, memberRole, out = []) {
        for (const child of container.children) {
            if (!child.getAttribute) continue;
            // An aria-hidden *subtree*. The member case is already covered
            // upstream — applyAccessibility drops the role of a hidden node,
            // so a hidden option is not an option here — but a hidden wrapper
            // keeps its children's roles, and those children are pruned from
            // the accessibility tree along with it.
            if (child.getAttribute("aria-hidden") === "true") continue;
            if (child.getAttribute("role") === memberRole) {
                if (!child.disabled) out.push(child);
                // A member is a leaf of this walk even when it holds elements:
                // its contents belong to it, and a tab inside a tab is not a
                // shape ARIA has.
                continue;
            }
            if (compositeMemberRole(child) === memberRole) continue;
            compositeMembers(child, memberRole, out);
        }
        return out;
    }

    // The other member walk: a toolbar's controls, in document order.
    //
    // Same subtree descent as compositeMembers and the same two prunings — an
    // aria-hidden subtree is not in the accessibility tree, and a disabled form
    // control cannot take the focus a tab stop implies. What differs is what
    // ends the descent.
    //
    // # A nested composite stops the walk, and does not become a member
    //
    // compositeMembers descends *through* a composite of the other kind,
    // because an option below a tablist is still the listbox's option — the
    // roles say whose it is. Nothing says whose a button is. So a control
    // inside a nested composite belongs to the nested one, and the walk stops
    // there rather than pooling both sets.
    //
    // The consequence is worth stating rather than hiding: a tablist inside a
    // toolbar keeps its own roving tabindex, so the toolbar has one tab stop
    // and the tablist has a second, and the toolbar's arrows step over the
    // whole strip. Two stops is not what ARIA describes for that shape, and it
    // is the honest outcome of a rule that will not guess — it leaves every
    // control reachable, which the alternatives do not. Nothing in this
    // repository builds one.
    //
    // # A member is a leaf
    //
    // A control holding elements — a Button with an icon and a label — is one
    // member, not two, so the walk does not descend into one it has found.
    function focusableMembers(container, out = []) {
        for (const child of container.children) {
            if (!child.getAttribute) continue;
            if (child.getAttribute("aria-hidden") === "true") continue;
            if (isComposite(child)) continue;
            if (isFocusableControl(child)) {
                out.push(child);
                continue;
            }
            focusableMembers(child, out);
        }
        return out;
    }

    // The composite a member belongs to, or null. Walks out rather than
    // searching, so a member nested three containers deep finds the same
    // answer its container's walk reached it from.
    //
    // Two rules again, and they are the inverses of the two walks in.
    //
    // For a role-named member the match is on the member's *own* role rather
    // than on "the nearest composite of any kind", and the difference is what
    // keeps this the inverse of compositeMembers. That walk descends through a
    // composite of the other kind — a tablist inside a listbox is not a
    // listbox's member and does not close it — so an option below one is still
    // the listbox's member, and a walk out that stopped at the tablist would
    // disagree with the walk in. Nobody writes that tree on purpose; the two
    // functions still have to answer the same question the same way.
    //
    // For a focusable member there is no role to match on, so the rule is the
    // literal inverse of focusableMembers: the nearest composite of any kind,
    // and it is only this member's owner if that composite is one of the ones
    // that take focusable members. A button inside a listbox is not a member of
    // anything — it is chrome the option walk stepped over — and answering
    // "the listbox" here would put the toolbar keyboard on a widget that has
    // its own.
    function compositeOf(member) {
        const role = member.getAttribute("role");
        if (MEMBER_ROLES.has(role)) {
            for (let el = member.parentNode; el && el.getAttribute; el = el.parentNode) {
                if (compositeMemberRole(el) === role) return el;
            }
            return null;
        }
        for (let el = member.parentNode; el && el.getAttribute; el = el.parentNode) {
            if (!isComposite(el)) continue;
            return COMPOSITE_FOCUSABLE.has(el.getAttribute("role")) ? el : null;
        }
        return null;
    }

    // Which member holds the widget's one tab stop.
    //
    // The order is what keeps two different things right at once, and both
    // were bugs in the version that only looked at aria-selected:
    //
    //   focus is inside      the member holding it. A user who has arrowed to
    //                        the third option without choosing it must not
    //                        have the tab stop yanked back to the second by
    //                        an unrelated patch landing.
    //   otherwise            the selected member, which is where ARIA says
    //                        Tab should enter a widget — and where a click
    //                        that changed the selection has just moved it.
    //   nothing selected     whatever already holds the stop, so a widget with
    //                        no selection at all still remembers where the
    //                        user left it.
    //   nothing at all       the first member, so the widget is enterable on
    //                        its very first render.
    function activeMemberIndex(members) {
        const focused = document.activeElement;
        const held = members.indexOf(focused);
        if (held >= 0) return held;
        const selected = members.findIndex(
            (m) => m.getAttribute("aria-selected") === "true");
        if (selected >= 0) return selected;
        const stop = members.findIndex((m) => m.getAttribute("tabindex") === "0");
        return stop >= 0 ? stop : 0;
    }

    // Writes the roving tabindex and wires the members' keys.
    //
    // Total over the member list on every call, for the reason applyStyle and
    // applyAccessibility are total: a member that stops being the active one
    // has to lose the "0", and a guarded write would leave two tab stops in
    // one widget. The listener is the one thing that is not re-done, because
    // adding it twice would run the handler twice — a fresh element from a
    // replace has no stamp and gets one, which is the case the stamp exists
    // for.
    function syncComposite(container) {
        const members = compositeMembersOf(container);
        if (members.length === 0) return;

        const active = activeMemberIndex(members);
        members.forEach((member, i) => {
            member.setAttribute("tabindex", i === active ? "0" : "-1");
            if (!member.dataset.grmobKeyNav) {
                member.dataset.grmobKeyNav = "true";
                member.addEventListener("keydown", handleCompositeKey);
            }
        });
    }

    // Whether this container's arrows run down the page.
    //
    // Read off aria-orientation rather than re-derived from the element's
    // flex-direction, and that is the whole point of the attribute existing.
    // The behaviour and the announcement used to be two derivations of one
    // fact — this function read the axis, and nothing wrote it down — so a
    // vertical tab strip took the Up/Down arrows while telling a reader in
    // browse mode that it was horizontal. applyAccessibility now writes the
    // orientation from the same Style this would have read (ariaOrientation,
    // which is htmlout's AriaOrientationFor), and this reads it back: the two
    // statements are the same string, and a drift between them is no longer
    // something that can be written.
    //
    // The fallback is for a container that has not been through
    // applyAccessibility with a role yet — nothing on the live paths, since
    // the style pass precedes both the create walk and the patch batch, but a
    // missing attribute must not silently mean "horizontal" for a listbox.
    function compositeIsVertical(container) {
        const stated = container.getAttribute("aria-orientation");
        if (stated) return stated === "vertical";
        return ARIA_ORIENTATIONS[container.getAttribute("role")] === "vertical";
    }

    // Moves the tab stop to one member and puts focus on it — and, when the
    // widget asked for it, chooses that member.
    //
    // Focus and the stop move together, always: they are two statements of one
    // fact, and a browser that focused a member holding tabindex="-1" would
    // put the next Tab back at the top of the document.
    //
    // # Selection follows focus
    //
    // The single funnel for every movement in this section — the arrows, Home
    // and End, and a typeahead match — which is why the selection hook is here
    // rather than in each of them. It is deliberately NOT reachable from
    // syncComposite: that runs on every patch, and a selection fired from there
    // would call back into Go, produce a patch, and fire again.
    //
    // The container is a parameter for this alone. Everything else here is
    // derived from the member list.
    function moveCompositeFocus(container, members, index) {
        members.forEach((member, i) => {
            member.setAttribute("tabindex", i === index ? "0" : "-1");
        });
        members[index].focus();
        if (container.getAttribute("data-grmob-selection-follows-focus") === "true") {
            selectCompositeMember(members[index]);
        }
    }

    // Chooses a member, by invoking the author's own OnTap.
    //
    // The runtime does not write aria-selected and could not: that attribute is
    // rendered from Go state, so the only way a keystroke reaches the selection
    // is the same way a tap does. What arrives is an ordinary render pass, and
    // the selected member is announced because Go said so — which is why this
    // is three lines rather than a second selection model.
    //
    // # No NATIVELY_ACTIVATED guard, and that is the difference from activation
    //
    // activateCompositeMember returns early for a <button> or an <a>, because
    // Enter and Space already make the browser fire a real click on those and
    // synthesizing one would run the author's handler twice. An arrow key fires
    // nothing on anything, so the guard would be wrong here in exactly the case
    // that matters most: a tab strip is built out of <button> elements, and
    // ARIA recommends selection-follows-focus for tabs above all.
    //
    // A member with no handler is focused and nothing else, the same rule
    // activation follows.
    function selectCompositeMember(member) {
        const cbId = member.dataset.listener_onClick;
        if (!cbId) return;
        window.GoInvokeCallback(cbId, {});
    }

    // Enter and Space on a member that is not a control the browser activates
    // for itself.
    //
    // The author's onClick is invoked directly rather than through a
    // synthesized click, which is the same move buildTabBar makes for a tab
    // button: the callback ID is read back off the dataset at fire time, so a
    // handler replaced by a later render pass is the one that runs.
    //
    // A member with no onClick is left alone entirely — including its
    // preventDefault, so Space still scrolls a page whose options do nothing.
    function activateCompositeMember(member, e) {
        if (NATIVELY_ACTIVATED.has(member.tagName)) return;
        const cbId = member.dataset.listener_onClick;
        if (!cbId) return;
        // Space would scroll the page and Enter would submit a surrounding
        // form; a member that is about to act on the key owns it.
        e.preventDefault();
        window.GoInvokeCallback(cbId, {});
    }

    // A member's name, lowercased, for matching a typed string against.
    //
    // aria-label first, because a member that names itself has said what it is
    // called and the text inside it may be an icon or a count. Otherwise the
    // member's own text, which is what a screen reader would compute.
    //
    // This is an approximation of the accessible name and is deliberately not
    // the whole calculation: aria-labelledby is not in this framework's
    // vocabulary, and the parts of the algorithm that involve CSS content and
    // title attributes describe documents this runtime does not build.
    function compositeMemberName(member) {
        const label = member.getAttribute("aria-label");
        const text = label || memberText(member).join(" ");
        return text.trim().toLowerCase();
    }

    // The text under an element, collected leaf by leaf.
    //
    // Not `el.textContent`, which would be the obvious call and is wrong twice.
    // In a browser it concatenates every descendant's text, so reading it at
    // each level of a walk would count the same words once per ancestor; in
    // wasm/verify's DOM shim it is a stored string set on leaves only, so
    // reading it on a container answers "" (dom.mjs says so, and guards
    // against the case where the two would diverge). Descending to the leaves
    // is the one reading that means the same thing in both.
    //
    // An aria-hidden subtree is skipped: it is pruned from the accessibility
    // tree, so it is not part of any name a reader would announce, and typing
    // toward it would jump to a member for a reason the user cannot perceive.
    function memberText(el, out = []) {
        if (el.getAttribute && el.getAttribute("aria-hidden") === "true") return out;
        if (el.children.length === 0) {
            if (el.textContent) out.push(el.textContent);
            return out;
        }
        for (const child of el.children) memberText(child, out);
        return out;
    }

    // Type-to-jump. Returns whether the key was consumed.
    //
    // # The two search modes, which are one rule
    //
    // ARIA's own: a buffer of one repeated character ("aaa") searches for that
    // character and *cycles*, while a growing string ("se") searches for the
    // string and stays put if the current member still matches. Both fall out
    // of one line — the query is the first character when every character is
    // the same, the whole buffer otherwise — and the start of the search
    // follows from it: a one-character query begins after the active member so
    // repeated presses walk the matches, and a longer one begins at it so
    // refining a search does not skip the item it was already on.
    //
    // # What is not typeahead
    //
    // Space is activation and is handled before this is reached, so a listbox
    // whose options begin with a space is not a shape anyone can type toward.
    // A modified key belongs to the browser: ctrl-f is find, cmd-l is the
    // address bar, and a widget that swallowed either would be worse than one
    // with no typeahead at all.
    //
    // A key that matches nothing is *not* consumed, so it reaches the page —
    // the same rule the arrows of the other axis follow. The buffer still
    // takes it, which is what lets a mistyped character be corrected by
    // finishing the word rather than by waiting out the timeout.
    function compositeTypeahead(container, members, at, e) {
        if (!COMPOSITE_TYPEAHEAD.has(container.getAttribute("role"))) return false;
        if (e.key.length !== 1 || e.ctrlKey || e.metaKey || e.altKey) return false;

        const now = Date.now();
        if (typeahead.container !== container || now - typeahead.at > TYPEAHEAD_RESET_MS) {
            typeahead.text = "";
        }
        typeahead.container = container;
        typeahead.at = now;
        typeahead.text += e.key.toLowerCase();

        const repeated = /^(.)\1*$/.test(typeahead.text);
        const query = repeated ? typeahead.text[0] : typeahead.text;
        const from = repeated ? at + 1 : at;
        for (let i = 0; i < members.length; i++) {
            const idx = (from + i) % members.length;
            if (compositeMemberName(members[idx]).startsWith(query)) {
                moveCompositeFocus(container, members, idx);
                return true;
            }
        }
        return false;
    }

    // One member's keydown.
    //
    // currentTarget rather than target: in a browser the key arrives at
    // whatever inside the row actually holds focus, and the member is the
    // element this listener is on.
    //
    // Movement wraps at both ends. ARIA makes wrapping optional for a listbox
    // and recommends it for a tablist; one rule for both is what keeps a user
    // who has learned the tab strip from finding the article list behaves
    // differently, and there is nothing at either end of these widgets that a
    // stop would protect.
    function handleCompositeKey(e) {
        const member = e.currentTarget;
        const container = compositeOf(member);
        if (!container) return;
        const members = compositeMembersOf(container);
        const at = members.indexOf(member);
        if (at < 0) return;

        const vertical = compositeIsVertical(container);
        let to = -1;
        switch (e.key) {
            case vertical ? "ArrowDown" : "ArrowRight":
                to = (at + 1) % members.length;
                break;
            case vertical ? "ArrowUp" : "ArrowLeft":
                to = (at - 1 + members.length) % members.length;
                break;
            case "Home":
                to = 0;
                break;
            case "End":
                to = members.length - 1;
                break;
            case "Enter":
            case " ":
                activateCompositeMember(member, e);
                return;
            default:
                // A printable key is a search in the widgets that have one.
                // Everything else belongs to the page: a listbox that
                // swallowed Tab would trap a keyboard user inside it.
                if (compositeTypeahead(container, members, at, e)) {
                    e.preventDefault();
                }
                return;
        }
        // Before the move, not after: the arrow keys scroll a page and Home
        // and End jump it to the ends, and a widget that moved its own focus
        // while the document scrolled underneath is the same widget twice.
        e.preventDefault();
        moveCompositeFocus(container, members, to);
    }

    // Syncs every composite in a subtree, skipping anything already visited in
    // this pass.
    //
    // The `seen` set is what bounds the cost: syncTouchedComposites walks down
    // from every touched element, and a batch that touched a container and
    // four of its children would otherwise walk the container's subtree five
    // times.
    function syncCompositesIn(el, done, seen) {
        if (!el || !el.getAttribute || seen.has(el)) return;
        seen.add(el);
        if (isComposite(el) && !done.has(el)) {
            done.add(el);
            syncComposite(el);
        }
        for (const child of el.children) syncCompositesIn(child, done, seen);
    }

    // The patch pass. Up from each touched element, because a member changing
    // its aria-selected moves the widget's tab stop and the member cannot see
    // its own container; and down, because an added or replaced subtree may
    // carry a whole composite the up-walk would never reach.
    function syncTouchedComposites(touched) {
        const done = new Set();
        const seen = new Set();
        for (const start of touched) {
            for (let el = start; el && el.getAttribute; el = el.parentNode) {
                if (isComposite(el) && !done.has(el)) {
                    done.add(el);
                    syncComposite(el);
                }
            }
            syncCompositesIn(start, done, seen);
        }
    }

    // The size of a Spacer, on both axes.
    //
    // core.Spacer(n) is n x n on both natives — Compose
    // Spacer(Modifier.size(n.dp)), SwiftUI Color.clear.frame(width:height:) —
    // and this used to set the height alone, so a Spacer between two items of
    // a Row held them 0px apart in the browser and n points apart on device.
    //
    // flex-shrink:0 is the other half. Every container this runtime draws is a
    // flex container (see stackAxisFor), a flex item's default is to
    // shrink under pressure, and a gap whose whole job is to hold a fixed
    // distance must not be the thing that gives way. The natives have fixed
    // frames and no equivalent to shrink, so this reproduces their behavior
    // rather than adding to it.
    //
    // A missing or zero size clears all three, which is what makes this safe
    // to call from the update path: a Spacer whose size prop goes away falls
    // back to nothing rather than keeping the last size it was handed.
    //
    // # The size is remembered on the element
    //
    // Because the two channels that own these three declarations arrive
    // separately. The size is a *prop*; the three properties are also Style
    // fields (Width, Height, FlexShrink), and styleFromGrMob is total — it
    // assigns all three on every pass — so an update-style patch on a Spacer
    // would clear the gap and nothing would put it back. That is the failure
    // the Modal chassis comment warns about ("a chassis set only at creation
    // would be wiped by the first update-style patch"), and a Spacer had it.
    //
    // So the size is recorded here and re-applied by applyStyle, which is the
    // one function both channels go through.
    function applySpacerSize(el, size) {
        const n = Number(size);
        el.dataset.spacerSize = n > 0 ? String(n) : "";
        applySpacerChassis(el);
    }

    // The Spacer's chassis, written where the author's own Style has not
    // spoken.
    //
    // Same rule as the Modal chassis in styleFromGrMob and as htmlout's
    // spacerChassis: the fixed look of a node *type* goes underneath the
    // author's style rather than over it. It used to go over it — renderNode
    // called applySpacerSize after createElement and the three assignments
    // were unconditional — so a hand-assembled Spacer carrying a Width lost it
    // to a prop, which is the one place in this runtime where a type default
    // outranked an author.
    //
    // It cannot live in styleFromGrMob with Modal's, for one reason: that
    // function never sees a prop, and the size is one. So applyStyle runs it
    // immediately after the style assignment instead — the same position in
    // the sequence — and records there which of the three the author claimed.
    //
    // Which of the three, and not "is the property currently empty". The two
    // channels arrive on different patches: a size change is an update-props
    // patch with no Style in it, and reading the live property then would find
    // the chassis's *own* last write and mistake it for an author's. That is
    // not hypothetical — it is the first thing this fix got wrong, and the
    // resize test caught it.
    //
    // The assignment is unconditional for every property the author has not
    // claimed, which is what keeps it total in the same sense styleFromGrMob
    // is: a Spacer whose size drops to zero clears the gap rather than keeping
    // the last one it was handed.
    //
    // core.Spacer(n) carries no Style at all, so on every tree core builds
    // this is the whole of a Spacer's look; a hand-assembled node that does
    // carry one is the case the authored list is for.
    function applySpacerChassis(el) {
        const n = Number(el.dataset.spacerSize);
        const px = n > 0 ? `${n}px` : "";
        const authored = (el.dataset.spacerAuthored || "").split(" ");
        if (!authored.includes("width")) el.style.width = px;
        if (!authored.includes("height")) el.style.height = px;
        if (!authored.includes("flexShrink")) el.style.flexShrink = px ? "0" : "";
    }

    // Applies a Go Style to a live element. Split from styleFromGrMob because
    // one Go field does not map to CSS at all: Disabled is an element
    // *property* on form controls (which is what makes the browser refuse to
    // dispatch the events whose callback IDs are wired above), and an ARIA
    // state plus pointer-events elsewhere. Both the initial render and the
    // update-style patch go through here so a control that becomes disabled
    // mid-session actually stops responding.
    function applyStyle(el, style, nodeType) {
        const css = styleFromGrMob(style, nodeType);
        Object.assign(el.style, css);
        // The Spacer's three declarations, restored under whatever the style
        // pass just wrote. Here rather than in styleFromGrMob because the size
        // is a prop and that function sees only a Style — see
        // applySpacerChassis, which is also where the shape of this list is
        // argued. styleFromGrMob is total, so an empty string in css is
        // exactly "the author said nothing about this one".
        if (nodeType === "Spacer") {
            el.dataset.spacerAuthored = ["width", "height", "flexShrink"]
                .filter((k) => css[k])
                .join(" ");
            applySpacerChassis(el);
        }
        // The display this element would have with nothing hiding it, kept
        // because hiding a tab page overwrites el.style.display and putting the
        // page back means restoring what the style pass computed — not clearing
        // the declaration, which would drop a Column page into block flow. This
        // function is total, so the record is refreshed on every style patch
        // and can never go stale.
        //
        // `?? ""` for the one node type styleFromGrMob abstains from assigning
        // a display to at all: a Modal's is its open/closed state and belongs
        // to the prop channel. Writing the bare undefined would land the
        // string "undefined" in the attribute, which is not a display and is
        // worse than the empty string a modal has always recorded here.
        el.dataset.baseDisplay = css.display ?? "";
        // core.Style.StackAlign, parked on the element rather than turned into
        // a declaration here. It is a *layer* property, and only the overlay
        // above knows whether this node is a layer — so syncOverlay reads it
        // back off the dataset and writes the grid placement. Setting
        // justify-self/align-self from here would place a flex item too, which
        // is the leak core.StackAlign's doc says the prop must not have.
        //
        // Total like the rest of this function: the key is removed when the
        // field is unset, so a layer that drops its placement is stamped back
        // to the centre on the next overlay pass rather than keeping the last
        // value it had. Kept in the same dataset channel data-tab-selected
        // uses, for the same reason — a fact one element records for another
        // to act on.
        if (style.StackAlign) {
            el.dataset.stackAlign = style.StackAlign;
        } else {
            delete el.dataset.stackAlign;
        }
        applyAccessibility(el, style, nodeType);

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

    // core.Style's accessibility fields -> the ARIA attributes that mean
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
    // The role becomes the `role` attribute and that is the whole mapping:
    // core.Role's values are ARIA's own spellings, chosen so neither DOM
    // target needs a table (see core/role.go). Emitted verbatim even when the
    // tag already implies it — suppressing the redundant case would mean this
    // knowing tagForType's table, and a redundant role is inert where a
    // missing one is not. A named container that says nothing about what it is
    // gets one supplied; see ariaRole below.
    //
    // The id and aria-controls come from core.Style.AccessibilityID and
    // core.Style.AccessibilityControls, verbatim in both directions — the
    // vocabulary's only two references. Nothing checks that the target of one
    // exists: a patch is applied to one element and this runtime has no index
    // of the document. A dangling IDREF is inert.
    //
    // The id has a second writer, wireTabPanel, and the two cannot both hold
    // it. canBeTabPanel resolves that in the author's favour — a page carrying
    // an AccessibilityID is left unwired — so by the time this writes, the
    // wiring has already stood down. The *role* is the other way round: the
    // wiring runs after this on every pass that can change it (renderNode
    // calls syncTabView after building the subtree, and syncTouchedTabViews
    // re-runs it at the end of every patch batch), so a wired panel ends the
    // pass carrying "tabpanel" whatever this wrote a moment earlier.
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
    function applyAccessibility(el, style, nodeType) {
        const hidden = !!style.AccessibilityHidden;
        // The self-roling node types: a role that comes from what the node IS
        // rather than from a Style. A Modal is the one with nothing to ride on
        // at all (core.ModalNode has no Style field); a Switch is the one the
        // DOM has no element for, so the role is the only thing that says a
        // checkbox is a switch. ownRole is the table, restated from Go's
        // htmlout.ownRoles and pinned to it by wasm/verify.
        //
        // An author's own role wins over the supplied one, which is the rule
        // htmlout's selfRoleSemantics states on the other side.
        //
        // Hidden wins over all of it — an element pruned from the
        // accessibility tree has no role to describe, and a Modal's aria-modal
        // would be claiming the document behind something a reader cannot
        // reach is inert.
        const own = hidden ? "" : ownRole(nodeType);
        const dialog = nodeType === "Modal" && !hidden;
        const role = hidden ? "" : (own
            ? (style.AccessibilityRole || own)
            : ariaRole(el, style));
        setOrRemove(el, "aria-hidden", hidden ? "true" : "");
        setOrRemove(el, "aria-label", hidden ? "" : (style.AccessibilityLabel || ""));
        setOrRemove(el, "aria-description", hidden ? "" : (style.AccessibilityHint || ""));
        setOrRemove(el, "role", role);
        // An authored role voids the tab-panel wiring's claim on this slot,
        // core.RoleTabPanel included — which is now a value an author can
        // write, and which is the one case the marker alone could not tell
        // from the wiring's own. This is the only place the runtime has the
        // Style in hand, so it is the only place that decision can be made;
        // htmlout makes the same one in tabPanelBox by reading the Style
        // directly, which is what keeps the two targets agreeing about which
        // pages are panels.
        //
        // RoleGroup needs no exemption even though the wiring may replace it:
        // the claim is dropped here and re-earned a moment later, because
        // canBeTabPanel accepts `group` on its own terms.
        if (style.AccessibilityRole) {
            delete el.dataset.grmobPanel;
        }
        setOrRemove(el, "id", hidden ? "" : (style.AccessibilityID || ""));
        setOrRemove(el, "aria-controls", hidden ? "" : (style.AccessibilityControls || ""));
        setOrRemove(el, "aria-modal", dialog ? "true" : "");
        setOrRemove(el, "aria-level", hidden ? "" : ariaLevel(style));
        // Which way a composite runs. Written here rather than derived where
        // it is used, so the keyboard and the announcement are one statement —
        // compositeIsVertical reads this attribute back. See ariaOrientation.
        setOrRemove(el, "aria-orientation", hidden ? "" : ariaOrientation(style, nodeType));
        // Whether the widget chooses the member the arrows land on.
        //
        // The one thing this function writes that is NOT an ARIA attribute,
        // and the reason is that ARIA has none: aria-* says what a widget is,
        // and this says what its keyboard does. So it rides in the data
        // channel, alongside data-tab-selected and data-stack-align — a fact
        // one pass records for another to act on.
        //
        // Written here anyway, rather than read off the Style at the keystroke,
        // because a keystroke has no Style: handleCompositeKey has an element
        // and nothing else. Total like every attribute above, so a widget that
        // stops asking for it stops getting it.
        //
        // Not suppressed for a non-composite role. A Box that carries the flag
        // and no composite role has nobody reading the attribute, which is the
        // same nothing that happens today; suppressing it would mean this
        // function knowing the runtime's composite tables, and the tables would
        // then be a fact in two places.
        //
        // What reports it instead is core.AuditTree, in debug mode, walking the
        // finished tree — which is where a rule about a node's role in relation
        // to a whole vocabulary belongs, and which needed the vocabulary to
        // exist in Go: core.KeyboardComposites() is the union of
        // COMPOSITE_MEMBERS' keys and COMPOSITE_FOCUSABLE, held to those two
        // tables by wasm/verify's keynav_test.go.
        setOrRemove(el, "data-grmob-selection-follows-focus",
            hidden ? "" : (style.AccessibilitySelectionFollowsFocus ? "true" : ""));
        // Both selection attributes are written on every call, not just the
        // one this role calls for. The role can change between passes — a
        // patch can turn a tab into a button — and the totality rule has to
        // hold across the *pair*: writing only the new one would leave the
        // other standing, so a node that had been a tab would be announced as
        // a selected tab and a pressed button at once.
        const selected = hidden ? ["", ""] : ariaSelected(style, nodeType);
        setOrRemove(el, "aria-selected", selected[0]);
        setOrRemove(el, "aria-pressed", selected[1]);
        setOrRemove(el, "aria-expanded", hidden ? "" : ariaExpanded(style, nodeType));
        // All four of the value family on every call, for the reason both
        // selection attributes are written: the role can change between passes,
        // and a bar that stops being a progressbar must not keep a range.
        const value = hidden ? EMPTY_VALUE : ariaValue(style);
        setOrRemove(el, "aria-valuenow", value.Now);
        setOrRemove(el, "aria-valuemin", value.Min);
        setOrRemove(el, "aria-valuemax", value.Max);
        setOrRemove(el, "aria-valuetext", value.Text);
    }

    // The empty range, for the aria-hidden path and for every role the value
    // family is not defined on. A shared frozen object rather than a fresh
    // literal per call: applyAccessibility runs for every node of every tree
    // and all but the progress bars take this branch.
    const EMPTY_VALUE = Object.freeze({ Now: "", Min: "", Max: "", Text: "" });

    // core.Style.AccessibilityValue as the four aria-value* values. The htmlout
    // twin of this is ariaValue in export.go and the two must agree; the whole
    // argument lives there and in core.ValueRange.
    //
    // The short version: this is the fourth state mapping and the narrowest.
    // ariaLevel resolves two Go fields onto one attribute, ariaSelected one
    // onto two, ariaExpanded one onto one; this resolves one onto four, because
    // Now, Min and Max are one fact in three parts — "45" is 45% out of ARIA's
    // implicit 0..100 and step 45 out of 1..50.
    //
    // A stated role with no range is not corrected to zero. ARIA spells an
    // *indeterminate* progress bar by leaving aria-valuenow off, so a bar that
    // is running with no idea how far is exactly this role and an unstated
    // range — and defaulting a 0 in would pin every one of them at the start.
    function ariaValue(style) {
        const v = style.AccessibilityValue;
        if (!v) return EMPTY_VALUE;
        switch (style.AccessibilityRole) {
            case "progressbar":
                return {
                    Now: v.Now || "",
                    Min: v.Min || "",
                    Max: v.Max || "",
                    Text: v.Text || "",
                };
            default:
                return EMPTY_VALUE;
        }
    }

    // The value of the role attribute for one element: what the author said,
    // or — when they said nothing and the element could not otherwise carry the
    // name they gave it — "group". The htmlout twin of this is ariaRole in
    // export.go and the two must agree; the whole argument lives there and in
    // core.RoleGroup.
    //
    // The short version: a <div> and a <span> carry the implicit ARIA role
    // `generic`, ARIA prohibits an accessible name on `generic`, and browsers
    // enforce that by pruning the name out of the accessibility tree. So an
    // AccessibilityLabel on a plain container was announced by VoiceOver and
    // TalkBack and by nothing on the web. `group` is the smallest role that
    // makes it legal: nameable, not a landmark, no required children, and its
    // own children stay readable.
    //
    // The tag is read off the element rather than derived from the node type,
    // which is this runtime's usual shortcut (it has the element in hand and
    // htmlout has only the type). GENERIC_TAGS is the same set htmlout states
    // in genericTags, pinned by TestRuntimeGenericTagsMatchGo.
    //
    // A self-roling node never reaches here — the caller answers ownRole
    // first — which is why there is no equivalent of htmlout's CarriesOwnRole
    // guard. That covers a Switch as well as a Modal, and it has to: an
    // <input> is not a generic tag, so the fallback below would refuse it the
    // group role and leave the slot empty rather than wrong, but the role it
    // *does* need would never be written.
    function ariaRole(el, style) {
        const authored = style.AccessibilityRole || "";
        if (authored) return authored;
        if (!style.AccessibilityLabel) return "";
        return GENERIC_TAGS.has(el.tagName.toLowerCase()) ? "group" : "";
    }

    // core.Style.AccessibilitySelected as the pair [aria-selected,
    // aria-pressed], at most one of which is non-empty. The htmlout twin of
    // this is ariaSelected in export.go and the two must agree; the reasoning
    // for every guard here lives there and in core.Style.
    //
    // This is ariaLevel's mirror. That one resolves two Go fields onto one
    // attribute; this resolves one Go field onto two attributes, with the same
    // switch on the role deciding. ARIA has two words and they are not
    // synonyms: aria-selected is one of a set (a tab among tabs), aria-pressed
    // is a toggle answering only for itself (a filter chip).
    //
    // A pair is returned rather than a name/value because the caller has to
    // clear the other attribute either way — see the note at the call site.
    //
    // The Button node type is checked only when the style names no role: a
    // core.Button already is a button, which is what lets components.Chip —
    // which renders as one and sets no role — carry a state at all. Same rule
    // that gives a Modal its dialog role.
    function ariaSelected(style, nodeType) {
        const value = style.AccessibilitySelected || "";
        if (!value) return ["", ""];
        switch (style.AccessibilityRole) {
            case "option":
            case "tab":
            case "row":
            case "columnheader":
                return [value, ""];
            case "button":
                return ["", value];
            case "":
            case undefined:
                return nodeType === "Button" ? ["", value] : ["", ""];
            default:
                return ["", ""];
        }
    }

    // core.Style.AccessibilityExpanded as the aria-expanded value, or "" when
    // there is nothing valid to write. The htmlout twin of this is
    // ariaExpanded in export.go and the two must agree; the reasoning for
    // every guard lives there and in core.Style.
    //
    // One field onto one attribute, which makes this the simplest of the three
    // state mappings — ariaLevel resolves two fields onto one attribute and
    // ariaSelected one field onto two. All the work is in the role list, and
    // the point of that list is that it is *not* ariaSelected's: aria-expanded
    // drops option and adds link and listbox. A shared guard would be wrong at
    // four roles.
    //
    // Only one attribute is written, so this returns a string rather than the
    // pair ariaSelected returns — there is no sibling attribute the totality
    // rule has to clear alongside it. The caller still writes on every call,
    // including the empty value, because an update-style patch carries the
    // whole new Style and a guarded write would leave a stale state standing.
    //
    // The Button node type is checked only when the style names no role, the
    // same rule that gives a Modal its dialog role. ARIA's disclosure pattern
    // is a button, so this is the case the attribute exists for rather than a
    // shortcut.
    function ariaExpanded(style, nodeType) {
        const value = style.AccessibilityExpanded || "";
        if (!value) return "";
        switch (style.AccessibilityRole) {
            case "button":
            case "link":
            case "listbox":
            case "row":
            case "columnheader":
            case "tab":
                return value;
            case "":
            case undefined:
                return nodeType === "Button" ? value : "";
            default:
                return "";
        }
    }

    // Which way a composite runs, per role, when its own axis cannot say.
    // ARIA's own defaults: a tablist and a toolbar are horizontal, a listbox is
    // vertical. The Go authority is ariaOrientations in htmlout/orientation.go,
    // where the argument for the whole attribute lives;
    // TestRuntimeOrientationTableMatchesGo holds the two together, and reads
    // this literal out of the source textually, so keep it a flat object of
    // string values.
    //
    // It is also the table compositeIsVertical falls back to, which is why
    // there is one table here and not two: the keyboard's default and the
    // announcement's default were always the same fact.
    const ARIA_ORIENTATIONS = { listbox: "vertical", tablist: "horizontal", toolbar: "horizontal" };

    // core.Style's role and layout axis as the aria-orientation value, or ""
    // for a role the attribute is not defined on. The htmlout twin of this is
    // AriaOrientationFor in orientation.go and the two must agree; the whole
    // argument lives there.
    //
    // The short version: the runtime reads a composite's axis to pick the arrow
    // pair, and nothing announced it — so a vertical tab strip behaved one way
    // and, through ARIA's horizontal default for a tablist, said the other. The
    // axis resolves exactly as styleFromGrMob resolves it for the CSS
    // declaration (an explicit FlexDirection over the node type's own stacking
    // direction), so the attribute cannot disagree with the layout; ARIA's
    // per-role default covers the one shape that has no axis, a role on a node
    // type that is not a stack.
    //
    // The author's own role, not the effective one: the two values
    // applyAccessibility can supply where the style stated none — `group` from
    // ariaRole and `dialog` for a Modal — are not oriented roles.
    function ariaOrientation(style, nodeType) {
        const def = ARIA_ORIENTATIONS[style.AccessibilityRole || ""];
        if (!def) return "";
        const axis = style.FlexDirection || stackAxisFor(nodeType);
        if (axis.startsWith("column")) return "vertical";
        if (axis.startsWith("row")) return "horizontal";
        return def;
    }

    // Whichever of core.Style's two level fields the node's role calls for, as
    // an aria-level value, or "" when there is nothing valid to write. The
    // htmlout twin of this is ariaLevel in export.go and the two must agree;
    // the reasoning for every guard here lives there and in core.Style.
    //
    // The switch is the point, not a tidier if-chain. ARIA defines aria-level
    // for exactly three roles, core carries a heading's tier and a collection
    // item's depth as separate ints (they are validated differently), and this
    // is where the two meet at the one attribute both become. Dispatching on
    // the role makes them mutually exclusive by construction: a node has one
    // role, so no arrangement of the two fields can produce two values for one
    // attribute.
    //
    // Both arms drop rather than clamp, and they drop different things. A
    // heading stops at 6 — that is as far as h1-h6 and SwiftUI's .h1-.h6 go —
    // and a nesting depth has no ARIA ceiling at all, so capping it would
    // flatten a legitimately deep tree.
    function ariaLevel(style) {
        switch (style.AccessibilityRole) {
            case "heading": {
                const level = style.AccessibilityHeadingLevel || 0;
                return level >= 1 && level <= 6 ? String(level) : "";
            }
            case "listitem":
            case "row": {
                const level = style.AccessibilityNestingLevel || 0;
                return level >= 1 ? String(level) : "";
            }
            default:
                return "";
        }
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

    // --- End reached (core.OnEndReached) -------------------------------------
    //
    // The DOM has no "you are near the bottom of this list" event, so this is
    // synthesized the way the platform makes cheapest: an IntersectionObserver
    // watching the list's *last* child, with a rootMargin that fires it a
    // screenful early so the next page is already in flight by the time the
    // reader gets there.
    //
    // # The last child, not a sentinel
    //
    // The obvious implementation appends an invisible sentinel row and watches
    // that. This runtime cannot: patches are addressed positionally
    // (data-node-path, and add-child derives the new index from
    // el.children.length), so an extra element inside a list would shift every
    // sibling index after it and misdirect every later patch. The TabView bar
    // is the one piece of chrome allowed inside a node's box, and it pays for
    // the privilege with chromeOffset. A list does not need to: its last child
    // is exactly as good an observation target and costs the tree nothing.
    //
    // The cost is that the target *moves* — every appended page makes a new
    // last child — so the observation has to be re-pointed whenever the
    // subtree changes. syncEndReached is that re-pointing, and it is called
    // from the same two places syncTabView is: after renderNode builds the
    // children, and after a patch batch lands (syncTouchedEndReached).
    //
    // # Firing more than once is expected
    //
    // The observer fires on entry, and again on any resize or scroll that
    // re-enters the margin, and again as soon as a new last child is observed
    // while still near the bottom. None of that is de-duplicated here on
    // purpose: core.OnEndReached debounces on the Go side by remembering the
    // row count at the last fire, which is the one place all four renderers'
    // different notions of "again" collapse into one answer.
    //
    // # No observer, no feature
    //
    // IntersectionObserver is guarded rather than assumed, the same way
    // getComputedStyle is in toastLayerHost: a host without it (an older
    // embedder, a minimal test DOM) renders the list correctly and simply
    // never reports the edge, which leaves a components.LoadMore button as the
    // manual fallback it was designed to be.
    const END_REACHED_MARGIN = "200px";

    // The observer and its current target hang off the element as plain
    // properties rather than dataset entries: neither is a string, and neither
    // is anything a patch, a serializer or a reader of the markup should see.
    const END_OBSERVER = "__grmobEndObserver";
    const END_TARGET = "__grmobEndTarget";

    // Records the callback ID. The observation itself waits for
    // syncEndReached, because on the create path this runs from inside
    // createElement, before the children this needs to watch exist at all.
    function attachEndReached(el, cbId) {
        el.dataset.listener_onEndReached = cbId;
    }

    // Points the element's observer at whatever its last child is now,
    // creating the observer on first need and tearing it down when the list
    // stops carrying the prop (pruneStaleListeners drops the dataset entry;
    // this is what notices).
    function syncEndReached(el) {
        const cbId = el.dataset.listener_onEndReached;
        const last = el.children.length ? el.children[el.children.length - 1] : null;
        if (!cbId || !last || typeof IntersectionObserver !== "function") {
            if (el[END_OBSERVER]) {
                el[END_OBSERVER].disconnect();
                el[END_OBSERVER] = null;
                el[END_TARGET] = null;
            }
            return;
        }
        // Already watching this exact element: re-observing would be harmless
        // but would also re-fire the entry callback for a target that never
        // left the viewport, turning every unrelated patch into an
        // end-reached report.
        if (el[END_TARGET] === last) return;
        if (!el[END_OBSERVER]) {
            el[END_OBSERVER] = new IntersectionObserver((entries) => {
                if (!entries.some((entry) => entry.isIntersecting)) return;
                // Re-read at fire time, as every listener here does: the ID is
                // positional and a pass landing mid-scroll may have refreshed
                // or pruned it.
                const latestCbId = el.dataset.listener_onEndReached;
                // {} rather than a value: Go registered this through the void
                // callback channel, and an envelope carrying a value would be
                // routed to the text or int map, where the ID does not exist.
                if (latestCbId) window.GoInvokeCallback(latestCbId, {});
            }, { rootMargin: END_REACHED_MARGIN });
        }
        el[END_OBSERVER].disconnect();
        el[END_TARGET] = last;
        el[END_OBSERVER].observe(last);
    }

    // The post-batch pass, sibling of syncTouchedTabViews and walking the same
    // ancestor chains for the same reason: a patch addressed to a *row* (or to
    // a cell inside one) is still a change to the list's last child, and the
    // list is the element that has to re-point its observer.
    //
    // The observer property is tested alongside the dataset entry so a list
    // that just *lost* its onEndReached — pruneStaleListeners having already
    // deleted the entry — is still visited, and torn down.
    function syncTouchedEndReached(touched) {
        const done = new Set();
        for (const start of touched) {
            for (let el = start; el; el = el.parentNode) {
                if (!el.dataset || done.has(el)) continue;
                if (el.dataset.listener_onEndReached || el[END_OBSERVER]) {
                    done.add(el);
                    syncEndReached(el);
                }
            }
        }
    }


    // --- Live maps (core.MapView) --------------------------------------------
    //
    // The web half of core.MapView: a <div> handed to Leaflet, with the pins
    // taken from the node's Marker children.
    //
    // # Leaflet is the host page's, not this runtime's
    //
    // Nothing here loads a library. The page that hosts the app adds Leaflet's
    // script and stylesheet (see wasm/index.html), and this code uses
    // window.L if it is there and draws a placeholder box if it is not.
    //
    // That is a deliberate split rather than a missing feature. A runtime that
    // injected a script tag would be fetching third-party code on behalf of
    // every app that uses it, including the ones with no map on any screen and
    // the ones whose content policy forbids it — and it would do so at a moment
    // (mid-patch) when there is nothing sensible to do about a failed load. The
    // host page is where a dependency belongs, and the placeholder is what makes
    // its absence legible rather than fatal.
    //
    // The placeholder adds no child element, which is a constraint rather than
    // a preference: a MapView's children are Marker nodes addressed positionally
    // by the patch stream, so chrome inside one would have to be counted by
    // chromeOffset and would shift every marker that arrived later. A styled box
    // with the region still on it in data attributes is exactly what htmlout
    // exports for the same node, so the two web targets degrade identically.
    //
    // # The echo guard, which is the whole usability of the node
    //
    // Go's region is applied only when it *changes*. The record below keeps the
    // last region this code handed to Leaflet and compares; a patch that
    // re-states the same region touches nothing.
    //
    // Without it, a map is unusable. The user drags, Go re-renders for any
    // unrelated reason, and setView puts the map back where Go last said —
    // under the finger. With it, Go moving the map is an instruction and Go
    // merely re-rendering is not. core.MapView's "The map is controlled, with
    // the echo guard a drag needs" is the statement of this contract that all
    // three live hosts implement.
    //
    // The record closes the loop's other half. setView makes Leaflet fire
    // moveend, and reporting that back to Go as a user gesture would have an app
    // echoing its own instruction into its own state on every programmatic move.
    // What suppresses it is the *same* comparison rather than a second
    // mechanism: a gesture is reported only when it ends somewhere other than
    // the region this code last applied.
    //
    // # Two memories, because they are two facts
    //
    // `applied` is the region GO last asked for. `settled` is where the MAP
    // last came to rest. They are equal for as long as nobody touches the map
    // and they diverge the moment somebody does, and each direction reads the
    // one that answers its own question:
    //
    //	apply   want !== applied   Go changed its mind — an instruction
    //	        want === settled   the map is already there — Go echoing the pan
    //	                           back into its own state, which must not turn
    //	                           into a setView landing a frame late
    //	report  next !== applied   not the moveend our own setView caused
    //	        next !== settled   not a second event for a map that has not
    //	                           moved since the last one
    //
    // These were one slot, and the browser is where that was caught. A pan
    // wrote the user's region into `applied`, so Go's UNCHANGED region then
    // read as a change, and the next patch to reach this map — a dropped pin,
    // a marker moving, any unrelated re-render — snapped the map back to where
    // Go last said. That is the exact failure this guard is named for, and the
    // fake never saw it: its pan test moves the map's centre without firing
    // moveend, so the report path never ran and never corrupted the slot. Real
    // Leaflet always fires moveend, which is what made it visible in a browser
    // and invisible in a unit test.
    //
    // That was a boolean window at first — set around the setView call — and a
    // test is what said it was wrong. The window only closes the case where
    // Leaflet fires moveend synchronously from inside setView, which is what it
    // does with animation off and is not a promise anyone made; an animated or
    // deferred moveend would land after the flag was cleared and be reported as
    // a user pan to a place the user never went. The comparison has no timing in
    // it at all, and it is also simply the truth: a view that matches what Go
    // asked for is not news for Go.

    const MAP_RECORD = "__grmobMap";
    const MARKER_RECORD = "__grmobMarker";

    // OpenStreetMap's own tile servers, which have a usage policy: identify
    // your app, do not bulk download, expect to be blocked above a modest
    // volume. Right for a preview, for the tutorial and for a small app;
    // wrong for anything with a real user base, which should point this at a
    // provider it pays for.
    //
    // A constant here rather than a prop on the node, because the tile source
    // is a deployment fact and not a property of a view — the same reasoning
    // that keeps the osmdroid and MapKit tile decisions inside their hosts.
    const MAP_TILE_URL = "https://tile.openstreetmap.org/{z}/{x}/{y}.png";
    const MAP_TILE_ATTRIBUTION =
        '&copy; <a href="https://www.openstreetmap.org/copyright">OpenStreetMap</a> contributors';

    // How long a gesture has to be quiet before its region is reported, in ms.
    //
    // Leaflet's moveend and zoomend already fire once per gesture rather than
    // per frame, so this is not the throttle a raw stream would need — it is
    // the coalescer for the gestures that fire *both*. A pinch-zoom ends as a
    // zoomend and a moveend a few milliseconds apart, and two full Go render
    // passes for one gesture is the cost this removes.
    const MAP_REGION_QUIET_MS = 120;

    let leafletWarned = false;

    // The region a map element is currently asking for, read back off the
    // dataset where applyMapProps wrote it.
    //
    // The dataset is the single source of truth for both web targets — htmlout
    // writes the same three attributes into its static export — so this
    // function is also what a loader upgrading an exported document would
    // write. Numbers come back as strings from a dataset, hence the Number().
    function mapRegion(el) {
        return {
            lat: Number(el.dataset.lat),
            lng: Number(el.dataset.lng),
            zoom: Number(el.dataset.zoom),
        };
    }

    // Exact, and this is the one host where that is safe on both paths.
    //
    // Leaflet caches the centre it was given: getCenter() returns _lastCenter
    // unchanged for as long as the map has not moved, so setView's own numbers
    // come straight back out and the report the setView causes compares equal.
    // The natives cannot do that — osmdroid quantises the centre to integer
    // pixels and MapKit re-derives the whole region — so both of them compare
    // to within half a pixel instead. GrMobMapView.kt's samePlaceOnScreen
    // carries the argument; the reason it is not restated here is that there is
    // nothing to tolerate.
    function sameRegion(a, b) {
        return !!a && !!b && a.lat === b.lat && a.lng === b.lng && a.zoom === b.zoom;
    }

    // A map with no engine behind it: the region's box, drawn so the screen has
    // the shape the layout gave it rather than a collapsed nothing.
    //
    // No children and no text, for the addressing reason in the section comment.
    // The console line is the only place the reason can be said, and it is said
    // once per page rather than per map per patch.
    function drawMapPlaceholder(el) {
        el.dataset.grmobMapPlaceholder = "1";
        // A neutral fill, not a themed one: this code has no theme. The Go
        // style still lands on the element (applyStyle ran in createElement),
        // so a caller who gave the map a background gets theirs — this is only
        // the floor under a map with no style at all.
        if (!el.style.background) el.style.background = "#E5E3DF";
        if (!leafletWarned) {
            leafletWarned = true;
            console.warn(
                "grmob: core.MapView needs Leaflet on the host page. Add " +
                "leaflet.css and leaflet.js to your index.html (see wasm/index.html) " +
                "or use components.StaticMap, which needs no engine."
            );
        }
    }

    // Creates the Leaflet map for an element, once.
    //
    // Deferred to a frame when the element is not yet in the document, which is
    // the initial-render case: renderNode builds the whole tree detached and
    // mount appends it when it is assembled, and Leaflet measures the container
    // when the map is created — a map built detached comes out 0x0 and stays
    // that way. The same deferral is harmless on the patch path, where the
    // element is already live.
    function createLeafletMap(L, el) {
        const region = mapRegion(el);
        const map = L.map(el, {
            center: [region.lat, region.lng],
            zoom: region.zoom,
            // Leaflet's default attribution control is a link, and the tile
            // licence requires the credit — so it stays. zoomControl stays too:
            // a map with no buttons is unusable with a mouse, and a phone user
            // pinches past it.
            attributionControl: true,
        });
        L.tileLayer(MAP_TILE_URL, {
            attribution: MAP_TILE_ATTRIBUTION,
            maxZoom: 19,
        }).addTo(map);

        const record = {
            map,
            // The region GO last asked for, which is also the region this code
            // last handed to Leaflet. Written only on the apply path. Seeded
            // with what the map was created at, so the first sync after
            // creation applies nothing.
            applied: region,
            // Where the MAP last came to rest, and so also the last thing told
            // to Go. Written only on the report path. Seeded with the creation
            // region because that is where the map is resting — which is what
            // makes a gesture that ends where it started report nothing.
            settled: region,
            markers: [],
            userLayer: null,
            userAccuracy: null,
            locating: false,
            quiet: null,
        };
        el[MAP_RECORD] = record;

        const report = () => {
            if (record.quiet) clearTimeout(record.quiet);
            record.quiet = setTimeout(() => {
                record.quiet = null;
                const c = map.getCenter();
                const next = { lat: c.lat, lng: c.lng, zoom: map.getZoom() };
                // The echo guard, read in this direction: the map is sitting
                // where this code put it, so there is nothing to tell Go. That
                // covers the moveend setView itself causes, and it covers a
                // gesture that happens to end where it started.
                //
                // `settled` is the second half, and it is why the user's
                // region does NOT go into `applied`: a map that fires two
                // events without moving between them has one thing to say, and
                // Go's own region is a separate fact that a pan must not
                // overwrite. See "Two memories, because they are two facts".
                if (sameRegion(record.applied, next)) return;
                if (sameRegion(record.settled, next)) return;
                // Re-read the ID at fire time, as every listener in this
                // runtime does: IDs are positional and a pass landing mid-
                // gesture may have refreshed or pruned this one.
                const cbId = el.dataset.listener_onRegionChange;
                // Recorded even with no handler attached, so a map that gains
                // one later does not immediately report a pan that happened
                // before anybody was listening.
                record.settled = next;
                if (!cbId) return;
                // The same wire form core.FormatRegion writes and
                // core.ParseRegion reads: "lat,lng,zoom". Go registered this
                // through the text channel, so the envelope carries a string.
                window.GoInvokeCallback(cbId, { value: `${next.lat},${next.lng},${next.zoom}` });
            }, MAP_REGION_QUIET_MS);
        };
        map.on("moveend", report);
        map.on("zoomend", report);

        map.on("click", (e) => {
            const cbId = el.dataset.listener_onMapTap;
            if (!cbId || !e || !e.latlng) return;
            window.GoInvokeCallback(cbId, { value: `${e.latlng.lat},${e.latlng.lng}` });
        });

        return record;
    }

    // Applies Go's region when it is Go's instruction rather than Go's echo.
    // See the section comment.
    function applyMapRegion(record, el) {
        const want = mapRegion(el);
        if (!Number.isFinite(want.lat) || !Number.isFinite(want.lng) || !Number.isFinite(want.zoom)) {
            return;
        }
        // Go has not changed its mind. Nothing to do — and emphatically not a
        // reason to re-centre: the map may be somewhere else entirely because
        // the user put it there, and this is the patch that would yank it back.
        if (sameRegion(record.applied, want)) return;
        record.applied = want;
        // Go HAS changed its mind, and has changed it to where the map already
        // is: the app echoed OnRegionChange into its own state, and this is
        // that value arriving a frame later. Recorded above, because Go is now
        // asking for this region and the next instruction has to be measured
        // against it — but not applied, because applying it is the round trip
        // an echoing app would otherwise fight, landing a setView on a map the
        // user may already have started moving again.
        if (sameRegion(record.settled, want)) return;
        // No animation. Not for the echo — the comparison above and in report
        // handles that whenever the moveend arrives — but because an animated
        // setView on a map the app is driving from its own state (following a
        // location, stepping through a list of places) queues animations behind
        // each other and lags the data it is showing.
        record.map.setView([want.lat, want.lng], want.zoom, { animate: false });
        // The map now rests here, so `settled` says so. Without this line the
        // slot holds wherever the user last left it, and a later instruction
        // back to that place would be skipped as "already there" while the map
        // sat somewhere else entirely — the invariant is that `settled` is
        // where the map is, however it got there.
        record.settled = want;
    }

    // Reconciles the Leaflet marker layer against the MapView's Marker child
    // elements.
    //
    // The Leaflet marker lives on the child element that describes it
    // (child[MARKER_RECORD]), which is what makes this a reconciliation rather
    // than a rebuild: a child the patch stream moved is the same element, so its
    // marker is moved; a child it removed is gone from el.children, so its
    // marker is removed. Nothing is recreated for a sibling's sake, which is the
    // whole reason core.Marker is a child node and not an entry in a prop array.
    function syncMapMarkers(L, record, el) {
        const live = [];
        for (const child of el.children) {
            if (!child.dataset || child.dataset.nodeType !== "Marker") continue;
            const lat = Number(child.dataset.lat);
            const lng = Number(child.dataset.lng);
            if (!Number.isFinite(lat) || !Number.isFinite(lng)) continue;

            let marker = child[MARKER_RECORD];
            if (!marker) {
                marker = L.marker([lat, lng]).addTo(record.map);
                child[MARKER_RECORD] = marker;
                marker.on("click", () => {
                    const cbId = el.dataset.listener_onMarkerTap;
                    if (!cbId) return;
                    // The id off the element at fire time, not the one captured
                    // when the marker was made: an update-props patch can
                    // rewrite it, and a marker reporting the id it was born with
                    // would open the wrong row.
                    window.GoInvokeCallback(cbId, { value: child.dataset.markerId || "" });
                });
            } else {
                const at = marker.getLatLng();
                // Guarded, unlike most writes in this runtime: setLatLng on an
                // open popup closes and reopens it, so a marker that has not
                // moved must not be told where it is.
                if (at.lat !== lat || at.lng !== lng) marker.setLatLng([lat, lng]);
            }

            const title = child.dataset.title || "";
            if (marker.__grmobTitle !== title) {
                marker.__grmobTitle = title;
                if (title) {
                    marker.bindPopup(title);
                } else {
                    marker.unbindPopup();
                }
            }
            live.push(marker);
        }

        // Whatever is no longer among the children. The markers array is this
        // code's own list rather than Leaflet's layer set, because the map also
        // holds the tile layer and the user-location circles and removing those
        // would blank the map.
        for (const marker of record.markers) {
            if (!live.includes(marker)) record.map.removeLayer(marker);
        }
        record.markers = live;
    }

    // core.ShowUserLocation, which Leaflet has no control for: the blue dot on
    // the two natives is the platform map's own feature, and here it is
    // Map.locate plus two circles.
    //
    // Watched rather than one-shot, because the dot is supposed to follow the
    // user — and stopped when the prop goes away, because a watch left running
    // is a GPS left running, which is the one cost in this file a user can
    // measure.
    //
    // The accuracy circle is not decoration. A fix with a 2km radius drawn as a
    // point is a confident lie, and on a desktop browser (where the position
    // comes from the network) that is the normal case.
    function syncMapUser(L, record, el) {
        const want = el.dataset.showUser === "true";
        if (want === record.locating) return;
        record.locating = want;
        if (!want) {
            record.map.stopLocate();
            if (record.userLayer) record.map.removeLayer(record.userLayer);
            if (record.userAccuracy) record.map.removeLayer(record.userAccuracy);
            record.userLayer = null;
            record.userAccuracy = null;
            return;
        }
        record.map.on("locationfound", (e) => {
            if (!record.locating || !e || !e.latlng) return;
            if (!record.userLayer) {
                record.userLayer = L.circleMarker(e.latlng, {
                    radius: 6, color: "#FFFFFF", weight: 2,
                    fillColor: "#1A73E8", fillOpacity: 1,
                }).addTo(record.map);
                record.userAccuracy = L.circle(e.latlng, {
                    radius: e.accuracy || 0, stroke: false,
                    fillColor: "#1A73E8", fillOpacity: 0.12,
                }).addTo(record.map);
            } else {
                record.userLayer.setLatLng(e.latlng);
                record.userAccuracy.setLatLng(e.latlng);
                record.userAccuracy.setRadius(e.accuracy || 0);
            }
        });
        // setView false: the dot is information, not a command to go there.
        // An app that wants to follow the user renders a Region from
        // hooks.UseLocation, which is a decision it makes rather than one this
        // code makes for it.
        record.map.locate({ watch: true, setView: false, enableHighAccuracy: true });
    }

    // The one entry point: make the element's Leaflet state match the node.
    // Called from renderNode once the Marker children exist, and from the
    // post-batch pass for every map a patch reached.
    function syncMap(el) {
        const L = typeof window !== "undefined" ? window.L : undefined;
        if (!L || typeof L.map !== "function") {
            drawMapPlaceholder(el);
            return;
        }
        let record = el[MAP_RECORD];
        if (!record) {
            // Leaflet measures the container at creation time, so a map built
            // while the tree is still detached comes out 0x0 and stays that
            // way. Wait a frame and try again — by then mount has appended the
            // tree, exactly as the focus command's deferral relies on.
            if (el.isConnected === false) {
                requestAnimationFrame(() => syncMap(el));
                return;
            }
            record = createLeafletMap(L, el);
        }
        applyMapRegion(record, el);
        syncMapMarkers(L, record, el);
        syncMapUser(L, record, el);
    }

    // The post-batch pass, sibling of syncTouchedEndReached and walking the same
    // ancestor chains for the same reason: a patch addressed to a *marker* is a
    // change to the map that holds it, and the map is the element with the
    // Leaflet state to reconcile.
    function syncTouchedMaps(touched) {
        const done = new Set();
        for (const start of touched) {
            for (let el = start; el; el = el.parentNode) {
                if (!el.dataset || done.has(el)) continue;
                if (el.dataset.nodeType === "MapView") {
                    done.add(el);
                    syncMap(el);
                }
            }
        }
    }

    // A MapView's region and a Marker's position, as dataset entries.
    //
    // The dataset rather than a closure over the props, because every listener
    // and every sync pass in this runtime re-reads its inputs off the element:
    // the element is what survives between a create and the patches that follow,
    // and a value captured at creation is a value that goes stale on the first
    // update-props. htmlout writes these same attributes into its static export,
    // which is what lets a loader upgrade one into a live map.
    //
    // Total, like every other writer here: an update-props patch carries the
    // whole new props map, so a key that is absent now means "gone", and a
    // guarded write would leave the old value standing. Keyed on the node type
    // because `lat`, `id` and `title` are plausible prop names for some future
    // node that means something else by them — the gate applySpacerSize sets the
    // precedent for.
    function applyMapProps(el, props, nodeType) {
        if (nodeType !== "MapView" && nodeType !== "Marker") return;
        const write = (key, value) => {
            if (value === undefined || value === null || value === "") {
                delete el.dataset[key];
            } else {
                el.dataset[key] = String(value);
            }
        };
        if (nodeType === "MapView") {
            write("lat", props.lat);
            write("lng", props.lng);
            write("zoom", props.zoom);
            write("showUser", props.showUser === true ? "true" : "");
            return;
        }
        write("markerId", props.id);
        write("lat", props.lat);
        write("lng", props.lng);
        write("title", props.title);
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
            // The run is the level of a grid whose spaces are content — an
            // indent, the gap between two coloured tokens, a terminal's blank
            // cells. See the grid chassis in styleFromGrMob for the other two
            // levels and why the significance sits down here.
            span.style.whiteSpace = "pre";
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


    // --- CodeEditor ----------------------------------------------------------
    //
    // core.CodeEditor is a transparent <textarea> laid over a mirror of
    // coloured rows, both inside one <pre>. The runtime owns both elements,
    // which is the whole reason this is a node type rather than a pure-Go
    // composition: only something that creates both can guarantee they are the
    // same font at the same pitch with the same line height, and a Go-side
    // ZStack of a TextArea over a TextGrid cannot (core.Style has no
    // font-family). See core/codeeditor.go for the long version.
    //
    //	<pre data-node-type="CodeEditor">          the scroll box, position:relative
    //	  <div data-grmob-chrome="codegutter">     line numbers, out of flow
    //	  <textarea data-grmob-chrome="codebuffer">the real buffer, ink transparent
    //	  <div data-node-path=".../0">             row 0: the coloured mirror
    //	  <div data-node-path=".../1">             row 1
    //	  <div data-grmob-chrome="codefiller">     a line Go has no row for yet
    //	</pre>
    //
    // # Why the rows stay direct children
    //
    // Patches are addressed positionally, so the DOM has to stay isomorphic to
    // the node tree: a wrapper around the rows would send every row patch to
    // the wrong element. htmlout emits the identical shape for the same reason.
    //
    // The chrome is not part of that tree — no data-node-path, marked
    // data-grmob-chrome — and the gutter and buffer are always *leading*, which
    // is what keeps chromeOffset a fixed shift rather than a search. The
    // fillers are the one exception and are trailing, which is why the
    // add-child arm counts node children rather than subtracting the leading
    // chrome; see nodeChildCount.
    //
    // # No CodeMirror
    //
    // Deliberately: a large optional dependency for colours Go has already
    // computed, and untestable against wasm/verify's DOM. Reconsider if
    // autocomplete or folding become drivers.

    // The line-number column. Out of flow at the left edge of the padding the
    // box opens for it, right-aligned so the digits line up on their units
    // column, and inert to the pointer so a drag that starts over the numbers
    // still selects text in the buffer behind them. Same declarations as
    // htmlout's codeGutterStyle.
    const CODE_GUTTER_STYLE = {
        position: "absolute",
        left: "0",
        top: "0",
        paddingRight: "1ch",
        boxSizing: "border-box",
        textAlign: "right",
        whiteSpace: "pre",
        opacity: "0.45",
        userSelect: "none",
        pointerEvents: "none",
    };

    // The real buffer, made invisible without being hidden.
    //
    // `color: transparent` rather than `opacity: 0` or `visibility: hidden`: an
    // invisible textarea still has to show a caret and a selection highlight,
    // and only the colour is transparent — `caret-color: currentColor` paints
    // the caret in the editor's own ink, and ::selection still paints behind
    // the glyphs. The other two spellings would take all three away.
    //
    // font and line-height inherit from the <pre>, which is what puts the
    // buffer's glyphs on exactly the mirror's cell grid; the shorthand is
    // assigned before lineHeight because `font` resets line-height.
    const CODE_BUFFER_STYLE = {
        position: "absolute",
        top: "0",
        left: "0",
        margin: "0",
        padding: "0",
        border: "0",
        outline: "none",
        resize: "none",
        overflow: "hidden",
        background: "transparent",
        color: "transparent",
        caretColor: "currentColor",
        font: "inherit",
        lineHeight: "inherit",
        whiteSpace: "pre",
    };

    // A blank stand-in for a buffer line the mirror has no row for. Same two
    // declarations a GridRow gets (styleFromGrMob), because it stands where a
    // row would and has to take the same height.
    const CODE_FILLER_STYLE = { minHeight: "1.2em", whiteSpace: "nowrap" };

    // The props this runtime handles through applyCodeEditorProps rather than
    // through createElement's generic per-key chain.
    //
    // Every one of them would otherwise be mishandled rather than merely
    // ignored: `onChange` and `onSelectionChange` would attach listeners to the
    // <pre> (and mark the listener slot taken, so the real wiring could never
    // be installed), `value` and `placeholder` would be written onto an element
    // that has neither, and the editor's own four would be dropped on the
    // floor. Same interception onLongPress and onEndReached need, for the same
    // reason.
    const CODE_EDITOR_PROPS = new Set([
        "value", "placeholder", "onChange", "onSelectionChange",
        "lineNumbers", "readOnly", "tabSize", "commentPrefix",
        "editorEpoch", "editorCommand",
    ]);

    // codeChrome finds one of an editor's chrome elements by its marker.
    // A loop rather than querySelector because the harness DOM supports one
    // selector shape and because the chrome is always within the first few
    // children anyway.
    function codeChrome(el, kind) {
        for (const child of el.children) {
            if (child.dataset.grmobChrome === kind) return child;
        }
        return null;
    }

    // buildCodeEditor creates the gutter and the buffer, once, and wires the
    // buffer's listeners. Idempotent: called from createElement on the create
    // path and defensively from the update path, because an editor that
    // arrived through some future route with no chrome would otherwise be a
    // read-only mirror with no way to type into it.
    //
    // Both elements are created even when lineNumbers is off — the gutter is
    // merely display:none — so that chromeOffset is the same number for the
    // life of the element. A gutter that came and went would change every
    // add-child index at the moment the toggle flipped.
    function buildCodeEditor(el) {
        if (codeChrome(el, "codebuffer")) return;

        const gutter = document.createElement("div");
        gutter.dataset.grmobChrome = "codegutter";
        // The numbers are not content: a reader announcing "one two three"
        // before every line is reading the chrome, not the code.
        gutter.setAttribute("aria-hidden", "true");
        Object.assign(gutter.style, CODE_GUTTER_STYLE);
        gutter.style.display = "none";

        const buffer = document.createElement("textarea");
        buffer.dataset.grmobChrome = "codebuffer";
        Object.assign(buffer.style, CODE_BUFFER_STYLE);
        // Every one of these corrupts source. They are attributes rather than
        // properties because three of the four are only attributes, and
        // because an exported document would carry the same four.
        buffer.setAttribute("spellcheck", "false");
        buffer.setAttribute("autocapitalize", "off");
        buffer.setAttribute("autocorrect", "off");
        buffer.setAttribute("autocomplete", "off");
        // A code line is one line: no soft wrapping, and a horizontal scroll
        // instead. The mirror says the same thing with white-space:nowrap.
        buffer.setAttribute("wrap", "off");
        buffer.value = "";

        // The echo ledger, exactly the pendingEchoes list both natives keep:
        // every value this editor sends upstream is queued, and an upstream
        // change matching a queued entry is an echo of our own edit rather
        // than Go speaking for itself. See applyCodeValue.
        el.__grmobEchoes = [];

        buffer.addEventListener("input", () => emitCodeChange(el, buffer));
        buffer.addEventListener("keydown", (e) => codeKeydown(el, buffer, e));
        // The selection, reported from the events that can move a caret.
        // Deliberately not document's `selectionchange`: it is the event
        // designed for this and it is also the one this runtime cannot rely on
        // — it is a document-level event, and reaching for the document to
        // learn about one element's caret is a listener that outlives the
        // element. reportCodeSelection dedupes, so the overlap between these
        // four costs nothing.
        for (const type of ["keyup", "mouseup", "select", "focus"]) {
            buffer.addEventListener(type, () => reportCodeSelection(el, buffer));
        }

        // Leading, and in this order, for the life of the element.
        el.insertBefore(gutter, el.children[0] || null);
        el.insertBefore(buffer, el.children[1] || null);
    }

    // applyCodeEditorProps is the editor's whole prop surface, on both the
    // create and the update path. props is the complete new props map in both
    // cases — reconcile emits new.Props and never a delta — so an `in` test
    // asks "did Go describe this", and core seeds every option on every pass
    // precisely so that the answer is always yes and "off" is a value rather
    // than an absence.
    function applyCodeEditorProps(el, props, created = false) {
        const buffer = codeChrome(el, "codebuffer");
        if (!buffer || !props) return;

        if ("readOnly" in props) {
            // The property, not the attribute, for the reason `checked` is a
            // property: this is live state, and the live state is what Go is
            // describing. A read-only textarea still focuses, still shows a
            // caret and still selects — which is the whole difference between
            // read-only and disabled.
            buffer.readOnly = !!props.readOnly;
            // And it leaves the tab order. This is the one thing the overlay
            // costs a *document*: a page of read-only code blocks would put a
            // tab stop in front of each one, where the <pre> they replace had
            // none — and there is nothing to do at that stop, because the
            // browser already lets anyone select and copy the mirror behind it.
            // tabindex="-1" rather than removing the element, so a programmatic
            // focus (the selectAll command) still works and chromeOffset stays
            // the same number for the life of the editor.
            buffer.setAttribute("tabindex", props.readOnly ? "-1" : "0");
        }
        if ("tabSize" in props) {
            const n = Number(props.tabSize) || 0;
            el.dataset.tabSize = n;
            // How wide a literal tab renders, which matters whenever the
            // buffer already contains tabs — the indent this runtime *inserts*
            // is spaces unless tabSize is 0. Set on both layers or the mirror
            // and the buffer disagree about where a tab ends.
            const stops = n > 0 ? String(n) : "4";
            el.style.tabSize = stops;
            buffer.style.tabSize = stops;
        }
        if ("commentPrefix" in props) {
            el.dataset.commentPrefix = String(props.commentPrefix ?? "");
        }
        if ("lineNumbers" in props) {
            el.dataset.lineNumbers = props.lineNumbers ? "true" : "false";
        }
        if ("placeholder" in props) {
            buffer.placeholder = String(props.placeholder ?? "");
        }
        // The callback IDs live on the *editor*, not on the buffer, and are
        // re-read at fire time — the same arrangement a TabView's bar uses,
        // and for the same reason: IDs are positional and a later pass may
        // have refreshed or pruned this one. pruneStaleListeners drops them
        // when a pass stops carrying them, leaving the editor inert rather
        // than dispatching to a dead handler.
        if ("onChange" in props) el.dataset.listener_onChange = props.onChange;
        if ("onSelectionChange" in props) {
            el.dataset.listener_onSelectionChange = props.onSelectionChange;
        }
        if ("value" in props) applyCodeValue(el, buffer, String(props.value ?? ""));
        if ("editorEpoch" in props) {
            // The epoch is the whole trigger; the command is only read once it
            // has moved. An update-props patch carries the entire new props
            // map, so an editor re-rendered for its value would otherwise
            // re-run whatever command was last issued. Epoch 0 means Go has
            // never issued one and stamps nothing at all, so it is unreachable
            // for an editor with no toolbar — checked anyway, because the two
            // props always travel together and a 0 must never be read as an
            // instruction.
            const changed = String(el.dataset.editorEpoch) !== String(props.editorEpoch);
            el.dataset.editorEpoch = props.editorEpoch;
            // `created` is why a freshly built editor adopts the stamp without
            // acting on it. A focus command deliberately re-fires on a field
            // that mounts while it is the target — that is what makes "push a
            // screen and put the cursor in its search box" work. An editor
            // command is the opposite: it names a *moment* and an edit, and an
            // editor that was not on the page when it was issued missed it.
            // Running it at creation would indent the buffer every time its
            // screen came back. All four hosts agree on this.
            if (changed && !created && Number(props.editorEpoch) !== 0) {
                runCodeCommand(el, String(props.editorCommand ?? ""));
            }
        }
    }

    // The echo guard, the same bookkeeping GrMobTextField keeps on both
    // natives: the buffer is the host's while focused and Go's otherwise.
    //
    // An upstream value that matches a queued echo is this editor's own edit
    // coming back, and is dropped — assigning it would move the caret to the
    // end mid-typing. The queue is dropped *through* the match rather than at
    // it, because Go may coalesce renders and skip intermediate values.
    //
    // An upstream value matching nothing we sent can only be Go speaking for
    // itself — a validator normalizing the text, a draft being cleared after a
    // submit — so it wins even mid-typing. Moving the caret then is correct:
    // the text under it was replaced.
    function applyCodeValue(el, buffer, value) {
        const echoes = el.__grmobEchoes || (el.__grmobEchoes = []);
        if (document.activeElement !== buffer) {
            // Go-owned while blurred; any queued echoes died with the session.
            echoes.length = 0;
            if (buffer.value !== value) buffer.value = value;
            return;
        }
        const echo = echoes.indexOf(value);
        if (echo >= 0) {
            echoes.splice(0, echo + 1);
            return;
        }
        echoes.length = 0;
        if (buffer.value !== value) {
            buffer.value = value;
            setCodeCaret(buffer, value.length, value.length);
        }
    }

    // emitCodeChange is the one path every local edit leaves by: the typing
    // the browser did for us, and the two keystrokes this runtime handles
    // itself. It repaints first and dispatches second, so the glyph is on
    // screen before Go has heard about it.
    function emitCodeChange(el, buffer) {
        const value = buffer.value;
        (el.__grmobEchoes || (el.__grmobEchoes = [])).push(value);
        syncCodeEditor(el);
        const cbId = el.dataset.listener_onChange;
        if (cbId) window.GoInvokeCallback(cbId, { value });
        reportCodeSelection(el, buffer);
    }

    // Tab and Enter, the two keys a programmer's editor must take away from
    // the browser.
    //
    // Tab would otherwise move focus out of the editor, which makes indenting
    // impossible; Enter would insert a bare newline at column zero, which
    // un-indents every block as it is written. Both are prevented and
    // re-implemented; every other key is left to the browser, including the
    // ones that make a textarea worth using (undo, word motion, IME
    // composition).
    function codeKeydown(el, buffer, e) {
        if (buffer.readOnly) return;
        if (e.key === "Tab") {
            e.preventDefault();
            insertInCode(el, buffer, codeIndentUnit(el));
            return;
        }
        if (e.key === "Enter") {
            e.preventDefault();
            insertInCode(el, buffer, "\n" + codeLeadingSpace(buffer));
        }
    }

    // codeIndentUnit is what one indent is made of: tabSize spaces, or a
    // literal tab when tabSize is 0 — which is what Go source wants.
    function codeIndentUnit(el) {
        const n = Number(el.dataset.tabSize) || 0;
        return n > 0 ? " ".repeat(n) : "\t";
    }

    // codeLeadingSpace is the indentation of the line the caret is in, which
    // Enter copies onto the new line. Read from the text before the caret so
    // that splitting a line mid-way still continues at that line's indent.
    function codeLeadingSpace(buffer) {
        const value = String(buffer.value ?? "");
        const caret = codeSelection(buffer).start;
        const lineStart = value.lastIndexOf("\n", caret - 1) + 1;
        const line = value.slice(lineStart, caret);
        return line.slice(0, line.length - line.trimStart().length);
    }

    // codeSelection reads the caret as two UTF-16 offsets, which is the unit
    // the DOM speaks. Defaulted rather than assumed present because the
    // harness DOM has no selection of its own and because a textarea that has
    // never been focused reports null in some browsers.
    function codeSelection(buffer) {
        const length = String(buffer.value ?? "").length;
        const start = Number(buffer.selectionStart ?? length) || 0;
        const end = Number(buffer.selectionEnd ?? start) || 0;
        return start <= end ? { start, end } : { start: end, end: start };
    }

    // setCodeCaret moves the selection, through the DOM's own method where
    // there is one. The fallback assignment is for the harness, which models
    // the two offsets as plain properties — and is also what a browser does
    // when the element is not focused.
    function setCodeCaret(buffer, start, end) {
        if (typeof buffer.setSelectionRange === "function") {
            buffer.setSelectionRange(start, end);
            return;
        }
        buffer.selectionStart = start;
        buffer.selectionEnd = end;
    }

    // insertInCode replaces the selection with text and leaves the caret after
    // it — the browser's own typing behaviour, re-implemented for the two keys
    // whose default this runtime had to prevent.
    function insertInCode(el, buffer, text) {
        const value = String(buffer.value ?? "");
        const { start, end } = codeSelection(buffer);
        buffer.value = value.slice(0, start) + text + value.slice(end);
        setCodeCaret(buffer, start + text.length, start + text.length);
        emitCodeChange(el, buffer);
    }

    // runCodeCommand applies one core.RunEditorCommand to the buffer. An
    // unknown command is a no-op rather than an error: a toolbar that outgrew
    // its editor must not crash the screen.
    function runCodeCommand(el, command) {
        const buffer = codeChrome(el, "codebuffer");
        if (!buffer) return;
        if (command === "selectAll") {
            // Allowed on a read-only buffer: selecting is reading, which is
            // exactly what read-only permits.
            setCodeCaret(buffer, 0, String(buffer.value ?? "").length);
            if (typeof buffer.focus === "function") buffer.focus();
            reportCodeSelection(el, buffer);
            return;
        }
        if (buffer.readOnly) return;
        if (command === "indent" || command === "outdent" || command === "commentLine") {
            transformCodeLines(el, buffer, command);
        }
    }

    // transformCodeLines rewrites every line the selection touches.
    //
    // The selection is first widened to whole lines, because all three
    // commands are line commands: indenting "the middle of line 4" means
    // indenting line 4. The rewritten block then takes the selection, so a
    // second indent indents the same lines rather than a range that has
    // drifted under the first one's inserted characters.
    function transformCodeLines(el, buffer, command) {
        const value = String(buffer.value ?? "");
        const { start, end } = codeSelection(buffer);
        const from = value.lastIndexOf("\n", start - 1) + 1;
        const nextNewline = value.indexOf("\n", end);
        const to = nextNewline < 0 ? value.length : nextNewline;

        const lines = value.slice(from, to).split("\n");
        const unit = codeIndentUnit(el);
        const prefix = el.dataset.commentPrefix ?? "";
        let out;

        if (command === "indent") {
            out = lines.map((line) => unit + line);
        } else if (command === "outdent") {
            out = lines.map((line) => outdentCodeLine(line, unit));
        } else {
            // A language with no line comment (JSON) sets an empty prefix and
            // gets a command that does nothing, rather than one that inserts a
            // marker making the document invalid.
            if (!prefix) return;
            // The toggle is decided for the whole run, not per line: a
            // partially-commented block becomes fully commented rather than
            // inverting line by line, which is what every editor does and what
            // makes the command its own undo. Blank lines do not vote.
            const allCommented = lines.every(
                (line) => line.trim() === "" || line.trimStart().startsWith(prefix)
            );
            out = allCommented
                ? lines.map((line) => uncommentCodeLine(line, prefix))
                : lines.map((line) => commentCodeLine(line, prefix));
        }

        const replaced = out.join("\n");
        buffer.value = value.slice(0, from) + replaced + value.slice(to);
        setCodeCaret(buffer, from, from + replaced.length);
        emitCodeChange(el, buffer);
    }

    // outdentCodeLine removes one indent's worth of leading white space, and
    // leaves a line that has none alone rather than eating a glyph.
    //
    // The tab is stripped whatever the unit is, because a buffer mixes them:
    // a file indented with tabs outdented by a four-space unit would otherwise
    // lose nothing at all.
    function outdentCodeLine(line, unit) {
        if (line.startsWith(unit)) return line.slice(unit.length);
        if (line.startsWith("\t")) return line.slice(1);
        let i = 0;
        while (i < unit.length && line[i] === " ") i++;
        return line.slice(i);
    }

    // commentCodeLine inserts the prefix at the start of the line's
    // *indentation*, not at column zero, so a commented block keeps the shape
    // of the code it came from. The space after the prefix is what every
    // formatter writes and what uncommentCodeLine takes back off.
    //
    // A blank line is left blank: a file of "// " on its empty lines is
    // trailing white space that a formatter will strip on the next save.
    function commentCodeLine(line, prefix) {
        if (line.trim() === "") return line;
        const indent = line.length - line.trimStart().length;
        return line.slice(0, indent) + prefix + " " + line.slice(indent);
    }

    // uncommentCodeLine removes the first prefix and the single space that
    // usually follows it. One space, not all of them: "//     aligned" is a
    // comment whose own indentation is part of what it says.
    function uncommentCodeLine(line, prefix) {
        const at = line.indexOf(prefix);
        if (at < 0) return line;
        let after = at + prefix.length;
        if (line[after] === " ") after++;
        return line.slice(0, at) + line.slice(after);
    }

    // reportCodeSelection dispatches the caret as "start:end" in *byte*
    // offsets into the UTF-8 value, which is the unit core.OnSelectionChange
    // promises and the one all four hosts can agree on. The DOM counts UTF-16
    // code units, so the conversion is this function's whole job beyond the
    // dispatch.
    //
    // Deduped against the last payload because the four events this is wired
    // to overlap heavily — a keystroke fires input and keyup — and each
    // dispatch is a Go render pass. An unchanged selection is not news.
    function reportCodeSelection(el, buffer) {
        const cbId = el.dataset.listener_onSelectionChange;
        if (!cbId) return;
        const value = String(buffer.value ?? "");
        const { start, end } = codeSelection(buffer);
        const payload =
            utf8Length(value.slice(0, start)) + ":" + utf8Length(value.slice(0, end));
        if (el.dataset.codeSelection === payload) return;
        el.dataset.codeSelection = payload;
        window.GoInvokeCallback(cbId, { value: payload });
    }

    // utf8Length is the byte length of a JavaScript string as UTF-8.
    //
    // Hand-counted rather than `new TextEncoder().encode(s).length` because
    // this runtime is evaluated in contexts that provide a deliberately small
    // set of globals (see wasm/verify/load.mjs), and because the count is the
    // only part of the encoder that is wanted — encoding a whole prefix to
    // measure it allocates a buffer per keystroke.
    //
    // The surrogate arithmetic is the part worth reading: a code point above
    // the BMP is two UTF-16 units and four UTF-8 bytes, so a high surrogate
    // contributes 4 and its partner is skipped.
    function utf8Length(s) {
        let bytes = 0;
        for (let i = 0; i < s.length; i++) {
            const c = s.charCodeAt(i);
            if (c < 0x80) bytes += 1;
            else if (c < 0x800) bytes += 2;
            else if (c >= 0xd800 && c <= 0xdbff && i + 1 < s.length) {
                bytes += 4;
                i++;
            } else bytes += 3;
        }
        return bytes;
    }

    // syncCodeEditor brings the mirror, the fillers and the gutter into
    // agreement with the buffer. Idempotent and cheap, and called from every
    // place either half can have moved: after the children exist on the create
    // path, after every patch batch that touched the editor, and after every
    // local edit.
    //
    // This is where rule 2 of the shared editor design lives — decoration is
    // advisory and per line — so it is worth stating plainly what it protects
    // against. Go is a keystroke behind for a few milliseconds after every
    // keypress: its rows describe the text as it was *before* the key. Painting
    // them anyway would show the user the old line while they type the new one.
    // Refusing to paint the whole editor would make it flash plain on every
    // keystroke. So the decision is per line, and only the line being edited
    // loses its colours, for one frame.
    function syncCodeEditor(el) {
        const buffer = codeChrome(el, "codebuffer");
        if (!buffer) return;
        const lines = String(buffer.value ?? "").split("\n");

        const rows = [];
        for (const child of el.children) {
            if (child.getAttribute("data-node-path") !== null) rows.push(child);
        }
        rows.forEach((row, i) => syncCodeRow(row, lines[i]));
        syncCodeFillers(el, rows.length, lines);
        // The gutter numbers what is on screen, which is the longer of the two
        // sides: the buffer's lines (Go is behind, and the fillers are standing
        // in) or Go's rows (Go is ahead, and the extra rows are still showing
        // their own text). Numbering only the buffer would leave a drawn line
        // with no number beside it for a frame, which reads as a dropped line.
        syncCodeGutter(el, Math.max(rows.length, lines.length));
        sizeCodeBuffer(el, buffer);
    }

    // syncCodeRow decides whether row i shows Go's colours or the buffer's own
    // plain text, and rebuilds it only when that decision (or the text behind
    // it) has actually changed.
    //
    // The runs are kept on the element rather than re-read from a patch,
    // because the decision has to be *re-made* on every keystroke while the
    // runs stand still: a line that disagreed a moment ago and agrees now must
    // get its colours back, and Go sends no new patch for a row whose runs did
    // not change. Storing them is what makes the rule reversible.
    function syncCodeRow(row, line) {
        const runs = Array.isArray(row.__grmobRuns) ? row.__grmobRuns : [];
        let text = "";
        for (const run of runs) text += String((run && run.t) ?? "");

        if (line !== undefined && text === line) {
            if (row.dataset.codeRow !== "decorated") {
                applyGridRuns(row, runs);
                row.dataset.codeRow = "decorated";
                delete row.dataset.codeText;
            }
            return;
        }
        // A row with no line under it at all (Go is ahead: it still has rows
        // for text the user has deleted) shows its own text rather than
        // nothing, so the mirror never goes blank under a live buffer.
        const plain = line === undefined ? text : line;
        if (row.dataset.codeRow === "plain" && row.dataset.codeText === plain) return;
        applyGridRuns(row, plain === "" ? [] : [{ t: plain }]);
        row.dataset.codeRow = "plain";
        row.dataset.codeText = plain;
    }

    // syncCodeFillers keeps one blank stand-in per buffer line the mirror has
    // no row for. Without them, the first thing a new line does is disappear:
    // the user presses Enter, the buffer has a line Go has not sent a row for,
    // and its glyphs are transparent over nothing.
    //
    // They are chrome — no data-node-path, no patch addressed to one — and
    // they are the only chrome in this runtime that *trails* the node
    // children, which is why nodeChildCount exists.
    function syncCodeFillers(el, rowCount, lines) {
        const have = [];
        for (const child of el.children) {
            if (child.dataset.grmobChrome === "codefiller") have.push(child);
        }
        const want = Math.max(0, lines.length - rowCount);
        while (have.length > want) have.pop().remove();
        while (have.length < want) {
            const filler = document.createElement("div");
            filler.dataset.grmobChrome = "codefiller";
            Object.assign(filler.style, CODE_FILLER_STYLE);
            el.appendChild(filler);
            have.push(filler);
        }
        have.forEach((filler, i) => {
            const text = lines[rowCount + i] ?? "";
            if (filler.dataset.codeText === text) return;
            filler.dataset.codeText = text;
            applyGridRuns(filler, text === "" ? [] : [{ t: text }]);
        });
    }

    // syncCodeGutter draws the line numbers and opens the padding they sit in.
    //
    // One text node with newlines in it rather than one element per line: the
    // gutter is `white-space: pre` at the box's own line height, so number N
    // lands beside row N by construction instead of by two element lists being
    // kept the same length.
    //
    // The width is in `ch` — the width of a "0", which in a fixed-pitch box is
    // the width of every glyph — so the gutter is exactly as wide as its widest
    // number at any font size, with nothing measured. htmlout computes the
    // same string (gutterWidth).
    function syncCodeGutter(el, lines) {
        const gutter = codeChrome(el, "codegutter");
        if (!gutter) return;
        if (el.dataset.lineNumbers !== "true") {
            gutter.style.display = "none";
            el.style.paddingLeft = "";
            return;
        }
        gutter.style.display = "";
        const width = String(String(lines).length + 2) + "ch";
        gutter.style.width = width;
        // Written after any update-style patch in the same batch, which is what
        // the sync pass running at the end of patch() buys: a style patch
        // assigns the `padding` shorthand and would otherwise wipe this.
        el.style.paddingLeft = width;

        let text = "";
        for (let i = 1; i <= lines; i++) text += (i > 1 ? "\n" : "") + i;
        if (gutter.textContent !== text) gutter.textContent = text;
    }

    // sizeCodeBuffer stretches the transparent textarea over the whole mirror,
    // including the parts of it that are scrolled out of view.
    //
    // An absolutely positioned child sized with `inset: 0` would cover the
    // *visible* box only, so a click below the fold — or the caret arriving
    // there — would miss the buffer entirely. The mirror's scroll size is the
    // only thing that knows how big "the whole buffer" is.
    //
    // Skipped whole where there is no layout to read (the wasm/verify DOM),
    // which is honest rather than convenient: a shim cannot answer a question
    // about pixels, and a number invented here would be a lie in a style
    // property. The claim this function makes is one for a browser to check.
    function sizeCodeBuffer(el, buffer) {
        if (typeof el.scrollWidth !== "number" || typeof el.scrollHeight !== "number") return;
        const gutter = codeChrome(el, "codegutter");
        const inset =
            gutter && gutter.style.display !== "none" ? gutter.offsetWidth || 0 : 0;
        buffer.style.left = inset + "px";
        buffer.style.width = Math.max(0, el.scrollWidth - inset) + "px";
        buffer.style.height = el.scrollHeight + "px";
    }

    // The patch pass. Up from each touched element as well as at it, because a
    // row's runs patch names the row and the editor is what has to re-decide
    // the stale-line rule for it.
    function syncTouchedCodeEditors(touched) {
        const done = new Set();
        for (const start of touched) {
            for (let el = start; el && el.dataset; el = el.parentNode) {
                if (el.dataset.nodeType !== "CodeEditor" || done.has(el)) continue;
                done.add(el);
                syncCodeEditor(el);
            }
        }
    }


    // --- RichTextEditor ------------------------------------------------------
    //
    // core.RichTextEditor is a `contenteditable` <div> whose value is a
    // richtext.Doc. The runtime owns both serializers — Doc to DOM and DOM back
    // to Doc — and every command is a *pure transformation of the Doc*.
    //
    // # Why the commands are Doc transformations and not Range surgery
    //
    // The obvious implementation of "make this bold" in a contenteditable is to
    // wrap the Range in a <strong>, and it is a trap. Browsers split and merge
    // nodes freely under contenteditable, a Range can start inside one element
    // and end inside another, and the result of the surgery is markup that the
    // serializer then has to make sense of anyway. document.execCommand would do
    // it for us and is deprecated, differs per browser, and produces a different
    // document in each.
    //
    // So the selection is only ever *read* and *restored*, never operated on:
    //
    //	 selection ──▶ two character offsets into the document's plain text
    //	                    │
    //	                    ▼
    //	 doc + offsets + command ──▶ a new doc          (pure, and tested)
    //	                    │
    //	                    ▼
    //	 rebuild the DOM, restore the caret at the same offsets
    //
    // Everything in the middle is a function of values with no DOM in it, which
    // is what lets wasm/verify test the whole command vocabulary against a DOM
    // that has no Selection API at all. The two browser-only steps are small
    // enough to read in one sitting and are the only things a real browser is
    // needed for.
    //
    // # Typing is the other direction, and does not re-render
    //
    // While the user types, the DOM is authoritative: the `input` handler reads
    // the DOM back into a Doc and sends it upstream, and nothing rebuilds. A
    // rebuild on every keystroke would put the caret at the start of the
    // document on every keystroke. Go's echo of that Doc is dropped by the same
    // guard core.TextArea has always used; a Doc Go sends that this editor never
    // sent is a deliberate rewrite and *does* rebuild.
    //
    // # Paste goes through the serializer, so foreign markup dies at the edge
    //
    // A paste is prevented, its text/plain taken, and inserted into the Doc as
    // text. Nothing a word processor or another web page puts on the clipboard
    // reaches the document — which is the only way to keep the value a
    // richtext.Doc rather than whatever HTML happened to be copied.
    //
    // # Undo is this runtime's own stack
    //
    // A browser's native undo does not survive the programmatic rebuilds a
    // command makes, so the runtime keeps a stack of Docs and their selections.
    // The two natives use their platform's undo manager, which does survive
    // their own edits; this is the one place the three live hosts genuinely
    // differ in mechanism rather than in spelling.

    // The wire shape, restated: core/richtext.go sends richtext.Doc's JSON, so
    // everything below works on `{b:[{k,r:[{t,b,i,u,s,c,l}]}]}` directly rather
    // than converting to some other shape first. One representation, and it is
    // the one that crosses the wire in both directions.
    //
    // The block kinds are richtext.BlockKind's values, which is why that type is
    // a string in Go: the `block:` command, the document and this table all name
    // a heading with the same token.
    const RICH_BLOCK_TAG = {
        p: "p",
        h1: "h1",
        h2: "h2",
        h3: "h3",
        bullet: "li",
        numbered: "li",
        quote: "blockquote",
        code: "pre",
    };

    // The inverse, for DOM -> Doc. `li` is absent because a list item's kind
    // depends on the list it is in, which richTextFromDOM reads from the parent.
    const RICH_TAG_KIND = {
        P: "p",
        H1: "h1",
        H2: "h2",
        H3: "h3",
        BLOCKQUOTE: "quote",
        PRE: "code",
    };

    // The five marks, as (Doc key, tag) pairs in the order they nest — link
    // outermost, code innermost. The order is the same one richtext's Markdown
    // and HTML emitters use, so a document drawn here and the same document
    // exported by htmlout have the same shape.
    const RICH_MARKS = [
        { key: "b", tag: "STRONG" },
        { key: "i", tag: "EM" },
        { key: "s", tag: "S" },
        { key: "u", tag: "U" },
        { key: "c", tag: "CODE" },
    ];

    const RICH_EDITOR_PROPS = new Set([
        "doc", "placeholder", "onChange", "onSelectionChange",
        "readOnly", "editorEpoch", "editorCommand",
    ]);

    // --- The pure half: a Doc, two offsets, and a command --------------------

    // richBlockText is one block's text; richDocText is the whole document's,
    // with blocks joined by a newline.
    //
    // This is the coordinate system every offset below is in, and it is
    // deliberately richtext.Doc.PlainText's: one newline per block boundary, so
    // an empty block is one character wide and a caret can sit in it.
    function richBlockText(block) {
        let out = "";
        for (const run of (block && block.r) || []) out += (run && run.t) || "";
        return out;
    }

    function richDocText(doc) {
        return ((doc && doc.b) || []).map(richBlockText).join("\n");
    }

    // richLocate converts a document offset into a block index and an offset
    // within it. Clamped at both ends, because a selection restored after a
    // command may name a position the new document is shorter than.
    function richLocate(doc, offset) {
        const blocks = (doc && doc.b) || [];
        let at = Math.max(0, offset);
        for (let i = 0; i < blocks.length; i++) {
            const length = richBlockText(blocks[i]).length;
            if (at <= length) return { block: i, offset: at };
            at -= length + 1; // the newline between this block and the next
        }
        const last = Math.max(0, blocks.length - 1);
        return { block: last, offset: richBlockText(blocks[last]).length };
    }

    // richSplitRuns cuts a block's runs at a character offset, so that a range
    // ending mid-run can be given its own marks without disturbing the rest.
    //
    // Returns a new array; nothing here mutates a Doc it was handed, which is
    // what makes the undo stack a stack of values rather than a stack of
    // promises.
    function richSplitRuns(runs, offset) {
        const out = [];
        let at = 0;
        for (const run of runs || []) {
            const text = (run && run.t) || "";
            const end = at + text.length;
            if (offset > at && offset < end) {
                out.push({ ...run, t: text.slice(0, offset - at) });
                out.push({ ...run, t: text.slice(offset - at) });
            } else {
                out.push({ ...run });
            }
            at = end;
        }
        return out;
    }

    // richMergeRuns collapses adjacent runs that carry identical formatting and
    // drops the empty ones.
    //
    // Not tidiness: the runs are the wire, and a document that gained a run
    // boundary on every keystroke would grow without bound while looking
    // identical on screen. It is the same maximal-run rule the highlight package
    // states for a GridRow, for the same reason.
    function richMergeRuns(runs) {
        const out = [];
        for (const run of runs || []) {
            if (!run || !run.t) continue;
            const previous = out[out.length - 1];
            if (previous && richSameMarks(previous, run)) {
                previous.t += run.t;
                continue;
            }
            out.push({ ...run });
        }
        return out;
    }

    function richSameMarks(a, b) {
        for (const mark of RICH_MARKS) {
            if (!!a[mark.key] !== !!b[mark.key]) return false;
        }
        return (a.l || "") === (b.l || "");
    }

    // richMapRange applies `edit` to every run inside [start, end) of the
    // document, splitting at both ends first so the range is run-aligned.
    //
    // An empty range touches nothing and returns the document unchanged, which
    // is what makes a mark command with a bare caret a no-op here — see
    // richPendingMarks for what happens instead.
    function richMapRange(doc, start, end, edit) {
        if (end <= start) return doc;
        const from = richLocate(doc, start);
        const to = richLocate(doc, end);
        const blocks = ((doc && doc.b) || []).map((block) => ({ ...block, r: (block.r || []).map((r) => ({ ...r })) }));

        for (let i = from.block; i <= to.block && i < blocks.length; i++) {
            const text = richBlockText(blocks[i]);
            const lo = i === from.block ? from.offset : 0;
            const hi = i === to.block ? to.offset : text.length;
            if (hi <= lo) continue;

            let runs = richSplitRuns(blocks[i].r || [], lo);
            runs = richSplitRuns(runs, hi);
            let at = 0;
            runs = runs.map((run) => {
                const next = at + run.t.length;
                const inside = at >= lo && next <= hi;
                at = next;
                return inside ? edit({ ...run }) : run;
            });
            blocks[i].r = richMergeRuns(runs);
        }
        return { b: blocks };
    }

    // richRangeHasMark reports whether *every* run in the range already carries
    // the mark, which is what decides a toggle's direction.
    //
    // All rather than any: selecting a sentence with one bold word in it and
    // pressing bold should make the sentence bold, not unbold the word. That is
    // what every editor does and the rule richtext's own commentLine toggle
    // makes one package over.
    function richRangeHasMark(doc, start, end, key) {
        if (end <= start) return false;
        let all = true;
        let seen = false;
        richMapRange(doc, start, end, (run) => {
            seen = true;
            if (!run[key]) all = false;
            return run;
        });
        return seen && all;
    }

    // richSetBlockKind makes every block the range touches the given kind.
    function richSetBlockKind(doc, start, end, kind) {
        const from = richLocate(doc, start).block;
        const to = richLocate(doc, Math.max(start, end)).block;
        const blocks = ((doc && doc.b) || []).map((block, i) => {
            if (i < from || i > to) return block;
            const next = { ...block, k: kind };
            // Paragraph is the wire's absent kind, so writing it explicitly
            // would make a document that round-trips through here differ from
            // one Go marshalled — same bytes, different keys.
            if (kind === "p") delete next.k;
            return next;
        });
        return { b: blocks };
    }

    // richInsertText replaces [start, end) with text, splitting blocks at every
    // newline in it.
    //
    // This is the paste path and the pending-mark path, and it is the one edit
    // that changes the document's *shape* rather than its marks — which is why
    // it is spelled out here rather than left to the browser. The inserted text
    // takes the marks of the run it lands in, which is what typing into the
    // middle of a bold word does.
    function richInsertText(doc, start, end, text) {
        const cleaned = String(text ?? "").replace(/\r\n/g, "\n");
        const cut = richMapRange(doc, start, end, () => ({ t: "" }));
        const at = richLocate(cut, start);
        const blocks = ((cut && cut.b) || []).map((block) => ({ ...block, r: (block.r || []).map((r) => ({ ...r })) }));
        if (blocks.length === 0) blocks.push({ r: [] });

        const target = blocks[Math.min(at.block, blocks.length - 1)];
        const runs = richSplitRuns(target.r || [], at.offset);
        // The marks the insertion inherits: the run to the left of the caret,
        // or the one to the right at the start of a block.
        let carried = {};
        let seen = 0;
        for (const run of runs) {
            if (seen >= at.offset) break;
            seen += run.t.length;
            carried = { ...run, t: "" };
        }
        if (seen === 0 && runs.length) carried = { ...runs[0], t: "" };

        const lines = cleaned.split("\n");
        const head = [];
        const tail = [];
        let position = 0;
        for (const run of runs) {
            (position < at.offset ? head : tail).push(run);
            position += run.t.length;
        }

        if (lines.length === 1) {
            target.r = richMergeRuns([...head, { ...carried, t: lines[0] }, ...tail]);
            return { b: blocks };
        }
        // A multi-line insertion splits the block: the first line joins what was
        // before the caret, the last joins what was after, and the lines between
        // become blocks of their own.
        const made = [{ ...target, r: richMergeRuns([...head, { ...carried, t: lines[0] }]) }];
        for (let i = 1; i < lines.length - 1; i++) {
            made.push({ ...target, r: richMergeRuns([{ ...carried, t: lines[i] }]) });
        }
        made.push({ ...target, r: richMergeRuns([{ ...carried, t: lines[lines.length - 1] }, ...tail]) });
        blocks.splice(at.block, 1, ...made);
        return { b: blocks };
    }

    // applyRichCommand is the whole command vocabulary as one pure function.
    //
    // Returns the new document, or null for a command this editor does not know
    // — which the caller treats as a no-op, because a toolbar that outgrew its
    // editor must not break the screen.
    //
    // `undo` and `redo` are absent on purpose: they are not transformations of
    // the document, they are movements through a history, so they are handled by
    // the caller where the history lives.
    function applyRichCommand(doc, start, end, command) {
        if (command.startsWith("block:")) {
            return richSetBlockKind(doc, start, end, command.slice("block:".length));
        }
        if (command.startsWith("link:")) {
            const url = command.slice("link:".length);
            return richMapRange(doc, start, end, (run) => ({ ...run, l: url }));
        }
        if (command === "unlink") {
            return richMapRange(doc, start, end, (run) => {
                const next = { ...run };
                delete next.l;
                return next;
            });
        }
        for (const mark of RICH_MARKS) {
            if (command !== richMarkCommand(mark.key)) continue;
            const on = !richRangeHasMark(doc, start, end, mark.key);
            return richMapRange(doc, start, end, (run) => {
                const next = { ...run };
                if (on) next[mark.key] = 1;
                else delete next[mark.key];
                return next;
            });
        }
        return null;
    }

    // The command name for a mark key. One table rather than two: the keys are
    // the wire's and the commands are core's Edit* constants, and this is the
    // one place they are paired.
    function richMarkCommand(key) {
        return { b: "bold", i: "italic", u: "underline", s: "strike", c: "code" }[key];
    }

    // richMarksAt reports the formatting active at a position, which is what a
    // toolbar draws its pressed state from.
    //
    // For a selection it is the marks every run in it carries; for a bare caret
    // it is the marks of the run to its left, which is what the next character
    // typed there would inherit.
    function richMarksAt(doc, start, end) {
        const marks = [];
        let link = "";
        const probeStart = end > start ? start : Math.max(0, start - 1);
        const probeEnd = end > start ? end : start;
        if (probeEnd > probeStart) {
            for (const mark of RICH_MARKS) {
                if (richRangeHasMark(doc, probeStart, probeEnd, mark.key)) {
                    marks.push(richMarkCommand(mark.key));
                }
            }
            richMapRange(doc, probeStart, probeEnd, (run) => {
                if (run.l) link = run.l;
                return run;
            });
        }
        return { marks, link };
    }

    // --- Doc -> DOM ----------------------------------------------------------

    // richTextToDOM rebuilds the editor's contents from a document.
    //
    // Every element it makes is chrome: no data-node-path, marked
    // data-grmob-chrome, and no patch is ever addressed to one. A
    // RichTextEditor has no node children at all — the document is one prop —
    // so, exactly like a <select>'s options, nothing has to count past these.
    //
    // Each run is wrapped in an element even when it carries no marks, and even
    // though a browser would be happy with a bare text node. Two reasons: the
    // serializer's element path and its text path then exercise the same shape,
    // and the harness DOM in wasm/verify has no text nodes at all, so a
    // structure that depended on them could not be tested anywhere.
    // # What this costs, measured, and why it is still a full rebuild
    //
    // Every write to this editor's DOM comes through here, and every one of
    // them recreates the whole document: the five callers are a pending mark
    // being applied, a paste, an undo or redo, any toolbar command, and any
    // rewrite arriving from Go. So the cost of one bold press is a function of
    // the document's size rather than of the edit's.
    //
    // Counted rather than timed, because a count is the same number in a
    // browser as it is against the harness DOM and a millisecond is not:
    //
    //	blocks x runs      elements destroyed and recreated
    //	   1 x 1                          2
    //	  10 x 4                         70
    //	 100 x 6                      1,000
    //	 500 x 6                      5,000
    //	2000 x 6                     20,000
    //
    // A patch that rewrote only the blocks an edit touched would put ~10 in
    // every one of those rows. That is the ratio, and it is why this is written
    // down rather than left to be re-derived: the next person to look at it
    // should start from the number.
    //
    // It is still a full rebuild for two reasons that are about correctness
    // rather than effort.
    //
    // First, a block is not an element here. Consecutive list blocks share one
    // <ul>, which this function's `list` variable is doing, so "replace block
    // N's element" is not a well-defined operation — the unit that can be
    // replaced is a maximal run of blocks sharing a container, and computing
    // that run is the part a partial rewrite has to get right on every edit.
    //
    // Second, and this is the one that decided it: leaving an element in place
    // is only safe if the element still describes its block, and between two
    // calls to this function the *browser* has been editing this DOM. Typing
    // under contenteditable splits text nodes and inserts elements of its own.
    // A full rebuild normalises all of that away every time; a partial one
    // would normalise the blocks it rewrote and leave the rest in whatever
    // shape the browser last left them. That is a drift between the model and
    // the screen, in an editor, and nothing in this repository can test for it
    // — the harness DOM has no Selection API and no contenteditable behaviour
    // at all, which is the same limit that made the whole command vocabulary a
    // pure document transformation in the first place.
    //
    // So the trade is: a rebuild proportional to the document on an action the
    // user initiated, against a normalisation hole that only a browser can
    // show. Taken deliberately, with the numbers above, and revisitable by
    // anyone who has a profile from a real document that says otherwise. The
    // proposal is written down in ai_docs/plans/non_goals.md rather than left
    // as a Next item that is re-argued every time somebody reads this function.
    //
    // # What was taken instead: the tree is built detached and attached once
    //
    // The element count above is unchanged and cannot be improved without the
    // partial rewrite this declines. What CAN be improved, with no bearing on
    // normalisation at all, is how many of those elements are inserted into a
    // LIVE contenteditable subtree.
    //
    // Each block's runs already went into a detached box; what did not was the
    // box itself, and the <ul>/<ol> a run of list blocks shares — both were
    // appended to `el`, which is in the document and is the element the browser
    // is editing. So the number of INSERTIONS into the attached tree was:
    //
    //	blocks x runs      appendChild onto an attached node
    //	   1 x 1                          1
    //	 100 x 6                        101      (100 blocks + their <ul>)
    //	2000 x 6                      2,001
    //
    // and it is 1 for every one of those rows now, whatever the document's
    // size: a fragment's children are spliced in by that single call, which is
    // the one case where appendChild is not "put this node here". The clear
    // before it is an innerHTML assignment, so the live tree is touched twice
    // in total rather than n+1 times.
    //
    // Pinned by wasm/verify/richtext_test.mjs at 1, 100 and 2000 blocks. As a
    // constant, because the regression to guard against is not a slowdown but
    // one more `el.appendChild` inside the loop — which every other test of
    // this function would go on passing, since they all assert shape and the
    // shape is identical either way.
    //
    // # Why that is worth a line of code, stated in the terms this file uses
    //
    // Not for layout. Browsers batch layout and nothing here reads geometry
    // back, so there was never a forced reflow per block to remove.
    //
    // For the editing machinery. `el` carries contenteditable, so the browser's
    // own editing implementation is watching this subtree, and so is any
    // MutationObserver an embedder attached. Building incrementally showed both
    // of them ~N intermediate states of a document mid-edit — partially
    // rewritten, briefly missing every block after the one being appended.
    // Building detached shows them one transition from the old document to the
    // new one, which is the only state that was ever true.
    //
    // It is also the half of the patch proposal that carries none of its risk:
    // every element is still new on every call, so the normalisation argument
    // above is untouched, word for word.
    function richTextToDOM(el, doc) {
        // The fragment stands in for `el` everywhere below, which is why
        // nothing in the loop had to change: appending a block to a fragment
        // and appending it to the editor are the same call.
        const frag = document.createDocumentFragment();
        let list = null;
        let listKind = "";
        for (const block of (doc && doc.b) || []) {
            const kind = block && block.k ? block.k : "p";
            const tag = RICH_BLOCK_TAG[kind] || "p";

            if (kind === "bullet" || kind === "numbered") {
                if (listKind !== kind) {
                    list = document.createElement(kind === "bullet" ? "ul" : "ol");
                    list.dataset.grmobChrome = "richblock";
                    frag.appendChild(list);
                    listKind = kind;
                }
            } else {
                list = null;
                listKind = "";
            }

            const box = document.createElement(tag);
            box.dataset.grmobChrome = "richblock";
            if (kind === "code") {
                // <pre> keeps its newlines, which is the whole reason it is the
                // element, and the inner <code> is what richtext.HTML writes.
                const code = document.createElement("code");
                code.dataset.grmobChrome = "richrun";
                code.textContent = richBlockText(block);
                box.appendChild(code);
            } else {
                richRunsToDOM(box, (block && block.r) || []);
            }
            (list || frag).appendChild(box);
        }
        // Cleared as late as possible: between here and the line above, `el`
        // still holds the document the reader was looking at. A clear at the
        // top would have emptied the editor for the duration of the build,
        // which is the intermediate state this arrangement exists to remove.
        el.innerHTML = "";
        el.appendChild(frag);
    }

    // richRunsToDOM writes one block's runs, nesting the mark elements in
    // RICH_MARKS' order with the link outermost — the same order richtext's own
    // HTML() uses, so a document drawn here and one exported by htmlout are the
    // same tree.
    function richRunsToDOM(box, runs) {
        for (const run of runs) {
            let node = document.createElement("span");
            node.dataset.grmobChrome = "richrun";
            node.textContent = (run && run.t) || "";
            // Reversed, so the *first* entry in RICH_MARKS ends up outermost:
            // each wrap encloses what came before it, so applying them in order
            // would put the last one on the outside. The order that results —
            // link, strong, em, s, u, code — is richtext's own HTML() order, so
            // a document drawn here and one exported by htmlout are the same
            // tree.
            for (const mark of [...RICH_MARKS].reverse()) {
                if (!run[mark.key]) continue;
                const wrapper = document.createElement(mark.tag.toLowerCase());
                wrapper.dataset.grmobChrome = "richrun";
                wrapper.appendChild(node);
                node = wrapper;
            }
            if (run.l) {
                const link = document.createElement("a");
                link.dataset.grmobChrome = "richrun";
                link.setAttribute("href", run.l);
                link.appendChild(node);
                node = link;
            }
            box.appendChild(node);
        }
        if (!runs.length) {
            // An empty block is a blank line the writer typed, and an empty
            // block element has no height. The <br> is what browsers put in an
            // empty contenteditable paragraph themselves, and it is what keeps a
            // caret able to sit there.
            const br = document.createElement("br");
            br.dataset.grmobChrome = "richrun";
            box.appendChild(br);
        }
    }

    // --- DOM -> Doc ----------------------------------------------------------

    // richTextFromDOM reads the editor's contents back into a document.
    //
    // Normalizing rather than trusting, which is the rule this whole direction
    // is written under: a browser under contenteditable splits and merges
    // elements freely, promotes a <span> to a <font>, leaves a stray <div> where
    // a paragraph was, and inserts <br>s nobody asked for. So nothing here reads
    // the structure it expects to find — it reads whatever is there and decides
    // what each piece *is*.
    //
    // The one thing it will not do is guess. An element whose tag means nothing
    // to this model contributes its text and none of its own meaning, which is
    // the same degradation richtext.normalizeKind makes for a block kind it does
    // not know: the words are never at risk, only their presentation.
    function richTextFromDOM(el) {
        const blocks = [];
        richCollectBlocks(el, blocks, "");
        if (!blocks.length) blocks.push({ r: [] });
        return { b: blocks };
    }

    function richCollectBlocks(parent, blocks, listKind) {
        for (const child of parent.children) {
            const tag = child.tagName;
            if (tag === "UL" || tag === "OL") {
                richCollectBlocks(child, blocks, tag === "UL" ? "bullet" : "numbered");
                continue;
            }
            if (tag === "PRE") {
                blocks.push({ k: "code", r: richTextOf(child) ? [{ t: richTextOf(child) }] : [] });
                continue;
            }
            const kind = tag === "LI" ? listKind || "bullet" : RICH_TAG_KIND[tag] || "p";
            const runs = richMergeRuns(richCollectRuns(child, {}));
            const block = { r: runs };
            if (kind !== "p") block.k = kind;
            blocks.push(block);
        }
        // A contenteditable whose blocks the browser stripped leaves bare text
        // under the editor itself. It is a paragraph, which is the only thing it
        // can be.
        if (!blocks.length && richTextOf(parent)) {
            blocks.push({ r: [{ t: richTextOf(parent) }] });
        }
    }

    // richCollectRuns walks one block, carrying the marks of whatever it is
    // nested inside. Both node kinds are handled: element children, which is
    // what this runtime builds, and text nodes, which is what a browser produces
    // the moment anybody types.
    function richCollectRuns(node, carried) {
        const out = [];
        const kids = richChildNodes(node);
        // The harness DOM (wasm/verify/dom.mjs) models elements only: an
        // element's text is a property on it rather than a child text node. So a
        // childless element carrying text is a text run there, and is never
        // reached in a browser — where a leaf element's childNodes holds the
        // text node this branch stands in for.
        if (!node.childNodes && kids.length === 0) {
            if (node.textContent) out.push({ ...carried, t: node.textContent });
            return out;
        }
        for (const child of kids) {
            if (child.nodeType === 3) {
                if (child.data) out.push({ ...carried, t: child.data });
                continue;
            }
            if (!child.tagName) continue;
            if (child.tagName === "BR") {
                // The filler a browser puts in an empty paragraph. It is not
                // content and must not become a newline inside a block: a block
                // is a line, and a newline in one would make the offsets
                // disagree with richDocText.
                continue;
            }
            const marks = { ...carried };
            for (const mark of RICH_MARKS) {
                if (child.tagName === mark.tag) marks[mark.key] = 1;
            }
            // The presentational spellings a browser substitutes for the
            // semantic ones, read as the same marks. A contenteditable that has
            // been through a paste or a native bold shortcut is full of them.
            if (child.tagName === "B") marks.b = 1;
            if (child.tagName === "I") marks.i = 1;
            if (child.tagName === "STRIKE" || child.tagName === "DEL") marks.s = 1;
            if (child.tagName === "INS") marks.u = 1;
            if (child.tagName === "A") {
                const href = child.getAttribute("href");
                if (href) marks.l = href;
            }
            out.push(...richCollectRuns(child, marks));
        }
        return out;
    }

    // richChildNodes is childNodes where there is one and children where there
    // is not — the harness DOM models elements only, and the element path is the
    // one this runtime's own output exercises.
    function richChildNodes(node) {
        return node.childNodes || node.children || [];
    }

    // richTextOf is textContent where the DOM computes it and a manual walk
    // where it does not. The harness stores textContent per element rather than
    // deriving it, so a <pre> holding a <code> reads as "" there.
    function richTextOf(node) {
        if (node.childNodes) return node.textContent || "";
        let out = node.textContent || "";
        for (const child of node.children || []) out += richTextOf(child);
        return out;
    }

    // --- The browser-only half: the selection, and the element's wiring ------

    // richSelectionOffsets reads the caret as two character offsets into the
    // document's plain text — the coordinate system every pure function above
    // works in — falling back to the last selection this editor remembered.
    //
    // # Why the fallback is not a convenience
    //
    // Clicking a toolbar button can take focus out of a contenteditable before
    // the click handler runs, and a browser collapses the selection when it
    // does. So by the time "make this bold" arrives, the live selection is
    // often gone — which would silently make every toolbar command act on a
    // bare caret at wherever the collapse left it. Remembering the selection as
    // it moves is the standard fix and the only one that does not depend on
    // intercepting mousedown on a button this runtime does not own: the toolbar
    // is Go's, several nodes away.
    //
    // It is also what makes the command vocabulary testable. wasm/verify's DOM
    // has no Selection API — a shim cannot answer a question about a caret, and
    // a number invented there would make every command test pass for the wrong
    // reason — so a test sets the remembered selection and drives the real path.
    function richSelectionOffsets(el) {
        const live = richLiveSelection(el);
        if (live) {
            el.__richSelection = live;
            return live;
        }
        return el.__richSelection || null;
    }

    function richLiveSelection(el) {
        if (typeof window.getSelection !== "function") return null;
        const selection = window.getSelection();
        if (!selection || selection.rangeCount === 0) return null;
        const range = selection.getRangeAt(0);
        if (typeof el.contains !== "function" || !el.contains(range.startContainer)) return null;
        const start = richOffsetOf(el, range.startContainer, range.startOffset);
        const end = richOffsetOf(el, range.endContainer, range.endOffset);
        if (start === null || end === null) return null;
        return start <= end ? { start, end } : { start: end, end: start };
    }

    // richOffsetOf converts a (container, offset) pair into a document offset.
    //
    // The walk is the same one richDocText describes: every block contributes
    // its text plus one newline, and every text node inside a block contributes
    // its length. Doing it by walking rather than by measuring means the two
    // cannot disagree about where a block boundary is.
    function richOffsetOf(el, container, offset) {
        let total = 0;
        let found = null;

        const walkBlock = (block) => {
            const visit = (node) => {
                if (found !== null) return;
                if (node === container && node.nodeType !== 3) {
                    // A container that is an element addresses its *children*,
                    // so the offset counts child nodes rather than characters.
                    let seen = 0;
                    for (const child of richChildNodes(node)) {
                        if (seen >= offset) break;
                        total += richTextOf(child).length;
                        seen++;
                    }
                    found = total;
                    return;
                }
                if (node.nodeType === 3) {
                    if (node === container) {
                        found = total + offset;
                        return;
                    }
                    total += node.data.length;
                    return;
                }
                if (node.tagName === "BR") return;
                for (const child of richChildNodes(node)) visit(child);
            };
            visit(block);
        };

        for (const block of richBlockElements(el)) {
            walkBlock(block);
            if (found !== null) return found;
            total += 1; // the newline between this block and the next
        }
        return found;
    }

    // richBlockElements is the editor's blocks in document order, flattening the
    // <ul>/<ol> wrappers so that a list item is a block like any other — which
    // is what the model says it is.
    function richBlockElements(el) {
        const out = [];
        for (const child of el.children) {
            if (child.tagName === "UL" || child.tagName === "OL") {
                for (const item of child.children) out.push(item);
                continue;
            }
            out.push(child);
        }
        return out;
    }

    // richRestoreSelection puts the caret back at two document offsets after a
    // rebuild. Browser-only, and silent where there is no Selection API.
    function richRestoreSelection(el, start, end) {
        // Remembered whether or not there is a live selection to set, because
        // the remembered one is what the next command will read — see
        // richSelectionOffsets.
        el.__richSelection = { start: Math.min(start, end), end: Math.max(start, end) };
        if (typeof window.getSelection !== "function" || typeof document.createRange !== "function") {
            return;
        }
        const from = richNodeAt(el, start);
        const to = richNodeAt(el, end);
        if (!from || !to) return;
        const range = document.createRange();
        range.setStart(from.node, from.offset);
        range.setEnd(to.node, to.offset);
        const selection = window.getSelection();
        selection.removeAllRanges();
        selection.addRange(range);
        el.__richSelection = { start: Math.min(start, end), end: Math.max(start, end) };
    }

    // richNodeAt is richOffsetOf's inverse: a document offset to the text node
    // and offset that holds it.
    function richNodeAt(el, offset) {
        let remaining = Math.max(0, offset);
        const blocks = richBlockElements(el);
        for (let i = 0; i < blocks.length; i++) {
            const length = richTextOf(blocks[i]).length;
            if (remaining <= length) {
                const hit = richTextNodeAt(blocks[i], remaining);
                if (hit) return hit;
                // A block with no text node in it at all (an empty paragraph
                // holding only its <br>) takes the caret at the block itself.
                return { node: blocks[i], offset: 0 };
            }
            remaining -= length + 1;
        }
        const last = blocks[blocks.length - 1];
        return last ? { node: last, offset: 0 } : null;
    }

    function richTextNodeAt(node, offset) {
        let remaining = offset;
        const visit = (current) => {
            if (current.nodeType === 3) {
                if (remaining <= current.data.length) return { node: current, offset: remaining };
                remaining -= current.data.length;
                return null;
            }
            for (const child of richChildNodes(current)) {
                const hit = visit(child);
                if (hit) return hit;
            }
            return null;
        };
        return visit(node);
    }

    // --- The element -------------------------------------------------------

    // buildRichTextEditor makes the div editable and wires the four events an
    // editor needs. Idempotent, like buildCodeEditor.
    function buildRichTextEditor(el) {
        if (el.dataset.richWired === "true") return;
        el.dataset.richWired = "true";
        el.setAttribute("role", "textbox");
        el.setAttribute("aria-multiline", "true");
        // The four that corrupt prose less than they corrupt code, and are still
        // wrong here: a document is the user's words.
        el.setAttribute("autocapitalize", "off");
        el.setAttribute("autocorrect", "off");
        el.setAttribute("spellcheck", "true"); // prose, unlike code, wants this

        // The editor's own history, and the document it last sent upstream.
        el.__richDoc = { b: [] };
        el.__richEchoes = [];
        el.__richUndo = [];
        el.__richRedo = [];
        el.__richPending = null;

        el.addEventListener("input", () => richInputHappened(el));
        el.addEventListener("paste", (e) => richPasteHappened(el, e));
        for (const type of ["keyup", "mouseup", "focus"]) {
            el.addEventListener(type, () => reportRichSelection(el));
        }
    }

    // The typing path. The DOM is authoritative here and nothing is rebuilt —
    // rebuilding on a keystroke would put the caret at the start of the document
    // on every keystroke.
    //
    // The one exception is a pending mark: "press bold, then type" has to make
    // the typed characters bold, and the browser knows nothing about the mark.
    // So the inserted range is given the marks and the document *is* rebuilt,
    // with the caret restored at exactly where it already was — which is
    // invisible, and happens once per pending mark rather than once per
    // keystroke.
    function richInputHappened(el) {
        if (el.isContentEditable === false) return;
        let doc = richTextFromDOM(el);
        const pending = el.__richPending;
        if (pending) {
            const where = richSelectionOffsets(el);
            const grew = richDocText(doc).length - richDocText(el.__richDoc).length;
            if (where && grew > 0) {
                doc = richMapRange(doc, where.end - grew, where.end, (run) => ({ ...run, ...pending }));
                richTextToDOM(el, doc);
                richRestoreSelection(el, where.end, where.end);
            }
            el.__richPending = null;
        }
        richPushHistory(el, el.__richDoc);
        richSendDoc(el, doc);
        reportRichSelection(el);
    }

    // Paste, prevented and re-done as a text insertion. This is the edge foreign
    // markup dies at: whatever a word processor put on the clipboard, what
    // reaches the document is its text.
    function richPasteHappened(el, e) {
        if (!e.clipboardData || typeof e.preventDefault !== "function") return;
        e.preventDefault();
        const text = e.clipboardData.getData("text/plain");
        if (!text) return;
        const where = richSelectionOffsets(el) || { start: 0, end: 0 };
        richPushHistory(el, el.__richDoc);
        const doc = richInsertText(el.__richDoc, where.start, where.end, text);
        richTextToDOM(el, doc);
        const caret = where.start + text.replace(/\r\n/g, "\n").length;
        richRestoreSelection(el, caret, caret);
        richSendDoc(el, doc);
        reportRichSelection(el);
    }

    // richSendDoc records a document as this editor's own and dispatches it.
    //
    // The echo ledger is the same one a CodeEditor keeps and a TextArea kept
    // before it: every value sent upstream is queued, and Go's echo of one is
    // dropped rather than applied.
    function richSendDoc(el, doc) {
        el.__richDoc = doc;
        const json = JSON.stringify(doc);
        (el.__richEchoes || (el.__richEchoes = [])).push(json);
        const cbId = el.dataset.listener_onChange;
        if (cbId) window.GoInvokeCallback(cbId, { value: json });
    }

    // The undo stack: documents, not operations. A stack of values needs no
    // inverse for each command and cannot drift from what is on screen, which
    // an operation log can; the cost is memory proportional to the edits, which
    // for a note is nothing.
    //
    // A browser's native undo is not usable here — it does not survive the
    // programmatic rebuilds a command makes — which is the one place the three
    // live hosts genuinely differ in mechanism: the natives use UndoManager and
    // EditText's own.
    const RICH_HISTORY_LIMIT = 100;

    function richPushHistory(el, doc) {
        const stack = el.__richUndo || (el.__richUndo = []);
        stack.push(JSON.stringify(doc));
        if (stack.length > RICH_HISTORY_LIMIT) stack.shift();
        el.__richRedo = [];
    }

    function richStep(el, from, to) {
        if (!from.length) return;
        to.push(JSON.stringify(el.__richDoc));
        const doc = JSON.parse(from.pop());
        richTextToDOM(el, doc);
        richSendDoc(el, doc);
        reportRichSelection(el);
    }

    // runRichCommand applies one core.RunEditorCommand.
    function runRichCommand(el, command) {
        if (el.dataset.readOnly === "true") return;
        if (command === "undo") {
            richStep(el, el.__richUndo || [], el.__richRedo || (el.__richRedo = []));
            return;
        }
        if (command === "redo") {
            richStep(el, el.__richRedo || [], el.__richUndo || (el.__richUndo = []));
            return;
        }
        const where = richSelectionOffsets(el) || { start: 0, end: 0 };

        // A mark command with nothing selected sets the typing attributes: the
        // marks the next characters typed will carry. Held on the element until
        // the next input, which is where they are applied — see
        // richInputHappened. A block or link command with an empty selection
        // still acts, because both are about the block or the run the caret is
        // in rather than about a range.
        if (where.end === where.start) {
            for (const mark of RICH_MARKS) {
                if (command !== richMarkCommand(mark.key)) continue;
                const pending = el.__richPending || {};
                if (pending[mark.key]) delete pending[mark.key];
                else pending[mark.key] = 1;
                el.__richPending = Object.keys(pending).length ? pending : null;
                reportRichSelection(el);
                return;
            }
        }

        const doc = applyRichCommand(el.__richDoc, where.start, where.end, command);
        // null is "this editor does not know that command", which is a no-op:
        // a toolbar that outgrew its editor must not break the screen.
        if (!doc) return;
        richPushHistory(el, el.__richDoc);
        richTextToDOM(el, doc);
        richRestoreSelection(el, where.start, where.end);
        richSendDoc(el, doc);
        reportRichSelection(el);
    }

    // reportRichSelection sends the caret and the formatting active at it, in
    // the shape core/richtext.go parses:
    //
    //	{"s":12,"e":18,"marks":["bold"],"link":"https://x","block":"h2"}
    //
    // Deduped against the last payload, because the events this is wired to
    // overlap heavily and each dispatch is a Go render pass.
    function reportRichSelection(el) {
        const cbId = el.dataset.listener_onSelectionChange;
        if (!cbId) return;
        const where = richSelectionOffsets(el) || { start: 0, end: 0 };
        const doc = el.__richDoc || { b: [] };
        const at = richMarksAt(doc, where.start, where.end);
        // A pending mark is part of what the toolbar should be showing: the user
        // pressed bold and has not typed yet, and the button has to look pressed
        // or the press looks like it did nothing.
        const marks = new Set(at.marks);
        for (const key of Object.keys(el.__richPending || {})) marks.add(richMarkCommand(key));

        const block = ((doc.b || [])[richLocate(doc, where.start).block] || {}).k || "p";
        const payload = JSON.stringify({
            s: where.start, e: where.end,
            marks: [...marks], link: at.link, block,
        });
        if (el.dataset.richSelection === payload) return;
        el.dataset.richSelection = payload;
        window.GoInvokeCallback(cbId, { value: payload });
    }

    // applyRichTextProps is the editor's whole prop surface, on both the create
    // and the update path. See applyCodeEditorProps for why `created` exists.
    function applyRichTextProps(el, props, created = false) {
        if (!props) return;

        if ("readOnly" in props) {
            const readOnly = !!props.readOnly;
            el.dataset.readOnly = readOnly ? "true" : "false";
            // contenteditable, not `disabled`: a read-only document is still
            // content the reader is meant to select and copy, which is the whole
            // difference between read-only and disabled. A non-editable
            // contenteditable is also out of the tab order by itself, which is
            // what a page of read-only notes wants.
            el.setAttribute("contenteditable", readOnly ? "false" : "true");
            el.setAttribute("aria-readonly", readOnly ? "true" : "false");
        }
        if ("placeholder" in props) {
            el.dataset.placeholder = String(props.placeholder ?? "");
        }
        if ("onChange" in props) el.dataset.listener_onChange = props.onChange;
        if ("onSelectionChange" in props) {
            el.dataset.listener_onSelectionChange = props.onSelectionChange;
        }
        if ("doc" in props) applyRichDoc(el, String(props.doc ?? "{}"));
        if ("editorEpoch" in props) {
            const changed = String(el.dataset.editorEpoch) !== String(props.editorEpoch);
            el.dataset.editorEpoch = props.editorEpoch;
            if (changed && !created && Number(props.editorEpoch) !== 0) {
                runRichCommand(el, String(props.editorCommand ?? ""));
            }
        }
        syncRichPlaceholder(el);
    }

    // The echo guard, over the document's JSON rather than over a string of
    // text. Same bookkeeping, same three arms: an echo is dropped, a rewrite
    // lands even mid-typing, and a blurred editor is Go's outright.
    //
    // The comparison is on the JSON string and not on the parsed value, which is
    // what makes it exact and cheap: Go marshals the document with a fixed key
    // order, so the same document is always the same bytes.
    function applyRichDoc(el, json) {
        const echoes = el.__richEchoes || (el.__richEchoes = []);
        const focused = document.activeElement === el;
        if (focused) {
            const echo = echoes.indexOf(json);
            if (echo >= 0) {
                echoes.splice(0, echo + 1);
                return;
            }
        }
        echoes.length = 0;
        let doc;
        try {
            doc = JSON.parse(json);
        } catch (err) {
            // A doc prop that is not a document leaves the editor as it was,
            // rather than emptying a note because one patch was malformed.
            return;
        }
        if (JSON.stringify(el.__richDoc) === json) return;
        el.__richDoc = doc;
        richTextToDOM(el, doc);
        // A rewrite from Go replaced the text the remembered offsets described,
        // so they describe nothing now. Dropped rather than clamped: a command
        // issued after a rewrite should act on wherever the user next puts the
        // caret, not on a position in a document that no longer exists.
        el.__richSelection = null;
        el.__richUndo = [];
        el.__richRedo = [];
    }

    // The placeholder, which a contenteditable has no native spelling for.
    //
    // A real element rather than a ::before rule, because this runtime writes no
    // stylesheet — every rule it applies is an inline style on an element it
    // made — and a pseudo-element cannot be set inline. It is marked chrome and
    // is skipped by the serializer like every other element the runtime draws,
    // so it can never become part of the document.
    function syncRichPlaceholder(el) {
        const prompt = el.dataset.placeholder || "";
        const empty = richDocText(el.__richDoc || { b: [] }).trim() === "";
        let node = null;
        for (const child of el.children) {
            if (child.dataset.grmobChrome === "richplaceholder") node = child;
        }
        if (!prompt || !empty) {
            if (node) node.remove();
            return;
        }
        if (!node) {
            node = document.createElement("span");
            node.dataset.grmobChrome = "richplaceholder";
            // Out of the document's flow and out of the pointer's way, so a tap
            // on it lands in the editor behind it.
            Object.assign(node.style, {
                position: "absolute",
                pointerEvents: "none",
                opacity: "0.45",
            });
            node.setAttribute("contenteditable", "false");
            el.insertBefore(node, el.children[0] || null);
        }
        if (node.textContent !== prompt) node.textContent = prompt;
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
            const target = focusTargetOf(el);
            if (!target) return;
            if (action === "focus") {
                target.focus();
            } else if (document.activeElement === target) {
                target.blur();
            }
        });
    }

    // The element a focus command actually lands on.
    //
    // For every field this is the node's own element — an <input>, a
    // <textarea> — and the resolution is the identity. A CodeEditor is the
    // exception: the node is a <pre> acting as the scroll box, and the thing a
    // browser can put a caret in is the transparent <textarea> this runtime
    // built inside it. Calling focus() on the <pre> would do nothing at all
    // (an element with no tabindex is not focusable), so a dismiss would leave
    // the keyboard up and a core.Focus would be silently lost.
    //
    // A RichTextEditor needs no resolution: the node's own element carries
    // contenteditable, so it *is* the focusable thing, and document.active
    // Element reports it as such. The gutter is never a candidate either way —
    // it is aria-hidden chrome, and nothing but the buffer is looked for here.
    //
    // Returning null rather than falling back to `el` is deliberate: a
    // CodeEditor whose chrome has not been built yet has nowhere to put a
    // caret, and focusing its <pre> instead would move focus off whatever
    // legitimately holds it. A command that misses is the honest outcome.
    function focusTargetOf(el) {
        if (el.dataset.nodeType === "CodeEditor") {
            return codeChrome(el, "codebuffer");
        }
        return el;
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
            Box: "column",
            SafeArea: "column",
            List: "column",
        }[nodeType] || "";
    }

    // Which node types are stacks — containers that lay their children out
    // along an axis whether or not the Style asks — and the axis each uses.
    // Read from styleFromGrMob alone on this target: that function is total
    // and createElement now runs it for every node, Style or no Style, so the
    // stacking default is planted and restated by one call site. htmlout still
    // consults its copy in two places (renderNode plants it, styleValue
    // restates it) because a static export has no patch path to be total for.
    //
    // Go's copy is stackAxes in htmlout/stack.go, which carries the reasoning
    // — including why Modal and Spacer are absent, and what TabView's row does
    // and does not fix (the axis, not the missing tab bar). The two are
    // compared by TestRuntimeStackAxesMatchGo, so keep the flat-literal shape.
    //
    // Fragment and Theme are this runtime's own rows: it boxes both in real
    // divs to keep positional patch addressing valid (see tagForType), and a
    // box that is not a stack would swallow its parent's layout the way any
    // other block-flow div does. htmlout emits no element for them at all, so
    // its table has no such rows — the one exemption the conformance test
    // makes, narrowed to these two types.
    function stackAxisFor(nodeType) {
        return {
            Row: "row",
            Column: "column",
            Card: "column",
            Box: "column",
            Scroll: "column",
            SafeArea: "column",
            List: "column",
            TabView: "column",
            // A sized void that can hold things: core.Spacer builds no
            // children, but a hand-assembled node can, and both natives now
            // stack them. Without this row they would run together on one
            // line here — see htmlout's stackAxes for the whole argument.
            Spacer: "column",
            Fragment: "column",
            Theme: "column",
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
        // The ternary is load-bearing in a way it did not used to be, and the
        // change made this target agree with the other one.
        //
        // core.Style's fields are `,omitzero` (see the note above that struct),
        // so an all-zero EdgeInsets is now absent from the payload rather than
        // present as an object of zeros. Every other line in this function was
        // already blind to that distinction — `style.FontSize ? ... : ""`
        // answers "" for a zero as well as for a missing key — but an object
        // full of zeros is truthy in JavaScript, so these two lines were the
        // only ones that could tell "the author said nothing" from "the author
        // said zero", and they answered "0px 0px 0px 0px" to both.
        //
        // htmlout has always omitted the declaration in that case (see
        // export.go, `if s.Padding != (core.EdgeInsets{})`), so the two web
        // targets disagreed: static HTML let a <button> keep the UA's own
        // padding and this runtime forced it to zero. They agree now. Do not
        // "fix" this by defaulting to an empty object — that would restore the
        // divergence in the other direction.
        out.padding = style.Padding ? edgeToCSS(style.Padding) : "";
        out.margin = style.Margin ? edgeToCSS(style.Margin) : "";
        out.borderRadius = style.BorderRadius ? `${style.BorderRadius}px` : "";
        // Rotation. Assigned unconditionally like everything else here: a
        // compass whose heading passes through 0 sends Rotate: 0 in the patch,
        // and a guarded write would leave the last angle standing on the live
        // element — the dial would stick one frame short of north and never
        // return. htmlout can omit the declaration instead because it builds a
        // fresh string per export and has no element to leave stale.
        out.transform = style.Rotate ? `rotate(${style.Rotate}deg)` : "";
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
        // "column" last: a node type outside the stack table that sets Gap
        // still needs an axis to space along, and vertical is what both DOM
        // targets have always used there.
        const dir = style.FlexDirection || stackAxisFor(nodeType) || "column";
        let alignItems = style.AlignItems || "";
        if (!alignItems && dir.startsWith("column") && alignFallbackAxisFor(nodeType)) {
            alignItems = crossAxisAlignFor(style.Align || "");
        }
        // Stack containers are flex whether or not this Style asks for it,
        // and this is the only place on this target that says so: createElement
        // calls applyStyle for a styleless node too, so the default is planted
        // here at build time and restated here on every update-style patch —
        // which the totality rule above requires, since a patch that dropped
        // the display would otherwise leave a Column in block flow. htmlout's
        // styleValue reads the same table for the same reason.
        // RowGap/ColumnGap promote a box exactly as Gap does — `gap` IS the
        // two of them, so a node setting one has asked for the same spacing
        // by another name. Only ever the deciding term for a node type
        // outside the stack table, since a stack container is promoted by the
        // table's own term anyway; htmlout's styleValue makes the same call
        // for every type.
        //
        // An overlay is a container and not a flex one, so it takes the branch
        // before the flex test rather than beside it: a ZStack that carried a
        // Gap or an AlignItems would otherwise be promoted to a flex container
        // and stop overlaying its children altogether — a silent, total loss
        // of the thing the node type exists for. htmlout's styleValue orders
        // the same two branches the same way.
        const overlay = OVERLAY_TYPES.has(nodeType);
        if (overlay) {
            // "inline-grid" is the same translation htmlout makes, and for
            // the same reason the flex path writes inline-flex: an
            // inline-level node that is also a grid needs both halves and
            // `display` has one slot.
            out.display = style.Display === "inline" ? "inline-grid" : "grid";
            out.flexDirection = "";
        } else if (style.Gap || style.RowGap || style.ColumnGap || style.JustifyContent ||
            alignItems || style.FlexDirection || stackAxisFor(nodeType)) {
            out.display = "flex";
            out.flexDirection = dir;
        } else {
            out.display = "";
            out.flexDirection = "";
        }
        // Gap is resolved into the two axis longhands rather than written as
        // the `gap` shorthand, because this function is *total*: it restates
        // every property it manages on every pass so an update-style patch
        // clears whatever the new Style dropped. `gap` IS row-gap plus
        // column-gap, so assigning the shorthand here and then the longhands
        // below — to "" whenever RowGap/ColumnGap are unset, which is almost
        // always — erased the gap that had just been set, and every
        // core.Gap() in every app silently rendered as no spacing at all.
        // Writing only the longhands says the same thing with no shorthand
        // left to clobber, and keeps the axis values winning over the
        // isotropic one, which is the order htmlout's declaration list
        // already produces via the CSS cascade.
        const rowGap = style.RowGap || style.Gap;
        const columnGap = style.ColumnGap || style.Gap;
        out.rowGap = rowGap ? `${rowGap}px` : "";
        out.columnGap = columnGap ? `${columnGap}px` : "";
        out.justifyContent = style.JustifyContent || "";
        // Centre on both axes is core.ZStack's whole alignment contract — the
        // one arrangement a SwiftUI ZStack, a Compose Box and a grid cell all
        // agree on. On a grid these are the *items* properties: justifyContent
        // would place the single track inside the container, which on an
        // auto-sized container does nothing at all.
        out.alignItems = overlay ? "center" : (alignItems || "");
        // Written on every pass like everything else here, so a node that
        // stops being an overlay (only ever via a replace, but totality is not
        // a case analysis) does not keep the centring.
        out.justifyItems = overlay ? "center" : "";
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
        // Not on an overlay: the branch above already translated an inline
        // display into inline-grid, which hugs on its own, and a fit-content
        // width on top of it would be a second answer to one question.
        if (!overlay && !style.Width && (style.Display === "inline" || style.Display === "inline-block")) {
            out.width = "fit-content";
        }
        // A flex *item* property: how this node behaves inside its parent's
        // layout, so it needs no display:flex of its own.
        out.flexGrow = style.FlexGrow ? `${style.FlexGrow}` : "";
        // The false arm is "none", not "", for the node types the browser draws
        // a border on unasked: clearing the inline declaration hands the
        // element back to the user-agent stylesheet, which is the bug rather
        // than the fix. It stays "" everywhere else, so totality is unaffected
        // — every element still gets exactly one of the three values on every
        // call. See BORDER_RESET_TYPES above.
        out.border = (style.BorderWidth && style.BorderColor)
            ? `${style.BorderWidth}px solid ${style.BorderColor}`
            : (BORDER_RESET_TYPES.has(nodeType) ? "none" : "");
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
        // fixed-pitch; this pins the margin a <pre> carries by default, a
        // line height the rows are sized against, and sideways scrolling for
        // a grid wider than the screen. Each row keeps one line even when it
        // has no runs, so the rows below it stay on the cell grid.
        //
        // The white-space rules are three levels deep and each says something
        // different: `normal` on the grid because the only white space
        // *between* rows is markup formatting, `nowrap` on the row because a
        // terminal row or a code line is one line and must not break between
        // two runs, and `pre` on each run (applyGridRuns) because a run's own
        // spaces are the one kind of white space in a grid that is content.
        // Same rules as htmlout's textGridChassis / gridRowChassis /
        // gridRunStyle — this runtime has no formatter that could disturb a
        // grid, and carries the rules so that it does not *differ* from the
        // exporter that does.
        if (nodeType === "TextGrid") {
            out.margin = out.margin || "0";
            out.lineHeight = out.lineHeight || "1.2";
            out.whiteSpace = out.whiteSpace || "normal";
            out.overflowX = out.overflow ? "" : "auto";
        }
        if (nodeType === "GridRow") {
            out.minHeight = out.minHeight || "1.2em";
            out.whiteSpace = out.whiteSpace || "nowrap";
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
        // The editor chassis (core.CodeEditor): the grid's rules plus the two
        // things an overlay needs. Written *after* the placement group above
        // rather than beside the TextGrid/GridRow chassis, and that placement
        // is load-bearing: `out.position` is assigned unconditionally up there
        // from style.Position, so a relative set earlier would be wiped by the
        // author's empty one. position:relative makes the box the containing
        // block for the gutter and the transparent textarea — without it both
        // escape to the nearest positioned ancestor, which is some screen — and
        // overflow:auto lets a buffer wider or taller than its frame scroll in
        // both directions rather than spill, which is what "no wrapping" costs
        // and the reason a code editor scrolls sideways where prose does not.
        //
        // Same declarations as htmlout's codeEditorChassis. The gutter's own
        // padding is not here: it is a function of the line count, so
        // syncCodeGutter writes it after every batch.
        if (nodeType === "CodeEditor") {
            out.margin = out.margin || "0";
            out.lineHeight = out.lineHeight || "1.2";
            out.whiteSpace = out.whiteSpace || "normal";
            out.overflow = out.overflow || "auto";
            out.position = out.position || "relative";
        }
        // The prose editor's chassis, which is the opposite of the code one:
        // a document wraps. position:relative is for the placeholder, which is
        // an absolutely positioned element rather than a ::before rule because
        // this runtime writes no stylesheet. htmlout states the same line height
        // (richTextChassis) and needs neither of the other two, having no
        // placeholder element and no caret.
        if (nodeType === "RichTextEditor") {
            out.lineHeight = out.lineHeight || "1.5";
            out.whiteSpace = out.whiteSpace || "normal";
            out.position = out.position || "relative";
            out.outline = out.outline || "none";
        }
        // Flex container properties that are deliberately NOT part of the
        // display:flex decision above. Unlike Gap/JustifyContent/AlignItems,
        // none of these does anything on its own — flex-wrap and the axis gaps
        // only have meaning once the box is already a flex container — so
        // promoting a box for them alone would change its layout to no
        // purpose.
        out.flexWrap = style.FlexWrap || "";
        // RowGap/ColumnGap are written with Gap above, where the three are
        // reconciled; only flex-wrap is left of that original group here.
        // Flex *item* properties, joining flexGrow above.
        out.alignSelf = style.AlignSelf || "";
        out.flexBasis = style.FlexBasis || "";
        // core.ShrinkNone (-1) is how the Go side spells a shrink factor of
        // ZERO, because every other number in a core.Style means "unset" by
        // being zero and flex-shrink is the one whose CSS initial value is not
        // zero. A truthiness test alone — which is what this was — turns "do
        // not shrink" into no declaration at all, which is the opposite
        // instruction. See core.ShrinkNone and Style.ShrinkFactor.
        out.flexShrink = style.FlexShrink === -1
            ? "0"
            : style.FlexShrink ? `${style.FlexShrink}` : "";

        // The Modal overlay chassis, on exactly the same terms as the grid's
        // and for the same two reasons: it is the fixed look of a node *type*,
        // and a chassis set only at creation would be wiped by the first
        // update-style patch, since every property above is reassigned on
        // every one. htmlout states the identical declarations in
        // modalChassis, ahead of the author's style so the author still wins
        // — which is what the `||` on each line here says.
        //
        // core.ModalNode carries no Style at all, so on every tree core builds
        // this is the whole of a modal's look; a hand-assembled node that does
        // carry one is the case the `||` is for. The z-index of 1000 sits
        // under the toast layer's 2000, so a toast confirming a dialog's
        // action is not drawn behind the dialog.
        if (nodeType === "Modal") {
            out.position = out.position || "fixed";
            out.top = out.top || "0";
            out.left = out.left || "0";
            out.right = out.right || "0";
            out.bottom = out.bottom || "0";
            // flex, not block: the overlay centres its content.
            out.flexDirection = out.flexDirection || "column";
            out.alignItems = out.alignItems || "center";
            out.justifyContent = out.justifyContent || "center";
            out.zIndex = out.zIndex || "1000";
            // The one exemption this function makes to its own totality rule,
            // and it is deliberate: a Modal's `display` IS its open/closed
            // state. It is written by the `visible` prop — at creation, and by
            // the prop patch that opens or closes the dialog — and this
            // function never sees a prop. Assigning anything here would close
            // an open modal on the next update-style patch; assigning ""
            // would open a closed one. Deleting the key is how the pass
            // abstains: Object.assign leaves an absent property alone, so the
            // prop channel keeps sole ownership of it.
            //
            // # What a second exemption would have to satisfy
            //
            // Three conditions, and the second is the one that is easy to
            // miss. wasm/verify/totality_test.mjs states them as a table and
            // checks each; the table is also what makes adding a `delete`
            // here a failing change until a row is written for it.
            //
            //  1. Some *prop* owns the property. Totality is not being
            //     dropped, it is being handed over — there has to be someone
            //     to hand it to, and it has to be a prop, because a Style
            //     field would have been assigned by this function.
            //
            //  2. That owner writes the property in EVERY state, not just the
            //     interesting one. `visible ? "flex" : "none"` qualifies; a
            //     prop that assigns only when it is truthy does not, because
            //     the value it wrote last would then stand forever — which is
            //     the exact failure totality exists to prevent, moved one
            //     channel over rather than fixed.
            //
            //  3. The exemption is keyed on the node type, so the property
            //     stays total for every other node. This one is inside the
            //     `nodeType === "Modal"` block for that reason.
            //
            // htmlout has no such split — it writes the whole declaration list
            // once from the props it can see, so its chassis carries the
            // display and this one does not.
            delete out.display;
        }
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
    // "Set explicitly" means non-zero, so a hand-built {Horizontal: 16,
    // Left: 0} cannot ask for a zero left inset. Both natives are lossy in
    // exactly the same way for the same reason (a Go zero value carries no
    // "was it set?" bit), and matching them is the point. Go states the rule
    // in htmlout/edges.go; the two are independent statements of one contract,
    // each with its own tests, the same arrangement the conformance replay
    // makes for the prop table.
    //
    // The DSL's per-side props are not subject to it: core.PaddingLeft and its
    // siblings dissolve the shorthand into the sides before writing their own,
    // so what arrives here is already stated per-side. Nothing changes on this
    // side of the wire — that is the property that made the fix a Style
    // transformation rather than a fifth copy of the resolution rule.
    // Two decimal places, the precision a CSS length measured in device pixels
    // is meaningful at. Go's copy is round2 in htmlout/export.go.
    function round2(v) {
        return Math.round(v * 100) / 100;
    }

    // The four-value CSS shorthand from one core.EdgeInsets, resolving the
    // Horizontal/Vertical pair the way htmlout.EdgeCSS and both natives do.
    //
    // All six fields are `json:",omitzero"` in Go, so most insets arrive here
    // with only the sides that were set. That needs no handling beyond what is
    // already written: `explicit || 0` reads undefined and 0 identically, which
    // is correct because the resolution rule itself defines a zero side as
    // "unset, take the axis". An absent key and a zero one have never been
    // different questions here, and Go no longer sends the second one.
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

            // The picker (core.Select). Its <option> elements are built from
            // the options prop by applySelectOptions, not from child nodes,
            // so they carry no data-node-path and no patch is addressed to
            // one.
            Select: "select",

            // Told apart from each other by inputTypeFor, below — except for
            // Checkbox and Switch, which inputTypeFor cannot tell apart at all
            // (both are type="checkbox") and which differ by the `switch`
            // attribute createElement writes for one of them.
            Input: "input",
            InputPassword: "input",
            NumericInput: "input",
            Checkbox: "input",
            Switch: "input",
            Slider: "input",

            // A monospace grid and its rows (core.TextGrid); see
            // applyGridRuns for the spans inside a row.
            TextGrid: "pre",
            GridRow: "div",

            // The programmer's editor (core.CodeEditor). The same <pre> a
            // TextGrid is, because its rows *are* a grid's rows — core builds
            // them with the same gridRowNode — with the gutter and the
            // transparent <textarea> added inside it as chrome. See the
            // CodeEditor section above.
            CodeEditor: "pre",

            // The prose editor (core.RichTextEditor). A <div>, made
            // contenteditable, holding the document's own block elements as
            // chrome — there is no element that means "a document", and htmlout
            // answers the same way for the same reason.
            RichTextEditor: "div",

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

            // The live map and its pins (core.MapView). The div is what gets
            // handed to Leaflet; a Marker is data rather than a box and gets an
            // element because patch paths are positional. See the map section.
            MapView: "div",
            Marker: "div",

            // The z-stack. A div like the rest — what makes it an overlay is
            // the single-cell grid styleFromGrMob gives it and the grid-area
            // syncOverlay stamps on its children, not the element.
            ZStack: "div",

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

    // The Go node type -> the <input> type attribute. Only the types
    // tagForType collapses onto <input> appear here; every other node has a
    // tag that already says what it is, and gets no type attribute (which is
    // why the fallback below is "" and not an error).
    //
    // Two keys share one value: a Switch is an <input type="checkbox"> with
    // the `switch` attribute on it, because HTML has no switch element. This
    // table stops one attribute short of separating them, and createElement
    // finishes the job — see the attribute it writes there, and core.Switch
    // for why a switch is a node type rather than a flag on a checkbox.
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
            Switch: "checkbox",
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

    // A picker's option list, decomposed into the menu a person sees — the
    // fourth transliteration of core.SelectMenuSections (core/select_menu.go),
    // beside GrMobSelectMenu.swift and GrMobSelectMenu.kt.
    //
    // # It used to not exist, and that used to be defensible
    //
    // Appending to the DOM has no closing step: a run ended when the next
    // option stopped being appended to the open <optgroup>, so this target
    // alone had no end-of-loop flush to forget — which is exactly what
    // htmlout, writing markup, did forget for a release. Building the runs
    // first was work with nothing bought.
    //
    // core.SelectOption.GroupDisabled is what ended that. A run's disabled
    // state is stated by *any* of its options, so it is not known until the
    // run is closed — and the <optgroup> it is written on has to be created
    // when the run is *opened*. No amount of appending gets past a rule that
    // reads forward, so this target now decomposes like the other three, and
    // is checked like them: wasm/verify drives every case from the same
    // generated table ios/verify uses (menuCases, internal/menufixture) and
    // rebuilds the sections back out of the DOM to compare.
    //
    // The shape is core's, transliterated: an item's `disabled` is already
    // resolved here — a disabled run marks every option in it, because two of
    // the four targets have no section-level control — so a caller of this
    // reads `section.disabled` only for the heading.
    function selectMenuSections(options) {
        const sections = [];
        // The run being filled and the heading it was opened with. `building`
        // is separate from the heading because the empty heading is a
        // legitimate one and would otherwise be indistinguishable from
        // "nothing open yet".
        let heading = "";
        let items = [];
        let runDisabled = false;
        let building = false;

        // Closing a run is two steps: the flush, and the walk back over the
        // items collected before the run's state was known.
        const closeRun = () => {
            if (runDisabled) for (const it of items) it.disabled = true;
            sections.push({ heading, disabled: runDisabled, items });
        };

        options.forEach((o, i) => {
            const group = o.group ?? "";
            if (!building || group !== heading) {
                if (building) closeRun();
                heading = group;
                items = [];
                runDisabled = false;
                building = true;
            }
            if (o.groupDisabled === "true") runDisabled = true;
            const value = o.value ?? "";
            // The label defaulted to the value at core.Select's flattening
            // seam, so this fallback is for a hand-assembled node that never
            // went through it — a menu row with no text is worse than one
            // showing its value.
            const label = o.label ?? "";
            items.push({
                index: i,
                value,
                label: label === "" ? value : label,
                // The wire carries the string "true", core.SelectedState's
                // spelling one property over. Anything else is not disabled.
                disabled: o.disabled === "true",
            });
        });
        if (building) closeRun();
        return sections;
    }

    // A picker's options (core.Select). The list is a prop rather than child
    // nodes, so the <option> elements are this runtime's to build — the same
    // arrangement a TabView's bar has, one step simpler because a Select has
    // no node children for the chrome to be counted past.
    //
    // # Rebuilt only when the list itself changed
    //
    // buildTabBar's reason, and a sharper version of it: rebuilding the
    // options on every props patch would close an open drop-down mid-choice,
    // because replacing the <option> elements resets the control. The
    // signature is the list as JSON — cheap for the handful of entries a
    // picker holds, and exact, where comparing lengths would miss a relabel.
    //
    // The value is assigned on every call regardless, because it is the half
    // that changes on every selection. Assigning it *after* the options is
    // required rather than tidy: a <select> silently ignores a value that
    // matches none of its current options, so setting it before they exist
    // leaves the picker showing its first entry.
    function applySelectOptions(el, options, value) {
        if (el.tagName.toLowerCase() !== "select") return;
        const list = Array.isArray(options) ? options : [];
        const signature = JSON.stringify(list);
        if (el.dataset.selectOptions !== signature) {
            el.dataset.selectOptions = signature;
            el.innerHTML = "";
            for (const section of selectMenuSections(list)) {
                // The ungrouped run has no wrapper: its options are children
                // of the <select> itself, which is where every option lived
                // before core.SelectOption.Group existed. htmlout makes the
                // same branch on the same condition.
                let parent = el;
                if (section.heading !== "") {
                    parent = document.createElement("optgroup");
                    // Chrome like the options themselves, below.
                    parent.dataset.grmobChrome = "optgroup";
                    // setAttribute, not textContent: an <optgroup>'s label is
                    // an attribute, and htmlout writes it through element's
                    // attribute path for the same reason.
                    parent.setAttribute("label", section.heading);
                    // One attribute for the whole run, which is the thing this
                    // target can do and the two natives cannot — it greys the
                    // heading as well as refusing the options. The per-option
                    // `disabled` below is still written, because core marks
                    // every item of a disabled run for the renderers that have
                    // no section-level control at all.
                    if (section.disabled) parent.disabled = true;
                    el.appendChild(parent);
                }
                for (const item of section.items) {
                    const opt = document.createElement("option");
                    // Chrome, like a TabView's bar: an element the runtime
                    // draws that no node asked for. It carries no
                    // data-node-path, no patch is ever addressed to it, and
                    // the marker is what keeps the conformance replay from
                    // comparing it against a Go node that does not exist.
                    // Unlike the bar it is not counted by chromeOffset —
                    // nothing needs to be, since a Select has no node children
                    // for an option to sit ahead of.
                    opt.dataset.grmobChrome = "option";
                    opt.setAttribute("value", item.value);
                    // A disabled option is still drawn and still announced —
                    // that is what disabling one buys over leaving it out — it
                    // simply cannot be chosen.
                    if (item.disabled) opt.disabled = true;
                    // textContent, not innerHTML: an option's label is content
                    // and is as user-originated as anything else here. htmlout
                    // escapes the same string through element's TE.
                    opt.textContent = item.label;
                    parent.appendChild(opt);
                }
            }
        }
        if (value !== undefined && el.value !== value) {
            el.value = value;
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
        // A <select>'s value is its chosen option's value, read off the
        // element exactly as a text field's is — which is why it joins this
        // list rather than needing an arm of its own. Go registered the
        // handler through the text callback channel (core.Select takes a
        // func(string)), so the string envelope is the right one.
        if (["input", "textarea", "numericinput", "inputpassword", "slider", "select"].includes(goType)) {
            return { value: e.target.value };
        }
        // Both boolean controls report their .checked, not their .value — an
        // <input type="checkbox">'s value is the string "on" whatever the box
        // is doing. Go registered both handlers through the bool callback
        // channel (core.Checkbox and core.Switch each take a func(bool)), so
        // the boolean envelope is the right one for both.
        if (goType === "checkbox" || goType === "switch") {
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
        // The initial render has no patch batch, so the composite pass runs
        // over the whole tree once. After the append, because the pass reads
        // document.activeElement and writes a tab stop, and both are questions
        // about an element that is in the document.
        syncCompositesIn(root, new Set(), new Set());
    }

    function patch(patchList) {
        const patches = typeof patchList === "string" ? JSON.parse(patchList) : patchList;

        // Every element this batch reached, plus its parent — the input to the
        // TabView pass at the end. The parent is collected too because a
        // "remove" detaches its element before the pass runs, and a detached
        // node has no ancestors left to walk.
        const touched = [];

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
                // The slot index is a *node* index; the DOM child list may
                // carry chrome ahead of the node children (a TabView's bar), so
                // it is shifted past that chrome. Without the shift a page
                // added to a TabView would land in front of the bar, and every
                // page after it would be one slot out of step with the path it
                // answers to.
                parent.insertBefore(added, parent.children[index + chromeOffset(parent)] || null);
                touched.push(parent);
                return;
            }

            const el = document.querySelector(`[data-node-path="${p.TargetID}"]`);
            if (!el) {
                return;
            }
            touched.push(el);
            if (el.parentNode) touched.push(el.parentNode);

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
                    // The map nodes' dataset, before the per-key loop and for
                    // the same reason the hint is: the sync pass reads lat, lng
                    // and zoom together, and the patch carries the whole new
                    // map. The Leaflet reconciliation itself runs once per batch
                    // in syncTouchedMaps, not here — a batch can carry a marker
                    // move and the map's own region, and doing the work per key
                    // would re-read the layer several times for one change.
                    applyMapProps(el, p.Changes, el.dataset.nodeType);
                    // The editor's own props, before the per-key loop and for
                    // the same reason the hint and the map's dataset are: they
                    // are read together (the command's epoch and its string,
                    // the gutter's flag and the line count) and the patch
                    // carries the whole new map, so deciding them per key
                    // would depend on the order Object.entries happened to
                    // yield them in. The loop below skips every key this
                    // consumed.
                    if (el.dataset.nodeType === "CodeEditor") {
                        buildCodeEditor(el);
                        applyCodeEditorProps(el, p.Changes);
                    }
                    if (el.dataset.nodeType === "RichTextEditor") {
                        buildRichTextEditor(el);
                        applyRichTextProps(el, p.Changes);
                    }
                    for (const [k, v] of Object.entries(p.Changes)) {
                        if (el.dataset.nodeType === "CodeEditor" && CODE_EDITOR_PROPS.has(k)) {
                            continue;
                        }
                        if (el.dataset.nodeType === "RichTextEditor" && RICH_EDITOR_PROPS.has(k)) {
                            continue;
                        }
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
                            el.__grmobRuns = v;
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
                        } else if (k === "onTabChange" && el.dataset.nodeType === "TabView") {
                            // Before the generic on* branch, exactly as on the
                            // create path: "tabchange" is not a DOM event. The
                            // bar's buttons read this ID back at click time,
                            // and pruneStaleListeners drops it — leaving the
                            // bar inert — when a pass stops carrying it.
                            el.dataset.listener_onTabChange = v;
                        } else if (k === "selectedIndex" && el.dataset.nodeType === "TabView") {
                            // This IS the tab-switch path: core.SelectedIndex
                            // is controlled state, so a switch reaches the page
                            // as a prop patch on the TabView and never as a
                            // subtree replacement — the pages are never rebuilt,
                            // only re-hidden. syncTouchedTabViews acts on it
                            // once the batch has landed.
                            el.dataset.tabSelected = v;
                        } else if (k === "onLongPress") {
                            attachLongPress(el, v);
                        } else if (k === "onEndReached") {
                            // Before the generic on* branch, as on the create
                            // path. Only the ID is refreshed here; whether the
                            // observation still points at the right row is
                            // syncTouchedEndReached's question, once the whole
                            // batch has landed and the children have settled.
                            attachEndReached(el, v);
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
                    // After the per-key loop, beside applyEnterKeyHint's
                    // reason for being before it: the bar is a function of
                    // tabs, selectedIndex and onTabChange together.
                    if (el.dataset.nodeType === "TabView") {
                        buildTabBar(el, p.Changes);
                    }
                    // The picker's options and value, likewise together. This
                    // IS the selection path: core.Select is controlled, so a
                    // choice reaches the element as a props patch carrying the
                    // new value and the same list — which the signature check
                    // in applySelectOptions turns into a value assignment and
                    // no rebuild.
                    if (el.dataset.nodeType === "Select") {
                        applySelectOptions(el, p.Changes.options, p.Changes.value);
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
                    // The element collected above is detached now; the one
                    // that took its place is what the TabView pass has to be
                    // able to walk up from.
                    touched.push(newEl);
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
                    // The new child's *node* index — see the "add" case above.
                    // A TabView with a bar would otherwise name its first page
                    // "…/1" and leave nothing answering to "…/0".
                    const index = nodeChildCount(el);
                    const newChild = renderNode(p.Changes, `${p.TargetID}/${index}`);
                    el.appendChild(newChild);
                    break;
            }
        });

        syncTouchedTabViews(touched);
        syncTouchedOverlays(touched);
        // After the tab pass, and this ordering is load-bearing: syncTabView
        // is what writes aria-selected onto the bar's buttons, and the tab
        // stop follows the selection. Running first would move the stop from
        // the state the batch replaced.
        syncTouchedComposites(touched);
        // After the tab pass, and after every add-child/remove has landed:
        // the observation target is the list's last child, and this batch is
        // exactly what may have replaced it.
        syncTouchedEndReached(touched);
        // After every structural patch for the same reason the map pass is:
        // an editor re-decides the stale-line rule against the rows this batch
        // added, changed or removed, and draws a gutter sized to how many
        // there now are.
        syncTouchedCodeEditors(touched);
        // Last, and after every structural patch for the same reason: a map's
        // Leaflet layer is reconciled against the Marker children this batch
        // added, moved or removed.
        syncTouchedMaps(touched);
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

    // The layer lands in the app's own box when the host gives the app a
    // positioned one, and falls back to the document otherwise.
    //
    // A toast is the app's chrome, so it belongs over the app's screen. On a
    // page where the app fills the window the two are the same rectangle and
    // this changes nothing — but a host that frames the app in a smaller box
    // (the tutorial site draws it inside a phone bezel) had its toasts
    // stretched across the whole browser window, detached from the app that
    // raised them. The box is only usable as an anchor if it establishes a
    // containing block, hence the `static` test: an unpositioned parent would
    // send an absolute child to the initial containing block, which is worse
    // than the fixed-to-viewport behavior it replaced.
    //
    // Sibling of the app root rather than child: mount() clears the mount
    // point's innerHTML, so a layer inside it would be silently detached by
    // the next RenderInitial and every toast after that would go nowhere.
    // The typeof guard keeps this working on a DOM without the full CSSOM —
    // wasm/verify's shim has no getComputedStyle — where the answer is simply
    // the document, exactly as it was before the app box became an option.
    function toastLayerHost() {
        const parent = rootElement && rootElement.parentElement;
        if (parent && parent !== document.body &&
            typeof getComputedStyle === "function" &&
            getComputedStyle(parent).position !== "static") {
            return { parent, position: "absolute" };
        }
        return { parent: document.body, position: "fixed" };
    }

    function ensureToastLayer() {
        // isConnected, not a plain null test: a re-mount can take the layer
        // out of the document, and a detached node would swallow every toast.
        if (toastLayer && toastLayer.isConnected) return toastLayer;
        const host = toastLayerHost();
        toastLayer = document.createElement("div");
        Object.assign(toastLayer.style, {
            position: host.position,
            bottom: "24px",
            left: 0, right: 0,
            display: "flex",
            flexDirection: "column",
            alignItems: "center",
            rowGap: "8px",
            zIndex: 2000,
            pointerEvents: "none",
        });
        host.parent.appendChild(toastLayer);
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

    // The browser half of core's heading sensor (core/heading.go). Answers
    // the "sensor" system event with kind "heading" and reports readings back
    // over GrMobWASM.HostEvent as the "heading" host event.
    //
    // # Three browsers, two events, one usable number
    //
    //   Chrome/Android   deviceorientationabsolute, alpha counter-clockwise
    //                    from north  ->  heading = 360 - alpha
    //   Safari/iOS       deviceorientation, event.webkitCompassHeading is
    //                    already clockwise from magnetic north, plus a
    //                    webkitCompassAccuracy in degrees
    //   desktop          neither fires; see the availability timeout
    //
    // A plain "deviceorientation" event without webkitCompassHeading is NOT
    // used even though it carries an alpha, because on Android that alpha is
    // relative to wherever the device happened to be when the listener
    // attached. It looks exactly like a compass and points somewhere
    // arbitrary, which is worse than reporting no compass at all.
    const heading = (() => {
        let running = false;
        let listener = null;      // the attached handler, for removal
        let eventName = "";
        let firstTimer = 0;       // availability timeout
        let lastSent = 0;         // throttle clock

        // ~15 Hz, matching what the natives throttle to. The sensors fire far
        // faster than that (Safari at 60 Hz), and every event that gets
        // through costs a full Go render pass.
        const MIN_INTERVAL_MS = 66;

        // How long to wait for a first reading before calling the device
        // compass-less. Desktop browsers define DeviceOrientationEvent and
        // simply never fire it, so a feature check cannot tell them apart
        // from a phone whose first event is still in flight; the only
        // difference is that the phone's arrives. Two seconds is long enough
        // for a cold magnetometer and short enough that a UI waiting on the
        // answer does not look hung.
        const AVAILABILITY_MS = 2000;

        function report(payload) {
            const host = window.GrMobWASM;
            if (!host || typeof host.HostEvent !== "function") return;
            host.HostEvent("heading", JSON.stringify(payload));
        }

        function unavailable(message) {
            report({ available: false, error: message });
        }

        function onEvent(e) {
            let magnetic = null;
            let accuracy;
            if (typeof e.webkitCompassHeading === "number" && !isNaN(e.webkitCompassHeading)) {
                // Safari: already the bearing core wants.
                magnetic = e.webkitCompassHeading;
                if (typeof e.webkitCompassAccuracy === "number" && e.webkitCompassAccuracy >= 0) {
                    accuracy = e.webkitCompassAccuracy;
                }
            } else if (e.absolute === true && typeof e.alpha === "number" && e.alpha !== null) {
                // The spec's alpha counts counter-clockwise from north, so the
                // clockwise bearing is its complement. Go normalises the 360
                // case back to 0.
                magnetic = 360 - e.alpha;
            }
            if (magnetic === null) return;

            // Clear the availability timer on the first real reading: the
            // device has answered, so the "no compass" verdict must not fire
            // behind it.
            if (firstTimer) { clearTimeout(firstTimer); firstTimer = 0; }

            const now = Date.now();
            if (now - lastSent < MIN_INTERVAL_MS) return;
            lastSent = now;

            const payload = { magnetic, ts: now };
            if (accuracy !== undefined) payload.accuracy = accuracy;
            report(payload);
        }

        function attach() {
            // deviceorientationabsolute is the one that means north on
            // Android; Safari does not implement it and answers the plain
            // event with webkitCompassHeading instead. Both are attached
            // through the same handler, which reads whichever fields it finds.
            eventName = ("ondeviceorientationabsolute" in window)
                ? "deviceorientationabsolute"
                : "deviceorientation";
            listener = onEvent;
            window.addEventListener(eventName, listener, true);
            firstTimer = setTimeout(() => {
                firstTimer = 0;
                unavailable("no compass on this device");
            }, AVAILABILITY_MS);
        }

        function start() {
            if (running) return;
            if (typeof window === "undefined" || typeof window.addEventListener !== "function"
                || typeof DeviceOrientationEvent === "undefined") {
                unavailable("device orientation is not supported");
                return;
            }
            running = true;
            lastSent = 0;

            // iOS 13+ gates orientation behind a prompt that only resolves
            // from inside a user gesture. core.StartHeading is the call that
            // makes it, per the contract on that function: there is no
            // separate permission API to forget to call.
            //
            // When the request is refused — or when it was made outside a
            // gesture, which rejects rather than prompting — the reason goes
            // back as an unavailable reading. That is what lets an app draw a
            // "tap to enable the compass" button and start again from inside
            // the tap, which is the only way to recover.
            const req = DeviceOrientationEvent.requestPermission;
            if (typeof req === "function") {
                req.call(DeviceOrientationEvent).then((state) => {
                    if (!running) return; // stopped while the prompt was up
                    if (state === "granted") { attach(); return; }
                    unavailable("motion access was not granted");
                }).catch((err) => {
                    if (!running) return;
                    unavailable(String((err && err.message) || err ||
                        "motion access must be requested from a user gesture"));
                });
                return;
            }
            attach();
        }

        function stop() {
            if (!running) return;
            running = false;
            if (firstTimer) { clearTimeout(firstTimer); firstTimer = 0; }
            if (listener) {
                window.removeEventListener(eventName, listener, true);
                listener = null;
            }
        }

        // The "sensor" system event's dispatcher. Unknown commands and kinds
        // are dropped, matching every host's contract for unknown events —
        // core's next sensor (location) adds a kind here without this file
        // needing to know about it in advance.
        function handle(cmd) {
            if (cmd.kind !== "heading") return;
            switch (cmd.command) {
                case "start": start(); break;
                case "stop": stop(); break;
            }
        }

        return { handle };
    })();

    // The browser half of Go's permission package. Answers the "permission"
    // system event and reports back over GrMobWASM.HostEvent as the
    // "permission" host event.
    //
    // # A browser cannot be asked, only used
    //
    // This is the host where the two commands are genuinely different
    // operations, and where one of them mostly cannot be honoured.
    //
    //   check     navigator.permissions.query({name}) — a real read, no UI,
    //             answering "granted" | "denied" | "prompt" in exactly the
    //             words Go's Status carries
    //   request   there is no such API. A page obtains camera, microphone or
    //             location by *calling the feature* — getUserMedia,
    //             geolocation.getCurrentPosition — and the browser puts the
    //             prompt up as a side effect of that call.
    //
    // So a request here does the smallest thing that actually prompts, and
    // hands back whatever came of it. For the media devices that is a
    // getUserMedia whose tracks are stopped the instant it resolves: the
    // prompt is the point, the stream is not, and a live track left running
    // is a recording indicator the user did not ask for. For geolocation it
    // is a single getCurrentPosition.
    //
    //   permission     request becomes
    //   ------------   ---------------------------------------------------
    //   camera         getUserMedia({video:true}), tracks stopped
    //   microphone     getUserMedia({audio:true}), tracks stopped
    //   location       geolocation.getCurrentPosition, result discarded
    //   storage        unavailable — a page reaches files through an <input>
    //                  or the file-system access API, both of which are a
    //                  gesture rather than a permission, so there is nothing
    //                  to ask for and nothing to read back
    //
    // # The query is what keeps the device shut
    //
    // The obvious version of the table above reaches for the device every
    // time, and it opened the camera for a moment on a page that already had
    // the camera permission — the recording indicator lighting up to answer a
    // question the browser had already written down. So every request reads
    // first:
    //
    //   query says granted   report it. Nothing is opened; the browser has
    //                        already recorded the answer this request would
    //                        have produced.
    //   query says denied    report it. A getUserMedia here rejects
    //                        immediately with no UI, so the call buys a
    //                        NotAllowedError and nothing else.
    //   query says prompt    reach for the device. This is the one state
    //                        where a request has something to do, and opening
    //                        the camera *is* the prompt.
    //   query cannot answer  reach for the device, as this always did. A
    //                        browser with no descriptor for this kind (Firefox
    //                        has no "camera") can only be asked by asking.
    //
    // The short-circuit on "denied" is the browser's own live answer, not
    // bookkeeping of this file's — which is the difference between it and the
    // Android shell, where a locally remembered refusal deliberately does not
    // short-circuit anything (Permissions.kt, "Auto-reset").
    //
    // # And what tells a Block from a dismissal
    //
    // A NotAllowedError does not say which happened, and the two need
    // different words: a dismissed prompt can be asked again, a Block cannot
    // and wants the user sent to the site settings. The Permissions API knows
    // — a Block is recorded as "denied", a dismissal leaves the state at
    // "prompt" — so a refusal is resolved by reading it back rather than
    // guessed at. Where there is no descriptor to read there is still no way
    // to tell, and the answer falls back to "denied", which is the direction
    // whose remedy is harmless to offer.
    const permission = (() => {
        // Go's Permission constants, mapped to the Permissions API descriptor
        // name where one exists. A permission with no descriptor is not
        // queryable and is answered "unavailable" — see the storage row above.
        const DESCRIPTORS = {
            camera: "camera",
            microphone: "microphone",
            location: "geolocation",
            storage: null,
        };

        function report(kind, status) {
            const host = window.GrMobWASM;
            if (!host || typeof host.HostEvent !== "function") return;
            host.HostEvent("permission", JSON.stringify({ kind, status }));
        }

        // A query result, or "unavailable" for anything this browser cannot
        // answer. Chrome and Safari disagree about which descriptors exist —
        // Firefox has no "camera" at all — and query() *throws* a TypeError
        // for a name it does not know rather than resolving to a state, so
        // the catch is the common path on some browsers and not an edge case.
        function query(kind) {
            const name = DESCRIPTORS[kind];
            if (!name || !navigator.permissions
                || typeof navigator.permissions.query !== "function") {
                return Promise.resolve("unavailable");
            }
            return navigator.permissions.query({ name })
                .then((s) => s.state)
                .catch(() => "unavailable");
        }

        function check(kind) {
            query(kind).then((status) => report(kind, status));
        }

        // getUserMedia's prompt, with the stream discarded. The tracks are
        // stopped in both settled paths because a resolved promise means a
        // live capture device: leaving it running would keep the browser's
        // recording indicator lit for a page that only wanted an answer.
        function askMedia(kind, constraints) {
            const md = navigator.mediaDevices;
            if (!md || typeof md.getUserMedia !== "function") {
                report(kind, "unavailable");
                return;
            }
            md.getUserMedia(constraints).then((stream) => {
                stream.getTracks().forEach((t) => t.stop());
                report(kind, "granted");
            }).catch((err) => {
                // NotAllowedError is a refusal; NotFoundError and
                // OverconstrainedError mean the device is not there at all,
                // which is Unavailable rather than Denied — the distinction
                // Go's two statuses exist to draw, since only one of them has
                // a fix in the browser's settings.
                const name = (err && err.name) || "";
                if (name === "NotFoundError" || name === "OverconstrainedError"
                    || name === "NotReadableError") {
                    report(kind, "unavailable");
                    return;
                }
                refused(kind);
            });
        }

        // Reports a refusal as the browser recorded it.
        //
        // The feature call cannot tell a Block from a dismissed prompt, and
        // the Permissions API can: see "And what tells a Block from a
        // dismissal" above. Only "prompt" upgrades the answer. A query that
        // says "granted" straight after a rejected request is a browser
        // contradicting itself, and reporting the grant would hand the app a
        // camera that had just refused it — so everything except "prompt"
        // stays denied.
        function refused(kind) {
            query(kind).then((state) => {
                report(kind, state === "prompt" ? "prompt" : "denied");
            });
        }

        function askLocation() {
            if (!navigator.geolocation
                || typeof navigator.geolocation.getCurrentPosition !== "function") {
                report("location", "unavailable");
                return;
            }
            navigator.geolocation.getCurrentPosition(
                () => report("location", "granted"),
                (err) => {
                    // PERMISSION_DENIED is 1; POSITION_UNAVAILABLE and TIMEOUT
                    // are failures of the *fix* rather than of the permission,
                    // so they fall back to a query — the page may well be
                    // authorised and simply indoors.
                    //
                    // The refusal reads the permission back too, for the
                    // different reason refused() exists: a dismissed prompt
                    // and a blocked origin arrive through the same code 1.
                    if (err && err.code === 1) { refused("location"); return; }
                    check("location");
                },
                { timeout: 10000, maximumAge: Infinity },
            );
        }

        // The feature call for one kind — the part of a request that actually
        // puts a prompt on screen, and the part that opens a device to do it.
        // Reached only when the read above could not answer; see "The query is
        // what keeps the device shut".
        function prompt(kind) {
            switch (kind) {
                case "camera": askMedia("camera", { video: true }); break;
                case "microphone": askMedia("microphone", { audio: true }); break;
                case "location": askLocation(); break;
            }
        }

        function request(kind) {
            // Nothing to ask for, and nothing to read: answered here rather
            // than sent through the query, which has no descriptor for it.
            // Answered rather than dropped, so a screen waiting on it stops
            // waiting.
            if (kind === "storage") {
                report("storage", "unavailable");
                return;
            }
            query(kind).then((state) => {
                if (state === "granted" || state === "denied") {
                    report(kind, state);
                    return;
                }
                prompt(kind);
            });
        }

        // The "permission" system event's dispatcher. An unknown kind or
        // command is dropped, matching every host's contract for unknown
        // events.
        function handle(cmd) {
            if (!Object.prototype.hasOwnProperty.call(DESCRIPTORS, cmd.kind)) return;
            switch (cmd.command) {
                case "check": check(cmd.kind); break;
                case "request": request(cmd.kind); break;
            }
        }

        return { handle };
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
        heading,
        permission,
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
    if (name === "sensor") {
        // core's sensor plumbing (core/heading.go): start/stop for one named
        // kind. Today the only kind is "heading".
        GrMob.heading.handle(JSON.parse(payloadJSON));
        return;
    }
    if (name === "permission") {
        // Go's permission package: check/request for one named capability.
        GrMob.permission.handle(JSON.parse(payloadJSON));
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
