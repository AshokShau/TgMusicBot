/*
 * TgMusicBot - Telegram Music Bot
 *  Copyright (c) 2025-2026 Ashok Shau
 *
 *  Licensed under GNU GPL v3
 *  See https://github.com/AshokShau/TgMusicBot
 */

package handlers

import (
	"ashokshau/tgmusic/src/vc"
	"context"
	"fmt"
	"os"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"time"

	"ashokshau/tgmusic/src/core/db"

	td "github.com/AshokShau/gotdbot"
	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/mem"
	"github.com/shirou/gopsutil/v3/process"
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

	SystemCPU string
	AppCPU    string

	SystemMemUsed  string
	SystemMemTotal string
	CPUCores       int
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

func systemMemoryStats() (used, total string) {
	v, err := mem.VirtualMemory()
	if err != nil {
		return "N/A", "N/A"
	}
	return humanBytes(v.Used), humanBytes(v.Total)
}

func appMemoryStats() (used, heap string, gc uint32, pause string) {
	var ms runtime.MemStats
	runtime.ReadMemStats(&ms)

	return humanBytes(ms.Alloc),
		humanBytes(ms.HeapAlloc),
		ms.NumGC,
		(time.Duration(ms.PauseTotalNs) * time.Nanosecond).String()
}

func systemCPUPercent() string {
	p, err := cpu.Percent(500*time.Millisecond, false)
	if err != nil || len(p) == 0 {
		return "N/A"
	}
	return fmt.Sprintf("%.2f%%", p[0])
}

func appCPUPercent() string {
	p, err := process.NewProcess(int32(os.Getpid()))
	if err != nil {
		return "N/A"
	}

	v, err := p.CPUPercent()
	if err != nil {
		return "N/A"
	}

	return fmt.Sprintf("%.2f%%", v)
}

func gatherAppStats() *AppStats {
	memUsed, heap, gcCount, gcPause := appMemoryStats()
	sysMemUsed, sysMemTotal := systemMemoryStats()

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

		SystemCPU:      systemCPUPercent(),
		AppCPU:         appCPUPercent(),
		SystemMemUsed:  sysMemUsed,
		SystemMemTotal: sysMemTotal,
		CPUCores:       runtime.NumCPU(),
	}

	if limit := readContainerMemLimit(); limit > 0 {
		stats.MemLimit = humanBytes(limit)
	}

	return stats
}

func statsHandler(c *td.Client, ctx *td.Context) error {
	if !isDev(ctx) {
		return td.EndGroups
	}

	msg := ctx.EffectiveMessage
	chatID := msg.ChatId
	if msg.IsPrivate() {
		chatID = 0
	}

	ctx2, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	sysMsg, err := msg.ReplyText(c, "Collecting system statistics...", nil)
	if err != nil {
		return err
	}

	stats := gatherAppStats()
	chats, _ := db.Instance.GetAllChats(ctx2)
	users, _ := db.Instance.GetAllUsers(ctx2)
	ntgCpuUsage, _ := vc.Calls.CpuUsage(chatID)

	var sb strings.Builder

	sb.WriteString(fmt.Sprintf(
		"<b>%s — Runtime Status</b>\n",
		c.Me.FirstName,
	))
	sb.WriteString(strings.Repeat("─", 36) + "\n\n")

	sb.WriteString("<b>System</b>\n")
	sb.WriteString(fmt.Sprintf(
		"• CPU usage: %s (%d cores)\n",
		stats.SystemCPU,
		stats.CPUCores,
	))
	sb.WriteString(fmt.Sprintf(
		"• Ram usage: %s | %s\n",
		stats.SystemMemUsed,
		stats.SystemMemTotal,
	))
	sb.WriteString(fmt.Sprintf(
		"• Storage: %s | %s\n\n",
		stats.DiskUsed,
		stats.DiskTotal,
	))

	sb.WriteString("<b>Application</b>\n")
	sb.WriteString(fmt.Sprintf(
		"• Uptime: %s\n• Goroutines: %d\n• Go Version: %s\n",
		stats.Uptime,
		stats.Goroutines,
		stats.GoVersion,
	))
	sb.WriteString(fmt.Sprintf(
		"• CPU usage: %s\n• NTG Calls CPU: %.2f%%\n",
		stats.AppCPU,
		ntgCpuUsage,
	))
	if stats.MemLimit != "" {
		sb.WriteString(fmt.Sprintf(
			"• Ram usage: %s | %s\n",
			stats.AppMemUsed,
			stats.MemLimit,
		))
	} else {
		sb.WriteString(fmt.Sprintf(
			"• Ram usage: %s\n",
			stats.AppMemUsed,
		))
	}
	sb.WriteString(fmt.Sprintf(
		"• Heap: %s\n• GC Runs: %d (pause %s)\n\n",
		stats.AppHeap,
		stats.GCCount,
		stats.GCPause,
	))

	sb.WriteString("<b>Database</b>\n")
	sb.WriteString(fmt.Sprintf(
		"• Chats: %d\n• Users: %d\n",
		len(chats),
		len(users),
	))

	sb.WriteString("\n" + strings.Repeat("─", 36))

	_, _ = sysMsg.EditText(c, sb.String(), &td.EditTextMessageOpts{ParseMode: "HTML"})
	return nil
}
