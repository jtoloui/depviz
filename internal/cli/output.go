package cli

import (
	"fmt"

	"github.com/common-nighthawk/go-figure"
	"github.com/jtoloui/depviz/internal/scanner"
)

const (
	reset   = "\033[0m"
	bold    = "\033[1m"
	dim     = "\033[2m"
	green   = "\033[32m"
	cyan    = "\033[36m"
	magenta = "\033[35m"
	yellow  = "\033[33m"
)

// Banner prints the ASCII art depviz logo.
func Banner() {
	fig := figure.NewColorFigure("depviz", "small", "purple", true)
	fmt.Println()
	fig.Print()
	fmt.Println()
}

// ScanResult prints a coloured summary after scanning.
func ScanResult(results []scanner.FileImports, output string) {
	totalImports := 0
	totalExports := 0
	totalLines := 0
	for _, r := range results {
		totalImports += len(r.Imports)
		totalExports += len(r.Exports)
		totalLines += r.Lines
	}

	fmt.Printf("  %s%s✓ Scan complete%s\n", bold, green, reset)
	fmt.Printf("  %s%sFiles%s    %d\n", dim, cyan, reset, len(results))
	fmt.Printf("  %s%sImports%s  %d\n", dim, magenta, reset, totalImports)
	fmt.Printf("  %s%sExports%s  %d\n", dim, yellow, reset, totalExports)
	fmt.Printf("  %s%sLines%s    %d\n", dim, cyan, reset, totalLines)
	fmt.Printf("\n  %s→%s %s\n\n", green, reset, output)
}

// ServeResult prints a coloured summary when the server starts.
func ServeResult(results []scanner.FileImports, port int) {
	fmt.Printf("  %s%s✓ Scan complete%s — %d files\n", bold, green, reset, len(results))
	fmt.Printf("\n  %s%s→%s http://localhost:%d\n\n", bold, green, reset, port)
}
