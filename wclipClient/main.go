package main

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"image"
	"image/color"
	"image/draw"
	_ "image/jpeg"
	"image/png"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"
	log "github.com/sirupsen/logrus"
	"golang.design/x/clipboard"
)

var (
	title             = "ClipMonitor"
	clipMonitorServer = "http://127.0.0.1:6233/clip"
	timeStamp         int64
	mu                sync.Mutex
	isManualWrite     bool
)

type ClipData struct {
	Content   string `json:"content"`
	Mime      string `json:"mime"`
	Timestamp int64  `json:"timestamp"`
}

func getData() ClipData {
	mu.Lock()
	defer mu.Unlock()

	resp, err := http.Get(clipMonitorServer)
	if err != nil {
		return ClipData{}
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return ClipData{}
	}

	body, _ := io.ReadAll(resp.Body)

	var data ClipData
	json.Unmarshal(body, &data)

	return data
}

func updateData(mime, content string) {
	mu.Lock()
	defer mu.Unlock()

	timeStamp = time.Now().Unix()
	data := ClipData{
		Content:   content,
		Mime:      mime,
		Timestamp: timeStamp,
	}

	body, _ := json.Marshal(data)
	resp, err := http.Post(clipMonitorServer, "application/json", bytes.NewBuffer(body))
	if err != nil {
		log.Error(err)
		return
	}

	io.Copy(io.Discard, resp.Body)
	resp.Body.Close()
}

func localClipboardWatch(ctx context.Context) {
	chText := clipboard.Watch(ctx, clipboard.FmtText)
	chImage := clipboard.Watch(ctx, clipboard.FmtImage)
	for {
		select {
		case data := <-chText:
			if isManualWrite {
				isManualWrite = false
				continue
			}

			content := strings.TrimSpace(string(data))
			if content != "" {
				updateData("text/plain", content)
			}
		case data := <-chImage:
			if isManualWrite {
				isManualWrite = false
				continue
			}

			if len(data) > 0 {
				updateData("image/png", base64.StdEncoding.EncodeToString(data))
			}
		case <-ctx.Done():
			return
		}
	}
}

func monitorClipServer(ctx context.Context) {
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			clipData := getData()
			if clipData.Timestamp != timeStamp && strings.TrimSpace(clipData.Content) != "" {
				switch clipData.Mime {
				case "text/plain":
					currentContent := clipboard.Read(clipboard.FmtText)
					if clipData.Content != string(currentContent) {
						timeStamp = clipData.Timestamp
						isManualWrite = true
						clipboard.Write(clipboard.FmtText, []byte(clipData.Content))
					}
				case "image/png":
					isManualWrite = true
					imgData, err := base64.StdEncoding.DecodeString(clipData.Content)
					if err != nil {
						log.Error(err)
						continue
					}

					img, _, _ := image.Decode(bytes.NewReader(imgData))
					bounds := img.Bounds()
					newImg := image.NewNRGBA(image.Rect(0, 0, bounds.Dx(), bounds.Dy()))
					draw.Draw(newImg, newImg.Bounds(), img, bounds.Min, draw.Src)
					buf := new(bytes.Buffer)
					png.Encode(buf, newImg)

					timeStamp = clipData.Timestamp
					clipboard.Write(clipboard.FmtImage, buf.Bytes())
				}
			}
		case <-ctx.Done():
			return
		}

	}
}

func init() {
	if err := clipboard.Init(); err != nil {
		log.Fatal(err)
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
		clipMonitorServer = s
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
			fyne.NewMenuItem("显示", func() {
				w.Show()
			}))
		desk.SetSystemTrayMenu(m)
		desk.SetSystemTrayWindow(w)
	}

	w.SetContent(content)

	w.SetCloseIntercept(func() {
		w.Hide()
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go monitorClipServer(ctx)
	go localClipboardWatch(ctx)

	w.ShowAndRun()
}
