# Bug: Mocker param parsing — colon inside param values breaks delimiter splitting

> Created: 2026-04-07 00:00

## Summary

The current param-splitting logic in `extractMockMethod` (`cmd/mock.go`) treats every unescaped `:` as a delimiter between positional parameters. This breaks when a param value itself contains a colon — most critically for `Date.time` and `Date.datetime` `from`/`to` strings (e.g. `18:00`) and for any custom format string that is not wrapped in `/…/`. The fix changes the convention so that every regular (non-regex) param value **must** be wrapped in curly braces `{value}`, while regex params continue to use `/…/`. The parser is updated to recognise `{…}` as a "value token" and strip the braces, and `:` outside both token types remains the delimiter. All existing tests, documentation strings, and example snippets must be updated to the new syntax.

## Affected Files

- `cmd/mock.go` — update `extractMockMethod` to parse `{…}` tokens; update the `Long` usage/example string
- `cmd/mock_test.go` — update `TestExtractMockMethod_ValidInputs` test cases to new syntax; add new cases covering `{…}` tokens and colons inside values
- `cmd/mock_e2e_test.go` — update every template string that uses the old `:param` syntax to `:{param}` syntax
- `mocker/mocker.go` — update the `List()` display strings to show new syntax in param signatures
- `README.md` — update every example that shows the old param syntax

## Steps

- [x] Step 1: **Update `extractMockMethod` in `cmd/mock.go`**

  The function currently has two "modes": normal character accumulation, and `inRegex` (activated by `/`). Add a third mode `inValue` activated by `{` and deactivated by `}`. Rules:

  - When `{` is encountered **outside** both `inRegex` and `inValue`: set `inValue = true`. Do **not** write `{` to the buffer.
  - When `}` is encountered while `inValue = true`: set `inValue = false`. Do **not** write `}` to the buffer.
  - When `/` is encountered **outside** `inValue`: toggle `inRegex` as before, and write `/` to the buffer.
  - When `:` is encountered **outside both** `inRegex` and `inValue`: treat as delimiter (flush buffer, reset) — same as today.
  - All other characters: write to buffer unconditionally.

  Error handling: if the input ends while `inValue` or `inRegex` is still `true`, treat whatever is in the buffer as the last token (current behaviour for unclosed regex; apply same leniency for unclosed `{`). This avoids a breaking panic — malformed input already produces wrong mock output rather than a crash.

  Resulting signature is unchanged: `func extractMockMethod(rawValue string) (string, []string)`.

- [x] Step 2: **Update `TestExtractMockMethod_ValidInputs` in `cmd/mock_test.go`**

  Update the existing test cases that used the old bare-value syntax, and add new cases to cover the new behaviour:

  - Rename/update `"mock function with params"` input from `"Boolean.booleanWithChance:10"` to `"Boolean.booleanWithChance:{10}"`, expected params `[]string{"10"}`.
  - Rename/update `"mock function with multiple params"` input from `"Function.with:multiple:params"` to `"Function.with:{multiple}:{params}"`, expected params `[]string{"multiple", "params"}`.
  - Add `"value param with colon inside"`: input `"Date.time:{18:00}:{20:00}"`, expected funcName `"Date.time"`, expected params `[]string{"18:00", "20:00"}`.
  - Add `"mixed value and regex params"`: input `"Date.time:{18:00}:{20:00}:/hh:mm/"`, expected funcName `"Date.time"`, expected params `[]string{"18:00", "20:00", "/hh:mm/"}`.
  - Add `"empty value param"`: input `"Number.number:{}:{5}:{10}"`, expected funcName `"Number.number"`, expected params `[]string{"", "5", "10"}` — an empty `{}`  becomes an empty string, preserving the existing "blank = use default" convention.
  - Keep existing regex-only cases unchanged (`Regex.regex:/…/` etc.), as regex parsing is not modified.

- [x] Step 3: **Update `TestProcessJsonMap_ValidInputs` in `cmd/mock_test.go`**

  The one test case that passes params (`"string value with params"`) uses `"{{ Boolean.booleanWithChance:10 }}"`. Update its value to `"{{ Boolean.booleanWithChance:{10} }}"`.

- [x] Step 4: **Update E2E test templates in `cmd/mock_e2e_test.go`**

  Every template string that feeds params with the old bare-colon syntax must be updated. Audit all template strings in the file:

  - `TestDateDate`: 
    - `"{{ Date.date:::/YYYY/ }}"` → `"{{ Date.date:::/YYYY/ }}"` — no change (regex already wrapped, empty slots become `{}`-less bare colons which are still fine as empty tokens; but for consistency change bare-empty params too — see note below).
    - `"{{ Date.date:::/YYYY-MM/ }}"` → same analysis.
  - `TestDateDate_Errors`:
    - `"{{ Date.date:2030-01-01:2020-01-01: }}"` → `"{{ Date.date:{2030-01-01}:{2020-01-01}: }}"` (the trailing `:` produces an empty token as before, which is acceptable — or wrap as `{}` for clarity; choose explicit `{}` for correctness). New value: `"{{ Date.date:{2030-01-01}:{2020-01-01}:{} }}"`.
  - `TestDateDatetime`:
    - `"{{ Date.datetime:::/DD-MM-YYYY hh:mm/ }}"` → no change (regex).
    - `"{{ Date.datetime:2020-01-01:2030-12-31: }}"` → `"{{ Date.datetime:{2020-01-01}:{2030-12-31}:{} }}"`.
    - `"{{ Date.datetime:2020-01-01:2030-12-31:/YYYY-MM-DD/ }}"` → `"{{ Date.datetime:{2020-01-01}:{2030-12-31}:/YYYY-MM-DD/ }}"`.
  - `TestCLIShouldMockFromParseStr`: the `"Hello {{ Person.name }}"` template has no params — no change.

  **Note on empty positional slots**: With the new parser, a bare `:` between two delimiters (i.e. `::`) still produces an empty string token `""` because the buffer is flushed on each `:`. This means templates like `Date.date:::/YYYY/` (where slots 0 and 1 are empty) continue to work without any change to those specific templates. Only templates where a non-empty bare value was passed need updating.

