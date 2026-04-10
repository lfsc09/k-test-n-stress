# Feature: Bounded Worker Pool Concurrency for `mock` Command

> Created: 2026-04-10 15:40

## Summary

The `mock` command currently generates all items sequentially in a single goroutine, making large generation counts (e.g., `--generate 1000000` or large inner `[n]` keys) a bottleneck. This plan adds a bounded worker pool that parallelizes generation across `runtime.NumCPU()` goroutines, keeps peak memory usage constant regardless of total item count via streaming output, adds atomic-file-write safety for file outputs, handles OS signals cleanly, and exposes an optional `--debug` flag for a live progress display on stderr. The plan also fixes two pre-existing global-rand calls in `mocker/mocker.go` that must be corrected before any concurrent code is written.

---

## Affected Files

- `mocker/mocker.go` — fix two global `math/rand` calls (`Person.cpf`, `Company.cnpj`) and one global `regen.Generate()` call (`Payment.creditCardCvv`) to use per-instance randomness; add a per-instance `regen.Generator` factory helper
- `mocker/helpers.go` — no functional changes; possible addition of a `newRegenGenerator(pattern string, rng *rand.Rand)` helper if chosen to centralise goregen wiring
- `mocker/helpers_test.go` — add race-detector-friendly tests for the fixed functions
- `cmd/mock.go` — add `analyzeTemplate`, `runWorkerPool`, streaming `routeOutput` rewrite, `--debug` flag, OS signal handling, atomic file write, `RunStats`, display goroutine
- `cmd/mock_test.go` — add unit tests for `analyzeTemplate` and the new helpers
- `cmd/mock_e2e_test.go` — add E2E tests for concurrency paths (output correctness, `--debug` flag, error propagation)
- `go.mod` / `go.sum` — no new dependencies required; all needed packages (`context`, `sync`, `sync/atomic`, `runtime`, `bufio`, `os/signal`, `syscall`) are from the standard library

---

## Steps

### Phase 0 — Fix Global Rand Calls in `mocker/mocker.go` (Prerequisite)

- [x] **Step 0.1 — Add a per-instance `math/rand.Rand` to `Mock`**

  In `mocker/mocker.go`, change the `Mock` struct from:
  ```go
  type Mock struct {
      jaswdrFaker *faker.Faker
  }
  ```
  to:
  ```go
  type Mock struct {
      jaswdrFaker *faker.Faker
      rng         *rand.Rand   // per-instance source; never share across goroutines
  }
  ```

  Update `New()` to seed it:
  ```go
  func New() *Mock {
      jaswdrFaker := faker.New()
      src := rand.NewSource(time.Now().UnixNano() ^ int64(os.Getpid())) // or crypto/rand for better entropy
      return &Mock{
          jaswdrFaker: &jaswdrFaker,
          rng:         rand.New(src),
      }
  }
  ```

  The import block must change: replace `"math/rand"` (already imported) with a named import if needed to avoid collision with `math/rand/v2` used by faker. Since faker uses `math/rand/v2` internally and mocker uses `math/rand` (v1) for `rand.Intn`, keep using `math/rand` v1 but bind it to the instance. Import line stays `"math/rand"` but remove the package-level bare calls.

  **Note on entropy**: Using `time.Now().UnixNano()` alone can collide when many goroutines call `mocker.New()` in quick succession. XOR with a monotonically-increasing atomic counter (or use `crypto/rand.Read` to seed) for better per-instance uniqueness. Add a package-level `atomic.Int64` seeder counter, e.g.:
  ```go
  var mockerSeedCounter atomic.Int64

  func New() *Mock {
      jaswdrFaker := faker.New()
      seed := time.Now().UnixNano() ^ (mockerSeedCounter.Add(1) * 0x9e3779b97f4a7c15)
      return &Mock{
          jaswdrFaker: &jaswdrFaker,
          rng:         rand.New(rand.NewSource(seed)),
      }
  }
  ```

