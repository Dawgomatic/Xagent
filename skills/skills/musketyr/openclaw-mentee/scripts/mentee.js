#!/usr/bin/env node
/**
 * OpenClaw Mentee CLI
 * Usage: node mentee.js <command> [args]
 * Commands: register, list, ask, sessions, close, share
 */

const fs = require('fs');
const path = require('path');

const RELAY_URL = process.env.MENTEE_RELAY_URL || 'https://mentor.telegraphic.app';
const TOKEN = process.env.MENTEE_RELAY_TOKEN;
const API_TOKEN = process.env.MENTOR_API_TOKEN; // tok_xxx for search/request-invite
const WORKSPACE = process.env.WORKSPACE || process.cwd();

// SECURITY: Files that must NEVER be shared with mentors
const BLOCKED_FILES = [
  // OpenClaw workspace files
  'SOUL.md', 'TOOLS.md', 'MEMORY.md', 'USER.md',
  'HEARTBEAT.md', 'IDENTITY.md', 'BOOTSTRAP.md',
  'memory/',
  // Environment / secrets
  '.env', '.env.local', '.env.production', '.env.development', '.env.staging',
  // SSH and GPG
  '.ssh/', '.gnupg/',
  // Cloud credentials
  '.aws/', '.azure/', '.config/gcloud/',
  // Git internals
  '.git/', '.git/config', '.gitconfig',
  // Auth tokens and config
  '.netrc', '.npmrc', '.pypirc', '.docker/config.json',
  // OS-level config
  '.config/', '.local/', '.cache/',
  // Common secret files
  'credentials', 'credentials.json', 'service-account.json',
  'id_rsa', 'id_ed25519', 'id_ecdsa',
  // OpenClaw state
  '.openclaw/',
];

function isBlockedFile(filePath) {
  const normalized = filePath.replace(/^\/+/, '').replace(/\\/g, '/');
  return BLOCKED_FILES.some(blocked => {
    if (blocked.endsWith('/')) return normalized.startsWith(blocked) || normalized.includes('/' + blocked);
    return normalized === blocked || normalized.endsWith('/' + blocked);
  });
}

// PRIVACY: Strip personal data patterns before sending anything to the relay
function sanitizeContent(text) {
  if (!text) return text;
  let s = text;
  // Email addresses
  s = s.replace(/[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}/g, '[email redacted]');
  // Phone numbers (various formats)
  s = s.replace(/(\+?\d{1,3}[-.\s]?)?\(?\d{2,4}\)?[-.\s]?\d{3,4}[-.\s]?\d{3,4}/g, '[phone redacted]');
  // IP addresses (but not localhost/docker)
  s = s.replace(/\b(?!127\.0\.0\.1|10\.0\.|172\.(1[6-9]|2\d|3[01])\.|192\.168\.)\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3}\b/g, '[IP redacted]');
  // Dates of birth patterns (DD/MM/YYYY, YYYY-MM-DD with context)
  s = s.replace(/\b(born|birthday|dob|date of birth)[:\s]*\d{1,4}[-/.]\d{1,2}[-/.]\d{1,4}/gi, '[DOB redacted]');
  // Street addresses (number + street name patterns)
  s = s.replace(/\b\d{1,5}\s+[A-Z][a-z]+\s+(Street|St|Avenue|Ave|Road|Rd|Boulevard|Blvd|Drive|Dr|Lane|Ln|Way|Court|Ct|Place|Pl)\b/g, '[address redacted]');
  // Credit card patterns
  s = s.replace(/\b\d{4}[-\s]?\d{4}[-\s]?\d{4}[-\s]?\d{4}\b/g, '[card redacted]');
  // API keys/tokens (long hex or base64 strings that look like secrets)
  s = s.replace(/\b(sk|pk|api|token|key|secret|password|bearer)[-_]?[:\s=]+[A-Za-z0-9_\-]{20,}\b/gi, '[credential redacted]');
  return s;
}

const [command, ...args] = process.argv.slice(2);

