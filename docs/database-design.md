# 数据库详细设计

## 1. MongoDB集合设计

### 1.1 用户集合 (users)

```javascript
{
  _id: ObjectId("..."),
  username: "张三",                          // 用户名
  email: "zhangsan@example.com",             // 邮箱（唯一）
  password: "$2b$12$...",                    // bcrypt加密密码
  avatar: "https://cdn.example.com/avatars/xxx.jpg",  // 头像URL
  createdAt: ISODate("2024-01-01T00:00:00Z"),
  updatedAt: ISODate("2024-01-01T00:00:00Z")
}

// 索引
db.users.createIndex({ email: 1 }, { unique: true })
db.users.createIndex({ username: 1 }, { unique: true })
```

### 1.2 会议集合 (meetings)

```javascript
{
  _id: ObjectId("..."),
  meetingId: "123456",                       // 6位会议号（唯一）
  title: "产品评审会议",                      // 会议标题
  description: "讨论Q1产品规划",              // 会议描述
  hostId: ObjectId("..."),                   // 主持人ID
  password: "abc123",                        // 会议密码（可选）
  settings: {
    maxParticipants: 200,                    // 最大参与人数
    enableWaitingRoom: false,                // 启用等候室
    enableRecording: true,                   // 允许录制
    allowScreenShare: true,                  // 允许屏幕共享
    allowChat: true,                         // 允许聊天
    allowWhiteboard: true,                   // 允许白板
    muteOnEntry: false,                      // 入会静音
    videoOnEntry: true                       // 入会开启视频
  },
  status: "active",                          // scheduled/waiting/active/ended
  scheduledStartTime: ISODate("..."),        // 计划开始时间
  actualStartTime: ISODate("..."),           // 实际开始时间
  endTime: ISODate("..."),                   // 结束时间
  participants: [                            // 参与者记录
    {
      userId: ObjectId("..."),
      role: "host",                          // host/cohost/participant
      joinedAt: ISODate("..."),
      leftAt: ISODate("..."),
      duration: 3600                         // 参与时长（秒）
    }
  ],
  createdAt: ISODate("..."),
  updatedAt: ISODate("...")
}

// 索引
db.meetings.createIndex({ meetingId: 1 }, { unique: true })
db.meetings.createIndex({ hostId: 1 })
db.meetings.createIndex({ status: 1 })
db.meetings.createIndex({ scheduledStartTime: 1 })
db.meetings.createIndex({ createdAt: -1 })
```

### 1.3 聊天记录集合 (chat_messages)

```javascript
{
  _id: ObjectId("..."),
  meetingId: "123456",                       // 会议号
  senderId: ObjectId("..."),                 // 发送者ID
  receiverId: ObjectId("..."),               // 接收者ID（私聊）
  type: "text",                              // text/file/image
  content: "大家好",                         // 消息内容
  fileName: "文档.pdf",                      // 文件名
  fileUrl: "https://cdn.example.com/...",   // 文件URL
  fileSize: 1024000,                         // 文件大小
  mimeType: "application/pdf",               // MIME类型
  timestamp: ISODate("...")                  // 发送时间
}

// 索引
db.chat_messages.createIndex({ meetingId: 1, timestamp: -1 })
db.chat_messages.createIndex({ senderId: 1 })
db.chat_messages.createIndex({ receiverId: 1 })
// TTL索引：30天后自动删除
db.chat_messages.createIndex({ timestamp: 1 }, { expireAfterSeconds: 2592000 })
```

### 1.4 录制记录集合 (recordings)

```javascript
{
  _id: ObjectId("..."),
  meetingId: "123456",                       // 会议号
  title: "产品评审会议-录制",                 // 录制标题
  initiatedBy: ObjectId("..."),              // 发起录制的用户
  status: "completed",                       // recording/completed/failed
  startTime: ISODate("..."),                 // 开始录制时间
  endTime: ISODate("..."),                   // 结束录制时间
  duration: 3600,                            // 录制时长（秒）
  fileUrl: "https://cdn.example.com/recordings/xxx.mp4",  // 文件URL
  fileSize: 104857600,                       // 文件大小（字节）
  thumbnailUrl: "https://cdn.example.com/thumbnails/xxx.jpg",  // 缩略图
  createdAt: ISODate("..."),
  updatedAt: ISODate("...")
}

// 索引
db.recordings.createIndex({ meetingId: 1 })
db.recordings.createIndex({ initiatedBy: 1 })
db.recordings.createIndex({ createdAt: -1 })
```

## 2. Redis键值设计

### 2.1 在线用户状态

```redis
# 用户Socket映射
# Key: user:socket:{userId}
# Value: socketId
# TTL: 24小时
SET user:socket:507f1f77bcf86cd799439011 "abc123" EX 86400

# 用户在线状态
# Key: user:status:{userId}
# Value: {"online": true, "meetingId": "123456", "lastSeen": 1704067200}
# TTL: 24小时
SET user:status:507f1f77bcf86cd799439011 '{"online":true,"meetingId":"123456","lastSeen":1704067200}' EX 86400
```

### 2.2 房间状态

