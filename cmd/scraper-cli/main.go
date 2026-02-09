package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/joho/godotenv"
	"scrapeanddown/internal/adapters/apify"
	"scrapeanddown/internal/adapters/downloader"
	"scrapeanddown/internal/adapters/localstorage"
	"scrapeanddown/internal/adapters/ytdlp"
	"scrapeanddown/internal/service"
)

func main() {
	// Load .env file if it exists
	if err := godotenv.Load(); err != nil {
		// It's okay if .env doesn't exist, environment variables might be set manually
		log.Println("No .env file found")
	}

	// Parse flags
	url := flag.String("url", "", "YouTube or TikTok video URL to scrape")
	dataDir := flag.String("data-dir", "./data", "Base directory for storing job data")
	flag.Parse()

	if *url == "" {
		fmt.Println("Usage: scraper-cli -url <video-url> [-data-dir <path>]")
		fmt.Println("\nExample:")
		fmt.Println("  scraper-cli -url https://www.youtube.com/watch?v=dQw4w9WgXcQ")
		fmt.Println("  scraper-cli -url https://www.tiktok.com/@user/video/1234567890")
		os.Exit(1)
	}

	// Setup logger
	logger := log.New(os.Stdout, "", log.LstdFlags)

	logger.Println("=== Video Scraper CLI ===")
	logger.Printf("URL: %s", *url)
	logger.Printf("Data Directory: %s", *dataDir)

	// Initialize adapters
	scraper, err := apify.NewApifyScraper()
	if err != nil {
		logger.Fatalf("Failed to initialize scraper: %v", err)
	}

	ytDlpClient := ytdlp.NewYtDlpDownloader()

	dl := downloader.NewHTTPDownloader()
	storage := localstorage.NewLocalStorage(*dataDir)

	// Create orchestrator
	orchestrator := service.NewOrchestrator(scraper, dl, storage, ytDlpClient, logger)

	// Setup context with cancellation
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigChan
		logger.Println("\nReceived interrupt signal, cancelling...")
		cancel()
	}()

	// Run the job
	result, err := orchestrator.RunJob(ctx, *url)
	if err != nil {
		logger.Printf("Job failed: %v", err)
		os.Exit(1)
	}

	// Print summary
	fmt.Println("\n=== Job Summary ===")
	fmt.Printf("Job ID:       %s\n", result.Job.ID)
	fmt.Printf("Platform:     %s\n", result.Job.Platform)
	fmt.Printf("Success:      %t\n", result.Success)
	fmt.Printf("Metadata:     %s\n", result.MetadataPath)
	fmt.Printf("Video:        %s\n", result.VideoPath)
	fmt.Printf("Completed At: %s\n", result.CompletedAt.Format("2006-01-02 15:04:05 UTC"))
}
