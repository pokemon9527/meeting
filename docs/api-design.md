# 业务API服务器详细设计

## 1. 项目结构

```
server/api/
├── src/
│   ├── config/
│   │   ├── index.ts              # 配置入口
│   │   ├── database.ts           # 数据库配置
│   │   ├── redis.ts              # Redis配置
│   │   ├── jwt.ts                # JWT配置
│   │   └── upload.ts             # 上传配置
│   │
│   ├── models/
│   │   ├── index.ts              # 模型导出
│   │   ├── User.model.ts         # 用户模型
│   │   ├── Meeting.model.ts      # 会议模型
│   │   ├── ChatMessage.model.ts  # 聊天记录模型
│   │   ├── Whiteboard.model.ts   # 白板数据模型
│   │   └── Recording.model.ts    # 录制记录模型
│   │
│   ├── routes/
│   │   ├── index.ts              # 路由注册
│   │   ├── auth.routes.ts        # 认证路由
│   │   ├── user.routes.ts        # 用户路由
│   │   ├── meeting.routes.ts     # 会议路由
│   │   ├── chat.routes.ts        # 聊天路由
│   │   ├── upload.routes.ts      # 上传路由
│   │   └── recording.routes.ts   # 录制路由
│   │
│   ├── controllers/
│   │   ├── auth.controller.ts    # 认证控制器
│   │   ├── user.controller.ts    # 用户控制器
│   │   ├── meeting.controller.ts # 会议控制器
│   │   ├── chat.controller.ts    # 聊天控制器
│   │   ├── upload.controller.ts  # 上传控制器
│   │   └── recording.controller.ts# 录制控制器
│   │
│   ├── services/
│   │   ├── auth.service.ts       # 认证服务
│   │   ├── user.service.ts       # 用户服务
│   │   ├── meeting.service.ts    # 会议服务
│   │   ├── chat.service.ts       # 聊天服务
│   │   ├── upload.service.ts     # 上传服务
│   │   └── recording.service.ts  # 录制服务
│   │
│   ├── middlewares/
│   │   ├── auth.middleware.ts     # JWT认证中间件
│   │   ├── validate.middleware.ts # 请求验证中间件
│   │   ├── rateLimit.middleware.ts# 限流中间件
│   │   ├── error.middleware.ts    # 错误处理中间件
│   │   └── upload.middleware.ts   # 文件上传中间件
│   │
│   ├── validators/
│   │   ├── auth.validator.ts     # 认证验证
│   │   ├── meeting.validator.ts  # 会议验证
│   │   └── chat.validator.ts     # 聊天验证
│   │
│   ├── types/
│   │   ├── express.d.ts          # Express类型扩展
│   │   ├── user.ts               # 用户类型
│   │   └── meeting.ts            # 会议类型
│   │
│   ├── utils/
│   │   ├── response.ts           # 响应工具
│   │   ├── helpers.ts            # 工具函数
│   │   └── logger.ts             # 日志工具
│   │
│   └── index.ts                  # 入口文件
│
├── package.json
└── tsconfig.json
```

## 2. 数据模型设计

### 2.1 User模型

```typescript
// models/User.model.ts
import mongoose, { Schema, Document } from 'mongoose';

export interface IUser extends Document {
  username: string;
  email: string;
  password: string;
  avatar: string;
  createdAt: Date;
  updatedAt: Date;
  comparePassword(candidatePassword: string): Promise<boolean>;
}

const UserSchema = new Schema<IUser>({
  username: {
    type: String,
    required: true,
    unique: true,
    trim: true,
    minlength: 2,
    maxlength: 30
  },
  email: {
    type: String,
    required: true,
    unique: true,
    lowercase: true,
    trim: true
  },
  password: {
    type: String,
    required: true,
    minlength: 6
  },
  avatar: {
    type: String,
    default: ''
  }
}, {
  timestamps: true
});

// 密码加密
UserSchema.pre('save', async function(next) {
  if (!this.isModified('password')) return next();
  this.password = await bcrypt.hash(this.password, 12);
  next();
});

// 密码比较
UserSchema.methods.comparePassword = async function(candidatePassword: string): Promise<boolean> {
  return bcrypt.compare(candidatePassword, this.password);
};

// 隐藏敏感字段
UserSchema.methods.toJSON = function() {
  const obj = this.toObject();
  delete obj.password;
  return obj;
};

export const User = mongoose.model<IUser>('User', UserSchema);
```

### 2.2 Meeting模型

