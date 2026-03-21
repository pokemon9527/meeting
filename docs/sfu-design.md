# SFU媒体服务器详细设计

## 1. 项目结构

```
server/sfu/
├── src/
│   ├── config/
│   │   ├── index.ts              # 配置入口
│   │   ├── mediasoup.ts          # mediasoup配置
│   │   └── network.ts            # 网络配置
│   │
│   ├── rooms/
│   │   ├── index.ts              # 房间管理器
│   │   ├── Room.ts               # Room类
│   │   └── Peer.ts               # Peer类
│   │
│   ├── media/
│   │   ├── index.ts              # 媒体管理器
│   │   ├── Router.ts             # mediasoup Router包装
│   │   ├── Transport.ts          # Transport管理
│   │   ├── Producer.ts           # Producer管理
│   │   ├── Consumer.ts           # Consumer管理
│   │   └── Simulcast.ts          # Simulcast管理
│   │
│   ├── api/
│   │   ├── index.ts              # API路由
│   │   ├── room.routes.ts        # 房间路由
│   │   ├── transport.routes.ts   # Transport路由
│   │   ├── producer.routes.ts    # Producer路由
│   │   └── consumer.routes.ts    # Consumer路由
│   │
│   ├── recording/
│   │   ├── index.ts              # 录制管理器
│   │   ├── Recorder.ts           # 录制器
│   │   └── Mixer.ts              # 混流器
│   │
│   ├── types/
│   │   ├── room.ts               # 房间类型
│   │   ├── peer.ts               # Peer类型
│   │   └── media.ts              # 媒体类型
│   │
│   ├── utils/
│   │   ├── logger.ts             # 日志工具
│   │   └── helpers.ts            # 工具函数
│   │
│   └── index.ts                  # 入口文件
│
├── package.json
└── tsconfig.json
```

## 2. mediasoup配置

```typescript
// config/mediasoup.ts
export const mediasoupConfig = {
  // Worker配置
  worker: {
    rtcMinPort: 10000,
    rtcMaxPort: 59999,
    logLevel: 'warn',
    logTags: [
      'info',
      'ice',
      'dtls',
      'rtp',
      'srtp',
      'rtcp',
      'rtx',
      'bwe',
      'score',
      'simulcast',
      'svc',
      'sctp'
    ],
    // 根据CPU核心数创建Worker
    numWorkers: Math.max(1, require('os').cpus().length - 1)
  },

  // Router配置
  router: {
    mediaCodecs: [
      {
        kind: 'audio',
        mimeType: 'audio/opus',
        clockRate: 48000,
        channels: 2
      },
      {
        kind: 'video',
        mimeType: 'video/VP8',
        clockRate: 90000,
        parameters: {
          'x-google-start-bitrate': 1000
        }
      },
      {
        kind: 'video',
        mimeType: 'video/VP9',
        clockRate: 90000,
        parameters: {
          'profile-id': 2,
          'x-google-start-bitrate': 1000
        }
      },
      {
        kind: 'video',
        mimeType: 'video/h264',
        clockRate: 90000,
        parameters: {
          'packetization-mode': 1,
          'profile-level-id': '4d0032',
          'level-asymmetry-allowed': 1,
          'x-google-start-bitrate': 1000
        }
      },
      {
        kind: 'video',
        mimeType: 'video/h264',
        clockRate: 90000,
        parameters: {
          'packetization-mode': 1,
          'profile-level-id': '42e01f',
          'level-asymmetry-allowed': 1,
          'x-google-start-bitrate': 1000
        }
      }
    ]
  },

  // WebRTC Transport配置
  webRtcTransport: {
    listenIps: [
      {
        ip: '0.0.0.0',
        announcedIp: process.env.ANNOUNCED_IP || '127.0.0.1'  // 公网IP
      }
    ],
    initialAvailableOutgoingBitrate: 1000000,
    minimumAvailableOutgoingBitrate: 600000,
    maxSctpMessageSize: 262144,
    maxIncomingBitrate: 1500000,
    // 启用Simulcast
    enableSimulcast: true,
    enableUdp: true,
    enableTcp: true,
    preferUdp: true
  },

  // PlainRtp Transport配置（用于录制）
  plainRtpTransport: {
    listenIp: { ip: '127.0.0.1', announcedIp: undefined },
    rtcpMux: true,
    comedia: false
  }
};

// Simulcast层配置
export const simulcastProfiles = {
  high: {
    width: 1280,
    height: 720,
    maxBitrate: 1500000,
    maxFps: 30
  },
  medium: {
    width: 640,
    height: 360,
    maxBitrate: 500000,
    maxFps: 30
  },
  low: {
    width: 320,
    height: 180,
    maxBitrate: 150000,
    maxFps: 15
  }
};
```

