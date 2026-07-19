// Сценарии личных заметок (notesvc).
package service

import (
	"context"
	"fmt"
	"strings"

	"github.com/DmitriyODS/gw2/back-go/alice/internal/domain"
)

func (s *Service) noteCreate(ctx context.Context, sess *session, title, text string) *domain.WebhookResponse {
	if strings.TrimSpace(title) == "" {
		return reply("Скажите название заметки: например, «создай заметку идеи».")
	}
	n, err := s.notes.CreateNote(ctx, sess.userID, sess.companyID, title, text)
	if err != nil {
		return s.errReply(err)
	}
	if text != "" {
		return reply(fmt.Sprintf("Готово, создала заметку «%s» с текстом.", n.Title))
	}
	return reply(fmt.Sprintf("Готово, создала заметку «%s».", n.Title))
}

func (s *Service) folderCreate(ctx context.Context, sess *session, name string) *domain.WebhookResponse {
	if err := s.notes.CreateFolder(ctx, sess.userID, name); err != nil {
		return s.errReply(err)
	}
	return reply(fmt.Sprintf("Готово, создала папку «%s».", name))
}

// findNote — одна заметка по названию: 0 — «не нашла», несколько — выбор.
func (s *Service) findNote(ctx context.Context, sess *session, query, kind, text string) (*domain.NoteRef, *domain.WebhookResponse) {
	notes, err := s.notes.FindNotes(ctx, sess.userID, sess.companyID, query, 5)
	if err != nil {
		return nil, s.errReply(err)
	}
	switch len(notes) {
	case 0:
		return nil, reply(fmt.Sprintf("Не нашла заметку по запросу «%s».", query))
	case 1:
		return &notes[0], nil
	}
	st := domain.DialogState{Pending: "choose_note", Kind: kind, Text: text}
	names := make([]string, 0, len(notes))
	for _, n := range notes {
		name := n.Title
		if name == "" {
			name = truncate(n.Snippet, 40)
		}
		st.Options = append(st.Options, domain.Option{ID: n.ID, Name: name})
		names = append(names, name)
	}
	return nil, replyState("Нашла несколько заметок. Какая из них?"+enumerate(names), st)
}

func (s *Service) noteAppend(ctx context.Context, sess *session, title, text string) *domain.WebhookResponse {
	n, r := s.findNote(ctx, sess, title, "note_append", text)
	if r != nil {
		return r
	}
	if strings.TrimSpace(text) == "" {
		st := domain.DialogState{Pending: "append_text", NoteID: n.ID, Title: n.Title}
		return replyState(fmt.Sprintf("Что дописать в заметку «%s»?", n.Title), st)
	}
	return s.noteAppendTo(ctx, sess, n.ID, n.Title, text)
}

func (s *Service) noteAppendTo(ctx context.Context, sess *session, noteID int64, title, text string) *domain.WebhookResponse {
	n, err := s.notes.AppendNote(ctx, sess.userID, sess.companyID, noteID, text)
	if err != nil {
		return s.errReply(err)
	}
	return reply(fmt.Sprintf("Дописала в заметку «%s».", n.Title))
}

func (s *Service) noteRead(ctx context.Context, sess *session, query string) *domain.WebhookResponse {
	n, r := s.findNote(ctx, sess, query, "note_read", "")
	if r != nil {
		return r
	}
	return s.noteReadByID(ctx, sess, n.ID)
}

func (s *Service) noteReadByID(ctx context.Context, sess *session, noteID int64) *domain.WebhookResponse {
	n, err := s.notes.GetNote(ctx, sess.userID, noteID)
	if err != nil {
		return s.errReply(err)
	}
	text := strings.TrimSpace(n.Text)
	if text == "" {
		return reply(fmt.Sprintf("Заметка «%s» пока пустая.", n.Title))
	}
	return reply(fmt.Sprintf("Заметка «%s»:\n%s", n.Title, text))
}

func (s *Service) noteDelete(ctx context.Context, sess *session, query string) *domain.WebhookResponse {
	n, r := s.findNote(ctx, sess, query, "note_delete", "")
	if r != nil {
		return r
	}
	st := domain.DialogState{Pending: "confirm_delete_note", NoteID: n.ID, Title: n.Title}
	return replyState(fmt.Sprintf("Удалить заметку «%s»? Скажите «да» или «нет».", n.Title), st)
}
