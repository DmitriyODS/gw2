package service

import (
	"context"
	"strconv"
	"strings"

	"github.com/DmitriyODS/gw2/back-go/auth/internal/domain"
)

/* Личная компания.

   Каждый аккаунт получает собственную компанию при первом входе: к компании
   привязано всё рабочее — задачи, юниты, статистика, реестры, календари,
   портал. Без неё человек попадал в приложение, где половина разделов
   отвечает «нужна активная компания», и первым делом был обязан завести
   организацию, даже если работает один.

   Состояние «пользователь без компаний» платформа по-прежнему переживает
   (членство отзывают, компанию удаляют), поэтому проверки нигде не снимаем —
   личная компания лишь избавляет от пустого старта. */

// personalCompanySuffix — по нему компания опознаётся как личная в списках.
const personalCompanySuffix = " — личное"

// PersonalCompanyName — название личной компании: имя владельца с пометкой.
// Фамилия и имя без отчества: «Иванов Иван — личное».
func PersonalCompanyName(fio string) string {
	parts := strings.Fields(strings.TrimSpace(fio))
	if len(parts) > 2 {
		parts = parts[:2]
	}
	name := strings.Join(parts, " ")
	if name == "" {
		name = "Моя компания"
	}
	return name + personalCompanySuffix
}

/* EnsurePersonalCompany — завести личную компанию, если у человека нет ни
   одной. Ошибки не поднимаем выше: вход не должен падать из-за того, что
   компанию завести не удалось (занято имя, упёрлись в лимит тарифа) —
   человек просто окажется в привычном состоянии «без компании». */
func (s *Service) EnsurePersonalCompany(ctx context.Context, user *domain.User) {
	if user == nil {
		return
	}
	memberships, err := s.repo.ListMemberships(ctx, user.ID)
	if err != nil {
		s.log.Warn("company.personal_check_failed", "user_id", user.ID, "error", err)
		return
	}
	if len(memberships) > 0 {
		return
	}

	company := &domain.Company{
		Name:      s.freePersonalName(ctx, PersonalCompanyName(user.FIO)),
		CreatedBy: &user.ID,
		Settings:  mergeSettings(nil, nil),
	}
	if err := s.companies.CreateCompany(ctx, company); err != nil {
		s.log.Warn("company.personal_create_failed", "user_id", user.ID, "error", err)
		return
	}
	if err := s.ensureAdminMembership(ctx, user.ID, company.ID); err != nil {
		s.log.Warn("company.personal_membership_failed",
			"user_id", user.ID, "company_id", company.ID, "error", err)
		return
	}
	s.log.Info("company.personal_created", "user_id", user.ID, "company_id", company.ID)
}

// freePersonalName — свободное название: имена компаний уникальны, а тёзки
// на платформе неизбежны. Добавляем номер, пока имя занято.
func (s *Service) freePersonalName(ctx context.Context, base string) string {
	name := base
	for i := 2; i < 100; i++ {
		if err := s.ensureCompanyNameFree(ctx, name, 0); err == nil {
			return name
		}
		name = strings.TrimSuffix(base, personalCompanySuffix) +
			" " + strconv.Itoa(i) + personalCompanySuffix
	}
	return name
}
