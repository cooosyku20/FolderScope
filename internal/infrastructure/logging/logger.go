// Package logging はロギング機能を提供します
package logging

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"time"
)

// LogEntry はログエントリを表す構造体です
type LogEntry struct {
	// Timestamp はログが記録された時刻をRFC3339形式で表します
	Timestamp string `json:"timestamp"`
	// Level はログレベル（INFO, WARN, ERROR等）を表します
	Level string `json:"level"`
	// Message はログメッセージの内容を表します
	Message string `json:"message"`
	// Error はエラーが発生した場合のエラーメッセージを表します
	Error string `json:"error,omitempty"`
}

// Logger は構造化ログを出力するためのインターフェースです
type Logger interface {
	Log(level, message string, err error)
}

// JSONLogger はJSONフォーマットでログを出力するロガーです
type JSONLogger struct {
	writer io.Writer
}

// NewJSONLogger は新しいJSONLoggerインスタンスを作成します
func NewJSONLogger(writer io.Writer) *JSONLogger {
	if writer == nil {
		writer = os.Stdout
	}
	return &JSONLogger{writer: writer}
}

// Log はメッセージをJSONフォーマットでログ出力します
func (l *JSONLogger) Log(level, message string, err error) {
	entry := LogEntry{
		Timestamp: time.Now().Format(time.RFC3339),
		Level:     level,
		Message:   message,
	}

	if err != nil {
		entry.Error = err.Error()
	}

	jsonData, err := json.Marshal(entry)
	if err != nil {
		fmt.Fprintf(os.Stderr, "ログのJSONエンコードに失敗: %v\n", err)
		return
	}

	fmt.Fprintln(l.writer, string(jsonData))
}
