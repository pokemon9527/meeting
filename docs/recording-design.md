# 录制服务详细设计

## 1. 项目结构

```
server/recording/
├── src/
│   ├── config/
│   │   ├── index.ts              # 配置入口
│   │   ├── ffmpeg.ts             # FFmpeg配置
│   │   └── storage.ts            # 存储配置
│   │
│   ├── core/
│   │   ├── index.ts              # 录制管理器
│   │   ├── Session.ts            # 录制会话
│   │   ├── Recorder.ts           # 录制器
│   │   └── Mixer.ts              # 混流器
│   │
│   ├── api/
│   │   ├── index.ts              # API路由
│   │   └── recording.routes.ts   # 录制路由
│   │
│   ├── storage/
│   │   ├── index.ts              # 存储接口
│   │   ├── LocalStorage.ts       # 本地存储
│   │   └── S3Storage.ts          # S3/MinIO存储
│   │
│   ├── types/
│   │   └── recording.ts          # 类型定义
│   │
│   └── index.ts                  # 入口文件
│
├── package.json
└── tsconfig.json
```

## 2. 录制架构

```
┌─────────────────────────────────────────────────────────────────┐
│                        SFU服务器                                  │
│                                                                  │
│  ┌─────────────────────────────────────────────────────────────┐│
│  │                         Router                               ││
│  │                                                              ││
│  │  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐         ││
│  │  │  Producer 1  │  │  Producer 2  │  │  Producer N  │         ││
│  │  │  (User A)    │  │  (User B)    │  │  (User N)    │         ││
│  │  └──────┬───────┘  └──────┬───────┘  └──────┬───────┘         ││
│  │         │                 │                 │                 ││
│  │         ▼                 ▼                 ▼                 ││
│  │  ┌─────────────────────────────────────────────────┐         ││
│  │  │           PlainRtpTransport (录制)               │         ││
│  │  │  Port: 5004 (音频)  Port: 5006 (视频)            │         ││
│  │  └─────────────────────────────────────────────────┘         ││
│  └─────────────────────────────────────────────────────────────┘│
│                              │                                   │
└──────────────────────────────┼───────────────────────────────────┘
                               │ RTP流
                               ▼
┌─────────────────────────────────────────────────────────────────┐
│                        录制服务                                    │
│                                                                  │
│  ┌─────────────────────────────────────────────────────────────┐│
│  │                      Recording Session                       ││
│  │                                                              ││
│  │  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐         ││
│  │  │  Recorder A  │  │  Recorder B  │  │  Recorder N  │         ││
│  │  │  ┌───┐       │  │  ┌───┐       │  │  ┌───┐       │         ││
│  │  │  │FFM│───────┼──┼──│FFM│───────┼──┼──│FFM│       │         ││
│  │  │  └───┘       │  │  └───┘       │  │  └───┘       │         ││
│  │  └──────┬───────┘  └──────┬───────┘  └──────┬───────┘         ││
│  │         │                 │                 │                 ││
│  │         ▼                 ▼                 ▼                 ││
│  │  ┌─────────────────────────────────────────────────┐         ││
│  │  │                  Mixer (混流)                     │         ││
│  │  │  ┌─────────────────────────────────────────┐   │         ││
│  │  │  │              FFmpeg 混流命令              │   │         ││
│  │  │  │  -i input1.mp4 -i input2.mp4 ...        │   │         ││
│  │  │  │  -filter_complex "[0:v][1:v]hstack..."  │   │         ││
│  │  │  │  -c:v libx264 -c:a aac output.mp4       │   │         ││
│  │  │  └─────────────────────────────────────────┘   │         ││
│  │  └─────────────────────────────────────────────────┘         ││
│  │                          │                                   ││
│  │                          ▼                                   ││
│  │  ┌─────────────────────────────────────────────────┐         ││
│  │  │                  Storage                          │         ││
│  │  │  Local / MinIO / AWS S3                          │         ││
│  │  └─────────────────────────────────────────────────┘         ││
│  └─────────────────────────────────────────────────────────────┘│
└─────────────────────────────────────────────────────────────────┘
```