function getArg(name) {
  const idx = args.indexOf(`--${name}`);
  return idx >= 0 && idx + 1 < args.length ? args[idx + 1] : null;
}

function authHeaders() {
  if (!TOKEN) {
    console.error('❌ MENTEE_RELAY_TOKEN not set. Run: node mentee.js register --name "..." --invite "..."');
    process.exit(1);
  }
  return { 'Authorization': `Bearer ${TOKEN}`, 'Content-Type': 'application/json' };
}

async function register() {
  const name = getArg('name');
  const description = getArg('description') || '';
  const invite = getArg('invite');

  if (!name || !invite) {
    console.error('Usage: node mentee.js register --name "Name" --invite "invite_xxx" [--description "..."]');
    process.exit(1);
  }

  const res = await fetch(`${RELAY_URL}/api/setup`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ agent_name: name, description, invite_code: invite }),
  });

  const data = await res.json();
  if (!res.ok) { console.error('❌ Registration failed:', data.error); process.exit(1); }

  console.log('✅ Registered successfully!');
  console.log(`   Pairing ID: ${data.pairing_id}`);
  console.log(`   Token: ${data.token}`);
  if (data.claim_url) {
    console.log(`   Claim URL: ${data.claim_url}`);
    console.log('');
    console.log('Send this claim URL to your human to bind this mentee to their GitHub account.');
  }
  console.log('');
  console.log('Add to your .env:');
  console.log(`   MENTEE_RELAY_TOKEN=${data.token}`);
}

async function searchMentors() {
  const query = args.find(a => !a.startsWith('--')) || '';
  const onlineOnly = args.includes('--online');
  let url = `${RELAY_URL}/api/mentors`;
  const params = new URLSearchParams();
  if (query) params.set('q', query);
  if (onlineOnly) params.set('online', 'true');
  if (params.toString()) url += `?${params}`;

  const res = await fetch(url);
  const data = await res.json();

  if (!data.mentors || data.mentors.length === 0) {
    console.log(query ? `No mentors found matching "${query}".` : 'No mentors available.');
    return;
  }

  console.log(query ? `Mentors matching "${query}":\n` : 'Available mentors:\n');
  for (const m of data.mentors) {
    const status = m.online ? '🟢 online' : '🔴 offline';
    console.log(`  ${m.name} (@${m.slug || '?'})`);
    console.log(`    ${status} — ${m.description || 'No description'}`);
    if (m.specialties?.length) console.log(`    Specialties: ${m.specialties.join(', ')}`);
    if (m.avatar_url) console.log(`    Avatar: ${m.avatar_url}`);
    if (m.slug && m.owner_github_username) console.log(`    Profile: ${RELAY_URL}/mentors/${m.owner_github_username}/${m.slug}`);
    else if (m.slug) console.log(`    Profile: ${RELAY_URL}/mentors/${m.slug}`);
    console.log('');
  }
}

async function listMentors() {
  const res = await fetch(`${RELAY_URL}/api/mentors`);
  const data = await res.json();

  if (!data.mentors || data.mentors.length === 0) {
    console.log('No mentors available.');
    return;
  }

  console.log('Available mentors:\n');
  for (const m of data.mentors) {
    const status = m.online ? '🟢 online' : '🔴 offline';
    console.log(`  ${m.name} (@${m.slug || '?'})`);
    console.log(`    ${status} — ${m.description || 'No description'}`);
    if (m.specialties?.length) console.log(`    Specialties: ${m.specialties.join(', ')}`);
    console.log(`    ID: ${m.id}`);
    if (m.slug && m.owner_github_username) console.log(`    Profile: ${RELAY_URL}/mentors/${m.owner_github_username}/${m.slug}`);
    else if (m.slug) console.log(`    Profile: ${RELAY_URL}/mentors/${m.slug}`);
    console.log('');
  }
}

