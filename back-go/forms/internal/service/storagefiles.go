package service

import (
	"context"
	"strconv"

	"github.com/DmitriyODS/gw2/back-go/forms/internal/domain"
	"github.com/DmitriyODS/gw2/back-go/pkg/storagefiles"
)

/* Раздел «Настройки → Хранилище»: биллинг спрашивает владельца файлов, что у
   него ещё живо, и просит удалить выбранное.

   Файлы форм лежат значениями внутри ответов. Платит за них владелец формы
   (или создатель её компании — companyIDs присылает сам биллинг), поэтому
   ищем и по человеку, и по компаниям. Удаление очищает файловый вопрос: сам
   ответ со всеми остальными значениями остаётся. */

func (s *Service) ListStorageFiles(ctx context.Context, userID int64, companyIDs []int64) ([]storagefiles.File, error) {
	scopes, err := s.repo.ResponsesOfOwner(ctx, userID, companyIDs)
	if err != nil {
		return nil, err
	}
	out := []storagefiles.File{}
	for _, sc := range scopes {
		for _, f := range domain.AnswerFiles(sc.Response.Answers) {
			out = append(out, storagefiles.File{
				Key: f.Path, Name: f.Name, Kind: "response",
				ID:        strconv.FormatInt(sc.Response.ID, 10),
				Title:     "Форма: " + sc.FormTitle,
				CompanyID: sc.CompanyID, CreatedAt: sc.Response.CreatedAt,
			})
			if f.Thumb != "" {
				out = append(out, storagefiles.File{
					Key: f.Thumb, Name: f.Name, Kind: "response",
					ID:        strconv.FormatInt(sc.Response.ID, 10),
					Title:     "Форма: " + sc.FormTitle,
					CompanyID: sc.CompanyID, CreatedAt: sc.Response.CreatedAt,
				})
			}
		}
	}
	return out, nil
}

func (s *Service) DeleteStorageFiles(ctx context.Context, userID int64, companyIDs []int64, keys []string) ([]string, error) {
	scopes, err := s.repo.ResponsesOfOwner(ctx, userID, companyIDs)
	if err != nil {
		return nil, err
	}
	drop := make(map[string]bool, len(keys))
	for _, k := range keys {
		drop[k] = true
	}

	deleted := []string{}
	questionsByForm := map[int64][]domain.Question{}
	for _, sc := range scopes {
		answers, changed, removed := domain.AnswersWithoutFiles(sc.Response.Answers, drop)
		if !changed {
			continue
		}
		questions, ok := questionsByForm[sc.FormID]
		if !ok {
			sections, err := s.repo.ListSections(ctx, sc.FormID)
			if err != nil {
				return nil, err
			}
			questions = domain.AllQuestions(sections)
			questionsByForm[sc.FormID] = questions
		}
		sc.Response.Answers = answers
		if err := s.repo.UpdateResponse(ctx, sc.Response, domain.SearchText(questions, answers), nil); err != nil {
			return nil, err
		}
		deleted = append(deleted, removed...)
		s.publish(ctx, sc.FormID, "response:updated", map[string]any{
			"form_id": sc.FormID, "response_id": sc.Response.ID,
		})
	}
	if len(deleted) > 0 {
		s.files.Remove(deleted)
	}
	return deleted, nil
}

var _ storagefiles.Owner = (*Service)(nil)
