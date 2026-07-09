// Package endpoint — go-kit обёртки use-case'ов notesvc: единая сигнатура
// (ctx, request) → (response, error). Та же схема, что в остальных сервисах.
package endpoint

import (
	"context"
	"encoding/json"

	"github.com/go-kit/kit/endpoint"

	"github.com/DmitriyODS/gw2/back-go/notes/internal/service"
)

type Endpoints struct {
	ListNotes  endpoint.Endpoint
	GetNote    endpoint.Endpoint
	CreateNote endpoint.Endpoint
	UpdateNote endpoint.Endpoint
	DeleteNote endpoint.Endpoint
	SetGroups  endpoint.Endpoint

	ListGroups  endpoint.Endpoint
	CreateGroup endpoint.Endpoint
	UpdateGroup endpoint.Endpoint
	DeleteGroup endpoint.Endpoint

	ListShares  endpoint.Endpoint
	CreateShare endpoint.Endpoint
	RevokeShare endpoint.Endpoint

	Upload endpoint.Endpoint
	Export endpoint.Endpoint
	Import endpoint.Endpoint

	SharedNote   endpoint.Endpoint
	SharedUpdate endpoint.Endpoint
}

// ── Request-типы ──

type ListNotesReq struct {
	UserID  int64
	GroupID int64
	Search  string
}

type NoteReq struct {
	UserID int64
	ID     int64
}

type CreateNoteReq struct {
	UserID int64
	Title  string
}

// UpdateNoteReq — частичная правка: nil-поля не меняются.
type UpdateNoteReq struct {
	UserID int64
	ID     int64
	Title  *string
	Color  *string
	Doc    json.RawMessage
}

type SetGroupsReq struct {
	UserID   int64
	ID       int64
	GroupIDs []int64
}

type GroupReq struct {
	UserID int64
	ID     int64
	Name   string
}

type ShareReq struct {
	UserID  int64
	NoteID  int64
	ShareID int64
	Access  string
}

type UploadReq struct {
	UserID   int64
	NoteID   int64
	FileName string
	Data     []byte
}

type ImportReq struct {
	UserID int64
	Text   string
}

type ExportResp struct {
	Data []byte
	Name string
}

type SharedUpdateReq struct {
	Code  string
	Title *string
	Doc   json.RawMessage
}

func New(s *service.Service) Endpoints {
	return Endpoints{
		ListNotes: func(ctx context.Context, request any) (any, error) {
			r := request.(ListNotesReq)
			return s.ListNotes(ctx, r.UserID, r.GroupID, r.Search)
		},
		GetNote: func(ctx context.Context, request any) (any, error) {
			r := request.(NoteReq)
			return s.GetNote(ctx, r.UserID, r.ID)
		},
		CreateNote: func(ctx context.Context, request any) (any, error) {
			r := request.(CreateNoteReq)
			return s.CreateNote(ctx, r.UserID, r.Title)
		},
		UpdateNote: func(ctx context.Context, request any) (any, error) {
			r := request.(UpdateNoteReq)
			return s.UpdateNote(ctx, r.UserID, r.ID, r.Title, r.Color, r.Doc)
		},
		DeleteNote: func(ctx context.Context, request any) (any, error) {
			r := request.(NoteReq)
			return nil, s.DeleteNote(ctx, r.UserID, r.ID)
		},
		SetGroups: func(ctx context.Context, request any) (any, error) {
			r := request.(SetGroupsReq)
			return s.SetGroups(ctx, r.UserID, r.ID, r.GroupIDs)
		},
		ListGroups: func(ctx context.Context, request any) (any, error) {
			r := request.(NoteReq)
			return s.ListGroups(ctx, r.UserID)
		},
		CreateGroup: func(ctx context.Context, request any) (any, error) {
			r := request.(GroupReq)
			return s.CreateGroup(ctx, r.UserID, r.Name)
		},
		UpdateGroup: func(ctx context.Context, request any) (any, error) {
			r := request.(GroupReq)
			return s.UpdateGroup(ctx, r.UserID, r.ID, r.Name)
		},
		DeleteGroup: func(ctx context.Context, request any) (any, error) {
			r := request.(GroupReq)
			return nil, s.DeleteGroup(ctx, r.UserID, r.ID)
		},
		ListShares: func(ctx context.Context, request any) (any, error) {
			r := request.(ShareReq)
			return s.ListShares(ctx, r.UserID, r.NoteID)
		},
		CreateShare: func(ctx context.Context, request any) (any, error) {
			r := request.(ShareReq)
			return s.CreateShare(ctx, r.UserID, r.NoteID, r.Access)
		},
		RevokeShare: func(ctx context.Context, request any) (any, error) {
			r := request.(ShareReq)
			return nil, s.RevokeShare(ctx, r.UserID, r.NoteID, r.ShareID)
		},
		Upload: func(ctx context.Context, request any) (any, error) {
			r := request.(UploadReq)
			path, err := s.Upload(ctx, r.UserID, r.NoteID, r.FileName, r.Data)
			if err != nil {
				return nil, err
			}
			return map[string]string{"path": path}, nil
		},
		Export: func(ctx context.Context, request any) (any, error) {
			r := request.(NoteReq)
			data, name, err := s.Export(ctx, r.UserID, r.ID)
			if err != nil {
				return nil, err
			}
			return ExportResp{Data: data, Name: name}, nil
		},
		Import: func(ctx context.Context, request any) (any, error) {
			r := request.(ImportReq)
			return s.Import(ctx, r.UserID, r.Text)
		},
		SharedNote: func(ctx context.Context, request any) (any, error) {
			return s.GetSharedNote(ctx, request.(string))
		},
		SharedUpdate: func(ctx context.Context, request any) (any, error) {
			r := request.(SharedUpdateReq)
			return s.UpdateSharedNote(ctx, r.Code, r.Title, r.Doc)
		},
	}
}
