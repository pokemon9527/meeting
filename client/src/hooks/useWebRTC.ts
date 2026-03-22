import { useCallback, useState } from 'react';

interface RemoteStream {
  peerId: string;
  producerId: string;
  kind: 'audio' | 'video';
  stream: MediaStream;
}

export const useWebRTC = () => {
  const [remoteStreams, setRemoteStreams] = useState<Map<string, RemoteStream>>(new Map());

  const loadDevice = useCallback(async () => {
    console.log('WebRTC: loadDevice skipped (no mediasoup server)');
  }, []);

  const createSendTransport = useCallback(async () => {
    console.log('WebRTC: createSendTransport skipped');
  }, []);

  const createRecvTransport = useCallback(async () => {
    console.log('WebRTC: createRecvTransport skipped');
  }, []);

  const produce = useCallback(async (_track: MediaStreamTrack, _appData?: any) => {
    console.log('WebRTC: produce skipped');
    return null;
  }, []);

  const consume = useCallback(async (
    _producerId: string,
    _remotePeerId: string,
    _kind: 'audio' | 'video'
  ) => {
    console.log('WebRTC: consume skipped');
    return null;
  }, []);

  const closeProducer = useCallback(async (_producerId: string) => {
    console.log('WebRTC: closeProducer skipped');
  }, []);

  const closeAll = useCallback(() => {
    setRemoteStreams(new Map());
  }, []);

  return {
    remoteStreams,
    loadDevice,
    createSendTransport,
    createRecvTransport,
    produce,
    consume,
    closeProducer,
    closeAll,
  };
};
