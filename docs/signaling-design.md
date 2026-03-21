# 信令服务器详细设计

## 1. 项目结构

```
server/signaling/
├── src/
│   ├── config/
│   │   ├── index.ts              # 配置入口
│   │   ├── redis.ts              # Redis配置
│   │   └── cors.ts               # CORS配置
│   │
│   ├── handlers/
│   │   ├── index.ts              # Handler注册
│   │   ├── room.handler.ts       # 房间事件处理
│   │   ├── transport.handler.ts  # WebRTC传输处理
│   │   ├── producer.handler.ts   # Producer处理
│   │   ├── consumer.handler.ts   # Consumer处理
│   │   ├── chat.handler.ts       # 聊天事件处理
│   │   ├── whiteboard.handler.ts # 白板事件处理
│   │   └── control.handler.ts    # 会议控制处理
│   │
│   ├── services/
│   │   ├── room.service.ts       # 房间服务
│   │   ├── sfu.service.ts        # SFU通信服务
│   │   ├── redis.service.ts      # Redis服务
│   │   └── chat.service.ts       # 聊天服务
│   │
│   ├── middlewares/
│   │   ├── auth.middleware.ts     # 认证中间件
│   │   ├── rateLimit.middleware.ts# 限流中间件
│   │   └── logger.middleware.ts   # 日志中间件
│   │
│   ├── types/
│   │   ├── socket.ts             # Socket类型定义
│   │   ├── room.ts               # 房间类型
│   │   └── peer.ts               # 参与者类型
│   │
│   ├── utils/
│   │   ├── validator.ts          # 数据验证
│   │   └── helpers.ts            # 工具函数
│   │
│   └── index.ts                  # 入口文件
│
├── package.json
└── tsconfig.json
```

## 2. 核心类设计

### 2.1 Room类

```typescript
// services/room.service.ts
class Room {
  readonly id: string;
  readonly meetingId: string;
  private peers: Map<string, Peer>;
  private sfu: SFUService;
  
  constructor(id: string, meetingId: string, sfu: SFUService);
  
  // 参与者管理
  addPeer(peer: Peer): Promise<void>;
  removePeer(peerId: string): void;
  getPeer(peerId: string): Peer | undefined;
  getAllPeers(): Peer[];
  getPeerCount(): number;
  
  // 权限管理
  isHost(peerId: string): boolean;
  isCohost(peerId: string): boolean;
  canProduce(peerId: string): boolean;
  
  // 状态
  toJSON(): RoomInfo;
}

interface RoomInfo {
  id: string;
  meetingId: string;
  peerCount: number;
  peers: PeerInfo[];
  isActive: boolean;
  createdAt: Date;
}
```

### 2.2 Peer类

```typescript
// types/peer.ts
class Peer {
  readonly id: string;
  readonly socketId: string;
  readonly userId: string;
  username: string;
  avatar: string;
  role: 'host' | 'cohost' | 'participant';
  
  // SFU相关
  private sendTransport: Transport | null;
  private recvTransport: Transport | null;
  private producers: Map<string, Producer>;
  private consumers: Map<string, Consumer>;
  
  // 媒体状态
  audioEnabled: boolean;
  videoEnabled: boolean;
  isScreenSharing: boolean;
  isHandRaised: boolean;
  
  constructor(params: PeerParams);
  
  // Transport管理
  setSendTransport(transport: Transport): void;
  setRecvTransport(transport: Transport): void;
  
  // Producer管理
  addProducer(producer: Producer): void;
  removeProducer(producerId: string): void;
  getProducers(): Producer[];
  
  // Consumer管理
  addConsumer(consumer: Consumer): void;
  removeConsumer(consumerId: string): void;
  getConsumers(): Consumer[];
  
  toJSON(): PeerInfo;
}
```

## 3. Socket事件处理

### 3.1 房间事件

