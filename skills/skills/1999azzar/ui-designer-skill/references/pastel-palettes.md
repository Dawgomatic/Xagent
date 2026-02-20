# Pastel Color Palettes

Pastel colors are soft, muted, and easy on the eyes. Preferred by Azzar.

## Curated Palettes

### 1. Spring Morning (Soft & Fresh)
- Primary: `#B2E2F2` (Soft Blue)
- Secondary: `#F2C6D2` (Soft Pink)
- Accent: `#D9F2B6` (Soft Green)
- Background: `#F9F9F9`

### 2. Sunset Glow (Warm & Muted)
- Primary: `#FFD8B1` (Peach)
- Secondary: `#FFB7B2` (Coral)
- Accent: `#E2F0CB` (Lime)

### 3. Lavender Dreams (Calm & Elegant)
- Primary: `#E0BBE4` (Lavender)
- Secondary: `#957DAD` (Deep Lavender)
- Accent: `#D291BC` (Orchid)

## Tailwind CSS Configuration

```javascript
// tailwind.config.js
module.exports = {
  theme: {
    extend: {
      colors: {
        pastel: {
          blue: '#B2E2F2',
          pink: '#F2C6D2',
          green: '#D9F2B6',
          peach: '#FFD8B1',
          lavender: '#E0BBE4',
        }
      }
    }
  }
}
```

## Best Practices
- Use pastels for backgrounds and large surfaces.
- Use a slightly darker version of the pastel color for text or borders to ensure accessibility.
- Mix with high-quality whitespace (Minimalism).
