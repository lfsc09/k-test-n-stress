# Feature: Add `--to-csv-file [optional_filename]` flag

> Created: 2026-04-10 11:12

## Summary

Add a `--to-csv-file [optional_filename]` flag to the `mock` command that writes the generated data as a CSV file, following the same optional-filename pattern used by `--to-json-file`. When no filename is given the flag uses a sentinel `_use_default_` (via `NoOptDefVal`), which resolves to `output.csv` beside the binary for `--parse-json`, or to the template name with `.template.json` replaced by `.csv` for `--parse-json-file`. When an explicit filename is given it is used as-is. The existing `marshalAsCSV` helper is reused for serialisation; the file is always written as standard (non-prettified) RFC 4180 CSV. The flag is incompatible with `--parse-str`, and at least one of `--to-stdout`, `--to-json-file`, or `--to-csv-file` must be present for `--parse-json` / `--parse-json-file`.

## Affected Files

- `cmd/mock.go` — flag definition, `RunE` variable extraction and validation, `routeOutput` signature and CSV-file writing block, `Long` usage string and examples
- `cmd/mock_e2e_test.go` — new E2E test cases for `--to-csv-file`

## Steps

- [x] **Step 1: Register the `--to-csv-file` flag in `NewMockCmd`**

  After the existing `--to-json-file` flag registration block (lines 268–271), add:

  ```go
  mockCmd.Flags().String("to-csv-file", "", "output result as CSV to a file; optional filename argument")
  mockCmd.Flags().Lookup("to-csv-file").NoOptDefVal = "_use_default_"
  ```

  This mirrors the exact pattern used for `--to-json-file`: the flag is a `String` flag, and `NoOptDefVal` is set so that passing `--to-csv-file` without a value is valid and results in the sentinel string `"_use_default_"`.

- [x] **Step 2: Read the new flag and its `Changed` status inside `RunE`**

  Directly after the existing lines that read `toJsonFile` and `toJsonFileSet` (lines 111–112), add:

  ```go
  toCsvFile, _ := cmd.Flags().GetString("to-csv-file")
  toCsvFileSet := cmd.Flags().Changed("to-csv-file")
  ```

- [x] **Step 3: Extend `--parse-str` incompatibility validation**

  The existing guard (line 166) currently reads:

  ```go
  if runningParseStr && (toStdout != "" || toJsonFileSet) {
  ```

  Change it to also cover the new flag:

  ```go
  if runningParseStr && (toStdout != "" || toJsonFileSet || toCsvFileSet) {
  ```

  The error message already says `--to-stdout and --to-json-file are not available with --parse-str`; update it to:

  ```
  "--parse-str always outputs to stdout; --to-stdout, --to-json-file and --to-csv-file are not available with --parse-str"
  ```

- [x] **Step 4: Extend the "at least one output flag" validation**

  The existing guard (line 171) currently reads:

  ```go
  if !runningParseStr && toStdout == "" && !toJsonFileSet {
  ```

  Change it to:

  ```go
  if !runningParseStr && toStdout == "" && !toJsonFileSet && !toCsvFileSet {
  ```

  The error message stays the same (`--parse-json and --parse-json-file require at least one output flag: --to-stdout or --to-json-file`), but update it to mention `--to-csv-file` as well:

  ```
  "--parse-json and --parse-json-file require at least one output flag: --to-stdout, --to-json-file or --to-csv-file"
  ```

- [x] **Step 5: Pass the new flag values through both `routeOutput` call sites**

  There are two calls to `routeOutput` in `RunE`:

  1. Inside the `runningParseJson` block (line 208)
  2. Inside the `runningParseJsonFile` block (line 252)

  Both must be updated to pass the two new arguments `toCsvFileSet` and `toCsvFile`.

  New signature (see Step 6):

  ```go
  routeOutput(
      parseMaps, generate,
      toStdout, toStdoutPrettify,
      toJsonFileSet, toJsonFile,
      toCsvFileSet, toCsvFile,
      source, defaultFilePath,
      opts.Out,
  )
  ```

  For the `parse-json` call, `defaultFilePath` stays `""` (same as today).
  For the `parse-json-file` call, a `defaultCsvFilePath` must also be computed right after `defaultFilePath` is set:

  ```go
  defaultCsvFileName := strings.Replace(filepath.Base(parseJsonFile), ".template.json", ".csv", 1)
  defaultCsvFilePath := filepath.Join(filepath.Dir(parseJsonFile), defaultCsvFileName)
  ```

  Then pass `defaultCsvFilePath` as the new `defaultCsvFilePath` parameter (see Step 6).

  For the `parse-json` call, pass `""` for `defaultCsvFilePath`.

