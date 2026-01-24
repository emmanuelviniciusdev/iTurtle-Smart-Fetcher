package musicbrainz

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	// MusicBrainz API base URL
	apiBaseURL = "https://musicbrainz.org/ws/2"
	// Cover Art Archive base URL
	coverArtBaseURL = "https://coverartarchive.org"
	// User-Agent is required by MusicBrainz API
	userAgent = "iturtle-smart-fetcher/1.0 (https://github.com/user/iturtle-smart-fetcher)"
	// Rate limit: 1 request per second
	rateLimitDelay = time.Second
)

// Client provides access to the MusicBrainz API.
type Client struct {
	httpClient  *http.Client
	lastRequest time.Time
}

// NewClient creates a new MusicBrainz API client.
func NewClient(httpClient *http.Client) *Client {
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 30 * time.Second}
	}
	return &Client{
		httpClient: httpClient,
	}
}

// Release represents a MusicBrainz release (album).
type Release struct {
	ID             string  `json:"id"`
	Title          string  `json:"title"`
	Status         string  `json:"status"`
	Date           string  `json:"date"`
	Country        string  `json:"country"`
	Barcode        string  `json:"barcode"`
	ArtistCredit   []ArtistCredit `json:"artist-credit"`
	LabelInfo      []LabelInfo    `json:"label-info"`
	Media          []Medium       `json:"media"`
	ReleaseGroup   *ReleaseGroup  `json:"release-group"`
	CoverArtArchive *CoverArtStatus `json:"cover-art-archive"`
}

// ArtistCredit represents artist credit information.
type ArtistCredit struct {
	Name   string `json:"name"`
	Artist Artist `json:"artist"`
	JoinPhrase string `json:"joinphrase"`
}

// Artist represents a MusicBrainz artist.
type Artist struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	SortName string `json:"sort-name"`
}

// LabelInfo represents label and catalog information.
type LabelInfo struct {
	CatalogNumber string `json:"catalog-number"`
	Label         *Label `json:"label"`
}

// Label represents a record label.
type Label struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// Medium represents a disc or other medium in a release.
type Medium struct {
	Position int     `json:"position"`
	Format   string  `json:"format"`
	Tracks   []Track `json:"tracks"`
}

// Track represents a single track on a medium.
type Track struct {
	ID        string     `json:"id"`
	Number    string     `json:"number"`
	Title     string     `json:"title"`
	Length    int        `json:"length"` // Duration in milliseconds
	Position  int        `json:"position"`
	Recording *Recording `json:"recording"`
}

// Recording represents the underlying recording of a track.
type Recording struct {
	ID           string         `json:"id"`
	Title        string         `json:"title"`
	Length       int            `json:"length"`
	ISRC         []string       `json:"isrcs"`
	ArtistCredit []ArtistCredit `json:"artist-credit"`
}

// ReleaseGroup represents a group of releases (e.g., different editions of same album).
type ReleaseGroup struct {
	ID        string `json:"id"`
	Title     string `json:"title"`
	PrimaryType string `json:"primary-type"`
}

// CoverArtStatus indicates whether cover art is available.
type CoverArtStatus struct {
	Artwork  bool `json:"artwork"`
	Front    bool `json:"front"`
	Back     bool `json:"back"`
	Count    int  `json:"count"`
}

// SearchResult contains search results from MusicBrainz.
type SearchResult struct {
	Releases []Release `json:"releases"`
	Count    int       `json:"count"`
	Offset   int       `json:"offset"`
}

// CoverArt represents cover art information from Cover Art Archive.
type CoverArt struct {
	Images []CoverArtImage `json:"images"`
	Release string `json:"release"`
}

// CoverArtImage represents a single cover art image.
type CoverArtImage struct {
	ID         int64    `json:"id"`
	Image      string   `json:"image"`
	Thumbnails Thumbnails `json:"thumbnails"`
	Front      bool     `json:"front"`
	Back       bool     `json:"back"`
	Types      []string `json:"types"`
	Approved   bool     `json:"approved"`
}

// Thumbnails contains URLs to thumbnail images.
type Thumbnails struct {
	Small  string `json:"small"`
	Large  string `json:"large"`
	Size250 string `json:"250"`
	Size500 string `json:"500"`
	Size1200 string `json:"1200"`
}

// rateLimit ensures we don't exceed the MusicBrainz rate limit.
func (c *Client) rateLimit() {
	elapsed := time.Since(c.lastRequest)
	if elapsed < rateLimitDelay {
		time.Sleep(rateLimitDelay - elapsed)
	}
	c.lastRequest = time.Now()
}

