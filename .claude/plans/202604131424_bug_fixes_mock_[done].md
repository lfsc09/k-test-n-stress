# Bug Fixes: Concurrent CSV Header Duplication, sanitizeJsonMap Array Recursion, Hot Regex Compilation, Unbounded --generate, Duplicate Test Runner Name, Missing Concurrent E2E Tests, processStr Silent Errors

> Created: 2026-04-13 14:24

## Summary

Seven issues have been identified in `cmd/mock.go`, `cmd/mock_test.go`, and `cmd/mock_e2e_test.go`. They range from a data-corruption bug in the concurrent CSV path (C1), a sanitization gap for nested array templates (C3), two performance regressions from hot regex compilation (M1), an unbounded memory allocation via `--generate` on the sequential path (M5), a confusing duplicate test runner name across packages (G3), missing E2E coverage for the concurrent path (G5), and silent error embedding in `--parse-str` output (G6). This plan describes every edit needed, in a safe incremental order.

## Affected Files

- `cmd/mock.go` — C1: remove redundant CSV header write inside `writeItem`; C3: add `[]any` branch to `sanitizeJsonMap`; M1: hoist two `regexp.MustCompile` calls to package-level vars; M5: add upper-bound guard on `--generate`; G6: surface `processStr` errors to stderr
- `cmd/mock_test.go` — G3: rename `TestMockCmdTestSuite` to `TestMockCmdUnitSuite`; add unit test for `sanitizeJsonMap` array recursion (C3 regression guard)
- `cmd/mock_e2e_test.go` — G3: rename `TestMockCmdTestSuite` to `TestMockCmdE2ESuite`; G5: add concurrent-path E2E tests for JSON and CSV output

## Steps

- [x] **Step 1 — M1: Hoist regex patterns to package-level vars**

  In `cmd/mock.go`, immediately below the existing package-level declaration on line 27:
  ```go
  var objKeyNumberRegex = regexp.MustCompile(`^[^\[\]\s]+\[(\d+)\]$`)
  ```
  add two new package-level vars:
  ```go
  // interpretStringRegex matches a value that is entirely {{ … }}.
  var interpretStringRegex = regexp.MustCompile(`^\s*{{(.*)}}\s*$`)

  // bracketCharRegex detects any '[' or ']' character in a key string.
  var bracketCharRegex = regexp.MustCompile(`[\[\]]`)
  ```

  Then in `interpretString` (line 725), remove the local `re := regexp.MustCompile(...)` declaration and replace the usage of `re` with `interpretStringRegex`.

  Then in `extractDigitInBrackets` (line 926), remove the inline `regexp.MustCompile(`[\[\]]`)` literal and replace it with `bracketCharRegex`.

  These are purely mechanical substitutions — no logic changes.

