package service

import (
	"context"
	"strings"
	"time"

	"github.com/DmitriyODS/gw2/back-go/notes/internal/domain"
)

// semanticLimit — сколько ближайших заметок берём из pgvector.
const semanticLimit = 40

// ReindexNoteAsync — фоновая (пере)индексация заметки под ИИ-поиск: векторизуем
// текст ключом активной компании и сохраняем эмбеддинг. Fire-and-forget — на
// путь сохранения заметки не влияет (ошибки только в лог). companyID==0 или
// выключенный AI — просто пропуск (поиск откатится на текстовый).
func (s *Service) ReindexNoteAsync(noteID, companyID int64) {
	if !s.aiEnabled() || companyID <= 0 {
		return
	}
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		if err := s.reindexNote(ctx, noteID, companyID); err != nil {
			s.log.Warn("notes.reindex_failed", "note", noteID, "error", err)
		}
	}()
}

func (s *Service) reindexNote(ctx context.Context, noteID, companyID int64) error {
	n, err := s.repo.GetNote(ctx, noteID)
	if err != nil || n == nil {
		return err
	}
	text := strings.TrimSpace(n.Title + "\n" + n.TextContent)
	if text == "" {
		return nil
	}
	vec, model, err := s.embedder.Embed(ctx, companyID, text)
	if err != nil {
		return err
	}
	return s.repo.UpsertNoteEmbedding(ctx, noteID, n.OwnerID, vec, model)
}

// semanticNotes — ИИ-поиск по СВОИМ заметкам (глобально): (плитки, ok). ok=false
// — откат на текстовый поиск (пустой запрос/ошибка эмбеддинга/нет проиндексиро-
// ванных совпадений). Fail-open как в задачах.
func (s *Service) semanticNotes(ctx context.Context, userID, companyID int64, query string, archived bool) ([]*domain.Note, bool) {
	vec, model, err := s.embedder.Embed(ctx, companyID, query)
	if err != nil {
		s.log.Warn("notes.search.embed_failed", "company", companyID, "error", err)
		return nil, false
	}
	ids, err := s.repo.SearchNoteEmbeddings(ctx, userID, vec, model, archived, semanticLimit)
	if err != nil {
		s.log.Warn("notes.search.query_failed", "error", err)
		return nil, false
	}
	if len(ids) == 0 {
		return nil, false // нечего или не проиндексировано — пусть добьёт текстовый
	}
	notes, err := s.repo.ListNotesByIDs(ctx, userID, ids, archived)
	if err != nil {
		return nil, false
	}
	return notes, true
}
