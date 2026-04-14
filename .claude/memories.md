# Project Memory

## Architecture Decisions

- **`workingDir()` uses `os.Getwd()`** — the default output path for `--to-json-file` and `--to-csv-file` (when source is `parse-json`) is resolved via `os.Getwd()`, not `os.Executable()`. `os.Executable()` returns the temp binary path under `go run .`, so files would be lost. `workingDir()` replaced the former `executableDir()` helper.
- **`Mock` is not goroutine-safe** — each goroutine must call `mocker.New()` independently; never share a `*Mock` across goroutines.
- **Per-instance `*rand.Rand` seeding** — `mocker.New()` seeds with three XOR factors: `time.Now().UnixNano() ^ (int64(os.Getpid()) * 0x517cc1b727220a95) ^ (mockerSeedCounter.Add(1) * -7046029254386353131)`. The last factor uses the Fibonacci hashing multiplier (`-7046029254386353131` = `0x9e3779b97f4a7c15` signed). `mockerSeedCounter` is a package-level `atomic.Int64`.
- **Inner split for `[n]` keys is implemented for `generate == 1` only** — when `generate > 1` with inner `[n]` keys, `analyzeTemplate` returns `Depth: 0` (root split) as a fallback. See `TODO(concurrency-generate-gt1-inner-split)` in `cmd/mock.go`.
- **Inner split buffers the inner array in the writer** — `streamOutput` accumulates all inner-array sub-batches before writing the parent object when `splitPoint.Depth > 0`. This is O(inner_count × item_size) memory. See `TODO(concurrency-inner-streaming)` in `cmd/mock.go`.
- **Sibling `[n]` keys at the same depth**: only the first qualifying sibling is used as the split node; the others fall inside the worker's sequential processing. Full sibling-sequential pool dispatch is deferred (`TODO(concurrency-siblings)`).
- **`streamOutput` now accepts a `SplitPoint` parameter** — replaces the previous implicit assumption that all results are complete root objects.
- **Inner split only supports map[string]any values at depth 1** — `analyzeTemplate` checks that the split key's value is a `map[string]any` and that `splitDepth == 1`. String values and depth > 1 nodes fall back to root split.
- **Inner split sentinel batch** — `runWorkerPoolInner` sends one sentinel batch first containing `{"_ktns_parent_ctx_": parentCopy}`. `streamOutput` receives this as the parent context and all subsequent batches as inner-array items to assemble.
- **`_use_default_` sentinel for optional-value flags** — `--to-json-file` and `--to-csv-file` use `NoOptDefVal = "_use_default_"` so the flag can be passed without a value. Wherever you see `toJsonFileValue != "" && toJsonFileValue != "_use_default_"`, that is the explicit-filename branch. Do NOT replace this with a simple empty-string check.
- **`generate == 1` produces a bare object, not a single-element array** — both `routeOutput` (sequential) and `streamOutput` (concurrent) have explicit `generate == 1` branches that omit the wrapping `[…]`. Both paths must stay consistent.
- **Atomic file writes** — `atomicFileCreate` writes to a temp file in the same directory as the destination (same-filesystem requirement for `os.Rename`), then renames on success or deletes on error. All file sinks go through this.
- **`--parse-json-file` requires `.template.json` suffix** — enforced in `RunE` before any I/O. The default output path strips `.template.json` → `.json` (or `.csv`).
- **`--parse-str` has no output routing flags** — it always writes to stdout. `--to-stdout`, `--to-json-file`, and `--to-csv-file` are incompatible with `--parse-str`; this is validated explicitly.
- **`streamOutput` maintains `csvHeaderWritten` separately from `firstItem`** — `firstItem` gates JSON separators; `csvHeaderWritten` gates the CSV header row. Both exist for distinct reasons; do not merge them.
- **`mocker.rng` (not global `rand`)** — `Person.cpf`, `Company.cnpj`, and `Payment.creditCardCvv` all use `m.rng` (the per-instance `*rand.Rand`). Global `rand.Intn` calls were a pre-concurrency bug; they were fixed in the concurrency plan (2026-04-10).
- **`Payment.creditCardCvv` uses a per-instance `regen.Generator`** — created at `mocker.New()` time, stored as `m.cvvGen`, backed by `m.rng`. The `Generate()` case also has a defensive re-init if `m.cvvGen == nil`. Never uses the package-level `regen.Generate`.
- **`Regex.regex` creates a new `regen.Generator` per call** — backed by `m.rng`, so it is per-instance in that sense, but the generator itself is not stored (unlike `cvvGen`). Strips `/…/` outer slashes before calling `regen.NewGenerator`.
- **JSON file output is always pretty-printed** — both `routeOutput` and `streamOutput` use `json.MarshalIndent` for file writes regardless of `--to-stdout-prettify`. `--to-stdout-prettify` only affects stdout JSON.
- **No progress bar** — `github.com/vbauerster/mpb/v8` and its `--no-progress` flag were removed entirely (2026-04-10). Do not re-introduce them.
- **`--debug` writes to `os.Stderr` only** — the live-progress display is always stderr; stdout/file output is unaffected.
- **`concurrencyThreshold = 10_000`** — workloads below this go through the sequential path (`routeOutput`); at or above, the worker pool (`runWorkerPool` + `streamOutput`) is used.
- **`targetSubBatch = 750`** — items per `WorkUnit` at the split level; divided by `InnerWeight` when `[n]` keys exist.

