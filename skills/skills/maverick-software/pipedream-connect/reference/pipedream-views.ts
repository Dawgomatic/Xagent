import { html, nothing } from "lit";

export type PipedreamApp = {
  slug: string;
  name: string;
  icon: string;
  connected: boolean;
  toolCount?: number;
  accountName?: string;
  serverName?: string;
};

export type PipedreamCredentials = {
  clientId: string;
  clientSecret: string;
  projectId: string;
  environment: "development" | "production";
};

export type PipedreamState = {
  loading: boolean;
  configured: boolean;
  credentials: PipedreamCredentials;
  showCredentialsForm: boolean;
  connectedApps: PipedreamApp[];
  availableApps: PipedreamApp[];
  error: string | null;
  success: string | null;
  testingApp: string | null;
  connectingApp: string | null;
  refreshingApp: string | null;
  externalUserId: string;
  // App browser modal
  showAppBrowser: boolean;
  appBrowserSearch: string;
  allApps: PipedreamApp[];
  loadingApps: boolean;
  // Manual slug entry
  manualSlug: string;
};

export type PipedreamProps = PipedreamState & {
  onConfigure: () => void;
  onSaveCredentials: () => void;
  onCancelCredentials: () => void;
  onCredentialChange: (field: keyof PipedreamCredentials, value: string) => void;
  onConnectApp: (appSlug: string) => void;
  onDisconnectApp: (appSlug: string) => void;
  onTestApp: (appSlug: string) => void;
  onRefreshToken: (appSlug: string) => void;
  onExternalUserIdChange: (value: string) => void;
  // App browser
  onOpenAppBrowser: () => void;
  onCloseAppBrowser: () => void;
  onAppBrowserSearchChange: (value: string) => void;
  onManualSlugChange: (value: string) => void;
  onConnectManualSlug: () => void;
};