- [x] **Step 0.2 — Fix `Person.cpf` to use `m.rng` instead of global `rand.Intn`**

  In the `case "Person.cpf":` block (lines 346–367 of `mocker/mocker.go`), replace every `rand.Intn(10)` with `m.rng.Intn(10)`.

- [x] **Step 0.3 — Fix `Company.cnpj` to use `m.rng` instead of global `rand.Intn`**

  In the `case "Company.cnpj":` block (lines 209–231), replace every `rand.Intn(10)` with `m.rng.Intn(10)`.

- [x] **Step 0.4 — Fix `Payment.creditCardCvv` to use a per-instance `regen` generator instead of package-level `regen.Generate`**

  The goregen package-level `Generate()` calls `rand.Int63()` (global math/rand v1 source) to seed a fresh per-call xorShift rng. This is not a data race in Go 1.20+ (global source is mutex-protected) but serializes all callers on the same lock.

  Replace:
  ```go
  case "Payment.creditCardCvv":
      cvv, err := regen.Generate("[0-9]{3}")
  ```
  with a `NewGenerator` call that supplies `m.rng` as the source:
  ```go
  case "Payment.creditCardCvv":
      gen, err := regen.NewGenerator("[0-9]{3}", &regen.GeneratorArgs{RngSource: m.rng})
      if err != nil {
          return "", fmt.Errorf("failed to build CVV generator: %w", err)
      }
      cvv := gen.Generate()
  ```

  Similarly update `Regex.regex` (`case "Regex.regex":`, lines 372–383) to use `regen.NewGenerator` with `m.rng` as the source.

  **Note**: `regen.GeneratorArgs.RngSource` accepts a `rand.Source` (v1). `*rand.Rand` implements `rand.Source` (it has `Int63()` and `Seed()`), so `m.rng` satisfies the interface directly.

- [x] **Step 0.5 — Verify `math/rand` import is still needed and clean up unused imports**

  After replacing all bare `rand.Intn` calls with `m.rng.Intn`, the only remaining use of `"math/rand"` is in `New()` itself (for `rand.NewSource` and `rand.New`). Keep the import.

- [x] **Step 0.6 — Add race-detector tests for the fixed functions in `mocker/helpers_test.go` (or a new `mocker/mocker_race_test.go`)**

  Add a test that spawns 8 goroutines, each with its own `mocker.New()`, and concurrently calls `Generate("Person.cpf", nil)`, `Generate("Company.cnpj", nil)`, `Generate("Payment.creditCardCvv", nil)`, and `Generate("Regex.regex", []string{"/[0-9]{4}/"})`. Assert that all return non-empty strings and no error. Run with `go test -race ./mocker/...` to confirm no races.

---

### Phase 1 — Template Analysis: `analyzeTemplate`

- [x] **Step 1.1 — Define the `SplitPoint` type in `cmd/mock.go`**

  Add the following type immediately after the `var objKeyNumberRegex` declaration:

  ```go
  // SplitPoint describes where the worker pool will chunk generation work.
  type SplitPoint struct {
      // Depth is 0 for a root-level split (--generate count), >0 for an inner [n] key.
      Depth int
      // KeyPath is the dot-separated path to the split key for depth > 0 (e.g. "employees").
      // Empty for depth-0 splits.
      KeyPath string
      // Count is the number of items at the split node (generate count or [n] value).
      Count int
      // InnerWeight is the product of all [n] multipliers below the split point.
      // For a leaf split, innerWeight == 1.
      InnerWeight int64
      // TotalWeight == Count * InnerWeight
      TotalWeight int64
      // UseSequential is true when TotalWeight < concurrencyThreshold; the caller
      // should skip the worker pool and use the existing sequential path.
      UseSequential bool
  }

  const concurrencyThreshold = 10_000 // items below this count go sequential
  const targetSubBatch = 750          // target items per WorkUnit at the split level
  ```

