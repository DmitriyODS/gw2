// ТВ-факт дня для брендового слайда. Портировано из
// back/app/services/tv_facts_service.py без изменения правил.
//
// Генерируем раз в час по каждой компании с включённым AI, храним в Redis
// (gw2:ai:tv_fact:{cid}). Жанры чередуются 50/50: "general" — общий факт о
// работе/продуктивности, "context" — наблюдение по статистике компании за
// последние 7 дней. Никаких ретраев на месте: упала генерация — пропускаем
// тик, лучше показать прошлый факт, чем рисковать стабильностью цикла.
package service

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand/v2"
	"strconv"
	"strings"
	"time"

	"github.com/DmitriyODS/gw2/back-go/ai/internal/domain"
)

const (
	// tvTickInterval — раз в час хорошо балансирует свежесть и затраты
	// (1 chat-completion / час / компания ≈ копейки).
	tvTickInterval = time.Hour

	// tvFactTTL вдвое больше тика — если следующий тик пропустится, факт
	// всё ещё будет на табло, а не превратится в фолбэк.
	tvFactTTL = 2 * tvTickInterval

	tvMaxTokens   = 180
	tvTemperature = 0.9
	tvTimeout     = 20 * time.Second
)

// mskZone — МСК = UTC+3, без DST. Для красивых дат в контексте.
var mskZone = time.FixedZone("MSK", 3*60*60)

const tvSystemPrompt = "Ты — короткий и остроумный спикер на корпоративном табло. " +
	"Никаких эмодзи, кавычек и преамбул. Только сам факт, 1–2 предложения, " +
	"до 220 символов, на русском."

const tvGeneralPrompt = "Сформулируй один интересный или забавный факт про работу, " +
	"продуктивность, тайм-менеджмент или командное взаимодействие. " +
	"Без банальностей, без воды."

// tvWeekWindowMSK — окно за последние 7 дней (включая сегодня) в МСК.
// Берём неделю, а не сегодняшний день: на свежей базе или после выходного
// сегодняшние цифры часто нулевые — «контекст» вырождался в воду.
func tvWeekWindowMSK(now time.Time) (time.Time, time.Time) {
	now = now.In(mskZone)
	end := time.Date(now.Year(), now.Month(), now.Day(), 23, 59, 59, 0, mskZone)
	startDay := end.AddDate(0, 0, -6)
	start := time.Date(startDay.Year(), startDay.Month(), startDay.Day(), 0, 0, 0, 0, mskZone)
	return start, end
}

func tvContextPrompt(ctx *domain.TVWeekContext) string {
	lines := []string{
		"Сделай короткое, живое наблюдение или вывод по статистике команды " +
			"за последние 7 дней. Цифры можно округлять для красоты.",
		fmt.Sprintf("Закрыто задач за неделю: %d.", ctx.ClosedWeek),
		fmt.Sprintf("Поступило задач за неделю: %d.", ctx.ReceivedWeek),
		fmt.Sprintf("Часы команды за неделю: %s.", pyFloat(ctx.TeamHoursWeek)),
	}
	if ctx.LeaderFIO != nil {
		lines = append(lines, fmt.Sprintf("Лидер недели — %s (%s ч).",
			*ctx.LeaderFIO, pyFloat(ptrFloat(ctx.LeaderHours))))
	}
	if ctx.TopDept != nil && *ctx.TopDept != "" {
		lines = append(lines, fmt.Sprintf("Самый активный отдел — %s.", *ctx.TopDept))
	}
	return strings.Join(lines, " ")
}

// pyFloat — float в промпте как Python str(): кратчайшая запись, но с
// обязательной десятичной частью ("12.0", не "12").
func pyFloat(v float64) string {
	s := strconv.FormatFloat(v, 'f', -1, 64)
	if !strings.ContainsAny(s, ".eE") {
		s += ".0"
	}
	return s
}

func ptrFloat(v *float64) float64 {
	if v == nil {
		return 0
	}
	return *v
}

