package service

import (
	"context"
	"strings"
	"time"

	"github.com/DmitriyODS/gw2/back-go/auth/internal/domain"
	"github.com/DmitriyODS/gw2/back-go/auth/internal/dto"
)

const companyInviteTTL = 7 * 24 * time.Hour

var errInvalidInvite = domain.NewError("INVALID_INVITE", "Приглашение недействительно или истекло", 404)

// CreateCompanyInvite — выслать email-приглашение в компанию с ролью. Только
// создатель компании или супер-админ. Перевыпуск перезаписывает прежний токен
// для этой пары компания+email.
func (s *Service) CreateCompanyInvite(ctx context.Context, actor *domain.User, companyID int64, email string, roleID int64) error {
	if err := s.creatorAuthority(ctx, actor, companyID); err != nil {
		return err
	}
	emailPtr, err := normalizeEmail(&email)
	if err != nil || emailPtr == nil {
		return errValidation("Укажите корректный email")
	}
	role, err := s.validMemberRole(ctx, roleID)
	if err != nil {
		return err
	}
	company, err := s.companies.GetCompany(ctx, companyID)
	if err != nil {
		return err
	}
	if company == nil {
		return errCompanyNotFound
	}

	tok, err := randomToken()
	if err != nil {
		return err
	}
	if err := s.companyInvites.Upsert(ctx, companyID, *emailPtr, roleID, tok, &actor.ID, time.Now().Add(companyInviteTTL)); err != nil {
		return err
	}
	link := strings.TrimRight(s.appBaseURL, "/") + "/invite/" + tok
	if err := s.mail.SendCompanyInvite(ctx, *emailPtr, company.Name, role.Name, link); err != nil {
		s.log.Warn("company.invite_send_failed", "company_id", companyID, "error", err)
		return domain.NewError("MAIL_FAILED", "Не удалось отправить письмо — проверьте настройки почты", 502)
	}
	s.log.Info("company.invite_email", "company_id", companyID, "actor_id", actor.ID)
	return nil
}

// GetCompanyInvitePreview — что показать получателю по ссылке до принятия.
func (s *Service) GetCompanyInvitePreview(ctx context.Context, token string) (*dto.InvitePreview, error) {
	inv, err := s.companyInvites.GetByToken(ctx, token)
	if err != nil {
		return nil, err
	}
	if inv == nil || time.Now().After(inv.ExpiresAt) {
		return nil, errInvalidInvite
	}
	return &dto.InvitePreview{CompanyName: inv.CompanyName, RoleName: inv.RoleName, Email: inv.Email}, nil
}

// AcceptCompanyInvite — авторизованный пользователь принимает приглашение:
// добавляется в компанию с указанной ролью, токен гасится, выдаётся сессия,
// переключённая на компанию. Токен — capability (как ссылка-код /join).
func (s *Service) AcceptCompanyInvite(ctx context.Context, userID int64, token string) (*dto.Session, error) {
	inv, err := s.companyInvites.GetByToken(ctx, token)
	if err != nil {
		return nil, err
	}
	if inv == nil || time.Now().After(inv.ExpiresAt) {
		return nil, errInvalidInvite
	}
	user, err := s.repo.GetByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if user == nil || !user.IsActive {
		return nil, errUserNotFound
	}
	if user.IsSuperAdmin {
		_ = s.companyInvites.Delete(ctx, inv.ID)
		return s.session(ctx, user, nil, true)
	}
	company, err := s.companies.GetCompany(ctx, inv.CompanyID)
	if err != nil {
		return nil, err
	}
	if company == nil {
		return nil, errCompanyNotFound
	}
	if !company.IsActive {
		return nil, companyDisabledErr(&company.Name)
	}
	// Принятие инвайта существующим участником апсертит роль — это смена роли,
	// поэтому гард «последнего администратора» действует и на этом пути.
	if existing, err := s.repo.GetMembership(ctx, userID, inv.CompanyID); err != nil {
		return nil, err
	} else if existing != nil {
		role, err := s.repo.GetRole(ctx, inv.RoleID)
		if err != nil {
			return nil, err
		}
		if role == nil {
			return nil, domain.NewError("ROLE_NOT_FOUND", "Роль не найдена", 500)
		}
		if err := s.guardLastAdmin(ctx, inv.CompanyID, existing.Role.Level, role.Level); err != nil {
			return nil, err
		}
	}
	if err := s.repo.AddMembership(ctx, userID, inv.CompanyID, inv.RoleID); err != nil {
		return nil, err
	}
	// Уже состоял — выставить роль из приглашения.
	if err := s.repo.UpdateMembershipRole(ctx, userID, inv.CompanyID, inv.RoleID); err != nil {
		return nil, err
	}
	_ = s.companyInvites.Delete(ctx, inv.ID)
	s.log.Info("company.invite_accept", "user_id", userID, "company_id", inv.CompanyID)
	return s.session(ctx, user, &inv.CompanyID, true)
}