- [x] **Step 1.2 — Implement `analyzeTemplate(parseMap map[string]any, generate int, numWorkers int) SplitPoint`**

  This function walks the template tree once (depth-first) to find the shallowest `[n]` key node where `n >= numWorkers`, and computes the product of all `[n]` multipliers below that node (`innerWeight`).

  Algorithm:
  1. Walk `parseMap` recursively. For each key matching `objKeyNumberRegex`, record `(depth, count)`.
  2. Sort findings by depth ascending; among ties sort by count descending.
  3. Find the shallowest depth `d` where any node has `count >= numWorkers`. That is the split point depth.
     - If no `[n]` key anywhere has count >= numWorkers (e.g., `company[3]` with 6 workers), take the deepest innermost `[n]` key and clamp numWorkers to its count. Flag this as an inner split with clamped workers.
     - If no `[n]` key exists at all, the split is at the root (`generate` count, depth 0).
  4. If the split is at depth 0 (root `--generate`):
     - `count = generate`
     - Walk the full template to compute `innerWeight` = product of all `[n]` counts in the tree.
  5. If the split is at depth > 0:
     - `count` = the `[n]` value at the split key.
     - `innerWeight` = product of all `[n]` counts **below** the split key.
  6. `totalWeight = int64(count) * innerWeight`
  7. If `totalWeight < concurrencyThreshold`, set `UseSequential = true` and return.

  Signature:
  ```go
  func analyzeTemplate(parseMap map[string]any, generate int, numWorkers int) SplitPoint
  ```

  Helper sub-functions (unexported, add to `cmd/mock.go`):
  - `walkBracketCounts(parseMap map[string]any, depth int) []bracketNode` — returns all `[n]` nodes at each depth.
  - `productBelowDepth(parseMap map[string]any, belowDepth int) int64` — returns the product of all `[n]` counts at depths strictly greater than `belowDepth`.

  Type:
  ```go
  type bracketNode struct {
      depth   int
      keyPath string
      count   int
  }
  ```

- [x] **Step 1.3 — Unit tests for `analyzeTemplate` in `cmd/mock_test.go`**

  Add test cases covering:
  - Flat template, generate=1: UseSequential=true (totalWeight=1 < threshold).
  - Flat template, generate=50000: depth-0 split, count=50000, innerWeight=1.
  - Template with one inner `[n]` key, generate=1, n=100000: depth-1 split.
  - Template with nested `[n]` keys at different depths: shallowest qualifying node is chosen.
  - Template with sibling `[n]` keys at the same depth: both qualify; first qualifying depth wins.
  - `[n]` node with count < numWorkers and no deeper qualifying node: clamped sequential.
  - TotalWeight just below threshold (9999): UseSequential=true.
  - TotalWeight at threshold (10000): UseSequential=false.

---

### Phase 2 — Worker Pool Infrastructure in `cmd/mock.go`

- [x] **Step 2.1 — Define the `WorkUnit` type**

  ```go
  // WorkUnit is a single chunk of work sent to a worker goroutine.
  type WorkUnit struct {
      startIdx         int
      endIdx           int
      templateSnapshot map[string]any // deep copy of the template at the split level; never shared
  }
  ```

- [x] **Step 2.2 — Define the `RunStats` struct**

  ```go
  import "sync/atomic"
  import "time"

  // RunStats holds both static configuration and live counters for the --debug display.
  type RunStats struct {
      // Static fields — set once before workers start
      CoresAvailable  int
      CoresUsed       int
      TotalWeight     int64
      WeightPerWorker int64
      EstMemPeakBytes int64
      StartTime       time.Time

      // Live counters — updated atomically by workers/writer
      Generated    atomic.Int64 // incremented per item completed by each worker
      BytesWritten atomic.Int64 // incremented by writer goroutine per write
  }
  ```

  Place this in `cmd/mock.go` alongside `WorkUnit`.