## Known Gotchas

- **`Generate` param-guard pattern** — any `Generate` case that reads `functionParams[0]` must first check `len(functionParams) == 0 || functionParams[0] == ""` and return `fmt.Errorf(...)`. The canonical example is `Regex.regex` (line 392). Empty braces `{}` produce `functionParams[0] = ""`, which is treated the same as "no parameter supplied".
- **`Number.number` uses `m.rng.Float64()`** — after the Bug 2 fix, `Number.number` no longer calls `jaswdrFaker.Float64`. It computes `min + m.rng.Float64()*(max-min)` and formats with `strconv.FormatFloat(..., 'f', decimals, 64)`. `min`/`max` are kept as `float64` throughout; no `int()` truncation.
- **`extractMockMethod` bare-token error** — any parameter after the function name that is not wrapped in `{…}` or `/…/` is an error, detected via the `inBare` flag. The function name itself (before the first `:`) is always bare and is exempt.
- **`interpretString` vs `processStr`** — `interpretString` only matches a value that is *entirely* `{{ … }}` (used in JSON map processing). `processStr` handles inline expressions mixed with literal text (used in `--parse-str`).
- **Map iteration order is non-deterministic** — `processJsonMap` snapshots `objKeys` before iterating so key deletion/addition during the loop does not cause issues. `sanitizeJsonMap` does the same.
- **CSV header derivation** — in `streamOutput`, headers are derived from the first item of the first batch (map key order is non-deterministic, so they are `sort.Strings`-sorted). This requires that all generated objects have the same top-level keys, which is guaranteed by the homogeneous template.
- **`os.Rename` across filesystems fails** — `atomicFileCreate` places the temp file in `filepath.Dir(finalPath)` specifically to avoid cross-device rename errors.
- **`--to-json-file` and `--to-csv-file` require `=` syntax for explicit filenames** — because `NoOptDefVal = "_use_default_"` is set, pflag treats a bare `--to-json-file value` as the flag followed by a separate positional argument. Explicit filenames must use `--to-json-file=myfile.json`. The `Long` help text and README both document this with the `Note:` lines.
- **`cmd.Flags().Changed("to-json-file")`** — use `Changed()` to distinguish "flag not passed" from "flag passed without value"; `GetString` alone cannot make this distinction because cobra sets the value to `NoOptDefVal` when the flag is passed bare.
- **`mocker` import alias shadowing** — inside `NewMockCmd`'s `RunE`, a local variable is named `mocker` (`mocker := mocker.New()`), shadowing the package import. This is intentional and pre-existing; do not rename the package.
- **`Date.*` format parameters use `/…/` for format strings** — e.g. `{{ Date.date:::/YYYY-MM-DD/ }}`. Colons inside `/…/` are never treated as delimiters.
- **`extractDigitInBrackets` only handles `"object"` place** — the `"file"` branch was removed in the `parse-json-file` refactor (2026-04-09). Passing any place other than `"object"` returns an error immediately.
- **`processJsonMap` does not increment `keyIndex` after expanding a `generateAmount > 1` map** — it re-processes the same key on the next iteration (now a `[]any`) to recurse into each copied map. This is intentional loop logic, not a bug.

