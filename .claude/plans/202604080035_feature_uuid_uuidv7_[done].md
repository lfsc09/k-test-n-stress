# Feature: Add UUID.uuidv7 mock function

> Created: 2026-04-08 00:35

## Summary

Add a new mock function `UUID.uuidv7` that generates a random UUID version 7 value. UUID v7 is time-ordered (uses a Unix timestamp prefix) and is increasingly preferred over v4 for database primary keys. The `jaswdr/faker` library (v2.9.1) only provides `V4()` — it has no v7 support — so a new dependency `github.com/google/uuid` (v1.6.0+, which introduced `NewV7()`) must be added. The implementation follows the same pattern as the existing `UUID.uuidv4` case: a single `case` in `Generate`, one `tableLineData` entry in `List`, a unit-level E2E test in `cmd/mock_e2e_test.go`, and a doc string update in `cmd/mock.go`.

## Affected Files

- `go.mod` — add `github.com/google/uuid` as a direct dependency
- `go.sum` — updated automatically by `go get`
- `mocker/mocker.go` — add `UUID.uuidv7` entry to `List()` and a `case "UUID.uuidv7"` in `Generate()`
- `cmd/mock_e2e_test.go` — add E2E test asserting the output matches a UUID v7 regex pattern
- `cmd/mock.go` — no structural change needed; the `Long` description is self-documenting via `--list`, but the doc comment on `Generate()` in `mocker.go` should be updated to mention v7

## Steps

- [x] Step 1: Add the `github.com/google/uuid` dependency

  Run the following from the module root to fetch and register the package:

  ```
  go get github.com/google/uuid@latest
  ```

  This updates `go.mod` (adds a new `require` line for `github.com/google/uuid`) and `go.sum`. Confirm with `go mod tidy` afterwards to remove any unused entries. The minimum version needed is v1.6.0 (when `NewV7()` was introduced); latest stable is recommended.

- [x] Step 2: Add the `UUID.uuidv7` entry to `List()` in `mocker/mocker.go`

  In the `List()` method, locate the existing UUID block (lines 145–146 of the current file):

  ```go
  fmt.Fprintf(out, "%s\n", tableLineData(colSizes, []string{"UUID.uuidv4", "Generates a random UUID v4"}))
  fmt.Fprintf(out, "%s\n", tableLineDivider(colSizes))
  ```

  Insert the new entry immediately after the `uuidv4` line and before the divider:

  ```go
  fmt.Fprintf(out, "%s\n", tableLineData(colSizes, []string{"UUID.uuidv4", "Generates a random UUID v4"}))
  fmt.Fprintf(out, "%s\n", tableLineData(colSizes, []string{"UUID.uuidv7", "Generates a random UUID v7"}))
  fmt.Fprintf(out, "%s\n", tableLineDivider(colSizes))
  ```

- [x] Step 3: Add `import` for `github.com/google/uuid` in `mocker/mocker.go`

  In the existing `import` block at the top of `mocker/mocker.go`, add the new import alongside the existing ones:

  ```go
  import (
      "fmt"
      "io"
      "math/rand"
      "strconv"
      "strings"
      "time"

      "github.com/google/uuid"
      "github.com/jaswdr/faker/v2"
      regen "github.com/zach-klippenstein/goregen"
  )
  ```

- [x] Step 4: Add the `case "UUID.uuidv7"` branch in `Generate()` in `mocker/mocker.go`

  In the `Generate()` switch, locate the existing UUID section (after the `Date` section):

  ```go
  /*
      UUID
  */
  case "UUID.uuidv4":
      return m.jaswdrFaker.UUID().V4(), nil
  ```

  Add the new case immediately after `uuidv4`:

  ```go
  /*
      UUID
  */
  case "UUID.uuidv4":
      return m.jaswdrFaker.UUID().V4(), nil
  case "UUID.uuidv7":
      v7, err := uuid.NewV7()
      if err != nil {
          return "", fmt.Errorf("failed to generate UUID v7: %w", err)
      }
      return v7.String(), nil
  ```

  `uuid.NewV7()` returns `(uuid.UUID, error)`. The `uuid.UUID` type has a `String()` method that returns the canonical hyphenated lowercase form: `xxxxxxxx-xxxx-7xxx-yxxx-xxxxxxxxxxxx`.

