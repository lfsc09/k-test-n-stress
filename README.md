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
- `--parse-json`: Pass a JSON object as a string. The mock data will be generated based on the provided object.
- `--parse-files`: Pass a path, directory, or glob pattern to find template files (`.template.json`). The mock data will be generated based on the found files.
- `--preserve-folder-structure`: If set, the folder structure of the input files will be preserved in the output files, otherwise output files will be flattened. (More info [here](#preservation-of-folder-structure))
- `--generate`: Pass the desired amount of root objects that will be generated (only available for `--parse-json`). (More info [here](#generating-multiple-values))

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

```bash
ktns mock --parse-json '{ "company": "{{ Company.name }}", "employee": { "name": "{{ Person.name }}" }'

# { "company": "Delvalle", "employee": { "name": "Josh Smith" } }
```

#### `--parse-files`

```json
// example.template.json
{
  "company": "{{ Company.name }}",
  "employee": {
    "name": "{{ Person.name }}",
    "age": "39"
  }
}
```

```bash
  ktns mock --parse-files example.template.json
  # Or
  ktns mock --parse-files "*.template.json"
  # Or
  ktns mock --parse-files "test/templates/*.template.json"
```

```json
// out/example.template.json
{
  "company": "Delvalle",
  "employee": {
    "name": "Josh Smith",
    "age": "39"
  }
}
```

#### `--preserve-folder-structure`

```json
// test/templates/example.template.json
{
  "company": "{{ Company.name }}",
  "employee": {
    "name": "{{ Person.name }}",
    "age": "39"
  }
}
```

```bash
  ktns mock --parse-files "test/templates" --preserve-folder-structure
```

```json
// out/test/templates/example.template.json
{
  "company": "Delvalle",
  "employee": {
    "name": "Josh Smith",
    "age": "39"
  }
}
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

If parsing a json object from the command line with `--parse-json`, use the flag `--generate <number>` to generate multiple root objects.

```bash
ktns mock --parse-json '{ "company": "{{ Company.name }}" }' --generate 10

# This will generate
[
  { "company": "Delvale" },
  { "company": "Infomatics" },
  { "company": "Braindance" },
  ...
]
```

When using `--parse-files`, **specify the desired number of root objects in the template file's name**, between brackets.

e.g.: A template file named `employees[5].template.json` bellow:

```json
{
  "name": "{{ Person.name }}"
}
```

Will produce an output file `employees[5].json` with:

```json
[
  {
    "name": "..."
  },
  {
    "name": "..."
  },
  {
    "name": "..."
  },
  {
    "name": "..."
  },
  {
    "name": "..."
  },
]
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

#### Preservation of folder structure

When using `--parse-files`, you can may have a folder structure, for instance, like this:

```
├── company.template.json
└── assets/
  ├── employee[10].template.json
  └── building[2].template.json
```

If you wish to generate the fake data and preserve this structure, use the flag `--preserve-folder-structure` to have a result like:

```
└── out/
  ├── company.json
  └── assets/
    ├── employee[10].json
    └── building[2].json
```

Otherwise your result files will be flatten:

```
└── out/
  ├── company.json
  ├── employee[10].json
  └── building[2].json
```

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
- [`Mpb`](https://github.com/vbauerster/mpb): Multi progress bar for Go CLI applications.
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

Creates a Mock instance wrapping an initialized `jaswdr/faker` instance. There's no configuration — one instance per CLI invocation.

##### Parameter Convention

`functionParams` is always `[]string`, even for numeric parameters. Each function is responsible for parsing and applying defaults.

Blank entries (`""`) mean "use default", enabling positional omission (e.g. `Number.number::18:50` in `Number.number:<decimal>:<min>:<max>` — decimals left blank).

##### Dependencies

| Dependency | Used for |
| -- | -- |
| `github.com/jaswdr/faker/v2` | ~90% of functions (addresses, persons, companies, etc.) |
| `github.com/zach-klippenstein/goregen` | `Regex.regex` and `Payment.creditCardCvv` |
| `math/rand` | Custom `Company.cnpj` and `Person.cpf` digit generation |

##### Custom Implementations

- `Person.cpf` and `Company.cnpj`: Generate random digit sequences and compute two mathematically valid checksum digits via the modulo-11 algorithm (`calculateChecksum` in `helpers.go`).
- `Regex.regex`: Accepts a regex pattern wrapped in `/…/`, strips the delimiters (and unescapes `\/` → `/`), then passes it to `goregen` to produce a matching random string.
- `Payment.creditCardCvv`: Uses `goregen` with [0-9]{3} instead of faker.

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
| `mock.go` | `mock` | Flag validation, parse modes, file I/O, concurrency for `--parse-files` |
| `request.go` | `request` | HTTP request construction, mock injection into URL/QS/body, response formatting |
| `utils.go` | — | Shared `CommandOptions`, duration/size formatters |
| `version.go` | — | Build-time version variable |

##### Concurrency in `--parse-files`

Each template file is processed in its own goroutine. A `sync.WaitGroup` coordinates completion. A `sync.Mutex` guards the shared `createdDirs` map used to avoid duplicate `os.MkdirAll` calls when writing output files. A `mocker.New()` instance is created per goroutine (not shared), so no locking is needed for generation.

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

This project uses a two-agent pipeline for LLM-assisted development. Plan files live in `.claude/plans/` and serve as the documentation trail for every feature and bug fix.

### Agents

| Agent | File | Responsibility |
| -- | -- | -- |
| `planner` | `.claude/agents/planner.md` | Reads the codebase and writes a detailed step-by-step plan to `.claude/plans/`. Does **not** write code. |
| `developer` | `.claude/agents/developer.md` | Reads a plan file, implements every step, marks each step done, and renames the file with a `_done` suffix when finished. |

### Pipeline

```
1. You describe the feature or bug to the planner
          ↓
2. planner reads the codebase and writes .claude/plans/<name>.md
          ↓
3. You review the plan (request changes if needed)
          ↓
4. You ask the developer to implement the plan file
          ↓
5. developer implements step by step, checks off each step,
   renames file to <name>_done.md when complete
```

### Plan file naming

- New features: `feature_<short_name>.md` → `feature_<short_name>_done.md`
- Bug fixes: `bug_<short_name>.md` → `bug_<short_name>_done.md`

### Examples

#### Adding a new mock function

```
"Plan the addition of an Internet.email mock function"
→ planner writes .claude/plans/feature_internet_email.md

"Implement .claude/plans/feature_internet_email.md"
→ developer implements and renames to feature_internet_email_done.md
```

#### Fixing a bug

```
"Plan a fix for the bug where --parse-files ignores --preserve-folder-structure"
→ planner writes .claude/plans/bug_preserve_folder_structure.md

"Implement .claude/plans/bug_preserve_folder_structure.md"
→ developer implements and renames to bug_preserve_folder_structure_done.md
```

#### Requesting plan changes before implementation

```
"Plan the addition of an Internet.email mock function"
→ planner writes .claude/plans/feature_internet_email.md

"Update the plan to also include Internet.url and Internet.ipv4"
→ planner updates the same plan file

"Implement .claude/plans/feature_internet_email.md"
→ developer implements the updated plan
```
