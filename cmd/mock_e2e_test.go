package cmd_test

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/lfsc09/k-test-n-stress/cmd"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type MockCmdE2ETestSuite struct {
	suite.Suite
}

func TestMockCmdE2ESuite(t *testing.T) {
	suite.Run(t, new(MockCmdE2ETestSuite))
}

func (suite *MockCmdE2ETestSuite) executeCommand(args ...string) (string, error) {
	// Create a buffer to capture the output
	outBuf := new(bytes.Buffer)

	opts := &cmd.CommandOptions{
		Out: outBuf,
	}

	rootCmd := cmd.NewRootCmd(opts)

	rootCmd.SetArgs(args)
	err := rootCmd.Execute()
	return outBuf.String(), err
}

/*
INVALID STATES
*/
func (suite *MockCmdE2ETestSuite) TestCLIShouldRaiseError_NothingToBeParsed() {
	testName := "Should raise error when nothing to be parsed"
	_, err := suite.executeCommand("mock")
	assert.Error(suite.T(), err, testName)
	assert.EqualError(suite.T(), err, "nothing to be parsed, ask for help -h or --help", testName)
}

func (suite *MockCmdE2ETestSuite) TestCLIShouldRaiseError_MultipleParseFlags() {
	tests := []struct {
		testName string
		input    []string
	}{
		{
			testName: "both --parse-str and --parse-json-file",
			input:    []string{"mock", "--parse-str", "Hello {{ Person.name }}", "--parse-json-file", "test.json"},
		},
		{
			testName: "both --parse-str and --parse-json",
			input:    []string{"mock", "--parse-str", "Hello {{ Person.name }}", "--parse-json", "' {\"name\": \"{{ Person.name }}\"} '"},
		},
		{
			testName: "both --parse-json and --parse-json-file",
			input:    []string{"mock", "--parse-json", "' {\"name\": \"{{ Person.name }}\"} '", "--parse-json-file", "test.json"},
		},
		{
			testName: "all three --parse-str, --parse-json and --parse-json-file",
			input:    []string{"mock", "--parse-str", "Hello {{ Person.name }}", "--parse-json", "' {\"name\": \"{{ Person.name }}\"} '", "--parse-json-file", "test.json"},
		},
	}
	for _, test := range tests {
		_, err := suite.executeCommand(test.input...)
		assert.Error(suite.T(), err, test.testName)
		assert.EqualError(suite.T(), err, "provide only one of the three options: --parse-json, --parse-json-file or --parse-str", test.testName)
	}
}

func (suite *MockCmdE2ETestSuite) TestCLIShouldRaiseError_GenerateFlagInvalidUse() {
	tests := []struct {
		testName string
		input    []string
	}{
		{
			testName: "--generate with --parse-str",
			input:    []string{"mock", "--parse-str", "Hello {{ Person.name }}", "--generate", "5"},
		},
	}
	for _, test := range tests {
		_, err := suite.executeCommand(test.input...)
		assert.Error(suite.T(), err, test.testName)
		assert.EqualError(suite.T(), err, "--generate option is only available when using --parse-json or --parse-json-file", test.testName)
	}
}

func (suite *MockCmdE2ETestSuite) TestCLIShouldRaiseError_GenerateFlagInvalidValues() {
	tests := []struct {
		testName string
		input    []string
	}{
		{
			testName: "--generate with value equal to 0",
			input:    []string{"mock", "--parse-json", "' {\"name\": \"{{ Person.name }}\"} '", "--generate", "0", "--to-json-stdout"},
		},
		{
			testName: "--generate with negative value",
			input:    []string{"mock", "--parse-json", "' {\"name\": \"{{ Person.name }}\"} '", "--generate", "-1", "--to-json-stdout"},
		},
	}
	for _, test := range tests {
		_, err := suite.executeCommand(test.input...)
		assert.Error(suite.T(), err, test.testName)
		assert.EqualError(suite.T(), err, "--generate option must be greater than 0", test.testName)
	}
}

func (suite *MockCmdE2ETestSuite) TestCLIShouldRaiseError_ParseJsonFileNotFound() {
	testName := "Should raise error when --parse-json-file file is not found"
	_, err := suite.executeCommand("mock", "--parse-json-file", "/nonexistent/path/file.template.json", "--to-json-stdout")
	assert.Error(suite.T(), err, testName)
	assert.Contains(suite.T(), err.Error(), "template file not found", testName)
}

