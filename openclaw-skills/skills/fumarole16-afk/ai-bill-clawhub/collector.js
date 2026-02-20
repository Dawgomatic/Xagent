import fs from 'fs';

// 경로 설정 - 사용자 환경에 맞게 수정 필요
const USAGE_PATH = './dist/usage.json';
const PRICES_PATH = './prices.json';
const SESSION_PATH = process.env.OPENCLAW_SESSIONS || '/root/.openclaw/agents/main/sessions/sessions.json';
const VAULT_PATH = './vault.json';

async function calculateUsage() {
    try {
        if (!fs.existsSync(SESSION_PATH)) return;

        // 파일 읽기 실패 방지 로직
        let sessionsRaw;
        try {
            sessionsRaw = fs.readFileSync(SESSION_PATH, 'utf8');
            if (!sessionsRaw || sessionsRaw.trim() === '') throw new Error("Empty file");
        } catch (e) {
            console.log(`[${new Date().toLocaleTimeString()}] Session file busy, retrying...`);
            return;
        }

        const sessions = JSON.parse(sessionsRaw);
        const prices = JSON.parse(fs.readFileSync(PRICES_PATH, 'utf8'));
        
        let vault = { openai: 0, claude: 0, grok: 0, kimi: 0, deepseek: 0, gemini: 0 };
        if (fs.existsSync(VAULT_PATH)) {
            vault = JSON.parse(fs.readFileSync(VAULT_PATH, 'utf8'));
        }
        
        const usageData = { 
            timestamp: new Date().toISOString(), 
            models: { openai: 0, claude: 0, gemini: 0, kimi: 0, deepseek: 0, grok: 0 } 
        };
        
        const stats = { openai: {}, claude: {}, gemini: {}, kimi: {}, deepseek: {}, grok: {} };

        Object.values(sessions).forEach(s => {
            // 다양한 위치에서 모델 정보 찾기
            let modelFull = '';
            if (s.model) modelFull = s.model.toLowerCase();
            else if (s.modelOverride) modelFull = s.modelOverride.toLowerCase();
            else if (s.providerOverride) modelFull = s.providerOverride.toLowerCase();
            
            const inTokens = s.inputTokens || 0;
            const outTokens = s.outputTokens || 0;
            
            let brand = '';
            if (modelFull.includes('claude')) brand = 'claude';
            else if (modelFull.includes('gemini')) brand = 'gemini';
            else if (modelFull.includes('gpt')) brand = 'openai';
            else if (modelFull.includes('kimi')) brand = 'kimi';
            else if (modelFull.includes('deepseek')) brand = 'deepseek';
            else if (modelFull.includes('grok')) brand = 'grok';

            if (brand && prices[brand]) {
                let priceInfo = null;
                const modelKey = Object.keys(prices[brand]).find(k => modelFull.includes(k));
                priceInfo = modelKey ? prices[brand][modelKey] : prices[brand][Object.keys(prices[brand])[0]];

                if (priceInfo) {
                    const cost = (inTokens * (priceInfo.in / 1000000)) + (outTokens * (priceInfo.out / 1000000));
                    usageData.models[brand] += cost;

                    const verLabel = modelFull.split('/').pop().replace(/-/g, ' ').toUpperCase();
                    stats[brand][verLabel] = (stats[brand][verLabel] || 0) + (inTokens + outTokens);
                }
            }
        });

        Object.keys(usageData.models).forEach(k => {
            const cost = usageData.models[k];
            usageData.models[k] = cost.toFixed(4); 
            usageData.models[k + '_bal'] = (k === 'gemini') ? "POST" : ((vault[k] || 0) - cost).toFixed(2);
            usageData.models[k + '_stats'] = stats[k];
        });

        const jsonStr = JSON.stringify(usageData, null, 2);
        fs.writeFileSync(USAGE_PATH, jsonStr);
        
        console.log(`[${new Date().toLocaleTimeString()}] Sync OK`);
    } catch (e) { 
        console.error(`[${new Date().toLocaleTimeString()}] Error:`, e.message); 
    }
}

calculateUsage();
setInterval(calculateUsage, 30000);
