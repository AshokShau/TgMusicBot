package dl

import (
	"crypto/aes"
	"crypto/cipher"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"tgmusic/pkg/config"
	"tgmusic/pkg/core/cache"
	"time"
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

// processSpotify manages the download and decryption of Spotify tracks.
// It returns the file path of the processed track or an error if any step fails.
func (d *Download) processSpotify() (string, error) {
	track := d.Track
	downloadsDir := config.Conf.DownloadsDir

	outputFile := filepath.Join(downloadsDir, fmt.Sprintf("%s.ogg", track.TC))
	if _, err := os.Stat(outputFile); err == nil {
		log.Printf("✅ The file already exists: %s", outputFile)
		return outputFile, nil
	}

	if track.Key == "" {
		return "", errMissingKey
	}

	startTime := time.Now()
	defer func() {
		log.Printf("The process was completed in %s.", time.Since(startTime))
	}()

	encryptedFile := filepath.Join(downloadsDir, fmt.Sprintf("%s.encrypted", track.TC))
	decryptedFile := filepath.Join(downloadsDir, fmt.Sprintf("%s_decrypted.ogg", track.TC))

	defer func() {
		_ = os.Remove(encryptedFile)
		_ = os.Remove(decryptedFile)
	}()

	if err := d.downloadAndDecrypt(encryptedFile, decryptedFile); err != nil {
		log.Printf("Failed to download and decrypt the file: %v", err)
		return "", err
	}

	if err := rebuildOGG(decryptedFile); err != nil {
		log.Printf("Failed to rebuild the OGG headers: %v", err)
	}

	return fixOGG(decryptedFile, track)
}

// downloadAndDecrypt handles the download and decryption of a file.
// It takes the paths for the encrypted and decrypted files and returns an error if any step fails.
func (d *Download) downloadAndDecrypt(encryptedPath, decryptedPath string) error {
	resp, err := http.Get(d.Track.CdnURL)
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

	decryptedData, decryptTime, err := decryptAudioFile(encryptedPath, d.Track.Key)
	if err != nil {
		return fmt.Errorf("failed to decrypt the audio file: %w", err)
	}
	log.Printf("Decryption was completed in %s.", decryptTime)

	return os.WriteFile(decryptedPath, decryptedData, defaultFilePerm)
}

// decryptAudioFile decrypts an audio file using AES-CTR encryption.
// It takes a file path and a hexadecimal key, and returns the decrypted data, decryption time, and any error encountered.
func decryptAudioFile(filePath, hexKey string) ([]byte, string, error) {
	// #nosec G304 - The file path is constructed internally and not from user input.
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return nil, "", fmt.Errorf("%w: %s", errFileNotFound, filePath)
	}

	key, err := hex.DecodeString(hexKey)
	if err != nil {
		return nil, "", fmt.Errorf("%w: %v", errInvalidHexKey, err)
	}

	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, "", fmt.Errorf("failed to read the file: %w", err)
	}

	audioAesIv, err := hex.DecodeString("72e067fbddcbcf77ebe8bc643f630d93")
	if err != nil {
		return nil, "", fmt.Errorf("%w: %v", errInvalidAESIV, err)
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, "", fmt.Errorf("failed to create the AES cipher: %w", err)
	}

	startTime := time.Now()
	ctr := cipher.NewCTR(block, audioAesIv)
	decrypted := make([]byte, len(data))
	ctr.XORKeyStream(decrypted, data)

	return decrypted, fmt.Sprintf("%dms", time.Since(startTime).Milliseconds()), nil
}

// rebuildOGG reconstructs the OGG header of a given file by patching specific offsets.
// This is necessary to make the decrypted file playable.
func rebuildOGG(filename string) error {
	// #nosec G304 - The filename is constructed internally.
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

	// OGG header patch structure.
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

// fixOGG uses ffmpeg to correct any remaining issues in the OGG file, ensuring it is playable.
// It takes the input file path and track information, and returns the final output file path or an error.
func fixOGG(inputFile string, track cache.TrackInfo) (string, error) {
	outputFile := filepath.Join(config.Conf.DownloadsDir, fmt.Sprintf("%s.ogg", track.TC))
	// #nosec G204 - The input file path is trusted as it's generated internally.
	cmd := exec.Command("ffmpeg", "-i", inputFile, "-c", "copy", outputFile)
	if output, err := cmd.CombinedOutput(); err != nil {
		return "", fmt.Errorf("ffmpeg failed with error: %w\nOutput: %s", err, string(output))
	}

	return outputFile, nil
}
