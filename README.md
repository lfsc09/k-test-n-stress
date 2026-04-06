![Go Badge](https://img.shields.io/badge/Go-1.24.1-00ADD8.svg?style=for-the-badge&logo=Go&logoColor=white)

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
ktns mock --parse-str 'Hello my name is {{ Person.name }}, I am {{ Number.number::1:100 }} years old.'

# Hello my name is John Smith, I am 40 years old.
```

#### `--parse-json`

```bash
ktns mock --parse-json '{ "company": "{{ Company.name }}", "employee": { "name": "{{ Person.fullName }}" }'

# { "company": "Delvalle", "employee": { "name": "Josh Smith" } }
```

#### `--parse-files`

```json
// example.template.json
{
  "company": "{{ Company.name }}",
  "employee": {
    "name": "{{ Person.fullName }}",
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
    "name": "{{ Person.fullName }}",
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

Some of the Mock functions accept additional parameters, and they are informed by delimiting with (colon) `:`.

> e.g.: `{{ functionName::arg1:arg2:... }}`

```json
{
  "words": "Loreum.words:5"
}
```

When working with multiple parameters, you may leave them blank if not used. _(They will assume default values)_

```json
// e.g.: `Number.number` expects 3 parameters (<decimal>:<min>:<max>)
// In this case <decimal> is left blank, and will use its default value.
{
  "age": "Number.number::18:50"
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
  --qs 'ageMin={{ Number.number:0:1:10 }}'
  --qs 'ageMax={{ Number.number:0:50:55 }}'
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

The project uses `Cobra` to read and parse the CLI flags and values.

The `Cobra` main commands (`mock`, `request` and `stress`) are in `/cmd` along with its tests.

Each main command file, has the functions to parse, interpret and run the sub-flag of each command.

The `/mocker` folder holds the package of the mocker object that currently only uses [`github.com/jaswdr/faker/v2`](https://github.com/jaswdr/faker) for most of the mock functions. Additional `Mock functions` were added manually.

### Execute app

```bash
go run . <command> <flags>
```

### Run tests

```bash
go test ./...
```
