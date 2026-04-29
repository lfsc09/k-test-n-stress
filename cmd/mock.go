package cmd

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/signal"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/lfsc09/k-test-n-stress/mocker"
	"github.com/spf13/cobra"
)

const defaultAtomicTempFilePrefix = ".ktns-tmp"
const defaultBufferSizeBytes uint = 1024 * 1024 // 1 MB

// parseDynamicValueRegex matches a value that is entirely {{ … }}.
var parseDynamicValueRegex *regexp.Regexp = regexp.MustCompile(`^\s*{{(.*)}}\s*$`)

// parseArrayItemBracketNumberRegex matches array item keys with bracketed numbers, e.g. "phones[5]" and captures the key and number.
var parseArrayItemBracketNumberRegex *regexp.Regexp = regexp.MustCompile(`^(.+)\[(\d+)\]$`)

/*
Blueprint Tree definition for JSON template compilation and generation.
*/
type TemplateNodeType int

const (
	NodeObject TemplateNodeType = iota
	NodeArray
	NodeString
)

type TemplateNode struct {
	Type     TemplateNodeType
	Key      string          // Sanitized JSON object key
	Value    string          // JSON object value (Only for NodeString type)
	Children []*TemplateNode // For NodeObject and NodeArray types
	Repeat   uint            // For key[n] syntax (Only for NodeObject and NodeString types)
}

/*
DebugStats for the --debug display.
*/
type DebugStats struct {
	StartTime        time.Time
	GenerateTotal    uint
	Generated        atomic.Uint32
	UsedMemPeakBytes atomic.Uint64
	BytesWritten     atomic.Uint32
}

/*
AtomicFile represents a file being written to with an atomic commit or abort.
The caller should write to the provided *os.File, then call commit() on success or abort() on failure.
*/
type AtomicFile struct {
	Path string
	File *os.File
}

// NewAtomicFile creates an AtomicFile struct for the given finalPath.
// It opens a temp file in the same directory as finalPath.
func NewAtomicFile(finalPath string) (*AtomicFile, error) {
	dir := filepath.Dir(finalPath)
	f, err := os.CreateTemp(dir, defaultAtomicTempFilePrefix+"*")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp file in '%s': %w", dir, err)
	}

	return &AtomicFile{
		Path: finalPath,
		File: f,
	}, nil
}

// Commit closes the file and renames it to the final path. If closing the file fails, it returns an error and does not rename.
func (af *AtomicFile) Commit() error {
	if cerr := af.File.Close(); cerr != nil {
		return cerr
	}
	return os.Rename(af.File.Name(), af.Path)
}

// Abort closes the file and removes the temp file. It ignores any errors since this is meant to be called in a defer after a failed write.
func (af *AtomicFile) Abort() {
	_ = af.File.Close()
	_ = os.Remove(af.File.Name())
}

/*
BufferWriter for buffered writing the output.
*/
type BufferWriter struct {
	atomicFileRef *AtomicFile
	writer        *bufio.Writer
	BufferSize    uint
}

// NewBufferWriter creates a BufferWriter with a bufio.Writer that writes to both stdout and the atomic file (if set).
// The buffer size is fixed to `defaultBufferSizeBytes`.
// Note: MultiWriter writes sequentially to each writer.
func NewBufferWriter(stdoutWriter io.Writer, atomicFile *AtomicFile) *BufferWriter {
	writers := []io.Writer{}
	if stdoutWriter != nil {
		writers = append(writers, stdoutWriter)
	}
	if atomicFile != nil {
		writers = append(writers, atomicFile.File)
	}

	if len(writers) == 0 {
		return nil
	}

	multiWriter := io.MultiWriter(writers...)
	bufferWriter := bufio.NewWriterSize(multiWriter, int(defaultBufferSizeBytes))

	return &BufferWriter{
		BufferSize:    defaultBufferSizeBytes,
		atomicFileRef: atomicFile,
		writer:        bufferWriter,
	}
}

// Done flushes the buffer and commits the atomic file if set. If flushing fails, it aborts the atomic file and returns the error.
func (mw *BufferWriter) Done() error {
	if err := mw.writer.Flush(); err != nil {
		mw.Abort()
		return err
	}
	if mw.atomicFileRef != nil {
		return mw.atomicFileRef.Commit()
	}
	return nil
}

// Abort aborts the atomic file if set in case of error.
func (mw *BufferWriter) Abort() {
	if mw.atomicFileRef != nil {
		mw.atomicFileRef.Abort()
	}
}

