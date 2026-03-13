#  Signal Generator for OpenClaw

Generate automated trading signals and send alerts to Discord/Telegram with just a few clicks.

##  Features

-  **2 Powerful Strategies** — BB Breakout, RSI Reversal
-  **Multi-Timeframe** — 15m, 1h, 4h, or any interval
-  **Easy Configuration** — No coding required, just edit JSON
-  **Real-Time Alerts** — Send to Discord, Telegram, or any channel
-  **Open Source** — Customize strategies as you want

##  Quick Start

### 1. Install

Copy the skill to your OpenClaw workspace:

```bash
cp -r signal-generator ~/.openclaw/workspace/skills/
```

### 2. Configure

```bash
cd ~/.openclaw/workspace/skills/signal-generator
cp config.json.example config.json
nano config.json  # or your favorite editor
```

Edit your settings:

```json
{
  "symbol": "BTC/USDT",
  "strategy": "bb_breakout",
  "intervals": ["15m", "1h"],
  "targets": [
    "discord:YOUR_CHANNEL_ID",
    "telegram:YOUR_CHAT_ID"
  ]
}
```

### 3. Run

```bash
python3 signal_generator.py
```

Or use the bot's Python environment:

```bash
/root/quant-trading-bot/venv/bin/python3 signal_generator.py
```

##  Output

The skill generates signals and saves them to `last_signal.json`:

```json
[
  {
    "strategy": "BB Breakout",
    "price": 77709.85,
    "long": false,
    "short": false,
    "squeeze": true,
    "bb_upper": 78390.25,
    "bb_lower": 75679.11,
    "rsi": 50,
    "interval": "1h",
    "timestamp": "2026-02-02T11:20:19"
  }
]
```

##  Strategies

### BB Breakout (Default)

**Logic:**
1. Detect BB Squeeze (Bollinger Bands inside Keltner Channels)
2. Wait for Breakout (Price closes outside BB)
3. Confirm with Volume Spike

**Long Signal:** Close > BB Upper + Volume > Average
**Short Signal:** Close < BB Lower + Volume > Average

### RSI Reversal

**Logic:**
1. RSI < 30 (Oversold) → Buy
2. RSI > 70 (Overbought) → Sell

**Long Signal:** RSI crosses below 30 then rises
**Short Signal:** RSI crosses above 70 then falls

##  Configuration

| Option | Description | Default |
|--------|-------------|---------|
| `symbol` | Trading pair (e.g., BTC/USDT) | BTC/USDT |
| `strategy` | `bb_breakout` or `rsi_reversal` | bb_breakout |
| `intervals` | Timeframes to check | ["15m", "1h"] |
| `targets` | Channel IDs for alerts | [] |

##  Usage Examples

### Run Manually

```bash
cd ~/.openclaw/workspace/skills/signal-generator
python3 signal_generator.py
```

### Schedule with Cron

Run every 5 minutes:

```bash
crontab -e
```

Add this line:

```bash
*/5 * * * * cd ~/.openclaw/workspace/skills/signal-generator && /root/quant-trading-bot/venv/bin/python3 signal_generator.py
```

### Send Alerts to Discord

1. Get your Discord channel ID
2. Add to config:
```json
"targets": ["discord:YOUR_CHANNEL_ID"]
```
3. Use OpenClaw's `message` tool to send alerts (or build a wrapper)

##  What's Included

-  `signal_generator.py` — Core signal engine
-  `config.json.example` — Configuration template
-  `SKILL.md` — Detailed documentation
-  `README.md` — This file

##  Notes

- Uses Binance public API (no keys required for OHLCV data)
- Requires Python 3.7+ with pandas, numpy, ccxt
- Designed for OpenClaw but can run standalone

##  License

This skill is provided as-is. Use at your own risk. Trading signals are not financial advice.

---

**Version:** 1.0.0
**Author:** Aether
**Platform:** OpenClaw
**Last Updated:** 2026-02-02
