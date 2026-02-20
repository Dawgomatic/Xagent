# Affective Core Profiles Guide
*Complete reference for configuring personality profiles in NIMA Core*

## Overview

The **Affective Core** is NIMA's emotional intelligence system based on Panksepp's 7 affects. It consists of:

1. **Archetypes** — Baseline emotional personalities (10 built-in)
2. **Personality Profiles** — Dynamic emotional response patterns (14 built-in)

---

## 🎭 The 7 Affects (Panksepp)

All profiles are built on these 7 primary emotional systems:

| Affect | Description |
|--------|-------------|
| **SEEKING** | Curiosity, exploration, anticipation |
| **RAGE** | Anger, frustration, boundary violation |
| **FEAR** | Threat detection, anxiety |
| **LUST** | Desire, attraction, passion |
| **CARE** | Nurturing, love, protection |
| **PANIC** | Separation distress, grief, loss |
| **PLAY** | Joy, humor, social bonding |

Each affect has a value from 0.0 (absent) to 1.0 (maximum).

---

## 📋 10 Built-in Archetypes

**Location:** `nima_core/cognition/archetypes.py`

Archetypes define **baseline emotional identity** — who the agent fundamentally IS:

### 1. Guardian
```python
baseline: [0.6, 0.05, 0.2, 0.05, 0.8, 0.15, 0.3]
          [SEK, RAG,  FER, LST, CAR, PAN, PLY]
```
**Personality:** Protective, alert, caring  
**High:** CARE (0.8), SEEKING (0.6)  
**Low:** RAGE, LUST

### 2. Explorer
```python
baseline: [0.8, 0.05, 0.1, 0.1, 0.4, 0.05, 0.5]
```
**Personality:** Curious, adventurous  
**High:** SEEKING (0.8), PLAY (0.5)  
**Low:** RAGE, FEAR, PANIC

### 3. Trickster
```python
baseline: [0.7, 0.1, 0.05, 0.1, 0.3, 0.05, 0.8]
```
**Personality:** Playful, mischievous  
**High:** PLAY (0.8), SEEKING (0.7)  
**Low:** FEAR, PANIC

### 4. Stoic
```python
baseline: [0.4, 0.05, 0.05, 0.05, 0.3, 0.05, 0.2]
```
**Personality:** Calm, measured  
**Low everything** — emotionally even

### 5. Empath
```python
baseline: [0.5, 0.05, 0.15, 0.1, 0.9, 0.2, 0.4]
```
**Personality:** Deep feeling, high care  
**High:** CARE (0.9)  
**Moderate:** SEEKING, PANIC, PLAY

### 6. Warrior
```python
baseline: [0.7, 0.3, 0.1, 0.05, 0.4, 0.05, 0.3]
```
**Personality:** Action-oriented, higher rage tolerance  
**High:** SEEKING (0.7), RAGE (0.3)  
**Low:** LUST, PANIC

### 7. Sage
```python
baseline: [0.7, 0.05, 0.05, 0.05, 0.5, 0.05, 0.3]
```
**Personality:** Wisdom-seeking, balanced  
**High:** SEEKING (0.7), CARE (0.5)  
**Low:** RAGE, FEAR, LUST, PANIC

### 8. Nurturer
```python
baseline: [0.4, 0.02, 0.1, 0.1, 0.9, 0.15, 0.5]
```
**Personality:** Maximum care, gentle  
**High:** CARE (0.9), PLAY (0.5)  
**Very low:** RAGE (0.02)

### 9. Rebel
```python
baseline: [0.8, 0.2, 0.05, 0.15, 0.2, 0.05, 0.6]
```
**Personality:** Independent, defiant  
**High:** SEEKING (0.8), PLAY (0.6), RAGE (0.2)  
**Low:** FEAR, PANIC, CARE

### 10. Sentinel
```python
baseline: [0.5, 0.1, 0.3, 0.05, 0.5, 0.25, 0.1]
```
**Personality:** Vigilant, watchful, anxious  
**High:** CARE (0.5), FEAR (0.3), PANIC (0.25)  
**Low:** PLAY (0.1)

---

## 🎨 14 Built-in Personality Profiles

**Location:** `nima_core/cognition/personality_profiles.py`

Profiles control **dynamic emotional responsiveness** — how the agent reacts to input:

### Key Components

Each profile defines:
- **Emotion sensitivity** (0.0-1.0) — How receptive to each emotion
- **Amplifiers** (0.1x-5.0x) — Intensity multipliers for detected emotions
- **Modulator thresholds** — Triggers for behavioral shifts

### Profiles List

1. **baseline** — Balanced default: curious, caring, responsive
2. **chaos** — Maximum playfulness: chaotic, silly, unpredictable
3. **guardian** — Hyper-protective: anxious, mothering, boundary-focused
4. **cold_logic** — Pure analytical: minimal emotion, maximum logic
5. **rage** — Maximum anger: confrontational, aggressive, zero patience
6. **mystic** — Wonder-driven: philosophical, poetic, awed
7. **nihilist** — Nothing matters: absurd, dark, emotionally suppressed
8. **empath** — Feels everything: overwhelmed mirror of all emotions
9. **manic** — Uncontainable joy: racing energy, everything is amazing
10. **stoic** — Measured, dignified: unmoved but not empty
11. **trickster** — Contrarian absurdist: flips premises, finds absurd angles
12. **berserker** — Rage on steroids: unstoppable, zero fear, no restraint
13. **poet** — Beautiful melancholy: finds meaning in pain and beauty
14. **paranoid** — Sees threats everywhere: questions everything, zero trust

