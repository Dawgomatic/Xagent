import { html, nothing } from "lit";
import type { DiscoveredKey } from "../controllers/apikeys.js";

export type ApiKeysProps = {
  error: string | null;
  keys: DiscoveredKey[];
  edits: Record<string, string>;
  busyKey: string | null;
  message: { kind: "success" | "error"; text: string } | null;
  loading?: boolean;
  onEdit: (keyId: string, value: string) => void;
  onSave: (keyId: string) => void;
  onClear: (keyId: string) => void;
  onRefresh?: () => void;
};

export function renderApiKeys(props: ApiKeysProps) {
  // Group keys by category
  const envKeys = props.keys.filter(k => k.path[0] === "env");
  const skillKeys = props.keys.filter(k => k.path[0] === "skills");
  const otherKeys = props.keys.filter(k => k.path[0] !== "env" && k.path[0] !== "skills");

  const configuredCount = props.keys.filter(k => k.configured).length;
  const totalCount = props.keys.length;

  return html`
    <section class="card">
      <div style="display: flex; justify-content: space-between; align-items: flex-start; flex-wrap: wrap; gap: 12px;">
        <div style="flex: 1; min-width: 200px;">
          <div class="card-title">API Keys</div>
          <div class="card-sub">
            Dynamically discovered from your config. ${configuredCount}/${totalCount} configured.
          </div>
        </div>
        <button
          class="btn"
          @click=${() => window.location.reload()}
          title="Refresh page"
        >
          ↻ Refresh
        </button>
      </div>

      ${props.error
        ? html`<div class="callout danger" style="margin-top: 12px;">${props.error}</div>`
        : nothing}

      ${props.message
        ? html`<div class="callout ${props.message.kind === "error" ? "danger" : "success"}" style="margin-top: 12px;">
            ${props.message.text}
          </div>`
        : nothing}

      ${props.loading
        ? html`<div style="padding: 24px; text-align: center; opacity: 0.6;">Scanning config for API keys...</div>`
        : html`
            ${envKeys.length > 0 ? html`
              <div style="margin-top: 20px;">
                <div class="section-label">Environment Keys</div>
                <div class="list">
                  ${envKeys.map((key) => renderKeyCard(key, props))}
                </div>
              </div>
            ` : nothing}

            ${skillKeys.length > 0 ? html`
              <div style="margin-top: 20px;">
                <div class="section-label">Skill Keys</div>
                <div class="list">
                  ${skillKeys.map((key) => renderKeyCard(key, props))}
                </div>
              </div>
            ` : nothing}

            ${otherKeys.length > 0 ? html`
              <div style="margin-top: 20px;">
                <div class="section-label">Other Keys</div>
                <div class="list">
                  ${otherKeys.map((key) => renderKeyCard(key, props))}
                </div>
              </div>
            ` : nothing}
          `}
    </section>

    <section class="card" style="margin-top: 24px;">
      <div class="card-title">About API Keys</div>
      <div class="card-sub" style="margin-top: 8px;">
        <p>This page automatically scans your config file for API keys, tokens, and secrets.</p>
        <p style="margin-top: 8px;">Keys are stored in <code>~/.clawdbot/clawdbot.json</code> and are never transmitted outside your machine except to their respective providers.</p>
        <p style="margin-top: 8px;">The AI agent <strong>never sees your keys</strong> — this UI writes directly to the config file.</p>
      </div>
    </section>

    <style>
      .section-label {
        font-size: 12px;
        font-weight: 600;
        text-transform: uppercase;
        letter-spacing: 0.05em;
        opacity: 0.6;
        margin-bottom: 8px;
        padding-left: 4px;
      }
    </style>
  `;
}

function renderKeyCard(key: DiscoveredKey, props: ApiKeysProps) {
  const busy = props.busyKey === key.id;
  const editValue = props.edits[key.id] ?? "";
  const hasEdit = editValue.length > 0;

  return html`
    <div class="list-item" style="flex-direction: column; align-items: stretch; gap: 12px;">
      <div style="display: flex; justify-content: space-between; align-items: flex-start; flex-wrap: wrap; gap: 8px; width: 100%;">
        <div style="flex: 1; min-width: 200px;">
          <div class="list-title">${key.name}</div>
          <div class="list-sub" style="margin-top: 4px;">${key.description}</div>
          <div class="chip-row" style="margin-top: 8px; flex-wrap: wrap; gap: 6px;">
            <span class="chip ${key.configured ? "chip-ok" : "chip-warn"}">
              ${key.configured ? "Configured" : "Not configured"}
            </span>
          </div>
          <div class="muted" style="margin-top: 8px; font-size: 11px; font-family: monospace; opacity: 0.5;">
            ${key.pathStr}
          </div>
        </div>
        <div style="display: flex; gap: 8px; align-items: center; flex-shrink: 0;">
          ${key.docsUrl
            ? html`<a
                class="btn"
                href=${key.docsUrl}
                target="_blank"
                rel="noopener noreferrer"
                title="Get API key"
                style="white-space: nowrap;"
              >
                Get key ↗
              </a>`
            : nothing}
        </div>
      </div>

      <div style="width: 100%;">
        <div class="field">
          <span>API Key</span>
          <div style="display: flex; gap: 8px; flex-wrap: wrap;">
            <input
              type="password"
              style="flex: 1; min-width: 200px;"
              .value=${editValue}
              placeholder=${key.configured
                ? key.maskedValue ?? "••••••••"
                : "Enter API key..."}
              @input=${(e: Event) =>
                props.onEdit(key.id, (e.target as HTMLInputElement).value)}
            />
            <div style="display: flex; gap: 8px; flex-shrink: 0;">
              <button
                class="btn primary"
                ?disabled=${busy || !hasEdit}
                @click=${() => props.onSave(key.id)}
              >
                ${busy ? "Saving…" : "Save"}
              </button>
              ${key.configured
                ? html`<button
                    class="btn danger"
                    ?disabled=${busy}
                    @click=${() => props.onClear(key.id)}
                    title="Remove this API key"
                  >
                    Clear
                  </button>`
                : nothing}
            </div>
          </div>
        </div>
      </div>
    </div>
  `;
}
