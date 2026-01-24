package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"

	"iturtle-smart-fetcher/internal/config"
	"iturtle-smart-fetcher/internal/downloader"
	"iturtle-smart-fetcher/internal/musicbrainz"
	"iturtle-smart-fetcher/internal/tools"
)

func main() {
	var cfg downloader.Config
	var (
		ytDLPPath       string
		ffmpegPath      string
		configFile      string
		musicBrainzID   string
		autoFetchQuery  string
		showExampleConf bool
	)

	flag.StringVar(&cfg.URL, "url", "", "YouTube video or playlist URL (required unless -config is used)")
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

	flag.StringVar(&configFile, "config", "", "Path to YAML batch configuration file")
	flag.StringVar(&musicBrainzID, "musicbrainz-id", "", "MusicBrainz release ID to fetch metadata")
	flag.StringVar(&autoFetchQuery, "auto-fetch-metadata", "", "Auto-search MusicBrainz (format: \"Artist - Album\")")
	flag.BoolVar(&showExampleConf, "example-config", false, "Print example configuration file and exit")

	flag.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), "iTurtle-Smart-Fetcher - download and tag music from YouTube\n\n")
		flag.PrintDefaults()
		fmt.Fprintf(flag.CommandLine.Output(), `
Examples:
  # Single album with manual metadata
  iturtle-smart-fetcher -url https://youtube.com/watch?v=VIDEO_ID -out ./music \
    -artist "Black Kids" -album "Partie Traumatic" -year 2008 -genre "Indie Pop"

  # Fetch metadata from MusicBrainz by ID
  iturtle-smart-fetcher -url "..." -musicbrainz-id "abc-123-def"

  # Auto-search MusicBrainz
  iturtle-smart-fetcher -url "..." -auto-fetch-metadata "Black Kids - Partie Traumatic"

  # Batch mode with configuration file
  iturtle-smart-fetcher -config albums.yaml

  # Generate example configuration file
  iturtle-smart-fetcher -example-config > albums.yaml
`)
	}
	flag.Parse()

	// Handle example config output
	if showExampleConf {
		fmt.Print(config.Example())
		os.Exit(0)
	}

	ctx := context.Background()

	// Resolve tool paths first
	manager := tools.New()
	paths, err := manager.Ensure(tools.Options{
		YtDLPPath:  ytDLPPath,
		FFmpegPath: ffmpegPath,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ Tool setup failed: %v\n", err)
		os.Exit(1)
	}

	// Batch mode with config file
	if configFile != "" {
		if err := runBatchMode(ctx, configFile, paths, cfg.AudioFormat); err != nil {
			fmt.Fprintf(os.Stderr, "\n❌ Batch download failed: %v\n", err)
			os.Exit(1)
		}
		return
	}

	// Single download mode
	if strings.TrimSpace(cfg.URL) == "" {
		flag.Usage()
		os.Exit(1)
	}

	cfg.YtDLPPath = paths.YtDLP
	cfg.FFmpegPath = paths.FFmpeg

	// Fetch metadata from MusicBrainz if requested
	if musicBrainzID != "" || autoFetchQuery != "" {
		pm, err := fetchMusicBrainzMetadata(ctx, musicBrainzID, autoFetchQuery)
		if err != nil {
			fmt.Fprintf(os.Stderr, "⚠️  MusicBrainz lookup failed: %v\n", err)
			fmt.Fprintf(os.Stderr, "    Continuing without MusicBrainz metadata...\n\n")
		} else {
			cfg.PlaylistMetadata = pm
			fmt.Fprintf(os.Stdout, "🎵 Found: %s - %s (%s)\n", pm.AlbumInfo.Artist, pm.AlbumInfo.Title, pm.AlbumInfo.Year)
			fmt.Fprintf(os.Stdout, "   %d tracks\n\n", len(pm.Tracks))
		}
	}

	dl := downloader.New(nil, nil)

	_, err = dl.Download(ctx, cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "\n❌ Download failed: %v\n", err)
		os.Exit(1)
	}
}

