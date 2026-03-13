---
name: Growth Engine
description: Design, execute, and measure growth systems — from North Star definition through viral loops, experimentation, and scaling. Complete AARRR+ framework with templates, scoring rubrics, and playbooks.
metadata: {"clawdbot":{"emoji":"","os":["linux","darwin","win32"]}}
---

# Growth Engine — Complete System

## Phase 1: North Star & Growth Model

### Define Your North Star Metric (NSM)

Your NSM is the ONE number that best captures the core value you deliver.

**Selection criteria** — must pass ALL four:
1. Reflects value delivered to customer (not just company revenue)
2. Is a leading indicator of revenue (predicts future revenue)
3. Every team can influence it (not siloed)
4. Measurable weekly or daily

**NSM by business type:**

| Business Model | North Star Metric | Why |
|---|---|---|
| B2B SaaS | Weekly Active Users performing core action | Value = usage |
| Marketplace | Transactions completed per week | Both sides got value |
| E-commerce | Purchase frequency per customer per quarter | Repeat = love |
| Content/Media | Weekly engaged reading time | Attention = value |
| Fintech | $ processed per user per month | More $ = more trust |
| Developer tools | Builds/deploys per user per week | Integration depth |
| Social | Daily content interactions | Network value |

**NSM Quality Test** (score 1-5 each, need 20+):
- [ ] Can you measure it weekly? ___/5
- [ ] Does improving it directly improve revenue? ___/5
- [ ] Can marketing influence it? ___/5
- [ ] Can product/engineering influence it? ___/5
- [ ] Would a competitor with a higher number be winning? ___/5
- Total: ___/25

### Growth Model — Input-Output Map

```yaml
growth_model:
  north_star: "[Your NSM]"
  inputs:
    - name: "New signups"
      current: 0
      target: 0
      lever: "acquisition"
    - name: "Activation rate"
      current: "0%"
      target: "0%"
      lever: "onboarding"
    - name: "Weekly retention"
      current: "0%"
      target: "0%"
      lever: "engagement"
    - name: "Referral rate"
      current: "0%"
      target: "0%"
      lever: "virality"
  formula: "NSM = new_signups × activation_rate × retention_rate × (1 + referral_rate)"
  current_nsm: 0
  target_nsm: 0
  bottleneck: "[Which input has the biggest gap?]"
```

**Rule:** Always fix the bottleneck input first. Growing acquisition when activation is broken = pouring water into a sieve.

---

## Phase 2: AARRR+ Funnel Deep Dive

### Stage 1 — Acquisition (How They Find You)

**Channel scoring matrix:**

```yaml
channel_evaluation:
  - channel: "[e.g., Google Ads]"
    estimated_cac: "$___"
    volume_potential: "low|medium|high"
    time_to_test: "days|weeks|months"
    competitive_density: "low|medium|high"
    content_fit: "1-5"  # How natural is your message here?
    ice_score: 0  # Impact × Confidence × Ease (1-10 each)
```

**Channel categories with tactics:**

**Paid (fast, expensive, measurable):**
- Google Ads → bottom-funnel, high intent, high CPC
- Meta Ads → top-funnel, broad targeting, visual-first
- LinkedIn Ads → B2B, expensive ($8-15 CPC), precise targeting
- Influencer/creator → trust transfer, variable ROI
- Podcast sponsorship → niche audiences, hard to track

**Organic (slow, cheap, compounding):**
- SEO → 3-6 month payoff, compounds forever
- Content marketing → thought leadership, lead magnets
- Social media → brand, community, distribution
- Community → forums, Discord, Slack groups
- Product Hunt / marketplaces → launch spikes

**Product-led (scalable, free):**
- Referral program → viral coefficient
- Integrations/marketplace → partner distribution
- Freemium → try-before-buy, land-and-expand
- Open source → community-driven awareness
- API/embeds → other products distribute you

**Sales-led (high ACV, relationship):**
- Outbound (email/LinkedIn) → targeted, low volume
- Partnerships/channel → leverage others' audiences
- Events/conferences → high-touch, expensive
- Account-based marketing → 1:many to 1:1

**Test protocol:**
1. Score all candidate channels using ICE
2. Pick top 3 (one from each category if possible)
3. Set budget cap: $500 or 2 weeks, whichever comes first
4. Define success metric BEFORE starting
5. Kill anything below threshold at budget cap
6. Double down on winner

