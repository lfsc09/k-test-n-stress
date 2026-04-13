package cmd_test

import (
	"bytes"
	"encoding/json"
	"fmt"
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

func TestMockCmdTestSuite(t *testing.T) {
	suite.Run(t, new(MockCmdE2ETestSuite))
}

// Test the command line interface (CLI) of the application. (Specifically for the mock command)
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
			input:    []string{"mock", "--parse-json", "' {\"name\": \"{{ Person.name }}\"} '", "--generate", "0", "--to-stdout", "as-json"},
		},
		{
			testName: "--generate with negative value",
			input:    []string{"mock", "--parse-json", "' {\"name\": \"{{ Person.name }}\"} '", "--generate", "-1", "--to-stdout", "as-json"},
		},
	}
	for _, test := range tests {
		_, err := suite.executeCommand(test.input...)
		assert.Error(suite.T(), err, test.testName)
		assert.EqualError(suite.T(), err, "--generate option must be greater than 0", test.testName)
	}
}

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

func (suite *MockCmdE2ETestSuite) TestCLIShouldRaiseError_ParseJsonFileNotFound() {
	testName := "Should raise error when --parse-json-file file is not found"
	_, err := suite.executeCommand("mock", "--parse-json-file", "/nonexistent/path/file.template.json", "--to-json-file")
	assert.Error(suite.T(), err, testName)
	assert.Contains(suite.T(), err.Error(), "template file not found", testName)
}

