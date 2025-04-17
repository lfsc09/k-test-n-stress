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

- `--parse`: Parse the template from the command line as a string `json` objsect.
- `--parseFrom`: Parse the template from one or more `.template.json` files. _(Separated by space)_
- `--saveTo`: If set, it will:
  - In case of `--parse`, save the result in a `mock-data.json` file.
  - In case of `--parseFrom`, save the result in its correspondent `.json` filename. _(Without `.template` part)_

#### Example (`--parse`)

```bash
ktns mock --parse '{ "company": "Company.name", "employee": { "name": "Person.fullName" }}'
```

#### Example (`--parseFrom`)

_Multiple files may be passed._

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
```

#### Example (`saveTo`)

Will generate a `mock-data.json` file with the result.

```bash
ktns mock --parse '{ "company": "Company.name", "employee": { "name": "Person.fullName" }}' --saveTo
```

Will generate `example.json` file.

```bash
ktns mock --parseFrom example.template.json --saveTo
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
ktns mock --parse '{ "words": "Loreum.words:5" }'
```

#### List of mock functions
