// Package http — HTTP-транспорт (Fiber): REST /api/messenger/*.
//
// Пути и формы JSON байт-в-байт совместимы с прежним Flask-блюпринтом
// api/messenger.py — фронт не меняется, nginx маршрутизирует префикс
// /api/messenger на этот сервис (кроме exact /api/messenger/presence,
// который остаётся во Flask: presence живёт в памяти процесса Socket.IO).
package http

import (
	"context"
	"log/slog"

	"github.com/gofiber/fiber/v2"

	"github.com/DmitriyODS/gw2/back-go/messenger/internal/domain"
	"github.com/DmitriyODS/gw2/back-go/messenger/internal/endpoint"
	"github.com/DmitriyODS/gw2/back-go/messenger/internal/service"
	"github.com/DmitriyODS/gw2/back-go/pkg/httpserver"
	"github.com/DmitriyODS/gw2/back-go/pkg/pasetoauth"
)

type Server struct {
	app *fiber.App
}

// authSource — сверка пользователя для pkg-мидлвари. Активная компания и роль
// в ней — ИЗ ТОКЕНА (active); из БД — глобальная активность аккаунта, признак
// супер-админа, профиль и активность выбранной компании. Многокомпанийный юзер
// скоупится по выбранной компании.
func authSource(users domain.UserReader) pasetoauth.AuthSource {
	return func(ctx context.Context, userID int64, active pasetoauth.Claims) (*pasetoauth.AuthInfo, error) {
		u, err := users.GetUser(ctx, userID)
		if err != nil || u == nil {
			return nil, err
		}
		u.CompanyID = active.CompanyID
		u.RoleLevel = active.RoleLevel
		companyActive, err := users.CompanyActive(ctx, active.CompanyID)
		if err != nil {
			return nil, err
		}
		u.CompanyActive = companyActive
		return &pasetoauth.AuthInfo{
			RoleLevel:     active.RoleLevel,
			IsActive:      u.IsActive,
			IsSuperAdmin:  u.IsSuperAdmin,
			CompanyActive: companyActive,
			User:          u,
		}, nil
	}
}

func NewServer(eps endpoint.Endpoints, svc service.MessengerService, users domain.UserReader,
	verifier *pasetoauth.Verifier, log *slog.Logger) *Server {

	// Вложения ≤25МБ проверяются в сервисе; лимит тела — с запасом
	// (как MAX_CONTENT_LENGTH=50МБ во Flask).
	app := httpserver.New(httpserver.Config{
		AppName: "gw2-msgsvc", Log: log, BodyLimit: 50 * 1024 * 1024,
	})
	auth := pasetoauth.NewMiddleware(verifier, authSource(users))
	h := &handlers{eps: eps, svc: svc, log: log}

	api := app.Group("/api/messenger", auth.RequireAuth)
	api.Get("/conversations", h.listConversations)
	api.Post("/conversations", h.openConversation)
	api.Get("/conversations/:id<int>/messages", h.listMessages)
	api.Post("/conversations/:id<int>/messages", h.postMessage)
	api.Post("/forward", h.forward)
	api.Post("/conversations/:id<int>/read", h.markRead)
	api.Post("/uploads", h.upload)
	api.Delete("/messages/:id<int>", h.deleteMessage)
	api.Patch("/messages/:id<int>", h.editMessage)
	api.Delete("/conversations/:id<int>", h.deleteConversation)
	api.Post("/conversations/:id<int>/pin", h.toggleConversationPin)
	api.Post("/messages/:id<int>/pin", h.toggleMessagePin)
	api.Post("/messages/:id<int>/reactions", h.toggleMessageReaction)
	api.Get("/messages/:id<int>/read-by", h.messageReadBy)
	api.Get("/conversations/:id<int>/pinned", h.listPinned)
	api.Get("/dev-chat", h.openDevChat)
	api.Get("/support-inbox", h.supportInbox)
	api.Get("/unread", h.unread)

	// Оформление чатов (личное, синк между устройствами).
	api.Get("/chat-bg", h.getChatBackgrounds)
	api.Put("/chat-bg", h.putChatBackground)
	api.Delete("/chat-bg", h.deleteChatBackground)

	// Папки чатов (личная навигация, синк между устройствами). Специфичный
	// /folders/reorder раньше :id<int>, чтобы не перехватывался.
	api.Get("/folders", h.listFolders)
	api.Post("/folders", h.createFolder)
	api.Post("/folders/reorder", h.reorderFolders)
	api.Patch("/folders/:id<int>", h.updateFolder)
	api.Delete("/folders/:id<int>", h.deleteFolder)
	api.Post("/folders/:id<int>/items", h.addFolderItem)
	api.Delete("/folders/:id<int>/items/:convId<int>", h.removeFolderItem)

	// ── Группы ───────────────────────────────────────────────────
	// Специфичные пути раньше :id<int>, чтобы не перехватывались (join/invite).
	api.Post("/groups", h.createGroup)
	api.Post("/groups/join/:code", h.joinGroup)
	api.Get("/groups/invite/:code", h.groupInvitePreview)
	api.Get("/groups/:id<int>", h.getGroup)
	api.Patch("/groups/:id<int>", h.patchGroup)
	api.Post("/groups/:id<int>/avatar", h.setGroupAvatar)
	api.Post("/groups/:id<int>/members", h.addGroupMembers)
	api.Delete("/groups/:id<int>/members/:userId<int>", h.removeGroupMember)
	api.Patch("/groups/:id<int>/members/:userId<int>", h.patchGroupMember)
	api.Post("/groups/:id<int>/members/:userId<int>/owner", h.transferOwnership)
	api.Post("/groups/:id<int>/leave", h.leaveGroup)
	api.Post("/groups/:id<int>/mute", h.muteGroup)
	api.Post("/groups/:id<int>/invite-link", h.groupInviteLink)
	api.Delete("/groups/:id<int>/invite-link", h.revokeGroupInviteLink)

	return &Server{app: app}
}

func (s *Server) Listen(addr string) error { return s.app.Listen(addr) }
func (s *Server) Shutdown() error          { return s.app.Shutdown() }
