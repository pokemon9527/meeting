package main

import (
	"log"

	"github.com/gin-gonic/gin"

	"meeting-go/internal/config"
	"meeting-go/internal/database"
	"meeting-go/internal/handlers"
	"meeting-go/internal/middleware"
	"meeting-go/internal/models"
	jwtUtil "meeting-go/pkg/jwt"
)

func main() {
	cfg := config.Load()

	jwtUtil.Init(cfg.JWTSecret, cfg.JWTRefresh)

	db := database.NewPostgres(cfg.PostgresDSN)

	database.AutoMigrate(db,
		&models.User{},
		&models.Meeting{},
		&models.Participant{},
		&models.ChatMessage{},
		&models.Recording{},
		&models.RecordingSegment{},
		&models.RecordingJob{},
		&models.RecordingAsset{},
	)

	r := gin.Default()

	r.Use(middleware.CORS())

	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	api := r.Group("/api")
	{
		auth := api.Group("/auth")
		{
			auth.POST("/register", handlers.Register(db))
			auth.POST("/login", handlers.Login(db))
			auth.POST("/refresh", handlers.RefreshToken())
			auth.GET("/me", middleware.Auth(), handlers.GetCurrentUser(db))
		}

		meetings := api.Group("/meetings")
		meetings.Use(middleware.Auth())
		{
			meetings.POST("", handlers.CreateMeeting(db))
			meetings.GET("", handlers.GetMyMeetings(db))
			meetings.GET("/:meetingId", handlers.GetMeeting(db))
			meetings.POST("/join", handlers.JoinMeeting(db))
			meetings.PUT("/:meetingId/settings", handlers.UpdateMeetingSettings(db))
			meetings.POST("/:meetingId/end", handlers.EndMeeting(db))
			meetings.DELETE("/:meetingId", handlers.DeleteMeeting(db))
		}

		recordings := api.Group("/recordings")
		recordings.Use(middleware.Auth())
		{
			recordings.GET("", handlers.GetMyRecordings(db))
			recordings.GET("/:recordingId", handlers.GetRecording(db))
			recordings.GET("/:recordingId/playlist", handlers.GetRecordingPlaylist(db))
			recordings.GET("/:recordingId/segments", handlers.GetRecordingSegments(db))
			recordings.DELETE("/:recordingId", handlers.DeleteRecording(db))
		}
	}

	log.Printf("API server starting on port %s", cfg.APIPort)
	if err := r.Run(":" + cfg.APIPort); err != nil {
		log.Fatal("Failed to start server:", err)
	}
}
