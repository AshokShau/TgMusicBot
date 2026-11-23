/*
 * TgMusicBot - Telegram Music Bot
 *  Copyright (c) 2025-2026 Ashok Shau
 *
 *  Licensed under GNU GPL v3
 *  See https://github.com/AshokShau/TgMusicBot
 */

package calls

import (
	"ashokshau/tgmusic/internal/cache"
	"ashokshau/tgmusic/ntgcalls"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"regexp"
	"runtime"
	"strings"
	"time"
)

type probedStream struct {
	CodecType  string `json:"codec_type"`
	CodecName  string `json:"codec_name"`
	Channels   int    `json:"channels"`
	Width      int    `json:"width"`
	Height     int    `json:"height"`
	HasBFrames *int   `json:"has_b_frames"`
}

type mediaProbe struct {
	Streams []probedStream `json:"streams"`
	Format  struct {
		Name string `json:"format_name"`
	} `json:"format"`
}

func probeMedia(filePath string) (mediaProbe, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "ffprobe", "-v", "error", "-show_entries",
		"format=format_name:stream=codec_type,codec_name,channels,width,height,has_b_frames", "-of", "json", filePath)
	out, err := cmd.Output()
	if err != nil {
		return mediaProbe{}, err
	}
	var probe mediaProbe
	if err := json.Unmarshal(out, &probe); err != nil {
		return mediaProbe{}, err
	}
	return probe, nil
}

func (p mediaProbe) stream(kind string) *probedStream {
	for i := range p.Streams {
		if p.Streams[i].CodecType == kind {
			return &p.Streams[i]
		}
	}
	return nil
}

func (p mediaProbe) hasFormat(name string) bool {
	for format := range strings.SplitSeq(p.Format.Name, ",") {
		if format == name {
			return true
		}
	}
	return false
}

func canCopyAudio(stream *probedStream) bool {
	return stream != nil && stream.CodecName == "opus" && (stream.Channels == 1 || stream.Channels == 2)
}

func canCopyVideo(stream *probedStream) bool {
	if stream == nil || stream.Width <= 0 || stream.Height <= 0 || stream.Width > 1280 || stream.Height > 720 {
		return false
	}

	switch stream.CodecName {
	case "h264":
		return stream.HasBFrames != nil && *stream.HasBFrames == 0
	case "vp8", "vp9", "av1":
		return true
	default:
		return false
	}
}

func shellQuote(value string) string {
	if runtime.GOOS == "windows" {
		return "\"" + strings.ReplaceAll(value, "\"", "\\\"") + "\""
	}
	return "'" + strings.ReplaceAll(value, "'", "'\"'\"'") + "'"
}

var urlRegex = regexp.MustCompile(`^https?://`)

func getMediaDescription(filePath string, isVideo bool, chatId int64, ffmpegParameters string) ntgcalls.MediaDescription {
	var duration int
	track := cache.ChatCache.GetPlayingTrack(chatId)
	if track != nil {
		duration = int(track.Duration)
	}

	probe, probeErr := probeMedia(filePath)
	if probeErr != nil {
		logger.Warn("[getMediaDescription] ffprobe failed for %s: %v", filePath, probeErr)
	}

	audioFormat := ntgcalls.MediaFormatPCM
	audioDescription := &ntgcalls.AudioDescription{
		MediaSource:  ntgcalls.MediaSourceShell,
		SampleRate:   48000,
		ChannelCount: 2,
		Format:       &audioFormat,
	}

	quotedPath := shellQuote(filePath)
	isURL := urlRegex.MatchString(filePath)
	isLive := isURL && duration == 0

	var seekFlags, filterFlags string
	if ffmpegParameters != "" {
		if strings.Contains(ffmpegParameters, "filter:") {
			filterFlags = ffmpegParameters
		} else {
			seekFlags = ffmpegParameters
		}
	}

	var inputFlags strings.Builder
	if isLive {
		inputFlags.WriteString("-reconnect 1 -reconnect_at_eof 1 -reconnect_streamed 1 -reconnect_delay_max 2 ")
	}

	if seekFlags != "" {
		inputFlags.WriteString(seekFlags + " ")
	}

	canPassthrough := probeErr == nil && filterFlags == "" && !isLive
	if audio := probe.stream("audio"); canPassthrough && canCopyAudio(audio) {
		audioFormat = ntgcalls.MediaFormatOgg
		audioDescription.ChannelCount = uint8(audio.Channels)
		if !isURL && seekFlags == "" && probe.hasFormat("ogg") {
			audioDescription.MediaSource = ntgcalls.MediaSourceFile
			audioDescription.Input = filePath
		} else {
			audioDescription.Input = fmt.Sprintf("ffmpeg %s-i %s -map 0:a:0 -vn -c:a copy -f ogg -v error pipe:1", inputFlags.String(), quotedPath)
		}
	} else {
		audioDescription.Input = fmt.Sprintf("ffmpeg %s-i %s %s-f s16le -ac %d -ar %d -v quiet pipe:1",
			inputFlags.String(), quotedPath, filterFlagsWithSpace(filterFlags), audioDescription.ChannelCount, audioDescription.SampleRate)
	}

	if !isVideo {
		return ntgcalls.MediaDescription{
			Microphone: audioDescription,
		}
	}

	video := probe.stream("video")
	width := 1280
	height := 720

	if video != nil && video.Width > 0 && video.Height > 0 {
		ratio := float64(video.Width) / float64(video.Height)

		newW := min(video.Width, width)
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

	videoFormat := ntgcalls.MediaFormatI420
	videoDescription := &ntgcalls.VideoDescription{
		MediaSource: ntgcalls.MediaSourceShell,
		Width:       int16(width),
		Height:      int16(height),
		Fps:         30,
		Format:      &videoFormat,
	}
	if canPassthrough && canCopyVideo(video) {
		videoFormat = ntgcalls.MediaFormatMatroska
		if !isURL && seekFlags == "" && (probe.hasFormat("matroska") || probe.hasFormat("webm")) {
			videoDescription.MediaSource = ntgcalls.MediaSourceFile
			videoDescription.Input = filePath
		} else {
			videoDescription.Input = fmt.Sprintf("ffmpeg %s-i %s -map 0:v:0 -an -c:v copy -f matroska -v error pipe:1", inputFlags.String(), quotedPath)
		}
	} else {
		videoDescription.Input = fmt.Sprintf("ffmpeg %s-i %s %s-f rawvideo -r %d -pix_fmt yuv420p -vf scale=%d:%d -v quiet pipe:1",
			inputFlags.String(), quotedPath, filterFlagsWithSpace(filterFlags), videoDescription.Fps, videoDescription.Width, videoDescription.Height)
	}

	return ntgcalls.MediaDescription{
		Microphone: audioDescription,
		Camera:     videoDescription,
	}
}

func filterFlagsWithSpace(flags string) string {
	if flags != "" {
		return flags + " "
	}
	return ""
}
