package service

import (
	"context"
	"fmt"
	"strings"

	"github.com/xuri/excelize/v2"

	"github.com/DmitriyODS/gw2/back-go/forms/internal/domain"
)

// ExportResponses — xlsx со всеми собранными ответами: строка на ответ,
// колонка на вопрос. Возвращает байты и название формы (для имени файла).
func (s *Service) ExportResponses(ctx context.Context, userID, formID int64) ([]byte, string, error) {
	a, err := s.actor(ctx, userID)
	if err != nil {
		return nil, "", err
	}
	form, err := s.require(ctx, a, formID, domain.AccessView)
	if err != nil {
		return nil, "", err
	}
	sections, err := s.repo.ListSections(ctx, formID)
	if err != nil {
		return nil, "", err
	}

	// Пояснительные блоки в выгрузку не идут: ответа у них нет, а пустая
	// колонка в отчёте только мешает.
	questions := make([]domain.Question, 0)
	for _, q := range domain.AllQuestions(sections) {
		if domain.Answerable(q.Type) {
			questions = append(questions, q)
		}
	}

	f := excelize.NewFile()
	defer f.Close()
	const sheet = "Ответы"
	f.SetSheetName(f.GetSheetName(0), sheet)

	header := []string{"Отправлен", "Кто отвечал"}
	if form.CollectEmail {
		header = append(header, "Почта")
	}
	for _, q := range questions {
		title := strings.TrimSpace(q.Title)
		if title == "" {
			title = "Вопрос " + domain.QuestionID(q.ID)
		}
		header = append(header, title)
	}
	if form.Quiz {
		header = append(header, "Балл")
	}
	for i, title := range header {
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		f.SetCellStr(sheet, cell, title)
	}

	row := 1
	if err := s.repo.EachResponse(ctx, formID, func(r *domain.Response) error {
		row++
		line := []string{r.CreatedAt.Format("02.01.2006 15:04"), respondent(r)}
		if form.CollectEmail {
			line = append(line, r.Email)
		}
		for _, q := range questions {
			line = append(line, domain.AnswerText(q, r.Answers[domain.QuestionID(q.ID)]))
		}
		if form.Quiz {
			line = append(line, fmt.Sprintf("%d / %d", r.Score, r.MaxScore))
		}
		for i, value := range line {
			cell, _ := excelize.CoordinatesToCellName(i+1, row)
			f.SetCellStr(sheet, cell, value)
		}
		return nil
	}); err != nil {
		return nil, "", err
	}

	buf, err := f.WriteToBuffer()
	if err != nil {
		return nil, "", err
	}
	return buf.Bytes(), form.Title, nil
}

// respondent — кто отвечал: имя из аккаунта, представившийся гость либо
// «Аноним» (форма могла принимать ответы без входа).
func respondent(r *domain.Response) string {
	if name := strings.TrimSpace(r.UserName); name != "" {
		return name
	}
	if name := strings.TrimSpace(r.Name); name != "" {
		return name
	}
	return "Аноним"
}