- [x] **Step 2.3 — Implement `runWorkerPool`**

  Signature:
  ```go
  func runWorkerPool(
      ctx context.Context,
      splitPoint SplitPoint,
      template map[string]any,
      generate int,
      numWorkers int,
      results chan<- []map[string]any,
      stats *RunStats,
  ) error
  ```

  Responsibilities:
  1. Compute `subBatchSize = max(1, targetSubBatch / int(splitPoint.InnerWeight))`.
  2. Create a buffered `jobs` channel of `WorkUnit` with buffer size `numWorkers`.
  3. Start `numWorkers` worker goroutines, each with its own `mocker.New()` instance.
  4. Each worker:
     a. Reads `WorkUnit` from `jobs` channel (range over channel — exits when closed).
     b. Allocates `batch := make([]map[string]any, 0, unit.endIdx-unit.startIdx)`.
     c. For each index in `[unit.startIdx, unit.endIdx)`:
        - Check `ctx.Done()` first; if cancelled, drain and return.
        - `cp := deepcopy.Copy(unit.templateSnapshot).(map[string]any)`
        - `processJsonMap(cp, workerMocker)` — return error if any.
        - `sanitizeJsonMap(cp)`
        - `batch = append(batch, cp)`
        - `stats.Generated.Add(1)`
     d. Send `batch` on `results` channel.
     e. On `ctx` cancellation or error, discard remaining work units and call `wg.Done()`.
  5. Main goroutine dispatches `WorkUnit`s in a loop:
     - For each chunk of `[start, end)` within `[0, splitPoint.Count)` of size `subBatchSize`:
       - Select between sending the unit to `jobs` and `ctx.Done()`.
       - If `ctx` cancelled, break.
  6. After all units dispatched, close `jobs` channel.
  7. A `sync.WaitGroup` tracks all workers; when all workers call `wg.Done()`, the main goroutine (or a coordinator goroutine) closes `results`.
  8. Return the first error encountered (use a shared `errOnce sync.Once` + error variable).

  Error handling: workers signal errors via a shared channel `errCh chan error` (buffered to `numWorkers`). After all workers finish, drain `errCh` and return the first error.

  **Inner split variant** (when `splitPoint.Depth > 0`): The template at the split key is extracted before dispatching. Workers produce `[]any` sub-arrays. The function reassembles them into a parent map before sending to results. This is more complex; document clearly. For the initial implementation, only the depth-0 (root) split is mandatory — the inner-split case can be left as a follow-up or implemented as a sequential fallback with a `// TODO: inner split` comment.

- [x] **Step 2.4 — Implement the writer goroutine (embedded in `newStreamingWriter`)**

  The writer goroutine is not a separate named function but is launched inline. It:
  1. Opens output sinks (see Phase 3).
  2. Reads `[]map[string]any` sub-batches from a `results <-chan []map[string]any` channel.
  3. Serializes each sub-batch immediately and discards it.
  4. Signals completion via a `writerDone chan struct{}`.

- [x] **Step 2.5 — Implement `startDebugDisplay(ctx context.Context, stats *RunStats, out io.Writer) (stop func())`**

  Returns a `stop` function the caller must invoke before returning. Internally:
  1. Launches a goroutine that ticks every 200ms.
  2. On each tick, reads `stats.Generated.Load()` and `stats.BytesWritten.Load()` atomically.
  3. Writes a single `\r`-prefixed line to `out` (which must be `os.Stderr` — see caller).
  4. On stop: writes a final `\n` and exits.

  Format string (no file output):
  ```
  \r[debug] Workers: %d/%d | PeakMem: ~%s | Progress: %s / %s (%.1f%%) | Elapsed: %.1fs
  ```
  Format string (with file output):
  ```
  \r[debug] Workers: %d/%d | PeakMem: ~%s | Progress: %s / %s (%.1f%%) | Elapsed: %.1fs | File: ~%s
  ```
  Use `formatSizeMetrics` (already in `cmd/utils.go`) for byte formatting.

  The `stop` function cancels an internal context, waits for the goroutine to finish, and performs the final render.

---

### Phase 3 — Streaming Output Writers in `cmd/mock.go`

The existing `routeOutput` function accepts a completed `[]map[string]any`. This must be replaced by a streaming variant. To avoid breaking the `--generate 1` sequential path (which remains), the existing `routeOutput` is kept for the sequential path and a new `streamOutput` function is added for the concurrent path.

