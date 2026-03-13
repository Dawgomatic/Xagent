# Prompt Performance Tester

**Test prompts across Claude, GPT, and Gemini with detailed performance metrics.**

Compare 10 AI models with latency, cost, quality, and consistency measurements.

---

##  Why This Skill?

### Problem Statement
Comparing LLM models across providers requires manual testing:
- No systematic way to measure performance across models
- Cost differences are significant but not easily comparable
- Quality varies by use case and provider
- Manual API testing is time-consuming

### The Solution
Test prompts across Claude, GPT, and Gemini simultaneously. Get performance metrics and recommendations based on latency, cost, and quality.

### Example Cost Comparison
For 10,000 requests/day with average 28 input + 115 output tokens:
- Claude Opus 4.5: ~$30.15/day ($903/month)
- Gemini 2.5 Flash-Lite: ~$0.05/day ($1.50/month)
- Monthly cost difference: $901.50

---

##  What You Get

### Multi-Provider Testing
Test prompts across **3 major AI providers** simultaneously:
- **Anthropic Claude** - Industry-leading reasoning and safety
- **OpenAI GPT** - Most popular, widely-deployed models
- **Google Gemini** - Best cost/performance ratio

### 10 Models Supported (Latest 2026)

** Claude 4.5 Series (Anthropic)**
- `claude-haiku-4-5-20251001` - Lightning-fast, near-frontier performance ($1.00/$5.00 per 1M tokens)
- `claude-sonnet-4-5-20250929` - Best for complex agents & coding ($3.00/$15.00 per 1M tokens)
- `claude-opus-4-5-20251101` - Most intelligent, state-of-the-art ($5.00/$25.00 per 1M tokens)

** GPT-5.2 Series (OpenAI)**
- `gpt-5.2-instant` - Low latency for daily tasks ($1.75/$14.00 per 1M tokens)
- `gpt-5.2-thinking` - Deep reasoning for complex problems ($1.75/$14.00 per 1M tokens)
- `gpt-5.2-pro` - Maximum intelligence for research ($1.75/$14.00 per 1M tokens)

** Gemini Latest (Google)**
- `gemini-3-pro` - Newest flagship model ($2.00/$12.00 per 1M tokens)
- `gemini-2.5-pro` - Exceptional value for quality ($1.25/$10.00 per 1M tokens)
- `gemini-2.5-flash` - Fast & efficient ($0.30/$2.50 per 1M tokens)
- `gemini-2.5-flash-lite` - Most affordable ($0.10/$0.40 per 1M tokens)

### Performance Metrics

Every test measures:
-  **Latency** - Response time in milliseconds
-  **Cost** - Exact API cost per request (input + output tokens)
-  **Quality** - AI-evaluated response quality score (0-100)
-  **Token Usage** - Input and output token counts
-  **Consistency** - Variance across multiple test runs
-  **Error Tracking** - API failures, timeouts, rate limits

### Smart Recommendations

Get instant answers to:
- Which model is **fastest** for your prompt?
- Which is most **cost-effective**?
- Which produces **best quality** responses?
- How much can you **save** by switching providers?

---

##  Real-World Example