### Stage 2 — Activation (First Value Moment)

**"Aha Moment" Discovery Framework:**

Step 1 — Hypothesize:
- What action do retained users take that churned users don't?
- When does a user first say "this is worth it"?
- What's the minimum experience to demonstrate core value?

Step 2 — Validate with data:
```
Compare: Users retained at Day 30 vs churned before Day 30
Find: What % of each group completed [action] in first [timeframe]?
If: Retained users do [action] at 3x+ rate → that's your aha moment
```

Step 3 — Define activation metric:
```yaml
activation:
  aha_moment: "[Specific action, e.g., 'Created first project with 3+ tasks']"
  target_timeframe: "[e.g., Within first 48 hours]"
  current_rate: "___% of signups reach aha moment"
  target_rate: "___% (aim for 40-60% minimum)"
  steps_to_aha:
    - step: "Sign up"
      current_completion: "100%"
      drop_off: "0%"
    - step: "[Step 2]"
      current_completion: "____%"
      drop_off: "____%"
    - step: "[Aha moment]"
      current_completion: "____%"
      drop_off: "____%"
```

**Activation optimization checklist:**
- [ ] Remove every form field not absolutely required at signup
- [ ] Show value BEFORE asking for commitment (email, payment)
- [ ] Personalize onboarding by use case / persona
- [ ] Use progressive disclosure — don't show everything at once
- [ ] Add progress indicators (steps 1/3, progress bars)
- [ ] Pre-populate with sample data so they see the product "working"
- [ ] Send triggered emails at each drop-off point
- [ ] Offer live chat/support during first session
- [ ] A/B test onboarding flows — small changes = big impact
- [ ] Measure time-to-value, not just completion rate

### Stage 3 — Retention (Do They Come Back?)

**Cohort analysis template:**

```
Week 0 | Week 1 | Week 2 | Week 3 | Week 4 | Week 8 | Week 12
100%   |  ___% |  ___% |  ___% |  ___% |  ___% |  ___% 
```

**Retention curve diagnosis:**
- **Flattens above 20%** → Product has core value. Focus on moving the flat line UP and reducing early drop-off
- **Flattens below 10%** → Niche value. Either expand use cases or accept small market
- **Never flattens (→ 0%)** → Product problem. Stop all growth spending. Fix product.
- **Early cliff (Week 1 drop > 60%)** → Activation problem. Users never got value.
- **Gradual decline** → Engagement problem. Need habit loops or re-engagement.

**Retention improvement playbook:**

| Retention Problem | Tactic | Expected Impact |
|---|---|---|
| Day 1 drop > 50% | Fix onboarding, reduce time-to-value | High |
| Week 1 drop > 70% | Trigger emails, in-app nudges, help | High |
| Gradual decline | Build habit loops, notifications, content | Medium |
| Sudden cliff at Day X | Find what breaks — billing? Feature wall? | High |
| Seasonal churn | Re-engagement campaigns before drop | Medium |

**Habit loop design:**
```yaml
habit_loop:
  trigger: "[What reminds them to return? Email, notification, calendar, peer]"
  action: "[What do they do? Must be easy and quick]"
  variable_reward: "[What's different each time? New content, data, social]"
  investment: "[What do they put in that makes leaving harder? Data, connections, customization]"
  frequency: "[How often should the loop fire? Daily, weekly, event-driven]"
```

### Stage 4 — Revenue (Are They Paying?)

**Monetization readiness checklist:**
- [ ] Users consistently reach aha moment (activation > 40%)
- [ ] Retention curve flattens (product-market fit signal)
- [ ] Users request features / complain about limits (willingness to pay signal)
- [ ] Competitor charges for similar value (market validation)
- [ ] Unit economics work at current scale (or projected)

**Pricing strategy quick-select:**

| Signal | Strategy | Example |
|---|---|---|
| High volume, low willingness to pay | Freemium + upsell | Slack, Dropbox |
| Low volume, high willingness to pay | Sales-led, annual contracts | Salesforce |
| Usage varies wildly | Usage-based | AWS, Twilio |
| Clear feature tiers | Good/Better/Best | Most SaaS |
| Network effects | Free for users, charge businesses | LinkedIn |

