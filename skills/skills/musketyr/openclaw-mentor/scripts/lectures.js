#!/usr/bin/env node
/**
 * OpenClaw Mentor -- Lecture Manager CLI
 * Usage: node lectures.js <command> [args]
 *
 * Commands:
 *   list                          List all lectures with sizes
 *   read <slug>                   Print a lecture's content
 *   create <slug> [--file path]   Create a lecture from stdin or file
 *   edit <slug> [--file path]     Replace a lecture's content
 *   delete <slug>                 Delete a lecture
 *   rename <old-slug> <new-slug>  Rename a lecture file
 *   generate <topic>               Generate a lecture on a specific topic
 *   generate --all                Regenerate all lectures from memory
 *   sync                          Update relay specialties from current lectures
 */

const fs = require('fs');
const path = require('path');

const SKILL_DIR = path.resolve(__dirname, '..');
const LECTURES_DIR = process.env.LECTURES_DIR
  ? path.resolve(process.env.LECTURES_DIR)
  : path.join(SKILL_DIR, 'lectures');

const RELAY_URL = process.env.MENTOR_RELAY_URL || 'https://mentor.telegraphic.app';
const RELAY_TOKEN = process.env.MENTOR_RELAY_TOKEN;

// --- Helpers ---

function ensureDir() {
  if (!fs.existsSync(LECTURES_DIR)) {
    fs.mkdirSync(LECTURES_DIR, { recursive: true });
  }
}

function lecturePath(slug) {
  // Sanitize slug
  const safe = slug.replace(/\.md$/, '').replace(/[^a-z0-9-]/gi, '-').toLowerCase();
  return path.join(LECTURES_DIR, `${safe}.md`);
}

function listLectures() {
  ensureDir();
  const files = fs.readdirSync(LECTURES_DIR).filter(f => f.endsWith('.md')).sort();
  if (files.length === 0) {
    console.log('No lectures found.');
    console.log(`Directory: ${LECTURES_DIR}`);
    console.log('Run "node lectures.js generate" to create lectures from experience.');
    return;
  }
  console.log(`Lectures (${files.length}) in ${LECTURES_DIR}:\n`);
  for (const file of files) {
    const stat = fs.statSync(path.join(LECTURES_DIR, file));
    const slug = file.replace(/\.md$/, '');
    const kb = (stat.size / 1024).toFixed(1);
    console.log(`  ${slug.padEnd(30)} ${kb} KB`);
  }
}

function slugToTitle(slug) {
  return slug
    .split('-')
    .map(w => w.charAt(0).toUpperCase() + w.slice(1))
    .join(' ')
    .replace(/ And /g, ' & ');
}

// --- Commands ---

function cmdList() {
  listLectures();
}

function cmdRead() {
  const slug = args[0];
  if (!slug) {
    console.error('Usage: node lectures.js read <slug>');
    process.exit(1);
  }
  const fp = lecturePath(slug);
  if (!fs.existsSync(fp)) {
    console.error(`Lecture not found: ${slug}`);
    console.error(`Expected at: ${fp}`);
    process.exit(1);
  }
  process.stdout.write(fs.readFileSync(fp, 'utf-8'));
}

function cmdCreate() {
  const slug = args[0];
  if (!slug) {
    console.error('Usage: node lectures.js create <slug> [--file path]');
    console.error('  Without --file, reads content from stdin.');
    process.exit(1);
  }

  const fp = lecturePath(slug);
  if (fs.existsSync(fp)) {
    console.error(`Lecture already exists: ${slug}`);
    console.error('Use "edit" to update, or "delete" first.');
    process.exit(1);
  }

  const content = getContent();
  ensureDir();
  fs.writeFileSync(fp, content);
  console.log(`Created: ${slug} (${(content.length / 1024).toFixed(1)} KB)`);
}

function cmdEdit() {
  const slug = args[0];
  if (!slug) {
    console.error('Usage: node lectures.js edit <slug> [--file path]');
    console.error('  Without --file, reads content from stdin.');
    process.exit(1);
  }

  const fp = lecturePath(slug);
  if (!fs.existsSync(fp)) {
    console.error(`Lecture not found: ${slug}`);
    console.error('Use "create" for new lectures.');
    process.exit(1);
  }

  const content = getContent();
  fs.writeFileSync(fp, content);
  console.log(`Updated: ${slug} (${(content.length / 1024).toFixed(1)} KB)`);
}

function cmdDelete() {
  const slug = args[0];
  if (!slug) {
    console.error('Usage: node lectures.js delete <slug>');
    process.exit(1);
  }

  const fp = lecturePath(slug);
  if (!fs.existsSync(fp)) {
    console.error(`Lecture not found: ${slug}`);
    process.exit(1);
  }

  fs.unlinkSync(fp);
  console.log(`Deleted: ${slug}`);
}

function cmdRename() {
  const oldSlug = args[0];
  const newSlug = args[1];
  if (!oldSlug || !newSlug) {
    console.error('Usage: node lectures.js rename <old-slug> <new-slug>');
    process.exit(1);
  }

  const oldPath = lecturePath(oldSlug);
  const newPath = lecturePath(newSlug);

  if (!fs.existsSync(oldPath)) {
    console.error(`Lecture not found: ${oldSlug}`);
    process.exit(1);
  }
  if (fs.existsSync(newPath)) {
    console.error(`Target already exists: ${newSlug}`);
    process.exit(1);
  }

  fs.renameSync(oldPath, newPath);
  console.log(`Renamed: ${oldSlug} -> ${newSlug}`);
}

