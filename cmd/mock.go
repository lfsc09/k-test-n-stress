package cmd

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/lfsc09/k-test-n-stress/mock"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var mockCmd = &cobra.Command{
	Use:   "mock",
	Short: "Generate mock data based on requested function names in the values of the parsed json object",
	Run: func(cmd *cobra.Command, args []string) {
		saveTo := viper.GetBool("saveTo")
		parseStr := viper.GetString("parse")
		parseFrom := viper.GetString("parseFrom")
		preserveFolderStructure := viper.GetBool("preserveFolderStructure")

		if parseStr == "" && parseFrom == "" {
			log.Fatalln("Nothing to be parsed. Ask for help -h or --help")
		} else if parseStr != "" && parseFrom != "" {
			log.Fatalln("Please provide only one of the two options: --parse or --parseFrom")
		}

		if preserveFolderStructure && parseFrom == "" {
			log.Fatalln("The --preserveFolderStructure option is only available when using --parseFrom")
		}

		// Parse single string object from `--parseStr`
		if parseStr != "" {
			var parseMap map[string]interface{}
			if err := json.Unmarshal([]byte(parseStr), &parseMap); err != nil {
				log.Fatalf("Opss..failed to parse JSON from the provided --parse <string>: %v\n", err)
			}

			mocker := mock.New()
			if err := processMap(parseMap, mocker); err != nil {
				log.Fatalln(err)
			}

			if saveTo {
				var mu sync.Mutex
				createdDirs := make(map[string]bool, 1)
				if err := toFile(false, "mocked-data.json", "", &parseMap, &mu, &createdDirs); err != nil {
					log.Fatalln(err)
					return
				}
			} else {
				if err := toStdout(&parseMap); err != nil {
					log.Fatalln(err)
					return
				}
			}
		}

		// Parse object from `--parseFrom` files
		if parseFrom != "" {
			foundTemplateFiles, err := findTemplateFiles(parseFrom)
			if err != nil {
				log.Fatalf("Failed to find template files from the provided --parseFrom <string>: %v\n", err)
			}
			if len(foundTemplateFiles) == 0 {
				log.Fatalf("No template files found in the provided --parseFrom <string>: %v\n", parseFrom)
			}

			var wg sync.WaitGroup
			var mu sync.Mutex
			createdDirs := make(map[string]bool)
			for _, inPath := range foundTemplateFiles {

				wg.Add(1)
				go func(inPath string) {
					defer wg.Done()

					templateFileContent, err := os.ReadFile(inPath)
					if err != nil {
						log.Printf("Opss..failed to read --parseFrom <file>: %v\n", err)
						return
					}

					var parseMap map[string]interface{}
					if err = json.Unmarshal(templateFileContent, &parseMap); err != nil {
						log.Printf("Opss..failed to parse JSON from the provided --parseFrom <file>: %v\n", err)
						return
					}

					mocker := mock.New()
					if err := processMap(parseMap, mocker); err != nil {
						log.Println(err)
						return
					}

					if saveTo {
						if err := toFile(preserveFolderStructure, inPath, parseFrom, &parseMap, &mu, &createdDirs); err != nil {
							log.Println(err)
							return
						}
					} else {
						if err := toStdout(&parseMap); err != nil {
							log.Println(err)
							return
						}
					}
				}(inPath)
			}
			wg.Wait()
		}
	},
}

func init() {
	mockCmd.Flags().String("saveTo", "", "Write mock data to './out/*.json' files")
	mockCmd.Flags().String("parse", "", "Parse json object as string")
	mockCmd.Flags().String("parseFrom", "", "Parse mock data from '.template.json' files from a path, directory, or glob")
	mockCmd.Flags().Bool("preserveFolderStructure", false, "Preserve folder structure when saving files or flatten them")

	viper.BindPFlag("saveTo", mockCmd.Flags().Lookup("saveTo"))
	viper.BindPFlag("parse", mockCmd.Flags().Lookup("parse"))
	viper.BindPFlag("parseFrom", mockCmd.Flags().Lookup("parseFrom"))
	viper.BindPFlag("preserveFolderStructure", mockCmd.Flags().Lookup("preserveFolderStructure"))

	rootCmd.AddCommand(mockCmd)
}

// Splits a raw string of format "func:arg1:arg2:...".
// It handles regex args wrapped with slashes (/.../) to avoid splitting inside them.
// Returns: function name, and slice of parameter strings.
func interpretString(rawValue string) (string, []string) {
	if rawValue == "" {
		return "", nil
	}
	var parts []string
	var buf strings.Builder
	inRegex := false
	for _, char := range rawValue {
		if char == '/' {
			inRegex = !inRegex
			// Always include slash
			buf.WriteByte(byte(char))
			continue
		}
		// If ':' outside regex — treat as delimiter
		if char == ':' && !inRegex {
			parts = append(parts, buf.String())
			// Start building next segment
			buf.Reset()
			continue
		}
		// Default: build the current token
		buf.WriteByte(byte(char))
	}
	// Add the final piece (there's no trailing `:`)
	if buf.Len() > 0 {
		parts = append(parts, buf.String())
	}
	return parts[0], parts[1:]
}

