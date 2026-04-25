package cmd

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/signal"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/lfsc09/k-test-n-stress/mocker"
	"github.com/mohae/deepcopy"
	"github.com/spf13/cobra"
)

// DebugStats holds both static configuration and live counters for the --debug display.
type DebugStats struct {
	// Static fields — set once before workers start
	CoresAvailable  int
	CoresUsed       int
	TotalWeight     int64
	WeightPerWorker int64
	EstMemPeakBytes int64
	StartTime       time.Time

	// Live counters — updated atomically by workers/writer
	Generated          atomic.Int64 // incremented per item completed by each worker
	BytesWrittenToJson atomic.Int64 // incremented by writer goroutine per write
	BytesWrittenToCsv  atomic.Int64 // incremented by writer goroutine per write
}

// parseDynamicValueRegex matches a value that is entirely {{ … }}.
var parseDynamicValueRegex *regexp.Regexp = regexp.MustCompile(`^\s*{{(.*)}}\s*$`)

// bracketNumberRegex matches keys with bracketed numbers, e.g. "employees[5]" and captures the number.
var bracketNumberRegex *regexp.Regexp = regexp.MustCompile(`^[^\[\]\s]+\[(\d+)\]$`)

// bracketCharRegex detects any '[' or ']' character in a key string.
var bracketCharRegex *regexp.Regexp = regexp.MustCompile(`[\[\]]`)

// Rough estimate of bytes per generated item (for --debug memory display)
const estimatedItemBytes int = 512

