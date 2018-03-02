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
	"imgSearch/src/dbOptions"
	"imgSearch/src/imgIndex"
	"strconv"
	"github.com/BurntSushi/graphics-go/graphics"
	"image"
	"imgSearch/src/config"
	"github.com/lxn/win"
	"imgSearch/src/util"
	"strings"
	"sort"
	"time"
	"imgTrain/src/dialogs"
	"unicode"
	"imgCache"
)

func main() {

	points := config.GetClipConfigById(0).GetClipsLeftTop()
	fmt.Println("leftTopPoints: ", points)

	mw := new(MyMainWindow)
	mw.waitForStart = make(chan bool, 1)
	mw.hasStarted = false
	mw.waitForDBVisitor = make(chan bool, 1)
	mw.imgHeight = 600
	mw.imgWidth = 600
	mw.wndWidth = 1366
	mw.wndHeight = 768
	mw.pickedWhiches = make(map[uint8]int)
	mw.pickedTagName = nil
	mw.trainResult = imgCache.NewMyMap(false)
	mw.model = NewNameValueModel()
	mw.toDrawAgain = nil
	mw.waitToFlushTagComobobox = make(chan bool, 1)
	mw.tagComboboxClocker = nil
	mw.reloadTagInfos()
	mw.markNotSameTopicLog = imgCache.NewMyMap(false)

	go asyncVisitDB(mw)

	go autoFlushTagCombox(mw)

	windowsCreateAndRun(mw)
}

