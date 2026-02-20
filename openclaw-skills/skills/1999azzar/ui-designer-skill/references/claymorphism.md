# Claymorphism Design Principles

Claymorphism is a friendly, 3D-inspired design style characterized by soft, "clay-like" textures. It feels tactile, inflated, and works exceptionally well with pastel color schemes.

## Core Concepts

### 1. Double Inner Shadows
- The "inflated" look is achieved using two inner shadows: one light (top-left) and one dark (bottom-right).
- Example: `inner-shadow-light` and `inner-shadow-dark`.

### 2. Large Border Radius
- Use extremely rounded corners (`rounded-3xl` or `rounded-[40px]`) to emphasize the soft, doughy feel.

### 3. Background Blur & Transparency
- Often combined with a subtle backdrop blur to keep the UI feeling "light" despite the 3D volume.

### 4. Floating Elements
- Use soft, large, blurred drop shadows to make elements feel like they are floating above the background.

## Tailwind CSS Implementation

```html
<div class="bg-blue-200 rounded-[40px] p-10 shadow-2xl 
            shadow-[inset_-8px_-8px_16px_rgba(0,0,0,0.1),inset_8px_8px_16px_rgba(255,255,255,0.8)]">
  <h2 class="text-3xl font-bold text-blue-900 mb-2">Soft & Tactile</h2>
  <p class="text-blue-800">Claymorphism brings a playful, 3D aesthetic to your interfaces.</p>
</div>
```

## Usage in Skill
When asked for Claymorphism:
- Suggest pastel backgrounds (`bg-pink-100`, `bg-indigo-100`).
- Enforce `rounded-[40px]`.
- Use complex `box-shadow` configurations for the inner-glow and outer-float effects.
- Keep the overall vibe playful and high-end.
