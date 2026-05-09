package mock

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type MockJsonTestSuite struct {
	suite.Suite
}

func TestMockJsonUnitSuite(t *testing.T) {
	suite.Run(t, new(MockJsonTestSuite))
}

/*
	VALID STATES
*/

func (suite *MockJsonTestSuite) TestCompileJSONTemplate_ValidInput() {
	tests := []struct {
		testName              string
		input                 string
		generate              uint
		expectedGenerateTotal uint
	}{
		{
			testName:              "Should compile simple JSON template with one mock function",
			input:                 `{"city":"{{Address.City}}"}`,
			generate:              1,
			expectedGenerateTotal: 1,
		},
		{
			testName:              "Should compile JSON template with multiple mock functions and nested structure",
			input:                 `{"user":{"literal":"literal value","name":"{{Person.Name}}"}}`,
			generate:              1,
			expectedGenerateTotal: 2,
		},
		{
			testName:              "Should compile JSON template with multiple mock functions, nested structure and dynamic arrays",
			input:                 `{"user[2]":{"literal":"literal value","name":"{{Person.Name}}"}}`,
			generate:              1,
			expectedGenerateTotal: 4,
		},
	}

	for _, tt := range tests {
		var rawJson map[string]any
		err := json.Unmarshal([]byte(tt.input), &rawJson)
		assert.NoError(suite.T(), err, "Test case '%s' failed", tt.testName)
		blueprintInitialNode := &TemplateJsonNode{Type: NodeObject, Repeat: tt.generate}
		templateGenerateTotal, err := CompileJSONTemplate(rawJson, blueprintInitialNode)
		assert.NoError(suite.T(), err, "Test case '%s' failed", tt.testName)
		assert.Equal(suite.T(), tt.expectedGenerateTotal, templateGenerateTotal, "Test case '%s' failed", tt.testName)
		// Assert the structure of the blueprintInitialNode
	}
}