```typescript
// handlers/room.handler.ts
export function registerRoomHandlers(socket: Socket, io: Server) {
  
  // 加入房间
  socket.on('join-room', async (data: JoinRoomData, callback) => {
    // 1. 验证token和会议权限
    // 2. 检查等候室设置
    // 3. 创建Peer对象
    // 4. 加入Socket Room
    // 5. 通知其他参与者
    // 6. 返回房间信息
  });
  
  // 离开房间
  socket.on('leave-room', async () => {
    // 1. 清理SFU资源
    // 2. 通知其他参与者
    // 3. 更新Redis状态
    // 4. 删除Peer对象
  });
  
  // 断开连接处理
  socket.on('disconnect', async () => {
    // 同leave-room逻辑
  });
}

interface JoinRoomData {
  meetingId: string;
  password?: string;
  userId: string;
  username: string;
  avatar: string;
}
```

### 3.2 WebRTC信令事件

```typescript
// handlers/transport.handler.ts
export function registerTransportHandlers(socket: Socket, io: Server, sfu: SFUService) {
  
  // 获取Router RTP Capabilities
  socket.on('get-router-rtp-capabilities', async (meetingId, callback) => {
    const capabilities = await sfu.getRouterRtpCapabilities(meetingId);
    callback({ capabilities });
  });
  
  // 创建Transport
  socket.on('create-transport', async (data: CreateTransportData, callback) => {
    const { direction, meetingId } = data;
    // 调用SFU创建Transport
    const transportParams = await sfu.createTransport(meetingId, direction);
    // 保存Transport关联
    callback(transportParams);
  });
  
  // 连接Transport
  socket.on('connect-transport', async (data: ConnectTransportData, callback) => {
    const { transportId, dtlsParameters } = data;
    await sfu.connectTransport(transportId, dtlsParameters);
    callback({ connected: true });
  });
}

interface CreateTransportData {
  meetingId: string;
  direction: 'send' | 'recv';
}

interface ConnectTransportData {
  transportId: string;
  dtlsParameters: any;
}
```

### 3.3 Producer/Consumer事件

```typescript
// handlers/producer.handler.ts
export function registerProducerHandlers(socket: Socket, io: Server, sfu: SFUService) {
  
  // 创建Producer
  socket.on('produce', async (data: ProduceData, callback) => {
    const { transportId, kind, rtpParameters, appData } = data;
    
    // 在SFU上创建Producer
    const producer = await sfu.produce(transportId, {
      kind,
      rtpParameters,
      appData
    });
    
    // 通知房间其他参与者有新的Producer
    socket.to(data.meetingId).emit('new-producer', {
      producerId: producer.id,
      peerId: socket.data.peerId,
      kind,
      appData
    });
    
    callback({ id: producer.id });
  });
  
  // 关闭Producer
  socket.on('close-producer', async (data: { producerId: string }) => {
    await sfu.closeProducer(data.producerId);
    socket.to(socket.data.meetingId).emit('producer-closed', {
      producerId: data.producerId
    });
  });
}

// handlers/consumer.handler.ts
export function registerConsumerHandlers(socket: Socket, io: Server, sfu: SFUService) {
  
  // 创建Consumer
  socket.on('consume', async (data: ConsumeData, callback) => {
    const { transportId, producerId, rtpCapabilities } = data;
    
    // 检查是否可以消费
    const canConsume = sfu.canConsume(producerId, rtpCapabilities);
    if (!canConsume) {
      return callback({ error: 'Cannot consume' });
    }
    
    // 创建Consumer
    const consumerParams = await sfu.consume(transportId, producerId, rtpCapabilities);
    callback(consumerParams);
  });
  
  // 恢复Consumer
  socket.on('consumer-resume', async (data: { consumerId: string }) => {
    await sfu.resumeConsumer(data.consumerId);
  });
}
```

### 3.4 聊天事件

