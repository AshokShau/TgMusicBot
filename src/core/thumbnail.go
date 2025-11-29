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
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"io"
	"math"
	"net/http"
	"os"
	"strings"

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

// resizeImage resizes an image to the specified width and height using bilinear interpolation
func resizeImage(img image.Image, width, height int) image.Image {
	srcBounds := img.Bounds()
	srcW := srcBounds.Dx()
	srcH := srcBounds.Dy()

	dst := image.NewRGBA(image.Rect(0, 0, width, height))

	xRatio := float64(srcW) / float64(width)
	yRatio := float64(srcH) / float64(height)

	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			srcX := float64(x) * xRatio
			srcY := float64(y) * yRatio

			// Bilinear interpolation
			x1 := int(math.Floor(srcX))
			y1 := int(math.Floor(srcY))
			x2 := x1 + 1
			y2 := y1 + 1

			if x2 >= srcW {
				x2 = srcW - 1
			}
			if y2 >= srcH {
				y2 = srcH - 1
			}

			// Get surrounding pixels
			q11 := img.At(x1, y1)
			q12 := img.At(x1, y2)
			q21 := img.At(x2, y1)
			q22 := img.At(x2, y2)

			// Interpolate
			dx := srcX - float64(x1)
			dy := srcY - float64(y1)

			c11 := color.RGBAModel.Convert(q11).(color.RGBA)
			c12 := color.RGBAModel.Convert(q12).(color.RGBA)
			c21 := color.RGBAModel.Convert(q21).(color.RGBA)
			c22 := color.RGBAModel.Convert(q22).(color.RGBA)

			r := bilinearInterpolate(c11.R, c12.R, c21.R, c22.R, dx, dy)
			g := bilinearInterpolate(c11.G, c12.G, c21.G, c22.G, dx, dy)
			b := bilinearInterpolate(c11.B, c12.B, c21.B, c22.B, dx, dy)
			a := bilinearInterpolate(c11.A, c12.A, c21.A, c22.A, dx, dy)

			dst.Set(x, y, color.RGBA{R: r, G: g, B: b, A: a})
		}
	}
	return dst
}

func bilinearInterpolate(q11, q12, q21, q22 uint8, dx, dy float64) uint8 {
	v1 := float64(q11)*(1-dx) + float64(q21)*dx
	v2 := float64(q12)*(1-dx) + float64(q22)*dx
	result := v1*(1-dy) + v2*dy
	return uint8(math.Round(result))
}

// applyBlur applies a simple box blur to the image
func applyBlur(img image.Image, radius int) image.Image {
	if radius <= 0 {
		return img
	}

	bounds := img.Bounds()
	dst := image.NewRGBA(bounds)

	// Simple box blur
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			var r, g, b, a, count uint32

			for dy := -radius; dy <= radius; dy++ {
				for dx := -radius; dx <= radius; dx++ {
					xx := x + dx
					yy := y + dy
					if xx >= bounds.Min.X && xx < bounds.Max.X && yy >= bounds.Min.Y && yy < bounds.Max.Y {
						c := color.RGBAModel.Convert(img.At(xx, yy)).(color.RGBA)
						r += uint32(c.R)
						g += uint32(c.G)
						b += uint32(c.B)
						a += uint32(c.A)
						count++
					}
				}
			}

			if count > 0 {
				dst.Set(x, y, color.RGBA{
					R: uint8(r / count),
					G: uint8(g / count),
					B: uint8(b / count),
					A: uint8(a / count),
				})
			}
		}
	}
	return dst
}

// adjustBrightness adjusts the brightness of an image
func adjustBrightness(img image.Image, factor float64) image.Image {
	bounds := img.Bounds()
	dst := image.NewRGBA(bounds)

	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			c := color.RGBAModel.Convert(img.At(x, y)).(color.RGBA)

			r := float64(c.R) * (1 + factor)
			g := float64(c.G) * (1 + factor)
			b := float64(c.B) * (1 + factor)

			r = clamp(r, 0, 255)
			g = clamp(g, 0, 255)
			b = clamp(b, 0, 255)

			dst.Set(x, y, color.RGBA{
				R: uint8(r),
				G: uint8(g),
				B: uint8(b),
				A: c.A,
			})
		}
	}
	return dst
}

func clamp(value, min, max float64) float64 {
	if value < min {
		return min
	}
	if value > max {
		return max
	}
	return value
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

	file, err := os.Open(tmpFile)
	if err != nil {
		return "", err
	}
	defer file.Close()

	img, _, err := image.Decode(file)
	if err != nil {
		return "", err
	}

	_ = os.Remove(tmpFile)

	bg := resizeImage(img, 1280, 720)
	bg = applyBlur(bg, 7)
	bg = adjustBrightness(bg, -0.5)

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
