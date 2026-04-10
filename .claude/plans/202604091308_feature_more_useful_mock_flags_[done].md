# Feature: More Useful `mock` Flags

> Created: 2026-04-09 13:08

## Summary

This feature strengthens the `mock` command in three directions: (1) enforces a naming constraint on `--parse-json-file` so that template files must end with `.template.json`; (2) adds new output-routing flags (`--to-stdout`, `--to-stdout-prettify`, `--to-json-file`) that let the caller decide independently whether the result goes to stdout (as JSON or CSV) or to a file (as JSON), replacing the current hard-wired behaviour where `--parse-json` always outputs to stdout and `--parse-json-file` always writes to a file; (3) adds `--no-progress` to suppress the mpb progress bar; and (4) updates all documentation accordingly. The `--parse-str` path is left unchanged — it always prints to stdout.

## Affected Files

- `cmd/mock.go` — add new flags, add validation logic, refactor output routing, add CSV serialisation helper
- `cmd/mock_e2e_test.go` — add E2E tests for the new constraint, the new flags, and their interactions
- `README.md` — update Flags table, add examples for `--to-stdout`, `--to-stdout-prettify`, `--to-json-file`, `--no-progress`

## Steps

- [x] Step 1: Add `.template.json` constraint for `--parse-json-file`

  In `NewMockCmd`, after the existing `os.Stat` check (line 188 area), add a new validation block:
  ```
  if !strings.HasSuffix(filepath.Base(parseJsonFile), ".template.json") {
      return fmt.Errorf("--parse-json-file requires the template filename to end with '.template.json', got '%s'", filepath.Base(parseJsonFile))
  }
  ```
  This check must run before any file I/O. Place it immediately after `parseJsonFile != ""` is confirmed to be the active mode (i.e., after the `parseCheck` block but before `os.Stat`).

- [x] Step 2: Register the four new flags in `NewMockCmd`

  Add the following flag declarations at the bottom of `NewMockCmd`, alongside the existing `mockCmd.Flags()` calls:
  ```go
  mockCmd.Flags().String("to-stdout", "", "output result to stdout as 'as-json' or 'as-csv'")
  mockCmd.Flags().Bool("to-stdout-prettify", false, "prettify the stdout output (only valid with --to-stdout)")
  mockCmd.Flags().String("to-json-file", "", "output result as JSON to a file; optional filename argument")
  mockCmd.Flags().Bool("no-progress", false, "suppress the progress bar")
  ```
  Note: `--to-json-file` is declared as a `String` flag, not a `Bool`, so the user can optionally pass a filename value. An empty string means "use the default filename". Because cobra's `String` flag requires an explicit value (`--to-json-file=""` or `--to-json-file myfile.json`), this is correct — the user sets it with or without a filename, or leaves the flag entirely absent.

- [x] Step 3: Read new flag values at the top of `RunE`

  At the top of the `RunE` function, alongside the existing `GetBool`/`GetString` calls, add:
  ```go
  toStdout, _         := cmd.Flags().GetString("to-stdout")
  toStdoutPrettify, _ := cmd.Flags().GetBool("to-stdout-prettify")
  toJsonFile, _       := cmd.Flags().GetString("to-json-file")
  toJsonFileSet       := cmd.Flags().Changed("to-json-file")  // true even if value is ""
  noProgress, _       := cmd.Flags().GetBool("no-progress")
  ```
  `cmd.Flags().Changed("to-json-file")` distinguishes "flag was not passed at all" from "flag was passed with no value" (which cobra sets to `""`).

