# 前端模块详细设计

## 1. 项目结构

```
client/
├── public/
│   ├── index.html
│   └── favicon.ico
├── src/
│   ├── assets/                    # 静态资源
│   │   ├── images/
│   │   ├── icons/
│   │   └── styles/
│   │       ├── global.scss
│   │       ├── variables.scss
│   │       └── mixins.scss
│   │
│   ├── components/                # 通用组件
│   │   ├── common/                # 基础通用
│   │   │   ├── Avatar/
│   │   │   ├── Button/
│   │   │   ├── Modal/
│   │   │   ├── Tooltip/
│   │   │   └── Loading/
│   │   │
│   │   ├── VideoGrid/             # 视频网格
│   │   │   ├── index.tsx
│   │   │   ├── VideoTile.tsx
│   │   │   ├── VideoControls.tsx
│   │   │   └── useVideoLayout.ts
│   │   │
│   │   ├── ChatPanel/             # 聊天面板
│   │   │   ├── index.tsx
│   │   │   ├── MessageList.tsx
│   │   │   ├── MessageInput.tsx
│   │   │   ├── MessageItem.tsx
│   │   │   └── FilePreview.tsx
│   │   │
│   │   ├── Whiteboard/            # 白板组件
│   │   │   ├── index.tsx
│   │   │   ├── Toolbar.tsx
│   │   │   ├── Canvas.tsx
│   │   │   └── useWhiteboard.ts
│   │   │
│   │   ├── ParticipantList/        # 参与者列表
│   │   │   ├── index.tsx
│   │   │   ├── ParticipantItem.tsx
│   │   │   └── ParticipantActions.tsx
│   │   │
│   │   ├── ScreenShare/           # 屏幕共享
│   │   │   ├── index.tsx
│   │   │   └── ScreenPreview.tsx
│   │   │
│   │   ├── Toolbar/               # 工具栏
│   │   │   ├── index.tsx
│   │   │   ├── MediaControls.tsx
│   │   │   ├── ActionButtons.tsx
│   │   │   └── MeetingInfo.tsx
│   │   │
│   │   ├── WaitingRoom/           # 等候室
│   │   │   ├── index.tsx
│   │   │   └── WaitingList.tsx
│   │   │
│   │   └── Recording/             # 录制控制
│   │       ├── index.tsx
│   │       └── RecordingIndicator.tsx
│   │
│   ├── hooks/                     # 自定义Hooks
│   │   ├── useWebRTC.ts          # WebRTC连接管理
│   │   ├── useMedia.ts           # 媒体设备管理
│   │   ├── useSocket.ts          # Socket连接
│   │   ├── useMeeting.ts         # 会议状态
│   │   ├── useChat.ts            # 聊天功能
│   │   ├── useScreenShare.ts     # 屏幕共享
│   │   ├── useRecording.ts       # 录制控制
│   │   └── usePermission.ts      # 权限管理
│   │
│   ├── stores/                    # Zustand状态
│   │   ├── userStore.ts
│   │   ├── meetingStore.ts
│   │   ├── mediaStore.ts
│   │   ├── chatStore.ts
│   │   └── whiteboardStore.ts
│   │
│   ├── pages/                     # 页面
│   │   ├── Home/                  # 首页
│   │   │   ├── index.tsx
│   │   │   ├── CreateMeeting.tsx
│   │   │   └── JoinMeeting.tsx
│   │   │
│   │   ├── Meeting/               # 会议页
│   │   │   ├── index.tsx
│   │   │   ├── MeetingRoom.tsx
│   │   │   └── MeetingLayout.tsx
│   │   │
│   │   ├── Lobby/                 # 等候室
│   │   │   └── index.tsx
│   │   │
│   │   └── Login/                 # 登录
│   │       └── index.tsx
│   │
│   ├── services/                  # API服务
│   │   ├── api.ts                 # Axios实例
│   │   ├── auth.ts                # 认证接口
│   │   ├── meeting.ts             # 会议接口
│   │   ├── user.ts                # 用户接口
│   │   └── upload.ts              # 上传接口
│   │
│   ├── socket/                    # Socket管理
│   │   ├── index.ts               # Socket实例
│   │   ├── handlers.ts            # 事件处理
│   │   └── events.ts              # 事件定义
│   │
│   ├── types/                     # TypeScript类型
│   │   ├── meeting.ts
│   │   ├── user.ts
│   │   ├── media.ts
│   │   ├── chat.ts
│   │   └── socket.ts
│   │
│   ├── utils/                     # 工具函数
│   │   ├── media.ts               # 媒体工具
│   │   ├── storage.ts             # 本地存储
│   │   ├── format.ts              # 格式化
│   │   └── device.ts              # 设备检测
│   │
│   ├── App.tsx
│   ├── main.tsx
│   └── vite-env.d.ts
│
├── tests/                         # 测试文件
├── .env.development
├── .env.production
├── index.html
├── package.json
├── tsconfig.json
└── vite.config.ts
```

