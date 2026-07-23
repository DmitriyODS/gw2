package service

import (
	"context"
	"strings"
	"unicode/utf8"

	"github.com/DmitriyODS/gw2/back-go/auth/internal/domain"
	"github.com/DmitriyODS/gw2/back-go/auth/internal/dto"
)

var errUserNotFound = domain.NewError("NOT_FOUND", "Пользователь не найден", 404)

// nilIfEmpty — пустая строка → NULL (снятие значения nullable-колонки).
func nilIfEmpty(s string) any {
	if s == "" {
		return nil
	}
	return s
}

// errCompanyScopeRequired — операция требует активной компании (выбирается при
// login/switch и кладётся в токен).
var errCompanyScopeRequired = domain.NewError("COMPANY_SCOPE_REQUIRED", "Требуется активная компания", 400)

var errSuperAdminProtected = domain.NewError("SUPER_ADMIN", "Супер-администратора нельзя изменять", 422)

// ListUsers — все пользователи платформы (список супер-админа).
func (s *Service) ListUsers(ctx context.Context) ([]dto.User, error) {
	users, err := s.repo.ListAll(ctx)
	if err != nil {
		return nil, err
	}
	return dto.NewUsers(users), nil
}

func (s *Service) ensureLoginFree(ctx context.Context, login string, selfID int64) error {
	existing, err := s.repo.GetByLogin(ctx, login)
	if err != nil {
		return err
	}
	if existing != nil && existing.ID != selfID {
		return domain.NewError("LOGIN_TAKEN", "Логин уже занят", 409)
	}
	return nil
}

func (s *Service) ensureEmailFree(ctx context.Context, email string, selfID int64) error {
	existing, err := s.repo.GetByEmail(ctx, email)
	if err != nil {
		return err
	}
	if existing != nil && existing.ID != selfID {
		return domain.NewError("EMAIL_TAKEN", "Email уже используется", 409)
	}
	return nil
}

// actorScope — активная компания актора (из токена); ошибка, если её нет.
func actorScope(actor *domain.User) (int64, error) {
	if actor == nil || actor.CompanyID == nil {
		return 0, errCompanyScopeRequired
	}
	return *actor.CompanyID, nil
}

// CreateUser — администратор компании заводит сотрудника в СВОЕЙ активной
// компании (идентичность + членство с ролью и должностью в этой компании).
func (s *Service) CreateUser(ctx context.Context, actor *domain.User, req dto.CreateUserRequest) (*dto.User, error) {
	companyID, err := actorScope(actor)
	if err != nil {
		return nil, err
	}
	if err := validateFIO(req.FIO); err != nil {
		return nil, err
	}
	if err := validateLogin(req.Login); err != nil {
		return nil, err
	}
	if err := validatePost(req.Post); err != nil {
		return nil, err
	}
	phone, err := normalizePhone(req.Phone)
	if err != nil {
		return nil, err
	}
	email, err := normalizeEmail(req.Email)
	if err != nil {
		return nil, err
	}
	if req.Password != nil {
		if err := validatePassword(*req.Password); err != nil {
			return nil, err
		}
	}

	role, err := s.validMemberRole(ctx, req.RoleID)
	if err != nil {
		return nil, err
	}
	if role.Level > actor.Level() {
		return nil, domain.NewError("ROLE_LEVEL_FORBIDDEN", "Нельзя назначить роль выше собственной", 403)
	}

	if err := s.ensureLoginFree(ctx, req.Login, 0); err != nil {
		return nil, err
	}
	if email != nil {
		if err := s.ensureEmailFree(ctx, *email, 0); err != nil {
			return nil, err
		}
	}

	// Без явного пароля — дефолтный <login>123 и обязательная смена при входе.
	password := req.Login + "123"
	isDefault := true
	if req.Password != nil {
		password = *req.Password
		isDefault = false
	}
	hashed, err := s.repo.HashPassword(ctx, password)
	if err != nil {
		return nil, err
	}

	user := &domain.User{
		FIO:           req.FIO,
		Login:         req.Login,
		HashPassword:  hashed,
		Phone:         phone,
		Email:         email,
		IsDefaultPass: isDefault,
		// Сотрудник заведён администратором — email не подтверждает (вместо
		// верификации обязательная смена пароля при входе при дефолтном пароле).
		EmailVerified: true,
	}
	if err := s.repo.Create(ctx, user); err != nil {
		return nil, err
	}
	if err := s.repo.AddMembership(ctx, user.ID, companyID, role.ID); err != nil {
		return nil, err
	}
	if req.Post != nil {
		if err := s.repo.SetMembershipPost(ctx, user.ID, companyID, req.Post); err != nil {
			return nil, err
		}
	}

	s.log.Info("user.create", "user_id", user.ID, "company_id", companyID, "actor_id", actor.ID)
	return s.freshMemberUser(ctx, companyID, user.ID)
}

