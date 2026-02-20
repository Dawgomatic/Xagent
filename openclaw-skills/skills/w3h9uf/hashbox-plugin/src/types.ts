export type PayloadType = "article" | "metric" | "audit";

export interface MetricEntry {
  label: string;
  value: number;
  unit?: string;
  trend?: "up" | "down" | "flat";
}

export interface AuditEntry {
  timestamp: string;
  event: string;
  severity: "info" | "warn" | "critical";
  details?: string;
}

export type HashBoxPayload =
  | {
      type: "article";
      channelName: string;
      channelIcon: string;
      title: string;
      body: string;
    }
  | {
      type: "metric";
      channelName: string;
      channelIcon: string;
      title: string;
      metrics: MetricEntry[];
    }
  | {
      type: "audit";
      channelName: string;
      channelIcon: string;
      title: string;
      entries: AuditEntry[];
    };

export interface PluginTool {
  name: string;
  description: string;
  execute: (...args: unknown[]) => Promise<unknown>;
}

export interface PluginAction {
  name: string;
  description: string;
  execute: (...args: unknown[]) => Promise<unknown>;
}

export interface HashBoxPlugin {
  name: string;
  description: string;
  tools: PluginTool[];
  actions: PluginAction[];
}
