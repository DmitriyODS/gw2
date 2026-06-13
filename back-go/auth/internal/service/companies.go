package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"sort"
	"strconv"

	"github.com/DmitriyODS/gw2/back-go/auth/internal/domain"
	"github.com/DmitriyODS/gw2/back-go/auth/internal/dto"
)

// Портировано из back/app/api/companies.py, services/company_service.py и
// utils/workweek.py без изменения правил.

var errCompanyNotFound = domain.NewError("NOT_FOUND", "Компания не найдена", 404)

func (s *Service) ListRoles(ctx context.Context) ([]dto.Role, error) {
	roles, err := s.repo.ListRoles(ctx)
	if err != nil {
		return nil, err
	}
	return dto.NewRoles(roles), nil
}

// mergeSettings — {**DEFAULT_SETTINGS, **base, **patch} (как _merge_settings).
func mergeSettings(base, patch map[string]any) map[string]any {
	merged := domain.DefaultCompanySettings()
	for k, v := range base {
		merged[k] = v
	}
	for k, v := range patch {
		merged[k] = v
	}
	return merged
}

// validateDirector — корневой Руководитель должен быть существующим,
// видимым пользователем.
func (s *Service) validateDirector(ctx context.Context, directorID *int64) error {
	if directorID == nil {
		return nil
	}
	user, err := s.repo.GetByID(ctx, *directorID)
	if err != nil {
		return err
	}
	if user == nil || user.IsHidden {
		return domain.NewError("DIRECTOR_NOT_FOUND", "Руководитель не найден", 404)
	}
	return nil
}

// roleIDByLevel — id фиксированной роли по уровню (роли создавать нельзя).
func (s *Service) roleIDByLevel(ctx context.Context, level int) (int64, error) {
	roles, err := s.repo.ListRoles(ctx)
	if err != nil {
		return 0, err
	}
	for _, r := range roles {
		if r.Level == level {
			return r.ID, nil
		}
	}
	return 0, domain.NewError("ROLE_NOT_FOUND", "Роль не найдена", 500)
}

// ensureDirectorMembership — назначение корневого Руководителя компании заводит
// (или поднимает до Руководителя) его членство в этой компании: иначе человек,
// уже состоящий в другой компании, не получал доступа ко второй (не видел ни
// пикера при логине, ни переключателя). Первичная компания пересчитывается.
func (s *Service) ensureDirectorMembership(ctx context.Context, userID, companyID int64) error {
	roleID, err := s.roleIDByLevel(ctx, domain.LevelDirector)
	if err != nil {
		return err
	}
	if err := s.repo.AddMembership(ctx, userID, companyID, roleID); err != nil {
		return err
	}
	if err := s.repo.UpdateMembershipRole(ctx, userID, companyID, roleID); err != nil {
		return err
	}
	return s.repo.SyncPrimaryCompany(ctx, userID)
}

func (s *Service) ensureCompanyNameFree(ctx context.Context, name string, selfID int64) error {
	existing, err := s.companies.GetCompanyByName(ctx, name)
	if err != nil {
		return err
	}
	if existing != nil && existing.ID != selfID {
		return domain.NewError("DUPLICATE", "Компания с таким названием уже существует", 409)
	}
	return nil
}

// enrichedCompany — перечитать компанию и навесить счётчики (_enrich).
func (s *Service) enrichedCompany(ctx context.Context, companyID int64) (*dto.Company, error) {
	company, err := s.companies.GetCompany(ctx, companyID)
	if err != nil {
		return nil, err
	}
	if company == nil {
		return nil, errCompanyNotFound
	}
	stats, err := s.companies.CompanyStats(ctx, []int64{companyID})
	if err != nil {
		return nil, err
	}
	out := dto.NewCompany(company, stats[companyID])
	return &out, nil
}

func (s *Service) ListCompanies(ctx context.Context) (*dto.CompanyList, error) {
	companies, err := s.companies.ListCompanies(ctx)
	if err != nil {
		return nil, err
	}
	ids := make([]int64, 0, len(companies))
	for _, c := range companies {
		ids = append(ids, c.ID)
	}
	stats, err := s.companies.CompanyStats(ctx, ids)
	if err != nil {
		return nil, err
	}
	items := make([]dto.Company, 0, len(companies))
	for _, c := range companies {
		items = append(items, dto.NewCompany(c, stats[c.ID]))
	}
	return &dto.CompanyList{Items: items, Total: len(items)}, nil
}

