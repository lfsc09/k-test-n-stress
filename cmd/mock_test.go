package cmd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/lfsc09/k-test-n-stress/mocker"
	"github.com/mohae/deepcopy"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type MockCmdTestSuite struct {
	suite.Suite
}

func TestMockCmdUnitSuite(t *testing.T) {
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
		funcName, params, err := extractMockMethod(tt.input)
		assert.NoError(suite.T(), err, "Test case '%s' failed", tt.testName)
		assert.Equal(suite.T(), tt.expectedFuncName, funcName, "Test case '%s' failed", tt.testName)
		assert.Equal(suite.T(), tt.expectedParams, params, "Test case '%s' failed", tt.testName)
	}
}

func (suite *MockCmdTestSuite) TestExtractMockMethod_BareValueRejection() {
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
		funcName, params, err := extractMockMethod(tt.input)
		assert.Error(suite.T(), err, "Test case '%s' failed", tt.testName)
		assert.Contains(suite.T(), err.Error(), "must be wrapped in", "Test case '%s' failed", tt.testName)
		assert.Empty(suite.T(), funcName, "Test case '%s' failed", tt.testName)
		assert.Nil(suite.T(), params, "Test case '%s' failed", tt.testName)
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
				"key": "{{ Boolean.booleanWithChance:{10} }}",
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
	}

	for _, tt := range tests {
		digit, err := extractDigitInBrackets(tt.inputPlace, tt.inputValue)
		assert.Equal(suite.T(), tt.expectedDigit, digit, "Test case '%s' failed", tt.testName)
		assert.Error(suite.T(), err, "Test case '%s' failed", tt.testName)
	}
}

func (suite *MockCmdTestSuite) TestAnalyzeTemplate() {
	tests := []struct {
		testName      string
		parseMap      map[string]any
		generate      int
		numWorkers    int
		wantSeq       bool
		wantCount     int
		wantDepth     int
		wantMinWeight int64 // TotalWeight must be >= this value when wantSeq==false
	}{
		{
			testName:      "flat template generate=1 is sequential",
			parseMap:      map[string]any{"name": "{{ Person.name }}"},
			generate:      1,
			numWorkers:    8,
			wantSeq:       true,
			wantCount:     1,
			wantDepth:     0,
			wantMinWeight: 1,
		},
		{
			testName:      "flat template generate=50000 is concurrent",
			parseMap:      map[string]any{"name": "{{ Person.name }}"},
			generate:      50000,
			numWorkers:    8,
			wantSeq:       false,
			wantCount:     50000,
			wantDepth:     0,
			wantMinWeight: 50000,
		},
		{
			// With generate==1 and count >= numWorkers, analyzeTemplate now returns an
			// inner split (Depth=1) rather than falling back to root.
			testName: "inner bracket key n=100000 with generate=1 uses inner split",
			parseMap: map[string]any{
				"employees[100000]": map[string]any{"name": "{{ Person.name }}"},
			},
			generate:      1,
			numWorkers:    8,
			wantSeq:       false,
			wantCount:     100000, // inner split count = splitNode.count
			wantDepth:     1,
			wantMinWeight: 100000,
		},
		{
			// generate > 1 with inner [n] keys falls back to root split.
			testName: "inner bracket key n=100000 with generate=2 falls back to root split",
			parseMap: map[string]any{
				"employees[100000]": map[string]any{"name": "{{ Person.name }}"},
			},
			generate:      2,
			numWorkers:    8,
			wantSeq:       false,
			wantCount:     2, // root split count = generate
			wantDepth:     0,
			wantMinWeight: 100000, // innerWeight includes the [n] key
		},
		{
			// generate==1: outer[20000] at depth 1 is the split node; inner[5000] at depth 2
			// contributes to innerWeight.
			testName: "nested n keys at different depths — inner split at shallowest with generate=1",
			parseMap: map[string]any{
				"outer[20000]": map[string]any{
					"inner[5000]": map[string]any{"name": "{{ Person.name }}"},
				},
			},
			generate:      1,
			numWorkers:    8,
			wantSeq:       false,
			wantCount:     20000, // inner split count = splitNode.count
			wantDepth:     1,
			wantMinWeight: 20000 * 5000, // Count * InnerWeight
		},
		{
			// generate > 1 with nested [n] keys falls back to root split.
			testName: "nested n keys at different depths — falls back to root with generate=2",
			parseMap: map[string]any{
				"outer[20000]": map[string]any{
					"inner[5000]": map[string]any{"name": "{{ Person.name }}"},
				},
			},
			generate:      2,
			numWorkers:    8,
			wantSeq:       false,
			wantCount:     2,
			wantDepth:     0,
			wantMinWeight: 20000 * 5000, // product of both
		},
		{
			// generate==1 with two siblings at depth 1: first qualifying sibling is the
			// split node; the other contributes to sibling sequential processing.
			testName: "sibling n keys at same depth with generate=1 uses inner split on first",
			parseMap: map[string]any{
				"truck[10000]": map[string]any{"model": "{{ Car.model }}"},
				"car[10000]":   map[string]any{"model": "{{ Car.model }}"},
			},
			generate:      1,
			numWorkers:    8,
			wantSeq:       false,
			wantCount:     10000, // inner split count = splitNode.count
			wantDepth:     1,
			wantMinWeight: 10000, // only nodes below splitDepth count in innerWeight
		},
		{
			testName: "all n nodes below numWorkers threshold — sequential via totalWeight",
			parseMap: map[string]any{
				"company[3]": map[string]any{"name": "{{ Company.name }}"},
			},
			generate:      1,
			numWorkers:    8,
			wantSeq:       true,
			wantCount:     1,
			wantDepth:     0,
			wantMinWeight: 1,
		},
		{
			testName:      "totalWeight just below threshold (9999) is sequential",
			parseMap:      map[string]any{"name": "{{ Person.name }}"},
			generate:      9999,
			numWorkers:    8,
			wantSeq:       true,
			wantCount:     9999,
			wantDepth:     0,
			wantMinWeight: 9999,
		},
		{
			testName:      "totalWeight at threshold (10000) is concurrent",
			parseMap:      map[string]any{"name": "{{ Person.name }}"},
			generate:      10000,
			numWorkers:    8,
			wantSeq:       false,
			wantCount:     10000,
			wantDepth:     0,
			wantMinWeight: 10000,
		},
	}

	for _, tt := range tests {
		sp := analyzeTemplate(tt.parseMap, tt.generate, tt.numWorkers)
		assert.Equal(suite.T(), tt.wantSeq, sp.UseSequential, "Test case '%s': UseSequential", tt.testName)
		assert.Equal(suite.T(), tt.wantCount, sp.Count, "Test case '%s': Count", tt.testName)
		assert.Equal(suite.T(), tt.wantDepth, sp.Depth, "Test case '%s': Depth", tt.testName)
		if !tt.wantSeq {
			assert.GreaterOrEqual(suite.T(), sp.TotalWeight, tt.wantMinWeight, "Test case '%s': TotalWeight", tt.testName)
		}
	}
}

