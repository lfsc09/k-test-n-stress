# Bug: Param Guard, Number Float Truncation, Silent Error Fall-through, Unused Interface

> Created: 2026-04-13 13:57

## Summary

Five related bugs in `mocker/mocker.go` and one test update in `cmd/mock_e2e_test.go`. The core issues are: (1) six parameterised `Generate` cases unconditionally index `functionParams[0]` without a bounds check, causing a panic when the caller supplies no parameters; (2) `Number.number` truncates its `float64` `min`/`max` to `int` before generating the value, silently ruining fractional bounds; (3) `Boolean.booleanWithChance` and all five `Lorem.*` cases silently swallow parse errors instead of returning them; (4) the `Mocker` interface is declared but never consumed by any other package and should be removed. After fixing the mocker, the e2e tests tagged `bug_loremboolean_no_param_guard` must be updated to add the previously-unsafe no-param test cases and to assert errors where the new code returns them.

## Affected Files

- `mocker/mocker.go` — remove `Mocker` interface; add bounds guards to six `Generate` cases; fix `Number.number` float generation; convert silent error fall-backs to `fmt.Errorf` returns
- `cmd/mock_e2e_test.go` — remove the two `TODO` comments tagged `bug_loremboolean_no_param_guard`; add no-param test cases for `Boolean.booleanWithChance` and `Lorem.*` functions; update the "non-numeric / empty" test cases for `Boolean.booleanWithChance` and `Lorem.*` to assert errors instead of a silent fallback

## Steps

- [x] **Step 1: Remove the `Mocker` interface from `mocker/mocker.go`**

  Delete the entire block at the top of the file (lines 18–21):

  ```
  type Mocker interface {
      List(out io.Writer)
      Generate(mockFunction string, functionParams []string) (string, error)
  }
  ```

  The `io` import is still required by `List(out io.Writer)` on `*Mock`, so do NOT remove it from the import block. After deletion, verify the file still compiles (`go build ./mocker/...`).

- [x] **Step 2: Add a bounds guard to `Boolean.booleanWithChance` and return a hard error on parse failure**

  Current code (lines 197–202):
  ```go
  case "Boolean.booleanWithChance":
      chance, err := strconv.Atoi(functionParams[0])
      if err != nil {
          return strconv.FormatBool(m.jaswdrFaker.Boolean().Bool()), nil
      }
      return strconv.FormatBool(m.jaswdrFaker.Boolean().BoolWithChance(chance)), nil
  ```

  Replace with:
  ```go
  case "Boolean.booleanWithChance":
      if len(functionParams) == 0 || functionParams[0] == "" {
          return "", fmt.Errorf("Boolean.booleanWithChance: 'chance' parameter is required")
      }
      chance, err := strconv.Atoi(functionParams[0])
      if err != nil {
          return "", fmt.Errorf("Boolean.booleanWithChance: 'chance' must be an integer, got %q", functionParams[0])
      }
      return strconv.FormatBool(m.jaswdrFaker.Boolean().BoolWithChance(chance)), nil
  ```

  Note: the existing tests for `chance = {}` (empty braces) pass `functionParams[0] = ""` — this will now become an error. The e2e test must be updated in Step 7 to reflect this.

- [x] **Step 3: Add a bounds guard and hard error to `Lorem.paragraph`**

  Current code (lines 285–290):
  ```go
  case "Lorem.paragraph":
      sentences, err := strconv.Atoi(functionParams[0])
      if err != nil {
          return m.jaswdrFaker.Lorem().Paragraph(1), nil
      }
      return m.jaswdrFaker.Lorem().Paragraph(sentences), nil
  ```

  Replace with:
  ```go
  case "Lorem.paragraph":
      if len(functionParams) == 0 || functionParams[0] == "" {
          return "", fmt.Errorf("Lorem.paragraph: 'sentences' parameter is required")
      }
      sentences, err := strconv.Atoi(functionParams[0])
      if err != nil {
          return "", fmt.Errorf("Lorem.paragraph: 'sentences' must be an integer, got %q", functionParams[0])
      }
      return m.jaswdrFaker.Lorem().Paragraph(sentences), nil
  ```