```typescript
// handlers/chat.handler.ts
export function registerChatHandlers(socket: Socket, io: Server) {
  
  socket.on('send-message', async (data: SendMessageData) => {
    // 1. 验证消息
    // 2. 保存到数据库
    // 3. 广播给房间（或私聊目标）
    
    const message = {
      id: generateId(),
      senderId: socket.data.userId,
      senderName: socket.data.username,
      content: data.content,
      type: data.type || 'text',
      timestamp: new Date(),
      isPrivate: !!data.receiverId
    };
    
    if (data.receiverId) {
      // 私聊
      const targetSocket = getSocketByUserId(data.receiverId);
      targetSocket?.emit('new-message', message);
      socket.emit('new-message', message); // 发送者也收到
    } else {
      // 群聊
      io.to(socket.data.meetingId).emit('new-message', message);
    }
  });
  
  socket.on('load-history', async (data: { before?: Date }, callback) => {
    const messages = await getChatHistory(socket.data.meetingId, data.before);
    callback({ messages });
  });
}

interface SendMessageData {
  content: string;
  type?: 'text' | 'file' | 'image';
  receiverId?: string;
  fileUrl?: string;
  fileName?: string;
}
```

### 3.5 白板事件

```typescript
// handlers/whiteboard.handler.ts
export function registerWhiteboardHandlers(socket: Socket, io: Server) {
  
  // 白板操作同步
  socket.on('whiteboard-operation', (data: WhiteboardOperationData) => {
    // 广播给房间其他人（不包括发送者）
    socket.to(socket.data.meetingId).emit('whiteboard-operation', {
      ...data,
      userId: socket.data.userId
    });
  });
  
  // 白板快照（定期同步）
  socket.on('whiteboard-snapshot', async (data: { snapshot: any }) => {
    // 保存到Redis
    await saveWhiteboardSnapshot(socket.data.meetingId, data.snapshot);
  });
  
  // 请求白板状态（新加入时）
  socket.on('whiteboard-sync', async (callback) => {
    const snapshot = await getWhiteboardSnapshot(socket.data.meetingId);
    callback({ snapshot });
  });
  
  // 清空白板
  socket.on('whiteboard-clear', () => {
    if (!canControlWhiteboard(socket.data.peerId)) {
      return;
    }
    io.to(socket.data.meetingId).emit('whiteboard-cleared');
  });
}
```

### 3.6 会议控制事件

```typescript
// handlers/control.handler.ts
export function registerControlHandlers(socket: Socket, io: Server, sfu: SFUService) {
  
  // 静音参与者
  socket.on('mute-participant', async (data: { targetPeerId: string }) => {
    // 验证权限（主持人/共同主持人）
    if (!canMuteParticipant(socket.data.peerId, data.targetPeerId)) {
      return socket.emit('error', { message: 'No permission' });
    }
    
    // 关闭目标的音频Producer
    const peer = getPeer(data.targetPeerId);
    const audioProducer = peer.getAudioProducer();
    if (audioProducer) {
      await sfu.pauseProducer(audioProducer.id);
      io.to(socket.data.meetingId).emit('participant-muted', {
        peerId: data.targetPeerId
      });
    }
  });
  
  // 全员静音
  socket.on('mute-all', async () => {
    if (!isHost(socket.data.peerId)) {
      return;
    }
    
    const peers = getRoomPeers(socket.data.meetingId);
    for (const peer of peers) {
      if (peer.id !== socket.data.peerId && peer.audioEnabled) {
        const audioProducer = peer.getAudioProducer();
        if (audioProducer) {
          await sfu.pauseProducer(audioProducer.id);
        }
      }
    }
    io.to(socket.data.meetingId).emit('all-muted');
  });
  
  // 移除参与者
  socket.on('remove-participant', async (data: { targetPeerId: string }) => {
    if (!canRemoveParticipant(socket.data.peerId, data.targetPeerId)) {
      return;
    }
    
    const targetSocket = getSocketByPeerId(data.targetPeerId);
    targetSocket?.emit('removed-from-meeting');
    targetSocket?.disconnect();
  });
  
  // 举手
  socket.on('raise-hand', (data: { raised: boolean }) => {
    const peer = getPeer(socket.data.peerId);
    peer.isHandRaised = data.raised;
    io.to(socket.data.meetingId).emit('hand-raised', {
      peerId: socket.data.peerId,
      raised: data.raised
    });
  });
  
  // 录制控制
  socket.on('start-recording', async () => {
    if (!isHost(socket.data.peerId)) {
      return;
    }
    
    // 调用录制服务
    await startRecording(socket.data.meetingId);
    io.to(socket.data.meetingId).emit('recording-started');
  });
  
  socket.on('stop-recording', async () => {
    if (!isHost(socket.data.peerId)) {
      return;
    }
    
    await stopRecording(socket.data.meetingId);
    io.to(socket.data.meetingId).emit('recording-stopped');
  });
}
```