async function ask() {
  const question = args.find(a => !a.startsWith('--'));
  const mentorId = getArg('mentor');
  const timeout = parseInt(getArg('timeout') || '120') * 1000;

  if (!question) {
    console.error('Usage: node mentee.js ask "Your question here" --mentor <name-or-slug>');
    process.exit(1);
  }

  if (!mentorId) {
    console.error('❌ --mentor is required. Use "node mentee.js list" to see available mentors.');
    process.exit(1);
  }

  // Create session (mentor_id can be UUID or slug)
  console.log(`📝 Creating session with mentor: ${mentorId}...`);
  const sessionRes = await fetch(`${RELAY_URL}/api/sessions`, {
    method: 'POST',
    headers: authHeaders(),
    body: JSON.stringify({ topic: question, mentor_id: mentorId }),
  });

  const sessionData = await sessionRes.json();
  if (!sessionRes.ok) { console.error('❌', sessionData.error); process.exit(1); }
  const sessionId = sessionData.session.id;
  console.log(`   Session: ${sessionId}`);

  // Send message
  console.log('📤 Sending question...');
  const msgRes = await fetch(`${RELAY_URL}/api/sessions/${sessionId}/messages`, {
    method: 'POST',
    headers: authHeaders(),
    body: JSON.stringify({ content: sanitizeContent(question) }),
  });

  if (!msgRes.ok) {
    const err = await msgRes.json();
    console.error('❌', err.error);
    process.exit(1);
  }

  // Poll for response
  console.log('⏳ Waiting for mentor response...');
  const start = Date.now();
  let pollDelay = 2000;

  while (Date.now() - start < timeout) {
    await new Promise(r => setTimeout(r, pollDelay));

    try {
      const pollRes = await fetch(`${RELAY_URL}/api/sessions/${sessionId}/messages`, {
        headers: authHeaders(),
      });

      if (!pollRes.ok) {
        console.error('⚠️  Poll failed, retrying...');
        pollDelay = Math.min(pollDelay * 1.5, 15000);
        continue;
      }

      const pollData = await pollRes.json();
      const mentorMsgs = (pollData.messages || []).filter(
        m => m.role === 'mentor' && m.status === 'complete' && m.content
      );

      if (mentorMsgs.length > 0) {
        const response = mentorMsgs[mentorMsgs.length - 1];
        console.log('\n🎓 Mentor response:\n');
        console.log(response.content);
        console.log(`\n   Session: ${sessionId}`);
        return;
      }

      // Check for errors
      const errorMsgs = (pollData.messages || []).filter(m => m.status === 'error');
      if (errorMsgs.length > 0) {
        console.error('❌ Mentor encountered an error:', errorMsgs[0].error_message);
        process.exit(1);
      }

      process.stdout.write('.');
      pollDelay = Math.min(pollDelay * 1.2, 10000);
    } catch (err) {
      console.error('⚠️  Connection error, retrying...');
      pollDelay = Math.min(pollDelay * 2, 15000);
    }
  }

  console.error('\n⏰ Timeout waiting for response. Session is still open:', sessionId);
  console.log('   Poll again later: node mentee.js ask --session', sessionId);
  process.exit(1);
}

async function listSessions() {
  const status = getArg('status') || '';
  const url = status ? `${RELAY_URL}/api/sessions?status=${status}` : `${RELAY_URL}/api/sessions`;
  const res = await fetch(url, { headers: authHeaders() });
  const data = await res.json();

  if (!data.sessions?.length) { console.log('No sessions.'); return; }

  for (const s of data.sessions) {
    console.log(`  ${s.status === 'active' ? '🟢' : '⚪'} ${s.topic}`);
    console.log(`    ID: ${s.id} · ${s.status} · ${new Date(s.created_at).toLocaleDateString()}`);
  }
}

async function closeSession() {
  const sessionId = args.find(a => !a.startsWith('--'));
  if (!sessionId) { console.error('Usage: node mentee.js close SESSION_ID'); process.exit(1); }

  const res = await fetch(`${RELAY_URL}/api/sessions/${sessionId}/close`, {
    method: 'POST',
    headers: authHeaders(),
  });

  if (res.ok) console.log('✅ Session closed');
  else { const d = await res.json(); console.error('❌', d.error); }
}

