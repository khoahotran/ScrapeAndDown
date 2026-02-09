package ports

import (
	"context"
	"io"
)

// ScrapeResult holds the raw metadata from a scraping operation.
// We use []byte to preserve the exact API response without data loss.
type ScrapeResult struct {
	RawMetadata []byte // Full JSON response, untouched
	VideoURL    string // Extracted video download URL
}

// Scraper defines the contract for fetching video metadata from an API.
type Scraper interface {
	// Scrape retrieves metadata for the given video URL.
	// Returns the raw API response and the direct video download URL.
	Scrape(ctx context.Context, videoPageURL string) (*ScrapeResult, error)
}

// Downloader defines the contract for downloading video files.
type Downloader interface {
	// Download fetches the video from the given URL.
	// Returns a ReadCloser that the caller must close.
	Download(ctx context.Context, videoURL string) (io.ReadCloser, error)
}

// Storage defines the contract for persisting job artifacts.
type Storage interface {
	// InitJob creates the job directory structure.
	InitJob(ctx context.Context, jobID string) error

	// SaveInput saves the job input metadata (URL, timestamp, etc.).
	SaveInput(ctx context.Context, jobID string, data []byte) error

	// SaveMetadata saves the raw API response without modification.
	SaveMetadata(ctx context.Context, jobID string, data []byte) error

	// SaveVideo saves the video file from the provided reader.
	SaveVideo(ctx context.Context, jobID string, reader io.Reader, filename string) error

	// GetJobPath returns the filesystem path for a given job ID.
	GetJobPath(jobID string) string
}