## 3. 核心类设计

### 3.1 WorkerManager

```typescript
// src/index.ts - Worker管理
class WorkerManager {
  private workers: mediasoup.types.Worker[] = [];
  private nextWorkerIndex = 0;
  
  async initialize(): Promise<void> {
    const numWorkers = mediasoupConfig.worker.numWorkers;
    
    for (let i = 0; i < numWorkers; i++) {
      const worker = await mediasoup.createWorker({
        rtcMinPort: mediasoupConfig.worker.rtcMinPort,
        rtcMaxPort: mediasoupConfig.worker.rtcMaxPort,
        logLevel: mediasoupConfig.worker.logLevel as any,
        logTags: mediasoupConfig.worker.logTags as any
      });
      
      worker.on('died', () => {
        console.error(`mediasoup Worker died, pid: ${worker.pid}`);
        // 从数组移除并创建新的Worker
        this.workers = this.workers.filter(w => w !== worker);
        this.createWorker();
      });
      
      this.workers.push(worker);
    }
    
    console.log(`Created ${this.workers.length} mediasoup Workers`);
  }
  
  // 轮询获取下一个Worker
  getNextWorker(): mediasoup.types.Worker {
    const worker = this.workers[this.nextWorkerIndex];
    this.nextWorkerIndex = (this.nextWorkerIndex + 1) % this.workers.length;
    return worker;
  }
  
  async closeAll(): Promise<void> {
    for (const worker of this.workers) {
      worker.close();
    }
    this.workers = [];
  }
}
```

### 3.2 Room类

```typescript
// rooms/Room.ts
class Room {
  readonly id: string;
  readonly meetingId: string;
  private router: mediasoup.types.Router;
  private peers: Map<string, Peer> = new Map();
  private producers: Map<string, mediasoup.types.Producer> = new Map();
  
  constructor(
    id: string,
    meetingId: string,
    worker: mediasoup.types.Worker
  ) {
    this.id = id;
    this.meetingId = meetingId;
    // Router在create时初始化
  }
  
  static async create(
    id: string,
    meetingId: string,
    worker: mediasoup.types.Worker
  ): Promise<Room> {
    const room = new Room(id, meetingId, worker);
    room.router = await worker.createRouter({
      mediaCodecs: mediasoupConfig.router.mediaCodecs
    });
    return room;
  }
  
  // 获取Router RTP Capabilities
  get rtpCapabilities(): mediasoup.types.RtpCapabilities {
    return this.router.rtpCapabilities;
  }
  
  // Peer管理
  async addPeer(peerId: string, data: PeerData): Promise<Peer> {
    const peer = new Peer(peerId, data, this.router);
    this.peers.set(peerId, peer);
    return peer;
  }
  
  removePeer(peerId: string): void {
    const peer = this.peers.get(peerId);
    if (peer) {
      peer.close();
      this.peers.delete(peerId);
    }
  }
  
  getPeer(peerId: string): Peer | undefined {
    return this.peers.get(peerId);
  }
  
  getPeers(): Peer[] {
    return Array.from(this.peers.values());
  }
  
  // Producer管理
  addProducer(producer: mediasoup.types.Producer, peerId: string): void {
    this.producers.set(producer.id, producer);
    
    // 新Producer加入，通知其他Peer创建Consumer
    for (const [id, peer] of this.peers) {
      if (id !== peerId) {
        peer.onNewProducer(producer);
      }
    }
  }
  
  removeProducer(producerId: string): void {
    this.producers.delete(producerId);
  }
  
  getProducer(producerId: string): mediasoup.types.Producer | undefined {
    return this.producers.get(producerId);
  }
  
  // 录制相关 - 获取PlainRtpTransport
  async createPlainTransport(): Promise<mediasoup.types.PlainTransport> {
    return this.router.createPlainTransport({
      listenIp: mediasoupConfig.plainRtpTransport.listenIp,
      rtcpMux: mediasoupConfig.plainRtpTransport.rtcpMux,
      comedia: mediasoupConfig.plainRtpTransport.comedia
    });
  }
  
  // 统计信息
  getStats(): RoomStats {
    return {
      id: this.id,
      meetingId: this.meetingId,
      peerCount: this.peers.size,
      producerCount: this.producers.size,
      consumerCount: Array.from(this.peers.values())
        .reduce((sum, peer) => sum + peer.getConsumerCount(), 0)
    };
  }
  
  close(): void {
    // 关闭所有Peer
    for (const peer of this.peers.values()) {
      peer.close();
    }
    this.peers.clear();
    this.producers.clear();
    
    // 关闭Router
    this.router.close();
  }
}
```

