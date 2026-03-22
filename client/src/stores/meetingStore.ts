import { create } from 'zustand';
import type { MeetingInfo, Peer, ChatMessage } from '../types';

interface RemoteStream {
  peerId: string;
  stream: MediaStream;
}

interface MeetingState {
  meetingId: string | null;
  meetingInfo: MeetingInfo | null;
  peerId: string | null;
  peers: Map<string, Peer>;
  remoteStreams: Map<string, RemoteStream>;
  localAudioEnabled: boolean;
  localVideoEnabled: boolean;
  isScreenSharing: boolean;
  isRecording: boolean;
  isHandRaised: boolean;
  activePanel: 'chat' | 'participants' | 'whiteboard' | null;
  messages: ChatMessage[];
  unreadMessages: number;

  setMeetingInfo: (info: MeetingInfo | null) => void;
  setPeerId: (peerId: string) => void;
  setPeers: (peers: Peer[]) => void;
  addPeer: (peer: Peer) => void;
  removePeer: (peerId: string) => void;
  updatePeer: (peerId: string, updates: Partial<Peer>) => void;
  addRemoteStream: (peerId: string, stream: MediaStream) => void;
  removeRemoteStream: (peerId: string) => void;
  toggleAudio: () => void;
  toggleVideo: () => void;
  toggleScreenShare: () => void;
  toggleRecording: () => void;
  toggleHandRaise: () => void;
  setActivePanel: (panel: 'chat' | 'participants' | 'whiteboard' | null) => void;
  addMessage: (message: ChatMessage) => void;
  setMessages: (messages: ChatMessage[]) => void;
  clearUnreadMessages: () => void;
  reset: () => void;
}

export const useMeetingStore = create<MeetingState>((set, get) => ({
  meetingId: null,
  meetingInfo: null,
  peerId: null,
  peers: new Map(),
  remoteStreams: new Map(),
  localAudioEnabled: true,
  localVideoEnabled: true,
  isScreenSharing: false,
  isRecording: false,
  isHandRaised: false,
  activePanel: null,
  messages: [],
  unreadMessages: 0,

  setMeetingInfo: (info) => set({ meetingInfo: info, meetingId: info?.meetingId || null }),

  setPeerId: (peerId) => set({ peerId }),

  setPeers: (peers) => {
    const peersMap = new Map<string, Peer>();
    peers.forEach((peer) => peersMap.set(peer.id, peer));
    set({ peers: peersMap });
  },

  addPeer: (peer) => {
    const peers = new Map(get().peers);
    peers.set(peer.id, peer);
    set({ peers });
  },

  removePeer: (peerId) => {
    const peers = new Map(get().peers);
    peers.delete(peerId);
    const remoteStreams = new Map(get().remoteStreams);
    remoteStreams.delete(peerId);
    set({ peers, remoteStreams });
  },

  updatePeer: (peerId, updates) => {
    const peers = new Map(get().peers);
    const peer = peers.get(peerId);
    if (peer) {
      peers.set(peerId, { ...peer, ...updates });
      set({ peers });
    }
  },

  addRemoteStream: (peerId, stream) => {
    const remoteStreams = new Map(get().remoteStreams);
    remoteStreams.set(peerId, { peerId, stream });
    set({ remoteStreams });
  },

  removeRemoteStream: (peerId) => {
    const remoteStreams = new Map(get().remoteStreams);
    remoteStreams.delete(peerId);
    set({ remoteStreams });
  },

  toggleAudio: () => set((state) => ({ localAudioEnabled: !state.localAudioEnabled })),

  toggleVideo: () => set((state) => ({ localVideoEnabled: !state.localVideoEnabled })),

  toggleScreenShare: () => set((state) => ({ isScreenSharing: !state.isScreenSharing })),

  toggleRecording: () => set((state) => ({ isRecording: !state.isRecording })),

  toggleHandRaise: () => set((state) => ({ isHandRaised: !state.isHandRaised })),

  setActivePanel: (panel) => set({ activePanel: panel }),

  addMessage: (message) => {
    const { activePanel } = get();
    set((state) => ({
      messages: [...state.messages, message],
      unreadMessages: activePanel === 'chat' ? state.unreadMessages : state.unreadMessages + 1,
    }));
  },

  setMessages: (messages) => set({ messages }),

  clearUnreadMessages: () => set({ unreadMessages: 0 }),

  reset: () =>
    set({
      meetingId: null,
      meetingInfo: null,
      peerId: null,
      peers: new Map(),
      remoteStreams: new Map(),
      localAudioEnabled: true,
      localVideoEnabled: true,
      isScreenSharing: false,
      isRecording: false,
      isHandRaised: false,
      activePanel: null,
      messages: [],
      unreadMessages: 0,
    }),
}));
