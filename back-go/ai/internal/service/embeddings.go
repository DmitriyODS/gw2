package service

import (
	"context"
	"strings"

	"github.com/DmitriyODS/gw2/back-go/ai/internal/domain"
)

// buildTaskText — что эмбеддим (_build_text_for_task во Flask): только то,
// что определяет смысл задачи — название, отдел, ответственный. Часто
// меняющиеся поля (color, is_archived, deadline) сюда заведомо не кладём.
func buildTaskText(t *domain.TaskText) string {
	parts := make([]string, 0, 3)
	if t.Name != "" {
		parts = append(parts, t.Name)
	}
	if t.DepartmentName != nil {
		parts = append(parts, "Отдел: "+*t.DepartmentName)
	}
	if t.ResponsibleFIO != nil {
		parts = append(parts, "Ответственный: "+*t.ResponsibleFIO)
	}
	return strings.Join(parts, "\n")
}

// ScheduleReindexTask — переиндексация одной задачи в фоне под семафором.
// gRPC отвечает сразу OK; все ошибки — только в лог (fail-open: задача
// останется без эмбеддинга, перегенерим на ближайшем ре-апдейте).
func (s *Service) ScheduleReindexTask(taskID int64) {
	go func() {
		s.reindexSem <- struct{}{}
		defer func() { <-s.reindexSem }()
		if err := s.reindexTaskOnce(context.Background(), taskID); err != nil {
			s.log.Warn("ai.reindex.async_failed", "task_id", taskID, "err", err)
		}
	}()
}

// reindexTaskOnce — порт reindex_task: nil и при «нечего делать»
// (задачи нет / AI выключен / пустой текст) — это не ошибки.
func (s *Service) reindexTaskOnce(ctx context.Context, taskID int64) error {
	task, err := s.repo.GetTaskText(ctx, taskID)
	if err != nil {
		return err
	}
	if task == nil || task.CompanyID == nil {
		return nil
	}
	client, err := s.clientFor(ctx, *task.CompanyID)
	if err != nil {
		return err
	}
	if client == nil {
		return nil
	}
	text := buildTaskText(task)
	if strings.TrimSpace(text) == "" {
		return nil
	}
	vecs, err := s.llm.Embed(ctx, client.apiKey, client.modelEmbedding, []string{text}, requestTimeout)
	if err != nil {
		s.log.Warn("ai.embed.failed", "task_id", taskID, "err", err)
		return nil // fail-open, как reindex_task → False
	}
	return s.repo.UpsertEmbedding(ctx, task.ID, *task.CompanyID, vecs[0], client.modelEmbedding)
}

// runBackfill — порт run_backfill: проход по непроиндексированным задачам
// компании пачками embedBatchSize. Ошибки пачек логируются и не прерывают
// остальные.
func (s *Service) runBackfill(ctx context.Context, companyID int64) {
	company, err := s.repo.GetCompanyAI(ctx, companyID)
	if err != nil || company == nil || !company.Enabled {
		if err != nil {
			s.log.Warn("ai.reindex.batch_failed", "company_id", companyID, "err", err)
		}
		return
	}
	ids, err := s.repo.FindUnindexedTaskIDs(ctx, companyID, company.EmbeddingModel())
	if err != nil {
		s.log.Warn("ai.reindex.batch_failed", "company_id", companyID, "err", err)
		return
	}
	indexed := 0
	for start := 0; start < len(ids); start += embedBatchSize {
		end := min(start+embedBatchSize, len(ids))
		indexed += s.reindexBatch(ctx, ids[start:end])
	}
	s.log.Info("ai.reindex.done", "company_id", companyID, "total", len(ids), "indexed", indexed)
}

// reindexBatch — порт reindex_tasks_batch: группировка по компании (клиент
// дёргается один раз), эмбеддинг пачкой, upsert поштучно. Возвращает число
// успешных.
func (s *Service) reindexBatch(ctx context.Context, taskIDs []int64) int {
	if len(taskIDs) == 0 {
		return 0
	}
	tasks, err := s.repo.ListTaskTexts(ctx, taskIDs)
	if err != nil {
		s.log.Warn("ai.embed_batch.failed", "err", err)
		return 0
	}
	byCompany := map[int64][]*domain.TaskText{}
	for _, t := range tasks {
		if t.CompanyID != nil {
			byCompany[*t.CompanyID] = append(byCompany[*t.CompanyID], t)
		}
	}
	okTotal := 0
	for companyID, group := range byCompany {
		client, err := s.clientFor(ctx, companyID)
		if err != nil || client == nil {
			continue
		}
		for start := 0; start < len(group); start += embedBatchSize {
			end := min(start+embedBatchSize, len(group))
			chunk := group[start:end]
			texts := make([]string, 0, len(chunk))
			for _, t := range chunk {
				texts = append(texts, buildTaskText(t))
			}
			vecs, err := s.llm.Embed(ctx, client.apiKey, client.modelEmbedding, texts, requestTimeout)
			if err != nil {
				s.log.Warn("ai.embed_batch.failed", "company_id", companyID, "err", err)
				continue
			}
			for i, t := range chunk {
				if err := s.repo.UpsertEmbedding(ctx, t.ID, companyID, vecs[i], client.modelEmbedding); err != nil {
					s.log.Warn("ai.embed_upsert.failed", "task_id", t.ID, "err", err)
					continue
				}
				okTotal++
			}
		}
	}
	return okTotal
}
