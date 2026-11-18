package disk

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/AshokShau/gotdbot/logger"
)

const maxIdle = 2 * time.Hour

var log = logger.New()
var mu sync.Mutex
var downloads int
var pending = make(map[string]int)
var playing = make(map[int64]string)

func Downloading() func() {
	mu.Lock()
	downloads++
	mu.Unlock()
	var once sync.Once
	return func() {
		once.Do(func() {
			mu.Lock()
			downloads--
			mu.Unlock()
		})
	}
}

func Hold(path string) func() {
	path = localPath(path)
	if path == "" {
		return func() {}
	}
	mu.Lock()
	pending[path]++
	mu.Unlock()
	var once sync.Once
	return func() {
		once.Do(func() {
			mu.Lock()
			pending[path]--
			if pending[path] == 0 {
				delete(pending, path)
			}
			mu.Unlock()
		})
	}
}

func SetPlaying(chatID int64, path string) {
	path = localPath(path)
	mu.Lock()
	if previous := playing[chatID]; previous != "" && previous != path {
		_ = os.Chtimes(previous, time.Now(), time.Now())
	}
	if path == "" {
		delete(playing, chatID)
	} else {
		playing[chatID] = path
	}
	mu.Unlock()
}

func Fresh(path string) {
	if path == "" || strings.Contains(path, "://") {
		return
	}
	mu.Lock()
	_ = os.Chtimes(path, time.Now(), time.Now())
	mu.Unlock()
}

func localPath(path string) string {
	if path == "" || strings.Contains(path, "://") {
		return ""
	}
	absolute, err := filepath.Abs(path)
	if err != nil {
		return ""
	}
	resolved, err := filepath.EvalSymlinks(absolute)
	if err == nil {
		return resolved
	}
	return absolute
}

func Start(ctx context.Context, directory string) {
	go func() {
		sweep(directory)
		ticker := time.NewTicker(10 * time.Minute)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				sweep(directory)
			}
		}
	}()
}

func sweep(directory string) {
	root, err := filepath.Abs(directory)
	if err != nil {
		log.Warn("disk cleanup: invalid download directory", "error", err)
		return
	}
	root, err = filepath.EvalSymlinks(root)
	if err != nil {
		log.Warn("disk cleanup: cannot resolve download directory", "error", err)
		return
	}
	workingDir, err := os.Getwd()
	if err != nil {
		log.Warn("disk cleanup: cannot resolve working directory", "error", err)
		return
	}
	if relative, err := filepath.Rel(root, workingDir); err == nil && (relative == "." || relative != ".." && !strings.HasPrefix(relative, ".."+string(filepath.Separator))) {
		log.Warn("disk cleanup: download directory contains the application directory", "directory", root)
		return
	}
	mu.Lock()
	defer mu.Unlock()
	if downloads != 0 {
		return
	}

	protected := make(map[string]bool, len(pending)+len(playing))
	for path := range pending {
		protected[path] = true
	}
	for _, path := range playing {
		protected[path] = true
	}
	cutoff := time.Now().Add(-maxIdle)
	var removed int
	var bytesFreed int64
	err = filepath.WalkDir(root, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			log.Warn("disk cleanup: cannot inspect path", "path", path, "error", walkErr)
			return nil
		}
		if entry.IsDir() || !entry.Type().IsRegular() || protected[path] {
			return nil
		}
		info, err := entry.Info()
		if err != nil || info.ModTime().After(cutoff) {
			return nil
		}
		if err := os.Remove(path); err != nil {
			log.Warn("disk cleanup: cannot remove file", "path", path, "error", err)
		} else {
			removed++
			bytesFreed += info.Size()
		}
		return nil
	})
	if err != nil {
		log.Warn("disk cleanup: scan failed", "directory", root, "error", err)
	}
	if removed > 0 {
		log.Info("disk cleanup completed", "files", removed, "bytes", bytesFreed)
	}
}
