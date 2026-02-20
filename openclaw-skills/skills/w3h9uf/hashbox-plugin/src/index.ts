import type { HashBoxPlugin } from "./types.js";
import { configureHashBox } from "./setupHashBox.js";
import { sendHashBoxNotification } from "./pushToHashBox.js";

export { configureHashBox } from "./setupHashBox.js";
export { sendHashBoxNotification } from "./pushToHashBox.js";
export type {
  PayloadType,
  MetricEntry,
  AuditEntry,
  HashBoxPayload,
  PluginTool,
  PluginAction,
  HashBoxPlugin,
} from "./types.js";

export const hashBoxPlugin: HashBoxPlugin = {
  name: "hashbox-plugin",
  description:
    "Connects an AI agent to the HashBox iOS app via Firebase webhook for push notifications",
  tools: [
    {
      name: "configure_hashbox",
      description: "Configure HashBox with an API token (must start with HB-)",
      execute: async (...args: unknown[]) => {
        const token = args[0] as string;
        return configureHashBox(token);
      },
    },
  ],
  actions: [
    {
      name: "send_hashbox_notification",
      description: "Send a push notification to the HashBox iOS app",
      execute: async (...args: unknown[]) => {
        const [payloadType, channelName, channelIcon, title, contentOrData] =
          args as Parameters<typeof sendHashBoxNotification>;
        return sendHashBoxNotification(
          payloadType, channelName, channelIcon, title, contentOrData
        );
      },
    },
  ],
};
