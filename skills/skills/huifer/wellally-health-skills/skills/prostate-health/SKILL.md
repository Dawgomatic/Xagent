---
name: prostate-health
description: Prostate health management including PSA monitoring, IPSS scoring, DRE records, and cancer screening
argument-hint: <operation_type, e.g.: psa 2.5/ipss/dre normal/ultrasound 32ml/screening/risk>
allowed-tools: Read, Write
schema: prostate-health/schema.json
---

# Prostate Health Management Skill

Prostate health tracking and management, including PSA monitoring, IPSS symptom scoring, prostate examination planning, and risk assessment.

## Core Flow

```
User Input → Identify Operation Type → Parse Information → Check Completeness → Generate JSON → Save Data
                                     |
                              [Ask when information insufficient]
```

## Step 1: Parse User Input

### Operation Type Recognition

| Input Keywords | Operation |
|----------------|-----------|
| psa, PSA | PSA test record |
| ipss, IPSS | IPSS symptom score |
| dre | DRE record |
| ultrasound | Prostate ultrasound record |
| status | View status |
| screening | Screening plan |
| risk | Risk assessment |

### PSA Value Recognition

| Input Format | Extract |
|--------------|--------|
| PSA 2.5 | total_psa: 2.5 |
| 总PSA 2.5 | total_psa: 2.5 |
| psa 4.2 free 0.9 | total: 4.2, free: 0.9 |

### IPSS Symptom Recognition

| Symptom | Keywords | Score Range |
|---------|----------|-------------|
| Incomplete emptying | not empty | 0-5 |
| Frequency | frequent | 0-5 |
| Nocturia | night | 0-5 |

### DRE Result Recognition

| Keywords | Result |
|----------|--------|
| normal | normal |
| enlarged | enlarged |
| firm | firm |
| nodule | nodule present |
| soft | soft |

## Step 2: Check Information Completeness

### PSA Test (psa)
- **Required**: Total PSA value
- **Optional**: Free PSA value, test date

### IPSS Score (ipss)
- **Required**: None (interactive scoring)

### DRE Exam (dre)
- **Required**: Exam result description
- **Recommended**: Size, texture, nodule status

## Step 3: Interactive Prompts (If Needed)

### Question Scenarios

#### Scenario A: PSA Missing Specific Value
```
Please provide PSA test result, for example:
• Total PSA: 2.5 ng/mL
• Free PSA: 0.8 ng/mL (optional)
```

#### Scenario B: IPSS Interactive Scoring
```
Please answer the following 7 questions, each 0-5 points:

1. Incomplete emptying: Often feel incomplete bladder emptying?
   0-None 1-Less than 1/5 2-Less than 1/2 3-About 1/2 4-More than 1/2 5-Almost always

[Ask each question sequentially...]
```

## Step 4: Generate JSON

### PSA Data Structure

```json
{
  "record_type": "psa_test",
  "date": "2025-06-15",
  "total_psa": 2.5,
  "free_psa": 0.8,
  "ratio": 0.32,
  "unit": "ng/mL",
  "reference": "<4.0",
  "trend": "stable",
  "risk_level": "low",
  "interpretation": "Normal"
}
```

### IPSS Data Structure

```json
{
  "record_type": "ipss_score",
  "date": "2025-06-20",
  "incomplete_emptying": 1,
  "frequency": 2,
  "intermittency": 1,
  "urgency": 2,
  "weak_stream": 1,
  "straining": 0,
  "nocturia": 2,
  "total_score": 9,
  "severity": "moderate",
  "quality_of_life_score": 2
}
```

### DRE Data Structure

```json
{
  "record_type": "dre_exam",
  "date": "2025-06-15",
  "findings": "normal",
  "size": "normal",
  "texture": "soft",
  "nodule": false,
  "tenderness": false,
  "mobility": "normal",
  "notes": ""
}
```

Complete schema definition: see [schema.json](schema.json).

## Step 5: Save Data

1. Generate record ID: `prostate_YYYYMMDD_XXX`
2. Save to `data/前列腺记录/YYYY-MM/YYYY-MM-DD_PSA检测.json`
3. Update `data/prostate-tracker.json`

## PSA Risk Assessment Standards

| Total PSA (ng/mL) | Risk Level | Recommendation |
|---------------------|------------|---------------|
| < 2.5 (under 50) or < 4.0 (over 50) | Low | Continue routine screening |
| 2.5-4.0 (under 50) or 4.0-10.0 | Moderate | Recommend repeat in 3 months |
| > 10.0 | High | Recommend immediate urology visit |

### Free/Total PSA Ratio

| Ratio | Interpretation |
|-------|---------------|
| > 0.25 | Suggests benign |
| 0.10-0.25 | Gray zone, comprehensive assessment needed |
| < 0.10 | Need to be alert for malignancy |

## IPSS Symptom Severity

| Total Score | Severity | Management Recommendation |
|-------------|----------|---------------------------|
| 0-7 | Mild | Observe and follow up |
| 8-19 | Moderate | Consider medication |
| 20-35 | Severe | Recommend urology evaluation |

## Prostate Volume Classification

| Volume | Classification |
|--------|---------------|
| < 20 mL | Small |
| 20-30 mL | Normal |
| 30-50 mL | Mildly enlarged (BPH Grade I) |
| 50-80 mL | Moderately enlarged (BPH Grade II) |
| > 80 mL | Severely enlarged (BPH Grade III) |

## Execution Instructions

```
1. Identify operation type
2. Parse input parameters, extract all available information
3. Check required fields, ask user if missing
4. Judge results based on assessment standards
5. Generate JSON compliant with schema.json
6. Save to appropriate file path
7. Confirm save to user and display summary and recommendations
```

## Example Interactions

### Example 1: Record PSA Test (Normal)
```
User: PSA检测 2.5
AI:  PSA检测已记录

    PSA Information:
    Total PSA: 2.5 ng/mL ✓ (reference: <4.0)
    Risk level: Low 

    Recommendation: Continue routine screening, next test in 1 year

    Saved to: data/前列腺记录/2025-06/2025-06-15_PSA检测.json
```

### Example 2: IPSS Score
```
User: IPSS症状评分
AI: 请回答以下7个问题，每个问题0-5分：

    [Interactive questioning...]

     IPSS评分已完成

    Total score: 9/35 points
    Severity: Moderate 

    Recommendation: Observe and follow up, avoid drinking before bed, reduce caffeine and alcohol

    Saved to: data/前列腺记录/2025-06/2025-06-20_IPSS评分.json
```

### Example 3: View Screening Plan
```
User: 前列腺筛查计划
AI:  Prostate Cancer Screening Plan

    Screening Recommendations:
     PSA test: Annually
     DRE (Digital Rectal Exam): Annually

    Next test: 2026-06-15 (362 days away)

    Test Preparation:
    • 24-48 hours after ejaculation
    • 48 hours after prostate massage
    • No acute urinary tract infection

    Data saved
```

## Important Notice

- This system is for prostate health tracking only, cannot replace professional medical advice
- Elevated PSA does not equal prostate cancer
- Regular screening is very important for early detection of prostate cancer
- Those with family history need closer monitoring

## Seek Immediate Medical Care For

- PSA significantly elevated (>10 ng/mL)
- DRE finds prostate nodule
- Severe difficulty urination or urinary retention
- Blood in urine (hematuria)
- Bone pain (suspected metastasis)
