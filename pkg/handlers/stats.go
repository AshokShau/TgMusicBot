package handlers

import (
	"fmt"
	"os"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/AshokShau/TgMusicBot/pkg/core/db"
	"github.com/AshokShau/TgMusicBot/pkg/lang"

	"github.com/amarnathcjd/gogram/telegram"
	"github.com/shirou/gopsutil/cpu"
	"github.com/shirou/gopsutil/disk"
	"github.com/shirou/gopsutil/mem"
	"github.com/shirou/gopsutil/process"
)

// AppStats holds both process and system info.
type AppStats struct {
	Uptime          string
	ProcessID       int32
	NumGoroutines   int
	CPUPercent      float64
	MemUsed         string
	MemPerc         float64
	MemLimit        string
	GoVersion       string
	Arch            string
	OS              string
	SystemCPUUsage  float64
	SystemMemUsed   string
	SystemMemTotal  string
	SystemDiskUsed  string
	SystemDiskTotal string
}

// Converts bytes to human-readable string.
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

// Reads memory limit if running inside Docker.
func readContainerMemLimit() uint64 {
	if data, err := os.ReadFile("/sys/fs/cgroup/memory/memory.limit_in_bytes"); err == nil {
		if limit, err := strconv.ParseUint(strings.TrimSpace(string(data)), 10, 64); err == nil {
			if limit > 0 && limit < (1<<60) {
				return limit
			}
		}
	}

	if data, err := os.ReadFile("/sys/fs/cgroup/memory.max"); err == nil {
		val := strings.TrimSpace(string(data))
		if val != "max" {
			if limit, err := strconv.ParseUint(val, 10, 64); err == nil && limit > 0 && limit < (1<<60) {
				return limit
			}
		}
	}
	return 0
}

// Collects both app and system-level stats.
func gatherAppStats() (*AppStats, error) {
	pid := int32(os.Getpid())
	proc, err := process.NewProcess(pid)
	if err != nil {
		return nil, err
	}

	cpuPercent, _ := proc.CPUPercent()
	memInfo, _ := proc.MemoryInfo()
	memPerc, _ := proc.MemoryPercent()

	// ---- System stats ----
	vmem, _ := mem.VirtualMemory()
	cpus, _ := cpu.Percent(0, false)

	// Choose root path for disk usage
	rootPath := "/"
	if runtime.GOOS == "windows" {
		rootPath = "C:\\"
	}
	diskUsage, _ := disk.Usage(rootPath)

	stats := &AppStats{
		Uptime:          time.Since(startTime).Round(time.Second).String(),
		ProcessID:       pid,
		NumGoroutines:   runtime.NumGoroutine(),
		CPUPercent:      cpuPercent,
		MemUsed:         humanBytes(memInfo.RSS),
		MemPerc:         float64(memPerc),
		GoVersion:       runtime.Version(),
		Arch:            fmt.Sprintf("%s (%d CPU cores)", runtime.GOARCH, runtime.NumCPU()),
		OS:              runtime.GOOS,
		SystemCPUUsage:  cpus[0],
		SystemMemUsed:   humanBytes(vmem.Used),
		SystemMemTotal:  humanBytes(vmem.Total),
		SystemDiskUsed:  humanBytes(diskUsage.Used),
		SystemDiskTotal: humanBytes(diskUsage.Total),
	}

	if limit := readContainerMemLimit(); limit > 0 {
		stats.MemLimit = humanBytes(limit)
	}

	return stats, nil
}

// Handles /stats command.
func sysStatsHandler(msg *telegram.NewMessage) error {
	ctx, cancel := db.Ctx()
	defer cancel()
	langCode := db.Instance.GetLang(ctx, msg.ChatID())
	sysMsg, err := msg.Reply(lang.GetString(langCode, "stats_gathering"))
	if err != nil {
		return err
	}

	info, err := gatherAppStats()
	if err != nil {
		_, _ = sysMsg.Edit(fmt.Sprintf(lang.GetString(langCode, "stats_error"), err))
		return nil
	}

	chats, _ := db.Instance.GetAllChats(ctx)
	users, _ := db.Instance.GetAllUsers(ctx)

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf(lang.GetString(langCode, "stats_header"), msg.Client.Me().FirstName))
	sb.WriteString(strings.Repeat("-", 40) + "\n\n")

	sb.WriteString(lang.GetString(langCode, "stats_app_header"))
	sb.WriteString(fmt.Sprintf(lang.GetString(langCode, "stats_uptime"), info.Uptime))
	sb.WriteString(fmt.Sprintf(lang.GetString(langCode, "stats_cpu"), info.CPUPercent))
	if info.MemLimit != "" {
		sb.WriteString(fmt.Sprintf(lang.GetString(langCode, "stats_mem_limited"),
			info.MemUsed, info.MemLimit, info.MemPerc))
	} else {
		sb.WriteString(fmt.Sprintf(lang.GetString(langCode, "stats_mem"), info.MemUsed, info.MemPerc))
	}
	sb.WriteString(fmt.Sprintf(lang.GetString(langCode, "stats_goroutines"), info.NumGoroutines))
	sb.WriteString(fmt.Sprintf(lang.GetString(langCode, "stats_db"), len(chats), len(users)))
	sb.WriteString(fmt.Sprintf(lang.GetString(langCode, "stats_go_version"), info.GoVersion))
	sb.WriteString(fmt.Sprintf(lang.GetString(langCode, "stats_platform"), info.OS, info.Arch))

	sb.WriteString(lang.GetString(langCode, "stats_server_header"))
	sb.WriteString(fmt.Sprintf(lang.GetString(langCode, "stats_server_cpu"), info.SystemCPUUsage))
	sb.WriteString(fmt.Sprintf(lang.GetString(langCode, "stats_server_ram"), info.SystemMemUsed, info.SystemMemTotal))
	sb.WriteString(fmt.Sprintf(lang.GetString(langCode, "stats_server_disk"), info.SystemDiskUsed, info.SystemDiskTotal))
	sb.WriteString(strings.Repeat("-", 40))

	_, _ = sysMsg.Edit(sb.String())
	return nil
}
