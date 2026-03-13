# Channel Security Checks

## What to Examine

From `config.channels.*`, check each enabled channel:

## Check 1: DM Policy

**Location**: `channels.<type>.dm.policy` or `channels.<type>.dmPolicy`

| Policy | Risk Level | Notes |
|--------|------------|-------|
| `disabled` |  None | DMs disabled |
| `pairing` |  Low | Default, requires approval |
| `allowlist` |  Low | Only pre-approved users |
| `open` |  Critical | Anyone can DM |

**Finding if `open`**:
- Scenario: `prompt-injection` or `unauthorized-access`
- Impact: Anyone on the internet can send commands to your bot
- Recommendation: Change to `pairing` or `allowlist`

## Check 2: Group Policy

**Location**: `channels.<type>.groupPolicy`

| Policy | Risk Level | Notes |
|--------|------------|-------|
| `disabled` |  None | Groups disabled |
| `allowlist` |  Low | Only listed groups |
| `open` |  High | Bot responds in any group it's added to |

**Finding if `open`**:
- Check if powerful tools (exec, browser, gateway) are available
- If yes:  Critical
- If minimal tools only:  Medium

## Check 3: Per-Channel Tool Overrides

**Location**: `channels.<type>.channels.<channelName>.tools`

Look for:
- `alsoAllow: ["exec", "browser", "gateway"]` in public/open channels
- Missing `deny` list for sensitive tools

**Risk Assessment**:
```
Channel is public/open + exec allowed =  Critical
Channel is allowlist + exec allowed =  Medium (depends on trust level)
Channel is private team + exec allowed =  Low (note it, don't alarm)
```

## Check 4: requireMention Setting

**Location**: `channels.<type>.channels.<channelName>.requireMention`

- `true`: Bot only responds when @mentioned
- `false`: Bot responds to all messages

**Risk**:
- `false` in public channels =  High (more attack surface)
- `false` in private channels =  Low (convenience feature)

## Check 5: allowBots Setting

**Location**: `channels.<type>.allowBots`

- `true`: Bot responds to other bots
- `false`: Ignores bot messages

**Risk if `true`**:
-  Info - Note this for awareness
- Only  Medium if: open channel + webhook integrations + exec tools
- Often required for legitimate webhook/automation use cases

## Check 6: Empty Allowlist

**Location**: `channels.<type>.channels` with `groupPolicy: allowlist`

**Critical Check**:
- If `groupPolicy: allowlist` but no channels defined = effectively disabled
- If allowlist channels exist but all have `allow: false` = effectively disabled
- Empty allowlist with DMs enabled = only DMs work

**Risk**:
- Misconfiguration that may not match user intent
-  Info - Confirm this is intentional

## Context Considerations

Before flagging as high risk, check:

1. **Is there a VPN/Tailscale overlay?** 
   - If `gateway.tailscale.mode` is active, network exposure is limited
   
2. **Is Control UI properly secured?**
   - Check `gateway.controlUi.allowInsecureAuth`
   - Check `gateway.auth.mode`

3. **User's stated environment**
   - Single user on personal machine? Lower risk.
   - Team deployment? Higher scrutiny needed.
