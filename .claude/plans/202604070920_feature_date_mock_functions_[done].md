# Date Mock Functions: Date.date / Date.time / Date.datetime / Date.now

> Created: 2026-04-07 09:20

## Summary

The codebase has a broken stub registered as `Time.date` in `mocker/mocker.go` that returns `"", nil` and is never implemented. The `cmd/mock.go` long description already references `Date.date` as the intended name, so the stub is both misnamed and empty. This feature removes the `Time.date` stub entirely, adds four fully implemented `Date.*` mock functions (`Date.date`, `Date.time`, `Date.datetime`, `Date.now`), introduces four private helper functions in `mocker/helpers.go` to support them, updates the `List()` output to show a DATE section instead of the TIME section, and adds unit tests for the helpers and E2E tests for all four functions via the CLI.

## Affected Files

- `mocker/helpers.go` — add `"time"` import; add `formatDatetime`, `parseDateOnly`, `parseDatetimeFull`, `parseTimeOnly`
- `mocker/mocker.go` — add `"time"` import; remove `/* TIME */` block and `case "Time.date"`; add `/* DATE */` block with four cases; update `List()` to replace the `Time.date` row with four `Date.*` rows
- `mocker/helpers_test.go` — extend the existing `MockerUtilsInternalTestSuite` with four new test methods covering the new helpers
- `cmd/mock_e2e_test.go` — add a new `MockDateSuite` test suite with table-driven E2E tests for all four `Date.*` functions; update the existing `TestCLIShouldReturnListOfMockFunctions` assertion for `"Time."` → `"Date."`

## Steps

- [x] Step 1: Add `"time"` import to `mocker/helpers.go`

  The current import block is:
  ```go
  import (
      "fmt"
      "strings"
  )
  ```
  Add `"time"` so it becomes:
  ```go
  import (
      "fmt"
      "strings"
      "time"
  )
  ```

- [x] Step 2: Add `formatDatetime(t time.Time, format string) string` to `mocker/helpers.go`

  Place this function after `extractRegex`. It performs ordered `strings.ReplaceAll` substitutions on `format`, replacing human-readable tokens with values taken from `t`. Substitution order must be: `sss` first (to prevent `ss` consuming the substring), then `YYYY`, `MM`, `DD`, `hh`, `mm`, `ss`.

  Token-to-value mapping:
  - `YYYY` → `fmt.Sprintf("%04d", t.Year())`
  - `MM`   → `fmt.Sprintf("%02d", int(t.Month()))`
  - `DD`   → `fmt.Sprintf("%02d", t.Day())`
  - `hh`   → `fmt.Sprintf("%02d", t.Hour())`
  - `mm`   → `fmt.Sprintf("%02d", t.Minute())`
  - `ss`   → `fmt.Sprintf("%02d", t.Second())`
  - `sss`  → `fmt.Sprintf("%03d", t.Nanosecond()/1_000_000)`

  The function signature is:
  ```go
  func formatDatetime(t time.Time, format string) string
  ```

- [x] Step 3: Add `parseDateOnly(s string) (time.Time, error)` to `mocker/helpers.go`

  Single-line implementation:
  ```go
  func parseDateOnly(s string) (time.Time, error) {
      return time.Parse("2006-01-02", s)
  }
  ```

- [x] Step 4: Add `parseTimeOnly(s string) (time.Time, error)` to `mocker/helpers.go`

  Single-line implementation:
  ```go
  func parseTimeOnly(s string) (time.Time, error) {
      return time.Parse("15:04", s)
  }
  ```

- [x] Step 5: Add `parseDatetimeFull(s string) (time.Time, error)` to `mocker/helpers.go`

  Try `time.Parse("2006-01-02T15:04", s)` first. If that returns an error, try `time.Parse("2006-01-02", s)`. Return the result of whichever succeeds. Only return an error if both fail — in that case, return the error from the second attempt (date-only parse).

  ```go
  func parseDatetimeFull(s string) (time.Time, error) {
      t, err := time.Parse("2006-01-02T15:04", s)
      if err == nil {
          return t, nil
      }
      return time.Parse("2006-01-02", s)
  }
  ```

