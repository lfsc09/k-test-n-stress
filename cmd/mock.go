package cmd

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/lfsc09/k-test-n-stress/mocker"
	"github.com/mohae/deepcopy"
	"github.com/spf13/cobra"
)

var objKeyNumberRegex = regexp.MustCompile(`^[^\[\]\s]+\[(\d+)\]$`)

func NewMockCmd(opts *CommandOptions) *cobra.Command {
	mockCmd := &cobra.Command{
		Use:   "mock",
		Short: "Generate mock data based from an object string or from template files",
		Long: `Generate mock data based on --parse-str, --parse-json or --parse-json-file options.

Mock functions:

* List available mock functions with --list.
* Always call the mock function with the format {{ functionName:{arg1}:{arg2}:{argN} }}. (Values not wrapped in double curly braces will be considered literal values)
* When passing parameters to the mock functions, wrap each value in curly braces ({value}) and use colon (:) outside the braces as the separator between parameters (e.g. {{ Number.number::{1}:{100} }}, {{ Date.time:{18:00}:{20:00} }}).
* Leave a parameter empty (bare :: or {}) to use its default value (e.g. {{ Number.number:::{100} }} leaves decimals and min at their defaults).
* For regex parameters, wrap the pattern in slashes (/pattern/) instead of curly braces (e.g. {{ Date.date:::/YYYY-MM-DD/ }}, {{ Regex.regex:/[a-z]{3}/ }}). Colons inside /…/ are never treated as delimiters.

Controling the number of generated data:

* Add --generate to specify the number of root objects to generate. (Available for --parse-json and --parse-json-file)
* For inner objects, also pass the desired number between brackets in the object's "key".

  e.g.:
  {
    "employees[5]": {
      "name": "{{ Person.name }}",
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

* To generate array of values, also use the format "key[5]". (e.g., { "phones[5]": "{{ Person.phoneNumber }}" } will generate an array of 5 phone numbers)

  e.g.:
  {
    "phones[5]": "{{ Person.phoneNumber }}"
  }

  Will generate an array of 5 phone numbers.

  {
    "phones": [ "...", "...", "...", "...", "..." ]
  }

* When using --parse-json-file, the template filename must end with ".template.json".
  The default output file is written alongside the template file.
  e.g.: "path/to/employees.template.json" → "path/to/employees.json"

Output routing (--parse-json and --parse-json-file):

* At least one of --to-stdout, --to-json-file or --to-csv-file must be provided.
* --to-stdout <as-json|as-csv>: print the result to stdout as JSON or CSV.
* --to-stdout-prettify: format the stdout output for readability (only valid with --to-stdout).
* --to-json-file [filename]: write the result as JSON to a file.
  If no filename is given, defaults to output.json beside the binary (for --parse-json)
  or to the template name without .template (for --parse-json-file).
  Note: --to-json-file requires an explicit value or use --to-json-file "" for the default.
* --to-csv-file [filename]: write the result as CSV to a file.
  If no filename is given, defaults to output.csv beside the binary (for --parse-json)
  or to the template name without .template with a .csv extension (for --parse-json-file).
  Note: --to-csv-file requires an explicit value or use --to-csv-file "" for the default.
* CSV output works best with flat (one-level-deep) JSON objects. Nested objects and arrays
  are serialised using their Go string representation.

Examples:
  ktns mock --parse-str '{{ Person.name }}'
  ktns mock --parse-str 'Hello my name is {{ Person.name }}, I am {{ Number.number:{0}:{1}:{100} }} years old'
  ktns mock --parse-json '{ "name": "{{ Person.name }}", "age": "{{ Number.number:{0}:{1}:{100} }}" }' --to-stdout as-json
  ktns mock --parse-json '{ "name": "{{ Person.name }}" }' --to-stdout as-csv
  ktns mock --parse-json '{ "name": "{{ Person.name }}" }' --to-stdout as-json --to-stdout-prettify
  ktns mock --parse-json '{ "name": "{{ Person.name }}" }' --to-json-file
  ktns mock --parse-json '{ "name": "{{ Person.name }}" }' --to-json-file mydata.json
  ktns mock --parse-json '{ "phones[2]": "{{ Person.phoneNumber }}" }' --generate 5 --to-stdout as-json
  ktns mock --parse-json-file "path/to/employees.template.json" --to-stdout as-csv
  ktns mock --parse-json-file "path/to/employees.template.json" --to-json-file
  ktns mock --parse-json-file "path/to/employees.template.json" --to-json-file myout.json
  ktns mock --parse-json-file "path/to/employees.template.json" --generate 5 --to-json-file
  ktns mock --parse-json '{ "name": "{{ Person.name }}" }' --to-csv-file
  ktns mock --parse-json '{ "name": "{{ Person.name }}" }' --to-csv-file mydata.csv
  ktns mock --parse-json-file "path/to/employees.template.json" --to-csv-file
  ktns mock --parse-json-file "path/to/employees.template.json" --to-csv-file myout.csv
  ktns mock --parse-json-file "path/to/employees.template.json" --generate 5 --to-csv-file
	`,
		RunE: func(cmd *cobra.Command, args []string) error {
			list, _ := cmd.Flags().GetBool("list")
			parseStr, _ := cmd.Flags().GetString("parse-str")
			parseJson, _ := cmd.Flags().GetString("parse-json")
			parseJsonFile, _ := cmd.Flags().GetString("parse-json-file")
			generate, _ := cmd.Flags().GetInt("generate")
			toStdout, _ := cmd.Flags().GetString("to-stdout")
			toStdoutPrettify, _ := cmd.Flags().GetBool("to-stdout-prettify")
			toJsonFile, _ := cmd.Flags().GetString("to-json-file")
			toJsonFileSet := cmd.Flags().Changed("to-json-file")
			toCsvFile, _ := cmd.Flags().GetString("to-csv-file")
			toCsvFileSet := cmd.Flags().Changed("to-csv-file")

			if list {
				mocker := mocker.New()
				mocker.List(opts.Out)
				return nil
			}

			runningParseStr, runningParseJson, runningParseJsonFile := false, false, false

			// Check if --parse-json, --parse-json-file or --parse-str is provided
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
			if parseCheck == 0 {
				return fmt.Errorf("nothing to be parsed, ask for help -h or --help")
			} else if parseCheck > 1 {
				return fmt.Errorf("provide only one of the three options: --parse-json, --parse-json-file or --parse-str")
			}

			if generate > 1 && !runningParseJson && !runningParseJsonFile {
				return fmt.Errorf("--generate option is only available when using --parse-json or --parse-json-file")
			}

			if generate <= 0 {
				return fmt.Errorf("--generate option must be greater than 0")
			}

			// Validate --parse-json-file filename constraint
			if runningParseJsonFile && !strings.HasSuffix(filepath.Base(parseJsonFile), ".template.json") {
				return fmt.Errorf("--parse-json-file requires the template filename to end with '.template.json', got '%s'", filepath.Base(parseJsonFile))
			}

			// Validate --to-stdout value
			if toStdout != "" && toStdout != "as-json" && toStdout != "as-csv" {
				return fmt.Errorf("--to-stdout only accepts 'as-json' or 'as-csv', got '%s'", toStdout)
			}

			// Validate --to-stdout-prettify requires --to-stdout
			if toStdoutPrettify && toStdout == "" {
				return fmt.Errorf("--to-stdout-prettify requires --to-stdout to be set")
			}

			// Validate --parse-str incompatibility with output flags
			if runningParseStr && (toStdout != "" || toJsonFileSet || toCsvFileSet) {
				return fmt.Errorf("--parse-str always outputs to stdout; --to-stdout, --to-json-file and --to-csv-file are not available with --parse-str")
			}

			// Validate at least one output flag when using --parse-json or --parse-json-file
			if !runningParseStr && toStdout == "" && !toJsonFileSet && !toCsvFileSet {
				return fmt.Errorf("--parse-json and --parse-json-file require at least one output flag: --to-stdout, --to-json-file or --to-csv-file")
			}

			if runningParseStr {
				// Process the string
				mocker := mocker.New()
				mockedStr := processStr(parseStr, mocker)

				// Print the mocked string to STDOUT
				fmt.Fprintf(opts.Out, "%s\n", mockedStr)
			}

			// Parse string json object from `--parse-json`
			if runningParseJson {
				// Parse the string object content
				var parseMap map[string]any
				if err := json.Unmarshal([]byte(parseJson), &parseMap); err != nil {
					return fmt.Errorf("failed to parse JSON from the provided --parse-json '%w'", err)
				}

				// Process the parsed map
				mocker := mocker.New()
				parseMaps := make([]map[string]any, generate)
				for i := range generate {
					cpParseMap := deepcopy.Copy(parseMap).(map[string]any)
					if err := processJsonMap(cpParseMap, mocker); err != nil {
						return fmt.Errorf("%w", err)
					}
					parseMaps[i] = deepcopy.Copy(cpParseMap).(map[string]any)
				}

				// Sanitize the parsed map
				for i := range parseMaps {
					sanitizeJsonMap(parseMaps[i])
				}

				if err := routeOutput(parseMaps, generate, toStdout, toStdoutPrettify, toJsonFileSet, toJsonFile, toCsvFileSet, toCsvFile, "parse-json", "", "", opts.Out); err != nil {
					return err
				}
			}

			// Parse object from `--parse-json-file` file
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

				// Parse the template file content
				var parseMap map[string]any
				if err = json.Unmarshal(templateFileContent, &parseMap); err != nil {
					return fmt.Errorf("failed to parse JSON from --parse-json-file '%w'", err)
				}

				// Determine the default output file path: same directory as the template file, .template.json → .json
				defaultFileName := strings.Replace(filepath.Base(parseJsonFile), ".template.json", ".json", 1)
				defaultFilePath := filepath.Join(filepath.Dir(parseJsonFile), defaultFileName)
				defaultCsvFileName := strings.Replace(filepath.Base(parseJsonFile), ".template.json", ".csv", 1)
				defaultCsvFilePath := filepath.Join(filepath.Dir(parseJsonFile), defaultCsvFileName)

				// Process the parsed map
				mocker := mocker.New()
				parseMaps := make([]map[string]any, generate)
				for i := range generate {
					cpParseMap := deepcopy.Copy(parseMap).(map[string]any)
					if err := processJsonMap(cpParseMap, mocker); err != nil {
						return fmt.Errorf("%w", err)
					}
					parseMaps[i] = deepcopy.Copy(cpParseMap).(map[string]any)
				}

				// Sanitize the parsed maps
				for i := range parseMaps {
					sanitizeJsonMap(parseMaps[i])
				}

				if err := routeOutput(parseMaps, generate, toStdout, toStdoutPrettify, toJsonFileSet, toJsonFile, toCsvFileSet, toCsvFile, "parse-json-file", defaultFilePath, defaultCsvFilePath, opts.Out); err != nil {
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
	mockCmd.Flags().Int("generate", 1, "pass the desired amount of root objects that will be generated (only available for --parse-json and --parse-json-file)")
	mockCmd.Flags().String("to-stdout", "", "output result to stdout as 'as-json' or 'as-csv'")
	mockCmd.Flags().Bool("to-stdout-prettify", false, "prettify the stdout output (only valid with --to-stdout)")
	mockCmd.Flags().String("to-json-file", "", "output result as JSON to a file; optional filename argument")

	// Allow --to-json-file to be used without a value (uses sentinel "_use_default_")
	mockCmd.Flags().Lookup("to-json-file").NoOptDefVal = "_use_default_"

	mockCmd.Flags().String("to-csv-file", "", "output result as CSV to a file; optional filename argument")
	mockCmd.Flags().Lookup("to-csv-file").NoOptDefVal = "_use_default_"

	// Configure cobra ouput streams to use the custom 'Out'
	mockCmd.SetOut(opts.Out)

	return mockCmd
}

// Splits a raw string of format "func:{arg1}:{arg2}:...".
// It handles regex args wrapped with slashes (/.../) and value args wrapped with curly
// braces ({...}) to avoid splitting inside them. The colon (:) outside both delimiters
// acts as the separator between positional parameters.
// Returns: function name, slice of parameter strings, and an error if any parameter
// after the function name is not wrapped in {…} or /…/.
func extractMockMethod(rawValue string) (string, []string, error) {
	if rawValue == "" {
		return "", nil, nil
	}
	var parts []string
	var buf strings.Builder
	inRegex := false
	inValue := false
	valueOpened := false // tracks if a {…} block was opened for the current param
	paramCount := 0
	inBare := false

	trimmed := strings.TrimSpace(rawValue)

	for _, char := range trimmed {
		switch {
		case char == '{' && !inRegex && !inValue:
			// Open a value token — do not write the brace
			inValue = true
			valueOpened = true
		case char == '}' && inValue:
			// Close the value token — do not write the brace
			inValue = false
		case char == '/' && !inValue:
			// Toggle regex mode; always include the slash in the buffer
			inRegex = !inRegex
			buf.WriteRune(char)
		case char == ':' && !inRegex && !inValue:
			// Delimiter outside both token types — flush buffer
			parts = append(parts, buf.String())
			buf.Reset()
			paramCount++
			// If we have passed the function name and a bare token was detected, error
			if paramCount >= 1 && inBare {
				return "", nil, fmt.Errorf("mock function parameter '%s' must be wrapped in {…} for a value or /…/ for a regex", parts[len(parts)-1])
			}
			inBare = false
			valueOpened = false
		default:
			// All other characters, including content inside {…} or /…/
			buf.WriteRune(char)
			// Mark bare only when outside any delimiter and past the function name
			if paramCount >= 1 && !inRegex && !inValue {
				inBare = true
			}
		}
	}

	// Add the final piece (there's no trailing `:` — but a trailing `:` means the last
	// param is an empty string that must still be included).
	// Include when: buffer has content, a {…} block was opened, still inside a delimiter,
	// or at least one separator colon has been seen (paramCount >= 1), which means a
	// trailing empty param after the last colon is intentional.
	if buf.Len() > 0 || inValue || valueOpened || paramCount >= 1 {
		parts = append(parts, buf.String())
	}

	// Check for bare token at end of input (after function name)
	if paramCount >= 1 && inBare {
		return "", nil, fmt.Errorf("mock function parameter '%s' must be wrapped in {…} for a value or /…/ for a regex", buf.String())
	}

	return parts[0], parts[1:], nil
}

// Interprets a string value, checking if it contains a mock function between {{ }}.
// If it does, it returns the function name and true.
// If not, it returns the original string and false.
func interpretString(rawValue string) (string, bool) {
	if rawValue == "" {
		return "", false
	}

	re := regexp.MustCompile(`^\s*{{(.*)}}\s*$`)
	matches := re.FindStringSubmatch(rawValue)

	if len(matches) > 0 {
		return strings.TrimSpace(matches[1]), true
	}

	return rawValue, false
}

// Iterates through the parsed json map and processes each value.
// It replaces string values with generated mock data based on the function name and parameters.
// It handles nested maps and arrays of strings or maps.
// Returns an error if any value is not a string or map.
func processJsonMap(parseMap map[string]any, mocker *mocker.Mock) error {
	objKeys := make([]string, 0, len(parseMap))
	for key := range parseMap {
		objKeys = append(objKeys, key)
	}

	for keyIndex := 0; keyIndex < len(objKeys); {
		objKey := objKeys[keyIndex]
		switch typedValue := parseMap[objKey].(type) {
		case string:
			// try to find [digit] in the "key"
			generateAmount, err := extractDigitInBrackets("object", objKey)
			if err != nil {
				return err
			}
			// try to find the mock function in the "value"
			interpretedValue, isMockFunction := interpretString(typedValue)
			// if it's not a mock function, just replace the value
			if !isMockFunction {
				parseMap[objKey] = interpretedValue
				keyIndex++
				continue
			}
			// if it's a mock function, extract the function name and parameters
			functionName, params, err := extractMockMethod(interpretedValue)
			if err != nil {
				return err
			}
			// either generate array of values, otherwise only one value
			if generateAmount > 1 {
				parseMap[objKey] = make([]string, generateAmount)
				for i := range generateAmount {
					mockValue, err := mocker.Generate(functionName, params)
					if err != nil {
						return err
					}
					parseMap[objKey].([]string)[i] = mockValue
				}
			} else {
				mockValue, err := mocker.Generate(functionName, params)
				if err != nil {
					return err
				}
				parseMap[objKey] = mockValue
			}
			keyIndex++
		case map[string]any:
			// try to find [digit] in the "key"
			generateAmount, err := extractDigitInBrackets("object", objKey)
			if err != nil {
				return err
			}
			// if generating multiple values, convert the map to a slice of maps (but force the type to generic any) and reprocess again
			if generateAmount > 1 {
				convertedValue := make([]any, generateAmount)
				for i := range generateAmount {
					convertedValue[i] = deepcopy.Copy(typedValue)
				}
				parseMap[objKey] = convertedValue
			} else {
				if err := processJsonMap(typedValue, mocker); err != nil {
					return err
				}
				keyIndex++
			}
		case []any:
			for itemKey, item := range typedValue {
				if itemStr, ok := item.(string); ok {
					interpretedValue, isMockFunction := interpretString(itemStr)
					if !isMockFunction {
						typedValue[itemKey] = interpretedValue
						continue
					}
					functionName, params, err := extractMockMethod(interpretedValue)
					if err != nil {
						return err
					}
					mockValue, err := mocker.Generate(functionName, params)
					if err != nil {
						return err
					}
					typedValue[itemKey] = mockValue
				} else if itemMap, ok := item.(map[string]any); ok {
					err := processJsonMap(itemMap, mocker)
					if err != nil {
						return err
					}
				} else {
					return fmt.Errorf("value '%v' is not a string or map", item)
				}
			}
			keyIndex++
		default:
			return fmt.Errorf("value '%v' is not a string, map or array", typedValue)
		}
	}
	return nil
}

// Iterates through the parsed json map and sanitizes the keys by removing segments between bracketes (e.g. [digits]).
// It handles nested maps.
func sanitizeJsonMap(parseMap map[string]any) {
	// Clone keys to avoid modifying map during iteration
	objKeys := make([]string, 0, len(parseMap))
	for objKey := range parseMap {
		objKeys = append(objKeys, objKey)
	}

	for _, objKey := range objKeys {
		objValue := parseMap[objKey]
		sanitizedKey := sanitizeKeyWithBrackets(objKey)

		// Recurse on nested maps
		if mapValue, ok := objValue.(map[string]any); ok {
			sanitizeJsonMap(mapValue)
		}

		if sanitizedKey != objKey {
			parseMap[sanitizedKey] = objValue
			delete(parseMap, objKey)
		}
	}
}

// Process a simple string value, checking if it contains a mock function.
// If it does, it generates the mock value using the mocker.
// If not, it returns the original string.
// Mock function calls are delimited by {{ and }} where }} is always the closing
// delimiter — single } inside the content (e.g. inside {value} param tokens) is allowed.
func processStr(parseStr string, mocker *mocker.Mock) string {
	var out strings.Builder
	s := parseStr
	for {
		// Find the next opening "{{"
		start := strings.Index(s, "{{")
		if start == -1 {
			out.WriteString(s)
			break
		}
		// Write everything before the opening "{{"
		out.WriteString(s[:start])
		s = s[start+2:] // skip past "{{"

		// Find the closing "}}" — single "}" is allowed inside
		end := -1
		for i := 0; i < len(s)-1; i++ {
			if s[i] == '}' && s[i+1] == '}' {
				end = i
				break
			}
		}
		if end == -1 {
			// No closing "}}" found — treat the rest as literal
			out.WriteString("{{")
			out.WriteString(s)
			break
		}

		inner := strings.TrimSpace(s[:end])
		s = s[end+2:] // skip past "}}"

		functionName, params, err := extractMockMethod(inner)
		if err != nil {
			out.WriteString(fmt.Sprintf("[%v]", err))
		} else {
			mockValue, err := mocker.Generate(functionName, params)
			if err != nil {
				out.WriteString(fmt.Sprintf("[%v]", err))
			} else {
				out.WriteString(mockValue)
			}
		}
	}
	return out.String()
}

// Extracts a digit from a string in the format "content[<digit>]".
// Only handles "object" place values. Returns an error immediately if place != "object".
// If the string doesn't contain brackets, it returns 1.
func extractDigitInBrackets(place string, str string) (int, error) {
	if place != "object" {
		return 0, fmt.Errorf("invalid value '%s' (must be 'object')", place)
	}

	matches := objKeyNumberRegex.FindStringSubmatch(str)

	if len(matches) != 2 {
		if !regexp.MustCompile(`[\[\]]`).MatchString(str) {
			return 1, nil
		}
		return 0, fmt.Errorf("invalid format '%s' (must be 'text[digit]')", str)
	}

	digit, err := strconv.Atoi(matches[1])
	if err != nil {
		return 0, fmt.Errorf("invalid content inside brackets in '%s'", str)
	}

	if digit <= 0 {
		return 0, fmt.Errorf("invalid digit in brackets '%s'", str)
	}

	return digit, nil
}

// Removes the segment of a string between brackets, including the brackets themselves.
// It returns the cleaned string.
func sanitizeKeyWithBrackets(str string) string {
	startBracket := strings.Index(str, "[")
	endBracket := strings.Index(str, "]")

	if startBracket != -1 && endBracket != -1 && endBracket > startBracket {
		segment := str[startBracket : endBracket+1]
		// Remove the segment from the original string
		strCleaned := strings.Replace(str, segment, "", 1)
		return strCleaned
	}
	return str
}

// executableDir returns the directory of the running ktns binary.
// Used to resolve the default output path for --to-json-file when parsing from stdin.
func executableDir() (string, error) {
	exe, err := os.Executable()
	if err != nil {
		return "", fmt.Errorf("could not determine executable path: %w", err)
	}
	return filepath.Dir(exe), nil
}

// marshalAsCSV serialises a slice of flat maps to CSV.
// If prettify is true, columns are padded to equal width for readability.
// If prettify is false, output is standard compact RFC 4180 CSV.
// Column order is determined by the sorted union of all keys across all rows.
// Values are always coerced to strings.
// Note: works best with flat (one-level-deep) JSON objects. Nested objects/arrays
// will be serialised as their Go string representation.
func marshalAsCSV(parseMaps []map[string]any, prettify bool) (string, error) {
	// Collect unique keys across all rows
	keySet := make(map[string]struct{})
	for _, m := range parseMaps {
		for k := range m {
			keySet[k] = struct{}{}
		}
	}
	headers := make([]string, 0, len(keySet))
	for k := range keySet {
		headers = append(headers, k)
	}
	sort.Strings(headers)

	// Build rows: header + one row per map
	rows := make([][]string, 0, len(parseMaps)+1)
	rows = append(rows, headers)
	for _, m := range parseMaps {
		row := make([]string, len(headers))
		for i, h := range headers {
			if val, ok := m[h]; ok {
				row[i] = fmt.Sprintf("%v", val)
			}
		}
		rows = append(rows, row)
	}

	if !prettify {
		var sb strings.Builder
		w := csv.NewWriter(&sb)
		if err := w.WriteAll(rows); err != nil {
			return "", fmt.Errorf("error writing CSV: %w", err)
		}
		w.Flush()
		if err := w.Error(); err != nil {
			return "", fmt.Errorf("error flushing CSV: %w", err)
		}
		// WriteAll adds a trailing newline; trim it so callers can add their own
		return strings.TrimRight(sb.String(), "\n"), nil
	}

	// Prettified: compute max column width
	colWidths := make([]int, len(headers))
	for _, row := range rows {
		for i, cell := range row {
			if len(cell) > colWidths[i] {
				colWidths[i] = len(cell)
			}
		}
	}

	var sb strings.Builder
	for rowIdx, row := range rows {
		// Pad each cell
		cells := make([]string, len(row))
		for i, cell := range row {
			cells[i] = fmt.Sprintf("%-*s", colWidths[i], cell)
		}
		sb.WriteString(strings.Join(cells, " | "))
		sb.WriteString("\n")
		// Separator after header row
		if rowIdx == 0 {
			sepParts := make([]string, len(headers))
			for i := range headers {
				sepParts[i] = strings.Repeat("-", colWidths[i])
			}
			sb.WriteString(strings.Join(sepParts, "---"))
			sb.WriteString("\n")
		}
	}

	return strings.TrimRight(sb.String(), "\n"), nil
}

// routeOutput handles all output routing for --parse-json and --parse-json-file results.
// parseMaps is the slice of processed root objects.
// generate is the --generate count (used to decide single-object vs array).
// toStdout is "" | "as-json" | "as-csv".
// toStdoutPrettify controls indented vs compact stdout output.
// toJsonFileSet indicates --to-json-file was passed (even with empty value).
// toJsonFileValue is the optional filename given with --to-json-file.
// toCsvFileSet indicates --to-csv-file was passed (even with empty value).
// toCsvFileValue is the optional filename given with --to-csv-file.
// source is "parse-json" or "parse-json-file".
// defaultFilePath is the full default JSON output path (used when source is "parse-json-file" and no explicit filename was given).
// defaultCsvFilePath is the full default CSV output path (used when source is "parse-json-file" and no explicit csv filename was given).
// out is the writer for stdout.
func routeOutput(
	parseMaps []map[string]any,
	generate int,
	toStdout string,
	toStdoutPrettify bool,
	toJsonFileSet bool,
	toJsonFileValue string,
	toCsvFileSet bool,
	toCsvFileValue string,
	source string,
	defaultFilePath string,
	defaultCsvFilePath string,
	out io.Writer,
) error {
	// Determine the data shape
	var data any
	if generate == 1 {
		data = parseMaps[0]
	} else {
		data = parseMaps
	}

	// Stdout output
	if toStdout != "" {
		switch toStdout {
		case "as-json":
			var jsonBytes []byte
			var err error
			if toStdoutPrettify {
				jsonBytes, err = json.MarshalIndent(data, "", "  ")
			} else {
				jsonBytes, err = json.Marshal(data)
			}
			if err != nil {
				return fmt.Errorf("error marshalling JSON for stdout: %w", err)
			}
			fmt.Fprintf(out, "%s\n", jsonBytes)
		case "as-csv":
			csvStr, err := marshalAsCSV(parseMaps, toStdoutPrettify)
			if err != nil {
				return err
			}
			fmt.Fprintf(out, "%s\n", csvStr)
		}
	}

	// File output
	if toJsonFileSet {
		var resolvedPath string
		switch {
		case toJsonFileValue != "" && toJsonFileValue != "_use_default_":
			resolvedPath = toJsonFileValue
		case source == "parse-json-file":
			resolvedPath = defaultFilePath
		default:
			// source == "parse-json": use output.json beside the binary
			dir, err := executableDir()
			if err != nil {
				return err
			}
			resolvedPath = filepath.Join(dir, "output.json")
		}

		// Delete any existing file at that path
		_ = os.Remove(resolvedPath)

		// Files are always pretty-printed for readability
		jsonBytes, err := json.MarshalIndent(data, "", "  ")
		if err != nil {
			return fmt.Errorf("error marshalling JSON for file: %w", err)
		}

		if err = os.WriteFile(resolvedPath, jsonBytes, 0644); err != nil {
			return fmt.Errorf("failed to write result to '%s': %w", resolvedPath, err)
		}
	}

	// CSV file output
	if toCsvFileSet {
		var resolvedCsvPath string
		switch {
		case toCsvFileValue != "" && toCsvFileValue != "_use_default_":
			resolvedCsvPath = toCsvFileValue
		case source == "parse-json-file":
			resolvedCsvPath = defaultCsvFilePath
		default:
			// source == "parse-json": use output.csv beside the binary
			dir, err := executableDir()
			if err != nil {
				return err
			}
			resolvedCsvPath = filepath.Join(dir, "output.csv")
		}

		// Delete any existing file at that path
		_ = os.Remove(resolvedCsvPath)

		// CSV files are written in standard (non-prettified) format
		csvStr, err := marshalAsCSV(parseMaps, false)
		if err != nil {
			return err
		}

		if err = os.WriteFile(resolvedCsvPath, []byte(csvStr+"\n"), 0644); err != nil {
			return fmt.Errorf("failed to write CSV result to '%s': %w", resolvedCsvPath, err)
		}
	}

	return nil
}
