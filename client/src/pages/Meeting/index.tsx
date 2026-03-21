import React, { useEffect, useState, useCallback } from 'react';
import { useParams, useNavigate } from 'react-router-dom';
import { message, Spin } from 'antd';
import { VideoGrid } from '../../components/VideoGrid';
import { Toolbar } from '../../components/Toolbar';
import { ChatPanel } from '../../components/ChatPanel';
import { ParticipantList } from '../../components/ParticipantList';
import { useSocket } from '../../hooks/useSocket';
import { useWebRTC } from '../../hooks/useWebRTC';
import { useMedia } from '../../hooks/useMedia';
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
    reset,
  } = useMeetingStore();

  const [loading, setLoading] = useState(true);

  const { connect, disconnect, emit } = useSocket();
  const {
    localStreamRef,
    screenStreamRef,
    audioEnabled,
    videoEnabled,
    getLocalStream,
    getScreenStream,
    toggleAudio,
    toggleVideo,
    stopLocalStream,
    stopScreenStream,
  } = useMedia();
  const {
    remoteStreams,
    loadDevice,
    createSendTransport,
    createRecvTransport,
    produce,
    consume,
    closeAll,
  } = useWebRTC();

  const localStream = localStreamRef.current;
  const screenStream = screenStreamRef.current;

  useEffect(() => {
    if (!meetingId) {
      navigate('/');
      return;
    }

    const joinMeeting = async () => {
      try {
        const response = await meetingApi.getMeeting(meetingId);
        setMeetingInfo(response.data.data.meeting);
      } catch {
        message.error('获取会议信息失败');
        navigate('/');
      }
    };

    joinMeeting();

    return () => {
      handleLeaveMeeting();
    };
  }, [meetingId]);

  useEffect(() => {
    if (!meetingId || !user) return;

    const initializeMeeting = async () => {
      try {
        const response = await meetingApi.joinMeeting({ meetingId });
        const { token } = response.data.data;

        const socket = connect(meetingId, token);

        socket.emit('join-room', {}, async (roomResponse: any) => {
          if (!roomResponse.success) {
            message.error(roomResponse.error || '加入会议失败');
            navigate('/');
            return;
          }

          setPeerId(roomResponse.peerId);
          setPeers(roomResponse.peers);

          try {
            await loadDevice(roomResponse.rtpCapabilities);
            await createSendTransport();
            await createRecvTransport();

            const stream = await getLocalStream();
            const audioTrack = stream.getAudioTracks()[0];
            const videoTrack = stream.getVideoTracks()[0];

            if (audioTrack) {
              await produce(audioTrack, { share: false });
            }

            if (videoTrack) {
              await produce(videoTrack, { share: false });
            }

            setLoading(false);
          } catch (error) {
            console.error('Failed to initialize WebRTC:', error);
            message.error('初始化媒体失败');
          }
        });

        socket.on('new-producer', async (data: any) => {
          try {
            await consume(data.producerId, data.peerId, data.kind);
          } catch (error) {
            console.error('Failed to consume:', error);
          }
        });
      } catch {
        message.error('加入会议失败');
        navigate('/');
      }
    };

    initializeMeeting();
  }, [meetingId, user]);

  const handleToggleAudio = useCallback(() => {
    toggleAudio();
    emit('toggle-audio', { enabled: !audioEnabled });
  }, [audioEnabled, toggleAudio, emit]);

  const handleToggleVideo = useCallback(() => {
    toggleVideo();
    emit('toggle-video', { enabled: !videoEnabled });
  }, [videoEnabled, toggleVideo, emit]);

  const handleToggleScreenShare = useCallback(async () => {
    try {
      if (screenStream) {
        stopScreenStream();
        emit('toggle-screenshare', { enabled: false });
      } else {
        const stream = await getScreenStream();
        const videoTrack = stream.getVideoTracks()[0];
        if (videoTrack) {
          await produce(videoTrack, { share: true });
          emit('toggle-screenshare', { enabled: true });
        }
      }
    } catch (error) {
      console.error('Screen share error:', error);
      message.error('屏幕共享失败');
    }
  }, [screenStream, getScreenStream, stopScreenStream, produce, emit]);

  const handleLeaveMeeting = useCallback(() => {
    emit('leave-room');
    disconnect();
    closeAll();
    stopLocalStream();
    stopScreenStream();
    reset();
    navigate('/');
  }, [emit, disconnect, closeAll, stopLocalStream, stopScreenStream, reset, navigate]);

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
