import os
import argparse

def main():
    parser = argparse.ArgumentParser(description="Apply UI design rules to .cursorrules")
    parser.add_argument("--style", choices=["material", "minimal", "glass", "m3-pastel", "neo-brutalism", "claymorphism"], default="minimal", help="The design language to use.")
    parser.add_argument("--palette", choices=["pastel", "dark", "vibrant"], default="pastel", help="The color palette preference.")
    args = parser.parse_args()

    # Base rules that always apply
    base_rules = """
# AI Design Rules (Azzar's UI Designer)

## Core Philosophy
- "Wong edan mah ajaib" (Crazy people are magical).
- Just enough engineering to get it done well. No bloat.
- Mobile-first, accessible, and responsive by default.
"""

    style_rules = {
        "material": """- Use Material You (M3) principles.
- Large rounded corners (rounded-3xl).
- Tonal palettes and subtle elevation.""",
        "minimal": """- Focus on typography and whitespace.
- Limit borders and shadows.
- High contrast for readability.""",
        "glass": """- Use backdrop-blur-md/lg.
- Semi-transparent backgrounds (bg-white/20).
- Subtle white borders (border-white/10).""",
        "m3-pastel": """- **M3 Pastel Glass (Hybrid)**
- **Shape**: Large rounded corners (28px/rounded-3xl).
- **Interaction**: Morph to sharper corners (12px) on hover + Lift + Glow.
- **Palette**: Pastel Blue (#D1E4FF) & Deep Blue (#003258).
- **Glass**: Use `bg-white/5` + `backdrop-blur-2xl`.""",
        "neo-brutalism": """- Use thick black borders (border-4 border-black).
- Use hard, non-blurred shadows (shadow-[8px_8px_0px_0px_rgba(0,0,0,1)]).
- Use vibrant, high-saturation clashing colors.
- Bold, often all-caps typography.""",
        "claymorphism": """- Use soft 3D "inflated" shapes.
- Implement double inner shadows (one light, one dark).
- Use very large border radius (rounded-[40px]).
- Pair with pastel colors and floating drop shadows."""
    }

    palette_rules = {
        "pastel": "- Use soft, muted colors. Avoid harsh saturation.\n- Backgrounds should be very light or off-white.",
        "dark": "- Use deep grays/zinc for backgrounds.\n- High-contrast accents for visibility.",
        "vibrant": "- Use bold, energetic colors for CTAs.\n- Strong gradients where appropriate."
    }

    rules_to_add = f"""
{base_rules}
### Style: {args.style.capitalize()}
{style_rules[args.style]}

### Palette: {args.palette.capitalize()}
{palette_rules[args.palette]}

### Technical Stack
- Tailwind CSS
- Font Awesome (fa-duotone/fa-light)
- React / Next.js
"""

    rules_file = ".cursorrules"
    mode = "a" if os.path.exists(rules_file) else "w"
    
    with open(rules_file, mode) as f:
        if mode == "a":
            f.write("\n\n--- Updated by ui-designer ---\n")
        f.write(rules_to_add)
    
    print(f"✅ Successfully applied {args.style} style with {args.palette} palette to {rules_file}")

if __name__ == "__main__":
    main()
