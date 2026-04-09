# Refactor: Replace `--parse-files` with `--parse-json-file`

> Created: 2026-04-09 10:47

## Summary

Replace the `--parse-files` flag (which accepted a glob pattern or directory and processed multiple template files concurrently) with a simpler `--parse-json-file` flag that reads a single, explicitly named `.template.json` file. The new flag reuses the existing `--generate` flag (same as `--parse-json`) to control how many root objects are produced. The output file is always written alongside the template file (not into an `out/` directory). All goroutine/concurrency logic, the progress-bar infrastructure, `--preserve-folder-structure`, and the filename-bracket-digit extraction for files are removed. The parsed JSON object produced by `--parse-json-file` is structurally identical to what `--parse-json` produces. All documentation (`README.md`, `cmd/mock.go` long-help, `cmd/utils.go`) is updated to reflect these changes.

## Affected Files

- `cmd/mock.go` — primary file: rename flag, remove goroutines/progress bar/preserve-folder-structure/findTemplateFiles/toFile/normalizeParseFrom/giveMeABar, rework RunE logic for `--parse-json-file`, update Long help text, update flag declarations and validation guards, simplify `extractDigitInBrackets` (remove `"file"` branch)
- `cmd/mock_test.go` — remove unit tests that cover the now-deleted `"file"` place branch of `extractDigitInBrackets`; update any test that mentions `--parse-files` or `--preserve-folder-structure`
- `cmd/mock_e2e_test.go` — delete/replace E2E tests that reference `--parse-files`, `--preserve-folder-structure`; add new E2E tests for `--parse-json-file` (file not found, successful single-file parse, `--generate` works with `--parse-json-file`)
- `README.md` — rewrite the `--parse-files` section to document `--parse-json-file`; remove `--preserve-folder-structure` section; update flag list, examples, Development Details table, and concurrency note

## Steps

- [x] Step 1: **Remove the `"file"` branch from `extractDigitInBrackets` in `cmd/mock.go`**

  The function currently handles `place == "file"` to extract a digit from filenames like `employees[5].template.json`. Since `--parse-json-file` no longer derives the generate count from the filename, this branch is dead code.

  - Delete the `place == "file"` conditional block inside `extractDigitInBrackets`.
  - Remove the `filenameNumberRegex` package-level variable (`var filenameNumberRegex = regexp.MustCompile(...)`) — it is only used by the `"file"` branch.
  - Update the function signature comment to reflect that it now only handles `"object"` place values.
  - The function should now return an error immediately if `place != "object"`.

- [x] Step 2: **Remove helper functions that are only needed for `--parse-files` in `cmd/mock.go`**

  The following functions exist exclusively to support the old multi-file, goroutine-based flow and must be deleted entirely:

  - `findTemplateFiles(input string) ([]string, error)` — discovers template files from a glob/directory.
  - `toFile(preserveFolderStructure bool, inPath string, outPath *string, parseFiles string, result *[]map[string]any, mu *sync.Mutex, createdDirs *map[string]bool) error` — writes output to `out/` directory.
  - `normalizeParseFrom(input string) (string, error)` — normalizes the parse-from path for folder-structure preservation.

  **`giveMeABar` must be kept** — it will be refactored in Step 4a (see below) to serve both `--parse-json` and `--parse-json-file`.

- [x] Step 3: **Remove only the unused imports from `cmd/mock.go`**

  After removing the goroutine/WaitGroup/Mutex block and the three deleted helper functions, only `"sync"` is no longer needed:

  - Remove `"sync"` — was used only for `sync.WaitGroup` and `sync.Mutex` in the goroutine loop.

  Keep all other imports, including `"time"`, `"github.com/vbauerster/mpb/v8"`, and `"github.com/vbauerster/mpb/v8/decor"` — these remain in use by the refactored `giveMeABar`.

  Full import list after this step: `"encoding/json"`, `"fmt"`, `"os"`, `"path/filepath"`, `"regexp"`, `"strconv"`, `"strings"`, `"time"`, `"github.com/lfsc09/k-test-n-stress/mocker"`, `"github.com/mohae/deepcopy"`, `"github.com/spf13/cobra"`, `"github.com/vbauerster/mpb/v8"`, `"github.com/vbauerster/mpb/v8/decor"`.

