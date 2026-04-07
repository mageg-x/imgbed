<script setup>
import { ref, onMounted } from 'vue'
import { ElMessage, ElTag } from 'element-plus'
import { Code, Copy, Check, FileText, Terminal, Braces, Image } from 'lucide-vue-next'

const isDark = ref(true)
const copiedSection = ref('')

onMounted(() => {
  isDark.value = !document.documentElement.classList.contains('light')
})

function copyToClipboard(code, section) {
  navigator.clipboard.writeText(code).then(() => {
    copiedSection.value = section
    ElMessage.success('已复制到剪贴板')
    setTimeout(() => {
      copiedSection.value = ''
    }, 2000)
  })
}

function getBaseUrl() {
  return window.location.origin
}

const curlUploadAnonymous = `curl -X POST ${getBaseUrl()}/api/v1/upload/anonymous \\
  -F "file=@${'{filepath}'}"`

const curlUploadToken = `curl -X POST ${getBaseUrl()}/api/v1/upload \\
  -H "X-API-Token: your_token" \\
  -H "X-API-Secret: your_secret" \\
  -F "file=@${'{filepath}'}"`

const pythonCode = `import requests

class ImgBedClient:
    def __init__(self, base_url, api_token=None, api_secret=None):
        self.base_url = base_url
        self.api_token = api_token
        self.api_secret = api_secret

    def upload(self, image_path):
        with open(image_path, 'rb') as f:
            files = {'file': f}
            headers = {}
            url = f"{self.base_url}/api/v1/upload/anonymous"

            if self.api_token and self.api_secret:
                headers = {
                    'X-API-Token': self.api_token,
                    'X-API-Secret': self.api_secret
                }
                url = f"{self.base_url}/api/v1/upload"

            response = requests.post(url, files=files, headers=headers)
            return response.json()

# 使用示例
client = ImgBedClient("${getBaseUrl()}", "your_token", "your_secret")
result = client.upload("image.jpg")
if result['code'] == 0:
    print(f"上传成功: {result['data']['links']['markdown']}")`

const javascriptCode = `<!DOCTYPE html>
<html>
<body>
    <textarea id="editor" placeholder="在这里写文章，Ctrl+V 粘贴图片..."></textarea>
    <script>
        const BASE_URL = '${getBaseUrl()}';
        const API_TOKEN = 'your_token';
        const API_SECRET = 'your_secret';
        async function uploadImage(file) {
            const formData = new FormData();
            formData.append('file', file);
            const headers = {};
            let url = BASE_URL + '/api/v1/upload/anonymous';
            if (API_TOKEN && API_SECRET) {
                headers['X-API-Token'] = API_TOKEN;
                headers['X-API-Secret'] = API_SECRET;
                url = BASE_URL + '/api/v1/upload';
            }
            const response = await fetch(url, { method: 'POST', headers, body: formData });
            return await response.json();
        }
        document.getElementById('editor').addEventListener('paste', async (e) => {
            const items = e.clipboardData.items;
            for (const item of items) {
                if (item.kind === 'file' && item.type.startsWith('image/')) {
                    e.preventDefault();
                    const file = item.getAsFile();
                    const result = await uploadImage(file);
                    if (result.code === 0) {
                        const markdown = result.data.links.markdown;
                        const textarea = e.target;
                        textarea.value += markdown;
                    }
                    break;
                }
            }
        });
    <\/script>
</body>
</html>`

const typoraAnonymous = `# Typora 图片上传服务配置
# 设置 -> 图像 -> 上传服务 -> Custom Command
curl -X POST ${getBaseUrl()}/api/v1/upload/anonymous \\
  -F "file=@${'{filepath}'}" \\
  | grep -o '"url":"[^"]*"' | cut -d'"' -f4`

const typoraToken = `# Typora 图片上传服务配置 (推荐)
# 设置 -> 图像 -> 上传服务 -> Custom Command
curl -X POST ${getBaseUrl()}/api/v1/upload \\
  -H "X-API-Token: your_token" \\
  -H "X-API-Secret: your_secret" \\
  -F "file=@${'{filepath}'}" \\
  | grep -o '"url":"[^"]*"' | cut -d'"' -f4`

const nodejsCode = `const axios = require('axios');
const fs = require('fs');
const path = require('path');

class ImgBedClient {
    constructor(baseUrl, token, secret) {
        this.baseUrl = baseUrl;
        this.token = token;
        this.secret = secret;
    }

    async upload(imagePath) {
        const fileStream = fs.createReadStream(imagePath);
        const formData = new FormData();
        formData.append('file', fileStream, path.basename(imagePath));

        const headers = {
            'X-API-Token': this.token,
            'X-API-Secret': this.secret
        };

        const response = await axios.post(\`\${this.baseUrl}/api/v1/upload\`, formData, { headers });
        return response.data;
    }
}

// 使用示例
const client = new ImgBedClient('${getBaseUrl()}', 'your_token', 'your_secret');
const result = await client.upload('./image.jpg');
console.log('上传成功:', result.data.links.markdown);`

