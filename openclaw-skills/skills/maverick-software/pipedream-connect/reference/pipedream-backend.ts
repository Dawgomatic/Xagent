import * as fs from "node:fs";
import * as path from "node:path";
import * as os from "node:os";

import type { GatewayRequestHandlers } from "./types.js";

// Pipedream credentials storage path
function getPipedreamCredentialsPath(): string {
  return path.join(os.homedir(), "clawd", "config", "pipedream-credentials.json");
}

interface PipedreamCredentials {
  clientId: string;
  clientSecret: string;
  projectId: string;
  environment: string;
  externalUserId: string;
}

function readPipedreamCredentials(): PipedreamCredentials | null {
  const credPath = getPipedreamCredentialsPath();
  try {
    if (fs.existsSync(credPath)) {
      return JSON.parse(fs.readFileSync(credPath, "utf-8"));
    }
  } catch {
    // ignore
  }
  return null;
}

function writePipedreamCredentials(creds: PipedreamCredentials): boolean {
  const credPath = getPipedreamCredentialsPath();
  try {
    const dir = path.dirname(credPath);
    if (!fs.existsSync(dir)) {
      fs.mkdirSync(dir, { recursive: true });
    }
    fs.writeFileSync(credPath, JSON.stringify(creds, null, 2), { mode: 0o600 });
    return true;
  } catch {
    return false;
  }
}

// Get access token from Pipedream
async function getPipedreamAccessToken(clientId: string, clientSecret: string): Promise<string> {
  const response = await fetch("https://api.pipedream.com/v1/oauth/token", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({
      grant_type: "client_credentials",
      client_id: clientId,
      client_secret: clientSecret,
    }),
  });

  if (!response.ok) {
    const errorText = await response.text();
    throw new Error(`Pipedream auth failed: ${errorText}`);
  }

  const data = await response.json();
  if (!data.access_token) {
    throw new Error("No access_token in Pipedream response");
  }

  return data.access_token;
}

// Find mcporter config file
function findMcporterConfig(): string | null {
  const candidates = [
    path.join(os.homedir(), "clawd", "config", "mcporter.json"),
    path.join(os.homedir(), "clawdbot", "config", "mcporter.json"),
    path.join(os.homedir(), ".config", "mcporter", "config.json"),
    path.join(os.homedir(), ".mcporter.json"),
  ];

  for (const configPath of candidates) {
    if (fs.existsSync(configPath)) {
      return configPath;
    }
  }
  return null;
}

function readMcporterConfig(): { servers: Record<string, unknown> } | null {
  const configPath = findMcporterConfig();
  if (!configPath) return null;

  try {
    const content = fs.readFileSync(configPath, "utf-8");
    const config = JSON.parse(content);
    return { servers: config.mcpServers || {} };
  } catch {
    return null;
  }
}

function writeMcporterConfig(servers: Record<string, unknown>): boolean {
  const configPath = findMcporterConfig();
  if (!configPath) {
    // Create default path
    const defaultPath = path.join(os.homedir(), "clawd", "config", "mcporter.json");
    const dir = path.dirname(defaultPath);
    if (!fs.existsSync(dir)) {
      fs.mkdirSync(dir, { recursive: true });
    }
    fs.writeFileSync(defaultPath, JSON.stringify({ mcpServers: servers, imports: [] }, null, 2));
    return true;
  }

  try {
    const content = fs.readFileSync(configPath, "utf-8");
    const config = JSON.parse(content);
    config.mcpServers = servers;
    fs.writeFileSync(configPath, JSON.stringify(config, null, 2));
    return true;
  } catch {
    return false;
  }
}

