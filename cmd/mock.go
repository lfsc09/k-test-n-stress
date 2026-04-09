package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/lfsc09/k-test-n-stress/mocker"
	"github.com/mohae/deepcopy"
	"github.com/spf13/cobra"
	"github.com/vbauerster/mpb/v8"
	"github.com/vbauerster/mpb/v8/decor"
)

var objKeyNumberRegex = regexp.MustCompile(`^[^\[\]\s]+\[(\d+)\]$`)

func NewMockCmd(opts *CommandOptions) *cobra.Command {
	mockCmd := &cobra.Command{
		Use:   "mock",
		Short: "Generate mock data based from an object string or from template files",
		Long: `Generate mock data based on --parse-str, --parse-json or --parse-json-file options.

Mock functions:

* List available mock functions with --list.
* Always call the mock function with the format {{ functionName:{arg1}:{arg2}:{argN} }}. (Values not wrapped in double curly braces will be considered raw values)
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

* When using --parse-json-file, the output file is written alongside the template file.
  e.g.: "path/to/employees.template.json" → "path/to/employees.json"

Examples:
  ktns mock --parse-str '{{ Person.name }}'
  ktns mock --parse-str 'Hello my name is {{ Person.name }}, I am {{ Number.number:{0}:{1}:{100} }} years old'
  ktns mock --parse-json '{ "name": "{{ Person.name }}", "age": "{{ Number.number:{0}:{1}:{100} }}" }'
  ktns mock --parse-json '{ "phones[2]": "{{ Person.phoneNumber }}" }' --generate 5
  ktns mock --parse-json-file "path/to/employees.template.json"
  ktns mock --parse-json-file "path/to/employees.template.json" --generate 5
	`,
		RunE: func(cmd *cobra.Command, args []string) error {
			list, _ := cmd.Flags().GetBool("list")
			parseStr, _ := cmd.Flags().GetString("parse-str")
			parseJson, _ := cmd.Flags().GetString("parse-json")
			parseJsonFile, _ := cmd.Flags().GetString("parse-json-file")
			generate, _ := cmd.Flags().GetInt("generate")

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

			mpbHandler := mpb.New(
				mpb.WithWidth(60),
				mpb.WithOutput(os.Stdout),
				mpb.WithAutoRefresh(),
			)

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

				// Progress bar: total = generate (one tick per root object produced)
				bar := giveMeABar("parse-json", int64(generate), nil, mpbHandler)

				// Process the parsed map
				mocker := mocker.New()
				parseMaps := make([]map[string]any, generate)
				for i := range generate {
					cpParseMap := deepcopy.Copy(parseMap).(map[string]any)
					if err := processJsonMap(cpParseMap, mocker); err != nil {
						bar.Abort(false)
						return fmt.Errorf("%w", err)
					}
					parseMaps[i] = deepcopy.Copy(cpParseMap).(map[string]any)
					bar.Increment()
				}

				// Sanitize the parsed map
				for i := range parseMaps {
					sanitizeJsonMap(parseMaps[i])
				}

				// Print the result to stdout
				var prettyJSON []byte
				var err error
				if generate == 1 {
					prettyJSON, err = json.MarshalIndent(parseMaps[0], "", "  ")
				} else {
					prettyJSON, err = json.MarshalIndent(parseMaps, "", "  ")
				}
				if err != nil {
					return fmt.Errorf("error marshalling JSON '%w'", err)
				}
				fmt.Fprintf(opts.Out, "%s\n", prettyJSON)
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

				// Determine output file path: same directory as the template file, .template.json → .json
				outName := strings.Replace(filepath.Base(parseJsonFile), ".template.json", ".json", 1)
				outPath := filepath.Join(filepath.Dir(parseJsonFile), outName)

				// Always delete the output file before writing a new one
				_ = os.Remove(outPath)

				// Progress bar: total = generate (one tick per root object produced); outPath for file-size display
				bar := giveMeABar(filepath.Base(parseJsonFile), int64(generate), &outPath, mpbHandler)

				// Process the parsed map
				mocker := mocker.New()
				parseMaps := make([]map[string]any, generate)
				for i := range generate {
					cpParseMap := deepcopy.Copy(parseMap).(map[string]any)
					if err := processJsonMap(cpParseMap, mocker); err != nil {
						bar.Abort(false)
						return fmt.Errorf("%w", err)
					}
					parseMaps[i] = deepcopy.Copy(cpParseMap).(map[string]any)
					bar.Increment()
				}

				// Sanitize the parsed maps
				for i := range parseMaps {
					sanitizeJsonMap(parseMaps[i])
				}

				// Marshal the result
				var prettyJSON []byte
				if generate == 1 {
					prettyJSON, err = json.MarshalIndent(parseMaps[0], "", "  ")
				} else {
					prettyJSON, err = json.MarshalIndent(parseMaps, "", "  ")
				}
				if err != nil {
					return fmt.Errorf("error marshalling JSON '%w'", err)
				}

				// Write the output file
				if err = os.WriteFile(outPath, prettyJSON, 0644); err != nil {
					return fmt.Errorf("failed to write result to '%s': '%w'", outPath, err)
				}
			}

			mpbHandler.Wait()

			return nil
		},
	}

	mockCmd.Flags().Bool("list", false, "list all available mock functions")
	mockCmd.Flags().String("parse-str", "", "pass a string to be parsed. The mock data will be generated based on this provided string")
	mockCmd.Flags().String("parse-json", "", "pass a JSON object as a string. The mock data will be generated based on this provided json object")
	mockCmd.Flags().String("parse-json-file", "", "pass a path to a single .template.json file. The mock data will be generated based on this file and written alongside it")
	mockCmd.Flags().Int("generate", 1, "pass the desired amount of root objects that will be generated (only available for --parse-json and --parse-json-file)")

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

// Creates and returns a progress bar for a given task, with the total number of ticks and an optional output file path for size display.
// The bar displays the task name, progress counters, elapsed time, and output file size (if provided).
// The elapsed time is updated dynamically as the task progresses, and the file size is displayed if an output path is provided and the file exists.
// The function uses the mpb library to create and manage the progress bar, and it returns the created bar for further updates.
// The progress bar is configured to automatically refresh and display the relevant information in a clear format, making it easy to track the progress of tasks.
// The function also handles the case where the output file may not exist yet, displaying "N/A" for the file size until the file is created and can be measured.
func giveMeABar(taskName string, total int64, outPath *string, mpbHandler *mpb.Progress) *mpb.Bar {
	startElapsedTime := time.Now()
	var elapsedTime time.Duration
	bar := mpbHandler.AddBar(total,
		mpb.PrependDecorators(
			decor.Name(taskName, decor.WCSyncWidthR),
			decor.CountersNoUnit(" %d/%d ", decor.WCSyncWidthR),
		),
		mpb.AppendDecorators(
			decor.Any(func(s decor.Statistics) string {
				if !s.Completed {
					elapsedTime = time.Since(startElapsedTime)
				}
				return formatDurationMetrics(elapsedTime)
			}, decor.WCSyncWidth),
			decor.Any(func(s decor.Statistics) string {
				if outPath == nil {
					return " [N/A] "
				}
				info, err := os.Stat(*outPath)
				if err != nil {
					return " [N/A] "
				}
				return formatSizeMetrics(info.Size())
			}, decor.WCSyncWidth),
		),
	)
	return bar
}
