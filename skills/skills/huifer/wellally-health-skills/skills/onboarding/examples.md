# Onboarding Examples

## Example 1: Complete Onboarding Flow

```
User: Hello

System:
Welcome to WellallyHealth!

I see you're new here. Let's set up your account.

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
STEP 1: Language Selection
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

Please select your preferred language:
1. English
2. 中文 (Chinese)

User: 1

System:
Language set to English.

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
STEP 2: Profile Setup
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

Let's set up your profile.

Please provide:
1. Gender (M/F)
2. Height (cm)
3. Weight (kg)
4. Birth date (YYYY-MM-DD)

Example: F 165 55 1990-01-15

User: F 165 55 1990-01-15

System:
Profile saved!

Gender: Female
Height: 165 cm
Weight: 55 kg
Birth date: 1990-01-15
Age: 36 years
BMI: 20.2 (Normal)

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
STEP 3: Feature Introduction
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

WellallyHealth offers these features:

Profile & Tracking
• /profile - View your profile
• /medication - Track medications
• /diet - Log meals and nutrition
• /fitness - Record exercise

Health Monitoring
• /symptom - Record symptoms
• /ai-health - Get AI insights
• /report - Generate health reports

Press Enter to continue...

User: [Enter]

System:
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
SETUP COMPLETE!
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

You're ready to use WellallyHealth!

Try these commands:
• /diet add - Log a meal
• /medication add - Add a medication
• /help - See all commands

Type "/onboarding complete" to finish.
```

## Example 2: Resume Onboarding

```
User: /onboarding continue

System:
Welcome back! Let's continue setup.

Progress:
✓ Language selection
✓ Profile setup
○ Feature introduction

Continuing from: Feature introduction

[Shows feature intro...]

User: skip

System:
Feature introduction skipped.

You can complete setup anytime with "/onboarding complete"
Or continue using any feature!
```

## Example 3: Skip Onboarding

```
User: /onboarding skip

System:
Onboarding skipped.

You can set up your profile later:
• /profile - Set your profile
• /language - Change language

Type "/help" to see all available commands.
```

## Example 4: Chinese Onboarding

```
User: [First message in Chinese]

System:
欢迎使用 WellallyHealth！

让我来帮您设置账户。

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
第一步：选择语言
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

请选择您的首选语言：
1. English
2. 中文 (中文)

User: 2

System:
语言已设置为中文。

[Continues in Chinese...]
```

## Example 5: Partial Completion Detection

```
[User has completed language and profile steps]

User: /medication add Aspirin 100mg

System:
[Detects onboarding not complete]
[Auto-completes onboarding]

Your profile is set up! Onboarding marked as complete.

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
Adding Medication: Aspirin
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

[Proceeds with medication addition...]
```

## Example 6: Onboarding Completion

```
User: /onboarding complete

System:
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
ONBOARDING COMPLETE!
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

Congratulations! You're all set up.

Your Profile:
────────────────────────────────────
Name: Not set
Gender: Female
Height: 165 cm
Weight: 55 kg
Age: 36 years
BMI: 20.2 (Normal)
────────────────────────────────────

Quick Start:
• /diet add [image] - Log a meal
• /medication add - Add medication
• /symptom - Record symptoms
• /ai-health - Get health insights

Type "/help" anytime to see all commands.

Welcome to WellallyHealth! 
```

## Example 7: State Persistence

```
[Before onboarding]
{
  "onboarding": {
    "completed": false,
    "language_set": false,
    "steps_completed": []
  }
}

[After language selection]
{
  "onboarding": {
    "completed": false,
    "language_set": true,
    "steps_completed": ["language_selection"]
  }
}

[After profile setup]
{
  "onboarding": {
    "completed": false,
    "language_set": true,
    "steps_completed": ["language_selection", "profile_setup"]
  }
}

[After completion]
{
  "onboarding": {
    "completed": true,
    "language_set": true,
    "steps_completed": ["language_selection", "profile_setup", "completion"]
  }
}
```