func (suite *MockCmdE2ETestSuite) TestCLIShouldParseJsonFile() {
	testName := "Should parse a single .template.json file and write output alongside it"
	tmpDir := suite.T().TempDir()
	templatePath := filepath.Join(tmpDir, "employee.template.json")
	templateContent := `{"name": "{{ Person.name }}"}`
	err := os.WriteFile(templatePath, []byte(templateContent), 0644)
	assert.NoError(suite.T(), err, testName)

	_, err = suite.executeCommand("mock", "--parse-json-file", templatePath, "--to-json-file")
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

	_, err = suite.executeCommand("mock", "--parse-json-file", templatePath, "--generate", "3", "--to-json-file")
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

	_, err = suite.executeCommand("mock", "--parse-json-file", templatePath, "--to-json-file")
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

// --- Step 11: .template.json constraint ---

func (suite *MockCmdE2ETestSuite) TestCLIShouldRaiseError_ParseJsonFileInvalidExtension() {
	testName := "Should raise error when --parse-json-file does not end with .template.json"
	_, err := suite.executeCommand("mock", "--parse-json-file", "/tmp/employee.json", "--to-json-file")
	assert.Error(suite.T(), err, testName)
	assert.Contains(suite.T(), err.Error(), "requires the template filename to end with '.template.json'", testName)
}

// --- Step 12: --to-stdout validation ---

func (suite *MockCmdE2ETestSuite) TestCLIShouldRaiseError_ToStdoutInvalidValue() {
	testName := "Should raise error when --to-stdout has an invalid value"
	_, err := suite.executeCommand("mock", "--parse-json", `{"name": "{{ Person.name }}"}`, "--to-stdout", "invalid")
	assert.Error(suite.T(), err, testName)
	assert.Contains(suite.T(), err.Error(), "only accepts 'as-json' or 'as-csv'", testName)
}

func (suite *MockCmdE2ETestSuite) TestCLIShouldRaiseError_ToStdoutPrettifyWithoutToStdout() {
	testName := "Should raise error when --to-stdout-prettify is used without --to-stdout"
	_, err := suite.executeCommand("mock", "--parse-json", `{"name": "{{ Person.name }}"}`, "--to-stdout-prettify")
	assert.Error(suite.T(), err, testName)
	assert.Contains(suite.T(), err.Error(), "requires --to-stdout to be set", testName)
}

func (suite *MockCmdE2ETestSuite) TestCLIShouldRaiseError_ParseStrWithToStdout() {
	testName := "Should raise error when --parse-str is combined with --to-stdout"
	_, err := suite.executeCommand("mock", "--parse-str", "Hello", "--to-stdout", "as-json")
	assert.Error(suite.T(), err, testName)
	assert.Contains(suite.T(), err.Error(), "--parse-str always outputs to stdout", testName)
}

func (suite *MockCmdE2ETestSuite) TestCLIShouldRaiseError_ParseStrWithToJsonFile() {
	testName := "Should raise error when --parse-str is combined with --to-json-file"
	_, err := suite.executeCommand("mock", "--parse-str", "Hello", "--to-json-file")
	assert.Error(suite.T(), err, testName)
	assert.Contains(suite.T(), err.Error(), "--parse-str always outputs to stdout", testName)
}

func (suite *MockCmdE2ETestSuite) TestCLIShouldRaiseError_ParseJsonWithNoOutputFlag() {
	testName := "Should raise error when --parse-json is used without any output flag"
	_, err := suite.executeCommand("mock", "--parse-json", `{"name": "{{ Person.name }}"}`)
	assert.Error(suite.T(), err, testName)
	assert.Contains(suite.T(), err.Error(), "require at least one output flag", testName)
}

func (suite *MockCmdE2ETestSuite) TestCLIShouldRaiseError_ParseJsonFileWithNoOutputFlag() {
	testName := "Should raise error when --parse-json-file is used without any output flag"
	tmpDir := suite.T().TempDir()
	templatePath := filepath.Join(tmpDir, "employee.template.json")
	_ = os.WriteFile(templatePath, []byte(`{"name": "{{ Person.name }}"}`), 0644)
	_, err := suite.executeCommand("mock", "--parse-json-file", templatePath)
	assert.Error(suite.T(), err, testName)
	assert.Contains(suite.T(), err.Error(), "require at least one output flag", testName)
}

// --- Step 13: --to-stdout as-json ---

func (suite *MockCmdE2ETestSuite) TestCLIShouldOutputToStdout_AsJson_ParseJson() {
	testName := "Should output compact JSON to stdout with --to-stdout as-json"
	stdOut, err := suite.executeCommand("mock", "--parse-json", `{"name": "{{ Person.name }}"}`, "--to-stdout", "as-json")
	assert.NoError(suite.T(), err, testName)
	// Should be valid JSON
	var result map[string]any
	assert.NoError(suite.T(), json.Unmarshal([]byte(strings.TrimSpace(stdOut)), &result), testName)
	// Compact: should NOT contain indentation (two-space)
	assert.NotContains(suite.T(), stdOut, "  ", testName)
}

func (suite *MockCmdE2ETestSuite) TestCLIShouldOutputToStdout_AsJson_ParseJson_Prettify() {
	testName := "Should output indented JSON to stdout with --to-stdout as-json --to-stdout-prettify"
	stdOut, err := suite.executeCommand("mock", "--parse-json", `{"name": "{{ Person.name }}"}`, "--to-stdout", "as-json", "--to-stdout-prettify")
	assert.NoError(suite.T(), err, testName)
	assert.Contains(suite.T(), stdOut, "  ", testName)
	// Should still be valid JSON
	var result map[string]any
	assert.NoError(suite.T(), json.Unmarshal([]byte(strings.TrimSpace(stdOut)), &result), testName)
}

func (suite *MockCmdE2ETestSuite) TestCLIShouldOutputToStdout_AsJson_ParseJsonFile() {
	testName := "Should output JSON to stdout from --parse-json-file with --to-stdout as-json (no file written)"
	tmpDir := suite.T().TempDir()
	templatePath := filepath.Join(tmpDir, "employee.template.json")
	_ = os.WriteFile(templatePath, []byte(`{"name": "{{ Person.name }}"}`), 0644)
	stdOut, err := suite.executeCommand("mock", "--parse-json-file", templatePath, "--to-stdout", "as-json")
	assert.NoError(suite.T(), err, testName)
	var result map[string]any
	assert.NoError(suite.T(), json.Unmarshal([]byte(strings.TrimSpace(stdOut)), &result), testName)
	// No output file should have been created (--to-json-file was not passed)
	defaultOut := filepath.Join(tmpDir, "employee.json")
	_, statErr := os.Stat(defaultOut)
	assert.True(suite.T(), os.IsNotExist(statErr), testName)
}

// --- Step 14: --to-stdout as-csv ---

func (suite *MockCmdE2ETestSuite) TestCLIShouldOutputToStdout_AsCsv_ParseJson() {
	testName := "Should output compact CSV to stdout with --to-stdout as-csv"
	stdOut, err := suite.executeCommand("mock", "--parse-json", `{"name": "{{ Person.name }}", "age": "{{ Number.number::{18}:{80} }}"}`, "--to-stdout", "as-csv")
	assert.NoError(suite.T(), err, testName)
	lines := strings.Split(strings.TrimSpace(stdOut), "\n")
	// Header line should be sorted keys
	assert.Equal(suite.T(), "age,name", lines[0], testName)
	// Exactly 2 lines: header + 1 data row
	assert.Len(suite.T(), lines, 2, testName)
}

func (suite *MockCmdE2ETestSuite) TestCLIShouldOutputToStdout_AsCsv_ParseJson_Prettify() {
	testName := "Should output prettified CSV to stdout with --to-stdout as-csv --to-stdout-prettify"
	stdOut, err := suite.executeCommand("mock", "--parse-json", `{"name": "{{ Person.name }}", "age": "{{ Number.number::{18}:{80} }}"}`, "--to-stdout", "as-csv", "--to-stdout-prettify")
	assert.NoError(suite.T(), err, testName)
	assert.Contains(suite.T(), stdOut, " | ", testName)
	assert.Contains(suite.T(), stdOut, "---", testName)
}

func (suite *MockCmdE2ETestSuite) TestCLIShouldOutputToStdout_AsCsv_ParseJson_WithGenerate() {
	testName := "Should output CSV with multiple rows when --generate is set"
	stdOut, err := suite.executeCommand("mock", "--parse-json", `{"name": "{{ Person.name }}"}`, "--generate", "3", "--to-stdout", "as-csv")
	assert.NoError(suite.T(), err, testName)
	lines := strings.Split(strings.TrimSpace(stdOut), "\n")
	// 1 header + 3 data rows
	assert.Len(suite.T(), lines, 4, testName)
}

// --- Step 15: --to-json-file ---

func (suite *MockCmdE2ETestSuite) TestCLIShouldOutputToJsonFile_DefaultName_ParseJson() {
	testName := "Should create output.json beside binary when --to-json-file is passed with no value (parse-json)"
	_, err := suite.executeCommand("mock", "--parse-json", `{"name": "{{ Person.name }}"}`, "--to-json-file")
	assert.NoError(suite.T(), err, testName)
	// The file is written beside os.Executable(); in tests that's a temp binary path
	// We simply assert no error was returned — the exact path is environment-dependent
}

func (suite *MockCmdE2ETestSuite) TestCLIShouldOutputToJsonFile_ExplicitName_ParseJson() {
	testName := "Should write JSON to explicit path with --to-json-file <path>"
	tmpDir := suite.T().TempDir()
	outPath := filepath.Join(tmpDir, "result.json")
	// With NoOptDefVal set, explicit values must use --flag=value syntax
	_, err := suite.executeCommand("mock", "--parse-json", `{"name": "{{ Person.name }}"}`, "--to-json-file="+outPath)
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
	_, err := suite.executeCommand("mock", "--parse-json-file", templatePath, "--to-json-file")
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
	_, err := suite.executeCommand("mock", "--parse-json-file", templatePath, "--to-json-file="+outPath)
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

// --- Step 16: --to-csv-file ---

func (suite *MockCmdE2ETestSuite) TestCLIShouldRaiseError_ParseStrWithToCsvFile() {
	testName := "Should raise error when --parse-str is combined with --to-csv-file"
	_, err := suite.executeCommand("mock", "--parse-str", "Hello", "--to-csv-file")
	assert.Error(suite.T(), err, testName)
	assert.Contains(suite.T(), err.Error(), "--parse-str always outputs to stdout", testName)
}

func (suite *MockCmdE2ETestSuite) TestCLIShouldOutputToCsvFile_DefaultName_ParseJson() {
	testName := "Should create output.csv beside binary when --to-csv-file is passed with no value (parse-json)"
	_, err := suite.executeCommand("mock", "--parse-json", `{"name": "{{ Person.name }}"}`, "--to-csv-file")
	assert.NoError(suite.T(), err, testName)
	// File is beside os.Executable(); in tests that's a temp binary path
	// We simply assert no error was returned — the exact path is environment-dependent
}

func (suite *MockCmdE2ETestSuite) TestCLIShouldOutputToCsvFile_ExplicitName_ParseJson() {
	testName := "Should write CSV to explicit path with --to-csv-file <path>"
	tmpDir := suite.T().TempDir()
	outPath := filepath.Join(tmpDir, "result.csv")
	_, err := suite.executeCommand("mock", "--parse-json", `{"name": "{{ Person.name }}", "age": "{{ Number.number::{18}:{80} }}"}`, "--to-csv-file="+outPath)
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
	_, err := suite.executeCommand("mock", "--parse-json-file", templatePath, "--to-csv-file")
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
	_, err := suite.executeCommand("mock", "--parse-json-file", templatePath, "--to-csv-file="+outPath)
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

func (suite *MockCmdE2ETestSuite) TestCLIShouldOutputToCsvFile_WithGenerate() {
	testName := "Should write CSV file with N data rows when --generate is used"
	tmpDir := suite.T().TempDir()
	outPath := filepath.Join(tmpDir, "result.csv")
	_, err := suite.executeCommand("mock", "--parse-json", `{"name": "{{ Person.name }}"}`, "--generate", "3", "--to-csv-file="+outPath)
	assert.NoError(suite.T(), err, testName)
	data, readErr := os.ReadFile(outPath)
	assert.NoError(suite.T(), readErr, testName)
	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	// 1 header + 3 data rows
	assert.Len(suite.T(), lines, 4, testName)
}

func (suite *MockCmdE2ETestSuite) TestCLIShouldOutputToCsvFile_DeletesPreviousOutput() {
	testName := "Should delete stale CSV output file before writing new one"
	tmpDir := suite.T().TempDir()
	templatePath := filepath.Join(tmpDir, "employee.template.json")
	_ = os.WriteFile(templatePath, []byte(`{"name": "{{ Person.name }}"}`), 0644)
	outPath := filepath.Join(tmpDir, "employee.csv")
	_ = os.WriteFile(outPath, []byte("stale,data\n1,2\n"), 0644)
	_, err := suite.executeCommand("mock", "--parse-json-file", templatePath, "--to-csv-file")
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
	_, err := suite.executeCommand("mock", "--parse-json", `{"name": "{{ Person.name }}"}`, "--to-json-file="+jsonPath, "--to-csv-file="+csvPath)
	assert.NoError(suite.T(), err, testName)
	_, jsonErr := os.Stat(jsonPath)
	assert.NoError(suite.T(), jsonErr, testName)
	_, csvErr := os.Stat(csvPath)
	assert.NoError(suite.T(), csvErr, testName)
}

type MockDateSuite struct {
	suite.Suite
}

func TestMockDateSuite(t *testing.T) {
	suite.Run(t, new(MockDateSuite))
}

func (suite *MockDateSuite) executeCommand(args ...string) (string, error) {
	outBuf := new(bytes.Buffer)
	opts := &cmd.CommandOptions{Out: outBuf}
	rootCmd := cmd.NewRootCmd(opts)
	rootCmd.SetArgs(args)
	err := rootCmd.Execute()
	return outBuf.String(), err
}

func (suite *MockDateSuite) TestDateDate() {
	tests := []struct {
		testName    string
		template    string
		assertRegex string
	}{
		{testName: "default format", template: "{{ Date.date }}", assertRegex: `^\d{4}-\d{2}-\d{2}\n$`},
		{testName: "YYYY format", template: "{{ Date.date:::/YYYY/ }}", assertRegex: `^\d{4}\n$`},
		{testName: "YYYY-MM format", template: "{{ Date.date:::/YYYY-MM/ }}", assertRegex: `^\d{4}-\d{2}\n$`},
	}

	for _, tt := range tests {
		stdOut, err := suite.executeCommand("mock", "--parse-str", tt.template)
		assert.NoError(suite.T(), err, tt.testName)
		assert.Regexp(suite.T(), regexp.MustCompile(tt.assertRegex), stdOut, tt.testName)
	}
}

func (suite *MockDateSuite) TestDateDate_Errors() {
	tests := []struct {
		testName         string
		template         string
		expectedInOutput string
	}{
		{testName: "from after to", template: "{{ Date.date:{2030-01-01}:{2020-01-01}:{} }}", expectedInOutput: "Date.date: 'from' must be before 'to'"},
	}

	for _, tt := range tests {
		stdOut, err := suite.executeCommand("mock", "--parse-str", tt.template)
		assert.NoError(suite.T(), err, tt.testName)
		assert.Contains(suite.T(), stdOut, tt.expectedInOutput, tt.testName)
	}
}

func (suite *MockDateSuite) TestDateTime() {
	tests := []struct {
		testName    string
		template    string
		assertRegex string
	}{
		{testName: "default format", template: "{{ Date.time }}", assertRegex: `^\d{2}:\d{2}:\d{2}\.\d{3}\n$`},
		{testName: "hh:mm format", template: "{{ Date.time:::/hh:mm/ }}", assertRegex: `^\d{2}:\d{2}\n$`},
		{testName: "hh:mm range (default format)", template: "{{ Date.time }}", assertRegex: `^\d{2}:\d{2}:\d{2}\.\d{3}\n$`},
	}

	for _, tt := range tests {
		stdOut, err := suite.executeCommand("mock", "--parse-str", tt.template)
		assert.NoError(suite.T(), err, tt.testName)
		assert.Regexp(suite.T(), regexp.MustCompile(tt.assertRegex), stdOut, tt.testName)
	}
}

func (suite *MockDateSuite) TestDateDatetime() {
	tests := []struct {
		testName    string
		template    string
		assertRegex string
	}{
		{testName: "default format", template: "{{ Date.datetime }}", assertRegex: `^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}\.\d{3}\n$`},
		{testName: "custom format", template: "{{ Date.datetime:::/DD-MM-YYYY hh:mm/ }}", assertRegex: `^\d{2}-\d{2}-\d{4} \d{2}:\d{2}\n$`},
		{testName: "date-only from/to", template: "{{ Date.datetime:{2020-01-01}:{2030-12-31}:{} }}", assertRegex: `^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}\.\d{3}\n$`},
		{testName: "date-only from/to with custom format", template: "{{ Date.datetime:{2020-01-01}:{2030-12-31}:/YYYY-MM-DD/ }}", assertRegex: `^\d{4}-\d{2}-\d{2}\n$`},
	}

	for _, tt := range tests {
		stdOut, err := suite.executeCommand("mock", "--parse-str", tt.template)
		assert.NoError(suite.T(), err, tt.testName)
		assert.Regexp(suite.T(), regexp.MustCompile(tt.assertRegex), stdOut, tt.testName)
	}
}

func (suite *MockDateSuite) TestDateNow() {
	tests := []struct {
		testName    string
		template    string
		assertRegex string
	}{
		{testName: "default format", template: "{{ Date.now }}", assertRegex: `^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}\.\d{3}\n$`},
		{testName: "YYYY-MM format", template: "{{ Date.now:/YYYY-MM/ }}", assertRegex: `^\d{4}-\d{2}\n$`},
		{testName: "YYYY only", template: "{{ Date.now:/YYYY/ }}", assertRegex: `^\d{4}\n$`},
	}

	for _, tt := range tests {
		stdOut, err := suite.executeCommand("mock", "--parse-str", tt.template)
		assert.NoError(suite.T(), err, tt.testName)
		assert.Regexp(suite.T(), regexp.MustCompile(tt.assertRegex), stdOut, tt.testName)
	}
}

// Step 7 — Bare-value rejection e2e tests on MockCmdE2ETestSuite
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
		_, err := suite.executeCommand("mock", "--parse-json", tt.jsonStr, "--to-stdout", "as-json")
		assert.Error(suite.T(), err, tt.testName)
		assert.Contains(suite.T(), err.Error(), "must be wrapped in", tt.testName)
	}
}

// Step 8 — Number.number param edge cases
type MockNumberSuite struct {
	suite.Suite
}

func TestMockNumberSuite(t *testing.T) {
	suite.Run(t, new(MockNumberSuite))
}

func (suite *MockNumberSuite) executeCommand(args ...string) (string, error) {
	outBuf := new(bytes.Buffer)
	opts := &cmd.CommandOptions{Out: outBuf}
	rootCmd := cmd.NewRootCmd(opts)
	rootCmd.SetArgs(args)
	err := rootCmd.Execute()
	return outBuf.String(), err
}

func (suite *MockNumberSuite) TestNumberNumber() {
	intOrFloatRe := regexp.MustCompile(`^-?\d+(\.\d+)?\n$`)
	intRe := regexp.MustCompile(`^-?\d+\n$`)
	twoDecimalRe := regexp.MustCompile(`^-?\d+\.\d{2}\n$`)
	positiveIntRe := regexp.MustCompile(`^\d+\n$`)

	tests := []struct {
		testName       string
		template       string
		assertRe       *regexp.Regexp
		assertContains string
		checkMin       int
		checkMax       int
	}{
		{testName: "no params (defaults)", template: "{{ Number.number }}", assertRe: intOrFloatRe},
		{testName: "empty decimals (default 0)", template: "{{ Number.number:{} }}", assertRe: intRe},
		{testName: "decimals=2", template: "{{ Number.number:{2} }}", assertRe: twoDecimalRe},
		{testName: "decimals=2, min and max empty (defaults)", template: "{{ Number.number:{2}:{}:{} }}", assertRe: twoDecimalRe},
		{testName: "non-numeric decimals (error, inlined)", template: "{{ Number.number:{abc} }}", assertContains: "[Number.number: 'decimals' must be an integer"},
		{testName: "three bare colons (all defaults)", template: "{{ Number.number::: }}", assertRe: intRe},
		{testName: "decimals=0, min=1, max=10", template: "{{ Number.number:{0}:{1}:{10} }}", assertRe: positiveIntRe, checkMin: 1, checkMax: 10},
	}

	for _, tt := range tests {
		stdOut, err := suite.executeCommand("mock", "--parse-str", tt.template)
		assert.NoError(suite.T(), err, tt.testName)
		if tt.assertContains != "" {
			assert.Contains(suite.T(), stdOut, tt.assertContains, tt.testName)
		} else {
			assert.Regexp(suite.T(), tt.assertRe, stdOut, tt.testName)
			if tt.checkMin != 0 || tt.checkMax != 0 {
				trimmed := regexp.MustCompile(`\n$`).ReplaceAllString(stdOut, "")
				var n int
				_, scanErr := fmt.Sscanf(trimmed, "%d", &n)
				assert.NoError(suite.T(), scanErr, tt.testName)
				assert.GreaterOrEqual(suite.T(), n, tt.checkMin, tt.testName)
				assert.LessOrEqual(suite.T(), n, tt.checkMax, tt.testName)
			}
		}
	}
}

// Step 9 — Boolean.booleanWithChance param edge cases
type MockBooleanSuite struct {
	suite.Suite
}

func TestMockBooleanSuite(t *testing.T) {
	suite.Run(t, new(MockBooleanSuite))
}

func (suite *MockBooleanSuite) executeCommand(args ...string) (string, error) {
	outBuf := new(bytes.Buffer)
	opts := &cmd.CommandOptions{Out: outBuf}
	rootCmd := cmd.NewRootCmd(opts)
	rootCmd.SetArgs(args)
	err := rootCmd.Execute()
	return outBuf.String(), err
}

func (suite *MockBooleanSuite) TestBooleanBooleanWithChance() {
	tests := []struct {
		testName       string
		template       string
		exactOutput    string
		assertContains string
	}{
		{testName: "chance = 100", template: "{{ Boolean.booleanWithChance:{100} }}", exactOutput: "true\n"},
		{testName: "chance = 0", template: "{{ Boolean.booleanWithChance:{0} }}", exactOutput: "false\n"},
		{testName: "chance empty (error, inlined)", template: "{{ Boolean.booleanWithChance:{} }}", assertContains: "[Boolean.booleanWithChance: 'chance' parameter is required"},
		{testName: "chance non-numeric (error, inlined)", template: "{{ Boolean.booleanWithChance:{abc} }}", assertContains: "[Boolean.booleanWithChance: 'chance' must be an integer"},
		{testName: "no param (error, inlined)", template: "{{ Boolean.booleanWithChance }}", assertContains: "[Boolean.booleanWithChance: 'chance' parameter is required"},
	}

	for _, tt := range tests {
		stdOut, err := suite.executeCommand("mock", "--parse-str", tt.template)
		assert.NoError(suite.T(), err, tt.testName)
		if tt.assertContains != "" {
			assert.Contains(suite.T(), stdOut, tt.assertContains, tt.testName)
		} else if tt.exactOutput != "" {
			assert.Equal(suite.T(), tt.exactOutput, stdOut, tt.testName)
		}
	}
}

// Step 10 — Lorem.* param edge cases
type MockLoremSuite struct {
	suite.Suite
}

func TestMockLoremSuite(t *testing.T) {
	suite.Run(t, new(MockLoremSuite))
}

func (suite *MockLoremSuite) executeCommand(args ...string) (string, error) {
	outBuf := new(bytes.Buffer)
	opts := &cmd.CommandOptions{Out: outBuf}
	rootCmd := cmd.NewRootCmd(opts)
	rootCmd.SetArgs(args)
	err := rootCmd.Execute()
	return outBuf.String(), err
}

func (suite *MockLoremSuite) TestLorem() {
	tests := []struct {
		testName       string
		template       string
		assertNonEmpty bool
		assertContains string
	}{
		{testName: "paragraph default (1 sentence)", template: "{{ Lorem.paragraph:{1} }}", assertNonEmpty: true},
		{testName: "paragraph N=3", template: "{{ Lorem.paragraph:{3} }}", assertNonEmpty: true},
		{testName: "paragraph empty param (error, inlined)", template: "{{ Lorem.paragraph:{} }}", assertContains: "[Lorem.paragraph: 'sentences' parameter is required"},
		{testName: "paragraphs N=2", template: "{{ Lorem.paragraphs:{2} }}", assertContains: "\n"},
		{testName: "sentence N=5", template: "{{ Lorem.sentence:{5} }}", assertNonEmpty: true},
		{testName: "sentences N=3", template: "{{ Lorem.sentences:{3} }}", assertContains: "\n"},
		{testName: "word (no params)", template: "{{ Lorem.word }}", assertNonEmpty: true},
		{testName: "words N=4", template: "{{ Lorem.words:{4} }}", assertNonEmpty: true},
		{testName: "paragraph no param (error, inlined)", template: "{{ Lorem.paragraph }}", assertContains: "[Lorem.paragraph: 'sentences' parameter is required"},
		{testName: "paragraphs no param (error, inlined)", template: "{{ Lorem.paragraphs }}", assertContains: "[Lorem.paragraphs: 'paragraphs' parameter is required"},
		{testName: "sentence no param (error, inlined)", template: "{{ Lorem.sentence }}", assertContains: "[Lorem.sentence: 'words' parameter is required"},
		{testName: "sentences no param (error, inlined)", template: "{{ Lorem.sentences }}", assertContains: "[Lorem.sentences: 'sentences' parameter is required"},
		{testName: "words no param (error, inlined)", template: "{{ Lorem.words }}", assertContains: "[Lorem.words: 'words' parameter is required"},
		{testName: "paragraphs empty param (error, inlined)", template: "{{ Lorem.paragraphs:{} }}", assertContains: "[Lorem.paragraphs: 'paragraphs' parameter is required"},
		{testName: "sentence empty param (error, inlined)", template: "{{ Lorem.sentence:{} }}", assertContains: "[Lorem.sentence: 'words' parameter is required"},
		{testName: "sentences empty param (error, inlined)", template: "{{ Lorem.sentences:{} }}", assertContains: "[Lorem.sentences: 'sentences' parameter is required"},
		{testName: "words empty param (error, inlined)", template: "{{ Lorem.words:{} }}", assertContains: "[Lorem.words: 'words' parameter is required"},
	}

	for _, tt := range tests {
		stdOut, err := suite.executeCommand("mock", "--parse-str", tt.template)
		assert.NoError(suite.T(), err, tt.testName)
		if tt.assertNonEmpty {
			assert.NotEmpty(suite.T(), stdOut, tt.testName)
		}
		if tt.assertContains != "" {
			assert.Contains(suite.T(), stdOut, tt.assertContains, tt.testName)
		}
	}
}

// Step 11 — Regex.regex param edge cases
type MockRegexSuite struct {
	suite.Suite
}

func TestMockRegexSuite(t *testing.T) {
	suite.Run(t, new(MockRegexSuite))
}

func (suite *MockRegexSuite) executeCommand(args ...string) (string, error) {
	outBuf := new(bytes.Buffer)
	opts := &cmd.CommandOptions{Out: outBuf}
	rootCmd := cmd.NewRootCmd(opts)
	rootCmd.SetArgs(args)
	err := rootCmd.Execute()
	return outBuf.String(), err
}

func (suite *MockRegexSuite) TestRegexRegex() {
	tests := []struct {
		testName       string
		template       string
		assertRegex    string
		assertContains string
		exactOutput    string
	}{
		{testName: "simple pattern", template: "{{ Regex.regex:/[a-z]{3}/ }}", assertRegex: `^[a-z]{3}\n$`},
		{testName: "digits pattern", template: "{{ Regex.regex:/[0-9]{4}/ }}", assertRegex: `^\d{4}\n$`},
		{testName: "empty regex (generates empty string)", template: "{{ Regex.regex:// }}", exactOutput: "\n"},
		{testName: "no params (returns error, inlined)", template: "{{ Regex.regex }}", assertContains: "[regex function requires"},
		{testName: "bare param (post-enforcement error)", template: "{{ Regex.regex:[a-z]{3} }}", assertContains: "must be wrapped in"},
		{testName: "pattern not wrapped in slashes", template: "{{ Regex.regex:{[a-z]{3}} }}", assertContains: "must be wrapped in"},
	}

	for _, tt := range tests {
		stdOut, err := suite.executeCommand("mock", "--parse-str", tt.template)
		assert.NoError(suite.T(), err, tt.testName)
		if tt.assertRegex != "" {
			assert.Regexp(suite.T(), regexp.MustCompile(tt.assertRegex), stdOut, tt.testName)
		}
		if tt.assertContains != "" {
			assert.Contains(suite.T(), stdOut, tt.assertContains, tt.testName)
		}
		if tt.exactOutput != "" {
			assert.Equal(suite.T(), tt.exactOutput, stdOut, tt.testName)
		}
	}
}

// Step 12 — Date.* additional param edge cases
func (suite *MockDateSuite) TestDateDate_Params() {
	tests := []struct {
		testName    string
		template    string
		assertRegex string
		assertYear  string
	}{
		{testName: "from only", template: "{{ Date.date:{2020-01-01}:{}:{} }}", assertRegex: `^\d{4}-\d{2}-\d{2}\n$`},
		{testName: "from and to", template: "{{ Date.date:{2025-01-01}:{2025-12-31}:{} }}", assertRegex: `^\d{4}-\d{2}-\d{2}\n$`, assertYear: "2025"},
		{testName: "all three params", template: "{{ Date.date:{2020-01-01}:{2020-12-31}:/YYYY/ }}", assertRegex: `^2020\n$`},
		{testName: "empty from and to", template: "{{ Date.date:{}:{}:{} }}", assertRegex: `^\d{4}-\d{2}-\d{2}\n$`},
		{testName: "invalid from date (silently uses default)", template: "{{ Date.date:{not-a-date}:{2030-01-01}:{} }}", assertRegex: `^\d{4}-\d{2}-\d{2}\n$`},
	}

	for _, tt := range tests {
		stdOut, err := suite.executeCommand("mock", "--parse-str", tt.template)
		assert.NoError(suite.T(), err, tt.testName)
		assert.Regexp(suite.T(), regexp.MustCompile(tt.assertRegex), stdOut, tt.testName)
		if tt.assertYear != "" {
			assert.True(suite.T(), len(stdOut) >= 4 && stdOut[:4] == tt.assertYear, tt.testName)
		}
	}
}

func (suite *MockDateSuite) TestDateDate_BareParams() {
	tests := []struct {
		testName       string
		template       string
		assertContains string
	}{
		{testName: "invalid format (bare param, caught by extractMockMethod)", template: "{{ Date.date:{}:{}:{YYYY-MM-DD} }}", assertContains: "must be wrapped in"},
	}

	for _, tt := range tests {
		stdOut, err := suite.executeCommand("mock", "--parse-str", tt.template)
		assert.NoError(suite.T(), err, tt.testName)
		assert.Contains(suite.T(), stdOut, tt.assertContains, tt.testName)
	}
}

func (suite *MockDateSuite) TestDatetime_Params() {
	tests := []struct {
		testName    string
		template    string
		assertRegex string
	}{
		{testName: "all params empty (defaults)", template: "{{ Date.datetime:{}:{}:{} }}", assertRegex: `^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}\.\d{3}\n$`},
		{testName: "from/to date-only format", template: "{{ Date.datetime:{2020-01-01}:{2030-12-31}:{} }}", assertRegex: `^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}\.\d{3}\n$`},
		{testName: "from/to datetime format", template: "{{ Date.datetime:{2020-01-01T08:00}:{2020-01-01T20:00}:{} }}", assertRegex: `^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}\.\d{3}\n$`},
		{testName: "custom format", template: "{{ Date.datetime:{2020-01-01}:{2020-12-31}:/YYYY-MM-DD/ }}", assertRegex: `^\d{4}-\d{2}-\d{2}\n$`},
	}

	for _, tt := range tests {
		stdOut, err := suite.executeCommand("mock", "--parse-str", tt.template)
		assert.NoError(suite.T(), err, tt.testName)
		assert.Regexp(suite.T(), regexp.MustCompile(tt.assertRegex), stdOut, tt.testName)
	}
}

func (suite *MockDateSuite) TestDatetime_Errors() {
	tests := []struct {
		testName         string
		template         string
		expectedInOutput string
	}{
		{testName: "from after to", template: "{{ Date.datetime:{2030-01-01}:{2020-01-01}:{} }}", expectedInOutput: "Date.datetime: 'from' must be before 'to'"},
	}

	for _, tt := range tests {
		stdOut, err := suite.executeCommand("mock", "--parse-str", tt.template)
		assert.NoError(suite.T(), err, tt.testName)
		assert.Contains(suite.T(), stdOut, tt.expectedInOutput, tt.testName)
	}
}

func (suite *MockDateSuite) TestDateTime_Params() {
	tests := []struct {
		testName         string
		template         string
		assertRegex      string
		expectedInOutput string
	}{
		{testName: "format override /hh:mm/", template: "{{ Date.time:{08:00}:{20:00}:/hh:mm/ }}", assertRegex: `^\d{2}:\d{2}\n$`},
		{testName: "all params empty (defaults)", template: "{{ Date.time:{}:{}:{} }}", assertRegex: `^\d{2}:\d{2}:\d{2}\.\d{3}\n$`},
		{testName: "from after to (error inlined)", template: "{{ Date.time:{20:00}:{08:00}:{} }}", expectedInOutput: "Date.time: 'from' must be before 'to'"},
	}

	for _, tt := range tests {
		stdOut, err := suite.executeCommand("mock", "--parse-str", tt.template)
		assert.NoError(suite.T(), err, tt.testName)
		if tt.assertRegex != "" {
			assert.Regexp(suite.T(), regexp.MustCompile(tt.assertRegex), stdOut, tt.testName)
		}
		if tt.expectedInOutput != "" {
			assert.Contains(suite.T(), stdOut, tt.expectedInOutput, tt.testName)
		}
	}
}

// Step 13 — parse-json bare values (covered via TestCLIShouldInlineError_BareParamValues)

// Step 14 — parse-json nested structures
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
		stdOut, err := suite.executeCommand("mock", "--parse-json", tt.jsonStr, "--to-stdout", "as-json")
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

// Step 15 — parse-json --generate flag
func (suite *MockCmdE2ETestSuite) TestCLIParseJson_GenerateFlag() {
	tests := []struct {
		testName    string
		args        []string
		assertRegex string
	}{
		{
			testName:    "generate=3, output is JSON array with 3 elements",
			args:        []string{"mock", "--parse-json", `{"name":"{{ Person.name }}"}`, "--generate", "3", "--to-stdout", "as-json"},
			assertRegex: `^\[`,
		},
		{
			testName:    "generate=1 (default), output is JSON object not array",
			args:        []string{"mock", "--parse-json", `{"name":"{{ Person.name }}"}`, "--to-stdout", "as-json"},
			assertRegex: `^\{`,
		},
	}

	for _, tt := range tests {
		stdOut, err := suite.executeCommand(tt.args...)
		assert.NoError(suite.T(), err, tt.testName)
		assert.Regexp(suite.T(), regexp.MustCompile(tt.assertRegex), stdOut, tt.testName)
	}
}

// Step 16 — processStr edge cases
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

type MockUUIDSuite struct {
	suite.Suite
}

func TestMockUUIDSuite(t *testing.T) {
	suite.Run(t, new(MockUUIDSuite))
}

func (suite *MockUUIDSuite) executeCommand(args ...string) (string, error) {
	outBuf := new(bytes.Buffer)
	opts := &cmd.CommandOptions{Out: outBuf}
	rootCmd := cmd.NewRootCmd(opts)
	rootCmd.SetArgs(args)
	err := rootCmd.Execute()
	return outBuf.String(), err
}

func (suite *MockUUIDSuite) TestUUIDv7() {
	tests := []struct {
		testName    string
		template    string
		assertRegex string
	}{
		{
			testName:    "uuidv7 matches canonical UUID v7 format",
			template:    "{{ UUID.uuidv7 }}",
			assertRegex: `^[0-9a-f]{8}-[0-9a-f]{4}-7[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}\n$`,
		},
	}

	for _, tt := range tests {
		stdOut, err := suite.executeCommand("mock", "--parse-str", tt.template)
		assert.NoError(suite.T(), err, tt.testName)
		assert.Regexp(suite.T(), regexp.MustCompile(tt.assertRegex), stdOut, tt.testName)
	}
}

// --- Concurrent path E2E tests ---

// TestCLIConcurrent_LargeGenerateAsJson verifies the worker pool produces a valid
// JSON array with the correct number of items when --generate exceeds the
// concurrencyThreshold.
func (suite *MockCmdE2ETestSuite) TestCLIConcurrent_LargeGenerateAsJson() {
	testName := "concurrent: --generate 50000 produces valid JSON array of 50000 objects"
	stdOut, err := suite.executeCommand(
		"mock",
		"--parse-json", `{"name":"{{ Person.name }}"}`,
		"--generate", "50000",
		"--to-stdout", "as-json",
	)
	assert.NoError(suite.T(), err, testName)

	var result []map[string]any
	assert.NoError(suite.T(), json.Unmarshal([]byte(strings.TrimSpace(stdOut)), &result), testName)
	assert.Len(suite.T(), result, 50000, testName)
	for _, item := range result {
		_, hasName := item["name"]
		assert.True(suite.T(), hasName, testName)
	}
}

// TestCLIConcurrent_SingleObjectShapePreserved verifies that when --generate 1 is
// used the output is a bare JSON object (not an array), even on the sequential path.
func (suite *MockCmdE2ETestSuite) TestCLIConcurrent_SingleObjectShapePreserved() {
	testName := "generate=1 produces a bare object, not an array"
	stdOut, err := suite.executeCommand(
		"mock",
		"--parse-json", `{"name":"{{ Person.name }}"}`,
		"--generate", "1",
		"--to-stdout", "as-json",
	)
	assert.NoError(suite.T(), err, testName)
	trimmed := strings.TrimSpace(stdOut)
	// Must NOT start with '[' — should be a plain object
	assert.True(suite.T(), len(trimmed) > 0 && trimmed[0] == '{', testName)
	var result map[string]any
	assert.NoError(suite.T(), json.Unmarshal([]byte(trimmed), &result), testName)
	_, hasName := result["name"]
	assert.True(suite.T(), hasName, testName)
}

// TestCLIConcurrent_LargeGenerateAsCsv verifies the worker pool produces valid CSV
// with a header row and the correct number of data rows.
func (suite *MockCmdE2ETestSuite) TestCLIConcurrent_LargeGenerateAsCsv() {
	testName := "concurrent: --generate 50000 produces valid CSV with 50000 data rows"
	stdOut, err := suite.executeCommand(
		"mock",
		"--parse-json", `{"name":"{{ Person.name }}"}`,
		"--generate", "50000",
		"--to-stdout", "as-csv",
	)
	assert.NoError(suite.T(), err, testName)
	lines := strings.Split(strings.TrimSpace(stdOut), "\n")
	// 1 header + 50000 data rows
	assert.Len(suite.T(), lines, 50001, testName)
	assert.Equal(suite.T(), "name", lines[0], testName)
}

// TestCLIConcurrent_LargeGenerateToJsonFile verifies atomic file writing: the final
// file exists, contains valid JSON, and no leftover temp file remains.
func (suite *MockCmdE2ETestSuite) TestCLIConcurrent_LargeGenerateToJsonFile() {
	testName := "concurrent: --generate 50000 --to-json-file produces valid JSON file with no temp remainder"
	tmpDir := suite.T().TempDir()
	outPath := filepath.Join(tmpDir, "result.json")

	_, err := suite.executeCommand(
		"mock",
		"--parse-json", `{"name":"{{ Person.name }}"}`,
		"--generate", "50000",
		"--to-json-file="+outPath,
	)
	assert.NoError(suite.T(), err, testName)

	// Final file must exist
	data, readErr := os.ReadFile(outPath)
	assert.NoError(suite.T(), readErr, testName)

	var result []map[string]any
	assert.NoError(suite.T(), json.Unmarshal(data, &result), testName)
	assert.Len(suite.T(), result, 50000, testName)

	// No temp files should remain
	entries, _ := os.ReadDir(tmpDir)
	for _, e := range entries {
		assert.False(suite.T(), strings.HasPrefix(e.Name(), ".ktns-tmp-"), "temp file should not remain: %s", e.Name())
	}
}

// TestCLIConcurrent_DebugFlagSucceeds verifies that --debug does not corrupt stdout
// output and the command still succeeds.
func (suite *MockCmdE2ETestSuite) TestCLIConcurrent_DebugFlagSucceeds() {
	testName := "concurrent: --debug flag succeeds and stdout contains only valid JSON"
	stdOut, err := suite.executeCommand(
		"mock",
		"--parse-json", `{"name":"{{ Person.name }}"}`,
		"--generate", "50000",
		"--to-stdout", "as-json",
		"--debug",
	)
	assert.NoError(suite.T(), err, testName)

	// opts.Out should contain only JSON — no debug lines (debug writes to stderr)
	trimmed := strings.TrimSpace(stdOut)
	assert.False(suite.T(), strings.Contains(trimmed, "[debug]"), testName)

	var result []map[string]any
	assert.NoError(suite.T(), json.Unmarshal([]byte(trimmed), &result), testName)
	assert.Len(suite.T(), result, 50000, testName)
}
