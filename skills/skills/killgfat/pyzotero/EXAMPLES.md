# Pyzotero CLI Examples

Real-world command-line examples and workflows for common tasks with pyzotero CLI.

## Table of Contents

1. [Basic Search](#basic-search)
2. [Advanced Search](#advanced-search)
3. [Collection Management](#collection-management)
4. [Working with Output](#working-with-output)
5. [Automation Scripts](#automation-scripts)
6. [Daily Research Workflow](#daily-research-workflow)
7. [Literature Review Workflow](#literature-review-workflow)

---

## Basic Search

### Simple Text Search

```bash
# Search for topic in titles and metadata
pyzotero search -q "machine learning"

# Phrase search (use quotes)
pyzotero search -q "\"deep learning\""

# Search for multiple terms
pyzotero search -q "python data science"
```

### Full-Text Search

Search within PDFs and attachments (requires Zotero to have indexed PDFs):

```bash
# Full-text search
pyzotero search -q "neural networks" --fulltext

# Full-text with quotes
pyzotero search --fulltext -q "\"attention mechanism\""
```

### Filter by Item Type

```bash
# Journal articles only
pyzotero search -q "python" --itemtype journalArticle

# Books only
pyzotero search -q "programming" --itemtype book

# Multiple item types
pyzotero search -q "data" --itemtype book --itemtype journalArticle

# Conference papers
pyzotero search -q "AI" --itemtype conferencePaper
```

---

## Advanced Search

### Search Within Specific Collection

```bash
# First, list collections to get IDs
pyzotero listcollections

# Search within specific collection
pyzotero search --collection ABC123 -q "topic"

# Full-text within collection
pyzotero search --collection DEF456 -q "neural" --fulltext
```

### Combined Search Criteria

```bash
# Search for topic in journal articles
pyzotero search -q "machine learning" --itemtype journalArticle

# Search phrase in books
pyzotero search -q "\"statistical methods\"" --itemtype book

# Full-text search in articles
pyzotero search --itemtype journalArticle --fulltext -q "transformer"
```

### Case-Insensitive Search

Search is case-insensitive by default:

```bash
# Both work the same
pyzotero search -q "Machine Learning"
pyzotero search -q "machine learning"
pyzotero search -q "MACHINE LEARNING"
```

---

## Collection Management

### List All Collections

```bash
pyzotero listcollections
```

Output example:
```
Collection: My Library (ABC123)
- Key: ABC123
- Total items: 150

Collection: Research Papers (DEF456)
- Key: DEF456
- Total items: 75

Collection: Books to Read (GHI789)
- Key: GHI789
- Total items: 20
```

### Search Within Collections

```bash
# Search in "Research Papers" collection
pyzotero search --collection DEF456 -q "deep learning"

# Full-text search in specific collection
pyzotero search --collection DEF456 -q "attention mechanism" --fulltext
```

### List Item Types

```bash
pyzotero itemtypes
```

Output example:
```
book
journalArticle
conferencePaper
thesis
report
webpage
preprint
...
```

---

## Working with Output

### Human-Readable Output (Default)

```bash
pyzotero search -q "machine learning"
```

Output example:
```
Title: Deep Learning for Natural Language Processing
Type: journalArticle
Key: ABC123
Authors: Smith, J.; Doe, J.
Date: 2024
Publication: Journal of Machine Learning Research
DOI: 10.1234/jmlr.2024.123

Title: Attention Is All You Need
Type: conferencePaper
Key: DEF456
Authors: Vaswani, A.; et al.
Date: 2017
Publication: Advances in Neural Information Processing Systems
```

### JSON Output

```bash
# Get JSON output
pyzotero search -q "machine learning" --json
```

Parse with jq:

```bash
# Extract titles
pyzotero search -q "topic" --json | jq '.[] | .title'

# Extract title and date
pyzotero search -q "topic" --json | jq '.[] | {title: .title, date: .date}'

# Extract authors
pyzotero search -q "topic" --json | jq '.[] | .authors'

# Count results
pyzotero search -q "topic" --json | jq 'length'

# Create a list
pyzotero search -q "topic" --json | jq -r '.[] | .title' | nl
```

### Export to File

```bash
# Export to JSON file
pyzotero search -q "machine learning" --json > ml_papers.json

# Export titles to text file
pyzotero search -q "topic" --json | jq -r '.[] | .title' > titles.txt

# Export with metadata
pyzotero search -q "topic" --json > results.json
```

---

## Automation Scripts

### Batch Search Script

Create file `batch_search.sh`:

```bash
#!/bin/bash

# Batch search script
TOPICS=("machine learning" "deep learning" "neural networks" "AI")

for topic in "${TOPICS[@]}"; do
    echo "=== Searching for: $topic ==="
    pyzotero search -q "$topic"
    echo ""
done
```

Run it:
```bash
chmod +x batch_search.sh
./batch_search.sh
```

### Create Research Report

```bash
#!/bin/bash

TOPIC="$1"
OUTPUT="research_report_${topic// /_}_$(date +%Y%m%d).md"

echo "# Research Report: $topic" > "$OUTPUT"
echo "Generated: $(date)" >> "$OUTPUT"
echo "" >> "$OUTPUT"

echo "## Search Results" >> "$OUTPUT"
pyzotero search -q "$topic" >> "$OUTPUT"

echo "" >> "$OUTPUT"
echo "## Full-Text Results" >> "$OUTPUT"
pyzotero search -q "$topic" --fulltext >> "$OUTPUT"

echo "Report saved to: $OUTPUT"
```

Usage:
```bash
./create_report.sh "machine learning"
```

### Find Missing References

```bash
#!/bin/bash

# Check if specific papers are in library
while IFS= read -r title; do
    result=$(pyzotero search -q "\"$title\"" --json | jq 'length')
    if [ "$result" -eq 0 ]; then
        echo "Missing: $title"
    else
        echo "Found: $title"
    fi
done < papers_to_check.txt
```

### Count Items by Year

```bash
#!/bin/bash

# Count items by year
pyzotero search --json | jq -r '.[] | select(.date != null) | .date' | \
    grep -oE '^[0-9]{4}' | \
    sort | uniq -c | \
    sort -nr
```

---

## Daily Research Workflow

### Morning Scan

```bash
#!/bin/bash
# morning_scan.sh

echo "=== Morning Research Scan - $(date) ==="
echo ""

# Check recent additions
echo "Search: recent research"
pyzotero search --fulltext -q "new research" | head -20

echo ""
echo "=== Collections Overview ==="
pyzotero listcollections

echo ""
echo "=== Quick Stats ==="
echo "Total items: $(pyzotero search --json | jq 'length')"
```

### Quick Lookup

```bash
# Function for quick lookup
zotsearch() {
    pyzotero search -q "$@"
}

# Add to ~/.bashrc or ~/.zshrc
alias zotsearch='pyzotero search'
alias zotlist='pyzotero listcollections'
alias zotfull='pyzotero search --fulltext'

# Usage
zotsearch "neural networks"
zotfull "attention mechanism"
```

### Find Citations for Writing

```bash
#!/bin/bash
# find_citations.sh

TOPIC="$1"

echo "=== Citations for: $topic ==="

# Find journal articles
echo "## Journal Articles"
pyzotero search -q "$topic" --itemtype journalArticle

# Find books
echo "## Books"
pyzotero search -q "$topic" --itemtype book

# Find conference papers
echo "## Conference Papers"
pyzotero search -q "$topic" --itemtype conferencePaper
```

Usage:
```bash
./find_citations.sh "machine learning"
```

---

## Literature Review Workflow

### Step 1: Topic Exploration

```bash
# Initial broad search
pyzotero search -q "machine learning"
```

### Step 2: Narrow Down

```bash
# Focus on journal articles
pyzotero search -q "machine learning" --itemtype journalArticle

# Search specific aspect
pyzotero search -q "machine learning applications" --fulltext
```

### Step 3: Organize by Collection

```bash
# List collections
pyzotero listcollections

# Search within specific collection
pyzotero search --collection RESEARCH -q "deep learning"
```

### Step 4: Extract Results

```bash
# Export to JSON for further processing
pyzotero search -q "machine learning" --json > ml_review.json

# Extract titles
cat ml_review.json | jq -r '.[] | .title' > ml_titles.txt

# Count by year
cat ml_review.json | jq -r '.[] | .date' | grep -oE '^[0-9]{4}' | sort | uniq -c | sort -nr
```

### Complete Literature Review Script

```bash
#!/bin/bash
# lit_review.sh

if [ -z "$1" ]; then
    echo "Usage: $0 <topic>"
    exit 1
fi

TOPIC="$1"
DATE=$(date +%Y%m%d)
DIR="lit_review_${TOPIC// /_}_${DATE}"

mkdir -p "$DIR"
cd "$DIR"

echo "=== Literature Review: $TOPIC ==="
echo "Working in: $DIR"

# 1. Broad search
echo "1. Broad search results..." > broad.txt
pyzotero search -q "$TOPIC" > broad.txt

# 2. Journal articles
echo "2. Journal articles..." > articles.txt
pyzotero search -q "$TOPIC" --itemtype journalArticle > articles.txt

# 3. Full-text search
echo "3. Full-text results..." > fulltext.txt
pyzotero search -q "$TOPIC" --fulltext > fulltext.txt

# 4. Export JSON
echo "4. Exporting JSON..."
pyzotero search -q "$TOPIC" --json > data.json

# 5. Analysis
echo "5. Analysis..."

# Total results
TOTAL=$(pyzotero search -q "$TOPIC" --json | jq 'length')
echo "Total results: $TOTAL" > analysis.txt

# By year
echo "By year:" >> analysis.txt
cat data.json | jq -r '.[] | .date' | grep -oE '^[0-9]{4}' | sort | uniq -c | sort -nr >> analysis.txt

# By type
echo "By type:" >> analysis.txt
cat data.json | jq -r '.[] | .itemType' | sort | uniq -c | sort -nr >> analysis.txt

echo "Literature review complete! See files in: $DIR"
```

Usage:
```bash
./lit_review.sh "transformer architecture"
```

---

## Advanced Workflows

### Cross-Reference Search

```bash
#!/bin/bash
# cross_reference.sh

# Search for common references between topics
TOPIC1="$1"
TOPIC2="$2"

echo "=== Cross-Reference: $topic1 vs $topic2 ==="

# Get results for each topic
pyzotero search -q "$TOPIC1" --json > topic1.json
pyzotero search -q "$TOPIC2" --json > topic2.json

# Find common items (same keys)
echo "Common items:"
comm -12 <(cat topic1.json | jq -r '.[] | .key' | sort) <(cat topic2.json | jq -r '.[] | .key' | sort) | while read key; do
    pyzotero search --json | jq ".[] | select(.key == \"$key\") | .title"
done
```

### Trend Analysis

```bash
#!/bin/bash
# trend_analysis.sh

TOPIC="$1"
START_YEAR="$2"
END_YEAR="${3:-$(date +%Y)}"

echo "=== Topic Trend: $TOPIC ($START_YEAR - $END_YEAR) ==="

for year in $(seq $START_YEAR $END_YEAR); do
    count=$(pyzotero search -q "$TOPIC" --json | jq "[.[] | select(.date | contains(\"$year\"))] | length")
    echo "$year: $count"
done
```

Usage:
```bash
./trend_analysis.sh "machine learning" 2010 2024
```

### Export for Academic Writing

```bash
#!/bin/bash
# export_for_paper.sh

TOPIC="$1"

echo "=== Exporting to BibTeX-style format ==="

pyzotero search -q "$TOPIC" --json | while IFS= read -r item; do
    title=$(echo "$item" | jq -r '.title // empty')
    date=$(echo "$item" | jq -r '.date // empty' | grep -oE '^[0-9]{4}')
    item_type=$(echo "$item" | jq -r '.itemType // empty')

    if [ ! -z "$title" ]; then
        echo "@$item_type{key,"
        echo "  title = {$title},"
        if [ ! -z "$date" ]; then
            echo "  year = {$date},"
        fi
        echo "}"
        echo ""
    fi
done > export.bib
```

---

## Tips and Tricks

### Create Aliases

Add to `~/.bashrc` or `~/.zshrc`:

```bash
# Basic aliases
alias zot='pyzotero'
alias zotsearch='pyzotero search'
alias zotlist='pyzotero listcollections'
alias zotfull='pyzotero search --fulltext'
alias zotjson='pyzotero search --json'

# Topic-specific aliases
alias zotml='pyzotero search -q "machine learning"'
alias zotdl='pyzotero search -q "deep learning"'
alias zotai='pyzotero search -q "artificial intelligence"'
```

### Use with FZF (Fuzzy Finder)

```bash
# Interactive search
zotsearch-fzf() {
    local topic=$(pyzotero search --json | jq -r '.[] | .title' | fzf --prompt="Search: ")
    if [ ! -z "$topic" ]; then
        pyzotero search -q "\"$topic\""
    fi
}

# Add to shell and call with: zotsearch-fzf
```

### Combine with Other Tools

```bash
# Send results to clipboard
pyzotero search -q "topic" | xclip -selection clipboard

# Open URLs in browser
pyzotero search -q "topic" --json | jq -r '.[] | .url' | xargs -I {} xdg-open {}

# Create markdown bibliography
pyzotero search -q "topic" --json | jq -r '.[] | "* " + .title + " (" + .date + ")"'
```

### Logging Searches

```bash
# Log all searches to file
zotsearch_log() {
    echo "$(date): pyzotero search -q $@" >> ~/.zotero_search.log
    pyzotero search -q "$@"
}

# View search history
tail -f ~/.zotero_search.log
```

---

## Troubleshooting Examples

### Debug Connection Issues

```bash
# Check if Zotero is running
pgrep -f zotero

# Test local API
curl -X GET http://localhost:23119/zotero/items

# Check collections (basic test)
pyzotero listcollections
```

### Verify Search Results

```bash
# Count results
pyzotero search -q "topic" --json | jq 'length'

# Check if results are empty
if [ $(pyzotero search -q "topic" --json | jq 'length') -eq 0 ]; then
    echo "No results found"
fi
```

### Check Available Item Types

```bash
# List all item types
pyzotero itemtypes

# Use specific type in search
pyzotero search -q "topic" --itemtype journalArticle
```

---

## Performance Tips

### For Large Libraries

```bash
# Search is case-insensitive by default - use exact phrases for precision
pyzotero search -q "\"Exact Phrase\""

# Use item type filters to reduce results
pyzotero search -q "topic" --itemtype journalArticle

# Use collection filters when possible
pyzotero search --collection ABC123 -q "topic"
```

### Batch Operations

```bash
# Process multiple searches with delays
for topic in "topic1" "topic2" "topic3"; do
    echo "Searching: $topic"
    pyzotero search -q "$topic" > "${topic// /_}.txt"
    sleep 1  # Prevent overwhelming Zotero
done
```

---

**For more information:**
- See [SKILL.md](SKILL.md) for command reference
- Check [QUICKSTART.md](QUICKSTART.md) for quick start guide
- Read [README.md](README.md) for full documentation

**Happy searching! 🚀**
