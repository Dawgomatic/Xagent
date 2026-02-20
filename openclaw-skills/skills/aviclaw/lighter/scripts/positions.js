#!/usr/bin/env node
// Lighter Positions Script

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
    accountIndex: parseInt(process.env.LIGHTER_ACCOUNT_INDEX) || 0,
  });

  try {
    await client.initialize();
    
    const positions = await client.getPositions();
    console.log('\n📈 Positions:');
    console.log('============');
    
    if (positions.length === 0) {
      console.log('No open positions');
    } else {
      positions.forEach((pos, i) => {
        console.log(`${i + 1}. ${pos.market}`);
        console.log(`   Size: ${pos.size}`);
        console.log(`   Entry Price: ${pos.entryPrice}`);
        console.log(`   PnL: ${pos.pnl}`);
        console.log();
      });
    }
    
  } catch (error) {
    console.error('❌ Error:', error.message);
    process.exit(1);
  }
}

main();
