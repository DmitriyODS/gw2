// Package endpoint — go-kit обёртки use-case'ов: единая сигнатура
// (ctx, request) → (response, error) независимо от транспорта.
// Та же схема, что в callsvc.
package endpoint

import (
	"context"

	"github.com/go-kit/kit/endpoint"

	"github.com/DmitriyODS/gw2/back-go/auth/internal/domain"
	"github.com/DmitriyODS/gw2/back-go/auth/internal/dto"
	"github.com/DmitriyODS/gw2/back-go/auth/internal/service"
)

type Endpoints struct {
	Login                endpoint.Endpoint
	Register             endpoint.Endpoint
	SuggestLogin         endpoint.Endpoint
	VerifyEmail          endpoint.Endpoint
	ResendVerification   endpoint.Endpoint
	RequestPasswordReset endpoint.Endpoint
	ResetPasswordByToken endpoint.Endpoint
	SelectCompany        endpoint.Endpoint
	SwitchCompany        endpoint.Endpoint
	Refresh              endpoint.Endpoint
	ChangeDefault        endpoint.Endpoint

	ListUsers              endpoint.Endpoint
	CreateUser             endpoint.Endpoint
	CreatePlatformUser     endpoint.Endpoint
	UpdatePlatformUser     endpoint.Endpoint
	ResetPlatformUser      endpoint.Endpoint
	DeactivatePlatformUser endpoint.Endpoint
	Directory              endpoint.Endpoint
	DirectoryUser          endpoint.Endpoint
	Me                     endpoint.Endpoint
	UpdateMe               endpoint.Endpoint
	UploadAvatar           endpoint.Endpoint
	DeleteAvatar           endpoint.Endpoint
	GetUser                endpoint.Endpoint
	UpdateUser             endpoint.Endpoint
	HideUser               endpoint.Endpoint
	AssignRole             endpoint.Endpoint
	ResetPassword          endpoint.Endpoint
	ListCompanyMembers     endpoint.Endpoint
	AddCompanyMember       endpoint.Endpoint
	SetMemberRole          endpoint.Endpoint
	RemoveMember           endpoint.Endpoint
	SearchCandidates       endpoint.Endpoint
	CompanyInvite          endpoint.Endpoint
	RegenerateInvite       endpoint.Endpoint
	JoinByCode             endpoint.Endpoint

	ListRoles endpoint.Endpoint

	ListCompanies         endpoint.Endpoint
	ListMyCompanies       endpoint.Endpoint
	CreateCompanyUser     endpoint.Endpoint
	UpdateCompanyMember   endpoint.Endpoint
	ResetCompanyMember    endpoint.Endpoint
	CreateCompanyInvite   endpoint.Endpoint
	GetInvitePreview      endpoint.Endpoint
	AcceptCompanyInvite   endpoint.Endpoint
	GetCompany            endpoint.Endpoint
	CreateCompany         endpoint.Endpoint
	UpdateCompany         endpoint.Endpoint
	ToggleCompanyActive   endpoint.Endpoint
	DeleteCompany         endpoint.Endpoint
	GetWeekendSettings    endpoint.Endpoint
	UpdateWeekendSettings endpoint.Endpoint
	GetGrooveSettings     endpoint.Endpoint
	UpdateGrooveSettings  endpoint.Endpoint

	ExportBackup endpoint.Endpoint
	ImportBackup endpoint.Endpoint
}

// Запросы, которым нужен действующий пользователь (actor) или составной ввод.

type ActorRequest struct {
	Actor  *domain.User
	UserID int64
}

// ImportBackupReq — ZIP-архив и выбранные к восстановлению разделы.
type ImportBackupReq struct {
	Zip      []byte
	Sections []string
}

type CreateUserEpRequest struct {
	Actor *domain.User
	Body  dto.CreateUserRequest
}

// Платформенные операции супер-админа над идентичностью (без компании).
type PlatformCreateEpRequest struct {
	Body dto.CreateUserRequest
}

type PlatformUpdateEpRequest struct {
	UserID int64
	Body   dto.UpdateUserRequest
}

type UpdateUserEpRequest struct {
	Actor  *domain.User
	UserID int64
	Body   dto.UpdateUserRequest
}

type UpdateMeEpRequest struct {
	UserID int64
	Body   dto.UpdateMeRequest
}

