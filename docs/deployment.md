# 部署方案文档

## 1. Docker Compose部署

### 1.1 目录结构

```
meeting/
├── docker/
│   ├── docker-compose.yml
│   ├── docker-compose.prod.yml
│   ├── .env
│   ├── nginx/
│   │   ├── nginx.conf
│   │   └── ssl/
│   │       ├── cert.pem
│   │       └── key.pem
│   ├── client/
│   │   └── Dockerfile
│   ├── api/
│   │   └── Dockerfile
│   ├── signaling/
│   │   └── Dockerfile
│   ├── sfu/
│   │   └── Dockerfile
│   └── recording/
│       └── Dockerfile
```

### 1.2 docker-compose.yml

```yaml
version: '3.8'

services:
  # Nginx反向代理
  nginx:
    image: nginx:alpine
    ports:
      - "80:80"
      - "443:443"
    volumes:
      - ./nginx/nginx.conf:/etc/nginx/nginx.conf:ro
      - ./nginx/ssl:/etc/nginx/ssl:ro
    depends_on:
      - client
      - api
      - signaling
      - sfu
    networks:
      - meeting-network

  # 前端静态文件服务
  client:
    build:
      context: ../client
      dockerfile: ../docker/client/Dockerfile
    ports:
      - "3000:80"
    networks:
      - meeting-network

  # 业务API服务器
  api:
    build:
      context: ../server/api
      dockerfile: ../../docker/api/Dockerfile
    environment:
      - NODE_ENV=production
      - PORT=3001
      - MONGODB_URI=mongodb://mongodb:27017/meeting
      - REDIS_URL=redis://redis:6379
      - JWT_SECRET=${JWT_SECRET}
      - MINIO_ENDPOINT=minio
      - MINIO_PORT=9000
      - MINIO_ACCESS_KEY=${MINIO_ACCESS_KEY}
      - MINIO_SECRET_KEY=${MINIO_SECRET_KEY}
    depends_on:
      - mongodb
      - redis
      - minio
    networks:
      - meeting-network
    restart: unless-stopped

  # 信令服务器
  signaling:
    build:
      context: ../server/signaling
      dockerfile: ../../docker/signaling/Dockerfile
    environment:
      - NODE_ENV=production
      - PORT=3002
      - REDIS_URL=redis://redis:6379
      - SFU_URL=http://sfu:3003
      - JWT_SECRET=${JWT_SECRET}
    depends_on:
      - redis
      - sfu
    networks:
      - meeting-network
    restart: unless-stopped

  # SFU媒体服务器
  sfu:
    build:
      context: ../server/sfu
      dockerfile: ../../docker/sfu/Dockerfile
    environment:
      - NODE_ENV=production
      - PORT=3003
      - ANNOUNCED_IP=${SERVER_PUBLIC_IP}
    ports:
      - "10000-59999:10000-59999/udp"  # WebRTC UDP端口
    networks:
      - meeting-network
    restart: unless-stopped

  # 录制服务
  recording:
    build:
      context: ../server/recording
      dockerfile: ../../docker/recording/Dockerfile
    environment:
      - NODE_ENV=production
      - PORT=3004
      - SFU_URL=http://sfu:3003
      - MINIO_ENDPOINT=minio
      - MINIO_PORT=9000
      - MINIO_ACCESS_KEY=${MINIO_ACCESS_KEY}
      - MINIO_SECRET_KEY=${MINIO_SECRET_KEY}
    depends_on:
      - sfu
      - minio
    volumes:
      - recording-data:/tmp/recording
    networks:
      - meeting-network
    restart: unless-stopped

  # MongoDB数据库
  mongodb:
    image: mongo:7
    ports:
      - "27017:27017"
    volumes:
      - mongodb-data:/data/db
    networks:
      - meeting-network
    restart: unless-stopped

  # Redis缓存
  redis:
    image: redis:7-alpine
    ports:
      - "6379:6379"
    volumes:
      - redis-data:/data
    command: redis-server --appendonly yes
    networks:
      - meeting-network
    restart: unless-stopped

  # MinIO对象存储
  minio:
    image: minio/minio
    ports:
      - "9000:9000"
      - "9001:9001"
    environment:
      - MINIO_ROOT_USER=${MINIO_ACCESS_KEY}
      - MINIO_ROOT_PASSWORD=${MINIO_SECRET_KEY}
    volumes:
      - minio-data:/data
    command: server /data --console-address ":9001"
    networks:
      - meeting-network
    restart: unless-stopped

volumes:
  mongodb-data:
  redis-data:
  minio-data:
  recording-data:

networks:
  meeting-network:
    driver: bridge
```

