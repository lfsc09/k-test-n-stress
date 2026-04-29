// Package mocker — randexp.go
//
// RandexpGenerator produces random strings that satisfy a given regular
// expression pattern. It works by parsing the pattern with the standard
// library's [regexp/syntax] package (Perl flag set) into an AST, simplifying
// it, and then recursively walking each node to emit random output.
//
// # Usage
//
//	gen, err := NewRandexpGenerator(`[A-Z]{2}[0-9]{4}`)
//	if err != nil {
//	    // pattern is syntactically invalid
//	}
//	s := gen.Generate(rng) // e.g. "BK7392"
//
// The same generator instance can be called from multiple goroutines concurrently.
// After construction the generator holds only the immutable parsed AST — all
// randomness is supplied by the caller-provided *rand.Rand on each Generate call.
// Each goroutine must pass its own *rand.Rand to avoid races on the source.
//
// # Covered AST nodes
//
//   - OpLiteral       — exact rune sequences, including case-folded variants ((?i))
//   - OpCharClass     — character classes such as [a-z], [^0-9], \d, \w, \s …
//   - OpAnyChar       — dot (.) with DOTALL flag
//   - OpAnyCharNotNL  — dot (.) without DOTALL flag
//   - OpQuest         — x?   (0 or 1 repetition)
//   - OpStar          — x*   (0 to maxRepeat repetitions)
//   - OpPlus          — x+   (1 to maxRepeat repetitions)
//   - OpRepeat        — x{n}, x{n,m}, x{n,}  (exact / bounded / unbounded)
//   - OpConcat        — concatenation of sub-expressions
//   - OpAlternate     — alternation  a|b|c  (one branch picked uniformly)
//   - OpCapture       — capturing group (transparent — child is generated)
//   - Anchors / zero-width assertions (^, $, \b, \B, OpEmptyMatch, OpNoMatch)
//     produce no output characters and are silently skipped.
//
// # Limitations
//
//   - Unbounded quantifiers (* + {n,}) are capped at [maxRepeat] repetitions
//     (default 10) to keep output finite. The cap is applied uniformly so output
//     length is bounded but the distribution is slightly skewed toward shorter
//     strings for open-ended patterns.
//
//   - Dot (. / OpAnyChar*) is restricted to printable ASCII [0x20, 0x7E].
//     True Unicode "any character" semantics are not emulated.
//
//   - Look-ahead and look-behind assertions are not supported by [regexp/syntax]
//     under the Perl flag set and will cause Parse to return an error.
//
//   - Back-references (\1, \k<name>) are not supported by [regexp/syntax] and
//     will similarly cause a parse error.
//
//   - Non-capturing group flags that change parsing semantics mid-pattern
//     (e.g. (?-i) toggling inside a group) are handled by [regexp/syntax]
//     transparently — FoldCase is propagated correctly per-node.
//
//   - The generator is not cryptographically random. Randomness is supplied
//     by the [math/rand/v2.Rand] passed to each Generate call.

package mocker

import (
	"fmt"
	"math/rand/v2"
	"regexp/syntax"
	"strings"
	"unicode"
)

// maxRepeat caps the number of repetitions for *, + and unbounded {n,} operators
// during generation to keep output finite and reasonable.
const maxRepeat = 10

// printableASCIIRanges covers [0x20, 0x7E] (space through tilde).
// Used for OpAnyChar / OpAnyCharNotNL so generated output stays legible.
var printableASCIIRanges = []rune{0x20, 0x7E}

// RandexpGenerator generates random strings that match a compiled regex pattern.
// It uses regexp/syntax to parse the pattern, then walks the resulting AST.
// Safe for concurrent use: the generator is immutable after construction;
// callers supply their own *rand.Rand on each Generate call.
type RandexpGenerator struct {
	re *syntax.Regexp
}

