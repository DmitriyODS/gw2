// Package service — бизнес-логика авторизации, пользователей, компаний,
// ролей и резервных копий. Портировано из back/app/services/
// {auth_service,user_service,company_service,backup_service}.py и
// api/{companies,roles,backup}.py без изменения правил.
package service

import (
	"context"
	"log/slog"

	"github.com/DmitriyODS/gw2/back-go/auth/internal/domain"
	"github.com/DmitriyODS/gw2/back-go/auth/internal/dto"
	"github.com/DmitriyODS/gw2/back-go/auth/internal/token"
)

// AuthService — все use-case'ы сервиса (auth + users).
type AuthService interface {
	Login(ctx context.Context, req dto.LoginRequest) (*dto.Session, error)
	Register(ctx context.Context, req dto.RegisterRequest) (*dto.RegisterResult, error)
	SuggestLogin(ctx context.Context, fio string) (string, error)
	VerifyEmail(ctx context.Context, req dto.VerifyEmailRequest) (*dto.Session, error)
	ResendVerification(ctx context.Context, email string) error
	RequestPasswordReset(ctx context.Context, email string) error
	ResetPasswordByToken(ctx context.Context, req dto.ResetPasswordRequest) (*dto.PasswordResetResult, error)
	SelectCompany(ctx context.Context, selectToken string, companyID int64) (*dto.Session, error)
	SwitchCompany(ctx context.Context, userID, companyID int64) (*dto.Session, error)
	Refresh(ctx context.Context, refreshToken string) (*dto.Session, error)
	ChangeDefault(ctx context.Context, req dto.ChangeDefaultRequest) (*dto.Session, error)

	// Спаривание устройств: QR-вход и авторизация ТВ-киоска по коду/QR.
	LinkStart(ctx context.Context, kind string) (*dto.LinkStartResult, error)
	LinkInfo(ctx context.Context, code string) (*dto.LinkInfo, error)
	LinkApprove(ctx context.Context, code string, userID int64, activeCompanyID *int64) error
	LinkClaim(ctx context.Context, code, secret string) (*dto.LinkClaimResult, error)

	ListUsers(ctx context.Context) ([]dto.User, error)
	CreateUser(ctx context.Context, actor *domain.User, req dto.CreateUserRequest) (*dto.User, error)
	CreatePlatformUser(ctx context.Context, req dto.CreateUserRequest) (*dto.User, error)
	UpdatePlatformUser(ctx context.Context, userID int64, req dto.UpdateUserRequest) (*dto.User, error)
	ResetPlatformUserPassword(ctx context.Context, actor *domain.User, userID int64) error
	DeactivatePlatformUser(ctx context.Context, actor *domain.User, userID int64) error
	Directory(ctx context.Context, req dto.DirectoryRequest) ([]dto.DirectoryUser, error)
	DirectoryUser(ctx context.Context, userID int64) (*dto.DirectoryUser, error)
	Me(ctx context.Context, userID int64) (*dto.User, error)
	UpdateMe(ctx context.Context, userID int64, req dto.UpdateMeRequest) (*dto.User, error)
	UploadAvatar(ctx context.Context, userID int64, fileBytes []byte) (*dto.User, error)
	DeleteAvatar(ctx context.Context, userID int64) (*dto.User, error)
	GetUser(ctx context.Context, userID int64) (*dto.User, error)
	UpdateUser(ctx context.Context, actor *domain.User, userID int64, req dto.UpdateUserRequest) (*dto.User, error)
	HideUser(ctx context.Context, actor *domain.User, userID int64) error
	AssignRole(ctx context.Context, actor *domain.User, userID, roleID int64) (*dto.User, error)
	ResetPassword(ctx context.Context, actor *domain.User, userID int64) error

	// Участники компании (multi-company; управляет Администратор системы в
	// карточке компании). Вступление по ссылке-приглашению — JoinByCode.
	ListCompanyMembers(ctx context.Context, actor *domain.User, companyID int64) ([]dto.DirectoryUser, error)
	AddCompanyMember(ctx context.Context, actor *domain.User, companyID, userID, roleID int64) error
	SetMemberRole(ctx context.Context, actor *domain.User, companyID, userID, roleID int64) error
	RemoveCompanyMember(ctx context.Context, actor *domain.User, companyID, userID int64) error
	SearchCandidates(ctx context.Context, actor *domain.User, companyID int64, query string) ([]dto.DirectoryUser, error)
	CompanyInvite(ctx context.Context, actor *domain.User, companyID int64) (string, error)
	RegenerateInvite(ctx context.Context, actor *domain.User, companyID int64) (string, error)
	JoinByCode(ctx context.Context, userID int64, code string) (*dto.Session, error)
	CreateCompanyInvite(ctx context.Context, actor *domain.User, companyID int64, email string, roleID int64) error
	GetCompanyInvitePreview(ctx context.Context, token string) (*dto.InvitePreview, error)
	AcceptCompanyInvite(ctx context.Context, userID int64, token string) (*dto.Session, error)

	ListRoles(ctx context.Context) ([]dto.Role, error)

	ListCompanies(ctx context.Context) (*dto.CompanyList, error)
	ListMyCompanies(ctx context.Context, actor *domain.User) (*dto.CompanyList, error)
	CreateCompanyUser(ctx context.Context, actor *domain.User, companyID int64, req dto.CreateUserRequest) (*dto.User, error)
	UpdateCompanyMember(ctx context.Context, actor *domain.User, companyID, userID int64, req dto.UpdateUserRequest) (*dto.User, error)
	ResetCompanyMemberPassword(ctx context.Context, actor *domain.User, companyID, userID int64) error
	GetCompany(ctx context.Context, actor *domain.User, companyID int64) (*dto.Company, error)
	CreateCompany(ctx context.Context, actor *domain.User, req dto.CompanyCreate) (*dto.Company, error)
	UpdateCompany(ctx context.Context, actor *domain.User, companyID int64, req dto.CompanyUpdate) (*dto.Company, error)
	ToggleCompanyActive(ctx context.Context, companyID int64, isActive bool) (*dto.Company, error)
	DeleteCompany(ctx context.Context, actor *domain.User, companyID int64) error
	GetWeekendSettings(ctx context.Context, actor *domain.User, companyID int64) (*dto.WeekendSettings, error)
	UpdateWeekendSettings(ctx context.Context, actor *domain.User, companyID int64, days []int) (*dto.WeekendSettings, error)
	GetGrooveSettings(ctx context.Context, actor *domain.User, companyID int64) (*dto.GrooveSettings, error)
	UpdateGrooveSettings(ctx context.Context, actor *domain.User, companyID int64, enabled bool) (*dto.GrooveSettings, error)

	ExportBackup(ctx context.Context, sections []string) ([]byte, error)
	ImportBackup(ctx context.Context, zipBytes []byte, sections []string) error

	// OAuth-провайдер (связка аккаунтов навыка Алисы) и вход через Яндекс ID.
	OAuthAuthorize(ctx context.Context, userID int64, companyID *int64, req dto.OAuthAuthorizeRequest) (string, error)
	OAuthToken(ctx context.Context, req dto.OAuthTokenRequest) (*dto.OAuthTokens, error)
	YandexAuthConfig() *dto.YandexAuthConfig
	YandexLogin(ctx context.Context, code string) (*dto.Session, error)
	YandexLinkStatus(ctx context.Context, userID int64) (bool, error)
	YandexLink(ctx context.Context, userID int64, code string) error
	YandexUnlink(ctx context.Context, userID int64) error
}