- [x] **Step 6: Update the `routeOutput` function signature**

  Current signature ends at line 702:

  ```go
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
  ) error {
  ```

  New signature inserts two parameters after `toJsonFileValue` and adds `defaultCsvFilePath` after `defaultFilePath`:

  ```go
  func routeOutput(
      parseMaps []map[string]any,
      generate int,
      toStdout string,
      toStdoutPrettify bool,
      toJsonFileSet bool,
      toJsonFileValue string,
      toCsvFileSet bool,
      toCsvFileValue string,
      source string,
      defaultFilePath string,
      defaultCsvFilePath string,
      out io.Writer,
  ) error {
  ```

  Update the doc comment above the function to describe the two new parameters.

- [x] **Step 7: Implement the CSV file-writing block inside `routeOutput`**

  After the closing `}` of the existing `if toJsonFileSet { ... }` block, add a new block:

  ```go
  // CSV file output
  if toCsvFileSet {
      var resolvedCsvPath string
      switch {
      case toCsvFileValue != "" && toCsvFileValue != "_use_default_":
          resolvedCsvPath = toCsvFileValue
      case source == "parse-json-file":
          resolvedCsvPath = defaultCsvFilePath
      default:
          // source == "parse-json": use output.csv beside the binary
          dir, err := executableDir()
          if err != nil {
              return err
          }
          resolvedCsvPath = filepath.Join(dir, "output.csv")
      }

      // Delete any existing file at that path
      _ = os.Remove(resolvedCsvPath)

      // CSV files are written in standard (non-prettified) format
      csvStr, err := marshalAsCSV(parseMaps, false)
      if err != nil {
          return err
      }

      if err = os.WriteFile(resolvedCsvPath, []byte(csvStr+"\n"), 0644); err != nil {
          return fmt.Errorf("failed to write CSV result to '%s': %w", resolvedCsvPath, err)
      }
  }
  ```

  Note: `marshalAsCSV` already trims the trailing newline; re-adding `"\n"` here ensures the file ends with a single newline, consistent with POSIX text-file convention.

- [x] **Step 8: Update the `Long` usage string and examples in `NewMockCmd`**

  In the `Long` field of `mockCmd`:

  1. In the "Output routing" section, add a bullet for `--to-csv-file` after the `--to-json-file` bullet:
     ```
     * --to-csv-file [filename]: write the result as CSV to a file.
       If no filename is given, defaults to output.csv beside the binary (for --parse-json)
       or to the template name without .template with a .csv extension (for --parse-json-file).
       Note: --to-csv-file requires an explicit value or use --to-csv-file "" for the default.
     ```
  2. Update the "At least one of" sentence to include `--to-csv-file`:
     ```
     * At least one of --to-stdout, --to-json-file or --to-csv-file must be provided.
     ```
  3. Add new examples at the bottom:
     ```
       ktns mock --parse-json '{ "name": "{{ Person.name }}" }' --to-csv-file
       ktns mock --parse-json '{ "name": "{{ Person.name }}" }' --to-csv-file mydata.csv
       ktns mock --parse-json-file "path/to/employees.template.json" --to-csv-file
       ktns mock --parse-json-file "path/to/employees.template.json" --to-csv-file myout.csv
       ktns mock --parse-json-file "path/to/employees.template.json" --generate 5 --to-csv-file
     ```

