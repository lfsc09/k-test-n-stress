package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/lfsc09/k-test-n-stress/internal/mock"
	"github.com/spf13/cobra"
)

func NewMockCmd(opts *CommandOptions) *cobra.Command {
	mockCmd := &cobra.Command{
		Use:   "mock",
		Short: "Generate mock data based from a string, an object string or from template files",
		Long: `Generate mock data based on --parse-str, (--parse-json / --parse-json-file) or (--parse-csv / --parse-csv-file) options, and output to --to-stdout and/or a file --to-file.

Template data:

* The data inside templates are interpreted either as literal blocks or dynamic blocks.
  Literal blocks are any values not wrapped in double curly braces (e.g. { "key": "thisIsALiteralBlock" }).
  Dynamic blocks are any values wrapped in double curly braces (e.g. { "key": "{{ thisIsADynamicBlock }}" }). The content inside the double curly braces is parsed as a mock function with optional parameters, and executed to generate mock data.
  Dynamic blocks can accept multiple mock function calls separated by pipe (|) to either:
    - Overwrite the output of a mock function with another (e.g. {{ Person.Name | OR_BLANK }}, will generate a random name and then overwrite it with a blank value)
    - Pipe the output of a mock function as an input parameter to another (e.g. {{ Person.Name | CACHE_WRITE:{key} }}, will generate a random name and then write it to a cache)

Function types:

* There are two types of functions (mock functions) and (pipe functions) that can be used in a dynamic block.
  (mock functions) are the ones that generate the mocked data. They are the most commonly used and are the ones listed with --list. Currently they will be in the format of Category.Function (2 parts divided by a dot), where each part has the first letter capitalized.
  (pipe functions) are the ones that process or transform the output of other functions. They are used in combination with mock functions to modify or enhance the generated data. Currently they will be in the format of FUNCTION_NAME (all uppercase letters).

Mock functions:

* List available mock functions with --list.
* Always call the mock function with the format {{ functionName:{arg1}:{arg2}:{argN} }}. (Values not wrapped in double curly braces will be considered literal values)
* When passing parameters to the mock functions, wrap each value in curly braces '{value}' and use colon ':' outside the braces as the separator between parameters (e.g. {{ Number.FloatBetween:{2}:{1}:{100} }}, {{ Date.Time:{18:00}:{20:00} }}).
* Leave a parameter empty (bare ':' or ':{}') to use its default value (e.g. {{ Number.FloatBetween:::{100} }} leaves 'decimals' and 'min' at their defaults).
* For regex parameters, wrap the pattern in slashes '{/pattern/}' (e.g. {{ Regex.Generate:{/[a-z]{3}/} }}).

Generate N root objects with --generate:

* Add --generate to specify the number of root objects to generate. (Not available for --parse-str)

Parsing JSON templates (--parse-json or --parse-json-file):

* For json inner objects, you may pass the desired number between brackets in the object's "key".

  e.g.:
  {
    "employees[5]": {
      "name": "{{ Person.Name }}",
    }
  }

  Will generate an array of 5 employees with random names.

  {
    "employees": [
      { "name": ... },
      { "name": ... },
      { "name": ... },
      { "name": ... },
      { "name": ... }
    ]
  }

* To generate array of values, also use the format "key[5]". (e.g., { "phones[5]": "{{ Person.Phone }}" } will generate an array of 5 phone numbers)

  e.g.:
  { "phones[5]": "{{ Person.Phone }}" }

  Will generate an array of 5 phone numbers.

  { "phones": [ "...", "...", "...", "...", "..." ] }

* When using --parse-json-file, the template filename must end with ".template.json".
  The default output file is written alongside the template file.
  e.g.: "path/to/employees.template.json" → "path/to/employees.json"

Parsing CSV templates (--parse-csv or --parse-csv-file):

* The CSV template must be a single depth Json object, where each 'key: value' pair is interpreted as 'colname: "value"'.
  The number of rows generated will be determined by the --generate flag (default 1).

  e.g.:
  { "name": "{{ Person.Name }}", "age": "{{ Number.IntBetween:{18}:{65} }}", "email": "{{ Person.Email }}" }

  Will generate a CSV with columns "name", "age" and "email" with corresponding mock data.

  name,age,email
  "Bill Smith",35,"bill.smith@example.com"

Output routing:

* Choose either --to-stdout and/or --to-file to output, the output data structure will be determinied by the parsed input.
* --to-file <filename|"">: write the result to a file.
	To use default filename, provide an empty string as filename.
  If no filename is given, defaults to output.<json|csv> in the current working directory (for --parse-json or --parse-csv)
  or to the template name without .template (for --parse-json-file or --parse-csv-file).

Examples:
  ktns mock --parse-str 'Hello my name is {{ Person.Name }}, I am {{ Number.IntBetween:{1}:{100} }} years old'
  ktns mock --parse-json '{ "name": "{{ Person.Name }}" }' --generate 5 --to-stdout --to-file ""
  ktns mock --parse-json '{ "name": "{{ Person.Name }}" }' --to-file "path/to/mydata.json"
  ktns mock --parse-json-file "path/to/employees.template.json" --generate 5 --to-stdout --to-file ""
  ktns mock --parse-json-file "path/to/employees.template.json" --to-file "path/to/mydata.json"
  ktns mock --parse-csv '{ "name": "{{ Person.Name }}" }' --generate 10 --to-stdout --to-file ""
  ktns mock --parse-csv '{ "name": "{{ Person.Name }}" }' --to-file "path/to/mydata.csv"
  ktns mock --parse-csv-file "path/to/employees.template.csv" --generate 10 --to-stdout --to-file ""
  ktns mock --parse-csv-file "path/to/employees.template.csv" --to-file "path/to/mydata.csv"
	`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Set up OS signal handling so Ctrl-C cancels in-flight work cleanly.
			ctx, stopSignal := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
			defer stopSignal()

			list, _ := cmd.Flags().GetBool("list")
			parseStr, _ := cmd.Flags().GetString("parse-str")
			parseJson, _ := cmd.Flags().GetString("parse-json")
			parseJsonFile, _ := cmd.Flags().GetString("parse-json-file")
			parseJsonFileSet := cmd.Flags().Changed("parse-json-file")
			parseCsv, _ := cmd.Flags().GetString("parse-csv")
			parseCsvFile, _ := cmd.Flags().GetString("parse-csv-file")
			parseCsvFileSet := cmd.Flags().Changed("parse-csv-file")
			generate, _ := cmd.Flags().GetInt("generate")
			toStdout, _ := cmd.Flags().GetBool("to-stdout")
			toFile, _ := cmd.Flags().GetString("to-file")
			toFileSet := cmd.Flags().Changed("to-file")
			debugMode, _ := cmd.Flags().GetBool("debug")

			if list {
				faker := mock.NewFaker()
				mockTableRows, pipeTableRows := faker.DocsToTable()
				mockTable := table{
					cols: []*tableCol{
						{size: 45, name: "MOCK FUNCTION"},
						{size: 60, name: "DESCRIPTION"},
					},
				}
				mockTable.print(opts.Out, mockTableRows, false)
				pipeTable := table{
					cols: []*tableCol{
						{size: 45, name: "PIPE FUNCTION"},
						{size: 60, name: "DESCRIPTION"},
					},
				}
				pipeTable.print(opts.Out, pipeTableRows, true)
				return nil
			}

			runningParseStr, runningParseJson, runningParseJsonFile, runningParseCsv, runningParseCsvFile := false, false, false, false, false

			// Check if --parse-str, --parse-json, --parse-json-file, --parse-csv or --parse-csv-file is provided
			parseCheck := 0
			if parseStr != "" {
				parseCheck++
				runningParseStr = true
			}
			if parseJson != "" {
				parseCheck++
				runningParseJson = true
			}
			if parseJsonFile != "" {
				parseCheck++
				runningParseJsonFile = true
			}
			if parseCsv != "" {
				parseCheck++
				runningParseCsv = true
			}
			if parseCsvFile != "" {
				parseCheck++
				runningParseCsvFile = true
			}
			if parseCheck == 0 {
				return fmt.Errorf("nothing to be parsed, ask for help -h or --help")
			} else if parseCheck > 1 {
				return fmt.Errorf("provide only one of the five options: --parse-str, --parse-json, --parse-json-file, --parse-csv or --parse-csv-file")
			}

			if generate > 1 && !runningParseJson && !runningParseJsonFile && !runningParseCsv && !runningParseCsvFile {
				return fmt.Errorf("--generate option is only available when using --parse-json, --parse-json-file, --parse-csv or --parse-csv-file")
			}

			if generate <= 0 {
				return fmt.Errorf("--generate option must be greater than 0")
			}

			// Validate --parse-json-file filename constraint
			if runningParseJsonFile && !strings.HasSuffix(filepath.Base(parseJsonFile), ".template.json") {
				return fmt.Errorf("--parse-json-file requires the template filename to end with '.template.json', got '%s'", filepath.Base(parseJsonFile))
			}

			// Validate --parse-csv-file filename constraint
			if runningParseCsvFile && !strings.HasSuffix(filepath.Base(parseCsvFile), ".template.csv") {
				return fmt.Errorf("--parse-csv-file requires the template filename to end with '.template.csv', got '%s'", filepath.Base(parseCsvFile))
			}

			// Validate --parse-str incompatibility with output flags
			if runningParseStr && (toStdout || toFileSet) {
				return fmt.Errorf("--parse-str always outputs to stdout; --to-stdout and --to-file are not available with --parse-str")
			}

			// Validate at least one output flag when using --parse-json, --parse-json-file, --parse-csv or --parse-csv-file
			if !runningParseStr && !toStdout && !toFileSet {
				return fmt.Errorf("--parse-json, --parse-json-file, --parse-csv and --parse-csv-file require at least one output flag: --to-stdout or --to-file")
			}

			if runningParseStr {
				// Compile the string value into mock blocks
				mockBlocks, err := mock.CompileMockBlocks(parseStr)
				if err != nil {
					return err
				}

				faker := mock.NewFaker()
				mockedStr, err := mock.ExecuteMockBlocks(mockBlocks, faker)
				if err != nil {
					return err
				}

				// Print the mocked string to STDOUT
				fmt.Fprintf(opts.Out, "%s\n", mockedStr)
				return nil
			}

			if runningParseJson || runningParseJsonFile {
				var jsonPath string
				if toFileSet {
					var err error
					jsonPath, err = mock.ResolveJSONToFilePath(parseJsonFileSet, parseJsonFile, toFile)
					if err != nil {
						return err
					}
				}

				var rawJson map[string]any
				if runningParseJson {
					// Parse the raw template object content
					if err := json.Unmarshal([]byte(parseJson), &rawJson); err != nil {
						return fmt.Errorf("failed to parse JSON from the provided --parse-json '%w'", err)
					}
				}
				if runningParseJsonFile {
					// Verify the file exists
					if _, err := os.Stat(parseJsonFile); os.IsNotExist(err) {
						return fmt.Errorf("template file not found: '%s'", parseJsonFile)
					}

					// Read the template file
					templateFileContent, err := os.ReadFile(parseJsonFile)
					if err != nil {
						return fmt.Errorf("failed to read --parse-json-file '%w'", err)
					}

					// Parse the raw template object content
					if err = json.Unmarshal(templateFileContent, &rawJson); err != nil {
						return fmt.Errorf("failed to parse JSON from --parse-json-file '%w'", err)
					}
				}

				blueprintInitialNode := &mock.TemplateJsonNode{Type: mock.NodeObject, Repeat: uint(generate)}
				templateGenerateTotal, err := mock.CompileJSONTemplate(rawJson, blueprintInitialNode)
				if err != nil {
					return err
				}

				stats := &mock.DebugStats{
					GenerateTotal: templateGenerateTotal * uint(generate),
					StartTime:     time.Now(),
				}

				if debugMode {
					debugStop := mock.StartDebugRoutine(ctx, stats, os.Stderr)
					defer debugStop()
				}

				var atomicFile *mock.AtomicFile
				if toFileSet {
					atomicFile, err = mock.NewAtomicFile(jsonPath)
					if err != nil {
						return err
					}
				}

				var stdoutWriter io.Writer
				if toStdout {
					stdoutWriter = opts.Out
				}

				bufferWriter := mock.NewBufferWriter(stdoutWriter, atomicFile)
				if bufferWriter == nil {
					return fmt.Errorf("failed to initialize output writer: no valid output destination configured")
				}

				faker := mock.NewFaker()
				if err = blueprintInitialNode.GenerateJSON(faker, stats, bufferWriter); err != nil {
					return err
				}
				if err = bufferWriter.Done(); err != nil {
					return err
				}
			}

			if runningParseCsv || runningParseCsvFile {
				var csvPath string
				if toFileSet {
					var err error
					csvPath, err = mock.ResolveCSVToFilePath(parseCsvFileSet, parseCsvFile, toFile)
					if err != nil {
						return err
					}
				}

				var csvTemplateObj map[string]any
				if runningParseCsv {
					// Parse the raw template object content
					if err := json.Unmarshal([]byte(parseCsv), &csvTemplateObj); err != nil {
						return fmt.Errorf("failed to parse CSV from the provided --parse-csv '%w'", err)
					}
				}
				if runningParseCsvFile {
					// Verify the file exists
					if _, err := os.Stat(parseCsvFile); os.IsNotExist(err) {
						return fmt.Errorf("template file not found: '%s'", parseCsvFile)
					}

					// Read the template file
					templateFileContent, err := os.ReadFile(parseCsvFile)
					if err != nil {
						return fmt.Errorf("failed to read --parse-csv-file '%w'", err)
					}

					// Parse the raw template object content
					if err = json.Unmarshal(templateFileContent, &csvTemplateObj); err != nil {
						return fmt.Errorf("failed to parse CSV from --parse-csv-file '%w'", err)
					}
				}

				blueprint := &mock.TemplateCsv{Repeat: uint(generate)}
				templateGenerateTotal, err := mock.CompileCSVTemplate(csvTemplateObj, blueprint)
				if err != nil {
					return err
				}

				stats := &mock.DebugStats{
					GenerateTotal: templateGenerateTotal * uint(generate),
					StartTime:     time.Now(),
				}

				if debugMode {
					debugStop := mock.StartDebugRoutine(ctx, stats, os.Stderr)
					defer debugStop()
				}

				var atomicFile *mock.AtomicFile
				if toFileSet {
					atomicFile, err = mock.NewAtomicFile(csvPath)
					if err != nil {
						return err
					}
				}

				var stdoutWriter io.Writer
				if toStdout {
					stdoutWriter = opts.Out
				}

				bufferWriter := mock.NewBufferWriter(stdoutWriter, atomicFile)
				if bufferWriter == nil {
					return fmt.Errorf("failed to initialize output writer: no valid output destination configured")
				}

				faker := mock.NewFaker()
				if err = blueprint.GenerateCSV(faker, stats, bufferWriter); err != nil {
					return err
				}
				if err = bufferWriter.Done(); err != nil {
					return err
				}
			}

			return nil
		},
	}

	mockCmd.Flags().Bool("list", false, "list all available mock functions")
	mockCmd.Flags().String("parse-str", "", "pass a string to be parsed. The mock data will be generated based on this provided string")
	mockCmd.Flags().String("parse-json", "", "pass a JSON object as a string. The mock data will be generated based on this provided json object")
	mockCmd.Flags().String("parse-json-file", "", "pass a path to a single .template.json file. The mock data will be generated based on this file")
	mockCmd.Flags().String("parse-csv", "", "pass a CSV template as a string. The mock data will be generated based on this provided CSV template")
	mockCmd.Flags().String("parse-csv-file", "", "pass a path to a single .template.csv file. The mock data will be generated based on this file")
	mockCmd.Flags().Int("generate", 1, "pass the desired amount of root objects that will be generated (not available for --parse-str)")
	mockCmd.Flags().Bool("to-stdout", false, "output result as JSON to stdout")
	mockCmd.Flags().String("to-file", "", "output result as JSON to a file; optional filename argument")
	mockCmd.Flags().Bool("debug", false, "show live generation progress on stderr")

	// Configure cobra ouput streams to use the custom 'Out'
	mockCmd.SetOut(opts.Out)

	return mockCmd
}