```typescript
// models/Meeting.model.ts
import mongoose, { Schema, Document } from 'mongoose';

export interface IMeeting extends Document {
  meetingId: string;           // 6位会议号
  title: string;
  description: string;
  hostId: mongoose.Types.ObjectId;
  password: string | null;
  settings: MeetingSettings;
  status: MeetingStatus;
  scheduledStartTime?: Date;
  actualStartTime?: Date;
  endTime?: Date;
  participants: IParticipant[];
  createdAt: Date;
  updatedAt: Date;
}

export interface MeetingSettings {
  maxParticipants: number;
  enableWaitingRoom: boolean;
  enableRecording: boolean;
  allowScreenShare: boolean;
  allowChat: boolean;
  allowWhiteboard: boolean;
  muteOnEntry: boolean;
  videoOnEntry: boolean;
}

export enum MeetingStatus {
  SCHEDULED = 'scheduled',
  WAITING = 'waiting',
  ACTIVE = 'active',
  ENDED = 'ended'
}

export interface IParticipant {
  userId: mongoose.Types.ObjectId;
  role: 'host' | 'cohost' | 'participant';
  joinedAt: Date;
  leftAt?: Date;
  duration: number;
}

const MeetingSchema = new Schema<IMeeting>({
  meetingId: {
    type: String,
    required: true,
    unique: true,
    index: true
  },
  title: {
    type: String,
    required: true,
    maxlength: 100
  },
  description: {
    type: String,
    maxlength: 500
  },
  hostId: {
    type: Schema.Types.ObjectId,
    ref: 'User',
    required: true
  },
  password: {
    type: String,
    default: null
  },
  settings: {
    maxParticipants: { type: Number, default: 200 },
    enableWaitingRoom: { type: Boolean, default: false },
    enableRecording: { type: Boolean, default: true },
    allowScreenShare: { type: Boolean, default: true },
    allowChat: { type: Boolean, default: true },
    allowWhiteboard: { type: Boolean, default: true },
    muteOnEntry: { type: Boolean, default: false },
    videoOnEntry: { type: Boolean, default: true }
  },
  status: {
    type: String,
    enum: Object.values(MeetingStatus),
    default: MeetingStatus.SCHEDULED
  },
  scheduledStartTime: Date,
  actualStartTime: Date,
  endTime: Date,
  participants: [{
    userId: { type: Schema.Types.ObjectId, ref: 'User' },
    role: { type: String, enum: ['host', 'cohost', 'participant'] },
    joinedAt: Date,
    leftAt: Date,
    duration: Number
  }]
}, {
  timestamps: true
});

// 生成6位随机会议号
MeetingSchema.statics.generateMeetingId = async function(): Promise<string> {
  let meetingId: string;
  let exists: boolean;
  
  do {
    meetingId = Math.floor(100000 + Math.random() * 900000).toString();
    exists = await this.exists({ meetingId });
  } while (exists);
  
  return meetingId;
};

export const Meeting = mongoose.model<IMeeting>('Meeting', MeetingSchema);
```

### 2.3 ChatMessage模型

```typescript
// models/ChatMessage.model.ts
import mongoose, { Schema, Document } from 'mongoose';

export interface IChatMessage extends Document {
  meetingId: string;
  senderId: mongoose.Types.ObjectId;
  receiverId?: mongoose.Types.ObjectId;  // 私聊时有值
  type: 'text' | 'file' | 'image';
  content: string;
  fileName?: string;
  fileUrl?: string;
  fileSize?: number;
  mimeType?: string;
  timestamp: Date;
}

const ChatMessageSchema = new Schema<IChatMessage>({
  meetingId: {
    type: String,
    required: true,
    index: true
  },
  senderId: {
    type: Schema.Types.ObjectId,
    ref: 'User',
    required: true
  },
  receiverId: {
    type: Schema.Types.ObjectId,
    ref: 'User',
    default: null
  },
  type: {
    type: String,
    enum: ['text', 'file', 'image'],
    default: 'text'
  },
  content: {
    type: String,
    required: true,
    maxlength: 5000
  },
  fileName: String,
  fileUrl: String,
  fileSize: Number,
  mimeType: String,
  timestamp: {
    type: Date,
    default: Date.now,
    index: true
  }
});

// 按会议和时间索引
ChatMessageSchema.index({ meetingId: 1, timestamp: -1 });

export const ChatMessage = mongoose.model<IChatMessage>('ChatMessage', ChatMessageSchema);
```

## 3. 路由设计

### 3.1 API接口列表