// pyISOUTC — datetime.now(timezone.utc).isoformat(): микросекунды ровно
// 6 цифр (опускаются, если нулевые), смещение +00:00.
func pyISOUTC(t time.Time) string {
	t = t.UTC()
	s := t.Format("2006-01-02T15:04:05")
	if us := t.Nanosecond() / 1000; us != 0 {
		s += fmt.Sprintf(".%06d", us)
	}
	return s + "+00:00"
}

// GetTVFact — текущий факт дня (REST GET /api/ai/tv-fact). Если AI выключен /
// факт не сгенерён / Redis лёг — nil с 200 OK: фронт молча падает на
// фолбэк-слайд.
func (s *Service) GetTVFact(ctx context.Context, companyID int64) (*domain.TVFact, error) {
	fact, err := s.facts.GetFact(ctx, companyID)
	if err != nil {
		return nil, nil // fail-open, как try/except вокруг Redis во Flask
	}
	return fact, nil
}

// GenerateTVFact — сгенерировать факт и положить в Redis. Всегда честно
// дёргает модель — кэш разруливает уровень выше (TTL + интервал цикла).
// AI у компании выключен → затираем кэш, чтобы табло сразу упало на фолбэк.
func (s *Service) GenerateTVFact(ctx context.Context, companyID int64) error {
	client, err := s.clientFor(ctx, companyID)
	if err != nil {
		return err
	}
	if client == nil {
		s.facts.DeleteFact(ctx, companyID)
		return nil
	}

	kind := "general"
	if rand.IntN(2) == 1 {
		kind = "context"
	}
	userPrompt := tvGeneralPrompt
	if kind == "context" {
		start, end := tvWeekWindowMSK(time.Now())
		wctx, err := s.repo.TVWeekContext(ctx, companyID, start, end)
		if err != nil {
			s.log.Warn("ai.tv_facts.context_failed", "company_id", companyID, "err", err)
			wctx = nil
		}
		if wctx.Meaningful() {
			userPrompt = tvContextPrompt(wctx)
		} else {
			kind = "general"
		}
	}

	messages, err := json.Marshal([]map[string]string{
		{"role": "system", "content": tvSystemPrompt},
		{"role": "user", "content": userPrompt},
	})
	if err != nil {
		return err
	}
	res, err := s.llm.ChatOnce(ctx, domain.ChatParams{
		APIKey:       client.apiKey,
		Model:        client.modelChat,
		MessagesJSON: string(messages),
		MaxTokens:    tvMaxTokens,
		Temperature:  tvTemperature,
		Timeout:      tvTimeout,
	})
	if err != nil {
		s.log.Warn("ai.tv_facts.gen_failed", "company_id", companyID, "err", err)
		return nil
	}

	text := strings.TrimSpace(strings.Trim(strings.TrimSpace(res.Content), `"«»`))
	if text == "" {
		return nil
	}
	fact := &domain.TVFact{
		GeneratedAt: pyISOUTC(time.Now()),
		Kind:        kind,
		Text:        text,
	}
	if err := s.facts.SetFact(ctx, companyID, fact, tvFactTTL); err != nil {
		s.log.Warn("ai.tv_facts.redis_set_failed", "company_id", companyID, "err", err)
	}
	return nil
}

// RunTVFactsLoop — фоновый цикл генерации (goroutine из main). При старте —
// один проход для всех компаний (вдруг Redis пуст), дальше раз в час.
func (s *Service) RunTVFactsLoop(ctx context.Context) {
	s.log.Info("ai.tv_facts.loop_start", "interval_sec", int(tvTickInterval.Seconds()))
	ticker := time.NewTicker(tvTickInterval)
	defer ticker.Stop()
	for {
		s.tvFactsTick(ctx)
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

func (s *Service) tvFactsTick(ctx context.Context) {
	companyIDs, err := s.repo.AICompanyIDs(ctx)
	if err != nil {
		s.log.Warn("ai.tv_facts.tick_failed", "err", err)
		return
	}
	for _, cid := range companyIDs {
		if err := s.GenerateTVFact(ctx, cid); err != nil {
			s.log.Warn("ai.tv_facts.iter_failed", "company_id", cid, "err", err)
		}
	}
}
