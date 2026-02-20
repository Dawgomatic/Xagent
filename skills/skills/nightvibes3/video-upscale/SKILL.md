---
name: video_upscale
description: "AI-powered video upscaling with Real-ESRGAN and Waifu2x. Use when user asks to enhance, upscale, improve video quality, make HD/4K. Supports anime and real footage with progress tracking."
metadata: {"openclaw":{"emoji":"🎬","triggers":["upscale","enhance","make HD","make 4K","improve quality"]}}
---

# Video Upscale Skill

AI-powered video upscaling with progress tracking and job isolation.

## Quick Usage

```bash
/home/bobby/video-tools/real-video-enhancer/upscale_video.sh "{{filepath}}" "{{output_path}}" "{{mode}}" "{{preset}}" "{{engine}}" "{{job_id}}"
```

## Parameters

| Parameter | Default | Description |
|-----------|---------|-------------|
| filepath | required | Input video path |
| mode | auto | `anime` or `real` (auto-detects) |
| preset | fast | `fast` (2x) or `high` (4x) |
| engine | auto | `waifu2x` or `realesrgan` |
| job_id | auto | `tg_<chatid>_<messageid>` |

## Mode Selection

- **anime** → Waifu2x (better for anime with text)
- **real** → Real-ESRGAN (better for photorealistic)

## Preset Selection

| Preset | Upscale | CRF | Speed |
|--------|---------|-----|-------|
| fast | 2x | 20 | Quick |
| high | 4x | 16 | Slower |

## Example Prompts

- "Upscale this"
- "Upscale this anime clip"  
- "Make this 4K"
- "Enhance in high quality, keep me updated"

## Output Format

```
JOB_ID: tg_123_456
PHASE: EXTRACTING_FRAMES
FRAMES_TOTAL: 780
PHASE: UPSCALING
PROGRESS: 78/780
...
STATUS: OK
UPSCALED_VIDEO: /path/to/output.mp4
```

## Installation

See [INSTALL.md](references/INSTALL.md) for setup instructions.
