// Package grpc — исходящий gRPC-транспорт diarysvc (его зовёт alicesvc:
// голосовые операции навыка Алисы над личным ежедневником владельца).
// Бизнес-ошибки уезжают полем Error в ответе — транспорт всегда отвечает OK.
package grpc

import (
	"context"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/DmitriyODS/gw2/back-go/diary/internal/domain"
	"github.com/DmitriyODS/gw2/back-go/diary/internal/service"
	"github.com/DmitriyODS/gw2/back-go/pkg/gen/diarypb"
)

type Server struct {
	diarypb.UnimplementedDiaryServiceServer
	svc *service.Service
}

func NewServer(svc *service.Service) *Server { return &Server{svc: svc} }

func pbError(err error) (*diarypb.Error, error) {
	if de := domain.AsDomainError(err); de != nil {
		return &diarypb.Error{Code: de.Code, Message: de.Message, HttpStatus: int32(de.HTTPStatus)}, nil
	}
	return nil, status.Error(codes.Internal, err.Error())
}

const dayLayout = "2006-01-02"

func pbDiary(d *domain.Diary) *diarypb.Diary {
	return &diarypb.Diary{
		Id: d.ID, Name: d.Name,
		ActiveCount: int32(d.ActiveCount), DoneCount: int32(d.DoneCount),
	}
}

func pbEntry(e *domain.Entry) *diarypb.Entry {
	return &diarypb.Entry{
		Id: e.ID, DiaryId: e.DiaryID, Date: e.Date.Format(dayLayout),
		Title: e.Title, Description: e.Description, Done: e.Done,
	}
}

func parseDay(raw string) (time.Time, error) {
	t, err := time.Parse(dayLayout, raw)
	if err != nil {
		return time.Time{}, domain.NewError("VALIDATION", "Неверный формат даты, ожидается YYYY-MM-DD", 422)
	}
	return t, nil
}

func (s *Server) ListDiaries(ctx context.Context, req *diarypb.ListDiariesRequest) (*diarypb.ListDiariesResponse, error) {
	items, err := s.svc.ListOwned(ctx, req.GetUserId())
	if err != nil {
		pe, ierr := pbError(err)
		return &diarypb.ListDiariesResponse{Error: pe}, ierr
	}
	out := make([]*diarypb.Diary, 0, len(items))
	for _, d := range items {
		out = append(out, pbDiary(d))
	}
	return &diarypb.ListDiariesResponse{Diaries: out}, nil
}

func (s *Server) CreateDiary(ctx context.Context, req *diarypb.CreateDiaryRequest) (*diarypb.CreateDiaryResponse, error) {
	d, err := s.svc.CreateDiary(ctx, req.GetUserId(), req.GetName())
	if err != nil {
		pe, ierr := pbError(err)
		return &diarypb.CreateDiaryResponse{Error: pe}, ierr
	}
	return &diarypb.CreateDiaryResponse{Diary: pbDiary(d)}, nil
}

func (s *Server) ListEntries(ctx context.Context, req *diarypb.ListEntriesRequest) (*diarypb.ListEntriesResponse, error) {
	p := service.ListParams{Archived: req.GetArchived()}
	if raw := req.GetFrom(); raw != "" {
		t, err := parseDay(raw)
		if err != nil {
			pe, ierr := pbError(err)
			return &diarypb.ListEntriesResponse{Error: pe}, ierr
		}
		p.From = &t
	}
	if raw := req.GetTo(); raw != "" {
		t, err := parseDay(raw)
		if err != nil {
			pe, ierr := pbError(err)
			return &diarypb.ListEntriesResponse{Error: pe}, ierr
		}
		p.To = &t
	}
	list, err := s.svc.ListEntries(ctx, req.GetUserId(), req.GetDiaryId(), p)
	if err != nil {
		pe, ierr := pbError(err)
		return &diarypb.ListEntriesResponse{Error: pe}, ierr
	}
	out := make([]*diarypb.Entry, 0, len(list.Items))
	for _, e := range list.Items {
		out = append(out, pbEntry(e))
	}
	return &diarypb.ListEntriesResponse{Entries: out}, nil
}

func (s *Server) CreateEntry(ctx context.Context, req *diarypb.CreateEntryRequest) (*diarypb.CreateEntryResponse, error) {
	date, err := parseDay(req.GetDate())
	if err != nil {
		pe, ierr := pbError(err)
		return &diarypb.CreateEntryResponse{Error: pe}, ierr
	}
	e, err := s.svc.CreateEntry(ctx, req.GetUserId(), req.GetDiaryId(), service.EntryInput{
		Date: date, Title: req.GetTitle(), Description: req.GetDescription(),
	})
	if err != nil {
		pe, ierr := pbError(err)
		return &diarypb.CreateEntryResponse{Error: pe}, ierr
	}
	return &diarypb.CreateEntryResponse{Entry: pbEntry(e)}, nil
}

func (s *Server) SetEntryDone(ctx context.Context, req *diarypb.SetEntryDoneRequest) (*diarypb.SetEntryDoneResponse, error) {
	e, err := s.svc.SetDone(ctx, req.GetUserId(), req.GetDiaryId(), req.GetEntryId(), req.GetDone())
	if err != nil {
		pe, ierr := pbError(err)
		return &diarypb.SetEntryDoneResponse{Error: pe}, ierr
	}
	return &diarypb.SetEntryDoneResponse{Entry: pbEntry(e)}, nil
}

func (s *Server) MoveEntry(ctx context.Context, req *diarypb.MoveEntryRequest) (*diarypb.MoveEntryResponse, error) {
	date, err := parseDay(req.GetDate())
	if err != nil {
		pe, ierr := pbError(err)
		return &diarypb.MoveEntryResponse{Error: pe}, ierr
	}
	target := req.GetTargetDiaryId()
	if target == 0 {
		target = req.GetDiaryId()
	}
	e, err := s.svc.MoveEntry(ctx, req.GetUserId(), req.GetDiaryId(), req.GetEntryId(), target, date)
	if err != nil {
		pe, ierr := pbError(err)
		return &diarypb.MoveEntryResponse{Error: pe}, ierr
	}
	return &diarypb.MoveEntryResponse{Entry: pbEntry(e)}, nil
}

func (s *Server) DeleteEntry(ctx context.Context, req *diarypb.DeleteEntryRequest) (*diarypb.DeleteEntryResponse, error) {
	if err := s.svc.DeleteEntry(ctx, req.GetUserId(), req.GetDiaryId(), req.GetEntryId()); err != nil {
		pe, ierr := pbError(err)
		return &diarypb.DeleteEntryResponse{Error: pe}, ierr
	}
	return &diarypb.DeleteEntryResponse{}, nil
}
