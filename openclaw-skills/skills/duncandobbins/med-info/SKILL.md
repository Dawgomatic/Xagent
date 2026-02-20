---
name: med-info
description: Medication info with citations and traceable IDs (RxCUI/NDC/set_id) from authoritative public sources (openFDA, RxNorm/RxClass, DailyMed, MedlinePlus). Includes label sections plus optional recalls/shortages/FAERS and Orange/Purple Book context.
metadata: {"clawdbot": {"emoji": "💊", "os": ["darwin", "linux"], "requires": {"bins": ["python3"]}}}
---

# med-info

**Medication info with citations—fast.**

Get a label-backed, traceable summary from authoritative public sources (FDA labeling + NLM identifiers). Designed for reference and research — **not** medical advice.

**Privacy:** don’t include PHI. Query by medication name, **RxCUI**, **NDC**, or label **set_id**.

## Highlights

- **Citations & traceability:** includes identifiers (RxCUI, NDC, SPL set_id) plus label dates and source links when available.
- **Label-first answers:** pulls key sections (boxed warning, indications, dosing, contraindications, warnings, interactions, adverse reactions).
- **Normalization:** resolves brand/generic names → RxNorm best-match RxCUI; supports NDC lookups.
- **Patient-friendly context:** links to MedlinePlus Connect when available.
- **Optional add-ons:** recalls, shortages, FAERS aggregates, drug classes, DailyMed history/media, Orange Book, Purple Book.

## Why install

- You want **“show your work”** medication info for notes, training materials, QA, or internal docs.
- You need a quick path from a messy drug name to **clean IDs** (RxCUI/NDC/set_id).
- You’re building automations and need **JSON output** you can audit and reproduce.
- You care about privacy: no patient context required (and none should be provided).

## Sources

This skill queries:
- **openFDA** (drug labels, NDC directory, recalls/enforcement reports, shortages, FAERS)
- **RxNorm (RxNav)** for normalization (RxCUI, brand↔generic mapping)
- **RxClass (RxNav)** for drug class membership
- **DailyMed** for SPL history/media (including labeler-submitted images)
- **Orange Book** data files for TE/RLD context
- **Purple Book** monthly data for biologics, biosimilars, and interchangeability
- **MedlinePlus Connect** for patient-friendly summaries

## Safety rules

- For clinical decisions, **verify against the full official label**. This tool extracts key sections and returns references.
- Do not include PHI (patient-identifying information).
- The script treats all user input as untrusted and **escapes values** when constructing openFDA `search` queries to prevent query-injection style surprises.

## Example commands

### 1) Summarize a drug by name
```bash
cd {baseDir}
python3 scripts/med_info.py "metoprolol succinate" 
```

### 2) Query by NDC
```bash
python3 scripts/med_info.py "70518-4370"     # product_ndc (example)
python3 scripts/med_info.py "70518-4370-0"   # package_ndc (example)
```

### 3) JSON output (for pipelines)
```bash
python3 scripts/med_info.py "ibuprofen" --json
```

### 4) Find a keyword in the label text
```bash
python3 scripts/med_info.py "Eliquis" --find ritonavir
python3 scripts/med_info.py "metformin" --find crush --find chew
```

### 5) Disambiguate labels (candidates, pick, set_id)
```bash
# show label candidates
python3 scripts/med_info.py "metformin" --candidates

# pick the 2nd candidate
python3 scripts/med_info.py "metformin" --candidates --pick 2

# force a specific label by set_id
python3 scripts/med_info.py "05999192-ebc6-4198-bd1e-f46abbfb4f8a"  # set_id
# or
python3 scripts/med_info.py "metformin" --set-id "05999192-ebc6-4198-bd1e-f46abbfb4f8a"
```

### 6) Recalls, shortages, FAERS, and drug classes (optional)
```bash
python3 scripts/med_info.py "metformin" --recalls
python3 scripts/med_info.py "amphetamine" --shortages
python3 scripts/med_info.py "Eliquis" --faers --faers-max 10
python3 scripts/med_info.py "Eliquis" --rxclass
```

### 7) DailyMed and images (optional)
```bash
python3 scripts/med_info.py "Eliquis" --dailymed
python3 scripts/med_info.py "Eliquis" --images

# Note: RxImage was retired in 2021, so --rximage is an alias for --images.
python3 scripts/med_info.py "Eliquis" --rximage
```

### 8) Orange Book and Purple Book (optional)
```bash
python3 scripts/med_info.py "metformin" --orangebook
python3 scripts/med_info.py "adalimumab" --purplebook
```

### 9) Output shaping (optional)
```bash
# only print a couple sections
python3 scripts/med_info.py "Eliquis" --sections contraindications,drug_interactions

# brief output
python3 scripts/med_info.py "Eliquis" --brief --sections all

# print redacted URLs queried
python3 scripts/med_info.py "Eliquis" --print-url --brief
```

## What it returns

- RxNorm resolution (best-match RxCUI + name)
- openFDA label match (effective_time, set_id when present) and key sections:
  - boxed warning
  - indications and usage
  - dosage and administration
  - contraindications
  - warnings and precautions
  - drug interactions
  - adverse reactions
- MedlinePlus Connect links (if available)

## Environment (optional)

- `OPENFDA_API_KEY`: increases openFDA rate limits for heavy usage.

## Implementation notes

- The scripts are intentionally conservative. If multiple candidates exist, it will show the top few and pick the best-scoring RxNorm match.
- Prefer querying by **RxCUI** (more precise) after resolution.