// runBatchMode processes albums from a configuration file.
func runBatchMode(ctx context.Context, configFile string, paths tools.Paths, defaultFormat string) error {
	batchCfg, err := config.LoadFromFile(configFile)
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	fmt.Fprintf(os.Stdout, "🐢 Processing %d album(s) from configuration...\n\n", len(batchCfg.Albums))

	dl := downloader.New(nil, nil)
	var failed []string

	for i, albumCfg := range batchCfg.Albums {
		fmt.Fprintf(os.Stdout, "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
		fmt.Fprintf(os.Stdout, "Album %d/%d", i+1, len(batchCfg.Albums))
		if albumCfg.Album != "" {
			fmt.Fprintf(os.Stdout, ": %s", albumCfg.Album)
		}
		if albumCfg.Artist != "" {
			fmt.Fprintf(os.Stdout, " by %s", albumCfg.Artist)
		}
		fmt.Fprintf(os.Stdout, "\n━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n\n")

		cfg := albumCfg.ToDownloaderConfig(".")
		cfg.YtDLPPath = paths.YtDLP
		cfg.FFmpegPath = paths.FFmpeg
		if cfg.AudioFormat == "" {
			cfg.AudioFormat = defaultFormat
		}

		// Fetch MusicBrainz metadata if needed
		if albumCfg.NeedsMusicBrainzLookup() {
			pm, err := fetchMusicBrainzMetadata(ctx, albumCfg.MusicBrainzID, albumCfg.AutoFetch)
			if err != nil {
				fmt.Fprintf(os.Stderr, "⚠️  MusicBrainz lookup failed: %v\n", err)
				fmt.Fprintf(os.Stderr, "    Continuing with manual metadata...\n\n")
			} else {
				cfg.PlaylistMetadata = pm
				fmt.Fprintf(os.Stdout, "🎵 Found: %s - %s (%s)\n", pm.AlbumInfo.Artist, pm.AlbumInfo.Title, pm.AlbumInfo.Year)
				fmt.Fprintf(os.Stdout, "   %d tracks\n\n", len(pm.Tracks))
			}
		}

		_, err := dl.Download(ctx, cfg)
		if err != nil {
			fmt.Fprintf(os.Stderr, "\n❌ Failed to download album: %v\n\n", err)
			name := albumCfg.Album
			if name == "" {
				name = albumCfg.URL
			}
			failed = append(failed, name)
			continue
		}
		fmt.Fprintf(os.Stdout, "\n")
	}

	// Summary
	fmt.Fprintf(os.Stdout, "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
	fmt.Fprintf(os.Stdout, "Batch Complete: %d/%d albums successful\n", len(batchCfg.Albums)-len(failed), len(batchCfg.Albums))
	if len(failed) > 0 {
		fmt.Fprintf(os.Stdout, "Failed albums:\n")
		for _, name := range failed {
			fmt.Fprintf(os.Stdout, "  - %s\n", name)
		}
		return fmt.Errorf("%d album(s) failed", len(failed))
	}

	return nil
}

// fetchMusicBrainzMetadata fetches album and track metadata from MusicBrainz.
func fetchMusicBrainzMetadata(ctx context.Context, mbID, autoQuery string) (*downloader.PlaylistMetadata, error) {
	client := musicbrainz.NewClient(nil)

	var release *musicbrainz.Release
	var err error

	if mbID != "" {
		// Fetch by MusicBrainz ID
		release, err = client.GetReleaseByID(ctx, mbID)
		if err != nil {
			return nil, fmt.Errorf("fetch release by ID: %w", err)
		}
	} else if autoQuery != "" {
		// Auto-search
		results, err := client.AutoSearch(ctx, autoQuery)
		if err != nil {
			return nil, fmt.Errorf("search releases: %w", err)
		}
		if len(results.Releases) == 0 {
			return nil, fmt.Errorf("no releases found for query: %s", autoQuery)
		}
		// Get full release details for the first result
		release, err = client.GetReleaseByID(ctx, results.Releases[0].ID)
		if err != nil {
			return nil, fmt.Errorf("fetch release details: %w", err)
		}
	} else {
		return nil, fmt.Errorf("either musicbrainz-id or auto-fetch-metadata is required")
	}

	// Try to get cover art
	var coverURL string
	coverURL, err = client.GetFrontCoverURL(ctx, release.ID)
	if err != nil {
		// Cover art is optional, continue without it
		coverURL = ""
	}

	return musicbrainz.ToPlaylistMetadataWithCover(release, coverURL), nil
}
