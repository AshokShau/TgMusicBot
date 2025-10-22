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
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/AshokShau/TgMusicBot/pkg/config"
	"github.com/AshokShau/TgMusicBot/pkg/core/cache"
	"github.com/AshokShau/TgMusicBot/pkg/core/dl"
	"github.com/AshokShau/TgMusicBot/pkg/vc/ntgcalls"

	"github.com/Laky-64/gologging"
	"github.com/amarnathcjd/gogram/telegram"
)

func getVideoDimensions(filePath string) (int, int) {
	cmd := exec.Command("ffprobe", "-v", "error", "-select_streams", "v:0", "-show_entries", "stream=width,height", "-of", "csv=s=x:p=0", filePath)
	out, err := cmd.Output()
	if err != nil {
		return 0, 0
	}
	dimensions := strings.Split(strings.TrimSpace(string(out)), "x")
	if len(dimensions) != 2 {
		return 0, 0
	}
	width, _ := strconv.Atoi(dimensions[0])
	height, _ := strconv.Atoi(dimensions[1])
	return width, height
}

// getMediaDescription creates a media description for ntgcalls based on the provided file path, video status, and ffmpeg parameters.
func getMediaDescription(filePath string, isVideo bool, ffmpegParameters string) ntgcalls.MediaDescription {
	audioDescription := &ntgcalls.AudioDescription{
		MediaSource:  ntgcalls.MediaSourceShell,
		SampleRate:   96000,
		ChannelCount: 2,
	}

	quotedPath := fmt.Sprintf("\"%s\"", filePath)
	isURL := regexp.MustCompile(`^https?://`).MatchString(filePath)

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

// DownloadSong downloads a song using the provided cached track information.
// It returns the file path, track information, and an error if the download fails.
func DownloadSong(ctx context.Context, song *cache.CachedTrack, bot *telegram.Client) (string, *cache.TrackInfo, error) {
	if song.Platform == cache.Telegram {
		file, err := telegram.ResolveBotFileID(song.TrackID)
		if err != nil {
			return "", nil, err
		}

		filePath, err := bot.DownloadMedia(file, &telegram.DownloadOptions{FileName: filepath.Join(config.Conf.DownloadsDir, song.Name)})
		return filePath, nil, err
	}

	songUrl := song.URL
	wrapper := dl.NewDownloaderWrapper(songUrl)

	if wrapper.IsValid() {
		trackInfo, err := wrapper.GetTrack(ctx)
		if err != nil {
			gologging.InfoF("[DownloadSong] Failed to get track information: %v", err)
			return "", nil, err
		}

		filePath, err := wrapper.DownloadTrack(ctx, trackInfo, song.IsVideo)
		reg := regexp.MustCompile(`t\.me/(\w+)/(\d+)`)
		if match := reg.FindStringSubmatch(filePath); match != nil {
			msg, err := dl.GetMessage(bot, filePath)
			if err != nil {
				return "", &trackInfo, fmt.Errorf("failed to get the message for %s: %w", trackInfo.Name, err)
			}

			fileName := msg.File.Name
			download, err := msg.Download(&telegram.DownloadOptions{FileName: filepath.Join(config.Conf.DownloadsDir, fileName)})
			if err != nil {
				return "", &trackInfo, fmt.Errorf("failed to download %s: %w", trackInfo.Name, err)
			}

			if trackInfo.Duration == 0 {
				trackInfo.Duration = cache.GetFileDur(msg)
			}

			return download, &trackInfo, nil
		}

		return filePath, &trackInfo, err
	}

	return "", nil, fmt.Errorf("the provided song URL is invalid: %s", songUrl)
}

// UpdateMembership updates the membership status of a user in a specific chat.
func (c *TelegramCalls) UpdateMembership(chatId, userId int64, status string) {
	cacheKey := fmt.Sprintf("%d:%d", chatId, userId)
	if c.statusCache != nil {
		c.statusCache.Set(cacheKey, status)
		gologging.InfoF("[UpdateMembership] The cache has been updated: chat=%d user=%d status=%s", chatId, userId, status)
	}
}

// UpdateInviteLink updates the invite link for a specific chat.
func (c *TelegramCalls) UpdateInviteLink(chatId int64, link string) {
	cacheKey := fmt.Sprintf("%d", chatId)
	c.inviteCache.Set(cacheKey, link)
}
