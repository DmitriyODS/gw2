package service

import (
	"context"

	"github.com/DmitriyODS/gw2/back-go/auth/internal/domain"
	"github.com/DmitriyODS/gw2/back-go/auth/internal/dto"
)

var errUserNotFound = domain.NewError("NOT_FOUND", "Пользователь не найден", 404)

func (s *Service) ListUsers(ctx context.Context) ([]dto.User, error) {
	users, err := s.repo.ListVisible(ctx)
	if err != nil {
		return nil, err
	}
	return dto.NewUsers(users), nil
}

// ensureLoginFree / ensureEmailFree — проверки уникальности с учётом
// «это тот же пользователь».
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

func (s *Service) CreateUser(ctx context.Context, actor *domain.User, req dto.CreateUserRequest) (*dto.User, error) {
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

	role, err := s.repo.GetRole(ctx, req.RoleID)
	if err != nil {
		return nil, err
	}
	if role == nil {
		return nil, domain.NewError("ROLE_NOT_FOUND", "Роль не найдена", 404)
	}
	// Нельзя назначить роль выше своей. Равную — можно (Админ системы может
	// создать ещё одного Админа системы).
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

	// Компанию из тела принимаем только у Администратора системы (company_id
	// NULL — он один создаёт пользователей вне своей компании). У остальных
	// (Руководитель/Менеджер) сотрудник принудительно создаётся в компании
	// создателя — иначе он создавался «без компании» и не появлялся в списке.
	companyID := req.CompanyID
	if actor.CompanyID != nil {
		companyID = actor.CompanyID
	}

	user := &domain.User{
		FIO:           req.FIO,
		Login:         req.Login,
		HashPassword:  hashed,
		Role:          *role,
		CompanyID:     companyID,
		Post:          req.Post,
		Phone:         phone,
		Email:         email,
		IsDefaultPass: isDefault,
	}
	if err := s.repo.Create(ctx, user); err != nil {
		return nil, err
	}
	// Первичная связка членства (роль в этой компании). users.company_id/role_id
	// уже проставлены Create — это и есть «первичная» компания нового юзера.
	if companyID != nil {
		if err := s.repo.AddMembership(ctx, user.ID, *companyID, role.ID); err != nil {
			return nil, err
		}
	}

	s.log.Info("user.create", "user_id", user.ID, "actor_id", actor.ID)
	return s.freshUser(ctx, user.ID)
}

// errCompanyScopeRequired — операция требует контекста компании (актив. компания
// из токена для обычного актора; ?company_id= для Администратора системы).
var errCompanyScopeRequired = domain.NewError("COMPANY_SCOPE_REQUIRED", "Требуется указать компанию (company_id)", 400)

// actorScope — активная компания актора: у обычного — из токена (actor.CompanyID),
// у Администратора системы хендлер проставляет её из ?company_id=.
func actorScope(actor *domain.User) (int64, error) {
	if actor == nil || actor.CompanyID == nil {
		return 0, errCompanyScopeRequired
	}
	return *actor.CompanyID, nil
}