// Iterates through the parsed json map and processes each value.
// It replaces string values with generated mock data based on the function name and parameters.
// It handles nested maps and arrays of strings or maps.
// Returns an error if any value is not a string or map.
func processMap(parseMap map[string]interface{}, mocker *mock.Mock) error {
	for objKey, objValue := range parseMap {
		switch typedValue := objValue.(type) {
		case string:
			functionName, params := interpretString(typedValue)
			mockValue, err := mocker.Generate(functionName, params)
			if err != nil {
				return err
			}
			parseMap[objKey] = mockValue
		case map[string]interface{}:
			err := processMap(typedValue, mocker)
			if err != nil {
				return err
			}
		case []interface{}:
			for itemKey, item := range typedValue {
				if itemStr, ok := item.(string); ok {
					functionName, params := interpretString(itemStr)
					mockValue, err := mocker.Generate(functionName, params)
					if err != nil {
						return err
					}
					typedValue[itemKey] = mockValue
				} else if itemMap, ok := item.(map[string]interface{}); ok {
					err := processMap(itemMap, mocker)
					if err != nil {
						return err
					}
				} else {
					return fmt.Errorf("Value '%v' is not a string or map", item)
				}
			}
		default:
			return fmt.Errorf("Value '%v' is not a string or map", typedValue)
		}
	}
	return nil
}

// Returns all *.template.json files from a path, directory, or glob.
// It's recursive for directories, and respects any wildcard pattern.
func findTemplateFiles(input string) ([]string, error) {
	var matchedFiles []string
	info, err := os.Stat(input)
	// Check if input exists and is a directory — if so, walk recursively
	if err == nil && info.IsDir() {
		err := filepath.Walk(input, func(path string, fi os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if !fi.IsDir() && strings.HasSuffix(path, ".template.json") {
				matchedFiles = append(matchedFiles, path)
			}
			return nil
		})
		if err != nil {
			return nil, fmt.Errorf("Error walking through directory: %v", err)
		}
		return matchedFiles, nil
	}
	// If it's not a directory, use filepath.Glob to match pattern (may include wildcard)
	globMatches, err := filepath.Glob(input)
	if err != nil {
		return nil, fmt.Errorf("Error matching pattern: %v", err)
	}
	for _, file := range globMatches {
		info, err := os.Stat(file)
		if err != nil {
			continue
		}
		if !info.IsDir() && strings.HasSuffix(file, ".template.json") {
			matchedFiles = append(matchedFiles, file)
		}
	}
	return matchedFiles, nil
}

// Writes the generated mock data to a file.
// It creates the directory structure if it doesn't exist.
// If `preserveFolderStructure` is true, it keeps the original folder structure.
func toFile(preserveFolderStructure bool, inPath string, parseFrom string, result *map[string]interface{}, mu *sync.Mutex, createdDirs *map[string]bool) error {
	prettyJSON, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return fmt.Errorf("Error marshalling JSON: %v", err)
	}

	outName := strings.Replace(filepath.Base(inPath), ".template.json", ".json", 1)
	var outPath string

	if preserveFolderStructure {
		relPath, err := filepath.Rel(parseFrom, inPath)
		if err != nil {
			return fmt.Errorf("Failed to get relative path: %v\n", err)
		}
		relPath = strings.Replace(relPath, ".template.json", ".json", 1)
		outPath = filepath.Join("out", relPath)
	} else {
		outPath = filepath.Join("out", outName)
	}

	// Any created folders must be Thread-safe
	dir := filepath.Dir(outPath)
	mu.Lock()
	if !(*createdDirs)[dir] {
		if err := os.MkdirAll(dir, 0755); err != nil {
			mu.Unlock()
			return fmt.Errorf("Failed to create directory '%s': %v\n", dir, err)
		}
		(*createdDirs)[dir] = true
	}
	mu.Unlock()

	err = os.WriteFile(outPath, prettyJSON, 0644)
	if err != nil {
		return fmt.Errorf("Failed to write result to '%s': %v", outPath, err)
	}
	return nil
}

// Writes the generated mock data to stdout.
// It formats the JSON with indentation for better readability.
func toStdout(result *map[string]interface{}) error {
	prettyJSON, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return fmt.Errorf("Error marshalling JSON: %v", err)
	}

	fmt.Println(string(prettyJSON))
	return nil
}
