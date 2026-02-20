# Xagent Security Audit Report
**Date:** 2026-02-15  
**Auditor:** SWE100821  
**Subject:** Backdoor and vulnerability assessment for xagent AI agent

---

## Executive Summary

✅ **NO CRITICAL SECURITY ISSUES OR BACKDOORS FOUND**

After thorough examination of the xagent codebase, I found **no evidence of malicious backdoors, data exfiltration, or unauthorized network communication**. The project appears to be a legitimate, open-source AI agent framework built in Go.

---

## Key Findings

### 1. ✅ Network Communication - CLEAN
**All external connections are legitimate and user-controlled:**

- **API Endpoints** (all configurable via `config.json`):
  - OpenRouter: `https://openrouter.ai/api/v1`
  - Anthropic: `https://api.anthropic.com/v1`
  - OpenAI: `https://api.openai.com/v1`
  - Gemini: `https://generativelanguage.googleapis.com/v1beta`
  - Groq: `https://api.groq.com/openai/v1`
  - Zhipu (Chinese): `https://open.bigmodel.cn/api/paas/v4`
  - Moonshot (Chinese): `https://api.moonshot.cn/v1`
  - DeepSeek: `https://api.deepseek.com/v1`
  - Brave Search: `https://api.search.brave.com/res/v1/web/search`
  - DuckDuckGo: `https://html.duckduckgo.com/html/`

**Verdict:** All endpoints are **explicit, documented, and user-configured**. No hidden connections detected.

---

### 2. ✅ Chinese Company Connection - LEGITIMATE
- **Company:** Sipeed (矽速科技) - established Chinese hardware maker
- **Domain:** Official site is `xagent.io` and `sipeed.com`
- **Purpose:** Low-cost AI agent for their hardware (LicheeRV-Nano, MaixCAM, etc.)
- **License:** MIT License (fully open source)
- **README Warning:** Explicitly warns about crypto scams using their name

**Verdict:** Transparent Chinese origin, but **no suspicious behavior**. Similar to other Chinese hardware companies (Sipeed makes RISC-V boards).

---

### 3. ✅ Data Handling - SECURE
**Configuration:**
- API keys stored locally in `~/.xagent/config.json`
- No hardcoded credentials found
- Workspace sandboxing enabled by default (`restrict_to_workspace: true`)

**Security Features:**
```json
{
  "restrict_to_workspace": true,  // ✅ Limits file access
  "workspace": "~/.xagent/workspace"  // ✅ Isolated directory
}
```

**Command Blocking:**
- Fork bombs: `:(){ :|:& };:`
- Disk formatting: `format`, `mkfs`, `diskpart`
- Bulk deletion: `rm -rf`, `del /f`
- System shutdown: `shutdown`, `reboot`
- Direct disk writes: `/dev/sd*`

---

### 4. ✅ No Telemetry or Tracking
**Search Results:**
- ❌ No telemetry code
- ❌ No analytics endpoints
- ❌ No "phone home" behavior
- ❌ No usage metrics collection

**Grep Results:** Only legitimate "report" references (e.g., "report current time", "report back results")

---

### 5. ✅ Dependencies - STANDARD
**Go Modules:** All from reputable sources
```
- Anthropic SDK: github.com/anthropics/anthropic-sdk-go
- OpenAI SDK: github.com/openai/openai-go/v3
- Discord: github.com/bwmarrin/discordgo
- Telegram: github.com/mymmrac/telego
- Slack: github.com/slack-go/slack
- OAuth2: golang.org/x/oauth2
```

**Chinese Dependencies:**
- `github.com/tencent-connect/botgo` (QQ bot SDK - expected for QQ channel)
- `github.com/bytedance/sonic` (JSON parser - widely used, open source)
- `github.com/open-dingtalk/dingtalk-stream-sdk-go` (DingTalk SDK - expected)

**Verdict:** Standard dependencies for a multi-platform chat agent. No suspicious packages.

---

### 6. ✅ Code Architecture - TRANSPARENT
**Main Entry Point:** `/cmd/xagent/main.go`
- Clean command structure
- No obfuscated code
- Clear initialization flow

**Provider System:** `/pkg/providers/http_provider.go`
- All API calls go through user-configured endpoints
- No hidden alternate endpoints
- Bearer token authentication only

**Tools:** `/pkg/tools/`
- Standard tools: filesystem, shell, web search
- No data exfiltration tools
- Workspace restrictions enforced

---

## Chinese Components - Risk Assessment

### ⚠️ Services with `.cn` domains:
1. **Zhipu AI** (`open.bigmodel.cn`) - Chinese LLM provider
   - **Risk:** LOW - Optional, user-configured, documented
   - **Mitigation:** Don't configure Zhipu if concerned

