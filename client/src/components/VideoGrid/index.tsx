import React, { useRef, useEffect, useMemo } from 'react';
import { useMeetingStore } from '../../stores/meetingStore';

interface VideoTileProps {
  peerId?: string;
  username: string;
  stream?: MediaStream;
  isLocal?: boolean;
  audioEnabled: boolean;
  videoEnabled: boolean;
  isScreenShare?: boolean;
}

const VideoTile: React.FC<VideoTileProps> = ({
  username,
  stream,
  isLocal,
  audioEnabled,
  videoEnabled,
  isScreenShare,
}) => {
  const videoRef = useRef<HTMLVideoElement>(null);

  useEffect(() => {
    if (videoRef.current && stream) {
      videoRef.current.srcObject = stream;
    }
  }, [stream]);

  return (
    <div className="video-tile">
      <video
        ref={videoRef}
        autoPlay
        playsInline
        muted={isLocal}
        style={{ display: videoEnabled ? 'block' : 'none' }}
      />
      {!videoEnabled && !isScreenShare && (
        <div className="video-placeholder">
          <div className="avatar">{username.charAt(0).toUpperCase()}</div>
        </div>
      )}
      <div className="video-overlay">
        <span className="username">
          {username} {isLocal && '(我)'}
        </span>
        <div className="status-icons">
          {!audioEnabled && <span className="muted-icon">🔇</span>}
        </div>
      </div>
    </div>
  );
};

interface VideoGridProps {
  localStream?: MediaStream;
  screenStream?: MediaStream;
  remoteStreams: Map<string, { peerId: string; kind: string; stream: MediaStream }>;
}

export const VideoGrid: React.FC<VideoGridProps> = ({
  localStream,
  screenStream,
  remoteStreams,
}) => {
  const { peerId, peers, localAudioEnabled, localVideoEnabled } = useMeetingStore();

  const gridLayout = useMemo(() => {
    const count = peers.size;
    if (count <= 1) return { cols: 1, rows: 1 };
    if (count <= 2) return { cols: 2, rows: 1 };
    if (count <= 4) return { cols: 2, rows: 2 };
    if (count <= 6) return { cols: 3, rows: 2 };
    if (count <= 9) return { cols: 3, rows: 3 };
    if (count <= 12) return { cols: 4, rows: 3 };
    if (count <= 16) return { cols: 4, rows: 4 };
    return { cols: 5, rows: Math.ceil(count / 5) };
  }, [peers.size]);

  const getStreamForPeer = (pId: string, kind: 'audio' | 'video'): MediaStream | undefined => {
    if (pId === peerId) {
      return kind === 'video' ? localStream : localStream;
    }

    for (const [, remoteStream] of remoteStreams) {
      if (remoteStream.peerId === pId && remoteStream.kind === kind) {
        return remoteStream.stream;
      }
    }
    return undefined;
  };

  return (
    <div className="video-grid-container">
      {screenStream && (
        <div className="screen-share-container">
          <video
            autoPlay
            playsInline
            ref={(el) => {
              if (el) el.srcObject = screenStream;
            }}
          />
        </div>
      )}

      <div
        className="video-grid"
        style={{
          gridTemplateColumns: `repeat(${gridLayout.cols}, 1fr)`,
          gridTemplateRows: `repeat(${gridLayout.rows}, 1fr)`,
        }}
      >
        {Array.from(peers.values()).map((peer) => (
          <VideoTile
            key={peer.id}
            peerId={peer.id}
            username={peer.username}
            stream={getStreamForPeer(peer.id, 'video')}
            isLocal={peer.id === peerId}
            audioEnabled={peer.id === peerId ? localAudioEnabled : peer.audioEnabled}
            videoEnabled={peer.id === peerId ? localVideoEnabled : peer.videoEnabled}
          />
        ))}
      </div>
    </div>
  );
};
