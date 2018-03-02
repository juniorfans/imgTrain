// Copyright 2012 The Walk Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package dialogs


import (
	"github.com/lxn/walk"
	. "github.com/lxn/walk/declarative"

	"strconv"
	"imgSearch/src/dbOptions"
	"imgSearch/src/imgIndex"
	"fmt"
	"strings"
	"log"
	"imgCache"
)


func ShowMarkNotSameTopickDBDlg(imgIdent []byte, markLog *imgCache.MyMap) {

	mw := &MarkClipNotSameTopicDBDlgWnd{}

	var widgets []Widget
	if !markLog.Contains(imgIdent){
		widgets = newSameTopickWidgetsFromQuery(mw, imgIdent, markLog)
	}else{
		widgets = newSameTopickWidgetsFromMarkLog(mw, imgIdent, markLog)
	}

	fmt.Println(string(ImgIndex.ParseImgKeyToPlainTxt(imgIdent[1:])),": widgets length: ", len(widgets))

	if len(widgets) == 0{
		return
	}

	imgName := string(ImgIndex.ParseImgKeyToPlainTxt(imgIdent[1:]))
	heigth := len(widgets) * 50 + 30

	mainWnd := MainWindow{
		AssignTo: &mw.MainWindow,
		Title:    "标记协同关系非主题相似 - " + imgName,
		MinSize:  Size{200, heigth},
		MaxSize:  Size{200, heigth},
		Size:     Size{200, heigth},
		Visible:true,
		Layout:   VBox{MarginsZero: true},
		Children: widgets,
	}

	if _,err := mainWnd.Run(); err != nil {
		log.Fatal(err)
		return
	}

	return
}

func doMarkLog(imgIdent []byte, left, right uint8, markOperation bool, markLog *imgCache.MyMap)  {
	var mv uint8
	if markOperation{
		mv = 1
	}else{
		mv = 0
	}

	addIfNeed := make([]byte, 3)
	addIfNeed[0]=left
	addIfNeed[1]=right
	addIfNeed[2]=mv

	interfaceValue := markLog.Get(imgIdent)
	if len(interfaceValue) == 0{
		markLog.Put(imgIdent, addIfNeed)
		return
	}

	value := interfaceValue[0].([]byte)

	edited := false
	for i:=0;i<len(value);i+=3 {
		group := value[i:i + 3]
		if group[0] == left && group[1] == right || group[0]==right || group[1] == left{
			edited = true
			group[2] = mv
		}
	}
	if edited{
		return
	}

	newValue := make([]byte, len(value) + len(addIfNeed))
	ci := copy(newValue, value)
	copy(newValue[ci:], addIfNeed)
	return
}

func newSameTopickWidgetsFromMarkLog(mw *MarkClipNotSameTopicDBDlgWnd, imgIdent []byte, cached *imgCache.MyMap) []Widget {
	if nil == cached || !cached.Contains(imgIdent){
		return nil
	}

	imgName := string(ImgIndex.ParseImgKeyToPlainTxt(imgIdent[1:]))

	prefixes := []string{"否定 " + imgName + " 主题相似:", "取消否定 " + imgName + " 主题相似:"}

	markLog := cached.Get(imgIdent)
	if len(markLog) != 1{
		fmt.Println("error, markLog mymap value len must be 1, but is: ", len(markLog))
		return nil
	}

	seqStr := markLog[0].([]byte)

	if len(seqStr) % 3 != 0{
		fmt.Println("否定信息长度错误, 不是 3 的倍数: ", len(seqStr))
		return nil
	}

	ret := make([]Widget, len(seqStr) / 3)
	ci := 0

	for i:=0;i<len(seqStr);i+=3{
		group := seqStr[i:i+3]

		//没有标志过就显示否定, 否则显示取消
		prefix := prefixes[int(group[2])]

		show := prefix
		show += strconv.Itoa(int(group[0]))
		show += "|"
		show += strconv.Itoa(int(group[1]))

		var curBtn *walk.PushButton
		tmp := PushButton{
			AssignTo: &curBtn,
			Text: show,
			MinSize:Size{366,50},
			MaxSize:Size{366,50},
			Font:Font{PointSize:14},
			Visible:true,
		}

		tmp.OnMouseUp = func(x, y int, button walk.MouseButton){
			mw.markSameClipButtonUpEven(imgIdent, curBtn, cached)
		}

		ret[ci] = tmp
		ci ++
	}

	return ret
}