- [x] Step 5: Update the doc comment on `Generate()` in `mocker/mocker.go`

  The existing comment reads:
  > "...time data, UUIDs, and user agents."

  Update it to:
  > "...time data, UUID v4 and UUID v7 values, and user agents."

  This keeps the comment consistent with the available functionality.

- [x] Step 6: Add an E2E test for `UUID.uuidv7` in `cmd/mock_e2e_test.go`

  Following the pattern established by `MockDateSuite` and the existing `--parse-str` tests, add a new test suite (or extend an existing one) for UUID functions. Inspect the file — there is currently no dedicated UUID suite, so create one modelled on `MockDateSuite`:

  Add the following at the end of `cmd/mock_e2e_test.go` (after the existing test functions):

  ```go
  type MockUUIDSuite struct {
      suite.Suite
  }

  func TestMockUUIDSuite(t *testing.T) {
      suite.Run(t, new(MockUUIDSuite))
  }

  func (suite *MockUUIDSuite) executeCommand(args ...string) (string, error) {
      outBuf := new(bytes.Buffer)
      opts := &cmd.CommandOptions{Out: outBuf}
      rootCmd := cmd.NewRootCmd(opts)
      rootCmd.SetArgs(args)
      err := rootCmd.Execute()
      return outBuf.String(), err
  }

  func (suite *MockUUIDSuite) TestUUIDv7() {
      tests := []struct {
          testName    string
          template    string
          assertRegex string
      }{
          {
              testName:    "uuidv7 matches canonical UUID v7 format",
              template:    "{{ UUID.uuidv7 }}",
              assertRegex: `^[0-9a-f]{8}-[0-9a-f]{4}-7[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}\n$`,
          },
      }

      for _, tt := range tests {
          stdOut, err := suite.executeCommand("mock", "--parse-str", tt.template)
          assert.NoError(suite.T(), err, tt.testName)
          assert.Regexp(suite.T(), regexp.MustCompile(tt.assertRegex), stdOut, tt.testName)
      }
  }
  ```

  The regex `^[0-9a-f]{8}-[0-9a-f]{4}-7[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}\n$` enforces:
  - The standard 8-4-4-4-12 hyphenated groups in lowercase hex.
  - The version nibble is `7` (4th group, first character).
  - The variant nibble is `8`, `9`, `a`, or `b` (5th group, first character) — RFC 9562 variant bits.

- [x] Step 7: Update the `TestCLIShouldReturnListOfMockFunctions` E2E test in `cmd/mock_e2e_test.go`

  The existing test (around line 144) already asserts `assert.Contains(suite.T(), stdOut, "UUID.", testName)`, which will pass since both `UUID.uuidv4` and `UUID.uuidv7` share the prefix. No change is strictly required, but optionally add a more specific assertion:

  ```go
  assert.Contains(suite.T(), stdOut, "UUID.uuidv7", testName)
  ```

  Add this line directly after the existing `assert.Contains(suite.T(), stdOut, "UUID.", testName)` assertion for clarity and future safety.

- [x] Step 8: Verify the build and tests pass

  Run from the module root:

  ```
  go build ./...
  go test ./...
  ```

  Confirm all existing tests still pass and the two new test cases (`TestMockUUIDSuite/TestUUIDv7` and the updated list assertion) are green.

## Notes

- `uuid.NewV7()` is available from `github.com/google/uuid` v1.6.0 onward. Earlier versions only exposed `NewRandom()` (v4). Running `go get github.com/google/uuid@latest` will pull the current stable release which is well past v1.6.0.
- The `uuid.UUID` type returned by `NewV7()` is a `[16]byte` array; calling `.String()` on it produces the standard lowercase hyphenated representation, matching the format of `uuidv4` (which `jaswdr/faker` also returns lowercase hex).
- `uuid.NewV7()` returns an error only in rare cases where the OS random source fails. The error path is modelled on the `Payment.creditCardCvv` pattern (wrap in `fmt.Errorf` and propagate).
- UUID v7 is specified in RFC 9562 (April 2024). The version nibble is always `7`; the variant bits are `10xx` (values `8`, `9`, `a`, `b` in hex), which the regex in Step 6 enforces.
- No changes to `cmd/mock.go` are needed beyond what the `--list` flag already covers. The `Long` help text does not enumerate individual functions; it only documents the calling convention.
- No changes are needed to `mocker/helpers.go` or `mocker/helpers_test.go` — `UUID.uuidv7` has no parameters and requires no helper utility.
