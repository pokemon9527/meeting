package handlers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"

	"meeting-go/internal/models"
	"meeting-go/pkg/response"
)

func GetMyRecordings(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := c.MustGet("user_id").(uuid.UUID)

		var recordings []models.Recording
		if err := db.Where("host_id = ?", userID).
			Order("created_at DESC").
			Preload("Assets").
			Find(&recordings).Error; err != nil {
			response.Error(c, http.StatusInternalServerError, "Failed to fetch recordings")
			return
		}

		var result []gin.H
		for _, r := range recordings {
			item := gin.H{
				"id":                r.ID,
				"meeting_id":        r.MeetingID,
				"title":             r.Title,
				"status":            r.Status,
				"duration_seconds":  r.DurationSeconds,
				"participant_count": r.ParticipantCount,
				"created_at":        r.CreatedAt,
				"actual_start_time": r.ActualStartTime,
				"end_time":          r.EndTime,
				"assets":            r.Assets,
			}
			result = append(result, item)
		}

		response.Success(c, http.StatusOK, result)
	}
}

func GetRecording(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		recordingID := c.Param("recordingId")

		var recording models.Recording
		if err := db.Preload("Segments").
			Preload("Assets").
			First(&recording, "id = ?", recordingID).Error; err != nil {
			response.Error(c, http.StatusNotFound, "Recording not found")
			return
		}

		response.Success(c, http.StatusOK, gin.H{
			"id":                recording.ID,
			"meeting_id":        recording.MeetingID,
			"title":             recording.Title,
			"status":            recording.Status,
			"duration_seconds":  recording.DurationSeconds,
			"participant_count": recording.ParticipantCount,
			"created_at":        recording.CreatedAt,
			"actual_start_time": recording.ActualStartTime,
			"end_time":          recording.EndTime,
			"segments":          recording.Segments,
			"assets":            recording.Assets,
		})
	}
}

func GetRecordingPlaylist(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		recordingID := c.Param("recordingId")
		quality := c.DefaultQuery("quality", "720p")

		var asset models.RecordingAsset
		if err := db.Where("recording_id = ? AND quality = ?", recordingID, quality).First(&asset).Error; err != nil {
			var defaultAsset models.RecordingAsset
			if err := db.Where("recording_id = ? AND is_primary = ?", recordingID, true).First(&defaultAsset).Error; err != nil {
				response.Error(c, http.StatusNotFound, "Playlist not found")
				return
			}
			asset = defaultAsset
		}

		var recording models.Recording
		db.First(&recording, "id = ?", recordingID)

		var segments []models.RecordingSegment
		db.Where("recording_id = ?", recordingID).Order("start_time ASC").Find(&segments)

		var participantTimeline []gin.H
		for _, seg := range segments {
			participantTimeline = append(participantTimeline, gin.H{
				"participant_id":   seg.ParticipantID,
				"participant_name": seg.ParticipantName,
				"start_time":       seg.StartTime,
				"end_time":         seg.EndTime,
				"sequence":         seg.SequenceNumber,
			})
		}

		response.Success(c, http.StatusOK, gin.H{
			"recording_id":     recordingID,
			"meeting_id":       recording.MeetingID,
			"title":            recording.Title,
			"quality":          asset.Quality,
			"playlist_path":    asset.PlaylistPath,
			"total_segments":   asset.TotalSegments,
			"duration_seconds": asset.DurationSeconds,
			"participants":     participantTimeline,
		})
	}
}

func GetRecordingSegments(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		recordingID := c.Param("recordingId")

		var segments []models.RecordingSegment
		if err := db.Where("recording_id = ?", recordingID).
			Order("start_time ASC").
			Find(&segments).Error; err != nil {
			response.Error(c, http.StatusInternalServerError, "Failed to fetch segments")
			return
		}

		var result []gin.H
		for _, s := range segments {
			result = append(result, gin.H{
				"id":               s.ID,
				"participant_id":   s.ParticipantID,
				"participant_name": s.ParticipantName,
				"start_time":       s.StartTime,
				"end_time":         s.EndTime,
				"duration_seconds": s.EndTime.Sub(s.StartTime).Seconds(),
				"sequence":         s.SequenceNumber,
				"transcoded":       s.Transcoded,
				"status":           s.Status,
			})
		}

		response.Success(c, http.StatusOK, result)
	}
}

func DeleteRecording(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		recordingID := c.Param("recordingId")

		var recording models.Recording
		if err := db.First(&recording, "id = ?", recordingID).Error; err != nil {
			response.Error(c, http.StatusNotFound, "Recording not found")
			return
		}

		db.Where("recording_id = ?", recordingID).Delete(&models.RecordingSegment{})
		db.Where("recording_id = ?", recordingID).Delete(&models.RecordingAsset{})
		db.Where("recording_id = ?", recordingID).Delete(&models.RecordingJob{})
		db.Delete(&recording)

		response.Success(c, http.StatusOK, gin.H{"message": "Recording deleted"})
	}
}

func CreateRecording(db *gorm.DB, meetingID string, hostID uuid.UUID) (*models.Recording, error) {
	now := time.Now()
	recording := &models.Recording{
		MeetingID:       meetingID,
		Title:           "Meeting Recording",
		HostID:          hostID,
		Status:          models.RecordingStatusPending,
		ActualStartTime: &now,
	}

	if err := db.Create(recording).Error; err != nil {
		return nil, err
	}

	return recording, nil
}

func CreateSegment(db *gorm.DB, recordingID uuid.UUID, participantID, participantName string, startTime time.Time, segmentPath string) (*models.RecordingSegment, error) {
	segment := &models.RecordingSegment{
		RecordingID:     recordingID,
		ParticipantID:   participantID,
		ParticipantName: participantName,
		StartTime:       startTime,
		EndTime:         time.Now(),
		SegmentPath:     segmentPath,
		Status:          models.SegmentStatusRecording,
	}

	if err := db.Create(segment).Error; err != nil {
		return nil, err
	}

	return segment, nil
}

func EndSegment(db *gorm.DB, segmentID uuid.UUID) error {
	return db.Model(&models.RecordingSegment{}).
		Where("id = ?", segmentID).
		Updates(map[string]interface{}{
			"end_time": time.Now(),
			"status":   models.SegmentStatusCompleted,
		}).Error
}

func EndRecording(db *gorm.DB, recordingID uuid.UUID) error {
	now := time.Now()
	return db.Model(&models.Recording{}).
		Where("id = ?", recordingID).
		Updates(map[string]interface{}{
			"end_time": now,
			"status":   models.RecordingStatusProcessing,
		}).Error
}