// Comprehensive list of popular Pipedream apps
const ALL_APPS: PipedreamApp[] = [
  // Google
  { slug: "gmail", name: "Gmail", icon: "📧", connected: false },
  { slug: "google-calendar", name: "Google Calendar", icon: "📅", connected: false },
  { slug: "google-sheets", name: "Google Sheets", icon: "📊", connected: false },
  { slug: "google-drive", name: "Google Drive", icon: "📁", connected: false },
  { slug: "google-docs", name: "Google Docs", icon: "📄", connected: false },
  { slug: "google-tasks", name: "Google Tasks", icon: "✔️", connected: false },
  { slug: "youtube", name: "YouTube", icon: "▶️", connected: false },
  { slug: "youtube-data-api", name: "YouTube Data API", icon: "📺", connected: false },
  // Communication
  { slug: "slack", name: "Slack", icon: "💬", connected: false },
  { slug: "discord", name: "Discord", icon: "🎮", connected: false },
  { slug: "telegram-bot-api", name: "Telegram", icon: "📱", connected: false },
  { slug: "twilio", name: "Twilio", icon: "📞", connected: false },
  { slug: "sendgrid", name: "SendGrid", icon: "✉️", connected: false },
  { slug: "mailgun", name: "Mailgun", icon: "📬", connected: false },
  { slug: "mailchimp", name: "Mailchimp", icon: "🐵", connected: false },
  { slug: "intercom", name: "Intercom", icon: "💭", connected: false },
  { slug: "zoom", name: "Zoom", icon: "📹", connected: false },
  { slug: "microsoft_teams", name: "Microsoft Teams", icon: "👥", connected: false },
  // Project Management
  { slug: "notion", name: "Notion", icon: "📝", connected: false },
  { slug: "linear", name: "Linear", icon: "📋", connected: false },
  { slug: "asana", name: "Asana", icon: "✅", connected: false },
  { slug: "trello", name: "Trello", icon: "📌", connected: false },
  { slug: "monday", name: "Monday.com", icon: "📆", connected: false },
  { slug: "clickup", name: "ClickUp", icon: "🎯", connected: false },
  { slug: "basecamp", name: "Basecamp", icon: "🏕️", connected: false },
  { slug: "jira", name: "Jira", icon: "🔷", connected: false },
  { slug: "todoist", name: "Todoist", icon: "☑️", connected: false },
  // Development
  { slug: "github", name: "GitHub", icon: "🐙", connected: false },
  { slug: "gitlab", name: "GitLab", icon: "🦊", connected: false },
  { slug: "bitbucket", name: "Bitbucket", icon: "🪣", connected: false },
  { slug: "vercel", name: "Vercel", icon: "▲", connected: false },
  { slug: "netlify", name: "Netlify", icon: "🌐", connected: false },
  { slug: "railway", name: "Railway", icon: "🚂", connected: false },
  { slug: "render", name: "Render", icon: "🖼️", connected: false },
  { slug: "fly-io", name: "Fly.io", icon: "✈️", connected: false },
  { slug: "sentry", name: "Sentry", icon: "🐛", connected: false },
  { slug: "datadog", name: "Datadog", icon: "🐕", connected: false },
  // CRM & Sales
  { slug: "hubspot", name: "HubSpot", icon: "🧲", connected: false },
  { slug: "salesforce_rest_api", name: "Salesforce", icon: "☁️", connected: false },
  { slug: "pipedrive", name: "Pipedrive", icon: "🔀", connected: false },
  { slug: "zoho-crm", name: "Zoho CRM", icon: "📊", connected: false },
  { slug: "freshsales", name: "Freshsales", icon: "🍃", connected: false },
  { slug: "close", name: "Close", icon: "🎯", connected: false },
  // Storage & Files
  { slug: "dropbox", name: "Dropbox", icon: "📦", connected: false },
  { slug: "box", name: "Box", icon: "📥", connected: false },
  { slug: "onedrive", name: "OneDrive", icon: "☁️", connected: false },
  { slug: "aws", name: "AWS", icon: "🟠", connected: false },
  { slug: "supabase", name: "Supabase", icon: "⚡", connected: false },
  { slug: "firebase", name: "Firebase", icon: "🔥", connected: false },
  { slug: "cloudinary", name: "Cloudinary", icon: "🖼️", connected: false },
  // Databases
  { slug: "airtable", name: "Airtable", icon: "📑", connected: false },
  { slug: "mongodb", name: "MongoDB", icon: "🍃", connected: false },
  { slug: "postgresql", name: "PostgreSQL", icon: "🐘", connected: false },
  { slug: "mysql", name: "MySQL", icon: "🐬", connected: false },
  { slug: "redis", name: "Redis", icon: "🔴", connected: false },
  { slug: "snowflake", name: "Snowflake", icon: "❄️", connected: false },
  // AI & ML
  { slug: "openai", name: "OpenAI", icon: "🤖", connected: false },
  { slug: "anthropic", name: "Anthropic", icon: "🧠", connected: false },
  { slug: "replicate", name: "Replicate", icon: "🔄", connected: false },
  { slug: "huggingface", name: "Hugging Face", icon: "🤗", connected: false },
  { slug: "stability-ai", name: "Stability AI", icon: "🎨", connected: false },
  { slug: "eleven-labs", name: "ElevenLabs", icon: "🔊", connected: false },
  { slug: "deepgram", name: "Deepgram", icon: "🎤", connected: false },
  // Social Media
  { slug: "twitter", name: "Twitter/X", icon: "🐦", connected: false },
  { slug: "linkedin", name: "LinkedIn", icon: "💼", connected: false },
  { slug: "facebook_pages", name: "Facebook Pages", icon: "📘", connected: false },
  { slug: "instagram_business", name: "Instagram", icon: "📸", connected: false },
  { slug: "tiktok", name: "TikTok", icon: "🎵", connected: false },
  { slug: "reddit", name: "Reddit", icon: "🤖", connected: false },
  { slug: "pinterest", name: "Pinterest", icon: "📍", connected: false },
  // Payments
  { slug: "stripe", name: "Stripe", icon: "💳", connected: false },
  { slug: "paypal", name: "PayPal", icon: "🅿️", connected: false },
  { slug: "square", name: "Square", icon: "⬛", connected: false },
  { slug: "shopify", name: "Shopify", icon: "🛍️", connected: false },
  { slug: "gumroad", name: "Gumroad", icon: "🎁", connected: false },
  { slug: "lemonsqueezy", name: "Lemon Squeezy", icon: "🍋", connected: false },
  // Forms & Surveys
  { slug: "typeform", name: "Typeform", icon: "📋", connected: false },
  { slug: "google-forms", name: "Google Forms", icon: "📝", connected: false },
  { slug: "surveymonkey", name: "SurveyMonkey", icon: "🐒", connected: false },
  { slug: "tally", name: "Tally", icon: "📊", connected: false },
  { slug: "jotform", name: "JotForm", icon: "📄", connected: false },
  // Analytics
  { slug: "google_analytics", name: "Google Analytics", icon: "📈", connected: false },
  { slug: "mixpanel", name: "Mixpanel", icon: "📊", connected: false },
  { slug: "amplitude", name: "Amplitude", icon: "📉", connected: false },
  { slug: "segment", name: "Segment", icon: "📀", connected: false },
  { slug: "posthog", name: "PostHog", icon: "🦔", connected: false },
  // Productivity
  { slug: "calendar", name: "Calendly", icon: "📅", connected: false },
  { slug: "loom", name: "Loom", icon: "🎬", connected: false },
  { slug: "coda", name: "Coda", icon: "📑", connected: false },
  { slug: "evernote", name: "Evernote", icon: "🐘", connected: false },
  { slug: "obsidian", name: "Obsidian", icon: "💎", connected: false },
  { slug: "roam", name: "Roam Research", icon: "🧠", connected: false },
  // Support
  { slug: "zendesk", name: "Zendesk", icon: "🎫", connected: false },
  { slug: "freshdesk", name: "Freshdesk", icon: "🍃", connected: false },
  { slug: "help-scout", name: "Help Scout", icon: "🆘", connected: false },
  { slug: "crisp", name: "Crisp", icon: "💬", connected: false },
  // Other
  { slug: "webhook", name: "Webhook", icon: "🪝", connected: false },
  { slug: "http", name: "HTTP/REST", icon: "🌐", connected: false },
  { slug: "rss", name: "RSS", icon: "📡", connected: false },
  { slug: "weather-api", name: "Weather API", icon: "🌤️", connected: false },
  { slug: "openweathermap", name: "OpenWeatherMap", icon: "☀️", connected: false },
  { slug: "spotify", name: "Spotify", icon: "🎵", connected: false },
  { slug: "pocket", name: "Pocket", icon: "📚", connected: false },
  { slug: "raindrop", name: "Raindrop.io", icon: "🌧️", connected: false },
];

