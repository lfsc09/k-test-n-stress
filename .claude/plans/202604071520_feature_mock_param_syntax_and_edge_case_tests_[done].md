# Feature: Enforce `{}` wrapping for non-regex mock function params + edge case test coverage

> Created: 2026-04-07 15:20

## Summary

Currently, `extractMockMethod` in `cmd/mock.go` accepts bare (unwrapped) positional
parameter values such as `Number.number:2:1:5`.  The documented syntax mandates that
non-regex values must be wrapped in `{…}` and regex values in `/…/`.  This plan
enforces that rule at parse time by making `extractMockMethod` return an error when it
encounters a positional parameter token that is not wrapped in either `{…}` or `/…/`.
It also expands the e2e and unit tests to cover every mock function's parameter
combinations, empty/missing params, wrong param values, and the new rejection of bare
(unwrapped) values.

---

## Affected Files

- `cmd/mock.go` — add validation inside `extractMockMethod`; update callers to handle
  the new error return; update the command long-description examples to only show
  the `{…}`-wrapped syntax.
- `cmd/mock_test.go` — add unit-test cases for `extractMockMethod` covering bare values
  (rejection) and all other edge cases.
- `cmd/mock_e2e_test.go` — add new `MockCmdE2ETestSuite` and per-function suites
  covering each param, multiple params, missing params, wrong values.

---

## Steps

### Part 1 – Enforce `{…}` or `/…/` wrapping in `extractMockMethod`

- [x] **Step 1 – Change the signature of `extractMockMethod` to return an error.**

  Current:
  ```go
  func extractMockMethod(rawValue string) (string, []string)
  ```
  New:
  ```go
  func extractMockMethod(rawValue string) (string, []string, error)
  ```

- [x] **Step 2 – Add bare-value detection inside the `extractMockMethod` loop.**

  A "bare" token is one where the loop finds a non-empty run of characters between two
  `:` delimiters (or between the function-name colon and the end of string) without
  the token having been opened by `{` or `/`.

  Implementation approach: introduce a third boolean flag `inBare bool`.  When the
  parser encounters a character that is not `{`, `/`, `:`, and is currently outside
  both `inRegex` and `inValue`, set `inBare = true` and write the character to `buf`.
  When the `:` delimiter or end-of-input is reached and `inBare` is `true`, do **not**
  flush the buffer as a param — instead return an error:
  ```
  "mock function parameter '%s' must be wrapped in {…} for a value or /…/ for a regex"
  ```
  where `%s` is the contents of `buf` (the bare token seen so far).

  Only the **first** occurrence in params triggers the error (fail fast).  The
  function name (parts[0], the token before the first `:`) is never a param, so the
  bare-value check applies **only to parts[1…]** — i.e. only after the first `:` has
  been encountered.

  Concrete changes inside the `for _, char := range trimmed` loop:
  - Add `paramCount int` and `inBare bool` variables in the same block as `inRegex`
    and `inValue`.
  - On the `case char == ':' && !inRegex && !inValue:` branch: after flushing the
    current buffer to `parts`, increment `paramCount`.  If `paramCount >= 1` (meaning
    we have moved past the function name) and `inBare` was `true`, return the error
    immediately.  Then reset `inBare = false`.
  - On the `default:` branch: if `paramCount >= 1` and `!inRegex && !inValue`, set
    `inBare = true`.
  - After the loop: if `inBare` is `true`, return the error for the trailing token.

  Empty tokens (bare `::` or `{}`) are never bare — they produce an empty string param
  and are still valid.

- [x] **Step 3 – Update every call site of `extractMockMethod` to handle the error.**

  There are three call sites in `cmd/mock.go`:

  1. `processJsonMap` (string-value path, line ~412):
     ```go
     functionName, params, err := extractMockMethod(interpretedValue)
     if err != nil {
         return err
     }
     ```

  2. `processJsonMap` (array-of-strings path, line ~458):
     ```go
     functionName, params, err := extractMockMethod(interpretedValue)
     if err != nil {
         return err
     }
     ```

  3. `processStr` (line ~543):
     ```go
     functionName, params, err := extractMockMethod(inner)
     if err != nil {
         out.WriteString(fmt.Sprintf("[%v]", err))
         // continue loop — do not crash the whole string
     } else {
         mockValue, err := mocker.Generate(functionName, params)
         ...
     }
     ```
     This mirrors how `mocker.Generate` errors are handled in the same function:
     inlined as `[error text]` in the output so the rest of the string keeps rendering.