// Directory — каталог. Со активной компанией — её члены (роль/должность из
// связки); без активной компании — глобальный поиск всех (контакты).
func (s *Service) Directory(ctx context.Context, req dto.DirectoryRequest) ([]dto.DirectoryUser, error) {
	if req.CompanyID == nil {
		// Поиск строго по логину с пустым запросом — пусто (не вываливаем весь
		// каталог по ФИО; собеседника ищут по конкретному логину).
		if req.LoginOnly && strings.TrimSpace(req.Query) == "" {
			return []dto.DirectoryUser{}, nil
		}
		users, err := s.repo.SearchDirectory(ctx, req.Query, req.ExcludeID, req.LoginOnly)
		if err != nil {
			return nil, err
		}
		return dto.NewDirectoryUsers(users), nil
	}
	users, err := s.repo.SearchDirectoryMembers(ctx, req.Query, req.ExcludeID, *req.CompanyID)
	if err != nil {
		return nil, err
	}
	return dto.NewDirectoryUsers(users), nil
}

func (s *Service) DirectoryUser(ctx context.Context, actor *domain.User, userID int64) (*dto.DirectoryUser, error) {
	user, err := s.repo.GetByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if user == nil || !user.IsActive {
		return nil, domain.NewError("NOT_FOUND", "Пользователь не найден", 404)
	}
	out := dto.NewDirectoryUser(user)
	// Телефон и email — только для себя и коллег по общей компании; иначе это
	// PII любого пользователя платформы по числовому id.
	if actor == nil || (actor.ID != userID) {
		shares := false
		if actor != nil {
			if shares, err = s.repo.SharesCompany(ctx, actor.ID, userID); err != nil {
				return nil, err
			}
		}
		if !shares {
			out.Phone = nil
			out.Email = nil
		}
	}
	return &out, nil
}

func (s *Service) Me(ctx context.Context, userID int64) (*dto.User, error) {
	return s.freshUser(ctx, userID)
}

func (s *Service) freshUser(ctx context.Context, userID int64) (*dto.User, error) {
	user, err := s.repo.GetByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, errUserNotFound
	}
	out := dto.NewUser(user)
	return &out, nil
}

// freshMemberUser — профиль пользователя с контекстом его роли/должности в
// конкретной компании (ответ операций управления членом компании).
func (s *Service) freshMemberUser(ctx context.Context, companyID, userID int64) (*dto.User, error) {
	user, err := s.repo.GetByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, errUserNotFound
	}
	m, err := s.repo.GetMembership(ctx, userID, companyID)
	if err != nil {
		return nil, err
	}
	if m != nil {
		user.CompanyID = &companyID
		user.Role = m.Role
		user.Post = m.Post
	}
	out := dto.NewUser(user)
	return &out, nil
}