// Featured apps shown in the main UI
const FEATURED_APPS = ALL_APPS.slice(0, 16);

export function renderPipedream(props: PipedreamProps) {
  const statusLabel = props.configured ? "Active" : "Not Configured";
  const statusClass = props.configured ? "chip-ok" : "chip-warn";

  return html`
    <div class="page-header">
      <h1>Pipedream</h1>
      <p class="muted">Connect to 2,000+ APIs with managed OAuth via Pipedream.</p>
    </div>

    ${props.error
      ? html`<div class="callout danger" style="margin-bottom: 16px;">${props.error}</div>`
      : nothing}
    
    ${props.success
      ? html`<div class="callout success" style="margin-bottom: 16px;">${props.success}</div>`
      : nothing}

    <section class="card">
      <div class="row" style="justify-content: space-between; align-items: flex-start;">
        <div>
          <div class="card-title">
            🔗 Connection Status
            <span class="chip ${statusClass}" style="margin-left: 8px;">${statusLabel}</span>
          </div>
          <div class="card-sub">
            ${props.configured
              ? `Connected with ${props.connectedApps.length} app(s) configured.`
              : "Configure your Pipedream credentials to get started."}
          </div>
        </div>
      </div>
    </section>

    <section class="card" style="margin-top: 16px;">
      <div class="row" style="justify-content: space-between; align-items: center;">
        <div class="card-title">🔑 Credentials</div>
        ${!props.showCredentialsForm
          ? html`
              <button class="btn" @click=${props.onConfigure}>
                ${props.configured ? "Edit" : "Configure"}
              </button>
            `
          : nothing}
      </div>
      
      ${props.showCredentialsForm
        ? renderCredentialsForm(props)
        : html`
            <div class="card-sub" style="margin-top: 8px;">
              ${props.configured
                ? html`
                    <div class="row" style="gap: 24px; margin-top: 12px;">
                      <div>
                        <span class="muted">Project ID:</span>
                        <code style="margin-left: 8px;">${props.credentials.projectId}</code>
                      </div>
                      <div>
                        <span class="muted">Environment:</span>
                        <span style="margin-left: 8px;">${props.credentials.environment}</span>
                      </div>
                      <div>
                        <span class="muted">User ID:</span>
                        <code style="margin-left: 8px;">${props.externalUserId || "clawdbot"}</code>
                      </div>
                    </div>
                  `
                : html`<p class="muted">No credentials configured. Click "Configure" to get started.</p>`}
            </div>
          `}
    </section>

    ${props.configured ? renderConnectedApps(props) : nothing}
    ${props.configured ? renderAvailableApps(props) : nothing}

    <section class="card" style="margin-top: 16px;">
      <div class="card-title">📚 Setup Guide</div>
      <div class="card-sub" style="margin-top: 8px;">
        <ol style="margin: 12px 0; padding-left: 20px; line-height: 1.8;">
          <li>
            <strong>Create OAuth Client</strong> — Go to
            <a href="https://pipedream.com/settings/api" target="_blank">pipedream.com/settings/api</a>
            and create a new OAuth client
          </li>
          <li>
            <strong>Create Project</strong> — Go to
            <a href="https://pipedream.com/projects" target="_blank">pipedream.com/projects</a>
            and create a project to store connected accounts
          </li>
          <li><strong>Enter Credentials</strong> — Paste your Client ID, Secret, and Project ID above</li>
          <li><strong>Connect Apps</strong> — Click "Connect" on any app below to authorize access</li>
          <li><strong>Use Tools</strong> — Your agent can now use the connected app's tools</li>
        </ol>
      </div>
    </section>
  `;
}

