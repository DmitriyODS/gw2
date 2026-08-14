package service

import (
	"context"
	"strings"
	"time"

	"github.com/DmitriyODS/gw2/back-go/registry/internal/domain"
)

/* Учётный реестр: выдача позиций под ответственного.

   Срок задаётся ЛИБО количеством дней, ЛИБО прямой датой возврата — на экране
   это два связанных поля, и пересчёт между ними идёт на клиенте. Сервер хранит
   дату: количество дней без точки отсчёта смысла не имеет, а «выдано до» нужно
   показывать и через месяц. Рабочие дни считает клиент по календарю компании —
   праздники платформе неизвестны, и об этом честно сказано в интерфейсе.

   Операции доступны с уровня edit — и участнику, и внешней ссылке. Поэтому
   каждая разделена надвое: проверка доступа (своя у каждого пути) и общее
   ядро, работающее с уже проверенными реестром и записью. */

// IssueParams — выдача позиции.
type IssueParams struct {
	IssuedTo     string
	HolderName   string
	HolderPhone  string
	HolderUserID *int64
	DueAt        *time.Time
	Comment      string
}

// Issue — выдать позицию. Двойную выдачу отбивает уникальный индекс открытых
// выдач: проверка «не выдана ли» и вставка не атомарны, а между ними успевает
// вклиниться вторая вкладка.
func (s *Service) Issue(ctx context.Context, userID, registryID, recordID int64, p IssueParams) (*domain.Issue, error) {
	reg, rec, err := s.requireAccounting(ctx, userID, registryID, recordID)
	if err != nil {
		return nil, err
	}
	return s.issueIn(ctx, reg, rec, &userID, p)
}

// Extend — продлить срок возврата открытой выдачи.
func (s *Service) Extend(ctx context.Context, userID, registryID, recordID int64, dueAt *time.Time, comment string) (*domain.Issue, error) {
	reg, rec, err := s.requireAccounting(ctx, userID, registryID, recordID)
	if err != nil {
		return nil, err
	}
	return s.extendIn(ctx, reg, rec, &userID, dueAt, comment)
}

// Return — принять позицию обратно.
func (s *Service) Return(ctx context.Context, userID, registryID, recordID int64, comment string) (*domain.Issue, error) {
	reg, rec, err := s.requireAccounting(ctx, userID, registryID, recordID)
	if err != nil {
		return nil, err
	}
	return s.returnIn(ctx, reg, rec, &userID, comment)
}

// IssueHistory — вся история движения позиции, свежие первыми.
func (s *Service) IssueHistory(ctx context.Context, userID, registryID, recordID int64) ([]*domain.Issue, error) {
	// История — сведения о самой позиции, поэтому её видит и уровень view:
	// «кто держит вещь» нужен всем, кто вообще видит реестр.
	a, err := s.actor(ctx, userID)
	if err != nil {
		return nil, err
	}
	reg, err := s.require(ctx, a, registryID, domain.AccessView)
	if err != nil {
		return nil, err
	}
	rec, err := s.accountingRecord(ctx, reg, recordID)
	if err != nil {
		return nil, err
	}
	return s.repo.IssueHistory(ctx, rec.ID)
}

// ── Те же операции по внешней ссылке ──
//
// Выдача — работа с записями, а не со структурой, поэтому её даёт уже уровень
// edit. Автор действия здесь может быть неизвестен (гость без аккаунта), и в
// журнале останется только то, что он ввёл руками.

func (s *Service) SharedIssue(ctx context.Context, code string, v Visitor, recordID int64, p IssueParams) (*domain.Issue, error) {
	reg, rec, err := s.sharedAccounting(ctx, code, v, recordID)
	if err != nil {
		return nil, err
	}
	return s.issueIn(ctx, reg, rec, v.UserID, p)
}

func (s *Service) SharedExtend(ctx context.Context, code string, v Visitor, recordID int64, dueAt *time.Time, comment string) (*domain.Issue, error) {
	reg, rec, err := s.sharedAccounting(ctx, code, v, recordID)
	if err != nil {
		return nil, err
	}
	return s.extendIn(ctx, reg, rec, v.UserID, dueAt, comment)
}

func (s *Service) SharedReturn(ctx context.Context, code string, v Visitor, recordID int64, comment string) (*domain.Issue, error) {
	reg, rec, err := s.sharedAccounting(ctx, code, v, recordID)
	if err != nil {
		return nil, err
	}
	return s.returnIn(ctx, reg, rec, v.UserID, comment)
}

func (s *Service) SharedIssueHistory(ctx context.Context, code string, v Visitor, recordID int64) ([]*domain.Issue, error) {
	reg, _, err := s.resolveShare(ctx, code, v)
	if err != nil {
		return nil, err
	}
	rec, err := s.accountingRecord(ctx, reg, recordID)
	if err != nil {
		return nil, err
	}
	return s.repo.IssueHistory(ctx, rec.ID)
}

