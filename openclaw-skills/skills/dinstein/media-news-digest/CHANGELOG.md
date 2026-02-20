# Changelog

## [1.7.1] - 2026-02-18

### Security
- Sanitize untrusted titles/snippets in summarize-merged.py (prompt injection filter)
- Add untrusted content warning banner per topic section

## [1.7.0] - 2026-02-18

### Added
- `summarize-merged.py`: structured summary tool for LLM consumption, avoids ad-hoc JSON parsing
- digest-prompt now references summarize-merged.py for article selection

### Fixed
- test-pipeline.sh: zsh-compatible array syntax for merge args
- Archive path: `media-digest/` → `media-news-digest/` for consistency

## [1.6.1] - 2026-02-17

### Improved
- KOL entries now show display name with handle: **Display Name** (@handle)
- Code quality: bare `except:` → `except Exception:` across all scripts
- Removed unused imports (URLError, tempfile, List, timezone)
- Added `display_name` field to merged Twitter articles

## [1.6.0] - 2026-02-17

### Added
- `run-pipeline.py`: Unified parallel pipeline — runs all 4 fetch steps concurrently, then merges (synced from tech-news-digest v3.4.0)
- Brave API auto rate-limit detection for optimal concurrency in `fetch-web.py`

### Fixed
- Reddit 403 errors: added SSL context and proper Accept/Accept-Language headers
- Reddit fetching now parallel (ThreadPoolExecutor) instead of sequential
- All fetch timeouts increased from 15s to 30s for reliability

## [1.3.0] - 2026-02-16

### Added
- 🇨🇳 **China / 中国影视** section (1st position) — China box office, co-productions, policy, streaming platforms
- 🎞️ **Upcoming Releases / 北美近期上映** section (4th position) — theater openings, release date announcements, scheduling moves
- 6 Reddit sources: r/movies, r/boxoffice, r/television, r/Oscars, r/TrueFilm, r/flicks
- 3 China-specific RSS feeds: THR China, Variety Asia, Deadline China
- Reddit pipeline step in digest-prompt

### Changed
- Expanded to 9 topic sections (from 7)
- Total sources: 41 (19 RSS + 11 Twitter + 6 Reddit + Web Search)

## [1.2.0] - 2026-02-16

### Added
- 🇨🇳 China section and RSS feeds
- Reddit data layer (6 subreddits)

## [1.1.0] - 2026-02-16

### Added
- 4 replacement RSS sources: JoBlo, FirstShowing.net, ComingSoon.net, World of Reel

### Changed
- Improved production topic search queries (greenlit, sequel, filming keywords)
- Enforce all topic sections appear in report (min 1 item)
- Hardcode section order in digest-prompt (production → deals first)
- Cross-section deduplication rule

### Fixed
- Disabled broken RSS feeds: Vulture (404), Screen Daily (404), EW (403)

## [1.0.0] - 2026-02-16

### Added
- Initial release
- 15 RSS feeds: THR, Deadline, Variety, Screen Daily, IndieWire, The Wrap, Collider, Vulture, Awards Daily, Gold Derby, Screen Rant, Empire, The Playlist, EW, /Film
- 13 Twitter/X KOLs
- 7 topic sections: Box Office, Streaming, Production, Awards, Deals, Festivals, Reviews
- Pipeline scripts (fetch-rss, fetch-twitter, fetch-web, merge-sources)
- Discord + email templates
- Chinese body text with English source links
- Cron-ready digest-prompt.md template