func (suite *MockCmdE2ETestSuite) TestCLIShouldRaiseError_ParseJsonFileInvalidExtension() {
	testName := "Should raise error when --parse-json-file does not end with .template.json"
	_, err := suite.executeCommand("mock", "--parse-json-file", "/tmp/employee.json", "--to-json-stdout")
	assert.Error(suite.T(), err, testName)
	assert.Contains(suite.T(), err.Error(), "requires the template filename to end with '.template.json'", testName)
}

func (suite *MockCmdE2ETestSuite) TestCLIShouldInlineError_BareParamValues() {
	// parse-str inlines errors in the output string
	parseStrCases := []struct {
		testName string
		template string
	}{
		{testName: "bare first param", template: "{{ Number.number:2 }}"},
		{testName: "bare second param, first is valid", template: "{{ Number.number:{0}:5 }}"},
		{testName: "bare first of three params", template: "{{ Date.date:2020-01-01:{2030-01-01}:{} }}"},
	}
	for _, tt := range parseStrCases {
		stdOut, err := suite.executeCommand("mock", "--parse-str", tt.template)
		assert.NoError(suite.T(), err, tt.testName)
		assert.Contains(suite.T(), stdOut, "must be wrapped in", tt.testName)
	}

	// parse-json propagates a Go error
	parseJsonCases := []struct {
		testName string
		jsonStr  string
	}{
		{testName: "bare first param in json", jsonStr: `{"n": "{{ Number.number:2:1:5 }}"}`},
		{testName: "bare second param in json", jsonStr: `{"n": "{{ Number.number:{0}:5 }}"}`},
	}
	for _, tt := range parseJsonCases {
		_, err := suite.executeCommand("mock", "--parse-json", tt.jsonStr, "--to-json-stdout")
		assert.Error(suite.T(), err, tt.testName)
		assert.Contains(suite.T(), err.Error(), "must be wrapped in", tt.testName)
	}
}

/*
VALID STATES
*/
func (suite *MockCmdE2ETestSuite) TestCLIShouldReturnListOfMockFunctions() {
	testName := "Should return list of mock functions"
	stdOut, err := suite.executeCommand("mock", "--list")
	assert.NoError(suite.T(), err, testName)
	assert.Contains(suite.T(), stdOut, "Address.", testName)
	assert.Contains(suite.T(), stdOut, "Boolean.", testName)
	assert.Contains(suite.T(), stdOut, "Car.", testName)
	assert.Contains(suite.T(), stdOut, "Company.", testName)
	assert.Contains(suite.T(), stdOut, "Currency.", testName)
	assert.Contains(suite.T(), stdOut, "File.", testName)
	assert.Contains(suite.T(), stdOut, "Internet.", testName)
	assert.Contains(suite.T(), stdOut, "Lorem.", testName)
	assert.Contains(suite.T(), stdOut, "Number.", testName)
	assert.Contains(suite.T(), stdOut, "Payment.", testName)
	assert.Contains(suite.T(), stdOut, "Person.", testName)
	assert.Contains(suite.T(), stdOut, "Regex.", testName)
	assert.Contains(suite.T(), stdOut, "Date.", testName)
	assert.Contains(suite.T(), stdOut, "UUID.", testName)
	assert.Contains(suite.T(), stdOut, "UUID.uuidv7", testName)
	assert.Contains(suite.T(), stdOut, "UserAgent.", testName)
}

func (suite *MockCmdE2ETestSuite) TestCLIShouldParseJsonFile() {
	testName := "Should parse a single .template.json file and write output alongside it"
	tmpDir := suite.T().TempDir()
	templatePath := filepath.Join(tmpDir, "employee.template.json")
	templateContent := `{"name": "{{ Person.name }}"}`
	err := os.WriteFile(templatePath, []byte(templateContent), 0644)
	assert.NoError(suite.T(), err, testName)

	_, err = suite.executeCommand("mock", "--parse-json-file", templatePath, "--to-json-stdout")
	assert.NoError(suite.T(), err, testName)

	outPath := filepath.Join(tmpDir, "employee.json")
	_, statErr := os.Stat(outPath)
	assert.NoError(suite.T(), statErr, testName)

	outContent, readErr := os.ReadFile(outPath)
	assert.NoError(suite.T(), readErr, testName)

	var result map[string]any
	jsonErr := json.Unmarshal(outContent, &result)
	assert.NoError(suite.T(), jsonErr, testName)
	_, hasName := result["name"]
	assert.True(suite.T(), hasName, testName)
}

