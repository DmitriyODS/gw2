package service

import (
	"context"
	"time"
)

// Фоновый цикл заботы petsvc (проверка болезней, дневной пересчёт
// характеров). Останавливается по ctx.Done().

const careTickInterval = time.Hour

// RunCareLoop — раз в час: проверка болезней и дневной пересчёт характеров.
// Работает для ВСЕХ активных компаний (болезнь не требует включённого ИИ).
func (s *Service) RunCareLoop(ctx context.Context) {
	s.log.Info("pets.care.loop_start", "interval", careTickInterval.String())
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
		s.log.Warn("pets.care.tick_failed", "error", err)
		return
	}
	for _, cid := range companyIDs {
		// Компания выключила режим «Мой Groove» — питомцы не болеют, характеры
		// не пересчитываем.
		if !s.grooveEnabled(ctx, cid) {
			continue
		}
		if _, err := s.CheckSicknessForCompany(ctx, cid); err != nil {
			s.log.Warn("pets.care.company_failed", "company_id", cid, "error", err)
		}
		// Характеры пересчитываем раз в день (метка в Redis; Redis лёг —
		// пересчитываем каждый тик, как прежний фолбэк Flask).
		key := "gw2:pets:personality:" + strconvI64(cid) + ":" + todayMSK().Format("2006-01-02")
		if !s.daily.Exists(ctx, key) {
			if err := s.RefreshPersonalitiesForCompany(ctx, cid); err != nil {
				s.log.Warn("pets.care.company_failed", "company_id", cid, "error", err)
				continue
			}
			s.daily.SetCache(ctx, key, "1", 48*time.Hour)
		}
	}
}
