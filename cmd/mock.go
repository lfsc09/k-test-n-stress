package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/lfsc09/k-test-n-stress/mocker"
	"github.com/mohae/deepcopy"
	"github.com/spf13/cobra"
	"github.com/vbauerster/mpb/v8"
	"github.com/vbauerster/mpb/v8/decor"
)

var filenameNumberRegex = regexp.MustCompile(`^[^\[\]\s]+\[(\d+)\]\.template\.json$`)
var objKeyNumberRegex = regexp.MustCompile(`^[^\[\]\s]+\[(\d+)\]$`)

func NewMockCmd(opts *CommandOptions) *cobra.Command {
	mockCmd := &cobra.Command{
		Use:   "mock",
		Short: "Generate mock data based from an object string or from template files",
		Long: `Generate mock data based on --parse-str, --parse-json or --parse-files options.
	
* Add --preserve-folder-structure to keep the folder structure of the input files. (Only works with --parse-files)

  e.g.: Having a template folder structure like this:

  ├── company.template.json
  └── assets/
    ├── employee[10].template.json
    └── building[2].template.json

  Will result in mocked results in the same structure.

  ├── company.json
  └── assets/
    ├── employee[10].json
    └── building[2].json

Mock functions:

* List available mock functions with --list.
* Always call the mock function with the format {{ functionName::arg1:arg2:... }}. (Values not wrapped in double brackets will be considered raw values)

Controling the number of generated data:

* Add --generate to specify the number of root objects to generate. (Only works with --parse-json)
* When using --parse-files, specify the desired number of root objects in the template file's name, between brackets.

  e.g.: A template file named "employees[5].template.json" will generate an array of 5 employees.

  Template:
  {
    "name": "{{ Person.name }}"
  }

  Will generate:
  [
    { "name": ... },
    { "name": ... },
    { "name": ... },
    { "name": ... },
    { "name": ... }
  ]

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

  Will generate an array of 5 employees with random names.

  {
    "phones": [ "...", "...", "...", "...", "..." ]
  }

Examples:
  ktns mock --parse-str '{{ Person.name }}'
  ktns mock --parse-str 'Hello my name is {{ Person.name }}, I am {{ Number.number::1:100 }} years old'
  ktns mock --parse-json '{ "name": "{{ Person.name }}", "age": "{{ Number.number::1:100 }}" }'
  ktns mock --parse-json '{ "phones[2]": "{{ Person.phoneNumber }}" }' --generate 5
  ktns mock --parse-files "*.template.json"
  ktns mock --parse-files "test/templates/*.template.json"
  ktns mock --parse-files "test/templates" --preserve-folder-structure
	`,
		RunE: func(cmd *cobra.Command, args []string) error {
			list, _ := cmd.Flags().GetBool("list")
			parseStr, _ := cmd.Flags().GetString("parse-str")
			parseJson, _ := cmd.Flags().GetString("parse-json")
			parseFiles, _ := cmd.Flags().GetString("parse-files")
			preserveFolderStructure, _ := cmd.Flags().GetBool("preserve-folder-structure")
			generate, _ := cmd.Flags().GetInt("generate")

			if list {
				mocker := mocker.New()
				mocker.List(opts.Out)
				return nil
			}

			runningParseStr, runningParseJson, runningParseFiles := false, false, false

			// Check if --parse-json, --parse-files or --parse-str is provided
			parseCheck := 0
			if parseStr != "" {
				parseCheck++
				runningParseStr = true
			}
			if parseJson != "" {
				parseCheck++
				runningParseJson = true
			}
			if parseFiles != "" {
				parseCheck++
				runningParseFiles = true
			}
			if parseCheck == 0 {
				return fmt.Errorf("nothing to be parsed, ask for help -h or --help")
			} else if parseCheck > 1 {
				return fmt.Errorf("provide only one of the three options: --parse-json, --parse-files or --parse-str")
			}

			if runningParseFiles && len(args) > 0 {
				return fmt.Errorf("you passed multiple files to --parse-files without quotes. Did you mean: --parse-files \"*.template.json\"?")
			}

			if preserveFolderStructure && !runningParseFiles {
				return fmt.Errorf("--preserve-folder-structure option is only available when using --parse-files")
			}

			if generate > 1 && !runningParseJson {
				return fmt.Errorf("--generate option is only available when using --parse-json")
			}

			if generate <= 0 {
				return fmt.Errorf("--generate option must be greater than 0")
			}

			// Clean previous output directory
			if err := os.RemoveAll("out"); err != nil {
				return fmt.Errorf("failed to remove previous output directory '%w'", err)
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
				outPath := ""
				bar := giveMeABar("CLI", &outPath, 4, mpbHandler)

				// Parse the string object content (STEP)
				var parseMap map[string]any
				if err := json.Unmarshal([]byte(parseJson), &parseMap); err != nil {
					return fmt.Errorf("failed to parse JSON from the provided --parse-json '%w'", err)
				}
				bar.Increment()

				// Process the parsed map (STEP)
				mocker := mocker.New()
				parseMaps := make([]map[string]any, generate)
				for i := range generate {
					cpParseMap := deepcopy.Copy(parseMap).(map[string]any)
					if err := processJsonMap(cpParseMap, mocker); err != nil {
						return fmt.Errorf("%w", err)
					}
					parseMaps[i] = deepcopy.Copy(cpParseMap).(map[string]any)
				}
				bar.Increment()

				// Sanitize the parsed map (STEP)
				for i := range parseMaps {
					sanitizeJsonMap(parseMaps[i])
				}
				bar.Increment()

				// Write the processed map to a file (STEP)
				var mu sync.Mutex
				createdDirs := make(map[string]bool, 1)
				if err := toFile(false, "mocked-data.json", &outPath, "", &parseMaps, &mu, &createdDirs); err != nil {
					return fmt.Errorf("%w", err)
				}
				bar.Increment()
			}

			// Parse object from `--parse-files` files
			if runningParseFiles {
				foundTemplateFiles, err := findTemplateFiles(parseFiles)
				if err != nil {
					return fmt.Errorf("failed to find template files from the provided --parse-files '%w'", err)
				}
				if len(foundTemplateFiles) == 0 {
					return fmt.Errorf("no template files found in the provided --parse-files '%s'", parseFiles)
				}

				var wg sync.WaitGroup
				var mu sync.Mutex
				createdDirs := make(map[string]bool)
				for _, inPath := range foundTemplateFiles {
					wg.Add(1)
					go func(inPath string) error {
						defer wg.Done()
						outPath := ""
						bar := giveMeABar(inPath, &outPath, 5, mpbHandler)

						// Read the template file (STEP)
						templateFileContent, err := os.ReadFile(inPath)
						if err != nil {
							bar.Abort(false)
							return fmt.Errorf("failed to read --parse-file '%w'", err)
						}
						generate, err := extractDigitInBrackets("file", inPath)
						if err != nil {
							bar.Abort(false)
							return fmt.Errorf("failed to extract [digit] from '%w'", err)
						}
						bar.Increment()

						// Parse the template file content (STEP)
						var parseMap map[string]any
						if err = json.Unmarshal(templateFileContent, &parseMap); err != nil {
							bar.Abort(false)
							return fmt.Errorf("failed to parse JSON from the provided --parse-file '%w'", err)
						}
						bar.Increment()

						// Process the parsed map (STEP)
						mocker := mocker.New()
						parseMaps := make([]map[string]any, generate)
						for i := range generate {
							cpParseMap := deepcopy.Copy(parseMap).(map[string]any)
							if err := processJsonMap(cpParseMap, mocker); err != nil {
								bar.Abort(false)
								return fmt.Errorf("%w", err)
							}
							parseMaps[i] = deepcopy.Copy(cpParseMap).(map[string]any)
						}
						bar.Increment()

						// Sanitize the parsed map (STEP)
						for i := range parseMaps {
							sanitizeJsonMap(parseMaps[i])
						}
						bar.Increment()

						// Write the processed map to a file (STEP)
						if err := toFile(preserveFolderStructure, inPath, &outPath, parseFiles, &parseMaps, &mu, &createdDirs); err != nil {
							bar.Abort(false)
							return fmt.Errorf("%w", err)
						}
						bar.Increment()

						return nil
					}(inPath)
				}
				wg.Wait()
			}

			mpbHandler.Wait()

			return nil
		},
	}

	mockCmd.Flags().Bool("list", false, "list all available mock functions")
	mockCmd.Flags().String("parse-str", "", "pass a string to be parsed. The mock data will be generated based on this provided string")
	mockCmd.Flags().String("parse-json", "", "pass a JSON object as a string. The mock data will be generated based on this provided json object")
	mockCmd.Flags().String("parse-files", "", "pass a path, directory, or glob pattern to find template files. The mock data will be generated based on the found template files")
	mockCmd.Flags().Bool("preserve-folder-structure", false, "if set, the folder structure of the input files will be preserved in the output files (only available for --parse-file)")
	mockCmd.Flags().Int("generate", 1, "pass the desired amount of root objects that will be generated (only available for --parse-json)")

	// Configure cobra ouput streams to use the custom 'Out'
	mockCmd.SetOut(opts.Out)

	return mockCmd
}