- [x] **Step 2 — C1: Remove duplicate CSV header write inside `writeItem`**

  In `cmd/mock.go`, inside the `writeItem` closure (~lines 1231–1347), locate the `// CSV file` block (lines 1321–1344). The current code is:
  ```go
  // CSV file
  if csvFileWriter != nil {
      if len(csvHeaders) == 0 {
          for k := range item {
              csvHeaders = append(csvHeaders, k)
          }
          sort.Strings(csvHeaders)
      }
      // Write header on first item for CSV file too (stdout CSV and file may share headers)
      if firstItem && csvFileWriter != nil {
          if err := csvFileWriter.Write(csvHeaders); err != nil {
              return err
          }
      }
      row := make([]string, len(csvHeaders))
      ...
  }
  ```

  Replace it with:
  ```go
  // CSV file
  if csvFileWriter != nil {
      row := make([]string, len(csvHeaders))
      for i, h := range csvHeaders {
          if val, ok := item[h]; ok {
              row[i] = fmt.Sprintf("%v", val)
          }
      }
      if err := csvFileWriter.Write(row); err != nil {
          return err
      }
  }
  ```

  Rationale: the outer batch loop (lines 1364–1382) already writes the CSV header to both `stdoutCsvWriter` and `csvFileWriter` on the very first item of the first batch, before `writeItem` is called. By the time `writeItem` executes for that first item, `csvHeaders` is already populated and the header has already been written. The `len(csvHeaders) == 0` guard inside `writeItem` is therefore always false on the first call (because the outer loop populates `csvHeaders` first), but the redundant `if firstItem && csvFileWriter != nil` block fires regardless because `firstItem` is still `true` when `writeItem` is called (it is set to `false` only after `writeItem` returns on line 1388). Removing both the header-derivation sub-block and the `if firstItem` header-write sub-block from `writeItem` eliminates the duplicate row.

  The `// Stdout CSV` block inside `writeItem` (lines 1278–1298) has its own guard `if len(csvHeaders) == 0` for the stdout path. That block must be left unchanged because the outer loop only writes the stdout CSV header when `stdoutCsvWriter != nil` and `csvFileWriter != nil` cases are joined. Actually, reading the outer loop at lines 1364–1382: it writes headers to `stdoutCsvWriter` AND `csvFileWriter` unconditionally when either sink is present. So the `if len(csvHeaders) == 0` guard inside `writeItem`'s stdout CSV section will always be false too. However, it is harmless (it never fires) and the plan issue description only flags the CSV file write, so leave the stdout CSV block as-is to minimise diff noise.

- [x] **Step 3 — C3: Add `[]any` recursion to `sanitizeJsonMap`**

  In `cmd/mock.go`, in the `sanitizeJsonMap` function (lines 840–861), after the block:
  ```go
  // Recurse on nested maps
  if mapValue, ok := objValue.(map[string]any); ok {
      sanitizeJsonMap(mapValue)
  }
  ```
  add a new block that recurses into slice elements:
  ```go
  // Recurse into array elements that are maps
  if sliceValue, ok := objValue.([]any); ok {
      for _, elem := range sliceValue {
          if elemMap, ok := elem.(map[string]any); ok {
              sanitizeJsonMap(elemMap)
          }
      }
  }
  ```

  This mirrors the existing pattern in `processJsonMap`'s `[]any` branch (lines 804–830) where it recurses into `map[string]any` array elements. After the key rename (`parseMap[sanitizedKey] = objValue; delete(parseMap, objKey)`), the array's elements still hold maps with their original unsanitized keys; this new block ensures those keys are sanitized too.

  Note: the order of the recursion block and the key-rename block matters. The `if mapValue, ok := ...` recursion block executes before the rename block (`if sanitizedKey != objKey`). The new `[]any` recursion block must be placed in the same pre-rename position so that `objValue` still holds the original reference. Slices in Go are reference types, so mutating the map keys inside each element of `sliceValue` mutates the data in-place — no reassignment of `objValue` is needed.

  Revised body of the inner `for _, objKey := range objKeys` loop:
  ```go
  for _, objKey := range objKeys {
      objValue := parseMap[objKey]
      sanitizedKey := sanitizeKeyWithBrackets(objKey)

      // Recurse on nested maps
      if mapValue, ok := objValue.(map[string]any); ok {
          sanitizeJsonMap(mapValue)
      }

      // Recurse into array elements that are maps
      if sliceValue, ok := objValue.([]any); ok {
          for _, elem := range sliceValue {
              if elemMap, ok := elem.(map[string]any); ok {
                  sanitizeJsonMap(elemMap)
              }
          }
      }

      if sanitizedKey != objKey {
          parseMap[sanitizedKey] = objValue
          delete(parseMap, objKey)
      }
  }
  ```

