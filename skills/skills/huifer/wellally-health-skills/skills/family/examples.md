# Family Skill Examples

## Example 1: Add Family Member

```
User: /family add-member father ZhangSan 1960-05-15 blood_type_A

AI:  Family member added

Basic Information:
━━━━━━━━━━━━━━━━━━━━━━━━━━━━
Member ID: mem_20250108_001
Name: ZhangSan
Relationship: Father
Gender: Male
Birth Date: 1960-05-15
Blood Type: A
Status: Living

Saved to family health record
```

## Example 2: Add Multiple Members

```
User: /family add-member mother LiSi 1962-08-20 blood_type_B

AI:  Family member added

Basic Information:
━━━━━━━━━━━━━━━━━━━━━━━━━━━━
Member ID: mem_20250108_002
Name: LiSi
Relationship: Mother
Gender: Female
Birth Date: 1962-08-20
Blood Type: B
Status: Living

Saved to family health record
```

## Example 3: Record Family Medical History

```
User: /family add-history father hypertension diagnosed_at_50

AI:  Family medical history recorded

Medical History Information:
━━━━━━━━━━━━━━━━━━━━━━━━━━━━
Disease: Hypertension
Category: Cardiovascular disease
Patient: ZhangSan (Father)
Age of Onset: 50 years
Severity: Moderate

Saved to family medical history records
```

## Example 4: View Family Member List

```
User: /family list

AI:  Family Member List

━━━━━━━━━━━━━━━━━━━━━━━━━━━━
Immediate Family
━━━━━━━━━━━━━━━━━━━━━━━━━━━━
• ZhangSan (Father) - 65 years old - Blood type A
  Status: Living
  Chronic Conditions: Hypertension

• LiSi (Mother) - 63 years old - Blood type B
  Status: Living
  Chronic Conditions: Diabetes

━━━━━━━━━━━━━━━━━━━━━━━━━━━━
Spouse
━━━━━━━━━━━━━━━━━━━━━━━━━━━━
• WangWu (Spouse) - 37 years old - Blood type O
  Status: Living

━━━━━━━━━━━━━━━━━━━━━━━━━━━━
Children
━━━━━━━━━━━━━━━━━━━━━━━━━━━━
• XiaoMing (Son) - 15 years old - Blood type A
  Status: Living

Total: 4 family members
```

## Example 5: Genetic Risk Assessment

```
User: /family risk

AI:  Genetic Risk Assessment Report

━━━━━━━━━━━━━━━━━━━━━━━━━━━━
Assessment Date: 2026-02-03

High-Risk Diseases
━━━━━━━━━━━━━━━━━━━━━━━━━━━━
 Hypertension
Risk Level: High Risk (75%)
Affected Members: Father
Risk Factors:
• First-degree relative affected
• Early-onset (<50 years)

Recommendations:
• Regular blood pressure monitoring (3 times per week)
• Limit sodium intake (<5g/day)
• Regular aerobic exercise (150 minutes per week)
• Start regular checkups at age 35

━━━━━━━━━━━━━━━━━━━━━━━━━━━━
 Diabetes
Risk Level: Medium Risk (55%)
Affected Members: Mother
Risk Factors:
• First-degree relative affected
• Middle-age onset

Recommendations:
• Control weight and waist circumference
• Low-sugar, low-fat diet
• Increase dietary fiber
• Regular exercise
• Start annual blood glucose checks at age 40

━━━━━━━━━━━━━━━━━━━━━━━━━━━━
Low-Risk Diseases
━━━━━━━━━━━━━━━━━━━━━━━━━━━━
 Coronary Heart Disease
Risk Level: Low Risk (25%)
Note: Only distant relatives affected, sporadic cases

━━━━━━━━━━━━━━━━━━━━━━━━━━━━
Screening Recommendation Schedule
━━━━━━━━━━━━━━━━━━━━━━━━━━━━
Item              | Frequency   | Start Age
------------------|-------------|----------
Blood Pressure     | 3x/week     | Now
Fasting Glucose   | 1x/year     | Age 40
Lipid Profile      | 1x/3 years  | Age 40
ECG                | 1x/5 years  | Age 40

━━━━━━━━━━━━━━━━━━━━━━━━━━━━
Important Notice
━━━━━━━━━━━━━━━━━━━━━━━━━━━━
This risk assessment is for reference only and does not predict individual disease.
Please consult a professional physician or genetic counselor for specific risks.
```

## Example 6: Generate Family Health Report

```
User: /family report

AI:  Family Health Analysis Report

━━━━━━━━━━━━━━━━━━━━━━━━━━━━
Family Overview
━━━━━━━━━━━━━━━━━━━━━━━━━━━━
Family Members: 4 people
Generations: 3 generations
Average Age: 45 years

━━━━━━━━━━━━━━━━━━━━━━━━━━━━
Family Medical History Summary
━━━━━━━━━━━━━━━━━━━━━━━━━━━━
Cardiovascular Disease: 2 people
• Hypertension: Father (onset at 50)
• Coronary Heart Disease: Grandfather (onset at 60)

Metabolic Disease: 1 person
• Diabetes: Mother (onset at 55)

Cancer: 0 people

━━━━━━━━━━━━━━━━━━━━━━━━━━━━
Genetic Risk Heatmap
━━━━━━━━━━━━━━━━━━━━━━━━━━━━
Hypertension    ████████████░░ 75% (High)
Diabetes        ████████░░░░░░░ 55% (Medium)
Coronary Disease ████░░░░░░░░░░░ 25% (Low)

━━━━━━━━━━━━━━━━━━━━━━━━━━━━
Common Health Issues
━━━━━━━━━━━━━━━━━━━━━━━━━━━━
1. Metabolic issues (hypertension, diabetes)
2. Dietary habits (high salt, high fat)
3. Insufficient exercise

━━━━━━━━━━━━━━━━━━━━━━━━━━━━
Prevention Recommendations Checklist
━━━━━━━━━━━━━━━━━━━━━━━━━━━━
Immediate:
□ Start recording daily blood pressure
□ Reduce daily salt intake to <5g
□ Exercise 3 times per week, 30 minutes each

Within 1 month:
□ Purchase home blood pressure monitor
□ Consult nutritionist for meal planning
□ Family health checkup

Within 3 months:
□ Establish healthy eating habits
□ Make exercise a daily routine
□ Follow-up to assess effectiveness

━━━━━━━━━━━━━━━━━━━━━━━━━━━━
Important Disclaimer
━━━━━━━━━━━━━━━━━━━━━━━━━━━━
This report is for reference only and cannot replace professional medical diagnosis.
Genetic risk assessment does not predict individual disease.
Please consult a professional physician for all medical decisions.
```

## Example 7: Track Member Health Data

```
User: /family track father blood_pressure 135/85

AI:  Health data recorded

Member: ZhangSan (Father)
Data Type: Blood Pressure
Value: 135/85 mmHg
Record Time: 2026-02-03 14:30

Assessment: Blood pressure slightly elevated, monitoring recommended

━━━━━━━━━━━━━━━━━━━━━━━━━━━━
Blood Pressure Reference Range:
Normal: <120/80 mmHg
High-Normal: 120-139/80-89 mmHg
Hypertension: ≥140/90 mmHg
```
