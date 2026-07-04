// Package endpoint — go-kit обёртки use-case'ов diarysvc: единая сигнатура
// (ctx, request) → (response, error). Та же схема, что в остальных сервисах.
package endpoint

import (
	"context"
	"time"

	"github.com/go-kit/kit/endpoint"

	"github.com/DmitriyODS/gw2/back-go/diary/internal/service"
)

type Endpoints struct {
	ListOwned   endpoint.Endpoint
	ListShared  endpoint.Endpoint
	GetDiary    endpoint.Endpoint
	CreateDiary endpoint.Endpoint
	UpdateDiary endpoint.Endpoint
	DeleteDiary endpoint.Endpoint

	ListEntries   endpoint.Endpoint
	GetEntry      endpoint.Endpoint
	CreateEntry   endpoint.Endpoint
	UpdateEntry   endpoint.Endpoint
	SetDone        endpoint.Endpoint
	SetLink        endpoint.Endpoint
	MoveEntry      endpoint.Endpoint
	ReorderEntries endpoint.Endpoint
	DeleteEntry   endpoint.Endpoint
	DeleteEntries endpoint.Endpoint
	ExportEntries endpoint.Endpoint

	ListShares  endpoint.Endpoint
	CreateShare endpoint.Endpoint
	RevokeShare endpoint.Endpoint

	ListMembers  endpoint.Endpoint
	AddMember    endpoint.Endpoint
	RemoveMember endpoint.Endpoint

	SharedDiary   endpoint.Endpoint
	SharedEntries endpoint.Endpoint
	SharedExport  endpoint.Endpoint
}

// ── Request-типы ──

type UserReq struct{ UserID int64 }

type DiaryReq struct {
	UserID int64
	ID     int64
}

type CreateDiaryReq struct {
	UserID int64
	Name   string
}

type UpdateDiaryReq struct {
	UserID int64
	ID     int64
	Name   string
}

type ListEntriesReq struct {
	UserID  int64
	DiaryID int64
	Params  service.ListParams
}

type EntryReq struct {
	UserID  int64
	DiaryID int64
	EntryID int64
}

type WriteEntryReq struct {
	UserID  int64
	DiaryID int64
	EntryID int64
	In      service.EntryInput
}

type DoneReq struct {
	UserID  int64
	DiaryID int64
	EntryID int64
	Done    bool
}

type LinkReq struct {
	UserID  int64
	DiaryID int64
	EntryID int64
	TaskID  *int64
}

type MoveEntryReq struct {
	UserID        int64
	DiaryID       int64
	EntryID       int64
	TargetDiaryID int64
	Date          time.Time // нулевая — день не меняется
}

type ReorderEntriesReq struct {
	UserID  int64
	DiaryID int64
	Date    time.Time
	IDs     []int64 // записи дня в желаемом порядке
}

type DeleteEntriesReq struct {
	UserID  int64
	DiaryID int64
	IDs     []int64
}

type ExportReq struct {
	UserID  int64
	DiaryID int64
	Params  service.ListParams
	IDs     []int64
}

type ExportResp struct {
	Data []byte
	Name string
}

type ShareReq struct {
	UserID  int64
	DiaryID int64
	ShareID int64
}

type MemberReq struct {
	UserID   int64
	DiaryID  int64
	MemberID int64
	CanCheck bool
}

type SharedEntriesReq struct {
	Code   string
	Params service.ListParams
}

type SharedExportReq struct {
	Code   string
	Params service.ListParams
	IDs    []int64
}