// Splits a raw string of format "func:arg1:arg2:...".
// It handles regex args wrapped with slashes (/.../) to avoid splitting inside them.
// Returns: function name, and slice of parameter strings.
func extractMockMethod(rawValue string) (string, []string) {
	if rawValue == "" {
		return "", nil
	}
	var parts []string
	var buf strings.Builder
	inRegex := false

	trimmed := strings.TrimSpace(rawValue)

	for _, char := range trimmed {
		if char == '/' {
			inRegex = !inRegex
			// Always include slash
			buf.WriteByte(byte(char))
			continue
		}
		// If ':' outside regex — treat as delimiter
		if char == ':' && !inRegex {
			parts = append(parts, buf.String())
			// Start building next segment
			buf.Reset()
			continue
		}
		// Default: build the current token
		buf.WriteByte(byte(char))
	}

	// Add the final piece (there's no trailing `:`)
	if buf.Len() > 0 {
		parts = append(parts, buf.String())
	}

	return parts[0], parts[1:]
}

// Interprets a string value, checking if it contains a mock function between {{ }}.
// If it does, it returns the function name and true.
// If not, it returns the original string and false.
func interpretString(rawValue string) (string, bool) {
	if rawValue == "" {
		return "", false
	}

	re := regexp.MustCompile(`^\s*{{\s*(.*?)\s*}}\s*$`)
	matches := re.FindStringSubmatch(rawValue)

	if len(matches) > 0 {
		return matches[1], true
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
			functionName, params := extractMockMethod(interpretedValue)
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
					functionName, params := extractMockMethod(interpretedValue)
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
func processStr(parseStr string, mocker *mocker.Mock) string {
	dBracketsPatterns := regexp.MustCompile(`{{\s*([^}]+?)\s*}}`)

	all := dBracketsPatterns.ReplaceAllStringFunc(parseStr, func(match string) string {
		interpretedValue := dBracketsPatterns.FindStringSubmatch(match)[1]
		interpretedValue = strings.TrimSpace(interpretedValue)

		functionName, params := extractMockMethod(interpretedValue)

		mockValue, err := mocker.Generate(functionName, params)
		if err != nil {
			return fmt.Sprintf("[%v]", err)
		}
		return mockValue
	})

	return all
}

// Extracts a digit from a string in the format "content[<digit>]" or "content[<digit>].template.json".
// If the string doesn't contain brackets, it returns 1.
func extractDigitInBrackets(place string, str string) (int, error) {
	var matches []string
	if place == "file" {
		matches = filenameNumberRegex.FindStringSubmatch(str)
	} else if place == "object" {
		matches = objKeyNumberRegex.FindStringSubmatch(str)
	} else {
		return 0, fmt.Errorf("invalid value '%s' (must be either 'file' or 'object')", place)
	}

	if len(matches) != 2 {
		if !regexp.MustCompile(`[\[\]]`).MatchString(str) {
			return 1, nil
		}
		if place == "file" {
			return 0, fmt.Errorf("invalid format '%s' (must be 'text[digit].template.json')", str)
		} else if place == "object" {
			return 0, fmt.Errorf("invalid format '%s' (must be 'text[digit]')", str)
		}
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

// Returns all *.template.json files from a path, directory, or glob.
// It's recursive for directories, and respects any wildcard pattern.
func findTemplateFiles(input string) ([]string, error) {
	var matchedFiles []string
	info, err := os.Stat(input)

	// Check if input exists and is a directory — if so, walk recursively
	if err == nil && info.IsDir() {
		err := filepath.Walk(input, func(path string, fi os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if !fi.IsDir() && strings.HasSuffix(path, ".template.json") {
				matchedFiles = append(matchedFiles, path)
			}
			return nil
		})
		if err != nil {
			return nil, fmt.Errorf("error walking through directory '%w'", err)
		}
		return matchedFiles, nil
	}

	// If it's not a directory, use filepath.Glob to match pattern (may include wildcard)
	globMatches, err := filepath.Glob(input)
	if err != nil {
		return nil, fmt.Errorf("error matching pattern '%w'", err)
	}

	for _, file := range globMatches {
		info, err := os.Stat(file)
		if err != nil {
			continue
		}
		if !info.IsDir() && strings.HasSuffix(file, ".template.json") {
			matchedFiles = append(matchedFiles, file)
		}
	}

	return matchedFiles, nil
}

// Writes the generated mock data to a file.
// It creates the directory structure if it doesn't exist.
// If `preserve-folder-structure` is true, it keeps the original folder structure.
func toFile(preserveFolderStructure bool, inPath string, outPath *string, parseFiles string, result *[]map[string]any, mu *sync.Mutex, createdDirs *map[string]bool) error {
	var prettyJSON []byte
	var err error
	if len(*result) == 1 {
		prettyJSON, err = json.MarshalIndent((*result)[0], "", "  ")
	} else {
		prettyJSON, err = json.MarshalIndent(result, "", "  ")
	}
	if err != nil {
		return fmt.Errorf("error marshalling JSON '%w'", err)
	}

	if preserveFolderStructure {
		normalizedParseFrom, err := normalizeParseFrom(parseFiles)
		if err != nil {
			return fmt.Errorf("failed to normalize '--parse-file' path '%w'", err)
		}
		relPath, err := filepath.Rel(normalizedParseFrom, inPath)
		if err != nil {
			return fmt.Errorf("failed to get relative path '%w'", err)
		}
		relPath = strings.Replace(relPath, ".template.json", ".json", 1)
		*outPath = filepath.Join("out", relPath)
	} else {
		outName := strings.Replace(filepath.Base(inPath), ".template.json", ".json", 1)
		*outPath = filepath.Join("out", outName)
	}

	// Any created folders must be Thread-safe
	dir := filepath.Dir(*outPath)
	mu.Lock()
	if !(*createdDirs)[dir] {
		if err := os.MkdirAll(dir, 0755); err != nil {
			mu.Unlock()
			return fmt.Errorf("failed to create directory '%v', '%w'", dir, err)
		}
		(*createdDirs)[dir] = true
	}
	mu.Unlock()

	err = os.WriteFile(*outPath, prettyJSON, 0644)
	if err != nil {
		return fmt.Errorf("failed to write result to '%v', '%w'", outPath, err)
	}
	return nil
}

// Normalizes the input path to a directory.
// If the input is a directory, it returns the directory path.
// If the input is a file or glob pattern, it returns the directory of the file.
func normalizeParseFrom(input string) (string, error) {
	// First check if it's a directory
	info, err := os.Stat(input)
	if err == nil && info.IsDir() {
		return input, nil
	}
	// Otherwise, assume it's a file or glob pattern
	base := filepath.Dir(input)
	return base, nil
}

func giveMeABar(taskName string, outPath *string, steps int64, mpbHandler *mpb.Progress) *mpb.Bar {
	startElapsedTime := time.Now()
	var elapsedTime time.Duration
	bar := mpbHandler.AddBar(steps,
		mpb.PrependDecorators(
			decor.Name(taskName, decor.WCSyncWidthR),
			decor.Any(func(s decor.Statistics) string {
				current := "unknown state"
				if s.Aborted {
					current = "failed"
				} else if s.Current == steps-5 {
					current = "reading"
				} else if s.Current == steps-4 {
					current = "parsing"
				} else if s.Current == steps-3 {
					current = "processing"
				} else if s.Current == steps-2 {
					current = "sanetizing"
				} else if s.Current == steps-1 {
					current = "writing"
				} else if s.Completed {
					current = "done"
				}
				return fmt.Sprintf("   %s   ", current)
			}, decor.WCSyncWidth),
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
