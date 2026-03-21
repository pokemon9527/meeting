package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Participant struct {
	ID        uuid.UUID  `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	MeetingID uuid.UUID  `gorm:"type:uuid;not null" json:"meeting_id"`
	UserID    uuid.UUID  `gorm:"type:uuid;not null" json:"user_id"`
	User      User       `gorm:"foreignKey:UserID" json:"user,omitempty"`
	Role      string     `gorm:"type:varchar(20);default:'participant'" json:"role"`
	JoinedAt  time.Time  `gorm:"default:now()" json:"joined_at"`
	LeftAt    *time.Time `json:"left_at"`
	Duration  int        `gorm:"default:0" json:"duration"`
}

func (p *Participant) BeforeCreate(tx *gorm.DB) error {
	if p.ID == uuid.Nil {
		p.ID = uuid.New()
	}
	return nil
}
