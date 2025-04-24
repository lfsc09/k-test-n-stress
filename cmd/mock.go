package cmd

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
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
		parseFromFiles := viper.GetStringSlice("parseFrom")

		if parseStr == "" && len(parseFromFiles) == 0 {
			log.Fatalln("Nothing to be parsed. Ask for help -h or --help")
		} else if parseStr != "" && len(parseFromFiles) != 0 {
			log.Fatalln("Please provide only one of the two options: --parse or --parseFrom")
		}

		// Parse single string object (From CLI)
		if parseStr != "" {
			var parseMap map[string]interface{}
			if err := json.Unmarshal([]byte(parseStr), &parseMap); err != nil {
				log.Fatalf("Opss..failed to parse JSON from the provided --parse <string>: %v\n", err)
			}

			mocker := mock.New()
			if err := processMap(parseMap, mocker); err != nil {
				log.Fatalln(err)
			}

			if err := processOutput(saveTo, "mocked-data.json", &parseMap); err != nil {
				log.Fatalln(err)
			}
		}

		// Parse from files
		if len(parseFromFiles) != 0 {
			var wg sync.WaitGroup
			for _, filename := range parseFromFiles {

				wg.Add(1)
				go func(filename string) {
					defer wg.Done()

					if !strings.HasSuffix(filename, ".template.json") {
						log.Printf("File '%s' must have a .template.json extension\n", filename)
						return
					}

					fileContent, err := os.ReadFile(filename)
					if err != nil {
						log.Printf("Opss..failed to read --parseFrom <file>: %v\n", err)
						return
					}

					var parseMap map[string]interface{}
					if err = json.Unmarshal(fileContent, &parseMap); err != nil {
						log.Printf("Opss..failed to parse JSON from the provided --parseFrom <file>: %v\n", err)
						return
					}

					mocker := mock.New()
					if err := processMap(parseMap, mocker); err != nil {
						log.Println(err)
						return
					}

					outFilename := strings.Replace(filename, ".template.json", ".json", 1)
					log.Printf("Parsing file %v -> %v\n", filename, outFilename)
					if err := processOutput(saveTo, outFilename, &parseMap); err != nil {
						log.Println(err)
						return
					}
				}(filename)
			}
			wg.Wait()
			log.Println("All files processed")
		}
	},
}

func init() {
	mockCmd.Flags().String("saveTo", "", "Write mock data to '*.json' files")
	mockCmd.Flags().String("parse", "", "Parse json object as string")
	mockCmd.Flags().String("parseFrom", "", "Parse mock data from '.template.json' files")

	viper.BindPFlag("saveTo", mockCmd.Flags().Lookup("saveTo"))
	viper.BindPFlag("parse", mockCmd.Flags().Lookup("parse"))
	viper.BindPFlag("parseFrom", mockCmd.Flags().Lookup("parseFrom"))

	rootCmd.AddCommand(mockCmd)
}

// Splits a raw string of format "func:arg1:arg2:..."
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

func processOutput(saveTo bool, filename string, result *map[string]interface{}) error {
	if saveTo {
		return toFile(filename, result)
	} else {
		return toStdout(result)
	}
}

func toFile(filename string, result *map[string]interface{}) error {
	prettyJSON, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return fmt.Errorf("Error marshalling JSON: %v", err)
	}

	err = os.WriteFile(filename, prettyJSON, 0644)
	if err != nil {
		return fmt.Errorf("Failed to write result to '%s': %v", filename, err)
	}
	return nil
}

func toStdout(result *map[string]interface{}) error {
	prettyJSON, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return fmt.Errorf("Error marshalling JSON: %v", err)
	}
	fmt.Println(string(prettyJSON))
	return nil
}
