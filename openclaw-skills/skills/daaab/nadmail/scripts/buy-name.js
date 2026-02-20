#!/usr/bin/env node
/**
 * NadMail Buy .nad Name Script (Direct Buy)
 *
 * Checks price, gets calldata from API, sends TX directly from user's wallet.
 * The user owns the NFT — no proxy involved.
 *
 * Usage:
 *   node buy-name.js <handle> [--wallet /path/to/key] [--yes]
 *
 * Environment:
 *   NADMAIL_PRIVATE_KEY — wallet private key (recommended)
 *
 * Flow:
 *   1. GET /api/register/nad-name-price/{handle}  → check availability + price
 *   2. GET /api/register/nad-name-sign/{handle}?buyer=0x...  → get calldata
 *   3. Send TX to NNS contract with calldata + value
 *   4. POST /api/auth/agent-register → bind email + auto-create meme coin
 */

const { ethers } = require('ethers');
const fs = require('fs');
const path = require('path');
const readline = require('readline');

const API_BASE = 'https://api.nadmail.ai';
const CONFIG_DIR = path.join(process.env.HOME, '.nadmail');
const AUDIT_FILE = path.join(CONFIG_DIR, 'audit.log');
const TOKEN_FILE = path.join(CONFIG_DIR, 'token.json');

const MONAD_RPC = 'https://rpc.monad.xyz';
const MONAD_CHAIN_ID = 143;

// Max wallet file size (1KB)
const MAX_WALLET_FILE_SIZE = 1024;

function getArg(name) {
  const args = process.argv.slice(2);
  const idx = args.indexOf(name);
  if (idx !== -1 && args[idx + 1]) return args[idx + 1];
  return null;
}

function hasFlag(name) {
  return process.argv.slice(2).includes(name);
}

function prompt(question) {
  const rl = readline.createInterface({ input: process.stdin, output: process.stdout });
  return new Promise(resolve => {
    rl.question(question, answer => { rl.close(); resolve(answer.trim()); });
  });
}

function logAudit(action, details = {}) {
  try {
    if (!fs.existsSync(CONFIG_DIR)) fs.mkdirSync(CONFIG_DIR, { recursive: true, mode: 0o700 });
    const entry = {
      timestamp: new Date().toISOString(),
      action,
      wallet: details.wallet ? `${details.wallet.slice(0, 6)}...${details.wallet.slice(-4)}` : null,
      success: details.success ?? true,
      error: details.error,
    };
    fs.appendFileSync(AUDIT_FILE, JSON.stringify(entry) + '\n', { mode: 0o600 });
  } catch (e) { /* ignore */ }
}

function validateWalletPath(walletPath) {
  const resolved = path.resolve(walletPath);
  const home = process.env.HOME;
  if (!resolved.startsWith(home)) {
    console.error(`Security: Wallet path must be under your home directory (${home})`);
    process.exit(1);
  }
  if (walletPath.includes('..') || walletPath.includes('\0')) {
    console.error('Security: Invalid path');
    process.exit(1);
  }
  try {
    const stat = fs.statSync(resolved);
    if (stat.size > MAX_WALLET_FILE_SIZE || !stat.isFile()) {
      console.error('Security: Invalid wallet file');
      process.exit(1);
    }
  } catch (e) { /* let caller handle */ }
  return resolved;
}

async function getPrivateKey() {
  if (process.env.NADMAIL_PRIVATE_KEY) {
    const key = process.env.NADMAIL_PRIVATE_KEY.trim();
    if (!/^0x[a-fA-F0-9]{64}$/.test(key)) {
      console.error('Error: NADMAIL_PRIVATE_KEY must be 0x + 64 hex chars');
      process.exit(1);
    }
    return key;
  }

  const walletArg = getArg('--wallet');
  if (walletArg) {
    const walletPath = validateWalletPath(walletArg.replace(/^~/, process.env.HOME));
    if (fs.existsSync(walletPath)) {
      const key = fs.readFileSync(walletPath, 'utf8').trim();
      if (!/^0x[a-fA-F0-9]{64}$/.test(key)) {
        console.error('Error: Invalid private key format');
        process.exit(1);
      }
      return key;
    }
    console.error(`Wallet file not found: ${walletPath}`);
    process.exit(1);
  }

  console.error('No wallet found. Set NADMAIL_PRIVATE_KEY or use --wallet /path/to/key');
  process.exit(1);
}

async function api(endpoint) {
  const res = await fetch(`${API_BASE}${endpoint}`);
  return res.json();
}

async function apiPost(endpoint, body, headers = {}) {
  const res = await fetch(`${API_BASE}${endpoint}`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json', ...headers },
    body: JSON.stringify(body),
  });
  return res.json();
}

