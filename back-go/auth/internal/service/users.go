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

	user := &domain.User{
		FIO:           req.FIO,
		Login:         req.Login,
		HashPassword:  hashed,
		Role:          *role,
		CompanyID:     req.CompanyID,
		Post:          req.Post,
		Phone:         phone,
		Email:         email,
		IsDefaultPass: isDefault,
	}
	if err := s.repo.Create(ctx, user); err != nil {
		return nil, err
	}

	s.log.Info("user.create", "user_id", user.ID, "actor_id", actor.ID)
	return s.freshUser(ctx, user.ID)
}

func (s *Service) Directory(ctx context.Context, req dto.DirectoryRequest) ([]dto.DirectoryUser, error) {
	me, err := s.repo.GetByID(ctx, req.ActorID)
	if err != nil {
		return nil, err
	}
	// company_id: обычным сотрудникам навязываем их компанию; Администратор
	// системы (без компании) выбирает через query или получает всех.
	companyID := req.CompanyID
	if me != nil && me.CompanyID != nil {
		companyID = me.CompanyID
	}
	users, err := s.repo.SearchDirectory(ctx, req.Query, req.ExcludeID, companyID)
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

	user, err := s.repo.GetByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if user == nil || user.IsHidden {
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
	if user.Role.Level >= domain.LevelAdmin {
		n, err := s.repo.CountVisibleByLevel(ctx, domain.LevelAdmin)
		if err != nil {
			return nil, err
		}
		if n <= 1 {
			return nil, domain.NewError("LAST_ADMIN", "Нельзя изменить роль единственного Администратора системы", 422)
		}
	}
	// Корневой Руководитель компании (companies.director_id) — разжаловать
	// может только Администратор системы: страховка от «дворцового переворота».
	if user.Role.Level >= domain.LevelDirector && newRole.Level < domain.LevelDirector {
		isRootDirector, err := s.repo.IsCompanyDirector(ctx, user.ID)
		if err != nil {
			return nil, err
		}
		if isRootDirector && actor.Level() < domain.LevelAdmin {
			return nil, domain.NewError("ROOT_DIRECTOR",
				"Корневого Руководителя компании может разжаловать только Администратор системы", 422)
		}
	}

	if err := s.repo.UpdateFields(ctx, userID, map[string]any{"role_id": roleID}); err != nil {
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
