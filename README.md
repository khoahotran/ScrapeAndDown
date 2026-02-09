# Video Scraper CLI

A robust command-line tool for scraping video metadata and downloading video files from YouTube.

## 🚀 Features

- **Robust YouTube Support**: Uses a reliable hybrid strategy.
  1.  **Apify** (`streamers/youtube-scraper`) for accurate metadata.
  2.  **yt-dlp** (local binary) ensures video downloading even when APIs fail.
- **Job-Based Architecture**: Each URL is a unique job with full traceability (UUIDs).
- **Data Preservation**: Saves raw metadata JSON exactly as received.
- **Hexagonal Architecture**: Clean separation of core logic, adapters, and CLI.
- **Graceful Shutdown**: Handles OS interrupts cleanly.

## 🛠️ Architecture

- **Core**: Domain logic, ports (interfaces), and Orchestrator.
- **Adapters**:
  - `apify`: Fetches metadata.
  - `ytdlp`: Responsible for extracting video download URLs.
  - `downloader`: Standard HTTP file downloader.
  - `localstorage`: FileSystem persistence.

## 📋 Prerequisites

- **Go** 1.21+
- **Apify API Token**
- **yt-dlp** (Required):
  - The tool handles downloading `yt-dlp.exe` automatically if needed on Windows.
  - Binaries available at: https://github.com/yt-dlp/yt-dlp

## ⚙️ Configuration

Create a `.env` file in the root directory:

```env
APIFY_API_TOKEN=your_apify_api_token
```

## 📦 Installation & Build

```bash
# Clone the repository
git clone <repo-url>
cd ScrapeAndDown

# Tidy dependencies (if needed)
go mod tidy

# Build the binary
go build -o scraper-cli.exe ./cmd/scraper-cli
```

## 🚀 Usage

Run the scraper with a video URL:

```bash
.\scraper-cli.exe -url "https://www.youtube.com/watch?v=dQw4w9WgXcQ"
```

**Options:**

- `-url`: (Required) The video URL to scrape.
- `-data-dir`: (Optional) Custom directory for output data (default: `./data`).

## 📂 Output Structure

```text
data/
└── jobs/
    └── <job-uuid>/
        ├── input.json          # Job input details
        ├── metadata_raw.json   # Full metadata from Apify
        └── video.mp4           # Downloaded video file
```

## 📝 License

MIT