## Current Feature State

| Feature | Status |
| --- | --- |
| `mock --parse-str` | Done |
| `mock --parse-json` | Done |
| `mock --parse-json-file` (single `.template.json`) | Done |
| `mock --generate` (root objects) | Done |
| `mock --to-stdout as-json\|as-csv` | Done |
| `mock --to-stdout-prettify` | Done |
| `mock --to-json-file [filename]` | Done |
| `mock --to-csv-file [filename]` | Done |
| `mock --debug` (live stderr progress) | Done |
| Worker pool concurrency (root split only) | Done |
| Inner split for depth > 0 `[n]` keys (`generate == 1`) | Done (after this plan) |
| Inner split for depth > 0 `[n]` keys (`generate > 1`) | TODO — `TODO(concurrency-generate-gt1-inner-split)` |
| Sibling `[n]` keys sequential pool dispatch | TODO — `TODO(concurrency-siblings)` |
| `request` command | Partially implemented (flags exist, logic in `cmd/request.go`) |
| `stress` command | Not yet implemented |

## Mock Functions Implemented

`Address`: latitude, longitude, postCode, country, state, city, streetName, buildingNumber  
`Boolean`: boolean, booleanWithChance:{chance}  
`Car`: maker, model, plate  
`Company`: name, suffix, catchPhrase, bs, jobTitle, cnpj  
`Currency`: currencyCode, currencyContry, currencyName, currencyNumber  
`Date`: date:{from}:{to}:/format/, time:{from}:{to}:/format/, datetime:{from}:{to}:/format/, now:/format/  
`File`: filenameWithExtension, extension  
`Internet`: domain, email, ipv4, macAddress, password, url  
`Lorem`: paragraph:{sentences}, paragraphs:{paragraphs}, sentence:{words}, sentences:{sentences}, word, words:{words}  
`Number`: number:{decimals}:{min}:{max}  
`Payment`: creditCardExpirationDate, creditCardNumber, creditCardType, creditCardCvv  
`Person`: name, firstName, lastName, phoneNumber, email, cpf  
`Regex`: regex:/regex/  
`UUID`: uuidv4, uuidv7  
`UserAgent`: userAgent  

## Removed / Superseded

- `--parse-files` (glob/directory multi-file flag) — replaced by `--parse-json-file` (2026-04-09)
- `--preserve-folder-structure` — removed along with `--parse-files`
- `--no-progress` — removed with `mpb` library (2026-04-10)
- `github.com/vbauerster/mpb/v8` — removed entirely (2026-04-10); do not re-add
- `Time.date` stub — replaced by four `Date.*` functions (2026-04-07)
- Old bare-colon param syntax (e.g. `Number.number::18:50`) — replaced by `{…}` wrapping (`Number.number::{18}:{50}`) (2026-04-07)

## Plan History (newest first)

| Plan | Date | Type |
| --- | --- | --- |
| `202604141746_bug_working_dir_and_tojsonfile_docs` | 2026-04-14 | bug |
| `202604131510_refactor_mock_concurrency` | 2026-04-13 | refactor |
| `202604131357_bug_paramguard_number_truncation_interface` | 2026-04-13 | bug |
| `202604101540_feature_mock_concurrency` | 2026-04-10 | feature |
| `202604101112_feature_to_csv_file` | 2026-04-10 | feature |
| `202604101057_refactor_remove_progress_bar` | 2026-04-10 | refactor |
| `202604091308_feature_more_useful_mock_flags` | 2026-04-09 | feature |
| `202604091047_refactor_parse_json_file` | 2026-04-09 | refactor |
| `202604080035_feature_uuid_uuidv7` | 2026-04-08 | feature |
| `202604071520_feature_mock_param_syntax_and_edge_case_tests` | 2026-04-07 | feature |
| `202604071310_bug_mocker_param_parsing` | 2026-04-07 | bug |
| `202604070920_feature_date_mock_functions` | 2026-04-07 | feature |