func newSameTopickWidgetsFromQuery(mw *MarkClipNotSameTopicDBDlgWnd, imgIdent []byte, cached *imgCache.MyMap) []Widget{
	sameClips := dbOptions.GetCoordinateClipsFromImgIdent(imgIdent,2)
	if len(sameClips) == 0{
		fmt.Println("没有协同关系数据")
		return nil
	}

	ret := make([]Widget, len(sameClips))
	imgName := string(ImgIndex.ParseImgKeyToPlainTxt(imgIdent[1:]))
	prefix := "否定 " + imgName + " 主题相似:"

	value := make([]byte, 3 * len(sameClips))
	ci := 0

	for i,sc := range sameClips{

		show := prefix
		show += strconv.Itoa(int(sc.Left))
		show += "|"
		show += strconv.Itoa(int(sc.Right))

		value[ci]=sc.Left; ci++
		value[ci]=sc.Right; ci++
		value[ci]=uint8(0); ci++

		var curBtn *walk.PushButton
		tmp := PushButton{
			AssignTo: &curBtn,
			Text: show,
			MinSize:Size{366,50},
			MaxSize:Size{366,50},
			Font:Font{PointSize:14},
			Visible:true,
		}

		tmp.OnMouseUp = func(x, y int, button walk.MouseButton){
			mw.markSameClipButtonUpEven(imgIdent, curBtn, cached)
		}

		ret[i] = tmp
	}

	cached.Put(imgIdent, value)

	return ret
}

type MarkClipNotSameTopicDBDlgWnd struct {
	*walk.MainWindow
}


func (this *MarkClipNotSameTopicDBDlgWnd) markSameClipButtonUpEven(imgIdent []byte, button *walk.PushButton, markLog *imgCache.MyMap)  {
	whichesStr := button.Text()
	if 0 == strings.Index(whichesStr, "取消"){
		this.cancelMarkNotSameClip(imgIdent, button, markLog)
	}else{
		this.markNotSameClip(imgIdent, button, markLog)
	}
}

func (this *MarkClipNotSameTopicDBDlgWnd) cancelMarkNotSameClip(imgIdent []byte, button *walk.PushButton, markLog *imgCache.MyMap)  {

	imgName := string(ImgIndex.ParseImgKeyToPlainTxt(imgIdent[1:]))
	whichesStr := button.Text()
	prefix := "取消否定 " + imgName + " 主题相似:"
	offset := strings.Index(whichesStr, ":")

	if len(whichesStr) != len(prefix) + 3 || offset+1 != len(prefix){
		walk.MsgBox(this, "Value", "取消否定信息错误" , walk.MsgBoxIconInformation)
		return
	}

	left := whichesStr[offset+1]
	right := whichesStr[offset+3]
	leftWhich := uint8(left) - uint8('0')
	rightWhich := uint8(right) - uint8('0')
	dbOptions.MarkClipsNotSameTopicCancel(imgIdent, leftWhich, rightWhich)
	fmt.Println("cancel mark clip not same ok: ",strconv.Itoa(int(imgIdent[0])), "_" ,  string(ImgIndex.ParseImgKeyToPlainTxt(imgIdent[1:])), " | ", leftWhich, ":", rightWhich)

	doMarkLog(imgIdent,leftWhich, rightWhich,false,markLog)

	//去掉取消二字
	whichesStr = whichesStr[len("取消"):]
	button.SetText(whichesStr)
	button.Invalidate()
}


func (this *MarkClipNotSameTopicDBDlgWnd)markNotSameClip(imgIdent []byte, button *walk.PushButton, markLog *imgCache.MyMap)  {
	imgName := string(ImgIndex.ParseImgKeyToPlainTxt(imgIdent[1:]))
	whichesStr := button.Text()
	prefix := "否定 " + imgName + " 主题相似:"
	offset := strings.Index(whichesStr, ":")

	if len(whichesStr) != len(prefix) + 3 || offset+1 != len(prefix){
		walk.MsgBox(this, "Value", "否定信息错误" , walk.MsgBoxIconInformation)
		return
	}

	left := whichesStr[offset+1]
	right := whichesStr[offset+3]
	leftWhich := uint8(left) - uint8('0')
	rightWhich := uint8(right) - uint8('0')
	dbOptions.MarkClipsNotSameTopic(imgIdent, leftWhich, rightWhich)
	fmt.Println("mark clip not same ok: ",strconv.Itoa(int(imgIdent[0])), "_" ,  string(ImgIndex.ParseImgKeyToPlainTxt(imgIdent[1:])), " | ", leftWhich, ":", rightWhich)


	doMarkLog(imgIdent,leftWhich, rightWhich, true,markLog)

	//加上取消二字
	whichesStr = "取消" + whichesStr
	button.SetText(whichesStr)
	button.Invalidate()
}