- [x] **Step 9: Add E2E tests in `cmd/mock_e2e_test.go`**

  Add a new section `// --- Step N: --to-csv-file ---` (after the existing `--to-json-file` section) with the following test methods on `MockCmdE2ETestSuite`:

  **9a. Validation: `--parse-str` with `--to-csv-file` raises error**
  ```go
  func (suite *MockCmdE2ETestSuite) TestCLIShouldRaiseError_ParseStrWithToCsvFile() {
      testName := "Should raise error when --parse-str is combined with --to-csv-file"
      _, err := suite.executeCommand("mock", "--parse-str", "Hello", "--to-csv-file")
      assert.Error(suite.T(), err, testName)
      assert.Contains(suite.T(), err.Error(), "--parse-str always outputs to stdout", testName)
  }
  ```

  **9b. Validation: `--parse-json` with no output flag still raises error (regression)**

  This test already exists (`TestCLIShouldRaiseError_ParseJsonWithNoOutputFlag`). Update its expected error message string if Step 4 changes the message text. No new test needed.

  **9c. Default filename: `--parse-json` + `--to-csv-file` (no value)**
  ```go
  func (suite *MockCmdE2ETestSuite) TestCLIShouldOutputToCsvFile_DefaultName_ParseJson() {
      testName := "Should create output.csv beside binary when --to-csv-file is passed with no value (parse-json)"
      _, err := suite.executeCommand("mock", "--parse-json", `{"name": "{{ Person.name }}"}`, "--to-csv-file")
      assert.NoError(suite.T(), err, testName)
      // File is beside os.Executable(); assert no error returned — exact path is environment-dependent
  }
  ```

  **9d. Explicit filename: `--parse-json` + `--to-csv-file=<path>`**
  ```go
  func (suite *MockCmdE2ETestSuite) TestCLIShouldOutputToCsvFile_ExplicitName_ParseJson() {
      testName := "Should write CSV to explicit path with --to-csv-file <path>"
      tmpDir := suite.T().TempDir()
      outPath := filepath.Join(tmpDir, "result.csv")
      _, err := suite.executeCommand("mock", "--parse-json", `{"name": "{{ Person.name }}", "age": "{{ Number.number::{18}:{80} }}"}`, "--to-csv-file="+outPath)
      assert.NoError(suite.T(), err, testName)
      data, readErr := os.ReadFile(outPath)
      assert.NoError(suite.T(), readErr, testName)
      lines := strings.Split(strings.TrimSpace(string(data)), "\n")
      // Header line should be sorted keys: age,name
      assert.Equal(suite.T(), "age,name", lines[0], testName)
      // 1 header + 1 data row
      assert.Len(suite.T(), lines, 2, testName)
  }
  ```

  **9e. Default filename: `--parse-json-file` + `--to-csv-file` (no value)**
  ```go
  func (suite *MockCmdE2ETestSuite) TestCLIShouldOutputToCsvFile_DefaultName_ParseJsonFile() {
      testName := "Should write default employee.csv alongside template when --to-csv-file is passed with no value"
      tmpDir := suite.T().TempDir()
      templatePath := filepath.Join(tmpDir, "employee.template.json")
      _ = os.WriteFile(templatePath, []byte(`{"name": "{{ Person.name }}"}`), 0644)
      _, err := suite.executeCommand("mock", "--parse-json-file", templatePath, "--to-csv-file")
      assert.NoError(suite.T(), err, testName)
      outPath := filepath.Join(tmpDir, "employee.csv")
      data, readErr := os.ReadFile(outPath)
      assert.NoError(suite.T(), readErr, testName)
      lines := strings.Split(strings.TrimSpace(string(data)), "\n")
      assert.Equal(suite.T(), "name", lines[0], testName)
      assert.Len(suite.T(), lines, 2, testName)
  }
  ```

  **9f. Explicit filename: `--parse-json-file` + `--to-csv-file=<path>`**
  ```go
  func (suite *MockCmdE2ETestSuite) TestCLIShouldOutputToCsvFile_ExplicitName_ParseJsonFile() {
      testName := "Should write to explicit CSV path and NOT create default file when explicit --to-csv-file is given"
      tmpDir := suite.T().TempDir()
      templatePath := filepath.Join(tmpDir, "employee.template.json")
      _ = os.WriteFile(templatePath, []byte(`{"name": "{{ Person.name }}"}`), 0644)
      outPath := filepath.Join(tmpDir, "out.csv")
      _, err := suite.executeCommand("mock", "--parse-json-file", templatePath, "--to-csv-file="+outPath)
      assert.NoError(suite.T(), err, testName)
      // Explicit output file must exist
      data, readErr := os.ReadFile(outPath)
      assert.NoError(suite.T(), readErr, testName)
      lines := strings.Split(strings.TrimSpace(string(data)), "\n")
      assert.Equal(suite.T(), "name", lines[0], testName)
      // Default file must NOT exist
      defaultOut := filepath.Join(tmpDir, "employee.csv")
      _, statErr := os.Stat(defaultOut)
      assert.True(suite.T(), os.IsNotExist(statErr), testName)
  }
  ```

  **9g. With `--generate`: multiple data rows in CSV file**
  ```go
  func (suite *MockCmdE2ETestSuite) TestCLIShouldOutputToCsvFile_WithGenerate() {
      testName := "Should write CSV file with N data rows when --generate is used"
      tmpDir := suite.T().TempDir()
      outPath := filepath.Join(tmpDir, "result.csv")
      _, err := suite.executeCommand("mock", "--parse-json", `{"name": "{{ Person.name }}"}`, "--generate", "3", "--to-csv-file="+outPath)
      assert.NoError(suite.T(), err, testName)
      data, readErr := os.ReadFile(outPath)
      assert.NoError(suite.T(), readErr, testName)
      lines := strings.Split(strings.TrimSpace(string(data)), "\n")
      // 1 header + 3 data rows
      assert.Len(suite.T(), lines, 4, testName)
  }
  ```

  **9h. Stale file is deleted before writing**
  ```go
  func (suite *MockCmdE2ETestSuite) TestCLIShouldOutputToCsvFile_DeletesPreviousOutput() {
      testName := "Should delete stale CSV output file before writing new one"
      tmpDir := suite.T().TempDir()
      templatePath := filepath.Join(tmpDir, "employee.template.json")
      _ = os.WriteFile(templatePath, []byte(`{"name": "{{ Person.name }}"}`), 0644)
      outPath := filepath.Join(tmpDir, "employee.csv")
      _ = os.WriteFile(outPath, []byte("stale,data\n1,2\n"), 0644)
      _, err := suite.executeCommand("mock", "--parse-json-file", templatePath, "--to-csv-file")
      assert.NoError(suite.T(), err, testName)
      data, readErr := os.ReadFile(outPath)
      assert.NoError(suite.T(), readErr, testName)
      assert.NotContains(suite.T(), string(data), "stale", testName)
  }
  ```

  **9i. Both `--to-json-file` and `--to-csv-file` can coexist**
  ```go
  func (suite *MockCmdE2ETestSuite) TestCLIShouldOutputBothJsonAndCsvFile() {
      testName := "Should write both JSON and CSV files when both --to-json-file and --to-csv-file are given"
      tmpDir := suite.T().TempDir()
      jsonPath := filepath.Join(tmpDir, "result.json")
      csvPath := filepath.Join(tmpDir, "result.csv")
      _, err := suite.executeCommand("mock", "--parse-json", `{"name": "{{ Person.name }}"}`, "--to-json-file="+jsonPath, "--to-csv-file="+csvPath)
      assert.NoError(suite.T(), err, testName)
      _, jsonErr := os.Stat(jsonPath)
      assert.NoError(suite.T(), jsonErr, testName)
      _, csvErr := os.Stat(csvPath)
      assert.NoError(suite.T(), csvErr, testName)
  }
  ```

