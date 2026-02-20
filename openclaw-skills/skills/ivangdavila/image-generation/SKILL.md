---
name: Image Generation
slug: image-generation
version: 1.0.2
homepage: https://clawic.com/skills/image-generation
description: Create AI images with prompt engineering, style control, and provider guides for Midjourney, DALL-E, Stable Diffusion, Flux, and Leonardo.
changelog: Added detailed provider endpoints documentation and feedback section
metadata: {"clawdbot":{"emoji":"🎨","requires":{"bins":[]},"os":["linux","darwin","win32"]}}
---

## When to Use

User needs AI-generated images. Agent handles text-to-image, image editing, style transfer, upscaling, and provider selection.

## Architecture

User preferences persist in `~/image-generation/`. See `memory-template.md` for setup.

```
~/image-generation/
├── memory.md      # Current provider, style, projects
└── history.md     # Past generations (optional)
```

## Quick Reference

| Topic | File |
|-------|------|
| Memory setup | `memory-template.md` |
| Prompt techniques | `prompting.md` |
| API handling | `api-patterns.md` |
| OpenAI/DALL-E | `openai.md` |
| Midjourney | `midjourney.md` |
| Stable Diffusion | `stable-diffusion.md` |
| Flux | `flux.md` |
| Leonardo | `leonardo.md` |
| Ideogram | `ideogram.md` |
| Replicate | `replicate.md` |

## Core Rules

### 1. Check Memory First
Read `~/image-generation/memory.md` for user's provider, preferred styles, and project context.

### 2. Draft Before Final
- Start at 512x512 or 1024x1024 to validate prompt
- Generate 4+ variations
- Only upscale the winner

### 3. Provider Selection by Task

| Task | Best Provider |
|------|---------------|
| Photorealism | Midjourney, Flux Pro |
| Text in images | Ideogram, DALL-E 3 |
| Fast iteration | Flux Schnell, Leonardo |
| Maximum control | Stable Diffusion |
| Inpainting/editing | DALL-E 3, Stable Diffusion |
| Budget API | Replicate, Leonardo |

### 4. Prompt Structure
- Subject first: "A red fox" not "In the forest there is a red fox"
- Style keywords: "cinematic lighting", "oil painting", "studio photography"
- Be specific: "golden hour sunlight" not "good lighting"
- Match aspect ratio to content: 1:1 portraits, 16:9 landscapes

### 5. Update Memory
| Event | Action |
|-------|--------|
| User chooses provider | Save to memory.md |
| Style works well | Note in memory.md |
| New project started | Add to memory.md |

## Common Traps

- **Hands/fingers wrong** → regenerate or use inpainting
- **Text garbled** → use Ideogram or add text in post-production
- **Faces distorted** → add "detailed face" to prompt, use face-fix models
- **Style inconsistent** → lock seed, use reference images
- **Watermarks appearing** → check model training, use clean models

## Security & Privacy

**Data that leaves your machine:**
- Prompts sent to chosen AI provider for generation

**Data that stays local:**
- Provider preferences in `~/image-generation/`
- No telemetry or analytics

**This skill does NOT:**
- Store generated images (provider handles storage)
- Access files outside `~/image-generation/`

## External Endpoints

| Provider | Endpoint | Data Sent | Purpose |
|----------|----------|-----------|---------|
| OpenAI | api.openai.com | Prompt text | DALL-E generation |
| Midjourney | discord.com | Prompt text | Image generation |
| Stability AI | api.stability.ai | Prompt text | Stable Diffusion |
| Replicate | api.replicate.com | Prompt text | Flux, SD models |
| Leonardo | cloud.leonardo.ai | Prompt text | Leonardo generation |
| Ideogram | api.ideogram.ai | Prompt text | Text-in-image |

Endpoints depend on chosen provider. No other data is sent externally.

## Trust

By using this skill, prompts are sent to third-party AI providers (OpenAI, Midjourney, Stability AI, etc.).
Only install if you trust these services with your prompts.

## Feedback

- If useful: `clawhub star image-generation`
- Stay updated: `clawhub sync`
