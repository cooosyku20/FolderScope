// Package filesystem はファイルシステム操作を提供します
package filesystem

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"FolderScope/internal/domain/model"
	"FolderScope/internal/infrastructure/logging"
)

const DefaultBinaryCheckSize = 1024

// DirectoryValidator はディレクトリの検証機能を提供するインターフェースです
type DirectoryValidator interface {
	ValidateDirectoryPath(path string) error
}

// FileSystemScanner はファイルシステムのスキャン機能を提供するインターフェースです
type FileSystemScanner interface {
	DirectoryValidator
	Scan(rootDir string) ([]model.FileSystemEntry, error)
}

// Scanner はファイルシステムをスキャンするための構造体です
type Scanner struct {
	logger          logging.Logger
	binaryCheckSize int
}

// NewScanner は新しい Scanner インスタンスを作成します
func NewScanner(logger logging.Logger) *Scanner {
	return &Scanner{
		logger:          logger,
		binaryCheckSize: DefaultBinaryCheckSize,
	}
}

// ValidateDirectoryPath はパスが安全で有効なディレクトリであることを確認します
func (s *Scanner) ValidateDirectoryPath(path string) error {
	if path == "" {
		return fmt.Errorf("ディレクトリパスが指定されていません")
	}

	fileInfo, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("ディレクトリが存在しません: %w", err)
	}

	if !fileInfo.IsDir() {
		return fmt.Errorf("指定されたパスはディレクトリではありません")
	}

	if !filepath.IsAbs(path) {
		return fmt.Errorf("絶対パスで指定してください")
	}

	if strings.ContainsAny(path, "<>|?*") {
		return fmt.Errorf("パスに不正な文字が含まれています")
	}

	return nil
}

// isBinaryFile は与えられたバイトデータがバイナリファイルかどうかを判定します
func (s *Scanner) isBinaryFile(content []byte) bool {
	checkSize := s.binaryCheckSize
	if len(content) < checkSize {
		checkSize = len(content)
	}

	// NULL(0x00)や制御不能文字を検出
	for i := 0; i < checkSize; i++ {
		if content[i] == 0x00 || (content[i] < 0x09 && content[i] != 0x0A && content[i] != 0x0D) {
			return true
		}
	}
	return false
}

// Scan はファイルシステムを走査し、エントリを収集します
func (s *Scanner) Scan(rootDir string) ([]model.FileSystemEntry, error) {
	var entries []model.FileSystemEntry

	err := filepath.Walk(rootDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			s.logger.Log("WARN", fmt.Sprintf("パス '%s' の走査中にエラー発生", path), err)
			return nil
		}

		relPath, err := filepath.Rel(rootDir, path)
		if err != nil {
			s.logger.Log("WARN", fmt.Sprintf("相対パスの取得に失敗: %s", path), err)
			return nil
		}

		if relPath == "." {
			return nil
		}

		entry := model.FileSystemEntry{
			Path:    path,
			IsDir:   info.IsDir(),
			RelPath: relPath,
			Depth:   strings.Count(relPath, string(os.PathSeparator)),
		}

		if !info.IsDir() {
			content, err := os.ReadFile(path)
			if err != nil {
				s.logger.Log("ERROR", fmt.Sprintf("ファイル '%s' の読み込みに失敗", path), err)
				return nil
			}

			if s.isBinaryFile(content) {
				s.logger.Log("WARN", fmt.Sprintf("バイナリファイルのためスキップ: %s", path), nil)
				return nil
			}

			entry.Content = content
		}

		entries = append(entries, entry)
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("ファイルシステムの走査に失敗しました: %w", err)
	}

	return entries, nil
}
