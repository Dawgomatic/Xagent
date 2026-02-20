# Memory Setup — Image Generation

## Initial Setup

Create directory on first use:
```bash
mkdir -p ~/image-generation
touch ~/image-generation/memory.md
```

## memory.md Template

Copy to `~/image-generation/memory.md`:

```markdown
# Image Generation Memory

## Provider
<!-- Preferred provider. Format: "provider: status" -->
<!-- Examples: midjourney: active subscription, dall-e: api configured -->

## Projects
<!-- What they're creating. Format: "project: description" -->
<!-- Examples: product shots: e-commerce catalog, book covers: fantasy series -->

## Preferences
<!-- Settings that work. Format: "setting: value" -->
<!-- Examples: style: cinematic lighting, resolution: 1024x1024 draft -->

---
*Last updated: YYYY-MM-DD*
```

## Optional: history.md

For users who want to track past generations:

```markdown
# Generation History

## Recent
<!-- Last 10 generations. Format: "date | prompt | provider | result" -->

## Successful Prompts
<!-- Prompts that worked well for reuse -->
```