func (suite *MockCmdTestSuite) TestAnalyzeTemplate_InnerSplit() {
	tests := []struct {
		testName        string
		parseMap        map[string]any
		generate        int
		numWorkers      int
		wantDepth       int
		wantCount       int
		wantInnerWeight int64
		wantTotalWeight int64
		wantSeq         bool
	}{
		{
			// employees[20] at depth 1, count >= numWorkers=4, no nested [n] keys.
			testName: "one inner [n] key at depth 1 above numWorkers",
			parseMap: map[string]any{
				"employees[20]": map[string]any{"name": "{{ Person.name }}"},
			},
			generate:        1,
			numWorkers:      4,
			wantDepth:       1,
			wantCount:       20,
			wantInnerWeight: 1,
			wantTotalWeight: 20,   // 1 * 20 * 1
			wantSeq:         true, // 20 < 10000
		},
		{
			// employees[20] at depth 1 with nested phones[5] at depth 2.
			testName: "inner [n] at depth 1 with nested [n] at depth 2",
			parseMap: map[string]any{
				"employees[20]": map[string]any{
					"phones[5]": "{{ Person.phoneNumber }}",
				},
			},
			generate:        1,
			numWorkers:      4,
			wantDepth:       1,
			wantCount:       20,
			wantInnerWeight: 5,
			wantTotalWeight: 100,  // 1 * 20 * 5
			wantSeq:         true, // 100 < 10000
		},
		{
			// company[2] at depth 1, count < numWorkers=8 → splitDepth==-1 → root split.
			testName: "root [n] key count below numWorkers falls back to root split",
			parseMap: map[string]any{
				"company[2]": map[string]any{"name": "{{ Company.name }}"},
			},
			generate:        1,
			numWorkers:      8,
			wantDepth:       0,
			wantCount:       1,
			wantInnerWeight: 2,
			wantTotalWeight: 2, // 1 * 1 * 2 (root split innerWeight product = 2)
			wantSeq:         true,
		},
		{
			// No [n] keys at all → root split, generate=1.
			testName: "no [n] keys uses root split",
			parseMap: map[string]any{
				"name": "{{ Person.name }}",
			},
			generate:        1,
			numWorkers:      8,
			wantDepth:       0,
			wantCount:       1,
			wantInnerWeight: 1,
			wantTotalWeight: 1,
			wantSeq:         true,
		},
		{
			// large inner array: employees[20000] → totalWeight = 20000 >= 10000.
			testName: "inner split is concurrent when totalWeight at threshold",
			parseMap: map[string]any{
				"employees[20000]": map[string]any{"name": "{{ Person.name }}"},
			},
			generate:        1,
			numWorkers:      4,
			wantDepth:       1,
			wantCount:       20000,
			wantInnerWeight: 1,
			wantTotalWeight: 20000,
			wantSeq:         false,
		},
	}

	for _, tt := range tests {
		sp := analyzeTemplate(tt.parseMap, tt.generate, tt.numWorkers)
		assert.Equal(suite.T(), tt.wantDepth, sp.Depth, "Test case '%s': Depth", tt.testName)
		assert.Equal(suite.T(), tt.wantCount, sp.Count, "Test case '%s': Count", tt.testName)
		assert.Equal(suite.T(), tt.wantInnerWeight, sp.InnerWeight, "Test case '%s': InnerWeight", tt.testName)
		assert.Equal(suite.T(), tt.wantTotalWeight, sp.TotalWeight, "Test case '%s': TotalWeight", tt.testName)
		assert.Equal(suite.T(), tt.wantSeq, sp.UseSequential, "Test case '%s': UseSequential", tt.testName)
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

func (suite *MockCmdTestSuite) TestResolveOutputPaths() {
	tests := []struct {
		testName         string
		toJsonFileSet    bool
		toJsonFileValue  string
		toCsvFileSet     bool
		toCsvFileValue   string
		source           string
		defaultFilePath  string
		defaultCsvPath   string
		wantJsonNotEmpty bool
		wantCsvNotEmpty  bool
		wantError        bool
	}{
		{
			testName:         "neither flag set returns empty paths",
			toJsonFileSet:    false,
			toCsvFileSet:     false,
			source:           "parse-json",
			wantJsonNotEmpty: false,
			wantCsvNotEmpty:  false,
		},
		{
			testName:         "explicit JSON path",
			toJsonFileSet:    true,
			toJsonFileValue:  "mydata.json",
			toCsvFileSet:     false,
			source:           "parse-json",
			wantJsonNotEmpty: true,
			wantCsvNotEmpty:  false,
		},
		{
			testName:         "explicit CSV path",
			toJsonFileSet:    false,
			toCsvFileSet:     true,
			toCsvFileValue:   "mydata.csv",
			source:           "parse-json",
			wantJsonNotEmpty: false,
			wantCsvNotEmpty:  true,
		},
		{
			testName:         "parse-json-file source uses defaults",
			toJsonFileSet:    true,
			toJsonFileValue:  "_use_default_",
			toCsvFileSet:     true,
			toCsvFileValue:   "_use_default_",
			source:           "parse-json-file",
			defaultFilePath:  "/tmp/output.json",
			defaultCsvPath:   "/tmp/output.csv",
			wantJsonNotEmpty: true,
			wantCsvNotEmpty:  true,
		},
	}

	for _, tt := range tests {
		jsonPath, csvPath, err := resolveOutputPaths(
			tt.toJsonFileSet, tt.toJsonFileValue,
			tt.toCsvFileSet, tt.toCsvFileValue,
			tt.source, tt.defaultFilePath, tt.defaultCsvPath,
		)
		if tt.wantError {
			assert.Error(suite.T(), err, "Test case '%s'", tt.testName)
		} else {
			assert.NoError(suite.T(), err, "Test case '%s'", tt.testName)
		}
		if tt.wantJsonNotEmpty {
			assert.NotEmpty(suite.T(), jsonPath, "Test case '%s': jsonPath", tt.testName)
		} else {
			assert.Empty(suite.T(), jsonPath, "Test case '%s': jsonPath", tt.testName)
		}
		if tt.wantCsvNotEmpty {
			assert.NotEmpty(suite.T(), csvPath, "Test case '%s': csvPath", tt.testName)
		} else {
			assert.Empty(suite.T(), csvPath, "Test case '%s': csvPath", tt.testName)
		}
	}
}

func (suite *MockCmdTestSuite) TestExtractCsvHeaders() {
	tests := []struct {
		testName string
		template map[string]any
		want     []string
	}{
		{
			testName: "flat template returns sorted keys",
			template: map[string]any{
				"name":  "{{ Person.name }}",
				"email": "{{ Internet.email }}",
				"age":   "{{ Number.number }}",
			},
			want: []string{"age", "email", "name"},
		},
		{
			testName: "key with bracket notation is sanitized",
			template: map[string]any{
				"phones[3]": "{{ Person.phoneNumber }}",
				"name":      "{{ Person.name }}",
			},
			want: []string{"name", "phones"},
		},
		{
			testName: "single key",
			template: map[string]any{"z": "val"},
			want:     []string{"z"},
		},
	}

	for _, tt := range tests {
		headers := extractCsvHeaders(tt.template)
		assert.Equal(suite.T(), tt.want, headers, "Test case '%s'", tt.testName)
	}
}

func (suite *MockCmdTestSuite) TestSanitizeJsonMap_ArrayRecursion() {
	tests := []struct {
		testName string
		input    map[string]any
		wantKeys []string // top-level keys expected after sanitization
	}{
		{
			testName: "array of maps with bracketed keys inside",
			input: map[string]any{
				"outer[2]": []any{
					map[string]any{"inner[3]": "value1"},
					map[string]any{"inner[3]": "value2"},
				},
			},
			wantKeys: []string{"outer"},
		},
		{
			testName: "flat array of strings is unaffected",
			input: map[string]any{
				"phones[3]": []any{"111", "222", "333"},
			},
			wantKeys: []string{"phones"},
		},
	}

	for _, tt := range tests {
		sanitizeJsonMap(tt.input)
		for _, wantKey := range tt.wantKeys {
			_, ok := tt.input[wantKey]
			assert.True(suite.T(), ok, "Test case '%s': expected key '%s' after sanitization", tt.testName, wantKey)
		}
		// Verify array elements also have sanitized keys
		if tt.testName == "array of maps with bracketed keys inside" {
			arr := tt.input["outer"].([]any)
			for i, elem := range arr {
				elemMap := elem.(map[string]any)
				_, hasUnsanitized := elemMap["inner[3]"]
				assert.False(suite.T(), hasUnsanitized, "Test case '%s': element %d still has unsanitized key", tt.testName, i)
				_, hasSanitized := elemMap["inner"]
				assert.True(suite.T(), hasSanitized, "Test case '%s': element %d missing sanitized key", tt.testName, i)
			}
		}
	}
}

func (suite *MockCmdTestSuite) TestAtomicFileCreate_CommitSucceeds() {
	dir := suite.T().TempDir()
	finalPath := filepath.Join(dir, "output.json")

	f, commit, _, err := atomicFileCreate(finalPath)
	assert.NoError(suite.T(), err)
	assert.NotNil(suite.T(), f)

	_, err = f.WriteString(`{"ok":true}`)
	assert.NoError(suite.T(), err)

	err = commit()
	assert.NoError(suite.T(), err)

	// Final file must exist with expected content
	content, err := os.ReadFile(finalPath)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), `{"ok":true}`, string(content))
}