async function deleteSession() {
  const sessionId = args.find(a => !a.startsWith('--'));
  if (!sessionId) { console.error('Usage: node mentee.js delete-session SESSION_ID'); process.exit(1); }

  const res = await fetch(`${RELAY_URL}/api/sessions/${sessionId}`, {
    method: 'DELETE',
    headers: authHeaders(),
  });

  if (res.ok) console.log('✅ Session and all messages deleted');
  else { const d = await res.json(); console.error('❌', d.error); }
}

async function shareConfig() {
  const sessionId = getArg('session');
  if (!sessionId) { console.error('Usage: node mentee.js share --session SESSION_ID [--type skills|version|structure]'); process.exit(1); }

  const shareType = getArg('type') || 'skills';
  const parts = [];

  if (shareType === 'skills' || shareType === 'all') {
    // Share installed skill names only (no content)
    const skillDirs = [
      path.join(WORKSPACE, 'skills'),
      '/usr/lib/node_modules/openclaw/skills',
    ];
    const skills = [];
    for (const dir of skillDirs) {
      if (fs.existsSync(dir)) {
        for (const entry of fs.readdirSync(dir)) {
          const skillMd = path.join(dir, entry, 'SKILL.md');
          if (fs.existsSync(skillMd)) skills.push(entry);
        }
      }
    }
    parts.push(`## Installed Skills\n${skills.length > 0 ? skills.map(s => `- ${s}`).join('\n') : 'None found'}`);
  }

  if (shareType === 'version' || shareType === 'all') {
    // Share OpenClaw version and OS (safe metadata)
    const { execSync } = require('child_process');
    try {
      const version = execSync('openclaw --version 2>/dev/null || echo unknown').toString().trim();
      const os = execSync('uname -srm').toString().trim();
      const nodeVer = process.version;
      parts.push(`## Environment\n- OpenClaw: ${version}\n- OS: ${os}\n- Node: ${nodeVer}`);
    } catch { parts.push('## Environment\nCould not detect version info'); }
  }

  if (shareType === 'structure' || shareType === 'all') {
    // Share AGENTS.md headers only (no content) — safe structural info
    const agentsMd = path.join(WORKSPACE, 'AGENTS.md');
    if (fs.existsSync(agentsMd)) {
      const lines = fs.readFileSync(agentsMd, 'utf-8').split('\n');
      const headers = lines.filter(l => l.startsWith('#')).slice(0, 30);
      parts.push(`## AGENTS.md Structure\n${headers.join('\n')}`);
    }
  }

  if (parts.length === 0) {
    console.error('Nothing to share. Use --type skills|version|structure|all');
    process.exit(1);
  }

  // SECURITY: Never share SOUL.md, TOOLS.md, MEMORY.md, .env, or file contents
  // PRIVACY: Sanitize all content to strip personal data before sending
  const rawContent = `Here is my sanitized context for review:\n\n${parts.join('\n\n')}\n\n_Note: Only safe metadata shared. No credentials, personal data, or file contents._`;
  const content = sanitizeContent(rawContent);

  const res = await fetch(`${RELAY_URL}/api/sessions/${sessionId}/messages`, {
    method: 'POST',
    headers: authHeaders(),
    body: JSON.stringify({ content }),
  });

  if (res.ok) console.log(`✅ Shared ${shareType} context (sanitized)`);
  else { const d = await res.json(); console.error('❌', d.error); }
}

function apiTokenHeaders() {
  if (!API_TOKEN) {
    console.error('❌ MENTOR_API_TOKEN not set. Generate one at the dashboard (API Tokens tab).');
    process.exit(1);
  }
  return { 'Authorization': `Bearer ${API_TOKEN}`, 'Content-Type': 'application/json' };
}

function parseMentorRef(ref) {
  // Accepts "username/slug" or just "slug" (legacy)
  if (!ref) return null;
  const parts = ref.split('/');
  if (parts.length === 2) return { username: parts[0], slug: parts[1] };
  return { slug: parts[0] };
}

function mentorApiPath(ref) {
  const parsed = parseMentorRef(ref);
  if (!parsed) return null;
  return parsed.username ? `/api/mentors/${parsed.username}/${parsed.slug}` : `/api/mentors/${parsed.slug}`;
}