// WriteJSONEncode marshals the given value into JSON and writes it to the buffer.
// This is used for writing JSON values, ensuring proper escaping and formatting.
func (mw *BufferWriter) WriteJSONEncode(value any, stats *DebugStats) error {
	jsonBytes, err := json.Marshal(value)
	if err != nil {
		mw.Abort()
		return err
	}

	byteWritten, err := mw.writer.Write(jsonBytes)
	if err != nil {
		mw.Abort()
		return err
	}

	if stats != nil {
		stats.BytesWritten.Add(uint32(byteWritten))
	}

	return nil
}

// WriteJSONRaw writes the given string to the buffer as is, without JSON encoding.
// This is used for writing JSON structural characters like {, }, [, ], :, and ,.
func (mw *BufferWriter) WriteJSONRaw(jsonStr string, stats *DebugStats) error {
	byteWritten, err := mw.writer.WriteString(jsonStr)
	if err != nil {
		mw.Abort()
		return err
	}

	if stats != nil {
		stats.BytesWritten.Add(uint32(byteWritten))
	}

	return nil
}

func NewMockCmd(opts *CommandOptions) *cobra.Command {
	mockCmd := &cobra.Command{
		Use:   "mock",
		Short: "Generate mock data based from a string, an object string or from template files",
		Long: `Generate mock data based on --parse-str, (--parse-json / --parse-json-file) or (--parse-csv / --parse-csv-file) options, and output to --to-stdout and/or a file --to-file.

Mock functions:

* List available mock functions with --list.
* Always call the mock function with the format {{ functionName:{arg1}:{arg2}:{argN} }}. (Values not wrapped in double curly braces will be considered literal values)
* When passing parameters to the mock functions, wrap each value in curly braces ({value}) and use colon (:) outside the braces as the separator between parameters (e.g. {{ Number.number::{1}:{100} }}, {{ Date.time:{18:00}:{20:00} }}).
* Leave a parameter empty (bare :: or {}) to use its default value (e.g. {{ Number.number:::{100} }} leaves decimals and min at their defaults).
* For regex parameters, wrap the pattern in slashes (/pattern/) instead of curly braces (e.g. {{ Date.date:::/YYYY-MM-DD/ }}, {{ Regex.regex:/[a-z]{3}/ }}). Colons inside /…/ are never treated as delimiters.

Generate N root objects with --generate:

* Add --generate to specify the number of root objects to generate. (Not available for --parse-str)

Parsing JSON templates (--parse-json or --parse-json-file):

* For json inner objects, you may pass the desired number between brackets in the object's "key".

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
  { "phones[5]": "{{ Person.phoneNumber }}" }

  Will generate an array of 5 phone numbers.

  { "phones": [ "...", "...", "...", "...", "..." ] }

* When using --parse-json-file, the template filename must end with ".template.json".
  The default output file is written alongside the template file.
  e.g.: "path/to/employees.template.json" → "path/to/employees.json"

Parsing CSV templates (--parse-csv or --parse-csv-file):

* The CSV template must be a single row of comma-separated values, where each value is a string with 'colname:::"value"'.

	e.g.:
	name:::"{{ Person.name }}",age:::"{{ Number.number:{18}:{65} }}",email:::"{{ Internet.email }}"

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
  ktns mock --parse-str 'Hello my name is {{ Person.name }}, I am {{ Number.number:{0}:{1}:{100} }} years old'
  ktns mock --parse-json '{ "name": "{{ Person.name }}" }' --generate 5 --to-stdout --to-json-file ""
  ktns mock --parse-json '{ "name": "{{ Person.name }}" }' --to-file "path/to/mydata.json"
  ktns mock --parse-json-file "path/to/employees.template.json" --generate 5 --to-stdout --to-file ""
  ktns mock --parse-json-file "path/to/employees.template.json" --to-json-file "path/to/mydata.json"
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
			// parseCsvFileSet := cmd.Flags().Changed("parse-csv-file")
			generate, _ := cmd.Flags().GetInt("generate")
			toStdout, _ := cmd.Flags().GetBool("to-stdout")
			toFile, _ := cmd.Flags().GetString("to-file")
			toFileSet := cmd.Flags().Changed("to-file")
			debugMode, _ := cmd.Flags().GetBool("debug")

			if list {
				mocker := mocker.New()
				mocker.CobraList(opts.Out)
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

			if runningParseJson || runningParseJsonFile {
				var jsonPath string
				if toFileSet {
					var err error
					jsonPath, err = resolveJSONToFilePath(parseJsonFileSet, parseJsonFile, toFile)
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

				blueprintInitialNode := &TemplateNode{Type: NodeObject, Repeat: uint(generate)}
				templateGenerateTotal, err := compileJSONTemplate(rawJson, blueprintInitialNode)
				if err != nil {
					return err
				}

				stats := &DebugStats{
					GenerateTotal: templateGenerateTotal * uint(generate),
					StartTime:     time.Now(),
				}

				if debugMode {
					debugStop := startDebugRoutine(ctx, stats, os.Stderr)
					defer debugStop()
				}

				var atomicFile *AtomicFile
				if toFileSet {
					atomicFile, err = NewAtomicFile(jsonPath)
					if err != nil {
						return err
					}
				}

				var stdoutWriter io.Writer
				if toStdout {
					stdoutWriter = opts.Out
				}

				bufferWriter := NewBufferWriter(stdoutWriter, atomicFile)
				if bufferWriter == nil {
					return fmt.Errorf("failed to initialize output writer: no valid output destination configured")
				}

				mocker := mocker.New()
				blueprintInitialNode.GenerateJSON(mocker, stats, bufferWriter)
				if err := bufferWriter.Done(); err != nil {
					return err
				}
			}

			if runningParseCsv || runningParseCsvFile {
				return nil
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

// GenerateJSON generates mock data based on the TemplateNode structure and writes the output as JSON to the provided writer.
func (tn *TemplateNode) GenerateJSON(mocker *mocker.Mock, stats *DebugStats, bw *BufferWriter) error {
	switch tn.Type {
	case NodeObject:
		// Generate array of objects if [n] syntax is used in the key, otherwise just one object
		if tn.Repeat > 1 {
			bw.WriteJSONRaw("[", stats)
		}
		for r := range tn.Repeat {
			if r > 0 {
				bw.WriteJSONRaw(",", stats)
			}
			bw.WriteJSONRaw("{", stats)
			for i, child := range tn.Children {
				if i > 0 {
					bw.WriteJSONRaw(",", stats)
				}
				keyStr := fmt.Sprintf("%q:", child.Key)
				bw.WriteJSONRaw(keyStr, stats)
				if err := child.GenerateJSON(mocker, stats, bw); err != nil {
					return err
				}
			}
			bw.WriteJSONRaw("}", stats)
		}
		if tn.Repeat > 1 {
			bw.WriteJSONRaw("]", stats)
		}
	case NodeArray:
		bw.WriteJSONRaw("[", stats)
		for i, child := range tn.Children {
			if i > 0 {
				bw.WriteJSONRaw(",", stats)
			}
			if err := child.GenerateJSON(mocker, stats, bw); err != nil {
				return err
			}
		}
		bw.WriteJSONRaw("]", stats)
	case NodeString:
		// Try to find the mock function in the "value"
		interpretedValue, isMockFunction := parseDynamicValue(tn.Value)
		mockValue := interpretedValue

		// If its a mock function, extract the function name and parameters
		var functionName string
		var params []string
		var err error
		if isMockFunction {
			functionName, params, err = parseMockFunction(interpretedValue)
			if err != nil {
				return err
			}
		}

		// Generate array of values if [n] syntax is used in the key, otherwise just one value
		if tn.Repeat > 1 {
			bw.WriteJSONRaw("[", stats)
		}
		for r := range tn.Repeat {
			if r > 0 {
				bw.WriteJSONRaw(",", stats)
			}
			if isMockFunction {
				var err error
				mockValue, err = mocker.Generate(functionName, params)
				if err != nil {
					return err
				}
			}
			bw.WriteJSONEncode(mockValue, stats)
		}
		if tn.Repeat > 1 {
			bw.WriteJSONRaw("]", stats)
		}
		if stats != nil {
			stats.Generated.Add(uint32(tn.Repeat))
		}
	}

	return nil
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

// compileJSONTemplate walks through the raw JSON template and compiles it into a TemplateNode tree structure, and
// calculates the total number of items to be generated based on the presence of [n] patterns in the keys.
func compileJSONTemplate(obj map[string]any, node *TemplateNode) (uint, error) {
	var totalGenerate uint = 0

	for key, value := range obj {
		// Handle key[n] syntax
		keyRepeat := uint(1)
		cleanKey := key
		if matches := parseArrayItemBracketNumberRegex.FindStringSubmatch(key); len(matches) == 3 {
			cleanKey = matches[1]
			keyRepeat64, _ := strconv.ParseUint(matches[2], 10, 64)
			keyRepeat = uint(keyRepeat64)
		}

		childNode := &TemplateNode{Key: cleanKey, Repeat: keyRepeat}

		switch typedValue := value.(type) {
		case map[string]any:
			childNode.Type = NodeObject
			childGenerate, err := compileJSONTemplate(typedValue, childNode)
			if err != nil {
				return 0, err
			}
			totalGenerate += childGenerate * keyRepeat
		case []any:
			childNode.Type = NodeArray
			if keyRepeat > 1 {
				return 0, fmt.Errorf("invalid key '%s': array item keys cannot use [n] syntax to specify repeat count", key)
			}
			for _, arrItem := range typedValue {
				arrItemNode := &TemplateNode{Repeat: 1}
				switch typedArrItem := arrItem.(type) {
				case map[string]any:
					arrItemNode.Type = NodeObject
					childGenerate, err := compileJSONTemplate(typedArrItem, arrItemNode)
					if err != nil {
						return 0, err
					}
					childNode.Children = append(childNode.Children, arrItemNode)
					totalGenerate += childGenerate
				case []any:
					return 0, fmt.Errorf("nested arrays are not supported in json template")
				case string:
					arrItemNode.Type = NodeString
					arrItemNode.Value = typedArrItem
					childNode.Children = append(childNode.Children, arrItemNode)
					totalGenerate++
				default:
					arrItemNode.Type = NodeString
					arrItemNode.Value = arrItem.(string)
					childNode.Children = append(childNode.Children, arrItemNode)
					totalGenerate++
				}
			}
		case string:
			childNode.Type = NodeString
			childNode.Value = typedValue
			totalGenerate += keyRepeat
		default:
			childNode.Type = NodeString
			childNode.Value = value.(string)
			totalGenerate += keyRepeat
		}

		node.Children = append(node.Children, childNode)
	}

	return totalGenerate, nil
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

// resolveJSONToFilePath determines the output file path for JSON output based on the provided flags and template filename. It follows these rules:
// 1. If --to-json-file is set with a non-empty filename, use that.
// 2. If --to-json-file is set but the filename is empty, and --parse-json-file is set, derive the output filename by removing ".template" from the template filename.
// 3. If --to-json-file is set but the filename is empty, and --parse-json-file is not set, default to "output.json" in the current working directory.
func resolveJSONToFilePath(parseJsonFileSet bool, parseJsonFile string, toJsonFile string) (string, error) {
	// Specific file provided via --to-json-file, use it directly
	if toJsonFile != "" {
		return toJsonFile, nil
	}

	// Default filename based on --parse-json-file template name
	if parseJsonFileSet {
		return strings.TrimSuffix(parseJsonFile, ".template.json") + ".json", nil
	}

	// Default filename in current working directory `output.json`
	dir, e := workingDir()
	if e != nil {
		return "", e
	}
	defaultPath := filepath.Join(dir, "output.json")
	return defaultPath, nil
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

// startDebugRoutine launches a goroutine that prints live progress to out (always
// os.Stderr in production) at 200ms intervals. Call the returned stop function to
// terminate the display and print a final newline.
func startDebugRoutine(ctx context.Context, stats *DebugStats, out io.Writer) (stop func()) {
	innerCtx, cancel := context.WithCancel(ctx)
	done := make(chan struct{})

	render := func(final bool) {
		elapsed := time.Since(stats.StartTime).Seconds()
		generated := stats.Generated.Load()
		outputSizeBytes := stats.BytesWritten.Load()
		percentDone := 0.0
		if stats.GenerateTotal > 0 {
			percentDone = float64(generated) / float64(stats.GenerateTotal) * 100
		}
		usedMemPeakBytes := stats.UsedMemPeakBytes.Load()
		usedMemBytes, reservedMemSystemBytes := memSnapshotBytes()

		if usedMemBytes > usedMemPeakBytes {
			usedMemPeakBytes = usedMemBytes
			stats.UsedMemPeakBytes.Store(usedMemPeakBytes)
		}

		// '\r' at the start to overwrite the previous line, making it look like a live-updating single line of output
		strOutput := fmt.Sprintf("\r[debug] Elapsed: %s | Progress: %s / %s (%.1f%%) | Mem: %s ⌈%s⌉ [%s] | Output Size: ~%s",
			formatDurationMetrics(elapsed),
			formatNumberMetrics(uint64(generated)),
			formatNumberMetrics(uint64(stats.GenerateTotal)),
			percentDone,
			formatSizeMetrics(usedMemBytes),
			formatSizeMetrics(usedMemPeakBytes),
			formatSizeMetrics(reservedMemSystemBytes),
			formatSizeMetrics(uint64(outputSizeBytes)),
		)

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
