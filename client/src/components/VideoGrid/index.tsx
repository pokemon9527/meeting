import React, { useRef, useEffect, useMemo } from 'react';
import { useMeetingStore } from '../../stores/meetingStore';

interface VideoTileProps {
  peerId?: string;
  username: string;
  stream?: MediaStream;
  isLocal?: boolean;
  audioEnabled: boolean;
  videoEnabled?: boolean;
  isScreenShare?: boolean;
}

const VideoTile: React.FC<VideoTileProps> = ({
  username,
  stream,
  isLocal,
  audioEnabled,
  videoEnabled = true,
  isScreenShare,
}) => {
  const videoRef = useRef<HTMLVideoElement>(null);

  useEffect(() => {
    if (videoRef.current && stream) {
      videoRef.current.srcObject = stream;
      videoRef.current.play().catch(() => {
        console.log('Autoplay prevented, waiting for user interaction');
      });
    }
  }, [stream]);

  return (
    <div className={`video-tile ${isScreenShare ? 'video-tile-screen-share' : ''}`}>
      {stream && videoEnabled ? (
        <video
          ref={videoRef}
          autoPlay
          playsInline
          muted={isLocal}
          style={{ width: '100%', height: '100%', objectFit: isScreenShare ? 'contain' : 'cover' }}
        />
      ) : (
        <div className="video-placeholder">
          <div className="avatar">{(username || '?').charAt(0).toUpperCase()}</div>
        </div>
      )}
      <div className="video-overlay">
        <span className="username">
          {username} {isLocal && '(我)'}
          {isScreenShare && ' - 屏幕共享'}
        </span>
        <div className="status-icons">
          {!audioEnabled && <span className="muted-icon">🔇</span>}
          {isScreenShare && <span className="screen-icon">🖥️</span>}
        </div>
      </div>
    </div>
  );
};

interface VideoGridProps {
  localStream?: MediaStream;
  screenStream?: MediaStream;
  remoteStreams: Map<string, { peerId: string; stream: MediaStream }>;
}

export const VideoGrid: React.FC<VideoGridProps> = ({
  localStream,
  screenStream,
  remoteStreams,
}) => {
  const { peerId, peers, localAudioEnabled, localVideoEnabled, isScreenSharing } = useMeetingStore();

  const screenSharePeers = useMemo(() => {
    const result: Array<{ peerId: string; stream: MediaStream }> = [];
    remoteStreams.forEach((data, id) => {
      const peer = peers.get(id);
      if (peer?.isScreenSharing) {
        result.push(data);
      }
    });
    return result;
  }, [remoteStreams, peers]);

  const normalPeers = useMemo(() => {
    return Array.from(peers.values()).filter(
      (peer) => !peer.isScreenSharing && peer.id !== peerId
    );
  }, [peers, peerId]);

  const gridLayout = useMemo(() => {
    const count = normalPeers.length + 1;
    if (count <= 1) return { cols: 1, rows: 1 };
    if (count <= 2) return { cols: 2, rows: 1 };
    if (count <= 4) return { cols: 2, rows: 2 };
    if (count <= 6) return { cols: 3, rows: 2 };
    if (count <= 9) return { cols: 3, rows: 3 };
    return { cols: 4, rows: Math.ceil(count / 4) };
  }, [normalPeers.length]);

  if (isScreenSharing || screenStream || screenSharePeers.length > 0) {
    const mainScreenStream = screenStream || screenSharePeers[0]?.stream;
    const mainScreenPeerId = screenSharePeers[0]?.peerId;
    const mainScreenPeer = mainScreenPeerId ? peers.get(mainScreenPeerId) : null;

    return (
      <div className="video-grid-container video-grid-screen-share-mode">
        <div className="main-content">
          {mainScreenStream && (
            <div className="screen-share-main">
              <VideoTile
                peerId={mainScreenPeerId}
                username={mainScreenPeer?.username || '屏幕共享'}
                stream={mainScreenStream}
                isScreenShare={true}
                audioEnabled={false}
                videoEnabled={true}
              />
            </div>
          )}
          
          {localStream && (
            <div className="local-pip">
              <VideoTile
                peerId={peerId || undefined}
                username="我"
                stream={localStream}
                isLocal={true}
                audioEnabled={localAudioEnabled}
                videoEnabled={localVideoEnabled}
              />
            </div>
          )}
        </div>

        {normalPeers.length > 0 && (
          <div className="participants-bar">
            {normalPeers.map((peer) => {
              const streamData = remoteStreams.get(peer.id);
              return (
                <div key={peer.id} className="participant-tile">
                  <VideoTile
                    peerId={peer.id}
                    username={peer.username}
                    stream={streamData?.stream}
                    audioEnabled={peer.audioEnabled}
                    videoEnabled={peer.videoEnabled}
                  />
                </div>
              );
            })}
          </div>
        )}
      </div>
    );
  }

  return (
    <div className="video-grid-container">
      <div
        className="video-grid"
        style={{
          gridTemplateColumns: `repeat(${gridLayout.cols}, 1fr)`,
          gridTemplateRows: `repeat(${gridLayout.rows}, 1fr)`,
        }}
      >
        {localStream && (
          <VideoTile
            peerId={peerId || undefined}
            username="我"
            stream={localStream}
            isLocal={true}
            audioEnabled={localAudioEnabled}
            videoEnabled={localVideoEnabled}
          />
        )}
        
        {Array.from(remoteStreams.values()).map((remoteStreamData) => {
          const peer = peers.get(remoteStreamData.peerId);
          return (
            <VideoTile
              key={remoteStreamData.peerId}
              peerId={remoteStreamData.peerId}
              username={peer?.username || `User ${remoteStreamData.peerId.slice(-4)}`}
              stream={remoteStreamData.stream}
              audioEnabled={peer?.audioEnabled ?? true}
              videoEnabled={peer?.videoEnabled ?? true}
            />
          );
        })}
      </div>
    </div>
  );
};
