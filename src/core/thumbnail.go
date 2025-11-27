/*
 * TgMusicBot - Telegram Music Bot
 *  Copyright (c) 2025 Ashok Shau
 *
 *  Licensed under GNU GPL v3
 *  See https://github.com/AshokShau/TgMusicBot
 */

package core

import (
	"fmt"
	"image/color"
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
	resp, err := http.Get(url)
	if err != nil {
		return err
	}

	defer resp.Body.Close()
	f, err := os.Create(filepath)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = io.Copy(f, resp.Body)
	return err
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

func GenThumb(song *cache.CachedTrack) (string, error) {
	vidID := song.TrackID
	cacheFile := fmt.Sprintf("cache/%s.png", vidID)
	if _, err := os.Stat(cacheFile); err == nil {
		return cacheFile, nil
	}

	title := song.Name
	channel := "TgMusic"
	duration := cache.SecToMin(song.Duration)
	views := "69K"

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