```typescript
// routes/index.ts
import { Router } from 'express';

export function createApiRouter(): Router {
  const router = Router();
  
  // 认证相关
  router.use('/auth', authRoutes);
  
  // 用户相关
  router.use('/users', userRoutes);
  
  // 会议相关
  router.use('/meetings', meetingRoutes);
  
  // 聊天相关
  router.use('/chat', chatRoutes);
  
  // 上传相关
  router.use('/upload', uploadRoutes);
  
  // 录制相关
  router.use('/recordings', recordingRoutes);
  
  return router;
}
```

### 3.2 认证路由

```typescript
// routes/auth.routes.ts
export const authRoutes = Router();

// POST /api/auth/register - 注册
authRoutes.post('/register', 
  validateRegister,
  authController.register
);

// POST /api/auth/login - 登录
authRoutes.post('/login',
  validateLogin,
  authController.login
);

// POST /api/auth/refresh - 刷新Token
authRoutes.post('/refresh',
  authController.refreshToken
);

// POST /api/auth/logout - 登出
authRoutes.post('/logout',
  authMiddleware,
  authController.logout
);

// POST /api/auth/verify - 验证Token
authRoutes.post('/verify',
  authController.verifyToken
);
```

### 3.3 会议路由

```typescript
// routes/meeting.routes.ts
export const meetingRoutes = Router();

// 创建会议
meetingRoutes.post('/',
  authMiddleware,
  validateCreateMeeting,
  meetingController.createMeeting
);

// 加入会议（验证密码）
meetingRoutes.post('/join',
  validateJoinMeeting,
  meetingController.joinMeeting
);

// 获取会议信息
meetingRoutes.get('/:meetingId',
  authMiddleware,
  meetingController.getMeeting
);

// 获取会议列表（我的会议）
meetingRoutes.get('/',
  authMiddleware,
  meetingController.getMyMeetings
);

// 更新会议设置
meetingRoutes.put('/:meetingId/settings',
  authMiddleware,
  meetingController.updateSettings
);

// 结束会议
meetingRoutes.post('/:meetingId/end',
  authMiddleware,
  meetingController.endMeeting
);

// 获取会议参与者
meetingRoutes.get('/:meetingId/participants',
  authMiddleware,
  meetingController.getParticipants
);

// 更新参与者角色
meetingRoutes.put('/:meetingId/participants/:participantId/role',
  authMiddleware,
  meetingController.updateParticipantRole
);

// 删除会议
meetingRoutes.delete('/:meetingId',
  authMiddleware,
  meetingController.deleteMeeting
);
```

## 4. 控制器实现

### 4.1 认证控制器

```typescript
// controllers/auth.controller.ts
class AuthController {
  
  async register(req: Request, res: Response, next: NextFunction) {
    try {
      const { username, email, password } = req.body;
      
      // 检查用户是否存在
      const existingUser = await User.findOne({ 
        $or: [{ email }, { username }] 
      });
      if (existingUser) {
        return res.status(400).json({
          success: false,
          message: '用户已存在'
        });
      }
      
      // 创建用户
      const user = await User.create({ username, email, password });
      
      // 生成Token
      const token = generateToken(user);
      const refreshToken = generateRefreshToken(user);
      
      res.status(201).json({
        success: true,
        data: {
          user: user.toJSON(),
          token,
          refreshToken
        }
      });
    } catch (error) {
      next(error);
    }
  }
  
  async login(req: Request, res: Response, next: NextFunction) {
    try {
      const { email, password } = req.body;
      
      // 查找用户
      const user = await User.findOne({ email });
      if (!user) {
        return res.status(401).json({
          success: false,
          message: '邮箱或密码错误'
        });
      }
      
      // 验证密码
      const isMatch = await user.comparePassword(password);
      if (!isMatch) {
        return res.status(401).json({
          success: false,
          message: '邮箱或密码错误'
        });
      }
      
      // 生成Token
      const token = generateToken(user);
      const refreshToken = generateRefreshToken(user);
      
      res.json({
        success: true,
        data: {
          user: user.toJSON(),
          token,
          refreshToken
        }
      });
    } catch (error) {
      next(error);
    }
  }
  
  async refreshToken(req: Request, res: Response, next: NextFunction) {
    try {
      const { refreshToken } = req.body;
      
      const decoded = jwt.verify(refreshToken, REFRESH_SECRET) as JwtPayload;
      const user = await User.findById(decoded.userId);
      
      if (!user) {
        return res.status(401).json({
          success: false,
          message: '用户不存在'
        });
      }
      
      const newToken = generateToken(user);
      const newRefreshToken = generateRefreshToken(user);
      
      res.json({
        success: true,
        data: {
          token: newToken,
          refreshToken: newRefreshToken
        }
      });
    } catch (error) {
      next(error);
    }
  }
}
```

