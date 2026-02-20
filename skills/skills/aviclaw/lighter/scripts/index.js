/**
 * Lighter Protocol Skill for OpenClaw
 * ZK rollup orderbook DEX
 * 
 * Environment variables:
 * - PRIVATE_KEY: Wallet private key (with 0x prefix)
 * - LIGHTER_API_KEY: API key for Lighter (optional)
 * - LIGHTER_RPC_URL: Ethereum RPC URL (defaults to public RPC)
 */

import { ethers } from 'ethers';
import pkg from 'lighter-sdk';
const { Lighter, Networks } = pkg;

// Default configuration
const DEFAULT_RPC = 'https://eth.llamarpc.com';

// Load environment
function loadEnv() {
  return {
    privateKey: process.env.PRIVATE_KEY,
    apiKey: process.env.LIGHTER_API_KEY,
    rpcUrl: process.env.LIGHTER_RPC_URL || DEFAULT_RPC
  };
}

// Initialize Lighter client
export async function getLighterClient() {
  const config = loadEnv();
  
  if (!config.privateKey) {
    throw new Error('PRIVATE_KEY environment variable is required');
  }
  
  const wallet = new ethers.Wallet(config.privateKey);
  const provider = new ethers.JsonRpcProvider(config.rpcUrl);
  const signer = wallet.connect(provider);
  
  const client = new Lighter({
    signer,
    network: Networks.MAINNET,
    apiKey: config.apiKey
  });
  
  return { client, signer, wallet: wallet.address };
}

// List all available markets
export async function listMarkets() {
  const { client } = await getLighterClient();
  const markets = await client.markets.getMarkets();
  
  return markets.map(m => ({
    id: m.id,
    name: m.name,
    baseToken: m.baseToken?.symbol || m.baseToken,
    quoteToken: m.quoteToken?.symbol || m.quoteToken,
    tickSize: m.tickSize,
    lotSize: m.lotSize
  }));
}

// Get current price for a market
export async function getPrice(marketId) {
  const { client } = await getLighterClient();
  
  // Get orderbook
  const orderbook = await client.markets.getOrderBook(marketId);
  
  const bestBid = orderbook.bids?.[0]?.price || null;
  const bestAsk = orderbook.asks?.[0]?.price || null;
  const lastPrice = orderbook.lastPrice || bestBid || bestAsk;
  
  return {
    marketId,
    lastPrice,
    bestBid,
    bestAsk,
    spread: bestBid && bestAsk ? (bestAsk - bestBid) / bestBid * 100 : null,
    timestamp: new Date().toISOString()
  };
}

// Place an order
export async function placeOrder({
  marketId,
  side, // 'buy' or 'sell'
  price,
  amount,
  orderType = 'limit' // 'limit' or 'market'
}) {
  const { client, wallet } = await getLighterClient();
  
  const order = await client.orders.createOrder({
    marketId,
    side,
    price: ethers.parseUnits(price.toString(), 8), // Price in 8 decimals
    amount: ethers.parseUnits(amount.toString(), 8), // Amount in 8 decimals
    orderType
  });
  
  // Sign and submit
  const signedOrder = await client.orders.signOrder(order);
  const result = await client.orders.submitOrder(signedOrder);
  
  return {
    orderId: result.orderId,
    status: result.status,
    side,
    price,
    amount,
    marketId,
    txHash: result.txHash,
    wallet
  };
}

// Cancel an order
export async function cancelOrder(orderId, marketId) {
  const { client, wallet } = await getLighterClient();
  
  const result = await client.orders.cancelOrder({
    orderId,
    marketId
  });
  
  return {
    orderId,
    status: result.status,
    txHash: result.txHash,
    wallet
  };
}

// Get user's positions
export async function getPositions() {
  const { client, wallet } = await getLighterClient();
  
  const positions = await client.positions.getPositions({ owner: wallet });
  
  return positions.map(p => ({
    marketId: p.marketId,
    side: p.side,
    size: ethers.formatUnits(p.size, 8),
    entryPrice: ethers.formatUnits(p.entryPrice, 8),
    unrealizedPnl: p.unrealizedPnl ? ethers.formatUnits(p.unrealizedPnl, 8) : null,
    liquidationPrice: p.liquidationPrice ? ethers.formatUnits(p.liquidationPrice, 8) : null
  }));
}

// Get account balance
export async function getBalance() {
  const { client, wallet, signer } = await getLighterClient();
  
  // Get account info
  const account = await client.accounts.getAccount(wallet);
  
  // Get token balances
  const balances = await client.balances.getBalances({ owner: wallet });
  
  return {
    wallet,
    account: {
      totalEquity: account?.totalEquity ? ethers.formatUnits(account.totalEquity, 8) : '0',
      availableMargin: account?.availableMargin ? ethers.formatUnits(account.availableMargin, 8) : '0',
      totalPnl: account?.totalPnl ? ethers.formatUnits(account.totalPnl, 8) : '0'
    },
    balances: balances.map(b => ({
      token: b.token?.symbol || b.token,
      balance: ethers.formatUnits(b.balance, b.tokenDecimals || 8),
      available: b.available ? ethers.formatUnits(b.available, b.tokenDecimals || 8) : null
    }))
  };
}

// Get open orders
export async function getOpenOrders(marketId) {
  const { client, wallet } = await getLighterClient();
  
  const orders = await client.orders.getOpenOrders({
    owner: wallet,
    marketId
  });
  
  return orders.map(o => ({
    orderId: o.id,
    marketId: o.marketId,
    side: o.side,
    price: ethers.formatUnits(o.price, 8),
    amount: ethers.formatUnits(o.filledAmount || o.amount, 8),
    remainingAmount: o.remainingAmount ? ethers.formatUnits(o.remainingAmount, 8) : null,
    status: o.status,
    createdAt: o.createdAt
  }));
}

// Get order history
export async function getOrderHistory(marketId, limit = 50) {
  const { client, wallet } = await getLighterClient();
  
  const orders = await client.orders.getOrderHistory({
    owner: wallet,
    marketId,
    limit
  });
  
  return orders.map(o => ({
    orderId: o.id,
    marketId: o.marketId,
    side: o.side,
    price: ethers.formatUnits(o.price, 8),
    amount: ethers.formatUnits(o.amount, 8),
    filledAmount: o.filledAmount ? ethers.formatUnits(o.filledAmount, 8) : '0',
    status: o.status,
    createdAt: o.createdAt,
    updatedAt: o.updatedAt
  }));
}

// Main handler for skill commands
export default async function handleCommand(command, args) {
  switch (command) {
    case 'list-markets':
    case 'markets':
      return await listMarkets();
    
    case 'price':
      if (!args.marketId) throw new Error('marketId is required');
      return await getPrice(args.marketId);
    
    case 'place-order':
    case 'order':
      if (!args.marketId || !args.side || !args.price || !args.amount) {
        throw new Error('marketId, side, price, and amount are required');
      }
      return await placeOrder(args);
    
    case 'cancel-order':
      if (!args.orderId || !args.marketId) {
        throw new Error('orderId and marketId are required');
      }
      return await cancelOrder(args.orderId, args.marketId);
    
    case 'positions':
      return await getPositions();
    
    case 'balance':
    case 'account':
      return await getBalance();
    
    case 'open-orders':
      return await getOpenOrders(args?.marketId);
    
    case 'order-history':
      return await getOrderHistory(args?.marketId, args?.limit);
    
    default:
      throw new Error(`Unknown command: ${command}`);
  }
}
