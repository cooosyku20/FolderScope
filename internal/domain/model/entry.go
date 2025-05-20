// package model はドメインモデルを定義します
package model

// FileSystemEntry はファイルシステムの要素（ファイルまたはディレクトリ）を表します
type FileSystemEntry struct {
	// Path は要素の絶対パスを表します
	Path string
	// IsDir はディレクトリであるかどうかを示します
	IsDir bool
	// RelPath はルートディレクトリからの相対パスを表します
	RelPath string
	// Depth はルートディレクトリからの深さを表します
	Depth int
	// ReadErr はファイル読み込み時のエラーを保持します
	ReadErr error
	// IsBinary はファイルがバイナリファイルであるかどうかを示します
	IsBinary bool
}
