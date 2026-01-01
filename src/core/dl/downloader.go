/*
 * TgMusicBot - Telegram Music Bot
 *  Copyright (c) 2025-2026 Ashok Shau
 *
 *  Licensed under GNU GPL v3
 *  See https://github.com/AshokShau/TgMusicBot
 */

package dl

import (
	"ashokshau/tgmusic/config"
	"ashokshau/tgmusic/src/utils"
	"context"
	"fmt"
	"os"
	"path/filepath"

	tg "github.com/amarnathcjd/gogram/telegram"
)

func exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func DownloadSong(ctx context.Context, cached *utils.CachedTrack, bot *tg.Client) (string, error) {
	if cached.Platform == utils.DirectLink {
		return cached.URL, nil
	}

	if cached.Platform == utils.Telegram {
		return downloadTelegramFile(cached, bot)
	}

	return downloadViaWrapper(ctx, cached, bot)
}

func downloadViaWrapper(ctx context.Context, cached *utils.CachedTrack, bot *tg.Client) (string, error) {
	wrapper := NewDownloaderWrapper(cached.URL)
	if !wrapper.IsValid() {
		return "", fmt.Errorf("invalid cached URL: %s", cached.URL)
	}

	track, err := wrapper.GetTrack(ctx)
	if err != nil {
		return "", fmt.Errorf("get track info: %w", err)
	}

	path, err := wrapper.DownloadTrack(ctx, track, cached.IsVideo)
	if err != nil {
		return "", err
	}

	if utils.TelegramMessageRegex.MatchString(path) {
		return downloadFromTelegramMessage(bot, path)
	}

	return path, nil
}

func downloadTelegramFile(cached *utils.CachedTrack, bot *tg.Client) (string, error) {
	file, err := tg.ResolveBotFileID(cached.TrackID)
	if err != nil {
		return "", fmt.Errorf("resolve telegram file id: %w", err)
	}

	fileName := filepath.Base(cached.Name)
	dst := filepath.Join(config.Conf.DownloadsDir, fileName)

	if exists(dst) {
		return dst, nil
	}

	path, err := bot.DownloadMedia(file, &tg.DownloadOptions{
		FileName: dst,
	})

	if err != nil {
		return "", fmt.Errorf("telegram download failed: %w", err)
	}

	return path, nil
}

func downloadFromTelegramMessage(bot *tg.Client, msgURL string) (string, error) {
	msg, err := utils.GetMessage(bot, msgURL)
	if err != nil {
		return "", fmt.Errorf("get telegram message: %w", err)
	}

	if msg.File == nil {
		return "", fmt.Errorf("telegram message has no downloadable file")
	}

	safeName := filepath.Base(msg.File.Name)
	dst := filepath.Join(config.Conf.DownloadsDir, safeName)
	if exists(dst) {
		return dst, nil
	}

	path, err := msg.Download(&tg.DownloadOptions{
		FileName: dst,
	})
	if err != nil {
		return "", fmt.Errorf("telegram message download failed: %w", err)
	}

	return path, nil
}