func (suite *MockCmdE2ETestSuite) TestCLIShouldParseJsonFile_WithGenerate() {
	testName := "Should parse a single .template.json file with --generate and produce a JSON array"
	tmpDir := suite.T().TempDir()
	templatePath := filepath.Join(tmpDir, "employee.template.json")
	templateContent := `{"name": "{{ Person.name }}"}`
	err := os.WriteFile(templatePath, []byte(templateContent), 0644)
	assert.NoError(suite.T(), err, testName)

	_, err = suite.executeCommand("mock", "--parse-json-file", templatePath, "--generate", "3", "--to-json-stdout")
	assert.NoError(suite.T(), err, testName)

	outPath := filepath.Join(tmpDir, "employee.json")
	outContent, readErr := os.ReadFile(outPath)
	assert.NoError(suite.T(), readErr, testName)

	var result []map[string]any
	jsonErr := json.Unmarshal(outContent, &result)
	assert.NoError(suite.T(), jsonErr, testName)
	assert.Len(suite.T(), result, 3, testName)
}

func (suite *MockCmdE2ETestSuite) TestCLIShouldParseJsonFile_DeletesPreviousOutput() {
	testName := "Should delete stale output file before writing new one"
	tmpDir := suite.T().TempDir()
	templatePath := filepath.Join(tmpDir, "employee.template.json")
	templateContent := `{"name": "{{ Person.name }}"}`
	err := os.WriteFile(templatePath, []byte(templateContent), 0644)
	assert.NoError(suite.T(), err, testName)

	outPath := filepath.Join(tmpDir, "employee.json")
	staleContent := `{"stale": "data"}`
	err = os.WriteFile(outPath, []byte(staleContent), 0644)
	assert.NoError(suite.T(), err, testName)

	_, err = suite.executeCommand("mock", "--parse-json-file", templatePath, "--to-json-file", "")
	assert.NoError(suite.T(), err, testName)

	outContent, readErr := os.ReadFile(outPath)
	assert.NoError(suite.T(), readErr, testName)
	assert.NotContains(suite.T(), string(outContent), "stale", testName)

	var result map[string]any
	jsonErr := json.Unmarshal(outContent, &result)
	assert.NoError(suite.T(), jsonErr, testName)
	_, hasName := result["name"]
	assert.True(suite.T(), hasName, testName)
}

func (suite *MockCmdE2ETestSuite) TestCLIShouldMockFromParseStr() {
	tests := []struct {
		testName      string
		input         []string
		expectedValue string
	}{
		{
			testName:      "Should mock from --parse-str (no mock functions)",
			input:         []string{"mock", "--parse-str", "Hello world"},
			expectedValue: "Hello world",
		},
		{
			testName:      "Should mock from --parse-str",
			input:         []string{"mock", "--parse-str", "Hello {{ Person.name }}"},
			expectedValue: "Hello ",
		},
	}
	for _, test := range tests {
		stdOut, err := suite.executeCommand(test.input...)
		assert.NoError(suite.T(), err, test.testName)
		assert.Contains(suite.T(), stdOut, test.expectedValue, test.testName)
	}
}

func (suite *MockCmdE2ETestSuite) TestCLIShouldOutputToJsonFile_DefaultName_ParseJson() {
	testName := "Should create output.json in the current working directory when --to-json-file is passed with no value (parse-json)"
	_, err := suite.executeCommand("mock", "--parse-json", `{"name": "{{ Person.name }}"}`, "--to-json-file", "")
	assert.NoError(suite.T(), err, testName)
	// The file is written in the process's working directory (os.Getwd()).
	// We simply assert no error was returned.
}

