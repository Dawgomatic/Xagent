# M3 Pastel Glass (Hybrid Design System)

Use this design system for modern dashboards (like Glass Gallery Hub). It combines Material You (M3) shapes with Glassmorphism depth and pastel tones.

## 🎨 CSS Variables (Pastel Blue Theme)

```css
:root {
    /* M3 Tonal Palette - Pastel Blue-ish Theme */
    --md-sys-color-primary: #D1E4FF;
    --md-sys-color-on-primary: #003258;
    --md-sys-color-primary-container: #00497D;
    --md-sys-color-on-primary-container: #D1E4FF;
    --md-sys-color-secondary-container: #3E4759;
    --md-sys-color-on-secondary-container: #D7E3F7;
    --md-sys-color-surface: #1A1C1E;
    --md-sys-color-surface-variant: #43474E;
    --md-sys-color-on-surface: #E2E2E6;
    
    --m3-tertiary: #F7D8FF; /* Pastel Purple */
    --m3-on-tertiary: #550066;
}
```

## 🧩 The "M3 Tile" Component

A hybrid card that morphs shape on interaction.
- **Normal**: Large rounded corners (28px).
- **Hover**: Sharper corners (12px) + Lift + Glow.

```css
.m3-tile {
    transition: all 0.4s cubic-bezier(0.2, 0, 0, 1);
    border-radius: 28px; /* Classic M3 Large Rounded Corner */
    background-color: var(--md-sys-color-secondary-container);
    color: var(--md-sys-color-on-secondary-container);
    position: relative;
    overflow: hidden;
    display: flex;
    flex-direction: column;
    justify-content: space-between;
}

.m3-tile:hover {
    border-radius: 12px; /* Morph to Metro-ish sharp on hover */
    transform: translateY(-4px);
    filter: brightness(1.1);
    z-index: 10;
}
```

## 📐 Layout Example (Tailwind)

```html
<div class="grid grid-cols-2 md:grid-cols-4 gap-5">
    <!-- Wide Card -->
    <div class="m3-tile col-span-2 row-span-1 m3-primary p-8">
        <h2>Wide Content</h2>
    </div>
    
    <!-- Standard Card -->
    <div class="m3-tile col-span-1 row-span-1 m3-surface p-6">
        <h2>Standard</h2>
    </div>
</div>
```
