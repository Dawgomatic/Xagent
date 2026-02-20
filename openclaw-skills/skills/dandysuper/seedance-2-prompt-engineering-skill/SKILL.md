# Seedance 2.0 JiMeng Skill (OpenClaw / ClawHub)

## Purpose
Create high-control English prompts for **Seedance 2.0** and **Seedance 2.0 Fast** using multimodal references (image/video/audio/text).

This skill is for:
- Prompt design from rough idea to production-ready prompt
- Mode choice: **First/Last Frame** vs **All-Reference**
- `@asset` mapping (what each image/video/audio controls)
- 4-15s duration planning and timeline beats
- Video extension / continuation prompts
- Character replacement and directed editing prompts
- Camera-language replication from reference videos

---

## Core Rules

1. Always declare mode first.
2. Always include an explicit **Assets Mapping** section.
3. Use timecoded beats with one major action per segment.
4. Keep prompts concise and controllable (avoid vague poetic-only wording).
5. Add negative constraints when user needs clean output.

---

## Platform Limits (Seedance 2.0)

- Mixed inputs total (image+video+audio): **max 12 files**
- Images: jpeg/png/webp/bmp/tiff/gif, **max 9**, each < 30MB
- Videos: mp4/mov, **max 3**, total duration 2-15s, total < 50MB
- Audio: mp3/wav, **max 3**, total <= 15s, total < 15MB
- Generation duration: **4-15s**
- Realistic human face references may be blocked by platform compliance

---

## Output Format (use by default)

1. **Mode**
2. **Assets Mapping**
3. **Final Prompt**
4. **Negative Constraints**
5. **Generation Settings**

Example skeleton:

```text
Mode: All-Reference
Assets Mapping:
- @image1: first frame / identity anchor
- @video1: camera language + motion rhythm
- @audio1: optional soundtrack pacing

Final Prompt:
[ratio], [duration], [style].
0-3s: [action + camera].
3-7s: [action + transition].
7-10s: [reveal/climax + end frame].
Preserve identity and scene continuity. Use physically plausible motion and coherent lighting.

Negative Constraints:
no watermark, no logo, no subtitles, no on-screen text.

Generation Settings:
Duration: 10s
Aspect Ratio: 9:16
```

---

## Special Cases

### A) Extend Video
Explicitly write: `Extend @video1 by Xs`.
Use generation duration equal to the **newly added segment**, not the full final length.

### B) Replace Character
Bind base motion/camera to `@video1`, bind replacement identity to `@image1`, and request strict choreography/timing preservation.

### C) Beat Sync
Use `@video`/`@audio` rhythm references and lock beats by time range.

---

## Files in this skill

- `SKILL.md` — main skill behavior
- `SKILL.sh` — quick local test helper
- `scripts/setup_seedance_prompt_workspace.sh` — scaffold helper files
- `references/recipes.md` — ready-to-use prompt recipes
- `references/modes-and-recipes.md` — mode and control notes
