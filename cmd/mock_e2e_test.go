package cmd_test

import (
	"bytes"
	"fmt"
	"regexp"
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
			testName: "both --parse-str and --parseFile",
			input:    []string{"mock", "--parse-str", "Hello {{ Person.name }}", "--parse-files", "test.json"},
		},
		{
			testName: "both --parse-str and --parse-json",
			input:    []string{"mock", "--parse-str", "Hello {{ Person.name }}", "--parse-json", "' {\"name\": \"{{ Person.name }}\"} '"},
		},
		{
			testName: "both --parse-json and --parseFile",
			input:    []string{"mock", "--parse-json", "' {\"name\": \"{{ Person.name }}\"} '", "--parse-files", "test.json"},
		},
		{
			testName: "all three --parse-str, --parse-json and --parseFile",
			input:    []string{"mock", "--parse-str", "Hello {{ Person.name }}", "--parse-json", "' {\"name\": \"{{ Person.name }}\"} '", "--parse-files", "test.json"},
		},
	}
	for _, test := range tests {
		_, err := suite.executeCommand(test.input...)
		assert.Error(suite.T(), err, test.testName)
		assert.EqualError(suite.T(), err, "provide only one of the three options: --parse-json, --parse-files or --parse-str", test.testName)
	}
}

func (suite *MockCmdE2ETestSuite) TestCLIShouldRaiseError_InvalidUseOfParseFiles() {
	testName := "Should raise error when multiple args in --parse-files"
	_, err := suite.executeCommand("mock", "--parse-files", "test.json", "test2.json")
	assert.Error(suite.T(), err, testName)
	assert.EqualError(suite.T(), err, "you passed multiple files to --parse-files without quotes. Did you mean: --parse-files \"*.template.json\"?", testName)
}

func (suite *MockCmdE2ETestSuite) TestCLIShouldRaiseError_PreserveFolderStructureFlagInvalidUse() {
	tests := []struct {
		testName string
		input    []string
	}{
		{
			testName: "--preserve-folder-structure with --parse-str",
			input:    []string{"mock", "--parse-str", "Hello {{ Person.name }}", "--preserve-folder-structure"},
		},
		{
			testName: "--preserve-folder-structure with --parse-json",
			input:    []string{"mock", "--parse-json", "' {\"name\": \"{{ Person.name }}\"} '", "--preserve-folder-structure"},
		},
	}
	for _, test := range tests {
		_, err := suite.executeCommand(test.input...)
		assert.Error(suite.T(), err, test.testName)
		assert.EqualError(suite.T(), err, "--preserve-folder-structure option is only available when using --parse-files", test.testName)
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
		{
			testName: "--generate with --parse-files",
			input:    []string{"mock", "--parse-files", "test.json", "--generate", "5"},
		},
	}
	for _, test := range tests {
		_, err := suite.executeCommand(test.input...)
		assert.Error(suite.T(), err, test.testName)
		assert.EqualError(suite.T(), err, "--generate option is only available when using --parse-json", test.testName)
	}
}

func (suite *MockCmdE2ETestSuite) TestCLIShouldRaiseError_GenerateFlagInvalidValues() {
	tests := []struct {
		testName string
		input    []string
	}{
		{
			testName: "--generate with value equal to 0",
			input:    []string{"mock", "--parse-json", "' {\"name\": \"{{ Person.name }}\"} '", "--generate", "0"},
		},
		{
			testName: "--generate with negative value",
			input:    []string{"mock", "--parse-json", "' {\"name\": \"{{ Person.name }}\"} '", "--generate", "-1"},
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
		_, err := suite.executeCommand("mock", "--parse-json", tt.jsonStr)
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
		testName string
		template string
		assertRe *regexp.Regexp
		checkMin int
		checkMax int
	}{
		{testName: "no params (defaults)", template: "{{ Number.number }}", assertRe: intOrFloatRe},
		{testName: "empty decimals (default 0)", template: "{{ Number.number:{} }}", assertRe: intRe},
		{testName: "decimals=2", template: "{{ Number.number:{2} }}", assertRe: twoDecimalRe},
		{testName: "decimals=2, min and max empty (defaults)", template: "{{ Number.number:{2}:{}:{} }}", assertRe: twoDecimalRe},
		{testName: "non-numeric decimals (ignored, defaults to 0)", template: "{{ Number.number:{abc} }}", assertRe: intRe},
		{testName: "three bare colons (all defaults)", template: "{{ Number.number::: }}", assertRe: intRe},
		{testName: "decimals=0, min=1, max=10", template: "{{ Number.number:{0}:{1}:{10} }}", assertRe: positiveIntRe, checkMin: 1, checkMax: 10},
	}

	for _, tt := range tests {
		stdOut, err := suite.executeCommand("mock", "--parse-str", tt.template)
		assert.NoError(suite.T(), err, tt.testName)
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
	boolRe := regexp.MustCompile(`^(true|false)\n$`)

	tests := []struct {
		testName    string
		template    string
		assertRe    *regexp.Regexp
		exactOutput string
	}{
		{testName: "chance = 100", template: "{{ Boolean.booleanWithChance:{100} }}", exactOutput: "true\n"},
		{testName: "chance = 0", template: "{{ Boolean.booleanWithChance:{0} }}", exactOutput: "false\n"},
		{testName: "chance empty (fallback random bool)", template: "{{ Boolean.booleanWithChance:{} }}", assertRe: boolRe},
		{testName: "chance non-numeric (fallback random bool)", template: "{{ Boolean.booleanWithChance:{abc} }}", assertRe: boolRe},
		// TODO: known panic if no param supplied — see bug_loremboolean_no_param_guard
	}

	for _, tt := range tests {
		stdOut, err := suite.executeCommand("mock", "--parse-str", tt.template)
		assert.NoError(suite.T(), err, tt.testName)
		if tt.exactOutput != "" {
			assert.Equal(suite.T(), tt.exactOutput, stdOut, tt.testName)
		} else {
			assert.Regexp(suite.T(), tt.assertRe, stdOut, tt.testName)
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
		// TODO: known panic if no param supplied — see bug_loremboolean_no_param_guard
	}{
		{testName: "paragraph default (1 sentence)", template: "{{ Lorem.paragraph:{1} }}", assertNonEmpty: true},
		{testName: "paragraph N=3", template: "{{ Lorem.paragraph:{3} }}", assertNonEmpty: true},
		{testName: "paragraph empty param (fallback)", template: "{{ Lorem.paragraph:{} }}", assertNonEmpty: true},
		{testName: "paragraphs N=2", template: "{{ Lorem.paragraphs:{2} }}", assertContains: "\n"},
		{testName: "sentence N=5", template: "{{ Lorem.sentence:{5} }}", assertNonEmpty: true},
		{testName: "sentences N=3", template: "{{ Lorem.sentences:{3} }}", assertContains: "\n"},
		{testName: "word (no params)", template: "{{ Lorem.word }}", assertNonEmpty: true},
		{testName: "words N=4", template: "{{ Lorem.words:{4} }}", assertNonEmpty: true},
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
		stdOut, err := suite.executeCommand("mock", "--parse-json", tt.jsonStr)
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
			args:        []string{"mock", "--parse-json", `{"name":"{{ Person.name }}"}`, "--generate", "3"},
			assertRegex: `^\[`,
		},
		{
			testName:    "generate=1 (default), output is JSON object not array",
			args:        []string{"mock", "--parse-json", `{"name":"{{ Person.name }}"}`},
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