function renderCredentialsForm(props: PipedreamProps) {
  return html`
    <div style="margin-top: 12px; padding: 16px; border: 1px solid var(--border); border-radius: 8px; background: var(--bg-secondary);">
      <div class="callout info" style="margin-bottom: 16px; font-size: 13px;">
        Get your credentials from
        <a href="https://pipedream.com/settings/api" target="_blank">pipedream.com/settings/api</a>
      </div>

      <label class="field">
        <span>Client ID</span>
        <input
          type="text"
          .value=${props.credentials.clientId}
          @input=${(e: Event) => props.onCredentialChange("clientId", (e.target as HTMLInputElement).value)}
          placeholder="Your OAuth Client ID"
        />
      </label>

      <label class="field" style="margin-top: 12px;">
        <span>Client Secret</span>
        <input
          type="password"
          .value=${props.credentials.clientSecret}
          @input=${(e: Event) => props.onCredentialChange("clientSecret", (e.target as HTMLInputElement).value)}
          placeholder="Your OAuth Client Secret"
        />
      </label>

      <label class="field" style="margin-top: 12px;">
        <span>Project ID</span>
        <input
          type="text"
          .value=${props.credentials.projectId}
          @input=${(e: Event) => props.onCredentialChange("projectId", (e.target as HTMLInputElement).value)}
          placeholder="proj_..."
        />
      </label>

      <div class="row" style="gap: 12px; margin-top: 12px;">
        <label class="field" style="flex: 1;">
          <span>Environment</span>
          <select
            .value=${props.credentials.environment}
            @change=${(e: Event) => props.onCredentialChange("environment", (e.target as HTMLSelectElement).value)}
          >
            <option value="development">Development</option>
            <option value="production">Production</option>
          </select>
        </label>

        <label class="field" style="flex: 1;">
          <span>External User ID</span>
          <input
            type="text"
            .value=${props.externalUserId}
            @input=${(e: Event) => props.onExternalUserIdChange((e.target as HTMLInputElement).value)}
            placeholder="clawdbot"
          />
        </label>
      </div>

      <div class="row" style="margin-top: 16px; gap: 8px;">
        <button class="btn primary" ?disabled=${props.loading} @click=${props.onSaveCredentials}>
          ${props.loading ? "Saving..." : "Save Credentials"}
        </button>
        <button class="btn" @click=${props.onCancelCredentials}>Cancel</button>
      </div>
    </div>
  `;
}

