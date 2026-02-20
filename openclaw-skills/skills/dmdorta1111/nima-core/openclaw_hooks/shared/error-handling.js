/**
 * NIMA Shared Error Handling Utilities
 * =====================================
 * Provides robust error handling patterns for all NIMA hooks:
 *   - Retry with exponential backoff
 *   - Safe JSON parsing with validation
 *   - Transient error detection
 *   - Error propagation helpers
 * 
 * Author: NIMA Core Team
 * Date: 2026-02-16
 * 
 * AUDIT FIX: Addresses silent failure issues from code review.
 * See docs/NIMA_HOOKS_CODE_REVIEW.md for details.
 */

/**
 * Sleep helper for async operations.
 * @param {number} ms - Milliseconds to sleep
 * @returns {Promise<void>}
 */
export function sleep(ms) {
  return new Promise(resolve => setTimeout(resolve, ms));
}

/**
 * Detect if an error is likely transient and worth retrying.
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
export function isTransientError(err) {
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
  
  // Voyage/API timeouts
  if (message.includes('timeout') || message.includes('timed out')) {
    return true;
  }
  
  // Rate limit messages
  if (message.includes('rate limit') || message.includes('too many requests')) {
    return true;
  }
  
  return false;
}

/**
 * Retry a function with exponential backoff.
 * 
 * AUDIT FIX: Implements retry logic for transient failures (Issue #3).
 * 
 * @param {Function} fn - Async function to retry (should throw on failure)
 * @param {Object} options - Retry options
 * @param {number} options.maxRetries - Maximum retry attempts (default: 3)
 * @param {number} options.baseDelayMs - Base delay in ms (default: 1000)
 * @param {number} options.maxDelayMs - Maximum delay in ms (default: 10000)
 * @param {Function} options.shouldRetry - Custom retry predicate (default: isTransientError)
 * @param {Function} options.onRetry - Callback on retry (receives attempt, error, delay)
 * @returns {Promise<any>} - Result of successful function call
 * @throws {Error} - Last error if all retries fail
 */
export async function retryWithBackoff(fn, options = {}) {
  const {
    maxRetries = 3,
    baseDelayMs = 1000,
    maxDelayMs = 10000,
    shouldRetry = isTransientError,
    onRetry = null
  } = options;
  
  let lastError;
  
  for (let attempt = 1; attempt <= maxRetries; attempt++) {
    try {
      return await fn();
    } catch (err) {
      lastError = err;
      
      // Check if we should retry
      if (attempt >= maxRetries || !shouldRetry(err)) {
        throw err;
      }
      
      // Calculate delay with exponential backoff + jitter
      const exponentialDelay = baseDelayMs * Math.pow(2, attempt - 1);
      const jitter = Math.random() * 0.3 * exponentialDelay;
      const delay = Math.min(exponentialDelay + jitter, maxDelayMs);
      
      // Notify callback if provided
      if (onRetry) {
        onRetry(attempt, err, delay);
      }
      
      await sleep(delay);
    }
  }
  
  // This should never be reached, but just in case
  throw lastError || new Error('Retry failed with unknown error');
}

/**
 * Synchronous retry for execFileSync operations.
 * 
 * @param {Function} fn - Sync function to retry
 * @param {Object} options - Same as retryWithBackoff
 * @returns {any} - Result of successful function call
 * @throws {Error} - Last error if all retries fail
 */
export function retrySync(fn, options = {}) {
  const {
    maxRetries = 3,
    baseDelayMs = 500,
    maxDelayMs = 5000,
    shouldRetry = isTransientError,
    onRetry = null
  } = options;
  
  let lastError;
  
  for (let attempt = 1; attempt <= maxRetries; attempt++) {
    try {
      return fn();
    } catch (err) {
      lastError = err;
      
      if (attempt >= maxRetries || !shouldRetry(err)) {
        throw err;
      }
      
      const delay = Math.min(baseDelayMs * Math.pow(2, attempt - 1), maxDelayMs);
      
      if (onRetry) {
        onRetry(attempt, err, delay);
      }
      
      // Sync sleep using Atomics.wait (only works in Node.js)
      const buffer = new SharedArrayBuffer(4);
      const view = new Int32Array(buffer);
      Atomics.wait(view, 0, 0, delay);
    }
  }
  
  throw lastError || new Error('Retry failed with unknown error');
}