type AvatarEpRequest struct {
	UserID int64
	File   []byte
}

type AssignRoleEpRequest struct {
	Actor  *domain.User
	UserID int64
	RoleID int64
}

type SelectCompanyEpRequest struct {
	SelectToken string
	CompanyID   int64
}

type SwitchCompanyEpRequest struct {
	UserID    int64
	CompanyID int64
}

type MemberEpRequest struct {
	Actor     *domain.User
	CompanyID int64
	UserID    int64
	RoleID    int64
}

type CandidatesEpRequest struct {
	Actor     *domain.User
	CompanyID int64
	Query     string
}

type CompanyActorEpRequest struct {
	Actor     *domain.User
	CompanyID int64
}

type JoinEpRequest struct {
	UserID int64
	Code   string
}

type CreateCompanyEpRequest struct {
	Actor *domain.User
	Body  dto.CompanyCreate
}

type CompanyUserCreateEpRequest struct {
	Actor     *domain.User
	CompanyID int64
	Body      dto.CreateUserRequest
}

type CompanyUserUpdateEpRequest struct {
	Actor     *domain.User
	CompanyID int64
	UserID    int64
	Body      dto.UpdateUserRequest
}

type CompanyMemberResetEpRequest struct {
	Actor     *domain.User
	CompanyID int64
	UserID    int64
}

type CreateInviteEpRequest struct {
	Actor     *domain.User
	CompanyID int64
	Email     string
	RoleID    int64
}

type AcceptInviteEpRequest struct {
	UserID int64
	Token  string
}

type UpdateCompanyEpRequest struct {
	Actor     *domain.User
	CompanyID int64
	Body      dto.CompanyUpdate
}

type ToggleCompanyEpRequest struct {
	CompanyID int64
	IsActive  bool
}

type CompanyScopeEpRequest struct {
	Actor     *domain.User
	CompanyID int64
}

type WeekendEpRequest struct {
	Actor     *domain.User
	CompanyID int64
	Days      []int
}

type GrooveEpRequest struct {
	Actor     *domain.User
	CompanyID int64
	Enabled   bool
}

