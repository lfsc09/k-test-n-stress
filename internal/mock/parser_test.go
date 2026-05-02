package mock

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type MockParserTestSuite struct {
	suite.Suite
}

func TestMockParserUnitSuite(t *testing.T) {
	suite.Run(t, new(MockParserTestSuite))
}

/*
	INVALID STATES
*/

func (suite *MockParserTestSuite) TestValidateDynamicBlockSyntax_InvalidInputs() {
	tests := []struct {
		testName string
		input    string
	}{
		{
			testName: "Should give error when only opening double curly braces",
			input:    "{{",
		},
		{
			testName: "Should give error when only closing double curly braces",
			input:    "}}",
		},
		{
			testName: "Should give error when missing closing double curly braces",
			input:    "{{ Module.function:{param1}:{param2}",
		},
		{
			testName: "Should give error when missing closing double curly braces with literal value before",
			input:    "hello {{ Module.function:{param1}:{param2}",
		},
		{
			testName: "Should give error when missing opening double curly braces",
			input:    "Module.function:{param1}:{param2} }}",
		},
		{
			testName: "Should give error when missing opening double curly braces with literal value after",
			input:    "Module.function:{param1}:{param2} }} world",
		},
		{
			testName: "Should give error when nested open double curly braces",
			input:    "{{ {{ Module.function:{param1}:{param2} }}",
		},
		{
			testName: "Should give error when unexpected trailing double curly braces",
			input:    "{{ Module.function:{param1}:{param2} }} }}",
		},
		{
			testName: "Should give error when fully nested open double curly braces",
			input:    "{{ {{ Module.function:{param1}:{param2} }} }}",
		},
	}

	for _, tt := range tests {
		err := validateDynamicBlockSyntax(tt.input)
		assert.Error(suite.T(), err, "Test case '%s' failed", tt.testName)
	}
}

func (suite *MockParserTestSuite) TestParseDynamicBlock_InvalidInputs() {
	tests := []struct {
		testName string
		input    string
	}{
		{
			testName: "Should give error when dynamic block is empty",
			input:    "{{}}",
		},
		{
			testName: "Should give error when dynamic block only contains a '|' separator",
			input:    "{{|}}",
		},
		{
			testName: "Should give error when not passing a value after a '|' separator",
			input:    "{{ Module.function:{param1}:{param2} | }}",
		},
		{
			testName: "Should give error when not passing a value before a '|' separator",
			input:    "{{ | Module.function:{param1}:{param2} }}",
		},
		{
			testName: "Should give error when not passing values on either side of a '|' separator",
			input:    "{{ Module.function:{param1}:{param2} |  | Module.function:{param1}:{param2} }}",
		},
		{
			testName: "Should give error when not passing values on either side of a '|' separator",
			input:    "{{ Module.function:{param1}:{param2} || Module.function:{param1}:{param2} }}",
		},
	}

	for _, tt := range tests {
		blockCalls, err := parseDynamicBlock(tt.input)
		assert.Error(suite.T(), err, "Test case '%s' failed", tt.testName)
		assert.Nil(suite.T(), blockCalls, "Test case '%s' failed", tt.testName)
	}
}

func (suite *MockParserTestSuite) TestParseFunctionAndParams_InvalidInputs() {
	tests := []struct {
		testName string
		input    string
	}{
		{
			testName: "Should give error when function name is empty",
			input:    ":{param1}",
		},
		{
			testName: "Should give error when function param is not wrapped in curly braces",
			input:    "Module.function:param1",
		},
		{
			testName: "Should give error on malformed param example 1",
			input:    "Module.function:{",
		},
		{
			testName: "Should give error on malformed param example 2",
			input:    "Module.function:{:",
		},
		{
			testName: "Should give error on malformed param example 3",
			input:    "Module.function:}:",
		},
		{
			testName: "Should give error on malformed param example 4",
			input:    "Module.function:{{param1}",
		},
		{
			testName: "Should give error on malformed param example 5",
			input:    "Module.function:{par{am1}",
		},
		{
			testName: "Should give error on malformed param example 6",
			input:    "Module.function:{param1}}",
		},
		{
			testName: "Should give error on malformed param example 7",
			input:    "Module.function:{par}am1}",
		},
		{
			testName: "Should give error on malformed param example 8",
			input:    "Module.function:{{param1}}",
		},
		{
			testName: "Should give error on malformed param example 9",
			input:    "Module.function:{par}am1}",
		},
		{
			testName: "Should give error on malformed param example 10",
			input:    "Module.function:}param1{",
		},
		{
			testName: "Should give error on malformed param example 11",
			input:    "Module.function:param1",
		},
		{
			testName: "Should give error on malformed param with regex example 1",
			input:    "Module.function:{/[a-z]+}",
		},
		{
			testName: "Should give error on malformed param with regex example 2",
			input:    "Module.function:{[a-z]+/}",
		},
	}

	for _, tt := range tests {
		funcName, params, err := parseFunctionAndParams(tt.input)
		assert.Error(suite.T(), err, "Test case '%s' failed", tt.testName)
		assert.Empty(suite.T(), funcName, "Test case '%s' failed", tt.testName)
		assert.Nil(suite.T(), params, "Test case '%s' failed", tt.testName)
	}
}

