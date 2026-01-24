package downloader

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestIsURL(t *testing.T) {
	if !isURL("https://example.com/video") {
		t.Fatalf("expected URL to be valid")
	}
	if isURL("not-a-url") {
		t.Fatalf("expected non URL to be invalid")
	}
}

func TestPrepareCoverWithURL(t *testing.T) {
	client := &http.Client{
		Transport: roundTripperFunc(func(r *http.Request) (*http.Response, error) {
			body := io.NopCloser(strings.NewReader("image-bytes"))
			return &http.Response{
				StatusCode: 200,
				Body:       body,
				Header:     http.Header{},
			}, nil
		}),
	}

	dl := New(nil, client)
	path, cleanup, err := dl.prepareCover(context.Background(), "https://example.com/cover.jpg")
	defer cleanup()
	if err != nil {
		t.Fatalf("prepareCover returned error: %v", err)
	}
	if path == "" {
		t.Fatalf("expected a downloaded cover path")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("expected cover file to exist: %v", err)
	}
	if string(data) != "image-bytes" {
		t.Fatalf("unexpected cover content: %s", string(data))
	}
}

func TestBuildFFmpegArgsWithCoverAndMetadata(t *testing.T) {
	meta := Metadata{
		Title:       "Song",
		Artist:      "Artist",
		Album:       "Album",
		AlbumArtist: "Album Artist",
		Composer:    "Composer",
		Year:        "2024",
		Genre:       "Genre",
		Track:       "1",
		Comment:     "Note",
	}

	args := buildFFmpegArgs("in.mp3", "out.mp3", meta, "cover.jpg")
	argsJoined := strings.Join(args, " ")

	expected := []string{
		"cover.jpg",
		"artist=Artist",
		"album=Album",
		"title=Song",
		"attached_pic",
		"out.mp3",
	}

	for _, val := range expected {
		if !strings.Contains(argsJoined, val) {
			t.Fatalf("expected ffmpeg args to contain %q; args: %s", val, argsJoined)
		}
	}
}

func TestDownloadFlowCreatesAndTagsFiles(t *testing.T) {
	tempDir := t.TempDir()
	runner := &fakeRunner{audioFormat: "mp3"}
	dl := New(runner, nil)

	cfg := Config{
		URL:         "https://example.com/playlist",
		OutputDir:   tempDir,
		AudioFormat: "mp3",
		Metadata: Metadata{
			Artist: "Tester",
			Album:  "Album",
			Year:   "2024",
		},
	}

	files, err := dl.Download(context.Background(), cfg)
	if err != nil {
		t.Fatalf("Download failed: %v", err)
	}

	if len(files) != 2 {
		t.Fatalf("expected 2 files, got %d (%v)", len(files), files)
	}

	for _, file := range files {
		if !strings.HasSuffix(file, ".mp3") {
			t.Fatalf("expected mp3 extension for %s", file)
		}
		if _, err := os.Stat(filepath.Join(tempDir, file)); err != nil {
			t.Fatalf("expected file to exist: %v", err)
		}
	}

	ffmpegCalls := 0
	for _, call := range runner.calls {
		if call.name == "ffmpeg" {
			ffmpegCalls++
		}
	}
	if ffmpegCalls != len(files) {
		t.Fatalf("expected ffmpeg to be called %d times, got %d", len(files), ffmpegCalls)
	}
}

func TestDownloadRequiresURL(t *testing.T) {
	dl := New(&fakeRunner{}, nil)
	_, err := dl.Download(context.Background(), Config{OutputDir: t.TempDir()})
	if err == nil {
		t.Fatalf("expected error for missing URL")
	}
}

type fakeRunner struct {
	audioFormat string
	calls       []cmdCall
}

type cmdCall struct {
	name string
	args []string
}

func (f *fakeRunner) Run(ctx context.Context, name string, args ...string) (string, error) {
	f.calls = append(f.calls, cmdCall{name: name, args: append([]string{}, args...)})

	switch name {
	case "yt-dlp":
		outDir := extractOutputDir(args)
		if outDir == "" {
			return "", errors.New("missing -o argument for yt-dlp")
		}
		if err := os.MkdirAll(outDir, 0o755); err != nil {
			return "", err
		}
		for i := 1; i <= 2; i++ {
			path := filepath.Join(outDir, fmt.Sprintf("track%d.%s", i, f.audioFormat))
			if err := os.WriteFile(path, []byte("audio"), 0o644); err != nil {
				return "", err
			}
		}
		return "ok", nil
	case "ffmpeg":
		if len(args) == 0 {
			return "", errors.New("ffmpeg missing args")
		}
		output := args[len(args)-1]
		if err := os.WriteFile(output, []byte("tagged"), 0o644); err != nil {
			return "", err
		}
		return "ok", nil
	default:
		return "", fmt.Errorf("unexpected command: %s", name)
	}
}

func extractOutputDir(args []string) string {
	for i := 0; i < len(args)-1; i++ {
		if args[i] == "-o" {
			return filepath.Dir(args[i+1])
		}
	}
	return ""
}

type roundTripperFunc func(*http.Request) (*http.Response, error)

func (f roundTripperFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	return f(r)
}

func TestExtractPlaylistIndex(t *testing.T) {
	tests := []struct {
		filename string
		expected int
	}{
		{"1 - Track Title.mp3", 1},
		{"01 - Track Title.mp3", 1},
		{"12 - Some Song.mp3", 12},
		{"Track Title.mp3", 0},
		{"NoIndex.mp3", 0},
		{"/path/to/3 - Song.mp3", 3},
	}

	for _, tc := range tests {
		result := extractPlaylistIndex(tc.filename)
		if result != tc.expected {
			t.Errorf("extractPlaylistIndex(%q) = %d, expected %d", tc.filename, result, tc.expected)
		}
	}
}