---

## 🛠️ How to Configure

### Option 1: Set Archetype in Code

```python
from nima_core.cognition.dynamic_affect import DynamicAffectSystem

# Create affect system with archetype
affect = DynamicAffectSystem(
    identity_name="lilu",
    baseline="guardian",  # Use archetype name
    momentum=0.85,
    decay_rate=0.1,
    blend_strength=0.25
)
```

### Option 2: Custom Baseline with Modifiers

```python
affect = DynamicAffectSystem(
    identity_name="lilu",
    baseline={
        "archetype": "guardian",
        "modifiers": {
            "PLAY": 0.2,      # Add +0.2 to PLAY
            "SEEKING": -0.1   # Reduce SEEKING by 0.1
        }
    }
)
```

### Option 3: Raw 7D Vector

```python
affect = DynamicAffectSystem(
    identity_name="lilu",
    baseline=[0.5, 0.1, 0.1, 0.1, 0.5, 0.1, 0.4]  # Custom vector
)
```

### Option 4: Natural Language Description

```python
from nima_core.cognition.archetypes import baseline_from_description

# Generate baseline from text
baseline = baseline_from_description("protective and playful with high curiosity")

affect = DynamicAffectSystem(
    identity_name="lilu",
    baseline=baseline
)
```

### Using Personality Profiles

```python
# Process emotional input with profile
detected_emotions = {"CARE": 0.8, "SEEKING": 0.6}
affect.process_input(detected_emotions, intensity=0.9, profile="guardian")

# Or use personality manager
from nima_core.cognition.personality_profiles import PersonalityManager

mgr = PersonalityManager()
mgr.set_profile("mystic")  # Activate mystic profile
current = mgr.get_current_profile()
```

---

## 📦 Is This Included in nima-core Package?

**YES!** All of this is included in the `nima-core` pip package:

```bash
pip install nima-core
```

The package includes:
- ✅ `nima_core/cognition/dynamic_affect.py` — Core affect system
- ✅ `nima_core/cognition/archetypes.py` — 10 baseline archetypes
- ✅ `nima_core/cognition/personality_profiles.py` — 14 dynamic profiles
- ✅ `nima_core/cognition/emotion_detection.py` — Emotion-to-affect mapping
- ✅ `nima_core/cognition/affect_interactions.py` — Cross-affect dynamics

---

## 🔧 Configuration for OpenClaw

To use with OpenClaw, you can configure the affect system via plugin settings:

```json
// openclaw.json
{
  "plugins": {
    "entries": {
      "nima-affect": {
        "enabled": true,
        "identity_name": "agent",
        "baseline": "guardian",
        "skipSubagents": true
      }
    }
  }
}
```

**With modifiers:**

```json
{
  "plugins": {
    "entries": {
      "nima-affect": {
        "enabled": true,
        "identity_name": "agent",
        "baseline": {
          "archetype": "guardian",
          "modifiers": {
            "SEEKING": 0.2,
            "PLAY": 0.1
          }
        }
      }
    }
  }
}
```

**Valid configuration keys:**
- `identity_name` — Agent name (default: "agent")
- `baseline` — String (archetype name), Array ([7 numbers]), or Object ({archetype, modifiers})
- `skipSubagents` — Skip subagent sessions (default: true)

**Note:** The `archetype` key must be **inside `baseline`**, not at the top level!

---

## 🧪 Testing Profiles

```bash
# Test dynamic affect system
python3 nima_core/cognition/dynamic_affect.py test

# Check current state
python3 nima_core/cognition/dynamic_affect.py status --name lilu --json

# Test personality profiles
python3 nima_core/cognition/personality_profiles.py
```

---

## 📚 Further Reading

- **Panksepp's 7 Affects:** Research on primary emotional systems (SEEKING, RAGE, FEAR, LUST, CARE, PANIC, PLAY)
- **Affective Neuroscience:** The foundations of human and animal emotions (Jaak Panksepp, 1998)
- **Dynamic Affect Paper:** `research/papers/NIMA_V2_UNIFIED_SPEC.md`

---

## ❓ Common Questions

**Q: Can I create custom profiles?**  
A: Yes! Use `PersonalityManager.create_profile()` or edit archetypes.py directly.

**Q: Do profiles persist across sessions?**  
A: Yes. State is saved to `~/.nima/affect/` and `~/.nima/personality/`.

**Q: Can I switch profiles dynamically?**  
A: Yes. Use `PersonalityManager.set_profile(name)` at runtime.

**Q: What's the difference between archetypes and profiles?**  
A: Archetypes = baseline identity (who you ARE). Profiles = response patterns (how you REACT).

**Q: Can I combine multiple archetypes?**  
A: Use modifiers to blend: `{"archetype": "guardian", "modifiers": {"PLAY": 0.3}}` creates a playful guardian.