- [x] **Step 4 – Update the command long-description (`Long` field in `NewMockCmd`).**

  Remove examples that use the old bare-value syntax (e.g. `Number.number::1:100`) and
  replace them with the `{…}`-wrapped equivalents (e.g. `Number.number:::{1}:{100}`).
  The current `Long` string already uses the correct wrapped syntax in its bullet points
  but the `Examples:` block still has `Number.number::1:100`.  Fix those lines only.

---

### Part 2 – Unit tests for `extractMockMethod` (in `cmd/mock_test.go`)

- [x] **Step 5 – Add a new test method `TestExtractMockMethod_BareValueRejection` inside
  `MockCmdTestSuite`.**

  Add the following cases, all of which must return an error:

  | testName | input |
  |---|---|
  | bare single param | `Number.number:2` |
  | bare first of two params | `Number.number:2:1` |
  | bare second param, first is wrapped | `Number.number:{2}:1` |
  | bare param after valid regex | `Date.date::{2030-01-01}:YYYY-MM-DD` — third param is bare |
  | bare param with spaces | `Number.number: 2 ` (trimmed outer spaces, then bare `2`) |

  Each case must assert: `funcName` and `params` are empty / nil and `err` is non-nil
  containing the substring `"must be wrapped in"`.

- [x] **Step 6 – Add edge cases to the existing `TestExtractMockMethod_ValidInputs` table.**

  | testName | input | expectedFuncName | expectedParams |
  |---|---|---|---|
  | no params at all | `Address.city` | `Address.city` | `[]string{}` |
  | single empty wrapped param | `Number.number:{}` | `Number.number` | `[""]` |
  | three params all empty | `Number.number:{}:{}:{}` | `Number.number` | `["","",""]` |
  | three params with bare colons (empty bare) | `Number.number:::` | `Number.number` | `["","",""]` — Note: bare colons `::` produce empty strings and are exempt from the bare check because `inBare` is never set (nothing was written to buf) |
  | single regex param | `Regex.regex:/[a-z]+/` | `Regex.regex` | `["/[a-z]+/"]` |
  | colon inside value param | `Date.time:{18:00}:{20:00}` | `Date.time` | `["18:00","20:00"]` |
  | colon inside regex param | `Date.time:::/hh:mm/` | `Date.time` | `["","","","/hh:mm/"]` — verify colon inside `/…/` is not split |

---

### Part 3 – E2E tests (in `cmd/mock_e2e_test.go`)

All new tests use `--parse-str` via `suite.executeCommand("mock", "--parse-str", template)`.
Group new tests by mock function family in new `Test*` methods on either the existing
suites or new ones.  Follow the pattern of `MockDateSuite` — one suite per family.

**Naming convention for new suites:** `MockBooleanSuite`, `MockLoremSuite`,
`MockNumberSuite`, `MockRegexSuite`, `MockPersonSuite`, `MockAddressSuite`, etc.

Each suite requires the standard `executeCommand` helper (copy from `MockDateSuite`).

#### Sub-step 5a – Bare-value rejection e2e tests (new method on `MockCmdE2ETestSuite`)

- [x] **Step 7 – `TestCLIShouldInlineError_BareParamValues`**

  For `--parse-str`, errors from `processStr` are inlined in the output string as
  `[error text]`, not returned as a Go error. So test:
  ```go
  stdOut, err := suite.executeCommand("mock", "--parse-str", "{{ Number.number:2:1:5 }}")
  assert.NoError(suite.T(), err)
  assert.Contains(suite.T(), stdOut, "must be wrapped in")
  ```

  For `--parse-json`, `processJsonMap` returns a Go error, so:
  ```go
  _, err := suite.executeCommand("mock", "--parse-json", `{"n": "{{ Number.number:2:1:5 }}"}`)
  assert.Error(suite.T(), err)
  assert.Contains(suite.T(), err.Error(), "must be wrapped in")
  ```

  Include at least these template variants (covering different positions):
  - `{{ Number.number:2 }}` — bare first param
  - `{{ Number.number:{0}:5 }}` — bare second param, first is valid
  - `{{ Date.date:2020-01-01:{2030-01-01}:{} }}` — bare first of three params

