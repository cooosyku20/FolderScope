package report

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"FolderScope/internal/domain/model"
)

// テスト用のファイルライクな構造体
type testFile struct {
	*os.File
}

func (f *testFile) Write(p []byte) (n int, err error) {
	return f.File.Write(p)
}

func createTestFile(t *testing.T) (*testFile, func()) {
	tempFile, err := os.CreateTemp("", "test_*.txt")
	if err != nil {
		t.Fatalf("一時ファイルの作成に失敗: %v", err)
	}

	return &testFile{File: tempFile}, func() {
		tempFile.Close()
		os.Remove(tempFile.Name())
	}
}

func TestGenerator_CreateOutputFile(t *testing.T) {
	generator := NewGenerator()

	// テスト用の一時ディレクトリを作成
	tempDir, err := os.MkdirTemp("", "generator_test")
	if err != nil {
		t.Fatalf("一時ディレクトリの作成に失敗: %v", err)
	}
	defer os.RemoveAll(tempDir)

	file, path, err := generator.CreateOutputFile(tempDir)
	if err != nil {
		t.Fatalf("CreateOutputFile() error = %v", err)
	}
	defer file.Close()

	if !strings.HasPrefix(filepath.Base(path), "output_") {
		t.Errorf("出力ファイル名が不正: got %v", filepath.Base(path))
	}

	if !strings.HasSuffix(path, ".txt") {
		t.Errorf("出力ファイルの拡張子が不正: got %v", filepath.Base(path))
	}
}

func TestGenerator_WriteFileSystemStructure(t *testing.T) {
	generator := NewGenerator()
	var buf strings.Builder

	entries := []model.FileSystemEntry{
		{
			Path:    "/test/dir",
			IsDir:   true,
			RelPath: "dir",
			Depth:   1,
		},
		{
			Path:    "/test/dir/file.txt",
			IsDir:   false,
			RelPath: "dir/file.txt",
			Depth:   2,
		},
	}

	generator.WriteFileSystemStructure(&buf, entries)

	output := buf.String()
	expectedLines := []string{
		"===== フォルダ・ファイル構成 =====",
		"  [DIR]  dir",
		"    [FILE] dir/file.txt",
	}

	for _, line := range expectedLines {
		if !strings.Contains(output, line) {
			t.Errorf("出力に期待される行が含まれていない: %v", line)
		}
	}
}

func TestGenerator_WriteFileContents(t *testing.T) {
	generator := NewGenerator()
	var buf strings.Builder

	// --- Test Setup: Create temporary files ---
	tempDir, err := os.MkdirTemp("", "generator_content_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// 1. Normal text file
	file1Path := filepath.Join(tempDir, "file1.txt")
	file1Content := "test content 1"
	if err := os.WriteFile(file1Path, []byte(file1Content), 0644); err != nil {
		t.Fatalf("Failed to write file1: %v", err)
	}

	// 2. Binary file
	file2Path := filepath.Join(tempDir, "file2.bin")
	file2Content := []byte{0x00, 0x01, 0x02} // Binary content
	if err := os.WriteFile(file2Path, file2Content, 0644); err != nil {
		t.Fatalf("Failed to write file2: %v", err)
	}

	// 3. File that will be made unreadable
	file3Path := filepath.Join(tempDir, "file3.txt")
	if err := os.WriteFile(file3Path, []byte("unreadable"), 0000); err != nil { // Write with 0000 permissions
		t.Fatalf("Failed to write file3: %v", err)
	}
	// --- End Test Setup ---

	entries := []model.FileSystemEntry{
		{
			Path:    file1Path, // Use actual path
			IsDir:   false,
			RelPath: "file1.txt",
			// Content: []byte("test content 1"), // Removed
		},
		{
			Path:    file2Path, // Use actual path
			IsDir:   false,
			RelPath: "file2.bin",
			// ReadErr: errors.New("読み込みエラー"), // Removed
		},
		{
			Path:    file3Path, // Use actual path for unreadable file
			IsDir:   false,
			RelPath: "file3.txt",
		},
		{
			Path:    filepath.Join(tempDir, "dir"),
			IsDir:   true,
			RelPath: "dir",
		},
	}

	generator.WriteFileContents(&buf, entries)

	output := buf.String()
	expectedSubstrings := []string{ // Use substrings as error messages might vary slightly
		"===== ファイル内容 =====",
		"----- file1.txt -----",
		file1Content, // Check for actual content
		"------------------------",
		"----- file2.bin -----",
		"[バイナリファイルのためスキップ]", // Check for binary skip message
		"------------------------",
		"----- file3.txt -----",
		"[読み込みエラー]",         // Check for read error indicator
		"permission denied", // Check for part of the expected permission error
		"------------------------",
	}

	for _, sub := range expectedSubstrings {
		if !strings.Contains(output, sub) {
			t.Errorf("出力に期待される部分文字列が含まれていない: %q\nOutput:\n%s", sub, output)
		}
	}

	// Check that the directory was skipped (no ----- dir -----)
	if strings.Contains(output, "----- dir -----") {
		t.Errorf("Directory entry was processed in WriteFileContents")
	}
}
