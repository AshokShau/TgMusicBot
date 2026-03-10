/*
 * TgMusicBot - Telegram Music Bot
 *  Copyright (c) 2025-2026 Ashok Shau
 *
 *  Licensed under GNU GPL v3
 *  See https://github.com/AshokShau/TgMusicBot
 */

package dl

import (
	"ashokshau/tgmusic/src/utils"
	"context"
	"fmt"
	"os"

	td "github.com/AshokShau/gotdbot"
)

func exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func DownloadSong(ctx context.Context, cached *utils.CachedTrack, bot *td.Client) (string, error) {
	if cached.Platform == utils.DirectLink {
		return cached.URL, nil
	}

	if cached.Platform == utils.Telegram {
		return downloadTelegramFile(cached, bot)
	}

	return downloadViaWrapper(ctx, cached, bot)
}

func downloadViaWrapper(ctx context.Context, cached *utils.CachedTrack, bot *td.Client) (string, error) {
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

func downloadTelegramFile(cached *utils.CachedTrack, bot *td.Client) (string, error) {
	file, err := bot.GetRemoteFile(cached.TrackID, nil)
	if err != nil {
		return "", err
	}

	download, err := file.Download(bot, 0, 0, 1, &td.DownloadFileOpts{Synchronous: true})
	if err != nil {
		return "", err
	}

	return download.Local.Path, nil
}

func downloadFromTelegramMessage(bot *td.Client, msgURL string) (string, error) {
	msg, err := utils.GetMessage(bot, msgURL)
	if err != nil {
		return "", fmt.Errorf("get telegram message: %w", err)
	}

	download, err := msg.Download(bot, 1, 0, 0, true)
	if err != nil {
		return "", err
	}

	if download == nil || download.Local == nil {
		return "", fmt.Errorf("failed to download file from Telegram message")
	}

	return download.Local.Path, nil
}
