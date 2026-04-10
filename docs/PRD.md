# ImgBed 产品需求文档 (PRD)

> 版本: v2.0  
> 日期: 2026-04-06  
> 状态: 已更新

---

## 1. 产品概述

### 1.1 产品定位

ImgBed 是一款**开源免费图床聚合工具**，专注于聚合各大免费存储渠道，提供简单易用的图片上传和外链管理能力。采用 **Go 后端 + Vue3 前端** 架构，前端和管理后台嵌入到单一二进制文件中，实现零依赖部署。

### 1.2 目标用户

- 个人用户：需要免费图床服务的博客作者、开发者
- 内容创作者：需要稳定免费图片外链的用户

### 1.3 核心价值

| 价值点 | 说明 |
|--------|------|
| 免费渠道聚合 | 支持多种免费存储渠道，最大化白嫖能力 |
| 极简部署 | 单一二进制文件，开箱即用 |
| 智能调度 | 自动切换渠道，防止封号 |
| 图片优化 | 自动压缩和格式转换，节省空间 |
| 统计报表 | 清晰展示各渠道使用情况 |

---

## 2. 功能需求

### 2.1 功能模块总览

```
┌─────────────────────────────────────────────────────────────────┐
│                           ImgBed                                │
├─────────────────────────────────────────────────────────────────┤
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────────────────┐  │
│  │   上传端    │  │   管理端    │  │        API 服务         │  │
│  ├─────────────┤  ├─────────────┤  ├─────────────────────────┤  │
│  │ - 图片上传  │  │ - 图片管理  │  │ - RESTful API          │  │
│  │ - 拖拽粘贴  │  │ - 渠道配置  │  │ - 批量操作 API         │  │
│  │ - 批量上传  │  │ - 统计报表  │  │                         │  │
│  │ - 链接复制  │  │ - 系统设置  │  │                         │  │
│  └─────────────┘  └─────────────┘  └─────────────────────────┘  │
└─────────────────────────────────────────────────────────────────┘
```

### 2.2 功能清单

#### 2.2.1 图片上传模块

| 功能 | 优先级 | 说明 |
|------|--------|------|
| 普通上传 | P0 | 支持图片上传 |
| 拖拽上传 | P1 | 支持拖拽文件到上传区域 |
| 批量上传 | P1 | 支持多图片同时上传 |
| 粘贴上传 | P1 | 支持剪贴板粘贴图片上传 |
| 上传进度 | P1 | 实时显示上传进度 |
| 图片压缩 | P0 | 上传前自动压缩，可配置压缩率 |
| 格式转换 | P0 | 自动转换为 WebP 等高效格式 |
| 上传限制 | P1 | 文件大小限制、类型限制 |
| **秒传** | **P0** | **基于 SHA256 校验，已存在文件直接返回** |

#### 2.2.2 图片管理模块

| 功能 | 优先级 | 说明 |
|------|--------|------|
| 图片列表 | P0 | 分页展示、缩略图/列表视图切换 |
| 图片搜索 | P1 | 按文件名搜索 |
| **时间范围筛选** | **P0** | **按上传时间筛选（支持日期范围、快速预设）** |
| **快速筛选预设** | **P0** | **一键筛选：7天前、30天前、90天前、1年前的图片** |
| **批量选择** | **P0** | **支持全选当前页、全选筛选结果** |
| 图片预览 | P0 | 图片在线预览 |
| 图片下载 | P0 | 单图片下载 |
| 图片删除 | P0 | 单图片删除、批量删除 |
| **一键清理** | **P1** | **一键删除指定时间范围前的所有图片** |
| 链接复制 | P0 | 一键复制多种格式链接（Markdown、HTML、直接链接等） |
| 图片信息 | P1 | 查看图片元数据（大小、类型、上传时间、存储渠道等） |

#### 2.2.3 存储渠道模块

| 功能 | 优先级 | 说明 |
|------|--------|------|
| 本地存储 | P0 | 本地文件系统存储 |
| Telegram | P0 | Telegram Channel 存储 |
| Cloudflare R2 | P0 | Cloudflare R2 对象存储 |
| S3 兼容存储 | P0 | AWS S3、MinIO 等 S3 兼容存储 |
| Discord | P1 | Discord Channel 存储 |
| HuggingFace | P1 | HuggingFace Dataset 存储 |
| 渠道配置 | P0 | 添加、编辑、删除、启用/禁用渠道 |
| 渠道测试 | P1 | 测试渠道连接是否正常 |
| 智能调度 | P0 | 多渠道自动切换，防止封号 |
| 失败重试 | P0 | 上传失败自动尝试其他渠道，支持轮询/随机/优先级策略 |

