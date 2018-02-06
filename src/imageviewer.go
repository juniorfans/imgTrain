// Copyright 2010 The Walk Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"log"
	"fmt"
)

import (
	"github.com/lxn/walk"
	. "github.com/lxn/walk/declarative"
	"io"
	"bytes"
	"image/jpeg"
	"dbOptions"
	"imgIndex"
	"strconv"
	"github.com/BurntSushi/graphics-go/graphics"
	"image"
	"bufio"
	"os"
	"config"
	"github.com/lxn/win"
)

func main() {
	mw := new(MyMainWindow)
	mw.waitForStart = make(chan bool, 1)
	mw.waitForDBVisitor = make(chan bool, 1)

	go waitAndVisitDB(mw)

	RunWndThread(mw)
}

func RunWndThread(mw *MyMainWindow){
	stdin := bufio.NewReader(os.Stdin)

	var dbIdStr string
	fmt.Print("input image db: ")
	fmt.Fscan(stdin, &dbIdStr)
	dbIdInt, _:= strconv.Atoi(dbIdStr)

	mw.trainDBId = uint8(dbIdInt)
	var openAction *walk.Action

	if err := (MainWindow{
		AssignTo: &mw.MainWindow,
		Title:    "Train Img",
		MenuItems: []MenuItem{
			Menu{
				Text: "&File",
				Items: []MenuItem{
					Action{
						AssignTo:    &openAction,
						Text:        "&Show",
						Image:       "../img/open.png",
						OnTriggered: mw.openAction_Triggered,
					},
					Separator{},
					Action{
						Text:        "Exit",
						OnTriggered: func() { mw.Close() },
					},
				},
			},
			Menu{
				Text: "&Help",
				Items: []MenuItem{
					Action{
						Text:        "About",
						OnTriggered: mw.aboutAction_Triggered,
					},
				},
			},
		},
		ToolBarItems: []MenuItem{
			ActionRef{&openAction},
		},
		MinSize: Size{800, 600},
		Size:    Size{1200, 900},
		Layout:  VBox{MarginsZero: true},
		Children: []Widget{

			ImageView{
				AssignTo: &mw.imageViewer,
				MaxSize: Size{800, 600},
				MinSize: Size{800, 600},
				OnMouseUp: mw.UserTrain,
				OnMouseMove: mw.TestMourseMove,
			},
			TabWidget{
				AssignTo: &mw.tabWidget,
				MaxSize: Size{800, 600},
				MinSize: Size{800, 600},
				OnMouseUp: mw.UserTrain,
				OnMouseMove: mw.TestMourseMove,
			},

			TextEdit{
				AssignTo: &mw.textEditor,
				ReadOnly: true,
				Text:     fmt.Sprintf(""),
				HScroll: 	true,
				VScroll: true,
			},
		},

		OnMouseUp: mw.UserTrain,
		OnMouseDown: mw.TestMourseDown,

	}.Create()); err != nil {
		log.Fatal(err)
	}

	walk.InitWrapperWindow(mw)
	mw.Run()
}

type Point struct {
	x,y int
}

type DrawInfo struct {
	imgData []byte
	imgIdent []byte
}

type MyMainWindow struct {
	*walk.MainWindow
	tabWidget        *walk.TabWidget
	imageViewer	 *walk.ImageView
	textEditor       *walk.TextEdit
	prevFilePath     string

	trainDBId        uint8
	waitForStart     chan bool
	waitForDBVisitor chan bool

	toDraw           DrawInfo

	//------------
	cliked           []Point
}


const WM_MY_MSG = 1025
func (mw *MyMainWindow) WndProc(hwnd win.HWND, msg uint32, wParam, lParam uintptr) uintptr {
	switch msg {
	case WM_MY_MSG:
		mw.addTrainInfo("to draw")
		mw.drawInMainThread()
	}

	return mw.FormBase.WndProc(hwnd, msg, wParam, lParam)
}




func (this *MyMainWindow) TestMourseDown(x, y int, button walk.MouseButton)  {
	this.addTrainInfo("mouse down")
}
func (this *MyMainWindow) TestMourseMove(x, y int, button walk.MouseButton)  {
	//this.addTrainInfo("mouse move")
}

