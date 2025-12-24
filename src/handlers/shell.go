/*
 * TgMusicBot - Telegram Music Bot
 *  Copyright (c) 2025 Ashok Shau
 *
 *  Licensed under GNU GPL v3
 *  See https://github.com/AshokShau/TgMusicBot
 */

package handlers

import (
	"ashokshau/tgmusic/config"
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	tg "github.com/amarnathcjd/gogram/telegram"
)

func runShellCommand(cmd string, timeout time.Duration) (string, string, int) {
	var shell string
	var args []string

	if runtime.GOOS == "windows" {
		shell = "cmd"
		args = []string{"/C", cmd}
	} else {
		shell = "bash"
		args = []string{"-c", cmd}
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	c := exec.CommandContext(ctx, shell, args...)

	var stdout, stderr bytes.Buffer
	c.Stdout = &stdout
	c.Stderr = &stderr

	err := c.Run()

	if errors.Is(ctx.Err(), context.DeadlineExceeded) {
		return "", fmt.Sprintf("Command timed out after %v seconds", timeout.Seconds()), -1
	}

	exitCode := 0
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			exitCode = exitErr.ExitCode()
		}
	}

	return strings.TrimSpace(stdout.String()), strings.TrimSpace(stderr.String()), exitCode
}

func shellRunner(m *tg.NewMessage) error {
	args := strings.TrimSpace(m.Args())
	if args == "" {
		_, _ = m.Reply("Usage: /sh cmd")
		return tg.ErrEndGroup
	}

	msg, err := m.Reply("Running...")
	if err != nil {
		return tg.ErrEndGroup
	}

	commands := strings.Split(args, "\n")
	var outputParts []string

	for _, cmd := range commands {
		cmd = strings.TrimSpace(cmd)
		if cmd == "" {
			continue
		}

		stdout, stderr, code := runShellCommand(cmd, 300*time.Second)
		part := fmt.Sprintf("<b>Command:</b> <code>%s</code>\n", cmd)
		if stdout != "" {
			part += fmt.Sprintf("<b>Output:</b>\n<pre>%s</pre>\n", stdout)
		}

		if stderr != "" {
			part += fmt.Sprintf("<b>Error:</b>\n<pre>%s</pre>\n", stderr)
		}
		part += fmt.Sprintf("<b>Exit Code:</b> <code>%d</code>\n", code)
		outputParts = append(outputParts, part)
	}

	finalOutput := strings.Join(outputParts, "\n")
	if strings.TrimSpace(finalOutput) == "" {
		finalOutput = "<b>📭 No output was returned</b>"
	}

	if len(finalOutput) <= 3500 {
		_, _ = msg.Edit(finalOutput)
		return tg.ErrEndGroup
	}

	file := filepath.Join(config.Conf.DownloadsDir, fmt.Sprintf("%d.txt", time.Now().UnixNano()))
	if err := os.WriteFile(file, []byte(finalOutput), 0644); err != nil {
		_, _ = msg.Edit(fmt.Sprintf("Failed to write output: %v", err))
		return tg.ErrEndGroup
	}
	defer os.Remove(file)

	_, err = msg.Edit("sending as file", &tg.SendOptions{
		Media:   file,
		Caption: "📁 Output too large, sending as file:",
	})

	if err != nil {
		_, _ = msg.Edit("Error: " + err.Error())
		return tg.ErrEndGroup
	}

	return tg.ErrEndGroup
}

// shellCommand handles /sh commands
func shellCommand(m *tg.NewMessage) error {
	// i don't trust gogram filters
	if !isDev(m) {
		_, _ = m.Reply("WTF ?")
		return tg.ErrEndGroup
	}

	return shellRunner(m)
}