const sections = [
  {
    id: 'curl',
    title: 'CURL 命令行',
    icon: Terminal,
    items: [
      { label: '匿名上传', code: curlUploadAnonymous },
      { label: 'Token 上传', code: curlUploadToken }
    ]
  },
  {
    id: 'typora',
    title: 'Typora 集成',
    icon: Image,
    items: [
      { label: '匿名上传', code: typoraAnonymous },
      { label: 'Token 上传 (推荐)', code: typoraToken }
    ]
  },
  {
    id: 'python',
    title: 'Python 脚本',
    icon: Code,
    items: [
      { label: '完整示例', code: pythonCode }
    ]
  },
  {
    id: 'javascript',
    title: 'JavaScript/HTML',
    icon: Braces,
    items: [
      { label: '粘贴上传示例', code: javascriptCode }
    ]
  },
  {
    id: 'nodejs',
    title: 'Node.js',
    icon: Terminal,
    items: [
      { label: '完整示例', code: nodejsCode }
    ]
  }
]
</script>

<template>
  <div class="space-y-4 sm:space-y-6">
    <!-- Token 提示 -->
    <div class="p-4 rounded-xl border"
      :class="isDark ? 'bg-indigo-500/10 border-indigo-500/30' : 'bg-indigo-50 border-indigo-200'">
      <div class="flex items-start gap-3">
        <FileText class="w-5 h-5 text-indigo-500 mt-0.5 flex-shrink-0" />
        <div>
          <p class="font-medium text-sm">获取 API Token</p>
          <p class="text-xs mt-1" :class="isDark ? 'text-gray-400' : 'text-gray-600'">
            在
            <router-link to="/tokens" class="text-indigo-500 hover:underline">API Token 管理</router-link>
            页面创建 Token，然后替换示例中的 your_token 和 your_secret
          </p>
        </div>
      </div>
    </div>

    <!-- 集成示例列表 -->
    <div class="space-y-6">
      <div v-for="section in sections" :key="section.id" class="card">
        <div class="flex items-center gap-2 mb-4">
          <component :is="section.icon" class="w-5 h-5 text-indigo-500" />
          <h3 class="text-lg font-semibold">{{ section.title }}</h3>
        </div>

        <div class="space-y-4">
          <div v-for="(item, idx) in section.items" :key="idx">
            <div class="flex items-center justify-between mb-2">
              <span class="text-sm font-medium" :class="isDark ? 'text-gray-300' : 'text-gray-700'">
                {{ item.label }}
              </span>
              <button @click="copyToClipboard(item.code, section.id + idx)"
                class="flex items-center gap-1 px-3 py-1.5 rounded-lg text-xs transition-all"
                :class="isDark ? 'bg-[var(--bg-hover)] hover:bg-[var(--bg-secondary)]' : 'bg-gray-100 hover:bg-gray-200'">
                <Check v-if="copiedSection === section.id + idx" class="w-3.5 h-3.5 text-green-500" />
                <Copy v-else class="w-3.5 h-3.5" />
                {{ copiedSection === section.id + idx ? '已复制' : '复制' }}
              </button>
            </div>
            <pre class="p-4 rounded-xl text-xs overflow-x-auto font-mono"
              :class="isDark ? 'bg-[var(--bg-hover)]' : 'bg-gray-50'"><code>{{ item.code }}</code></pre>
          </div>
        </div>
      </div>
    </div>

    <!-- 响应格式说明 -->
    <div class="card">
      <div class="flex items-center gap-2 mb-4">
        <Code class="w-5 h-5 text-indigo-500" />
        <h3 class="text-lg font-semibold">响应格式</h3>
      </div>

      <div class="bg-green-500/10 border border-green-500/30 rounded-xl p-4 mb-4">
        <p class="text-xs font-medium text-green-400 mb-2">成功响应 (code: 0)</p>
        <pre class="text-xs font-mono overflow-x-auto" :class="isDark ? 'text-gray-300' : 'text-gray-700'">{
  "code": 0,
  "message": "success",
  "data": {
    "id": "abc123",
    "name": "image.png",
    "url": "https://cdn.example.com/image.png",
    "size": 102400,
    "type": "image/webp",
    "channel": "telegram",
    "links": {
      "url": "https://cdn.example.com/image.png",
      "markdown": "![image](https://cdn.example.com/image.png)",
      "html": "&lt;img src=\"https://cdn.example.com/image.png\" alt=\"image\"&gt;"
    }
  }
}</pre>
      </div>

      <div class="grid grid-cols-1 sm:grid-cols-3 gap-3 text-xs">
        <div class="p-3 rounded-lg" :class="isDark ? 'bg-[var(--bg-hover)]' : 'bg-gray-50'">
          <span class="font-medium text-green-500">上传接口</span>
          <p class="mt-1 font-mono" :class="isDark ? 'text-gray-400' : 'text-gray-600'">POST /api/v1/upload</p>
        </div>
        <div class="p-3 rounded-lg" :class="isDark ? 'bg-[var(--bg-hover)]' : 'bg-gray-50'">
          <span class="font-medium text-blue-500">匿名接口</span>
          <p class="mt-1 font-mono" :class="isDark ? 'text-gray-400' : 'text-gray-600'">POST /api/v1/upload/anonymous
          </p>
        </div>
        <div class="p-3 rounded-lg" :class="isDark ? 'bg-[var(--bg-hover)]' : 'bg-gray-50'">
          <span class="font-medium text-purple-500">批量接口</span>
          <p class="mt-1 font-mono" :class="isDark ? 'text-gray-400' : 'text-gray-600'">POST /api/v1/upload/multiple</p>
        </div>
      </div>
    </div>
  </div>
</template>