- [x] **Step 3.1 — Add `resolveOutputPaths` helper**

  Extract the path-resolution logic currently duplicated inside `routeOutput` into a standalone helper:
  ```go
  func resolveOutputPaths(
      toJsonFileSet bool, toJsonFileValue string,
      toCsvFileSet bool, toCsvFileValue string,
      source string,
      defaultFilePath string, defaultCsvFilePath string,
  ) (jsonPath string, csvPath string, err error)
  ```
  Returns the resolved absolute paths for JSON and CSV outputs (or empty strings if the respective flag was not set).

- [x] **Step 3.2 — Implement atomic temp-file helper**

  ```go
  // atomicFileCreate opens a temp file in the same directory as finalPath.
  // The caller writes to the returned *os.File, then calls commit() on success
  // or abort() on failure. commit() renames the temp to finalPath atomically.
  // abort() deletes the temp file.
  func atomicFileCreate(finalPath string) (f *os.File, commit func() error, abort func(), err error)
  ```

  Implementation:
  1. `dir := filepath.Dir(finalPath)`
  2. `f, err = os.CreateTemp(dir, ".ktns-tmp-*")`
  3. `commit = func() error { f.Close(); return os.Rename(f.Name(), finalPath) }`
  4. `abort = func() { f.Close(); os.Remove(f.Name()) }`

- [x] **Step 3.3 — Implement `streamOutput`**

  Signature:
  ```go
  func streamOutput(
      ctx context.Context,
      results <-chan []map[string]any,
      generate int,
      toStdout string,
      toStdoutPrettify bool,
      toJsonFileSet bool,
      jsonPath string,
      toCsvFileSet bool,
      csvPath string,
      stats *RunStats,
      out io.Writer,
  ) error
  ```

  Responsibilities:

  **JSON stdout (`toStdout == "as-json"`)**:
  - Uses a `json.Encoder` wrapping a `bufio.NewWriter(out)`.
  - Track `firstItem bool`. For `generate == 1`, emit a bare object `{}` (not wrapped in `[]`). For `generate > 1`, wrap in `[...]`.
  - For each sub-batch: iterate over items, tracking whether to write `,` separator before each item (skip before the very first item in the stream, write for all subsequent).
  - After all sub-batches: write `]\n` (for array) or nothing extra (for single object), then `Flush()`.
  - For `toStdoutPrettify`: use `json.MarshalIndent` per item rather than `json.Encoder.Encode`.

  **CSV stdout (`toStdout == "as-csv"`)**:
  - Walk the template's sanitized keys **before** dispatching workers to determine headers (sorted, deterministic). Write the header row immediately via `csv.NewWriter(bufio.NewWriter(out))`.
  - For each sub-batch: iterate items, write rows.
  - After all sub-batches: `csv.Writer.Flush()`.

  **JSON file (`toJsonFileSet`)**:
  - Call `atomicFileCreate(jsonPath)` before dispatching workers.
  - Write via `json.Encoder` wrapping `bufio.NewWriter(tempFile)`.
  - Same array framing as JSON stdout (array vs. single object based on `generate`).
  - On success: call `commit()`. On any error or `ctx.Done()`: call `abort()`.
  - Increment `stats.BytesWritten` after each write using `bufio.Writer`'s written byte count or by counting before/after.

  **CSV file (`toCsvFileSet`)**:
  - Call `atomicFileCreate(csvPath)` before dispatching workers.
  - Write header row immediately (same header logic as CSV stdout).
  - For each sub-batch: iterate and write rows.
  - On success: `csv.Writer.Flush()`, `commit()`. On error: `abort()`.

  The function reads from `results` until the channel is closed, then finalizes all sinks.

- [x] **Step 3.4 — Implement `extractCsvHeaders(template map[string]any) []string`**

  Walks the top-level keys of a sanitized template copy to produce sorted CSV headers. This mirrors what `marshalAsCSV` does for the header row today, but is called once before workers start.

  ```go
  func extractCsvHeaders(template map[string]any) []string
  ```

  Sanitize a deepcopy of the template map (to strip `[n]` from keys), collect top-level keys, sort, and return.

---

### Phase 4 — OS Signal Handling

