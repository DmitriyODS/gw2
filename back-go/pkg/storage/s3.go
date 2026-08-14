package storage

import (
	"bytes"
	"context"
	"fmt"
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
//
// Запись ПРОВЕРЯЕТСЯ сразу же: у хранилища бывает «успешный» ответ, после
// которого объекта в бакете нет, — и тогда карточка записи навсегда ссылается
// на пустоту, а человек узнаёт об этом через неделю, открыв её. Лучше честно
// отказать в загрузке: файл ещё у человека в руках, повторить ничего не стоит.
func (s *s3Store) Put(ctx context.Context, key string, data []byte, contentType string) error {
	if contentType == "" {
		contentType = "application/octet-stream"
	}
	if _, err := s.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(s.bucket),
		Key:         aws.String(key),
		Body:        bytes.NewReader(data),
		ContentType: aws.String(contentType),
		ACL:         types.ObjectCannedACLPublicRead,
	}); err != nil {
		return err
	}

	size, err := s.Size(ctx, key)
	if err != nil {
		s.log.Error("storage.put_unverified", "key", key, "error", err)
		return fmt.Errorf("объект %s не подтверждён хранилищем: %w", key, err)
	}
	if size != int64(len(data)) {
		s.log.Error("storage.put_size_mismatch", "key", key, "want", len(data), "got", size)
		return fmt.Errorf("объект %s записан не полностью (%d из %d байт)", key, size, len(data))
	}
	return nil
}

// s3PartSize — размер части multipart-загрузки. Столько же памяти держит
// PutStream: части читаются по одной, а не весь объект целиком.
const s3PartSize = 8 << 20

// PutStream — потоковая запись через multipart upload. Гигабайтный файл нельзя
// собрать в []byte, поэтому поток режется на части по s3PartSize и уходит
// частями; объект короче одной части отправляется обычным Put.
//
// Незавершённая загрузка ОБРЫВАЕТСЯ явно (AbortMultipartUpload): брошенные
// части остаются в бакете и занимают место, за которое платят.
func (s *s3Store) PutStream(ctx context.Context, key string, r io.Reader, size int64, contentType string) error {
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	first := make([]byte, s3PartSize)
	n, err := io.ReadFull(r, first)
	if err != nil && err != io.ErrUnexpectedEOF && err != io.EOF {
		return err
	}
	if n < s3PartSize {
		return s.Put(ctx, key, first[:n], contentType)
	}

	created, err := s.client.CreateMultipartUpload(ctx, &s3.CreateMultipartUploadInput{
		Bucket:      aws.String(s.bucket),
		Key:         aws.String(key),
		ContentType: aws.String(contentType),
		ACL:         types.ObjectCannedACLPublicRead,
	})
	if err != nil {
		return err
	}
	uploadID := created.UploadId

	abort := func(cause error) error {
		if _, aerr := s.client.AbortMultipartUpload(ctx, &s3.AbortMultipartUploadInput{
			Bucket: aws.String(s.bucket), Key: aws.String(key), UploadId: uploadID,
		}); aerr != nil {
			s.log.Warn("storage.multipart_abort_failed", "key", key, "error", aerr)
		}
		return cause
	}

	var (
		parts   []types.CompletedPart
		written int64
		part    = first[:n]
	)
	for num := int32(1); ; num++ {
		out, err := s.client.UploadPart(ctx, &s3.UploadPartInput{
			Bucket:     aws.String(s.bucket),
			Key:        aws.String(key),
			UploadId:   uploadID,
			PartNumber: aws.Int32(num),
			Body:       bytes.NewReader(part),
		})
		if err != nil {
			return abort(err)
		}
		parts = append(parts, types.CompletedPart{ETag: out.ETag, PartNumber: aws.Int32(num)})
		written += int64(len(part))

		buf := make([]byte, s3PartSize)
		m, err := io.ReadFull(r, buf)
		if m > 0 {
			part = buf[:m]
			continue
		}
		if err == nil || err == io.EOF || err == io.ErrUnexpectedEOF {
			break
		}
		return abort(err)
	}

	if _, err := s.client.CompleteMultipartUpload(ctx, &s3.CompleteMultipartUploadInput{
		Bucket:          aws.String(s.bucket),
		Key:             aws.String(key),
		UploadId:        uploadID,
		MultipartUpload: &types.CompletedMultipartUpload{Parts: parts},
	}); err != nil {
		return abort(err)
	}

	// Та же сверка, что и у Put: хранилище отвечало «успех», не сохранив
	// объект, и запись навсегда ссылалась на пустоту.
	stored, err := s.Size(ctx, key)
	if err != nil {
		s.log.Error("storage.put_unverified", "key", key, "error", err)
		return fmt.Errorf("объект %s не подтверждён хранилищем: %w", key, err)
	}
	want := written
	if size > 0 {
		want = size
	}
	if stored != want {
		s.log.Error("storage.put_size_mismatch", "key", key, "want", want, "got", stored)
		return fmt.Errorf("объект %s записан не полностью (%d из %d байт)", key, stored, want)
	}
	return nil
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

// Size — размер объекта (HeadObject: тело не качаем).
func (s *s3Store) Size(ctx context.Context, key string) (int64, error) {
	out, err := s.client.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(s.bucket), Key: aws.String(key),
	})
	if err != nil {
		return 0, err
	}
	if out.ContentLength == nil {
		return 0, nil
	}
	return *out.ContentLength, nil
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
