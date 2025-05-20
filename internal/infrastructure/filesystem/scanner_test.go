package filesystem

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"

	"FolderScope/internal/domain/model"

	"github.com/stretchr/testify/assert"
)

type mockLogger struct {
	logs []struct {
		level   string
		message string
		err     error
	}
}

func (m *mockLogger) Log(level, message string, err error) {
	m.logs = append(m.logs, struct {
		level   string
		message string
		err     error
	}{level, message, err})
}

func TestFileSystemScanner_ValidateDirectoryPath(t *testing.T) {
	logger := &mockLogger{}
	scanner := NewScanner(logger, nil, false)

	// テスト用の一時ディレクトリを作成
	tempDir, err := os.MkdirTemp("", "scanner_test")
	if err != nil {
		t.Fatalf("一時ディレクトリの作成に失敗: %v", err)
	}
	defer os.RemoveAll(tempDir)

	tests := []struct {
		name    string
		path    string
		wantErr bool
	}{
		{
			name:    "有効なディレクトリパス",
			path:    tempDir,
			wantErr: false,
		},
		{
			name:    "空のパス",
			path:    "",
			wantErr: true,
		},
		{
			name:    "存在しないパス",
			path:    filepath.Join(tempDir, "notexist"),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := scanner.ValidateDirectoryPath(tt.path)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateDirectoryPath() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestFileSystemScanner_Scan(t *testing.T) {
	logger := &mockLogger{}

	// テスト用のディレクトリ構造を作成
	baseDir, err := os.MkdirTemp("", "scan_test_base")
	assert.NoError(t, err)
	defer os.RemoveAll(baseDir)

	// テストファイル/ディレクトリのセットアップ
	// `.git` ディレクトリ (デフォルトで無視される)
	err = os.MkdirAll(filepath.Join(baseDir, ".git", "objects"), 0755)
	assert.NoError(t, err)
	_, err = os.Create(filepath.Join(baseDir, ".git", "HEAD"))
	assert.NoError(t, err)

	// 通常のディレクトリとファイル
	err = os.Mkdir(filepath.Join(baseDir, "dir1"), 0755)
	assert.NoError(t, err)
	_, err = os.Create(filepath.Join(baseDir, "dir1", "file1a.txt"))
	assert.NoError(t, err)
	_, err = os.Create(filepath.Join(baseDir, "file1.txt"))
	assert.NoError(t, err)

	// バイナリファイル (意図的に非テキストデータを含む)
	binaryContent := []byte{0x00, 0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0a, 0x0b, 0x0c, 0x0d, 0x0e, 0x0f} // より多くの非ASCII文字
	err = os.WriteFile(filepath.Join(baseDir, "binary.bin"), binaryContent, 0644)
	assert.NoError(t, err)

	// 無視パターンにマッチするファイル
	_, err = os.Create(filepath.Join(baseDir, "ignored.log"))
	assert.NoError(t, err)
	err = os.Mkdir(filepath.Join(baseDir, "build"), 0755)
	assert.NoError(t, err)
	_, err = os.Create(filepath.Join(baseDir, "build", "output.o"))
	assert.NoError(t, err)

	// シンボリックリンク (サポートされていれば)
	// 注意: シンボリックリンクのテストはOSや環境に依存する可能性がある
	// ここでは、シンボリックリンク自体がスキャン対象になるか（エラーにならないか）を主に確認
	symlinkTarget := filepath.Join(baseDir, "file1.txt")
	symlinkPath := filepath.Join(baseDir, "symlink_to_file1.txt")
	// os.Symlink のエラーは必ずしもテスト失敗を意味しない（例えばWindowsの権限不足）
	// そのため、ここではエラーをチェックしないか、より寛容なチェックを行う
	_ = os.Symlink(symlinkTarget, symlinkPath) // エラーは無視する

	testCases := []struct {
		name              string
		rootPath          string
		ignorePatterns    []string
		ignoreBinaryFiles bool
		wantEntries       []model.FileSystemEntry
		wantErr           bool
	}{
		{
			name:              "基本スキャン（バイナリ含む、特定無視なし）",
			rootPath:          baseDir,
			ignorePatterns:    nil,
			ignoreBinaryFiles: false,
			wantEntries: []model.FileSystemEntry{
				{Path: filepath.Join(baseDir, "dir1"), IsDir: true, RelPath: "dir1", Depth: 0},
				{Path: filepath.Join(baseDir, "dir1", "file1a.txt"), IsDir: false, RelPath: "dir1/file1a.txt", Depth: 1, IsBinary: false},
				{Path: filepath.Join(baseDir, "file1.txt"), IsDir: false, RelPath: "file1.txt", Depth: 0, IsBinary: false},
				{Path: filepath.Join(baseDir, "binary.bin"), IsDir: false, RelPath: "binary.bin", Depth: 0, IsBinary: true},
				{Path: filepath.Join(baseDir, "ignored.log"), IsDir: false, RelPath: "ignored.log", Depth: 0, IsBinary: false},
				{Path: filepath.Join(baseDir, "build"), IsDir: true, RelPath: "build", Depth: 0},
				{Path: filepath.Join(baseDir, "build", "output.o"), IsDir: false, RelPath: "build/output.o", Depth: 1, IsBinary: false},
				{Path: symlinkPath, IsDir: false, RelPath: "symlink_to_file1.txt", Depth: 0, IsBinary: false},
			},
			wantErr: false,
		},
		{
			name:              "バイナリファイルを無視",
			rootPath:          baseDir,
			ignorePatterns:    nil,
			ignoreBinaryFiles: true,
			wantEntries: []model.FileSystemEntry{
				{Path: filepath.Join(baseDir, "dir1"), IsDir: true, RelPath: "dir1", Depth: 0},
				{Path: filepath.Join(baseDir, "dir1", "file1a.txt"), IsDir: false, RelPath: "dir1/file1a.txt", Depth: 1, IsBinary: false},
				{Path: filepath.Join(baseDir, "file1.txt"), IsDir: false, RelPath: "file1.txt", Depth: 0, IsBinary: false},
				// binary.bin は無視される
				{Path: filepath.Join(baseDir, "ignored.log"), IsDir: false, RelPath: "ignored.log", Depth: 0, IsBinary: false},
				{Path: filepath.Join(baseDir, "build"), IsDir: true, RelPath: "build", Depth: 0},
				{Path: filepath.Join(baseDir, "build", "output.o"), IsDir: false, RelPath: "build/output.o", Depth: 1, IsBinary: false},
				{Path: symlinkPath, IsDir: false, RelPath: "symlink_to_file1.txt", Depth: 0, IsBinary: false},
			},
			wantErr: false,
		},
		{
			name:              "特定のパターンを無視（*.log と build/ ディレクトリ）",
			rootPath:          baseDir,
			ignorePatterns:    []string{"*.log", "build/"},
			ignoreBinaryFiles: false, // バイナリは無視しない設定でテスト
			wantEntries: []model.FileSystemEntry{
				{Path: filepath.Join(baseDir, "dir1"), IsDir: true, RelPath: "dir1", Depth: 0},
				{Path: filepath.Join(baseDir, "dir1", "file1a.txt"), IsDir: false, RelPath: "dir1/file1a.txt", Depth: 1, IsBinary: false},
				{Path: filepath.Join(baseDir, "file1.txt"), IsDir: false, RelPath: "file1.txt", Depth: 0, IsBinary: false},
				{Path: filepath.Join(baseDir, "binary.bin"), IsDir: false, RelPath: "binary.bin", Depth: 0, IsBinary: true},
				// ignored.log は無視される
				// build ディレクトリとその中身は無視される
				{Path: symlinkPath, IsDir: false, RelPath: "symlink_to_file1.txt", Depth: 0, IsBinary: false},
			},
			wantErr: false,
		},
		{
			name:              "存在しないルートパス",
			rootPath:          filepath.Join(baseDir, "non_existent_dir"),
			ignorePatterns:    nil,
			ignoreBinaryFiles: false,
			wantEntries:       nil, // エラーなのでエントリは空
			wantErr:           true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			scanner := NewScanner(logger, tc.ignorePatterns, tc.ignoreBinaryFiles)
			entries, err := scanner.Scan(context.Background(), tc.rootPath)

			if tc.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				// スライスをソートして比較可能にする
				sortEntries(entries)
				sortEntries(tc.wantEntries)

				// Symlink の存在確認 (エラーを許容し、存在すれば期待値に追加)
				// このアプローチは symlink のテストをより堅牢にする
				// ただし、期待値の管理が複雑になるので、ここでは単純化のためコメントアウト
				// updateWantEntriesForSymlink(tc.wantEntries, symlinkPath, baseDir, tc.ignorePatterns, tc.ignoreBinaryFiles)

				// reflect.DeepEqual はポインタフィールド（ReadErrなど）も比較するため、
				// Path, IsDir, RelPath, Depth, IsBinary のみを比較するカスタム比較関数を検討する。
				// ここでは簡略化のため DeepEqual を使用するが、必要に応じて修正する。
				// assert.Equal(t, len(tc.wantEntries), len(entries), "Number of entries mismatch")

				// 詳細な比較を行う
				// if !reflect.DeepEqual(entries, tc.wantEntries) {
				// 	t.Errorf("Scan() got = %v, want %v", entries, tc.wantEntries)
				// 	// 差分を詳細に出力
				// 	for i := 0; i < min(len(entries), len(tc.wantEntries)); i++ {
				// 		if !reflect.DeepEqual(entries[i], tc.wantEntries[i]) {
				// 			t.Logf("Difference at index %d: got %+v, want %+v", i, entries[i], tc.wantEntries[i])
				// 		}
				// 	}
				// }
				// assert.True(t, reflect.DeepEqual(entries, tc.wantEntries), "Entries mismatch.\nGot: %v\nWant: %v", entries, tc.wantEntries)
				assertFileSystemEntrySlicesEqual(t, tc.wantEntries, entries)
			}
		})
	}
}

// sortEntries は FileSystemEntry のスライスを Path でソートするヘルパー関数
func sortEntries(entries []model.FileSystemEntry) {
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Path < entries[j].Path
	})
}

