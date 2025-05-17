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

Or by specifying `commands` and `flags` in a `.yaml` file and feeding it to `ktns`.

```bash
ktns -f <file.yaml>
```

```yaml
command: <command>
  <flag>: <flag-value>
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
- `--parseJson`: Pass a JSON object as a string. The mock data will be generated based on the provided object.
- `--parseFiles`: Pass a path, directory, or glob pattern to find template files (`.template.json`). The mock data will be generated based on the found files.
- `--preserveFolderStructure`: If set, the folder structure of the input files will be preserved in the output files.
- `--generate`: Pass the desired amount of root objects that will be generated (only available for `--parseJson`). (More info [here](#generating-multiple-values))

</br>

### Examples

#### Example (`--parseJson`)

```bash
ktns mock --parseJson '{ "company": "{{ Company.name }}", "employee": { "name": "{{ Person.fullName }}" }}'
```

#### Example (`--parseFiles`)

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
  ktns mock --parseFiles example.template.json
  ktns mock --parseFiles "*.template.json"
  ktns mock --parseFiles "test/templates/*.template.json"
  ktns mock --parseFiles "test/templates" --preserveFolderStructure
```

</br>

### Details

#### Limitations of the template objects

The `value` of an object key may be:
- A `string` value with the **Faker function name *(between double brackets)***.
- An `object`, detailing an inner object.
- An `array` of either `string` OR `object`.

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

Generating muliple root objects can be done with the flag `--generate <number>` if using `--parseJson`.

```bash
ktns mock --parseJson '{ "company": "{{ Company.name }}", "employee": { "name": "{{ Person.fullName }}" }}' --generate 10
```

When using `--parseFiles`, specify the desired number of root objects in the template file's name, between brackets.

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