/**
 * Safely parse JSON with error handling and optional validation.
 * 
 * AUDIT FIX: Addresses Issue #4 - Missing JSON parse error handling.
 * 
 * @param {string} jsonString - JSON string to parse
 * @param {Object} options - Parsing options
 * @param {string} options.context - Context for error messages (e.g., "Python script output")
 * @param {Function} options.validator - Optional validation function (receives parsed object, returns boolean)
 * @param {any} options.defaultValue - Default value if parsing fails (undefined throws instead)
 * @returns {any} - Parsed JSON object
 * @throws {Error} - If parsing fails and no defaultValue provided
 */
export function safeJsonParse(jsonString, options = {}) {
  const { context = 'JSON', validator = null, defaultValue = undefined } = options;
  
  // Handle empty/null input
  if (jsonString === null || jsonString === undefined || jsonString === '') {
    if (defaultValue !== undefined) return defaultValue;
    throw new Error(`${context}: Empty or null input`);
  }
  
  // Ensure string type
  const str = String(jsonString).trim();
  if (!str) {
    if (defaultValue !== undefined) return defaultValue;
    throw new Error(`${context}: Empty string after trimming`);
  }
  
  // Attempt parse
  let parsed;
  try {
    parsed = JSON.parse(str);
  } catch (parseErr) {
    // Log first 500 chars of invalid JSON for debugging
    const preview = str.length > 500 ? str.substring(0, 500) + '...' : str;
    const err = new Error(`${context}: Invalid JSON - ${parseErr.message}. Preview: ${preview}`);
    err.originalError = parseErr;
    err.rawInput = str;
    
    if (defaultValue !== undefined) {
      console.error(`[safe-json-parse] ${err.message}`);
      return defaultValue;
    }
    throw err;
  }
  
  // Validate structure if validator provided
  if (validator && typeof validator === 'function') {
    try {
      if (!validator(parsed)) {
        const err = new Error(`${context}: Validation failed for parsed JSON`);
        err.parsedValue = parsed;
        
        if (defaultValue !== undefined) {
          console.error(`[safe-json-parse] ${err.message}`);
          return defaultValue;
        }
        throw err;
      }
    } catch (validationErr) {
      if (validationErr.message.includes('Validation failed')) throw validationErr;
      
      const err = new Error(`${context}: Validator threw error - ${validationErr.message}`);
      err.originalError = validationErr;
      err.parsedValue = parsed;
      
      if (defaultValue !== undefined) {
        console.error(`[safe-json-parse] ${err.message}`);
        return defaultValue;
      }
      throw err;
    }
  }
  
  return parsed;
}

/**
 * Validate that an object has the expected structure.
 * 
 * @param {any} obj - Object to validate
 * @param {string[]} requiredFields - Required field names
 * @param {Object} typeChecks - Field type checks { fieldName: 'string' | 'number' | 'array' | 'object' }
 * @returns {boolean} - True if valid
 */
export function validateStructure(obj, requiredFields = [], typeChecks = {}) {
  if (!obj || typeof obj !== 'object') return false;
  
  // Check required fields
  for (const field of requiredFields) {
    if (!(field in obj)) return false;
  }
  
  // Check types
  for (const [field, expectedType] of Object.entries(typeChecks)) {
    const value = obj[field];
    
    switch (expectedType) {
      case 'string':
        if (typeof value !== 'string') return false;
        break;
      case 'number':
        if (typeof value !== 'number' || isNaN(value)) return false;
        break;
      case 'array':
        if (!Array.isArray(value)) return false;
        break;
      case 'object':
        if (typeof value !== 'object' || value === null || Array.isArray(value)) return false;
        break;
      case 'boolean':
        if (typeof value !== 'boolean') return false;
        break;
    }
  }
  
  return true;
}

