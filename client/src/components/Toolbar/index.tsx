import React from 'react';
import {
  AudioOutlined,
  AudioMutedOutlined,
  VideoCameraOutlined,
  VideoCameraAddOutlined,
  DesktopOutlined,
  MessageOutlined,
  TeamOutlined,
  LogoutOutlined,
  LikeOutlined,
} from '@ant-design/icons';
import { Button, Badge, Tooltip } from 'antd';
import { useMeetingStore } from '../../stores/meetingStore';

interface ToolbarProps {
  onToggleAudio: () => void;
  onToggleVideo: () => void;
  onToggleScreenShare: () => void;
  onLeaveMeeting: () => void;
}

export const Toolbar: React.FC<ToolbarProps> = ({
  onToggleAudio,
  onToggleVideo,
  onToggleScreenShare,
  onLeaveMeeting,
}) => {
  const {
    localAudioEnabled,
    localVideoEnabled,
    isScreenSharing,
    isHandRaised,
    unreadMessages,
    activePanel,
    setActivePanel,
    toggleHandRaise,
    peers,
  } = useMeetingStore();

  return (
    <div className="toolbar">
      <div className="toolbar-left">
        <div className="meeting-info">
          <span className="meeting-title">
            参会人数: {peers.size}
          </span>
        </div>
      </div>

      <div className="toolbar-center">
        <Tooltip title={localAudioEnabled ? '静音' : '取消静音'}>
          <Button
            type="primary"
            shape="circle"
            size="large"
            icon={localAudioEnabled ? <AudioOutlined /> : <AudioMutedOutlined />}
            onClick={onToggleAudio}
            danger={!localAudioEnabled}
          />
        </Tooltip>

        <Tooltip title={localVideoEnabled ? '关闭视频' : '开启视频'}>
          <Button
            type="primary"
            shape="circle"
            size="large"
            icon={localVideoEnabled ? <VideoCameraOutlined /> : <VideoCameraAddOutlined />}
            onClick={onToggleVideo}
            danger={!localVideoEnabled}
          />
        </Tooltip>

        <Tooltip title={isScreenSharing ? '停止共享' : '共享屏幕'}>
          <Button
            type={isScreenSharing ? 'primary' : 'default'}
            shape="circle"
            size="large"
            icon={<DesktopOutlined />}
            onClick={onToggleScreenShare}
          />
        </Tooltip>

        <Tooltip title={isHandRaised ? '放下手' : '举手'}>
          <Button
            type={isHandRaised ? 'primary' : 'default'}
            shape="circle"
            size="large"
            icon={<LikeOutlined />}
            onClick={toggleHandRaise}
          />
        </Tooltip>

        <Tooltip title="聊天">
          <Badge count={unreadMessages} size="small">
            <Button
              type={activePanel === 'chat' ? 'primary' : 'default'}
              shape="circle"
              size="large"
              icon={<MessageOutlined />}
              onClick={() =>
                setActivePanel(activePanel === 'chat' ? null : 'chat')
              }
            />
          </Badge>
        </Tooltip>

        <Tooltip title="参与者">
          <Button
            type={activePanel === 'participants' ? 'primary' : 'default'}
            shape="circle"
            size="large"
            icon={<TeamOutlined />}
            onClick={() =>
              setActivePanel(
                activePanel === 'participants' ? null : 'participants'
              )
            }
          />
        </Tooltip>

        <Tooltip title="离开会议">
          <Button
            type="primary"
            danger
            shape="circle"
            size="large"
            icon={<LogoutOutlined />}
            onClick={onLeaveMeeting}
          />
        </Tooltip>
      </div>

      <div className="toolbar-right">
        <span className="time-display">
          {new Date().toLocaleTimeString()}
        </span>
      </div>
    </div>
  );
};
