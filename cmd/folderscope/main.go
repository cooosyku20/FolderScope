// Package main はアプリケーションのエントリーポイントを提供します
package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"FolderScope/internal/gui"
	"FolderScope/internal/infrastructure/filesystem"
	"FolderScope/internal/infrastructure/logging"
	"FolderScope/internal/usecase/report"
)

func main() {
	// ロガーの初期化
	logger := logging.NewJSONLogger(os.Stdout)

	// ファイルシステムスキャナーの初期化
	scanner := filesystem.NewScanner(logger)

	// ディレクトリセレクターの初期化（Fyneベース）
	selector := gui.NewDirectorySelector(scanner)

	// レポートジェネレーターの初期化
	generator := report.NewGenerator()

	// フォルダ選択処理の実行
	dirs, err := gui.SelectDirectories(selector)
	if err != nil {
		logger.Log("ERROR", "フォルダ選択に失敗", err)
		log.Fatalf("エラー: %v", err)
	}

	sourceDir := dirs.Source
	outputDir := dirs.Output
	logger.Log("INFO", fmt.Sprintf("選択されたフォルダ - 調査対象: %s, 出力先: %s", sourceDir, outputDir), nil)

	// 出力ファイルの作成
	outputFile, outputPath, err := generator.CreateOutputFile(outputDir)
	if err != nil {
		logger.Log("ERROR", "出力ファイルの作成に失敗", err)
		log.Fatalf("エラー: %v", err)
	}
	defer outputFile.Close()
	logger.Log("INFO", "出力ファイルを作成しました", nil)

	// フォルダ構造のスキャン
	entries, err := scanner.Scan(context.Background(), sourceDir)
	if err != nil {
		logger.Log("ERROR", "フォルダ構造のスキャンに失敗", err)
		log.Fatalf("エラー: %v", err)
	}
	logger.Log("INFO", "フォルダ構造のスキャンが完了しました", nil)

	// レポートの生成
	generator.WriteFileSystemStructure(outputFile, entries)
	generator.WriteFileContents(outputFile, entries)
	logger.Log("INFO", fmt.Sprintf("レポートを生成しました: %s", outputPath), nil)

	logger.Log("INFO", "処理が完了しました", nil)
	log.Printf("処理が完了しました。出力先: %s\n", outputPath)

	// プログラム終了前にEnterキーの入力を待機
	fmt.Print("\nEnterキーを押して終了してください...")
	fmt.Scanln()
}
