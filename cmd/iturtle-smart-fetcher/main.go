package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"

	"iturtle-smart-fetcher/internal/downloader"
	"iturtle-smart-fetcher/internal/tools"
)

func main() {
	var cfg downloader.Config
	var (
		ytDLPPath  string
		ffmpegPath string
	)

	flag.StringVar(&cfg.URL, "url", "", "YouTube video or playlist URL (required)")
	flag.StringVar(&cfg.OutputDir, "out", ".", "Directory where songs will be stored")
	flag.StringVar(&cfg.Cover, "cover", "", "Path or URL to album / track cover image")
	flag.StringVar(&cfg.AudioFormat, "format", "mp3", "Audio format to save (mp3 recommended)")
	flag.StringVar(&ytDLPPath, "yt-dlp-path", "", "Path to yt-dlp binary (optional, searches PATH if not specified)")
	flag.StringVar(&ffmpegPath, "ffmpeg-path", "", "Path to ffmpeg binary (optional, searches PATH if not specified)")

	flag.StringVar(&cfg.Metadata.Title, "title", "", "Song title metadata override")
	flag.StringVar(&cfg.Metadata.Artist, "artist", "", "Artist metadata")
	flag.StringVar(&cfg.Metadata.Album, "album", "", "Album metadata")
	flag.StringVar(&cfg.Metadata.AlbumArtist, "album-artist", "", "Album artist metadata")
	flag.StringVar(&cfg.Metadata.Composer, "composer", "", "Composer metadata")
	flag.StringVar(&cfg.Metadata.Year, "year", "", "Release year metadata")
	flag.StringVar(&cfg.Metadata.Genre, "genre", "", "Genre metadata")
	flag.StringVar(&cfg.Metadata.Track, "track", "", "Track number metadata")
	flag.StringVar(&cfg.Metadata.Comment, "comment", "", "Comment metadata")

	flag.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), "iTurtle-Smart-Fetcher - download and tag music from YouTube\n\n")
		flag.PrintDefaults()
		fmt.Fprintf(flag.CommandLine.Output(), "\nExample:\n  iturtle-smart-fetcher -url https://youtube.com/watch?v=VIDEO_ID -out ./music -artist \"Black Kids\" -album \"Partie Traumatic\" -year 2008 -genre \"Indie Pop\"\n")
	}
	flag.Parse()

	if strings.TrimSpace(cfg.URL) == "" {
		flag.Usage()
		os.Exit(1)
	}

	ctx := context.Background()
	manager := tools.New()
	paths, err := manager.Ensure(tools.Options{
		YtDLPPath:  ytDLPPath,
		FFmpegPath: ffmpegPath,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ Tool setup failed: %v\n", err)
		os.Exit(1)
	}

	cfg.YtDLPPath = paths.YtDLP
	cfg.FFmpegPath = paths.FFmpeg

	dl := downloader.New(nil, nil)

	_, err = dl.Download(ctx, cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "\n❌ Download failed: %v\n", err)
		os.Exit(1)
	}
}
