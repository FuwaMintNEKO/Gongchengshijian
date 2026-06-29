# 校园二手交易平台

Go + MySQL 构建的校园二手交易平台，支持用户注册登录、商品发布浏览、订单管理等完整交易流程；自带独立部署的 **UAS 统一身份认证平台**，为二手交易应用提供 OAuth2 单点登录。

## 系统组成

| 模块             | 端口  | 目录                       | 说明                                                                 |
| ---------------- | ----- | -------------------------- | -------------------------------------------------------------------- |
| 二手交易应用     | 28080 | `backend/` + `frontend/`   | Go 后端 + 原生 HTML 前端，主应用                                    |
| UAS 后端 API     | 8081  | `uas/backend/`             | Go + Gin，OAuth2 授权、用户管理、JWT 签发，**直接 serve 授权页 HTML** |
| UAS 管理前端     | 8082  | `uas/frontend/`（可选）    | Vue3 + Element Plus，UAS 平台管理后台，仅用于 UAS 管理员操作         |
| MySQL 数据库     | 3306  | -                          | 同时承载 `school_trade` 与 `uas_db` 两个数据库                      |

> **OAuth 流程只需 28080 + 8081 两个服务**，授权页是原生 HTML 由 UAS 后端直接返回，不需要启动 8082 前端服务器。8082 仅在需要进 UAS 管理后台时启动。

## 技术栈

| 层级   | 技术                                       |
| ------ | ------------------------------------------ |
| 前端   | 原生 HTML + CSS + JS / Vue3 + Element Plus  |
| 后端   | Go + Gin 框架                              |
| 数据库 | MySQL 8.0                                  |
| 认证   | OAuth2.0 + JWT + 图形验证码                |
| 部署   | Docker / Nginx / systemd                   |

---

## 完整部署指南（Ubuntu / CentOS）

### 1. 安装依赖

```bash
# 更新系统
sudo apt update && sudo apt upgrade -y        # Ubuntu
# sudo yum update -y                           # CentOS

# 安装 Docker（推荐）
curl -fsSL https://get.docker.com | sudo sh
sudo usermod -aG docker $USER
# 退出重新登录，或执行: newgrp docker

# 安装 Git
sudo apt install -y git                         # Ubuntu
# sudo yum install -y git                       # CentOS

# 安装 Nginx
sudo apt install -y nginx                       # Ubuntu
# sudo yum install -y nginx                     # CentOS
sudo systemctl enable nginx
sudo systemctl start nginx
```

### 2. 克隆项目

```bash
git clone https://github.com/FuwaMintNEKO/Gongchengshijian.git school-trade
cd school-trade
```

### 3. 使用 Docker Compose 启动（一键部署所有服务）

根目录 `docker-compose.yml` 统一编排 3 个服务，**一次启动全部到位**：
- **MySQL** (3306)：同时初始化 `school_trade` 和 `uas_db` 两个数据库（自动导入 UAS 表结构）
- **校园二手交易应用** (28080)：Go 后端 + 原生 HTML 前端
- **UAS 统一认证后端** (8081)：OAuth2 授权页 + 用户管理 API

```bash
# 首次部署 / 更新代码后，必须加 --build 重新构建镜像（否则用的是旧代码镜像）
docker compose up -d --build

# 查看所有服务状态
docker compose ps

# 查看应用日志
docker compose logs -f app

# 验证健康检查
curl http://localhost:28080/health
# 预期: {"database":true,"status":"ok","time":"..."}

curl http://localhost:8081/api/health
# 预期: {"database":true,"status":"ok","time":"..."}
```

> **重要**：更新代码后必须用 `docker compose up -d --build` 重新构建镜像，否则容器跑的还是旧代码（常见问题：登录页看不到 UAS 按钮、OAuth 报错等，都是因为没重新 build）。

> 无需手动安装 MySQL！`docker-compose.yml` 会自动：
> - 创建 MySQL 8.0 容器
> - 创建 `school_trade` 空库（应用启动时自动建表+示例数据）
> - 导入 `uas/docs/init_tables.sql` 创建 `uas_db` 库 + 表结构 + 默认数据（admin/admin123、示例应用等）
> - 配置 school-trade 与 UAS 容器间网络互通（`UAS_BASE_URL=http://uas-backend:8081`）
> - 自动填入 OAuth ClientSecret（`a15df289schooltradebf6cfbda`）