### 3.3 Peer类

```typescript
// rooms/Peer.ts
class Peer {
  readonly id: string;
  readonly userId: string;
  readonly username: string;
  
  private router: mediasoup.types.Router;
  private sendTransport: mediasoup.types.WebRtcTransport | null = null;
  private recvTransport: mediasoup.types.WebRtcTransport | null = null;
  private producers: Map<string, mediasoup.types.Producer> = new Map();
  private consumers: Map<string, mediasoup.types.Consumer> = new Map();
  
  // 新Producer回调
  private newProducerCallback: ((producer: mediasoup.types.Producer) => void) | null = null;
  
  constructor(
    id: string,
    data: PeerData,
    router: mediasoup.types.Router
  ) {
    this.id = id;
    this.userId = data.userId;
    this.username = data.username;
    this.router = router;
  }
  
  // Transport创建
  async createTransport(direction: 'send' | 'recv'): Promise<TransportParams> {
    const transport = await this.router.createWebRtcTransport({
      ...mediasoupConfig.webRtcTransport,
      appData: { peerId: this.id, direction }
    });
    
    // 监听DTLS状态变化
    transport.on('dtlsstatechange', (dtlsState) => {
      if (dtlsState === 'closed') {
        transport.close();
      }
    });
    
    // 监听ICE状态
    transport.on('icestatechange', (iceState) => {
      console.log(`Transport ${transport.id} ICE state: ${iceState}`);
    });
    
    if (direction === 'send') {
      this.sendTransport = transport;
    } else {
      this.recvTransport = transport;
    }
    
    return {
      id: transport.id,
      iceParameters: transport.iceParameters,
      iceCandidates: transport.iceCandidates,
      dtlsParameters: transport.dtlsParameters,
      sctpParameters: transport.sctpParameters
    };
  }
  
  // 连接Transport
  async connectTransport(transportId: string, dtlsParameters: any): Promise<void> {
    const transport = this.getTransport(transportId);
    if (!transport) {
      throw new Error(`Transport ${transportId} not found`);
    }
    await transport.connect({ dtlsParameters });
  }
  
  // 创建Producer
  async produce(
    transportId: string,
    options: {
      kind: mediasoup.types.MediaKind;
      rtpParameters: mediasoup.types.RtpParameters;
      appData?: any;
    }
  ): Promise<mediasoup.types.Producer> {
    if (this.sendTransport?.id !== transportId) {
      throw new Error('Invalid transport for producing');
    }
    
    const producer = await this.sendTransport.produce({
      kind: options.kind,
      rtpParameters: options.rtpParameters,
      appData: { ...options.appData, peerId: this.id }
    });
    
    this.producers.set(producer.id, producer);
    
    // 监听Producer事件
    producer.on('transportclose', () => {
      this.producers.delete(producer.id);
    });
    
    producer.on('score', (score) => {
      // 可用于监控质量
    });
    
    producer.on('videoorientationchange', (orientation) => {
      // 视频方向变化
    });
    
    return producer;
  }
  
  // 创建Consumer
  async consume(
    transportId: string,
    producer: mediasoup.types.Producer,
    rtpCapabilities: mediasoup.types.RtpCapabilities
  ): Promise<ConsumerParams> {
    // 检查是否可以消费
    if (!this.router.canConsume({ producerId: producer.id, rtpCapabilities })) {
      throw new Error('Cannot consume this producer');
    }
    
    const transport = this.getTransport(transportId);
    if (!transport || transport !== this.recvTransport) {
      throw new Error('Invalid transport for consuming');
    }
    
    const consumer = await transport.consume({
      producerId: producer.id,
      rtpCapabilities,
      paused: true  // 先暂停，客户端准备好了再resume
    });
    
    this.consumers.set(consumer.id, consumer);
    
    // 监听Consumer事件
    consumer.on('transportclose', () => {
      this.consumers.delete(consumer.id);
    });
    
    consumer.on('producerclose', () => {
      this.consumers.delete(consumer.id);
    });
    
    return {
      id: consumer.id,
      producerId: producer.id,
      kind: consumer.kind,
      rtpParameters: consumer.rtpParameters,
      type: consumer.type,
      producerPaused: consumer.producerPaused
    };
  }
  
  // 新Producer回调（用于自动创建Consumer）
  onNewProducer(callback: (producer: mediasoup.types.Producer) => void): void {
    this.newProducerCallback = callback;
  }
  
  onNewProducer(producer: mediasoup.types.Producer): void {
    // 通知回调
    this.newProducerCallback?.(producer);
  }
  
  // 暂停/恢复Producer
  async pauseProducer(producerId: string): Promise<void> {
    const producer = this.producers.get(producerId);
    if (producer) {
      await producer.pause();
    }
  }
  
  async resumeProducer(producerId: string): Promise<void> {
    const producer = this.producers.get(producerId);
    if (producer) {
      await producer.resume();
    }
  }
  
  // 暂停/恢复Consumer
  async resumeConsumer(consumerId: string): Promise<void> {
    const consumer = this.consumers.get(consumerId);
    if (consumer) {
      await consumer.resume();
    }
  }
  
  // Simulcast层选择
  async setConsumerPreferredLayers(
    consumerId: string,
    layers: { spatialLayer: number; temporalLayer: number }
  ): Promise<void> {
    const consumer = this.consumers.get(consumerId);
    if (consumer && consumer.type === 'simulcast') {
      await consumer.setPreferredLayers(layers);
    }
  }
  
  // 获取统计信息
  async getProducerStats(producerId: string): Promise<any> {
    const producer = this.producers.get(producerId);
    if (!producer) throw new Error('Producer not found');
    return producer.getStats();
  }
  
  async getConsumerStats(consumerId: string): Promise<any> {
    const consumer = this.consumers.get(consumerId);
    if (!consumer) throw new Error('Consumer not found');
    return consumer.getStats();
  }
  
  // 工具方法
  private getTransport(transportId: string): mediasoup.types.WebRtcTransport | null {
    if (this.sendTransport?.id === transportId) return this.sendTransport;
    if (this.recvTransport?.id === transportId) return this.recvTransport;
    return null;
  }
  
  getAudioProducer(): mediasoup.types.Producer | undefined {
    return Array.from(this.producers.values()).find(p => p.kind === 'audio');
  }
  
  getVideoProducer(): mediasoup.types.Producer | undefined {
    return Array.from(this.producers.values()).find(p => p.kind === 'video' && !p.appData.share);
  }
  
  getScreenProducer(): mediasoup.types.Producer | undefined {
    return Array.from(this.producers.values()).find(p => p.appData?.share === true);
  }
  
  getProducerCount(): number {
    return this.producers.size;
  }
  
  getConsumerCount(): number {
    return this.consumers.size;
  }
  
  close(): void {
    // 关闭所有Producer
    for (const producer of this.producers.values()) {
      producer.close();
    }
    this.producers.clear();
    
    // 关闭所有Consumer
    for (const consumer of this.consumers.values()) {
      consumer.close();
    }
    this.consumers.clear();
    
    // 关闭Transport
    this.sendTransport?.close();
    this.recvTransport?.close();
  }
}
```

