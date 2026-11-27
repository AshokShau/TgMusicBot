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
	"math"
	"os"
	"os/exec"
	"regexp"
	"runtime"
	"strconv"
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
	FrameRate  string `json:"avg_frame_rate"`
}

type mediaProbe struct {
	Streams []probedStream `json:"streams"`
	Format  struct {
		Duration string `json:"duration"`
	} `json:"format"`
}

func probeMedia(filePath string) (mediaProbe, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "ffprobe", "-v", "error", "-show_entries",
		"format=duration:stream=codec_type,codec_name,channels,width,height,has_b_frames,avg_frame_rate", "-of", "json", filePath)
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

func canCopyAudio(stream *probedStream) bool {
	return stream != nil && stream.CodecName == "opus" && (stream.Channels == 1 || stream.Channels == 2)
}

func canCopyVideo(stream *probedStream, width, height, fps int) bool {
	if stream == nil || stream.Width <= 0 || stream.Height <= 0 ||
		stream.Width != width || stream.Height != height ||
		(stream.HasBFrames != nil && *stream.HasBFrames != 0) {
		return false
	}
	frameRate := parseFrameRate(stream.FrameRate)
	if frameRate <= 0 || frameRate > float64(fps) {
		return false
	}

	switch stream.CodecName {
	case "h264", "vp8", "vp9", "av1":
		return true
	default:
		return false
	}
}

func parseFrameRate(value string) float64 {
	numerator, denominator, hasDenominator := strings.Cut(value, "/")
	n, err := strconv.ParseFloat(numerator, 64)
	if err != nil {
		return 0
	}
	d := 1.0
	if hasDenominator {
		d, err = strconv.ParseFloat(denominator, 64)
		if err != nil || d == 0 {
			return 0
		}
	}
	rate := n / d
	if math.IsNaN(rate) || math.IsInf(rate, 0) {
		return 0
	}
	return rate
}

func seekOnly(parameters string) bool {
	args := strings.Fields(parameters)
	if len(args)%2 != 0 {
		return false
	}
	for i := 0; i < len(args); i += 2 {
		switch args[i] {
		case "-ss", "-to", "-t":
		default:
			return false
		}
	}
	return true
}

func shellQuote(value string) string {
	if runtime.GOOS == "windows" {
		return "\"" + strings.ReplaceAll(value, "\"", "\\\"") + "\""
	}
	return "'" + strings.ReplaceAll(value, "'", "'\"'\"'") + "'"
}

var urlRegex = regexp.MustCompile(`^https?://`)

const opusBitrate uint32 = 128_000

