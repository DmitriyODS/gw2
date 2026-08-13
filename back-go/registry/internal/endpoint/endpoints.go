// Package endpoint — go-kit обёртки use-case'ов registrysvc: единая сигнатура
// (ctx, request) → (response, error). Та же схема, что в остальных сервисах.
package endpoint

import (
	"context"

	"github.com/go-kit/kit/endpoint"

	"github.com/DmitriyODS/gw2/back-go/registry/internal/domain"
	"github.com/DmitriyODS/gw2/back-go/registry/internal/service"
)

type Endpoints struct {
	ListRegistries endpoint.Endpoint
	GetRegistry    endpoint.Endpoint
	CreateRegistry endpoint.Endpoint
	UpdateRegistry endpoint.Endpoint
	DeleteRegistry endpoint.Endpoint
	ReplaceFields  endpoint.Endpoint

	ListRecords   endpoint.Endpoint
	SearchRecords endpoint.Endpoint
	GetRecord     endpoint.Endpoint
	CreateRecord  endpoint.Endpoint
	UpdateRecord  endpoint.Endpoint
	DeleteRecord  endpoint.Endpoint
	DeleteRecords endpoint.Endpoint
	ExportRecords endpoint.Endpoint

	Upload endpoint.Endpoint

	// Публичные ссылки.
	ListShares  endpoint.Endpoint
	CreateShare endpoint.Endpoint
	RevokeShare endpoint.Endpoint

	SharedRegistry endpoint.Endpoint
	SharedRecords  endpoint.Endpoint
	SharedExport   endpoint.Endpoint

	// Ссылка уровня edit: правка записей без авторизации.
	SharedCreateRecord endpoint.Endpoint
	SharedUpdateRecord endpoint.Endpoint
	SharedUpload       endpoint.Endpoint
}

// ── Request-типы ──

type CompanyReq struct{ CompanyID int64 }

type RegistryReq struct {
	CompanyID int64
	ID        int64
}

type CreateRegistryReq struct {
	CompanyID int64
	UserID    int64
	Name      string
}

type UpdateRegistryReq struct {
	CompanyID int64
	ID        int64
	Name      string
	// TagFieldID — поле-источник тегов (nil выключает их); учитывается только
	// вместе с TagFieldSet.
	TagFieldID  *int64
	TagFieldSet bool
}

type ReplaceFieldsReq struct {
	CompanyID int64
	ID        int64
	Fields    []domain.Field
}

// SearchRecordsReq — глобальный поиск рабочего стола по записям компании.
type SearchRecordsReq struct {
	CompanyID int64
	Query     string
	Limit     int
}

type ListRecordsReq struct {
	CompanyID  int64
	RegistryID int64
	Params     service.RecordListParams
}

type RecordReq struct {
	CompanyID  int64
	RegistryID int64
	RecordID   int64
}

type WriteRecordReq struct {
	CompanyID  int64
	RegistryID int64
	UserID     int64
	RecordID   int64
	Data       map[string]any
}

type DeleteRecordsReq struct {
	CompanyID  int64
	RegistryID int64
	Params     service.BulkParams
}

type ExportReq struct {
	CompanyID  int64
	RegistryID int64
	Params     service.ExportParams
}

type ExportResp struct {
	Data []byte
	Name string
}

type UploadReq struct {
	CompanyID int64
	UserID    int64
	FileName  string
	Mime      string
	Data      []byte
}

type ShareReq struct {
	CompanyID  int64
	RegistryID int64
	UserID     int64
	ShareID    int64
	Access     string // view | edit (только при создании)
}

type SharedRecordsReq struct {
	Code   string
	Params service.RecordListParams
}

type SharedExportReq struct {
	Code   string
	Params service.ExportParams
}

type SharedWriteReq struct {
	Code     string
	RecordID int64
	Data     map[string]any
}

type SharedUploadReq struct {
	Code     string
	FileName string
	Mime     string
	Data     []byte
}