- [x] **Step 4 — M5: Add upper-bound guard on `--generate`**

  In `cmd/mock.go`, in `NewMockCmd`'s `RunE`, after the existing lower-bound guard:
  ```go
  if generate <= 0 {
      return fmt.Errorf("--generate option must be greater than 0")
  }
  ```
  add immediately below it:
  ```go
  const maxGenerate = 10_000_000
  if generate > maxGenerate {
      return fmt.Errorf("--generate option must not exceed %d", maxGenerate)
  }
  ```

  This prevents the sequential path from pre-allocating a `[]map[string]any` of arbitrarily large size. The constant `maxGenerate = 10_000_000` is defined locally inside `RunE` (not at package level) because it is a policy constraint, not a shared constant. The concurrent path is unaffected — it streams output and does not pre-allocate.

- [x] **Step 5 — G6: Surface `processStr` errors to stderr**

  In `cmd/mock.go`, in the `processStr` function (lines 868–913), the two error cases currently embed error text inline:
  ```go
  out.WriteString(fmt.Sprintf("[%v]", err))
  ```
  These occur once for `extractMockMethod` errors and once for `mocker.Generate` errors.

  Change the function signature from:
  ```go
  func processStr(parseStr string, mocker *mocker.Mock) string
  ```
  to:
  ```go
  func processStr(parseStr string, mocker *mocker.Mock, errOut io.Writer) string
  ```

  Then replace each `out.WriteString(fmt.Sprintf("[%v]", err))` occurrence with:
  ```go
  fmt.Fprintf(errOut, "warning: --parse-str mock substitution error: %v\n", err)
  out.WriteString(fmt.Sprintf("[%v]", err))
  ```

  This preserves the existing inline-error behavior (so downstream parsers still receive the bracketed placeholder) while also printing a warning to `errOut` so the user is informed. The placeholder text in the output is left unchanged so existing callers see the same behavior; a stderr warning line is added.

  Update the single call site in `RunE` (line 505):
  ```go
  mockedStr := processStr(parseStr, mocker)
  ```
  to:
  ```go
  mockedStr := processStr(parseStr, mocker, os.Stderr)
  ```

  No test update is required because the existing `TestCLIShouldMockFromParseStr` tests check stdout output only. The warning goes to `os.Stderr` directly, not through `opts.Out`, which is the correct channel for diagnostic messages (consistent with `--debug` behavior).

- [x] **Step 6 — G3: Rename duplicate test runner functions**

  In `cmd/mock_test.go` (package `cmd`), rename the top-level runner on line 17:
  ```go
  func TestMockCmdTestSuite(t *testing.T) {
  ```
  to:
  ```go
  func TestMockCmdUnitSuite(t *testing.T) {
  ```

  In `cmd/mock_e2e_test.go` (package `cmd_test`), rename the top-level runner on line 21:
  ```go
  func TestMockCmdTestSuite(t *testing.T) {
  ```
  to:
  ```go
  func TestMockCmdE2ESuite(t *testing.T) {
  ```

  The suite struct names (`MockCmdTestSuite` and `MockCmdE2ETestSuite`) are already distinct; only the `Test*` runner function names need changing. No other references to `TestMockCmdTestSuite` exist in the codebase (it is a `testing.T`-registered function, not called explicitly).

