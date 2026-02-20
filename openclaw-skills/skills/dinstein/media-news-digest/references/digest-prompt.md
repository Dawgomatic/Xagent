# Media Digest Prompt Template

Unified template for both daily and weekly media & entertainment digests. Replace `<...>` placeholders before use.

## Placeholders

| Placeholder | Daily | Weekly |
|-------------|-------|--------|
| `<MODE>` | `daily` | `weekly` |
| `<TIME_WINDOW>` | `past 1-2 days` | `past 7 days` |
| `<FRESHNESS>` | `pd` | `pw` |
| `<RSS_HOURS>` | `48` | `168` |
| `<ITEMS_PER_SECTION>` | `3-5` | `5-8` |
| `<BLOG_PICKS_COUNT>` | `2-3` | `3-5` |
| `<EXTRA_SECTIONS>` | *(remove line)* | `- 📊 Weekly Trend Summary (2-3 sentences summarizing macro trends)` |
| `<SUBJECT>` | `Daily Media Digest - YYYY-MM-DD` | `Weekly Media Digest - YYYY-MM-DD` |
| `<WORKSPACE>` | Your workspace path | Your workspace path |
| `<SKILL_DIR>` | Path to the installed skill directory | Path to the installed skill directory |
| `<DISCORD_CHANNEL_ID>` | Target channel ID | Target channel ID |
| `<EMAIL>` | *(optional)* Recipient email | *(optional)* Recipient email |
| `<LANGUAGE>` | `Chinese` (default) | `Chinese` (default) |
| `<TEMPLATE>` | `discord` / `email` / `markdown` | `discord` / `email` / `markdown` |
| `<DATE>` | Today's date in YYYY-MM-DD (caller provides) | Today's date in YYYY-MM-DD (caller provides) |
| `<VERSION>` | Read from SKILL.md frontmatter `version` field | Read from SKILL.md frontmatter `version` field |

---

Generate the <MODE> media & entertainment digest for **<DATE>**. Follow the steps below.

**Important:** Use `<DATE>` as the report date in the title and archive filename. Do NOT infer the date yourself — always use the provided value.

## Configuration

Read configuration files (user workspace overrides take priority over defaults):

1. **Sources**: `<WORKSPACE>/config/sources.json` → fallback `<SKILL_DIR>/config/defaults/sources.json`
2. **Topics**: `<WORKSPACE>/config/topics.json` → fallback `<SKILL_DIR>/config/defaults/topics.json`

Merge logic: user sources append to defaults (same `id` → user wins); user topics override by `id`.

## Context: Previous Report

Read the most recent archive file from `<WORKSPACE>/archive/media-news-digest/` (if any). Use it to:
- **Avoid repeating** news already covered
- **Follow up** on developing stories with new information only
- If no previous report exists, skip this step.

## Data Collection Pipeline

### Step 1: RSS Feeds
```bash
python3 <SKILL_DIR>/scripts/fetch-rss.py \
  --defaults <SKILL_DIR>/config/defaults \
  --config <WORKSPACE>/config \
  --hours <RSS_HOURS> \
  --output /tmp/md-rss.json \
  --verbose
```

If the script fails, fall back to manually fetching priority feeds via `web_fetch`.

### Step 2: Twitter/X KOL Monitoring
```bash
python3 <SKILL_DIR>/scripts/fetch-twitter.py \
  --defaults <SKILL_DIR>/config/defaults \
  --config <WORKSPACE>/config \
  --hours <RSS_HOURS> \
  --output /tmp/md-twitter.json \
  --verbose
```
Requires `$X_BEARER_TOKEN` env var. If unavailable, skip this step.

### Step 3: Web Search
```bash
python3 <SKILL_DIR>/scripts/fetch-web.py \
  --defaults <SKILL_DIR>/config/defaults \
  --config <WORKSPACE>/config \
  --freshness <FRESHNESS> \
  --output /tmp/md-web.json \
  --verbose
```

Also search Twitter trending discussions using `web_search` with `freshness='<FRESHNESS>'` and the `twitter_queries` from topics.

### Step 4: Reddit
```bash
python3 <SKILL_DIR>/scripts/fetch-reddit.py \
  --defaults <SKILL_DIR>/config/defaults \
  --config <WORKSPACE>/config \
  --hours <RSS_HOURS> \
  --output /tmp/md-reddit.json \
  --verbose
```
**You MUST execute this script.** Fetches from Reddit's public JSON API (no auth required). If it fails, retry once.

