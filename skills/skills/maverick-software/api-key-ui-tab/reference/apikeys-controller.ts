import type { GatewayBrowserClient } from "../gateway";

export type DiscoveredKey = {
  id: string;
  path: string[];
  pathStr: string;
  name: string;
  description: string;
  configured: boolean;
  maskedValue?: string;
  docsUrl?: string;
  provider?: string;
};

export type ApiKeysState = {
  client: GatewayBrowserClient | null;
  connected: boolean;
  apikeysLoading?: boolean;
  apikeysError?: string | null;
  apikeysKeys?: DiscoveredKey[];
  apikeysEdits?: Record<string, string>;
  apikeysBusyKey?: string | null;
  apikeysMessage?: { kind: "success" | "error"; text: string } | null;
  apikeysConfigHash?: string | null;
  apikeysRefreshSuccess?: boolean;
};

/**
 * Known provider metadata for better UX on recognized keys
 */
const KNOWN_PROVIDERS: Record<string, { name: string; description: string; docsUrl?: string }> = {
  // env keys
  "ANTHROPIC_API_KEY": { name: "Anthropic", description: "Claude models", docsUrl: "https://console.anthropic.com/settings/keys" },
  "OPENAI_API_KEY": { name: "OpenAI", description: "GPT, Whisper, DALL-E, embeddings", docsUrl: "https://platform.openai.com/api-keys" },
  "BRAVE_API_KEY": { name: "Brave Search", description: "Web search API", docsUrl: "https://brave.com/search/api/" },
  "DEEPGRAM_API_KEY": { name: "Deepgram", description: "Speech-to-text", docsUrl: "https://console.deepgram.com/" },
  "GOOGLE_API_KEY": { name: "Google", description: "Gemini models, Google APIs", docsUrl: "https://aistudio.google.com/apikey" },
  "ELEVENLABS_API_KEY": { name: "ElevenLabs", description: "Text-to-speech", docsUrl: "https://elevenlabs.io/app/settings/api-keys" },
  "OPENROUTER_API_KEY": { name: "OpenRouter", description: "Multi-provider LLM access", docsUrl: "https://openrouter.ai/settings/keys" },
  "GROQ_API_KEY": { name: "Groq", description: "Fast Llama/Mixtral inference", docsUrl: "https://console.groq.com/keys" },
  "FIREWORKS_API_KEY": { name: "Fireworks AI", description: "Open-source model inference", docsUrl: "https://fireworks.ai/account/api-keys" },
  "MISTRAL_API_KEY": { name: "Mistral AI", description: "Mistral models", docsUrl: "https://console.mistral.ai/api-keys/" },
  "XAI_API_KEY": { name: "xAI (Grok)", description: "Grok models", docsUrl: "https://console.x.ai/" },
  "PERPLEXITY_API_KEY": { name: "Perplexity", description: "AI-powered search", docsUrl: "https://www.perplexity.ai/settings/api" },
  "GITHUB_TOKEN": { name: "GitHub", description: "GitHub API access", docsUrl: "https://github.com/settings/tokens" },
  // Common nested paths
  "elevenlabs": { name: "ElevenLabs", description: "Text-to-speech", docsUrl: "https://elevenlabs.io/app/settings/api-keys" },
  "openai": { name: "OpenAI", description: "GPT, Whisper, DALL-E", docsUrl: "https://platform.openai.com/api-keys" },
  "anthropic": { name: "Anthropic", description: "Claude models", docsUrl: "https://console.anthropic.com/settings/keys" },
  "deepgram": { name: "Deepgram", description: "Speech-to-text", docsUrl: "https://console.deepgram.com/" },
  "sag": { name: "ElevenLabs (sag)", description: "TTS skill", docsUrl: "https://elevenlabs.io/app/settings/api-keys" },
  "openai-whisper-api": { name: "OpenAI Whisper", description: "Transcription skill", docsUrl: "https://platform.openai.com/api-keys" },
  "openai-image-gen": { name: "OpenAI Images", description: "DALL-E skill", docsUrl: "https://platform.openai.com/api-keys" },
  "deepgram-streaming": { name: "Deepgram Streaming", description: "Live transcription", docsUrl: "https://console.deepgram.com/" },
};

/**
 * Patterns that indicate an API key field
 */
const KEY_PATTERNS = [
  /apiKey$/i,
  /api_key$/i,
  /^token$/i,
  /secret$/i,
  /_KEY$/,
  /_TOKEN$/,
  /_SECRET$/,
];

/**
 * Paths to skip when scanning (not API keys)
 */
const SKIP_PATHS = [
  "meta",
  "wizard",
  "update",
  "browser",
  "auth.profiles",
  "gateway.auth.token", // Gateway auth token is not an API key
  "channels.telegram.botToken", // Bot tokens handled separately
  "channels.discord.token",
];

/**
 * Check if a field name looks like an API key
 */
function isKeyField(fieldName: string): boolean {
  return KEY_PATTERNS.some(pattern => pattern.test(fieldName));
}

