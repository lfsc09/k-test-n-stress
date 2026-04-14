# Refactor: Complete and Correct the Concurrency Implementation in the `mock` Command

> Created: 2026-04-13 15:10

## Summary

The `mock` command has a partial concurrency implementation. The worker pool fires for the root-level `--generate` split, but the inner `[n]` key split (depth > 0) is a `TODO` that falls back to the root split unconditionally. Before completing the inner split, two bugs in `mocker/mocker.go` must be fixed: `Payment.creditCardCvv` previously called the package-level `regen.Generate` (now already using `m.cvvGen` per memories — see verification note in Step 1), and `Person.cpf` used the global `rand.Intn`. The memories confirm both were fixed in the `202604101540_feature_mock_concurrency` plan. This plan therefore focuses on (1) verifying and documenting the current mocker state, (2) implementing the true inner `[n]` split in `analyzeTemplate` and `runWorkerPool`, (3) confirming the sequential path, output streaming, atomic writes, signal handling, and `--debug` flag are complete and correct, (4) adding race-detector tests for the new inner-split code path, and (5) benchmarking `deepcopy` cost. The plan proceeds in the order: mocker verification, inner split implementation, sequential path correctness check, streaming/output audit, race-detector coverage, deepcopy benchmark.

---

## Affected Files

- `mocker/mocker.go` — verify and, if needed, fix any remaining global-rand calls; confirm `m.cvvGen` and `m.rng` are used exclusively
- `mocker/helpers_test.go` — add race-detector compatible tests for `Person.cpf` and `Payment.creditCardCvv` called concurrently
- `cmd/mock.go` — implement inner split in `analyzeTemplate` and `runWorkerPool`; audit sequential path, streaming, atomic writes, signal handling, debug display
- `cmd/mock_test.go` — add unit tests for `analyzeTemplate` inner-split logic, `walkBracketCounts`, and `runWorkerPool` inner-split dispatch
- `cmd/mock_e2e_test.go` — add E2E tests exercising inner `[n]` key templates above `concurrencyThreshold` to ensure correct output shape
- `Makefile` — confirm `test-race` target exists and covers `./...`; add to CI documentation if needed

---

## Steps

### Step 1: Verify mocker.go global-rand status (mocker/mocker.go) [x]

Read `mocker/mocker.go` lines 377–421 (the `Payment.creditCardCvv` and `Person.cpf` cases) and confirm:

- `Payment.creditCardCvv`: uses `m.cvvGen.Generate()` (per-instance `regen.Generator` created in `New()` backed by `m.rng`). The defensive `m.cvvGen == nil` re-init branch also uses `m.rng`. No call to package-level `regen.Generate`.
- `Person.cpf`: uses `m.rng.Intn(10)` — NOT `rand.Intn(10)`.
- `Company.cnpj`: uses `m.rng.Intn(10)` — NOT `rand.Intn(10)`.

The memories state both were fixed. This step is a read-only audit. If any global `rand.Intn` or package-level `regen.Generate` call is found in the hot paths, fix it before proceeding — replace with the corresponding `m.rng` call or `m.cvvGen` method.

**Expected outcome**: No changes needed (confirmed by memories). If changes are required, they are 1-line substitutions.

**Files touched**: `mocker/mocker.go` (read-only; edit only if a discrepancy is found).

---

### Step 2: Add concurrent mocker tests to confirm race-detector cleanliness (mocker/helpers_test.go) [x]

In `mocker/helpers_test.go`, add a new test method to `MockerUtilsInternalTestSuite` named `TestMockerConcurrency_NoConcurrentStateSharing`. The test should:

1. Launch `runtime.NumCPU() * 2` goroutines, each calling `mocker.New()` independently.
2. Each goroutine calls `m.Generate("Person.cpf", nil)` and `m.Generate("Payment.creditCardCvv", nil)` 500 times in a loop.
3. All goroutines are started together with a `sync.WaitGroup`.
4. Assert no goroutine returns an error (use a shared error channel buffered to `goroutine_count`).
5. The test has no output-shape assertions — it exists solely to be run under `go test -race ./...` without data-race reports.