- [x] Step 4a: **Refactor `giveMeABar` in `cmd/mock.go` — remove labels, base progress on `generate` count**

  The old signature was:
  ```go
  func giveMeABar(taskName string, outPath *string, labels []string, mpbHandler *mpb.Progress) *mpb.Bar
  ```
  The old bar had a fixed total of `len(labels)` (always 5) and showed a step label string like `"reading"`, `"parsing"`, etc.

  **New signature:**
  ```go
  func giveMeABar(taskName string, total int64, outPath *string, mpbHandler *mpb.Progress) *mpb.Bar
  ```

  - `total` is `int64(generate)` — the bar tracks one tick per generated root object, giving precise per-object progress.
  - `outPath` remains a `*string` pointer; if `nil` (or the file does not yet exist), the size decorator shows `[N/A]`. This is only meaningful for `--parse-json-file`; pass `nil` for `--parse-json`.
  - Remove the `labels []string` parameter and the `decor.Any` closure that printed step labels.
  - Keep the `decor.CountersNoUnit(" %d/%d ", ...)` decorator to show `<objects done>/<total objects>`.
  - Keep the elapsed-time `decor.Any` closure (unchanged logic).
  - Keep the file-size `decor.Any` closure (unchanged logic, still reads `*outPath`).

  Full replacement body:
  ```go
  func giveMeABar(taskName string, total int64, outPath *string, mpbHandler *mpb.Progress) *mpb.Bar {
      startElapsedTime := time.Now()
      var elapsedTime time.Duration
      bar := mpbHandler.AddBar(total,
          mpb.PrependDecorators(
              decor.Name(taskName, decor.WCSyncWidthR),
              decor.CountersNoUnit(" %d/%d ", decor.WCSyncWidthR),
          ),
          mpb.AppendDecorators(
              decor.Any(func(s decor.Statistics) string {
                  if !s.Completed {
                      elapsedTime = time.Since(startElapsedTime)
                  }
                  return formatDurationMetrics(elapsedTime)
              }, decor.WCSyncWidth),
              decor.Any(func(s decor.Statistics) string {
                  if outPath == nil {
                      return " [N/A] "
                  }
                  info, err := os.Stat(*outPath)
                  if err != nil {
                      return " [N/A] "
                  }
                  return formatSizeMetrics(info.Size())
              }, decor.WCSyncWidth),
          ),
      )
      return bar
  }
  ```

- [x] Step 4: **Replace the `--parse-files` flag declaration and remove `--preserve-folder-structure` in `cmd/mock.go`**

  In the block at the bottom of `NewMockCmd` where flags are declared:

  - Remove: `mockCmd.Flags().String("parse-files", "", "...")`
  - Remove: `mockCmd.Flags().Bool("preserve-folder-structure", false, "...")`
  - Add: `mockCmd.Flags().String("parse-json-file", "", "pass a path to a single .template.json file. The mock data will be generated based on this file and written alongside it")`
  - The `--generate` flag declaration stays unchanged. Its description must be updated to say "only available for `--parse-json` and `--parse-json-file`".