#### 2.2.4 统计报表模块

| 功能 | 优先级 | 说明 |
|------|--------|------|
| 渠道使用统计 | P0 | 各渠道上传成功/失败统计 |
| 上传量统计 | P1 | 每日/每周上传量统计 |
| 可视化展示 | P1 | 图表形式展示统计数据 |

#### 2.2.5 数据备份模块

| 功能 | 优先级 | 说明 |
|------|--------|------|
| 自动备份 | P1 | 可配置备份周期，自动备份 SQLite 数据库 |
| 手动备份 | P0 | 一键创建备份 |
| 备份管理 | P0 | 查看备份列表、删除备份 |
| 一键恢复 | P0 | 从备份文件恢复数据库 |

#### 2.2.6 系统配置模块

| 功能 | 优先级 | 说明 |
|------|--------|------|
| 访问密码 | P0 | 设置访问密码保护 |
| 上传配置 | P0 | 默认渠道、文件限制、压缩配置 |
| 页面配置 | P1 | 站点名称等 |

#### 2.2.6 第三方接入模块

| 功能 | 优先级 | 说明 |
|------|--------|------|
| **API Token 管理** | **P0** | **创建、删除、启用/禁用 API Token** |
| **Token 权限控制** | **P0** | **Token 细粒度权限控制（upload/read/delete）** |
| **Token 认证接口** | **P0** | **API Token 认证的完整接口** |
| **粘贴上传支持** | **P1** | **支持剪贴板粘贴图片上传（前端示例）** |
| **集成示例** | **P1** | **提供 Typora、Python、JavaScript 等集成示例** |

---

## 3. 技术架构

### 3.1 整体架构

```
┌─────────────────────────────────────────────────────────────────────┐
│                           用户层                                     │
│  ┌─────────────────┐  ┌─────────────────┐  ┌─────────────────────┐  │
│  │    Web 浏览器    │  │   移动端浏览器   │  │     API 客户端      │  │
│  └─────────────────┘  └─────────────────┘  └─────────────────────┘  │
└─────────────────────────────────────────────────────────────────────┘
                                    │
                                    ▼
┌─────────────────────────────────────────────────────────────────────┐
│                    单一二进制文件                           │
│  ┌─────────────────────────────────────────────────────────────┐    │
│  │                   嵌入式前端资源                      │    │
│  │  ┌─────────────┐  ┌─────────────┐                        │  │    │
│  │  │   上传端    │  │   管理端    │                        │  │    │
│  │  │  /upload    │  │  /admin     │                        │  │    │
│  │  └─────────────┘  └─────────────┘                        │  │    │
│  └─────────────────────────────────────────────────────────────┘    │
│  ┌─────────────────────────────────────────────────────────────┐    │
│  │                      API Gateway (Gin/Echo)                  │    │
│  │  ┌─────────────┐  ┌─────────────┐                            │  │    │
│  │  │  认证中间件  │  │  日志中间件  │                            │  │    │
│  │  └─────────────┘  └─────────────┘                            │  │    │
│  └─────────────────────────────────────────────────────────────┘    │
│  ┌─────────────────────────────────────────────────────────────┐    │
│  │                       业务逻辑层                             │    │
│  │  ┌───────────┐  ┌───────────┐  ┌───────────┐              │  │    │
│  │  │ 文件服务   │  │ 图片处理   │  │ 配置服务   │              │  │    │
│  │  └───────────┘  └───────────┘  └───────────┘              │  │    │
│  │  ┌───────────┐  ┌───────────┐                                │  │    │
│  │  │ 调度服务   │  │ 统计服务   │                                │  │    │
│  │  └───────────┘  └───────────┘                                │  │    │
│  └─────────────────────────────────────────────────────────────┘    │
│  ┌─────────────────────────────────────────────────────────────┐    │
│  │                     存储驱动抽象层                           │    │
│  │  ┌─────────┐ ┌─────────┐ ┌─────────┐ ┌─────────┐ ┌───────┐  │    │
│  │  │ Local   │ │Telegram │ │   R2    │ │   S3    │ │Discord│  │    │
│  │  │ Driver  │ │ Driver  │ │ Driver  │ │ Driver  │ │Driver │  │    │
│  │  └─────────┘ └─────────┘ └─────────┘ └─────────┘ └───────┘  │    │
│  │                      HuggingFace Driver                      │    │
│  └─────────────────────────────────────────────────────────────┘    │
└─────────────────────────────────────────────────────────────────────┘
                                    │
                                    ▼
┌─────────────────────────────────────────────────────────────────────┐
│                          存储层                                      │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────────────────────┐  │
│  │   SQLite    │  │  本地文件    │  │        外部存储服务          │  │
│  │  (元数据)   │  │  (可选)      │  │  Telegram/R2/S3/Discord/HF  │  │
│  └─────────────┘  └─────────────┘  └─────────────────────────────┘  │
└─────────────────────────────────────────────────────────────────────┘
```