```redis
# 房间参与者集合
# Key: room:{meetingId}:peers
# Type: Hash
# Field: peerId
# Value: JSON序列化的Peer信息
HSET room:123456:peers peer1 '{"userId":"...","username":"张三","role":"host","audioEnabled":true,"videoEnabled":true}'
HSET room:123456:peers peer2 '{"userId":"...","username":"李四","role":"participant","audioEnabled":false,"videoEnabled":true}'

# 房间参与者计数
# Key: room:{meetingId}:peerCount
# Value: 整数
SET room:123456:peerCount 15

# 房间Producer列表
# Key: room:{meetingId}:producers
# Type: Hash
# Field: producerId
# Value: JSON序列化的Producer信息
HSET room:123456:producers prod1 '{"peerId":"peer1","kind":"audio","appData":{}}'
HSET room:123456:producers prod2 '{"peerId":"peer1","kind":"video","appData":{}}'

# 房间Consumer映射
# Key: room:{meetingId}:consumers:{peerId}
# Type: Set
SADD room:123456:consumers:peer1 cons1 cons2 cons3

# 在线会议集合
SADD online:meetings 123456 789012
```

### 2.3 Transport映射

```redis
# Transport信息
# Key: transport:{transportId}
# Value: {"peerId":"peer1","meetingId":"123456","direction":"send"}
# TTL: 1小时
SET transport:abc123 '{"peerId":"peer1","meetingId":"123456","direction":"send"}' EX 3600
```

### 2.4 白板数据

```redis
# 白板快照
# Key: whiteboard:{meetingId}
# Value: JSON序列化的白板对象数组
SET whiteboard:123456 '{"objects":[...],"version":42}'

# 白板操作队列（可选，用于实时同步）
# Key: whiteboard:{meetingId}:operations
# Type: List
LPUSH whiteboard:123456:operations '{"id":"op1","type":"create","objectId":"obj1","data":{}}'
LTRIM whiteboard:123456:operations 0 999  # 保留最新1000条
```

### 2.5 限流计数器

```redis
# API限流
# Key: ratelimit:{ip}:{endpoint}
# Value: 请求次数
# TTL: 60秒
INCR ratelimit:192.168.1.1:/api/meetings
EXPIRE ratelimit:192.168.1.1:/api/meetings 60

# Socket连接限流
# Key: ratelimit:socket:{socketId}:{event}
INCR ratelimit:socket:abc123:send-message
EXPIRE ratelimit:socket:abc123:send-message 1
```

## 3. 数据关系图

```
┌─────────────────────────────────────────────────────────────────┐
│                          MongoDB                                  │
│                                                                   │
│  ┌─────────────┐         ┌─────────────────────┐                │
│  │    Users     │         │      Meetings        │                │
│  │             │         │                      │                │
│  │ _id         │◄────────│ hostId              │                │
│  │ username    │    ┌───►│ participants.userId │                │
│  │ email       │    │    │                     │                │
│  │ password    │    │    │ meetingId ──────────┼───┐            │
│  │ avatar      │    │    │ title               │   │            │
│  └─────────────┘    │    │ settings            │   │            │
│                     │    │ status              │   │            │
│                     │    └─────────────────────┘   │            │
│                     │                              │            │
│  ┌─────────────┐    │    ┌─────────────────────┐   │            │
│  │   Chat      │    │    │    Recordings       │   │            │
│  │  Messages   │    │    │                      │   │            │
│  │             │    │    │ _id                  │   │            │
│  │ meetingId ──┼────┼────┼─ meetingId          │   │            │
│  │ senderId ──┼┘   │    │ initiatedBy          │   │            │
│  │ content    │    │    │ fileUrl              │   │            │
│  │ timestamp  │    │    │ duration             │   │            │
│  └─────────────┘    │    └─────────────────────┘   │            │
│                     │                              │            │
└─────────────────────┴──────────────────────────────┴────────────┘
                      │
┌─────────────────────┴────────────────────────────────────────────┐
│                          Redis                                     │
│                                                                    │
│  ┌──────────────────────────────────────────────────────────────┐│
│  │  room:{meetingId}:peers      - 房间在线参与者                  ││
│  │  room:{meetingId}:producers  - 房间媒体生产者                  ││
│  │  room:{meetingId}:consumers:{peerId} - 参与者消费者           ││
│  │  user:socket:{userId}        - 用户Socket映射                 ││
│  │  user:status:{userId}        - 用户在线状态                   ││
│  │  transport:{transportId}     - Transport映射                 ││
│  │  whiteboard:{meetingId}      - 白板快照                      ││
│  │  online:meetings             - 在线会议集合                   ││
│  └──────────────────────────────────────────────────────────────┘│
│                                                                    │
└────────────────────────────────────────────────────────────────────┘
```

## 4. 数据备份策略

### 4.1 MongoDB备份

```bash
# 每日全量备份
mongodump --uri="mongodb://localhost:27017/meeting" --out=/backup/$(date +%Y%m%d)

# 增量备份（使用Oplog）
mongodump --uri="mongodb://localhost:27017/local" --collection=oplog.rs --out=/backup/oplog

# 恢复
mongorestore --uri="mongodb://localhost:27017/meeting" /backup/20240101/meeting
```

### 4.2 Redis持久化

```redis
# AOF持久化配置
appendonly yes
appendfsync everysec

# RDB快照配置
save 900 1
save 300 10
save 60 10000
```
