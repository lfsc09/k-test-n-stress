# Fix: Replace `os.Executable()` with `os.Getwd()` and Document `--to-json-file`/`--to-csv-file` Syntax

> Created: 2026-04-14 17:46

## Summary

Two related fixes. First, `executableDir()` in `cmd/mock.go` currently calls `os.Executable()` to determine where to place the default `output.json` / `output.csv` file. Under `go run .`, the binary is compiled into a temp directory (e.g. `/tmp/go-runXXXXXX/exe/`) that is deleted when the process exits, causing the output file to be immediately lost. The fix renames the function to `workingDir()` and replaces its body with a simple `os.Getwd()` call, so the default output lands in the current working directory where the user invoked the command. All four call sites (two in `resolveOutputPaths`, two in `routeOutput`) must be updated. Second, because both flags use `NoOptDefVal`, an explicit filename can only be given using `=` syntax (e.g. `--to-json-file=myfile.json`, not `--to-json-file myfile.json`). The existing docs contain incorrect space-separated examples in both the `Long` help text and `README.md`. These must be corrected and the default path description in both places must be updated from "beside the binary" to "in the current working directory".

## Affected Files

- `cmd/mock.go` — rename `executableDir()` to `workingDir()`, rewrite its body, update all four call sites
- `cmd/mock_e2e_test.go` — update two test names/comments that say "beside binary" to say "in the current working directory (CWD)"
- `README.md` — update default output path description and any space-separated `--to-json-file` / `--to-csv-file` examples to use `=` syntax
- The `Long` description string in `NewMockCmd` in `cmd/mock.go` — same corrections

## Steps

- [x] Step 1: Rename and rewrite `executableDir()` in `cmd/mock.go`

  Locate lines 1252–1260 of `cmd/mock.go`:

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

  Replace with:

  ```go
  // workingDir returns the current working directory.
  // Used to resolve the default output path for --to-json-file and --to-csv-file
  // when no explicit filename is provided and the source is not --parse-json-file.
  func workingDir() (string, error) {
      dir, err := os.Getwd()
      if err != nil {
          return "", fmt.Errorf("could not determine working directory: %w", err)
      }
      return dir, nil
  }
  ```

  Note: the `os` import is already present; no import changes are needed. The `filepath` import is still used elsewhere so it must not be removed.

- [x] Step 2: Update `resolveOutputPaths` — JSON default branch (line ~1361)

  In `resolveOutputPaths`, the `default` case of the `toJsonFileSet` switch currently reads:

  ```go
  dir, e := executableDir()
  if e != nil {
      return "", "", e
  }
  jsonPath = filepath.Join(dir, "output.json")
  ```

  Change `executableDir()` to `workingDir()`.

- [x] Step 3: Update `resolveOutputPaths` — CSV default branch (line ~1376)

  In the `toCsvFileSet` switch `default` case directly below, change:

  ```go
  dir, e := executableDir()
  ```

  to:

  ```go
  dir, e := workingDir()
  ```

- [x] Step 4: Update `routeOutput` — JSON default branch (line ~1918)

  In `routeOutput`, the file-output block's `default` case reads:

  ```go
  // source == "parse-json": use output.json beside the binary
  dir, err := executableDir()
  if err != nil {
      return err
  }
  resolvedPath = filepath.Join(dir, "output.json")
  ```

  Change the comment and the function call:

  ```go
  // source == "parse-json": use output.json in the current working directory
  dir, err := workingDir()
  if err != nil {
      return err
  }
  resolvedPath = filepath.Join(dir, "output.json")
  ```

- [x] Step 5: Update `routeOutput` — CSV default branch (line ~1954)

  In the CSV file-output block's `default` case, change:

  ```go
  // source == "parse-json": use output.csv beside the binary
  dir, err := executableDir()
  ```

  to:

  ```go
  // source == "parse-json": use output.csv in the current working directory
  dir, err := workingDir()
  ```

- [x] Step 6: Update the `Long` help text in `NewMockCmd` (`cmd/mock.go`)

  There are two things to fix in the `Long` string (lines ~670–698):

  **6a. Fix the default path description for `--to-json-file` and `--to-csv-file`.**

  Change (line ~671):
  ```
  If no filename is given, defaults to output.json beside the binary (for --parse-json)
  ```
  To:
  ```
  If no filename is given, defaults to output.json in the current working directory (for --parse-json)
  ```

  Change (line ~673):
  ```
  Note: --to-json-file requires an explicit value or use --to-json-file "" for the default.
  ```
  To:
  ```
  Note: when specifying a filename, use = syntax: --to-json-file=myfile.json
  ```

  Change (line ~675):
  ```
  or to the template name without .template with a .csv extension (for --parse-json-file).
  Note: --to-csv-file requires an explicit value or use --to-csv-file "" for the default.
  ```
  To (keep the first line, change only the Note):
  ```
  or to the template name without .template with a .csv extension (for --parse-json-file).
  Note: when specifying a filename, use = syntax: --to-csv-file=myfile.csv
  ```

  **6b. Fix space-separated examples in the `Examples:` section.**

  There are no space-separated examples in the current `Long` text (lines 687–698). Looking at the examples block, all existing examples either use a bare flag or already use `=` syntax, so no changes are needed there.

  Full corrected `--to-json-file` description block (replace lines ~670–677):
  ```
  * --to-json-file [filename]: write the result as JSON to a file.
    If no filename is given, defaults to output.json in the current working directory (for --parse-json)
    or to the template name without .template (for --parse-json-file).
    Note: when specifying a filename, use = syntax: --to-json-file=myfile.json
  * --to-csv-file [filename]: write the result as CSV to a file.
    If no filename is given, defaults to output.csv in the current working directory (for --parse-json)
    or to the template name without .template with a .csv extension (for --parse-json-file).
    Note: when specifying a filename, use = syntax: --to-csv-file=myfile.csv
  ```

