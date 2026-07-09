package service

import (
	"context"
	"strings"

	"github.com/DmitriyODS/gw2/back-go/notes/internal/domain"
)

// Export — заметка плоским текстом: заголовок, пустая строка, текст документа.
// Имя файла — заголовок заметки (fallback «Заметка»).
func (s *Service) Export(ctx context.Context, userID, id int64) (data []byte, name string, err error) {
	n, err := s.requireOwned(ctx, userID, id)
	if err != nil {
		return nil, "", err
	}
	name = strings.TrimSpace(n.Title)
	if name == "" {
		name = "Заметка"
	}
	content := strings.TrimSpace(n.Title)
	if n.TextContent != "" {
		if content != "" {
			content += "\n\n"
		}
		content += n.TextContent
	}
	return []byte(content), name, nil
}

// Import — заметка из .txt: первая строка → заголовок, остальное → документ из
// параграфов.
func (s *Service) Import(ctx context.Context, userID int64, text string) (*domain.Note, error) {
	text = strings.ReplaceAll(text, "\r\n", "\n")
	title, body, _ := strings.Cut(text, "\n")
	title = strings.TrimSpace(title)
	if r := []rune(title); len(r) > 300 {
		title = string(r[:300])
	}
	body = strings.TrimLeft(body, "\n")
	doc := domain.TextToDoc(body)
	n := &domain.Note{
		OwnerID: userID, Title: title, Doc: doc,
		TextContent: domain.DocText(doc), GroupIDs: []int64{},
	}
	if err := s.repo.CreateNote(ctx, n); err != nil {
		return nil, err
	}
	s.publishNote(ctx, "note:created", n)
	return n, nil
}
