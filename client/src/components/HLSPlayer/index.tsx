import React, { useEffect, useRef, useState } from 'react';
import { Player, ControlBar, PlaybackRateMenuButton, VolumeMenuButton } from 'video.js';
import 'video.js/dist/video-js.css';

interface HLSPlayerProps {
  src: string;
  poster?: string;
  autoplay?: boolean;
  onError?: (error: Error) => void;
  onLoadedMetadata?: () => void;
}

const HLSPlayer: React.FC<HLSPlayerProps> = ({
  src,
  poster,
  autoplay = false,
  onError,
  onLoadedMetadata,
}) => {
  const videoRef = useRef<HTMLVideoElement>(null);
  const playerRef = useRef<Player | null>(null);
  const [isReady, setIsReady] = useState(false);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    if (!videoRef.current) return;

    const video = videoRef.current;

    if (video.canPlayType('application/vnd.apple.mpegurl')) {
      video.src = src;
      video.addEventListener('loadedmetadata', () => {
        setIsReady(true);
        onLoadedMetadata?.();
      });
      video.addEventListener('error', () => {
        const err = new Error('Failed to load video');
        setError(err.message);
        onError?.(err);
      });
      return;
    }

    const player = Player.getPlayer(video);
    if (player) {
      playerRef.current = player;
    } else {
      playerRef.current = Player(video, {
        controls: true,
        autoplay: autoplay,
        responsive: true,
        fluid: true,
        poster: poster,
        playbackRates: [0.5, 1, 1.5, 2],
        sources: [{ src, type: 'application/x-mpegURL' }],
      });
    }

    const playerInstance = playerRef.current;

    playerInstance.ready(() => {
      setIsReady(true);
      onLoadedMetadata?.();
    });

    playerInstance.on('error', () => {
      const err = new Error(playerInstance.error()?.message || 'Video playback error');
      setError(err.message);
      onError?.(err);
    });

    return () => {
      if (playerRef.current && !playerRef.current.isDisposed()) {
        playerRef.current.dispose();
        playerRef.current = null;
      }
    };
  }, [src, poster, autoplay, onError, onLoadedMetadata]);

  return (
    <div
      className="hls-player-wrapper"
      style={{
        position: 'relative',
        width: '100%',
        backgroundColor: '#000',
        borderRadius: '8px',
        overflow: 'hidden',
      }}
    >
      <video
        ref={videoRef}
        className="video-js vjs-big-play-centered"
        playsInline
        style={{
          width: '100%',
          height: 'auto',
        }}
      />
      {error && (
        <div
          style={{
            position: 'absolute',
            top: '50%',
            left: '50%',
            transform: 'translate(-50%, -50%)',
            color: '#fff',
            textAlign: 'center',
          }}
        >
          <p>{error}</p>
        </div>
      )}
      {!isReady && !error && (
        <div
          style={{
            position: 'absolute',
            top: '50%',
            left: '50%',
            transform: 'translate(-50%, -50%)',
            color: '#fff',
          }}
        >
          Loading...
        </div>
      )}
    </div>
  );
};

export default HLSPlayer;