**Revenue metrics to track:**
- MRR / ARR (growth rate month-over-month)
- ARPU (average revenue per user) — segment by plan
- Conversion rate (free → paid) — benchmark: 2-5% freemium, 10-25% free trial
- Expansion revenue % (upsells + cross-sells as % of new revenue)
- Net Revenue Retention (NRR) — benchmark: >100% good, >120% great
- LTV:CAC ratio — benchmark: >3:1
- Payback period — benchmark: <12 months

### Stage 5 — Referral (Are They Telling Others?)

**Viral coefficient formula:**
```
K = invitations_per_user × conversion_rate_per_invitation
K > 1 = exponential growth (very rare)
K = 0.3-0.7 = meaningful viral supplement
K < 0.1 = referral isn't a growth lever (yet)
```

**Referral program design template:**

```yaml
referral_program:
  type: "double-sided|single-sided|milestone"
  giver_incentive: "[What the referrer gets]"
  receiver_incentive: "[What the new user gets]"
  trigger_moment: "[When to ask — after value delivery, not before]"
  mechanic: "link|code|invite|auto-detect"
  sharing_channels:
    - channel: "email"
      template: "[Pre-written share message]"
    - channel: "social"
      template: "[Share-optimized copy + visual]"
    - channel: "in-app"
      template: "[Invite flow within product]"
  fraud_prevention:
    - "Reward on activation, not signup"
    - "Limit rewards per user per month"
    - "Flag same-IP signups"
  tracking:
    - invites_sent_per_user
    - invite_conversion_rate
    - time_from_invite_to_activation
    - viral_coefficient_k
```

