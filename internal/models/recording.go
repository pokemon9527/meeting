package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type RecordingStatus string

const (
	RecordingStatusPending    RecordingStatus = "pending"
	RecordingStatusRecording  RecordingStatus = "recording"
	RecordingStatusProcessing RecordingStatus = "processing"
	RecordingStatusCompleted  RecordingStatus = "completed"
	RecordingStatusFailed     RecordingStatus = "failed"
)

type SegmentStatus string

const (
	SegmentStatusPending     SegmentStatus = "pending"
	SegmentStatusRecording   SegmentStatus = "recording"
	SegmentStatusCompleted   SegmentStatus = "completed"
	SegmentStatusTranscoding SegmentStatus = "transcoding"
	SegmentStatusTranscoded  SegmentStatus = "transcoded"
	SegmentStatusFailed      SegmentStatus = "failed"
)

type JobStatus string

const (
	JobStatusQueued     JobStatus = "queued"
	JobStatusProcessing JobStatus = "processing"
	JobStatusCompleted  JobStatus = "completed"
	JobStatusFailed     JobStatus = "failed"
)

type Recording struct {
	ID                 uuid.UUID          `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	MeetingID          string             `gorm:"type:varchar(6);not null;index" json:"meeting_id"`
	Title              string             `gorm:"type:varchar(100);not null" json:"title"`
	HostID             uuid.UUID          `gorm:"type:uuid;not null" json:"host_id"`
	Status             RecordingStatus    `gorm:"type:varchar(20);default:'pending'" json:"status"`
	DurationSeconds    int64              `gorm:"default:0" json:"duration_seconds"`
	ParticipantCount   int                `gorm:"default:0" json:"participant_count"`
	ScheduledStartTime *time.Time         `json:"scheduled_start_time"`
	ActualStartTime    *time.Time         `json:"actual_start_time"`
	EndTime            *time.Time         `json:"end_time"`
	CreatedAt          time.Time          `json:"created_at"`
	UpdatedAt          time.Time          `json:"updated_at"`
	DeletedAt          gorm.DeletedAt     `gorm:"index" json:"-"`
	Segments           []RecordingSegment `gorm:"foreignKey:RecordingID" json:"segments,omitempty"`
	Assets             []RecordingAsset   `gorm:"foreignKey:RecordingID" json:"assets,omitempty"`
}

func (r *Recording) BeforeCreate(tx *gorm.DB) error {
	if r.ID == uuid.Nil {
		r.ID = uuid.New()
	}
	return nil
}

type RecordingSegment struct {
	ID              uuid.UUID      `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	RecordingID     uuid.UUID      `gorm:"type:uuid;not null;index" json:"recording_id"`
	ParticipantID   string         `gorm:"type:varchar(255);not null;index" json:"participant_id"`
	ParticipantName string         `gorm:"type:varchar(100)" json:"participant_name"`
	StartTime       time.Time      `gorm:"not null;index" json:"start_time"`
	EndTime         time.Time      `gorm:"not null" json:"end_time"`
	SegmentPath     string         `gorm:"type:varchar(500);not null" json:"segment_path"`
	FileSize        int64          `gorm:"default:0" json:"file_size"`
	SequenceNumber  int            `gorm:"default:0" json:"sequence_number"`
	Status          SegmentStatus  `gorm:"type:varchar(20);default:'pending'" json:"status"`
	Transcoded      bool           `gorm:"default:false" json:"transcoded"`
	CreatedAt       time.Time      `json:"created_at"`
	UpdatedAt       time.Time      `json:"updated_at"`
	DeletedAt       gorm.DeletedAt `gorm:"index" json:"-"`
}

func (s *RecordingSegment) BeforeCreate(tx *gorm.DB) error {
	if s.ID == uuid.Nil {
		s.ID = uuid.New()
	}
	return nil
}

type RecordingJob struct {
	ID           uuid.UUID  `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	RecordingID  uuid.UUID  `gorm:"type:uuid;not null;index" json:"recording_id"`
	SegmentID    *uuid.UUID `gorm:"type:uuid" json:"segment_id"`
	Status       JobStatus  `gorm:"type:varchar(20);default:'queued';index" json:"status"`
	Quality      string     `gorm:"type:varchar(20);default:'1080p'" json:"quality"`
	InputPath    string     `gorm:"type:varchar(500)" json:"input_path"`
	OutputPath   string     `gorm:"type:varchar(500)" json:"output_path"`
	ErrorMessage string     `gorm:"type:text" json:"error_message"`
	RetryCount   int        `gorm:"default:0" json:"retry_count"`
	StartedAt    *time.Time `json:"started_at"`
	CompletedAt  *time.Time `json:"completed_at"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
}

func (j *RecordingJob) BeforeCreate(tx *gorm.DB) error {
	if j.ID == uuid.Nil {
		j.ID = uuid.New()
	}
	return nil
}

type RecordingAsset struct {
	ID              uuid.UUID  `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	RecordingID     uuid.UUID  `gorm:"type:uuid;not null;index" json:"recording_id"`
	SegmentID       *uuid.UUID `gorm:"type:uuid" json:"segment_id"`
	Quality         string     `gorm:"type:varchar(20);not null" json:"quality"`
	PlaylistPath    string     `gorm:"type:varchar(500);not null" json:"playlist_path"`
	TotalSegments   int        `gorm:"default:0" json:"total_segments"`
	TotalSize       int64      `gorm:"default:0" json:"total_size"`
	DurationSeconds int64      `gorm:"default:0" json:"duration_seconds"`
	IsPrimary       bool       `gorm:"default:false" json:"is_primary"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
}

func (a *RecordingAsset) BeforeCreate(tx *gorm.DB) error {
	if a.ID == uuid.Nil {
		a.ID = uuid.New()
	}
	return nil
}
