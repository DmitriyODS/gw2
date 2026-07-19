package clients

import (
	"context"

	"google.golang.org/grpc"

	"github.com/DmitriyODS/gw2/back-go/alice/internal/domain"
	"github.com/DmitriyODS/gw2/back-go/pkg/gen/notespb"
)

type Notes struct {
	conn *grpc.ClientConn
	stub notespb.NotesServiceClient
}

var _ domain.NotesClient = (*Notes)(nil)

func NewNotes(addr string) (*Notes, error) {
	conn, err := dial(addr)
	if err != nil {
		return nil, err
	}
	return &Notes{conn: conn, stub: notespb.NewNotesServiceClient(conn)}, nil
}

func (c *Notes) Close() { _ = c.conn.Close() }

func pbNote(n *notespb.NoteRef) *domain.NoteRef {
	if n == nil {
		return nil
	}
	return &domain.NoteRef{ID: n.GetId(), Title: n.GetTitle(), Snippet: n.GetSnippet()}
}

func (c *Notes) CreateFolder(ctx context.Context, userID int64, name string) error {
	ctx, cancel := context.WithTimeout(ctx, callTimeout)
	defer cancel()
	resp, err := c.stub.CreateFolder(ctx, &notespb.CreateFolderRequest{UserId: userID, Name: name})
	if err != nil {
		return err
	}
	return pbErr(resp.GetError())
}

func (c *Notes) CreateNote(ctx context.Context, userID, companyID int64, title, text string) (*domain.NoteRef, error) {
	ctx, cancel := context.WithTimeout(ctx, callTimeout)
	defer cancel()
	resp, err := c.stub.CreateNote(ctx, &notespb.CreateNoteRequest{
		UserId: userID, CompanyId: companyID, Title: title, Text: text,
	})
	if err != nil {
		return nil, err
	}
	if err := pbErr(resp.GetError()); err != nil {
		return nil, err
	}
	return pbNote(resp.GetNote()), nil
}

func (c *Notes) FindNotes(ctx context.Context, userID, companyID int64, query string, limit int) ([]domain.NoteRef, error) {
	ctx, cancel := context.WithTimeout(ctx, callTimeout)
	defer cancel()
	resp, err := c.stub.FindNotes(ctx, &notespb.FindNotesRequest{
		UserId: userID, CompanyId: companyID, Query: query, Limit: int32(limit),
	})
	if err != nil {
		return nil, err
	}
	if err := pbErr(resp.GetError()); err != nil {
		return nil, err
	}
	out := make([]domain.NoteRef, 0, len(resp.GetNotes()))
	for _, n := range resp.GetNotes() {
		out = append(out, *pbNote(n))
	}
	return out, nil
}

func (c *Notes) GetNote(ctx context.Context, userID, noteID int64) (*domain.Note, error) {
	ctx, cancel := context.WithTimeout(ctx, callTimeout)
	defer cancel()
	resp, err := c.stub.GetNote(ctx, &notespb.GetNoteRequest{UserId: userID, NoteId: noteID})
	if err != nil {
		return nil, err
	}
	if err := pbErr(resp.GetError()); err != nil {
		return nil, err
	}
	return &domain.Note{ID: resp.GetId(), Title: resp.GetTitle(), Text: resp.GetText()}, nil
}

func (c *Notes) AppendNote(ctx context.Context, userID, companyID, noteID int64, text string) (*domain.NoteRef, error) {
	ctx, cancel := context.WithTimeout(ctx, callTimeout)
	defer cancel()
	resp, err := c.stub.AppendNote(ctx, &notespb.AppendNoteRequest{
		UserId: userID, CompanyId: companyID, NoteId: noteID, Text: text,
	})
	if err != nil {
		return nil, err
	}
	if err := pbErr(resp.GetError()); err != nil {
		return nil, err
	}
	return pbNote(resp.GetNote()), nil
}

func (c *Notes) DeleteNote(ctx context.Context, userID, noteID int64) error {
	ctx, cancel := context.WithTimeout(ctx, callTimeout)
	defer cancel()
	resp, err := c.stub.DeleteNote(ctx, &notespb.DeleteNoteRequest{UserId: userID, NoteId: noteID})
	if err != nil {
		return err
	}
	return pbErr(resp.GetError())
}
