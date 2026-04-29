package mocker

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type MockerHelpersTestSuite struct {
	suite.Suite
}

func TestMockerHelpersTestSuite(t *testing.T) {
	suite.Run(t, new(MockerHelpersTestSuite))
}

/*
	INVALID STATES
*/

func (suite *MockerHelpersTestSuite) TestExtractRegex_InvalidInputs() {
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

/*
	VALID STATES
*/

func (suite *MockerHelpersTestSuite) TestCalculateChecksum_ValidInputs() {
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

func (suite *MockerHelpersTestSuite) TestExtractRegex_ValidInputs() {
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

func (suite *MockerHelpersTestSuite) TestFormatDatetime() {
	fixedTime := time.Date(2026, 4, 7, 15, 4, 5, 123000000, time.UTC)
	tests := []struct {
		testName       string
		format         string
		expectedOutput string
	}{
		{testName: "YYYY only", format: "YYYY", expectedOutput: "2026"},
		{testName: "YYYY-MM-DD", format: "YYYY-MM-DD", expectedOutput: "2026-04-07"},
		{testName: "hh:mm:ss.sss", format: "hh:mm:ss.sss", expectedOutput: "15:04:05.123"},
		{testName: "DD - mm:ss", format: "DD - mm:ss", expectedOutput: "07 - 04:05"},
		{testName: "YYYY-MM-DDThh:mm:ss.sss", format: "YYYY-MM-DDThh:mm:ss.sss", expectedOutput: "2026-04-07T15:04:05.123"},
		{testName: "sss before ss no conflict", format: "ss.sss", expectedOutput: "05.123"},
	}

	for _, tt := range tests {
		result := formatDatetime(fixedTime, tt.format)
		assert.Equal(suite.T(), tt.expectedOutput, result, "Test case '%s' failed", tt.testName)
	}
}

func (suite *MockerHelpersTestSuite) TestParseDateOnly() {
	tests := []struct {
		testName    string
		input       string
		expectError bool
	}{
		{testName: "valid date", input: "2026-04-07", expectError: false},
		{testName: "invalid string", input: "not-a-date", expectError: true},
		{testName: "empty string", input: "", expectError: true},
		{testName: "wrong format (datetime)", input: "2026-04-07T15:04", expectError: true},
	}

	for _, tt := range tests {
		t, err := parseDateOnly(tt.input)
		if tt.expectError {
			assert.Error(suite.T(), err, "Test case '%s' failed", tt.testName)
		} else {
			assert.NoError(suite.T(), err, "Test case '%s' failed", tt.testName)
			assert.Equal(suite.T(), 2026, t.Year(), "Test case '%s' failed", tt.testName)
			assert.Equal(suite.T(), 4, int(t.Month()), "Test case '%s' failed", tt.testName)
			assert.Equal(suite.T(), 7, t.Day(), "Test case '%s' failed", tt.testName)
		}
	}
}

func (suite *MockerHelpersTestSuite) TestParseTimeOnly() {
	tests := []struct {
		testName    string
		input       string
		expectError bool
	}{
		{testName: "valid time", input: "15:04", expectError: false},
		{testName: "invalid hour", input: "25:00", expectError: true},
		{testName: "invalid string", input: "abc", expectError: true},
		{testName: "empty string", input: "", expectError: true},
	}

	for _, tt := range tests {
		t, err := parseTimeOnly(tt.input)
		if tt.expectError {
			assert.Error(suite.T(), err, "Test case '%s' failed", tt.testName)
		} else {
			assert.NoError(suite.T(), err, "Test case '%s' failed", tt.testName)
			assert.Equal(suite.T(), 15, t.Hour(), "Test case '%s' failed", tt.testName)
			assert.Equal(suite.T(), 4, t.Minute(), "Test case '%s' failed", tt.testName)
		}
	}
}

func (suite *MockerHelpersTestSuite) TestParseDatetimeFull() {
	tests := []struct {
		testName    string
		input       string
		expectError bool
	}{
		{testName: "valid datetime", input: "2026-04-07T15:04", expectError: false},
		{testName: "valid date-only fallback", input: "2026-04-07", expectError: false},
		{testName: "invalid string", input: "not-valid", expectError: true},
		{testName: "empty string", input: "", expectError: true},
	}

	for _, tt := range tests {
		t, err := parseDatetimeFull(tt.input)
		if tt.expectError {
			assert.Error(suite.T(), err, "Test case '%s' failed", tt.testName)
		} else {
			assert.NoError(suite.T(), err, "Test case '%s' failed", tt.testName)
			switch tt.testName {
			case "valid datetime":
				assert.Equal(suite.T(), 15, t.Hour(), "Test case '%s' failed", tt.testName)
				assert.Equal(suite.T(), 4, t.Minute(), "Test case '%s' failed", tt.testName)
			case "valid date-only fallback":
				assert.Equal(suite.T(), 0, t.Hour(), "Test case '%s' failed", tt.testName)
				assert.Equal(suite.T(), 0, t.Minute(), "Test case '%s' failed", tt.testName)
			}
		}
	}
}
