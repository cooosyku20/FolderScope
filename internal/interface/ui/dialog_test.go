package ui

import (
	"errors"
	"testing"

	"FolderScope/internal/domain/model"
	"FolderScope/internal/infrastructure/filesystem"
)

type mockScanner struct {
	filesystem.FileSystemScanner
	validateError error
}

func (m *mockScanner) ValidateDirectoryPath(path string) error {
	return m.validateError
}

func (m *mockScanner) Scan(rootDir string) ([]model.FileSystemEntry, error) {
	return nil, nil // テストでは使用しないため、空の実装
}

func TestDirectorySelector_SelectDirectory(t *testing.T) {
	// dialog.Directory()はモック化が難しいため、
	// ここではValidateDirectoryPathの動作のみをテストします

	tests := []struct {
		name          string
		validateError error
		wantErr      bool
	}{
		{
			name:          "バリデーション成功",
			validateError: nil,
			wantErr:      false,
		},
		{
			name:          "バリデーションエラー",
			validateError: errors.New("無効なディレクトリ"),
			wantErr:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockScanner := &mockScanner{validateError: tt.validateError}
			selector := NewDirectorySelector(mockScanner)

			// dialog.Directory()の呼び出しはスキップされるため、
			// このテストは実際のダイアログ表示なしで実行されます
			if tt.validateError != nil {
				if _, err := selector.SelectDirectory("テスト"); err == nil {
					t.Error("SelectDirectory() error = nil, wantErr true")
				}
			}
		})
	}
}