func (s *Service) GetCompany(ctx context.Context, companyID int64) (*dto.Company, error) {
	return s.enrichedCompany(ctx, companyID)
}

func (s *Service) CreateCompany(ctx context.Context, req dto.CompanyCreate) (*dto.Company, error) {
	if err := s.ensureCompanyNameFree(ctx, req.Name, 0); err != nil {
		return nil, err
	}
	if err := s.validateDirector(ctx, req.DirectorID); err != nil {
		return nil, err
	}

	company := &domain.Company{
		Name:        req.Name,
		Description: req.Description,
		DirectorID:  req.DirectorID,
		Settings:    mergeSettings(nil, req.Settings),
	}
	if err := s.companies.CreateCompany(ctx, company); err != nil {
		return nil, err
	}
	if !req.IsActive {
		if err := s.companies.UpdateCompanyFields(ctx, company.ID,
			map[string]any{"is_active": false}); err != nil {
			return nil, err
		}
	}

	// Заводим членство Руководителя в созданной компании (в т.ч. для того, кто
	// уже состоит в других компаниях — так появляется многокомпанийность).
	if req.DirectorID != nil {
		if err := s.ensureDirectorMembership(ctx, *req.DirectorID, company.ID); err != nil {
			return nil, err
		}
	}

	s.log.Info("company.create", "company_id", company.ID)
	return s.enrichedCompany(ctx, company.ID)
}

func (s *Service) UpdateCompany(ctx context.Context, companyID int64, req dto.CompanyUpdate) (*dto.Company, error) {
	company, err := s.companies.GetCompany(ctx, companyID)
	if err != nil {
		return nil, err
	}
	if company == nil {
		return nil, errCompanyNotFound
	}

	if req.Name != nil && *req.Name != company.Name {
		if err := s.ensureCompanyNameFree(ctx, *req.Name, companyID); err != nil {
			return nil, err
		}
	}
	if req.DirectorSet {
		if err := s.validateDirector(ctx, req.DirectorID); err != nil {
			return nil, err
		}
	}

	fields := map[string]any{}
	if req.Name != nil {
		fields["name"] = *req.Name
	}
	if req.DescriptionSet {
		fields["description"] = req.Description
	}
	if req.DirectorSet {
		fields["director_id"] = req.DirectorID
	}
	if req.IsActive != nil {
		fields["is_active"] = *req.IsActive
	}
	if req.SettingsSet {
		fields["settings"] = mergeSettings(company.Settings, req.Settings)
	}
	if len(fields) > 0 {
		if err := s.companies.UpdateCompanyFields(ctx, companyID, fields); err != nil {
			return nil, err
		}
	}
	// Назначение нового Руководителя — завести/поднять его членство в компании.
	if req.DirectorSet && req.DirectorID != nil {
		if err := s.ensureDirectorMembership(ctx, *req.DirectorID, companyID); err != nil {
			return nil, err
		}
	}
	return s.enrichedCompany(ctx, companyID)
}

func (s *Service) ToggleCompanyActive(ctx context.Context, companyID int64, isActive bool) (*dto.Company, error) {
	company, err := s.companies.GetCompany(ctx, companyID)
	if err != nil {
		return nil, err
	}
	if company == nil {
		return nil, errCompanyNotFound
	}
	if err := s.companies.UpdateCompanyFields(ctx, companyID,
		map[string]any{"is_active": isActive}); err != nil {
		return nil, err
	}
	return s.enrichedCompany(ctx, companyID)
}

func (s *Service) DeleteCompany(ctx context.Context, companyID int64) error {
	company, err := s.companies.GetCompany(ctx, companyID)
	if err != nil {
		return err
	}
	if company == nil {
		return errCompanyNotFound
	}
	if err := s.companies.DeleteCompany(ctx, companyID); err != nil {
		return err
	}
	s.log.Info("company.delete", "company_id", companyID)
	return nil
}

// ── Ссылка-приглашение и вступление по ней ──

