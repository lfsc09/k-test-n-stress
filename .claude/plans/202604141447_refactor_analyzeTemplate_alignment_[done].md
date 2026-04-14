# Refactor: analyzeTemplate Alignment with Brainstorm

> Created: 2026-04-14 14:47

## Summary

`analyzeTemplate` and its helpers in `cmd/mock.go` were built incrementally and now diverge from the brainstorm in three ways: (1) only the *first* qualifying sibling at `splitDepth` is collected, but the brainstorm requires *all* siblings to be dispatched sequentially through the same pool; (2) the inner split is hard-capped to `splitDepth == 1`, but the brainstorm says "depth > 0" with no depth cap; (3) `innerWeight` computation is copy-pasted in four places and the `generate > 1` fallback branch duplicates the root-split return three more times. This refactor aligns `analyzeTemplate` (and only `analyzeTemplate` and its helpers) with the brainstorm before any further concurrency work is built on top. The worker pool, `streamOutput`, and signal handling are **not** touched here.

---

## Affected Files

- `cmd/mock.go` — all changes are confined to this file: `SplitPoint` struct, `bracketNode` struct, `walkBracketCounts`, `analyzeTemplate`, and two call sites that read `SplitPoint.KeyPath` (`runWorkerPoolInner` and `streamOutput`)

---

## Steps

### Structural changes

- [x] **Step 1 — Extend `SplitPoint` with `KeyPaths []string` and `ParentKeyPath []string`**

  File: `cmd/mock.go`, lines 37–53 (the `SplitPoint` struct definition).

  Replace the single `KeyPath string` field with two new fields:

  ```
  KeyPaths       []string  // all qualifying sibling keys at the split depth (empty for depth-0 splits)
  ParentKeyPath  []string  // dot-separated path segments to the *parent* map that contains the split keys
                           // empty when split depth == 1 (keys are direct children of the root template map)
  ```

  Remove the old `KeyPath string` field entirely.

  Rationale for `ParentKeyPath`: when `splitDepth > 1`, `runWorkerPoolInner` currently does `template[splitPoint.KeyPath]` — a single-level lookup that only works at depth 1. For depth 2+ the caller needs to walk `ParentKeyPath` segments to reach the sub-map that contains the split keys. Storing the path segments in `SplitPoint` avoids re-walking the tree at dispatch time.

  After this step the struct reads:
  ```
  type SplitPoint struct {
      Depth          int
      KeyPaths       []string   // NEW: all siblings at the split depth
      ParentKeyPath  []string   // NEW: path from root to the parent map of the split keys
      Count          int        // total items across all siblings (sum of sibling counts)
      InnerWeight    int64
      TotalWeight    int64
      UseSequential  bool
  }
  ```

  Note: `Count` now means the **sum** of counts across all siblings, not a single node's count. This is the correct denominator for `TotalWeight` and for `runWorkerPool`'s dispatch loop.

- [x] **Step 2 — Extend `bracketNode` with `parentPath []string`**

  File: `cmd/mock.go`, lines 59–64 (the `bracketNode` struct).

  Add one field:
  ```
  parentPath []string  // path segments from root to the map that directly contains this key
                       // empty when depth == 1 (key is a direct child of the root)
  ```

  This field is populated in `walkBracketCounts` (Step 3) and consumed in `analyzeTemplate` (Step 5) to populate `SplitPoint.ParentKeyPath`.

---

### Logic changes in helpers

- [x] **Step 3 — Update `walkBracketCounts` to propagate `parentPath`**

  File: `cmd/mock.go`, lines 68–84 (the `walkBracketCounts` function).

  Change the signature from:
  ```go
  func walkBracketCounts(parseMap map[string]any, depth int) []bracketNode
  ```
  to:
  ```go
  func walkBracketCounts(parseMap map[string]any, depth int, parentPath []string) []bracketNode
  ```

  Inside the function:
  - When appending a `bracketNode`, set `parentPath: parentPath` (copy the slice — use `append([]string(nil), parentPath...)` to avoid aliasing).
  - When recursing into a nested map, pass `append(parentPath, key)` as the new `parentPath`. **Important:** the key passed here is the *raw* bracket key (e.g. `"employees[100]"`). The caller only needs it to navigate back at dispatch time; it will use this exact string as the lookup key in the map at that level.

  Update the single call site that calls `walkBracketCounts` in `analyzeTemplate` (line 91) to pass `nil` as the initial `parentPath`:
  ```go
  nodes := walkBracketCounts(parseMap, 1, nil)
  ```