//界面线程
func (this *MyMainWindow) UserTrain(x, y int, button walk.MouseButton)  {

	var whichMouse string
	switch button {
	case walk.LeftButton:
		whichMouse = "left"
		break;
	case walk.RightButton:
		whichMouse = "right"
		break;
	case walk.MiddleButton:
		whichMouse = "middle"
		break;
	default:
		whichMouse = "unknow mourse"
		break;
	}

	userSay := "user train, " + whichMouse +  ", x: " + strconv.Itoa(x) + ", y: " + strconv.Itoa(y)
	fmt.Println(userSay)
	this.addTrainInfo(userSay)
	//end for train
	if button == walk.RightButton{
		this.cliked = nil
		this.addTrainInfo("finihsed trained")
		//继续读取数据库
		this.waitForDBVisitor <- true
	}else{
		//training
		click := Point{x:x,y:y}
		this.cliked = append(this.cliked, click)
	}
}

func (this *MyMainWindow) addTrainInfo(text string)  {
	exsits := this.textEditor.Text()
	exsits += "\r\n" + text
	this.textEditor.SetText(exsits)
}


func waitAndVisitDB(this *MyMainWindow)  {
	//wait for start
	if ! <- this.waitForStart{
		return
	}

	imgDB := dbOptions.PickImgDB(this.trainDBId)
	iter := imgDB.DBPtr.NewIterator(nil, &imgDB.ReadOptions)
	iter.First()
	if !iter.Valid(){
		fmt.Println("no data to train")
		return
	}

	imgIdent := make([]byte, ImgIndex.IMG_IDENT_LENGTH)
	imgIdent[0] = this.trainDBId

	for iter.Valid(){
		if config.IsValidUserDBKey(iter.Key()){
			copy(imgIdent[1:], iter.Key())
			this.toDraw.imgIdent = imgIdent
			this.toDraw.imgData = iter.Value()

			//通知 UI 线程进行 draw
			fmt.Println("to notify GUI to draw")
			this.SendMessage(WM_MY_MSG,0,0)
			//等待指令, 读取数据库
			<- this.waitForDBVisitor
		}



		iter.Next()
	}
}

func (this *MyMainWindow) drawInMainThread()  {
	drawInfo := this.toDraw

	imgName := strconv.Itoa(int(drawInfo.imgIdent[0])) + "_" + string(ImgIndex.ParseImgKeyToPlainTxt(drawInfo.imgIdent[1:]))

	this.drawImage(drawInfo.imgData, imgName)

	this.addTrainInfo("wait for user train")
}

func (mw *MyMainWindow) drawImage(imgData []byte, title string) error {
	var reader io.Reader = bytes.NewReader(imgData)
	img, err := jpeg.Decode(reader)
	if err != nil {
		return err
	}

	dst := image.NewRGBA(image.Rect(0, 0, 600,600))
	if nil != graphics.Scale(dst, img){
		return err
	}

	walkImage, err := walk.NewBitmapFromImage(dst);
	if err != nil{
		return err
	}

	var succeeded bool
	defer func() {
		if !succeeded {
			walkImage.Dispose()
		}
	}()

	page, err := walk.NewTabPage()
	if err != nil {
		return err
	}

	if page.SetTitle(title); err != nil {
		return err
	}
	page.SetLayout(walk.NewHBoxLayout())
	page.MouseDown()

	defer func() {
		if !succeeded {
			page.Dispose()
		}
	}()

	imageView, err := walk.NewImageView(page)
	if err != nil {
		return err
	}

	defer func() {
		if !succeeded {
			imageView.Dispose()
		}
	}()

	imageView.SetEnabled(true)
	if err := imageView.SetImage(walkImage); err != nil {
		return err
	}

	if mw.tabWidget.Pages().Len() > 8{
		mw.tabWidget.Pages().RemoveAt(0)
	}

	if err := mw.tabWidget.Pages().Add(page); err != nil {
		return err
	}

	if err := mw.tabWidget.SetCurrentIndex(mw.tabWidget.Pages().Len() - 1); err != nil {
		return err
	}

	succeeded = true

	return nil
}



func (this *MyMainWindow) openAction_Triggered() {
	this.waitForStart <- true
	this.addTrainInfo("gui set to start")
}

func (mw *MyMainWindow) aboutAction_Triggered() {
	walk.MsgBox(mw, "About", "Walk Image Viewer Example", walk.MsgBoxIconInformation)
}