func (s *Service) UpdateMe(ctx context.Context, userID int64, req dto.UpdateMeRequest) (*dto.User, error) {
	user, err := s.repo.GetByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, errUserNotFound
	}

	updates := map[string]any{}

	if req.FIO != nil {
		if err := validateFIO(*req.FIO); err != nil {
			return nil, err
		}
		updates["fio"] = *req.FIO
	}
	if req.Phone != nil {
		phone, err := normalizePhone(req.Phone)
		if err != nil {
			return nil, err
		}
		updates["phone"] = phone
	}
	if req.Email != nil {
		email, err := normalizeEmail(req.Email)
		if err != nil {
			return nil, err
		}
		if email != nil {
			if err := s.ensureEmailFree(ctx, *email, userID); err != nil {
				return nil, err
			}
		}
		updates["email"] = email
	}
	if req.Login != nil {
		if err := validateLogin(*req.Login); err != nil {
			return nil, err
		}
		if err := s.ensureLoginFree(ctx, *req.Login, userID); err != nil {
			return nil, err
		}
		updates["login"] = *req.Login
	}

	if req.StatusEmoji != nil {
		updates["status_emoji"] = nilIfEmpty(strings.TrimSpace(*req.StatusEmoji))
	}
	if req.StatusText != nil {
		text := strings.TrimSpace(*req.StatusText)
		if utf8.RuneCountInString(text) > 80 {
			return nil, domain.NewError("VALIDATION", "Статус не длиннее 80 символов", 400)
		}
		updates["status_text"] = nilIfEmpty(text)
	}

	if req.OnVacation != nil {
		updates["on_vacation"] = *req.OnVacation
	}

	if req.NewPassword != nil && *req.NewPassword != "" {
		if req.CurrentPassword == nil || *req.CurrentPassword == "" {
			return nil, domain.NewError("CURRENT_PASSWORD_REQUIRED", "Введите текущий пароль", 400)
		}
		ok, err := s.repo.VerifyPassword(ctx, *req.CurrentPassword, user.HashPassword)
		if err != nil {
			return nil, err
		}
		if !ok {
			return nil, domain.NewError("WRONG_PASSWORD", "Неверный текущий пароль", 400)
		}
		if err := validatePassword(*req.NewPassword); err != nil {
			return nil, err
		}
		if req.ConfirmPassword == nil || *req.NewPassword != *req.ConfirmPassword {
			return nil, domain.NewError("PASSWORDS_MISMATCH", "Пароли не совпадают", 400)
		}
		hashed, err := s.repo.HashPassword(ctx, *req.NewPassword)
		if err != nil {
			return nil, err
		}
		updates["hash_password"] = hashed
	}

	if len(updates) > 0 {
		if err := s.repo.UpdateFields(ctx, userID, updates); err != nil {
			return nil, err
		}
	}
	return s.freshUser(ctx, userID)
}

func (s *Service) UploadAvatar(ctx context.Context, userID int64, fileBytes []byte) (*dto.User, error) {
	user, err := s.repo.GetByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, errUserNotFound
	}

	if user.AvatarPath != nil {
		s.avatars.Delete(*user.AvatarPath)
	}
	path, err := s.avatars.Save(fileBytes)
	if err != nil {
		return nil, err
	}
	if err := s.repo.UpdateFields(ctx, userID, map[string]any{"avatar_path": path}); err != nil {
		return nil, err
	}
	return s.freshUser(ctx, userID)
}

func (s *Service) DeleteAvatar(ctx context.Context, userID int64) (*dto.User, error) {
	user, err := s.repo.GetByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, errUserNotFound
	}
	if user.AvatarPath != nil {
		s.avatars.Delete(*user.AvatarPath)
		if err := s.repo.UpdateFields(ctx, userID, map[string]any{"avatar_path": nil}); err != nil {
			return nil, err
		}
	}
	return s.freshUser(ctx, userID)
}

// GetUser — администратор компании смотрит карточку члена СВОЕЙ активной
// компании. Пользователь вне этой компании недоступен (иначе — перебор
// идентичностей всей платформы по числовому id).
func (s *Service) GetUser(ctx context.Context, actor *domain.User, userID int64) (*dto.User, error) {
	companyID, err := actorScope(actor)
	if err != nil {
		return nil, err
	}
	user, err := s.repo.GetByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if user == nil || !user.IsActive {
		return nil, errUserNotFound
	}
	m, err := s.repo.GetMembership(ctx, userID, companyID)
	if err != nil {
		return nil, err
	}
	if m == nil {
		return nil, errUserNotFound
	}
	out := dto.NewUser(user)
	return &out, nil
}

// UpdateUser — администратор компании правит профиль члена своей активной
// компании (идентичность + должность в этой компании).
func (s *Service) UpdateUser(ctx context.Context, actor *domain.User, userID int64,
	req dto.UpdateUserRequest) (*dto.User, error) {

	companyID, err := actorScope(actor)
	if err != nil {
		return nil, err
	}
	user, err := s.repo.GetByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if user == nil || !user.IsActive {
		return nil, errUserNotFound
	}
	m, err := s.repo.GetMembership(ctx, userID, companyID)
	if err != nil {
		return nil, err
	}
	if m == nil {
		return nil, errUserNotFound
	}
	return s.applyUserUpdate(ctx, companyID, userID, req)
}