2. **Moonshot AI** (`api.moonshot.cn`) - Chinese LLM provider  
   - **Risk:** LOW - Optional, user-configured, documented
   - **Mitigation:** Use Western providers instead

3. **Tencent QQ** (via `tencent-connect/botgo`)
   - **Risk:** MEDIUM - If you enable QQ channel
   - **Mitigation:** Disable QQ channel, use Telegram/Discord

**Overall Assessment:** Chinese API endpoints are **optional features** that can be completely disabled. The software doesn't force their use.

---

## Potential Security Concerns (Minor)

### 1. Workspace Escape Risk (if disabled)
```json
"restrict_to_workspace": false  // ⚠️ Don't do this
```
**Impact:** Agent can access entire filesystem  
**Mitigation:** Keep default `true` value

### 2. Shell Execution Tool
```go
// In pkg/tools/shell.go
cmd := exec.CommandContext(ctx, "sh", "-c", command)
```
**Impact:** Can run arbitrary commands in sandbox  
**Mitigation:** Workspace restrictions + command blocking

### 3. Third-Party Skills
```bash
xagent skills install <github-repo>
```
**Impact:** Installs code from external GitHub repos  
**Mitigation:** Only install from trusted sources

---

## Recommendations

### ✅ Safe Configuration (Recommended)
```json
{
  "agents": {
    "defaults": {
      "workspace": "~/.xagent/workspace",
      "restrict_to_workspace": true,  // ✅ Keep enabled
      "model": "anthropic/claude-sonnet-4.5"  // ✅ Use Western provider
    }
  },
  "providers": {
    "anthropic": {
      "api_key": "YOUR_KEY",
      "api_base": "https://api.anthropic.com/v1"
    }
  },
  "channels": {
    "telegram": { "enabled": true },  // ✅ Safe
    "discord": { "enabled": true },   // ✅ Safe
    "qq": { "enabled": false },       // ⚠️ Avoid if paranoid
    "feishu": { "enabled": false }    // ⚠️ Chinese Lark/Feishu
  }
}
```

### 🛡️ Hardening Steps
1. **Use Western AI providers** (Anthropic, OpenAI, OpenRouter)
2. **Disable Chinese channels** (QQ, Feishu/Lark)
3. **Keep workspace restrictions** (`restrict_to_workspace: true`)
4. **Review skills before installing** (check GitHub repos)
5. **Run in Docker** (additional isolation layer)
6. **Monitor logs** (check for unexpected connections)

---

## Comparison with Similar Projects

| Feature | Xagent | AutoGPT | LangChain | Agent Zero |
|---------|----------|---------|-----------|------------|
| Open Source | ✅ MIT | ✅ MIT | ✅ MIT | ✅ MIT |
| Telemetry | ❌ None | ⚠️ Optional | ⚠️ Some | ❌ None |
| Chinese APIs | ⚠️ Optional | ❌ None | ⚠️ Some | ❌ None |
| Sandboxing | ✅ Yes | ⚠️ Partial | ❌ No | ⚠️ Partial |

---

## Final Verdict

### ✅ SAFE TO USE (with standard precautions)

**Reasons:**
1. ✅ No backdoors or malicious code found
2. ✅ All network calls are user-controlled and documented
3. ✅ Open source MIT license with clean code
4. ✅ Strong security defaults (workspace sandboxing)
5. ✅ No telemetry or data collection
6. ✅ Chinese components are **optional** and **documented**

**Risk Level:** **LOW**

**Recommendation:** 
- ✅ Safe for personal use with Western API providers
- ✅ Safe for development/testing
- ⚠️ For enterprise: Disable Chinese channels and use vetted providers
- ⚠️ For paranoid: Compile from source, audit dependencies, use Docker

---

## Test Commands

To verify the security yourself:

```bash
# Check for hidden network calls
cd xagent
grep -r "http\." pkg/ cmd/ | grep -v "test"
grep -r "POST\|GET" pkg/ | grep -v "test\|comment"

# Check for Chinese domains
grep -ri "\.cn" pkg/ cmd/

# Check for telemetry
grep -ri "telemetry\|analytics\|tracking" pkg/ cmd/

# Review all external dependencies
cat go.mod

# Audit configuration
cat config/config.example.json
```

---

## References
- Repository: https://github.com/sipeed/xagent
- License: MIT (confirmed at `LICENSE`)
- Company: Sipeed (https://sipeed.com)
- Inspired by: nanobot (HKUDS)

---

**Report Generated:** 2026-02-15  
**Auditor:** SWE100821
