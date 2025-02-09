package gui

import (
	"fmt"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/dialog"
)

// DirectoryValidator は、ディレクトリパスの検証を行うインターフェース
type DirectoryValidator interface {
	ValidateDirectoryPath(path string) error
}

// DirectorySelector は、Fyneを使用してディレクトリ選択を行う構造体
type DirectorySelector struct {
	validator DirectoryValidator
}

// NewDirectorySelector は、DirectorySelectorの新しいインスタンスを作成します
func NewDirectorySelector(validator DirectoryValidator) *DirectorySelector {
	return &DirectorySelector{
		validator: validator,
	}
}

// SelectDirectory は、Fyneダイアログを使用してディレクトリを選択し、
// 選択されたパスまたはエラーを返します
func (s *DirectorySelector) SelectDirectory(title string) (string, error) {
	var result struct {
		path string
		err  error
	}

	// Fyneアプリケーションの作成
	a := app.New()
	w := a.NewWindow(title)
	w.Resize(fyne.NewSize(800, 600))
	done := make(chan struct{})

	// ダイアログを作成して表示
	d := dialog.NewFolderOpen(func(lu fyne.ListableURI, err error) {
		if err != nil {
			result.err = fmt.Errorf("フォルダ選択エラー: %w", err)
			close(done)
			return
		}
		if lu == nil {
			result.err = fmt.Errorf("ユーザーがキャンセルしました")
			close(done)
			return
		}

		// 選択されたパスの検証
		path := lu.Path()
		if err := s.validator.ValidateDirectoryPath(path); err != nil {
			result.err = fmt.Errorf("パス検証エラー: %w", err)
			close(done)
			return
		}

		result.path = path
		close(done)
	}, w)

	// ダイアログを表示
	d.Show()
	w.Show()

	// イベントループを開始
	go func() {
		<-done
		a.Quit()
	}()

	// メインウィンドウを実行（メインゴルーチンでのみ実行可能）
	a.Run()

	return result.path, result.err
}

// DirectoryPaths は、選択されたディレクトリパスを保持する構造体です
type DirectoryPaths struct {
	Source string // 調査対象フォルダ
	Output string // 出力先フォルダ
}

// SelectDirectories は、調査対象フォルダと出力先フォルダの選択を一括で行います
func SelectDirectories(selector *DirectorySelector) (*DirectoryPaths, error) {
	// Fyneアプリケーションの作成
	a := app.New()
	w := a.NewWindow("FolderScope")
	w.Resize(fyne.NewSize(800, 600))

	var paths DirectoryPaths
	var currentError error
	done := make(chan struct{})

	var showSourceDialog func()
	var showOutputDialog func()

	// 出力先フォルダの選択ダイアログ
	showOutputDialog = func() {
		d := dialog.NewFolderOpen(func(lu fyne.ListableURI, err error) {
			if err != nil {
				currentError = fmt.Errorf("出力先フォルダの選択エラー: %w", err)
				close(done)
				return
			}
			if lu == nil {
				currentError = fmt.Errorf("出力先フォルダの選択がキャンセルされました")
				close(done)
				return
			}

			// パスの検証
			path := lu.Path()
			if err := selector.validator.ValidateDirectoryPath(path); err != nil {
				currentError = fmt.Errorf("出力先フォルダが無効です: %w", err)
				close(done)
				return
			}

			paths.Output = path
			close(done)
		}, w)
		w.SetTitle("出力先フォルダを選択")
		d.Show()
	}

	// 調査対象フォルダの選択ダイアログ
	showSourceDialog = func() {
		d := dialog.NewFolderOpen(func(lu fyne.ListableURI, err error) {
			if err != nil {
				currentError = fmt.Errorf("調査対象フォルダの選択エラー: %w", err)
				close(done)
				return
			}
			if lu == nil {
				currentError = fmt.Errorf("調査対象フォルダの選択がキャンセルされました")
				close(done)
				return
			}

			// パスの検証
			path := lu.Path()
			if err := selector.validator.ValidateDirectoryPath(path); err != nil {
				currentError = fmt.Errorf("調査対象フォルダが無効です: %w", err)
				close(done)
				return
			}

			paths.Source = path
			// 次のダイアログを表示
			showOutputDialog()
		}, w)
		w.SetTitle("調査対象フォルダを選択")
		d.Show()
	}

	// 最初のダイアログを表示
	showSourceDialog()
	w.Show()

	// イベントループを開始
	go func() {
		<-done
		a.Quit()
	}()

	// メインウィンドウを実行
	a.Run()

	if currentError != nil {
		return nil, currentError
	}

	return &paths, nil
}
