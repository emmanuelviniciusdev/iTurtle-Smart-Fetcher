package tools

import (
	"os"
	"path/filepath"
	"testing"
)

func TestEnsureUsesOverrides(t *testing.T) {
	tmpDir := t.TempDir()

	// Create fake binaries
	ytdlpPath := filepath.Join(tmpDir, "yt-dlp")
	ffmpegPath := filepath.Join(tmpDir, "ffmpeg")

	if err := os.WriteFile(ytdlpPath, []byte("fake"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(ffmpegPath, []byte("fake"), 0755); err != nil {
		t.Fatal(err)
	}

	m := New()
	paths, err := m.Ensure(Options{
		YtDLPPath:  ytdlpPath,
		FFmpegPath: ffmpegPath,
	})
	if err != nil {
		t.Fatalf("Ensure failed: %v", err)
	}

	if paths.YtDLP != ytdlpPath {
		t.Errorf("expected yt-dlp path %s, got %s", ytdlpPath, paths.YtDLP)
	}
	if paths.FFmpeg != ffmpegPath {
		t.Errorf("expected ffmpeg path %s, got %s", ffmpegPath, paths.FFmpeg)
	}
}

func TestEnsureFailsWithInvalidPaths(t *testing.T) {
	m := New()
	_, err := m.Ensure(Options{
		YtDLPPath:  "/nonexistent/yt-dlp",
		FFmpegPath: "/nonexistent/ffmpeg",
	})
	if err == nil {
		t.Error("expected error for nonexistent paths")
	}
}

func TestEnsureSearchesPATH(t *testing.T) {
	// This test will only pass if yt-dlp and ffmpeg are actually installed
	// Skip if not available
	m := New()
	paths, err := m.Ensure(Options{})

	// If tools aren't on PATH, this should fail with a helpful message
	if err != nil {
		t.Logf("Tools not found on PATH (expected if not installed): %v", err)
		t.Skip("yt-dlp or ffmpeg not on PATH, skipping")
	}

	// If we got here, tools were found
	if paths.YtDLP == "" {
		t.Error("yt-dlp path should not be empty")
	}
	if paths.FFmpeg == "" {
		t.Error("ffmpeg path should not be empty")
	}
}
