// Голосовые операции навыка Алисы (gRPC от alicesvc): работа с плоским
// текстом — TipTap-doc строится/дополняется на нашей стороне.
package service

import (
	"context"

	"github.com/DmitriyODS/gw2/back-go/notes/internal/domain"
)

// CreateNoteFromText — заметка с плоским текстом тела (может быть пустым).
func (s *Service) CreateNoteFromText(ctx context.Context, userID int64, title, text string, folderID *int64) (*domain.Note, error) {
	if err := s.checkOwnFolder(ctx, userID, folderID); err != nil {
		return nil, err
	}
	doc := domain.TextToDoc(text)
	n := &domain.Note{
		OwnerID: userID, FolderID: folderID, Title: title,
		Doc: doc, TextContent: domain.DocText(doc), TagIDs: []int64{},
	}
	if err := s.repo.CreateNote(ctx, n); err != nil {
		return nil, err
	}
	s.publishNote(ctx, "note:created", n)
	return n, nil
}

// AppendText — дописать текст абзацами в конец заметки владельца.
func (s *Service) AppendText(ctx context.Context, userID, noteID int64, text string) (*domain.Note, error) {
	n, err := s.requireOwned(ctx, userID, noteID)
	if err != nil {
		return nil, err
	}
	return s.applyUpdate(ctx, n, nil, domain.AppendTextToDoc(n.Doc, text))
}
