/*
 * TgMusicBot - Telegram Music Bot
 *  Copyright (c) 2025-2026 Ashok Shau
 *
 *  Licensed under GNU GPL v3
 *  See https://github.com/AshokShau/TgMusicBot
 */

package downloader

import (
	"ashokshau/tgmusic/internal/config"
	"ashokshau/tgmusic/internal/utils"
	"context"
	"crypto/aes"
	"crypto/cipher"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
)

const (
	defaultFilePerm = 0644
)

var (
	errMissingKey    = errors.New("missing CDN key")
	errFileNotFound  = errors.New("file not found")
	errInvalidHexKey = errors.New("invalid hex key")
	errInvalidAESIV  = errors.New("invalid AES IV")
)

func processSpotify(track *utils.TrackInfo) (string, error) {
	downloadsDir := config.DownloadsDir
	sanitizedTrackID := filepath.Base(track.Id)

	outputFile := filepath.Join(downloadsDir, fmt.Sprintf("%s.ogg", sanitizedTrackID))
	if _, err := os.Stat(outputFile); err == nil {
		return outputFile, nil
	}

	if track.Key == "" {
		log.Errorf("Spotify track %s has no CDN decryption key", sanitizedTrackID)
		return "", errMissingKey
	}

	encryptedFile := filepath.Join(downloadsDir, fmt.Sprintf("%s.encrypted", sanitizedTrackID))
	decryptedFile := filepath.Join(downloadsDir, fmt.Sprintf("%s_decrypted.ogg", sanitizedTrackID))

	defer func() {
		_ = os.Remove(encryptedFile)
		_ = os.Remove(decryptedFile)
	}()

	if err := downloadAndDecrypt(track, encryptedFile, decryptedFile); err != nil {
		log.Errorf("Spotify download or decrypt failed for track %s: %v", sanitizedTrackID, err)
		return "", err
	}

	if err := rebuildOGG(decryptedFile); err != nil {
		log.Errorf("rebuild Spotify OGG headers for track %s: %v", sanitizedTrackID, err)
		return "", fmt.Errorf("rebuild OGG: %w", err)
	}
	if err := utils.RepairOGG(decryptedFile, outputFile); err != nil {
		log.Errorf("repair Spotify OGG for track %s: %v", sanitizedTrackID, err)
		return "", fmt.Errorf("repair OGG: %w", err)
	}
	return outputFile, nil
}

func downloadAndDecrypt(track *utils.TrackInfo, encryptedPath, decryptedPath string) error {
	ctx, cancel := context.WithTimeout(context.Background(), downloadTimeout)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, track.CdnURL, nil)
	if err != nil {
		return fmt.Errorf("create CDN request: %w", err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to download the file: %w", err)
	}

	defer func(Body io.ReadCloser) {
		_ = Body.Close()
	}(resp.Body)

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read the response body: %w", err)
	}

	if err := os.WriteFile(encryptedPath, data, defaultFilePerm); err != nil {
		return fmt.Errorf("failed to write the encrypted file: %w", err)
	}

	decryptedData, err := decryptAudioFile(encryptedPath, track.Key)
	if err != nil {
		return fmt.Errorf("failed to decrypt the audio file: %w", err)
	}

	return os.WriteFile(decryptedPath, decryptedData, defaultFilePerm)
}

// decryptAudioFile decrypts an audio file using AES-CTR encryption.
func decryptAudioFile(filePath, hexKey string) ([]byte, error) {
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return nil, fmt.Errorf("%w: %s", errFileNotFound, filePath)
	}

	key, err := hex.DecodeString(hexKey)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", errInvalidHexKey, err)
	}

	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read the file: %w", err)
	}

	audioAesIv, err := hex.DecodeString("72e067fbddcbcf77ebe8bc643f630d93")
	if err != nil {
		return nil, fmt.Errorf("%w: %v", errInvalidAESIV, err)
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("failed to create the AES cipher: %w", err)
	}

	ctr := cipher.NewCTR(block, audioAesIv)
	decrypted := make([]byte, len(data))
	ctr.XORKeyStream(decrypted, data)

	return decrypted, nil
}

// rebuildOGG reconstructs the OGG header of a given file by patching specific offsets.
func rebuildOGG(filename string) error {
	file, err := os.OpenFile(filename, os.O_RDWR, defaultFilePerm)
	if err != nil {
		return fmt.Errorf("error opening the file: %w", err)
	}
	defer func(file *os.File) {
		_ = file.Close()
	}(file)

	writeAt := func(offset int64, data string) error {
		_, err := file.WriteAt([]byte(data), offset)
		return err
	}

	patches := map[int64]string{
		0:  "OggS",
		6:  "\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00",
		26: "\x01\x1E\x01vorbis",
		39: "\x02",
		40: "\x44\xAC\x00\x00",
		48: "\x00\xE2\x04\x00",
		56: "\xB8\x01",
		58: "OggS",
		62: "\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00",
	}

	for offset, data := range patches {
		if err := writeAt(offset, data); err != nil {
			return fmt.Errorf("failed to write at offset %d: %w", offset, err)
		}
	}

	return nil
}
