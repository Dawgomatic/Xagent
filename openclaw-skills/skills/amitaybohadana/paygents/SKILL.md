---
name: paygents
description: AI agent payment skill — generate wallet deeplinks (MetaMask, Trust Wallet), verify transactions on-chain, generate receipts, check balances. No custody, no backend, no API keys. Human approves in their own wallet.
metadata:
  openclaw:
    requires:
      bins:
        - node
        - bash
---

# EVM Payment Deeplink Skill

Generate wallet deeplinks for EVM payments. The user taps the link, approves in MetaMask, and the agent verifies the transaction on-chain.

## Flow

1. Agent collects payment details (recipient, amount, chain, token).
2. Agent runs the link generator script → gets a MetaMask deeplink.
3. Agent sends the link to the user.
4. User taps → MetaMask opens with pre-filled transfer → user approves.
5. User confirms "sent" → agent verifies the tx on-chain.

## Wallet Detection

Before generating a link, the agent should know which wallet the user has. Ask once, remember forever.

**Supported wallets with native deeplinks:**

| Wallet | `--wallet` flag | Deeplink format |
|--------|----------------|-----------------|
| MetaMask | `metamask` (default) | `https://link.metamask.io/send/...` |
| Trust Wallet | `trust` | `https://link.trustwallet.com/send?...` |

**Not supported (no send deeplinks):**
- Rabby — only has in-app browser, no direct send deeplink
- Coinbase Wallet — only has dapp browser deeplink, no direct send
- Phantom — requires encrypted handshake, not a simple URL

If the user's wallet isn't supported, default to MetaMask (most common) or let them know.

Store the user's wallet preference so you don't ask again. The agent should note the preference in its local session context (e.g. memory file). The user can change or clear this at any time by telling the agent.

## Inputs Required

| Field | Required | Description |
|-------|----------|-------------|
| `--to` | Yes | Recipient address (`0x...`) |
| `--amount` | Yes | Human-readable amount (e.g. `1.5`) |
| `--chain-id` | No | Chain ID (default: `8453` Base) |
| `--asset` | No | `ETH` or `ERC20` (default: `ERC20`) |
| `--token` | No | ERC20 contract address (auto-detected for USDC on known chains) |
| `--decimals` | No | Token decimals (default: `6` for USDC, `18` for ETH) |
| `--symbol` | No | Token symbol for display (default: `USDC` or `ETH`) |
| `--wallet` | No | `metamask` or `trust` (default: `metamask`) |

## Commands

### Generate Payment Link

**ERC20 (USDC) — MetaMask:**
```bash
skills/evm-usdc-wallet-poc/scripts/evm-payment-link.sh \
  --to 0x1234...5678 \
  --amount 10 \
  --chain-id 8453
```

**Native ETH — Trust Wallet:**
```bash
skills/evm-usdc-wallet-poc/scripts/evm-payment-link.sh \
  --to 0x1234...5678 \
  --amount 0.01 \
  --asset ETH \
  --chain-id 11155111 \
  --wallet trust
```

Output is JSON with:
- `intent` — structured payment details
- `deeplink` — MetaMask deeplink URL
- `messageTemplate` — ready-to-send message for the user

### Verify Transaction

After the user says "sent", verify on-chain:
```bash
skills/evm-usdc-wallet-poc/scripts/evm-verify-tx.sh \
  --chain-id 11155111 \
  --from 0xSENDER \
  --to 0xRECIPIENT \
  --asset ETH \
  --amount 0.001 \
  --blocks 50
```

Returns the matching tx hash if found, or "not found".

## Supported Chains

| Chain | ID | Default USDC |
|-------|----|-------------|
| Ethereum | 1 | `0xA0b86991c6218b36c1d19d4a2e9eb0ce3606eb48` |
| Base | 8453 | `0x833589fCD6eDb6E08f4c7C32D4f71b54bDa02913` |
| Sepolia | 11155111 | `0x1c7d4b196cb0c7b01d743fbc6116a902379c7238` |
| Base Sepolia | 84532 | `0x036CbD53842c5426634e7929541eC2318f3dCf7e` |

## User Message Pattern

When sending the link, always include:
1. Amount + token + chain
2. Recipient (truncated)
3. "Tap to open MetaMask and approve"
4. "Reject if recipient or amount doesn't match"

### Check Wallet Balance

Query native + major ERC20 balances across all supported chains:

```bash
# All chains at once
scripts/evm-balance.sh --address 0x1234...5678

# Single chain
scripts/evm-balance.sh --address 0x1234...5678 --chain-id 8453
```

Returns JSON with native balance + USDC, USDT, WETH, WBTC, DAI per chain.

Supported chains: Ethereum, Base, Polygon, Arbitrum, Optimism, BNB Chain, Sepolia, Base Sepolia.

No API key needed — uses public RPCs directly.

### Generate Receipt

After a transaction is verified, generate a structured receipt:

```bash
skills/evm-usdc-wallet-poc/scripts/evm-receipt.sh \
  --tx-hash 0xabc123... \
  --chain-id 8453 \
  --memo "order-42" \
  --merchant "Cool Store"
```

Options:
- `--format json | markdown | both` (default: `both`)
- `--out <directory>` — save receipt files to disk (JSON + markdown)
- `--memo` — order ID or note
- `--merchant` — merchant/payee name

Output includes: status, amount, token, from/to, gas fee, block, explorer link, timestamp.

The receipt can be sent to the user as a confirmation message, saved for bookkeeping, or forwarded to a merchant.

## RPC Configuration

By default, scripts use public RPC endpoints. You can override them for privacy or reliability:

**Option 1 — Environment variables** (highest priority):
```bash
export RPC_1="https://my-private-eth-node.com"
export RPC_8453="https://my-base-rpc.com"
```

**Option 2 — Config file** (copy `config.example.json` → `config.json`):
```json
{
  "rpc": {
    "1": "https://my-private-eth-node.com",
    "8453": "https://my-base-rpc.com"
  }
}
```

**Option 3 — Public fallbacks** (default, no setup needed):
Used automatically if no env var or config is set. Public providers (e.g. `eth.llamarpc.com`) will see your wallet addresses and tx hashes.

Resolution order: env var `RPC_<chainId>` → `config.json` → public fallback.

## Security & Privacy

- The wallet is the trust boundary — agent cannot force-execute.
- Verification checks the actual on-chain receipt, not user claims.
- No backend, no policy enforcement. This is a POC skill.
- Never store or handle private keys.
- **RPC privacy:** If using public fallback RPCs, third-party providers will see wallet addresses and tx hashes you query. Set your own RPCs via env vars or config.json for privacy.
- **Wallet preference:** The agent may store the user's preferred wallet (metamask/trust) in its memory. This is only the wallet app name, no keys or sensitive data. The user can ask the agent to clear it at any time.
