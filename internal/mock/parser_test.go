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
			input:    "{{ Module.Fn:{param1}:{param2}",
		},
		{
			testName: "Should give error when missing closing double curly braces with literal value before",
			input:    "hello {{ Module.Fn:{param1}:{param2}",
		},
		{
			testName: "Should give error when missing opening double curly braces",
			input:    "Module.Fn:{param1}:{param2} }}",
		},
		{
			testName: "Should give error when missing opening double curly braces with literal value after",
			input:    "Module.Fn:{param1}:{param2} }} world",
		},
		{
			testName: "Should give error when nested open double curly braces",
			input:    "{{ {{ Module.Fn:{param1}:{param2} }}",
		},
		{
			testName: "Should give error when unexpected trailing double curly braces",
			input:    "{{ Module.Fn:{param1}:{param2} }} }}",
		},
		{
			testName: "Should give error when fully nested open double curly braces",
			input:    "{{ {{ Module.Fn:{param1}:{param2} }} }}",
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
			input:    "{{ Module.Fn:{param1}:{param2} | }}",
		},
		{
			testName: "Should give error when not passing a value before a '|' separator",
			input:    "{{ | Module.Fn:{param1}:{param2} }}",
		},
		{
			testName: "Should give error when not passing values on either side of a '|' separator",
			input:    "{{ Module.Fn:{param1}:{param2} |  | Module.Fn:{param1}:{param2} }}",
		},
		{
			testName: "Should give error when not passing values on either side of a '|' separator",
			input:    "{{ Module.Fn:{param1}:{param2} || Module.Fn:{param1}:{param2} }}",
		},
		{
			testName: "Should give error when using '|' outside of a dynamic block",
			input:    "{{ Module.Fn:{param1}:{param2} }} | {{ Module.Fn:{param1}:{param2} }}",
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
			input:    "Module.Fn:param1",
		},
		{
			testName: "Should give error on malformed param example 1",
			input:    "Module.Fn:{",
		},
		{
			testName: "Should give error on malformed param example 2",
			input:    "Module.Fn:{:",
		},
		{
			testName: "Should give error on malformed param example 3",
			input:    "Module.Fn:}:",
		},
		{
			testName: "Should give error on malformed param example 4",
			input:    "Module.Fn:{{param1}",
		},
		{
			testName: "Should give error on malformed param example 5",
			input:    "Module.Fn:{par{am1}",
		},
		{
			testName: "Should give error on malformed param example 6",
			input:    "Module.Fn:{param1}}",
		},
		{
			testName: "Should give error on malformed param example 7",
			input:    "Module.Fn:{par}am1}",
		},
		{
			testName: "Should give error on malformed param example 8",
			input:    "Module.Fn:{{param1}}",
		},
		{
			testName: "Should give error on malformed param example 9",
			input:    "Module.Fn:{par}am1}",
		},
		{
			testName: "Should give error on malformed param example 10",
			input:    "Module.Fn:}param1{",
		},
		{
			testName: "Should give error on malformed param example 11",
			input:    "Module.Fn:param1",
		},
		{
			testName: "Should give error on malformed param with regex example 1",
			input:    "Module.Fn:{/[a-z]+}",
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
			input:    "{{Module.Fn}}",
		},
		{
			testName: "Should pass for valid dynamic block with literal value before and after",
			input:    "hello {{Module.Fn}} world",
		},
		{
			testName: "Should pass for multiple valid dynamic blocks",
			input:    "{{Module.Fn}} {{Module.Fn}} {{Module.Fn}}",
		},
		{
			testName: "Should pass for valid dynamic block with a function that has params example 1",
			input:    "{{Module.Fn:{param1}:{param2}}}",
		},
		{
			testName: "Should pass for valid dynamic block with a function that has params example 2",
			input:    "{{Module.Fn:{p1:v1}:{p2:v2}}}",
		},
		{
			testName: "Should pass for valid dynamic block with multiple functions that has params",
			input:    "{{Module.Fn:{param1}:{param2} | Module.Fn:{param1}:{param2}}}",
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
			input:    "{{Module.Fn}}",
			expectedBlockCalls: []*MockCall{
				{
					FunctionName: "Module.Fn",
					Params:       []string{},
				},
			},
		},
		{
			testName: "Should parse a dynamic block with one function and multiple params",
			input:    "{{Module.Fn:{param1}:{param2}}}",
			expectedBlockCalls: []*MockCall{
				{
					FunctionName: "Module.Fn",
					Params:       []string{"param1", "param2"},
				},
			},
		},
		{
			testName: "Should parse a dynamic block with multiple functions separated by '|'",
			input:    "{{Module.Fn:{param1}:{param2} | Module.Fn:{param1}:{param2}}}",
			expectedBlockCalls: []*MockCall{
				{
					FunctionName: "Module.Fn",
					Params:       []string{"param1", "param2"},
				},
				{
					FunctionName: "Module.Fn",
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
			input:            "Module.Fn",
			expectedFuncName: "Module.Fn",
			expectedParams:   []string{},
		},
		{
			testName:         "Should parse function with one param",
			input:            "Module.Fn:{str1}",
			expectedFuncName: "Module.Fn",
			expectedParams:   []string{"str1"},
		},
		{
			testName:         "Should parse function with one number param",
			input:            "Module.Fn:{1}",
			expectedFuncName: "Module.Fn",
			expectedParams:   []string{"1"},
		},
		{
			testName:         "Should parse function with one signed number param",
			input:            "Module.Fn:{-1}",
			expectedFuncName: "Module.Fn",
			expectedParams:   []string{"-1"},
		},
		{
			testName:         "Should parse function with multiple params",
			input:            "Module.Fn:{str1}:{str2}",
			expectedFuncName: "Module.Fn",
			expectedParams:   []string{"str1", "str2"},
		},
		{
			testName:         "Should parse function with one empty param",
			input:            "Module.Fn:{}",
			expectedFuncName: "Module.Fn",
			expectedParams:   []string{""},
		},
		{
			testName:         "Should parse function with multiple empty params",
			input:            "Module.Fn:{}:{}",
			expectedFuncName: "Module.Fn",
			expectedParams:   []string{"", ""},
		},
		{
			testName:         "Should parse function with one empty param (no curly braces)",
			input:            "Module.Fn:",
			expectedFuncName: "Module.Fn",
			expectedParams:   []string{""},
		},
		{
			testName:         "Should parse function with multiple empty params (no curly braces)",
			input:            "Module.Fn::",
			expectedFuncName: "Module.Fn",
			expectedParams:   []string{"", ""},
		},
		{
			testName:         "Should parse function with two params where the second param is empty (no curly braces)",
			input:            "Module.Fn:{str1}:",
			expectedFuncName: "Module.Fn",
			expectedParams:   []string{"str1", ""},
		},
		{
			testName:         "Should parse function with three params where only the last param is not empty (no curly braces)",
			input:            "Module.Fn:::{str3}",
			expectedFuncName: "Module.Fn",
			expectedParams:   []string{"", "", "str3"},
		},
		{
			testName:         "Should parse function with regex param",
			input:            "Module.Fn:{/[a-z]+/}",
			expectedFuncName: "Module.Fn",
			expectedParams:   []string{"/[a-z]+/"},
		},
		{
			testName:         "Should parse function with regex param (that has restricted characters)",
			input:            "Module.Fn:{/[a-z]{2}\\/a\\:\\{\\{\\}\\}\\|b/}",
			expectedFuncName: "Module.Fn",
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
					Calls:    nil,
				},
			},
		},
		{
			testName: "Should create a single dynamic block segment",
			input:    "{{Module.Fn}}",
			expectedCompiledBlocks: []*MockBlock{
				{
					Type:     DynamicBlock,
					RawValue: "{{Module.Fn}}",
					Calls: []*MockCall{
						{
							FunctionName: "Module.Fn",
							Params:       []string{},
						},
					},
				},
			},
		},
		{
			testName: "Should create a mixed literal and dynamic block segment",
			input:    "Hello {{Module.Fn1}}, are you from {{Module.Fn2}}?",
			expectedCompiledBlocks: []*MockBlock{
				{
					Type:     LiteralBlock,
					RawValue: "Hello ",
					Calls:    nil,
				},
				{
					Type:     DynamicBlock,
					RawValue: "{{Module.Fn1}}",
					Calls: []*MockCall{
						{
							FunctionName: "Module.Fn1",
							Params:       []string{},
						},
					},
				},
				{
					Type:     LiteralBlock,
					RawValue: ", are you from ",
					Calls:    nil,
				},
				{
					Type:     DynamicBlock,
					RawValue: "{{Module.Fn2}}",
					Calls: []*MockCall{
						{
							FunctionName: "Module.Fn2",
							Params:       []string{},
						},
					},
				},
				{
					Type:     LiteralBlock,
					RawValue: "?",
					Calls:    nil,
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
