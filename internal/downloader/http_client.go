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
	"context"
	"crypto/rand"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"math/big"
	"net"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"
)

const (
	defaultRequestTimeout = 30 * time.Second
	defaultConnectTimeout = 15 * time.Second
	maxRetries            = 2
	initialBackoff        = 1 * time.Second
)

var client = &http.Client{
	Timeout: defaultRequestTimeout,
	Transport: &http.Transport{
		TLSHandshakeTimeout: defaultConnectTimeout,
		TLSClientConfig: &tls.Config{
			MinVersion: tls.VersionTLS12,
		},

		ResponseHeaderTimeout: defaultRequestTimeout,
		ExpectContinueTimeout: 1 * time.Second,
		DialContext: (&net.Dialer{
			Timeout:   defaultConnectTimeout,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		IdleConnTimeout:     90 * time.Second,
		MaxIdleConns:        100,
		MaxIdleConnsPerHost: 10,
		MaxConnsPerHost:     20,
		DisableCompression:  false,
		ForceAttemptHTTP2:   true,
	},
	CheckRedirect: func(req *http.Request, via []*http.Request) error {
		if len(via) >= 2 {
			return fmt.Errorf("too many redirects (%d)", len(via))
		}
		return nil
	},
}

// sendRequest retries transient HTTP failures and leaves successful responses to the caller to close.
func sendRequest(method, fullURL string, headers map[string]string) (*http.Response, error) {
	baseReq, err := http.NewRequest(method, fullURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create base request: %w", err)
	}

	baseReq.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36")
	baseReq.Header.Set("Accept", "*/*")

	for k, v := range headers {
		baseReq.Header.Set(k, v)
	}

	var reqErr error

	for attempt := range maxRetries {
		if attempt > 0 {
			time.Sleep(initialBackoff * time.Duration(1<<(attempt-1)))
		}

		ctx, cancel := context.WithTimeout(context.Background(), defaultRequestTimeout)
		req, err := http.NewRequestWithContext(ctx, method, fullURL, nil)
		if err != nil {
			cancel()
			reqErr = err
			break
		}
		req.Header = baseReq.Header.Clone()

		var resp *http.Response
		resp, reqErr = client.Do(req)
		if reqErr == nil {
			if resp.StatusCode < 500 {
				resp.Body = &cancelOnClose{ReadCloser: resp.Body, cancel: cancel}
				return resp, nil
			}
			_ = resp.Body.Close()
			cancel()
			reqErr = fmt.Errorf("unexpected status code: %d", resp.StatusCode)
		} else {
			cancel()
			if !isTemporaryError(reqErr) {
				break
			}
		}
		if attempt+1 < maxRetries {
			if resp != nil {
				log.Warnf("request to %s returned HTTP %d on attempt %d/%d; retrying", baseReq.URL.Host, resp.StatusCode, attempt+1, maxRetries)
			} else {
				log.Warnf("temporary request to %s failed on attempt %d/%d (%T); retrying", baseReq.URL.Host, attempt+1, maxRetries, reqErr)
			}
		}
	}

	if reqErr == nil {
		reqErr = fmt.Errorf("request failed after %d attempts", maxRetries)
	}

	if urlErr, ok := errors.AsType[*url.Error](reqErr); ok {
		reqErr = fmt.Errorf("%s %s: %w", method, baseReq.URL.Host, urlErr.Err)
	}
	log.Warnf("request to %s failed: %s", baseReq.URL.Host, maskSensitiveInfo(reqErr.Error()))
	if config.ApiKey != "" && strings.Contains(reqErr.Error(), config.ApiKey) {
		return nil, fmt.Errorf("request failed: %s", maskSensitiveInfo(reqErr.Error()))
	}
	return nil, fmt.Errorf("request failed: %w", reqErr)
}

// maskSensitiveInfo removes the API key from error messages.
func maskSensitiveInfo(msg string) string {
	if config.ApiKey == "" {
		return msg
	}
	return strings.ReplaceAll(msg, config.ApiKey, "REDACTED")
}

// isTemporaryError determines if an error is temporary and thus worth retrying.
func isTemporaryError(err error) bool {
	if netErr, ok := errors.AsType[net.Error](err); ok {
		return netErr.Timeout() || netErr.Temporary()
	}
	return false
}

// generateUniqueName creates a pseudo-random filename using a combination of the current timestamp and a random number.
func generateUniqueName(ext string) (string, error) {
	n, err := rand.Int(rand.Reader, big.NewInt(99999))
	if err != nil {
		return "", fmt.Errorf("generate download filename: %w", err)
	}
	return fmt.Sprintf("%d_%05d%s", time.Now().UnixNano(), n.Int64(), ext), nil
}

// determineFilename safely determines a valid filename for a download.
func determineFilename(urlStr, contentDisp string) (string, error) {
	if filename := extractFilename(contentDisp); filename != "" {
		if sanitized := sanitizeFilename(filename); sanitized != "" && sanitized != "." {
			return filepath.Join(config.DownloadsDir, sanitized), nil
		}
	}

	if parsedURL, err := url.Parse(urlStr); err == nil {
		filename := path.Base(parsedURL.Path)
		if sanitized := sanitizeFilename(filename); sanitized != "" && sanitized != "." && sanitized != "/" {
			return filepath.Join(config.DownloadsDir, sanitized), nil
		}
	}

	name, err := generateUniqueName(".tmp")
	if err != nil {
		return "", err
	}
	return filepath.Join(config.DownloadsDir, name), nil
}

// downloadFile downloads a file from a URL and saves it to a local path.
func downloadFile(urlStr, fileName string, overwrite bool) (string, error) {
	if urlStr == "" {
		return "", errors.New("an empty URL was provided")
	}

	ctx, cancel := context.WithTimeout(context.Background(), downloadTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, urlStr, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create the request: %w", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("the request failed: %w", err)
	}
	defer func(Body io.ReadCloser) {
		_ = Body.Close()
	}(resp.Body)

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("unexpected status code received: %d", resp.StatusCode)
	}

	if fileName == "" {
		fileName, err = determineFilename(urlStr, resp.Header.Get("Content-Disposition"))
		if err != nil {
			return "", err
		}
	}

	if !overwrite {
		if _, err := os.Stat(fileName); err == nil {
			return fileName, nil
		} else if !errors.Is(err, os.ErrNotExist) {
			return "", fmt.Errorf("inspect existing file: %w", err)
		}
	}

	if err := os.MkdirAll(filepath.Dir(fileName), defaultDownloadDirPerm); err != nil {
		return "", fmt.Errorf("failed to create the directory: %w", err)
	}

	temp, err := os.CreateTemp(filepath.Dir(fileName), ".download-*.part")
	if err != nil {
		return "", fmt.Errorf("create temporary download: %w", err)
	}
	defer os.Remove(temp.Name())
	if _, err := io.Copy(temp, resp.Body); err != nil {
		_ = temp.Close()
		return "", fmt.Errorf("write temporary download: %w", err)
	}
	if err := temp.Close(); err != nil {
		return "", fmt.Errorf("close temporary download: %w", err)
	}
	if err := ctx.Err(); err != nil {
		return "", err
	}
	if !overwrite {
		if _, err := os.Stat(fileName); err == nil {
			return fileName, nil
		} else if !errors.Is(err, os.ErrNotExist) {
			return "", fmt.Errorf("inspect existing file: %w", err)
		}
	}

	if err := os.Rename(temp.Name(), fileName); err != nil {
		return "", fmt.Errorf("failed to rename the temporary file: %w", err)
	}

	return fileName, nil
}

// cancelOnClose cancels the request context after the response body is closed.
type cancelOnClose struct {
	io.ReadCloser
	cancel context.CancelFunc
}

func (c *cancelOnClose) Close() error {
	err := c.ReadCloser.Close()
	c.cancel()
	return err
}
