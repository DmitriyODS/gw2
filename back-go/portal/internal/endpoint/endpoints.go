// Package endpoint — go-kit обёртки use-case'ов portalsvc: единая сигнатура
// (ctx, request) → (response, error). Та же схема, что в остальных сервисах.
package endpoint

import (
	"context"

	"github.com/go-kit/kit/endpoint"

	"github.com/DmitriyODS/gw2/back-go/portal/internal/service"
)

type Endpoints struct {
	ListTopics  endpoint.Endpoint
	CreateTopic endpoint.Endpoint
	UpdateTopic endpoint.Endpoint
	DeleteTopic endpoint.Endpoint

	ListPosts  endpoint.Endpoint
	GetPost    endpoint.Endpoint
	CreatePost endpoint.Endpoint
	UpdatePost endpoint.Endpoint
	DeletePost endpoint.Endpoint
	Pin        endpoint.Endpoint
	Unpin      endpoint.Endpoint

	Upload           endpoint.Endpoint
	RemoveAttachment endpoint.Endpoint

	ListComments  endpoint.Endpoint
	CreateComment endpoint.Endpoint
	DeleteComment endpoint.Endpoint

	AddReaction    endpoint.Endpoint
	RemoveReaction endpoint.Endpoint

	ForwardPost endpoint.Endpoint

	UnreadCount endpoint.Endpoint
	MarkSeen    endpoint.Endpoint
}

// ── Request-типы ──

type CompanyReq struct{ CompanyID int64 }

type TopicReq struct {
	CompanyID int64
	ID        int64
}

type WriteTopicReq struct {
	CompanyID int64
	ID        int64
	UserID    int64
	Name      string
	Color     *string
	Icon      *string
}

type ListPostsReq struct {
	CompanyID int64
	ViewerID  int64
	Params    service.PostListParams
}

type PostReq struct {
	CompanyID int64
	ID        int64
	ViewerID  int64
}

type WritePostReq struct {
	CompanyID int64
	ID        int64
	UserID    int64
	RoleLevel int
	TopicID   *int64
	Title     *string
	Body      string
}

type PinReq struct {
	CompanyID int64
	ID        int64
	UserID    int64
	RoleLevel int
	// Days — автоистечение пина в днях (0 — бессрочно, кламп 1..30 в сервисе).
	Days int
}

type UploadReq struct {
	CompanyID int64
	PostID    int64
	UserID    int64
	RoleLevel int
	FileName  string
	Mime      string
	Data      []byte
}

type AttachmentReq struct {
	CompanyID int64
	ID        int64
	UserID    int64
	RoleLevel int
}

type ListCommentsReq struct {
	CompanyID int64
	PostID    int64
}

type CreateCommentReq struct {
	CompanyID int64
	PostID    int64
	AuthorID  int64
	Text      string
}

type DeleteCommentReq struct {
	CompanyID int64
	CommentID int64
	UserID    int64
	RoleLevel int
}

type ReactionReq struct {
	CompanyID int64
	PostID    int64
	UserID    int64
	Emoji     string
}

type SeenReq struct {
	CompanyID int64
	UserID    int64
}

type ForwardReq struct {
	CompanyID       int64
	PostID          int64
	SenderID        int64
	ConversationIDs []int64
	UserIDs         []int64
}

