package apify

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"scrapeanddown/internal/core/ports"
)

const (
	apifyBaseURL = "https://api.apify.com/v2"
	// Actor IDs for different platforms (using internal Apify IDs)
	youtubeMetadataActorID = "h7sDV53CddomktSi5"        // streamers/youtube-scraper
	youtubeDownloadActorID = "apify~youtube-downloader" // Unused (replaced by fallback strategy)
	tiktokActorID          = "GdWCkxBtKWOsKjdch"        // clockworks~tiktok-scraper
)

// ApifyScraper implements ports.Scraper using Apify REST API.
type ApifyScraper struct {
	apiToken string
	client   *http.Client
}

// NewApifyScraper creates a new ApifyScraper.
// Reads the API token from APIFY_API_TOKEN environment variable.
func NewApifyScraper() (*ApifyScraper, error) {
	token := os.Getenv("APIFY_API_TOKEN")
	if token == "" {
		return nil, fmt.Errorf("APIFY_API_TOKEN environment variable not set")
	}
	return &ApifyScraper{
		apiToken: token,
		client: &http.Client{
			Timeout: 5 * time.Minute,
		},
	}, nil
}

// Scrape fetches metadata for the given video URL using Apify.
func (s *ApifyScraper) Scrape(ctx context.Context, videoPageURL string) (*ports.ScrapeResult, error) {
	platform := detectPlatform(videoPageURL)
	if platform == "" {
		return nil, fmt.Errorf("unsupported platform for URL: %s", videoPageURL)
	}

	actorID := s.getActorID(platform)
	if actorID == "" {
		return nil, fmt.Errorf("no actor configured for platform: %s", platform)
	}

	// Start the actor run
	runID, err := s.startActorRun(ctx, actorID, videoPageURL, platform)
	if err != nil {
		return nil, fmt.Errorf("failed to start actor run: %w", err)
	}

	// Wait for completion and get results
	rawData, err := s.waitAndGetResults(ctx, runID)
	if err != nil {
		return nil, fmt.Errorf("failed to get results: %w", err)
	}

	// Extract video URL if possible (optional for YouTube since we use RapidAPI)
	videoURL, _ := s.extractVideoURL(rawData, platform)

	return &ports.ScrapeResult{
		RawMetadata: rawData,
		VideoURL:    videoURL,
	}, nil
}

// scrapeYouTubeDualActor is removed as we now handle downloads via RapidAPI/yt-dlp in Orchestrator

func (s *ApifyScraper) getActorID(platform string) string {
	switch platform {
	case "youtube":
		return youtubeMetadataActorID
	case "tiktok":
		return tiktokActorID
	default:
		return ""
	}
}

func (s *ApifyScraper) startActorRun(ctx context.Context, actorID, videoURL, platform string) (string, error) {
	url := fmt.Sprintf("%s/acts/%s/runs?token=%s", apifyBaseURL, actorID, s.apiToken)

	// Build input based on platform
	input := s.buildInput(videoURL, platform)
	body, _ := json.Marshal(input)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		respBody, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("failed to start actor: status %d, body: %s", resp.StatusCode, string(respBody))
	}

	var result struct {
		Data struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}

	return result.Data.ID, nil
}

func (s *ApifyScraper) buildInput(videoURL, platform string) map[string]interface{} {
	switch platform {
	case "youtube":
		return map[string]interface{}{
			"startUrls":  []map[string]string{{"url": videoURL}},
			"maxResults": 1,
		}
	case "tiktok":
		return map[string]interface{}{
			"postURLs":       []string{videoURL},
			"resultsPerPage": 1,
		}
	default:
		return map[string]interface{}{"url": videoURL}
	}
}

func (s *ApifyScraper) waitAndGetResults(ctx context.Context, runID string) ([]byte, error) {
	// Poll for run completion
	statusURL := fmt.Sprintf("%s/actor-runs/%s?token=%s", apifyBaseURL, runID, s.apiToken)

	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(3 * time.Second):
		}

		req, _ := http.NewRequestWithContext(ctx, http.MethodGet, statusURL, nil)
		resp, err := s.client.Do(req)
		if err != nil {
			return nil, err
		}

		var status struct {
			Data struct {
				Status           string `json:"status"`
				DefaultDatasetID string `json:"defaultDatasetId"`
			} `json:"data"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&status); err != nil {
			resp.Body.Close()
			return nil, err
		}
		resp.Body.Close()

		switch status.Data.Status {
		case "SUCCEEDED":
			return s.getDatasetItems(ctx, status.Data.DefaultDatasetID)
		case "FAILED", "ABORTED", "TIMED-OUT":
			return nil, fmt.Errorf("actor run failed with status: %s", status.Data.Status)
		}
		// Still running, continue polling
	}
}

func (s *ApifyScraper) getDatasetItems(ctx context.Context, datasetID string) ([]byte, error) {
	url := fmt.Sprintf("%s/datasets/%s/items?token=%s", apifyBaseURL, datasetID, s.apiToken)

	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	resp, err := s.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	return io.ReadAll(resp.Body)
}

func (s *ApifyScraper) extractVideoURL(rawData []byte, platform string) (string, error) {
	var items []map[string]interface{}
	if err := json.Unmarshal(rawData, &items); err != nil {
		return "", err
	}

	if len(items) == 0 {
		return "", fmt.Errorf("no results returned from scraper")
	}

	item := items[0]

	// For other platforms, try common video URL field names
	fieldNames := []string{"videoUrl", "video_url", "downloadUrl", "download_url", "videoPlayUrl"}
	for _, field := range fieldNames {
		if val, ok := item[field].(string); ok && val != "" {
			return val, nil
		}
	}

	// Try extracting from 'formats' array (typical for some YouTube scrapers)
	if formats, ok := item["formats"].([]interface{}); ok && len(formats) > 0 {
		if format, ok := formats[len(formats)-1].(map[string]interface{}); ok {
			if url, ok := format["url"].(string); ok && url != "" {
				return url, nil
			}
		}
	}

	return "", fmt.Errorf("could not find video URL in response")
}

func detectPlatform(url string) string {
	lowerURL := strings.ToLower(url)
	if strings.Contains(lowerURL, "youtube.com") || strings.Contains(lowerURL, "youtu.be") {
		return "youtube"
	}
	if strings.Contains(lowerURL, "tiktok.com") {
		return "tiktok"
	}
	return ""
}