## 3. 核心类设计

### 3.1 RecordingManager

```typescript
// core/index.ts
class RecordingManager {
  private sessions: Map<string, RecordingSession> = new Map();
  private storage: StorageAdapter;
  private sfuClient: SFUClient;
  
  constructor(config: RecordingConfig) {
    this.storage = config.storage;
    this.sfuClient = config.sfuClient;
  }
  
  // 开始录制
  async startRecording(meetingId: string, options: RecordingOptions): Promise<RecordingSession> {
    // 检查是否已在录制
    if (this.sessions.has(meetingId)) {
      throw new Error('Recording already in progress');
    }
    
    // 从SFU获取当前Producers
    const producers = await this.sfuClient.getRoomProducers(meetingId);
    
    // 创建录制会话
    const session = new RecordingSession({
      meetingId,
      producers,
      options,
      storage: this.storage
    });
    
    await session.start();
    this.sessions.set(meetingId, session);
    
    return session;
  }
  
  // 停止录制
  async stopRecording(meetingId: string): Promise<RecordingResult> {
    const session = this.sessions.get(meetingId);
    if (!session) {
      throw new Error('No recording in progress');
    }
    
    const result = await session.stop();
    this.sessions.delete(meetingId);
    
    return result;
  }
  
  // 暂停录制
  async pauseRecording(meetingId: string): Promise<void> {
    const session = this.sessions.get(meetingId);
    if (!session) {
      throw new Error('No recording in progress');
    }
    await session.pause();
  }
  
  // 恢复录制
  async resumeRecording(meetingId: string): Promise<void> {
    const session = this.sessions.get(meetingId);
    if (!session) {
      throw new Error('No recording in progress');
    }
    await session.resume();
  }
  
  // 添加新的Producer（有人加入会议）
  async addProducer(meetingId: string, producer: ProducerInfo): Promise<void> {
    const session = this.sessions.get(meetingId);
    if (session) {
      await session.addProducer(producer);
    }
  }
  
  // 移除Producer（有人离开会议）
  async removeProducer(meetingId: string, producerId: string): Promise<void> {
    const session = this.sessions.get(meetingId);
    if (session) {
      await session.removeProducer(producerId);
    }
  }
  
  // 获取录制状态
  getRecordingStatus(meetingId: string): RecordingStatus | null {
    const session = this.sessions.get(meetingId);
    return session?.getStatus() || null;
  }
}
```

### 3.2 RecordingSession