type tableCol struct {
	size int
	name string
}

type table struct {
	cols []*tableCol
}

// printTable prints a formatted table with the given rows to the specified output writer.
func (t table) print(out io.Writer, rows [][]string, hasBottomDivider bool) {
	fmt.Fprintf(out, "%s\n", t.divider())
	fmt.Fprintf(out, "%s\n", t.header())
	fmt.Fprintf(out, "%s\n", t.divider())
	for _, row := range rows {
		fmt.Fprintf(out, "%s\n", t.data(row))
	}
	if hasBottomDivider {
		fmt.Fprintf(out, "%s\n", t.divider())
	}
}

// divider generates a string that represents a divider line for a table based on the provided column sizes.
func (t table) divider() string {
	var line strings.Builder
	for idx, col := range t.cols {
		if idx == 0 {
			line.WriteString(strings.Repeat("-", col.size))
		} else {
			line.WriteString("+" + strings.Repeat("-", col.size))
		}
	}
	return line.String()
}

// header generates a string that represents the header line for a table based on the provided column sizes.
func (t table) header() string {
	var line strings.Builder
	for idx, col := range t.cols {
		if idx == 0 {
			fmt.Fprintf(&line, "%-*s", col.size, col.name)
		} else {
			fmt.Fprintf(&line, "| %-*s", col.size, col.name)
		}
	}
	return line.String()
}

// data generates a string that represents a data line for a table based on the provided column sizes and data.
func (t table) data(row []string) string {
	var line strings.Builder
	for idx, col := range t.cols {
		if idx == 0 {
			fmt.Fprintf(&line, "%-*s", col.size, row[idx])
		} else {
			fmt.Fprintf(&line, "| %-*s", col.size, row[idx])
		}
	}
	return line.String()
}
