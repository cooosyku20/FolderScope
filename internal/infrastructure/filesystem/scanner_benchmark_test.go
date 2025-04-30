package filesystem

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"testing"

	"FolderScope/internal/infrastructure/logging"
)

// setupBenchmarkDir creates a temporary directory structure for benchmarking.
func setupBenchmarkDir(tb testing.TB, depth, filesPerDir, dirsPerDir int) string {
	tb.Helper()
	tempDir, err := os.MkdirTemp("", "benchmark_scan_*")
	if err != nil {
		tb.Fatalf("Failed to create temp dir: %v", err)
	}

	createDirContents(tb, tempDir, depth, filesPerDir, dirsPerDir)

	return tempDir
}

func createDirContents(tb testing.TB, currentPath string, depth, filesPerDir, dirsPerDir int) {
	tb.Helper()
	if depth <= 0 {
		return
	}

	// Create files
	for i := 0; i < filesPerDir; i++ {
		fileName := filepath.Join(currentPath, fmt.Sprintf("file_%d_%d.txt", depth, i))
		content := []byte(fmt.Sprintf("Content for file %d at depth %d", i, depth))
		if err := os.WriteFile(fileName, content, 0644); err != nil {
			// Cleanup already created files/dirs in case of error
			os.RemoveAll(filepath.Dir(fileName)) // Attempt cleanup, might fail
			tb.Fatalf("Failed to write file %s: %v", fileName, err)
		}
	}

	// Create subdirectories
	for i := 0; i < dirsPerDir; i++ {
		subDir := filepath.Join(currentPath, fmt.Sprintf("subdir_%d_%d", depth, i))
		if err := os.Mkdir(subDir, 0755); err != nil {
			// Cleanup already created files/dirs in case of error
			os.RemoveAll(filepath.Dir(subDir)) // Attempt cleanup, might fail
			tb.Fatalf("Failed to create subdir %s: %v", subDir, err)
		}
		createDirContents(tb, subDir, depth-1, filesPerDir, dirsPerDir)
	}
}

// BenchmarkScanner_Scan benchmarks the Scan function.
func BenchmarkScanner_Scan(b *testing.B) {
	// Use a logger that discards output to avoid interfering with benchmark timing.
	// Alternatively, use the mockLogger if log verification is needed (less ideal for pure perf).
	logger := logging.NewJSONLogger(io.Discard) // Discard logs during benchmark
	scanner := NewScanner(logger)

	// Setup: Create a moderately complex directory structure
	// Adjust depth, filesPerDir, dirsPerDir for different scenarios
	depth := 3
	filesPerDir := 5
	dirsPerDir := 2
	tempDir := setupBenchmarkDir(b, depth, filesPerDir, dirsPerDir)

	// Cleanup the temporary directory after the benchmark finishes.
	b.Cleanup(func() {
		os.RemoveAll(tempDir)
	})

	// Reset timer to exclude setup time.
	b.ResetTimer()
	// Report memory allocations.
	b.ReportAllocs()

	// Run the function b.N times.
	for i := 0; i < b.N; i++ {
		_, err := scanner.Scan(context.Background(), tempDir)
		if err != nil {
			b.Fatalf("Scan failed during benchmark: %v", err)
		}
	}
}
