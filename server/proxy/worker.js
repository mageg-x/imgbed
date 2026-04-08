// Base58 字母表（不含 0OIl）
const ALPHABET = "123456789ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz";
const BASE = ALPHABET.length;

// 预建映射表
const CHAR_TO_VALUE = new Map();
for (let i = 0; i < ALPHABET.length; i++) {
    CHAR_TO_VALUE.set(ALPHABET[i], i);
}

/**
 * Base58 解码为字符串
 */
function base58Decode(str) {
    let result = 0n;
    for (const ch of str) {
        const val = CHAR_TO_VALUE.get(ch);
        if (val === undefined) throw new Error(`Invalid char: ${ch}`);
        result = result * BigInt(BASE) + BigInt(val);
    }
    // 转为字节数组
    const bytes = [];
    while (result > 0n) {
        bytes.unshift(Number(result & 0xFFn));
        result >>= 8n;
    }
    // 处理前导 '1'（对应 0 值）
    for (let i = 0; i < str.length && str[i] === '1'; i++) {
        bytes.unshift(0);
    }
    return new TextDecoder().decode(new Uint8Array(bytes));
}

async function handleRequest(request) {
    const url = new URL(request.url);
    const parts = url.pathname.slice(1).split('/');
    if (parts.length < 2) {
        return new Response('Missing base58 prefix or filename', { status: 400 });
    }
    const encoded = parts[0];
    const filePath = parts.slice(1).join('/');
    let baseUrl;
    try {
        baseUrl = base58Decode(encoded);
    } catch (e) {
        return new Response(`Invalid base58: ${e.message}`, { status: 400 });
    }
    // 拼接完整目标地址
    const target = baseUrl.replace(/\/$/, '') + '/' + filePath + (url.search || '');
    const response = await fetch(target, {
        headers: { 'User-Agent': 'Cloudflare-Worker' }
    });
    const newResponse = new Response(response.body, response);
    newResponse.headers.set('Cache-Control', 'public, max-age=31536000');
    newResponse.headers.set('Access-Control-Allow-Origin', '*');
    return newResponse;
}

addEventListener('fetch', event => {
    event.respondWith(handleRequest(event.request));
});