// assertFileSystemEntrySlicesEqual は2つの FileSystemEntry スライスが（主要なフィールドにおいて）等しいか検証します。
// ReadErr は比較から除外します。
func assertFileSystemEntrySlicesEqual(t *testing.T, expected, actual []model.FileSystemEntry) {
	t.Helper()
	if len(expected) != len(actual) {
		t.Errorf("Number of entries mismatch. Expected %d, got %d.\nExpected: %+v\nActual:   %+v", len(expected), len(actual), expected, actual)
		return
	}

	// 比較のために正規化とソートを行う
	normalizeAndSort := func(entries []model.FileSystemEntry) []model.FileSystemEntry {
		normalized := make([]model.FileSystemEntry, len(entries))
		for i, e := range entries {
			// Pathを絶対パスから相対パスに変換して比較を安定させる (もしベースパスが同じなら)
			// ここでは、テストケースで RelPath が正しく設定されていることを前提とする。
			// IsBinary の値も比較対象に含める
			normalized[i] = model.FileSystemEntry{
				Path:     e.Path, // Path はそのまま比較（テストケースで期待値を設定）
				IsDir:    e.IsDir,
				RelPath:  e.RelPath,
				Depth:    e.Depth,
				IsBinary: e.IsBinary, // IsBinary を比較対象に
				// ReadErr は無視
			}
		}
		sort.Slice(normalized, func(i, j int) bool {
			return normalized[i].Path < normalized[j].Path
		})
		return normalized
	}

	normalizedExpected := normalizeAndSort(expected)
	normalizedActual := normalizeAndSort(actual)

	if !reflect.DeepEqual(normalizedExpected, normalizedActual) {
		t.Errorf("FileSystemEntry slices are not equal.")
		t.Logf("Expected (normalized):\n%s", prettyPrintEntries(normalizedExpected))
		t.Logf("Actual (normalized):\n%s", prettyPrintEntries(normalizedActual))

		// 詳細な差分出力
		for i := 0; i < len(normalizedExpected); i++ {
			if i >= len(normalizedActual) || !reflect.DeepEqual(normalizedExpected[i], normalizedActual[i]) {
				expVal := normalizedExpected[i]
				actVal := model.FileSystemEntry{} // 範囲外なら空
				if i < len(normalizedActual) {
					actVal = normalizedActual[i]
				}
				t.Logf("Mismatch at index %d (after sorting):\nExpected: Path=%s, IsDir=%t, RelPath=%s, Depth=%d, IsBinary=%t\nActual:   Path=%s, IsDir=%t, RelPath=%s, Depth=%d, IsBinary=%t",
					i,
					expVal.Path, expVal.IsDir, expVal.RelPath, expVal.Depth, expVal.IsBinary,
					actVal.Path, actVal.IsDir, actVal.RelPath, actVal.Depth, actVal.IsBinary,
				)
				if i < len(normalizedActual) && normalizedExpected[i].Path != normalizedActual[i].Path {
					t.Logf("  Path diff: E: %s, A: %s", normalizedExpected[i].Path, normalizedActual[i].Path)
				}
				if i < len(normalizedActual) && normalizedExpected[i].RelPath != normalizedActual[i].RelPath {
					t.Logf("  RelPath diff: E: %s, A: %s", normalizedExpected[i].RelPath, normalizedActual[i].RelPath)
				}
			}
		}
	}
}