### 3.2 技术选型

#### 3.2.1 后端技术栈

| 组件 | 技术选型 | 说明 |
|------|----------|------|
| 编程语言 | Go 1.21+ | 高性能、并发友好 |
| Web 框架 | Gin / Echo | 轻量级、高性能 |
| ORM | GORM | 数据库操作 |
| 配置管理 | Viper | 多格式配置支持 |
| 日志 | Zap | 结构化日志 |
| 嵌入资源 | embed | 前端资源嵌入二进制 |
| 图片处理 | imaging / gif | 图片压缩、格式转换 |

#### 3.2.2 前端技术栈

| 组件 | 技术选型 | 说明 |
|------|----------|------|
| 框架 | Vue 3 | Composition API |
| 构建工具 | Vite | 快速构建 |
| UI 组件库 | Element Plus | 企业级组件 |
| 状态管理 | Pinia | Vue3 官方推荐 |
| 路由 | Vue Router 4 | 前端路由 |
| HTTP 客户端 | Axios | HTTP 请求 |
| 图表库 | ECharts | 数据可视化 |

#### 3.2.3 存储技术

| 组件 | 技术选型 | 说明 |
|------|----------|------|
| 元数据库 | SQLite | 轻量级、零配置、嵌入二进制同目录 |
| 本地存储 | 文件系统 | 可选本地存储 |
| 对象存储 | AWS SDK for Go v2 | 兼容 R2、MinIO 等 |

---

## 4. 核心功能设计

### 4.1 图片压缩与格式转换

#### 4.1.1 设计目标
- 上传前自动压缩图片，减小文件大小
- 支持转换为高效格式（WebP）
- 可配置压缩质量和目标格式

#### 4.1.2 功能说明

| 配置项 | 说明 | 默认值 |
|--------|------|--------|
| `enabled` | 是否启用压缩 | true |
| `quality` | 压缩质量 (0-100) | 80 |
| `format` | 目标格式 (original/webp) | webp |
| `maxWidth` | 最大宽度（超过则缩放） | 1920 |
| `maxHeight` | 最大高度（超过则缩放） | 1080 |

#### 4.1.3 处理流程
1. 接收上传图片
2. 检查是否启用压缩
3. 如果启用，进行压缩和格式转换
4. 保存处理后的图片
5. 上传到存储渠道

---

### 4.2 智能调度与失败重试

#### 4.2.1 设计目标
- 自动选择可用渠道
- 上传失败时自动切换其他渠道重试
- 支持多种调度策略

#### 4.2.2 调度策略

| 策略 | 说明 |
|------|------|
| `round_robin` | 轮询选择渠道 |
| `random` | 随机选择渠道 |
| `priority` | 按优先级选择渠道 |
| `weight` | 按权重比例加权随机选择（轮盘赌算法） |

#### 4.2.3 重试机制
- 上传失败时自动尝试下一个渠道
- 最多重试 N 次（可配置）
- 记录失败渠道，下次优先选择其他渠道

#### 4.2.4 渠道状态
- `healthy` - 健康可用
- `unhealthy` - 连续失败，暂时不可用
- `disabled` - 手动禁用

---

### 4.3 统计报表

#### 4.3.1 设计目标
- 清晰展示各渠道使用情况
- 可视化统计数据
- 帮助用户了解各渠道表现

