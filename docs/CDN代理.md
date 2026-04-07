# 为图床项目添加 Cloudflare Worker CDN 代理的完整指南

本文档旨在指导其他 AI（或开发者）如何为自己的图床项目增加一个**基于 Cloudflare Worker 的通用 CDN 代理**功能。该功能可以将任意原始图片直链（例如存储于 R2、Telegram、HuggingFace 等）转换为国内可访问、且长期缓存的加速地址。

## 1. 背景与原理

许多免费或自建的图床会返回一个原始存储 URL（例如 `https://pub-xxx.r2.dev/abc.jpg` 或 `https://api.telegram.org/file/bot...`）。这些地址在国内往往无法直接访问（被墙或速度极慢）。通过部署一个 Cloudflare Worker 作为**反向代理**，我们可以：

- 接收一个格式如 `https://your-worker.workers.dev/<base58编码>/<文件名>` 的请求。
- 解码出原始的基础 URL（例如 `https://pub-xxx.r2.dev`）。
- 拼接完整的原始图片地址并拉取图片。
- 返回图片并添加长期缓存头，后续请求直接命中 CDN 缓存。

**核心优势**：
- 零成本（Cloudflare Workers 免费套餐每日 10 万次请求）。
- 国内访问速度极快（通过 Cloudflare 的全球网络，并可配合 EdgeOne 等进一步加速）。
- URL 简洁美观，不暴露原始路径和 token。
- 通用性强，支持任何 HTTP/HTTPS 图片源。

## 2. 整体架构

```
用户请求 → Cloudflare Worker（代理） → 原始图片服务器（R2/Telegram等）
                ↑
         你的图床后端生成这种 URL
```

你的图床上传成功后，不再返回原始直链，而是返回：
```
https://你的worker域名/<base58(原始基础URL)>/<原始文件名>
```

## 3. 部署 Cloudflare Worker

### 3.1 创建 Worker