// applyUserUpdate — применить правки профиля члена компании (идентичность +
// должность в этой компании). Общая часть UpdateUser и UpdateCompanyMember;
// проверки доступа и членства — у вызывающего.
func (s *Service) applyUserUpdate(ctx context.Context, companyID, userID int64, req dto.UpdateUserRequest) (*dto.User, error) {
	updates := map[string]any{}
	if req.FIO != nil {
		if err := validateFIO(*req.FIO); err != nil {
			return nil, err
		}
		updates["fio"] = *req.FIO
	}
	if req.Login != nil {
		if err := validateLogin(*req.Login); err != nil {
			return nil, err
		}
		if err := s.ensureLoginFree(ctx, *req.Login, userID); err != nil {
			return nil, err
		}
		updates["login"] = *req.Login
	}
	if req.Phone != nil {
		phone, err := normalizePhone(req.Phone)
		if err != nil {
			return nil, err
		}
		updates["phone"] = phone
	}
	if req.Email != nil {
		email, err := normalizeEmail(req.Email)
		if err != nil {
			return nil, err
		}
		if email != nil {
			if err := s.ensureEmailFree(ctx, *email, userID); err != nil {
				return nil, err
			}
		}
		updates["email"] = email
	}
	if req.OnVacation != nil {
		updates["on_vacation"] = *req.OnVacation
	}

	if len(updates) > 0 {
		if err := s.repo.UpdateFields(ctx, userID, updates); err != nil {
			return nil, err
		}
	}
	if req.Post != nil {
		if err := validatePost(req.Post); err != nil {
			return nil, err
		}
		if err := s.repo.SetMembershipPost(ctx, userID, companyID, req.Post); err != nil {
			return nil, err
		}
	}
	return s.freshMemberUser(ctx, companyID, userID)
}

// HideUser — исключить пользователя из активной компании актора (удаление
// членства). Глобальный аккаунт сохраняется.
func (s *Service) HideUser(ctx context.Context, actor *domain.User, userID int64) error {
	if userID == actor.ID {
		return domain.NewError("SELF_HIDE", "Нельзя исключить самого себя", 422)
	}
	companyID, err := actorScope(actor)
	if err != nil {
		return err
	}
	user, err := s.repo.GetByID(ctx, userID)
	if err != nil {
		return err
	}
	if user == nil {
		return errUserNotFound
	}
	if user.IsSuperAdmin {
		return errSuperAdminProtected
	}
	m, err := s.repo.GetMembership(ctx, userID, companyID)
	if err != nil {
		return err
	}
	if m == nil {
		return errUserNotFound
	}
	if m.Role.Level > actor.Level() {
		return domain.NewError("ROLE_LEVEL_FORBIDDEN", "Нельзя исключить пользователя с более высокой ролью", 403)
	}
	if err := s.guardLastAdmin(ctx, companyID, m.Role.Level, domain.LevelEmployee); err != nil {
		return err
	}
	if err := s.repo.RemoveMembership(ctx, userID, companyID); err != nil {
		return err
	}
	s.log.Info("member.remove", "user_id", userID, "company_id", companyID, "actor_id", actor.ID)
	return nil
}

// guardLastAdmin — защита «последнего администратора компании»: запрет
// убрать/понизить единственного администратора (level 3 → ниже).
func (s *Service) guardLastAdmin(ctx context.Context, companyID int64, fromLevel, toLevel int) error {
	if fromLevel < domain.LevelAdmin || toLevel >= domain.LevelAdmin {
		return nil
	}
	n, err := s.repo.CountCompanyMembersByLevel(ctx, companyID, domain.LevelAdmin)
	if err != nil {
		return err
	}
	if n <= 1 {
		return domain.NewError("LAST_ADMIN", "Нельзя убрать единственного администратора компании", 422)
	}
	return nil
}

