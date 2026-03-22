import React, { useEffect, useState, useCallback, useRef } from 'react';
import { useParams, useNavigate } from 'react-router-dom';
import { message, Spin } from 'antd';
import { VideoGrid } from '../../components/VideoGrid';
import { Toolbar } from '../../components/Toolbar';
import { ChatPanel } from '../../components/ChatPanel';
import { ParticipantList } from '../../components/ParticipantList';
import { useSocket, setLocalStream, setScreenStream } from '../../hooks/useSocket';
import { useMeetingStore } from '../../stores/meetingStore';
import { useUserStore } from '../../stores/userStore';
import { meetingApi } from '../../services/api';
import './Meeting.css';

const Meeting: React.FC = () => {
  const { meetingId } = useParams<{ meetingId: string }>();
  const navigate = useNavigate();
  const { user } = useUserStore();
  const {
    setMeetingInfo,
    setPeerId,
    setPeers,
    activePanel,
    remoteStreams,
    isScreenSharing,
    setActivePanel,
    reset,
    toggleScreenShare,
  } = useMeetingStore();

  const [loading, setLoading] = useState(true);
  const [localStream, setLocalStreamState] = useState<MediaStream | null>(null);
  const [screenStream, setScreenStreamState] = useState<MediaStream | null>(null);

  const { connect, disconnect, emit, startScreenShare, stopScreenShare } = useSocket();
  const socketRef = useRef<any>(null);
  const initializedRef = useRef(false);

  useEffect(() => {
    if (!meetingId || initializedRef.current) return;

    const init = async () => {
      initializedRef.current = true;

      try {
        const meetingResponse = await meetingApi.getMeeting(meetingId);
        setMeetingInfo(meetingResponse.data.data.meeting);
      } catch {
        message.error('获取会议信息失败');
        navigate('/');
        return;
      }

      try {
        let stream = null;
        try {
          stream = await navigator.mediaDevices.getUserMedia({
            video: true,
            audio: true,
          });
          setLocalStream(stream);
          setLocalStreamState(stream);
        } catch (mediaError) {
          console.error('Failed to get media devices:', mediaError);
          message.warning('无法访问摄像头或麦克风');
        }

        const joinResponse = await meetingApi.joinMeeting({ meetingId });
        const { token } = joinResponse.data.data;

        const socket = connect(meetingId, token);
        socketRef.current = socket;

        socket.on('join-response', (data: any) => {
          if (!data.success) {
            message.error(data.error || '加入会议失败');
            navigate('/');
            return;
          }

          setPeerId(data.peer_id);
          
          const localPeer = {
            id: data.peer_id,
            userId: user?.id,
            username: user?.username || '我',
            avatar: user?.avatar || '',
            role: 'host',
            audioEnabled: true,
            videoEnabled: true,
            isScreenSharing: false,
            isHandRaised: false,
          };
          
          const peersData = [
            localPeer,
            ...(data.producers || []).map((p: any) => ({
              id: p.peer_id,
              userId: '',
              username: `User ${p.peer_id.slice(-4)}`,
              avatar: '',
              role: 'participant',
              audioEnabled: true,
              videoEnabled: true,
              isScreenSharing: false,
              isHandRaised: false,
            })),
          ];
          setPeers(peersData);

          setLoading(false);
        });
      } catch {
        message.error('加入会议失败');
        navigate('/');
      }
    };

    init();

    return () => {
      if (socketRef.current) {
        emit('leave-room');
        disconnect();
        if (localStream) {
          localStream.getTracks().forEach(track => track.stop());
        }
        if (screenStream) {
          screenStream.getTracks().forEach(track => track.stop());
        }
        reset();
      }
    };
  }, [meetingId]);

  const handleToggleAudio = useCallback(() => {
    if (localStream) {
      const audioTrack = localStream.getAudioTracks()[0];
      if (audioTrack) {
        audioTrack.enabled = !audioTrack.enabled;
      }
    }
  }, [localStream]);

  const handleToggleVideo = useCallback(() => {
    if (localStream) {
      const videoTrack = localStream.getVideoTracks()[0];
      if (videoTrack) {
        videoTrack.enabled = !videoTrack.enabled;
      }
    }
  }, [localStream]);

  const handleToggleScreenShare = useCallback(async () => {
    if (isScreenSharing) {
      stopScreenShare();
      setScreenStreamState(null);
      setScreenStream(null);
      toggleScreenShare();
      message.info('已停止屏幕共享');
    } else {
      try {
        const stream = await navigator.mediaDevices.getDisplayMedia({
          video: {
            width: { ideal: 1920 },
            height: { ideal: 1080 },
          },
          audio: false,
        } as MediaStreamConstraints);

        const started = await startScreenShare(stream);
        if (started) {
          setScreenStreamState(stream);
          setScreenStream(stream);
          toggleScreenShare();
          message.success('屏幕共享已开启');

          stream.getVideoTracks()[0].onended = () => {
            stopScreenShare();
            setScreenStreamState(null);
            setScreenStream(null);
            toggleScreenShare();
            message.info('屏幕共享已结束');
          };
        }
      } catch (error) {
        console.error('Screen share error:', error);
        if ((error as Error).name !== 'NotAllowedError') {
          message.error('无法启动屏幕共享');
        }
      }
    }
  }, [isScreenSharing, startScreenShare, stopScreenShare, toggleScreenShare]);

  const handleLeaveMeeting = useCallback(() => {
    emit('leave-room');
    disconnect();
    if (localStream) {
      localStream.getTracks().forEach(track => track.stop());
    }
    if (screenStream) {
      screenStream.getTracks().forEach(track => track.stop());
    }
    reset();
    navigate('/');
  }, [emit, disconnect, localStream, screenStream, reset, navigate]);

  const handleToggleChat = useCallback(() => {
    setActivePanel(activePanel === 'chat' ? null : 'chat');
  }, [activePanel, setActivePanel]);

  const handleToggleParticipants = useCallback(() => {
    setActivePanel(activePanel === 'participants' ? null : 'participants');
  }, [activePanel, setActivePanel]);

  if (loading) {
    return (
      <div className="meeting-loading">
        <Spin size="large" />
        <p>正在加入会议...</p>
      </div>
    );
  }

  return (
    <div className="meeting-container">
      <div className="meeting-main">
        <VideoGrid
          localStream={localStream || undefined}
          screenStream={screenStream || undefined}
          remoteStreams={remoteStreams}
        />

        <Toolbar
          onToggleAudio={handleToggleAudio}
          onToggleVideo={handleToggleVideo}
          onToggleScreenShare={handleToggleScreenShare}
          onLeaveMeeting={handleLeaveMeeting}
          onToggleChat={handleToggleChat}
          onToggleParticipants={handleToggleParticipants}
          isScreenSharing={isScreenSharing}
        />
      </div>

      {activePanel && (
        <div className="meeting-sidebar">
          {activePanel === 'chat' && <ChatPanel />}
          {activePanel === 'participants' && <ParticipantList />}
        </div>
      )}
    </div>
  );
};

export default Meeting;
