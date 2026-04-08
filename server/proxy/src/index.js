import { AwsClient } from 'aws4fetch';

// ==================== Base58 ====================

const ALPHABET = "123456789ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz";
const BASE = ALPHABET.length;
const CHAR_TO_VALUE = new Map();
for (let i = 0; i < ALPHABET.length; i++) {
    CHAR_TO_VALUE.set(ALPHABET[i], i);
}

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

// ==================== 下载代理 ====================

async function handleDownload(url) {
    const parts = url.pathname.slice(1).split('/');
    if (parts.length < 2) {
        return new Response('参数错误', { status: 400 });
    }
    const encodedHost = parts[0];
    const filePath = parts.slice(1).join('/');
    let targetHost;
    try {
        targetHost = base58Decode(encodedHost);
    } catch (e) {
        return new Response(`解码失败: ${e.message}`, { status: 400 });
    }
    const targetUrl = targetHost.replace(/\/$/, '') + '/' + filePath + (url.search || '');
    const response = await fetch(targetUrl, {
        headers: { 'User-Agent': 'Cloudflare-Worker-Proxy' }
    });
    const proxyResponse = new Response(response.body, response);
    proxyResponse.headers.set('Cache-Control', 'public, max-age=31536000');
    proxyResponse.headers.set('Access-Control-Allow-Origin', '*');
    return proxyResponse;
}

// ==================== 上传代理 ====================

const HEADERS_TO_REMOVE = ['host', 'cf-connecting-ip', 'cf-ipcountry', 'cf-ray', 'cf-visitor', 'cf-worker', 'x-forwarded-for', 'x-forwarded-proto', 'x-real-ip'];

async function handleUpload(request, url) {
    const parts = url.pathname.slice('/proxy/'.length).split('/');
    if (parts.length < 2) {
        return new Response('参数错误', { status: 400 });
    }
    const encodedHost = parts[0];
    const requestPath = parts.slice(1).join('/');
    let targetHost;
    try {
        targetHost = base58Decode(encodedHost);
    } catch (e) {
        return new Response(`解码失败: ${e.message}`, { status: 400 });
    }
    const targetUrl = targetHost.replace(/\/$/, '') + '/' + requestPath + (url.search || '');
    const proxyHeaders = new Headers();
    for (const [key, value] of request.headers.entries()) {
        if (!HEADERS_TO_REMOVE.includes(key.toLowerCase())) {
            proxyHeaders.set(key, value);
        }
    }
    proxyHeaders.set('User-Agent', 'Cloudflare-Worker-Proxy');
    const response = await fetch(targetUrl, {
        method: request.method,
        headers: proxyHeaders,
        body: request.body,
    });
    const proxyResponse = new Response(response.body, response);
    for (const [key, value] of response.headers.entries()) {
        proxyResponse.headers.set(key, value);
    }
    proxyResponse.headers.set('Access-Control-Allow-Origin', '*');
    proxyResponse.headers.delete('cf-cache-status');
    proxyResponse.headers.delete('cf-ray');
    return proxyResponse;
}

// ==================== S3 代理 ====================

async function handleS3Proxy(request) {
    const accessKey = request.headers.get("X-Aws-Access-Key");
    const secretKey = request.headers.get("X-Aws-Secret-Key");
    const region = request.headers.get("X-Aws-Region") || "auto";
    const targetUrl = request.headers.get("X-Target-Url");
    const contentType = request.headers.get("Content-Type") || "application/octet-stream";

    if (!accessKey || !secretKey || !targetUrl) {
        return new Response("缺少凭证", { status: 400 });
    }

    const target = new URL(targetUrl);
    const pathname = target.pathname;
    const host = target.host;

    let bucket, key;
    if (host.includes(".cos.") || host.includes(".r2.") || host.includes(".s3.")) {
        const parts = host.split(".");
        bucket = parts[0];
        key = pathname.slice(1);
    } else {
        const parts = pathname.slice(1).split("/");
        bucket = parts[0];
        key = parts.slice(1).join("/");
    }

    const bodyData = await request.arrayBuffer();

    const aws = new AwsClient({
        accessKeyId: accessKey,
        secretAccessKey: secretKey,
        region: region,
        service: "s3",
    });

    const signedRequest = await aws.sign(targetUrl, {
        method: "PUT",
        headers: {
            "Content-Type": contentType,
            "Content-Length": bodyData.byteLength,
        },
        body: bodyData,
    });

    const response = await fetch(signedRequest);

    if (response.ok) {
        const xmlResponse = `<?xml version="1.0" encoding="UTF-8"?>
<PutObjectResult xmlns="http://s3.amazonaws.com/doc/2006-03-01/">
  <Key>${key}</Key>
  <ETag>"${Date.now()}"</ETag>
</PutObjectResult>`;
        return new Response(xmlResponse, {
            status: 200,
            headers: {
                "Content-Type": "application/xml",
                "Access-Control-Allow-Origin": "*",
            },
        });
    } else {
        const errorText = await response.text();
        return new Response(errorText, {
            status: response.status,
            headers: {
                "Content-Type": "text/plain",
                "Access-Control-Allow-Origin": "*",
            },
        });
    }
}

// ==================== 路由 ====================

async function handleRequest(request) {
    const url = new URL(request.url);

    if (url.pathname === "/s3-proxy" || url.pathname.startsWith("/s3-proxy/")) {
        return handleS3Proxy(request);
    }

    if (url.pathname.startsWith('/proxy/')) {
        return handleUpload(request, url);
    }

    return handleDownload(url);
}

addEventListener('fetch', event => {
    event.respondWith(handleRequest(event.request));
});