function renderConnectedApps(props: PipedreamProps) {
  if (props.connectedApps.length === 0) return nothing;

  return html`
    <section class="card" style="margin-top: 16px;">
      <div class="card-title">✅ Connected Apps</div>
      <div class="card-sub">Apps your agent can use. Tokens refresh automatically.</div>
      
      <div class="list" style="margin-top: 12px;">
        ${props.connectedApps.map((app) => renderConnectedApp(app, props))}
      </div>
    </section>
  `;
}

function renderConnectedApp(app: PipedreamApp, props: PipedreamProps) {
  const isTesting = props.testingApp === app.slug;
  const isRefreshing = props.refreshingApp === app.slug;

  return html`
    <div class="list-item">
      <div class="list-main">
        <div class="list-title">
          <span style="font-size: 20px; margin-right: 8px;">${app.icon}</span>
          ${app.name}
        </div>
        <div class="list-sub">
          ${app.accountName ? `Connected as ${app.accountName}` : "Connected"}
          ${app.toolCount ? ` · ${app.toolCount} tools available` : ""}
        </div>
      </div>
      <div class="list-actions">
        <button
          class="btn small"
          ?disabled=${isRefreshing}
          @click=${() => props.onRefreshToken(app.slug)}
          title="Refresh OAuth token"
        >
          ${isRefreshing ? "..." : "🔄"}
        </button>
        <button
          class="btn small"
          ?disabled=${isTesting}
          @click=${() => props.onTestApp(app.slug)}
        >
          ${isTesting ? "Testing..." : "Test"}
        </button>
        <button
          class="btn small danger"
          @click=${() => props.onDisconnectApp(app.slug)}
        >
          Disconnect
        </button>
      </div>
    </div>
  `;
}

function renderAvailableApps(props: PipedreamProps) {
  // Filter out already connected apps
  const connectedSlugs = new Set(props.connectedApps.map((a) => a.slug));
  const available = FEATURED_APPS.filter((a) => !connectedSlugs.has(a.slug));

  return html`
    <section class="card" style="margin-top: 16px;">
      <div class="row" style="justify-content: space-between; align-items: flex-start;">
        <div>
          <div class="card-title">➕ Available Apps</div>
          <div class="card-sub">Connect more apps to expand your agent's capabilities.</div>
        </div>
        <button class="btn" @click=${props.onOpenAppBrowser}>
          Browse All Apps
        </button>
      </div>
      
      <div style="display: grid; grid-template-columns: repeat(auto-fill, minmax(200px, 1fr)); gap: 12px; margin-top: 16px;">
        ${available.slice(0, 12).map((app) => renderAvailableApp(app, props))}
      </div>
      
      <!-- Manual slug entry -->
      <div style="margin-top: 20px; padding-top: 16px; border-top: 1px solid var(--border);">
        <div class="muted" style="font-size: 13px; margin-bottom: 8px;">
          Know the app slug? Connect directly:
        </div>
        <div class="row" style="gap: 8px;">
          <input
            type="text"
            .value=${props.manualSlug}
            @input=${(e: Event) => props.onManualSlugChange((e.target as HTMLInputElement).value)}
            placeholder="e.g., spotify, notion, stripe..."
            style="flex: 1; max-width: 300px;"
          />
          <button
            class="btn primary small"
            ?disabled=${!props.manualSlug.trim() || props.connectingApp === props.manualSlug}
            @click=${props.onConnectManualSlug}
          >
            ${props.connectingApp === props.manualSlug ? "Connecting..." : "Connect"}
          </button>
        </div>
        <div class="muted" style="font-size: 12px; margin-top: 6px;">
          Find app slugs at
          <a href="https://mcp.pipedream.com" target="_blank">mcp.pipedream.com</a>
        </div>
      </div>
    </section>
    
    ${props.showAppBrowser ? renderAppBrowserModal(props) : nothing}
  `;
}