- [x] Step 5: **Update `List()` display strings in `mocker/mocker.go`**

  The `tableLineData` calls that show parameter signatures in the function name column must reflect the new `{…}` convention. Update these lines:

  - `"Boolean.booleanWithChance:[chance]"` → `"Boolean.booleanWithChance:{chance}"`
  - `"Lorem.paragraph:[sentences]"` → `"Lorem.paragraph:{sentences}"`
  - `"Lorem.paragraphs:[paragraphs]"` → `"Lorem.paragraphs:{paragraphs}"`
  - `"Lorem.sentence:[words]"` → `"Lorem.sentence:{words}"`
  - `"Lorem.sentences:[sentences]"` → `"Lorem.sentences:{sentences}"`
  - `"Lorem.words:[words]"` → `"Lorem.words:{words}"`
  - `"Number.number:[decimals]:[min]:[max]"` → `"Number.number:{decimals}:{min}:{max}"`
  - `"Regex.regex:[regex]"` → `"Regex.regex:/regex/"` (regex is already `/…/`, so the display should reflect that)
  - `"Date.date:[from]:[to]:[format]"` → `"Date.date:{from}:{to}:/format/"`
  - `"Date.time:[from]:[to]:[format]"` → `"Date.time:{from}:{to}:/format/"`
  - `"Date.datetime:[from]:[to]:[format]"` → `"Date.datetime:{from}:{to}:/format/"`
  - `"Date.now:[format]"` → `"Date.now:/format/"`

- [x] Step 6: **Update the `Long` usage string in `cmd/mock.go`**

  The `Long` field on `NewMockCmd` contains inline documentation with examples using the old syntax. Update:

  - The bullet point `* Always call the mock function with the format {{ functionName:arg1:arg2:argN }}.` → update the example format to `{{ functionName:{arg1}:{arg2}:{argN} }}` (or `{{ functionName:{arg1}:{arg2}:{argN} }}`).
  - The bullet point explaining colon-as-separator and regex-in-slashes — rewrite to explain the new `{value}` convention and that `/regex/` continues to work for regex params.
  - The Examples section: update `{{ Number.number::1:100 }}` → `{{ Number.number:::{1}:{100} }}` ... actually the existing example `Number.number::1:100` maps to `Number.number:<decimals=blank>:<min=1>:<max=100>`. Under the new syntax this becomes `Number.number::{1}:{100}`. Update all inline examples in the `Long` string accordingly.

- [x] Step 7: **Update `README.md`**

  Find and update every example that shows the old bare-value param syntax:

  - Line 74: `{{ Number.number::1:100 }}` → `{{ Number.number::{1}:{100} }}`
  - Line 167 (json example): `"Number.number::18:50"` → `"Number.number::{18}:{50}"`
  - Lines 392-394 (request examples): `{{ Number.number:0:1:10 }}` → `{{ Number.number:{0}:{1}:{10} }}` and `{{ Number.number:0:50:55 }}` → `{{ Number.number:{0}:{50}:{55} }}`
  - Any other occurrence of the old `:value:` pattern in mock function call examples. Search thoroughly for `Number.number:` and similar patterns.

- [x] Step 8: **Verify tests pass**

  Run `go test ./...` and confirm all tests pass. Pay particular attention to:
  - `TestExtractMockMethod_ValidInputs` — all new and updated cases.
  - `TestProcessJsonMap_ValidInputs` — the updated `Boolean.booleanWithChance` case.
  - `TestDateTime` and `TestDateDate` e2e suites — the templates with updated syntax.
  - `TestDateDate_Errors` — the updated error template.

## Notes

- **Backward compatibility**: The new syntax is a breaking change for any user who has existing template files or scripts using the old bare-value `param` style. This is intentional and noted in the bug description. No migration shim is planned.
- **Empty params (`{}` vs bare `::`)**: An empty `{}` and a bare empty token (two consecutive `::`) both produce `""` in `functionParams`, which is already how "use default" is signalled in `mocker.go`. Both forms will remain valid after this change because a bare `:` outside any token type is still a delimiter producing an empty buffer flush. This is acceptable — users can omit braces for empty positional slots.
- **`{` and `}` characters in regex params**: Since regex params are already handled by the `/…/` mode and the `{…}` mode is only entered outside of `inRegex`, curly braces inside a `/…/` param (e.g. `/[a-z]{3}/`) are never interpreted as value delimiters. No conflict exists.
- **`inValue` nesting**: The parser does not support nested `{…}` — a second `{` inside an already-open `inValue` block is written to the buffer as a literal character. This is consistent with how `/` inside an already-open regex is written literally.
- **`request.go`**: This file calls `processStr` and `processJsonMap` (both defined in `mock.go`) which call `extractMockMethod` internally. It does not define its own parsing, so updating `extractMockMethod` fixes it automatically. However, the README examples for the `request` command (lines 392-394) still need their inline param strings updated as noted in Step 7.
- **`mocker/mocker.go` `Generate` function**: No changes are needed inside `Generate` itself. The `functionParams []string` slice it receives is already parsed and stripped of delimiters and wrappers by `extractMockMethod` before being passed in. `Date.time`, `Date.datetime`, and all other cases continue to receive the raw param value (e.g. `"18:00"`) without the `{…}` wrapper.