**Expected outcome**: Test passes with and without `-race`. Under `-race`, no data-race warnings because all randomness flows through per-instance `m.rng`.

**Files touched**: `mocker/helpers_test.go`.

---

### Step 3: Implement inner-split support in `analyzeTemplate` (cmd/mock.go, lines 90–169) [x]

Currently, `analyzeTemplate` finds `splitNode` at the shallowest qualifying depth but then discards it (the `_ = splitNode` line at 159) and always returns `Depth: 0`. The fix is to return the actual inner split when `splitDepth > 0`.

**Changes to `analyzeTemplate`** (lines 142–169):

Replace the `TODO(concurrency)` block (lines 151–168) with the following logic:

1. When `splitDepth > 0` and a qualifying `splitNode` exists:
   a. Compute `innerWeight` as the product of all `[n]` counts in `nodes` that are **below** the split depth (i.e., `n.depth > splitDepth`), not the split node itself. If none exist below, `innerWeight = 1`.
   b. Compute `totalWeight = int64(generate) * int64(splitNode.count) * innerWeight`.
   c. Return:
      ```
      SplitPoint{
          Depth:         splitDepth,
          KeyPath:       splitNode.keyPath,
          Count:         splitNode.count,
          InnerWeight:   innerWeight,
          TotalWeight:   totalWeight,
          UseSequential: totalWeight < concurrencyThreshold,
      }
      ```
2. Keep the existing root-split fallback path unchanged for when `splitDepth == -1`.

**Note on sibling keys**: When multiple `[n]` keys exist at `splitDepth`, the brainstorm requires sequential dispatch through the same warm pool. For the initial inner-split delivery, handle only the **first qualifying sibling** (i.e., the node already selected as `splitNode`). The full sibling-sequential dispatch is a follow-on enhancement (see Step 4). Document this limitation with a `// TODO(concurrency-siblings)` comment.

**Also fix `analyzeTemplate`'s `innerWeight` computation for the `splitDepth == -1` fallback** (lines 125–140): the current code multiplies all `[n]` nodes — this is correct only if there is no split node. Confirm the formula is `product of all n.count values in nodes`.

**Expected outcome**: `analyzeTemplate` now returns `Depth > 0` and a correct `KeyPath` for templates with inner `[n]` keys where `count >= numWorkers`.

**Files touched**: `cmd/mock.go`.

---

### Step 4: Add unit tests for `analyzeTemplate` inner-split logic (cmd/mock_test.go) [x]

Add a new test method `TestAnalyzeTemplate_InnerSplit` to `MockCmdTestSuite`. Cover:

1. Template with one `[n]` key at depth 1 where `count >= numWorkers` (e.g., `employees[20]` with `numWorkers = 4`): assert `Depth == 1`, `KeyPath == "employees[20]"`, `Count == 20`, `InnerWeight == 1`.
2. Template with a `[n]` key at depth 1 (`employees[20]`) that itself contains a nested `[n]` key at depth 2 (`phones[5]`): assert `Depth == 1`, `InnerWeight == 5`, `TotalWeight == generate * 20 * 5`.
3. Template with a root `[n]` key where count < `numWorkers` (e.g., `company[2]` with `numWorkers = 8`): assert `Depth == 0` (falls back to root), `Count == generate`.
4. Template with no `[n]` keys: assert `Depth == 0`, `Count == generate`, `InnerWeight == 1`.
5. `UseSequential == true` when `totalWeight < concurrencyThreshold`, `false` when at or above.

Note: `analyzeTemplate`, `walkBracketCounts`, `SplitPoint`, and related types are package-private within `cmd`, so these tests live in `package cmd` (the same file as existing unit tests).

**Expected outcome**: All cases pass deterministically with no goroutines.

**Files touched**: `cmd/mock_test.go`.

---

### Step 5: Implement inner-split dispatch in `runWorkerPool` (cmd/mock.go, lines 200–287) [x]

When `splitPoint.Depth > 0`, the worker pool must handle a fundamentally different work unit: instead of generating complete root objects, each worker generates a slice of the **inner array** identified by `splitPoint.KeyPath`.

**Changes to `runWorkerPool`**:

1. Add a branch at the top of `runWorkerPool`: `if splitPoint.Depth == 0 { /* existing root-split path */ }`.

2. **Inner split path** (`splitPoint.Depth > 0`):
   a. Extract the **sub-template** at `splitPoint.KeyPath` from the full `template` map. The key is the raw bracket key (e.g., `"employees[20]"`). Deep-copy it once before dispatching.
   b. Process the parent map **once sequentially** on the calling goroutine: call `deepcopy.Copy(template)` → `processJsonMap` on all keys **except** the split key → do not call `sanitizeJsonMap` yet. Store the partially-processed parent as `parentContext map[string]any`.
   c. Determine the clean (sanitized) key name: `sanitizeKeyWithBrackets(splitPoint.KeyPath)` — e.g., `"employees"`.
   d. Dispatch `WorkUnit` batches where each unit covers a range of the inner array indices (`startIdx`..`endIdx` within `0..splitPoint.Count`). The `templateSnapshot` in each unit is a deep copy of the inner sub-template.
   e. Each worker: for each index in the range, deep-copy `templateSnapshot` → `processJsonMap` → `sanitizeJsonMap` → append to sub-batch `[]map[string]any`. Send sub-batch to `results`.
   f. The writer (`streamOutput`) receives sub-batches of the **inner** items — it assembles the final object: `parentContext[cleanKey] = assembledInnerSlice`. For the inner-split case, the writer must accumulate the full inner array before writing the parent object, which means the inner items are buffered within the writer until the inner array is complete.

   **Memory implication**: buffering the inner array in the writer re-introduces O(inner_count × item_size) memory for the duration of writing one root object. For large inner arrays (e.g., `employees[100000]`), this is a concern. Accept this trade-off for correctness in the first implementation, and note it with a `// TODO(concurrency-inner-streaming)` comment for future improvement.

3. The `results` channel type `chan []map[string]any` remains unchanged. Workers send `[]map[string]any` sub-batches regardless of split depth.

4. The signature of `runWorkerPool` remains unchanged. The `splitPoint.Depth` field is the sole discriminant.

**Expected outcome**: `runWorkerPool` routes to the correct dispatch path based on `splitPoint.Depth`. Workers generate only the inner-array items; the parent map context is processed once.

**Files touched**: `cmd/mock.go`.

---

### Step 6: Adapt `streamOutput` to handle inner-split results (cmd/mock.go, lines 1151–1473) [x]

The current `streamOutput` assumes each item it receives from `results` is a **complete root object** ready to write. For the inner split, the writer receives sub-batches of inner array items and must accumulate them, then wrap them in the parent context.

**Changes to `streamOutput`**:

1. Add a `splitDepth int` and `parentContext map[string]any` and `cleanInnerKey string` parameter (or pass a `SplitPoint` value directly — pass `SplitPoint` to avoid parameter sprawl).

2. When `splitPoint.Depth == 0`: behavior is exactly as today — each received item is written immediately.

3. When `splitPoint.Depth > 0`:
   a. Accumulate all received inner-array items into a local `innerSlice []any`.
   b. After `results` is drained (channel closed), assemble the final root object: `parentContext[cleanInnerKey] = innerSlice`, then sanitize `parentContext`, then call `writeItem(parentContext, false)` once.
   c. The `generate` value at the root level is always 1 for the inner split case (the root is processed once), so `writeItem` uses the `generate == 1` bare-object path.

   **Note**: if `generate > 1` with inner `[n]` keys, each root object has its own inner array. For simplicity in this first delivery, restrict the inner split to `generate == 1` only. When `generate > 1` with inner `[n]` keys, fall back to the root split (the root-level `--generate` chunking still parallelizes work). Document with `// TODO(concurrency-generate-gt1-inner-split)`.

4. Update the call site in `runConcurrent` (around line 576) to pass the `SplitPoint` to `streamOutput`.

5. Update the signature of `streamOutput` — add `splitPoint SplitPoint` parameter. Update the single call site accordingly.

**Expected outcome**: For inner-split runs, `streamOutput` produces a single, correctly-structured root object containing the fully-generated inner array.

**Files touched**: `cmd/mock.go`.

---