## 2. 核心状态管理设计

### 2.1 UserStore

```typescript
// stores/userStore.ts
interface UserState {
  // 状态
  currentUser: User | null;
  isAuthenticated: boolean;
  token: string | null;
  
  // 操作
  login: (credentials: LoginCredentials) => Promise<void>;
  logout: () => void;
  updateProfile: (data: Partial<User>) => void;
}

interface User {
  id: string;
  username: string;
  email: string;
  avatar: string;
}
```

### 2.2 MeetingStore

```typescript
// stores/meetingStore.ts
interface MeetingState {
  // 状态
  meetingId: string | null;
  meetingInfo: MeetingInfo | null;
  participants: Map<string, Participant>;
  localParticipant: LocalParticipant | null;
  
  // 媒体状态
  localAudioEnabled: boolean;
  localVideoEnabled: boolean;
  isScreenSharing: boolean;
  
  // UI状态
  activePanel: 'chat' | 'participants' | 'whiteboard' | null;
  layoutMode: 'grid' | 'speaker' | 'gallery';
  
  // 操作
  joinMeeting: (meetingId: string, password?: string) => Promise<void>;
  leaveMeeting: () => void;
  toggleAudio: () => void;
  toggleVideo: () => void;
  setActivePanel: (panel: MeetingState['activePanel']) => void;
  setLayoutMode: (mode: MeetingState['layoutMode']) => void;
}

interface MeetingInfo {
  id: string;
  title: string;
  hostId: string;
  startTime: Date;
  settings: MeetingSettings;
}

interface Participant {
  id: string;
  userId: string;
  username: string;
  avatar: string;
  role: 'host' | 'cohost' | 'participant';
  audioEnabled: boolean;
  videoEnabled: boolean;
  isScreenSharing: boolean;
  isHandRaised: boolean;
}

interface LocalParticipant extends Participant {
  stream: MediaStream | null;
  screenStream: MediaStream | null;
}
```

### 2.3 MediaStore

```typescript
// stores/mediaStore.ts
interface MediaState {
  // 设备状态
  audioDevices: MediaDeviceInfo[];
  videoDevices: MediaDeviceInfo[];
  selectedAudioDevice: string | null;
  selectedVideoDevice: string | null;
  
  // 媒体流
  producers: Map<string, Producer>;        // 本地生产的流
  consumers: Map<string, Consumer>;        // 远程消费的流
  
  // 操作
  initDevices: () => Promise<void>;
  selectAudioDevice: (deviceId: string) => void;
  selectVideoDevice: (deviceId: string) => void;
  createProducer: (kind: 'audio' | 'video') => Promise<void>;
  closeProducer: (producerId: string) => void;
}
```

## 3. 核心Hook设计

### 3.1 useWebRTC

```typescript
// hooks/useWebRTC.ts
/**
 * WebRTC核心Hook
 * 负责管理与SFU的WebRTC连接，包括Transport创建、Producer/Consumer管理
 */
interface UseWebRTC {
  // 状态
  device: mediasoupClient.types.Device | null;
  sendTransport: Transport | null;
  recvTransport: Transport | null;
  producers: Map<string, Producer>;
  consumers: Map<string, Consumer>;
  
  // 方法
  loadDevice: (routerRtpCapabilities: RtpCapabilities) => Promise<void>;
  createSendTransport: (params: TransportParams) => Promise<void>;
  createRecvTransport: (params: TransportParams) => Promise<void>;
  produce: (track: MediaStreamTrack, options?: ProducerOptions) => Promise<Producer>;
  consume: (params: ConsumeParams) => Promise<Consumer>;
  closeTransport: (direction: 'send' | 'recv') => void;
  closeProducer: (producerId: string) => void;
  closeConsumer: (consumerId: string) => void;
}

// 内部流程
// 1. 加载mediasoup Device
// 2. 创建Send Transport (用于发布本地媒体)
// 3. 创建Recv Transport (用于订阅远程媒体)
// 4. 通过Transport创建Producer发布音视频
// 5. 收到newProducer事件后创建Consumer订阅
```

