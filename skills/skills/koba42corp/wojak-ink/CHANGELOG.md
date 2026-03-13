# Changelog

## v2.0.0 - Advanced Analytics & Rarity Features (2026-01-29)

###  Major Features Added

#### 1. Rarity Analysis System (`lib/rarity.js`)
- **Rarity scoring** based on trait frequency
- **Tier classification** (Common → Legendary)
- **Rank estimation** within collection
- **Character-type aware** scoring
- Rarest trait identification
- NFT comparison tools

**New Commands:**
```bash
wojak rarity <id>           # Estimate rarity score & tier
```

#### 2. Price History Tracking (`lib/history.js`)
- **Sales history** storage (local JSON)
- **Price trend detection** (rising/falling/stable)
- **Statistical analysis** (min/max/avg/change%)
- **Volume tracking** by time period
- **Top sales** leaderboard
- Automated data persistence

**New Commands:**
```bash
wojak history recent        # Last 10 sales
wojak history trend [hours] # Trend detection
wojak history stats [hours] # Price statistics
wojak track [character]     # Record current floor
```

**Data Storage:**
- `data/price_history.json` - Floor price snapshots
- `data/sales.json` - Individual sale records
- Automatic cleanup (keeps 30 days by default)

#### 3. Trait Analysis (`lib/traits.js`)
- **Trait extraction** from NFT metadata
- **Distribution analysis** by trait type
- **Rarity calculations** per trait
- **Combination detection** (rare trait combos)
- **Naked floor** finder (cheapest per trait)
- NFT trait comparison

**New Commands:**
```bash
wojak traits                # List trait categories
wojak traits <category>     # View distribution
```

**Trait Categories:**
- Base, Face, Face Wear, Mouth
- Head, Clothes, Background

#### 4. Deal Finder
- **Smart pricing** analysis
- **Discount detection** (% below average)
- **Auto-sorting** by best deals
- **Savings calculation**

**New Commands:**
```bash
wojak deals [threshold]     # Find underpriced NFTs
                           # Default: 10% below average
```

###  Enhanced Analytics

**Rarity Tiers:**
-  Legendary (score ≥ 10)
-  Epic (score ≥ 7)
-  Rare (score ≥ 5)
-  Uncommon (score ≥ 3)
-  Common (score < 3)

**Price Trends:**
-  Rising (positive slope)
-  Falling (negative slope)
-  Stable (flat)
- Confidence scoring

**Market Stats:**
- Current floor price
- Price change % (24h, 7d, custom)
- Volume analysis
- Sales count
- Min/max/average prices

###  Technical Improvements

**New Dependencies:**
- File system operations for data persistence
- JSON storage with automatic backups
- Modular architecture (4 new libraries)

**Architecture:**
```
wojak-ink/
├── lib/
│   ├── api.js          # Existing API client
│   ├── format.js       # Existing formatting
│   ├── rarity.js       #  NEW: Rarity analysis
│   ├── history.js      #  NEW: Price tracking
│   └── traits.js       #  NEW: Trait analysis
├── data/               #  NEW: Local storage
│   ├── price_history.json
│   └── sales.json
```

**Code Quality:**
- Full JSDoc comments
- Error handling
- Data validation
- Automatic caching

###  Documentation Updates

- **SKILL.md** - Full command reference
- **README.md** - Quick start guide
- **CHANGELOG.md** - This file
- Help text with all new commands

###  Testing

All new commands tested and working:
-  `wojak rarity 1` - Rarity estimation
-  `wojak rarity 4001` - Legendary tier detection
-  `wojak traits` - Trait categories list
-  `wojak history recent` - Sales history
-  `wojak deals` - Deal finder

###  Usage Examples

**Find rare NFTs:**
```bash
wojak rarity 2501           # Papa Tang rarity
wojak rarity 4001           # Alien Waifu (Legendary)
```

**Track market trends:**
```bash
wojak track                 # Record current floor
wojak history trend 24      # 24h trend
wojak history stats 168     # 7-day stats
```

**Find deals:**
```bash
wojak deals                 # 10%+ off
wojak deals 20              # 20%+ off
```

**Analyze traits:**
```bash
wojak traits                # List categories
wojak traits Head           # Head trait distribution
```

###  Next Steps

**To fully unlock trait features:**
1. Scrape full collection metadata
2. Build trait database
3. Calculate accurate rarity scores
4. Enable trait-based filtering

**Possible integrations:**
- Connect to `mint-garden` skill for metadata
- Link to `dexie` skill for sales data
- Use `spacescan` for wallet tracking

###  Notes

- Rarity scores are **estimates** until full metadata is loaded
- Price tracking requires **periodic execution** (`wojak track`)
- Sales history builds **over time**
- Data stored in `~/clawd/skills/wojak-ink/data/`

---

## v1.0.0 - Initial Release (2026-01-29)

- Basic NFT browsing
- Floor price tracking
- Marketplace listings
- Character type filtering
- Collection statistics
- MintGarden & Dexie API integration
