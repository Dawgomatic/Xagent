/**
 * Agent path utilities
 */

import path from "node:path";
import os from "node:os";

/**
 * Resolve the Clawdbot agent directory (~/.clawdbot)
 */
export function resolveClawdbotAgentDir(): string {
  return process.env.CLAWDBOT_DIR || path.join(os.homedir(), ".clawdbot");
}
