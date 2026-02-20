#!/usr/bin/env python3
"""
Social Video Analyzer — Download and analyze any social media video.

Downloads via yt-dlp, uploads to Gemini for native video understanding,
returns structured analysis with transcript, visuals, and takeaways.

Usage:
    python3 analyze_video.py "https://youtube.com/watch?v=..." 
    python3 analyze_video.py "https://tiktok.com/..." --prompt "What product is shown?"
    python3 analyze_video.py "https://instagram.com/reel/..." --json

Requires: GOOGLE_AI_API_KEY environment variable
"""

import argparse
import json
import mimetypes
import os
import subprocess
import sys
import tempfile
import time
import urllib.request
import urllib.error

WORKSPACE = os.path.abspath(os.path.join(os.path.dirname(__file__), "..", "..", ".."))
YTDLP = os.path.join(WORKSPACE, ".venv", "bin", "yt-dlp")
GOOGLE_API_KEY = os.environ.get("GOOGLE_AI_API_KEY", "")
MODEL = "gemini-2.5-flash"
BASE_URL = "https://generativelanguage.googleapis.com"

DEFAULT_PROMPT = """Analyze this video comprehensively. Return your analysis in this exact format:

## Transcript
[Full transcript of all spoken words. If no speech, write "No spoken content."]

## Visual Description
[Scene-by-scene description of what's shown visually]

## Key Takeaways
- [Bullet point 1]
- [Bullet point 2]
- [Continue as needed]

## Content Format Analysis
- **Platform Style**: [e.g., TikTok vertical, YouTube long-form, Instagram Reel]
- **Production Quality**: [Amateur/Semi-pro/Professional]
- **Content Type**: [Tutorial, Entertainment, News, Ad, Vlog, etc.]
- **Target Audience**: [Who this is for]
- **Estimated Duration**: [Your estimate]
- **Notable Elements**: [Music, effects, text overlays, etc.]
"""


def download_video(url: str, output_dir: str) -> str:
    """Download video using yt-dlp. Returns path to downloaded file."""
    output_template = os.path.join(output_dir, "video.%(ext)s")
    cmd = [
        YTDLP,
        "--no-playlist",
        "--format", "bv*[filesize<100M]+ba/b[filesize<100M]/bv*+ba/b",
        "--merge-output-format", "mp4",
        "--output", output_template,
        "--no-warnings",
        url,
    ]
    print(f"[video-analyzer] Downloading: {url}", file=sys.stderr)
    result = subprocess.run(cmd, capture_output=True, text=True)
    if result.returncode != 0:
        raise RuntimeError(f"yt-dlp failed: {result.stderr}")

    # Find the downloaded file
    for f in os.listdir(output_dir):
        if f.startswith("video."):
            path = os.path.join(output_dir, f)
            size_mb = os.path.getsize(path) / (1024 * 1024)
            print(f"[video-analyzer] Downloaded: {f} ({size_mb:.1f}MB)", file=sys.stderr)
            return path
    raise RuntimeError("No video file found after download")


