package storage

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

type Storage interface {
	Upload(ctx context.Context, bucket, objectName string, reader io.Reader, size int64, contentType string) error
	Download(ctx context.Context, bucket, objectName string, writer io.Writer) error
	Delete(ctx context.Context, bucket, objectName string) error
	List(ctx context.Context, bucket, prefix string) ([]ObjectInfo, error)
	Exists(ctx context.Context, bucket, objectName string) (bool, error)
	GetPresignedURL(ctx context.Context, bucket, objectName string, expiry time.Duration) (string, error)
}

type ObjectInfo struct {
	Key          string
	Size         int64
	LastModified time.Time
}

type MinIOStorage struct {
	client *minio.Client
}

type MinIOConfig struct {
	Endpoint  string
	AccessKey string
	SecretKey string
	Bucket    string
	UseSSL    bool
}

func NewMinIOStorage(cfg MinIOConfig) (*MinIOStorage, error) {
	client, err := minio.New(cfg.Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.AccessKey, cfg.SecretKey, ""),
		Secure: cfg.UseSSL,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create minio client: %w", err)
	}

	ctx := context.Background()
	exists, err := client.BucketExists(ctx, cfg.Bucket)
	if err != nil {
		return nil, fmt.Errorf("failed to check bucket: %w", err)
	}

	if !exists {
		err = client.MakeBucket(ctx, cfg.Bucket, minio.MakeBucketOptions{})
		if err != nil {
			return nil, fmt.Errorf("failed to create bucket: %w", err)
		}
		log.Printf("Created bucket: %s", cfg.Bucket)
	}

	return &MinIOStorage{client: client}, nil
}

func (m *MinIOStorage) Upload(ctx context.Context, bucket, objectName string, reader io.Reader, size int64, contentType string) error {
	_, err := m.client.PutObject(ctx, bucket, objectName, reader, size, minio.PutObjectOptions{
		ContentType: contentType,
	})
	if err != nil {
		return fmt.Errorf("failed to upload object: %w", err)
	}
	return nil
}

func (m *MinIOStorage) UploadBytes(ctx context.Context, bucket, objectName string, data []byte, contentType string) error {
	return m.Upload(ctx, bucket, objectName, bytes.NewReader(data), int64(len(data)), contentType)
}

func (m *MinIOStorage) Download(ctx context.Context, bucket, objectName string, writer io.Writer) error {
	obj, err := m.client.GetObject(ctx, bucket, objectName, minio.GetObjectOptions{})
	if err != nil {
		return fmt.Errorf("failed to get object: %w", err)
	}
	defer obj.Close()

	_, err = io.Copy(writer, obj)
	if err != nil {
		return fmt.Errorf("failed to download object: %w", err)
	}
	return nil
}

func (m *MinIOStorage) DownloadToFile(ctx context.Context, bucket, objectName, filePath string) error {
	obj, err := m.client.GetObject(ctx, bucket, objectName, minio.GetObjectOptions{})
	if err != nil {
		return fmt.Errorf("failed to get object: %w", err)
	}
	defer obj.Close()

	localFile, err := os.Create(filePath)
	if err != nil {
		return fmt.Errorf("failed to create local file: %w", err)
	}
	defer localFile.Close()

	_, err = io.Copy(localFile, obj)
	if err != nil {
		return fmt.Errorf("failed to copy to local file: %w", err)
	}
	return nil
}

func (m *MinIOStorage) Delete(ctx context.Context, bucket, objectName string) error {
	err := m.client.RemoveObject(ctx, bucket, objectName, minio.RemoveObjectOptions{})
	if err != nil {
		return fmt.Errorf("failed to delete object: %w", err)
	}
	return nil
}

func (m *MinIOStorage) List(ctx context.Context, bucket, prefix string) ([]ObjectInfo, error) {
	var objects []ObjectInfo

	ch := m.client.ListObjects(ctx, bucket, minio.ListObjectsOptions{
		Prefix:    prefix,
		Recursive: true,
	})

	for obj := range ch {
		if obj.Err != nil {
			return nil, fmt.Errorf("failed to list objects: %w", obj.Err)
		}
		objects = append(objects, ObjectInfo{
			Key:          obj.Key,
			Size:         obj.Size,
			LastModified: obj.LastModified,
		})
	}
	return objects, nil
}

func (m *MinIOStorage) Exists(ctx context.Context, bucket, objectName string) (bool, error) {
	_, err := m.client.StatObject(ctx, bucket, objectName, minio.StatObjectOptions{})
	if err != nil {
		errResp := minio.ToErrorResponse(err)
		if errResp.Code == "NoSuchKey" {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func (m *MinIOStorage) GetPresignedURL(ctx context.Context, bucket, objectName string, expiry time.Duration) (string, error) {
	URL, err := m.client.PresignedGetObject(ctx, bucket, objectName, expiry, nil)
	if err != nil {
		return "", fmt.Errorf("failed to generate presigned URL: %w", err)
	}
	return URL.String(), nil
}

func init() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
}
