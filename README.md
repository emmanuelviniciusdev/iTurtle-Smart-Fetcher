![iTurtle-Smart-Fetcher](logo.png)

A zero-dependency Go CLI utility for downloading music from YouTube and embedding rich ID3 metadata. Download single videos or entire playlists, save as MP3 (or other audio formats), and automatically tag files with artist, album, year, cover art, and more.

## Features

- **YouTube Downloads**: Fetch single videos or complete playlists via `yt-dlp`
- **Audio Extraction**: Save as MP3 (default) or other audio formats to any target directory
- **Rich Metadata Embedding**: Apply ID3 tags including title, artist, album, album artist, composer, year/date, genre, track number, and comments
- **Cover Art Support**: Embed cover art from local files or URLs using `ffmpeg`
- **Safe Tagging**: Only files created by the current run are modified—existing files are never touched
- **Batch Processing**: Download entire playlists with uniform metadata applied to all tracks
- **Progress Feedback**: Beautiful turtle-themed progress indicators 🐢 with real-time download and tagging status
- **Cross-Platform**: Supports Linux (x86-64, x86, ARM64), macOS (x86-64, ARM64), and Windows (x86-64, x86)
- **Zero Go Dependencies**: Built entirely with Go's standard library

## Requirements

- **Go 1.25+** (for building from source)
- **External Tools** (required):
  - `yt-dlp` - YouTube downloader
  - `ffmpeg` - Audio conversion and metadata tagging
- **Network Access**: Required for YouTube and cover image URLs

### Installing Dependencies

**macOS** (using Homebrew):
```bash
brew install yt-dlp ffmpeg
```

**Ubuntu/Debian**:
```bash
sudo apt install yt-dlp ffmpeg
```

**Arch Linux**:
```bash
sudo pacman -S yt-dlp ffmpeg
```

**Windows** (using Chocolatey):
```bash
choco install yt-dlp ffmpeg
```

