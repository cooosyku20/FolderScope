// Package filesystem はファイルシステム操作を提供します
package filesystem

import (
	"context"
	"fmt"
	"io/fs"
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
	Scan(ctx context.Context, rootDir string) ([]model.FileSystemEntry, error)
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
// context.Context を受け取り、キャンセル可能にします
func (s *Scanner) Scan(ctx context.Context, rootDir string) ([]model.FileSystemEntry, error) {
	var entries []model.FileSystemEntry

	// filepath.WalkDir を使用 (Go 1.16+)
	err := filepath.WalkDir(rootDir, func(path string, d fs.DirEntry, err error) error {
		// コンテキストのキャンセルをチェック
		select {
		case <-ctx.Done():
			return ctx.Err() // コンテキストがキャンセルされたらエラーを返す
		default:
			// 処理を続行
		}

		// WalkDir自体が返すエラー（例：権限）をチェック
		if err != nil {
			s.logger.Log("WARN", fmt.Sprintf("パス '%s' のアクセス中にエラー発生", path), err)
			// ディレクトリ自体にアクセスできない場合など、スキップ可能なエラーはnilを返す
			if d == nil || !d.IsDir() {
				return nil // ファイルへのアクセスエラーはスキップ
			}
			return fmt.Errorf("ディレクトリ '%s' のアクセスエラー: %w", path, err) // ディレクトリへのアクセスエラーは致命的として返す
		}

		// ルートディレクトリ自体はスキップ
		if path == rootDir {
			return nil
		}

		relPath, err := filepath.Rel(rootDir, path)
		if err != nil {
			s.logger.Log("WARN", fmt.Sprintf("相対パスの取得に失敗: %s", path), err)
			return nil // 相対パス取得エラーはスキップ
		}

		// info, err := d.Info() // fs.DirEntryからFileInfoを取得 - Not needed anymore
		// if err != nil {
		// 	s.logger.Log("WARN", fmt.Sprintf("ファイル情報 '%s' の取得に失敗", path), err)
		// 	return nil // FileInfo取得エラーはスキップ
		// }

		entry := model.FileSystemEntry{
			Path:    path,
			IsDir:   d.IsDir(), // DirEntryから IsDir を取得
			RelPath: relPath,
			Depth:   strings.Count(relPath, string(os.PathSeparator)),
		}

		if !d.IsDir() {
			// コンテキストのキャンセルを再度チェック（ファイル読み込み前）
			select {
			case <-ctx.Done():
				return ctx.Err()
			default:
			}

			// // content, err := os.ReadFile(path) // Remove file reading
			// if err != nil {
			// 	s.logger.Log("ERROR", fmt.Sprintf("ファイル '%s' の読み込みに失敗", path), err)
			// 	return nil // ファイル読み込みエラーはスキップ（エラーを伝播させたい場合は err を返す）
			// }

			// // if s.isBinaryFile(content) { // Remove binary check
			// // 	s.logger.Log("WARN", fmt.Sprintf("バイナリファイルのためスキップ: %s", path), nil)
			// // 	return nil
			// // }

			// entry.Content = content // Remove content assignment
		}

		entries = append(entries, entry)
		return nil // このパスの処理は成功
	})

	if err != nil {
		// WalkDir自体から返されたエラー、またはコールバック内で返されたエラー
		return nil, fmt.Errorf("ファイルシステムの走査中にエラーが発生しました: %w", err)
	}

	return entries, nil
}
