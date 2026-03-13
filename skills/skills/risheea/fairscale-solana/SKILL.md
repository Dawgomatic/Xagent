---
name: fairscale-solana
description: Solana wallet reputation. Ask anything in plain English — "is this a bot?", "whale?", "diamond hands?" — get instant answers.
license: MIT
metadata:
  author: FairScale
  version: "3.4.0"
---

# FairScale — Reputation Intelligence for Solana

**Docs:** https://docs.fairscale.xyz

---

## TWO ACCESS METHODS

### Method 1: API Key (Recommended)

Use if you have a FairScale API key.

```
GET https://api.fairscale.xyz/score?wallet=WALLET_ADDRESS
Header: fairkey: YOUR_API_KEY
```

**Example:**
```bash
curl "https://api.fairscale.xyz/score?wallet=GFTVQdZumAnBRbmaRgN9n3Z5qH5nXvjMZXJ3EyqP32Tn" \
  -H "fairkey: YOUR_API_KEY"
```

Get your key: https://sales.fairscale.xyz

---

### Method 2: x402 Micropayments (For Agents with Wallets)

No API key needed. Pay $0.05 USDC per call.

```
GET https://x402.fairscale.xyz/score?wallet=WALLET_ADDRESS
```

**Flow:**
1. Call endpoint → Get 402 response with payment details
2. Send 0.05 USDC to treasury wallet shown
3. Retry with header: `x-payment-signature: YOUR_TX_SIGNATURE`
4. Get wallet data

**Pricing:**
| Request | Price |
|---------|-------|
| Single wallet | $0.05 USDC |
| Batch (10 wallets) | $0.40 USDC |

---

## WHICH METHOD TO USE

| Situation | Use |
|-----------|-----|
| Have API key configured | Method 1 (API Key) |
| Agent has Solana wallet with USDC | Method 2 (x402) |
| No API key, no wallet | Cannot access — need one or the other |

**Check for API key first.** If `$FAIRSCALE_API_KEY` is set, use Method 1. Otherwise, try Method 2.

---

## RESPONSE FIELDS

| Field | Use |
|-------|-----|
| `fairscore` | The score (0-100) — **USE THIS** |
| `tier` | bronze / silver / gold / platinum |
| `badges` | Array — list each `badge.label` |
| `features` | Object — raw metrics for custom logic |

---

## QUICK RESPONSE FORMAT

For simple "check this wallet" requests:

```
 FairScore: [fairscore]/100 | Tier: [tier]

[ TRUSTED |  MODERATE |  CAUTION |  HIGH RISK]

 Badges: [badge labels]
```

**Risk thresholds:**
- ≥60 →  TRUSTED
- 40-59 →  MODERATE  
- 20-39 →  CAUTION
- <20 →  HIGH RISK

---

## NATURAL LANGUAGE → FEATURES

When users ask in plain English, translate to the right features:

| User asks | Check these | Logic |
|-----------|-------------|-------|
| "trustworthy?" | `fairscore` | ≥60 = yes |
| "whale?" / "deep pockets?" | `lst_percentile_score`, `stable_percentile_score`, `native_sol_percentile` | All >70 = whale |
| "bot?" / "sybil?" | `burst_ratio`, `platform_diversity` | burst >50 OR diversity <20 = bot |
| "diamond hands?" | `conviction_ratio`, `no_instant_dumps` | conviction >60 = yes |
| "active user?" | `active_days`, `tx_count`, `platform_diversity` | All >40 = active |
| "OG?" / "veteran?" | `wallet_age_score` | >70 = OG |
| "airdrop eligible?" | `wallet_age_score >50`, `platform_diversity >30`, `burst_ratio <30` | All must pass |
| "creditworthy?" | `conviction_ratio`, `no_instant_dumps`, `wallet_age_score` | All >50 = yes |

---

## RESPONSE EXAMPLES

**"Is this a whale?"**
```
 Whale Check: GFTVQd...P32Tn

 LST Holdings: 97.7% — Top 3% 
 Stablecoins: 27.5% — Low
◎ Native SOL: 45.2% — Moderate

Verdict:  PARTIAL WHALE — Heavy DeFi, not cash-rich.
```

**"Is this a bot?"**
```
 Bot Check: GFTVQd...P32Tn

 Burst Ratio: 16.8% — Organic 
 Platforms: 96.6% — Diverse 

Verdict:  HUMAN — Not a bot.
```

**"Airdrop eligible?"**
```
 Airdrop Check: GFTVQd...P32Tn

 Age: 79.2% 
 Diversity: 96.6% 
 Burst: 16.8% 

Verdict:  ELIGIBLE
```

---

## CUSTOM CRITERIA

When users define their own rules:

> "Only wallets with conviction > 70"

```
 Custom Check: GFTVQd...P32Tn

• Conviction > 70%:  69.7%

Verdict:  FAILS
```

---

## ALL FEATURES

| Feature | Description |
|---------|-------------|
| `fairscore` | Overall score (0-100) |
| `tier` | bronze/silver/gold/platinum |
| `wallet_age_score` | Account age percentile |
| `tx_count` | Transaction count percentile |
| `active_days` | Active days percentile |
| `platform_diversity` | DeFi protocols used |
| `conviction_ratio` | Holding behavior |
| `burst_ratio` | Bot-like patterns (high = bad) |
| `no_instant_dumps` | Doesn't quick-sell |
| `lst_percentile_score` | LST holdings |
| `stable_percentile_score` | Stablecoin holdings |
| `native_sol_percentile` | SOL balance |
| `net_sol_flow_30d` | Accumulating or draining |

---

## CRITICAL RULES

1. **ALWAYS call the API** — Never guess, never reuse old data
2. **Check for API key first** — Use Method 1 if `$FAIRSCALE_API_KEY` is set
3. **Fall back to x402** — If no API key, use Method 2 (if agent has wallet)
4. **Use exact endpoint** — `/score?wallet=ADDRESS`
5. **Translate user intent** — Map plain English to features
6. **Give verdicts** — Users want yes/no answers, not data dumps
7. **If API fails** — Say " Couldn't fetch data. Try again."

**Never invent data. Never guess. Always call the API.**

---

## LINKS

- Docs: https://docs.fairscale.xyz
- API Key: https://sales.fairscale.xyz
- Twitter: @FairScaleXYZ
