const { readState, writeState, discoverFromSessions } = require('./state');

/**
 * OpenClaw tool implementations for session-state-tracker.
 *
 * These functions are called by OpenClaw when the agent uses the tools.
 * Each receives (ctx, args) where ctx provides access to other tools.
 */

module.exports = {
  /**
   * Read the current SESSION_STATE.md
   * @param {object} ctx - tool context (contains tools, agent, etc.)
   * @param {object} args - arguments (none)
   * @returns {Promise<object>} state object including frontmatter fields and optional body
   */
  async session_state_read(ctx, args) {
    const state = await readState();
    if (!state) {
      throw new Error('SESSION_STATE.md does not exist or is empty');
    }
    return state;
  },

  /**
   * Update one or more fields in SESSION_STATE.md
   * @param {object} ctx - tool context
   * @param {object} args - partial state updates, e.g., { "task": "New task", "status": "active" }
   * @returns {Promise<object>} { success: true, fields: [...] }
   */
  async session_state_write(ctx, args) {
    if (!args || typeof args !== 'object' || Object.keys(args).length === 0) {
      throw new Error('session_state_write requires at least one field to update');
    }
    await writeState(args);
    return {
      success: true,
      fields: Object.keys(args),
      updated: new Date().toISOString()
    };
  },

  /**
   * Discover current state from session transcripts using memory_search.
   * This is useful when SESSION_STATE.md is missing or stale.
   * @param {object} ctx - tool context (provides memory_search)
   * @param {object} args - arguments (optional, can override query)
   * @returns {Promise<object>} synthesized state object
   */
  async session_state_discover(ctx, args) {
    // The memory_search tool is available via ctx.tools
    const memorySearch = ctx.tools?.memory_search;
    if (!memorySearch) {
      throw new Error('memory_search tool not available. Ensure session transcript indexing is enabled (agents.defaults.memorySearch.experimental.sessionMemory = true)');
    }

    // Optional custom query via args
    const query = args?.query || 'project|task|working on|next step|implementing|building';
    const limit = args?.limit || 10;
    const minScore = args?.minScore || 0.3;

    // Call memory_search
    const results = await memorySearch({
      query,
      sources: ['sessions'],
      limit,
      minScore
    });

    // Synthesize state using the helper from state.js
    const state = await discoverFromSessions(() => Promise.resolve(results));
    // Write discovered state automatically
    await writeState(state);
    return {
      ...state,
      _meta: {
        snippets: results.length,
        topSource: results[0]?.file || 'none'
      }
    };
  }
};
