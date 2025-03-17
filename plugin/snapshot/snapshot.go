package snapshot

import (
	"fmt"
	"image"
	"image/color"
	"image/jpeg"
	"log"
	"os"
	"path/filepath"
	"time"

	"m7s.live/engine/v4"
	"m7s.live/engine/v4/config"
)

type SnapshotConfig struct {
	Enable    bool    `yaml:"enable"`
	SavePath  string  `yaml:"savePath"`
	FrameRate float64 `yaml:"frameRate"`
	StreamKey string  `yaml:"streamKey"`
}

type SnapshotPlugin struct {
	engine.Plugin
	SnapshotConfig
	lastTime time.Time
}

var snapshotPlugin = &SnapshotPlugin{}

func (p *SnapshotPlugin) OnEvent(event any) {
	switch ev := event.(type) {
	case *engine.VideoFrame:
		now := time.Now()
		if now.Sub(p.lastTime) >= time.Duration(1000.0/p.FrameRate)*time.Millisecond {
			p.lastTime = now
			p.captureFrame(ev)
		}
	}
}

func (p *SnapshotPlugin) captureFrame(frame *engine.VideoFrame) {
	data, ok := frame.Data.([]byte)
	if !ok {
		log.Println("[Snapshot] Frame data is not a byte slice")
		return
	}

	img := frameToImage(frame, data)
	if img == nil {
		log.Println("[Snapshot] Failed to decode YUV frame")
		return
	}

	filename := filepath.Join(p.SavePath, fmt.Sprintf("%d.jpg", time.Now().UnixNano()))
	file, err := os.Create(filename)
	if err != nil {
		log.Println("[Snapshot] Error creating file:", err)
		return
	}
	defer file.Close()

	err = jpeg.Encode(file, img, &jpeg.Options{Quality: 90})
	if err != nil {
		log.Println("[Snapshot] Error encoding image:", err)
	} else {
		log.Println("[Snapshot] Image saved:", filename)
	}
}

func frameToImage(frame *engine.VideoFrame, data []byte) *image.RGBA {
	bounds := image.Rect(0, 0, int(frame.Width), int(frame.Height))
	img := image.NewRGBA(bounds)

	for y := 0; y < int(frame.Height); y++ {
		for x := 0; x < int(frame.Width); x++ {
			Y := data[y*int(frame.Width)+x]
			U := data[int(frame.Width)*int(frame.Height)+(y/2)*(int(frame.Width)/2)+x/2]
			V := data[int(frame.Width)*int(frame.Height)*5/4+(y/2)*(int(frame.Width)/2)+x/2]

			img.Set(x, y, yuvToRGB(Y, U, V))
		}
	}
	return img
}

func yuvToRGB(y, u, v uint8) color.Color {
	r := float64(y) + 1.402*(float64(v)-128)
	g := float64(y) - 0.34414*(float64(u)-128) - 0.71414*(float64(v)-128)
	b := float64(y) + 1.772*(float64(u)-128)
	return color.RGBA{uint8(r), uint8(g), uint8(b), 255}
}

func InitSnapshotPlugin() {
	conf := &SnapshotConfig{}
	config.LoadConfig("snapshot", conf)

	snapshotPlugin = &SnapshotPlugin{
		SnapshotConfig: *conf,
	}

	os.MkdirAll(snapshotPlugin.SavePath, os.ModePerm)
	engine.InstallPlugin(snapshotPlugin)
}

func init() {
	InitSnapshotPlugin()
}