- [x] **Step 4.1 — Add signal context to the `RunE` closure in `cmd/mock.go`**

  At the top of the `RunE` closure (before any parsing), after reading flags, add:
  ```go
  ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
  defer stop()
  ```

  Pass `ctx` into `runWorkerPool` and `streamOutput`. All internal selects and worker loops check `ctx.Done()`.

  Required imports: `"context"`, `"os/signal"`, `"syscall"`.

---

### Phase 5 — Wire Everything Together in `cmd/mock.go`

- [x] **Step 5.1 — Add `--debug` flag to `NewMockCmd`**

  After the existing flag registrations:
  ```go
  mockCmd.Flags().Bool("debug", false, "show live generation progress on stderr (opt-in)")
  ```

  In `RunE`, read it:
  ```go
  debugMode, _ := cmd.Flags().GetBool("debug")
  ```

- [x] **Step 5.2 — Refactor `runningParseJson` block**

  Replace the current sequential loop:
  ```go
  mocker := mocker.New()
  parseMaps := make([]map[string]any, generate)
  for i := range generate { ... }
  for i := range parseMaps { sanitizeJsonMap(...) }
  routeOutput(parseMaps, ...)
  ```

  With the new concurrent path:
  ```go
  numWorkers := runtime.NumCPU()
  splitPoint := analyzeTemplate(parseMap, generate, numWorkers)

  if splitPoint.UseSequential {
      // existing sequential loop (unchanged)
      mocker := mocker.New()
      parseMaps := make([]map[string]any, generate)
      for i := range generate {
          cpParseMap := deepcopy.Copy(parseMap).(map[string]any)
          if err := processJsonMap(cpParseMap, mocker); err != nil {
              return fmt.Errorf("%w", err)
          }
          sanitizeJsonMap(cpParseMap)
          parseMaps[i] = cpParseMap
      }
      if err := routeOutput(parseMaps, generate, ...); err != nil {
          return err
      }
  } else {
      // concurrent path
      jsonPath, csvPath, err := resolveOutputPaths(...)
      if err != nil { return err }

      stats := &RunStats{
          CoresAvailable:  numWorkers,
          CoresUsed:       min(numWorkers, splitPoint.Count),
          TotalWeight:     splitPoint.TotalWeight,
          WeightPerWorker: splitPoint.TotalWeight / int64(min(numWorkers, splitPoint.Count)),
          EstMemPeakBytes: int64(min(numWorkers, splitPoint.Count)+2) * int64(targetSubBatch) * estimatedItemBytes,
          StartTime:       time.Now(),
      }

      var debugStop func()
      if debugMode {
          debugStop = startDebugDisplay(ctx, stats, os.Stderr)
          defer debugStop()
      }

      results := make(chan []map[string]any, numWorkers)

      var poolErr error
      go func() {
          poolErr = runWorkerPool(ctx, splitPoint, parseMap, generate, min(numWorkers, splitPoint.Count), results, stats)
          // runWorkerPool closes results when all workers are done
      }()

      if err := streamOutput(ctx, results, generate, toStdout, toStdoutPrettify, toJsonFileSet, jsonPath, toCsvFileSet, csvPath, stats, opts.Out); err != nil {
          return err
      }
      if poolErr != nil {
          return poolErr
      }
  }
  ```

  Add a package-level constant `estimatedItemBytes = 512` as a rough estimate for memory calculation.

  **Important**: `runWorkerPool` must close `results` after all workers are done, so `streamOutput`'s range over the channel terminates naturally.

- [x] **Step 5.3 — Apply the same refactor to the `runningParseJsonFile` block**

  The logic is identical; the only difference is the template source (`parseMap` from file) and the default file paths already computed (`defaultFilePath`, `defaultCsvFilePath`).

- [x] **Step 5.4 — Keep the existing `routeOutput` function for the sequential path**

  Do not remove `routeOutput`. It continues to serve `generate == 1` and sub-threshold cases. The two functions (`routeOutput` for sequential, `streamOutput` for concurrent) coexist.

  Update `routeOutput`'s file writing to use `atomicFileCreate` (temp-then-rename) instead of the current `os.Remove` + `os.WriteFile` pattern. This applies to both JSON and CSV file outputs even on the sequential path.