---

### Simplification: extract `productOfCounts` helpers

- [x] **Step 4 — Extract `productOfCounts` and `productOfCountsBelow` helpers**

  File: `cmd/mock.go` — add two small unexported functions immediately before `analyzeTemplate` (around line 90).

  **`productOfCounts(nodes []bracketNode) int64`**
  Returns the product of `n.count` for every node in `nodes`. Returns `1` for an empty slice.

  **`productOfCountsBelow(nodes []bracketNode, depth int) int64`**
  Returns the product of `n.count` for every node whose `n.depth > depth`. Returns `1` if no such nodes exist.

  These two functions replace the four hand-rolled `innerWeight` loops that currently appear in `analyzeTemplate` (lines 128–130, 157–159, 187–190, 205–208). No caller outside `analyzeTemplate` uses these; keep them unexported.

---

### Simplification: extract `buildRootSplit`

- [x] **Step 5A — Extract `buildRootSplit` helper**

  File: `cmd/mock.go` — add one small unexported function immediately before `analyzeTemplate`.

  **`buildRootSplit(generate int, innerWeight int64) SplitPoint`**

  Returns a `SplitPoint` with:
  - `Depth: 0`
  - `KeyPaths: nil`
  - `ParentKeyPath: nil`
  - `Count: generate`
  - `InnerWeight: innerWeight`
  - `TotalWeight: int64(generate) * innerWeight`
  - `UseSequential: int64(generate)*innerWeight < concurrencyThreshold`

  This collapses the four nearly-identical root-split `return SplitPoint{…}` blocks (lines 96–105, 160–168, 207–217, and the current fallback at the bottom of `analyzeTemplate`) into a single code path.

---

### Logic change: move `generate > 1` check earlier

- [x] **Step 5B — Move the `generate > 1` fallback check immediately after `nodes` is populated**

  File: `cmd/mock.go`, lines 90–218 (the full `analyzeTemplate` body).

  Currently the `generate > 1` check appears at line 155, **after** the split-depth search. This means for every `generate > 1` call the code still sorts `nodes` and scans for a `splitDepth` it will never use.

  Restructure `analyzeTemplate` so the check comes right after the `len(nodes) == 0` guard:

  ```
  1. Call walkBracketCounts → nodes
  2. If len(nodes) == 0 → return buildRootSplit(generate, 1)
  3. If generate > 1 → return buildRootSplit(generate, productOfCounts(nodes))
     // TODO(concurrency-generate-gt1-inner-split) preserved here
  4. Sort nodes by (depth asc, count desc)
  5. Find splitDepth …
  6. If splitDepth == -1 → return buildRootSplit(generate, productOfCounts(nodes))
  7. Collect all siblings + build inner SplitPoint (see Step 6)
  8. If no valid inner split → return buildRootSplit(generate, productOfCounts(nodes))
  ```

  This eliminates the duplicate sort+scan that currently happens for `generate > 1` calls. The `TODO(concurrency-generate-gt1-inner-split)` comment must be kept verbatim at step 3.

---

### Logic change: collect all siblings at `splitDepth`