func (suite *MockCmdTestSuite) TestAtomicFileCreate_AbortCleansUp() {
	dir := suite.T().TempDir()
	finalPath := filepath.Join(dir, "output.json")

	f, _, abort, err := atomicFileCreate(finalPath)
	assert.NoError(suite.T(), err)

	tmpName := f.Name()
	_, err = f.WriteString("partial")
	assert.NoError(suite.T(), err)

	abort()

	// Temp file must be gone
	_, err = os.Stat(tmpName)
	assert.True(suite.T(), os.IsNotExist(err), "temp file should be removed after abort")

	// Final file must NOT exist
	_, err = os.Stat(finalPath)
	assert.True(suite.T(), os.IsNotExist(err), "final file must not exist after abort")
}

// BenchmarkDeepCopyTemplate measures the reflection cost of deepcopy.Copy for a
// representative nested template. Run with:
//
//	go test -bench=BenchmarkDeepCopyTemplate -benchmem ./cmd/
//
// If ns/op is high (>5000 per iteration), consider a hand-rolled clone.
// TODO(deepcopy-hand-rolled): replace deepcopy.Copy with a type-aware clone for
// map[string]any templates — see concurrency-brainstorm.md for the trade-offs.
func BenchmarkDeepCopyTemplate(b *testing.B) {
	template := map[string]any{
		"field1":  "{{ Person.name }}",
		"field2":  "{{ Internet.email }}",
		"field3":  "{{ UUID.uuidv4 }}",
		"field4":  "{{ Address.city }}",
		"field5":  "{{ Company.name }}",
		"field6":  "{{ Number.number::{0}:{1}:{100} }}",
		"field7":  "{{ Boolean.boolean }}",
		"field8":  "{{ Date.date }}",
		"field9":  "{{ Car.model }}",
		"field10": "{{ Lorem.word }}",
		"nested": map[string]any{
			"sub1": "{{ Person.firstName }}",
			"sub2": "{{ Person.lastName }}",
			"sub3": "{{ Internet.url }}",
			"sub4": "{{ UUID.uuidv4 }}",
			"sub5": "{{ Address.postCode }}",
		},
	}

	b.ResetTimer()
	for range b.N {
		_ = deepcopy.Copy(template).(map[string]any)
	}
}
