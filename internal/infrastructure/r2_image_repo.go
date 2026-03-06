package infrastructure

import (
	"context"
	"fmt"
	"io"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/google/uuid"
)

type R2StorageConfig struct {
	AccountID       string
	AccessKeyID     string
	AccessKeySecret string
	Bucket          string
	PublicURL       string
}

type R2Storage struct {
	client    *s3.Client
	bucket    string
	publicURL string
}

func NewR2Storage(opts *R2StorageConfig) (*R2Storage, error) {
	cfg, err := config.LoadDefaultConfig(context.TODO(),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(opts.AccessKeyID, opts.AccessKeySecret, "")),
		config.WithRegion("auto"), // required by SDK, not used by R2
	)
	if err != nil {
		return nil, fmt.Errorf("failed to load R2 config: %w", err)
	}

	client := s3.NewFromConfig(cfg, func(o *s3.Options) {
		// ✅ Current approach — BaseEndpoint directly on s3.Options
		o.BaseEndpoint = aws.String(fmt.Sprintf("https://%s.r2.cloudflarestorage.com", opts.AccountID))
	})

	return &R2Storage{
		client:    client,
		bucket:    opts.Bucket,
		publicURL: opts.PublicURL,
	}, nil
}

func (r *R2Storage) Upload(ctx context.Context, id uuid.UUID, file io.Reader, contentType string) (string, error) {
	key := id.String()

	_, err := r.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:       aws.String(r.bucket),
		Key:          aws.String(key),
		Body:         file,
		ContentType:  aws.String(contentType),
		CacheControl: aws.String("public, max-age=2592000"),
	})
	if err != nil {
		return "", fmt.Errorf("failed to upload to R2: %w", err)
	}

	return fmt.Sprintf("%s/%s", r.publicURL, key), nil
}

func (r *R2Storage) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := r.client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(r.bucket),
		Key:    aws.String(id.String()),
	})
	if err != nil {
		return fmt.Errorf("failed to delete from R2: %w", err)
	}
	return nil
}

// infrastructure/r2_storage.go
func (r *R2Storage) Replace(ctx context.Context, id uuid.UUID, file io.Reader, contentType string) (string, error) {
	_, err := r.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:       aws.String(r.bucket),
		Key:          aws.String(id.String()),
		Body:         file,
		ContentType:  aws.String(contentType),
		CacheControl: aws.String("public, max-age=2592000"),
	})
	if err != nil {
		return "", fmt.Errorf("failed to replace image: %w", err)
	}
	return fmt.Sprintf("%s/%s", r.publicURL, id.String()), nil
}

func (r *R2Storage) Download(ctx context.Context, id uuid.UUID) ([]byte, error) {
	result, err := r.client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(r.bucket),
		Key:    aws.String(id.String()),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to download from R2: %w", err)
	}
	defer result.Body.Close()
	return io.ReadAll(result.Body)
}