def upload_to_gemini(filepath: str) -> tuple[str, str]:
    """Upload video to Gemini Files API. Returns (file_uri, mime_type)."""
    filesize = os.path.getsize(filepath)
    mime_type = mimetypes.guess_type(filepath)[0] or "video/mp4"
    display_name = os.path.basename(filepath)

    # Initiate resumable upload
    headers = {
        "X-Goog-Upload-Protocol": "resumable",
        "X-Goog-Upload-Command": "start",
        "X-Goog-Upload-Header-Content-Length": str(filesize),
        "X-Goog-Upload-Header-Content-Type": mime_type,
        "Content-Type": "application/json",
    }
    metadata = json.dumps({"file": {"display_name": display_name}}).encode()
    req = urllib.request.Request(
        f"{BASE_URL}/upload/v1beta/files?key={GOOGLE_API_KEY}",
        data=metadata, headers=headers, method="POST",
    )
    with urllib.request.urlopen(req) as resp:
        upload_url = resp.headers.get("X-Goog-Upload-URL")
    if not upload_url:
        raise RuntimeError("Failed to get upload URL from Gemini")

    # Upload bytes
    with open(filepath, "rb") as f:
        file_data = f.read()
    req2 = urllib.request.Request(
        upload_url, data=file_data,
        headers={
            "X-Goog-Upload-Offset": "0",
            "X-Goog-Upload-Command": "upload, finalize",
            "Content-Length": str(filesize),
        },
        method="PUT",
    )
    with urllib.request.urlopen(req2) as resp:
        result = json.loads(resp.read())

    file_uri = result.get("file", {}).get("uri", "")
    file_name = result.get("file", {}).get("name", "")
    state = result.get("file", {}).get("state", "")
    print(f"[video-analyzer] Uploaded to Gemini. State: {state}", file=sys.stderr)

    # Wait for processing
    if state == "PROCESSING":
        for _ in range(60):
            time.sleep(5)
            check = urllib.request.Request(f"{BASE_URL}/v1beta/{file_name}?key={GOOGLE_API_KEY}")
            with urllib.request.urlopen(check) as resp:
                status = json.loads(resp.read())
            state = status.get("state", "")
            if state == "ACTIVE":
                print("[video-analyzer] Processing complete.", file=sys.stderr)
                break
            if state == "FAILED":
                raise RuntimeError(f"Gemini processing failed: {status}")

    return file_uri, mime_type


def analyze_video(file_uri: str, mime_type: str, prompt: str) -> str:
    """Send video to Gemini for analysis."""
    payload = {
        "contents": [{
            "parts": [
                {"file_data": {"mime_type": mime_type, "file_uri": file_uri}},
                {"text": prompt},
            ]
        }],
        "generationConfig": {"temperature": 0.3, "maxOutputTokens": 8192},
    }
    req = urllib.request.Request(
        f"{BASE_URL}/v1beta/models/{MODEL}:generateContent?key={GOOGLE_API_KEY}",
        data=json.dumps(payload).encode(),
        headers={"Content-Type": "application/json"},
        method="POST",
    )
    with urllib.request.urlopen(req, timeout=180) as resp:
        result = json.loads(resp.read())

    candidates = result.get("candidates", [])
    if candidates:
        parts = candidates[0].get("content", {}).get("parts", [])
        return "\n".join(p.get("text", "") for p in parts if "text" in p)
    return f"No response from Gemini. Raw: {json.dumps(result)}"


def main():
    parser = argparse.ArgumentParser(description="Analyze social media videos with AI")
    parser.add_argument("url", help="Video URL (YouTube, TikTok, Instagram, X, etc.)")
    parser.add_argument("--prompt", "-p", default=DEFAULT_PROMPT, help="Custom analysis prompt")
    parser.add_argument("--json", "-j", action="store_true", help="Output as JSON")
    parser.add_argument("--keep", "-k", action="store_true", help="Keep downloaded video")
    args = parser.parse_args()

    if not GOOGLE_API_KEY:
        print("Error: Set GOOGLE_AI_API_KEY environment variable", file=sys.stderr)
        sys.exit(1)

    if not os.path.exists(YTDLP):
        print(f"Error: yt-dlp not found at {YTDLP}", file=sys.stderr)
        sys.exit(1)

    tmpdir = tempfile.mkdtemp(prefix="video-analyze-")
    try:
        video_path = download_video(args.url, tmpdir)
        file_uri, mime_type = upload_to_gemini(video_path)
        analysis = analyze_video(file_uri, mime_type, args.prompt)

        if args.json:
            print(json.dumps({
                "url": args.url,
                "analysis": analysis,
                "model": MODEL,
            }, indent=2))
        else:
            print(analysis)

        if args.keep:
            import shutil
            keep_path = os.path.join(os.getcwd(), os.path.basename(video_path))
            shutil.copy2(video_path, keep_path)
            print(f"[video-analyzer] Saved to: {keep_path}", file=sys.stderr)
    finally:
        if not args.keep:
            import shutil
            shutil.rmtree(tmpdir, ignore_errors=True)


if __name__ == "__main__":
    main()
