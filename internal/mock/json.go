package mock

import (
	"fmt"
	"regexp"
	"strconv"
)

// parseArrayItemBracketNumberRegex matches array item keys with bracketed numbers, e.g. "phones[5]" and captures the key and number.
var parseArrayItemBracketNumberRegex *regexp.Regexp = regexp.MustCompile(`^(.+)\[(\d+)\]$`)

type TemplateJsonNodeType int

const (
	NodeObject TemplateJsonNodeType = iota
	NodeArray
	NodeString
)

type TemplateJsonNode struct {
	Type     TemplateJsonNodeType
	Key      string              // Sanitized JSON object key
	Value    string              // JSON object value (Only for NodeString type)
	Children []*TemplateJsonNode // For NodeObject and NodeArray types
	Repeat   uint                // For key[n] syntax (Only for NodeObject and NodeString types)
}

// GenerateJSON generates mock data based on the TemplateJsonNode structure and writes the output as JSON to the provided writer.
func (tn *TemplateJsonNode) GenerateJSON(faker *Faker, stats *DebugStats, bw *BufferWriter) error {
	switch tn.Type {
	case NodeObject:
		// Generate array of objects if [n] syntax is used in the key, otherwise just one object
		if tn.Repeat > 1 {
			if err := bw.WriteJSONRaw("[", stats); err != nil {
				return err
			}
		}
		for r := range tn.Repeat {
			if r > 0 {
				if err := bw.WriteJSONRaw(",", stats); err != nil {
					return err
				}
			}
			if err := bw.WriteJSONRaw("{", stats); err != nil {
				return err
			}
			for i, child := range tn.Children {
				if i > 0 {
					if err := bw.WriteJSONRaw(",", stats); err != nil {
						return err
					}
				}
				keyStr := fmt.Sprintf("%q:", child.Key)
				if err := bw.WriteJSONRaw(keyStr, stats); err != nil {
					return err
				}
				if err := child.GenerateJSON(faker, stats, bw); err != nil {
					return err
				}
			}
			if err := bw.WriteJSONRaw("}", stats); err != nil {
				return err
			}
		}
		if tn.Repeat > 1 {
			if err := bw.WriteJSONRaw("]", stats); err != nil {
				return err
			}
		}
	case NodeArray:
		if err := bw.WriteJSONRaw("[", stats); err != nil {
			return err
		}
		for i, child := range tn.Children {
			if i > 0 {
				if err := bw.WriteJSONRaw(",", stats); err != nil {
					return err
				}
			}
			if err := child.GenerateJSON(faker, stats, bw); err != nil {
				return err
			}
		}
		if err := bw.WriteJSONRaw("]", stats); err != nil {
			return err
		}
	case NodeString:
		// Compile the string value into mock blocks
		mockBlocks, err := CompileMockBlocks(tn.Value)
		if err != nil {
			return err
		}

		// Generate array of values if [n] syntax is used in the key, otherwise just one value
		if tn.Repeat > 1 {
			if err := bw.WriteJSONRaw("[", stats); err != nil {
				return err
			}
		}
		for r := range tn.Repeat {
			if r > 0 {
				if err := bw.WriteJSONRaw(",", stats); err != nil {
					return err
				}
			}
			mockedStr, err := ExecuteMockBlocks(mockBlocks, faker)
			if err != nil {
				return err
			}
			if err := bw.WriteJSONEncode(mockedStr, stats); err != nil {
				return err
			}
		}
		if tn.Repeat > 1 {
			if err := bw.WriteJSONRaw("]", stats); err != nil {
				return err
			}
		}
		if stats != nil {
			stats.Generated.Add(uint32(tn.Repeat))
		}
	}

	return nil
}

// CompileJSONTemplate walks through the raw JSON template and compiles it into a TemplateJsonNode tree structure, and
// calculates the total number of items to be generated based on the presence of [n] patterns in the keys.
func CompileJSONTemplate(obj map[string]any, node *TemplateJsonNode) (uint, error) {
	var totalGenerate uint = 0

	for key, value := range obj {
		// Handle key[n] syntax
		keyRepeat := uint(1)
		cleanKey := key
		if matches := parseArrayItemBracketNumberRegex.FindStringSubmatch(key); len(matches) == 3 {
			cleanKey = matches[1]
			keyRepeat64, _ := strconv.ParseUint(matches[2], 10, 64)
			keyRepeat = uint(keyRepeat64)
		}

		childNode := &TemplateJsonNode{Key: cleanKey, Repeat: keyRepeat}

		switch typedValue := value.(type) {
		case map[string]any:
			childNode.Type = NodeObject
			childGenerate, err := CompileJSONTemplate(typedValue, childNode)
			if err != nil {
				return 0, err
			}
			totalGenerate += childGenerate * keyRepeat
		case []any:
			childNode.Type = NodeArray
			if keyRepeat > 1 {
				return 0, fmt.Errorf("invalid key '%s': array item keys cannot use [n] syntax to specify repeat count", key)
			}
			for _, arrItem := range typedValue {
				arrItemNode := &TemplateJsonNode{Repeat: 1}
				switch typedArrItem := arrItem.(type) {
				case map[string]any:
					arrItemNode.Type = NodeObject
					childGenerate, err := CompileJSONTemplate(typedArrItem, arrItemNode)
					if err != nil {
						return 0, err
					}
					childNode.Children = append(childNode.Children, arrItemNode)
					totalGenerate += childGenerate
				case []any:
					return 0, fmt.Errorf("nested arrays are not supported in json template")
				case string:
					arrItemNode.Type = NodeString
					arrItemNode.Value = typedArrItem
					childNode.Children = append(childNode.Children, arrItemNode)
					totalGenerate++
				default:
					arrItemNode.Type = NodeString
					arrItemNode.Value = arrItem.(string)
					childNode.Children = append(childNode.Children, arrItemNode)
					totalGenerate++
				}
			}
		case string:
			childNode.Type = NodeString
			childNode.Value = typedValue
			totalGenerate += keyRepeat
		default:
			childNode.Type = NodeString
			childNode.Value = value.(string)
			totalGenerate += keyRepeat
		}

		node.Children = append(node.Children, childNode)
	}

	return totalGenerate, nil
}