// NewRandexpGenerator parses pattern and returns a generator ready to call Generate.
func NewRandexpGenerator(pattern string) (*RandexpGenerator, error) {
	re, err := syntax.Parse(pattern, syntax.Perl)
	if err != nil {
		return nil, fmt.Errorf("Randexp: invalid pattern %q: %w", pattern, err)
	}
	// Simplify converts {0,1}→OpQuest, {0,}→OpStar, {1,}→OpPlus, etc.
	re = re.Simplify()
	return &RandexpGenerator{re: re}, nil
}

// Generate returns a new random string matching the pattern.
// rng must not be shared across goroutines; each goroutine should pass its own.
func (g *RandexpGenerator) Generate(rng *rand.Rand) string {
	var sb strings.Builder
	g.walk(&sb, g.re, rng)
	return sb.String()
}

func (g *RandexpGenerator) walk(sb *strings.Builder, re *syntax.Regexp, rng *rand.Rand) {
	switch re.Op {

	case syntax.OpLiteral:
		// re.Rune holds the literal runes in order.
		for _, r := range re.Rune {
			if re.Flags&syntax.FoldCase != 0 {
				// Case-insensitive flag: randomly pick upper or lower.
				if rng.IntN(2) == 0 {
					r = unicode.ToUpper(r)
				} else {
					r = unicode.ToLower(r)
				}
			}
			sb.WriteRune(r)
		}

	case syntax.OpCharClass:
		// re.Rune is a flattened list of [lo, hi, lo, hi, ...] Unicode ranges.
		sb.WriteRune(g.randFromClass(re.Rune, rng))

	case syntax.OpAnyCharNotNL: // dot . without DOTALL flag
		sb.WriteRune(g.randFromClass(printableASCIIRanges, rng))

	case syntax.OpAnyChar: // dot . with DOTALL flag
		sb.WriteRune(g.randFromClass(printableASCIIRanges, rng))

	case syntax.OpQuest: // x?  →  0 or 1
		if rng.IntN(2) == 0 {
			g.walk(sb, re.Sub[0], rng)
		}

	case syntax.OpStar: // x*  →  0 to maxRepeat
		n := rng.IntN(maxRepeat + 1)
		for range n {
			g.walk(sb, re.Sub[0], rng)
		}

	case syntax.OpPlus: // x+  →  1 to maxRepeat
		n := rng.IntN(maxRepeat) + 1
		for range n {
			g.walk(sb, re.Sub[0], rng)
		}

	case syntax.OpRepeat: // x{min,max}
		min := re.Min
		max := re.Max
		if max < 0 { // unbounded: {n,}
			max = min + maxRepeat
		}
		n := min
		if max > min {
			n = min + rng.IntN(max-min+1)
		}
		for range n {
			g.walk(sb, re.Sub[0], rng)
		}

	case syntax.OpConcat: // ab  →  generate each child in order
		for _, sub := range re.Sub {
			g.walk(sb, sub, rng)
		}

	case syntax.OpAlternate: // a|b  →  pick one child at random
		g.walk(sb, re.Sub[rng.IntN(len(re.Sub))], rng)

	case syntax.OpCapture: // (x)  →  transparent, just generate the child
		g.walk(sb, re.Sub[0], rng)

	// Anchors and zero-width assertions produce no output characters.
	case syntax.OpBeginText, syntax.OpEndText,
		syntax.OpBeginLine, syntax.OpEndLine,
		syntax.OpWordBoundary, syntax.OpNoWordBoundary,
		syntax.OpEmptyMatch, syntax.OpNoMatch:
		// intentionally empty
	}
}

// randFromClass picks a uniformly random rune from a flattened [lo, hi, ...] class.
func (g *RandexpGenerator) randFromClass(class []rune, rng *rand.Rand) rune {
	total := 0
	for i := 0; i+1 < len(class); i += 2 {
		total += int(class[i+1]-class[i]) + 1
	}
	if total <= 0 {
		return 'x'
	}
	pick := rng.IntN(total)
	for i := 0; i+1 < len(class); i += 2 {
		size := int(class[i+1]-class[i]) + 1
		if pick < size {
			return class[i] + rune(pick)
		}
		pick -= size
	}
	return class[0]
}
