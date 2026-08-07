// Package clients — исходящие gRPC-вызовы биллинга к сервисам-владельцам
// файлов (billing.v1.FileOwnerService).
package clients

import (
	"context"
	"log/slog"
	"sort"
	"strings"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/DmitriyODS/gw2/back-go/billing/internal/domain"
	"github.com/DmitriyODS/gw2/back-go/pkg/gen/billingpb"
)

// callTimeout — обход владельцев идёт по кнопке в настройках, но зависший
// сосед не должен держать запрос целиком.
const callTimeout = 10 * time.Second

// FileOwners — пул соединений с владельцами файлов. Адреса задаёт одна
// переменная окружения FILE_OWNER_ADDRS вида
// «messenger=messenger:9092,notes=notes:9103»: список владельцев подвижен, и
// плодить по переменной на каждого — лишний шум в compose.
//
// Коды сервисов ОБЯЗАНЫ совпадать с теми, под которыми файлы попадают в
// журнал (billing_storage_files.service): по ним раздел «Хранилище» связывает
// строку журнала с тем, кого о ней спрашивать.
type FileOwners struct {
	conns map[string]billingpb.FileOwnerServiceClient
	close []func() error
	log   *slog.Logger
}

func NewFileOwners(spec string, log *slog.Logger) (*FileOwners, error) {
	out := &FileOwners{conns: map[string]billingpb.FileOwnerServiceClient{}, log: log}
	for _, pair := range strings.Split(spec, ",") {
		pair = strings.TrimSpace(pair)
		if pair == "" {
			continue
		}
		service, addr, ok := strings.Cut(pair, "=")
		service, addr = strings.TrimSpace(service), strings.TrimSpace(addr)
		if !ok || service == "" || addr == "" {
			log.Warn("fileowners.bad_spec", "entry", pair)
			continue
		}
		conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
		if err != nil {
			out.Close()
			return nil, err
		}
		out.conns[service] = billingpb.NewFileOwnerServiceClient(conn)
		out.close = append(out.close, conn.Close)
	}
	log.Info("fileowners.configured", "services", out.Services())
	return out, nil
}

func (c *FileOwners) Close() {
	for _, closeConn := range c.close {
		_ = closeConn()
	}
}

func contextWithTimeout(ctx domain.Ctx) (context.Context, context.CancelFunc) {
	return context.WithTimeout(ctx, callTimeout)
}

func (c *FileOwners) Services() []string {
	out := make([]string, 0, len(c.conns))
	for service := range c.conns {
		out = append(out, service)
	}
	sort.Strings(out) // порядок обхода стабилен — удобнее читать логи
	return out
}

func (c *FileOwners) ListFiles(ctx domain.Ctx, service string, userID int64, companyIDs []int64) ([]*domain.OwnedFile, error) {
	client, ok := c.conns[service]
	if !ok {
		return nil, nil
	}
	rctx, cancel := contextWithTimeout(ctx)
	defer cancel()
	res, err := client.ListFiles(rctx, &billingpb.ListFilesRequest{
		UserId: userID, CompanyIds: companyIDs,
	})
	if err != nil {
		return nil, err
	}
	out := make([]*domain.OwnedFile, 0, len(res.GetFiles()))
	for _, f := range res.GetFiles() {
		file := &domain.OwnedFile{
			Key: f.GetKey(), Name: f.GetName(), RefKind: f.GetRefKind(),
			RefID: f.GetRefId(), RefTitle: f.GetTitle(), CompanyID: f.GetCompanyId(),
		}
		if ts := f.GetCreatedAt(); ts > 0 {
			file.CreatedAt = time.Unix(ts, 0)
		}
		out = append(out, file)
	}
	return out, nil
}

func (c *FileOwners) DeleteFiles(ctx domain.Ctx, service string, userID int64, companyIDs []int64, keys []string) ([]string, error) {
	client, ok := c.conns[service]
	if !ok {
		return nil, nil
	}
	rctx, cancel := contextWithTimeout(ctx)
	defer cancel()
	res, err := client.DeleteFiles(rctx, &billingpb.DeleteFilesRequest{
		UserId: userID, CompanyIds: companyIDs, Keys: keys,
	})
	if err != nil {
		return nil, err
	}
	return res.GetDeletedKeys(), nil
}