func (suite *MockCmdE2ETestSuite) TestCLIShouldOutputToJsonFile_ExplicitName_ParseJson() {
	testName := "Should write JSON to explicit path with --to-json-file <path>"
	tmpDir := suite.T().TempDir()
	outPath := filepath.Join(tmpDir, "result.json")
	// With NoOptDefVal set, explicit values must use --flag=value syntax
	_, err := suite.executeCommand("mock", "--parse-json", `{"name": "{{ Person.name }}"}`, "--to-json-file", outPath)
	assert.NoError(suite.T(), err, testName)
	data, readErr := os.ReadFile(outPath)
	assert.NoError(suite.T(), readErr, testName)
	var result map[string]any
	assert.NoError(suite.T(), json.Unmarshal(data, &result), testName)
}

func (suite *MockCmdE2ETestSuite) TestCLIShouldOutputToJsonFile_DefaultName_ParseJsonFile() {
	testName := "Should write default employee.json alongside template when --to-json-file is passed with no value"
	tmpDir := suite.T().TempDir()
	templatePath := filepath.Join(tmpDir, "employee.template.json")
	_ = os.WriteFile(templatePath, []byte(`{"name": "{{ Person.name }}"}`), 0644)
	_, err := suite.executeCommand("mock", "--parse-json-file", templatePath, "--to-json-file", "")
	assert.NoError(suite.T(), err, testName)
	outPath := filepath.Join(tmpDir, "employee.json")
	data, readErr := os.ReadFile(outPath)
	assert.NoError(suite.T(), readErr, testName)
	var result map[string]any
	assert.NoError(suite.T(), json.Unmarshal(data, &result), testName)
}

func (suite *MockCmdE2ETestSuite) TestCLIShouldOutputToJsonFile_ExplicitName_ParseJsonFile() {
	testName := "Should write to explicit path and NOT create default file when explicit --to-json-file is given"
	tmpDir := suite.T().TempDir()
	templatePath := filepath.Join(tmpDir, "employee.template.json")
	_ = os.WriteFile(templatePath, []byte(`{"name": "{{ Person.name }}"}`), 0644)
	outPath := filepath.Join(tmpDir, "out.json")
	// With NoOptDefVal set, explicit values must use --flag=value syntax
	_, err := suite.executeCommand("mock", "--parse-json-file", templatePath, "--to-json-file", outPath)
	assert.NoError(suite.T(), err, testName)
	// Explicit output file must exist
	data, readErr := os.ReadFile(outPath)
	assert.NoError(suite.T(), readErr, testName)
	var result map[string]any
	assert.NoError(suite.T(), json.Unmarshal(data, &result), testName)
	// Default file must NOT exist
	defaultOut := filepath.Join(tmpDir, "employee.json")
	_, statErr := os.Stat(defaultOut)
	assert.True(suite.T(), os.IsNotExist(statErr), testName)
}

func (suite *MockCmdE2ETestSuite) TestCLIShouldOutputToCsvFile_DefaultName_ParseJson() {
	testName := "Should create output.csv in the current working directory when --to-csv-file is passed with no value (parse-json)"
	_, err := suite.executeCommand("mock", "--parse-json", `{"name": "{{ Person.name }}"}`, "--to-csv-file")
	assert.NoError(suite.T(), err, testName)
	// The file is written in the process's working directory (os.Getwd()).
	// We simply assert no error was returned.
}

func (suite *MockCmdE2ETestSuite) TestCLIShouldOutputToCsvFile_ExplicitName_ParseJson() {
	testName := "Should write CSV to explicit path with --to-csv-file <path>"
	tmpDir := suite.T().TempDir()
	outPath := filepath.Join(tmpDir, "result.csv")
	_, err := suite.executeCommand("mock", "--parse-json", `{"name": "{{ Person.name }}", "age": "{{ Number.number::{18}:{80} }}"}`, "--to-csv-file", outPath)
	assert.NoError(suite.T(), err, testName)
	data, readErr := os.ReadFile(outPath)
	assert.NoError(suite.T(), readErr, testName)
	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	// Header line should be sorted keys: age,name
	assert.Equal(suite.T(), "age,name", lines[0], testName)
	// 1 header + 1 data row
	assert.Len(suite.T(), lines, 2, testName)
}

