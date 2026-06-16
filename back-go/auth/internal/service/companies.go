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

var errCompanyNotFound = domain.NewError("NOT_FOUND", "Компания не найдена", 404)

func (s *Service) ListRoles(ctx context.Context) ([]dto.Role, error) {
	roles, err := s.repo.ListRoles(ctx)
	if err != nil {
		return nil, err
	}
	return dto.NewRoles(roles), nil
}

// mergeSettings — {**DEFAULT_SETTINGS, **base, **patch}.
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

// ensureAdminMembership — гарантировать членство пользователя в компании с
// ролью администратора (создатель компании; повышение существующего члена).
func (s *Service) ensureAdminMembership(ctx context.Context, userID, companyID int64) error {
	role, err := s.repo.RoleByLevel(ctx, domain.LevelAdmin)
	if err != nil {
		return err
	}
	if role == nil {
		return domain.NewError("ROLE_NOT_FOUND", "Роль не найдена", 500)
	}
	if err := s.repo.AddMembership(ctx, userID, companyID, role.ID); err != nil {
		return err
	}
	return s.repo.UpdateMembershipRole(ctx, userID, companyID, role.ID)
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

// ListCompanies — все компании платформы (супер-админ).
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

// ListMyCompanies — компании, где actor — администратор (раздел «Компании»
// обычного пользователя). Создателя видно по полю created_by в ответе.
func (s *Service) ListMyCompanies(ctx context.Context, actor *domain.User) (*dto.CompanyList, error) {
	companies, err := s.companies.ListCompaniesWhereAdmin(ctx, actor.ID)
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

func (s *Service) GetCompany(ctx context.Context, actor *domain.User, companyID int64) (*dto.Company, error) {
	if _, err := s.companyAuthority(ctx, actor, companyID); err != nil {
		return nil, err
	}
	return s.enrichedCompany(ctx, companyID)
}

// CreateCompany — создать компанию может ЛЮБОЙ пользователь; создатель
// становится её администратором (членство уровня администратора).
func (s *Service) CreateCompany(ctx context.Context, actor *domain.User, req dto.CompanyCreate) (*dto.Company, error) {
	if err := s.ensureCompanyNameFree(ctx, req.Name, 0); err != nil {
		return nil, err
	}

	company := &domain.Company{
		Name:        req.Name,
		Description: req.Description,
		CreatedBy:   &actor.ID,
		Settings:    mergeSettings(nil, req.Settings),
	}
	if err := s.companies.CreateCompany(ctx, company); err != nil {
		return nil, err
	}
	if err := s.ensureAdminMembership(ctx, actor.ID, company.ID); err != nil {
		return nil, err
	}

	s.log.Info("company.create", "company_id", company.ID, "creator_id", actor.ID)
	return s.enrichedCompany(ctx, company.ID)
}

func (s *Service) UpdateCompany(ctx context.Context, actor *domain.User, companyID int64, req dto.CompanyUpdate) (*dto.Company, error) {
	if _, err := s.companyAuthority(ctx, actor, companyID); err != nil {
		return nil, err
	}
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

	fields := map[string]any{}
	if req.Name != nil {
		fields["name"] = *req.Name
	}
	if req.DescriptionSet {
		fields["description"] = req.Description
	}
	if req.SettingsSet {
		fields["settings"] = mergeSettings(company.Settings, req.Settings)
	}
	if len(fields) > 0 {
		if err := s.companies.UpdateCompanyFields(ctx, companyID, fields); err != nil {
			return nil, err
		}
	}
	return s.enrichedCompany(ctx, companyID)
}

// ToggleCompanyActive — включение/выключение компании (модерация платформы).
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

func (s *Service) DeleteCompany(ctx context.Context, actor *domain.User, companyID int64) error {
	if err := s.creatorAuthority(ctx, actor, companyID); err != nil {
		return err
	}
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
	s.log.Info("company.delete", "company_id", companyID, "actor_id", actor.ID)
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

func (s *Service) RegenerateInvite(ctx context.Context, actor *domain.User, companyID int64) (string, error) {
	if err := s.creatorAuthority(ctx, actor, companyID); err != nil {
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
// Сотрудник). Возвращает сессию, переключённую на эту компанию. Супер-админу
// вступать не нужно — он и так видит все компании.
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
	if user == nil || !user.IsActive {
		return nil, errUserNotFound
	}
	if user.IsSuperAdmin {
		return s.session(ctx, user, nil, true)
	}
	role, err := s.repo.RoleByLevel(ctx, domain.LevelEmployee)
	if err != nil {
		return nil, err
	}
	if role == nil {
		return nil, domain.NewError("ROLE_NOT_FOUND", "Роль не найдена", 500)
	}
	if err := s.repo.AddMembership(ctx, userID, company.ID, role.ID); err != nil {
		return nil, err
	}
	s.log.Info("company.join", "user_id", userID, "company_id", company.ID)
	return s.session(ctx, user, &company.ID, true)
}

// ── Настройки компании (администратор компании / супер-админ) ──

// weekendDays — множество выходных дней из settings.weekend_days: мусор или
// отсутствие → дефолт Сб+Вс; элементы вне 0..6 отфильтровываются.
func weekendDays(settings map[string]any) []int {
	raw, ok := settings["weekend_days"]
	if !ok {
		return append([]int(nil), domain.DefaultWeekend...)
	}
	var items []any
	switch list := raw.(type) {
	case []any:
		items = list
	case []int:
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
	if _, err := s.companyAuthority(ctx, actor, companyID); err != nil {
		return nil, err
	}
	company, err := s.companies.GetCompany(ctx, companyID)
	if err != nil {
		return nil, err
	}
	if company == nil {
		return nil, errCompanyNotFound
	}
	return &dto.WeekendSettings{WeekendDays: weekendDays(company.Settings)}, nil
}

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
	if _, err := s.companyAuthority(ctx, actor, companyID); err != nil {
		return nil, err
	}
	company, err := s.companies.GetCompany(ctx, companyID)
	if err != nil {
		return nil, err
	}
	if company == nil {
		return nil, errCompanyNotFound
	}
	return &dto.GrooveSettings{Enabled: grooveEnabled(company.Settings)}, nil
}

func (s *Service) UpdateGrooveSettings(ctx context.Context, actor *domain.User, companyID int64, enabled bool) (*dto.GrooveSettings, error) {
	if _, err := s.companyAuthority(ctx, actor, companyID); err != nil {
		return nil, err
	}
	company, err := s.companies.GetCompany(ctx, companyID)
	if err != nil {
		return nil, err
	}
	if company == nil {
		return nil, errCompanyNotFound
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
	if _, err := s.companyAuthority(ctx, actor, companyID); err != nil {
		return nil, err
	}
	company, err := s.companies.GetCompany(ctx, companyID)
	if err != nil {
		return nil, err
	}
	if company == nil {
		return nil, errCompanyNotFound
	}

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
