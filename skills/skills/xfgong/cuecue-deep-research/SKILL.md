---
name: cuecue-deep-research
description: Conduct deep financial research using CueCue's AI-powered multi-agent system
version: 1.0.3
author: CueCue Team
keywords:
  - research
  - financial-analysis
  - ai-agents
  - report-generation
  - data-analysis
  - imitation-writing
metadata: {"clawdbot":{"emoji":"","requires":{"env":["CUECUE_API_KEY"]},"primaryEnv": "CUECUE_API_KEY"}}

---

# CueCue Deep Research TypeScript Skill

Execute comprehensive financial research queries using CueCue's multi-agent AI system. This TypeScript implementation provides the same functionality as the Python version with modern async/await patterns and full type safety.

## What This Skill Does

CueCue Deep Research orchestrates multiple AI agents to:

1. **Analyze** your research question and break it down into actionable tasks
2. **Research** using web crawling, financial databases, and knowledge retrieval
3. **Synthesize** findings into a comprehensive markdown report
4. **Generate** a shareable report URL

The skill filters the verbose agent workflow to show only:
-  Task titles (from the supervisor agent)
-  Final research report (from the reporter agent)
-  Report URL for web viewing

 **Execution Time**: Depending on the complexity of your research question, the process may take **5-30 minutes**. The system performs comprehensive research including web crawling, data analysis, and report generation. Please be patient and wait for the complete results.

## For AI Assistants

**Important**: When using this skill, you MUST monitor the research progress by checking the command output:

1. **Progress Monitoring**: The research process outputs progress information in real-time. You should check the output **every 5 minutes** to:
   - Verify the research is still running
   - Report task progress to the user ( Task updates)
   - Detect any errors or issues
   - Inform the user when report generation begins ( Generating Report...)

2. **Progress URL**: The command will output a URL like "Research begin. You can view progress at: https://cuecue.cn/c/..." - this URL is for **human users** to view the web interface, NOT for you to fetch. You should monitor progress through the command's stdout output.

3. **User Communication**: Keep the user informed about:
   - When research begins
   - Each major task that starts
   - When report generation begins
   - When research completes
   - Any errors or timeouts

4. **Timeout Handling**: If the command appears to hang or timeout, inform the user that the research may still be processing on the server, and they can check the web interface URL.

5. **User-Facing Communication Style**: When informing users about progress monitoring:
   -  **DO SAY**: "我会每 5 分钟自动检查进度并汇报" / "I'll check progress every 5 minutes and update you"
   -  **DO SAY**: "研究完成后我会总结关键发现" / "I'll summarize key findings when complete"
   -  **DON'T SAY**: "我创建了一个 cron 任务" / "I created a cron job"
   -  **DON'T SAY**: "完成后会删除 cron" / "Will delete the cron when done"
   -  **DON'T SAY**: Technical implementation details (session IDs, job IDs, internal mechanisms)
   
   **Why**: Users care about *what* you'll do, not *how* you do it. Keep communication focused on outcomes and user value, not internal plumbing.

### Automated Progress Monitoring with Cron

Since AI assistants cannot actively "loop and check" on their own, use OpenClaw's cron system to automate progress monitoring.

**Recommended Approach: Use Isolated Session**

Using `sessionTarget: "isolated"` with `payload.kind: "agentTurn"` is the most reliable way to get progress updates delivered directly to the chat channel.

**Step 1: Start research in background**

```javascript
// Start the research task
exec({
  command: "cuecue-research 'Your research query' --output ~/clawd/cuecue-reports/2026-02-01-10-00-research.md",
  background: true
})
// Returns: { sessionId: "wild-river", pid: 12345 }
```

**Step 2: Create a cron job to monitor progress**

```javascript
cron.add({
  name: "Monitor CueCue Research: wild-river",
  schedule: {
    kind: "every",
    everyMs: 300000  // 5 minutes
  },
  sessionTarget: "isolated",
  wakeMode: "now",  // IMPORTANT: Use "now" to trigger immediately
  payload: {
    kind: "agentTurn",
    message: "检查 CueCue 研究进度 (session: wild-river)。使用 process log wild-river 检查输出。如果看到 ' Research complete'，则：1) 读取报告文件并总结关键发现；2) 使用 cron.remove 删除此监控任务。如果仍在运行，汇报最新的  Task 进度。",
    deliver: true,
    channel: "feishu",  // or "telegram", "discord", etc.
    to: "GROUP_ID_OR_CHAT_ID"  // The channel where the research was requested
  }
})
// Returns: { id: "abc-123-def", ... }
```

