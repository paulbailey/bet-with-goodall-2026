package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

type stateWriter interface {
	Write(ctx context.Context, state StateJSON) error
}

// ── S3 writer ─────────────────────────────────────────────────────────────────

type s3Uploader struct {
	client *s3.Client
	bucket string
	key    string
	logger *slog.Logger
}

func newS3Uploader(region, bucket, key string, logger *slog.Logger) (*s3Uploader, error) {
	cfg, err := config.LoadDefaultConfig(context.Background(), config.WithRegion(region))
	if err != nil {
		return nil, fmt.Errorf("load AWS config: %w", err)
	}
	return &s3Uploader{
		client: s3.NewFromConfig(cfg),
		bucket: bucket,
		key:    key,
		logger: logger,
	}, nil
}

func (u *s3Uploader) Write(ctx context.Context, state StateJSON) error {
	data, err := json.Marshal(state)
	if err != nil {
		return fmt.Errorf("marshal state: %w", err)
	}
	_, err = u.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:       aws.String(u.bucket),
		Key:          aws.String(u.key),
		Body:         bytes.NewReader(data),
		ContentType:  aws.String("application/json"),
		CacheControl: aws.String("no-cache"),
	})
	if err != nil {
		return fmt.Errorf("put s3://%s/%s: %w", u.bucket, u.key, err)
	}
	u.logger.Info("uploaded to S3", "bucket", u.bucket, "key", u.key, "bytes", len(data))
	return nil
}

// ── Local file writer ──────────────────────────────────────────────────────────

type fileWriter struct {
	path   string
	logger *slog.Logger
}

func newFileWriter(path string, logger *slog.Logger) *fileWriter {
	return &fileWriter{path: path, logger: logger}
}

func (w *fileWriter) Write(_ context.Context, state StateJSON) error {
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal state: %w", err)
	}
	if err := os.WriteFile(w.path, data, 0o644); err != nil {
		return fmt.Errorf("write %s: %w", w.path, err)
	}
	w.logger.Info("wrote state file", "path", w.path, "bytes", len(data))
	return nil
}
