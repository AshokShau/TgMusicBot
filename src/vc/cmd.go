package vc

import (
	"ashokshau/tgmusic/src/vc/ntgcalls"
	"fmt"
	"regexp"
	"strings"
)

var isURLRegex = regexp.MustCompile(`^https?://`)

// getMediaDescription creates a media description for ntgcalls based on the provided file path, video status, and ffmpeg parameters.
func getMediaDescription(filePath string, isVideo bool, ffmpegParameters string) ntgcalls.MediaDescription {
	audioDescription := &ntgcalls.AudioDescription{
		MediaSource:  ntgcalls.MediaSourceShell,
		SampleRate:   48000,
		ChannelCount: 2,
	}

	quotedPath := fmt.Sprintf("\"%s\"", filePath)
	isURL := isURLRegex.MatchString(filePath)

	var audioCmd strings.Builder
	audioCmd.WriteString("ffmpeg ")
	if isURL {
		audioCmd.WriteString("-reconnect 1 -reconnect_at_eof 1 -reconnect_streamed 1 -reconnect_delay_max 2 ")
	}

	var seekFlags, filterFlags string
	if ffmpegParameters != "" {
		if strings.Contains(ffmpegParameters, "filter:") {
			filterFlags = ffmpegParameters
		} else {
			seekFlags = ffmpegParameters
		}
	}

	if seekFlags != "" {
		audioCmd.WriteString(seekFlags + " ")
	}

	audioCmd.WriteString("-i " + quotedPath + " ")
	if filterFlags != "" {
		audioCmd.WriteString(filterFlags + " ")
	}

	audioCmd.WriteString(fmt.Sprintf("-f s16le -ac %d -ar %d -v quiet pipe:1",
		audioDescription.ChannelCount,
		audioDescription.SampleRate,
	))
	audioDescription.Input = audioCmd.String()

	if !isVideo {
		return ntgcalls.MediaDescription{
			Microphone: audioDescription,
		}
	}

	originalWidth, originalHeight := getVideoDimensions(filePath)

	width := 1280
	height := 720

	if originalWidth > 0 && originalHeight > 0 {
		ratio := float64(originalWidth) / float64(originalHeight)
		newW := min(originalWidth, width)
		newH := int(float64(newW) / ratio)

		if newH > height {
			newH = height
			newW = int(float64(newH) * ratio)
		}

		if newW%2 != 0 {
			newW--
		}
		if newH%2 != 0 {
			newH--
		}

		width = newW
		height = newH
	}

	videoDescription := &ntgcalls.VideoDescription{
		MediaSource: ntgcalls.MediaSourceShell,
		Width:       int16(width),
		Height:      int16(height),
		Fps:         30,
	}

	var videoCmd strings.Builder
	videoCmd.WriteString("ffmpeg ")

	if isURL {
		videoCmd.WriteString("-reconnect 1 -reconnect_at_eof 1 -reconnect_streamed 1 -reconnect_delay_max 2 ")
	}

	if seekFlags != "" {
		videoCmd.WriteString(seekFlags + " ")
	}

	videoCmd.WriteString(fmt.Sprintf("-i %s ", quotedPath))
	if filterFlags != "" {
		videoCmd.WriteString(filterFlags + " ")
	}

	videoCmd.WriteString(fmt.Sprintf("-f rawvideo -r %d -pix_fmt yuv420p -vf scale=%d:%d -v quiet pipe:1",
		videoDescription.Fps,
		videoDescription.Width,
		videoDescription.Height,
	))
	videoDescription.Input = videoCmd.String()

	return ntgcalls.MediaDescription{
		Microphone: audioDescription,
		Camera:     videoDescription,
	}
}
