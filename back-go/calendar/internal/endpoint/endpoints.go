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
	MoveCalendar   endpoint.Endpoint
	DeleteCalendar endpoint.Endpoint
	ReplaceFields  endpoint.Endpoint

	ListEntries   endpoint.Endpoint
	Agenda        endpoint.Endpoint
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
// UserID — кто действует: доступ считается по нему, а не по активной компании.

type UserReq struct{ UserID int64 }

type CalendarReq struct {
	UserID int64
	ID     int64
}

type CreateCalendarReq struct {
	UserID int64
	// CompanyID — пространство: nil — личное.
	CompanyID *int64
	Name      string
}

type UpdateCalendarReq struct {
	UserID int64
	ID     int64
	Name   string
}

type MoveCalendarReq struct {
	UserID     int64
	ID         int64
	CompanyID  *int64
	TeamAccess string
}

type ReplaceFieldsReq struct {
	UserID int64
	ID     int64
	Fields []domain.Field
}

type ListEntriesReq struct {
	UserID     int64
	CalendarID int64
	Params     service.EntryListParams
}

// AgendaReq — ближайшие события всех доступных календарей.
type AgendaReq struct {
	UserID   int64
	From, To time.Time
	Limit    int
}

type EntryReq struct {
	UserID     int64
	CalendarID int64
	EntryID    int64
}

type WriteEntryReq struct {
	UserID     int64
	CalendarID int64
	EntryID    int64
	EventAt    time.Time
	Data       map[string]any
}

type DeleteEntriesReq struct {
	UserID     int64
	CalendarID int64
	IDs        []int64
}

type ExportReq struct {
	UserID     int64
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
	UserID int64
	// CalendarID — для какого календаря файл (0 — неизвестно): решает, чья
	// квота платит.
	CalendarID int64
	FileName   string
	Mime       string
	Data       []byte
}

type ShareReq struct {
	UserID     int64
	CalendarID int64
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
			return s.ListCalendars(ctx, request.(UserReq).UserID)
		},
		GetCalendar: func(ctx context.Context, request any) (any, error) {
			r := request.(CalendarReq)
			return s.GetCalendar(ctx, r.UserID, r.ID)
		},
		CreateCalendar: func(ctx context.Context, request any) (any, error) {
			r := request.(CreateCalendarReq)
			return s.CreateCalendar(ctx, r.UserID, r.CompanyID, r.Name)
		},
		UpdateCalendar: func(ctx context.Context, request any) (any, error) {
			r := request.(UpdateCalendarReq)
			return s.UpdateCalendar(ctx, r.UserID, r.ID, r.Name)
		},
		MoveCalendar: func(ctx context.Context, request any) (any, error) {
			r := request.(MoveCalendarReq)
			return s.MoveCalendar(ctx, r.UserID, r.ID, r.CompanyID, r.TeamAccess)
		},
		DeleteCalendar: func(ctx context.Context, request any) (any, error) {
			r := request.(CalendarReq)
			return nil, s.DeleteCalendar(ctx, r.UserID, r.ID)
		},
		ReplaceFields: func(ctx context.Context, request any) (any, error) {
			r := request.(ReplaceFieldsReq)
			return s.ReplaceFields(ctx, r.UserID, r.ID, r.Fields)
		},
		Agenda: func(ctx context.Context, request any) (any, error) {
			r := request.(AgendaReq)
			return s.Agenda(ctx, r.UserID, r.From, r.To, r.Limit)
		},
		ListEntries: func(ctx context.Context, request any) (any, error) {
			r := request.(ListEntriesReq)
			return s.ListEntries(ctx, r.UserID, r.CalendarID, r.Params)
		},
		GetEntry: func(ctx context.Context, request any) (any, error) {
			r := request.(EntryReq)
			return s.GetEntry(ctx, r.UserID, r.CalendarID, r.EntryID)
		},
		CreateEntry: func(ctx context.Context, request any) (any, error) {
			r := request.(WriteEntryReq)
			return s.CreateEntry(ctx, r.UserID, r.CalendarID, r.EventAt, r.Data)
		},
		UpdateEntry: func(ctx context.Context, request any) (any, error) {
			r := request.(WriteEntryReq)
			return s.UpdateEntry(ctx, r.UserID, r.CalendarID, r.EntryID, r.EventAt, r.Data)
		},
		DeleteEntry: func(ctx context.Context, request any) (any, error) {
			r := request.(EntryReq)
			return nil, s.DeleteEntry(ctx, r.UserID, r.CalendarID, r.EntryID)
		},
		DeleteEntries: func(ctx context.Context, request any) (any, error) {
			r := request.(DeleteEntriesReq)
			return s.DeleteEntries(ctx, r.UserID, r.CalendarID, r.IDs)
		},
		ExportEntries: func(ctx context.Context, request any) (any, error) {
			r := request.(ExportReq)
			data, name, err := s.ExportEntries(ctx, r.UserID, r.CalendarID, r.FieldIDs, r.Params, r.IDs)
			if err != nil {
				return nil, err
			}
			return ExportResp{Data: data, Name: name}, nil
		},
		Upload: func(ctx context.Context, request any) (any, error) {
			r := request.(UploadReq)
			return s.SaveUpload(ctx, r.UserID, r.CalendarID, r.FileName, r.Mime, r.Data)
		},
		ListShares: func(ctx context.Context, request any) (any, error) {
			r := request.(ShareReq)
			return s.ListShares(ctx, r.UserID, r.CalendarID)
		},
		CreateShare: func(ctx context.Context, request any) (any, error) {
			r := request.(ShareReq)
			return s.CreateShare(ctx, r.UserID, r.CalendarID)
		},
		RevokeShare: func(ctx context.Context, request any) (any, error) {
			r := request.(ShareReq)
			return nil, s.RevokeShare(ctx, r.UserID, r.CalendarID, r.ShareID)
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
