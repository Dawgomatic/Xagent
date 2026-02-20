#!/usr/bin/env node
// Lighter Markets List Script

const { LighterClient } = require('lighter-sdk');

const PRIVATE_KEY = process.env.PRIVATE_KEY;

async function main() {
  if (!PRIVATE_KEY) {
    console.error('Error: PRIVATE_KEY environment variable required');
    process.exit(1);
  }

  const client = new LighterClient({
    url: 'https://mainnet.zklighter.elliot.ai',
    privateKey: PRIVATE_KEY,
    chainId: 300,
    accountIndex: 0,
  });

  try {
    await client.initialize();
    
    const markets = await client.getMarkets();
    console.log('\n📊 Available Markets:');
    console.log('===================');
    
    markets.forEach((market, i) => {
      console.log(`${i + 1}. ${market.symbol} (${market.base}/${market.quote})`);
      console.log(`   Index: ${market.index}`);
      console.log(`   Min Order: ${market.minOrderSize}`);
      console.log(`   Tick Size: ${market.tickSize}`);
      console.log();
    });
    
  } catch (error) {
    console.error('❌ Error:', error.message);
    process.exit(1);
  }
}

main();