func New(s *service.Service) Endpoints {
	return Endpoints{
		ListTopics: func(ctx context.Context, request any) (any, error) {
			r := request.(CompanyReq)
			return s.ListTopics(ctx, r.CompanyID)
		},
		CreateTopic: func(ctx context.Context, request any) (any, error) {
			r := request.(WriteTopicReq)
			return s.CreateTopic(ctx, r.CompanyID, r.UserID, r.Name, r.Color, r.Icon)
		},
		UpdateTopic: func(ctx context.Context, request any) (any, error) {
			r := request.(WriteTopicReq)
			return s.UpdateTopic(ctx, r.CompanyID, r.ID, r.Name, r.Color, r.Icon)
		},
		DeleteTopic: func(ctx context.Context, request any) (any, error) {
			r := request.(TopicReq)
			return nil, s.DeleteTopic(ctx, r.CompanyID, r.ID)
		},
		ListPosts: func(ctx context.Context, request any) (any, error) {
			r := request.(ListPostsReq)
			return s.ListPosts(ctx, r.CompanyID, r.ViewerID, r.Params)
		},
		GetPost: func(ctx context.Context, request any) (any, error) {
			r := request.(PostReq)
			return s.GetPost(ctx, r.CompanyID, r.ID, r.ViewerID)
		},
		CreatePost: func(ctx context.Context, request any) (any, error) {
			r := request.(WritePostReq)
			return s.CreatePost(ctx, r.CompanyID, r.UserID, r.TopicID, r.Title, r.Body)
		},
		UpdatePost: func(ctx context.Context, request any) (any, error) {
			r := request.(WritePostReq)
			return s.UpdatePost(ctx, r.CompanyID, r.ID, r.UserID, r.RoleLevel, r.TopicID, r.Title, r.Body)
		},
		DeletePost: func(ctx context.Context, request any) (any, error) {
			r := request.(PinReq)
			return nil, s.DeletePost(ctx, r.CompanyID, r.ID, r.UserID, r.RoleLevel)
		},
		Pin: func(ctx context.Context, request any) (any, error) {
			r := request.(PinReq)
			return s.Pin(ctx, r.CompanyID, r.ID, r.UserID, r.RoleLevel, r.Days)
		},
		Unpin: func(ctx context.Context, request any) (any, error) {
			r := request.(PinReq)
			return s.Unpin(ctx, r.CompanyID, r.ID, r.UserID, r.RoleLevel)
		},
		Upload: func(ctx context.Context, request any) (any, error) {
			r := request.(UploadReq)
			return s.AddAttachment(ctx, r.CompanyID, r.PostID, r.UserID, r.RoleLevel, r.FileName, r.Mime, r.Data)
		},
		RemoveAttachment: func(ctx context.Context, request any) (any, error) {
			r := request.(AttachmentReq)
			return nil, s.RemoveAttachment(ctx, r.CompanyID, r.ID, r.UserID, r.RoleLevel)
		},
		ListComments: func(ctx context.Context, request any) (any, error) {
			r := request.(ListCommentsReq)
			return s.ListComments(ctx, r.CompanyID, r.PostID)
		},
		CreateComment: func(ctx context.Context, request any) (any, error) {
			r := request.(CreateCommentReq)
			return s.CreateComment(ctx, r.CompanyID, r.PostID, r.AuthorID, r.Text)
		},
		DeleteComment: func(ctx context.Context, request any) (any, error) {
			r := request.(DeleteCommentReq)
			return nil, s.DeleteComment(ctx, r.CompanyID, r.CommentID, r.UserID, r.RoleLevel)
		},
		AddReaction: func(ctx context.Context, request any) (any, error) {
			r := request.(ReactionReq)
			return nil, s.AddReaction(ctx, r.CompanyID, r.PostID, r.UserID, r.Emoji)
		},
		RemoveReaction: func(ctx context.Context, request any) (any, error) {
			r := request.(ReactionReq)
			return nil, s.RemoveReaction(ctx, r.CompanyID, r.PostID, r.UserID, r.Emoji)
		},
		ForwardPost: func(ctx context.Context, request any) (any, error) {
			r := request.(ForwardReq)
			return s.ForwardPost(ctx, r.CompanyID, r.PostID, r.SenderID, r.ConversationIDs, r.UserIDs)
		},
		UnreadCount: func(ctx context.Context, request any) (any, error) {
			r := request.(SeenReq)
			return s.UnreadCount(ctx, r.UserID, r.CompanyID)
		},
		MarkSeen: func(ctx context.Context, request any) (any, error) {
			r := request.(SeenReq)
			return nil, s.MarkSeen(ctx, r.UserID, r.CompanyID)
		},
	}
}
