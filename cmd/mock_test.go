package cmd

import (
	"testing"

	"github.com/lfsc09/k-test-n-stress/mock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type MockCmdTestSuite struct {
	suite.Suite
}

func TestMockCmdTestSuite(t *testing.T) {
	suite.Run(t, new(MockCmdTestSuite))
}

/*
TEST "interpretString" function
*/
func (suite *MockCmdTestSuite) TestInterpretString_ValidInputs() {
	tests := []struct {
		input            string
		expectedFuncName string
		expectedParams   []string
	}{
		{"", "", nil},
		{"Address.city", "Address.city", []string{}},
		{"Boolean.booleanWithChance:10", "Boolean.booleanWithChance", []string{"10"}},
		{"Function.with:multiple:params", "Function.with", []string{"multiple", "params"}},
		{"Regex.regex://", "Regex.regex", []string{"//"}},
		{"Regex.regex:/[a-z0-9]{1,64}/", "Regex.regex", []string{"/[a-z0-9]{1,64}/"}},
		{"Regex.regex:/[a-z0-9]{1,64}/:param2", "Regex.regex", []string{"/[a-z0-9]{1,64}/", "param2"}},
	}

	for _, tt := range tests {
		funcName, params := interpretString(tt.input)
		assert.Equal(suite.T(), tt.expectedFuncName, funcName)
		assert.Equal(suite.T(), tt.expectedParams, params)
	}
}

func (suite *MockCmdTestSuite) TestProcessMap_ValidInputs() {
	tests := []struct {
		testName string
		input    map[string]interface{}
	}{
		{
			testName: "string value",
			input: map[string]interface{}{
				"key": "Address.city",
			},
		},
		{
			testName: "string value with params",
			input: map[string]interface{}{
				"key": "Boolean.booleanWithChance:10",
			},
		},
		{
			testName: "2 string values",
			input: map[string]interface{}{
				"key":  "Address.city",
				"key2": "Address.state",
			},
		},
		{
			testName: "nested map",
			input: map[string]interface{}{
				"level": map[string]interface{}{
					"key": "Address.city",
				},
			},
		},
		{
			testName: "nested map with 2 values",
			input: map[string]interface{}{
				"level": map[string]interface{}{
					"key":  "Address.city",
					"key2": "Address.state",
				},
			},
		},
		{
			testName: "2 nested map on same level",
			input: map[string]interface{}{
				"level": map[string]interface{}{
					"key": "Address.city",
				},
				"level2": map[string]interface{}{
					"key": "Address.city",
				},
			},
		},
		{
			testName: "2 nested map, one inside the other",
			input: map[string]interface{}{
				"level_0": map[string]interface{}{
					"key": "Address.city",
					"level_1": map[string]interface{}{
						"key": "Address.city",
					},
				},
			},
		},
		{
			testName: "arrays of 2 string values",
			input: map[string]interface{}{
				"array": []interface{}{"Address.city", "Person.firstName"},
			},
		},
		{
			testName: "2 arrays of 2 strings values",
			input: map[string]interface{}{
				"array":  []interface{}{"Address.city", "Person.firstName"},
				"array2": []interface{}{"Address.city", "Person.firstName"},
			},
		},
		{
			testName: "nested map with array of strings inside",
			input: map[string]interface{}{
				"level": map[string]interface{}{
					"key":   "Address.city",
					"array": []interface{}{"Address.city", "Person.firstName"},
				},
			},
		},
		{
			testName: "array of nested maps",
			input: map[string]interface{}{
				"array": []interface{}{
					map[string]interface{}{"key": "Address.city"},
					map[string]interface{}{"key": "Person.firstName", "key2": "Person.lastName"},
				},
			},
		},
		{
			testName: "array of nested maps with more maps and arrays inside",
			input: map[string]interface{}{
				"array": []interface{}{
					map[string]interface{}{
						"key":   "Address.city",
						"array": []interface{}{"Address.city", "Person.firstName"},
					},
					map[string]interface{}{
						"key":  "Person.firstName",
						"key2": "Person.lastName",
						"map": map[string]interface{}{
							"key":   "Address.city",
							"array": []interface{}{"Address.city", "Person.firstName", map[string]interface{}{"key": "Person.lastName"}},
						},
					},
				},
			},
		},
	}

	for _, tt := range tests {
		mockObj := mock.New()
		err := processMap(tt.input, mockObj)
		assert.NoError(suite.T(), err)
	}
}

func (suite *MockCmdTestSuite) TestProcessMap_InvalidInputs() {
	tests := []struct {
		testName string
		input    map[string]interface{}
	}{
		{
			testName: "invalid value in map [integer value]",
			input: map[string]interface{}{
				"key": 123,
			},
		},
		{
			testName: "invalid type in array [integer value]",
			input: map[string]interface{}{
				"array": []interface{}{123},
			},
		},
	}

	for _, tt := range tests {
		mockObj := mock.New()
		err := processMap(tt.input, mockObj)
		assert.Error(suite.T(), err)
	}
}