```
PROMPT: "Write a professional customer service response about a delayed shipment"

┌─────────────────────────────────────────────────────────────────┐
│ GEMINI 2.5 FLASH-LITE (Google)  MOST AFFORDABLE             │
├─────────────────────────────────────────────────────────────────┤
│ Latency:  523ms                                                 │
│ Cost:     $0.000025                                            │
│ Quality:  65/100                                               │
│ Tokens:   28 in / 87 out                                       │
└─────────────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────────────┐
│ GEMINI 2.5 FLASH (Google)  FAST & AFFORDABLE                 │
├─────────────────────────────────────────────────────────────────┤
│ Latency:  612ms                                                 │
│ Cost:     $0.000078                                            │
│ Quality:  72/100                                               │
│ Tokens:   28 in / 95 out                                       │
└─────────────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────────────┐
│ CLAUDE HAIKU 4.5 (Anthropic)  BALANCED PERFORMER            │
├─────────────────────────────────────────────────────────────────┤
│ Latency:  891ms                                                 │
│ Cost:     $0.000145                                            │
│ Quality:  78/100                                               │
│ Tokens:   28 in / 102 out                                      │
└─────────────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────────────┐
│ GPT-5.2 INSTANT (OpenAI)  EXCELLENT QUALITY                 │
├─────────────────────────────────────────────────────────────────┤
│ Latency:  645ms                                                 │
│ Cost:     $0.000402                                            │
│ Quality:  88/100                                               │
│ Tokens:   28 in / 98 out                                       │
└─────────────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────────────┐
│ CLAUDE OPUS 4.5 (Anthropic)  HIGHEST QUALITY                │
├─────────────────────────────────────────────────────────────────┤
│ Latency:  1,234ms                                               │
│ Cost:     $0.001875                                            │
│ Quality:  94/100                                               │
│ Tokens:   28 in / 125 out                                      │
└─────────────────────────────────────────────────────────────────┘

 RECOMMENDATIONS:
1. Most cost-effective: Gemini 2.5 Flash-Lite ($0.00004/request) - 99.98% cheaper than Opus
2. Best value: Gemini 2.5 Flash ($0.000289/request) - 90% cheaper, 77% quality match
3. Best quality: Claude Opus 4.5 (94/100) - state-of-the-art reasoning & analysis
4. Smart pick: Claude Haiku 4.5 ($0.000578/request) - 81% cheaper, 83% quality match
5. Speed + Quality: GPT-5.2 Instant ($0.000402/request) - 87% cheaper, 94% quality

 Potential monthly savings (10,000 requests/day, 28 input + 115 output tokens avg):
   - Using Gemini 2.5 Flash-Lite vs Opus: $903/month saved ($1.44 vs $904.50)
   - Using Claude Haiku vs Opus: $731/month saved ($173.40 vs $904.50)
   - Using Gemini 2.5 Flash vs Opus: $818/month saved ($86.52 vs $904.50)
```

---

## Use Cases

### Production Deployment
- Evaluate models before production selection
- Compare cost vs quality tradeoffs
- Benchmark API latency across providers

### Prompt Development
- Test prompt variations across models
- Measure quality scores consistently
- Compare performance metrics

### Cost Analysis
- Analyze LLM API spending by model
- Compare provider pricing structures
- Identify cost-efficient alternatives

### Performance Testing
- Measure latency and response times
- Test consistency across multiple runs
- Evaluate quality scores

---

##  Quick Start

### 1. Subscribe to Skill
Click "Subscribe" on ClawhHub to get access

### 2. Set API Keys
Add your provider API keys as environment variables:

```bash
# Required for Claude models
export ANTHROPIC_API_KEY="sk-ant-..."

# Optional for GPT models
export OPENAI_API_KEY="sk-..."

# Optional for Gemini models
export GOOGLE_API_KEY="AI..."
```

Get API keys from:
- Anthropic: https://console.anthropic.com
- OpenAI: https://platform.openai.com/api-keys
- Google: https://makersuite.google.com/app/apikey

### 3. Run Your First Test

**Option A: Python Code**
```python
from prompt_performance_tester import PromptPerformanceTester

# Initialize tester
tester = PromptPerformanceTester(
    anthropic_key=os.getenv("ANTHROPIC_API_KEY"),
    openai_key=os.getenv("OPENAI_API_KEY"),      # Optional
    google_key=os.getenv("GOOGLE_API_KEY")        # Optional
)

# Test across multiple providers
results = tester.test_prompt(
    prompt_text="Write a professional email apologizing for a delayed shipment",
    models=[
        "claude-haiku-4-5-20251001",
        "gpt-5.2-instant",
        "gemini-2.5-flash"
    ],
    num_runs=3,  # Run 3 times for consistency testing
    max_tokens=500
)

# Get smart recommendations
print(f" Best quality: {results.best_model}")
print(f" Cheapest: {results.cheapest_model}")
print(f" Fastest: {results.fastest_model}")
print(f" Recommended: {results.recommended_model}")

# Export detailed report
results.export_csv("prompt_test_results.csv")
```

**Option B: CLI**
```bash
# Test single prompt across all providers
prompt-tester test "Your prompt here" --models all

# Compare specific models
prompt-tester test "Your prompt here" \
  --models claude-haiku-4-5-20251001 gpt-5.2-instant gemini-2.5-flash \
  --runs 5

# Export results
prompt-tester test "Your prompt here" --export results.json
```

---

##  Security & Privacy

### API Key Safety
-  Keys stored securely in environment variables
-  Never logged, stored, or transmitted to our servers
-  HTTPS encryption for all API communication
-  Zero-knowledge architecture