---

### Phase 6 — Tests

- [x] **Step 6.1 — Unit tests for `analyzeTemplate` (already listed in Step 1.3)**

- [x] **Step 6.2 — Unit tests for `resolveOutputPaths`, `extractCsvHeaders`, `atomicFileCreate` in `cmd/mock_test.go`**

  - `resolveOutputPaths`: test all combinations of source, flag set/unset, explicit filename.
  - `extractCsvHeaders`: given a sanitized template, assert sorted keys returned.
  - `atomicFileCreate`: verify that on `commit()` the final path exists with content, and on `abort()` neither the temp nor the final file remains.

- [x] **Step 6.3 — E2E tests in `cmd/mock_e2e_test.go` for the concurrent path**

  Add cases:
  - `--parse-json '{"name":"{{ Person.name }}"}' --generate 50000 --to-stdout as-json`: output is a valid JSON array of 50,000 objects, each with a `"name"` key.
  - `--parse-json '{"emp[10000]":{"name":"{{ Person.name }}"}}' --generate 1 --to-stdout as-json`: output is a single JSON object with an `"emp"` array of 10,000 items.
  - `--parse-json '{"name":"{{ Person.name }}"}' --generate 1 --to-stdout as-json`: output is a bare object `{}` (not an array) — verifies the single-object shape is preserved.
  - `--parse-json '{"name":"{{ Person.name }}"}' --generate 50000 --to-stdout as-csv`: output is valid CSV with a header row and 50,000 data rows.
  - `--parse-json '{"name":"{{ Person.name }}"}' --generate 50000 --to-json-file <tmpfile>`: file is valid JSON array of 50,000 items; temp file does not remain; no partial file on success.
  - Error propagation: inject a broken mock function in the template; confirm the returned error is non-nil and no partial output file remains (abort path).
  - `--debug` flag: ensure command still succeeds and produces correct output; does not write progress to stdout (check `opts.Out` buffer contains only valid JSON/CSV, not debug lines).

- [x] **Step 6.4 — Add `-race` flag to the test invocation**

  If a `Makefile` exists, add a `test-race` target:
  ```makefile
  test-race:
      go test -race ./...
  ```
  Document in a comment in the plan that all new concurrent code must pass `go test -race ./...` before being merged.

---

### Phase 7 — Import Cleanup and Build Verification

- [x] **Step 7.1 — Audit `cmd/mock.go` imports**

  After all changes, ensure the import block includes:
  - `"bufio"` — streaming writers
  - `"context"` — cancellation
  - `"os/signal"` — signal handling
  - `"syscall"` — SIGTERM
  - `"runtime"` — NumCPU
  - `"sync"` — WaitGroup, Once, Mutex
  - `"sync/atomic"` — (used via `atomic.Int64` in `RunStats` struct)
  - `"time"` — RunStats.StartTime, debug display elapsed
  - Existing: `"encoding/csv"`, `"encoding/json"`, `"fmt"`, `"io"`, `"os"`, `"path/filepath"`, `"regexp"`, `"sort"`, `"strconv"`, `"strings"`, `"github.com/lfsc09/k-test-n-stress/mocker"`, `"github.com/mohae/deepcopy"`, `"github.com/spf13/cobra"`

- [x] **Step 7.2 — Audit `mocker/mocker.go` imports**

  Ensure `"math/rand"` (v1) is present for `rand.NewSource`, `rand.New`, `m.rng.Intn`. Ensure the bare `rand.Intn` calls are gone. Ensure `regen` import is still present (still used in `Regex.regex` and `Payment.creditCardCvv` via `NewGenerator`).

- [x] **Step 7.3 — Build and test**

  ```
  go build ./...
  go test ./...
  go test -race ./...
  ```

  All must pass before considering the implementation done.

---

## Notes

### Sequential Path Preservation

The sequential path (when `UseSequential == true` or `generate == 1`) must remain byte-for-byte identical in output to today's behavior. The E2E test suite for the sequential path must continue to pass without modification. No behavior change is acceptable for sub-threshold workloads.

