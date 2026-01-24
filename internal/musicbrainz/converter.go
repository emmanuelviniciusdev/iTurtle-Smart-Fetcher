package musicbrainz

import (
	"iturtle-smart-fetcher/internal/downloader"
)

// ToPlaylistMetadata converts a MusicBrainz Release to PlaylistMetadata.
func ToPlaylistMetadata(release *Release) *downloader.PlaylistMetadata {
	if release == nil {
		return nil
	}

	pm := &downloader.PlaylistMetadata{
		AlbumInfo: downloader.AlbumMetadata{
			Title:   release.Title,
			Artist:  GetArtistName(release.ArtistCredit),
			Year:    ExtractYear(release.Date),
			Country: release.Country,
		},
	}

	// Set album artist same as artist by default
	pm.AlbumInfo.AlbumArtist = pm.AlbumInfo.Artist

	// Extract label and catalog info
	if len(release.LabelInfo) > 0 {
		if release.LabelInfo[0].Label != nil {
			pm.AlbumInfo.Label = release.LabelInfo[0].Label.Name
		}
		pm.AlbumInfo.CatalogNum = release.LabelInfo[0].CatalogNumber
	}

	// Count total tracks across all media
	totalTracks := 0
	for _, medium := range release.Media {
		totalTracks += len(medium.Tracks)
	}
	pm.AlbumInfo.TotalTracks = totalTracks

	// Extract track metadata
	trackPosition := 0
	for discNum, medium := range release.Media {
		for _, track := range medium.Tracks {
			trackPosition++
			tm := downloader.TrackMetadata{
				Position: trackPosition,
				Title:    track.Title,
				Duration: FormatDuration(track.Length),
			}

			// Multi-disc support
			if len(release.Media) > 1 {
				tm.DiscNumber = discNum + 1
				tm.TotalDiscs = len(release.Media)
			}

			// Get track artist if different from album artist
			if track.Recording != nil {
				if track.Recording.Title != "" {
					tm.Title = track.Recording.Title
				}
				if len(track.Recording.ISRC) > 0 {
					tm.ISRC = track.Recording.ISRC[0]
				}
				// Check if track has different artist
				trackArtist := GetArtistName(track.Recording.ArtistCredit)
				if trackArtist != "" && trackArtist != pm.AlbumInfo.Artist {
					tm.Artist = trackArtist
				}
			}

			pm.Tracks = append(pm.Tracks, tm)
		}
	}

	return pm
}

// ToPlaylistMetadataWithCover is like ToPlaylistMetadata but also sets the cover URL.
func ToPlaylistMetadataWithCover(release *Release, coverURL string) *downloader.PlaylistMetadata {
	pm := ToPlaylistMetadata(release)
	if pm != nil && coverURL != "" {
		pm.AlbumInfo.CoverURL = coverURL
	}
	return pm
}