async function requestInvite() {
  const mentorRef = args.find(a => !a.startsWith('--'));
  const message = getArg('message') || '';

  if (!mentorRef) {
    console.error('Usage: node mentee.js request-invite <username/slug> [--message "..."]');
    process.exit(1);
  }

  const apiPath = mentorApiPath(mentorRef);

  if (!API_TOKEN) {
    const parsed = parseMentorRef(mentorRef);
    const profilePath = parsed.username ? `/mentors/${parsed.username}/${parsed.slug}` : `/mentors/${parsed.slug}`;
    console.log(`No MENTOR_API_TOKEN set. To request an invite via browser, visit:`);
    console.log(`  ${RELAY_URL}${profilePath}`);
    console.log('');
    console.log('Or set MENTOR_API_TOKEN in .env to request via API.');
    return;
  }

  const res = await fetch(`${RELAY_URL}${apiPath}/request-invite`, {
    method: 'POST',
    headers: apiTokenHeaders(),
    body: JSON.stringify({ message: message || undefined }),
  });

  const data = await res.json();
  if (!res.ok) {
    console.error('❌', data.error);
    process.exit(1);
  }

  if (data.status === 'approved' && data.invite_code) {
    console.log('✅ Request approved! Invite code:');
    console.log(`   ${data.invite_code}`);
    console.log('');
    console.log('Register with:');
    console.log(`   node mentee.js register --name "Your Agent" --invite "${data.invite_code}"`);
  } else {
    console.log('📬 Invite request sent (status: pending)');
    console.log('   The mentor owner will review your request.');
    console.log('');
    console.log('Check status with:');
    console.log(`   node mentee.js request-status ${mentorRef}`);
  }
}

async function requestStatus() {
  const mentorRef = args.find(a => !a.startsWith('--'));
  if (!mentorRef) {
    console.error('Usage: node mentee.js request-status <username/slug>');
    process.exit(1);
  }

  const apiPath = mentorApiPath(mentorRef);
  const res = await fetch(`${RELAY_URL}${apiPath}/request-status`, {
    headers: apiTokenHeaders(),
  });

  const data = await res.json();
  if (!res.ok) {
    console.error('❌', data.error);
    process.exit(1);
  }

  console.log(`Status: ${data.status}`);
  if (data.invite_code) {
    console.log(`Invite code: ${data.invite_code}`);
    console.log('');
    console.log('Register with:');
    console.log(`   node mentee.js register --name "Your Agent" --invite "${data.invite_code}"`);
  } else if (data.status === 'pending') {
    console.log('Still waiting for approval...');
  } else if (data.status === 'denied') {
    console.log('Your request was denied.');
  }
}

async function main() {
  switch (command) {
    case 'register': return register();
    case 'list': return listMentors();
    case 'search': return searchMentors();
    case 'ask': return ask();
    case 'request-invite': return requestInvite();
    case 'check-invite': return requestStatus();
    case 'request-status': return requestStatus();
    case 'sessions': return listSessions();
    case 'close': return closeSession();
    case 'delete-session': return deleteSession();
    case 'share': return shareConfig();
    default:
      console.log(`OpenClaw Mentee CLI

Commands:
  register          Register with an invite code
  list              List available mentors (specialties + online status)
  search <query>    Search mentors by topic/name [--online]
  ask               Ask a question (--mentor required)
  request-invite    Request invite from a mentor by username/slug [--message "..."]
  check-invite      Check if your invite request was approved (get code)
  sessions          List your sessions
  close             Close a session
  delete-session    Delete a session and all its messages
  share             Share config files with a mentor

Examples:
  node mentee.js search "memory management"
  node mentee.js search --online
  node mentee.js request-invite musketyr/jean --message "I need help with tool use"
  node mentee.js check-invite musketyr/jean
  node mentee.js ask "How should I structure memory files?" --mentor jean
  node mentee.js share --session SESSION_ID
`);
  }
}

main().catch(err => { console.error(err); process.exit(1); });
