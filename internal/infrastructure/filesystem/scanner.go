// Package filesystem はファイルシステム操作を提供します
package filesystem

import (
	"context"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"FolderScope/internal/domain/model"
	"FolderScope/internal/infrastructure/logging"
)

const DefaultBinaryCheckSize = 1024

// DefaultIgnorePatterns はデフォルトで無視するパターンです。
var DefaultIgnorePatterns = []string{".git", ".DS_Store", ".idea", ".vscode"} // デフォルト無視パターンを追加

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
	logger            logging.Logger
	binaryCheckSize   int
	ignorePatterns    []string // 追加
	ignoreBinaryFiles bool     // 追加
}

// NewScanner は新しい Scanner インスタンスを作成します
// 引数に ignorePatterns と ignoreBinaryFiles を追加
func NewScanner(logger logging.Logger, ignorePatterns []string, ignoreBinaryFiles bool) *Scanner {
	// デフォルトの無視パターンとユーザー指定の無視パターンをマージ
	allIgnorePatterns := append(DefaultIgnorePatterns, ignorePatterns...) // DefaultIgnorePatterns を先に
	// 重複を削除する場合 (オプション)
	// uniquePatterns := make(map[string]struct{})
	// for _, p := range allIgnorePatterns {
	//  uniquePatterns[p] = struct{}{}
	// }
	// finalPatterns := make([]string, 0, len(uniquePatterns))
	// for p := range uniquePatterns {
	//  finalPatterns = append(finalPatterns, p)
	// }

	return &Scanner{
		logger:            logger,
		binaryCheckSize:   DefaultBinaryCheckSize,
		ignorePatterns:    allIgnorePatterns, // マージしたパターンを使用
		ignoreBinaryFiles: ignoreBinaryFiles,
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

	// NULLバイトのみをチェックする
	if strings.ContainsRune(path, '\000') {
		return fmt.Errorf("パスにNULLバイトが含まれています")
	}

	return nil
}

// isBinaryFile は与えられたバイトデータがバイナリファイルかどうかを判定します
func (s *Scanner) isBinaryFile(content []byte) bool {
	limit := len(content)
	if limit == 0 { // 空のファイルはバイナリではない
		return false
	}
	if limit > s.binaryCheckSize {
		limit = s.binaryCheckSize
	}

	for i := 0; i < limit; i++ {
		if content[i] == 0x00 { // NULLバイトがあればバイナリとみなす
			return true
		}
		// 制御文字の判定をより厳密に (ただし、UTF-8テキスト内の特定の制御文字は許容される場合がある)
		// ここでは簡略化のため、NULLバイトのみを主な判定基準とする
		// if (content[i] < 0x09 && content[i] != 0x0A && content[i] != 0x0D) {
		//  return true
		// }
	}
	// textChars / totalChars の比率で判定する方法もあるが、ここでは単純な NULL バイトチェック
	return false
}

// matchesIgnorePattern は指定されたパスが無視パターンに一致するかどうかを確認します
func (s *Scanner) matchesIgnorePattern(path string, d fs.DirEntry) (bool, error) {
	name := d.Name() // ディレクトリ名またはファイル名で比較
	for _, pattern := range s.ignorePatterns {
		// パターンがディレクトリを示す場合 (例: "node_modules/") は、ディレクトリ名全体と比較
		if strings.HasSuffix(pattern, string(filepath.Separator)) {
			if d.IsDir() && strings.TrimSuffix(pattern, string(filepath.Separator)) == name {
				return true, nil
			}
		} else {
			// ファイル名またはディレクトリ名に対する glob パターンマッチ
			matched, err := filepath.Match(pattern, name)
			if err != nil {
				s.logger.Log("WARN", fmt.Sprintf("無視パターンの評価エラー: %s on %s", pattern, name), err)
				// パターンエラーは無視して処理を続行（またはエラーとして扱うか選択）
				continue
			}
			if matched {
				return true, nil
			}
		}
	}
	// フルパスに対するマッチも追加 (オプション)
	// for _, pattern := range s.ignorePatterns {
	//   if strings.HasPrefix(path, pattern) { // 例: "/abs/path/to/ignore_this_dir"
	//     return true, nil
	//   }
	// }
	return false, nil
}

// Scan はファイルシステムを走査し、エントリを収集します
// context.Context を受け取り、キャンセル可能にします
func (s *Scanner) Scan(ctx context.Context, rootDir string) ([]model.FileSystemEntry, error) {
	var entries []model.FileSystemEntry
	absRootDir, err := filepath.Abs(rootDir)
	if err != nil {
		return nil, fmt.Errorf("ルートディレクトリの絶対パス取得に失敗: %w", err)
	}

	// Scan開始前にルートディレクトリの存在と種類を確認
	info, err := os.Stat(absRootDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("指定されたルートディレクトリが存在しません: %s", absRootDir)
		}
		return nil, fmt.Errorf("ルートディレクトリ情報の取得に失敗: %w", err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("指定されたルートパスはディレクトリではありません: %s", absRootDir)
	}

	err = filepath.WalkDir(absRootDir, func(path string, d fs.DirEntry, walkErr error) error {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		if walkErr != nil {
			// WalkDir からのエラー（権限など）
			// 特定のエラー（例: os.ErrPermission）をより詳細にハンドリングすることも可能
			s.logger.Log("WARN", fmt.Sprintf("パス '%s' のアクセス中にエラー発生 (WalkDir)", path), walkErr)
			if d != nil && d.IsDir() {
				return fs.SkipDir // ディレクトリへのアクセスエラーの場合、そのディレクトリはスキップ
			}
			return nil // ファイルへのアクセスエラーはスキップして処理を続行
		}

		// ルートディレクトリ自体は結果に含めない
		if path == absRootDir {
			return nil
		}

		// 無視パターンのチェック
		// WalkDir はディレクトリを先に処理するため、ここでディレクトリを無視すればその中身もスキップされる
		isIgnored, patternErr := s.matchesIgnorePattern(path, d)
		if patternErr != nil {
			// パターン評価エラーのログは matchesIgnorePattern 内で記録済み
			// ここではエラーを返さずに処理を続けるか、エラーを返すか選択
		}
		if isIgnored {
			s.logger.Log("DEBUG", fmt.Sprintf("パス '%s' は無視パターンに一致しました。", path), nil)
			if d.IsDir() {
				return fs.SkipDir // ディレクトリの場合は中身もスキップ
			}
			return nil // ファイルの場合はこのファイルのみスキップ
		}

		relPath, err := filepath.Rel(absRootDir, path)
		if err != nil {
			s.logger.Log("WARN", fmt.Sprintf("相対パスの取得に失敗: %s", path), err)
			return nil
		}
		relPath = filepath.ToSlash(relPath) // パス区切りを '/' に統一

		depth := strings.Count(relPath, "/")
		// ルート直下は Depth 0 だが、一般的には1から数えるため調整 (オプション)
		// if relPath != "" { depth++ }

		entry := model.FileSystemEntry{
			Path:    path,
			IsDir:   d.IsDir(),
			RelPath: relPath,
			Depth:   depth, // ルートからの階層 (ルート直下を0とするか1とするかは要件次第)
			// Size と ModTime は fs.DirEntry から取得可能 (d.Info())
		}

		if !d.IsDir() {
			// ファイルの場合、バイナリ判定とスキップ処理
			var fileContent []byte

			// os.ReadFile は Go 1.16+
			// fileContent, readErrForBinaryCheck = os.ReadFile(path)

			// より制御しやすくするために os.Open, Read, Close を使う
			file, openErr := os.Open(path)
			if openErr != nil {
				s.logger.Log("WARN", fmt.Sprintf("ファイル '%s' のオープンに失敗", path), openErr)
				entry.ReadErr = openErr
				// オープン失敗時はバイナリ判定不可、エラーとしてマーク
				// IsBinary はデフォルトで false のまま
			} else {
				defer file.Close() // walkDir の各イテレーションで呼ばれるため、確実にクローズする
				buffer := make([]byte, s.binaryCheckSize)
				n, readErr := file.Read(buffer)
				if readErr != nil && readErr != io.EOF {
					s.logger.Log("WARN", fmt.Sprintf("ファイル '%s' の読み込みに失敗（バイナリ判定用）", path), readErr)
					entry.ReadErr = readErr
				}
				fileContent = buffer[:n] // 実際に読み込めた部分だけを渡す

				// file.Close() は defer で実行される
			}

			if entry.ReadErr == nil { // ファイルが正常に（一部でも）読み込めた場合のみバイナリ判定
				entry.IsBinary = s.isBinaryFile(fileContent)
			}

			if s.ignoreBinaryFiles && entry.IsBinary {
				s.logger.Log("DEBUG", fmt.Sprintf("バイナリファイル '%s' は無視されます。", path), nil)
				return nil // バイナリファイルを無視する設定の場合、スキップ
			}
		}

		entries = append(entries, entry)
		return nil
	})

	if err != nil && err != fs.SkipDir { // SkipDir はエラーとして扱わない
		// WalkDir自体から返されたエラー、またはコールバック内で返されたエラー
		// ctx.Err() の場合もここに到達する
		if err == context.Canceled || err == context.DeadlineExceeded {
			s.logger.Log("INFO", "スキャン処理がキャンセルまたはタイムアウトしました。", err)
			return nil, err
		}
		return nil, fmt.Errorf("ファイルシステムの走査中にエラーが発生しました: %w", err)
	}

	return entries, nil
}