#### Sub-step 5b – `Number.number` param edge cases

- [x] **Step 8 – `TestNumberNumber` on a new `MockNumberSuite`**

  | testName | template | assertion |
  |---|---|---|
  | no params (defaults) | `{{ Number.number }}` | output matches `^-?\d+\n$` |
  | empty decimals (default 0) | `{{ Number.number:{} }}` | matches `^-?\d+\n$` |
  | decimals=2 | `{{ Number.number:{2} }}` | matches `^-?\d+\.\d{2}\n$` |
  | decimals=0, min=1, max=10 | `{{ Number.number:{0}:{1}:{10} }}` | matches `^\d+\n$`; parsed int in [1,10] |
  | decimals=2, min and max empty (defaults) | `{{ Number.number:{2}:{}:{} }}` | matches `^-?\d+\.\d{2}\n$` |
  | non-numeric decimals (ignored, defaults to 0) | `{{ Number.number:{abc} }}` | no error, matches `^-?\d+\n$` |
  | three bare colons (all defaults) | `{{ Number.number::: }}` | no error, `^-?\d+\n$` — this must pass because no bare token is written |

#### Sub-step 5c – `Boolean.booleanWithChance` param edge cases

- [x] **Step 9 – `TestBooleanBooleanWithChance` on a new `MockBooleanSuite`**

  | testName | template | assertion |
  |---|---|---|
  | chance = 100 | `{{ Boolean.booleanWithChance:{100} }}` | output is `true\n` |
  | chance = 0 | `{{ Boolean.booleanWithChance:{0} }}` | output is `false\n` |
  | chance empty (fallback random bool) | `{{ Boolean.booleanWithChance:{} }}` | output matches `^(true|false)\n$` |
  | chance non-numeric (fallback random bool) | `{{ Boolean.booleanWithChance:{abc} }}` | output matches `^(true|false)\n$` |
  | no param at all | `{{ Boolean.booleanWithChance }}` | panics on `functionParams[0]` — see Notes below |

  NOTE on "no param at all": `Boolean.booleanWithChance` accesses `functionParams[0]`
  without a length guard, so calling it with `params = []string{}` panics. The plan
  records this as a known bug; the step below adds a regression test confirming the
  panic turns into a proper error once a guard is added in `mocker.go` — but fixing
  the guard itself is outside this plan's scope. For now, skip that sub-case or mark
  it xfail with a comment explaining the known issue.

  Similarly, `Lorem.paragraph`, `Lorem.paragraphs`, `Lorem.sentence`, `Lorem.sentences`,
  and `Lorem.words` all access `functionParams[0]` without a length check.

#### Sub-step 5d – `Lorem.*` param edge cases

- [x] **Step 10 – `TestLorem*` methods on a new `MockLoremSuite`**

  | testName | template | assertion |
  |---|---|---|
  | paragraph default (1 sentence) | `{{ Lorem.paragraph:{1} }}` | non-empty string |
  | paragraph N=3 | `{{ Lorem.paragraph:{3} }}` | non-empty string |
  | paragraph empty param (fallback) | `{{ Lorem.paragraph:{} }}` | no error |
  | paragraphs N=2 | `{{ Lorem.paragraphs:{2} }}` | output contains `\n` |
  | sentence N=5 | `{{ Lorem.sentence:{5} }}` | non-empty string |
  | sentences N=3 | `{{ Lorem.sentences:{3} }}` | output contains `\n` |
  | word (no params) | `{{ Lorem.word }}` | non-empty string |
  | words N=4 | `{{ Lorem.words:{4} }}` | non-empty string |

#### Sub-step 5e – `Regex.regex` param edge cases