func TestMergeTrackMetadata(t *testing.T) {
	album := AlbumMetadata{
		Title:       "Test Album",
		Artist:      "Album Artist",
		AlbumArtist: "Album Artist",
		Year:        "2024",
		Genre:       "Rock",
		TotalTracks: 10,
	}

	track := TrackMetadata{
		Position: 3,
		Title:    "Track Title",
		Composer: "Composer Name",
	}

	meta := MergeTrackMetadata(album, track, 3)

	if meta.Title != "Track Title" {
		t.Errorf("expected title %q, got %q", "Track Title", meta.Title)
	}
	if meta.Artist != "Album Artist" {
		t.Errorf("expected artist %q, got %q", "Album Artist", meta.Artist)
	}
	if meta.Album != "Test Album" {
		t.Errorf("expected album %q, got %q", "Test Album", meta.Album)
	}
	if meta.Year != "2024" {
		t.Errorf("expected year %q, got %q", "2024", meta.Year)
	}
	if meta.Composer != "Composer Name" {
		t.Errorf("expected composer %q, got %q", "Composer Name", meta.Composer)
	}
	if meta.Track != "3/10" {
		t.Errorf("expected track %q, got %q", "3/10", meta.Track)
	}
}

func TestMergeTrackMetadataWithTrackArtist(t *testing.T) {
	album := AlbumMetadata{
		Title:  "Compilation",
		Artist: "Various Artists",
	}

	track := TrackMetadata{
		Title:  "Guest Track",
		Artist: "Guest Artist",
	}

	meta := MergeTrackMetadata(album, track, 1)

	if meta.Artist != "Guest Artist" {
		t.Errorf("expected track artist to override, got %q", meta.Artist)
	}
}

func TestFormatTrackNumber(t *testing.T) {
	tests := []struct {
		track    int
		total    int
		expected string
	}{
		{1, 10, "1/10"},
		{5, 0, "5"},
		{12, 12, "12/12"},
	}

	for _, tc := range tests {
		result := formatTrackNumber(tc.track, tc.total)
		if result != tc.expected {
			t.Errorf("formatTrackNumber(%d, %d) = %q, expected %q", tc.track, tc.total, result, tc.expected)
		}
	}
}

func TestDownloadWithPlaylistMetadata(t *testing.T) {
	tempDir := t.TempDir()
	runner := &fakeRunnerWithIndex{audioFormat: "mp3"}
	dl := New(runner, nil)

	cfg := Config{
		URL:         "https://example.com/playlist",
		OutputDir:   tempDir,
		AudioFormat: "mp3",
		PlaylistMetadata: &PlaylistMetadata{
			AlbumInfo: AlbumMetadata{
				Title:       "Test Album",
				Artist:      "Test Artist",
				Year:        "2024",
				TotalTracks: 2,
			},
			Tracks: []TrackMetadata{
				{Position: 1, Title: "First Track"},
				{Position: 2, Title: "Second Track"},
			},
		},
	}

	files, err := dl.Download(context.Background(), cfg)
	if err != nil {
		t.Fatalf("Download failed: %v", err)
	}

	if len(files) != 2 {
		t.Fatalf("expected 2 files, got %d", len(files))
	}

	// Verify ffmpeg was called for each file with metadata
	ffmpegCalls := 0
	for _, call := range runner.calls {
		if call.name == "ffmpeg" {
			ffmpegCalls++
			// Check that metadata args are present
			argsStr := strings.Join(call.args, " ")
			if !strings.Contains(argsStr, "Test Album") {
				t.Errorf("expected ffmpeg args to contain album name")
			}
		}
	}

	if ffmpegCalls != 2 {
		t.Errorf("expected 2 ffmpeg calls, got %d", ffmpegCalls)
	}
}

// fakeRunnerWithIndex creates files with playlist index prefix
type fakeRunnerWithIndex struct {
	audioFormat string
	calls       []cmdCall
}

func (f *fakeRunnerWithIndex) Run(ctx context.Context, name string, args ...string) (string, error) {
	f.calls = append(f.calls, cmdCall{name: name, args: append([]string{}, args...)})

	switch name {
	case "yt-dlp":
		outDir := extractOutputDir(args)
		if outDir == "" {
			return "", errors.New("missing -o argument for yt-dlp")
		}
		if err := os.MkdirAll(outDir, 0o755); err != nil {
			return "", err
		}
		// Create files with playlist index prefix
		for i := 1; i <= 2; i++ {
			path := filepath.Join(outDir, fmt.Sprintf("%d - track%d.%s", i, i, f.audioFormat))
			if err := os.WriteFile(path, []byte("audio"), 0o644); err != nil {
				return "", err
			}
		}
		return "ok", nil
	case "ffmpeg":
		if len(args) == 0 {
			return "", errors.New("ffmpeg missing args")
		}
		output := args[len(args)-1]
		if err := os.WriteFile(output, []byte("tagged"), 0o644); err != nil {
			return "", err
		}
		return "ok", nil
	default:
		return "", fmt.Errorf("unexpected command: %s", name)
	}
}

func TestBuildYtDlpArgsHighestQuality(t *testing.T) {
	args := buildYtDlpArgs("https://example.com/video", "/output", "mp3")
	argsStr := strings.Join(args, " ")

	// Check for highest quality flag
	if !strings.Contains(argsStr, "--audio-quality 0") {
		t.Errorf("expected --audio-quality 0 for highest quality, args: %s", argsStr)
	}

	// Check for playlist index in template
	if !strings.Contains(argsStr, "%(playlist_index") {
		t.Errorf("expected playlist_index in output template, args: %s", argsStr)
	}
}
