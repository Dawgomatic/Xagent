---
name: Playwright
description: Write, debug, and maintain Playwright tests and scrapers with resilient selectors, flaky test fixes, and CI/CD integration.
---

## Trigger

Use when writing Playwright tests, debugging failures, scraping with Playwright, or setting up CI/CD pipelines.

## Selector Priority (Always)

1. `getByRole()` — accessible, resilient
2. `getByTestId()` — explicit, stable
3. `getByLabel()`, `getByPlaceholder()` — form elements
4. `getByText()` — visible content (exact match preferred)
5. CSS/XPath — last resort, avoid nth-child and generated classes

## Core Capabilities

| Task | Reference |
|------|-----------|
| Test scaffolding & POMs | `testing.md` |
| Selector strategies | `selectors.md` |
| Scraping patterns | `scraping.md` |
| CI/CD integration | `ci-cd.md` |
| Debugging failures | `debugging.md` |

## Critical Rules

- **Never use `page.waitForTimeout()`** — use `waitFor*` methods or `expect` with polling
- **Always close contexts** — `browser.close()` or `context.close()` to prevent memory leaks
- **Prefer `networkidle` with caution** — SPAs may never reach idle; use DOM-based waits instead
- **Use `test.describe.configure({ mode: 'parallel' })`** — for independent tests
- **Trace on failure only** — `trace: 'on-first-retry'` in config, not always-on

## Quick Fixes

| Symptom | Fix |
|---------|-----|
| Element not found | Use `waitFor()` before interaction, check frame context |
| Flaky clicks | Use `click({ force: true })` or `waitFor({ state: 'visible' })` first |
| Timeout in CI | Increase timeout, add `expect.poll()`, check viewport size |
| Stale element | Re-query the locator, avoid storing element references |
| Auth lost between tests | Use `storageState` to persist cookies/localStorage |
