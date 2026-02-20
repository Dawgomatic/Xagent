# Neo-Brutalism Design Principles

Neo-Brutalism is a bold, raw, and high-contrast evolution of traditional brutalism, optimized for the digital age. It rejects soft shadows and subtle gradients in favor of sharp lines and vibrant, clashing colors.

## Core Concepts

### 1. High Contrast & Hard Shadows
- Use thick, pitch-black borders (`border-4 border-black`).
- Avoid blurred shadows. Use hard, offset shadows for depth (`shadow-[8px_8px_0px_0px_rgba(0,0,0,1)]`).

### 2. Vibrant & Unapologetic Palettes
- Combine high-saturation colors: Electric Yellow, Hot Pink, Cyber Blue, and Neon Green.
- Use these colors as "loud" backgrounds for cards or buttons.

### 3. Typography as an Element
- Large, bold, and often "ugly" or unconventional fonts.
- Use all-caps or oversized tracking for emphasis.

### 4. Raw Layouts
- Elements often overlap or feel slightly "misaligned."
- Grid lines can be visible and thick.

## Tailwind CSS Implementation

```html
<div class="bg-yellow-400 border-4 border-black p-6 shadow-[8px_8px_0px_0px_rgba(0,0,0,1)] hover:translate-x-[2px] hover:translate-y-[2px] hover:shadow-[4px_4px_0px_0px_rgba(0,0,0,1)] transition-all">
  <h2 class="text-3xl font-black uppercase mb-4">Brutal Impact</h2>
  <p class="text-lg font-bold">No soft edges here. Just raw energy and high visibility.</p>
</div>
```

## Usage in Skill
When asked for Neo-Brutalism:
- Suggest bold colors like `bg-lime-400` or `bg-violet-500`.
- Enforce `border-black` and `border-2` or `border-4`.
- Use `font-black` or `font-extrabold`.
- Use hard-offset shadows.