- [x] **Step 4: Add a bounds guard and hard error to `Lorem.paragraphs`**

  Current code (lines 291–296):
  ```go
  case "Lorem.paragraphs":
      paragraphs, err := strconv.Atoi(functionParams[0])
      if err != nil {
          return strings.Join(m.jaswdrFaker.Lorem().Paragraphs(1), ""), nil
      }
      return strings.Join(m.jaswdrFaker.Lorem().Paragraphs(paragraphs), "\n"), nil
  ```

  Replace with:
  ```go
  case "Lorem.paragraphs":
      if len(functionParams) == 0 || functionParams[0] == "" {
          return "", fmt.Errorf("Lorem.paragraphs: 'paragraphs' parameter is required")
      }
      paragraphs, err := strconv.Atoi(functionParams[0])
      if err != nil {
          return "", fmt.Errorf("Lorem.paragraphs: 'paragraphs' must be an integer, got %q", functionParams[0])
      }
      return strings.Join(m.jaswdrFaker.Lorem().Paragraphs(paragraphs), "\n"), nil
  ```

- [x] **Step 5: Add a bounds guard and hard error to `Lorem.sentence`**

  Current code (lines 297–302):
  ```go
  case "Lorem.sentence":
      words, err := strconv.Atoi(functionParams[0])
      if err != nil {
          return m.jaswdrFaker.Lorem().Sentence(1), nil
      }
      return m.jaswdrFaker.Lorem().Sentence(words), nil
  ```

  Replace with:
  ```go
  case "Lorem.sentence":
      if len(functionParams) == 0 || functionParams[0] == "" {
          return "", fmt.Errorf("Lorem.sentence: 'words' parameter is required")
      }
      words, err := strconv.Atoi(functionParams[0])
      if err != nil {
          return "", fmt.Errorf("Lorem.sentence: 'words' must be an integer, got %q", functionParams[0])
      }
      return m.jaswdrFaker.Lorem().Sentence(words), nil
  ```

- [x] **Step 6: Add a bounds guard and hard error to `Lorem.sentences`**

  Current code (lines 303–308):
  ```go
  case "Lorem.sentences":
      sentences, err := strconv.Atoi(functionParams[0])
      if err != nil {
          return strings.Join(m.jaswdrFaker.Lorem().Sentences(1), ""), nil
      }
      return strings.Join(m.jaswdrFaker.Lorem().Sentences(sentences), "\n"), nil
  ```

  Replace with:
  ```go
  case "Lorem.sentences":
      if len(functionParams) == 0 || functionParams[0] == "" {
          return "", fmt.Errorf("Lorem.sentences: 'sentences' parameter is required")
      }
      sentences, err := strconv.Atoi(functionParams[0])
      if err != nil {
          return "", fmt.Errorf("Lorem.sentences: 'sentences' must be an integer, got %q", functionParams[0])
      }
      return strings.Join(m.jaswdrFaker.Lorem().Sentences(sentences), "\n"), nil
  ```

- [x] **Step 7: Add a bounds guard and hard error to `Lorem.words`**

  Current code (lines 311–316):
  ```go
  case "Lorem.words":
      words, err := strconv.Atoi(functionParams[0])
      if err != nil {
          return strings.Join(m.jaswdrFaker.Lorem().Words(1), ""), nil
      }
      return strings.Join(m.jaswdrFaker.Lorem().Words(words), " "), nil
  ```

  Replace with:
  ```go
  case "Lorem.words":
      if len(functionParams) == 0 || functionParams[0] == "" {
          return "", fmt.Errorf("Lorem.words: 'words' parameter is required")
      }
      words, err := strconv.Atoi(functionParams[0])
      if err != nil {
          return "", fmt.Errorf("Lorem.words: 'words' must be an integer, got %q", functionParams[0])
      }
      return strings.Join(m.jaswdrFaker.Lorem().Words(words), " "), nil
  ```