- [x] **Step 7 — C3 unit test: Add `TestSanitizeJsonMap_ArrayRecursion` to `mock_test.go`**

  In `cmd/mock_test.go`, add a new test method to `MockCmdTestSuite` that exercises the `[]any` recursion path added in Step 3:

  ```go
  func (suite *MockCmdTestSuite) TestSanitizeJsonMap_ArrayRecursion() {
      tests := []struct {
          testName string
          input    map[string]any
          wantKeys []string // top-level keys expected after sanitization
      }{
          {
              testName: "array of maps with bracketed keys inside",
              input: map[string]any{
                  "outer[2]": []any{
                      map[string]any{"inner[3]": "value1"},
                      map[string]any{"inner[3]": "value2"},
                  },
              },
              wantKeys: []string{"outer"},
          },
          {
              testName: "flat array of strings is unaffected",
              input: map[string]any{
                  "phones[3]": []any{"111", "222", "333"},
              },
              wantKeys: []string{"phones"},
          },
      }

      for _, tt := range tests {
          sanitizeJsonMap(tt.input)
          for _, wantKey := range tt.wantKeys {
              _, ok := tt.input[wantKey]
              assert.True(suite.T(), ok, "Test case '%s': expected key '%s' after sanitization", tt.testName, wantKey)
          }
          // Verify array elements also have sanitized keys
          if tt.testName == "array of maps with bracketed keys inside" {
              arr := tt.input["outer"].([]any)
              for i, elem := range arr {
                  elemMap := elem.(map[string]any)
                  _, hasUnsanitized := elemMap["inner[3]"]
                  assert.False(suite.T(), hasUnsanitized, "Test case '%s': element %d still has unsanitized key", tt.testName, i)
                  _, hasSanitized := elemMap["inner"]
                  assert.True(suite.T(), hasSanitized, "Test case '%s': element %d missing sanitized key", tt.testName, i)
              }
          }
      }
  }
  ```

- [x] **Step 8 — G5: Add concurrent-path E2E tests to `mock_e2e_test.go`**

  In `cmd/mock_e2e_test.go`, add the following test methods to `MockCmdE2ETestSuite` after the existing `TestCLIShouldOutputBothJsonAndCsvFile` test:

  ```go
  // --- Concurrent path (--generate >= 10000) ---

  func (suite *MockCmdE2ETestSuite) TestCLIConcurrentPath_JsonOutput_CorrectCount() {
      testName := "Concurrent path: JSON file contains exactly 10001 items"
      tmpDir := suite.T().TempDir()
      outPath := filepath.Join(tmpDir, "result.json")
      _, err := suite.executeCommand(
          "mock", "--parse-json", `{"name": "{{ Person.name }}"}`,
          "--generate", "10001",
          "--to-json-file="+outPath,
      )
      assert.NoError(suite.T(), err, testName)

      data, readErr := os.ReadFile(outPath)
      assert.NoError(suite.T(), readErr, testName)

      var result []map[string]any
      jsonErr := json.Unmarshal(data, &result)
      assert.NoError(suite.T(), jsonErr, testName)
      assert.Len(suite.T(), result, 10001, testName)
  }

  func (suite *MockCmdE2ETestSuite) TestCLIConcurrentPath_CsvOutput_NoDuplicateHeader() {
      testName := "Concurrent path: CSV file has exactly one header row and 10001 data rows"
      tmpDir := suite.T().TempDir()
      outPath := filepath.Join(tmpDir, "result.csv")
      _, err := suite.executeCommand(
          "mock", "--parse-json", `{"name": "{{ Person.name }}"}`,
          "--generate", "10001",
          "--to-csv-file="+outPath,
      )
      assert.NoError(suite.T(), err, testName)

      data, readErr := os.ReadFile(outPath)
      assert.NoError(suite.T(), readErr, testName)

      lines := strings.Split(strings.TrimRight(string(data), "\n"), "\n")
      // Exactly 1 header + 10001 data rows = 10002 lines total
      assert.Len(suite.T(), lines, 10002, testName)
      // First line must be the header
      assert.Equal(suite.T(), "name", lines[0], testName)
      // Second line must NOT equal the header (would indicate duplicate)
      if len(lines) > 1 {
          assert.NotEqual(suite.T(), "name", lines[1], testName)
      }
  }

  func (suite *MockCmdE2ETestSuite) TestCLIConcurrentPath_StdoutJson_CorrectStructure() {
      testName := "Concurrent path: stdout JSON is a valid array with 10001 elements"
      stdOut, err := suite.executeCommand(
          "mock", "--parse-json", `{"name": "{{ Person.name }}"}`,
          "--generate", "10001",
          "--to-stdout", "as-json",
      )
      assert.NoError(suite.T(), err, testName)

      var result []map[string]any
      jsonErr := json.Unmarshal([]byte(strings.TrimSpace(stdOut)), &result)
      assert.NoError(suite.T(), jsonErr, testName)
      assert.Len(suite.T(), result, 10001, testName)
  }

  func (suite *MockCmdE2ETestSuite) TestCLIConcurrentPath_StdoutCsv_NoDuplicateHeader() {
      testName := "Concurrent path: stdout CSV has exactly one header row and 10001 data rows"
      stdOut, err := suite.executeCommand(
          "mock", "--parse-json", `{"name": "{{ Person.name }}"}`,
          "--generate", "10001",
          "--to-stdout", "as-csv",
      )
      assert.NoError(suite.T(), err, testName)

      lines := strings.Split(strings.TrimRight(stdOut, "\n"), "\n")
      // Exactly 1 header + 10001 data rows = 10002 lines total
      assert.Len(suite.T(), lines, 10002, testName)
      assert.Equal(suite.T(), "name", lines[0], testName)
      if len(lines) > 1 {
          assert.NotEqual(suite.T(), "name", lines[1], testName)
      }
  }
  ```

  These tests use `--generate 10001` (one above `concurrencyThreshold = 10_000`) to force the concurrent path without making the threshold configurable. They cover: correct item count in JSON output, no duplicate CSV header in file output, correct JSON array structure on stdout, and no duplicate CSV header on stdout.

  Note: these tests will be slow (generating 10001 items). If the CI pipeline imposes a per-test timeout that is too tight, consider reducing to `--generate 10000` (the threshold itself is inclusive: `totalWeight < concurrencyThreshold` is false at exactly 10000, so `--generate 10000` also routes through the concurrent path).