// doRequest performs an HTTP request with proper headers.
func (c *Client) doRequest(ctx context.Context, url string) ([]byte, error) {
	c.rateLimit()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("execute request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode == http.StatusNotFound {
		return nil, ErrNotFound
	}
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("API error: status %d: %s", resp.StatusCode, string(body))
	}

	return body, nil
}

// ErrNotFound is returned when a resource is not found.
var ErrNotFound = fmt.Errorf("not found")

// GetReleaseByID fetches a release by its MusicBrainz ID.
func (c *Client) GetReleaseByID(ctx context.Context, mbid string) (*Release, error) {
	url := fmt.Sprintf("%s/release/%s?inc=artist-credits+labels+recordings+release-groups+isrcs&fmt=json",
		apiBaseURL, url.PathEscape(mbid))

	body, err := c.doRequest(ctx, url)
	if err != nil {
		return nil, err
	}

	var release Release
	if err := json.Unmarshal(body, &release); err != nil {
		return nil, fmt.Errorf("parse release: %w", err)
	}

	return &release, nil
}

// SearchReleases searches for releases matching the given query.
// Query format: "artist:Artist Name AND release:Album Name"
func (c *Client) SearchReleases(ctx context.Context, query string, limit int) (*SearchResult, error) {
	if limit <= 0 {
		limit = 10
	}

	searchURL := fmt.Sprintf("%s/release?query=%s&limit=%d&fmt=json",
		apiBaseURL, url.QueryEscape(query), limit)

	body, err := c.doRequest(ctx, searchURL)
	if err != nil {
		return nil, err
	}

	var result SearchResult
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("parse search results: %w", err)
	}

	return &result, nil
}

// SearchByArtistAndAlbum searches for releases by artist and album name.
func (c *Client) SearchByArtistAndAlbum(ctx context.Context, artist, album string) (*SearchResult, error) {
	query := fmt.Sprintf("artist:%q AND release:%q", artist, album)
	return c.SearchReleases(ctx, query, 10)
}

// AutoSearch parses a query string like "Artist - Album" and searches for it.
func (c *Client) AutoSearch(ctx context.Context, query string) (*SearchResult, error) {
	// Try to parse "Artist - Album" format
	parts := strings.SplitN(query, " - ", 2)
	if len(parts) == 2 {
		return c.SearchByArtistAndAlbum(ctx, strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1]))
	}

	// Fall back to general search
	return c.SearchReleases(ctx, query, 10)
}

// GetCoverArt fetches cover art information from Cover Art Archive.
func (c *Client) GetCoverArt(ctx context.Context, releaseID string) (*CoverArt, error) {
	url := fmt.Sprintf("%s/release/%s", coverArtBaseURL, url.PathEscape(releaseID))

	body, err := c.doRequest(ctx, url)
	if err != nil {
		return nil, err
	}

	var coverArt CoverArt
	if err := json.Unmarshal(body, &coverArt); err != nil {
		return nil, fmt.Errorf("parse cover art: %w", err)
	}

	return &coverArt, nil
}

// GetFrontCoverURL returns the URL to the front cover image for a release.
// Returns empty string if no front cover is available.
func (c *Client) GetFrontCoverURL(ctx context.Context, releaseID string) (string, error) {
	coverArt, err := c.GetCoverArt(ctx, releaseID)
	if err != nil {
		return "", err
	}

	for _, img := range coverArt.Images {
		if img.Front {
			// Prefer high-quality thumbnail, fall back to full image
			if img.Thumbnails.Size1200 != "" {
				return img.Thumbnails.Size1200, nil
			}
			if img.Thumbnails.Size500 != "" {
				return img.Thumbnails.Size500, nil
			}
			return img.Image, nil
		}
	}

	// No explicit front cover, try first image
	if len(coverArt.Images) > 0 {
		img := coverArt.Images[0]
		if img.Thumbnails.Size1200 != "" {
			return img.Thumbnails.Size1200, nil
		}
		return img.Image, nil
	}

	return "", ErrNotFound
}

// GetArtistName extracts the combined artist name from artist credits.
func GetArtistName(credits []ArtistCredit) string {
	var parts []string
	for _, credit := range credits {
		name := credit.Name
		if name == "" {
			name = credit.Artist.Name
		}
		parts = append(parts, name+credit.JoinPhrase)
	}
	return strings.Join(parts, "")
}

// FormatDuration converts milliseconds to "MM:SS" format.
func FormatDuration(ms int) string {
	if ms <= 0 {
		return ""
	}
	seconds := ms / 1000
	minutes := seconds / 60
	seconds = seconds % 60
	return fmt.Sprintf("%d:%02d", minutes, seconds)
}

// ExtractYear extracts the year from a date string (YYYY-MM-DD or YYYY).
func ExtractYear(date string) string {
	if len(date) >= 4 {
		return date[:4]
	}
	return date
}
