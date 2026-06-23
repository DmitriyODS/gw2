// Package endpoint — go-kit обёртки use-case'ов calendarsvc: единая сигнатура
// (ctx, request) → (response, error). Та же схема, что в остальных сервисах.
package endpoint

import (
	"context"
	"time"

	"github.com/go-kit/kit/endpoint"

	"github.com/DmitriyODS/gw2/back-go/calendar/internal/domain"
	"github.com/DmitriyODS/gw2/back-go/calendar/internal/service"
)

type Endpoints struct {
	ListCalendars  endpoint.Endpoint
	GetCalendar    endpoint.Endpoint
	CreateCalendar endpoint.Endpoint
	UpdateCalendar endpoint.Endpoint
	DeleteCalendar endpoint.Endpoint
	ReplaceFields  endpoint.Endpoint

	ListEntries   endpoint.Endpoint
	GetEntry      endpoint.Endpoint
	CreateEntry   endpoint.Endpoint
	UpdateEntry   endpoint.Endpoint
	DeleteEntry   endpoint.Endpoint
	DeleteEntries endpoint.Endpoint
	ExportEntries endpoint.Endpoint

	Upload endpoint.Endpoint

	// Публичные ссылки.
	ListShares  endpoint.Endpoint
	CreateShare endpoint.Endpoint
	RevokeShare endpoint.Endpoint

	SharedCalendar endpoint.Endpoint
	SharedEntries  endpoint.Endpoint
	SharedExport   endpoint.Endpoint
}

// ── Request-типы ──

type CompanyReq struct{ CompanyID int64 }

type CalendarReq struct {
	CompanyID int64
	ID        int64
}

type CreateCalendarReq struct {
	CompanyID int64
	UserID    int64
	Name      string
}

type UpdateCalendarReq struct {
	CompanyID int64
	ID        int64
	Name      string
}

type ReplaceFieldsReq struct {
	CompanyID int64
	ID        int64
	Fields    []domain.Field
}

type ListEntriesReq struct {
	CompanyID  int64
	CalendarID int64
	Params     service.EntryListParams
}

type EntryReq struct {
	CompanyID  int64
	CalendarID int64
	EntryID    int64
}

type WriteEntryReq struct {
	CompanyID  int64
	CalendarID int64
	UserID     int64
	EntryID    int64
	EventAt    time.Time
	Data       map[string]any
}

type DeleteEntriesReq struct {
	CompanyID  int64
	CalendarID int64
	IDs        []int64
}

type ExportReq struct {
	CompanyID  int64
	CalendarID int64
	FieldIDs   []int64
	Params     service.EntryListParams
	IDs        []int64
}

type ExportResp struct {
	Data []byte
	Name string
}

type UploadReq struct {
	FileName string
	Mime     string
	Data     []byte
}

type ShareReq struct {
	CompanyID  int64
	CalendarID int64
	UserID     int64
	ShareID    int64
}

type SharedEntriesReq struct {
	Code   string
	Params service.EntryListParams
}

type SharedExportReq struct {
	Code     string
	FieldIDs []int64
	Params   service.EntryListParams
	IDs      []int64
}

func New(s *service.Service) Endpoints {
	return Endpoints{
		ListCalendars: func(ctx context.Context, request any) (any, error) {
			r := request.(CompanyReq)
			return s.ListCalendars(ctx, r.CompanyID)
		},
		GetCalendar: func(ctx context.Context, request any) (any, error) {
			r := request.(CalendarReq)
			return s.GetCalendar(ctx, r.CompanyID, r.ID)
		},
		CreateCalendar: func(ctx context.Context, request any) (any, error) {
			r := request.(CreateCalendarReq)
			return s.CreateCalendar(ctx, r.CompanyID, r.UserID, r.Name)
		},
		UpdateCalendar: func(ctx context.Context, request any) (any, error) {
			r := request.(UpdateCalendarReq)
			return s.UpdateCalendar(ctx, r.CompanyID, r.ID, r.Name)
		},
		DeleteCalendar: func(ctx context.Context, request any) (any, error) {
			r := request.(CalendarReq)
			return nil, s.DeleteCalendar(ctx, r.CompanyID, r.ID)
		},
		ReplaceFields: func(ctx context.Context, request any) (any, error) {
			r := request.(ReplaceFieldsReq)
			return s.ReplaceFields(ctx, r.CompanyID, r.ID, r.Fields)
		},
		ListEntries: func(ctx context.Context, request any) (any, error) {
			r := request.(ListEntriesReq)
			return s.ListEntries(ctx, r.CompanyID, r.CalendarID, r.Params)
		},
		GetEntry: func(ctx context.Context, request any) (any, error) {
			r := request.(EntryReq)
			return s.GetEntry(ctx, r.CompanyID, r.CalendarID, r.EntryID)
		},
		CreateEntry: func(ctx context.Context, request any) (any, error) {
			r := request.(WriteEntryReq)
			return s.CreateEntry(ctx, r.CompanyID, r.CalendarID, r.UserID, r.EventAt, r.Data)
		},
		UpdateEntry: func(ctx context.Context, request any) (any, error) {
			r := request.(WriteEntryReq)
			return s.UpdateEntry(ctx, r.CompanyID, r.CalendarID, r.EntryID, r.EventAt, r.Data)
		},
		DeleteEntry: func(ctx context.Context, request any) (any, error) {
			r := request.(EntryReq)
			return nil, s.DeleteEntry(ctx, r.CompanyID, r.CalendarID, r.EntryID)
		},
		DeleteEntries: func(ctx context.Context, request any) (any, error) {
			r := request.(DeleteEntriesReq)
			return s.DeleteEntries(ctx, r.CompanyID, r.CalendarID, r.IDs)
		},
		ExportEntries: func(ctx context.Context, request any) (any, error) {
			r := request.(ExportReq)
			data, name, err := s.ExportEntries(ctx, r.CompanyID, r.CalendarID, r.FieldIDs, r.Params, r.IDs)
			if err != nil {
				return nil, err
			}
			return ExportResp{Data: data, Name: name}, nil
		},
		Upload: func(ctx context.Context, request any) (any, error) {
			r := request.(UploadReq)
			return s.SaveUpload(r.FileName, r.Mime, r.Data)
		},
		ListShares: func(ctx context.Context, request any) (any, error) {
			r := request.(ShareReq)
			return s.ListShares(ctx, r.CompanyID, r.CalendarID)
		},
		CreateShare: func(ctx context.Context, request any) (any, error) {
			r := request.(ShareReq)
			return s.CreateShare(ctx, r.CompanyID, r.CalendarID, r.UserID)
		},
		RevokeShare: func(ctx context.Context, request any) (any, error) {
			r := request.(ShareReq)
			return nil, s.RevokeShare(ctx, r.CompanyID, r.CalendarID, r.ShareID)
		},
		SharedCalendar: func(ctx context.Context, request any) (any, error) {
			return s.SharedCalendar(ctx, request.(string))
		},
		SharedEntries: func(ctx context.Context, request any) (any, error) {
			r := request.(SharedEntriesReq)
			return s.SharedEntries(ctx, r.Code, r.Params)
		},
		SharedExport: func(ctx context.Context, request any) (any, error) {
			r := request.(SharedExportReq)
			data, name, err := s.SharedExport(ctx, r.Code, r.FieldIDs, r.Params, r.IDs)
			if err != nil {
				return nil, err
			}
			return ExportResp{Data: data, Name: name}, nil
		},
	}
}
