// Package debugutil contains utility functions, mostly helpful for writting tests,
// related to code generation.
package debugutil

import (
	"fmt"
	"strconv"
	"strings"
)

// WithLineNumbers returns a version of the string with line numbers printed on
// the left hand side of each line.
func WithLineNumbers(str string) string {
	lines := strings.Split(str, "\n")

	widthNeeded := len(strconv.Itoa(len(lines)))
	format := "%" + strconv.Itoa(widthNeeded) + "d: %s"
	for i, line := range lines {
		lines[i] = fmt.Sprintf(format, i+1, line)
	}
	return strings.Join(lines, "\n")
}

// SideBySide returns a string with a and b printed side by side, separated by
// pipe character (for equal lines) or a delta character (for different lines).
func SideBySide(a, b string) string {
	linesA := strings.Split(replaceTabs(a), "\n")
	linesB := strings.Split(replaceTabs(b), "\n")
	lhsWidth := maxWidth(linesA)

	lineOrBlank := func(lines []string, i int) (string, bool) {
		if i < len(lines) {
			return lines[i], true
		}
		return "", false
	}

	var outLines []string
	for i := 0; ; i++ {
		if i >= len(linesA) && i >= len(linesB) {
			break
		}

		lineA, existsA := lineOrBlank(linesA, i)
		lineB, existsB := lineOrBlank(linesB, i)
		sep := "| "
		if lineA != lineB || existsA != existsB {
			sep = "Δ"
		}
		outLines = append(outLines, fmt.Sprintf("%s%s %s", pad(lineA, lhsWidth), sep, lineB))
	}
	return WithLineNumbers(strings.Join(outLines, "\n"))
}

func replaceTabs(str string) string {
	return strings.ReplaceAll(str, "\t", "  ")
}

func maxWidth(lines []string) int {
	max := 0
	for _, line := range lines {
		l := len(line)
		if l > max {
			max = l
		}
	}
	return max
}

func pad(line string, width int) string {
	out := line
	padSize := width - len(line)
	for i := 0; i < padSize; i++ {
		out += " "
	}
	return out
}
