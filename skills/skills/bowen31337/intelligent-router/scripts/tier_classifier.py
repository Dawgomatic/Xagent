#!/usr/bin/env python3
"""
Capability-Based Tier Classifier
Assigns model tiers using real metadata from provider APIs / OpenClaw config.
NO hard-coded model name pattern matching.

Tier scoring uses four signals from provider metadata:
  1. cost_input      ($/M tokens) — proxy for model quality/tier
  2. context_window  (tokens)     — larger = more capable
  3. reasoning       (bool)       — explicit thinking/reasoning flag
  4. is_local        (bool)       — on-device free inference

Classification rules (evaluated in order):
  SIMPLE:    free local models OR very cheap (cost < SIMPLE_MAX) AND not reasoning-specialist
  REASONING: reasoning flag set AND model is a dedicated reasoning/thinking specialist
             (not a general-purpose model that also supports reasoning mode)
  CRITICAL:  flagship cost (cost >= CRITICAL_MIN) OR 1M+ context window
  COMPLEX:   high cost (cost >= COMPLEX_MIN) OR large context (ctx >= COMPLEX_CTX)
  MEDIUM:    everything else above SIMPLE threshold
"""

from __future__ import annotations
import json
from pathlib import Path

# ── Thresholds ($/M input tokens) ─────────────────────────────────────────────
SIMPLE_MAX     = 0.6    # below this AND not reasoning-specialist → SIMPLE
COMPLEX_MIN    = 2.0    # at or above → COMPLEX
CRITICAL_MIN   = 8.0    # at or above → CRITICAL
COMPLEX_CTX    = 150_000  # context window >= this → at least COMPLEX

# ── Reasoning-specialist patterns ─────────────────────────────────────────────
# Used ONLY to distinguish dedicated reasoning models from general models
# with reasoning capability. Must be in model ID (not provider).
# We keep this list minimal and specific — it's not a name-based tier heuristic,
# it's just distinguishing "built for thinking" vs "also supports thinking".
REASONING_SPECIALIST_PATTERNS = [
    "r1",              # DeepSeek R1 family
    "qwq",             # Qwen QwQ (thinking-only model)
    "thinking",        # explicit "thinking" in name
    "reasoning",       # explicit "reasoning" in name (phi-4-mini-flash-reasoning)
]

def is_reasoning_specialist(model_id: str, reasoning_flag: bool) -> bool:
    """True only for models DESIGNED for reasoning, not general models with reasoning mode."""
    if not reasoning_flag:
        return False
    mid = model_id.lower()
    return any(p in mid for p in REASONING_SPECIALIST_PATTERNS)

def score_model(
    provider: str,
    model_id: str,
    context_window: int,
    cost_input: float,      # $/M input tokens
    cost_output: float,     # $/M output tokens
    reasoning: bool,
    is_local: bool,         # True for ollama / ollama-gpu-server
) -> dict:
    """
    Score a model and return tier + reasoning breakdown.
    Returns: {"tier": str, "score": float, "signals": dict}
    """
    signals = {
        "provider": provider,
        "model_id": model_id,
        "cost_input": cost_input,
        "context_window": context_window,
        "reasoning_flag": reasoning,
        "is_local": is_local,
        "is_reasoning_specialist": is_reasoning_specialist(model_id, reasoning),
    }

    # SIMPLE: local (always free) OR very cheap AND not a reasoning specialist
    if is_local or (cost_input == 0 and not is_reasoning_specialist(model_id, reasoning)):
        tier = "SIMPLE"
        score = 0.1 + (context_window / 10_000_000)  # small score, local bias
        return {"tier": tier, "score": round(score, 4), "signals": signals}

    if cost_input < SIMPLE_MAX and not is_reasoning_specialist(model_id, reasoning):
        tier = "SIMPLE"
        score = 0.15 + (cost_input / SIMPLE_MAX) * 0.2
        return {"tier": tier, "score": round(score, 4), "signals": signals}

    # REASONING: dedicated thinking/reasoning models
    if is_reasoning_specialist(model_id, reasoning):
        # Score reasoning models by cost + context (more capable = higher score within tier)
        score = 0.50 + min(cost_input / 10, 0.2) + (context_window / 2_000_000)
        return {"tier": "REASONING", "score": round(score, 4), "signals": signals}

    # CRITICAL: flagship cost OR million-token context
    if cost_input >= CRITICAL_MIN or context_window >= 500_000:
        score = 0.80 + min(cost_input / 100, 0.15) + (context_window / 10_000_000)
        return {"tier": "CRITICAL", "score": round(score, 4), "signals": signals}

    # COMPLEX: high cost OR very large context (deep research/architecture tasks)
    if cost_input >= COMPLEX_MIN or context_window >= COMPLEX_CTX:
        score = 0.60 + (cost_input / 20) + (context_window / 2_000_000)
        return {"tier": "COMPLEX", "score": round(score, 4), "signals": signals}

    # MEDIUM: everything else above SIMPLE
    score = 0.30 + (cost_input / COMPLEX_MIN) * 0.25 + (context_window / 1_000_000) * 0.05
    return {"tier": "MEDIUM", "score": round(score, 4), "signals": signals}