func prettyPrintEntries(entries []model.FileSystemEntry) string {
	var sb strings.Builder
	for _, e := range entries {
		sb.WriteString(formatEntry(e))
		sb.WriteString("\n")
	}
	return sb.String()
}

func formatEntry(entry model.FileSystemEntry) string {
	return strings.ReplaceAll(
		filepath.ToSlash(
			fmt.Sprintf("Path=%s, IsDir=%t, RelPath=%s, Depth=%d, IsBinary=%t",
				entry.Path, entry.IsDir, entry.RelPath, entry.Depth, entry.IsBinary),
		),
		"\\", "/", // Ensure slashes are consistent for logging
	)
}

// isWindows は現在のOSがWindowsかどうかを返します。
// func isWindows() bool {
// 	return runtime.GOOS == "windows"
// }

// min は2つのintの小さい方を返します。
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// updateWantEntriesForSymlink は、シンボリックリンクの存在に基づいて期待されるエントリリストを更新します。
// これは、シンボリックリンクのテストをより柔軟にするためのヘルパーですが、今回の修正では直接使用しません。
// func updateWantEntriesForSymlink(wantEntries []model.FileSystemEntry, symlinkPath, baseDir string, ignorePatterns []string, ignoreBinaryFiles bool) []model.FileSystemEntry {
// 	// シンボリックリンクが存在し、無視パターンにマッチしない場合、期待値に追加
// 	// これは例であり、実際のロジックはより複雑になる可能性がある
// 	if _, err := os.Lstat(symlinkPath); err == nil {
// 		// シンボリックリンクが無視パターンにマッチするかどうかを確認
// 		// ignore.Match (filepath.Match と同様) を使用
// 		// ここでは簡略化のため、無視パターンのチェックは省略
// 		isIgnored := false
// 		for _, pattern := range ignorePatterns {
// 			// 実際には filepath.Match などを使用
// 			if strings.Contains(filepath.Base(symlinkPath), strings.TrimSuffix(strings.TrimPrefix(pattern, "*"), "*")) {
// 				isIgnored = true
// 				break
// 			}
// 		}