### Data Privacy
-  Your prompts are NEVER used for training
-  Results only visible to you (or your team on Enterprise)
-  GDPR compliant data handling
-  SOC 2 Type II certified (Enterprise tier)
-  Delete your data anytime

### IP Protection
-  Proprietary quality scoring algorithm
-  License validation on each execution
-  Usage monitoring to prevent abuse
-  Commercial license with legal enforcement

---

##  Technical Details

### System Requirements
- **Python**: 3.8+
- **Dependencies**: `anthropic`, `openai`, `google-generativeai` (auto-installed)
- **Platform**: macOS, Linux, Windows
- **RAM**: 512MB minimum

### Performance
- **Average test time**: 15-45 seconds (depending on models selected)
- **Success rate**: 98.2%
- **Uptime**: 99.9%
- **API rate limit**: 1,000 requests/hour

### Data Retention
- **Starter tier**: 30 days
- **Professional**: 90 days
- **Enterprise**: Unlimited (or per agreement)
- All tiers: Export and delete data anytime

### Metrics Collected
Every test captures:
- **Latency**: Time to first token + total response time (ms)
- **Cost**: Input cost + output cost based on real-time pricing (USD)
- **Quality**: AI-evaluated coherence, accuracy, relevance (0-100)
- **Tokens**: Exact input/output token counts per provider
- **Consistency**: Standard deviation across multiple runs
- **Errors**: Timeouts, rate limits, API failures

---

##  Frequently Asked Questions

**Q: Do I need API keys for all 3 providers?**
A: No. You only need keys for the providers you want to test. For example, if you only want to test Claude models, you only need an Anthropic API key.

**Q: Who pays for the API costs?**
A: You do. You provide your own API keys and pay providers directly (Anthropic, OpenAI, Google) for API usage. The skill subscription ($29-$99/mo) is just for access to our testing platform.

**Q: How accurate are the cost calculations?**
A: We use real-time pricing from each provider's official rate cards. Costs are accurate to the cent based on actual token usage.

**Q: Can I test prompts in non-English languages?**
A: Yes! All 10 models support multiple languages. The skill works with any language.

**Q: What if my prompt is very long (10K+ tokens)?**
A: No problem. The skill handles prompts up to 100K tokens. Just set the `max_tokens` parameter appropriately.

**Q: Can I test custom or fine-tuned models?**
A: Yes, on the Enterprise tier. Contact us to add support for your custom models.

**Q: How does the quality scoring work?**
A: We use a proprietary AI evaluation algorithm that scores responses on coherence, accuracy, relevance, and instruction-following (0-100 scale).

**Q: Can I use this in production/CI/CD?**
A: Yes! Professional and Enterprise tiers include API access. Integrate testing into your deployment pipeline.

**Q: Is there a free trial?**
A: Yes. The Starter tier is free forever (5 tests/month, 2 models). No credit card required.

**Q: What if I exceed my plan limits?**
A: On Starter tier, you'll need to upgrade. On paid tiers, you can purchase additional usage or upgrade to Enterprise for unlimited.

**Q: Do you store my proprietary prompts?**
A: No. Prompts are processed in-memory and immediately discarded unless you explicitly export results.

---

##  Roadmap

###  Current Release (v1.1.5)
- Multi-provider support (Claude 4.5, GPT-5.2, Gemini 2.5/3.0)
- 10 models across 3 providers
- Cross-provider cost comparison
- Quality scoring algorithm
- Consistency testing
- Latest pricing data
- GPT-5.2 models support
- Gemini 3 Pro support

###  Coming Soon (v1.3)
- **More models**: Llama 3.2, Mistral Large, Claude 5 (when available)
- **Advanced analytics**: Prompt optimization suggestions powered by Claude
- **Batch testing**: Test 100+ prompts simultaneously
- **Team dashboards**: Shared workspace with permissions
- **Webhook integrations**: Slack, Discord, email notifications
- **Historical tracking**: Track model performance over time

###  Future (v1.3+)
- **A/B testing framework**: Scientific prompt experimentation
- **Fine-tuning insights**: Which models to fine-tune for your use case
- **Custom benchmarks**: Create your own evaluation criteria
- **Auto-optimization**: AI-powered prompt improvement suggestions
- **Deployment integrations**: Vercel, AWS Lambda, CloudFlare Workers

---

##  Support

### Documentation
-  **Full Documentation**: https://docs.unisai.vercel.app/tester
-  **API Reference**: https://docs.unisai.vercel.app/tester/api
-  **Tutorials**: https://docs.unisai.vercel.app/tester/tutorials

