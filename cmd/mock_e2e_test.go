package cmd_test

import (
	"bytes"
	"os"
	"regexp"
	"testing"

	"github.com/lfsc09/k-test-n-stress/cmd"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

const tempInputFilesDir = ".testdata/input"
const tempOutputFilesDir = ".testdata/output"

type MockCmdE2ETestSuite struct {
	suite.Suite
}

func TestMockCmdE2ESuite(t *testing.T) {
	suite.Run(t, new(MockCmdE2ETestSuite))
}

func (suite *MockCmdE2ETestSuite) SetupSuite() {
	// Create temp directories for input and output files
	err := os.MkdirAll(tempInputFilesDir, os.ModePerm)
	assert.NoError(suite.T(), err, "Failed to create temp input files directory")
	err = os.MkdirAll(tempOutputFilesDir, os.ModePerm)
	assert.NoError(suite.T(), err, "Failed to create temp output files directory")
}

func (suite *MockCmdE2ETestSuite) TearDownSuite() {
	// Clean up temp directories after tests
	err := os.RemoveAll(tempInputFilesDir)
	assert.NoError(suite.T(), err, "Failed to remove temp input files directory")
	err = os.RemoveAll(tempOutputFilesDir)
	assert.NoError(suite.T(), err, "Failed to remove temp output files directory")
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
func (suite *MockCmdE2ETestSuite) TestCLINothingToBeParsed() {
	testName := "Should raise error when nothing to be parsed"
	_, err := suite.executeCommand("mock")
	assert.Error(suite.T(), err, testName)
}

func (suite *MockCmdE2ETestSuite) TestCLIMultipleParseFlagsSimultaneously() {
	tests := []struct {
		testName string
		input    []string
	}{
		{
			testName: "Should raise error when both --parse-str and --parse-json-file are provided",
			input:    []string{"mock", "--parse-str", "Hello {{ Person.name }}", "--parse-json-file", "test.json"},
		},
		{
			testName: "Should raise error when both --parse-str and --parse-json are provided",
			input:    []string{"mock", "--parse-str", "Hello {{ Person.name }}", "--parse-json", "' {\"name\": \"{{ Person.name }}\"} '"},
		},
		{
			testName: "Should raise error when both --parse-json and --parse-json-file are provided",
			input:    []string{"mock", "--parse-json", "' {\"name\": \"{{ Person.name }}\"} '", "--parse-json-file", "test.json"},
		},
		{
			testName: "Should raise error when all three --parse-str, --parse-json and --parse-json-file are provided",
			input:    []string{"mock", "--parse-str", "Hello {{ Person.name }}", "--parse-json", "' {\"name\": \"{{ Person.name }}\"} '", "--parse-json-file", "test.json"},
		},
	}

	for _, test := range tests {
		_, err := suite.executeCommand(test.input...)
		assert.Error(suite.T(), err, test.testName)
	}
}

func (suite *MockCmdE2ETestSuite) TestCLIParseStrWithGenerateFlagInvalidUse() {
	tests := []struct {
		testName string
		input    []string
	}{
		{
			testName: "Should raise error when using --generate with --parse-str",
			input:    []string{"mock", "--parse-str", "Hello {{ Person.name }}", "--generate", "5"},
		},
		{
			testName: "Should raise error when --generate value is equal to 0",
			input:    []string{"mock", "--parse-json", "' {\"name\": \"{{ Person.name }}\"} '", "--generate", "0", "--to-json-stdout"},
		},
		{
			testName: "Should raise error when --generate value is negative",
			input:    []string{"mock", "--parse-json", "' {\"name\": \"{{ Person.name }}\"} '", "--generate", "-1", "--to-json-stdout"},
		},
	}

	for _, test := range tests {
		_, err := suite.executeCommand(test.input...)
		assert.Error(suite.T(), err, test.testName)
	}
}

func (suite *MockCmdE2ETestSuite) TestCLIParseJsonFileWithInvalidExtension() {
	tests := []struct {
		testName string
		input    []string
	}{
		{
			testName: "Should raise error when --parse-json-file does not end with .template.json",
			input:    []string{"mock", "--parse-json-file", "/tmp/employee.json", "--to-json-stdout"},
		},
	}

	for _, test := range tests {
		_, err := suite.executeCommand(test.input...)
		assert.Error(suite.T(), err, test.testName)
	}
}

func (suite *MockCmdE2ETestSuite) TestCLIParseWithoutOutputFlag() {
	tests := []struct {
		testName string
		input    []string
	}{
		{
			testName: "Should raise error when using --parse-json without an output flag",
			input:    []string{"mock", "--parse-json", "' {\"name\": \"{{ Person.name }}\"} '"},
		},
		{
			testName: "Should raise error when using --parse-json-file without an output flag",
			input:    []string{"mock", "--parse-json-file", "/tmp/employee.template.json"},
		},
	}

	for _, test := range tests {
		_, err := suite.executeCommand(test.input...)
		assert.Error(suite.T(), err, test.testName)
	}
}

/*
VALID STATES
*/

func (suite *MockCmdE2ETestSuite) TestCLIParseStr_ValidInputs() {
	tests := []struct {
		testName    string
		input       []string
		assertRegex string
	}{
		{
			testName:    "Should mock from --parse-str (literal value)",
			input:       []string{"mock", "--parse-str", "Hello world"},
			assertRegex: `Hello world`,
		},
		{
			testName:    "Should mock from --parse-str (dynamic value)",
			input:       []string{"mock", "--parse-str", "Hello {{ Person.name }}"},
			assertRegex: `Hello .+`,
		},
	}

	for _, test := range tests {
		stdOut, err := suite.executeCommand(test.input...)
		assert.NoError(suite.T(), err, test.testName)
		assert.Regexp(suite.T(), regexp.MustCompile(test.assertRegex), stdOut, test.testName)
	}
}

func (suite *MockCmdE2ETestSuite) TestCLIParseJson_ValidInputs() {
	tests := []struct {
		testName    string
		args        []string
		assertRegex string
	}{
		{
			testName:    "Should mock from --parse-json with simple JSON object [literal value]",
			args:        []string{"mock", "--parse-json", `{"name":"John Smith"}`, "--to-stdout"},
			assertRegex: `^\{"name":"John Smith"\}$`,
		},
		{
			testName:    "Should mock from --parse-json with simple JSON object [dynamic value]",
			args:        []string{"mock", "--parse-json", `{"name":"{{ Person.name }}"}`, "--to-stdout"},
			assertRegex: `^\{"name":".+"\}$`,
		},
		{
			testName:    "Should mock from --parse-json with --generate producing a JSON array of 2 objects",
			args:        []string{"mock", "--parse-json", `{"name":"{{ Person.name }}"}`, "--generate", "2", "--to-stdout"},
			assertRegex: `^\[\{"name":".+"\},\{"name":".+"\}\]$`,
		},
		{
			testName:    "Should mock from --parse-json with simple JSON object [fixed array values]",
			args:        []string{"mock", "--parse-json", `{"names":["John Smith", "{{ Person.name }}"]}`, "--to-stdout"},
			assertRegex: `^\{"names":\["John Smith",".+"\]\}$`,
		},
		{
			testName:    "Should mock from --parse-json with nested JSON object [dynamic value]",
			args:        []string{"mock", "--parse-json", `{"employee":{"name":"{{ Person.name }}"}}`, "--to-stdout"},
			assertRegex: `^\{"employee":\{"name":".+"\}\}$`,
		},
		{
			testName:    "Should mock from --parse-json with simple JSON object [dynamic array values]",
			args:        []string{"mock", "--parse-json", `{"names[2]":"{{ Person.name }}"}`, "--to-stdout"},
			assertRegex: `^\{"names":\[".+",".+"\]\}$`,
		},
		{
			testName:    "Should mock from --parse-json with nested JSON object [dynamic array value]",
			args:        []string{"mock", "--parse-json", `{"employee[2]":{"name":"{{ Person.name }}"}}`, "--to-stdout"},
			assertRegex: `^\{"employee":\[\{"name":".+"\},\{"name":".+"\}\]\}$`,
		},
		{
			testName:    "Should mock from --parse-json with --generate producing a JSON root array of 2 objects [dynamic array value]",
			args:        []string{"mock", "--parse-json", `{"employee[2]":{"name":"{{ Person.name }}"}}`, "--generate", "2", "--to-stdout"},
			assertRegex: `^\[\{"employee":\[\{"name":".+"\},\{"name":".+"\}\]\},\{"employee":\[\{"name":".+"\},\{"name":".+"\}\]\}\]$`,
		},
	}

	for _, tt := range tests {
		stdOut, err := suite.executeCommand(tt.args...)
		assert.NoError(suite.T(), err, tt.testName)
		assert.Regexp(suite.T(), regexp.MustCompile(tt.assertRegex), stdOut, tt.testName)
	}
}

func (suite *MockCmdE2ETestSuite) TestCLIParseJsonFile_ValidInputs() {
	tests := []struct {
		testName     string
		args         []string
		jsonTemplate string
		assertRegex  string
	}{
		{
			testName:     "Should mock from --parse-json-file with simple JSON object [literal value]",
			args:         []string{"mock", "--parse-json-file", "", "--to-stdout"},
			jsonTemplate: `{"name":"John Smith"}`,
			assertRegex:  `^\{"name":"John Smith"\}$`,
		},
		{
			testName:     "Should mock from --parse-json-file with simple JSON object [dynamic value]",
			args:         []string{"mock", "--parse-json-file", "", "--to-stdout"},
			jsonTemplate: `{"name":"{{ Person.name }}"}`,
			assertRegex:  `^\{"name":".+"\}$`,
		},
		{
			testName:     "Should mock from --parse-json-file with --generate producing a JSON root array of 2 objects",
			args:         []string{"mock", "--parse-json-file", "", "--generate", "2", "--to-stdout"},
			jsonTemplate: `{"name":"{{ Person.name }}"}`,
			assertRegex:  `^\[\{"name":".+"\},\{"name":".+"\}\]$`,
		},
		{
			testName:     "Should mock from --parse-json-file with simple JSON object [fixed array values]",
			args:         []string{"mock", "--parse-json-file", "", "--to-stdout"},
			jsonTemplate: `{"names":["John Smith","{{ Person.name }}"]}`,
			assertRegex:  `^\{"names":\["John Smith",".+"\]\}$`,
		},
		{
			testName:     "Should mock from --parse-json-file with nested JSON object [dynamic value]",
			args:         []string{"mock", "--parse-json-file", "", "--to-stdout"},
			jsonTemplate: `{"employee":{"name":"{{ Person.name }}"}}`,
			assertRegex:  `^\{"employee":\{"name":".+"\}\}$`,
		},
		{
			testName:     "Should mock from --parse-json-file with simple JSON object [dynamic array values]",
			args:         []string{"mock", "--parse-json-file", "", "--to-stdout"},
			jsonTemplate: `{"names[2]":"{{ Person.name }}"}`,
			assertRegex:  `^\{"names":\[".+",".+"\]\}$`,
		},
		{
			testName:     "Should mock from --parse-json-file with nested JSON object [dynamic array value]",
			args:         []string{"mock", "--parse-json-file", "", "--to-stdout"},
			jsonTemplate: `{"employee[2]":{"name":"{{ Person.name }}"}}`,
			assertRegex:  `^\{"employee":\[\{"name":".+"\},\{"name":".+"\}\]\}$`,
		},
		{
			testName:     "Should mock from --parse-json-file with --generate producing a JSON root array of 2 objects [dynamic array value]",
			args:         []string{"mock", "--parse-json-file", "", "--generate", "2", "--to-stdout"},
			jsonTemplate: `{"employee[2]":{"name":"{{ Person.name }}"}}`,
			assertRegex:  `^\[\{"employee":\[\{"name":".+"\},\{"name":".+"\}\]\},\{"employee":\[\{"name":".+"\},\{"name":".+"\}\]\}\]$`,
		},
	}

	for _, tt := range tests {
		// Create template temp file
		tmpFile, err := os.CreateTemp(tempInputFilesDir, "*.template.json")
		assert.NoError(suite.T(), err, tt.testName)
		defer os.Remove(tmpFile.Name())
		_, err = tmpFile.WriteString(tt.jsonTemplate)
		assert.NoError(suite.T(), err, tt.testName)
		tmpFile.Close()

		// Set the generated temp file path as the argument for --parse-json-file
		// --parse-json-file is always the 3rd argument in the test cases
		tt.args[2] = tmpFile.Name()

		stdOut, err := suite.executeCommand(tt.args...)
		assert.NoError(suite.T(), err, tt.testName)
		assert.Regexp(suite.T(), regexp.MustCompile(tt.assertRegex), stdOut, tt.testName)
	}
}

func (suite *MockCmdE2ETestSuite) TestCLIParseJsonToFile_ValidInputs() {
	outputFilename := tempOutputFilesDir + "/output.json"
	tests := []struct {
		testName    string
		args        []string
		assertRegex string
	}{
		{
			testName:    "Should mock from --parse-json with --generate producing a JSON root array of 2 objects to an output file [dynamic array value]",
			args:        []string{"mock", "--parse-json", `{"employee[2]":{"name":"{{ Person.name }}"}}`, "--generate", "2", "--to-file", outputFilename},
			assertRegex: `^\[\{"employee":\[\{"name":".+"\},\{"name":".+"\}\]\},\{"employee":\[\{"name":".+"\},\{"name":".+"\}\]\}\]$`,
		},
	}

	for _, tt := range tests {
		_, err := suite.executeCommand(tt.args...)
		assert.NoError(suite.T(), err, tt.testName)

		// Read the output file and assert its content
		outputBytes, err := os.ReadFile(outputFilename)
		assert.NoError(suite.T(), err, tt.testName)
		output := string(outputBytes)
		assert.Regexp(suite.T(), regexp.MustCompile(tt.assertRegex), output, tt.testName)
		os.Remove(outputFilename)
	}
}
