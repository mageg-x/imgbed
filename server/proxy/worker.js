/**
 * Cloudflare Worker - 图片代理服务
 * 
 * 功能：
 * 1. 下载代理（CDN加速）：将图片直链转换为代理地址，实现加速访问
 * 2. 上传代理（突破限制）：代理上传请求到 Telegram/Discord/HuggingFace 等被墙服务
 * 
 * 路由规则：
 * - GET /{base58_host}/{filepath}          → 下载代理，用于 CDN 加速
 * - *   /proxy/{base58_host}/{path}        → 上传代理，透传 HTTP 请求
 * 
 * 部署方式：
 * 1. 登录 Cloudflare Dashboard → Workers & Pages
 * 2. 创建新 Worker，粘贴此代码
 * 3. 绑定自定义域名（可选）
 */

// ==================== Base58 编解码 ====================

const ALPHABET = "123456789ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz";
const BASE = ALPHABET.length;

const CHAR_TO_VALUE = new Map();
for (let i = 0; i < ALPHABET.length; i++) {
    CHAR_TO_VALUE.set(ALPHABET[i], i);
}

/**
 * Base58 解码
 * @param {string} str - Base58 编码的字符串
 * @returns {string} - 解码后的原始字符串
 */
function base58Decode(str) {
    let result = 0n;
    for (const ch of str) {
        const val = CHAR_TO_VALUE.get(ch);
        if (val === undefined) throw new Error(`Invalid char: ${ch}`);
        result = result * BigInt(BASE) + BigInt(val);
    }
    
    const bytes = [];
    while (result > 0n) {
        bytes.unshift(Number(result & 0xFFn));
        result >>= 8n;
    }
    
    for (let i = 0; i < str.length && str[i] === '1'; i++) {
        bytes.unshift(0);
    }
    
    return new TextDecoder().decode(new Uint8Array(bytes));
}

/**
 * Base58 编码
 * @param {string} str - 原始字符串
 * @returns {string} - Base58 编码后的字符串
 */
function base58Encode(str) {
    const encoder = new TextEncoder();
    const bytes = encoder.encode(str);
    let num = 0n;
    
    for (const b of bytes) {
        num = num * 256n + BigInt(b);
    }
    
    let result = '';
    while (num > 0n) {
        result = ALPHABET[Number(num % BigInt(BASE))] + result;
        num /= BigInt(BASE);
    }
    
    for (const b of bytes) {
        if (b === 0) result = '1' + result;
        else break;
    }
    
    return result || '1';
}

// ==================== 请求头过滤 ====================

/**
 * 需要过滤的请求头列表
 * 这些头部会暴露客户端真实信息或 Cloudflare 特有信息，不应转发给目标服务器
 */
const HEADERS_TO_REMOVE = [
    'host',                  // 原始主机名
    'cf-connecting-ip',      // 客户端真实 IP
    'cf-ipcountry',          // 客户端国家代码
    'cf-ray',                // Cloudflare Ray ID
    'cf-visitor',            // Cloudflare 访问者信息
    'cf-worker',             // Cloudflare Worker 标识
    'x-forwarded-for',       // 代理转发的 IP 链
    'x-forwarded-proto',     // 原始请求协议
    'x-real-ip',             // 客户端真实 IP
];

// ==================== 下载代理（CDN 加速）====================

/**
 * 处理下载请求 - 用于 CDN 加速
 * 
 * URL 格式: /{base58_host}/{filepath}
 * 示例: /5Qr5Eyk/bot123456:ABC/documents/file_0.png
 * 
 * 工作流程:
 * 1. 从 URL 中提取 base58 编码的目标主机地址
 * 2. 解码得到原始主机（如 https://api.telegram.org）
 * 3. 拼接完整目标 URL 并请求
 * 4. 返回图片内容，设置长期缓存
 * 
 * @param {URL} url - 请求的 URL 对象
 * @returns {Response} - 代理响应
 */
async function handleDownload(url) {
    const parts = url.pathname.slice(1).split('/');
    
    if (parts.length < 2) {
        return new Response('缺少参数：需要 base58 主机编码和文件路径', { status: 400 });
    }
    
    const encodedHost = parts[0];
    const filePath = parts.slice(1).join('/');
    
    let targetHost;
    try {
        targetHost = base58Decode(encodedHost);
    } catch (e) {
        return new Response(`Base58 解码失败: ${e.message}`, { status: 400 });
    }
    
    // 拼接完整目标 URL
    const targetUrl = targetHost.replace(/\/$/, '') + '/' + filePath + (url.search || '');
    
    // 发起请求
    const response = await fetch(targetUrl, {
        headers: { 'User-Agent': 'Cloudflare-Worker-Proxy' }
    });
    
    // 构建响应，设置缓存和 CORS
    const proxyResponse = new Response(response.body, response);
    proxyResponse.headers.set('Cache-Control', 'public, max-age=31536000'); // 缓存 1 年
    proxyResponse.headers.set('Access-Control-Allow-Origin', '*');
    
    return proxyResponse;
}