- [x] **Step 6 — Replace single-sibling selection with all-sibling collection**

  File: `cmd/mock.go`, lines 142–202 (the block that finds and returns the inner `SplitPoint`).

  **Current behaviour (lines 142–149):**
  Picks only the first node at `splitDepth` with `count >= numWorkers` (the `break` exits after the first match).

  **New behaviour:**
  Collect *all* nodes at `splitDepth` where `count >= numWorkers` into a `siblings []bracketNode` slice. No `break`.

  Then:
  - Sum the sibling counts: `totalSiblingCount := sum of s.count for s in siblings`
  - All siblings must be direct children of the same parent map. Verify this by checking that every sibling has the same `parentPath` slice contents. If they differ (should not happen by construction, but guard defensively), fall back to `buildRootSplit`.
  - `innerWeight := productOfCountsBelow(nodes, splitDepth)`
  - `totalWeight := int64(generate) * int64(totalSiblingCount) * innerWeight`

  Validate that every sibling's value in the template is a `map[string]any`. To do this:
  - Navigate to the parent map using `ParentKeyPath`. For `splitDepth == 1`, the parent map is `parseMap` itself (empty `parentPath`). For `splitDepth > 1`, walk `parentPath` segments through the original `parseMap`.
  - For each sibling key, check `parentMap[sibling.keyPath].(map[string]any)`. If any sibling's value is not a `map[string]any`, fall back to `buildRootSplit`.

  If all guards pass, return:
  ```go
  SplitPoint{
      Depth:         splitDepth,
      KeyPaths:      []string{sibling.keyPath for each sibling},
      ParentKeyPath: siblings[0].parentPath,  // same for all siblings
      Count:         totalSiblingCount,
      InnerWeight:   innerWeight,
      TotalWeight:   totalWeight,
      UseSequential: totalWeight < concurrencyThreshold,
  }
  ```

  Remove the now-superseded `splitDepth == 1` guard (previously on line 184). The new code handles depth 1 through N generically.

---

### Update call sites that read `SplitPoint.KeyPath`

- [x] **Step 7 — Update `runWorkerPoolInner` to use `KeyPaths` and `ParentKeyPath`**

  File: `cmd/mock.go`, lines 361–506 (`runWorkerPoolInner` function).

  This is a **scope note, not an implementation change for this refactor.** The body of `runWorkerPoolInner` currently does:
  ```go
  subTemplate, ok := template[splitPoint.KeyPath].(map[string]any)
  ```
  and:
  ```go
  delete(parentCopy, splitPoint.KeyPath)
  ```
  and passes `splitPoint.KeyPath` to the error message strings.

  After removing `KeyPath` from `SplitPoint`, these lines will fail to compile. **For this refactor**, adapt them minimally to use `KeyPaths[0]` (the first sibling) so the existing single-sibling dispatch logic keeps working without regressions, and add a `// TODO(concurrency-siblings)` comment noting that multi-sibling sequential dispatch is deferred. Specifically:

  - Replace `splitPoint.KeyPath` with `splitPoint.KeyPaths[0]` everywhere inside `runWorkerPoolInner`.
  - Add a comment above the `subTemplate` extraction:
    ```
    // TODO(concurrency-siblings): only KeyPaths[0] is dispatched here.
    // Full sibling-sequential pool dispatch is deferred.
    ```
  - The `ParentKeyPath` navigation is not needed for `splitDepth == 1` (the common current case). Add a defensive guard: if `len(splitPoint.ParentKeyPath) > 0`, fall back to root-split behaviour with an error log, since depth > 1 inner dispatch is not yet implemented. This prevents a silent wrong result when `splitDepth > 1` is returned for the first time.

- [x] **Step 8 — Update `streamOutput` to use `KeyPaths` and `ParentKeyPath`**

  File: `cmd/mock.go`, lines 1597–1677 (the `if splitPoint.Depth > 0` branch in `streamOutput`).

  `streamOutput` currently does:
  ```go
  cleanKey := sanitizeKeyWithBrackets(splitPoint.KeyPath)
  ```
  and:
  ```go
  parentCtx[splitPoint.KeyPath] = innerSlice
  ```

  After removing `KeyPath`, replace with `splitPoint.KeyPaths[0]` for both, mirroring the same `TODO(concurrency-siblings)` comment. The single-sibling behaviour is preserved; multi-sibling output routing is deferred.

---

### Verification

- [x] **Step 9 — Confirm compilation and run existing tests**

  After all edits, run:
  ```
  go build ./...
  go test ./...
  ```

  No new tests are required for this refactor because the observable behaviour (sequential path, single-sibling inner split with `generate == 1`, root split) is unchanged. The structural changes only affect the internal representation of `SplitPoint` and the code paths inside `analyzeTemplate`. Any existing E2E test that exercises the concurrent path implicitly validates that `KeyPaths[0]` works correctly in the updated call sites.

  If any test exercises `SplitPoint` fields directly (e.g. by inspecting the struct in a unit test), update those assertions to use `KeyPaths` instead of `KeyPath`.

- [x] **Step 10 — Apply `## Proposed Memory Updates` below to `.claude/memories.md`**

---

## Notes