### Step 7: Add E2E tests for inner-split concurrency path (cmd/mock_e2e_test.go) [x]

Add a new test method `TestCLI_InnerSplit_ConcurrentPath` to `MockCmdE2ETestSuite`. Cover:

1. Template `{"employees[200]": {"name": "{{ Person.name }}"}}` with `--generate 1` and `--to-stdout as-json`. Total weight = 200, above threshold. Assert:
   - Output is valid JSON.
   - Top-level is a bare object (not a single-element array), because `generate == 1`.
   - The `"employees"` key holds an array of exactly 200 objects.
   - Each element has a non-empty `"name"` string field.

2. Same template with `--generate 1` and `--to-stdout as-csv` (CSV of the outer object — shallow serialization of the inner array). Assert output contains a header row and one data row.

3. Template `{"tags[15000]": "{{ UUID.uuidv4 }}"}` with `--generate 1` and `--to-stdout as-json`. Total weight = 15000, above threshold. Assert the `"tags"` key is an array of 15000 UUID strings.

4. Template `{"items[5]": {"id": "{{ UUID.uuidv4 }}"}}` with `--generate 3` and `--to-stdout as-json`. Total weight = 15, below threshold — must use sequential path. Assert output is a JSON array of 3 objects each with an `"items"` array of 5 elements.

These tests use the existing `suite.executeCommand(args...)` helper and `bytes.Buffer` out pattern.

**Expected outcome**: All E2E tests pass. Running with `-race` flag produces no data-race reports.

**Files touched**: `cmd/mock_e2e_test.go`.

---

### Step 8: Audit sequential path for correctness (cmd/mock.go, lines 528–543) [x]

Read the `splitPoint.UseSequential` branch in `runConcurrent`. Confirm:

1. The sequential path builds `parseMaps` slice of length `generate`, deep-copies the template for each iteration, calls `processJsonMap` then `sanitizeJsonMap`, and passes the slice to `routeOutput`.
2. It uses a single `mocker.New()` instance (correct — sequential, no goroutines).
3. There is no double `deepcopy.Copy` being called on the same map (line 537 copies the already-processed `cpParseMap` — confirm this is needed and not a bug; it may be copying an already-mutated map. If `processJsonMap` already mutates in-place, the second `deepcopy.Copy` on line 537 is unnecessary. Remove it if so — it doubles memory and reflection cost for the sequential path).

If the double-copy is unnecessary, remove `parseMaps[i] = deepcopy.Copy(cpParseMap).(map[string]any)` and replace with `parseMaps[i] = cpParseMap`.

**Expected outcome**: Sequential path is correct and memory-efficient. One fewer `deepcopy.Copy` call per iteration if the double-copy is confirmed redundant.

**Files touched**: `cmd/mock.go` (1-line change if double-copy is removed).

---

### Step 9: Audit `streamOutput` JSON comma framing (cmd/mock.go, lines 1250–1295) [x]

Verify the `writeItem` function correctly handles comma framing for the JSON array case. Specifically:

1. When `generate > 1` and writing to stdout JSON: confirm that `stdoutEnc.Encode(item)` (which appends a trailing newline) does not produce invalid JSON inside the `[...]` array. The `Encode` call produces `{...}\n` — inside a manually-framed array, this yields `[{...}\n,{...}\n]` which is technically valid JSON (whitespace is allowed) but not pretty. Confirm the current behavior is acceptable or switch to `json.Marshal` + manual write for the array-item path to avoid the stray newline inside the array brackets.

2. Confirm the `sep` logic: `sep = !firstItem` is set before `writeItem` is called and `firstItem = false` is set after — this is correct ordering.

3. Confirm the closing `]\n` is written in the finalization section regardless of whether any items were written (e.g., `generate == 0` is prevented by validation, but `generate == 1` must emit a bare object without `[...]`).

If the `Encode` trailing-newline issue is deemed a quality problem, switch the stdout JSON array path to use `json.Marshal(item)` + manual `stdoutBuf.Write(b)` for both compact and prettified variants inside the array (compact uses `json.Marshal`, prettified uses `json.MarshalIndent`). The `generate == 1` bare-object path can continue using `Encode`.

