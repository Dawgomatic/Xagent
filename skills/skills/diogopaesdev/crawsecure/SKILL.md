Offline security scanner for ClawHub skills — detect unsafe patterns before installation.

# CrawSecure

CrawSecure is an **offline security analysis skill** designed to help users evaluate potential risks in ClawHub / OpenClaw skills **before installing or trusting them**.

It promotes safer usage, transparency, and awareness when working with third-party skills.

---

##  What CrawSecure does

- Analyzes skill-related files locally
- Detects potentially dangerous patterns
- Highlights security risks clearly
- Helps users make informed decisions before installation

---

##  Risk Signals Analyzed

- Dangerous command patterns (e.g. destructive or execution-related behavior)
- References to sensitive files or credentials
- Indicators of unsafe or risky practices

---

##  Security Philosophy

- Read-only analysis
- No network access
- No code execution
- No file modifications

CrawSecure exists to **increase trust** inside the ClawHub ecosystem.

---

##  Execution Model

CrawSecure does **NOT** execute or install any third-party code.

This skill provides a **local CLI tool** that users run manually.

### Using npx (recommended)

```bash
npx crawsecure ./path-to-skill
