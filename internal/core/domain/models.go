package domain

import "time"

// Job represents a single scraping job.
type Job struct {
	ID        string    `json:"job_id"`
	URL       string    `json:"url"`
	Platform  string    `json:"platform"` // "youtube" or "tiktok"
	CreatedAt time.Time `json:"created_at"`
}

// JobResult holds the outcome of a completed job.
type JobResult struct {
	Job          Job
	MetadataPath string
	VideoPath    string
	Success      bool
	ErrorMessage string
	CompletedAt  time.Time
}