- [x] **Step 9 — M5 E2E test: Add `--generate` upper-bound test to `mock_e2e_test.go`**

  In `cmd/mock_e2e_test.go`, extend the existing `TestCLIShouldRaiseError_GenerateFlagInvalidValues` test suite or add a companion test:

  ```go
  func (suite *MockCmdE2ETestSuite) TestCLIShouldRaiseError_GenerateFlagExceedsMax() {
      testName := "--generate above 10,000,000 should return an error"
      _, err := suite.executeCommand(
          "mock", "--parse-json", `{"name": "{{ Person.name }}"}`,
          "--generate", "10000001",
          "--to-stdout", "as-json",
      )
      assert.Error(suite.T(), err, testName)
      assert.Contains(suite.T(), err.Error(), "must not exceed", testName)
  }
  ```

- [x] **Step 10 — Verify: run the test suite**

  Run:
  ```
  go test ./cmd/... -v -count=1
  ```

  Confirm:
  - `TestMockCmdUnitSuite` runs in `package cmd` (previously `TestMockCmdTestSuite`).
  - `TestMockCmdE2ESuite` runs in `package cmd_test` (previously `TestMockCmdTestSuite`).
  - No duplicate runner name warning in output.
  - `TestSanitizeJsonMap_ArrayRecursion` passes.
  - `TestCLIConcurrentPath_CsvOutput_NoDuplicateHeader` passes (C1 regression verified).
  - `TestCLIConcurrentPath_JsonOutput_CorrectCount` passes.
  - `TestCLIShouldRaiseError_GenerateFlagExceedsMax` passes.
  - All pre-existing tests still pass.

- [ ] **Step 11 — Apply `## Proposed Memory Updates` below to `.claude/memories.md`.**

## Notes

