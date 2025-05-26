package cmd

import (
	"testing"

	"github.com/lfsc09/k-test-n-stress/mocker"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type MockCmdTestSuite struct {
	suite.Suite
}

func TestMockCmdTestSuite(t *testing.T) {
	suite.Run(t, new(MockCmdTestSuite))
}

func (suite *MockCmdTestSuite) TestExtractMockMethod_ValidInputs() {
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
			input:            "Boolean.booleanWithChance:10",
			expectedFuncName: "Boolean.booleanWithChance",
			expectedParams:   []string{"10"},
		},
		{
			testName:         "mock function with multiple params",
			input:            "Function.with:multiple:params",
			expectedFuncName: "Function.with",
			expectedParams:   []string{"multiple", "params"},
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
			input:            "Regex.regex:/[a-z0-9]{1,64}/:param2",
			expectedFuncName: "Regex.regex",
			expectedParams:   []string{"/[a-z0-9]{1,64}/", "param2"},
		},
	}

	for _, tt := range tests {
		funcName, params := extractMockMethod(tt.input)
		assert.Equal(suite.T(), tt.expectedFuncName, funcName, "Test case '%s' failed", tt.testName)
		assert.Equal(suite.T(), tt.expectedParams, params, "Test case '%s' failed", tt.testName)
	}
}

func (suite *MockCmdTestSuite) TestInterpretString_ValidInputs() {
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
		value, isMock := interpretString(tt.input)
		assert.Equal(suite.T(), tt.expectedValue, value, "Test case '%s' failed", tt.testName)
		assert.Equal(suite.T(), tt.expectedIsMock, isMock, "Test case '%s' failed", tt.testName)
	}
}

func (suite *MockCmdTestSuite) TestProcessJsonMap_ValidInputs() {
	tests := []struct {
		testName string
		input    map[string]any
	}{
		{
			testName: "string value",
			input: map[string]any{
				"key": "{{ Address.city }}",
			},
		},
		{
			testName: "string value with params",
			input: map[string]any{
				"key": "{{ Boolean.booleanWithChance:10 }}",
			},
		},
		{
			testName: "2 string values",
			input: map[string]any{
				"key":  "{{ Address.city }}",
				"key2": "{{ Address.state }}",
			},
		},
		{
			testName: "string value asking for two results",
			input: map[string]any{
				"key[2]": "{{ Address.city }}",
			},
		},
		{
			testName: "nested map",
			input: map[string]any{
				"level": map[string]any{
					"key": "{{ Address.city }}",
				},
			},
		},
		{
			testName: "nested map with 2 values",
			input: map[string]any{
				"level": map[string]any{
					"key":  "{{ Address.city }}",
					"key2": "{{ Address.state }}",
				},
			},
		},
		{
			testName: "nested map asking for two objects",
			input: map[string]any{
				"level[2]": map[string]any{
					"key": "{{ Address.city }}",
				},
			},
		},
		{
			testName: "2 nested map on same level",
			input: map[string]any{
				"level": map[string]any{
					"key": "{{ Address.city }}",
				},
				"level2": map[string]any{
					"key": "{{ Address.city }}",
				},
			},
		},
		{
			testName: "2 nested map, one inside the other",
			input: map[string]any{
				"level_0": map[string]any{
					"key": "{{ Address.city }}",
					"level_1": map[string]any{
						"key": "{{ Address.city }}",
					},
				},
			},
		},
		{
			testName: "arrays of 2 string values",
			input: map[string]any{
				"array": []any{"{{ Address.city }}", "{{ Person.firstName }}"},
			},
		},
		{
			testName: "2 arrays of 2 strings values",
			input: map[string]any{
				"array":  []any{"{{ Address.city }}", "{{ Person.firstName }}"},
				"array2": []any{"{{ Address.city }}", "{{ Person.firstName }}"},
			},
		},
		{
			testName: "nested map with array of strings inside",
			input: map[string]any{
				"level": map[string]any{
					"key":   "{{ Address.city }}",
					"array": []any{"{{ Address.city }}", "{{ Person.firstName }}"},
				},
			},
		},
		{
			testName: "array of nested maps",
			input: map[string]any{
				"array": []any{
					map[string]any{"key": "{{ Address.city }}"},
					map[string]any{"key": "{{ Person.firstName }}", "key2": "{{ Person.lastName }}"},
				},
			},
		},
		{
			testName: "array of nested maps with more maps and arrays inside",
			input: map[string]any{
				"array": []any{
					map[string]any{
						"key":   "{{ Address.city }}",
						"array": []any{"{{ Address.city }}", "{{ Person.firstName }}"},
					},
					map[string]any{
						"key":  "{{ Person.firstName }}",
						"key2": "{{ Person.lastName }}",
						"map": map[string]any{
							"key":   "{{ Address.city }}",
							"array": []any{"{{ Address.city }}", "{{ Person.firstName }}", map[string]any{"key": "{{ Person.lastName }}"}},
						},
					},
				},
			},
		},
	}

	for _, tt := range tests {
		mockerObj := mocker.New()
		err := processJsonMap(tt.input, mockerObj)
		assert.NoError(suite.T(), err, "Test case '%s' failed", tt.testName)
	}
}

