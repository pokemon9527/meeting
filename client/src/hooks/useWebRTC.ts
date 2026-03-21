import { useRef, useCallback, useState } from 'react';
import * as mediasoupClient from 'mediasoup-client';
import { getSocket } from './useSocket';

interface RemoteStream {
  peerId: string;
  producerId: string;
  kind: 'audio' | 'video';
  stream: MediaStream;
}

export const useWebRTC = () => {
  const deviceRef = useRef<mediasoupClient.Device | null>(null);
  const sendTransportRef = useRef<mediasoupClient.types.Transport | null>(null);
  const recvTransportRef = useRef<mediasoupClient.types.Transport | null>(null);
  const producersRef = useRef<Map<string, mediasoupClient.types.Producer>>(new Map());
  const consumersRef = useRef<Map<string, mediasoupClient.types.Consumer>>(new Map());
  const [remoteStreams, setRemoteStreams] = useState<Map<string, RemoteStream>>(new Map());

  const loadDevice = useCallback(async (rtpCapabilities: any) => {
    const device = new mediasoupClient.Device();
    await device.load({ routerRtpCapabilities: rtpCapabilities });
    deviceRef.current = device;
    console.log('mediasoup Device loaded');
    return device;
  }, []);

  const createSendTransport = useCallback(async (): Promise<mediasoupClient.types.Transport> => {
    const socket = getSocket();
    if (!socket || !deviceRef.current) {
      throw new Error('Socket or device not initialized');
    }

    return new Promise((resolve, reject) => {
      socket.emit('create-transport', { direction: 'send' }, async (response: any) => {
        if (!response.success) {
          reject(new Error(response.error));
          return;
        }

        const transport = deviceRef.current!.createSendTransport({
          id: response.id,
          iceParameters: response.iceParameters,
          iceCandidates: response.iceCandidates,
          dtlsParameters: response.dtlsParameters,
        });

        transport.on('connect', async ({ dtlsParameters }, callback, errback) => {
          try {
            socket.emit(
              'connect-transport',
              { transportId: transport.id, dtlsParameters },
              (res: any) => {
                if (res.success) {
                  callback();
                } else {
                  errback(new Error(res.error));
                }
              }
            );
          } catch (error: any) {
            errback(error);
          }
        });

        transport.on('produce', async ({ kind, rtpParameters, appData }, callback, errback) => {
          try {
            socket.emit(
              'produce',
              {
                transportId: transport.id,
                kind,
                rtpParameters,
                appData,
              },
              (res: any) => {
                if (res.success) {
                  callback({ id: res.id });
                } else {
                  errback(new Error(res.error));
                }
              }
            );
          } catch (error: any) {
            errback(error);
          }
        });

        sendTransportRef.current = transport;
        resolve(transport);
      });
    });
  }, []);

  const createRecvTransport = useCallback(async (): Promise<mediasoupClient.types.Transport> => {
    const socket = getSocket();
    if (!socket || !deviceRef.current) {
      throw new Error('Socket or device not initialized');
    }

    return new Promise((resolve, reject) => {
      socket.emit('create-transport', { direction: 'recv' }, async (response: any) => {
        if (!response.success) {
          reject(new Error(response.error));
          return;
        }

        const transport = deviceRef.current!.createRecvTransport({
          id: response.id,
          iceParameters: response.iceParameters,
          iceCandidates: response.iceCandidates,
          dtlsParameters: response.dtlsParameters,
        });

        transport.on('connect', async ({ dtlsParameters }, callback, errback) => {
          try {
            socket.emit(
              'connect-transport',
              { transportId: transport.id, dtlsParameters },
              (res: any) => {
                if (res.success) {
                  callback();
                } else {
                  errback(new Error(res.error));
                }
              }
            );
          } catch (error: any) {
            errback(error);
          }
        });

        recvTransportRef.current = transport;
        resolve(transport);
      });
    });
  }, []);

  const produce = useCallback(
    async (track: MediaStreamTrack, appData?: any): Promise<mediasoupClient.types.Producer> => {
      if (!sendTransportRef.current) {
        throw new Error('Send transport not created');
      }

      const producer = await sendTransportRef.current.produce({
        track,
        appData,
      });

      producersRef.current.set(producer.id, producer);

      producer.on('transportclose', () => {
        producersRef.current.delete(producer.id);
      });

      return producer;
    },
    []
  );

  const consume = useCallback(
    async (
      producerId: string,
      remotePeerId: string,
      kind: 'audio' | 'video'
    ): Promise<RemoteStream> => {
      const socket = getSocket();
      if (!socket || !recvTransportRef.current || !deviceRef.current) {
        throw new Error('Socket, transport or device not initialized');
      }

      return new Promise((resolve, reject) => {
        socket.emit(
          'consume',
          {
            transportId: recvTransportRef.current!.id,
            producerId,
            rtpCapabilities: deviceRef.current!.rtpCapabilities,
          },
          async (response: any) => {
            if (!response.success) {
              reject(new Error(response.error));
              return;
            }

            try {
              const consumer = await recvTransportRef.current!.consume({
                id: response.id,
                producerId: response.producerId,
                kind: response.kind,
                rtpParameters: response.rtpParameters,
              });

              consumersRef.current.set(consumer.id, consumer);

              consumer.on('transportclose', () => {
                consumersRef.current.delete(consumer.id);
              });

              (consumer as any).on('producerclose', () => {
                consumersRef.current.delete(consumer.id);
                const key = `${remotePeerId}_${producerId}`;
                setRemoteStreams((prev) => {
                  const newMap = new Map(prev);
                  newMap.delete(key);
                  return newMap;
                });
              });

              socket.emit('consumer-resume', { consumerId: consumer.id });

              const stream = new MediaStream([consumer.track]);
              const remoteStream: RemoteStream = {
                peerId: remotePeerId,
                producerId,
                kind,
                stream,
              };

              const key = `${remotePeerId}_${producerId}`;
              setRemoteStreams((prev) => {
                const newMap = new Map(prev);
                newMap.set(key, remoteStream);
                return newMap;
              });

              resolve(remoteStream);
            } catch (error) {
              reject(error);
            }
          }
        );
      });
    },
    []
  );

  const closeProducer = useCallback(async (producerId: string) => {
    const socket = getSocket();
    const producer = producersRef.current.get(producerId);

    if (producer) {
      producer.close();
      producersRef.current.delete(producerId);
    }

    if (socket) {
      socket.emit('close-producer', { producerId });
    }
  }, []);

  const closeAll = useCallback(() => {
    producersRef.current.forEach((producer) => producer.close());
    consumersRef.current.forEach((consumer) => consumer.close());
    sendTransportRef.current?.close();
    recvTransportRef.current?.close();

    producersRef.current.clear();
    consumersRef.current.clear();
    sendTransportRef.current = null;
    recvTransportRef.current = null;
    setRemoteStreams(new Map());
  }, []);

  return {
    device: deviceRef.current,
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
