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
	"strings"
)

func main() {

	points := config.GetClipConfigById(0).GetClipsLeftTop()
	fmt.Println("leftTopPoints: ", points)

	mw := new(MyMainWindow)
	mw.waitForStart = make(chan bool, 1)
	mw.waitForDBVisitor = make(chan bool, 1)
	mw.imgHeight = 600
	mw.imgWidth = 600
	mw.wndWidth = 1600
	mw.wndHeight = 900
	mw.pickedWhich = make(map[uint8]int)
	mw.model = NewEnvModel()

	go asyncVisitDB(mw)

	windowsCreateAndRun(mw)
}

func windowsCreateAndRun(mw *MyMainWindow){
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
				Text: "&Start",
				Items: []MenuItem{
					Action{
						AssignTo:    &openAction,
						Text:        "&Start",
						Image:       "../img/open.png",
						OnTriggered: mw.openAction_Triggered,
					},
				},
			},
			Menu{
				Text: "&Exit",
				Items: []MenuItem{
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
		MinSize: Size{mw.wndWidth, mw.wndHeight},
		Size:    Size{mw.wndWidth, mw.wndHeight},
		Layout:  HBox{MarginsZero: true},
		Children: []Widget{
			ListBox{
				AssignTo: &mw.listBox,
				Model:    mw.model,
				OnCurrentIndexChanged: mw.lb_CurrentIndexChanged,
				OnItemActivated:       mw.lb_ItemActivated,
			},
			TextEdit{
				AssignTo: &mw.textEditor,
				ReadOnly: true,
				Text:     fmt.Sprintf(""),
				HScroll: 	true,
				VScroll: true,
				MinSize:Size{600,300},
				MaxSize:Size{600,300},
				Font:Font{PointSize:14},
				OnMouseDown:func(x,y int, button walk.MouseButton){
					//fmt.Println("textedit mouse down: ", int(button))
				},
			},
		},
	}.Create()); err != nil {
		log.Fatal(err)
	}

	walk.InitWrapperWindow(mw)

	if myImgViewer, err := NewMyImgViewer(mw); nil != err{
		log.Fatal(err)
	}else{
		myImgViewer.SetName("image viewer")
		mw.imageViewer = myImgViewer
	}

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
	imageViewer	 *MyImgViewer
	textEditor       *walk.TextEdit
	listBox		*walk.ListBox
	model 		*EnvModel
	prevFilePath     string

	trainDBId        uint8
	waitForStart     chan bool
	waitForDBVisitor chan bool

	toDraw           DrawInfo

	//------------ const
	wndHeight	int
	wndWidth	int
	imgHeight	int
	imgWidth	int

	//------------
	//cliked           []Point
	pickedWhich	map[uint8]int
}


const WM_MY_MSG = 1025
func (mw *MyMainWindow) WndProc(hwnd win.HWND, msg uint32, wParam, lParam uintptr) uintptr {
	switch msg {
	case WM_MY_MSG:
		mw.drawInMainThread()
	}

	return mw.FormBase.WndProc(hwnd, msg, wParam, lParam)
}


func (this *MyMainWindow) TestMourseDown(x, y int, button walk.MouseButton)  {

}
func (this *MyMainWindow) TestMourseMove(x, y int, button walk.MouseButton)  {
	//this.addTrainInfo("mouse move")
}

//界面线程
func (this *MyMainWindow) onImgClickedEvent(x, y int, button walk.MouseButton)  {

	imgIdent := this.toDraw.imgIdent
	dbIdStr := strconv.Itoa(int(imgIdent[0]))
	imgKeyStr := string(ImgIndex.ParseImgKeyToPlainTxt(imgIdent[1:]))
	imgName := dbIdStr + "_" + imgKeyStr

	//confire
	if button == walk.RightButton{
		if 0 == len(this.pickedWhich){
			//this.appendToTextEditor("abort ---- \r\n")
			return
		}

		this.appendToTextEditor("\r\nconfirmed:  [" + imgName + "]: ")

		for which,_ := range this.pickedWhich{
			this.appendToTextEditor(strconv.Itoa(int(which)) + " ")

			//to save result
		}

		this.pickedWhich = make(map[uint8]int)
		//继续读取数据库
		this.waitForDBVisitor <- true
	}else if button == walk.LeftButton{
		which := this.whichClip(x, y)
		if 255==which{
			//invalid pick
			return
		}
		this.pickedWhich[which]=1
		this.appendToTextEditor(strconv.Itoa(int(which)) + " ")
	}else if button == walk.MiddleButton{
		//remove
		noneConfirm := this.whichClip(x, y)
		if 255==noneConfirm{
			//invalid pick
			return
		}
		delete(this.pickedWhich, noneConfirm)
		this.appendToTextEditor("\r\nconfirming: [" + imgName + "]: ")

		for w, _ := range this.pickedWhich{
			if w!=noneConfirm{
				this.appendToTextEditor(strconv.Itoa(int(w)) + " ")
			}
		}

	}else{

	}
}

