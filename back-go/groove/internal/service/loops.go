package service

import (
	"context"
	"time"

	"github.com/DmitriyODS/gw2/back-go/groove/internal/domain"
)

// Фоновые циклы groovesvc (бывшие run_groove_care_loop / run_groove_ai_loop
// во Flask). Останавливаются по ctx.Done().

const (
	careTickInterval = time.Hour
	aiTickInterval   = 15 * time.Minute
)

// RunCareLoop — раз в час: проверка болезней + дневной пересчёт характеров.
// Работает для ВСЕХ активных компаний (болезнь не требует включённого ИИ).
func (s *Service) RunCareLoop(ctx context.Context) {
	s.log.Info("groove.care.loop_start", "interval", careTickInterval.String())
	ticker := time.NewTicker(careTickInterval)
	defer ticker.Stop()
	for {
		s.careTick(ctx)
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

func (s *Service) careTick(ctx context.Context) {
	companyIDs, err := s.companies.ActiveCompanyIDs(ctx)
	if err != nil {
		s.log.Warn("groove.care.tick_failed", "error", err)
		return
	}
	for _, cid := range companyIDs {
		if _, err := s.CheckSicknessForCompany(ctx, cid); err != nil {
			s.log.Warn("groove.care.company_failed", "company_id", cid, "error", err)
		}
		// Характеры пересчитываем раз в день (метка в Redis; Redis лёг —
		// пересчитываем каждый тик, как прежний фолбэк Flask).
		key := "gw2:groove:personality:" + strconvI64(cid) + ":" + todayMSK().Format("2006-01-02")
		if !s.daily.Exists(ctx, key) {
			if err := s.RefreshPersonalitiesForCompany(ctx, cid); err != nil {
				s.log.Warn("groove.care.company_failed", "company_id", cid, "error", err)
				continue
			}
			s.daily.SetCache(ctx, key, "1", 48*time.Hour)
		}
	}
}

// RunAILoop — пул реплик кормления + утренний дайджест (только компании
// с включённым ИИ).
func (s *Service) RunAILoop(ctx context.Context) {
	s.log.Info("groove.ai.loop_start", "interval", aiTickInterval.String())
	ticker := time.NewTicker(aiTickInterval)
	defer ticker.Stop()
	for {
		s.aiTick(ctx)
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

func (s *Service) aiTick(ctx context.Context) {
	companyIDs, err := s.companies.AICompanyIDs(ctx)
	if err != nil {
		s.log.Warn("groove.ai.tick_failed", "error", err)
		return
	}
	for _, cid := range companyIDs {
		// Пул фраз кормления: держим свежим всегда.
		if !s.daily.Exists(ctx, phrasesKeyPrefix+strconvI64(cid)) {
			s.refreshPhrases(ctx, cid)
		}
		// Дайджест: один раз в день после digestHourMSK.
		now := time.Now().In(domain.MSK)
		if now.Hour() >= digestHourMSK {
			key := digestKeyPrefix + strconvI64(cid) + ":" + now.Format("2006-01-02")
			if !s.daily.Exists(ctx, key) && s.generateDigest(ctx, cid) {
				s.daily.SetCache(ctx, key, "1", digestTTL)
			}
		}
	}
}
