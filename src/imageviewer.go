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
	"imgCache"
	"util"
	"strings"
	"sort"
)

func main() {

	points := config.GetClipConfigById(0).GetClipsLeftTop()
	fmt.Println("leftTopPoints: ", points)

	mw := new(MyMainWindow)
	mw.waitForStart = make(chan bool, 1)
	mw.waitForDBVisitor = make(chan bool, 1)
	mw.imgHeight = 600
	mw.imgWidth = 600
	mw.wndWidth = 1200
	mw.wndHeight = 900
	mw.pickedWhich = make(map[uint8]int)
	mw.trainResult = imgCache.NewMyMap(false)
	mw.model = NewEnvModel()
	mw.toDrawAgain = nil

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
		MaxSize: Size{mw.wndWidth, mw.wndHeight},
		Size:    Size{mw.wndWidth, mw.wndHeight},
		Layout:  HBox{},
		Children: []Widget{
			Composite{
				Layout:VBox{},
				Children: []Widget{
					ListBox{
						Font:Font{PointSize:14},
						AssignTo: &mw.listBox,
						MinSize:Size{400,300},
						MaxSize:Size{400,300},
					//	OnCurrentIndexChanged: mw.lb_CurrentIndexChanged,
						OnSelectedIndexesChanged:mw.lb_SelectedChanged,
					},
					PushButton{
						AssignTo: &mw.imgPreViewer,
						MinSize:Size{400,400},
						MaxSize:Size{400,400},
						AlwaysConsumeSpace:true,

					},

					Label{
						MinSize:Size{400,60},
						MaxSize:Size{400,60},
						Text: "-----------------------------",
					},
					TextEdit{
						AssignTo: &mw.listTestEditor,
						ReadOnly: true,
						Text:     fmt.Sprintf(""),
						HScroll: false,
						VScroll: false,
						MinSize:Size{400,40},
						MaxSize:Size{400,40},
						Font:Font{PointSize:14},
					},
					PushButton{
						AssignTo:&mw.doAgainButton,
						Text:"Again",
						MinSize:Size{400,100},
						MaxSize:Size{400,100},
						Visible:true,
						OnMouseUp: func(x, y int, button walk.MouseButton){
							item := mw.GetCurrentListBoxItemData()
							imgIdent := GetImgIdentFromImgName(item.name)
							mw.toDrawAgain = &DrawInfo{imgIdent:imgIdent, imgData:item.data}
							//清除信息
							mw.pickedWhich = make(map[uint8]int)
							mw.TrainAgain()
						},
					},
				},
			},

			TextEdit{
				AssignTo: &mw.textEditor,
				ReadOnly: true,
				Text:     fmt.Sprintf(""),
				HScroll: 	true,
				VScroll: true,
				MinSize:Size{400,900},
				MaxSize:Size{400,900},
				Font:Font{PointSize:14},
				OnMouseDown:func(x,y int, button walk.MouseButton){
					//fmt.Println("textedit mouse down: ", int(button))
				},
			},
			ImageView{
				AssignTo: &mw.imageViewer,
				MinSize:Size{600,800},
				MaxSize:Size{600,800},
				AlwaysConsumeSpace:true,

			},
		},
	}.Create()); err != nil {
		log.Fatal(err)
	}

	walk.InitWrapperWindow(mw)

	mw.imageViewer.MouseDown().Attach(func(x, y int, button walk.MouseButton) {
		mw.onImgClickedEvent(x,y, button)
	})
/*
	if myImgViewer, err := NewMyImgViewer(mw); nil != err{
		log.Fatal(err)
	}else{
		myImgViewer.SetName("image viewer")
		mw.imageViewer = myImgViewer
	}
*/
	mw.Run()
}

func (this *MyMainWindow) TrainAgain()  {
	if nil == this.toDrawAgain{
		return
	}
	imgIdent := this.toDrawAgain.imgIdent

	imgData := this.toDrawAgain.imgData //dbOptions.PickImgDB(dbId).ReadFor(imgKey)
	if 0 == len(imgData){
		walk.MsgBox(this, "Value", "重做失败, 查询图片数据失败", walk.MsgBoxIconInformation)
		return
	}

	imgName := GetImgNamgeFromImgIdent(imgIdent)

	drawImage(this.imageViewer, this.imgWidth, this.imgHeight, imgData, imgName)

	this.appendToTextEditor("\r\n----------------------------\r\n")
	this.appendToTextEditor("confirming: [" + imgName + "]: ")
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
	//imageViewer	 *MyImgViewer
	imageViewer	 *walk.ImageView
	textEditor       *walk.TextEdit
	listBox		*walk.ListBox
	imgPreViewer	 *walk.PushButton
	model *EnvModel
	listTestEditor  *walk.TextEdit
	doAgainButton  *walk.PushButton

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
	toDrawAgain	*DrawInfo
	pickedWhich	map[uint8]int
	trainResult	*imgCache.MyMap
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

	var imgIdent []byte
	if nil != this.toDrawAgain{
		imgIdent = this.toDrawAgain.imgIdent
	}else{
		imgIdent = this.toDraw.imgIdent
	}

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

		ans := make([]uint8, len(this.pickedWhich))
		ci := 0
		for which,_ := range this.pickedWhich{
			this.appendToTextEditor(strconv.Itoa(int(which)) + " ")
			ans[ci] = which
			ci ++
			//to save result
		}
		this.trainResult.Put(imgIdent, ans)

		this.ReInitListBox()
		this.pickedWhich = make(map[uint8]int)

		//当有重做的任务正在进行, 需要把当前做完后才做已经缓存起来的任务
		if this.toDrawAgain == nil{
			//继续读取数据库
			this.waitForDBVisitor <- true
		}else{
			//继续把原有任务做完
			this.toDrawAgain = nil
			this.drawInMainThread()
		}

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
			this.toDraw.imgIdent = fileUtil.CopyBytesTo(imgIdent)
			this.toDraw.imgData = fileUtil.CopyBytesTo(iter.Value())

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

	drawImage(this.imageViewer, this.imgWidth, this.imgHeight, drawInfo.imgData, imgName)

	this.appendToTextEditor("\r\n----------------------------\r\n")
	this.appendToTextEditor("confirming: [" + imgName + "]: ")
}

