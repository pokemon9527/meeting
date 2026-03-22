import React, { useState, useEffect } from 'react';
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
  onToggleChat?: () => void;
  onToggleParticipants?: () => void;
  isScreenSharing?: boolean;
}

export const Toolbar: React.FC<ToolbarProps> = ({
  onToggleAudio,
  onToggleVideo,
  onToggleScreenShare,
  onLeaveMeeting,
  onToggleChat,
  onToggleParticipants,
  isScreenSharing: isScreenSharingProp,
}) => {
  const {
    localAudioEnabled,
    localVideoEnabled,
    isHandRaised,
    unreadMessages,
    activePanel,
    setActivePanel,
    toggleHandRaise,
    peers,
  } = useMeetingStore();

  const [currentTime, setCurrentTime] = useState(new Date());

  useEffect(() => {
    const timer = setInterval(() => {
      setCurrentTime(new Date());
    }, 1000);
    return () => clearInterval(timer);
  }, []);

  const screenSharing = isScreenSharingProp ?? false;

  const handleToggleChat = () => {
    if (onToggleChat) {
      onToggleChat();
    } else {
      setActivePanel(activePanel === 'chat' ? null : 'chat');
    }
  };

  const handleToggleParticipants = () => {
    if (onToggleParticipants) {
      onToggleParticipants();
    } else {
      setActivePanel(activePanel === 'participants' ? null : 'participants');
    }
  };

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
            className="toolbar-btn"
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
            className="toolbar-btn"
          />
        </Tooltip>

        <Tooltip title={screenSharing ? '停止共享' : '共享屏幕'}>
          <Button
            type={screenSharing ? 'primary' : 'default'}
            shape="circle"
            size="large"
            icon={<DesktopOutlined />}
            onClick={onToggleScreenShare}
            className={`toolbar-btn ${screenSharing ? 'toolbar-btn-active' : ''}`}
          />
        </Tooltip>

        <Tooltip title={isHandRaised ? '放下手' : '举手'}>
          <Button
            type={isHandRaised ? 'primary' : 'default'}
            shape="circle"
            size="large"
            icon={<LikeOutlined />}
            onClick={toggleHandRaise}
            className="toolbar-btn"
          />
        </Tooltip>

        <div className="toolbar-divider" />

        <Tooltip title="聊天">
          <Badge count={unreadMessages} size="small" offset={[0, -4]}>
            <Button
              type={activePanel === 'chat' ? 'primary' : 'default'}
              shape="circle"
              size="large"
              icon={<MessageOutlined />}
              onClick={handleToggleChat}
              className="toolbar-btn"
            />
          </Badge>
        </Tooltip>

        <Tooltip title="参与者">
          <Button
            type={activePanel === 'participants' ? 'primary' : 'default'}
            shape="circle"
            size="large"
            icon={<TeamOutlined />}
            onClick={handleToggleParticipants}
            className="toolbar-btn"
          />
        </Tooltip>

        <div className="toolbar-divider" />

        <Tooltip title="离开会议">
          <Button
            type="primary"
            danger
            shape="circle"
            size="large"
            icon={<LogoutOutlined />}
            onClick={onLeaveMeeting}
            className="toolbar-btn toolbar-btn-leave"
          />
        </Tooltip>
      </div>

      <div className="toolbar-right">
        <span className="time-display">
          {currentTime.toLocaleTimeString('zh-CN', { 
            hour: '2-digit', 
            minute: '2-digit' 
          })}
        </span>
      </div>
    </div>
  );
};
