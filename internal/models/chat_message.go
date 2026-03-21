package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type ChatMessage struct {
	ID         uuid.UUID      `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	MeetingID  string         `gorm:"type:varchar(6);not null;index" json:"meeting_id"`
	SenderID   uuid.UUID      `gorm:"type:uuid;not null" json:"sender_id"`
	Sender     User           `gorm:"foreignKey:SenderID" json:"sender,omitempty"`
	ReceiverID *uuid.UUID     `gorm:"type:uuid" json:"receiver_id,omitempty"`
	Type       string         `gorm:"type:varchar(20);default:'text'" json:"type"`
	Content    string         `gorm:"type:text;not null" json:"content"`
	FileName   string         `gorm:"type:varchar(255)" json:"file_name,omitempty"`
	FileURL    string         `gorm:"type:text" json:"file_url,omitempty"`
	FileSize   int64          `json:"file_size,omitempty"`
	MimeType   string         `gorm:"type:varchar(100)" json:"mime_type,omitempty"`
	CreatedAt  time.Time      `json:"created_at"`
	DeletedAt  gorm.DeletedAt `gorm:"index" json:"-"`
}

func (m *ChatMessage) BeforeCreate(tx *gorm.DB) error {
	if m.ID == uuid.Nil {
		m.ID = uuid.New()
	}
	return nil
}
