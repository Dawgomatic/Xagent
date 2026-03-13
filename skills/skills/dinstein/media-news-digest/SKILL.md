---
name: media-news-digest
description: Generate media & entertainment industry news digests. Covers Hollywood trades (THR, Deadline, Variety), box office, streaming, awards season, film festivals, and production news. Four-layer data collection from RSS feeds, Twitter/X KOLs, Reddit, and web search. Pipeline-based scripts with retry mechanisms and deduplication. Supports Discord, email, and markdown templates.
version: "1.7.1"
homepage: https://github.com/draco-agent/media-news-digest
source: https://github.com/draco-agent/media-news-digest
metadata:
  openclaw:
    requires:
      bins: ["python3"]
    optionalBins: ["gog"]
    credentialAccess: >
      This skill does NOT read, store, or manage any platform credentials itself.
      Email delivery uses the external `gog` CLI (Google Workspace CLI) which manages
      its own OAuth tokens separately. Twitter and Brave API keys are passed via
      environment variables and used only for outbound API calls within fetch scripts.
      No credentials are written to disk by this skill.
env:
  - name: X_BEARER_TOKEN
    required: false
    description: Twitter/X API bearer token for KOL monitoring
  - name: BRAVE_API_KEY
    required: false
    description: Brave Search API key for web search layer
---

# Media News Digest

Automated media & entertainment industry news digest system. Covers Hollywood trades, box office, streaming platforms, awards season, film festivals, production news, and industry deals.

## Quick Start

1. **Generate Digest** (unified pipeline — runs all 4 layers in parallel):
   ```bash
   python3 scripts/run-pipeline.py \
     --defaults <SKILL_DIR>/config/defaults \
     --hours 48 --freshness pd \
     --output /tmp/md-merged.json --verbose --force
   ```

2. **Use Templates**: Apply Discord or email templates to merged output

## Data Sources (44 total, 35 enabled)

- **RSS Feeds (15)**: THR, Deadline, Variety, Screen Daily, IndieWire, The Wrap, Collider, Vulture, Awards Daily, Gold Derby, Screen Rant, Empire, The Playlist, Entertainment Weekly, /Film
- **Twitter/X KOLs (13)**: @THR, @DEADLINE, @Variety, @FilmUpdates, @DiscussingFilm, @ScottFeinberg, @kristapley, @BoxOfficeMojo, @GiteshPandya, @MattBelloni, @Borys_Kit, and more

## Topics (7 sections)

-  Box Office — NA/global box office, opening weekends
-  Streaming — Netflix, Disney+, Apple TV+, HBO, viewership data
-  Production — New projects, casting, filming updates
-  Awards — Oscars, Golden Globes, Emmys, BAFTAs, campaigns
-  Deals & Business — M&A, rights, talent deals, restructuring
-  Film Festivals — Cannes, Venice, TIFF, Sundance, Berlin
-  Reviews & Buzz — Critical reception, RT/Metacritic scores

## Scripts Pipeline

Scripts are shared with tech-news-digest architecture:

1. `fetch-rss.py` — RSS feed fetcher with retry & parallel fetching
2. `fetch-twitter.py` — Twitter/X KOL monitor (requires `$X_BEARER_TOKEN`)
3. `fetch-web.py` — Web search via Brave API or agent fallback
4. `merge-sources.py` — Quality scoring & deduplication
5. `validate-config.py` — Configuration validator

## Cron Integration

Reference `references/digest-prompt.md` in cron prompts. See digest-prompt.md for placeholder documentation.

### Daily Digest
```
读取 <SKILL_DIR>/references/digest-prompt.md，按照其中的完整流程生成日报。
- MODE = daily, FRESHNESS = pd, RSS_HOURS = 48
- DISCORD_CHANNEL_ID = <channel_id>
- EMAIL = <email>
- LANGUAGE = Chinese
```

### Weekly Digest
```
读取 <SKILL_DIR>/references/digest-prompt.md，按照其中的完整流程生成周报。
- MODE = weekly, FRESHNESS = pw, RSS_HOURS = 168
- DISCORD_CHANNEL_ID = <channel_id>
- EMAIL = <email>
- LANGUAGE = Chinese
```

## Dependencies

```bash
pip install -r requirements.txt
```

All scripts work with Python 3.8+ standard library only. `feedparser` optional but recommended.
