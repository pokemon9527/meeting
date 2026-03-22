import React, { useState } from 'react';
import { useParams } from 'react-router-dom';
import { Card, List, Space, Button, Spin, message, Select, Typography, Empty, Tag } from 'antd';
import { PlayCircleOutlined, DeleteOutlined, DownloadOutlined } from '@ant-design/icons';
import axios from 'axios';
import dayjs from 'dayjs';
import duration from 'dayjs/plugin/duration';
import HLSPlayer from '../../components/HLSPlayer';
import './RecordingPlayback.css';

dayjs.extend(duration);

interface RecordingAsset {
  id: string;
  quality: string;
  playlist_path: string;
  total_segments: number;
  total_size: number;
  duration_seconds: number;
  is_primary: boolean;
}

interface Recording {
  id: string;
  meeting_id: string;
  title: string;
  status: string;
  duration_seconds: number;
  participant_count: number;
  created_at: string;
  actual_start_time: string;
  end_time: string;
  assets: RecordingAsset[];
}

interface PlaylistInfo {
  recording_id: string;
  meeting_id: string;
  title: string;
  quality: string;
  playlist_path: string;
  total_segments: number;
  duration_seconds: number;
  participants: {
    participant_id: string;
    participant_name: string;
    start_time: string;
    end_time: string;
    sequence: number;
  }[];
}

const { Title, Text } = Typography;

const RecordingPlayback: React.FC = () => {
  const { recordingId } = useParams<{ recordingId: string }>();
  const [recordings, setRecordings] = useState<Recording[]>([]);
  const [loading, setLoading] = useState(true);
  const [selectedRecording, setSelectedRecording] = useState<Recording | null>(null);
  const [playlistInfo, setPlaylistInfo] = useState<PlaylistInfo | null>(null);
  const [loadingPlaylist, setLoadingPlaylist] = useState(false);
  const [selectedQuality, setSelectedQuality] = useState<string>('720p');

  useState(() => {
    fetchRecordings();
  });

  const fetchRecordings = async () => {
    try {
      setLoading(true);
      const token = localStorage.getItem('token');
      const response = await axios.get('/api/recordings', {
        headers: { Authorization: `Bearer ${token}` },
      });
      if (response.data.success) {
        setRecordings(response.data.data);
      }
    } catch (error) {
      message.error('Failed to fetch recordings');
    } finally {
      setLoading(false);
    }
  };

  const loadPlaylist = async (recording: Recording, quality?: string) => {
    try {
      setSelectedRecording(recording);
      setLoadingPlaylist(true);
      const token = localStorage.getItem('token');
      const qualityParam = quality || selectedQuality;
      const response = await axios.get(`/api/recordings/${recording.id}/playlist`, {
        params: { quality: qualityParam },
        headers: { Authorization: `Bearer ${token}` },
      });
      if (response.data.success) {
        setPlaylistInfo(response.data.data);
      }
    } catch (error) {
      message.error('Failed to load recording');
    } finally {
      setLoadingPlaylist(false);
    }
  };

  const handleQualityChange = (value: string) => {
    setSelectedQuality(value);
    if (selectedRecording) {
      loadPlaylist(selectedRecording, value);
    }
  };

  const handleDelete = async (recordingId: string) => {
    try {
      const token = localStorage.getItem('token');
      const response = await axios.delete(`/api/recordings/${recordingId}`, {
        headers: { Authorization: `Bearer ${token}` },
      });
      if (response.data.success) {
        message.success('Recording deleted');
        fetchRecordings();
        if (selectedRecording?.id === recordingId) {
          setSelectedRecording(null);
          setPlaylistInfo(null);
        }
      }
    } catch (error) {
      message.error('Failed to delete recording');
    }
  };

  const getStatusColor = (status: string) => {
    switch (status) {
      case 'completed':
        return 'green';
      case 'processing':
        return 'blue';
      case 'failed':
        return 'red';
      default:
        return 'default';
    }
  };

  const formatDuration = (seconds: number) => {
    const d = dayjs.duration(seconds, 'seconds');
    return d.format('HH:mm:ss');
  };

  const formatFileSize = (bytes: number) => {
    if (bytes === 0) return '0 B';
    const k = 1024;
    const sizes = ['B', 'KB', 'MB', 'GB'];
    const i = Math.floor(Math.log(bytes) / Math.log(k));
    return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i];
  };

  if (loading) {
    return (
      <div className="recording-playback-loading">
        <Spin size="large" />
      </div>
    );
  }

  return (
    <div className="recording-playback-container">
      <div className="recording-list-section">
        <Title level={4}>My Recordings</Title>
        {recordings.length === 0 ? (
          <Empty description="No recordings yet" />
        ) : (
          <List
            dataSource={recordings}
            renderItem={(recording) => (
              <List.Item
                className={`recording-item ${selectedRecording?.id === recording.id ? 'selected' : ''}`}
                actions={[
                  <Button
                    type="text"
                    icon={<DeleteOutlined />}
                    onClick={() => handleDelete(recording.id)}
                    danger
                  />,
                ]}
              >
                <List.Item.Meta
                  title={
                    <Space>
                      <span>{recording.title}</span>
                      <Tag color={getStatusColor(recording.status)}>
                        {recording.status}
                      </Tag>
                    </Space>
                  }
                  description={
                    <Space direction="vertical" size="small">
                      <Text type="secondary">
                        Meeting ID: {recording.meeting_id}
                      </Text>
                      <Text type="secondary">
                        {dayjs(recording.created_at).format('YYYY-MM-DD HH:mm')}
                      </Text>
                      <Text type="secondary">
                        Duration: {formatDuration(recording.duration_seconds)}
                      </Text>
                    </Space>
                  }
                />
                {recording.status === 'completed' && (
                  <Button
                    type="primary"
                    icon={<PlayCircleOutlined />}
                    onClick={() => loadPlaylist(recording)}
                  >
                    Play
                  </Button>
                )}
              </List.Item>
            )}
          />
        )}
      </div>

      <div className="recording-player-section">
        {loadingPlaylist ? (
          <div className="recording-playback-loading">
            <Spin size="large" />
          </div>
        ) : playlistInfo ? (
          <>
            <div className="player-header">
              <Title level={4}>{playlistInfo.title}</Title>
              <Space>
                <Text>Quality:</Text>
                <Select
                  value={selectedQuality}
                  onChange={handleQualityChange}
                  style={{ width: 120 }}
                >
                  <Select.Option value="1080p">1080p</Select.Option>
                  <Select.Option value="720p">720p</Select.Option>
                  <Select.Option value="360p">360p</Select.Option>
                </Select>
                <Button icon={<DownloadOutlined />}>Download</Button>
              </Space>
            </div>
            <div className="player-wrapper">
              <HLSPlayer
                src={`/api/recordings/${recordingId}/stream/${playlistInfo.playlist_path}`}
                autoplay
                onError={(err) => message.error(err.message)}
              />
            </div>
            <Card className="participant-timeline" title="Timeline">
              <div className="timeline-list">
                {playlistInfo.participants.map((p, index) => (
                  <div key={index} className="timeline-item">
                    <Tag color="blue">{p.participant_name}</Tag>
                    <Text type="secondary">
                      {dayjs(p.start_time).format('HH:mm:ss')} -{' '}
                      {dayjs(p.end_time).format('HH:mm:ss')}
                    </Text>
                  </div>
                ))}
              </div>
            </Card>
          </>
        ) : (
          <div className="no-recording-selected">
            <Empty description="Select a recording to play" />
          </div>
        )}
      </div>
    </div>
  );
};

export default RecordingPlayback;
