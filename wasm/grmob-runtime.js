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

    // The two structural pairs, container role to member role. Everything in
    // this section is driven by this table and nothing else knows the words.
    //
    // Kept to the two roles that have a keyboard pattern *and* a container
    // that owns its children. `list`/`listitem` is content rather than a
    // control and has no pattern; `menu`, `tree` and `grid` are patterns core
    // has no roles for yet.
    const COMPOSITE_MEMBERS = { listbox: "option", tablist: "tab" };

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

    // The composite a member belongs to, or null. Walks out rather than
    // searching, so a member nested three containers deep finds the same
    // answer compositeMembers reached it from.
    //
    // The match is on the member's *own* role rather than on "the nearest
    // composite of any kind", and the difference is the one thing that keeps
    // this the inverse of compositeMembers. That walk descends through a
    // composite of the other kind — a tablist inside a listbox is not a
    // listbox's member and does not close it — so an option below one is still
    // the listbox's member, and a walk out that stopped at the tablist would
    // disagree with the walk in. Nobody writes that tree on purpose; the two
    // functions still have to answer the same question the same way.
    function compositeOf(member) {
        const role = member.getAttribute("role");
        for (let el = member.parentNode; el && el.getAttribute; el = el.parentNode) {
            if (compositeMemberRole(el) === role) return el;
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
        const memberRole = compositeMemberRole(container);
        if (!memberRole) return;
        const members = compositeMembers(container, memberRole);
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

    // Moves the tab stop to one member and puts focus on it.
    //
    // Focus and the stop move together, always: they are two statements of one
    // fact, and a browser that focused a member holding tabindex="-1" would
    // put the next Tab back at the top of the document.
    function moveCompositeFocus(members, index) {
        members.forEach((member, i) => {
            member.setAttribute("tabindex", i === index ? "0" : "-1");
        });
        members[index].focus();
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
                moveCompositeFocus(members, idx);
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
        const members = compositeMembers(container, compositeMemberRole(container));
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
        moveCompositeFocus(members, to);
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
        if (compositeMemberRole(el) && !done.has(el)) {
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
                if (compositeMemberRole(el) && !done.has(el)) {
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
        const css = styleFromGrMob(style, nodeType);
        Object.assign(el.style, css);
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
        // A Modal's semantics come from its node type, not from a Style:
        // core.ModalNode has no Style field, so there is nothing for a role to
        // ride on. Hidden still wins over both — an overlay pruned from the
        // accessibility tree has no element for role="dialog" to describe, and
        // aria-modal would claim the document behind it is inert.
        const dialog = nodeType === "Modal" && !hidden;
        const role = hidden ? "" : (dialog
            ? (style.AccessibilityRole || "dialog")
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
    // A Modal never reaches here — the caller answers the dialog case first —
    // which is why there is no equivalent of htmlout's CarriesOwnRole guard.
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
        out.flexShrink = style.FlexShrink ? `${style.FlexShrink}` : "";

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
            // The open <optgroup>, if any, and the label it was opened with.
            // core.SelectOption.Group makes *consecutive* options with the
            // same heading one group, in the order they were written — see
            // that field for why a gather would silently reorder the list —
            // so the loop closes a run by simply stopping appending to it.
            //
            // core.SelectMenuSections is the authority for that split, and the
            // three other renderers each follow it as a run of sections. This
            // one is the exception, on purpose: a DOM append has no closing
            // step. A run ends when the next option stops being appended to
            // the element, so there is no end-of-loop flush here to forget —
            // which is exactly what htmlout, writing markup, did forget.
            let group = null;
            let openLabel = "";
            for (const o of list) {
                const label = o.group ?? "";
                if (label !== openLabel) {
                    openLabel = label;
                    group = null;
                    if (label !== "") {
                        group = document.createElement("optgroup");
                        // Chrome like the options themselves, below.
                        group.dataset.grmobChrome = "optgroup";
                        // setAttribute, not textContent: an <optgroup>'s label
                        // is an attribute, and htmlout writes it through
                        // element's attribute path for the same reason.
                        group.setAttribute("label", label);
                        el.appendChild(group);
                    }
                }
                const opt = document.createElement("option");
                // Chrome, like a TabView's bar: an element the runtime draws
                // that no node asked for. It carries no data-node-path, no
                // patch is ever addressed to it, and the marker is what keeps
                // the conformance replay from comparing it against a Go node
                // that does not exist. Unlike the bar it is not counted by
                // chromeOffset — nothing needs to be, since a Select has no
                // node children for an option to sit ahead of.
                opt.dataset.grmobChrome = "option";
                opt.setAttribute("value", o.value ?? "");
                // A disabled option is still drawn and still announced — that
                // is what disabling one buys over leaving it out — it simply
                // cannot be chosen. The wire carries the string "true", the
                // spelling core.SelectedState uses one property over.
                if (o.disabled === "true") {
                    opt.disabled = true;
                }
                // textContent, not innerHTML: an option's label is content and
                // is as user-originated as anything else here. htmlout escapes
                // the same string through element's TE.
                opt.textContent = o.label ?? o.value ?? "";
                (group ?? el).appendChild(opt);
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
                    // The new child's *node* index, which is the DOM child
                    // count minus whatever chrome sits ahead of the pages —
                    // see the "add" case above. A TabView with a bar would
                    // otherwise name its first page "…/1" and leave nothing
                    // answering to "…/0".
                    const index = el.children.length - chromeOffset(el);
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
