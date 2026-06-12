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
	Login         endpoint.Endpoint
	Refresh       endpoint.Endpoint
	ChangeDefault endpoint.Endpoint

	ListUsers     endpoint.Endpoint
	CreateUser    endpoint.Endpoint
	Directory     endpoint.Endpoint
	DirectoryUser endpoint.Endpoint
	Me            endpoint.Endpoint
	UpdateMe      endpoint.Endpoint
	UploadAvatar  endpoint.Endpoint
	DeleteAvatar  endpoint.Endpoint
	GetUser       endpoint.Endpoint
	UpdateUser    endpoint.Endpoint
	HideUser      endpoint.Endpoint
	AssignRole    endpoint.Endpoint
	ResetPassword endpoint.Endpoint
}

// Запросы, которым нужен действующий пользователь (actor) или составной ввод.

type ActorRequest struct {
	Actor  *domain.User
	UserID int64
}

type CreateUserEpRequest struct {
	Actor *domain.User
	Body  dto.CreateUserRequest
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

func New(svc service.AuthService) Endpoints {
	return Endpoints{
		Login: func(ctx context.Context, request any) (any, error) {
			return svc.Login(ctx, request.(dto.LoginRequest))
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
	}
}
