package ytdlp

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"
)

// YtDlpDownloader uses the local yt-dlp binary to fetch video URLs.
type YtDlpDownloader struct {
	binaryPath string
}

// NewYtDlpDownloader creates a new downloader.
func NewYtDlpDownloader() *YtDlpDownloader {
	// Check if yt-dlp.exe exists in current directory
	if _, err := os.Stat("yt-dlp.exe"); err == nil {
		return &YtDlpDownloader{binaryPath: ".\\yt-dlp.exe"}
	}
	return &YtDlpDownloader{
		binaryPath: "yt-dlp", // Assumes yt-dlp is in PATH
	}
}

// GetVideoURL fetches the direct download link using yt-dlp --get-url.
func (d *YtDlpDownloader) GetVideoURL(ctx context.Context, videoURL string) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()

	// -f best: Select best quality
	// --get-url: Only output the URL
	// --no-warnings: Suppress warnings
	cmd := exec.CommandContext(ctx, d.binaryPath, "-f", "b", "--get-url", "--no-warnings", videoURL)

	var out bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("yt-dlp failed: %w, stderr: %s", err, stderr.String())
	}

	urlStr := strings.TrimSpace(out.String())
	if urlStr == "" {
		return "", fmt.Errorf("yt-dlp returned empty URL")
	}

	// yt-dlp might return multiple URLs (video + audio), just take the first one
	urls := strings.Split(urlStr, "\n")
	if len(urls) > 0 {
		return urls[0], nil
	}

	return urlStr, nil
}
