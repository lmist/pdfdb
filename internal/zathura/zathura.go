package zathura

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"syscall"
)

const AppBinary = "/Applications/Zathura.app/Contents/MacOS/zathura"

type OpenDocument struct {
	Slug string `json:"slug"`
	PID  int    `json:"pid"`
	Path string `json:"path"`
}

type Process struct {
	PID     int
	Command string
}

func Open(ctx context.Context, path string) error {
	if _, err := os.Stat(path); err != nil {
		return fmt.Errorf("cached PDF is not available at %s: %w", path, err)
	}
	if _, err := os.Stat(AppBinary); err == nil {
		return exec.CommandContext(ctx, AppBinary, "--fork", path).Start()
	}
	if openPath, err := exec.LookPath("open"); err == nil {
		return exec.CommandContext(ctx, openPath, "-a", "Zathura", path).Run()
	}
	return errors.New("Zathura.app is not installed at /Applications/Zathura.app")
}

func OpenDocuments(ctx context.Context, paths map[string]string) ([]OpenDocument, error) {
	processes, err := ListProcesses(ctx)
	if err != nil {
		return nil, err
	}
	return MatchOpenDocuments(processes, paths), nil
}

func Close(ctx context.Context, slug string, path string) error {
	open, err := OpenDocuments(ctx, map[string]string{slug: path})
	if err != nil {
		return err
	}
	if len(open) == 0 {
		return nil
	}
	return closeOpenDocuments(ctx, open)
}

func closeOpenDocuments(ctx context.Context, open []OpenDocument) error {
	for _, item := range open {
		proc, err := os.FindProcess(item.PID)
		if err != nil {
			return err
		}
		if err := proc.Signal(syscall.SIGTERM); err != nil {
			return fmt.Errorf("close zathura pid %d: %w", item.PID, err)
		}
	}
	return nil
}

func ListProcesses(ctx context.Context) ([]Process, error) {
	out, err := exec.CommandContext(ctx, "ps", "-axo", "pid=,command=").Output()
	if err != nil {
		return nil, err
	}
	return ParseProcesses(out), nil
}

func ParseProcesses(out []byte) []Process {
	var processes []Process
	scanner := bufio.NewScanner(bytes.NewReader(out))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		pidText, command, ok := strings.Cut(line, " ")
		if !ok {
			continue
		}
		pid, err := strconv.Atoi(strings.TrimSpace(pidText))
		if err != nil {
			continue
		}
		command = strings.TrimSpace(command)
		if command == "" {
			continue
		}
		processes = append(processes, Process{PID: pid, Command: command})
	}
	return processes
}

func MatchOpenDocuments(processes []Process, paths map[string]string) []OpenDocument {
	var out []OpenDocument
	for _, proc := range processes {
		if !strings.Contains(strings.ToLower(proc.Command), "zathura") {
			continue
		}
		for slug, path := range paths {
			if commandContainsPath(proc.Command, path) {
				out = append(out, OpenDocument{Slug: slug, PID: proc.PID, Path: path})
			}
		}
	}
	return out
}

func commandContainsPath(command, path string) bool {
	if strings.Contains(command, path) {
		return true
	}
	escaped := strings.ReplaceAll(path, " ", "\\ ")
	return escaped != path && strings.Contains(command, escaped)
}
