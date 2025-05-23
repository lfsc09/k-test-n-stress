package cmd_test

import (
	"bytes"
	"testing"

	"github.com/lfsc09/k-test-n-stress/cmd"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type MockCmdE2ETestSuite struct {
	suite.Suite
	// ViperInstance *viper.Viper
}

func TestMockCmdTestSuite(t *testing.T) {
	suite.Run(t, new(MockCmdE2ETestSuite))
}

// Test the command line interface (CLI) of the application. (Specifically for the mock command)
func (suite *MockCmdE2ETestSuite) executeCommand(args ...string) (string, error) {
	// Create a buffer to capture the output
	outBuf := new(bytes.Buffer)

	opts := &cmd.CommandOptions{
		Out:   outBuf,
		Viper: viper.New(),
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
			testName: "both --parseStr and --parseFile",
			input:    []string{"mock", "--parseStr", "Hello {{ Person.name }}", "--parseFiles", "test.json"},
		},
		{
			testName: "both --parseStr and --parseJson",
			input:    []string{"mock", "--parseStr", "Hello {{ Person.name }}", "--parseJson", "' {\"name\": \"{{ Person.name }}\"} '"},
		},
		{
			testName: "both --parseJson and --parseFile",
			input:    []string{"mock", "--parseJson", "' {\"name\": \"{{ Person.name }}\"} '", "--parseFiles", "test.json"},
		},
		{
			testName: "all three --parseStr, --parseJson and --parseFile",
			input:    []string{"mock", "--parseStr", "Hello {{ Person.name }}", "--parseJson", "' {\"name\": \"{{ Person.name }}\"} '", "--parseFiles", "test.json"},
		},
	}
	for _, test := range tests {
		_, err := suite.executeCommand(test.input...)
		assert.Error(suite.T(), err, test.testName)
		assert.EqualError(suite.T(), err, "provide only one of the three options: --parseJson, --parseFiles or --parseStr", test.testName)
	}
}

func (suite *MockCmdE2ETestSuite) TestCLIShouldRaiseError_InvalidUseOfParseFiles() {
	testName := "Should raise error when multiple args in --parseFiles"
	_, err := suite.executeCommand("mock", "--parseFiles", "test.json", "test2.json")
	assert.Error(suite.T(), err, testName)
	assert.EqualError(suite.T(), err, "you passed multiple files to --parseFiles without quotes. Did you mean: --parseFiles \"*.template.json\"?", testName)
}

func (suite *MockCmdE2ETestSuite) TestCLIShouldRaiseError_PreserveFolderStructureFlagInvalidUse() {
	tests := []struct {
		testName string
		input    []string
	}{
		{
			testName: "--preserveFolderStructure with --parseStr",
			input:    []string{"mock", "--parseStr", "Hello {{ Person.name }}", "--preserveFolderStructure"},
		},
		{
			testName: "--preserveFolderStructure with --parseJson",
			input:    []string{"mock", "--parseJson", "' {\"name\": \"{{ Person.name }}\"} '", "--preserveFolderStructure"},
		},
	}
	for _, test := range tests {
		_, err := suite.executeCommand(test.input...)
		assert.Error(suite.T(), err, test.testName)
		assert.EqualError(suite.T(), err, "--preserveFolderStructure option is only available when using --parseFiles", test.testName)
	}
}

func (suite *MockCmdE2ETestSuite) TestCLIShouldRaiseError_GenerateFlagInvalidUse() {
	tests := []struct {
		testName string
		input    []string
	}{
		{
			testName: "--generate with --parseStr",
			input:    []string{"mock", "--parseStr", "Hello {{ Person.name }}", "--generate", "5"},
		},
		{
			testName: "--generate with --parseFiles",
			input:    []string{"mock", "--parseFiles", "test.json", "--generate", "5"},
		},
	}
	for _, test := range tests {
		_, err := suite.executeCommand(test.input...)
		assert.Error(suite.T(), err, test.testName)
		assert.EqualError(suite.T(), err, "--generate option is only available when using --parseJson", test.testName)
	}
}

func (suite *MockCmdE2ETestSuite) TestCLIShouldRaiseError_GenerateFlagInvalidValues() {
	tests := []struct {
		testName string
		input    []string
	}{
		{
			testName: "--generate with value equal to 0",
			input:    []string{"mock", "--parseJson", "' {\"name\": \"{{ Person.name }}\"} '", "--generate", "0"},
		},
		{
			testName: "--generate with negative value",
			input:    []string{"mock", "--parseJson", "' {\"name\": \"{{ Person.name }}\"} '", "--generate", "-1"},
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
	assert.Contains(suite.T(), stdOut, "Time.", testName)
	assert.Contains(suite.T(), stdOut, "UUID.", testName)
	assert.Contains(suite.T(), stdOut, "UserAgent.", testName)
}

func (suite *MockCmdE2ETestSuite) TestCLIShouldMockFromParseStr() {
	tests := []struct {
		testName      string
		input         []string
		expectedValue string
	}{
		{
			testName:      "Should mock from --parseStr (no mock functions)",
			input:         []string{"mock", "--parseStr", "Hello world"},
			expectedValue: "Hello world",
		},
		{
			testName:      "Should mock from --parseStr",
			input:         []string{"mock", "--parseStr", "Hello {{ Person.name }}"},
			expectedValue: "Hello ",
		},
	}
	for _, test := range tests {
		stdOut, err := suite.executeCommand(test.input...)
		assert.NoError(suite.T(), err, test.testName)
		assert.Contains(suite.T(), stdOut, test.expectedValue, test.testName)
	}
}