function renderAvailableApp(app: PipedreamApp, props: PipedreamProps) {
  const isConnecting = props.connectingApp === app.slug;

  return html`
    <div style="
      padding: 12px;
      border: 1px solid var(--border);
      border-radius: 8px;
      background: var(--bg-secondary);
      display: flex;
      align-items: center;
      justify-content: space-between;
    ">
      <div style="display: flex; align-items: center; gap: 8px;">
        <span style="font-size: 20px;">${app.icon}</span>
        <span style="font-weight: 500;">${app.name}</span>
      </div>
      <button
        class="btn small primary"
        ?disabled=${isConnecting}
        @click=${() => props.onConnectApp(app.slug)}
      >
        ${isConnecting ? "..." : "Connect"}
      </button>
    </div>
  `;
}

function renderAppBrowserModal(props: PipedreamProps) {
  const connectedSlugs = new Set(props.connectedApps.map((a) => a.slug));
  const search = props.appBrowserSearch.toLowerCase().trim();
  
  // Use allApps if loaded, otherwise fall back to ALL_APPS
  const appsToSearch = props.allApps.length > 0 ? props.allApps : ALL_APPS;
  
  // Filter apps by search query and exclude connected
  const filteredApps = appsToSearch.filter((app) => {
    if (connectedSlugs.has(app.slug)) return false;
    if (!search) return true;
    return (
      app.name.toLowerCase().includes(search) ||
      app.slug.toLowerCase().includes(search)
    );
  });

  // Group apps by category based on icon/slug patterns
  const categories = groupAppsByCategory(filteredApps);

  return html`
    <div class="modal-backdrop" @click=${props.onCloseAppBrowser}>
      <div class="modal" style="width: 90%; max-width: 900px; max-height: 85vh; display: flex; flex-direction: column;" @click=${(e: Event) => e.stopPropagation()}>
        <div class="modal-header" style="flex-shrink: 0;">
          <h2 style="margin: 0;">Browse Apps</h2>
          <button class="btn small" @click=${props.onCloseAppBrowser}>✕</button>
        </div>
        
        <div style="padding: 16px; border-bottom: 1px solid var(--border); flex-shrink: 0;">
          ${props.error
            ? html`<div class="callout danger" style="margin-bottom: 12px; font-size: 13px;">${props.error}</div>`
            : nothing}
          ${props.success
            ? html`<div class="callout success" style="margin-bottom: 12px; font-size: 13px;">${props.success}</div>`
            : nothing}
          ${!props.configured
            ? html`<div class="callout warning" style="margin-bottom: 12px; font-size: 13px;">
                ⚠️ Configure your Pipedream credentials first before connecting apps.
                <a href="#" @click=${(e: Event) => { e.preventDefault(); props.onCloseAppBrowser(); props.onConfigure(); }} style="margin-left: 4px;">Configure Now →</a>
              </div>`
            : nothing}
          <input
            type="text"
            .value=${props.appBrowserSearch}
            @input=${(e: Event) => props.onAppBrowserSearchChange((e.target as HTMLInputElement).value)}
            placeholder="Search apps by name or slug..."
            style="width: 100%; font-size: 16px;"
            autofocus
          />
          <div class="muted" style="margin-top: 8px; font-size: 13px;">
            ${filteredApps.length} apps available
            ${props.loadingApps ? html` <span style="margin-left: 8px;">Loading more...</span>` : nothing}
          </div>
        </div>
        
        <div style="flex: 1; overflow-y: auto; padding: 16px;">
          ${props.loadingApps && filteredApps.length === 0
            ? html`<div style="text-align: center; padding: 40px; color: var(--muted);">Loading apps...</div>`
            : filteredApps.length === 0
            ? html`
                <div style="text-align: center; padding: 40px;">
                  <div style="font-size: 48px; margin-bottom: 16px;">🔍</div>
                  <div style="color: var(--muted);">No apps found matching "${props.appBrowserSearch}"</div>
                  <div class="muted" style="margin-top: 8px; font-size: 13px;">
                    Try a different search, or enter the slug directly in "Connect by slug" below
                  </div>
                </div>
              `
            : search
            ? renderAppGrid(filteredApps, props)
            : renderCategorizedApps(categories, props)}
        </div>
        
        <div style="padding: 16px; border-top: 1px solid var(--border); flex-shrink: 0;">
          <div class="muted" style="font-size: 13px; margin-bottom: 8px;">
            Can't find your app? Enter the slug directly:
          </div>
          <div class="row" style="gap: 8px;">
            <input
              type="text"
              .value=${props.manualSlug}
              @input=${(e: Event) => props.onManualSlugChange((e.target as HTMLInputElement).value)}
              placeholder="e.g., spotify, notion, stripe..."
              style="flex: 1;"
              @keydown=${(e: KeyboardEvent) => {
                if (e.key === "Enter" && props.manualSlug.trim()) {
                  props.onConnectManualSlug();
                }
              }}
            />
            <button
              class="btn primary"
              ?disabled=${!props.manualSlug.trim() || props.connectingApp === props.manualSlug}
              @click=${props.onConnectManualSlug}
            >
              ${props.connectingApp === props.manualSlug ? "Connecting..." : "Connect by Slug"}
            </button>
          </div>
          <div class="muted" style="font-size: 12px; margin-top: 6px;">
            Find all available app slugs at
            <a href="https://mcp.pipedream.com" target="_blank">mcp.pipedream.com</a>
          </div>
        </div>
      </div>
    </div>
  `;
}