### `generate == 1` Output Shape

The current `routeOutput` emits a bare object `{}` (not `[{}]`) when `generate == 1`. The streaming `streamOutput` must replicate this. The `generate` parameter is always available before any worker is started, so the framing decision is deterministic and can be made at writer setup time.

### Sibling `[n]` Keys

The plan addresses sibling keys (multiple `[n]` keys at the same depth, e.g., `truck[10000]`, `car[10000]`, `bus[10000]`) by dispatching them **sequentially** through the same warm worker pool. The pool is created once before the first sibling and torn down after the last. The results channel carries sub-batches of one sibling at a time; the writer never needs to demux by key. This avoids the memory growth that would result from interleaved sibling dispatch.

For the initial implementation, sibling handling applies only when the split is at depth > 0. If the split is at the root (`--generate`), there are no sibling split keys.

### Inner Split Complexity

The inner split (depth > 0) is significantly more complex than the root split because workers produce partial sub-arrays that must be reassembled into the parent map. For the initial delivery, if time is constrained, implement the root split (depth 0) fully and fall back to the sequential path for all inner splits. Add a `// TODO(concurrency): inner split` comment to track the gap. The root split already covers the most common high-volume case (`--generate N` with a flat or mildly nested template).

### `deepcopy` Reflection Cost

`deepcopy.Copy` uses reflection and can be expensive at 1,000,000 iterations. Before finalizing, benchmark with `go test -bench=. ./cmd/...`. If deepcopy is measured to be a bottleneck exceeding faker call cost, consider a hand-rolled clone for the top-level template map (which is statically known at analysis time). This is an optimization, not a correctness concern, and can be deferred to a follow-up.

### Memory Peak Estimate

```
M_peak ≈ (numWorkers + chanBuf + 1) × subBatchSize × estimatedItemBytes
```

`estimatedItemBytes = 512` bytes is a conservative default. The `--debug` display uses `EstMemPeakBytes` from `RunStats` which is computed once before workers start.

### `goregen` Thread Safety

The `regen.NewGenerator` call itself is not goroutine-safe (it parses a regex internally). Each worker goroutine should call `regen.NewGenerator` once at startup (or per-job if the pattern is dynamic) and reuse the generator for the duration of the job. For `Payment.creditCardCvv` the pattern `"[0-9]{3}"` is static — create the generator once per `mocker.New()` call and store it in the `Mock` struct, or create it once per worker and reuse it.

### CSV Header Determinism

CSV headers are extracted from the template **before** any workers are started. Headers must be sorted (already done this way in `marshalAsCSV`). This ensures that the header row written immediately matches the column order of all subsequent data rows produced by workers.

### Error Cancellation Flow

1. Worker encounters an error → sends on `errCh`, checks `ctx.Done()` immediately after and stops processing.
2. Coordinator detects first error → calls `cancelCtx()`.
3. Other workers detect `ctx.Done()`, drain their current unit, and call `wg.Done()`.
4. After all workers done → coordinator closes `results`.
5. Writer drains `results` (may receive partial sub-batches from workers that completed before cancel).
6. `streamOutput` returns `ctx.Err()` or the original worker error.
7. Temp files are aborted — no partial output remains.

### `--debug` Flag and Stderr

The `--debug` display writes exclusively to `os.Stderr`, never to `opts.Out`. This ensures that piped consumers (`ktns mock ... --to-stdout as-json | jq .`) are unaffected. The `startDebugDisplay` function receives `io.Writer` for testability but callers must always pass `os.Stderr`.

### Go Version Compatibility

The module uses Go 1.26.1. `atomic.Int64` (used in `RunStats`) requires Go 1.19+. `runtime.NumCPU()` and `signal.NotifyContext` (Go 1.16+) are all available. No compatibility concerns.

### Existing Tests Must Continue to Pass

The `cmd/mock_e2e_test.go` test suite already covers all flag combinations on the sequential path. These tests must not be modified except to add new cases. Run `go test ./cmd/...` after every step to catch regressions early.
