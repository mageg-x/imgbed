# ImgBed

开源图床聚合工具，采用 Go 后端 + Vue3 前端架构，支持多种存储渠道，提供简单易用的图片上传和外链管理能力。

## 功能特性

### 核心功能

| 功能 | 说明 |
|------|------|
| **多渠道存储** | 支持本地存储、Telegram Channel、Cloudflare R2、S3 兼容（AWS/MinIO/阿里云 OSS）、Discord、HuggingFace Dataset |
| **图片压缩** | 上传前自动压缩，支持 WebP 格式转换，可配置压缩质量和最大尺寸 |
| **智能调度** | 支持轮询（round_robin）、随机（random）、优先级（priority）三种策略，自动切换可用渠道 |
| **失败重试** | 上传失败自动尝试其他渠道，防止单渠道故障导致服务中断 |
| **多格式链接** | 上传成功后返回 URL、Markdown、HTML 三种格式的链接 |
| **秒传** | 基于 SHA-256 校验，已存在的文件可直接跳过上传 |

### 上传功能

| 功能 | 说明 |
|------|------|
| **普通上传** | 支持拖拽、粘贴、多文件同时上传 |
| **批量上传** | 一次选择多个文件，批量处理 |
| **粘贴上传** | 支持 Ctrl+V 粘贴剪贴板图片 |
| **上传进度** | 实时显示上传进度 |
| **秒传加速** | 已存在的文件直接返回链接，无需重复上传 |
| **速率限制** | 基于 IP 的上传频率限制 |

### 文件管理

| 功能 | 说明 |
|------|------|
| **文件列表** | 分页展示，支持缩略图/列表视图切换 |
| **搜索筛选** | 支持文件名搜索、时间范围筛选 |
| **快速筛选** | 支持 今天、7天、30天、90天 快速预设 |
| **批量操作** | 支持全选当前页、批量删除 |
| **一键清理** | 一键删除指定时间范围前的所有文件 |
| **链接复制** | 一键复制文件链接 |
| **图片预览** | 点击文件可在线预览 |

### 渠道管理

| 功能 | 说明 |
|------|------|
| **渠道配置** | 添加、编辑、删除、启用/禁用存储渠道 |
| **渠道权重** | 设置不同渠道的上传权重 |
| **健康检查** | 批量检测渠道可用性 |
| **连接测试** | 配置完成后测试渠道连接是否正常 |
| **额度控制** | 支持每日/每小时上传限制、配额限制 |
| **冷却机制** | 失败后自动进入冷却期，一段时间后自动恢复 |

### 认证与安全

| 功能 | 说明 |
|------|------|
| **访问密码** | 设置管理后台访问密码 |
| **API Token** | 创建、删除、启用/禁用 API Token |
| **权限控制** | Token 细粒度权限控制（upload/upload:multiple/read/delete） |
| **IP 限流** | 上传 IP 级别限流保护 |
| **CORS** | 跨域访问支持 |
| **CSRF 保护** | 表单提交 CSRF 令牌保护 |
| **安全头** | X-Frame-Options、X-Content-Type-Options 等安全响应头 |

### 统计报表

| 功能 | 说明 |
|------|------|
| **总览统计** | 总文件数、总大小、今日上传数 |
| **渠道统计** | 各渠道上传成功/失败数、成功率 |
| **使用趋势** | 每日/每周上传量趋势图 |

### 第三方集成

支持 Typora、VS Code、Python、JavaScript 等客户端集成，详见管理后台「集成示例」页面。

### 前端特性

| 功能 | 说明 |
|------|------|
| **多语言** | 支持中文、英文 |
| **主题切换** | 支持亮色/暗色模式 |
| **响应式设计** | 适配桌面端和移动端 |

## 技术栈

### 后端
- Go 1.25+
- Gin (Web 框架)
- GORM (ORM)
- SQLite (数据库)
- Viper (配置管理)
- Zap (日志)
- github.com/deepteams/webp (WebP 编码，纯 Go 无需 CGO)
- github.com/disintegration/imaging (图片缩放)

### 前端
- Vue 3 (Composition API + `<script setup>` 语法)
- Vite (构建工具)
- Element Plus (UI 组件库)
- Tailwind CSS (样式)
- Pinia (状态管理)
- Vue Router (路由)
- ECharts (图表)

## 快速开始

### 开发环境

```bash
# 安装前端依赖并构建
cd admin && npm install && npm run build
cd ../site && npm install && npm run build

# 启动后端
cd server && go run .
```

### 构建

```bash
# 构建所有（前端 + 后端）
make build
```



## 配置

ImgBed 运行后会创建数据库文件，位置因平台而异：

| 平台 | 数据库路径 |
|------|---------|
| Windows | `%APPDATA%\ImgBed\imgbed.db` |
| macOS | `~/Library/Application Support/ImgBed/imgbed.db` |
| Linux | `~/.imgbed/imgbed.db` |

所有配置通过管理后台界面在线修改，无需编辑配置文件。

## API 接口

| 接口 | 方法 | 说明 |
|------|------|------|
| `/api/v1/auth/login` | POST | 用户登录 |
| `/api/v1/auth/admin/login` | POST | 管理员登录 |
| `/api/v1/upload` | POST | 上传文件（需认证） |
| `/api/v1/upload/multiple` | POST | 批量上传 |
| `/api/v1/file/check/:checksum` | GET | 秒传检查 |
| `/api/v1/files` | GET | 文件列表 |
| `/api/v1/files` | DELETE | 批量删除 |
| `/api/v1/files/cleanup` | POST | 一键清理 |
| `/api/v1/tokens` | GET/POST/DELETE | Token 管理 |
| `/api/v1/channel` | GET/POST | 渠道管理 |
| `/api/v1/stats/overview` | GET | 统计概览 |
| `/api/v1/config` | GET/PUT | 配置管理 |

详细 API 文档见 [PRD.md](./docs/PRD.md)

## 端口

| 服务 | 端口 |
|------|------|
| 后端 API | 8080 |

## License

MIT