func (s *Service) AssignRole(ctx context.Context, actor *domain.User, userID, roleID int64) (*dto.User, error) {
	if userID == actor.ID {
		return nil, domain.NewError("SELF_ROLE_CHANGE", "Нельзя изменить свою роль", 422)
	}
	companyID, err := actorScope(actor)
	if err != nil {
		return nil, err
	}
	user, err := s.repo.GetByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if user == nil || !user.IsActive {
		return nil, errUserNotFound
	}
	if user.IsSuperAdmin {
		return nil, errSuperAdminProtected
	}
	membership, err := s.repo.GetMembership(ctx, userID, companyID)
	if err != nil {
		return nil, err
	}
	if membership == nil {
		return nil, errUserNotFound
	}
	newRole, err := s.validMemberRole(ctx, roleID)
	if err != nil {
		return nil, err
	}
	if newRole.Level > actor.Level() {
		return nil, domain.NewError("ROLE_LEVEL_FORBIDDEN", "Нельзя назначить роль выше собственной", 403)
	}
	if err := s.guardLastAdmin(ctx, companyID, membership.Role.Level, newRole.Level); err != nil {
		return nil, err
	}
	if err := s.repo.UpdateMembershipRole(ctx, userID, companyID, roleID); err != nil {
		return nil, err
	}
	return s.freshMemberUser(ctx, companyID, userID)
}

func (s *Service) ResetPassword(ctx context.Context, actor *domain.User, userID int64) error {
	if userID == actor.ID {
		return domain.NewError("SELF_RESET", "Нельзя сбросить собственный пароль", 422)
	}
	companyID, err := actorScope(actor)
	if err != nil {
		return err
	}
	user, err := s.repo.GetByID(ctx, userID)
	if err != nil {
		return err
	}
	if user == nil || !user.IsActive {
		return errUserNotFound
	}
	if user.IsSuperAdmin {
		return errSuperAdminProtected
	}
	m, err := s.repo.GetMembership(ctx, userID, companyID)
	if err != nil {
		return err
	}
	if m == nil {
		return errUserNotFound
	}
	if m.Role.Level > actor.Level() {
		return domain.NewError("ROLE_LEVEL_FORBIDDEN", "Нельзя сбросить пароль пользователю с более высокой ролью", 403)
	}

	hashed, err := s.repo.HashPassword(ctx, user.Login+"123")
	if err != nil {
		return err
	}
	if err := s.repo.UpdateFields(ctx, userID, map[string]any{
		"hash_password": hashed, "is_default_pass": true,
	}); err != nil {
		return err
	}
	s.log.Info("user.reset_password", "user_id", userID, "actor_id", actor.ID)
	return nil
}

// ── Членство в компаниях (multi-company) ──

var errCompanyForbidden = domain.NewError("FORBIDDEN", "Недостаточно прав в этой компании", 403)

// companyAuthority — полномочия актора в КОНКРЕТНОЙ компании: супер-админ
// (платформа) или член компании с ролью администратора. Для чтения карточки
// и настроек компании (любой её администратор).
func (s *Service) companyAuthority(ctx context.Context, actor *domain.User, companyID int64) (int, error) {
	if actor != nil && actor.IsSuperAdmin {
		return domain.LevelAdmin, nil
	}
	am, err := s.repo.GetMembership(ctx, actor.ID, companyID)
	if err != nil {
		return 0, err
	}
	if am == nil || am.Role.Level < domain.LevelAdmin {
		return 0, errCompanyForbidden
	}
	return am.Role.Level, nil
}

// creatorAuthority — право управлять участниками/ролями и создавать/редактировать
// пользователей компании: только её СОЗДАТЕЛЬ (created_by) или супер-админ.
// Не-создатель-администратор имеет ограниченные права (видит компанию и её
// настройки, но участниками не управляет).
func (s *Service) creatorAuthority(ctx context.Context, actor *domain.User, companyID int64) error {
	if actor != nil && actor.IsSuperAdmin {
		return nil
	}
	company, err := s.companies.GetCompany(ctx, companyID)
	if err != nil {
		return err
	}
	if company == nil {
		return errCompanyNotFound
	}
	if actor == nil || company.CreatedBy == nil || *company.CreatedBy != actor.ID {
		return errCompanyForbidden
	}
	return nil
}

// validMemberRole — роль участника компании (Сотрудник/Менеджер/Администратор).
func (s *Service) validMemberRole(ctx context.Context, roleID int64) (*domain.Role, error) {
	role, err := s.repo.GetRole(ctx, roleID)
	if err != nil {
		return nil, err
	}
	if role == nil {
		return nil, domain.NewError("ROLE_NOT_FOUND", "Роль не найдена", 404)
	}
	if role.Level < domain.LevelEmployee || role.Level > domain.LevelAdmin {
		return nil, domain.NewError("ROLE_LEVEL_FORBIDDEN", "Недопустимая роль в компании", 422)
	}
	return role, nil
}