// 		if !isIgnored {
// 			// シンボリックリンク先のファイルがバイナリかどうかを判断
// 			// ここでは簡略化のため、symlinkTarget がテキストファイルであると仮定
// 			isBinary := false
// 			if ignoreBinaryFiles && isBinary {
// 				// バイナリ無視が有効で、かつファイルがバイナリなら追加しない
// 			} else {
// 				// wantEntries = append(wantEntries, model.FileSystemEntry{
// 				// 	Path:     symlinkPath,
// 				// 	IsDir:    false, // シンボリックリンクはファイルとして扱われる
// 				// 	RelPath:  filepath.Base(symlinkPath),
// 				// 	Depth:    1,
// 				// 	IsBinary: isBinary, // リンク先のファイルに依存
// 				// })
// 				// sortEntries(wantEntries) // 再ソート
// 			}
// 		}
// 	}
// 	return wantEntries
// }

// TestScanner_isBinaryFile は削除されました。
// scanner.go から isBinaryFile が削除されたため。

// strconv と time をインポートリストに追加する必要がある
// import (
// 	"context"
// 	"os"
// 	"path/filepath"
// 	"reflect"
// 	"sort"
// 	"strconv" // 追加
// 	"strings"
// 	"testing"
// 	"time"    // 追加

// 	"github.com/stretchr/testify/assert"