- [x] **Step 8: Fix `Number.number` — replace jaswdrFaker.Float64 with direct rng arithmetic**

  Current code (line 333):
  ```go
  return strconv.FormatFloat(m.jaswdrFaker.Float64(decimals, int(min), int(max)), 'f', decimals, 64), nil
  ```

  The bug is that `int(min)` and `int(max)` truncate fractional bounds (e.g. `min=0.5, max=0.9` becomes `int(0)` and `int(0)`, always producing `0`). Additionally, `jaswdrFaker.Float64` uses its own RNG, not `m.rng`, breaking per-instance determinism.

  Replace the entire `Number.number` case with:
  ```go
  case "Number.number":
      decimals := 0
      min := -1000.0
      max := 1000.0
      if len(functionParams) > 0 && functionParams[0] != "" {
          v, err := strconv.Atoi(functionParams[0])
          if err != nil {
              return "", fmt.Errorf("Number.number: 'decimals' must be an integer, got %q", functionParams[0])
          }
          decimals = v
      }
      if len(functionParams) > 1 && functionParams[1] != "" {
          v, err := strconv.ParseFloat(functionParams[1], 64)
          if err != nil {
              return "", fmt.Errorf("Number.number: 'min' must be a number, got %q", functionParams[1])
          }
          min = v
      }
      if len(functionParams) > 2 && functionParams[2] != "" {
          v, err := strconv.ParseFloat(functionParams[2], 64)
          if err != nil {
              return "", fmt.Errorf("Number.number: 'max' must be a number, got %q", functionParams[2])
          }
          max = v
      }
      if max < min {
          return "", fmt.Errorf("Number.number: 'max' (%v) must be >= 'min' (%v)", max, min)
      }
      var value float64
      if max == min {
          value = min
      } else {
          value = min + m.rng.Float64()*(max-min)
      }
      return strconv.FormatFloat(value, 'f', decimals, 64), nil
  ```

  Key changes:
  - Uses `m.rng.Float64()` exclusively — no `jaswdrFaker.Float64` call — preserving per-instance RNG.
  - Preserves `min`/`max` as `float64` throughout; no truncation to `int`.
  - `strconv.FormatFloat(..., 'f', decimals, 64)` already rounds to `decimals` decimal places via Go's formatting.
  - Converts the previously-silent parse errors (`_`) into explicit returned errors.
  - Adds a `max < min` guard.
  - Handles the degenerate `max == min` case by returning `min` directly (avoids `rng.Float64() * 0`).

- [x] **Step 9: Update `cmd/mock_e2e_test.go` — `TestBooleanBooleanWithChance`**

  Locate `TestBooleanBooleanWithChance` (currently around line 740). The test table currently has:
  - `"chance empty (fallback random bool)"` — asserts no error and a bool output.
  - `"chance non-numeric (fallback random bool)"` — asserts no error and a bool output.
  - A `TODO` comment for the no-param case.

  After the fix, both the empty-param case and the no-param case now return errors. Update as follows:

  1. Remove the `TODO` comment line tagged `bug_loremboolean_no_param_guard`.
  2. Change the `"chance empty (fallback random bool)"` test case: set `exactOutput: ""`, `assertRe: nil`, and add an `assertError: true` field (see struct extension below).
  3. Change the `"chance non-numeric (fallback random bool)"` test case the same way: `assertError: true`.
  4. Add a new test case: `{testName: "no param (error)", template: "{{ Boolean.booleanWithChance }}", assertError: true}`.

  The test struct needs an `assertError bool` field. The loop body must check `if tt.assertError { assert.Error(...) } else { assert.NoError(...) }` and skip regex/exact-output assertions when an error is expected.

  The template `"{{ Boolean.booleanWithChance }}"` with no colon causes `extractMockMethod` to return `params = []string{}`. The `Generate` call will receive `functionParams = []string{}` and the new guard will return an error.

