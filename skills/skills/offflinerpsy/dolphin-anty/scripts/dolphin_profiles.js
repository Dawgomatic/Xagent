/**
 * Dolphin Anty Profile Manager
 * Управление профилями через локальный API (localhost:3001)
 * 
 * Использование:
 *   node dolphin_profiles.js list                                  — список профилей
 *   node dolphin_profiles.js stop --profile-id <ID>                — остановить профиль
 *   node dolphin_profiles.js create --name "Name" [--proxy ...]    — создать профиль
 *   node dolphin_profiles.js delete --profile-id <ID>              — удалить профиль
 *   node dolphin_profiles.js status                                — проверить что Dolphin запущен
 */

const http = require('http');
const https = require('https');
const fs = require('fs');
const path = require('path');
const { argv } = process;

const LOCAL_API = 'http://localhost:3001';
const CLOUD_API = 'https://dolphin-anty-api.com';
const TOKEN_FILE = path.join(__dirname, '..', '.token');

// ─── Token Management ──────────────────────────────────────────

function getToken() {
  if (fs.existsSync(TOKEN_FILE)) {
    return fs.readFileSync(TOKEN_FILE, 'utf8').trim();
  }
  return null;
}

// ─── HTTP Helper ─────────────────────────────────────────────────

function apiRequest(method, apiPath, body = null, useCloud = false) {
  return new Promise((resolve, reject) => {
    const base = useCloud ? CLOUD_API : LOCAL_API;
    const fullPath = useCloud ? apiPath : '/v1.0' + apiPath;
    const url = new URL(base + fullPath);
    const isHttps = url.protocol === 'https:';
    const token = getToken();
    
    const headers = { 'Content-Type': 'application/json' };
    if (token) headers['Authorization'] = 'Bearer ' + token;
    
    const options = {
      hostname: url.hostname,
      port: url.port || (isHttps ? 443 : 80),
      path: url.pathname + url.search,
      method: method,
      headers: headers,
      timeout: 15000,
    };

    const transport = isHttps ? https : http;
    const req = transport.request(options, (res) => {
      let data = '';
      res.on('data', (chunk) => (data += chunk));
      res.on('end', () => {
        try {
          const parsed = JSON.parse(data);
          if (parsed.success === false && parsed.error === 'invalid session token') {
            reject(new Error(
              'Токен не настроен! Выполните:\n' +
              '  node dolphin_setup.js --token <YOUR_TOKEN>\n\n' +
              'Токен получите на https://dolphin-anty.com/panel → API'
            ));
            return;
          }
          resolve({ status: res.statusCode, data: parsed });
        } catch {
          resolve({ status: res.statusCode, data: data });
        }
      });
    });

    req.on('error', (err) => {
      if (err.code === 'ECONNREFUSED') {
        reject(new Error(
          'Dolphin Anty не запущен! Откройте приложение Dolphin Anty и попробуйте снова.'
        ));
      } else {
        reject(err);
      }
    });

    req.on('timeout', () => {
      req.destroy();
      reject(new Error('Таймаут запроса к Dolphin Anty API'));
    });

    if (body) {
      req.write(JSON.stringify(body));
    }
    req.end();
  });
}

// ─── Commands ─────────────────────────────────────────────────

async function listProfiles() {
  try {
    // Try cloud API which returns profiles list
    const res = await apiRequest('GET', '/browser_profiles?limit=50', null, true);
    const profiles = res.data.data || [];
    
    if (profiles.length === 0) {
      console.log('Профилей не найдено.');
      return;
    }

    console.log(`\n Найдено профилей: ${profiles.length}\n`);
    console.log('ID'.padEnd(12) + 'Имя'.padEnd(30) + 'Статус'.padEnd(12) + 'Прокси');
    console.log('─'.repeat(80));
    
    for (const p of profiles) {
      const id = String(p.id).padEnd(12);
      const name = (p.name || 'Без имени').substring(0, 28).padEnd(30);
      const status = (p.status?.toLowerCase() === 'running' ? ' Running' : ' Stopped').padEnd(12);
      const proxy = p.proxy ? `${p.proxy.type}://${p.proxy.host}:${p.proxy.port}` : 'нет';
      console.log(`${id}${name}${status}${proxy}`);
    }
    console.log();
  } catch (err) {
    console.error('', err.message);
    process.exit(1);
  }
}

