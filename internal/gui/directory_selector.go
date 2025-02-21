// Package gui はGUIを提供します
package gui

import (
	"fmt"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/dialog"
)

// Default window size constants
const (
	DefaultWindowWidth  = 800
	DefaultWindowHeight = 600
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
	done := make(chan struct{})
	var result struct {
		path string
		err  error
	}

	a := app.New()
	w := a.NewWindow(title)
	w.Resize(fyne.NewSize(DefaultWindowWidth, DefaultWindowHeight))

	// ダイアログを作成して表示
	d := dialog.NewFolderOpen(func(selectedURI fyne.ListableURI, err error) {
		// コールバック: ユーザーがディレクトリを選択した結果を受け取る
		if err != nil {
			result.err = fmt.Errorf("フォルダ選択エラー: %w", err)
			close(done)
			return
		}
		if selectedURI == nil {
			result.err = fmt.Errorf("ユーザーがキャンセルしました")
			close(done)
			return
		}
		path := selectedURI.Path()
		if err := s.validator.ValidateDirectoryPath(path); err != nil {
			result.err = fmt.Errorf("パス検証エラー: %w", err)
			close(done)
			return
		}
		result.path = path
		close(done)
	}, w)
	d.Show()
	w.Show()

	// イベントループ内で待機するため、a.Run() を実行
	go func() {
		<-done
		a.Quit()
	}()
	a.Run()
	return result.path, result.err
}

// DirectoryPaths は、選択されたディレクトリパスを保持する構造体です
type DirectoryPaths struct {
	Source string // 調査対象フォルダ
	Output string // 出力先フォルダ
}

// openFolderDialog は、指定のウィンドウとタイトルでフォルダ選択ダイアログを表示し、
// 選択されたパスまたはエラーを返します。
// この関数は、コールバックを連鎖させるために使用します。
func openFolderDialog(w fyne.Window, title string, validator DirectoryValidator) (string, error) {
	var selectedPath string
	var resultErr error

	d := dialog.NewFolderOpen(func(selectedURI fyne.ListableURI, err error) {
		// コールバック: ユーザーがディレクトリを選択した結果を受け取る
		if err != nil {
			resultErr = fmt.Errorf("%s選択エラー: %w", title, err)
			w.Close()
			return
		}
		if selectedURI == nil {
			resultErr = fmt.Errorf("%sの選択がキャンセルされました", title)
			w.Close()
			return
		}
		path := selectedURI.Path()
		if err := validator.ValidateDirectoryPath(path); err != nil {
			resultErr = fmt.Errorf("%sが無効です: %w", title, err)
			w.Close()
			return
		}
		selectedPath = path
		// ダイアログ処理完了後、ウィンドウはそのまま残す
	}, w)
	w.SetTitle(title)
	d.Show()
	return selectedPath, resultErr
}

// SelectDirectories は、調査対象フォルダと出力先フォルダの選択を一括で行います。
// UI 操作はメインスレッド上で、コールバックを連鎖させる形で実現します。
func SelectDirectories(selector *DirectorySelector) (*DirectoryPaths, error) {
	a := app.New()
	w := a.NewWindow("FolderScope")
	w.Resize(fyne.NewSize(DefaultWindowWidth, DefaultWindowHeight))

	paths := &DirectoryPaths{}
	var currentError error

	// チェーン形式でダイアログを連続表示する
	// まず、調査対象フォルダの選択
	dialog.NewFolderOpen(func(sourceURI fyne.ListableURI, err error) {
		if err != nil {
			currentError = fmt.Errorf("調査対象フォルダの選択エラー: %w", err)
			w.Close()
			a.Quit()
			return
		}
		if sourceURI == nil {
			currentError = fmt.Errorf("調査対象フォルダの選択がキャンセルされました")
			w.Close()
			a.Quit()
			return
		}
		sourcePath := sourceURI.Path()
		if err := selector.validator.ValidateDirectoryPath(sourcePath); err != nil {
			currentError = fmt.Errorf("調査対象フォルダが無効です: %w", err)
			w.Close()
			a.Quit()
			return
		}
		paths.Source = sourcePath

		// 次に、出力先フォルダの選択
		dialog.NewFolderOpen(func(outputURI fyne.ListableURI, err error) {
			if err != nil {
				currentError = fmt.Errorf("出力先フォルダの選択エラー: %w", err)
				w.Close()
				a.Quit()
				return
			}
			if outputURI == nil {
				currentError = fmt.Errorf("出力先フォルダの選択がキャンセルされました")
				w.Close()
				a.Quit()
				return
			}
			outputPath := outputURI.Path()
			if err := selector.validator.ValidateDirectoryPath(outputPath); err != nil {
				currentError = fmt.Errorf("出力先フォルダが無効です: %w", err)
				w.Close()
				a.Quit()
				return
			}
			paths.Output = outputPath
			// 両方の選択が完了したのでウィンドウを閉じる
			w.Close()
			a.Quit()
		}, w).Show()

	}, w).Show()

	// ウィンドウ表示
	w.Show()
	a.Run()

	if currentError != nil {
		return nil, currentError
	}
	return paths, nil
}