**环境变量说明（可选）：**

| 变量         | 默认值     | 说明           |
| ------------ | ---------- | -------------- |
| `PORT`       | `28080`    | 应用端口       |
| `DB_PASSWORD`| `114514`   | MySQL 密码     |
| `DB_NAME`    | `school_trade` | 数据库名   |
| `JWT_SECRET` | `uas-secret-key-2026-school-trade` | UAS JWT 密钥 |

```bash
# 自定义启动示例
DB_PASSWORD=MySecurePwd docker compose up -d --build
```

### 4. 配置 Nginx 反向代理

创建 Nginx 配置文件：

```bash
sudo nano /etc/nginx/sites-available/school-trade
# 或 Ubuntu:  /etc/nginx/conf.d/school-trade.conf
```

写入以下内容：

```nginx
server {
    listen 80;
    server_name your-domain.com;  # 改成你的域名或服务器 IP

    # 后端 API + 静态文件（由 Go 服务统一处理）
    location / {
        proxy_pass http://127.0.0.1:28080;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;

        # WebSocket 支持（如果需要）
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection "upgrade";

        # 超时设置
        proxy_connect_timeout 60s;
        proxy_read_timeout 60s;
        proxy_send_timeout 60s;
    }
}
```

启用并重载 Nginx：

```bash
# Ubuntu
sudo ln -s /etc/nginx/sites-available/school-trade /etc/nginx/sites-enabled/
sudo nginx -t          # 测试配置是否正确
sudo systemctl reload nginx   # 重载 Nginx

# CentOS
sudo nginx -t
sudo systemctl reload nginx
```

### 5. 开放防火墙端口

```bash
# Ubuntu (ufw)
sudo ufw allow 80/tcp
sudo ufw allow 22/tcp
sudo ufw --force enable

# CentOS (firewalld)
sudo firewall-cmd --permanent --add-port=80/tcp
sudo firewall-cmd --reload
```

### 6. 访问验证

- 浏览器访问 `http://your-domain.com` 或 `http://服务器IP`
- 默认管理员账号：`admin` / `admin123`
- 测试用户：`alice` / `alice123`、`bob` / `bob123`

### 7. 配置 HTTPS（推荐）

使用 Let's Encrypt 免费证书：

```bash
sudo apt install -y certbot python3-certbot-nginx   # Ubuntu
# sudo yum install -y certbot python3-certbot-nginx # CentOS

sudo certbot --nginx -d your-domain.com
```

Certbot 会自动修改 Nginx 配置并开启 HTTPS。

### 8. 日常维护

```bash
# 查看服务状态
docker compose ps

# 查看应用日志
docker compose logs -f app      # 二手交易应用
docker compose logs -f uas-backend  # UAS 认证后端

# 重启单个服务
docker compose restart app
docker compose restart uas-backend

# 更新代码后重新部署（必须 --build，否则用旧镜像）
git pull
docker compose up -d --build

# 停止所有服务
docker compose down

# 备份数据库
docker compose exec -T mysql mysqldump -uroot -p114514 school_trade > backup.sql
```

---

## 非 Docker 手动部署

如果不使用 Docker，需自行安装 MySQL 8.0+ 和 Go 1.24+，并配置环境变量：

```bash
# 设置数据库环境变量
export DB_HOST=127.0.0.1
export DB_PORT=3306
export DB_USER=root
export DB_PASSWORD=你的密码
export DB_NAME=school_trade
export PORT=28080

# 手动创建数据库
mysql -uroot -p -e "CREATE DATABASE IF NOT EXISTS school_trade DEFAULT CHARSET utf8mb4;"

# 编译并运行
cd backend
go build -o school-trade .
./school-trade
```

---

## UAS 统一身份认证平台部署

> **如果你已用根目录 `docker compose up -d --build` 部署**，UAS 后端已包含在内（`uas-backend` 服务），无需再单独部署 UAS。本章节仅用于需要**独立部署 UAS** 的场景。

UAS 是独立部署的 OAuth2.0 认证平台，为二手交易应用提供单点登录。

> **重要**：UAS 后端已 Docker 化，在 Linux 服务器上通过 Docker 自动编译为 Linux 二进制运行。**不要使用仓库中的 `uas-server.exe`（Windows 二进制，Linux 无法运行）**。