// 	"github.com/kubefriendly/kubeswitch/internal/domain/model"
// )
// 上記は import ブロックにまとめて記述されているので、個別の追加は不要。
// ただし、strconv, time が実際に使われているか確認。
// -> formatEntry で使われている。
// -> reflect は assertFileSystemEntrySlicesEqual で使われている。
// -> sort は sortEntries で使われている。
// -> strings は formatEntry で使われている。
// -> testify/assert は TestFileSystemScanner_Scan で使われている。
// -> domain/model は TestFileSystemScanner_Scan で使われている。

// 最終的なインポートリストの確認:
// "context"
// "os"
// "path/filepath"
// "reflect"
// "sort"
// "strconv"
// "strings"
// "testing"
// "time" // time は formatEntry のコメントアウト部分で使われていたが、現在は使われていない。削除可能。
// "github.com/stretchr/testify/assert"
// "github.com/kubefriendly/kubeswitch/internal/domain/model"

// time は不要なので、インポートから削除することを検討。
// -> `formatEntry` 内のコメントアウトされた `entry.ModTime.Format(time.RFC3339)` で使われていた。
// -> 現在は使われていないので `time` のインポートは不要。
//
// `strconv` は `formatEntry` で `strconv.FormatBool` と `strconv.Itoa` を使用しているので必要。

// `reflect.DeepEqual` を使用している箇所での注意:
// `model.FileSystemEntry` に `ReadErr error` フィールドがある。
// `error` 型は直接 `DeepEqual` で比較すると、同じエラーメッセージでも異なるインスタンスであれば `false` になる。
// そのため、`assertFileSystemEntrySlicesEqual` で `ReadErr` を除外して比較するか、
// エラーの場合はエラーメッセージのみを比較するなどの工夫が必要。
// 今回の `assertFileSystemEntrySlicesEqual` では `ReadErr` を明示的に比較対象から外している。
// (normalizedEntry を作成する際に `ReadErr` を含めていない)

// `symlinkPath` の `IsBinary` フラグについて:
// `wantEntries` の `symlinkPath` エントリの `IsBinary` は、シンボリックリンクが指すファイルの内容に依存する。
// `symlinkTarget := filepath.Join(baseDir, "file1.txt")` であり、`file1.txt` は空のテキストファイルなので、
// `IsBinary: false` で正しい。

// `binary.bin` の `IsBinary` フラグについて:
// `wantEntries` の `binary.bin` エントリの `IsBinary` は、`ignoreBinaryFiles` の設定に依存する。
// - `ignoreBinaryFiles: false` の場合: `IsBinary: true` (ファイル自体がバイナリなので)
// - `ignoreBinaryFiles: true` の場合: `binary.bin` エントリ自体が `wantEntries` に含まれない。
// これはテストケースで正しく設定されている。

// `build/output.o` の `IsBinary` フラグについて:
// `output.o` は空ファイルとして作成されているため、現在の `isBinary` の判定ロジックではテキストファイル扱い (IsBinary: false) となる。
// もし `.o` ファイルを常にバイナリとして扱いたい場合は、ファイル拡張子に基づく判定ロジックを `scanner` に追加する必要があるが、
// 現在の `scanner` は内容ベースの判定のみ。
// よって、`IsBinary: false` で正しい。

// 全体的なテスト構造:
// - `TestFileSystemScanner_ValidateDirectoryPath`: パス検証ロジックのテスト。
// - `TestFileSystemScanner_Scan`: 主なスキャン機能のテスト。
//   - セットアップ: テスト用のファイルとディレクトリ構造を作成。
//     - `.git` (デフォルト無視)
//     - 通常ファイル、ディレクトリ
//     - バイナリファイル
//     - 無視パターン用ファイル (`.log`, `build/`)
//     - シンボリックリンク
//   - テストケース:
//     - 基本スキャン (バイナリ含む)
//     - バイナリファイル無視
//     - 特定パターン無視
//     - 存在しないルートパス (エラーケース)
//   - アサーション:
//     - `scanner.Scan` の結果 (エントリリストとエラー) を期待値と比較。
//     - エントリリストの比較は、ソートしてから `reflect.DeepEqual` またはカスタム比較関数 (`assertFileSystemEntrySlicesEqual`) を使用。
//     - `assertFileSystemEntrySlicesEqual` は、`Path`, `IsDir`, `RelPath`, `Depth`, `IsBinary` のフィールドを比較する。`ReadErr` は無視。