## 4. 与SFU服务通信

```typescript
// services/sfu.service.ts
class SFUService {
  private sfuUrl: string;
  
  constructor(config: { url: string }) {
    this.sfuUrl = config.url;
  }
  
  // HTTP请求到SFU服务
  private async request<T>(path: string, data?: any): Promise<T> {
    const response = await axios.post(`${this.sfuUrl}${path}`, data);
    return response.data;
  }
  
  // 创建/加入Room
  async createRoom(meetingId: string): Promise<{ rtpCapabilities: any }> {
    return this.request('/rooms/create', { meetingId });
  }
  
  async joinRoom(meetingId: string, peerId: string): Promise<{ rtpCapabilities: any }> {
    return this.request('/rooms/join', { meetingId, peerId });
  }
  
  // Transport
  async createTransport(meetingId: string, peerId: string, direction: string): Promise<TransportParams> {
    return this.request('/transports/create', { meetingId, peerId, direction });
  }
  
  async connectTransport(transportId: string, dtlsParameters: any): Promise<void> {
    return this.request('/transports/connect', { transportId, dtlsParameters });
  }
  
  // Producer
  async produce(transportId: string, options: any): Promise<{ id: string }> {
    return this.request('/producers/produce', { transportId, ...options });
  }
  
  async closeProducer(producerId: string): Promise<void> {
    return this.request('/producers/close', { producerId });
  }
  
  async pauseProducer(producerId: string): Promise<void> {
    return this.request('/producers/pause', { producerId });
  }
  
  async resumeProducer(producerId: string): Promise<void> {
    return this.request('/producers/resume', { producerId });
  }
  
  // Consumer
  async consume(transportId: string, producerId: string, rtpCapabilities: any): Promise<ConsumerParams> {
    return this.request('/consumers/consume', { transportId, producerId, rtpCapabilities });
  }
  
  async resumeConsumer(consumerId: string): Promise<void> {
    return this.request('/consumers/resume', { consumerId });
  }
  
  // Room查询
  async getRoomPeers(meetingId: string): Promise<PeerInfo[]> {
    return this.request('/rooms/peers', { meetingId });
  }
  
  async getProducerList(meetingId: string): Promise<ProducerInfo[]> {
    return this.request('/rooms/producers', { meetingId });
  }
}
```

## 5. Redis状态管理

