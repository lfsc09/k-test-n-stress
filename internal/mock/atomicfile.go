package mock

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const defaultAtomicTempFilePrefix = ".ktns-tmp"

// AtomicFile represents a file being written to with an atomic commit or abort.
type AtomicFile struct {
	Path string
	File *os.File
}

// NewAtomicFile creates an AtomicFile struct for the given finalPath.
// It opens a temp file in the same directory as finalPath.
func NewAtomicFile(finalPath string) (*AtomicFile, error) {
	dir := filepath.Dir(finalPath)
	f, err := os.CreateTemp(dir, defaultAtomicTempFilePrefix+"*")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp file in '%s': %w", dir, err)
	}

	return &AtomicFile{
		Path: finalPath,
		File: f,
	}, nil
}

// Commit closes the file and renames it to the final path. If closing the file fails, it returns an error and does not rename.
func (af *AtomicFile) Commit() error {
	if cerr := af.File.Close(); cerr != nil {
		return cerr
	}
	return os.Rename(af.File.Name(), af.Path)
}

// Abort closes the file and removes the temp file. It ignores any errors since this is meant to be called in a defer after a failed write.
func (af *AtomicFile) Abort() {
	_ = af.File.Close()
	_ = os.Remove(af.File.Name())
}

// ResolveJSONToFilePath determines the output file path for JSON output based on the provided flags and template filename. It follows these rules:
// 1. If --to-json-file is set with a non-empty filename, use that.
// 2. If --to-json-file is set but the filename is empty, and --parse-json-file is set, derive the output filename by removing ".template" from the template filename.
// 3. If --to-json-file is set but the filename is empty, and --parse-json-file is not set, default to "output.json" in the current working directory.
func ResolveJSONToFilePath(parseJsonFileSet bool, parseJsonFile string, toJsonFile string) (string, error) {
	// Specific file provided via --to-json-file, use it directly
	if toJsonFile != "" {
		return toJsonFile, nil
	}

	// Default filename based on --parse-json-file template name
	if parseJsonFileSet {
		return strings.TrimSuffix(parseJsonFile, ".template.json") + ".json", nil
	}

	// Default filename in current working directory `output.json`
	dir, e := workingDir()
	if e != nil {
		return "", e
	}
	defaultPath := filepath.Join(dir, "output.json")
	return defaultPath, nil
}

// ResolveCSVToFilePath determines the output file path for CSV output based on the provided flags and template filename. It follows these rules:
// 1. If --to-csv-file is set with a non-empty filename, use that.
// 2. If --to-csv-file is set but the filename is empty, and --parse-csv-file is set, derive the output filename by removing ".template" from the template filename.
// 3. If --to-csv-file is set but the filename is empty, and --parse-csv-file is not set, default to "output.csv" in the current working directory.
func ResolveCSVToFilePath(parseCsvFileSet bool, parseCsvFile string, toCsvFile string) (string, error) {
	// Specific file provided via --to-csv-file, use it directly
	if toCsvFile != "" {
		return toCsvFile, nil
	}

	// Default filename based on --parse-csv-file template name
	if parseCsvFileSet {
		return strings.TrimSuffix(parseCsvFile, ".template.csv") + ".csv", nil
	}

	// Default filename in current working directory `output.csv`
	dir, e := workingDir()
	if e != nil {
		return "", e
	}
	defaultPath := filepath.Join(dir, "output.csv")
	return defaultPath, nil
}

// workingDir returns the current working directory.
// Used to resolve the default output path for --to-json-file and --to-csv-file
// when no explicit filename is provided and the source is not --parse-json-file.
func workingDir() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("could not determine working directory: %w", err)
	}
	return dir, nil
}
