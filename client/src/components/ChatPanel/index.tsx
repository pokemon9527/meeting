import React, { useState, useRef, useEffect } from 'react';
import { Input, Button, Avatar } from 'antd';
import { SendOutlined } from '@ant-design/icons';
import { useMeetingStore } from '../../stores/meetingStore';
import { useUserStore } from '../../stores/userStore';
import { getSocket } from '../../hooks/useSocket';

export const ChatPanel: React.FC = () => {
  const [inputValue, setInputValue] = useState('');
  const messagesEndRef = useRef<HTMLDivElement>(null);
  const { messages, clearUnreadMessages } = useMeetingStore();
  const { user } = useUserStore();

  useEffect(() => {
    clearUnreadMessages();
  }, [clearUnreadMessages]);

  useEffect(() => {
    messagesEndRef.current?.scrollIntoView({ behavior: 'smooth' });
  }, [messages]);

  const handleSend = () => {
    if (!inputValue.trim()) return;

    const socket = getSocket();
    if (socket) {
      socket.emit('send-message', {
        content: inputValue.trim(),
        type: 'text',
      });
    }

    setInputValue('');
  };

  const handleKeyPress = (e: React.KeyboardEvent) => {
    if (e.key === 'Enter' && !e.shiftKey) {
      e.preventDefault();
      handleSend();
    }
  };

  const formatTime = (date: Date) => {
    const d = new Date(date);
    return d.toLocaleTimeString('zh-CN', {
      hour: '2-digit',
      minute: '2-digit',
    });
  };

  return (
    <div className="chat-panel">
      <div className="chat-header">
        <h3>聊天</h3>
      </div>

      <div className="chat-messages">
        {messages.map((msg) => (
          <div
            key={msg.id}
            className={`message ${msg.senderId === user?.id ? 'self' : ''}`}
          >
            <Avatar size="small" src={msg.senderAvatar}>
              {msg.senderName.charAt(0)}
            </Avatar>
            <div className="message-content">
              <div className="message-header">
                <span className="sender-name">{msg.senderName}</span>
                <span className="message-time">{formatTime(msg.timestamp)}</span>
              </div>
              <div className="message-text">{msg.content}</div>
            </div>
          </div>
        ))}
        <div ref={messagesEndRef} />
      </div>

      <div className="chat-input">
        <Input.TextArea
          value={inputValue}
          onChange={(e) => setInputValue(e.target.value)}
          onKeyPress={handleKeyPress}
          placeholder="输入消息..."
          autoSize={{ minRows: 1, maxRows: 4 }}
        />
        <Button
          type="primary"
          icon={<SendOutlined />}
          onClick={handleSend}
          disabled={!inputValue.trim()}
        />
      </div>
    </div>
  );
};
