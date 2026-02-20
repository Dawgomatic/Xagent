#!/usr/bin/env node

const fs = require('fs');
const path = require('path');
const os = require('os');

const DEFAULT_SESSION_PATH = path.join(os.homedir(), '.openclaw', 'agents', 'main', 'sessions');
const DEFAULT_CONFIG_PATH = path.join(__dirname, '..', 'leak-check.json');

function parseArgs(argv) {
  const args = {
    format: 'discord',
    config: DEFAULT_CONFIG_PATH,
    help: false
  };
  let i = 0;

  while (i < argv.length) {
    if (argv[i] === '--help' || argv[i] === '-h') {
      args.help = true;
    } else if (argv[i] === '--format' && i + 1 < argv.length) {
      args.format = argv[++i];
    } else if (argv[i] === '--config' && i + 1 < argv.length) {
      args.config = argv[++i];
    }
    i++;
  }

  return args;
}

function printHelp() {
  console.log('Usage: node leak-check.js [options]');
  console.log('');
  console.log('Scan OpenClaw session logs for leaked credentials.');
  console.log('');
  console.log('Options:');
  console.log('  --format <type>    Output format: discord (default) or json');
  console.log('  --config <path>    Path to credential config file');
  console.log(`                     Default: leak-check.json in skill root`);
  console.log('  --help, -h         Show this help message');
  console.log('');
  console.log('Examples:');
  console.log('  node leak-check.js                        # Discord format');
  console.log('  node leak-check.js --format json           # JSON output');
  console.log('  node leak-check.js --config /path/to.json  # Custom config');
}

function loadCredentials(configPath) {
  if (!fs.existsSync(configPath)) {
    console.error(`Error: Config file not found: ${configPath}`);
    process.exit(1);
  }

  const raw = JSON.parse(fs.readFileSync(configPath, 'utf-8'));
  // Filter out entries with empty search values
  return raw.filter(entry => entry.search && entry.search.trim() !== '');
}

function escapeRegex(str) {
  return str.replace(/[.*+?^${}()|[\]\\]/g, '\\$&');
}

function buildMatcher(search) {
  const hasAsterisk = search.includes('*');

  if (!hasAsterisk) {
    // Plain substring match
    return (text) => text.includes(search);
  }

  // Convert wildcard to regex where * matches non-whitespace/non-quote chars
  // Capped at 200 chars to avoid matching inside large base64 blobs
  const TOKEN_CHAR = '[^\\s"]{0,200}';
  const parts = search.split('*');
  const regexStr = parts.map(escapeRegex).join(TOKEN_CHAR);
  const regex = new RegExp(regexStr);
  return (text) => regex.test(text);
}

function findSessionFiles(dirPath) {
  const files = [];

  function walk(dir) {
    let entries;
    try {
      entries = fs.readdirSync(dir, { withFileTypes: true });
    } catch {
      return;
    }

    for (const entry of entries) {
      const fullPath = path.join(dir, entry.name);

      if (entry.isDirectory()) {
        walk(fullPath);
      } else if (entry.isFile() && entry.name.includes('.jsonl')) {
        // Catches .jsonl, .jsonl.deleted.*, .jsonl.reset.*
        files.push(fullPath);
      }
    }
  }

  walk(dirPath);
  return files;
}

function extractSessionGuid(filePath) {
  const basename = path.basename(filePath);
  // UUID is the part before .jsonl
  const match = basename.match(/^([0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12})/i);
  return match ? match[1] : basename.split('.')[0];
}

function isRealProvider(provider) {
  return provider && provider !== 'openclaw';
}

function scanFile(filePath, credentials, matchers) {
  const content = fs.readFileSync(filePath, 'utf-8');
  const lines = content.split('\n');
  const sessionGuid = extractSessionGuid(filePath);

  // Track the last real provider/model seen in this file
  let lastRealProvider = 'unknown';
  let lastRealModel = 'unknown';

  // Track matches: Map<credName, { session, timestamp, provider }>
  const matches = new Map();

  for (const line of lines) {
    if (!line.trim()) continue;

    // Update provider tracking by parsing JSON for relevant lines
    // We check for provider-related keywords first to avoid parsing every line
    if (line.includes('"model_change"') || line.includes('"provider"')) {
      try {
        const record = JSON.parse(line);

        // model_change events
        if (record.type === 'model_change' && isRealProvider(record.provider)) {
          lastRealProvider = record.provider;
          if (record.model) lastRealModel = record.model;
        }

        // Cost-bearing message entries
        if (record.type === 'message' && record.message) {
          const msg = record.message;
          if (isRealProvider(msg.provider) && msg.usage?.cost?.total > 0) {
            lastRealProvider = msg.provider;
            if (msg.model) lastRealModel = msg.model;
          }
        }
      } catch {
        // Skip malformed lines
      }
    }

    // Check each credential pattern against raw line text
    for (let i = 0; i < credentials.length; i++) {
      const cred = credentials[i];
      if (matches.has(cred.name)) continue; // Already found in this file, skip

      if (matchers[i](line)) {
        // Match found — parse JSON to get details
        let timestamp = null;
        let provider = lastRealProvider;
        let model = lastRealModel;

        try {
          const record = JSON.parse(line);
          timestamp = record.timestamp || null;

          // If this line itself has a real provider, use it
          if (record.message && isRealProvider(record.message.provider) && record.message.provider !== 'openclaw') {
            provider = record.message.provider;
            if (record.message.model) model = record.message.model;
          }
        } catch {
          // If we can't parse, use tracked provider
        }

        matches.set(cred.name, {
          credential: cred.name,
          session: sessionGuid,
          timestamp: timestamp,
          provider: provider,
          model: model
        });
      }
    }
  }

  return Array.from(matches.values());
}

