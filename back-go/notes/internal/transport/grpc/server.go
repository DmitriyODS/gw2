// Package grpc — исходящий gRPC-транспорт notesvc (его зовёт alicesvc:
// голосовые операции навыка Алисы над личными заметками владельца).
// Бизнес-ошибки уезжают полем Error в ответе — транспорт всегда отвечает OK.
package grpc

import (
	"context"
	"strings"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/DmitriyODS/gw2/back-go/notes/internal/domain"
	"github.com/DmitriyODS/gw2/back-go/notes/internal/service"
	"github.com/DmitriyODS/gw2/back-go/pkg/gen/notespb"
)

type Server struct {
	notespb.UnimplementedNotesServiceServer
	svc *service.Service
}

func NewServer(svc *service.Service) *Server { return &Server{svc: svc} }

func pbError(err error) (*notespb.Error, error) {
	if de := domain.AsDomainError(err); de != nil {
		return &notespb.Error{Code: de.Code, Message: de.Message, HttpStatus: int32(de.HTTPStatus)}, nil
	}
	return nil, status.Error(codes.Internal, err.Error())
}

const snippetRunes = 100

func pbNote(n *domain.Note) *notespb.NoteRef {
	snippet := n.TextContent
	if snippet == "" {
		snippet = n.Excerpt
	}
	snippet = strings.TrimSpace(snippet)
	if r := []rune(snippet); len(r) > snippetRunes {
		snippet = string(r[:snippetRunes]) + "…"
	}
	return &notespb.NoteRef{Id: n.ID, Title: n.Title, Snippet: snippet}
}

func optID(id int64) *int64 {
	if id == 0 {
		return nil
	}
	return &id
}

func (s *Server) ListFolders(ctx context.Context, req *notespb.ListFoldersRequest) (*notespb.ListFoldersResponse, error) {
	tree, err := s.svc.ListFolders(ctx, req.GetUserId())
	if err != nil {
		pe, ierr := pbError(err)
		return &notespb.ListFoldersResponse{Error: pe}, ierr
	}
	out := make([]*notespb.Folder, 0, len(tree.Folders))
	for _, f := range tree.Folders {
		pf := &notespb.Folder{Id: f.ID, Name: f.Name}
		if f.ParentID != nil {
			pf.ParentId = *f.ParentID
		}
		out = append(out, pf)
	}
	return &notespb.ListFoldersResponse{Folders: out}, nil
}

func (s *Server) CreateFolder(ctx context.Context, req *notespb.CreateFolderRequest) (*notespb.CreateFolderResponse, error) {
	f, err := s.svc.CreateFolder(ctx, req.GetUserId(), req.GetName(), "", optID(req.GetParentId()))
	if err != nil {
		pe, ierr := pbError(err)
		return &notespb.CreateFolderResponse{Error: pe}, ierr
	}
	pf := &notespb.Folder{Id: f.ID, Name: f.Name}
	if f.ParentID != nil {
		pf.ParentId = *f.ParentID
	}
	return &notespb.CreateFolderResponse{Folder: pf}, nil
}

func (s *Server) CreateNote(ctx context.Context, req *notespb.CreateNoteRequest) (*notespb.CreateNoteResponse, error) {
	n, err := s.svc.CreateNoteFromText(ctx, req.GetUserId(), req.GetTitle(), req.GetText(), optID(req.GetFolderId()))
	if err != nil {
		pe, ierr := pbError(err)
		return &notespb.CreateNoteResponse{Error: pe}, ierr
	}
	if cid := req.GetCompanyId(); cid > 0 {
		s.svc.ReindexNoteAsync(n.ID, cid)
	}
	return &notespb.CreateNoteResponse{Note: pbNote(n)}, nil
}

func (s *Server) FindNotes(ctx context.Context, req *notespb.FindNotesRequest) (*notespb.FindNotesResponse, error) {
	notes, err := s.svc.ListNotes(ctx, req.GetUserId(), req.GetCompanyId(), service.ListNotesParams{
		Search: req.GetQuery(),
	})
	if err != nil {
		pe, ierr := pbError(err)
		return &notespb.FindNotesResponse{Error: pe}, ierr
	}
	limit := int(req.GetLimit())
	if limit <= 0 {
		limit = 5
	}
	if len(notes) > limit {
		notes = notes[:limit]
	}
	out := make([]*notespb.NoteRef, 0, len(notes))
	for _, n := range notes {
		out = append(out, pbNote(n))
	}
	return &notespb.FindNotesResponse{Notes: out}, nil
}

func (s *Server) GetNote(ctx context.Context, req *notespb.GetNoteRequest) (*notespb.GetNoteResponse, error) {
	n, err := s.svc.GetNote(ctx, req.GetUserId(), req.GetNoteId())
	if err != nil {
		pe, ierr := pbError(err)
		return &notespb.GetNoteResponse{Error: pe}, ierr
	}
	text := n.TextContent
	if text == "" {
		text = domain.DocText(n.Doc)
	}
	return &notespb.GetNoteResponse{Id: n.ID, Title: n.Title, Text: text}, nil
}

func (s *Server) AppendNote(ctx context.Context, req *notespb.AppendNoteRequest) (*notespb.AppendNoteResponse, error) {
	n, err := s.svc.AppendText(ctx, req.GetUserId(), req.GetNoteId(), req.GetText())
	if err != nil {
		pe, ierr := pbError(err)
		return &notespb.AppendNoteResponse{Error: pe}, ierr
	}
	if cid := req.GetCompanyId(); cid > 0 {
		s.svc.ReindexNoteAsync(n.ID, cid)
	}
	return &notespb.AppendNoteResponse{Note: pbNote(n)}, nil
}

func (s *Server) DeleteNote(ctx context.Context, req *notespb.DeleteNoteRequest) (*notespb.DeleteNoteResponse, error) {
	if err := s.svc.DeleteNote(ctx, req.GetUserId(), req.GetNoteId()); err != nil {
		pe, ierr := pbError(err)
		return &notespb.DeleteNoteResponse{Error: pe}, ierr
	}
	return &notespb.DeleteNoteResponse{}, nil
}
