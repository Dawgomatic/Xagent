// Patch an OpenClaw config JSON to include outsideclaw-installed skill path.
// Safe behavior:
// - Creates a .bak backup next to the config.
// - Only adds skill if not present.
// Usage:
//   node patch_openclaw_config.js --config /path/to/config.json

const fs = require("node:fs");
const os = require("node:os");
const path = require("node:path");

function parseArgs(argv) {
  const flags = {};
  for (let i = 2; i < argv.length; i++) {
    const a = argv[i];
    if (a.startsWith("--")) {
      const k = a.slice(2);
      const n = argv[i + 1];
      if (n != null && !n.startsWith("--")) {
        flags[k] = n;
        i++;
      } else {
        flags[k] = true;
      }
    }
  }
  return flags;
}

const flags = parseArgs(process.argv);
const configPath = flags.config;
if (!configPath) {
  console.error("Usage: node patch_openclaw_config.js --config /path/to/config.json");
  process.exit(1);
}

const OUTSIDECLAW_HOME = process.env.OUTSIDECLAW_HOME || path.join(os.homedir(), ".outsideclaw");
const OUTSIDECLAW_APP_DIR = process.env.OUTSIDECLAW_APP_DIR || path.join(OUTSIDECLAW_HOME, "app", "outsideclaw");
const skillPath = path.join(OUTSIDECLAW_APP_DIR, "skills", "trail-nav-telegram");

if (!fs.existsSync(skillPath)) {
  console.error("E:SKILL_PATH_NOT_FOUND", skillPath);
  process.exit(2);
}

const raw = fs.readFileSync(configPath, "utf8");
let cfg;
try {
  cfg = JSON.parse(raw);
} catch (e) {
  console.error("E:BAD_JSON");
  process.exit(3);
}

cfg.skills = Array.isArray(cfg.skills) ? cfg.skills : [];
const exists = cfg.skills.some((s) => s && (s.name === "trail-nav-telegram" || s.path === skillPath));
if (!exists) {
  cfg.skills.push({ name: "trail-nav-telegram", path: skillPath });
}

const bak = configPath + ".bak";
fs.copyFileSync(configPath, bak);
fs.writeFileSync(configPath, JSON.stringify(cfg, null, 2), "utf8");

process.stdout.write(
  JSON.stringify(
    {
      ok: true,
      configPath,
      backupPath: bak,
      added: !exists,
      skill: { name: "trail-nav-telegram", path: skillPath },
    },
    null,
    2
  ) + "\n"
);