func New(s *service.Service) Endpoints {
	return Endpoints{
		ListOwned: func(ctx context.Context, request any) (any, error) {
			return s.ListOwned(ctx, request.(UserReq).UserID)
		},
		ListShared: func(ctx context.Context, request any) (any, error) {
			return s.ListShared(ctx, request.(UserReq).UserID)
		},
		GetDiary: func(ctx context.Context, request any) (any, error) {
			r := request.(DiaryReq)
			return s.GetDiary(ctx, r.UserID, r.ID)
		},
		CreateDiary: func(ctx context.Context, request any) (any, error) {
			r := request.(CreateDiaryReq)
			return s.CreateDiary(ctx, r.UserID, r.Name)
		},
		UpdateDiary: func(ctx context.Context, request any) (any, error) {
			r := request.(UpdateDiaryReq)
			return s.UpdateDiary(ctx, r.UserID, r.ID, r.Name)
		},
		DeleteDiary: func(ctx context.Context, request any) (any, error) {
			r := request.(DiaryReq)
			return nil, s.DeleteDiary(ctx, r.UserID, r.ID)
		},
		ListEntries: func(ctx context.Context, request any) (any, error) {
			r := request.(ListEntriesReq)
			return s.ListEntries(ctx, r.UserID, r.DiaryID, r.Params)
		},
		GetEntry: func(ctx context.Context, request any) (any, error) {
			r := request.(EntryReq)
			return s.GetEntry(ctx, r.UserID, r.DiaryID, r.EntryID)
		},
		CreateEntry: func(ctx context.Context, request any) (any, error) {
			r := request.(WriteEntryReq)
			return s.CreateEntry(ctx, r.UserID, r.DiaryID, r.In)
		},
		UpdateEntry: func(ctx context.Context, request any) (any, error) {
			r := request.(WriteEntryReq)
			return s.UpdateEntry(ctx, r.UserID, r.DiaryID, r.EntryID, r.In)
		},
		SetDone: func(ctx context.Context, request any) (any, error) {
			r := request.(DoneReq)
			return s.SetDone(ctx, r.UserID, r.DiaryID, r.EntryID, r.Done)
		},
		SetLink: func(ctx context.Context, request any) (any, error) {
			r := request.(LinkReq)
			return s.SetLink(ctx, r.UserID, r.DiaryID, r.EntryID, r.TaskID)
		},
		MoveEntry: func(ctx context.Context, request any) (any, error) {
			r := request.(MoveEntryReq)
			return s.MoveEntry(ctx, r.UserID, r.DiaryID, r.EntryID, r.TargetDiaryID, r.Date)
		},
		ReorderEntries: func(ctx context.Context, request any) (any, error) {
			r := request.(ReorderEntriesReq)
			return nil, s.ReorderEntries(ctx, r.UserID, r.DiaryID, r.Date, r.IDs)
		},
		DeleteEntry: func(ctx context.Context, request any) (any, error) {
			r := request.(EntryReq)
			return nil, s.DeleteEntry(ctx, r.UserID, r.DiaryID, r.EntryID)
		},
		DeleteEntries: func(ctx context.Context, request any) (any, error) {
			r := request.(DeleteEntriesReq)
			return s.DeleteEntries(ctx, r.UserID, r.DiaryID, r.IDs)
		},
		ExportEntries: func(ctx context.Context, request any) (any, error) {
			r := request.(ExportReq)
			data, name, err := s.ExportEntries(ctx, r.UserID, r.DiaryID, r.Params, r.IDs)
			if err != nil {
				return nil, err
			}
			return ExportResp{Data: data, Name: name}, nil
		},
		ListShares: func(ctx context.Context, request any) (any, error) {
			r := request.(ShareReq)
			return s.ListShares(ctx, r.UserID, r.DiaryID)
		},
		CreateShare: func(ctx context.Context, request any) (any, error) {
			r := request.(ShareReq)
			return s.CreateShare(ctx, r.UserID, r.DiaryID)
		},
		RevokeShare: func(ctx context.Context, request any) (any, error) {
			r := request.(ShareReq)
			return nil, s.RevokeShare(ctx, r.UserID, r.DiaryID, r.ShareID)
		},
		ListMembers: func(ctx context.Context, request any) (any, error) {
			r := request.(ShareReq)
			return s.ListMembers(ctx, r.UserID, r.DiaryID)
		},
		AddMember: func(ctx context.Context, request any) (any, error) {
			r := request.(MemberReq)
			return s.AddMember(ctx, r.UserID, r.DiaryID, r.MemberID, r.CanCheck)
		},
		RemoveMember: func(ctx context.Context, request any) (any, error) {
			r := request.(MemberReq)
			return nil, s.RemoveMember(ctx, r.UserID, r.DiaryID, r.MemberID)
		},
		SharedDiary: func(ctx context.Context, request any) (any, error) {
			return s.SharedDiary(ctx, request.(string))
		},
		SharedEntries: func(ctx context.Context, request any) (any, error) {
			r := request.(SharedEntriesReq)
			return s.SharedEntries(ctx, r.Code, r.Params)
		},
		SharedExport: func(ctx context.Context, request any) (any, error) {
			r := request.(SharedExportReq)
			data, name, err := s.SharedExport(ctx, r.Code, r.Params, r.IDs)
			if err != nil {
				return nil, err
			}
			return ExportResp{Data: data, Name: name}, nil
		},
	}
}