- [x] Step 7: Update `README.md` — default path description

  In the `--to-json-file` flag description line (line ~61):

  Change:
  ```
  If no filename is given, defaults to `output.json` beside the binary (for `--parse-json`) or to the template name without `.template` in the same directory (for `--parse-json-file`). If a filename is given using `--to-json-file=myfile.json`, it is used as-is.
  ```

  To:
  ```
  If no filename is given, defaults to `output.json` in the current working directory (for `--parse-json`) or to the template name without `.template` in the same directory as the template (for `--parse-json-file`). When specifying an explicit filename, the `=` syntax is required: `--to-json-file=myfile.json`.
  ```

  In the `--to-csv-file` flag description line (line ~62):

  Change:
  ```
  Same filename-resolution rules as `--to-json-file` (default `output.csv` for `--parse-json`, template name for `--parse-json-file`). Can be combined with `--to-stdout` and `--to-json-file`. Note: CSV output works best with flat (one-level-deep) JSON objects.
  ```

  To:
  ```
  Same filename-resolution rules as `--to-json-file` (default `output.csv` in the current working directory for `--parse-json`, template name for `--parse-json-file`). When specifying an explicit filename, the `=` syntax is required: `--to-csv-file=myfile.csv`. Can be combined with `--to-stdout` and `--to-json-file`. Note: CSV output works best with flat (one-level-deep) JSON objects.
  ```

- [x] Step 8: Update `README.md` — fix the inline comment on the default-name write example

  Line ~112 currently says:
  ```
  Write to a file (default name `output.json` beside the binary):
  ```

  Change to:
  ```
  Write to a file (default name `output.json` in the current working directory):
  ```

- [x] Step 9: Update E2E test names and comments in `cmd/mock_e2e_test.go`

  In `TestCLIShouldOutputToJsonFile_DefaultName_ParseJson` (line ~380):

  Change:
  ```go
  testName := "Should create output.json beside binary when --to-json-file is passed with no value (parse-json)"
  ```
  To:
  ```go
  testName := "Should create output.json in the current working directory when --to-json-file is passed with no value (parse-json)"
  ```

  Change the comment on line ~383:
  ```go
  // The file is written beside os.Executable(); in tests that's a temp binary path
  // We simply assert no error was returned — the exact path is environment-dependent
  ```
  To:
  ```go
  // The file is written in the process's working directory (os.Getwd()).
  // We simply assert no error was returned.
  ```

  In `TestCLIShouldOutputToCsvFile_DefaultName_ParseJson` (line ~444):

  Change:
  ```go
  testName := "Should create output.csv beside binary when --to-csv-file is passed with no value (parse-json)"
  ```
  To:
  ```go
  testName := "Should create output.csv in the current working directory when --to-csv-file is passed with no value (parse-json)"
  ```

  Change the comment on line ~448:
  ```go
  // File is beside os.Executable(); in tests that's a temp binary path
  // We simply assert no error was returned — the exact path is environment-dependent
  ```
  To:
  ```go
  // The file is written in the process's working directory (os.Getwd()).
  // We simply assert no error was returned.
  ```

- [x] Step 10: Apply `## Proposed Memory Updates` below to `.claude/memories.md`.

## Notes

- The four call sites are: two `default` branches in `resolveOutputPaths` (lines ~1361 and ~1376) and two `default` branches in `routeOutput` (lines ~1918 and ~1954). All four must change; none should be missed.
- The `filepath` import in `cmd/mock.go` is used extensively elsewhere; removing it would cause a compile error. The only import that becomes unused after this change is `os.Executable` — but Go does not import functions individually, so there is no import to remove. The `os` package import remains fully used.
- The `TestCLIShouldOutputToJsonFile_DefaultName_ParseJson` and `TestCLIShouldOutputToCsvFile_DefaultName_ParseJson` tests do not assert on the file's location (they only assert `NoError`). After this fix the file will be written to the test binary's working directory (wherever `go test` is run from, typically the repo root) rather than `/tmp/go-testXXXX/...`. The tests remain valid as-is after the comment/name updates.
- No new tests are required. The existing default-name tests already cover the happy path without asserting the exact directory.
- The `--to-json-file` / `--to-csv-file` space-separated examples `--to-json-file myout.json` do not appear anywhere in the current `Long` text or README (the README already uses `--to-json-file=mydata.json`). The only documentation fix needed is the "beside the binary" wording and the `Note:` lines.

## Proposed Memory Updates

Add to the **Architecture Decisions** section:

- **`workingDir()` uses `os.Getwd()`** — the default output path for `--to-json-file` and `--to-csv-file` (when source is `parse-json`) is resolved via `os.Getwd()`, not `os.Executable()`. `os.Executable()` returns the temp binary path under `go run .`, so files would be lost. `workingDir()` replaced the former `executableDir()` helper.

Update the **Known Gotchas** section entry that mentions `--to-json-file` syntax:

- **`--to-json-file` and `--to-csv-file` require `=` syntax for explicit filenames** — because `NoOptDefVal = "_use_default_"` is set, pflag treats a bare `--to-json-file value` as the flag followed by a separate positional argument. Explicit filenames must use `--to-json-file=myfile.json`. The `Long` help text and README both document this with the `Note:` lines.

Update the **Plan History** table (newest first):

| `202604141746_bug_working_dir_and_tojsonfile_docs` | 2026-04-14 | bug |
