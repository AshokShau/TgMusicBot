//go:build ignore

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
)

// Release represents a GitHub release.
type Release struct {
	TagName string `json:"tag_name"` // TagName is the name of the tag.
	Assets  []struct {
		Name               string `json:"name"`                 // Name is the name of the asset.
		BrowserDownloadURL string `json:"browser_download_url"` // BrowserDownloadURL is the URL to download the asset.
	} `json:"assets"`
}

// main is the entry point for the ntgcalls setup script.
// It downloads and extracts the latest ntgcalls library from GitHub.
func main() {
	buildType := "shared"
	if len(os.Args) > 1 {
		buildType = os.Args[1]
	}

	release := getLatestRelease()
	targetAsset := pickAsset(release, buildType)
	if targetAsset == "" {
		fmt.Println("No matching asset found.")
		return
	}

	fmt.Println("Downloading:", targetAsset)
	tmpZip := "ntgcalls.zip"
	downloadFile(tmpZip, targetAsset)

	fmt.Println("Extracting...")
	unzip(tmpZip, "ntgcalls_tmp")

	destHeader := "pkg/vc/ntgcalls"
	destLib := "pkg/vc"
	os.MkdirAll(destHeader, 0755)
	os.MkdirAll(destLib, 0755)

	filepath.Walk("ntgcalls_tmp", func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}
		name := filepath.Base(path)
		switch {
		case name == "ntgcalls.h":
			copyFile(path, filepath.Join(destHeader, name))
		case strings.HasPrefix(name, "libntgcalls.") || strings.HasPrefix(name, "ntgcalls."):
			copyFile(path, filepath.Join(destLib, name))
		}
		return nil
	})

	fmt.Println("✅ Done!")
	os.RemoveAll("ntgcalls_tmp")
	os.Remove(tmpZip)
}

// getLatestRelease fetches the latest release information from the ntgcalls GitHub repository.
// It returns a Release object.
func getLatestRelease() Release {
	resp, err := http.Get("https://api.github.com/repos/pytgcalls/ntgcalls/releases/latest")
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()
	var r Release
	if err := json.NewDecoder(resp.Body).Decode(&r); err != nil {
		panic(err)
	}
	return r
}

// pickAsset selects the appropriate asset from a release based on the operating system, architecture, and build type.
// It takes a Release object and a build type as input.
// It returns the URL of the selected asset.
func pickAsset(r Release, buildType string) string {
	goos := runtime.GOOS
	arch := runtime.GOARCH

	if arch == "amd64" {
		arch = "x86_64"
	}

	search := fmt.Sprintf("ntgcalls.%s-%s-%s_libs.zip", goos, arch, buildType)

	for _, a := range r.Assets {
		if strings.Contains(a.Name, search) {
			return a.BrowserDownloadURL
		}
	}
	fmt.Println("Search pattern:", search)
	return ""
}

// downloadFile downloads a file from a URL and saves it to a local file.
// It takes a filename and a URL as input.
func downloadFile(filename, url string) {
	resp, err := http.Get(url)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()
	out, err := os.Create(filename)
	if err != nil {
		panic(err)
	}
	defer out.Close()
	io.Copy(out, resp.Body)
}

// unzip extracts a zip archive to a destination directory.
// It takes a source zip file and a destination directory as input.
func unzip(src, dest string) {
	r, err := zip.OpenReader(src)
	if err != nil {
		panic(err)
	}
	defer r.Close()
	os.MkdirAll(dest, 0755)
	for _, f := range r.File {
		fp := filepath.Join(dest, f.Name)
		if f.FileInfo().IsDir() {
			os.MkdirAll(fp, f.Mode())
			continue
		}
		os.MkdirAll(filepath.Dir(fp), 0755)
		rc, err := f.Open()
		if err != nil {
			panic(err)
		}
		out, err := os.Create(fp)
		if err != nil {
			panic(err)
		}
		io.Copy(out, rc)
		rc.Close()
		out.Close()
	}
}

// copyFile copies a file from a source path to a destination path.
// It takes a source path and a destination path as input.
func copyFile(src, dst string) {
	in, err := os.Open(src)
	if err != nil {
		panic(err)
	}
	defer in.Close()
	out, err := os.Create(dst)
	if err != nil {
		panic(err)
	}
	defer out.Close()
	io.Copy(out, in)
}
