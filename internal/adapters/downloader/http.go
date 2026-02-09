package downloader

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"time"
)

// HTTPDownloader implements ports.Downloader using standard HTTP.
type HTTPDownloader struct {
	client *http.Client
}

// NewHTTPDownloader creates a new HTTPDownloader.
func NewHTTPDownloader() *HTTPDownloader {
	return &HTTPDownloader{
		client: &http.Client{
			Timeout: 30 * time.Minute, // Videos can be large
		},
	}
}

// Download fetches the video from the given URL.
func (d *HTTPDownloader) Download(ctx context.Context, videoURL string) (io.ReadCloser, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, videoURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := d.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to download video: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	return resp.Body, nil
}