- [x] Step 4: Add new flag validation rules

  After the existing `parseCheck` / `--generate` validations, add these new validation blocks in order:

  a. `--to-stdout` must be either `"as-json"` or `"as-csv"` when non-empty:
  ```go
  if toStdout != "" && toStdout != "as-json" && toStdout != "as-csv" {
      return fmt.Errorf("--to-stdout only accepts 'as-json' or 'as-csv', got '%s'", toStdout)
  }
  ```

  b. `--to-stdout-prettify` is only valid when `--to-stdout` is also set:
  ```go
  if toStdoutPrettify && toStdout == "" {
      return fmt.Errorf("--to-stdout-prettify requires --to-stdout to be set")
  }
  ```

  c. `--to-stdout` and `--to-json-file` may not be used together (they are alternative output destinations, but actually they CAN coexist — they are independent routing decisions). Re-read the feature description: both can be used at the same time. Remove this restriction — no mutual-exclusion error needed here.

  d. `--parse-str` must not be combined with `--to-stdout` or `--to-json-file`:
  ```go
  if runningParseStr && (toStdout != "" || toJsonFileSet) {
      return fmt.Errorf("--parse-str always outputs to stdout; --to-stdout and --to-json-file are not available with --parse-str")
  }
  ```

  e. When using `--parse-json` or `--parse-json-file`, the user MUST specify at least one of `--to-stdout` or `--to-json-file`:
  ```go
  if !runningParseStr && toStdout == "" && !toJsonFileSet {
      return fmt.Errorf("--parse-json and --parse-json-file require at least one output flag: --to-stdout or --to-json-file")
  }
  ```

- [x] Step 5: Refactor the `--parse-json` output routing

  The current `--parse-json` block (lines 143–183) marshals the result and prints it to `opts.Out`. Replace the entire output section (after `sanitizeJsonMap` loop) with a call to the new shared output helper (see Step 7).

  New behaviour: call `routeOutput(parseMaps, generate, toStdout, toStdoutPrettify, toJsonFileSet, toJsonFile, "parse-json", "", opts.Out)` — see Step 7 for that function's signature and logic. Since validation in Step 4 guarantees at least one output flag is set, there is no fallback default path needed here.

  Also: suppress or pass `noProgress` to the `giveMeABar` call so that when `noProgress` is `true` the bar is skipped (see Step 6).

- [x] Step 6: Refactor the `--parse-json-file` output routing

  The current `--parse-json-file` block (lines 186–247) computes `outPath`, deletes the old output file, marshals, and writes the file. After this feature the output destination is no longer fixed to that file.

  Changes to this block:
  - Keep the `.template.json` filename constraint (Step 1).
  - Keep `os.Stat` existence check.
  - Keep `json.Unmarshal` of the template file.
  - Remove the hard-wired `outPath` computation and `os.Remove` — this is now the responsibility of `routeOutput`.
  - Pass the default filename hint (`strings.Replace(filepath.Base(parseJsonFile), ".template.json", ".json", 1)` combined with `filepath.Dir(parseJsonFile)`) to `routeOutput` as the `defaultFileDir` and `defaultFileName` arguments.
  - For the progress bar: wrap the `giveMeABar` call in `if !noProgress { ... }` — if `noProgress` is true, use a nil/no-op bar. The simplest approach is to create a small helper `maybeBar` or simply check `noProgress` before calling `giveMeABar` and using a `*mpb.Bar` stub. Since `mpb.Bar` methods are called unconditionally (`bar.Increment()`, `bar.Abort()`), the cleanest solution is: when `noProgress`, use `mpb.New(mpb.WithOutput(io.Discard))` — this creates a real but silent progress handler. Pass `io.Discard` for the mpb output stream.
  - Call `routeOutput(parseMaps, generate, toStdout, toStdoutPrettify, toJsonFileSet, toJsonFile, "parse-json-file", defaultFilePath, opts.Out)` after the processing loop.

  Amend `mpbHandler` initialisation to honour `noProgress`:
  ```go
  mpbOut := opts.Out  // or os.Stdout for the real case
  if noProgress {
      mpbOut = io.Discard
  }
  mpbHandler := mpb.New(
      mpb.WithWidth(60),
      mpb.WithOutput(mpbOut),
      mpb.WithAutoRefresh(),
  )
  ```
  Note: `mpb.WithOutput(io.Discard)` suppresses all bar rendering without changing the calling code.

  Also note: the current code uses `mpb.WithOutput(os.Stdout)` hardcoded rather than `opts.Out`. This should be corrected to use `os.Stderr` (or keep `os.Stdout`) consistently. For now, leave the progress bar writing to `os.Stdout` (as it currently does for the real binary), but pass `io.Discard` when `noProgress` is set.