#### 4.3.2 统计指标

| 指标 | 说明 |
|------|------|
| 总上传数 | 累计上传图片数量 |
| 今日上传数 | 今日上传图片数量 |
| 总成功数 | 累计上传成功数量 |
| 总失败数 | 累计上传失败数量 |
| 成功率 | 上传成功率 |

#### 4.3.3 渠道统计

| 指标 | 说明 |
|------|------|
| 渠道上传数 | 该渠道上传总数 |
| 渠道成功数 | 该渠道成功数 |
| 渠道失败数 | 该渠道失败数 |
| 渠道成功率 | 该渠道成功率 |
| 最后使用时间 | 该渠道最后使用时间 |

#### 4.3.4 可视化图表
- 渠道使用占比饼图
- 每日上传趋势折线图
- 各渠道成功率柱状图

---

## 5. API 接口设计

### 5.1 设计原则

| 原则 | 说明 |
|------|------|
| RESTful 风格 | 资源导向，HTTP 方法语义化 |
| 版本控制 | URL 路径版本 `/api/v1`，便于后续扩展 |
| 统一响应 | 标准化的成功/错误响应格式 |

### 5.2 统一响应格式

**成功响应**
```json
{
  "code": 0,
  "message": "success",
  "data": { ... }
}
```

**错误响应**
```json
{
  "code": 10001,
  "message": "图片不存在",
  "data": null
}
```

**分页响应**
```json
{
  "code": 0,
  "message": "success",
  "data": {
    "items": [...],
    "pagination": {
      "page": 1,
      "pageSize": 50,
      "total": 100,
      "totalPages": 2
    }
  }
}
```

### 5.3 错误码设计

| 错误码 | 说明 |
|--------|------|
| 0 | 成功 |
| 10001 | 资源不存在 |
| 10002 | 参数错误 |
| 10003 | 未授权访问 |
| 20001 | 服务器内部错误 |
| 30001 | 渠道不可用 |
| 30002 | 上传失败 |

### 5.4 接口概览