func (this *MyMainWindow)whichClip(x, y int) uint8 {
	clipConfig := config.GetClipConfigById(0)
	toClipX := x * clipConfig.BigPicWidth / this.imgWidth
	toclipY := y * clipConfig.BigPicHeight / this.imgHeight
	return clipConfig.WhichClip(toClipX, toclipY)
}

func (this *MyMainWindow) appendToTextEditor(text string)  {
	/*exsits := this.textEditor.Text()
	exsits += "\r\n" + text
	this.textEditor.SetText(exsits)
	*/
	//自动滚动到最下面
	this.textEditor.AppendText(text)
}


func asyncVisitDB(this *MyMainWindow)  {
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

	this.appendToTextEditor("\r\n----------------------------\r\n")
	this.appendToTextEditor("confirming: [" + imgName + "]: ")
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


	mw.imageViewer.SetEnabled(true)
	if err := mw.imageViewer.SetImage(walkImage); err != nil {
		return err
	}

	succeeded = true

	return nil
}



func (this *MyMainWindow) openAction_Triggered() {
	this.waitForStart <- true
}

func (mw *MyMainWindow) aboutAction_Triggered() {
	walk.MsgBox(mw, "About", "Walk Image Viewer Example", walk.MsgBoxIconInformation)
}



type MyImgViewer struct {
	*walk.ImageView
}


func NewMyImgViewer(parent *MyMainWindow) (*MyImgViewer, error) {
	imgView, err := walk.NewImageView(parent)
	if nil != err{
		return nil, err
	}
	myImgView := &MyImgViewer{imgView}

	//这里有坑: 如果调用下面这一行, 则 MyImgViewer 的 wndProc 函数才会被注册.
	//而如果调用则会造成重复调用: NewImageView 中已经调用过. 这会造成控件被破坏, 显示不了图片

	// InitWrapperWindow has been called in walk.NewImageView(parent)
	// if call again, imgView can not dispaly images(i guess init again course this)
	// but if not call this line, MyImgViewer.WndProc will not work(i guess the event have not been registered)
	//walk.InitWrapperWindow(myImgView)

	imgView.MouseDown().Attach(func(x, y int, button walk.MouseButton) {
		parent.onImgClickedEvent(x,y, button)
	})

	return myImgView, nil
}

func (*MyImgViewer) MinSizeHint() walk.Size {
	return walk.Size{800, 600}
}
func (*MyImgViewer) MaxSizeHint() walk.Size {
	return walk.Size{800, 600}
}

func (this *MyImgViewer)WndProc(hwnd win.HWND, msg uint32, wParam, lParam uintptr) uintptr {
	switch msg {
	case win.WM_LBUTTONDOWN:
		log.Printf("WM_LBUTTONDOWN")
		break
	case win.WM_RBUTTONDOWN:
		log.Printf("WM_RBUTTONDOWN")
		break
	case win.WM_MBUTTONDOWN:
		log.Printf("WM_MBUTTONDOWN")
		break
	}

	return this.WindowBase.WndProc(hwnd, msg, wParam, lParam)
}


func (mw *MyMainWindow) lb_CurrentIndexChanged() {
	i := mw.listBox.CurrentIndex()
	item := &mw.model.items[i]

//	mw.te.SetText(item.value)

	fmt.Println("CurrentIndex: ", i)
	fmt.Println("CurrentEnvVarName: ", item.name)
}

func (mw *MyMainWindow) lb_ItemActivated() {
	value := mw.model.items[mw.listBox.CurrentIndex()].value

	walk.MsgBox(mw, "Value", value, walk.MsgBoxIconInformation)
}

type EnvItem struct {
	name  string
	value string
}
type EnvModel struct {
	walk.ListModelBase
	items []EnvItem
}

func NewEnvModel() *EnvModel {
	env := os.Environ()

	m := &EnvModel{items: make([]EnvItem, len(env))}

	for i, e := range env {
		j := strings.Index(e, "=")
		if j == 0 {
			continue
		}

		name := e[0:j]
		value := strings.Replace(e[j+1:], ";", "\r\n", -1)

		m.items[i] = EnvItem{name, value}
	}

	return m
}

func (m *EnvModel) ItemCount() int {
	return len(m.items)
}

func (m *EnvModel) Value(index int) interface{} {
	return m.items[index].name
}
