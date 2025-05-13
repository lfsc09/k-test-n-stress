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

Mock function must be wrapped in `{{  }}`, or values passed will be interpreted as raw values.

### Flags

- `--list`: If set, it will list all available mock functions.
- `--parseStr`: Pass a JSON object as a string. The mock data will be generated based on the provided object.
- `--parseFrom`: Pass a path, directory, or glob pattern to find template files (`.template.json`). The mock data will be generated based on the found files.
- `--preserveFolderStructure`: If set, the folder structure of the input files will be preserved in the output files.

</br>

### Examples

#### Example (`--parseStr`)

```bash
ktns mock --parseStr '{ "company": "{{ Company.name }}", "employee": { "name": "{{ Person.fullName }}" }}'
```

#### Example (`--parseFrom`)

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
  ktns mock --parseFrom example.template.json
  ktns mock --parseFrom "*.template.json"
  ktns mock --parseFrom "test/templates/*.template.json"
  ktns mock --parseFrom "test/templates" --preserveFolderStructure
```

</br>

### Details

#### Limitations of `template.json` files

- The `values` of each json key may be:
  - A `string` value with the **Faker function name**.
  - An `object`, detailing an inner object.
  - An `array` of either `string` OR `object`. _(Matrixes not treated)_

#### Mock functions optional parameters

Some of the mock functions accept additional parameters, and they are informed by delimiting with `:`.

```bash
ktns mock --parseStr '{ "words": "Loreum.words:5" }'
```

#### List of mock functions

```bash
ktns mock --list
```
