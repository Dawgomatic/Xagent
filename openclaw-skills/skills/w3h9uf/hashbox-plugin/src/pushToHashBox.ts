import { readFile } from "node:fs/promises";
import { join } from "node:path";
import type { PayloadType, MetricEntry, AuditEntry, HashBoxPayload } from "./types.js";

const CONFIG_FILENAME = "hashbox_config.json";
const WEBHOOK_BASE_URL =
  "https://webhook-vcphors6kq-uc.a.run.app/webhook";

interface HashBoxConfig {
  token: string;
}

interface SendResult {
  status: number;
  message: string;
}

async function loadConfig(): Promise<HashBoxConfig> {
  const configPath = join(process.cwd(), CONFIG_FILENAME);
  try {
    const raw = await readFile(configPath, "utf-8");
    return JSON.parse(raw) as HashBoxConfig;
  } catch {
    throw new Error(
      "HashBox config not found. Run configureHashBox first."
    );
  }
}

function buildPayload(
  payloadType: PayloadType,
  channelName: string,
  channelIcon: string,
  title: string,
  contentOrData: string | MetricEntry[] | AuditEntry[]
): HashBoxPayload {
  if (payloadType === "article") {
    return {
      type: "article",
      channelName,
      channelIcon,
      title,
      body: contentOrData as string,
    };
  }
  if (payloadType === "metric") {
    return {
      type: "metric",
      channelName,
      channelIcon,
      title,
      metrics: contentOrData as MetricEntry[],
    };
  }
  return {
    type: "audit",
    channelName,
    channelIcon,
    title,
    entries: contentOrData as AuditEntry[],
  };
}

export async function sendHashBoxNotification(
  payloadType: PayloadType,
  channelName: string,
  channelIcon: string,
  title: string,
  contentOrData: string | MetricEntry[] | AuditEntry[]
): Promise<SendResult> {
  const config = await loadConfig();
  const url = `${WEBHOOK_BASE_URL}?token=${config.token}`;
  const payload = buildPayload(
    payloadType, channelName, channelIcon, title, contentOrData
  );
  const body = JSON.stringify(payload);

  try {
    const response = await fetch(url, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body,
    });

    return {
      status: response.status,
      message: response.ok
        ? "Notification sent successfully"
        : `Request failed with status ${response.status}`,
    };
  } catch (err: unknown) {
    const errorMessage =
      err instanceof Error ? err.message : "Unknown network error";
    return {
      status: 0,
      message: `Network error: ${errorMessage}`,
    };
  }
}