```
/api/v1
│
├── /auth                          # 认证接口
│   ├── POST /login                # 登录
│   ├── POST /admin/login          # 管理员登录
│   ├── POST /logout               # 登出
│   ├── GET  /check                # 会话检查
│   └── POST /password             # 设置密码（管理员）
│
├── /file                          # 文件访问接口
│   ├── GET  /check/:checksum      # 秒传检查（基于 SHA256）
│   ├── GET  /:id                  # 获取文件
│   ├── GET  /:id/info             # 获取文件信息
│   ├── GET  /:id/download         # 下载文件
│   └── GET  /:id/proxy            # 代理访问
│
├── /upload                        # 上传接口（需认证）
│   ├── POST /                     # 上传图片
│   └── POST /multiple             # 批量上传
│
├── /files                         # 文件管理接口（需认证）
│   ├── GET  /                     # 文件列表
│   ├── GET  /ids                  # 文件ID列表
│   ├── DELETE /                   # 批量删除
│   ├── POST /cleanup/preview      # 一键清理预览
│   └── POST /cleanup              # 执行一键清理
│
├── /channel                       # 存储渠道接口（需认证）
│   ├── GET  /                     # 渠道列表
│   ├── GET  /:id                  # 渠道详情
│   ├── GET  /:id/status           # 渠道状态
│   ├── GET  /:id/stats            # 渠道统计
│   ├── POST /                     # 创建渠道（管理员）
│   ├── PUT  /:id                  # 更新渠道（管理员）
│   ├── DELETE /:id                # 删除渠道（管理员）
│   ├── PUT  /:id/enable           # 启用/禁用渠道（管理员）
│   ├── PUT  /:id/weight           # 设置渠道权重（管理员）
│   ├── GET  /:id/health           # 健康检查（管理员）
│   └── POST /:id/test             # 测试渠道连接（管理员）
│
├── /channels                      # 渠道状态接口
│   ├── GET  /status               # 所有渠道状态
│   └── POST /health-check         # 健康检查（管理员）
│
├── /stats                         # 统计接口
│   ├── GET  /overview             # 总览统计
│   ├── GET  /channels             # 渠道统计
│   ├── GET  /trend                # 趋势统计
│   └── GET  /weekly              # 周统计
│
├── /tokens                        # API Token 接口（需认证）
│   ├── GET  /                     # Token 列表
│   ├── POST /                     # 创建 Token
│   ├── DELETE /:token             # 删除 Token
│   └── PUT  /:token/enable        # 启用/禁用 Token
│
├── /config                        # 配置接口
│   ├── GET  /                     # 获取所有配置
│   ├── GET  /:key                 # 获取单个配置
│   ├── PUT  /                     # 更新配置
│   ├── GET  /upload               # 获取上传配置
│   ├── PUT  /upload               # 更新上传配置
│   ├── GET  /site                 # 获取站点配置
│   ├── PUT  /site                 # 更新站点配置
│   ├── GET  /auth                # 获取认证配置
│   ├── PUT  /auth                # 更新认证配置
│   ├── GET  /rate-limit          # 获取限速配置
│   ├── PUT  /rate-limit          # 更新限速配置
│   ├── GET  /schedule             # 获取调度配置
│   ├── PUT  /schedule             # 更新调度配置
│   ├── GET  /cdn                 # 获取 CDN 配置
│   ├── PUT  /cdn                 # 更新 CDN 配置
│   ├── GET  /backup              # 获取备份配置
│   └── PUT  /backup              # 更新备份配置
│
├── /admin                         # 管理后台接口（需管理员）
│   ├── GET  /dashboard            # 仪表盘数据
│   ├── GET  /statistics           # 统计数据
│   ├── GET  /files               # 文件列表
│   ├── DELETE /files/:id         # 删除文件
│   ├── DELETE /files             # 批量删除文件
│   ├── GET  /channels            # 渠道列表
│   ├── POST /channels            # 创建渠道
│   ├── PUT  /channels/:id        # 更新渠道
│   ├── DELETE /channels/:id      # 删除渠道
│   ├── PUT  /channels/:id/enable # 启用/禁用渠道
│   ├── POST /channels/:id/test   # 测试渠道
│   ├── GET  /settings            # 获取设置
│   ├── PUT  /settings            # 更新设置
│   ├── GET  /tokens             # Token 列表
│   ├── POST /tokens             # 创建 Token
│   ├── DELETE /tokens/:id       # 删除 Token
│   ├── PUT  /tokens/:id/enable  # 启用/禁用 Token
│   ├── GET  /backup/list        # 备份列表
│   ├── POST /backup/create      # 创建备份
│   ├── DELETE /backup           # 删除备份
│   └── POST /backup/restore     # 恢复备份
│
└── /image/:id                     # 图片访问 (GET/HEAD)
```

### 5.5 核心接口详细设计

#### 5.5.1 认证接口

**登录**
```
POST /api/v1/auth/login
Content-Type: application/json

Request:
{
  "password": "your_password"
}

Response:
{
  "code": 0,
  "message": "success",
  "data": {
    "token": "eyJhbGciOiJIUzI1NiIs...",
    "expiresAt": 1712345678
  }
}
```

#### 5.5.2 图片上传接口

**上传图片（密码/Token 认证）**
```
POST /api/v1/upload
Content-Type: multipart/form-data
Authorization: Bearer <token>
或
X-API-Token: <api_token>
X-API-Secret: <api_secret>

Form Data:
- file: 图片文件 (必填)
- channel: 指定渠道 ID (可选)

Response:
{
  "code": 0,
  "message": "success",
  "data": {
    "id": "abc123",
    "name": "image.png",
    "url": "https://cdn.discord.com/attachments/xxx/image.png",
    "size": 102400,
    "type": "image/webp",
    "channel": "telegram-main",
    "uploadedAt": 1712345678,
    "links": {
      "url": "https://cdn.discord.com/attachments/xxx/image.png",
      "markdown": "![image](https://cdn.discord.com/attachments/xxx/image.png)",
      "html": "<img src=\"https://cdn.discord.com/attachments/xxx/image.png\" alt=\"image\">"
    }
  }
}
```

**批量上传（Token 认证）**
```
POST /api/v1/upload/multiple
Content-Type: multipart/form-data
X-API-Token: <api_token>
X-API-Secret: <api_secret>

Form Data:
- files: 图片文件数组 (必填)

Response:
{
  "code": 0,
  "message": "success",
  "data": [
    {
      "name": "image1.jpg",
      "success": true,
      "result": {
        "id": "abc123",
        "url": "https://cdn.discord.com/attachments/xxx/image1.jpg",
        "links": { ... }
      }
    },
    {
      "name": "image2.jpg",
      "success": false,
      "error": "文件过大"
    }
  ]
}
```

