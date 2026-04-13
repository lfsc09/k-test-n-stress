![Go Badge](https://img.shields.io/badge/Go-1.26.1-00ADD8.svg?style=for-the-badge&logo=Go&logoColor=white)

# The project

k-test-n-stress is a simple tool written with GO to facilate:

1. `[mock]` Generate fake data, with specific outputs.
2. `[request]` Generate http requests to specified URLs, with a variety of options.
3. `[stress]` Stress test specific URLs, with a variety of options.

### Run

```bash
ktns <command> <flags>
```

</br>
</br>

# Commands

</br>

## `mock`

```bash
ktns mock <flags>
```

### How it works

The mock command makes use of `dynamic values` which will generate mocked data depending on the desired type of data, by calling specific `Mock functions`.

These dynamic values must be called wrapped in (double curly braces) `{{ }}`, like `{{ Person.name }}`, which will dynamically generate a person's name.

If not wrapped, the value will be interpreted as a literal string value.

```bash
ktns mock --parse-str 'Name: {{ Person.name }}'

# Will generate
# Name: John Smith
```

```bash
ktns mock --parse-str 'Name: Person.name'

# Will generate
# Name: Person.name
```

### Flags

- `--list`: To list all available mock functions.
- `--parse-str`: Pass a literal string to be parsed. The mock data will be generated based on the provided string.
- `--parse-json`: Pass a JSON object as a string. The mock data will be generated based on the provided object. Requires `--to-stdout` or `--to-json-file`.
- `--parse-json-file`: Pass a path to a single `.template.json` file (the filename **must** end with `.template.json`). The mock data will be generated based on this file. Requires `--to-stdout` or `--to-json-file`.
- `--generate`: Pass the desired amount of root objects that will be generated (available for `--parse-json` and `--parse-json-file`). (More info [here](#generating-multiple-values))
- `--to-stdout <as-json|as-csv>`: Output the result to stdout as a JSON object/array or as a CSV table. Must be used with `--parse-json` or `--parse-json-file`. Use `--to-stdout-prettify` to format for readability. Note: CSV output works best with flat (one-level-deep) JSON objects; nested objects and arrays are serialised using their Go string representation.
- `--to-stdout-prettify`: Prettify the stdout output (indented JSON or padded-column CSV). Only valid with `--to-stdout`.
- `--to-json-file [filename]`: Write the result as JSON to a file. If no filename is given, defaults to `output.json` beside the binary (for `--parse-json`) or to the template name without `.template` in the same directory (for `--parse-json-file`). If a filename is given using `--to-json-file=myfile.json`, it is used as-is. Can be combined with `--to-stdout`.
- `--to-csv-file [filename]`: Write the result as CSV to a file. Same filename-resolution rules as `--to-json-file` (default `output.csv` for `--parse-json`, template name for `--parse-json-file`). Can be combined with `--to-stdout` and `--to-json-file`. Note: CSV output works best with flat (one-level-deep) JSON objects.
- `--debug`: Print a live generation-progress line to stderr during large runs (workers, memory estimate, throughput, elapsed time). Writes to stderr only — stdout/file output is unaffected. Opt-in; default `false`.

### Examples

#### `--list`

Get a list of all the available `Mock functions`.

```bash
ktns mock --list
```

#### `--parse-str`

```bash
ktns mock --parse-str 'Hello my name is {{ Person.name }}, I am {{ Number.number::{1}:{100} }} years old.'

# Hello my name is John Smith, I am 40 years old.
```

#### `--parse-json`

Output to stdout as compact JSON:

```bash
ktns mock --parse-json '{ "company": "{{ Company.name }}", "employee": { "name": "{{ Person.name }}" }}' --to-stdout as-json

# {"company":"Delvalle","employee":{"name":"Josh Smith"}}
```

Output to stdout as prettified JSON:

```bash
ktns mock --parse-json '{ "company": "{{ Company.name }}" }' --to-stdout as-json --to-stdout-prettify

# {
#   "company": "Delvalle"
# }
```

Output to stdout as CSV:

```bash
ktns mock --parse-json '{ "name": "{{ Person.name }}", "age": "{{ Number.number::{18}:{80} }}" }' --to-stdout as-csv

# age,name
# 34,Josh Smith
```

Write to a file (default name `output.json` beside the binary):

```bash
ktns mock --parse-json '{ "company": "{{ Company.name }}" }' --to-json-file
```

Write to a specific file:

```bash
ktns mock --parse-json '{ "company": "{{ Company.name }}" }' --to-json-file=mydata.json
```

Both stdout and file simultaneously:

```bash
ktns mock --parse-json '{ "company": "{{ Company.name }}" }' --to-stdout as-json --to-json-file=mydata.json
```

#### `--parse-json-file`

The template file **must** end with `.template.json`.

```json
// employee.template.json
{
  "company": "{{ Company.name }}",
  "employee": {
    "name": "{{ Person.name }}",
    "age": "39"
  }
}
```

Write the default output file alongside the template (`employee.json`):

```bash
ktns mock --parse-json-file employee.template.json --to-json-file
```

```json
// employee.json  (written alongside the template file)
{
  "company": "Delvalle",
  "employee": {
    "name": "Josh Smith",
    "age": "39"
  }
}
```

Write to a specific file:

```bash
ktns mock --parse-json-file employee.template.json --to-json-file=myout.json
```

Output to stdout as CSV:

```bash
ktns mock --parse-json-file employee.template.json --to-stdout as-csv
```

With `--generate` to produce an array:

```bash
ktns mock --parse-json-file employee.template.json --generate 3 --to-json-file
```

```json
// employee.json
[
  { "company": "Delvalle", "employee": { "name": "Josh Smith", "age": "39" } },
  { "company": "Infomatics", "employee": { "name": "Jane Doe", "age": "39" } },
  { "company": "Braindance", "employee": { "name": "Sam Lee", "age": "39" } }
]
```

### More Details

#### Mock functions optional parameters

Some of the Mock functions accept additional parameters. Each value parameter must be wrapped in curly braces (`{value}`) and separated by a colon (`:`).

> e.g.: `{{ functionName:{arg1}:{arg2}:... }}`

```json
{
  "words": "Loreum.words:{5}"
}
```

When working with multiple parameters, you may leave them blank if not used. _(They will assume default values)_

```json
// e.g.: `Number.number` expects 3 parameters (<decimal>:<min>:<max>)
// In this case <decimal> is left blank, and will use its default value.
{
  "age": "Number.number::{18}:{50}"
}
```

#### Generating multiple values

##### Root objects

Use the flag `--generate <number>` with `--parse-json` or `--parse-json-file` to generate multiple root objects.

```bash
ktns mock --parse-json '{ "company": "{{ Company.name }}" }' --generate 10 --to-stdout as-json

# This will generate
[
  { "company": "Delvale" },
  { "company": "Infomatics" },
  { "company": "Braindance" },
  ...
]
```

```bash
ktns mock --parse-json-file employees.template.json --generate 5 --to-json-file
# Writes employees.json with an array of 5 objects in the same directory as the template file.
```

##### Inner objects

For inner objects, also pass the desired number between brackets in the object's `key`.

```json
{
  "phones[3]": "{{ Person.phoneNumber }}",  // Will generate an array of 3 values
  "employees[2]": {                         // Will generate an array of employees with 2 objects
    "name": "{{ Person.name }}"
  }
}
```

Will produce:

```json
{
  "phones": ["...", "...", "..."],
  "employees": [
    {
      "name": "..."
    },
    {
      "name": "..."
    }
  ]
}
```

#### Limitations of the json values

The `value` of a json object key may be:

1. A literal raw value.

```json
{
  "name": "Some literal value"
}
```

2. A `dynamic value` value with the **Faker function name *(between double curly braces)***. 

```json
{
  "name": "{{ Person.name }}"
}
```

3. An `object`, detailing an inner object.

```json
{
  "employee": {
    ...
  }
}
```

4. An `array` of either `strings`/`dynamic values` OR `objects`.

```json
{
  "names": ["Some name 1", "{{ Person.name }}"]
}

// Or

{
  "employees": [
    { ... },
    { ... }
  ]
}
```

> Also create dynamic array values as detailed [here](#inner-objects).


</br>
</br>

## `request`

```bash
ktns request <flags>
```

### How it works

Make use of `Mock functions` inside `--data`, `--qs` and `--url`, to mock dynamic values for the request body, query string and url params.

### Flags

- `--method`: Define the request **method** (`GET` | `POST` | `PUT` | `PATCH` | `DELETE`).
- `--url`: The request **URL** with added **URL params**. _(e.g. `http://localhost:3000`, `localhost:8000/api/users`, `api.com/user/{{UUID.uuidv4}}`)_
- `--https`: A Flag to overwrite the `url` protocol to HTTPS. _(If not set, it will use whichever the `url` protocol is)_
- `--header`: A `string` `<key>: <value>` pair to be used as header in the request. *(Multiple flags can be used)*
- `--data`: A `Json string format` defining data to be used in the request body.
- `--qs`: A `string` `<key>=<value>` pair to be used as query string in the request. *(Multiple flags can be used)*
- `--response-accessor`: A `string` value to specify how the response should be accessed, with the idea of returning a more specific segment of the response. _(If unable to access, it returns the whole response)_
- `--with-metrics`: A Flag to add metrics of the request in the response.
- `--only-response-body`: A Flag to force the return to only show the response's body.

### Examples

#### Simple examples

```bash
ktns request --method GET --url http://localhost:3000/endpoint
```

```bash
ktns request --method GET --url https://some-api.com/endpoint
```

```bash
# The same as the previous, but forcing HTTPS with the flag, instead of informing it in the URL.
# If the flag was not used, HTTP protocol would be used by default
ktns request --method GET --https --url some-api.com/endpoint
```

```bash
ktns request --method GET --url http://localhost:3000/endpoint -qs 'q=Josh'
```

```bash
ktns request --method POST --url https://some-api.com/endpoint --data '{ "name": "John Smith" }'
```

```bash
ktns request --method DELETE --url https://some-api.com/endpoint/20313189-596e-4a65-a706-45a1531ea317
```

#### Mocking data

Query Strings

```bash
ktns request
  --medhod GET
  --url https://some-api.com/person
  --qs 'ageMin={{ Number.number:{0}:{1}:{10} }}'
  --qs 'ageMax={{ Number.number:{0}:{50}:{55} }}'
```

Request Body

```bash
ktns request
  --medhod POST
  --url https://some-api.com/endpoint
  --data '{ "name": "{{ Person.name }}", "phones[3]": "{{ Person.phoneNumber }}" }'
```

URL Params

```bash
ktns request --method DELETE --url https://some-api.com/endpoint/{{ UUID.uuidv4 }}
```

#### Adding authorization header

```bash
ktns request
  --medhod GET
  --url https://some-api/person
  --header "Authorization: Bearer fb80ea0..."
```

### Response

A standard response will be like:

```bash
Status: 200 OK
URL: https://some-api/objects
Headers:
  Content-Length: 2731
  Access-Control-Allow-Origin: *
  Content-Type: application/json; charset=utf-8
  Server: Some
  Via: 2.0 some-router
  X-Ratelimit-Remaining: 48
  Date: Mon, 02 May 2024 13:17:41 GMT
  ...
Body:
{ ... }
```

#### Adding metrics with `--with-metrics`

Would include calculated metrics right bellow `Status`.

```bash
Status: 200 OK
Metrics:
  Duration: [968.00ms] 
  Size: [2.67 KB]
URL: https://some-api/objects
Headers:
  ...
Body:
{ ... }
```

#### Get only body result `--only-response-body`

Would result in only the response body to be shown.

```bash
{ ... }
```

#### Get more specific body response `--response-accessor <accessor>`

> To be done.

</br>
</br>

# Development Details

## Specifications

### Main Dependencies

- [`Cobra`](github.com/spf13/cobra): A commandder for modern Go CLI interations.
- [`Faker/v2`](github.com/jaswdr/faker/v2): Fake data generator for Go.
- [`Gogoren`](github.com/zach-klippenstein/goregen): Randexp for Go.
- [`Deepcopy`](github.com/mohae/deepcopy): Deepcopy things.
- [`Testify`](github.com/stretchr/testify): Toolkit with common assertions and mocks that plays nicely with the standard library.

### Project Packages

#### `mocker` Package

##### Purpose

Provides the fake data generation engine used by the `mock` and `request` commands. It abstracts all data generation behind a single interface, keeping CLI logic decoupled from generation logic.

##### Constructor

```Golang
func New() *Mock
```

Creates a `Mock` instance wrapping an initialized `jaswdr/faker` instance. Each instance gets its own `*rand.Rand` source seeded from the current time, PID, and an atomic counter — guaranteeing unique seeds even when many goroutines call `New()` simultaneously. Instances are **not** goroutine-safe; each goroutine must create its own via `mocker.New()`.

##### Parameter Convention

`functionParams` is always `[]string`, even for numeric parameters. Each function is responsible for parsing and applying defaults.

Blank entries (`""`) mean "use default", enabling positional omission (e.g. `Number.number::18:50` in `Number.number:<decimal>:<min>:<max>` — decimals left blank).

##### Dependencies

| Dependency | Used for |
| -- | -- |
| `github.com/jaswdr/faker/v2` | ~90% of functions (addresses, persons, companies, etc.) |
| `github.com/zach-klippenstein/goregen` | `Regex.regex` and `Payment.creditCardCvv` (via per-instance generator) |
| `math/rand` | Per-instance `*rand.Rand` in `Mock` — used by `Company.cnpj`, `Person.cpf`, and as the RNG source for goregen generators |

##### Custom Implementations

- `Person.cpf` and `Company.cnpj`: Generate random digit sequences using the instance's own `*rand.Rand` and compute two mathematically valid checksum digits via the modulo-11 algorithm (`calculateChecksum` in `helpers.go`).
- `Regex.regex`: Accepts a regex pattern wrapped in `/…/`, strips the delimiters (and unescapes `\/` → `/`), then calls `regen.NewGenerator` with the instance's `*rand.Rand` as the RNG source to produce a matching random string.
- `Payment.creditCardCvv`: Uses a `regen.Generator` (created at `mocker.New()` time, stored on the instance) backed by the instance's `*rand.Rand` to produce a 3-digit string — no global random source is touched.

##### Adding a New Mock Function

1. Add the display entry in `List()` with `tableLineData`.
2. Add a `case` in the `Generate` switch, parsing `functionParams` as needed.
3. If it requires a utility (e.g. checksum logic), add it to `helpers.go`.
4. Add test coverage in `helpers_test.go`.

#### `cmd` Package

##### Purpose

Houses all Cobra command definitions, flag parsing, input validation, and orchestration logic for every CLI subcommand. It bridges user input to the `mocker` package and the standard library's `net/http`.

##### Entry Points

```Golang
// called from main.go — uses default options (os.Stdout)
func Execute()

// testable constructor
func NewRootCmd(opts *CommandOptions) *cobra.Command
```

`CommandOptions` carries a single `Out io.Writer`, which lets tests redirect output without touching `os.Stdout`. Both `mock` and `request` subcommands receive and honor this same `opts`.

##### Version Injection

`Version` in `version.go` is an empty `var` set at build time via `-ldflags`:

```Golang
go build -ldflags "-X github.com/lfsc09/k-test-n-stress/cmd.Version=x.y.z"
```

##### Subcommand Files

| File | Subcommand | Responsability |
| -- | -- | -- |
| `root.go` | root | Wires subcommands, sets version, silences usage on error |
| `mock.go` | `mock` | Flag validation, parse modes, streaming file I/O, bounded worker-pool concurrency for large generation counts |
| `request.go` | `request` | HTTP request construction, mock injection into URL/QS/body, response formatting |
| `utils.go` | — | Shared `CommandOptions`, duration/size formatters |
| `version.go` | — | Build-time version variable |

##### Testing Structure

| Filename Sulfix | What should test |
| -- | -- |
| `_test.go` | Unit tests for internal helpers, e.g. `extractMockMethod`, `interpretString`, `processJsonMap`, etc. Uses `testify/suite` |
| `_e2e_test.go` | End-to-end CLI tests via `NewRootCmd` with a captured `bytes.Buffer` as `Out`. Validates full flag combinations and output |

##### Adding a New Subcommand

1. Create `cmd/<name>.go` with a `NewXxxCmd(opts *CommandOptions) *cobra.Command` constructor.
2. Register it in `NewRootCmd` with `rootCmd.AddCommand(NewXxxCmd(opts))`.
3. Set `cmd.SetOut(opts.Out)` inside the constructor so output is testable.
4. Add E2E tests in `cmd/<name>_e2e_test.go` using the `executeCommand` helper pattern from `mock_e2e_test.go`.

</br>

## Installation

### 1. Clone the repository

```bash
git clone git@github.com:lfsc09/k-test-n-stress.git
cd k-test-n-stress
```

### 2. Install dependencies

```bash
go mod download
```

### 3. Configure git hooks

For auto-bump version on commits.

```bash
make install-hooks
```

</br>

## Running

### Execute app

```bash
go run . <command> <flags>
```

### Run tests

```bash
go test ./...
```

Run with the race detector (required for all concurrent code):

```bash
go test -race ./...
# or via Makefile
make test-race
```

</br>

## Maintaining

### Updating dependencies

```bash
# Download updates for all dependencies to their latest minor/patch versions
go get -u ./...

# Tidy: remove unused deps and add any missing ones
go mod tidy
```

</br>

## LLM Development

This project uses a three-agent system for LLM-assisted development. Plan files live in `.claude/plans/` and serve as the documentation trail for every feature, bug fix, and refactor. Audit reports live in `.claude/reports/`.

### Agents

| Agent | File | Responsibility |
| -- | -- | -- |
| `planner` | `.claude/agents/planner.md` | Reads the codebase and writes a detailed step-by-step plan to `.claude/plans/`. Does **not** write code. |
| `developer` | `.claude/agents/developer.md` | Reads a plan file, implements every step, marks each step done, and renames the file with a `_[done]` suffix when finished. |
| `auditor` | `.claude/agents/auditor.md` | Independently audits the full codebase for correctness, security, architecture, performance, tests, and documentation. Writes a dated report to `.claude/reports/`. Does **not** modify any files. |

### Pipeline

```
1. You describe the feature, bug, or refactor to the planner
          ↓
2. planner captures the current datetime, reads the codebase,
   and writes .claude/plans/YYYYMMDDhhmm_<type>_<name>.md
          ↓
3. You review the plan (request changes if needed)
          ↓
4. You ask the developer to implement the plan file
          ↓
5. developer implements step by step, checks off each step,
   renames file to YYYYMMDDhhmm_<type>_<name>_[done].md when complete
```

### Plan file naming

All plan files are prefixed with the datetime at creation time (`YYYYMMDDhhmm`):

- New features: `YYYYMMDDhhmm_feature_<short_name>.md` → `YYYYMMDDhhmm_feature_<short_name>_[done].md`
- Bug fixes: `YYYYMMDDhhmm_bug_<short_name>.md` → `YYYYMMDDhhmm_bug_<short_name>_[done].md`
- Refactors: `YYYYMMDDhhmm_refactor_<short_name>.md` → `YYYYMMDDhhmm_refactor_<short_name>_[done].md`

### Audit pipeline

The auditor runs independently of the planner/developer cycle. Invoke it at any point to get an objective snapshot of the project's health. Audit reports are written to `.claude/reports/` and are intended for human review — each finding can then be handed to the planner as a specific task.

```
1. You ask the auditor to audit the project
          ↓
2. auditor reads .claude/memories.md and all source files,
   then writes .claude/reports/YYYYMMDDhhmm_audit.md
          ↓
3. You review the report and decide which findings to act on
          ↓
4. You describe each chosen finding to the planner as a task
          ↓
5. planner → developer → done  (normal pipeline)
```

### Audit report naming

Audit reports are prefixed with the datetime at creation time:

- `YYYYMMDDhhmm_audit.md`

### Examples

#### Adding a new mock function

```
"Plan the addition of an Internet.email mock function"
→ planner writes .claude/plans/202604091530_feature_internet_email.md

"Implement .claude/plans/202604091530_feature_internet_email.md"
→ developer implements and renames to 202604091530_feature_internet_email_[done].md
```

#### Fixing a bug

```
"Plan a fix for the bug where --parse-files ignores --preserve-folder-structure"
→ planner writes .claude/plans/202604091530_bug_preserve_folder_structure.md

"Implement .claude/plans/202604091530_bug_preserve_folder_structure.md"
→ developer implements and renames to 202604091530_bug_preserve_folder_structure_[done].md
```

#### Refactoring code

```
"Plan a refactor of mocker/helpers.go to split it into focused files"
→ planner writes .claude/plans/202604091530_refactor_mocker_helpers_split.md

"Implement .claude/plans/202604091530_refactor_mocker_helpers_split.md"
→ developer implements and renames to 202604091530_refactor_mocker_helpers_split_[done].md
```

#### Requesting plan changes before implementation

```
"Plan the addition of an Internet.email mock function"
→ planner writes .claude/plans/202604091530_feature_internet_email.md

"Update the plan to also include Internet.url and Internet.ipv4"
→ planner updates the same plan file

"Implement .claude/plans/202604091530_feature_internet_email.md"
→ developer implements the updated plan
```

#### Auditing the codebase

```
"Audit the project"
→ auditor reads all source files and .claude/memories.md,
  writes .claude/reports/202604131200_audit.md
```

```
"Run an audit and generate a report"
→ same as above — a new dated report is always created
```

After reviewing the report:

```
// C1 finding in the report: path traversal in --to-json-file
"Plan a fix for the path traversal vulnerability in --to-json-file"
→ planner writes .claude/plans/202604131210_bug_path_traversal_json_file.md

"Implement .claude/plans/202604131210_bug_path_traversal_json_file.md"
→ developer implements and renames to …_[done].md
```
