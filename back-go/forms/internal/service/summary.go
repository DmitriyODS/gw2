package service

import (
	"context"
	"sort"
	"strconv"
	"time"

	"github.com/DmitriyODS/gw2/back-go/forms/internal/domain"
)

/* Сводка ответов.

   Считается ПОТОКОМ по всем ответам формы (repo.EachResponse): держать в памяти
   тысячи ответов ради нескольких счётчиков незачем. Каждый тип вопроса сводится
   к тому, что о нём вообще имеет смысл сказать: выбор — распределение по
   вариантам, шкала и оценка — среднее и распределение, сетка — матрица,
   текст — последние ответы, файлы — их число. */

// maxTextSamples — сколько текстовых ответов показывает сводка (остальные
// смотрят в таблице ответов и выгрузке).
const maxTextSamples = 30

// timelineDays — глубина графика «ответы по дням».
const timelineDays = 30

type Summary struct {
	Total int `json:"total"`
	// Quiz — сводка теста (nil, если форма не тест).
	Quiz      *QuizSummary      `json:"quiz,omitempty"`
	Timeline  []DayCount        `json:"timeline"`
	Questions []QuestionSummary `json:"questions"`
}

type QuizSummary struct {
	MaxScore    int     `json:"max_score"`
	AverageScore float64 `json:"average_score"`
	// Distribution — сколько ответов набрало столько-то баллов (по возрастанию).
	Distribution []ScoreBucket `json:"distribution"`
}

type ScoreBucket struct {
	Score int `json:"score"`
	Count int `json:"count"`
}

type DayCount struct {
	Date  string `json:"date"` // ГГГГ-ММ-ДД
	Count int    `json:"count"`
}

type QuestionSummary struct {
	QuestionID int64  `json:"question_id"`
	Type       string `json:"type"`
	Title      string `json:"title"`
	Answered   int    `json:"answered"`
	// Options — распределение по вариантам (выбор, шкала, оценка).
	Options []OptionCount `json:"options,omitempty"`
	// Rows — матрица сетки: строка и распределение по столбцам.
	Rows []GridRowSummary `json:"rows,omitempty"`
	// Average — среднее (шкала, оценка).
	Average float64 `json:"average,omitempty"`
	// Texts — примеры свободных ответов.
	Texts []string `json:"texts,omitempty"`
	// Files — сколько файлов приложено.
	Files int `json:"files,omitempty"`
}

type OptionCount struct {
	Label string `json:"label"`
	Count int    `json:"count"`
	// Other — вариант, которого нет в списке (человек вписал свой).
	Other bool `json:"other,omitempty"`
}

type GridRowSummary struct {
	Row     string        `json:"row"`
	Options []OptionCount `json:"options"`
}

// Summary — сводка ответов формы (уровень view и выше).
func (s *Service) Summary(ctx context.Context, userID, formID int64) (*Summary, error) {
	a, err := s.actor(ctx, userID)
	if err != nil {
		return nil, err
	}
	form, err := s.require(ctx, a, formID, domain.AccessView)
	if err != nil {
		return nil, err
	}
	return s.buildSummary(ctx, form)
}

func (s *Service) buildSummary(ctx context.Context, form *domain.Form) (*Summary, error) {
	sections, err := s.repo.ListSections(ctx, form.ID)
	if err != nil {
		return nil, err
	}
	questions := domain.AllQuestions(sections)

	aggs := make([]*aggregate, 0, len(questions))
	byID := make(map[string]*aggregate, len(questions))
	for _, q := range questions {
		if !domain.Answerable(q.Type) {
			continue
		}
		agg := newAggregate(q)
		aggs = append(aggs, agg)
		byID[domain.QuestionID(q.ID)] = agg
	}

	out := &Summary{}
	days := map[string]int{}
	scores := map[int]int{}
	scoreSum := 0

	if err := s.repo.EachResponse(ctx, form.ID, func(r *domain.Response) error {
		out.Total++
		days[r.CreatedAt.Format("2006-01-02")]++
		if form.Quiz {
			scores[r.Score]++
			scoreSum += r.Score
		}
		for key, v := range r.Answers {
			if agg, ok := byID[key]; ok {
				agg.add(v)
			}
		}
		return nil
	}); err != nil {
		return nil, err
	}

	out.Timeline = timeline(days)
	out.Questions = make([]QuestionSummary, 0, len(aggs))
	for _, agg := range aggs {
		out.Questions = append(out.Questions, agg.result())
	}
	if form.Quiz {
		out.Quiz = quizSummary(questions, scores, scoreSum, out.Total)
	}
	return out, nil
}