func (suite *MockCmdTestSuite) TestProcessJsonMap_InvalidInputs() {
	tests := []struct {
		testName string
		input    map[string]any
	}{
		{
			testName: "invalid value in map [integer value]",
			input: map[string]any{
				"key": 123,
			},
		},
		{
			testName: "invalid type in array [integer value]",
			input: map[string]any{
				"array": []any{123},
			},
		},
	}

	for _, tt := range tests {
		mockerObj := mocker.New()
		err := processJsonMap(tt.input, mockerObj)
		assert.Error(suite.T(), err, "Test case '%s' failed", tt.testName)
	}
}

func (suite *MockCmdTestSuite) TestExtractDigitInBrackets_ValidInputs() {
	tests := []struct {
		testName      string
		inputPlace    string
		inputValue    string
		expectedDigit int
	}{
		{
			testName:      "test 1",
			inputPlace:    "object",
			inputValue:    "",
			expectedDigit: 1,
		},
		{
			testName:      "test 2",
			inputPlace:    "object",
			inputValue:    "text",
			expectedDigit: 1,
		},
		{
			testName:      "test 3",
			inputPlace:    "object",
			inputValue:    "text[10]",
			expectedDigit: 10,
		},
		{
			testName:      "test 4",
			inputPlace:    "file",
			inputValue:    "text.template.json",
			expectedDigit: 1,
		},
		{
			testName:      "test 5",
			inputPlace:    "file",
			inputValue:    "text[10].template.json",
			expectedDigit: 10,
		},
	}

	for _, tt := range tests {
		digit, err := extractDigitInBrackets(tt.inputPlace, tt.inputValue)
		assert.Equal(suite.T(), tt.expectedDigit, digit, "Test case '%s' failed", tt.testName)
		assert.NoError(suite.T(), err, "Test case '%s' failed", tt.testName)
	}
}