## Notes

- **`NoOptDefVal` mechanics**: setting `NoOptDefVal = "_use_default_"` on a `String` flag means Cobra will use that sentinel when the flag is provided without a value (e.g. `--to-csv-file`). When a value is given it must use `=` syntax (`--to-csv-file=myfile.csv`) because the flag is otherwise consumed as a boolean-like token. This is identical to how `--to-json-file` already works, and the E2E tests must consistently use `--to-csv-file=<path>` for explicit paths.
- **`marshalAsCSV` reuse**: the existing helper is used with `prettify=false` for file output. There is no `--to-csv-file-prettify` flag; prettified (human-readable) CSV output remains a stdout-only concern via `--to-stdout as-csv --to-stdout-prettify`.
- **Trailing newline in file**: `marshalAsCSV` trims trailing newlines; the writing block must append `"\n"` when calling `os.WriteFile` so the resulting file is a well-formed POSIX text file.
- **Error message updates**: Steps 3 and 4 both touch validation error messages. The existing E2E tests that assert on those exact message strings (`TestCLIShouldRaiseError_ParseStrWithToJsonFile` and `TestCLIShouldRaiseError_ParseJsonWithNoOutputFlag` / `TestCLIShouldRaiseError_ParseJsonFileWithNoOutputFlag`) will need their expected strings updated if the messages change. Review them before finalising.
- **No new helpers needed**: all resolution logic is self-contained in `routeOutput`; no additional functions in `helpers.go` or `utils.go` are required.
- **`defaultCsvFilePath` for `parse-json` source**: when `source == "parse-json"`, no template file exists, so `defaultCsvFilePath` is passed as `""`. The `switch` inside `routeOutput` falls through to the `default` branch (binary-dir `output.csv`) in that case, so the empty string is never dereferenced.