func New(s *service.Service) Endpoints {
	return Endpoints{
		ListRegistries: func(ctx context.Context, request any) (any, error) {
			r := request.(CompanyReq)
			return s.ListRegistries(ctx, r.CompanyID)
		},
		GetRegistry: func(ctx context.Context, request any) (any, error) {
			r := request.(RegistryReq)
			return s.GetRegistry(ctx, r.CompanyID, r.ID)
		},
		CreateRegistry: func(ctx context.Context, request any) (any, error) {
			r := request.(CreateRegistryReq)
			return s.CreateRegistry(ctx, r.CompanyID, r.UserID, r.Name)
		},
		UpdateRegistry: func(ctx context.Context, request any) (any, error) {
			r := request.(UpdateRegistryReq)
			return s.UpdateRegistry(ctx, r.CompanyID, r.ID, service.RegistryPatch{
				Name: r.Name, TagFieldID: r.TagFieldID, TagFieldSet: r.TagFieldSet,
			})
		},
		DeleteRegistry: func(ctx context.Context, request any) (any, error) {
			r := request.(RegistryReq)
			return nil, s.DeleteRegistry(ctx, r.CompanyID, r.ID)
		},
		ReplaceFields: func(ctx context.Context, request any) (any, error) {
			r := request.(ReplaceFieldsReq)
			return s.ReplaceFields(ctx, r.CompanyID, r.ID, r.Fields)
		},
		SearchRecords: func(ctx context.Context, request any) (any, error) {
			r := request.(SearchRecordsReq)
			return s.SearchRecords(ctx, r.CompanyID, r.Query, r.Limit)
		},
		ListRecords: func(ctx context.Context, request any) (any, error) {
			r := request.(ListRecordsReq)
			return s.ListRecords(ctx, r.CompanyID, r.RegistryID, r.Params)
		},
		GetRecord: func(ctx context.Context, request any) (any, error) {
			r := request.(RecordReq)
			return s.GetRecord(ctx, r.CompanyID, r.RegistryID, r.RecordID)
		},
		CreateRecord: func(ctx context.Context, request any) (any, error) {
			r := request.(WriteRecordReq)
			return s.CreateRecord(ctx, r.CompanyID, r.RegistryID, r.UserID, r.Data)
		},
		UpdateRecord: func(ctx context.Context, request any) (any, error) {
			r := request.(WriteRecordReq)
			return s.UpdateRecord(ctx, r.CompanyID, r.RegistryID, r.RecordID, r.Data)
		},
		DeleteRecord: func(ctx context.Context, request any) (any, error) {
			r := request.(RecordReq)
			return nil, s.DeleteRecord(ctx, r.CompanyID, r.RegistryID, r.RecordID)
		},
		DeleteRecords: func(ctx context.Context, request any) (any, error) {
			r := request.(DeleteRecordsReq)
			return s.DeleteRecords(ctx, r.CompanyID, r.RegistryID, r.Params)
		},
		ExportRecords: func(ctx context.Context, request any) (any, error) {
			r := request.(ExportReq)
			data, name, err := s.ExportRecords(ctx, r.CompanyID, r.RegistryID, r.Params)
			if err != nil {
				return nil, err
			}
			return ExportResp{Data: data, Name: name}, nil
		},
		Upload: func(ctx context.Context, request any) (any, error) {
			r := request.(UploadReq)
			return s.SaveUpload(ctx, r.CompanyID, r.UserID, r.FileName, r.Mime, r.Data)
		},
		ListShares: func(ctx context.Context, request any) (any, error) {
			r := request.(ShareReq)
			return s.ListShares(ctx, r.CompanyID, r.RegistryID)
		},
		CreateShare: func(ctx context.Context, request any) (any, error) {
			r := request.(ShareReq)
			return s.CreateShare(ctx, r.CompanyID, r.RegistryID, r.UserID, r.Access)
		},
		RevokeShare: func(ctx context.Context, request any) (any, error) {
			r := request.(ShareReq)
			return nil, s.RevokeShare(ctx, r.CompanyID, r.RegistryID, r.ShareID)
		},
		SharedRegistry: func(ctx context.Context, request any) (any, error) {
			return s.SharedRegistry(ctx, request.(string))
		},
		SharedRecords: func(ctx context.Context, request any) (any, error) {
			r := request.(SharedRecordsReq)
			return s.SharedRecords(ctx, r.Code, r.Params)
		},
		SharedExport: func(ctx context.Context, request any) (any, error) {
			r := request.(SharedExportReq)
			data, name, err := s.SharedExport(ctx, r.Code, r.Params)
			if err != nil {
				return nil, err
			}
			return ExportResp{Data: data, Name: name}, nil
		},
		SharedCreateRecord: func(ctx context.Context, request any) (any, error) {
			r := request.(SharedWriteReq)
			return s.SharedCreateRecord(ctx, r.Code, r.Data)
		},
		SharedUpdateRecord: func(ctx context.Context, request any) (any, error) {
			r := request.(SharedWriteReq)
			return s.SharedUpdateRecord(ctx, r.Code, r.RecordID, r.Data)
		},
		SharedUpload: func(ctx context.Context, request any) (any, error) {
			r := request.(SharedUploadReq)
			return s.SharedUpload(ctx, r.Code, r.FileName, r.Mime, r.Data)
		},
	}
}