type Service struct {
	repo           domain.UserRepository
	companies      domain.CompanyRepository
	backup         domain.BackupStore
	throttle       domain.LoginThrottle
	tokens         *token.Issuer
	avatars        domain.AvatarStorage
	verifications  domain.VerificationStore
	passwordResets domain.PasswordResetStore
	companyInvites domain.CompanyInviteStore
	link           domain.DeviceLinkStore
	mail           domain.MailClient
	appBaseURL     string // публичный базовый URL для ссылок в письмах
	log            *slog.Logger

	// OAuth-провайдер для связки аккаунтов Алисы (WithOAuth; nil — выключен).
	oauthCodes        domain.OAuthCodeStore
	oauthClientID     string
	oauthClientSecret string
	// Вход через Яндекс ID (WithYandex; nil — выключен).
	yandex         domain.YandexOAuthClient
	yandexClientID string
}

func New(repo domain.UserRepository, companies domain.CompanyRepository,
	backup domain.BackupStore, throttle domain.LoginThrottle, tokens *token.Issuer,
	avatars domain.AvatarStorage, verifications domain.VerificationStore,
	passwordResets domain.PasswordResetStore, companyInvites domain.CompanyInviteStore,
	link domain.DeviceLinkStore, mail domain.MailClient, appBaseURL string, log *slog.Logger) *Service {
	return &Service{repo: repo, companies: companies, backup: backup,
		throttle: throttle, tokens: tokens, avatars: avatars,
		verifications: verifications, passwordResets: passwordResets,
		companyInvites: companyInvites, link: link, mail: mail, appBaseURL: appBaseURL, log: log}
}

