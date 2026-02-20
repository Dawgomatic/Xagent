---
name: fund-news-summary
description: Automatically collect and summarize the latest core news for specified funds.
---

# Fund News Summary Skill

## Execution

When this skill is triggered, the Agent should:

1. **Run the script directly**: `python3 /root/clawd/skills/fund-news-summary/fund_news.py`
2. **Read output**: The script automatically generates a report and outputs to stdout
3. **Send to Telegram**: Send the script output directly to Telegram

## Script Features

- ✅ **Rate limiting**: Maximum 2 concurrent searches, 1.5 second request interval
- ✅ **Retry mechanism**: Automatic retry on rate limit
- ✅ **Error handling**: Individual fund failures don't affect others
- ✅ **Formatted output**: Bold list format as required

## Manual Execution (Debug)

```bash
cd /root/clawd/skills/fund-news-summary
python3 fund_news.py
```

## Fund Configuration

The script has built-in keywords for the following funds:
- 华宝纳斯达克精选股票 (QDII)C
- 摩根欧洲动力策略股票 (QDII)A
- 摩根日本精选股票 (QDI)A
- 易方达黄金 ETF 联接 C
- 标普500 (S&P 500 Index)

## Auto Trigger

This skill is bound to the Cron job DailyFundNews, executing automatically at 11:00 Beijing Time daily.

## Obsidian Sync

Generated reports are automatically saved to Obsidian vault:
- **Save path**: `/root/clawd/obsidian-vault/reports/fund/YYYY-MM-DD.md`
- **Filename format**: `YYYY-MM-DD.md`
- **Sync method**: Bidirectional sync to your Obsidian vault via GitHub
