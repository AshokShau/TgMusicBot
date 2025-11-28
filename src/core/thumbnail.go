/*
 * TgMusicBot - Telegram Music Bot
 *  Copyright (c) 2025 Ashok Shau
 *
 *  Licensed under GNU GPL v3
 *  See https://github.com/AshokShau/TgMusicBot
 */

package core

import (
	"bytes"
	"fmt"
	"image/color"
	"image/jpeg"
	"image/png"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/disintegration/imaging"
	"github.com/fogleman/gg"
	"golang.org/x/image/font"
	"golang.org/x/image/font/opentype"

	"ashokshau/tgmusic/src/core/cache"
)

const (
	Font1 = "assets/font.ttf"
	Font2 = "assets/font2.ttf"
)

func clearTitle(text string) string {
	words := strings.Split(text, " ")
	out := ""
	for _, w := range words {
		if len(out)+len(w) < 60 {
			out += " " + w
		}
	}
	return strings.TrimSpace(out)
}

func downloadImage(url, filepath string) error {
	if strings.Contains(url, "ytimg.com") {
		url = strings.Replace(url, "hqdefault.jpg", "maxresdefault.jpg", 1)
	}

	client := &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return nil
		},
	}

	resp, err := client.Get(url)
	if err != nil {
		return err
	}
	defer func(Body io.ReadCloser) {
		_ = Body.Close()
	}(resp.Body)

	ct := resp.Header.Get("Content-Type")
	if !strings.Contains(ct, "image") {
		return fmt.Errorf("not an image: %s", ct)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	img, err := jpeg.Decode(bytes.NewReader(body))
	if err != nil {
		img, err = png.Decode(bytes.NewReader(body))
		if err != nil {
			return fmt.Errorf("decode failed (%s): %v - only JPEG and PNG supported", ct, err)
		}
	}

	file, err := os.Create(filepath)
	if err != nil {
		return err
	}
	defer func(file *os.File) {
		_ = file.Close()
	}(file)

	return png.Encode(file, img)
}

func loadFont(path string, size float64) (font.Face, error) {
	fontBytes, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	f, err := opentype.Parse(fontBytes)
	if err != nil {
		return nil, err
	}
	face, err := opentype.NewFace(f, &opentype.FaceOptions{
		Size:    size,
		DPI:     72,
		Hinting: font.HintingFull,
	})
	return face, err
}

func GenThumb(song cache.CachedTrack) (string, error) {
	if song.Thumbnail == "" {
		return "", nil
	}

	if song.Platform == cache.Telegram {
		return "", nil
	}

	if song.Channel == "" {
		song.Channel = "TgMusicBot"
	}

	if song.Views == "" {
		song.Views = "699K"
	}

	vidID := song.TrackID
	cacheFile := fmt.Sprintf("cache/%s.png", vidID)
	if _, err := os.Stat(cacheFile); err == nil {
		return cacheFile, nil
	}

	title := song.Name
	duration := cache.SecToMin(song.Duration)
	channel := song.Channel
	views := song.Views
	thumb := song.Thumbnail
	tmpFile := fmt.Sprintf("cache/tmp_%s.png", vidID)

	err := downloadImage(thumb, tmpFile)
	if err != nil {
		return "", err
	}

	img, err := imaging.Open(tmpFile)
	if err != nil {
		return "", err
	}

	_ = os.Remove(tmpFile)

	bg := imaging.Resize(img, 1280, 720, imaging.Lanczos)
	bg = imaging.Blur(bg, 7)
	bg = imaging.AdjustBrightness(bg, -0.5)

	dc := gg.NewContextForImage(bg)

	fontTitle, _ := loadFont(Font1, 30)
	fontMeta, _ := loadFont(Font2, 30)

	dc.SetFontFace(fontMeta)
	dc.SetColor(color.White)

	dc.DrawStringAnchored(channel+" | "+views, 90, 580, 0, 0)
	dc.SetFontFace(fontTitle)
	dc.DrawStringAnchored(clearTitle(title), 90, 620, 0, 0)

	dc.SetColor(color.White)
	dc.SetLineWidth(5)
	dc.DrawLine(55, 660, 1220, 660)
	dc.Stroke()

	dc.DrawCircle(930, 660, 12)
	dc.Fill()

	dc.SetFontFace(fontMeta)
	dc.DrawStringAnchored("00:00", 40, 690, 0, 0)
	dc.DrawStringAnchored(duration, 1240, 690, 1, 0)

	err = dc.SavePNG(cacheFile)
	if err != nil {
		return "", err
	}

	return cacheFile, nil
}