func windowsCreateAndRun(mw *MyMainWindow){

	//选择 img dbid
	mw.trainDBId = showAndPickImgDBId()

	if err := (MainWindow{
		AssignTo: &mw.MainWindow,
		Title:    "李志浩图片识别训练器",
		MenuItems: []MenuItem{
			Menu{
				Text: "&Exit",
				Items: []MenuItem{
					Action{
						Text:        "Exit",
						OnTriggered: func() {
							mw.appendToTextEditor("用户退出, 保存训练结果")
							mw.flushTrainRes()
							mw.Close()
						},
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
				Layout:VBox{MarginsZero:true},
				Children: []Widget{
					ListBox{
						Font:Font{PointSize:14},
						AssignTo: &mw.listBox,
						MinSize:Size{400,200},
						MaxSize:Size{400,200},
						OnSelectedIndexesChanged:mw.lb_SelectedChanged,
					},
					CustomWidget{
						MinSize:Size{400,400},
						MaxSize:Size{400,400},
						AssignTo:            &mw.imgPreViewer,
						ClearsBackground:    true,
						InvalidatesOnResize: true,
						Paint:               mw.drawPreViewImage,
					},
					PushButton{
						AssignTo:&mw.doAgainButton,
						Text:"Again",
						MinSize:Size{400,60},
						MaxSize:Size{400,60},
						Visible:true,
						Font:Font{PointSize:14},
						OnMouseUp: func(x, y int, button walk.MouseButton){
							if !mw.hasStarted{
								return
							}
							item := mw.GetCurrentListBoxItemData()
							if nil == item{
								return
							}
							if strings.Compare(mw.doAgainButton.Text(), "没有可重做任务") == 0{
								return
							}
							mw.toDrawAgain = &DrawInfo{imgIdent:item.imgIdent, imgData:item.imgData}
							//清除信息
							mw.pickedWhiches = make(map[uint8]int)
							mw.pickedTagName = nil
							mw.TrainAgain()
						},
					},
					Label{
						MinSize:Size{400,40},
						MaxSize:Size{400,40},
						Text: "-----------------------------",
					},
					PushButton{
						AssignTo:&mw.flushAllButton,
						Text:"保存结果",
						MinSize:Size{400,68},
						MaxSize:Size{400,68},
						Visible:true,
						Font:Font{PointSize:14},
						OnMouseUp: func(x, y int, button walk.MouseButton){
							if !mw.hasStarted{
								return
							}
							mw.flushTrainRes()
						},
					},
				},
			},

			//以下 Composite 366 * 768
			Composite{
				Layout:VBox{MarginsZero:true},
				Children: []Widget{
					PushButton{
						//AssignTo:&mw.,
						Text:"确认当前图片主题",
						MinSize:Size{366,50},
						MaxSize:Size{366,50},
						Font:Font{PointSize:14},
						Visible:true,
						OnMouseUp: func(x, y int, button walk.MouseButton){
							input := mw.tagBombobox.Text()
							if 0 == len(input){
								return
							}
							mw.pickedTagName = []byte(input)
						},
					},

					ComboBox{
						AssignTo:&mw.tagBombobox,
						MinSize:Size{366,400},
						MaxSize:Size{366,400},
						Font:Font{PointSize:14},
						Editable: true,
						OnKeyUp: mw.tagComboboxKeyUp ,

					},

					Label{
						MinSize:Size{366,20},
						MaxSize:Size{366,20},
						Font:Font{PointSize:14},
						Text: "---------------------------------------",
					},
					//以上 470
					PushButton{
						//AssignTo:&mw.,
						Text:"标记协同关系非主题相似",
						MinSize:Size{366,50},
						MaxSize:Size{366,50},
						Font:Font{PointSize:14},
						Visible:true,
						OnMouseUp: func(x, y int, button walk.MouseButton){
							if len(mw.markNotSameIdent) == 0{
								return
							}
							dialogs.ShowMarkNotSameTopickDBDlg(mw.markNotSameIdent, mw.markNotSameTopicLog)
						},
					},


					TextEdit{
						AssignTo: &mw.adviserText,
						ReadOnly: true,
						Text:     fmt.Sprintf(""),
						MinSize:Size{366,100},
						MaxSize:Size{366,100},
						Font:Font{PointSize:14},
						OnMouseDown:func(x,y int, button walk.MouseButton){

						},
					},

					Label{
						MinSize:Size{366,20},
						MaxSize:Size{366,20},
						Font:Font{PointSize:14},
						Text: "---------------------------------------",
					},

					//以上 640
					//以下 100
					PushButton{
						Text:"新加入一个标签",
						MinSize:Size{366,50},
						MaxSize:Size{366,50},
						Visible:true,
						Font:Font{PointSize:14},
						OnMouseUp: func(x, y int, button walk.MouseButton){
							inputTagName := mw.newTagTextEditor.Text()
							inputTagName = myStringTrim(inputTagName)
							fmt.Println(inputTagName)

							if 0 == len(inputTagName){
								return
							}
							if nil != mw.cachedTotalTagInfoList {
								if mw.cachedTotalTagInfoList.IsTagExsits([]byte(inputTagName)){
									walk.MsgBox(mw, "Value", "tag 已存在: " + inputTagName, walk.MsgBoxIconInformation)
									return
								}
							}

							err := dbOptions.WriteATag([]byte(inputTagName))
							if nil != err{
								walk.MsgBox(mw, "Value", "写入 tag 错误: " + err.Error(), walk.MsgBoxIconInformation)
							}else{
								//	walk.MsgBox(mw, "Value", "写入 tag 成功: " + inputTagName, walk.MsgBoxIconInformation)
								mw.reloadTagInfos()
								mw.ReInitTagCombobox(mw.cachedTotalTagInfoList)

								mw.tagBombobox.SetText(string(inputTagName))

								mw.newTagTextEditor.SetText("")
							}
						},
					},


					TextEdit{
						AssignTo: &mw.newTagTextEditor,
						ReadOnly: false,
						Text:     fmt.Sprintf(""),
						MinSize:Size{366,50},
						MaxSize:Size{366,50},
						Font:Font{PointSize:14},
						OnMouseDown:func(x,y int, button walk.MouseButton){

						},
					},

				},
			},

			Composite{
				Layout:VBox{MarginsZero:true},
				Children: []Widget{
					PushButton{
						AssignTo:&mw.skipCurrentButton,
						Text:"Skip",
						MinSize:Size{600,50},
						MaxSize:Size{600,50},
						Font:Font{PointSize:14},
						Visible:true,
						OnMouseUp: func(x, y int, button walk.MouseButton){
							if !mw.hasStarted{
								return
							}
							//清除信息
							mw.pickedWhiches = make(map[uint8]int)
							mw.pickedTagName = nil
							//跳过当前
							mw.waitForDBVisitor <- true
						},
					},
					ImageView{
						AssignTo: &mw.imageViewer,
						MinSize:Size{600, 600},
						MaxSize:Size{600, 600},
						AlwaysConsumeSpace:true,

					},
					TextEdit{
						AssignTo: &mw.dispalyTextEditor,
						ReadOnly: true,
						Text:     fmt.Sprintf(""),
						HScroll: 	true,
						VScroll: true,
						MinSize:Size{600,120},	//因为图片600 高度实际上不用使用 600, 此处可以略高, 以撑开所有像素
						MaxSize:Size{600,120},
						Font:Font{PointSize:14},
						OnMouseDown:func(x,y int, button walk.MouseButton){
							//fmt.Println("textedit mouse down: ", int(button))
						},
					},
				},
			},
		},
	}.Create()); err != nil {
		log.Fatal(err)
	}

	walk.InitWrapperWindow(mw)

	mw.imageViewer.MouseDown().Attach(func(x, y int, button walk.MouseButton) {
		mw.onImgClickedEvent(x,y, button)
	})


	mw.hasStarted = true
	mw.waitForStart <- mw.hasStarted

	mw.ReInitTagCombobox(mw.cachedTotalTagInfoList)

	mw.Run()
}

func (this *MyMainWindow) reloadTagInfos()  {
	this.cachedTotalTagInfoList = dbOptions.GetAllTagInfos()
	this.cachedTotalTagInfoList.Print()
}

func autoFlushTagCombox(this* MyMainWindow)  {
	//等待开始
	<- this.waitToFlushTagComobobox
	for{
		<- this.tagComboboxClocker.C	//等待计时到达

		input := this.tagBombobox.Text()
		editInput := trimLRSpace([]byte(input))
		if 0 == len(editInput){
			this.ReInitTagCombobox(this.cachedTotalTagInfoList)
		}else{
			fndTags := this.cachedTotalTagInfoList.FindByNameOrPinyin([]byte(editInput))
			this.ReInitTagCombobox(fndTags)
		}
	}
}


var hasClicked = false
func (this *MyMainWindow) tagComboboxKeyUp (key walk.Key)  {
	//如果在 1 秒内再次输入则重置计时器, 再等待 1 s
	//如果一秒后用户输入则刷新 tag bombobox
	//timer 只会触发一次

	if nil == this.tagComboboxClocker{
		this.tagComboboxClocker = time.NewTimer(time.Second)

	}else{

		this.tagComboboxClocker.Reset(time.Second)
	}

	if !hasClicked{
		hasClicked = true
		this.waitToFlushTagComobobox <- true
	}

}

func (this *MyMainWindow) ReInitTagCombobox(tags []dbOptions.TagInfo)  {

	this.tagBombobox.SetHeight(200)
	this.tagBombobox.SetWidth(40)

	if len(tags) == 0{
		this.tagBombobox.SetModel([]string{""})
		return
	}

	model := make([]string, len(tags))
	for i,tag := range tags{
		model[i] = string(tag.TagName)
	}

	this.tagBombobox.SetModel(model)
	if 1 == len(tags){
		this.tagBombobox.SetText(string(tags[0].TagName))
	}
	this.tagBombobox.SetFocus()
}

func (this *MyMainWindow) flushTrainRes()  {
	//保存
	err := dbOptions.ImgRrainResultsBatchSave(this.trainDBId, this.trainResult)
	if nil != err{
		walk.MsgBox(this, "Value", "保存训练结果失败: " + err.Error(), walk.MsgBoxIconInformation)
		return
	}

	//清除重做信息
	this.doAgainButton.SetText("没有可重做任务")
	this.doAgainButton.SetName("")
	this.toDrawAgain = nil

	//清空结果
	this.trainResult.Clear()

	//清空 listbox
	this.ReInitListBox()

	//显示当前正在处理的图的预览
	this.imgPreViewer.Invalidate()
	this.imgPreViewer.SetEnabled(true)

	walk.MsgBox(this, "Value", "保存成功", walk.MsgBoxIconInformation)
}

func fromImgDataToWalkImg(imgData []byte) (*walk.Bitmap, error) {
	var reader io.Reader = bytes.NewReader(imgData)
	img, err := jpeg.Decode(reader)
	if err != nil {
		return nil, err
	}

	dst := image.NewRGBA(image.Rect(0, 0, 400, 400))
	if nil != graphics.Scale(dst, img){
		return nil, err
	}

	return walk.NewBitmapFromImage(dst)
}

func (this *MyMainWindow) drawPreViewImage(canvas *walk.Canvas, updateBounds walk.Rectangle) error {
	item := this.GetCurrentListBoxItemData()
	if nil == item{
		return nil
	}

	if nil == item.imgData{
		item.imgData = dbOptions.PickImgDB(uint8(item.imgIdent[0])).ReadFor(item.imgIdent[1:])
		if nil == item.imgData{
			walk.MsgBox(this, "Value", "查询图片数据失败", walk.MsgBoxIconInformation)
			return nil
		}
	}

	walkImage, err := fromImgDataToWalkImg(item.imgData)
	if err != nil{
		return err
	}

	canvas.DrawImage(walkImage,walk.Point{0,0})

	return nil
}


func (this *MyMainWindow) TrainAgain()  {
	if nil == this.toDrawAgain{
		return
	}
	imgIdent := this.toDrawAgain.imgIdent

	imgData := this.toDrawAgain.imgData
	if 0 == len(imgData){
		walk.MsgBox(this, "Value", "重做失败, 查询图片数据失败", walk.MsgBoxIconInformation)
		return
	}

	imgName := GetImgNamgeFromImgIdent(imgIdent)

	drawImage(this.imageViewer, this.imgWidth, this.imgHeight, imgData, imgName)

	this.putTrainData(this.toDrawAgain)
	this.markNotSameIdent = this.toDrawAgain.imgIdent
	this.skipCurrentButton.SetText("跳过当前: " + imgName)
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
	imageViewer             *walk.ImageView
	tagInputTextEditor      *walk.TextEdit
	dispalyTextEditor       *walk.TextEdit
	listBox                 *walk.ListBox
	imgPreViewer            *walk.CustomWidget
	model                   *NameValueModel
				       //	listTestEditor  *walk.TextEdit
	doAgainButton           *walk.PushButton
	flushAllButton          *walk.PushButton

	markNotSameIdent	[]byte
	markNotSameTopicLog     *imgCache.MyMap

	skipCurrentButton       *walk.PushButton

	adviserText             *walk.TextEdit

	tagBombobox             *walk.ComboBox
	newTagTextEditor        *walk.TextEdit

	trainDBId               uint8
	waitForStart            chan bool
	hasStarted              bool
	waitForDBVisitor        chan bool

	//timeOfLastKeyUpOnTagComboBox int64
	tagComboboxClocker      *time.Timer
	waitToFlushTagComobobox chan bool


	toDraw                  DrawInfo

				       //------------ const
	wndHeight               int
	wndWidth                int
	imgHeight               int
	imgWidth                int

				       //------------
				       //cliked           []Point
	toDrawAgain             *DrawInfo
	pickedWhiches           map[uint8]int //键是 picked-which 用来限制同一个 which 只能被选一次
	pickedTagName           []byte
	trainResult             *imgCache.MyMap

	tagComboboxModel        dbOptions.TagInfoList
	cachedTotalTagInfoList  dbOptions.TagInfoList
}


const WM_USER_TO_TRAIN = 1025
const WM_USER_TO_STOP = 1026
func (mw *MyMainWindow) WndProc(hwnd win.HWND, msg uint32, wParam, lParam uintptr) uintptr {
	switch msg {
	case WM_USER_TO_TRAIN:
		mw.drawInMainThread()
		break
	case WM_USER_TO_STOP:
		mw.Close()
		break
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

	//confirm
	if button == walk.RightButton{
		if 0 == len(this.pickedWhiches){
			//this.appendToTextEditor("abort ---- \r\n")
			return
		}

		this.appendToTextEditor("\r\nconfirmed:  [" + imgName + "]: ")

		ans := make([]uint8, len(this.pickedWhiches))
		ci := 0
		for which,_ := range this.pickedWhiches {
			this.appendToTextEditor(strconv.Itoa(int(which)) + " ")
			ans[ci] = which
			ci ++
			//to save result
		}

		var pickedTagIndex []byte = nil
		if nil != this.pickedTagName && 0 != len(this.pickedTagName){
			if nil != this.cachedTotalTagInfoList{
				tag := this.cachedTotalTagInfoList.MustOnlyOneByName(this.pickedTagName)
				if tag != nil {
					pickedTagIndex = tag.TagIndex
					this.appendToTextEditor("| tag: " + string(this.pickedTagName))
				}else{
					walk.MsgBox(this, "Value", "错误: 不存在此 tag: " + string(this.pickedTagName), walk.MsgBoxIconInformation)
				}
			}else{
				walk.MsgBox(this, "Value", "错误: 缓存的 tagNameToId 为空: ", walk.MsgBoxIconInformation)
			}
		}

		this.trainResult.Put(imgIdent, &dbOptions.TrainResultItem{Whiches:ans, TagName: fileUtil.CopyBytesTo(this.pickedTagName), TagIndex:pickedTagIndex})

		this.ReInitListBox()
		this.pickedWhiches = make(map[uint8]int)
		this.pickedTagName = nil

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
		this.pickedWhiches[which]=1
		this.appendToTextEditor(strconv.Itoa(int(which)) + " ")
	}else if button == walk.MiddleButton{
		//remove
		noneConfirm := this.whichClip(x, y)
		if 255==noneConfirm{
			//invalid pick
			return
		}
		delete(this.pickedWhiches, noneConfirm)
		this.appendToTextEditor("\r\nconfirming: [" + imgName + "]: ")

		for w, _ := range this.pickedWhiches {
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
	this.dispalyTextEditor.AppendText(text)
}


func asyncVisitDB(this *MyMainWindow)  {
	//wait for start
	if this.hasStarted || ! <- this.waitForStart{
		return
	}

	this.hasStarted = true
	iterPtr := dbOptions.GetToTrainIterator(this.trainDBId)
	if nil == iterPtr{
		walk.MsgBox(this, "Error", "程序退出中...请检查图片库是否在使用中", walk.MsgBoxIconInformation)
		this.SendMessage(WM_USER_TO_STOP,0,0)
		return
	}
	iter := *iterPtr
	if !iter.Valid(){
		fmt.Println("no data to train")
		return
	}else{
		beginKey := iter.Key()
		fmt.Println(beginKey)
	}

	imgIdent := make([]byte, ImgIndex.IMG_IDENT_LENGTH)
	imgIdent[0] = this.trainDBId

	for iter.Valid(){
		if config.IsValidUserDBKey(iter.Key()){
			copy(imgIdent[1:], iter.Key())
			this.toDraw.imgIdent = fileUtil.CopyBytesTo(imgIdent)
			this.toDraw.imgData = fileUtil.CopyBytesTo(iter.Value())

			//通知 UI 线程可以开始 draw 出图片让 user 开始训练
			this.SendMessage(WM_USER_TO_TRAIN,0,0)
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

	this.putTrainData(&drawInfo)

	this.markNotSameIdent = drawInfo.imgIdent

	this.skipCurrentButton.SetText("跳过当前: " + imgName)
	this.appendToTextEditor("\r\n----------------------------\r\n")
	this.appendToTextEditor("confirming: [" + imgName + "]: ")
}




func (this *MyMainWindow) putTrainData(drawInfo *DrawInfo)  {
	//投放建议内容
	recoginitionRes := dbOptions.ImgRecognitionByImgIdent(drawInfo.imgIdent)

	this.adviserText.SetText(recoginitionRes.ToString())
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

}

func showAndPickImgDBId() uint8 {
	recvDbId := make(chan uint8, 1)
	dialogs.ShowPickDBDlg(&recvDbId)
	return <- recvDbId
}

func (mw *MyMainWindow) aboutAction_Triggered() {
	walk.MsgBox(mw, "About", "李志浩专属图片训练器", walk.MsgBoxIconInformation)
}

func (mw *MyMainWindow) GetCurrentListBoxItemData() *NameValueItem {
	i := mw.listBox.CurrentIndex()
	//当 listbox 中没有数据时返回当前正在处理的. 这个逻辑是为了点击"保存结果"按钮后刷新原来的预览图.
	if (i < 0 || i >= len(mw.model.items)){
		if nil != mw.toDraw.imgIdent{

			imgIdent := mw.toDraw.imgIdent
			imgName := strconv.Itoa(int(imgIdent[0]))+"_"+string(ImgIndex.ParseImgKeyToPlainTxt(imgIdent[1:]))
			whichStr := ""
			dispalyName := imgName + " ----> " + whichStr
			return &NameValueItem{name:dispalyName, value:[]byte(whichStr), imgIdent:imgIdent, imgData: mw.toDraw.imgData,whiches:nil}
		}
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

	if nil == item.imgData{
		item.imgData = dbOptions.PickImgDB(uint8(item.imgIdent[0])).ReadFor(item.imgIdent[1:])
		if nil == item.imgData{
			walk.MsgBox(mw, "Value", "查询图片数据失败", walk.MsgBoxIconInformation)
			return
		}
	}

	mw.imgPreViewer.Invalidate()
	mw.imgPreViewer.SetEnabled(true)
}


func (this *MyMainWindow) ReInitListBox()  {
	res := this.trainResult
	keys := res.KeySet()

	if 0 == len(keys){
		this.model.items = []NameValueItem{}
		this.listBox.SetModel(this.model)
		return
	}

	this.model.items = make([]NameValueItem, len(keys))

	index := 0
	for _,key := range keys{
		values := res.Get(key)
		if 0 == len(values){
			continue
		}

		whiches := values[0].(*dbOptions.TrainResultItem).Whiches

		tagName := values[0].(*dbOptions.TrainResultItem).TagName


		imgName := strconv.Itoa(int(key[0]))+"_"+string(ImgIndex.ParseImgKeyToPlainTxt(key[1:]))
		displayStr := ""
		for _,which := range whiches{
			displayStr += strconv.Itoa(int(which)) + ","
		}


		if len(displayStr) > 0{
			displayStr = displayStr[:len(displayStr)-1]
		}

		displayStr += " | " + string(tagName)

		dispalyName := imgName + " ----> " + displayStr
		this.model.items[index] = NameValueItem{name:dispalyName, value:[]byte(displayStr), imgIdent:key, imgData:nil,whiches:whiches}
		index ++
	}

	sort.Sort(nameValueItemList(this.model.items))
	this.listBox.SetModel(this.model)

	newFocusIndex := len(this.model.items)-1
	//this.listBox.SetSelectedIndexes([]int{len(this.model.items)-1})
	this.listBox.SetCurrentIndex(newFocusIndex)

	this.doAgainButton.SetText("重做: " + this.model.items[newFocusIndex].name)
	this.doAgainButton.SetName(this.model.items[newFocusIndex].name)

	//重新绘制 pre image
	this.imgPreViewer.Invalidate()
	this.imgPreViewer.SetEnabled(true)
}



type NameValueItem struct {
	name     string  //用于展示
	value    []byte
	imgIdent []byte  //image identify
	imgData  []byte  //原始 image bytes
	whiches  []uint8 //选择的 whiches
}

func (this nameValueItemList)Len() int {
	return len(this)
}

func (this nameValueItemList) Swap(i, j int) {
	this[i], this[j] = this[j], this[i]
}

func (this nameValueItemList) Less(i, j int) bool {
	return strings.Compare(this[i].name,this[j].name) < 0
}


type nameValueItemList []NameValueItem
type NameValueModel struct {
	walk.ListModelBase
	items []NameValueItem
}

func NewNameValueModel() *NameValueModel {

	m := &NameValueModel{items: nil}

	return m
}

func (m *NameValueModel) ItemCount() int {
	return len(m.items)
}

func (m *NameValueModel) Value(index int) interface{} {
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





func trimLRSpace(input []byte) []byte {
	start := 0
	limit := len(input)
	for i:=0;i<len(input);i++{
		if ' ' == input[i]{
			start = i+1
		}else{
			break
		}
	}

	for j:=len(input)-1;j >= 0;j --{
		if ' ' == input[j]{
			limit = j
		}else{
			break
		}
	}
	if start>=len(input){
		return []byte{}
	}else if limit <=0 {
		return []byte{}
	}
	return input[start : limit]
}

func myStringTrim(input string) string {

	pre := 0
	us := []rune(input)
	for _,u := range us{
		if !unicode.IsSpace(u){
			break
		}
		pre ++
	}

	suf := len(us) -1
	for i:=suf;i != -1;i --{
		if !unicode.IsSpace(us[i]){
			break
		}
		suf --
	}

	//pre 是第一个非空格的位置, suf 是倒数第一个非空格的位置
	if suf +1 > pre{
		res := us[pre:suf+1]
		return string(res)
	}else{
		return ""
	}

}