/**
 * Check if a path should be skipped
 */
function shouldSkipPath(path: string[]): boolean {
  const pathStr = path.join(".");
  return SKIP_PATHS.some(skip => pathStr.startsWith(skip));
}

/**
 * Mask an API key for display
 */
function maskKey(key: string): string {
  if (!key || key.length <= 8) return "••••••••";
  return `${key.slice(0, 4)}...${key.slice(-4)}`;
}

/**
 * Generate a human-readable name from a path
 */
function generateName(path: string[]): string {
  // Check for known provider in path
  for (const segment of path) {
    const known = KNOWN_PROVIDERS[segment];
    if (known) return known.name;
  }
  
  // For env keys like ANTHROPIC_API_KEY
  if (path[0] === "env" && path[1]) {
    const known = KNOWN_PROVIDERS[path[1]];
    if (known) return known.name;
    // Convert SOME_API_KEY to "Some"
    return path[1].replace(/_API_KEY$|_KEY$|_TOKEN$|_SECRET$/i, "").split("_").map(
      w => w.charAt(0).toUpperCase() + w.slice(1).toLowerCase()
    ).join(" ");
  }
  
  // For skill keys like skills.entries.sag.apiKey
  if (path[0] === "skills" && path[1] === "entries" && path[2]) {
    const known = KNOWN_PROVIDERS[path[2]];
    if (known) return known.name;
    return `Skill: ${path[2]}`;
  }
  
  // For nested paths like messages.tts.elevenlabs.apiKey
  const secondToLast = path[path.length - 2];
  if (secondToLast) {
    const known = KNOWN_PROVIDERS[secondToLast];
    if (known) return known.name;
    return secondToLast.charAt(0).toUpperCase() + secondToLast.slice(1);
  }
  
  return path.join(".");
}

/**
 * Generate description from path
 */
function generateDescription(path: string[]): string {
  // Check known providers
  for (const segment of path) {
    const known = KNOWN_PROVIDERS[segment];
    if (known) return known.description;
  }
  if (path[0] === "env" && path[1]) {
    const known = KNOWN_PROVIDERS[path[1]];
    if (known) return known.description;
  }
  
  // Generate from path
  if (path[0] === "skills" && path[1] === "entries") {
    return `API key for ${path[2]} skill`;
  }
  if (path[0] === "env") {
    return `Environment variable`;
  }
  return `Config: ${path.slice(0, -1).join(".")}`;
}

/**
 * Get docs URL if known
 */
function getDocsUrl(path: string[]): string | undefined {
  for (const segment of path) {
    const known = KNOWN_PROVIDERS[segment];
    if (known?.docsUrl) return known.docsUrl;
  }
  if (path[0] === "env" && path[1]) {
    const known = KNOWN_PROVIDERS[path[1]];
    if (known?.docsUrl) return known.docsUrl;
  }
  return undefined;
}

/**
 * Recursively scan config for API keys
 */
function scanForKeys(
  obj: Record<string, unknown>,
  path: string[] = [],
  results: DiscoveredKey[] = []
): DiscoveredKey[] {
  if (!obj || typeof obj !== "object") return results;
  
  for (const [key, value] of Object.entries(obj)) {
    const currentPath = [...path, key];
    
    // Skip certain paths
    if (shouldSkipPath(currentPath)) continue;
    
    // Check if this field is an API key
    if (isKeyField(key)) {
      const pathStr = currentPath.join(".");
      const hasValue = typeof value === "string" && value.length > 0;
      
      results.push({
        id: pathStr,
        path: currentPath,
        pathStr,
        name: generateName(currentPath),
        description: generateDescription(currentPath),
        configured: hasValue,
        maskedValue: hasValue ? maskKey(value as string) : undefined,
        docsUrl: getDocsUrl(currentPath),
      });
    }
    
    // Recurse into objects (but not arrays)
    if (value && typeof value === "object" && !Array.isArray(value)) {
      scanForKeys(value as Record<string, unknown>, currentPath, results);
    }
  }
  
  return results;
}

/**
 * Add common env keys that might not exist yet
 */
function addMissingCommonKeys(keys: DiscoveredKey[], config: Record<string, unknown>): DiscoveredKey[] {
  const existingEnvKeys = new Set(
    keys.filter(k => k.path[0] === "env").map(k => k.path[1])
  );
  
  const commonEnvKeys = [
    "ANTHROPIC_API_KEY",
    "OPENAI_API_KEY", 
    "BRAVE_API_KEY",
    "DEEPGRAM_API_KEY",
    "GOOGLE_API_KEY",
    "ELEVENLABS_API_KEY",
    "GITHUB_TOKEN",
  ];
  
  for (const envKey of commonEnvKeys) {
    if (!existingEnvKeys.has(envKey)) {
      const path = ["env", envKey];
      const known = KNOWN_PROVIDERS[envKey];
      keys.push({
        id: path.join("."),
        path,
        pathStr: path.join("."),
        name: known?.name || envKey,
        description: known?.description || "Not configured",
        configured: false,
        docsUrl: known?.docsUrl,
      });
    }
  }
  
  return keys;
}

