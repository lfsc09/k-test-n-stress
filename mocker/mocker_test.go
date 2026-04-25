package mocker

import (
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type MockerTestSuite struct {
	suite.Suite
	m *Mock
}

func TestMockerTestSuite(t *testing.T) {
	suite.Run(t, new(MockerTestSuite))
}

func (suite *MockerTestSuite) SetupTest() {
	suite.m = New()
}

/*
	CONCURRENCY TESTS
*/

// TestMockerConcurrency_NoConcurrentStateSharing verifies that multiple goroutines
// can call mocker.New() independently and use their own instance without data races.
// Run with: go test -race ./mocker/
func (suite *MockerTestSuite) TestMockerConcurrency_NoConcurrentStateSharing() {
	numGoroutines := runtime.NumCPU() * 2
	errCh := make(chan error, numGoroutines)

	var wg sync.WaitGroup
	for range numGoroutines {
		wg.Go(func() {
			m := New()
			for range 500 {
				if _, err := m.Generate("Person.cpf", nil); err != nil {
					errCh <- err
					return
				}
				if _, err := m.Generate("Payment.creditCardCvv", nil); err != nil {
					errCh <- err
					return
				}
			}
		})
	}

	wg.Wait()
	close(errCh)

	for err := range errCh {
		assert.NoError(suite.T(), err)
	}
}

/*
	INVALID STATES
*/

func (suite *MockerTestSuite) TestDateDate_InvalidInputs() {
	tests := []struct {
		testName      string
		params        []string
		expectedError string
	}{
		{
			testName:      "from after to",
			params:        []string{"2030-01-01", "2020-01-01"},
			expectedError: "Date.date: 'from' must be before 'to'",
		},
		{
			testName:      "format not wrapped in slashes",
			params:        []string{"2024-01-01", "2024-12-31", "YYYY-MM-DD"},
			expectedError: "must be wrapped in /.../",
		},
	}

	for _, tt := range tests {
		result, err := suite.m.Generate("Date.date", tt.params)
		assert.Error(suite.T(), err, "Test case '%s' should return an error", tt.testName)
		assert.Empty(suite.T(), result, "Test case '%s' should return empty result", tt.testName)
		assert.Contains(suite.T(), err.Error(), tt.expectedError, "Test case '%s' error message mismatch", tt.testName)
	}
}

func (suite *MockerTestSuite) TestDateTime_InvalidInputs() {
	tests := []struct {
		testName      string
		params        []string
		expectedError string
	}{
		{
			testName:      "from after to",
			params:        []string{"23:00", "10:00"},
			expectedError: "Date.time: 'from' must be before 'to'",
		},
		{
			testName:      "format not wrapped in slashes",
			params:        []string{"08:00", "18:00", "hh:mm"},
			expectedError: "must be wrapped in /.../",
		},
	}

	for _, tt := range tests {
		result, err := suite.m.Generate("Date.time", tt.params)
		assert.Error(suite.T(), err, "Test case '%s' should return an error", tt.testName)
		assert.Empty(suite.T(), result, "Test case '%s' should return empty result", tt.testName)
		assert.Contains(suite.T(), err.Error(), tt.expectedError, "Test case '%s' error message mismatch", tt.testName)
	}
}

func (suite *MockerTestSuite) TestDateDatetime_InvalidInputs() {
	tests := []struct {
		testName      string
		params        []string
		expectedError string
	}{
		{
			testName:      "from after to",
			params:        []string{"2030-01-01T00:00", "2020-01-01T00:00"},
			expectedError: "Date.datetime: 'from' must be before 'to'",
		},
		{
			testName:      "format not wrapped in slashes",
			params:        []string{"2024-01-01", "2024-12-31", "YYYY-MM-DD"},
			expectedError: "must be wrapped in /.../",
		},
	}

	for _, tt := range tests {
		result, err := suite.m.Generate("Date.datetime", tt.params)
		assert.Error(suite.T(), err, "Test case '%s' should return an error", tt.testName)
		assert.Empty(suite.T(), result, "Test case '%s' should return empty result", tt.testName)
		assert.Contains(suite.T(), err.Error(), tt.expectedError, "Test case '%s' error message mismatch", tt.testName)
	}
}

func (suite *MockerTestSuite) TestDateNow_InvalidInputs() {
	tests := []struct {
		testName      string
		params        []string
		expectedError string
	}{
		{
			testName:      "format not wrapped in slashes",
			params:        []string{"YYYY-MM-DD"},
			expectedError: "must be wrapped in /.../",
		},
	}

	for _, tt := range tests {
		result, err := suite.m.Generate("Date.now", tt.params)
		assert.Error(suite.T(), err, "Test case '%s' should return an error", tt.testName)
		assert.Empty(suite.T(), result, "Test case '%s' should return empty result", tt.testName)
		assert.Contains(suite.T(), err.Error(), tt.expectedError, "Test case '%s' error message mismatch", tt.testName)
	}
}

func (suite *MockerTestSuite) TestNumberNumber_InvalidInputs() {
	tests := []struct {
		testName      string
		params        []string
		expectedError string
	}{
		{
			testName:      "non-integer decimals",
			params:        []string{"abc"},
			expectedError: "Number.number: 'decimals' must be an integer",
		},
		{
			testName:      "non-numeric min",
			params:        []string{"0", "abc"},
			expectedError: "Number.number: 'min' must be a number",
		},
		{
			testName:      "non-numeric max",
			params:        []string{"0", "0", "abc"},
			expectedError: "Number.number: 'max' must be a number",
		},
		{
			testName:      "max less than min",
			params:        []string{"0", "100", "50"},
			expectedError: "Number.number: 'max'",
		},
	}

	for _, tt := range tests {
		result, err := suite.m.Generate("Number.number", tt.params)
		assert.Error(suite.T(), err, "Test case '%s' should return an error", tt.testName)
		assert.Empty(suite.T(), result, "Test case '%s' should return empty result", tt.testName)
		assert.Contains(suite.T(), err.Error(), tt.expectedError, "Test case '%s' error message mismatch", tt.testName)
	}
}

func (suite *MockerTestSuite) TestRegexRegex_InvalidInputs() {
	tests := []struct {
		testName      string
		params        []string
		expectedError string
	}{
		{
			testName:      "no params",
			params:        nil,
			expectedError: "regex function requires a regex pattern as parameter",
		},
		{
			testName:      "pattern not wrapped in slashes",
			params:        []string{"[0-9]{3}"},
			expectedError: "must be wrapped in /.../",
		},
		{
			testName:      "invalid regex pattern - unclosed character class",
			params:        []string{"/[invalid(/"},
			expectedError: "failed to generate regex",
		},
	}

	for _, tt := range tests {
		result, err := suite.m.Generate("Regex.regex", tt.params)
		assert.Error(suite.T(), err, "Test case '%s' should return an error", tt.testName)
		assert.Empty(suite.T(), result, "Test case '%s' should return empty result", tt.testName)
		assert.Contains(suite.T(), err.Error(), tt.expectedError, "Test case '%s' error message mismatch", tt.testName)
	}
}

/*
	VALID STATES
*/

func (suite *MockerTestSuite) TestDateDate_ValidInputs() {
	dateRegex := regexp.MustCompile(`^\d{4}-\d{2}-\d{2}$`)

	tests := []struct {
		testName string
		params   []string
		validate func(result string)
	}{
		{
			testName: "no params - defaults YYYY-MM-DD within ±5 years",
			params:   nil,
			validate: func(result string) {
				assert.Regexp(suite.T(), dateRegex, result)
				t, err := time.Parse("2006-01-02", result)
				assert.NoError(suite.T(), err)
				now := time.Now()
				assert.True(suite.T(), !t.Before(now.AddDate(-5, 0, -1)))
				assert.True(suite.T(), !t.After(now.AddDate(5, 0, 1)))
			},
		},
		{
			testName: "explicit from and to - result within range",
			params:   []string{"2024-01-01", "2024-12-31"},
			validate: func(result string) {
				t, err := time.Parse("2006-01-02", result)
				assert.NoError(suite.T(), err)
				from, _ := time.Parse("2006-01-02", "2024-01-01")
				to, _ := time.Parse("2006-01-02", "2024-12-31")
				assert.True(suite.T(), !t.Before(from) && !t.After(to))
			},
		},
		{
			testName: "only from param - to defaults to +5 years",
			params:   []string{"2025-06-01"},
			validate: func(result string) {
				t, err := time.Parse("2006-01-02", result)
				assert.NoError(suite.T(), err)
				from, _ := time.Parse("2006-01-02", "2025-06-01")
				assert.True(suite.T(), !t.Before(from))
			},
		},
		{
			testName: "empty from, explicit to - from defaults to -5 years",
			params:   []string{"", "2025-12-31"},
			validate: func(result string) {
				t, err := time.Parse("2006-01-02", result)
				assert.NoError(suite.T(), err)
				to, _ := time.Parse("2006-01-02", "2025-12-31")
				assert.True(suite.T(), !t.After(to))
			},
		},
		{
			testName: "same from and to - returns exact date",
			params:   []string{"2026-03-15", "2026-03-15"},
			validate: func(result string) {
				assert.Equal(suite.T(), "2026-03-15", result)
			},
		},
		{
			testName: "custom format /DD/MM/YYYY/",
			params:   []string{"2024-01-01", "2024-12-31", "/DD/MM/YYYY/"},
			validate: func(result string) {
				assert.Regexp(suite.T(), regexp.MustCompile(`^\d{2}/\d{2}/\d{4}$`), result)
			},
		},
		{
			testName: "invalid from falls back to default -5 years",
			params:   []string{"not-a-date", "2030-01-01"},
			validate: func(result string) {
				assert.Regexp(suite.T(), dateRegex, result)
				t, err := time.Parse("2006-01-02", result)
				assert.NoError(suite.T(), err)
				to, _ := time.Parse("2006-01-02", "2030-01-01")
				assert.True(suite.T(), !t.After(to))
			},
		},
		{
			testName: "invalid to falls back to default +5 years",
			params:   []string{"2020-01-01", "not-a-date"},
			validate: func(result string) {
				assert.Regexp(suite.T(), dateRegex, result)
				t, err := time.Parse("2006-01-02", result)
				assert.NoError(suite.T(), err)
				from, _ := time.Parse("2006-01-02", "2020-01-01")
				assert.True(suite.T(), !t.Before(from))
			},
		},
	}

	for _, tt := range tests {
		result, err := suite.m.Generate("Date.date", tt.params)
		assert.NoError(suite.T(), err, "Test case '%s' should not return error", tt.testName)
		assert.NotEmpty(suite.T(), result, "Test case '%s' should not return empty result", tt.testName)
		tt.validate(result)
	}
}

func (suite *MockerTestSuite) TestDateTime_ValidInputs() {
	// Date.time always builds the result with time.Date(..., time.UTC), so
	// all hour/minute comparisons are timezone-safe.
	timeRegex := regexp.MustCompile(`^\d{2}:\d{2}:\d{2}\.\d{3}$`)

	tests := []struct {
		testName string
		params   []string
		validate func(result string)
	}{
		{
			testName: "no params - defaults hh:mm:ss.sss within 00:00-23:59",
			params:   nil,
			validate: func(result string) {
				assert.Regexp(suite.T(), timeRegex, result)
				t, err := time.Parse("15:04:05.000", result)
				assert.NoError(suite.T(), err)
				assert.True(suite.T(), t.Hour() >= 0 && t.Hour() <= 23)
			},
		},
		{
			testName: "explicit from and to - result within range",
			params:   []string{"08:00", "18:00"},
			validate: func(result string) {
				assert.Regexp(suite.T(), timeRegex, result)
				t, err := time.Parse("15:04:05.000", result)
				assert.NoError(suite.T(), err)
				resultMins := t.Hour()*60 + t.Minute()
				assert.True(suite.T(), resultMins >= 8*60 && resultMins <= 18*60)
			},
		},
		{
			testName: "only from param - to defaults to 23:59",
			params:   []string{"16:00"},
			validate: func(result string) {
				assert.Regexp(suite.T(), timeRegex, result)
				t, err := time.Parse("15:04:05.000", result)
				assert.NoError(suite.T(), err)
				assert.True(suite.T(), t.Hour() >= 16)
			},
		},
		{
			testName: "empty from, explicit to - from defaults to 00:00",
			params:   []string{"", "06:00"},
			validate: func(result string) {
				assert.Regexp(suite.T(), timeRegex, result)
				t, err := time.Parse("15:04:05.000", result)
				assert.NoError(suite.T(), err)
				resultMins := t.Hour()*60 + t.Minute()
				assert.True(suite.T(), resultMins <= 6*60)
			},
		},
		{
			testName: "same from and to - result starts with expected hh:mm",
			params:   []string{"14:00", "14:00"},
			validate: func(result string) {
				assert.True(suite.T(), strings.HasPrefix(result, "14:00:"))
			},
		},
		{
			testName: "custom format /hh-mm/",
			params:   []string{"08:00", "18:00", "/hh-mm/"},
			validate: func(result string) {
				assert.Regexp(suite.T(), regexp.MustCompile(`^\d{2}-\d{2}$`), result)
			},
		},
		{
			testName: "invalid from falls back to default 00:00",
			params:   []string{"not-a-time", "12:00"},
			validate: func(result string) {
				assert.Regexp(suite.T(), timeRegex, result)
				t, err := time.Parse("15:04:05.000", result)
				assert.NoError(suite.T(), err)
				resultMins := t.Hour()*60 + t.Minute()
				assert.True(suite.T(), resultMins <= 12*60)
			},
		},
		{
			testName: "invalid to falls back to default 23:59",
			params:   []string{"12:00", "not-a-time"},
			validate: func(result string) {
				assert.Regexp(suite.T(), timeRegex, result)
				t, err := time.Parse("15:04:05.000", result)
				assert.NoError(suite.T(), err)
				assert.True(suite.T(), t.Hour() >= 12)
			},
		},
	}

	for _, tt := range tests {
		result, err := suite.m.Generate("Date.time", tt.params)
		assert.NoError(suite.T(), err, "Test case '%s' should not return error", tt.testName)
		assert.NotEmpty(suite.T(), result, "Test case '%s' should not return empty result", tt.testName)
		tt.validate(result)
	}
}

func (suite *MockerTestSuite) TestDateDatetime_ValidInputs() {
	datetimeRegex := regexp.MustCompile(`^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}\.\d{3}$`)

	tests := []struct {
		testName string
		params   []string
		validate func(result string)
	}{
		{
			testName: "no params - defaults YYYY-MM-DDThh:mm:ss.sss within ±5 years",
			params:   nil,
			validate: func(result string) {
				assert.Regexp(suite.T(), datetimeRegex, result)
				year, err := strconv.Atoi(result[:4])
				assert.NoError(suite.T(), err)
				now := time.Now()
				assert.True(suite.T(), year >= now.Year()-5 && year <= now.Year()+5)
			},
		},
		{
			testName: "explicit from and to using datetime format - valid format returned",
			params:   []string{"2023-03-01T00:00", "2023-09-30T23:59"},
			validate: func(result string) {
				assert.Regexp(suite.T(), datetimeRegex, result)
			},
		},
		{
			testName: "explicit from and to using date-only format - valid format returned",
			params:   []string{"2022-01-01", "2022-12-31"},
			validate: func(result string) {
				assert.Regexp(suite.T(), datetimeRegex, result)
			},
		},
		{
			testName: "only from param - valid format returned",
			params:   []string{"2025-06-01T00:00"},
			validate: func(result string) {
				assert.Regexp(suite.T(), datetimeRegex, result)
			},
		},
		{
			testName: "empty from, explicit to - valid format returned",
			params:   []string{"", "2025-12-31T23:59"},
			validate: func(result string) {
				assert.Regexp(suite.T(), datetimeRegex, result)
			},
		},
		{
			testName: "same from and to - valid format returned",
			params:   []string{"2024-06-15T12:00", "2024-06-15T12:00"},
			validate: func(result string) {
				assert.Regexp(suite.T(), datetimeRegex, result)
			},
		},
		{
			testName: "custom format /YYYY-MM-DD/",
			params:   []string{"2024-01-01", "2024-12-31", "/YYYY-MM-DD/"},
			validate: func(result string) {
				assert.Regexp(suite.T(), regexp.MustCompile(`^\d{4}-\d{2}-\d{2}$`), result)
			},
		},
		{
			testName: "invalid from falls back to default -5 years",
			params:   []string{"not-a-datetime", "2030-01-01"},
			validate: func(result string) {
				assert.Regexp(suite.T(), datetimeRegex, result)
			},
		},
		{
			testName: "invalid to falls back to default +5 years",
			params:   []string{"2020-01-01", "not-a-datetime"},
			validate: func(result string) {
				assert.Regexp(suite.T(), datetimeRegex, result)
			},
		},
	}

	for _, tt := range tests {
		result, err := suite.m.Generate("Date.datetime", tt.params)
		assert.NoError(suite.T(), err, "Test case '%s' should not return error", tt.testName)
		assert.NotEmpty(suite.T(), result, "Test case '%s' should not return empty result", tt.testName)
		tt.validate(result)
	}
}

func (suite *MockerTestSuite) TestDateNow_ValidInputs() {
	tests := []struct {
		testName string
		params   []string
		validate func(result string)
	}{
		{
			testName: "no params - default format YYYY-MM-DDThh:mm:ss.sss",
			params:   nil,
			validate: func(result string) {
				assert.Regexp(suite.T(), regexp.MustCompile(`^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}\.\d{3}$`), result)
				year, err := strconv.Atoi(result[:4])
				assert.NoError(suite.T(), err)
				assert.Equal(suite.T(), time.Now().Year(), year)
			},
		},
		{
			testName: "custom format /YYYY-MM-DD/ - matches today's date",
			params:   []string{"/YYYY-MM-DD/"},
			validate: func(result string) {
				assert.Regexp(suite.T(), regexp.MustCompile(`^\d{4}-\d{2}-\d{2}$`), result)
				assert.Equal(suite.T(), time.Now().Format("2006-01-02"), result)
			},
		},
		{
			testName: "empty format param - falls back to default format",
			params:   []string{""},
			validate: func(result string) {
				assert.Regexp(suite.T(), regexp.MustCompile(`^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}\.\d{3}$`), result)
			},
		},
	}

	for _, tt := range tests {
		result, err := suite.m.Generate("Date.now", tt.params)
		assert.NoError(suite.T(), err, "Test case '%s' should not return error", tt.testName)
		assert.NotEmpty(suite.T(), result, "Test case '%s' should not return empty result", tt.testName)
		tt.validate(result)
	}
}

func (suite *MockerTestSuite) TestCompanyCnpj_ValidInputs() {
	cnpjRegex := regexp.MustCompile(`^\d{2}\.\d{3}\.\d{3}/\d{4}-\d{2}$`)

	tests := []struct {
		testName string
		params   []string
		validate func(result string)
	}{
		{
			testName: "no params - format matches XX.XXX.XXX/XXXX-XX",
			params:   nil,
			validate: func(result string) {
				assert.Regexp(suite.T(), cnpjRegex, result)
			},
		},
		{
			testName: "no params - checksum digits are valid",
			params:   nil,
			validate: func(result string) {
				digits := strings.Map(func(r rune) rune {
					if r >= '0' && r <= '9' {
						return r
					}
					return -1
				}, result)
				assert.Len(suite.T(), digits, 14)
				d := make([]int, 14)
				for i, c := range digits {
					d[i] = int(c - '0')
				}
				assert.Equal(suite.T(), calculateChecksum(d[:12], []int{5, 4, 3, 2, 9, 8, 7, 6, 5, 4, 3, 2}), d[12])
				assert.Equal(suite.T(), calculateChecksum(d[:13], []int{6, 5, 4, 3, 2, 9, 8, 7, 6, 5, 4, 3, 2}), d[13])
			},
		},
	}

	for _, tt := range tests {
		result, err := suite.m.Generate("Company.cnpj", tt.params)
		assert.NoError(suite.T(), err, "Test case '%s' should not return error", tt.testName)
		assert.NotEmpty(suite.T(), result, "Test case '%s' should not return empty result", tt.testName)
		tt.validate(result)
	}
}

func (suite *MockerTestSuite) TestNumberNumber_ValidInputs() {
	tests := []struct {
		testName string
		params   []string
		validate func(result string)
	}{
		{
			testName: "no params - integer in default range [-1000, 1000]",
			params:   nil,
			validate: func(result string) {
				assert.NotContains(suite.T(), result, ".")
				v, err := strconv.ParseFloat(result, 64)
				assert.NoError(suite.T(), err)
				assert.True(suite.T(), v >= -1000 && v <= 1000)
			},
		},
		{
			testName: "decimals=2, min=0, max=100 - float in range with 2 decimal places",
			params:   []string{"2", "0", "100"},
			validate: func(result string) {
				v, err := strconv.ParseFloat(result, 64)
				assert.NoError(suite.T(), err)
				assert.True(suite.T(), v >= 0 && v <= 100)
				parts := strings.SplitN(result, ".", 2)
				assert.Len(suite.T(), parts, 2)
				assert.Len(suite.T(), parts[1], 2)
			},
		},
		{
			testName: "min=max=5 - returns exactly 5",
			params:   []string{"0", "5", "5"},
			validate: func(result string) {
				assert.Equal(suite.T(), "5", result)
			},
		},
		{
			testName: "empty decimals param - defaults to 0 decimals",
			params:   []string{"", "10", "20"},
			validate: func(result string) {
				assert.NotContains(suite.T(), result, ".")
				v, err := strconv.ParseFloat(result, 64)
				assert.NoError(suite.T(), err)
				assert.True(suite.T(), v >= 10 && v <= 20)
			},
		},
		{
			testName: "negative range - result within [-50, -10]",
			params:   []string{"0", "-50", "-10"},
			validate: func(result string) {
				v, err := strconv.ParseFloat(result, 64)
				assert.NoError(suite.T(), err)
				assert.True(suite.T(), v >= -50 && v <= -10)
			},
		},
		{
			testName: "decimals=3, min=1.5, max=1.5 - returns exactly 1.500",
			params:   []string{"3", "1.5", "1.5"},
			validate: func(result string) {
				assert.Equal(suite.T(), "1.500", result)
			},
		},
	}

	for _, tt := range tests {
		result, err := suite.m.Generate("Number.number", tt.params)
		assert.NoError(suite.T(), err, "Test case '%s' should not return error", tt.testName)
		assert.NotEmpty(suite.T(), result, "Test case '%s' should not return empty result", tt.testName)
		tt.validate(result)
	}
}

func (suite *MockerTestSuite) TestPaymentCreditCardCvv_ValidInputs() {
	cvvRegex := regexp.MustCompile(`^\d{3}$`)

	tests := []struct {
		testName string
		params   []string
		validate func(result string)
	}{
		{
			testName: "no params - produces a 3-digit string",
			params:   nil,
			validate: func(result string) {
				assert.Regexp(suite.T(), cvvRegex, result)
			},
		},
		{
			testName: "multiple calls all produce valid 3-digit strings",
			params:   nil,
			validate: func(result string) {
				assert.Regexp(suite.T(), cvvRegex, result)
				for range 10 {
					r, err := suite.m.Generate("Payment.creditCardCvv", nil)
					assert.NoError(suite.T(), err)
					assert.Regexp(suite.T(), cvvRegex, r)
				}
			},
		},
	}

	for _, tt := range tests {
		result, err := suite.m.Generate("Payment.creditCardCvv", tt.params)
		assert.NoError(suite.T(), err, "Test case '%s' should not return error", tt.testName)
		assert.NotEmpty(suite.T(), result, "Test case '%s' should not return empty result", tt.testName)
		tt.validate(result)
	}
}

func (suite *MockerTestSuite) TestPersonCpf_ValidInputs() {
	cpfRegex := regexp.MustCompile(`^\d{3}\.\d{3}\.\d{3}-\d{2}$`)

	tests := []struct {
		testName string
		params   []string
		validate func(result string)
	}{
		{
			testName: "no params - format matches XXX.XXX.XXX-XX",
			params:   nil,
			validate: func(result string) {
				assert.Regexp(suite.T(), cpfRegex, result)
			},
		},
		{
			testName: "no params - checksum digits are valid",
			params:   nil,
			validate: func(result string) {
				digits := strings.Map(func(r rune) rune {
					if r >= '0' && r <= '9' {
						return r
					}
					return -1
				}, result)
				assert.Len(suite.T(), digits, 11)
				d := make([]int, 11)
				for i, c := range digits {
					d[i] = int(c - '0')
				}
				assert.Equal(suite.T(), calculateChecksum(d[:9], []int{10, 9, 8, 7, 6, 5, 4, 3, 2}), d[9])
				assert.Equal(suite.T(), calculateChecksum(d[:10], []int{11, 10, 9, 8, 7, 6, 5, 4, 3, 2}), d[10])
			},
		},
	}

	for _, tt := range tests {
		result, err := suite.m.Generate("Person.cpf", tt.params)
		assert.NoError(suite.T(), err, "Test case '%s' should not return error", tt.testName)
		assert.NotEmpty(suite.T(), result, "Test case '%s' should not return empty result", tt.testName)
		tt.validate(result)
	}
}

func (suite *MockerTestSuite) TestRegexRegex_ValidInputs() {
	tests := []struct {
		testName string
		params   []string
		validate func(result string)
	}{
		{
			testName: "3 lowercase letters",
			params:   []string{"/[a-z]{3}/"},
			validate: func(result string) {
				assert.Regexp(suite.T(), regexp.MustCompile(`^[a-z]{3}$`), result)
			},
		},
		{
			testName: "4 digits",
			params:   []string{"/[0-9]{4}/"},
			validate: func(result string) {
				assert.Regexp(suite.T(), regexp.MustCompile(`^\d{4}$`), result)
			},
		},
		{
			testName: "fixed string pattern",
			params:   []string{"/abc/"},
			validate: func(result string) {
				assert.Equal(suite.T(), "abc", result)
			},
		},
		{
			testName: "pattern with escaped slash",
			params:   []string{"/a\\/b/"},
			validate: func(result string) {
				assert.Equal(suite.T(), "a/b", result)
			},
		},
	}

	for _, tt := range tests {
		result, err := suite.m.Generate("Regex.regex", tt.params)
		assert.NoError(suite.T(), err, "Test case '%s' should not return error", tt.testName)
		assert.NotEmpty(suite.T(), result, "Test case '%s' should not return empty result", tt.testName)
		tt.validate(result)
	}
}
