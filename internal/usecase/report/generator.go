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

// Generator はレポート生成機能を提供します
type Generator struct{}

// NewGenerator は新しい Generator インスタンスを作成します
func NewGenerator() *Generator {
	return &Generator{}
}

// CreateOutputFile は出力ファイルを作成します
func (g *Generator) CreateOutputFile(outputDir string) (*os.File, string, error) {
	timestamp := time.Now().Format("20060102_150405")
	outputPath := filepath.Join(outputDir, fmt.Sprintf("output_%s.txt", timestamp))

	outputFile, err := os.Create(outputPath)
	if err != nil {
		return nil, "", fmt.Errorf("出力ファイルの作成に失敗しました: %w", err)
	}

	return outputFile, outputPath, nil
}

// WriteFileSystemStructure はファイルシステムの構造を出力します
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

// WriteFileContents はファイルの内容を出力します
func (g *Generator) WriteFileContents(writer io.Writer, entries []model.FileSystemEntry) {
	fmt.Fprintln(writer, "\n===== ファイル内容 =====")

	for _, entry := range entries {
		if entry.IsDir {
			continue
		}

		fmt.Fprintf(writer, "----- %s -----\n", entry.RelPath)
		if entry.ReadErr != nil {
			fmt.Fprintf(writer, "[読み込みエラー] %v\n", entry.ReadErr)
		} else {
			fmt.Fprintln(writer, string(entry.Content))
		}
		fmt.Fprintln(writer, "------------------------")
	}
}