- [x] Step 5: **Rewrite `RunE` in `cmd/mock.go`**

  Replace the body of the `RunE` closure with the new logic below. Keep the `--parse-str` and `--parse-json` branches exactly as they are today. Only the `--parse-files` block is replaced.

  5a. **Flag reads** — replace:
  ```
  parseFiles, _ := cmd.Flags().GetString("parse-files")
  preserveFolderStructure, _ := cmd.Flags().GetBool("preserve-folder-structure")
  ```
  with:
  ```
  parseJsonFile, _ := cmd.Flags().GetString("parse-json-file")
  ```

  5b. **Mutual-exclusion counter** — rename the `runningParseFiles` variable to `runningParseJsonFile` and check `parseJsonFile != ""` instead of `parseFiles != ""`. Update the error message for `parseCheck > 1` to read: `"provide only one of the three options: --parse-json, --parse-json-file or --parse-str"`.

  5c. **Remove the multi-arg guard** — delete:
  ```go
  if runningParseFiles && len(args) > 0 {
      return fmt.Errorf("you passed multiple files to --parse-files without quotes...")
  }
  ```
  This check no longer applies because `--parse-json-file` takes a single path, not a glob.

  5d. **Remove the `--preserve-folder-structure` guard** — delete:
  ```go
  if preserveFolderStructure && !runningParseFiles {
      return fmt.Errorf("--preserve-folder-structure option is only available when using --parse-files")
  }
  ```

  5e. **Update the `--generate` guard** — change:
  ```go
  if generate > 1 && !runningParseJson {
  ```
  to:
  ```go
  if generate > 1 && !runningParseJson && !runningParseJsonFile {
  ```
  and update the error message to: `"--generate option is only available when using --parse-json or --parse-json-file"`.

  5f. **Keep the `mpb.New(...)` / `mpbHandler.Wait()` calls** — the `mpbHandler` is still created at the top of RunE and `mpbHandler.Wait()` is still called at the bottom. Both `--parse-json` and `--parse-json-file` will use it.

  5g. **Wire a progress bar into the existing `--parse-json` block** — add a `giveMeABar` call and `bar.Increment()` inside the generation loop:

  ```go
  if runningParseJson {
      // Parse the string object content
      var parseMap map[string]any
      if err := json.Unmarshal([]byte(parseJson), &parseMap); err != nil {
          return fmt.Errorf("failed to parse JSON from the provided --parse-json '%w'", err)
      }

      // Progress bar: total = generate (one tick per root object produced)
      bar := giveMeABar("parse-json", int64(generate), nil, mpbHandler)

      // Process the parsed map
      mocker := mocker.New()
      parseMaps := make([]map[string]any, generate)
      for i := range generate {
          cpParseMap := deepcopy.Copy(parseMap).(map[string]any)
          if err := processJsonMap(cpParseMap, mocker); err != nil {
              bar.Abort(false)
              return fmt.Errorf("%w", err)
          }
          parseMaps[i] = deepcopy.Copy(cpParseMap).(map[string]any)
          bar.Increment()
      }

      // Sanitize the parsed map
      for i := range parseMaps {
          sanitizeJsonMap(parseMaps[i])
      }

      // Print the result to stdout
      var prettyJSON []byte
      var err error
      if generate == 1 {
          prettyJSON, err = json.MarshalIndent(parseMaps[0], "", "  ")
      } else {
          prettyJSON, err = json.MarshalIndent(parseMaps, "", "  ")
      }
      if err != nil {
          return fmt.Errorf("error marshalling JSON '%w'", err)
      }
      fmt.Fprintf(opts.Out, "%s\n", prettyJSON)
  }
  ```

  5h. **New `--parse-json-file` block** — replace the entire `if runningParseFiles { ... }` block with:

  ```go
  if runningParseJsonFile {
      // Verify the file exists
      if _, err := os.Stat(parseJsonFile); os.IsNotExist(err) {
          return fmt.Errorf("template file not found: '%s'", parseJsonFile)
      }

      // Read the template file
      templateFileContent, err := os.ReadFile(parseJsonFile)
      if err != nil {
          return fmt.Errorf("failed to read --parse-json-file '%w'", err)
      }

      // Parse the template file content
      var parseMap map[string]any
      if err = json.Unmarshal(templateFileContent, &parseMap); err != nil {
          return fmt.Errorf("failed to parse JSON from --parse-json-file '%w'", err)
      }

      // Determine output file path: same directory as the template file, .template.json → .json
      outName := strings.Replace(filepath.Base(parseJsonFile), ".template.json", ".json", 1)
      outPath := filepath.Join(filepath.Dir(parseJsonFile), outName)

      // Always delete the output file before writing a new one
      _ = os.Remove(outPath)

      // Progress bar: total = generate (one tick per root object produced); outPath for file-size display
      bar := giveMeABar(filepath.Base(parseJsonFile), int64(generate), &outPath, mpbHandler)

      // Process the parsed map
      mocker := mocker.New()
      parseMaps := make([]map[string]any, generate)
      for i := range generate {
          cpParseMap := deepcopy.Copy(parseMap).(map[string]any)
          if err := processJsonMap(cpParseMap, mocker); err != nil {
              bar.Abort(false)
              return fmt.Errorf("%w", err)
          }
          parseMaps[i] = deepcopy.Copy(cpParseMap).(map[string]any)
          bar.Increment()
      }

      // Sanitize the parsed maps
      for i := range parseMaps {
          sanitizeJsonMap(parseMaps[i])
      }

      // Marshal the result
      var prettyJSON []byte
      if generate == 1 {
          prettyJSON, err = json.MarshalIndent(parseMaps[0], "", "  ")
      } else {
          prettyJSON, err = json.MarshalIndent(parseMaps, "", "  ")
      }
      if err != nil {
          return fmt.Errorf("error marshalling JSON '%w'", err)
      }

      // Write the output file
      if err = os.WriteFile(outPath, prettyJSON, 0644); err != nil {
          return fmt.Errorf("failed to write result to '%s': '%w'", outPath, err)
      }
  }
  ```

  Note on bar placement for `--parse-json-file`: `outPath` is computed and the old file removed **before** the bar is created, so the bar's file-size decorator starts reading from a clean state. The size will show `[N/A]` during generation (file not yet written) and reflect the actual output size only if the bar is still rendering after the write — which is fine since `mpbHandler.Wait()` holds the display open until all bars complete.

