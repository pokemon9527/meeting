package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type MeetingStatus string

const (
	MeetingStatusScheduled MeetingStatus = "scheduled"
	MeetingStatusWaiting   MeetingStatus = "waiting"
	MeetingStatusActive    MeetingStatus = "active"
	MeetingStatusEnded     MeetingStatus = "ended"
)

type Meeting struct {
	ID                 uuid.UUID      `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	MeetingID          string         `gorm:"type:varchar(6);uniqueIndex;not null" json:"meeting_id"`
	Title              string         `gorm:"type:varchar(100);not null" json:"title"`
	Description        string         `gorm:"type:text" json:"description"`
	HostID             uuid.UUID      `gorm:"type:uuid;not null" json:"host_id"`
	Host               User           `gorm:"foreignKey:HostID" json:"host,omitempty"`
	Password           *string        `gorm:"type:varchar(255)" json:"-"`
	Status             MeetingStatus  `gorm:"type:varchar(20);default:'waiting'" json:"status"`
	MaxParticipants    int            `gorm:"default:200" json:"max_participants"`
	EnableWaitingRoom  bool           `gorm:"default:false" json:"enable_waiting_room"`
	EnableRecording    bool           `gorm:"default:true" json:"enable_recording"`
	AllowScreenShare   bool           `gorm:"default:true" json:"allow_screen_share"`
	AllowChat          bool           `gorm:"default:true" json:"allow_chat"`
	AllowWhiteboard    bool           `gorm:"default:true" json:"allow_whiteboard"`
	MuteOnEntry        bool           `gorm:"default:false" json:"mute_on_entry"`
	VideoOnEntry       bool           `gorm:"default:true" json:"video_on_entry"`
	ScheduledStartTime *time.Time     `json:"scheduled_start_time"`
	ActualStartTime    *time.Time     `json:"actual_start_time"`
	EndTime            *time.Time     `json:"end_time"`
	CreatedAt          time.Time      `json:"created_at"`
	UpdatedAt          time.Time      `json:"updated_at"`
	DeletedAt          gorm.DeletedAt `gorm:"index" json:"-"`
}

func (m *Meeting) BeforeCreate(tx *gorm.DB) error {
	if m.ID == uuid.Nil {
		m.ID = uuid.New()
	}
	return nil
}

func (m *Meeting) HasPassword() bool {
	return m.Password != nil && *m.Password != ""
}

func (m *Meeting) CheckPassword(password string) bool {
	if !m.HasPassword() {
		return true
	}
	return *m.Password == password
}