## 4. API路由设计

```typescript
// api/index.ts
import express from 'express';

export function createApiRouter(roomManager: RoomManager): express.Router {
  const router = express.Router();
  
  // 房间管理
  router.post('/rooms/create', async (req, res) => {
    const { meetingId } = req.body;
    const room = await roomManager.createRoom(meetingId);
    res.json({ roomId: room.id, rtpCapabilities: room.rtpCapabilities });
  });
  
  router.post('/rooms/join', async (req, res) => {
    const { meetingId, peerId, peerData } = req.body;
    const room = await roomManager.getRoom(meetingId);
    if (!room) {
      return res.status(404).json({ error: 'Room not found' });
    }
    const peer = await room.addPeer(peerId, peerData);
    res.json({ 
      rtpCapabilities: room.rtpCapabilities,
      existingProducers: getExistingProducerIds(room)
    });
  });
  
  // Transport管理
  router.post('/transports/create', async (req, res) => {
    const { meetingId, peerId, direction } = req.body;
    const room = await roomManager.getRoom(meetingId);
    const peer = room?.getPeer(peerId);
    if (!peer) {
      return res.status(404).json({ error: 'Peer not found' });
    }
    const transportParams = await peer.createTransport(direction);
    res.json(transportParams);
  });
  
  router.post('/transports/connect', async (req, res) => {
    const { transportId, dtlsParameters, meetingId, peerId } = req.body;
    const room = await roomManager.getRoom(meetingId);
    const peer = room?.getPeer(peerId);
    if (!peer) {
      return res.status(404).json({ error: 'Peer not found' });
    }
    await peer.connectTransport(transportId, dtlsParameters);
    res.json({ connected: true });
  });
  
  // Producer管理
  router.post('/producers/produce', async (req, res) => {
    const { transportId, kind, rtpParameters, appData, meetingId, peerId } = req.body;
    const room = await roomManager.getRoom(meetingId);
    const peer = room?.getPeer(peerId);
    if (!peer) {
      return res.status(404).json({ error: 'Peer not found' });
    }
    const producer = await peer.produce(transportId, { kind, rtpParameters, appData });
    room.addProducer(producer, peerId);
    res.json({ id: producer.id });
  });
  
  router.post('/producers/pause', async (req, res) => {
    const { producerId, meetingId, peerId } = req.body;
    const room = await roomManager.getRoom(meetingId);
    const peer = room?.getPeer(peerId);
    if (!peer) {
      return res.status(404).json({ error: 'Peer not found' });
    }
    await peer.pauseProducer(producerId);
    res.json({ paused: true });
  });
  
  router.post('/producers/resume', async (req, res) => {
    const { producerId, meetingId, peerId } = req.body;
    const room = await roomManager.getRoom(meetingId);
    const peer = room?.getPeer(peerId);
    if (!peer) {
      return res.status(404).json({ error: 'Peer not found' });
    }
    await peer.resumeProducer(producerId);
    res.json({ resumed: true });
  });
  
  // Consumer管理
  router.post('/consumers/consume', async (req, res) => {
    const { transportId, producerId, rtpCapabilities, meetingId, peerId } = req.body;
    const room = await roomManager.getRoom(meetingId);
    const peer = room?.getPeer(peerId);
    if (!peer) {
      return res.status(404).json({ error: 'Peer not found' });
    }
    const producer = room.getProducer(producerId);
    if (!producer) {
      return res.status(404).json({ error: 'Producer not found' });
    }
    const consumerParams = await peer.consume(transportId, producer, rtpCapabilities);
    res.json(consumerParams);
  });
  
  router.post('/consumers/resume', async (req, res) => {
    const { consumerId, meetingId, peerId } = req.body;
    const room = await roomManager.getRoom(meetingId);
    const peer = room?.getPeer(peerId);
    if (!peer) {
      return res.status(404).json({ error: 'Peer not found' });
    }
    await peer.resumeConsumer(consumerId);
    res.json({ resumed: true });
  });
  
  // 查询接口
  router.get('/rooms/:meetingId/peers', (req, res) => {
    const room = roomManager.getRoom(req.params.meetingId);
    if (!room) {
      return res.status(404).json({ error: 'Room not found' });
    }
    res.json({ peers: room.getPeers().map(p => p.toJSON()) });
  });
  
  router.get('/rooms/:meetingId/stats', (req, res) => {
    const room = roomManager.getRoom(req.params.meetingId);
    if (!room) {
      return res.status(404).json({ error: 'Room not found' });
    }
    res.json(room.getStats());
  });
  
  return router;
}
```

