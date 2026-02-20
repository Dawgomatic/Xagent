# Modes and Interaction Notes (JiMeng Seedance 2.0)

## Mode Selection

### 1) First/Last Frame Mode
Use when user provides only first-frame image (or first+last frame) plus text prompt.

Recommended phrasing:
- `Use @image1 as first frame.`
- `End near @image2 composition.`

### 2) All-Reference Mode
Use for multimodal control with images/videos/audio/text.

Recommended phrasing:
- `@image1 for character design consistency`
- `@video1 for camera language and transition rhythm`
- `@audio1 for pacing and atmosphere`

## @Asset Invocation Pattern

Always map assets before prompt body to reduce mistakes:

```text
Assets Mapping:
- @image1 = first frame
- @image2 = environment style
- @video1 = motion + camera reference
- @audio1 = music rhythm
```

## Control Strengths to Exploit

Seedance 2.0 is strong at:
- Motion/camera replication from reference video
- Character and style consistency via image references
- Smooth extension/continuation of existing clips
- Directed editing of existing videos (replace/add/remove elements)
- Music-beat synchronization and emotional timing

## Common Pitfalls

- Overloading too many files without explicit roles
- Missing continuity instruction for identity/wardrobe/props
- Confusing full target duration vs extension length
- Asking for realistic face references that policy may block

## Quick Validation Checklist

- [ ] File counts and size limits respected
- [ ] Total mixed files <= 12
- [ ] Duration between 4 and 15 seconds
- [ ] Every reference has explicit `@asset` role
- [ ] Prompt has clear timeline beats
- [ ] Negative constraints included when required
