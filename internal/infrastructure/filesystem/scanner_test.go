package filesystem

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type mockLogger struct {
	logs []struct {
		level   string
		message string
		err     error
	}
}

func (m *mockLogger) Log(level, message string, err error) {
	m.logs = append(m.logs, struct {
		level   string
		message string
		err     error
	}{level, message, err})
}

func TestScanner_ValidateDirectoryPath(t *testing.T) {
	logger := &mockLogger{}
	scanner := NewScanner(logger)

	// テスト用の一時ディレクトリを作成
	tempDir, err := os.MkdirTemp("", "scanner_test")
	if err != nil {
		t.Fatalf("一時ディレクトリの作成に失敗: %v", err)
	}
	defer os.RemoveAll(tempDir)

	tests := []struct {
		name    string
		path    string
		wantErr bool
	}{
		{
			name:    "有効なディレクトリパス",
			path:    tempDir,
			wantErr: false,
		},
		{
			name:    "空のパス",
			path:    "",
			wantErr: true,
		},
		{
			name:    "存在しないパス",
			path:    filepath.Join(tempDir, "notexist"),
			wantErr: true,
		},
		{
			name:    "不正な文字を含むパス",
			path:    filepath.Join(tempDir, "test<>|?*"),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := scanner.ValidateDirectoryPath(tt.path)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateDirectoryPath() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestScanner_Scan(t *testing.T) {
	logger := &mockLogger{}
	scanner := NewScanner(logger)

	// テスト用の一時ディレクトリを作成
	tempDir, err := os.MkdirTemp("", "scanner_test")
	if err != nil {
		t.Fatalf("一時ディレクトリの作成に失敗: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// テスト用のファイルとディレクトリを作成
	testDir := filepath.Join(tempDir, "testdir")
	if err := os.Mkdir(testDir, 0755); err != nil {
		t.Fatalf("テストディレクトリの作成に失敗: %v", err)
	}

	testFile := filepath.Join(tempDir, "testfile.txt")
	if err := os.WriteFile(testFile, []byte("test content"), 0644); err != nil {
		t.Fatalf("テストファイルの作成に失敗: %v", err)
	}

	// バイナリファイルの作成
	binaryFile := filepath.Join(tempDir, "binary.bin")
	binaryContent := []byte{0x00, 0x01, 0x02, 0x03}
	if err := os.WriteFile(binaryFile, binaryContent, 0644); err != nil {
		t.Fatalf("バイナリファイルの作成に失敗: %v", err)
	}

	entries, err := scanner.Scan(tempDir)
	if err != nil {
		t.Fatalf("Scan() error = %v", err)
	}

	// エントリの数を確認（バイナリファイルはスキップされるため2つ）
	expectedEntries := 2 // testdir と testfile.txt
	if len(entries) != expectedEntries {
		t.Errorf("Scan() got %v entries, want %v", len(entries), expectedEntries)
	}

	// ファイルとディレクトリの存在を確認
	var foundDir, foundFile, foundBinary bool
	for _, entry := range entries {
		switch filepath.Base(entry.Path) {
		case "testdir":
			foundDir = true
		case "testfile.txt":
			foundFile = true
			if string(entry.Content) != "test content" {
				t.Errorf("File content = %v, want %v", string(entry.Content), "test content")
			}
		case "binary.bin":
			foundBinary = true
		}
	}

	if !foundDir {
		t.Error("Directory not found in scan results")
	}
	if !foundFile {
		t.Error("File not found in scan results")
	}
	if foundBinary {
		t.Error("Binary file should be skipped but was found in scan results")
	}

	// ログメッセージの確認
	var foundBinaryLog bool
	for _, log := range logger.logs {
		if log.level == "WARN" && strings.Contains(log.message, "バイナリファイルのためスキップ") {
			foundBinaryLog = true
			break
		}
	}

	if !foundBinaryLog {
		t.Error("バイナリファイルスキップのログが出力されていません")
	}
}

func TestScanner_isBinaryFile(t *testing.T) {
	logger := &mockLogger{}
	scanner := NewScanner(logger)

	tests := []struct {
		name     string
		content  []byte
		expected bool
	}{
		{
			name:     "テキストファイル",
			content:  []byte("This is a text file\nwith multiple lines\n"),
			expected: false,
		},
		{
			name:     "NULLを含むバイナリファイル",
			content:  []byte{0x48, 0x65, 0x6c, 0x6c, 0x6f, 0x00, 0x57, 0x6f, 0x72, 0x6c, 0x64},
			expected: true,
		},
		{
			name:     "制御文字を含むバイナリファイル",
			content:  []byte{0x48, 0x65, 0x6c, 0x6c, 0x6f, 0x03, 0x57, 0x6f, 0x72, 0x6c, 0x64},
			expected: true,
		},
		{
			name:     "空のファイル",
			content:  []byte{},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := scanner.isBinaryFile(tt.content)
			if result != tt.expected {
				t.Errorf("isBinaryFile() = %v, want %v", result, tt.expected)
			}
		})
	}
}
