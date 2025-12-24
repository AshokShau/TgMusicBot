//go:build ignore

/*
 * TgMusicBot - Telegram Music Bot
 *  Copyright (c) 2025 Ashok Shau
 *
 *  Licensed under GNU GPL v3
 *  See https://github.com/AshokShau/TgMusicBot
 */

// Usage: go run setup_ntgcalls.go [shared|static]
package main

import (
	"archive/zip"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

const (
	destHeader = "src/vc/ntgcalls"
	destLib    = "src/vc"
	releaseUrl = "https://api.github.com/repos/AshokShau/ntgcalls/releases/latest"
)

type Release struct {
	TagName string `json:"tag_name"`
	Assets  []struct {
		Name               string `json:"name"`
		BrowserDownloadURL string `json:"browser_download_url"`
	} `json:"assets"`
}

func main() {
	if len(os.Args) > 1 && (os.Args[1] == "-h" || os.Args[1] == "--help") {
		fmt.Println("Usage: go run setup_ntgcalls.go [static|shared]")
		fmt.Println("  static: Download static library build")
		fmt.Println("  shared: Download shared library build (default)")
		return
	}

	start := time.Now()
	defer func() {
		fmt.Printf("\nTime elapsed: %v\n", time.Since(start))
	}()

	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "\nError: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("\nSetup completed successfully!")
}

func run() error {
	buildType := "shared"
	if len(os.Args) > 1 {
		buildType = os.Args[1]
		if buildType != "shared" && buildType != "static" {
			return fmt.Errorf("invalid build type: %s (use 'shared' or 'static')", buildType)
		}
	}

	fmt.Printf("Looking for %s/%s %s build...\n", runtime.GOOS, runtime.GOARCH, buildType)
	release, err := getLatestRelease()
	if err != nil {
		return fmt.Errorf("failed to get latest release: %w", err)
	}

	fmt.Printf("Latest release: %s\n", release.TagName)
	targetAsset := pickAsset(release, buildType)
	if targetAsset == "" {
		return fmt.Errorf("no matching asset found for your platform")
	}

	fmt.Printf("Downloading: %s\n", filepath.Base(targetAsset))
	tmpZip := "ntgcalls.zip"
	if err := downloadFile(tmpZip, targetAsset); err != nil {
		return fmt.Errorf("download failed: %w", err)
	}
	defer os.Remove(tmpZip)

	tmpDir := "ntgcalls_tmp"
	fmt.Println("Extracting...")
	if err := unzip(tmpZip, tmpDir); err != nil {
		return fmt.Errorf("extraction failed: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	return organizeFiles(tmpDir)
}

func getLatestRelease() (Release, error) {
	var r Release

	resp, err := http.Get(releaseUrl)
	if err != nil {
		return r, fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return r, fmt.Errorf("GitHub API returned: %s", resp.Status)
	}

	if err := json.NewDecoder(resp.Body).Decode(&r); err != nil {
		return r, fmt.Errorf("failed to decode JSON: %w", err)
	}

	return r, nil
}

func pickAsset(r Release, buildType string) string {
	goos := runtime.GOOS
	arch := runtime.GOARCH

	archMap := map[string]string{
		"amd64": "x86_64",
		"arm64": "arm64",
	}

	if mappedArch, ok := archMap[arch]; ok {
		arch = mappedArch
	}

	osMap := map[string]string{
		"darwin":  "macos",
		"windows": "windows",
	}
	if mappedOS, ok := osMap[goos]; ok {
		goos = mappedOS
	}

	pattern := fmt.Sprintf("ntgcalls.%s-%s-%s_libs.zip", goos, arch, buildType)
	for _, asset := range r.Assets {
		if strings.Contains(strings.ToLower(asset.Name), strings.ToLower(pattern)) {
			fmt.Printf("Found: %s\n", asset.Name)
			return asset.BrowserDownloadURL
		}
	}

	return ""
}

func downloadFile(filename, url string) error {
	resp, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download failed with status: %s", resp.Status)
	}

	out, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer out.Close()

	totalSize := resp.ContentLength
	writer := &progressWriter{
		total:   totalSize,
		prefix:  "   Progress: ",
		lastPct: -1,
	}

	_, err = io.Copy(io.MultiWriter(out, writer), resp.Body)
	if err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	if writer.lastPct >= 0 {
		fmt.Println()
	}

	return nil
}

type progressWriter struct {
	total   int64
	written int64
	prefix  string
	lastPct int
}

func (pw *progressWriter) Write(p []byte) (int, error) {
	n := len(p)
	pw.written += int64(n)

	if pw.total > 0 {
		pct := int(float64(pw.written) / float64(pw.total) * 100)
		if pct != pw.lastPct && pct%10 == 0 {
			fmt.Printf("\r%s%d%%", pw.prefix, pct)
			pw.lastPct = pct
		}
	}

	return n, nil
}

func unzip(src, dest string) error {
	r, err := zip.OpenReader(src)
	if err != nil {
		return fmt.Errorf("failed to open zip: %w", err)
	}
	defer r.Close()

	os.MkdirAll(dest, 0755)
	for _, f := range r.File {
		fp := filepath.Join(dest, f.Name)
		if !strings.HasPrefix(fp, filepath.Clean(dest)+string(os.PathSeparator)) {
			return fmt.Errorf("invalid file path: %s", f.Name)
		}

		if f.FileInfo().IsDir() {
			os.MkdirAll(fp, f.Mode())
			continue
		}

		if err := os.MkdirAll(filepath.Dir(fp), 0755); err != nil {
			return fmt.Errorf("failed to create directory: %w", err)
		}

		rc, err := f.Open()
		if err != nil {
			return fmt.Errorf("failed to open file in zip: %w", err)
		}

		out, err := os.OpenFile(fp, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
		if err != nil {
			rc.Close()
			return fmt.Errorf("failed to create file: %w", err)
		}

		_, err = io.Copy(out, rc)
		rc.Close()
		out.Close()

		if err != nil {
			return fmt.Errorf("failed to extract file: %w", err)
		}
	}

	return nil
}

func organizeFiles(tmpDir string) error {
	var filesCopied []string
	err := filepath.Walk(tmpDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return fmt.Errorf("walk error: %w", err)
		}

		if info.IsDir() {
			return nil
		}

		name := filepath.Base(path)
		var dest string

		switch {
		case name == "ntgcalls.h":
			dest = filepath.Join(destHeader, name)
			filesCopied = append(filesCopied, dest)

			if err := copyFile(path, dest); err != nil {
				return fmt.Errorf("failed to copy header: %w", err)
			}

		case strings.HasPrefix(name, "libntgcalls.") || strings.HasPrefix(name, "ntgcalls."):
			dest = filepath.Join(destLib, name)
			filesCopied = append(filesCopied, dest)

			if err := copyFile(path, dest); err != nil {
				return fmt.Errorf("failed to copy library: %w", err)
			}
		}

		return nil
	})

	if err != nil {
		return err
	}

	if len(filesCopied) == 0 {
		return fmt.Errorf("no files were copied - check the zip contents")
	}

	fmt.Println("Files copied:")
	for _, file := range filesCopied {
		relPath, _ := filepath.Rel(".", file)
		fmt.Printf("   ✓ %s\n", relPath)
	}

	return nil
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("failed to open source: %w", err)
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return fmt.Errorf("failed to create destination: %w", err)
	}
	defer out.Close()

	_, err = io.Copy(out, in)
	if err != nil {
		return fmt.Errorf("failed to copy data: %w", err)
	}

	info, err := in.Stat()
	if err == nil {
		os.Chmod(dst, info.Mode())
	}

	return nil
}