### 3.2 useMedia

```typescript
// hooks/useMedia.ts
/**
 * 媒体设备管理Hook
 * 负责获取和管理本地媒体流
 */
interface UseMedia {
  // 状态
  localStream: MediaStream | null;
  audioEnabled: boolean;
  videoEnabled: boolean;
  audioLevel: number;
  
  // 方法
  initStream: (constraints?: MediaStreamConstraints) => Promise<MediaStream>;
  toggleAudio: () => void;
  toggleVideo: () => void;
  switchAudioDevice: (deviceId: string) => Promise<void>;
  switchVideoDevice: (deviceId: string) => Promise<void>;
  getDisplayMedia: () => Promise<MediaStream>;
  stopStream: () => void;
}

// 设备约束配置
const defaultConstraints: MediaStreamConstraints = {
  audio: {
    echoCancellation: true,
    noiseSuppression: true,
    autoGainControl: true,
    sampleRate: 48000
  },
  video: {
    width: { ideal: 1280 },
    height: { ideal: 720 },
    frameRate: { ideal: 30 }
  }
};
```

### 3.3 useSocket

```typescript
// hooks/useSocket.ts
/**
 * Socket连接管理Hook
 */
interface UseSocket {
  // 状态
  socket: Socket | null;
  connected: boolean;
  
  // 方法
  connect: (token: string) => void;
  disconnect: () => void;
  emit: <T>(event: string, data: T, callback?: Function) => void;
  on: <T>(event: string, handler: (data: T) => void) => void;
  off: (event: string, handler?: Function) => void;
}

// Socket事件定义
enum SocketEvent {
  // 房间事件
  JOIN_ROOM = 'join-room',
  LEAVE_ROOM = 'leave-room',
  ROOM_JOINED = 'room-joined',
  PARTICIPANT_JOINED = 'participant-joined',
  PARTICIPANT_LEFT = 'participant-left',
  
  // WebRTC信令
  GET_ROUTER_RTP_CAPABILITIES = 'get-router-rtp-capabilities',
  CREATE_TRANSPORT = 'create-transport',
  CONNECT_TRANSPORT = 'connect-transport',
  PRODUCE = 'produce',
  CONSUME = 'consume',
  PRODUCER_CLOSED = 'producer-closed',
  NEW_PRODUCER = 'new-producer',
  
  // 媒体控制
  TOGGLE_AUDIO = 'toggle-audio',
  TOGGLE_VIDEO = 'toggle-video',
  START_SCREEN_SHARE = 'start-screen-share',
  STOP_SCREEN_SHARE = 'stop-screen-share',
  
  // 聊天
  SEND_MESSAGE = 'send-message',
  NEW_MESSAGE = 'new-message',
  
  // 白板
  WHITEBOARD_UPDATE = 'whiteboard-update',
  WHITEBOARD_SYNC = 'whiteboard-sync',
  
  // 会议控制
  MUTE_PARTICIPANT = 'mute-participant',
  REMOVE_PARTICIPANT = 'remove-participant',
  RAISE_HAND = 'raise-hand',
  
  // 录制
  START_RECORDING = 'start-recording',
  STOP_RECORDING = 'stop-recording',
  RECORDING_STATUS = 'recording-status'
}
```

### 3.4 useChat

```typescript
// hooks/useChat.ts
interface UseChat {
  // 状态
  messages: ChatMessage[];
  unreadCount: number;
  
  // 方法
  sendMessage: (content: string, type?: 'text' | 'file') => void;
  sendFile: (file: File) => Promise<void>;
  markAsRead: () => void;
  loadHistory: (before?: Date) => Promise<void>;
}

interface ChatMessage {
  id: string;
  senderId: string;
  senderName: string;
  senderAvatar: string;
  content: string;
  type: 'text' | 'file' | 'image';
  fileUrl?: string;
  fileName?: string;
  timestamp: Date;
  isPrivate: boolean;
  receiverId?: string;
}
```

