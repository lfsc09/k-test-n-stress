# Refactor: Remove Progress Bar Functionality

> Created: 2026-04-10 10:57

## Summary

Remove all progress bar machinery from the codebase. This means deleting the `github.com/vbauerster/mpb/v8` direct dependency (and its transitive-only dependents), removing the `--no-progress` flag and its associated `noProgress` variable, deleting the `giveMeABar` function and its `"time"` import from `cmd/mock.go`, dropping the `mpb.New` / `mpbHandler.Wait()` block, removing the `bar.Increment()` / `bar.Abort(false)` calls inside the generation loops, removing the `"io"` and `"os"` imports that were only needed for `mpbOut`, cleaning up the two E2E tests that exercise `--no-progress`, and updating every piece of documentation (the command `Long` string, `README.md`, and the agent knowledge files) that references the flag or the library.

The two helper functions `formatDurationMetrics` and `formatSizeMetrics` in `cmd/utils.go` are **retained** because they are still used by `cmd/request.go`.

## Affected Files

- `cmd/mock.go` — remove `mpb` imports, `noProgress` flag read, `mpbOut`/`mpbHandler` block, `bar` variables, `bar.Increment()`, `bar.Abort(false)`, `mpbHandler.Wait()`, `giveMeABar` function and its doc comment, `--no-progress` flag declaration, `"time"` import (only used by `giveMeABar`), Long help text line referencing `--no-progress`, example line referencing `--no-progress`
- `cmd/mock_e2e_test.go` — remove `TestCLIShouldSuppressProgressBar_ParseJson` and `TestCLIShouldSuppressProgressBar_ParseJsonFile` test functions and the `// --- Step 16: --no-progress ---` comment header above them
- `go.mod` — remove `github.com/vbauerster/mpb/v8 v8.12.0` from the `require` block; remove `github.com/VividCortex/ewma`, `github.com/acarl005/stripansi`, and `github.com/mattn/go-runewidth` from the indirect block (they are transitive dependencies of mpb only)
- `go.sum` — run `go mod tidy` to remove the checksum entries for the removed packages (not a manual edit — handled by the tidy command)
- `README.md` — remove the `--no-progress` flag bullet in the Flags section, remove the "Suppress the progress bar" example block, remove the `Mpb` entry from the Main Dependencies table
- `.claude/agents/planner.md` — remove the `github.com/vbauerster/mpb/v8` row from the Key Dependencies table
- `.claude/agents/developer.md` — remove the `github.com/vbauerster/mpb/v8` row from the Key Dependencies table

## Steps

- [x] Step 1: **Edit `cmd/mock.go` — remove imports**

  Remove the two mpb-specific import lines:
  ```
  "github.com/vbauerster/mpb/v8"
  "github.com/vbauerster/mpb/v8/decor"
  ```
  Also remove `"time"` from the import block — it is only referenced inside `giveMeABar`, which will be deleted in Step 3.
  The `"io"` import must be checked: after the refactor the `io.Writer` type alias is no longer used anywhere in `mock.go` (the `mpbOut` variable is the only use). Remove `"io"` as well.
  The `"os"` import is still needed (`os.Stat`, `os.ReadFile`, `os.Remove`, `os.WriteFile`) — keep it.

- [x] Step 2: **Edit `cmd/mock.go` — remove `noProgress` flag read and `--no-progress` flag declaration**

  Inside `RunE`, delete the line:
  ```go
  noProgress, _ := cmd.Flags().GetBool("no-progress")
  ```

  In the flag declarations block at the bottom of `NewMockCmd`, delete:
  ```go
  mockCmd.Flags().Bool("no-progress", false, "suppress the progress bar")
  ```

- [x] Step 3: **Edit `cmd/mock.go` — remove the `mpbHandler` setup and teardown**

  Delete the block that creates the progress handler (currently lines 181–189):
  ```go
  mpbOut := io.Writer(os.Stdout)
  if noProgress {
      mpbOut = io.Discard
  }
  mpbHandler := mpb.New(
      mpb.WithWidth(60),
      mpb.WithOutput(mpbOut),
      mpb.WithAutoRefresh(),
  )
  ```

  Delete the teardown call at the bottom of RunE (currently line 283):
  ```go
  mpbHandler.Wait()
  ```

- [x] Step 4: **Edit `cmd/mock.go` — remove bar usage inside the `--parse-json` generation block**

  Inside the `if runningParseJson` block, remove:
  - The comment `// Progress bar: total = generate (one tick per root object produced)`
  - The `bar := giveMeABar("parse-json", int64(generate), nil, mpbHandler)` call
  - The `bar.Abort(false)` call inside the error return branch
  - The `bar.Increment()` call at the end of the loop body

- [x] Step 5: **Edit `cmd/mock.go` — remove bar usage inside the `--parse-json-file` generation block**

  Inside the `if runningParseJsonFile` block, remove:
  - The comment `// Progress bar: total = generate (one tick per root object produced); defaultFilePath for file-size display`
  - The `bar := giveMeABar(filepath.Base(parseJsonFile), int64(generate), &defaultFilePath, mpbHandler)` call
  - The `bar.Abort(false)` call inside the error return branch
  - The `bar.Increment()` call at the end of the loop body

