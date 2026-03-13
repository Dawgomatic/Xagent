---
name: get-profile
description: Query and display user basic medical information with visual formatting including BMI gauge and weight trends.
argument-hint: <>
allowed-tools: Read, Write
schema: get-profile/schema.json
---

# User Basic Profile Query Skill

Display user's basic medical parameters and calculated indicators with beautiful visual formatting.

## Core Flow

```
Execute -> Read profile.json -> Validate Data -> Visual Display -> Quick Actions Prompt
```

## Step 1: Read Data

Read user basic information from `data/profile.json`.

## Step 2: Validate Data

- Check if data exists
- If not exists, prompt to set up
- If partially missing, use simplified display

## Step 3: Visual Display

### Complete Display Format

```
╔══════════════════════════════════════════════════╗
║                  Personal Health Profile       ║
╠══════════════════════════════════════════════════╣
║                                                  ║
║   Basic Information                           ║
║  ────────────────────────────────────────────   ║
║  Height:    ████ 175 cm                         ║
║  Weight:    ██████ 70 kg                        ║
║  Birth Date: 1990-01-01                         ║
║  Age:       35 years                            ║
║                                                  ║
╠══════════════════════════════════════════════════╣
║                                                  ║
║   Health Indicators                           ║
║  ────────────────────────────────────────────   ║
║                                                  ║
║  BMI Index:                                     ║
║  ┌────────────────────────────────────────┐    ║
║  │ Underweight  Normal    Overweight  Obese │    ║
║  │ 18.5        18.5      24.0       28.0    │    ║
║  │             ▼ 22.9                       │    ║
║  └────────────────────────────────────────┘    ║
║  Current: 22.9  [Normal]                        ║
║                                                  ║
║  Body Surface Area (BSA): 1.85 m²               ║
║  (Correction parameter for radiation dose)      ║
║                                                  ║
╚══════════════════════════════════════════════════╝
```

### Simplified Display (Incomplete Data)

```
┌────────────────────────────────────┐
│       Personal Health Profile    │
├────────────────────────────────────┤
│   Basic Information             │
│  ──────────────────────────────    │
│  Height:    175 cm                │
│  Weight:    ---                   │
│  Birth Date: 1990-01-01           │
│                                    │
│   Tip: Use /profile set to complete info │
└────────────────────────────────────┘
```

## Step 4: BMI Status Color Coding

| BMI Range | Status | Symbol |
|-----------|--------|--------|
| < 18.5 | Underweight |  |
| 18.5-23.9 | Normal |  |
| 24-27.9 | Overweight |  |
| >= 28 | Obese |  |

## Step 5: History Display

If `history` array has data, display weight trend:

```
┌────────────────────────────────────┐
│   Weight History (Last 5)        │
├────────────────────────────────────┤
│  2025-12-31  →  70.0 kg (BMI: 22.9)│
│  2025-11-15  →  71.5 kg (BMI: 23.4)│
│  2025-10-01  →  72.0 kg (BMI: 23.5)│
│  2025-08-20  →  73.2 kg (BMI: 23.9)│
│  2025-07-05  →  74.0 kg (BMI: 24.2)│
│                                    │
│   Change: -4.0 kg (-5.4%)        │
└────────────────────────────────────┘
```

## Step 6: Quick Actions Prompt

```
────────────────────────────────────────
 Quick Actions:
   /profile set [height] [weight] [dob]  - Update info
   /vitals [bp] [glucose]                - Record vitals
   /query lab                            - Query lab records
────────────────────────────────────────
```

## Execution Instructions

```
1. Read data/profile.json
2. Check data completeness
3. Select display format based on completeness
4. Render visual output
5. Add quick actions prompt
```

## Example Interactions

### Complete Data Display
```
User invokes skill
-> Display complete profile
```

### Data Missing Prompt
```
User invokes skill (no data)
-> Display setup prompt
```
