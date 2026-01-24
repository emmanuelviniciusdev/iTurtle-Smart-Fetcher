package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
	"iturtle-smart-fetcher/internal/downloader"
)

// BatchConfig represents the root configuration file structure.
type BatchConfig struct {
	Albums []AlbumConfig `yaml:"albums"`
}

// AlbumConfig represents configuration for a single album download.
type AlbumConfig struct {
	URL            string        `yaml:"url"`
	Artist         string        `yaml:"artist"`
	Album          string        `yaml:"album"`
	AlbumArtist    string        `yaml:"album_artist"`
	Year           string        `yaml:"year"`
	Genre          string        `yaml:"genre"`
	Cover          string        `yaml:"cover"`
	OutputDir      string        `yaml:"output_dir"`
	MusicBrainzID  string        `yaml:"musicbrainz_id"`
	AutoFetch      string        `yaml:"auto_fetch"` // "Artist - Album" format for auto-search
	Tracks         []TrackConfig `yaml:"tracks"`
}

// TrackConfig represents per-track configuration.
type TrackConfig struct {
	Num      int    `yaml:"num"`
	Title    string `yaml:"title"`
	Artist   string `yaml:"artist"`
	Composer string `yaml:"composer"`
	Duration string `yaml:"duration"`
	Comment  string `yaml:"comment"`
}

// LoadFromFile reads and parses a YAML configuration file.
func LoadFromFile(path string) (*BatchConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config file: %w", err)
	}

	return Parse(data)
}

// Parse parses YAML configuration data.
func Parse(data []byte) (*BatchConfig, error) {
	var cfg BatchConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}

	if len(cfg.Albums) == 0 {
		return nil, fmt.Errorf("no albums defined in configuration")
	}

	// Validate each album config
	for i, album := range cfg.Albums {
		if album.URL == "" {
			return nil, fmt.Errorf("album %d: url is required", i+1)
		}
	}

	return &cfg, nil
}

// ToDownloaderConfig converts an AlbumConfig to a downloader.Config.
func (ac *AlbumConfig) ToDownloaderConfig(defaultOutputDir string) downloader.Config {
	outputDir := ac.OutputDir
	if outputDir == "" {
		outputDir = defaultOutputDir
	}
	if outputDir == "" {
		outputDir = "."
	}

	cfg := downloader.Config{
		URL:       ac.URL,
		OutputDir: outputDir,
		Cover:     ac.Cover,
		Metadata: downloader.Metadata{
			Artist:      ac.Artist,
			Album:       ac.Album,
			AlbumArtist: ac.AlbumArtist,
			Year:        ac.Year,
			Genre:       ac.Genre,
		},
	}

	// Convert track configs to playlist metadata if present
	if len(ac.Tracks) > 0 || ac.Artist != "" || ac.Album != "" {
		pm := &downloader.PlaylistMetadata{
			AlbumInfo: downloader.AlbumMetadata{
				Title:       ac.Album,
				Artist:      ac.Artist,
				AlbumArtist: ac.AlbumArtist,
				Year:        ac.Year,
				Genre:       ac.Genre,
				TotalTracks: len(ac.Tracks),
				CoverURL:    ac.Cover,
			},
		}

		// If album artist not specified, use artist
		if pm.AlbumInfo.AlbumArtist == "" {
			pm.AlbumInfo.AlbumArtist = ac.Artist
		}

		for _, tc := range ac.Tracks {
			pm.Tracks = append(pm.Tracks, downloader.TrackMetadata{
				Position: tc.Num,
				Title:    tc.Title,
				Artist:   tc.Artist,
				Composer: tc.Composer,
				Duration: tc.Duration,
				Comment:  tc.Comment,
			})
		}

		cfg.PlaylistMetadata = pm
	}

	return cfg
}

// NeedsMusicBrainzLookup returns true if the album should fetch metadata from MusicBrainz.
func (ac *AlbumConfig) NeedsMusicBrainzLookup() bool {
	return ac.MusicBrainzID != "" || ac.AutoFetch != ""
}

// Example returns an example configuration file content.
func Example() string {
	return `# iturtle-smart-fetcher batch configuration
albums:
  # Example 1: Manual metadata
  - url: "https://youtube.com/playlist?list=PLxxxxxx"
    artist: "Black Kids"
    album: "Partie Traumatic"
    year: "2008"
    genre: "Indie Pop"
    cover: "https://example.com/cover.jpg"
    output_dir: "./music/Black Kids"
    tracks:
      - {num: 1, title: "Hit The Heartbrakes"}
      - {num: 2, title: "Partie Traumatic"}
      - {num: 3, title: "I'm Not Gonna Teach Your Boyfriend How to Dance with You"}

  # Example 2: Auto-fetch from MusicBrainz by ID
  - url: "https://youtube.com/playlist?list=PLyyyyyy"
    musicbrainz_id: "abc-123-def-456"
    output_dir: "./music/Motion City Soundtrack"

  # Example 3: Auto-search MusicBrainz
  - url: "https://youtube.com/playlist?list=PLzzzzzz"
    auto_fetch: "Motion City Soundtrack - Commit This to Memory"
    output_dir: "./music/Motion City Soundtrack"
`
}
