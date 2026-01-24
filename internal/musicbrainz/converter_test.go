package musicbrainz

import (
	"testing"
)

func TestToPlaylistMetadata(t *testing.T) {
	release := &Release{
		ID:    "test-id",
		Title: "Partie Traumatic",
		Date:  "2008-06-24",
		Country: "US",
		ArtistCredit: []ArtistCredit{
			{Name: "Black Kids"},
		},
		LabelInfo: []LabelInfo{
			{
				CatalogNumber: "CAT-001",
				Label:         &Label{Name: "Columbia Records"},
			},
		},
		Media: []Medium{
			{
				Position: 1,
				Tracks: []Track{
					{
						Position: 1,
						Title:    "Hit The Heartbrakes",
						Length:   195000,
						Recording: &Recording{
							Title: "Hit The Heartbrakes",
							ISRC:  []string{"USRC10800001"},
						},
					},
					{
						Position: 2,
						Title:    "I'm Not Gonna Teach Your Boyfriend How to Dance with You",
						Length:   210000,
						Recording: &Recording{
							Title: "I'm Not Gonna Teach Your Boyfriend How to Dance with You",
						},
					},
				},
			},
		},
	}

	pm := ToPlaylistMetadata(release)

	if pm == nil {
		t.Fatal("expected non-nil PlaylistMetadata")
	}

	// Check album info
	if pm.AlbumInfo.Title != "Partie Traumatic" {
		t.Errorf("expected album title %q, got %q", "Partie Traumatic", pm.AlbumInfo.Title)
	}
	if pm.AlbumInfo.Artist != "Black Kids" {
		t.Errorf("expected artist %q, got %q", "Black Kids", pm.AlbumInfo.Artist)
	}
	if pm.AlbumInfo.Year != "2008" {
		t.Errorf("expected year %q, got %q", "2008", pm.AlbumInfo.Year)
	}
	if pm.AlbumInfo.Country != "US" {
		t.Errorf("expected country %q, got %q", "US", pm.AlbumInfo.Country)
	}
	if pm.AlbumInfo.Label != "Columbia Records" {
		t.Errorf("expected label %q, got %q", "Columbia Records", pm.AlbumInfo.Label)
	}
	if pm.AlbumInfo.CatalogNum != "CAT-001" {
		t.Errorf("expected catalog number %q, got %q", "CAT-001", pm.AlbumInfo.CatalogNum)
	}
	if pm.AlbumInfo.TotalTracks != 2 {
		t.Errorf("expected 2 total tracks, got %d", pm.AlbumInfo.TotalTracks)
	}

	// Check tracks
	if len(pm.Tracks) != 2 {
		t.Fatalf("expected 2 tracks, got %d", len(pm.Tracks))
	}

	track1 := pm.Tracks[0]
	if track1.Position != 1 {
		t.Errorf("expected track 1 position 1, got %d", track1.Position)
	}
	if track1.Title != "Hit The Heartbrakes" {
		t.Errorf("expected track 1 title %q, got %q", "Hit The Heartbrakes", track1.Title)
	}
	if track1.Duration != "3:15" {
		t.Errorf("expected track 1 duration %q, got %q", "3:15", track1.Duration)
	}
	if track1.ISRC != "USRC10800001" {
		t.Errorf("expected track 1 ISRC %q, got %q", "USRC10800001", track1.ISRC)
	}
}

func TestToPlaylistMetadataNil(t *testing.T) {
	pm := ToPlaylistMetadata(nil)
	if pm != nil {
		t.Error("expected nil for nil release")
	}
}

func TestToPlaylistMetadataMultiDisc(t *testing.T) {
	release := &Release{
		ID:    "test-id",
		Title: "Double Album",
		ArtistCredit: []ArtistCredit{
			{Name: "Artist"},
		},
		Media: []Medium{
			{
				Position: 1,
				Tracks: []Track{
					{Position: 1, Title: "Disc 1 Track 1"},
					{Position: 2, Title: "Disc 1 Track 2"},
				},
			},
			{
				Position: 2,
				Tracks: []Track{
					{Position: 1, Title: "Disc 2 Track 1"},
					{Position: 2, Title: "Disc 2 Track 2"},
				},
			},
		},
	}

	pm := ToPlaylistMetadata(release)

	if len(pm.Tracks) != 4 {
		t.Fatalf("expected 4 total tracks, got %d", len(pm.Tracks))
	}

	// Check disc numbers
	if pm.Tracks[0].DiscNumber != 1 || pm.Tracks[0].TotalDiscs != 2 {
		t.Errorf("track 1: expected disc 1/2, got %d/%d", pm.Tracks[0].DiscNumber, pm.Tracks[0].TotalDiscs)
	}
	if pm.Tracks[2].DiscNumber != 2 || pm.Tracks[2].TotalDiscs != 2 {
		t.Errorf("track 3: expected disc 2/2, got %d/%d", pm.Tracks[2].DiscNumber, pm.Tracks[2].TotalDiscs)
	}

	// Check track positions are sequential across discs
	if pm.Tracks[0].Position != 1 {
		t.Errorf("expected track 1 position 1, got %d", pm.Tracks[0].Position)
	}
	if pm.Tracks[2].Position != 3 {
		t.Errorf("expected track 3 position 3, got %d", pm.Tracks[2].Position)
	}
}

func TestToPlaylistMetadataWithCover(t *testing.T) {
	release := &Release{
		ID:    "test-id",
		Title: "Test Album",
		ArtistCredit: []ArtistCredit{
			{Name: "Artist"},
		},
	}

	coverURL := "https://example.com/cover.jpg"
	pm := ToPlaylistMetadataWithCover(release, coverURL)

	if pm.AlbumInfo.CoverURL != coverURL {
		t.Errorf("expected cover URL %q, got %q", coverURL, pm.AlbumInfo.CoverURL)
	}
}

func TestToPlaylistMetadataWithCoverEmpty(t *testing.T) {
	release := &Release{
		ID:    "test-id",
		Title: "Test Album",
		ArtistCredit: []ArtistCredit{
			{Name: "Artist"},
		},
	}

	pm := ToPlaylistMetadataWithCover(release, "")

	if pm.AlbumInfo.CoverURL != "" {
		t.Errorf("expected empty cover URL, got %q", pm.AlbumInfo.CoverURL)
	}
}

func TestToPlaylistMetadataDifferentTrackArtist(t *testing.T) {
	release := &Release{
		ID:    "test-id",
		Title: "Compilation Album",
		ArtistCredit: []ArtistCredit{
			{Name: "Various Artists"},
		},
		Media: []Medium{
			{
				Position: 1,
				Tracks: []Track{
					{
						Position: 1,
						Title:    "Track 1",
						Recording: &Recording{
							Title: "Track 1",
							ArtistCredit: []ArtistCredit{
								{Name: "Artist A"},
							},
						},
					},
					{
						Position: 2,
						Title:    "Track 2",
						Recording: &Recording{
							Title: "Track 2",
							ArtistCredit: []ArtistCredit{
								{Name: "Artist B"},
							},
						},
					},
				},
			},
		},
	}

	pm := ToPlaylistMetadata(release)

	if pm.Tracks[0].Artist != "Artist A" {
		t.Errorf("expected track 1 artist %q, got %q", "Artist A", pm.Tracks[0].Artist)
	}
	if pm.Tracks[1].Artist != "Artist B" {
		t.Errorf("expected track 2 artist %q, got %q", "Artist B", pm.Tracks[1].Artist)
	}
}
