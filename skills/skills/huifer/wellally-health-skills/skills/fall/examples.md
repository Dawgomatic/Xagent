# fall Skill Examples

## I. Record Fall Events

### Example 1: Bathroom Fall
```
User: /fall record 2025-03-15 bathroom slippery_floor bruise

Save:
{
  "id": "fall_20250315_001",
  "date": "2025-03-15",
  "location": "bathroom",
  "cause": "slippery_floor",
  "injury_level": "bruise",
  "required_medical_attention": false
}
```

### Example 2: Bedroom Fall (Natural Language)
```
User: /fall record today bedroom slippery_floor minor_abrasion

After parsing, save:
{
  "date": "2025-03-15",
  "location": "bedroom",
  "cause": "slippery_floor",
  "injury_level": "bruise"
}
```

### Example 3: Dizziness-Induced Fall
```
User: /fall record 2025-03-10 living_room dizziness none

Save:
{
  "date": "2025-03-10",
  "location": "living_room",
  "cause": "dizziness",
  "injury_level": "none"
}
```

## II. Balance Function Tests

### Example 4: TUG Test (Normal)
```
User: /fall tug 18

Save:
{
  "id": "tug_20250315_001",
  "date": "2025-03-15",
  "time_seconds": 18,
  "interpretation": "Basically normal",
  "fall_risk": "low_risk"
}
```

### Example 5: TUG Test (Assistance Needed)
```
User: /fall tug 22seconds

Save:
{
  "time_seconds": 22,
  "interpretation": "Mobility limited",
  "fall_risk": "moderate_risk"
}
```

### Example 6: Berg Balance Scale
```
User: /fall berg 42

Save:
{
  "score": 42,
  "interpretation": "Independent walking",
  "fall_risk": "low_risk"
}
```

### Example 7: Single Leg Stance Test
```
User: /fall single-leg-stance 8

Save:
{
  "eyes_open_seconds": 8,
  "interpretation": "Decreased balance ability"
}
```

## III. Gait Analysis

### Example 8: Walking Speed Test
```
User: /fall gait speed 0.8

Save:
{
  "speed_m_per_s": 0.8,
  "abnormalities": ["normal"],
  "interpretation": "Mobility limited"
}
```

### Example 9: Gait Abnormalities
```
User: /fall gait abnormal shortened_step widened_base

Save:
{
  "abnormalities": ["shortened_step", "widened_base"],
  "interpretation": "Gait abnormality, increased fall risk"
}
```

### Example 10: Comprehensive Gait Assessment
```
User: /fall gait speed 0.7 shortened_step unsteady_gait

Save:
{
  "speed_m_per_s": 0.7,
  "abnormalities": ["shortened_step", "unsteady_gait"],
  "interpretation": "Significantly limited mobility"
}
```

## IV. Home Environment Assessment

### Example 11: Living Room Assessment
```
User: /fall home living_room floor_slippery false

Save:
{
  "living_room": {
    "floor_slippery": false
  }
}
```

### Example 12: Bathroom Assessment
```
User: /fall home bathroom grab_bars true

Save:
{
  "bathroom": {
    "grab_bars": true
  }
}
```

### Example 13: Comprehensive Assessment
```
User: /fall home assessment

Output assessment results and improvement recommendations for all rooms
```

## V. Risk Assessment

### Example 14: View Risk Level
```
User: /fall risk

Output:
 Fall Risk Assessment

Overall Risk Level: Moderate Risk (8 points)
━━━━━━━━━━━━━━━━━━━━━━━━━━

Risk Factor Analysis:
━━━━━━━━━━━━━━━━━━━━━━━━━━
 Age: Not over 75
 Previous Falls: 1 fall in the past year
 Balance: TUG test 22 seconds, mobility limited
 Gait: Basically normal
 Home Environment: Bathroom lacks grab bars

Main Risk Factors:
━━━━━━━━━━━━━━━━━━━━━━━━━━
1. History of falls
2. Decreased balance function
3. Home environmental hazards

Recommended Interventions:
━━━━━━━━━━━━━━━━━━━━━━━━━━
• Install grab bars in bathroom
• Consider using shower chair
• Balance and strength training
• Remove or secure rugs at home
• Improve lighting
```

### Example 15: View Risk Factors
```
User: /fall risk-factors

Output:
Fall Risk Factor Checklist
━━━━━━━━━━━━━━━━━━━━━━━━━━
 Controlled:
  - Age risk
  - Vision problems

 Uncontrolled:
  - Previous falls
  - Balance problems
  - Home environmental hazards
```

## VI. Intervention Recommendations

### Example 16: Get Intervention Recommendations
```
User: /fall interventions

Output:
Fall Prevention Intervention Recommendations
━━━━━━━━━━━━━━━━━━━━━━━━━━

Balance and Strength Training:
━━━━━━━━━━━━━━━━━━━━━━━━━━
• Tai Chi, 2-3 times per week
• Single leg stance practice, 3 sets daily
• Straight leg raises, 10 times daily
• Hip abduction exercises, 10 times daily

Home Environment Modifications:
━━━━━━━━━━━━━━━━━━━━━━━━━━
• Install grab bars in bathroom
• Use non-slip mats
• Install night lights
• Remove or secure rugs
• Install handrails on both sides of stairs

Medication Adjustment Recommendations:
━━━━━━━━━━━━━━━━━━━━━━━━━━
(Consult physician)
• Review medications causing dizziness
• Adjust antihypertensive dosage (if needed)

Other Recommendations:
━━━━━━━━━━━━━━━━━━━━━━━━━━
• Regular eye examinations
• Use appropriate mobility aids
• Wear non-slip shoes
```