#### 5.5.3 API Token 接口

**Token 列表**
```
GET /api/v1/tokens

Response:
{
  "code": 0,
  "message": "success",
  "data": {
    "items": [
      {
        "id": "token123",
        "name": "我的博客",
        "token": "pk_xxx",
        "permissions": ["upload"],
        "enabled": true,
        "expiresAt": 1720000000,
        "createdAt": 1712345678,
        "lastUsedAt": 1712400000
      }
    ]
  }
}
```

**创建 Token**
```
POST /api/v1/tokens
Content-Type: application/json

Request:
{
  "name": "我的博客",
  "permissions": ["upload"],
  "expiresAt": 1720000000
}

Response:
{
  "code": 0,
  "message": "success",
  "data": {
    "id": "token123",
    "name": "我的博客",
    "token": "pk_abc123xyz",
    "secret": "sk_secret123",
    "permissions": ["upload"],
    "enabled": true,
    "expiresAt": 1720000000,
    "createdAt": 1712345678
  }
}
注意: secret 只显示一次，请妥善保存！
```

**删除 Token**
```
DELETE /api/v1/tokens/:id

Response:
{
  "code": 0,
  "message": "success",
  "data": null
}
```

**启用/禁用 Token**
```
POST /api/v1/tokens/:id/toggle

Response:
{
  "code": 0,
  "message": "success",
  "data": {
    "id": "token123",
    "enabled": false
  }
}
```

#### 5.5.4 图片接口

**图片列表**
```
GET /api/v1/images?page=1&pageSize=50&search=keyword

Query Parameters:
- page: 页码 (默认 1)
- pageSize: 每页数量 (默认 50)
- search: 搜索关键字
- channel: 渠道筛选
- startTime: 开始时间 (Unix 时间戳)
- endTime: 结束时间 (Unix 时间戳)
- olderThan: 筛选 N 天前的图片 (如 olderThan=30 表示 30 天前)
- sortBy: 排序字段 (name,size,uploadedAt)
- sortOrder: 排序方向 (asc,desc)

Response:
{
  "code": 0,
  "message": "success",
  "data": {
    "items": [
      {
        "id": "abc123",
        "name": "image.png",
        "url": "/image/abc123",
        "size": 102400,
        "type": "image/webp",
        "channel": "telegram-main",
        "uploadedAt": 1712345678
      }
    ],
    "pagination": {
      "page": 1,
      "pageSize": 50,
      "total": 100,
      "totalPages": 2
    }
  }
}
```

**删除图片**
```
DELETE /api/v1/images/abc123

Response:
{
  "code": 0,
  "message": "success",
  "data": null
}
```

**批量删除**
```
POST /api/v1/images/batch-delete
Content-Type: application/json

Request:
{
  "ids": ["abc123", "def456"]
}

Response:
{
  "code": 0,
  "message": "success",
  "data": {
    "success": ["abc123", "def456"],
    "failed": []
  }
}
```

**一键清理（预览）**
```
POST /api/v1/images/cleanup/preview
Content-Type: application/json

Request:
{
  "olderThan": 30,           // 删除 N 天前的图片
  "startTime": 1710000000,  // 或者指定开始时间
  "endTime": 1712000000,    // 或者指定结束时间
  "channel": "telegram-main" // 可选，只清理指定渠道
}

Response:
{
  "code": 0,
  "message": "success",
  "data": {
    "count": 150,            // 将要删除的图片数量
    "totalSize": 52428800,   // 将要释放的空间（字节）
    "preview": [              // 前 10 个预览
      {
        "id": "abc123",
        "name": "old-image.jpg",
        "size": 1048576,
        "uploadedAt": 1710000000
      }
    ]
  }
}
```

