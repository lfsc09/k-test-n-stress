package mock

import (
	"fmt"
	"regexp"
	"strings"
)

// splitBlocksRegex matches all occurrences of {{ … }} in a string.
var splitBlocksRegex = regexp.MustCompile(`\{\{.*?\}\}`)

type BlockType int

const (
	LiteralBlock BlockType = iota
	DynamicBlock
)

type MockCall struct {
	FunctionName string
	Params       []string
}

type MockBlock struct {
	Type     BlockType
	RawValue string
	Calls    []*MockCall
}

// CompileMockBlocks takes a raw string value, and breaks it into a slice of MockBlock, where each MockBlock is either
// a literal block (a string of text that should be output verbatim) or a dynamic block
// (a string containing one or more function calls in the format "{{ functionName:{arg1}:{arg2}:{argN} | functionName:{arg1}:{arg2}:{argN} | ... }}").
func CompileMockBlocks(rawValue string) ([]*MockBlock, error) {
	compiledBlocks := []*MockBlock{}

	if rawValue == "" {
		return compiledBlocks, nil
	}

	if err := validateDynamicBlockSyntax(rawValue); err != nil {
		return nil, err
	}

	// Find the location (start and end indexes) of all dynamic blocks in the rawValue
	dynamicBlockIdxRanges := splitBlocksRegex.FindAllStringIndex(rawValue, -1)

	// No dynamic blocks found, treat the entire raw value as a single literal block
	if len(dynamicBlockIdxRanges) == 0 {
		compiledBlocks = append(compiledBlocks, &MockBlock{
			Type:     LiteralBlock,
			RawValue: rawValue,
		})
		return compiledBlocks, nil
	}

	currPos := 0
	for _, dynamicBlockIdxRange := range dynamicBlockIdxRanges {
		start, end := dynamicBlockIdxRange[0], dynamicBlockIdxRange[1]

		// Add any preceding literal block
		if start > currPos {
			compiledBlocks = append(compiledBlocks, &MockBlock{
				Type:     LiteralBlock,
				RawValue: rawValue[currPos:start],
			})
		}

		// Parse the dynamic block to extract function calls and parameters
		dynamicBlockStr := rawValue[start:end]
		blockCalls, err := parseDynamicBlock(dynamicBlockStr)
		if err != nil {
			return nil, err
		}

		// Add the dynamic block
		compiledBlocks = append(compiledBlocks, &MockBlock{
			Type:     DynamicBlock,
			RawValue: dynamicBlockStr,
			Calls:    blockCalls,
		})

		currPos = end
	}

	// Add any remaining literal block
	if currPos < len(rawValue) {
		compiledBlocks = append(compiledBlocks, &MockBlock{
			Type:     LiteralBlock,
			RawValue: rawValue[currPos:],
		})
	}

	return compiledBlocks, nil
}

// ExecuteMockBlocks takes a slice of MockBlock and executes the function calls in dynamic blocks using the provided faker,
// and returns the final string result with all dynamic blocks replaced by their executed values.
func ExecuteMockBlocks(blocks []*MockBlock, faker *Faker) (string, error) {
	var resultBuilder strings.Builder

	for _, block := range blocks {
		switch block.Type {
		case LiteralBlock:
			resultBuilder.WriteString(block.RawValue)
		case DynamicBlock:
			var blockResultBuilder strings.Builder

			// Each function call executes sequentially, but their values overwrite each other and only the result of the last function call is returned as the value of the block
			for _, call := range block.Calls {
				blockResultBuilder.Reset()
				mockedValue, err := faker.Generate(call.FunctionName, call.Params)
				if err != nil {
					return "", err
				}
				blockResultBuilder.WriteString(mockedValue)
			}

			resultBuilder.WriteString(blockResultBuilder.String())
		}
	}

	return resultBuilder.String(), nil
}

// validateDynamicBlockSyntax checks the syntax of dynamic blocks in the rawValue string.
// It ensures that all '{{' are properly closed with '}}' and that there are no nested or overlapping blocks.
func validateDynamicBlockSyntax(rawValue string) error {
	inBlock := false
	rawValueLen := len(rawValue)
	idx := 0
	for idx < rawValueLen {
		if rawValue[idx] == '{' && idx+1 < rawValueLen && rawValue[idx+1] == '{' {
			if inBlock {
				return fmt.Errorf("invalid template: nested '{{' at position %d", idx)
			}
			inBlock = true
			idx += 2
		} else if rawValue[idx] == '}' && idx+1 < rawValueLen && rawValue[idx+1] == '}' {
			if !inBlock {
				return fmt.Errorf("invalid template: unexpected '}}' at position %d (no matching '{{')", idx)
			}
			inBlock = false
			idx += 2
		} else {
			idx++
		}
	}
	if inBlock {
		return fmt.Errorf("invalid template: unclosed '{{' (missing '}}')")
	}
	return nil
}

