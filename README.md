# The project

k-test-n-stress is a simple tool to facilate:

1. Generation of fake data.
2. Running http tests on endpoints.
3. Generating stress tests on endpoints.

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
  - <flag>
  <flag>: <flag-value>
```

</br>

## Faker (`mock`) command

```bash
ktns mock <flags>
```

#### Flags

- `--list`: If set, it will list all available mock functions.
- `--parseStr`: Pass a JSON object as a string. The mock data will be generated based on the provided object.
- `--parseFrom`: Pass a path, directory, or glob pattern to find template files (`.template.json`). The mock data will be generated based on the found files.
- `--preserveFolderStructure`: If set, the folder structure of the input files will be preserved in the output files.

#### Example (`--parseStr`)

```bash
ktns mock --parseStr '{ "company": "Company.name", "employee": { "name": "Person.fullName" }}'
```

#### Example (`--parseFrom`)

```json
{
  "company": "Company.name",
  "employee": {
    "name": "Person.fullName"
  }
}
```

```bash
  ktns mock --parseFrom example.template.json
  ktns mock --parseFrom "*.template.json"
  ktns mock --parseFrom "test/templates/*.template.json"
  ktns mock --parseFrom "test/templates" --preserveFolderStructure
```

#### The Json template object

The Json template informated have some limitations.

- The `values` of each json key may be:
  - A `string` with the Faker function name.
  - Another `object`, at any depth.
  - An `array` of either `string` OR `object`. _(Matrixes not treated)_

#### Mock functions optional parameters

Some of the mock functions accept additional parameters, and they may be informed by delimiting with `:`.

```bash
ktns mock --parseStr '{ "words": "Loreum.words:5" }'
```

#### List of mock functions

```bash
ktns mock --list
```