Or download manually from:
- [yt-dlp releases](https://github.com/yt-dlp/yt-dlp/releases)
- [ffmpeg downloads](https://ffmpeg.org/download.html)

## Installation

### Option 1: Using Make (Recommended)

```bash
git clone https://github.com/emmanuelviniciusdev/iTurtle-Smart-Fetcher.git
cd iTurtle-Smart-Fetcher

# Build for current platform
make build

# Or install to $GOPATH/bin
make install

# See all available commands
make help
```

### Option 2: Manual Build from Source

```bash
git clone https://github.com/emmanuelviniciusdev/iTurtle-Smart-Fetcher.git
cd iTurtle-Smart-Fetcher
go build -o iturtle-smart-fetcher ./cmd/iturtle-smart-fetcher
```

### Option 3: Install to GOPATH/bin

```bash
go install github.com/emmanuelviniciusdev/iTurtle-Smart-Fetcher/cmd/iturtle-smart-fetcher@latest
```

### Option 4: Download Pre-built Binary

Download the latest release for your platform from the [Releases](https://github.com/emmanuelviniciusdev/iTurtle-Smart-Fetcher/releases) page.

## Quick Start

### Download a Single Video

```bash
iturtle-smart-fetcher \
  -url "https://youtube.com/watch?v=VIDEO_ID" \
  -out ./music \
  -artist "Black Kids" \
  -album "Partie Traumatic" \
  -year 2008 \
  -genre "Indie Pop"
```

### Download a Full Playlist with Cover Art

```bash
iturtle-smart-fetcher \
  -url "https://youtube.com/playlist?list=PLAYLIST_ID" \
  -out ./downloads \
  -cover ./art/album.jpg \
  -artist "Black Kids" \
  -album "Partie Traumatic" \
  -year 2008 \
  -genre "Indie Pop"
```

### Using a Remote Cover Image

```bash
iturtle-smart-fetcher \
  -url "https://youtube.com/watch?v=abc123" \
  -cover "https://example.com/cover.jpg" \
  -artist "Artist Name"
```

## CLI Reference

### Required Flags

| Flag | Description |
|------|-------------|
| `-url` | YouTube video or playlist URL |

### Output Options

| Flag | Default | Description |
|------|---------|-------------|
| `-out` | `.` (current directory) | Directory where audio files will be saved |
| `-format` | `mp3` | Audio format (mp3 recommended for ID3 support) |
| `-cover` | (none) | Local path or URL to cover art image |

### Metadata Flags

| Flag | Description |
|------|-------------|
| `-title` | Song title (overrides YouTube title) |
| `-artist` | Artist name |
| `-album` | Album name |
| `-album-artist` | Album artist (for compilations) |
| `-composer` | Composer name |
| `-year` | Release year |
| `-genre` | Music genre |
| `-track` | Track number |
| `-comment` | Additional comments |

**Note on Metadata Behavior**: All metadata flags are optional. If a metadata field is not specified, the original metadata extracted by `yt-dlp` from YouTube (such as video title, uploader name, etc.) is preserved in the downloaded file. Only the metadata fields you explicitly provide will override the YouTube-extracted values.

### Tool Path Overrides

| Flag | Default | Description |
|------|---------|-------------|
| `-yt-dlp-path` | (searches PATH) | Path to `yt-dlp` binary |
| `-ffmpeg-path` | (searches PATH) | Path to `ffmpeg` binary |

By default, the tool searches for `yt-dlp` and `ffmpeg` in your system PATH. Use these flags to specify custom locations if needed.

## How It Works

```mermaid
flowchart TD
    A[Parse CLI Flags] --> B[Resolve Tools]
    B --> C{Tools Available?}
    C -->|Yes| D[Snapshot Existing Files]
    C -->|No + AutoDownload| E[Download Tools]
    E --> D
    C -->|No| F[Exit with Error]
    D --> G[Run yt-dlp]
    G --> H[Extract Audio to Output Dir]
    H --> I[Diff Files: Find New Tracks]
    I --> J{Cover Provided?}
    J -->|Local File| K[Validate Path]
    J -->|URL| L[Download to Temp File]
    J -->|No| M[Skip Cover]
    K --> N[Apply Metadata with ffmpeg]
    L --> N
    M --> N
    N --> O[Tagged Files Saved]
    O --> P[Output File List]
```

### Detailed Flow

1. **Tool Resolution**: The CLI first resolves paths to `yt-dlp` and `ffmpeg` using the priority: explicit path flag → system PATH → auto-download (if enabled)

2. **Pre-download Snapshot**: Before downloading, the tool captures a list of all existing files with the target format in the output directory

3. **Audio Download**: Executes `yt-dlp` with flags:
   ```
   yt-dlp --extract-audio --audio-format mp3 --prefer-ffmpeg --yes-playlist --ignore-errors --no-continue --newline -o "%(title)s.%(ext)s" <URL>
   ```
   The `--prefer-ffmpeg` flag ensures audio is properly converted to the requested format (MP3) instead of falling back to .webm or other container formats.

4. **File Diff Detection**: After download, compares the new file list against the snapshot to identify only newly created files

5. **Cover Preparation**: If a cover is specified:
   - **Local path**: Validates the file exists
   - **URL**: Downloads to a temporary file (cleaned up after processing)

6. **Metadata Application**: For each new file, runs `ffmpeg` to embed ID3v2.3 tags and optional cover art:
   ```
   ffmpeg -y -i input.mp3 [-i cover.jpg] -map 0:a [-map 1] \
     [-c:v mjpeg -disposition:v:0 attached_pic] \
     -metadata artist="..." -metadata album="..." \
     -id3v2_version 3 output.mp3
   ```

## Tool Resolution Strategy

The tool manager follows this simple resolution order for both `yt-dlp` and `ffmpeg`:

```
1. Explicit Path (if provided via -yt-dlp-path or -ffmpeg-path)
   └─> Validate exists and is executable
       ├─> Found: Use it
       └─> Not found: Error

2. System PATH (default behavior)
   └─> exec.LookPath("yt-dlp" or "ffmpeg")
       ├─> Found: Use it
       └─> Not found: Error with installation instructions
```

**Example error message when tools are missing:**
```
❌ Tool setup failed: yt-dlp not found on PATH. Install it or use -yt-dlp-path to specify location
```

## Project Structure

```
iturtle-smart-fetcher/
├── cmd/
│   └── iturtle-smart-fetcher/
│       └── main.go              # CLI entry point, flag parsing
├── internal/
│   ├── downloader/
│   │   ├── downloader.go        # Core download and tagging orchestration
│   │   ├── downloader_test.go   # Unit tests with mocked dependencies
│   │   ├── metadata.go          # Config and Metadata type definitions
│   │   └── runner.go            # Command execution interface
│   └── tools/
│       ├── tools.go             # Tool resolution and auto-download
│       └── tools_test.go        # Unit tests
├── go.mod                       # Go module (1.25.5, zero dependencies)
└── README.md
```

### Key Types

```go
// Config defines the parameters for a download run
type Config struct {
    URL         string   // YouTube video or playlist URL (required)
    OutputDir   string   // Output directory for downloads
    Cover       string   // Local file path or URL to cover image
    AudioFormat string   // Audio format (default: "mp3")
    YtDLPPath   string   // Path to yt-dlp binary
    FFmpegPath  string   // Path to ffmpeg binary
    Metadata    Metadata // Metadata to embed
}

// Metadata holds ID3 tags to embed into audio files
type Metadata struct {
    Title       string
    Artist      string
    Album       string
    AlbumArtist string
    Composer    string
    Year        string
    Genre       string
    Track       string
    Comment     string
}
```

## Development

### Using the Makefile

The project includes a comprehensive Makefile for common development tasks:

```bash
# Show all available commands
make help

# Build for current platform
make build

# Install to $GOPATH/bin
make install

# Run tests
make test

# Run tests with coverage report
make test-coverage

# Format and lint code
make lint

# Build for all platforms
make build-all

# Create release packages
make release

# Run with example parameters
make run-example

# Clean build artifacts
make clean
```

**Common Make Targets:**

| Target | Description |
|--------|-------------|
| `make build` | Build binary for current platform to `./bin/` |
| `make install` | Install binary to `$GOPATH/bin` |
| `make test` | Run all tests |
| `make test-coverage` | Run tests and generate HTML coverage report |
| `make lint` | Format code and run go vet |
| `make clean` | Remove all build artifacts |
| `make build-all` | Build for Linux, macOS, and Windows (all architectures) |
| `make release` | Create release archives for all platforms |
| `make run ARGS="..."` | Build and run with custom arguments |
| `make dev ARGS="..."` | Run directly with `go run` (no build) |

### Testing

Run the test suite:

```bash
# Using make
make test

# Or directly with go
go test ./...
```

If you encounter cache permission issues:

```bash
GOCACHE=/tmp/go-build go test ./...
```

**Test Coverage:**

- **downloader**: URL validation, cover download with mock HTTP, ffmpeg argument construction, end-to-end download flow with fake command runner
- **tools**: Explicit path resolution, system PATH lookup, error handling for missing tools

Generate coverage report:

```bash
make test-coverage
# Opens coverage.html in your browser
```

## Troubleshooting

### Common Issues

| Problem | Solution |
|---------|----------|
| `yt-dlp not found on PATH` | Install yt-dlp: `brew install yt-dlp` (macOS) or see installation section |
| `ffmpeg not found on PATH` | Install ffmpeg: `brew install ffmpeg` (macOS) or see installation section |
| `yt-dlp failed` | Ensure the URL is reachable. Try updating: `yt-dlp -U` |
| `ffmpeg failed` | Verify `ffmpeg` supports MP3 metadata embedding: `ffmpeg -version` |
| `no new audio files found` | Check the output directory is writable and the video/playlist is public |
| `Files downloaded as .webm` | Ensure ffmpeg is installed and accessible; the tool uses `--prefer-ffmpeg` for conversion |
| `cover file: no such file` | Verify the cover path exists and is accessible |
| `download cover: unexpected status` | Check the cover URL is accessible and returns an image |

### Debug Tips

1. **Verify tools are accessible**:
   ```bash
   which yt-dlp ffmpeg
   yt-dlp --version
   ffmpeg -version
   ```

2. **Test yt-dlp manually**:
   ```bash
   yt-dlp --extract-audio --audio-format mp3 "https://youtube.com/watch?v=..."
   ```

3. **Specify custom tool paths if needed**:
   ```bash
   iturtle-smart-fetcher -url "..." -yt-dlp-path /custom/path/to/yt-dlp
   ```

## Limitations

- **External Dependencies**: Requires `yt-dlp` and `ffmpeg` to be installed separately (not pure Go implementations)
- **Uniform Metadata**: All tracks in a playlist receive the same metadata; per-track overrides are not yet supported
- **Audio Formats**: While other formats are supported, MP3 is recommended for best ID3 tag compatibility

## Examples

### Download Album with Full Metadata

```bash
iturtle-smart-fetcher \
  -url "https://youtube.com/playlist?list=PLAYLIST_ID" \
  -out ./music \
  -artist "Black Kids" \
  -album "Partie Traumatic" \
  -album-artist "Black Kids" \
  -year "2008" \
  -genre "Indie Pop" \
  -cover "https://example.com/cover.jpg"
```

### Download Single Track with Metadata

```bash
iturtle-smart-fetcher \
  -url "https://youtube.com/watch?v=VIDEO_ID" \
  -out ./music \
  -title "I'm Not Gonna Teach Your Boyfriend How to Dance with You" \
  -artist "Black Kids" \
  -album "Partie Traumatic" \
  -year "2008" \
  -genre "Indie Pop" \
  -track "7"
```

### Use Custom Tool Paths

```bash
iturtle-smart-fetcher \
  -url "https://youtube.com/watch?v=abc123" \
  -yt-dlp-path /opt/yt-dlp/yt-dlp \
  -ffmpeg-path /opt/ffmpeg/bin/ffmpeg
```

## License

MIT License - see [LICENSE](LICENSE) for details.

## Contributing

Contributions are welcome! Please open an issue or submit a pull request.

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request