### 1.3 Nginx配置

```nginx
# nginx/nginx.conf
events {
    worker_connections 1024;
}

http {
    upstream api_backend {
        server api:3001;
    }
    
    upstream signaling_backend {
        server signaling:3002;
    }
    
    upstream client_backend {
        server client:80;
    }
    
    # API服务
    server {
        listen 80;
        server_name api.meeting.example.com;
        
        location / {
            proxy_pass http://api_backend;
            proxy_set_header Host $host;
            proxy_set_header X-Real-IP $remote_addr;
            proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
            proxy_set_header X-Forwarded-Proto $scheme;
        }
    }
    
    # 信令服务（WebSocket）
    server {
        listen 80;
        server_name signaling.meeting.example.com;
        
        location / {
            proxy_pass http://signaling_backend;
            proxy_http_version 1.1;
            proxy_set_header Upgrade $http_upgrade;
            proxy_set_header Connection "upgrade";
            proxy_set_header Host $host;
            proxy_set_header X-Real-IP $remote_addr;
            proxy_read_timeout 86400;
        }
    }
    
    # SFU服务（直连，用于WebRTC）
    server {
        listen 80;
        server_name sfu.meeting.example.com;
        
        location / {
            proxy_pass http://sfu:3003;
            proxy_set_header Host $host;
            proxy_set_header X-Real-IP $remote_addr;
        }
    }
    
    # 前端静态文件
    server {
        listen 80;
        server_name meeting.example.com;
        
        location / {
            proxy_pass http://client_backend;
            proxy_set_header Host $host;
        }
    }
}

# RTMP/WebRTC不需要在nginx中配置，直接连接SFU
```

### 1.4 Dockerfile示例

```dockerfile
# docker/sfu/Dockerfile
FROM node:20-alpine

# 安装FFmpeg（录制需要）
RUN apk add --no-cache ffmpeg

WORKDIR /app

# 复制依赖文件
COPY package*.json ./
RUN npm ci --only=production

# 复制源代码
COPY dist/ ./dist/

# 创建非root用户
RUN addgroup -g 1001 -S nodejs
RUN adduser -S mediasoup -u 1001
USER mediasoup

EXPOSE 3003

CMD ["node", "dist/index.js"]
```

## 2. 生产环境配置

### 2.1 环境变量

```bash
# .env
# JWT密钥
JWT_SECRET=your-super-secret-jwt-key-here
JWT_REFRESH_SECRET=your-refresh-secret-key-here

# 服务器公网IP（WebRTC需要）
SERVER_PUBLIC_IP=your.server.public.ip

# MinIO配置
MINIO_ACCESS_KEY=minioadmin
MINIO_SECRET_KEY=minioadmin

# MongoDB
MONGODB_URI=mongodb://mongodb:27017/meeting

# Redis
REDIS_URL=redis://redis:6379

# 客户端URL
CLIENT_URL=https://meeting.example.com
```

### 2.2 SSL证书配置

```bash
# 使用Let's Encrypt获取证书
sudo certbot certonly --standalone -d meeting.example.com -d api.meeting.example.com -d signaling.meeting.example.com

# 或使用Nginx插件
sudo certbot --nginx -d meeting.example.com
```

## 3. 性能优化配置

### 3.1 系统参数优化

```bash
# /etc/sysctl.conf
# 增加UDP缓冲区（WebRTC）
net.core.rmem_max=16777216
net.core.wmem_max=16777216
net.ipv4.udp_mem=4096 87380 16777216

# 增加文件描述符
fs.file-max=100000

# 网络优化
net.core.somaxconn=65535
net.ipv4.tcp_max_syn_backlog=65535
net.ipv4.ip_local_port_range=1024 65535
```

```bash
# /etc/security/limits.conf
* soft nofile 65535
* hard nofile 65535
* soft nproc 65535
* hard nproc 65535
```

### 3.2 Node.js优化