async function cmdGenerate() {
  // Delegate to generate-lectures.js
  const { execFileSync } = require('child_process');
  const genScript = path.join(__dirname, 'generate-lectures.js');
  const genArgs = [];

  const allFlag = args.includes('--all');

  if (allFlag) {
    // Full generation from all memory — no --topic flag
  } else {
    // Requires a topic description as remaining args
    const topic = args.filter(a => a !== '--all').join(' ').trim();
    if (!topic) {
      console.error('Usage: node lectures.js generate <topic description>');
      console.error('       node lectures.js generate --all');
      console.error('');
      console.error('Examples:');
      console.error('  node lectures.js generate "VPS hosting and server setup"');
      console.error('  node lectures.js generate "Docker troubleshooting patterns"');
      console.error('  node lectures.js generate --all   # regenerate everything from memory');
      process.exit(1);
    }
    genArgs.push('--topic', topic);
  }

  try {
    execFileSync(process.execPath, [genScript, ...genArgs], {
      env: process.env,
      stdio: 'inherit',
      timeout: 300000,
    });
  } catch (err) {
    console.error(`Generation failed: ${err.message}`);
    process.exit(1);
  }
}

async function cmdSync() {
  if (!RELAY_TOKEN) {
    console.error('MENTOR_RELAY_TOKEN is required for sync.');
    process.exit(1);
  }

  ensureDir();
  const files = fs.readdirSync(LECTURES_DIR).filter(f => f.endsWith('.md')).sort();
  if (files.length === 0) {
    console.error('No lectures found. Generate or create some first.');
    process.exit(1);
  }

  const specialties = files.map(f => slugToTitle(f.replace(/\.md$/, '')));
  console.log(`Syncing ${specialties.length} specialties to relay: ${specialties.join(', ')}`);

  try {
    const res = await fetch(`${RELAY_URL}/api/mentor/profile`, {
      method: 'PATCH',
      headers: {
        'Content-Type': 'application/json',
        'Authorization': `Bearer ${RELAY_TOKEN}`,
      },
      body: JSON.stringify({ specialties }),
    });

    if (res.ok) {
      const data = await res.json();
      console.log('Relay specialties updated.');
    } else {
      const err = await res.text();
      console.error(`Sync failed (${res.status}): ${err}`);
      process.exit(1);
    }
  } catch (err) {
    console.error(`Could not reach relay: ${err.message}`);
    process.exit(1);
  }
}

// --- Content reader (stdin or --file) ---

function getContent() {
  const fileIdx = args.indexOf('--file');
  if (fileIdx >= 0 && args[fileIdx + 1]) {
    const filePath = path.resolve(args[fileIdx + 1]);
    if (!fs.existsSync(filePath)) {
      console.error(`File not found: ${filePath}`);
      process.exit(1);
    }
    return fs.readFileSync(filePath, 'utf-8');
  }

  // Read from stdin
  try {
    return fs.readFileSync(0, 'utf-8');
  } catch {
    console.error('No content provided. Use --file or pipe content via stdin.');
    process.exit(1);
  }
}

// --- Main ---

const [command, ...args] = process.argv.slice(2);

if (!command || command === 'help' || command === '--help') {
  console.log(`OpenClaw Mentor -- Lecture Manager

Usage: node lectures.js <command> [args]

Commands:
  list                          List all lectures with sizes
  read <slug>                   Print a lecture's content
  create <slug> [--file path]   Create a lecture (from stdin or file)
  edit <slug> [--file path]     Replace a lecture (from stdin or file)
  delete <slug>                 Delete a lecture
  rename <old> <new>            Rename a lecture
  generate <topic>              Generate a lecture on a specific topic
  generate --all                Regenerate all lectures from memory
  sync                          Update relay specialties from lectures

Environment:
  LECTURES_DIR          Lecture directory (default: ./lectures/)
  MENTOR_RELAY_URL      Relay server URL
  MENTOR_RELAY_TOKEN    Mentor API token (for sync)
  OPENCLAW_GATEWAY_URL  Gateway URL (for generate)
  OPENCLAW_GATEWAY_TOKEN  Gateway token (for generate)

Examples:
  node lectures.js list
  node lectures.js read memory-management
  echo "# Docker Tips\\n..." | node lectures.js create docker-tips
  node lectures.js create docker-tips --file /path/to/lecture.md
  node lectures.js edit docker-tips --file /path/to/updated.md
  node lectures.js delete n8n-workflows
  node lectures.js rename old-name new-name
  node lectures.js generate "CI/CD pipelines"
  node lectures.js generate --all
  node lectures.js sync`);
  process.exit(0);
}

switch (command) {
  case 'list': return cmdList();
  case 'read': case 'show': case 'cat': return cmdRead();
  case 'create': case 'add': case 'new': return cmdCreate();
  case 'edit': case 'update': case 'replace': return cmdEdit();
  case 'delete': case 'rm': case 'remove': return cmdDelete();
  case 'rename': case 'mv': case 'move': return cmdRename();
  case 'generate': case 'gen': return cmdGenerate();
  case 'sync': return cmdSync();
  default:
    console.error(`Unknown command: ${command}`);
    console.error('Run "node lectures.js help" for usage.');
    process.exit(1);
}