- [x] Step 6: **Edit `cmd/mock.go` — delete the `giveMeABar` function**

  Delete the full function including its multi-line doc comment (currently lines 621–655):
  ```go
  // Creates and returns a progress bar for a given task, ...
  func giveMeABar(taskName string, total int64, outPath *string, mpbHandler *mpb.Progress) *mpb.Bar { ... }
  ```

- [x] Step 7: **Edit `cmd/mock.go` — clean up the `Long` help text**

  In the `Long` field of the command, remove:
  - The bullet point line: `* --no-progress: suppress the progress bar.`
  - The example line at the bottom of the examples list: `ktns mock --parse-json-file "path/to/employees.template.json" --no-progress --to-json-file`

- [x] Step 8: **Edit `cmd/mock_e2e_test.go` — remove the two `--no-progress` tests**

  Delete the comment header and both test functions:
  ```go
  // --- Step 16: --no-progress ---

  func (suite *MockCmdE2ETestSuite) TestCLIShouldSuppressProgressBar_ParseJson() { ... }

  func (suite *MockCmdE2ETestSuite) TestCLIShouldSuppressProgressBar_ParseJsonFile() { ... }
  ```
  Verify that no other import in `mock_e2e_test.go` becomes unused after this deletion (the functions only use packages already required by other tests).

- [x] Step 9: **Edit `go.mod` — remove mpb direct dependency and its exclusive transitive dependencies**

  From the `require` block, remove:
  ```
  github.com/vbauerster/mpb/v8 v8.12.0
  ```

  From the `// indirect` block, remove the three entries that have no other dependent in the module:
  ```
  github.com/VividCortex/ewma v1.2.0 // indirect
  github.com/acarl005/stripansi v0.0.0-20180116102854-5a71ef0e047d // indirect
  github.com/mattn/go-runewidth v0.0.22 // indirect
  ```

  After editing, run `go mod tidy` in the terminal to regenerate `go.sum` and confirm no other hidden dependency on these packages exists.

- [x] Step 10: **Edit `README.md` — remove `--no-progress` from the Flags section**

  In the `### Flags` bullet list under `## mock`, delete the line:
  ```
  - `--no-progress`: Suppress the progress bar.
  ```

- [x] Step 11: **Edit `README.md` — remove the "Suppress the progress bar" example block**

  Under `#### --parse-json-file`, delete the sub-section that reads:

  ```
  Suppress the progress bar:

  ```bash
  ktns mock --parse-json-file employee.template.json --to-json-file --no-progress
  ```
  ```

- [x] Step 12: **Edit `README.md` — remove the `Mpb` entry from Main Dependencies**

  In the `### Main Dependencies` list under `## Development Details`, delete the line:
  ```
  - [`Mpb`](https://github.com/vbauerster/mpb): Multi progress bar for Go CLI applications.
  ```

- [x] Step 13: **Edit `.claude/agents/planner.md` — remove mpb from Key Dependencies table**

  In the Key Dependencies table, delete the row:
  ```
  | `github.com/vbauerster/mpb/v8` | Progress bar for `--parse-files` |
  ```

- [x] Step 14: **Edit `.claude/agents/developer.md` — remove mpb from Key Dependencies table**

  In the Key Dependencies table, delete the row:
  ```
  | `github.com/vbauerster/mpb/v8` | Progress bar for `--parse-files` |
  ```

- [x] Step 15: **Run `go build ./...` and `go test ./...` to confirm everything compiles and all tests pass**

## Notes

- `formatDurationMetrics` and `formatSizeMetrics` in `cmd/utils.go` are **not** removed. They continue to be used by `cmd/request.go` for the `--with-metrics` flag output. No changes to `cmd/utils.go` are needed.
- The `"os"` import in `cmd/mock.go` survives because `os.Stat`, `os.ReadFile`, `os.Remove`, and `os.WriteFile` are all still in use after removing the progress bar code.
- `golang.org/x/sys` is listed as an indirect dependency in `go.mod` but is also a transitive dependency of other packages (e.g. `goregen`). Do not remove it manually — let `go mod tidy` decide.
- After `go mod tidy` the `go.sum` file will have several lines removed corresponding to the deleted packages. This is expected and correct.
- The `"io"` standard library package: confirm it has no other use in `cmd/mock.go` before removing. The only current usage is `io.Writer(os.Stdout)` and `io.Discard` inside the `mpbOut` block, both of which will be gone after Step 3. The `routeOutput` function signature uses `io.Writer` as a parameter type — check whether this is declared in `mock.go` or referenced inline. If `io.Writer` still appears (e.g. in `routeOutput`'s signature), keep the import. Looking at the current code: `routeOutput` accepts `out io.Writer` — this is declared in `mock.go`, so the `"io"` import must be retained.
