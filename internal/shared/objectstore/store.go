package objectstore

import (
	"context"
	"io"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"

	"aphrodite/pkg/config"
)

type Store interface {
	Put(ctx context.Context, key string, body io.Reader, size int64, contentType string) error
	Delete(ctx context.Context, key string) error
	PresignedGet(ctx context.Context, key string, expiry time.Duration) (string, error)
}

type MinIO struct {
	client *minio.Client
	bucket string
}

type S3 = MinIO

var _ Store = (*MinIO)(nil)

func New(cfg config.StorageConfig) (*MinIO, error) {
	client, err := minio.New(cfg.Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.AccessKey, cfg.SecretKey, ""),
		Secure: cfg.UseSSL,
		Region: cfg.Region,
		BucketLookup: func() minio.BucketLookupType {
			if cfg.ForcePathStyle {
				return minio.BucketLookupPath
			}
			return minio.BucketLookupAuto
		}(),
	})
	if err != nil {
		return nil, err
	}
	return &MinIO{client: client, bucket: cfg.Bucket}, nil
}

func NewS3(cfg config.StorageConfig) (*S3, error) {
	return New(cfg)
}

func (s *MinIO) EnsureBucket(ctx context.Context) error {
	exists, err := s.client.BucketExists(ctx, s.bucket)
	if err != nil {
		return err
	}
	if exists {
		return nil
	}
	return s.client.MakeBucket(ctx, s.bucket, minio.MakeBucketOptions{})
}

func (s *MinIO) Put(ctx context.Context, key string, body io.Reader, size int64, contentType string) error {
	_, err := s.client.PutObject(ctx, s.bucket, key, body, size, minio.PutObjectOptions{ContentType: contentType})
	return err
}

func (s *MinIO) Delete(ctx context.Context, key string) error {
	return s.client.RemoveObject(ctx, s.bucket, key, minio.RemoveObjectOptions{})
}

func (s *MinIO) PresignedGet(ctx context.Context, key string, expiry time.Duration) (string, error) {
	url, err := s.client.PresignedGetObject(ctx, s.bucket, key, expiry, nil)
	if err != nil {
		return "", err
	}
	return url.String(), nil
}