```typescript
// core/Session.ts
class RecordingSession {
  readonly meetingId: string;
  readonly sessionId: string;
  readonly startTime: Date;
  
  private recorders: Map<string, StreamRecorder> = new Map();
  private mixer: StreamMixer;
  private storage: StorageAdapter;
  private status: 'recording' | 'paused' | 'stopped' = 'recording';
  
  constructor(config: SessionConfig) {
    this.meetingId = config.meetingId;
    this.sessionId = generateId();
    this.startTime = new Date();
    this.storage = config.storage;
    this.mixer = new StreamMixer(config.options.layout);
    
    // 为每个Producer创建Recorder
    for (const producer of config.producers) {
      this.createRecorder(producer);
    }
  }
  
  async start(): Promise<void> {
    // 创建临时录制目录
    const tempDir = path.join(os.tmpdir(), 'recording', this.sessionId);
    await fs.mkdir(tempDir, { recursive: true });
    
    // 启动所有Recorder
    for (const recorder of this.recorders.values()) {
      await recorder.start();
    }
    
    console.log(`Recording session ${this.sessionId} started`);
  }
  
  async stop(): Promise<RecordingResult> {
    this.status = 'stopped';
    
    // 停止所有Recorder
    for (const recorder of this.recorders.values()) {
      await recorder.stop();
    }
    
    // 混流
    const outputPath = await this.mixer.mix(
      Array.from(this.recorders.values()).map(r => r.getFilePath())
    );
    
    // 上传到存储
    const fileUrl = await this.storage.upload(outputPath, {
      meetingId: this.meetingId,
      sessionId: this.sessionId
    });
    
    // 生成缩略图
    const thumbnailUrl = await this.generateThumbnail(outputPath);
    
    // 清理临时文件
    await this.cleanup();
    
    return {
      sessionId: this.sessionId,
      meetingId: this.meetingId,
      fileUrl,
      thumbnailUrl,
      duration: Math.floor((Date.now() - this.startTime.getTime()) / 1000),
      fileSize: (await fs.stat(outputPath)).size
    };
  }
  
  async pause(): Promise<void> {
    this.status = 'paused';
    for (const recorder of this.recorders.values()) {
      await recorder.pause();
    }
  }
  
  async resume(): Promise<void> {
    this.status = 'recording';
    for (const recorder of this.recorders.values()) {
      await recorder.resume();
    }
  }
  
  async addProducer(producer: ProducerInfo): Promise<void> {
    if (!this.recorders.has(producer.id)) {
      this.createRecorder(producer);
      await this.recorders.get(producer.id)!.start();
    }
  }
  
  async removeProducer(producerId: string): Promise<void> {
    const recorder = this.recorders.get(producerId);
    if (recorder) {
      await recorder.stop();
      this.recorders.delete(producerId);
    }
  }
  
  getStatus(): RecordingStatus {
    return {
      meetingId: this.meetingId,
      sessionId: this.sessionId,
      status: this.status,
      startTime: this.startTime,
      duration: Math.floor((Date.now() - this.startTime.getTime()) / 1000),
      producerCount: this.recorders.size
    };
  }
  
  private createRecorder(producer: ProducerInfo): void {
    const recorder = new StreamRecorder({
      producerId: producer.id,
      kind: producer.kind,
      meetingId: this.meetingId,
      sessionId: this.sessionId,
      rtpParameters: producer.rtpParameters
    });
    this.recorders.set(producer.id, recorder);
  }
  
  private async generateThumbnail(videoPath: string): Promise<string> {
    // 使用FFmpeg生成缩略图
    const thumbnailPath = videoPath.replace('.mp4', '_thumb.jpg');
    await execPromise(
      `ffmpeg -i ${videoPath} -ss 00:00:05 -vframes 1 ${thumbnailPath}`
    );
    return this.storage.upload(thumbnailPath);
  }
  
  private async cleanup(): Promise<void> {
    const tempDir = path.join(os.tmpdir(), 'recording', this.sessionId);
    await fs.rm(tempDir, { recursive: true, force: true });
  }
}
```

### 3.3 StreamRecorder

