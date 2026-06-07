package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
)

// blobStore is a tiny key/value abstraction over the two extra JSON objects the
// daily summary needs to read and write: the public summary file and the
// builder's private snapshot state. The existing stateWriter only ever writes a
// single fixed key, so the summary keeps its own store rather than overloading
// it. Get returns (nil, nil) when the key is absent — a fresh deployment hasn't
// written either object yet, and that's the normal first-run case.
type blobStore interface {
	Get(ctx context.Context, key string) ([]byte, error)
	Put(ctx context.Context, key string, data []byte) error
}

// ── S3 ──────────────────────────────────────────────────────────────────────

type s3Blob struct {
	client *s3.Client
	bucket string
}

func newS3Blob(region, bucket string) (*s3Blob, error) {
	cfg, err := config.LoadDefaultConfig(context.Background(), config.WithRegion(region))
	if err != nil {
		return nil, fmt.Errorf("load AWS config: %w", err)
	}
	return &s3Blob{client: s3.NewFromConfig(cfg), bucket: bucket}, nil
}

func (b *s3Blob) Get(ctx context.Context, key string) ([]byte, error) {
	out, err := b.client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(b.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		var nsk *types.NoSuchKey
		if errors.As(err, &nsk) {
			return nil, nil
		}
		return nil, fmt.Errorf("get s3://%s/%s: %w", b.bucket, key, err)
	}
	defer out.Body.Close()
	buf := new(bytes.Buffer)
	if _, err := buf.ReadFrom(out.Body); err != nil {
		return nil, fmt.Errorf("read s3://%s/%s: %w", b.bucket, key, err)
	}
	return buf.Bytes(), nil
}

func (b *s3Blob) Put(ctx context.Context, key string, data []byte) error {
	_, err := b.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:       aws.String(b.bucket),
		Key:          aws.String(key),
		Body:         bytes.NewReader(data),
		ContentType:  aws.String("application/json"),
		CacheControl: aws.String("no-cache"),
	})
	if err != nil {
		return fmt.Errorf("put s3://%s/%s: %w", b.bucket, key, err)
	}
	return nil
}

// ── Local filesystem ──────────────────────────────────────────────────────────

// fileBlob writes summary objects alongside the local state file so a local
// `go run .` produces the same set of data files the frontend expects.
type fileBlob struct {
	dir string
}

func newFileBlob(dir string) *fileBlob {
	return &fileBlob{dir: dir}
}

// path maps an S3-style key ("data/daily-summary.json") onto a flat filename in
// the output directory, so it sits next to state.json rather than in a nested
// data/ subdirectory the dev server wouldn't serve.
func (b *fileBlob) path(key string) string {
	return filepath.Join(b.dir, filepath.Base(key))
}

func (b *fileBlob) Get(_ context.Context, key string) ([]byte, error) {
	data, err := os.ReadFile(b.path(key))
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	return data, err
}

func (b *fileBlob) Put(_ context.Context, key string, data []byte) error {
	return os.WriteFile(b.path(key), data, 0o644)
}

// newBlobStore builds the store matching the writer mode: a local directory when
// LOCAL_OUTPUT is set, otherwise the same S3 bucket the state file goes to.
func newBlobStore(env Env, logger *slog.Logger) (blobStore, error) {
	if env.LocalOutput != "" {
		dir := filepath.Dir(env.LocalOutput)
		logger.Info("daily summary: writing to local dir", "dir", dir)
		return newFileBlob(dir), nil
	}
	return newS3Blob(env.Region, env.Bucket)
}