## 4. 视频网格布局设计

### 4.1 自适应布局算法

```typescript
// components/VideoGrid/useVideoLayout.ts
/**
 * 视频网格布局Hook
 * 根据参与者数量自动计算最优布局
 */
interface VideoLayout {
  rows: number;
  cols: number;
  tileWidth: number;
  tileHeight: number;
  visibleParticipants: string[];
  overflowCount: number;
}

// 布局策略
const layoutStrategies = {
  1: { rows: 1, cols: 1 },      // 1人
  2: { rows: 1, cols: 2 },      // 2人
  4: { rows: 2, cols: 2 },      // 3-4人
  6: { rows: 2, cols: 3 },      // 5-6人
  9: { rows: 3, cols: 3 },      // 7-9人
  12: { rows: 3, cols: 4 },     // 10-12人
  16: { rows: 4, cols: 4 },     // 13-16人
  // 超过16人使用分页或滚动
};

// 虚拟滚动实现（50-200人场景）
// 只渲染可见区域的视频流，其余显示占位符
```

### 4.2 视频Tile组件

```typescript
// components/VideoGrid/VideoTile.tsx
interface VideoTileProps {
  participant: Participant;
  stream?: MediaStream;
  isLocal: boolean;
  isPinned: boolean;
  isSpeaking: boolean;
  onPin: () => void;
  onMute: () => void;
}

// 功能：
// - 视频渲染（使用video元素）
// - 音频指示器
// - 用户名显示
// - 静音/关闭摄像头状态图标
// - 右键菜单（静音、移除、设为主持人）
// - 拖拽排序（画廊模式）
```

## 5. 聊天模块设计

### 5.1 消息类型

```typescript
enum MessageType {
  TEXT = 'text',
  FILE = 'file',
  IMAGE = 'image',
  SYSTEM = 'system'    // 系统消息（加入/离开等）
}

interface FileMessage {
  fileName: string;
  fileSize: number;
  fileUrl: string;
  mimeType: string;
}
```

### 5.2 聊天功能点

- [x] 发送/接收文字消息
- [x] 发送/接收文件（拖拽上传）
- [x] 发送表情
- [x] @某人
- [x] 私聊
- [x] 消息历史记录
- [x] 未读消息提醒
- [x] 消息搜索

## 6. 白板模块设计

### 6.1 使用tldraw

```typescript
// components/Whiteboard/index.tsx
// 使用tldraw作为白板基础，二次开发实现协作

interface WhiteboardProps {
  meetingId: string;
  isReadOnly: boolean;
}

// 协作机制：
// 1. 每个操作生成Operation对象
// 2. 通过Socket广播到其他客户端
// 3. 其他客户端应用Operation
// 4. 定期生成Snapshot保存到Redis
// 5. 新加入用户加载最新Snapshot + 后续Operation
```

### 6.2 操作同步

```typescript
interface WhiteboardOperation {
  id: string;
  type: 'create' | 'update' | 'delete';
  objectId: string;
  data: any;
  userId: string;
  timestamp: number;
}

interface WhiteboardSnapshot {
  id: string;
  meetingId: string;
  objects: any[];
  version: number;
  createdAt: Date;
}
```

## 7. 性能优化策略

### 7.1 大房间优化

```typescript
// 视频订阅优化
class VideoSubscriptionManager {
  private maxVisibleVideos = 16;  // 最大同时显示视频数
  private speakingParticipants: Set<string> = new Set();
  private pinnedParticipants: Set<string> = new Set();
  
  // 根据以下优先级决定订阅哪些视频：
  // 1. 本地用户（始终订阅）
  // 2. 当前说话的人
  // 3. 被固定的人
  // 4. 视频网格可见的人
  // 5. 最近活跃的人
  
  getSubscriptions(): string[] {
    // 返回需要订阅的participantId列表
  }
}
```

### 7.2 渲染优化

- 使用React.memo避免不必要的重渲染
- 视频元素使用requestAnimationFrame控制帧率
- 大列表使用虚拟滚动
- 使用Web Worker处理复杂计算

### 7.3 网络优化

- 视频simulcast自动切换分辨率
- 音频始终订阅，视频按需订阅
- 断线自动重连
- 弱网环境降级策略