// ==================== 上传代理（突破限制）====================

/**
 * 处理上传请求 - 用于代理被墙服务的 API 请求
 * 
 * URL 格式: /proxy/{base58_host}/{path}
 * 示例: /proxy/5Qr5Eyk/bot123456:ABC/sendDocument
 * 
 * 工作流程:
 * 1. 从 URL 中提取 base58 编码的目标主机地址
 * 2. 解码得到原始主机（如 https://api.telegram.org）
 * 3. 拼接完整目标 URL
 * 4. 过滤敏感请求头，透传请求方法、请求体
 * 5. 透传响应状态码、响应头、响应体
 * 
 * @param {Request} request - 原始请求对象
 * @param {URL} url - 请求的 URL 对象
 * @returns {Response} - 代理响应
 */
async function handleUpload(request, url) {
    const parts = url.pathname.slice('/proxy/'.length).split('/');
    
    if (parts.length < 2) {
        return new Response('缺少参数：需要 base58 主机编码和请求路径', { status: 400 });
    }
    
    const encodedHost = parts[0];
    const requestPath = parts.slice(1).join('/');
    
    let targetHost;
    try {
        targetHost = base58Decode(encodedHost);
    } catch (e) {
        return new Response(`Base58 解码失败: ${e.message}`, { status: 400 });
    }
    
    // 拼接完整目标 URL
    const targetUrl = targetHost.replace(/\/$/, '') + '/' + requestPath + (url.search || '');
    
    // 构建代理请求头，过滤敏感信息
    const proxyHeaders = new Headers();
    for (const [key, value] of request.headers.entries()) {
        if (!HEADERS_TO_REMOVE.includes(key.toLowerCase())) {
            proxyHeaders.set(key, value);
        }
    }
    proxyHeaders.set('User-Agent', 'Cloudflare-Worker-Proxy');
    
    // 发起代理请求，透传方法和请求体
    const response = await fetch(targetUrl, {
        method: request.method,
        headers: proxyHeaders,
        body: request.body,
    });
    
    // 构建响应，透传原始响应头
    const proxyResponse = new Response(response.body, response);
    for (const [key, value] of response.headers.entries()) {
        proxyResponse.headers.set(key, value);
    }
    
    // 设置 CORS 和清理 Cloudflare 特有头部
    proxyResponse.headers.set('Access-Control-Allow-Origin', '*');
    proxyResponse.headers.delete('cf-cache-status');
    proxyResponse.headers.delete('cf-ray');
    
    return proxyResponse;
}

// ==================== 主请求处理 ====================

/**
 * 处理 CORS 预检请求
 * @returns {Response} - CORS 预检响应
 */
function handleCorsPreflight() {
    return new Response(null, {
        status: 204,
        headers: {
            'Access-Control-Allow-Origin': '*',
            'Access-Control-Allow-Methods': 'GET, POST, PUT, DELETE, PATCH, OPTIONS',
            'Access-Control-Allow-Headers': '*',
            'Access-Control-Max-Age': '86400',
        }
    });
}

/**
 * 主请求处理函数
 * 
 * 路由规则:
 * - OPTIONS *                    → CORS 预检请求
 * - GET /{base58}/{path}         → 下载代理（CDN 加速）
 * - *   /proxy/{base58}/{path}   → 上传代理（API 透传）
 * 
 * @param {Request} request - 请求对象
 * @returns {Response} - 响应对象
 */
async function handleRequest(request) {
    // 处理 CORS 预检请求
    if (request.method === 'OPTIONS') {
        return handleCorsPreflight();
    }
    
    const url = new URL(request.url);
    
    // 上传代理路由：/proxy/{base58_host}/{path}
    if (url.pathname.startsWith('/proxy/')) {
        return handleUpload(request, url);
    }
    
    // 下载代理路由：/{base58_host}/{filepath}
    return handleDownload(url);
}

// ==================== 事件监听 ====================

addEventListener('fetch', event => {
    event.respondWith(handleRequest(event.request));
});
