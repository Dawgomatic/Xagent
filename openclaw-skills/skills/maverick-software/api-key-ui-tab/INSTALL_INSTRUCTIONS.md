# API Keys UI - Installation Instructions

This skill adds an **API Keys** tab to the OpenClaw Control dashboard under **Settings**.

## Prerequisites

✅ **OpenClaw v2026.1.0+** with source access at `~/clawdbot`

---

## Installation Steps

### Step 1: Copy View File

**Copy to:** `ui/src/ui/views/apikeys.ts`

```bash
cp ~/clawd/skills/apikeys-ui/reference/apikeys-views.ts ~/clawdbot/ui/src/ui/views/apikeys.ts
```

### Step 2: Copy Controller File

**Copy to:** `ui/src/ui/controllers/apikeys.ts`

```bash
cp ~/clawd/skills/apikeys-ui/reference/apikeys-controller.ts ~/clawdbot/ui/src/ui/controllers/apikeys.ts
```

### Step 3: Add Navigation Tab

**File:** `ui/src/ui/navigation.ts`

1. Add to Settings group in `TAB_GROUPS` (put it first in Settings):
```typescript
  { label: "Settings", tabs: ["apikeys", "config", "debug", "logs"] },
```

2. Add to `Tab` type union:
```typescript
  | "apikeys"
```

3. Add to `TAB_PATHS`:
```typescript
  apikeys: "/apikeys",
```

4. Add to `iconForTab` switch:
```typescript
    case "apikeys":
      return "key";
```

5. Add to `titleForTab` switch:
```typescript
    case "apikeys":
      return "API Keys";
```

6. Add to `subtitleForTab` switch:
```typescript
    case "apikeys":
      return "Manage API keys for AI providers, search, TTS, and embeddings.";
```

### Step 4: Add App State

**File:** `ui/src/ui/app.ts`

Add these state variables in the ClawdApp class:

```typescript
  // API Keys state
  @state() apikeysLoading = false;
  @state() apikeysError: string | null = null;
  @state() apikeysKeys: import("./views/apikeys.js").ApiKeyStatus[] = [];
  @state() apikeysEdits: Record<string, string> = {};
  @state() apikeysBusyKey: string | null = null;
  @state() apikeysMessage: { kind: "success" | "error"; text: string } | null = null;
  @state() apikeysConfigHash: string | null = null;
```

### Step 5: Add View Rendering

**File:** `ui/src/ui/app-render.ts`

1. Add import near the top:
```typescript
import { renderApiKeys } from "./views/apikeys.js";
import {
  loadApiKeys,
  updateApiKeyEdit,
  saveApiKey,
  clearApiKey,
} from "./controllers/apikeys.js";
```

2. Add rendering in the main content area (look for similar tab conditionals):
```typescript
        ${state.tab === "apikeys"
          ? renderApiKeys({
              loading: state.apikeysLoading ?? false,
              error: state.apikeysError ?? null,
              keys: state.apikeysKeys ?? [],
              edits: state.apikeysEdits ?? {},
              busyKey: state.apikeysBusyKey ?? null,
              message: state.apikeysMessage ?? null,
              onRefresh: () => loadApiKeys(state),
              onEdit: (keyId, value) => updateApiKeyEdit(state, keyId, value),
              onSave: (keyId) => saveApiKey(state, keyId),
              onClear: (keyId) => clearApiKey(state, keyId),
            })
          : nothing}
```

### Step 6: Add Tab Loading

**File:** `ui/src/ui/app-settings.ts`

1. Add import:
```typescript
import { loadApiKeys } from "./controllers/apikeys.js";
```

2. Add the loading logic in `refreshActiveTab` function:
```typescript
  if (host.tab === "apikeys") {
    await loadApiKeys(host);
  }
```

### Step 7: Build and Test

```bash
cd ~/clawdbot

# Build UI
pnpm ui:build

# Restart gateway
clawdbot gateway restart
```

### Step 8: Verify Installation

1. Open the OpenClaw dashboard
2. Look in sidebar under **Settings**
3. Click **API Keys** tab
4. Should see list of 13 providers with status indicators

---

## Quick Copy Commands

```bash
# Step 1-2: Copy files
cp ~/clawd/skills/apikeys-ui/reference/apikeys-views.ts ~/clawdbot/ui/src/ui/views/apikeys.ts
cp ~/clawd/skills/apikeys-ui/reference/apikeys-controller.ts ~/clawdbot/ui/src/ui/controllers/apikeys.ts

# Step 7: Build
cd ~/clawdbot && pnpm ui:build && clawdbot gateway restart
```

Steps 3-6 require code edits as documented above.

---

## Troubleshooting

### Tab doesn't appear
- Check TAB_GROUPS includes "apikeys" in Settings group
- Hard refresh browser (Ctrl+Shift+R)
- Check browser console for errors

### Keys not loading
- Check that `config.get` RPC works: Debug → RPC → `config.get`
- Check browser console for errors

### Key not saving
- Check gateway logs: `clawdbot logs --follow`
- Try manual: `clawdbot config patch '{"env":{"TEST":"value"}}'`

### Build errors
- Check TypeScript errors in terminal
- Make sure all imports use `.js` extension
- Run `pnpm typecheck` to see all errors
