// Package storagefiles — сторона СЕРВИСА-ВЛАДЕЛЬЦА файлов в контракте
// billing.v1.FileOwnerService: биллинг спрашивает, какие файлы ещё живы, и
// просит удалить лишние (раздел «Настройки → Хранилище»).
//
// Владельцев семеро, а транспорт у них один и тот же, поэтому здесь и живёт
// общая часть: конвертация в protobuf и отсев пустых ключей. Сервису остаётся
// реализовать Owner — два метода поверх собственных таблиц.
package storagefiles

import (
	"context"
	"time"

	"google.golang.org/grpc"

	"github.com/DmitriyODS/gw2/back-go/pkg/gen/billingpb"
)

// File — файл глазами владельца. Размер сюда не входит: его знает журнал
// биллинга, а для незнакомых ключей он померит сам — обходить хранилище
// объект за объектом ради каждой сверки слишком дорого.
type File struct {
	Key string // ключ в хранилище, он же хвост адреса /uploads/<key>
	// Name — исходное имя файла, если владелец его хранит.
	Name string
	// Kind/ID — сущность-носитель: раздел «Хранилище» строит по ним переход.
	Kind string
	ID   string
	// Title — человекочитаемое «где именно»: «Заметка «Планы»», «Чат с Ивановым».
	Title     string
	CompanyID int64 // > 0 — файл компании
	CreatedAt time.Time
}

// Owner — сервис, хранящий пользовательские файлы.
//
// companyIDs — компании, чью квоту оплачивает этот пользователь (он их
// создал). Кто создатель, знает биллинг, поэтому владелец получает готовый
// список и сам про создателей не рассуждает; для сервисов с чисто личными
// файлами (заметки, доски, аватарки) список просто игнорируется.
type Owner interface {
	// ListStorageFiles — файлы, на которые ещё ссылается хоть одна живая
	// сущность. Всё, чего здесь нет, а в журнале есть, — сирота.
	ListStorageFiles(ctx context.Context, userID int64, companyIDs []int64) ([]File, error)
	// DeleteStorageFiles — снять файлы с их сущностей и удалить объекты.
	// Возвращает реально удалённые ключи; чужой ключ молча пропускается.
	DeleteStorageFiles(ctx context.Context, userID int64, companyIDs []int64, keys []string) ([]string, error)
}

// Register — поднять FileOwnerService поверх Owner на общем gRPC-сервере.
func Register(srv *grpc.Server, owner Owner) {
	billingpb.RegisterFileOwnerServiceServer(srv, &server{owner: owner})
}

type server struct {
	billingpb.UnimplementedFileOwnerServiceServer
	owner Owner
}

func (s *server) ListFiles(ctx context.Context, in *billingpb.ListFilesRequest) (*billingpb.ListFilesResponse, error) {
	files, err := s.owner.ListStorageFiles(ctx, in.GetUserId(), in.GetCompanyIds())
	if err != nil {
		return nil, err
	}
	out := make([]*billingpb.OwnedFile, 0, len(files))
	for _, f := range files {
		if f.Key == "" {
			continue
		}
		item := &billingpb.OwnedFile{
			Key: f.Key, Name: f.Name, RefKind: f.Kind, RefId: f.ID,
			Title: f.Title, CompanyId: f.CompanyID,
		}
		if !f.CreatedAt.IsZero() {
			item.CreatedAt = f.CreatedAt.Unix()
		}
		out = append(out, item)
	}
	return &billingpb.ListFilesResponse{Files: out}, nil
}

func (s *server) DeleteFiles(ctx context.Context, in *billingpb.DeleteFilesRequest) (*billingpb.DeleteFilesResponse, error) {
	if len(in.GetKeys()) == 0 {
		return &billingpb.DeleteFilesResponse{}, nil
	}
	deleted, err := s.owner.DeleteStorageFiles(ctx, in.GetUserId(), in.GetCompanyIds(), in.GetKeys())
	if err != nil {
		return nil, err
	}
	return &billingpb.DeleteFilesResponse{DeletedKeys: deleted}, nil
}
