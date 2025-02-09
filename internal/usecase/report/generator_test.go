package report

import (
	"errors"
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

	entries := []model.FileSystemEntry{
		{
			Path:    "/test/file1.txt",
			IsDir:   false,
			RelPath: "file1.txt",
			Content: []byte("test content 1"),
		},
		{
			Path:    "/test/file2.txt",
			IsDir:   false,
			RelPath: "file2.txt",
			ReadErr: errors.New("読み込みエラー"),
		},
		{
			Path:  "/test/dir",
			IsDir: true,
		},
	}

	generator.WriteFileContents(&buf, entries)

	output := buf.String()
	expectedLines := []string{
		"===== ファイル内容 =====",
		"----- file1.txt -----",
		"test content 1",
		"----- file2.txt -----",
		"[読み込みエラー]",
	}

	for _, line := range expectedLines {
		if !strings.Contains(output, line) {
			t.Errorf("出力に期待される行が含まれていない: %v", line)
		}
	}
}
