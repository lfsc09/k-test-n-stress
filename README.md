![Go Badge](https://img.shields.io/badge/Go-1.26-00ADD8.svg?style=for-the-badge&logo=Go&logoColor=white)

# The project

K-test-n-stress is a simple Go cli tool to:

1. [`[mock]`](#mock) Generate random, structurally valid fake data on demand.
2. [`[request]`](#request) Generate http requests to URLs.
3. `[stress]` Stress test URLs.

### Basic usage

```bash
ktns <command> <flags>
```

### Advantages

### Disadvantages

</br>

# Commands

## `mock`

```bash
ktns mock <flags>
```

### How it works

A template, which is a Json object, is given as input to be parsed to generate either JSON or CSV output. The template's object values are compiled into `literal` and `dynamic` blocks that will be processed to generate the mock data.

`dynamic blocks` is where you specify the `mock functions` that will generate mocked data depending on the desired type of data. These dynamic blocks must be wrapped in (double curly braces) `{{ }}` (`{{ Person.Name }}`, which will dynamically generate a person's name).

Anything outside `{{ }}` is treated as literal values and is just reflected to the output as is.

```bash
ktns mock --parse-str 'Name: {{ Person.Name }}'

# Name: John Smith
```

```bash
ktns mock --parse-str 'Name: Person.Name'

# Name: Person.Name
```

### Flags

- `--list`: To list all available mock functions.
- `--parse-str`: A string input to be parsed.
- `--parse-json`: A JSON template object as a string to be parsed.
- `--parse-json-file`: A path to a single `.template.json` JSON template file (the filename **must** end with `.template.json`) to be parsed.
- `--parse-csv`: A CSV template as a string to be parsed.
- `--parse-csv-file`: A path to a single `.template.csv` CSV template file (the filename **must** end with `.template.csv`) to be parsed.
- `--generate`: The desired amount of root objects to be generated (**not** available for `--parse-str`). (More info [here](#generating-multiple-values))
- `--to-stdout`: To output the result to STDOUT.
- `--to-file <filename|"">`: To output the result to a file. If no filename is given:
  - Defaults to `output.<json|csv>` in the current working directory (for `--parse-json` or `--parse-csv`);
  - Or to the template name without `.template` in the same directory as the template file (for `--parse-json-file` or `--parse-csv-file`);
- `--debug`: To output debug information to STDERR (elapsed time, progress, memory consumption, output size).

> Either (`--to-stdout` and/or `--to-file`) **must** be provided if (`--parse-json` or `--parse-json-file` or `--parse-csv` or `--parse-csv-file`) is used.

### Input Template

#### JSON example

```json
{
  // A literal value
  "key1": "literal value",
  // A mock|pipe function (dynamic block)
  "key2": "{{ Lorem.Word }}",
  // A fixed Array of (literal value, mock|pipe function (dynamic block), object)
  "key3": ["literal value", "{{ Lorem.Word }}", { "s1key1": "literal value", ... }],
  // An object
  "key4": {
    "s2key1": "literal word",
    ...
  },
  // An Array Generation of literal values
  "key5[2]": "literal value",
  // An Array Generation of mock|pipe function (dynamic block) values
  "key6[2]": "{{ Lorem.Word }}",
  // An Array Generation of object
  "key7[2]": {
    "s3key1": "literal word",
    ...
  }
}
```

#### CSV example

```json
{
  "col1": "literal value",
  "col2": "{{ Lorem.Word }}"
}
```

### Command details

#### `--list`

Get a list of all the available `mock functions`.

```bash
ktns mock --list
```

#### `--parse-str`

```bash
ktns mock --parse-str 'Hello my name is {{ Person.Name }}, I am {{ Number.IntBetween:{1}:{100} }} years old.'

# Hello my name is John Smith, I am 40 years old.
```

#### `--parse-json`

Output examples:

To **STDOUT**

```bash
ktns mock --parse-json '{ "company": "{{ Company.Name }}", "employee": { "name": "{{ Person.Name }}" }}' --to-stdout

# { "company": "Delvalle", "employee": { "name": "Josh Smith" }}
```

To **JSON file**

```bash
ktns mock --parse-json '{ "company": "{{ Company.Name }}", "employee": { "name": "{{ Person.Name }}" }}' --to-file ""
```

```json
// output.json
{ "company": "Delvalle", "employee": { "name": "Josh Smith" }}
```

> If no output filename was given the program default to `output.json` in the current working directory.

To a specific **JSON file**

```bash
ktns mock --parse-json '{ "company": "{{ Company.Name }}" }' --to-file path/to/mydata.json
```

```json
// path/to/mydata.json
{ "company": "Delvalle", "employee": { "name": "Josh Smith" }}
```

#### `--parse-json-file`

> The template file **must** end as `.template.json`.

```json
// employee.template.json
{
  "company": "{{ Company.Name }}",
  "employee": {
    "name": "{{ Person.Name }}",
    "age": "39"
  }
}
```

Output examples:

To **STDOUT**

```bash
ktns mock --parse-json-file employee.template.json --to-stdout

# { "company": "Delvalle", "employee": { "name": "Josh Smith", "age": 39 }}
```

To **JSON file**

```bash
ktns mock --parse-json-file employee.template.json --to-file ""
```

```json
// employee.json  (created alongside the template file)
{ "company": "Delvalle", "employee": { "name": "Josh Smith", "age": "39" }}
```

> If no output filename was given the program will use the template's filename (`employee.json`) in the same directory.

To a specific **JSON file**

```bash
ktns mock --parse-json-file employee.template.json --to-file path/to/myout.json
```

```json
// path/to/myout.json
{ "company": "Delvalle", "employee": { "name": "Josh Smith", "age": "39" }}
```

#### `--generate`

```bash
ktns mock --parse-json-file employee.template.json --generate 3 --to-stdout
```

```json
[
  { "company": "Delvalle", "employee": { "name": "Josh Smith", "age": "39" }},
  { "company": "Infomatics", "employee": { "name": "Jane Doe", "age": "39" }},
  { "company": "Braindance", "employee": { "name": "Sam Lee", "age": "39" }}
]
```

#### `--debug`

```bash
ktns mock --parse-json-file employee.template.json --generate 100000 --to-file "" --debug

# [debug] Elapsed: 38.70s | Progress: 22Mi / 22Mi (100.0%) | Mem: 6.91MB ⌈7.70MB⌉ [27.21MB] | Output Size: ~484.79MB
```

Where:

- `Elapsed`: Is the total time elapsed since the start.
- `Progress`: `v1` / `v2` (`perc`)
  - `v1`: Is the current count of generated elements. (Including literal values)
  - `v2`: Is the total count of elements to be generated. (dynamic + literal)
  - `perc`: Is the percentage done.
- `Mem`: `v1` ⌈`v2`⌉ [`v3`]
  - `v1`: Is the current allocated heap size from `runtime.MemStat.Alloc`.
  - `v2`: Is the highest value of `runtime.MemStat.Alloc` reached.
  - `v3`: Is the system reserved amount from `runtime.MemStat.System`.
- `Output Size`: Is the total amount of bytes written to the buffer. (Approximate size of the result object)

### More Details

#### Function types

There are two types of functions `mock functions` and `pipe functions` that can be used in a dynamic block.

`mock functions` are the ones that generate the mocked data. They are the most commonly used and are the ones listed with `--list`. Currently they will be in the format of `Category.Function` (2 parts divided by a dot), where each part has the first letter capitalized.

`pipe functions` are the ones that process or transform the output of other functions. They are used in combination with mock functions to modify or enhance the generated data. Currently they will be in the format of `FUNCTION_NAME` (all uppercase letters).

#### Piping functions

In a single `dynamic block` you can pipe functions with `|` to either:

- Overwrite the output of a function with another;
- Pipe the output of a function as an input parameter to another;

Usually the first function is used to generate a value, and the second function will use that value as input to process it in some way, or to overwrite it with a new value.

```json
// In this case the OR_BLANK function will overwrite the person's name generated
{ "name": "{{ Person.Name | OR_BLANK }}" }
```

```json
// In this case a person's name is generated, used as input to CACHE_WRITE function and then returned as output of the whole block
{ "name": "{{ Person.Name | CACHE_WRITE:{key} }}" }
```

It is possible though to pipe multiple mock functions together, but only the last one's value will be returned as output of the whole block.

```json
{ "name": "{{ Person.Name | Person.Phone }}" }
```

#### Mock|Pipe functions parameters

Some of the mock functions accept additional parameters. Each value parameter must be wrapped in curly braces (`{value}`) and separated by a colon (`:`).

```txt
{{ functionName:{arg1}:{arg2}:{argN} }}
```

```json
{ "words": "Loreum.Sentence:{5}" }
```

It is possible to use blank parameters (without value) to assume the **default** value, by leaving blank after the colon `fn:` or a curly brace with nothing inside `fn:{}`.

When working with multiple parameters, you may leave them blank if not used. _(They will assume default values)_

```json
// e.g.: `Number.FloatBetween` expects 3 parameters (<decimal>:<min>:<max>)
// In this case <decimal> is left blank, and will use its default value.
{ "age": "Number.FloatBetween::{18}:{50}" }

// Or
{ "age": "Number.FloatBetween:{}:{18}:{50}" }
```

#### Generating multiple values

##### Root objects

Use the flag `--generate <number>` to generate:

- Multiple **root objects** when working with JSON;
- Multiple **lines** when working with CSV;

```bash
ktns mock --parse-json '{ "company": "{{ Company.name }}" }' --generate 5 --to-stdout
```

```json
[
  { "company": "Delvale" },
  { "company": "Infomatics" },
  { "company": "Braindance" },
  { "company": "Colleative" },
  { "company": "Jimbo" },
]
```

```bash
ktns mock --parse-csv 'Company:::"{{ Company.Name }}"' --generate 5 --to-stdout
```

```csv
Company
Delvale
Infomatics
Braindance
Colleative
Jimbo
```

##### Inner keys

When working with JSON, to generate arrays of data, pass the desired number between brackets in the object's `key`.

```json
{
  "phones[3]": "{{ Person.Phone }}",  // Will generate an array of 3 values
  "employees[2]": {                   // Will generate an array of employees with 2 objects
    "name": "{{ Person.Name }}"
  }
}
```

```json
{
  "phones": ["...", "...", "..."],
  "employees": [
    { "name": "..." },
    { "name": "..." }
  ]
}
```

#### Limitations of the json values

As described in the JSON template example [here](#json-example), the `value` of a json object key may either:

A `literal block`.

```json
{ "name": "Some literal value" }
```

A `dynamic block` with the **mock function name *(between double curly braces)***. 

```json
{ "name": "{{ Person.Name }}" }
```

An `object`, detailing an inner object.

```json
{ "employee": { ... }}
```

A fixed `array` of (`literal block`, `dynamic block`,  `object`).

```json
{ "names": ["Some name 1", "{{ Person.Name }}", { ... }]}
```

</br>

## `request`

```bash
ktns request <flags>
```

### How it works

Make use of `mock functions` inside `--data` (request body), `--qs` (query strings) and `--url` (url params).

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

# Development Details

### Installation

Clone the repository.

```bash
git clone git@github.com:lfsc09/k-test-n-stress.git
cd k-test-n-stress
```

Install dependencies.

```bash
go mod download
```

Configure git hooks for auto-bump version on commits.

```bash
make install-hooks
```

### Running

Execute the app in terminal.

```bash
go run . <command> <flags>
```

Run tests:

```bash
go test ./...
```

Run with the race detector (for concurrent code):

```bash
go test -race ./...
```

</br>

## LLM Development (Beta)

***Still having mediocre results in code generation and plan creation.***

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
// C1 finding in the report: path traversal in --to-file
"Plan a fix for the path traversal vulnerability in --to-file"
→ planner writes .claude/plans/202604131210_bug_path_traversal_json_file.md

"Implement .claude/plans/202604131210_bug_path_traversal_json_file.md"
→ developer implements and renames to …_[done].md
```
