package mock

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type MockAtomicFileTestSuite struct {
	suite.Suite
}

func TestMockAtomicFileUnitSuite(t *testing.T) {
	suite.Run(t, new(MockAtomicFileTestSuite))
}

/*
	VALID STATES
*/

func (suite *MockAtomicFileTestSuite) TestResolveJSONToFilePath_ValidInputs() {
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
		jsonPath, err := ResolveJSONToFilePath(
			tt.parseJsonFileSet,
			tt.parseJsonFile,
			tt.toJsonFile,
		)
		assert.NoError(suite.T(), err, "Test case '%s' failed", tt.testName)
		assert.Contains(suite.T(), jsonPath, tt.wantJsonToContain, "Test case '%s' failed", tt.testName)
	}
}
