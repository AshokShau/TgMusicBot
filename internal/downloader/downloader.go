/*
 * TgMusicBot - Telegram Music Bot
 *  Copyright (c) 2025-2026 Ashok Shau
 *
 *  Licensed under GNU GPL v3
 *  See https://github.com/AshokShau/TgMusicBot
 */

package downloader

import (
	"ashokshau/tgmusic/internal/disk"
	"ashokshau/tgmusic/internal/utils"
	"fmt"

	td "github.com/AshokShau/gotdbot"
)

func DlCachedTrack(cached *utils.PlayerCache, bot *td.Client, message *td.Message) (path string, err error) {
	if cached.Platform == utils.DirectLink {
		return cached.URL, nil
	}
	finished := disk.Downloading()
	defer func() {
		if err == nil {
			disk.Fresh(path)
		}
		finished()
	}()

	if cached.Platform == utils.Telegram {
		if message != nil {
			return downloadTelegramMessage(message, bot)
		}
		return downloadTelegramFile(cached, bot)
	}

	dlBot := bot
	if DlBot != nil {
		dlBot = DlBot
	}

	return downloadViaWrapper(cached, dlBot)
}

func downloadViaWrapper(cached *utils.PlayerCache, dlBot *td.Client) (string, error) {
	wrapper := NewDlWrapper(cached.URL)
	if !wrapper.IsValid() {
		return "", fmt.Errorf("invalid cached URL: %s", cached.URL)
	}

	track, err := wrapper.GetTrack()
	if err != nil {
		return "", fmt.Errorf("get track info: %w", err)
	}

	path, err := wrapper.DownloadTrack(track, cached.IsVideo)
	if err != nil {
		return "", err
	}

	if utils.TelegramMessageRegex.MatchString(path) {
		return downloadFromTelegramMessage(dlBot, path)
	}

	return path, nil
}

func downloadTelegramFile(cached *utils.PlayerCache, bot *td.Client) (string, error) {
	file, err := bot.GetRemoteFile(cached.TrackID, nil)
	if err != nil {
		return "", err
	}
	if file == nil {
		return "", fmt.Errorf("Telegram file was not found")
	}

	d, err := file.Download(bot, 1, &td.DownloadFileOpts{Synchronous: true})
	if err != nil {
		return "", err
	}
	if d == nil || d.Local == nil || d.Local.Path == "" {
		return "", fmt.Errorf("Telegram download returned no local file")
	}

	return d.Local.Path, nil
}

func downloadFromTelegramMessage(bot *td.Client, msgURL string) (string, error) {
	msg, err := utils.GetMessage(bot, msgURL)
	if err != nil {
		return "", fmt.Errorf("get telegram message: %w", err)
	}
	return downloadTelegramMessage(msg, bot)
}

func downloadTelegramMessage(msg *td.Message, bot *td.Client) (string, error) {
	file, err := msg.Download(bot, 1, &td.DownloadFileOpts{Synchronous: true})
	if err != nil {
		return "", err
	}

	if file == nil || file.Local == nil || file.Local.Path == "" {
		return "", fmt.Errorf("failed to download file from Telegram message")
	}

	return file.Local.Path, nil
}