## 5. Simulcast管理

```typescript
// media/Simulcast.ts
class SimulcastManager {
  // 根据Consumer数量动态调整订阅的Simulcast层
  static getOptimalLayers(
    producer: mediasoup.types.Producer,
    consumerCount: number
  ): { spatialLayer: number; temporalLayer: number } {
    // Simulcast层说明：
    // spatial: 0=180p, 1=360p, 2=720p
    // temporal: 0=低帧率, 1=中帧率, 2=高帧率
    
    if (consumerCount <= 5) {
      // 少量观众，全质量
      return { spatialLayer: 2, temporalLayer: 2 };
    } else if (consumerCount <= 15) {
      // 中等观众，中等质量
      return { spatialLayer: 1, temporalLayer: 1 };
    } else {
      // 大量观众，低质量
      return { spatialLayer: 0, temporalLayer: 0 };
    }
  }
  
  // 根据网络状况调整
  static async adjustForNetwork(
    consumer: mediasoup.types.Consumer,
    score: number
  ): Promise<void> {
    if (consumer.type !== 'simulcast') return;
    
    const currentLayers = consumer.currentLayers;
    let newLayers = { ...currentLayers };
    
    if (score < 5) {
      // 网络差，降级
      newLayers.spatialLayer = Math.max(0, currentLayers.spatialLayer - 1);
    } else if (score > 8) {
      // 网络好，升级
      newLayers.spatialLayer = Math.min(2, currentLayers.spatialLayer + 1);
    }
    
    if (newLayers.spatialLayer !== currentLayers.spatialLayer) {
      await consumer.setPreferredLayers(newLayers);
    }
  }
}
```

## 6. 性能优化

### 6.1 Worker负载均衡

```typescript
class LoadBalancer {
  private workers: Map<string, { worker: Worker; load: number }> = new Map();
  
  getLeastLoadedWorker(): Worker {
    let minLoad = Infinity;
    let selected: Worker | null = null;
    
    for (const { worker, load } of this.workers.values()) {
      if (load < minLoad) {
        minLoad = load;
        selected = worker;
      }
    }
    
    return selected!;
  }
  
  updateLoad(workerId: string, load: number): void {
    const entry = this.workers.get(workerId);
    if (entry) {
      entry.load = load;
    }
  }
}
```

### 6.2 内存管理

```typescript
// 定期清理不活跃的Room
class RoomCleaner {
  private cleanInterval: NodeJS.Timer;
  
  start(intervalMs: number = 60000): void {
    this.cleanInterval = setInterval(() => {
      this.cleanInactiveRooms();
    }, intervalMs);
  }
  
  private cleanInactiveRooms(): void {
    const rooms = roomManager.getAllRooms();
    for (const room of rooms) {
      if (room.getPeers().length === 0) {
        roomManager.removeRoom(room.meetingId);
        console.log(`Cleaned up empty room: ${room.meetingId}`);
      }
    }
  }
}
```