**`Count` semantics shift:** In the current code `SplitPoint.Count` is always a single node's `[n]` value. After Step 6 it becomes the *sum* of all qualifying sibling counts. Every consumer of `splitPoint.Count` must be audited:
  - `runWorkerPoolRoot`: uses `splitPoint.Count` as the dispatch loop limit. Unaffected — root split still has a single count.
  - `runWorkerPoolInner`: uses `splitPoint.Count` as the inner array dispatch loop limit. With only `KeyPaths[0]` dispatched (per Step 7), this should continue to use `KeyPaths[0]`'s individual count, **not** `splitPoint.Count` (which now sums all siblings). Add a local variable `singleSiblingCount := siblings[0].count` and store it somewhere `runWorkerPoolInner` can consume. The cleanest option: `runWorkerPoolInner` re-derives the count from the sub-template's own `[n]` bracket value by calling `extractDigitInBrackets("object", splitPoint.KeyPaths[0])`. This avoids adding yet another field to `SplitPoint` for this interim state.
  - `runConcurrent` in `RunE`: uses `splitPoint.Count` to clamp `actualWorkers`. After the change, `Count` is `totalSiblingCount`, which is still the correct number to clamp against (total parallelism available), so no change needed there.

**`ParentKeyPath` navigation helper:** Step 6 introduces walking `parentPath` segments to validate sibling values. Extract this as a small unexported helper `navigateToMap(root map[string]any, path []string) (map[string]any, bool)` rather than inlining the loop inside `analyzeTemplate`. This same helper will be needed in `runWorkerPoolInner` (Step 7) when depth > 1 dispatch is eventually implemented.

**No behaviour change for the `generate > 1` path:** Moving the check earlier (Step 5B) does not change what is returned — it only avoids unnecessary sorting. The `TODO(concurrency-generate-gt1-inner-split)` comment is preserved.

**`splitDepth == 1` guard removal:** Removing the `splitDepth == 1` cap in Step 6 means `analyzeTemplate` can now *return* a `SplitPoint` with `Depth > 1`. However, Step 7 adds an explicit defensive guard in `runWorkerPoolInner` that falls back to root-split behaviour for `len(ParentKeyPath) > 0`. This ensures no silent regression while the actual depth > 1 dispatch remains deferred.

**`bracketNode.parentPath` slice aliasing:** `walkBracketCounts` builds `parentPath` via successive `append` calls. Go's append can share underlying arrays. Always use `append([]string(nil), parentPath...)` when storing `parentPath` in a `bracketNode` to ensure each node owns its own slice.

---

## Proposed Memory Updates

Add/update the following in `.claude/memories.md` under **Architecture Decisions**:

- Change the existing bullet about `SplitPoint.KeyPath` and siblings to:
  - **`SplitPoint.KeyPaths []string` (not `KeyPath string`)** — holds all qualifying sibling keys at the split depth. `ParentKeyPath []string` holds the path segments from root to the parent map containing those siblings (empty at depth 1). `Count` is the *sum* of all sibling counts. `runWorkerPoolInner` and `streamOutput` currently consume only `KeyPaths[0]` (single-sibling dispatch); full sibling-sequential pool dispatch remains deferred (`TODO(concurrency-siblings)`).
  - Remove the bullet: "Sibling `[n]` keys at the same depth: only the first qualifying sibling is used as the split node; the others fall inside the worker's sequential processing. Full sibling-sequential pool dispatch is deferred (`TODO(concurrency-siblings)`)."
  - Add: **`analyzeTemplate` supports `splitDepth > 1`** — the `splitDepth == 1` cap was removed in the Refactor 1 plan (2026-04-14). `runWorkerPoolInner` has a defensive guard that returns an error for `len(ParentKeyPath) > 0` until actual depth > 1 dispatch is implemented.
  - Add: **`productOfCounts` / `productOfCountsBelow` / `buildRootSplit` / `navigateToMap`** — four small unexported helpers extracted from `analyzeTemplate` in Refactor 1 (2026-04-14). Do not inline their logic elsewhere.

Update **Plan History**:
```
| `20260414_refactor_analyzeTemplate_alignment` | 2026-04-14 | refactor |
```

Update **Current Feature State** — the "Sibling `[n]` keys sequential pool dispatch" row remains TODO but update its note to reflect that `analyzeTemplate` now *identifies* all siblings; only the *dispatch* side is deferred.
