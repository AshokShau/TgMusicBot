/*
 * TgMusicBot - Telegram Music Bot
 * Copyright (c) 2025 Ashok Shau
 *
 * Licensed under GNU GPL v3
 * https://github.com/AshokShau/TgMusicBot
 */

package handlers

import (
	"context"
	"fmt"
	"os"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"time"

	"ashokshau/tgmusic/src/core/db"

	"github.com/amarnathcjd/gogram/telegram"
)

type AppStats struct {
	Uptime     string
	Goroutines int
	GoVersion  string

	AppMemUsed string
	AppHeap    string
	GCCount    uint32
	GCPause    string

	MemLimit  string
	DiskUsed  string
	DiskTotal string
}

func humanBytes(bytes uint64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.2f %ciB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

// Docker / cgroup memory limit
func readContainerMemLimit() uint64 {
	if data, err := os.ReadFile("/sys/fs/cgroup/memory.max"); err == nil {
		val := strings.TrimSpace(string(data))
		if val != "max" {
			if v, err := strconv.ParseUint(val, 10, 64); err == nil {
				return v
			}
		}
	}

	if data, err := os.ReadFile("/sys/fs/cgroup/memory/memory.limit_in_bytes"); err == nil {
		if v, err := strconv.ParseUint(strings.TrimSpace(string(data)), 10, 64); err == nil && v < (1<<60) {
			return v
		}
	}
	return 0
}

// Disk usage using syscall
func diskUsage(path string) (used, total string) {
	var stat syscall.Statfs_t
	if err := syscall.Statfs(path, &stat); err != nil {
		return "N/A", "N/A"
	}

	totalBytes := stat.Blocks * uint64(stat.Bsize)
	freeBytes := stat.Bfree * uint64(stat.Bsize)
	usedBytes := totalBytes - freeBytes

	return humanBytes(usedBytes), humanBytes(totalBytes)
}

func appMemoryStats() (used, heap string, gc uint32, pause string) {
	var ms runtime.MemStats
	runtime.ReadMemStats(&ms)

	return humanBytes(ms.Alloc),
		humanBytes(ms.HeapAlloc),
		ms.NumGC,
		(time.Duration(ms.PauseTotalNs) * time.Nanosecond).String()
}

func gatherAppStats() *AppStats {
	memUsed, heap, gcCount, gcPause := appMemoryStats()

	root := "/"
	if runtime.GOOS == "windows" {
		root = "C:\\"
	}

	dUsed, dTotal := diskUsage(root)

	stats := &AppStats{
		Uptime:     time.Since(startTime).Round(time.Second).String(),
		Goroutines: runtime.NumGoroutine(),
		GoVersion:  runtime.Version(),

		AppMemUsed: memUsed,
		AppHeap:    heap,
		GCCount:    gcCount,
		GCPause:    gcPause,

		DiskUsed:  dUsed,
		DiskTotal: dTotal,
	}

	if limit := readContainerMemLimit(); limit > 0 {
		stats.MemLimit = humanBytes(limit)
	}

	return stats
}

func sysStatsHandler(msg *telegram.NewMessage) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	sysMsg, err := msg.Reply("📊 Collecting system statistics...")
	if err != nil {
		return err
	}

	stats := gatherAppStats()

	chats, _ := db.Instance.GetAllChats(ctx)
	users, _ := db.Instance.GetAllUsers(ctx)

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf(
		"📊 <b>%s — Runtime Status</b>\n",
		msg.Client.Me().FirstName,
	))
	sb.WriteString(strings.Repeat("─", 36) + "\n\n")

	sb.WriteString("🤖 <b>Application</b>\n")
	sb.WriteString(fmt.Sprintf(
		"• Uptime: %s\n• Goroutines: %d\n• Go Version: %s\n\n",
		stats.Uptime,
		stats.Goroutines,
		stats.GoVersion,
	))

	sb.WriteString("🧠 <b>Memory Usage</b>\n")
	if stats.MemLimit != "" {
		sb.WriteString(fmt.Sprintf(
			"• App Memory: %s / %s\n",
			stats.AppMemUsed,
			stats.MemLimit,
		))
	} else {
		sb.WriteString(fmt.Sprintf(
			"• App Memory: %s\n",
			stats.AppMemUsed,
		))
	}
	sb.WriteString(fmt.Sprintf(
		"• Heap: %s\n• GC Runs: %d (pause %s)\n\n",
		stats.AppHeap,
		stats.GCCount,
		stats.GCPause,
	))

	sb.WriteString("💾 <b>Storage</b>\n")
	sb.WriteString(fmt.Sprintf(
		"• Disk Usage: %s / %s\n\n",
		stats.DiskUsed,
		stats.DiskTotal,
	))

	sb.WriteString("📦 <b>Database</b>\n")
	sb.WriteString(fmt.Sprintf(
		"• Chats: %d\n• Users: %d\n",
		len(chats),
		len(users),
	))

	sb.WriteString("\n" + strings.Repeat("─", 36))

	_, _ = sysMsg.Edit(sb.String())
	return nil
}
