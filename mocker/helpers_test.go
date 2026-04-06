package mocker

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type MockerUtilsInternalTestSuite struct {
	suite.Suite
}

func TestMockerUtilsInternalTestSuite(t *testing.T) {
	suite.Run(t, new(MockerUtilsInternalTestSuite))
}

func (suite *MockerUtilsInternalTestSuite) TestCalculateChecksum_ValidInputs() {
	tests := []struct {
		testName       string
		digits         []int
		multipliers    []int
		expectedOutput int
	}{
		{
			testName:       "remainder > 2",
			digits:         []int{1, 2, 3, 4, 5},
			multipliers:    []int{5, 4, 3, 2, 1},
			expectedOutput: 9, // (1*5 + 2*4 + 3*3 + 4*2 + 5*1 = 35, 35%11 = 2, 11-2 = 9)
		},
		{
			testName:       "remainder < 2",
			digits:         []int{1, 1, 1},
			multipliers:    []int{1, 1, 1},
			expectedOutput: 8, // (1*1 + 1*1 + 1*1 = 3, 3%11 = 3, 11-3 = 8)
		},
		{
			testName:       "remainder = 0",
			digits:         []int{0, 0, 0},
			multipliers:    []int{5, 4, 3},
			expectedOutput: 0, // (0*5 + 0*4 + 0*3 = 0, 0%11 = 0, 0 < 2, return 0)
		},
		{
			testName:       "sum divisible by 11",
			digits:         []int{1, 2, 3, 5},
			multipliers:    []int{2, 3, 4, 2},
			expectedOutput: 3, // (1*2 + 2*3 + 3*4 + 5*2 = 2 + 6 + 12 + 10 = 30, 30%11 = 8, 11-8 = 3)
		},
	}

	for _, tt := range tests {
		result := calculateChecksum(tt.digits, tt.multipliers)
		assert.Equal(suite.T(), tt.expectedOutput, result, "Test case '%s' failed", tt.testName)
	}
}

func (suite *MockerUtilsInternalTestSuite) TestExtractRegex_ValidInputs() {
	tests := []struct {
		testName       string
		input          string
		expectedOutput string
	}{
		{
			testName:       "empty regex",
			input:          "//",
			expectedOutput: "",
		},
		{
			testName:       "simple regex",
			input:          "/abc/",
			expectedOutput: "abc",
		},
		{
			testName:       "with escaping slashes",
			input:          "/a\\/b\\//",
			expectedOutput: "a/b/",
		},
		{
			testName:       "more complex regex",
			input:          "/[a-z0-9]{1,64}/",
			expectedOutput: "[a-z0-9]{1,64}",
		},
	}

	for _, tt := range tests {
		output, err := extractRegex(tt.input)
		assert.Equal(suite.T(), tt.expectedOutput, output, "Test case '%s' failed", tt.testName)
		assert.NoError(suite.T(), err, "Test case '%s' failed", tt.testName)
	}
}

func (suite *MockerUtilsInternalTestSuite) TestExtractRegex_InvalidInputs() {
	tests := []struct {
		testName       string
		input          string
		expectedOutput string
	}{
		{
			testName:       "not between slashes",
			input:          "abc",
			expectedOutput: "",
		},
		{
			testName:       "no trailing slash",
			input:          "/abc",
			expectedOutput: "",
		},
		{
			testName:       "no begining slash",
			input:          "abc/",
			expectedOutput: "",
		},
		{
			testName:       "not between slashes 2",
			input:          "abc/def",
			expectedOutput: "",
		},
	}

	for _, tt := range tests {
		output, err := extractRegex(tt.input)
		assert.Empty(suite.T(), output, "Test case '%s' failed", tt.testName)
		assert.Error(suite.T(), err, "Test case '%s' failed", tt.testName)
	}
}
