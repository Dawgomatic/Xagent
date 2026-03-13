# Safety First: Non-Custodial Trading Reality Check

>  **MANDATORY READING** — Read this entire file before proceeding with any trading or account management operations.

## Table of Contents

1. [The Non-Custodial Model](#the-non-custodial-model)
2. [What AI Can and Cannot Do](#what-ai-can-and-cannot-do)
3. [Risk Acknowledgment](#risk-acknowledgment)
4. [Testnet Before Mainnet](#testnet-before-mainnet)
5. [Emergency Procedures](#emergency-procedures)
6. [Security Checklist](#security-checklist)

---

## The Non-Custodial Model

### Understanding 01.xyz Architecture

01.xyz operates on a **non-custodial** model that differs fundamentally from centralized exchanges (CEX):

| Aspect | Centralized Exchange (CEX) | 01.xyz (Non-Custodial) |
|--------|---------------------------|------------------------|
| **Key Custody** | Exchange holds your keys | You hold your keys |
| **Signing** | Exchange signs transactions | You sign via local API |
| **Withdrawals** | Request withdrawal, wait for approval | Instant, you control funds |
| **Counterparty Risk** | High (exchange can freeze/lose funds) | Minimal (smart contract risk only) |
| **Responsibility** | Exchange responsible for security | You are responsible |

### What This Means for You

**You are fully responsible for:**
-  Private key security
-  Transaction verification
-  Local API configuration
-  All trading decisions
-  Understanding risks and liquidation mechanics

**You never have to trust:**
-  The exchange with your private keys
-  Third parties with fund custody
-  AI systems with spending authority

---

## What AI Can and Cannot Do

### What AI CAN Do

 **Read public market data** — Prices, orderbooks, funding rates (safe, no auth)  
 **Read your positions** — When you provide wallet address  
 **Calculate health metrics** — Margin fraction, liquidation prices  
 **Suggest strategies** — Based on market conditions and risk parameters  
 **Monitor and alert** — Funding payments, liquidations, price movements  
 **Generate code** — SDK integration, automation scripts  
 **Explain concepts** — Order types, margin mechanics, N1 protocol  

### What AI CANNOT Do

 **Access your private keys** — We never have them  
 **Sign transactions** — Only your local API can sign  
 **Move your funds** — Impossible without your keys  
 **Place orders without confirmation** — Safety guardrails prevent this  
 **Guarantee profits** — No financial advice, all trading is risky  
 **Prevent liquidation** — We can warn, but you must act  
 **Reverse transactions** — Blockchain transactions are final  

### The Human-in-the-Loop Requirement

Every trading action **requires explicit human confirmation**:

```
AI Suggestion: Buy 0.5 SOL at $145.00 limit

User Confirmation Required:
☐ Yes, execute this order
☐ No, cancel
☐ Modify parameters first
```

**Never automate trading without:**
1. Extensive paper trading
2. Circuit breakers
3. Stop-losses
4. Position size limits
5. Manual kill switch

---

## Risk Acknowledgment

### Perpetual Futures Are High-Risk

Trading perpetual futures involves **substantial risk of loss**:

| Risk Type | Description | Mitigation |
|-----------|-------------|------------|
| **Liquidation Risk** | Position forcibly closed if margin < maintenance | Monitor margin fraction, use stop-losses |
| **Funding Rate Risk** | Periodic payments can erode PnL | Watch funding rates, consider direction |
| **Smart Contract Risk** | Bugs in N1 protocol or 01 contracts | Use testnet, monitor audits |
| **Oracle Risk** | Price feed manipulation | 01 uses robust oracle network |
| **Volatility Risk** | Crypto markets are highly volatile | Position sizing, don't overleverage |
| **Technical Risk** | API failures, network issues | Redundancy, manual backup plans |

### Required Acknowledgment

Before proceeding to trading operations, you must acknowledge:

> **I understand that:**
> 1. Trading perpetual futures carries substantial risk of loss
> 2. I could lose part or all of my margin
> 3. Liquidation may result in total position loss
> 4. AI cannot prevent losses or guarantee profits
> 5. I am solely responsible for my trading decisions
> 6. I have read and understood 01.xyz documentation

**Type "I understand and accept these risks" to proceed.**

---

## Testnet Before Mainnet

### The Golden Rule

```
 NEVER deploy untested code to mainnet with real funds
 ALWAYS test thoroughly on devnet first
 ONLY proceed to mainnet after successful devnet validation
```

### Devnet vs Mainnet

| Feature | Devnet | Mainnet |
|---------|--------|---------|
| **URL** | `https://zo-devnet.n1.xyz` | `https://zo-mainnet.n1.xyz` |
| **Funds** | Fake/test tokens | Real SOL/USDC |
| **Purpose** | Development, testing | Live trading |
| **Risk** | Zero financial risk | Real financial risk |
| **Availability** | May be reset | Permanent |

### Testing Checklist

Before using mainnet:

☐ **SDK Integration** — Can connect to devnet  
☐ **Market Data** — Can fetch prices, orderbooks  
☐ **Account Query** — Can read account state  
☐ **Order Placement** — Can place and cancel orders  
☐ **Error Handling** — Graceful handling of failures  
☐ **Position Management** — Can monitor and adjust positions  
☐ **Emergency Flow** — Can close positions quickly  

### Testnet Workflow

```javascript
// Step 1: Connect to devnet
const nord = await Nord.new({
  app: 'zoau54n5U24GHNKqyoziVaVxgsiQYnPMx33fKmLLCT5',
  webServerUrl: 'https://zo-devnet.n1.xyz', // DEVNET!
  // ...
});

// Step 2: Test all operations
// - Get market data
// - Read account (use test wallet)
// - Place small orders
// - Cancel orders
// - Close positions

// Step 3: Only then switch to mainnet
// webServerUrl: 'https://zo-mainnet.n1.xyz'
```

---

## Emergency Procedures

### Scenario 1: Approaching Liquidation

**Symptoms:**
- Margin fraction dropping below 15%
- Liquidation price approaching current price
- Position showing high unrealized loss

**Immediate Actions:**
1. **Assess the situation** — How close to liquidation?
2. **Option A: Add margin** — Deposit more collateral
3. **Option B: Reduce position** — Close part of position
4. **Option C: Full exit** — Close entire position to preserve capital
5. **Never** — Wait and hope, add more leverage

### Scenario 2: API Unresponsive

**Symptoms:**
- Orders not going through
- Cannot read account data
- Connection timeouts

**Troubleshooting:**
1. Check local API status: `curl http://localhost:3000/health`
2. Check network connectivity
3. Restart local API service
4. Check N1 network status
5. Have backup plan for manual trading via UI

### Scenario 3: Unexpected Order Fills

**Symptoms:**
- Orders filling at unexpected prices
- Duplicate orders executed
- Orders you don't remember placing

**Investigation:**
1. Check order history
2. Look for stale orders from previous sessions
3. Verify no other bots/scripts running
4. Check for API key compromise
5. Cancel all pending orders immediately

### Emergency Contacts

- **01.xyz Support**: https://01.xyz/support
- **N1 Discord**: Community support channel
- **Status Page**: Check N1 network health

---

## Security Checklist

### Pre-Trading Security

☐ **Secure your private key** — Hardware wallet recommended (Ledger/Trezor)  
☐ **Secure your server** — Local API on encrypted machine  
☐ **Enable 2FA everywhere** — Email, exchange accounts  
☐ **Use dedicated trading wallet** — Don't mix with personal funds  
☐ **Test withdrawals** — Verify you can withdraw funds  
☐ **Document configurations** — Save API endpoints, market IDs  

### Ongoing Security

☐ **Monitor for unusual activity** — Set up alerts  
☐ **Keep software updated** — SDK, local API, OS  
☐ **Review access logs** — Check for unauthorized access  
☐ **Backup configurations** — API keys, settings  
☐ **Use rate limiting** — Don't overwhelm API  
☐ **Log out when done** — Clear sessions  

### What NOT To Do

 **Never share private keys** — With anyone, including "support"  
 **Never commit keys to git** — Use environment variables  
 **Never run local API on public server** — Local machine only  
 **Never ignore liquidation warnings** — Act immediately  
 **Never trade more than you can afford to lose** — Risk management first  

---

## Summary

**Remember the three pillars of safe trading:**

1. **Non-custodial = You are responsible** — No exchange to call for help
2. **Test everything first** — Devnet before mainnet, always
3. **Never give AI spending authority** — Human confirmation required

**When in doubt:**
- Ask questions
- Test on devnet
- Start small
- Keep learning

---

*Read [monitoring-guide.md](monitoring-guide.md) next for safe, read-only operations.*
