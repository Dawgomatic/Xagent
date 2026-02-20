# Pyzotero CLI Quick Start Guide

Get started with the pyzotero CLI in just a few minutes!

## 🚀 3-Minute Setup

### Step 1: Install pyzotero CLI (1 min)

Choose one of these methods:

#### Option A: pipx (Recommended for PEP 668 systems)
```bash
# Install pipx (if not already installed)
sudo apt install pipx  # Debian/Ubuntu
sudo pacman -S pipx    # Arch
sudo dnf install pipx  # Fedora

pipx ensurepath
pipx install "pyzotero[cli]"
```

#### Option B: pip (Simple)
```bash
pip install --user "pyzotero[cli]"
export PATH="$HOME/.local/bin:$PATH"
```

### Step 2: Enable Local Zotero Access (1 min)

**Required for CLI usage:**

1. Open Zotero 7 (or newer)
2. Go to **Edit > Preferences** (or **Zotero > Settings** on macOS)
3. Click on the **Advanced** tab
4. Check: **"Allow other applications on this computer to communicate with Zotero"**
5. **Click OK and restart Zotero**

⚠️ **Important:** Zotero must be running to use the CLI!

### Step 3: Try Your First Command (1 min)

```bash
# Make sure Zotero is running, then:
pyzotero listcollections
```

You should see a list of your Zotero collections.

```bash
# Try a search
pyzotero search -q "test"
```

---

## 📚 Common Tasks

### List Collections

```bash
pyzotero listcollections
```

Output:
```
Collection: My Library (ABC123)
  - Total items: 50

Collection: Research (DEF456)
  - Total items: 25

Collection: Articles (GHI789)
  - Total items: 15
```

### Search Your Library

```bash
# Basic search
pyzotero search -q "machine learning"

# Phrase search
pyzotero search -q "\"deep learning\""

# Search in specific fields
pyzotero search -q "python" --itemtype journalArticle
```

### Full-Text Search

Search in PDFs and attachments (requires Zotero to have indexed your PDFs):

```bash
# Full-text search
pyzotero search -q "neural networks" --fulltext
```

### Search Within Collection

```bash
# First, find your collection ID
pyzotero listcollections

# Then search within that collection
pyzotero search --collection ABC123 -q "topic"
```

### Filter by Item Type

```bash
# Search for books only
pyzotero search -q "python" --itemtype book

# Search for multiple types
pyzotero search -q "data" --itemtype book --itemtype journalArticle
```

---

## 🎯 CLI Quick Reference

### Search Commands

| Command | Description |
|---------|-------------|
| `pyzotero search -q "topic"` | Search library by topic |
| `pyzotero search -q "topic" --fulltext` | Search with full-text (includes PDFs) |
| `pyzotero search --collection ID -q "topic"` | Search within collection |
| `pyzotero search -q "topic" --itemtype TYPE` | Filter by item type |

### List Commands

| Command | Description |
|---------|-------------|
| `pyzotero listcollections` | List all collections |
| `pyzotero itemtypes` | List available item types |

### Output Formats

| Command | Description |
|---------|-------------|
| `pyzotero search -q "topic"` | Human-readable output |
| `pyzotero search -q "topic" --json` | JSON output (for scripts) |

---

## 📊 Example Workflows

### Find Recent Papers on a Topic

```bash
# Search for topic
pyzotero search -q "machine learning"

# Filter to journal articles
pyzotero search -q "machine learning" --itemtype journalArticle

# Search with full-text (find mentions in PDFs)
pyzotero search -q "transformer architecture" --fulltext
```

### Organize Research by Topic

```bash
# List collections
pyzotero listcollections

# Search for topic
pyzotero search -q "quantum computing"

# Search within specific collection
pyzotero search --collection RESEARCH -q "quantum"
```

### Quick Citation Lookup

```bash
# Search for specific paper
pyzotero search -q "\"Attention Is All You Need\""

# Get JSON output for processing
pyzotero search -q "\"Attention Is All You Need\"" --json
```

---

## 🔧 Advanced Usage

### Process JSON Output