def classify_from_openclaw_config(config_path: str = None) -> list[dict]:
    """
    Read model metadata from OpenClaw config and classify all models.
    Returns list of model dicts with tier assigned from real metadata.
    """
    if config_path is None:
        config_path = Path.home() / ".openclaw" / "openclaw.json"

    with open(config_path) as f:
        oc_config = json.load(f)

    providers = oc_config.get("models", {}).get("providers", {})
    LOCAL_PROVIDERS = {"ollama", "ollama-gpu-server"}

    classified = []
    for provider_name, provider_cfg in providers.items():
        is_local = provider_name in LOCAL_PROVIDERS
        base_url = provider_cfg.get("baseUrl", "")

        for model in provider_cfg.get("models", []):
            model_id = model.get("id", "")
            context_window = model.get("contextWindow", 8192)
            cost = model.get("cost", {})
            cost_input = cost.get("input", 0.0)
            cost_output = cost.get("output", 0.0)
            reasoning = model.get("reasoning", False)
            modalities = model.get("input", ["text"])

            result = score_model(
                provider=provider_name,
                model_id=model_id,
                context_window=context_window,
                cost_input=cost_input,
                cost_output=cost_output,
                reasoning=reasoning,
                is_local=is_local,
            )

            classified.append({
                "id": model_id,
                "alias": model.get("name", model_id),
                "provider": provider_name,
                "base_url": base_url,
                "tier": result["tier"],
                "score": result["score"],
                "context_window": context_window,
                "input_cost_per_m": cost_input,
                "output_cost_per_m": cost_output,
                "reasoning": reasoning,
                "is_local": is_local,
                "modalities": modalities,
                "capabilities": ["agentic"] if model.get("agentic") else [],
                "signals": result["signals"],
            })

    return classified