```typescript
// core/Recorder.ts
class StreamRecorder {
  readonly producerId: string;
  readonly kind: 'audio' | 'video';
  
  private transport: PlainRtpTransport | null = null;
  private consumer: Consumer | null = null;
  private ffmpegProcess: ChildProcess | null = null;
  private outputPath: string;
  private status: 'idle' | 'recording' | 'paused' = 'idle';
  
  constructor(config: RecorderConfig) {
    this.producerId = config.producerId;
    this.kind = config.kind;
    this.outputPath = path.join(
      os.tmpdir(),
      'recording',
      config.sessionId,
      `${config.producerId}_${config.kind}.webm`
    );
  }
  
  async start(): Promise<void> {
    // 创建PlainRtpTransport连接到SFU
    this.transport = await sfuClient.createPlainTransport(this.meetingId);
    
    // 创建Consumer消费Producer
    this.consumer = await sfuClient.consume(
      this.transport.id,
      this.producerId,
      routerRtpCapabilities
    );
    
    // 启动FFmpeg接收RTP流并保存
    await this.startFFmpeg();
    
    this.status = 'recording';
  }
  
  async stop(): Promise<void> {
    this.status = 'stopped';
    
    // 停止FFmpeg
    if (this.ffmpegProcess) {
      this.ffmpegProcess.kill('SIGTERM');
      await new Promise(resolve => this.ffmpegProcess!.on('close', resolve));
    }
    
    // 关闭Consumer和Transport
    await this.consumer?.close();
    await this.transport?.close();
  }
  
  async pause(): Promise<void> {
    this.status = 'paused';
    await this.consumer?.pause();
  }
  
  async resume(): Promise<void> {
    this.status = 'recording';
    await this.consumer?.resume();
  }
  
  getFilePath(): string {
    return this.outputPath;
  }
  
  private async startFFmpeg(): Promise<void> {
    const port = this.transport!.tuple.localPort;
    
    if (this.kind === 'audio') {
      // 音频录制
      this.ffmpegProcess = spawn('ffmpeg', [
        '-protocol_whitelist', 'file,udp,rtp',
        '-i', `rtp://127.0.0.1:${port}`,
        '-c:a', 'libopus',
        '-b:a', '128k',
        '-f', 'webm',
        this.outputPath
      ]);
    } else {
      // 视频录制
      this.ffmpegProcess = spawn('ffmpeg', [
        '-protocol_whitelist', 'file,udp,rtp',
        '-i', `rtp://127.0.0.1:${port}`,
        '-c:v', 'libvpx-vp9',
        '-b:v', '2M',
        '-f', 'webm',
        this.outputPath
      ]);
    }
    
    this.ffmpegProcess.stderr?.on('data', (data) => {
      console.log(`FFmpeg [${this.producerId}]: ${data}`);
    });
    
    this.ffmpegProcess.on('error', (err) => {
      console.error(`FFmpeg error [${this.producerId}]:`, err);
    });
  }
}
```

## 4. 混流策略

### 4.1 布局模式

```typescript
// core/Mixer.ts
enum LayoutMode {
  GRID = 'grid',          // 网格布局
  SPEAKER = 'speaker',     // 主讲者模式
  SIDEBAR = 'sidebar'      // 侧边栏模式
}

class StreamMixer {
  private layoutMode: LayoutMode;
  private width: number;
  private height: number;
  
  constructor(options: MixerOptions) {
    this.layoutMode = options.layout || LayoutMode.GRID;
    this.width = options.width || 1920;
    this.height = options.height || 1080;
  }
  
  async mix(inputs: string[]): Promise<string> {
    const outputPath = inputs[0].replace(/_\w+\.webm$/, '_mixed.mp4');
    
    // 构建FFmpeg滤镜
    const filter = this.buildFilter(inputs.length);
    
    const args = [
      ...inputs.flatMap(input => ['-i', input]),
      '-filter_complex', filter,
      '-c:v', 'libx264',
      '-preset', 'medium',
      '-crf', '23',
      '-c:a', 'aac',
      '-b:a', '128k',
      '-shortest',
      outputPath
    ];
    
    await execPromise(`ffmpeg ${args.join(' ')}`);
    return outputPath;
  }
  
  private buildFilter(count: number): string {
    switch (this.layoutMode) {
      case LayoutMode.GRID:
        return this.buildGridLayout(count);
      case LayoutMode.SPEAKER:
        return this.buildSpeakerLayout(count);
      default:
        return this.buildGridLayout(count);
    }
  }
  
  private buildGridLayout(count: number): string {
    // 计算网格行列数
    const cols = Math.ceil(Math.sqrt(count));
    const rows = Math.ceil(count / cols);
    const cellWidth = Math.floor(this.width / cols);
    const cellHeight = Math.floor(this.height / rows);
    
    // 缩放所有输入
    const scaleFilters = Array.from({ length: count }, (_, i) =>
      `[${i}:v]scale=${cellWidth}:${cellHeight}:force_original_aspect_ratio=decrease,pad=${cellWidth}:${cellHeight}:(ow-iw)/2:(oh-ih)/2[v${i}]`
    );
    
    // 水平堆叠
    const rowsFilters: string[] = [];
    for (let r = 0; r < rows; r++) {
      const rowInputs = [];
      for (let c = 0; c < cols && r * cols + c < count; c++) {
        rowInputs.push(`v${r * cols + c}`);
      }
      if (rowInputs.length > 1) {
        rowsFilters.push(`[${rowInputs.join('][')}]hstack=inputs=${rowInputs.length}[row${r}]`);
      } else {
        rowsFilters.push(`[${rowInputs[0]}]copy[row${r}]`);
      }
    }
    
    // 垂直堆叠
    const rowOutputs = Array.from({ length: rows }, (_, i) => `[row${i}]`);
    const vstackFilter = `${rowOutputs.join('')}vstack=inputs=${rows}[out]`;
    
    return [...scaleFilters, ...rowsFilters, vstackFilter].join(';');
  }
  
