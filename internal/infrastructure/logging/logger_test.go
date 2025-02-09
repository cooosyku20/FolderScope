package logging

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"
)

func TestJSONLogger(t *testing.T) {
	tests := []struct {
		name    string
		level   string
		message string
		err     error
	}{
		{
			name:    "エラーなしのログ",
			level:   "info",
			message: "テストメッセージ",
			err:     nil,
		},
		{
			name:    "エラーありのログ",
			level:   "error",
			message: "エラーメッセージ",
			err:     errors.New("テストエラー"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf strings.Builder
			logger := NewJSONLogger(&buf)

			logger.Log(tt.level, tt.message, tt.err)

			// 出力を検証
			output := buf.String()
			var logEntry LogEntry
			if err := json.Unmarshal([]byte(strings.TrimSpace(output)), &logEntry); err != nil {
				t.Errorf("JSONの解析に失敗: %v", err)
			}

			// 各フィールドを検証
			if logEntry.Message != tt.message {
				t.Errorf("メッセージが不正: got %v, want %v", logEntry.Message, tt.message)
			}
			if logEntry.Level != tt.level {
				t.Errorf("ログレベルが不正: got %v, want %v", logEntry.Level, tt.level)
			}
			if tt.err != nil {
				if logEntry.Error != tt.err.Error() {
					t.Errorf("エラーメッセージが不正: got %v, want %v", logEntry.Error, tt.err.Error())
				}
			} else if logEntry.Error != "" {
				t.Errorf("エラーメッセージが不正: got %v, want empty", logEntry.Error)
			}

			// タイムスタンプが現在時刻に近いことを確認
			logTime, err := time.Parse(time.RFC3339, logEntry.Timestamp)
			if err != nil {
				t.Errorf("タイムスタンプの解析に失敗: %v", err)
			}
			
			timeDiff := time.Since(logTime)
			if timeDiff > time.Minute {
				t.Errorf("タイムスタンプが不正: got %v, 現在との差が1分以上", logEntry.Timestamp)
			}
		})
	}
}