// parseDynamicBlock takes a raw dynamic block string in the format "{{ functionName:{arg1}:{arg2}:{argN} | functionName:{arg1}:{arg2}:{argN} | ... }}".
// A dynamic block may contain multiple function calls separated by "|".
// It returns a slice of MockCall, where each MockCall contains the function name and its parameters.
func parseDynamicBlock(rawValue string) ([]*MockCall, error) {
	trimmed := strings.TrimSpace(rawValue)

	inner := strings.TrimSpace(trimmed[2 : len(trimmed)-2])
	if inner == "" {
		return nil, fmt.Errorf("empty dynamic block")
	}

	// Split inner content on top-level '|' (not inside parameters 'fn:{…}')
	// Track curly brace depth so that '|' inside parameter values is treated as a literal character, not as a function separator
	var functionSegmentsStr []string
	start := 0
	curlyBraceDepth := 0

	for i := 0; i < len(inner); i++ {
		switch inner[i] {
		case '{':
			curlyBraceDepth++
		case '}':
			if curlyBraceDepth > 0 {
				curlyBraceDepth--
			}
		case '|':
			if curlyBraceDepth == 0 {
				functionSegmentsStr = append(functionSegmentsStr, inner[start:i])
				start = i + 1
			}
		}
	}
	// Flush the last segment
	functionSegmentsStr = append(functionSegmentsStr, inner[start:])

	// Build block calls
	var blockCalls []*MockCall
	for _, functionSegmentStr := range functionSegmentsStr {
		funcName, params, err := parseFunctionAndParams(functionSegmentStr)
		if err != nil {
			return nil, err
		}
		// A '|' separator with no function call before or after it is invalid syntax
		if funcName == "" {
			return nil, fmt.Errorf("'|' separator must always separate two function calls")
		}
		blockCalls = append(blockCalls, &MockCall{
			FunctionName: funcName,
			Params:       params,
		})
	}

	return blockCalls, nil
}

// parseFunctionAndParams parses "funcName:{arg1}:{arg2}:{argN}" and returns the function name and parameter slice.
// Parameters must be enclosed in '{}'; empty parameters may be written as 'fn:{}' or as an empty slot 'fn:'.
// Regex parameters are written as {/pattern/} — backslash-escaped slashes (\/) inside the pattern are not treated as closing delimiters.
// The '{}' braces are stripped from returned parameter values.
func parseFunctionAndParams(rawValue string) (string, []string, error) {
	trimmed := strings.TrimSpace(rawValue)
	if trimmed == "" {
		return "", []string{}, nil
	}

	// Extract function name
	firstColonIdx := strings.IndexByte(trimmed, ':')
	if firstColonIdx < 0 {
		// Just a function name, zero parameters
		return trimmed, []string{}, nil
	}
	if firstColonIdx == 0 {
		return "", nil, fmt.Errorf("empty function name")
	}
	funcName := trimmed[:firstColonIdx]
	restStr := trimmed[firstColonIdx+1:]
	restStrLen := len(restStr)

	// Parse parameters
	// Stack-allocated backing array when ≤4 params, if >4 params, a heap allocation will occur
	var paramsBuf [4]string
	params := paramsBuf[:0]

	var buf strings.Builder
	buf.Grow(32)

	currIdx := 0
	for {
		// End of input, flush whatever is in the string buffer as the last param
		if currIdx >= restStrLen {
			params = append(params, buf.String())
			break
		}

		switch restStr[currIdx] {
		// If rune is ':' flush the string buffer as a parameter and continue to the next one
		case ':':
			params = append(params, buf.String())
			buf.Reset()
			currIdx++

		// If rune is '{' start parsing a parameter value until the closing '}'
		case '{':
			if buf.Len() > 0 {
				return "", nil, fmt.Errorf("unexpected '{' in parameter")
			}

			// Move to the next rune without writing '{' to the string buffer
			currIdx++

			if currIdx < restStrLen && restStr[currIdx] == '/' {
				// If the current rune is now '/' then we are parsing a regex parameter, so we need to keep parsing until the closing '/'
				buf.WriteByte('/')
				currIdx++
				isRegexSlashClosed := false

				for currIdx < restStrLen {
					if restStr[currIdx] == '\\' && currIdx+1 < restStrLen {
						// If we encounter a backslash, we need to check the next character and copy both verbatim to the buffer, to allow escaping of '/' and '\'
						buf.WriteByte(restStr[currIdx])
						buf.WriteByte(restStr[currIdx+1])
						currIdx += 2
					} else if restStr[currIdx] == '/' {
						// If we encounter an unescaped '/', this is the end of the regex pattern
						buf.WriteByte('/')
						currIdx++
						isRegexSlashClosed = true
						break
					} else {
						// Otherwise, keep adding characters to the regex pattern
						buf.WriteByte(restStr[currIdx])
						currIdx++
					}
				}

				if !isRegexSlashClosed {
					return "", nil, fmt.Errorf("unclosed regex: missing closing '/'")
				}

				// Process the '}' after the regex pattern to close the parameter value
				if currIdx >= restStrLen || restStr[currIdx] != '}' {
					return "", nil, fmt.Errorf("expected '}' after regex closing '/'")
				}

				// Move to the next rune without writing '}' to the string buffer
				currIdx++
			} else {
				// Otherwise, we are parsing a normal parameter value, so we keep parsing until the closing '}'
				for currIdx < restStrLen && restStr[currIdx] != '}' {
					if restStr[currIdx] == '{' {
						return "", nil, fmt.Errorf("unexpected nested '{' in parameter value")
					}
					buf.WriteByte(restStr[currIdx])
					currIdx++
				}

				// No closing '}' found
				if currIdx >= restStrLen {
					return "", nil, fmt.Errorf("unclosed '{' in parameter value")
				}

				// Move to the next rune without writing '}' to the string buffer
				currIdx++
			}

			// After '}' the next character must be ':' (another param) or end-of-input
			if currIdx < restStrLen && restStr[currIdx] != ':' {
				return "", nil, fmt.Errorf("unexpected character '%c' after '}'", restStr[currIdx])
			}

		// Bare character (not allowed in parameter position)
		default:
			return "", nil, fmt.Errorf("parameter value must be wrapped in {…} or {/…/}, got bare character '%c'", restStr[currIdx])
		}
	}

	return funcName, params, nil
}
