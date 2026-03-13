/**
 * Test script for shrimp coin system
 */

import WebSocket from 'ws';

const SERVER_URL = 'ws://localhost:8080';

async function testCoins() {
    console.log(' Testing Shrimp Coin System...\n');

    const ws = new WebSocket(SERVER_URL);

    return new Promise((resolve) => {
        ws.on('open', () => {
            console.log(' Connected to server');

            // Identify as agent
            ws.send(JSON.stringify({
                type: 'identify',
                role: 'agent',
                agentName: 'CoinTester'
            }));
        });

        ws.on('message', (data) => {
            const msg = JSON.parse(data.toString());

            if (msg.type === 'auth_success') {
                console.log(' Authenticated as CoinTester\n');

                // Test sequence
                setTimeout(() => testGetBalance(ws), 500);
                setTimeout(() => testVisitIsland(ws), 1000);
                setTimeout(() => testVisitIsland(ws), 1500);  // Second visit same island
                setTimeout(() => testVisitAnotherIsland(ws), 2000);
                setTimeout(() => testLikeIsland(ws), 2500);
                setTimeout(() => testLikeAgain(ws), 3000);  // Try to like again
                setTimeout(() => testGetBalance(ws), 3500);
                setTimeout(() => testBuyLand(ws), 4000);
            }

            // Log all coin-related messages
            if (msg.type === 'balance') {
                console.log(` Balance: ${msg.balance} `);
                console.log(`   Total earned: ${msg.totalEarned}`);
                console.log(`   Total spent: ${msg.totalSpent}\n`);
            }

            if (msg.type === 'coin_reward') {
                console.log(` Coin reward: +${msg.amount}  (${msg.reason})`);
                console.log(`   New balance: ${msg.balance}\n`);
            }

            if (msg.type === 'like_result') {
                if (msg.success) {
                    console.log(` Like successful! Reward: +${msg.reward} `);
                    console.log(`   Island likes: ${msg.likes}`);
                    console.log(`   New balance: ${msg.balance}\n`);
                } else {
                    console.log(` Like failed: ${msg.error}\n`);
                }
            }

            if (msg.type === 'buy_result') {
                if (msg.success) {
                    console.log(` Purchase successful!`);
                    console.log(`   Island: ${msg.island.name}`);
                    console.log(`   Price: ${msg.price} `);
                    console.log(`   Remaining: ${msg.balance} \n`);
                } else {
                    console.log(` Purchase failed: ${msg.error}\n`);
                }
            }
        });

        // Close after tests
        setTimeout(() => {
            ws.close();
            console.log(' Test complete!');
            resolve();
        }, 5000);
    });
}

function testGetBalance(ws) {
    console.log(' Requesting balance...');
    ws.send(JSON.stringify({ type: 'get_balance' }));
}

function testVisitIsland(ws) {
    console.log(' Visiting island_nfc08t4z9z...');
    ws.send(JSON.stringify({
        type: 'island_visit',
        islandId: 'island_nfc08t4z9z'
    }));
}

function testVisitAnotherIsland(ws) {
    console.log(' Visiting spawn_island...');
    ws.send(JSON.stringify({
        type: 'island_visit',
        islandId: 'spawn_island'
    }));
}

function testLikeIsland(ws) {
    console.log(' Liking island_nfc08t4z9z...');
    ws.send(JSON.stringify({
        type: 'island_like',
        islandId: 'island_nfc08t4z9z'
    }));
}

function testLikeAgain(ws) {
    console.log(' Trying to like again (should fail - daily limit)...');
    ws.send(JSON.stringify({
        type: 'island_like',
        islandId: 'spawn_island'
    }));
}

function testBuyLand(ws) {
    console.log(' Trying to buy auction land (should fail - not enough coins)...');
    ws.send(JSON.stringify({
        type: 'buy_auction_land',
        islandId: 'island_nfc08t4z9z'
    }));
}

testCoins().catch(console.error);