### 4.2 会议控制器

```typescript
// controllers/meeting.controller.ts
class MeetingController {
  
  async createMeeting(req: Request, res: Response, next: NextFunction) {
    try {
      const userId = req.user!.userId;
      const { title, description, password, settings, scheduledStartTime } = req.body;
      
      // 生成会议号
      const meetingId = await Meeting.generateMeetingId();
      
      // 创建会议
      const meeting = await Meeting.create({
        meetingId,
        title,
        description,
        hostId: userId,
        password: password || null,
        settings,
        scheduledStartTime,
        status: scheduledStartTime ? MeetingStatus.SCHEDULED : MeetingStatus.WAITING,
        participants: [{
          userId,
          role: 'host',
          joinedAt: new Date(),
          duration: 0
        }]
      });
      
      res.status(201).json({
        success: true,
        data: { meeting }
      });
    } catch (error) {
      next(error);
    }
  }
  
  async joinMeeting(req: Request, res: Response, next: NextFunction) {
    try {
      const { meetingId, password } = req.body;
      
      // 查找会议
      const meeting = await Meeting.findOne({ meetingId });
      if (!meeting) {
        return res.status(404).json({
          success: false,
          message: '会议不存在'
        });
      }
      
      // 检查会议状态
      if (meeting.status === MeetingStatus.ENDED) {
        return res.status(400).json({
          success: false,
          message: '会议已结束'
        });
      }
      
      // 验证密码
      if (meeting.password && meeting.password !== password) {
        return res.status(401).json({
          success: false,
          message: '会议密码错误'
        });
      }
      
      // 检查人数限制
      if (meeting.participants.length >= meeting.settings.maxParticipants) {
        return res.status(400).json({
          success: false,
          message: '会议人数已满'
        });
      }
      
      // 生成加入Token
      const joinToken = generateJoinToken({
        meetingId,
        meetingMongoId: meeting._id
      });
      
      res.json({
        success: true,
        data: {
          meeting: {
            meetingId: meeting.meetingId,
            title: meeting.title,
            settings: meeting.settings
          },
          token: joinToken
        }
      });
    } catch (error) {
      next(error);
    }
  }
  
  async getMeeting(req: Request, res: Response, next: NextFunction) {
    try {
      const { meetingId } = req.params;
      
      const meeting = await Meeting.findOne({ meetingId })
        .populate('hostId', 'username avatar')
        .populate('participants.userId', 'username avatar');
      
      if (!meeting) {
        return res.status(404).json({
          success: false,
          message: '会议不存在'
        });
      }
      
      res.json({
        success: true,
        data: { meeting }
      });
    } catch (error) {
      next(error);
    }
  }
  
  async updateSettings(req: Request, res: Response, next: NextFunction) {
    try {
      const { meetingId } = req.params;
      const userId = req.user!.userId;
      const settings = req.body;
      
      const meeting = await Meeting.findOne({ meetingId });
      if (!meeting) {
        return res.status(404).json({
          success: false,
          message: '会议不存在'
        });
      }
      
      // 只有主持人可以修改设置
      if (meeting.hostId.toString() !== userId) {
        return res.status(403).json({
          success: false,
          message: '无权限修改'
        });
      }
      
      meeting.settings = { ...meeting.settings, ...settings };
      await meeting.save();
      
      res.json({
        success: true,
        data: { meeting }
      });
    } catch (error) {
      next(error);
    }
  }
  
  async endMeeting(req: Request, res: Response, next: NextFunction) {
    try {
      const { meetingId } = req.params;
      const userId = req.user!.userId;
      
      const meeting = await Meeting.findOne({ meetingId });
      if (!meeting) {
        return res.status(404).json({
          success: false,
          message: '会议不存在'
        });
      }
      
      if (meeting.hostId.toString() !== userId) {
        return res.status(403).json({
          success: false,
          message: '无权限结束会议'
        });
      }
      
      meeting.status = MeetingStatus.ENDED;
      meeting.endTime = new Date();
      
      // 计算参与者时长
      meeting.participants.forEach(p => {
        if (!p.leftAt) {
          p.leftAt = new Date();
          p.duration = Math.floor((p.leftAt.getTime() - p.joinedAt.getTime()) / 1000);
        }
      });
      
      await meeting.save();
      
      // TODO: 通知信令服务器关闭房间
      
      res.json({
        success: true,
        message: '会议已结束'
      });
    } catch (error) {
      next(error);
    }
  }
}
```

