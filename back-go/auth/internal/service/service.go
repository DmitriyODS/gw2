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
	SelectCompany(ctx context.Context, selectToken string, companyID int64) (*dto.Session, error)
	SwitchCompany(ctx context.Context, userID, companyID int64) (*dto.Session, error)
	Refresh(ctx context.Context, refreshToken string) (*dto.Session, error)
	ChangeDefault(ctx context.Context, req dto.ChangeDefaultRequest) (*dto.Session, error)

	ListUsers(ctx context.Context) ([]dto.User, error)
	CreateUser(ctx context.Context, actor *domain.User, req dto.CreateUserRequest) (*dto.User, error)
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

	ListRoles(ctx context.Context) ([]dto.Role, error)

	ListCompanies(ctx context.Context) (*dto.CompanyList, error)
	GetCompany(ctx context.Context, companyID int64) (*dto.Company, error)
	CreateCompany(ctx context.Context, req dto.CompanyCreate) (*dto.Company, error)
	UpdateCompany(ctx context.Context, companyID int64, req dto.CompanyUpdate) (*dto.Company, error)
	ToggleCompanyActive(ctx context.Context, companyID int64, isActive bool) (*dto.Company, error)
	DeleteCompany(ctx context.Context, companyID int64) error
	GetWeekendSettings(ctx context.Context, actor *domain.User, companyID int64) (*dto.WeekendSettings, error)
	UpdateWeekendSettings(ctx context.Context, actor *domain.User, companyID int64, days []int) (*dto.WeekendSettings, error)
	GetGrooveSettings(ctx context.Context, actor *domain.User, companyID int64) (*dto.GrooveSettings, error)
	UpdateGrooveSettings(ctx context.Context, actor *domain.User, companyID int64, enabled bool) (*dto.GrooveSettings, error)

	ExportBackup(ctx context.Context) ([]byte, error)
	ImportBackup(ctx context.Context, zipBytes []byte) error
}

type Service struct {
	repo      domain.UserRepository
	companies domain.CompanyRepository
	backup    domain.BackupStore
	throttle  domain.LoginThrottle
	tokens    *token.Issuer
	avatars   domain.AvatarStorage
	log       *slog.Logger
}

func New(repo domain.UserRepository, companies domain.CompanyRepository,
	backup domain.BackupStore, throttle domain.LoginThrottle, tokens *token.Issuer,
	avatars domain.AvatarStorage, log *slog.Logger) *Service {
	return &Service{repo: repo, companies: companies, backup: backup,
		throttle: throttle, tokens: tokens, avatars: avatars, log: log}
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
// точка для login/select/switch/refresh/change-default). activeCompanyID —
// выбранная компания сессии: роль и реквизиты берутся из членства; nil —
// Администратор системы (без компании, роль из users.role_id).
func (s *Service) session(ctx context.Context, u *domain.User, activeCompanyID *int64, withRefresh bool) (*dto.Session, error) {
	claims := token.Claims{
		UserID:      u.ID,
		ForceChange: u.IsDefaultPass,
		RoleLevel:   u.Role.Level,
		IsRootAdmin: u.IsRootAdmin,
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
		IsRootAdmin:     claims.IsRootAdmin,
		Companies:       dto.NewMemberships(memberships),
	}
	if withRefresh {
		if sess.RefreshToken, err = s.tokens.RefreshToken(u.ID, claims.CompanyID); err != nil {
			return nil, err
		}
	}
	return sess, nil
}
