package musicbrainz

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestGetArtistName(t *testing.T) {
	tests := []struct {
		name     string
		credits  []ArtistCredit
		expected string
	}{
		{
			name:     "empty credits",
			credits:  nil,
			expected: "",
		},
		{
			name: "single artist",
			credits: []ArtistCredit{
				{Name: "Black Kids", Artist: Artist{Name: "Black Kids"}},
			},
			expected: "Black Kids",
		},
		{
			name: "multiple artists with join phrase",
			credits: []ArtistCredit{
				{Name: "Artist A", JoinPhrase: " feat. "},
				{Name: "Artist B"},
			},
			expected: "Artist A feat. Artist B",
		},
		{
			name: "fallback to artist name",
			credits: []ArtistCredit{
				{Name: "", Artist: Artist{Name: "Fallback Artist"}},
			},
			expected: "Fallback Artist",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := GetArtistName(tc.credits)
			if result != tc.expected {
				t.Errorf("expected %q, got %q", tc.expected, result)
			}
		})
	}
}

func TestFormatDuration(t *testing.T) {
	tests := []struct {
		ms       int
		expected string
	}{
		{0, ""},
		{-1, ""},
		{1000, "0:01"},
		{60000, "1:00"},
		{123456, "2:03"},
		{3661000, "61:01"},
	}

	for _, tc := range tests {
		result := FormatDuration(tc.ms)
		if result != tc.expected {
			t.Errorf("FormatDuration(%d) = %q, expected %q", tc.ms, result, tc.expected)
		}
	}
}

func TestExtractYear(t *testing.T) {
	tests := []struct {
		date     string
		expected string
	}{
		{"2008-06-24", "2008"},
		{"2008", "2008"},
		{"", ""},
		{"20", "20"},
	}

	for _, tc := range tests {
		result := ExtractYear(tc.date)
		if result != tc.expected {
			t.Errorf("ExtractYear(%q) = %q, expected %q", tc.date, result, tc.expected)
		}
	}
}

func TestGetReleaseByID(t *testing.T) {
	release := Release{
		ID:    "test-id",
		Title: "Test Album",
		Date:  "2008-06-24",
		ArtistCredit: []ArtistCredit{
			{Name: "Test Artist"},
		},
		Media: []Medium{
			{
				Position: 1,
				Tracks: []Track{
					{Position: 1, Title: "Track 1", Length: 180000},
					{Position: 2, Title: "Track 2", Length: 210000},
				},
			},
		},
	}

	client := &http.Client{
		Transport: roundTripperFunc(func(r *http.Request) (*http.Response, error) {
			body, _ := json.Marshal(release)
			return &http.Response{
				StatusCode: 200,
				Body:       io.NopCloser(strings.NewReader(string(body))),
				Header:     http.Header{},
			}, nil
		}),
	}

	mbClient := NewClient(client)
	result, err := mbClient.GetReleaseByID(context.Background(), "test-id")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Title != "Test Album" {
		t.Errorf("expected title %q, got %q", "Test Album", result.Title)
	}

	if len(result.Media) != 1 {
		t.Errorf("expected 1 medium, got %d", len(result.Media))
	}

	if len(result.Media[0].Tracks) != 2 {
		t.Errorf("expected 2 tracks, got %d", len(result.Media[0].Tracks))
	}
}

func TestGetReleaseByIDNotFound(t *testing.T) {
	client := &http.Client{
		Transport: roundTripperFunc(func(r *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: 404,
				Body:       io.NopCloser(strings.NewReader("not found")),
				Header:     http.Header{},
			}, nil
		}),
	}

	mbClient := NewClient(client)
	_, err := mbClient.GetReleaseByID(context.Background(), "invalid-id")
	if err != ErrNotFound {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestSearchReleases(t *testing.T) {
	searchResult := SearchResult{
		Releases: []Release{
			{ID: "id-1", Title: "Album 1"},
			{ID: "id-2", Title: "Album 2"},
		},
		Count: 2,
	}

	client := &http.Client{
		Transport: roundTripperFunc(func(r *http.Request) (*http.Response, error) {
			body, _ := json.Marshal(searchResult)
			return &http.Response{
				StatusCode: 200,
				Body:       io.NopCloser(strings.NewReader(string(body))),
				Header:     http.Header{},
			}, nil
		}),
	}

	mbClient := NewClient(client)
	result, err := mbClient.SearchReleases(context.Background(), "test query", 10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result.Releases) != 2 {
		t.Errorf("expected 2 releases, got %d", len(result.Releases))
	}
}

func TestGetFrontCoverURL(t *testing.T) {
	coverArt := CoverArt{
		Images: []CoverArtImage{
			{
				ID:    1,
				Image: "https://example.com/image.jpg",
				Front: true,
				Thumbnails: Thumbnails{
					Size1200: "https://example.com/image-1200.jpg",
				},
			},
		},
	}

	client := &http.Client{
		Transport: roundTripperFunc(func(r *http.Request) (*http.Response, error) {
			body, _ := json.Marshal(coverArt)
			return &http.Response{
				StatusCode: 200,
				Body:       io.NopCloser(strings.NewReader(string(body))),
				Header:     http.Header{},
			}, nil
		}),
	}

	mbClient := NewClient(client)
	url, err := mbClient.GetFrontCoverURL(context.Background(), "test-id")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if url != "https://example.com/image-1200.jpg" {
		t.Errorf("expected 1200px thumbnail URL, got %q", url)
	}
}

func TestAutoSearch(t *testing.T) {
	tests := []struct {
		query          string
		expectedArtist string
		expectedAlbum  string
	}{
		{
			query:          "Black Kids - Partie Traumatic",
			expectedArtist: "Black Kids",
			expectedAlbum:  "Partie Traumatic",
		},
	}

	for _, tc := range tests {
		t.Run(tc.query, func(t *testing.T) {
			var capturedQuery string
			client := &http.Client{
				Transport: roundTripperFunc(func(r *http.Request) (*http.Response, error) {
					capturedQuery = r.URL.Query().Get("query")
					result := SearchResult{Releases: []Release{{ID: "test"}}}
					body, _ := json.Marshal(result)
					return &http.Response{
						StatusCode: 200,
						Body:       io.NopCloser(strings.NewReader(string(body))),
						Header:     http.Header{},
					}, nil
				}),
			}

			mbClient := NewClient(client)
			_, err := mbClient.AutoSearch(context.Background(), tc.query)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			// Verify the query contains both artist and album
			if !strings.Contains(capturedQuery, tc.expectedArtist) {
				t.Errorf("query should contain artist %q, got %q", tc.expectedArtist, capturedQuery)
			}
			if !strings.Contains(capturedQuery, tc.expectedAlbum) {
				t.Errorf("query should contain album %q, got %q", tc.expectedAlbum, capturedQuery)
			}
		})
	}
}

type roundTripperFunc func(*http.Request) (*http.Response, error)

func (f roundTripperFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	return f(r)
}