export const pipedreamHandlers: GatewayRequestHandlers = {
  "pipedream.status": async ({ respond }) => {
    try {
      const config = readMcporterConfig();

      const apps: Array<{
        slug: string;
        name: string;
        serverName: string;
      }> = [];

      // First check for stored credentials (from saveCredentials)
      let credentials = readPipedreamCredentials();

      // Also scan MCP servers for connected apps and legacy credentials
      if (config) {
        for (const [name, server] of Object.entries(config.servers)) {
          if (name.startsWith("pipedream-")) {
            const srv = server as {
              env?: Record<string, string>;
              headers?: Record<string, string>;
            };
            const env = srv.env || {};
            const headers = srv.headers || {};

            // Fallback: extract credentials from first pipedream server if not stored
            if (!credentials && env.PIPEDREAM_CLIENT_ID) {
              credentials = {
                clientId: env.PIPEDREAM_CLIENT_ID || "",
                clientSecret: env.PIPEDREAM_CLIENT_SECRET || "",
                projectId: env.PIPEDREAM_PROJECT_ID || "",
                environment: env.PIPEDREAM_ENVIRONMENT || "development",
                externalUserId:
                  env.PIPEDREAM_AGENT_ID || headers["x-pd-external-user-id"] || "clawdbot",
              };
            }

            // Extract app info
            const appSlug = env.PIPEDREAM_APP_SLUG || headers["x-pd-app-slug"];
            if (appSlug) {
              apps.push({
                slug: appSlug,
                name: appSlugToName(appSlug),
                serverName: name,
              });
            }
          }
        }
      }

      respond(
        true,
        {
          configured: !!credentials,
          credentials: credentials
            ? {
                clientId: credentials.clientId,
                // Don't expose full secret, just indicate it exists
                hasSecret: !!credentials.clientSecret,
                projectId: credentials.projectId,
                environment: credentials.environment,
                externalUserId: credentials.externalUserId,
              }
            : null,
          apps,
        },
        undefined,
      );
    } catch (e) {
      respond(true, { error: e instanceof Error ? e.message : String(e) }, undefined);
    }
  },

  "pipedream.saveCredentials": async ({ params, respond }) => {
    try {
      const { clientId, clientSecret, projectId, environment, externalUserId } = params as {
        clientId: string;
        clientSecret: string;
        projectId: string;
        environment: string;
        externalUserId: string;
      };

      // Validate credentials by getting an access token from Pipedream
      let accessToken: string;
      try {
        accessToken = await getPipedreamAccessToken(clientId, clientSecret);
      } catch (e) {
        respond(
          true,
          { success: false, error: e instanceof Error ? e.message : String(e) },
          undefined,
        );
        return;
      }

      // Store credentials securely
      const stored = writePipedreamCredentials({
        clientId,
        clientSecret,
        projectId,
        environment,
        externalUserId,
      });

      if (!stored) {
        respond(true, { success: false, error: "Failed to store credentials" }, undefined);
        return;
      }

      // Credentials are valid and stored - return success with the access token
      respond(true, { success: true, accessToken }, undefined);
    } catch (e) {
      respond(
        true,
        { success: false, error: e instanceof Error ? e.message : String(e) },
        undefined,
      );
    }
  },

  "pipedream.getToken": async ({ respond }) => {
    try {
      const credentials = readPipedreamCredentials();
      if (!credentials) {
        respond(true, { success: false, error: "No credentials configured" }, undefined);
        return;
      }

      const accessToken = await getPipedreamAccessToken(
        credentials.clientId,
        credentials.clientSecret,
      );
      respond(
        true,
        {
          success: true,
          accessToken,
          credentials: {
            clientId: credentials.clientId,
            projectId: credentials.projectId,
            environment: credentials.environment,
            externalUserId: credentials.externalUserId,
          },
        },
        undefined,
      );
    } catch (e) {
      respond(
        true,
        { success: false, error: e instanceof Error ? e.message : String(e) },
        undefined,
      );
    }
  },

  "pipedream.getConnectUrl": async ({ params, respond }) => {
    try {
      const { appSlug } = params as { appSlug: string };

      const credentials = readPipedreamCredentials();
      if (!credentials) {
        respond(true, { success: false, error: "No credentials configured" }, undefined);
        return;
      }

      // Get access token
      const accessToken = await getPipedreamAccessToken(
        credentials.clientId,
        credentials.clientSecret,
      );

      // Create a connect token via Pipedream Connect API
      const connectResponse = await fetch(
        `https://api.pipedream.com/v1/connect/${credentials.projectId}/tokens`,
        {
          method: "POST",
          headers: {
            "Content-Type": "application/json",
            Authorization: `Bearer ${accessToken}`,
            "x-pd-environment": credentials.environment,
          },
          body: JSON.stringify({
            external_user_id: credentials.externalUserId,
            app: appSlug.replace(/-/g, "_"), // google-calendar -> google_calendar
          }),
        },
      );

      if (!connectResponse.ok) {
        const errorText = await connectResponse.text();
        respond(
          true,
          { success: false, error: `Failed to create connect token: ${errorText}` },
          undefined,
        );
        return;
      }

      const connectData = await connectResponse.json();

      // Append app parameter to the URL so it goes directly to that app's OAuth
      const appParam = appSlug.replace(/-/g, "_"); // google-calendar -> google_calendar
      const connectUrl = connectData.connect_link_url.includes("?")
        ? `${connectData.connect_link_url}&app=${appParam}`
        : `${connectData.connect_link_url}?app=${appParam}`;

      respond(
        true,
        {
          success: true,
          connectUrl,
          token: connectData.token,
          expiresAt: connectData.expires_at,
        },
        undefined,
      );
    } catch (e) {
      respond(
        true,
        { success: false, error: e instanceof Error ? e.message : String(e) },
        undefined,
      );
    }
  },

  "pipedream.connectApp": async ({ params, respond }) => {
    try {
      const { appSlug, accessToken } = params as {
        appSlug: string;
        accessToken: string;
      };

      // Read stored credentials (don't rely on UI to pass secrets)
      const credentials = readPipedreamCredentials();
      if (!credentials) {
        respond(true, { success: false, error: "No credentials configured" }, undefined);
        return;
      }

      const config = readMcporterConfig() || { servers: {} };
      const serverName = `pipedream-${credentials.externalUserId}-${appSlug}`.replace(
        /[^a-z0-9-]/g,
        "-",
      );

      // MCP endpoint uses underscores (google_calendar) not hyphens (google-calendar)
      const mcpAppSlug = appSlug.replace(/-/g, "_");

      const servers = config.servers as Record<string, unknown>;
      servers[serverName] = {
        baseUrl: "https://remote.mcp.pipedream.net",
        headers: {
          Authorization: `Bearer ${accessToken}`,
          "x-pd-project-id": credentials.projectId,
          "x-pd-environment": credentials.environment,
          "x-pd-external-user-id": credentials.externalUserId,
          "x-pd-app-slug": mcpAppSlug,
          Accept: "application/json, text/event-stream",
          "Content-Type": "application/json",
        },
        env: {
          PIPEDREAM_CLIENT_ID: credentials.clientId,
          PIPEDREAM_CLIENT_SECRET: credentials.clientSecret,
          PIPEDREAM_PROJECT_ID: credentials.projectId,
          PIPEDREAM_ENVIRONMENT: credentials.environment,
          PIPEDREAM_AGENT_ID: credentials.externalUserId,
          PIPEDREAM_APP_SLUG: mcpAppSlug,
        },
      };

      const success = writeMcporterConfig(servers);
      respond(true, { success, serverName }, undefined);
    } catch (e) {
      respond(
        true,
        { success: false, error: e instanceof Error ? e.message : String(e) },
        undefined,
      );
    }
  },

  "pipedream.disconnectApp": async ({ params, respond }) => {
    try {
      const { serverName } = params as { serverName: string };

      const config = readMcporterConfig();
      if (!config) {
        respond(true, { success: false, error: "No config found" }, undefined);
        return;
      }

      const servers = config.servers as Record<string, unknown>;
      delete servers[serverName];

      const success = writeMcporterConfig(servers);
      respond(true, { success }, undefined);
    } catch (e) {
      respond(
        true,
        { success: false, error: e instanceof Error ? e.message : String(e) },
        undefined,
      );
    }
  },

  "pipedream.refreshToken": async ({ params, respond }) => {
    try {
      const { serverName, accessToken } = params as {
        serverName: string;
        accessToken: string;
      };

      const config = readMcporterConfig();
      if (!config) {
        respond(true, { success: false, error: "No config found" }, undefined);
        return;
      }

      const servers = config.servers as Record<string, unknown>;
      const server = servers[serverName] as { headers?: Record<string, string> } | undefined;

      if (!server) {
        respond(true, { success: false, error: "Server not found" }, undefined);
        return;
      }

      if (!server.headers) server.headers = {};
      server.headers.Authorization = `Bearer ${accessToken}`;

      const success = writeMcporterConfig(servers);
      respond(true, { success }, undefined);
    } catch (e) {
      respond(
        true,
        { success: false, error: e instanceof Error ? e.message : String(e) },
        undefined,
      );
    }
  },
};

function appSlugToName(slug: string): string {
  const names: Record<string, string> = {
    gmail: "Gmail",
    "google-calendar": "Google Calendar",
    "google-sheets": "Google Sheets",
    "google-drive": "Google Drive",
    slack: "Slack",
    notion: "Notion",
    github: "GitHub",
    linear: "Linear",
    discord: "Discord",
    twitter: "Twitter/X",
    airtable: "Airtable",
    hubspot: "HubSpot",
    asana: "Asana",
    trello: "Trello",
    dropbox: "Dropbox",
    openai: "OpenAI",
  };
  return (
    names[slug] ||
    slug
      .split("-")
      .map((w) => w.charAt(0).toUpperCase() + w.slice(1))
      .join(" ")
  );
}
