package localstorage

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// LocalStorage implements ports.Storage for the local filesystem.
type LocalStorage struct {
	BaseDir string
}

// NewLocalStorage creates a new LocalStorage instance.
func NewLocalStorage(baseDir string) *LocalStorage {
	return &LocalStorage{BaseDir: baseDir}
}

// InitJob creates the job directory.
func (s *LocalStorage) InitJob(ctx context.Context, jobID string) error {
	path := s.GetJobPath(jobID)
	if err := os.MkdirAll(path, 0755); err != nil {
		return fmt.Errorf("failed to create job directory %s: %w", path, err)
	}
	return nil
}

// SaveInput saves the job input metadata.
func (s *LocalStorage) SaveInput(ctx context.Context, jobID string, data []byte) error {
	path := filepath.Join(s.GetJobPath(jobID), "input.json")
	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to save input.json: %w", err)
	}
	return nil
}

// SaveMetadata saves the raw API response.
func (s *LocalStorage) SaveMetadata(ctx context.Context, jobID string, data []byte) error {
	path := filepath.Join(s.GetJobPath(jobID), "metadata_raw.json")
	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to save metadata_raw.json: %w", err)
	}
	return nil
}

// SaveVideo saves the video file.
func (s *LocalStorage) SaveVideo(ctx context.Context, jobID string, reader io.Reader, filename string) error {
	if filename == "" {
		filename = "video.mp4"
	}
	path := filepath.Join(s.GetJobPath(jobID), filename)

	file, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("failed to create video file %s: %w", path, err)
	}
	defer file.Close()

	if _, err := io.Copy(file, reader); err != nil {
		return fmt.Errorf("failed to write video file: %w", err)
	}
	return nil
}

// GetJobPath returns the path for a job directory.
func (s *LocalStorage) GetJobPath(jobID string) string {
	return filepath.Join(s.BaseDir, "jobs", jobID)
}