/*
	VALID STATES
*/

func (suite *MockParserTestSuite) TestValidateDynamicBlockSyntax_ValidInputs() {
	tests := []struct {
		testName string
		input    string
	}{
		{
			testName: "Should pass for simple valid dynamic block",
			input:    "{{Module.function}}",
		},
		{
			testName: "Should pass for valid dynamic block with literal value before and after",
			input:    "hello {{Module.function}} world",
		},
		{
			testName: "Should pass for multiple valid dynamic blocks",
			input:    "{{Module.function}} {{Module.function}} {{Module.function}}",
		},
		{
			testName: "Should pass for valid dynamic block with a function that has params example 1",
			input:    "{{Module.function:{param1}:{param2}}}",
		},
		{
			testName: "Should pass for valid dynamic block with a function that has params example 2",
			input:    "{{Module.function:{p1:v1}:{p2:v2}}}",
		},
		{
			testName: "Should pass for valid dynamic block with multiple functions that has params",
			input:    "{{Module.function:{param1}:{param2}}} | {{Module.function:{param1}:{param2}}}",
		},
	}

	for _, tt := range tests {
		err := validateDynamicBlockSyntax(tt.input)
		assert.NoError(suite.T(), err, "Test case '%s' failed", tt.testName)
	}
}

func (suite *MockParserTestSuite) TestParseDynamicBlock_ValidInput() {
	tests := []struct {
		testName           string
		input              string
		expectedBlockCalls []*MockCall
	}{
		{
			testName: "Should parse a simple dynamic block with one function and no params",
			input:    "{{Module.function}}",
			expectedBlockCalls: []*MockCall{
				{
					FunctionName: "Module.function",
					Params:       []string{},
				},
			},
		},
		{
			testName: "Should parse a dynamic block with one function and multiple params",
			input:    "{{Module.function:{param1}:{param2}}}",
			expectedBlockCalls: []*MockCall{
				{
					FunctionName: "Module.function",
					Params:       []string{"param1", "param2"},
				},
			},
		},
		{
			testName: "Should parse a dynamic block with multiple functions separated by '|'",
			input:    "{{Module.function:{param1}:{param2}}} | {{Module.function:{param1}:{param2}}}",
			expectedBlockCalls: []*MockCall{
				{
					FunctionName: "Module.function",
					Params:       []string{"param1", "param2"},
				},
				{
					FunctionName: "Module.function",
					Params:       []string{"param1", "param2"},
				},
			},
		},
	}

	for _, tt := range tests {
		blockCalls, err := parseDynamicBlock(tt.input)
		assert.NoError(suite.T(), err, "Test case '%s' failed", tt.testName)
		assert.Equal(suite.T(), tt.expectedBlockCalls, blockCalls, "Test case '%s' failed", tt.testName)
	}
}