- [x] Step 7: Add `routeOutput` helper function

  Add a new package-private function in `cmd/mock.go`:

  ```go
  // routeOutput handles all output routing for --parse-json and --parse-json-file results.
  // parseMaps is the slice of processed root objects.
  // generate is the --generate count (used to decide single-object vs array).
  // toStdout is "" | "as-json" | "as-csv".
  // toStdoutPrettify controls indented vs compact stdout output.
  // toJsonFileSet indicates --to-json-file was passed (even with empty value).
  // toJsonFileValue is the optional filename given with --to-json-file.
  // source is "parse-json" or "parse-json-file".
  // defaultFilePath is the full default output path (used when source is "parse-json-file" and no explicit filename was given).
  // out is the writer for stdout.
  func routeOutput(
      parseMaps []map[string]any,
      generate int,
      toStdout string,
      toStdoutPrettify bool,
      toJsonFileSet bool,
      toJsonFileValue string,
      source string,
      defaultFilePath string,
      out io.Writer,
  ) error
  ```

  Behaviour:

  **Determine whether to write to stdout:**
  - If `toStdout != ""`: serialise and print to `out`.
    - `"as-json"`:
      - If `toStdoutPrettify`: use `json.MarshalIndent(data, "", "  ")`.
      - Otherwise: use `json.Marshal(data)` (compact).
      - `data` is `parseMaps[0]` if `generate == 1`, else `parseMaps`.
      - `fmt.Fprintf(out, "%s\n", bytes)`
    - `"as-csv"`:
      - Flatten the objects to CSV using `marshalAsCSV(parseMaps, toStdoutPrettify)` (see Step 8).
      - `fmt.Fprintf(out, "%s\n", csvStr)`

  **Determine whether to write to a JSON file:**
  - If `toJsonFileSet`:
    - Resolve the output filepath:
      - If `toJsonFileValue != ""` (and not `"_use_default_"`): use `toJsonFileValue` as-is.
      - Else if `source == "parse-json-file"`: use `defaultFilePath`.
      - Else (`source == "parse-json"`): use `filepath.Join(executableDir(), "output.json")` where `executableDir()` is a small helper that calls `os.Executable()` and returns its directory (see Step 9).
    - Delete any existing file at that path: `os.Remove(resolvedPath)` (ignore error).
    - Marshal: `json.MarshalIndent(data, "", "  ")` (files are always pretty-printed for readability).
    - Write: `os.WriteFile(resolvedPath, bytes, 0644)`.
    - Return error on failure.

  Both stdout and file outputs can happen simultaneously — they are independent; apply both if both conditions are true. There is no fallback default behaviour: validation in Step 4 guarantees at least one of these two paths is always active.

- [x] Step 8: Add `marshalAsCSV` helper function

  Add in `cmd/mock.go`:

  ```go
  // marshalAsCSV serialises a slice of flat maps to CSV.
  // If prettify is true, columns are padded to equal width for readability.
  // If prettify is false, output is standard compact RFC 4180 CSV.
  // Column order is determined by the sorted union of all keys across all rows.
  // Values are always coerced to strings.
  func marshalAsCSV(parseMaps []map[string]any, prettify bool) (string, error)
  ```

  Implementation notes:
  - Collect all unique keys across all `parseMaps` entries, sort them alphabetically to produce stable column order.
  - First row is the header (the sorted keys).
  - Each subsequent row is the values for each map in order, using `fmt.Sprintf("%v", val)` for any non-string type.
  - For standard (non-prettified) CSV: use the stdlib `encoding/csv` package — create a `strings.Builder`, wrap with `csv.NewWriter`, write header and rows, call `w.Flush()`.
  - For prettified CSV: compute maximum width per column across all rows (header included), pad each cell with spaces to that width, separate columns with `" | "`, separate rows with newlines. Add a separator line (`---`) between the header and data rows.
  - Return `(string, error)`.