function renderAppGrid(apps: PipedreamApp[], props: PipedreamProps) {
  return html`
    <div style="display: grid; grid-template-columns: repeat(auto-fill, minmax(200px, 1fr)); gap: 12px;">
      ${apps.map((app) => renderBrowserApp(app, props))}
    </div>
  `;
}

function renderCategorizedApps(categories: Record<string, PipedreamApp[]>, props: PipedreamProps) {
  const categoryOrder = [
    "Google", "Communication", "Project Management", "Development",
    "AI & ML", "Social Media", "CRM & Sales", "Storage & Files",
    "Databases", "Payments", "Forms & Surveys", "Analytics",
    "Productivity", "Support", "Other"
  ];

  return html`
    ${categoryOrder.map((category) => {
      const apps = categories[category];
      if (!apps || apps.length === 0) return nothing;
      
      return html`
        <div style="margin-bottom: 24px;">
          <h3 style="margin: 0 0 12px 0; font-size: 14px; text-transform: uppercase; color: var(--muted); letter-spacing: 0.5px;">
            ${category}
          </h3>
          <div style="display: grid; grid-template-columns: repeat(auto-fill, minmax(180px, 1fr)); gap: 10px;">
            ${apps.map((app) => renderBrowserApp(app, props))}
          </div>
        </div>
      `;
    })}
  `;
}

function renderBrowserApp(app: PipedreamApp, props: PipedreamProps) {
  const isConnecting = props.connectingApp === app.slug;

  return html`
    <div style="
      padding: 10px 12px;
      border: 1px solid var(--border);
      border-radius: 8px;
      background: var(--bg-secondary);
      display: flex;
      align-items: center;
      justify-content: space-between;
      gap: 8px;
    ">
      <div style="display: flex; align-items: center; gap: 8px; min-width: 0; flex: 1;">
        <span style="font-size: 18px; flex-shrink: 0;">${app.icon}</span>
        <span style="font-weight: 500; font-size: 13px; white-space: nowrap; overflow: hidden; text-overflow: ellipsis;" title="${app.name}">
          ${app.name}
        </span>
      </div>
      <button
        class="btn small primary"
        style="flex-shrink: 0; padding: 4px 10px; font-size: 12px;"
        ?disabled=${isConnecting}
        @click=${(e: Event) => {
          e.stopPropagation();
          console.log("[Pipedream] Connect button clicked for:", app.slug);
          props.onConnectApp(app.slug);
        }}
      >
        ${isConnecting ? "..." : "Connect"}
      </button>
    </div>
  `;
}

