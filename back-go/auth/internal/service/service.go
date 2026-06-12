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

	ListRoles(ctx context.Context) ([]dto.Role, error)

	ListCompanies(ctx context.Context) (*dto.CompanyList, error)
	GetCompany(ctx context.Context, companyID int64) (*dto.Company, error)
	CreateCompany(ctx context.Context, req dto.CompanyCreate) (*dto.Company, error)
	UpdateCompany(ctx context.Context, companyID int64, req dto.CompanyUpdate) (*dto.Company, error)
	ToggleCompanyActive(ctx context.Context, companyID int64, isActive bool) (*dto.Company, error)
	DeleteCompany(ctx context.Context, companyID int64) error
	GetWeekendSettings(ctx context.Context, actor *domain.User, companyID int64) (*dto.WeekendSettings, error)
	UpdateWeekendSettings(ctx context.Context, actor *domain.User, companyID int64, days []int) (*dto.WeekendSettings, error)

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

// session — выпуск пары токенов + клеймы в тело ответа (общая точка для
// login/refresh/change-default, как _build_claims во Flask).
func (s *Service) session(u *domain.User, withRefresh bool) (*dto.Session, error) {
	claims := token.Claims{
		UserID:      u.ID,
		ForceChange: u.IsDefaultPass,
		CompanyID:   u.CompanyID,
		RoleLevel:   u.Role.Level,
		IsRootAdmin: u.IsRootAdmin,
	}
	if u.Company != nil {
		claims.CompanyName = &u.Company.Name
		claims.CompanySettings = u.Company.Settings
	}

	access, err := s.tokens.AccessToken(claims)
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
	}
	if withRefresh {
		if sess.RefreshToken, err = s.tokens.RefreshToken(u.ID); err != nil {
			return nil, err
		}
	}
	return sess, nil
}
