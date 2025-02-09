// Package ui はユーザーインターフェース機能を提供します
package ui

import (
	"fmt"
	"github.com/sqweek/dialog"
	"FolderScope/internal/infrastructure/filesystem"
)

// DirectorySelector はディレクトリ選択機能を提供します
type DirectorySelector struct {
	// validator はディレクトリパスの検証を行うインターフェースです
	validator filesystem.DirectoryValidator
}

// NewDirectorySelector は新しい DirectorySelector インスタンスを作成します
func NewDirectorySelector(validator filesystem.DirectoryValidator) *DirectorySelector {
	return &DirectorySelector{validator: validator}
}

// SelectDirectory はダイアログを表示してディレクトリを選択します
func (d *DirectorySelector) SelectDirectory(title string) (string, error) {
	selectedDir, err := dialog.Directory().Title(title).Browse()
	if err != nil {
		return "", fmt.Errorf("ディレクトリの選択がキャンセルまたはエラーになりました: %w", err)
	}

	if err := d.validator.ValidateDirectoryPath(selectedDir); err != nil {
		return "", fmt.Errorf("無効なディレクトリが選択されました: %w", err)
	}

	return selectedDir, nil
}
