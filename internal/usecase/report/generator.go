// Package report はレポート生成機能を提供します
package report

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"FolderScope/internal/domain/model"
)

const (
	OutputFilePrefix = "output_"
	OutputFileSuffix = ".txt"
	TimestampLayout  = "20060102_150405"
)

// Generator はレポート生成機能を提供します
type Generator struct{}

// NewGenerator は新しい Generator インスタンスを作成します
func NewGenerator() *Generator { // [cite: 270]
	return &Generator{} // [cite: 270]
}

// CreateOutputFile は出力ファイルを作成します
func (g *Generator) CreateOutputFile(outputDir string) (*os.File, string, error) {
	timestamp := time.Now().Format(TimestampLayout)
	outputPath := filepath.Join(outputDir, fmt.Sprintf("%s%s%s", OutputFilePrefix, timestamp, OutputFileSuffix))

	outputFile, err := os.Create(outputPath)
	if err != nil {
		return nil, "", fmt.Errorf("出力ファイルの作成に失敗しました: %w", err)
	}

	return outputFile, outputPath, nil
}

// WriteFileSystemStructure はエントリの深さに応じたインデントを付与し,
// フォルダ（[DIR]）とファイル（[FILE]）を一覧で出力します。
// バイナリファイルは出力から除外されます。
func (g *Generator) WriteFileSystemStructure(writer io.Writer, entries []model.FileSystemEntry) {
	fmt.Fprintln(writer, "===== フォルダ・ファイル構成 =====")

	for _, entry := range entries {
		// バイナリファイルであり、かつディレクトリでない場合はスキップ
		if !entry.IsDir && entry.IsBinary {
			continue
		}

		indent := strings.Repeat("  ", entry.Depth)
		entryType := "[FILE]"
		if entry.IsDir {
			entryType = "[DIR] "
		}
		fmt.Fprintf(writer, "%s%s %s\n", indent, entryType, entry.RelPath)
	}
}

// WriteFileContents はファイルの内容を読み込んで出力します
// バイナリファイルの場合は内容をスキップし、その旨を記述します。
func (g *Generator) WriteFileContents(writer io.Writer, entries []model.FileSystemEntry) {
	fmt.Fprintln(writer, "\n===== ファイル内容 =====")

	for _, entry := range entries {
		if entry.IsDir {
			continue
		}

		fmt.Fprintf(writer, "----- %s -----\n", entry.RelPath)

		if entry.IsBinary {
			fmt.Fprintln(writer, "[バイナリファイルのためスキップ]")
		} else if entry.ReadErr != nil {
			// Scannerでのバイナリ判定時の読み込みエラーを考慮
			fmt.Fprintf(writer, "[ファイル読み込みエラー（スキャン時）のため内容表示不可] %v\n", entry.ReadErr)
		} else {
			// テキストファイルと判定された（かつスキャン時にエラーがなかった）場合のみ内容を読み込む
			content, err := os.ReadFile(entry.Path)
			if err != nil {
				fmt.Fprintf(writer, "[ファイル読み込みエラー（レポート生成時）] %v\n", err)
			} else {
				// 念のため、ここで再度バイナリチェックを行うことも検討可能だが、
				// 基本的にはScannerの判定を信頼する。
				// もしScannerの判定が不完全で、大きなファイルの場合、
				// ここでの読み込みが問題になる可能性はある。
				fmt.Fprintln(writer, string(content))
			}
		}
		fmt.Fprintln(writer, "------------------------")
	}
}
