// Package companydata — сторона СЕРВИСА-ВЛАДЕЛЬЦА в контракте
// auth.v1.CompanyDataService: администратор выгружает компанию файлом или
// переносит её в другую систему, а собирает архив authsvc.
//
// Владельцев четверо (задачи, реестры, календари, портал), транспорт у них
// одинаковый — поэтому здесь живёт общая часть, а сервису остаётся реализовать
// Owner двумя методами поверх своих таблиц.
package companydata

import (
	"context"

	"google.golang.org/grpc"

	"github.com/DmitriyODS/gw2/back-go/pkg/gen/authpb"
)

// Export — что владелец отдаёт в архив.
type Export struct {
	// Payload — JSON владельца. Для оркестратора он непрозрачен: устройство
	// своих таблиц знает только сам сервис.
	Payload []byte
	// FileKeys — ключи хранилища, на которые ссылается Payload. Объекты
	// копирует оркестратор — хранилище общее.
	FileKeys []string
	// Count — сколько записей внутри (для отчёта о переносе).
	Count int
}

// Import — что владелец получает при вливании.
type Import struct {
	CompanyID int64
	// ActorID — кто импортирует: им подменяются несопоставленные авторы.
	ActorID int64
	Payload []byte
	// Users — исходный user_id → id в этой системе. Кого не нашли, в карте нет.
	Users map[int64]int64
	// Files — исходный ключ хранилища → новый (объекты уже скопированы).
	Files map[string]string
}

// UserID — сопоставленный автор или тот, кто импортирует. Владельцы зовут его
// на каждой ссылке на человека, поэтому правило подстановки одно на всех.
func (in Import) UserID(orig int64) int64 {
	if id, ok := in.Users[orig]; ok && id > 0 {
		return id
	}
	return in.ActorID
}

// FileKey — новый ключ файла. Незнакомый ключ отдаётся как есть: объект,
// которого не было в архиве, всё равно уже недоступен, а ссылку сохраняем —
// по ней видно, что файл был.
func (in Import) FileKey(orig string) string {
	if key, ok := in.Files[orig]; ok && key != "" {
		return key
	}
	return orig
}

// Owner — сервис, хранящий контент компании.
type Owner interface {
	// ExportCompany — весь свой контент компании одним JSON.
	ExportCompany(ctx context.Context, companyID int64) (Export, error)
	// ImportCompany — влить контент в компанию, созданную под импорт.
	ImportCompany(ctx context.Context, in Import) (int, error)
}

// Register — поднять CompanyDataService поверх Owner на общем gRPC-сервере.
func Register(srv *grpc.Server, owner Owner) {
	authpb.RegisterCompanyDataServiceServer(srv, &server{owner: owner})
}

type server struct {
	authpb.UnimplementedCompanyDataServiceServer
	owner Owner
}

func (s *server) ExportCompany(ctx context.Context, in *authpb.ExportCompanyRequest) (*authpb.ExportCompanyResponse, error) {
	res, err := s.owner.ExportCompany(ctx, in.GetCompanyId())
	if err != nil {
		return nil, err
	}
	return &authpb.ExportCompanyResponse{
		Payload:  res.Payload,
		FileKeys: res.FileKeys,
		Count:    int32(res.Count),
	}, nil
}

func (s *server) ImportCompany(ctx context.Context, in *authpb.ImportCompanyRequest) (*authpb.ImportCompanyResponse, error) {
	users := make(map[int64]int64, len(in.GetUserMap()))
	for orig, id := range in.GetUserMap() {
		if n, err := parseID(orig); err == nil {
			users[n] = id
		}
	}
	count, err := s.owner.ImportCompany(ctx, Import{
		CompanyID: in.GetCompanyId(),
		ActorID:   in.GetActorId(),
		Payload:   in.GetPayload(),
		Users:     users,
		Files:     in.GetFileMap(),
	})
	if err != nil {
		return nil, err
	}
	return &authpb.ImportCompanyResponse{Count: int32(count)}, nil
}