// ── Ядро операций (доступ уже проверен) ──

func (s *Service) issueIn(ctx context.Context, reg *domain.Registry, rec *domain.Record, actorID *int64, p IssueParams) (*domain.Issue, error) {
	issue := &domain.Issue{
		RegistryID:   reg.ID,
		RecordID:     rec.ID,
		IssuedTo:     strings.TrimSpace(p.IssuedTo),
		HolderName:   strings.TrimSpace(p.HolderName),
		HolderPhone:  strings.TrimSpace(p.HolderPhone),
		HolderUserID: p.HolderUserID,
		IssuedBy:     actorID,
		DueAt:        p.DueAt,
	}
	if err := s.repo.CreateIssue(ctx, issue, strings.TrimSpace(p.Comment)); err != nil {
		return nil, err
	}
	s.publishIssue(ctx, reg, rec.ID, issue)
	return issue, nil
}

func (s *Service) extendIn(ctx context.Context, reg *domain.Registry, rec *domain.Record, actorID *int64, dueAt *time.Time, comment string) (*domain.Issue, error) {
	open, err := s.openIssue(ctx, rec.ID)
	if err != nil {
		return nil, err
	}
	if err := s.repo.ExtendIssue(ctx, open.ID, dueAt, strings.TrimSpace(comment), actorID); err != nil {
		return nil, err
	}
	open.DueAt = dueAt
	s.publishIssue(ctx, reg, rec.ID, open)
	return open, nil
}

func (s *Service) returnIn(ctx context.Context, reg *domain.Registry, rec *domain.Record, actorID *int64, comment string) (*domain.Issue, error) {
	open, err := s.openIssue(ctx, rec.ID)
	if err != nil {
		return nil, err
	}
	at := time.Now()
	ok, err := s.repo.ReturnIssue(ctx, open.ID, at, strings.TrimSpace(comment), actorID)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, domain.ErrNotIssued
	}
	open.ReturnedAt = &at
	s.publishIssue(ctx, reg, rec.ID, nil)
	return open, nil
}

// ── Проверки доступа ──

// requireAccounting — учётный реестр с правом вести записи и сама запись.
// Выдача — не правка структуры, поэтому достаточно уровня edit.
func (s *Service) requireAccounting(ctx context.Context, userID, registryID, recordID int64) (*domain.Registry, *domain.Record, error) {
	a, err := s.actor(ctx, userID)
	if err != nil {
		return nil, nil, err
	}
	reg, err := s.require(ctx, a, registryID, domain.AccessEdit)
	if err != nil {
		return nil, nil, err
	}
	rec, err := s.accountingRecord(ctx, reg, recordID)
	if err != nil {
		return nil, nil, err
	}
	return reg, rec, nil
}

// sharedAccounting — то же по коду ссылки уровня edit.
func (s *Service) sharedAccounting(ctx context.Context, code string, v Visitor, recordID int64) (*domain.Registry, *domain.Record, error) {
	reg, err := s.resolveShareAt(ctx, code, v, domain.AccessEdit)
	if err != nil {
		return nil, nil, err
	}
	rec, err := s.accountingRecord(ctx, reg, recordID)
	if err != nil {
		return nil, nil, err
	}
	return reg, rec, nil
}

// accountingRecord — запись УЧЁТНОГО реестра (доступ уже проверен).
func (s *Service) accountingRecord(ctx context.Context, reg *domain.Registry, recordID int64) (*domain.Record, error) {
	if !reg.Accounting {
		return nil, domain.ErrNotAccounting
	}
	rec, err := s.repo.GetRecord(ctx, recordID)
	if err != nil {
		return nil, err
	}
	if rec == nil || rec.RegistryID != reg.ID {
		return nil, domain.ErrRecordNotFound
	}
	return rec, nil
}

func (s *Service) openIssue(ctx context.Context, recordID int64) (*domain.Issue, error) {
	open, err := s.repo.OpenIssues(ctx, []int64{recordID})
	if err != nil {
		return nil, err
	}
	issue := open[recordID]
	if issue == nil {
		return nil, domain.ErrNotIssued
	}
	return issue, nil
}

// publishIssue — состояние позиции изменилось: плашка в чужих таблицах должна
// обновиться сама. Событие несёт открытую выдачу (nil — вещь вернули).
func (s *Service) publishIssue(ctx context.Context, reg *domain.Registry, recordID int64, issue *domain.Issue) {
	s.publish(ctx, reg.ID, "record:issue", map[string]any{
		"registry_id": reg.ID, "record_id": recordID, "issue": issue,
	})
}
