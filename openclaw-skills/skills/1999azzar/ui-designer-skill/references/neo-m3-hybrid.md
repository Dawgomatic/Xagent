# Neo-M3 Hybrid Design Language

A sophisticated blend of Neo-Brutalism's raw energy and Material You's user-centric softness. Inspired by tech journalism giants like **Wired** and **The Verge**.

## Core Principles

### 1. Structure: Industrial Softness
- **Borders:** 3px to 4px solid black (#000000). Use dashed borders for experimental or "Edan" elements.
- **Corners:** Large rounded corners (24px to 32px / `rounded-3xl`) on main containers and cards.
- **Shadows:** Hard, non-blurry offset shadows (6px to 10px). Vibe: "Physical items on a grid."

### 2. Typography: The "Wired" Impact
- **Headlines:** Sans-serif with heavy weights (800-900). Tight letter spacing.
- **Font Choice:** Plus Jakarta Sans or Lexend Mega.
- **Monospace Accents:** Use JetBrains Mono for system stats, timestamps, and metadata to give an "engineering terminal" feel.

### 3. Color: Tonal Pastels & High Contrast
- **Base Background:** Neutral Slate-50 (#f8fafc) or Light Grey (#f0f0f0).
- **Accents:** Use a "Verge" inspired tonal palette:
  - Lavender (#e9d5ff)
  - Sky Blue (#dbeafe)
  - Rose (#fce7f3)
  - Mint (#dcfce7)
- **High Contrast:** Text is always pure black (#000000) on cards, or white on pure black footers/marquees.

### 4. Elements
- **Marquees:** Use top-edge scrolling marquees for system status.
- **Navigation:** Floating pills with high-contrast active states.
- **Labels:** Small, inverted background labels (white text on black) for tags/categories.

## Sample Tailwind Config Extension

```javascript
{
  theme: {
    extend: {
      colors: {
        'neo-bg': '#f8fafc',
        'neo-accent': '#6366f1',
        'neo-pink': '#ff00ff',
      },
      boxShadow: {
        'brutal': '8px 8px 0px 0px rgba(0,0,0,1)',
      }
    }
  }
}
```