/**
 * Wrap an error with additional context.
 * Preserves the original stack trace while adding context.
 * 
 * AUDIT FIX: Addresses Issue #1 - Silent failures swallow errors.
 * Use this to propagate errors with context.
 * 
 * @param {Error} err - Original error
 * @param {string} context - Context to add (e.g., "storeMemory")
 * @param {Object} metadata - Additional metadata to attach
 * @returns {Error} - Wrapped error
 */
export function wrapError(err, context, metadata = {}) {
  const wrappedMessage = `[${context}] ${err.message}`;
  const wrapped = new Error(wrappedMessage);
  
  // Preserve original error
  wrapped.cause = err;
  wrapped.originalMessage = err.message;
  wrapped.originalStack = err.stack;
  
  // Copy error properties
  if (err.code) wrapped.code = err.code;
  if (err.status) wrapped.status = err.status;
  if (err.statusCode) wrapped.statusCode = err.statusCode;
  
  // Attach metadata
  for (const [key, value] of Object.entries(metadata)) {
    wrapped[key] = value;
  }
  
  // Combine stack traces
  wrapped.stack = `${wrapped.stack}\n\nCaused by:\n${err.stack}`;
  
  return wrapped;
}

/**
 * Create a circuit breaker for protecting against cascading failures.
 * 
 * @param {Object} options - Circuit breaker options
 * @param {number} options.failureThreshold - Failures before opening circuit (default: 5)
 * @param {number} options.resetTimeoutMs - Time before half-open state (default: 30000)
 * @param {number} options.successThreshold - Successes in half-open to close (default: 2)
 * @returns {Object} - Circuit breaker instance with execute() method
 */
export function createCircuitBreaker(options = {}) {
  const {
    failureThreshold = 5,
    resetTimeoutMs = 30000,
    successThreshold = 2
  } = options;
  
  let state = 'CLOSED';
  let failures = 0;
  let successes = 0;
  let lastFailureTime = 0;
  
  return {
    /**
     * Get current circuit state.
     * @returns {'CLOSED' | 'OPEN' | 'HALF_OPEN'}
     */
    getState() {
      return state;
    },
    
    /**
     * Execute a function through the circuit breaker.
     * @param {Function} fn - Async function to execute
     * @returns {Promise<any>} - Result of function
     * @throws {Error} - If circuit is open or function fails
     */
    async execute(fn) {
      // Check if we should transition from OPEN to HALF_OPEN
      if (state === 'OPEN') {
        if (Date.now() - lastFailureTime > resetTimeoutMs) {
          state = 'HALF_OPEN';
          successes = 0;
        } else {
          throw new Error('Circuit breaker is OPEN - request blocked');
        }
      }
      
      try {
        const result = await fn();
        
        // Success handling
        if (state === 'HALF_OPEN') {
          successes++;
          if (successes >= successThreshold) {
            state = 'CLOSED';
            failures = 0;
          }
        } else {
          failures = 0;
        }
        
        return result;
      } catch (err) {
        // Failure handling
        failures++;
        lastFailureTime = Date.now();
        
        if (failures >= failureThreshold) {
          state = 'OPEN';
        }
        
        throw err;
      }
    },
    
    /**
     * Reset the circuit breaker to closed state.
     */
    reset() {
      state = 'CLOSED';
      failures = 0;
      successes = 0;
      lastFailureTime = 0;
    }
  };
}

export default {
  sleep,
  isTransientError,
  retryWithBackoff,
  retrySync,
  safeJsonParse,
  validateStructure,
  wrapError,
  createCircuitBreaker
};