```bash
# Get JSON output
pyzotero search -q "topic" --json

# Parse with jq
pyzotero search -q "topic" --json | jq '.[] | .title'

# Extract specific fields
pyzotero search -q "topic" --json | jq '.[] | {title: .title, date: .date}'

# Count results
pyzotero search -q "topic" --json | jq 'length'
```

### Search Multiple Topics

```bash
# Create a shell script to search multiple topics
for topic in "machine learning" "deep learning" "neural networks"; do
    echo "=== $topic ==="
    pyzotero search -q "$topic"
    echo ""
done
```

### Export Results to File

```bash
# Export to JSON file
pyzotero search -q "topic" --json > results.json

# Export titles to text file
pyzotero search -q "topic" | grep "Title:" > titles.txt
```

---

## 📖 Common Use Cases

### Morning Literature Scan

```bash
# Search for recent additions
pyzotero search --fulltext -q "new research"

# Check specific collection
pyzotero search --collection TO_READ -q ""
```

### Find References for Paper

```bash
# Search by topic
pyzotero search -q "reinforcement learning" --itemtype journalArticle

# Search within collection
pyzotero search --collection BIBLIOGRAPHY -q "attention"
```

### Quick Lookup During Writing

```bash
# Find specific paper
pyzotero search -q "\"paper title\""

# Get details in JSON
pyzotero search -q "\"paper title\"" --json
```

---

## 🆘 Quick Troubleshooting

### Command Not Found
```bash
# Update PATH
export PATH="$HOME/.local/bin:$PATH"
pipx ensurepath
```

### Connection Error
```bash
# Make sure Zotero is running
# Enable local API: Edit > Preferences > Advanced
# Check the box: "Allow other applications on this computer to communicate with Zotero"
# Restart Zotero
```

### No Results Found
```bash
# Verify Zotero has data
# Try different search terms
# Check if using correct collection-id
pyzotero listcollections
```

### Permission Error
```bash
# Use pipx instead of pip
pipx install "pyzotero[cli]"
```

---

## 📋 Item Types Reference

Use `pyzotero itemtypes` to see all available types. Common types:

- `book` - Books
- `journalArticle` - Journal articles
- `conferencePaper` - Conference papers
- `thesis` - Thesis or dissertation
- `report` - Reports
- `webpage` - Web pages
- `preprint` - Preprints

Example usage:
```bash
pyzotero search -q "topic" --itemtype journalArticle
pyzotero search -q "topic" --itemtype book
```

---

## 🎉 You're Ready!

You now have pyzotero CLI installed and configured. Here are some ideas to get you started:

**Daily Research:**
- Morning scan: `pyzotero search --fulltext -q "new"`
- Quick lookup: `pyzotero search -q "specific term"`

**Literature Review:**
- Search topic: `pyzotero search -q "machine learning"`
- Filter by type: `pyzotero search -q "topic" --itemtype journalArticle`
- Full-text search: `pyzotero search -q "topic" --fulltext`

**Citation Management:**
- Find paper: `pyzotero search -q "\"Paper Title\""`
- Get JSON: `pyzotero search -q "\"Paper Title\"" --json`

**Data Export:**
```bash
# Export to JSON
pyzotero search -q "topic" --json > export.json

# Count results
pyzotero search -q "topic" --json | jq '.[] | .title' | wc -l
```

---

## 📖 Next Steps

1. **Explore more examples:** See [EXAMPLES.md](EXAMPLES.md) for advanced workflows
2. **Read complete documentation:** Check [README.md](README.md) for features
3. **Install permanently:** `pipx install "pyzotero[cli]"`
4. **Create aliases:** Add shortcuts to your shell config for common searches

---

## 💡 Pro Tips

1. **Create shell aliases** for common searches:
   ```bash
   alias zotsearch='pyzotero search'
   alias zotlist='pyzotero listcollections'
   ```

2. **Use JSON output** for automation:
   ```bash
   pyzotero search -q "topic" --json | jq ...
   ```

3. **Combine with other tools:**
   ```bash
   pyzotero search -q "topic" --json | jq '.[] | .title' | xargs -I {} echo "Found: {}"
   ```

4. **Keep Zotero running** - CLI connects to local Zotero, not online API

---

**Happy searching! 📚✨**

Remember: **Zotero must be running** for the CLI to work.