**C1 execution path trace**: In `streamOutput`, the outer batch loop runs first. For the very first item of the first batch, `firstItem == true` and `!csvHeaderWritten` — so headers are derived, `csvHeaderWritten = true`, and headers are written to both `stdoutCsvWriter` and `csvFileWriter`. Then `writeItem(item, sep)` is called with `firstItem` still `true` (line 1384 calls writeItem, line 1388 sets `firstItem = false`). Inside `writeItem`, the `// Stdout CSV` block guards on `len(csvHeaders) == 0` — this is `false` because the outer loop already populated `csvHeaders`, so no second stdout CSV header write occurs. But the `// CSV file` block guards on `firstItem && csvFileWriter != nil` — `firstItem` is still `true`, so the file CSV header is written a second time. The fix removes both the header-derivation sub-block and the `if firstItem` guard from `writeItem`'s CSV file section entirely.

**C3 apply-order note**: In `sanitizeJsonMap`, the `[]any` recursion block must come before the `if sanitizedKey != objKey` rename block. Since `objValue` captures the slice reference before the rename, and slices are reference types, the in-place mutation of each element's map keys is reflected in `parseMap` automatically — no separate assignment needed.

**M1 naming**: The new package-level vars are named `interpretStringRegex` and `bracketCharRegex`. They sit alongside `objKeyNumberRegex` at the top of `mock.go`. All three follow the `xxxRegex` convention.

**M5 constant placement**: `maxGenerate = 10_000_000` is declared as a local constant inside `RunE` rather than a package-level constant because it is a policy enforcement value. This keeps it close to the validation code and avoids polluting the package namespace. If future code needs this value elsewhere, promote it to package level at that time.

**G5 test performance**: Each concurrent-path E2E test generates 10001 items. At typical mock generation rates (~500k items/sec on a 4-core machine using the worker pool), each test takes roughly 20–100ms plus file I/O overhead. This is acceptable for an E2E suite. If the project ever grows a `-short` test mode, tag these with `if testing.Short() { t.Skip(...) }`.

**G6 scope**: The issue description gives two options: return an error or write to stderr. The plan chooses the stderr-warning approach (keeping the inline placeholder behavior) because: (1) `--parse-str` is documented as "always outputs to stdout" — returning an error and producing zero output would be a bigger behavioral change, and (2) the placeholder text in the output is a visible signal that something went wrong, which some users may rely on. The stderr warning makes it machine-detectable without breaking the output contract.

**Interaction between C1 fix and stdoutCsvWriter**: After the C1 fix, `writeItem`'s `// Stdout CSV` block still has `if len(csvHeaders) == 0` as its guard. In practice this guard is always `false` when `writeItem` is first called (the outer loop populates `csvHeaders` before calling `writeItem`). This guard is therefore dead code after the fix but is harmless. It is left in place to avoid unnecessary noise in the diff.

## Proposed Memory Updates

Add to **Known Gotchas** section:

- **`writeItem` in `streamOutput` does not write CSV headers** — CSV headers for both `stdoutCsvWriter` and `csvFileWriter` are written exclusively in the outer batch loop (before the `writeItem` call) when `firstItem && !csvHeaderWritten`. The `writeItem` closure only writes data rows. Any future refactor that removes the outer-loop header write must add an equivalent header-write inside `writeItem`.
- **`sanitizeJsonMap` recurses into `[]any` slices** — as of the C3 fix, `sanitizeJsonMap` walks into `map[string]any` elements inside `[]any` values, mirroring `processJsonMap`'s array branch. Both functions must stay in sync when array handling is extended.
- **`processStr` emits warnings to `os.Stderr`** — when `extractMockMethod` or `mocker.Generate` fails, `processStr` writes a `warning:` line to the `errOut io.Writer` argument (called with `os.Stderr` from `RunE`) and still embeds `[error]` inline in the output. Exit code remains 0.

Add to **Architecture Decisions** section:

- **`--generate` maximum is 10,000,000** — enforced in `RunE` after the `> 0` guard. The sequential path pre-allocates a `[]map[string]any` of `generate` elements; without this cap, a large value exhausts memory. The concurrent path is streaming and is not affected, but the cap applies to both paths for consistency.

Add to **Plan History** section (newest first row):

| `202604130000_bug_fixes_mock` | 2026-04-13 | bug |
