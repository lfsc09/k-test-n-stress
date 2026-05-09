package mock

import (
	"fmt"
)

type TemplateCsv struct {
	Repeat  uint
	Columns []*TemplateCsvNode
}

type TemplateCsvNode struct {
	ColName string
	Value   string
}

// GenerateCSV generates mock data based on the TemplateCsv structure and writes the output as CSV to the provided writer.
func (blueprint *TemplateCsv) GenerateCSV(faker *Faker, stats *DebugStats, bw *BufferWriter) error {
	// Write CSV header
	for i, colNode := range blueprint.Columns {
		if i > 0 {
			if err := bw.WriteCSVRaw(",", stats); err != nil {
				return err
			}
		}
		if err := bw.WriteCSV(colNode.ColName, stats); err != nil {
			return err
		}
	}
	if err := bw.WriteCSVRaw("\n", stats); err != nil {
		return err
	}

	// Generate rows based on the blueprint
	for range blueprint.Repeat {
		for i, colNode := range blueprint.Columns {
			if i > 0 {
				if err := bw.WriteCSVRaw(",", stats); err != nil {
					return err
				}
			}

			// Compile the string value into mock blocks
			mockBlocks, err := CompileMockBlocks(colNode.Value)
			if err != nil {
				return err
			}

			mockedStr, err := ExecuteMockBlocks(mockBlocks, faker)
			if err != nil {
				return err
			}

			if err := bw.WriteCSV(mockedStr, stats); err != nil {
				return err
			}
		}
		if err := bw.WriteCSVRaw("\n", stats); err != nil {
			return err
		}
	}

	return nil
}

// CompileCSVTemplate compiles the CSV template from a map of column definitions and calculates
// the total number of rows to be generated based on the presence of [n] patterns in the column keys.
func CompileCSVTemplate(obj map[string]any, blueprint *TemplateCsv) (uint, error) {
	var totalGenerate uint = 0

	for key, value := range obj {
		switch typedValue := value.(type) {
		case string:
			blueprint.Columns = append(blueprint.Columns, &TemplateCsvNode{
				ColName: key,
				Value:   typedValue,
			})
			totalGenerate++
		default:
			return 0, fmt.Errorf("invalid value for key '%s': expected a string in the format \"value\"", key)
		}
	}

	return totalGenerate * blueprint.Repeat, nil
}