async function main() {
  const handle = process.argv[2];
  const autoConfirm = hasFlag('--yes');

  if (!handle || handle.startsWith('--')) {
    console.log('NadMail — Buy .nad Name (Direct Buy)\n');
    console.log('Usage: node buy-name.js <handle> [--wallet /path/to/key] [--yes]\n');
    console.log('Steps:');
    console.log('  1. Check price + availability');
    console.log('  2. Get calldata from API');
    console.log('  3. Send TX from your wallet to NNS contract');
    console.log('  4. Register email + auto-create meme coin');
    process.exit(0);
  }

  console.log('NadMail — Buy .nad Name (Direct Buy)');
  console.log('========================================\n');

  const privateKey = await getPrivateKey();
  const provider = new ethers.JsonRpcProvider(MONAD_RPC, MONAD_CHAIN_ID);
  const wallet = new ethers.Wallet(privateKey, provider);
  const address = wallet.address;

  console.log(`Wallet: ${address}`);
  console.log(`Handle: ${handle}\n`);

  // Step 1: Check price
  console.log('1. Checking price...');
  const priceData = await api(`/api/register/nad-name-price/${handle}`);

  if (priceData.error) {
    console.error(`   Error: ${priceData.error}`);
    process.exit(1);
  }

  const available = priceData.nns_available !== false;
  console.log(`   NNS available: ${available ? 'Yes' : 'No'}`);
  console.log(`   Base price: ${priceData.basePrice || priceData.base_price} MON`);
  if (priceData.discount) {
    console.log(`   Discount: ${priceData.discount.name} (${priceData.discount.percent}% off)`);
    console.log(`   Final price: ${priceData.discountedPrice || priceData.discounted_price} MON`);
  }

  if (!available) {
    if (priceData.nns_owner) {
      console.log(`\n   .nad name owned by: ${priceData.nns_owner}`);
      console.log('   If you own this NFT, you can still register the email.');
    } else {
      console.error('\n   Name not available.');
      process.exit(1);
    }
  }

  // Confirmation
  if (!autoConfirm) {
    const answer = await prompt('\nProceed with purchase? (yes/no): ');
    if (answer.toLowerCase() !== 'yes' && answer.toLowerCase() !== 'y') {
      console.log('Cancelled.');
      process.exit(0);
    }
  }

  // Step 2: Get calldata
  console.log('\n2. Getting calldata...');
  const signData = await api(`/api/register/nad-name-sign/${handle}?buyer=${address}`);

  if (signData.error) {
    console.error(`   Error: ${signData.error}`);
    process.exit(1);
  }

  console.log(`   Contract: ${signData.registrar}`);
  console.log(`   Value: ${ethers.formatEther(signData.value)} MON`);

  // Step 3: Send TX
  console.log('\n3. Sending transaction...');
  try {
    const tx = await wallet.sendTransaction({
      to: signData.registrar,
      data: signData.calldata,
      value: BigInt(signData.value),
      chainId: MONAD_CHAIN_ID,
    });

    console.log(`   TX hash: ${tx.hash}`);
    console.log('   Waiting for confirmation...');

    const receipt = await tx.wait();
    console.log(`   Confirmed in block ${receipt.blockNumber}`);
    logAudit('buy_name', { wallet: address, success: true });
  } catch (err) {
    console.error(`   TX failed: ${err.message}`);
    logAudit('buy_name', { wallet: address, success: false, error: err.message });
    process.exit(1);
  }

  // Step 4: Register email
  console.log('\n4. Registering email...');

  const startData = await apiPost('/api/auth/start', { address });
  if (!startData.message) {
    console.error('   Auth start failed:', startData);
    process.exit(1);
  }

  const signature = await wallet.signMessage(startData.message);

  const regData = await apiPost('/api/auth/agent-register', {
    address,
    message: startData.message,
    signature,
    handle,
  });

  if (!regData.token) {
    console.error('   Registration failed:', regData);
    process.exit(1);
  }

  // Save token
  if (!fs.existsSync(CONFIG_DIR)) fs.mkdirSync(CONFIG_DIR, { recursive: true, mode: 0o700 });
  const tokenData = {
    token: regData.token,
    email: regData.email || `${handle}@nadmail.ai`,
    handle: regData.handle || handle,
    wallet: address.toLowerCase(),
    saved_at: new Date().toISOString(),
    expires_hint: '24h',
  };
  fs.writeFileSync(TOKEN_FILE, JSON.stringify(tokenData, null, 2), { mode: 0o600 });

  console.log('\n' + '='.repeat(40));
  console.log('Success!');
  console.log('='.repeat(40));
  console.log(`\nEmail: ${regData.email || handle + '@nadmail.ai'}`);
  if (regData.token_address) {
    console.log(`Meme coin: $${regData.token_symbol} (${regData.token_address})`);
  }
  console.log(`Token saved to: ${TOKEN_FILE}`);
  console.log('\nNext: node scripts/send.js someone@nadmail.ai "Hi" "Hello!"');

  logAudit('register_via_buy', { wallet: address, success: true });
}

main().catch(err => {
  console.error('Error:', err.message);
  process.exit(1);
});
