package model

import (
	"errors"
	"testing"
)

func TestFileSystemEntry(t *testing.T) {
	tests := []struct {
		name     string
		entry    FileSystemEntry
		wantPath string
		wantDir  bool
		wantErr  error
	}{
		{
			name: "ディレクトリエントリ",
			entry: FileSystemEntry{
				Path:    "/test/dir",
				IsDir:   true,
				RelPath: "dir",
				Depth:   1,
			},
			wantPath: "/test/dir",
			wantDir:  true,
		},
		{
			name: "ファイルエントリ",
			entry: FileSystemEntry{
				Path:    "/test/file.txt",
				IsDir:   false,
				Content: []byte("test content"),
				RelPath: "file.txt",
				Depth:   1,
			},
			wantPath: "/test/file.txt",
			wantDir:  false,
		},
		{
			name: "エラーを含むエントリ",
			entry: FileSystemEntry{
				Path:    "/test/error.txt",
				IsDir:   false,
				RelPath: "error.txt",
				Depth:   1,
				ReadErr: errors.New("読み込みエラー"),
			},
			wantPath: "/test/error.txt",
			wantDir:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.entry.Path != tt.wantPath {
				t.Errorf("Path = %v, want %v", tt.entry.Path, tt.wantPath)
			}
			if tt.entry.IsDir != tt.wantDir {
				t.Errorf("IsDir = %v, want %v", tt.entry.IsDir, tt.wantDir)
			}
		})
	}
}
