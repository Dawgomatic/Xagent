#!/usr/bin/env node
// Lighter Order Script - Place orders on Lighter

const { LighterClient } = require('lighter-sdk');

const PRIVATE_KEY = process.env.PRIVATE_KEY;

async function main() {
  const args = process.argv.slice(2);
  
  // Parse arguments
  const marketArg = args.find(a => a.startsWith('--market='));
  const sideArg = args.find(a => a.startsWith('--side='));
  const amountArg = args.find(a => a.startsWith('--amount='));
  const priceArg = args.find(a => a.startsWith('--price='));
  const typeArg = args.find(a => a.startsWith('--type=')) || '--type=limit';
  
  if (!marketArg || !sideArg || !amountArg) {
    console.log('Usage: node order.js --market=ETH-USDC --side=buy|sell --amount=0.1 [--price=3000] [--type=limit|market]');
    process.exit(1);
  }
  
  const market = marketArg.split('=')[1];
  const side = sideArg.split('=')[1];
  const amount = parseFloat(amountArg.split('=')[1]);
  const price = priceArg ? parseFloat(priceArg.split('=')[1]) : 0;
  const type = typeArg.split('=')[1];
  
  if (!PRIVATE_KEY) {
    console.error('Error: PRIVATE_KEY environment variable required');
    process.exit(1);
  }

  console.log(`Placing ${type} ${side} order on ${market}`);
  console.log(`Amount: ${amount}, Price: ${price || 'market'}`);
  
  const client = new LighterClient({
    url: 'https://mainnet.zklighter.elliot.ai',
    privateKey: PRIVATE_KEY,
    chainId: 300,
    accountIndex: parseInt(process.env.LIGHTER_ACCOUNT_INDEX) || 0,
  });

  try {
    await client.initialize();
    
    const marketIndex = await client.getMarketIndex(market);
    if (marketIndex === -1) {
      console.error(`Error: Market ${market} not found`);
      process.exit(1);
    }
    
    // Place order
    const result = await client.createOrder({
      marketIndex: marketIndex,
      side: side === 'buy' ? 0 : 1, // 0 = buy, 1 = sell
      amount: amount,
      price: price,
    });
    
    console.log('\n✅ Order placed!');
    console.log(`Tx Hash: ${result.txHash}`);
    console.log(`Response:`, result.response.data);
    
  } catch (error) {
    console.error('❌ Error:', error.message);
    process.exit(1);
  }
}

main();
