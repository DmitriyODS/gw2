package domain

import (
	"context"
	"encoding/json"
	"time"
)

// Ctx — алиас, чтобы сигнатуры портов не разбухали.
type Ctx = context.Context

// BackgroundRepository — персональное оформление ленты портала (одна строка
// на пользователя). recipe — непрозрачный JSON, форму которого владеет фронт.
type BackgroundRepository interface {
	GetPortalBackground(ctx Ctx, userID int64) (json.RawMessage, error)
	UpsertPortalBackground(ctx Ctx, userID int64, recipe []byte) error
	DeletePortalBackground(ctx Ctx, userID int64) error
}

// TopicRepository — персистентность тематических разделов.
type TopicRepository interface {
	ListTopics(ctx Ctx, companyID int64) ([]*Topic, error)
	GetTopic(ctx Ctx, id int64) (*Topic, error)
	CreateTopic(ctx Ctx, t *Topic) error
	UpdateTopic(ctx Ctx, id int64, name string, color, icon *string) error
	DeleteTopic(ctx Ctx, id int64) error
}

// PostRepository — персистентность постов, вложений, комментариев и реакций.
type PostRepository interface {
	ListPosts(ctx Ctx, f PostListFilter, viewerID int64) ([]*Post, error)
	GetPost(ctx Ctx, id int64) (*Post, error)
	// GetPostForViewer — пост с батч-подгруженными вложениями/счётчиками для
	// одной карточки (те же данные, что ListPosts, для единичной записи).
	GetPostForViewer(ctx Ctx, id, viewerID int64) (*Post, error)
	// CreatePost/UpdatePost также сохраняют хештеги поста (p.Tags) в одной
	// транзакции с самой записью — тело поста и его теги не расходятся.
	CreatePost(ctx Ctx, p *Post) error
	UpdatePost(ctx Ctx, p *Post) error
	DeletePost(ctx Ctx, id int64) error
	// PopularTags — топ хештегов компании по числу постов (для панели
	// «Популярные теги» ленты).
	PopularTags(ctx Ctx, companyID int64, limit int) ([]TagCount, error)
	// PinPost — закрепить пост (until — автоистечение, nil = бессрочно),
	// атомарно соблюдая лимит одновременно закреплённых на компанию (проверка
	// лимита и UPDATE — в одной транзакции под локом компании; false — лимит
	// исчерпан). Истёкшие пины в лимите НЕ считаются. Повторное закрепление
	// уже закреплённого поста лишь обновляет pinned_at/pinned_by/pinned_until.
	PinPost(ctx Ctx, id, companyID, pinnedBy int64, until *time.Time, limit int) (bool, error)
	// SetPinned — открепление (pinnedAt=nil, pinned_until сбрасывается);
	// лимит не участвует.
	SetPinned(ctx Ctx, id int64, pinnedAt *time.Time, pinnedBy *int64) error
	AddAttachment(ctx Ctx, a *Attachment) error
	// GetAttachment — вложение по id; nil — не найдено.
	GetAttachment(ctx Ctx, id int64) (*Attachment, error)
	DeleteAttachment(ctx Ctx, id int64) error
	ListAttachments(ctx Ctx, postID int64) ([]Attachment, error)
	// AttachmentPaths — пути файлов поста (для чистки хранилища при удалении).
	AttachmentPaths(ctx Ctx, postID int64) ([]string, error)

	// ListComments — обсуждение поста плоским списком в хронологии; дерево
	// строит клиент по reply_to_id. Лайки (счётчик + «мой») приходят батчем
	// для viewerID — по одному запросу на список, не на комментарий.
	ListComments(ctx Ctx, postID, viewerID int64) ([]*Comment, error)
	GetComment(ctx Ctx, id int64) (*Comment, error)
	CreateComment(ctx Ctx, c *Comment) error
	DeleteComment(ctx Ctx, id int64) error
	// ToggleCommentLike — поставить/снять лайк одной операцией; возвращает
	// состояние после переключения (liked) и актуальный счётчик.
	ToggleCommentLike(ctx Ctx, commentID, userID int64) (liked bool, count int, err error)

	AddReaction(ctx Ctx, r *Reaction) error
	RemoveReaction(ctx Ctx, postID, userID int64, emoji string) error

	// MarkView — зафиксировать просмотр поста зрителем (идемпотентно: повторный
	// просмотр того же поста тем же пользователем счётчик не наращивает).
	MarkView(ctx Ctx, postID, userID int64) error
}

// SeenRepository — отметка «портал просмотрен» и счётчик непрочитанных постов
// (бейдж в навигации, серверный — общий между устройствами пользователя).
type SeenRepository interface {
	// SeenAt — момент последнего просмотра портала; nil — ещё не открывал.
	SeenAt(ctx Ctx, userID, companyID int64) (*time.Time, error)
	MarkSeen(ctx Ctx, userID, companyID int64) error
	// CountPostsAfter — число постов компании с created_at > after (after == nil —
	// все), НЕ авторства excludeAuthorID: свои посты непрочитанными не считаются.
	CountPostsAfter(ctx Ctx, companyID, excludeAuthorID int64, after *time.Time) (int, error)
}

// Repository — вся персистентность portalsvc (одна реализация — postgres.Repo).
type Repository interface {
	TopicRepository
	PostRepository
	SeenRepository
	BackgroundRepository
}

// UserReader — read-only идентичность пользователей (владелец таблицы — authsvc).
type UserReader interface {
	GetUser(ctx Ctx, id int64) (*User, error)
	CompanyActive(ctx Ctx, companyID *int64) (bool, error)
}

// FileStore — хранение вложений постов (общий uploads-том или S3).
type FileStore interface {
	// Save — записать файл, вернуть относительный путь (ключ) хранилища.
	Save(fileName string, data []byte) (string, error)
	// Remove — best-effort удаление файлов по ключам (чистка при удалении постов).
	Remove(paths []string)
}

// EventBus — сокет-события клиентам через Redis gw2:portal:events
// (realtime-шлюз gatewaysvc доставляет их в WS-комнаты вербатим).
type EventBus interface {
	Publish(ctx Ctx, event string, rooms []string, payload any)
}

// PostPreview — снапшот поста для пересылки в мессенджер.
type PostPreview struct {
	Title    string
	Excerpt  string
	CoverURL string
}

// MessengerClient — gRPC-клиент msgsvc: пересылка поста как структурного
// сообщения kind='post' в существующем диалоге.
type MessengerClient interface {
	// EnsureDialog — найти/создать парный диалог (для пересылки по user_ids,
	// когда диалога ещё нет).
	EnsureDialog(ctx Ctx, userAID, userBID int64) (int64, error)
	// CreatePostMessage — плашка поста kind='post' в диалоге (msgsvc сам
	// проверяет участие отправителя и сам публикует message:new в
	// gw2:messenger:events — тем же путём событие видит pushsvc). Возвращает
	// готовый JSON-снапшот сообщения (форма REST msgsvc) и адресатов события.
	CreatePostMessage(ctx Ctx, conversationID, senderID, postID int64, preview PostPreview) (messageJSON string, notifyUserIDs []int64, err error)
}