**Expected outcome**: JSON array output is valid and clean for both compact and prettified variants.

**Files touched**: `cmd/mock.go` (conditional edit if the Encode trailing-newline is fixed).

---

### Step 10: Verify atomic file writes and signal handling are complete (cmd/mock.go) [x]

Read `atomicFileCreate` (lines 1119–1136) and `streamOutput`'s finalization section (lines 1401–1472). Confirm:

1. All four file write paths (JSON file, CSV file) use `atomicFileCreate` → `commit()` on success / `abort()` on error.
2. `abortAll()` is called in every early-return error path within `streamOutput`, including the `ctx.Done()` check.
3. The `signal.NotifyContext` call in `RunE` (line 431) correctly registers `os.Interrupt` and `syscall.SIGTERM`. The shared `ctx` is passed all the way to `runWorkerPool` and `streamOutput`.
4. Workers check `ctx.Done()` between items (the `select { case <-ctx.Done(): return; default: }` block in the worker goroutine, line 225).

If any of the above is missing or incorrect, add the fix as an inline correction within this step.

**Expected outcome**: No partial files are left on disk after SIGINT, error, or panic in any code path.

**Files touched**: `cmd/mock.go` (read-only audit; edit only if a gap is found).

---

### Step 11: Verify `--debug` flag and `RunStats` are complete (cmd/mock.go, lines 178–337) [x]

Read the `--debug` implementation and confirm:

1. `RunStats` struct fields: static fields set before workers start, `Generated atomic.Int64` incremented by workers (line 236), `BytesWritten atomic.Int64` incremented by `writeItem` (line 1339).
2. `startDebugDisplay` is only called when `debugMode == true` (line 561).
3. The display goroutine writes to `os.Stderr` only (not `out`/stdout).
4. The `stop` function is deferred so it fires a final render on exit, including on error returns.
5. The `--debug` flag is registered with `default false` (line 649).
6. The display loop uses `\r` carriage return for in-place overwrite (lines 307, 310).

If `BytesWritten` is only incremented for the JSON file path (line 1339) and not for the CSV file path, add `stats.BytesWritten.Add(int64(n))` after the CSV file write byte count is available. Note that the `csvFileWriter.Write` call does not directly return byte count — to track bytes, wrap the `csvFileBuf` writes or use a `countingWriter` helper.

For the first delivery, tracking bytes written only for JSON file output is acceptable (mark CSV tracking as `// TODO(debug-csv-bytes)`).

**Expected outcome**: `--debug` flag works correctly; no goroutine leaks; final stats line always printed.

**Files touched**: `cmd/mock.go` (minor edit if CSV byte tracking TODO is added).

---

### Step 12: Benchmark `deepcopy` reflection cost (cmd/mock_test.go or a new benchmark file) [x]

Add a benchmark `BenchmarkDeepCopyTemplate` in `cmd/mock_test.go` (or a new `cmd/mock_bench_test.go`). The benchmark should:

1. Define a representative nested template: one top-level map with 10 string fields and one nested map with 5 string fields.
2. Run `deepcopy.Copy(template).(map[string]any)` in the benchmark loop (`b.N` iterations).
3. Report ns/op.

Run: `go test -bench=BenchmarkDeepCopyTemplate -benchmem ./cmd/`

Record the result in a comment at the top of the benchmark. If ns/op > 5000 at 1M iterations (extrapolated), note the performance concern and add a `// TODO(deepcopy-hand-rolled)` comment pointing to the brainstorm document section on hand-rolled clone.

No code change is required from this step beyond adding the benchmark. The optimization itself is deferred.

**Expected outcome**: Baseline benchmark recorded. Any replacement decision is deferred to a follow-on refactor.

**Files touched**: `cmd/mock_test.go` (or new `cmd/mock_bench_test.go`).

---

### Step 13: Confirm `test-race` Makefile target covers all packages [x]

Read `Makefile`. Confirm `test-race` runs `go test -race ./...` (already present per current file content). No change needed.

If a CI config file exists (`.github/workflows/`, `.gitlab-ci.yml`, etc.), check whether `test-race` is invoked there and add it if absent.