func (suite *MockParserTestSuite) TestParseFunctionAndParams_ValidInput() {
	tests := []struct {
		testName         string
		input            string
		expectedFuncName string
		expectedParams   []string
	}{
		{
			testName:         "Should parse function with no params",
			input:            "Module.function",
			expectedFuncName: "Module.function",
			expectedParams:   []string{},
		},
		{
			testName:         "Should parse function with one param",
			input:            "Module.function:{str1}",
			expectedFuncName: "Module.function",
			expectedParams:   []string{"str1"},
		},
		{
			testName:         "Should parse function with one number param",
			input:            "Module.function:{1}",
			expectedFuncName: "Module.function",
			expectedParams:   []string{"1"},
		},
		{
			testName:         "Should parse function with one signed number param",
			input:            "Module.function:{-1}",
			expectedFuncName: "Module.function",
			expectedParams:   []string{"-1"},
		},
		{
			testName:         "Should parse function with multiple params",
			input:            "Module.function:{str1}:{str2}",
			expectedFuncName: "Module.function",
			expectedParams:   []string{"str1", "str2"},
		},
		{
			testName:         "Should parse function with one empty param",
			input:            "Module.function:{}",
			expectedFuncName: "Module.function",
			expectedParams:   []string{""},
		},
		{
			testName:         "Should parse function with multiple empty params",
			input:            "Module.function:{}:{}",
			expectedFuncName: "Module.function",
			expectedParams:   []string{"", ""},
		},
		{
			testName:         "Should parse function with one empty param (no curly braces)",
			input:            "Module.function:",
			expectedFuncName: "Module.function",
			expectedParams:   []string{""},
		},
		{
			testName:         "Should parse function with multiple empty params (no curly braces)",
			input:            "Module.function::",
			expectedFuncName: "Module.function",
			expectedParams:   []string{"", ""},
		},
		{
			testName:         "Should parse function with two params where the second param is empty (no curly braces)",
			input:            "Module.function:{str1}:",
			expectedFuncName: "Module.function",
			expectedParams:   []string{"str1", ""},
		},
		{
			testName:         "Should parse function with three params where only the last param is not empty (no curly braces)",
			input:            "Module.function::{str3}",
			expectedFuncName: "Module.function",
			expectedParams:   []string{"", "", "str3"},
		},
		{
			testName:         "Should parse function with regex param",
			input:            "Module.function:{/[a-z]+/}",
			expectedFuncName: "Module.function",
			expectedParams:   []string{"/[a-z]+/"},
		},
		{
			testName:         "Should parse function with regex param",
			input:            "Module.function:{/YYYY-MM-DD/}",
			expectedFuncName: "Module.function",
			expectedParams:   []string{"/YYYY-MM-DD hh:mm/"},
		},
		{
			testName:         "Should parse function with regex param (that has restricted characters)",
			input:            "Module.function:{/[a-z]{2}\\/a\\:\\{\\{\\}\\}\\|b/}",
			expectedFuncName: "Module.function",
			expectedParams:   []string{"/[a-z]{2}\\/a\\:\\{\\{\\}\\}\\|b/"},
		},
	}

	for _, tt := range tests {
		funcName, params, err := parseFunctionAndParams(tt.input)
		assert.NoError(suite.T(), err, "Test case '%s' failed", tt.testName)
		assert.Equal(suite.T(), tt.expectedFuncName, funcName, "Test case '%s' failed", tt.testName)
		assert.Equal(suite.T(), tt.expectedParams, params, "Test case '%s' failed", tt.testName)
	}
}

func (suite *MockParserTestSuite) TestNewMockBlockSegments_ValidInput() {
	tests := []struct {
		testName               string
		input                  string
		expectedCompiledBlocks []*MockBlock
	}{
		{
			testName: "Should create a single literal block segment",
			input:    "Hello world",
			expectedCompiledBlocks: []*MockBlock{
				{
					Type:     LiteralBlock,
					RawValue: "Hello world",
					Calls:    []*MockCall{},
				},
			},
		},
		{
			testName: "Should create a single dynamic block segment",
			input:    "{{Address.city}}",
			expectedCompiledBlocks: []*MockBlock{
				{
					Type:     DynamicBlock,
					RawValue: "{{Address.city}}",
					Calls: []*MockCall{
						{
							FunctionName: "Address.city",
							Params:       []string{},
						},
					},
				},
			},
		},
		{
			testName: "Should create a mixed literal and dynamic block segment",
			input:    "Hello {{Person.name}}, are you from {{Address.city}}?",
			expectedCompiledBlocks: []*MockBlock{
				{
					Type:     LiteralBlock,
					RawValue: "Hello ",
					Calls:    []*MockCall{},
				},
				{
					Type:     LiteralBlock,
					RawValue: "Hello ",
					Calls:    []*MockCall{},
				},
				{
					Type:     DynamicBlock,
					RawValue: "{{Person.name}}",
					Calls: []*MockCall{
						{
							FunctionName: "Person.name",
							Params:       []string{},
						},
					},
				},
				{
					Type:     LiteralBlock,
					RawValue: ", are you from ",
					Calls:    []*MockCall{},
				},
				{
					Type:     DynamicBlock,
					RawValue: "{{Address.city}}",
					Calls: []*MockCall{
						{
							FunctionName: "Address.city",
							Params:       []string{},
						},
					},
				},
				{
					Type:     LiteralBlock,
					RawValue: "?",
					Calls:    []*MockCall{},
				},
			},
		},
	}

	for _, tt := range tests {
		blockSegments, err := CompileMockBlocks(tt.input)
		assert.NoError(suite.T(), err, "Test case '%s' failed", tt.testName)
		assert.Equal(suite.T(), tt.expectedCompiledBlocks, blockSegments, "Test case '%s' failed", tt.testName)
	}
}
