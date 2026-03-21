export interface User {
  id: string;
  username: string;
  email: string;
  avatar: string;
}

export interface MeetingInfo {
  meetingId: string;
  title: string;
  description?: string;
  hostId: string;
  password?: string;
  settings: MeetingSettings;
  status: 'scheduled' | 'waiting' | 'active' | 'ended';
  createdAt: string;
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

export interface Peer {
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

export interface ChatMessage {
  id: string;
  senderId: string;
  senderName: string;
  senderAvatar: string;
  content: string;
  type: 'text' | 'file' | 'image';
  timestamp: Date;
  isPrivate: boolean;
  receiverId?: string;
}

export interface TransportParams {
  id: string;
  iceParameters: any;
  iceCandidates: any[];
  dtlsParameters: any;
}

export interface ProducerInfo {
  id: string;
  peerId: string;
  kind: 'audio' | 'video';
  appData?: any;
}

export interface ConsumerInfo {
  id: string;
  producerId: string;
  peerId: string;
  kind: 'audio' | 'video';
  rtpParameters: any;
}
