#!/usr/bin/env node
// Lighter Account Info Script

const { LighterClient } = require('lighter-sdk');

const PRIVATE_KEY = process.env.PRIVATE_KEY;
const API_KEY = process.env.LIGHTER_API_KEY;

async function main() {
  if (!PRIVATE_KEY) {
    console.error('Error: PRIVATE_KEY environment variable required');
    process.exit(1);
  }

  const client = new LighterClient({
    url: 'https://mainnet.zklighter.elliot.ai',
    privateKey: PRIVATE_KEY,
    chainId: 300,
    apiKeyIndex: API_KEY || 0,
    accountIndex: parseInt(process.env.LIGHTER_ACCOUNT_INDEX) || 0,
  });

  try {
    await client.initialize();
    console.log('✅ Lighter client initialized');
    
    // Get account info
    const account = await client.getAccount();
    console.log('\nAccount Info:');
    console.log(JSON.stringify(account, null, 2));
    
  } catch (error) {
    console.error('❌ Error:', error.message);
    process.exit(1);
  }
}

main();
