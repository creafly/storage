package minio

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"path"
	"time"

	"github.com/creafly/logger"
	"github.com/creafly/storage/internal/config"
	"github.com/google/uuid"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

type Client struct {
	client     *minio.Client
	bucketName string
	endpoint   string
	useSSL     bool
}

func NewClient(cfg config.MinIOConfig) (*Client, error) {
	client, err := minio.New(cfg.Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.AccessKey, cfg.SecretKey, ""),
		Secure: cfg.UseSSL,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create minio client: %w", err)
	}

	return &Client{
		client:     client,
		bucketName: cfg.BucketName,
		endpoint:   cfg.Endpoint,
		useSSL:     cfg.UseSSL,
	}, nil
}

func (c *Client) EnsureBucket(ctx context.Context) error {
	exists, err := c.client.BucketExists(ctx, c.bucketName)
	if err != nil {
		return fmt.Errorf("failed to check bucket existence: %w", err)
	}

	if !exists {
		err = c.client.MakeBucket(ctx, c.bucketName, minio.MakeBucketOptions{})
		if err != nil {
			return fmt.Errorf("failed to create bucket: %w", err)
		}
		logger.Info().Str("bucket", c.bucketName).Msg("Created bucket")
	}

	return nil
}

func (c *Client) Upload(ctx context.Context, tenantID uuid.UUID, fileName string, contentType string, data []byte) (string, error) {
	objectPath := path.Join(tenantID.String(), fileName)

	reader := bytes.NewReader(data)
	_, err := c.client.PutObject(ctx, c.bucketName, objectPath, reader, int64(len(data)), minio.PutObjectOptions{
		ContentType: contentType,
	})
	if err != nil {
		return "", fmt.Errorf("failed to upload file: %w", err)
	}

	return objectPath, nil
}

func (c *Client) Download(ctx context.Context, objectPath string) ([]byte, error) {
	obj, err := c.client.GetObject(ctx, c.bucketName, objectPath, minio.GetObjectOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get object: %w", err)
	}
	defer func() { _ = obj.Close() }()

	data, err := io.ReadAll(obj)
	if err != nil {
		return nil, fmt.Errorf("failed to read object: %w", err)
	}

	return data, nil
}

func (c *Client) Delete(ctx context.Context, objectPath string) error {
	err := c.client.RemoveObject(ctx, c.bucketName, objectPath, minio.RemoveObjectOptions{})
	if err != nil {
		return fmt.Errorf("failed to delete object: %w", err)
	}
	return nil
}

func (c *Client) GetPresignedURL(ctx context.Context, objectPath string, expiry time.Duration) (string, error) {
	url, err := c.client.PresignedGetObject(ctx, c.bucketName, objectPath, expiry, nil)
	if err != nil {
		return "", fmt.Errorf("failed to generate presigned URL: %w", err)
	}
	return url.String(), nil
}

func (c *Client) GetPublicURL(objectPath string) string {
	protocol := "http"
	if c.useSSL {
		protocol = "https"
	}
	return fmt.Sprintf("%s://%s/%s/%s", protocol, c.endpoint, c.bucketName, objectPath)
}