func (suite *MockCmdE2ETestSuite) TestCLIShouldOutputToCsvFile_DefaultName_ParseJsonFile() {
	testName := "Should write default employee.csv alongside template when --to-csv-file is passed with no value"
	tmpDir := suite.T().TempDir()
	templatePath := filepath.Join(tmpDir, "employee.template.json")
	_ = os.WriteFile(templatePath, []byte(`{"name": "{{ Person.name }}"}`), 0644)
	_, err := suite.executeCommand("mock", "--parse-json-file", templatePath, "--to-csv-file", "")
	assert.NoError(suite.T(), err, testName)
	outPath := filepath.Join(tmpDir, "employee.csv")
	data, readErr := os.ReadFile(outPath)
	assert.NoError(suite.T(), readErr, testName)
	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	assert.Equal(suite.T(), "name", lines[0], testName)
	assert.Len(suite.T(), lines, 2, testName)
}

func (suite *MockCmdE2ETestSuite) TestCLIShouldOutputToCsvFile_ExplicitName_ParseJsonFile() {
	testName := "Should write to explicit CSV path and NOT create default file when explicit --to-csv-file is given"
	tmpDir := suite.T().TempDir()
	templatePath := filepath.Join(tmpDir, "employee.template.json")
	_ = os.WriteFile(templatePath, []byte(`{"name": "{{ Person.name }}"}`), 0644)
	outPath := filepath.Join(tmpDir, "out.csv")
	_, err := suite.executeCommand("mock", "--parse-json-file", templatePath, "--to-csv-file", outPath)
	assert.NoError(suite.T(), err, testName)
	// Explicit output file must exist
	data, readErr := os.ReadFile(outPath)
	assert.NoError(suite.T(), readErr, testName)
	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	assert.Equal(suite.T(), "name", lines[0], testName)
	// Default file must NOT exist
	defaultOut := filepath.Join(tmpDir, "employee.csv")
	_, statErr := os.Stat(defaultOut)
	assert.True(suite.T(), os.IsNotExist(statErr), testName)
}

func (suite *MockCmdE2ETestSuite) TestCLIShouldOutputToCsvFile_DeletesPreviousOutput() {
	testName := "Should delete stale CSV output file before writing new one"
	tmpDir := suite.T().TempDir()
	templatePath := filepath.Join(tmpDir, "employee.template.json")
	_ = os.WriteFile(templatePath, []byte(`{"name": "{{ Person.name }}"}`), 0644)
	outPath := filepath.Join(tmpDir, "employee.csv")
	_ = os.WriteFile(outPath, []byte("stale,data\n1,2\n"), 0644)
	_, err := suite.executeCommand("mock", "--parse-json-file", templatePath, "--to-csv-file", outPath)
	assert.NoError(suite.T(), err, testName)
	data, readErr := os.ReadFile(outPath)
	assert.NoError(suite.T(), readErr, testName)
	assert.NotContains(suite.T(), string(data), "stale", testName)
}

func (suite *MockCmdE2ETestSuite) TestCLIShouldOutputBothJsonAndCsvFile() {
	testName := "Should write both JSON and CSV files when both --to-json-file and --to-csv-file are given"
	tmpDir := suite.T().TempDir()
	jsonPath := filepath.Join(tmpDir, "result.json")
	csvPath := filepath.Join(tmpDir, "result.csv")
	_, err := suite.executeCommand("mock", "--parse-json", `{"name": "{{ Person.name }}"}`, "--to-json-file", jsonPath, "--to-csv-file", csvPath)
	assert.NoError(suite.T(), err, testName)
	_, jsonErr := os.Stat(jsonPath)
	assert.NoError(suite.T(), jsonErr, testName)
	_, csvErr := os.Stat(csvPath)
	assert.NoError(suite.T(), csvErr, testName)
}

