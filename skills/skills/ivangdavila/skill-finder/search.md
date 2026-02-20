# Search Strategies

Reference — load when searching for skills.

## Command

```bash
npx clawhub search <query>
```

## Search by Need, Not Name

User says "I need help with PDFs" — don't just search "pdf".

Think about what they actually need:
- Edit PDFs? → search "pdf edit", "pdf modify"
- Create PDFs? → search "pdf create", "pdf generate"
- Extract from PDFs? → search "pdf extract", "pdf parse"

## Expand Search Terms

If first search yields poor results:
1. Try synonyms (edit → modify, create → generate)
2. Try related tools (pdf → document, docx)
3. Try the underlying task (pdf form → form filling)

## Interpret Results

Search returns: name, description, author, downloads.

**Quick filters:**
- High downloads + recent update = likely maintained
- Clear description = probably well-structured
- Vague description = might be low quality

## When No Results

If nothing found:
1. Try broader terms
2. Try the tool/domain name directly
3. Consider if need is too niche (maybe suggest building custom)

## Multiple Results

When several skills match:
1. List top 3-5 candidates
2. Briefly note what each offers
3. Recommend based on user's specific need
4. Let user choose or ask clarifying questions