- [x] Step 6: Add `"time"` to the import block in `mocker/mocker.go`

  The current import block is:
  ```go
  import (
      "fmt"
      "io"
      "math/rand"
      "strconv"
      "strings"

      "github.com/jaswdr/faker/v2"
      regen "github.com/zach-klippenstein/goregen"
  )
  ```
  Add `"time"` to the standard library section.

- [x] Step 7: Remove the `/* TIME */` block from `Generate()` in `mocker/mocker.go`

  Find and delete the following block (lines 379-382 in the current file):
  ```go
  /*
      TIME
  */
  case "Time.date":
      return "", nil
  ```

- [x] Step 8: Add the `/* DATE */` comment block with `case "Date.date"` in `Generate()` in `mocker/mocker.go`

  Insert this block in the same position where `/* TIME */` was (alphabetically between `REGEX` and `UUID`):

  ```go
  /*
      DATE
  */
  case "Date.date":
      fromDefault := time.Now().AddDate(-5, 0, 0)
      toDefault := time.Now().AddDate(5, 0, 0)
      format := "YYYY-MM-DD"
      fromTime := fromDefault
      toTime := toDefault
      if len(functionParams) > 0 && functionParams[0] != "" {
          if t, err := parseDateOnly(functionParams[0]); err == nil {
              fromTime = t
          }
      }
      if len(functionParams) > 1 && functionParams[1] != "" {
          if t, err := parseDateOnly(functionParams[1]); err == nil {
              toTime = t
          }
      }
      if len(functionParams) > 2 && functionParams[2] != "" {
          extracted, err := extractRegex(functionParams[2])
          if err != nil {
              return "", err
          }
          format = extracted
      }
      delta := toTime.Unix() - fromTime.Unix()
      if delta < 0 {
          return "", fmt.Errorf("Date.date: 'from' must be before 'to'")
      }
      // Pick a random whole day within the range
      daySeconds := int64(24 * 60 * 60)
      days := delta / daySeconds
      randomDay := rand.Int63n(days + 1)
      result := fromTime.Add(time.Duration(randomDay * daySeconds) * time.Second)
      return formatDatetime(result, format), nil
  ```

  Note on `from > to`: the spec says "assert error" for this case in tests. The implementation should return an error when `delta < 0`.

- [x] Step 9: Add `case "Date.time"` to the `/* DATE */` block in `Generate()` in `mocker/mocker.go`

  ```go
  case "Date.time":
      fromDefault, _ := parseTimeOnly("00:00")
      toDefault, _ := parseTimeOnly("23:59")
      format := "hh:mm:ss.sss"
      fromTime := fromDefault
      toTime := toDefault
      if len(functionParams) > 0 && functionParams[0] != "" {
          if t, err := parseTimeOnly(functionParams[0]); err == nil {
              fromTime = t
          }
      }
      if len(functionParams) > 1 && functionParams[1] != "" {
          if t, err := parseTimeOnly(functionParams[1]); err == nil {
              toTime = t
          }
      }
      if len(functionParams) > 2 && functionParams[2] != "" {
          extracted, err := extractRegex(functionParams[2])
          if err != nil {
              return "", err
          }
          format = extracted
      }
      fromSecs := fromTime.Hour()*3600 + fromTime.Minute()*60
      toSecs := toTime.Hour()*3600 + toTime.Minute()*60
      totalRange := toSecs - fromSecs + 59 // +59 to include seconds within the final minute
      if totalRange < 0 {
          return "", fmt.Errorf("Date.time: 'from' must be before 'to'")
      }
      randomSecs := rand.Intn(totalRange + 1)
      randomMs := rand.Intn(1000)
      h := (fromSecs + randomSecs) / 3600
      remaining := (fromSecs + randomSecs) % 3600
      min := remaining / 60
      sec := remaining % 60
      result := time.Date(0, 1, 1, h, min, sec, randomMs*1_000_000, time.UTC)
      return formatDatetime(result, format), nil
  ```