- [x] Step 9: Add `executableDir` helper function

  Add in `cmd/mock.go` (or `cmd/utils.go`):

  ```go
  // executableDir returns the directory of the running ktns binary.
  // Used to resolve the default output path for --to-json-file when parsing from stdin.
  func executableDir() (string, error) {
      exe, err := os.Executable()
      if err != nil {
          return "", fmt.Errorf("could not determine executable path: %w", err)
      }
      return filepath.Dir(exe), nil
  }
  ```

  In `routeOutput`, when computing the default `"parse-json"` output file, call this and propagate the error upward.

- [x] Step 10: Update the `Long` description and examples in `NewMockCmd`

  The `Long` field of `mockCmd` contains inline documentation and examples. Update it to:
  - Remove the line "When using --parse-json-file, the output file is written alongside the template file."
  - Add a note that `--parse-json-file` requires the filename to end with `.template.json`.
  - Add descriptions for `--to-stdout`, `--to-stdout-prettify`, `--to-json-file`, and `--no-progress`.
  - Add CLI examples:
    ```
    ktns mock --parse-json '{ "name": "{{ Person.name }}" }' --to-stdout as-json
    ktns mock --parse-json '{ "name": "{{ Person.name }}" }' --to-stdout as-csv
    ktns mock --parse-json '{ "name": "{{ Person.name }}" }' --to-stdout as-json --to-stdout-prettify
    ktns mock --parse-json '{ "name": "{{ Person.name }}" }' --to-json-file
    ktns mock --parse-json '{ "name": "{{ Person.name }}" }' --to-json-file mydata.json
    ktns mock --parse-json-file "path/to/employees.template.json" --to-stdout as-csv
    ktns mock --parse-json-file "path/to/employees.template.json" --to-json-file
    ktns mock --parse-json-file "path/to/employees.template.json" --to-json-file myout.json
    ktns mock --parse-json-file "path/to/employees.template.json" --no-progress
    ```

- [x] Step 11: Add E2E tests for the new `.template.json` constraint

  In `cmd/mock_e2e_test.go`, add a test method to `MockCmdE2ETestSuite`:

  ```
  TestCLIShouldRaiseError_ParseJsonFileInvalidExtension
  ```
  - Pass `--parse-json-file` with a file path ending in `.json` (not `.template.json`) — e.g. `/tmp/employee.json`.
  - Assert `err` is non-nil.
  - Assert `err.Error()` contains `"requires the template filename to end with '.template.json'"`.

- [x] Step 12: Add E2E tests for `--to-stdout` validation

  In `cmd/mock_e2e_test.go`, add:

  `TestCLIShouldRaiseError_ToStdoutInvalidValue`
  - Pass `--parse-json '{"name": "{{ Person.name }}"}'` `--to-stdout invalid`.
  - Assert error contains `"only accepts 'as-json' or 'as-csv'"`.

  `TestCLIShouldRaiseError_ToStdoutPrettifyWithoutToStdout`
  - Pass `--parse-json '{"name": "{{ Person.name }}"}'` `--to-stdout-prettify`.
  - Assert error contains `"requires --to-stdout to be set"`.

  `TestCLIShouldRaiseError_ParseStrWithToStdout`
  - Pass `--parse-str 'Hello'` `--to-stdout as-json`.
  - Assert error contains `"--parse-str always outputs to stdout"`.

  `TestCLIShouldRaiseError_ParseStrWithToJsonFile`
  - Pass `--parse-str 'Hello'` `--to-json-file`.
  - Assert error contains `"--parse-str always outputs to stdout"`.

  `TestCLIShouldRaiseError_ParseJsonWithNoOutputFlag`
  - Pass `--parse-json '{"name": "{{ Person.name }}"}'` with neither `--to-stdout` nor `--to-json-file`.
  - Assert error contains `"require at least one output flag"`.

  `TestCLIShouldRaiseError_ParseJsonFileWithNoOutputFlag`
  - Pass `--parse-json-file <path>` with neither `--to-stdout` nor `--to-json-file`.
  - Assert error contains `"require at least one output flag"`.

