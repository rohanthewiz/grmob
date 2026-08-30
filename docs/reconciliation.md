# The GrMob Reconciliation Engine: Architecture and Rationale

## Abstract

In modern declarative UI systems, reconciliation is the bridge between intention and rendering. It determines how a new virtual tree of UI components should translate into efficient, minimal updates to the actual interface. This document explores the inner workings of the GrMob reconciliation engine—its design, its motivations, and its mechanisms—providing a deep technical understanding of how GrMob handles rendering, state, and diffing to ensure efficient and predictable UI updates.

---

## Introduction

User interfaces are not static. They change with state, with time, and with interaction. In a declarative model such as GrMob, the developer expresses what the UI should look like given the current state. The job of the reconciliation engine is to decide what needs to be updated between frames.

Unlike imperative systems where changes are applied directly, GrMob adopts a virtual view model. The application renders a tree of `Node` structures, representing the desired UI. These nodes are compared with the previous tree using a diffing algorithm, producing a set of patches. These patches are then applied to the native layer or renderer.

---

## Goals of the Reconciliation System

1. **Performance**: Avoid full re-renders of the UI when only parts have changed.
2. **Predictability**: Make updates deterministic and debuggable.
3. **Minimal Diffing**: Optimize for speed and memory usage.
4. **Stability**: Preserve the identity of components to maintain local state.

---

## Core Concepts

### Node

The `Node` is the atomic structure representing a UI element. It includes:
- `Type`: The type of component (e.g., "Text", "Button", "Input").
- `Props`: Key-value attributes.
- `Style`: Styling configuration.
- `Children`: Nested nodes.

### Tree

A component renders to a `Node`. A UI is a tree of such nodes.

### State

State is maintained per component using a slot-based allocation system. Each `Context` owns a cursor and a slot array. When a component uses `NewState`, a value is stored in the slot at the cursor, and subsequent renders retrieve it.

### Callback Registration

Callbacks are registered with a deterministic or incremental ID. These IDs are passed down into the rendered `Node` tree as `onClick`, `onChange`, etc. During re-renders, used callbacks are marked, and unused ones are purged.

---

## Reconciliation Algorithm

The reconciliation process is invoked when:
- The user interacts with the UI (e.g., types text).
- A state value is updated.

The algorithm performs the following steps:

1. **Render a new tree**: The application function is invoked, and a new tree is generated.

2. **Compare trees**: `Diff(oldTree, newTree, path)` is invoked.

3. **Compute Patches**:
    - If a node is added or removed, emit `add` or `remove` patches.
    - If a node’s type changes, emit `replace` patch.
    - If props or style change, emit `update-props` or `update-style`.
    - If children have structural differences, recurse with indexed paths.

4. **Return the patch set**: This set is then serialized (e.g., to JSON) and sent to the native renderer.

5. **Update currentTree**: After each render, the `currentTree` in the `RenderManager` is updated with the new tree.


---

## Purging Callbacks

To prevent memory leaks and ensure efficiency:
- During render, all callback IDs are marked as "used".
- After diffing, any callbacks not marked are removed from memory.
- This guarantees that only live, reachable event handlers persist.

---

## Handling Identity

Component identity is preserved by position and structure. If a node stays at the same path and has the same type, its internal state (slots) remains intact. This is similar to how React relies on stable keys in lists.

---

## Conclusion

The reconciliation engine in GrMob is designed to bring the rigor of functional design into the performance constraints of mobile UI development. Through immutability, intelligent diffing, and a structured render flow, it achieves the balance between expressiveness and efficiency.

Future enhancements will include keyed diffing, partial subtree memoization, and asynchronous scheduling—bringing it closer to production-grade engines like React Fiber or Flutter's Element tree.

---

## Appendix

- RenderManager: Orchestrates render cycles.
- Reconciler: Computes diffs.
- Context: Maintains slots, callbacks, and themes.
- Node: Structural representation of UI.
- Patch: Delta applied to native UI.

---

