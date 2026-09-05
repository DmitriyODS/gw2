// Команда s3acl — проверка и починка анонимного доступа к объектам бакета.
//
// Файлы платформы раздаёт nginx проксированием на бакет, а анонимное чтение
// держится на ACL public-read КАЖДОГО объекта: ключ Beget не имеет прав на
// политику бакета. Объект, оставшийся приватным, отвечает браузеру 403 — и
// карточка записи навсегда показывает пустое место, хотя файл на месте.
//
// Проверка (ничего не меняет):
//
//	S3_ENDPOINT=… S3_BUCKET=… S3_ACCESS_KEY=… S3_SECRET_KEY=… \
//	go run ./cmd/s3acl -prefix registry/
//
// Починка найденного:
//
//	… go run ./cmd/s3acl -prefix registry/ -fix
//
// Отдельные ключи (например, из консоли браузера) — без обхода бакета:
//
//	… go run ./cmd/s3acl -keys registry/abc.jpg,registry/def.jpg
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/aws/smithy-go"

	"github.com/DmitriyODS/gw2/back-go/pkg/bootstrap"
)

// allUsers — группа «кто угодно» в ACL S3: чтение объекта анонимом даёт именно
// её grant с правом READ.
const allUsers = "http://acs.amazonaws.com/groups/global/AllUsers"

func main() {
	prefix := flag.String("prefix", "", "префикс ключей (пусто — весь бакет)")
	keys := flag.String("keys", "", "проверить только эти ключи через запятую")
	fix := flag.Bool("fix", false, "проставить public-read найденным приватным объектам")
	flag.Parse()

	log := bootstrap.Logger()
	client := s3.New(s3.Options{
		Region:       bootstrap.Env("S3_REGION", "ru1"),
		BaseEndpoint: aws.String(endpoint()),
		UsePathStyle: true, // Beget требует path-style
		Credentials: credentials.NewStaticCredentialsProvider(
			bootstrap.MustEnv(log, "S3_ACCESS_KEY"), bootstrap.MustEnv(log, "S3_SECRET_KEY"), ""),
	})
	bucket := bootstrap.MustEnv(log, "S3_BUCKET")
	ctx := context.Background()

	list, err := collect(ctx, client, bucket, *prefix, *keys)
	if err != nil {
		log.Error("s3acl.list", "error", err)
		os.Exit(1)
	}

	var missing, private, fixed, failed int
	for _, key := range list {
		public, err := isPublic(ctx, client, bucket, key)
		switch {
		case isNotFound(err):
			missing++
			fmt.Printf("НЕТ ОБЪЕКТА  %s\n", key)
			continue
		case err != nil:
			failed++
			fmt.Printf("ОШИБКА       %s: %v\n", key, err)
			continue
		case public:
			continue
		}
		private++
		if !*fix {
			fmt.Printf("ПРИВАТНЫЙ    %s\n", key)
			continue
		}
		if _, err := client.PutObjectAcl(ctx, &s3.PutObjectAclInput{
			Bucket: aws.String(bucket), Key: aws.String(key),
			ACL: types.ObjectCannedACLPublicRead,
		}); err != nil {
			failed++
			fmt.Printf("НЕ ИСПРАВЛЕН %s: %v\n", key, err)
			continue
		}
		fixed++
	}

	log.Info("s3acl.done", "проверено", len(list), "приватных", private,
		"исправлено", fixed, "нет объекта", missing, "ошибок", failed)
}

func endpoint() string {
	scheme := "https://"
	if bootstrap.Env("S3_SECURE", "true") == "false" {
		scheme = "http://"
	}
	return scheme + os.Getenv("S3_ENDPOINT")
}

// collect — что проверяем: перечисленные ключи либо весь префикс бакета.
func collect(ctx context.Context, client *s3.Client, bucket, prefix, keys string) ([]string, error) {
	if keys != "" {
		var out []string
		for _, k := range strings.Split(keys, ",") {
			if k = strings.TrimSpace(k); k != "" {
				out = append(out, k)
			}
		}
		return out, nil
	}
	var out []string
	p := s3.NewListObjectsV2Paginator(client, &s3.ListObjectsV2Input{
		Bucket: aws.String(bucket), Prefix: aws.String(prefix),
	})
	for p.HasMorePages() {
		page, err := p.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		for _, o := range page.Contents {
			out = append(out, aws.ToString(o.Key))
		}
	}
	return out, nil
}

// isPublic — есть ли у объекта grant «AllUsers: READ». Заодно отвечает на
// вопрос «существует ли он»: у пропавшего объекта ACL не спросить.
func isPublic(ctx context.Context, client *s3.Client, bucket, key string) (bool, error) {
	acl, err := client.GetObjectAcl(ctx, &s3.GetObjectAclInput{
		Bucket: aws.String(bucket), Key: aws.String(key),
	})
	if err != nil {
		return false, err
	}
	for _, g := range acl.Grants {
		if g.Grantee == nil || aws.ToString(g.Grantee.URI) != allUsers {
			continue
		}
		if g.Permission == types.PermissionRead || g.Permission == types.PermissionFullControl {
			return true, nil
		}
	}
	return false, nil
}

// isNotFound — объекта в бакете нет. Отличать это от «приватный» важно:
// анониму хранилище в обоих случаях отвечает одинаковым 403, и по ответу
// браузера причину не назвать.
func isNotFound(err error) bool {
	var nsk *types.NoSuchKey
	if errors.As(err, &nsk) {
		return true
	}
	var api smithy.APIError
	if errors.As(err, &api) {
		code := api.ErrorCode()
		return code == "NoSuchKey" || code == "NotFound"
	}
	return false
}
