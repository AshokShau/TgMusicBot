/*
 * TgMusicBot - Telegram Music Bot
 *  Copyright (c) 2025 Ashok Shau
 *
 *  Licensed under GNU GPL v3
 *  See https://github.com/AshokShau/TgMusicBot
 */

package vc

import (
	"context"
	"fmt"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"time"

	"ashokshau/tgmusic/src/vc/ntgcalls"

	"github.com/amarnathcjd/gogram/telegram"
)

// handleFlood manages flood wait errors by pausing execution for the specified duration.
// It returns true if a flood wait error is handled, and false otherwise.
func handleFlood(err error) bool {
	if wait := telegram.GetFloodWait(err); wait > 0 {
		logger.Warnf("A flood wait has been detected. Sleeping for %ds.", wait)
		time.Sleep(time.Duration(wait) * time.Second)
		return true
	}
	return false
}

func getVideoDimensions(filePath string) (int, int) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "ffprobe", "-v", "error", "-select_streams", "v:0", "-show_entries", "stream=width,height", "-of", "csv=s=x:p=0", filePath)
	out, err := cmd.Output()
	if err != nil {
		logger.Warnf("[getVideoDimensions] Failed to get video dimensions (%s): %v", filePath, err)
		return 0, 0
	}
	dimensions := strings.Split(strings.TrimSpace(string(out)), "x")
	if len(dimensions) != 2 {
		logger.Warnf("[getVideoDimensions] Invalid video dimensions(%s): %s", filePath, string(out))
		return 0, 0
	}

	width, _ := strconv.Atoi(dimensions[0])
	height, _ := strconv.Atoi(dimensions[1])
	return width, height
}

var isURLRegex = regexp.MustCompile(`^https?://`)

// getMediaDescription creates a media description for ntgcalls based on the provided file path, video status, and ffmpeg parameters.
func getMediaDescription(filePath string, isVideo bool, ffmpegParameters string) ntgcalls.MediaDescription {
	audioDescription := &ntgcalls.AudioDescription{
		MediaSource:  ntgcalls.MediaSourceShell,
		SampleRate:   96000,
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

// UpdateMembership updates the membership status of a user in a specific chat.
func (c *TelegramCalls) UpdateMembership(chatId, userId int64, status string) {
	cacheKey := fmt.Sprintf("%d:%d", chatId, userId)
	if c.statusCache != nil {
		c.statusCache.Set(cacheKey, status)
		logger.Info("[UpdateMembership] The cache has been updated: chat=%d user=%d status=%s", chatId, userId, status)
	}
}

// UpdateInviteLink updates the invite link for a specific chat.
func (c *TelegramCalls) UpdateInviteLink(chatId int64, link string) {
	cacheKey := fmt.Sprintf("%d", chatId)
	c.inviteCache.Set(cacheKey, link)
}