func getMediaDescription(filePath string, isVideo bool, chatId int64, ffmpegParameters string) ntgcalls.MediaDescription {
	// Let the native reader handle supported local files so compatible streams
	// can pass through without starting FFmpeg.
	if ffmpegParameters == "" {
		if file, err := os.Stat(filePath); err == nil && file.Mode().IsRegular() {
			if info, err := ntgcalls.GetMediaInfo(filePath, ntgcalls.CaptureStream); err == nil {
				if desc, ok := nativeMediaDescription(filePath, isVideo, info); ok {
					return desc
				}
			} else {
				logger.Debug("NTgCalls could not inspect local media", "path", filePath, "error", err)
			}
		}
	}

	probe, probeErr := probeMedia(filePath)
	if probeErr != nil {
		logger.Warn("[getMediaDescription] ffprobe failed for %s: %v", filePath, probeErr)
	}

	audioFormat := ntgcalls.MediaFormatPCM
	bitrate := opusBitrate
	audioDescription := &ntgcalls.AudioDescription{
		MediaSource:  ntgcalls.MediaSourceShell,
		SampleRate:   48000,
		ChannelCount: 2,
		Bitrate:      &bitrate,
		Format:       &audioFormat,
	}

	quotedPath := shellQuote(filePath)
	isURL := urlRegex.MatchString(filePath)
	isLive := isURL && (probeErr == nil && probe.Format.Duration == "")
	if isURL && probeErr != nil {
		track := cache.ChatCache.GetPlayingTrack(chatId)
		isLive = track == nil || track.Duration == 0
	}

	var seekFlags, filterFlags string
	if ffmpegParameters != "" {
		if strings.Contains(ffmpegParameters, "filter:") {
			filterFlags = ffmpegParameters
		} else {
			seekFlags = ffmpegParameters
		}
	}

	var inputFlags strings.Builder
	if isURL && !isLive {
		inputFlags.WriteString("-reconnect 1 -reconnect_streamed 1 -reconnect_delay_max 2 ")
	}

	if seekFlags != "" {
		inputFlags.WriteString(seekFlags + " ")
	}

	canPassthrough := probeErr == nil && filterFlags == "" && seekOnly(seekFlags)
	if audio := probe.stream("audio"); canPassthrough && canCopyAudio(audio) {
		audioFormat = ntgcalls.MediaFormatOgg
		audioDescription.ChannelCount = uint8(audio.Channels)
		audioDescription.Input = fmt.Sprintf("ffmpeg %s-i %s -map 0:a:0 -vn -c:a copy -page_duration 20000 -f ogg -v error pipe:1", inputFlags.String(), quotedPath)
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
	width, height := 1280, 720
	if video != nil {
		width, height = fitVideoResolution(video.Width, video.Height)
	}

	videoFormat := ntgcalls.MediaFormatI420
	videoDescription := &ntgcalls.VideoDescription{
		MediaSource: ntgcalls.MediaSourceShell,
		Width:       int16(width),
		Height:      int16(height),
		Fps:         30,
		Format:      &videoFormat,
	}
	if canPassthrough && canCopyVideo(video, width, height, int(videoDescription.Fps)) {
		videoFormat = ntgcalls.MediaFormatMatroska
		videoDescription.Input = fmt.Sprintf("ffmpeg %s-i %s -map 0:v:0 -an -c:v copy -f matroska -v error pipe:1", inputFlags.String(), quotedPath)
	} else {
		videoDescription.Input = fmt.Sprintf("ffmpeg %s-i %s %s-f rawvideo -r %d -pix_fmt yuv420p -vf scale=%d:%d -v quiet pipe:1",
			inputFlags.String(), quotedPath, filterFlagsWithSpace(filterFlags), videoDescription.Fps, videoDescription.Width, videoDescription.Height)
	}

	return ntgcalls.MediaDescription{
		Microphone: audioDescription,
		Camera:     videoDescription,
	}
}

func nativeMediaDescription(path string, isVideo bool, info ntgcalls.MediaInfo) (ntgcalls.MediaDescription, bool) {
	if info.Audio == nil || !info.Audio.Supported || (isVideo && (info.Video == nil || !info.Video.Supported)) {
		return ntgcalls.MediaDescription{}, false
	}
	format := ntgcalls.MediaFormatAuto
	bitrate := opusBitrate
	desc := ntgcalls.MediaDescription{
		Microphone: &ntgcalls.AudioDescription{
			MediaSource:  ntgcalls.MediaSourceFile,
			Input:        path,
			SampleRate:   48000,
			ChannelCount: 2,
			Bitrate:      &bitrate,
			Format:       &format,
		},
	}
	if info.Audio.Passthrough && (info.Audio.Channels == 1 || info.Audio.Channels == 2) {
		desc.Microphone.ChannelCount = info.Audio.Channels
	}
	if isVideo {
		width, height := fitVideoResolution(int(info.Video.Width), int(info.Video.Height))
		desc.Camera = &ntgcalls.VideoDescription{
			MediaSource: ntgcalls.MediaSourceFile,
			Input:       path,
			Width:       int16(width),
			Height:      int16(height),
			Fps:         30,
			Format:      &format,
		}
	}
	return desc, true
}

func fitVideoResolution(sourceWidth, sourceHeight int) (int, int) {
	width, height := 1280, 720
	if sourceWidth <= 0 || sourceHeight <= 0 {
		return width, height
	}
	ratio := float64(sourceWidth) / float64(sourceHeight)
	width = min(sourceWidth, width)
	height = int(float64(width) / ratio)
	if height > 720 {
		height = 720
		width = int(float64(height) * ratio)
	}
	return width &^ 1, height &^ 1
}

func filterFlagsWithSpace(flags string) string {
	if flags != "" {
		return flags + " "
	}
	return ""
}
