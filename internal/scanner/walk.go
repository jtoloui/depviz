package scanner

import (
	"os"
	"path/filepath"
	"runtime"
	"sync"
)

// parseFunc extracts imports from a single file.
// Returns nil to skip the file (e.g. no imports found).
type parseFunc func(root, path string) (*FileImports, error)

// walkAndParse walks root, filters with shouldInclude, and fans out
// parsing to numWorkers goroutines.
func walkAndParse(root string, skip map[string]bool, shouldInclude func(string, os.FileInfo) bool, parse parseFunc) ([]FileImports, error) {
	numWorkers := runtime.NumCPU()

	paths := make(chan string)
	type result struct {
		fi  *FileImports
		err error
	}
	results := make(chan result)

	// Walk sends paths to the channel; walk error sent through results.
	var wg sync.WaitGroup
	wg.Add(numWorkers + 1) // +1 for walker
	go func() {
		defer wg.Done()
		defer close(paths)
		err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if info.IsDir() && skip[info.Name()] {
				return filepath.SkipDir
			}
			if shouldInclude(path, info) {
				paths <- path
			}
			return nil
		})
		if err != nil {
			results <- result{err: err}
		}
	}()

	// N workers parse files.
	for range numWorkers {
		go func() {
			defer wg.Done()
			for path := range paths {
				fi, err := parse(root, path)
				results <- result{fi, err}
			}
		}()
	}

	// Close results when all workers finish.
	go func() {
		wg.Wait()
		close(results)
	}()

	// Collect results.
	var files []FileImports
	for r := range results {
		if r.err != nil {
			return nil, r.err
		}
		if r.fi != nil {
			files = append(files, *r.fi)
		}
	}

	return files, nil
}
