# Scraping Patterns

## Basic Extraction

```typescript
const browser = await chromium.launch();
const page = await browser.newPage();
await page.goto('https://example.com/products');

const products = await page.$$eval('.product-card', cards =>
  cards.map(card => ({
    name: card.querySelector('.name')?.textContent?.trim(),
    price: card.querySelector('.price')?.textContent?.trim(),
    url: card.querySelector('a')?.href,
  }))
);

await browser.close();
```

## Wait Strategies for SPAs

```typescript
// Wait for specific element
await page.waitForSelector('[data-loaded="true"]');

// Wait for network idle (careful with SPAs)
await page.goto(url, { waitUntil: 'networkidle' });

// Wait for loading indicator to disappear
await page.waitForSelector('.loading-spinner', { state: 'hidden' });

// Custom condition with polling
await expect.poll(async () => {
  return await page.locator('.product').count();
}).toBeGreaterThan(0);
```

## Infinite Scroll

```typescript
async function scrollToBottom(page: Page) {
  let previousHeight = 0;
  
  while (true) {
    const currentHeight = await page.evaluate(() => document.body.scrollHeight);
    if (currentHeight === previousHeight) break;
    
    previousHeight = currentHeight;
    await page.evaluate(() => window.scrollTo(0, document.body.scrollHeight));
    await page.waitForTimeout(1000);  // Allow content to load
  }
}
```

## Pagination

```typescript
// Click-based pagination
async function scrapeAllPages(page: Page) {
  const allData = [];
  
  while (true) {
    const pageData = await extractData(page);
    allData.push(...pageData);
    
    const nextButton = page.getByRole('button', { name: 'Next' });
    if (await nextButton.isDisabled()) break;
    
    await nextButton.click();
    await page.waitForLoadState('networkidle');
  }
  
  return allData;
}
```

## Anti-Bot Evasion

```typescript
const browser = await chromium.launch({
  headless: false,  // Some sites detect headless
});

const context = await browser.newContext({
  userAgent: 'Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36',
  viewport: { width: 1920, height: 1080 },
  locale: 'en-US',
  timezoneId: 'America/New_York',
});

// Add realistic behavior
await page.mouse.move(100, 100);
await page.waitForTimeout(Math.random() * 2000 + 1000);
```

## Session Management

```typescript
// Save cookies
await context.storageState({ path: 'session.json' });

// Restore session
const context = await browser.newContext({
  storageState: 'session.json',
});
```

## Error Handling

```typescript
async function scrapeWithRetry(url: string, retries = 3) {
  for (let i = 0; i < retries; i++) {
    try {
      const page = await context.newPage();
      await page.goto(url, { timeout: 30000 });
      return await extractData(page);
    } catch (error) {
      if (i === retries - 1) throw error;
      await new Promise(r => setTimeout(r, 2000 * (i + 1)));
    } finally {
      await page.close();
    }
  }
}
```

## Rate Limiting

```typescript
class RateLimiter {
  private lastRequest = 0;
  
  constructor(private minDelay: number) {}
  
  async wait() {
    const elapsed = Date.now() - this.lastRequest;
    if (elapsed < this.minDelay) {
      await new Promise(r => setTimeout(r, this.minDelay - elapsed));
    }
    this.lastRequest = Date.now();
  }
}

const limiter = new RateLimiter(2000);  // 2s between requests

for (const url of urls) {
  await limiter.wait();
  await scrape(url);
}
```

## Proxy Rotation

```typescript
const proxies = ['proxy1:8080', 'proxy2:8080', 'proxy3:8080'];
let proxyIndex = 0;

async function getNextProxy() {
  const proxy = proxies[proxyIndex];
  proxyIndex = (proxyIndex + 1) % proxies.length;
  return proxy;
}

const browser = await chromium.launch({
  proxy: { server: await getNextProxy() },
});
```
