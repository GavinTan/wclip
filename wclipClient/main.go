package main

import (
	"bytes"
	"context"
	"encoding/json"
	"image/color"
	"io"
	"net/http"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"
	"golang.design/x/clipboard"
)

var (
	title             = "ClipMonitor"
	clipMonitorServer = "http://127.0.0.1:6233/clip"
	isManualWrite     bool
)

type ClipData struct {
	// windows: 1  darwin: 2  linux: 3  android: 4  ios: 5
	Type    int    `json:"type"`
	Content string `json:"content"`
}

func localClipboardWatch() {
	err := clipboard.Init()
	if err != nil {
		panic(err)
	}

	ch := clipboard.Watch(context.TODO(), clipboard.FmtText)
	for data := range ch {
		if isManualWrite {
			isManualWrite = false
			continue
		}

		jdata := map[string]interface{}{
			"type":    1,
			"content": string(data),
		}

		body, _ := json.Marshal(jdata)
		resp, err := http.Post(clipMonitorServer, "application/json", bytes.NewBuffer(body))
		if err != nil {
			return
		}

		io.Copy(io.Discard, resp.Body)
		resp.Body.Close()
	}
}

func getClipData() ClipData {
	resp, err := http.Get(clipMonitorServer)
	if err != nil {
		return ClipData{}
	}

	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()

	var data ClipData
	json.Unmarshal(body, &data)

	return data
}

func monitorClipServer() {
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	for range ticker.C {
		currentContent := clipboard.Read(clipboard.FmtText)
		clipData := getClipData()

		if clipData.Type > 1 && clipData.Content != "" {
			if clipData.Content != string(currentContent) {
				isManualWrite = true
				clipboard.Write(clipboard.FmtText, []byte(clipData.Content))
			}
		}
	}
}

func main() {
	a := app.NewWithID(title)
	a.SetIcon(resourceLogoPng)
	w := a.NewWindow(title)
	w.SetFixedSize(true)
	w.Resize(fyne.NewSize(300, 380))

	titleText := canvas.NewText("剪贴板监听助手", color.NRGBA{R: 0, G: 0, B: 0, A: 255})
	titleText.Alignment = fyne.TextAlignCenter
	titleText.TextStyle.Bold = true
	titleText.TextSize = 16

	tagText := canvas.NewText("Windows  ⇆  苹果 & Android", color.NRGBA{R: 105, G: 105, B: 105, A: 255})
	tagText.Alignment = fyne.TextAlignCenter
	tagText.TextSize = 12

	logo := canvas.NewImageFromResource(resourceLogoPng)
	logo.FillMode = canvas.ImageFillContain
	logo.SetMinSize(fyne.NewSize(192, 192))

	clipMonitorServer = a.Preferences().StringWithFallback("clipMonitorServer", clipMonitorServer)

	input := widget.NewEntry()
	input.SetText(clipMonitorServer)
	input.OnChanged = func(s string) {
		a.Preferences().SetString("clipMonitorServer", s)
	}

	content := container.NewVBox(
		logo,
		container.NewGridWithRows(2, titleText, tagText),
		layout.NewSpacer(),
		container.New(layout.NewFormLayout(), widget.NewLabel("服务器:"), input),
	)

	if desk, ok := a.(desktop.App); ok {
		m := fyne.NewMenu("ClipMonitor",
			fyne.NewMenuItem("Show", func() {
				w.Show()
			}))
		desk.SetSystemTrayMenu(m)
	}

	w.SetContent(content)

	w.SetCloseIntercept(func() {
		w.Hide()
	})

	go monitorClipServer()
	go localClipboardWatch()

	w.ShowAndRun()
}