- [x] Step 10: Add `case "Date.datetime"` to the `/* DATE */` block in `Generate()` in `mocker/mocker.go`

  ```go
  case "Date.datetime":
      fromDefault := time.Now().AddDate(-5, 0, 0)
      toDefault := time.Now().AddDate(5, 0, 0)
      format := "YYYY-MM-DDThh:mm:ss.sss"
      fromTime := fromDefault
      toTime := toDefault
      if len(functionParams) > 0 && functionParams[0] != "" {
          if t, err := parseDatetimeFull(functionParams[0]); err == nil {
              fromTime = t
          }
      }
      if len(functionParams) > 1 && functionParams[1] != "" {
          if t, err := parseDatetimeFull(functionParams[1]); err == nil {
              toTime = t
          }
      }
      if len(functionParams) > 2 && functionParams[2] != "" {
          extracted, err := extractRegex(functionParams[2])
          if err != nil {
              return "", err
          }
          format = extracted
      }
      deltaMs := toTime.UnixMilli() - fromTime.UnixMilli()
      if deltaMs < 0 {
          return "", fmt.Errorf("Date.datetime: 'from' must be before 'to'")
      }
      randomDelta := rand.Int63n(deltaMs + 1)
      result := time.UnixMilli(fromTime.UnixMilli() + randomDelta)
      return formatDatetime(result, format), nil
  ```

- [x] Step 11: Add `case "Date.now"` to the `/* DATE */` block in `Generate()` in `mocker/mocker.go`

  ```go
  case "Date.now":
      format := "YYYY-MM-DDThh:mm:ss.sss"
      if len(functionParams) > 0 && functionParams[0] != "" {
          extracted, err := extractRegex(functionParams[0])
          if err != nil {
              return "", err
          }
          format = extracted
      }
      return formatDatetime(time.Now(), format), nil
  ```

- [x] Step 12: Update `List()` in `mocker/mocker.go` — replace the TIME section with a DATE section

  Find these two lines (currently lines 139-140):
  ```go
  fmt.Fprintf(out, "%s\n", tableLineData(colSizes, []string{"Time.date", "Generates a random date"}))
  fmt.Fprintf(out, "%s\n", tableLineDivider(colSizes))
  ```

  Replace them with:
  ```go
  fmt.Fprintf(out, "%s\n", tableLineData(colSizes, []string{"Date.date:[from]:[to]:[format]", "Random date. from/to: YYYY-MM-DD. Default format: YYYY-MM-DD"}))
  fmt.Fprintf(out, "%s\n", tableLineData(colSizes, []string{"Date.time:[from]:[to]:[format]", "Random time. from/to: hh:mm. Default format: hh:mm:ss.sss"}))
  fmt.Fprintf(out, "%s\n", tableLineData(colSizes, []string{"Date.datetime:[from]:[to]:[format]", "Random datetime. from/to: YYYY-MM-DDThh:mm. Default: YYYY-MM-DDThh:mm:ss.sss"}))
  fmt.Fprintf(out, "%s\n", tableLineData(colSizes, []string{"Date.now:[format]", "Current datetime. Default format: YYYY-MM-DDThh:mm:ss.sss"}))
  fmt.Fprintf(out, "%s\n", tableLineDivider(colSizes))
  ```

  Note: the section header comment (e.g. `/* DATE */`) in the `List()` function body is optional but consistent with the surrounding code style — there are no section comments in `List()` currently, only in `Generate()`, so no comment is needed here.

- [x] Step 13: Add helper tests to `mocker/helpers_test.go` — `TestFormatDatetime`

  Add a new method `TestFormatDatetime` to the existing `MockerUtilsInternalTestSuite`. Use a fixed deterministic time value: `time.Date(2026, 4, 7, 15, 4, 5, 123000000, time.UTC)`.

  Table-driven sub-tests:
  | testName | format input | expectedOutput |
  |---|---|---|
  | `"YYYY only"` | `"YYYY"` | `"2026"` |
  | `"YYYY-MM-DD"` | `"YYYY-MM-DD"` | `"2026-04-07"` |
  | `"hh:mm:ss.sss"` | `"hh:mm:ss.sss"` | `"15:04:05.123"` |
  | `"DD - mm:ss"` | `"DD - mm:ss"` | `"07 - 04:05"` |
  | `"YYYY-MM-DDThh:mm:ss.sss"` | `"YYYY-MM-DDThh:mm:ss.sss"` | `"2026-04-07T15:04:05.123"` |
  | `"sss before ss no conflict"` | `"ss.sss"` | `"05.123"` (verifies `sss` replaced before `ss`) |

  Each sub-test calls `formatDatetime(fixedTime, tt.format)` and asserts `assert.Equal(suite.T(), tt.expectedOutput, result, ...)`.

