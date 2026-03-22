import React from 'react';
import { List, Avatar, Tag, Button, Dropdown } from 'antd';
import {
  AudioOutlined,
  AudioMutedOutlined,
  VideoCameraOutlined,
  VideoCameraAddOutlined,
  MoreOutlined,
  LikeOutlined,
} from '@ant-design/icons';
import { useMeetingStore } from '../../stores/meetingStore';
import { getSocket } from '../../hooks/useSocket';

export const ParticipantList: React.FC = () => {
  const { peers, peerId } = useMeetingStore();
  const peerArray = Array.from(peers.values());

  const handleMute = (targetPeerId: string) => {
    const socket = getSocket();
    if (socket && socket.readyState === WebSocket.OPEN) {
      socket.send(JSON.stringify({
        event: 'mute-participant',
        data: { target_peer_id: targetPeerId },
      }));
    }
  };

  const handleRemove = (targetPeerId: string) => {
    const socket = getSocket();
    if (socket && socket.readyState === WebSocket.OPEN) {
      socket.send(JSON.stringify({
        event: 'remove-participant',
        data: { target_peer_id: targetPeerId },
      }));
    }
  };

  const localPeer = peerId ? peers.get(peerId) : null;
  const isHost = localPeer?.role === 'host';

  return (
    <div className="participant-list">
      <div className="participant-header">
        <h3>参与者 ({peerArray.length})</h3>
      </div>

      <List
        dataSource={peerArray}
        renderItem={(peer) => (
          <List.Item
            key={peer.id}
            actions={
              isHost && peer.id !== peerId
                ? [
                    <Dropdown
                      menu={{
                        items: [
                          {
                            key: 'mute',
                            label: peer.audioEnabled ? '静音' : '取消静音',
                            onClick: () => handleMute(peer.id),
                          },
                          {
                            key: 'remove',
                            label: '移除',
                            danger: true,
                            onClick: () => handleRemove(peer.id),
                          },
                        ],
                      }}
                    >
                      <Button type="text" icon={<MoreOutlined />} />
                    </Dropdown>,
                  ]
                : []
            }
          >
            <List.Item.Meta
              avatar={<Avatar src={peer.avatar}>{(peer.username || '?').charAt(0).toUpperCase()}</Avatar>}
              title={
                <div className="participant-title">
                  <span>{peer.username || '未知用户'}</span>
                  {peer.id === peerId && <Tag color="blue">我</Tag>}
                  {peer.role === 'host' && <Tag color="gold">主持人</Tag>}
                  {peer.role === 'cohost' && <Tag color="green">联席主持人</Tag>}
                  {peer.isHandRaised && (
                    <LikeOutlined style={{ color: '#faad14' }} />
                  )}
                </div>
              }
              description={
                <div className="participant-status">
                  {peer.audioEnabled ? (
                    <AudioOutlined style={{ color: '#52c41a' }} />
                  ) : (
                    <AudioMutedOutlined style={{ color: '#ff4d4f' }} />
                  )}
                  {peer.videoEnabled ? (
                    <VideoCameraOutlined style={{ color: '#52c41a', marginLeft: 8 }} />
                  ) : (
                    <VideoCameraAddOutlined style={{ color: '#ff4d4f', marginLeft: 8 }} />
                  )}
                </div>
              }
            />
          </List.Item>
        )}
      />
    </div>
  );
};
