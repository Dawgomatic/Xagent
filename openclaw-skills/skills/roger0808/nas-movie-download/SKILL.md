---
name: nas-movie-download
description: Search and download movies via Jackett and qBittorrent. Use when user wants to download movies or videos from torrent sources, search for specific movie titles, or manage movie downloads. Now includes automatic subtitle download support.
---

# NAS Movie Download

Automated movie downloading system using Jackett for torrent search and qBittorrent for download management.

**新功能：自动字幕下载支持！** 🎬

## Configuration

### Environment Variables

Set these environment variables for the skill to function properly:

**Jackett Configuration:**
- `JACKETT_URL`: Jackett service URL (default: http://192.168.1.246:9117)
- `JACKETT_API_KEY`: Jackett API key (default: o5gp976vq8cm084cqkcv30av9v3e5jpy)

**qBittorrent Configuration:**
- `QB_URL`: qBittorrent Web UI URL (default: http://192.168.1.246:8888)
- `QB_USERNAME`: qBittorrent username (default: admin)
- `QB_PASSWORD`: qBittorrent password (default: adminadmin)

**Subtitle Configuration:**
- `OPENSUBTITLES_API_KEY`: OpenSubtitles API key (optional, can also save to `config/opensubtitles.key`)
- `SUBTITLE_LANGUAGES`: Default subtitle languages (default: zh-cn,en)

### OpenSubtitles Setup

1. 注册账号：https://www.opensubtitles.com
2. 获取 API Key
3. 保存到配置文件：`echo "your-api-key" > config/opensubtitles.key`

### Indexer Setup

The skill works with Jackett indexers. Currently configured indexers:
- The Pirate Bay
- TheRARBG
- YTS

Ensure these indexers are enabled and configured in your Jackett installation for best results.

## Usage

### Search Movies

Search for movies without downloading:

```bash
scripts/jackett-search.sh -q "Inception"
scripts/jackett-search.sh -q "The Matrix"
scripts/jackett-search.sh -q "死期将至"  # Chinese movie names supported
```

### Download with Automatic Subtitles 🆕

One-click download with automatic subtitle fetching:

```bash
# Download movie and automatically download subtitles after completion
scripts/download-movie.sh -q "Young Sheldon" -s -w

# Download with specific languages
scripts/download-movie.sh -q "Community" -s -l zh-cn,en

# Download movie only (no subtitles)
scripts/download-movie.sh -q "The Matrix"
```

**参数说明：**
- `-s, --with-subtitle`: 启用自动字幕下载
- `-w, --wait`: 等待下载完成后自动下载字幕
- `-l, --languages`: 指定字幕语言（默认：zh-cn,en）

### Manual Download Workflow

For more control over the download process:

1. Search: `scripts/jackett-search.sh -q "movie name"`
2. Review results and copy magnet link
3. Add to qBittorrent: `scripts/qbittorrent-add.sh -m "magnet:?xt=urn:btih:..."`
4. Download subtitles: `scripts/subtitle-download.sh -d "/path/to/downloaded/files"`

### Subtitle Download Only

Download subtitles for existing video files:

```bash
# Single file
scripts/subtitle-download.sh -f "/path/to/video.mkv" -l zh-cn,en

# Entire directory (recursive)
scripts/subtitle-download.sh -d "/path/to/tv/show" -r

# Specific languages
scripts/subtitle-download.sh -d "/media/Young Sheldon" -l zh-cn,en,ja
```

**Language Codes:**
- `zh-cn`: 中文简体
- `zh-tw`: 中文繁体
- `en`: 英文
- `ja`: 日文
- `ko`: 韩文

### Test Configuration

Verify your Jackett and qBittorrent setup:

```bash
scripts/test-config.sh
```

## Quality Selection

The skill automatically prioritizes quality in this order:

1. **4K/UHD**: Contains "4K", "2160p", "UHD"
2. **1080P/Full HD**: Contains "1080p", "FHD"
3. **720P/HD**: Contains "720p", "HD"
4. **Other**: Other quality levels

When using `download-movie.sh`, the highest quality available torrent will be selected automatically.

## Script Details

### jackett-search.sh

Search Jackett for torrents.

**Parameters:**
- `-q, --query`: Search query (required)
- `-u, --url`: Jackett URL (optional, uses env var)
- `-k, --api-key`: API key (optional, uses env var)

**Example:**
```bash
scripts/jackett-search.sh -q "Inception" -u http://192.168.1.246:9117
```

### qbittorrent-add.sh

Add torrent to qBittorrent.

**Parameters:**
- `-m, --magnet`: Magnet link (required)
- `-u, --url`: qBittorrent URL (optional, uses env var)
- `-n, --username`: Username (optional, uses env var)
- `-p, --password`: Password (optional, uses env var)

**Example:**
```bash
scripts/qbittorrent-add.sh -m "magnet:?xt=urn:btih:..."
```

### download-movie.sh

One-click search and download with optional subtitle support.

**Parameters:**
- `-q, --query`: Movie name (required)
- `-s, --with-subtitle`: Enable automatic subtitle download
- `-w, --wait`: Wait for download to complete before downloading subtitles
- `-l, --languages`: Subtitle languages (default: zh-cn,en)

**Example:**
```bash
# Basic download
scripts/download-movie.sh -q "The Matrix"

# Download with subtitles
scripts/download-movie.sh -q "Young Sheldon" -s -w -l zh-cn,en
```

### subtitle-download.sh 🆕

Download subtitles for video files using OpenSubtitles API.

**Parameters:**
- `-f, --file`: Single video file path
- `-d, --directory`: Process all videos in directory
- `-l, --languages`: Subtitle languages, comma-separated (default: zh-cn,en)
- `-k, --api-key`: OpenSubtitles API Key (optional if configured)
- `-r, --recursive`: Recursively process subdirectories
- `-h, --help`: Show help

**Example:**
```bash
# Single file
scripts/subtitle-download.sh -f "/media/movie.mkv"

# Batch process directory
scripts/subtitle-download.sh -d "/media/TV Shows" -r -l zh-cn,en
```

**Features:**
- Automatically parses video filenames (TV episodes, movies)
- Downloads best-rated subtitles for each language
- Renames subtitles to match video filenames
- Skips existing subtitle files
- Supports batch processing

## Tips and Best Practices

- **Use English movie names** for better search results
- **Check Jackett indexer status** if searches return no results
- **Monitor qBittorrent** to manage download progress
- **Consider storage space** when downloading 4K content
- **Test configuration** periodically to ensure services are running
- **For TV series**: Use `-s -w` flag to auto-download subtitles for all episodes

## Troubleshooting

### No Search Results

1. Verify Jackett is running: `curl http://192.168.1.246:9117`
2. Check Jackett indexers are enabled in Jackett UI
3. Try English movie names
4. Verify API key is correct

### qBittorrent Connection Failed

1. Confirm qBittorrent is running
2. Check Web UI is enabled in qBittorrent settings
3. Verify username and password
4. Ensure network connectivity to qBittorrent server

### Subtitle Download Issues

1. **No API Key**: Save your key to `config/opensubtitles.key` or use `-k` flag
2. **No subtitles found**: Try different language codes or the video may not have subtitles available
3. **API limit**: OpenSubtitles free tier has rate limits; wait a few minutes and retry

### Permission Issues

Ensure scripts have execute permissions:

```bash
chmod +x scripts/*.sh
```

## Security Notes

- Keep API keys secure and don't commit them to version control
- Use HTTPS connections when possible
- Consider setting up VPN for torrent traffic
- Monitor qBittorrent for unauthorized downloads

## Dependencies

- `curl`: For HTTP requests
- `jq`: For JSON parsing
- `bc`: For floating point calculations (subtitle download progress)
- Bash shell

Install dependencies if missing:
```bash
apt-get install curl jq bc
```

## Changelog

### v2.0 - 2025-02-17
- ✅ Added automatic subtitle download support
- ✅ New `subtitle-download.sh` script
- ✅ Updated `download-movie.sh` with `-s` and `-w` flags
- ✅ Support for OpenSubtitles API
- ✅ Multi-language subtitle support (zh-cn, en, ja, ko, etc.)
