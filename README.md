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

### 上传功能

| 功能 | 说明 |
|------|------|
| **普通上传** | 支持拖拽、粘贴、多文件同时上传 |
| **批量上传** | 一次选择多个文件，批量处理 |
| **粘贴上传** | 支持 Ctrl+V 粘贴剪贴板图片 |
| **上传进度** | 实时显示上传进度 |
| **匿名上传** | 无需登录即可上传（5次/分钟限制，文件最大5MB） |
| **目录管理** | 支持创建目录分类管理文件 |

### 文件管理

| 功能 | 说明 |
|------|------|
| **文件列表** | 分页展示，支持缩略图/列表视图 |
| **时间筛选** | 按上传时间筛选，支持 7天/30天/90天/1年 快速预设 |
| **批量操作** | 支持全选当前页、批量删除 |
| **一键清理** | 一键删除指定时间范围前的所有文件 |
| **链接复制** | 一键复制文件链接 |

### 渠道管理

| 功能 | 说明 |
|------|------|
| **渠道配置** | 添加、编辑、删除、启用/禁用存储渠道 |
| **渠道权重** | 设置不同渠道的上传权重 |
| **健康检查** | 批量检测渠道可用性 |
| **连接测试** | 配置完成后测试渠道连接是否正常 |
| **额度控制** | 支持每日/每小时上传限制、配额限制 |

### 认证与安全

| 功能 | 说明 |
|------|------|
| **访问密码** | 设置管理后台访问密码 |
| **API Token** | 创建、删除、启用/禁用 API Token |
| **权限控制** | Token 细粒度权限控制（upload/upload:multiple/read/delete） |
| **IP 限流** | 匿名上传 IP 级别限流保护 |
| **CORS** | 跨域访问支持 |

### 统计报表

| 功能 | 说明 |
|------|------|
| **总览统计** | 总文件数、总大小、今日上传数 |
| **渠道统计** | 各渠道上传成功/失败数、成功率 |
| **使用趋势** | 每日/每周上传量趋势图 |

### 第三方集成

支持 Typora、VS Code、Python、JavaScript 等客户端集成，详见管理后台「集成示例」页面。

## 技术栈

### 后端
- Go 1.21+
- Gin (Web 框架)
- GORM (ORM)
- SQLite (数据库)
- Viper (配置管理)
- Zap (日志)
- github.com/chai2010/webp (WebP 编码)
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
# 安装前端依赖
cd admin && npm install
cd ../site && npm install

# 构建前端资源到嵌入目录
cd admin && npm run build
cd ../site && npm run build

# 启动后端
cd server && go run .
```

### 构建

```bash
# 构建所有（前端 + 后端）
make build

# 仅构建后端
make build-server

# 仅构建前端
make build-frontend
```

### Docker 部署

```bash
docker build -t imgbed .
docker run -p 8080:8080 -v ./data:/app/data imgbed
```

## 配置

配置文件：`server/config.yaml`

```yaml
app:
  name: ImgBed
  mode: debug      # debug / release
  host: 0.0.0.0
  port: 8080

database:
  type: sqlite
  path: ./data/imgbed.db

jwt:
  secret: your-secret-key
  expire: 86400    # 24小时

upload:
  maxSize: 20971520      # 20MB
  defaultChannel: local

compression:
  enabled: true
  quality: 80
  format: webp
  maxWidth: 1920
  maxHeight: 1080

anonymous:
  enabled: true
  rateLimit: 5            # 5次/分钟
  dailyLimit: 100
  maxFileSize: 5242880    # 5MB
```

## API 接口

| 接口 | 方法 | 说明 |
|------|------|------|
| `/api/v1/upload` | POST | 上传文件（需认证） |
| `/api/v1/upload/multiple` | POST | 批量上传 |
| `/api/v1/upload/anonymous` | POST | 匿名上传 |
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