function formatDiscordOutput(leaks, filesScanned, credsChecked) {
  const lines = ['🔐 **Credential Leak Check**'];

  if (leaks.length === 0) {
    lines.push(`✅ No leaked credentials found (checked ${filesScanned} files, ${credsChecked} credentials)`);
    return lines.join('\n');
  }

  lines.push('');
  lines.push(`⚠️ **${leaks.length} leaked credential${leaks.length === 1 ? '' : 's'} found**`);

  // Group by credential name
  const grouped = {};
  for (const leak of leaks) {
    if (!grouped[leak.credential]) grouped[leak.credential] = [];
    grouped[leak.credential].push(leak);
  }

  // Summary counts by credential name
  lines.push('');
  for (const [credName, entries] of Object.entries(grouped)) {
    lines.push(`• **${credName}**: ${entries.length} session${entries.length === 1 ? '' : 's'}`);
  }

  // Detail per credential
  for (const [credName, entries] of Object.entries(grouped)) {
    lines.push('');
    lines.push(`**${credName}**`);
    for (const entry of entries) {
      const shortSession = entry.session.slice(0, 8);
      const ts = entry.timestamp ? formatTimestamp(entry.timestamp) : 'unknown time';
      const modelInfo = entry.model && entry.model !== 'unknown' ? `${entry.provider}/${entry.model}` : entry.provider;
      lines.push(`• \`${shortSession}\` | ${ts} | ${modelInfo}`);
    }
  }

  return lines.join('\n');
}

function formatTimestamp(ts) {
  try {
    const d = new Date(ts);
    const year = d.getUTCFullYear();
    const month = String(d.getUTCMonth() + 1).padStart(2, '0');
    const day = String(d.getUTCDate()).padStart(2, '0');
    const hours = String(d.getUTCHours()).padStart(2, '0');
    const minutes = String(d.getUTCMinutes()).padStart(2, '0');
    return `${year}-${month}-${day} ${hours}:${minutes} UTC`;
  } catch {
    return 'unknown time';
  }
}

function formatJsonOutput(leaks, filesScanned, credsChecked) {
  return JSON.stringify({
    leaks: leaks,
    summary: {
      filesScanned: filesScanned,
      credentialsChecked: credsChecked,
      leaksFound: leaks.length
    }
  }, null, 2);
}

function main() {
  const args = parseArgs(process.argv.slice(2));

  if (args.help) {
    printHelp();
    process.exit(0);
  }

  const credentials = loadCredentials(args.config);

  if (credentials.length === 0) {
    if (args.format === 'json') {
      console.log(JSON.stringify({
        leaks: [],
        summary: { filesScanned: 0, credentialsChecked: 0, leaksFound: 0 },
        message: 'No credentials configured'
      }, null, 2));
    } else {
      console.log('🔐 **Credential Leak Check**');
      console.log('No credentials configured in leak-check.json');
    }
    return;
  }

  // Build matchers once
  const matchers = credentials.map(c => buildMatcher(c.search));

  if (!fs.existsSync(DEFAULT_SESSION_PATH)) {
    console.error(`Error: Session directory not found: ${DEFAULT_SESSION_PATH}`);
    process.exit(1);
  }

  const files = findSessionFiles(DEFAULT_SESSION_PATH);

  // Deduplicate: keep earliest occurrence per credential+session
  const leakMap = new Map(); // key: "credName|session"

  for (const file of files) {
    try {
      const fileLeaks = scanFile(file, credentials, matchers);
      for (const leak of fileLeaks) {
        const key = `${leak.credential}|${leak.session}`;
        if (!leakMap.has(key)) {
          leakMap.set(key, leak);
        } else {
          // Keep earliest timestamp
          const existing = leakMap.get(key);
          if (leak.timestamp && (!existing.timestamp || leak.timestamp < existing.timestamp)) {
            leakMap.set(key, leak);
          }
        }
      }
    } catch {
      // Skip files that can't be read
    }
  }

  const leaks = Array.from(leakMap.values());

  if (args.format === 'json') {
    console.log(formatJsonOutput(leaks, files.length, credentials.length));
  } else {
    console.log(formatDiscordOutput(leaks, files.length, credentials.length));
  }
}

main();
