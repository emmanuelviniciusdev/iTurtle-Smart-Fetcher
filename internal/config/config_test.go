package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParse(t *testing.T) {
	yaml := `
albums:
  - url: "https://youtube.com/playlist?list=PLtest"
    artist: "Black Kids"
    album: "Partie Traumatic"
    year: "2008"
    genre: "Indie Pop"
    tracks:
      - {num: 1, title: "Hit The Heartbrakes"}
      - {num: 2, title: "Partie Traumatic"}
`
	cfg, err := Parse([]byte(yaml))
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if len(cfg.Albums) != 1 {
		t.Fatalf("expected 1 album, got %d", len(cfg.Albums))
	}

	album := cfg.Albums[0]
	if album.URL != "https://youtube.com/playlist?list=PLtest" {
		t.Errorf("unexpected URL: %s", album.URL)
	}
	if album.Artist != "Black Kids" {
		t.Errorf("unexpected artist: %s", album.Artist)
	}
	if album.Album != "Partie Traumatic" {
		t.Errorf("unexpected album: %s", album.Album)
	}
	if album.Year != "2008" {
		t.Errorf("unexpected year: %s", album.Year)
	}
	if len(album.Tracks) != 2 {
		t.Errorf("expected 2 tracks, got %d", len(album.Tracks))
	}
	if album.Tracks[0].Title != "Hit The Heartbrakes" {
		t.Errorf("unexpected track 1 title: %s", album.Tracks[0].Title)
	}
}

func TestParseEmpty(t *testing.T) {
	yaml := `albums: []`
	_, err := Parse([]byte(yaml))
	if err == nil {
		t.Fatal("expected error for empty albums")
	}
}

func TestParseMissingURL(t *testing.T) {
	yaml := `
albums:
  - artist: "Test"
    album: "Test Album"
`
	_, err := Parse([]byte(yaml))
	if err == nil {
		t.Fatal("expected error for missing URL")
	}
}

func TestParseMultipleAlbums(t *testing.T) {
	yaml := `
albums:
  - url: "https://youtube.com/playlist?list=PL1"
    artist: "Artist 1"
    album: "Album 1"
  - url: "https://youtube.com/playlist?list=PL2"
    artist: "Artist 2"
    album: "Album 2"
  - url: "https://youtube.com/playlist?list=PL3"
    musicbrainz_id: "abc-123-def"
`
	cfg, err := Parse([]byte(yaml))
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if len(cfg.Albums) != 3 {
		t.Fatalf("expected 3 albums, got %d", len(cfg.Albums))
	}
}

func TestLoadFromFile(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "albums.yaml")

	yaml := `
albums:
  - url: "https://youtube.com/playlist?list=PLtest"
    artist: "Test Artist"
`
	if err := os.WriteFile(configPath, []byte(yaml), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	cfg, err := LoadFromFile(configPath)
	if err != nil {
		t.Fatalf("LoadFromFile failed: %v", err)
	}

	if len(cfg.Albums) != 1 {
		t.Fatalf("expected 1 album, got %d", len(cfg.Albums))
	}
}

func TestLoadFromFileNotExists(t *testing.T) {
	_, err := LoadFromFile("/nonexistent/path/albums.yaml")
	if err == nil {
		t.Fatal("expected error for nonexistent file")
	}
}

func TestToDownloaderConfig(t *testing.T) {
	album := AlbumConfig{
		URL:       "https://youtube.com/playlist",
		Artist:    "Test Artist",
		Album:     "Test Album",
		Year:      "2024",
		Genre:     "Rock",
		OutputDir: "./music",
		Cover:     "https://example.com/cover.jpg",
		Tracks: []TrackConfig{
			{Num: 1, Title: "Track 1", Artist: "Guest Artist"},
			{Num: 2, Title: "Track 2"},
		},
	}

	cfg := album.ToDownloaderConfig("")

	if cfg.URL != "https://youtube.com/playlist" {
		t.Errorf("unexpected URL: %s", cfg.URL)
	}
	if cfg.OutputDir != "./music" {
		t.Errorf("unexpected output dir: %s", cfg.OutputDir)
	}
	if cfg.Metadata.Artist != "Test Artist" {
		t.Errorf("unexpected artist: %s", cfg.Metadata.Artist)
	}

	if cfg.PlaylistMetadata == nil {
		t.Fatal("expected PlaylistMetadata to be set")
	}
	if len(cfg.PlaylistMetadata.Tracks) != 2 {
		t.Errorf("expected 2 tracks, got %d", len(cfg.PlaylistMetadata.Tracks))
	}
	if cfg.PlaylistMetadata.Tracks[0].Artist != "Guest Artist" {
		t.Errorf("expected guest artist for track 1")
	}
}

func TestToDownloaderConfigDefaultOutputDir(t *testing.T) {
	album := AlbumConfig{
		URL:    "https://youtube.com/playlist",
		Artist: "Test",
	}

	cfg := album.ToDownloaderConfig("/default/dir")
	if cfg.OutputDir != "/default/dir" {
		t.Errorf("expected default output dir, got %s", cfg.OutputDir)
	}

	cfg = album.ToDownloaderConfig("")
	if cfg.OutputDir != "." {
		t.Errorf("expected current dir, got %s", cfg.OutputDir)
	}
}

func TestNeedsMusicBrainzLookup(t *testing.T) {
	tests := []struct {
		name     string
		album    AlbumConfig
		expected bool
	}{
		{
			name:     "no lookup needed",
			album:    AlbumConfig{URL: "test", Artist: "Test"},
			expected: false,
		},
		{
			name:     "musicbrainz ID set",
			album:    AlbumConfig{URL: "test", MusicBrainzID: "abc-123"},
			expected: true,
		},
		{
			name:     "auto fetch set",
			album:    AlbumConfig{URL: "test", AutoFetch: "Artist - Album"},
			expected: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := tc.album.NeedsMusicBrainzLookup()
			if result != tc.expected {
				t.Errorf("expected %v, got %v", tc.expected, result)
			}
		})
	}
}

func TestExample(t *testing.T) {
	example := Example()
	if example == "" {
		t.Fatal("Example should return non-empty string")
	}

	// Verify it's valid YAML by parsing it
	cfg, err := Parse([]byte(example))
	if err != nil {
		t.Fatalf("Example config should be valid YAML: %v", err)
	}

	if len(cfg.Albums) == 0 {
		t.Error("Example should contain at least one album")
	}
}