## 5. 中间件设计

### 5.1 JWT认证中间件

```typescript
// middlewares/auth.middleware.ts
import { Request, Response, NextFunction } from 'express';
import jwt from 'jsonwebtoken';

declare global {
  namespace Express {
    interface Request {
      user?: {
        userId: string;
        username: string;
        avatar: string;
      };
    }
  }
}

export const authMiddleware = (req: Request, res: Response, next: NextFunction) => {
  const authHeader = req.headers.authorization;
  
  if (!authHeader || !authHeader.startsWith('Bearer ')) {
    return res.status(401).json({
      success: false,
      message: '未提供认证Token'
    });
  }
  
  const token = authHeader.split(' ')[1];
  
  try {
    const decoded = jwt.verify(token, process.env.JWT_SECRET!) as JwtPayload;
    req.user = {
      userId: decoded.userId,
      username: decoded.username,
      avatar: decoded.avatar
    };
    next();
  } catch (error) {
    if (error instanceof jwt.TokenExpiredError) {
      return res.status(401).json({
        success: false,
        message: 'Token已过期'
      });
    }
    return res.status(401).json({
      success: false,
      message: '无效的Token'
    });
  }
};
```

### 5.2 请求验证中间件

```typescript
// middlewares/validate.middleware.ts
import { Request, Response, NextFunction } from 'express';
import { validationResult, ValidationChain } from 'express-validator';

export const validate = (validations: ValidationChain[]) => {
  return async (req: Request, res: Response, next: NextFunction) => {
    await Promise.all(validations.map(validation => validation.run(req)));
    
    const errors = validationResult(req);
    if (errors.isEmpty()) {
      return next();
    }
    
    res.status(400).json({
      success: false,
      errors: errors.array()
    });
  };
};

// 验证规则示例
export const validateCreateMeeting = validate([
  body('title')
    .trim()
    .notEmpty().withMessage('会议标题不能为空')
    .isLength({ max: 100 }).withMessage('标题最多100个字符'),
  body('password')
    .optional()
    .isLength({ min: 4, max: 20 }).withMessage('密码长度4-20位'),
  body('settings.maxParticipants')
    .optional()
    .isInt({ min: 2, max: 200 }).withMessage('参与人数2-200')
]);
```

### 5.3 错误处理中间件

```typescript
// middlewares/error.middleware.ts
import { Request, Response, NextFunction } from 'express';

export class AppError extends Error {
  statusCode: number;
  isOperational: boolean;
  
  constructor(message: string, statusCode: number) {
    super(message);
    this.statusCode = statusCode;
    this.isOperational = true;
    Error.captureStackTrace(this, this.constructor);
  }
}

export const errorHandler = (
  err: Error | AppError,
  req: Request,
  res: Response,
  next: NextFunction
) => {
  const statusCode = 'statusCode' in err ? err.statusCode : 500;
  const message = err.message || '服务器内部错误';
  
  console.error('Error:', {
    message: err.message,
    stack: err.stack,
    url: req.url,
    method: req.method
  });
  
  res.status(statusCode).json({
    success: false,
    message,
    ...(process.env.NODE_ENV === 'development' && { stack: err.stack })
  });
};
```

## 6. 服务器启动

```typescript
// index.ts
import express from 'express';
import { createServer } from 'http';
import mongoose from 'mongoose';
import cors from 'cors';
import helmet from 'helmet';
import compression from 'compression';
import { createApiRouter } from './routes';
import { errorHandler } from './middlewares/error.middleware';

async function bootstrap() {
  const app = express();
  const httpServer = createServer(app);
  
  // 中间件
  app.use(helmet());
  app.use(cors({ origin: process.env.CLIENT_URL, credentials: true }));
  app.use(compression());
  app.use(express.json({ limit: '10mb' }));
  app.use(express.urlencoded({ extended: true }));
  
  // 路由
  app.use('/api', createApiRouter());
  
  // 健康检查
  app.get('/health', (req, res) => {
    res.json({ status: 'ok', timestamp: new Date().toISOString() });
  });
  
  // 错误处理
  app.use(errorHandler);
  
  // 数据库连接
  await mongoose.connect(process.env.MONGODB_URI!);
  console.log('MongoDB connected');
  
  // 启动服务器
  const PORT = process.env.PORT || 3001;
  httpServer.listen(PORT, () => {
    console.log(`API server running on port ${PORT}`);
  });
}

bootstrap();
```