1. 登录 [Cloudflare Dashboard](https://dash.cloudflare.com/)。
2. 进入 **Workers 和 Pages** → 点击 **创建应用程序** → **创建 Worker**。
3. 将 Worker 命名为 `img-proxy`（或你喜欢的名称）。

### 3.2 写入 Worker 代码

将以下代码完全替换 Worker 中的默认内容：

```javascript
// Base58 字母表（不含 0OIl）
const ALPHABET = "123456789ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz";
const BASE = ALPHABET.length;
const CHAR_TO_VALUE = new Map();
for (let i = 0; i < ALPHABET.length; i++) CHAR_TO_VALUE.set(ALPHABET[i], i);

/**
 * 将 Base58 字符串解码为原始字符串
 */
function base58Decode(str) {
    let result = 0n;
    for (const ch of str) {
        const val = CHAR_TO_VALUE.get(ch);
        if (val === undefined) throw new Error(`Invalid base58 char: ${ch}`);
        result = result * BigInt(BASE) + BigInt(val);
    }
    // 转换为字节数组（大端序）
    const bytes = [];
    while (result > 0n) {
        bytes.unshift(Number(result & 0xFFn));
        result >>= 8n;
    }
    // 处理前导 '1'（对应数值 0）
    for (let i = 0; i < str.length && str[i] === '1'; i++) {
        bytes.unshift(0);
    }
    return new TextDecoder().decode(new Uint8Array(bytes));
}

async function handleRequest(request) {
    const url = new URL(request.url);
    const pathParts = url.pathname.slice(1).split('/'); // 去掉开头的 '/'
    if (pathParts.length < 2) {
        return new Response('Missing base58 prefix or filename', { status: 400 });
    }
    const encodedPrefix = pathParts[0];
    const filePath = pathParts.slice(1).join('/'); // 剩余部分作为文件路径
    if (!encodedPrefix || !filePath) {
        return new Response('Invalid path format', { status: 400 });
    }
    let baseUrl;
    try {
        baseUrl = base58Decode(encodedPrefix);
    } catch (err) {
        return new Response(`Decode error: ${err.message}`, { status: 400 });
    }
    // 拼接完整的目标 URL
    const targetUrl = baseUrl.replace(/\/$/, '') + '/' + filePath + (url.search || '');
    try {
        const proxyRes = await fetch(targetUrl, {
            headers: {
                'User-Agent': 'Cloudflare-Worker-Proxy',
                'Accept': request.headers.get('Accept') || '*/*'
            }
        });
        const response = new Response(proxyRes.body, proxyRes);
        // 设置缓存：CDN 和浏览器缓存 1 年（图片通常不变）
        response.headers.set('Cache-Control', 'public, max-age=31536000, immutable');
        response.headers.set('Access-Control-Allow-Origin', '*');
        return response;
    } catch (err) {
        return new Response(`Proxy error: ${err.message}`, { status: 502 });
    }
}

addEventListener('fetch', event => {
    event.respondWith(handleRequest(event.request));
});
```

### 3.3 部署并获取域名

点击 **保存并部署**。之后你会获得一个默认域名：
```
https://img-proxy.<你的子域>.workers.dev
```
（如果你希望使用自定义域名，可以在 Worker 的“触发器”中添加路由，但非必需。）

## 4. 修改图床后端：生成 CDN URL

你的图床上传成功后，原本会返回一个原始直链（例如 `https://pub-xxx.r2.dev/folder/abc.jpg`）。现在需要将其转换为 CDN 代理地址。

### 4.1 转换规则

1. 提取**基础 URL**：原始直链中**最后一个 `/` 之前**的部分（不包含尾随 `/`）。
2. 提取**文件名**：最后一个 `/` 之后的部分（可能包含路径，例如 `folder/sub/abc.jpg`）。
3. 对基础 URL 进行 **Base58 编码**（注意：整个基础 URL 作为二进制字符串编码，不保留尾随 `/`）。
4. 拼接最终地址：
   ```
   https://你的worker域名/<base58编码结果>/<文件名>
   ```

### 4.2 提供 Base58 编码函数（多语言）

#### Node.js / JavaScript（适用于浏览器或后端）

```javascript
const ALPHABET = "123456789ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz";

function base58Encode(str) {
    const bytes = new TextEncoder().encode(str);
    let num = 0n;
    for (const b of bytes) {
        num = (num << 8n) | BigInt(b);
    }
    if (num === 0n) return "1";
    let result = "";
    while (num > 0n) {
        const mod = Number(num % 58n);
        result = ALPHABET[mod] + result;
        num = num / 58n;
    }
    // 处理前导零字节
    for (let i = 0; i < bytes.length && bytes[i] === 0; i++) {
        result = "1" + result;
    }
    return result;
}

// 使用示例
const rawUrl = "https://pub-553ab347d3f54de69102221443baac51.r2.dev";
const encoded = base58Encode(rawUrl);
console.log(encoded);
```

#### Python

```python
import base58

def base58_encode_str(s: str) -> str:
    # 需要安装 base58 库: pip install base58
    return base58.b58encode(s.encode('utf-8')).decode('utf-8')

# 使用示例
raw_url = "https://pub-553ab347d3f54de69102221443baac51.r2.dev"
encoded = base58_encode_str(raw_url)
print(encoded)
```

#### PHP

```php
function base58_encode($str) {
    $alphabet = '123456789ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz';
    $bytes = array_values(unpack('C*', $str));
    $num = gmp_init(0);
    foreach ($bytes as $byte) {
        $num = gmp_add(gmp_mul($num, 256), $byte);
    }
    $result = '';
    while (gmp_cmp($num, 0) > 0) {
        $mod = gmp_intval(gmp_mod($num, 58));
        $result = $alphabet[$mod] . $result;
        $num = gmp_div($num, 58);
    }
    // 处理前导零
    for ($i = 0; $i < count($bytes) && $bytes[$i] === 0; $i++) {
        $result = '1' . $result;
    }
    return $result;
}

// 使用示例
$raw_url = "https://pub-553ab347d3f54de69102221443baac51.r2.dev";
$encoded = base58_encode($raw_url);
echo $encoded;
```

### 4.3 修改上传返回逻辑

在你的图床后端（以 Node.js 为例）：

```javascript
const CDN_PROXY_BASE = "https://img-proxy.your-subdomain.workers.dev"; // 你的 Worker 域名

function convertToCDNUrl(originalUrl) {
    const lastSlashIndex = originalUrl.lastIndexOf('/');
    const baseUrl = originalUrl.substring(0, lastSlashIndex);
    const filename = originalUrl.substring(lastSlashIndex + 1);
    const encodedBase = base58Encode(baseUrl);
    return `${CDN_PROXY_BASE}/${encodedBase}/${filename}`;
}

// 上传成功后...
const originalUrl = "https://pub-xxx.r2.dev/abc.jpg";
const cdnUrl = convertToCDNUrl(originalUrl);
// 将 cdnUrl 返回给前端或存入数据库
```

**注意**：如果原始 URL 可能包含查询参数（如 `?token=...`），需要额外处理。通常图床直链不带查询参数，若有，建议将其追加到最终 URL 的末尾（Worker 代码已支持 `url.search` 透传）。

### 4.4 添加配置选项

为了灵活性，建议在你的图床后台增加一个配置项：**CDN 代理地址**。用户可填入自己的 Worker 域名。如果不填，则回退到原始直链。

## 5. 测试验证

1. **测试 Worker 是否存活**：直接访问 `https://你的worker域名/`，应返回错误 `Missing base58 prefix or filename`。
2. **用已知图片测试**：
   - 原始图片：`https://www.google.com/images/branding/googlelogo/2x/googlelogo_color_272x92dp.png`
   - 基础 URL：`https://www.google.com/images/branding/googlelogo/2x`
   - 编码该基础 URL（用上面提供的函数），得到 `xxxxx`。
   - 访问 `https://你的worker域名/xxxxx/googlelogo_color_272x92dp.png`，应显示 Google Logo。
3. **测试你自己的存储**：按照第 4 节生成 CDN URL 并访问。

## 6. 可选：使用自定义域名

如果你希望 CDN 地址更美观（例如 `https://img.yourdomain.com/...`），可以：

1. 在 Cloudflare Workers → 你的 Worker → 触发器 → 自定义域，添加 `img.yourdomain.com`。
2. 在你的 DNS 提供商处将 `img.yourdomain.com` 通过 CNAME 指向 `img-proxy.your-subdomain.workers.dev`（Cloudflare 会自动处理）。
3. 然后将图床配置中的 `CDN_PROXY_BASE` 改为 `https://img.yourdomain.com`。

## 7. 注意事项

### 7.1 缓存策略
Worker 代码中设置了 `Cache-Control: public, max-age=31536000, immutable`，表示图片会被 Cloudflare CDN 及浏览器缓存一年。如果你需要更新图片（同名覆盖），建议使用不同的文件名或手动刷新缓存。

### 7.2 费用
Cloudflare Workers 免费套餐包含：
- 每日 10 万次请求
- 每月 1000 万次请求（超出后按 $0.30/百万次计费，个人使用几乎不会超）
- 无带宽限制

### 7.3 隐私与安全
- Worker 仅做转发，不存储任何图片内容。
- 原始基础 URL 经过 base58 编码，并非加密，但可防止简单爬取。
- 如果你担心泄露，可以在 Worker 中添加 IP 白名单或请求签名校验（本指南不展开）。

### 7.4 兼容性
- Worker 代码支持任何 HTTP/HTTPS 源，包括 R2、S3、Telegram、GitHub、HuggingFace 等。
- 如果原始服务器需要特定的 User-Agent 或 Referer，可以在 Worker 的 `fetch` 中自行添加。

## 8. 常见问题

**Q: 为什么不直接用 base64？**  
A: base64 包含 `+`、`/`、`=` 等特殊字符，在 URL 中需要编码，会破坏简洁性。base58 无特殊字符，且长度更短。

**Q: 原始 URL 中包含非 ASCII 字符怎么办？**  
A: 本方案将所有字符按 UTF-8 编码后 base58，完全支持中文等 Unicode 路径，但请确保原始服务器能正确处理 URL 编码后的路径。通常图片文件名都是 ASCII。

**Q: 如果原始图片很大（超过 100MB）？**  
A: Cloudflare Workers 免费版限制响应体大小为 100MB。超出会失败。建议只代理普通图片（小于 100MB）。

**Q: 能否同时代理多个不同的存储源？**  
A: 可以。本方案基于基础 URL 的 base58 编码，每个不同的基础 URL 会得到不同的编码前缀，Worker 自动区分。你只需在上传时对当前图片的原始基础 URL 进行编码即可。

## 9. 总结

通过部署一个 Cloudflare Worker 并修改图床后端生成 CDN URL，你就能以**零成本**将任何原始图片直链转换为国内可快速访问的地址。整套方案已经过实际验证，可稳定运行。

**下一步**：将本文档交给其他 AI 或开发者，他们就能按照步骤完成集成。如果在实现过程中遇到问题，可以检查 Worker 日志（在 Cloudflare 控制台 Workers → 相应 Worker → 日志）进行调试。