var _ AuthService = (*Service)(nil)

var errNotAMember = domain.NewError("NOT_A_MEMBER", "Нет доступа к выбранной компании", 403)

func companyDisabledErr(name *string) error {
	return domain.NewErrorExtra(
		"COMPANY_DISABLED",
		"Ваша компания отключена. Обратитесь к администратору.",
		403,
		map[string]any{"company_name": name},
	)
}

// session — выпуск пары токенов + клеймы и список членств в тело ответа (общая
// точка для login/select/switch/refresh/change-default/register).
// activeCompanyID — выбранная компания сессии (роль/реквизиты — из членства);
// nil — активной компании нет (нормальное состояние: мессенджер/профиль).
func (s *Service) session(ctx context.Context, u *domain.User, activeCompanyID *int64, withRefresh bool) (*dto.Session, error) {
	claims := token.Claims{
		UserID:       u.ID,
		ForceChange:  u.IsDefaultPass,
		IsSuperAdmin: u.IsSuperAdmin,
	}
	if activeCompanyID != nil {
		m, err := s.repo.GetMembership(ctx, u.ID, *activeCompanyID)
		if err != nil {
			return nil, err
		}
		if m == nil {
			return nil, errNotAMember
		}
		if m.Company == nil || !m.Company.IsActive {
			var name *string
			if m.Company != nil {
				name = &m.Company.Name
			}
			return nil, companyDisabledErr(name)
		}
		claims.CompanyID = &m.CompanyID
		claims.RoleLevel = m.Role.Level
		claims.CompanyName = &m.Company.Name
		claims.CompanySettings = m.Company.Settings
	}

	access, err := s.tokens.AccessToken(claims)
	if err != nil {
		return nil, err
	}
	memberships, err := s.repo.ListMemberships(ctx, u.ID)
	if err != nil {
		return nil, err
	}
	sess := &dto.Session{
		AccessToken:     access,
		UserID:          u.ID,
		ForceChange:     u.IsDefaultPass,
		CompanyID:       claims.CompanyID,
		CompanyName:     claims.CompanyName,
		CompanySettings: claims.CompanySettings,
		RoleLevel:       claims.RoleLevel,
		IsSuperAdmin:    u.IsSuperAdmin,
		Companies:       dto.NewMemberships(memberships),
	}
	if withRefresh {
		if sess.RefreshToken, err = s.tokens.RefreshToken(u.ID, claims.CompanyID); err != nil {
			return nil, err
		}
	}
	return sess, nil
}

// startSession — выдать сессию после успешной аутентификации (login/register/
// change-default). Супер-админ и пользователь без компаний входят без активной
// компании; одна компания — автоактивна; несколько — этап выбора (login-gate).
func (s *Service) startSession(ctx context.Context, u *domain.User) (*dto.Session, error) {
	if u.IsSuperAdmin {
		return s.session(ctx, u, nil, true)
	}
	memberships, err := s.repo.ListMemberships(ctx, u.ID)
	if err != nil {
		return nil, err
	}
	switch len(memberships) {
	case 0:
		return s.session(ctx, u, nil, true)
	case 1:
		return s.session(ctx, u, &memberships[0].CompanyID, true)
	default:
		selectTok, err := s.tokens.SelectToken(u.ID)
		if err != nil {
			return nil, err
		}
		return &dto.Session{
			UserID:                u.ID,
			NeedsCompanySelection: true,
			SelectToken:           selectTok,
			Companies:             dto.NewMemberships(memberships),
		}, nil
	}
}