func New(svc service.AuthService) Endpoints {
	return Endpoints{
		Login: func(ctx context.Context, request any) (any, error) {
			return svc.Login(ctx, request.(dto.LoginRequest))
		},
		Register: func(ctx context.Context, request any) (any, error) {
			return svc.Register(ctx, request.(dto.RegisterRequest))
		},
		SuggestLogin: func(ctx context.Context, request any) (any, error) {
			return svc.SuggestLogin(ctx, request.(string))
		},
		VerifyEmail: func(ctx context.Context, request any) (any, error) {
			return svc.VerifyEmail(ctx, request.(dto.VerifyEmailRequest))
		},
		ResendVerification: func(ctx context.Context, request any) (any, error) {
			return nil, svc.ResendVerification(ctx, request.(string))
		},
		RequestPasswordReset: func(ctx context.Context, request any) (any, error) {
			return nil, svc.RequestPasswordReset(ctx, request.(string))
		},
		ResetPasswordByToken: func(ctx context.Context, request any) (any, error) {
			return svc.ResetPasswordByToken(ctx, request.(dto.ResetPasswordRequest))
		},
		SelectCompany: func(ctx context.Context, request any) (any, error) {
			req := request.(SelectCompanyEpRequest)
			return svc.SelectCompany(ctx, req.SelectToken, req.CompanyID)
		},
		SwitchCompany: func(ctx context.Context, request any) (any, error) {
			req := request.(SwitchCompanyEpRequest)
			return svc.SwitchCompany(ctx, req.UserID, req.CompanyID)
		},
		Refresh: func(ctx context.Context, request any) (any, error) {
			return svc.Refresh(ctx, request.(string))
		},
		ChangeDefault: func(ctx context.Context, request any) (any, error) {
			return svc.ChangeDefault(ctx, request.(dto.ChangeDefaultRequest))
		},
		ListUsers: func(ctx context.Context, _ any) (any, error) {
			return svc.ListUsers(ctx)
		},
		CreateUser: func(ctx context.Context, request any) (any, error) {
			req := request.(CreateUserEpRequest)
			return svc.CreateUser(ctx, req.Actor, req.Body)
		},
		CreatePlatformUser: func(ctx context.Context, request any) (any, error) {
			return svc.CreatePlatformUser(ctx, request.(PlatformCreateEpRequest).Body)
		},
		UpdatePlatformUser: func(ctx context.Context, request any) (any, error) {
			req := request.(PlatformUpdateEpRequest)
			return svc.UpdatePlatformUser(ctx, req.UserID, req.Body)
		},
		ResetPlatformUser: func(ctx context.Context, request any) (any, error) {
			req := request.(ActorRequest)
			return nil, svc.ResetPlatformUserPassword(ctx, req.Actor, req.UserID)
		},
		DeactivatePlatformUser: func(ctx context.Context, request any) (any, error) {
			req := request.(ActorRequest)
			return nil, svc.DeactivatePlatformUser(ctx, req.Actor, req.UserID)
		},
		Directory: func(ctx context.Context, request any) (any, error) {
			return svc.Directory(ctx, request.(dto.DirectoryRequest))
		},
		DirectoryUser: func(ctx context.Context, request any) (any, error) {
			return svc.DirectoryUser(ctx, request.(int64))
		},
		Me: func(ctx context.Context, request any) (any, error) {
			return svc.Me(ctx, request.(int64))
		},
		UpdateMe: func(ctx context.Context, request any) (any, error) {
			req := request.(UpdateMeEpRequest)
			return svc.UpdateMe(ctx, req.UserID, req.Body)
		},
		UploadAvatar: func(ctx context.Context, request any) (any, error) {
			req := request.(AvatarEpRequest)
			return svc.UploadAvatar(ctx, req.UserID, req.File)
		},
		DeleteAvatar: func(ctx context.Context, request any) (any, error) {
			return svc.DeleteAvatar(ctx, request.(int64))
		},
		GetUser: func(ctx context.Context, request any) (any, error) {
			return svc.GetUser(ctx, request.(int64))
		},
		UpdateUser: func(ctx context.Context, request any) (any, error) {
			req := request.(UpdateUserEpRequest)
			return svc.UpdateUser(ctx, req.Actor, req.UserID, req.Body)
		},
		HideUser: func(ctx context.Context, request any) (any, error) {
			req := request.(ActorRequest)
			return nil, svc.HideUser(ctx, req.Actor, req.UserID)
		},
		AssignRole: func(ctx context.Context, request any) (any, error) {
			req := request.(AssignRoleEpRequest)
			return svc.AssignRole(ctx, req.Actor, req.UserID, req.RoleID)
		},
		ResetPassword: func(ctx context.Context, request any) (any, error) {
			req := request.(ActorRequest)
			return nil, svc.ResetPassword(ctx, req.Actor, req.UserID)
		},
		ListCompanyMembers: func(ctx context.Context, request any) (any, error) {
			req := request.(CompanyActorEpRequest)
			return svc.ListCompanyMembers(ctx, req.Actor, req.CompanyID)
		},
		AddCompanyMember: func(ctx context.Context, request any) (any, error) {
			req := request.(MemberEpRequest)
			return nil, svc.AddCompanyMember(ctx, req.Actor, req.CompanyID, req.UserID, req.RoleID)
		},
		SetMemberRole: func(ctx context.Context, request any) (any, error) {
			req := request.(MemberEpRequest)
			return nil, svc.SetMemberRole(ctx, req.Actor, req.CompanyID, req.UserID, req.RoleID)
		},
		RemoveMember: func(ctx context.Context, request any) (any, error) {
			req := request.(MemberEpRequest)
			return nil, svc.RemoveCompanyMember(ctx, req.Actor, req.CompanyID, req.UserID)
		},
		SearchCandidates: func(ctx context.Context, request any) (any, error) {
			req := request.(CandidatesEpRequest)
			return svc.SearchCandidates(ctx, req.Actor, req.CompanyID, req.Query)
		},
		CompanyInvite: func(ctx context.Context, request any) (any, error) {
			req := request.(CompanyActorEpRequest)
			return svc.CompanyInvite(ctx, req.Actor, req.CompanyID)
		},
		RegenerateInvite: func(ctx context.Context, request any) (any, error) {
			req := request.(CompanyActorEpRequest)
			return svc.RegenerateInvite(ctx, req.Actor, req.CompanyID)
		},
		JoinByCode: func(ctx context.Context, request any) (any, error) {
			req := request.(JoinEpRequest)
			return svc.JoinByCode(ctx, req.UserID, req.Code)
		},

		ListRoles: func(ctx context.Context, _ any) (any, error) {
			return svc.ListRoles(ctx)
		},

		ListCompanies: func(ctx context.Context, _ any) (any, error) {
			return svc.ListCompanies(ctx)
		},
		ListMyCompanies: func(ctx context.Context, request any) (any, error) {
			return svc.ListMyCompanies(ctx, request.(*domain.User))
		},
		CreateCompanyUser: func(ctx context.Context, request any) (any, error) {
			req := request.(CompanyUserCreateEpRequest)
			return svc.CreateCompanyUser(ctx, req.Actor, req.CompanyID, req.Body)
		},
		UpdateCompanyMember: func(ctx context.Context, request any) (any, error) {
			req := request.(CompanyUserUpdateEpRequest)
			return svc.UpdateCompanyMember(ctx, req.Actor, req.CompanyID, req.UserID, req.Body)
		},
		ResetCompanyMember: func(ctx context.Context, request any) (any, error) {
			req := request.(CompanyMemberResetEpRequest)
			return nil, svc.ResetCompanyMemberPassword(ctx, req.Actor, req.CompanyID, req.UserID)
		},
		CreateCompanyInvite: func(ctx context.Context, request any) (any, error) {
			req := request.(CreateInviteEpRequest)
			return nil, svc.CreateCompanyInvite(ctx, req.Actor, req.CompanyID, req.Email, req.RoleID)
		},
		GetInvitePreview: func(ctx context.Context, request any) (any, error) {
			return svc.GetCompanyInvitePreview(ctx, request.(string))
		},
		AcceptCompanyInvite: func(ctx context.Context, request any) (any, error) {
			req := request.(AcceptInviteEpRequest)
			return svc.AcceptCompanyInvite(ctx, req.UserID, req.Token)
		},
		GetCompany: func(ctx context.Context, request any) (any, error) {
			req := request.(CompanyActorEpRequest)
			return svc.GetCompany(ctx, req.Actor, req.CompanyID)
		},
		CreateCompany: func(ctx context.Context, request any) (any, error) {
			req := request.(CreateCompanyEpRequest)
			return svc.CreateCompany(ctx, req.Actor, req.Body)
		},
		UpdateCompany: func(ctx context.Context, request any) (any, error) {
			req := request.(UpdateCompanyEpRequest)
			return svc.UpdateCompany(ctx, req.Actor, req.CompanyID, req.Body)
		},
		ToggleCompanyActive: func(ctx context.Context, request any) (any, error) {
			req := request.(ToggleCompanyEpRequest)
			return svc.ToggleCompanyActive(ctx, req.CompanyID, req.IsActive)
		},
		DeleteCompany: func(ctx context.Context, request any) (any, error) {
			req := request.(CompanyActorEpRequest)
			return nil, svc.DeleteCompany(ctx, req.Actor, req.CompanyID)
		},
		GetWeekendSettings: func(ctx context.Context, request any) (any, error) {
			req := request.(CompanyScopeEpRequest)
			return svc.GetWeekendSettings(ctx, req.Actor, req.CompanyID)
		},
		UpdateWeekendSettings: func(ctx context.Context, request any) (any, error) {
			req := request.(WeekendEpRequest)
			return svc.UpdateWeekendSettings(ctx, req.Actor, req.CompanyID, req.Days)
		},
		GetGrooveSettings: func(ctx context.Context, request any) (any, error) {
			req := request.(CompanyScopeEpRequest)
			return svc.GetGrooveSettings(ctx, req.Actor, req.CompanyID)
		},
		UpdateGrooveSettings: func(ctx context.Context, request any) (any, error) {
			req := request.(GrooveEpRequest)
			return svc.UpdateGrooveSettings(ctx, req.Actor, req.CompanyID, req.Enabled)
		},

		ExportBackup: func(ctx context.Context, request any) (any, error) {
			sections, _ := request.([]string)
			return svc.ExportBackup(ctx, sections)
		},
		ImportBackup: func(ctx context.Context, request any) (any, error) {
			req := request.(ImportBackupReq)
			return nil, svc.ImportBackup(ctx, req.Zip, req.Sections)
		},
	}
}
