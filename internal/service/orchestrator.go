package service

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/url"
	"strings"
	"time"

	"github.com/google/uuid"

	"scrapeanddown/internal/adapters/ytdlp"
	"scrapeanddown/internal/core/domain"
	"scrapeanddown/internal/core/ports"
)

// Orchestrator coordinates the scraping workflow.
type Orchestrator struct {
	scraper    ports.Scraper
	downloader ports.Downloader
	storage    ports.Storage
	ytDlp      *ytdlp.YtDlpDownloader
	logger     *log.Logger
}

// NewOrchestrator creates a new Orchestrator.
func NewOrchestrator(
	scraper ports.Scraper,
	downloader ports.Downloader,
	storage ports.Storage,
	ytDlp *ytdlp.YtDlpDownloader,
	logger *log.Logger,
) *Orchestrator {
	return &Orchestrator{
		scraper:    scraper,
		downloader: downloader,
		storage:    storage,
		ytDlp:      ytDlp,
		logger:     logger,
	}
}

// RunJob executes a complete scraping job for the given URL.
func (o *Orchestrator) RunJob(ctx context.Context, url string) (*domain.JobResult, error) {
	// Generate job ID and create job
	jobID := uuid.New().String()
	job := domain.Job{
		ID:        jobID,
		URL:       url,
		Platform:  detectPlatform(url),
		CreatedAt: time.Now().UTC(),
	}

	result := &domain.JobResult{Job: job, Success: false}
	o.logger.Printf("[JOB %s] Starting job for URL: %s", jobID, url)

	if err := o.storage.InitJob(ctx, jobID); err != nil {
		result.ErrorMessage = fmt.Sprintf("failed to init job: %v", err)
		o.logger.Printf("[JOB %s] ERROR: %s", jobID, result.ErrorMessage)
		return result, err
	}

	inputData, _ := json.MarshalIndent(job, "", "  ")
	_ = o.storage.SaveInput(ctx, jobID, inputData)

	// Step 3: Scrape Metadata (Apify)
	o.logger.Printf("[JOB %s] Scraping metadata via Apify...", jobID)
	scrapeResult, err := o.scraper.Scrape(ctx, url)
	if err != nil {
		result.ErrorMessage = fmt.Sprintf("failed to scrape metadata: %v", err)
		o.logger.Printf("[JOB %s] ERROR: %s", jobID, result.ErrorMessage)
		return result, err
	}
	o.logger.Printf("[JOB %s] Apify scrape completed, saved metadata", jobID)

	if err := o.storage.SaveMetadata(ctx, jobID, scrapeResult.RawMetadata); err != nil {
		result.ErrorMessage = fmt.Sprintf("failed to save metadata: %v", err)
		return result, err
	}
	result.MetadataPath = o.storage.GetJobPath(jobID) + "/metadata_raw.json"

	// Step 4: Get Video URL via yt-dlp
	var videoDownloadURL string
	
	if job.Platform == "youtube" {
		o.logger.Printf("[JOB %s] Fetching download link via yt-dlp...", jobID)
		ytUrl, ytErr := o.ytDlp.GetVideoURL(ctx, url)
		if ytErr == nil && ytUrl != "" {
			videoDownloadURL = ytUrl
			o.logger.Printf("[JOB %s] Success: Got video URL from yt-dlp", jobID)
		} else {
			result.ErrorMessage = fmt.Sprintf("yt-dlp failed: %v", ytErr)
			o.logger.Printf("[JOB %s] ERROR: %s", jobID, result.ErrorMessage)
			return result, ytErr
		}
	} else {
		// TikTok fallback logic (Apify)
		videoDownloadURL = scrapeResult.VideoURL
	}

	if videoDownloadURL == "" {
		return result, fmt.Errorf("no video url resolved")
	}

	// Step 5: Download
	o.logger.Printf("[JOB %s] Downloading video stream...", jobID)
	videoReader, err := o.downloader.Download(ctx, videoDownloadURL)
	if err != nil {
		result.ErrorMessage = fmt.Sprintf("failed to download video: %v", err)
		o.logger.Printf("[JOB %s] ERROR: %s", jobID, result.ErrorMessage)
		return result, err
	}
	defer videoReader.Close()

	if err := o.storage.SaveVideo(ctx, jobID, videoReader, "video.mp4"); err != nil {
		result.ErrorMessage = fmt.Sprintf("failed to save video: %v", err)
		return result, err
	}
	result.VideoPath = o.storage.GetJobPath(jobID) + "/video.mp4"
	o.logger.Printf("[JOB %s] Saved video.mp4", jobID)

	// Success
	result.Success = true
	result.CompletedAt = time.Now().UTC()
	
	o.logger.Printf("[JOB %s] Job completed successfully!", jobID)
	o.logger.Printf("[JOB %s] Artifacts saved to: %s", jobID, o.storage.GetJobPath(jobID))

	// Print summary
	fmt.Println("\n=== Job Summary ===")
	fmt.Printf("Job ID:       %s\n", result.Job.ID)
	fmt.Printf("Platform:     %s\n", result.Job.Platform)
	fmt.Printf("Success:      %v\n", result.Success)
	if !result.Success {
		fmt.Printf("Error:        %s\n", result.ErrorMessage)
	} else {
		fmt.Printf("Metadata:     %s\n", result.MetadataPath)
		fmt.Printf("Video:        %s\n", result.VideoPath)
	}
	fmt.Printf("Completed At: %s\n", result.CompletedAt.Format(time.RFC3339))

	return result, nil
}

func detectPlatform(url string) string {
	if containsAny(url, "youtube.com", "youtu.be") {
		return "youtube"
	}
	if containsAny(url, "tiktok.com") {
		return "tiktok"
	}
	return "unknown"
}

func containsAny(s string, substrs ...string) bool {
	for _, sub := range substrs {
		if len(s) >= len(sub) {
			for i := 0; i <= len(s)-len(sub); i++ {
				if s[i:i+len(sub)] == sub {
					return true
				}
			}
		}
	}
	return false
}

func extractVideoID(videoURL string) string {
	u, err := url.Parse(videoURL)
	if err != nil {
		return ""
	}
	if u.Host == "youtu.be" {
		return strings.TrimPrefix(u.Path, "/")
	}
	qty := u.Query()
	return qty.Get("v")
}