func (s *Service) ListCompanyMembers(ctx context.Context, actor *domain.User, companyID int64) ([]dto.DirectoryUser, error) {
	if _, err := s.companyAuthority(ctx, actor, companyID); err != nil {
		return nil, err
	}
	users, err := s.repo.SearchDirectoryMembers(ctx, "", 0, companyID)
	if err != nil {
		return nil, err
	}
	return dto.NewDirectoryUsers(users), nil
}

func (s *Service) SearchCandidates(ctx context.Context, actor *domain.User, companyID int64, query string) ([]dto.DirectoryUser, error) {
	if _, err := s.companyAuthority(ctx, actor, companyID); err != nil {
		return nil, err
	}
	users, err := s.repo.SearchNonMembers(ctx, query, companyID)
	if err != nil {
		return nil, err
	}
	return dto.NewDirectoryUsers(users), nil
}

func (s *Service) AddCompanyMember(ctx context.Context, actor *domain.User, companyID, userID, roleID int64) error {
	if err := s.creatorAuthority(ctx, actor, companyID); err != nil {
		return err
	}
	role, err := s.validMemberRole(ctx, roleID)
	if err != nil {
		return err
	}
	target, err := s.repo.GetByID(ctx, userID)
	if err != nil {
		return err
	}
	if target == nil || !target.IsActive {
		return errUserNotFound
	}
	if target.IsSuperAdmin {
		return errSuperAdminProtected
	}
	// Повторное добавление существующего участника апсертит роль — это смена
	// роли, и гард «последнего администратора» обязан действовать и здесь.
	existing, err := s.repo.GetMembership(ctx, userID, companyID)
	if err != nil {
		return err
	}
	if existing != nil {
		if err := s.guardLastAdmin(ctx, companyID, existing.Role.Level, role.Level); err != nil {
			return err
		}
	}
	if err := s.repo.AddMembership(ctx, userID, companyID, roleID); err != nil {
		return err
	}
	// Уже состоял — выставить указанную роль.
	if err := s.repo.UpdateMembershipRole(ctx, userID, companyID, roleID); err != nil {
		return err
	}
	s.log.Info("member.add", "user_id", userID, "company_id", companyID, "actor_id", actor.ID)
	return nil
}

func (s *Service) SetMemberRole(ctx context.Context, actor *domain.User, companyID, userID, roleID int64) error {
	if err := s.creatorAuthority(ctx, actor, companyID); err != nil {
		return err
	}
	role, err := s.validMemberRole(ctx, roleID)
	if err != nil {
		return err
	}
	m, err := s.repo.GetMembership(ctx, userID, companyID)
	if err != nil {
		return err
	}
	if m == nil {
		return errUserNotFound
	}
	if err := s.guardLastAdmin(ctx, companyID, m.Role.Level, role.Level); err != nil {
		return err
	}
	return s.repo.UpdateMembershipRole(ctx, userID, companyID, roleID)
}

func (s *Service) RemoveCompanyMember(ctx context.Context, actor *domain.User, companyID, userID int64) error {
	if err := s.creatorAuthority(ctx, actor, companyID); err != nil {
		return err
	}
	m, err := s.repo.GetMembership(ctx, userID, companyID)
	if err != nil {
		return err
	}
	if m == nil {
		return errUserNotFound
	}
	if err := s.guardLastAdmin(ctx, companyID, m.Role.Level, domain.LevelEmployee); err != nil {
		return err
	}
	if err := s.repo.RemoveMembership(ctx, userID, companyID); err != nil {
		return err
	}
	s.log.Info("member.remove", "user_id", userID, "company_id", companyID, "actor_id", actor.ID)
	return nil
}

// ── Создание/редактирование сотрудников В КОНКРЕТНОЙ компании (раздел
// «Компании»; только создатель компании или супер-админ) ──

