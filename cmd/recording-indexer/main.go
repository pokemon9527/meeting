package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"meeting-go/internal/kafka"
	"meeting-go/internal/models"
)

type Config struct {
	KafkaBrokers   []string
	DBDSN          string
	MinIOEndpoint  string
	MinIOAccessKey string
	MinIOSecretKey string
	MinIOBucket    string
	MinIOUseSSL    bool
}

func main() {
	cfg := Config{
		KafkaBrokers:   []string{getEnv("KAFKA_BROKERS", "localhost:9092")},
		DBDSN:          getEnv("POSTGRES_DSN", "host=localhost user=postgres password=postgres dbname=meeting port=5432 sslmode=disable"),
		MinIOEndpoint:  getEnv("MINIO_ENDPOINT", "localhost:9000"),
		MinIOAccessKey: getEnv("MINIO_ACCESS_KEY", "minioadmin"),
		MinIOSecretKey: getEnv("MINIO_SECRET_KEY", "minioadmin"),
		MinIOBucket:    getEnv("MINIO_BUCKET", "recordings"),
		MinIOUseSSL:    getEnv("MINIO_USE_SSL", "false") == "true",
	}

	log.Println("Connecting to PostgreSQL...")
	db, err := gorm.Open(postgres.Open(cfg.DBDSN), &gorm.Config{})
	if err != nil {
		log.Fatalf("Failed to connect to PostgreSQL: %v", err)
	}
	log.Println("Connected to PostgreSQL")

	consumer := kafka.NewConsumer(cfg.KafkaBrokers, kafka.TopicRecordingTranscoded, "recording-indexer-group")
	defer consumer.Close()

	log.Println("Recording Indexer started, waiting for messages...")

	ctx := context.Background()
	err = consumer.Consume(ctx, func(data []byte) error {
		var msg kafka.TranscodedMessage
		if err := json.Unmarshal(data, &msg); err != nil {
			log.Printf("Failed to unmarshal message: %v", err)
			return err
		}

		log.Printf("Indexing transcoded asset: %s for recording: %s", msg.SegmentID, msg.RecordingID)
		return indexAsset(db, msg)
	})

	if err != nil && err != context.Canceled {
		log.Fatalf("Consumer error: %v", err)
	}
}

func indexAsset(db *gorm.DB, msg kafka.TranscodedMessage) error {
	var recording models.Recording
	if err := db.First(&recording, "id = ?", msg.RecordingID).Error; err != nil {
		log.Printf("Recording not found: %s", msg.RecordingID)
		return err
	}

	var asset models.RecordingAsset
	if err := db.First(&asset, "playlist_path = ?", msg.PlaylistPath).Error; err == nil {
		asset.TotalSegments = msg.TotalSegments
		asset.TotalSize = msg.TotalSize
		asset.DurationSeconds = msg.DurationSeconds
		db.Save(&asset)
	} else {
		log.Printf("Asset not found in DB, creating: %s", msg.PlaylistPath)
	}

	var pendingSegments int64
	db.Model(&models.RecordingSegment{}).Where("recording_id = ? AND transcoded = ?", msg.RecordingID, false).Count(&pendingSegments)

	if pendingSegments == 0 {
		var segments []models.RecordingSegment
		db.Where("recording_id = ?", msg.RecordingID).Order("start_time ASC").Find(&segments)

		var totalDuration int64
		for _, s := range segments {
			totalDuration += int64(s.EndTime.Sub(s.StartTime).Seconds())
		}

		recording.DurationSeconds = totalDuration
		recording.Status = models.RecordingStatusCompleted
		recording.EndTime = func() *time.Time { t := time.Now(); return &t }()
		db.Save(&recording)

		log.Printf("Recording %s completed. Total duration: %d seconds", msg.RecordingID, totalDuration)
	}

	return nil
}

func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}

func formatDuration(seconds int64) string {
	h := seconds / 3600
	m := (seconds % 3600) / 60
	s := seconds % 60
	return fmt.Sprintf("%02d:%02d:%02d", h, m, s)
}

func extractSegmentInfo(m3u8Content string) (segments int, duration float64) {
	lines := strings.Split(m3u8Content, "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "#EXTINF:") {
			parts := strings.Split(strings.TrimPrefix(line, "#EXTINF:"), ",")
			if len(parts) > 0 {
				var d float64
				fmt.Sscanf(parts[0], "%f", &d)
				duration += d
				segments++
			}
		}
	}
	return segments, duration
}
