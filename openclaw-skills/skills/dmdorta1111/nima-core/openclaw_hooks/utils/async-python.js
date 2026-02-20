/**
 * Async Python Execution Utility
 * ==============================
 * Provides non-blocking Python execution with:
 * - Async/Await interface (promisified execFile)
 * - Circuit Breaker pattern (prevent cascade failures)
 * - Rate Limiting (prevent process spawn exhaustion)
 *
 * Usage:
 *   import { execPython } from "../utils/async-python.js";
 *   const result = await execPython(scriptPath, args, { timeout: 5000 });
 */

import { promisify } from "node:util";
import { execFile } from "node:child_process";
const execFileAsync = promisify(execFile);

// =============================================================================
// RATE LIMITER (Token Bucket)
// =============================================================================

class SimpleRateLimiter {
  constructor(tokensPerInterval, interval, fireImmediately = false) {
    this.tokensPerInterval = tokensPerInterval;
    this.interval = interval === 'second' ? 1000 : interval;
    this.tokens = tokensPerInterval;
    this.lastRefill = Date.now();
  }

  async removeTokens(count) {
    this.refill();
    if (this.tokens >= count) {
      this.tokens -= count;
      return true;
    }
    
    // Wait for tokens
    const now = Date.now();
    const timeToNextRefill = this.interval - (now - this.lastRefill);
    if (timeToNextRefill > 0) {
      await new Promise(resolve => setTimeout(resolve, timeToNextRefill));
      this.refill();
    }
    
    if (this.tokens >= count) {
      this.tokens -= count;
      return true;
    }
    return false; // Should not happen if we waited correctly
  }

  refill() {
    const now = Date.now();
    const elapsed = now - this.lastRefill;
    if (elapsed >= this.interval) {
      this.tokens = this.tokensPerInterval;
      this.lastRefill = now;
    }
  }
}

// Global rate limiter for Python spawns
// Limit: 10 processes per second (bursty is fine, but sustained high load is bad)
const pythonSpawnLimiter = new SimpleRateLimiter(10, 'second');


// =============================================================================
// CIRCUIT BREAKER
// =============================================================================

class CircuitBreaker {
  constructor(name, maxFailures = 5, resetAfterMs = 60000) {
    this.name = name;
    this.maxFailures = maxFailures;
    this.resetAfterMs = resetAfterMs;
    this.failures = 0;
    this.lastFailureTime = 0;
    this.state = "CLOSED"; // CLOSED (normal), OPEN (broken), HALF-OPEN (testing)
  }

  isOpen() {
    if (this.state === "OPEN") {
      const now = Date.now();
      if (now - this.lastFailureTime > this.resetAfterMs) {
        this.state = "HALF-OPEN"; // Try one request
        return false;
      }
      return true;
    }
    return false;
  }

  recordSuccess() {
    this.failures = 0;
    this.state = "CLOSED";
  }

  recordFailure() {
    this.failures++;
    this.lastFailureTime = Date.now();
    if (this.failures >= this.maxFailures) {
      this.state = "OPEN";
      console.error(`[CircuitBreaker:${this.name}] OPEN - too many failures (${this.failures})`);
    }
  }
}

// Global breakers map
const breakers = new Map();

function getBreaker(id) {
  if (!breakers.has(id)) {
    breakers.set(id, new CircuitBreaker(id));
  }
  return breakers.get(id);
}


// =============================================================================
// EXPORTED API
// =============================================================================

/**
 * Execute a Python script asynchronously with safety rails.
 * 
 * @param {string} scriptPath - Path to Python script (or "python3" command if args[0] is script)
 * @param {string[]} args - Arguments passed to Python
 * @param {object} options - execFile options (timeout, cwd, etc.) + { breakerId: string }
 * @returns {Promise<string>} - stdout
 */
export async function execPython(scriptPath, args = [], options = {}) {
  const { breakerId = "default", ...execOptions } = options;
  const breaker = getBreaker(breakerId);

  // 1. Circuit Breaker Check
  if (breaker.isOpen()) {
    throw new Error(`Circuit breaker '${breakerId}' is OPEN. skipping execution.`);
  }

  // 2. Rate Limiting
  await pythonSpawnLimiter.removeTokens(1);

  try {
    // 3. Execution
    // Handle case where scriptPath is just "python3" and script is in args
    const cmd = scriptPath.endsWith(".py") ? "python3" : scriptPath;
    const finalArgs = scriptPath.endsWith(".py") ? [scriptPath, ...args] : args;

    const { stdout, stderr } = await execFileAsync(cmd, finalArgs, {
      encoding: "utf-8",
      maxBuffer: 10 * 1024 * 1024, // 10MB buffer
      ...execOptions
    });

    if (stderr && stderr.trim().length > 0) {
        // Log stderr but don't fail unless it exit code was non-zero (handled by catch)
        // Some scripts write non-errors to stderr
        // console.debug(`[execPython] stderr: ${stderr.substring(0, 200)}`);
    }

    breaker.recordSuccess();
    return stdout;

  } catch (err) {
    breaker.recordFailure();
    throw err;
  }
}