async function stopProfile(profileId) {
  try {
    console.log(`Останавливаю профиль ${profileId}...`);
    const res = await apiRequest('GET', `/browser_profiles/${profileId}/stop`);
    
    if (res.status === 200) {
      console.log(` Профиль ${profileId} остановлен.`);
    } else {
      console.log(` Ответ: ${JSON.stringify(res.data)}`);
    }
  } catch (err) {
    console.error('', err.message);
    process.exit(1);
  }
}

async function createProfile(name, proxy) {
  const body = {
    name: name,
    platform: 'windows',
    browserType: 'anty',
    mainWebsite: '',
    useragent: { mode: 'manual' },
    webrtc: { mode: 'altered' },
    canvas: { mode: 'real' },
    webgl: { mode: 'real' },
    webglInfo: { mode: 'manual' },
    timezone: { mode: 'auto' },
    locale: { mode: 'auto' },
    geolocation: { mode: 'auto' },
    cpu: { mode: 'manual', value: 4 },
    memory: { mode: 'manual', value: 8 },
    doNotTrack: false,
  };

  if (proxy) {
    // Формат: type://user:pass@host:port  или  type://host:port
    const proxyMatch = proxy.match(/^(\w+):\/\/(?:(.+):(.+)@)?(.+):(\d+)$/);
    if (proxyMatch) {
      body.proxy = {
        type: proxyMatch[1],
        host: proxyMatch[4],
        port: proxyMatch[5],
        login: proxyMatch[2] || '',
        password: proxyMatch[3] || '',
      };
    }
  }

  try {
    console.log(`Создаю профиль "${name}"...`);
    const res = await apiRequest('POST', '/browser_profiles', body, true);
    
    if (res.data && res.data.browserProfileId) {
      console.log(` Профиль создан! ID: ${res.data.browserProfileId}`);
    } else {
      console.log(`Ответ API:`, JSON.stringify(res.data, null, 2));
    }
  } catch (err) {
    console.error('', err.message);
    process.exit(1);
  }
}

async function deleteProfile(profileId) {
  try {
    console.log(`Удаляю профиль ${profileId}...`);
    const res = await apiRequest('DELETE', `/browser_profiles/${profileId}`, null, true);
    
    if (res.status === 200) {
      console.log(` Профиль ${profileId} удалён.`);
    } else {
      console.log(` Ответ: ${JSON.stringify(res.data)}`);
    }
  } catch (err) {
    console.error('', err.message);
    process.exit(1);
  }
}

async function checkStatus() {
  try {
    const token = getToken();
    if (!token) {
      console.log(' Токен не настроен. Выполните: node dolphin_setup.js --token <YOUR_TOKEN>');
      console.log('   Токен получите на https://dolphin-anty.com/panel → раздел API');
      return;
    }
    // Auth the local API
    await apiRequest('POST', '/auth/login-with-token', { token: token });
    console.log(' Токен сохранён в локальном API.');
    
    const res = await apiRequest('GET', '/browser_profiles?limit=1', null, true);
    console.log(' Dolphin Anty API доступен.');
    const total = res.data.total || 'неизвестно';
    console.log(`   Всего профилей: ${total}`);
  } catch (err) {
    console.error(' Dolphin Anty НЕ доступен:', err.message);
    process.exit(1);
  }
}

// ─── CLI Parser ─────────────────────────────────────────────────

function getArg(name) {
  const idx = argv.indexOf(name);
  if (idx !== -1 && idx + 1 < argv.length) return argv[idx + 1];
  return null;
}

async function main() {
  const command = argv[2];

  switch (command) {
    case 'list':
      await listProfiles();
      break;
    case 'stop':
      const stopId = getArg('--profile-id');
      if (!stopId) { console.error('Нужен --profile-id <ID>'); process.exit(1); }
      await stopProfile(stopId);
      break;
    case 'create':
      const name = getArg('--name');
      if (!name) { console.error('Нужен --name "Имя"'); process.exit(1); }
      const proxy = getArg('--proxy');
      await createProfile(name, proxy);
      break;
    case 'delete':
      const delId = getArg('--profile-id');
      if (!delId) { console.error('Нужен --profile-id <ID>'); process.exit(1); }
      await deleteProfile(delId);
      break;
    case 'status':
      await checkStatus();
      break;
    default:
      console.log(`
Dolphin Anty Profile Manager

Использование:
  node dolphin_profiles.js list                                    — список профилей
  node dolphin_profiles.js stop --profile-id <ID>                  — остановить профиль  
  node dolphin_profiles.js create --name "Name" [--proxy ...]      — создать профиль
  node dolphin_profiles.js delete --profile-id <ID>                — удалить профиль
  node dolphin_profiles.js status                                  — проверить Dolphin Anty
      `);
  }
}

main();