- [x] **Step 10: Update `cmd/mock_e2e_test.go` — `TestLorem`**

  Locate `TestLoremSuite.TestLorem` (currently around line 785). The test table currently has:
  - `"paragraph empty param (fallback)"` — asserts no error and non-empty output.
  - A single `TODO` comment for no-param cases.

  After the fix, empty-param cases (`{}`) and no-param cases all return errors. Update as follows:

  1. Remove the `TODO` comment line tagged `bug_loremboolean_no_param_guard`.
  2. Add an `assertError bool` field to the existing test struct.
  3. Change `"paragraph empty param (fallback)"` to `assertError: true` (template `{{ Lorem.paragraph:{} }}`).
  4. Add the following new error-asserting test cases:
     - `{testName: "paragraph no param (error)", template: "{{ Lorem.paragraph }}", assertError: true}`
     - `{testName: "paragraphs no param (error)", template: "{{ Lorem.paragraphs }}", assertError: true}`
     - `{testName: "sentence no param (error)", template: "{{ Lorem.sentence }}", assertError: true}`
     - `{testName: "sentences no param (error)", template: "{{ Lorem.sentences }}", assertError: true}`
     - `{testName: "words no param (error)", template: "{{ Lorem.words }}", assertError: true}`
     - `{testName: "paragraph empty param (error)", template: "{{ Lorem.paragraph:{} }}", assertError: true}` — update the existing entry
     - `{testName: "paragraphs empty param (error)", template: "{{ Lorem.paragraphs:{} }}", assertError: true}`
     - `{testName: "sentence empty param (error)", template: "{{ Lorem.sentence:{} }}", assertError: true}`
     - `{testName: "sentences empty param (error)", template: "{{ Lorem.sentences:{} }}", assertError: true}`
     - `{testName: "words empty param (error)", template: "{{ Lorem.words:{} }}", assertError: true}`
  5. Update the loop body: when `tt.assertError` is true, assert `err != nil` and skip the `assertNonEmpty` / `assertContains` checks.

- [x] **Step 11: Run tests and verify**

  Run `go test ./mocker/... ./cmd/...` (or `go test -race ./...`). All previously-passing tests must still pass. The newly added error-asserting cases must pass. No panics.

- [x] **Step 12: Apply `## Proposed Memory Updates` below to `.claude/memories.md`.**

## Notes

- `extractMockMethod` called with a bare token like `"Boolean.booleanWithChance"` (no colon) sets `params = []string{}`. The `Generate` call receives this empty slice. All six functions hit index 0 before checking length.
- The `Regex.regex` case (line 392) is the canonical correct pattern: `if len(functionParams) == 0 { return "", fmt.Errorf(...) }`. Follow it exactly.
- The existing "empty braces" tests (`{{ Boolean.booleanWithChance:{} }}`) pass `functionParams[0] = ""`. After Step 2 this becomes an error because the guard checks `functionParams[0] == ""`. This is intentional — an empty braces call is effectively "no parameter supplied" and should fail loudly rather than silently picking a default.
- For `Number.number`, the previous code called `strconv.Atoi` for `decimals` (correct) but `strconv.ParseFloat` for `min`/`max` and then cast to `int` before passing to `jaswdrFaker.Float64`. The new code keeps `min`/`max` as `float64` and uses `m.rng.Float64()` to avoid both the truncation bug and the jaswdrFaker RNG independence issue.
- `strconv.FormatFloat(value, 'f', decimals, 64)` handles rounding to `decimals` decimal places automatically via Go's formatting. No explicit `math.Round` is needed.
- Do not change the `Lorem.word` case — it has no parameters and is not affected.
- Do not change the `Number.number` `decimals` parse from `strconv.Atoi` to `strconv.ParseFloat`; decimals must remain an integer.
- The `Mocker` interface removal does not require any caller-side changes; a project-wide grep for `mocker.Mocker` should return zero hits before the change, confirming it is safe.

## Proposed Memory Updates

Add to **Known Gotchas** section:

- **`Generate` param-guard pattern** — any `Generate` case that reads `functionParams[0]` must first check `len(functionParams) == 0 || functionParams[0] == ""` and return `fmt.Errorf(...)`. The canonical example is `Regex.regex` (line 392). Empty braces `{}` produce `functionParams[0] = ""`, which is treated the same as "no parameter supplied".
- **`Number.number` uses `m.rng.Float64()`** — after the Bug 2 fix, `Number.number` no longer calls `jaswdrFaker.Float64`. It computes `min + m.rng.Float64()*(max-min)` and formats with `strconv.FormatFloat(..., 'f', decimals, 64)`. `min`/`max` are kept as `float64` throughout; no `int()` truncation.

Add to **Plan History** (newest first):

| `202604131357_bug_paramguard_number_truncation_interface` | 2026-04-13 | bug |