### 1. 使用 Docker Compose 启动 UAS（推荐）

```bash
cd uas
docker compose up -d

# 查看日志
docker compose logs -f uas-backend

# 验证健康检查
curl http://localhost:8081/api/health
# 预期: {"database":true,"status":"ok","time":"..."}
```

`uas/docker-compose.yml` 会自动：
- 启动 MySQL 8.0 容器（端口 3307，避免与 school-trade 的 3306 冲突）
- 自动导入 `uas/docs/init_tables.sql` 初始化表结构和默认数据
- 编译并启动 UAS 后端（端口 8081）

**UAS 环境变量（可选，通过 docker compose 覆盖）：**

| 变量                  | 默认值                              | 说明                    |
| --------------------- | ----------------------------------- | ----------------------- |
| `UAS_PORT`            | `8081`                              | UAS 后端端口            |
| `DB_HOST`             | `mysql`（容器内）                   | MySQL 主机              |
| `DB_PORT`             | `3306`                              | MySQL 端口              |
| `DB_USER`             | `root`                              | MySQL 用户名            |
| `DB_PASSWORD`         | `114514`                            | MySQL 密码              |
| `DB_NAME`             | `uas_db`                            | UAS 数据库名            |
| `JWT_SECRET`          | `uas-secret-key-2026-school-trade`  | JWT 签名密钥            |
| `JWT_EXPIRE_HOURS`    | `24`                                | Token 有效期（小时）    |
| `OAUTH_CODE_EXPIRE`   | `300`                               | 授权码有效期（秒）      |
| `OAUTH_TOKEN_EXPIRE`  | `604800`                            | OAuth Token 有效期（秒）|

默认管理员账号：`admin` / `admin123`（首次登录后请修改密码）

### 2. 手动部署（不使用 Docker）

需自行安装 MySQL 8.0+ 和 Go 1.24+：

```bash
# 创建 UAS 数据库并导入初始数据
mysql -uroot -p < uas/docs/init_tables.sql

# 编译并运行（Linux 服务器会生成 Linux 二进制）
cd uas/backend
export DB_HOST=127.0.0.1
export DB_PORT=3306
export DB_USER=root
export DB_PASSWORD=你的密码
export DB_NAME=uas_db
export UAS_PORT=8081
export JWT_SECRET=uas-secret-key-2026-school-trade
go build -o uas-server .
./uas-server
```

### 3. 启动 UAS 管理前端（端口 8082，可选）

> **OAuth 授权流程不需要此步**。授权页 `uas/backend/public/oauth/authorize.html` 是原生 HTML，已由 UAS 后端 8081 直接 serve。
> 仅当需要使用 UAS 管理后台（用户管理、应用管理、统计等）时才启动 8082 前端。

```bash
cd uas/frontend

# 安装依赖（首次部署需要 Node.js 18+）
npm install

# 生产构建
npm run build
# 构建产物在 dist/ 目录

# 用 vite preview 启动静态服务（端口 8082）
nohup npx vite preview --port 8082 --host 0.0.0.0 > uas-frontend.log 2>&1 &

# 或用 Nginx 托管 dist/ 静态文件
```

**Nginx 托管 UAS 管理前端（推荐）：**

```nginx
server {
    listen 8082;
    server_name _;

    root /home/ubuntu/school-trade/uas/frontend/dist;
    index index.html;

    location / {
        try_files $uri $uri/ /index.html;
    }

    # API 反代到 UAS 后端
    location /api/ {
        proxy_pass http://127.0.0.1:8081;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
    }

    location /uploads/ {
        proxy_pass http://127.0.0.1:8081;
    }
}
```

### 4. 二手交易应用接入 UAS

UAS 初始化时已自动注册「校园二手交易平台」应用（AppID: `KK790SCHOOLTRADE`），数据库中注册的回调地址 path 为 `/oauth/callback`。

**回调地址校验策略**：UAS 后端只校验 redirect_uri 的 path 部分，允许 host:port 变化，因此本地、云服务器（同 IP 不同端口）均可直接工作，无需手动改 redirect_uri。