func randomInviteCode() (string, error) {
	b := make([]byte, 12)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// CompanyInvite — текущий код-приглашение (Админ системы или Руководитель этой
// компании); пустая строка — приглашение ещё не выдано.
func (s *Service) CompanyInvite(ctx context.Context, actor *domain.User, companyID int64) (string, error) {
	if _, err := s.companyAuthority(ctx, actor, companyID); err != nil {
		return "", err
	}
	company, err := s.companies.GetCompany(ctx, companyID)
	if err != nil {
		return "", err
	}
	if company == nil {
		return "", errCompanyNotFound
	}
	if company.InviteCode == nil {
		return "", nil
	}
	return *company.InviteCode, nil
}

// RegenerateInvite — выдать/перевыпустить код (старая ссылка перестаёт работать).
func (s *Service) RegenerateInvite(ctx context.Context, actor *domain.User, companyID int64) (string, error) {
	if _, err := s.companyAuthority(ctx, actor, companyID); err != nil {
		return "", err
	}
	company, err := s.companies.GetCompany(ctx, companyID)
	if err != nil {
		return "", err
	}
	if company == nil {
		return "", errCompanyNotFound
	}
	code, err := randomInviteCode()
	if err != nil {
		return "", err
	}
	if err := s.companies.UpdateCompanyFields(ctx, companyID, map[string]any{"invite_code": code}); err != nil {
		return "", err
	}
	return code, nil
}

// JoinByCode — авторизованный пользователь вступает в компанию по ссылке (роль
// Сотрудник). Возвращает сессию, переключённую на эту компанию. Администратору
// системы вступать не нужно — у него доступ ко всем компаниям.
func (s *Service) JoinByCode(ctx context.Context, userID int64, code string) (*dto.Session, error) {
	company, err := s.companies.GetCompanyByInviteCode(ctx, code)
	if err != nil {
		return nil, err
	}
	if company == nil {
		return nil, domain.NewError("INVALID_INVITE", "Ссылка-приглашение недействительна", 404)
	}
	if !company.IsActive {
		return nil, companyDisabledErr(&company.Name)
	}
	user, err := s.repo.GetByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if user == nil || user.IsHidden {
		return nil, errUserNotFound
	}
	if isSystemAdmin(user) {
		return s.session(ctx, user, nil, true)
	}
	roleID, err := s.roleIDByLevel(ctx, domain.LevelEmployee)
	if err != nil {
		return nil, err
	}
	if err := s.repo.AddMembership(ctx, userID, company.ID, roleID); err != nil {
		return nil, err
	}
	if err := s.repo.SyncPrimaryCompany(ctx, userID); err != nil {
		return nil, err
	}
	s.log.Info("company.join", "user_id", userID, "company_id", company.ID)
	return s.session(ctx, user, &company.ID, true)
}

// ── Выходные дни (Руководитель своей компании / Администратор системы) ──

// checkCompanyAccess — как _check_company_access во Flask: доступ по
// is_root_admin или собственной компании (Админ без is_root_admin и без
// компании доступа НЕ имеет — поведение сохранено).
func checkCompanyAccess(actor *domain.User, companyID int64) error {
	if actor != nil && actor.IsRootAdmin {
		return nil
	}
	if actor != nil && actor.CompanyID != nil && *actor.CompanyID == companyID {
		return nil
	}
	return domain.NewError("FORBIDDEN", "Нет доступа к настройкам этой компании", 403)
}

// weekendDays — множество выходных дней из settings.weekend_days: мусор или
// отсутствие → дефолт Сб+Вс; элементы вне 0..6 отфильтровываются
// (utils/workweek.weekend_days).
func weekendDays(settings map[string]any) []int {
	raw, ok := settings["weekend_days"]
	if !ok {
		return append([]int(nil), domain.DefaultWeekend...)
	}
	var items []any
	switch list := raw.(type) {
	case []any: // JSONB из БД
		items = list
	case []int: // только что записанные настройки (свой же PUT)
		items = make([]any, len(list))
		for i, d := range list {
			items[i] = d
		}
	default:
		return append([]int(nil), domain.DefaultWeekend...)
	}
	seen := map[int]bool{}
	for _, item := range items {
		d, ok := toInt(item)
		if !ok {
			return append([]int(nil), domain.DefaultWeekend...)
		}
		if d >= 0 && d <= 6 {
			seen[d] = true
		}
	}
	out := make([]int, 0, len(seen))
	for d := range seen {
		out = append(out, d)
	}
	sort.Ints(out)
	return out
}

// toInt — как int(d) в Python: число (float усекается), строка-число.
func toInt(v any) (int, bool) {
	switch x := v.(type) {
	case float64:
		return int(x), true
	case int:
		return x, true
	case int64:
		return int(x), true
	case bool:
		if x {
			return 1, true
		}
		return 0, true
	case string:
		n, err := strconv.Atoi(x)
		if err != nil {
			return 0, false
		}
		return n, true
	default:
		return 0, false
	}
}

func (s *Service) GetWeekendSettings(ctx context.Context, actor *domain.User, companyID int64) (*dto.WeekendSettings, error) {
	company, err := s.companies.GetCompany(ctx, companyID)
	if err != nil {
		return nil, err
	}
	if company == nil {
		return nil, errCompanyNotFound
	}
	if err := checkCompanyAccess(actor, company.ID); err != nil {
		return nil, err
	}
	return &dto.WeekendSettings{WeekendDays: weekendDays(company.Settings)}, nil
}

// ── Режим «Мой Groove» (Руководитель своей компании / Администратор системы) ──

// grooveEnabled — режим «Мой Groove» из settings.uses_groove: отсутствие или
// мусор → включён (как и на фронте, uses_groove !== false).
func grooveEnabled(settings map[string]any) bool {
	v, ok := settings["uses_groove"]
	if !ok {
		return true
	}
	b, ok := v.(bool)
	if !ok {
		return true
	}
	return b
}

func (s *Service) GetGrooveSettings(ctx context.Context, actor *domain.User, companyID int64) (*dto.GrooveSettings, error) {
	company, err := s.companies.GetCompany(ctx, companyID)
	if err != nil {
		return nil, err
	}
	if company == nil {
		return nil, errCompanyNotFound
	}
	if err := checkCompanyAccess(actor, company.ID); err != nil {
		return nil, err
	}
	return &dto.GrooveSettings{Enabled: grooveEnabled(company.Settings)}, nil
}

func (s *Service) UpdateGrooveSettings(ctx context.Context, actor *domain.User, companyID int64, enabled bool) (*dto.GrooveSettings, error) {
	company, err := s.companies.GetCompany(ctx, companyID)
	if err != nil {
		return nil, err
	}
	if company == nil {
		return nil, errCompanyNotFound
	}
	if err := checkCompanyAccess(actor, company.ID); err != nil {
		return nil, err
	}

	settings := map[string]any{}
	for k, v := range company.Settings {
		settings[k] = v
	}
	settings["uses_groove"] = enabled
	if err := s.companies.UpdateCompanyFields(ctx, companyID,
		map[string]any{"settings": settings}); err != nil {
		return nil, err
	}
	return &dto.GrooveSettings{Enabled: enabled}, nil
}

func (s *Service) UpdateWeekendSettings(ctx context.Context, actor *domain.User, companyID int64, days []int) (*dto.WeekendSettings, error) {
	company, err := s.companies.GetCompany(ctx, companyID)
	if err != nil {
		return nil, err
	}
	if company == nil {
		return nil, errCompanyNotFound
	}
	if err := checkCompanyAccess(actor, company.ID); err != nil {
		return nil, err
	}

	// sorted(set(days)) — уникальные по возрастанию.
	seen := map[int]bool{}
	for _, d := range days {
		seen[d] = true
	}
	sorted := make([]int, 0, len(seen))
	for d := range seen {
		sorted = append(sorted, d)
	}
	sort.Ints(sorted)

	settings := map[string]any{}
	for k, v := range company.Settings {
		settings[k] = v
	}
	settings["weekend_days"] = sorted
	if err := s.companies.UpdateCompanyFields(ctx, companyID,
		map[string]any{"settings": settings}); err != nil {
		return nil, err
	}
	return &dto.WeekendSettings{WeekendDays: sorted}, nil
}