- [x] **Step 11 – `TestRegexRegex` on a new `MockRegexSuite`**

  | testName | template | assertion |
  |---|---|---|
  | simple pattern | `{{ Regex.regex:/[a-z]{3}/ }}` | output matches `^[a-z]{3}\n$` |
  | digits pattern | `{{ Regex.regex:/[0-9]{4}/ }}` | output matches `^\d{4}\n$` |
  | empty regex (// — valid, generates empty string) | `{{ Regex.regex:// }}` | output is `\n` |
  | no params (returns error, inlined) | `{{ Regex.regex }}` | output contains `[regex function requires` |
  | bare param (post-enforcement error) | `{{ Regex.regex:[a-z]{3} }}` | output contains `must be wrapped in` |
  | pattern not wrapped in slashes | `{{ Regex.regex:{[a-z]{3}} }}` — value param, not regex | output contains `must be wrapped in /.../ ` (from `extractRegex`) |

  Note: when `Regex.regex` receives a `{…}`-wrapped value param (not a `/…/` regex),
  `extractRegex` returns an error because the stripped value `[a-z]{3}` does not start
  and end with `/`.

#### Sub-step 5f – `Date.*` param edge cases (extending existing `MockDateSuite`)

- [x] **Step 12 – Extend `TestDateDate_Errors` and add `TestDateDate_Params`**

  Additional error cases:
  | testName | template | expected inline error |
  |---|---|---|
  | invalid from date | `{{ Date.date:{not-a-date}:{2030-01-01}:{} }}` | no error (silently uses default from); just assert output matches date regex |
  | invalid format (not wrapped in //) | `{{ Date.date:{}:{}:{YYYY-MM-DD} }}` | output contains `must be wrapped in` (bare param, caught in extractMockMethod) |
  | format not in slashes (value param) | `{{ Date.date:{}:{}:{YYYY-MM-DD} }}` | (same as above — post enforcement) |

  Valid param variations for `Date.date`:
  | testName | template | assertion |
  |---|---|---|
  | from only | `{{ Date.date:{2020-01-01}:{}:{} }}` | matches `^\d{4}-\d{2}-\d{2}\n$` |
  | from and to | `{{ Date.date:{2025-01-01}:{2025-12-31}:{} }}` | year is 2025 |
  | all three params | `{{ Date.date:{2020-01-01}:{2020-12-31}:/YYYY/ }}` | matches `^2020\n$` |
  | empty from and to | `{{ Date.date:{}:{}:{} }}` | matches `^\d{4}-\d{2}-\d{2}\n$` |

  Similarly, add `TestDatetime_Params` and `TestDatetime_Errors` covering:
  - valid from/to in `YYYY-MM-DDThh:mm` format
  - from/to in date-only `YYYY-MM-DD` format (parseDatetimeFull fallback)
  - from after to (error)
  - all params empty (defaults)

  And `TestDateTime_Params`:
  - from = `08:00`, to = `20:00`, output hour should be 8..20
  - from after to (error inlined)
  - format override `/hh:mm/`

#### Sub-step 5g – `--parse-json` edge cases on `MockCmdE2ETestSuite`

- [x] **Step 13 – `TestCLIParseJson_BareValues`** (covered partially in Step 7)

- [x] **Step 14 – `TestCLIParseJson_NestedStructures`**

  | testName | template json | assertion |
  |---|---|---|
  | nested object with params | `{"person":{"name":"{{ Person.name }}","age":"{{ Number.number:{0}:{18}:{90} }}"}}` | no error; name non-empty; age numeric 18-90 |
  | array key generates multiple objects | `{"phones[3]":"{{ Person.phoneNumber }}"}` | output has 3 phone numbers |
  | inner object array | `{"employees[2]":{"name":"{{ Person.name }}"}}` | output has `"employees"` with 2 entries |
  | invalid value type (integer) | `{"x":123}` | error |
  | unknown mock function | `{"x":"{{ Unknown.function }}"}` | error containing `unknown mock function` |

- [x] **Step 15 – `TestCLIParseJson_GenerateFlag`**

  | testName | args | assertion |
  |---|---|---|
  | generate=3, single object | `--parse-json {"name":"{{ Person.name }}"} --generate 3` | output is JSON array with 3 elements |
  | generate=1 (default), output is object not array | `--parse-json {"name":"{{ Person.name }}"}` | output is JSON object (not array) |

#### Sub-step 5h – `processStr` edge cases on `MockCmdE2ETestSuite`

- [x] **Step 16 – `TestCLIParseStr_EdgeCases`**

  | testName | input string | assertion |
  |---|---|---|
  | literal only (no mock) | `Hello world` | output is `Hello world\n` |
  | unclosed `{{` | `Hello {{ Person.name` | output is `Hello {{ Person.name\n` (literal) |
  | two mock functions in one string | `{{ Person.firstName }} {{ Person.lastName }}` | output contains space between two non-empty tokens |
  | unknown function inlined as error | `{{ Unknown.fn }}` | output contains `[unknown mock function 'Unknown.fn']` |
  | bare param inlined as error (post-enforcement) | `{{ Number.number:5 }}` | output contains `[mock function parameter '5' must be wrapped in` |
  | empty mock call `{{  }}` | `{{  }}` | output contains `[unknown mock function '']` |
  | mock surrounded by literals | `start {{ Person.name }} end` | starts with `start `, ends with ` end\n` |

---

## Notes

### Bare-value detection and the function name token

The function name (the first `:` delimited token) is always bare by its nature —
`Address.city` has no `{…}` around it.  The `inBare` flag and error must only activate
**after** the first `:` has been seen (i.e. `paramCount >= 1`).

### Empty tokens are never "bare"

Consecutive colons (`::`) or an empty `{}` produce an empty string param.  Because
`inBare` is only set when a non-delimiter character is written to `buf` outside `{…}`
or `/…/`, an empty token never sets `inBare = true`.  This means `:::` (all defaults)
still works and is valid.

### `Boolean.booleanWithChance` and `Lorem.*` out-of-bounds panic

Several mocker cases in `mocker/mocker.go` index `functionParams[0]` without checking
`len(functionParams) > 0`:
- `Boolean.booleanWithChance` (line 179)
- `Lorem.paragraph` (line 267)
- `Lorem.paragraphs` (line 273)
- `Lorem.sentence` (line 279)
- `Lorem.sentences` (line 285)
- `Lorem.words` (line 296)

When no params are passed (e.g. `{{ Boolean.booleanWithChance }}`), `params` is
`[]string{}`, and these functions panic.  Fixing those guards is a separate bug fix
task and is out of scope here.  The edge case tests in this plan must pass only
`{…}`-wrapped params (or at least one element) to avoid triggering the panic.  Add a
`// TODO: known panic if no param supplied — see bug_loremboolean_no_param_guard` comment
next to any skipped test sub-case.

### `processStr` vs `processJsonMap` error propagation

`processStr` (used by `--parse-str`) never returns a Go error — it inlines errors as
`[error text]` in the output string.  `processJsonMap` (used by `--parse-json` and
`--parse-files`) propagates Go errors that surface as `cmd.Execute()` errors.  E2E
tests must assert accordingly.

### `extractMockMethod` change is purely additive to the parser state machine

The flag `inBare` requires no backtracking and adds O(1) state.  The only behavioral
change: previously, `Number.number:2:1:5` silently produced params `["2","1","5"]`.
After this change it returns an error.  All existing valid usages (only `{…}` or `/…/`
wrapped params) continue to work unchanged.

### Test file for `mock_test.go` lives in package `cmd` (not `cmd_test`)

Unit tests for internal helpers (`extractMockMethod`, `interpretString`, etc.) are in
`package cmd` (white-box).  E2e tests in `mock_e2e_test.go` are in `package cmd_test`
(black-box).  New tests must follow the same split.

### Updating the `Long` description examples

The `Long` string in `NewMockCmd` still contains `Number.number::1:100` in the
Examples block.  After enforcement, that would fail if interpreted literally.  Replace
with `Number.number:::{1}:{100}` (three-param version: decimals empty, min=1, max=100)
or `Number.number:{0}:{1}:{100}`.
