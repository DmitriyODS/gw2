// Сценарии личного ежедневника (diarysvc).
package service

import (
	"context"
	"fmt"

	"github.com/DmitriyODS/gw2/back-go/alice/internal/domain"
)

const defaultDiaryName = "Ежедневник"

func (s *Service) diaryCreate(ctx context.Context, sess *session, name string) *domain.WebhookResponse {
	d, err := s.diary.CreateDiary(ctx, sess.userID, name)
	if err != nil {
		return s.errReply(err)
	}
	return reply(fmt.Sprintf("Готово, создала ежедневник «%s».", d.Name))
}

// diaryAdd — запись в ежедневник: единственный используется сразу; нет ни
// одного — создаётся автоматически; несколько — уточняющий выбор.
func (s *Service) diaryAdd(ctx context.Context, sess *session, title, date string) *domain.WebhookResponse {
	if title == "" {
		return reply("Скажите, что записать: например, «запиши на завтра позвонить клиенту».")
	}
	if date == "" {
		date = sess.now.Format(dayLayout)
	}
	diaries, err := s.diary.ListDiaries(ctx, sess.userID)
	if err != nil {
		return s.errReply(err)
	}
	switch len(diaries) {
	case 0:
		d, err := s.diary.CreateDiary(ctx, sess.userID, defaultDiaryName)
		if err != nil {
			return s.errReply(err)
		}
		return s.diaryAddTo(ctx, sess, d.ID, title, date)
	case 1:
		return s.diaryAddTo(ctx, sess, diaries[0].ID, title, date)
	}
	st := domain.DialogState{Pending: "choose_diary", Kind: "diary_add", Title: title, Date: date}
	names := make([]string, 0, len(diaries))
	for _, d := range diaries {
		st.Options = append(st.Options, domain.Option{ID: d.ID, Name: d.Name})
		names = append(names, d.Name)
	}
	return replyState("В какой ежедневник записать?"+enumerate(names), st)
}

func (s *Service) diaryAddTo(ctx context.Context, sess *session, diaryID int64, title, date string) *domain.WebhookResponse {
	if date == "" {
		date = sess.now.Format(dayLayout)
	}
	e, err := s.diary.CreateEntry(ctx, sess.userID, diaryID, date, title)
	if err != nil {
		return s.errReply(err)
	}
	return reply(fmt.Sprintf("Записала на %s: «%s».", HumanDate(e.Date, sess.now), e.Title))
}

// diaryList — план на день по ВСЕМ своим ежедневникам.
func (s *Service) diaryList(ctx context.Context, sess *session, date string) *domain.WebhookResponse {
	if date == "" {
		date = sess.now.Format(dayLayout)
	}
	diaries, err := s.diary.ListDiaries(ctx, sess.userID)
	if err != nil {
		return s.errReply(err)
	}
	var names []string
	for _, d := range diaries {
		entries, err := s.diary.ListEntries(ctx, sess.userID, d.ID, date, date)
		if err != nil {
			return s.errReply(err)
		}
		for _, e := range entries {
			names = append(names, e.Title)
		}
	}
	day := HumanDate(date, sess.now)
	if len(names) == 0 {
		return reply(fmt.Sprintf("На %s записей нет.", day))
	}
	return reply(fmt.Sprintf("На %s записей: %d.", day, len(names)) + enumerate(names))
}

// findEntries — активные записи всех ежедневников, совпадающие с текстом.
func (s *Service) findEntries(ctx context.Context, sess *session, query string) ([]domain.EntryOption, *domain.WebhookResponse) {
	diaries, err := s.diary.ListDiaries(ctx, sess.userID)
	if err != nil {
		return nil, s.errReply(err)
	}
	var found []domain.EntryOption
	for _, d := range diaries {
		entries, err := s.diary.ListEntries(ctx, sess.userID, d.ID, "", "")
		if err != nil {
			return nil, s.errReply(err)
		}
		for _, e := range entries {
			if wordsMatch(query, e.Title) {
				name := fmt.Sprintf("%s (%s)", e.Title, HumanDate(e.Date, sess.now))
				found = append(found, domain.EntryOption{EntryID: e.ID, DiaryID: d.ID, Name: name})
			}
		}
	}
	return found, nil
}

// resolveEntry — одна запись по тексту: 0 — «не нашла», несколько — выбор.
func (s *Service) resolveEntry(ctx context.Context, sess *session, query, kind, date string) (*domain.EntryOption, *domain.WebhookResponse) {
	found, r := s.findEntries(ctx, sess, query)
	if r != nil {
		return nil, r
	}
	switch len(found) {
	case 0:
		return nil, reply(fmt.Sprintf("Не нашла запись по запросу «%s».", query))
	case 1:
		return &found[0], nil
	}
	if len(found) > 5 {
		found = found[:5]
	}
	st := domain.DialogState{Pending: "choose_entry", Kind: kind, Date: date, EntryOptions: found}
	names := make([]string, 0, len(found))
	for _, f := range found {
		names = append(names, f.Name)
	}
	return nil, replyState("Нашла несколько записей. Какая из них?"+enumerate(names), st)
}

func (s *Service) diaryDone(ctx context.Context, sess *session, query string) *domain.WebhookResponse {
	eo, r := s.resolveEntry(ctx, sess, query, "diary_done", "")
	if r != nil {
		return r
	}
	return s.entryDone(ctx, sess, eo.DiaryID, eo.EntryID)
}

func (s *Service) entryDone(ctx context.Context, sess *session, diaryID, entryID int64) *domain.WebhookResponse {
	e, err := s.diary.SetEntryDone(ctx, sess.userID, diaryID, entryID, true)
	if err != nil {
		return s.errReply(err)
	}
	return reply(fmt.Sprintf("Отметила «%s» выполненным. Так держать!", e.Title))
}

func (s *Service) diaryMove(ctx context.Context, sess *session, query, date string) *domain.WebhookResponse {
	eo, r := s.resolveEntry(ctx, sess, query, "diary_move", date)
	if r != nil {
		return r
	}
	if date == "" {
		st := domain.DialogState{Pending: "move_date", DiaryID: eo.DiaryID, EntryID: eo.EntryID, Title: eo.Name}
		return replyState("На какой день перенести? Например, «на завтра».", st)
	}
	return s.entryMove(ctx, sess, eo.DiaryID, eo.EntryID, eo.Name, date)
}

func (s *Service) entryMove(ctx context.Context, sess *session, diaryID, entryID int64, name, date string) *domain.WebhookResponse {
	if date == "" {
		st := domain.DialogState{Pending: "move_date", DiaryID: diaryID, EntryID: entryID, Title: name}
		return replyState("На какой день перенести? Например, «на завтра».", st)
	}
	e, err := s.diary.MoveEntry(ctx, sess.userID, diaryID, entryID, date)
	if err != nil {
		return s.errReply(err)
	}
	return reply(fmt.Sprintf("Перенесла «%s» на %s.", e.Title, HumanDate(e.Date, sess.now)))
}

func (s *Service) diaryDelete(ctx context.Context, sess *session, query string) *domain.WebhookResponse {
	eo, r := s.resolveEntry(ctx, sess, query, "diary_delete", "")
	if r != nil {
		return r
	}
	st := domain.DialogState{Pending: "confirm_delete_entry",
		DiaryID: eo.DiaryID, EntryID: eo.EntryID, Title: eo.Name}
	return replyState(fmt.Sprintf("Удалить запись «%s»? Скажите «да» или «нет».", eo.Name), st)
}
