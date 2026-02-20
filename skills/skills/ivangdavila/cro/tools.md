# CRO Tools & Integrations

## Analytics Platforms

### Google Analytics 4 (GA4)
**Use for:** Funnel analysis, drop-off rates, traffic source performance
- Conversion paths and attribution
- Segment comparison (device, source, geography)
- Export to BigQuery for advanced analysis

**Key reports:**
- Funnel exploration (custom funnels)
- Path exploration (user journeys)
- Cohort analysis (retention)

### Mixpanel
**Use for:** Event-based funnels, retention, feature adoption
- More flexible than GA4 for product analytics
- Better for SaaS conversion tracking
- Real-time insights

### Amplitude
**Use for:** Behavioral cohorts, user paths, experiment analysis
- Strong for identifying high-converting behaviors
- Path finder for optimization opportunities
- Built-in experimentation analysis

## A/B Testing Tools

### VWO (Visual Website Optimizer)
**Use for:** Full-stack testing with visual editor
- Create variants without code
- Bayesian statistics engine
- Heatmaps and session recordings included

**When to use:** Marketing teams, landing page tests, low-code needs

### Optimizely
**Use for:** Enterprise testing, feature flags, personalization
- Server-side and client-side testing
- Stats accelerator for faster results
- Multi-page experiments

**When to use:** Large traffic, complex tests, product experiments

### LaunchDarkly
**Use for:** Feature flags with targeting
- Progressive rollouts
- A/B testing at code level
- Kill switches for quick rollback

**When to use:** Product teams, feature releases, backend experiments

## Heatmaps & Session Recording

### Hotjar
**Use for:** Visual understanding of user behavior
- Click, scroll, and move heatmaps
- Session recordings
- Form analytics
- Feedback polls

**Best for:** Qualitative insights, hypothesis generation

### FullStory
**Use for:** Frustration detection, rage clicks, error analysis
- Auto-captures all interactions
- Search sessions by behavior
- Frustration scoring

**Best for:** Identifying UX problems at scale

### Microsoft Clarity
**Use for:** Free alternative to Hotjar
- Unlimited recordings
- Rage click detection
- Integration with GA4

**Best for:** Budget-conscious, good-enough insights

## Form Analytics

### Zuko (formerly Formisimo)
**Use for:** Field-level conversion analysis
- Time per field
- Abandonment by field
- Error rates
- Comparison between form versions

### Hotjar Forms
**Use for:** Basic form analytics
- Abandonment rates
- Field interaction times
- Part of Hotjar suite

## Speed Testing

### Google PageSpeed Insights
**Use for:** Core Web Vitals, quick diagnostics
- Lab and field data
- Specific recommendations
- API for automation

### WebPageTest
**Use for:** Detailed waterfall analysis
- Multiple locations and devices
- Video comparison
- Custom scripting

### Lighthouse (Chrome DevTools)
**Use for:** Local testing, CI integration
- Performance scoring
- Accessibility checks
- SEO basics

## Tag Management

### Google Tag Manager (GTM)
**Use for:** Deploying tracking without code changes
- Conversion pixels
- Event tracking
- A/B test snippets

### Segment
**Use for:** Single source of truth for events
- Send data to multiple tools
- Clean event taxonomy
- Server-side tracking

## Recommended Stack by Size

### Startup (<10k monthly visitors)
- GA4 (free)
- Microsoft Clarity (free)
- Google Optimize successor or VWO (free tier)
- PageSpeed Insights

### Growth (10k-100k visitors)
- GA4 + Mixpanel
- Hotjar
- VWO or AB Tasty
- GTM

### Scale (100k+ visitors)
- Amplitude or Mixpanel
- FullStory
- Optimizely
- Segment
- Custom dashboards

## Integration Patterns

### Event Tracking Standard
Use consistent naming:
```
[Object]_[Action]
form_started
form_submitted
checkout_initiated
purchase_completed
```

### Passing Experiment Data
Send variant info to analytics:
```javascript
dataLayer.push({
  event: 'experiment_viewed',
  experiment_id: 'pricing_test_v2',
  variant: 'annual_discount'
});
```

### Connecting Tools
1. A/B tool → Analytics (for deeper analysis)
2. Heatmap tool → A/B tool (for variant insights)
3. Form tool → Analytics (for funnel context)
4. Speed tool → Alerting (for regressions)