/**
 * Sort keys: configured first, then by name
 */
function sortKeys(keys: DiscoveredKey[]): DiscoveredKey[] {
  return keys.sort((a, b) => {
    // Env keys first
    const aIsEnv = a.path[0] === "env" ? 0 : 1;
    const bIsEnv = b.path[0] === "env" ? 0 : 1;
    if (aIsEnv !== bIsEnv) return aIsEnv - bIsEnv;
    
    // Then by name
    return a.name.localeCompare(b.name);
  });
}

/**
 * Load API keys by scanning the config
 */
export async function loadApiKeys(state: ApiKeysState): Promise<void> {
  if (!state.client || !state.connected) return;

  state.apikeysError = null;

  try {
    const configRes = (await state.client.request("config.get", {})) as {
      config?: Record<string, unknown>;
      hash?: string;
    } | null;

    const config = configRes?.config ?? {};
    state.apikeysConfigHash = configRes?.hash || null;

    // Scan config for all API keys
    let keys = scanForKeys(config);
    
    // Add common keys that might not exist
    keys = addMissingCommonKeys(keys, config);
    
    // Sort keys
    keys = sortKeys(keys);

    state.apikeysKeys = keys;
    state.apikeysError = null;
    state.apikeysLoading = false;
    state.apikeysRefreshSuccess = true;
    
    // Clear success indicator after 2 seconds
    setTimeout(() => {
      state.apikeysRefreshSuccess = false;
    }, 2000);
  } catch (err) {
    state.apikeysError = err instanceof Error ? err.message : String(err);
    state.apikeysLoading = false;
  }
}

/**
 * Update edit buffer
 */
export function updateApiKeyEdit(
  state: ApiKeysState,
  keyId: string,
  value: string
): void {
  state.apikeysEdits = { ...(state.apikeysEdits ?? {}), [keyId]: value };
}

/**
 * Build nested patch object from path
 */
function buildPatch(path: string[], value: unknown): Record<string, unknown> {
  if (path.length === 0) return {};
  if (path.length === 1) return { [path[0]]: value };

  const result: Record<string, unknown> = {};
  let current = result;

  for (let i = 0; i < path.length - 1; i++) {
    current[path[i]] = {};
    current = current[path[i]] as Record<string, unknown>;
  }

  current[path[path.length - 1]] = value;
  return result;
}

/**
 * Save an API key
 */
export async function saveApiKey(state: ApiKeysState, keyId: string): Promise<void> {
  if (!state.client || !state.connected) return;

  const key = state.apikeysKeys?.find(k => k.id === keyId);
  if (!key) return;

  const value = state.apikeysEdits?.[keyId];
  if (!value || value.trim().length === 0) return;

  state.apikeysBusyKey = keyId;
  state.apikeysError = null;
  state.apikeysMessage = null;

  try {
    const patch = buildPatch(key.path, value.trim());

    await state.client.request("config.patch", {
      raw: JSON.stringify(patch),
      baseHash: state.apikeysConfigHash || undefined,
    });

    // Clear edit buffer
    const edits = { ...(state.apikeysEdits ?? {}) };
    delete edits[keyId];
    state.apikeysEdits = edits;

    state.apikeysMessage = {
      kind: "success",
      text: `${key.name} saved successfully.`,
    };

    // Reload after gateway restart
    setTimeout(() => loadApiKeys(state), 2500);
  } catch (err) {
    const message = err instanceof Error ? err.message : String(err);
    state.apikeysError = message;
    state.apikeysMessage = {
      kind: "error",
      text: `Failed to save ${key.name}: ${message}`,
    };
  } finally {
    state.apikeysBusyKey = null;
  }
}

/**
 * Clear an API key
 */
export async function clearApiKey(state: ApiKeysState, keyId: string): Promise<void> {
  if (!state.client || !state.connected) return;

  const key = state.apikeysKeys?.find(k => k.id === keyId);
  if (!key) return;

  state.apikeysBusyKey = keyId;
  state.apikeysError = null;
  state.apikeysMessage = null;

  try {
    const patch = buildPatch(key.path, null);

    await state.client.request("config.patch", {
      raw: JSON.stringify(patch),
      baseHash: state.apikeysConfigHash || undefined,
    });

    state.apikeysMessage = {
      kind: "success",
      text: `${key.name} removed.`,
    };

    // Reload after gateway restart
    setTimeout(() => loadApiKeys(state), 2500);
  } catch (err) {
    const message = err instanceof Error ? err.message : String(err);
    state.apikeysError = message;
    state.apikeysMessage = {
      kind: "error",
      text: `Failed to clear ${key.name}: ${message}`,
    };
  } finally {
    state.apikeysBusyKey = null;
  }
}