func NewMockCmd(opts *CommandOptions) *cobra.Command {
	mockCmd := &cobra.Command{
		Use:   "mock",
		Short: "Generate mock data based from a string, an object string or from template files",
		Long: `Generate mock data based on --parse-str, --parse-json or --parse-json-file options, and output to --to-json-stdout and/or a file --to-json-file and/or --to-csv-file.

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

* Choose either to output to --to-json-stdout, --to-json-file or --to-csv-file.
* --to-json-file <filename|"">: write the result as JSON to a file.
	To use default filename, provide an empty string as filename.
  If no filename is given, defaults to output.json in the current working directory (for --parse-json)
  or to the template name without .template (for --parse-json-file).
* --to-csv-file <filename|"">: write the result as CSV to a file.
	To use default filename, provide an empty string as filename.
  If no filename is given, defaults to output.csv in the current working directory (for --parse-json)
  or to the template name without .template with a .csv extension (for --parse-json-file).
* CSV output works best with flat (one-level-deep) JSON objects. Nested objects and arrays
  are serialised using their Go string representation.

Examples:
  ktns mock --parse-str '{{ Person.name }}'
  ktns mock --parse-str 'Hello my name is {{ Person.name }}, I am {{ Number.number:{0}:{1}:{100} }} years old'
  ktns mock --parse-json '{ "name": "{{ Person.name }}" }' --to-json-stdout
  ktns mock --parse-json '{ "name": "{{ Person.name }}" }' --to-json-file ""
  ktns mock --parse-json '{ "name": "{{ Person.name }}" }' --to-json-file "path/to/mydata.json"
  ktns mock --parse-json '{ "name": "{{ Person.name }}" }' --to-csv-file ""
  ktns mock --parse-json '{ "name": "{{ Person.name }}" }' --to-csv-file "path/to/mydata.csv"
  ktns mock --parse-json '{ "name": "{{ Person.name }}" }' --generate 5 --to-json-file ""
  ktns mock --parse-json-file "path/to/employees.template.json" --to-json-stdout
  ktns mock --parse-json-file "path/to/employees.template.json" --to-json-file ""
  ktns mock --parse-json-file "path/to/employees.template.json" --generate 5 --to-json-file ""
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
			generate, _ := cmd.Flags().GetInt("generate")
			toJsonStdout, _ := cmd.Flags().GetBool("to-json-stdout")
			toJsonFile, _ := cmd.Flags().GetString("to-json-file")
			toJsonFileSet := cmd.Flags().Changed("to-json-file")
			toCsvFile, _ := cmd.Flags().GetString("to-csv-file")
			toCsvFileSet := cmd.Flags().Changed("to-csv-file")
			debugMode, _ := cmd.Flags().GetBool("debug")

			if list {
				mocker := mocker.New()
				mocker.CobraList(opts.Out)
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

			// Validate --parse-str incompatibility with output flags
			if runningParseStr && (toJsonStdout || toJsonFileSet || toCsvFileSet) {
				return fmt.Errorf("--parse-str always outputs to stdout; --to-json-stdout, --to-json-file and --to-csv-file are not available with --parse-str")
			}

			// Validate at least one output flag when using --parse-json or --parse-json-file
			if !runningParseStr && !toJsonStdout && !toJsonFileSet && !toCsvFileSet {
				return fmt.Errorf("--parse-json and --parse-json-file require at least one output flag: --to-json-stdout, --to-json-file or --to-csv-file")
			}

			if runningParseStr {
				// Process the string
				mocker := mocker.New()
				mockedStr, err := processMFStr(parseStr, mocker)
				if err != nil {
					return err
				}

				// Print the mocked string to STDOUT
				fmt.Fprintf(opts.Out, "%s\n", mockedStr)
				return nil
			}

			var jsonPath, csvPath string
			if toJsonFileSet || toCsvFileSet {
				var err error
				jsonPath, csvPath, err = resolveOutputFilePaths(parseJsonFileSet, parseJsonFile, toJsonFileSet, toJsonFile, toCsvFileSet, toCsvFile)
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

			usedWorkers := 1
			availableWorkers := runtime.NumCPU()
			estimatedWork := estimateMFJson(rawJson, generate)

			stats := &DebugStats{
				CoresAvailable:  availableWorkers,
				CoresUsed:       usedWorkers,
				TotalWeight:     estimatedWork,
				WeightPerWorker: estimatedWork / int64(usedWorkers),
				EstMemPeakBytes: 0,
				StartTime:       time.Now(),
			}

			if debugMode {
				debugStop := startDebugRoutine(ctx, stats, os.Stderr)
				defer debugStop()
			}

			mocker := mocker.New()

			processedJsonArr := make([]map[string]any, generate)
			for i := range generate {
				cpRawJson := deepcopy.Copy(rawJson).(map[string]any)
				if err := processMFJson(cpRawJson, mocker, stats); err != nil {
					return fmt.Errorf("%w", err)
				}
				processedJsonArr[i] = cpRawJson
			}

			for i := range processedJsonArr {
				sanitizeGeneratedJson(processedJsonArr[i])
			}

			var outWriter io.Writer
			if toJsonStdout {
				outWriter = opts.Out
			}
			if err := routeOutput(processedJsonArr, generate, jsonPath, csvPath, outWriter); err != nil {
				return fmt.Errorf("%w", err)
			}

			return nil
		},
	}

	mockCmd.Flags().Bool("list", false, "list all available mock functions")
	mockCmd.Flags().String("parse-str", "", "pass a string to be parsed. The mock data will be generated based on this provided string")
	mockCmd.Flags().String("parse-json", "", "pass a JSON object as a string. The mock data will be generated based on this provided json object")
	mockCmd.Flags().String("parse-json-file", "", "pass a path to a single .template.json file. The mock data will be generated based on this file")
	mockCmd.Flags().Int("generate", 1, "pass the desired amount of root objects that will be generated (only available for --parse-json and --parse-json-file)")
	mockCmd.Flags().Bool("to-json-stdout", false, "output result as JSON to stdout")
	mockCmd.Flags().String("to-json-file", "", "output result as JSON to a file; optional filename argument")
	mockCmd.Flags().String("to-csv-file", "", "output result as CSV to a file; optional filename argument")
	mockCmd.Flags().Bool("debug", false, "show live generation progress on stderr (opt-in, only active when worker pool is used)")

	// Configure cobra ouput streams to use the custom 'Out'
	mockCmd.SetOut(opts.Out)

	return mockCmd
}

// processMFStr process mock functions from a string. The function looks for patterns in the format {{ functionName:{arg1}:{arg2}:{argN} }}.
func processMFStr(parseStr string, mocker *mocker.Mock) (string, error) {
	var parsedStr strings.Builder
	s := parseStr
	for {
		// Find the next opening "{{"
		start := strings.Index(s, "{{")
		if start == -1 {
			parsedStr.WriteString(s)
			break
		}
		// Write everything before the opening "{{"
		parsedStr.WriteString(s[:start])
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
			parsedStr.WriteString("{{")
			parsedStr.WriteString(s)
			break
		}

		inner := strings.TrimSpace(s[:end])
		s = s[end+2:] // skip past "}}"

		functionName, params, err := parseMockFunction(inner)
		if err != nil {
			return "", fmt.Errorf("%w", err)
		} else {
			mockValue, err := mocker.Generate(functionName, params)
			if err != nil {
				return "", fmt.Errorf("%w", err)
			} else {
				parsedStr.WriteString(mockValue)
			}
		}
	}
	return parsedStr.String(), nil
}

// estimateMFJson walks the parseMap to find all [n] patterns and estimates the total work
// by calculating the weight of the nodes found. The weight is based on the number of generations
// to be made (including nested generations) and the number of items generated at each level.
func estimateMFJson(parseMap map[string]any, generate int) int64 {
	var traverse func(map[string]any) int64

	traverse = func(rawJson map[string]any) int64 {
		var totalWeight int64 = 0
		for key, val := range rawJson {
			weight := int64(0)
			switch typedValue := val.(type) {
			case string:
				weight = 1
			case map[string]any:
				weight = traverse(typedValue)
			case []any:
				for _, item := range typedValue {
					if _, ok := item.(string); ok {
						weight += 1
					} else if itemMap, ok := item.(map[string]any); ok {
						weight += traverse(itemMap)
					}
				}
			}

			inBracketsMatches := bracketNumberRegex.FindStringSubmatch(key)
			if len(inBracketsMatches) == 2 {
				count, err := strconv.Atoi(inBracketsMatches[1])
				if err == nil && count > 0 {
					weight *= int64(count)
				}
			}
			totalWeight += weight
		}
		return totalWeight
	}

	return traverse(parseMap) * int64(generate)
}

// processMFJson iterates through the parsed json map and processes each value.
// It replaces string values with generated mock data based on the function name and parameters.
// It handles nested maps and arrays of strings or maps.
// Returns an error if any value is not a string or map.
func processMFJson(parseMap map[string]any, mocker *mocker.Mock, stats *DebugStats) error {
	objKeys := make([]string, 0, len(parseMap))
	for key := range parseMap {
		objKeys = append(objKeys, key)
	}

	for keyIndex := 0; keyIndex < len(objKeys); {
		objKey := objKeys[keyIndex]
		switch typedValue := parseMap[objKey].(type) {
		case string:
			// try to find [n] in the "key"
			generateAmount, err := extractDigitInBrackets("object", objKey)
			if err != nil {
				return err
			}
			// try to find the mock function in the "value"
			interpretedValue, isMockFunction := parseDynamicValue(typedValue)
			// if it's not a mock function, just replace the value
			if !isMockFunction {
				parseMap[objKey] = interpretedValue
				keyIndex++
				if stats != nil {
					stats.Generated.Add(1)
				}
				continue
			}
			// if it's a mock function, extract the function name and parameters
			functionName, params, err := parseMockFunction(interpretedValue)
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
			if stats != nil {
				stats.Generated.Add(int64(generateAmount))
			}
		case map[string]any:
			// try to find [n] in the "key"
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
				if err := processMFJson(typedValue, mocker, stats); err != nil {
					return err
				}
				keyIndex++
			}
		case []any:
			for itemKey, item := range typedValue {
				if itemStr, ok := item.(string); ok {
					interpretedValue, isMockFunction := parseDynamicValue(itemStr)
					if !isMockFunction {
						typedValue[itemKey] = interpretedValue
						if stats != nil {
							stats.Generated.Add(1)
						}
						continue
					}
					functionName, params, err := parseMockFunction(interpretedValue)
					if err != nil {
						return err
					}
					mockValue, err := mocker.Generate(functionName, params)
					if err != nil {
						return err
					}
					typedValue[itemKey] = mockValue
					if stats != nil {
						stats.Generated.Add(1)
					}
				} else if itemMap, ok := item.(map[string]any); ok {
					err := processMFJson(itemMap, mocker, stats)
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

// sanitizeGeneratedJson iterates through the parsed json map and sanitizes the keys by removing segments between bracketes (e.g. [digits]).
// It handles nested maps.
func sanitizeGeneratedJson(parseMap map[string]any) {
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
			sanitizeGeneratedJson(mapValue)
		}

		// Recurse into array elements that are maps
		if sliceValue, ok := objValue.([]any); ok {
			for _, elem := range sliceValue {
				if elemMap, ok := elem.(map[string]any); ok {
					sanitizeGeneratedJson(elemMap)
				}
			}
		}

		if sanitizedKey != objKey {
			parseMap[sanitizedKey] = objValue
			delete(parseMap, objKey)
		}
	}
}

// parseDynamicValue interprets a string value, checking if it contains a mock function between {{ }}.
// If it does, it returns the function name and true.
// If not, it returns the original string and false.
func parseDynamicValue(rawValue string) (string, bool) {
	if rawValue == "" {
		return "", false
	}

	matches := parseDynamicValueRegex.FindStringSubmatch(rawValue)

	if len(matches) > 0 {
		return strings.TrimSpace(matches[1]), true
	}
	return rawValue, false
}

// parseMockFunction splits a raw string of format "func:{arg1}:{arg2}:...".
// It handles regex args wrapped with slashes (/.../) and value args wrapped with curly
// braces ({...}) to avoid splitting inside them. The colon (:) outside both delimiters
// acts as the separator between positional parameters.
// Returns: function name, slice of parameter strings, and an error if any parameter
// after the function name is not wrapped in {…} or /…/.
func parseMockFunction(rawValue string) (string, []string, error) {
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

// extractDigitInBrackets extracts a digit from a string in the format "content[<digit>]".
// Only handles "object" place values. Returns an error immediately if place != "object".
// If the string doesn't contain brackets, it returns 1.
func extractDigitInBrackets(place string, str string) (int, error) {
	if place != "object" {
		return 0, fmt.Errorf("invalid value '%s' (must be 'object')", place)
	}

	matches := bracketNumberRegex.FindStringSubmatch(str)

	if len(matches) != 2 {
		if !bracketCharRegex.MatchString(str) {
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

// sanitizeKeyWithBrackets removes the segment of a string between brackets, including the brackets themselves.
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

// marshalAsCSV serialises a slice of flat maps to CSV (standard RFC 4180).
// Column order is determined by the sorted union of all keys across all rows.
// Values are always coerced to strings.
// Nested objects/arrays will be serialised as their Go string representation.
func marshalAsCSV(parseMaps []map[string]any) (string, error) {
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

// workingDir returns the current working directory.
// Used to resolve the default output path for --to-json-file and --to-csv-file
// when no explicit filename is provided and the source is not --parse-json-file.
func workingDir() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("could not determine working directory: %w", err)
	}
	return dir, nil
}

// atomicFileCreate opens a temp file in the same directory as finalPath.
// The caller writes to the returned *os.File, then calls commit() on success
// or abort() on failure. commit() renames the temp to finalPath atomically.
// abort() deletes the temp file.
func atomicFileCreate(finalPath string) (f *os.File, commit func() error, abort func(), err error) {
	dir := filepath.Dir(finalPath)
	f, err = os.CreateTemp(dir, ".ktns-tmp-*")
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to create temp file in '%s': %w", dir, err)
	}
	commit = func() error {
		if cerr := f.Close(); cerr != nil {
			return cerr
		}
		return os.Rename(f.Name(), finalPath)
	}
	abort = func() {
		_ = f.Close()
		_ = os.Remove(f.Name())
	}
	return f, commit, abort, nil
}

// resolveOutputFilePaths resolves the final JSON and CSV output file paths from flags and defaults.
// Returns empty strings for paths whose corresponding flag was not set.
func resolveOutputFilePaths(
	parseJsonFileSet bool,
	parseJsonFile string,
	toJsonFileSet bool,
	toJsonFileValue string,
	toCsvFileSet bool,
	toCsvFileValue string,
) (jsonPath string, csvPath string, err error) {
	if toJsonFileSet {
		switch {
		// Specific filename provided with --to-json-file
		case toJsonFileValue != "":
			jsonPath = toJsonFileValue
		// Default filename based on --parse-json-file template name
		case parseJsonFileSet && parseJsonFile != "":
			defaultFileName := strings.Replace(filepath.Base(parseJsonFile), ".template.json", ".json", 1)
			defaultFilePath := filepath.Join(filepath.Dir(parseJsonFile), defaultFileName)
			jsonPath = defaultFilePath
		// Default filename in current working directory `output.json`
		default:
			dir, e := workingDir()
			if e != nil {
				return "", "", e
			}
			jsonPath = filepath.Join(dir, "output.json")
		}
	}

	if toCsvFileSet {
		switch {
		// Specific filename provided with --to-csv-file
		case toCsvFileValue != "":
			csvPath = toCsvFileValue
		// Default filename based on --parse-json-file template name
		case parseJsonFileSet && parseJsonFile != "":
			defaultCsvFileName := strings.Replace(filepath.Base(parseJsonFile), ".template.json", ".csv", 1)
			defaultCsvFilePath := filepath.Join(filepath.Dir(parseJsonFile), defaultCsvFileName)
			csvPath = defaultCsvFilePath
		// Default filename in current working directory `output.csv`
		default:
			dir, e := workingDir()
			if e != nil {
				return "", "", e
			}
			csvPath = filepath.Join(dir, "output.csv")
		}
	}

	return jsonPath, csvPath, nil
}

// routeOutput handles all output routing for --parse-json and --parse-json-file results.
// processedJsonArr is the slice of processed root objects.
// generate is the --generate count (used to decide single-object vs array).
// jsonFilePath is the resolved JSON output file path.
// csvFilePath is the resolved CSV output file path.
// out is the io.Writer for stdout output.
func routeOutput(
	processedJsonArr []map[string]any,
	generate int,
	jsonFilePath string,
	csvFilePath string,
	out io.Writer,
) error {
	// Determine the data shape
	var data any
	if generate == 1 {
		data = processedJsonArr[0]
	} else {
		data = processedJsonArr
	}

	// Stdout output
	if out != nil {
		jsonBytes, err := json.Marshal(data)
		if err != nil {
			return fmt.Errorf("error marshalling JSON for stdout: %w", err)
		}
		fmt.Fprintf(out, "%s\n", string(jsonBytes))
	}

	// Json file output
	if jsonFilePath != "" {
		jsonBytes, err := json.Marshal(data)
		if err != nil {
			return fmt.Errorf("error marshalling JSON for file: %w", err)
		}

		f, commit, abort, err := atomicFileCreate(jsonFilePath)
		if err != nil {
			return err
		}
		if _, err = f.Write(jsonBytes); err != nil {
			abort()
			return fmt.Errorf("failed to write result to '%s': %w", jsonFilePath, err)
		}
		if err = commit(); err != nil {
			return fmt.Errorf("failed to commit JSON file '%s': %w", jsonFilePath, err)
		}
	}

	// CSV file output
	if csvFilePath != "" {
		f, commit, abort, err := atomicFileCreate(csvFilePath)
		if err != nil {
			return err
		}
		csvStr, err := marshalAsCSV(processedJsonArr)
		if err != nil {
			return err
		}
		if _, err = f.Write([]byte(csvStr + "\n")); err != nil {
			abort()
			return fmt.Errorf("failed to write CSV result to '%s': %w", csvFilePath, err)
		}
		if err = commit(); err != nil {
			return fmt.Errorf("failed to commit CSV file '%s': %w", csvFilePath, err)
		}
	}
	return nil
}

// startDebugRoutine launches a goroutine that prints live progress to out (always
// os.Stderr in production) at 200ms intervals. Call the returned stop function to
// terminate the display and print a final newline.
func startDebugRoutine(ctx context.Context, stats *DebugStats, out io.Writer) (stop func()) {
	innerCtx, cancel := context.WithCancel(ctx)

	done := make(chan struct{})
	render := func(final bool) {
		elapsed := time.Since(stats.StartTime).Seconds()
		gen := stats.Generated.Load()
		bytesWrittenToJson := stats.BytesWrittenToJson.Load()
		bytesWrittenToCsv := stats.BytesWrittenToCsv.Load()
		percentDone := 0.0
		if stats.TotalWeight > 0 {
			percentDone = float64(gen) / float64(stats.TotalWeight) * 100
		}
		estMemPeakStr := formatSizeMetrics(stats.EstMemPeakBytes)

		// Create the debug output string with all relevant stats, '\r' at the start to overwrite the previous line, making it look like a live-updating single line of output
		strOutput := fmt.Sprintf("\r[debug] Workers: %d/%d | PeakMem: ~%s | Progress: %s / %s (%.1f%%) | Elapsed: %s",
			stats.CoresUsed, stats.CoresAvailable, estMemPeakStr, formatNumberMetrics(gen), formatNumberMetrics(stats.TotalWeight), percentDone, formatDurationMetrics(elapsed))

		if bytesWrittenToJson > 0 {
			strOutput += fmt.Sprintf(" | JSON File: ~%s", formatSizeMetrics(bytesWrittenToJson))
		}
		if bytesWrittenToCsv > 0 {
			strOutput += fmt.Sprintf(" | CSV File: ~%s", formatSizeMetrics(bytesWrittenToCsv))
		}
		fmt.Fprint(out, strOutput)

		if final {
			fmt.Fprintln(out)
		}
	}

	go func() {
		defer close(done)
		ticker := time.NewTicker(200 * time.Millisecond)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				render(false)
			case <-innerCtx.Done():
				render(true)
				return
			}
		}
	}()

	return func() {
		cancel()
		<-done
	}
}
