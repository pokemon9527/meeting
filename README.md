# 网页视频会议系统 (Meeting Web)

支持50-200人大型会议的网页视频会议系统，采用SFU架构实现多人音视频通话，后端使用Go语言重构，媒体服务基于Pion WebRTC。

## 技术栈

| 模块 | 技术 |
|------|------|
| 前端 | React 18 + TypeScript + Vite + Ant Design + Zustand |
| 后端API | Go + Gin |
| 信令 | Go + Gorilla WebSocket |
| 媒体SFU | Go + Pion WebRTC |
| 数据库 | PostgreSQL + GORM |
| 缓存 | Redis |
| 部署 | Docker + Nginx |

## 项目结构

```
meeting/
├── client/                        # 前端 React 应用
├── cmd/                           # Go 服务入口
│   ├── api/                       # API 服务 (Gin)
│   ├── sfu/                       # SFU 媒体服务 (Pion)
│   └── signaling/                 # 信令服务
├── internal/                      # 内部包
│   ├── config/                    # 配置
│   ├── database/                  # 数据库连接
│   ├── models/                    # 数据模型
│   ├── handlers/                  # HTTP/WS 处理器
│   ├── middleware/                # 中间件
│   └── webrtc/                    # WebRTC 相关
├── pkg/                           # 公共包
│   ├── jwt/                       # JWT 工具
│   └── response/                  # 响应封装
├── docker/                        # Docker 配置
│   ├── Dockerfile.api
│   ├── Dockerfile.sfu
│   ├── Dockerfile.client
│   └── nginx.conf
├── docker-compose.yml
├── go.mod
└── go.sum
```

## 快速开始

### Docker 部署（推荐）

```bash
# 1. 克隆项目
cd meeting

# 2. 设置服务器公网IP（用于WebRTC）
export SERVER_PUBLIC_IP=your.server.ip

# 3. 构建并启动所有服务
docker-compose up -d

# 4. 查看服务状态
docker-compose ps

# 5. 查看日志
docker-compose logs -f

# 6. 访问应用
# 打开浏览器访问 http://localhost
```

### 本地开发

#### 前置要求
- Go 1.22+
- Node.js 18+
- PostgreSQL 16+
- Redis 7+

#### 启动步骤

```bash
# 1. 启动数据库
docker run -d --name postgres -e POSTGRES_DB=meeting -e POSTGRES_PASSWORD=postgres -p 5432:5432 postgres:16-alpine
docker run -d --name redis -p 6379:6379 redis:7-alpine

# 2. 启动 Go 后端
export POSTGRES_DSN="host=localhost user=postgres password=postgres dbname=meeting port=5432 sslmode=disable"
go run ./cmd/api
# API 服务运行在 http://localhost:8080

# 3. 启动 SFU 服务
go run ./cmd/sfu
# SFU 服务运行在 http://localhost:8082

# 4. 启动前端
cd client
npm install
npm run dev
# 前端运行在 http://localhost:5173
```

## API 接口

### 认证

| 方法 | 路径 | 说明 |
|------|------|------|
| POST | /api/auth/register | 用户注册 |
| POST | /api/auth/login | 用户登录 |
| POST | /api/auth/refresh | 刷新Token |
| GET  | /api/auth/me | 获取当前用户 |

### 会议

| 方法 | 路径 | 说明 |
|------|------|------|
| POST   | /api/meetings | 创建会议 |
| GET    | /api/meetings | 我的会议列表 |
| GET    | /api/meetings/:id | 会议详情 |
| POST   | /api/meetings/join | 加入会议 |
| PUT    | /api/meetings/:id/settings | 更新设置 |
| POST   | /api/meetings/:id/end | 结束会议 |
| DELETE | /api/meetings/:id | 删除会议 |

### WebSocket

连接地址: `ws://localhost/ws?meeting_id=xxx&user_id=xxx&username=xxx`

事件类型:
- `join-room` - 加入房间
- `leave-room` - 离开房间
- `peer-joined` - 参与者加入
- `peer-left` - 参与者离开
- `toggle-audio` - 切换音频
- `toggle-video` - 切换视频
- `send-message` - 发送消息
- `raise-hand` - 举手

## 配置说明

### 环境变量

| 变量 | 默认值 | 说明 |
|------|--------|------|
| API_PORT | 8080 | API 服务端口 |
| SFU_PORT | 8082 | SFU 服务端口 |
| POSTGRES_DSN | - | PostgreSQL 连接串 |
| REDIS_ADDR | localhost:6379 | Redis 地址 |
| JWT_SECRET | - | JWT 密钥 |
| JWT_REFRESH_SECRET | - | JWT 刷新密钥 |
| PUBLIC_IP | 127.0.0.1 | 服务器公网IP |

### Docker Compose 环境变量

```bash
# .env 文件
SERVER_PUBLIC_IP=your.server.ip
JWT_SECRET=your-secret-key
JWT_REFRESH_SECRET=your-refresh-secret
```

## 使用说明

1. **注册/登录**：访问首页，注册新账号或登录
2. **创建会议**：点击"创建会议"，设置会议主题和密码（可选）
3. **加入会议**：点击"加入会议"，输入6位会议号
4. **会议功能**：
   - 静音/取消静音
   - 开启/关闭摄像头
   - 屏幕共享
   - 实时聊天
   - 参与者管理
   - 举手功能

## 常见问题

### 1. WebRTC 连接失败
- 确保 `PUBLIC_IP` 设置正确
- 检查防火墙是否开放 UDP 50000-50100 端口

### 2. 编译失败
```bash
# 清理并重新下载依赖
go clean -modcache
go mod download
```

### 3. 数据库连接失败
- 确认 PostgreSQL 已启动
- 检查连接串配置

## 许可证

MIT