func (suite *MockCmdE2ETestSuite) TestCLIParseJson_NestedStructures() {
	tests := []struct {
		testName       string
		jsonStr        string
		expectError    bool
		assertContains string
	}{
		{
			testName: "nested object with params",
			jsonStr:  `{"person":{"name":"{{ Person.name }}","age":"{{ Number.number:{0}:{18}:{90} }}"}}`,
		},
		{
			testName:       "array key generates multiple values",
			jsonStr:        `{"phones[3]":"{{ Person.phoneNumber }}"}`,
			assertContains: "phones",
		},
		{
			testName:       "inner object array",
			jsonStr:        `{"employees[2]":{"name":"{{ Person.name }}"}}`,
			assertContains: "employees",
		},
		{
			testName:    "invalid value type (integer)",
			jsonStr:     `{"x":123}`,
			expectError: true,
		},
		{
			testName:       "unknown mock function",
			jsonStr:        `{"x":"{{ Unknown.function }}"}`,
			expectError:    true,
			assertContains: "unknown mock function",
		},
	}

	for _, tt := range tests {
		stdOut, err := suite.executeCommand("mock", "--parse-json", tt.jsonStr, "--to-json-stdout")
		if tt.expectError {
			assert.Error(suite.T(), err, tt.testName)
			if tt.assertContains != "" {
				assert.Contains(suite.T(), err.Error(), tt.assertContains, tt.testName)
			}
		} else {
			assert.NoError(suite.T(), err, tt.testName)
			if tt.assertContains != "" {
				assert.Contains(suite.T(), stdOut, tt.assertContains, tt.testName)
			}
		}
	}
}

func (suite *MockCmdE2ETestSuite) TestCLIParseJson_GenerateFlag() {
	tests := []struct {
		testName    string
		args        []string
		assertRegex string
	}{
		{
			testName:    "generate=3, output is JSON array with 3 elements",
			args:        []string{"mock", "--parse-json", `{"name":"{{ Person.name }}"}`, "--generate", "3", "--to-json-stdout"},
			assertRegex: `^\[`,
		},
		{
			testName:    "generate=1 (default), output is JSON object not array",
			args:        []string{"mock", "--parse-json", `{"name":"{{ Person.name }}"}`, "--to-json-stdout"},
			assertRegex: `^\{`,
		},
	}

	for _, tt := range tests {
		stdOut, err := suite.executeCommand(tt.args...)
		assert.NoError(suite.T(), err, tt.testName)
		assert.Regexp(suite.T(), regexp.MustCompile(tt.assertRegex), stdOut, tt.testName)
	}
}

func (suite *MockCmdE2ETestSuite) TestCLIParseStr_EdgeCases() {
	tests := []struct {
		testName       string
		template       string
		exactOutput    string
		assertContains string
		assertPrefix   string
		assertSuffix   string
	}{
		{testName: "literal only (no mock)", template: "Hello world", exactOutput: "Hello world\n"},
		{testName: "unclosed {{", template: "Hello {{ Person.name", exactOutput: "Hello {{ Person.name\n"},
		{testName: "unknown function inlined as error", template: "{{ Unknown.fn }}", assertContains: "[unknown mock function 'Unknown.fn']"},
		{testName: "bare param inlined as error", template: "{{ Number.number:5 }}", assertContains: "[mock function parameter '5' must be wrapped in"},
		{testName: "empty mock call", template: "{{  }}", assertContains: "[unknown mock function '']"},
		{testName: "two mock functions in one string", template: "{{ Person.firstName }} {{ Person.lastName }}", assertContains: " "},
		{testName: "mock surrounded by literals", template: "start {{ Person.name }} end", assertPrefix: "start ", assertSuffix: " end\n"},
	}

	for _, tt := range tests {
		stdOut, err := suite.executeCommand("mock", "--parse-str", tt.template)
		assert.NoError(suite.T(), err, tt.testName)
		if tt.exactOutput != "" {
			assert.Equal(suite.T(), tt.exactOutput, stdOut, tt.testName)
		}
		if tt.assertContains != "" {
			assert.Contains(suite.T(), stdOut, tt.assertContains, tt.testName)
		}
		if tt.assertPrefix != "" {
			assert.True(suite.T(), len(stdOut) >= len(tt.assertPrefix) && stdOut[:len(tt.assertPrefix)] == tt.assertPrefix, tt.testName)
		}
		if tt.assertSuffix != "" {
			assert.True(suite.T(), len(stdOut) >= len(tt.assertSuffix) && stdOut[len(stdOut)-len(tt.assertSuffix):] == tt.assertSuffix, tt.testName)
		}
	}
}