- [x] Step 14: Add helper tests to `mocker/helpers_test.go` — `TestParseDateOnly`

  Add method `TestParseDateOnly` to `MockerUtilsInternalTestSuite`. Table-driven:
  | testName | input | expectError |
  |---|---|---|
  | `"valid date"` | `"2026-04-07"` | false |
  | `"invalid string"` | `"not-a-date"` | true |
  | `"empty string"` | `""` | true |
  | `"wrong format (datetime)"` | `"2026-04-07T15:04"` | true |

  Use `assert.NoError` / `assert.Error` accordingly, and for the valid case also assert that `t.Year() == 2026`, `int(t.Month()) == 4`, `t.Day() == 7`.

- [x] Step 15: Add helper tests to `mocker/helpers_test.go` — `TestParseTimeOnly`

  Add method `TestParseTimeOnly` to `MockerUtilsInternalTestSuite`. Table-driven:
  | testName | input | expectError |
  |---|---|---|
  | `"valid time"` | `"15:04"` | false |
  | `"invalid hour"` | `"25:00"` | true |
  | `"invalid string"` | `"abc"` | true |
  | `"empty string"` | `""` | true |

  For the valid case also assert `t.Hour() == 15` and `t.Minute() == 4`.

- [x] Step 16: Add helper tests to `mocker/helpers_test.go` — `TestParseDatetimeFull`

  Add method `TestParseDatetimeFull` to `MockerUtilsInternalTestSuite`. Table-driven:
  | testName | input | expectError |
  |---|---|---|
  | `"valid datetime"` | `"2026-04-07T15:04"` | false |
  | `"valid date-only fallback"` | `"2026-04-07"` | false |
  | `"invalid string"` | `"not-valid"` | true |
  | `"empty string"` | `""` | true |

  For the datetime case assert `t.Hour() == 15` and `t.Minute() == 4`. For the date-only fallback assert `t.Hour() == 0` and `t.Minute() == 0`.

- [x] Step 17: Add a new E2E test suite `MockDateSuite` in `cmd/mock_e2e_test.go`

  Add a new suite struct and runner alongside the existing `MockCmdE2ETestSuite`. The suite reuses the same `executeCommand` helper pattern:

  ```go
  type MockDateSuite struct {
      suite.Suite
  }

  func TestMockDateSuite(t *testing.T) {
      suite.Run(t, new(MockDateSuite))
  }

  func (suite *MockDateSuite) executeCommand(args ...string) (string, error) {
      outBuf := new(bytes.Buffer)
      opts := &cmd.CommandOptions{Out: outBuf}
      rootCmd := cmd.NewRootCmd(opts)
      rootCmd.SetArgs(args)
      err := rootCmd.Execute()
      return outBuf.String(), err
  }
  ```

  Add `"regexp"` to the import block if not already present.

- [x] Step 18: Add `Date.date` E2E tests to `MockDateSuite` in `cmd/mock_e2e_test.go`

  Method `TestDateDate`. Table-driven using `regexp.MustCompile` + `assert.Regexp` / `assert.NoError` / `assert.Error`:

  | testName | template | assertRegex | assertErr |
  |---|---|---|---|
  | `"default format"` | `{{ Date.date }}` | `^\d{4}-\d{2}-\d{2}\n$` | NoError |
  | `"YYYY format"` | `{{ Date.date::/YYYY/ }}` | `^\d{4}\n$` | NoError |
  | `"YYYY-MM format"` | `{{ Date.date::/YYYY-MM/ }}` | `^\d{4}-\d{2}\n$` | NoError |

  For the error cases, add a separate method `TestDateDate_Errors`:
  | testName | template | expected error behavior |
  |---|---|---|
  | `"from after to"` | `{{ Date.date:2030-01-01:2020-01-01: }}` | assert.Error |

  Note: the template syntax uses `::` to pass an empty third param (format), which falls back to default. Use `--parse-str` to pass the template.