func (s *Service) Directory(ctx context.Context, req dto.DirectoryRequest) ([]dto.DirectoryUser, error) {
	// req.CompanyID уже разрешён хендлером: активная компания актора (из токена)
	// либо ?company_id= Администратора системы. Список — члены этой компании с
	// ролью в ней (user_companies). nil (админ без выбора) — все видимые.
	if req.CompanyID == nil {
		users, err := s.repo.SearchDirectory(ctx, req.Query, req.ExcludeID, nil)
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

func (s *Service) DirectoryUser(ctx context.Context, userID int64) (*dto.DirectoryUser, error) {
	user, err := s.repo.GetByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if user == nil || user.IsHidden {
		return nil, domain.NewError("NOT_FOUND", "Сотрудник не найден", 404)
	}
	out := dto.NewDirectoryUser(user)
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
	if req.Post != nil {
		if err := validatePost(req.Post); err != nil {
			return nil, err
		}
		updates["post"] = *req.Post
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

func (s *Service) GetUser(ctx context.Context, userID int64) (*dto.User, error) {
	user, err := s.repo.GetByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if user == nil || user.IsHidden {
		return nil, errUserNotFound
	}
	out := dto.NewUser(user)
	return &out, nil
}

func (s *Service) UpdateUser(ctx context.Context, actor *domain.User, userID int64,
	req dto.UpdateUserRequest) (*dto.User, error) {

	user, err := s.repo.GetByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if user == nil || user.IsHidden {
		return nil, errUserNotFound
	}

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
	if req.Post != nil {
		if err := validatePost(req.Post); err != nil {
			return nil, err
		}
		updates["post"] = *req.Post
	}
	if req.CompanyID != nil {
		updates["company_id"] = *req.CompanyID
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

	if len(updates) > 0 {
		if err := s.repo.UpdateFields(ctx, userID, updates); err != nil {
			return nil, err
		}
	}
	return s.freshUser(ctx, userID)
}

func (s *Service) HideUser(ctx context.Context, actor *domain.User, userID int64) error {
	if userID == actor.ID {
		return domain.NewError("SELF_HIDE", "Нельзя скрыть самого себя", 422)
	}

	user, err := s.repo.GetByID(ctx, userID)
	if err != nil {
		return err
	}
	if user == nil || user.IsHidden {
		return errUserNotFound
	}
	// Обычный актор управляет только членами своей активной компании.
	if actor.CompanyID != nil {
		m, err := s.repo.GetMembership(ctx, userID, *actor.CompanyID)
		if err != nil {
			return err
		}
		if m == nil {
			return errUserNotFound
		}
	}

	// Нельзя скрыть пользователя с более высоким уровнем; равный допускаем
	// (Админ системы может удалить другого Админа системы).
	if user.Role.Level > actor.Level() {
		return domain.NewError("ROLE_LEVEL_FORBIDDEN", "Нельзя удалить пользователя с более высокой ролью", 403)
	}
	// Корневой Администратор системы защищён от скрытия. Запасная защита:
	// единственный Администратор системы тоже неприкосновенен.
	if user.IsRootAdmin {
		return domain.NewError("ROOT_ADMIN", "Корневого Администратора системы нельзя удалить", 422)
	}
	if user.Role.Level >= domain.LevelAdmin {
		n, err := s.repo.CountVisibleByLevel(ctx, domain.LevelAdmin)
		if err != nil {
			return err
		}
		if n <= 1 {
			return domain.NewError("LAST_ADMIN", "Нельзя скрыть единственного Администратора системы", 422)
		}
	}
	// Корневого Руководителя компании может скрыть только Админ системы.
	if user.Role.Level >= domain.LevelDirector {
		isRootDirector, err := s.repo.IsCompanyDirector(ctx, user.ID)
		if err != nil {
			return err
		}
		if isRootDirector && actor.Level() < domain.LevelAdmin {
			return domain.NewError("ROOT_DIRECTOR",
				"Корневого Руководителя компании может удалить только Администратор системы", 422)
		}
	}

	if err := s.repo.UpdateFields(ctx, userID, map[string]any{"is_hidden": true}); err != nil {
		return err
	}
	s.log.Info("user.hide", "user_id", userID, "actor_id", actor.ID)
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
	if user == nil || user.IsHidden {
		return nil, errUserNotFound
	}
	// Роль меняется В АКТИВНОЙ КОМПАНИИ актора — целевой должен быть её членом.
	membership, err := s.repo.GetMembership(ctx, userID, companyID)
	if err != nil {
		return nil, err
	}
	if membership == nil {
		return nil, errUserNotFound
	}

	newRole, err := s.repo.GetRole(ctx, roleID)
	if err != nil {
		return nil, err
	}
	if newRole == nil {
		return nil, domain.NewError("ROLE_NOT_FOUND", "Роль не найдена", 404)
	}
	// Нельзя назначить роль выше своего уровня; равную — можно.
	if newRole.Level > actor.Level() {
		return nil, domain.NewError("ROLE_LEVEL_FORBIDDEN", "Нельзя назначить роль выше собственной", 403)
	}
	if user.IsRootAdmin {
		return nil, domain.NewError("ROOT_ADMIN", "Корневому Администратору системы нельзя сменить роль", 422)
	}
	// Понижение Руководителя в этой компании: корневого Руководителя
	// (companies.director_id) разжалует только Админ системы; единственного
	// Руководителя компании разжаловать нельзя.
	if membership.Role.Level >= domain.LevelDirector && newRole.Level < domain.LevelDirector {
		isRootDirector, err := s.repo.IsCompanyDirector(ctx, user.ID)
		if err != nil {
			return nil, err
		}
		if isRootDirector && actor.Level() < domain.LevelAdmin {
			return nil, domain.NewError("ROOT_DIRECTOR",
				"Корневого Руководителя компании может разжаловать только Администратор системы", 422)
		}
		n, err := s.repo.CountCompanyMembersByLevel(ctx, companyID, domain.LevelDirector)
		if err != nil {
			return nil, err
		}
		if n <= 1 {
			return nil, domain.NewError("LAST_DIRECTOR", "Нельзя разжаловать единственного Руководителя компании", 422)
		}
	}

	if err := s.repo.UpdateMembershipRole(ctx, userID, companyID, roleID); err != nil {
		return nil, err
	}
	if err := s.repo.SyncPrimaryCompany(ctx, userID); err != nil {
		return nil, err
	}
	return s.freshUser(ctx, userID)
}

func (s *Service) ResetPassword(ctx context.Context, actor *domain.User, userID int64) error {
	if userID == actor.ID {
		return domain.NewError("SELF_RESET", "Нельзя сбросить собственный пароль", 422)
	}

	user, err := s.repo.GetByID(ctx, userID)
	if err != nil {
		return err
	}
	if user == nil || user.IsHidden {
		return errUserNotFound
	}
	// Обычный актор управляет только членами своей активной компании.
	if actor.CompanyID != nil {
		m, err := s.repo.GetMembership(ctx, userID, *actor.CompanyID)
		if err != nil {
			return err
		}
		if m == nil {
			return errUserNotFound
		}
	}
	if user.Role.Level > actor.Level() {
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

// ── Членство пользователя в компаниях (multi-company) ──

// companyAuthority — уровень полномочий актора в КОНКРЕТНОЙ компании: Админ
// системы — LevelAdmin; иначе нужно членство уровня ≥ Руководитель.
func (s *Service) companyAuthority(ctx context.Context, actor *domain.User, companyID int64) (int, error) {
	if isSystemAdmin(actor) {
		return domain.LevelAdmin, nil
	}
	am, err := s.repo.GetMembership(ctx, actor.ID, companyID)
	if err != nil {
		return 0, err
	}
	if am == nil || am.Role.Level < domain.LevelDirector {
		return 0, domain.NewError("FORBIDDEN", "Недостаточно прав в этой компании", 403)
	}
	return am.Role.Level, nil
}

var errMembersAdminOnly = domain.NewError("FORBIDDEN", "Управлять участниками компании может только Администратор системы", 403)

// clearDirectorIfMatches — если пользователь был корневым Руководителем компании
// (companies.director_id), снять привязку (членство меняется/убирается).
func (s *Service) clearDirectorIfMatches(ctx context.Context, companyID, userID int64) error {
	company, err := s.companies.GetCompany(ctx, companyID)
	if err != nil {
		return err
	}
	if company != nil && company.DirectorID != nil && *company.DirectorID == userID {
		return s.companies.UpdateCompanyFields(ctx, companyID, map[string]any{"director_id": nil})
	}
	return nil
}

// validMemberRole — роль участника компании: только Сотрудник/Менеджер/
// Руководитель (Администратор системы — глобальная роль вне компаний).
func (s *Service) validMemberRole(ctx context.Context, roleID int64) (*domain.Role, error) {
	role, err := s.repo.GetRole(ctx, roleID)
	if err != nil {
		return nil, err
	}
	if role == nil {
		return nil, domain.NewError("ROLE_NOT_FOUND", "Роль не найдена", 404)
	}
	if role.Level >= domain.LevelAdmin {
		return nil, domain.NewError("ROLE_LEVEL_FORBIDDEN", "Роль в компании не может быть Администратором системы", 422)
	}
	return role, nil
}

func (s *Service) ListCompanyMembers(ctx context.Context, actor *domain.User, companyID int64) ([]dto.DirectoryUser, error) {
	if !isSystemAdmin(actor) {
		return nil, errMembersAdminOnly
	}
	users, err := s.repo.SearchDirectoryMembers(ctx, "", 0, companyID)
	if err != nil {
		return nil, err
	}
	return dto.NewDirectoryUsers(users), nil
}

func (s *Service) SearchCandidates(ctx context.Context, actor *domain.User, companyID int64, query string) ([]dto.DirectoryUser, error) {
	if !isSystemAdmin(actor) {
		return nil, errMembersAdminOnly
	}
	users, err := s.repo.SearchNonMembers(ctx, query, companyID)
	if err != nil {
		return nil, err
	}
	return dto.NewDirectoryUsers(users), nil
}

func (s *Service) AddCompanyMember(ctx context.Context, actor *domain.User, companyID, userID, roleID int64) error {
	// Добавлять в компанию может ТОЛЬКО Администратор системы (в карточке
	// компании). Самостоятельное вступление — по ссылке-приглашению (JoinByCode).
	if !isSystemAdmin(actor) {
		return errMembersAdminOnly
	}
	if _, err := s.validMemberRole(ctx, roleID); err != nil {
		return err
	}
	target, err := s.repo.GetByID(ctx, userID)
	if err != nil {
		return err
	}
	if target == nil || target.IsHidden {
		return errUserNotFound
	}
	if target.IsRootAdmin {
		return domain.NewError("ROOT_ADMIN", "Администратора системы нельзя добавить в компанию", 422)
	}
	if err := s.repo.AddMembership(ctx, userID, companyID, roleID); err != nil {
		return err
	}
	// Уже состоял — выставить указанную роль.
	if err := s.repo.UpdateMembershipRole(ctx, userID, companyID, roleID); err != nil {
		return err
	}
	if err := s.repo.SyncPrimaryCompany(ctx, userID); err != nil {
		return err
	}
	s.log.Info("member.add", "user_id", userID, "company_id", companyID, "actor_id", actor.ID)
	return nil
}

func (s *Service) SetMemberRole(ctx context.Context, actor *domain.User, companyID, userID, roleID int64) error {
	if !isSystemAdmin(actor) {
		return errMembersAdminOnly
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
	if err := s.repo.UpdateMembershipRole(ctx, userID, companyID, roleID); err != nil {
		return err
	}
	if role.Level < domain.LevelDirector {
		if err := s.clearDirectorIfMatches(ctx, companyID, userID); err != nil {
			return err
		}
	}
	return s.repo.SyncPrimaryCompany(ctx, userID)
}

func (s *Service) RemoveCompanyMember(ctx context.Context, actor *domain.User, companyID, userID int64) error {
	if !isSystemAdmin(actor) {
		return errMembersAdminOnly
	}
	m, err := s.repo.GetMembership(ctx, userID, companyID)
	if err != nil {
		return err
	}
	if m == nil {
		return errUserNotFound
	}
	// Если это был корневой Руководитель — снять director_id, иначе ссылка повиснет.
	if err := s.clearDirectorIfMatches(ctx, companyID, userID); err != nil {
		return err
	}
	if err := s.repo.RemoveMembership(ctx, userID, companyID); err != nil {
		return err
	}
	if err := s.repo.SyncPrimaryCompany(ctx, userID); err != nil {
		return err
	}
	s.log.Info("member.remove", "user_id", userID, "company_id", companyID, "actor_id", actor.ID)
	return nil
}