- [x] Step 13: Add E2E tests for `--to-stdout as-json`

  `TestCLIShouldOutputToStdout_AsJson_ParseJson`
  - Run `mock --parse-json '{"name": "{{ Person.name }}"}' --to-stdout as-json`.
  - Assert no error.
  - Assert stdout is valid compact JSON (parse with `json.Unmarshal`, confirm no whitespace indentation).

  `TestCLIShouldOutputToStdout_AsJson_ParseJson_Prettify`
  - Run with `--to-stdout as-json --to-stdout-prettify`.
  - Assert stdout contains `"  "` indentation (indented JSON).

  `TestCLIShouldOutputToStdout_AsJson_ParseJsonFile`
  - Create a temp `.template.json` file.
  - Run `mock --parse-json-file <path> --to-stdout as-json`.
  - Assert stdout is valid JSON.
  - Assert no output file was created at the default `.json` path (since `--to-json-file` was not passed).

- [x] Step 14: Add E2E tests for `--to-stdout as-csv`

  `TestCLIShouldOutputToStdout_AsCsv_ParseJson`
  - Run `mock --parse-json '{"name": "{{ Person.name }}", "age": "{{ Number.number::{18}:{80} }}"}' --to-stdout as-csv`.
  - Assert stdout first line is `age,name` (sorted header, compact CSV).
  - Assert stdout has exactly 2 lines total (header + 1 data row).

  `TestCLIShouldOutputToStdout_AsCsv_ParseJson_Prettify`
  - Run with `--to-stdout as-csv --to-stdout-prettify`.
  - Assert stdout contains ` | ` column separator.
  - Assert stdout contains a `---` separator line.

  `TestCLIShouldOutputToStdout_AsCsv_ParseJson_WithGenerate`
  - Run with `--generate 3 --to-stdout as-csv`.
  - Assert stdout has 4 lines (1 header + 3 data rows).

- [x] Step 15: Add E2E tests for `--to-json-file`

  `TestCLIShouldOutputToJsonFile_DefaultName_ParseJson`
  - Run `mock --parse-json '{"name": "{{ Person.name }}"}' --to-json-file`.
  - Assert no error.
  - Assert a file named `output.json` exists alongside the binary (skip this assertion in test environments where `os.Executable()` returns a temp path — assert the file was created by checking the returned error is nil; in tests the path may be under `/tmp/go-test*`). Alternatively, accept that the test just asserts no error and that stdout is empty.

  `TestCLIShouldOutputToJsonFile_ExplicitName_ParseJson`
  - Use a temp dir; run with `--to-json-file <tmpdir>/result.json`.
  - Assert no error.
  - Assert `result.json` exists in tmpdir and contains valid JSON.

  `TestCLIShouldOutputToJsonFile_DefaultName_ParseJsonFile`
  - Create temp `employee.template.json`.
  - Run `mock --parse-json-file <path> --to-json-file`.
  - Assert the default file `employee.json` was created alongside the template.

  `TestCLIShouldOutputToJsonFile_ExplicitName_ParseJsonFile`
  - Create temp `employee.template.json`.
  - Run `mock --parse-json-file <path> --to-json-file <tmpdir>/out.json`.
  - Assert `out.json` exists at the explicit path.
  - Assert `employee.json` does NOT exist (since explicit path overrides default).

