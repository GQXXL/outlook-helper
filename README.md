# Outlook取件助手

<div align="center">

![Outlook取件助手](https://img.shields.io/badge/Outlook-取件助手-blue?style=for-the-badge&logo=microsoft-outlook)
![Go](https://img.shields.io/badge/Go-1.23+-00ADD8?style=for-the-badge&logo=go)
![Vue.js](https://img.shields.io/badge/Vue.js-3.0+-4FC08D?style=for-the-badge&logo=vue.js)
![License](https://img.shields.io/badge/License-MIT-green?style=for-the-badge)

**专为Outlook令牌号邮箱设计的批量管理工具，支持令牌验证、邮件查询、邮箱标记等操作**

[功能特性](#功能特性) • [快速开始](#快速开始) • [Docker部署](#docker部署) • [API文档](#api文档)

</div>

## 📖 项目简介

Outlook取件助手是一个专门为**Outlook令牌号邮箱**设计的批量管理工具。通过标准的令牌格式（邮箱----密码----客户端ID----RefreshToken），实现对大量Outlook邮箱的自动化管理、邮件监控和批量操作。

项目没有杂七杂八的功能，只能获取邮件和给邮件打标签，仅此而已，也不会自动监控邮箱状态，需要收邮件了去点一下即可。

### 🎯 什么是令牌号邮箱？
令牌号邮箱是指通过特定格式组织的Outlook邮箱凭据：
```
邮箱地址----密码----客户端ID----RefreshToken
```

### 🎯 适用场景
- **邮件营销团队** - 批量管理营销邮箱，监控发送状态
- **企业IT管理** - 统一管理员工Outlook账户
- **自动化系统** - 集成邮件API，实现自动化邮件处理
- **薅羊毛** - 白嫖各种服务试用

## ✨ 功能特性

### 核心功能
- 📧 **令牌邮箱管理** - 支持标准令牌格式的邮箱批量导入和管理
- 🔄 **Outlook API集成** - 自动验证令牌有效性，获取邮件，清空收件箱
- 🏷️ **标签系统** - 对令牌邮箱进行分类管理，支持批量标记
- 📱 **现代化界面** - 响应式设计，完美适配各种设备

### 技术特色
- **标准令牌格式** - 完全支持 `邮箱----密码----客户端ID----RefreshToken` 格式
- **智能批量处理** - 支持最多30个令牌邮箱的批量添加和验证
- **并发令牌验证** - 可配置的并发数，高效验证大量令牌有效性
- **令牌状态监控** - 实时监控令牌有效性，自动标记失效账户
- **轻量级部署** - SQLite数据库 + Docker容器，部署简单可靠

## 🛠️ 技术栈

| 类型 | 技术 |
|------|------|
| **后端** | Go 1.23+ • Gin Web框架 • SQLite数据库 • JWT认证 |
| **前端** | Vue.js 3 • TypeScript • Element Plus • Vite • Pinia |
| **部署** | Docker • Docker Compose • Alpine Linux |

## 🚀 快速开始

## 前置操作

部署OutLook-API服务

部署API请参考此项目：[msOauth2api](https://github.com/HChaoHui/msOauth2api) 使用Vercel部署即可


### 方式一：Docker部署（推荐）

```bash
# 克隆项目
git clone https://github.com/your-username/outlook-helper.git
cd outlook-helper

# 配置环境变量
cp .env.example .env
# 编辑 .env 文件，修改管理员账户等配置

# 使用Docker Compose启动
docker-compose up -d

# 访问应用
open http://localhost:8080
```

### 方式二：本地开发

#### 环境要求
- Go 1.23+
- Node.js 18+
- SQLite3

#### 安装步骤

```bash
# 1. 克隆项目
git clone https://github.com/your-username/outlook-helper.git
cd outlook-helper

# 2. 配置环境变量
cp .env.example .env
# 编辑 .env 文件，修改管理员账户等配置

# 3. 安装依赖
go mod tidy
cd frontend && npm install && cd ..

# 4. 构建前端
cd frontend && npm run build && cd ..

# 5. 启动应用
go run backend/cmd/main.go
```

### 默认访问信息
- **访问地址**: http://ip:8080
- **登录方式**: 使用配置的AUTH_TOKEN授权码登录

## 🐳 Docker部署

### 使用Docker Compose（推荐）

```bash
# 启动服务
docker-compose up -d

# 查看日志
docker-compose logs -f

# 停止服务
docker-compose down
```

### 使用Docker

```bash
# 构建镜像
docker build -t outlook-helper .

# 运行容器（重要：必须挂载数据卷以防止数据丢失）
docker run -d \
  --name outlook-helper \
  -p 8080:8080 \
  -v outlook_data:/app/data \
  -v outlook_logs:/app/logs \
  -e AUTH_TOKEN=your-super-secret-auth-token \
  -e OUTLOOK_API_BASE_URL=https://your-outlook-api-domain.vercel.app \
  linqiu1199/outlook-helper:latest
```

> ⚠️ **重要提醒**：SQLite数据库文件存储在 `/app/data` 目录中，必须挂载数据卷或本地目录，否则容器重启会导致所有数据丢失！

### custom分支二开部署

`custom` 分支支持管理员创建用户授权码，并把授权码绑定到指定邮箱。管理员使用 `AUTH_TOKEN` 登录后可以查看和管理全部邮箱；用户使用管理员分配的授权码登录后，只能查看绑定邮箱并取件。管理员还可以在邮箱管理中使用“授权添加”，通过 Microsoft 设备授权首次获取 Outlook RefreshToken 并添加邮箱；如果已有邮箱令牌过期，可以在邮箱行内使用“重新授权”覆盖新的 RefreshToken。

```bash
git clone -b custom https://github.com/GQXXL/outlook-helper.git /opt/outlook-helper
cd /opt/outlook-helper

docker build -t outlook-helper:custom .
mkdir -p /opt/outlook-helper/data /opt/outlook-helper/logs

docker run -d \
  --name outlook-helper \
  --restart unless-stopped \
  -p 8388:8080 \
  -v /opt/outlook-helper/data:/app/data \
  -v /opt/outlook-helper/logs:/app/logs \
  -e APP_PORT=8080 \
  -e APP_ENV=production \
  -e DB_PATH=/app/data/outlook_helper.db \
  -e LOG_FILE=/app/logs/app.log \
  -e AUTH_TOKEN=your-admin-auth-token \
  -e JWT_SECRET=your-jwt-secret \
  -e OUTLOOK_API_BASE_URL=https://outlook2api-gold.vercel.app \
  outlook-helper:custom
```

如果你的 msOauth2api 配置了 `PASSWORD`，在 `docker run` 中额外加入：

```bash
  -e OUTLOOK_API_PASSWORD=your-msoauth2api-password \
```

### 环境变量配置

| 变量名 | 说明 | 默认值                |
|--------|------|--------------------|
| `AUTH_TOKEN` | 授权码（**必须配置**） | 无，必须设置             |
| `OUTLOOK_API_BASE_URL` | Outlook API地址（**必须配置**） | 无，必须设置 |
| `OUTLOOK_API_PASSWORD` | msOauth2api 的 `PASSWORD`，未配置密码时留空 | 空 |
| `EMAIL_VALIDATION_WORKERS` | 令牌验证并发数 | 5                  |

### 📁 数据持久化

**重要：** 本项目使用SQLite数据库，所有数据存储在容器的 `/app/data` 目录中。为防止容器重启导致数据丢失，必须正确配置数据卷挂载。

#### 推荐的数据挂载方式：

1. **使用Docker卷（推荐）**：
   ```bash
   -v outlook_data:/app/data
   ```

2. **使用本地目录挂载**：
   ```bash
   -v $(pwd)/data:/app/data
   ```

3. **生产环境建议**：
   ```bash
   # 创建专门的数据目录
   mkdir -p /opt/outlook-helper/{data,logs}

   # 挂载到固定路径
   -v /opt/outlook-helper/data:/app/data \
   -v /opt/outlook-helper/logs:/app/logs
   ```

#### 数据备份：
```bash
# 备份数据库
docker cp outlook-helper:/app/data/outlook_helper.db ./backup/

# 恢复数据库
docker cp ./backup/outlook_helper.db outlook-helper:/app/data/
```

## 📚 使用指南

### 1. 登录系统
使用配置的AUTH_TOKEN授权码登录系统

### 2. 添加令牌邮箱
支持三种方式添加令牌邮箱：
- **单个添加**：手动输入完整的令牌信息
- **批量添加**：一次性添加多个令牌邮箱（最多30个）
- **文件导入**：上传txt或csv文件批量导入令牌

### 3. 令牌格式说明
标准令牌格式：`邮箱地址----密码----客户端ID----RefreshToken`

**示例：**
```
example@outlook.com----password123----client_id_here----refresh_token_here
user@hotmail.com----mypass456----another_client_id----another_refresh_token
```

### 4. 令牌验证与监控
- 自动验证令牌有效性
- 实时监控邮箱状态
- 标记失效的令牌账户
- 统计令牌成功率

### 5. 标签管理
- 创建自定义标签对令牌邮箱进行分类
- 支持批量标记和取消标记操作
- 标签支持颜色区分，便于管理

### 6. 邮件操作
- 获取最新邮件和全部邮件
- 清空收件箱和垃圾箱
- 支持批量清空操作
- 监控邮件处理状态

## 📊 API文档

### 核心接口

| 方法 | 路径 | 说明 |
|------|------|------|
| `GET` | `/api/health` | 健康检查 |
| `POST` | `/api/auth/login` | 用户登录 |
| `GET` | `/api/emails` | 获取令牌邮箱列表 |
| `POST` | `/api/emails/batch` | 批量添加令牌邮箱 |
| `POST` | `/api/emails/import` | 文件导入令牌邮箱 |
| `GET` | `/api/emails/:id/latest` | 获取最新邮件 |
| `DELETE` | `/api/emails/:id/inbox` | 清空收件箱 |
| `GET` | `/api/tags` | 获取标签列表 |
| `GET` | `/api/dashboard` | 获取仪表盘数据 |



## 🤝 贡献指南

欢迎提交Issue和Pull Request来帮助改进项目！

### 开发环境设置

```bash
# 克隆项目
git clone https://github.com/your-username/outlook-helper.git
cd outlook-helper

# 安装依赖
go mod tidy
cd frontend && npm install && cd ..

# 启动开发服务器
# 后端
go run backend/cmd/main.go

# 前端（新终端）
cd frontend && npm run dev
```


## 📄 许可证

本项目采用 [MIT License](LICENSE) 开源协议。