- [x] Step 6: **Update the `Long` help text in `NewMockCmd` in `cmd/mock.go`**

  Rewrite the `Long` field to remove all references to `--parse-files`, `--preserve-folder-structure`, the `out/` directory, and the filename-bracket convention. Add documentation for `--parse-json-file`. The new Long text should:

  - List `--parse-str`, `--parse-json`, and `--parse-json-file` as the three parse modes.
  - Document that `--generate` is available for both `--parse-json` and `--parse-json-file`.
  - Explain that the output for `--parse-json-file` is written next to the template file (e.g. `employees.template.json` → `employees.json` in the same directory).
  - Remove the multi-file/glob/folder-structure examples.
  - Add a new example: `ktns mock --parse-json-file "path/to/employees.template.json"` and `ktns mock --parse-json-file "path/to/employees.template.json" --generate 5`.
  - Remove the example line `ktns mock --parse-files "*.template.json"` and `ktns mock --parse-files "test/templates" --preserve-folder-structure`.

- [x] Step 7: **Update unit tests in `cmd/mock_test.go`**

  7a. In `TestExtractDigitInBrackets_ValidInputs`, remove the two test cases that use `inputPlace: "file"` (test 4 and test 5), since the `"file"` branch no longer exists in `extractDigitInBrackets`.

  7b. In `TestExtractDigitInBrackets_InvalidInputs`, remove all test cases that use `inputPlace: "file"` (tests 16 through 21), since those code paths are deleted.

  7c. No other unit test functions in `mock_test.go` reference `--parse-files` or `--preserve-folder-structure`; they all test internal helpers that are unchanged.

- [x] Step 8: **Update E2E tests in `cmd/mock_e2e_test.go`**

  8a. **Delete `TestCLIShouldRaiseError_InvalidUseOfParseFiles`** — this test checks the multi-arg error for `--parse-files`, which no longer applies.

  8b. **Delete `TestCLIShouldRaiseError_PreserveFolderStructureFlagInvalidUse`** — the `--preserve-folder-structure` flag is gone.

  8c. **Update `TestCLIShouldRaiseError_MultipleParseFlags`** — replace every occurrence of `"--parse-files"` with `"--parse-json-file"`. Update the expected error string to: `"provide only one of the three options: --parse-json, --parse-json-file or --parse-str"`.

  8d. **Update `TestCLIShouldRaiseError_GenerateFlagInvalidUse`** — remove the test case that pairs `--generate` with `--parse-files`. The `--generate` flag is now valid with `--parse-json-file`, so that combination must NOT be present in the invalid-use test. Update the expected error string to: `"--generate option is only available when using --parse-json or --parse-json-file"`. The remaining test case (`--generate` with `--parse-str`) remains.

  8e. **Add `TestCLIShouldRaiseError_ParseJsonFileNotFound`** — new test that passes a non-existent file path to `--parse-json-file` and asserts the error message contains `"template file not found"`.

  8f. **Add `TestCLIShouldParseJsonFile`** — new E2E test that:
    - Creates a temporary directory using `t.TempDir()` (or `os.MkdirTemp`).
    - Writes a minimal `.template.json` file into that directory (e.g. `employee.template.json` with `{"name": "{{ Person.name }}"}`).
    - Executes `mock --parse-json-file <path-to-template>`.
    - Asserts no error is returned.
    - Asserts the output file `employee.json` was created in the same directory.
    - Reads `employee.json` and asserts it contains a valid JSON object with a `"name"` key.
    - Cleans up (temp dir handles this automatically).

  8g. **Add `TestCLIShouldParseJsonFile_WithGenerate`** — same setup as 8f but calls `mock --parse-json-file <path> --generate 3`. Asserts the output file is a JSON array of 3 objects.

  8h. **Add `TestCLIShouldParseJsonFile_DeletesPreviousOutput`** — same setup but creates a pre-existing `employee.json` in the temp dir with stale content, then runs the command, and asserts the output file was replaced (stale content is gone).

