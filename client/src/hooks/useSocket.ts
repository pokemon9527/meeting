import { useCallback } from 'react';
import { io, Socket } from 'socket.io-client';
import { useUserStore } from '../stores/userStore';
import { useMeetingStore } from '../stores/meetingStore';

const SIGNALING_URL = import.meta.env.VITE_SIGNALING_URL || 'http://localhost:3002';

let socket: Socket | null = null;

export const getSocket = () => socket;

export const useSocket = () => {
  const { user } = useUserStore();
  const {
    addPeer,
    removePeer,
    updatePeer,
    addMessage,
  } = useMeetingStore();

  const connect = useCallback(
    (meetingIdParam: string, joinToken: string) => {
      if (socket?.connected) {
        socket.disconnect();
      }

      socket = io(SIGNALING_URL, {
        auth: {
          token: joinToken,
          meetingId: meetingIdParam,
          userId: user?.id,
          username: user?.username,
          avatar: user?.avatar,
        },
        transports: ['websocket', 'polling'],
      });

      socket.on('connect', () => {
        console.log('Socket connected');
      });

      socket.on('disconnect', () => {
        console.log('Socket disconnected');
      });

      socket.on('peer-joined', (data) => {
        console.log('Peer joined:', data);
        addPeer({
          id: data.peerId,
          userId: data.userId,
          username: data.username,
          avatar: data.avatar,
          role: data.role,
          audioEnabled: true,
          videoEnabled: true,
          isScreenSharing: false,
          isHandRaised: false,
        });
      });

      socket.on('peer-left', (data) => {
        console.log('Peer left:', data);
        removePeer(data.peerId);
      });

      socket.on('peer-updated', (data) => {
        updatePeer(data.peerId, data);
      });

      socket.on('hand-raised', (data) => {
        updatePeer(data.peerId, { isHandRaised: data.raised });
      });

      socket.on('new-message', (message) => {
        addMessage(message);
      });

      return socket;
    },
    [user, addPeer, removePeer, updatePeer, addMessage]
  );

  const disconnect = useCallback(() => {
    if (socket) {
      socket.disconnect();
      socket = null;
    }
  }, []);

  const emit = useCallback((event: string, data?: any, callback?: Function) => {
    if (socket?.connected) {
      if (callback) {
        socket.emit(event, data, callback);
      } else {
        socket.emit(event, data);
      }
    }
  }, []);

  const on = useCallback((event: string, handler: Function) => {
    socket?.on(event, handler as any);
  }, []);

  const off = useCallback((event: string, handler?: Function) => {
    if (handler) {
      socket?.off(event, handler as any);
    } else {
      socket?.off(event);
    }
  }, []);

  return { connect, disconnect, emit, on, off, socket };
};
