package downloader

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Downloader orchestrates fetching audio with yt-dlp and tagging it with ffmpeg.
type Downloader struct {
	runner     Runner
	httpClient *http.Client
	progress   *ProgressPrinter
}

// New creates a Downloader with sensible defaults for runner and HTTP client.
func New(r Runner, client *http.Client) *Downloader {
	if r == nil {
		r = ExecRunner{}
	}
	if client == nil {
		client = http.DefaultClient
	}
	return &Downloader{
		runner:     r,
		httpClient: client,
		progress:   NewProgressPrinter(os.Stdout),
	}
}

// Download fetches audio from the provided URL, embeds metadata and cover art,
// and returns the relative paths of the new files.
func (d *Downloader) Download(ctx context.Context, cfg Config) ([]string, error) {
	if strings.TrimSpace(cfg.URL) == "" {
		return nil, errors.New("url is required")
	}

	if cfg.OutputDir == "" {
		cfg.OutputDir = "."
	}

	format := strings.ToLower(strings.TrimSpace(cfg.AudioFormat))
	if format == "" {
		format = "mp3"
	}

	if err := os.MkdirAll(cfg.OutputDir, 0o755); err != nil {
		return nil, fmt.Errorf("create output dir: %w", err)
	}

	before, err := snapshotFiles(cfg.OutputDir, format)
	if err != nil {
		return nil, err
	}

	ytCmd := strings.TrimSpace(cfg.YtDLPPath)
	if ytCmd == "" {
		ytCmd = "yt-dlp"
	}

	d.progress.PrintSection("Downloading from YouTube")
	d.progress.PrintStart(fmt.Sprintf("Fetching audio from %s", cfg.URL))

	ytArgs := buildYtDlpArgs(cfg.URL, cfg.OutputDir, format)
	if _, err := d.runner.Run(ctx, ytCmd, ytArgs...); err != nil {
		d.progress.PrintError("Download failed")
		return nil, err
	}

	after, err := snapshotFiles(cfg.OutputDir, format)
	if err != nil {
		return nil, err
	}

	newFiles := diffFiles(before, after)
	if len(newFiles) == 0 {
		d.progress.PrintError("No new audio files found")
		return nil, errors.New("no new audio files found after download")
	}

	d.progress.PrintComplete("Downloaded", len(newFiles))
	for _, file := range newFiles {
		d.progress.PrintFile(file)
	}

	// Determine cover path - check playlist metadata first, then config
	coverSource := cfg.Cover
	if cfg.PlaylistMetadata != nil && cfg.PlaylistMetadata.AlbumInfo.CoverURL != "" {
		coverSource = cfg.PlaylistMetadata.AlbumInfo.CoverURL
	}
	if cfg.PlaylistMetadata != nil && cfg.PlaylistMetadata.AlbumInfo.CoverPath != "" {
		coverSource = cfg.PlaylistMetadata.AlbumInfo.CoverPath
	}

	coverPath, cleanup, err := d.prepareCover(ctx, coverSource)
	if err != nil {
		d.progress.PrintWarning(fmt.Sprintf("Cover preparation failed: %v", err))
	}
	defer cleanup()

	// Check if any metadata or cover is being applied
	hasMetadata := cfg.Metadata.Title != "" || cfg.Metadata.Artist != "" ||
		cfg.Metadata.Album != "" || cfg.Metadata.AlbumArtist != "" ||
		cfg.Metadata.Composer != "" || cfg.Metadata.Year != "" ||
		cfg.Metadata.Genre != "" || cfg.Metadata.Track != "" ||
		cfg.Metadata.Comment != "" || coverPath != "" ||
		cfg.PlaylistMetadata != nil

	if hasMetadata {
		d.progress.PrintSection("Applying Metadata")
		d.progress.PrintStart("Embedding ID3 tags and cover art")

		ffmpegCmd := strings.TrimSpace(cfg.FFmpegPath)
		if ffmpegCmd == "" {
			ffmpegCmd = "ffmpeg"
		}

		for i, file := range newFiles {
			d.progress.PrintProgress(fmt.Sprintf("Tagging %d/%d: %s", i+1, len(newFiles), filepath.Base(file)))

			// Determine metadata for this file
			meta := cfg.Metadata
			if cfg.PlaylistMetadata != nil {
				meta = d.getTrackMetadata(cfg.PlaylistMetadata, file, i)
			}

			if err := d.applyMetadata(ctx, ffmpegCmd, filepath.Join(cfg.OutputDir, file), coverPath, meta); err != nil {
				d.progress.ClearLine()
				d.progress.PrintError(fmt.Sprintf("Failed to tag %s: %v", file, err))
				return newFiles, err
			}
		}
		d.progress.ClearLine()
		d.progress.PrintComplete("Metadata applied to all files", len(newFiles))
	}

	d.progress.PrintSection("Complete")
	fmt.Fprintf(os.Stdout, "🎵 Successfully processed %d file(s) 🎵\n\n", len(newFiles))

	return newFiles, nil
}

func buildYtDlpArgs(url, outputDir, format string) []string {
	// Use playlist index in filename to ensure proper ordering for per-track metadata
	template := filepath.Join(outputDir, "%(playlist_index|0)s - %(title)s.%(ext)s")
	return []string{
		"--extract-audio",
		"--audio-format", format,
		"--audio-quality", "0", // Highest quality (0 = best, 10 = worst for VBR)
		"--prefer-ffmpeg",
		"--yes-playlist",
		"--ignore-errors",
		"--no-continue",
		"--newline",
		"-o", template,
		url,
	}
}

