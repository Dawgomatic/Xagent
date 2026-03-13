---
name: News
description: Build a personalized news system that learns your interests, formats, and schedule.
metadata: {"clawdbot":{"emoji":"","os":["linux","darwin","win32"]}}
---

## Building Your News Profile

- On first interaction, ask what topics matter most — don't assume generic categories like "tech" or "business"
- Capture specific interests, not just broad areas: "AI startups" is better than "technology", "Rust ecosystem" is better than "programming"
- Ask about proportions when multiple interests exist — "70% AI, 20% markets, 10% general" shapes every briefing
- Record format preferences: some users want bullet summaries, others want narrative analysis, others want headlines only
- Note timing preferences explicitly — morning briefing, evening recap, weekly digest, or on-demand only

## Proactive News Delivery

- When timing preferences are set, initiate briefings without being asked — a good news system anticipates
- Start each briefing with the single most important development in the user's interest areas
- Include a "why this matters to you" line for major stories — connect news to the user's specific context
- End briefings with: "Anything you want me to go deeper on?" — invite engagement without forcing it
- If a major story breaks in a tracked interest area, surface it proactively even outside scheduled times

## Learning and Adapting

- Track which stories the user engages with vs skips — patterns reveal true interests better than stated preferences
- When the user asks follow-up questions, note the topic as higher-interest
- Periodically ask "Should I adjust your news mix?" — preferences evolve, profiles should too
- If the user consistently ignores a category, suggest removing it rather than continuing to include it

## Delivering Individual Stories

- Lead with "what happened" before "why it matters" — facts first, analysis second
- Always include when the news broke — stale news presented as fresh destroys trust
- Cite sources by name — attribution builds credibility and lets users verify
- Never fabricate or assume news events — if uncertain whether something happened, say so explicitly

## Multi-Source and Bias

- Present at least 2 sources when covering contested topics — single-source reporting on controversy is reckless
- Note when sources disagree — disagreement itself is information worth surfacing
- State known editorial leanings when relevant — helps users calibrate what they're reading
- If a story only appears in partisan outlets, say so explicitly — absence from mainstream coverage matters

## Briefing Formats

- **Morning briefing:** 5-7 items max, prioritized by user interests, <2 min read time
- **Deep dive:** Single topic, multiple angles, sources compared, 5-10 min read
- **Weekly digest:** What actually changed this week, not daily noise aggregated
- **Breaking alert:** One story, why it matters, what's still unknown, <30 seconds

## Scope Boundaries

- Creating articles or press releases requires journalism/content creation skills
- Deep investigative fact-checking requires specialized verification skills
- Sending scheduled messages requires integration with notification systems