func quizSummary(questions []domain.Question, scores map[int]int, sum, total int) *QuizSummary {
	q := &QuizSummary{MaxScore: domain.MaxScore(questions)}
	if total > 0 {
		q.AverageScore = float64(sum) / float64(total)
	}
	q.Distribution = make([]ScoreBucket, 0, len(scores))
	for score, count := range scores {
		q.Distribution = append(q.Distribution, ScoreBucket{Score: score, Count: count})
	}
	sort.Slice(q.Distribution, func(i, j int) bool {
		return q.Distribution[i].Score < q.Distribution[j].Score
	})
	return q
}

// timeline — ответы по дням за последние timelineDays суток; дни без ответов
// остаются в ряду нулями, иначе график врал бы про темп.
func timeline(days map[string]int) []DayCount {
	out := make([]DayCount, 0, timelineDays)
	today := time.Now()
	for i := timelineDays - 1; i >= 0; i-- {
		date := today.AddDate(0, 0, -i).Format("2006-01-02")
		out = append(out, DayCount{Date: date, Count: days[date]})
	}
	return out
}

// ── Накопитель по одному вопросу ─────────────────────────────────

type aggregate struct {
	q        domain.Question
	answered int
	counts   map[string]int
	// grid — счётчики по строкам сетки.
	grid  map[string]map[string]int
	sum   float64
	texts []string
	files int
}

func newAggregate(q domain.Question) *aggregate {
	return &aggregate{q: q, counts: map[string]int{}, grid: map[string]map[string]int{}}
}

func (a *aggregate) add(v any) {
	if !domain.Filled(v) {
		return
	}
	a.answered++
	switch a.q.Type {
	case domain.QRadio, domain.QDropdown:
		a.counts[domain.Text(v)]++
	case domain.QCheckbox:
		for _, item := range domain.List(v) {
			a.counts[item]++
		}
	case domain.QScale, domain.QRating:
		if n, ok := domain.Number(v); ok {
			a.sum += n
			a.counts[domain.Text(v)]++
		}
	case domain.QGridRadio, domain.QGridCheckbox:
		raw, _ := v.(map[string]any)
		for row, value := range raw {
			cells, ok := a.grid[row]
			if !ok {
				cells = map[string]int{}
				a.grid[row] = cells
			}
			if a.q.Type == domain.QGridRadio {
				cells[domain.Text(value)]++
				continue
			}
			for _, col := range domain.List(value) {
				cells[col]++
			}
		}
	case domain.QFile:
		list, _ := v.([]any)
		a.files += len(list)
	default:
		if len(a.texts) < maxTextSamples {
			a.texts = append(a.texts, domain.AnswerText(a.q, v))
		}
	}
}

func (a *aggregate) result() QuestionSummary {
	out := QuestionSummary{
		QuestionID: a.q.ID, Type: a.q.Type, Title: a.q.Title, Answered: a.answered,
		Texts: a.texts, Files: a.files,
	}
	switch {
	case domain.Choice(a.q.Type):
		out.Options = ordered(a.counts, a.q.Options())
	case a.q.Type == domain.QScale || a.q.Type == domain.QRating:
		out.Options = ordered(a.counts, scaleLabels(a.q))
		if a.answered > 0 {
			out.Average = a.sum / float64(a.answered)
		}
	case domain.Grid(a.q.Type):
		cols := a.q.GridCols()
		for _, row := range a.q.GridRows() {
			cells, ok := a.grid[row]
			if !ok {
				cells = map[string]int{}
			}
			out.Rows = append(out.Rows, GridRowSummary{Row: row, Options: ordered(cells, cols)})
		}
	}
	return out
}

// ordered — счётчики в порядке вариантов вопроса; вписанное человеком («Другое»)
// идёт следом, от частого к редкому.
func ordered(counts map[string]int, known []string) []OptionCount {
	out := make([]OptionCount, 0, len(counts))
	seen := map[string]bool{}
	for _, label := range known {
		seen[label] = true
		out = append(out, OptionCount{Label: label, Count: counts[label]})
	}
	extra := make([]OptionCount, 0)
	for label, count := range counts {
		if !seen[label] {
			extra = append(extra, OptionCount{Label: label, Count: count, Other: true})
		}
	}
	sort.Slice(extra, func(i, j int) bool {
		if extra[i].Count != extra[j].Count {
			return extra[i].Count > extra[j].Count
		}
		return extra[i].Label < extra[j].Label
	})
	return append(out, extra...)
}

// scaleLabels — деления шкалы или оценки как подписи вариантов.
func scaleLabels(q domain.Question) []string {
	min, max := 1, q.RatingMax()
	if q.Type == domain.QScale {
		min, max = q.Scale()
	}
	out := make([]string, 0, max-min+1)
	for i := min; i <= max; i++ {
		out = append(out, strconv.Itoa(i))
	}
	return out
}