- [x] Step 19: Add `Date.time` E2E tests to `MockDateSuite` in `cmd/mock_e2e_test.go`

  Method `TestDateTime`. Table-driven:

  | testName | template | assertRegex |
  |---|---|---|
  | `"default format"` | `{{ Date.time }}` | `^\d{2}:\d{2}:\d{2}\.\d{3}\n$` |
  | `"hh:mm format"` | `{{ Date.time::/hh:mm/ }}` | `^\d{2}:\d{2}\n$` |
  | `"hh:mm range"` | `{{ Date.time:08:00:17:00: }}` | `^\d{2}:\d{2}:\d{2}\.\d{3}\n$` |

  All assert NoError.

- [x] Step 20: Add `Date.datetime` E2E tests to `MockDateSuite` in `cmd/mock_e2e_test.go`

  Method `TestDateDatetime`. Table-driven:

  | testName | template | assertRegex |
  |---|---|---|
  | `"default format"` | `{{ Date.datetime }}` | `^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}\.\d{3}\n$` |
  | `"custom format"` | `{{ Date.datetime::/DD-MM-YYYY hh:mm/ }}` | `^\d{2}-\d{2}-\d{4} \d{2}:\d{2}\n$` |
  | `"date-only from/to"` | `{{ Date.datetime:2020-01-01:2030-12-31: }}` | `^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}\.\d{3}\n$` |
  | `"datetime from/to"` | `{{ Date.datetime:2020-01-01T08:00:2020-01-01T17:00: }}` | `^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}\.\d{3}\n$` |

  All assert NoError.

- [x] Step 21: Add `Date.now` E2E tests to `MockDateSuite` in `cmd/mock_e2e_test.go`

  Method `TestDateNow`. Table-driven:

  | testName | template | assertRegex |
  |---|---|---|
  | `"default format"` | `{{ Date.now }}` | `^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}\.\d{3}\n$` |
  | `"YYYY-MM format"` | `{{ Date.now:/YYYY-MM/ }}` | `^\d{4}-\d{2}\n$` |
  | `"YYYY only"` | `{{ Date.now:/YYYY/ }}` | `^\d{4}\n$` |

  All assert NoError.

- [x] Step 22: Update the existing `TestCLIShouldReturnListOfMockFunctions` in `cmd/mock_e2e_test.go`

  The test currently asserts `assert.Contains(suite.T(), stdOut, "Time.", testName)`. Since the TIME section is removed and replaced with DATE, change that line to:
  ```go
  assert.Contains(suite.T(), stdOut, "Date.", testName)
  ```

## Notes

- **Token collision (`ss` vs `sss`)**: `formatDatetime` must replace `sss` before `ss`. The plan specifies this explicitly in Step 2. Failing to do so would cause `sss` to first partially match as `ss` + leftover `s`, producing wrong output like `05s` instead of `123`.

- **`Date.date` random logic**: The spec says "truncate to whole days". The implementation picks a random offset in whole days (not seconds), ensuring the result is always midnight of a day within `[from, to]`. This means the time component of `from` is discarded.

- **`Date.time` range calculation**: The `+59` in `totalRange` accounts for the seconds [0..59] within the last minute of the `to` boundary, since `to` format is `hh:mm` (minute precision only). Without it, times like `23:59:59` would never be generated when `to=23:59`.

- **`from > to` error handling**: The spec's E2E test says "assert error" for `Date.date` with `from > to`. Steps 8, 9, and 10 all return explicit errors when `delta < 0`. This is the behavior the E2E tests validate.

- **`functionParams` bounds**: All four cases follow the existing project convention: check `len(functionParams) > i && functionParams[i] != ""` before accessing index `i`. No panics from out-of-bounds access.

- **No changes to `cmd/mock.go`**: The `Long` description already contains correct `Date.date` examples. No edits needed.

- **`time.UTC` for `Date.time` result**: The `Date.time` case constructs a `time.Time` using `time.Date(0, 1, 1, h, min, sec, ms, time.UTC)`. Only the time component is used by `formatDatetime`, so the zero-date values for year/month/day are irrelevant.

- **E2E test regex anchoring**: Templates passed via `--parse-str` produce output followed by a newline (`\n`). Regex patterns in E2E tests should account for this (e.g. `^\d{4}-\d{2}-\d{2}\n$`).

- **Import `"regexp"` in `cmd/mock_e2e_test.go`**: The existing E2E test file does not import `"regexp"`. Step 17 requires adding it when setting up the new suite. Inspect the import block before editing to avoid duplicates.