**一键清理（执行）**
```
POST /api/v1/images/cleanup
Content-Type: application/json

Request:
{
  "olderThan": 30,           // 删除 N 天前的图片
  "startTime": 1710000000,  // 或者指定开始时间
  "endTime": 1712000000,    // 或者指定结束时间
  "channel": "telegram-main" // 可选，只清理指定渠道
}

Response:
{
  "code": 0,
  "message": "success",
  "data": {
    "deletedCount": 150,      // 成功删除数量
    "failedCount": 0,         // 失败数量
    "freedSize": 52428800,    // 释放空间（字节）
    "failedIds": []           // 失败的 ID 列表
  }
}
```

#### 5.5.4 存储渠道接口

**渠道列表**
```
GET /api/v1/channels

Response:
{
  "code": 0,
  "message": "success",
  "data": {
    "items": [
      {
        "id": "telegram-main",
        "name": "Telegram 主渠道",
        "type": "telegram",
        "enabled": true,
        "status": "healthy",
        "priority": 1
      }
    ]
  }
}
```

**测试渠道连接**
```
POST /api/v1/channels/telegram-main/test

Response:
{
  "code": 0,
  "message": "success",
  "data": {
    "connected": true,
    "message": "连接成功"
  }
}
```

#### 5.5.5 统计接口

**总览统计**
```
GET /api/v1/stats/overview

Response:
{
  "code": 0,
  "message": "success",
  "data": {
    "totalUploads": 1500,
    "todayUploads": 25,
    "totalSuccess": 1450,
    "totalFailed": 50,
    "successRate": 96.67
  }
}
```

**渠道统计**
```
GET /api/v1/stats/channels

Response:
{
  "code": 0,
  "message": "success",
  "data": {
    "items": [
      {
        "channelId": "telegram-main",
        "channelName": "Telegram 主渠道",
        "totalUploads": 800,
        "successCount": 780,
        "failedCount": 20,
        "successRate": 97.5,
        "lastUsedAt": 1712345678
      }
    ]
  }
}
```

**趋势统计**
```
GET /api/v1/stats/trend?days=30

Response:
{
  "code": 0,
  "message": "success",
  "data": {
    "items": [
      {
        "date": "2026-03-01",
        "uploads": 50,
        "success": 48,
        "failed": 2
      }
    ]
  }
}
```

#### 5.5.6 配置接口

**获取配置**
```
GET /api/v1/config

Response:
{
  "code": 0,
  "message": "success",
  "data": {
    "site": {
      "name": "ImgBed"
    },
    "upload": {
      "defaultChannel": "telegram-main",
      "maxFileSize": 20971520,
      "allowedTypes": ["image/*"],
      "compression": {
        "enabled": true,
        "quality": 80,
        "format": "webp",
        "maxWidth": 1920,
        "maxHeight": 1080
      },
      "retry": {
        "enabled": true,
        "maxRetries": 3,
        "strategy": "round_robin"
      }
    }
  }
}
```

**更新配置**
```
PUT /api/v1/config
Content-Type: application/json

Request:
{
  "site": {
    "name": "My ImgBed"
  },
  "upload": {
    "compression": {
      "quality": 90
    }
  }
}

Response:
{
  "code": 0,
  "message": "success",
  "data": null
}
```

---

## 6. 第三方接入与集成

### 6.1 产品定位说明

ImgBed 是一个**免费图床聚合工具**，专注于：

- ✅ **代理上传流量**：图片上传经过 ImgBed 分发到各个免费存储渠道
- ❌ **不代理访问流量**：图片直接使用原始存储渠道的 URL，节省服务器带宽

### 6.2 Token 权限说明

| 权限 | 说明 |
|------|------|
| `upload` | 上传图片 |
| `upload:multiple` | 批量上传 |
| `read` | 读取文件列表和信息 |
| `delete` | 删除文件 |
| `*` | 所有权限 |

### 6.3 认证方式

#### 6.3.1 Token 认证（推荐）

**在请求 Header 中携带：**
```http
X-API-Token: your_token_here
X-API-Secret: your_secret_here
```

### 6.4 使用场景示例

#### 6.4.1 场景 1：Typora 配置

在 Typora 的「偏好设置」→「图像」中：

- **上传服务**：选择 `Custom Command`
- **命令**：
```bash
curl -X POST https://your-imgbed.com/api/v1/upload \
  -H "X-API-Token: your_token" \
  -H "X-API-Secret: your_secret" \
  -F "file=@${filepath}" \
  | grep -o '"url":"[^"]*"' | cut -d'"' -f4
```

#### 6.4.2 场景 2：Python 脚本上传