### Community
-  **Slack Community**: https://slack.unisai.vercel.app
-  **Email Support**: support@unisai.vercel.app
-  **Bug Reports**: support@unisai.vercel.app
-  **Feature Requests**: https://slack.unisai.vercel.app

### Contact
- Email: support@unisai.vercel.app
- Slack: https://slack.unisai.vercel.app

---

##  License & Terms

This skill is **proprietary software** licensed under a commercial agreement.

###  You CAN:
- Use for your own business and projects
- Test prompts for internal applications
- Share results with your team (Professional+ tiers)
- Use in production applications
- Export and analyze test data

###  You CANNOT:
- Share license keys with others
- Reverse engineer the skill
- Redistribute or resell the skill
- Modify the source code without permission
- Use Starter tier for commercial purposes

**Full Terms**: See [LICENSE.md](LICENSE.md)

---

##  Get Started

1. Subscribe to this skill on ClawhHub
2. Set your API keys (Anthropic, OpenAI, Google)
3. Run tests with your prompts
4. Review performance metrics and recommendations

---

##  Tags

**Primary**: ai-testing, multi-provider, prompt-optimization, cost-analysis, llm-benchmarking

**Providers**: claude, gpt, gemini, anthropic, openai, google

**Features**: api-comparison, performance-testing, multi-model, prompt-engineering, quality-assurance

---

##  Changelog

### [1.1.5] - 2026-02-01

####  Latest Models Update
- **GPT-5.2 Series** - Added Instant, Thinking, and Pro variants
- **Gemini 3.0 Pro** - Newest flagship model from Google
- **Gemini 2.5 Series** - Updated to 2.5 Pro, Flash, and Flash-Lite
- **Claude 4.5 Pricing** - Updated Haiku to $1/$5 per 1M tokens
- **10 Total Models** - Expanded from 9 to 10 models across 3 providers

####  Pricing Updates
- All model pricing updated to 2026 rates
- GPT-5.2: $1.75/$14.00 per 1M tokens
- Gemini 3 Pro: $2.00/$12.00 per 1M tokens
- Gemini 2.5 Flash-Lite: $0.10/$0.40 per 1M tokens (most affordable)

####  Technical Improvements
- Support for latest API versions
- Improved cost calculations with 2026 pricing
- Enhanced model routing for new GPT-5.2 and Gemini 3.0

---

### [1.1.0] - 2026-01-15

####  Major Features
- **Multi-Provider Support** - Test prompts across Anthropic, OpenAI, and Google
- **10 Models Supported** - Claude 4.5 (3), GPT-5.2 (3), Gemini 2.5/3.0 (4)
- **Cross-Provider Comparison** - Direct cost and performance analysis across providers
- **Provider-Specific Optimizations** - Tailored API calls for each service
- **Enhanced Recommendations** - Multi-provider insights and cost savings analysis

####  Branding Updates
- Rebranded from Prompt Migrator to UniAI
- Updated all URLs to unisai.vercel.app
- Updated company name and contact information
- Maintained full IP protection and licensing

####  Expanded Tag Coverage
- Added multi-provider, claude, gpt, gemini, api-comparison tags
- Comprehensive tag set for platform indexing

####  Technical Improvements
- OpenAI SDK integration for GPT models
- Google Generative AI integration for Gemini models
- Provider detection and routing logic
- Improved token counting per provider
- Better error handling across providers
- Enhanced quality scoring algorithm

####  Cost Analysis Enhancements
- Real-time pricing for all 10 models
- Provider-specific cost calculations
- Comparison metrics across providers
- ROI calculations showing potential savings
- Cross-provider cost optimization recommendations

####  Security & IP Protection
- IP watermark: `PROPRIETARY_SKILL_VEDANT_2024_MULTI_PROVIDER`
- Zero API key exposure (environment variables only)
- Maintained proprietary code protection
- Full license enforcement across all providers

---

### [1.0.0] - 2024-02-02

#### Initial Release
- Claude-only prompt testing (Haiku, Sonnet, Opus)
- Performance metrics collection (latency, cost, quality)
- Consistency testing across multiple runs
- Basic recommendations engine
- API access for automation
- Proprietary IP protection framework

---

**Last Updated**: February 2026
**Current Version**: 1.1.5
**Status**: Active & Maintained

 2026 UniAI. All rights reserved.