- [x] Step 16: Add E2E tests for `--no-progress`

  `TestCLIShouldSuppressProgressBar_ParseJson`
  - Run `mock --parse-json '{"n": "{{ Person.name }}"}' --to-stdout as-json --no-progress`.
  - Assert no error.
  - Assert stdout contains valid JSON (i.e., output is not corrupted by bar characters).

  `TestCLIShouldSuppressProgressBar_ParseJsonFile`
  - Create temp template; run `mock --parse-json-file <path> --no-progress`.
  - Assert no error, output file exists.

- [x] Step 17: Update `README.md`

  In the `## mock` section:

  a. Under `### Flags`, add entries for:
  - `--to-stdout <as-json|as-csv>`: Output result to stdout as a JSON object or a CSV table. Must be used with `--parse-json` or `--parse-json-file`. Use `--to-stdout-prettify` to format for readability.
  - `--to-stdout-prettify`: Prettify the stdout output. Only valid with `--to-stdout`.
  - `--to-json-file [filename]`: Write result as JSON to a file. If no filename is given, defaults to `output.json` beside the binary (for `--parse-json`) or to the template name without `.template` (for `--parse-json-file`). If a filename is given, it is used as-is.
  - `--no-progress`: Suppress the progress bar.

  b. Update the description of `--parse-json-file` to state that the file must end with `.template.json`.

  c. Update the existing `--parse-json-file` example section to clarify:
  - Default behaviour (no new flags) still writes `.json` file alongside template.
  - Add new examples using `--to-stdout`, `--to-json-file`, and `--no-progress`.

  d. Update the `--parse-json` examples to show that by default (no new flags) it outputs to stdout, and the new flags redirect/format that output.

## Notes

- **`--to-json-file` as a `String` flag with optional value**: Cobra does not natively support optional-value string flags (`--flag` with no value). The cleanest approach is to declare it as `String` and use `cmd.Flags().Changed("to-json-file")` to detect presence. When the user types `--to-json-file` without a value, cobra will error ("flag needs an argument"). To work around this, document that the user should pass `--to-json-file ""` to use the default, OR declare the flag with a default of `"_default_"` (sentinel) and detect it differently. The most user-friendly approach: declare `to-json-file` as a `String` with default `""` and use `Changed`. Users must do `--to-json-file ""` or `--to-json-file myfile.json`. The developer should verify this UX and adjust if cobra supports `NoOptDefVal` for String flags (which allows `--flag` without value to use a sentinel). **Recommended approach**: set `mockCmd.Flags().Lookup("to-json-file").NoOptDefVal = "_use_default_"` so that `--to-json-file` alone sets value to `"_use_default_"` while `--to-json-file myfile.json` sets it to `"myfile.json"`. In `routeOutput`, treat `"_use_default_"` as "no explicit filename given".

- **CSV for non-flat objects**: The `marshalAsCSV` helper should document that it works best with flat (one-level-deep) JSON objects. Nested objects/arrays will be serialised as their `%v` string representation, which may not be ideal. No error needs to be raised — just apply `fmt.Sprintf("%v", val)`. Add a note to the README warning about this limitation.

- **Progress bar and `io.Discard`**: Using `mpb.WithOutput(io.Discard)` cleanly suppresses all bar output without altering the calling code that calls `bar.Increment()` and `bar.Abort()`. This is safe because mpb writes to its output writer asynchronously.

- **`executableDir` in tests**: `os.Executable()` in tests returns the test binary path (somewhere in `/tmp`). The test for `TestCLIShouldOutputToJsonFile_DefaultName_ParseJson` should not hardcode the expected directory, but instead resolve it the same way the implementation does, or just verify the error is nil and a file exists at the path the function reports.

- **CSV import**: `encoding/csv` is a stdlib package — no new dependency needed.

- **Test file vs stdout duplication**: When both `--to-stdout` and `--to-json-file` are used together, both outputs should be produced. No test currently covers this combination but it should be implicitly covered by the E2E tests if both conditions are asserted in the same run.