**Referral timing rules:**
-  Ask AFTER user achieves success (completed project, got result, hit milestone)
-  Ask when user gives positive feedback (NPS 9-10, support thank you)
-  Never ask during onboarding (they haven't gotten value yet)
-  Never ask immediately after payment (feels extractive)
-  Ask when user invites a team member (they're already sharing)

---

## Phase 3: Growth Loops

### Loop 1 — Content Loop (SEO + Content)

```
Create valuable content → Google/social indexes it → New users discover it
→ Some users create content (UGC) or share → More content → More discovery
```

**Content-led growth playbook:**
1. Find 50 keywords your audience searches (tools: Ahrefs, Google autocomplete, "People also ask")
2. Cluster into 5-7 topic pillars
3. Create 1 pillar page per cluster (3,000+ words, comprehensive)
4. Create 5-10 supporting posts per pillar (long-tail keywords)
5. Internal link everything to pillar pages
6. Add lead magnets to top 20% of traffic pages
7. Repurpose top posts into social, email, video
8. Track: organic traffic → signups → activation → revenue per content piece

**Content ROI tracking:**
```yaml
content_piece:
  title: ""
  url: ""
  publish_date: ""
  target_keyword: ""
  monthly_traffic: 0
  signups_attributed: 0
  revenue_attributed: "$0"
  cac_equivalent: "$0"  # What would this traffic cost via ads?
  status: "growing|plateau|declining"
```

### Loop 2 — Viral Loop (Product-Led)

```
User gets value → Shares/invites → New users see value → They share → Compound
```

**Viral mechanics ranked by strength:**
1. **Inherent virality** — product requires others (Zoom, Slack, Figma multiplayer)
2. **Collaborative virality** — better with others (Notion shared workspaces)
3. **Output virality** — work product is visible (Canva "Made with Canva")
4. **Incentivized virality** — rewards for sharing (Dropbox extra storage)
5. **Social proof virality** — badges, profiles, leaderboards
6. **Word of mouth** — so good people talk about it (no mechanic needed)

**Design your viral loop:**
```yaml
viral_loop:
  type: "[inherent|collaborative|output|incentivized|social_proof|wom]"
  trigger: "[What makes them share?]"
  payload: "[What does the recipient see?]"
  landing: "[Where do they land? Must show value immediately]"
  conversion: "[What's the first action for the new user?]"
  cycle_time: "[How long from share to new share?]"
  current_k: 0
  target_k: 0
```

### Loop 3 — Paid Loop (Profitable Acquisition)

```
Revenue → Reinvest in ads → New users → Revenue → Reinvest more
```

**Unit economics requirement:**
```
LTV > 3× CAC (minimum for paid to be sustainable)
Payback period < 12 months (cash flow)
Marginal CAC < Average CAC (scaling efficiently)
```

**Paid growth scaling checklist:**
- [ ] CAC stable or declining at current spend level
- [ ] Creative fatigue monitored (refresh every 2-4 weeks)
- [ ] Audience segmented (lookalikes, retargeting, cold)
- [ ] Attribution tracked (UTM, pixel, conversion API)
- [ ] Landing pages A/B tested per channel
- [ ] Budget increases in 20% increments (not 2x jumps)
- [ ] Daily spend caps set to prevent blowouts
- [ ] Negative keywords / exclusions maintained weekly

### Loop 4 — Sales Loop (High ACV)

```
Sales closes deal → Customer succeeds → Case study + referral → Pipeline → Sales closes
```

**Sales-led growth framework:**
```yaml
sales_loop:
  ideal_customer:
    company_size: ""
    industry: ""
    budget_range: ""
    buying_trigger: ""
  outbound_velocity:
    emails_per_week: 0
    meetings_per_week: 0
    proposals_per_month: 0
    close_rate: "0%"
  case_study_production:
    cadence: "Every closed deal > $X"
    format: "Problem → Solution → Results (with numbers)"
    distribution: ["website", "sales deck", "social", "email"]
  referral_ask:
    timing: "90 days post-close, after first success milestone"
    script: "Who else in your network faces [problem we solved for you]?"
```

---

## Phase 4: Experimentation Engine

### Experiment Design Template

```yaml
experiment:
  id: "EXP-001"
  name: ""
  hypothesis: "If we [change], then [metric] will [increase/decrease] by [amount] because [reason]"
  primary_metric: ""
  secondary_metrics: []
  funnel_stage: "acquisition|activation|retention|revenue|referral"
  ice_score:
    impact: 0  # 1-10: How much will this move the metric?
    confidence: 0  # 1-10: How confident based on evidence?
    ease: 0  # 1-10: How fast/cheap to implement?
    total: 0  # I × C × E
  sample_size_needed: 0
  duration: ""
  variant_a: "[Control — current experience]"
  variant_b: "[Treatment — the change]"
  success_threshold: "[e.g., >10% improvement at 95% confidence]"
  status: "planned|running|complete|killed"
  result: ""
  learning: ""
```

### ICE Prioritization Board

Run experiments highest ICE score first. Review weekly.

```
| ID | Name | I | C | E | ICE | Stage | Status |
|----|------|---|---|---|-----|-------|--------|
```

### Statistical Significance Rules

**Minimum sample sizes (for 95% confidence, 80% power):**

| Baseline Rate | Minimum Detectable Effect | Sample per Variant |
|---|---|---|
| 2% | 50% relative (2% → 3%) | ~4,700 |
| 5% | 20% relative (5% → 6%) | ~14,700 |
| 10% | 10% relative (10% → 11%) | ~14,400 |
| 30% | 5% relative (30% → 31.5%) | ~22,600 |

**Rules:**
- Never peek at results before minimum duration (peeking inflates false positives)
- Minimum 1 full business cycle (usually 1-2 weeks)
- If you can't get enough traffic, test bigger changes (not subtle ones)
- Sequential testing frameworks (e.g., Bayesian) allow earlier stopping if needed
- Document EVERY experiment — even failures teach

### Experiment Velocity Benchmarks

| Company Stage | Experiments per Month | Notes |
|---|---|---|
| Pre-PMF (<50 users) | 2-4 | Big bets, qualitative validation |
| Early growth (50-1K) | 4-8 | Mix of big and small |
| Growth (1K-10K) | 8-15 | Data-driven, statistical rigor |
| Scale (10K+) | 15-30+ | Micro-optimizations compound |

---

## Phase 5: Growth Scoring & Health Dashboard

### Weekly Growth Health Score (0-100)

Score each dimension 0-20:

**1. Acquisition Health (0-20)**
- 20: CAC declining, volume increasing, 3+ channels working
- 15: CAC stable, volume growing, 2 channels working
- 10: CAC stable, volume flat, 1 channel working
- 5: CAC rising or volume declining
- 0: No systematic acquisition

**2. Activation Health (0-20)**
- 20: >60% reach aha moment, improving trend
- 15: 40-60% activation, stable
- 10: 20-40% activation
- 5: <20% activation
- 0: Aha moment undefined or unmeasured

**3. Retention Health (0-20)**
- 20: Cohort curve flattens >40%, NRR >120%
- 15: Flattens >25%, NRR >100%
- 10: Flattens >15%, NRR 90-100%
- 5: Flattens <15%
- 0: Curve trends to zero

**4. Revenue Health (0-20)**
- 20: LTV:CAC >5:1, payback <6mo, expansion revenue >30% of new
- 15: LTV:CAC >3:1, payback <12mo
- 10: LTV:CAC 2-3:1
- 5: LTV:CAC 1-2:1
- 0: Unit economics negative

**5. Experimentation Health (0-20)**
- 20: >10 experiments/month, documented learnings, velocity increasing
- 15: 5-10 experiments/month, mostly documented
- 10: 2-4 experiments/month
- 5: <2 experiments/month or undocumented
- 0: No systematic experimentation

**Total: ___/100**
- 80-100: Growth machine — optimize and scale
- 60-79: Solid foundation — fix weakest dimension
- 40-59: Growth fundamentals incomplete — focus on basics
- 20-39: Pre-growth — product/market fit work needed
- 0-19: No growth system — start from Phase 1

### Weekly Growth Dashboard YAML

```yaml
growth_dashboard:
  week_of: "YYYY-MM-DD"
  north_star:
    metric: ""
    current: 0
    previous_week: 0
    wow_change: "0%"
    target: 0
    on_track: true|false
  acquisition:
    new_signups: 0
    by_channel:
      organic: 0
      paid: 0
      referral: 0
      direct: 0
    total_cac: "$0"
    cac_by_channel: {}
  activation:
    signup_to_aha_rate: "0%"
    median_time_to_aha: ""
    onboarding_completion: "0%"
  retention:
    week1_retention: "0%"
    week4_retention: "0%"
    week12_retention: "0%"
    dau_mau_ratio: 0
  revenue:
    mrr: "$0"
    mrr_growth: "0%"
    arpu: "$0"
    ltv: "$0"
    nrr: "0%"
    free_to_paid_conversion: "0%"
  referral:
    viral_coefficient_k: 0
    referral_invites_sent: 0
    referral_conversion: "0%"
  experiments:
    running: 0
    completed_this_week: 0
    wins_this_week: 0
    win_rate_last_30_days: "0%"
  health_score: 0
  top_priority: "[What to fix this week]"
  blockers: []
```

---

## Phase 6: Growth Playbooks by Stage

### Pre-PMF (0-50 Users)

**Goal:** Find product-market fit. Growth spending = waste.

- Talk to 20+ potential users (interviews, not surveys)
- Build MVP that solves ONE problem for ONE persona
- Get 5 users who would be "very disappointed" without your product (Sean Ellis test)
- Manually onboard every user — learn what confuses them
- Don't optimize funnels. Don't run ads. Don't build referral programs.
- **Signal you're ready for growth:** 40%+ "very disappointed" AND retention curve flattens

### Early Growth (50-500 Users)

**Goal:** Find 1-2 scalable channels. Prove unit economics.

- Double down on whatever got your first 50 users
- Test 3 acquisition channels with small budgets ($500 each)
- Build onboarding that gets 40%+ to aha moment without manual help
- Start measuring AARRR weekly
- Implement basic referral mechanic (even just "invite a friend" link)
- **Signal you're ready to scale:** One channel produces users at <1/3 LTV CAC

### Growth (500-5,000 Users)

**Goal:** Scale proven channels. Build growth loops.

- Increase spend on winning channels (20% increments)
- Build content engine (SEO pillar + supporting content)
- Launch formal referral program with incentives
- Run 5-10 experiments per month
- Hire first growth-focused role (or allocate 50%+ of your time)
- Build retention loops (email sequences, notifications, habit features)
- **Signal you're ready to scale:** Multiple channels working, NRR >100%

### Scale (5,000+ Users)

**Goal:** Efficiency at volume. Compound loops.

- Diversify to 5+ acquisition channels
- Build growth team (analyst, engineer, marketer minimum)
- Automate experiment pipeline (feature flags, A/B framework)
- Focus on micro-optimizations (1% improvements compound)
- Build second-order growth loops (content → SEO → signups → content)
- International expansion if applicable
- Develop partnerships and channel/integration strategy

---

## Phase 7: Advanced Growth Tactics

### Pricing as a Growth Lever

Pricing changes are the highest-ROI growth tactic — they require zero traffic increase.

**Quick tests:**
- Raise price 20% for new users → measure conversion rate change
- Add annual plan with 2-month discount → measure plan mix shift
- Add usage-based component → measure expansion revenue
- Remove cheapest plan → measure conversion to next tier
- Add enterprise tier with "Contact us" → measure inbound

**1% price increase = 11% profit increase** (on average, across industries)

### Product-Led Growth (PLG) Framework

```yaml
plg_checklist:
  self_serve_signup: true|false
  time_to_value: "[< 5 minutes ideal]"
  free_tier_or_trial: "freemium|free_trial|both|neither"
  in_product_upsell: true|false
  usage_limits_as_upgrade_triggers: true|false
  team_invite_built_in: true|false
  public_api_or_integrations: true|false
  community_or_forum: true|false
  product_qualified_leads_defined: true|false
  expansion_revenue_automated: true|false
```

### Network Effects Playbook

**Types of network effects:**
1. **Direct** — more users = more value (social networks, messaging)
2. **Indirect/Cross-side** — more supply = more demand value (marketplaces)
3. **Data** — more usage = better product (ML, recommendations)
4. **Platform** — more developers = more apps = more users (iOS, Shopify)

**Building network effects:**
- Start with the "hard side" of the market (supply for marketplaces, creators for platforms)
- Seed with curated content/supply before opening up
- Build switching costs through data, relationships, integrations
- Create local network effects first (geographic, community, niche)

### Expansion Revenue Playbook

Expansion > new revenue (cheaper, higher close rate, compounds).

**Expansion signals to track:**
- Usage approaching plan limit (trigger upsell)
- Team size growing (trigger seat expansion)
- New use case adoption (trigger cross-sell)
- Power user behavior (trigger premium feature pitch)
- Account requesting features in higher tier (trigger upgrade conversation)

**Expansion tactics:**
1. Usage-based pricing with natural expansion (Twilio model)
2. Feature gating by plan tier with in-app upgrade prompts
3. Seat-based with team invite friction removal
4. Success milestones → celebration + "unlock more" offer
5. QBR (Quarterly Business Review) with ROI data + expansion recommendation

---

## Phase 8: Common Growth Mistakes (Avoid These)

### The 10 Growth Killers

1. **Scaling before PMF** — Pouring gasoline on a broken engine. Fix retention first.
2. **Too many channels** — 5 half-tested channels < 1 proven channel scaled hard.
3. **Vanity metrics** — Signups, pageviews, followers mean nothing without activation/revenue.
4. **No measurement** — "I think it's working" isn't growth. Instrument everything.
5. **Premature optimization** — A/B testing button colors when onboarding is 10% completion.
6. **Ignoring retention** — Acquisition is glamorous. Retention is profitable. Fix the bucket.
7. **Copying competitors** — Their strategy fits their context. Understand principles, not tactics.
8. **No experiment discipline** — Running tests without hypotheses, sample sizes, or documentation.
9. **Discounting as growth** — Discounts attract price-sensitive users who churn. Build value instead.
10. **Feature-as-growth** — "If we just build X, growth will come." Features don't acquire users.

### Diagnostic: Why Growth Stalled

| Symptom | Root Cause | Fix |
|---|---|---|
| Traffic up, signups flat | Landing page / messaging problem | A/B test headlines, social proof, CTA |
| Signups up, activation flat | Onboarding broken or aha moment unclear | Map and fix first-run experience |
| Activation up, retention flat | Product value is one-time, not recurring | Build habit loops, recurring value |
| Retention up, revenue flat | Monetization timing or pricing wrong | Test pricing, add expansion paths |
| Revenue up, growth slowing | Channel saturation | Diversify channels, build new loops |
| Everything flat | PMF lost or market shifted | Back to user interviews |

---

## Phase 9: Growth Team Design

### Solo Founder Growth Stack

Do these yourself, in this order:
1. Weekly user interviews (30 min each, 3 per week)
2. One content piece per week (SEO-optimized)
3. Basic email sequences (welcome, activation, re-engagement)
4. Monthly experiment (one real A/B test)
5. Weekly dashboard review (30 min)

### First Growth Hire

**Hire when:** You've found 1 working channel but can't scale it alone.

**Profile:** T-shaped — deep in one channel (paid, content, or product) + broad understanding of full funnel. Must be data-comfortable.

**Don't hire:** A "growth hacker" who promises 10x with tricks. Hire someone who can build systems.

### Growth Team at Scale

```
Head of Growth
├── Growth Engineering (build experiments, instrumentation)
├── Growth Marketing (channels, content, campaigns)
├── Growth Analytics (measurement, dashboards, insights)
└── Growth Product (onboarding, activation, monetization)
```

---

## Phase 10: Templates & Quick-Start Commands

### Natural Language Commands

Use these to interact with this skill:

1. **"Audit my growth"** → Run full AARRR assessment, identify bottleneck, create action plan
2. **"Score my growth health"** → Calculate 0-100 health score across 5 dimensions
3. **"Design a growth loop for [business type]"** → Select and design optimal loop
4. **"Plan an experiment for [metric]"** → Create full experiment YAML with hypothesis, sample size, duration
5. **"Diagnose why [metric] stalled"** → Root cause analysis with fix recommendations
6. **"Build my referral program"** → Design double-sided referral with mechanics, timing, tracking
7. **"Create my weekly dashboard"** → Generate growth dashboard YAML customized for your business
8. **"Evaluate [channel]"** → Score acquisition channel with ICE, estimate ROI, create test plan
9. **"Design my pricing for growth"** → Select pricing model, tier structure, expansion mechanics
10. **"What should I focus on?"** → Based on current metrics, identify single highest-leverage action
11. **"Build my content growth engine"** → Keyword clusters, content calendar, distribution plan
12. **"Calculate my unit economics"** → LTV, CAC, payback, LTV:CAC with health assessment

---

## Edge Cases & Advanced Situations

### B2B vs B2C Growth Differences

| Dimension | B2C | B2B |
|---|---|---|
| Decision maker | Individual | Committee (3-7 people) |
| Sales cycle | Minutes to days | Weeks to months |
| CAC | $1-50 | $100-10,000+ |
| Primary channels | Paid, viral, content | Content, outbound, events |
| Retention metric | DAU/MAU | Monthly active accounts |
| Expansion | Upsell features | Add seats, departments |
| Key growth lever | Virality + activation | Content + sales efficiency |

### Marketplace Growth (Two-Sided)

**The chicken-and-egg problem:**
1. Pick one side to subsidize (usually supply)
2. Start hyper-local or hyper-niche (Uber = SF, Airbnb = events)
3. Manually fill supply initially (founders do the work)
4. Build tools that make supply side's life better (even without demand)
5. Measure liquidity: % of searches that result in transaction

### International Growth

**Expansion decision framework:**
- Market size > $10M opportunity? (or strategic importance)
- Product works without localization? Test with English first.
- Legal/regulatory barriers? Research BEFORE building.
- Local competitors? If dominant, need 10x differentiation.
- Support coverage? Need timezone-appropriate support.

**Localization priority:**
1. Currency and pricing (mandatory)
2. Language (high impact)
3. Payment methods (region-specific)
4. Content/marketing (local references)
5. Support (native speakers)

### Growth for Developer Tools

- Documentation IS your growth engine
- Free tier should be genuinely useful (not crippled)
- API-first: let developers build on you
- Community (Discord, GitHub, forums) > traditional marketing
- Measure: API calls, integrations built, docs traffic
- Content: tutorials, use cases, comparisons, migration guides

### Zero-Budget Growth

When you can't spend money on acquisition:
1. **SEO + content** — write what your audience searches for
2. **Community participation** — be helpful in forums, Reddit, HN, Discord
3. **Product virality** — build sharing into the product experience
4. **Partnerships** — find complementary products, cross-promote
5. **Cold outreach** — personal emails to ideal customers (10/day, personalized)
6. **Launch platforms** — Product Hunt, HN Show, Indie Hackers, Reddit
7. **Integration marketplaces** — Shopify, Slack, Zapier app stores
