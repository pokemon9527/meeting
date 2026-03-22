package handlers

import (
	"math/rand"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"meeting-go/internal/models"
	jwtUtil "meeting-go/pkg/jwt"
	"meeting-go/pkg/response"
)

type CreateMeetingRequest struct {
	Title       string `json:"title" binding:"required"`
	Description string `json:"description"`
	Password    string `json:"password"`
}

type JoinMeetingRequest struct {
	MeetingID string `json:"meetingId" binding:"required"`
	Password  string `json:"password"`
}

func CreateMeeting(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := c.GetString("user_id")

		var req CreateMeetingRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			response.BadRequest(c, err.Error())
			return
		}

		meetingID := generateMeetingID(db)

		meeting := models.Meeting{
			MeetingID:   meetingID,
			Title:       req.Title,
			Description: req.Description,
			HostID:      parseUUID(userID),
			Status:      models.MeetingStatusWaiting,
		}

		if req.Password != "" {
			meeting.Password = &req.Password
		}

		if err := db.Create(&meeting).Error; err != nil {
			response.InternalError(c, "创建会议失败")
			return
		}

		response.SuccessWithMessage(c, 201, "会议创建成功", gin.H{"meeting": meeting})
	}
}

func JoinMeeting(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req JoinMeetingRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			response.BadRequest(c, err.Error())
			return
		}

		var meeting models.Meeting
		if err := db.Where("meeting_id = ?", req.MeetingID).First(&meeting).Error; err != nil {
			response.NotFound(c, "会议不存在")
			return
		}

		if meeting.Status == models.MeetingStatusEnded {
			response.BadRequest(c, "会议已结束")
			return
		}

		if !meeting.CheckPassword(req.Password) {
			response.Unauthorized(c, "会议密码错误")
			return
		}

		joinToken, _ := jwtUtil.GenerateToken(req.MeetingID, "", "")

		response.Success(c, 200, gin.H{
			"meeting": gin.H{
				"meeting_id": meeting.MeetingID,
				"title":      meeting.Title,
			},
			"token": joinToken,
		})
	}
}

func GetMeeting(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		meetingID := c.Param("meetingId")

		var meeting models.Meeting
		if err := db.Preload("Host").Where("meeting_id = ?", meetingID).First(&meeting).Error; err != nil {
			response.NotFound(c, "会议不存在")
			return
		}

		response.Success(c, 200, gin.H{"meeting": meeting})
	}
}

func GetMyMeetings(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := c.GetString("user_id")

		var meetings []models.Meeting
		db.Where("host_id = ?", userID).Order("created_at DESC").Limit(50).Find(&meetings)

		response.Success(c, 200, gin.H{"meetings": meetings})
	}
}

func UpdateMeetingSettings(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		meetingID := c.Param("meetingId")
		userID := c.GetString("user_id")

		var meeting models.Meeting
		if err := db.Where("meeting_id = ?", meetingID).First(&meeting).Error; err != nil {
			response.NotFound(c, "会议不存在")
			return
		}

		if meeting.HostID.String() != userID {
			response.Forbidden(c, "无权限修改")
			return
		}

		var settings map[string]interface{}
		if err := c.ShouldBindJSON(&settings); err != nil {
			response.BadRequest(c, err.Error())
			return
		}

		db.Model(&meeting).Updates(settings)
		response.Success(c, 200, gin.H{"meeting": meeting})
	}
}

func EndMeeting(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		meetingID := c.Param("meetingId")
		userID := c.GetString("user_id")

		var meeting models.Meeting
		if err := db.Where("meeting_id = ?", meetingID).First(&meeting).Error; err != nil {
			response.NotFound(c, "会议不存在")
			return
		}

		if meeting.HostID.String() != userID {
			response.Forbidden(c, "无权限结束会议")
			return
		}

		now := time.Now()
		meeting.Status = models.MeetingStatusEnded
		meeting.EndTime = &now
		db.Save(&meeting)

		response.SuccessWithMessage(c, 200, "会议已结束", nil)
	}
}

func DeleteMeeting(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		meetingID := c.Param("meetingId")
		userID := c.GetString("user_id")

		var meeting models.Meeting
		if err := db.Where("meeting_id = ?", meetingID).First(&meeting).Error; err != nil {
			response.NotFound(c, "会议不存在")
			return
		}

		if meeting.HostID.String() != userID {
			response.Forbidden(c, "无权限删除")
			return
		}

		db.Where("meeting_id = ?", meetingID).Delete(&models.ChatMessage{})
		db.Delete(&meeting)

		response.SuccessWithMessage(c, 200, "会议已删除", nil)
	}
}

func generateMeetingID(db *gorm.DB) string {
	for {
		id := strconv.Itoa(100000 + rand.Intn(900000))
		var count int64
		db.Model(&models.Meeting{}).Where("meeting_id = ?", id).Count(&count)
		if count == 0 {
			return id
		}
	}
}
