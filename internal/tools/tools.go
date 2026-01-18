package tools

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// Options controls how external tool binaries are resolved.
type Options struct {
	YtDLPPath  string
	FFmpegPath string
}

// Paths contains resolved executable paths for required tools.
type Paths struct {
	YtDLP  string
	FFmpeg string
}

// Manager resolves yt-dlp and ffmpeg paths.
type Manager struct{}

// New returns a new Manager.
func New() *Manager {
	return &Manager{}
}

// Ensure locates yt-dlp and ffmpeg using explicit paths or system PATH.
func (m *Manager) Ensure(opts Options) (Paths, error) {
	ytdlp, err := resolveTool("yt-dlp", opts.YtDLPPath)
	if err != nil {
		return Paths{}, err
	}

	ffmpeg, err := resolveTool("ffmpeg", opts.FFmpegPath)
	if err != nil {
		return Paths{}, err
	}

	return Paths{YtDLP: ytdlp, FFmpeg: ffmpeg}, nil
}

// resolveTool finds a tool binary using the explicit path or system PATH.
func resolveTool(name, explicitPath string) (string, error) {
	// If explicit path provided, use it
	if strings.TrimSpace(explicitPath) != "" {
		if isExecutable(explicitPath) {
			return explicitPath, nil
		}
		return "", fmt.Errorf("%s not found or not executable: %s", name, explicitPath)
	}

	// Try to find in system PATH
	path, err := exec.LookPath(name)
	if err == nil && isExecutable(path) {
		return path, nil
	}

	return "", fmt.Errorf("%s not found on PATH. Install it or use -%s-path to specify location", name, name)
}

// isExecutable checks if a path points to an executable file.
func isExecutable(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return !info.IsDir()
}
