# Video Upscale - Installation

## Required Tools

This skill requires:
- `ffmpeg` - Video processing
- `bc` - Math calculations
- `md5sum` - File hashing

## AI Upscaling Tools

You need to install either or both:

### Waifu2x (recommended for anime)
```bash
cd /home/bobby/video-tools
mkdir -p waifu2x-ncnn-vulkan
cd waifu2x-ncnn-vulkan
curl -L -o waifu2x.zip "https://github.com/nihui/waifu2x-ncnn-vulkan/releases/download/20220728/waifu2x-ncnn-vulkan-20220728-ubuntu.zip"
unzip waifu2x.zip
# Rename folder to: waifu2x-ncnn-vulkan-20220728-ubuntu
```

### Real-ESRGAN (better for real footage)
```bash
cd /home/bobby/video-tools
mkdir -p real-video-enhancer
cd real-video-enhancer
curl -L -o realesrgan.zip "https://github.com/xinntao/Real-ESRGAN/releases/download/v0.2.5.0/realesrgan-ncnn-vulkan-20220424-ubuntu.zip"
unzip realesrgan.zip
chmod +x realesrgan-ncnn-vulkan
```

## Directory Structure

Expected structure:
```
/home/bobby/video-tools/
├── real-video-enhancer/
│   ├── upscale_video.sh
│   ├── realesrgan-ncnn-vulkan
│   └── models/
└── waifu2x-ncnn-vulkan/
    └── waifu2x-ncnn-vulkan-20220728-ubuntu/
        ├── waifu2x-ncnn-vulkan
        └── models-cunet/
```

## Environment Variables (optional)

Override tool paths:
```bash
export VIDEO_UPSCALE_REALESRGAN="/path/to/real-video-enhancer"
export VIDEO_UPSCALE_WAIFU2X="/path/to/waifu2x-ncnn-vulkan"
```