func (suite *MockCmdTestSuite) TestExtractDigitInBrackets_InvalidInputs() {
	tests := []struct {
		testName      string
		inputPlace    string
		inputValue    string
		expectedDigit int
	}{
		{
			testName:      "test 1",
			inputPlace:    "other",
			inputValue:    "text[0]",
			expectedDigit: 0,
		},
		{
			testName:      "test 2",
			inputPlace:    "object",
			inputValue:    "text[-2]",
			expectedDigit: 0,
		},
		{
			testName:      "test 3",
			inputPlace:    "object",
			inputValue:    "text[0]",
			expectedDigit: 0,
		},
		{
			testName:      "test 4",
			inputPlace:    "object",
			inputValue:    "[5]text",
			expectedDigit: 0,
		},
		{
			testName:      "test 5",
			inputPlace:    "object",
			inputValue:    "te[5]xt",
			expectedDigit: 0,
		},
		{
			testName:      "test 6",
			inputPlace:    "object",
			inputValue:    "text10]",
			expectedDigit: 0,
		},
		{
			testName:      "test 7",
			inputPlace:    "object",
			inputValue:    "text[]",
			expectedDigit: 0,
		},
		{
			testName:      "test 8",
			inputPlace:    "object",
			inputValue:    "text[something]",
			expectedDigit: 0,
		},
		{
			testName:      "test 9",
			inputPlace:    "object",
			inputValue:    "text[1a9]",
			expectedDigit: 0,
		},
		{
			testName:      "test 10",
			inputPlace:    "object",
			inputValue:    "text[a1]",
			expectedDigit: 0,
		},
		{
			testName:      "test 11",
			inputPlace:    "object",
			inputValue:    "text[@!]",
			expectedDigit: 0,
		},
		{
			testName:      "test 12",
			inputPlace:    "object",
			inputValue:    "text[10][5]",
			expectedDigit: 0,
		},
		{
			testName:      "test 13",
			inputPlace:    "object",
			inputValue:    "text[[5]]",
			expectedDigit: 0,
		},
		{
			testName:      "test 14",
			inputPlace:    "object",
			inputValue:    "text]1[",
			expectedDigit: 0,
		},
		{
			testName:      "test 15",
			inputPlace:    "object",
			inputValue:    "text[5 ]",
			expectedDigit: 0,
		},
		{
			testName:      "test ",
			inputPlace:    "object",
			inputValue:    "text[ 10]",
			expectedDigit: 0,
		},
		{
			testName:      "test 16",
			inputPlace:    "file",
			inputValue:    "text[5]",
			expectedDigit: 0,
		},
		{
			testName:      "test 17",
			inputPlace:    "file",
			inputValue:    "text[5].temp",
			expectedDigit: 0,
		},
		{
			testName:      "test 18",
			inputPlace:    "file",
			inputValue:    "text[5].json",
			expectedDigit: 0,
		},
		{
			testName:      "test 19",
			inputPlace:    "file",
			inputValue:    "[5].template.json",
			expectedDigit: 0,
		},
		{
			testName:      "test 20",
			inputPlace:    "file",
			inputValue:    "text[5 ].template.json",
			expectedDigit: 0,
		},
		{
			testName:      "test 21",
			inputPlace:    "file",
			inputValue:    "text [5].template.json",
			expectedDigit: 0,
		},
	}

	for _, tt := range tests {
		digit, err := extractDigitInBrackets(tt.inputPlace, tt.inputValue)
		assert.Equal(suite.T(), tt.expectedDigit, digit, "Test case '%s' failed", tt.testName)
		assert.Error(suite.T(), err, "Test case '%s' failed", tt.testName)
	}
}

func (suite *MockCmdTestSuite) TestSanitizeKeyWithBrackets_ValidInputs() {
	tests := []struct {
		testName          string
		input             string
		expectedSanitized string
	}{
		{
			testName:          "test 1",
			input:             "",
			expectedSanitized: "",
		},
		{
			testName:          "test 2",
			input:             "text",
			expectedSanitized: "text",
		},
		{
			testName:          "test 3",
			input:             "text[10]",
			expectedSanitized: "text",
		},
		{
			testName:          "test 4",
			input:             "text[10].template.json",
			expectedSanitized: "text.template.json",
		},
		{
			testName:          "test 5",
			input:             "te[5]xt",
			expectedSanitized: "text",
		},
		{
			testName:          "test 6",
			input:             "text10]",
			expectedSanitized: "text10]",
		},
		{
			testName:          "test 7",
			input:             "text]1[",
			expectedSanitized: "text]1[",
		},
	}

	for _, tt := range tests {
		sanitized := sanitizeKeyWithBrackets(tt.input)
		assert.Equal(suite.T(), tt.expectedSanitized, sanitized, "Test case '%s' failed", tt.testName)
	}
}
