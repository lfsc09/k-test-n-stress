![Go Badge](https://img.shields.io/badge/Go-1.24.1-00ADD8.svg?style=for-the-badge&logo=Go&logoColor=white)

# The project

k-test-n-stress is a simple tool to facilate:

1. `[mock]` Generation of fake data.
2. `[request]` Running http tests on endpoints.
3. `[stress]` Generating stress tests on endpoints.
4. `[feed]` Feed (Seed) databases with fake data and `table` templates. ???

Run it like:

All in the command line with flags.

```bash
ktns <command> <flags>
```

</br>
</br>

## Faker (`mock`) command

```bash
ktns mock <flags>
```

Mock function must be wrapped in `{{ Person.name }}`, or values passed will be interpreted as raw values.

### Flags

- `--list`: If set, it will list all available mock functions.
- `--parse-str`: Pass a string to be parsed. The mock data will be generated based on the provided string.
- `--parse-json`: Pass a JSON object as a string. The mock data will be generated based on the provided object.
- `--parse-files`: Pass a path, directory, or glob pattern to find template files (`.template.json`). The mock data will be generated based on the found files.
- `--preserve-folder-structure`: If set, the folder structure of the input files will be preserved in the output files.
- `--generate`: Pass the desired amount of root objects that will be generated (only available for `--parse-json`). (More info [here](#generating-multiple-values))

</br>

### Examples

#### Example (`--parse-str`)

```bash
ktns mock --parse-str 'Hello my name is {{ Person.name }}, I am {{ Number.number::1:100 }} years old.'
```

#### Example (`--parse-json`)

```bash
ktns mock --parse-json '{ "company": "{{ Company.name }}", "employee": { "name": "{{ Person.fullName }}" }}'
```

#### Example (`--parse-files`)

```json
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
  ktns mock --parse-files "*.template.json"
  ktns mock --parse-files "test/templates/*.template.json"
  ktns mock --parse-files "test/templates" --preserve-folder-structure
```

</br>

### Details

#### Limitations of the template objects

The `value` of an object key may be:
- A `string` value with the **Faker function name *(between double brackets)***.
- An `object`, detailing an inner object.
- An `array` of either `string` OR `object`.

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

#### Mock functions optional parameters

Some of the mock functions accept additional parameters, and they are informed by delimiting with `:`.

e.g.: `{{ functionName::arg1:arg2:... }}`

```json
{
  "words": "Loreum.words:5"
}
```

When working with multiple parameters, you may leave them blank if not used. _(They will assume default values)_

```json
// Number.number expects 3 parameters (<decimal>:<min>:<max>)
// In this case <decimal> is left blank, and will use default values.
{
  "age": "Number.number::18:50"
}
```

#### List of mock functions

Get a list of all the available Mock functions.

```bash
ktns mock --list
```

#### Generating multiple values

##### Root objects

Generating muliple root objects can be done with the flag `--generate <number>` if using `--parse-json`.

```bash
ktns mock --parse-json '{ "company": "{{ Company.name }}", "employee": { "name": "{{ Person.fullName }}" }}' --generate 10
```

When using `--parse-files`, specify the desired number of root objects in the template file's name, between brackets.

A template file named `employees[5].template.json` bellow:

```json
{
  "name": "{{ Person.name }}"
}
```

Will produce a `employees[5].json` of results like:

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
  "employees[2]": {                         // Will generate an array of employees with 5 objects
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

</br>
</br>

## Http (`request`) command

```bash
ktns request <flags>
```

> ***Data can be mocked currently only in `--data`, `--qs` and `--url` flags.**

### Flags

- `--method`: The Http method (`GET` | `POST` | `PUT` | `DELETE`).
- `--https`: If set, force https. _(If not stated, it will use the `url` protocol)_
- `--url`: The request url with added Url params. _(e.g. `localhost:3000`, `localhost:8000/api/users`, `api.com/user/{{UUID.uuidv4}}`)_
- `--header`: Multiple `string` values to be set as headers for the requests.
- `--data`: A `string` json object defining data to be used at the request body.
- `--qs`: Multiple `string` values defining query string values to be used at the request.
- `--response-accessor`: A `string` value to specify how the response should be accessed, with the idea of returning a more specific segment of the response. _(If unable to access, it returns the whole response)_
- `--with-metrics`: If set, show metrics of the request on the response.
- `--only-response-body`: If set, will return only the response's body.

</br>

### Examples

#### Simple examples

```bash
ktns request --method GET --url https://some-api.com/object
ktns request --method POST --url https://some-api.com/object/new --data '{ "name": "A new object" }'
ktns request --method PUT --url https://some-api.com/object/<object-id> --data '{ "name": "A changed object" }'
ktns request --method DELETE --url https://some-api.com/object/<object-id>
```

#### Mocking request data

```bash
ktns request
  --medhod POST
  --url https://some-api.com/person/new
  --data '{ "name": "{{ Person.name }}", "phones[3]": "{{ Person.phoneNumber }}" }'
```

#### Mocking query string data

```bash
ktns request
  --medhod GET
  --url https://some-api.com/person
  --qs 'ageMin={{ Number.number:0:1:10 }}'
  --qs 'ageMax={{ Number.number:0:50:55 }}'
```

#### Mocking url param data

```bash
ktns request
  --medhod GET
  --url https://some-api.com/person/{{ UUID.uuidv4 }}
```

#### Forcing HTTPS

Forcing it, will produce `https://localhost:8000/person`.

```bash
ktns request
  --medhod GET
  --url localhost:8000/person
  --https
```

#### Adding authentication header

```bash
ktns request
  --medhod GET
  --url https://some-api/person
  --header "Authentication: Bearer <token>"
```

</br>

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

```bash
{ ... }
```

#### Get more specific body response `--response-accessor <accessor>`

> To be done.

</br>
</br>

# Development

The `Cobra` commands are in `/cmd` along with its tests.

The `/mocker` folder holds the mocker object that currently only uses [`github.com/jaswdr/faker/v2`](https://github.com/jaswdr/faker) for most of the mock functions. Additional function were added manually.

### Execute app

```bash
go run . <subparam> [flags]
```

### Run tests

```bash
go test ./...
```