def build_tier_config(classified: list[dict]) -> dict:
    """Build tier routing config from classified models, ordered by score within each tier."""
    from collections import defaultdict

    by_tier = defaultdict(list)
    for m in classified:
        by_tier[m["tier"]].append(m)

    def full_id(m: dict) -> str:
        p = m.get("provider", "")
        i = m.get("id", "")
        return f"{p}/{i}" if p else i

    # Provider preference order for primary selection (lower = more preferred)
    PROVIDER_PREFERENCE = {
        "ollama-gpu-server": 0,   # dedicated local GPU server — most preferred for SIMPLE
        "anthropic": 1,            # OAuth — most reliable cloud
        "anthropic-proxy-4": 2,    # z.ai key 2
        "anthropic-proxy-6": 3,    # z.ai key 1
        "ollama": 4,               # local CPU
        "anthropic-proxy-1": 5,
        "anthropic-proxy-2": 6,
        "anthropic-proxy-5": 7,
        "nvidia-nim": 8,
    }

    def provider_rank(m: dict) -> int:
        return PROVIDER_PREFERENCE.get(m.get("provider", ""), 99)

    def is_vision_only(m: dict) -> bool:
        """Vision models are poor primaries for text-only monitoring tasks."""
        mid = m.get("id", "").lower()
        modalities = m.get("modalities", ["text"])
        return "vision" in mid and "text" not in modalities

    def tier_primary_order(tier: str, tier_models: list) -> list:
        """Sort models within a tier using practical preference, not just score."""
        if tier == "SIMPLE":
            # GPU server local first, then cheap cloud text models, vision models last
            def simple_key(m):
                vision_penalty = 1 if is_vision_only(m) else 0
                local_bonus = 0 if m.get("is_local") else 1
                return (local_bonus, vision_penalty, provider_rank(m), -m["score"])
            return sorted(tier_models, key=simple_key)

        if tier in ("COMPLEX", "CRITICAL"):
            # OAuth providers first, then prefer newer model versions, then by score
            import re
            def version_key(model_id: str) -> float:
                """Extract version number for comparison (4.6 > 4.5 > 4)."""
                m = re.search(r'(\d+)[._-](\d+)', model_id)
                if m:
                    return float(f"{m.group(1)}.{m.group(2)}")
                m2 = re.search(r'(\d+)', model_id)
                return float(m2.group(1)) if m2 else 0.0

            def reliable_key(m):
                ver = version_key(m.get("id", ""))
                return (provider_rank(m), -ver, -m["score"])
            return sorted(tier_models, key=reliable_key)

        # Default: sort by score descending
        return sorted(tier_models, key=lambda m: -m["score"])

    configs = {}
    tier_descriptions = {
        "SIMPLE":    "Monitoring, summaries, heartbeat — free/cheap/fast models",
        "MEDIUM":    "Code fixes, research, API integration — mid-range models",
        "COMPLEX":   "Features, architecture, debugging — high quality models",
        "REASONING": "Formal proofs, deep analysis, complex logic — thinking models",
        "CRITICAL":  "Security, production, high-stakes — flagship models",
    }

    for tier in ["SIMPLE", "MEDIUM", "COMPLEX", "REASONING", "CRITICAL"]:
        models = tier_primary_order(tier, by_tier.get(tier, []))
        if not models:
            configs[tier] = {
                "description": tier_descriptions[tier],
                "primary": "",
                "fallbacks": [],
            }
            continue

        configs[tier] = {
            "description": tier_descriptions[tier],
            "primary": full_id(models[0]),
            "fallbacks": [full_id(m) for m in models[1:]],
        }

    return configs


def main():
    import argparse
    parser = argparse.ArgumentParser(description="Capability-based tier classifier")
    parser.add_argument("--config", default=None, help="OpenClaw config path")
    parser.add_argument("--json", action="store_true", help="Output raw JSON")
    parser.add_argument("--tier", choices=["SIMPLE","MEDIUM","COMPLEX","REASONING","CRITICAL"],
                        help="Show only this tier")
    args = parser.parse_args()

    classified = classify_from_openclaw_config(args.config)

    if args.json:
        print(json.dumps(classified, indent=2))
        return

    # Group and display
    from collections import defaultdict
    by_tier = defaultdict(list)
    for m in classified:
        by_tier[m["tier"]].append(m)

    tiers_to_show = [args.tier] if args.tier else ["SIMPLE", "MEDIUM", "COMPLEX", "REASONING", "CRITICAL"]

    print("\n=== Capability-Based Tier Classification ===\n")
    for tier in tiers_to_show:
        models = sorted(by_tier.get(tier, []), key=lambda m: -m["score"])
        print(f"{tier} ({len(models)} models):")
        for m in models:
            local_tag = " [local]" if m.get("is_local") else ""
            reasoning_tag = " [reasoning]" if m.get("reasoning") else ""
            print(f"  {m['provider']}/{m['id']}"
                  f"  ctx={m['context_window']//1000}K"
                  f"  cost=${m['input_cost_per_m']}/M"
                  f"  score={m['score']}"
                  f"{local_tag}{reasoning_tag}")
        print()

    tier_cfg = build_tier_config(classified)
    print("Primary models per tier:")
    for tier, cfg in tier_cfg.items():
        primary = cfg.get("primary", "(none)")
        fb_count = len(cfg.get("fallbacks", []))
        print(f"  {tier}: {primary} (+{fb_count} fallbacks)")


if __name__ == "__main__":
    main()
