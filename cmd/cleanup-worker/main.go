package main

import (
	"context"
	"log"
	"os"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"meeting-go/internal/models"
	"meeting-go/internal/storage"
)

type Config struct {
	DBDSN                   string
	MinIOEndpoint           string
	MinIOAccessKey          string
	MinIOSecretKey          string
	MinIOBucket             string
	MinIOUseSSL             bool
	RawRetentionDays        int
	TranscodedRetentionDays int
}

func main() {
	cfg := Config{
		DBDSN:                   getEnv("POSTGRES_DSN", "host=localhost user=postgres password=postgres dbname=meeting port=5432 sslmode=disable"),
		MinIOEndpoint:           getEnv("MINIO_ENDPOINT", "localhost:9000"),
		MinIOAccessKey:          getEnv("MINIO_ACCESS_KEY", "minioadmin"),
		MinIOSecretKey:          getEnv("MINIO_SECRET_KEY", "minioadmin"),
		MinIOBucket:             getEnv("MINIO_BUCKET", "recordings"),
		MinIOUseSSL:             getEnv("MINIO_USE_SSL", "false") == "true",
		RawRetentionDays:        1,
		TranscodedRetentionDays: 30,
	}

	log.Println("Connecting to PostgreSQL...")
	db, err := gorm.Open(postgres.Open(cfg.DBDSN), &gorm.Config{})
	if err != nil {
		log.Fatalf("Failed to connect to PostgreSQL: %v", err)
	}
	log.Println("Connected to PostgreSQL")

	log.Println("Connecting to MinIO...")
	store, err := storage.NewMinIOStorage(storage.MinIOConfig{
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

	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()

	runCleanup(context.Background(), db, store, cfg)

	for range ticker.C {
		runCleanup(context.Background(), db, store, cfg)
	}
}

func runCleanup(ctx context.Context, db *gorm.DB, store *storage.MinIOStorage, cfg Config) {
	log.Println("Starting cleanup job...")

	cutoffRaw := time.Now().AddDate(0, 0, -cfg.RawRetentionDays)
	cutoffTranscoded := time.Now().AddDate(0, 0, -cfg.TranscodedRetentionDays)

	var rawSegments []models.RecordingSegment
	db.Where("transcoded = ? AND created_at < ?", false, cutoffRaw).Find(&rawSegments)

	log.Printf("Found %d raw segments to delete (created before %s)", len(rawSegments), cutoffRaw)

	for _, segment := range rawSegments {
		if err := store.Delete(ctx, cfg.MinIOBucket, segment.SegmentPath); err != nil {
			log.Printf("Failed to delete segment from storage: %s - %v", segment.SegmentPath, err)
		}

		db.Delete(&segment)
		log.Printf("Deleted raw segment: %s", segment.ID)
	}

	var transcodedAssets []models.RecordingAsset
	db.Where("created_at < ?", cutoffTranscoded).Find(&transcodedAssets)

	log.Printf("Found %d transcoded assets to delete (created before %s)", len(transcodedAssets), cutoffTranscoded)

	deletedRecordings := make(map[string]bool)

	for _, asset := range transcodedAssets {
		parts := []string{}
		for _, p := range []string{asset.PlaylistPath} {
			idx := 0
			for i := len(p) - 1; i >= 0; i-- {
				if p[i] == '/' {
					idx = i
					break
				}
			}
			if idx > 0 {
				parts = append(parts, p[:idx])
			}
		}

		prefix := ""
		for _, part := range parts {
			if len(part) > 0 {
				prefix = part
				break
			}
		}

		if prefix != "" {
			objects, err := store.List(ctx, cfg.MinIOBucket, prefix)
			if err != nil {
				log.Printf("Failed to list objects for deletion: %v", err)
				continue
			}

			for _, obj := range objects {
				if err := store.Delete(ctx, cfg.MinIOBucket, obj.Key); err != nil {
					log.Printf("Failed to delete object: %s - %v", obj.Key, err)
				}
			}
		}

		db.Delete(&asset)
		log.Printf("Deleted transcoded asset: %s (path: %s)", asset.ID, asset.PlaylistPath)

		deletedRecordings[asset.RecordingID.String()] = true
	}

	for recordingID := range deletedRecordings {
		var segments []models.RecordingSegment
		db.Where("recording_id = ?", recordingID).Find(&segments)

		allTranscoded := true
		for _, s := range segments {
			if !s.Transcoded {
				allTranscoded = false
				break
			}
		}

		if allTranscoded && len(segments) > 0 {
			var assets []models.RecordingAsset
			db.Where("recording_id = ?", recordingID).Find(&assets)

			if len(assets) == 0 {
				db.Where("recording_id = ?", recordingID).Delete(&models.RecordingSegment{})
				db.Where("recording_id = ?", recordingID).Delete(&models.RecordingJob{})
				db.Where("recording_id = ?", recordingID).Delete(&models.RecordingAsset{})
				db.Where("id = ?", recordingID).Delete(&models.Recording{})
				log.Printf("Deleted completed recording: %s", recordingID)
			}
		}
	}

	log.Println("Cleanup job completed")
}

func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}
