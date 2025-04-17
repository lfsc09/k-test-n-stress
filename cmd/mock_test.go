package cmd

import (
	"bytes"
	"io"
	"log"
	"os"
	"path/filepath"
	"testing"

	"github.com/lfsc09/k-test-n-stress/mock"
	"github.com/stretchr/testify/assert"
)

func TestInterpretString(t *testing.T) {
	tests := []struct {
		input    string
		funcName string
		params   []string
		err      error
	}{
		{"Address.city", "Address.city", nil, nil},
		{"Boolean.booleanWithChance:10", "Boolean.booleanWithChance", []string{"10"}, nil},
		{"Function.with:multiple:params", "Function.with", []string{"multiple", "params"}, nil},
		{"", "", nil, nil},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			funcName, params, err := interpretString(tt.input)
			assert.Equal(t, tt.funcName, funcName)
			assert.Equal(t, tt.params, params)
			assert.Equal(t, tt.err, err)
		})
	}
}

func TestProcessMap(t *testing.T) {
	// Redirect log output to avoid printing during tests
	oldLogger := log.Writer()
	log.SetOutput(io.Discard)
	defer log.SetOutput(oldLogger)

	// Setup a mock implementation
	mockObj := mock.New()

	tests := []struct {
		name    string
		input   map[string]interface{}
		wantErr bool
	}{
		{
			name: "string value",
			input: map[string]interface{}{
				"key": "Address.city",
			},
			wantErr: false,
		},
		{
			name: "nested map",
			input: map[string]interface{}{
				"nested": map[string]interface{}{
					"key": "Address.city",
				},
			},
			wantErr: false,
		},
		{
			name: "array of strings",
			input: map[string]interface{}{
				"array": []interface{}{"Address.city", "Name.firstName"},
			},
			wantErr: false,
		},
		{
			name: "array of maps",
			input: map[string]interface{}{
				"array": []interface{}{
					map[string]interface{}{"key": "Address.city"},
				},
			},
			wantErr: false,
		},
		{
			name: "invalid type in array",
			input: map[string]interface{}{
				"array": []interface{}{123},
			},
			wantErr: true,
		},
		{
			name:    "invalid type",
			input:   map[string]interface{}{"key": 123},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a copy of the input to avoid modifying the test case
			inputCopy := make(map[string]interface{})
			for k, v := range tt.input {
				inputCopy[k] = v
			}

			err := processMap(inputCopy, mockObj)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestProcessOutput(t *testing.T) {
	data := map[string]interface{}{"key": "value"}

	t.Run("to stdout", func(t *testing.T) {
		oldStdout := os.Stdout
		r, w, _ := os.Pipe()
		os.Stdout = w

		err := processOutput(false, "test.json", &data)
		w.Close()
		os.Stdout = oldStdout

		var buf bytes.Buffer
		io.Copy(&buf, r)
		output := buf.String()

		assert.NoError(t, err)
		assert.Contains(t, output, `"key": "value"`)
	})

	t.Run("to file", func(t *testing.T) {
		// Create a temporary directory
		tempDir, err := os.MkdirTemp("", "test")
		assert.NoError(t, err)
		defer os.RemoveAll(tempDir)

		filename := filepath.Join(tempDir, "test.json")

		err = processOutput(true, filename, &data)
		assert.NoError(t, err)

		// Check if file exists and contains the expected content
		content, err := os.ReadFile(filename)
		assert.NoError(t, err)
		assert.Contains(t, string(content), `"key": "value"`)
	})
}

func TestToFile(t *testing.T) {
	data := map[string]interface{}{"key": "value"}

	// Create a temporary directory
	tempDir, err := os.MkdirTemp("", "test")
	assert.NoError(t, err)
	defer os.RemoveAll(tempDir)

	filename := filepath.Join(tempDir, "test.json")

	err = toFile(filename, &data)
	assert.NoError(t, err)

	// Check if file exists and contains the expected content
	content, err := os.ReadFile(filename)
	assert.NoError(t, err)
	assert.Contains(t, string(content), `"key": "value"`)

	// Test error handling with invalid path
	err = toFile("/invalid/path/test.json", &data)
	assert.Error(t, err)
}

func TestToStdout(t *testing.T) {
	data := map[string]interface{}{"key": "value"}

	// Redirect stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := toStdout(&data)
	w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	io.Copy(&buf, r)
	output := buf.String()

	assert.NoError(t, err)
	assert.Contains(t, output, `"key": "value"`)

	// Test marshaling error
	invalidData := map[string]interface{}{
		"invalid": make(chan int), // channels can't be marshaled to JSON
	}
	err = toStdout(&invalidData)
	assert.Error(t, err)
}