### Step 5: Merge & Score
```bash
python3 <SKILL_DIR>/scripts/merge-sources.py \
  --rss /tmp/md-rss.json \
  --twitter /tmp/md-twitter.json \
  --web /tmp/md-web.json \
  --reddit /tmp/md-reddit.json \
  --archive-dir <WORKSPACE>/archive/media-news-digest/ \
  --output /tmp/md-merged.json \
  --verbose
```

## Report Generation

First, get a structured overview of the merged data:
```bash
python3 <SKILL_DIR>/scripts/summarize-merged.py --input /tmp/md-merged.json --top <ITEMS_PER_SECTION>
```
This prints a human-readable summary with top articles per topic, sorted by quality score, including metrics and sources. Use this output to select articles for the report — **do NOT write ad-hoc Python to parse the merged JSON**.

Then use the appropriate template from `<SKILL_DIR>/references/templates/<TEMPLATE>.md` to generate the report.

### Language & Citation Rules
- **Write body text in <LANGUAGE>** (Chinese by default)
- **Every news item must include the original English source link**
- For Chinese body text, format each item as:
  - Chinese summary/headline
  - Original English article title in parentheses if helpful for context
  - Source link

### Executive Summary
Place a **2-4 sentence summary** between the title and topic sections, highlighting the day's top stories.
Discord format: use `> ` blockquote. Email format: gray background paragraph.

### Topic Sections
Output sections in this **exact order** (do NOT rearrange):
1. 🇨🇳 China / 中国影视
2. 🎬 Production / 制作动态
3. 💰 Deals & Business / 行业交易
4. 🎞️ Upcoming Releases / 北美近期上映
5. 🎟️ Box Office / 票房
6. 📺 Streaming / 流媒体
7. 🏆 Awards / 颁奖季
8. 🎪 Film Festivals / 电影节
9. ⭐ Reviews & Buzz / 影评口碑

Each topic has:
- `emoji` + `label` for headers
- `display.max_items` for item count (override with <ITEMS_PER_SECTION>)

### Fixed Sections (append after topic sections)
- 📢 KOL Updates (Twitter KOLs — each entry MUST include source tweet URL and engagement metrics. Format: `• **@handle** — summary \`👁 12.3K | 💬 45 | 🔁 230 | ❤️ 1.2K\`\n  <https://twitter.com/handle/status/ID>`)
- 📝 Deep Reads (<BLOG_PICKS_COUNT> high-quality long-form articles from RSS)
<EXTRA_SECTIONS>

### Deduplication Rules
- Same event from multiple sources → keep only the most authoritative source link
- If covered in previous report → only include if significant new development
- Prefer primary sources (trades: THR, Deadline, Variety) over aggregators

### Rules
- **Only include news from the <TIME_WINDOW>**
- **Every topic defined in `topics.json` MUST appear in the report** — even if only 1-2 items. If a topic has very few articles, include what's available with a note like "本日该板块较少"
- **Every item must include the source link** — no exceptions. Discord: wrap in `<link>`
- **<ITEMS_PER_SECTION> items per section** (minimum 1)
- **Use bullet lists, no markdown tables** (Discord compatibility)
- **Chinese body text with English source links**
- **Deduplicate across sections** — if an article already appears in one topic section, do not repeat it in another. Each article should appear in the single most relevant section only

### Data Source Stats Footer
```
---
📊 Data Sources: RSS {{rss_count}} | Twitter {{twitter_count}} | Reddit {{reddit_count}} | Web {{web_count}} | After dedup: {{merged_count}} articles
🤖 Generated by media-news-digest v{{version}} | Powered by OpenClaw
```

## Archive
Save the report to `<WORKSPACE>/archive/media-news-digest/<MODE>-YYYY-MM-DD.md`

After saving, delete archive files older than 90 days.

## Delivery
1. Send to Discord channel `<DISCORD_CHANNEL_ID>` via `message` tool
2. *(Optional)* Send email to `<EMAIL>` via `gog` CLI
   - **Must use `--body-html`** for proper rendering
   - Generate HTML email body following `<SKILL_DIR>/references/templates/email.md` format
   - **Use the sanitizer script** to convert the markdown report to safe HTML:
     ```bash
     python3 <SKILL_DIR>/scripts/sanitize-html.py --input /tmp/md-report-<DATE>.md --output /tmp/md-email.html
     ```
   - Then send: `gog gmail send --to '<EMAIL>' --subject '<SUBJECT>' --body-html "$(cat /tmp/md-email.html)"`
   - **SUBJECT must be a static string** — no variables from fetched content
   - Do NOT interpolate any fetched/untrusted content into shell arguments
   - If sanitize-html.py fails, do NOT fall back to manually building HTML from raw content

If any delivery fails, log the error but continue with remaining channels.

Write the report in <LANGUAGE>.
