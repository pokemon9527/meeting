import { useRef, useCallback, useState } from 'react';

export const useMedia = () => {
  const localStreamRef = useRef<MediaStream | null>(null);
  const screenStreamRef = useRef<MediaStream | null>(null);
  const [audioEnabled, setAudioEnabled] = useState(true);
  const [videoEnabled, setVideoEnabled] = useState(true);

  const getLocalStream = useCallback(
    async (constraints?: MediaStreamConstraints): Promise<MediaStream> => {
      if (localStreamRef.current) {
        return localStreamRef.current;
      }

      const stream = await navigator.mediaDevices.getUserMedia(
        constraints || {
          audio: {
            echoCancellation: true,
            noiseSuppression: true,
            autoGainControl: true,
          },
          video: {
            width: { ideal: 1280 },
            height: { ideal: 720 },
            frameRate: { ideal: 30 },
          },
        }
      );

      localStreamRef.current = stream;
      return stream;
    },
    []
  );

  const getScreenStream = useCallback(async (): Promise<MediaStream> => {
    const stream = await navigator.mediaDevices.getDisplayMedia({
      video: {
        cursor: 'always',
      } as any,
      audio: false,
    });

    screenStreamRef.current = stream;

    stream.getVideoTracks()[0].addEventListener('ended', () => {
      screenStreamRef.current = null;
    });

    return stream;
  }, []);

  const toggleAudio = useCallback(() => {
    if (localStreamRef.current) {
      const audioTrack = localStreamRef.current.getAudioTracks()[0];
      if (audioTrack) {
        audioTrack.enabled = !audioTrack.enabled;
        setAudioEnabled(audioTrack.enabled);
      }
    }
  }, []);

  const toggleVideo = useCallback(() => {
    if (localStreamRef.current) {
      const videoTrack = localStreamRef.current.getVideoTracks()[0];
      if (videoTrack) {
        videoTrack.enabled = !videoTrack.enabled;
        setVideoEnabled(videoTrack.enabled);
      }
    }
  }, []);

  const stopLocalStream = useCallback(() => {
    if (localStreamRef.current) {
      localStreamRef.current.getTracks().forEach((track) => track.stop());
      localStreamRef.current = null;
    }
  }, []);

  const stopScreenStream = useCallback(() => {
    if (screenStreamRef.current) {
      screenStreamRef.current.getTracks().forEach((track) => track.stop());
      screenStreamRef.current = null;
    }
  }, []);

  return {
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
  };
};
