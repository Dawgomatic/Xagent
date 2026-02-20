#!/usr/bin/env node
/**
 * Session State Tracker - Core Module
 *
 * Provides utilities for reading, writing, and discovering session state
 * from SESSION_STATE.md and indexed session transcripts.
 */

const fs = require('fs');
const path = require('path');
const yaml = require('js-yaml');

const WORKSPACE = process.env.OPENCLAW_WORKSPACE || process.cwd();
const STATE_FILE = path.resolve(WORKSPACE, 'SESSION_STATE.md');

/**
 * Parse YAML frontmatter and body from a Markdown file.
 * Uses js-yaml for robust parsing.
 * @param {string} content
 * @returns {object} { frontmatter: object, body: string }
 */
function parseFile(content) {
  // Split on first '---' delimiter
  const parts = content.split(/^---$/m);
  if (parts.length < 3) {
    throw new Error('Invalid SESSION_STATE.md: missing YAML frontmatter delimiters');
  }
  const fmStr = parts[1].trim();
  const body = parts.slice(2).join('---').trim(); // Rejoin in case body contains '---'

  try {
    const frontmatter = yaml.load(fmStr);
    if (!frontmatter || typeof frontmatter !== 'object') {
      throw new Error('Invalid YAML frontmatter');
    }
    return { frontmatter, body };
  } catch (err) {
    err.message = `Failed to parse YAML frontmatter: ${err.message}`;
    throw err;
  }
}

/**
 * Read SESSION_STATE.md and return full state (frontmatter + body).
 * @returns {Promise<object>} state object with all frontmatter fields and optional body
 */
async function readState() {
  try {
    const content = fs.readFileSync(STATE_FILE, 'utf-8');
    const parsed = parseFile(content);
    return { ...parsed.frontmatter, body: parsed.body };
  } catch (err) {
    if (err.code === 'ENOENT') {
      return null;
    }
    throw err;
  }
}

/**
 * Write SESSION_STATE.md with given updates.
 * Preserves existing fields not mentioned in updates.
 * Updates `updated` timestamp automatically.
 *
 * @param {object} updates - partial state updates
 * @returns {Promise<void>}
 */
async function writeState(updates) {
  const current = await readState() || {};
  // Separate body from frontmatter fields
  const { body: currentBody, ...currentFm } = current;
  const merged = { ...currentFm, ...updates, updated: new Date().toISOString() };
  const finalBody = updates.body !== undefined ? updates.body : currentBody;

  // Reconstruct file
  const fmYaml = yaml.dump(merged, {
    lineWidth: -1, // no wrapping
    indent: 2
  });
  const output = `---\n${fmYaml}---\n${finalBody ? finalBody + '\n' : ''}`;

  // Ensure directory exists
  fs.mkdirSync(path.dirname(STATE_FILE), { recursive: true });
  fs.writeFileSync(STATE_FILE, output, 'utf-8');
}

/**
 * Discover current session state from indexed transcripts.
 * Uses memory_search on sessions to find recent mentions of project/task.
 *
 * @param {object} memorySearch - the memory_search tool function (injected)
 * @returns {Promise<object>} synthesized state object (includes body)
 */
async function discoverFromSessions(memorySearch) {
  // Query for recent task-related snippets
  const query = 'project|task|working on|next step|implementing|building';
  const results = await memorySearch({
    query,
    sources: ['sessions'],
    limit: 10,
    minScore: 0.3
  });

  if (!results || results.length === 0) {
    return {
      project: '',
      task: '',
      status: '',
      last_action: '',
      next_steps: [],
      updated: new Date().toISOString(),
      body: 'Auto-discovered from session transcripts (no clear task found)'
    };
  }

  // Very naive synthesis: take the top result's snippet as task hint
  const top = results[0];
  const snippet = top.text.replace(/\n/g, ' ').substring(0, 200);
  const projectMatch = snippet.match(/(?:project|working on)\s+([A-Za-z0-9_-]+)/i);
  const taskMatch = snippet.match(/(?:task|implementing|building)\s+([A-Za-z0-9_-]+(?:\s+[A-Za-z0-9_-]+)*)/i);

  const project = projectMatch ? projectMatch[1] : (top.file.includes('session-state-tracker') ? 'session-state-tracker' : '');
  const task = taskMatch ? taskMatch[1] : (snippet.slice(0, 80));

  const body = `Top snippet: ${snippet}\n\nSource: ${top.file}~${top.fromLine}-${top.toLine}`;

  return {
    project: project || '',
    task: task || 'Discovered from recent conversation',
    status: 'active',
    last_action: `Discovered from ${results.length} session snippets`,
    next_steps: [],
    updated: new Date().toISOString(),
    body
  };
}

module.exports = {
  readState,
  writeState,
  discoverFromSessions,
  parseFile,
  STATE_FILE
};
