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
func NewGenerator() *Generator {
	return &Generator{}
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
func (g *Generator) WriteFileSystemStructure(writer io.Writer, entries []model.FileSystemEntry) {
	fmt.Fprintln(writer, "===== フォルダ・ファイル構成 =====")

	for _, entry := range entries {
		indent := strings.Repeat("  ", entry.Depth)
		entryType := "[FILE]"
		if entry.IsDir {
			entryType = "[DIR] "
		}
		fmt.Fprintf(writer, "%s%s %s\n", indent, entryType, entry.RelPath)
	}
}

// isBinaryFile は与えられたバイトデータがバイナリファイルかどうかを判定します
// (Scannerから移動)
func isBinaryFile(content []byte) bool {
	const checkSize = 1024 // DefaultBinaryCheckSize に合わせる
	limit := len(content)
	if limit > checkSize {
		limit = checkSize
	}

	for i := 0; i < limit; i++ {
		if content[i] == 0x00 || (content[i] < 0x09 && content[i] != 0x0A && content[i] != 0x0D) {
			return true
		}
	}
	return false
}

// WriteFileContents はファイルの内容を読み込んで出力します
func (g *Generator) WriteFileContents(writer io.Writer, entries []model.FileSystemEntry) {
	fmt.Fprintln(writer, "\n===== ファイル内容 =====")

	for _, entry := range entries {
		if entry.IsDir {
			continue
		}

		fmt.Fprintf(writer, "----- %s -----\n", entry.RelPath)

		content, err := os.ReadFile(entry.Path) // Read file content here
		if err != nil {
			fmt.Fprintf(writer, "[読み込みエラー] %v\n", err) // Handle read error
		} else if isBinaryFile(content) {
			fmt.Fprintln(writer, "[バイナリファイルのためスキップ]") // Handle binary file
		} else {
			fmt.Fprintln(writer, string(content)) // Write content
		}

		fmt.Fprintln(writer, "------------------------")
	}
}