**Expected outcome**: `make test-race` exercises all new concurrent code paths with the race detector.

**Files touched**: `Makefile` (read-only audit; edit CI config only if a gap is found).

---

### Step 14: Apply `## Proposed Memory Updates` below to `.claude/memories.md` [x]

(Always the final step.)

---

## Notes

- **Inner split scope restriction**: Steps 5–6 implement inner split only for `generate == 1`. For `generate > 1` with inner `[n]` keys, the code falls back to root split (existing behavior). This is a deliberate deferral, not a regression.

- **Sibling keys deferred**: Multiple `[n]` keys at the same depth are handled by picking the first qualifying sibling. Full sibling-sequential pool dispatch (as described in the brainstorm) is deferred to a follow-on plan. Mark with `// TODO(concurrency-siblings)`.

- **`writeItem` CSV header in inner-split mode**: When `splitPoint.Depth > 0`, the CSV writer will receive a parent object (not inner items directly). The CSV header derivation from the first item will yield the parent object's top-level keys (e.g., `"employees"` → a JSON array serialized as a string). This is consistent with existing behavior for nested objects in CSV. No special handling needed.

- **`runWorkerPool` inner split and `stats.Generated`**: For the inner split, `stats.Generated` should count inner items (not root objects), to keep the debug progress display meaningful. Workers increment `stats.Generated` per inner item, same as today.

- **`deepcopy.Copy` of sub-template in inner split**: The inner sub-template (the map value at `splitPoint.KeyPath`) is extracted once from the full template and deep-copied per `WorkUnit`. The full template is NOT deep-copied per root object in the inner split path — only the sub-template is. This is the memory-saving benefit of inner splitting.

- **`processJsonMap` on parent in inner split (Step 5b)**: The partial parent processing (all keys except the split key) must skip `[n]` key expansion for the split key itself. The simplest way: temporarily delete `splitPoint.KeyPath` from the copy before calling `processJsonMap`, process, then restore the key slot to be filled by the assembled inner array later.

- **`sanitizeJsonMap` order in inner split**: `sanitizeJsonMap(parentContext)` must be called **after** the inner array is assembled into `parentContext`, so that both the outer key (e.g., `"employees[20]"` → `"employees"`) and inner keys are sanitized in one pass.

- **The `_ = splitNode` line at 159 is removed** when implementing Step 3, because `splitNode` is now used to populate the returned `SplitPoint`.

- **No changes to `processJsonMap` or `sanitizeJsonMap`** — they remain pure per-map operations as the brainstorm requires.

---

## Proposed Memory Updates

Add or update the following entries in `.claude/memories.md`:

### Architecture Decisions (add)

- **Inner split for `[n]` keys is implemented for `generate == 1` only** — when `generate > 1` with inner `[n]` keys, `analyzeTemplate` returns `Depth: 0` (root split) as a fallback. See `TODO(concurrency-generate-gt1-inner-split)` in `cmd/mock.go`.
- **Inner split buffers the inner array in the writer** — `streamOutput` accumulates all inner-array sub-batches before writing the parent object when `splitPoint.Depth > 0`. This is O(inner_count × item_size) memory. See `TODO(concurrency-inner-streaming)` in `cmd/mock.go`.
- **Sibling `[n]` keys at the same depth**: only the first qualifying sibling is used as the split node; the others fall inside the worker's sequential processing. Full sibling-sequential pool dispatch is deferred (`TODO(concurrency-siblings)`).
- **`streamOutput` now accepts a `SplitPoint` parameter** — replaces the previous implicit assumption that all results are complete root objects.

### Current Feature State (update)

| Feature | Status |
| --- | --- |
| Inner split for depth > 0 `[n]` keys (`generate == 1`) | Done (after this plan) |
| Inner split for depth > 0 `[n]` keys (`generate > 1`) | TODO — `TODO(concurrency-generate-gt1-inner-split)` |
| Sibling `[n]` keys sequential pool dispatch | TODO — `TODO(concurrency-siblings)` |

### Plan History (add, newest first)

| Plan | Date | Type |
| --- | --- | --- |
| `202604131510_refactor_mock_concurrency` | 2026-04-13 | refactor |