```typescript
// services/redis.service.ts
class RedisService {
  private client: Redis;
  
  constructor(config: RedisConfig) {
    this.client = new Redis(config);
  }
  
  // 房间在线用户
  async addPeerToRoom(meetingId: string, peerId: string, peerData: any): Promise<void> {
    await this.client.hset(`room:${meetingId}:peers`, peerId, JSON.stringify(peerData));
    await this.client.sadd('online:meetings', meetingId);
  }
  
  async removePeerFromRoom(meetingId: string, peerId: string): Promise<void> {
    await this.client.hdel(`room:${meetingId}:peers`, peerId);
    const count = await this.client.hlen(`room:${meetingId}:peers`);
    if (count === 0) {
      await this.client.srem('online:meetings', meetingId);
    }
  }
  
  async getRoomPeers(meetingId: string): Promise<Map<string, any>> {
    const data = await this.client.hgetall(`room:${meetingId}:peers`);
    const peers = new Map();
    for (const [key, value] of Object.entries(data)) {
      peers.set(key, JSON.parse(value));
    }
    return peers;
  }
  
  // 用户Socket映射
  async setUserSocket(userId: string, socketId: string): Promise<void> {
    await this.client.set(`user:${userId}:socket`, socketId);
  }
  
  async getUserSocket(userId: string): Promise<string | null> {
    return this.client.get(`user:${userId}:socket`);
  }
  
  async removeUserSocket(userId: string): Promise<void> {
    await this.client.del(`user:${userId}:socket`);
  }
  
  // Transport/Producer/Consumer映射
  async saveTransportMapping(transportId: string, data: any): Promise<void> {
    await this.client.set(`transport:${transportId}`, JSON.stringify(data), 'EX', 3600);
  }
  
  async getTransportMapping(transportId: string): Promise<any> {
    const data = await this.client.get(`transport:${transportId}`);
    return data ? JSON.parse(data) : null;
  }
  
  // 白板快照
  async saveWhiteboardSnapshot(meetingId: string, snapshot: any): Promise<void> {
    await this.client.set(`whiteboard:${meetingId}`, JSON.stringify(snapshot));
  }
  
  async getWhiteboardSnapshot(meetingId: string): Promise<any> {
    const data = await this.client.get(`whiteboard:${meetingId}`);
    return data ? JSON.parse(data) : null;
  }
  
  // 清理房间资源
  async cleanupRoom(meetingId: string): Promise<void> {
    const keys = await this.client.keys(`*:${meetingId}:*`);
    if (keys.length > 0) {
      await this.client.del(...keys);
    }
    await this.client.srem('online:meetings', meetingId);
  }
}
```

## 6. 认证中间件

```typescript
// middlewares/auth.middleware.ts
export function authMiddleware(socket: Socket, next: (err?: Error) => void) {
  const token = socket.handshake.auth.token;
  
  if (!token) {
    return next(new Error('Authentication required'));
  }
  
  try {
    const decoded = jwt.verify(token, JWT_SECRET) as JwtPayload;
    socket.data.userId = decoded.userId;
    socket.data.username = decoded.username;
    socket.data.avatar = decoded.avatar;
    next();
  } catch (error) {
    next(new Error('Invalid token'));
  }
}
```

## 7. 服务器启动

```typescript
// index.ts
async function bootstrap() {
  // 1. 创建HTTP服务器
  const app = express();
  const httpServer = createServer(app);
  
  // 2. 创建Socket.IO服务器
  const io = new Server(httpServer, {
    cors: { origin: CLIENT_URL, credentials: true },
    transports: ['websocket', 'polling']
  });
  
  // 3. 初始化服务
  const redisService = new RedisService(REDIS_CONFIG);
  const sfuService = new SFUService({ url: SFU_URL });
  const roomManager = new RoomManager(redisService, sfuService);
  
  // 4. 应用中间件
  io.use(authMiddleware);
  io.use(rateLimitMiddleware);
  io.use(loggerMiddleware);
  
  // 5. 连接处理
  io.on('connection', (socket) => {
    console.log(`User connected: ${socket.id}`);
    
    // 注册所有事件处理器
    registerRoomHandlers(socket, io, roomManager);
    registerTransportHandlers(socket, io, sfuService);
    registerProducerHandlers(socket, io, sfuService);
    registerConsumerHandlers(socket, io, sfuService);
    registerChatHandlers(socket, io);
    registerWhiteboardHandlers(socket, io);
    registerControlHandlers(socket, io, sfuService);
  });
  
  // 6. 启动服务器
  httpServer.listen(PORT, () => {
    console.log(`Signaling server running on port ${PORT}`);
  });
}

bootstrap();
```
