-- Create recordings table
CREATE TABLE IF NOT EXISTS recordings (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    meeting_id VARCHAR(6) NOT NULL,
    title VARCHAR(100) NOT NULL,
    host_id UUID NOT NULL,
    status VARCHAR(20) DEFAULT 'pending',
    duration_seconds BIGINT DEFAULT 0,
    participant_count INT DEFAULT 0,
    scheduled_start_time TIMESTAMP WITH TIME ZONE,
    actual_start_time TIMESTAMP WITH TIME ZONE,
    end_time TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    deleted_at TIMESTAMP WITH TIME ZONE
);

CREATE INDEX idx_recordings_meeting_id ON recordings(meeting_id);
CREATE INDEX idx_recordings_host_id ON recordings(host_id);
CREATE INDEX idx_recordings_status ON recordings(status);
CREATE INDEX idx_recordings_created_at ON recordings(created_at);

-- Create recording_segments table (原始 TS 片段)
CREATE TABLE IF NOT EXISTS recording_segments (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    recording_id UUID NOT NULL REFERENCES recordings(id) ON DELETE CASCADE,
    participant_id VARCHAR(255) NOT NULL,
    participant_name VARCHAR(100),
    start_time TIMESTAMP WITH TIME ZONE NOT NULL,
    end_time TIMESTAMP WITH TIME ZONE NOT NULL,
    segment_path VARCHAR(500) NOT NULL,
    file_size BIGINT DEFAULT 0,
    sequence_number INT DEFAULT 0,
    status VARCHAR(20) DEFAULT 'pending',
    transcoded BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    deleted_at TIMESTAMP WITH TIME ZONE
);

CREATE INDEX idx_segments_recording_id ON recording_segments(recording_id);
CREATE INDEX idx_segments_participant_id ON recording_segments(participant_id);
CREATE INDEX idx_segments_status ON recording_segments(status);
CREATE INDEX idx_segments_start_time ON recording_segments(start_time);

-- Create recording_jobs table (转码任务)
CREATE TABLE IF NOT EXISTS recording_jobs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    recording_id UUID NOT NULL REFERENCES recordings(id) ON DELETE CASCADE,
    segment_id UUID REFERENCES recording_segments(id),
    status VARCHAR(20) DEFAULT 'queued',
    quality VARCHAR(20) DEFAULT '1080p',
    input_path VARCHAR(500),
    output_path VARCHAR(500),
    error_message TEXT,
    retry_count INT DEFAULT 0,
    started_at TIMESTAMP WITH TIME ZONE,
    completed_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX idx_jobs_recording_id ON recording_jobs(recording_id);
CREATE INDEX idx_jobs_status ON recording_jobs(status);

-- Create recording_assets table (转码后的 HLS 资产)
CREATE TABLE IF NOT EXISTS recording_assets (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    recording_id UUID NOT NULL REFERENCES recordings(id) ON DELETE CASCADE,
    segment_id UUID REFERENCES recording_segments(id),
    quality VARCHAR(20) NOT NULL,
    playlist_path VARCHAR(500) NOT NULL,
    total_segments INT DEFAULT 0,
    total_size BIGINT DEFAULT 0,
    duration_seconds BIGINT DEFAULT 0,
    is_primary BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX idx_assets_recording_id ON recording_assets(recording_id);
CREATE INDEX idx_assets_quality ON recording_assets(quality);
