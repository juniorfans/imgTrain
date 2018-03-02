// Copyright 2012 The Walk Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package dialogs


import (
	"github.com/lxn/walk"
	. "github.com/lxn/walk/declarative"

	"strconv"
	"imgSearch/src/dbOptions"
)


func ShowPickDBDlg(resSig *chan uint8) {
	mw := &PickDBDlgWnd{}

	if _, err := (MainWindow{
		AssignTo: &mw.MainWindow,
		Title:    "选择图片库",
		MinSize:  Size{200, 150},
		MaxSize:  Size{200, 150},
		Size:     Size{200, 150},
		Layout:   VBox{MarginsZero: true},
		Children: []Widget{
			TextEdit{
				AssignTo: &mw.textEditor,
				MinSize:  Size{200, 80},
				MaxSize:  Size{200, 80},
				Font:Font{PointSize:16},
				ReadOnly: false,
			},

			Label{
				MinSize:Size{200,20},
				MaxSize:Size{200,20},
				Text: "---------------输入 img dbid--------------",
			},
			PushButton{
				//AssignTo: &mw.textEditor,
				MinSize:  Size{200, 50},
				MaxSize:  Size{200, 50},
				Text:"确认选择",
				OnClicked:func(){
					sel := mw.textEditor.Text()
					selInt, err := strconv.Atoi(sel)
					if nil!=err || selInt > 255 || selInt < 0{
						walk.MsgBox(mw, "Error", "请输入合法的 dbid (0 -> 255)" , walk.MsgBoxIconInformation)
						return
					}else{
						if dbOptions.IsValidImgId(uint8(selInt)){
							res := uint8(selInt)
							walk.MsgBox(mw, "Error", "已选择图片库: " + sel, walk.MsgBoxIconInformation)
							mw.Close()
							(*resSig) <- res
						}else{
							walk.MsgBox(mw, "Error", "无效的图片库/被别的进程使用中: " + sel, walk.MsgBoxIconInformation)
						}
					}
				},
			},
			Label{
				MinSize:Size{200,20},
				MaxSize:Size{200,20},
			},

		},
	}.Run()); err != nil {

	}
}

type PickDBDlgWnd struct {
	*walk.MainWindow
	textEditor *walk.TextEdit
}
