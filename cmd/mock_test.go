package cmd

import (
	"encoding/json"
	"testing"

	"github.com/lfsc09/k-test-n-stress/mocker"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type MockCmdTestSuite struct {
	suite.Suite
}

func TestMockCmdUnitSuite(t *testing.T) {
	suite.Run(t, new(MockCmdTestSuite))
}

/*
	INVALID STATES
*/

func (suite *MockCmdTestSuite) TestParseMockFunction_InvalidInputs() {
	tests := []struct {
		testName string
		input    string
	}{
		{
			testName: "bare single param",
			input:    "Number.number:2",
		},
		{
			testName: "bare first of two params",
			input:    "Number.number:2:1",
		},
		{
			testName: "bare second param, first is wrapped",
			input:    "Number.number:{2}:1",
		},
		{
			testName: "bare param after valid value",
			input:    "Date.date::{2030-01-01}:YYYY-MM-DD",
		},
		{
			testName: "bare param with spaces",
			input:    "Number.number: 2 ",
		},
	}

	for _, tt := range tests {
		funcName, params, err := parseMockFunction(tt.input)
		assert.Error(suite.T(), err, "Test case '%s' failed", tt.testName)
		assert.Contains(suite.T(), err.Error(), "must be wrapped in", "Test case '%s' failed", tt.testName)
		assert.Empty(suite.T(), funcName, "Test case '%s' failed", tt.testName)
		assert.Nil(suite.T(), params, "Test case '%s' failed", tt.testName)
	}
}

/*
	VALID STATES
*/

func (suite *MockCmdTestSuite) TestParseMockFunction_ValidInputs() {
	tests := []struct {
		testName         string
		input            string
		expectedFuncName string
		expectedParams   []string
	}{
		{
			testName:         "empty string",
			input:            "",
			expectedFuncName: "",
			expectedParams:   nil,
		},
		{
			testName:         "simple mock function",
			input:            "Address.city",
			expectedFuncName: "Address.city",
			expectedParams:   []string{},
		},
		{
			testName:         "mock function with params",
			input:            "Boolean.booleanWithChance:{10}",
			expectedFuncName: "Boolean.booleanWithChance",
			expectedParams:   []string{"10"},
		},
		{
			testName:         "mock function with multiple params",
			input:            "Function.with:{multiple}:{params}",
			expectedFuncName: "Function.with",
			expectedParams:   []string{"multiple", "params"},
		},
		{
			testName:         "value param with colon inside",
			input:            "Date.time:{18:00}:{20:00}",
			expectedFuncName: "Date.time",
			expectedParams:   []string{"18:00", "20:00"},
		},
		{
			testName:         "mixed value and regex params",
			input:            "Date.time:{18:00}:{20:00}:/hh:mm/",
			expectedFuncName: "Date.time",
			expectedParams:   []string{"18:00", "20:00", "/hh:mm/"},
		},
		{
			testName:         "empty value param",
			input:            "Number.number:{}:{5}:{10}",
			expectedFuncName: "Number.number",
			expectedParams:   []string{"", "5", "10"},
		},
		{
			testName:         "regex mock function with empty regex",
			input:            "Regex.regex://",
			expectedFuncName: "Regex.regex",
			expectedParams:   []string{"//"},
		},
		{
			testName:         "regular regex mock function",
			input:            "Regex.regex:/[a-z0-9]{1,64}/",
			expectedFuncName: "Regex.regex",
			expectedParams:   []string{"/[a-z0-9]{1,64}/"},
		},
		{
			testName:         "regex mock function with params",
			input:            "Regex.regex:/[a-z0-9]{1,64}/:{param2}",
			expectedFuncName: "Regex.regex",
			expectedParams:   []string{"/[a-z0-9]{1,64}/", "param2"},
		},
		{
			testName:         "no params at all",
			input:            "Address.city",
			expectedFuncName: "Address.city",
			expectedParams:   []string{},
		},
		{
			testName:         "single empty wrapped param",
			input:            "Number.number:{}",
			expectedFuncName: "Number.number",
			expectedParams:   []string{""},
		},
		{
			testName:         "three params all empty wrapped",
			input:            "Number.number:{}:{}:{}",
			expectedFuncName: "Number.number",
			expectedParams:   []string{"", "", ""},
		},
		{
			testName:         "three params with bare colons (empty bare)",
			input:            "Number.number:::",
			expectedFuncName: "Number.number",
			expectedParams:   []string{"", "", ""},
		},
		{
			testName:         "single regex param",
			input:            "Regex.regex:/[a-z]+/",
			expectedFuncName: "Regex.regex",
			expectedParams:   []string{"/[a-z]+/"},
		},
		{
			testName:         "colon inside value param",
			input:            "Date.time:{18:00}:{20:00}",
			expectedFuncName: "Date.time",
			expectedParams:   []string{"18:00", "20:00"},
		},
		{
			testName:         "colon inside regex param",
			input:            "Date.time:::/hh:mm/",
			expectedFuncName: "Date.time",
			expectedParams:   []string{"", "", "/hh:mm/"},
		},
	}

	for _, tt := range tests {
		funcName, params, err := parseMockFunction(tt.input)
		assert.NoError(suite.T(), err, "Test case '%s' failed", tt.testName)
		assert.Equal(suite.T(), tt.expectedFuncName, funcName, "Test case '%s' failed", tt.testName)
		assert.Equal(suite.T(), tt.expectedParams, params, "Test case '%s' failed", tt.testName)
	}
}

func (suite *MockCmdTestSuite) TestParseDynamicValue_ValidInputs() {
	tests := []struct {
		testName       string
		input          string
		expectedValue  string
		expectedIsMock bool
	}{
		{
			testName:       "empty string",
			input:          "",
			expectedValue:  "",
			expectedIsMock: false,
		},
		{
			testName:       "no whitespace",
			input:          "{{Address.city}}",
			expectedValue:  "Address.city",
			expectedIsMock: true,
		},
		{
			testName:       "regular whitespaces",
			input:          "{{ Address.city }}",
			expectedValue:  "Address.city",
			expectedIsMock: true,
		},
		{
			testName:       "multiple whitespaces at begining",
			input:          "{{    Address.city }}",
			expectedValue:  "Address.city",
			expectedIsMock: true,
		},
		{
			testName:       "multiple whitespaces at end",
			input:          "{{ Address.city    }}",
			expectedValue:  "Address.city",
			expectedIsMock: true,
		},
		{
			testName:       "multiple whitespaces at begining and end",
			input:          "{{     Address.city    }}",
			expectedValue:  "Address.city",
			expectedIsMock: true,
		},
		{
			testName:       "whitespaces before brackets",
			input:          "  {{     Address.city    }}",
			expectedValue:  "Address.city",
			expectedIsMock: true,
		},
		{
			testName:       "content but no brackets",
			input:          "Address.city",
			expectedValue:  "Address.city",
			expectedIsMock: false,
		},
		{
			testName:       "brackets in middle of content",
			input:          "{{ Address.cit}}y",
			expectedValue:  "{{ Address.cit}}y",
			expectedIsMock: false,
		},
	}

	for _, tt := range tests {
		value, isMock := parseDynamicValue(tt.input)
		assert.Equal(suite.T(), tt.expectedValue, value, "Test case '%s' failed", tt.testName)
		assert.Equal(suite.T(), tt.expectedIsMock, isMock, "Test case '%s' failed", tt.testName)
	}
}

func (suite *MockCmdTestSuite) TestResolveJSONToFilePath_ValidInputs() {
	tests := []struct {
		testName          string
		parseJsonFileSet  bool
		parseJsonFile     string
		toJsonFile        string
		wantJsonToContain string
	}{
		{
			testName:          "--parse-json with --to-file ''",
			parseJsonFileSet:  false,
			toJsonFile:        "",
			wantJsonToContain: "output.json",
		},
		{
			testName:          "--parse-json with --to-file 'mydata.json'",
			parseJsonFileSet:  false,
			toJsonFile:        "mydata.json",
			wantJsonToContain: "mydata.json",
		},
		{
			testName:          "--parse-json-file 'example.template.json' with --to-file ''",
			parseJsonFileSet:  true,
			parseJsonFile:     "example.template.json",
			toJsonFile:        "",
			wantJsonToContain: "example.json",
		},
		{
			testName:          "--parse-json-file 'example.template.json' with --to-file 'mydata.json'",
			parseJsonFileSet:  true,
			parseJsonFile:     "example.template.json",
			toJsonFile:        "mydata.json",
			wantJsonToContain: "mydata.json",
		},
	}

	for _, tt := range tests {
		jsonPath, err := resolveJSONToFilePath(
			tt.parseJsonFileSet,
			tt.parseJsonFile,
			tt.toJsonFile,
		)
		assert.NoError(suite.T(), err, "Test case '%s' failed", tt.testName)
		assert.Contains(suite.T(), jsonPath, tt.wantJsonToContain, "Test case '%s' failed", tt.testName)
	}
}

func (suite *MockCmdTestSuite) TestProcessMFStr_ValidInputs() {
	tests := []struct {
		testName    string
		input       string
		assertRegex string
	}{
		{
			testName:    "Should process simple mock function",
			input:       "{{Address.city}}",
			assertRegex: `^.+$`,
		},
		{
			testName:    "Should process simple mock function mixing literal values and dynamic values",
			input:       "Hello {{Address.city}}, welcome to {{Company.name}}!",
			assertRegex: `^Hello .+, welcome to .+!$`,
		},
	}

	mockerInstance := mocker.New()

	for _, tt := range tests {
		output, err := processMFStr(tt.input, mockerInstance)
		assert.NoError(suite.T(), err, "Test case '%s' failed", tt.testName)
		assert.Regexp(suite.T(), tt.assertRegex, output, "Test case '%s' failed", tt.testName)
	}
}

func (suite *MockCmdTestSuite) TestCompileJSONTemplate_ValidInput() {
	tests := []struct {
		testName              string
		input                 string
		generate              uint
		expectedGenerateTotal uint
	}{
		{
			testName:              "Should compile simple JSON template with one mock function",
			input:                 `{"city":"{{Address.city}}"}`,
			generate:              1,
			expectedGenerateTotal: 1,
		},
		{
			testName:              "Should compile JSON template with multiple mock functions and nested structure",
			input:                 `{"user":{"literal":"literal value","name":"{{Person.name}}"}}`,
			generate:              1,
			expectedGenerateTotal: 2,
		},
		{
			testName:              "Should compile JSON template with multiple mock functions, nested structure and dynamic arrays",
			input:                 `{"user[2]":{"literal":"literal value","name":"{{Person.name}}"}}`,
			generate:              1,
			expectedGenerateTotal: 4,
		},
	}

	for _, tt := range tests {
		var rawJson map[string]any
		err := json.Unmarshal([]byte(tt.input), &rawJson)
		assert.NoError(suite.T(), err, "Test case '%s' failed", tt.testName)
		blueprintInitialNode := &TemplateNode{Type: NodeObject, Repeat: tt.generate}
		templateGenerateTotal, err := compileJSONTemplate(rawJson, blueprintInitialNode)
		assert.NoError(suite.T(), err, "Test case '%s' failed", tt.testName)
		assert.Equal(suite.T(), tt.expectedGenerateTotal, templateGenerateTotal, "Test case '%s' failed", tt.testName)
		// Assert the structure of the blueprintInitialNode
	}
}
