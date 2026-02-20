/**
 * NIMA Resilient Hook Wrappers
 * =============================
 * Provides error-safe wrappers for OpenClaw hook handlers.
 * Prevents hook failures from crashing the agent system.
 *
 * Features:
 *   - HookExecutionError class for typed error handling
 *   - resilientHook - async wrapper with fallback
 *   - resilientHookSync - sync wrapper with fallback
 *   - withRetry - retry wrapper with exponential backoff
 *
 * Uses error-handling.js for logging utilities.
 *
 * Author: NIMA Core Team
 * Date: Feb 17, 2026
 */

/**
 * Custom error class for hook execution failures.
 * Preserves original error context while providing structured metadata.
 */
export class HookExecutionError extends Error {
  /**
   * @param {string} hookName - Name of the hook that failed
   * @param {Error} originalError - The original error thrown
   * @param {Object} metadata - Additional context about the failure
   */
  constructor(hookName, originalError, metadata = {}) {
    super(`Hook '${hookName}' failed: ${originalError.message}`);
    this.name = 'HookExecutionError';
    this.hookName = hookName;
    this.originalError = originalError;
    this.metadata = metadata;
    this.timestamp = new Date().toISOString();

    // Preserve stack trace
    if (Error.captureStackTrace) {
      Error.captureStackTrace(this, HookExecutionError);
    }
  }

  /**
   * Get a formatted error summary for logging.
   * @returns {string} Formatted error summary
   */
  getSummary() {
    const parts = [
      `[${this.name}]`,
      `Hook: ${this.hookName}`,
      `Error: ${this.originalError.message}`,
    ];
    if (Object.keys(this.metadata).length > 0) {
      parts.push(`Context: ${JSON.stringify(this.metadata)}`);
    }
    return parts.join(' ');
  }
}

/**
 * Log hook error to stderr (consistent with NIMA conventions).
 * @param {string} hookName - Name of the failed hook
 * @param {Error} err - The error that occurred
 */
function logHookError(hookName, err) {
  const isHookError = err instanceof HookExecutionError;
  const message = isHookError ? err.getSummary() : `[HookError] Hook '${hookName}' failed: ${err.message}`;
  console.error(message);
}

/**
 * Async wrapper for hook handlers with error recovery.
 *
 * Catches any errors thrown by the hook function, logs them,
 * and returns a fallback result instead of crashing.
 *
 * @param {string} hookName - Name of the hook (for logging)
 * @param {Function} hookFn - Async hook function to wrap
 * @param {any} fallbackResult - Value to return on error (default: undefined)
 * @returns {Function} Wrapped async function
 *
 * @example
 * api.on("before_agent_start", resilientHook("before_agent_start", async (event, ctx) => {
 *   // Hook logic that might throw
 *   return { prependContext: "..." };
 * }, undefined));
 */
export function resilientHook(hookName, hookFn, fallbackResult = undefined) {
  return async function(event, ctx) {
    try {
      return await hookFn(event, ctx);
    } catch (err) {
      logHookError(hookName, err);
      return fallbackResult;
    }
  };
}

/**
 * Synchronous wrapper for hook handlers with error recovery.
 *
 * Same as resilientHook but for synchronous hooks.
 *
 * @param {string} hookName - Name of the hook (for logging)
 * @param {Function} hookFn - Synchronous hook function to wrap
 * @param {any} fallbackResult - Value to return on error (default: undefined)
 * @returns {Function} Wrapped sync function
 *
 * @example
 * api.on("message_received", resilientHookSync("message_received", (event, ctx) => {
 *   // Sync hook logic that might throw
 * }, undefined));
 */
export function resilientHookSync(hookName, hookFn, fallbackResult = undefined) {
  return function(event, ctx) {
    try {
      return hookFn(event, ctx);
    } catch (err) {
      logHookError(hookName, err);
      return fallbackResult;
    }
  };
}

/**
 * Sleep helper for retry delays.
 * @param {number} ms - Milliseconds to sleep
 * @returns {Promise<void>}
 */
function sleep(ms) {
  return new Promise(resolve => setTimeout(resolve, ms));
}

/**
 * Retry wrapper for hook handlers with exponential backoff.
 *
 * Retries the hook function on transient failures with exponential backoff.
 * Uses a simplified transient error check suitable for hook operations.
 *
 * @param {string} hookName - Name of the hook (for logging)
 * @param {Function} hookFn - Async hook function to wrap
 * @param {number} maxRetries - Maximum retry attempts (default: 3)
 * @param {number} retryDelay - Base delay in milliseconds (default: 100)
 * @returns {Function} Wrapped async function with retry logic
 *
 * @example
 * api.on("before_agent_start", withRetry("before_agent_start", async (event, ctx) => {
 *   // Hook logic that might fail transiently
 * }, 3, 200));
 */
export function withRetry(hookName, hookFn, maxRetries = 3, retryDelay = 100) {
  return async function(event, ctx) {
    let lastError;

    for (let attempt = 1; attempt <= maxRetries; attempt++) {
      try {
        return await hookFn(event, ctx);
      } catch (err) {
        lastError = err;

        // Check if this might be a transient error worth retrying
        const isTransient = isTransientError(err);

        if (!isTransient || attempt >= maxRetries) {
          // Either not transient or out of retries
          break;
        }

        // Calculate exponential backoff delay
        const delay = retryDelay * Math.pow(2, attempt - 1);
        console.error(`[withRetry] Hook '${hookName}' failed (attempt ${attempt}/${maxRetries}), retrying in ${delay}ms: ${err.message}`);

        await sleep(delay);
      }
    }

    // All retries exhausted or non-transient error
    logHookError(hookName, lastError);
    return undefined;
  };
}

/**
 * Check if an error is likely transient and worth retrying.
 *
 * Transient errors include:
 * - Network timeouts (ETIMEDOUT, ECONNRESET)
 * - Rate limits (429)
 * - Database locks (SQLITE_BUSY, SQLITE_LOCKED)
 * - API service errors (500, 502, 503, 504)
 *
 * @param {Error} err - The error to check
 * @returns {boolean} - True if error is likely transient
 */
function isTransientError(err) {
  if (!err) return false;

  const message = (err.message || '').toLowerCase();
  const code = err.code || '';
  const status = err.status || err.statusCode || 0;

  // Network errors
  if (['ETIMEDOUT', 'ECONNRESET', 'ECONNREFUSED', 'ENOTFOUND', 'EPIPE'].includes(code)) {
    return true;
  }

  // SQLite locks
  if (message.includes('sqlite_busy') || message.includes('sqlite_locked') ||
      message.includes('database is locked')) {
    return true;
  }

  // HTTP status codes
  if (status === 429 || (status >= 500 && status < 600)) {
    return true;
  }

  // Timeout errors
  if (message.includes('timeout') || message.includes('timed out')) {
    return true;
  }

  // Rate limit messages
  if (message.includes('rate limit') || message.includes('too many requests')) {
    return true;
  }

  return false;
}

export default {
  HookExecutionError,
  resilientHook,
  resilientHookSync,
  withRetry,
};
