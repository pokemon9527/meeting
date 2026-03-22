import { useCallback } from 'react';
import { useUserStore } from '../stores/userStore';
import { useMeetingStore } from '../stores/meetingStore';

const SFU_URL = import.meta.env.VITE_SFU_URL || '';

export type SFUSocket = {
  connected: boolean;
  on: (event: string, handler: (data: any) => void) => void;
  off: (event: string, handler?: (data: any) => void) => void;
  emit: (event: string, data?: any) => void;
  disconnect: () => void;
};

let socketRef: WebSocket | null = null;
const eventHandlers: Map<string, Set<(data: any) => void>> = new Map();
let localStreamRef: MediaStream | null = null;
let pcRef: RTCPeerConnection | null = null;
let lastOfferSenderPeerId: string | null = null;
let pendingIceCandidates: any[] = [];

// Screen sharing state
let screenStreamRef: MediaStream | null = null;
let screenTrackSender: RTCRtpSender | null = null;
let screenTrackPeerIdMap: Map<string, MediaStream> = new Map();

export const setLocalStream = (stream: MediaStream) => {
  localStreamRef = stream;
};

export const setScreenStream = (stream: MediaStream | null) => {
  screenStreamRef = stream;
};

export const getSocket = () => socketRef;

function flushPendingIceCandidates() {
  if (!pcRef) return;
  while (pendingIceCandidates.length > 0) {
    const candidate = pendingIceCandidates.shift();
    pcRef.addIceCandidate(new RTCIceCandidate(candidate))
      .then(() => console.log('Flushed pending ICE candidate'))
      .catch(e => console.error('Failed to add pending ICE candidate:', e));
  }
}

async function renegotiate() {
  if (!pcRef || !socketRef || socketRef.readyState !== WebSocket.OPEN) return;
  
  if (pcRef.signalingState !== 'stable') {
    console.log('Skipping renegotiation, signaling state is:', pcRef.signalingState);
    return;
  }

  try {
    const offer = await pcRef.createOffer();
    await pcRef.setLocalDescription(offer);
    
    await new Promise<void>((resolve) => {
      const checkGathering = () => {
        if (pcRef!.iceGatheringState === 'complete') {
          resolve();
        } else if (pcRef!.localDescription?.sdp && pcRef!.localDescription.sdp.length > 500) {
          resolve();
        }
      };
      checkGathering();
      const interval = setInterval(checkGathering, 100);
      setTimeout(() => {
        clearInterval(interval);
        resolve();
      }, 3000);
    });

    socketRef.send(JSON.stringify({
      event: 'offer',
      data: {
        sdp: pcRef.localDescription!.sdp,
        type: pcRef.localDescription!.type,
      },
    }));
    console.log('Sent renegotiation offer, SDP length:', pcRef.localDescription!.sdp.length);
  } catch (e) {
    console.error('Failed to renegotiate:', e);
  }
}

