package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/google/uuid"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"meeting-go/internal/kafka"
	"meeting-go/internal/models"
	"meeting-go/internal/storage"
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

	log.Println("Connecting to MinIO...")
	minioStorage, err := storage.NewMinIOStorage(storage.MinIOConfig{
		Endpoint:  cfg.MinIOEndpoint,
		AccessKey: cfg.MinIOAccessKey,
		SecretKey: cfg.MinIOSecretKey,
		Bucket:    cfg.MinIOBucket,
		UseSSL:    cfg.MinIOUseSSL,
	})
	if err != nil {
		log.Fatalf("Failed to connect to MinIO: %v", err)
	}
	log.Println("Connected to MinIO")

	consumer := kafka.NewConsumer(cfg.KafkaBrokers, kafka.TopicRecordingRaw, "recording-worker-group")
	defer consumer.Close()

	log.Println("Recording Worker started, waiting for messages...")

	ctx := context.Background()
	err = consumer.Consume(ctx, func(data []byte) error {
		var msg kafka.RawSegmentMessage
		if err := json.Unmarshal(data, &msg); err != nil {
			log.Printf("Failed to unmarshal message: %v", err)
			return err
		}

		log.Printf("Processing segment: %s for recording: %s", msg.SegmentID, msg.RecordingID)
		return processSegment(ctx, db, minioStorage, cfg.MinIOBucket, msg)
	})

	if err != nil && err != context.Canceled {
		log.Fatalf("Consumer error: %v", err)
	}
}

func processSegment(ctx context.Context, db *gorm.DB, store *storage.MinIOStorage, bucket string, msg kafka.RawSegmentMessage) error {
	var segment models.RecordingSegment
	if err := db.First(&segment, "id = ?", msg.SegmentID).Error; err != nil {
		log.Printf("Segment not found: %s", msg.SegmentID)
		return err
	}

	segment.Status = models.SegmentStatusTranscoding
	db.Save(&segment)

	job := models.RecordingJob{
		RecordingID: uuid.MustParse(msg.RecordingID),
		SegmentID:   &segment.ID,
		Status:      models.JobStatusProcessing,
		Quality:     "1080p",
		InputPath:   msg.SegmentPath,
	}
	now := time.Now()
	job.StartedAt = &now
	db.Create(&job)

	tmpDir := filepath.Join(os.TempDir(), "recording-work", msg.SegmentID)
	os.MkdirAll(tmpDir, 0755)
	defer os.RemoveAll(tmpDir)

	inputFile := filepath.Join(tmpDir, "input.ts")
	if err := store.DownloadToFile(ctx, bucket, msg.SegmentPath, inputFile); err != nil {
		log.Printf("Failed to download input file: %v", err)
		job.Status = models.JobStatusFailed
		job.ErrorMessage = err.Error()
		db.Save(&job)
		return err
	}

	qualities := []struct {
		name    string
		width   string
		height  string
		bitrate string
	}{
		{"1080p", "1920", "1080", "5000k"},
		{"720p", "1280", "720", "2500k"},
		{"360p", "640", "360", "1000k"},
	}

	var assets []models.RecordingAsset

	for _, q := range qualities {
		outputDir := filepath.Join(tmpDir, q.name)
		os.MkdirAll(outputDir, 0755)

		outputM3U8 := filepath.Join(outputDir, "playlist.m3u8")

		cmd := exec.Command("ffmpeg", "-i", inputFile,
			"-c:v", "libx264", "-preset", "medium", "-b:v", q.bitrate,
			"-vf", fmt.Sprintf("scale=%s:%s", q.width, q.height),
			"-c:a", "aac", "-b:a", "128k",
			"-f", "hls",
			"-hls_time", "10",
			"-hls_list_size", "0",
			"-hls_segment_filename", filepath.Join(outputDir, "segment%03d.ts"),
			outputM3U8,
		)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr

		log.Printf("Transcoding to %s...", q.name)
		if err := cmd.Run(); err != nil {
			log.Printf("FFmpeg failed for %s: %v", q.name, err)
			continue
		}

		playlistPath := fmt.Sprintf("transcoded/%s/%s/%s", msg.RecordingID, msg.SegmentID, q.name)

		segmentFiles, _ := filepath.Glob(filepath.Join(outputDir, "*.ts"))
		var totalSize int64
		for _, f := range segmentFiles {
			info, _ := os.Stat(f)
			if info != nil {
				totalSize += info.Size()
			}

			tsName := filepath.Base(f)
			tsData, _ := os.ReadFile(f)
			if err := store.UploadBytes(ctx, bucket, filepath.Join(playlistPath, tsName), tsData, "video/MP2T"); err != nil {
				log.Printf("Failed to upload ts file: %v", err)
			}
		}

		m3u8Data, _ := os.ReadFile(outputM3U8)
		if err := store.UploadBytes(ctx, bucket, filepath.Join(playlistPath, "playlist.m3u8"), m3u8Data, "application/x-mpegURL"); err != nil {
			log.Printf("Failed to upload m3u8: %v", err)
		}

		job.OutputPath = playlistPath
		job.Status = models.JobStatusCompleted
		db.Save(&job)

		completed := time.Now()
		job.CompletedAt = &completed

		asset := models.RecordingAsset{
			RecordingID:     uuid.MustParse(msg.RecordingID),
			SegmentID:       &segment.ID,
			Quality:         q.name,
			PlaylistPath:    filepath.Join(playlistPath, "playlist.m3u8"),
			TotalSegments:   len(segmentFiles),
			TotalSize:       totalSize,
			DurationSeconds: int64(segment.EndTime.Sub(segment.StartTime).Seconds()),
			IsPrimary:       q.name == "720p",
		}
		db.Create(&asset)
		assets = append(assets, asset)

		log.Printf("Completed transcoding to %s: %d segments, %d bytes", q.name, len(segmentFiles), totalSize)

		producer := kafka.NewProducer(kafkaBrokers(), kafka.TopicRecordingTranscoded)
		transcodedMsg := kafka.TranscodedMessage{
			RecordingID:     msg.RecordingID,
			SegmentID:       msg.SegmentID,
			Quality:         q.name,
			PlaylistPath:    asset.PlaylistPath,
			TotalSegments:   asset.TotalSegments,
			TotalSize:       asset.TotalSize,
			DurationSeconds: asset.DurationSeconds,
		}
		if err := producer.PublishTranscoded(ctx, transcodedMsg); err != nil {
			log.Printf("Failed to publish transcoded message: %v", err)
		}
		producer.Close()
	}

	segment.Status = models.SegmentStatusTranscoded
	segment.Transcoded = true
	db.Save(&segment)

	log.Printf("Segment %s transcoding completed", msg.SegmentID)
	return nil
}

func kafkaBrokers() []string {
	brokers := os.Getenv("KAFKA_BROKERS")
	if brokers == "" {
		return []string{"localhost:9092"}
	}
	return []string{brokers}
}

func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}
