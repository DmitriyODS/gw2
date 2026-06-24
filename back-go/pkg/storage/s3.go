package storage

import (
	"bytes"
	"context"
	"io"
	"log/slog"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
)

// S3Config — параметры подключения к S3-совместимому хранилищу (Beget:
// endpoint s3.ru1.storage.beget.cloud, регион ru1, обязателен path-style).
type S3Config struct {
	Endpoint  string
	Region    string
	Bucket    string
	AccessKey string
	SecretKey string
	Secure    bool
}

type s3Store struct {
	client *s3.Client
	bucket string
	log    *slog.Logger
}

func NewS3(cfg S3Config, log *slog.Logger) (Storage, error) {
	scheme := "https://"
	if !cfg.Secure {
		scheme = "http://"
	}
	client := s3.New(s3.Options{
		Region:       cfg.Region,
		BaseEndpoint: aws.String(scheme + cfg.Endpoint),
		UsePathStyle: true, // Beget требует path-style
		Credentials:  credentials.NewStaticCredentialsProvider(cfg.AccessKey, cfg.SecretKey, ""),
	})
	return &s3Store{client: client, bucket: cfg.Bucket, log: log}, nil
}

// Put — объект помечается public-read: ключ хранилища Beget не имеет прав на
// политику бакета, поэтому анонимная отдача (nginx /uploads/) обеспечивается
// ACL на уровне объекта (рекомендованный Beget способ).
func (s *s3Store) Put(ctx context.Context, key string, data []byte, contentType string) error {
	if contentType == "" {
		contentType = "application/octet-stream"
	}
	_, err := s.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(s.bucket),
		Key:         aws.String(key),
		Body:        bytes.NewReader(data),
		ContentType: aws.String(contentType),
		ACL:         types.ObjectCannedACLPublicRead,
	})
	return err
}

func (s *s3Store) Open(ctx context.Context, key string) (io.ReadCloser, error) {
	out, err := s.client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return nil, err
	}
	return out.Body, nil
}

func (s *s3Store) Copy(ctx context.Context, srcKey, dstKey string) error {
	_, err := s.client.CopyObject(ctx, &s3.CopyObjectInput{
		Bucket:     aws.String(s.bucket),
		Key:        aws.String(dstKey),
		CopySource: aws.String(s.bucket + "/" + srcKey),
		ACL:        types.ObjectCannedACLPublicRead,
	})
	return err
}

func (s *s3Store) Remove(ctx context.Context, keys ...string) {
	for _, k := range keys {
		if k == "" {
			continue
		}
		if _, err := s.client.DeleteObject(ctx, &s3.DeleteObjectInput{
			Bucket: aws.String(s.bucket),
			Key:    aws.String(k),
		}); err != nil {
			s.log.Warn("storage.remove_failed", "key", k, "error", err)
		}
	}
}

func (s *s3Store) List(ctx context.Context, prefix string) ([]string, error) {
	var keys []string
	p := s3.NewListObjectsV2Paginator(s.client, &s3.ListObjectsV2Input{
		Bucket: aws.String(s.bucket),
		Prefix: aws.String(prefix),
	})
	for p.HasMorePages() {
		page, err := p.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		for _, o := range page.Contents {
			keys = append(keys, aws.ToString(o.Key))
		}
	}
	return keys, nil
}