// `filepath.Join` と `RelPath` の一貫性:
// `wantEntries` の `RelPath` は、`filepath.Join` を使わずに手動で結合されている場合がある (`filepath.Join("dir1", "file1a.txt")`)。
// OS によってパス区切り文字が異なるため、`filepath.Join` で統一するのが望ましい。
// ただし、Go の `filepath` パッケージは実行環境の OS に適した区切り文字を使用するため、
// 文字列リテラルで `/` を使っていても、Windows 以外では問題ない。
// Windows でも `filepath.ToSlash` などで比較前に正規化すれば問題ない。
// `formatEntry` で `filepath.ToSlash` を使用しているので、ログ出力は統一される。
// `assertFileSystemEntrySlicesEqual` 内の `normalized[i].RelPath = e.RelPath` では、
// `e.RelPath` が期待通りに設定されていれば良い。
// `model.FileSystemEntry` の `RelPath` は `scanner` によって設定される。
// `scanner` 内で `filepath.Rel` や `filepath.Join` が適切に使われていれば、
// `RelPath` はプラットフォーム依存性のない形式 (通常は `/` 区切り) で格納されることが期待される。
// もしそうでなければ、比較前に正規化が必要。
// `scanner.go` の実装を確認すると、`relPath := filepath.ToSlash(strings.TrimPrefix(strings.TrimPrefix(path, root), string(filepath.Separator)))`
// のように `filepath.ToSlash` を使って `/` 区切りに正規化しているため、`RelPath` は `/` 区切りで統一されている。
// よって、`wantEntries` の `RelPath` も `/` 区切りで記述するのが良い。
// (例: `RelPath: "dir1/file1a.txt"`)
// 現在のコードでは `filepath.Join` が使われている箇所と文字列リテラルが混在しているが、
// `filepath.Join` は OS 依存の区切り文字を生成するため、テストの期待値としては `/` 区切りに統一する方が良い。
// ただし、Go のテストでは、`filepath.Join` で生成されたパスと、期待値として `/` 区切りで書かれたパスを
// `filepath.Clean` や `filepath.ToSlash` を通して比較すれば問題ない。
// `assertFileSystemEntrySlicesEqual` で `Path` を比較キーにしているため、`RelPath` の微妙な違いはここでは問題になりにくいが、
// `RelPath` 自体を比較する場合は注意が必要。

// 今回の修正では、`wantEntries` の `RelPath` の期待値を、`scanner` が生成する `/` 区切りの形式に合わせる。

// 修正後の wantEntries の RelPath (例):
// {Path: filepath.Join(baseDir, "dir1", "file1a.txt"), IsDir: false, RelPath: "dir1/file1a.txt", Depth: 2, IsBinary: false},
// {Path: filepath.Join(baseDir, "build", "output.o"), IsDir: false, RelPath: "build/output.o", Depth: 2, IsBinary: false},

// これに合わせて wantEntries を修正する。
// 以下の修正は edit_file の code_edit に含める。

// 修正箇所: TestFileSystemScanner_Scan の testCases 内の wantEntries の RelPath
// 例: RelPath: filepath.Join("dir1", "file1a.txt") を RelPath: "dir1/file1a.txt" に変更
//     RelPath: filepath.Join("build", "output.o") を RelPath: "build/output.o" に変更
//     RelPath: filepath.Base(symlinkPath) を RelPath: "symlink_to_file1.txt" (ファイル名そのまま) に変更 (scanner の動作に合わせる)
// scanner の RelPath 生成ロジック: `filepath.ToSlash(strings.TrimPrefix(strings.TrimPrefix(path, root), string(filepath.Separator)))`
// これにより、`symlink_to_file1.txt` はそのまま `symlink_to_file1.txt` となる。

// wantEntries の修正を反映した code_edit を作成する。
// (実際の編集時には、このコメントブロック内の思考は省略し、最終的なコードのみを提示する)
// `time` の import も削除する。
// `formatEntry` 内のコメントも整理する。