**Important Configuration:**
- **`sessionTarget: "isolated"`** - Creates an isolated sub-agent session
- **`payload.kind: "agentTurn"`** - Runs the agent and delivers the response
- **`deliver: true`** - Ensures the response is sent to the chat
- **`channel`** - Specify the messaging platform (feishu, telegram, discord, etc.)
- **`to`** - The target chat/group ID where updates should be sent
- **`wakeMode: "now"`** - Triggers immediately without waiting for heartbeat

**Step 3: Cron will automatically check every 5 minutes**

The cron job will:
- Run in an isolated session every 5 minutes
- Check `process log wild-river` for new output
- Send progress updates directly to the specified chat channel
- When complete, read the report, summarize findings, and delete itself

**Step 4: Manual cleanup (if needed)**

If the research fails or you need to stop monitoring:

```javascript
// List all cron jobs
cron.list()

// Remove the monitoring job
cron.remove({ jobId: "abc-123-def" })
```

**Complete Example Workflow:**

```javascript
// 1. Start research
const result = exec({
  command: "cuecue-research '2026年金银价格分析' --output ~/clawd/cuecue-reports/2026-02-01-gold-analysis.md",
  background: true
})
const sessionId = result.sessionId  // e.g., "wild-river"

// 2. Get current channel info (from runtime context)
const channel = "feishu"  // Current channel
const chatId = "oc_abac3e3037a0726ef4b4aa330d5ed590"  // Current group/chat ID

// 3. Create monitoring cron
const cronJob = cron.add({
  name: `Monitor CueCue: ${sessionId}`,
  schedule: { kind: "every", everyMs: 300000 },
  sessionTarget: "isolated",
  wakeMode: "now",
  payload: {
    kind: "agentTurn",
    message: `检查研究进度 (session: ${sessionId})。完成后读取报告并总结，然后删除此 cron。`,
    deliver: true,
    channel: channel,
    to: chatId
  }
})

// 4. Inform user (user-friendly, no technical details)
reply(` 研究已启动！
 进度追踪: https://cuecue.cn/c/...
 我会每 5 分钟自动检查进度并汇报
`)

// 5. Cron handles the rest automatically
// The isolated session will:
// - Check progress every 5 minutes
// - Send updates directly to the chat
// - Summarize the report when complete
// - Delete itself
```

**Troubleshooting:**

If you don't receive progress updates:
1. Check `cron.list()` to verify the job is running (`lastStatus: "ok"`)
2. Ensure `wakeMode: "now"` is set (not `"next-heartbeat"`)
3. Verify `deliver: true` and correct `channel` + `to` values
4. Check `cron.runs({ jobId })` for execution history and errors

**Alternative: Main Session (Not Recommended)**

If you prefer to use `sessionTarget: "main"` with `payload.kind: "systemEvent"`, note that:
- Responses are NOT automatically sent to the chat
- You must ensure `HEARTBEAT.md` contains non-comment content to avoid `empty-heartbeat-file` errors
- The isolated session approach is more reliable for automated notifications

**Note**: The cron payload should include logic to delete itself. Use `cron.remove({ jobId: "<job-id>" })` when the research completes.

## Prerequisites