- [x] Step 9: **Update `README.md`**

  9a. In the **Flags** subsection under `## mock`, replace:
    - `` `--parse-files`: Pass a path, directory, or glob pattern... `` → `` `--parse-json-file`: Pass a path to a single `.template.json` file. The mock data will be generated based on this file and written alongside it. ``
    - Remove `` `--preserve-folder-structure`: If set... `` bullet entirely.
    - Update `` `--generate`: ... (only available for `--parse-json`) `` → ``(available for `--parse-json` and `--parse-json-file`)``.

  9b. Replace the entire **`--parse-files`** example section with a new **`--parse-json-file`** section showing:
    - A template file `employee.template.json`.
    - The command `ktns mock --parse-json-file employee.template.json`.
    - The resulting `employee.json` written in the same directory.
    - An example with `--generate 3` producing an array.

  9c. Delete the entire **`--preserve-folder-structure`** section (heading, explanation, code blocks, and folder-tree diagrams).

  9d. In **Generating multiple values / Root objects**, remove the paragraph "When using `--parse-files`, specify the desired number of root objects in the template file's name..." and its associated example. Add a note that `--parse-json-file` uses `--generate` exactly as `--parse-json` does.

  9e. In the **Development Details / Subcommand Files** table, update the `mock.go` row description from `"...concurrency for --parse-files"` to `"...single-threaded file I/O for --parse-json-file"`.

  9f. Delete the **Concurrency in `--parse-files`** subsection entirely (the paragraph describing goroutines, WaitGroup, Mutex, and per-goroutine mocker).

  9g. In the **Preservation of folder structure** section, delete it entirely.

  9h. Verify all example command lines in the README no longer reference `--parse-files` or `--preserve-folder-structure`.

- [x] Step 10: **Verify `go.mod` / dependency cleanup**

  The `github.com/vbauerster/mpb/v8` and `github.com/vbauerster/mpb/v8/decor` imports are **kept** (used by the refactored `giveMeABar`). The only dependency change is the removal of `"sync"`, which is a standard library package and has no `go.mod` entry.

  Run `go mod tidy` to confirm the module graph is consistent and no other orphaned transitive dependency was introduced or left behind.

## Notes

- The `objKeyNumberRegex` package-level variable (`^[^\[\]\s]+\[(\d+)\]$`) must be kept — it is still used by the `"object"` branch of `extractDigitInBrackets`, which is called by `processJsonMap` for inner-object bracket parsing.
- `giveMeABar` is kept and refactored (Step 4a). Its new signature is `giveMeABar(taskName string, total int64, outPath *string, mpbHandler *mpb.Progress) *mpb.Bar`. The `labels []string` parameter and step-label decorator are removed. `total` is always `int64(generate)`, giving one progress tick per generated root object instead of one tick per fixed processing stage.
- For `--parse-json`, pass `nil` as `outPath` — the bar will show `[N/A]` for the file-size decorator (no output file is written).
- For `--parse-json-file`, pass `&outPath` — the bar will attempt `os.Stat(*outPath)` on each render tick. During generation the file does not yet exist so it shows `[N/A]`; after writing it would reflect the actual file size if the bar is still rendering.
- `mpbHandler.Wait()` at the bottom of RunE ensures the terminal output is flushed and the bar is fully rendered before the command returns. This applies to both modes.
- The `sanitizeKeyWithBrackets` function and `sanitizeJsonMap` are not affected by this refactor; they are still needed for object-key cleanup.
- The `processStr`, `interpretString`, `extractMockMethod`, and `processJsonMap` functions are completely unchanged.
- `--parse-json-file` accepts a raw file path (not a glob). There is no need for glob expansion or recursive directory walking. The `os.Stat` check is sufficient to confirm existence.
- The `os.Remove(outPath)` call before writing is intentional per requirements and should not propagate an error if the file does not yet exist (hence the `_ =` discard).
- `filepath.Dir(parseJsonFile)` will return `"."` when the caller passes a bare filename with no directory component; `filepath.Join(".", outName)` correctly resolves to `outName` in the current directory, which is acceptable behavior.
- E2E tests for `--parse-json-file` must use `t.TempDir()` (or equivalent) to avoid polluting the repo working directory with generated files. Do not write test template files into the repo source tree.
- The `--generate` validation guard changes from `generate > 1 && !runningParseJson` to `generate > 1 && !runningParseJson && !runningParseJsonFile`. This means `--generate 1` (the default) remains valid for all three modes and will not trigger an error, which preserves backward compatibility for the `--parse-str` mode that does not use `--generate` at all.
- When `generate == 1`, output is a single JSON object (not an array), matching the existing `--parse-json` behavior exactly — no structural difference as required.