  private buildSpeakerLayout(count: number): string {
    // 主讲者占大部分，其他人显示在底部
    const mainWidth = Math.floor(this.width * 0.75);
    const mainHeight = this.height;
    const thumbWidth = Math.floor(this.width * 0.25);
    const thumbHeight = Math.floor(this.height / (count - 1));
    
    const filters = [
      `[0:v]scale=${mainWidth}:${mainHeight}[main]`
    ];
    
    for (let i = 1; i < count; i++) {
      filters.push(
        `[${i}:v]scale=${thumbWidth}:${thumbHeight}[thumb${i}]`
      );
    }
    
    const thumbInputs = Array.from({ length: count - 1 }, (_, i) => `[thumb${i + 1}]`);
    filters.push(`${thumbInputs.join('')}vstack=inputs=${count - 1}[thumbs]`);
    filters.push(`[main][thumbs]hstack[out]`);
    
    return filters.join(';');
  }
}
```

## 5. API接口

```typescript
// api/recording.routes.ts
import { Router } from 'express';

const router = Router();

// 开始录制
router.post('/start', authMiddleware, async (req, res) => {
  const { meetingId } = req.body;
  
  // 验证权限（只有主持人可以录制）
  const meeting = await Meeting.findOne({ meetingId });
  if (!meeting || meeting.hostId.toString() !== req.user!.userId) {
    return res.status(403).json({ success: false, message: '无权限' });
  }
  
  const session = await recordingManager.startRecording(meetingId, {
    layout: LayoutMode.GRID,
    quality: 'high'
  });
  
  res.json({
    success: true,
    data: { sessionId: session.sessionId }
  });
});

// 停止录制
router.post('/stop', authMiddleware, async (req, res) => {
  const { meetingId } = req.body;
  
  const result = await recordingManager.stopRecording(meetingId);
  
  // 保存录制记录
  await Recording.create({
    meetingId,
    sessionId: result.sessionId,
    initiatedBy: req.user!.userId,
    status: 'completed',
    startTime: result.startTime,
    endTime: new Date(),
    duration: result.duration,
    fileUrl: result.fileUrl,
    fileSize: result.fileSize,
    thumbnailUrl: result.thumbnailUrl
  });
  
  res.json({ success: true, data: result });
});

// 暂停录制
router.post('/pause', authMiddleware, async (req, res) => {
  const { meetingId } = req.body;
  await recordingManager.pauseRecording(meetingId);
  res.json({ success: true });
});

// 恢复录制
router.post('/resume', authMiddleware, async (req, res) => {
  const { meetingId } = req.body;
  await recordingManager.resumeRecording(meetingId);
  res.json({ success: true });
});

// 获取录制状态
router.get('/status/:meetingId', authMiddleware, async (req, res) => {
  const status = recordingManager.getRecordingStatus(req.params.meetingId);
  res.json({ success: true, data: status });
});

// 获取录制列表
router.get('/list/:meetingId', authMiddleware, async (req, res) => {
  const recordings = await Recording.find({ meetingId: req.params.meetingId })
    .sort({ createdAt: -1 });
  res.json({ success: true, data: recordings });
});

// 下载录制文件
router.get('/download/:recordingId', authMiddleware, async (req, res) => {
  const recording = await Recording.findById(req.params.recordingId);
  if (!recording) {
    return res.status(404).json({ success: false, message: '录制不存在' });
  }
  
  // 返回预签名URL（S3/MinIO）
  const url = await storage.getSignedUrl(recording.fileUrl);
  res.json({ success: true, data: { url } });
});

export default router;
```