// CreateCompanyUser — создатель компании заводит сотрудника: идентичность +
// членство с ролью/должностью. Email подтверждён сразу (аккаунт заведён
// администратором; вместо верификации — обязательная смена пароля при входе).
func (s *Service) CreateCompanyUser(ctx context.Context, actor *domain.User, companyID int64, req dto.CreateUserRequest) (*dto.User, error) {
	if err := s.creatorAuthority(ctx, actor, companyID); err != nil {
		return nil, err
	}
	if err := validateFIO(req.FIO); err != nil {
		return nil, err
	}
	if err := validateLogin(req.Login); err != nil {
		return nil, err
	}
	if err := validatePost(req.Post); err != nil {
		return nil, err
	}
	phone, err := normalizePhone(req.Phone)
	if err != nil {
		return nil, err
	}
	email, err := normalizeEmail(req.Email)
	if err != nil {
		return nil, err
	}
	if req.Password != nil {
		if err := validatePassword(*req.Password); err != nil {
			return nil, err
		}
	}
	role, err := s.validMemberRole(ctx, req.RoleID)
	if err != nil {
		return nil, err
	}
	if err := s.ensureLoginFree(ctx, req.Login, 0); err != nil {
		return nil, err
	}
	if email != nil {
		if err := s.ensureEmailFree(ctx, *email, 0); err != nil {
			return nil, err
		}
	}

	password := req.Login + "123"
	isDefault := true
	if req.Password != nil {
		password = *req.Password
		isDefault = false
	}
	hashed, err := s.repo.HashPassword(ctx, password)
	if err != nil {
		return nil, err
	}
	user := &domain.User{
		FIO: req.FIO, Login: req.Login, HashPassword: hashed,
		Phone: phone, Email: email, IsDefaultPass: isDefault, EmailVerified: true,
	}
	if err := s.repo.Create(ctx, user); err != nil {
		return nil, err
	}
	if err := s.repo.AddMembership(ctx, user.ID, companyID, role.ID); err != nil {
		return nil, err
	}
	if req.Post != nil {
		if err := s.repo.SetMembershipPost(ctx, user.ID, companyID, req.Post); err != nil {
			return nil, err
		}
	}
	s.log.Info("company.user_create", "user_id", user.ID, "company_id", companyID, "actor_id", actor.ID)
	return s.freshMemberUser(ctx, companyID, user.ID)
}

// UpdateCompanyMember — создатель компании правит профиль её члена.
func (s *Service) UpdateCompanyMember(ctx context.Context, actor *domain.User, companyID, userID int64, req dto.UpdateUserRequest) (*dto.User, error) {
	if err := s.creatorAuthority(ctx, actor, companyID); err != nil {
		return nil, err
	}
	user, err := s.repo.GetByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if user == nil || !user.IsActive {
		return nil, errUserNotFound
	}
	if user.IsSuperAdmin {
		return nil, errSuperAdminProtected
	}
	m, err := s.repo.GetMembership(ctx, userID, companyID)
	if err != nil {
		return nil, err
	}
	if m == nil {
		return nil, errUserNotFound
	}
	return s.applyUserUpdate(ctx, companyID, userID, req)
}

// ResetCompanyMemberPassword — сброс пароля члена компании на дефолтный
// <login>123 с обязательной сменой при входе (создатель компании).
func (s *Service) ResetCompanyMemberPassword(ctx context.Context, actor *domain.User, companyID, userID int64) error {
	if actor != nil && userID == actor.ID {
		return domain.NewError("SELF_RESET", "Нельзя сбросить собственный пароль", 422)
	}
	if err := s.creatorAuthority(ctx, actor, companyID); err != nil {
		return err
	}
	user, err := s.repo.GetByID(ctx, userID)
	if err != nil {
		return err
	}
	if user == nil || !user.IsActive {
		return errUserNotFound
	}
	if user.IsSuperAdmin {
		return errSuperAdminProtected
	}
	m, err := s.repo.GetMembership(ctx, userID, companyID)
	if err != nil {
		return err
	}
	if m == nil {
		return errUserNotFound
	}
	hashed, err := s.repo.HashPassword(ctx, user.Login+"123")
	if err != nil {
		return err
	}
	if err := s.repo.UpdateFields(ctx, userID, map[string]any{
		"hash_password": hashed, "is_default_pass": true,
	}); err != nil {
		return err
	}
	s.log.Info("company.user_reset_password", "user_id", userID, "company_id", companyID, "actor_id", actor.ID)
	return nil
}
