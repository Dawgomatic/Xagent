#!/usr/bin/env npx tsx
/**
 * Interactive setup for Gekko Yield skill
 * Configures wallet using environment variable
 * 
 * Usage: npx tsx setup.ts
 */

import * as fs from 'fs';
import * as path from 'path';
import * as readline from 'readline';

const CONFIG_DIR = path.join(process.env.HOME || process.env.USERPROFILE || '', '.config', 'gekko-yield');
const CONFIG_PATH = path.join(CONFIG_DIR, 'config.json');

interface Config {
  wallet: {
    source: 'env';
    envVar: string;
  };
  rpc: string;
}

function ask(rl: readline.Interface, question: string): Promise<string> {
  return new Promise((resolve) => {
    rl.question(question, resolve);
  });
}

async function main() {
  const rl = readline.createInterface({
    input: process.stdin,
    output: process.stdout,
  });

  console.log(' Gekko Yield — Setup\n');
  console.log('This will configure your wallet for the Moonwell USDC vault.\n');

  // Check existing config
  if (fs.existsSync(CONFIG_PATH)) {
    console.log(' Existing configuration found.\n');
    const overwrite = await ask(rl, 'Overwrite existing config? [y/N]: ');
    if (overwrite.toLowerCase() !== 'y') {
      console.log('\nSetup cancelled. Existing config preserved.');
      rl.close();
      return;
    }
    console.log('');
  }

  console.log('━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━');
  console.log(' STEP 1: Set your private key as an environment variable');
  console.log('━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n');

  console.log('In PowerShell, run this command (replace with your actual key):\n');
  console.log('  $env:PRIVATE_KEY="0x1234567890abcdef..."\n');
  console.log('  Make sure to include the "0x" prefix!\n');

  const continueSetup = await ask(rl, 'Have you set the PRIVATE_KEY environment variable? [y/N]: ');

  if (continueSetup.toLowerCase() !== 'y') {
    console.log('\n Please set the environment variable first, then run setup again.');
    console.log('\nFor permanent setup (survives terminal restart):');
    console.log('  [System.Environment]::SetEnvironmentVariable("PRIVATE_KEY", "YOUR_KEY", "User")');
    rl.close();
    return;
  }

  // Verify the environment variable
  const testKey = process.env.PRIVATE_KEY;
  if (!testKey) {
    console.log('\n PRIVATE_KEY environment variable not found.');
    console.log('   Make sure you set it in the SAME terminal where you run this script.');
    console.log('   Run: $env:PRIVATE_KEY="your-key" and then run setup again.\n');
    rl.close();
    return;
  }

  if (!testKey.startsWith('0x') || testKey.length !== 66) {
    console.log('\n  Warning: Private key format looks incorrect.');
    console.log('   Should start with "0x" and be 66 characters total.');
    const proceed = await ask(rl, 'Continue anyway? [y/N]: ');
    if (proceed.toLowerCase() !== 'y') {
      rl.close();
      return;
    }
  }

  console.log('\n Environment variable found!\n');

  // RPC configuration
  console.log('━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━');
  console.log(' STEP 2: RPC Configuration');
  console.log('━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n');
  console.log('Default RPC: https://mainnet.base.org');
  console.log('(Script will automatically retry with alternative RPCs if rate limited)\n');
  
  const customRpc = await ask(rl, 'Use custom RPC URL? [N]: ');
  const rpcUrl = customRpc.trim() || 'https://mainnet.base.org';

  // Create config
  const config: Config = {
    wallet: {
      source: 'env',
      envVar: 'PRIVATE_KEY',
    },
    rpc: rpcUrl,
  };

  // Save config
  fs.mkdirSync(CONFIG_DIR, { recursive: true });
  fs.writeFileSync(CONFIG_PATH, JSON.stringify(config, null, 2));
  console.log(` Config saved to ${CONFIG_PATH}\n`);

  console.log('━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━');
  console.log(' Setup Complete!');
  console.log('━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n');

  console.log(' Important: Keep your PRIVATE_KEY environment variable set!');
  console.log('   If you close this terminal, you\'ll need to set it again:\n');
  console.log('   $env:PRIVATE_KEY="your-private-key"\n');
  
  console.log(' Tip: If you encounter rate limit errors (429), the script will');
  console.log('   automatically retry with alternative RPC endpoints.\n');

  console.log(' Next Steps:\n');
  console.log('1. Check your position:');
  console.log('   npx tsx status.ts\n');
  console.log('2. Deposit USDC (make sure you have USDC and ETH for gas on Base):');
  console.log('   npx tsx deposit.ts 100\n');
  console.log('3. Withdraw (when needed):');
  console.log('   npx tsx withdraw.ts 50');
  console.log('   npx tsx withdraw.ts all\n');

  rl.close();
  
  // Explicitly exit to close any open connections
  process.exit(0);
}

main().catch((err) => {
  console.error(' Setup failed:', err instanceof Error ? err.message : String(err));
  process.exit(1);
});