- Node.js 18+ or Deno
- CueCue API key (obtain from your CueCue account settings from https://cuecue.cn)
- npm or yarn package manager

## Configuration

### Setting up CUECUE_API_KEY

The skill requires a CueCue API key to function. You can configure it in two ways:

#### Option 1: OpenClaw Config (Recommended)

Set the API key in your OpenClaw configuration using the CLI:

```bash
# One-line command to set the API key(openclaw command may be clawdbot or moltbot)
openclaw config set skills.entries.cuecue-deep-research.env.CUECUE_API_KEY "your-api-key-here"
```

This will:
- Add the key to `~/.openclaw/openclaw.json` under `skills.entries.cuecue-deep-research.env`
- Make it available to the skill automatically
- Restart the gateway to apply changes

To verify the configuration:
```bash
openclaw config get skills.entries.cuecue-deep-research.env.CUECUE_API_KEY
```

then restart the gateway
```
openclaw gateway restart
```

#### Option 2: Command-Line Argument

You can also pass the API key directly when running the command:
```bash
cuecue-research "Your query" --api-key YOUR_API_KEY
```

**Note**: The OpenClaw config method is recommended because:
- You don't need to specify the key every time
- The key is stored securely in the OpenClaw config
- All AI assistants can use the skill without needing the key in prompts
- One command to set it up

### Getting Your API Key

1. Visit https://cuecue.cn
2. Log in to your account
3. Go to Account Settings
4. Find the API Key section
5. Copy your API key

## Installation

### As a CLI Tool

```bash
# Install globally
npm install -g cuecue-deep-research@1.0.3

# Or install locally in your project
npm install cuecue-deep-research@1.0.3
```

## Usage

### Command-Line Interface

#### Basic Research

```bash
# Using environment variable (recommended)
cuecue-research "Tesla Q3 2024 revenue analysis"

# Or specify API key directly
cuecue-research "Tesla Q3 2024 revenue analysis" --api-key YOUR_API_KEY
```

#### Save Report to File

```bash
cuecue-research "BYD vs Tesla market comparison" --output ~/clawd/cuecue-reports/2026-01-30-14-30-byd-tesla-comparison.md
```

**Note**: The output path should use the format `~/clawd/cuecue-reports/YYYY-MM-DD-HH-MM-descriptive-name.md` where the timestamp represents when the research was initiated. The `~` will be expanded to your home directory.

#### Use a Research Template

```bash
cuecue-research "Analyze CATL competitive advantages" \
  --output ~/clawd/cuecue-reports/2026-01-30-11-20-catl-analysis.md \
  --template-id TEMPLATE_ID
```

#### Continue Existing Conversation

```bash
cuecue-research "Further analyze supply chain risks" \
  --output ~/clawd/cuecue-reports/2026-01-30-15-45-supply-chain-risks.md \
  --conversation-id EXISTING_CONV_ID
```

#### Mimic Writing Style

```bash
cuecue-research "Electric vehicle market analysis" \
  --output ~/clawd/cuecue-reports/2026-01-30-16-00-ev-market-analysis.md \
  --mimic-url https://example.com/sample-article
```

The mimic feature analyzes the writing style, tone, and structure of the provided URL and applies it to the generated research report. This is useful for:
- Matching your organization's reporting style
- Adapting to specific audience preferences
- Maintaining consistency across reports

 **Note**: The `--mimic-url` and `--template-id` options cannot be used together. Choose one approach:
- Use `--template-id` for predefined research frameworks (goal, search plan, report format)
- Use `--mimic-url` for style mimicking without a template


## Command-Line Options

| Option | Required | Description |
|--------|----------|-------------|
| `query` |  | Research question or topic |
| `--api-key` |  | Your CueCue API key (defaults to `CUECUE_API_KEY` env var) |
| `--base-url` |  | CueCue API base URL (defaults to `CUECUE_BASE_URL` env var or https://cuecue.cn) |
| `--conversation-id` |  | Continue an existing conversation |
| `--template-id` |  | Use a predefined research template (cannot be used with `--mimic-url`) |
| `--mimic-url` |  | URL to mimic the writing style from (cannot be used with `--template-id`) |
| `--output`, `-o` |  | Save report to file (markdown format). Recommended format: `~/clawd/cuecue-reports/clawd/cuecue-reports/YYYY-MM-DD-HH-MM-descriptive-name.md` (e.g., `~/clawd/2026-01-30-12-41-tesla-analysis.md`). The `~` will be expanded to your home directory. |
| `--verbose`, `-v` |  | Enable verbose logging |
| `--help`, `-h` |  | Show help message |

## Output Format

The skill provides real-time streaming output:

```
Starting Deep Research: Tesla Q3 2024 Financial Analysis

Check Progress: https://cuecue.cn/c/12345678-1234-1234-1234-123456789abc

 Task: Search for Tesla Q3 2024 financial data

 Task: Analyze revenue and profit trends

 Generating Report...

# Tesla Q3 2024 Financial Analysis

## Executive Summary
[Report content streams here in real-time...]

 Research complete

============================================================
 Research Summary
============================================================
Conversation ID: 12345678-1234-1234-1234-123456789abc
Tasks completed: 2
Report URL: https://cuecue.cn/c/12345678-1234-1234-1234-123456789abc
 Report saved to: ~/clawd/cuecue-reports/2026-01-30-10-15-tesla-q3-analysis.md
```

## Troubleshooting

### 401 Unauthorized
- Verify your API key is correct
- Check if the API key has expired
- Ensure you have necessary permissions

### Connection Timeout
- Verify the base URL is correct
- Check network connectivity
- Research queries typically take 5-30 minutes depending on complexity - this is normal
- If you see a timeout, the research may still be processing on the server - check the web interface

### Empty Report
- Ensure your research question is clear and specific
- Check server logs for errors
- Try a different query to test connectivity

## Support

For issues or questions:
- [CueCue Website](https://cuecue.cn)
- Email: cue-admin@sensedeal.ai