**OAuth 跳转地址自动推断**：二手交易应用 `backend/handlers/oauth.go` 会从当前请求的 Host 自动推断：
- UAS 授权页地址：`http://服务器IP:8081/oauth/authorize`（同 IP + UAS 后端端口）
- 回调地址：`http://服务器IP:28080/oauth/callback`（同 IP + 当前请求端口）

如需固定地址，可通过环境变量覆盖：

| 环境变量           | 默认（空时自动推断）                | 说明                          |
| ------------------ | ----------------------------------- | ----------------------------- |
| `UAS_BASE_URL`     | `http://localhost:8081`             | UAS 后端 API 地址             |
| `UAS_FRONTEND_URL` | 空（推断为 `同IP:8081`）            | UAS 授权页地址                |
| `UAS_REDIRECT_URI` | 空（推断为 `同IP:当前端口/oauth/callback`） | OAuth 回调地址          |
| `UAS_CLIENT_ID`    | `KK790SCHOOLTRADE`                  | 应用 AppID                    |
| `UAS_CLIENT_SECRET`| 空（不填则 OAuth 按钮置灰）         | 应用 AppSecret                |

### 6. 访问入口

| 入口                      | 地址                                  |
| ------------------------- | ------------------------------------- |
| 二手交易应用首页          | `http://服务器IP:28080/`              |
| 二手交易应用登录页        | `http://服务器IP:28080/pages/login.html` |
| UAS 管理后台（可选）      | `http://服务器IP:8082/`               |
| UAS 后端 API              | `http://服务器IP:8081/api/`           |
| UAS 授权页（OAuth2 入口） | `http://服务器IP:8081/oauth/authorize`，由二手交易应用登录页自动跳转 |

### 6. UAS 日常维护

```bash
cd uas

# 查看服务状态
docker compose ps

# 查看后端日志
docker compose logs -f uas-backend

# 重启 UAS 后端
docker compose restart uas-backend

# 更新 UAS 代码后重新部署（Docker 会自动重新编译为 Linux 二进制）
cd school-trade
git pull
cd uas
docker compose build uas-backend
docker compose up -d

# 授权页是原生 HTML（uas/backend/public/oauth/authorize.html），由后端直接 serve，无需构建
# 仅 UAS 管理前端（8082）需构建：
cd frontend && npm run build  # 前端重新构建即可，无需重启 Nginx

# 停止所有 UAS 服务
docker compose down

# 备份 UAS 数据库
docker compose exec -T mysql mysqldump -uroot -p114514 uas_db > uas_backup.sql
```

## 项目结构

```
school-trade/
├── backend/              # 二手交易应用 Go 后端
│   ├── handlers/         # 接口处理（含 oauth.go 对接 UAS）
│   ├── middleware/       # JWT 认证中间件
│   ├── models/           # 数据模型
│   └── store/            # 数据库层
├── frontend/             # 二手交易应用前端页面
│   ├── css/
│   ├── js/
│   └── pages/
├── uas/                  # UAS 统一身份认证平台
│   ├── backend/          # UAS Go 后端
│   │   ├── handlers/     # OAuth2、用户、应用、菜单等接口
│   │   ├── middleware/   # JWT 鉴权中间件
│   │   ├── models/       # UAS 数据模型
│   │   ├── config/       # 环境变量配置
│   │   ├── public/       # 静态资源（OAuth 授权页 HTML 由后端直接 serve）
│   │   │   └── oauth/authorize.html
│   │   ├── utils/        # 工具函数
│   │   ├── Dockerfile    # UAS 后端 Docker 构建（Linux 二进制）
│   │   └── .dockerignore
│   ├── frontend/         # UAS Vue3 管理前端
│   │   ├── src/
│   │   │   ├── api/      # 接口封装
│   │   │   ├── layout/   # 后台布局
│   │   │   ├── router/   # 路由配置
│   │   │   └── views/    # 页面（dashboard/system/user/app/...）
│   │   └── vite.config.js
│   ├── docs/
│   │   └── init_tables.sql  # UAS 数据库初始化脚本（docker compose 自动导入）
│   └── docker-compose.yml   # UAS 后端 + MySQL 编排
├── diagrams/             # 项目架构图、ER图、甘特图等
├── docker-compose.yml    # 二手交易应用 Docker 编排
├── Dockerfile            # 二手交易应用 Dockerfile
└── README.md
```
