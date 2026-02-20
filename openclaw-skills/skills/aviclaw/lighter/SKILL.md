---
name: lighter
description: Interact with Lighter protocol - a ZK rollup orderbook DEX. Use when you need to trade on Lighter, check prices, manage positions, or query account data.
---

# Lighter Protocol

Trade on Lighter - a zero-knowledge rollup orderbook DEX with millisecond latency and zero fees for retail traders.

## Quick Start

```bash
# Set environment variables
export PRIVATE_KEY="your-private-key"
export LIGHTER_API_KEY="your-api-key"
export LIGHTER_ACCOUNT_INDEX="0"

# Check account
node scripts/account.js

# Get markets
node scripts/markets.js

# Place order
node scripts/order.js --market ETH-USDC --side buy --amount 0.1 --price 3000
```

## Getting Credentials

### PRIVATE_KEY

Your Ethereum wallet private key (with `0x` prefix).

**How to get:**
1. Open MetaMask → Account details
2. Click "Export private key"
3. Enter password
4. Copy the key (starts with `0x...`)

**⚠️ Never share this!** Store in environment variables, never in code.

### LIGHTER_API_KEY

API key for Lighter protocol access.

**How to get:**
1. Go to https://dashboard.zklighter.io
2. Connect your wallet
3. Go to API Keys section
4. Create new API key
5. Copy the key

**Note:** Some operations work without API key but rate-limited.

### LIGHTER_ACCOUNT_INDEX

Your account index on Lighter.

**How to get:**
1. Go to https://dashboard.zklighter.io
2. Connect your wallet
3. Your account index is shown in the dashboard (usually `0` for first account)
4. Or check via `node scripts/account.js` after setting PRIVATE_KEY

## What is Lighter?

Lighter is a fully verifiable decentralized exchange built with custom ZK infrastructure:
- **Zero fees** for retail traders
- **Millisecond latency** - tens of thousands of orders per second  
- **ZK proofs** of all operations
- **Ethereum security** as base layer
- **Backed by** a16z, Dragonfly, Coatue

## Features

- **Market Data**: List markets, get prices, order books
- **Trading**: Place/cancel orders, manage positions
- **Account**: Check balances, deposits, withdrawals
- **WebSocket**: Real-time market data stream

## Installation

```bash
cd ~/.openclaw/workspace/skills/lighter
npm install
```

## Usage

### Account Operations

```bash
# Check account balance
node scripts/account.js

# Deposit
node scripts/deposit.js --token USDC --amount 1000

# Withdraw  
node scripts/withdraw.js --token USDC --amount 1000 --to 0x...
```

### Market Data

```bash
# List all markets
node scripts/markets.js

# Get market data for ETH-USDC
node scripts/market.js --market ETH-USDC

# Get order book
node scripts/orderbook.js --market ETH-USDC

# Get recent trades
node scripts/trades.js --market ETH-USDC --limit 50
```

### Orders

```bash
# Place limit order
node scripts/order.js --market ETH-USDC --side buy --amount 0.1 --price 3000 --type limit

# Place market order
node scripts/order.js --market ETH-USDC --side buy --amount 0.1 --type market

# Cancel order
node scripts/cancel.js --order-id 12345

# List open orders
node scripts/orders.js
```

### Positions

```bash
# Get positions
node scripts/positions.js

# Get position history
node scripts/position-history.js --market ETH-USDC
```

## Architecture

```
lighter/
├── SKILL.md              # This file
├── package.json
├── scripts/
│   ├── account.js       # Account info
│   ├── markets.js       # List markets
│   ├── market.js        # Market details
│   ├── orderbook.js     # Order book
│   ├── order.js         # Place/cancel orders
│   ├── orders.js        # List open orders
│   ├── positions.js     # Positions
│   └── trades.js        # Recent trades
└── references/
    └── api.md           # API docs
```

## Environment Variables

| Variable | Description | Required | How to Get |
|----------|-------------|----------|------------|
| `PRIVATE_KEY` | Wallet private key (0x...) | Yes | MetaMask → Account → Export private key |
| `LIGHTER_API_KEY` | API key from Lighter | No | dashboard.zklighter.io → API Keys |
| `LIGHTER_ACCOUNT_INDEX` | Account index | No | dashboard.zklighter.io (usually 0) |

## Notes

- Mainnet RPC: https://mainnet.zklighter.elliot.ai
- Chain ID: 300
- Requires ETH on Lighter for gas fees
- Orders match by price-time priority
- All operations generate ZK proofs
- **Testnet:** Use chain ID 111 or testnet RPC for testing