export const useSocket = () => {
  const { user } = useUserStore();
  const {
    addPeer,
    removePeer,
    addRemoteStream,
    setPeerId,
    addMessage,
  } = useMeetingStore();

  const handleOffer = async (data: any) => {
    if (!pcRef || !socketRef) return;
    
    try {
      await pcRef.setRemoteDescription(new RTCSessionDescription({
        type: data.type,
        sdp: data.sdp,
      }));
      flushPendingIceCandidates();
      
      const answer = await pcRef.createAnswer();
      await pcRef.setLocalDescription(answer);
      
      socketRef.send(JSON.stringify({
        event: 'answer',
        data: {
          sdp: pcRef.localDescription!.sdp,
          type: pcRef.localDescription!.type,
        },
      }));
      console.log('Sent answer to SFU');
    } catch (e) {
      console.error('Failed to handle offer:', e);
    }
  };

  const connect = useCallback(
    (meetingIdParam: string, _joinToken: string): SFUSocket => {
      if (socketRef?.readyState === WebSocket.OPEN) {
        socketRef.close();
      }

      const wsProtocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
      const wsUrl = SFU_URL
        ? `${SFU_URL}/sfu/ws`
        : `${wsProtocol}//${window.location.host}/sfu/ws`;

      socketRef = new WebSocket(wsUrl);
      eventHandlers.clear();
      screenTrackPeerIdMap.clear();

      const serverHost = window.location.hostname;
      const config: RTCConfiguration = {
        iceServers: [
          { urls: 'stun:stun.l.google.com:19302' },
          { urls: `stun:${serverHost}:3478` },
          {
            urls: `turn:${serverHost}:3478`,
            username: 'meeting',
            credential: 'meeting123',
          },
        ],
      };

      pcRef = new RTCPeerConnection(config);

      pcRef.onicecandidate = (event) => {
        if (event.candidate && socketRef?.readyState === WebSocket.OPEN) {
          socketRef.send(JSON.stringify({
            event: 'ice-candidate',
            data: {
              candidate: event.candidate.toJSON(),
            },
          }));
        }
      };

      pcRef.onconnectionstatechange = () => {
        console.log('🔵 SFU connection state:', pcRef?.connectionState);
      };

      pcRef.oniceconnectionstatechange = () => {
        console.log('🟢 SFU ICE state:', pcRef?.iceConnectionState);
      };

      pcRef.onicegatheringstatechange = () => {
        console.log('📡 ICE gathering state:', pcRef?.iceGatheringState);
      };

      socketRef.onopen = () => {
        console.log('SFU WebSocket connected');

        if (localStreamRef) {
          localStreamRef.getTracks().forEach(track => {
            console.log('Adding local track:', track.kind);
            pcRef!.addTrack(track, localStreamRef!);
          });
        } else {
          pcRef!.addTransceiver('video', { direction: 'sendrecv' });
          pcRef!.addTransceiver('audio', { direction: 'sendrecv' });
          console.log('Added sendrecv transceivers (no local stream)');
        }

        socketRef?.send(JSON.stringify({
          event: 'join',
          data: {
            meeting_id: meetingIdParam,
            peer_id: `${user?.id}_${Date.now()}`,
            username: user?.username || 'User',
            avatar: user?.avatar || '',
          },
        }));
      };

      socketRef.onmessage = async (event) => {
        try {
          const message = JSON.parse(event.data);
          const { event: eventName, data } = message;
          console.log('SFU Received:', eventName);

          const handlers = eventHandlers.get(eventName);
          if (handlers) {
            handlers.forEach(handler => handler(data));
          }

          switch (eventName) {
            case 'join-response':
              console.log('🎬 SFU Join response:', data);
              if (data.success) {
                setPeerId(data.peer_id);

                addPeer({
                  id: data.peer_id,
                  userId: user?.id || '',
                  username: user?.username || '我',
                  avatar: user?.avatar || '',
                  role: 'host',
                  audioEnabled: true,
                  videoEnabled: true,
                  isScreenSharing: false,
                  isHandRaised: false,
                });

                if (pcRef && pcRef.signalingState === 'stable') {
                  const offer = await pcRef.createOffer();
                  await pcRef.setLocalDescription(offer);
                  
                  await new Promise<void>((resolve) => {
                    const checkGathering = () => {
                      if (pcRef!.iceGatheringState === 'complete') {
                        resolve();
                      } else if (pcRef!.localDescription?.sdp && pcRef!.localDescription.sdp.length > 500) {
                        resolve();
                      }
                    };
                    checkGathering();
                    const interval = setInterval(checkGathering, 100);
                    setTimeout(() => {
                      clearInterval(interval);
                      resolve();
                    }, 3000);
                  });
                  
                  socketRef?.send(JSON.stringify({
                    event: 'offer',
                    data: {
                      sdp: pcRef!.localDescription!.sdp,
                      type: pcRef!.localDescription!.type,
                    },
                  }));
                }

                if (data.producers) {
                  data.producers.forEach((p: any) => {
                    addPeer({
                      id: p.peer_id,
                      userId: '',
                      username: `User ${p.peer_id.slice(-4)}`,
                      avatar: '',
                      role: 'participant',
                      audioEnabled: true,
                      videoEnabled: true,
                      isScreenSharing: false,
                      isHandRaised: false,
                    });
                  });
                }
              }
              break;

            case 'answer':
              console.log('📞 SFU Answer received');
              if (pcRef) {
                try {
                  await pcRef.setRemoteDescription(new RTCSessionDescription({
                    type: data.type,
                    sdp: data.sdp,
                  }));
                  flushPendingIceCandidates();
                } catch (e) {
                  console.error('Failed to set remote description:', e);
                }
              }
              break;

            case 'offer':
              console.log('📞 SFU Offer received');
              if (data.peer_id) {
                lastOfferSenderPeerId = data.peer_id;
              }
              await handleOffer(data);
              break;

            case 'new-producer':
              console.log('📡 New producer:', data);
              const isScreenShare = data.kind === 'screen';
              addPeer({
                id: data.peer_id,
                userId: '',
                username: `User ${data.peer_id.slice(-4)}`,
                avatar: '',
                role: 'participant',
                audioEnabled: true,
                videoEnabled: !isScreenShare,
                isScreenSharing: isScreenShare,
                isHandRaised: false,
              });
              break;

            case 'peer-joined':
              console.log('👋 Peer joined:', data);
              addPeer({
                id: data.peer_id,
                userId: data.user_id || '',
                username: data.username || `User ${data.peer_id.slice(-4)}`,
                avatar: data.avatar || '',
                role: data.role || 'participant',
                audioEnabled: true,
                videoEnabled: true,
                isScreenSharing: false,
                isHandRaised: false,
              });
              break;

            case 'peer-left':
              console.log('👋 Peer left:', data);
              removePeer(data.peer_id);
              screenTrackPeerIdMap.delete(data.peer_id);
              break;

            case 'track-removed':
              console.log('❌ Track removed:', data);
              screenTrackPeerIdMap.delete(data.peer_id);
              break;

            case 'ice-candidate':
              if (data.candidate && pcRef) {
                if (pcRef.remoteDescription) {
                  pcRef.addIceCandidate(new RTCIceCandidate(data.candidate))
                    .catch(e => console.error('Failed to add ICE candidate:', e));
                } else {
                  pendingIceCandidates.push(data.candidate);
                }
              }
              break;

            case 'new-message':
              console.log('💬 New message received:', data);
              addMessage({
                id: data.id || `msg_${Date.now()}`,
                senderId: data.peer_id || '',
                senderName: data.username || 'Unknown',
                senderAvatar: data.avatar || '',
                content: data.content,
                timestamp: data.timestamp ? new Date(data.timestamp) : new Date(),
                type: data.type || 'text',
                isPrivate: false,
              });
              break;

            case 'peer-screen-share':
              console.log('🖥️ Peer screen share:', data);
              if (data.peer_id && data.screen_stream_id) {
                screenTrackPeerIdMap.set(data.peer_id, { id: data.screen_stream_id } as MediaStream);
              }
              break;
          }
        } catch (e) {
          console.error('Failed to parse SFU message:', e);
        }
      };

      pcRef.ontrack = (event) => {
        console.log('🔴 SFU received track:', event.track.kind, 'trackId:', event.track.id, 'streams:', event.streams.length);
        
        const peerId = lastOfferSenderPeerId || event.streams[0]?.id;
        
        event.streams.forEach(stream => {
          console.log('Adding remote stream:', stream.id, 'for peer:', peerId);
          const { addRemoteStream } = useMeetingStore.getState();
          addRemoteStream(peerId || stream.id, stream);
        });
      };

      pcRef.onnegotiationneeded = () => {
        console.log('Negotiation needed');
        renegotiate();
      };

      socketRef.onerror = (error) => {
        console.error('SFU WebSocket error:', error);
      };

      socketRef.onclose = () => {
        console.log('SFU WebSocket disconnected');
        pcRef?.close();
        pcRef = null;
        screenTrackSender = null;
        screenTrackPeerIdMap.clear();
      };

      const socket: SFUSocket = {
        connected: true,
        on: (event: string, handler: (data: any) => void) => {
          if (!eventHandlers.has(event)) {
            eventHandlers.set(event, new Set());
          }
          eventHandlers.get(event)!.add(handler);
        },
        off: (event: string, handler?: (data: any) => void) => {
          if (handler) {
            eventHandlers.get(event)?.delete(handler);
          } else {
            eventHandlers.delete(event);
          }
        },
        emit: (event: string, data?: any) => {
          if (socketRef?.readyState === WebSocket.OPEN) {
            socketRef.send(JSON.stringify({ event, data: data || {} }));
          }
        },
        disconnect: () => {
          socketRef?.send(JSON.stringify({ event: 'leave', data: {} }));
          pcRef?.close();
          pcRef = null;
          socketRef?.close();
          socketRef = null;
          eventHandlers.clear();
          lastOfferSenderPeerId = null;
          pendingIceCandidates = [];
          screenTrackSender = null;
          screenTrackPeerIdMap.clear();
        },
      };

      return socket;
    },
    [user, addPeer, removePeer, addRemoteStream, setPeerId, addMessage]
  );

  const disconnect = useCallback(() => {
    socketRef?.send(JSON.stringify({ event: 'leave', data: {} }));
    pcRef?.close();
    pcRef = null;
    socketRef?.close();
    socketRef = null;
    eventHandlers.clear();
    lastOfferSenderPeerId = null;
    pendingIceCandidates = [];
    screenTrackSender = null;
    screenTrackPeerIdMap.clear();
  }, []);

  const emit = useCallback((event: string, data?: any) => {
    if (socketRef?.readyState === WebSocket.OPEN) {
      socketRef.send(JSON.stringify({ event, data: data || {} }));
    }
  }, []);

  const startScreenShare = useCallback(async (screenStream: MediaStream) => {
    if (!pcRef || !socketRef) {
      console.error('No peer connection available');
      return false;
    }

    try {
      screenStreamRef = screenStream;
      const screenTrack = screenStream.getVideoTracks()[0];
      
      if (!screenTrack) {
        console.error('No video track in screen stream');
        return false;
      }

      screenTrack.onended = () => {
        console.log('Screen share track ended');
        stopScreenShare();
      };

      screenTrackSender = pcRef.addTrack(screenTrack, screenStream);
      console.log('Added screen share track');

      socketRef.send(JSON.stringify({
        event: 'toggle-screen-share',
        data: {
          enabled: true,
          screen_stream_id: screenStream.id,
        },
      }));

      await renegotiate();
      console.log('Screen share started');
      return true;
    } catch (e) {
      console.error('Failed to start screen share:', e);
      return false;
    }
  }, []);

  const stopScreenShare = useCallback(async () => {
    if (!pcRef || !socketRef) {
      return;
    }

    try {
      if (screenTrackSender) {
        pcRef.removeTrack(screenTrackSender);
        screenTrackSender = null;
      }

      if (screenStreamRef) {
        screenStreamRef.getTracks().forEach(track => track.stop());
        screenStreamRef = null;
      }

      socketRef.send(JSON.stringify({
        event: 'toggle-screen-share',
        data: {
          enabled: false,
        },
      }));

      await renegotiate();
      console.log('Screen share stopped');
    } catch (e) {
      console.error('Failed to stop screen share:', e);
    }
  }, []);

  return { 
    connect, 
    disconnect, 
    emit, 
    socket: socketRef,
    startScreenShare,
    stopScreenShare,
  };
};