function groupAppsByCategory(apps: PipedreamApp[]): Record<string, PipedreamApp[]> {
  const categories: Record<string, PipedreamApp[]> = {};
  
  const categoryMap: Record<string, string> = {
    // Google
    "gmail": "Google", "google-calendar": "Google", "google-sheets": "Google",
    "google-drive": "Google", "google-docs": "Google", "google-tasks": "Google",
    "youtube": "Google", "youtube-data-api": "Google", "google_analytics": "Google",
    "google-forms": "Google",
    // Communication
    "slack": "Communication", "discord": "Communication", "telegram-bot-api": "Communication",
    "twilio": "Communication", "sendgrid": "Communication", "mailgun": "Communication",
    "mailchimp": "Communication", "intercom": "Communication", "zoom": "Communication",
    "microsoft_teams": "Communication",
    // Project Management
    "notion": "Project Management", "linear": "Project Management", "asana": "Project Management",
    "trello": "Project Management", "monday": "Project Management", "clickup": "Project Management",
    "basecamp": "Project Management", "jira": "Project Management", "todoist": "Project Management",
    // Development
    "github": "Development", "gitlab": "Development", "bitbucket": "Development",
    "vercel": "Development", "netlify": "Development", "railway": "Development",
    "render": "Development", "fly-io": "Development", "sentry": "Development",
    "datadog": "Development",
    // AI & ML
    "openai": "AI & ML", "anthropic": "AI & ML", "replicate": "AI & ML",
    "huggingface": "AI & ML", "stability-ai": "AI & ML", "eleven-labs": "AI & ML",
    "deepgram": "AI & ML",
    // Social Media
    "twitter": "Social Media", "linkedin": "Social Media", "facebook_pages": "Social Media",
    "instagram_business": "Social Media", "tiktok": "Social Media", "reddit": "Social Media",
    "pinterest": "Social Media",
    // CRM & Sales
    "hubspot": "CRM & Sales", "salesforce_rest_api": "CRM & Sales", "pipedrive": "CRM & Sales",
    "zoho-crm": "CRM & Sales", "freshsales": "CRM & Sales", "close": "CRM & Sales",
    // Storage & Files
    "dropbox": "Storage & Files", "box": "Storage & Files", "onedrive": "Storage & Files",
    "aws": "Storage & Files", "cloudinary": "Storage & Files",
    // Databases
    "airtable": "Databases", "mongodb": "Databases", "postgresql": "Databases",
    "mysql": "Databases", "redis": "Databases", "snowflake": "Databases",
    "supabase": "Databases", "firebase": "Databases",
    // Payments
    "stripe": "Payments", "paypal": "Payments", "square": "Payments",
    "shopify": "Payments", "gumroad": "Payments", "lemonsqueezy": "Payments",
    // Forms & Surveys
    "typeform": "Forms & Surveys", "surveymonkey": "Forms & Surveys",
    "tally": "Forms & Surveys", "jotform": "Forms & Surveys",
    // Analytics
    "mixpanel": "Analytics", "amplitude": "Analytics", "segment": "Analytics",
    "posthog": "Analytics",
    // Productivity
    "calendar": "Productivity", "loom": "Productivity", "coda": "Productivity",
    "evernote": "Productivity", "obsidian": "Productivity", "roam": "Productivity",
    // Support
    "zendesk": "Support", "freshdesk": "Support", "help-scout": "Support",
    "crisp": "Support",
  };
  
  for (const app of apps) {
    const category = categoryMap[app.slug] || "Other";
    if (!categories[category]) {
      categories[category] = [];
    }
    categories[category].push(app);
  }
  
  return categories;
}

// Export for use in controller
export { ALL_APPS };
