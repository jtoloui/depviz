package cli

import (
	"fmt"
	"sort"
	"strings"

	"github.com/jtoloui/depviz/internal/classify"
	"github.com/jtoloui/depviz/internal/scanner"
)

const barWidth = 20

// Stats prints a coloured terminal stats dashboard.
func Stats(results []scanner.FileImports, cl *classify.Classifier) {
	totalFiles := len(results)
	totalImports := 0
	totalExports := 0
	totalLines := 0
	langCount := map[string]int{}
	catCount := map[string]int{}
	impFreq := map[string]int{}

	for _, r := range results {
		totalImports += len(r.Imports)
		totalExports += len(r.Exports)
		totalLines += r.Lines
		lang := r.Lang
		if lang == "" {
			lang = "go"
		}
		langCount[lang]++
		for _, imp := range r.Imports {
			cat := string(cl.ClassifyWithLang(imp, r.Lang))
			catCount[cat]++
			impFreq[imp]++
		}
	}

	avg := 0
	if totalFiles > 0 {
		avg = totalImports / totalFiles
	}

	fmt.Printf("\n  %s%sdepviz stats%s\n\n", bold, magenta, reset)

	fmt.Printf("  %sFiles%s      %-12d %sLines%s    %d\n", cyan, reset, totalFiles, cyan, reset, totalLines)
	fmt.Printf("  %sImports%s    %-12d %sExports%s  %d\n", cyan, reset, totalImports, cyan, reset, totalExports)
	fmt.Printf("  %sAvg/file%s   %d\n\n", cyan, reset, avg)

	fmt.Printf("  %s%sLanguages%s\n", bold, cyan, reset)
	printBar(langCount, totalFiles, cyan)

	fmt.Printf("  %s%sCategories%s\n", bold, cyan, reset)
	printBarColoured(catCount, totalImports)

	fmt.Printf("  %s%sTop 5 Imports%s\n", bold, cyan, reset)
	for _, kv := range topN(impFreq, 5) {
		fmt.Printf("    %s%-45s%s %d files\n", dim, kv.key, reset, kv.val)
	}
	fmt.Println()

	var spots []kv
	for _, r := range results {
		if len(r.Imports) >= 8 {
			spots = append(spots, kv{r.File, len(r.Imports)})
		}
	}
	if len(spots) > 0 {
		sort.Slice(spots, func(i, j int) bool { return spots[i].val > spots[j].val })
		fmt.Printf("  %s%sCoupling Hotspots%s\n", bold, yellow, reset)
		for _, s := range spots {
			fmt.Printf("    %s%-45s%s %d imports\n", dim, s.key, reset, s.val)
		}
		fmt.Println()
	}
}

func printBar(counts map[string]int, total int, colour string) {
	for label, count := range counts {
		pct, filled := barCalc(count, total)
		bar := strings.Repeat("█", filled) + strings.Repeat("░", barWidth-filled)
		fmt.Printf("    %s%s%s %s  %d (%d%%)\n", colour, bar, reset, label, count, pct)
	}
	fmt.Println()
}

var catColours = map[string]string{
	"stdlib":   green,
	"internal": magenta,
	"private":  "\033[34m",
	"external": yellow,
}

func printBarColoured(counts map[string]int, total int) {
	for _, label := range []string{"stdlib", "internal", "private", "external"} {
		count, ok := counts[label]
		if !ok {
			continue
		}
		pct, filled := barCalc(count, total)
		colour := catColours[label]
		bar := strings.Repeat("█", filled) + strings.Repeat("░", barWidth-filled)
		fmt.Printf("    %s%s%s %-10s %d (%d%%)\n", colour, bar, reset, label, count, pct)
	}
	fmt.Println()
}

func barCalc(count, total int) (pct, filled int) {
	if total > 0 {
		pct = count * 100 / total
		filled = count * barWidth / total
	}
	return
}

type kv struct {
	key string
	val int
}

func topN(m map[string]int, n int) []kv {
	items := make([]kv, 0, len(m))
	for k, v := range m {
		items = append(items, kv{k, v})
	}
	sort.Slice(items, func(i, j int) bool { return items[i].val > items[j].val })
	if len(items) > n {
		items = items[:n]
	}
	return items
}