```bash
# PM2配置 (ecosystem.config.js)
module.exports = {
  apps: [
    {
      name: 'api',
      script: './server/api/dist/index.js',
      instances: 4,  // 根据CPU核心数
      exec_mode: 'cluster',
      env: {
        NODE_ENV: 'production',
        NODE_OPTIONS: '--max-old-space-size=2048'
      }
    },
    {
      name: 'signaling',
      script: './server/signaling/dist/index.js',
      instances: 4,
      exec_mode: 'cluster',
      env: {
        NODE_ENV: 'production',
        NODE_OPTIONS: '--max-old-space-size=2048'
      }
    }
  ]
};
```

## 4. 监控方案

### 4.1 推荐监控栈

```yaml
# docker-compose.monitoring.yml
version: '3.8'

services:
  # Prometheus
  prometheus:
    image: prom/prometheus
    ports:
      - "9090:9090"
    volumes:
      - ./prometheus.yml:/etc/prometheus/prometheus.yml

  # Grafana
  grafana:
    image: grafana/grafana
    ports:
      - "3005:3000"
    environment:
      - GF_SECURITY_ADMIN_PASSWORD=admin
    volumes:
      - grafana-data:/var/lib/grafana

  # Node Exporter
  node-exporter:
    image: prom/node-exporter
    ports:
      - "9100:9100"

volumes:
  grafana-data:
```

### 4.2 关键监控指标

| 指标 | 说明 | 告警阈值 |
|------|------|---------|
| CPU使用率 | 服务器CPU | > 80% |
| 内存使用率 | 服务器内存 | > 85% |
| 带宽使用 | 网络带宽 | > 80% |
| 在线会议数 | 活跃会议 | 根据容量 |
| 参与者数量 | 总参与者 | > 80%容量 |
| WebRTC连接数 | 活跃连接 | 根据配置 |
| API响应时间 | 接口延迟 | > 500ms |
| 错误率 | 错误请求比例 | > 1% |

## 5. 扩展部署

### 5.1 多实例SFU部署

```yaml
# 多个SFU实例，通过负载均衡分配
sfu-1:
  build:
    context: ../server/sfu
    dockerfile: ../../docker/sfu/Dockerfile
  environment:
    - ANNOUNCED_IP=${SERVER_PUBLIC_IP_1}
    - RTC_MIN_PORT=10000
    - RTC_MAX_PORT=29999
  networks:
    - meeting-network

sfu-2:
  build:
    context: ../server/sfu
    dockerfile: ../../docker/sfu/Dockerfile
  environment:
    - ANNOUNCED_IP=${SERVER_PUBLIC_IP_2}
    - RTC_MIN_PORT=30000
    - RTC_MAX_PORT=59999
  networks:
    - meeting-network
```

### 5.2 Socket.IO多实例

```typescript
// 使用Redis Adapter实现多实例
import { createAdapter } from '@socket.io/redis-adapter';
import { createClient } from 'redis';

const pubClient = createClient({ url: 'redis://redis:6379' });
const subClient = pubClient.duplicate();

io.adapter(createAdapter(pubClient, subClient));
```

## 6. 部署步骤

```bash
# 1. 克隆代码
git clone https://github.com/your-repo/meeting.git
cd meeting

# 2. 配置环境变量
cp docker/.env.example docker/.env
vim docker/.env

# 3. 构建镜像
cd docker
docker-compose build

# 4. 启动服务
docker-compose up -d

# 5. 检查服务状态
docker-compose ps
docker-compose logs -f

# 6. 初始化MinIO存储桶
docker exec -it meeting-minio-1 mc mb /data/recordings
docker exec -it meeting-minio-1 mc anonymous set download /data/recordings

# 7. 验证服务
curl http://localhost/api/health
curl http://localhost:3003/health
```

## 7. 常见问题

### 7.1 WebRTC连接失败

- 检查服务器防火墙UDP端口（10000-59999）
- 确认ANNOUNCED_IP配置正确（公网IP）
- 检查STUN/TURN服务器配置

### 7.2 性能问题

- 增加SFU Worker数量（根据CPU核心数）
- 启用视频Simulcast降低带宽
- 限制同时订阅的视频流数量

### 7.3 录制失败

- 检查FFmpeg是否正确安装
- 确认MinIO存储桶权限
- 检查磁盘空间
