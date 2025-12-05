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
	"image/draw"
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
	Font1         = "assets/font.ttf"
	Font2         = "assets/font2.ttf"
	maxBlurRadius = 10
	targetWidth   = 1280
	targetHeight  = 720
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

// toRGBA converts any image to *image.RGBA for efficient pixel access
func toRGBA(img image.Image) *image.RGBA {
	if rgba, ok := img.(*image.RGBA); ok {
		return rgba
	}

	bounds := img.Bounds()
	rgba := image.NewRGBA(bounds)
	draw.Draw(rgba, bounds, img, bounds.Min, draw.Src)
	return rgba
}

// resizeImage resizes an image to the specified width and height using bilinear interpolation
func resizeImage(img image.Image, width, height int) *image.RGBA {
	src := toRGBA(img)
	srcBounds := src.Bounds()
	srcW := srcBounds.Dx()
	srcH := srcBounds.Dy()

	dst := image.NewRGBA(image.Rect(0, 0, width, height))
	dstPix := dst.Pix
	srcPix := src.Pix
	srcStride := src.Stride

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

			i11 := (y1-srcBounds.Min.Y)*srcStride + (x1-srcBounds.Min.X)*4
			i12 := (y2-srcBounds.Min.Y)*srcStride + (x1-srcBounds.Min.X)*4
			i21 := (y1-srcBounds.Min.Y)*srcStride + (x2-srcBounds.Min.X)*4
			i22 := (y2-srcBounds.Min.Y)*srcStride + (x2-srcBounds.Min.X)*4

			r11, g11, b11, a11 := srcPix[i11], srcPix[i11+1], srcPix[i11+2], srcPix[i11+3]
			r12, g12, b12, a12 := srcPix[i12], srcPix[i12+1], srcPix[i12+2], srcPix[i12+3]
			r21, g21, b21, a21 := srcPix[i21], srcPix[i21+1], srcPix[i21+2], srcPix[i21+3]
			r22, g22, b22, a22 := srcPix[i22], srcPix[i22+1], srcPix[i22+2], srcPix[i22+3]

			dx := srcX - float64(x1)
			dy := srcY - float64(y1)

			r := bilinearInterpolate(r11, r12, r21, r22, dx, dy)
			g := bilinearInterpolate(g11, g12, g21, g22, dx, dy)
			b := bilinearInterpolate(b11, b12, b21, b22, dx, dy)
			a := bilinearInterpolate(a11, a12, a21, a22, dx, dy)

			idx := y*dst.Stride + x*4
			dstPix[idx] = r
			dstPix[idx+1] = g
			dstPix[idx+2] = b
			dstPix[idx+3] = a
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
func applyBlur(img image.Image, radius int) *image.RGBA {
	if radius <= 0 {
		return toRGBA(img)
	}

	if radius > maxBlurRadius {
		radius = maxBlurRadius
	}

	src := toRGBA(img)
	bounds := src.Bounds()
	dst := image.NewRGBA(bounds)
	srcPix := src.Pix
	dstPix := dst.Pix
	stride := src.Stride

	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			var r, g, b, a uint32

			yStart := m(y-radius, bounds.Min.Y)
			yEnd := i(y+radius, bounds.Max.Y-1)
			xStart := m(x-radius, bounds.Min.X)
			xEnd := i(x+radius, bounds.Max.X-1)

			actualArea := (yEnd - yStart + 1) * (xEnd - xStart + 1)

			for ky := yStart; ky <= yEnd; ky++ {
				for kx := xStart; kx <= xEnd; kx++ {
					idx := (ky-bounds.Min.Y)*stride + (kx-bounds.Min.X)*4
					r += uint32(srcPix[idx])
					g += uint32(srcPix[idx+1])
					b += uint32(srcPix[idx+2])
					a += uint32(srcPix[idx+3])
				}
			}

			dstIdx := (y-bounds.Min.Y)*stride + (x-bounds.Min.X)*4
			dstPix[dstIdx] = uint8(r / uint32(actualArea))
			dstPix[dstIdx+1] = uint8(g / uint32(actualArea))
			dstPix[dstIdx+2] = uint8(b / uint32(actualArea))
			dstPix[dstIdx+3] = uint8(a / uint32(actualArea))
		}
	}
	return dst
}

// adjustBrightness adjusts the brightness of an image
func adjustBrightness(img image.Image, factor float64) *image.RGBA {
	src := toRGBA(img)
	bounds := src.Bounds()
	dst := image.NewRGBA(bounds)
	srcPix := src.Pix
	dstPix := dst.Pix

	for i := 0; i < len(srcPix); i += 4 {
		r := float64(srcPix[i]) * (1 + factor)
		g := float64(srcPix[i+1]) * (1 + factor)
		b := float64(srcPix[i+2]) * (1 + factor)

		dstPix[i] = clampUint8(r)
		dstPix[i+1] = clampUint8(g)
		dstPix[i+2] = clampUint8(b)
		dstPix[i+3] = srcPix[i+3]
	}
	return dst
}

func clampUint8(value float64) uint8 {
	if value < 0 {
		return 0
	}
	if value > 255 {
		return 255
	}
	return uint8(math.Round(value))
}

func i(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func m(a, b int) int {
	if a > b {
		return a
	}
	return b
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
	defer func(file *os.File) {
		_ = file.Close()
	}(file)

	img, _, err := image.Decode(file)
	if err != nil {
		return "", err
	}

	_ = os.Remove(tmpFile)

	bg := resizeImage(img, targetWidth, targetHeight)
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