func snapshotFiles(dir, format string) (map[string]struct{}, error) {
	files := map[string]struct{}{}
	targetExt := strings.ToLower("." + strings.TrimPrefix(format, "."))

	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if targetExt == "." && !strings.HasPrefix(format, ".") {
			return nil
		}
		if !strings.EqualFold(filepath.Ext(d.Name()), targetExt) {
			return nil
		}
		rel, err := filepath.Rel(dir, path)
		if err != nil {
			return err
		}
		files[rel] = struct{}{}
		return nil
	})

	return files, err
}

func diffFiles(before, after map[string]struct{}) []string {
	var files []string
	for path := range after {
		if _, exists := before[path]; !exists {
			files = append(files, path)
		}
	}
	sort.Strings(files)
	return files
}

func (d *Downloader) prepareCover(ctx context.Context, cover string) (string, func(), error) {
	if strings.TrimSpace(cover) == "" {
		return "", func() {}, nil
	}

	if !isURL(cover) {
		if _, err := os.Stat(cover); err != nil {
			return "", func() {}, fmt.Errorf("cover file: %w", err)
		}
		return cover, func() {}, nil
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, cover, nil)
	if err != nil {
		return "", func() {}, fmt.Errorf("create cover request: %w", err)
	}

	resp, err := d.httpClient.Do(req)
	if err != nil {
		return "", func() {}, fmt.Errorf("download cover: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		return "", func() {}, fmt.Errorf("download cover: unexpected status %d", resp.StatusCode)
	}

	tmp, err := os.CreateTemp("", "iturtle-cover-*")
	if err != nil {
		return "", func() {}, fmt.Errorf("create temp cover: %w", err)
	}
	defer tmp.Close()

	if _, err := io.Copy(tmp, resp.Body); err != nil {
		return "", func() {}, fmt.Errorf("write cover: %w", err)
	}

	return tmp.Name(), func() { _ = os.Remove(tmp.Name()) }, nil
}

func (d *Downloader) applyMetadata(ctx context.Context, ffmpegCmd, filePath, coverPath string, meta Metadata) error {
	tmpPath := filePath + ".tagged"
	_ = os.Remove(tmpPath)

	args := buildFFmpegArgs(filePath, tmpPath, meta, coverPath)
	if _, err := d.runner.Run(ctx, ffmpegCmd, args...); err != nil {
		return err
	}

	return os.Rename(tmpPath, filePath)
}

func buildFFmpegArgs(input, output string, meta Metadata, coverPath string) []string {
	args := []string{"-y", "-i", input}
	hasCover := strings.TrimSpace(coverPath) != ""

	if hasCover {
		args = append(args, "-i", coverPath)
	}

	args = append(args, "-map", "0:a")
	if hasCover {
		args = append(args,
			"-map", "1",
			"-c:a", "copy",
			"-c:v", "mjpeg",
			"-metadata:s:v", "title=Album cover",
			"-metadata:s:v", "comment=Cover (front)",
			"-disposition:v:0", "attached_pic",
		)
	} else {
		args = append(args, "-c", "copy")
	}

	args = appendMetadata(args, meta)
	args = append(args, "-id3v2_version", "3")

	// Explicitly specify output format for ffmpeg 8.x compatibility
	// (needed because .tagged extension doesn't auto-detect as mp3)
	args = append(args, "-f", "mp3", output)
	return args
}

func appendMetadata(args []string, meta Metadata) []string {
	add := func(key, value string) {
		value = strings.TrimSpace(value)
		if value != "" {
			args = append(args, "-metadata", fmt.Sprintf("%s=%s", key, value))
		}
	}

	add("title", meta.Title)
	add("artist", meta.Artist)
	add("album", meta.Album)
	add("album_artist", meta.AlbumArtist)
	add("composer", meta.Composer)
	add("year", meta.Year)
	add("date", meta.Year)
	add("genre", meta.Genre)
	add("track", meta.Track)
	add("comment", meta.Comment)
	return args
}

func isURL(value string) bool {
	u, err := url.Parse(value)
	if err != nil {
		return false
	}
	return u.Scheme != "" && u.Host != ""
}

// getTrackMetadata determines the metadata for a specific file based on playlist metadata.
// It tries to match by playlist index in the filename, falling back to position-based matching.
func (d *Downloader) getTrackMetadata(pm *PlaylistMetadata, filename string, fileIndex int) Metadata {
	// Try to extract playlist index from filename (format: "N - title.ext")
	trackIndex := extractPlaylistIndex(filename)
	if trackIndex <= 0 {
		// Fall back to file index (1-based)
		trackIndex = fileIndex + 1
	}

	// Find matching track metadata
	var trackMeta TrackMetadata
	if trackIndex > 0 && trackIndex <= len(pm.Tracks) {
		trackMeta = pm.Tracks[trackIndex-1]
	} else if len(pm.Tracks) > fileIndex {
		trackMeta = pm.Tracks[fileIndex]
	}

	return MergeTrackMetadata(pm.AlbumInfo, trackMeta, trackIndex)
}

// extractPlaylistIndex extracts the playlist index from a filename.
// Expected format: "N - title.ext" where N is the playlist index.
func extractPlaylistIndex(filename string) int {
	base := filepath.Base(filename)
	// Look for pattern "N - " at the start
	for i, c := range base {
		if c == ' ' && i > 0 && i+2 < len(base) && base[i+1] == '-' && base[i+2] == ' ' {
			// Parse the number before the dash
			numStr := base[:i]
			var num int
			if _, err := fmt.Sscanf(numStr, "%d", &num); err == nil {
				return num
			}
			break
		}
	}
	return 0
}