func drawImage(imgViewer * walk.ImageView, width, height int, imgData []byte, title string) error {
	var reader io.Reader = bytes.NewReader(imgData)
	img, err := jpeg.Decode(reader)
	if err != nil {
		return err
	}

	dst := image.NewRGBA(image.Rect(0, 0, width, height))
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


	imgViewer.SetEnabled(true)
	if err := imgViewer.SetImage(walkImage); err != nil {
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

func (mw *MyMainWindow) GetCurrentListBoxItemData() *EnvItem {
	i := mw.listBox.CurrentIndex()
	if i < 0 || i >= len(mw.model.items){
		return nil
	}
	return &mw.model.items[i]
}

func (mw *MyMainWindow) lb_SelectedChanged() {
	i := mw.listBox.CurrentIndex()
	if i < 0 || i >= len(mw.model.items){
		return
	}
	fmt.Println("current index: ", i, ", items length: ", len(mw.model.items))
	item := &mw.model.items[i]
	mw.doAgainButton.SetText("重做: " + item.name)
	mw.doAgainButton.SetName(item.name)
	mw.listTestEditor.SetText(item.value)

	if 0 == len(item.data){
		imgIdent := GetImgIdentFromImgName(item.name)
		imgData := dbOptions.PickImgDB(uint8(imgIdent[0])).ReadFor(imgIdent[1:])
		if nil == imgData{
			walk.MsgBox(mw, "Value", "查询图片数据失败", walk.MsgBoxIconInformation)
		}
		item.data = imgData
	}

	var reader io.Reader = bytes.NewReader(item.data)
	img, err := jpeg.Decode(reader)
	if err != nil {
		return
	}

	dst := image.NewRGBA(image.Rect(0, 0, 300, 300))
	if nil != graphics.Scale(dst, img){
		return
	}

	walkImage, err := walk.NewBitmapFromImage(dst);

	mw.imgPreViewer.SetImage(walkImage)
	mw.imgPreViewer.SetEnabled(true)
}


func (this *MyMainWindow) ReInitListBox()  {
	res := this.trainResult
	keys := res.KeySet()

	this.model.items = make([]EnvItem, len(keys))

	index := 0
	for _,key := range keys{
		values := res.Get(key)
		if 0 == len(values){
			continue
		}
		whiches := values[0].([]uint8)
		imgName := strconv.Itoa(int(key[0]))+"_"+string(ImgIndex.ParseImgKeyToPlainTxt(key[1:]))
		whichStr := ""
		for _,which := range whiches{
			whichStr += strconv.Itoa(int(which))
		}
		this.model.items[index] = EnvItem{imgName, whichStr, nil}
		index ++
	}

	sort.Sort(envItemList(this.model.items))
	this.listBox.SetModel(this.model)
}

type EnvItem struct {
	name  string
	value string
	data []byte
}

func (this envItemList)Len() int {
	return len(this)
}

func (this envItemList) Swap(i, j int) {
	this[i], this[j] = this[j], this[i]
}

func (this envItemList) Less(i, j int) bool {
	return strings.Compare(this[i].name,this[j].name) < 0
}


type envItemList []EnvItem
type EnvModel struct {
	walk.ListModelBase
	items []EnvItem
}

func NewEnvModel() *EnvModel {

	m := &EnvModel{items: nil}

	return m
}

func (m *EnvModel) ItemCount() int {
	return len(m.items)
}

func (m *EnvModel) Value(index int) interface{} {
	return m.items[index].name
}

func GetImgNamgeFromImgIdent (imgIdent []byte) string {
	return strconv.Itoa(int(imgIdent[0])) + "_" + string(ImgIndex.ParseImgKeyToPlainTxt(imgIdent[1:]))
}

func GetImgIdentFromImgName(imgName string) []byte {
	res := strings.Split(imgName, "_")
	ret := make([]byte, ImgIndex.IMG_IDENT_LENGTH)
	dbId, _ := strconv.Atoi(res[0])
	imgKey := ImgIndex.FormatImgKey([]byte(res[1]))
	ret[0] = byte(dbId)
	copy(ret[1:], imgKey)
	return ret

}