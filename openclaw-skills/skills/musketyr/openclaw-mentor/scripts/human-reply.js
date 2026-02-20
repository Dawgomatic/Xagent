#!/usr/bin/env node
/**
 * Human Reply Helper
 * Usage: node human-reply.js <session-id> "Your guidance here"
 * 
 * Writes human guidance to the consult file that the mentor listener polls for.
 * Use this when a mentee question triggers [NEEDS_HUMAN] and you want to provide input.
 */

const fs = require('fs');

const [sessionId, ...messageParts] = process.argv.slice(2);
const message = messageParts.join(' ');

if (!sessionId || !message) {
  console.error('Usage: node human-reply.js <session-id> "Your guidance here"');
  console.error('');
  console.error('Check /tmp/mentor-consult-*.txt for pending consultations.');
  process.exit(1);
}

const consultFile = `/tmp/mentor-consult-${sessionId}.txt`;
fs.writeFileSync(consultFile, message, 'utf-8');
console.log(`✅ Guidance written for session ${sessionId}`);
console.log(`   The mentor listener will pick this up and generate a response.`);
