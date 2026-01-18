package downloader

// Metadata holds tags to embed into downloaded audio files.
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

// Config defines the parameters for a download run.
type Config struct {
	URL         string
	OutputDir   string
	Cover       string // Local file path or URL
	AudioFormat string
	YtDLPPath   string
	FFmpegPath  string
	Metadata    Metadata
}