```python
import requests

class ImgBedClient:
    def __init__(self, base_url, api_token, api_secret):
        self.base_url = base_url
        self.api_token = api_token
        self.api_secret = api_secret

    def upload(self, image_path):
        with open(image_path, 'rb') as f:
            files = {'file': f}
            headers = {
                'X-API-Token': self.api_token,
                'X-API-Secret': self.api_secret
            }
            url = self.base_url + '/api/v1/upload'
            response = requests.post(url, files=files, headers=headers)
            return response.json()

# 使用 Token（需先在管理后台创建 API Token）
client = ImgBedClient(
    "https://your-imgbed.com",
    api_token="your_token",
    api_secret="your_secret"
)

result = client.upload("image.jpg")
if result['code'] == 0:
    print('Upload success:', result['data']['links']['markdown'])
```

#### 6.4.3 场景 3：JavaScript 博客编辑器集成（支持粘贴上传）

```html
<!DOCTYPE html>
<html>
<body>
    <textarea id="editor" placeholder="在这里写文章，Ctrl+V 粘贴图片..."></textarea>

    <script>
        const BASE_URL = 'https://your-imgbed.com';
        const API_TOKEN = 'your_token';
        const API_SECRET = 'your_secret';

        async function uploadImage(file) {
            const formData = new FormData();
            formData.append('file', file);
            const headers = {
                'X-API-Token': API_TOKEN,
                'X-API-Secret': API_SECRET
            };
            const response = await fetch(BASE_URL + '/api/v1/upload', {
                method: 'POST',
                headers,
                body: formData
            });
            return await response.json();
        }

        // 监听粘贴事件
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
                        const start = textarea.selectionStart;
                        const end = textarea.selectionEnd;
                        const text = textarea.value;

                        textarea.value = text.substring(0, start) + markdown + text.substring(end);
                        textarea.selectionStart = textarea.selectionEnd = start + markdown.length;
                    }
                    break;
                }
            }
        });
    </script>
</body>
</html>
```

### 6.5 安全建议

1. **创建专用 Token**：不要共用 Token，为每个应用创建独立 Token
2. **权限最小化**：只给必要的权限（如仅 `upload`）
3. **定期轮换 Token**：建议每 3-6 个月更换一次
4. **使用 HTTPS**：确保 Token 传输安全
5. **不要泄露 Token**：不要将 Token 提交到公开代码仓库

### 6.6 错误码补充说明

| 错误码 | 说明 |
|--------|------|
| 0 | 成功 |
| 10001 | 参数错误 |
| 10002 | 文件过大 |
| 10003 | 不支持的文件类型 |
| 10004 | 超过限速 |
| 30001 | 无效的 Token |
| 30002 | Token 已禁用 |
| 30003 | Token 已过期 |
| 30004 | 权限不足 |
| 40001 | 上传失败 |

---

## 8. 存储驱动设计

### 8.1 驱动接口定义

```go
type StorageDriver interface {
    Name() string
    Type() StorageType
    
    Upload(ctx context.Context, req *UploadRequest) (*UploadResult, error)
    GetURL(ctx context.Context, fileID string) (string, error)
    Delete(ctx context.Context, fileID string) error
    HealthCheck(ctx context.Context) error
}
```

### 8.2 支持的存储驱动

| 驱动 | 类型标识 | 特点 |
|------|----------|------|
| **本地存储** | `local` | 零依赖、直接文件系统存储 |
| **Telegram** | `telegram` | 免费、适合小文件 (≤20MB) |
| **Cloudflare R2** | `cfr2` | 无出站流量费用、S3 兼容 |
| **S3 兼容** | `s3` | AWS/MinIO/阿里云 OSS 等 |
| **Discord** | `discord` | 免费、支持较大文件 (≤25MB) |
| **HuggingFace** | `huggingface` | 开源项目托管、免费存储 |

---

## 9. 变更记录

| 版本 | 日期 | 说明 |
|------|------|------|
| **v2.1** | **2026-04-10** | **新增秒传功能、权重调度策略、数据备份恢复、API Token 权限控制** |
| v2.0 | 2026-04-06 | 聚焦个人白嫖图床需求，删除过度设计功能，新增图片压缩、统计报表、失败重试增强 |
| v1.2 | 2026-04-06 | 初始版本 |
