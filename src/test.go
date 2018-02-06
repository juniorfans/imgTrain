package main

import (
	"log"
	"github.com/lxn/walk"
	. "github.com/lxn/walk/declarative"
	"github.com/lxn/win"
)

type TMainWindow struct {
	*walk.MainWindow
}

func (mmw *TMainWindow) WndProc(hwnd win.HWND, msg uint32,wParam, lParam uintptr) uintptr {
	switch msg {
	case win.WM_SYSCOMMAND:
		log.Println("here", wParam, win.SC_MINIMIZE)
	}
	return mmw.MainWindow.WndProc(hwnd, msg, wParam, lParam)
}
func main_() {
	mmw := &TMainWindow{}
	if err := (MainWindow {
		AssignTo: &mmw.MainWindow,
		Title: "Minimize to hide",
		Size: Size{400,300},
		Layout: HBox{},
	}).Create(); err != nil {
		log.Fatal(err)
	}
	walk.InitWrapperWindow(mmw)
	mmw.Run()
}