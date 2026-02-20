#!/usr/bin/env bash
# Social Video Analyzer — download and analyze any social media video
# Usage: ./analyze_video.sh <URL> [custom_prompt]
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
WORKSPACE="$(cd "$SCRIPT_DIR/../../.." && pwd)"
PY="$WORKSPACE/.venv/bin/python3"
YTDLP="$WORKSPACE/.venv/bin/yt-dlp"
VIDEO_ANALYZER="$WORKSPACE/tools/video_analyze.py"

URL="${1:?Usage: analyze_video.sh <URL> [prompt]}"
PROMPT="${2:-Analyze this video comprehensively. Provide: 1) FULL TRANSCRIPT of all spoken words, 2) VISUAL DESCRIPTION scene by scene, 3) KEY TAKEAWAYS as bullet points, 4) CONTENT FORMAT ANALYSIS (platform style, production quality, target audience, duration estimate).}"

# Temp dir for downloads
TMPDIR=$(mktemp -d /tmp/video-analyze-XXXXXX)
trap "rm -rf $TMPDIR" EXIT

echo "[video-analyzer] Downloading from: $URL" >&2

# Download video (best quality under 100MB, prefer mp4)
"$YTDLP" \
  --no-playlist \
  --format 'bv*[filesize<100M]+ba/b[filesize<100M]/bv*+ba/b' \
  --merge-output-format mp4 \
  --output "$TMPDIR/video.%(ext)s" \
  --no-warnings \
  --quiet \
  "$URL" 2>&1 >&2

# Find downloaded file
VIDEO_FILE=$(find "$TMPDIR" -type f \( -name '*.mp4' -o -name '*.webm' -o -name '*.mkv' \) | head -1)

if [ -z "$VIDEO_FILE" ]; then
  echo "ERROR: Failed to download video from $URL" >&2
  exit 1
fi

SIZE=$(stat -c%s "$VIDEO_FILE" 2>/dev/null || stat -f%z "$VIDEO_FILE")
echo "[video-analyzer] Downloaded: $(basename "$VIDEO_FILE") ($(( SIZE / 1024 / 1024 ))MB)" >&2

# Analyze with Gemini
GOOGLE_AI_API_KEY="${GOOGLE_AI_API_KEY:?Set GOOGLE_AI_API_KEY}" \
  "$PY" "$VIDEO_ANALYZER" "$VIDEO_FILE" "$PROMPT"
